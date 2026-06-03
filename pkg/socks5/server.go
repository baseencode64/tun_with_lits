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
	defer conn.Close()
	
	// Set connection timeout
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
	
	// Relay data between client and destination
	s.relay(conn, destConn)
	
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

// sendReply sends SOCKS5 reply
func (s *Server) sendReply(conn net.Conn, reply byte, addrType byte) error {
	// Build reply: VER REP RSV ATYP BND.ADDR BND.PORT
	resp := []byte{
		socks5Version,
		reply,
		0x00, // Reserved
		addrType,
	}
	
	// Add bind address (0.0.0.0:0 for simplicity)
	if addrType == addrIPv4 {
		resp = append(resp, 0, 0, 0, 0) // 0.0.0.0
	} else if addrType == addrIPv6 {
		resp = append(resp, make([]byte, 16)...) // ::
	}
	
	// Add bind port (0)
	resp = append(resp, 0, 0)
	
	_, err := conn.Write(resp)
	return err
}

// relay relays data between two connections
func (s *Server) relay(client, dest net.Conn) {
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
