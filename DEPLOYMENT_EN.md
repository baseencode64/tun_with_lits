# 📦 Build and Installation of GoXRay on Debian 13

## ✅ Ready Binary File

**File**: `goxray_v1.6.0_linux_amd64`  
**Size**: ~47.3 MB  
**Architecture**: Linux amd64 (x86_64)  
**Version**: v1.6.0 with auto-reconnect  
**Status**: ✅ Compiled and ready to use

---

## 🚀 Quick Start

### Method 1: Simple Copy

```bash
# 1. Copy goxray_v1.6.0_linux_amd64 to Debian server
scp goxray_v1.6.0_linux_amd64 user@debian:/usr/local/bin/goxray

# 2. On Debian server:
ssh user@debian

# Make executable
sudo chmod +x /usr/local/bin/goxray

# Set capabilities
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# Run with auto-reconnect
sudo goxray --from-raw https://example.com/links.txt
```

### Method 2: System Service Installation

```bash
# 1. Copy binary and set capabilities (as above)

# 2. Create service file
sudo tee /etc/systemd/system/goxray.service > /dev/null <<'EOF'
[Unit]
Description=GoXRay VPN Client v1.6.0
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/goxray --from-raw https://your-server.com/links.txt --max-retries 0
Restart=on-failure
RestartSec=10
Capabilities=CAP_NET_RAW,CAP_NET_ADMIN,CAP_NET_BIND_SERVICE+eip
AmbientCapabilities=CAP_NET_RAW,CAP_NET_ADMIN,CAP_NET_BIND_SERVICE

NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/tmp

[Install]
WantedBy=multi-user.target
EOF

# 3. Enable and start
sudo systemctl daemon-reload
sudo systemctl enable goxray
sudo systemctl start goxray

# 4. Check status
sudo systemctl status goxray

# 5. View logs
sudo journalctl -u goxray -f
```

---

## 📋 Configuration

### Basic Usage

```bash
# Direct VLESS link
sudo goxray vless://your-vless-link-here

# Server list with auto-selection
sudo goxray --from-raw https://example.com/links.txt

# With auto-reconnect (infinite retries)
sudo goxray --from-raw https://example.com/links.txt --max-retries 0

# With IPv6
sudo goxray --from-raw https://example.com/links.txt --ipv6

# Full configuration
sudo goxray \
  --from-raw https://example.com/links.txt \
  --refresh-interval 10m \
  --max-servers 15 \
  --timeout 5s \
  --ipv6 \
  --dns-protection \
  --max-retries 0 \
  --metrics-port 9090 \
  --log-format json \
  --log-file /var/log/goxray/goxray.log
```

### YAML Configuration File

Create `/etc/goxray/config.yaml`:

```yaml
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
  timeout: "5s"

reconnection:
  max_retries: 0
  min_backoff: "5s"
  max_backoff: "5m"
  backoff_factor: 2.0

logging:
  format: "json"
  level: "info"
  file: "/var/log/goxray/goxray.log"
  max_size: 100
  max_backups: 3
  max_age: 28

health_monitoring:
  check_interval: "10s"
  timeout: "5s"
  max_retries: 3
```

Run:

```bash
sudo goxray --config /etc/goxray/config.yaml
```

---

## 🔧 System Requirements

### Dependencies

```bash
sudo apt update
sudo apt install -y iproute2 iputils-ping curl ca-certificates

# Check TUN support
sudo modprobe tun
lsmod | grep tun
```

### Permissions

Option 1 (recommended) — capabilities:

```bash
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
```

Option 2 — sudo:

```bash
sudo goxray --from-raw https://example.com/links.txt
```

---

## 📊 Monitoring

### Prometheus Metrics

```bash
# Enable metrics on port 9090
sudo goxray --from-raw https://example.com/links.txt --metrics-port 9090

# View metrics
curl http://localhost:9090/metrics
```

### Logging

```bash
# JSON format with rotation
sudo goxray \
  --from-raw https://example.com/links.txt \
  --log-format json \
  --log-file /var/log/goxray/goxray.log

# View service logs
sudo journalctl -u goxray -f
```

### Health Status

Health status is logged every 30 seconds:

```
INFO VPN Health Status status={"connected":true,"current_server_idx":0,...}
```

---

## 🔍 Diagnostics

### Connection Check

```bash
# Check routes
ip route show
ip -6 route show

# Check DNS
ping -c 4 google.com
nslookup google.com

# Check external IP
curl -s https://api.ipify.org
```

### Log Filtering

```bash
# All logs
sudo journalctl -u goxray -n 100

# Health check
sudo journalctl -u goxray | grep "Health check"

# Failover
sudo journalctl -u goxray | grep "failover\|Failover"

# Reconnection
sudo journalctl -u goxray | grep "Reconnection\|Reconnector"
```

---

## 🐳 Docker

Build Docker image:

```dockerfile
FROM debian:13-slim
COPY goxray_v1.6.0_linux_amd64 /usr/local/bin/goxray
RUN chmod +x /usr/local/bin/goxray && \
    apt update && apt install -y --no-install-recommends \
    iproute2 iptables ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
ENTRYPOINT ["goxray"]
CMD ["--help"]
```

---

## 🔒 Security

1. **Always use HTTPS** for server list downloads
2. **Update binary regularly**
3. **Use capabilities** instead of running as root
4. **Verify binary integrity** using SHA256 checksum
5. **Configure firewall** to restrict metrics endpoint access

---

## 📚 Additional Documentation

- [README.md](README.md) — General documentation
- [README_RU.md](README_RU.md) — Documentation in Russian
- [CLI_FLAGS_EN.md](CLI_FLAGS_EN.md) — Complete CLI reference (EN)
- [HEALTH_MONITORING_EN.md](HEALTH_MONITORING_EN.md) — Health monitoring (EN)
- [PERIODIC_REFRESH_EN.md](PERIODIC_REFRESH_EN.md) — Periodic refresh (EN)
- [CHANGELOG.md](CHANGELOG.md) — Changelog (EN)
