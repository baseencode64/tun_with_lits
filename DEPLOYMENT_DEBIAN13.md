# 📦 Deployment Guide - Debian 13 amd64

## Binary Information

**File**: `goxray_debian13_amd64`  
**Size**: 45.78 MB (45,779,387 bytes)  
**Platform**: Linux amd64 (Debian 13 compatible)  
**Build Date**: 2026-05-22 14:17:24  
**SHA256**: `0C523C972B1FC5A6BECB5D0EC4FF9BEEA06049125C4D378F1A1C3B4A596C7BA4`

### Features Included
✅ Full XRay VPN client functionality  
✅ IPv6 support (via `--ipv6` flag)  
✅ Health monitoring & automatic failover  
✅ Server selection from raw lists  
✅ Periodic server list refresh  

---

## 🚀 Quick Installation

### Option 1: Direct Usage (Recommended for testing)

```bash
# 1. Transfer binary to Debian 13 server
scp goxray_debian13_amd64 user@debian-server:/tmp/

# 2. SSH into server
ssh user@debian-server

# 3. Verify checksum
echo "0C523C972B1FC5A6BECB5D0EC4FF9BEEA06049125C4D378F1A1C3B4A596C7BA4  /tmp/goxray_debian13_amd64" | sha256sum -c

# 4. Make executable and move to system path
chmod +x /tmp/goxray_debian13_amd64
sudo mv /tmp/goxray_debian13_amd64 /usr/local/bin/goxray

# 5. Set capabilities (avoid running as root)
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# 6. Test it works
goxray --help
```

### Option 2: System Service Installation

```bash
# 1. Install binary (steps 1-5 from Option 1)

# 2. Create systemd service file
sudo tee /etc/systemd/system/goxray.service > /dev/null <<'EOF'
[Unit]
Description=GoXRay VPN Client
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/goxray --from-raw https://your-server.com/links.txt --ipv6
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=goxray

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/goxray

[Install]
WantedBy=multi-user.target
EOF

# 3. Reload systemd
sudo systemctl daemon-reload

# 4. Enable service (auto-start on boot)
sudo systemctl enable goxray

# 5. Start service
sudo systemctl start goxray

# 6. Check status
sudo systemctl status goxray

# 7. View logs
sudo journalctl -u goxray -f
```

---

## ⚙️ Configuration

### Basic Usage (IPv4 only)

```bash
sudo goxray vless://your-vless-link-here
```

### With IPv6 Support

```bash
sudo goxray --from-raw https://example.com/links.txt --ipv6
```

### Advanced Options

```bash
sudo goxray \
  --from-raw https://example.com/links.txt \
  --ipv6 \
  --refresh-interval 10m \
  --max-servers 15 \
  --timeout 5s
```

### Environment Variable Mode

```bash
export GOXRAY_CONFIG_URL="vless://your-link-here"
sudo goxray
```

---

## 🔍 Verification

### Check if Running

```bash
# Process check
ps aux | grep goxray

# Service status (if using systemd)
sudo systemctl status goxray

# Network interfaces
ip addr show tun0

# Routing table
ip route show | grep tun0
ip -6 route show | grep tun0  # If IPv6 enabled
```

### Test Connectivity

```bash
# IPv4 test
curl -4 https://api.ipify.org

# IPv6 test (if --ipv6 flag used)
curl -6 https://api64.ipify.org

# General connectivity
ping -c 3 8.8.8.8
curl https://google.com
```

### Check Logs

```bash
# Real-time logs
sudo journalctl -u goxray -f

# Last 100 lines
sudo journalctl -u goxray -n 100

# Error logs only
sudo journalctl -u goxray -p err

# IPv6 specific logs
sudo journalctl -u goxray -f | grep -i ipv6
```

---

## 🛡️ Security Considerations

### Capabilities vs Root

**Recommended**: Use Linux capabilities instead of running as root

```bash
# Set required capabilities
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# Verify capabilities
getcap /usr/local/bin/goxray
```

**Benefits**:
- ✅ Reduced attack surface
- ✅ Principle of least privilege
- ✅ Better security posture

### Firewall Configuration

If using firewall, allow VPN traffic:

```bash
# For UFW
sudo ufw allow out on tun0

# For iptables
sudo iptables -A OUTPUT -o tun0 -j ACCEPT
sudo ip6tables -A OUTPUT -o tun0 -j ACCEPT  # If IPv6 enabled
```

### IPv6 Privacy Extensions

Disable privacy extensions on VPN interface:

```bash
# Temporary
sudo sysctl -w net.ipv6.conf.tun0.use_tempaddr=0

# Permanent - add to /etc/sysctl.d/99-goxray.conf
echo "net.ipv6.conf.tun0.use_tempaddr=0" | sudo tee /etc/sysctl.d/99-goxray.conf
sudo sysctl -p /etc/sysctl.d/99-goxray.conf
```

---

## 🐛 Troubleshooting

### Issue: Permission Denied

**Symptom**: `operation not permitted` or `permission denied`

**Solution**:
```bash
# Option 1: Set capabilities (recommended)
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# Option 2: Run with sudo (less secure)
sudo goxray --from-raw url
```

### Issue: TUN Interface Not Created

**Symptom**: `create tun: operation not permitted`

**Solution**:
```bash
# Check if TUN module is loaded
lsmod | grep tun

# Load if missing
sudo modprobe tun

# Ensure device exists
sudo mkdir -p /dev/net
sudo mknod /dev/net/tun c 10 200
sudo chmod 600 /dev/net/tun
```

### Issue: Routes Not Added

**Symptom**: `add route: operation not permitted`

**Solution**:
```bash
# Check capabilities
getcap /usr/local/bin/goxray

# Re-set if needed
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
```

### Issue: Service Won't Start

**Symptom**: `systemctl start goxray` fails

**Solution**:
```bash
# Check detailed error
sudo systemctl status goxray
sudo journalctl -u goxray -n 50 --no-pager

# Common fixes:
# 1. Verify binary path in service file
# 2. Check capabilities are set
# 3. Ensure network is online
sudo systemctl restart systemd-networkd
```

### Issue: IPv6 Not Working

**Symptom**: IPv6 sites timeout after enabling `--ipv6`

**Solution**:
```bash
# 1. Check if IPv6 routes exist
ip -6 route show | grep tun0

# 2. Verify kernel IPv6 support
cat /proc/sys/net/ipv6/conf/all/disable_ipv6

# 3. Enable if disabled
sudo sysctl -w net.ipv6.conf.all.disable_ipv6=0

# 4. Check firewall
sudo ip6tables -L -n -v

# 5. Test without VPN first
curl -6 https://ipv6.google.com  # Should work before enabling VPN
```

---

## 🔄 Updates

### Manual Update

```bash
# 1. Stop service
sudo systemctl stop goxray

# 2. Backup current binary
sudo cp /usr/local/bin/goxray /usr/local/bin/goxray.bak

# 3. Install new binary
sudo cp goxray_debian13_amd64 /usr/local/bin/goxray
sudo chmod +x /usr/local/bin/goxray
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# 4. Restart service
sudo systemctl start goxray

# 5. Verify
sudo systemctl status goxray
```

### Rollback

```bash
# If new version has issues
sudo systemctl stop goxray
sudo cp /usr/local/bin/goxray.bak /usr/local/bin/goxray
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
sudo systemctl start goxray
```

---

## 📊 Monitoring

### Resource Usage

```bash
# Memory and CPU
top -p $(pgrep goxray)

# Network usage
iftop -i tun0

# Connection stats
ss -tunp | grep goxray
```

### Health Status

The application logs health status every 30 seconds:

```bash
sudo journalctl -u goxray -f | grep "Health Status"
```

### Metrics (Future Enhancement)

When Prometheus metrics endpoint is implemented:

```bash
# Access metrics
curl http://localhost:9090/metrics

# Grafana dashboard integration
# (Configuration to be added)
```

---

## 📝 Maintenance

### Log Rotation

Configure log rotation to prevent disk space issues:

```bash
sudo tee /etc/logrotate.d/goxray > /dev/null <<'EOF'
/var/log/goxray/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 root adm
    postrotate
        systemctl reload goxray > /dev/null 2>&1 || true
    endscript
}
EOF
```

### Automatic Restarts

Systemd already handles automatic restarts:

```ini
# In service file
Restart=always
RestartSec=10
```

### Backup Configuration

```bash
# Backup service configuration
sudo cp /etc/systemd/system/goxray.service /backup/

# Backup any config files
sudo cp -r /etc/goxray/ /backup/ 2>/dev/null || true
```

---

## 🎯 Best Practices

### Production Deployment Checklist

- [ ] Verify SHA256 checksum
- [ ] Set Linux capabilities (don't run as root)
- [ ] Configure systemd service
- [ ] Enable automatic restarts
- [ ] Set up log rotation
- [ ] Configure firewall rules
- [ ] Test IPv6 if enabled
- [ ] Monitor resource usage
- [ ] Document custom configurations
- [ ] Set up alerting (optional)

### Performance Tuning

For high-traffic scenarios:

```bash
# Increase TUN buffer size (in code, requires rebuild)
# Current: 1500 MTU

# Optimize TCP stack
sudo sysctl -w net.core.rmem_max=16777216
sudo sysctl -w net.core.wmem_max=16777216
sudo sysctl -w net.ipv4.tcp_rmem="4096 87380 16777216"
sudo sysctl -w net.ipv4.tcp_wmem="4096 65536 16777216"
```

---

## 📞 Support

### Documentation
- IPv6 Guide: [`IPv6_SUPPORT.md`](IPv6_SUPPORT.md)
- Quick Reference: [`IPV6_QUICK_REFERENCE.md`](IPV6_QUICK_REFERENCE.md)
- Migration Guide: [`IPV6_MIGRATION.md`](IPV6_MIGRATION.md)

### Resources
- GitHub Issues: https://github.com/goxray/tun/issues
- Project README: [`README.md`](README.md)

### Community
- Report bugs via GitHub Issues
- Include logs and system information
- Specify Debian version and kernel: `uname -a`

---

## 🎉 Summary

✅ **Binary ready**: `goxray_debian13_amd64` (45.78 MB)  
✅ **Platform**: Linux amd64 (Debian 13)  
✅ **Features**: Full VPN + IPv6 + Health monitoring  
✅ **Security**: Capability-based (no root required)  
✅ **Deployment**: Simple copy + setcap + optional systemd  

**Ready for production deployment!** 🚀
