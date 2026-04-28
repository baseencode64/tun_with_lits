#!/bin/bash
# Автоматический скрипт установки GoXRay на Debian 13
# Использование: sudo ./install_goxray.sh

set -e  # Выход при ошибке

BINARY_NAME="goxray"
INSTALL_DIR="/usr/local/bin"
BINARY_PATH="${INSTALL_DIR}/${BINARY_NAME}"
RAW_LIST_URL=""  # Можно указать URL со списком серверов

echo "╔══════════════════════════════════════════╗"
echo "║   GoXRay VPN Client Installer for Debian ║"
echo "╚══════════════════════════════════════════╝"
echo ""

# Проверка прав root
if [ "$EUID" -ne 0 ]; then 
    echo "❌ Ошибка: Запустите скрипт от root (sudo ./install_goxray.sh)"
    exit 1
fi

# Определение архитектуры
ARCH=$(uname -m)
case $ARCH in
    x86_64) GOARCH="amd64" ;;
    aarch64) GOARCH="arm64" ;;
    *) echo "❌ Неподдерживаемая архитектура: $ARCH"; exit 1 ;;
esac

echo "✓ Архитектура: $ARCH"

# Проверка наличия бинарного файла
if [ ! -f "./${BINARY_NAME}_linux_${GOARCH}" ]; then
    echo "❌ Бинарный файл ${BINARY_NAME}_linux_${GOARCH} не найден!"
    echo "   Сначала скопируйте его в текущую директорию"
    exit 1
fi

echo "✓ Бинарный файл найден"

# Установка системных зависимостей
echo ""
echo "📦 Установка системных зависимостей..."
apt-get update -qq
apt-get install -y -qq iproute2 iputils-ping curl ca-certificates > /dev/null 2>&1
echo "✓ Зависимости установлены"

# Проверка модуля TUN
echo ""
echo "🔍 Проверка TUN модуля..."
if ! lsmod | grep -q tun; then
    echo "⚠ Модуль TUN не загружен. Загрузка..."
    modprobe tun || echo "⚠ Не удалось загрузить модуль TUN автоматически"
else
    echo "✓ Модуль TUN загружен"
fi

# Установка бинарного файла
echo ""
echo "📥 Установка GoXRay..."
cp "./${BINARY_NAME}_linux_${GOARCH}" "${BINARY_PATH}"
chmod +x "${BINARY_PATH}"
echo "✓ Установлен в: ${BINARY_PATH}"

# Настройка capabilities
echo ""
echo "🔐 Настройка сетевых привилегий..."
setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip "${BINARY_PATH}"
echo "✓ Capabilities настроены"

# Создание systemd сервиса (опционально)
echo ""
read -p "Создать systemd сервис для автозапуска? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    read -p "Введите URL с raw списком серверов (или оставьте пустым для ручной настройки): " RAW_URL
    
    cat > /etc/systemd/system/goxray.service << EOF
[Unit]
Description=GoXRay VPN Client
After=network.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BINARY_PATH} --from-raw ${RAW_URL:-https://example.com/links.txt}
Restart=on-failure
RestartSec=10
AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN CAP_NET_BIND_SERVICE

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/tmp

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable goxray
    systemctl start goxray
    
    echo "✓ systemd сервис создан и запущен"
    echo "  Управление:"
    echo "    - Старт:     sudo systemctl start goxray"
    echo "    - Стоп:      sudo systemctl stop goxray"
    echo "    - Статус:    sudo systemctl status goxray"
    echo "    - Логи:      sudo journalctl -u goxray -f"
else
    echo "ℹ systemd сервис не создан"
fi

# Проверка установки
echo ""
echo "🧪 Проверка установки..."
if command -v goxray &> /dev/null; then
    echo "✓ GoXRay доступен в PATH"
    echo "  Версия: $(goxray --help 2>&1 | head -n 1 || echo 'N/A')"
else
    echo "❌ GoXRay не найден в PATH"
fi

# Итоговая информация
echo ""
echo "╔════════════════════════════════════════════════════╗"
echo "║          ✅ Установка завершена!                  ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""
echo "📖 Использование:"
echo "   Ручной запуск:"
echo "     goxray --from-raw <URL>"
echo "     goxray vless://uuid@server.com:443"
echo ""
echo "   Просмотр логов (если установлен как сервис):"
echo "     sudo journalctl -u goxray -f"
echo ""
echo "🔗 Документация:"
echo "   INSTALL_DEBIAN.md"
echo ""
echo "Удачи! 🚀"
