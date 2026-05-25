# 📦 Руководство по развёртыванию - Debian 13 amd64

## Информация о бинарном файле

**Файл**: `goxray_v1.6.0_linux_amd64`  
**Размер**: ~47.3 MB  
**Платформа**: Linux amd64 (Debian 13)  
**Версия**: v1.6.0 с auto-reconnect

### Включённые возможности

✅ Полноценный XRay VPN клиент  
✅ IPv6 поддержка (флаг `--ipv6`)  
✅ Health monitoring & automatic failover  
✅ Выбор сервера из списков (raw URL)  
✅ Периодическое обновление списка серверов  
✅ Auto-reconnect с exponential backoff  
✅ Prometheus метрики  
✅ DNS leak protection

---

## 🚀 Быстрая установка

### Вариант 1: Прямое использование (рекомендуется для тестирования)

```bash
# 1. Передать бинарник на Debian 13 сервер
scp goxray_v1.6.0_linux_amd64 user@debian-server:/tmp/

# 2. Зайти на сервер
ssh user@debian-server

# 3. Сделать исполняемым и переместить
chmod +x /tmp/goxray_v1.6.0_linux_amd64
sudo mv /tmp/goxray_v1.6.0_linux_amd64 /usr/local/bin/goxray

# 4. Настроить capabilities (безопаснее чем root)
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# 5. Проверить
goxray --help
```

### Вариант 2: Установка как системный сервис

```bash
# 1. Установить бинарник (шаги 1-4 из Варианта 1)

# 2. Создать systemd сервис
sudo tee /etc/systemd/system/goxray.service > /dev/null <<'EOF'
[Unit]
Description=GoXRay VPN Client v1.6.0
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/goxray --from-raw https://your-server.com/links.txt --max-retries 0
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=goxray

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/goxray

[Install]
WantedBy=multi-user.target
EOF

# 3. Перезагрузить systemd
sudo systemctl daemon-reload

# 4. Включить автозапуск
sudo systemctl enable goxray

# 5. Запустить сервис
sudo systemctl start goxray

# 6. Проверить статус
sudo systemctl status goxray

# 7. Просмотр логов
sudo journalctl -u goxray -f
```

---

## ⚙️ Конфигурация

### Базовое использование (IPv4)

```bash
sudo goxray vless://your-vless-link-here
```

### Со списком серверов (рекомендуется)

```bash
# Автовыбор лучшего сервера
sudo goxray --from-raw https://example.com/links.txt

# С периодическим обновлением каждые 10 минут
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 10m

# С авто-переподключением
sudo goxray --from-raw https://example.com/links.txt --max-retries 0
```

### С IPv6 поддержкой

```bash
sudo goxray --from-raw https://example.com/links.txt --ipv6
```

### С DNS защитой

```bash
sudo goxray --from-raw https://example.com/links.txt --dns-protection
```

### Полная конфигурация

```bash
sudo goxray \
  --from-raw https://example.com/links.txt \
  --refresh-interval 10m \
  --max-servers 15 \
  --timeout 5s \
  --ipv6 \
  --dns-protection \
  --max-retries 0 \
  --metrics-port 9090 \
  --log-format json \
  --log-level info \
  --log-file /var/log/goxray/goxray.log
```

---

## 📋 Конфигурационный файл YAML

Создайте `/etc/goxray/config.yaml`:

```yaml
connection:
  from_raw_urls:
    - "https://primary.example.com/links.txt"
    - "https://backup.example.com/links.txt"
  enable_ipv6: true
  enable_dns_protection: true
  metrics_port: 9090

server_selection:
  refresh_interval: "10m"
  max_servers: 15
  timeout: "5s"

reconnection:
  max_retries: 0
  min_backoff: "5s"
  max_backoff: "5m"
  backoff_factor: 2.0

logging:
  format: "json"
  level: "info"
  file: "/var/log/goxray/goxray.log"
  max_size: 100
  max_backups: 3
  max_age: 28

health_monitoring:
  check_interval: "10s"
  timeout: "5s"
  max_retries: 3
```

Запуск:

```bash
sudo goxray --config /etc/goxray/config.yaml
```

---

## 🔧 Требования к системе

### Зависимости

```bash
sudo apt update
sudo apt install -y iproute2 iputils-ping curl ca-certificates

# Проверка поддержки TUN
sudo modprobe tun
lsmod | grep tun
```

### Права доступа

Вариант 1 (рекомендуемый) — capabilities:

```bash
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
```

Вариант 2 — sudo:

```bash
sudo goxray --from-raw https://example.com/links.txt
```

---

## 📊 Мониторинг

### Prometheus метрики

```bash
# Включить метрики на порту 9090
sudo goxray --from-raw https://example.com/links.txt --metrics-port 9090

# Просмотр метрик
curl http://localhost:9090/metrics
```

### Логирование

```bash
# JSON формат с ротацией
sudo goxray \
  --from-raw https://example.com/links.txt \
  --log-format json \
  --log-file /var/log/goxray/goxray.log

# Просмотр логов сервиса
sudo journalctl -u goxray -f
```

### Health статус

Статус здоровья выводится в логи каждые 30 секунд:

```
INFO VPN Health Status status={"connected":true,"current_server_idx":0,...}
```

---

## 🔍 Диагностика

### Проверка подключения

```bash
# Проверить маршруты
ip route show
ip -6 route show

# Проверить DNS
ping -c 4 google.com
nslookup google.com

# Проверить внешний IP
curl -s https://api.ipify.org
```

### Проверка логов

```bash
# Все логи
sudo journalctl -u goxray -n 100

# Фильтр по health check
sudo journalctl -u goxray | grep "Health check"

# Фильтр по failover
sudo journalctl -u goxray | grep "failover\|Failover"

# Фильтр по реконнекту
sudo journalctl -u goxray | grep "Reconnection\|Reconnector"
```

---

## 🐳 Docker

Сборка Docker образа:

```dockerfile
FROM debian:13-slim
COPY goxray_v1.6.0_linux_amd64 /usr/local/bin/goxray
RUN chmod +x /usr/local/bin/goxray && \
    apt update && apt install -y --no-install-recommends \
    iproute2 iptables ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
ENTRYPOINT ["goxray"]
CMD ["--help"]
```

---

## 🔒 Безопасность

1. **Всегда используйте HTTPS** для загрузки списков серверов
2. **Регулярно обновляйте** бинарный файл
3. **Используйте capabilities** вместо запуска от root
4. **Проверяйте целостность** бинарного файла
5. **Настройте файрвол** для ограничения доступа к метрикам

---

## 📚 Дополнительная документация

- [README_RU.md](README_RU.md) — Общая документация
- [CLI_FLAGS.md](CLI_FLAGS.md) — Полный справочник CLI
- [HEALTH_MONITORING.md](HEALTH_MONITORING.md) — Система мониторинга здоровья
- [PERIODIC_REFRESH.md](PERIODIC_REFRESH.md) — Периодическое обновление
- [CHANGELOG_RU.md](CHANGELOG_RU.md) — Журнал изменений
