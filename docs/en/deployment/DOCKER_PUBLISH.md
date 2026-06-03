# 🐳 DockerHub Publishing Guide - GoXRay VPN Client

**Version**: v1.7.0  
**Date**: 2026-06-01  
**Status**: ✅ Ready to Publish

---

## 📖 Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Quick Start](#quick-start)
4. [Manual Publishing](#manual-publishing)
5. [Automated Publishing](#automated-publishing)
6. [Multi-Architecture Builds](#multi-architecture-builds)
7. [Verification](#verification)
8. [Troubleshooting](#troubleshooting)

---

## Overview

This guide explains how to publish GoXRay Docker images to DockerHub, making them available for public or private use.

### What Gets Published

- **Image Name**: `your-username/goxray`
- **Tags**: `1.7.0`, `latest`
- **Architecture**: `linux/amd64` (default), `linux/arm64` (optional)
- **Size**: ~50MB (compressed)

---

## Prerequisites

### 1. DockerHub Account

Create a free account at [hub.docker.com](https://hub.docker.com)

### 2. Docker Installed

```bash
# Verify Docker installation
docker --version
# Should show: Docker version 20.10+
```

### 3. Login to DockerHub

```bash
# Login to DockerHub
docker login

# Enter your credentials:
# Username: your-dockerhub-username
# Password: your-dockerhub-password (or access token)
```

**Recommended**: Use access token instead of password:

1. Go to [DockerHub Settings → Security](https://hub.docker.com/settings/security)
2. Click "New Access Token"
3. Copy token and use as password

---

## Quick Start

### Option 1: Using Bash Script (Linux/macOS/WSL)

```bash
# 1. Set your DockerHub username
export DOCKER_USERNAME="your-dockerhub-username"

# 2. Make script executable
chmod +x publish-docker.sh

# 3. Run the script
./publish-docker.sh 1.7.0

# Or specify version as argument
./publish-docker.sh 1.8.0
```

### Option 2: Using PowerShell Script (Windows)

```powershell
# 1. Set your DockerHub username
$env:DOCKER_USERNAME = "your-dockerhub-username"

# 2. Run the script
.\publish-docker.ps1 -Version "1.7.0"

# Or specify username directly
.\publish-docker.ps1 -Username "your-dockerhub-username" -Version "1.7.0"
```

---

## Manual Publishing

### Step 1: Build the Image

```bash
# Set variables
DOCKER_USERNAME="your-dockerhub-username"
VERSION="1.7.0"

# Build image
docker build -t ${DOCKER_USERNAME}/goxray:${VERSION} .
```

**Expected output**:

```
[+] Building 45.2s (15/15) FINISHED
 => [internal] load build definition from Dockerfile
 => => transferring dockerfile: 1.23kB
 => [internal] load .dockerignore
 ...
 => => naming to docker.io/your-username/goxray:1.7.0
```

### Step 2: Tag as Latest

```bash
# Tag as latest
docker tag ${DOCKER_USERNAME}/goxray:${VERSION} ${DOCKER_USERNAME}/goxray:latest
```

### Step 3: Test the Image

```bash
# Test help command
docker run --rm ${DOCKER_USERNAME}/goxray:${VERSION} --help

# Expected output:
# ERROR: no config provided
# usage: goxray [options]
# ...
```

### Step 4: Push to DockerHub

```bash
# Push version tag
docker push ${DOCKER_USERNAME}/goxray:${VERSION}

# Push latest tag
docker push ${DOCKER_USERNAME}/goxray:latest
```

**Expected output**:

```
The push refers to repository [docker.io/your-username/goxray]
5f70bf18a086: Pushed
1.7.0: digest: sha256:abc123... size: 1234
```

### Step 5: Verify on DockerHub

Visit: `https://hub.docker.com/r/your-username/goxray`

You should see:

- ✅ Tags: `1.7.0`, `latest`
- ✅ Last pushed: Just now
- ✅ Image size: ~50MB

---

## Automated Publishing

### GitHub Actions (Recommended)

Create `.github/workflows/docker-publish.yml`:

```yaml
name: Publish Docker Image

on:
  push:
    tags:
      - "v*.*.*"
  workflow_dispatch:

jobs:
  build-and-push:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2

      - name: Login to DockerHub
        uses: docker/login-action@v2
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Extract version from tag
        id: version
        run: echo "VERSION=${GITHUB_REF#refs/tags/v}" >> $GITHUB_OUTPUT

      - name: Build and push
        uses: docker/build-push-action@v4
        with:
          context: .
          push: true
          tags: |
            ${{ secrets.DOCKERHUB_USERNAME }}/goxray:${{ steps.version.outputs.VERSION }}
            ${{ secrets.DOCKERHUB_USERNAME }}/goxray:latest
          platforms: linux/amd64,linux/arm64
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

**Setup**:

1. Add secrets to GitHub repository:
   - Go to Settings → Secrets → Actions
   - Add `DOCKERHUB_USERNAME`
   - Add `DOCKERHUB_TOKEN` (access token from DockerHub)

2. Create and push a tag:

   ```bash
   git tag v1.7.0
   git push origin v1.7.0
   ```

3. GitHub Actions will automatically build and push the image

---

## Multi-Architecture Builds

Build for multiple architectures (amd64, arm64):

### Using Docker Buildx

```bash
# 1. Create builder
docker buildx create --name multiarch --use

# 2. Build and push for multiple platforms
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ${DOCKER_USERNAME}/goxray:${VERSION} \
  -t ${DOCKER_USERNAME}/goxray:latest \
  --push \
  .
```

**Supported platforms**:

- `linux/amd64` - Intel/AMD 64-bit (most common)
- `linux/arm64` - ARM 64-bit (Raspberry Pi 4, Apple M1/M2)
- `linux/arm/v7` - ARM 32-bit (Raspberry Pi 3)

---

## Verification

### 1. Check Image on DockerHub

```bash
# View image details
docker manifest inspect ${DOCKER_USERNAME}/goxray:${VERSION}
```

### 2. Pull and Test

```bash
# Pull the image
docker pull ${DOCKER_USERNAME}/goxray:${VERSION}

# Run test
docker run --rm ${DOCKER_USERNAME}/goxray:${VERSION} --help
```

### 3. Check Image Size

```bash
# List images
docker images ${DOCKER_USERNAME}/goxray

# Expected output:
# REPOSITORY              TAG       IMAGE ID       CREATED         SIZE
# your-username/goxray    1.7.0     abc123def456   2 minutes ago   48.5MB
# your-username/goxray    latest    abc123def456   2 minutes ago   48.5MB
```

### 4. Scan for Vulnerabilities

```bash
# Scan image (requires Docker Scout or Trivy)
docker scout cves ${DOCKER_USERNAME}/goxray:${VERSION}

# Or use Trivy
trivy image ${DOCKER_USERNAME}/goxray:${VERSION}
```

---

## Troubleshooting

### Issue 1: Authentication Failed

**Symptoms**:

```
Error response from daemon: Get https://registry-1.docker.io/v2/: unauthorized
```

**Solution**:

```bash
# Re-login to DockerHub
docker logout
docker login

# Or use access token
docker login -u your-username -p your-access-token
```

---

### Issue 2: Build Failed

**Symptoms**:

```
ERROR [internal] load metadata for docker.io/library/golang:1.25.6-alpine
```

**Solution**:

```bash
# Pull base image manually
docker pull golang:1.25.6-alpine

# Retry build
docker build -t ${DOCKER_USERNAME}/goxray:${VERSION} .
```

---

### Issue 3: Push Failed (Rate Limited)

**Symptoms**:

```
Error response from daemon: toomanyrequests: You have reached your pull rate limit
```

**Solution**:

- Wait 6 hours (free tier limit)
- Or upgrade to DockerHub Pro
- Or use authenticated pulls (already logged in)

---

### Issue 4: Image Too Large

**Symptoms**:

```
Image size: 500MB (expected ~50MB)
```

**Solution**:

```bash
# Use multi-stage build (already in Dockerfile)
# Ensure .dockerignore excludes unnecessary files

# Check .dockerignore
cat .dockerignore

# Should contain:
# .git
# *.md
# tests/
# etc.
```

---

## Best Practices

### 1. Use Semantic Versioning

```bash
# Good
docker tag image:1.7.0
docker tag image:1.7
docker tag image:1
docker tag image:latest

# Bad
docker tag image:new
docker tag image:fixed
```

### 2. Always Tag with Version

```bash
# Always push both version and latest
docker push ${DOCKER_USERNAME}/goxray:1.7.0
docker push ${DOCKER_USERNAME}/goxray:latest
```

### 3. Add Image Labels

In `Dockerfile`:

```dockerfile
LABEL org.opencontainers.image.title="GoXRay VPN Client"
LABEL org.opencontainers.image.description="VPN client with Split Tunneling"
LABEL org.opencontainers.image.version="1.7.0"
LABEL org.opencontainers.image.authors="GoXRay Team"
LABEL org.opencontainers.image.source="https://github.com/baseencode64/tun_with_lits"
```

### 4. Create README on DockerHub

Add description to DockerHub repository:

````markdown
# GoXRay VPN Client

High-performance VPN client with advanced features:

- ✅ Split Tunneling (selective routing)
- ✅ Kill Switch (traffic protection)
- ✅ DNS Leak Protection
- ✅ IPv6 Support
- ✅ Prometheus Metrics
- ✅ Auto-reconnection

## Quick Start

```bash
docker run -d --privileged --network host \
  -v ./config.yaml:/etc/goxray/config.yaml:ro \
  your-username/goxray:latest \
  --config /etc/goxray/config.yaml
```
````

## Documentation

- [GitHub Repository](https://github.com/baseencode64/tun_with_lits)
- [Docker Deployment Guide](DOCKER_DEPLOYMENT.md)
- [Split Tunneling Guide](SPLIT_TUNNELING_USAGE.md)

````

---

## Commands Reference

### Build Commands

```bash
# Build for current platform
docker build -t username/goxray:1.7.0 .

# Build with no cache
docker build --no-cache -t username/goxray:1.7.0 .

# Build for specific platform
docker build --platform linux/amd64 -t username/goxray:1.7.0 .

# Build multi-arch
docker buildx build --platform linux/amd64,linux/arm64 -t username/goxray:1.7.0 --push .
````

### Tag Commands

```bash
# Tag as latest
docker tag username/goxray:1.7.0 username/goxray:latest

# Tag with multiple versions
docker tag username/goxray:1.7.0 username/goxray:1.7
docker tag username/goxray:1.7.0 username/goxray:1
```

### Push Commands

```bash
# Push specific tag
docker push username/goxray:1.7.0

# Push all tags
docker push username/goxray --all-tags
```

### Cleanup Commands

```bash
# Remove local images
docker rmi username/goxray:1.7.0

# Remove all unused images
docker image prune -a

# Remove build cache
docker builder prune
```

---

## Example: Complete Publishing Workflow

```bash
#!/bin/bash
# Complete publishing workflow

# 1. Set variables
export DOCKER_USERNAME="your-dockerhub-username"
export VERSION="1.7.0"

# 2. Login
docker login

# 3. Build
docker build -t ${DOCKER_USERNAME}/goxray:${VERSION} .

# 4. Tag
docker tag ${DOCKER_USERNAME}/goxray:${VERSION} ${DOCKER_USERNAME}/goxray:latest
docker tag ${DOCKER_USERNAME}/goxray:${VERSION} ${DOCKER_USERNAME}/goxray:1.7
docker tag ${DOCKER_USERNAME}/goxray:${VERSION} ${DOCKER_USERNAME}/goxray:1

# 5. Test
docker run --rm ${DOCKER_USERNAME}/goxray:${VERSION} --help

# 6. Push all tags
docker push ${DOCKER_USERNAME}/goxray:${VERSION}
docker push ${DOCKER_USERNAME}/goxray:latest
docker push ${DOCKER_USERNAME}/goxray:1.7
docker push ${DOCKER_USERNAME}/goxray:1

# 7. Verify
docker pull ${DOCKER_USERNAME}/goxray:latest
docker images ${DOCKER_USERNAME}/goxray

echo "✅ Published successfully!"
echo "Image: ${DOCKER_USERNAME}/goxray:${VERSION}"
echo "DockerHub: https://hub.docker.com/r/${DOCKER_USERNAME}/goxray"
```

---

## Support

**Documentation**:

- [DOCKER_DEPLOYMENT.md](DOCKER_DEPLOYMENT.md) - Docker deployment guide
- [README.md](README.md) - Main documentation

**Issues**: Report bugs via GitHub Issues

**DockerHub**: [hub.docker.com](https://hub.docker.com)

---

**Version**: v1.7.0  
**Status**: ✅ Ready to Publish  
**Last Updated**: 2026-06-01
