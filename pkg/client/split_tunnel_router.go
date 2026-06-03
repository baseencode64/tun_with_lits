package client

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/goxray/core/network/route"
)

// SplitTunnelRouter manages split tunneling routing decisions
type SplitTunnelRouter struct {
	logger *slog.Logger
	config SplitTunnelConfig

	// Parsed CIDR networks for fast lookup
	excludeNets []*net.IPNet
	includeNets []*net.IPNet
}

// NewSplitTunnelRouter creates a new router
func NewSplitTunnelRouter(logger *slog.Logger, config SplitTunnelConfig) (*SplitTunnelRouter, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	router := &SplitTunnelRouter{
		logger: logger,
		config: config,
	}

	// Parse and cache CIDR networks
	if err := router.parseCIDRs(); err != nil {
		return nil, fmt.Errorf("parse CIDRs: %w", err)
	}

	logger.Info("Split tunnel router initialized",
		"mode", config.Mode,
		"exclude_count", len(router.excludeNets),
		"include_count", len(router.includeNets))

	return router, nil
}

// parseCIDRs parses CIDR strings into net.IPNet for fast lookup
func (r *SplitTunnelRouter) parseCIDRs() error {
	// Parse exclude CIDRs
	for _, cidrStr := range r.config.ExcludeCIDRs {
		_, ipNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			return fmt.Errorf("parse exclude CIDR %s: %w", cidrStr, err)
		}
		r.excludeNets = append(r.excludeNets, ipNet)
		r.logger.Debug("Parsed exclude CIDR", "cidr", cidrStr)
	}

	// Parse include CIDRs
	for _, cidrStr := range r.config.IncludeCIDRs {
		_, ipNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			return fmt.Errorf("parse include CIDR %s: %w", cidrStr, err)
		}
		r.includeNets = append(r.includeNets, ipNet)
		r.logger.Debug("Parsed include CIDR", "cidr", cidrStr)
	}

	return nil
}

// ShouldRouteThroughVPN determines if traffic to destIP should go through VPN
func (r *SplitTunnelRouter) ShouldRouteThroughVPN(destIP net.IP) bool {
	if !r.config.Enabled {
		return true // Split tunneling disabled, all traffic through VPN
	}

	switch r.config.Mode {
	case SplitTunnelModeExclude:
		// Check if IP is in exclude list
		for _, ipNet := range r.excludeNets {
			if ipNet.Contains(destIP) {
				r.logger.Debug("IP matched exclude rule - routing direct",
					"ip", destIP.String(),
					"cidr", ipNet.String())
				return false // Exclude from VPN (go direct)
			}
		}
		return true // Not excluded, go through VPN

	case SplitTunnelModeInclude:
		// Check if IP is in include list
		for _, ipNet := range r.includeNets {
			if ipNet.Contains(destIP) {
				r.logger.Debug("IP matched include rule - routing through VPN",
					"ip", destIP.String(),
					"cidr", ipNet.String())
				return true // Include in VPN
			}
		}
		return false // Not included, go direct

	default:
		return true // Default: all through VPN
	}
}

// GetExcludedRoutes returns route.Addr entries for excluded CIDRs
// These routes will point to the default gateway (bypass VPN)
func (r *SplitTunnelRouter) GetExcludedRoutes() []*route.Addr {
	if r.config.Mode != SplitTunnelModeExclude {
		return nil
	}

	var routes []*route.Addr
	for _, cidrStr := range r.config.ExcludeCIDRs {
		routes = append(routes, route.MustParseAddr(cidrStr))
	}
	return routes
}

// GetIncludedRoutes returns route.Addr entries for included CIDRs
// These routes will point to TUN device (through VPN)
func (r *SplitTunnelRouter) GetIncludedRoutes() []*route.Addr {
	if r.config.Mode != SplitTunnelModeInclude {
		return nil
	}

	var routes []*route.Addr
	for _, cidrStr := range r.config.IncludeCIDRs {
		routes = append(routes, route.MustParseAddr(cidrStr))
	}
	return routes
}

// GetStats returns statistics about routing decisions
func (r *SplitTunnelRouter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":       r.config.Enabled,
		"mode":          r.config.Mode,
		"exclude_count": len(r.excludeNets),
		"include_count": len(r.includeNets),
	}
}
