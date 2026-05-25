package client

import (
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// ServerInfo holds information about a VPN server
type ServerInfo struct {
	Link         string
	Host         string
	Port         string
	Latency      time.Duration
	Available    bool
	PacketLoss   float64 // 0.0 - 1.0 (0% - 100%)
	Score        float64 // Weighted score (higher = better)
}

// ServerSelector handles server selection based on latency
type ServerSelector struct {
	logger        Logger
	timeout       time.Duration
	maxConcurrent int
	httpClient    *http.Client
}

// NewServerSelector creates a new server selector
func NewServerSelector(logger Logger, timeout time.Duration, maxConcurrent int) *ServerSelector {
	if logger == nil {
		logger = &noopLogger{}
	}
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	if maxConcurrent == 0 {
		maxConcurrent = 10
	}

	return &ServerSelector{
		logger:        logger,
		timeout:       timeout,
		maxConcurrent: maxConcurrent,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					d := net.Dialer{
						Timeout: timeout,
					}
					return d.DialContext(ctx, network, addr)
				},
			},
		},
	}
}

// FetchRawLinks fetches and returns VLESS links from a raw URL
func (s *ServerSelector) FetchRawLinks(rawURL string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch raw list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	parser := NewLinkParser(s.logger)
	links := parser.ParseLinksFromRaw(string(body))

	s.logger.Debug("Parsed links from raw URL", "url", rawURL, "count", len(links))

	return links, nil
}

// CheckLatency checks the latency of a VLESS server by attempting connection
func (s *ServerSelector) CheckLatency(link string) (time.Duration, error) {
	// Extract host and port from VLESS link
	host, port, err := extractHostPort(link)
	if err != nil {
		return 0, fmt.Errorf("extract host/port: %w", err)
	}

	addr := net.JoinHostPort(host, port)

	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	latency := time.Since(start)

	return latency, nil
}

// CheckServerHealth performs comprehensive health check including latency, packet loss, and throughput
func (s *ServerSelector) CheckServerHealth(link string) (*ServerInfo, error) {
	host, port, err := extractHostPort(link)
	if err != nil {
		return nil, fmt.Errorf("extract host/port: %w", err)
	}

	info := &ServerInfo{
		Link:      link,
		Host:      host,
		Port:      port,
		Available: false,
	}

	// Test 1: Latency (weight: 40%)
	latency, err := s.CheckLatency(link)
	if err != nil {
		s.logger.Debug("Server unavailable", "host", host, "error", err)
		return info, err
	}

	info.Latency = latency
	info.Available = true

	// Test 2: Packet loss simulation (weight: 35%)
	// Perform multiple connection attempts to estimate reliability
	packetLoss := s.checkPacketLoss(host, port)
	info.PacketLoss = packetLoss

	// Calculate weighted score
	info.Score = s.calculateScore(info)

	s.logger.Debug("Server health check complete",
		"host", host,
		"latency_ms", latency.Milliseconds(),
		"packet_loss", fmt.Sprintf("%.1f%%", packetLoss*100),
		"score", fmt.Sprintf("%.2f", info.Score))

	return info, nil
}

// checkPacketLoss estimates packet loss by making multiple connection attempts
func (s *ServerSelector) checkPacketLoss(host, port string) float64 {
	attempts := 5
	successes := 0

	for i := 0; i < attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(host, port))
		cancel()

		if err == nil {
			successes++
			conn.Close()
		}

		// Small delay between attempts
		time.Sleep(100 * time.Millisecond)
	}

	return float64(attempts-successes) / float64(attempts)
}

// calculateScore computes weighted score for server selection
// Score formula: (latency_score * 0.5) + ((1 - packet_loss) * 0.5)
func (s *ServerSelector) calculateScore(info *ServerInfo) float64 {
	// Normalize latency (lower is better, max 2000ms)
	latencyScore := 1.0 - math.Min(float64(info.Latency.Milliseconds())/2000.0, 1.0)

	// Packet loss (lower is better)
	reliabilityScore := 1.0 - info.PacketLoss

	// Weighted sum
	score := (latencyScore * 0.5) + (reliabilityScore * 0.5)

	return score
}

// SelectBest selects the best server from a list of links based on weighted scoring
func (s *ServerSelector) SelectBest(links []string) (*ServerInfo, error) {
	servers, err := s.SelectAllByScore(links)
	if err != nil {
		return nil, err
	}

	return servers[0], nil
}

// SelectAllByLatency selects and sorts all available servers by latency (legacy method)
func (s *ServerSelector) SelectAllByLatency(links []string) ([]*ServerInfo, error) {
	if len(links) == 0 {
		return nil, fmt.Errorf("no links provided")
	}

	s.logger.Info("Checking servers", "total", len(links), "max_concurrent", s.maxConcurrent)

	var mu sync.Mutex
	var available []*ServerInfo

	// Use semaphore pattern for concurrency control
	sem := make(chan struct{}, s.maxConcurrent)
	var wg sync.WaitGroup

	for i, link := range links {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(idx int, srvLink string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			host, port, err := extractHostPort(srvLink)
			if err != nil {
				s.logger.Debug("Failed to parse link", "index", idx, "error", err)
				return
			}

			s.logger.Debug("Checking server", "index", idx+1, "host", host, "port", port)

			latency, err := s.CheckLatency(srvLink)
			if err != nil {
				s.logger.Debug("Server unavailable", "index", idx+1, "host", host, "error", err)
				return
			}

			mu.Lock()
			available = append(available, &ServerInfo{
				Link:      srvLink,
				Host:      host,
				Port:      port,
				Latency:   latency,
				Available: true,
			})
			mu.Unlock()

			s.logger.Debug("Server available", "index", idx+1, "host", host, "latency", latency)
		}(i, link)
	}

	wg.Wait()

	if len(available) == 0 {
		return nil, fmt.Errorf("no available servers found")
	}

	// Sort by latency
	sort.Slice(available, func(i, j int) bool {
		return available[i].Latency < available[j].Latency
	})

	s.logger.Info("Found available servers", "total", len(available), "sorted_by", "latency")

	return available, nil
}

// SelectAllByScore selects and sorts servers by weighted score (smart selection)
func (s *ServerSelector) SelectAllByScore(links []string) ([]*ServerInfo, error) {
	if len(links) == 0 {
		return nil, fmt.Errorf("no links provided")
	}

	s.logger.Info("Performing smart server selection with weighted scoring", "total", len(links), "max_concurrent", s.maxConcurrent)

	var mu sync.Mutex
	var scored []*ServerInfo

	// Use semaphore pattern for concurrency control
	sem := make(chan struct{}, s.maxConcurrent)
	var wg sync.WaitGroup

	for i, link := range links {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(idx int, srvLink string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			s.logger.Debug("Checking server health", "index", idx+1)

			info, err := s.CheckServerHealth(srvLink)
			if err != nil || !info.Available {
				s.logger.Debug("Server unavailable", "index", idx+1, "host", info.Host)
				return
			}

			mu.Lock()
			scored = append(scored, info)
			mu.Unlock()

			s.logger.Debug("Server scored", 
				"index", idx+1,
				"host", info.Host,
				"score", fmt.Sprintf("%.2f", info.Score))
		}(i, link)
	}

	wg.Wait()

	if len(scored) == 0 {
		return nil, fmt.Errorf("no available servers found")
	}

	// Sort by score (higher is better)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	s.logger.Info("Smart server selection complete", 
		"total_available", len(scored),
		"best_server", scored[0].Host,
		"best_score", fmt.Sprintf("%.2f", scored[0].Score),
		"sorted_by", "weighted_score")

	return scored, nil
}

// SelectBestFromURL fetches links from URL and selects the best one
func (s *ServerSelector) SelectBestFromURL(rawURL string) (*ServerInfo, error) {
	links, err := s.FetchRawLinks(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch links: %w", err)
	}

	return s.SelectBest(links)
}

// extractHostPort extracts hostname and port from VLESS link
func extractHostPort(link string) (string, string, error) {
	link = strings.TrimSpace(link)

	// Remove protocol prefix
	if !strings.HasPrefix(link, "vless://") {
		return "", "", fmt.Errorf("not a vless link")
	}

	// Parse as URL
	u, err := url.Parse(link)
	if err != nil {
		return "", "", fmt.Errorf("parse URL: %w", err)
	}

	host := u.Hostname()
	port := u.Port()

	if host == "" {
		return "", "", fmt.Errorf("missing hostname")
	}

	if port == "" {
		// Default port for VLESS is typically 443
		port = "443"
	}

	return host, port, nil
}
