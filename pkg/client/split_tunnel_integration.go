package client

import (
	"fmt"
	"os/exec"

	"github.com/goxray/core/network/route"
)

// setupSplitTunneling configures split tunnel routing
func (c *Client) setupSplitTunneling(appConfig *AppConfig) error {
	if !appConfig.SplitTunnel.Enabled {
		c.cfg.Logger.Info("Split tunneling disabled")
		return nil
	}

	c.cfg.Logger.Info("Setting up split tunneling",
		"mode", appConfig.SplitTunnel.Mode)

	// Create router
	router, err := NewSplitTunnelRouter(c.cfg.Logger, appConfig.SplitTunnel)
	if err != nil {
		return fmt.Errorf("create split tunnel router: %w", err)
	}
	c.splitTunnelRouter = router

	// Update Prometheus metrics
	if appConfig.SplitTunnel.Enabled {
		goxrayConfigSplitTunnelEnabled.Set(1)
	} else {
		goxrayConfigSplitTunnelEnabled.Set(0)
	}

	// Reset mode metrics
	goxrayConfigSplitTunnelMode.Reset()
	goxrayConfigSplitTunnelMode.WithLabelValues(string(appConfig.SplitTunnel.Mode)).Set(1)

	switch appConfig.SplitTunnel.Mode {
	case SplitTunnelModeExclude:
		return c.setupExcludeMode()
	case SplitTunnelModeInclude:
		return c.setupIncludeMode()
	default:
		return fmt.Errorf("unsupported split tunnel mode: %s", appConfig.SplitTunnel.Mode)
	}
}

// setupExcludeMode: All traffic through VPN EXCEPT excluded CIDRs
func (c *Client) setupExcludeMode() error {
	c.cfg.Logger.Info("Configuring exclude mode split tunneling")

	// Get excluded routes (these go direct through gateway)
	excludedRoutes := c.splitTunnelRouter.GetExcludedRoutes()

	if len(excludedRoutes) == 0 {
		return fmt.Errorf("exclude mode requires at least one excluded CIDR")
	}

	// Add routes for excluded CIDRs to go through default gateway
	// This ensures they bypass the VPN tunnel
	for _, routeAddr := range excludedRoutes {
		routeOpts := route.Opts{
			Gateway: *c.cfg.GatewayIP,
			Routes:  []*route.Addr{routeAddr},
		}

		if err := c.routes.Add(routeOpts); err != nil {
			c.cfg.Logger.Warn("Failed to add excluded route",
				"route", routeAddr.String(),
				"error", err)
			// Continue with other routes
		} else {
			c.cfg.Logger.Info("Added excluded route (direct)",
				"route", routeAddr.String(),
				"gateway", c.cfg.GatewayIP.String())
		}
	}

	c.cfg.Logger.Info("Exclude mode configured",
		"excluded_routes", len(excludedRoutes))

	return nil
}

// setupIncludeMode: Only included CIDRs through VPN, rest goes direct
func (c *Client) setupIncludeMode() error {
	c.cfg.Logger.Info("Configuring include mode split tunneling")

	// Get included routes (these go through VPN)
	includedRoutes := c.splitTunnelRouter.GetIncludedRoutes()

	if len(includedRoutes) == 0 {
		return fmt.Errorf("include mode requires at least one included CIDR")
	}

	// IMPORTANT: In include mode, we REPLACE DefaultRoutesToTUN
	// with ONLY the included routes
	c.cfg.RoutesToTUN = includedRoutes

	c.cfg.Logger.Info("Include mode configured",
		"included_routes", len(includedRoutes),
		"note", "Only specified CIDRs will route through VPN")

	return nil
}

// cleanupSplitTunneling removes split tunnel routes
func (c *Client) cleanupSplitTunneling() error {
	if c.splitTunnelRouter == nil {
		return nil
	}

	c.cfg.Logger.Info("Cleaning up split tunneling routes")

	if c.splitTunnelRouter.config.Mode == SplitTunnelModeExclude {
		// Remove excluded routes
		excludedRoutes := c.splitTunnelRouter.GetExcludedRoutes()
		for _, routeAddr := range excludedRoutes {
			routeOpts := route.Opts{
				Gateway: *c.cfg.GatewayIP,
				Routes:  []*route.Addr{routeAddr},
			}

			if err := c.routes.Delete(routeOpts); err != nil {
				c.cfg.Logger.Debug("Failed to delete excluded route",
					"route", routeAddr.String(),
					"error", err)
			}
		}
	}

	c.cfg.Logger.Info("Split tunneling cleanup complete")
	return nil
}

// integrateSplitTunnelWithKillSwitch adds excluded CIDRs to kill switch whitelist
func (c *Client) integrateSplitTunnelWithKillSwitch(useIPv6 bool) error {
	if c.splitTunnelRouter == nil || c.splitTunnelRouter.config.Mode != SplitTunnelModeExclude {
		return nil // No integration needed
	}

	cmdName := "iptables"
	chainName := "goxray_killswitch"

	if useIPv6 {
		cmdName = "ip6tables"
	}

	c.cfg.Logger.Info("Integrating split tunneling with kill switch", "cmd", cmdName)

	// Add exceptions for excluded CIDRs
	for _, cidr := range c.splitTunnelRouter.config.ExcludeCIDRs {
		// Allow traffic to excluded CIDRs
		allowCmd := exec.Command(cmdName, "-I", chainName, "1", "-d", cidr, "-j", "ACCEPT")
		if err := allowCmd.Run(); err != nil {
			c.cfg.Logger.Warn("Failed to add kill switch exception for split tunnel",
				"cidr", cidr, "error", err)
			// Continue with other CIDRs
		} else {
			c.cfg.Logger.Debug("Added kill switch exception for split tunnel",
				"cidr", cidr, "cmd", cmdName)
		}
	}

	c.cfg.Logger.Info("Split tunneling integrated with kill switch successfully")
	return nil
}
