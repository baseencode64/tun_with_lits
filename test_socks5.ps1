# PowerShell script to test SOCKS5 proxy functionality
# This script tests if SOCKS5 proxy routes traffic through VPN

Write-Host "=== GoXRay SOCKS5 Proxy Test ===" -ForegroundColor Cyan
Write-Host ""

# Configuration
$SOCKS5_HOST = "192.168.88.252"
$SOCKS5_PORT = 1080
$TEST_URL = "https://api.ipify.org"

Write-Host "Testing SOCKS5 proxy at ${SOCKS5_HOST}:${SOCKS5_PORT}" -ForegroundColor Yellow
Write-Host ""

# Test 1: Check if SOCKS5 port is listening
Write-Host "[1/4] Checking if SOCKS5 port is listening..." -ForegroundColor Green
try {
    $tcpClient = New-Object System.Net.Sockets.TcpClient
    $tcpClient.Connect($SOCKS5_HOST, $SOCKS5_PORT)
    $tcpClient.Close()
    Write-Host "✓ SOCKS5 port ${SOCKS5_PORT} is open and listening" -ForegroundColor Green
} catch {
    Write-Host "✗ SOCKS5 port ${SOCKS5_PORT} is NOT listening" -ForegroundColor Red
    Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Make sure GoXRay is running with SOCKS5 enabled:" -ForegroundColor Yellow
    Write-Host "  socks5:" -ForegroundColor Gray
    Write-Host "    enabled: true" -ForegroundColor Gray
    Write-Host "    listen_addr: '0.0.0.0:1080'" -ForegroundColor Gray
    exit 1
}
Write-Host ""

# Test 2: Get real IP (without proxy)
Write-Host "[2/4] Getting your real IP address (without proxy)..." -ForegroundColor Green
try {
    $realIP = (Invoke-WebRequest -Uri $TEST_URL -UseBasicParsing).Content.Trim()
    Write-Host "✓ Your real IP: $realIP" -ForegroundColor Green
} catch {
    Write-Host "✗ Failed to get real IP" -ForegroundColor Red
    Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Red
    $realIP = "UNKNOWN"
}
Write-Host ""

# Test 3: Test SOCKS5 connection using curl (if available)
Write-Host "[3/4] Testing SOCKS5 proxy with curl..." -ForegroundColor Green

# Check if curl.exe exists (not PowerShell alias)
$curlPath = Get-Command curl.exe -ErrorAction SilentlyContinue

if ($curlPath) {
    Write-Host "Using curl.exe from: $($curlPath.Source)" -ForegroundColor Gray
    
    try {
        # Use curl.exe with SOCKS5 proxy
        $proxyIP = & curl.exe --socks5 "${SOCKS5_HOST}:${SOCKS5_PORT}" $TEST_URL --max-time 10 --silent
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ SOCKS5 proxy connection successful" -ForegroundColor Green
            Write-Host "  IP through SOCKS5: $proxyIP" -ForegroundColor Cyan
            
            # Compare IPs
            if ($proxyIP -eq $realIP) {
                Write-Host ""
                Write-Host "⚠ WARNING: IP addresses match!" -ForegroundColor Yellow
                Write-Host "  Real IP:        $realIP" -ForegroundColor Yellow
                Write-Host "  SOCKS5 IP:      $proxyIP" -ForegroundColor Yellow
                Write-Host ""
                Write-Host "This means traffic is NOT going through VPN!" -ForegroundColor Red
                Write-Host "Possible issues:" -ForegroundColor Yellow
                Write-Host "  1. VPN is not connected" -ForegroundColor Gray
                Write-Host "  2. SOCKS5 is not routing through VPN tunnel" -ForegroundColor Gray
                Write-Host "  3. Routing rules are not configured correctly" -ForegroundColor Gray
            } else {
                Write-Host ""
                Write-Host "✓ SUCCESS: Traffic is going through VPN!" -ForegroundColor Green
                Write-Host "  Real IP:        $realIP" -ForegroundColor Cyan
                Write-Host "  VPN IP (SOCKS5): $proxyIP" -ForegroundColor Cyan
            }
        } else {
            Write-Host "✗ SOCKS5 proxy connection failed" -ForegroundColor Red
            Write-Host "  curl exit code: $LASTEXITCODE" -ForegroundColor Red
        }
    } catch {
        Write-Host "✗ Error testing SOCKS5 with curl" -ForegroundColor Red
        Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Red
    }
} else {
    Write-Host "⚠ curl.exe not found, skipping curl test" -ForegroundColor Yellow
    Write-Host "  Install curl from: https://curl.se/windows/" -ForegroundColor Gray
}
Write-Host ""

# Test 4: Manual SOCKS5 handshake test
Write-Host "[4/4] Testing SOCKS5 handshake..." -ForegroundColor Green
try {
    $tcpClient = New-Object System.Net.Sockets.TcpClient
    $tcpClient.Connect($SOCKS5_HOST, $SOCKS5_PORT)
    $stream = $tcpClient.GetStream()
    
    # Send SOCKS5 greeting: Version 5, 1 auth method (no auth)
    $greeting = [byte[]]@(0x05, 0x01, 0x00)
    $stream.Write($greeting, 0, $greeting.Length)
    
    # Read response
    $response = New-Object byte[] 2
    $bytesRead = $stream.Read($response, 0, 2)
    
    if ($bytesRead -eq 2 -and $response[0] -eq 0x05 -and $response[1] -eq 0x00) {
        Write-Host "✓ SOCKS5 handshake successful" -ForegroundColor Green
        Write-Host "  Version: $($response[0])" -ForegroundColor Gray
        Write-Host "  Auth method: $($response[1]) (no auth)" -ForegroundColor Gray
    } else {
        Write-Host "✗ Invalid SOCKS5 response" -ForegroundColor Red
        Write-Host "  Expected: 0x05 0x00" -ForegroundColor Gray
        Write-Host "  Received: 0x$($response[0].ToString('X2')) 0x$($response[1].ToString('X2'))" -ForegroundColor Gray
    }
    
    $stream.Close()
    $tcpClient.Close()
} catch {
    Write-Host "✗ SOCKS5 handshake failed" -ForegroundColor Red
    Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# Summary
Write-Host "=== Test Summary ===" -ForegroundColor Cyan
Write-Host "SOCKS5 Proxy: ${SOCKS5_HOST}:${SOCKS5_PORT}" -ForegroundColor Gray
Write-Host "Test URL: $TEST_URL" -ForegroundColor Gray
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. If SOCKS5 is working but not routing through VPN:" -ForegroundColor Gray
Write-Host "   - Check GoXRay logs for routing information" -ForegroundColor Gray
Write-Host "   - Verify VPN connection is active (check TUN device)" -ForegroundColor Gray
Write-Host "   - Check routing rules: ip route show" -ForegroundColor Gray
Write-Host ""
Write-Host "2. To use SOCKS5 in applications:" -ForegroundColor Gray
Write-Host "   - Browser: Set SOCKS5 proxy to 127.0.0.1:1080" -ForegroundColor Gray
Write-Host "   - curl: curl --socks5 127.0.0.1:1080 https://example.com" -ForegroundColor Gray
Write-Host "   - wget: wget -e use_proxy=yes -e socks_proxy=127.0.0.1:1080 https://example.com" -ForegroundColor Gray
Write-Host ""
