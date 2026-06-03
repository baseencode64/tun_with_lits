# 🚀 GoXRay - Быстрый старт

Руководство по быстрой установке и запуску GoXRay VPN клиента.

---

## 📋 Требования

- **ОС**: Linux (Debian 13, Ubuntu 24.10+, или совместимые)
- **Архитектура**: AMD64
- **Права**: root/sudo доступ
- **Зависимости**: iptables, iproute2

Подробнее: [Системные требования](SYSTEM_REQUIREMENTS.md)

---

## ⚡ Быстрая установка

### Шаг 1: Скачать бинарник

```bash
# Скачать последнюю версию
wget https://github.com/baseencode64/tun_with_lits/releases/download/v1.7.0/goxray_v1.7.0_linux_amd64

# Сделать исполняемым
chmod +x goxray_v1.7.0_linux_amd64

# Переместить в системную директорию
sudo mv goxray_v1.7.0_linux_amd64 /usr/local/bin/goxray
```

### Шаг 2: Установить capabilities

```bash
# Необходимо для работы с TUN устройством
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
```

### Шаг 3: Создать конфигурацию

```bash
# Создать директорию для конфигурации
sudo mkdir -p /etc/goxray

# Создать базовый config.yaml
sudo tee /etc/goxray/config.yaml > /dev/null <<EOF
connection:
  from_raw_urls:
    - "https://raw.githubusercontent.com/your-repo/servers.txt"

  enable_kill_switch: true
  enable_ipv6: false
  enable_dns_protection: true

logging:
  level: "info"
  file: "/var/log/goxray/goxray.log"

metrics:
  enabled: true
  port: 9090
EOF
```

### Шаг 4: Запустить

```bash
# Запуск в foreground (для тестирования)
sudo goxray --config /etc/goxray/config.yaml

# Или как systemd service (рекомендуется)
# См. раздел "Systemd Service" ниже
```

---

## 🔧 Systemd Service (рекомендуется)

### Создать service файл

```bash
sudo tee /etc/systemd/system/goxray.service > /dev/null <<EOF
[Unit]
Description=GoXRay VPN Client
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/goxray --config /etc/goxray/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF
```

### Запустить service

```bash
# Перезагрузить systemd
sudo systemctl daemon-reload

# Включить автозапуск
sudo systemctl enable goxray

# Запустить
sudo systemctl start goxray

# Проверить статус
sudo systemctl status goxray

# Просмотр логов
sudo journalctl -u goxray -f
```

---

## ✅ Проверка работы

### 1. Проверить подключение

```bash
# Проверить статус VPN
sudo systemctl status goxray

# Ожидаемый вывод:
# ● goxray.service - GoXRay VPN Client
#    Active: active (running)
```

### 2. Проверить TUN интерфейс

```bash
# Проверить наличие TUN устройства
ip addr show tun0

# Ожидаемый вывод:
# tun0: <POINTOPOINT,MULTICAST,NOARP,UP,LOWER_UP> mtu 1500
#     inet 192.18.0.1/32 scope global tun0
```

### 3. Проверить маршруты

```bash
# Проверить маршруты через TUN
ip route show | grep tun0

# Ожидаемый вывод:
# 0.0.0.0/1 dev tun0 scope link
# 128.0.0.0/1 dev tun0 scope link
```

### 4. Проверить публичный IP

```bash
# Проверить IP (должен показать IP VPN сервера)
curl https://api.ipify.org

# Или
curl https://ifconfig.me
```

### 5. Проверить метрики (если включены)

```bash
# Открыть в браузере
http://your-server-ip:9090/metrics

# Или через curl
curl http://localhost:9090/metrics | grep vpn_connected
# Ожидаемый вывод: vpn_connected 1
```

---

## 🎯 Основные команды

```bash
# Запустить VPN
sudo systemctl start goxray

# Остановить VPN
sudo systemctl stop goxray

# Перезапустить VPN
sudo systemctl restart goxray

# Проверить статус
sudo systemctl status goxray

# Просмотр логов (real-time)
sudo journalctl -u goxray -f

# Просмотр последних 100 строк логов
sudo journalctl -u goxray -n 100

# Проверить конфигурацию
goxray --config /etc/goxray/config.yaml --validate
```

---

## 🔧 Базовая конфигурация

### Минимальная конфигурация

```yaml
# /etc/goxray/config.yaml
connection:
  from_raw_urls:
    - "https://your-server-list-url.txt"
```

### Рекомендуемая конфигурация

```yaml
# /etc/goxray/config.yaml
connection:
  from_raw_urls:
    - "https://your-server-list-url.txt"

  # Kill Switch - защита от IP утечек
  enable_kill_switch: true

  # IPv6 поддержка (если нужна)
  enable_ipv6: false

  # DNS Protection - защита от DNS утечек
  enable_dns_protection: true

  # Health check
  health_check_interval: "30s"
  health_check_timeout: "10s"

logging:
  level: "info" # debug, info, warn, error
  file: "/var/log/goxray/goxray.log"

metrics:
  enabled: true
  port: 9090
```

### Продвинутая конфигурация (с Split Tunneling)

```yaml
# /etc/goxray/config.yaml
connection:
  from_raw_urls:
    - "https://your-server-list-url.txt"
  enable_kill_switch: true
  enable_dns_protection: true

# Split Tunneling - исключить локальную сеть
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16" # Локальная сеть
    - "10.0.0.0/8" # Корпоративная сеть

# SOCKS5 Proxy
socks5:
  enabled: true
  listen_addr: "0.0.0.0:1080"
  username: "" # Опционально
  password: "" # Опционально

logging:
  level: "info"
  file: "/var/log/goxray/goxray.log"

metrics:
  enabled: true
  port: 9090
```

---

## 🐛 Troubleshooting

### VPN не подключается

```bash
# Проверить логи
sudo journalctl -u goxray -n 100

# Проверить конфигурацию
cat /etc/goxray/config.yaml

# Проверить доступность серверов
curl https://your-server-list-url.txt
```

### Нет интернета после подключения

```bash
# Проверить TUN интерфейс
ip addr show tun0

# Проверить маршруты
ip route show

# Проверить Kill Switch
sudo iptables -L goxray_killswitch -n -v

# Если Kill Switch активен - деактивировать
sudo iptables -D OUTPUT -j goxray_killswitch
sudo iptables -F goxray_killswitch
sudo iptables -X goxray_killswitch
```

### DNS не работает

```bash
# Проверить DNS
nslookup google.com

# Проверить /etc/resolv.conf
cat /etc/resolv.conf

# Если DNS Protection включен - проверить маршруты
ip route show | grep "8.8.8.8"
```

Подробнее: [Troubleshooting Guide](../troubleshooting/COMMON_ISSUES.md)

---

## 📚 Следующие шаги

После успешного запуска:

1. **Настройте Kill Switch**: [Kill Switch Guide](../features/KILLSWITCH.md)
2. **Настройте Split Tunneling**: [Split Tunneling Guide](../features/SPLIT_TUNNELING.md)
3. **Настройте SOCKS5 Proxy**: [SOCKS5 Guide](../features/SOCKS5_PROXY.md)
4. **Настройте мониторинг**: [Health Monitoring](../features/HEALTH_MONITORING.md)
5. **Production deployment**: [Production Guide](../deployment/PRODUCTION.md)

---

## 🆘 Получение помощи

- **Документация**: [docs/README.md](../README.md)
- **FAQ**: [Troubleshooting](../troubleshooting/FAQ.md)
- **Issues**: [GitHub Issues](https://github.com/baseencode64/tun_with_lits/issues)

---

**Версия**: v1.7.0  
**Последнее обновление**: 2026-06-03
