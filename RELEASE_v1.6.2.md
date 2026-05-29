# 🚀 Release v1.6.2 - Metrics Accumulation & IP Updates

**Release Date**: 2024-12-19  
**Binary**: `goxray_v1.6.2_linux_amd64` (45.12 MB)  
**Platforms**: Debian 13, Ubuntu 24.04  
**Status**: ✅ Production Ready

---

## 📋 Summary

This release fixes critical issues with Prometheus metrics when VPN clients experience failover or reconnection scenarios. Metrics now correctly accumulate traffic data across all connections and properly reflect server IP changes.

---

## 🔧 What's Fixed

### 1. ✅ Metrics Byte Accumulation

**Problem**: `vpn_bytes_read_total` and `vpn_bytes_written_total` were reset to 0 during reconnection, losing traffic history.

**Solution**:

- Added cumulative byte counters to the Client structure
- Bytes are preserved when disconnecting (added to cumulative counter)
- Metrics calculation now sums: `cumulative bytes + current connection bytes`
- Result: Metrics now show **total traffic from client startup**, never reset

**Example**:

```
Before reconnect: vpn_bytes_read_total = 1,000,000,000 bytes (1 GB)
Server fails → Failover triggered
After reconnect: vpn_bytes_read_total = 1,000,000,000+ (continues from 1 GB)
```

### 2. ✅ Server IP Metric Updates

**Problem**: `vpn_server_ip` metric was not updated when connecting to a new server during failover.

**Solution**:

- Explicit metric reset and update in `Connect()` method
- Metric now updates immediately when connection established
- All TUN interface IP metrics also updated

**Example**:

```
Before failover: vpn_server_ip{ip_address="1.2.3.4"} = 1
Server fails → Reconnecting to different server
After failover: vpn_server_ip{ip_address="5.6.7.8"} = 1  ✅ Updated!
```

---

## 📊 Affected Metrics

| Metric                    | Behavior    | Before             | After              |
| ------------------------- | ----------- | ------------------ | ------------------ |
| `vpn_bytes_read_total`    | Cumulative  | Reset to 0 ❌      | Accumulates ✅     |
| `vpn_bytes_written_total` | Cumulative  | Reset to 0 ❌      | Accumulates ✅     |
| `vpn_server_ip`           | IP tracking | Stale ❌           | Updates ✅         |
| `vpn_tun_ipv4`            | IP tracking | Stale ❌           | Updates ✅         |
| `vpn_tun_ipv6`            | IP tracking | Stale ❌           | Updates ✅         |
| `vpn_connections_total`   | Increment   | Works correctly ✅ | Works correctly ✅ |

---

## 🔍 Technical Details

### Code Changes (pkg/client/client.go)

1. **New fields in Client struct**:

   ```go
   cumulativeBytesRead    int64  // Preserved across reconnections
   cumulativeBytesWritten int64  // Preserved across reconnections
   ```

2. **Disconnect() method** - Preserves bytes:

   ```go
   currentRead := int64(c.BytesRead())
   atomic.AddInt64(&c.cumulativeBytesRead, currentRead)  // Add to cumulative
   ```

3. **Connect() method** - Updates IP metrics:

   ```go
   vpnServerIP.Reset()
   vpnServerIP.WithLabelValues(serverIP).Set(1)  // Set new IP
   ```

4. **Metrics update loop** - Sums cumulative:
   ```go
   totalRead := atomic.LoadInt64(&c.cumulativeBytesRead) + int64(c.BytesRead())
   ```

### Files Modified

- `pkg/client/client.go` (+30 lines): Cumulative tracking logic

---

## 🧪 Testing Checklist

### Manual Test 1: Bytes Accumulation

```bash
# Terminal 1: Start VPN
sudo ./goxray_v1.6.2_linux_amd64 --from-raw "vless://..." --metrics-port 9090

# Terminal 2: Monitor
watch -n 1 'curl -s http://localhost:9090/metrics | grep vpn_bytes'

# Send traffic, note bytes value
# Simulate failover (stop server or wait for health check fail)
# After reconnection, bytes should NOT reset
```

### Manual Test 2: IP Update

```bash
# Monitor IP before failover
curl -s http://localhost:9090/metrics | grep 'vpn_server_ip{ip'
# Example: vpn_server_ip{ip_address="1.2.3.4"} 1

# Trigger failover...

# Check after reconnection
curl -s http://localhost:9090/metrics | grep 'vpn_server_ip{ip'
# Should show NEW IP: vpn_server_ip{ip_address="5.6.7.8"} 1
```

### Manual Test 3: Metrics Availability

```bash
# Verify metrics endpoint stays available during reconnection
while true; do
  curl -s http://localhost:9090/metrics > /dev/null && echo "✅ $(date)" || echo "❌ $(date)"
  sleep 1
done

# Should show only ✅ (no connectivity errors during reconnection)
```

---

## 📝 Migration Guide

### For existing users:

No configuration changes required. Simply replace your binary:

```bash
# Backup old version
sudo mv /usr/local/bin/goxray /usr/local/bin/goxray.v1.6.1

# Install v1.6.2
sudo cp goxray_v1.6.2_linux_amd64 /usr/local/bin/goxray
sudo chmod +x /usr/local/bin/goxray

# Restart service
sudo systemctl restart goxray
```

### For Docker users:

```dockerfile
FROM ubuntu:24.04
COPY goxray_v1.6.2_linux_amd64 /usr/local/bin/goxray
RUN chmod +x /usr/local/bin/goxray
```

---

## 🔐 Security

- ✅ No changes to cryptographic operations
- ✅ Metrics endpoint still subject to firewall rules
- ✅ No additional network exposure
- ✅ Atomic operations ensure thread-safety

---

## 📊 Prometheus Integration

### Updated Dashboard Recommendations

For monitoring, use these queries:

```promql
# Total traffic across all sessions
rate(vpn_bytes_read_total[5m])          # Read rate in bytes/sec
rate(vpn_bytes_written_total[5m])       # Write rate in bytes/sec

# Server availability tracking
vpn_server_ip                            # Current server IP
vpn_connections_total                    # Total reconnection count

# Connection quality
vpn_connection_duration                  # Seconds connected
increase(vpn_disconnections_total[1h])   # Disconnections per hour
```

---

## 🐛 Known Issues

None. All critical metrics issues from v1.6.0 and v1.6.1 are resolved.

---

## 📚 Related Issues

- **v1.6.1**: Fixed metrics server restart on reconnection
- **v1.6.0**: Initial Prometheus metrics implementation
- **v1.6.2** ← You are here: Fixed metrics accumulation and IP updates

---

## 📦 Installation

### Debian 13 / Ubuntu 24.04

```bash
# Download
wget https://github.com/your-repo/releases/download/v1.6.2/goxray_v1.6.2_linux_amd64

# Install
sudo install -m755 goxray_v1.6.2_linux_amd64 /usr/local/bin/goxray

# Run
goxray --config /etc/goxray/config.yaml --metrics-port 9090
```

### From source

```bash
git clone <repo>
cd gotun_with_raw
git checkout v1.6.2
go build -o goxray
```

---

## 📞 Support

For issues or questions:

- Check logs: `journalctl -u goxray -f`
- Monitor metrics: `curl http://localhost:9090/metrics`
- Review documentation: See METRICS_ACCUMULATION_FIX.md

---

## ✅ Changelog

### v1.6.2 (2024-12-19)

- **FIX**: Make metrics accumulative across reconnections
- **FIX**: Update server IP metrics on new connection
- **ADD**: Cumulative byte tracking to preserve traffic history
- **ADD**: Enhanced logging for metrics state transitions

### v1.6.1 (2024-12-18)

- **FIX**: Restart Prometheus metrics server on reconnection

### v1.6.0

- **ADD**: Initial Prometheus metrics implementation

---

**Ready to deploy! 🚀**
