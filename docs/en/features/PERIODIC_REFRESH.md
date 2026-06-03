# 🔄 Периодическое обновление списка серверов

## Overview

Добавлена возможность **периодического обновления списка серверов** from raw URL with настраиваемым интервалом. Это позволяет автоматически обнаруживать новые серверы and поддерживать актуальный список доступных вариантов.

---

## 🎯 Issue and Solution

### Issue
Ранее список серверов загружался один раз when запуске. Если появлялись новые серверы or менялась доступность существующих, требовался ручной перезапуск программы.

### Solution
Добавлен ключ `--refresh-interval` который:
- ✅ **Периодически обновляет** список серверов from raw URL
- ✅ **Обнаруживает новые серверы** автоматически
- ✅ **Логирует изменения** in списке доступных серверов
- ✅ **Working in фоне** without прерывания VPN подключения
- ✅ **Гибко настраивается** через командную строку

---

## 🚀 Usage

### Базовый пример

```bash
# Обновление каждые 5 минут
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 5m

# Обновление каждые 10 минут
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 10m

# Обновление каждый час
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 1h
```

### Все доступные опции

```bash
sudo goxray --from-raw <URL> [опции]

Опции:
  --refresh-interval <duration> - Интервал обновления списка (by умолчанию: 0 = выключено)
                                  Формат: 5m, 10m, 30m, 1h, and т.д.
  
  --max-servers <n>             - Максимальное количество проверяемых серверов (by умолчанию: 10)
  
  --timeout <duration>          - Таймаут проверки одного сервера (by умолчанию: 5s)
                                  Формат: 3s, 5s, 10s, and т.д.
```

---

## 📋 Examples использования

### Example 1: Стандартная конфигурация

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 10m \
  --max-servers 15 \
  --timeout 5s
```

**Что происходит:**
1. Загружается список серверов from URL
2. Проверяются первые 15 серверов with таймаутом 5s on каждый
3. Connection to лучшему серверу
4. Каждые 10 минут список обновляется
5. Health monitoring проверяет сервер каждые 10s

### Example 2: Агрессивное обновление (for нестабильных сетей)

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 2m \
  --max-servers 20 \
  --timeout 3s
```

**Benefits:**
- Быстрое обнаружение новых серверов (каждые 2 минуты)
- Больше вариантов for выбора (20 серверов)
- Меньший таймаут for ускорения проверок

### Example 3: Экономия трафика

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 1h \
  --max-servers 5 \
  --timeout 10s
```

**Benefits:**
- Редкое обновление экономит трафик (раз in час)
- Меньше проверок = меньше нагрузки
- Больший таймаут for надежности

---

## 🔄 How it works периодическое обновление

### Процесс работы

```
Запуск программы
    ↓
Первичная загрузка списка серверов
    ↓
Verification доступности and сортировка by latency
    ↓
Connection to лучшему серверу
    ↓
Запуск health monitoring (каждые 10s)
    ↓
┌─────────────────────────────────────┐
│ Каждые N минут (refresh-interval): │
│   ├─ Загрузка нового списка        │
│   ├─ Verification доступности          │
│   ├─ Сравнение with текущим списком   │
│   ├─ Логирование изменений         │
│   └─ Продолжение работы            │
└─────────────────────────────────────┘
    ↓
При обнаружении проблем со здоровьем:
    ├─ Автоматический failover
    ├─ Выбор следующего лучшего сервера
    └─ Usage актуального списка
```

### Example логов

```
INFO Fetching server list from raw URL url=https://example.com/links.txt refresh_interval=10m0s
INFO Checking servers total=15 max_concurrent=10
INFO Found available servers total=8 sorted_by=latency
INFO Server selection results:
=== VPN Server Selection Report ===
Total servers scanned: 15
Available servers: 8

1. server1.com:443 - Latency: 45ms - ★ RECOMMENDED
2. server2.com:443 - Latency: 78ms - ✓ Available
...

INFO Attempting VPN connection with fallback support servers_count=8
INFO Successfully connected to VPN server host=server1.com port=443 latency=45ms
INFO Starting health checks host=server1.com port=443 interval=10s timeout=5s max_retries=3
INFO Periodic server list refresh enabled interval=10m0s

# ... через 10 минут ...

INFO Refreshing server list from raw URL url=https://example.com/links.txt
INFO Checking servers total=16 max_concurrent=10
INFO Found available servers total=9 sorted_by=latency
INFO Server list refreshed successfully total_servers=9 new_servers_available=9
INFO Updated server list:
=== VPN Server Selection Report ===
Total servers scanned: 16
Available servers: 9

1. server1.com:443 - Latency: 45ms - ★ RECOMMENDED
2. new-server.com:443 - Latency: 65ms - ✓ Available  ← НОВЫЙ СЕРВЕР!
3. server2.com:443 - Latency: 78ms - ✓ Available
...
```

---

## 💡 Сценарии использования

### Scenario 1: Стабильная сеть (редкое обновление)

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 2h \
  --max-servers 10
```

**When to use:**
- Серверы редко добавляются/удаляются
- Important минимизировать трафик
- Достаточно базового мониторинга

### Scenario 2: Динамическая среда (частое обновление)

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 5m \
  --max-servers 20 \
  --timeout 3s
```

**When to use:**
- Серверы часто меняются
- Нужна максимальная доступность
- Traffic не критичен

### Scenario 3: Критичный сервис (очень частое обновление)

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 1m \
  --max-servers 30 \
  --timeout 2s
```

**When to use:**
- Бизнес-критичное подключение
- Серверы могут падать часто
- Требуется постоянная доступность

---

## 🔧 Технические детали

### Взаимодействие with Health Monitoring

Периодическое обновление **работает вместе** with health monitoring:

| Компонент | Частота | Назначение |
|-----------|---------|------------|
| **Health Check** | Каждые 10s | Verification текущего сервера |
| **Failover** | При необходимости | Переключение when проблемах |
| **List Refresh** | Каждые N минут | Обновление списка серверов |

**Example взаимодействия:**
```
10:00:00 - Health check ✓ (server1 OK)
10:00:10 - Health check ✓ (server1 OK)
10:00:20 - Health check ✓ (server1 OK)
10:05:00 - List refresh → обнаружен new-server.com (latency: 50ms)
10:05:10 - Health check ✓ (server1 OK, latency: 80ms)
10:05:20 - Health check ✓ (server1 OK)
10:05:30 - Auto-failover triggered (server1 стал медленным)
           → переключение on new-server.com (50ms)
```

### Расход ресурсов

| Параметр | Значение |
|----------|----------|
| **CPU when refresh** | ~1-2% (on время проверки) |
| **RAM** | ~100 KB (хранение списка) |
| **Network** | ~1 HTTP запрос + TCP проверки |
| **Время refresh** | ~5-30s (зависит от кол-ва серверов) |

---

## ⚙️ Recommendations by настройке

### Оптимальные настройки for разных случаев

#### Для дома (экономия ресурсов)
```bash
--refresh-interval 1h \
--max-servers 5 \
--timeout 10s
```

#### Для офиса (баланс)
```bash
--refresh-interval 10m \
--max-servers 15 \
--timeout 5s
```

#### Для продакшена (максимальная надежность)
```bash
--refresh-interval 5m \
--max-servers 30 \
--timeout 3s
```

---

## 📊 Статистика and мониторинг

### Что логируется when обновлении

```
INFO Refreshing server list from raw URL url=...
INFO Checking servers total=15 max_concurrent=10
INFO Found available servers total=8 sorted_by=latency
INFO Server list refreshed successfully total_servers=8 new_servers_available=8
INFO Updated server list:
=== VPN Server Selection Report ===
...
```

### Ключевые метрики

- **total_servers** - всего серверов in списке
- **available_servers** - доступных серверов сейчас
- **new_servers_available** - сколько серверов доступно (всегда равно available)
- **latency rankings** - рейтинг серверов by скорости

---

## 🧪 Testing

### Быстрое тестирование

```bash
# Обновление каждую минуту for быстрого наблюдения
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 1m

# Наблюдать логи обновления каждую минуту
sudo journalctl -u goxray -f | grep "refresh\|Refreshing"
```

### Verification работы

```bash
# 1. Start with коротким интервалом
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 2m

# 2. Подождать 2 минуты
# 3. Увидеть in логах "Refreshing server list..."
# 4. Check что список обновился
```

---

## 🎉 Итоги

✅ **Автоматическое обновление** - список серверов всегда актуален  
✅ **Гибкая настройка** - интервал от минут до часов  
✅ **Фоновая работа** - не мешает VPN подключению  
✅ **Обнаружение новинок** - автоматическое нахождение новых серверов  
✅ **Integration with health monitoring** - комплексная система надежности  
✅ **Минимальный overhead** - обновление only by таймеру  

**Теперь ваш VPN client всегда in курсе лучших серверов!** 🚀
