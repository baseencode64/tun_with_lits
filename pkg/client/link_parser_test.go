package client

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinkParser_ParseLinksFromRaw(t *testing.T) {
	parser := NewLinkParser(nil)

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

			// Verify all links are valid VLESS format
			for _, link := range links {
				require.True(t, strings.HasPrefix(link, "vless://"))
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

func TestLinkParser_ValidateLink(t *testing.T) {
	parser := NewLinkParser(nil)

	err := parser.ValidateLink("vless://test@example.com:443")
	require.NoError(t, err)

	err = parser.ValidateLink("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid VLESS link")
}
