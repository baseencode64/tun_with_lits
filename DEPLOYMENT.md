# 📦 Сборка и установка GoXRay на Debian 13

## ✅ Готовый бинарный файл

**Файл**: `goxray_linux_amd64`  
**Размер**: ~45.7 MB  
**Архитектура**: Linux amd64 (x86_64)  
**Статус**: ✅ Скомпилирован и готов к использованию

---

## 🚀 Быстрый старт

### Способ 1: Простое копирование

```bash
# 1. Скопируйте файл goxray_linux_amd64 на Debian сервер
scp goxray_linux_amd64 user@debian:/usr/local/bin/goxray

# 2. На Debian сервере:
ssh user@debian

# Сделать исполняемым
sudo chmod +x /usr/local/bin/goxray

# Настроить права
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# Запустить
sudo goxray --from-raw https://example.com/links.txt
```

### Способ 2: Автоматическая установка

```bash
# 1. Скопируйте файлы на сервер:
scp goxray_linux_amd64 install_goxray.sh user@debian:/tmp/

# 2. Запустите скрипт установки:
ssh user@debian
cd /tmp
chmod +x install_goxray.sh
sudo ./install_goxray.sh
```

Скрипт автоматически:
- Установит необходимые зависимости
- Проверит модуль TUN
- Установит бинарный файл
- Настроит capabilities
- Предложит создать systemd сервис

---

## 📋 Системные требования

### Минимальные требования

- **ОС**: Debian 13 (Bookworm) или новее
- **Архитектура**: amd64 (x86_64)
- **Ядро**: 4.0+ с поддержкой TUN/TAP
- **RAM**: 128 MB минимум
- **Диск**: 100 MB свободного места

### Необходимые пакеты

```bash
sudo apt update
sudo apt install -y iproute2 iputils-ping curl ca-certificates
```

### Проверка TUN модуля

```bash
# Проверить загрузку модуля
lsmod | grep tun

# Если не загружен, загрузить
sudo modprobe tun

# Проверить устройство
ls -la /dev/net/tun
```

---

## 🔧 Ручная установка (по шагам)

### Шаг 1: Копирование файла

```bash
sudo cp goxray_linux_amd64 /usr/local/bin/goxray
sudo chmod +x /usr/local/bin/goxray
```

### Шаг 2: Настройка прав доступа

Есть два варианта:

**Вариант A: Запуск от root**
```bash
sudo goxray --from-raw https://example.com/links.txt
```

**Вариант B: Использование capabilities (рекомендуется)**
```bash
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# Теперь можно запускать без sudo
goxray --from-raw https://example.com/links.txt
```

### Шаг 3: Проверка работы

```bash
# Проверить доступность
goxray --help

# Тестовое подключение
sudo goxray --from-raw https://example.com/links.txt
```

---

## 🐳 Docker установка (альтернатива)

### Сборка образа

```bash
# На Windows (из директории проекта)
docker build -t goxray .

# Или с указанием платформы
docker build --platform linux/amd64 -t goxray .
```

### Запуск контейнера

```bash
# С raw списком
docker run --rm -it \
  --cap-add NET_ADMIN \
  --cap-add NET_RAW \
  --device /dev/net/tun:/dev/net/tun \
  goxray --from-raw https://example.com/links.txt

# С прямой ссылкой
docker run --rm -it \
  --cap-add NET_ADMIN \
  --cap-add NET_RAW \
  --device /dev/net/tun:/dev/net/tun \
  goxray vless://uuid@server.com:443
```

### Docker Compose

Создайте `docker-compose.yml`:

```yaml
version: '3.8'

services:
  goxray:
    build: .
    container_name: goxray-vpn
    restart: unless-stopped
    command: ["--from-raw", "https://example.com/links.txt"]
    cap_add:
      - NET_ADMIN
      - NET_RAW
    devices:
      - /dev/net/tun:/dev/net/tun
    volumes:
      - goxray-data:/tmp

volumes:
  goxray-data:
```

Запуск:
```bash
docker-compose up -d
docker-compose logs -f
```

---

## 🎯 Использование

### Базовые команды

```bash
# Подключение из raw списка (рекомендуется)
sudo goxray --from-raw https://example.com/links.txt

# Прямое подключение
sudo goxray vless://uuid@server.com:443

# Помощь
goxray --help
```

### Управление через systemd (если установлен сервис)

```bash
# Запуск
sudo systemctl start goxray

# Автозапуск при загрузке
sudo systemctl enable goxray

# Просмотр статуса
sudo systemctl status goxray

# Логи в реальном времени
sudo journalctl -u goxray -f

# Остановка
sudo systemctl stop goxray

# Перезапуск
sudo systemctl restart goxray
```

---

## 🔍 Диагностика

### Проверка подключения

```bash
# Проверить маршрут по умолчанию
ip route show

# Проверить DNS
nslookup google.com

# Проверить внешний IP
curl -s https://api.ipify.org

# Пинг тест
ping -c 4 google.com
```

### Логи и ошибки

```bash
# Просмотр логов systemd
sudo journalctl -u goxray -n 50

# Логи с момента последней загрузки
sudo journalctl -u goxray -b

# Debug режим
sudo RUST_LOG=debug goxray --from-raw https://example.com/links.txt
```

### Распространенные проблемы

**Проблема: "permission denied"**
```bash
# Решение
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
```

**Проблема: "TUN device not found"**
```bash
# Загрузить модуль
sudo modprobe tun

# Создать устройство вручную
sudo mkdir -p /dev/net
sudo mknod /dev/net/tun c 10 200
sudo chmod 600 /dev/net/tun
```

**Проблема: "no available servers"**
```bash
# Проверить доступность URL
curl -I https://example.com/links.txt

# Проверить формат ссылок
cat links.txt

# Тест одного сервера вручную
sudo goxray vless://uuid@server.com:443
```

---

## 📊 Информация о файлах проекта

| Файл | Описание | Размер |
|------|----------|--------|
| `goxray_linux_amd64` | Готовый бинарник для Debian | ~45.7 MB |
| `install_goxray.sh` | Скрипт автоматической установки | ~3 KB |
| `INSTALL_DEBIAN.md` | Полная инструкция по установке | ~8 KB |
| `Dockerfile` | Образ Docker для контейнеризации | ~1 KB |
| `.dockerignore` | Исключения для Docker | ~0.5 KB |

---

## 🔐 Безопасность

### Рекомендации

1. **Используйте capabilities вместо root**:
   ```bash
   sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
   ```

2. **Всегда используйте HTTPS** для загрузки списков серверов:
   ```bash
   ✅ goxray --from-raw https://example.com/links.txt
   ❌ goxray --from-raw http://example.com/links.txt
   ```

3. **Настройте firewall**:
   ```bash
   sudo ufw default deny outgoing
   sudo ufw allow out 443/tcp
   sudo ufw allow out 80/tcp
   sudo ufw enable
   ```

4. **Регулярно обновляйте** бинарный файл при выходе новых версий

---

## 📞 Поддержка

При возникновении проблем:

1. 📖 Проверьте [INSTALL_DEBIAN.md](INSTALL_DEBIAN.md) - полная документация
2. 🔍 Просмотрите логи: `sudo journalctl -u goxray -f`
3. ✅ Убедитесь в загрузке TUN: `lsmod | grep tun`
4. 🔐 Проверьте права: `getcap /usr/local/bin/goxray`
5. 🌐 Протестируйте сеть: `ping`, `curl`, `traceroute`

---

## 🎉 Готово!

Бинарный файл **`goxray_linux_amd64`** успешно собран и готов к использованию на Debian 13!

**Следующие шаги:**
1. Скопируйте файл на Debian сервер
2. Выполните установку (ручную или автоматическую)
3. Настройте VPN подключение
4. Наслаждайтесь безопасным соединением! 🚀
