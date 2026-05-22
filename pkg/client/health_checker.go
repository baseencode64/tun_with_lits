package client

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

// HealthChecker monitors VPN server health
type HealthChecker struct {
	logger        *slog.Logger
	checkInterval time.Duration
	timeout       time.Duration
	maxRetries    int
	
	mu          sync.RWMutex
	isHealthy   bool
	lastCheck   time.Time
	consecutiveFailures int
	stopChan    chan struct{}
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(logger *slog.Logger, checkInterval time.Duration, timeout time.Duration, maxRetries int) *HealthChecker {
	return &HealthChecker{
		logger:        logger,
		checkInterval: checkInterval,
		timeout:       timeout,
		maxRetries:    maxRetries,
		isHealthy:     true,
		stopChan:      make(chan struct{}),
	}
}

// Start begins health checking loop
func (h *HealthChecker) Start(ctx context.Context, serverHost string, serverPort string, onUnhealthy func()) {
	// For SOCKS proxy health check, we use the same port but on localhost
	socksHost := "127.0.0.1"
	h.logger.Info("Starting health checks", "server", serverHost+":"+serverPort, 
		"socks_proxy", socksHost+":"+serverPort,
		"interval", h.checkInterval, "timeout", h.timeout, "max_retries", h.maxRetries)

	go h.healthCheckLoop(ctx, socksHost, serverPort, onUnhealthy)
}

// Stop stops health checking with proper synchronization
func (h *HealthChecker) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Close stopChan only once to prevent panic
	select {
	case <-h.stopChan:
		// Already closed
	default:
		close(h.stopChan)
	}
}

// IsHealthy returns current health status
func (h *HealthChecker) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isHealthy
}

// healthCheckLoop runs periodic health checks with proper context handling
func (h *HealthChecker) healthCheckLoop(ctx context.Context, host string, port string, onUnhealthy func()) {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Health check stopped due to context cancellation")
			return
		case <-h.stopChan:
			h.logger.Info("Health check stopped")
			return
		case <-ticker.C:
			// Check if we should still proceed
			select {
			case <-ctx.Done():
				h.logger.Info("Health check cancelled before execution")
				return
			case <-h.stopChan:
				h.logger.Info("Health check stopped before execution")
				return
			default:
			}
			
			if err := h.checkHealth(host, port); err != nil {
				h.handleUnhealthy(err, onUnhealthy)
			} else {
				h.markHealthy()
			}
		}
	}
}

// checkHealth performs a single health check by testing SOCKS proxy connectivity
func (h *HealthChecker) checkHealth(host string, port string) error {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	// Check SOCKS proxy instead of remote server
	// This is more reliable as it tests the actual tunnel functionality
	socksAddr := net.JoinHostPort("127.0.0.1", port)
	
	startTime := time.Now()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", socksAddr)
	if err != nil {
		return fmt.Errorf("SOCKS proxy dial failed after %v: %w", time.Since(startTime), err)
	}
	defer conn.Close()

	dialDuration := time.Since(startTime)
	h.logger.Debug("Health check: SOCKS proxy dial successful", "duration_ms", dialDuration.Milliseconds())

	// Set deadline for response
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}

	// Send SOCKS5 greeting to verify proxy is responsive
	// Version 5, 1 auth method (no auth)
	greeting := []byte{0x05, 0x01, 0x00}
	writeStart := time.Now()
	_, err = conn.Write(greeting)
	writeDuration := time.Since(writeStart)
	
	if err != nil {
		return fmt.Errorf("failed to write SOCKS greeting after %v: %w", writeDuration, err)
	}

	// Read response
	buf := make([]byte, 2)
	readStart := time.Now()
	_, err = conn.Read(buf)
	readDuration := time.Since(readStart)
	
	if err != nil {
		return fmt.Errorf("failed to read SOCKS response after %v: %w", readDuration, err)
	}

	// Verify SOCKS5 response
	if buf[0] != 0x05 || buf[1] != 0x00 {
		return fmt.Errorf("invalid SOCKS response: version=%d, method=%d", buf[0], buf[1])
	}

	totalDuration := time.Since(startTime)
	h.logger.Debug("Health check successful",
		"dial_ms", dialDuration.Milliseconds(),
		"write_ms", writeDuration.Milliseconds(),
		"read_ms", readDuration.Milliseconds(),
		"total_ms", totalDuration.Milliseconds())

	return nil
}

// handleUnhealthy processes failed health check
func (h *HealthChecker) handleUnhealthy(err error, onUnhealthy func()) {
	h.mu.Lock()
	h.consecutiveFailures++
	failures := h.consecutiveFailures
	h.mu.Unlock()

	h.logger.Warn("Health check failed", 
		"attempt", failures,
		"max_retries", h.maxRetries,
		"error", err)

	if failures >= h.maxRetries {
		h.logger.Error("Server unhealthy - exceeded max retries", "failures", failures)
		
		h.mu.Lock()
		wasHealthy := h.isHealthy
		h.isHealthy = false
		h.mu.Unlock()

		// Call onUnhealthy OUTSIDE the lock to prevent deadlock
		if wasHealthy && onUnhealthy != nil {
			h.logger.Info("Triggering failover to next server")
			onUnhealthy()
		}
	}
}

// markHealthy resets health status after successful check
func (h *HealthChecker) markHealthy() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.isHealthy {
		h.logger.Info("Server recovered - marked as healthy again")
	}
	
	h.isHealthy = true
	h.consecutiveFailures = 0
	h.lastCheck = time.Now()
}

// GetStatus returns detailed health status
func (h *HealthChecker) GetStatus() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"is_healthy":            h.isHealthy,
		"consecutive_failures":  h.consecutiveFailures,
		"last_check":            h.lastCheck,
		"check_interval":        h.checkInterval,
		"max_retries":           h.maxRetries,
	}
}
