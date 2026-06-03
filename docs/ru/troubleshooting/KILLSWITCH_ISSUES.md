# 🔧 Kill Switch - Troubleshooting Guide

Руководство по решению проблем с Kill Switch функциональностью.

---

## 🐛 Известные проблемы и решения

### 1. DNS блокировка при reconnection (ИСПРАВЛЕНО в v1.7.0)

**Проблема**: После длительной работы (24+ часа) при попытке переподключения возникает ошибка:

```
write udp 192.168.88.252:54261->192.168.88.1:53: write: operation not permitted
```

**Симптомы**:

- ❌ DNS запросы блокируются
- ❌ Невозможно получить список серверов
- ❌ Клиент входит в бесконечный reconnection loop
- ❌ Проблема возникает только после длительной работы

**Root Cause**:
Kill Switch блокировал все DNS запросы, включая те, которые необходимы для получения нового списка серверов при reconnection.

**Решение** (реализовано в v1.7.0):

- ✅ DNS запросы к gateway (192.168.x.1:53) разрешены
- ✅ DNS запросы к публичным DNS (8.8.8.8, 1.1.1.1, etc.) разрешены
- ✅ Поддержка UDP и TCP DNS (port 53)
- ✅ Reconnection работает надежно

**Обновление**:

```bash
# Обновитесь до v1.7.0 или новее
sudo systemctl stop goxray
sudo cp goxray_v1.7.0_linux_amd64 /usr/local/bin/goxray
sudo systemctl start goxray
```

---

### 2. IPv6 leak при включенном Kill Switch

**Проблема**: Если IPv6 включен в системе, но Kill Switch настроен только для IPv4, трафик может утекать через IPv6.

**Симптомы**:

- IPv4 трафик заблокирован
- IPv6 трафик проходит (IP leak!)
- Реальный IPv6 адрес виден внешним сервисам

**Решение** (реализовано в v1.7.0):
Kill Switch теперь применяется к IPv6 только если `EnableIPv6: true` в конфигурации.

**Проверка**:

```bash
# Проверить IPv6 Kill Switch правила
sudo ip6tables -L goxray_killswitch -n -v

# Если EnableIPv6: true - должны быть правила
# Если EnableIPv6: false - цепочка не должна существовать
```

**Конфигурация**:

```yaml
# config.yaml
connection:
  enable_ipv6: true # Включить IPv6 поддержку
  enable_kill_switch: true
```

---

### 3. Kill Switch остается активным после аварийного завершения

**Проблема**: Если процесс goxray завершается аварийно (kill -9, kernel panic, power loss), Kill Switch остается активным и блокирует весь трафик даже после перезагрузки.

**Симптомы**:

- После перезагрузки сервера нет интернета
- iptables содержит правила `goxray_killswitch`
- Требуется ручная очистка

**Решение 1: Ручная очистка**

```bash
# IPv4 cleanup
sudo iptables -D OUTPUT -j goxray_killswitch
sudo iptables -F goxray_killswitch
sudo iptables -X goxray_killswitch

# IPv6 cleanup
sudo ip6tables -D OUTPUT -j goxray_killswitch
sudo ip6tables -F goxray_killswitch
sudo ip6tables -X goxray_killswitch

# Verify cleanup
sudo iptables -L OUTPUT -n
sudo ip6tables -L OUTPUT -n
```

**Решение 2: Startup cleanup script**

Создайте systemd service для автоматической очистки при загрузке:

```bash
# /etc/systemd/system/goxray-cleanup.service
[Unit]
Description=GoXRay Kill Switch Cleanup
Before=network.target
DefaultDependencies=no

[Service]
Type=oneshot
ExecStart=/usr/local/bin/goxray-killswitch-cleanup.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
```

```bash
# /usr/local/bin/goxray-killswitch-cleanup.sh
#!/bin/bash

# Cleanup IPv4 kill switch
iptables -D OUTPUT -j goxray_killswitch 2>/dev/null
iptables -F goxray_killswitch 2>/dev/null
iptables -X goxray_killswitch 2>/dev/null

# Cleanup IPv6 kill switch
ip6tables -D OUTPUT -j goxray_killswitch 2>/dev/null
ip6tables -F goxray_killswitch 2>/dev/null
ip6tables -X goxray_killswitch 2>/dev/null

echo "Kill switch cleanup completed"
```

```bash
# Установка
sudo chmod +x /usr/local/bin/goxray-killswitch-cleanup.sh
sudo systemctl enable goxray-cleanup.service
```

---

### 4. Kill Switch блокирует локальную сеть

**Проблема**: Kill Switch может блокировать доступ к локальным ресурсам (NAS, принтеры, локальные сервисы).

**Симптомы**:

- Нет доступа к 192.168.x.x, 10.x.x.x
- SSH к локальным серверам не работает
- Локальные сервисы недоступны

**Текущее поведение**:

- ✅ Loopback (127.0.0.1) разрешен
- ❌ Локальная сеть (192.168.x.x) НЕ разрешена

**Workaround**: Используйте Split Tunneling для исключения локальной сети:

```yaml
# config.yaml
split_tunneling:
  enabled: true
  mode: "exclude"
  exclude_cidrs:
    - "192.168.0.0/16"
    - "10.0.0.0/8"
    - "172.16.0.0/12"
```

**Планируется**: В будущих версиях будет добавлена опция `allow_local_network` в конфигурации Kill Switch.

---

### 5. Конфликт с другими firewall правилами

**Проблема**: Kill Switch может конфликтовать с существующими iptables правилами (ufw, firewalld, custom rules).

**Симптомы**:

- Неожиданное поведение трафика
- Правила Kill Switch не работают как ожидается
- Конфликты с другими VPN/firewall решениями

**Проверка конфликтов**:

```bash
# Проверить UFW
sudo ufw status
# Если active - может конфликтовать

# Проверить firewalld
sudo firewall-cmd --state
# Если running - может конфликтовать

# Проверить существующие OUTPUT правила
sudo iptables -L OUTPUT -n -v
# Ищите DROP/REJECT правила
```

**Решение**:

1. **Отключить UFW** (если используется):

```bash
sudo ufw disable
```

2. **Отключить firewalld** (если используется):

```bash
sudo systemctl stop firewalld
sudo systemctl disable firewalld
```

3. **Использовать только GoXRay Kill Switch** для управления firewall.

---

### 6. Kill Switch не деактивируется при успешном подключении

**Проблема**: Если функция `deactivateKillSwitch()` завершается с ошибкой, Kill Switch остается активным даже после успешного подключения VPN.

**Симптомы**:

- VPN подключен, но весь трафик (кроме исключений) заблокирован
- Нет доступа к интернету через VPN
- Логи показывают "VPN connected" но трафик не идет

**Диагностика**:

```bash
# Проверить статус Kill Switch
sudo iptables -L goxray_killswitch -n -v

# Проверить логи
sudo journalctl -u goxray -n 100 | grep -i "kill"

# Ожидаемый вывод при успешном подключении:
# "Kill switch deactivated - output traffic restored"
```

**Решение**:

```bash
# Ручная деактивация
sudo iptables -D OUTPUT -j goxray_killswitch
sudo iptables -F goxray_killswitch
sudo iptables -X goxray_killswitch

# Перезапуск GoXRay
sudo systemctl restart goxray
```

---

## 🔍 Диагностика

### Проверка состояния Kill Switch

```bash
# Check if kill switch is active
sudo iptables -L goxray_killswitch -n -v

# Вывод если активен:
# Chain goxray_killswitch (1 references)
# target     prot opt source               destination
# ACCEPT     all  --  0.0.0.0/0            0.0.0.0/0            /* loopback */
# ACCEPT     all  --  0.0.0.0/0            <XRay_IP>            /* xray server */
# DROP       all  --  0.0.0.0/0            0.0.0.0/0            /* block all */

# Вывод если не активен:
# iptables: No chain/target/match by that name.
```

### Проверка DNS во время Kill Switch

```bash
# Должно работать (DNS разрешен в v1.7.0+)
nslookup google.com

# Должно timeout (публичный трафик заблокирован)
curl -m 5 https://example.com
```

### Проверка логов

```bash
# Логи Kill Switch
sudo journalctl -u goxray -f | grep -i "kill"

# Ожидаемые сообщения:
# "Activating kill switch - blocking all traffic"
# "Kill switch activated (protection: IPv4 only)" или "IPv4 and IPv6"
# "Kill switch DNS exceptions configured"
# "Kill switch deactivated - output traffic restored"
```

---

## 📋 Checklist для troubleshooting

При проблемах с Kill Switch проверьте:

- [ ] Версия GoXRay >= v1.7.0 (для DNS fix)
- [ ] `enable_kill_switch: true` в config.yaml
- [ ] Нет конфликтующих firewall (ufw, firewalld)
- [ ] IPv6 настроен корректно (`enable_ipv6` соответствует системе)
- [ ] Логи не показывают ошибки iptables
- [ ] Kill Switch деактивируется при успешном подключении
- [ ] DNS работает во время Kill Switch (v1.7.0+)

---

## 🆘 Получение помощи

Если проблема не решена:

1. **Соберите диагностическую информацию**:

```bash
# Версия GoXRay
./goxray --version

# Конфигурация
cat /etc/goxray/config.yaml

# Состояние iptables
sudo iptables -L -n -v > iptables.txt
sudo ip6tables -L -n -v > ip6tables.txt

# Логи
sudo journalctl -u goxray -n 500 > goxray.log
```

2. **Создайте issue** на GitHub с:
   - Описанием проблемы
   - Версией GoXRay
   - Конфигурацией (без чувствительных данных)
   - Логами
   - Диагностической информацией

3. **Временный workaround**: Отключите Kill Switch:

```yaml
# config.yaml
connection:
  enable_kill_switch: false
```

---

**Версия документа**: v1.7.0  
**Последнее обновление**: 2026-06-03
