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
	h.logger.Info("Starting health checks", "host", serverHost, "port", serverPort,
		"interval", h.checkInterval, "timeout", h.timeout, "max_retries", h.maxRetries)

	go h.healthCheckLoop(ctx, serverHost, serverPort, onUnhealthy)
}

// Stop stops health checking
func (h *HealthChecker) Stop() {
	close(h.stopChan)
}

// IsHealthy returns current health status
func (h *HealthChecker) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isHealthy
}

// healthCheckLoop runs periodic health checks
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
			if err := h.checkHealth(host, port); err != nil {
				h.handleUnhealthy(err, onUnhealthy)
			} else {
				h.markHealthy()
			}
		}
	}
}

// checkHealth performs a single health check
func (h *HealthChecker) checkHealth(host string, port string) error {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	addr := net.JoinHostPort(host, port)
	
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	// Set deadline for response
	if err := conn.SetReadDeadline(time.Now().Add(h.timeout)); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}

	// Try to read/write to verify connection is working
	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err != nil && err != net.ErrClosed {
		// Connection reset or other error might indicate issues
		h.logger.Debug("Health check read warning", "error", err)
	}

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
