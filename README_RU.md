# Go VPN клиент для XRay

![Static Badge](https://img.shields.io/badge/OS-macOS%20%7C%20Linux-blue?style=flat&logo=linux&logoColor=white&logoSize=auto&color=blue)
![Static Badge](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go&logoColor=white)
[![Go Report Card](https://goreportcard.com/badge/github.com/goxray/tun)](https://goreportcard.com/report/github.com/goxray/tun)
[![Go Reference](https://pkg.go.dev/badge/github.com/goxray/tun.svg)](https://pkg.go.dev/github.com/goxray/tun)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/goxray/tun/total?color=blue)

Этот проект представляет собой полнофункциональный [XRay](https://github.com/XTLS/Xray-core) VPN клиент, реализованный на Go.

> Для десктопной версии см. https://github.com/goxray/desktop

<img alt="Пример вывода в терминале" align="center" src="/.github/images/carbon.png">

> [!NOTE]
> Программа не повреждает ваши правила маршрутизации. Основной маршрут остаётся нетронутым — добавляются только дополнительные правила на время работы TUN устройства. Также реализованы дополнительные процедуры очистки.

#### Что такое XRay?

Посетите https://xtls.github.io/en для получения дополнительной информации.

#### Протестировано и поддерживается на:

- macOS (протестировано на Sequoia 15.1.1)
- Linux (протестировано на Ubuntu 24.10)

> Протестируйте на вашей системе и сообщите об ошибках в Issues :)

## ✨ Возможности

- Чрезвычайно прост в использовании
- Поддерживает все протоколы [Xray-core](https://github.com/XTLS/Xray-core) (vless, vmess и др.) через ссылки (`vless://` и т.д.)
- Применяются только мягкие правила маршрутизации — изменения в основные маршруты не вносятся
- **IPv6 поддержка** — Полный dual-stack IPv4/IPv6 туннель (включение через `--ipv6`)
- **JSON логирование** — Структурированное логирование с автоматической ротацией
- **Prometheus метрики** — Мониторинг через эндпоинт `/metrics`
- **DNS защита** — Предотвращение утечек DNS (включение через `--dns-protection`)
- **Health monitoring** — Автоматический мониторинг здоровья подключения и failover
- **Auto-reconnect** — Автоматическое переподключение с exponential backoff при потере соединения
- **Smart server selection** — Выбор сервера по latency + packet loss scoring
- **Поддержка нескольких URL серверов** — Fallback между источниками списков серверов

## ⚡️ Установка

Приложение может использоваться как самостоятельный бинарный файл, скомпилированный и размещённый в директории из PATH.

##### 📦 Сторонний Debian пакет (поддерживается [twdragon](https://github.com/twdragon))

Клиент доступен из PPA репозитория `ppa:twdragon/xray`. Сетевые привилегии настраиваются автоматически postinstall-скриптом. Пакет синхронизирован с релизными тегами этого репозитория. Подробнее в [репозитории пакета](https://github.com/twdragon/xray-debian-pkg). Установка:

```bash
sudo add-apt-repository ppa:twdragon/xray
sudo apt update
sudo apt install goxray-cli
```

После установки пакет может обновляться автоматически (как в Ubuntu). Пакеты подписаны [twdragon](https://github.com/twdragon) и опубликованы на [Launchpad](https://launchpad.net/~twdragon/+archive/ubuntu/xray). Экспериментальные сборки доступны в [пайплайне](https://github.com/twdragon/xray-debian-pkg/actions).

## ⚡️ Использование

> [!IMPORTANT]
>
> - Требуется `sudo`
> - На Linux выполните: `sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /path/to/goxray`

### Автономное приложение:

Запуск VPN на вашей машине максимально прост:

```bash
sudo go run . <proto_link>
```

Где `proto_link` — это ваша XRay ссылка (например, `vless://example.com...`), которую можно получить от VPN-провайдера или XRay-сервера.

#### Использование конфигурационного файла (рекомендуется)

Создайте YAML конфигурационный файл для удобного управления:

```bash
# Скопировать пример конфига
cp config.yaml.example goxray.yaml

# Отредактировать под себя
nano goxray.yaml

# Запустить с конфигом
sudo go run . --config goxray.yaml
```

Конфигурационные файлы поддерживают все настройки: подключение, логирование, health monitoring, выбор сервера и реконнект. CLI аргументы переопределяют значения из конфигурационного файла.

**Множественные URL списков серверов с Fallback:**

```yaml
connection:
  from_raw_urls:
    - "https://primary.example.com/links.txt"
    - "https://backup1.example.com/links.txt"
    - "https://backup2.example.com/links.txt"
```

Клиент пробует каждый URL по порядку. Если первый недоступен, автоматически переключается на следующий.

#### Опции реконнекта (auto-reconnect)

При исчерпании всех серверов клиент автоматически переходит в режим переподключения с экспоненциальной задержкой:

```bash
sudo go run . --from-raw https://example.com/links.txt \
  --max-retries 0 \
  --min-backoff 5s \
  --max-backoff 5m \
  --backoff-factor 2.0
```

#### Опции логирования

Включение JSON логирования с ротацией:

```bash
sudo go run . --from-raw https://example.com/links.txt \
  --log-file /var/log/goxray/goxray.log \
  --log-format json \
  --log-level info
```

Или через конфигурационный файл (рекомендуется для сложных настроек):

```yaml
# goxray.yaml
connection:
  from_raw: "https://example.com/links.txt"
logging:
  format: "json"
  file: "/var/log/goxray/goxray.log"
  max_size: 200
  max_backups: 5
```

### Как библиотека в вашем проекте:

> [!NOTE]
> Этот проект построен на пакете `core`, подробнее на https://github.com/goxray/core

Установка:

```bash
go get github.com/goxray/tun/pkg/client
```

Пример:

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
vpn, _ := client.NewClientWithOpts(client.Config{
  TLSAllowInsecure: false,
  Logger:           logger,
})

_ = vpn.Connect(clientLink)
defer vpn.Disconnect(context.Background())

time.Sleep(60 * time.Second)
```

> Обратитесь к godoc для полного списка поддерживаемых методов и типов.

### Docker

Если вам нужно использовать приложение в Docker — смотрите [предложенную реализацию](https://github.com/goxray/tun/pull/8).

## 🛠 Сборка

Проект компилируется как обычная Go программа:

```bash
go build -o goxray_cli .
```

#### Кросс-компиляция

```bash
# Для macOS (Intel)
env GOOS=darwin GOARCH=amd64 go build -o goxray_cli_darwin_amd64 .

# Для Linux ARM64 (через Docker)
docker run --platform=linux/arm64 -v=${PWD}:/app --workdir=/app arm64v8/golang:1.24 env GOARCH=arm64 go build -o goxray_cli_linux_arm64 .

# Для Linux AMD64 (через Docker)
docker run --platform=linux/amd64 -v=${PWD}:/app --workdir=/app amd64/golang:1.24 env GOARCH=amd64 go build -o goxray_cli_linux_amd64 .
```

## Как это работает

- Приложение создаёт новый TUN-устройство.
- Добавляет дополнительные маршруты для направления всего системного трафика на это TUN-устройство.
- Добавляет исключение для исходящего адреса XRay (IP вашего VPN-сервера).
- Создаётся туннель для обработки всех входящих IP-пакетов через TCP/IP стек. Весь исходящий трафик направляется через входящий прокси XRay, а все входящие пакеты возвращаются обратно через TUN-устройство.

## 📝 TODO

- [x] Добавить защиту от утечек DNS
- [ ] Добавить веб-панель / TUI интерфейс
- [x] Добавить Prometheus метрики
- [x] Добавить поддержку конфигурационных файлов (YAML)
- [ ] Добавить kill switch
- [x] Добавить Health Monitoring & Auto-Failover
- [x] Добавить Auto-Reconnect с exponential backoff
