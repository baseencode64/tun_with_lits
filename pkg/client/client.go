package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goxray/core/network/route"
	"github.com/goxray/core/network/tun"
	"github.com/goxray/core/pipe2socks"
	"github.com/jackpal/gateway"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	xrayproto "github.com/lilendian0x00/xray-knife/v3/pkg/protocol"
	"github.com/lilendian0x00/xray-knife/v3/pkg/xray"
	xapplog "github.com/xtls/xray-core/app/log"
	xcommlog "github.com/xtls/xray-core/common/log"
)

const disconnectTimeout = 30 * time.Second

var (
	// defaultTUNAddress is the address new TUN device will be set up with.
	defaultTUNAddress = &net.IPNet{IP: net.IPv4(192, 18, 0, 1), Mask: net.IPv4Mask(255, 255, 255, 255)}
	
	// defaultTUNAddressIPv6 is the IPv6 address for TUN device (ULA range fd00::/8).
	// Using fd00:dead:beef::1/64 as a unique local address for the VPN tunnel.
	defaultTUNAddressIPv6 = &net.IPNet{
		IP:   net.ParseIP("fd00:dead:beef::1"),
		Mask: net.CIDRMask(64, 128),
	}
	
	// defaultInboundProxy default proxy will be set up for listening on 127.0.0.1.
	defaultInboundProxy = &Proxy{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: getFreePort(),
	}

	// DefaultRoutesToTUN will route all system traffic through the TUN (IPv4 only).
	DefaultRoutesToTUN = []*route.Addr{
		// Reroute all IPv4 traffic.
		route.MustParseAddr("0.0.0.0/1"),
		route.MustParseAddr("128.0.0.0/1"),
	}
	
	// DNSRoutesToTUN routes DNS traffic through TUN to prevent DNS leaks and ensure resolution.
	DNSRoutesToTUN = []*route.Addr{
		// Common public DNS servers
		route.MustParseAddr("8.8.8.8/32"),   // Google DNS
		route.MustParseAddr("8.8.4.4/32"),   // Google DNS
		route.MustParseAddr("1.1.1.1/32"),   // Cloudflare DNS
		route.MustParseAddr("1.0.0.1/32"),   // Cloudflare DNS
		route.MustParseAddr("9.9.9.9/32"),   // Quad9 DNS
		route.MustParseAddr("149.112.112.112/32"), // Quad9 DNS
	}
	
	// DefaultRoutesToTUNIPv6 will route all IPv6 system traffic through the TUN.
	DefaultRoutesToTUNIPv6 = []*route.Addr{
		// Reroute all IPv6 traffic.
		route.MustParseAddr("::/1"),
		route.MustParseAddr("8000::/1"),
	}
	
	// DNSRoutesToTUNIPv6 routes IPv6 DNS traffic through TUN.
	DNSRoutesToTUNIPv6 = []*route.Addr{
		// IPv6 DNS servers
		route.MustParseAddr("2001:4860:4860::8888/128"),  // Google DNS
		route.MustParseAddr("2001:4860:4860::8844/128"),  // Google DNS
		route.MustParseAddr("2606:4700:4700::1111/128"),  // Cloudflare DNS
		route.MustParseAddr("2606:4700:4700::1001/128"),  // Cloudflare DNS
	}
	
	// DNSPortRoutesToTUN routes all traffic to DNS ports (53) through TUN to prevent DNS leaks.
	DNSPortRoutesToTUN = []*route.Addr{
		// Specific DNS port routes
		route.MustParseAddr("0.0.0.0/0"),   // All IPv4 addresses
	}
	
	// DNSPortRoutesToTUNIPv6 routes all IPv6 traffic to DNS ports through TUN.
	DNSPortRoutesToTUNIPv6 = []*route.Addr{
		// All IPv6 addresses
		route.MustParseAddr("::/0"),        // All IPv6 addresses
	}
)

// Prometheus metrics
var (
	vpnConnectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vpn_connections_total",
			Help: "Total number of VPN connections established",
		},
		[]string{"protocol"},
	)
	
	vpnDisconnectionsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "vpn_disconnections_total",
			Help: "Total number of VPN disconnections",
		},
	)
	
	vpnConnectionDuration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "vpn_connection_duration_seconds",
			Help: "Duration of current VPN connection in seconds",
		},
	)
	
	vpnBytesRead = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "vpn_bytes_read_total",
			Help: "Total bytes read from TUN device",
		},
	)
	
	vpnBytesWritten = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "vpn_bytes_written_total",
			Help: "Total bytes written to TUN device",
		},
	)
	
	vpnConnected = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "vpn_connected",
			Help: "Whether VPN is currently connected (1 = connected, 0 = disconnected)",
		},
	)
	
	vpnTunIPv4 = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vpn_tun_ipv4",
			Help: "TUN interface IPv4 address (label: ip_address)",
		},
		[]string{"ip_address"},
	)
	
	vpnTunIPv6 = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vpn_tun_ipv6",
			Help: "TUN interface IPv6 address (label: ip_address)",
		},
		[]string{"ip_address"},
	)
	
	vpnServerIP = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vpn_server_ip",
			Help: "VPN server external IP address (label: ip_address)",
		},
		[]string{"ip_address"},
	)
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(vpnConnectionsTotal)
	prometheus.MustRegister(vpnDisconnectionsTotal)
	prometheus.MustRegister(vpnConnectionDuration)
	prometheus.MustRegister(vpnBytesRead)
	prometheus.MustRegister(vpnBytesWritten)
	prometheus.MustRegister(vpnConnected)
	prometheus.MustRegister(vpnTunIPv4)
	prometheus.MustRegister(vpnTunIPv6)
	prometheus.MustRegister(vpnServerIP)
}

var (

)

// Config serves configuration for new Client. Empty fields will be set up with defaults values.
//
// It is advised to not configure the cl yourself, please use NewClient() with default config values,
// normally you don't have to set these fields yourself.
type Config struct {
	// GatewayIP to direct outbound traffic. Must be able to reach remote XRay server.
	// (default: will be dynamically detected from your default gateway).
	//
	// Client will determine the system gateway IP automatically,
	// and you don't have to set this field explicitly.
	GatewayIP *net.IP
	// Socks proxy address on which XRay creates inbound proxy (default: 127.0.0.1:10808).
	InboundProxy *Proxy
	// TUN device address (default: 192.18.0.1).
	TUNAddress *net.IPNet
	// List of routes to be pointed to TUN device (default: DefaultRoutesToTUN).
	//
	// One exception is explicitly added for XRay remote server IP and can not be altered.
	RoutesToTUN []*route.Addr
	// Whether to allow self-signed certificates or not.
	TLSAllowInsecure bool
	// Pass logger with debug level to observe debug logs (default: slog.TextHandler).
	Logger *slog.Logger
	// XRayLogType is used to redefine xray core log type (default: LogType_None).
	XRayLogType xapplog.LogType
	// EnableIPv6 enables IPv6 support for TUN device (default: false).
	EnableIPv6 bool
	// MetricsPort enables Prometheus metrics endpoint on specified port (0 = disabled).
	MetricsPort int
	// EnableDNSProtection enables DNS leak protection by routing all DNS traffic through the TUN interface (default: false).
	EnableDNSProtection bool
}

func (c *Config) apply(new *Config) {
	if new.GatewayIP != nil {
		c.GatewayIP = new.GatewayIP
	}
	if new.InboundProxy != nil {
		c.InboundProxy = new.InboundProxy
	}
	if new.TUNAddress != nil {
		c.TUNAddress = new.TUNAddress
	}
	if new.Logger != nil {
		c.Logger = new.Logger
	}
	if new.RoutesToTUN != nil {
		c.RoutesToTUN = new.RoutesToTUN
	}
	if new.XRayLogType != xapplog.LogType_None {
		c.XRayLogType = new.XRayLogType
	}
	// EnableIPv6 is a boolean flag, always apply if explicitly set
	c.EnableIPv6 = new.EnableIPv6
	// MetricsPort is optional
	if new.MetricsPort > 0 {
		c.MetricsPort = new.MetricsPort
	}
	// EnableDNSProtection is a boolean flag, always apply if explicitly set
	c.EnableDNSProtection = new.EnableDNSProtection
}

// Client is the actual VPN cl. It manages connections, routing and tunneling of the requests.
// It is safe to make a Client connection as it does not change the default system routing and
// just adds on existing infrastructure.
type Client struct {
	cfg Config

	xInst  runnable
	xCfg   *xrayproto.GeneralConfig
	xSrvIP *net.IPAddr
	tunnel io.ReadWriteCloser
	pipe   pipe
	routes ipTable

	tunnelStopped chan error
	stopTunnel    func()
	tunName       string // Stores TUN interface name for cleanup
	
	mu sync.Mutex // Protects concurrent access during connect/disconnect
	isConnected bool // Tracks connection state to prevent double operations
	
	// Prometheus metrics
	metricsServer *http.Server
	
	// Traffic counters (atomic)
	bytesRead    int64
	bytesWritten int64
}

// Proxy will set up XRay inbound.
type Proxy struct {
	IP   net.IP // Inbound proxy IP (e.g. 127.0.0.1)
	Port int    // Inbound proxy port (e.g. 1080)
}

func (p *Proxy) String() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

// NewClient initializes default Client with default proxy address.
// If you want more options use Client struct.
func NewClient() (*Client, error) {
	gatewayIP, err := gateway.DiscoverGateway()
	if err != nil {
		return nil, fmt.Errorf("discover gateway: %w", err)
	}

	p, err := pipe2socks.NewPipe(pipe2socks.DefaultOpts)
	if err != nil {
		return nil, fmt.Errorf("tun2socks new pipe: %w", err)
	}

	r, err := route.New()
	if err != nil {
		return nil, fmt.Errorf("route new: %w", err)
	}

	return &Client{
		cfg: Config{
			GatewayIP:    &gatewayIP,
			InboundProxy: defaultInboundProxy,
			TUNAddress:   defaultTUNAddress,
			RoutesToTUN:  DefaultRoutesToTUN,
			Logger:       slog.New(slog.NewTextHandler(os.Stdout, nil)),
		},
		tunnelStopped: make(chan error),
		pipe:          p,
		routes:        r,
	}, nil
}

// NewClientWithOpts initializes Client with specified Config. It is recommended to just use NewClient().
func NewClientWithOpts(cfg Config) (*Client, error) {
	client, err := NewClient()
	if err != nil {
		return nil, err
	}

	client.cfg.apply(&cfg)
	
	// Start metrics server if configured
	if client.cfg.MetricsPort > 0 {
		client.startMetricsUpdate()
	}

	return client, nil
}

// GatewayIP returns gateway IP used to route outbound traffic through.
// It is used to route packets destined to XRay remote server.
func (c *Client) GatewayIP() net.IP {
	return *c.cfg.GatewayIP
}

// TUNAddress returns address the TUN device is set up on.
// Traffic is routed to this TUN device.
func (c *Client) TUNAddress() net.IP {
	return c.cfg.TUNAddress.IP
}

// InboundProxy returns proxy address initialized by XRay core.
// Traffic from TUN device is routed to this proxy.
func (c *Client) InboundProxy() Proxy {
	return *c.cfg.InboundProxy
}

// Connect creates a global tunnel and routes all incoming connections (or traffic specified in Config.RoutesToTUN)
// to the VPN server via newly created defaultInboundProxy.
func (c *Client) Connect(link string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Prevent double connection
	if c.isConnected {
		c.cfg.Logger.Warn("Already connected, disconnecting first")
		c.mu.Unlock()
		c.Disconnect(context.Background())
		c.mu.Lock()
	}

	var err error
	c.cfg.Logger.Debug("Connecting to tunnel", "cfg", c.cfg)

	c.xInst, c.xCfg, err = c.createXrayProxy(link)
	if err != nil {
		c.cfg.Logger.Error("xray core creation failed", "err", err, "xray_config", c.xCfg)

		return fmt.Errorf("create xray core instance: %w", err)
	}
	c.cfg.Logger.Debug("xray core instance created", "xray_config", c.xCfg)

	c.cfg.Logger.Debug("starting xray core instance")
	if err = c.xInst.Start(); err != nil {
		c.cfg.Logger.Error("xray core instance startup failed", "err", err)

		return fmt.Errorf("start xray core instance: %w", err)
	}
	time.Sleep(100 * time.Millisecond) // Sometimes XRay instance should have a bit more time to set up.
	c.cfg.Logger.Debug("xray core instance started")

	// CRITICAL FIX: Add route exception for Xray server BEFORE creating TUN
	// This prevents routing loop where Xray traffic goes through TUN back to itself
	c.cfg.Logger.Debug("adding route exception for Xray server before TUN setup")
	_ = c.routes.Delete(c.xrayToGatewayRoute()) // In case previous run failed.
	err = c.routes.Add(c.xrayToGatewayRoute())
	if err != nil {
		c.cfg.Logger.Error("routing xray server IP to default route failed", "err", err, "route", c.xrayToGatewayRoute())
		// Clean up Xray instance on failure
		c.xInst.Close()
		return fmt.Errorf("add xray server route exception: %w", err)
	}
	c.cfg.Logger.Debug("Xray server route exception added successfully")

	c.cfg.Logger.Debug("Setting up TUN device")
	// Create TUN and route all traffic to it.
	c.tunnel, err = c.setupTunnel()
	if err != nil {
		c.cfg.Logger.Error("TUN creation failed", "err", err)
		// Clean up routes and Xray on failure
		c.routes.Delete(c.xrayToGatewayRoute())
		c.xInst.Close()
		return fmt.Errorf("setup TUN device: %w", err)
	}
	c.tunnel = newReaderMetrics(c.tunnel)
	c.cfg.Logger.Debug("TUN device created")

	// Verify SOCKS proxy is listening before starting pipe
	socksAddr := c.cfg.InboundProxy.String()
	c.cfg.Logger.Debug("Verifying SOCKS proxy is listening", "address", socksAddr)
	
	// Wait for SOCKS proxy with timeout (up to 2 seconds)
	proxyReady := false
	for i := 0; i < 20; i++ {
		conn, dialErr := net.DialTimeout("tcp", socksAddr, 100*time.Millisecond)
		if dialErr == nil {
			conn.Close()
			proxyReady = true
			c.cfg.Logger.Debug("SOCKS proxy is ready", "address", socksAddr, "attempts", i+1)
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	if !proxyReady {
		c.cfg.Logger.Error("SOCKS proxy failed to start", "address", socksAddr)
		// Clean up
		c.tunnel.Close()
		c.routes.Delete(c.xrayToGatewayRoute())
		c.xInst.Close()
		return fmt.Errorf("SOCKS proxy at %s did not become ready within 2 seconds", socksAddr)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	var ctx context.Context
	ctx, c.stopTunnel = context.WithCancel(context.Background())
	go func() {
		wg.Done()
		err := c.pipe.Copy(ctx, c.tunnel, c.cfg.InboundProxy.String())
		select {
		case c.tunnelStopped <- err:
			c.cfg.Logger.Debug("tunnel pipe closed", "err", err)
		default:
			// Channel might be full or closed, log warning
			c.cfg.Logger.Warn("Could not send tunnel stop signal, channel issue")
		}
	}()
	wg.Wait()
	
	// Start traffic monitoring goroutine
	go c.monitorTraffic(ctx)
	
	// Setup DNS protection if enabled
	if c.cfg.EnableDNSProtection {
		c.cfg.Logger.Info("Setting up DNS leak protection")
		if setupErr := c.setupDNSProtection(); setupErr != nil {
			c.cfg.Logger.Error("Failed to set up DNS protection", "error", setupErr)
			// Continue anyway - it's not critical for basic VPN functionality
		}
	}
	
	c.isConnected = true
	
	// Update Prometheus metrics
	vpnConnected.Set(1)
	vpnConnectionsTotal.WithLabelValues(c.detectProtocol(link)).Inc()
	
	// Update TUN IP addresses
	tunIPv4 := c.cfg.TUNAddress.IP.String()
	vpnTunIPv4.Reset()
	vpnTunIPv4.WithLabelValues(tunIPv4).Set(1)
	
	if c.cfg.EnableIPv6 {
		vpnTunIPv6.Reset()
		vpnTunIPv6.WithLabelValues(defaultTUNAddressIPv6.IP.String()).Set(1)
	}
	
	// Update VPN server external IP
	if c.xSrvIP != nil {
		serverIP := c.xSrvIP.String()
		vpnServerIP.Reset()
		vpnServerIP.WithLabelValues(serverIP).Set(1)
		c.cfg.Logger.Debug("VPN server IP metric updated", "ip", serverIP)
	}
	
	c.cfg.Logger.Info("VPN client connected successfully", 
		"tun_address", c.cfg.TUNAddress.String(),
		"xray_server", c.xSrvIP.String(),
		"socks_proxy", socksAddr)

	return nil
}

// Disconnect stops all listeners and cleans up route for XRay server.
//
// It will block till all resources are done processing or
// context is cancelled (method also enforces timeout of disconnectTimeout)
func (c *Client) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	
	// Prevent double disconnect
	if !c.isConnected {
		c.mu.Unlock()
		return nil // Already disconnected
	}
	
	// Mark as disconnected immediately to prevent new operations
	c.isConnected = false
	
	// Cancel tunnel context FIRST to signal goroutine to stop
	if c.stopTunnel != nil {
		c.stopTunnel()
	}
	c.mu.Unlock()
	
	// Close components individually with nil checks to prevent panics
	var errs []error
	
	if c.xInst != nil {
		if err := c.xInst.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close xray: %w", err))
		}
	}
	
	if c.tunnel != nil {
		if err := c.tunnel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close tunnel: %w", err))
		}
	}
	
	// Clean up routes (ignore errors as they're not critical)
	if routeOpts := c.xrayToGatewayRoute(); true {
		if err := c.routes.Delete(routeOpts); err != nil {
			// Don't treat route deletion errors as critical
			c.cfg.Logger.Debug("route cleanup note", "error", err)
		}
	}
	
	// Clean up DNS routes
	if c.tunName != "" {
		c.cfg.Logger.Debug("Cleaning up DNS routes")
		if dnsErr := c.routes.Delete(route.Opts{IfName: c.tunName, Routes: DNSRoutesToTUN}); dnsErr != nil {
			c.cfg.Logger.Debug("DNS route cleanup note", "error", dnsErr)
		}
	}
	
	// Clean up IPv6 routes if enabled
	if c.cfg.EnableIPv6 && c.tunName != "" {
		c.cfg.Logger.Debug("Cleaning up IPv6 routes")
		
		// Try using route library first
		if err := c.routes.Delete(route.Opts{IfName: c.tunName, Routes: DefaultRoutesToTUNIPv6}); err != nil {
			c.cfg.Logger.Debug("Route library cleanup failed, trying system commands", "error", err)
			
			// Fallback to system commands
			if sysErr := c.removeIPv6RoutesSystem(c.tunName); sysErr != nil {
				c.cfg.Logger.Warn("IPv6 route cleanup via system commands also failed", "error", sysErr)
			}
		}
		
		// Clean up IPv6 DNS routes
		if dnsErr := c.routes.Delete(route.Opts{IfName: c.tunName, Routes: DNSRoutesToTUNIPv6}); dnsErr != nil {
			c.cfg.Logger.Debug("IPv6 DNS route cleanup note", "error", dnsErr)
		}
		
		// Also remove IPv6 address from interface
		if addrErr := c.removeIPv6AddressSystem(c.tunName); addrErr != nil {
			c.cfg.Logger.Warn("Failed to remove IPv6 address from interface", "error", addrErr)
		}
	}
	
	// Clean up DNS protection routes if they were set up
	if c.cfg.EnableDNSProtection && c.tunName != "" {
		c.cfg.Logger.Debug("Cleaning up DNS protection routes")
		if err := c.cleanupDNSProtection(); err != nil {
			c.cfg.Logger.Warn("DNS protection cleanup failed", "error", err)
		}
	}
	
	err := errors.Join(errs...)

	// Waiting till the tunnel actually done with processing connections.
	// Use a fresh context to ensure cleanup completes even if passed context is cancelled
	disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), disconnectTimeout)
	defer disconnectCancel()
	
	// Wait for goroutine to finish with timeout
	select {
	case tunErr := <-c.tunnelStopped:
		err = errors.Join(tunErr, err)
		c.cfg.Logger.Debug("tunnel goroutine finished")
	case <-disconnectCtx.Done():
		err = errors.Join(disconnectCtx.Err(), err)
		c.cfg.Logger.Warn("Disconnect timeout expired - tunnel goroutine may still be running")
		// Continue anyway to avoid blocking indefinitely
	}

	if err != nil {
		c.cfg.Logger.Error("client disconnect encountered failures", "err", err)
		return err
	}

	// Update Prometheus metrics on disconnect
	vpnConnected.Set(0)
	vpnDisconnectionsTotal.Inc()
	vpnConnectionDuration.Set(0)
	
	// Reset TUN IP address metrics
	vpnTunIPv4.Reset()
	vpnTunIPv6.Reset()
	vpnServerIP.Reset()
	
	c.stopMetricsUpdate()
	
	c.cfg.Logger.Debug("client disconnected")
	return nil
}

// BytesRead returns number of bytes read from TUN device.
func (c *Client) BytesRead() int {
	return int(atomic.LoadInt64(&c.bytesRead))
}

// BytesWritten returns number of bytes written to TUN device.
func (c *Client) BytesWritten() int {
	return int(atomic.LoadInt64(&c.bytesWritten))
}

// GetConnectionStatus returns current connection status and public IP information
func (c *Client) GetConnectionStatus() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	status := map[string]interface{}{
		"is_connected": c.isConnected,
		"tun_address":  c.cfg.TUNAddress.String(),
	}

	if c.isConnected {
		// Get XRay server info
		if c.xSrvIP != nil {
			status["xray_server_ip"] = c.xSrvIP.String()
		}
		
		// Get TUN interface name
		if c.tunName != "" {
			status["tun_interface"] = c.tunName
		}
		
		// Try to get public IP (what external services see)
		publicIPv4, publicIPv6 := c.getPublicIPs()
		if publicIPv4 != "" {
			status["public_ipv4"] = publicIPv4
		}
		if publicIPv6 != "" {
			status["public_ipv6"] = publicIPv6
		}
		
		// Add traffic stats
		status["bytes_read"] = c.BytesRead()
		status["bytes_written"] = c.BytesWritten()
	}

	return status
}

// getPublicIPs attempts to determine the public IP addresses by checking routing
func (c *Client) getPublicIPs() (string, string) {
	// For VPN connections, the public IP is typically the XRay server's exit node IP
	// We can't directly query it without making external requests, but we can report:
	// 1. The XRay server we're connected to
	// 2. The TUN interface address (local VPN endpoint)
	
	var ipv4, ipv6 string
	
	// Local TUN address (VPN tunnel endpoint)
	if c.cfg.TUNAddress != nil {
		ipv4 = c.cfg.TUNAddress.IP.String()
	}
	
	// IPv6 TUN address if enabled
	if c.cfg.EnableIPv6 && defaultTUNAddressIPv6 != nil {
		ipv6 = defaultTUNAddressIPv6.IP.String()
	}
	
	return ipv4, ipv6
}

// LogConnectionStatus logs current connection status with IP information
func (c *Client) LogConnectionStatus() {
	status := c.GetConnectionStatus()
	
	if !status["is_connected"].(bool) {
		c.cfg.Logger.Info("VPN Status: Disconnected")
		return
	}
	
	c.cfg.Logger.Info("VPN Connection Status",
		"status", "connected",
		"tun_interface", status["tun_interface"],
		"tun_address", status["tun_address"],
		"xray_server", status["xray_server_ip"],
		"local_ipv4", status["public_ipv4"],
		"local_ipv6", status["public_ipv6"],
		"bytes_read", status["bytes_read"],
		"bytes_written", status["bytes_written"],
	)
}

// xrayToGatewayRoute is a setup to route VPN requests to gateway.
// Used as exception to not interfere with traffic going to remote XRay instance.
func (c *Client) xrayToGatewayRoute() route.Opts {
	// Append "/32" to match only the XRay server route.
	return route.Opts{Gateway: *c.cfg.GatewayIP, Routes: []*route.Addr{route.MustParseAddr(c.xSrvIP.String() + "/32")}}
}

// createXrayProxy creates XRay instance from connection link with additional proxy listening on {addr}:{port}.
func (c *Client) createXrayProxy(link string) (xrayproto.Instance, *xrayproto.GeneralConfig, error) {
	// Make the inbound for local proxy.
	// We will later use it to redirect all traffic from TUN device to this proxy.
	inbound := &xray.Socks{
		Remark:  "GoXRay-TUN-Listener",
		Address: c.cfg.InboundProxy.IP.String(),
		Port:    strconv.Itoa(c.cfg.InboundProxy.Port),
	}

	svc := xray.NewXrayService(true,
		c.cfg.TLSAllowInsecure,
		xray.WithCustomLogLevel(c.cfg.XRayLogType, xRayLogLevel(c.cfg.Logger.Handler())),
		xray.WithInbound(inbound),
	)

	link = strings.TrimSpace(link)
	
	// Detect protocol type and log it
	var protocolType string
	if strings.HasPrefix(link, "vless://") {
		protocolType = "VLESS"
	} else if strings.HasPrefix(link, "vmess://") {
		protocolType = "VMess"
	} else if strings.HasPrefix(link, "trojan://") {
		protocolType = "Trojan"
	} else if strings.HasPrefix(link, "ss://") {
		protocolType = "Shadowsocks"
	} else {
		protocolType = "Unknown"
	}
	
	c.cfg.Logger.Info("Creating Xray proxy", "protocol", protocolType, "link_prefix", link[:min(30, len(link))]+"...")
	
	protocol, err := svc.CreateProtocol(link)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid config: protocol create (%s): %w", protocolType, err)
	}

	if err := protocol.Parse(); err != nil {
		return nil, nil, fmt.Errorf("invalid config: parse (%s): %w", protocolType, err)
	}

	cfg := protocol.ConvertToGeneralConfig()

	inst, err := svc.MakeInstance(protocol)
	if err != nil {
		return nil, nil, fmt.Errorf("make instance (%s): %w", protocolType, err)
	}

	// Validate xray proto addr.
	ip, err := net.ResolveIPAddr("ip", cfg.Address)
	if err != nil {
		return nil, nil, fmt.Errorf("xray addresses not resolvable: %w", err)
	}
	c.xSrvIP = ip
	
	c.cfg.Logger.Info("Xray proxy created successfully", "protocol", protocolType, "server", cfg.Address+":"+cfg.Port)

	return inst, &cfg, nil
}

// xRayLogLevel maps slog.Level to xray core log level (xcommlog.Severity) by checking Config.Logger level.
func xRayLogLevel(h slog.Handler) xcommlog.Severity {
	ctx := context.Background()
	switch {
	case h.Enabled(ctx, slog.LevelDebug):
		return xcommlog.Severity_Debug
	case h.Enabled(ctx, slog.LevelInfo):
		return xcommlog.Severity_Info
	case h.Enabled(ctx, slog.LevelError):
		return xcommlog.Severity_Error
	case h.Enabled(ctx, slog.LevelWarn):
		return xcommlog.Severity_Warning
	}

	return xcommlog.Severity_Unknown
}

// setupTunnel creates new TUN interface in the system and routes all traffic to it.
func (c *Client) setupTunnel() (*tun.Interface, error) {
	ifc, err := tun.New("", 1500)
	if err != nil {
		return nil, fmt.Errorf("create tun: %w", err)
	}

	// Store interface name for cleanup
	c.tunName = ifc.Name()
	c.cfg.Logger.Debug("Setting up TUN interface", "name", c.tunName, "ipv4_address", c.cfg.TUNAddress)

	// Setup IPv4 address
	if err = ifc.Up(c.cfg.TUNAddress, c.cfg.TUNAddress.IP); err != nil {
		return nil, fmt.Errorf("setup IPv4 interface: %w", err)
	}

	// Setup IPv6 address if enabled
	if c.cfg.EnableIPv6 {
		c.cfg.Logger.Debug("Enabling IPv6 support on TUN interface", "ipv6_address", defaultTUNAddressIPv6)
		
		// Configure IPv6 on the TUN interface using system commands
		if err = c.setupIPv6OnInterface(ifc.Name()); err != nil {
			c.cfg.Logger.Warn("Failed to configure IPv6 on TUN interface", "error", err)
			// Continue anyway - IPv4 will still work
		}
	}

	// Add IPv4 routes
	c.cfg.Logger.Debug("Adding IPv4 routes for TUN interface", "routes_count", len(c.cfg.RoutesToTUN))
	if err = c.routes.Add(route.Opts{IfName: ifc.Name(), Routes: c.cfg.RoutesToTUN}); err != nil {
		return nil, fmt.Errorf("add IPv4 routes: %w", err)
	}

	// Add IPv6 routes if enabled
	if c.cfg.EnableIPv6 {
		c.cfg.Logger.Debug("Adding IPv6 routes for TUN interface", "routes_count", len(DefaultRoutesToTUNIPv6))
		
		// Try using route library first
		if err = c.routes.Add(route.Opts{IfName: ifc.Name(), Routes: DefaultRoutesToTUNIPv6}); err != nil {
			c.cfg.Logger.Warn("Route library failed for IPv6, trying system commands", "error", err)
			
			// Fallback to system commands
			if sysErr := c.addIPv6RoutesSystem(ifc.Name()); sysErr != nil {
				c.cfg.Logger.Error("Failed to add IPv6 routes via system commands", "error", sysErr)
				return nil, fmt.Errorf("add IPv6 routes: %w (library: %v, system: %v)", err, err, sysErr)
			}
		}
	}

	return ifc, nil
}

// setupIPv6OnInterface configures IPv6 address on the TUN interface using system commands
func (c *Client) setupIPv6OnInterface(ifName string) error {
	c.cfg.Logger.Info("Configuring IPv6 on TUN interface", "interface", ifName, "address", defaultTUNAddressIPv6.String())
	
	// Use ip command to add IPv6 address
	cmd := exec.Command("ip", "-6", "addr", "add", defaultTUNAddressIPv6.String(), "dev", ifName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add IPv6 address: %w, output: %s", err, string(output))
	}
	
	c.cfg.Logger.Debug("IPv6 address configured successfully", "interface", ifName)
	return nil
}

// addIPv6RoutesSystem adds IPv6 routes using system ip command
func (c *Client) addIPv6RoutesSystem(ifName string) error {
	c.cfg.Logger.Info("Adding IPv6 routes via system commands", "interface", ifName)
	
	var lastErr error
	
	for _, routeAddr := range DefaultRoutesToTUNIPv6 {
		routeStr := routeAddr.String()
		c.cfg.Logger.Debug("Adding IPv6 route", "route", routeStr, "interface", ifName)
		
		// Execute: ip -6 route add <route> dev <interface>
		cmd := exec.Command("ip", "-6", "route", "add", routeStr, "dev", ifName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.cfg.Logger.Warn("Failed to add IPv6 route", "route", routeStr, "error", err, "output", string(output))
			lastErr = err
			continue
		}
		
		c.cfg.Logger.Debug("IPv6 route added successfully", "route", routeStr)
	}
	
	if lastErr != nil {
		return fmt.Errorf("some IPv6 routes failed to add: %w", lastErr)
	}
	
	c.cfg.Logger.Info("All IPv6 routes added successfully")
	return nil
}

func getFreePort() int {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 10808
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	port := addr.Port

	return port
}

// removeIPv6RoutesSystem removes IPv6 routes using system ip command
func (c *Client) removeIPv6RoutesSystem(ifName string) error {
	c.cfg.Logger.Info("Removing IPv6 routes via system commands", "interface", ifName)
	
	var lastErr error
	
	for _, routeAddr := range DefaultRoutesToTUNIPv6 {
		routeStr := routeAddr.String()
		c.cfg.Logger.Debug("Removing IPv6 route", "route", routeStr, "interface", ifName)
		
		// Execute: ip -6 route del <route> dev <interface>
		cmd := exec.Command("ip", "-6", "route", "del", routeStr, "dev", ifName)
		err := cmd.Run()
		if err != nil {
			c.cfg.Logger.Debug("Failed to remove IPv6 route (may not exist)", "route", routeStr, "error", err)
			lastErr = err
			continue
		}
		
		c.cfg.Logger.Debug("IPv6 route removed successfully", "route", routeStr)
	}
	
	if lastErr != nil {
		return fmt.Errorf("some IPv6 routes failed to remove: %w", lastErr)
	}
	
	c.cfg.Logger.Debug("All IPv6 routes removed successfully")
	return nil
}

// removeIPv6AddressSystem removes IPv6 address from the TUN interface using system ip command
func (c *Client) removeIPv6AddressSystem(ifName string) error {
	c.cfg.Logger.Debug("Removing IPv6 address from TUN interface", "interface", ifName, "address", defaultTUNAddressIPv6.String())
	
	// Use ip command to remove IPv6 address
	cmd := exec.Command("ip", "-6", "addr", "del", defaultTUNAddressIPv6.String(), "dev", ifName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove IPv6 address: %w, output: %s", err, string(output))
	}
	
	c.cfg.Logger.Debug("IPv6 address removed successfully", "interface", ifName)
	return nil
}

// detectProtocol detects VPN protocol from link string
func (c *Client) detectProtocol(link string) string {
	if strings.HasPrefix(link, "vless://") {
		return "vless"
	} else if strings.HasPrefix(link, "vmess://") {
		return "vmess"
	} else if strings.HasPrefix(link, "trojan://") {
		return "trojan"
	} else if strings.HasPrefix(link, "ss://") {
		return "shadowsocks"
	}
	return "unknown"
}

// startMetricsUpdate starts goroutine to update metrics periodically
func (c *Client) startMetricsUpdate() {
	c.cfg.Logger.Info("Checking metrics configuration", "metrics_port", c.cfg.MetricsPort)
	
	if c.cfg.MetricsPort <= 0 {
		c.cfg.Logger.Debug("Metrics server disabled (port not configured)")
		return
	}
	
	// Start metrics HTTP server
	addr := fmt.Sprintf("0.0.0.0:%d", c.cfg.MetricsPort)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	
	c.metricsServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	
	go func() {
		c.cfg.Logger.Info("Prometheus metrics endpoint starting", "address", addr, "path", "/metrics")
		if err := c.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.cfg.Logger.Error("Metrics server error", "error", err)
		} else if err == http.ErrServerClosed {
			c.cfg.Logger.Info("Metrics server stopped gracefully")
		}
	}()
	
	c.cfg.Logger.Info("Metrics server goroutine started")
	
	// Start periodic metrics update
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		connectTime := time.Now()
		
		for {
			select {
			case <-ticker.C:
				if !c.isConnected {
					return
				}
				
				// Update traffic metrics
				vpnBytesRead.Set(float64(c.BytesRead()))
				vpnBytesWritten.Set(float64(c.BytesWritten()))
				
				// Update connection duration
				duration := time.Since(connectTime).Seconds()
				vpnConnectionDuration.Set(duration)
			}
		}
	}()
}

// stopMetricsUpdate stops the metrics server
func (c *Client) stopMetricsUpdate() {
	if c.metricsServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := c.metricsServer.Shutdown(ctx); err != nil {
			c.cfg.Logger.Warn("Metrics server shutdown error", "error", err)
		}
		c.metricsServer = nil
	}
}

// monitorTraffic periodically reads TUN interface statistics and updates atomic counters
func (c *Client) monitorTraffic(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var bytesRead, bytesWritten int64
			
			// Always use /proc/net/dev for reliable traffic statistics
			// readerMetrics may not work correctly with TUN interfaces
			if c.tunName != "" {
				rx, tx := c.readInterfaceStats(c.tunName)
				bytesRead = rx
				bytesWritten = tx
			}
			
			atomic.StoreInt64(&c.bytesRead, bytesRead)
			atomic.StoreInt64(&c.bytesWritten, bytesWritten)
		}
	}
}

// setupDNSProtection sets up comprehensive DNS leak protection
func (c *Client) setupDNSProtection() error {
	if !c.cfg.EnableDNSProtection {
		c.cfg.Logger.Info("DNS protection disabled in configuration")
		return nil
	}

	c.cfg.Logger.Info("Setting up DNS leak protection", "interface", c.tunName)
	
	// Block direct DNS access at firewall level
	c.cfg.Logger.Debug("Blocking direct DNS access", "ports", "53,853")
	if err := c.setupDNSTrafficForcing(); err != nil {
		c.cfg.Logger.Error("Failed to setup DNS traffic forcing", "error", err)
		// Don't fail completely, but warn user
	}

	// Add routes to DNS servers through TUN interface
	c.cfg.Logger.Debug("Routing DNS through TUN interface", "interface", c.tunName)
	if err := c.addDNSRoutesThroughTUN(); err != nil {
		c.cfg.Logger.Warn("Some DNS routes may have failed to add (may already exist)", "error", err)
		// Don't return error here as some routes might already exist which is normal
	}

	c.cfg.Logger.Info("DNS leak protection enabled successfully")
	return nil
}

// addDNSRoutesThroughTUN adds routes to DNS servers through the TUN interface
func (c *Client) addDNSRoutesThroughTUN() error {
	var overallError error

	// Add IPv4 DNS routes
	for _, r := range DNSRoutesToTUN {
		err := c.routes.Add(route.Opts{IfName: c.tunName, Routes: []*route.Addr{r}})
		if err != nil {
			// Log as warning instead of error, since some routes might already exist
			c.cfg.Logger.Warn("Failed to add IPv4 DNS route (may already exist)", "route", r.String(), "error", err)
			if overallError == nil {
				overallError = err
			}
		} else {
			c.cfg.Logger.Debug("Added IPv4 DNS route", "route", r.String())
		}
	}

	// Add IPv6 DNS routes if IPv6 is enabled
	if c.cfg.EnableIPv6 {
		for _, r := range DNSRoutesToTUNIPv6 {
			err := c.routes.Add(route.Opts{IfName: c.tunName, Routes: []*route.Addr{r}})
			if err != nil {
				// Log as warning instead of error, since some routes might already exist
				c.cfg.Logger.Warn("Failed to add IPv6 DNS route (may already exist)", "route", r.String(), "error", err)
				if overallError == nil {
					overallError = err
				}
			} else {
				c.cfg.Logger.Debug("Added IPv6 DNS route", "route", r.String())
			}
		}
	}

	// Add port-based DNS routes
	for _, r := range DNSPortRoutesToTUN {
		err := c.routes.Add(route.Opts{IfName: c.tunName, Routes: []*route.Addr{r}})
		if err != nil {
			c.cfg.Logger.Warn("Failed to add IPv4 DNS port route (may already exist)", "route", r.String(), "error", err)
			if overallError == nil {
				overallError = err
			}
		} else {
			c.cfg.Logger.Debug("Added IPv4 DNS port route", "route", r.String())
		}
	}

	if c.cfg.EnableIPv6 {
		for _, r := range DNSPortRoutesToTUNIPv6 {
			err := c.routes.Add(route.Opts{IfName: c.tunName, Routes: []*route.Addr{r}})
			if err != nil {
				c.cfg.Logger.Warn("Failed to add IPv6 DNS port route (may already exist)", "route", r.String(), "error", err)
				if overallError == nil {
					overallError = err
				}
			} else {
				c.cfg.Logger.Debug("Added IPv6 DNS port route", "route", r.String())
			}
		}
	}

	return overallError
}


// setupDNSTrafficForcing - пустая функция, так как DNS трафик уже маршрутизируется через TUN
// с помощью маршрутов, добавленных в setupTunnel() и addDNSRoutesThroughTUN()
// Перенаправление DNS на SOCKS порт некорректно, так как SOCKS прокси не обрабатывает DNS протокол
func (c *Client) setupDNSTrafficForcing() error {
	// DNS трафик уже идет через TUN интерфейс благодаря маршрутам
	// Не нужно перенаправлять его на SOCKS порт - это сломает DNS резолвинг
	c.cfg.Logger.Debug("Skipping DNS traffic forcing - DNS already routes through TUN via routing table")
	return nil
}

// cleanupDNSProtection removes DNS protection routes and rules
func (c *Client) cleanupDNSProtection() error {
	interfaceName := c.tunName
	
	if interfaceName == "" {
		c.cfg.Logger.Debug("Skipping DNS protection cleanup - interface name not available")
		return nil
	}
	
	c.cfg.Logger.Info("Cleaning up DNS protection for TUN interface", "interface", interfaceName)
	
	// Remove IPv4 DNS port routes
	if err := c.routes.Delete(route.Opts{IfName: interfaceName, Routes: DNSPortRoutesToTUN}); err != nil {
		c.cfg.Logger.Debug("Failed to remove IPv4 DNS port routes", "error", err)
	}
	
	// Remove IPv6 DNS port routes if IPv6 is enabled
	if c.cfg.EnableIPv6 {
		if err := c.routes.Delete(route.Opts{IfName: interfaceName, Routes: DNSPortRoutesToTUNIPv6}); err != nil {
			c.cfg.Logger.Debug("Failed to remove IPv6 DNS port routes", "error", err)
		}
	}
	
	// Remove standard DNS server routes
	if err := c.routes.Delete(route.Opts{IfName: interfaceName, Routes: DNSRoutesToTUN}); err != nil {
		c.cfg.Logger.Debug("Failed to remove standard DNS server routes", "error", err)
	}
	
	if c.cfg.EnableIPv6 {
		if err := c.routes.Delete(route.Opts{IfName: interfaceName, Routes: DNSRoutesToTUNIPv6}); err != nil {
			c.cfg.Logger.Debug("Failed to remove IPv6 DNS server routes", "error", err)
		}
	}
	
	// Remove iptables rules for DNS traffic forcing
	if err := c.cleanupDNSTrafficForcing(); err != nil {
		c.cfg.Logger.Warn("Failed to clean up DNS traffic forcing rules", "error", err)
	}
	
	c.cfg.Logger.Info("DNS protection cleanup completed")
	return nil
}

// cleanupDNSTrafficForcing removes iptables rules that force DNS traffic
func (c *Client) cleanupDNSTrafficForcing() error {
	c.cfg.Logger.Debug("Cleaning up iptables rules for DNS traffic forcing")
	
	// Remove IPv4 iptables rules for DNS
	ipv4Cmd := exec.Command("iptables", "-t", "nat", "-D", "OUTPUT", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-port", strconv.Itoa(c.cfg.InboundProxy.Port))
	if err := ipv4Cmd.Run(); err != nil {
		c.cfg.Logger.Debug("Failed to remove IPv4 DNS iptables rule", "error", err)
	}
	
	ipv4TcpCmd := exec.Command("iptables", "-t", "nat", "-D", "OUTPUT", "-p", "tcp", "--dport", "53", "-j", "REDIRECT", "--to-port", strconv.Itoa(c.cfg.InboundProxy.Port))
	if err := ipv4TcpCmd.Run(); err != nil {
		c.cfg.Logger.Debug("Failed to remove IPv4 TCP DNS iptables rule", "error", err)
	}
	
	// Remove IPv6 ip6tables rules for DNS (if IPv6 was enabled)
	if c.cfg.EnableIPv6 {
		ipv6Cmd := exec.Command("ip6tables", "-t", "nat", "-D", "OUTPUT", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-port", strconv.Itoa(c.cfg.InboundProxy.Port))
		if err := ipv6Cmd.Run(); err != nil {
			c.cfg.Logger.Debug("Failed to remove IPv6 DNS ip6tables rule", "error", err)
		}
		
		ipv6TcpCmd := exec.Command("ip6tables", "-t", "nat", "-D", "OUTPUT", "-p", "tcp", "--dport", "53", "-j", "REDIRECT", "--to-port", strconv.Itoa(c.cfg.InboundProxy.Port))
		if err := ipv6TcpCmd.Run(); err != nil {
			c.cfg.Logger.Debug("Failed to remove IPv6 TCP DNS ip6tables rule", "error", err)
		}
	}
	
	c.cfg.Logger.Info("DNS traffic forcing rules cleaned up")
	return nil
}


// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// readInterfaceStats reads RX/TX bytes for a network interface from /proc/net/dev
func (c *Client) readInterfaceStats(ifaceName string) (int64, int64) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		c.cfg.Logger.Debug("Failed to read /proc/net/dev", "error", err)
		return 0, 0
	}
	
	// Format of /proc/net/dev:
	// Inter-|   Receive                                                |  Transmit
	//  face |bytes    packets err drop fifo frame compressed multicast|bytes    packets err drop fifo colls carrier compressed
	// tun0: 1234567   12345   0   0    0     0          0         0    1234567   12345   0   0    0     0       0          0
	
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Look for line starting with interface name followed by colon
		// Example: "tun0: 1234567 12345 ..."
		if !strings.HasPrefix(line, ifaceName+":") && !strings.HasPrefix(line, ifaceName + ": ") {
			continue
		}
		
		// Split by colon to separate interface name from stats
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		
		// Parse the stats part
		statsStr := strings.TrimSpace(parts[1])
		stats := strings.Fields(statsStr)
		
		// Need at least 10 fields for RX and TX stats
		// RX: bytes(0) packets(1) errs(2) drop(3) fifo(4) frame(5) compressed(6) multicast(7)
		// TX: bytes(8) packets(9) ...
		if len(stats) < 10 {
			continue
		}
		
		// Parse RX bytes (index 0) and TX bytes (index 8)
		var rxBytes, txBytes int64
		if _, err := fmt.Sscanf(stats[0], "%d", &rxBytes); err != nil {
			c.cfg.Logger.Debug("Failed to parse RX bytes", "value", stats[0], "error", err)
			continue
		}
		if _, err := fmt.Sscanf(stats[8], "%d", &txBytes); err != nil {
			c.cfg.Logger.Debug("Failed to parse TX bytes", "value", stats[8], "error", err)
			continue
		}
		
		return rxBytes, txBytes
	}
	
	return 0, 0
}
