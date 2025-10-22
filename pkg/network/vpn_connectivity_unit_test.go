package network

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test buildConnectivityCheckScript
func TestVPNConnectivityChecker_BuildConnectivityCheckScript(t *testing.T) {
	checker := &VPNConnectivityChecker{}

	tests := []struct {
		name     string
		targetIP string
		contains []string
	}{
		{
			"Basic IPv4",
			"10.8.0.2",
			[]string{
				"#!/bin/bash",
				"TARGET_IP=\"10.8.0.2\"",
				"INTERFACE=\"wg0\"",
				"VPN Connectivity Check",
				"Target: $TARGET_IP",
				"ip link show $INTERFACE",
				"ping -c 10",
				"wg show",
				"nc -zv",
				"ip route",
			},
		},
		{
			"Different IP",
			"10.9.0.5",
			[]string{
				"TARGET_IP=\"10.9.0.5\"",
				"ping -c 10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := checker.buildConnectivityCheckScript(tt.targetIP)

			assert.NotEmpty(t, script)
			for _, expected := range tt.contains {
				assert.Contains(t, script, expected)
			}
		})
	}
}

// Test script structure
func TestVPNConnectivityChecker_ScriptStructure(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should start with shebang
	assert.True(t, strings.HasPrefix(script, "#!/bin/bash"))

	// Should contain critical sections
	sections := []string{
		"set -e",
		"VPN Connectivity Check",
		"Check if WireGuard interface exists",
		"Check if interface is up",
		"Perform ping test",
		"Check WireGuard peer status",
		"Test TCP connectivity",
		"Check routing table",
		"Check Complete",
	}

	for _, section := range sections {
		assert.Contains(t, script, section,
			"Script should contain section: %s", section)
	}
}

// Test script error handling
func TestVPNConnectivityChecker_ScriptErrorHandling(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	errorChecks := []string{
		"ERROR: WireGuard interface",
		"not found",
		"interface $INTERFACE is down",
		"exit 1",
	}

	for _, check := range errorChecks {
		assert.Contains(t, script, check)
	}
}

// Test script ping configuration
func TestVPNConnectivityChecker_ScriptPingConfig(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Ping should be: 10 packets, 2 second timeout, via wg0
	pingConfig := []string{
		"ping -c 10",
		"-W 2",
		"-I $INTERFACE",
		"$TARGET_IP",
		"/tmp/ping_result",
	}

	for _, cfg := range pingConfig {
		assert.Contains(t, script, cfg)
	}
}

// Test script output parsing markers
func TestVPNConnectivityChecker_ScriptOutputMarkers(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Check for output markers that parseConnectivityOutput will look for
	markers := []string{
		"PING_TEST:",
		"PING_STATUS:SUCCESS",
		"PING_STATUS:FAILED",
		"PACKET_LOSS:",
		"AVG_LATENCY:",
		"WIREGUARD_STATUS:",
		"HANDSHAKE:ACTIVE",
		"HANDSHAKE:NONE",
		"TCP_TEST:",
		"SSH_PORT:OPEN",
		"SSH_PORT:CLOSED",
		"ROUTING:",
	}

	for _, marker := range markers {
		assert.Contains(t, script, marker,
			"Script should contain output marker: %s", marker)
	}
}

// Test script WireGuard checks
func TestVPNConnectivityChecker_ScriptWireGuardChecks(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	wgCommands := []string{
		"ip link show $INTERFACE",
		"state UP",
		"wg show $INTERFACE peers",
		"wg show $INTERFACE latest-handshakes",
	}

	for _, cmd := range wgCommands {
		assert.Contains(t, script, cmd)
	}
}

// Test script TCP connectivity test
func TestVPNConnectivityChecker_ScriptTCPTest(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should test SSH port (22)
	assert.Contains(t, script, "nc -zv $TARGET_IP 22")
	assert.Contains(t, script, "timeout 2")
}

// Test script routing check
func TestVPNConnectivityChecker_ScriptRoutingCheck(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	assert.Contains(t, script, "ip route | grep $TARGET_IP")
	assert.Contains(t, script, "No specific route for $TARGET_IP")
}

// Test parseConnectivityOutput with successful ping
func TestVPNConnectivityChecker_ParseSuccessfulOutput(t *testing.T) {
	checker := &VPNConnectivityChecker{}

	output := `=== VPN Connectivity Check ===
Target: 10.8.0.2
PING_TEST:
PING_STATUS:SUCCESS
PACKET_LOSS:0
AVG_LATENCY:1.234
WIREGUARD_STATUS:
peers here
HANDSHAKE:ACTIVE
TCP_TEST:
SSH_PORT:OPEN
=== Check Complete ===`

	status := &ConnectionStatus{}
	checker.parseConnectivityOutput(output, status)

	assert.True(t, status.IsConnected, "Should be connected on SUCCESS")
}

// Test parseConnectivityOutput with failed ping
func TestVPNConnectivityChecker_ParseFailedOutput(t *testing.T) {
	checker := &VPNConnectivityChecker{}

	output := `=== VPN Connectivity Check ===
Target: 10.8.0.2
PING_TEST:
PING_STATUS:FAILED
PACKET_LOSS:100
=== Check Complete ===`

	status := &ConnectionStatus{}
	checker.parseConnectivityOutput(output, status)

	assert.False(t, status.IsConnected, "Should not be connected on FAILED")
}

// Test 50 connectivity check scenarios
func Test50ConnectivityCheckScenarios(t *testing.T) {
	checker := &VPNConnectivityChecker{}

	scenarios := []struct {
		ip        string
		ifaceName string
	}{
		{"10.8.0.1", "wg0"},
		{"10.8.0.2", "wg0"},
		{"10.9.0.1", "wg0"},
	}

	// Generate 47 more scenarios
	for i := 0; i < 47; i++ {
		ip := "10.8.0." + string(rune('1'+i%250))
		scenarios = append(scenarios, struct {
			ip        string
			ifaceName string
		}{ip, "wg0"})
	}

	for i, scenario := range scenarios {
		t.Run("Scenario_"+string(rune('A'+i%26))+string(rune('0'+i/26)), func(t *testing.T) {
			script := checker.buildConnectivityCheckScript(scenario.ip)

			// Validate script contains IP
			assert.Contains(t, script, scenario.ip)

			// Validate script contains interface
			assert.Contains(t, script, scenario.ifaceName)

			// Validate script structure
			assert.True(t, strings.HasPrefix(script, "#!/bin/bash"))
			assert.Contains(t, script, "set -e")
			assert.Contains(t, script, "VPN Connectivity Check")
		})
	}
}

// Test ConnectionStatus structure
func TestConnectionStatus_Structure(t *testing.T) {
	now := time.Now()
	status := &ConnectionStatus{
		TargetNode:  "node1",
		TargetIP:    "10.8.0.2",
		IsConnected: true,
		PacketLoss:  0.0,
		Latency:     1234 * time.Microsecond,
		LastCheck:   now,
		Error:       nil,
	}

	assert.True(t, status.IsConnected)
	assert.Equal(t, 0.0, status.PacketLoss)
	assert.Equal(t, 1234*time.Microsecond, status.Latency)
	assert.Equal(t, now, status.LastCheck)
	assert.Nil(t, status.Error)
}

// Test script with different target IPs
func TestVPNConnectivityChecker_DifferentTargetIPs(t *testing.T) {
	checker := &VPNConnectivityChecker{}

	ips := []string{
		"10.8.0.1",
		"10.8.0.100",
		"10.9.0.50",
		"172.16.0.10",
		"192.168.1.50",
	}

	for _, ip := range ips {
		t.Run("IP_"+ip, func(t *testing.T) {
			script := checker.buildConnectivityCheckScript(ip)
			assert.Contains(t, script, "TARGET_IP=\""+ip+"\"")
		})
	}
}

// Test script ping packet count
func TestVPNConnectivityChecker_PingPacketCount(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should send 10 packets
	assert.Contains(t, script, "ping -c 10")
}

// Test script ping timeout
func TestVPNConnectivityChecker_PingTimeout(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should have 2 second timeout per packet
	assert.Contains(t, script, "-W 2")
}

// Test script uses correct interface
func TestVPNConnectivityChecker_InterfaceName(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should use wg0 interface
	assert.Contains(t, script, "INTERFACE=\"wg0\"")
	assert.Contains(t, script, "-I $INTERFACE")
}

// Test script temporary file usage
func TestVPNConnectivityChecker_TempFileUsage(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should use /tmp/ping_result
	assert.Contains(t, script, "/tmp/ping_result")
	assert.Contains(t, script, "cat /tmp/ping_result")
}

// Test script packet loss parsing
func TestVPNConnectivityChecker_PacketLossParsing(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should extract packet loss percentage
	assert.Contains(t, script, "grep \"packet loss\"")
	assert.Contains(t, script, "PACKET_LOSS:")
	assert.Contains(t, script, ":-0}") // Default to 0
}

// Test script latency parsing
func TestVPNConnectivityChecker_LatencyParsing(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should extract average latency
	assert.Contains(t, script, "grep \"rtt min/avg/max\"")
	assert.Contains(t, script, "AVG_LATENCY:")
	assert.Contains(t, script, "cut -d'/' -f5")
}

// Test script handshake check
func TestVPNConnectivityChecker_HandshakeCheck(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should check for recent handshakes
	assert.Contains(t, script, "wg show $INTERFACE latest-handshakes")
	assert.Contains(t, script, "grep -q \"$TARGET_IP\"")
}

// Test script SSH port test timeout
func TestVPNConnectivityChecker_SSHPortTimeout(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// SSH port check should timeout after 2 seconds
	assert.Contains(t, script, "timeout 2 nc")
}

// Test script timestamp
func TestVPNConnectivityChecker_ScriptTimestamp(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should include timestamp
	assert.Contains(t, script, "Timestamp: $(date)")
}

// Test script header and footer
func TestVPNConnectivityChecker_ScriptHeaderFooter(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should have clear header and footer
	assert.Contains(t, script, "=== VPN Connectivity Check ===")
	assert.Contains(t, script, "=== Check Complete ===")
}

// Test script interface up check
func TestVPNConnectivityChecker_InterfaceUpCheck(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should check if interface is UP
	assert.Contains(t, script, "ip link show $INTERFACE | grep -q \"state UP\"")
	assert.Contains(t, script, "ERROR: WireGuard interface $INTERFACE is down")
}

// Test script peer listing
func TestVPNConnectivityChecker_PeerListing(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should list WireGuard peers
	assert.Contains(t, script, "wg show $INTERFACE peers")
	assert.Contains(t, script, "WIREGUARD_STATUS:")
}

// Test output parsing markers exist
func TestVPNConnectivityChecker_OutputMarkersExist(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// All status markers should exist in script
	requiredMarkers := []string{
		"PING_STATUS:SUCCESS",
		"PING_STATUS:FAILED",
		"PACKET_LOSS:",
		"AVG_LATENCY:",
		"HANDSHAKE:ACTIVE",
		"HANDSHAKE:NONE",
		"SSH_PORT:OPEN",
		"SSH_PORT:CLOSED",
	}

	for _, marker := range requiredMarkers {
		assert.Contains(t, script, marker,
			"Script must contain marker: %s", marker)
	}
}

// Test script handles no route case
func TestVPNConnectivityChecker_NoRouteHandling(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should handle case where no route exists
	assert.Contains(t, script, "|| echo \"No specific route for $TARGET_IP\"")
}

// Test contains helper function
func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{"Contains", "hello world", "world", true},
		{"Not contains", "hello world", "foo", false},
		{"Empty string", "", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.str, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test script bash safety
func TestVPNConnectivityChecker_BashSafety(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should use 'set -e' for error handling
	assert.Contains(t, script, "set -e")

	// Should redirect stderr appropriately
	assert.Contains(t, script, "&>/dev/null")
	assert.Contains(t, script, "2>&1")
}

// Test script network tools usage
func TestVPNConnectivityChecker_NetworkTools(t *testing.T) {
	checker := &VPNConnectivityChecker{}
	script := checker.buildConnectivityCheckScript("10.8.0.2")

	// Should use standard network tools
	tools := []string{
		"ip link",
		"ip route",
		"ping",
		"wg show",
		"nc -zv",
		"grep",
	}

	for _, tool := range tools {
		assert.Contains(t, script, tool)
	}
}
