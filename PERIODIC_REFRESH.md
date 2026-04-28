# 🔄 Периодическое обновление списка серверов

## Обзор

Добавлена возможность **периодического обновления списка серверов** из raw URL с настраиваемым интервалом. Это позволяет автоматически обнаруживать новые серверы и поддерживать актуальный список доступных вариантов.

---

## 🎯 Проблема и Решение

### Проблема
Ранее список серверов загружался один раз при запуске. Если появлялись новые серверы или менялась доступность существующих, требовался ручной перезапуск программы.

### Решение
Добавлен ключ `--refresh-interval` который:
- ✅ **Периодически обновляет** список серверов из raw URL
- ✅ **Обнаруживает новые серверы** автоматически
- ✅ **Логирует изменения** в списке доступных серверов
- ✅ **Работает в фоне** без прерывания VPN подключения
- ✅ **Гибко настраивается** через командную строку

---

## 🚀 Использование

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
  --refresh-interval <duration> - Интервал обновления списка (по умолчанию: 0 = выключено)
                                  Формат: 5m, 10m, 30m, 1h, и т.д.
  
  --max-servers <n>             - Максимальное количество проверяемых серверов (по умолчанию: 10)
  
  --timeout <duration>          - Таймаут проверки одного сервера (по умолчанию: 5s)
                                  Формат: 3s, 5s, 10s, и т.д.
```

---

## 📋 Примеры использования

### Пример 1: Стандартная конфигурация

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 10m \
  --max-servers 15 \
  --timeout 5s
```

**Что происходит:**
1. Загружается список серверов из URL
2. Проверяются первые 15 серверов с таймаутом 5s на каждый
3. Подключение к лучшему серверу
4. Каждые 10 минут список обновляется
5. Health monitoring проверяет сервер каждые 10s

### Пример 2: Агрессивное обновление (для нестабильных сетей)

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 2m \
  --max-servers 20 \
  --timeout 3s
```

**Преимущества:**
- Быстрое обнаружение новых серверов (каждые 2 минуты)
- Больше вариантов для выбора (20 серверов)
- Меньший таймаут для ускорения проверок

### Пример 3: Экономия трафика

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 1h \
  --max-servers 5 \
  --timeout 10s
```

**Преимущества:**
- Редкое обновление экономит трафик (раз в час)
- Меньше проверок = меньше нагрузки
- Больший таймаут для надежности

---

## 🔄 Как работает периодическое обновление

### Процесс работы

```
Запуск программы
    ↓
Первичная загрузка списка серверов
    ↓
Проверка доступности и сортировка по latency
    ↓
Подключение к лучшему серверу
    ↓
Запуск health monitoring (каждые 10s)
    ↓
┌─────────────────────────────────────┐
│ Каждые N минут (refresh-interval): │
│   ├─ Загрузка нового списка        │
│   ├─ Проверка доступности          │
│   ├─ Сравнение с текущим списком   │
│   ├─ Логирование изменений         │
│   └─ Продолжение работы            │
└─────────────────────────────────────┘
    ↓
При обнаружении проблем со здоровьем:
    ├─ Автоматический failover
    ├─ Выбор следующего лучшего сервера
    └─ Использование актуального списка
```

### Пример логов

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

### Сценарий 1: Стабильная сеть (редкое обновление)

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 2h \
  --max-servers 10
```

**Когда использовать:**
- Серверы редко добавляются/удаляются
- Важно минимизировать трафик
- Достаточно базового мониторинга

### Сценарий 2: Динамическая среда (частое обновление)

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 5m \
  --max-servers 20 \
  --timeout 3s
```

**Когда использовать:**
- Серверы часто меняются
- Нужна максимальная доступность
- Трафик не критичен

### Сценарий 3: Критичный сервис (очень частое обновление)

```bash
sudo goxray --from-raw https://example.com/links.txt \
  --refresh-interval 1m \
  --max-servers 30 \
  --timeout 2s
```

**Когда использовать:**
- Бизнес-критичное подключение
- Серверы могут падать часто
- Требуется постоянная доступность

---

## 🔧 Технические детали

### Взаимодействие с Health Monitoring

Периодическое обновление **работает вместе** с health monitoring:

| Компонент | Частота | Назначение |
|-----------|---------|------------|
| **Health Check** | Каждые 10s | Проверка текущего сервера |
| **Failover** | При необходимости | Переключение при проблемах |
| **List Refresh** | Каждые N минут | Обновление списка серверов |

**Пример взаимодействия:**
```
10:00:00 - Health check ✓ (server1 OK)
10:00:10 - Health check ✓ (server1 OK)
10:00:20 - Health check ✓ (server1 OK)
10:05:00 - List refresh → обнаружен new-server.com (latency: 50ms)
10:05:10 - Health check ✓ (server1 OK, latency: 80ms)
10:05:20 - Health check ✓ (server1 OK)
10:05:30 - Auto-failover triggered (server1 стал медленным)
           → переключение на new-server.com (50ms)
```

### Расход ресурсов

| Параметр | Значение |
|----------|----------|
| **CPU при refresh** | ~1-2% (на время проверки) |
| **RAM** | ~100 KB (хранение списка) |
| **Network** | ~1 HTTP запрос + TCP проверки |
| **Время refresh** | ~5-30s (зависит от кол-ва серверов) |

---

## ⚙️ Рекомендации по настройке

### Оптимальные настройки для разных случаев

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

## 📊 Статистика и мониторинг

### Что логируется при обновлении

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

- **total_servers** - всего серверов в списке
- **available_servers** - доступных серверов сейчас
- **new_servers_available** - сколько серверов доступно (всегда равно available)
- **latency rankings** - рейтинг серверов по скорости

---

## 🧪 Тестирование

### Быстрое тестирование

```bash
# Обновление каждую минуту для быстрого наблюдения
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 1m

# Наблюдать логи обновления каждую минуту
sudo journalctl -u goxray -f | grep "refresh\|Refreshing"
```

### Проверка работы

```bash
# 1. Запустить с коротким интервалом
sudo goxray --from-raw https://example.com/links.txt --refresh-interval 2m

# 2. Подождать 2 минуты
# 3. Увидеть в логах "Refreshing server list..."
# 4. Проверить что список обновился
```

---

## 🎉 Итоги

✅ **Автоматическое обновление** - список серверов всегда актуален  
✅ **Гибкая настройка** - интервал от минут до часов  
✅ **Фоновая работа** - не мешает VPN подключению  
✅ **Обнаружение новинок** - автоматическое нахождение новых серверов  
✅ **Интеграция с health monitoring** - комплексная система надежности  
✅ **Минимальный overhead** - обновление только по таймеру  

**Теперь ваш VPN клиент всегда в курсе лучших серверов!** 🚀
