package socks5

import (
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// newTestServer starts a SOCKS5 server on an ephemeral port and returns it
// together with the address it actually ended up listening on.
func newTestServer(t *testing.T, timeout time.Duration) (*Server, string) {
	t.Helper()

	srv := NewServer(Config{
		ListenAddr: "127.0.0.1:0",
		Timeout:    timeout,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	require.NoError(t, srv.Start())
	t.Cleanup(func() { _ = srv.Stop() })

	return srv, srv.listener.Addr().String()
}

// startEchoServer runs a TCP backend that echoes every byte back to its peer
// and returns its listen address.
func startEchoServer(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed during cleanup
			}

			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()

	return ln.Addr().String()
}

// socks5Greeting performs the version/method negotiation on conn.
func socks5Greeting(t *testing.T, conn net.Conn) {
	t.Helper()

	require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))

	// Greeting: version 5, one offered method, "no authentication".
	_, err := conn.Write([]byte{socks5Version, 0x01, authNone})
	require.NoError(t, err)

	reply := make([]byte, 2)
	_, err = io.ReadFull(conn, reply)
	require.NoError(t, err)
	require.Equal(t, byte(socks5Version), reply[0], "unexpected SOCKS version in greeting reply")
	require.Equal(t, byte(authNone), reply[1], "proxy rejected the no-auth method")
}

// readReply reads one complete SOCKS5 reply. The reply is self-describing: its
// ATYP octet declares how many BND.ADDR octets follow, so a reply whose ATYP
// disagrees with the number of octets actually written cannot be read to
// completion and fails here.
func readReply(t *testing.T, conn net.Conn) []byte {
	t.Helper()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))

	header := make([]byte, 4)
	_, err := io.ReadFull(conn, header)
	require.NoError(t, err, "server did not send a complete VER/REP/RSV/ATYP header")
	require.Equal(t, byte(socks5Version), header[0], "unexpected SOCKS version in reply")

	var addrLen int     // total BND.ADDR octets declared by ATYP
	var consumed []byte // address octets already read while determining addrLen

	switch header[3] {
	case addrIPv4:
		addrLen = net.IPv4len
	case addrIPv6:
		addrLen = net.IPv6len
	case addrDomain:
		lenBuf := make([]byte, 1)
		_, err = io.ReadFull(conn, lenBuf)
		require.NoError(t, err, "reply declares a domain ATYP but carries no length octet")
		consumed = lenBuf
		addrLen = 1 + int(lenBuf[0])
	default:
		t.Fatalf("reply carries unsupported ATYP %#x", header[3])
	}

	// Read whatever BND.ADDR octets remain, followed by BND.PORT.
	rest := make([]byte, addrLen-len(consumed)+2)
	_, err = io.ReadFull(conn, rest)
	require.NoError(t, err, "reply is truncated: BND.ADDR/BND.PORT incomplete for the declared ATYP")

	reply := append(append(header, consumed...), rest...)
	require.Len(t, reply, 4+addrLen+2, "reply length must match the declared ATYP")

	// Nothing may follow BND.PORT, otherwise the client stream is desynchronised
	// and every relayed payload would be misframed. Any read error is acceptable
	// here: it means either that no further octets arrived (deadline hit) or that
	// the server closed, which it does right after a failure reply. On Windows
	// that close surfaces as a connection reset rather than EOF, because the
	// server tears the connection down without draining the address octets it
	// deliberately chose not to parse.
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(150*time.Millisecond)))
	trailing := make([]byte, 1)
	if n, readErr := conn.Read(trailing); readErr == nil && n > 0 {
		t.Fatalf("reply carries %d unexpected trailing octet(s): %v", n, trailing[:n])
	}

	// Restore an unlimited deadline for the caller.
	require.NoError(t, conn.SetReadDeadline(time.Time{}))

	return reply
}

// socks5SendConnect sends a CONNECT request carrying the given ATYP and raw
// address octets and returns the server's reply.
func socks5SendConnect(t *testing.T, conn net.Conn, addrType byte, addr []byte, port uint16) []byte {
	t.Helper()

	req := []byte{socks5Version, cmdConnect, 0x00, addrType}
	req = append(req, addr...)
	req = binary.BigEndian.AppendUint16(req, port)

	require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))
	_, err := conn.Write(req)
	require.NoError(t, err)

	return readReply(t, conn)
}

// socks5Dial performs the full SOCKS5 handshake towards an IPv4 target and
// returns the connection once the proxy has accepted the tunnel.
func socks5Dial(t *testing.T, proxyAddr, target string) net.Conn {
	t.Helper()

	host, portStr, err := net.SplitHostPort(target)
	require.NoError(t, err)

	ip := net.ParseIP(host).To4()
	require.NotNil(t, ip, "test targets must be IPv4 literals, got %q", host)

	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	conn, err := net.Dial("tcp", proxyAddr)
	require.NoError(t, err)

	socks5Greeting(t, conn)

	reply := socks5SendConnect(t, conn, addrIPv4, ip, uint16(port))
	require.Equal(t, byte(replySuccess), reply[1], "CONNECT rejected with reply code %d", reply[1])

	return conn
}

// TestRelayOutlivesEstablishmentTimeout is the regression test for the bug
// where the handshake deadline stayed armed while payload was being relayed.
// Config.Timeout is documented as the time allowed to establish a connection,
// but because it was never lifted, every proxied connection was torn down
// Timeout after it was accepted - breaking video, SSH and large downloads.
func TestRelayOutlivesEstablishmentTimeout(t *testing.T) {
	const timeout = 300 * time.Millisecond

	echoAddr := startEchoServer(t)
	_, proxyAddr := newTestServer(t, timeout)

	conn := socks5Dial(t, proxyAddr, echoAddr)
	defer conn.Close()

	// Keep exchanging data for several times longer than Config.Timeout. With
	// the deadline still armed on the proxy side, the first exchange happening
	// after `timeout` would fail with an i/o timeout or a closed connection.
	until := time.Now().Add(4 * timeout)
	for i := 0; time.Now().Before(until); i++ {
		payload := []byte("ping-" + strconv.Itoa(i))

		// Deadlines below belong to the client side only and are deliberately
		// far longer than the proxy's establishment timeout.
		require.NoError(t, conn.SetDeadline(time.Now().Add(2*time.Second)))

		_, err := conn.Write(payload)
		require.NoErrorf(t, err, "write failed on exchange %d", i)

		got := make([]byte, len(payload))
		_, err = io.ReadFull(conn, got)
		require.NoErrorf(t, err, "read failed on exchange %d", i)
		require.Equalf(t, payload, got, "echo mismatch on exchange %d", i)

		time.Sleep(50 * time.Millisecond)
	}
}

// TestEstablishmentDeadlineStillEnforced makes sure that lifting the deadline
// for the relay phase did not remove the protection against peers that connect
// and then stall in the middle of the handshake.
func TestEstablishmentDeadlineStillEnforced(t *testing.T) {
	const timeout = 300 * time.Millisecond

	_, proxyAddr := newTestServer(t, timeout)

	conn, err := net.Dial("tcp", proxyAddr)
	require.NoError(t, err)
	defer conn.Close()

	// Send only a partial greeting, then go silent forever.
	_, err = conn.Write([]byte{socks5Version})
	require.NoError(t, err)

	// The server must drop the stalled connection once the establishment
	// deadline expires, which surfaces here as an error rather than a hang.
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))

	buf := make([]byte, 16)
	_, err = conn.Read(buf)
	require.Error(t, err, "server kept a stalled handshake open past Config.Timeout")
}

// TestStopInterruptsIdleRelay guards against a shutdown hang. Now that relays
// are no longer bounded by a deadline, Stop must still return when a peer keeps
// its side of an established tunnel open and completely idle.
func TestStopInterruptsIdleRelay(t *testing.T) {
	echoAddr := startEchoServer(t)
	srv, proxyAddr := newTestServer(t, 5*time.Second)

	conn := socks5Dial(t, proxyAddr, echoAddr)
	defer conn.Close()

	stopped := make(chan error, 1)
	go func() { stopped <- srv.Stop() }()

	select {
	case err := <-stopped:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Stop() did not return while an idle relay was active")
	}
}

// TestConnectViaDomainName covers the ATYP 0x03 path end to end. The reply used
// to be emitted without any BND.ADDR octets while still declaring ATYP 0x03, so
// a client parsed a length octet and a port out of a 6-octet reply and hit EOF:
// every domain-name CONNECT - the common case for browsers and curl - failed.
func TestConnectViaDomainName(t *testing.T) {
	echoAddr := startEchoServer(t)
	_, proxyAddr := newTestServer(t, 5*time.Second)

	_, portStr, err := net.SplitHostPort(echoAddr)
	require.NoError(t, err)

	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	conn, err := net.Dial("tcp", proxyAddr)
	require.NoError(t, err)
	defer conn.Close()

	socks5Greeting(t, conn)

	// The echo backend listens on 127.0.0.1, so use a name that resolves to it
	// from the hosts file rather than depending on external DNS.
	const host = "localhost"
	addr := append([]byte{byte(len(host))}, []byte(host)...)

	reply := socks5SendConnect(t, conn, addrDomain, addr, uint16(port))
	require.Equal(t, byte(replySuccess), reply[1], "domain CONNECT rejected with reply code %d", reply[1])
	require.Len(t, reply, 4+net.IPv4len+2, "reply is not a well-formed fixed-width response")
	require.Equal(t, byte(addrIPv4), reply[3], "ATYP must match the BND.ADDR width actually sent")

	// The tunnel must be usable. A truncated reply would have desynchronised the
	// stream, so this payload would be misframed or lost entirely.
	const payload = "hello-over-domain-name"

	require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))
	_, err = conn.Write([]byte(payload))
	require.NoError(t, err)

	got := make([]byte, len(payload))
	_, err = io.ReadFull(conn, got)
	require.NoError(t, err)
	require.Equal(t, payload, string(got))
}

// TestUnsupportedAddressTypeReply checks that an ATYP the proxy cannot parse is
// answered with the dedicated "address type not supported" code required by
// RFC 1928 section 6, instead of the generic failure it used to return.
func TestUnsupportedAddressTypeReply(t *testing.T) {
	const unsupportedAddrType = 0x07

	_, proxyAddr := newTestServer(t, 5*time.Second)

	conn, err := net.Dial("tcp", proxyAddr)
	require.NoError(t, err)
	defer conn.Close()

	socks5Greeting(t, conn)

	reply := socks5SendConnect(t, conn, unsupportedAddrType, []byte{0x01, 0x02, 0x03, 0x04}, 8080)
	require.Equal(t, byte(replyAddressNotSupported), reply[1],
		"unsupported ATYP must be answered with reply code %#x", byte(replyAddressNotSupported))
	require.Len(t, reply, 4+net.IPv4len+2, "failure reply must still be well-formed")
	require.Equal(t, byte(addrIPv4), reply[3], "ATYP must match the BND.ADDR width actually sent")
}

// TestSendReplyIsWellFormed pins the wire format of every reply the server can
// emit: the ATYP octet must always agree with the number of BND.ADDR octets
// that follow it, for successful and failing replies alike.
func TestSendReplyIsWellFormed(t *testing.T) {
	tests := []struct {
		name       string
		addrType   byte
		wantATYP   byte
		wantLength int
	}{
		{"IPv4", addrIPv4, addrIPv4, 4 + net.IPv4len + 2},
		{"domain", addrDomain, addrIPv4, 4 + net.IPv4len + 2},
		{"IPv6", addrIPv6, addrIPv6, 4 + net.IPv6len + 2},
		{"unsupported", 0x07, addrIPv4, 4 + net.IPv4len + 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := NewServer(Config{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})

			client, server := net.Pipe()
			defer client.Close()
			defer server.Close()

			require.NoError(t, client.SetReadDeadline(time.Now().Add(5*time.Second)))

			read := make(chan []byte, 1)
			go func() {
				buf := make([]byte, 64)
				n, err := client.Read(buf)
				if err != nil {
					t.Errorf("reading reply: %v", err)
					return
				}
				read <- buf[:n]
			}()

			require.NoError(t, srv.sendReply(server, replySuccess, tc.addrType))

			var resp []byte
			select {
			case resp = <-read:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for the reply")
			}

			require.Len(t, resp, tc.wantLength, "reply length must match the declared ATYP")
			require.Equal(t, byte(socks5Version), resp[0], "VER")
			require.Equal(t, byte(replySuccess), resp[1], "REP")
			require.Equal(t, byte(0x00), resp[2], "RSV must be zero")
			require.Equal(t, tc.wantATYP, resp[3], "ATYP must match the BND.ADDR width")
			require.Equal(t, []byte{0x00, 0x00}, resp[tc.wantLength-2:], "BND.PORT must be present and zero")
		})
	}
}
