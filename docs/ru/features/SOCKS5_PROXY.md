# 🪟 Windows VPN Proxy Guide - SOCKS5 Proxy через GoXRay

**Version**: v1.7.0  
**Date**: 2026-06-01  
**Platform**: Windows 10/11, Linux

---

## 📖 Содержание

1. [Обзор](#обзор)
2. [Встроенный SOCKS5 сервер](#встроенный-socks5-сервер)
3. [Конфигурация](#конфигурация)
4. [Использование](#использование)
5. [Аутентификация](#аутентификация)
6. [Безопасность](#безопасность)
7. [Troubleshooting](#troubleshooting)

---

## Обзор

GoXRay VPN Client теперь включает **встроенный SOCKS5 прокси-сервер**, который позволяет приложениям маршрутизировать трафик через VPN туннель.

### Зачем нужен SOCKS5 прокси?

Некоторые приложения могут не автоматически использовать TUN интерфейс, созданный GoXRay. Это особенно актуально для:

- ✅ Приложений, которые не учитывают системные таблицы маршрутизации
- ✅ Docker контейнеров
- ✅ WSL2 (Windows Subsystem for Linux)
- ✅ Некоторых браузеров и менеджеров загрузок
- ✅ Приложений, требующих явной настройки прокси

### Преимущества встроенного SOCKS5

- ✅ **Не требуется внешний прокси-сервер** (Dante, SSH tunnel, Privoxy)
- ✅ **Автоматический запуск/остановка** вместе с VPN
- ✅ **Трафик автоматически идет через VPN** туннель
- ✅ **Поддержка аутентификации** username/password
- ✅ **Graceful shutdown** при остановке VPN
- ✅ **Подробное логирование** всех событий

---

## Встроенный SOCKS5 сервер

### Архитектура

```
Windows/Linux Application
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

**Ключевая особенность:** SOCKS5 сервер запускается **внутри GoXRay** и автоматически маршрутизирует весь трафик через VPN туннель.

---

## Конфигурация

### 1. Включить SOCKS5 в config.yaml

```yaml
# SOCKS5 proxy server (optional)
# Allows Windows applications to route traffic through VPN via SOCKS5 proxy
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

### 2. Запустить GoXRay VPN

```bash
# С конфигурационным файлом
./goxray --config config.yaml

# Или с CLI аргументами + from-raw
./goxray --from-raw https://example.com/links.txt
```

**SOCKS5 сервер запустится автоматически** после успешного VPN подключения!

### 3. Проверить логи

Вы увидите в логах:

```
INFO VPN client connected successfully tun_address=192.18.0.1/32 xray_server=1.2.3.4:443
INFO Starting SOCKS5 proxy server address=0.0.0.0:1080
INFO SOCKS5 proxy server started successfully address=0.0.0.0:1080 auth="no authentication" timeout=30s
```

---

## Использование

### Тестирование SOCKS5

#### Windows PowerShell тестовый скрипт

GoXRay включает PowerShell скрипт для тестирования на Windows:

```powershell
# Запустить тестовый скрипт
.\test_socks5.ps1
```

**Скрипт проверит:**

- ✅ Слушает ли SOCKS5 порт
- ✅ Ваш реальный IP (без прокси)
- ✅ SOCKS5 подключение с curl.exe
- ✅ SOCKS5 handshake протокол
- ✅ Сравнит IP адреса для подтверждения что трафик идет через VPN

**Ожидаемый вывод:**

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

#### Linux/macOS тестирование

```bash
# Проверить IP без VPN
curl https://api.ipify.org
# Вывод: 203.0.113.1 (ваш реальный IP)

# Проверить IP через SOCKS5 прокси
curl --socks5 localhost:1080 https://api.ipify.org
# Вывод: 198.51.100.1 (IP VPN сервера)
```

#### С аутентификацией

```bash
# Если настроена аутентификация
curl --socks5 myuser:mypass@localhost:1080 https://api.ipify.org
```

#### Проверка с wget

```bash
# Без аутентификации
wget -e use_proxy=yes -e socks_proxy=localhost:1080 https://api.ipify.org -O -

# С аутентификацией
wget --proxy-user=myuser --proxy-password=mypass \
     -e use_proxy=yes -e socks_proxy=localhost:1080 \
     https://api.ipify.org -O -
```

### Настройка браузеров

#### Firefox

1. Открыть **Settings** → **Network Settings**
2. Выбрать **Manual proxy configuration**
3. **SOCKS Host**: `localhost`
4. **Port**: `1080`
5. **SOCKS v5**: ✓ (включить)
6. **Proxy DNS when using SOCKS v5**: ✓ (включить для защиты DNS)

#### Chrome/Edge (через расширение)

Установить **Proxy SwitchyOmega**:

1. [Chrome Web Store](https://chrome.google.com/webstore/detail/proxy-switchyomega/)
2. Создать новый профиль → **Proxy Profile**
3. **Protocol**: SOCKS5
4. **Server**: localhost
5. **Port**: 1080

#### Chrome/Edge (системные настройки)

**Windows:**

```powershell
# Открыть настройки прокси через GUI
start ms-settings:network-proxy

# Или через PowerShell (требуется Administrator)
Set-ItemProperty -Path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings" -Name ProxyEnable -Value 1
Set-ItemProperty -Path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings" -Name ProxyServer -Value "socks=localhost:1080"
```

**Примечание для пользователей Windows:**

- Windows не поддерживает SOCKS5 в системных настройках прокси нативно
- Используйте расширения браузера (Proxy SwitchyOmega) или настройки прокси в приложениях
- Для системного SOCKS5 используйте сторонние инструменты: Proxifier или ProxyCap

**Linux:**

```bash
# Через переменные окружения
export ALL_PROXY=socks5://localhost:1080
export HTTP_PROXY=socks5://localhost:1080
export HTTPS_PROXY=socks5://localhost:1080

# Запустить Chrome
google-chrome --proxy-server="socks5://localhost:1080"
```

### Настройка приложений

#### Git

```bash
# Глобально
git config --global http.proxy socks5://localhost:1080
git config --global https.proxy socks5://localhost:1080

# Для конкретного репозитория
git config http.proxy socks5://localhost:1080

# Отключить прокси
git config --global --unset http.proxy
git config --global --unset https.proxy
```

#### Docker (внутри контейнера)

```bash
# Запустить контейнер с прокси
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

## Аутентификация

### Включить аутентификацию

```yaml
socks5:
  enabled: true
  listen_addr: "0.0.0.0:1080"
  username: "myuser"
  password: "VeryStr0ngP@ssw0rd!"
  timeout: "30s"
```

После перезапуска GoXRay:

```
INFO SOCKS5 proxy server started successfully address=0.0.0.0:1080 auth="username/password authentication (user: myuser)" timeout=30s
```

### Использование с аутентификацией

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

## Безопасность

### Рекомендации

#### 1. Используйте аутентификацию для удаленного доступа

Если SOCKS5 доступен из сети (`0.0.0.0:1080`), **обязательно** используйте аутентификацию:

```yaml
socks5:
  enabled: true
  listen_addr: "0.0.0.0:1080" # Доступен из сети
  username: "stronguser"
  password: "VeryStr0ngP@ssw0rd!123"
  timeout: "30s"
```

#### 2. Ограничьте доступ только localhost

Если удаленный доступ не нужен, слушайте только на localhost:

```yaml
socks5:
  enabled: true
  listen_addr: "127.0.0.1:1080" # Только localhost
  username: ""
  password: ""
  timeout: "30s"
```

#### 3. Используйте firewall

**Linux (iptables):**

```bash
# Разрешить только локальную сеть
sudo iptables -A INPUT -p tcp --dport 1080 -s 192.168.0.0/16 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 1080 -j DROP

# Сохранить правила
sudo iptables-save > /etc/iptables/rules.v4
```

**Windows (PowerShell Administrator):**

```powershell
# Разрешить только локальную сеть
New-NetFirewallRule -DisplayName "Allow SOCKS5 Local" `
                    -Direction Inbound `
                    -LocalPort 1080 `
                    -Protocol TCP `
                    -RemoteAddress 192.168.0.0/16 `
                    -Action Allow

# Заблокировать все остальное
New-NetFirewallRule -DisplayName "Block SOCKS5 External" `
                    -Direction Inbound `
                    -LocalPort 1080 `
                    -Protocol TCP `
                    -Action Block
```

#### 4. Используйте Kill Switch

Включите Kill Switch для предотвращения утечек IP при разрыве VPN:

```yaml
connection:
  enable_kill_switch: true
```

---

## Troubleshooting

### Issue 1: SOCKS5 не запускается

**Симптомы:**

```
WARN Failed to start SOCKS5 server error="listen tcp 0.0.0.0:1080: bind: address already in use"
```

**Диагностика:**

```bash
# Linux
sudo netstat -tuln | grep 1080
sudo lsof -i :1080

# Windows
netstat -an | findstr 1080
```

**Решение:**

1. **Изменить порт:**

```yaml
socks5:
  listen_addr: "0.0.0.0:1081" # Использовать другой порт
```

2. **Остановить конфликтующий процесс:**

```bash
# Linux
sudo kill $(sudo lsof -t -i:1080)

# Windows
# Найти PID
netstat -ano | findstr 1080
# Остановить процесс
taskkill /PID <PID> /F
```

---

### Issue 2: Приложение не подключается к SOCKS5

**Симптомы:**

```
Connection refused to localhost:1080
```

**Диагностика:**

```bash
# Проверить, что SOCKS5 запущен
curl --socks5 localhost:1080 https://api.ipify.org

# Проверить логи GoXRay
./goxray --config config.yaml --log-level debug
```

**Решение:**

1. **Проверить, что VPN подключен:**

```bash
# SOCKS5 запускается ТОЛЬКО после успешного VPN подключения
# Проверить логи:
INFO VPN client connected successfully
INFO SOCKS5 proxy server started successfully
```

2. **Проверить firewall:**

```bash
# Linux
sudo iptables -L -n | grep 1080

# Windows
Get-NetFirewallRule | Where-Object {$_.LocalPort -eq 1080}
```

3. **Проверить, что порт слушается:**

```bash
# Linux
sudo netstat -tuln | grep 1080
# Должно показать: tcp 0 0 0.0.0.0:1080 0.0.0.0:* LISTEN

# Windows
netstat -an | findstr 1080
# Должно показать: TCP 0.0.0.0:1080 0.0.0.0:0 LISTENING
```

---

### Issue 3: DNS не резолвится через прокси

**Симптомы:**

```
curl: (6) Could not resolve host: example.com
```

**Решение:**

1. **Включить DNS protection в GoXRay:**

```yaml
connection:
  enable_dns_protection: true
```

2. **В браузере (Firefox):**

- Включить **"Proxy DNS when using SOCKS v5"**

3. **В curl использовать --socks5-hostname:**

```bash
# Резолвить DNS через SOCKS5
curl --socks5-hostname localhost:1080 https://example.com
```

---

### Issue 4: Медленная скорость через SOCKS5

**Диагностика:**

```bash
# Проверить скорость напрямую (без SOCKS5)
curl -o /dev/null https://speed.cloudflare.com/__down?bytes=100000000

# Проверить через SOCKS5
curl --socks5 localhost:1080 -o /dev/null https://speed.cloudflare.com/__down?bytes=100000000
```

**Решение:**

1. **Увеличить timeout:**

```yaml
socks5:
  timeout: "60s" # Увеличить с 30s до 60s
```

2. **Проверить скорость VPN сервера:**

```bash
# Проверить ping
ping <vpn-server-ip>

# Проверить через VPN
curl --socks5 localhost:1080 https://speed.cloudflare.com/cdn-cgi/trace
```

3. **Использовать Split Tunneling:**

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16" # Локальная сеть напрямую
    - "10.0.0.0/8" # Корпоративная сеть
```

---

### Issue 5: SOCKS5 работает, но IP не меняется

**Симптомы:**

```bash
curl --socks5 localhost:1080 https://api.ipify.org
# Показывает реальный IP, а не VPN
```

**Диагностика:**

```bash
# Проверить, что VPN подключен
./goxray --config config.yaml
# Должно показать: INFO VPN client connected successfully

# Проверить маршруты (Linux/macOS)
ip route show
# Должно показать маршруты через tun0
```

**Решение:**

1. **Запустить тестовый скрипт (Windows):**

```powershell
.\test_socks5.ps1
```

Скрипт автоматически диагностирует проблему и покажет идет ли трафик через VPN.

2. **Проверить что VPN действительно работает (Linux/macOS):**

```bash
# Проверить IP через TUN интерфейс
curl --interface tun0 https://api.ipify.org
# Должен показать IP VPN сервера
```

3. **Перезапустить GoXRay с debug логами:**

```bash
# Остановить
pkill goxray

# Запустить с debug логированием
./goxray --config config.yaml --log-level debug
```

4. **Проверить правила маршрутизации:**

На Linux/macOS GoXRay создает правила маршрутизации, которые направляют весь трафик (включая от SOCKS5 сервера) через TUN устройство. Проверьте что эти правила существуют:

```bash
# Linux
ip route show | grep tun0

# macOS
netstat -rn | grep tun0
```

---

## Best Practices

### 1. Используйте config.yaml

Не используйте CLI аргументы для SOCKS5 - используйте `config.yaml`:

```yaml
socks5:
  enabled: true
  listen_addr: "127.0.0.1:1080"
  username: ""
  password: ""
  timeout: "30s"
```

### 2. Комбинируйте с Split Tunneling

```yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16" # Локальная сеть напрямую
    - "10.0.0.0/8" # Корпоративная сеть

socks5:
  enabled: true
  listen_addr: "0.0.0.0:1080"
```

**Результат:**

- Локальные ресурсы доступны напрямую
- Интернет через VPN
- SOCKS5 для приложений, которые не поддерживают TUN

### 3. Используйте Kill Switch

```yaml
connection:
  enable_kill_switch: true
  enable_dns_protection: true

socks5:
  enabled: true
```

**Результат:**

- Защита от утечек IP при разрыве VPN
- Защита от утечек DNS
- SOCKS5 для явной настройки прокси

### 4. Мониторинг

```bash
# Проверить статус VPN
curl --socks5 localhost:1080 https://api.ipify.org

# Проверить метрики (если включены)
curl http://localhost:9090/metrics | grep vpn_connected

# Проверить логи
tail -f /var/log/goxray/goxray.log
```

---

## Summary

### Рекомендуемая конфигурация

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

### Результат

- ✅ Весь интернет-трафик через VPN
- ✅ Локальная сеть напрямую (Split Tunneling)
- ✅ DNS защищен от утечек
- ✅ Kill Switch предотвращает утечки IP
- ✅ SOCKS5 для приложений с явной настройкой прокси
- ✅ Простая настройка и использование

---

## Особенности для Windows

### Тестирование на Windows

GoXRay предоставляет PowerShell тестовый скрипт (`test_socks5.ps1`) специально для пользователей Windows:

```powershell
# Скачать и запустить тестовый скрипт
.\test_socks5.ps1
```

Этот скрипт:

- Тестирует SOCKS5 подключение
- Проверяет маршрутизацию трафика через VPN
- Предоставляет детальную диагностику
- Работает без curl.exe (использует нативные команды PowerShell)

### Windows Firewall

Если SOCKS5 недоступен с других машин в вашей сети:

```powershell
# Разрешить SOCKS5 через Windows Firewall
New-NetFirewallRule -DisplayName "GoXRay SOCKS5" `
                    -Direction Inbound `
                    -LocalPort 1080 `
                    -Protocol TCP `
                    -Action Allow
```

### Windows приложения

Многие Windows приложения поддерживают SOCKS5 прокси:

- **Браузеры**: Firefox (нативно), Chrome/Edge (через расширение)
- **Менеджеры загрузок**: IDM, Free Download Manager
- **Torrent клиенты**: qBittorrent, Transmission
- **Git**: Git for Windows
- **WSL2**: Настройка прокси в WSL2 окружении

---

## Поддержка

**Документация:**

- [Главный README](../../../README_RU.md) - Обзор проекта
- [Split Tunneling](SPLIT_TUNNELING.md) - Выборочная маршрутизация
- [Kill Switch](KILLSWITCH.md) - Защита от утечек IP
- [Docker Deployment](../deployment/DOCKER.md) - Настройка Docker

**Тестирование:**

- Windows: `test_socks5.ps1` (включен в репозиторий)
- Linux/macOS: `curl --socks5 localhost:1080 https://api.ipify.org`

**Issues**: Сообщайте об ошибках через [GitHub Issues](https://github.com/baseencode64/tun_with_lits/issues)

---

**Версия**: v1.7.0  
**Последнее обновление**: 2026-06-03  
**Протестировано на**: Windows 10/11, Linux (Ubuntu, Debian), macOS
