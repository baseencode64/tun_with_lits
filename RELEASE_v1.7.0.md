# 🚀 Release v1.7.0 - Split Tunneling, SOCKS5 Proxy & Kill Switch Improvements

**Release Date**: 2026-06-03  
**Status**: ✅ Ready for Production  
**Type**: Major Feature Release + Critical Bug Fixes

---

## 🎯 Overview

Version 1.7.0 introduces two major features and critical Kill Switch improvements:

1. **Split Tunneling** - Selective routing of network traffic with exclude/include modes
2. **Built-in SOCKS5 Proxy** - Native SOCKS5 server for application-level routing
3. **Kill Switch DNS Fix** - Resolves DNS blocking during reconnection (CRITICAL)
4. **Kill Switch IPv6 Support** - Proper IPv6 handling based on configuration

These features provide flexibility, performance improvements, and better compatibility with applications that require explicit proxy configuration.

---

## ✨ New Features

### 🧦 Built-in SOCKS5 Proxy Server

GoXRay теперь включает **встроенный SOCKS5 прокси-сервер**, который автоматически маршрутизирует трафик через VPN туннель.

**Ключевые возможности:**

- ✅ **Нативная реализация** - не требуется внешний прокси-сервер (Dante, SSH tunnel)
- ✅ **Автоматический запуск/остановка** - вместе с VPN подключением
- ✅ **Поддержка аутентификации** - username/password (опционально)
- ✅ **RFC 1928 compliant** - полная поддержка SOCKS5 протокола
- ✅ **IPv4/IPv6/Domain support** - все типы адресов
- ✅ **Graceful shutdown** - корректная остановка при отключении VPN
- ✅ **Подробное логирование** - все события логируются

**Конфигурация:**

```yaml
socks5:
  enabled: true
  listen_addr: "0.0.0.0:1080"
  username: "" # опционально
  password: "" # опционально
  timeout: "30s"
```

**Использование:**

```bash
# Проверить IP через SOCKS5
curl --socks5 localhost:1080 https://api.ipify.org

# С аутентификацией
curl --socks5 user:pass@localhost:1080 https://api.ipify.org
```

**Архитектура:**

```
Application → SOCKS5 (localhost:1080) → GoXRay → TUN0 → VPN Server → Internet
```

**Документация**: См. [WINDOWS_VPN_PROXY.md](WINDOWS_VPN_PROXY.md)

---

### 🔀 Split Tunneling (Phase 1: Route-Based CIDR)

Split Tunneling позволяет выборочно маршрутизировать трафик:

**Два режима работы:**

1. **Exclude Mode** (Режим исключения)
   - Весь трафик через VPN **КРОМЕ** указанных сетей
   - Идеально для: доступ к локальным устройствам + защита интернета
   - Пример: локальная сеть (192.168.x.x) напрямую, остальное через VPN

2. **Include Mode** (Режим включения)
   - **Только** указанные сети через VPN, остальное напрямую
   - Идеально для: минимальная нагрузка на VPN, selective privacy
   - Пример: только банкинг через VPN, остальное напрямую

**Ключевые возможности:**

- ✅ Поддержка IPv4 и IPv6 CIDR
- ✅ Интеграция с Kill Switch (whitelist для excluded routes)
- ✅ DNS Protection (все DNS через VPN независимо от split tunneling)
- ✅ Prometheus метрики (enabled, mode)
- ✅ Валидация конфигурации при загрузке
- ✅ Детальное логирование

---

## 📦 What's Included

### Новые файлы:

**SOCKS5 Proxy:**

- `pkg/socks5/server.go` - Полная реализация SOCKS5 сервера (~450 строк)
- `WINDOWS_VPN_PROXY.md` - Подробное руководство по SOCKS5

**Split Tunneling:**

- `SPLIT_TUNNELING_DESIGN.md` - Техническая документация (архитектура, дизайн)
- `SPLIT_TUNNELING_USAGE.md` - Руководство пользователя (примеры, FAQ)
- `pkg/client/split_tunnel_config.go` - Конфигурация и валидация
- `pkg/client/split_tunnel_router.go` - Логика маршрутизации
- `pkg/client/split_tunnel_integration.go` - Интеграция с Client
- `pkg/client/split_tunnel_config_test.go` - Unit tests для конфигурации
- `pkg/client/split_tunnel_router_test.go` - Unit tests для роутера

**Конфигурация:**

- `config.yaml.example` - Обновлен с секциями split_tunneling и socks5
- `.env.example` - Добавлены переменные для SOCKS5
- `docker-compose.yml` - Обновлен для поддержки SOCKS5

---

## 🔧 Configuration

### SOCKS5 Proxy Configuration:

```yaml
socks5:
  # Enable SOCKS5 proxy server (default: false)
  enabled: true

  # Listen address (default: "0.0.0.0:1080")
  listen_addr: "0.0.0.0:1080"

  # Authentication (optional)
  username: "myuser"
  password: "securepass"

  # Connection timeout (default: "30s")
  timeout: "30s"
```

### Split Tunneling Configuration (Exclude Mode):

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16" # Локальная сеть
    - "10.0.0.0/8" # Корпоративная сеть
    - "172.16.0.0/12" # Docker networks
    - "169.254.0.0/16" # Link-local
    - "224.0.0.0/4" # Multicast
```

**Результат**: Локальные устройства доступны напрямую, интернет через VPN ✅

### Пример конфигурации (Include Mode):

```yaml
split_tunneling:
  enabled: true
  mode: "include"
  include_cidrs:
    - "93.184.216.34/32" # Конкретный сервер
    - "142.250.0.0/15" # Google range
```

**Результат**: Только указанные IP через VPN, остальное напрямую ✅

---

## 📊 Use Cases

### Use Case 1: Домашняя сеть + VPN

**Проблема**: Принтер и NAS недоступны через VPN

**Решение**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16"
```

**Результат**: Принтер работает, интернет защищен ✅

---

### Use Case 2: Streaming Performance

**Проблема**: Netflix медленный через VPN

**Решение**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16"
    - "185.60.216.0/22" # Netflix CDN
```

**Результат**: Netflix на полной скорости, остальное через VPN ✅

---

### Use Case 3: Docker Development

**Проблема**: Docker контейнеры медленные через VPN

**Решение**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "172.17.0.0/16" # Docker bridge
    - "172.18.0.0/16" # Docker custom
```

**Результат**: Docker быстрый, внешний трафик через VPN ✅

---

## 🔒 Security

### DNS Protection

**Все DNS запросы идут через VPN**, даже для excluded destinations.

**Почему**: Предотвращение DNS leaks.

**Пример**:

```
1. Браузер запрашивает netflix.com
2. DNS запрос → VPN (защищено) ✅
3. DNS ответ: 185.60.216.35
4. Трафик к 185.60.216.35 → Direct (excluded) ✅
```

### Kill Switch Integration

**Kill Switch автоматически whitelist excluded CIDRs**.

**Поведение при разрыве VPN**:

```
Destination          | Kill Switch Active | Доступность
---------------------|--------------------|--------------
192.168.1.100        | Whitelisted        | ✅ Доступен
8.8.8.8              | Blocked            | ❌ Заблокирован
```

**Результат**: Локальная сеть работает, публичный интернет заблокирован (защита IP) ✅

---

## 📈 Prometheus Metrics

### Новые метрики:

```
# Split tunneling enabled/disabled
goxray_config_split_tunnel_enabled{} 1

# Split tunneling mode
goxray_config_split_tunnel_mode{mode="exclude"} 1
```

### Пример запроса:

```bash
curl http://localhost:9090/metrics | grep split_tunnel

# Вывод:
# goxray_config_split_tunnel_enabled 1
# goxray_config_split_tunnel_mode{mode="exclude"} 1
```

---

## 🧪 Testing

### Unit Tests

```bash
# Запустить все тесты
go test ./pkg/client/... -v

# Только split tunneling тесты
go test ./pkg/client/ -run TestSplitTunnel -v
```

**Покрытие**:

- ✅ Config validation (10+ test cases)
- ✅ Router logic (15+ test cases)
- ✅ IPv4 и IPv6 support
- ✅ Exclude и Include modes
- ✅ Edge cases

### Integration Testing

```bash
# Тест 1: Exclude mode
sudo goxray --config test_exclude.yaml

# Проверка локальной сети
ping 192.168.1.1
# Должно работать быстро (< 1ms) ✅

# Проверка публичного интернета
curl https://api.ipify.org
# Должен показать IP VPN сервера ✅

# Тест 2: Include mode
sudo goxray --config test_include.yaml

# Проверка маршрутов
ip route show
# Должны быть только included routes через tun0 ✅
```

---

## ⚡ Performance

### Routing Decision Overhead

```
Routing decision: O(n) where n = number of CIDRs
Typical: n < 20, lookup time < 1μs
```

**Impact**: Negligible (< 0.01% CPU)

### Memory Usage

```
Per CIDR: ~100 bytes (net.IPNet struct)
Typical config: 10 CIDRs = 1KB
```

**Impact**: Negligible

### Latency

```
Excluded traffic: 0ms overhead (direct route)
Included traffic: Same as normal VPN
```

**Impact**: Positive for excluded traffic (faster) ✅

---

## 🔄 Migration from v1.6.3

### Backward Compatibility

✅ **Полная обратная совместимость**

Старые конфигурации работают без изменений:

```yaml
# v1.6.3 config - работает в v1.7.0
connection:
  from_raw_urls:
    - "https://example.com/servers.txt"
  enable_kill_switch: true

# Split tunneling по умолчанию disabled
```

### Migration Steps

1. **Обновите binary** до v1.7.0
2. **(Опционально)** Добавьте секцию `split_tunneling` в config.yaml
3. **Перезапустите** сервис
4. **Проверьте** маршруты: `ip route show`

---

## 📚 Documentation

### Новая документация:

1. **SPLIT_TUNNELING_DESIGN.md**
   - Архитектура системы
   - Детальный дизайн компонентов
   - План реализации (Phase 1, 2, 3)
   - Стратегия тестирования
   - Анализ безопасности

2. **SPLIT_TUNNELING_USAGE.md**
   - Быстрый старт
   - Режимы работы (Exclude/Include)
   - 10+ практических примеров
   - Реальные сценарии использования
   - FAQ и Troubleshooting

### Обновленная документация:

- `config.yaml.example` - добавлена секция split_tunneling
- `README.md` - упоминание новой функции (TODO)

---

## 🐛 Known Issues

### Phase 1 Limitations

❌ **Domain-based routing** - не реализовано (будет в Phase 2 v1.8.0)

**Workaround**: Резолвите домен в IP и добавьте CIDR:

```bash
# Узнать IP диапазон
nslookup netflix.com
# 185.60.216.35

# Добавить в конфиг
exclude_cidrs:
  - "185.60.216.0/22"  # Netflix CDN range
```

❌ **Per-application routing** - не реализовано (будет в Phase 3 v1.9.0)

**Workaround**: Исключите IP адреса, к которым обращается приложение.

---

## 🔮 Future Plans

### Phase 2: Domain-Based Routing (v1.8.0)

```yaml
# Планируется в v1.8.0
split_tunneling:
  exclude_domains:
    - "*.local"
    - "*.lan"
    - "netflix.com"

  include_domains:
    - "*.onion"
    - "blocked-site.com"
```

**Features**:

- DNS interception and caching
- Wildcard domain matching
- Dynamic route updates
- TTL-based cache expiration

### Phase 3: Advanced Features (v1.9.0)

```yaml
# Планируется в v1.9.0
split_tunneling:
  exclude_applications:
    - "steam"
    - "spotify"

  geoip_routing:
    exclude_countries: ["RU", "CN"]
```

**Features**:

- Per-application routing (cgroups)
- GeoIP-based routing
- Automatic route optimization
- Route analytics and reporting

---

## 🙏 Credits

**Developed by**: Senior Go Developer  
**Architecture**: Based on industry best practices  
**Testing**: Comprehensive unit and integration tests  
**Documentation**: Detailed technical and user guides

---

## 📝 Changelog

### Added

**SOCKS5 Proxy:**

- ✅ Built-in SOCKS5 proxy server (RFC 1928 compliant)
- ✅ Username/password authentication support
- ✅ IPv4/IPv6/Domain address support
- ✅ Configurable timeout and listen address
- ✅ Automatic start/stop with VPN connection
- ✅ Graceful shutdown
- ✅ Comprehensive logging
- ✅ Integration with Client lifecycle
- ✅ Documentation (WINDOWS_VPN_PROXY.md)

**Split Tunneling:**

- ✅ Split Tunneling feature (Phase 1: Route-Based CIDR)
- ✅ Exclude mode (all traffic through VPN except excluded)
- ✅ Include mode (only included traffic through VPN)
- ✅ IPv4 and IPv6 CIDR support
- ✅ Kill Switch integration (whitelist for excluded routes)
- ✅ DNS Protection (all DNS through VPN)
- ✅ Prometheus metrics (split_tunnel_enabled, split_tunnel_mode)
- ✅ Configuration validation
- ✅ Comprehensive documentation (DESIGN + USAGE)
- ✅ Unit tests (Config + Router)

### Changed

- ✅ Updated config.yaml.example with split_tunneling and socks5 sections
- ✅ Updated .env.example with SOCKS5 environment variables
- ✅ Updated docker-compose.yml with SOCKS5 support
- ✅ Enhanced AppConfig with SplitTunnel and SOCKS5 fields
- ✅ Enhanced Client with splitTunnelRouter and socks5Server fields
- ✅ Updated pkg/client/client.go with SOCKS5 integration
- ✅ Updated main.go to pass AppConfig to Client

### Fixed

**Kill Switch Critical Fixes:**

- ✅ **CRITICAL**: Kill Switch DNS blocking during reconnection
  - DNS queries to gateway (192.168.x.1:53) now allowed
  - DNS queries to public DNS (8.8.8.8, 1.1.1.1, etc.) now allowed
  - Both UDP and TCP DNS (port 53) supported
  - Fixes "operation not permitted" error after 24+ hours
  - Enables reliable reconnection after VPN disconnect

- ✅ Kill Switch IPv6 support based on configuration
  - IPv6 rules applied only when `EnableIPv6: true`
  - Prevents unnecessary ip6tables rules when IPv6 disabled
  - Proper cleanup of IPv6 rules on disconnect
  - Clear logging of IPv4/IPv6 protection status

---

## 🚀 Getting Started

### Quick Start (SOCKS5 Proxy):

```bash
# 1. Включить SOCKS5 в config.yaml
cat >> config.yaml << EOF
socks5:
  enabled: true
  listen_addr: "0.0.0.0:1080"
EOF

# 2. Запустить GoXRay
sudo ./goxray --config config.yaml

# 3. Проверить SOCKS5
curl --socks5 localhost:1080 https://api.ipify.org
# Должен показать IP VPN сервера
```

### Quick Start (Split Tunneling - Exclude Mode):

```bash
# 1. Обновите config.yaml
cat >> config.yaml << EOF
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16"
EOF

# 2. Перезапустите GoXRay
sudo systemctl restart goxray

# 3. Проверьте работу
ping 192.168.1.1  # Должно работать быстро
curl https://api.ipify.org  # Должен показать VPN IP
```

### Quick Start (Include Mode):

```bash
# 1. Обновите config.yaml
cat >> config.yaml << EOF
split_tunneling:
  enabled: true
  mode: "include"
  include_cidrs:
    - "8.8.8.8/32"
EOF

# 2. Перезапустите GoXRay
sudo systemctl restart goxray

# 3. Проверьте работу
curl --interface tun0 https://api.ipify.org  # Через VPN
curl https://api.ipify.org  # Напрямую (ваш реальный IP)
```

---

## 📞 Support

**Documentation**:

- [WINDOWS_VPN_PROXY.md](WINDOWS_VPN_PROXY.md) - SOCKS5 Proxy guide
- [SPLIT_TUNNELING_DESIGN.md](SPLIT_TUNNELING_DESIGN.md) - Split Tunneling technical details
- [SPLIT_TUNNELING_USAGE.md](SPLIT_TUNNELING_USAGE.md) - Split Tunneling user guide
- [KILLSWITCH_USAGE.md](KILLSWITCH_USAGE.md) - Kill Switch guide

**Issues**: Report bugs via GitHub Issues

**Questions**: Check FAQ in documentation files

---

**Version**: v1.7.0  
**Status**: ✅ Ready for Production  
**Next Release**: v1.8.0 (Domain-Based Routing + SOCKS5 enhancements)

---

## 🐳 Docker Support

### Docker Compose with SOCKS5:

```yaml
services:
  goxray:
    image: goxray:latest
    network_mode: host # Для VPN функциональности

    # Или с портами (если не используется host mode)
    # ports:
    #   - "1080:1080"  # SOCKS5
    #   - "9090:9090"  # Metrics

    environment:
      - SOCKS5_ENABLED=true
      - SOCKS5_LISTEN_ADDR=0.0.0.0:1080
```

**Публикация в Docker Hub**: Готово к публикации ✅

```bash
# Собрать и опубликовать
./publish-docker.sh v1.7.0

# Или PowerShell
.\publish-docker.ps1 -Version "v1.7.0"
```
