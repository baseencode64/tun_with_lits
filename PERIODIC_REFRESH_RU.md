# 🔄 Периодическое обновление списка серверов - Краткая справка

## Быстрый старт

### Базовое использование

```bash
# Обновление каждые 5 минут
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 5m

# Обновление каждый час
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 1h
```

### Все опции командной строки

```bash
sudo goxray --from-raw <URL> \
  --refresh-interval <duration> \
  --max-servers <n> \
  --timeout <duration>
```

| Параметр | По умолчанию | Описание |
|----------|--------------|----------|
| `--refresh-interval` | 0 (выключено) | Интервал обновления списка (5m, 10m, 1h) |
| `--max-servers` | 10 | Макс. количество проверяемых серверов |
| `--timeout` | 5s | Таймаут проверки одного сервера |

---

## 💡 Примеры конфигурации

### Экономия трафика
```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 2h \
  --max-servers 5
```

### Стандартный режим
```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 10m \
  --max-servers 15 \
  --timeout 5s
```

### Максимальная надежность
```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 2m \
  --max-servers 30 \
  --timeout 3s
```

---

## 🔍 Как это работает

1. **При запуске**: Загружается список серверов из raw URL
2. **Подключение**: Выбирается лучший сервер по latency
3. **Health monitoring**: Каждые 10s проверяется доступность
4. **Периодический refresh**: Каждые N минут обновляется список
5. **Failover**: При проблемах автоматически переключается

---

## 📊 Что видно в логах

```
INFO Fetching server list from raw URL url=... refresh_interval=10m0s
INFO Periodic server list refresh enabled interval=10m0s

# ... через 10 минут ...

INFO Refreshing server list from raw URL url=...
INFO Server list refreshed successfully total_servers=9
INFO Updated server list:
=== VPN Server Selection Report ===
1. server1.com:443 - Latency: 45ms - ★ RECOMMENDED
2. new-server.com:443 - Latency: 65ms - ✓ Available
...
```

---

## ⚡ Взаимодействие с Health Monitoring

| Компонент | Частота | Назначение |
|-----------|---------|------------|
| Health Check | Каждые 10s | Проверка текущего сервера |
| List Refresh | Каждые N мин | Обновление списка серверов |
| Auto Failover | При необходимости | Переключение на другой сервер |

**Преимущество**: Список всегда актуален + автоматическое восстановление при сбоях

---

## 🎯 Когда использовать

✅ **Часто меняющиеся серверы** - включайте refresh каждые 5-10 минут  
✅ **Новые серверы появляются** - refresh обнаружит их автоматически  
✅ **Критичная доступность** - используйте вместе с health monitoring  
✅ **Экономия времени** - не нужно перезапускать вручную  

❌ **Стабильные серверы** - refresh можно выключить (`--refresh-interval 0`)  
❌ **Ограниченный трафик** - увеличьте интервал до 1-2 часов  

---

## 🚀 Полная команда для продакшена

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 5m \
  --max-servers 20 \
  --timeout 3s
```

**Что получаете:**
- ✅ Актуальный список серверов (обновление каждые 5 минут)
- ✅ Проверка до 20 серверов для лучшего выбора
- ✅ Быстрый таймаут (3s) для ускорения загрузки
- ✅ Health monitoring каждые 10 секунд
- ✅ Автоматический failover при проблемах (~30s)

---

## 📚 Полная документация

Подробная документация доступна в файле **[PERIODIC_REFRESH.md](file://d:\gotun_with_raw\PERIODIC_REFRESH.md)**
