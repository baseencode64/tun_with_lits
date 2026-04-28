# Установка GoXRay на Debian 13

## 📦 Готовый бинарный файл

Бинарный файл `goxray_linux_amd64` скомпилирован и готов к использованию на Debian 13 (amd64).

---

## 🚀 Быстрая установка

### Шаг 1: Копирование файла на сервер

```bash
# С вашего Windows компьютера скопируйте файл на Debian сервер
scp goxray_linux_amd64 user@debian-server:/usr/local/bin/goxray
```

Или используйте любой другой способ передачи файлов (WinSCP, rsync, и т.д.)

### Шаг 2: Настройка прав доступа на Debian сервере

Подключитесь к Debian серверу и выполните:

```bash
# Сделать файл исполняемым
sudo chmod +x /usr/local/bin/goxray

# Проверить версию
goxray --help || true
```

### Шаг 3: Настройка сетевых привилегий

Для работы с TUN устройством требуются специальные права:

```bash
# Вариант A: Запуск от root (просто, но менее безопасно)
sudo goxray --from-raw https://example.com/links.txt

# Вариант B: Настройка capabilities (рекомендуется)
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray

# После настройки capabilities можно запускать без sudo
goxray --from-raw https://example.com/links.txt
```

---

## 📋 Системные требования Debian 13

### Необходимые пакеты

```bash
sudo apt update
sudo apt install -y \
    iproute2 \
    iputils-ping \
    curl \
    ca-certificates
```

### Проверка ядра

Убедитесь, что ядро поддерживает TUN/TAP:

```bash
# Проверить наличие модуля TUN
lsmod | grep tun

# Если модуль не загружен
sudo modprobe tun

# Проверить поддержку в ядре
zcat /proc/config.gz | grep CONFIG_TUN 2>/dev/null || grep CONFIG_TUN /boot/config-* 2>/dev/null
```

---

## 🔧 Способы установки

### Способ 1: Ручная установка (рекомендуется)

См. раздел "Быстрая установка" выше.

### Способ 2: Использование systemd service

Создайте файл сервиса `/etc/systemd/system/goxray.service`:

```ini
[Unit]
Description=GoXRay VPN Client
After=network.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/goxray --from-raw https://your-server.com/links.txt
Restart=on-failure
RestartSec=10
Capabilities=CAP_NET_RAW,CAP_NET_ADMIN,CAP_NET_BIND_SERVICE+eip
AmbientCapabilities=CAP_NET_RAW,CAP_NET_ADMIN,CAP_NET_BIND_SERVICE

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/tmp

[Install]
WantedBy=multi-user.target
```

Активация сервиса:

```bash
sudo systemctl daemon-reload
sudo systemctl enable goxray
sudo systemctl start goxray

# Проверить статус
sudo systemctl status goxray

# Просмотр логов
sudo journalctl -u goxray -f
```

### Способ 3: Установка из PPA (для Ubuntu/Debian)

```bash
# Добавить репозиторий
sudo add-apt-repository ppa:twdragon/xray
sudo apt update

# Установить пакет
sudo apt install goxray-cli
```

---

## 🎯 Использование

### Прямое подключение

```bash
# С прямой ссылкой
sudo goxray vless://uuid@server.com:443

# Из raw списка (рекомендуется)
sudo goxray --from-raw https://example.com/vless_links.txt
```

### Проверка подключения

```bash
# Проверить маршрут
ip route show

# Проверить DNS
ping -c 4 google.com

# Проверить IP адрес
curl -s https://api.ipify.org
```

---

## 🔍 Диагностика проблем

### Логирование

```bash
# Запуск с подробным логированием
sudo RUST_LOG=debug goxray --from-raw https://example.com/links.txt

# Или через systemd
sudo journalctl -u goxray -n 50 --no-pager
```

### Распространенные ошибки

**Ошибка: "permission denied"**
```bash
# Решение: добавить capabilities
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
```

**Ошибка: "TUN device not available"**
```bash
# Загрузить модуль TUN
sudo modprobe tun

# Проверить права доступа
ls -la /dev/net/tun
```

**Ошибка: "connection refused"**
```bash
# Проверить доступность серверов
ping server.com
curl -I https://example.com/links.txt
```

---

## 📊 Информация о бинарном файле

| Параметр | Значение |
|----------|----------|
| **Файл** | `goxray_linux_amd64` |
| **Размер** | ~45.7 MB |
| **Архитектура** | amd64 (x86_64) |
| **ОС** | Linux |
| **Компилятор** | Go 1.25.6 |
| **Статическая линковка** | Да |

Проверка на Debian:
```bash
file goxray_linux_amd64
# Ожидаемый вывод: ELF 64-bit LSB executable, x86-64, statically linked
```

---

## 🔐 Безопасность

### Рекомендации

1. **Используйте capabilities вместо root**:
   ```bash
   sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/goxray
   ```

2. **Настройте firewall**:
   ```bash
   sudo ufw allow out 443/tcp
   sudo ufw allow out 80/tcp
   ```

3. **Регулярно обновляйте бинарный файл**:
   ```bash
   # Проверить версию
   goxray version 2>&1 || echo "Version command not available"
   ```

4. **Используйте TLS для загрузки списков**:
   ```bash
   # Всегда используйте HTTPS
   goxray --from-raw https://...  # ✅ Правильно
   goxray --from-raw http://...   # ❌ Небезопасно
   ```

---

## 📞 Поддержка

При возникновении проблем:

1. Проверьте логи: `journalctl -u goxray -f`
2. Убедитесь в загрузке модуля TUN: `lsmod | grep tun`
3. Проверьте права доступа: `getcap /usr/local/bin/goxray`
4. Протестируйте подключение к серверам вручную

---

## 🎉 Готово!

После выполнения установки вы можете использовать все функции GoXRay:

```bash
# Автоматический выбор лучшего сервера
goxray --from-raw https://your-server.com/vless_list.txt

# Или прямое подключение
goxray vless://uuid@server.com:443
```

**Happy connecting!** 🚀
