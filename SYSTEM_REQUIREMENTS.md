# System Requirements for GoXRay VPN Client

## Supported Operating Systems

| OS                               | Status           | Notes                                            |
| -------------------------------- | ---------------- | ------------------------------------------------ |
| **Linux** (x86_64, arm64)        | ✅ Full support  | Tested on Ubuntu 24.10, Debian 13                |
| **macOS** (Intel, Apple Silicon) | ✅ Full support  | Tested on Sequoia 15.1.1                         |
| **Windows**                      | ❌ Not supported | Lack of native TUN driver integration. Use WSL2. |

---

## Minimum Hardware Requirements

| Component   | Minimum                      | Recommended                                      |
| ----------- | ---------------------------- | ------------------------------------------------ |
| **CPU**     | 1 core, 1.0 GHz              | 2+ cores, 2.0+ GHz                               |
| **RAM**     | 64 MB (client only)          | 512 MB+                                          |
| **Storage** | 50 MB for binary             | 500 MB (with logs)                               |
| **Network** | Any wired/wireless interface | Gigabit Ethernet recommended for high throughput |

### Expected Runtime Resource Usage

| Metric          | Value               |
| --------------- | ------------------- |
| **CPU (idle)**  | < 5%                |
| **RAM (RSS)**   | ~20-30 MB           |
| **Goroutines**  | 3-5 active          |
| **Disk writes** | ~1-10 KB/s for logs |

---

## Network Interface Requirements

### TUN Device

The application requires **TUN (tunnel) device** support on the system. This is a virtual network kernel interface that operates at Layer 3 (IP packets).

| Requirement        | Description                                     |
| ------------------ | ----------------------------------------------- |
| **TUN module**     | Must be available and loaded (`modprobe tun`)   |
| **Permissions**    | Root or `CAP_NET_ADMIN` capability required     |
| **Routing rights** | Must be able to modify routing tables           |
| **Interface name** | Auto-generated (typically `tun0`, `tun1`, etc.) |
| **IPv4 address**   | `192.18.0.1/32` (default, configurable)         |
| **IPv6 address**   | `fd00:dead:beef::1/64` (when `--ipv6` enabled)  |
| **MTU**            | 1500 bytes (default)                            |

### Network Capabilities Required

```bash
# Minimal capabilities (recommended)
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# These provide:
# - cap_net_raw: RAW socket access (ping, traffic monitoring)
# - cap_net_admin: Network interface configuration (TUN, routes, iptables)
# - cap_net_bind_service: Bind to privileged ports (< 1024)
```

### Linux Capabilities Explained

| Capability             | Required For                                                   |
| ---------------------- | -------------------------------------------------------------- |
| `CAP_NET_ADMIN`        | Creating TUN interface, adding/removing routes, iptables rules |
| `CAP_NET_RAW`          | Reading /proc/net/dev, raw socket operations                   |
| `CAP_NET_BIND_SERVICE` | Binding SOCKS proxy to any port                                |

### Network Ports Used

| Port             | Protocol | Direction           | Purpose                               |
| ---------------- | -------- | ------------------- | ------------------------------------- |
| **Dynamic**      | TCP      | Outbound            | XRay server connection (usually 443)  |
| **Dynamic**      | TCP      | Inbound (localhost) | SOCKS5 proxy for tun2socks            |
| **Configurable** | TCP      | Inbound             | Prometheus metrics (`--metrics-port`) |

---

## Software Dependencies

### Required (installed on most systems by default)

| Package                 | Purpose                                      |
| ----------------------- | -------------------------------------------- |
| `iproute2`              | Route management (`ip route`, `ip -6 route`) |
| `iptables` / `nftables` | DNS leak protection (optional)               |
| `ca-certificates`       | HTTPS verification for fetching server lists |

### Optional

| Package        | Purpose                            |
| -------------- | ---------------------------------- |
| `curl`         | Testing external IP / connectivity |
| `iputils-ping` | Network diagnostics                |
| `jq`           | Parsing JSON health status output  |

---

## Permissions Matrix

### Operation → Required Permission

| Operation              | Without capabilities | With capabilities      |
| ---------------------- | -------------------- | ---------------------- |
| **Run program**        | `sudo goxray`        | `goxray`               |
| **Create TUN**         | `sudo`               | `CAP_NET_ADMIN`        |
| **Modify routes**      | `sudo`               | `CAP_NET_ADMIN`        |
| **SOCKS proxy**        | `sudo` or user       | `CAP_NET_BIND_SERVICE` |
| **Traffic monitoring** | `sudo`               | `CAP_NET_RAW`          |
| **iptables (DNS)**     | `sudo`               | `CAP_NET_ADMIN`        |

---

## Firewall / Security Policy Requirements

If running under a strict firewall or SELinux/AppArmor:

1. **Outbound TCP** to XRay server port must be allowed (usually 443)
2. **TUN device creation** must be permitted
3. **Routing table modification** must be allowed
4. **iptables** rules modification (for DNS protection) must be permitted

### SELinux (Red Hat / Fedora)

```bash
# Allow TUN device access
sudo setsebool -P domain_can_mmap_files 1

# Or create custom policy for goxray
```

### AppArmor (Debian / Ubuntu)

AppArmor profiles typically allow TUN and networking by default. If issues occur:

```bash
# Check if profile is blocking
sudo journalctl | grep "apparmor"

# Temporarily disable for goxray
sudo aa-complain /usr/local/bin/goxray
```

---

## Internet Connectivity

### Requirements for Initial Connection

1. **DNS resolution** working for XRay server hostname
2. **TCP connectivity** to XRay server IP:port
3. **Access to raw URL** (HTTPS) for server list fetching (if using `--from-raw`)
4. **No transparent proxy** intercepting XRay traffic on port 443

### Server List URL Requirements

| Protocol                | Status                                   |
| ----------------------- | ---------------------------------------- |
| **HTTPS** (recommended) | ✅ Full support                          |
| **HTTP**                | ✅ Supported (not recommended)           |
| **Local file**          | ✅ Supported (file:///path/to/links.txt) |

---

## Filesystem Requirements

| Path                    | Access       | Purpose                   |
| ----------------------- | ------------ | ------------------------- |
| `/usr/local/bin/goxray` | Read+Execute | Binary location           |
| `/var/log/goxray/`      | Write        | Log files (if configured) |
| `/tmp/`                 | Write        | Temporary operations      |
| `/proc/net/dev`         | Read         | Traffic statistics        |

---

## Summary

To run GoXRay VPN Client you need:

1. ✅ **Linux x86_64 or arm64** (or macOS)
2. ✅ **TUN device support** (`modprobe tun`)
3. ✅ **Root or capabilities** for network operations
4. ✅ **~30 MB RAM** and **< 5% CPU**
5. ✅ **Outbound TCP** access to XRay server
6. ✅ **~50 MB disk** space

All other requirements are optional and depend on your configuration (IPv6, DNS protection, metrics, etc.).
