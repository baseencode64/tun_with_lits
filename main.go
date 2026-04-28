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
usage: %s <config_url_or_link>
  - config_url - xray connection link, like "vless://example..."
  - or raw list URL: --from-raw <https://example.com/links.txt>
  - or set GOXRAY_CONFIG_URL env var
`

func main() {
	// Get connection link from cmd arguments
	var clientLink string
	var fromRaw bool
	var rawURL string

	args := os.Args[1:]

	// Check for --from-raw flag
	if len(args) > 0 && args[0] == "--from-raw" {
		fromRaw = true
		if len(args) < 2 {
			fmt.Printf(cmdArgsErr, os.Args[0])
			os.Exit(1)
		}
		rawURL = args[1]
	} else if len(args) > 0 {
		clientLink = args[0]
	} else {
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
		slog.Info("Fetching server list from raw URL", "url", rawURL)

		loggerAdapter := client.NewSlogAdapter(logger)
		selector := client.NewServerSelector(loggerAdapter, 5*time.Second, 10)
		
		// Fetch all available servers sorted by latency
		links, fetchErr := selector.FetchRawLinks(rawURL)
		if fetchErr != nil {
			log.Fatalf("Failed to fetch links: %v", fetchErr)
		}

		servers, selectErr := selector.SelectAllByLatency(links)
		if selectErr != nil {
			log.Fatalf("Failed to select servers: %v", selectErr)
		}

		// Create VPN connector with fallback support
		connector := client.NewVPNConnector(vpn, selector, logger)
		
		// Show connection report
		report := connector.GetConnectionReport(servers)
		slog.Info("Server selection results:\n" + report)

		// Connect with automatic fallback
		slog.Info("Attempting VPN connection with fallback support", "servers_count", len(servers))
		if connErr := connector.ConnectWithFallback(servers); connErr != nil {
			log.Fatalf("Failed to connect: %v", connErr)
		}
	} else {
		// Direct connection mode
		slog.Info("Connecting to VPN server")
		err = vpn.Connect(clientLink)
		if err != nil {
			log.Fatal(err)
		}
		slog.Info("Connected to VPN server")
	}

	<-sigterm
	slog.Info("Received term signal, disconnecting...")
	if err = vpn.Disconnect(context.Background()); err != nil {
		slog.Warn("Disconnecting VPN failed", "error", err)
		os.Exit(0)
	}

	slog.Info("VPN disconnected successfully")
	os.Exit(0)
}
