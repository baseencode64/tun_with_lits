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

// socks5Dial performs the SOCKS5 greeting and issues a CONNECT request for
// target, returning the connection once the proxy has accepted the tunnel.
func socks5Dial(t *testing.T, proxyAddr, target string) net.Conn {
	t.Helper()

	conn, err := net.Dial("tcp", proxyAddr)
	require.NoError(t, err)

	// Greeting: version 5, one offered method, "no authentication".
	_, err = conn.Write([]byte{socks5Version, 0x01, authNone})
	require.NoError(t, err)

	greeting := make([]byte, 2)
	_, err = io.ReadFull(conn, greeting)
	require.NoError(t, err)
	require.Equal(t, byte(socks5Version), greeting[0], "unexpected SOCKS version in greeting reply")
	require.Equal(t, byte(authNone), greeting[1], "proxy rejected the no-auth method")

	host, portStr, err := net.SplitHostPort(target)
	require.NoError(t, err)

	ip := net.ParseIP(host).To4()
	require.NotNil(t, ip, "test targets must be IPv4 literals, got %q", host)

	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	// CONNECT request carrying an IPv4 destination.
	req := []byte{socks5Version, cmdConnect, 0x00, addrIPv4}
	req = append(req, ip...)
	req = binary.BigEndian.AppendUint16(req, uint16(port))

	_, err = conn.Write(req)
	require.NoError(t, err)

	reply := make([]byte, 10)
	_, err = io.ReadFull(conn, reply)
	require.NoError(t, err)
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
