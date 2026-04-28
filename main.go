package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goxray/tun/pkg/client"
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
`

func main() {
	// Get connection link from cmd arguments
	var clientLink string
	var fromRaw bool
	var rawURL string
	var refreshInterval time.Duration = 0 // Default: no refresh
	var maxServers int = 10
	var timeout time.Duration = 5 * time.Second

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

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	vpn, err := client.NewClientWithOpts(client.Config{
		TLSAllowInsecure: false,
		Logger:           logger,
	})
	if err != nil {
		log.Fatal(err)
	}

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

			report := connector.GetConnectionReport(servers)
			slog.Info("Server selection results:\n" + report)

			slog.Info("Attempting VPN connection with fallback support", "servers_count", len(servers))
			if connErr := connector.ConnectWithFallback(servers); connErr != nil {
				return fmt.Errorf("connect: %w", connErr)
			}

			// Start periodic server list refresh if enabled
			if refreshInterval > 0 {
				slog.Info("Periodic server list refresh enabled", "interval", refreshInterval)
				go startPeriodicRefresh(connector, selector, rawURL, refreshInterval, timeout, maxServers)
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
						return
					}
				}
			}()

			// Wait for termination signal
			<-sigterm
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
	currentConnector *client.VPNConnector,
	selector *client.ServerSelector,
	rawURL string,
	interval time.Duration,
	timeout time.Duration,
	maxServers int,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		slog.Info("Refreshing server list from raw URL", "url", rawURL)

		links, err := selector.FetchRawLinks(rawURL)
		if err != nil {
			slog.Warn("Failed to refresh server list", "error", err)
			continue
		}

		servers, err := selector.SelectAllByLatency(links)
		if err != nil {
			slog.Warn("Failed to select servers from refreshed list", "error", err)
			continue
		}

		slog.Info("Server list refreshed successfully", "total_servers", len(servers), "new_servers_available", len(servers))

		// Note: Current implementation logs the update but doesn't force reconnect
		// The health monitoring system will automatically switch to better servers if needed
		report := currentConnector.GetConnectionReport(servers)
		slog.Info("Updated server list:\n" + report)
	}
}
