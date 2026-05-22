# ✅ IPv6 Routes Fix - Quick Verification

## Problem
IPv6 routes were not appearing when using `--ipv6` flag.

## Solution
Binary updated to use system `ip` commands for IPv6 configuration.

---

## 🚀 Quick Test (5 minutes)

### 1. Install Fixed Binary

```bash
sudo cp goxray_debian13_amd64_ipv6_fix /usr/local/bin/goxray
sudo chmod +x /usr/local/bin/goxray
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
```

### 2. Start VPN with IPv6

```bash
sudo goxray --from-raw https://example.com/links.txt --ipv6
```

### 3. Check IPv4 Routes (should work before and after fix)

```bash
ip route show | grep tun0
```

**Expected:**
```
0.0.0.0/1 dev tun0 metric 1
128.0.0.0/1 dev tun0 metric 1
```

### 4. Check IPv6 Routes ← THE KEY TEST

```bash
ip -6 route show | grep tun0
```

**Before Fix:**
```
(empty - nothing shows)
```

**After Fix:**
```
::/1 dev tun0 metric 1024 pref medium
8000::/1 dev tun0 metric 1024 pref medium
```

### 5. Check IPv6 Address

```bash
ip -6 addr show tun0
```

**Expected:**
```
inet6 fd00:goxray::1/64 scope global
   valid_lft forever preferred_lft forever
```

### 6. Test IPv6 Connectivity

```bash
curl -6 https://ipv6.google.com
curl -6 https://api64.ipify.org
```

---

## 📊 Summary

| Check | Before Fix | After Fix |
|-------|-----------|-----------|
| IPv4 routes | ✅ Works | ✅ Works |
| IPv6 routes | ❌ Missing | ✅ Present |
| IPv6 address | ❌ Not configured | ✅ Configured |
| IPv6 connectivity | ❌ Broken | ✅ Working |

---

## 🔍 If Still Not Working

```bash
# 1. Check if ip command exists
which ip

# 2. If missing, install
sudo apt install iproute2

# 3. Check kernel IPv6 support
cat /proc/sys/net/ipv6/conf/all/disable_ipv6
# Should be: 0

# 4. Enable if disabled
sudo sysctl -w net.ipv6.conf.all.disable_ipv6=0

# 5. Check logs
sudo journalctl -u goxray -f | grep -i ipv6

# 6. Manual test
sudo ip -6 route add ::/1 dev tun0
sudo ip -6 route add 8000::/1 dev tun0
ip -6 route show | grep tun0
```

---

## 📝 Full Documentation

See [`IPV6_ROUTES_FIX.md`](IPV6_ROUTES_FIX.md) for complete details.
