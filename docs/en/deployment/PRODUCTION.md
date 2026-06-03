# 📦 Building and Installing GoXRay on Debian 13

## ✅ Ready Binary File

**File**: `goxray_linux_amd64`  
**Size**: ~45.7 MB  
**Architecture**: Linux amd64 (x86_64)  
**Status**: ✅ Compiled and ready to use

---

## 🚀 Quick Start

### Method 1: Simple Copy

```bash
# 1. Copy goxray_linux_amd64 file to Debian server
scp goxray_linux_amd64 user@debian:/usr/local/bin/goxray

# 2. On Debian server:
ssh user@debian

# Make executable
sudo chmod +x /usr/local/bin/goxray

# Configure permissions
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# Start
sudo goxray --from-raw https://example.com/links.txt
```

### Method 2: Automated Installation

```bash
# 1. Copy files to server:
scp goxray_linux_amd64 install_goxray.sh user@debian:/tmp/

# 2. Run installation script:
ssh user@debian
cd /tmp
chmod +x install_goxray.sh
sudo ./install_goxray.sh
```

The script automatically:

- Installs necessary dependencies
- Checks TUN module
- Installs binary file
- Configures capabilities
- Offers to create systemd service

---

## 📋 System Requirements

### Minimum Requirements

- **OS**: Debian 13 (Bookworm) or newer
- **Architecture**: amd64 (x86_64)
- **Kernel**: 4.0+ with TUN/TAP support
- **RAM**: 128 MB minimum
- **Disk**: 100 MB free space

### Required Packages

```bash
sudo apt update
sudo apt install -y iproute2 iputils-ping curl ca-certificates
```

### TUN Module Verification

```bash
# Check module loading
lsmod | grep tun

# If not loaded, load it
sudo modprobe tun

# Check device
ls -la /dev/net/tun
```

---

## 🔧 Manual Installation (Step by Step)

### Step 1: Copy File

```bash
sudo cp goxray_linux_amd64 /usr/local/bin/goxray
sudo chmod +x /usr/local/bin/goxray
```

### Step 2: Configure Permissions

There are two options:

**Option A: Run as root**

```bash
sudo goxray --from-raw https://example.com/links.txt
```

**Option B: Use capabilities (recommended)**

```bash
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# Now can run without sudo
goxray --from-raw https://example.com/links.txt
```

### Step 3: Verify Operation

```bash
# Check availability
goxray --help

# Test connection
sudo goxray --from-raw https://example.com/links.txt
```

---

## 🐳 Docker Installation (Alternative)

### Build Image

```bash
# On Windows (from project directory)
docker build -t goxray .

# Or with platform specification
docker build --platform linux/amd64 -t goxray .
```

### Run Container

```bash
# With raw list
docker run --rm -it \
  --cap-add NET_ADMIN \
  --cap-add NET_RAW \
  --device /dev/net/tun:/dev/net/tun \
  goxray --from-raw https://example.com/links.txt

# With direct link
docker run --rm -it \
  --cap-add NET_ADMIN \
  --cap-add NET_RAW \
  --device /dev/net/tun:/dev/net/tun \
  goxray vless://uuid@server.com:443
```

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: "3.8"

services:
  goxray:
    build: .
    container_name: goxray-vpn
    restart: unless-stopped
    command: ["--from-raw", "https://example.com/links.txt"]
    cap_add:
      - NET_ADMIN
      - NET_RAW
    devices:
      - /dev/net/tun:/dev/net/tun
    volumes:
      - goxray-data:/tmp

volumes:
  goxray-data:
```

Launch:

```bash
docker-compose up -d
docker-compose logs -f
```

---

## 🎯 Usage

### Basic Commands

```bash
# Connect from raw list (recommended)
sudo goxray --from-raw https://example.com/links.txt

# Direct connection
sudo goxray vless://uuid@server.com:443

# Help
goxray --help
```

### Management via systemd (if service installed)

```bash
# Start
sudo systemctl start goxray

# Auto-start on boot
sudo systemctl enable goxray

# View status
sudo systemctl status goxray

# Real-time logs
sudo journalctl -u goxray -f

# Stop
sudo systemctl stop goxray

# Restart
sudo systemctl restart goxray
```

---

## 🔍 Diagnostics

### Verify Connection

```bash
# Check default route
ip route show

# Check DNS
nslookup google.com

# Check external IP
curl -s https://api.ipify.org

# Ping test
ping -c 4 google.com
```

### Logs and Errors

```bash
# View systemd logs
sudo journalctl -u goxray -n 50

# Logs since last boot
sudo journalctl -u goxray -b

# Debug mode
sudo RUST_LOG=debug goxray --from-raw https://example.com/links.txt
```

### Common Issues

**Issue: "permission denied"**

```bash
# Solution
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
```

**Issue: "TUN device not found"**

```bash
# Load module
sudo modprobe tun

# Create device manually
sudo mkdir -p /dev/net
sudo mknod /dev/net/tun c 10 200
sudo chmod 600 /dev/net/tun
```

**Issue: "no available servers"**

```bash
# Check URL availability
curl -I https://example.com/links.txt

# Check link format
cat links.txt

# Test single server manually
sudo goxray vless://uuid@server.com:443
```

---

## 📊 Project Files Information

| File                 | Description                       | Size     |
| -------------------- | --------------------------------- | -------- |
| `goxray_linux_amd64` | Ready binary for Debian           | ~45.7 MB |
| `install_goxray.sh`  | Automated installation script     | ~3 KB    |
| `INSTALL_DEBIAN.md`  | Complete installation guide       | ~8 KB    |
| `Dockerfile`         | Docker image for containerization | ~1 KB    |
| `.dockerignore`      | Docker exclusions                 | ~0.5 KB  |

---

## 🔐 Security

### Recommendations

1. **Use capabilities instead of root**:

   ```bash
   sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
   ```

2. **Always use HTTPS** for loading server lists:

   ```bash
   ✅ goxray --from-raw https://example.com/links.txt
   ❌ goxray --from-raw http://example.com/links.txt
   ```

3. **Configure firewall**:

   ```bash
   sudo ufw default deny outgoing
   sudo ufw allow out 443/tcp
   sudo ufw allow out 80/tcp
   sudo ufw enable
   ```

4. **Regularly update** binary file when new versions are released

---

## 📞 Support

If you encounter problems:

1. 📖 Check [INSTALL_DEBIAN.md](INSTALL_DEBIAN.md) - complete documentation
2. 🔍 View logs: `sudo journalctl -u goxray -f`
3. ✅ Ensure TUN is loaded: `lsmod | grep tun`
4. 🔐 Check permissions: `getcap /usr/local/bin/goxray`
5. 🌐 Test network: `ping`, `curl`, `traceroute`

---

## 🎉 Ready!

Binary file **`goxray_linux_amd64`** successfully built and ready to use on Debian 13!

**Next steps:**

1. Copy file to Debian server
2. Perform installation (manual or automated)
3. Configure VPN connection
4. Enjoy secure connection! 🚀
