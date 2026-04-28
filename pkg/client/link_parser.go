package client

import (
	"fmt"
	"net/url"
	"strings"
)

// LinkParser handles parsing and validation of VLESS links
type LinkParser struct {
	logger Logger
}

// NewLinkParser creates a new link parser instance
func NewLinkParser(logger Logger) *LinkParser {
	if logger == nil {
		logger = &noopLogger{}
	}
	return &LinkParser{logger: logger}
}

// ParseLinksFromRaw parses raw text and extracts VLESS links
func (p *LinkParser) ParseLinksFromRaw(rawText string) []string {
	var links []string

	lines := strings.Split(rawText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if p.isValidVLESSLink(line) {
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

// ValidateLink validates a single VLESS link
func (p *LinkParser) ValidateLink(link string) error {
	link = strings.TrimSpace(link)

	if !p.isValidVLESSLink(link) {
		return fmt.Errorf("invalid VLESS link format: %s", link)
	}

	return nil
}

// noopLogger is a logger that does nothing
type noopLogger struct{}

func (l *noopLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (l *noopLogger) Info(msg string, keysAndValues ...interface{})  {}
func (l *noopLogger) Error(msg string, keysAndValues ...interface{}) {}
