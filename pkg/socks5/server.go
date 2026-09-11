package socks5

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

// SOCKS5 protocol constants
const (
	socks5Version = 0x05
	
	// Authentication methods
	authNone     = 0x00
	authPassword = 0x02
	authNoAccept = 0xFF
	
	// Commands
	cmdConnect = 0x01
	cmdBind    = 0x02
	cmdUDP     = 0x03
	
	// Address types
	addrIPv4   = 0x01
	addrDomain = 0x03
	addrIPv6   = 0x04
	
	// Reply codes
	replySuccess              = 0x00
	replyGeneralFailure       = 0x01
	replyConnectionNotAllowed = 0x02
	replyNetworkUnreachable   = 0x03
	replyHostUnreachable      = 0x04
	replyConnectionRefused    = 0x05
	replyTTLExpired           = 0x06
	replyCommandNotSupported  = 0x07
	replyAddressNotSupported  = 0x08
)

// Config holds SOCKS5 server configuration
type Config struct {
	// Listen address (e.g., "0.0.0.0:1080")
	ListenAddr string
	
	// Authentication (optional)
	Username string
	Password string
	
	// Timeout for connections
	Timeout time.Duration
	
	// Logger
	Logger *slog.Logger
}

// Server represents a SOCKS5 proxy server
type Server struct {
	config   Config
	listener net.Listener
	logger   *slog.Logger
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewServer creates a new SOCKS5 server
func NewServer(config Config) *Server {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Server{
		config: config,
		logger: config.Logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the SOCKS5 server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.ListenAddr, err)
	}
	
	s.listener = listener
	s.logger.Info("SOCKS5 server started", "address", s.config.ListenAddr)
	
	s.wg.Add(1)
	go s.acceptLoop()
	
	return nil
}

// Stop stops the SOCKS5 server
func (s *Server) Stop() error {
	s.logger.Info("Stopping SOCKS5 server")
	s.cancel()
	
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.logger.Warn("Error closing listener", "error", err)
		}
	}
	
	s.wg.Wait()
	s.logger.Info("SOCKS5 server stopped")
	return nil
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop() {
	defer s.wg.Done()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		
		// Set accept deadline to allow checking context
		if err := s.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
			s.logger.Error("Failed to set deadline", "error", err)
			continue
		}
		
		conn, err := s.listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Timeout is expected, check context and continue
			}
			
			select {
			case <-s.ctx.Done():
				return
			default:
				s.logger.Error("Failed to accept connection", "error", err)
				continue
			}
		}
		
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single SOCKS5 connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer closeGracefully(conn)
	
	// Bound only the establishment phase (handshake, authentication and the
	// CONNECT request) so that idle or stalled peers cannot hold a goroutine
	// indefinitely. The deadline is lifted in processRequest once the tunnel is
	// established and payload relaying begins.
	if err := conn.SetDeadline(time.Now().Add(s.config.Timeout)); err != nil {
		s.logger.Error("Failed to set connection deadline", "error", err)
		return
	}
	
	// SOCKS5 handshake
	if err := s.handshake(conn); err != nil {
		s.logger.Warn("Handshake failed", "error", err, "remote", conn.RemoteAddr())
		return
	}
	
	// Authentication (if required)
	if s.config.Username != "" || s.config.Password != "" {
		if err := s.authenticate(conn); err != nil {
			s.logger.Warn("Authentication failed", "error", err, "remote", conn.RemoteAddr())
			return
		}
	}
	
	// Process request
	if err := s.processRequest(conn); err != nil {
		s.logger.Warn("Request processing failed", "error", err, "remote", conn.RemoteAddr())
		return
	}
}

// closeDrainTimeout bounds how long closeGracefully waits for the peer to stop
// sending before the socket is torn down. It only needs to cover a request the
// server deliberately stopped parsing halfway through.
const closeDrainTimeout = 250 * time.Millisecond

// closeGracefully shuts conn down so that whatever the server already wrote is
// guaranteed to reach the peer.
//
// Closing a socket that still holds unread received data makes TCP send a RST
// rather than a FIN (RFC 1122 section 4.2.2.13), and that RST discards data
// already queued at the peer. The failure paths reply and return without
// consuming the rest of the request - for an unsupported command or address
// type the server cannot even know how many octets remain - so a plain Close
// would swallow the reply and the client would only observe a connection reset.
// Draining first lets the connection end with a FIN instead.
func closeGracefully(conn net.Conn) {
	// Signal the peer that no more data will be sent. The read side stays open.
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.CloseWrite()
	}
	
	// Absorb anything already buffered. The deadline keeps a peer that never
	// closes its side from stalling shutdown, which Stop waits for.
	_ = conn.SetDeadline(time.Now().Add(closeDrainTimeout))
	_, _ = io.Copy(io.Discard, conn)
	
	_ = conn.Close()
}

// handshake performs SOCKS5 handshake
func (s *Server) handshake(conn net.Conn) error {
	// Read version and methods
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return fmt.Errorf("read version: %w", err)
	}
	
	version := buf[0]
	nMethods := buf[1]
	
	if version != socks5Version {
		return fmt.Errorf("unsupported SOCKS version: %d", version)
	}
	
	// Read methods
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return fmt.Errorf("read methods: %w", err)
	}
	
	// Select authentication method
	var selectedMethod byte = authNoAccept
	
	if s.config.Username != "" || s.config.Password != "" {
		// Password authentication required
		for _, method := range methods {
			if method == authPassword {
				selectedMethod = authPassword
				break
			}
		}
	} else {
		// No authentication required
		for _, method := range methods {
			if method == authNone {
				selectedMethod = authNone
				break
			}
		}
	}
	
	// Send selected method
	if _, err := conn.Write([]byte{socks5Version, selectedMethod}); err != nil {
		return fmt.Errorf("write method: %w", err)
	}
	
	if selectedMethod == authNoAccept {
		return fmt.Errorf("no acceptable authentication method")
	}
	
	return nil
}

// authenticate performs username/password authentication
func (s *Server) authenticate(conn net.Conn) error {
	// Read version
	buf := make([]byte, 1)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return fmt.Errorf("read auth version: %w", err)
	}
	
	if buf[0] != 0x01 {
		return fmt.Errorf("unsupported auth version: %d", buf[0])
	}
	
	// Read username length
	if _, err := io.ReadFull(conn, buf); err != nil {
		return fmt.Errorf("read username length: %w", err)
	}
	usernameLen := int(buf[0])
	
	// Read username
	username := make([]byte, usernameLen)
	if _, err := io.ReadFull(conn, username); err != nil {
		return fmt.Errorf("read username: %w", err)
	}
	
	// Read password length
	if _, err := io.ReadFull(conn, buf); err != nil {
		return fmt.Errorf("read password length: %w", err)
	}
	passwordLen := int(buf[0])
	
	// Read password
	password := make([]byte, passwordLen)
	if _, err := io.ReadFull(conn, password); err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	
	// Verify credentials
	var status byte = 0x01 // Failure
	if string(username) == s.config.Username && string(password) == s.config.Password {
		status = 0x00 // Success
	}
	
	// Send response
	if _, err := conn.Write([]byte{0x01, status}); err != nil {
		return fmt.Errorf("write auth response: %w", err)
	}
	
	if status != 0x00 {
		return fmt.Errorf("invalid credentials")
	}
	
	return nil
}

// processRequest processes SOCKS5 request
func (s *Server) processRequest(conn net.Conn) error {
	// Read request header
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return fmt.Errorf("read request header: %w", err)
	}
	
	version := buf[0]
	cmd := buf[1]
	// buf[2] is reserved
	addrType := buf[3]
	
	if version != socks5Version {
		return fmt.Errorf("unsupported SOCKS version: %d", version)
	}
	
	// Only support CONNECT command
	if cmd != cmdConnect {
		s.sendReply(conn, replyCommandNotSupported, addrType)
		return fmt.Errorf("unsupported command: %d", cmd)
	}
	
	// Reject address types this implementation cannot parse with the dedicated
	// reply code mandated by RFC 1928 section 6, rather than the generic failure,
	// so that clients can distinguish a malformed request from a transport error.
	switch addrType {
	case addrIPv4, addrDomain, addrIPv6:
	default:
		s.sendReply(conn, replyAddressNotSupported, addrType)
		return fmt.Errorf("unsupported address type: %#x", addrType)
	}
	
	// Read destination address
	destAddr, err := s.readAddress(conn, addrType)
	if err != nil {
		s.sendReply(conn, replyGeneralFailure, addrType)
		return fmt.Errorf("read address: %w", err)
	}
	
	s.logger.Debug("SOCKS5 CONNECT request", "destination", destAddr, "remote", conn.RemoteAddr())
	
	// Connect to destination
	destConn, err := net.DialTimeout("tcp", destAddr, s.config.Timeout)
	if err != nil {
		s.logger.Warn("Failed to connect to destination", "destination", destAddr, "error", err)
		s.sendReply(conn, replyHostUnreachable, addrType)
		return fmt.Errorf("connect to destination: %w", err)
	}
	defer destConn.Close()
	
	// Send success reply
	if err := s.sendReply(conn, replySuccess, addrType); err != nil {
		return fmt.Errorf("send reply: %w", err)
	}
	
	s.logger.Info("SOCKS5 connection established", "destination", destAddr, "remote", conn.RemoteAddr())
	
	// Lift the establishment deadline before relaying payload. config.Timeout
	// bounds connection setup only; leaving it armed would tear every proxied
	// connection down Timeout seconds after it was accepted, breaking long-lived
	// streams such as video playback, SSH sessions or large downloads.
	if err := conn.SetDeadline(time.Time{}); err != nil {
		s.logger.Warn("Failed to clear connection deadline", "error", err)
	}
	
	// Relay data between client and destination
	s.relay(s.ctx, conn, destConn)
	
	return nil
}

// readAddress reads destination address from connection
func (s *Server) readAddress(conn net.Conn, addrType byte) (string, error) {
	switch addrType {
	case addrIPv4:
		buf := make([]byte, 4)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return "", err
		}
		ip := net.IP(buf)
		
		// Read port
		portBuf := make([]byte, 2)
		if _, err := io.ReadFull(conn, portBuf); err != nil {
			return "", err
		}
		port := int(portBuf[0])<<8 | int(portBuf[1])
		
		return fmt.Sprintf("%s:%d", ip.String(), port), nil
		
	case addrDomain:
		// Read domain length
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return "", err
		}
		domainLen := int(lenBuf[0])
		
		// Read domain
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", err
		}
		
		// Read port
		portBuf := make([]byte, 2)
		if _, err := io.ReadFull(conn, portBuf); err != nil {
			return "", err
		}
		port := int(portBuf[0])<<8 | int(portBuf[1])
		
		return fmt.Sprintf("%s:%d", string(domain), port), nil
		
	case addrIPv6:
		buf := make([]byte, 16)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return "", err
		}
		ip := net.IP(buf)
		
		// Read port
		portBuf := make([]byte, 2)
		if _, err := io.ReadFull(conn, portBuf); err != nil {
			return "", err
		}
		port := int(portBuf[0])<<8 | int(portBuf[1])
		
		return fmt.Sprintf("[%s]:%d", ip.String(), port), nil
		
	default:
		return "", fmt.Errorf("unsupported address type: %d", addrType)
	}
}

// sendReply sends a SOCKS5 reply.
//
// BND.ADDR and BND.PORT report the address the server bound to. This
// implementation never binds a particular address for CONNECT, so a
// placeholder is reported instead. The ATYP octet must always agree with the
// number of address octets that follow it: the previous version echoed the
// ATYP requested by the client but only appended address bytes for IPv4 and
// IPv6, so domain-name requests produced a truncated 6-octet reply that
// clients could not parse. Domain and unknown address types are therefore
// reported in the fixed-width IPv4 form.
func (s *Server) sendReply(conn net.Conn, reply byte, addrType byte) error {
	// Only IPv6 keeps its own address family; IPv4, domain-name and unknown
	// address types are all reported as the 4-octet IPv4 placeholder.
	boundAddrType := addrType
	if boundAddrType != addrIPv6 {
		boundAddrType = addrIPv4
	}
	
	// Build reply: VER REP RSV ATYP BND.ADDR BND.PORT
	resp := []byte{
		socks5Version,
		reply,
		0x00, // Reserved
		boundAddrType,
	}
	
	// Add bind address (0.0.0.0 or :: for simplicity)
	if boundAddrType == addrIPv6 {
		resp = append(resp, make([]byte, net.IPv6len)...) // ::
	} else {
		resp = append(resp, make([]byte, net.IPv4len)...) // 0.0.0.0
	}
	
	// Add bind port (0)
	resp = append(resp, 0, 0)
	
	_, err := conn.Write(resp)
	return err
}

// relay relays data between two connections
func (s *Server) relay(ctx context.Context, client, dest net.Conn) {
	// Relays are no longer bounded by a deadline, so make them interruptible.
	// Closing either endpoint unblocks the io.Copy running in the opposite
	// direction, which lets Stop() return promptly instead of waiting on peers
	// that never close their side.
	watcherDone := make(chan struct{})
	defer close(watcherDone)
	go func() {
		select {
		case <-ctx.Done():
			client.Close()
			dest.Close()
		case <-watcherDone:
		}
	}()
	
	var wg sync.WaitGroup
	wg.Add(2)
	
	// Client -> Destination
	go func() {
		defer wg.Done()
		written, err := io.Copy(dest, client)
		if err != nil {
			s.logger.Debug("Client->Dest copy error", "error", err, "bytes", written)
		}
		dest.(*net.TCPConn).CloseWrite()
	}()
	
	// Destination -> Client
	go func() {
		defer wg.Done()
		written, err := io.Copy(client, dest)
		if err != nil {
			s.logger.Debug("Dest->Client copy error", "error", err, "bytes", written)
		}
		client.(*net.TCPConn).CloseWrite()
	}()
	
	wg.Wait()
}
