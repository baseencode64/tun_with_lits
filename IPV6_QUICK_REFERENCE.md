# IPv6 Quick Reference

## 🚀 Quick Start

### Enable IPv6 (CLI)
```bash
sudo goxray --from-raw https://example.com/links.txt --ipv6
```

### Enable IPv6 (Library)
```go
vpn, _ := client.NewClientWithOpts(client.Config{
    EnableIPv6: true,
})
```

---

## 🔍 Verification Commands

### Test IPv6 Connectivity
```bash
curl -6 https://ipv6.google.com
ping6 ipv6.google.com
curl -6 https://api64.ipify.org  # Show your IPv6
```

### Check Routes
```bash
ip -6 route show | grep tun
```

### Check Interface
```bash
ip addr show tun0 | grep inet6
```

### Monitor Traffic
```bash
tcpdump -i tun0 ip6
```

---

## 🧪 Leak Testing

Visit these sites while connected:
- https://test-ipv6.com
- https://ipleak.net
- https://dnsleaktest.com

**Expected**: VPN server's IPv6 address shown, no local leaks.

---

## ⚙️ Configuration

### Default Values
```
IPv6 Address: fd00:goxray::1/64
IPv6 Routes:  ::/1, 8000::/1
Default State: Disabled (opt-in)
```

### Linux sysctl Settings
```bash
# Enable IPv6 forwarding
sudo sysctl -w net.ipv6.conf.all.forwarding=1

# Disable privacy extensions on VPN interface
sudo sysctl -w net.ipv6.conf.tun0.use_tempaddr=0
```

---

## 🔥 Firewall Rules

### Basic IPv6 Firewall
```bash
# Allow forwarding
sudo ip6tables -A FORWARD -i tun0 -j ACCEPT
sudo ip6tables -A FORWARD -o tun0 -j ACCEPT

# NAT (if needed)
sudo ip6tables -t nat -A POSTROUTING -o eth0 -j MASQUERADE

# Kill switch (block direct IPv6)
sudo ip6tables -A OUTPUT -o eth0 -j DROP
sudo ip6tables -A OUTPUT -o tun0 -j ACCEPT
```

---

## 🐛 Troubleshooting

### Issue: "Failed to add IPv6 routes"
```bash
# Solution: Run with sudo or set capabilities
sudo goxray --from-raw url --ipv6
# OR
sudo setcap cap_net_admin,cap_net_raw+eip goxray_binary
```

### Issue: No IPv6 connectivity
```bash
# Check if VPN server supports IPv6
dig AAAA your-vpn-server.com

# Test without VPN first
curl -6 https://ipv6.google.com

# Check firewall
sudo ip6tables -L -n -v
```

### Issue: IPv6 leak detected
```bash
# Temporary fix: Disable IPv6 at OS level
sudo sysctl -w net.ipv6.conf.all.disable_ipv6=1

# Or configure proper firewall rules (see above)
```

### Clear IPv6 Routes Manually
```bash
sudo ip -6 route flush dev tun0
```

---

## 📊 Monitoring

### Watch Logs
```bash
sudo journalctl -u goxray -f | grep -i ipv6
```

### Check Status
```bash
# Connection status
ip addr show tun0

# Route table
ip -6 route show

# Active connections
ss -6 -t
```

---

## 🔄 Rollback

### Disable IPv6
```bash
# Simply remove --ipv6 flag
sudo goxray --from-raw https://example.com/links.txt
```

### Emergency Cleanup
```bash
# Stop service
sudo systemctl stop goxray

# Flush IPv6 routes
sudo ip -6 route flush dev tun0

# Restart without IPv6
sudo systemctl start goxray
```

---

## 📚 Resources

- Full Guide: [`IPv6_SUPPORT.md`](IPv6_SUPPORT.md)
- Migration: [`IPV6_MIGRATION.md`](IPV6_MIGRATION.md)
- Summary: [`IPV6_IMPLEMENTATION_SUMMARY.md`](IPV6_IMPLEMENTATION_SUMMARY.md)

---

## 💡 Tips

✅ Always test in staging first  
✅ Monitor logs after enabling  
✅ Verify no leaks with online tools  
✅ Keep firewall rules updated  
✅ Document any custom configurations  

---

**Need Help?** → https://github.com/goxray/tun/issues
