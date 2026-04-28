package client

import (
	"context"
	"fmt"
	"io"
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
	Link      string
	Host      string
	Port      string
	Latency   time.Duration
	Available bool
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

// SelectBest selects the best server from a list of links based on latency
func (s *ServerSelector) SelectBest(links []string) (*ServerInfo, error) {
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

	best := available[0]
	s.logger.Info("Selected best server", "host", best.Host, "port", best.Port,
		"latency", best.Latency, "rank", fmt.Sprintf("1/%d", len(available)))

	return best, nil
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
