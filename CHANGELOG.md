# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Structured server logging with JSON-compatible format ([#XX](link-to-issue))
  - New `LogServerReport()` method for structured per-server log entries
  - Each server logged separately with fields: rank, host, port, latency_ms, status
  - Compatible with ELK Stack, Grafana Loki, Splunk
  - Replaced multi-line text reports with machine-readable JSON entries

### Changed
- IPv6 address changed from invalid `fd00:goxray::1` to valid ULA `fd00:dead:beef::1`
- Server report logging now uses structured format instead of concatenated strings

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

| Version | Date | Key Features |
|---------|------|--------------|
| 1.2.0 | 2026-05-22 | IPv6 support, JSON logging, periodic refresh, health monitoring |
| 1.1.0 | 2025-01-XX | Auto server selection, fallback connection, raw URL support |
| 1.0.0 | Initial | Basic VPN client, TUN device, protocol support |

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
