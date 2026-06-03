// Proxy Auto-Configuration (PAC) file for GoXRay VPN
// This file automatically routes traffic through SOCKS5 proxy or direct connection
//
// Usage:
// 1. Save this file to a location accessible by your browser
// 2. In browser settings, set Automatic proxy configuration URL to:
//    file:///C:/path/to/proxy.pac
//    or
//    http://localhost:8000/proxy.pac (if served via HTTP)
//
// Version: 1.7.0
// Date: 2026-06-01

function FindProxyForURL(url, host) {
  // Convert host to lowercase for case-insensitive matching
  host = host.toLowerCase();

  // ============================================
  // 1. DIRECT CONNECTION (bypass VPN)
  // ============================================

  // Local hostnames (no dots)
  if (isPlainHostName(host)) {
    return "DIRECT";
  }

  // Localhost and loopback
  if (
    host == "localhost" ||
    host == "127.0.0.1" ||
    shExpMatch(host, "*.local") ||
    shExpMatch(host, "*.localhost")
  ) {
    return "DIRECT";
  }

  // Private IP ranges (RFC 1918)
  if (
    isInNet(host, "10.0.0.0", "255.0.0.0") || // Class A private
    isInNet(host, "172.16.0.0", "255.240.0.0") || // Class B private
    isInNet(host, "192.168.0.0", "255.255.0.0") // Class C private
  ) {
    return "DIRECT";
  }

  // Link-local addresses (APIPA)
  if (isInNet(host, "169.254.0.0", "255.255.0.0")) {
    return "DIRECT";
  }

  // Multicast addresses
  if (isInNet(host, "224.0.0.0", "240.0.0.0")) {
    return "DIRECT";
  }

  // IPv6 link-local (fe80::/10)
  if (shExpMatch(host, "fe80:*")) {
    return "DIRECT";
  }

  // IPv6 unique local (fc00::/7)
  if (shExpMatch(host, "fc*:*") || shExpMatch(host, "fd*:*")) {
    return "DIRECT";
  }

  // ============================================
  // 2. OPTIONAL: Specific domains/services DIRECT
  // ============================================

  // Uncomment to bypass VPN for specific services:

  // Microsoft services (Windows Update, Office, etc.)
  // if (
  //   shExpMatch(host, "*.microsoft.com") ||
  //   shExpMatch(host, "*.windows.com") ||
  //   shExpMatch(host, "*.windowsupdate.com") ||
  //   shExpMatch(host, "*.office.com")
  // ) {
  //   return "DIRECT";
  // }

  // CDN services (for better performance)
  // if (
  //   shExpMatch(host, "*.cloudflare.com") ||
  //   shExpMatch(host, "*.akamai.net") ||
  //   shExpMatch(host, "*.fastly.net")
  // ) {
  //   return "DIRECT";
  // }

  // Streaming services (Netflix, YouTube, etc.)
  // if (
  //   shExpMatch(host, "*.netflix.com") ||
  //   shExpMatch(host, "*.nflxvideo.net") ||
  //   shExpMatch(host, "*.youtube.com") ||
  //   shExpMatch(host, "*.googlevideo.com") ||
  //   shExpMatch(host, "*.twitch.tv") ||
  //   shExpMatch(host, "*.ttvnw.net")
  // ) {
  //   return "DIRECT";
  // }

  // Banking/financial services (for security)
  // if (
  //   shExpMatch(host, "*.bank.com") ||
  //   shExpMatch(host, "*.paypal.com")
  // ) {
  //   return "DIRECT";
  // }

  // ============================================
  // 3. VPN CONNECTION (through SOCKS5 proxy)
  // ============================================

  // All other traffic goes through SOCKS5 proxy
  // Format: "SOCKS5 host:port; DIRECT"
  // The "DIRECT" fallback ensures connectivity if proxy is down
  return "SOCKS5 localhost:1080; DIRECT";
}

// ============================================
// HELPER FUNCTIONS (built-in PAC functions)
// ============================================

// isPlainHostName(host)
//   Returns true if host has no dots (e.g., "intranet")

// dnsDomainIs(host, domain)
//   Returns true if host is in domain (e.g., dnsDomainIs("www.example.com", ".example.com"))

// localHostOrDomainIs(host, hostdom)
//   Returns true if host matches hostdom exactly

// isResolvable(host)
//   Returns true if host can be resolved via DNS

// isInNet(host, pattern, mask)
//   Returns true if host IP is in the specified network
//   Example: isInNet("192.168.1.100", "192.168.0.0", "255.255.0.0")

// dnsResolve(host)
//   Resolves hostname to IP address

// myIpAddress()
//   Returns the IP address of the machine running the browser

// dnsDomainLevels(host)
//   Returns the number of DNS domain levels (dots) in hostname

// shExpMatch(str, pattern)
//   Shell-style pattern matching (* and ? wildcards)
//   Example: shExpMatch("www.example.com", "*.example.com")

// weekdayRange(wd1, wd2, gmt)
//   Returns true if current day is in range
//   Example: weekdayRange("MON", "FRI") for weekdays only

// dateRange(...)
//   Returns true if current date is in range

// timeRange(...)
//   Returns true if current time is in range

// ============================================
// ADVANCED EXAMPLES
// ============================================

// Example 1: Time-based routing (VPN only during work hours)
// function FindProxyForURL(url, host) {
//   if (timeRange(9, 17)) {  // 9 AM to 5 PM
//     return "SOCKS5 localhost:1080; DIRECT";
//   }
//   return "DIRECT";
// }

// Example 2: Geo-based routing (VPN for specific countries)
// function FindProxyForURL(url, host) {
//   // Use VPN for blocked countries
//   if (
//     shExpMatch(host, "*.cn") ||  // China
//     shExpMatch(host, "*.ru") ||  // Russia
//     shExpMatch(host, "*.ir")     // Iran
//   ) {
//     return "SOCKS5 localhost:1080; DIRECT";
//   }
//   return "DIRECT";
// }

// Example 3: Protocol-based routing (HTTPS through VPN, HTTP direct)
// function FindProxyForURL(url, host) {
//   if (url.substring(0, 6) == "https:") {
//     return "SOCKS5 localhost:1080; DIRECT";
//   }
//   return "DIRECT";
// }

// Example 4: Load balancing (multiple SOCKS5 proxies)
// function FindProxyForURL(url, host) {
//   // Try multiple proxies in order
//   return "SOCKS5 localhost:1080; SOCKS5 localhost:1081; DIRECT";
// }
