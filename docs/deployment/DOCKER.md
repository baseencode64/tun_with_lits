# 🐳 Docker Deployment Guide - GoXRay VPN Client

**Version**: v1.7.0  
**Date**: 2026-06-01  
**Status**: ✅ Production Ready

---

## 📖 Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Quick Start](#quick-start)
4. [Configuration](#configuration)
5. [Deployment Options](#deployment-options)
6. [Monitoring](#monitoring)
7. [Troubleshooting](#troubleshooting)
8. [Security](#security)
9. [Advanced Usage](#advanced-usage)

---

## Overview

GoXRay VPN Client can be deployed using Docker and Docker Compose for easy containerized deployment. This guide covers all aspects of Docker-based deployment.

### Features

- ✅ **Containerized deployment** - Easy to deploy and manage
- ✅ **Privileged mode** - Full access to network stack
- ✅ **Host network mode** - Direct access to host interfaces
- ✅ **TUN device support** - VPN tunnel creation
- ✅ **Health checks** - Automatic container health monitoring
- ✅ **Persistent storage** - Configuration and logs persistence
- ✅ **Environment variables** - Easy configuration via .env file

---

## Prerequisites

### System Requirements

**Operating System**:

- Linux (Ubuntu 20.04+, Debian 11+, CentOS 8+)
- Kernel 4.9+ (for TUN device support)

**Software**:

- Docker 20.10+
- Docker Compose 1.29+ (or Docker Compose V2)

**Permissions**:

- Root or sudo access (required for privileged containers)

### Install Docker

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Start Docker service
sudo systemctl enable docker
sudo systemctl start docker

# Verify installation
docker --version
docker-compose --version
```

---

## Quick Start

### Step 1: Clone Repository

```bash
git clone https://github.com/baseencode64/tun_with_lits.git
cd tun_with_lits
```

### Step 2: Prepare Configuration

```bash
# Copy example config
cp config.yaml.example config.yaml

# Edit configuration
nano config.yaml
```

**Minimal config.yaml**:

```yaml
connection:
  from_raw_urls:
    - "https://example.com/servers.txt"

  enable_ipv6: false
  enable_dns_protection: true
  enable_kill_switch: true

logging:
  format: "text"
  level: "info"

health_monitoring:
  check_interval: "10s"
  timeout: "5s"
  max_retries: 3
```

### Step 3: Create Environment File

```bash
# Copy example .env
cp .env.example .env

# Edit environment variables (optional)
nano .env
```

### Step 4: Build and Run

```bash
# Build image
docker-compose build

# Start container
docker-compose up -d

# Check logs
docker-compose logs -f
```

### Step 5: Verify Connection

```bash
# Check container status
docker-compose ps

# Check VPN connection
docker exec goxray-vpn ip addr show tun0

# Check public IP (should show VPN server IP)
curl https://api.ipify.org
```

---

## Configuration

### docker-compose.yml

The `docker-compose.yml` file defines the GoXRay service:

```yaml
version: "3.8"

services:
  goxray:
    build:
      context: .
      dockerfile: Dockerfile

    image: goxray:latest
    container_name: goxray-vpn

    # Required for VPN functionality
    privileged: true
    network_mode: host

    cap_add:
      - NET_ADMIN
      - NET_RAW
      - NET_BIND_SERVICE

    devices:
      - /dev/net/tun:/dev/net/tun

    volumes:
      - ./config.yaml:/etc/goxray/config.yaml:ro
      - ./logs:/var/log/goxray
      - goxray-data:/var/lib/goxray

    environment:
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - METRICS_PORT=${METRICS_PORT:-9090}

    restart: unless-stopped
```

### Environment Variables

**Available variables** (defined in `.env`):

| Variable                | Default   | Description                          |
| ----------------------- | --------- | ------------------------------------ |
| `LOG_LEVEL`             | `info`    | Log level (debug, info, warn, error) |
| `LOG_FORMAT`            | `text`    | Log format (text, json)              |
| `METRICS_PORT`          | `9090`    | Prometheus metrics port              |
| `ENABLE_IPV6`           | `false`   | Enable IPv6 support                  |
| `ENABLE_DNS_PROTECTION` | `true`    | Enable DNS leak protection           |
| `ENABLE_KILL_SWITCH`    | `true`    | Enable kill switch                   |
| `SPLIT_TUNNEL_ENABLED`  | `false`   | Enable split tunneling               |
| `SPLIT_TUNNEL_MODE`     | `exclude` | Split tunnel mode (exclude/include)  |

### config.yaml

Main configuration file mounted at `/etc/goxray/config.yaml`:

```yaml
connection:
  from_raw_urls:
    - "https://example.com/servers.txt"

  enable_ipv6: false
  enable_dns_protection: true
  enable_kill_switch: true
  metrics_port: 9090

split_tunneling:
  enabled: false
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16"
    - "10.0.0.0/8"

logging:
  format: "text"
  level: "info"
  file: "/var/log/goxray/goxray.log"
```

---

## Deployment Options

### Option 1: Standard Deployment

**Use case**: Single VPN client on dedicated server

```bash
# Start container
docker-compose up -d

# View logs
docker-compose logs -f goxray

# Stop container
docker-compose down
```

### Option 2: Development Mode

**Use case**: Testing and debugging

```bash
# Build with no cache
docker-compose build --no-cache

# Run with debug logging
LOG_LEVEL=debug docker-compose up

# Interactive shell
docker exec -it goxray-vpn sh
```

### Option 3: Production Deployment

**Use case**: Production environment with monitoring

```yaml
# docker-compose.prod.yml
version: "3.8"

services:
  goxray:
    extends:
      file: docker-compose.yml
      service: goxray

    environment:
      - LOG_LEVEL=info
      - LOG_FORMAT=json
      - METRICS_PORT=9090

    logging:
      driver: "json-file"
      options:
        max-size: "50m"
        max-file: "5"

    deploy:
      resources:
        limits:
          cpus: "2"
          memory: 512M
        reservations:
          cpus: "0.5"
          memory: 128M
```

**Deploy**:

```bash
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### Option 4: Multi-Instance Deployment

**Use case**: Multiple VPN clients with different configurations

```bash
# Create separate directories
mkdir -p vpn1 vpn2

# Copy configs
cp config.yaml vpn1/
cp config.yaml vpn2/

# Edit configs with different server lists
nano vpn1/config.yaml
nano vpn2/config.yaml

# Start instances
docker-compose -f vpn1/docker-compose.yml up -d
docker-compose -f vpn2/docker-compose.yml up -d
```

---

## Monitoring

### Health Checks

Docker Compose includes automatic health checks:

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:9090/metrics"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 10s
```

**Check health status**:

```bash
# View health status
docker-compose ps

# Expected output:
# NAME          STATUS                    PORTS
# goxray-vpn    Up (healthy)
```

### Prometheus Metrics

Access metrics at `http://localhost:9090/metrics`:

```bash
# View metrics
curl http://localhost:9090/metrics

# Key metrics:
# - vpn_connected{} 1
# - vpn_bytes_read_total{} 1234567
# - vpn_bytes_written_total{} 7654321
# - goxray_config_split_tunnel_enabled{} 1
```

### Logs

**View logs**:

```bash
# Follow logs
docker-compose logs -f

# Last 100 lines
docker-compose logs --tail=100

# Specific service
docker-compose logs goxray

# Save logs to file
docker-compose logs > goxray.log
```

**Log files** (if configured):

```bash
# View log file inside container
docker exec goxray-vpn cat /var/log/goxray/goxray.log

# Copy logs from container
docker cp goxray-vpn:/var/log/goxray/goxray.log ./
```

---

## Troubleshooting

### Issue 1: Container Fails to Start

**Symptoms**:

```bash
docker-compose ps
# NAME          STATUS
# goxray-vpn    Exited (1)
```

**Diagnosis**:

```bash
# Check logs
docker-compose logs goxray

# Common errors:
# - "permission denied" → Need privileged mode
# - "TUN device not found" → Missing /dev/net/tun
# - "config file not found" → Check volume mount
```

**Solutions**:

```bash
# 1. Ensure privileged mode
grep "privileged: true" docker-compose.yml

# 2. Check TUN device
ls -l /dev/net/tun
# Should show: crw-rw-rw- 1 root root 10, 200

# 3. Create TUN device if missing
sudo mkdir -p /dev/net
sudo mknod /dev/net/tun c 10 200
sudo chmod 666 /dev/net/tun

# 4. Verify config file
ls -l config.yaml
```

---

### Issue 2: VPN Not Connecting

**Symptoms**:

```bash
# Metrics show disconnected
curl http://localhost:9090/metrics | grep vpn_connected
# vpn_connected{} 0
```

**Diagnosis**:

```bash
# Check logs for connection errors
docker-compose logs | grep -i error

# Common errors:
# - "invalid config" → Check config.yaml syntax
# - "server unreachable" → Check from_raw_urls
# - "authentication failed" → Check VLESS link
```

**Solutions**:

```bash
# 1. Validate config
docker exec goxray-vpn cat /etc/goxray/config.yaml

# 2. Test server connectivity
docker exec goxray-vpn ping -c 3 example.com

# 3. Check DNS resolution
docker exec goxray-vpn nslookup example.com

# 4. Restart container
docker-compose restart
```

---

### Issue 3: DNS Leaks

**Symptoms**:

```bash
# DNS queries go through ISP
nslookup google.com
# Server: 192.168.1.1  ← ISP DNS (leak!)
```

**Diagnosis**:

```bash
# Check DNS protection setting
grep "enable_dns_protection" config.yaml
# Should be: enable_dns_protection: true

# Check logs
docker-compose logs | grep -i dns
```

**Solutions**:

```bash
# 1. Enable DNS protection in config
nano config.yaml
# Set: enable_dns_protection: true

# 2. Restart container
docker-compose restart

# 3. Verify DNS routes
docker exec goxray-vpn ip route | grep tun0
```

---

### Issue 4: Split Tunneling Not Working

**Symptoms**:

```bash
# Local network not accessible
ping 192.168.1.1
# Request timeout
```

**Diagnosis**:

```bash
# Check split tunneling config
grep -A 10 "split_tunneling" config.yaml

# Check routes
docker exec goxray-vpn ip route show
```

**Solutions**:

```bash
# 1. Verify split tunneling enabled
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16"

# 2. Check excluded routes
docker exec goxray-vpn ip route | grep 192.168

# 3. Restart container
docker-compose restart
```

---

## Security

### Best Practices

1. **Use read-only config**:

   ```yaml
   volumes:
     - ./config.yaml:/etc/goxray/config.yaml:ro # :ro = read-only
   ```

2. **Limit container resources**:

   ```yaml
   deploy:
     resources:
       limits:
         cpus: "2"
         memory: 512M
   ```

3. **Use secrets for sensitive data**:

   ```bash
   # Store VLESS links in Docker secrets
   echo "vless://..." | docker secret create vless_link -
   ```

4. **Enable kill switch**:

   ```yaml
   connection:
     enable_kill_switch: true
   ```

5. **Regular updates**:

   ```bash
   # Pull latest image
   docker-compose pull

   # Rebuild
   docker-compose build --no-cache

   # Restart
   docker-compose up -d
   ```

### Network Isolation

**Option 1: Host network** (current, required for VPN):

```yaml
network_mode: host
```

**Option 2: Custom network** (not recommended for VPN):

```yaml
networks:
  vpn_network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

---

## Advanced Usage

### Custom Dockerfile

Create custom Dockerfile for additional tools:

```dockerfile
# Dockerfile.custom
FROM goxray:latest

# Install additional tools
RUN apk add --no-cache \
    tcpdump \
    net-tools \
    bind-tools

# Custom entrypoint
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
```

**Build**:

```bash
docker build -f Dockerfile.custom -t goxray:custom .
```

### Docker Swarm Deployment

```yaml
# docker-stack.yml
version: "3.8"

services:
  goxray:
    image: goxray:latest

    deploy:
      mode: replicated
      replicas: 1
      placement:
        constraints:
          - node.role == manager

      resources:
        limits:
          cpus: "2"
          memory: 512M

    networks:
      - vpn_network

networks:
  vpn_network:
    driver: overlay
```

**Deploy**:

```bash
docker stack deploy -c docker-stack.yml goxray
```

### Kubernetes Deployment

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: goxray
spec:
  replicas: 1
  selector:
    matchLabels:
      app: goxray
  template:
    metadata:
      labels:
        app: goxray
    spec:
      hostNetwork: true
      containers:
        - name: goxray
          image: goxray:latest
          securityContext:
            privileged: true
            capabilities:
              add:
                - NET_ADMIN
                - NET_RAW
          volumeMounts:
            - name: config
              mountPath: /etc/goxray
            - name: tun
              mountPath: /dev/net/tun
      volumes:
        - name: config
          configMap:
            name: goxray-config
        - name: tun
          hostPath:
            path: /dev/net/tun
```

---

## Commands Reference

### Docker Compose Commands

```bash
# Build image
docker-compose build

# Start containers
docker-compose up -d

# Stop containers
docker-compose down

# Restart containers
docker-compose restart

# View logs
docker-compose logs -f

# Check status
docker-compose ps

# Execute command in container
docker-compose exec goxray sh

# Remove containers and volumes
docker-compose down -v
```

### Docker Commands

```bash
# List containers
docker ps

# View logs
docker logs goxray-vpn

# Execute command
docker exec -it goxray-vpn sh

# Inspect container
docker inspect goxray-vpn

# View resource usage
docker stats goxray-vpn

# Remove container
docker rm -f goxray-vpn

# Remove image
docker rmi goxray:latest
```

---

## Maintenance

### Backup

```bash
# Backup configuration
tar -czf goxray-backup-$(date +%Y%m%d).tar.gz \
  config.yaml \
  .env \
  docker-compose.yml

# Backup volumes
docker run --rm \
  -v goxray-data:/data \
  -v $(pwd):/backup \
  alpine tar -czf /backup/goxray-data-$(date +%Y%m%d).tar.gz /data
```

### Restore

```bash
# Restore configuration
tar -xzf goxray-backup-20260601.tar.gz

# Restore volumes
docker run --rm \
  -v goxray-data:/data \
  -v $(pwd):/backup \
  alpine tar -xzf /backup/goxray-data-20260601.tar.gz -C /
```

### Updates

```bash
# Pull latest code
git pull origin main

# Rebuild image
docker-compose build --no-cache

# Restart with new image
docker-compose up -d

# Clean old images
docker image prune -a
```

---

## Support

**Documentation**:

- [README.md](README.md) - Main documentation
- [SPLIT_TUNNELING_USAGE.md](SPLIT_TUNNELING_USAGE.md) - Split tunneling guide
- [KILLSWITCH_USAGE.md](KILLSWITCH_USAGE.md) - Kill switch guide

**Issues**: Report bugs via GitHub Issues

**Community**: Join discussions on GitHub

---

**Version**: v1.7.0  
**Status**: ✅ Production Ready  
**Last Updated**: 2026-06-01
