package client

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig_ValidConfig(t *testing.T) {
	configContent := `
connection:
  from_raw: "https://example.com/links.txt"
  enable_ipv6: true
  tls_allow_insecure: false

server_selection:
  refresh_interval: "5m"
  max_servers: 15
  timeout: "10s"

logging:
  format: "json"
  level: "debug"
  file: "/var/log/goxray/test.log"
  max_size: 50
  max_backups: 5
  max_age: 14

health_monitoring:
  check_interval: "15s"
  timeout: "8s"
  max_retries: 5
`
	tmpFile := createTempConfig(t, configContent)
	defer os.Remove(tmpFile)

	config, err := LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load valid config: %v", err)
	}

	// Verify connection settings
	if config.Connection.FromRaw != "https://example.com/links.txt" {
		t.Errorf("Expected from_raw 'https://example.com/links.txt', got '%s'", config.Connection.FromRaw)
	}
	if !config.Connection.EnableIPv6 {
		t.Error("Expected enable_ipv6 to be true")
	}

	// Verify server selection settings
	interval, _ := config.ServerSelection.GetRefreshInterval()
	if interval != 5*time.Minute {
		t.Errorf("Expected refresh_interval 5m, got %v", interval)
	}
	if config.ServerSelection.MaxServers != 15 {
		t.Errorf("Expected max_servers 15, got %d", config.ServerSelection.MaxServers)
	}

	// Verify logging settings
	if config.Logging.Format != "json" {
		t.Errorf("Expected format 'json', got '%s'", config.Logging.Format)
	}
	if config.Logging.Level != "debug" {
		t.Errorf("Expected level 'debug', got '%s'", config.Logging.Level)
	}

	// Verify health monitoring settings
	checkInterval, _ := config.HealthMonitoring.GetCheckInterval()
	if checkInterval != 15*time.Second {
		t.Errorf("Expected check_interval 15s, got %v", checkInterval)
	}
	if config.HealthMonitoring.MaxRetries != 5 {
		t.Errorf("Expected max_retries 5, got %d", config.HealthMonitoring.MaxRetries)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	configContent := `
connection:
  link: "vless://test@host:443"
`
	tmpFile := createTempConfig(t, configContent)
	defer os.Remove(tmpFile)

	config, err := LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load config with defaults: %v", err)
	}

	// Verify defaults are applied
	if config.ServerSelection.MaxServers != 10 {
		t.Errorf("Expected default max_servers 10, got %d", config.ServerSelection.MaxServers)
	}
	if config.Logging.Format != "text" {
		t.Errorf("Expected default format 'text', got '%s'", config.Logging.Format)
	}
	if config.Logging.Level != "info" {
		t.Errorf("Expected default level 'info', got '%s'", config.Logging.Level)
	}
	if config.Logging.MaxSize != 100 {
		t.Errorf("Expected default max_size 100, got %d", config.Logging.MaxSize)
	}
	if config.HealthMonitoring.MaxRetries != 3 {
		t.Errorf("Expected default max_retries 3, got %d", config.HealthMonitoring.MaxRetries)
	}
}

func TestLoadConfig_MissingRequiredField(t *testing.T) {
	configContent := `
server_selection:
  max_servers: 10
`
	tmpFile := createTempConfig(t, configContent)
	defer os.Remove(tmpFile)

	_, err := LoadConfig(tmpFile)
	if err == nil {
		t.Fatal("Expected error for missing required field, got nil")
	}
}

func TestLoadConfig_InvalidLogFormat(t *testing.T) {
	configContent := `
connection:
  link: "vless://test@host:443"
logging:
  format: "invalid"
`
	tmpFile := createTempConfig(t, configContent)
	defer os.Remove(tmpFile)

	_, err := LoadConfig(tmpFile)
	if err == nil {
		t.Fatal("Expected error for invalid log format, got nil")
	}
}

func TestLoadConfig_InvalidLogLevel(t *testing.T) {
	configContent := `
connection:
  link: "vless://test@host:443"
logging:
  level: "verbose"
`
	tmpFile := createTempConfig(t, configContent)
	defer os.Remove(tmpFile)

	_, err := LoadConfig(tmpFile)
	if err == nil {
		t.Fatal("Expected error for invalid log level, got nil")
	}
}

func TestLoadConfig_BothLinkAndFromRaw(t *testing.T) {
	configContent := `
connection:
  link: "vless://test@host:443"
  from_raw: "https://example.com/links.txt"
`
	tmpFile := createTempConfig(t, configContent)
	defer os.Remove(tmpFile)

	_, err := LoadConfig(tmpFile)
	if err == nil {
		t.Fatal("Expected error when both link and from_raw are specified, got nil")
	}
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	_, err := LoadConfig("/non/existent/path.yaml")
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	configContent := `
connection:
  from_raw: "https://example.com/links.txt"
  invalid yaml here: [
`
	tmpFile := createTempConfig(t, configContent)
	defer os.Remove(tmpFile)

	_, err := LoadConfig(tmpFile)
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

func TestGetRefreshInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		expected time.Duration
		hasError bool
	}{
		{"Empty interval", "", 0, false},
		{"5 minutes", "5m", 5 * time.Minute, false},
		{"1 hour", "1h", 1 * time.Hour, false},
		{"30 seconds", "30s", 30 * time.Second, false},
		{"Invalid format", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ServerSelectionConfig{RefreshInterval: tt.interval}
			result, err := cfg.GetRefreshInterval()
			
			if tt.hasError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func createTempConfig(t *testing.T, content string) string {
	t.Helper()
	
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")
	
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	
	return tmpFile
}
