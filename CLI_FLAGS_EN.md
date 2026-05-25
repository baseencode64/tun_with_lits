# GoXRay VPN Client - Complete CLI Flags & Configuration Guide

## Overview

GoXRay VPN Client supports running via command-line arguments as well as YAML configuration files. CLI arguments override values from the configuration file.

## Launch Methods

### 1. Direct Server Link

```bash
sudo goxray "vless://uuid@server.example.com:443?type=tcp&security=reality&..."
```

### 2. Configuration File

```bash
sudo goxray --config /path/to/config.yaml
```

### 3. Server List (raw URL)

```bash
sudo goxray --from-raw https://example.com/links.txt
```

### 4. Environment Variable

```bash
export GOXRAY_CONFIG_URL="vless://uuid@server.example.com:443?..."
sudo goxray
```

---

## All Available Flags

### 🔗 Connection Parameters

#### Direct Link

```bash
<vless://...>  # Positional argument - direct VLESS link
```

**Example:**

```bash
sudo goxray "vless://a6b071ef-0d82-4f46-b04b-3310b8d6ca82@3.112.126.206:54055?type=tcp&security=reality&pbk=..."
```

#### Configuration File

```bash
--config <path>  # Path to YAML configuration file
```

**Example:**

```bash
sudo goxray --config /etc/goxray/config.yaml
```

#### Server List from URL

```bash
--from-raw <url>  # URL for VLESS links list
```

**Example:**

```bash
sudo goxray --from-raw https://raw.githubusercontent.com/user/repo/main/links.txt
```

---

### 🔄 Server List Refresh Parameters

#### Refresh Interval

```bash
--refresh-interval <duration>  # Periodic server list refresh
```

**Format:** `5m`, `10m`, `1h`, `30m` etc.

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 10m
```

**Default:** `0` (refresh disabled)

#### Maximum Servers

```bash
--max-servers <n>  # Maximum number of servers to check
```

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --max-servers 20
```

**Default:** `10`

---

### 🔗 Reconnection Parameters (Connection Persistence & Auto-Reconnect)

Controls client behavior when all servers in the list are exhausted. The client automatically enters a reconnection loop with exponential backoff.

#### Maximum Retries

```bash
--max-retries <n>  # Maximum reconnection attempts
```

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --max-retries 5
```

**Default:** `0` (unlimited retries, until stopped via Ctrl+C)

#### Minimum Backoff

```bash
--min-backoff <duration>  # Initial backoff before reconnection
```

**Format:** `5s`, `10s`, `30s` etc.

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --min-backoff 10s
```

**Default:** `5s`

#### Maximum Backoff

```bash
--max-backoff <duration>  # Maximum backoff before reconnection
```

**Format:** `5m`, `10m`, `30m` etc.

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --max-backoff 10m
```

**Default:** `5m`

#### Backoff Factor

```bash
--backoff-factor <factor>  # Exponential backoff multiplier
```

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --backoff-factor 3.0
```

**Default:** `2.0`

**Formula:**

```
backoff(n) = min(min_backoff × factor^(n-1) + jitter, max_backoff)
```

**Default sequence:**

```
Attempt 1: 5s    (min_backoff)
Attempt 2: 10s   (5s × 2.0)
Attempt 3: 20s   (10s × 2.0)
Attempt 4: 40s   (20s × 2.0)
Attempt 5: 80s   (40s × 2.0)
Attempt 6: 160s  (80s × 2.0)
Attempt 7: 5m    (capped at max_backoff)
...
```

_Actual time may vary by ±25% due to jitter for load distribution._

---

### ⏱️ Timeouts

#### Server Check Timeout

```bash
--timeout <duration>  # Timeout for each server health check
```

**Format:** `5s`, `10s`, `30s` etc.

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --timeout 10s
```

**Default:** `5s`

---

### 🌐 Network Settings

#### IPv6 Support

```bash
--ipv6  # Enable IPv6 dual-stack support
```

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --ipv6
```

**Default:** `false`

**What is enabled:**

- IPv6 address on TUN interface: `fd00:dead:beef::1/64`
- IPv6 traffic routing through VPN
- IPv6 DNS server support

#### DNS Leak Protection

```bash
--dns-protection  # Enable DNS leak protection
```

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --dns-protection
```

**Default:** `false`

**What is enabled:**

- DNS traffic routing through TUN interface
- Routes to public DNS servers (Google, Cloudflare, Quad9)
- Both IPv4 and IPv6 DNS support

---

### 📊 Prometheus Metrics

#### Metrics Port

```bash
--metrics-port <port>  # Enable Prometheus metrics endpoint
```

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --metrics-port 9090
```

**Default:** `0` (disabled)

**Available metrics:**

- `vpn_connections_total` - Total connections
- `vpn_disconnections_total` - Total disconnections
- `vpn_connection_duration_seconds` - Current connection duration
- `vpn_bytes_read_total` - Total bytes read
- `vpn_bytes_written_total` - Total bytes written
- `vpn_connected` - Connection status (1=connected, 0=disconnected)
- `vpn_tun_ipv4` - TUN interface IPv4 address
- `vpn_tun_ipv6` - TUN interface IPv6 address
- `vpn_server_ip` - VPN server IP address

**Endpoint:** `http://0.0.0.0:9090/metrics`

---

### 📝 Logging Settings

#### Log Format

```bash
--log-format <format>  # Output log format
```

**Options:** `text`, `json`

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --log-format json
```

**Default:** `text`

#### Log Level

```bash
--log-level <level>  # Log verbosity level
```

**Options:** `debug`, `info`, `warn`, `error`

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --log-level debug
```

**Default:** `info`

#### Log File

```bash
--log-file <path>  # Path to log file
```

**Example:**

```bash
sudo goxray --from-raw https://example.com/links.txt --log-file /var/log/goxray/goxray.log
```

**Default:** (stdout only)

#### Log File Size

```bash
--log-max-size <MB>  # Maximum log file size before rotation
```

**Example:**

```bash
sudo goxray --log-file /var/log/goxray/goxray.log --log-max-size 200
```

**Default:** `100` MB

#### Backup Count

```bash
--log-max-backups <count>  # Maximum number of backup log files
```

**Example:**

```bash
sudo goxray --log-file /var/log/goxray/goxray.log --log-max-backups 5
```

**Default:** `3`

#### Backup Age

```bash
--log-max-age <days>  # Maximum age of backup log files in days
```

**Example:**

```bash
sudo goxray --log-file /var/log/goxray/goxray.log --log-max-age 30
```

**Default:** `28` days

---

## Configuration File (YAML)

All parameters can be specified in a YAML file:

```yaml
# connection - connection settings
connection:
  link: "vless://uuid@server.example.com:443?type=tcp&security=reality&..."

  # Single raw URL (legacy, use from_raw_urls)
  # from_raw: "https://example.com/links.txt"

  # Multiple URLs with fallback support
  from_raw_urls:
    - "https://primary.example.com/links.txt"
    - "https://backup1.example.com/links.txt"
    - "https://backup2.example.com/links.txt"

  enable_ipv6: false
  enable_dns_protection: false
  tls_allow_insecure: false
  metrics_port: 9090

# server_selection - server selection settings
server_selection:
  refresh_interval: "10m"
  max_servers: 10
  timeout: "5s"

# reconnection - reconnection settings (persistence & auto-reconnect)
reconnection:
  max_retries: 0
  min_backoff: "5s"
  max_backoff: "5m"
  backoff_factor: 2.0

# logging - logging settings
logging:
  format: "text"
  level: "info"
  # file: "/var/log/goxray/goxray.log"
  max_size: 100
  max_backups: 3
  max_age: 28

# health_monitoring - health monitoring settings
health_monitoring:
  check_interval: "10s"
  timeout: "5s"
  max_retries: 3
```

### Parameter Priority

1. **CLI arguments** (highest priority)
2. **Configuration file**
3. **Environment variables**
4. **Default values** (lowest priority)

---

## Usage Examples

### Example 1: Simple connection

```bash
sudo goxray "vless://uuid@server:443?type=tcp&security=reality&..."
```

### Example 2: Server list with auto-selection

```bash
sudo goxray --from-raw https://example.com/links.txt
```

### Example 2a: With auto-reconnect

```bash
sudo goxray \
  --from-raw https://example.com/links.txt \
  --max-retries 10 \
  --min-backoff 5s \
  --max-backoff 10m \
  --backoff-factor 2.0
```

**What happens:**

1. Loads server list
2. On connection failure → waits 5s
3. Re-fetches server list
4. Retries connection
5. On failure → waits 10s → 20s → 40s ... (up to 10m)
6. After 10 attempts → exit with error
7. Ctrl+C at any time → graceful shutdown

### Example 3: Full configuration with logging

```bash
sudo goxray \
  --from-raw https://example.com/links.txt \
  --refresh-interval 10m \
  --max-servers 20 \
  --timeout 10s \
  --ipv6 \
  --dns-protection \
  --metrics-port 9090 \
  --log-format json \
  --log-level info \
  --log-file /var/log/goxray/goxray.log \
  --log-max-size 200 \
  --log-max-backups 5 \
  --log-max-age 30
```

### Example 4: Using configuration file

```bash
# Create config file
cat > /etc/goxray/config.yaml << EOF
connection:
  from_raw_urls:
    - "https://primary.example.com/links.txt"
    - "https://backup.example.com/links.txt"
  enable_ipv6: true
  enable_dns_protection: true
  metrics_port: 9090

server_selection:
  refresh_interval: "10m"
  max_servers: 15
  timeout: "10s"

reconnection:
  max_retries: 0
  min_backoff: "5s"
  max_backoff: "5m"
  backoff_factor: 2.0

logging:
  format: "json"
  level: "info"
  file: "/var/log/goxray/goxray.log"
  max_size: 200
  max_backups: 5
  max_age: 30

health_monitoring:
  check_interval: "10s"
  timeout: "5s"
  max_retries: 3
EOF

# Run with config
sudo goxray --config /etc/goxray/config.yaml
```

---

## Environment Variables

### GOXRAY_CONFIG_URL

```bash
export GOXRAY_CONFIG_URL="vless://uuid@server:443?..."
sudo goxray  # Uses link from environment variable
```

---

## Important Notes

### Launch Requirements

1. **sudo** - required for network interface operations
2. **Linux capabilities** (optional):
   ```bash
   sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
   ```

### Supported OS

- **Linux** (tested on Ubuntu 24.10, Debian 13)
- **macOS** (tested on Sequoia 15.1.1)

### Limitations

- Maximum 10 concurrent server checks (configurable via `--max-servers`)
- Default check timeout is 5 seconds
- Maximum log file size is 100MB (configurable)

---

## Troubleshooting

### Check available flags

```bash
sudo goxray --help
```

### Enable debug logging

```bash
sudo goxray --from-raw https://example.com/links.txt --log-level debug
```

### View Prometheus metrics

```bash
curl http://localhost:9090/metrics
```

### Check connection status

```bash
# Logs show every 30 seconds:
# VPN Connection Status: connected, tun_interface=tun0, xray_server=3.112.126.206
```

---

## Additional Resources

- [README.md](README.md) - General information and examples
- [HEALTH_MONITORING.md](HEALTH_MONITORING.md) - Health monitoring system details
- [PERIODIC_REFRESH.md](PERIODIC_REFRESH.md) - Periodic refresh settings
- [DEPLOYMENT_DEBIAN13.md](DEPLOYMENT_DEBIAN13.md) - Deployment instructions
