package client

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// VPNConnector manages VPN connection with fallback support and health monitoring
type VPNConnector struct {
	client        *Client
	selector      *ServerSelector
	logger        *slog.Logger
	healthChecker *HealthChecker
	ctx           context.Context
	cancelFunc    context.CancelFunc
	
	currentServerIndex int
	servers            []*ServerInfo
}

// NewVPNConnector creates a new VPN connector with fallback support
func NewVPNConnector(client *Client, selector *ServerSelector, logger *slog.Logger) *VPNConnector {
	ctx, cancel := context.WithCancel(context.Background())
	return &VPNConnector{
		client:             client,
		selector:           selector,
		logger:             logger,
		ctx:                ctx,
		cancelFunc:         cancel,
		currentServerIndex: -1,
	}
}

// ConnectWithFallback connects to the best available server, trying next ones if connection fails
func (c *VPNConnector) ConnectWithFallback(servers []*ServerInfo) error {
	if len(servers) == 0 {
		return fmt.Errorf("no servers provided")
	}

	c.servers = servers
	c.logger.Info("Attempting VPN connection with fallback support", "servers_count", len(servers))

	var lastErr error
	for i, server := range servers {
		select {
		case <-c.ctx.Done():
			return fmt.Errorf("connection cancelled")
		default:
		}

		c.logger.Info("Trying server", "attempt", i+1, "total", len(servers),
			"host", server.Host, "port", server.Port, "latency", server.Latency)

		err := c.client.Connect(server.Link)
		if err == nil {
			c.currentServerIndex = i
			c.logger.Info("Successfully connected to VPN server",
				"host", server.Host, "port", server.Port,
				"latency", server.Latency, "attempt_number", i+1)
			
			// Start health monitoring
			c.startHealthMonitoring(server)
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

// startHealthMonitoring begins health checking for current server
func (c *VPNConnector) startHealthMonitoring(server *ServerInfo) {
	// Create health checker with default settings
	c.healthChecker = NewHealthChecker(
		c.logger,
		10*time.Second,  // check interval
		5*time.Second,   // timeout
		3,               // max retries before failover
	)

	// Start health checks with automatic failover
	c.healthChecker.Start(c.ctx, server.Host, server.Port, func() {
		c.logger.Warn("Health check failed - initiating automatic failover")
		go c.performFailover()
	})
}

// performFailover switches to next available server
func (c *VPNConnector) performFailover() {
	if len(c.servers) <= c.currentServerIndex+1 {
		c.logger.Error("No more servers to failover to", 
			"current_index", c.currentServerIndex, "total_servers", len(c.servers))
		return
	}

	nextIndex := c.currentServerIndex + 1
	nextServer := c.servers[nextIndex]

	c.logger.Info("Failing over to next server",
		"from_index", c.currentServerIndex,
		"to_index", nextIndex,
		"next_host", nextServer.Host,
		"next_port", nextServer.Port)

	// Disconnect from current server
	if err := c.client.Disconnect(c.ctx); err != nil {
		c.logger.Warn("Failed to disconnect from unhealthy server", "error", err)
	}

	// Try to connect to next server
	c.logger.Info("Connecting to next server", "host", nextServer.Host, "port", nextServer.Port)
	err := c.client.Connect(nextServer.Link)
	if err != nil {
		c.logger.Error("Failed to connect to next server", "error", err)
		// Continue failover recursively
		c.currentServerIndex = nextIndex
		c.performFailover()
		return
	}

	c.currentServerIndex = nextIndex
	c.logger.Info("Successfully failed over to next server",
		"host", nextServer.Host, "port", nextServer.Port, "index", nextIndex)

	// Restart health monitoring for new server
	c.healthChecker.Stop()
	c.startHealthMonitoring(nextServer)
}

// Stop stops health monitoring and cancels all operations
func (c *VPNConnector) Stop() {
	if c.healthChecker != nil {
		c.healthChecker.Stop()
	}
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
}

// GetHealthStatus returns current health status
func (c *VPNConnector) GetHealthStatus() map[string]interface{} {
	status := map[string]interface{}{
		"connected":          c.currentServerIndex >= 0,
		"current_server_idx": c.currentServerIndex,
		"total_servers":      len(c.servers),
	}

	if c.healthChecker != nil {
		status["health"] = c.healthChecker.GetStatus()
	}

	if c.currentServerIndex >= 0 && c.currentServerIndex < len(c.servers) {
		status["current_server"] = c.servers[c.currentServerIndex]
	}

	return status
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
