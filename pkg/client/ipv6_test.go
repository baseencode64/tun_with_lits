package client

import (
	"log/slog"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDefaultRoutesToTUNIPv6 verifies that IPv6 routes are properly defined
func TestDefaultRoutesToTUNIPv6(t *testing.T) {
	require.NotNil(t, DefaultRoutesToTUNIPv6, "IPv6 routes should be defined")
	require.Len(t, DefaultRoutesToTUNIPv6, 2, "Should have 2 IPv6 routes (::/1 and 8000::/1)")

	// Verify first route covers lower half of IPv6 space
	require.Equal(t, "::/1", DefaultRoutesToTUNIPv6[0].String(), "First route should be ::/1")

	// Verify second route covers upper half of IPv6 space
	require.Equal(t, "8000::/1", DefaultRoutesToTUNIPv6[1].String(), "Second route should be 8000::/1")
}

// TestDefaultTUNAddressIPv6 verifies IPv6 TUN address configuration
func TestDefaultTUNAddressIPv6(t *testing.T) {
	require.NotNil(t, defaultTUNAddressIPv6, "IPv6 TUN address should be defined")
	require.NotNil(t, defaultTUNAddressIPv6.IP, "IPv6 IP should not be nil")

	// Verify it's a valid IPv6 address
	require.Equal(t, 16, len(defaultTUNAddressIPv6.IP), "IPv6 address should be 16 bytes")

	// Verify it's in ULA range (fd00::/8)
	require.True(t, defaultTUNAddressIPv6.IP[0] == 0xfd && defaultTUNAddressIPv6.IP[1] == 0x00,
		"IPv6 address should be in ULA range fd00::/8")

	// Verify prefix length is /64
	ones, bits := defaultTUNAddressIPv6.Mask.Size()
	require.Equal(t, 64, ones, "IPv6 mask should be /64")
	require.Equal(t, 128, bits, "IPv6 total bits should be 128")
}

// TestConfig_EnableIPv6 tests the EnableIPv6 configuration field
func TestConfig_EnableIPv6(t *testing.T) {
	tests := []struct {
		name     string
		initial  bool
		override bool
		expected bool
	}{
		{
			name:     "default false to true",
			initial:  false,
			override: true,
			expected: true,
		},
		{
			name:     "true to false",
			initial:  true,
			override: false,
			expected: false,
		},
		{
			name:     "true stays true",
			initial:  true,
			override: true,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				EnableIPv6: tt.initial,
			}

			newCfg := &Config{
				EnableIPv6: tt.override,
			}

			cfg.apply(newCfg)
			require.Equal(t, tt.expected, cfg.EnableIPv6)
		})
	}
}

// TestClient_ConfigWithIPv6 tests client initialization with IPv6 enabled
func TestClient_ConfigWithIPv6(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := Config{
		Logger:           logger,
		TLSAllowInsecure: false,
		EnableIPv6:       true,
	}

	cl, err := NewClientWithOpts(cfg)
	require.NoError(t, err, "Should create client with IPv6 enabled")
	require.NotNil(t, cl, "Client should not be nil")
	require.True(t, cl.cfg.EnableIPv6, "IPv6 should be enabled in config")
}

// TestClient_ConfigWithoutIPv6 tests client initialization with IPv6 disabled (default)
func TestClient_ConfigWithoutIPv6(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := Config{
		Logger:           logger,
		TLSAllowInsecure: false,
		EnableIPv6:       false, // Explicitly disabled
	}

	cl, err := NewClientWithOpts(cfg)
	require.NoError(t, err, "Should create client with IPv6 disabled")
	require.NotNil(t, cl, "Client should not be nil")
	require.False(t, cl.cfg.EnableIPv6, "IPv6 should be disabled in config")
}

// TestClient_DefaultConfigIPv6 tests that IPv6 is disabled by default
func TestClient_DefaultConfigIPv6(t *testing.T) {
	cl, err := NewClient()
	require.NoError(t, err, "Should create default client")
	require.NotNil(t, cl, "Client should not be nil")
	require.False(t, cl.cfg.EnableIPv6, "IPv6 should be disabled by default")
}

// TestIPv6RoutesCoverage verifies that IPv6 routes cover entire address space
func TestIPv6RoutesCoverage(t *testing.T) {
	// Parse test addresses to verify they fall within our routes
	testAddresses := []string{
		"::1",                    // loopback
		"2001:db8::1",           // documentation
		"fe80::1",               // link-local
		"fd00::1",               // ULA
		"2001:4860:4860::8888",  // Google DNS
		"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", // max IPv6
	}

	for _, addrStr := range testAddresses {
		t.Run(addrStr, func(t *testing.T) {
			addr := net.ParseIP(addrStr)
			require.NotNil(t, addr, "Should parse valid IPv6 address: %s", addrStr)

			// Check if address falls within one of our routes
			covered := false
			for _, routeAddr := range DefaultRoutesToTUNIPv6 {
				// Convert route.Addr to net.IPNet for comparison
				_, ipNet, err := net.ParseCIDR(routeAddr.String())
				if err != nil {
					continue
				}
				if ipNet.Contains(addr) {
					covered = true
					break
				}
			}
			require.True(t, covered, "IPv6 address %s should be covered by routes", addrStr)
		})
	}
}

// TestIPv4AndIPv6RoutesIndependence verifies that IPv4 and IPv6 routes are independent
func TestIPv4AndIPv6RoutesIndependence(t *testing.T) {
	require.NotNil(t, DefaultRoutesToTUN, "IPv4 routes should be defined")
	require.NotNil(t, DefaultRoutesToTUNIPv6, "IPv6 routes should be defined")

	// Verify different number of routes (both have 2, but this is intentional)
	require.Len(t, DefaultRoutesToTUN, 2, "IPv4 should have 2 routes")
	require.Len(t, DefaultRoutesToTUNIPv6, 2, "IPv6 should have 2 routes")

	// Verify no overlap in address families
	for _, ipv4Route := range DefaultRoutesToTUN {
		for _, ipv6Route := range DefaultRoutesToTUNIPv6 {
			require.NotEqual(t, ipv4Route.String(), ipv6Route.String(),
				"IPv4 and IPv6 routes should not overlap")
		}
	}
}

// BenchmarkIPv6RouteLookup benchmarks IPv6 route matching performance
func BenchmarkIPv6RouteLookup(b *testing.B) {
	testAddr := net.ParseIP("2001:db8::1")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, routeAddr := range DefaultRoutesToTUNIPv6 {
			_, ipNet, _ := net.ParseCIDR(routeAddr.String())
			if ipNet.Contains(testAddr) {
				break
			}
		}
	}
}
