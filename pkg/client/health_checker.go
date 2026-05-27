package client

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Default E2E check settings
const (
	DefaultE2ECheckURL  = "" // Empty = disabled
	DefaultE2ETimeout   = 10 * time.Second
)

// HealthChecker monitors VPN server health
type HealthChecker struct {
	logger        *slog.Logger
	checkInterval time.Duration
	timeout       time.Duration
	maxRetries    int

	// E2E traffic check settings
	e2eCheckURL    string // External URL to check (e.g., "http://ipinfo.io/ip")
	e2eCheckHost   string // Resolved host for SOCKS5 CONNECT
	e2eRequestStr  string // Raw HTTP request to send

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

// NewHealthCheckerWithE2E creates a new health checker with E2E traffic verification.
// e2eCheckURL should be an HTTP URL (e.g., "http://ipinfo.io/ip").
// If empty, E2E check is skipped (standard SOCKS-only health check).
func NewHealthCheckerWithE2E(
	logger *slog.Logger,
	checkInterval time.Duration,
	timeout time.Duration,
	maxRetries int,
	e2eCheckURL string,
) *HealthChecker {
	hc := NewHealthChecker(logger, checkInterval, timeout, maxRetries)

	if e2eCheckURL != "" {
		hc.e2eCheckURL = e2eCheckURL
		// Pre-parse URL for SOCKS5 CONNECT
		parsedURL, err := url.Parse(e2eCheckURL)
		if err == nil {
			host := parsedURL.Hostname()
			port := parsedURL.Port()
			if port == "" {
				if parsedURL.Scheme == "https" {
					port = "443"
				} else {
					port = "80"
				}
			}
			hc.e2eCheckHost = net.JoinHostPort(host, port)

			// Build raw HTTP GET request
			path := parsedURL.Path
			if path == "" {
				path = "/"
			}
			hc.e2eRequestStr = fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: GoXRayHealthCheck/1.0\r\nConnection: close\r\n\r\n", path, host)

			logger.Debug("E2E health check configured",
				"url", e2eCheckURL,
				"socks_target", hc.e2eCheckHost,
				"request", hc.e2eRequestStr)
		} else {
			logger.Warn("Invalid E2E check URL, falling back to SOCKS-only check", "url", e2eCheckURL, "error", err)
			hc.e2eCheckURL = ""
		}
	}

	return hc
}

// Start begins health checking loop for SOCKS proxy
func (h *HealthChecker) Start(ctx context.Context, serverHost string, socksPort string, onUnhealthy func()) {
	h.logger.Info("Starting health checks", "server", serverHost, 
		"socks_proxy", "127.0.0.1:"+socksPort,
		"interval", h.checkInterval, "timeout", h.timeout, "max_retries", h.maxRetries,
		"e2e_check_url", h.e2eCheckURL)

	go h.healthCheckLoop(ctx, "127.0.0.1", socksPort, onUnhealthy)
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

// checkHealth performs a single health check
func (h *HealthChecker) checkHealth(host string, port string) error {
	startTime := time.Now()

	// Step 1: Basic SOCKS5 proxy connectivity check (always performed)
	if err := h.checkSOCKSProxy(host, port); err != nil {
		return fmt.Errorf("SOCKS check failed: %w", err)
	}

	socksDuration := time.Since(startTime)
	h.logger.Debug("Health check: SOCKS proxy check passed", "socks_ms", socksDuration.Milliseconds())

	// Step 2: E2E traffic check through SOCKS5 (only if URL configured)
	if h.e2eCheckURL != "" && h.e2eCheckHost != "" {
		e2eStart := time.Now()
		if err := h.checkTrafficThroughSOCKS(host, port, startTime); err != nil {
			return fmt.Errorf("E2E traffic check failed: %w", err)
		}
		e2eDuration := time.Since(e2eStart)
		totalDuration := time.Since(startTime)
		
		h.logger.Debug("Health check: E2E traffic check passed",
			"socks_ms", socksDuration.Milliseconds(),
			"e2e_ms", e2eDuration.Milliseconds(),
			"total_ms", totalDuration.Milliseconds())
	}

	return nil
}

// checkSOCKSProxy verifies the SOCKS5 proxy is responsive with a basic handshake
func (h *HealthChecker) checkSOCKSProxy(host string, port string) error {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	socksAddr := net.JoinHostPort(host, port)
	
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", socksAddr)
	if err != nil {
		return fmt.Errorf("SOCKS dial failed: %w", err)
	}
	defer conn.Close()

	// Set deadline for response
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}

	// Send SOCKS5 greeting: Version 5, 1 auth method (no auth)
	greeting := []byte{0x05, 0x01, 0x00}
	_, err = conn.Write(greeting)
	if err != nil {
		return fmt.Errorf("failed to write SOCKS greeting: %w", err)
	}

	// Read response
	buf := make([]byte, 2)
	_, err = conn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read SOCKS response: %w", err)
	}

	// Verify SOCKS5 response
	if buf[0] != 0x05 || buf[1] != 0x00 {
		return fmt.Errorf("invalid SOCKS response: version=%d, method=%d", buf[0], buf[1])
	}

	return nil
}

// checkTrafficThroughSOCKS performs end-to-end traffic check through the SOCKS5 proxy.
// It opens a new SOCKS5 connection, sends a CONNECT request to the target host,
// and performs an HTTP GET request to verify actual data flow.
func (h *HealthChecker) checkTrafficThroughSOCKS(host string, port string, overallStart time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	socksAddr := net.JoinHostPort(host, port)

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", socksAddr)
	if err != nil {
		return fmt.Errorf("E2E SOCKS dial failed: %w", err)
	}
	defer conn.Close()

	// Set overall deadline
	deadline := time.Now().Add(h.timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}

	// Step 1: SOCKS5 greeting
	greeting := []byte{0x05, 0x01, 0x00}
	if _, err := conn.Write(greeting); err != nil {
		return fmt.Errorf("E2E SOCKS greeting write: %w", err)
	}

	buf := make([]byte, 2)
	if _, err := conn.Read(buf); err != nil {
		return fmt.Errorf("E2E SOCKS greeting read: %w", err)
	}
	if buf[0] != 0x05 || buf[1] != 0x00 {
		return fmt.Errorf("E2E invalid SOCKS response: version=%d, method=%d", buf[0], buf[1])
	}

	// Step 2: SOCKS5 CONNECT to target host
	hostname, targetPort, err := net.SplitHostPort(h.e2eCheckHost)
	if err != nil {
		return fmt.Errorf("E2E invalid target: %w", err)
	}

	// Build SOCKS5 CONNECT request (IPv4 or domain name)
	var connectReq []byte
	ip := net.ParseIP(hostname)
	if ip != nil {
		// IPv4: VER=5, CMD=1, RSV=0, ATYP=1, DST.ADDR=4 bytes, DST.PORT=2 bytes
		ip4 := ip.To4()
		if ip4 != nil {
			connectReq = []byte{0x05, 0x01, 0x00, 0x01}
			connectReq = append(connectReq, ip4...)
		} else {
			// IPv6: VER=5, CMD=1, RSV=0, ATYP=4, DST.ADDR=16 bytes, DST.PORT=2 bytes
			ip16 := ip.To16()
			connectReq = []byte{0x05, 0x01, 0x00, 0x04}
			connectReq = append(connectReq, ip16...)
		}
	} else {
		// Domain name: VER=5, CMD=1, RSV=0, ATYP=3, LEN, DST.ADDR, DST.PORT
		connectReq = []byte{0x05, 0x01, 0x00, 0x03, byte(len(hostname))}
		connectReq = append(connectReq, []byte(hostname)...)
	}

	// Append port (big-endian)
	portInt := 0
	fmt.Sscanf(targetPort, "%d", &portInt)
	connectReq = append(connectReq, byte(portInt>>8), byte(portInt))

	if _, err := conn.Write(connectReq); err != nil {
		return fmt.Errorf("E2E SOCKS CONNECT write: %w", err)
	}

	// Read SOCKS5 CONNECT response (expect 10 bytes: VER, REP, RSV, ATYP, BND.ADDR, BND.PORT)
	resp := make([]byte, 10)
	if _, err := conn.Read(resp); err != nil {
		return fmt.Errorf("E2E SOCKS CONNECT read: %w", err)
	}

	// Check response: VER=5, REP=0 (success)
	if resp[0] != 0x05 {
		return fmt.Errorf("E2E SOCKS CONNECT bad version: %d", resp[0])
	}
	if resp[1] != 0x00 {
		repCodes := map[byte]string{
			0x01: "general SOCKS server failure",
			0x02: "connection not allowed by ruleset",
			0x03: "network unreachable",
			0x04: "host unreachable",
			0x05: "connection refused",
			0x06: "TTL expired",
			0x07: "command not supported",
			0x08: "address type not supported",
		}
		reason := repCodes[resp[1]]
		if reason == "" {
			reason = fmt.Sprintf("unknown error code 0x%02x", resp[1])
		}
		return fmt.Errorf("E2E SOCKS CONNECT rejected: %s", reason)
	}

	// Step 3: Send HTTP GET request through established SOCKS5 tunnel
	if _, err := conn.Write([]byte(h.e2eRequestStr)); err != nil {
		return fmt.Errorf("E2E HTTP request write: %w", err)
	}

	// Step 4: Read HTTP response (just check for valid HTTP response)
	reader := bufio.NewReaderSize(conn, 1024)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("E2E HTTP response read: %w", err)
	}

	response = strings.TrimSpace(response)

	// Check for valid HTTP status line (e.g., "HTTP/1.1 200 OK")
	if !strings.HasPrefix(response, "HTTP/") {
		return fmt.Errorf("E2E invalid HTTP response: %s", response)
	}

	// Check for HTTP error status (4xx, 5xx)
	if len(response) >= 12 {
		statusCode := response[9:12]
		if statusCode[0] == '4' || statusCode[0] == '5' {
			h.logger.Warn("E2E health check got HTTP error status", "status", statusCode, "response_line", response)
			// Allow 4xx/5xx as "pass" - the traffic path works, just the resource returned an error
			// Only fail if we got no response at all (handled above)
		}
	}

	h.logger.Debug("E2E traffic check passed", "response", response, "target", h.e2eCheckHost)

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

	status := map[string]interface{}{
		"is_healthy":            h.isHealthy,
		"consecutive_failures":  h.consecutiveFailures,
		"last_check":            h.lastCheck,
		"check_interval":        h.checkInterval,
		"max_retries":           h.maxRetries,
		"socks_only":            h.e2eCheckURL == "",
	}

	if h.e2eCheckURL != "" {
		status["e2e_check_url"] = h.e2eCheckURL
	}

	return status
}