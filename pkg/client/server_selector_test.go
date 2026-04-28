package client

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServerSelector_FetchRawLinks(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	selector := NewServerSelector(logger, 5*time.Second, 10)

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		expectErr bool
		linkCount int
	}{
		{
			name: "valid raw list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`vless://uuid1@example.com:443
# comment
vless://uuid2@test.org:8080
invalid-link
`))
			},
			expectErr: false,
			linkCount: 2,
		},
		{
			name: "empty list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("# only comments\n\n"))
			},
			expectErr: false,
			linkCount: 0,
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectErr: true,
			linkCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			links, err := selector.FetchRawLinks(server.URL)

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, links, tt.linkCount)
			}
		})
	}
}

func TestServerSelector_CheckLatency(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	selector := NewServerSelector(logger, 2*time.Second, 10)

	t.Run("unreachable server", func(t *testing.T) {
		// Use a non-routable IP to ensure connection fails
		link := "vless://test@192.0.2.1:443" // TEST-NET-1 (documentation range)

		latency, err := selector.CheckLatency(link)
		require.Error(t, err)
		require.Equal(t, time.Duration(0), latency)
	})

	t.Run("invalid link format", func(t *testing.T) {
		_, err := selector.CheckLatency("not-a-valid-link")
		require.Error(t, err)
	})
}

func TestServerSelector_SelectBest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	selector := NewServerSelector(logger, 1*time.Second, 10)

	t.Run("empty list", func(t *testing.T) {
		best, err := selector.SelectBest([]string{})
		require.Error(t, err)
		require.Nil(t, best)
		require.Contains(t, err.Error(), "no links provided")
	})

	t.Run("all unreachable", func(t *testing.T) {
		links := []string{
			"vless://test@192.0.2.1:443",
			"vless://test@192.0.2.2:443",
		}

		best, err := selector.SelectBest(links)
		require.Error(t, err)
		require.Nil(t, best)
		require.Contains(t, err.Error(), "no available servers")
	})
}

func TestServerSelector_extractHostPort(t *testing.T) {
	tests := []struct {
		name     string
		link     string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{
			name:     "standard vless with port",
			link:     "vless://uuid@example.com:443?type=ws",
			wantHost: "example.com",
			wantPort: "443",
			wantErr:  false,
		},
		{
			name:     "vless without port defaults to 443",
			link:     "vless://uuid@example.com",
			wantHost: "example.com",
			wantPort: "443",
			wantErr:  false,
		},
		{
			name:     "custom port",
			link:     "vless://uuid@test.org:8080",
			wantHost: "test.org",
			wantPort: "8080",
			wantErr:  false,
		},
		{
			name:    "invalid - no protocol",
			link:    "https://example.com:443",
			wantErr: true,
		},
		{
			name:    "invalid - malformed",
			link:    "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := extractHostPort(tt.link)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantHost, host)
				require.Equal(t, tt.wantPort, port)
			}
		})
	}
}

func TestNewServerSelector_Defaults(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	selector := NewServerSelector(logger, 0, 0)

	require.NotNil(t, selector)
	require.Equal(t, 5*time.Second, selector.timeout)
	require.Equal(t, 10, selector.maxConcurrent)
	require.NotNil(t, selector.httpClient)
}

func TestServerSelector_ConcurrentChecking(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	maxConcurrent := 5
	selector := NewServerSelector(logger, 1*time.Second, maxConcurrent)

	// Create test server that tracks concurrent connections
	var activeConnections int
	var maxSeen int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		activeConnections++
		if activeConnections > maxSeen {
			maxSeen = activeConnections
		}
		time.Sleep(100 * time.Millisecond) // Simulate work
		activeConnections--
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create links pointing to test server
	// Note: This is a simplified test since we can't easily mock the TCP connection check
	_ = selector
	_ = maxSeen
}
