package client

import (
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ConnectionKiller handles forceful termination of stale TCP connections
// to prevent conntrack table overflow on routers during failover.
type ConnectionKiller struct {
	logger *slog.Logger
}

// NewConnectionKiller creates a new connection killer
func NewConnectionKiller(logger *slog.Logger) *ConnectionKiller {
	return &ConnectionKiller{logger: logger}
}

// KillConnectionsToServer forcefully terminates all TCP connections to the specified server.
// This is critical during failover to prevent stale connections from accumulating
// in the router's conntrack table.
//
// Methods used (in order of preference):
// 1. ss -K (socket kill) - sends RST to terminate connections at socket level
// 2. conntrack -D - deletes conntrack entries directly
// 3. iptables REJECT - temporarily adds rules to reject with RST
func (ck *ConnectionKiller) KillConnectionsToServer(serverIP net.IP, serverPort string) error {
	if runtime.GOOS != "linux" {
		ck.logger.Debug("Connection killing is only supported on Linux", "os", runtime.GOOS)
		return nil
	}

	if serverIP == nil {
		return fmt.Errorf("server IP is nil")
	}

	ipStr := serverIP.String()
	ck.logger.Info("Killing stale connections to old VPN server",
		"server_ip", ipStr,
		"server_port", serverPort)

	var errors []string
	success := false

	// Method 1: Try ss -K (most reliable for killing TCP connections)
	if err := ck.killWithSS(ipStr, serverPort); err != nil {
		ck.logger.Debug("ss -K method failed or unavailable", "error", err)
		errors = append(errors, fmt.Sprintf("ss: %v", err))
	} else {
		success = true
	}

	// Method 2: Try conntrack -D (removes conntrack entries)
	if err := ck.killWithConntrack(ipStr, serverPort); err != nil {
		ck.logger.Debug("conntrack method failed or unavailable", "error", err)
		errors = append(errors, fmt.Sprintf("conntrack: %v", err))
	} else {
		success = true
	}

	// Method 3: Try iptables REJECT with RST (fallback - sends RST to existing connections)
	if err := ck.killWithIptablesReject(ipStr, serverPort); err != nil {
		ck.logger.Debug("iptables REJECT method failed or unavailable", "error", err)
		errors = append(errors, fmt.Sprintf("iptables: %v", err))
	} else {
		success = true
	}

	if success {
		ck.logger.Info("Stale connection cleanup completed", "server_ip", ipStr)
		return nil
	}

	errMsg := fmt.Sprintf("all connection cleanup methods failed: %s", strings.Join(errors, "; "))
	ck.logger.Warn(errMsg, "server_ip", ipStr)
	return fmt.Errorf(errMsg)
}

// killWithSS uses `ss -K` to kill TCP connections to the specified destination.
// ss -K is available on Linux with iproute2 and requires kernel support (CONFIG_INET_DIAG_DESTROY).
func (ck *ConnectionKiller) killWithSS(ipStr string, port string) error {
	// Check if ss is available
	if _, err := exec.LookPath("ss"); err != nil {
		return fmt.Errorf("ss not found: %w", err)
	}

	// Build destination filter
	dst := ipStr
	if port != "" {
		dst = net.JoinHostPort(ipStr, port)
	}

	// ss -K dst <ip>:<port> - kills all TCP connections to destination
	// -K: kill sockets, -t: TCP only
	cmd := exec.Command("ss", "-K", "-t", "dst", dst)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// ss -K returns error if no connections found, which is OK
		outputStr := string(output)
		if strings.Contains(outputStr, "No such file") ||
			strings.Contains(outputStr, "No sockets") ||
			strings.Contains(outputStr, "Cannot open") {
			ck.logger.Debug("ss -K: no connections found or already closed", "dst", dst)
			return nil
		}
		return fmt.Errorf("ss -K failed: %w, output: %s", err, outputStr)
	}

	ck.logger.Info("ss -K killed connections", "dst", dst, "output", strings.TrimSpace(string(output)))
	return nil
}

// killWithConntrack uses `conntrack -D` to delete connection tracking entries.
// This requires conntrack-tools to be installed.
func (ck *ConnectionKiller) killWithConntrack(ipStr string, port string) error {
	// Check if conntrack is available
	if _, err := exec.LookPath("conntrack"); err != nil {
		return fmt.Errorf("conntrack not found: %w", err)
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP: %s", ipStr)
	}

	var args []string
	args = append(args, "-D") // Delete

	if ip.To4() != nil {
		args = append(args, "-f", "ipv4")
	} else {
		args = append(args, "-f", "ipv6")
	}

	// Delete connections where destination is the server
	args = append(args, "--dport", port)
	args = append(args, "-p", "tcp")
	args = append(args, "--dst", ipStr)

	cmd := exec.Command("conntrack", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		// conntrack returns error if no entries found, which is OK
		if strings.Contains(outputStr, "0 entries") ||
			strings.Contains(outputStr, "No such file") {
			ck.logger.Debug("conntrack: no entries found or already deleted", "dst", ipStr)
			return nil
		}
		return fmt.Errorf("conntrack -D failed: %w, output: %s", err, outputStr)
	}

	ck.logger.Info("conntrack deleted entries", "dst", ipStr, "output", strings.TrimSpace(string(output)))
	return nil
}

// killWithIptablesReject adds temporary iptables REJECT rules to send RST packets
// to connections to the old server, then removes them after a short delay.
func (ck *ConnectionKiller) killWithIptablesReject(ipStr string, port string) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP: %s", ipStr)
	}

	var cmdName string
	if ip.To4() != nil {
		cmdName = "iptables"
	} else {
		cmdName = "ip6tables"
	}

	// Check if iptables is available
	if _, err := exec.LookPath(cmdName); err != nil {
		return fmt.Errorf("%s not found: %w", cmdName, err)
	}

	// Build rule to REJECT with tcp-reset
	// This will cause the kernel to send RST packets for all matching connections
	rule := []string{"-A", "OUTPUT", "-d", ipStr}
	if port != "" {
		rule = append(rule, "-p", "tcp", "--dport", port)
	} else {
		rule = append(rule, "-p", "tcp")
	}
	rule = append(rule, "-j", "REJECT", "--reject-with", "tcp-reset")

	// Add REJECT rule
	addCmd := exec.Command(cmdName, rule...)
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add REJECT rule: %w, output: %s", err, string(output))
	}

	ck.logger.Debug("Added iptables REJECT rule", "cmd", cmdName, "dst", ipStr, "port", port)

	// Schedule removal of the rule after a short delay
	// This allows existing connections to receive RST
	go func() {
		time.Sleep(3 * time.Second)

		// Remove the rule (change -A to -D)
		deleteRule := make([]string, len(rule))
		copy(deleteRule, rule)
		deleteRule[0] = "-D"

		delCmd := exec.Command(cmdName, deleteRule...)
		if output, err := delCmd.CombinedOutput(); err != nil {
			ck.logger.Warn("Failed to remove REJECT rule",
				"cmd", cmdName, "error", err, "output", string(output))
		} else {
			ck.logger.Debug("Removed iptables REJECT rule", "cmd", cmdName, "dst", ipStr)
		}
	}()

	return nil
}

// KillConnectionsToServerWithTimeout wraps KillConnectionsToServer with a timeout.
func (ck *ConnectionKiller) KillConnectionsToServerWithTimeout(serverIP net.IP, serverPort string, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- ck.KillConnectionsToServer(serverIP, serverPort)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		ck.logger.Warn("Connection cleanup timed out", "server_ip", serverIP, "timeout", timeout)
		return fmt.Errorf("connection cleanup timed out after %v", timeout)
	}
}