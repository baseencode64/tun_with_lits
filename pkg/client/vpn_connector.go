package client

import (
	"fmt"
	"log/slog"
)

// VPNConnector manages VPN connection with fallback support
type VPNConnector struct {
	client   *Client
	selector *ServerSelector
	logger   *slog.Logger
}

// NewVPNConnector creates a new VPN connector with fallback support
func NewVPNConnector(client *Client, selector *ServerSelector, logger *slog.Logger) *VPNConnector {
	return &VPNConnector{
		client:   client,
		selector: selector,
		logger:   logger,
	}
}

// ConnectWithFallback connects to the best available server, trying next ones if connection fails
func (c *VPNConnector) ConnectWithFallback(servers []*ServerInfo) error {
	if len(servers) == 0 {
		return fmt.Errorf("no servers provided")
	}

	c.logger.Info("Attempting VPN connection with fallback support", "servers_count", len(servers))

	var lastErr error
	for i, server := range servers {
		c.logger.Info("Trying server", "attempt", i+1, "total", len(servers),
			"host", server.Host, "port", server.Port, "latency", server.Latency)

		err := c.client.Connect(server.Link)
		if err == nil {
			c.logger.Info("Successfully connected to VPN server",
				"host", server.Host, "port", server.Port,
				"latency", server.Latency, "attempt_number", i+1)
			return nil
		}

		lastErr = err
		c.logger.Warn("Failed to connect to server, trying next",
			"host", server.Host, "port", server.Port,
			"latency", server.Latency, "error", err, "remaining", len(servers)-i-1)
	}

	c.logger.Error("Failed to connect to all servers", "total_tried", len(servers), "last_error", lastErr)
	return fmt.Errorf("failed to connect to %d servers: %w", len(servers), lastErr)
}

// ConnectFromRawURL is a convenience method that fetches servers from URL and connects with fallback
func (c *VPNConnector) ConnectFromRawURL(rawURL string) error {
	c.logger.Info("Fetching server list from raw URL", "url", rawURL)

	servers, err := c.selector.SelectAllByLatency(nil)
	if err != nil {
		// Try fetching first
		links, fetchErr := c.selector.FetchRawLinks(rawURL)
		if fetchErr != nil {
			return fmt.Errorf("fetch links: %w", fetchErr)
		}

		servers, err = c.selector.SelectAllByLatency(links)
		if err != nil {
			return fmt.Errorf("select servers: %w", err)
		}
	}

	return c.ConnectWithFallback(servers)
}

// GetConnectionReport returns a summary of available servers
func (c *VPNConnector) GetConnectionReport(servers []*ServerInfo) string {
	if len(servers) == 0 {
		return "No servers available"
	}

	report := fmt.Sprintf("=== VPN Server Selection Report ===\n")
	report += fmt.Sprintf("Total servers scanned: %d\n", len(servers))
	report += fmt.Sprintf("Available servers: %d\n\n", len(servers))

	for i, srv := range servers {
		status := "✓ Available"
		if i == 0 {
			status = "★ RECOMMENDED"
		}
		report += fmt.Sprintf("%d. %s:%s - Latency: %v - %s\n",
			i+1, srv.Host, srv.Port, srv.Latency, status)
	}

	return report
}
