# 🔀 Split Tunneling - User Guide

**Version**: v1.7.0  
**Status**: 📋 Documentation  
**Date**: 2026-06-01

---

## 📖 Содержание

1. [Что такое Split Tunneling?](#что-такое-split-tunneling)
2. [Быстрый старт](#быстрый-старт)
3. [Режимы работы](#режимы-работы)
4. [Примеры конфигурации](#примеры-конфигурации)
5. [Практические сценарии](#практические-сценарии)
6. [Интеграция с другими функциями](#интеграция-с-другими-функциями)
7. [Мониторинг и отладка](#мониторинг-и-отладка)
8. [Troubleshooting](#troubleshooting)
9. [FAQ](#faq)

---

## Что такое Split Tunneling?

### Определение

**Split Tunneling** (раздельное туннелирование) - это технология, которая позволяет **выборочно** направлять сетевой трафик:

- **Часть трафика** → через VPN туннель (защищено, анонимно)
- **Часть трафика** → напрямую через ISP (быстрее, без VPN)

### Зачем это нужно?

#### ❌ Без Split Tunneling

```
┌──────────────┐
│ Ваш компьютер│
└──────┬───────┘
       │
       │ ВСЁ через VPN
       ▼
┌──────────────┐      ┌──────────────┐
│  VPN Сервер  │─────▶│   Интернет   │
└──────────────┘      └──────────────┘

Проблемы:
❌ Локальный принтер недоступен
❌ Netflix медленный (VPN далеко)
❌ Лишняя нагрузка на VPN сервер
```

#### ✅ Со Split Tunneling

```
┌──────────────┐
│ Ваш компьютер│
└──────┬───────┘
       │
       ├─────────────────────────────┐
       │                             │
       │ Публичный трафик            │ Локальный трафик
       ▼                             ▼
┌──────────────┐      ┌──────────────┐
│  VPN Сервер  │      │ Локальная    │
│              │      │ сеть / ISP   │
└──────┬───────┘      └──────────────┘
       │
       ▼
┌──────────────┐
│   Интернет   │
└──────────────┘

Преимущества:
✅ Локальные устройства доступны
✅ Streaming на полной скорости
✅ Экономия bandwidth VPN
✅ Гибкий контроль трафика
```

---

## Быстрый старт

### Шаг 1: Обновите конфигурацию

Добавьте секцию `split_tunneling` в ваш `config.yaml`:

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
    - "192.168.0.0/16" # Локальная сеть
    - "10.0.0.0/8" # Корпоративная сеть
```

### Шаг 2: Перезапустите GoXRay

```bash
# Остановите текущий процесс
sudo systemctl stop goxray

# Запустите с новой конфигурацией
sudo systemctl start goxray

# Проверьте логи
sudo journalctl -u goxray -f | grep "split tunnel"
```

### Шаг 3: Проверьте работу

```bash
# Проверка 1: Локальная сеть доступна напрямую
ping 192.168.1.1
# Должно работать быстро (< 1ms)

# Проверка 2: Публичный интернет через VPN
curl https://api.ipify.org
# Должен показать IP VPN сервера

# Проверка 3: Маршруты
ip route show
# Должны быть маршруты для 192.168.0.0/16 через gateway
```

---

## Режимы работы

### Режим 1: Exclude (Исключение)

**Описание**: Весь трафик через VPN, **КРОМЕ** указанных сетей.

**Когда использовать**:

- Нужна защита для большинства трафика
- Локальные устройства должны быть доступны
- Определенные сервисы (Netflix) должны идти напрямую

**Конфигурация**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16" # Локальная сеть
    - "10.0.0.0/8" # Корпоративная сеть
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

### Режим 2: Include (Включение)

**Описание**: Только указанные сети через VPN, **остальное** напрямую.

**Когда использовать**:

- Нужна защита только для конкретных ресурсов
- Экономия bandwidth VPN сервера
- Минимальная нагрузка на VPN

**Конфигурация**:

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

## Примеры конфигурации

### Пример 1: Домашняя сеть + VPN

**Задача**: Доступ к локальным устройствам (принтер, NAS), остальное через VPN.

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    # Локальная домашняя сеть
    - "192.168.0.0/16"

    # Link-local (для mDNS, Bonjour)
    - "169.254.0.0/16"

    # Multicast (для DLNA, Chromecast)
    - "224.0.0.0/4"
```

**Результат**:

- ✅ Принтер (192.168.1.100) доступен
- ✅ NAS (192.168.1.50) доступен
- ✅ Chromecast работает
- ✅ Интернет через VPN

---

### Пример 2: Корпоративная сеть

**Задача**: Доступ к корпоративным ресурсам напрямую, остальное через VPN.

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    # Корпоративная сеть
    - "10.0.0.0/8"

    # Локальная сеть офиса
    - "192.168.0.0/16"

    # VPN корпоративный (если есть)
    - "172.16.0.0/12"
```

**Результат**:

- ✅ Корпоративные серверы (10.x.x.x) доступны
- ✅ Локальные принтеры работают
- ✅ Публичный интернет через VPN (защита)

---

### Пример 3: Streaming + Privacy

**Задача**: Streaming сервисы напрямую (скорость), остальное через VPN (приватность).

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    # Локальная сеть
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

**Результат**:

- ✅ Netflix на полной скорости ISP
- ✅ YouTube без буферизации
- ✅ Twitch streams быстро
- ✅ Остальной трафик через VPN

---

### Пример 4: Selective Privacy (Include Mode)

**Задача**: Только критичные сервисы через VPN, остальное напрямую.

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

**Результат**:

- ✅ Банкинг через VPN (защита)
- ✅ Email через VPN (приватность)
- ✅ Остальное напрямую (скорость)

---

### Пример 5: Docker + Kubernetes

**Задача**: Docker контейнеры и K8s pods не должны идти через VPN.

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

**Результат**:

- ✅ Docker контейнеры общаются напрямую
- ✅ K8s pods доступны
- ✅ Внешний трафик через VPN

---

## Практические сценарии

### Сценарий 1: Работа из дома

**Ситуация**:

```
Вы работаете из дома, подключены к VPN для обхода блокировок.
Но нужен доступ к:
- Домашнему принтеру (192.168.1.100)
- NAS серверу (192.168.1.50)
- Smart TV (192.168.1.200)
```

**Решение**:

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16" # Вся домашняя сеть
```

**Проверка**:

```bash
# Принтер доступен
ping 192.168.1.100
# PING 192.168.1.100: 56 data bytes
# 64 bytes from 192.168.1.100: icmp_seq=0 ttl=64 time=0.5 ms ✅

# Интернет через VPN
curl https://api.ipify.org
# 203.0.113.45 (IP VPN сервера) ✅
```

---

### Сценарий 2: Gaming + VPN

**Ситуация**:

```
Вы играете в онлайн игры, но нужен VPN для доступа к заблокированным сайтам.
Проблема: Игровые серверы медленные через VPN.
```

**Решение**:

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

    # Локальная сеть
    - "192.168.0.0/16"
```

**Результат**:

- ✅ Игры на низком ping (direct)
- ✅ Веб-серфинг через VPN (защита)

---

### Сценарий 3: Разработчик с Docker

**Ситуация**:

```
Вы разработчик, используете Docker для локальной разработки.
Docker контейнеры не должны идти через VPN (медленно).
```

**Решение**:

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

    # Локальная сеть
    - "192.168.0.0/16"
```

**Проверка**:

```bash
# Docker контейнер доступен напрямую
docker run -d -p 8080:80 nginx
curl http://localhost:8080
# Быстрый ответ ✅

# Внешний API через VPN
curl https://api.github.com
# Через VPN ✅
```

---

## Интеграция с другими функциями

### Split Tunneling + Kill Switch

**Вопрос**: Что происходит при разрыве VPN?

**Ответ**: Kill Switch блокирует **только VPN трафик**, excluded маршруты продолжают работать.

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
192.168.1.100        | Whitelisted        | ✅ Доступен
10.0.0.50            | Blocked            | ❌ Заблокирован
8.8.8.8              | Blocked            | ❌ Заблокирован
```

**Логика**:

1. VPN разрывается
2. Kill Switch активируется
3. Excluded CIDRs добавляются в whitelist Kill Switch
4. Локальная сеть продолжает работать
5. Публичный интернет заблокирован (защита IP)

---

### Split Tunneling + DNS Protection

**Вопрос**: Как работает DNS для excluded доменов?

**Ответ**: **Все DNS запросы идут через VPN**, даже для excluded destinations.

**Причина**: Предотвращение DNS leaks.

**Пример**:

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
1. Браузер запрашивает netflix.com
2. DNS запрос → VPN (защищено) ✅
3. DNS ответ: 185.60.216.35
4. Трафик к 185.60.216.35 → Direct (excluded) ✅
```

**Результат**: DNS защищен, трафик быстрый.

---

### Split Tunneling + IPv6

**Поддержка**: ✅ Полная

**Конфигурация**:

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

**Проверка**:

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

## Мониторинг и отладка

### Проверка маршрутов

```bash
# Показать все маршруты
ip route show

# Ожидаемый вывод (exclude mode):
# 192.168.0.0/16 via 192.168.1.1 dev eth0  ← Excluded (direct)
# 10.0.0.0/8 via 192.168.1.1 dev eth0      ← Excluded (direct)
# 0.0.0.0/1 dev tun0                        ← VPN
# 128.0.0.0/1 dev tun0                      ← VPN
```

### Проверка IPv6 маршрутов

```bash
# IPv6 маршруты
ip -6 route show

# Ожидаемый вывод:
# fd00::/8 via fe80::1 dev eth0             ← Excluded (direct)
# ::/1 dev tun0                             ← VPN
# 8000::/1 dev tun0                         ← VPN
```

### Логи Split Tunneling

```bash
# Все логи split tunneling
sudo journalctl -u goxray | grep -i "split"

# Ожидаемые сообщения:
# "Split tunnel router initialized" mode=exclude exclude_count=3
# "Configuring exclude mode split tunneling"
# "Added excluded route (direct)" route=192.168.0.0/16 gateway=192.168.1.1
# "Exclude mode configured" excluded_routes=3
```

### Тестирование маршрутизации

```bash
# Тест 1: Локальная сеть (должна идти direct)
traceroute 192.168.1.1
# 1  192.168.1.1 (192.168.1.1)  0.5 ms ✅

# Тест 2: Публичный интернет (должен идти через VPN)
traceroute 8.8.8.8
# 1  192.18.0.1 (192.18.0.1)  1.2 ms  ← TUN device
# 2  * * *
# 3  <VPN server IP> ✅

# Тест 3: Проверка IP адреса
curl https://api.ipify.org
# <VPN server public IP> ✅
```

### Prometheus метрики

```bash
# Получить метрики
curl http://localhost:9090/metrics | grep split_tunnel

# Ожидаемые метрики:
# split_tunnel_routes_vpn_total 1234
# split_tunnel_routes_direct_total 567
# goxray_config_split_tunnel_enabled 1
```

---

## Troubleshooting

### Проблема 1: Локальная сеть недоступна

**Симптомы**:

```bash
ping 192.168.1.1
# Request timeout ❌
```

**Диагностика**:

```bash
# Проверка 1: Split tunneling включен?
grep "split_tunneling" /etc/goxray/config.yaml
# enabled: true ✅

# Проверка 2: Маршруты добавлены?
ip route show | grep 192.168
# 192.168.0.0/16 via 192.168.1.1 dev eth0 ✅

# Проверка 3: Логи
sudo journalctl -u goxray | grep "excluded route"
# "Added excluded route (direct)" route=192.168.0.0/16 ✅
```

**Решение**:

1. Проверьте CIDR в конфигурации:

   ```yaml
   exclude_cidrs:
     - "192.168.0.0/16" # Должен покрывать 192.168.1.1
   ```

2. Перезапустите GoXRay:
   ```bash
   sudo systemctl restart goxray
   ```

---

### Проблема 2: Весь трафик идет через VPN

**Симптомы**:

```bash
traceroute 192.168.1.1
# 1  192.18.0.1 (TUN device) ❌ Не должно быть
```

**Диагностика**:

```bash
# Проверка режима
grep "mode:" /etc/goxray/config.yaml
# mode: "exclude" ✅

# Проверка маршрутов
ip route show | grep 192.168
# (пусто) ❌ Маршруты не добавлены
```

**Решение**:

1. Проверьте валидность CIDR:

   ```bash
   # Неправильно:
   exclude_cidrs:
     - "192.168.1.0/24"  # Слишком узкий

   # Правильно:
   exclude_cidrs:
     - "192.168.0.0/16"  # Покрывает всю сеть
   ```

2. Проверьте логи на ошибки:
   ```bash
   sudo journalctl -u goxray | grep -i error
   ```

---

### Проблема 3: Публичный интернет идет напрямую (не через VPN)

**Симптомы**:

```bash
curl https://api.ipify.org
# <Ваш реальный IP> ❌ Должен быть VPN IP
```

**Диагностика**:

```bash
# Проверка 1: Не исключили ли вы 0.0.0.0/0?
grep "0.0.0.0" /etc/goxray/config.yaml
# exclude_cidrs:
#   - "0.0.0.0/0" ❌ ОШИБКА!

# Проверка 2: Режим include с пустым списком?
grep -A 5 "mode: \"include\"" /etc/goxray/config.yaml
# include_cidrs: [] ❌ ОШИБКА!
```

**Решение**:

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

### Проблема 4: Kill Switch блокирует локальную сеть

**Симптомы**:

```bash
# После разрыва VPN
ping 192.168.1.1
# Request timeout ❌
```

**Диагностика**:

```bash
# Проверка Kill Switch rules
sudo iptables -L goxray_killswitch -n

# Ожидаемый вывод:
# Chain goxray_killswitch
# ACCEPT  all  --  0.0.0.0/0  192.168.0.0/16  ← Должно быть!
# ACCEPT  all  --  0.0.0.0/0  <XRAY_IP>
# DROP    all  --  0.0.0.0/0  0.0.0.0/0
```

**Решение**:

Это автоматически обрабатывается в коде. Если не работает:

1. Проверьте версию GoXRay:

   ```bash
   goxray --version
   # v1.7.0 или выше ✅
   ```

2. Проверьте логи интеграции:
   ```bash
   sudo journalctl -u goxray | grep "kill switch exception"
   # "Added kill switch exception for split tunnel" cidr=192.168.0.0/16 ✅
   ```

---

### Проблема 5: DNS leaks для excluded доменов

**Симптомы**:

```bash
# DNS запрос идет через ISP DNS
nslookup netflix.com
# Server: 192.168.1.1 ❌ Локальный DNS (leak!)
```

**Решение**:

DNS Protection автоматически защищает все DNS запросы:

```yaml
connection:
  enable_dns_protection: true # ✅ Обязательно включите
```

**Проверка**:

```bash
# DNS должен идти через VPN
sudo tcpdump -i tun0 port 53
# Должны видеть DNS пакеты ✅
```

---

## FAQ

### Q1: Можно ли использовать домены вместо IP?

**A**: В Phase 1 (v1.7.0) - **только CIDR** (IP ranges).

**Workaround**: Резолвите домен в IP и добавьте CIDR:

```bash
# Узнать IP диапазон Netflix
nslookup netflix.com
# 185.60.216.35

# Добавить в конфиг
exclude_cidrs:
  - "185.60.216.0/22"  # Netflix CDN range
```

**Будущее**: Phase 2 (v1.8.0) добавит поддержку доменов:

```yaml
# Планируется в v1.8.0
exclude_domains:
  - "*.netflix.com"
  - "*.youtube.com"
```

---

### Q2: Влияет ли Split Tunneling на скорость VPN?

**A**: **Нет**, даже улучшает:

- **Excluded трафик**: Быстрее (direct, без VPN overhead)
- **VPN трафик**: Та же скорость, что и без split tunneling
- **Routing overhead**: < 1μs (negligible)

**Benchmark**:

```
Без Split Tunneling:
  Локальная сеть: 50ms (через VPN)
  Публичный интернет: 100ms (через VPN)

Со Split Tunneling:
  Локальная сеть: 0.5ms (direct) ✅ 100x быстрее
  Публичный интернет: 100ms (через VPN) ✅ Без изменений
```

---

### Q3: Безопасно ли использовать Split Tunneling?

**A**: **Да**, при правильной конфигурации:

✅ **Безопасно**:

- DNS всегда через VPN (защита от DNS leaks)
- Kill Switch работает с excluded routes
- Excluded трафик - только локальная сеть

⚠️ **Риски**:

- Если исключить публичные IP, они пойдут напрямую (без VPN)
- Неправильная конфигурация может привести к IP leaks

**Рекомендации**:

1. Исключайте только приватные сети (192.168.x.x, 10.x.x.x)
2. Включайте DNS Protection
3. Включайте Kill Switch
4. Проверяйте конфигурацию перед применением

---

### Q4: Можно ли исключить конкретное приложение?

**A**: В Phase 1 (v1.7.0) - **нет**.

**Workaround**: Исключите IP адреса, к которым обращается приложение.

**Будущее**: Phase 3 (v1.9.0) добавит per-application routing:

```yaml
# Планируется в v1.9.0
split_tunneling:
  exclude_applications:
    - "steam"
    - "spotify"
```

---

### Q5: Работает ли Split Tunneling с IPv6?

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

**Проверка**:

```bash
# IPv6 маршруты
ip -6 route show | grep fd00
# fd00::/8 via fe80::1 dev eth0 ✅
```

---

### Q6: Что делать если конфигурация не валидна?

**A**: GoXRay проверяет конфигурацию при старте:

```bash
# Запуск с невалидной конфигурацией
sudo goxray --config config.yaml

# Ошибка:
# FATAL: invalid split tunnel mode: invalid (must be 'exclude', 'include', or 'disabled')
```

**Решение**:

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

**Метод 3: Мониторинг трафика**

```bash
# Запустить VPN
sudo goxray --config config.yaml

# Открыть сервис (например, Netflix)
# В другом терминале:
sudo tcpdump -i tun0 -n | grep -v "192.168"
# Записать IP адреса, добавить в exclude_cidrs
```

---

## Заключение

### Основные моменты

1. ✅ **Split Tunneling** - мощный инструмент для гибкой маршрутизации
2. ✅ **Exclude mode** - для большинства случаев (локальная сеть + VPN)
3. ✅ **Include mode** - для минимальной нагрузки на VPN
4. ✅ **Безопасность** - DNS Protection + Kill Switch работают вместе
5. ✅ **Производительность** - Excluded трафик быстрее

### Рекомендации

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
    - "<только критичные IP>"
```

### Дополнительные ресурсы

- [SPLIT_TUNNELING_DESIGN.md](SPLIT_TUNNELING_DESIGN.md) - Техническая документация
- [README.md](README.md) - Общая документация GoXRay
- [KILLSWITCH_USAGE.md](KILLSWITCH_USAGE.md) - Kill Switch руководство

---

**Версия**: v1.7.0  
**Статус**: ✅ Готово к использованию  
**Поддержка**: Phase 1 (Route-Based CIDR)  
**Следующая фаза**: v1.8.0 (Domain-Based Routing)
