# Go VPN client for XRay

![Static Badge](https://img.shields.io/badge/OS-macOS%20%7C%20Linux-blue?style=flat&logo=linux&logoColor=white&logoSize=auto&color=blue)
![Static Badge](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go&logoColor=white)
[![Go Report Card](https://goreportcard.com/badge/github.com/goxray/tun)](https://goreportcard.com/report/github.com/goxray/tun)
[![Go Reference](https://pkg.go.dev/badge/github.com/goxray/tun.svg)](https://pkg.go.dev/github.com/goxray/tun)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/goxray/tun/total?color=blue)

This project brings fully functioning [XRay](https://github.com/XTLS/Xray-core) VPN client implementation in Go.

> For desktop version see https://github.com/goxray/desktop

<img alt="Terminal example output" align="center" src="/.github/images/carbon.png">

> [!NOTE]
> The program will not damage your routing rules, default route is intact and only additional rules are added for the lifetime of application's TUN device. There are also additional complementary clean up procedures in place.

#### What is XRay?

Please visit https://xtls.github.io/en for more info.

#### System Requirements

See [docs/getting-started/SYSTEM_REQUIREMENTS.md](docs/getting-started/SYSTEM_REQUIREMENTS.md) for detailed hardware, network interface, and OS requirements.

#### Tested and supported on:

- macOS (tested on Sequoia 15.1.1)
- Linux (tested on Ubuntu 24.10, Debian 13)

> Feel free to test this on your system and let me know in the issues :)

## ✨ Features

- Stupidly easy to use
- Supports all [Xray-core](https://github.com/XTLS/Xray-core) protocols (vless, vmess e.t.c.) using link notation (`vless://` e.t.c.)
- Only soft routing rules are applied, no changes made to default routes
- **IPv6 support** - Full dual-stack IPv4/IPv6 tunneling (enable with `--ipv6` flag)
- **Kill Switch** - Prevents IP leaks when VPN disconnects (see [Kill Switch Guide](docs/features/KILLSWITCH.md))
- **Split Tunneling** - Selective routing with exclude/include modes (see [Split Tunneling Guide](docs/features/SPLIT_TUNNELING.md))
- **SOCKS5 Proxy** - Built-in SOCKS5 server for application-level routing (see [SOCKS5 Guide](docs/features/SOCKS5_PROXY.md))
- **JSON logging** - Structured logging with automatic rotation
- **E2E health check** - Real traffic verification through SOCKS5 tunnel to detect silent connection drops
- **Prometheus metrics** - Built-in metrics endpoint for monitoring

## ⚡️ Installation

The application can be used standalone, as compiled and thrown somewhere in the directory mentioned in PATH.

##### 📦 3rd party Debian package (maintained by [twdragon](https://github.com/twdragon))

The client is available from the PPA repository `ppa:twdragon/xray`, maintained by [twdragon](https://github.com/twdragon). The network privileges in specified automatically by the postinstall script. The package is in sync with this repo's release tags. You can check the pipeline at the [dedicated repository](https://github.com/twdragon/xray-debian-pkg). To install, use:

```bash
sudo add-apt-repository ppa:twdragon/xray
sudo apt update
sudo apt install goxray-cli
```

After the installation, the package might be updated automatically, as is done in Ubuntu. Packages are signed by [twdragon](https://github.com/twdragon) and published on [Launchpad](https://launchpad.net/~twdragon/+archive/ubuntu/xray). Experimental builds are also available in [pipeline repository](https://github.com/twdragon/xray-debian-pkg/actions).

## ⚡️ Usage

> [!IMPORTANT]
>
> - `sudo` is required
> - On linux set `sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip goxray_binary_path`

### Standalone application:

Running the VPN on your machine is as simple as running this little command:

```bash
sudo go run . <proto_link>
```

Where `proto_link` is your XRay link (like `vless://example.com...`), you can get this from your VPN provider or get it from your XRay server.

#### Using Configuration File (Recommended)

Create a YAML configuration file for easier management:

```bash
# Copy example config
cp config.yaml.example goxray.yaml

# Edit with your settings
nano goxray.yaml

# Run with config
sudo go run . --config goxray.yaml
```

Configuration files support all settings including connection, logging, health monitoring, and server selection. CLI arguments override config file values.

**Multiple Server List URLs with Fallback:**

```yaml
connection:
  from_raw_urls:
    - "https://primary.example.com/links.txt"
    - "https://backup1.example.com/links.txt"
    - "https://backup2.example.com/links.txt"
```

The client will try each URL in order. If the first fails, it automatically falls back to the next one.

For detailed documentation, see [docs/configuration/CLI_FLAGS.md](docs/configuration/CLI_FLAGS.md) and [config.yaml.example](config.yaml.example).

#### E2E Health Check (Real Traffic Verification)

By default, health monitoring checks only the local SOCKS5 proxy (`127.0.0.1:port`). This can miss cases where the SOCKS proxy is alive but the VPN tunnel is broken (traffic stops passing through, causing TLS EOF errors).

Enable **E2E (end-to-end) health check** to perform a real HTTP GET request through the VPN tunnel:

```bash
sudo go run . --from-raw https://example.com/links.txt \
  --e2e-check-url "http://ipinfo.io/ip"
```

How it works:

1. Health checker opens a SOCKS5 connection
2. Sends SOCKS5 CONNECT to the target host (through the tunnel)
3. Performs an HTTP GET request (through the tunnel)
4. Verifies a valid HTTP response is received
5. If 3 consecutive checks fail → automatic failover to next server

> [!TIP]
> Use HTTP URLs (not HTTPS) to avoid TLS overhead during health checks. Good options:
>
> - `http://ipinfo.io/ip`
> - `http://connectivitycheck.gstatic.com/generate_204`
> - `http://httpbin.org/get`

Via YAML config:

```yaml
connection:
  e2e_check_url: "http://ipinfo.io/ip"
```

**Default:** empty (SOCKS-only check, backward compatible)

#### Logging Options

Enable JSON logging with rotation:

```bash
sudo go run . --from-raw https://example.com/links.txt \
  --log-file /var/log/goxray/goxray.log \
  --log-format json \
  --log-level info
```

Or use a configuration file (recommended for complex setups):

```yaml
# goxray.yaml
connection:
  from_raw: "https://example.com/links.txt"
logging:
  format: "json"
  file: "/var/log/goxray/goxray.log"
  max_size: 200
  max_backups: 5
```

For more details, see [docs/getting-started/QUICKSTART.md](docs/getting-started/QUICKSTART.md).

### As library in your own project:

> [!NOTE]
> This project is built upon the `core` package, see details and documentation at https://github.com/goxray/core

Install:

```bash
go get github.com/goxray/tun/pkg/client
```

Example:

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
vpn, _ := client.NewClientWithOpts(client.Config{
  TLSAllowInsecure: false,
  Logger:           logger,
})

_ = vpn.Connect(clientLink)
defer vpn.Disconnect(context.Background())

time.Sleep(60 * time.Second)
```

> Please refer to godoc for supported methods and types.

### As a dockerized experience

If you need to use it with Docker - you can look at [this proposed implementation](https://github.com/goxray/tun/pull/8).

## 🛠 Build

The project compiles like a regular Go program:

```bash
go build -o goxray_cli .
```

#### Cross-compilation

```bash
env GOOS=darwin GOARCH=amd64 go build -o goxray_cli_darwin_amd64 .
```

To cross-compile from macOS to Linux arm/amd I use these commands:

```bash
docker run --platform=linux/arm64 -v=${PWD}:/app --workdir=/app arm64v8/golang:1.24 env GOARCH=arm64 go build -o goxray_cli_linux_arm64 .
```

```bash
docker run --platform=linux/amd64 -v=${PWD}:/app --workdir=/app amd64/golang:1.24 env GOARCH=amd64 go build -o goxray_cli_linux_amd64 .
```

## How it works

- Application sets up new TUN device.
- Adds additional routes to route all system traffic to this newly created TUN device.
- Adds exception for XRay outbound address (basically your VPN server IP).
- Tunnel is created to process all incoming IP packets via TCP/IP stack. All outbound traffic is routed through the XRay inbound proxy and all incoming packets are routed back via TUN device.

## 📚 Documentation

> **Choose your language:**
>
> - **[🇬🇧 English Documentation](docs/en/README.md)** - Full English docs
> - **[🇷🇺 Русская документация](docs/ru/README.md)** - Полная русская документация

### Quick Links (English):

- **[Quick Start Guide](docs/en/getting-started/QUICKSTART.md)** - Get started in 5 minutes
- **[Kill Switch](docs/en/features/KILLSWITCH.md)** - IP leak protection
- **[Split Tunneling](docs/en/features/SPLIT_TUNNELING.md)** - Selective routing
- **[SOCKS5 Proxy](docs/en/features/SOCKS5_PROXY.md)** - Built-in SOCKS5 server
- **[Deployment Guide](docs/en/deployment/PRODUCTION.md)** - Production deployment

### Быстрые ссылки (Русский):

- **[Быстрый старт](docs/ru/getting-started/QUICKSTART.md)** - Начните за 5 минут
- **[Kill Switch](docs/ru/features/KILLSWITCH.md)** - Защита от утечек IP
- **[Split Tunneling](docs/ru/features/SPLIT_TUNNELING.md)** - Выборочная маршрутизация
- **[SOCKS5 Proxy](docs/ru/features/SOCKS5_PROXY.md)** - Встроенный SOCKS5 сервер
- **[Руководство по развертыванию](docs/ru/deployment/PRODUCTION.md)** - Production окружение

## 📦 Latest Release

**v1.7.0** - [Release Notes](RELEASE_v1.7.0.md)

New features:

- ✅ Split Tunneling (Phase 1: Route-Based CIDR)
- ✅ Built-in SOCKS5 Proxy Server
- ✅ Kill Switch DNS fix (CRITICAL)
- ✅ Kill Switch IPv6 support

## 📝 TODO

- [х] Add DNS leak protection
- [х] Add kill switch functionality
- [х] Add split tunneling
- [х] Add SOCKS5 proxy server
- [х] Add Prometheus metrics endpoint
- [х] Add configuration file support (YAML/TOML)
- [ ] Add Web Dashboard / TUI interface
- [ ] Add domain-based routing (Phase 2)
- [ ] Add per-application routing (Phase 3)
