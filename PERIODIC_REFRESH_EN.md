# 🔄 Periodic Server List Refresh

## Overview

Added the ability to **periodically refresh the server list** from a raw URL with a configurable interval. This allows automatically discovering new servers and maintaining an up-to-date list of available options.

---

## 🎯 Problem and Solution

### Problem

Previously, the server list was loaded only once at startup. If new servers appeared or the availability of existing servers changed, a manual restart of the program was required.

### Solution

Added the `--refresh-interval` flag which:

- ✅ **Periodically refreshes** the server list from raw URL
- ✅ **Discovers new servers** automatically
- ✅ **Logs changes** in the available server list
- ✅ **Runs in the background** without interrupting the VPN connection
- ✅ **Flexibly configurable** via command line

---

## 🚀 Usage

### Basic example

```bash
# Refresh every 5 minutes
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 5m

# Refresh every 10 minutes
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 10m

# Refresh every hour
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 1h
```

### All available options

```bash
sudo goxray --from-raw <URL> [options]

Options:
  --refresh-interval <duration> - Server list refresh interval (default: 0 = disabled)
                                  Format: 5m, 10m, 30m, 1h, etc.

  --max-servers <n>             - Maximum number of servers to check (default: 10)

  --timeout <duration>          - Timeout per server check (default: 5s)
                                  Format: 3s, 5s, 10s, etc.
```

---

## 📋 Usage examples

### Example 1: Standard configuration

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 10m \
  --max-servers 15 \
  --timeout 5s
```

**What happens:**

1. Server list is loaded from URL
2. First 15 servers are checked with 5s timeout each
3. Connection to best server
4. Every 10 minutes the list is refreshed
5. Health monitoring checks the server every 10s

### Example 2: Aggressive refresh (unstable networks)

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 2m \
  --max-servers 20 \
  --timeout 3s
```

**Benefits:**

- Fast new server discovery (every 2 minutes)
- More connection options (20 servers)
- Shorter timeout for faster checks

### Example 3: Traffic saving

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 1h \
  --max-servers 5 \
  --timeout 10s
```

**Benefits:**

- Rare refresh saves traffic (once per hour)
- Fewer checks = less load
- Longer timeout for reliability

---

## 🔄 How periodic refresh works

### Process

```
Program startup
    ↓
Initial server list load
    ↓
Availability check and latency sorting
    ↓
Connection to best server
    ↓
Health monitoring starts (every 10s)
    ↓
┌─────────────────────────────────────┐
│ Every N minutes (refresh-interval): │
│   ├─ Load new server list           │
│   ├─ Check availability             │
│   ├─ Compare with current list      │
│   ├─ Log changes                    │
│   └─ Continue working               │
└─────────────────────────────────────┘
    ↓
On health problems:
    ├─ Automatic failover
    ├─ Select next best server
    └─ Use current server list
```

### Log example

```
INFO Fetching server list from raw URL url=https://example.com/links.txt refresh_interval=10m0s
INFO Checking servers total=15 max_concurrent=10
INFO Found available servers total=8 sorted_by=latency
INFO Server selection results:
=== VPN Server Selection Report ===
Total servers scanned: 15
Available servers: 8

1. server1.com:443 - Latency: 45ms - ★ RECOMMENDED
2. server2.com:443 - Latency: 78ms - ✓ Available
...

INFO Attempting VPN connection with fallback support servers_count=8
INFO Successfully connected to VPN server host=server1.com port=443 latency=45ms
INFO Starting health checks host=server1.com port=443 interval=10s timeout=5s max_retries=3
INFO Periodic server list refresh enabled interval=10m0s

# ... after 10 minutes ...

INFO Refreshing server list from raw URL url=https://example.com/links.txt
INFO Checking servers total=16 max_concurrent=10
INFO Found available servers total=9 sorted_by=latency
INFO Server list refreshed successfully total_servers=9 new_servers_available=9
=== VPN Server Selection Report ===
Total servers scanned: 16
Available servers: 9

1. server1.com:443 - Latency: 45ms - ★ RECOMMENDED
2. new-server.com:443 - Latency: 65ms - ✓ Available  ← NEW SERVER!
3. server2.com:443 - Latency: 78ms - ✓ Available
...
```

---

## 💡 Usage scenarios

### Scenario 1: Stable network (rare refresh)

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 2h \
  --max-servers 10
```

**When to use:**

- Servers rarely added/removed
- Minimize traffic is important
- Basic monitoring is sufficient

### Scenario 2: Dynamic environment (frequent refresh)

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 5m \
  --max-servers 20 \
  --timeout 3s
```

**When to use:**

- Servers change frequently
- Maximum availability needed
- Traffic is not critical

### Scenario 3: Critical service (very frequent refresh)

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 1m \
  --max-servers 30 \
  --timeout 2s
```

**When to use:**

- Business-critical connection
- Servers may fail frequently
- Constant availability required

---

## 🔧 Technical details

### Interaction with Health Monitoring

Periodic refresh **works together** with health monitoring:

| Component        | Frequency       | Purpose              |
| ---------------- | --------------- | -------------------- |
| **Health Check** | Every 10s       | Check current server |
| **Failover**     | On demand       | Switch on problems   |
| **List Refresh** | Every N minutes | Update server list   |

**Interaction example:**

```
10:00:00 - Health check ✓ (server1 OK)
10:00:10 - Health check ✓ (server1 OK)
10:00:20 - Health check ✓ (server1 OK)
10:05:00 - List refresh → new-server.com discovered (latency: 50ms)
10:05:10 - Health check ✓ (server1 OK, latency: 80ms)
10:05:20 - Health check ✓ (server1 OK)
10:05:30 - Auto-failover triggered (server1 became slow)
           → switched to new-server.com (50ms)
```

---

## ⚙️ Configuration recommendations

### Optimal settings for different cases

#### For home (resource saving)

```bash
--refresh-interval 1h \
--max-servers 5 \
--timeout 10s
```

#### For office (balance)

```bash
--refresh-interval 10m \
--max-servers 15 \
--timeout 5s
```

#### For production (maximum reliability)

```bash
--refresh-interval 5m \
--max-servers 30 \
--timeout 3s
```

---

## 📊 Statistics and monitoring

### What gets logged on refresh

```
INFO Refreshing server list from raw URL url=...
INFO Checking servers total=15 max_concurrent=10
INFO Found available servers total=8 sorted_by=latency
INFO Server list refreshed successfully total_servers=8 new_servers_available=8
=== VPN Server Selection Report ===
...
```

### Key metrics

- **total_servers** - total servers in list
- **available_servers** - currently available servers
- **new_servers_available** - how many servers are available
- **latency rankings** - servers sorted by speed

---

## 🧪 Testing

### Quick test

```bash
# Refresh every minute for quick observation
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 1m

# Observe refresh logs every minute
sudo journalctl -u goxray -f | grep "refresh\|Refreshing"
```

### Verify operation

```bash
# 1. Run with short interval
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 2m

# 2. Wait 2 minutes
# 3. See "Refreshing server list..." in logs
# 4. Check that list was updated
```

---

## 🎉 Summary

✅ **Automatic refresh** - server list always up to date  
✅ **Flexible configuration** - interval from minutes to hours  
✅ **Background operation** - doesn't interfere with VPN connection  
✅ **New server discovery** - automatic detection of new servers  
✅ **Integration with health monitoring** - comprehensive reliability system  
✅ **Minimal overhead** - refresh only on timer

**Your VPN client now always knows the best servers!** 🚀
