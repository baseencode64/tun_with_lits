# 🔀 Split Tunneling - Technical Design Document

**Version**: v1.7.0  
**Status**: 📋 Design Phase  
**Author**: Senior Go Developer  
**Date**: 2026-06-01

---

## 📖 Table of Contents

1. [Overview](#overview)
2. [Motivation & Use Cases](#motivation--use-cases)
3. [Architecture Design](#architecture-design)
4. [Implementation Plan](#implementation-plan)
5. [Configuration](#configuration)
6. [Integration Points](#integration-points)
7. [Security Considerations](#security-considerations)
8. [Testing Strategy](#testing-strategy)
9. [Performance Impact](#performance-impact)
10. [Migration Path](#migration-path)

---

## Overview

### What is Split Tunneling?

Split Tunneling позволяет выборочно маршрутизировать трафик:

- **Часть трафика** → through VPN туннель (защищено)
- **Часть трафика** → directly через ISP (быстрее)

### Goals

1. ✅ **Performance**: Local трафик without VPN overhead
2. ✅ **Гибкость**: Пользователь контролирует что идет through VPN
3. ✅ **Bandwidth**: Экономия ресурсов VPN сервера
4. ✅ **Совместимость**: Работа with Kill Switch, DNS Protection, IPv6

### Non-Goals (Phase 1)

- ❌ Per-application routing (требует cgroups)
- ❌ Domain-based routing (требует DNS interception)
- ❌ Dynamic route updates (будет in Phase 2)

---

## Motivation & Use Cases

### Use Case 1: Corporate Network Access

**Issue**:

```
Пользователь работает from дома, подключен to VPN for доступа to заблокированным сайтам.
НО: Local принтер (192.168.1.100) and NAS (192.168.1.50) недоступны through VPN.
```

**Solution**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16" # Local network directly
    - "10.0.0.0/8" # Corporate network directly
```

**Result**: Принтер and NAS доступны directly, остальное through VPN ✅

---

### Use Case 2: Streaming Performance

**Issue**:

```
Netflix/YouTube through VPN медленные (VPN server далеко).
Но нужен VPN for доступа to заблокированным ресурсам.
```

**Solution**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "185.60.216.0/22" # Netflix CDN
    - "172.217.0.0/16" # Google (YouTube)
```

**Result**: Streaming on полной скорости, остальное through VPN ✅

---

### Use Case 3: Selective Privacy

**Issue**:

```
Нужна приватность only for конкретных сайтов (банкинг, email).
Остальной трафик can идти directly (экономия bandwidth).
```

**Solution**:

```yaml
split_tunneling:
  enabled: true
  mode: "include" # Только указанные through VPN
  include_cidrs:
    - "93.184.216.34/32" # example-bank.com
    - "142.250.0.0/15" # Gmail servers
```

**Result**: Только критичный трафик through VPN, остальное directly ✅

---

## Architecture Design

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Layer                       │
│  (Browser, curl, etc.)                                       │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                    Routing Decision                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  SplitTunnelRouter.ShouldRouteThroughVPN(destIP)     │   │
│  │                                                       │   │
│  │  if mode == "exclude":                               │   │
│  │    if destIP in excludeCIDRs → return false (direct) │   │
│  │    else → return true (VPN)                          │   │
│  │                                                       │   │
│  │  if mode == "include":                               │   │
│  │    if destIP in includeCIDRs → return true (VPN)     │   │
│  │    else → return false (direct)                      │   │
│  └──────────────────────────────────────────────────────┘   │
└────────────┬───────────────────────────┬────────────────────┘
             │                           │
             │ VPN Route                 │ Direct Route
             ▼                           ▼
┌────────────────────────┐   ┌──────────────────────────┐
│   TUN Device           │   │   Default Gateway        │
│   (192.18.0.1)         │   │   (e.g., 192.168.1.1)    │
│                        │   │                          │
│   → SOCKS5 Proxy       │   │   → ISP                  │
│   → XRay Server        │   │                          │
└────────────────────────┘   └──────────────────────────┘
```

### Component Design

#### 1. SplitTunnelConfig (Configuration)

```go
// pkg/client/split_tunnel_config.go

package client

import (
    "fmt"
    "net"
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
```

#### 2. SplitTunnelRouter (Routing Logic)

```go
// pkg/client/split_tunnel_router.go

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
```

#### 3. Integration into Client

```go
// pkg/client/client.go (additions)

type Client struct {
    // ... existing fields ...

    // Split tunneling support
    splitTunnelRouter *SplitTunnelRouter
}

// setupSplitTunneling configures split tunnel routing
func (c *Client) setupSplitTunneling() error {
    if !c.cfg.SplitTunnel.Enabled {
        c.cfg.Logger.Info("Split tunneling disabled")
        return nil
    }

    c.cfg.Logger.Info("Setting up split tunneling",
        "mode", c.cfg.SplitTunnel.Mode)

    // Create router
    router, err := NewSplitTunnelRouter(c.cfg.Logger, c.cfg.SplitTunnel)
    if err != nil {
        return fmt.Errorf("create split tunnel router: %w", err)
    }
    c.splitTunnelRouter = router

    switch c.cfg.SplitTunnel.Mode {
    case SplitTunnelModeExclude:
        return c.setupExcludeMode()
    case SplitTunnelModeInclude:
        return c.setupIncludeMode()
    default:
        return fmt.Errorf("unsupported split tunnel mode: %s", c.cfg.SplitTunnel.Mode)
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

    if c.cfg.SplitTunnel.Mode == SplitTunnelModeExclude {
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
```

---

## Implementation Plan

### Phase 1: Route-Based CIDR (v1.7.0)

#### Step 1: Configuration Structure

```
- [ ] Create pkg/client/split_tunnel_config.go
- [ ] Add SplitTunnelConfig struct
- [ ] Implement Validate() method
- [ ] Add to main Config struct
- [ ] Update config.yaml.example
```

#### Step 2: Router Implementation

```
- [ ] Create pkg/client/split_tunnel_router.go
- [ ] Implement SplitTunnelRouter struct
- [ ] Add CIDR parsing logic
- [ ] Implement ShouldRouteThroughVPN()
- [ ] Add GetExcludedRoutes() / GetIncludedRoutes()
```

#### Step 3: Client Integration

```
- [ ] Add splitTunnelRouter field to Client
- [ ] Implement setupSplitTunneling()
- [ ] Implement setupExcludeMode()
- [ ] Implement setupIncludeMode()
- [ ] Implement cleanupSplitTunneling()
- [ ] Call from Connect() and Disconnect()
```

#### Step 4: Testing

```
- [ ] Unit tests for SplitTunnelConfig.Validate()
- [ ] Unit tests for SplitTunnelRouter
- [ ] Integration tests for exclude mode
- [ ] Integration tests for include mode
- [ ] Manual testing with real VPN
```

#### Step 5: Documentation

```
- [ ] SPLIT_TUNNELING_USAGE.md (user guide)
- [ ] Update README.md
- [ ] Update config.yaml.example
- [ ] Add examples for common use cases
```

### Phase 2: Domain-Based Routing (v1.8.0)

```
- [ ] DNS interception and caching
- [ ] Domain to IP resolution
- [ ] Wildcard domain matching
- [ ] Dynamic route updates
- [ ] TTL-based cache expiration
```

### Phase 3: Advanced Features (v1.9.0)

```
- [ ] Per-application routing (cgroups)
- [ ] GeoIP-based routing
- [ ] Automatic route optimization
- [ ] Route analytics and reporting
```

---

## Configuration

### YAML Configuration

```yaml
# config.yaml

connection:
  from_raw_urls:
    - "https://example.com/servers.txt"

  enable_ipv6: true
  enable_dns_protection: true
  enable_kill_switch: true
  metrics_port: 9090

# Split Tunneling Configuration
split_tunneling:
  # Enable/disable split tunneling
  enabled: true

  # Mode: "exclude" or "include"
  # - exclude: All traffic through VPN EXCEPT excluded CIDRs
  # - include: Only included CIDRs through VPN, rest goes direct
  mode: "exclude"

  # Excluded CIDRs (used in "exclude" mode)
  # These IP ranges will bypass VPN and go direct through ISP
  exclude_cidrs:
    # Private networks (RFC 1918)
    - "192.168.0.0/16" # Local network
    - "10.0.0.0/8" # Corporate network
    - "172.16.0.0/12" # Docker networks

    # Link-local
    - "169.254.0.0/16" # APIPA addresses

    # Multicast
    - "224.0.0.0/4" # Multicast

    # Optional: Specific services
    # - "185.60.216.0/22" # Netflix CDN
    # - "172.217.0.0/16"  # Google services

  # Included CIDRs (used in "include" mode)
  # Only these IP ranges will go through VPN
  include_cidrs: []
    # Example for include mode:
    # - "93.184.216.34/32"  # Specific server
    # - "142.250.0.0/15"    # Google range

  # Future: Domain-based routing (Phase 2)
  exclude_domains: []
    # - "*.local"
    # - "*.lan"
    # - "netflix.com"

  include_domains: []
    # - "*.onion"
    # - "blocked-site.com"

server_selection:
  max_servers: 10
  timeout: "5s"

health_monitoring:
  check_interval: "10s"
  timeout: "5s"
  max_retries: 3

logging:
  format: "json"
  level: "info"
  file: "/var/log/goxray/goxray.log"
```

### CLI Flags (Future)

```bash
# Exclude mode
sudo goxray \
  --from-raw "https://example.com/servers.txt" \
  --split-tunnel-mode exclude \
  --split-tunnel-exclude "192.168.0.0/16,10.0.0.0/8"

# Include mode
sudo goxray \
  --from-raw "https://example.com/servers.txt" \
  --split-tunnel-mode include \
  --split-tunnel-include "93.184.216.34/32"
```

---

## Integration Points

### 1. Integration with Kill Switch

**Challenge**: Kill Switch blocks all traffic, but split tunneling needs to allow excluded routes.

**Solution**: Whitelist excluded CIDRs in Kill Switch rules.

```go
// In activateKillSwitch():
if c.cfg.SplitTunnel.Enabled && c.cfg.SplitTunnel.Mode == SplitTunnelModeExclude {
    // Add exceptions for excluded CIDRs
    for _, cidr := range c.cfg.SplitTunnel.ExcludeCIDRs {
        // iptables -A goxray_killswitch -d <CIDR> -j ACCEPT
        allowCmd := exec.Command("iptables", "-A", "goxray_killswitch", "-d", cidr, "-j", "ACCEPT")
        if err := allowCmd.Run(); err != nil {
            c.cfg.Logger.Warn("Failed to add kill switch exception for split tunnel",
                "cidr", cidr, "error", err)
        }
    }
}
```

### 2. Integration with DNS Protection

**Challenge**: DNS queries for excluded domains might leak.

**Solution**: Force all DNS through VPN regardless of split tunneling.

```go
// DNS always goes through VPN for security
// Even if destination is excluded, DNS resolution is protected
```

### 3. Integration with IPv6

**Challenge**: Split tunneling must work with both IPv4 and IPv6.

**Solution**: Support both IPv4 and IPv6 CIDRs in configuration.

```yaml
split_tunneling:
  exclude_cidrs:
    - "192.168.0.0/16" # IPv4
    - "fd00::/8" # IPv6 ULA
    - "fe80::/10" # IPv6 link-local
```

### 4. Integration with Metrics

**Challenge**: Track split tunnel routing decisions.

**Solution**: Add Prometheus metrics.

```go
var (
    splitTunnelRoutesVPN = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "split_tunnel_routes_vpn_total",
            Help: "Total routes through VPN",
        },
    )

    splitTunnelRoutesDirect = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "split_tunnel_routes_direct_total",
            Help: "Total routes direct (bypassing VPN)",
        },
    )
)
```

---

## Security Considerations

### 1. DNS Leaks

**Risk**: Excluded domains might leak DNS queries.

**Mitigation**:

- Force all DNS through VPN (even for excluded destinations)
- DNS Protection remains active regardless of split tunneling
- Log DNS queries for excluded domains

### 2. IP Leaks During Failover

**Risk**: During failover, excluded routes might expose real IP.

**Mitigation**:

- Kill Switch activates during failover
- Excluded CIDRs whitelisted in Kill Switch
- No traffic leaks during transition

### 3. Misconfiguration

**Risk**: User accidentally excludes critical traffic.

**Mitigation**:

- Validate configuration on startup
- Warn if common mistakes detected (e.g., excluding 0.0.0.0/0)
- Provide safe defaults

### 4. Route Conflicts

**Risk**: Excluded routes conflict with VPN routes.

**Mitigation**:

- Check for conflicts during setup
- Log warnings for overlapping CIDRs
- Fail-safe: VPN route takes precedence

---

## Testing Strategy

### Unit Tests

```go
// pkg/client/split_tunnel_config_test.go

func TestSplitTunnelConfig_Validate(t *testing.T) {
    tests := []struct {
        name    string
        config  SplitTunnelConfig
        wantErr bool
    }{
        {
            name: "valid exclude mode",
            config: SplitTunnelConfig{
                Enabled: true,
                Mode:    SplitTunnelModeExclude,
                ExcludeCIDRs: []string{"192.168.0.0/16"},
            },
            wantErr: false,
        },
        {
            name: "invalid CIDR",
            config: SplitTunnelConfig{
                Enabled: true,
                Mode:    SplitTunnelModeExclude,
                ExcludeCIDRs: []string{"invalid"},
            },
            wantErr: true,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

// pkg/client/split_tunnel_router_test.go

func TestSplitTunnelRouter_ShouldRouteThroughVPN(t *testing.T) {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    config := SplitTunnelConfig{
        Enabled: true,
        Mode:    SplitTunnelModeExclude,
        ExcludeCIDRs: []string{"192.168.0.0/16", "10.0.0.0/8"},
    }

    router, err := NewSplitTunnelRouter(logger, config)
    if err != nil {
        t.Fatalf("NewSplitTunnelRouter() error = %v", err)
    }

    tests := []struct {
        name     string
        destIP   string
        wantVPN  bool
    }{
        {"local network", "192.168.1.100", false},  // Excluded
        {"corporate", "10.0.5.50", false},          // Excluded
        {"public internet", "8.8.8.8", true},       // Not excluded
        {"cloudflare", "1.1.1.1", true},            // Not excluded
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ip := net.ParseIP(tt.destIP)
            got := router.ShouldRouteThroughVPN(ip)
            if got != tt.wantVPN {
                t.Errorf("ShouldRouteThroughVPN(%s) = %v, want %v", tt.destIP, got, tt.wantVPN)
            }
        })
    }
}
```

### Integration Tests

```bash
#!/bin/bash
# test_split_tunneling.sh

echo "=== Split Tunneling Integration Test ==="

# Test 1: Exclude mode - local network direct
echo "Test 1: Exclude mode"
sudo goxray --config test_exclude.yaml &
GOXRAY_PID=$!
sleep 5

# Verify local network goes direct
ping -c 1 192.168.1.1 && echo "✅ Local network accessible"

# Verify public internet goes through VPN
curl -s https://api.ipify.org | grep -v "$(curl -s --interface eth0 https://api.ipify.org)" && echo "✅ Public traffic through VPN"

sudo kill $GOXRAY_PID

# Test 2: Include mode - only specific IPs through VPN
echo "Test 2: Include mode"
sudo goxray --config test_include.yaml &
GOXRAY_PID=$!
sleep 5

# Verify included IP goes through VPN
# Verify excluded IP goes direct

sudo kill $GOXRAY_PID

echo "=== Tests Complete ==="
```

### Manual Testing Checklist

```
- [ ] Exclude mode: Local network (192.168.x.x) accessible
- [ ] Exclude mode: Public internet through VPN
- [ ] Include mode: Only specified IPs through VPN
- [ ] Include mode: Other traffic goes direct
- [ ] Kill Switch compatibility
- [ ] DNS Protection compatibility
- [ ] IPv6 support
- [ ] Failover during split tunneling
- [ ] Metrics reporting
- [ ] Configuration validation
- [ ] Route cleanup on disconnect
```

---

## Performance Impact

### Routing Decision Overhead

**Measurement**:

```
Routing decision: O(n) where n = number of CIDRs
Typical: n < 20, lookup time < 1μs
```

**Impact**: Negligible (< 0.01% CPU)

### Memory Usage

```
Per CIDR: ~100 bytes (net.IPNet struct)
Typical config: 10 CIDRs = 1KB
```

**Impact**: Negligible

### Latency

```
Excluded traffic: 0ms overhead (direct route)
Included traffic: Same as normal VPN
```

**Impact**: Positive for excluded traffic (faster)

---

## Migration Path

### From v1.6.3 to v1.7.0

**Backward Compatibility**: ✅ Full

```yaml
# Old config (v1.6.3) - still works
connection:
  from_raw_urls:
    - "https://example.com/servers.txt"
  enable_kill_switch: true

# New config (v1.7.0) - split tunneling optional
connection:
  from_raw_urls:
    - "https://example.com/servers.txt"
  enable_kill_switch: true

split_tunneling:
  enabled: false  # Default: disabled
```

**Migration Steps**:

1. Update binary to v1.7.0
2. (Optional) Add split_tunneling section to config
3. Restart service
4. Verify routing with `ip route` and `traceroute`

---

## Appendix

### Common CIDR Ranges

```yaml
# Private Networks (RFC 1918)
- "10.0.0.0/8" # Class A private
- "172.16.0.0/12" # Class B private
- "192.168.0.0/16" # Class C private

# Special Use
- "127.0.0.0/8" # Loopback
- "169.254.0.0/16" # Link-local (APIPA)
- "224.0.0.0/4" # Multicast
- "240.0.0.0/4" # Reserved

# IPv6
- "::1/128" # Loopback
- "fe80::/10" # Link-local
- "fd00::/8" # Unique local (ULA)
- "ff00::/8" # Multicast
```

### Troubleshooting Commands

```bash
# Check routing table
ip route show

# Trace route to destination
traceroute 8.8.8.8
traceroute 192.168.1.1

# Check if traffic goes through VPN
curl --interface tun0 https://api.ipify.org

# Check if traffic goes direct
curl --interface eth0 https://api.ipify.org

# Monitor routing decisions (logs)
sudo journalctl -u goxray -f | grep "split tunnel"
```

---

**Status**: 📋 Design Complete - Ready for Implementation  
**Next Step**: Begin Phase 1 implementation  
**Estimated Effort**: 3-5 days for Phase 1
