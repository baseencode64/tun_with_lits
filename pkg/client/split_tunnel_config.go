package client

import (
	"fmt"
	"net"
	"time"
)

// SplitTunnelMode defines routing behavior
type SplitTunnelMode string

const (
	// SplitTunnelModeExclude: All traffic through VPN EXCEPT excluded
	SplitTunnelModeExclude SplitTunnelMode = "exclude"

	// SplitTunnelModeInclude: Only included traffic through VPN
	SplitTunnelModeInclude SplitTunnelMode = "include"

	// SplitTunnelModeDisabled: All traffic through VPN (default)
	SplitTunnelModeDisabled SplitTunnelMode = "disabled"
)

// SplitTunnelConfig holds split tunneling configuration
type SplitTunnelConfig struct {
	// Enabled controls whether split tunneling is active
	Enabled bool `yaml:"enabled"`

	// Mode determines routing behavior ("exclude", "include", "disabled")
	Mode SplitTunnelMode `yaml:"mode"`

	// ExcludeCIDRs: IP ranges that bypass VPN (go direct)
	// Only used when Mode == "exclude"
	ExcludeCIDRs []string `yaml:"exclude_cidrs"`

	// IncludeCIDRs: IP ranges that go through VPN
	// Only used when Mode == "include"
	IncludeCIDRs []string `yaml:"include_cidrs"`

	// ExcludeDomains: Domains that bypass VPN (future: Phase 2)
	ExcludeDomains []string `yaml:"exclude_domains,omitempty"`

	// IncludeDomains: Domains that go through VPN (future: Phase 2)
	IncludeDomains []string `yaml:"include_domains,omitempty"`
}

// Validate checks configuration validity
func (c *SplitTunnelConfig) Validate() error {
	if !c.Enabled {
		return nil // Disabled config is always valid
	}

	// Validate mode
	switch c.Mode {
	case SplitTunnelModeExclude, SplitTunnelModeInclude, SplitTunnelModeDisabled:
		// Valid modes
	default:
		return fmt.Errorf("invalid split tunnel mode: %s (must be 'exclude', 'include', or 'disabled')", c.Mode)
	}

	// Validate CIDRs
	if c.Mode == SplitTunnelModeExclude {
		if len(c.ExcludeCIDRs) == 0 {
			return fmt.Errorf("exclude mode requires at least one exclude_cidr")
		}
		for _, cidr := range c.ExcludeCIDRs {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				return fmt.Errorf("invalid exclude CIDR %s: %w", cidr, err)
			}
		}
	}

	if c.Mode == SplitTunnelModeInclude {
		if len(c.IncludeCIDRs) == 0 {
			return fmt.Errorf("include mode requires at least one include_cidr")
		}
		for _, cidr := range c.IncludeCIDRs {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				return fmt.Errorf("invalid include CIDR %s: %w", cidr, err)
			}
		}
	}

	return nil
}

// SetDefaults applies default values
func (c *SplitTunnelConfig) SetDefaults() {
	if c.Mode == "" {
		c.Mode = SplitTunnelModeDisabled
	}
}

// GetRefreshInterval parses and returns refresh interval duration
func (c *SplitTunnelConfig) GetRefreshInterval() (time.Duration, error) {
	// Future: for dynamic route updates
	return 0, nil
}
