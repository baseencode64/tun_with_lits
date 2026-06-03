package client

import (
	"log/slog"
	"net"
	"os"
	"testing"
)

func TestNewSplitTunnelRouter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name    string
		config  SplitTunnelConfig
		wantErr bool
	}{
		{
			name: "valid exclude mode",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeExclude,
				ExcludeCIDRs: []string{"192.168.0.0/16"},
			},
			wantErr: false,
		},
		{
			name: "invalid config",
			config: SplitTunnelConfig{
				Enabled: true,
				Mode:    "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid CIDR",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeExclude,
				ExcludeCIDRs: []string{"invalid-cidr"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, err := NewSplitTunnelRouter(logger, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSplitTunnelRouter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && router == nil {
				t.Error("NewSplitTunnelRouter() returned nil router without error")
			}
		})
	}
}

func TestSplitTunnelRouter_ShouldRouteThroughVPN(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name    string
		config  SplitTunnelConfig
		destIP  string
		wantVPN bool
	}{
		// Exclude mode tests
		{
			name: "exclude mode - local network excluded",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeExclude,
				ExcludeCIDRs: []string{"192.168.0.0/16"},
			},
			destIP:  "192.168.1.100",
			wantVPN: false,
		},
		{
			name: "exclude mode - public internet through VPN",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeExclude,
				ExcludeCIDRs: []string{"192.168.0.0/16"},
			},
			destIP:  "8.8.8.8",
			wantVPN: true,
		},
		{
			name: "exclude mode - multiple CIDRs",
			config: SplitTunnelConfig{
				Enabled: true,
				Mode:    SplitTunnelModeExclude,
				ExcludeCIDRs: []string{
					"192.168.0.0/16",
					"10.0.0.0/8",
				},
			},
			destIP:  "10.5.10.50",
			wantVPN: false,
		},
		// Include mode tests
		{
			name: "include mode - included IP through VPN",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeInclude,
				IncludeCIDRs: []string{"8.8.8.8/32"},
			},
			destIP:  "8.8.8.8",
			wantVPN: true,
		},
		{
			name: "include mode - not included goes direct",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeInclude,
				IncludeCIDRs: []string{"8.8.8.8/32"},
			},
			destIP:  "1.1.1.1",
			wantVPN: false,
		},
		// Disabled mode tests
		{
			name: "disabled mode - all through VPN",
			config: SplitTunnelConfig{
				Enabled: false,
			},
			destIP:  "192.168.1.1",
			wantVPN: true,
		},
		// IPv6 tests
		{
			name: "exclude mode - IPv6 excluded",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeExclude,
				ExcludeCIDRs: []string{"fd00::/8"},
			},
			destIP:  "fd00:dead:beef::1",
			wantVPN: false,
		},
		{
			name: "exclude mode - IPv6 not excluded",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeExclude,
				ExcludeCIDRs: []string{"fd00::/8"},
			},
			destIP:  "2001:4860:4860::8888",
			wantVPN: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, err := NewSplitTunnelRouter(logger, tt.config)
			if err != nil {
				t.Fatalf("NewSplitTunnelRouter() error = %v", err)
			}

			ip := net.ParseIP(tt.destIP)
			if ip == nil {
				t.Fatalf("Invalid test IP: %s", tt.destIP)
			}

			got := router.ShouldRouteThroughVPN(ip)
			if got != tt.wantVPN {
				t.Errorf("ShouldRouteThroughVPN(%s) = %v, want %v", tt.destIP, got, tt.wantVPN)
			}
		})
	}
}

func TestSplitTunnelRouter_GetExcludedRoutes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name       string
		config     SplitTunnelConfig
		wantRoutes int
	}{
		{
			name: "exclude mode returns routes",
			config: SplitTunnelConfig{
				Enabled: true,
				Mode:    SplitTunnelModeExclude,
				ExcludeCIDRs: []string{
					"192.168.0.0/16",
					"10.0.0.0/8",
				},
			},
			wantRoutes: 2,
		},
		{
			name: "include mode returns nil",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeInclude,
				IncludeCIDRs: []string{"8.8.8.8/32"},
			},
			wantRoutes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, err := NewSplitTunnelRouter(logger, tt.config)
			if err != nil {
				t.Fatalf("NewSplitTunnelRouter() error = %v", err)
			}

			routes := router.GetExcludedRoutes()
			if len(routes) != tt.wantRoutes {
				t.Errorf("GetExcludedRoutes() returned %d routes, want %d", len(routes), tt.wantRoutes)
			}
		})
	}
}

func TestSplitTunnelRouter_GetIncludedRoutes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name       string
		config     SplitTunnelConfig
		wantRoutes int
	}{
		{
			name: "include mode returns routes",
			config: SplitTunnelConfig{
				Enabled: true,
				Mode:    SplitTunnelModeInclude,
				IncludeCIDRs: []string{
					"8.8.8.8/32",
					"1.1.1.1/32",
				},
			},
			wantRoutes: 2,
		},
		{
			name: "exclude mode returns nil",
			config: SplitTunnelConfig{
				Enabled:      true,
				Mode:         SplitTunnelModeExclude,
				ExcludeCIDRs: []string{"192.168.0.0/16"},
			},
			wantRoutes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, err := NewSplitTunnelRouter(logger, tt.config)
			if err != nil {
				t.Fatalf("NewSplitTunnelRouter() error = %v", err)
			}

			routes := router.GetIncludedRoutes()
			if len(routes) != tt.wantRoutes {
				t.Errorf("GetIncludedRoutes() returned %d routes, want %d", len(routes), tt.wantRoutes)
			}
		})
	}
}

func TestSplitTunnelRouter_GetStats(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	config := SplitTunnelConfig{
		Enabled: true,
		Mode:    SplitTunnelModeExclude,
		ExcludeCIDRs: []string{
			"192.168.0.0/16",
			"10.0.0.0/8",
			"172.16.0.0/12",
		},
	}

	router, err := NewSplitTunnelRouter(logger, config)
	if err != nil {
		t.Fatalf("NewSplitTunnelRouter() error = %v", err)
	}

	stats := router.GetStats()

	if stats["enabled"] != true {
		t.Errorf("GetStats() enabled = %v, want true", stats["enabled"])
	}

	if stats["mode"] != SplitTunnelModeExclude {
		t.Errorf("GetStats() mode = %v, want %v", stats["mode"], SplitTunnelModeExclude)
	}

	if stats["exclude_count"] != 3 {
		t.Errorf("GetStats() exclude_count = %v, want 3", stats["exclude_count"])
	}

	if stats["include_count"] != 0 {
		t.Errorf("GetStats() include_count = %v, want 0", stats["include_count"])
	}
}
