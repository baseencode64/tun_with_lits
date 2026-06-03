# 🪟 Windows VPN Proxy Guide - SOCKS5 Proxy with GoXRay

**Version**: v1.7.0  
**Date**: 2026-06-03  
**Platform**: Windows 10/11, Linux, macOS

---

## 📖 Table of Contents

1. [Overview](#overview)
2. [Built-in SOCKS5 Server](#built-in-socks5-server)
3. [Configuration](#configuration)
4. [Usage](#usage)
5. [Authentication](#authentication)
6. [Security](#security)
7. [Troubleshooting](#troubleshooting)
8. [Windows-Specific Notes](#windows-specific-notes)

---

## Overview

GoXRay VPN Client now includes a **built-in SOCKS5 proxy server** that allows applications to route traffic through the VPN tunnel.

### Why do you need a SOCKS5 proxy?

Some applications may not automatically use the TUN interface created by GoXRay. This is especially relevant for:

- ✅ Applications that don't respect system routing tables
- ✅ Docker containers
- ✅ WSL2 (Windows Subsystem for Linux)
- ✅ Some browsers and download managers
- ✅ Applications requiring explicit proxy configuration

### Benefits of Built-in SOCKS5

- ✅ **No external proxy server required** (Dante, SSH tunnel, Privoxy)
- ✅ **Automatic start/stop** with VPN connection
- ✅ **Traffic automatically routed through VPN** tunnel
- ✅ **Authentication support** username/password
- ✅ **Graceful shutdown** when VPN disconnects
- ✅ **Detailed logging** of all events

---

## Built-in SOCKS5 Server

### Architecture

```
Windows/Linux/macOS Application
  ↓
SOCKS5 Proxy (localhost:1080)
  ↓
GoXRay VPN Client
  ↓
TUN Interface (tun0)
  ↓
VPN Server
  ↓
Internet
```

**Key feature:** SOCKS5 server runs **inside GoXRay** and automatically routes all traffic through the VPN tunnel.

---

## Configuration

### 1. Enable SOCKS5 in config.yaml

```yaml
# SOCKS5 proxy server (optional)
# Allows applications to route traffic through VPN via SOCKS5 proxy
# Useful for applications that don't support TUN/TAP or need explicit proxy configuration
socks5:
  # Enable SOCKS5 proxy server (default: false)
  enabled: true

  # Listen address (default: "0.0.0.0:1080")
  # Use "0.0.0.0:1080" to listen on all interfaces
  # Use "127.0.0.1:1080" to listen only on localhost
  listen_addr: "0.0.0.0:1080"

  # Authentication (optional, leave empty for no auth)
  # If both username and password are set, SOCKS5 will require authentication
  username: ""
  password: ""

  # Connection timeout (default: "30s")
  # Maximum time to wait for connection establishment
  timeout: "30s"
```

### 2. Start GoXRay VPN

```bash
# With configuration file
./goxray --config config.yaml

# Or with CLI arguments + from-raw
./goxray --from-raw https://example.com/links.txt
```

**SOCKS5 server will start automatically** after successful VPN connection!

### 3. Check logs

You will see in the logs:

```
INFO VPN client connected successfully tun_address=192.18.0.1/32 xray_server=1.2.3.4:443
INFO Starting SOCKS5 proxy server address=0.0.0.0:1080
INFO SOCKS5 proxy server started successfully address=0.0.0.0:1080 auth="no authentication" timeout=30s
```

---

## Usage

### Testing SOCKS5

#### Windows PowerShell Testing Script

GoXRay includes a PowerShell test script for Windows users:

```powershell
# Run the test script
.\test_socks5.ps1
```

**The script will:**

- ✅ Check if SOCKS5 port is listening
- ✅ Get your real IP (without proxy)
- ✅ Test SOCKS5 connection with curl.exe
- ✅ Verify SOCKS5 handshake protocol
- ✅ Compare IPs to confirm traffic goes through VPN

**Expected output:**

```
=== GoXRay SOCKS5 Proxy Test ===

[1/4] Checking if SOCKS5 port is listening...
✓ SOCKS5 port 1080 is open and listening

[2/4] Getting your real IP address (without proxy)...
✓ Your real IP: 85.202.184.14

[3/4] Testing SOCKS5 proxy with curl...
✓ SOCKS5 proxy connection successful
  IP through SOCKS5: 45.77.236.204

✓ SUCCESS: Traffic is going through VPN!
  Real IP:        85.202.184.14
  VPN IP (SOCKS5): 45.77.236.204

[4/4] Testing SOCKS5 handshake...
✓ SOCKS5 handshake successful
```

#### Linux/macOS Testing

```bash
# Check IP without VPN
curl https://api.ipify.org
# Output: 203.0.113.1 (your real IP)

# Check IP through SOCKS5 proxy
curl --socks5 localhost:1080 https://api.ipify.org
# Output: 198.51.100.1 (VPN server IP)
```

#### With Authentication

```bash
# If authentication is configured
curl --socks5 myuser:mypass@localhost:1080 https://api.ipify.org
```

#### Testing with wget

```bash
# Without authentication
wget -e use_proxy=yes -e socks_proxy=localhost:1080 https://api.ipify.org -O -

# With authentication
wget --proxy-user=myuser --proxy-password=mypass \
     -e use_proxy=yes -e socks_proxy=localhost:1080 \
     https://api.ipify.org -O -
```

### Browser Configuration

#### Firefox

1. Open **Settings** → **Network Settings**
2. Select **Manual proxy configuration**
3. **SOCKS Host**: `localhost`
4. **Port**: `1080`
5. **SOCKS v5**: ✓ (enable)
6. **Proxy DNS when using SOCKS v5**: ✓ (enable for DNS protection)

#### Chrome/Edge (via extension)

Install **Proxy SwitchyOmega**:

1. [Chrome Web Store](https://chrome.google.com/webstore/detail/proxy-switchyomega/)
2. Create new profile → **Proxy Profile**
3. **Protocol**: SOCKS5
4. **Server**: localhost
5. **Port**: 1080

#### Chrome/Edge (system settings)

**Windows:**

```powershell
# Open proxy settings GUI
start ms-settings:network-proxy

# Or via PowerShell (requires Administrator)
Set-ItemProperty -Path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings" -Name ProxyEnable -Value 1
Set-ItemProperty -Path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings" -Name ProxyServer -Value "socks=localhost:1080"
```

**Note for Windows users:**

- Windows doesn't natively support SOCKS5 in system proxy settings
- Use browser extensions (Proxy SwitchyOmega) or application-specific proxy settings
- For system-wide SOCKS5, use third-party tools like Proxifier or ProxyCap

**Linux:**

```bash
# Via environment variables
export ALL_PROXY=socks5://localhost:1080
export HTTP_PROXY=socks5://localhost:1080
export HTTPS_PROXY=socks5://localhost:1080

# Start Chrome
google-chrome --proxy-server="socks5://localhost:1080"
```

### Application Configuration

#### Git

```bash
# Globally
git config --global http.proxy socks5://localhost:1080
git config --global https.proxy socks5://localhost:1080

# For specific repository
git config http.proxy socks5://localhost:1080

# Disable proxy
git config --global --unset http.proxy
git config --global --unset https.proxy
```

#### Docker (inside container)

```bash
# Run container with proxy
docker run -e HTTP_PROXY=socks5://host.docker.internal:1080 \
           -e HTTPS_PROXY=socks5://host.docker.internal:1080 \
           alpine wget -O - https://api.ipify.org
```

#### Python (requests)

```python
import requests

proxies = {
    'http': 'socks5://localhost:1080',
    'https': 'socks5://localhost:1080'
}

response = requests.get('https://api.ipify.org', proxies=proxies)
print(response.text)
```

#### Node.js

```javascript
const SocksProxyAgent = require("socks-proxy-agent");
const fetch = require("node-fetch");

const agent = new SocksProxyAgent("socks5://localhost:1080");

fetch("https://api.ipify.org", { agent })
  .then((res) => res.text())
  .then((body) => console.log(body));
```

---

## Authentication

### Enable Authentication

```yaml
socks5:
  enabled: true
  listen_addr: "0.0.0.0:1080"
  username: "myuser"
  password: "VeryStr0ngP@ssw0rd!"
  timeout: "30s"
```

After restarting GoXRay:

```
INFO SOCKS5 proxy server started successfully address=0.0.0.0:1080 auth="username/password authentication (user: myuser)" timeout=30s
```

### Using with Authentication

```bash
# curl
curl --socks5 myuser:VeryStr0ngP@ssw0rd!@localhost:1080 https://api.ipify.org

# wget
wget --proxy-user=myuser --proxy-password='VeryStr0ngP@ssw0rd!' \
     -e use_proxy=yes -e socks_proxy=localhost:1080 \
     https://api.ipify.org -O -

# Git
git config --global http.proxy socks5://myuser:VeryStr0ngP@ssw0rd!@localhost:1080

# Python
proxies = {
    'http': 'socks5://myuser:VeryStr0ngP@ssw0rd!@localhost:1080',
    'https': 'socks5://myuser:VeryStr0ngP@ssw0rd!@localhost:1080'
}
```

---

## Security

### Recommendations

#### 1. Use authentication for remote access

If SOCKS5 is accessible from network (`0.0.0.0:1080`), **always** use authentication:

```yaml
socks5:
  enabled: true
  listen_addr: "0.0.0.0:1080" # Accessible from network
  username: "stronguser"
  password: "VeryStr0ngP@ssw0rd!123"
  timeout: "30s"
```

#### 2. Restrict access to localhost only

If remote access is not needed, listen only on localhost:

```yaml
socks5:
  enabled: true
  listen_addr: "127.0.0.1:1080" # Localhost only
  username: ""
  password: ""
  timeout: "30s"
```

#### 3. Use firewall

**Linux (iptables):**

```bash
# Allow only local network
sudo iptables -A INPUT -p tcp --dport 1080 -s 192.168.0.0/16 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 1080 -j DROP

# Save rules
sudo iptables-save > /etc/iptables/rules.v4
```

**Windows (PowerShell Administrator):**

```powershell
# Allow only local network
New-NetFirewallRule -DisplayName "Allow SOCKS5 Local" `
                    -Direction Inbound `
                    -LocalPort 1080 `
                    -Protocol TCP `
                    -RemoteAddress 192.168.0.0/16 `
                    -Action Allow

# Block everything else
New-NetFirewallRule -DisplayName "Block SOCKS5 External" `
                    -Direction Inbound `
                    -LocalPort 1080 `
                    -Protocol TCP `
                    -Action Block
```

#### 4. Use Kill Switch

Enable Kill Switch to prevent IP leaks when VPN disconnects:

```yaml
connection:
  enable_kill_switch: true
```

---

## Troubleshooting

### Issue 1: SOCKS5 doesn't start

**Symptoms:**

```
WARN Failed to start SOCKS5 server error="listen tcp 0.0.0.0:1080: bind: address already in use"
```

**Diagnosis:**

```bash
# Linux
sudo netstat -tuln | grep 1080
sudo lsof -i :1080

# Windows
netstat -an | findstr 1080
```

**Solution:**

1. **Change port:**

```yaml
socks5:
  listen_addr: "0.0.0.0:1081" # Use different port
```

2. **Stop conflicting process:**

```bash
# Linux
sudo kill $(sudo lsof -t -i:1080)

# Windows
# Find PID
netstat -ano | findstr 1080
# Stop process
taskkill /PID <PID> /F
```

---

### Issue 2: Application can't connect to SOCKS5

**Symptoms:**

```
Connection refused to localhost:1080
```

**Diagnosis:**

```bash
# Check if SOCKS5 is running
curl --socks5 localhost:1080 https://api.ipify.org

# Check GoXRay logs
./goxray --config config.yaml --log-level debug
```

**Solution:**

1. **Check that VPN is connected:**

```bash
# SOCKS5 starts ONLY after successful VPN connection
# Check logs:
INFO VPN client connected successfully
INFO SOCKS5 proxy server started successfully
```

2. **Check firewall:**

```bash
# Linux
sudo iptables -L -n | grep 1080

# Windows
Get-NetFirewallRule | Where-Object {$_.LocalPort -eq 1080}
```

3. **Check that port is listening:**

```bash
# Linux
sudo netstat -tuln | grep 1080
# Should show: tcp 0 0 0.0.0.0:1080 0.0.0.0:* LISTEN

# Windows
netstat -an | findstr 1080
# Should show: TCP 0.0.0.0:1080 0.0.0.0:0 LISTENING
```

---

### Issue 3: DNS doesn't resolve through proxy

**Symptoms:**

```
curl: (6) Could not resolve host: example.com
```

**Solution:**

1. **Enable DNS protection in GoXRay:**

```yaml
connection:
  enable_dns_protection: true
```

2. **In browser (Firefox):**

- Enable **"Proxy DNS when using SOCKS v5"**

3. **In curl use --socks5-hostname:**

```bash
# Resolve DNS through SOCKS5
curl --socks5-hostname localhost:1080 https://example.com
```

---

### Issue 4: Slow speed through SOCKS5

**Diagnosis:**

```bash
# Check speed directly (without SOCKS5)
curl -o /dev/null https://speed.cloudflare.com/__down?bytes=100000000

# Check through SOCKS5
curl --socks5 localhost:1080 -o /dev/null https://speed.cloudflare.com/__down?bytes=100000000
```

**Solution:**

1. **Increase timeout:**

```yaml
socks5:
  timeout: "60s" # Increase from 30s to 60s
```

2. **Check VPN server speed:**

```bash
# Check ping
ping <vpn-server-ip>

# Check through VPN
curl --socks5 localhost:1080 https://speed.cloudflare.com/cdn-cgi/trace
```

3. **Use Split Tunneling:**

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16" # Local network direct
    - "10.0.0.0/8" # Corporate network
```

---

### Issue 5: SOCKS5 works but IP doesn't change

**Symptoms:**

```bash
curl --socks5 localhost:1080 https://api.ipify.org
# Shows real IP instead of VPN IP
```

**Diagnosis:**

```bash
# Check if VPN is connected
./goxray --config config.yaml
# Should show: INFO VPN client connected successfully

# Check routes (Linux/macOS)
ip route show
# Should show routes through tun0
```

**Solution:**

1. **Run the test script (Windows):**

```powershell
.\test_socks5.ps1
```

This will automatically diagnose the issue and show if traffic is going through VPN.

2. **Verify VPN is actually working (Linux/macOS):**

```bash
# Check IP through TUN interface
curl --interface tun0 https://api.ipify.org
# Should show VPN server IP
```

3. **Restart GoXRay with debug logs:**

```bash
# Stop
pkill goxray

# Start with debug logging
./goxray --config config.yaml --log-level debug
```

4. **Check routing rules:**

On Linux/macOS, GoXRay creates routing rules that direct all traffic (including from SOCKS5 server) through the TUN device. Verify these rules exist:

```bash
# Linux
ip route show | grep tun0

# macOS
netstat -rn | grep tun0
```

---

## Best Practices

### 1. Use config.yaml

Don't use CLI arguments for SOCKS5 - use `config.yaml`:

```yaml
socks5:
  enabled: true
  listen_addr: "127.0.0.1:1080"
  username: ""
  password: ""
  timeout: "30s"
```

### 2. Combine with Split Tunneling

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16" # Local network direct
    - "10.0.0.0/8" # Corporate network

socks5:
  enabled: true
  listen_addr: "0.0.0.0:1080"
```

**Result:**

- Local resources accessible directly
- Internet through VPN
- SOCKS5 for applications that don't support TUN

### 3. Use Kill Switch

```yaml
connection:
  enable_kill_switch: true
  enable_dns_protection: true

socks5:
  enabled: true
```

**Result:**

- Protection from IP leaks when VPN disconnects
- Protection from DNS leaks
- SOCKS5 for explicit proxy configuration

### 4. Monitoring

```bash
# Check VPN status
curl --socks5 localhost:1080 https://api.ipify.org

# Check metrics (if enabled)
curl http://localhost:9090/metrics | grep vpn_connected

# Check logs
tail -f /var/log/goxray/goxray.log
```

---

## Summary

### Recommended Configuration

```yaml
# config.yaml
connection:
  from_raw_urls:
    - "https://example.com/links.txt"
  enable_ipv6: false
  enable_dns_protection: true
  enable_kill_switch: true
  metrics_port: 9090

split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16"
    - "10.0.0.0/8"

socks5:
  enabled: true
  listen_addr: "127.0.0.1:1080"
  username: ""
  password: ""
  timeout: "30s"
```

### Result

- ✅ All internet traffic through VPN
- ✅ Local network direct (Split Tunneling)
- ✅ DNS protected from leaks
- ✅ Kill Switch prevents IP leaks
- ✅ SOCKS5 for applications with explicit proxy configuration
- ✅ Simple setup and usage

---

## Windows-Specific Notes

### Testing on Windows

GoXRay provides a PowerShell test script (`test_socks5.ps1`) specifically for Windows users:

```powershell
# Download and run the test script
.\test_socks5.ps1
```

This script:

- Tests SOCKS5 connectivity
- Verifies traffic routing through VPN
- Provides detailed diagnostics
- Works without requiring curl.exe (uses PowerShell native commands)

### Windows Firewall

If SOCKS5 is not accessible from other machines on your network:

```powershell
# Allow SOCKS5 through Windows Firewall
New-NetFirewallRule -DisplayName "GoXRay SOCKS5" `
                    -Direction Inbound `
                    -LocalPort 1080 `
                    -Protocol TCP `
                    -Action Allow
```

### Windows Applications

Many Windows applications support SOCKS5 proxy:

- **Browsers**: Firefox (native), Chrome/Edge (via extension)
- **Download Managers**: IDM, Free Download Manager
- **Torrent Clients**: qBittorrent, Transmission
- **Git**: Git for Windows
- **WSL2**: Configure proxy in WSL2 environment

---

## Support

**Documentation:**

- [Main README](../../../README.md) - Project overview
- [Split Tunneling](SPLIT_TUNNELING.md) - Selective routing
- [Kill Switch](KILLSWITCH.md) - IP leak protection
- [Docker Deployment](../deployment/DOCKER.md) - Docker setup

**Testing:**

- Windows: `test_socks5.ps1` (included in repository)
- Linux/macOS: `curl --socks5 localhost:1080 https://api.ipify.org`

**Issues**: Report bugs via [GitHub Issues](https://github.com/baseencode64/tun_with_lits/issues)

---

**Version**: v1.7.0  
**Last Updated**: 2026-06-03  
**Tested on**: Windows 10/11, Linux (Ubuntu, Debian), macOS
