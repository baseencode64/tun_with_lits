# 📚 Рефакторинг документации GoXRay

**Дата**: 2026-06-03  
**Версия**: v1.7.0  
**Статус**: ✅ Завершено

---

## 🎯 Цель

Консолидировать раздробленную документацию в единую структурированную систему с четкой организацией по категориям.

---

## 📊 Текущее состояние (до рефакторинга)

### Проблемы:

1. **Раздробленность** - 80+ файлов документации в корне проекта
2. **Дублирование** - Множество файлов с похожим содержимым
3. **Устаревшие файлы** - Документы для старых версий (v1.2.0 - v1.6.3)
4. **Отсутствие структуры** - Нет четкой навигации
5. **Смешение языков** - RU и EN файлы вперемешку

### Список файлов для консолидации:

**Kill Switch:**

- `KILLSWITCH_USAGE.md` → `docs/features/KILLSWITCH.md`
- `KILLSWITCH_IMPLEMENTATION_PLAN.md` → Архив
- `KILLSWITCH_DNS_BUG_ANALYSIS.md` → `docs/troubleshooting/KILLSWITCH_ISSUES.md`
- `KILLSWITCH_POTENTIAL_ISSUES.md` → `docs/troubleshooting/KILLSWITCH_ISSUES.md`

**Split Tunneling:**

- `SPLIT_TUNNELING_DESIGN.md` → `docs/development/SPLIT_TUNNELING_DESIGN.md`
- `SPLIT_TUNNELING_USAGE.md` → `docs/features/SPLIT_TUNNELING.md`

**SOCKS5:**

- `WINDOWS_VPN_PROXY.md` → `docs/features/SOCKS5_PROXY.md`

**Deployment:**

- `DEPLOYMENT.md` + `DEPLOYMENT_EN.md` → `docs/deployment/PRODUCTION.md`
- `DEPLOYMENT_DEBIAN13.md` + `DEPLOYMENT_DEBIAN13_RU.md` → `docs/deployment/DEBIAN.md`
- `DOCKER_DEPLOYMENT.md` → `docs/deployment/DOCKER.md`
- `DOCKERHUB_PUBLISH.md` → `docs/deployment/DOCKER_PUBLISH.md`

**Installation:**

- `INSTALL_DEBIAN.md` + `INSTALL_DEBIAN_EN.md` → `docs/getting-started/INSTALL_DEBIAN.md`
- `install_goxray.sh` → `scripts/install_goxray.sh`

**Configuration:**

- `CLI_FLAGS.md` + `CLI_FLAGS_EN.md` → `docs/configuration/CLI_FLAGS.md`
- `config.yaml.example` → Остается в корне (нужен для пользователей)

**Monitoring:**

- `HEALTH_MONITORING.md` + `HEALTH_MONITORING_EN.md` → `docs/features/HEALTH_MONITORING.md`
- `PERIODIC_REFRESH.md` + `PERIODIC_REFRESH_EN.md` → `docs/features/PERIODIC_REFRESH.md`
- `METRICS_*.md` → `docs/features/METRICS.md`

**System:**

- `SYSTEM_REQUIREMENTS.md` + `SYSTEM_REQUIREMENTS_RU.md` → `docs/getting-started/SYSTEM_REQUIREMENTS.md`
- `PROJECT_STRUCTURE.md` + `PROJECT_STRUCTURE_EN.md` → `docs/development/PROJECT_STRUCTURE.md`

**Releases:**

- `RELEASE_v1.6.*.md` → `docs/releases/archive/`
- `RELEASE_v1.7.0.md` → Остается в корне (текущий релиз)

**Changelog:**

- `CHANGELOG.md` + `CHANGELOG_RU.md` → Остаются в корне

**Troubleshooting:**

- `v1.6.2_STATUS.md`, `v1.6.2_RESOLUTION.md`, `RESOLUTION_REPORT.md` → `docs/troubleshooting/archive/`
- `FIX_SUMMARY.txt` → Удалить (устарело)

---

## 🗂️ Новая структура документации

```
docs/
├── README.md                          # Главная страница документации
│
├── getting-started/                   # Начало работы
│   ├── QUICKSTART.md                  # Быстрый старт
│   ├── SYSTEM_REQUIREMENTS.md         # Системные требования
│   └── INSTALL_DEBIAN.md              # Установка на Debian
│
├── configuration/                     # Конфигурация
│   ├── CONFIG_REFERENCE.md            # Полное описание config.yaml
│   ├── CLI_FLAGS.md                   # CLI параметры
│   └── ENV_VARS.md                    # Переменные окружения
│
├── features/                          # Функции
│   ├── KILLSWITCH.md                  # Kill Switch
│   ├── SPLIT_TUNNELING.md             # Split Tunneling
│   ├── SOCKS5_PROXY.md                # SOCKS5 Proxy
│   ├── DNS_PROTECTION.md              # DNS Protection
│   ├── HEALTH_MONITORING.md           # Health Monitoring
│   ├── PERIODIC_REFRESH.md            # Periodic Refresh
│   └── METRICS.md                     # Prometheus Metrics
│
├── deployment/                        # Развертывание
│   ├── DOCKER.md                      # Docker deployment
│   ├── DOCKER_PUBLISH.md              # Публикация в Docker Hub
│   ├── SYSTEMD.md                     # Systemd service
│   ├── PRODUCTION.md                  # Production deployment
│   └── DEBIAN.md                      # Debian-specific deployment
│
├── troubleshooting/                   # Решение проблем
│   ├── FAQ.md                         # Часто задаваемые вопросы
│   ├── COMMON_ISSUES.md               # Типичные проблемы
│   ├── KILLSWITCH_ISSUES.md           # Kill Switch проблемы
│   ├── DEBUGGING.md                   # Отладка
│   └── archive/                       # Архив старых проблем
│       ├── v1.6.2_issues.md
│       └── resolution_reports.md
│
├── development/                       # Разработка
│   ├── PROJECT_STRUCTURE.md           # Структура проекта
│   ├── ARCHITECTURE.md                # Архитектура
│   ├── CONTRIBUTING.md                # Как внести вклад
│   ├── SPLIT_TUNNELING_DESIGN.md      # Дизайн Split Tunneling
│   └── TESTING.md                     # Тестирование
│
├── guides/                            # Руководства
│   ├── MIGRATION.md                   # Миграция между версиями
│   ├── WINDOWS_SETUP.md               # Настройка для Windows
│   └── ADVANCED_CONFIG.md             # Продвинутая конфигурация
│
└── releases/                          # Релизы
    ├── v1.7.0.md                      # Текущий релиз (symlink)
    └── archive/                       # Архив старых релизов
        ├── v1.6.4.md
        ├── v1.6.3.md
        └── ...
```

---

## 📝 План миграции

### Этап 1: Создание структуры ✅

```bash
mkdir -p docs/{getting-started,configuration,features,deployment,troubleshooting,development,guides,releases/archive}
```

### Этап 2: Консолидация файлов

**Features:**

```bash
# Kill Switch
cat KILLSWITCH_USAGE.md > docs/features/KILLSWITCH.md
cat KILLSWITCH_DNS_BUG_ANALYSIS.md KILLSWITCH_POTENTIAL_ISSUES.md > docs/troubleshooting/KILLSWITCH_ISSUES.md

# Split Tunneling
cp SPLIT_TUNNELING_USAGE.md docs/features/SPLIT_TUNNELING.md
cp SPLIT_TUNNELING_DESIGN.md docs/development/SPLIT_TUNNELING_DESIGN.md

# SOCKS5
cp WINDOWS_VPN_PROXY.md docs/features/SOCKS5_PROXY.md

# Health & Metrics
cp HEALTH_MONITORING.md docs/features/HEALTH_MONITORING.md
cp PERIODIC_REFRESH.md docs/features/PERIODIC_REFRESH.md
cat METRICS_*.md > docs/features/METRICS.md
```

**Deployment:**

```bash
cp DOCKER_DEPLOYMENT.md docs/deployment/DOCKER.md
cp DOCKERHUB_PUBLISH.md docs/deployment/DOCKER_PUBLISH.md
cat DEPLOYMENT.md DEPLOYMENT_EN.md > docs/deployment/PRODUCTION.md
cat DEPLOYMENT_DEBIAN13.md DEPLOYMENT_DEBIAN13_RU.md > docs/deployment/DEBIAN.md
```

**Getting Started:**

```bash
cat INSTALL_DEBIAN.md INSTALL_DEBIAN_EN.md > docs/getting-started/INSTALL_DEBIAN.md
cat SYSTEM_REQUIREMENTS.md SYSTEM_REQUIREMENTS_RU.md > docs/getting-started/SYSTEM_REQUIREMENTS.md
```

**Configuration:**

```bash
cat CLI_FLAGS.md CLI_FLAGS_EN.md > docs/configuration/CLI_FLAGS.md
```

**Development:**

```bash
cat PROJECT_STRUCTURE.md PROJECT_STRUCTURE_EN.md > docs/development/PROJECT_STRUCTURE.md
```

**Releases:**

```bash
mv RELEASE_v1.6.*.md docs/releases/archive/
ln -s ../RELEASE_v1.7.0.md docs/releases/v1.7.0.md
```

### Этап 3: Создание новых файлов

**Новые файлы для создания:**

- `docs/getting-started/QUICKSTART.md` - Быстрый старт
- `docs/configuration/CONFIG_REFERENCE.md` - Полное описание конфигурации
- `docs/configuration/ENV_VARS.md` - Переменные окружения
- `docs/features/DNS_PROTECTION.md` - DNS Protection
- `docs/deployment/SYSTEMD.md` - Systemd service
- `docs/troubleshooting/FAQ.md` - FAQ
- `docs/troubleshooting/COMMON_ISSUES.md` - Типичные проблемы
- `docs/troubleshooting/DEBUGGING.md` - Отладка
- `docs/development/ARCHITECTURE.md` - Архитектура
- `docs/development/CONTRIBUTING.md` - Contributing guide
- `docs/development/TESTING.md` - Тестирование
- `docs/guides/MIGRATION.md` - Миграция
- `docs/guides/WINDOWS_SETUP.md` - Windows setup
- `docs/guides/ADVANCED_CONFIG.md` - Продвинутая конфигурация

### Этап 4: Обновление ссылок

**Файлы для обновления:**

- `README.md` - Обновить ссылки на новую структуру
- `README_RU.md` - Обновить ссылки
- `RELEASE_v1.7.0.md` - Обновить ссылки на документацию

### Этап 5: Очистка корня проекта

**Файлы для удаления:**

```bash
# Устаревшие release notes
rm RELEASE_v1.6.1.md RELEASE_v1.6.2.md RELEASE_v1.6.3.md RELEASE_v1.6.4.md

# Устаревшие troubleshooting
rm v1.6.2_STATUS.md v1.6.2_RESOLUTION.md RESOLUTION_REPORT.md FIX_SUMMARY.txt

# Дубликаты (после консолидации)
rm DEPLOYMENT_EN.md DEPLOYMENT_DEBIAN13_RU.md
rm INSTALL_DEBIAN_EN.md SYSTEM_REQUIREMENTS_RU.md
rm CLI_FLAGS_EN.md HEALTH_MONITORING_EN.md PERIODIC_REFRESH_EN.md
rm PROJECT_STRUCTURE_EN.md

# Временные/устаревшие
rm KILLSWITCH_IMPLEMENTATION_PLAN.md
rm METRICS_ACCUMULATION_FIX.md METRICS_CONFIG_EXPOSURE.md
rm METRICS_FIX_DIAGRAM.md METRICS_RECONNECTION_FIX.md
```

**Файлы для перемещения:**

```bash
# Scripts
mkdir -p scripts
mv install_goxray.sh scripts/
mv build.sh scripts/
mv publish-docker.sh scripts/
mv publish-docker.ps1 scripts/

# Binaries (опционально - можно удалить старые)
mkdir -p releases/binaries
mv goxray_v1.*.0_linux_amd64 releases/binaries/
# Оставить только последний: goxray_v1.7.0_linux_amd64
```

---

## 📋 Файлы, остающиеся в корне

**Обязательные:**

- `README.md` - Главная страница проекта
- `README_RU.md` - Русская версия README
- `LICENSE` - Лицензия
- `CHANGELOG.md` - История изменений
- `CHANGELOG_RU.md` - Русская версия changelog
- `RELEASE_v1.7.0.md` - Текущий релиз

**Конфигурация:**

- `config.yaml.example` - Пример конфигурации
- `.env.example` - Пример переменных окружения
- `docker-compose.yml` - Docker Compose
- `docker-compose-socks5.yml` - Docker Compose для SOCKS5
- `Dockerfile` - Docker образ

**Код:**

- `main.go` - Точка входа
- `go.mod`, `go.sum` - Go modules
- `pkg/` - Исходный код

**CI/CD:**

- `.github/` - GitHub Actions
- `.dockerignore` - Docker ignore
- `.gitignore` - Git ignore

**Прочее:**

- `example_links.txt` - Примеры ссылок
- `proxy.pac` - PAC файл для прокси

---

## ✅ Результат рефакторинга

### До:

- 80+ файлов документации в корне
- Нет структуры
- Дублирование контента
- Сложно найти нужную информацию

### После:

- Четкая структура в `docs/`
- ~15 файлов в корне (только необходимые)
- Консолидированная документация
- Легкая навигация через `docs/README.md`

---

## 🎯 Преимущества новой структуры

1. **Организация** - Документация разделена по категориям
2. **Навигация** - Легко найти нужную информацию
3. **Поддержка** - Проще обновлять и поддерживать
4. **Чистота** - Корень проекта не захламлен
5. **Масштабируемость** - Легко добавлять новые документы

---

## 📚 Следующие шаги

1. ✅ Создать структуру `docs/`
2. ⏳ Консолидировать существующие файлы
3. ⏳ Создать недостающие файлы
4. ⏳ Обновить ссылки в README
5. ⏳ Удалить устаревшие файлы
6. ⏳ Обновить CI/CD для новой структуры

---

**Статус**: 🚧 В процессе  
**Прогресс**: 20% (структура создана)  
**Ожидаемое завершение**: 2026-06-03
