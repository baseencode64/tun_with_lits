package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/goxray/tun/pkg/client"
	"gopkg.in/natefinch/lumberjack.v2"
)

var cmdArgsErr = `ERROR: no config provided
usage: %s <config_url_or_link> [options]
  - config_url - xray connection link, like "vless://example..."
  - or raw list URL: --from-raw <https://example.com/links.txt>
  - or set GOXRAY_CONFIG_URL env var

Options for --from-raw mode:
  --refresh-interval <duration> - periodically refresh server list (e.g., 5m, 10m, 1h)
                                  default: 0 (no refresh)
  --max-servers <n>             - maximum number of servers to check (default: 10)
  --timeout <duration>          - timeout per server health check (default: 5s)
  --ipv6                        - enable IPv6 support (default: false)

Logging options:
  --log-format <format>         - log format: text or json (default: text)
  --log-level <level>           - log level: debug, info, warn, error (default: info)
  --log-file <path>             - log file path (optional, default: stdout only)
  --log-max-size <MB>           - max log file size in MB before rotation (default: 100)
  --log-max-backups <count>     - max number of backup log files (default: 3)
  --log-max-age <days>          - max age of backup logs in days (default: 28)
`

func main() {
	// Get connection link from cmd arguments
	var clientLink string
	var fromRaw bool
	var rawURL string
	var refreshInterval time.Duration = 0 // Default: no refresh
	var maxServers int = 10
	var timeout time.Duration = 5 * time.Second
	var enableIPv6 bool = false
	
	// Logging configuration
	var logFormat string = "text"
	var logLevel string = "info"
	var logFile string = ""
	var logMaxSize int = 100
	var logMaxBackups int = 3
	var logMaxAge int = 28

	args := os.Args[1:]

	// Parse arguments with support for flags
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--from-raw":
			fromRaw = true
			if i+1 >= len(args) {
				fmt.Printf(cmdArgsErr, os.Args[0])
				os.Exit(1)
			}
			i++
			rawURL = args[i]
		case "--refresh-interval":
			if i+1 >= len(args) {
				log.Fatal("--refresh-interval requires a value (e.g., 5m, 10m, 1h)")
			}
			i++
			var err error
			refreshInterval, err = time.ParseDuration(args[i])
			if err != nil {
				log.Fatalf("Invalid refresh interval format: %v", err)
			}
		case "--max-servers":
			if i+1 >= len(args) {
				log.Fatal("--max-servers requires a value")
			}
			i++
			fmt.Sscanf(args[i], "%d", &maxServers)
			if maxServers <= 0 {
				log.Fatal("--max-servers must be positive")
			}
		case "--timeout":
			if i+1 >= len(args) {
				log.Fatal("--timeout requires a value (e.g., 5s, 10s)")
			}
			i++
			var err error
			timeout, err = time.ParseDuration(args[i])
			if err != nil {
				log.Fatalf("Invalid timeout format: %v", err)
			}
		case "--ipv6":
			enableIPv6 = true
		case "--log-format":
			if i+1 >= len(args) {
				log.Fatal("--log-format requires a value (text or json)")
			}
			i++
			logFormat = strings.ToLower(args[i])
			if logFormat != "text" && logFormat != "json" {
				log.Fatal("--log-format must be 'text' or 'json'")
			}
		case "--log-level":
			if i+1 >= len(args) {
				log.Fatal("--log-level requires a value (debug, info, warn, error)")
			}
			i++
			logLevel = strings.ToLower(args[i])
			if logLevel != "debug" && logLevel != "info" && logLevel != "warn" && logLevel != "error" {
				log.Fatal("--log-level must be 'debug', 'info', 'warn', or 'error'")
			}
		case "--log-file":
			if i+1 >= len(args) {
				log.Fatal("--log-file requires a path")
			}
			i++
			logFile = args[i]
		case "--log-max-size":
			if i+1 >= len(args) {
				log.Fatal("--log-max-size requires a value in MB")
			}
			i++
			fmt.Sscanf(args[i], "%d", &logMaxSize)
			if logMaxSize <= 0 {
				log.Fatal("--log-max-size must be positive")
			}
		case "--log-max-backups":
			if i+1 >= len(args) {
				log.Fatal("--log-max-backups requires a value")
			}
			i++
			fmt.Sscanf(args[i], "%d", &logMaxBackups)
			if logMaxBackups < 0 {
				log.Fatal("--log-max-backups cannot be negative")
			}
		case "--log-max-age":
			if i+1 >= len(args) {
				log.Fatal("--log-max-age requires a value in days")
			}
			i++
			fmt.Sscanf(args[i], "%d", &logMaxAge)
			if logMaxAge < 0 {
				log.Fatal("--log-max-age cannot be negative")
			}
		default:
			// Positional argument (direct link)
			if clientLink == "" {
				clientLink = args[i]
			}
		}
		i++
	}

	if clientLink == "" && !fromRaw {
		clientLink = os.Getenv("GOXRAY_CONFIG_URL")
	}

	if clientLink == "" && !fromRaw {
		fmt.Printf(cmdArgsErr, os.Args[0])
		os.Exit(0)
	}

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

	// Create logger with JSON/text format and optional file rotation
	logger, logCleanup := createLogger(logFormat, logLevel, logFile, logMaxSize, logMaxBackups, logMaxAge)
	defer logCleanup()

	vpn, err := client.NewClientWithOpts(client.Config{
		TLSAllowInsecure: false,
		Logger:           logger,
		EnableIPv6:       enableIPv6,
	})
	if err != nil {
		log.Fatal(err)
	}

	if enableIPv6 {
		slog.Info("IPv6 support enabled")
	}

	// Log configuration info
	slog.Info("Logging configuration", 
		"format", logFormat, 
		"level", logLevel,
		"file", logFile,
		"max_size_mb", logMaxSize,
		"max_backups", logMaxBackups,
		"max_age_days", logMaxAge)

	// If using raw URL, fetch and select servers with fallback support
	if fromRaw {
		slog.Info("Fetching server list from raw URL", "url", rawURL, "refresh_interval", refreshInterval)

		loggerAdapter := client.NewSlogAdapter(logger)
		selector := client.NewServerSelector(loggerAdapter, timeout, maxServers)

		// Initial fetch and connect
		connectToServers := func() error {
			links, fetchErr := selector.FetchRawLinks(rawURL)
			if fetchErr != nil {
				return fmt.Errorf("fetch links: %w", fetchErr)
			}

			servers, selectErr := selector.SelectAllByLatency(links)
			if selectErr != nil {
				return fmt.Errorf("select servers: %w", selectErr)
			}

			connector := client.NewVPNConnector(vpn, selector, logger)
			defer connector.Stop()

			// Log server report in structured format (JSON-compatible)
			connector.LogServerReport(servers)

			slog.Info("Attempting VPN connection with fallback support", "servers_count", len(servers))
			if connErr := connector.ConnectWithFallback(servers); connErr != nil {
				return fmt.Errorf("connect: %w", connErr)
			}

			// Create a cancellable context for monitoring goroutines
			monitorCtx, monitorCancel := context.WithCancel(context.Background())
			
			// Ensure cleanup happens on function exit
			defer func() {
				slog.Info("Stopping all monitoring goroutines")
				monitorCancel()
			}()

			// Start periodic server list refresh if enabled
			if refreshInterval > 0 {
				slog.Info("Periodic server list refresh enabled", "interval", refreshInterval)
				go startPeriodicRefresh(monitorCtx, connector, selector, rawURL, refreshInterval, timeout, maxServers)
			}

			// Monitor health status periodically
			go func() {
				ticker := time.NewTicker(30 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						status := connector.GetHealthStatus()
						slog.Info("VPN Health Status", "status", status)
					case <-sigterm:
						monitorCancel() // Signal other goroutines to stop
						return
					case <-monitorCtx.Done():
						return // Context cancelled (e.g., during shutdown)
					}
				}
			}()

			// Wait for termination signal
			<-sigterm
			slog.Info("Termination signal received, shutting down...")
			return nil
		}

		if err := connectToServers(); err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}
	} else {
		// Direct connection mode (no health monitoring)
		slog.Info("Connecting to VPN server")
		err = vpn.Connect(clientLink)
		if err != nil {
			log.Fatal(err)
		}
		slog.Info("Connected to VPN server")

		<-sigterm
		slog.Info("Received term signal, disconnecting...")
		if err = vpn.Disconnect(context.Background()); err != nil {
			slog.Warn("Disconnecting VPN failed", "error", err)
			os.Exit(0)
		}

		slog.Info("VPN disconnected successfully")
		os.Exit(0)
	}
}

// startPeriodicRefresh periodically fetches new server list and updates connection
func startPeriodicRefresh(
	ctx context.Context,
	currentConnector *client.VPNConnector,
	selector *client.ServerSelector,
	rawURL string,
	interval time.Duration,
	timeout time.Duration,
	maxServers int,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Periodic refresh stopped due to context cancellation")
			return
		case <-ticker.C:
			slog.Info("Refreshing server list from raw URL", "url", rawURL)

			links, err := selector.FetchRawLinks(rawURL)
			if err != nil {
				slog.Warn("Failed to refresh server list", "error", err)
				continue
			}

			servers, err := selector.SelectAllByLatency(links)
			if err != nil {
				slog.Warn("Failed to select servers from refreshed list", "error", err)
				links = nil // Explicitly clear old data to allow GC
				continue
			}

			slog.Info("Server list refreshed successfully", "total_servers", len(servers), "new_servers_available", len(servers))

			// Note: Current implementation logs the update but doesn't force reconnect
			// The health monitoring system will automatically switch to better servers if needed
			currentConnector.LogServerReport(servers)
		}
	}
}

// createLogger creates a slog logger with JSON/text format and optional file rotation
func createLogger(format string, level string, logFile string, maxSize int, maxBackups int, maxAge int) (*slog.Logger, func()) {
	// Parse log level
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	handlerOpts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	var writers []io.Writer
	var cleanupFuncs []func()

	// Always log to stdout
	writers = append(writers, os.Stdout)

	// If log file specified, add file writer with rotation
	var fileWriter io.WriteCloser
	if logFile != "" {
		// Ensure directory exists
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Fatalf("Failed to create log directory %s: %v", logDir, err)
		}

		// Create lumberjack logger for rotation
		fileWriter = &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    maxSize,    // megabytes
			MaxBackups: maxBackups, // number of backup files
			MaxAge:     maxAge,     // days
			Compress:   true,       // compress old logs
		}
		
		writers = append(writers, fileWriter)
		
		// Add cleanup function to close file
		cleanupFuncs = append(cleanupFuncs, func() {
			if err := fileWriter.Close(); err != nil {
				log.Printf("Warning: failed to close log file: %v", err)
			}
		})
	}

	// Create multi-writer
	multiWriter := io.MultiWriter(writers...)

	// Create handler based on format
	var handler slog.Handler
	if strings.ToLower(format) == "json" {
		handler = slog.NewJSONHandler(multiWriter, handlerOpts)
	} else {
		handler = slog.NewTextHandler(multiWriter, handlerOpts)
	}

	logger := slog.New(handler)
	
	// Set as default logger for slog package
	slog.SetDefault(logger)

	// Combined cleanup function
	cleanup := func() {
		for _, fn := range cleanupFuncs {
			fn()
		}
	}

	return logger, cleanup
}
