# Kill Switch Functionality Implementation Plan

**Status**: Planning  
**Version**: v1.6.3  
**Date**: 2026-05-29

---

## 📖 Overview

Kill Switch - критический функционал безопасности, который:

- ✅ Блокирует весь интернет трафик при разрыве VPN соединения
- ✅ Предотвращает утечку реального IP адреса пользователя
- ✅ Может быть включен/отключен через конфигурацию

---

## 🏗️ Architecture

### Current Flow

```
Connect()
  ├─ Setup XRay
  ├─ Setup TUN
  ├─ Setup Routes  → Route all traffic through TUN
  └─ Setup Health Checks

Disconnect()
  ├─ Stop Health Checks
  ├─ Remove Routes
  ├─ Cleanup TUN
  ├─ Stop XRay
  └─ (No traffic blocking currently)
```

### New Flow with Kill Switch

```
Connect()
  ├─ Setup XRay
  ├─ Setup TUN
  ├─ Setup Routes
  ├─ Setup Health Checks
  └─ deactivateKillSwitch()  ← NEW: Allow traffic

Disconnect() / Failover
  ├─ Stop Health Checks
  ├─ activateKillSwitch()    ← NEW: Block traffic immediately
  ├─ Remove Routes
  ├─ Cleanup TUN
  ├─ Stop XRay
  └─ Keep killswitch until next successful connect
```

---

## 🔐 Implementation Details

### 1. Configuration (config.go)

Add to `ConnectionConfig`:

```go
type ConnectionConfig struct {
    // ... existing fields ...

    // Kill switch: block all traffic if VPN disconnects
    EnableKillSwitch bool `yaml:"enable_kill_switch"`
}
```

YAML example:

```yaml
connection:
  enable_kill_switch: true
```

### 2. Kill Switch State (client.go)

Add to `Client` struct:

```go
type Client struct {
    // ... existing fields ...

    killSwitchActive bool          // Track current state
    killSwitchMutex  sync.RWMutex  // Thread-safe access
}
```

### 3. Core Methods

#### activateKillSwitch()

```
When called:
1. If not enabled in config → return (no-op)
2. If already active → return (idempotent)
3. Log: "Activating kill switch - blocking all traffic"
4. Execute: setupKillSwitchRules()
5. Set: killSwitchActive = true
6. Monitor: Keep traffic blocked until reconnect
```

Rules to apply:

```bash
# Allow loopback (local communication)
iptables -A OUTPUT -o lo -j ACCEPT
ip6tables -A OUTPUT -o lo -j ACCEPT

# Block everything else
iptables -A OUTPUT -j DROP
ip6tables -A OUTPUT -j DROP

# Save rules
iptables-save > /tmp/goxray-killswitch-ipv4.rules
ip6tables-save > /tmp/goxray-killswitch-ipv6.rules
```

#### deactivateKillSwitch()

```
When called (on successful connect):
1. If not active → return (idempotent)
2. Log: "Deactivating kill switch - restoring traffic"
3. Execute: cleanupKillSwitchRules()
4. Set: killSwitchActive = false
5. Restore: Flush rules, restore normal routing
```

Cleanup:

```bash
# Remove all rules
iptables -F OUTPUT
iptables -X goxray_killswitch
ip6tables -F OUTPUT
ip6tables -X goxray_killswitch

# Restore routes
# (normal routes will be added by setupTunnel)
```

---

## 🔄 Integration Points

### 1. Connect() Method

**Location**: After successful metrics startup

```go
// At end of Connect() method, before returning nil:
if c.cfg.Connection.EnableKillSwitch {
    if err := c.deactivateKillSwitch(); err != nil {
        c.cfg.Logger.Error("Failed to deactivate kill switch", "error", err)
        // Log but don't fail connection - traffic should work anyway
    }
}
```

### 2. Disconnect() Method

**Location**: At beginning, before route cleanup

```go
// At start of Disconnect() method:
if c.cfg.Connection.EnableKillSwitch {
    if err := c.activateKillSwitch(); err != nil {
        c.cfg.Logger.Error("Failed to activate kill switch", "error", err)
        // Log but continue cleanup - partial protection is better than nothing
    }
}
```

### 3. Failover Trigger

**Location**: In health check failure handler

```go
// When failover is triggered (max retries exceeded):
if c.cfg.Connection.EnableKillSwitch {
    if err := c.activateKillSwitch(); err != nil {
        c.cfg.Logger.Warn("Kill switch activation failed during failover", "error", err)
    }
}
```

---

## 🛡️ Implementation Strategy

### Phase 1: Core Implementation

- [ ] Add EnableKillSwitch to config.go
- [ ] Add killSwitchActive field to Client struct
- [ ] Implement activateKillSwitch() method
- [ ] Implement deactivateKillSwitch() method
- [ ] Implement setupKillSwitchRules() method
- [ ] Implement cleanupKillSwitchRules() method

### Phase 2: Integration

- [ ] Call deactivateKillSwitch() in Connect()
- [ ] Call activateKillSwitch() in Disconnect()
- [ ] Add proper logging and error handling
- [ ] Test basic functionality

### Phase 3: Failover Integration

- [ ] Call activateKillSwitch() on health check failure
- [ ] Ensure clean transitions during failover
- [ ] Test multiple failovers with kill switch

### Phase 4: Testing & Polish

- [ ] Unit tests for each method
- [ ] Integration tests with failover scenarios
- [ ] Documentation updates
- [ ] Release notes

---

## 🧪 Testing Strategy

### Unit Tests

1. **activateKillSwitch()** - Apply rules correctly
2. **deactivateKillSwitch()** - Clean up rules correctly
3. **Idempotency** - Can call multiple times safely
4. **Config disabled** - No-op when disabled

### Integration Tests

1. **Normal connect/disconnect** - Rules applied and removed
2. **Failover scenario** - Kill switch activated during failover
3. **Multiple failovers** - State correctly managed
4. **Traffic blocking** - Actual traffic is blocked when active

### Manual Tests

```bash
# Enable kill switch in config
enable_kill_switch: true

# Start client
sudo goxray --config /etc/goxray/config.yaml

# Check rules are applied (after successful connect)
sudo iptables -L OUTPUT -n
sudo ip6tables -L OUTPUT -n

# Simulate disconnect
pkill goxray  # or trigger failover

# Verify rules block traffic
ping -c 4 8.8.8.8  # Should timeout/fail

# Restart
sudo goxray --config /etc/goxray/config.yaml

# Verify rules cleared
sudo iptables -L OUTPUT -n
ping -c 4 8.8.8.8  # Should work
```

---

## ⚙️ Technical Details

### iptables Rule Strategy

```
Use custom chain for clean management:

1. Create custom chain: goxray_killswitch
2. Add specific rules to it
3. Jump from OUTPUT to goxray_killswitch
4. Easier to remove/reset without affecting system

Commands:
iptables -N goxray_killswitch          # Create chain
iptables -A goxray_killswitch -o lo -j ACCEPT
iptables -A goxray_killswitch -j DROP
iptables -A OUTPUT -j goxray_killswitch  # Jump from OUTPUT
```

### Rule Persistence

```
Option 1: In-memory only (current approach)
- Rules lost on reboot
- Faster
- Suitable for systemd service

Option 2: Save/Restore
- iptables-save / iptables-restore
- Survives reboots if needed
- More complex

Current: Use Option 1 (in-memory)
```

### Error Handling

```
If iptables fails:
- activateKillSwitch(): Log error but don't fail - partial protection
- deactivateKillSwitch(): Log error but restore what we can

Graceful degradation - if kill switch fails,
normal routing/DNS protection still works
```

---

## 📊 Behavior Matrix

| State              | Config   | Action        | Result                          |
| ------------------ | -------- | ------------- | ------------------------------- |
| Connected          | Disabled | Normal        | ✅ Traffic flows                |
| Connected          | Enabled  | Deactivate KS | ✅ Traffic flows                |
| Disconnected       | Disabled | Normal        | ✅ Traffic blocked by no routes |
| Disconnected       | Enabled  | Activate KS   | ✅ Traffic blocked by iptables  |
| Failover           | Enabled  | Activate KS   | ✅ Traffic blocked immediately  |
| Failover→Connected | Enabled  | Deactivate KS | ✅ Traffic restored             |

---

## 🔍 Monitoring & Logging

### Log Messages

```
INFO: Activating kill switch - blocking all traffic
INFO: Kill switch activated - output traffic blocked
INFO: Deactivating kill switch - restoring traffic
INFO: Kill switch deactivated - output traffic restored

WARN: Failed to activate kill switch: {error}
WARN: Failed to deactivate kill switch: {error}

DEBUG: Kill switch rules applied successfully
DEBUG: Kill switch rules cleaned up
```

### Metrics (Optional)

```
Kill switch specific metrics:
- kill_switch_activations_total
- kill_switch_deactivations_total
- kill_switch_active (gauge: 0/1)
```

---

## 🚀 Performance Impact

- **Memory**: Minimal (few iptables rules)
- **CPU**: Negligible (rules only processed once)
- **Latency**: None (rules already in kernel)
- **Overall**: Negligible impact

---

## 🔒 Security Considerations

### Threat Protection

1. **IP Leak Prevention** ✅
   - Blocks all outbound traffic immediately on disconnect
   - Prevents brief window where real IP could leak

2. **DNS Leak Prevention** ✅
   - Works with existing DNS protection
   - Double protection layer

3. **Fail-Secure** ✅
   - Blocks traffic on error (safe default)
   - User explicitly enables it

### Limitations

1. **Doesn't protect**:
   - Local network traffic (intentional, by design)
   - IPv6 (if not enabled/supported)
   - Traffic already in-flight (TUN buffer)

2. **Known Issues**:
   - If client crashes without cleanup, rules remain active
   - Solution: Manual cleanup or reboot

---

## 📚 Related Issue

This addresses TODO in README.md:

```
- [ ] Add kill switch functionality  ← This
```

---

## ✅ Definition of Done

- [ ] Code implemented and tested
- [ ] All integration points working
- [ ] Logging and error handling complete
- [ ] Documentation updated
- [ ] Release notes prepared
- [ ] Binary built: goxray_v1.6.3_linux_amd64
- [ ] Git commit and tag created

---

## 🔄 Future Enhancements

1. **Persist Rules**: Option to save/restore rules across reboots
2. **Whitelist IPs**: Allow local network/VPN server IPs
3. **Custom Rules**: User-specified exceptions
4. **Metrics Dashboard**: Visual kill switch status
5. **systemd Integration**: Auto-cleanup on crash

---

**Ready to implement!** 🚀
