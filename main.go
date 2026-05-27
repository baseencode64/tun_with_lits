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
usage: %s [options]
  - Direct link: <vless://example...>
  - Config file: --config <path/to/config.yaml>
  - Raw list URL: --from-raw <https://example.com/links.txt>
  - or set GOXRAY_CONFIG_URL env var

Options for --from-raw mode:
  --refresh-interval <duration> - periodically refresh server list (e.g., 5m, 10m, 1h)
                                  default: 0 (no refresh)
  --max-servers <n>             - maximum number of servers to check (default: 10)
  --timeout <duration>          - timeout per server health check (default: 5s)
  --ipv6                        - enable IPv6 support (default: false)
  --dns-protection              - enable DNS leak protection (default: false)

Reconnection options (auto-reconnect when all servers fail):
  --max-retries <n>             - max reconnection attempts (0 = unlimited, default: 0)
  --min-backoff <duration>      - initial backoff between reconnections (default: 5s)
  --max-backoff <duration>      - max backoff between reconnections (default: 5m)
  --backoff-factor <factor>     - exponential backoff multiplier (default: 2.0)

Prometheus metrics:
  --metrics-port <port>         - enable Prometheus metrics endpoint (default: 0 = disabled)

Logging options:
  --log-format <format>         - log format: text or json (default: text)
  --log-level <level>           - log level: debug, info, warn, error (default: info)
  --log-file <path>             - log file path (optional, default: stdout only)
  --log-max-size <MB>           - max log file size in MB before rotation (default: 100)
  --log-max-backups <count>     - max number of backup log files (default: 3)
  --log-max-age <days>          - max age of backup logs in days (default: 28)

Config file option:
  --config <path>               - load configuration from YAML file
                                  CLI arguments override config file values
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
	var enableDNSProtection bool = false

	// Prometheus metrics configuration
	var metricsPort int = 0 // Default: disabled

	// Logging configuration
	var logFormat string = "text"
	var logLevel string = "info"
	var logFile string = ""
	var logMaxSize int = 100
	var logMaxBackups int = 3
	var logMaxAge int = 28

	// Reconnection configuration
	var reconnectMaxRetries int = client.DefaultMaxRetries     // 0 = unlimited
	var reconnectMinBackoff time.Duration = client.DefaultMinBackoff   // 5s
	var reconnectMaxBackoff time.Duration = client.DefaultMaxBackoff   // 5m
	var reconnectBackoffFactor float64 = client.DefaultBackoffFactor   // 2.0

	// E2E health check configuration
	var e2eCheckURL string = ""  // Default: SOCKS-only health check

	// Config file path
	var configFile string = ""

	args := os.Args[1:]

	// Parse arguments with support for flags
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--config":
			if i+1 >= len(args) {
				log.Fatal("--config requires a file path")
			}
			i++
			configFile = args[i]
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
		case "--dns-protection":
			enableDNSProtection = true
		case "--metrics-port":
			if i+1 >= len(args) {
				log.Fatal("--metrics-port requires a port number")
			}
			i++
			fmt.Sscanf(args[i], "%d", &metricsPort)
			if metricsPort <= 0 || metricsPort > 65535 {
				log.Fatal("--metrics-port must be between 1 and 65535")
			}
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
		case "--max-retries":
			if i+1 >= len(args) {
				log.Fatal("--max-retries requires a value")
			}
			i++
			fmt.Sscanf(args[i], "%d", &reconnectMaxRetries)
			if reconnectMaxRetries < 0 {
				log.Fatal("--max-retries cannot be negative (0 = unlimited)")
			}
		case "--min-backoff":
			if i+1 >= len(args) {
				log.Fatal("--min-backoff requires a duration (e.g., 5s, 10s)")
			}
			i++
			var err error
			reconnectMinBackoff, err = time.ParseDuration(args[i])
			if err != nil {
				log.Fatalf("Invalid min-backoff format: %v", err)
			}
		case "--max-backoff":
			if i+1 >= len(args) {
				log.Fatal("--max-backoff requires a duration (e.g., 5m, 10m)")
			}
			i++
			var err error
			reconnectMaxBackoff, err = time.ParseDuration(args[i])
			if err != nil {
				log.Fatalf("Invalid max-backoff format: %v", err)
			}
		case "--backoff-factor":
			if i+1 >= len(args) {
				log.Fatal("--backoff-factor requires a value (e.g., 2.0)")
			}
			i++
			fmt.Sscanf(args[i], "%f", &reconnectBackoffFactor)
			if reconnectBackoffFactor <= 1.0 {
				log.Fatal("--backoff-factor must be > 1.0")
			}
		case "--e2e-check-url":
			if i+1 >= len(args) {
				log.Fatal("--e2e-check-url requires a URL (e.g., http://ipinfo.io/ip)")
			}
			i++
			e2eCheckURL = args[i]
		default:
			// Positional argument (direct link)
			if clientLink == "" {
				clientLink = args[i]
			}
		}
		i++
	}

	// Load config file if specified
	var appConfig *client.AppConfig
	if configFile != "" {
		slog.Info("Loading configuration from file", "path", configFile)
		var err error
		appConfig, err = client.LoadConfig(configFile)
		if err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}
		slog.Info("Configuration loaded successfully")
	}

	// Apply config file values as defaults, CLI args override them
	var rawURLsFromConfig []string // Store all URLs from config for fallback
	
	if appConfig != nil {
		// Connection settings (CLI overrides config)
		if clientLink == "" {
			clientLink = appConfig.Connection.Link
		}
		
		// Handle from_raw_urls (multiple URLs with fallback)
		if !fromRaw && len(appConfig.Connection.FromRawURLs) > 0 {
			fromRaw = true
			rawURLsFromConfig = appConfig.Connection.FromRawURLs
			rawURL = appConfig.Connection.FromRawURLs[0] // Use first as primary
			slog.Info("Using multiple raw URLs from config", "primary_url", rawURL, "fallback_count", len(rawURLsFromConfig)-1)
		} else if !fromRaw && appConfig.Connection.FromRaw != "" {
			// Legacy single from_raw support
			fromRaw = true
			rawURL = appConfig.Connection.FromRaw
		}
		
		if !enableIPv6 && appConfig.Connection.EnableIPv6 {
			enableIPv6 = true
		}
		
		if !enableDNSProtection && appConfig.Connection.EnableDNSProtection {
			enableDNSProtection = true
		}

		// Server selection settings (CLI overrides config)
		if refreshInterval == 0 && appConfig.ServerSelection.RefreshInterval != "" {
			var err error
			refreshInterval, err = appConfig.ServerSelection.GetRefreshInterval()
			if err != nil {
				log.Fatalf("Invalid refresh interval in config: %v", err)
			}
		}
		if maxServers == 10 && appConfig.ServerSelection.MaxServers != 10 {
			maxServers = appConfig.ServerSelection.MaxServers
		}
		if timeout == 5*time.Second && appConfig.ServerSelection.Timeout != "" {
			var err error
			timeout, err = appConfig.ServerSelection.GetTimeout()
			if err != nil {
				log.Fatalf("Invalid timeout in config: %v", err)
			}
		}

		// Reconnection settings (CLI overrides config)
		if reconnectMaxRetries == client.DefaultMaxRetries && appConfig.Reconnection.MaxRetries > 0 {
			reconnectMaxRetries = appConfig.Reconnection.MaxRetries
		}
		if reconnectMinBackoff == client.DefaultMinBackoff && appConfig.Reconnection.MinBackoff > 0 {
			reconnectMinBackoff = appConfig.Reconnection.MinBackoff
		}
		if reconnectMaxBackoff == client.DefaultMaxBackoff && appConfig.Reconnection.MaxBackoff > 0 {
			reconnectMaxBackoff = appConfig.Reconnection.MaxBackoff
		}
		if reconnectBackoffFactor == client.DefaultBackoffFactor && appConfig.Reconnection.BackoffFactor > 1.0 {
			reconnectBackoffFactor = appConfig.Reconnection.BackoffFactor
		}

		// Logging settings (CLI overrides config)
		if logFormat == "text" && appConfig.Logging.Format != "text" {
			logFormat = appConfig.Logging.Format
		}
		if logLevel == "info" && appConfig.Logging.Level != "info" {
			logLevel = appConfig.Logging.Level
		}
		if logFile == "" && appConfig.Logging.File != "" {
			logFile = appConfig.Logging.File
		}
		if logMaxSize == 100 && appConfig.Logging.MaxSize != 100 {
			logMaxSize = appConfig.Logging.MaxSize
		}
		if logMaxBackups == 3 && appConfig.Logging.MaxBackups != 3 {
			logMaxBackups = appConfig.Logging.MaxBackups
		}
		if logMaxAge == 28 && appConfig.Logging.MaxAge != 28 {
			logMaxAge = appConfig.Logging.MaxAge
		}
		
		// E2E health check settings (CLI overrides config)
		if e2eCheckURL == "" && appConfig.Connection.E2ECheckURL != "" {
			e2eCheckURL = appConfig.Connection.E2ECheckURL
		}

		// Metrics settings (CLI overrides config)
		if metricsPort == 0 && appConfig.Connection.MetricsPort > 0 {
			metricsPort = appConfig.Connection.MetricsPort
		}
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
		TLSAllowInsecure:    false,
		Logger:              logger,
		EnableIPv6:          enableIPv6,
		MetricsPort:         metricsPort,
		EnableDNSProtection: enableDNSProtection,
		E2ECheckURL:         e2eCheckURL,
	})
	if err != nil {
		log.Fatal(err)
	}

	slog.Info("VPN client initialized", "metrics_port", metricsPort)
	
	if enableIPv6 {
		slog.Info("IPv6 support enabled")
	}
	
	if enableDNSProtection {
		slog.Info("DNS leak protection enabled")
	}
	
	if metricsPort > 0 {
		slog.Info("Prometheus metrics enabled", "port", metricsPort, "endpoint", fmt.Sprintf("http://0.0.0.0:%d/metrics", metricsPort))
	}

	// Log configuration info
	slog.Info("Logging configuration", 
		"format", logFormat, 
		"level", logLevel,
		"file", logFile,
		"max_size_mb", logMaxSize,
		"max_backups", logMaxBackups,
		"max_age_days", logMaxAge)

	// Log reconnection configuration
	slog.Info("Reconnection configuration",
		"max_retries", reconnectMaxRetries,
		"min_backoff", reconnectMinBackoff,
		"max_backoff", reconnectMaxBackoff,
		"backoff_factor", reconnectBackoffFactor)

	// Log E2E health check configuration
	if e2eCheckURL != "" {
		slog.Info("E2E health check enabled", "check_url", e2eCheckURL)
	} else {
		slog.Info("E2E health check disabled (using SOCKS-only check)")
	}

	// If using raw URL(s), fetch and select servers with fallback support
	if fromRaw {
		// Build list of URLs to try (support multiple URLs with fallback)
		var rawURLs []string
		
		// Check if config has multiple URLs
		if len(rawURLsFromConfig) > 0 {
			rawURLs = rawURLsFromConfig
			slog.Info("Using multiple raw URLs from config", "urls_count", len(rawURLs))
		} else if rawURL != "" {
			// Single URL from CLI or legacy config
			rawURLs = []string{rawURL}
			slog.Info("Fetching server list from raw URL", "url", rawURL, "refresh_interval", refreshInterval)
		} else {
			log.Fatal("No raw URL specified")
		}

		loggerAdapter := client.NewSlogAdapter(logger)
		selector := client.NewServerSelector(loggerAdapter, timeout, maxServers)

		// Reconnection config
		reconnCfg := client.ReconnectionConfig{
			MaxRetries:    reconnectMaxRetries,
			MinBackoff:    reconnectMinBackoff,
			MaxBackoff:    reconnectMaxBackoff,
			BackoffFactor: reconnectBackoffFactor,
		}

		// createConnector creates a new VPN connector and connects to servers.
		// Returns the connector on success, or error if all servers failed.
		createConnector := func(ctx context.Context) (*client.VPNConnector, error) {
			var links []string
			var lastFetchErr error
			
			// Try each URL in order until one succeeds
			for i, url := range rawURLs {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				default:
				}

				slog.Info("Attempting to fetch server list", "url_index", i+1, "total_urls", len(rawURLs), "url", url)
				
				var fetchErr error
				links, fetchErr = selector.FetchRawLinks(url)
				if fetchErr == nil {
					slog.Info("Successfully fetched server list", "url_index", i+1, "links_count", len(links))
					break
				}
				
				lastFetchErr = fetchErr
				slog.Warn("Failed to fetch from URL, trying next", "url_index", i+1, "url", url, "error", fetchErr)
			}
			
			// All URLs failed
			if len(links) == 0 {
				return nil, fmt.Errorf("failed to fetch from all %d URLs, last error: %w", len(rawURLs), lastFetchErr)
			}

			servers, selectErr := selector.SelectAllByScore(links)
			if selectErr != nil {
				return nil, fmt.Errorf("select servers: %w", selectErr)
			}

			connector := client.NewVPNConnectorWithReconnect(vpn, selector, logger, reconnCfg)

			// Log server report in structured format (JSON-compatible)
			connector.LogServerReport(servers)

			slog.Info("Attempting VPN connection with fallback support", "servers_count", len(servers))
			if connErr := connector.ConnectWithFallback(servers); connErr != nil {
				connector.Stop()
				return nil, fmt.Errorf("connect: %w", connErr)
			}

			// Log initial connection status with IP information
			slog.Info("VPN connected successfully, logging connection details")
			vpn.LogConnectionStatus()

			return connector, nil
		}

		// Initial connection attempt
		connector, err := createConnector(context.Background())
		if err != nil {
			slog.Warn("Initial connection failed, starting reconnection loop", "error", err)
			
			// Set up reconnection with the connector creation as the reconnect function
			reconCtx, reconCancel := context.WithCancel(context.Background())
			defer reconCancel()

			reconnectFunc := func(ctx context.Context) error {
				// Create a fresh connector
				newConnector, connErr := createConnector(ctx)
				if connErr != nil {
					return connErr
				}
				// Store for use below
				connector = newConnector
				return nil
			}

			refreshFunc := func(ctx context.Context) error {
				// Just a placeholder - createConnector already re-fetches the server list.
				// This is here for future use, e.g., DNS refresh or different fetch strategy.
				slog.Debug("Reconnection refresh: will re-fetch server list on next attempt")
				return nil
			}

			reconnector := client.NewReconnector(logger, reconnCfg, reconnectFunc, refreshFunc)
			
			if err := reconnector.Start(reconCtx); err != nil {
				if err == client.ErrReconnectionStopped {
					slog.Info("Reconnection loop stopped")
				} else {
					log.Fatalf("Reconnection failed: %v", err)
				}
				return
			}
		}

		defer connector.Stop()

		// Create a cancellable context for monitoring goroutines
		monitorCtx, monitorCancel := context.WithCancel(context.Background())
		
		// Ensure cleanup happens on function exit
		defer func() {
			slog.Info("Stopping all monitoring goroutines")
			monitorCancel()
		}()

		// Start periodic server list refresh if enabled
		if refreshInterval > 0 {
			// Use first URL for periodic refresh
			successfulURL := rawURLs[0]
			slog.Info("Periodic server list refresh enabled", "interval", refreshInterval, "url", successfulURL)
			go startPeriodicRefresh(monitorCtx, connector, selector, successfulURL, refreshInterval, timeout, maxServers)
		}

		// Monitor health status and connection info periodically
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			
			// Log initial connection status immediately
			vpn.LogConnectionStatus()
			
			for {
				select {
				case <-ticker.C:
					// Log health status
					status := connector.GetHealthStatus()
					slog.Info("VPN Health Status", "status", status)
					
					// Log connection status with IP info
					vpn.LogConnectionStatus()
				case <-sigterm:
					monitorCancel() // Signal other goroutines to stop
					return
				case <-monitorCtx.Done():
					return // Context cancelled (e.g., during shutdown)
				}
			}
		}()

		// Wait for termination signal or fatal error from connector in a loop
		for {
			select {
			case <-sigterm:
				slog.Info("Termination signal received, shutting down...")
				return
			case err := <-connector.FatalError():
				slog.Error("Fatal connector error", "error", err)
				
				if err == client.ErrAllServersExhausted {
					slog.Info("All servers exhausted. Initiating full reconnection loop.")
					
					// Stop the current connector and its monitoring
					monitorCancel()
					connector.Stop()

					// Set up reconnection with the connector creation as the reconnect function
					reconCtx, reconCancel := context.WithCancel(context.Background())

					reconnectFunc := func(ctx context.Context) error {
						// Create a fresh connector
						newConnector, connErr := createConnector(ctx)
						if connErr != nil {
							return connErr
						}
						// Update references for next loop
						connector = newConnector
						
						// Re-start monitoring
						monitorCtx, monitorCancel = context.WithCancel(context.Background())
						
						if refreshInterval > 0 {
							go startPeriodicRefresh(monitorCtx, connector, selector, rawURLs[0], refreshInterval, timeout, maxServers)
						}
						
						// Re-start the ticker for health status logging
						go func() {
							ticker := time.NewTicker(30 * time.Second)
							defer ticker.Stop()
							vpn.LogConnectionStatus()
							for {
								select {
								case <-ticker.C:
									status := connector.GetHealthStatus()
									slog.Info("VPN Health Status", "status", status)
									vpn.LogConnectionStatus()
								case <-sigterm:
									monitorCancel()
									return
								case <-monitorCtx.Done():
									return
								}
							}
						}()
						
						return nil
					}

					refreshFunc := func(ctx context.Context) error {
						slog.Debug("Reconnection refresh: will re-fetch server list on next attempt")
						return nil
					}

					reconnector := client.NewReconnector(logger, reconnCfg, reconnectFunc, refreshFunc)
					
					// Handle signal while reconnecting
					go func() {
						<-sigterm
						slog.Info("Termination signal received during reconnection")
						reconCancel()
					}()

					if err := reconnector.Start(reconCtx); err != nil {
						if err == client.ErrReconnectionStopped || err == context.Canceled {
							slog.Info("Reconnection loop stopped")
						} else {
							log.Fatalf("Reconnection failed: %v", err)
						}
						reconCancel()
						return
					} else {
						slog.Info("Successfully fully reconnected. Monitoring active.")
						reconCancel()
						// Continue the outer loop to wait on the new connector.FatalError() or sigterm
					}
				} else {
					return
				}
			}
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
// Supports multiple URLs with fallback - tries each URL in order on failure
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
				slog.Warn("Failed to refresh server list", "error", err, "url", rawURL)
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