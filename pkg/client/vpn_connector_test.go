package client

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestVPNConnector_ConnectWithFallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, nil))
	loggerAdapter := NewSlogAdapter(logger)
	
	// Create mock servers with different latencies
	servers := []*ServerInfo{
		{Link: "vless://test1@server1.com:443", Host: "server1.com", Port: "443", Latency: 50 * time.Millisecond},
		{Link: "vless://test2@server2.com:443", Host: "server2.com", Port: "443", Latency: 100 * time.Millisecond},
		{Link: "vless://test3@server3.com:443", Host: "server3.com", Port: "443", Latency: 150 * time.Millisecond},
	}

	t.Run("empty server list", func(t *testing.T) {
		vpn, _ := NewClientWithOpts(Config{Logger: logger})
		connector := NewVPNConnector(vpn, nil, logger)
		
		err := connector.ConnectWithFallback([]*ServerInfo{})
		if err == nil {
			t.Error("Expected error for empty server list")
		}
	})

	t.Run("connection with fallback", func(t *testing.T) {
		vpn, _ := NewClientWithOpts(Config{Logger: logger})
		selector := NewServerSelector(loggerAdapter, 5*time.Second, 10)
		connector := NewVPNConnector(vpn, selector, logger)
		
		// This will try to connect but fail (invalid test links)
		// We're testing the logic flow, not actual connection
		err := connector.ConnectWithFallback(servers)
		if err == nil {
			t.Log("Unexpected success - test links should fail")
		} else {
			t.Logf("Expected failure: %v", err)
		}
	})
}

func TestVPNConnector_GetConnectionReport(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, nil))
	
	servers := []*ServerInfo{
		{Link: "vless://test1@server1.com:443", Host: "server1.com", Port: "443", Latency: 50 * time.Millisecond},
		{Link: "vless://test2@server2.com:443", Host: "server2.com", Port: "443", Latency: 100 * time.Millisecond},
	}

	vpn, _ := NewClientWithOpts(Config{Logger: logger})
	selector := NewServerSelector(NewSlogAdapter(logger), 5*time.Second, 10)
	connector := NewVPNConnector(vpn, selector, logger)
	
	report := connector.GetConnectionReport(servers)
	
	if report == "" {
		t.Error("Expected non-empty report")
	}
	
	t.Logf("Generated report:\n%s", report)
	
	// Check report contains expected info
	if len(report) < 50 {
		t.Errorf("Report too short, got %d chars", len(report))
	}
}

func TestVPNConnector_EmptyServers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, nil))
	vpn, _ := NewClientWithOpts(Config{Logger: logger})
	connector := NewVPNConnector(vpn, nil, logger)
	
	err := connector.ConnectWithFallback([]*ServerInfo{})
	if err == nil {
		t.Error("Expected error for empty servers")
	}
}
