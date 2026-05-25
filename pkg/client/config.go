package client

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// AppConfig represents the complete application configuration
type AppConfig struct {
	// Connection settings
	Connection ConnectionConfig `yaml:"connection"`
	
	// Server selection settings
	ServerSelection ServerSelectionConfig `yaml:"server_selection"`
	
	// Reconnection settings (persistence & auto-reconnect)
	Reconnection ReconnectionConfig `yaml:"reconnection"`
	
	// Logging settings
	Logging LoggingConfig `yaml:"logging"`
	
	// Health monitoring settings
	HealthMonitoring HealthMonitoringConfig `yaml:"health_monitoring"`
}

// ConnectionConfig holds VPN connection related settings
type ConnectionConfig struct {
	// Direct VLESS link (mutually exclusive with FromRaw URLs)
	Link string `yaml:"link,omitempty"`
	
	// Single raw URL for server list (legacy, use FromRawURLs instead)
	FromRaw string `yaml:"from_raw,omitempty"`
	
	// Multiple raw URLs for server lists with fallback support
	// If first URL fails, tries next ones in order
	FromRawURLs []string `yaml:"from_raw_urls,omitempty"`
	
	// Enable IPv6 support
	EnableIPv6 bool `yaml:"enable_ipv6"`
	
	// Enable DNS leak protection
	EnableDNSProtection bool `yaml:"enable_dns_protection"`
	
	// TLS allow insecure certificates
	TLSAllowInsecure bool `yaml:"tls_allow_insecure"`
	
	// Prometheus metrics endpoint port (0 = disabled)
	MetricsPort int `yaml:"metrics_port"`
}

// ServerSelectionConfig holds server selection and refresh settings
type ServerSelectionConfig struct {
	// Periodic refresh interval (e.g., "5m", "10m", "1h")
	RefreshInterval string `yaml:"refresh_interval,omitempty"`
	
	// Maximum number of servers to check
	MaxServers int `yaml:"max_servers"`
	
	// Timeout per server health check (e.g., "5s", "10s")
	Timeout string `yaml:"timeout,omitempty"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	// Log format: text or json
	Format string `yaml:"format"`
	
	// Log level: debug, info, warn, error
	Level string `yaml:"level"`
	
	// Log file path (empty for stdout only)
	File string `yaml:"file,omitempty"`
	
	// Max log file size in MB before rotation
	MaxSize int `yaml:"max_size"`
	
	// Max number of backup log files
	MaxBackups int `yaml:"max_backups"`
	
	// Max age of backup logs in days
	MaxAge int `yaml:"max_age"`
}

// HealthMonitoringConfig holds health check settings
type HealthMonitoringConfig struct {
	// Check interval duration (e.g., "10s", "30s")
	CheckInterval string `yaml:"check_interval,omitempty"`
	
	// Check timeout duration (e.g., "5s", "10s")
	Timeout string `yaml:"timeout,omitempty"`
	
	// Max retries before failover
	MaxRetries int `yaml:"max_retries"`
}

// LoadConfig loads configuration from YAML file
func LoadConfig(filePath string) (*AppConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	// Set defaults if not specified
	config.setDefaults()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for unspecified fields
func (c *AppConfig) setDefaults() {
	// Server selection defaults
	if c.ServerSelection.MaxServers == 0 {
		c.ServerSelection.MaxServers = 10
	}
	if c.ServerSelection.Timeout == "" {
		c.ServerSelection.Timeout = "5s"
	}

	// Reconnection defaults
	c.Reconnection.SetDefaults()

	// Logging defaults
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.MaxSize == 0 {
		c.Logging.MaxSize = 100
	}
	if c.Logging.MaxBackups == 0 {
		c.Logging.MaxBackups = 3
	}
	if c.Logging.MaxAge == 0 {
		c.Logging.MaxAge = 28
	}

	// Health monitoring defaults
	if c.HealthMonitoring.CheckInterval == "" {
		c.HealthMonitoring.CheckInterval = "10s"
	}
	if c.HealthMonitoring.Timeout == "" {
		c.HealthMonitoring.Timeout = "5s"
	}
	if c.HealthMonitoring.MaxRetries == 0 {
		c.HealthMonitoring.MaxRetries = 3
	}
}

// Validate validates the configuration
func (c *AppConfig) Validate() error {
	// Validate connection settings
	hasLink := c.Connection.Link != ""
	hasFromRaw := c.Connection.FromRaw != ""
	hasFromRawURLs := len(c.Connection.FromRawURLs) > 0
	
	// Must have at least one connection method
	if !hasLink && !hasFromRaw && !hasFromRawURLs {
		return fmt.Errorf("must specify either 'connection.link', 'connection.from_raw', or 'connection.from_raw_urls'")
	}
	
	// Cannot mix link with raw URLs
	if hasLink && (hasFromRaw || hasFromRawURLs) {
		return fmt.Errorf("'connection.link' cannot be used with 'from_raw' or 'from_raw_urls'")
	}
	
	// Cannot use both single and multiple raw URLs
	if hasFromRaw && hasFromRawURLs {
		return fmt.Errorf("'connection.from_raw' and 'connection.from_raw_urls' are mutually exclusive, use only one")
	}

	// Validate log format
	if c.Logging.Format != "text" && c.Logging.Format != "json" {
		return fmt.Errorf("logging.format must be 'text' or 'json', got '%s'", c.Logging.Format)
	}

	// Validate log level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("logging.level must be 'debug', 'info', 'warn', or 'error', got '%s'", c.Logging.Level)
	}

	// Validate durations
	if c.ServerSelection.RefreshInterval != "" {
		if _, err := time.ParseDuration(c.ServerSelection.RefreshInterval); err != nil {
			return fmt.Errorf("invalid server_selection.refresh_interval: %w", err)
		}
	}
	if _, err := time.ParseDuration(c.ServerSelection.Timeout); err != nil {
		return fmt.Errorf("invalid server_selection.timeout: %w", err)
	}
	if _, err := time.ParseDuration(c.HealthMonitoring.CheckInterval); err != nil {
		return fmt.Errorf("invalid health_monitoring.check_interval: %w", err)
	}
	if _, err := time.ParseDuration(c.HealthMonitoring.Timeout); err != nil {
		return fmt.Errorf("invalid health_monitoring.timeout: %w", err)
	}

	// Validate numeric values
	if c.ServerSelection.MaxServers <= 0 {
		return fmt.Errorf("server_selection.max_servers must be positive")
	}
	if c.Logging.MaxSize <= 0 {
		return fmt.Errorf("logging.max_size must be positive")
	}
	if c.Logging.MaxBackups < 0 {
		return fmt.Errorf("logging.max_backups cannot be negative")
	}
	if c.Logging.MaxAge < 0 {
		return fmt.Errorf("logging.max_age cannot be negative")
	}
	if c.HealthMonitoring.MaxRetries <= 0 {
		return fmt.Errorf("health_monitoring.max_retries must be positive")
	}

	// Validate reconnection settings
	if c.Reconnection.MaxRetries < 0 {
		return fmt.Errorf("reconnection.max_retries cannot be negative")
	}
	if c.Reconnection.MinBackoff <= 0 && c.Reconnection.MinBackoffStr != "" {
		if _, err := time.ParseDuration(c.Reconnection.MinBackoffStr); err != nil {
			return fmt.Errorf("invalid reconnection.min_backoff: %w", err)
		}
	}
	if c.Reconnection.MaxBackoff <= 0 && c.Reconnection.MaxBackoffStr != "" {
		if _, err := time.ParseDuration(c.Reconnection.MaxBackoffStr); err != nil {
			return fmt.Errorf("invalid reconnection.max_backoff: %w", err)
		}
	}
	if c.Reconnection.BackoffFactor <= 1.0 && c.Reconnection.BackoffFactor != 0 {
		return fmt.Errorf("reconnection.backoff_factor must be > 1.0, got %.1f", c.Reconnection.BackoffFactor)
	}

	return nil
}

// GetRefreshInterval parses refresh interval as time.Duration
func (c *ServerSelectionConfig) GetRefreshInterval() (time.Duration, error) {
	if c.RefreshInterval == "" {
		return 0, nil
	}
	return time.ParseDuration(c.RefreshInterval)
}

// GetTimeout parses timeout as time.Duration
func (c *ServerSelectionConfig) GetTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Timeout)
}

// GetCheckInterval parses check interval as time.Duration
func (c *HealthMonitoringConfig) GetCheckInterval() (time.Duration, error) {
	return time.ParseDuration(c.CheckInterval)
}

// GetHealthTimeout parses health check timeout as time.Duration
func (c *HealthMonitoringConfig) GetHealthTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Timeout)
}
