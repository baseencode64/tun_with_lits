package client

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
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
	
	mu               sync.Mutex  // Protects concurrent access during failover
	currentServerIndex int
	servers            []*ServerInfo
	isFailingOver      bool  // Prevents concurrent failover attempts
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

		// Add timeout for each connection attempt to prevent hanging
		connectCtx, connectCancel := context.WithTimeout(c.ctx, 30*time.Second)
		done := make(chan error, 1)
		
		go func() {
			done <- c.client.Connect(server.Link)
		}()
		
		var err error
		select {
		case err = <-done:
			// Connection completed
		case <-connectCtx.Done():
			err = fmt.Errorf("connection timeout after 30s")
			c.logger.Warn("Connection attempt timed out", "host", server.Host, "port", server.Port)
		}
		connectCancel()
		
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
	// Stop previous health checker if exists to prevent goroutine leaks
	if c.healthChecker != nil {
		c.logger.Info("Stopping previous health checker before starting new one")
		c.healthChecker.Stop()
		// Wait a bit to ensure goroutine exits
		time.Sleep(100 * time.Millisecond)
		c.healthChecker = nil
	}
	
	// Create health checker with default settings
	c.healthChecker = NewHealthChecker(
		c.logger,
		10*time.Second,  // check interval
		5*time.Second,   // timeout
		3,               // max retries before failover
	)

	// Start health checks with automatic failover
	// Use sync.Once to prevent multiple concurrent failover triggers
	var failoverOnce sync.Once
	c.healthChecker.Start(c.ctx, server.Host, server.Port, func() {
		failoverOnce.Do(func() {
			c.logger.Warn("Health check failed - initiating automatic failover")
			c.performFailover()
		})
	})
}

// performFailover switches to next available server with proper synchronization
func (c *VPNConnector) performFailover() {
	// Prevent concurrent failover attempts using atomic operation
	c.mu.Lock()
	if c.isFailingOver {
		c.logger.Warn("Failover already in progress, skipping")
		c.mu.Unlock()
		return
	}
	c.isFailingOver = true
	
	// Save old cancel func and unlock immediately
	oldCancel := c.cancelFunc
	c.mu.Unlock()
	
	// Cancel all previous operations immediately to stop goroutines
	if oldCancel != nil {
		c.logger.Info("Cancelling previous operations before failover")
		oldCancel()
	}
	
	defer func() {
		c.mu.Lock()
		c.isFailingOver = false
		c.mu.Unlock()
	}()

	// Try to failover in a loop (NOT recursively!)
	for {
		c.mu.Lock()
		currentIndex := c.currentServerIndex
		totalServers := len(c.servers)
		
		if totalServers <= currentIndex+1 {
			c.mu.Unlock()
			c.logger.Error("No more servers to failover to", 
				"current_index", currentIndex, "total_servers", totalServers)
			return
		}

		nextIndex := currentIndex + 1
		
		// Check bounds
		if nextIndex >= totalServers {
			c.mu.Unlock()
			c.logger.Error("Failed to failover - no valid next server")
			return
		}
		
		nextServer := c.servers[nextIndex]
		c.mu.Unlock()

		c.logger.Info("Failing over to next server",
			"from_index", currentIndex,
			"to_index", nextIndex,
			"next_host", nextServer.Host,
			"next_port", nextServer.Port)
		
		// Stop health checker BEFORE disconnect to prevent stale callbacks
		if c.healthChecker != nil {
			c.logger.Info("Stopping old health checker")
			c.healthChecker.Stop()
			c.healthChecker = nil // Clear reference to allow GC
		}

		// Disconnect from current server with timeout (reduced from 30s to 5s)
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 5*time.Second)
		
		if err := c.client.Disconnect(disconnectCtx); err != nil {
			c.logger.Warn("Disconnect warning (continuing with failover)", "error", err)
		}
		disconnectCancel()

		// Reduced delay to allow cleanup but prevent connection races
		time.Sleep(500 * time.Millisecond)

		// Create new context for the new connection
		newCtx, newCancel := context.WithCancel(context.Background())
		
		c.mu.Lock()
		c.ctx = newCtx
		c.cancelFunc = newCancel
		c.currentServerIndex = nextIndex
		c.mu.Unlock()

		// Try to connect to next server with NEW context
		c.logger.Info("Connecting to next server", "host", nextServer.Host, "port", nextServer.Port)
		
		// Use a channel to make Connect cancellable with the new context
		connectDone := make(chan error, 1)
		go func() {
			connectDone <- c.client.Connect(nextServer.Link)
		}()
		
		var err error
		select {
		case err = <-connectDone:
			// Connection completed
		case <-newCtx.Done():
			err = fmt.Errorf("connection cancelled by context")
			c.logger.Warn("Connection cancelled during failover")
		}
		
		if err != nil {
			c.logger.Error("Failed to connect to next server", "error", err, "host", nextServer.Host, "port", nextServer.Port)
			
			// Continue loop to try next server (NO RECURSION!)
			c.mu.Lock()
			remainingServers := c.currentServerIndex < len(c.servers)-1
			c.mu.Unlock()
			
			if remainingServers {
				c.logger.Info("Trying next server in list")
				// Small delay before next attempt
				time.Sleep(200 * time.Millisecond)
				// Loop continues automatically
				continue
			} else {
				c.logger.Error("Exhausted all servers in failover")
				return
			}
		}

		c.logger.Info("Successfully failed over to next server",
			"host", nextServer.Host, "port", nextServer.Port, "index", nextIndex)

		// Start health monitoring for new server ONLY after successful connection
		c.startHealthMonitoring(nextServer)
		return // Success - exit function
	}
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
