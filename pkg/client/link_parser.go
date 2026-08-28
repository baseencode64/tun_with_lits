package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// LinkParser handles parsing and validation of VLESS and VMess links
type LinkParser struct {
	logger Logger
}

// VMessConfig represents the JSON structure inside a VMess link
type VMessConfig struct {
	V    string `json:"v"`    // Protocol version
	PS   string `json:"ps"`   // Remark/alias
	Add  string `json:"add"`  // Server address (IP or domain)
	Port string `json:"port"` // Server port
	ID   string `json:"id"`   // User ID (UUID)
	Aid  string `json:"aid"`  // Alter ID
	Scy  string `json:"scy"`  // Security/cipher method
	Net  string `json:"net"`  // Network type (tcp, ws, grpc, etc.)
	Type string `json:"type"` // Header type (none, http, etc.)
	Host string `json:"host"` // Request host
	Path string `json:"path"` // Request path
	TLS  string `json:"tls"`  // TLS enabled ("tls" or "")
	SNI  string `json:"sni"`  // TLS SNI
	Alpn string `json:"alpn"` // ALPN protocol
}

// NewLinkParser creates a new link parser instance
func NewLinkParser(logger Logger) *LinkParser {
	if logger == nil {
		logger = &noopLogger{}
	}
	return &LinkParser{logger: logger}
}

// ParseLinksFromRaw parses raw text and extracts VLESS and VMess links
func (p *LinkParser) ParseLinksFromRaw(rawText string) []string {
	var links []string

	lines := strings.Split(rawText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if p.isValidVLESSLink(line) || p.isValidVMessLink(line) {
			links = append(links, line)
		} else {
			p.logger.Debug("Skipping invalid link", "link", line)
		}
	}

	return links
}

// isValidVLESSLink checks if a string is a valid VLESS link
func (p *LinkParser) isValidVLESSLink(link string) bool {
	if !strings.HasPrefix(link, "vless://") {
		return false
	}

	// Try to parse as URL to validate structure
	u, err := url.Parse(link)
	if err != nil {
		return false
	}

	// Basic validation: should have host and port
	if u.Hostname() == "" || u.Port() == "" {
		return false
	}

	return true
}

// isValidVMessLink checks if a string is a valid VMess link
// VMess link format: vmess://base64(JSON)
func (p *LinkParser) isValidVMessLink(link string) bool {
	if !strings.HasPrefix(link, "vmess://") {
		return false
	}

	// Extract base64 payload
	payload := strings.TrimPrefix(link, "vmess://")
	payload = strings.TrimSpace(payload)

	if payload == "" {
		return false
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		// Try URL-safe base64 (some providers use it)
		decoded, err = base64.URLEncoding.DecodeString(payload)
		if err != nil {
			// Try with padding fix
			decoded, err = base64.RawStdEncoding.DecodeString(payload)
			if err != nil {
				return false
			}
		}
	}

	// Parse JSON
	var cfg VMessConfig
	if err := json.Unmarshal(decoded, &cfg); err != nil {
		return false
	}

	// Basic validation: must have address and port
	if cfg.Add == "" {
		return false
	}

	// Port is required
	if cfg.Port == "" {
		return false
	}

	return true
}

// ParseVMessLink parses a VMess link and returns the VMessConfig
func (p *LinkParser) ParseVMessLink(link string) (*VMessConfig, error) {
	link = strings.TrimSpace(link)

	if !strings.HasPrefix(link, "vmess://") {
		return nil, fmt.Errorf("not a vmess link")
	}

	payload := strings.TrimPrefix(link, "vmess://")
	payload = strings.TrimSpace(payload)

	// Decode base64 with fallbacks
	var decoded []byte
	var err error

	decoded, err = base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(payload)
		if err != nil {
			decoded, err = base64.RawStdEncoding.DecodeString(payload)
			if err != nil {
				return nil, fmt.Errorf("decode base64: %w", err)
			}
		}
	}

	// Parse JSON
	var cfg VMessConfig
	if err := json.Unmarshal(decoded, &cfg); err != nil {
		return nil, fmt.Errorf("parse vmess JSON: %w", err)
	}

	if cfg.Add == "" {
		return nil, fmt.Errorf("vmess link missing address")
	}

	if cfg.Port == "" {
		return nil, fmt.Errorf("vmess link missing port")
	}

	return &cfg, nil
}

// ValidateLink validates a single VLESS or VMess link
func (p *LinkParser) ValidateLink(link string) error {
	link = strings.TrimSpace(link)

	if p.isValidVLESSLink(link) {
		return nil
	}

	if p.isValidVMessLink(link) {
		return nil
	}

	return fmt.Errorf("invalid link format (not VLESS or VMess): %s", link)
}

// noopLogger is a logger that does nothing
type noopLogger struct{}

func (l *noopLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (l *noopLogger) Info(msg string, keysAndValues ...interface{})  {}
func (l *noopLogger) Error(msg string, keysAndValues ...interface{}) {}
