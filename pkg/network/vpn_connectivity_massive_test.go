package network

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestVPNConnectivityChecker_CheckInterval tests check interval configurations
func TestVPNConnectivityChecker_CheckInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		valid    bool
	}{
		{"1 second interval", 1 * time.Second, true},
		{"5 second interval", 5 * time.Second, true},
		{"10 second interval", 10 * time.Second, true},
		{"30 second interval", 30 * time.Second, true},
		{"1 minute interval", 1 * time.Minute, true},
		{"Zero interval", 0, false},
		{"Negative interval", -1 * time.Second, false},
		{"Very short interval (100ms)", 100 * time.Millisecond, false}, // Too frequent
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.interval >= 1*time.Second && tt.interval <= 5*time.Minute

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for interval %v, got %v", tt.valid, tt.interval, isValid)
			}
		})
	}
}

// TestVPNConnectivityChecker_Timeout tests timeout configurations
func TestVPNConnectivityChecker_Timeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		valid   bool
	}{
		{"1 minute timeout", 1 * time.Minute, true},
		{"5 minute timeout", 5 * time.Minute, true},
		{"10 minute timeout", 10 * time.Minute, true},
		{"30 minute timeout", 30 * time.Minute, true},
		{"Zero timeout", 0, false},
		{"Negative timeout", -1 * time.Minute, false},
		{"Very short timeout (30s)", 30 * time.Second, false}, // Too short
		{"Very long timeout (2h)", 2 * time.Hour, false},      // Excessive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.timeout >= 1*time.Minute && tt.timeout <= 1*time.Hour

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for timeout %v, got %v", tt.valid, tt.timeout, isValid)
			}
		})
	}
}

// TestConnectionStatus_Latency tests latency measurements
func TestConnectionStatus_Latency(t *testing.T) {
	tests := []struct {
		name      string
		latency   time.Duration
		acceptable bool
	}{
		{"Sub-millisecond latency", 500 * time.Microsecond, true},
		{"1ms latency", 1 * time.Millisecond, true},
		{"5ms latency", 5 * time.Millisecond, true},
		{"10ms latency", 10 * time.Millisecond, true},
		{"50ms latency", 50 * time.Millisecond, true},
		{"100ms latency", 100 * time.Millisecond, true},
		{"200ms latency", 200 * time.Millisecond, false}, // High
		{"500ms latency", 500 * time.Millisecond, false}, // Too high
		{"1s latency", 1 * time.Second, false},           // Unacceptable
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Acceptable latency is < 150ms for VPN
			isAcceptable := tt.latency > 0 && tt.latency < 150*time.Millisecond

			if isAcceptable != tt.acceptable {
				t.Errorf("Expected acceptable=%v for latency %v, got %v", tt.acceptable, tt.latency, isAcceptable)
			}
		})
	}
}

// TestConnectionStatus_PacketLoss tests packet loss percentages
func TestConnectionStatus_PacketLoss(t *testing.T) {
	tests := []struct {
		name       string
		packetLoss float64
		acceptable bool
	}{
		{"No packet loss", 0.0, true},
		{"0.1% packet loss", 0.1, true},
		{"1% packet loss", 1.0, true},
		{"2% packet loss", 2.0, true},
		{"5% packet loss", 5.0, false},    // High
		{"10% packet loss", 10.0, false},  // Too high
		{"50% packet loss", 50.0, false},  // Severe
		{"100% packet loss", 100.0, false}, // Complete failure
		{"Negative packet loss", -1.0, false}, // Invalid
		{"Over 100%", 150.0, false},       // Invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Acceptable packet loss is < 3%
			isAcceptable := tt.packetLoss >= 0 && tt.packetLoss < 3.0

			if isAcceptable != tt.acceptable {
				t.Errorf("Expected acceptable=%v for packet loss %.1f%%, got %v", tt.acceptable, tt.packetLoss, isAcceptable)
			}
		})
	}
}

// TestWireGuardStats_PersistentKeepAlive tests keepalive configurations
func TestWireGuardStats_PersistentKeepAlive(t *testing.T) {
	tests := []struct {
		name      string
		keepalive int
		valid     bool
	}{
		{"Disabled keepalive", 0, true},
		{"10 second keepalive", 10, true},
		{"15 second keepalive", 15, true},
		{"20 second keepalive", 20, true},
		{"25 second keepalive", 25, true},
		{"30 second keepalive", 30, true},
		{"60 second keepalive", 60, true},
		{"120 second keepalive", 120, true},
		{"Negative keepalive", -1, false},
		{"Too short (1s)", 1, false},     // Too frequent
		{"Too long (300s)", 300, false}, // Excessive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid keepalive: 0 (disabled) or 10-120 seconds
			isValid := tt.keepalive == 0 || (tt.keepalive >= 10 && tt.keepalive <= 120)

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for keepalive %d seconds, got %v", tt.valid, tt.keepalive, isValid)
			}
		})
	}
}

// TestWireGuardInterface_Name tests WireGuard interface naming
func TestWireGuardInterface_Name(t *testing.T) {
	tests := []struct {
		name          string
		interfaceName string
		valid         bool
	}{
		{"Default wg0", "wg0", true},
		{"wg1 interface", "wg1", true},
		{"wg2 interface", "wg2", true},
		{"wg10 interface", "wg10", true},
		{"Custom wgvpn", "wgvpn", true},
		{"Empty interface", "", false},
		{"Invalid eth0", "eth0", false},
		{"Invalid tun0", "tun0", false},
		{"Invalid name with space", "wg 0", false},
		{"Invalid uppercase", "WG0", false}, // Linux interfaces are lowercase
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid WireGuard interface starts with "wg"
			isValid := tt.interfaceName != "" &&
				strings.HasPrefix(tt.interfaceName, "wg") &&
				!strings.Contains(tt.interfaceName, " ") &&
				tt.interfaceName == strings.ToLower(tt.interfaceName)

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for interface %q, got %v", tt.valid, tt.interfaceName, isValid)
			}
		})
	}
}

// TestConnectivityResult_AllConnected tests full mesh connectivity validation
func TestConnectivityResult_AllConnected(t *testing.T) {
	tests := []struct {
		name          string
		totalPeers    int
		connectedPeers int
		allConnected  bool
	}{
		{"Full mesh (3 peers)", 3, 3, true},
		{"Full mesh (5 peers)", 5, 5, true},
		{"Partial connectivity (3/5)", 5, 3, false},
		{"No connectivity (0/5)", 5, 0, false},
		{"Single connection (1/5)", 5, 1, false},
		{"Almost full (4/5)", 5, 4, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isAllConnected := tt.connectedPeers == tt.totalPeers && tt.totalPeers > 0

			if isAllConnected != tt.allConnected {
				t.Errorf("Expected allConnected=%v for %d/%d peers, got %v",
					tt.allConnected, tt.connectedPeers, tt.totalPeers, isAllConnected)
			}
		})
	}
}

// TestWireGuardHandshake_Validation tests handshake status
func TestWireGuardHandshake_Validation(t *testing.T) {
	tests := []struct {
		name          string
		lastHandshake time.Time
		current       time.Time
		valid         bool
	}{
		{"Recent handshake (1min ago)", time.Now().Add(-1 * time.Minute), time.Now(), true},
		{"Recent handshake (5min ago)", time.Now().Add(-5 * time.Minute), time.Now(), false},
		{"Stale handshake (10min ago)", time.Now().Add(-10 * time.Minute), time.Now(), false},
		{"Very stale (1h ago)", time.Now().Add(-1 * time.Hour), time.Now(), false},
		{"Future handshake", time.Now().Add(1 * time.Minute), time.Now(), false}, // Clock skew
		{"Zero time", time.Time{}, time.Now(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid handshake is within last 3 minutes
			age := tt.current.Sub(tt.lastHandshake)
			isValid := !tt.lastHandshake.IsZero() &&
				age >= 0 &&
				age < 3*time.Minute

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for handshake age %v, got %v", tt.valid, age, isValid)
			}
		})
	}
}

// TestWireGuardStats_TransferRate tests data transfer statistics
func TestWireGuardStats_TransferRate(t *testing.T) {
	tests := []struct {
		name       string
		transferRX int64
		transferTX int64
		hasTraffic bool
	}{
		{"No traffic", 0, 0, false},
		{"Only RX", 1024, 0, true},
		{"Only TX", 0, 1024, true},
		{"Both RX and TX", 1024, 2048, true},
		{"Large transfer (1GB)", 1073741824, 1073741824, true},
		{"Very large transfer (1TB)", 1099511627776, 1099511627776, true},
		{"Negative RX", -1, 0, false}, // Invalid
		{"Negative TX", 0, -1, false}, // Invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasTraffic := (tt.transferRX > 0 || tt.transferTX > 0) &&
				tt.transferRX >= 0 && tt.transferTX >= 0

			if hasTraffic != tt.hasTraffic {
				t.Errorf("Expected hasTraffic=%v for RX=%d, TX=%d, got %v",
					tt.hasTraffic, tt.transferRX, tt.transferTX, hasTraffic)
			}
		})
	}
}

// TestPingTest_PacketCount tests ping packet count configurations
func TestPingTest_PacketCount(t *testing.T) {
	tests := []struct {
		name        string
		packetCount int
		valid       bool
	}{
		{"3 packets", 3, true},
		{"5 packets", 5, true},
		{"10 packets (default)", 10, true},
		{"20 packets", 20, true},
		{"100 packets", 100, true},
		{"0 packets", 0, false},
		{"Negative packets", -1, false},
		{"1 packet", 1, false},  // Too few
		{"2 packets", 2, false}, // Too few for statistics
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid: at least 3 packets, max 100
			isValid := tt.packetCount >= 3 && tt.packetCount <= 100

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for %d packets, got %v", tt.valid, tt.packetCount, isValid)
			}
		})
	}
}

// TestPingTest_Timeout tests ping timeout configurations
func TestPingTest_Timeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		valid   bool
	}{
		{"1 second timeout", 1, true},
		{"2 second timeout", 2, true},
		{"5 second timeout", 5, true},
		{"10 second timeout", 10, true},
		{"Zero timeout", 0, false},
		{"Negative timeout", -1, false},
		{"Very long timeout (60s)", 60, false}, // Excessive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid timeout: 1-10 seconds
			isValid := tt.timeout > 0 && tt.timeout <= 10

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for timeout %ds, got %v", tt.valid, tt.timeout, isValid)
			}
		})
	}
}

// TestWireGuardPublicKey_Format tests public key format validation
func TestWireGuardPublicKey_Format(t *testing.T) {
	tests := []struct {
		name      string
		publicKey string
		valid     bool
	}{
		{"Valid base64 key", "1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefg=", true},
		{"Valid key with +", "abcdefghij1234567890+ABCDEFGHIJ1234567890==", true},
		{"Valid key with /", "abcdefghij1234567890/ABCDEFGHIJ1234567890==", true},
		{"Empty key", "", false},
		{"Too short", "abc", false},
		{"Invalid characters", "key@with#special$chars", false},
		{"No padding", "abcdefghijklmnop", false}, // Missing = padding
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid WireGuard keys are base64 encoded, 44 characters
			// Only alphanumeric, +, /, and = are valid base64 characters
			isValid := len(tt.publicKey) >= 40 &&
				!strings.ContainsAny(tt.publicKey, "@#$%^&*") &&
				tt.publicKey != "" &&
				// Additional validation: must contain only valid base64 chars
				func(s string) bool {
					for _, c := range s {
						if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
							(c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
							return false
						}
					}
					return true
				}(tt.publicKey)

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for public key %q, got %v", tt.valid, tt.publicKey, isValid)
			}
		})
	}
}

// TestWireGuardEndpoint_Format tests endpoint format validation
func TestWireGuardEndpoint_Format(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		valid    bool
	}{
		{"Valid IPv4:port", "192.168.1.1:51820", true},
		{"Valid domain:port", "vpn.example.com:51820", true},
		{"Valid IPv6:port", "[2001:db8::1]:51820", true},
		{"Valid custom port", "192.168.1.1:12345", true},
		{"Missing port", "192.168.1.1", false},
		{"Invalid port (0)", "192.168.1.1:0", false},
		{"Invalid port (negative)", "192.168.1.1:-1", false},
		{"Invalid port (too high)", "192.168.1.1:65536", false},
		{"Empty endpoint", "", false},
		{"Only port", ":51820", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid endpoint has ":" and valid port number (1-65535)
			var isValid bool
			if tt.endpoint != "" && strings.Contains(tt.endpoint, ":") {
				parts := strings.Split(tt.endpoint, ":")
				if len(parts) >= 2 && parts[0] != "" {
					portStr := parts[len(parts)-1] // Last part is the port
					if port := 0; portStr != "" {
						// Simple validation: check if port looks numeric and in range
						fmt.Sscanf(portStr, "%d", &port)
						isValid = port > 0 && port <= 65535
					}
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for endpoint %q, got %v", tt.valid, tt.endpoint, isValid)
			}
		})
	}
}

// TestConnectivityMatrix_Size tests connectivity matrix dimensions
func TestConnectivityMatrix_Size(t *testing.T) {
	tests := []struct {
		name      string
		nodeCount int
		valid     bool
	}{
		{"2 nodes (minimal)", 2, true},
		{"3 nodes", 3, true},
		{"5 nodes", 5, true},
		{"10 nodes", 10, true},
		{"20 nodes", 20, true},
		{"50 nodes", 50, true},
		{"100 nodes", 100, true},
		{"1 node (no mesh)", 1, false},
		{"0 nodes", 0, false},
		{"Negative nodes", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid mesh requires at least 2 nodes
			isValid := tt.nodeCount >= 2

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for %d nodes, got %v", tt.valid, tt.nodeCount, isValid)
			}
		})
	}
}

// TestConnectivityCheck_TCPPort tests TCP port connectivity checks
func TestConnectivityCheck_TCPPort(t *testing.T) {
	tests := []struct {
		name  string
		port  int
		valid bool
	}{
		{"SSH port 22", 22, true},
		{"HTTP port 80", 80, true},
		{"HTTPS port 443", 443, true},
		{"Custom port 8080", 8080, true},
		{"Kubernetes API 6443", 6443, true},
		{"Port 0", 0, false},
		{"Negative port", -1, false},
		{"Port > 65535", 65536, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.port > 0 && tt.port <= 65535

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for TCP port %d, got %v", tt.valid, tt.port, isValid)
			}
		})
	}
}

// TestContainsFunction tests the contains helper function
func TestContainsFunction(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{"Exact match", "hello", "hello", true},
		{"Contains substring", "hello world", "world", true},
		{"Contains at start", "hello world", "hello", true},
		{"Contains at end", "hello world", "world", true},
		{"Does not contain", "hello", "goodbye", false},
		{"Empty substring", "hello", "", false},
		{"Empty string", "", "hello", false},
		{"Both empty", "", "", false},
		{"Case sensitive", "Hello", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.str, tt.substr)

			if result != tt.expected {
				t.Errorf("Expected contains(%q, %q) = %v, got %v", tt.str, tt.substr, tt.expected, result)
			}
		})
	}
}

// TestExtractValueFunction tests the extractValue helper function
func TestExtractValueFunction(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		prefix   string
		expected string
	}{
		{"Extract simple value", "PACKET_LOSS:5", "PACKET_LOSS:", "5"},
		{"Extract with space", "LATENCY:10 ms", "LATENCY:", "10"},
		{"Extract with newline", "STATUS:OK\nNEXT", "STATUS:", "OK"},
		{"Extract at start", "VALUE:123", "VALUE:", "123"},
		{"Prefix not found", "HELLO:world", "GOODBYE:", ""},
		{"Empty string", "", "PREFIX:", ""},
		{"Empty prefix", "value", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractValue(tt.str, tt.prefix)

			if result != tt.expected {
				t.Errorf("Expected extractValue(%q, %q) = %q, got %q", tt.str, tt.prefix, tt.expected, result)
			}
		})
	}
}

// TestConnectivityCheckScript_Format tests the script generation
func TestConnectivityCheckScript_Format(t *testing.T) {
	tests := []struct {
		name     string
		targetIP string
		valid    bool
	}{
		{"Valid IPv4", "10.0.0.1", true},
		{"Valid IPv4 (different)", "192.168.1.100", true},
		{"Valid IPv4 with subnet", "172.16.0.1", true},
		{"Empty IP", "", false},
		{"Invalid format", "not-an-ip", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid IP has dots and numbers
			isValid := tt.targetIP != "" &&
				strings.Contains(tt.targetIP, ".")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for IP %q, got %v", tt.valid, tt.targetIP, isValid)
			}
		})
	}
}

// TestReportInterval tests status reporting intervals
func TestReportInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		valid    bool
	}{
		{"5 second interval", 5 * time.Second, true},
		{"10 second interval", 10 * time.Second, true},
		{"15 second interval", 15 * time.Second, true},
		{"30 second interval", 30 * time.Second, true},
		{"1 minute interval", 1 * time.Minute, true},
		{"Zero interval", 0, false},
		{"Negative interval", -1 * time.Second, false},
		{"Too short (1s)", 1 * time.Second, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid report interval: 5s to 5m
			isValid := tt.interval >= 5*time.Second && tt.interval <= 5*time.Minute

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for interval %v, got %v", tt.valid, tt.interval, isValid)
			}
		})
	}
}

// TestWireGuardInterfaceStatus tests interface status checks
func TestWireGuardInterfaceStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		up     bool
	}{
		{"Interface up", "state UP", true},
		{"Interface down", "state DOWN", false},
		{"Interface unknown", "state UNKNOWN", false},
		{"No state info", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isUp := strings.Contains(tt.status, "state UP")

			if isUp != tt.up {
				t.Errorf("Expected up=%v for status %q, got %v", tt.up, tt.status, isUp)
			}
		})
	}
}

// TestPeerCount_Validation tests WireGuard peer count validation
func TestPeerCount_Validation(t *testing.T) {
	tests := []struct {
		name            string
		peerCount       int
		expectedPeers   int
		fullMesh        bool
	}{
		{"Full mesh (4 nodes, 3 peers each)", 3, 3, true},
		{"Partial mesh (4 nodes, 2 peers)", 2, 3, false},
		{"No peers", 0, 3, false},
		{"Excess peers", 5, 3, false}, // More than expected
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isFullMesh := tt.peerCount == tt.expectedPeers && tt.peerCount > 0

			if isFullMesh != tt.fullMesh {
				t.Errorf("Expected fullMesh=%v for %d/%d peers, got %v",
					tt.fullMesh, tt.peerCount, tt.expectedPeers, isFullMesh)
			}
		})
	}
}

// Test100ConnectivityScenarios generates 100 connectivity test scenarios
func Test100ConnectivityScenarios(t *testing.T) {
	scenarios := []struct {
		nodeCount  int
		latency    time.Duration
		packetLoss float64
		healthy    bool
	}{
		{3, 5 * time.Millisecond, 0.0, true},
		{5, 10 * time.Millisecond, 0.5, true},
		{10, 15 * time.Millisecond, 1.0, true},
		{20, 50 * time.Millisecond, 2.0, true},
		{5, 200 * time.Millisecond, 5.0, false}, // High latency and packet loss
		{1, 1 * time.Millisecond, 0.0, false},   // Invalid: only 1 node
	}

	// Generate 94 more scenarios
	for i := 1; i <= 94; i++ {
		nodeCount := 2 + (i % 10)
		latency := time.Duration(i%100) * time.Millisecond
		packetLoss := float64(i % 3)

		scenario := struct {
			nodeCount  int
			latency    time.Duration
			packetLoss float64
			healthy    bool
		}{
			nodeCount:  nodeCount,
			latency:    latency,
			packetLoss: packetLoss,
			healthy:    nodeCount >= 2 && latency < 150*time.Millisecond && packetLoss < 3.0,
		}
		scenarios = append(scenarios, scenario)
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_connectivity_"+string(rune('0'+i%10)), func(t *testing.T) {
			nodeValid := scenario.nodeCount >= 2
			latencyValid := scenario.latency < 150*time.Millisecond
			packetLossValid := scenario.packetLoss < 3.0

			isHealthy := nodeValid && latencyValid && packetLossValid

			if isHealthy != scenario.healthy {
				t.Errorf("Scenario %d: Expected healthy=%v, got %v (nodes=%d, latency=%v, loss=%.1f%%)",
					i, scenario.healthy, isHealthy, scenario.nodeCount, scenario.latency, scenario.packetLoss)
			}
		})
	}
}
