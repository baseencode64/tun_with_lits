# ✅ FIX: Metrics Accumulation & IP Update on Reconnection

## 🔧 Проблемы (Исправлено)

### ❌ Проблема 1: vpn_bytes_read и vpn_bytes_written сбрасываются при reconnect
**Было**: При переподключении значения bytes обнулялись, не учитывался весь трафик
**Решение**: Добавлены глобальные накопительные счетчики
**Результат**: ✅ Метрики теперь показывают СУММУ всего трафика с момента запуска клиента

### ❌ Проблема 2: vpn_server_ip и xray_server не обновляются после reconnect  
**Было**: IP адреса сервера оставались старыми
**Решение**: Добавлено явное обновление IP в Connect() + логирование
**Результат**: ✅ Метрики обновляются с каждым новым подключением

---

## 💾 Структурные Изменения (pkg/client/client.go)

### 1. Добавлены новые поля в структуру Client
```go
type Client struct {
    // ... existing fields ...
    
    // Traffic counters (atomic)
    bytesRead    int64
    bytesWritten int64
    
    // 🆕 Cumulative traffic counters (preserved across reconnections)
    cumulativeBytesRead    int64
    cumulativeBytesWritten int64
}
```

**Назначение**:
- `bytesRead/bytesWritten` - текущий tunnel, обнуляются при reconnect
- `cumulativeBytesRead/Written` - сумма всех bytes со всех tunnels, никогда не обнуляются

---

## 🔄 Логика Потока Данных

### При Connect() - Обновление IP метрик
```go
// Update VPN server external IP
if c.xSrvIP != nil {
    serverIP := c.xSrvIP.String()
    vpnServerIP.Reset()                           // Очистить старое значение
    vpnServerIP.WithLabelValues(serverIP).Set(1)  // Установить новое
    c.cfg.Logger.Info("VPN server IP metric updated", "ip", serverIP)
}

// Preserve cumulative traffic counters on reconnection
c.cfg.Logger.Debug("Metrics state on reconnection",
    "cumulative_bytes_read", atomic.LoadInt64(&c.cumulativeBytesRead),
    "current_bytes_read", c.BytesRead(),
    "cumulative_bytes_written", atomic.LoadInt64(&c.cumulativeBytesWritten),
    "current_bytes_written", c.BytesWritten())
```

✅ IP метрики обновляются с новым сервером

### При Disconnect() - Сохранение Bytes
```go
// Preserve cumulative bytes: add current bytes to cumulative counters
// This ensures metrics show total traffic across all reconnections
currentRead := int64(c.BytesRead())
currentWritten := int64(c.BytesWritten())
atomic.AddInt64(&c.cumulativeBytesRead, currentRead)
atomic.AddInt64(&c.cumulativeBytesWritten, currentWritten)
c.cfg.Logger.Info("Bytes preserved on disconnect",
    "current_read", currentRead, "current_written", currentWritten,
    "new_cumulative_read", atomic.LoadInt64(&c.cumulativeBytesRead),
    "new_cumulative_written", atomic.LoadInt64(&c.cumulativeBytesWritten))
```

✅ Перед тем как TUN закроется, bytes сохраняются в cumulative счетчик

### При обновлении метрик (в startMetricsUpdate()) - Сумма
```go
// Update traffic metrics from atomic counters
// Include both cumulative bytes and current connection bytes
totalRead := atomic.LoadInt64(&c.cumulativeBytesRead) + int64(c.BytesRead())
totalWritten := atomic.LoadInt64(&c.cumulativeBytesWritten) + int64(c.BytesWritten())
vpnBytesRead.Set(float64(totalRead))
vpnBytesWritten.Set(float64(totalWritten))
```

✅ Метрики показывают сумму: cumulative + current tunnel

---

## 📊 Пример Workflow с Reconnect

```
┌────────────────────────────────────────────────────────────┐
│ Lifecycle: Показывает как обновляются метрики             │
└────────────────────────────────────────────────────────────┘

Start App
│
├─ cumulativeBytesRead = 0
├─ cumulativeBytesWritten = 0
│
▼

Connect("vless://server-a.com:443")
│
├─ Create TUN tunnel (readerMetrics)
│  └─ bytesRead = 0, bytesWritten = 0
│
├─ Update IP metrics:
│  ├─ vpn_server_ip = 1.2.3.4
│  └─ vpn_tun_ipv4 = 192.18.0.1
│
└─ Connected! ✅

┌─ 5 seconds later ─┐
│ Client sends 1 GB
│ bytesRead = 1,000,000,000
│ bytesWritten = 500,000,000
└───────────────────┘

Health Check FAILS → Failover

Disconnect()
│
├─ Preserve bytes:
│  ├─ cumulativeBytesRead += 1,000,000,000  → 1,000,000,000
│  ├─ cumulativeBytesWritten += 500,000,000 → 500,000,000
│  └─ bytesRead/Written reset on new tunnel
│
├─ Reset IP metrics:
│  ├─ vpn_server_ip = (cleared)
│  └─ vpn_tun_ipv4 = (cleared)
│
└─ Disconnected ✅

┌──────────────── Reconnection ───────────────┐
│                                             │
Connect("vless://server-b.com:443")  ← NEW!
│
├─ Create NEW TUN tunnel (readerMetrics)
│  └─ bytesRead = 0 (fresh)
│  └─ bytesWritten = 0 (fresh)
│
├─ Update IP metrics:
│  ├─ vpn_server_ip = 5.6.7.8 ← НОВЫЙ IP! ✅
│  └─ vpn_tun_ipv4 = 192.18.0.1
│
└─ Connected to new server! ✅

┌─ Metrics Update (every 5 sec) ─┐
│ totalRead = 1GB + 0 = 1GB       │
│ totalWritten = 500MB + 0 = 500MB│
│ (accumulating across servers!)  │
└────────────────────────────────┘

┌─ 3 seconds later ─┐
│ Client sends 500MB
│ bytesRead = 500,000,000 (current tunnel)
│ cumulativeBytesRead = 1,000,000,000
└────────────────────┘

Metrics show:
├─ vpn_bytes_read_total = 1.5 GB ✅
├─ vpn_bytes_written_total = 750 MB ✅
├─ vpn_server_ip = 5.6.7.8 ✅
└─ vpn_connections_total{protocol="vless"} = 2 ✅
```

---

## 🧪 Тестирование

### Test Case 1: Проверить что bytes накапливаются
```bash
# Terminal 1: Start VPN with metrics
sudo go run . --from-raw "..." --metrics-port 9090

# Terminal 2: Monitor bytes
watch -n 1 'curl -s http://localhost:9090/metrics | grep vpn_bytes'

# Expected before reconnect (5 sec):
# vpn_bytes_read_total 100000000
# vpn_bytes_written_total 50000000

# After failover and reconnection, bytes should:
# - NOT reset to 0
# - Continue growing from where they left off
# vpn_bytes_read_total 100000001  (continuing from previous)
```

### Test Case 2: Проверить что IP обновляется
```bash
# Monitor server IP before failover
curl -s http://localhost:9090/metrics | grep 'vpn_server_ip{ip'
# Result: vpn_server_ip{ip_address="1.2.3.4"} 1

# Trigger failover (wait for health check fail or force disconnect)

# Monitor server IP after reconnection
curl -s http://localhost:9090/metrics | grep 'vpn_server_ip{ip'
# Result: vpn_server_ip{ip_address="5.6.7.8"} 1 ← CHANGED! ✅
```

### Test Case 3: Полный цикл
```bash
# Start monitoring
curl http://localhost:9090/metrics | grep -E 'vpn_(bytes|server_ip|connections_total)'

# Should show:
# - vpn_bytes_read_total: starts at 0, grows with traffic
# - vpn_bytes_written_total: starts at 0, grows with traffic
# - vpn_server_ip{ip_address="..."}: has current server IP
# - vpn_connections_total: increments with each reconnect

# After reconnect:
# - vpn_bytes_*: CONTINUE from where they left off (NOT RESET) ✅
# - vpn_server_ip: UPDATES to new server IP ✅
# - vpn_connections_total: INCREMENTS by 1 ✅
```

---

## 🔍 Изменённые Файлы

| Файл | Строки | Изменения |
|------|--------|-----------|
| `pkg/client/client.go` | +30 | Добавлены накопительные счетчики и логика обновления |
| **Всего** | **+30** | Минимальные, целевые изменения |

---

## ✅ Checklist

- [x] Добавлены поля `cumulativeBytesRead` и `cumulativeBytesWritten` в Client
- [x] Логика сохранения bytes при Disconnect()
- [x] Логика суммирования при обновлении метрик
- [x] IP метрики явно обновляются при Connect()
- [x] Логирование для отладки
- [x] Код компилируется без ошибок
- [x] Backward compatible (нет breaking changes)

---

## 🚀 Результат

| Метрика | Поведение |
|---------|-----------|
| `vpn_bytes_read_total` | ✅ Сумма всего трафика (accumulates) |
| `vpn_bytes_written_total` | ✅ Сумма всего трафика (accumulates) |
| `vpn_server_ip` | ✅ Обновляется при каждом новом подключении |
| `vpn_tun_ipv4` | ✅ Обновляется при каждом новом подключении |
| `vpn_connections_total` | ✅ Увеличивается при каждом reconnect |

**Теперь метрики полностью корректны!** 🎉
