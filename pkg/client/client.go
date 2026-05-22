package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goxray/core/network/route"
	"github.com/goxray/core/network/tun"
	"github.com/goxray/core/pipe2socks"
	"github.com/jackpal/gateway"

	xrayproto "github.com/lilendian0x00/xray-knife/v3/pkg/protocol"
	"github.com/lilendian0x00/xray-knife/v3/pkg/xray"
	xapplog "github.com/xtls/xray-core/app/log"
	xcommlog "github.com/xtls/xray-core/common/log"
)

const disconnectTimeout = 30 * time.Second

var (
	// defaultTUNAddress is the address new TUN device will be set up with.
	defaultTUNAddress = &net.IPNet{IP: net.IPv4(192, 18, 0, 1), Mask: net.IPv4Mask(255, 255, 255, 255)}
	
	// defaultTUNAddressIPv6 is the IPv6 address for TUN device (ULA range).
	defaultTUNAddressIPv6 = &net.IPNet{
		IP:   net.ParseIP("fd00:goxray::1"),
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
	
	// DefaultRoutesToTUNIPv6 will route all IPv6 system traffic through the TUN.
	DefaultRoutesToTUNIPv6 = []*route.Addr{
		// Reroute all IPv6 traffic.
		route.MustParseAddr("::/1"),
		route.MustParseAddr("8000::/1"),
	}
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

	c.cfg.Logger.Debug("Setting up TUN device")
	// Create TUN and route all traffic to it.
	c.tunnel, err = c.setupTunnel()
	if err != nil {
		c.cfg.Logger.Error("TUN creation failed", "err", err)

		return fmt.Errorf("setup TUN device: %w", err)
	}
	c.tunnel = newReaderMetrics(c.tunnel)
	c.cfg.Logger.Debug("TUN device created")

	c.cfg.Logger.Debug("adding routes for TUN device")
	// Set XRay remote address to be routed through the default gateway, so that we don't get a loop.
	_ = c.routes.Delete(c.xrayToGatewayRoute()) // In case previous run failed.
	c.cfg.Logger.Debug("deleted dangling routes")
	err = c.routes.Add(c.xrayToGatewayRoute())
	if err != nil {
		c.cfg.Logger.Error("routing xray server IP to default route failed", "err", err, "route", c.xrayToGatewayRoute())

		return fmt.Errorf("add xray server route exception: %w", err)
	}
	c.cfg.Logger.Debug("routing xray server IP to default route")

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
	
	c.isConnected = true
	c.cfg.Logger.Debug("client connected")

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
	
	// Clean up IPv6 routes if enabled
	if c.cfg.EnableIPv6 && c.tunName != "" {
		c.cfg.Logger.Debug("Cleaning up IPv6 routes")
		
		// Try route library first
		if err := c.routes.Delete(route.Opts{IfName: c.tunName, Routes: DefaultRoutesToTUNIPv6}); err != nil {
			c.cfg.Logger.Debug("Route library cleanup failed, trying system commands", "error", err)
			
			// Fallback to system commands
			if sysErr := c.removeIPv6RoutesSystem(c.tunName); sysErr != nil {
				c.cfg.Logger.Warn("IPv6 route cleanup via system commands also failed", "error", sysErr)
			}
		}
		
		// Also remove IPv6 address from interface
		if addrErr := c.removeIPv6AddressSystem(c.tunName); addrErr != nil {
			c.cfg.Logger.Warn("Failed to remove IPv6 address from interface", "error", addrErr)
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

	c.cfg.Logger.Debug("client disconnected")
	return nil
}

// BytesRead returns number of bytes read from TUN device.
func (c *Client) BytesRead() int {
	if c.tunnel == nil {
		return 0
	}

	return c.tunnel.(*readerMetrics).BytesRead()
}

// BytesWritten returns number of bytes written to TUN device.
func (c *Client) BytesWritten() int {
	if c.tunnel == nil {
		return 0
	}

	return c.tunnel.(*readerMetrics).BytesWritten()
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
	protocol, err := svc.CreateProtocol(link)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid config: protocol create: %w", err)
	}

	if err := protocol.Parse(); err != nil {
		return nil, nil, fmt.Errorf("invalid config: parse: %w", err)
	}

	cfg := protocol.ConvertToGeneralConfig()

	inst, err := svc.MakeInstance(protocol)
	if err != nil {
		return nil, nil, fmt.Errorf("make instance: %w", err)
	}

	// Validate xray proto addr.
	ip, err := net.ResolveIPAddr("ip", cfg.Address)
	if err != nil {
		return nil, nil, fmt.Errorf("xray address not resolvable: %w", err)
	}
	c.xSrvIP = ip

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
