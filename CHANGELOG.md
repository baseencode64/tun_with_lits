# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.5.13] - 2026-05-25

### Fixed

- **Critical DNS resolution issue caused by incorrect DNS leak protection implementation**
  - Fixed duplicate DNS protection setup that was blocking DNS traffic even without `--dns-protection` flag
  - Removed incorrect iptables REDIRECT rules that were redirecting DNS traffic to SOCKS proxy port
  - SOCKS proxy cannot handle raw DNS protocol, causing complete DNS resolution failure
  - DNS now routes correctly through TUN interface via routing table entries only
  - Removed unused functions: `setupDNSLeakProtection()`, `blockDirectDNSAccess()`, `addFirewallRule()`, `addFirewallRuleNftables()`, `routeDNSThroughTUN()`
  - `setupDNSTrafficForcing()` now correctly does nothing, as DNS routing is handled by existing routes

### Changed

- Simplified DNS leak protection to only use routing table entries (no iptables manipulation)
- DNS protection is now only enabled when explicitly requested via `--dns-protection` flag
- Improved code maintainability by removing dead code and duplicate functionality

---

## [1.5.11] - 2026-05-22

### Added

- **DNS leak protection feature**
  - Added `--dns-protection` flag to enable comprehensive DNS leak prevention
  - Implemented iptables/ip6tables rules to force all DNS traffic through the VPN tunnel
  - Added new configuration option `enable_dns_protection` in YAML config
  - Ensures both IPv4 and IPv6 DNS queries are routed securely through TUN interface
  - Added explicit routes for major public DNS servers (Google, Cloudflare, Quad9)

### Changed

- Enhanced DNS security by implementing multiple layers of DNS leak prevention
- Improved routing logic to guarantee all DNS traffic passes through the VPN tunnel
- Updated CLI help text to document the new DNS protection option

---

## [1.5.10] - 2026-05-22

### Added

- IPv6 support with fallback mechanisms using system commands
- Support for multiple raw URLs with fallback capability
- Prometheus metrics endpoint with various VPN-related metrics
- JSON logging support with automatic rotation
- Configuration file support (YAML)

### Changed

- Improved failover mechanism with proper synchronization
- Enhanced health monitoring with SOCKS proxy verification
- Better error handling and logging throughout the application

---

## [1.5.9] - 2026-05-21

### Fixed

- Race condition fixes in connection management
- Goroutine leak prevention during failover operations
- Proper cleanup of resources during disconnection

---

## [1.5.8] - 2026-05-20

### Added

- Smart server selection algorithm with weighted scoring
- Health monitoring with automatic failover capability
- Performance optimizations for connection handling

---

## [1.5.7] - 2026-05-19

### Added

- Support for multiple VPN protocols (VLESS, VMess, Trojan, Shadowsocks)
- Improved TUN interface configuration and management
- Better error reporting and diagnostics

---

## [1.5.6] - 2026-05-18

### Fixed

- TUN interface setup and routing issues
- Memory leak fixes in connection management
- Stability improvements for long-running connections

---

## [1.5.5] - 2026-05-17

### Added

- SOCKS5 proxy support for traffic routing
- Automatic gateway detection and configuration
- Traffic monitoring and statistics

---

## [1.5.4] - 2026-05-16

### Changed

- Refactored client architecture for better modularity
- Improved logging and debugging capabilities
- Enhanced security measures for data transmission

---

## [1.5.3] - 2026-05-15

### Fixed

- Various bug fixes and stability improvements
- Memory management optimizations
- Connection handling improvements

---

## [1.5.2] - 2026-05-14

### Added

- Support for VLESS protocol with Reality TLS
- Configuration options for custom settings
- Better error handling and recovery mechanisms

---

## [1.5.1] - 2026-05-13

### Added

- Initial release with basic VPN functionality
- TUN interface support for packet routing
- XRay core integration for protocol handling

---

## [1.5.0] - 2026-05-22

### Added

- **DNS routing through TUN interface**
  - Added explicit DNS routes for major public DNS servers (Google, Cloudflare, Quad9)
  - Ensures DNS resolution works correctly through VPN tunnel
  - Prevents DNS leaks by routing all DNS queries through TUN
  - Supports both IPv4 and IPv6 DNS servers when IPv6 is enabled
  - Fixes issue where applications couldn't resolve domain names after VPN connection

### Changed

- DNS routes added immediately after main traffic routes in `setupTunnel()`
- DNS route cleanup added to `Disconnect()` for proper resource management
- DNS route failures treated as warnings (non-critical) to maintain connectivity

---

## [1.4.4] - 2026-05-22

### Fixed

- **Health check using wrong port causing false failovers**
  - Fixed health checker to use SOCKS proxy port from `InboundProxy.Port` instead of remote server port
  - Previously tried to connect to `127.0.0.1:8443` (server port) instead of actual SOCKS port (e.g., 42883)
  - This caused immediate connection refused errors and triggered unnecessary failovers every 10 seconds
  - Now correctly monitors the dynamic SOCKS proxy port assigned during VPN connection

---

## [1.4.3] - 2026-05-22

### Fixed

- **Unreliable health checks causing false failovers**
  - Changed health check from direct TCP dial to remote server to SOCKS proxy verification
  - Now properly tests tunnel functionality by sending SOCKS5 greeting and verifying response
  - Eliminates false positives with Reality/VLESS servers that reject plain TCP connections
  - Resolves intermittent traffic loss due to unnecessary server failovers
- **Health check timeout errors in logs**
  - Removed misleading "read tcp ... i/o timeout" warnings for Reality connections
  - Health checks now complete successfully with proper SOCKS protocol handshake
  - Added detailed timing metrics (dial_ms, write_ms, read_ms, total_ms)

### Changed

- Health checker now monitors `127.0.0.1:{port}` instead of remote server address
- Improved health check reliability for all protocols (VLESS, VMess, Trojan, Shadowsocks)

---

## [1.4.1] - 2026-05-22

### Fixed

- **Critical routing loop causing TLS EOF errors**
  - Fixed order of operations in `Connect()`: Xray server route exception now added BEFORE TUN interface creation
  - Prevents infinite routing loop where Xray traffic was being routed through TUN back to itself
  - Resolves "TLS connect error: unexpected eof while reading" when using curl or other HTTPS clients
- **Missing SOCKS proxy readiness check**
  - Added verification that Xray SOCKS proxy is listening before starting pipe.Copy
  - Waits up to 2 seconds for proxy to become ready with 100ms polling intervals
  - Proper cleanup of all resources (TUN, routes, Xray instance) on any failure
- **Insufficient health check diagnostics**
  - Enhanced health checker with dial/read duration metrics
  - Better logging of connection timing for troubleshooting
  - More detailed error messages with timing information

### Changed

- Improved error handling and resource cleanup in `Connect()` method
- Enhanced logging with connection status details (TUN address, Xray server IP, SOCKS proxy address)
- Health check logs now include performance metrics (dial_ms, read_ms)

---

## [1.4.0]

### Added

- **Connection status logging with IP information**
  - New `GetConnectionStatus()` method returns detailed connection info
  - New `LogConnectionStatus()` logs current status every 30 seconds
  - Logs include: TUN interface, local IPv4/IPv6 addresses, XRay server IP, traffic stats
  - Initial status logged immediately after successful connection
  - Periodic health and connection status updates in monitoring loop
- **Multiple raw URL support with fallback**
  - New `connection.from_raw_urls` config option (array of URLs)
  - Automatic fallback: if first URL fails, tries next ones in order
  - Works with both config files and CLI (`--from-raw` uses first successful URL for refresh)
  - Enhanced error reporting shows which URL succeeded/failed
  - Backward compatible: single `from_raw` still works
- **Configuration file support** (YAML format)
  - New `--config` flag to load settings from YAML file
  - Support for all application settings: connection, logging, health monitoring, server selection
  - CLI arguments override config file values (priority: CLI > config > defaults)
  - Automatic validation with clear error messages
  - Example configuration: `config.yaml.example`
- Structured server logging with JSON-compatible format ([#XX](link-to-issue))
  - New `LogServerReport()` method for structured per-server log entries
  - Each server logged separately with fields: rank, host, port, latency_ms, status
  - Compatible with ELK Stack, Grafana Loki, Splunk
  - Replaced multi-line text reports with machine-readable JSON entries

### Fixed

- **Config file not loading `from_raw_urls`**: Fixed issue where multiple raw URLs from YAML config were ignored
  - Now properly reads and uses all URLs from `connection.from_raw_urls` array
  - Implements automatic fallback between URLs on fetch failure

### Changed

- IPv6 address changed from invalid `fd00:goxray::1` to valid ULA `fd00:dead:beef::1`
- Server report logging now uses structured format instead of concatenated strings
- Improved periodic refresh logging with better error context

---

## [1.3.1]

### Fixed

- **Config file not loading `from_raw_urls`**: Fixed issue where multiple raw URLs from YAML config were ignored
  - Now properly reads and uses all URLs from `connection.from_raw_urls` array
  - Implements automatic fallback between URLs on fetch failure

## [1.3.0]

### Added

- **Multiple raw URL support with fallback**
  - New `connection.from_raw_urls` config option (array of URLs)
  - Automatic fallback: if first URL fails, tries next ones in order
  - Works with both config files and CLI (`--from-raw` uses first successful URL for refresh)
  - Enhanced error reporting shows which URL succeeded/failed
  - Backward compatible: single `from_raw` still works
- **Configuration file support** (YAML format)
  - New `--config` flag to load settings from YAML file
  - Support for all application settings: connection, logging, health monitoring, server selection
  - CLI arguments override config file values (priority: CLI > config > defaults)
  - Automatic validation with clear error messages
  - Example configuration: `config.yaml.example`
  - Comprehensive documentation in `CONFIG_FILE.md`
- Structured server logging with JSON-compatible format ([#XX](link-to-issue))
  - New `LogServerReport()` method for structured per-server log entries
  - Each server logged separately with fields: rank, host, port, latency_ms, status
  - Compatible with ELK Stack, Grafana Loki, Splunk
  - Replaced multi-line text reports with machine-readable JSON entries

### Changed

- IPv6 address changed from invalid `fd00:goxray::1` to valid ULA `fd00:dead:beef::1`
- Server report logging now uses structured format instead of concatenated strings
- Improved periodic refresh logging with better error context

---

## [1.2.0] - 2026-05-22

### Added

- **Full IPv6 support** with dual-stack tunneling
  - Enable with `--ipv6` flag
  - Automatic IPv6 route configuration (::/1 and 8000::/1)
  - IPv6 address: fd00:dead:beef::1/64 (ULA range)
  - Proper cleanup on disconnect
  - System command fallback when library doesn't support IPv6 routing tables
- **JSON logging format** with log rotation
  - New CLI flags: `--log-format`, `--log-level`, `--log-file`
  - Rotation settings: `--log-max-size`, `--log-max-backups`, `--log-max-age`
  - Uses lumberjack library for automatic rotation
  - Compatible with centralized logging systems (ELK, Loki, Splunk)
- **Periodic server list refresh**
  - Flag: `--refresh-interval` (e.g., 5m, 10m, 1h)
  - Automatically updates server list without restart
  - Discovers new servers dynamically
  - Integrates with health monitoring system

- **Health monitoring system**
  - Automatic health checks every 10 seconds
  - Configurable timeout and retry count
  - Automatic failover to backup servers (~30s recovery)
  - Periodic health status reporting

### Changed

- Server selection now logs all available servers sorted by latency
- Improved error handling and graceful degradation
- Updated documentation structure

### Fixed

- IPv6 routing table configuration (library fallback to system commands)
- Invalid IPv6 address format (changed to proper ULA address)
- Goroutine leaks in health checker
- Race conditions in concurrent server checking
- Panic on nil pointer dereference in link parser

---

## [1.1.0] - 2025-01-XX

### Added

- **Automatic server selection from raw VLESS lists**
  - New `--from-raw <URL>` flag
  - Fetches server list from HTTP/HTTPS URL
  - Parses VLESS links, ignores comments and invalid lines
  - Concurrent latency checking (configurable max parallelism)
  - Selects optimal server based on lowest latency
- **VPN connection with automatic fallback**
  - Tries multiple servers sequentially if first fails
  - Detailed connection reports
  - Smart sorting by latency
- **Link parser and server selector components**
  - `LinkParser` for VLESS link validation
  - `ServerSelector` for intelligent server selection
  - `SlogAdapter` for logging integration
- **Comprehensive test coverage**
  - Unit tests for link parsing
  - Unit tests for server selection
  - Integration tests for VPN connector

### Changed

- Main application refactored to support both direct links and raw URLs
- Improved error messages and logging
- Better documentation structure

### Fixed

- Various stability improvements
- Memory leak prevention in server list processing

---

## [1.0.0] - Initial Release

### Added

- Basic XRay VPN client functionality
- TUN device setup and traffic routing
- Support for all Xray-core protocols (vless, vmess, etc.)
- Command-line interface
- Library API for integration into other projects
- Docker support with multi-stage builds
- Linux capabilities configuration
- Cross-compilation support (Linux ARM64, AMD64)

### Features

- Soft routing rules (cleaned up on exit)
- TLS configuration options
- Protocol link notation support
- Health checking via ping
- Metrics collection

---

## Version History Summary

| Version | Date       | Key Features                                                    |
| ------- | ---------- | --------------------------------------------------------------- |
| 1.2.0   | 2026-05-22 | IPv6 support, JSON logging, periodic refresh, health monitoring |
| 1.1.0   | 2025-01-XX | Auto server selection, fallback connection, raw URL support     |
| 1.0.0   | Initial    | Basic VPN client, TUN device, protocol support                  |

---

## Migration Guide

### From 1.1.x to 1.2.0

**New flags available:**

```bash
# Enable IPv6
sudo goxray --from-raw url --ipv6

# Use JSON logging with rotation
sudo goxray --from-raw url \
  --log-format json \
  --log-file /var/log/goxray/goxray.log \
  --log-max-size 100 \
  --log-max-backups 10 \
  --log-max-age 30

# Enable periodic refresh
sudo goxray --from-raw url --refresh-interval 5m
```

**Breaking changes:** None - fully backward compatible.

### From 1.0.x to 1.1.0

**New usage pattern:**

```bash
# Old way (still works)
sudo goxray vless://uuid@server.com:443

# New way (recommended)
sudo goxray --from-raw https://example.com/links.txt
```

**Breaking changes:** None - old syntax still supported.

---

## Deprecated Features

None currently.

---

## Removed Features

None currently.

---

## Security Notices

- Always verify server authenticity before connecting
- Use `TLSAllowInsecure: false` in production (default)
- Regularly update server lists
- Monitor health status for anomalies

---

## Contributors

See GitHub repository for full contributor list.

---

**Note**: For detailed technical documentation on specific features, see:

- README.md - General usage and examples
- HEALTH_MONITORING.md - Health check system details
- DEPLOYMENT.md - Deployment instructions
