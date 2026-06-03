# 📚 GoXRay Documentation

Добро пожаловать в документацию GoXRay - VPN клиента на базе Xray-core с поддержкой TUN устройства.

---

## 📖 Содержание

### 🚀 Начало работы

- [Быстрый старт](getting-started/QUICKSTART.md) - Установка и первый запуск
- [Системные требования](getting-started/SYSTEM_REQUIREMENTS.md) - Минимальные требования
- [Установка на Debian](getting-started/INSTALL_DEBIAN.md) - Пошаговая установка

### ⚙️ Конфигурация

- [Основная конфигурация](configuration/CONFIG_REFERENCE.md) - Полное описание config.yaml
- [CLI флаги](configuration/CLI_FLAGS.md) - Параметры командной строки
- [Переменные окружения](configuration/ENV_VARS.md) - Environment variables

### 🔧 Функции

- [Kill Switch](features/KILLSWITCH.md) - Защита от IP утечек
- [Split Tunneling](features/SPLIT_TUNNELING.md) - Выборочная маршрутизация
- [SOCKS5 Proxy](features/SOCKS5_PROXY.md) - Встроенный SOCKS5 сервер
- [DNS Protection](features/DNS_PROTECTION.md) - Защита от DNS утечек
- [Health Monitoring](features/HEALTH_MONITORING.md) - Мониторинг соединения
- [Metrics](features/METRICS.md) - Prometheus метрики

### 🐳 Развертывание

- [Docker Deployment](deployment/DOCKER.md) - Запуск в Docker
- [Systemd Service](deployment/SYSTEMD.md) - Настройка systemd сервиса
- [Production Deployment](deployment/PRODUCTION.md) - Production окружение

### 🔍 Troubleshooting

- [FAQ](troubleshooting/FAQ.md) - Часто задаваемые вопросы
- [Common Issues](troubleshooting/COMMON_ISSUES.md) - Типичные проблемы
- [Debugging](troubleshooting/DEBUGGING.md) - Отладка

### 📦 Releases

- [v1.7.0](../RELEASE_v1.7.0.md) - Текущая версия (Split Tunneling, SOCKS5, Kill Switch fixes)
- [Changelog](../CHANGELOG.md) - История изменений
- [Migration Guide](guides/MIGRATION.md) - Миграция между версиями

### 🏗️ Разработка

- [Project Structure](development/PROJECT_STRUCTURE.md) - Структура проекта
- [Contributing](development/CONTRIBUTING.md) - Как внести вклад
- [Architecture](development/ARCHITECTURE.md) - Архитектура системы

---

## 🎯 Быстрые ссылки

### Для пользователей

- **Первый запуск**: [Quickstart Guide](getting-started/QUICKSTART.md)
- **Настройка Kill Switch**: [Kill Switch Guide](features/KILLSWITCH.md)
- **Настройка Split Tunneling**: [Split Tunneling Guide](features/SPLIT_TUNNELING.md)
- **Проблемы**: [Troubleshooting](troubleshooting/COMMON_ISSUES.md)

### Для администраторов

- **Production Deployment**: [Production Guide](deployment/PRODUCTION.md)
- **Docker Setup**: [Docker Guide](deployment/DOCKER.md)
- **Monitoring**: [Health Monitoring](features/HEALTH_MONITORING.md)
- **Metrics**: [Prometheus Metrics](features/METRICS.md)

### Для разработчиков

- **Architecture**: [System Architecture](development/ARCHITECTURE.md)
- **Code Structure**: [Project Structure](development/PROJECT_STRUCTURE.md)
- **Contributing**: [Contribution Guide](development/CONTRIBUTING.md)

---

## 📋 Версии документации

- **Русский**: Основная документация на русском языке
- **English**: [English documentation](../README.md)

---

## 🆘 Поддержка

- **Issues**: [GitHub Issues](https://github.com/baseencode64/tun_with_lits/issues)
- **Discussions**: [GitHub Discussions](https://github.com/baseencode64/tun_with_lits/discussions)

---

**Версия документации**: v1.7.0  
**Последнее обновление**: 2026-06-03
