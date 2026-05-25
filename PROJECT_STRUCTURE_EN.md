# Project Structure (v1.6.0)

```
gotun_with_raw/
│
├── main.go                       - Entry point with CLI, --from-raw, auto-reconnect
├── go.mod                        - Go module (github.com/goxray/tun)
├── go.sum                        - Go module checksums
│
├── README.md                     [EN] - Main documentation
├── README_RU.md                  [RU] - Main documentation (Russian)
├── CHANGELOG.md                  [EN] - Version changelog
├── CHANGELOG_RU.md               [RU] - Version changelog (Russian)
├── CLI_FLAGS.md                  [RU] - CLI flags reference
├── CLI_FLAGS_EN.md               [EN] - CLI flags reference
├── HEALTH_MONITORING.md          [RU] - Health monitoring system
├── HEALTH_MONITORING_EN.md       [EN] - Health monitoring system
├── PERIODIC_REFRESH.md           [RU] - Periodic refresh settings
├── PERIODIC_REFRESH_EN.md        [EN] - Periodic refresh settings
├── INSTALL_DEBIAN.md             [RU] - Debian installation guide
├── INSTALL_DEBIAN_EN.md          [EN] - Debian installation guide
├── DEPLOYMENT.md                 [RU] - Deployment guide
├── DEPLOYMENT_EN.md              [EN] - Deployment guide
├── DEPLOYMENT_DEBIAN13.md        [EN] - Debian 13 deployment
├── DEPLOYMENT_DEBIAN13_RU.md     [RU] - Debian 13 deployment
├── PROJECT_STRUCTURE.md          [RU] - Project structure
├── PROJECT_STRUCTURE_EN.md       [EN] - Project structure (this file)
│
├── config.yaml                   - YAML configuration file
├── config.yaml.example           - Example configuration file
├── example_links.txt             - Example raw VLESS links list
│
├── build.sh                      - Build script
├── install_goxray.sh             - Installation script
├── Dockerfile                    - Docker build
├── .dockerignore                 - Docker ignore rules
├── .gitignore                    - Git ignore rules
│
├── goxray_v1.6.0_linux_amd64     - Compiled binary (v1.6.0, Linux amd64)
│
└── pkg/
    └── client/
        ├── client.go             - Main VPN client (TUN, XRay, routing, DNS)
        ├── config.go             - YAML config (ConnectionConfig, ReconnectionConfig, etc.)
        ├── config_test.go        - Config tests
        ├── interfaces.go         - Interfaces (Logger, pipe, ipTable, runnable)
        ├── health_checker.go     - Health monitoring (SOCKS5 proxy check)
        ├── health_checker_test.go - Health checker tests
        ├── vpn_connector.go      - VPN connection with fallback + reconnect
        ├── vpn_connector_test.go - VPN connector tests
        ├── server_selector.go    - Server selection (latency + packet loss scoring)
        ├── server_selector_test.go - Server selector tests
        ├── reconnector.go        - [NEW v1.6.0] Auto-reconnect with exponential backoff
        ├── reconnector_test.go   - [NEW v1.6.0] Reconnector tests (12 tests)
        ├── link_parser.go        - VLESS link parser
        ├── link_parser_test.go   - Link parser tests
        ├── slog_adapter.go       - slog.Logger adapter
        │
        └── mocks/
            └── client_mocks.go   - Mock objects for testing
```

---

## 📁 File Descriptions

### Core Files

**`main.go`** — Application entry point

- CLI arguments parsing (direct link, --from-raw, --config)
- Reconnection flags: `--max-retries`, `--min-backoff`, `--max-backoff`, `--backoff-factor`
- Logger setup (text/json format with rotation via lumberjack)
- Prometheus metrics endpoint
- Initial connection → Reconnector loop on failure
- Signal handling (SIGTERM, SIGINT) for graceful shutdown

**`pkg/client/client.go`** — Core VPN client (1324+ lines)

- TUN interface setup (192.18.0.1/32 IPv4, fd00:dead:beef::1/64 IPv6)
- XRay instance management (SOCKS5 inbound proxy)
- Traffic routing through route library (goxray/core/network/route)
- IPv4/IPv6 dual-stack support
- DNS leak protection (routes to Google/Cloudflare/Quad9)
- Prometheus metrics (9 metrics)
- Traffic monitoring (/proc/net/dev parsing)

**`pkg/client/reconnector.go`** — [NEW v1.6.0] Auto-reconnect

- `Reconnector` struct with exponential backoff
- `ReconnectionConfig` (max_retries, min_backoff, max_backoff, backoff_factor)
- Jitter (±25%) for load distribution
- Refresh callback for updating server list before retry
- Graceful stop via context or Stop()

**`pkg/client/vpn_connector.go`** — Connection management

- `ConnectWithFallback()` — sequential server attempts
- `performFailover()` — automatic switch on health failure
- `NewVPNConnectorWithReconnect()` — constructor with custom reconnection config
- `ErrAllServersExhausted` — sentinel error

**`pkg/client/health_checker.go`** — Health monitoring

- SOCKS5 proxy verification (greeting → response check)
- Configurable interval (10s), timeout (5s), max_retries (3)
- Callback-based failover trigger

**`pkg/client/server_selector.go`** — Smart server selection

- Concurrent latency checking (semaphore pattern)
- Weighted scoring: latency (50%) + packet loss (50%)
- `SelectAllByScore()` — sorts by weighted score
- `SelectAllByLatency()` — sorts by latency (legacy)

**`pkg/client/config.go`** — YAML configuration

- `AppConfig` → Connection, ServerSelection, Reconnection, Logging, HealthMonitoring
- `LoadConfig()` — read, unmarshal, set defaults, validate
- Full validation of all sections

---

## 📊 Code Statistics (v1.6.0)

| Category                | Count                                     |
| ----------------------- | ----------------------------------------- |
| **Go source files**     | 15                                        |
| **Test files**          | 6                                         |
| **Test count**          | ~40+ (including 12 new reconnector tests) |
| **Documentation files** | 18 (9 EN + 9 RU)                          |
| **Binary size**         | ~47.3 MB (Linux amd64)                    |

### Key Dependencies

| Package                                  | Purpose             |
| ---------------------------------------- | ------------------- |
| `github.com/goxray/core`                 | TUN device, routing |
| `github.com/xtls/xray-core`              | XRay-core protocols |
| `github.com/xjasonlyu/tun2socks/v2`      | TUN → SOCKS5 bridge |
| `github.com/lilendian0x00/xray-knife/v3` | VLESS link parsing  |
| `github.com/prometheus/client_golang`    | Metrics             |
| `gopkg.in/natefinch/lumberjack.v2`       | Log rotation        |

---

## 🔄 Data Flow (v1.6.0)

```
System traffic → TUN Device (192.18.0.1)
                    ↓
                pipe2socks (tun2socks)
                    ↓
         SOCKS5 Proxy (127.0.0.1:XXXXX)
                    ↓
              XRay Core (Outbound)
         (VLESS/VMess/Trojan/Shadowsocks)
                    ↓
         Remote XRay Server → Internet
                    ↓
         Health Checker (every 10s)
                    ↓
         Fail on 3 errors → performFailover()
                    ↓
         All servers exhausted → Reconnector
         (exponential backoff: 5s → 10s → 20s → ... → 5m)
                    ↓
         Refresh server list → retry → Ctrl+C = exit
```
