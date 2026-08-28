package client

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// buildVMessLink creates a VMess link from a config map for testing
func buildVMessLink(t *testing.T, cfg map[string]string) string {
	t.Helper()
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	return "vmess://" + base64.StdEncoding.EncodeToString(data)
}

func TestLinkParser_ParseLinksFromRaw(t *testing.T) {
	parser := NewLinkParser(nil)

	validVMess := buildVMessLink(t, map[string]string{
		"v": "2", "ps": "test", "add": "example.com", "port": "443",
		"id": "uuid", "aid": "0", "scy": "auto", "net": "ws",
	})

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name: "valid vless links",
			input: `vless://uuid1@example.com:443?type=ws&security=tls
# This is a comment
vless://uuid2@example.org:8080?type=tcp

invalid-link
vless://uuid3@test.net:443`,
			expected: 3,
		},
		{
			name: "valid vmess links",
			input: validVMess + `
# comment
invalid-vmess
` + buildVMessLink(t, map[string]string{
				"v": "2", "add": "test.org", "port": "8080",
			}),
			expected: 2,
		},
		{
			name: "mixed vless and vmess links",
			input: `vless://uuid1@example.com:443?type=ws
` + validVMess + `
vless://uuid2@test.net:443`,
			expected: 3,
		},
		{
			name: "empty lines and comments only",
			input: `# Comment 1
# Comment 2

`,
			expected: 0,
		},
		{
			name:     "no valid links",
			input:    "invalid1\ninvalid2\nnot-a-link",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links := parser.ParseLinksFromRaw(tt.input)
			require.Len(t, links, tt.expected)

			// Verify all links are valid VLESS or VMess format
			for _, link := range links {
				require.True(t, strings.HasPrefix(link, "vless://") || strings.HasPrefix(link, "vmess://"),
					"link should be vless or vmess: %s", link)
			}
		})
	}
}

func TestLinkParser_isValidVLESSLink(t *testing.T) {
	parser := NewLinkParser(nil)

	tests := []struct {
		name    string
		link    string
		isValid bool
	}{
		{
			name:    "valid vless with port",
			link:    "vless://abc123@example.com:443?type=ws",
			isValid: true,
		},
		{
			name:    "valid vless without params",
			link:    "vless://abc123@example.com:8080",
			isValid: true,
		},
		{
			name:    "missing port",
			link:    "vless://abc123@example.com",
			isValid: false,
		},
		{
			name:    "wrong protocol",
			link:    "vmess://abc123@example.com:443",
			isValid: false,
		},
		{
			name:    "not a URL",
			link:    "just-some-text",
			isValid: false,
		},
		{
			name:    "empty string",
			link:    "",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.isValidVLESSLink(tt.link)
			require.Equal(t, tt.isValid, result)
		})
	}
}

func TestLinkParser_isValidVMessLink(t *testing.T) {
	parser := NewLinkParser(nil)

	validLink := buildVMessLink(t, map[string]string{
		"v": "2", "ps": "test", "add": "example.com", "port": "443",
		"id": "uuid", "aid": "0", "scy": "auto", "net": "ws",
	})

	tests := []struct {
		name    string
		link    string
		isValid bool
	}{
		{
			name:    "valid vmess link",
			link:    validLink,
			isValid: true,
		},
		{
			name:    "valid vmess minimal fields",
			link:    buildVMessLink(t, map[string]string{"add": "example.com", "port": "443"}),
			isValid: true,
		},
		{
			name:    "missing address",
			link:    buildVMessLink(t, map[string]string{"port": "443"}),
			isValid: false,
		},
		{
			name:    "missing port",
			link:    buildVMessLink(t, map[string]string{"add": "example.com"}),
			isValid: false,
		},
		{
			name:    "invalid base64",
			link:    "vmess://not-valid-base64!!!",
			isValid: false,
		},
		{
			name:    "valid base64 but invalid json",
			link:    "vmess://" + base64.StdEncoding.EncodeToString([]byte("not json")),
			isValid: false,
		},
		{
			name:    "empty payload",
			link:    "vmess://",
			isValid: false,
		},
		{
			name:    "wrong protocol prefix",
			link:    "vless://abc@example.com:443",
			isValid: false,
		},
		{
			name:    "empty string",
			link:    "",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.isValidVMessLink(tt.link)
			require.Equal(t, tt.isValid, result)
		})
	}
}

func TestLinkParser_ParseVMessLink(t *testing.T) {
	parser := NewLinkParser(nil)

	t.Run("valid vmess link", func(t *testing.T) {
		link := buildVMessLink(t, map[string]string{
			"v": "2", "ps": "my-server", "add": "example.com", "port": "443",
			"id": "test-uuid", "aid": "0", "scy": "auto", "net": "ws",
			"type": "none", "host": "example.com", "path": "/ws", "tls": "tls",
		})

		cfg, err := parser.ParseVMessLink(link)
		require.NoError(t, err)
		require.Equal(t, "example.com", cfg.Add)
		require.Equal(t, "443", cfg.Port)
		require.Equal(t, "my-server", cfg.PS)
		require.Equal(t, "test-uuid", cfg.ID)
		require.Equal(t, "ws", cfg.Net)
		require.Equal(t, "tls", cfg.TLS)
	})

	t.Run("not a vmess link", func(t *testing.T) {
		_, err := parser.ParseVMessLink("vless://test@example.com:443")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not a vmess link")
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := parser.ParseVMessLink("vmess://invalid!!!")
		require.Error(t, err)
	})

	t.Run("missing address", func(t *testing.T) {
		link := buildVMessLink(t, map[string]string{"port": "443"})
		_, err := parser.ParseVMessLink(link)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing address")
	})

	t.Run("missing port", func(t *testing.T) {
		link := buildVMessLink(t, map[string]string{"add": "example.com"})
		_, err := parser.ParseVMessLink(link)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing port")
	})
}

func TestLinkParser_ValidateLink(t *testing.T) {
	parser := NewLinkParser(nil)

	// Valid VLESS
	err := parser.ValidateLink("vless://test@example.com:443")
	require.NoError(t, err)

	// Valid VMess
	validVMess := buildVMessLink(t, map[string]string{
		"add": "example.com", "port": "443",
	})
	err = parser.ValidateLink(validVMess)
	require.NoError(t, err)

	// Invalid link
	err = parser.ValidateLink("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid link format")
}