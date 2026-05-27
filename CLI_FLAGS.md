# GoXRay VPN Client - Полное руководство по флагам и конфигурации

## Обзор

GoXRay VPN Client поддерживает запуск как через аргументы командной строки, так и через YAML конфигурационные файлы. CLI аргументы имеют приоритет над значениями из конфигурационного файла.

## Способы запуска

### 1. Прямая ссылка на сервер

```bash
sudo goxray "vless://uuid@server.example.com:443?type=tcp&security=reality&..."
```

### 2. Из конфигурационного файла

```bash
sudo goxray --config /path/to/config.yaml
```

### 3. Из списка серверов (raw URL)

```bash
sudo goxray --from-raw https://example.com/links.txt
```

### 4. Через переменную окружения

```bash
export GOXRAY_CONFIG_URL="vless://uuid@server.example.com:443?..."
sudo goxray
```

---

## Все доступные флаги

### 🔗 Параметры подключения

#### Прямая ссылка

```bash
<vless://...>  # Позиционный аргумент - прямая ссылка на сервер
```

**Пример:**

```bash
sudo goxray "vless://a6b071ef-0d82-4f46-b04b-3310b8d6ca82@3.112.126.206:54055?type=tcp&security=reality&pbk=..."
```

#### Конфигурационный файл

```bash
--config <path>  # Путь к YAML конфигурационному файлу
```

**Пример:**

```bash
sudo goxray --config /etc/goxray/config.yaml
```

#### Список серверов из URL

```bash
--from-raw <url>  # URL для получения списка VLESS ссылок
```

**Пример:**

```bash
sudo goxray --from-raw https://raw.githubusercontent.com/user/repo/main/links.txt
```

---

### 🔄 Параметры обновления списка серверов

#### Интервал обновления

```bash
--refresh-interval <duration>  # Периодическое обновление списка серверов
```

**Формат:** `5m`, `10m`, `1h`, `30m` и т.д.

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 10m
```

**По умолчанию:** `0` (обновление отключено)

#### Максимальное количество серверов

```bash
--max-servers <n>  # Максимальное количество серверов для проверки
```

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --max-servers 20
```

**По умолчанию:** `10`

---

### 🔗 Параметры переподключения (Connection Persistence & Auto-Reconnect)

Управляют поведением клиента, когда все серверы в списке исчерпаны. Клиент автоматически переходит в режим переподключения с экспоненциальной задержкой.

#### Максимальное количество попыток

```bash
--max-retries <n>  # Максимальное количество попыток переподключения
```

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --max-retries 5
```

**По умолчанию:** `0` (безлимитные попытки, до остановки через Ctrl+C)

#### Минимальная задержка

```bash
--min-backoff <duration>  # Начальная задержка перед переподключением
```

**Формат:** `5s`, `10s`, `30s` и т.д.

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --min-backoff 10s
```

**По умолчанию:** `5s`

#### Максимальная задержка

```bash
--max-backoff <duration>  # Максимальная задержка перед переподключением
```

**Формат:** `5m`, `10m`, `30m` и т.д.

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --max-backoff 10m
```

**По умолчанию:** `5m`

#### Коэффициент экспоненциального роста

```bash
--backoff-factor <factor>  # Множитель для экспоненциальной задержки
```

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --backoff-factor 3.0
```

**По умолчанию:** `2.0`

**Формула расчета:**

```
backoff(n) = min(min_backoff × factor^(n-1) + jitter, max_backoff)
```

**Пример последовательности (по умолчанию):**

```
Attempt 1: 5s    (min_backoff)
Attempt 2: 10s   (5s × 2.0)
Attempt 3: 20s   (10s × 2.0)
Attempt 4: 40s   (20s × 2.0)
Attempt 5: 80s   (40s × 2.0)
Attempt 6: 160s  (80s × 2.0)
Attempt 7: 5m    (capped at max_backoff)
...
```

_Фактическое время может отличаться на ±25% из-за jitter для распределения нагрузки._

---

### 🏥 E2E Проверка трафика (End-to-End Health Check)

Выполняет реальный HTTP запрос через VPN туннель для обнаружения тихих обрывов соединения (ошибки TLS EOF).

```bash
--e2e-check-url <url>  # HTTP URL для проверки прохождения трафика
```

**Как это работает:**

1. Открывает SOCKS5 соединение
2. Отправляет SOCKS5 CONNECT к целевому хосту (через туннель)
3. Выполняет HTTP GET запрос (через туннель)
4. Проверяет получение корректного HTTP ответа
5. При 3 ошибках подряд → автоматический failover на следующий сервер

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --e2e-check-url "http://ipinfo.io/ip"
```

**Используйте HTTP URL** (не HTTPS), чтобы избежать накладных расходов на TLS:

- `http://ipinfo.io/ip`
- `http://connectivitycheck.gstatic.com/generate_204`
- `http://httpbin.org/get`

**По умолчанию:** пустая строка (проверка только локального SOCKS, обратная совместимость)

---

### ⏱️ Таймауты

#### Таймаут проверки сервера

```bash
--timeout <duration>  # Таймаут для каждой проверки сервера
```

**Формат:** `5s`, `10s`, `30s` и т.д.

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --timeout 10s
```

**По умолчанию:** `5s`

---

### 🌐 Сетевые настройки

#### IPv6 поддержка

```bash
--ipv6  # Включить поддержку IPv6 (dual-stack)
```

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --ipv6
```

**По умолчанию:** `false`

**Что включается:**

- Настройка IPv6 адреса на TUN интерфейсе: `fd00:dead:beef::1/64`
- Маршрутизация IPv6 трафика через VPN
- Поддержка IPv6 DNS серверов

#### DNS защита от утечек

```bash
--dns-protection  # Включить защиту от утечек DNS
```

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --dns-protection
```

**По умолчанию:** `false`

**Что включается:**

- Маршрутизация DNS трафика через TUN интерфейс
- Добавление маршрутов к публичным DNS серверам (Google, Cloudflare, Quad9)
- Поддержка как IPv4, так и IPv6 DNS серверов

---

### 📊 Prometheus метрики

#### Порт метрик

```bash
--metrics-port <port>  # Включить endpoint Prometheus метрик
```

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --metrics-port 9090
```

**По умолчанию:** `0` (отключено)

**Доступные метрики:**

- `vpn_connections_total` - Всего подключений
- `vpn_disconnections_total` - Всего отключений
- `vpn_connection_duration_seconds` - Длительность текущего подключения
- `vpn_bytes_read_total` - Всего байт прочитано
- `vpn_bytes_written_total` - Всего байт записано
- `vpn_connected` - Статус подключения (1=подключен, 0=отключен)
- `vpn_tun_ipv4` - IPv4 адрес TUN интерфейса
- `vpn_tun_ipv6` - IPv6 адрес TUN интерфейса
- `vpn_server_ip` - IP адрес VPN сервера

**Endpoint:** `http://0.0.0.0:9090/metrics`

---

### 📝 Настройки логирования

#### Формат логов

```bash
--log-format <format>  # Формат вывода логов
```

**Варианты:** `text`, `json`

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --log-format json
```

**По умолчанию:** `text`

#### Уровень логирования

```bash
--log-level <level>  # Уровень детализации логов
```

**Варианты:** `debug`, `info`, `warn`, `error`

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --log-level debug
```

**По умолчанию:** `info`

#### Файл логов

```bash
--log-file <path>  # Путь к файлу для записи логов
```

**Пример:**

```bash
sudo goxray --from-raw https://example.com/links.txt --log-file /var/log/goxray/goxray.log
```

**По умолчанию:** (только stdout)

#### Размер файла логов

```bash
--log-max-size <MB>  # Максимальный размер файла логов перед ротацией
```

**Пример:**

```bash
sudo goxray --log-file /var/log/goxray/goxray.log --log-max-size 200
```

**По умолчанию:** `100` MB

#### Количество резервных файлов

```bash
--log-max-backups <count>  # Максимальное количество резервных файлов логов
```

**Пример:**

```bash
sudo goxray --log-file /var/log/goxray/goxray.log --log-max-backups 5
```

**По умолчанию:** `3`

#### Возраст резервных файлов

```bash
--log-max-age <days>  # Максимальный возраст резервных файлов в днях
```

**Пример:**

```bash
sudo goxray --log-file /var/log/goxray/goxray.log --log-max-age 30
```

**По умолчанию:** `28` дней

---

## Конфигурационный файл (YAML)

Все параметры могут быть заданы в YAML файле:

```yaml
# connection - настройки подключения
connection:
  # Прямая ссылка на сервер (использовать И это ИЛИ from_raw/from_raw_urls)
  link: "vless://uuid@server.example.com:443?type=tcp&security=reality&..."

  # Один URL для списка серверов (устарело, используйте from_raw_urls)
  # from_raw: "https://example.com/links.txt"

  # Несколько URL с поддержкой fallback
  from_raw_urls:
    - "https://primary.example.com/links.txt"
    - "https://backup1.example.com/links.txt"
    - "https://backup2.example.com/links.txt"

  # Включить IPv6 поддержку
  enable_ipv6: false

  # Включить защиту от утечек DNS
  enable_dns_protection: false

  # Разрешить самоподписанные сертификаты
  tls_allow_insecure: false

  # Порт для Prometheus метрик (0 = отключено)
  metrics_port: 9090

# server_selection - настройки выбора сервера
server_selection:
  # Интервал обновления списка серверов (например, "5m", "10m", "1h")
  refresh_interval: "10m"

  # Максимальное количество серверов для проверки
  max_servers: 10

  # Таймаут проверки каждого сервера
  timeout: "5s"

# reconnection - настройки переподключения (persistence & auto-reconnect)
reconnection:
  # Максимальное количество попыток (0 = безлимитно)
  max_retries: 0

  # Начальная задержка перед переподключением
  min_backoff: "5s"

  # Максимальная задержка перед переподключением
  max_backoff: "5m"

  # Множитель экспоненциальной задержки
  backoff_factor: 2.0

# logging - настройки логирования
logging:
  # Формат логов: "text" или "json"
  format: "text"

  # Уровень логирования: "debug", "info", "warn", "error"
  level: "info"

  # Путь к файлу логов (опционально)
  # file: "/var/log/goxray/goxray.log"

  # Максимальный размер файла в MB
  max_size: 100

  # Максимальное количество резервных файлов
  max_backups: 3

  # Максимальный возраст резервных файлов в днях
  max_age: 28

# health_monitoring - настройки мониторинга здоровья
health_monitoring:
  # Интервал проверки здоровья
  check_interval: "10s"

  # Таймаут проверки
  timeout: "5s"

  # Максимальное количество попыток перед переключением
  max_retries: 3
```

### Приоритет параметров

1. **CLI аргументы** (наивысший приоритет)
2. **Конфигурационный файл**
3. **Переменные окружения**
4. **Значения по умолчанию** (наинизший приоритет)

---

## Примеры использования

### Пример 1: Простое подключение

```bash
sudo goxray "vless://uuid@server:443?type=tcp&security=reality&..."
```

### Пример 2: Из списка серверов с автоматическим выбором

```bash
sudo goxray --from-raw https://example.com/links.txt
```

### Пример 2a: С авто-переподключением

```bash
sudo goxray \
  --from-raw https://example.com/links.txt \
  --max-retries 10 \
  --min-backoff 5s \
  --max-backoff 10m \
  --backoff-factor 2.0
```

**Что происходит:**

1. Загружается список серверов
2. При неудаче подключения ко всем серверам → ждет 5с
3. Перезагружает список серверов
4. Пробует снова
5. При неудаче → ждет 10с → 20с → 40с ... (до 10м)
6. После 10 попыток → выход с ошибкой
7. Ctrl+C в любой момент → graceful shutdown

### Пример 3: Полная конфигурация с логированием

```bash
sudo goxray \
  --from-raw https://example.com/links.txt \
  --refresh-interval 10m \
  --max-servers 20 \
  --timeout 10s \
  --ipv6 \
  --dns-protection \
  --metrics-port 9090 \
  --log-format json \
  --log-level info \
  --log-file /var/log/goxray/goxray.log \
  --log-max-size 200 \
  --log-max-backups 5 \
  --log-max-age 30
```

### Пример 4: Использование конфигурационного файла

```bash
# Создать конфигурационный файл
cat > /etc/goxray/config.yaml << EOF
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
  timeout: "10s"

reconnection:
  max_retries: 0
  min_backoff: "5s"
  max_backoff: "5m"
  backoff_factor: 2.0

logging:
  format: "json"
  level: "info"
  file: "/var/log/goxray/goxray.log"
  max_size: 200
  max_backups: 5
  max_age: 30

health_monitoring:
  check_interval: "10s"
  timeout: "5s"
  max_retries: 3
EOF

# Запустить с конфигурацией
sudo goxray --config /etc/goxray/config.yaml
```

### Пример 5: Только JSON логирование

```bash
sudo goxray \
  --from-raw https://example.com/links.txt \
  --log-format json \
  --log-file /var/log/goxray/goxray.json
```

---

## Переменные окружения

### GOXRAY_CONFIG_URL

```bash
export GOXRAY_CONFIG_URL="vless://uuid@server:443?..."
sudo goxray  # Использует ссылку из переменной окружения
```

---

## Важные замечания

### Требования к запуску

1. **sudo** - требуется для работы с сетевыми интерфейсами
2. **Linux capabilities** (опционально):
   ```bash
   sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
   ```

### Поддерживаемые ОС

- **Linux** (протестировано on Ubuntu 24.10, Debian 13)
- **macOS** (протестировано on Sequoia 15.1.1)

### Ограничения

- Максимум 10 одновременных проверок серверов (настраивается через `--max-servers`)
- Таймаут проверки по умолчанию 5 секунд
- Максимальный размер файла логов 100MB (настраивается)

---

## Диагностика и troubleshooting

### Проверка доступных флагов

```bash
sudo goxray --help
```

### Включение debug логирования

```bash
sudo goxray --from-raw https://example.com/links.txt --log-level debug
```

### Просмотр Prometheus метрик

```bash
curl http://localhost:9090/metrics
```

### Проверка статуса подключения

```bash
# В логах будет выводиться каждые 30 секунд:
# VPN Connection Status: connected, tun_interface=tun0, xray_server=3.112.126.206
```

---

## Дополнительные ресурсы

- [README.md](README.md) - Общая информация и примеры
- [HEALTH_MONITORING.md](HEALTH_MONITORING.md) - Детали системы мониторинга здоровья
- [PERIODIC_REFRESH.md](PERIODIC_REFRESH.md) - Настройки периодического обновления
- [DEPLOYMENT_DEBIAN13.md](DEPLOYMENT_DEBIAN13.md) - Инструкция по развертыванию
