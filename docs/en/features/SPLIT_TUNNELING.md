# 🔀 Split Tunneling - User Guide

**Version**: v1.7.0  
**Status**: 📋 Documentation  
**Date**: 2026-06-01

---

## 📖 Table of Contents

1. [What is Split Tunneling?](#что-такое-split-tunneling)
2. [Quick Start](#быстрый-старт)
3. [Operating Modes](#режимы-работы)
4. [Examples конфигурации](#примеры-конфигурации)
5. [Practical Scenarios](#практические-сценарии)
6. [Integration with Other Features](#интеграция-with-другими-функциями)
7. [Monitoring and Debugging](#мониторинг-and-отладка)
8. [Troubleshooting](#troubleshooting)
9. [FAQ](#faq)

---

## What is Split Tunneling?

### Definition

**Split Tunneling** (раздельное туннелирование) - это технология, которая позволяет **выборочно** направлять сетевой трафик:

- **Часть трафика** → through VPN туннель (защищено, анонимно)
- **Часть трафика** → directly через ISP (быстрее, without VPN)

### Зачем это need?

#### ❌ Без Split Tunneling

```
┌──────────────┐
│ Ваш компьютер│
└──────┬───────┘
       │
       │ ВСЁ through VPN
       ▼
┌──────────────┐      ┌──────────────┐
│  VPN Server  │─────▶│   Internet   │
└──────────────┘      └──────────────┘

Problems:
❌ Local принтер недоступен
❌ Netflix медленный (VPN далеко)
❌ Лишняя нагрузка on VPN server
```

#### ✅ Со Split Tunneling

```
┌──────────────┐
│ Ваш компьютер│
└──────┬───────┘
       │
       ├─────────────────────────────┐
       │                             │
       │ Public трафик            │ Local трафик
       ▼                             ▼
┌──────────────┐      ┌──────────────┐
│  VPN Server  │      │ Локальная    │
│              │      │ сеть / ISP   │
└──────┬───────┘      └──────────────┘
       │
       ▼
┌──────────────┐
│   Internet   │
└──────────────┘

Benefits:
✅ Локальные устройства доступны
✅ Streaming on полной скорости
✅ Экономия bandwidth VPN
✅ Гибкий контроль трафика
```

---

## Quick Start

### Step 1: Обновите конфигурацию

Добавьте секцию `split_tunneling` in ваш `config.yaml`:

```yaml
# config.yaml

connection:
  from_raw_urls:
    - "https://example.com/servers.txt"
  enable_ipv6: true
  enable_dns_protection: true
  enable_kill_switch: true

# Новая секция: Split Tunneling
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16" # Local network
    - "10.0.0.0/8" # Corporate network
```

### Step 2: Перезапустите GoXRay

```bash
# Остановите текущий процесс
sudo systemctl stop goxray

# Запустите with новой конфигурацией
sudo systemctl start goxray

# Проверьте логи
sudo journalctl -u goxray -f | grep "split tunnel"
```

### Step 3: Проверьте работу

```bash
# Verification 1: Local network доступна directly
ping 192.168.1.1
# Должно работать быстро (< 1ms)

# Verification 2: Public интернет through VPN
curl https://api.ipify.org
# Должен показать VPN server IP

# Verification 3: Routes
ip route show
# Должны быть маршруты for 192.168.0.0/16 через gateway
```

---

## Operating Modes

### Mode 1: Exclude (Исключение)

**Description**: Весь трафик through VPN, **КРОМЕ** указанных сетей.

**When to use**:

- Нужна защита for большинства трафика
- Локальные устройства должны быть доступны
- Определенные сервисы (Netflix) должны идти directly

**Configuration**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16" # Local network
    - "10.0.0.0/8" # Corporate network
    - "185.60.216.0/22" # Netflix CDN (опционально)
```

**Поведение**:

```
Destination          | Route
---------------------|------------------
192.168.1.100        | Direct (excluded)
10.5.10.50           | Direct (excluded)
8.8.8.8              | VPN (not excluded)
google.com           | VPN (not excluded)
```

---

### Mode 2: Include (Включение)

**Description**: Только указанные сети through VPN, **остальное** directly.

**When to use**:

- Нужна защита only for конкретных ресурсов
- Экономия bandwidth VPN сервера
- Минимальная нагрузка on VPN

**Configuration**:

```yaml
split_tunneling:
  enabled: true
  mode: "include"
  include_cidrs:
    - "93.184.216.34/32" # Конкретный сервер
    - "142.250.0.0/15" # Google range
```

**Поведение**:

```
Destination          | Route
---------------------|------------------
93.184.216.34        | VPN (included)
142.250.10.50        | VPN (included)
192.168.1.100        | Direct (not included)
8.8.8.8              | Direct (not included)
```

---

## Examples конфигурации

### Example 1: Home network + VPN

**Task**: Access to локальным устройствам (принтер, NAS), остальное through VPN.

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    # Локальная домашняя сеть
    - "192.168.0.0/16"

    # Link-local (for mDNS, Bonjour)
    - "169.254.0.0/16"

    # Multicast (for DLNA, Chromecast)
    - "224.0.0.0/4"
```

**Result**:

- ✅ Принтер (192.168.1.100) доступен
- ✅ NAS (192.168.1.50) доступен
- ✅ Chromecast работает
- ✅ Internet through VPN

---

### Example 2: Corporate network

**Task**: Access to корпоративным ресурсам directly, остальное through VPN.

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    # Corporate network
    - "10.0.0.0/8"

    # Local network офиса
    - "192.168.0.0/16"

    # VPN корпоративный (if есть)
    - "172.16.0.0/12"
```

**Result**:

- ✅ Корпоративные серверы (10.x.x.x) доступны
- ✅ Локальные принтеры работают
- ✅ Public интернет through VPN (защита)

---

### Example 3: Streaming + Privacy

**Task**: Streaming сервисы directly (скорость), остальное through VPN (приватность).

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    # Local network
    - "192.168.0.0/16"

    # Netflix CDN
    - "185.60.216.0/22"
    - "185.60.218.0/23"

    # YouTube / Google
    - "172.217.0.0/16"
    - "216.58.192.0/19"

    # Twitch
    - "151.101.0.0/16"
```

**Result**:

- ✅ Netflix on полной скорости ISP
- ✅ YouTube without буферизации
- ✅ Twitch streams быстро
- ✅ Остальной трафик through VPN

---

### Example 4: Selective Privacy (Include Mode)

**Task**: Только критичные сервисы through VPN, остальное directly.

```yaml
split_tunneling:
  enabled: true
  mode: "include"
  include_cidrs:
    # Банковские сервисы (пример)
    - "93.184.216.0/24"

    # Email серверы (Gmail)
    - "142.250.0.0/15"

    # Конкретные заблокированные сайты
    - "104.16.0.0/12" # Cloudflare range
```

**Result**:

- ✅ Банкинг through VPN (защита)
- ✅ Email through VPN (приватность)
- ✅ Остальное directly (скорость)

---

### Example 5: Docker + Kubernetes

**Task**: Docker контейнеры and K8s pods не должны идти through VPN.

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    # Docker default bridge
    - "172.17.0.0/16"

    # Docker custom networks
    - "172.18.0.0/16"
    - "172.19.0.0/16"

    # Kubernetes pod network (пример)
    - "10.244.0.0/16"

    # Kubernetes service network
    - "10.96.0.0/12"
```

**Result**:

- ✅ Docker контейнеры общаются directly
- ✅ K8s pods доступны
- ✅ Внешний трафик through VPN

---

## Practical Scenarios

### Scenario 1: Работа from дома

**Situation**:

```
Вы работаете from дома, подключены to VPN for обхода блокировок.
Но нужен доступ to:
- Домашнему принтеру (192.168.1.100)
- NAS серверу (192.168.1.50)
- Smart TV (192.168.1.200)
```

**Solution**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16" # Вся домашняя сеть
```

**Verification**:

```bash
# Принтер доступен
ping 192.168.1.100
# PING 192.168.1.100: 56 data bytes
# 64 bytes from 192.168.1.100: icmp_seq=0 ttl=64 time=0.5 ms ✅

# Internet through VPN
curl https://api.ipify.org
# 203.0.113.45 (VPN server IP) ✅
```

---

### Scenario 2: Gaming + VPN

**Situation**:

```
Вы играете in онлайн игры, но нужен VPN for доступа to заблокированным сайтам.
Issue: Игровые серверы медленные through VPN.
```

**Solution**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    # Steam CDN
    - "155.133.0.0/16"

    # Epic Games
    - "23.32.0.0/11"

    # Battle.net
    - "24.105.0.0/16"

    # Local network
    - "192.168.0.0/16"
```

**Result**:

- ✅ Игры on низком ping (direct)
- ✅ Веб-серфинг through VPN (защита)

---

### Scenario 3: Разработчик with Docker

**Situation**:

```
Вы разработчик, используете Docker for локальной разработки.
Docker контейнеры не должны идти through VPN (медленно).
```

**Solution**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    # Docker networks
    - "172.17.0.0/16"
    - "172.18.0.0/16"
    - "172.19.0.0/16"

    # Localhost
    - "127.0.0.0/8"

    # Local network
    - "192.168.0.0/16"
```

**Verification**:

```bash
# Docker контейнер доступен directly
docker run -d -p 8080:80 nginx
curl http://localhost:8080
# Быстрый ответ ✅

# Внешний API through VPN
curl https://api.github.com
# Through VPN ✅
```

---

## Integration with Other Features

### Split Tunneling + Kill Switch

**Вопрос**: Что происходит when разрыве VPN?

**Ответ**: Kill Switch блокирует **only VPN трафик**, excluded маршруты продолжают работать.

**Поведение**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16"

connection:
  enable_kill_switch: true
```

**При разрыве VPN**:

```
Destination          | Kill Switch Active | Доступность
---------------------|--------------------|--------------
192.168.1.100        | Whitelisted        | ✅ Available
10.0.0.50            | Blocked            | ❌ Заблокирован
8.8.8.8              | Blocked            | ❌ Заблокирован
```

**Логика**:

1. VPN разрывается
2. Kill Switch активируется
3. Excluded CIDRs добавляются in whitelist Kill Switch
4. Local network продолжает работать
5. Public интернет заблокирован (защита IP)

---

### Split Tunneling + DNS Protection

**Вопрос**: How it works DNS for excluded доменов?

**Ответ**: **Все DNS запросы идут through VPN**, даже for excluded destinations.

**Причина**: Предотвращение DNS leaks.

**Example**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "185.60.216.0/22" # Netflix CDN

connection:
  enable_dns_protection: true
```

**Поведение**:

```
1. Browser запрашивает netflix.com
2. DNS запрос → VPN (защищено) ✅
3. DNS ответ: 185.60.216.35
4. Traffic to 185.60.216.35 → Direct (excluded) ✅
```

**Result**: DNS защищен, трафик быстрый.

---

### Split Tunneling + IPv6

**Support**: ✅ Полная

**Configuration**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    # IPv4
    - "192.168.0.0/16"

    # IPv6
    - "fd00::/8" # Unique Local Address
    - "fe80::/10" # Link-local
    - "ff00::/8" # Multicast

connection:
  enable_ipv6: true
```

**Verification**:

```bash
# IPv4 локальная сеть
ping 192.168.1.1
# ✅ Direct

# IPv6 локальная сеть
ping6 fd00::1
# ✅ Direct

# IPv6 публичный интернет
ping6 2001:4860:4860::8888
# ✅ Через VPN
```

---

## Monitoring and Debugging

### Verification маршрутов

```bash
# Показать all маршруты
ip route show

# Expected output (exclude mode):
# 192.168.0.0/16 via 192.168.1.1 dev eth0  ← Excluded (direct)
# 10.0.0.0/8 via 192.168.1.1 dev eth0      ← Excluded (direct)
# 0.0.0.0/1 dev tun0                        ← VPN
# 128.0.0.0/1 dev tun0                      ← VPN
```

### Verification IPv6 маршрутов

```bash
# IPv6 маршруты
ip -6 route show

# Expected output:
# fd00::/8 via fe80::1 dev eth0             ← Excluded (direct)
# ::/1 dev tun0                             ← VPN
# 8000::/1 dev tun0                         ← VPN
```

### Логи Split Tunneling

```bash
# Все логи split tunneling
sudo journalctl -u goxray | grep -i "split"

# Expected messages:
# "Split tunnel router initialized" mode=exclude exclude_count=3
# "Configuring exclude mode split tunneling"
# "Added excluded route (direct)" route=192.168.0.0/16 gateway=192.168.1.1
# "Exclude mode configured" excluded_routes=3
```

### Testing маршрутизации

```bash
# Тест 1: Local network (должна идти direct)
traceroute 192.168.1.1
# 1  192.168.1.1 (192.168.1.1)  0.5 ms ✅

# Тест 2: Public интернет (should идти through VPN)
traceroute 8.8.8.8
# 1  192.18.0.1 (192.18.0.1)  1.2 ms  ← TUN device
# 2  * * *
# 3  <VPN server IP> ✅

# Тест 3: Verification IP addresses
curl https://api.ipify.org
# <VPN server public IP> ✅
```

### Prometheus метрики

```bash
# Получить метрики
curl http://localhost:9090/metrics | grep split_tunnel

# Expected metrics:
# split_tunnel_routes_vpn_total 1234
# split_tunnel_routes_direct_total 567
# goxray_config_split_tunnel_enabled 1
```

---

## Troubleshooting

### Issue 1: Local network недоступна

**Symptoms**:

```bash
ping 192.168.1.1
# Request timeout ❌
```

**Diagnosis**:

```bash
# Verification 1: Split tunneling включен?
grep "split_tunneling" /etc/goxray/config.yaml
# enabled: true ✅

# Verification 2: Routes добавлены?
ip route show | grep 192.168
# 192.168.0.0/16 via 192.168.1.1 dev eth0 ✅

# Verification 3: Логи
sudo journalctl -u goxray | grep "excluded route"
# "Added excluded route (direct)" route=192.168.0.0/16 ✅
```

**Solution**:

1. Проверьте CIDR in конфигурации:

   ```yaml
   exclude_cidrs:
     - "192.168.0.0/16" # Должен покрывать 192.168.1.1
   ```

2. Перезапустите GoXRay:
   ```bash
   sudo systemctl restart goxray
   ```

---

### Issue 2: Весь трафик идет through VPN

**Symptoms**:

```bash
traceroute 192.168.1.1
# 1  192.18.0.1 (TUN device) ❌ Не should быть
```

**Diagnosis**:

```bash
# Verification режима
grep "mode:" /etc/goxray/config.yaml
# mode: "exclude" ✅

# Verification маршрутов
ip route show | grep 192.168
# (пусто) ❌ Routes не добавлены
```

**Solution**:

1. Проверьте валидность CIDR:

   ```bash
   # Неправильно:
   exclude_cidrs:
     - "192.168.1.0/24"  # Слишком узкий

   # Правильно:
   exclude_cidrs:
     - "192.168.0.0/16"  # Покрывает всю сеть
   ```

2. Проверьте логи on ошибки:
   ```bash
   sudo journalctl -u goxray | grep -i error
   ```

---

### Issue 3: Public интернет идет directly (не through VPN)

**Symptoms**:

```bash
curl https://api.ipify.org
# <Ваш real IP> ❌ Должен быть VPN IP
```

**Diagnosis**:

```bash
# Verification 1: Не исключили ли вы 0.0.0.0/0?
grep "0.0.0.0" /etc/goxray/config.yaml
# exclude_cidrs:
#   - "0.0.0.0/0" ❌ ОШИБКА!

# Verification 2: Mode include with пустым списком?
grep -A 5 "mode: \"include\"" /etc/goxray/config.yaml
# include_cidrs: [] ❌ ОШИБКА!
```

**Solution**:

1. **Exclude mode**: Не исключайте 0.0.0.0/0

   ```yaml
   exclude_cidrs:
     - "192.168.0.0/16" # ✅ Только локальная сеть
     # НЕ добавляйте 0.0.0.0/0!
   ```

2. **Include mode**: Добавьте нужные CIDR
   ```yaml
   mode: "include"
   include_cidrs:
     - "8.8.8.8/32" # ✅ Конкретные IP
   ```

---

### Issue 4: Kill Switch блокирует локальную сеть

**Symptoms**:

```bash
# После разрыва VPN
ping 192.168.1.1
# Request timeout ❌
```

**Diagnosis**:

```bash
# Verification Kill Switch rules
sudo iptables -L goxray_killswitch -n

# Expected output:
# Chain goxray_killswitch
# ACCEPT  all  --  0.0.0.0/0  192.168.0.0/16  ← Должно быть!
# ACCEPT  all  --  0.0.0.0/0  <XRAY_IP>
# DROP    all  --  0.0.0.0/0  0.0.0.0/0
```

**Solution**:

Это автоматически обрабатывается in коде. Если не работает:

1. Проверьте версию GoXRay:

   ```bash
   goxray --version
   # v1.7.0 or выше ✅
   ```

2. Проверьте логи интеграции:
   ```bash
   sudo journalctl -u goxray | grep "kill switch exception"
   # "Added kill switch exception for split tunnel" cidr=192.168.0.0/16 ✅
   ```

---

### Issue 5: DNS leaks for excluded доменов

**Symptoms**:

```bash
# DNS запрос идет через ISP DNS
nslookup netflix.com
# Server: 192.168.1.1 ❌ Local DNS (leak!)
```

**Solution**:

DNS Protection автоматически защищает all DNS запросы:

```yaml
connection:
  enable_dns_protection: true # ✅ Обязательно включите
```

**Verification**:

```bash
# DNS should идти through VPN
sudo tcpdump -i tun0 port 53
# Должны видеть DNS пакеты ✅
```

---

## FAQ

### Q1: Можно ли использовать домены вместо IP?

**A**: В Phase 1 (v1.7.0) - **only CIDR** (IP ranges).

**Workaround**: Резолвите домен in IP and добавьте CIDR:

```bash
# Узнать IP диапазон Netflix
nslookup netflix.com
# 185.60.216.35

# Add in конфиг
exclude_cidrs:
  - "185.60.216.0/22"  # Netflix CDN range
```

**Будущее**: Phase 2 (v1.8.0) добавит поддержку доменов:

```yaml
# Планируется in v1.8.0
exclude_domains:
  - "*.netflix.com"
  - "*.youtube.com"
```

---

### Q2: Влияет ли Split Tunneling on скорость VPN?

**A**: **Нет**, даже улучшает:

- **Excluded трафик**: Быстрее (direct, without VPN overhead)
- **VPN трафик**: Та же скорость, что and without split tunneling
- **Routing overhead**: < 1μs (negligible)

**Benchmark**:

```
Без Split Tunneling:
  Local network: 50ms (through VPN)
  Public интернет: 100ms (through VPN)

Со Split Tunneling:
  Local network: 0.5ms (direct) ✅ 100x быстрее
  Public интернет: 100ms (through VPN) ✅ Без изменений
```

---

### Q3: Безопасно ли использовать Split Tunneling?

**A**: **Да**, when правильной конфигурации:

✅ **Безопасно**:

- DNS всегда through VPN (защита от DNS leaks)
- Kill Switch работает with excluded routes
- Excluded трафик - only локальная сеть

⚠️ **Риски**:

- Если исключить публичные IP, они пойдут directly (without VPN)
- Неправильная конфигурация can привести to IP leaks

**Recommendations**:

1. Исключайте only приватные сети (192.168.x.x, 10.x.x.x)
2. Включайте DNS Protection
3. Включайте Kill Switch
4. Проверяйте конфигурацию перед применением

---

### Q4: Можно ли исключить конкретное приложение?

**A**: В Phase 1 (v1.7.0) - **нет**.

**Workaround**: Исключите IP addresses, to которым обращается приложение.

**Будущее**: Phase 3 (v1.9.0) добавит per-application routing:

```yaml
# Планируется in v1.9.0
split_tunneling:
  exclude_applications:
    - "steam"
    - "spotify"
```

---

### Q5: Working ли Split Tunneling with IPv6?

**A**: **Да**, полная поддержка IPv6:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    # IPv4
    - "192.168.0.0/16"

    # IPv6
    - "fd00::/8" # ULA
    - "fe80::/10" # Link-local
```

**Verification**:

```bash
# IPv6 маршруты
ip -6 route show | grep fd00
# fd00::/8 via fe80::1 dev eth0 ✅
```

---

### Q6: Что делать if конфигурация не валидна?

**A**: GoXRay проверяет конфигурацию when старте:

```bash
# Запуск with невалидной конфигурацией
sudo goxray --config config.yaml

# Error:
# FATAL: invalid split tunnel mode: invalid (must be 'exclude', 'include', or 'disabled')
```

**Solution**:

1. Проверьте синтаксис YAML:

   ```bash
   yamllint config.yaml
   ```

2. Проверьте CIDR формат:

   ```bash
   # Правильно:
   - "192.168.0.0/16"

   # Неправильно:
   - "192.168.0.0"      # Нет маски
   - "192.168.0.0/33"   # Неверная маска
   ```

3. Проверьте режим:
   ```yaml
   mode: "exclude"  # ✅ Правильно
   mode: "EXCLUDE"  # ❌ Неправильно (case-sensitive)
   ```

---

### Q7: Как узнать какие CIDR использует сервис?

**A**: Несколько способов:

**Метод 1: nslookup + whois**

```bash
# Узнать IP
nslookup netflix.com
# 185.60.216.35

# Узнать CIDR
whois 185.60.216.35 | grep CIDR
# CIDR: 185.60.216.0/22
```

**Метод 2: BGP looking glass**

```bash
# Онлайн сервисы:
# - https://bgp.he.net/
# - https://www.robtex.com/
```

**Метод 3: Monitoring трафика**

```bash
# Start VPN
sudo goxray --config config.yaml

# Открыть сервис (например, Netflix)
# В другом терминале:
sudo tcpdump -i tun0 -n | grep -v "192.168"
# Записать IP addresses, добавить in exclude_cidrs
```

---

## Заключение

### Основные моменты

1. ✅ **Split Tunneling** - мощный инструмент for гибкой маршрутизации
2. ✅ **Exclude mode** - for большинства случаев (локальная сеть + VPN)
3. ✅ **Include mode** - for минимальной нагрузки on VPN
4. ✅ **Security** - DNS Protection + Kill Switch работают вместе
5. ✅ **Performance** - Excluded трафик быстрее

### Recommendations

**Для домашнего использования**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16"
    - "169.254.0.0/16"
    - "224.0.0.0/4"
```

**Для корпоративного использования**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "10.0.0.0/8"
    - "172.16.0.0/12"
    - "192.168.0.0/16"
```

**Для максимальной приватности**:

```yaml
split_tunneling:
  enabled: true
  mode: "include"
  include_cidrs:
    - "<only критичные IP>"
```

### Дополнительные ресурсы

- [SPLIT_TUNNELING_DESIGN.md](SPLIT_TUNNELING_DESIGN.md) - Техническая документация
- [README.md](README.md) - Общая документация GoXRay
- [KILLSWITCH_USAGE.md](KILLSWITCH_USAGE.md) - Kill Switch руководство

---

**Version**: v1.7.0  
**Статус**: ✅ Ready to использованию  
**Support**: Phase 1 (Route-Based CIDR)  
**Следующая фаза**: v1.8.0 (Domain-Based Routing)
