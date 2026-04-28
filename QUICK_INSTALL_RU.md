# 📦 Инструкция по установке на Debian 13

## ✅ Бинарный файл готов

**Файл**: `goxray_linux_amd64`  
**Размер**: 43.6 MB  
**Архитектура**: Linux amd64 (x86_64)  
**Готов к использованию**: ✅ Да

---

## 🚀 Быстрая установка

### Шаг 1: Копирование на сервер

```bash
# С Windows на Debian сервер (через SCP)
scp goxray_linux_amd64 user@debian-server:/usr/local/bin/goxray

# Или через rsync
rsync -avz goxray_linux_amd64 user@debian-server:/usr/local/bin/goxray
```

### Шаг 2: Настройка на Debian сервере

```bash
# Подключиться к серверу
ssh user@debian-server

# Сделать файл исполняемым
sudo chmod +x /usr/local/bin/goxray

# Настроить capabilities (безопаснее чем root)
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
```

### Шаг 3: Проверка зависимостей

```bash
# Установить необходимые пакеты
sudo apt update
sudo apt install -y iproute2 iputils-ping curl ca-certificates

# Проверить модуль TUN
lsmod | grep tun
# Если не загружен:
sudo modprobe tun
```

### Шаг 4: Использование

```bash
# Автоматический выбор лучшего сервера с fallback
sudo goxray --from-raw https://example.com/links.txt

# Или прямая ссылка
sudo goxray vless://uuid@server.com:443
```

---

## 🔧 Ручная установка (альтернативный способ)

```bash
# Создать директорию
sudo mkdir -p /opt/goxray

# Скопировать бинарный файл
sudo cp goxray_linux_amd64 /opt/goxray/goxray
sudo chmod +x /opt/goxray/goxray

# Создать symlink для удобного запуска
sudo ln -s /opt/goxray/goxray /usr/local/bin/goxray

# Настроить capabilities
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /opt/goxray/goxray
```

---

## 🐳 Docker установка

Если предпочитаете контейнеризацию:

```bash
# Сборка образа
docker build -t goxray .

# Запуск с поддержкой TUN
docker run --rm -it \
  --cap-add NET_ADMIN \
  --cap-add NET_RAW \
  --device /dev/net/tun:/dev/net/tun \
  goxray --from-raw https://example.com/links.txt
```

---

## ⚙️ systemd сервис (для постоянного использования)

Создайте файл `/etc/systemd/system/goxray.service`:

```ini
[Unit]
Description=GoXRAY VPN Client
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/goxray --from-raw https://example.com/links.txt
Restart=on-failure
RestartSec=10
Capabilities=CAP_NET_RAW,CAP_NET_ADMIN,CAP_NET_BIND_SERVICE+ep
ProtectSystem=full
ProtectHome=true

[Install]
WantedBy=multi-user.target
```

Активация сервиса:

```bash
sudo systemctl daemon-reload
sudo systemctl enable goxray
sudo systemctl start goxray

# Проверка статуса
sudo systemctl status goxray

# Просмотр логов
sudo journalctl -u goxray -f
```

---

## 🔍 Диагностика

### Проверка бинарного файла

```bash
# Информация о файле
file /usr/local/bin/goxray
# Ожидаемый вывод: ELF 64-bit LSB executable, x86-64

# Проверка capabilities
getcap /usr/local/bin/goxray
# Ожидаемый вывод: cap_net_admin,cap_net_bind_service,cap_net_raw=eip
```

### Проверка работы

```bash
# Запуск в режиме отладки
sudo RUST_LOG=debug goxray --from-raw https://example.com/links.txt

# Проверка сетевого интерфейса TUN
ip addr show | grep tun

# Проверка маршрутов
ip route show
```

---

## ❗ Устранение проблем

### Проблема: "Permission denied"
**Решение**:
```bash
sudo chmod +x /usr/local/bin/goxray
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
```

### Проблема: "TUN device not available"
**Решение**:
```bash
sudo modprobe tun
echo 'tun' | sudo tee -a /etc/modules
```

### Проблема: "Connection refused"
**Решение**:
- Проверьте доступность серверов из raw списка
- Увеличьте timeout если сеть медленная
- Проверьте firewall настройки

---

## 📊 Характеристики

| Параметр | Значение |
|----------|----------|
| **ОС** | Linux (Debian 13 Bookworm) |
| **Архитектура** | amd64 (x86_64) |
| **Минимальная RAM** | 128 MB |
| **Место на диске** | ~100 MB |
| **Ядро** | 4.0+ с поддержкой TUN/TAP |
| **Компилятор** | Go 1.25.6 |
| **Линковка** | Статическая |

---

## 🎯 Возможности

✅ **Автоматический выбор сервера** из raw списка  
✅ **Fallback логика** - перебирает серверы при недоступности  
✅ **Измерение latency** и сортировка по скорости  
✅ **Параллельная проверка** до 10 серверов одновременно  
✅ **Production-ready** код с comprehensive тестами  

---

## 📚 Документация

Полная документация доступна в файлах:

- 📘 **[README.md](README.md)** - Основная документация
- 📗 **[FALLBACK_RU.md](FALLBACK_RU.md)** - Fallback логика на русском
- 📙 **[INSTALL_DEBIAN.md](INSTALL_DEBIAN.md)** - Детальная инструкция
- 📕 **[DEPLOYMENT.md](DEPLOYMENT.md)** - Руководство по развертыванию

---

## ✨ Готово!

Бинарный файл успешно собран и готов к использованию на Debian 13! 🚀

Для быстрого старта используйте:
```bash
sudo goxray --from-raw https://example.com/links.txt
```
