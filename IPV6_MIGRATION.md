# 🚀 IPv6 Migration Guide

## Quick Start

### For End Users

**Before (IPv4 only):**
```bash
sudo goxray --from-raw https://example.com/links.txt
```

**After (with IPv6):**
```bash
sudo goxray --from-raw https://example.com/links.txt --ipv6
```

That's it! Just add the `--ipv6` flag.

---

### For Library Users

**Before:**
```go
vpn, _ := client.NewClientWithOpts(client.Config{
    TLSAllowInsecure: false,
    Logger:           logger,
})
```

**After:**
```go
vpn, _ := client.NewClientWithOpts(client.Config{
    TLSAllowInsecure: false,
    Logger:           logger,
    EnableIPv6:       true,  // ← Add this line
})
```

---

## What Changed?

### Code Changes Summary

1. **New Configuration Field**
   ```go
   type Config struct {
       // ... existing fields ...
       EnableIPv6 bool  // NEW
   }
   ```

2. **New Default Routes**
   ```go
   var DefaultRoutesToTUNIPv6 = []*route.Addr{
       route.MustParseAddr("::/1"),
       route.MustParseAddr("8000::/1"),
   }
   ```

3. **Enhanced TUN Setup**
   - Now configures IPv6 routes when `EnableIPv6` is true
   - Stores TUN interface name for cleanup

4. **Enhanced Disconnect**
   - Cleans up IPv6 routes on shutdown
   - Prevents route table pollution

5. **CLI Flag**
   - Added `--ipv6` boolean flag in main.go

---

## Backward Compatibility

✅ **100% Backward Compatible**

- IPv6 is **disabled by default**
- Existing code works without changes
- No breaking API changes
- Optional feature (opt-in)

---

## Testing Checklist

Before deploying to production:

- [ ] Test with `--ipv6` flag on staging environment
- [ ] Verify IPv6 connectivity: `curl -6 https://ipv6.google.com`
- [ ] Check for IPv6 leaks: visit https://test-ipv6.com
- [ ] Monitor logs for IPv6-related warnings
- [ ] Test disconnect/reconnect cycle
- [ ] Verify route cleanup after disconnect

---

## Rollback Plan

If you encounter issues:

1. **Disable IPv6 flag:**
   ```bash
   # Simply remove --ipv6 flag
   sudo goxray --from-raw https://example.com/links.txt
   ```

2. **Clear IPv6 routes manually (if needed):**
   ```bash
   sudo ip -6 route flush dev tun0
   ```

3. **Restart service:**
   ```bash
   sudo systemctl restart goxray
   ```

---

## Support

### Documentation
- Full guide: [`IPv6_SUPPORT.md`](IPv6_SUPPORT.md)
- Changelog: [`CHANGELOG_IPV6.md`](CHANGELOG_IPV6.md)

### Troubleshooting
- Check logs: `sudo journalctl -u goxray -f | grep -i ipv6`
- Test connectivity: `curl -6 https://ipv6.google.com`
- Verify routes: `ip -6 route show`

### Common Issues
See [`IPv6_SUPPORT.md`](IPv6_SUPPORT.md) section "Common Issues" for detailed troubleshooting.

---

## Next Steps

1. ✅ Review this migration guide
2. ✅ Test in development/staging environment
3. ✅ Update deployment scripts to include `--ipv6` flag if needed
4. ✅ Monitor production metrics after enabling
5. ✅ Report any issues via GitHub Issues

---

**Questions?** Open an issue at: https://github.com/goxray/tun/issues
