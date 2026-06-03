# 🏥 Health Monitoring & Automatic Failover

## Overview

Реализована система **непрерывного мониторинга здоровья** VPN подключения with **автоматическим переключением** on следующий сервер when обнаружении проблем.

---

## 🎯 Issue and Solution

### Issue

Ранее клиент проверял доступность сервера only **один раз** when подключении. Если после подключения сервер становился недоступным (трафик перестал проходить), пользователь оставался without VPN до момента ручного вмешательства.

### Solution

Добавлена система **Health Check**, которая:

- ✅ **Непрерывно мониторит** подключение to VPN серверу
- ✅ **Автоматически переключает** on следующий лучший сервер when проблемах
- ✅ **Периодически проверяет** доступность через TCP connection check
- ✅ **Считывает consecutive failures** перед триггером failover
- ✅ **Логирует статус** здоровья каждые 30 секунд

---

## 📦 Новые компоненты

### 1. `pkg/client/health_checker.go`

Класс **HealthChecker** - отвечает за мониторинг здоровья сервера:

```go
type HealthChecker struct {
    logger        *slog.Logger
    checkInterval time.Duration  // Интервал проверок (by умолчанию 10s)
    timeout       time.Duration  // Таймаут каждой проверки (by умолчанию 5s)
    maxRetries    int            // Макс. попыток перед failover (by умолчанию 3)

    mu          sync.RWMutex
    isHealthy   bool
    consecutiveFailures int
    stopChan    chan struct{}
}
```

**Ключевые методы:**

- **Start(ctx, host, port, onUnhealthy)** - запускает цикл проверок
- **Stop()** - останавливает проверки
- **IsHealthy()** - возвращает текущий статус
- **GetStatus()** - детальная информация о здоровье

### 2. Обновленный `vpn_connector.go`

Integration HealthChecker with автоматическим failover:

```go
type VPNConnector struct {
    client        *Client
    selector      *ServerSelector
    logger        *slog.Logger
    healthChecker *HealthChecker  // NEW
    ctx           context.Context
    cancelFunc    context.CancelFunc

    currentServerIndex int
    servers            []*ServerInfo
}
```

**Новые методы:**

- **startHealthMonitoring(server)** - начинает мониторинг текущего сервера
- **performFailover()** - автоматически переключает on следующий сервер
- **GetHealthStatus()** - возвращает полный статус здоровья
- **Stop()** - корректная остановка всех процессов

---

## 🔄 Как это работает

### Процесс Health Checking

```
Connection to серверу
         ↓
Запуск Health Checker
         ↓
Каждые 10 секунд:
    ├─ TCP подключение to серверу
    ├─ Verification ответа
    ├─ Успех → сброс счетчика ошибок
    └─ Error → увеличение consecutive_failures
         ↓
Если consecutive_failures >= 3:
    ├─ Пометить сервер как unhealthy
    ├─ Start performFailover()
    ├─ Отключиться от текущего сервера
    ├─ Подключиться to следующему by списку
    └─ Restart Health Checker
```

### Example лого

```
INFO Starting health checks host=server1.com port=443 interval=10s timeout=5s max_retries=3
INFO VPN connected successfully
INFO VPN Health Status status={"connected":true,"current_server_idx":0,...}
WARN Health check failed attempt=1 max_retries=3 error="dial failed: timeout"
WARN Health check failed attempt=2 max_retries=3 error="dial failed: timeout"
WARN Health check failed attempt=3 max_retries=3 error="dial failed: timeout"
ERROR Server unhealthy - exceeded max retries failures=3
INFO Triggering failover to next server
INFO Failing over to next server from_index=0 to_index=1 next_host=server2.com
INFO Connecting to next server host=server2.com port=443
INFO Successfully failed over to next server host=server2.com port=443 index=1
INFO Starting health checks host=server2.com port=443 interval=10s timeout=5s max_retries=3
```

---

## ⚙️ Configuration

### Настройки Health Checker

```go
// Создание with настройками by умолчанию
healthChecker := NewHealthChecker(
    logger,
    10*time.Second,  // интервал проверок
    5*time.Second,   // таймаут каждой проверки
    3,               // макс. попыток перед failover
)
```

### Параметры

| Параметр        | По умолчанию | Description                           |
| --------------- | ------------ | ---------------------------------- |
| `checkInterval` | 10s          | Как часто проверять сервер         |
| `timeout`       | 5s           | Максимальное время ожидания ответа |
| `maxRetries`    | 3            | Сколько ошибок перед failover      |

**Время до failover**: `checkInterval × maxRetries = 10s × 3 = 30s`

---

## 🚀 Usage

### Автоматический режим (with raw списком)

```bash
# Health monitoring включен автоматически
sudo goxray --from-raw https://example.com/links.txt
```

### Как библиотека

```go
package main

import (
    "context"
    "log/slog"
    "time"
    "github.com/goxray/tun/pkg/client"
)

func main() {
    logger := slog.New(slog.NewTextHandler(nil, nil))
    vpn, _ := client.NewClientWithOpts(client.Config{Logger: logger})

    // Создание селектора
    selector := client.NewServerSelector(loggerAdapter, 5*time.Second, 10)
    links, _ := selector.FetchRawLinks("https://example.com/links.txt")
    servers, _ := selector.SelectAllByLatency(links)

    // Создание коннектора with health monitoring
    connector := client.NewVPNConnector(vpn, selector, logger)
    defer connector.Stop()

    // Connection with автоматическим health check
    connector.ConnectWithFallback(servers)

    // Health checker запущен автоматически!
    // Каждые 10 секунд проверяет сервер
    // При 3 неудачных попытках - автоматический failover

    // Monitoring статуса
    for {
        status := connector.GetHealthStatus()
        log.Printf("Health: %+v", status)
        time.Sleep(30 * time.Second)
    }
}
```

---

## 📊 Статус здоровья

### GetHealthStatus() возвращает

```json
{
  "connected": true,
  "current_server_idx": 0,
  "total_servers": 5,
  "current_server": {
    "Link": "vless://...",
    "Host": "server1.com",
    "Port": "443",
    "Latency": 50000000
  },
  "health": {
    "is_healthy": true,
    "consecutive_failures": 0,
    "last_check": "2026-04-29T02:00:00Z",
    "check_interval": 10000000000,
    "max_retries": 3
  }
}
```

---

## 🔍 Сценарии работы

### Scenario 1: Server стал недоступен

```
1. Connection to server1 (50ms) ✓
2. Health check #1 (10s): ✓ Healthy
3. Health check #2 (20s): ✓ Healthy
4. Server1 падает ✗
5. Health check #3 (30s): ✗ Failed (attempt 1/3)
6. Health check #4 (40s): ✗ Failed (attempt 2/3)
7. Health check #5 (50s): ✗ Failed (attempt 3/3)
8. TRIGGER FAILOVER → automatic switch to server2 (100ms)
9. Connection to server2 ✓
10. Health check продолжается for server2
```

### Scenario 2: Временные проблемы with сетью

```
1. Connection to server1 ✓
2. Health check #1: ✓ Healthy
3. Health check #2: ✗ Failed (временная ошибка)
4. Health check #3: ✓ Healthy (восстановилось)
5. consecutive_failures сброшен in 0
6. FAILOVER НЕ происходит (было < 3 ошибок подряд)
```

### Scenario 3: Все серверы недоступны

```
1. Попытка server1: ✗ Failed
2. Попытка server2: ✗ Failed
3. Попытка server3: ✗ Failed
4. ... all варианты исчерпаны
5. ERROR: "Failed to connect to all servers"
6. Программа завершается with ошибкой
```

---

## 🧪 Testing

### Запуск тестов

```bash
# Тесты Health Checker
go test ./pkg/client/... -v -run TestHealthChecker

# Тесты VPN Connector with health monitoring
go test ./pkg/client/... -v -run TestVPNConnector

# Все тесты
go test ./pkg/client/... -v
```

### Ручное тестирование

```bash
# 1. Start with реальным списком
sudo goxray --from-raw https://example.com/links.txt

# 2. Наблюдать health статус in логах
# Каждые 30 секунд выводится статус

# 3. Для проверки failover:
# - Заблокировать текущий сервер in firewall
# - Или отключить сеть временно
# - Подождать ~30 секунд
# - Увидеть автоматическое переключение
```

---

## 💡 Recommendations by использованию

### 1. Setup интервалов

Для разных сценариев:

**Быстрое обнаружение** (критичные сервисы):

```go
NewHealthChecker(logger, 5*time.Second, 3*time.Second, 2)
// Failover через: 5s × 2 = 10s
```

**Стандартный режим**:

```go
NewHealthChecker(logger, 10*time.Second, 5*time.Second, 3)
// Failover через: 10s × 3 = 30s
```

### E2E Verification трафика (End-to-End)

По умолчанию Health Checker проверяет only доступность локального SOCKS прокси. Чтобы обнаруживать ситуации, когда SOCKS прокси работает, но VPN туннель разорван (например, TLS EOF ошибки or тихие обрывы сервером), включите сквозную (E2E) проверку:

**Через CLI:**

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --e2e-check-url "http://ipinfo.io/ip"
```

**Через конфигурационный файл:**

```yaml
connection:
  e2e_check_url: "http://ipinfo.io/ip"
```

How it works E2E проверка:

1. Открывает SOCKS5 соединение with `127.0.0.1:{socks_port}`
2. Отправляет SOCKS5 CONNECT запрос to целевому хосту (через туннель)
3. Выполняет HTTP GET запрос через установленный туннель
4. Проверяет корректность HTTP ответа (например, `HTTP/1.1 200 OK`)
5. При любой ошибке (включая EOF) → считает сервер нездоровым → после `max_retries` вызывает failover

> **Important:** Используйте HTTP URL (не HTTPS) for E2E проверок, чтобы избежать лишней вычислительной нагрузки от установки TLS соединения when каждой проверке. Цель — проверить маршрутизацию данных через туннель. Рекомендуемые URL: `http://ipinfo.io/ip`, `http://connectivitycheck.gstatic.com/generate_204`.

**Экономия трафика** (медленные сети):

```go
NewHealthChecker(logger, 30*time.Second, 10*time.Second, 5)
// Failover через: 30s × 5 = 150s (2.5 мин)
```

### 2. Monitoring логов

Настройте сбор логов for анализа failover:

```bash
# systemd journal
sudo journalctl -u goxray -f | grep -i "health\|failover"

# Или лог файл
sudo goxray --from-raw https://example.com/links.txt 2>&1 | tee /var/log/goxray.log
```

### 3. Graceful shutdown

Всегда вызывайте `Stop()` when завершении:

```go
connector := client.NewVPNConnector(vpn, selector, logger)
defer connector.Stop() // Important for остановки health checker!
```

---

## 🔐 Security

Health checker использует **TCP подключение** without отправки данных:

- ✅ Не передает sensitive information
- ✅ Не выполняет handshake with VPN
- ✅ Только проверяет доступность порта
- ✅ Минимальный overhead (~1 пакет каждые 10s)

---

## 📈 Performance

### Resource Usage

| Метрика        | Значение                    |
| -------------- | --------------------------- |
| **CPU**        | < 0.1% (проверка раз in 10s) |
| **RAM**        | ~50 KB per HealthChecker    |
| **Network**    | ~1 TCP packet / 10s         |
| **Goroutines** | +1 per connector            |

### Overhead

- **Initial connection**: +0ms (health check starts after)
- **Per check**: ~5-100ms (depends on latency)
- **Failover time**: ~2-5s (disconnect + reconnect)

---

## 🎉 Итоги

✅ **Непрерывный мониторинг** - проверка каждые 10 секунд  
✅ **Автоматический failover** - переключение without участия пользователя  
✅ **Настраиваемость** - гибкая конфигурация интервалов  
✅ **Надежность** - защита от временных сбоев (consecutive failures)  
✅ **Наблюдаемость** - подробные логи and статус  
✅ **Graceful degradation** - корректная обработка всех ошибок

**Ваш VPN теперь сам восстанавливается when проблемах!** 🚀
