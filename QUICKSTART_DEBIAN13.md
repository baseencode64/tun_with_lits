# 🚀 Quick Start - Debian 13

## Binary Info
- **File**: `goxray_debian13_amd64`
- **Size**: 45.78 MB
- **SHA256**: `0C523C972B1FC5A6BECB5D0EC4FF9BEEA06049125C4D378F1A1C3B4A596C7BA4`

---

## ⚡ 3-Minute Setup

### 1. Upload & Install

```bash
# From your local machine
scp goxray_debian13_amd64 user@debian-server:/tmp/

# On Debian server
ssh user@debian-server
sudo mv /tmp/goxray_debian13_amd64 /usr/local/bin/goxray
sudo chmod +x /usr/local/bin/goxray
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
```

### 2. Verify

```bash
goxray --help
```

### 3. Run

```bash
# IPv4 only
sudo goxray vless://your-link-here

# With IPv6
sudo goxray --from-raw https://example.com/links.txt --ipv6
```

---

## 🔧 System Service (Optional)

```bash
# Create service
sudo tee /etc/systemd/system/goxray.service > /dev/null <<EOF
[Unit]
Description=GoXRay VPN Client
After=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/goxray --from-raw https://example.com/links.txt --ipv6
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable goxray
sudo systemctl start goxray

# Check status
sudo systemctl status goxray
```

---

## ✅ Verification

```bash
# Check it's running
ps aux | grep goxray

# Test connectivity
curl https://api.ipify.org

# View logs
sudo journalctl -u goxray -f
```

---

## 📚 Full Documentation

- Complete guide: [`DEPLOYMENT_DEBIAN13.md`](DEPLOYMENT_DEBIAN13.md)
- IPv6 support: [`IPv6_SUPPORT.md`](IPv6_SUPPORT.md)
- Troubleshooting: See DEPLOYMENT_DEBIAN13.md section "Troubleshooting"

---

**That's it!** Your VPN is now running on Debian 13. 🎉
