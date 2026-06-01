# Configuration Metrics - Feature Status in Prometheus

**Version**: v1.6.4  
**Status**: ✅ Implemented  
**Date**: 2026-06-01

---

## 📊 Overview

All enabled/disabled features from configuration are now exposed as Prometheus metrics. This allows you to:

- ✅ Monitor which features are enabled via metrics dashboard
- ✅ Alert on unexpected configuration changes
- ✅ Verify actual configuration at runtime
- ✅ Debug configuration issues
- ✅ Track feature adoption patterns

---

## 🔢 New Metrics

### Configuration Metrics (Gauge - 0 or 1)

| Metric Name                            | Help Text                  | Value                        |
| -------------------------------------- | -------------------------- | ---------------------------- |
| `goxray_config_ipv6_enabled`           | IPv6 support enabled       | 1 = enabled, 0 = disabled    |
| `goxray_config_dns_protection_enabled` | DNS protection enabled     | 1 = enabled, 0 = disabled    |
| `goxray_config_kill_switch_enabled`    | Kill switch enabled        | 1 = enabled, 0 = disabled    |
| `goxray_config_metrics_enabled`        | Metrics endpoint enabled   | 1 = enabled, 0 = disabled    |
| `goxray_config_tls_insecure_allowed`   | TLS insecure certs allowed | 1 = allowed, 0 = not allowed |
| `goxray_config_e2e_check_enabled`      | E2E connectivity check     | 1 = enabled, 0 = disabled    |

---

## 📈 Example Prometheus Output

```bash
$ curl -s http://localhost:9090/metrics | grep goxray_config

# HELP goxray_config_ipv6_enabled IPv6 support enabled in configuration (1 = enabled, 0 = disabled)
# TYPE goxray_config_ipv6_enabled gauge
goxray_config_ipv6_enabled 1

# HELP goxray_config_dns_protection_enabled DNS protection enabled in configuration (1 = enabled, 0 = disabled)
# TYPE goxray_config_dns_protection_enabled gauge
goxray_config_dns_protection_enabled 1

# HELP goxray_config_kill_switch_enabled Kill switch functionality enabled in configuration (1 = enabled, 0 = disabled)
# TYPE goxray_config_kill_switch_enabled gauge
goxray_config_kill_switch_enabled 1

# HELP goxray_config_metrics_enabled Prometheus metrics enabled in configuration (1 = enabled, 0 = disabled)
# TYPE goxray_config_metrics_enabled gauge
goxray_config_metrics_enabled 1

# HELP goxray_config_tls_insecure_allowed TLS insecure certificates allowed (1 = allowed, 0 = not allowed)
# TYPE goxray_config_tls_insecure_allowed gauge
goxray_config_tls_insecure_allowed 0

# HELP goxray_config_e2e_check_enabled End-to-end connectivity check enabled (1 = enabled, 0 = disabled)
# TYPE goxray_config_e2e_check_enabled gauge
goxray_config_e2e_check_enabled 1
```

---

## 🔧 How It Works

### Automatic Update on Connect

When client connects (calls `Connect()`), all configuration metrics are automatically updated:

```go
func (c *Client) Connect(link string) error {
    // ... connection setup ...

    // Update configuration metrics
    c.updateConfigMetrics()  // ← Called automatically

    // ... rest of connection ...
}
```

### Configuration to Metric Mapping

```
YAML Config                          → Prometheus Metric
─────────────────────────────────────────────────────────
enable_ipv6: true                    → goxray_config_ipv6_enabled = 1
enable_ipv6: false                   → goxray_config_ipv6_enabled = 0

enable_dns_protection: true          → goxray_config_dns_protection_enabled = 1
enable_dns_protection: false         → goxray_config_dns_protection_enabled = 0

enable_kill_switch: true             → goxray_config_kill_switch_enabled = 1
enable_kill_switch: false            → goxray_config_kill_switch_enabled = 0

metrics_port: 9090 (>0)              → goxray_config_metrics_enabled = 1
metrics_port: 0                      → goxray_config_metrics_enabled = 0

tls_allow_insecure: true             → goxray_config_tls_insecure_allowed = 1
tls_allow_insecure: false            → goxray_config_tls_insecure_allowed = 0

e2e_check_url: "http://example.com"  → goxray_config_e2e_check_enabled = 1
e2e_check_url: "" (empty)            → goxray_config_e2e_check_enabled = 0
```

---

## 📊 Monitoring & Alerting Examples

### Prometheus Query Examples

**Check which features are enabled:**

```promql
goxray_config_ipv6_enabled{job="goxray"}
goxray_config_kill_switch_enabled{job="goxray"}
goxray_config_dns_protection_enabled{job="goxray"}
```

**Count all enabled features:**

```promql
goxray_config_ipv6_enabled
+ goxray_config_dns_protection_enabled
+ goxray_config_kill_switch_enabled
+ goxray_config_metrics_enabled
+ goxray_config_e2e_check_enabled
```

**Alert if kill switch is disabled:**

```yaml
alert: KillSwitchDisabled
expr: goxray_config_kill_switch_enabled == 0
for: 5m
annotations:
  summary: "Kill switch is disabled on {{ $labels.instance }}"
  description: "Kill switch should be enabled for security"
```

**Alert if metrics are disabled:**

```yaml
alert: MetricsDisabled
expr: goxray_config_metrics_enabled == 0
for: 5m
annotations:
  summary: "Metrics are disabled on {{ $labels.instance }}"
  description: "Cannot monitor goxray without metrics"
```

---

## 🎯 Use Cases

### 1. **Security Compliance**

Verify that all security features (Kill Switch, DNS Protection) are enabled across fleet:

```promql
# Find instances without kill switch
goxray_config_kill_switch_enabled == 0

# Find instances with insecure TLS allowed
goxray_config_tls_insecure_allowed == 1
```

### 2. **Configuration Auditing**

Track when configurations change:

```promql
# Rate of config changes
rate(goxray_config_kill_switch_enabled[5m]) != 0
```

### 3. **Deployment Verification**

After deploying new config, verify features are enabled:

```bash
$ curl http://goxray-instance:9090/metrics | \
  grep goxray_config | \
  grep " 0$"  # Find any disabled features
```

### 4. **Dashboard Integration**

Create dashboard panels showing:

- Feature enablement status (1 = green, 0 = red)
- Configuration coverage
- Security posture

### 5. **Troubleshooting**

When issues occur, check what's enabled:

```bash
# Get all config metrics
curl http://goxray:9090/metrics | grep goxray_config_

# Expected output shows all features enabled/disabled
```

---

## 📝 Example Configuration & Output

### Config File

```yaml
connection:
  from_raw_urls:
    - "https://example.com/servers.txt"
  enable_ipv6: true
  enable_dns_protection: true
  enable_kill_switch: true
  tls_allow_insecure: false

metrics_port: 9090

health_monitoring:
  e2e_check_url: "http://ipinfo.io/ip"
```

### Resulting Metrics

```
goxray_config_ipv6_enabled 1
goxray_config_dns_protection_enabled 1
goxray_config_kill_switch_enabled 1
goxray_config_metrics_enabled 1
goxray_config_tls_insecure_allowed 0
goxray_config_e2e_check_enabled 1
```

**Interpretation**: All security features enabled ✅

---

## 🔍 Logging

Configuration metrics are also logged for debugging:

```
DEBUG Configuration metrics updated ipv6_enabled=true dns_protection_enabled=true kill_switch_enabled=true metrics_enabled=true tls_insecure_allowed=false e2e_check_enabled=true
```

---

## 🚀 Integration

### Grafana Dashboard

Create panels:

```json
{
  "title": "Kill Switch Status",
  "targets": [
    {
      "expr": "goxray_config_kill_switch_enabled",
      "legendFormat": "Kill Switch Enabled"
    }
  ],
  "thresholds": [
    { "value": 0.5, "color": "red" },
    { "value": 0.9, "color": "green" }
  ]
}
```

### Alertmanager

Route alerts based on config state:

```yaml
- alert: InsecureConfigDetected
  expr: |
    (goxray_config_tls_insecure_allowed == 1) or
    (goxray_config_kill_switch_enabled == 0) or
    (goxray_config_dns_protection_enabled == 0)
  for: 5m
  annotations:
    summary: "Insecure configuration detected"
    severity: "warning"
```

---

## 📋 Metric Details

### Implementation Details

**Type**: Gauge (0 or 1)  
**Scope**: Set on each Connect()  
**Persistence**: Lost on disconnect (re-set on next connect)  
**Update Frequency**: Once per connect operation

### Values

- `1` = Feature Enabled / Configured
- `0` = Feature Disabled / Not Configured

### Thread Safety

- ✅ Safe to read at any time
- ✅ Updated under mutex protection during Connect()
- ✅ Safe for concurrent scraping

---

## 🔄 Version History

### v1.6.4 (Current)

- ✅ Added 6 configuration metrics
- ✅ Auto-update on Connect()
- ✅ Debug logging
- ✅ Full monitoring support

---

## 📞 Example Usage

### Shell Script to Check Configuration

```bash
#!/bin/bash
# Check goxray configuration status via metrics

METRICS_URL="http://localhost:9090/metrics"

echo "=== GoXray Configuration Status ==="
echo ""

check_metric() {
    local name=$1
    local label=$2
    local value=$(curl -s "$METRICS_URL" | grep "^$name " | awk '{print $2}')

    if [ "$value" == "1" ]; then
        echo "✅ $label: ENABLED"
    else
        echo "❌ $label: DISABLED"
    fi
}

check_metric "goxray_config_ipv6_enabled" "IPv6"
check_metric "goxray_config_dns_protection_enabled" "DNS Protection"
check_metric "goxray_config_kill_switch_enabled" "Kill Switch"
check_metric "goxray_config_metrics_enabled" "Metrics"
check_metric "goxray_config_e2e_check_enabled" "E2E Check"

echo ""
echo "Insecure TLS Allowed:"
value=$(curl -s "$METRICS_URL" | grep "^goxray_config_tls_insecure_allowed " | awk '{print $2}')
if [ "$value" == "1" ]; then
    echo "⚠️  YES (Not Recommended)"
else
    echo "✅ NO (Secure)"
fi
```

Output:

```
=== GoXray Configuration Status ===

✅ IPv6: ENABLED
✅ DNS Protection: ENABLED
✅ Kill Switch: ENABLED
✅ Metrics: ENABLED
✅ E2E Check: ENABLED

Insecure TLS Allowed:
✅ NO (Secure)
```

---

## ✅ Summary

Configuration metrics provide visibility into runtime feature status, enabling:

- ✅ Real-time monitoring of enabled features
- ✅ Automated compliance checking
- ✅ Configuration verification
- ✅ Troubleshooting and debugging
- ✅ Dashboard integration
- ✅ Alerting on security posture changes

All metrics are **automatically exposed** via the `/metrics` endpoint and require **no additional configuration**!
