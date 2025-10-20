package network

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"sloth-kubernetes/pkg/providers"
)

// VPNConnectivityChecker validates VPN connectivity between all nodes
type VPNConnectivityChecker struct {
	ctx           *pulumi.Context
	nodes         []*providers.NodeOutput
	sshKeyPath    string
	results       map[string]*ConnectivityResult
	mu            sync.RWMutex
	checkInterval time.Duration
	timeout       time.Duration
}

// ConnectivityResult represents the connectivity status from one node to all others
type ConnectivityResult struct {
	SourceNode   string
	Timestamp    time.Time
	Connections  map[string]*ConnectionStatus
	AllConnected bool
	Error        error
}

// ConnectionStatus represents a single connection status
type ConnectionStatus struct {
	TargetNode     string
	TargetIP       string
	IsConnected    bool
	Latency        time.Duration
	PacketLoss     float64
	LastCheck      time.Time
	Error          error
	WireGuardStats *WireGuardStats
}

// WireGuardStats contains WireGuard interface statistics
type WireGuardStats struct {
	Interface           string
	PublicKey           string
	Endpoint            string
	LastHandshake       time.Time
	TransferRX          int64
	TransferTX          int64
	PersistentKeepAlive int
}

// NewVPNConnectivityChecker creates a new VPN connectivity checker
func NewVPNConnectivityChecker(ctx *pulumi.Context) *VPNConnectivityChecker {
	return &VPNConnectivityChecker{
		ctx:           ctx,
		nodes:         make([]*providers.NodeOutput, 0),
		results:       make(map[string]*ConnectivityResult),
		checkInterval: 5 * time.Second,
		timeout:       5 * time.Minute,
	}
}

// AddNode adds a node to be monitored
func (v *VPNConnectivityChecker) AddNode(node *providers.NodeOutput) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.nodes = append(v.nodes, node)
	v.results[node.Name] = &ConnectivityResult{
		SourceNode:  node.Name,
		Connections: make(map[string]*ConnectionStatus),
	}
}

// SetSSHKeyPath sets the SSH key path for connections
func (v *VPNConnectivityChecker) SetSSHKeyPath(path string) {
	v.sshKeyPath = path
}

// VerifyFullMeshConnectivity verifies that all nodes can reach each other via WireGuard
func (v *VPNConnectivityChecker) VerifyFullMeshConnectivity() error {
	v.ctx.Log.Info("Starting VPN full mesh connectivity verification", nil)

	ctx, cancel := context.WithTimeout(context.Background(), v.timeout)
	defer cancel()

	// Channel to collect results
	resultChan := make(chan *ConnectivityResult, len(v.nodes))
	errorChan := make(chan error, len(v.nodes))

	// Create a WaitGroup to track all goroutines
	var wg sync.WaitGroup

	// Launch a goroutine for each node to check connectivity to all other nodes
	for _, sourceNode := range v.nodes {
		wg.Add(1)
		go v.checkNodeConnectivity(ctx, &wg, sourceNode, resultChan, errorChan)
	}

	// Monitor goroutine to wait for all checks to complete
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Status reporter goroutine
	go v.reportConnectivityStatus(ctx)

	// Collect results and check for full connectivity
	allConnected := true
	failedConnections := []string{}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for VPN connectivity verification")

		case err := <-errorChan:
			if err != nil {
				v.ctx.Log.Warn("Connectivity check error", nil)
			}

		case result, ok := <-resultChan:
			if !ok {
				// All results collected
				if !allConnected {
					return fmt.Errorf("VPN connectivity verification failed: %v", failedConnections)
				}

				v.ctx.Log.Info("VPN full mesh connectivity verified successfully!", nil)
				return nil
			}

			// Update results
			v.mu.Lock()
			v.results[result.SourceNode] = result
			v.mu.Unlock()

			// Check if this node has full connectivity
			if !result.AllConnected {
				allConnected = false
				for targetNode, conn := range result.Connections {
					if !conn.IsConnected {
						failedConnections = append(failedConnections,
							fmt.Sprintf("%s -> %s", result.SourceNode, targetNode))
					}
				}
			}

			// Log successful connections
			v.ctx.Log.Info("Node connectivity status", nil)
		}
	}
}

// checkNodeConnectivity runs connectivity checks from one node to all others
func (v *VPNConnectivityChecker) checkNodeConnectivity(
	ctx context.Context,
	wg *sync.WaitGroup,
	sourceNode *providers.NodeOutput,
	resultChan chan<- *ConnectivityResult,
	errorChan chan<- error,
) {
	defer wg.Done()

	ticker := time.NewTicker(v.checkInterval)
	defer ticker.Stop()

	result := &ConnectivityResult{
		SourceNode:  sourceNode.Name,
		Connections: make(map[string]*ConnectionStatus),
		Timestamp:   time.Now(),
	}

	// Keep checking until all connections are established or timeout
	for {
		select {
		case <-ctx.Done():
			resultChan <- result
			return

		case <-ticker.C:
			allConnected := true

			// Check connectivity to all other nodes
			for _, targetNode := range v.nodes {
				// Skip self
				if targetNode.Name == sourceNode.Name {
					continue
				}

				// Check if already connected
				if conn, exists := result.Connections[targetNode.Name]; exists && conn.IsConnected {
					// Refresh stats but don't recheck connectivity
					v.updateWireGuardStats(sourceNode, targetNode, conn)
					continue
				}

				// Perform connectivity check
				conn := v.performConnectivityCheck(sourceNode, targetNode)
				result.Connections[targetNode.Name] = conn

				if !conn.IsConnected {
					allConnected = false
				}
			}

			result.AllConnected = allConnected
			result.Timestamp = time.Now()

			// If all connections are established, send result and exit
			if allConnected {
				v.ctx.Log.Info("Node has full connectivity", nil)
				resultChan <- result
				return
			}
		}
	}
}

// performConnectivityCheck checks connectivity between two nodes
func (v *VPNConnectivityChecker) performConnectivityCheck(source, target *providers.NodeOutput) *ConnectionStatus {
	status := &ConnectionStatus{
		TargetNode: target.Name,
		TargetIP:   target.WireGuardIP,
		LastCheck:  time.Now(),
	}

	// Build connectivity check script
	checkScript := v.buildConnectivityCheckScript(target.WireGuardIP)

	// Execute the check via SSH on the source node
	cmd, err := remote.NewCommand(v.ctx,
		fmt.Sprintf("vpn-check-%s-to-%s-%d", source.Name, target.Name, time.Now().Unix()),
		&remote.CommandArgs{
			Connection: &remote.ConnectionArgs{
				Host:       source.PublicIP,
				Port:       pulumi.Float64(22),
				User:       pulumi.String(source.SSHUser),
				PrivateKey: pulumi.String(v.getSSHPrivateKey()),
			},
			Create: pulumi.String(checkScript),
		},
		pulumi.IgnoreChanges([]string{"create"}),
	)

	if err != nil {
		status.Error = err
		return status
	}

	// Parse the output
	cmd.Stdout.ApplyT(func(output string) string {
		v.parseConnectivityOutput(output, status)
		return output
	})

	return status
}

// buildConnectivityCheckScript creates the script to check connectivity
func (v *VPNConnectivityChecker) buildConnectivityCheckScript(targetIP string) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

TARGET_IP="%s"
INTERFACE="wg0"

echo "=== VPN Connectivity Check ==="
echo "Target: $TARGET_IP"
echo "Timestamp: $(date)"
echo ""

# Check if WireGuard interface exists
if ! ip link show $INTERFACE &>/dev/null; then
    echo "ERROR: WireGuard interface $INTERFACE not found"
    exit 1
fi

# Check if interface is up
if ! ip link show $INTERFACE | grep -q "state UP"; then
    echo "ERROR: WireGuard interface $INTERFACE is down"
    exit 1
fi

# Perform ping test (10 packets)
echo "PING_TEST:"
if ping -c 10 -W 2 -I $INTERFACE $TARGET_IP > /tmp/ping_result 2>&1; then
    echo "PING_STATUS:SUCCESS"

    # Extract statistics
    PACKET_LOSS=$(grep "packet loss" /tmp/ping_result | sed 's/.*\([0-9]\+\)%%.*/\1/')
    AVG_LATENCY=$(grep "rtt min/avg/max" /tmp/ping_result | cut -d'/' -f5)

    echo "PACKET_LOSS:${PACKET_LOSS:-0}"
    echo "AVG_LATENCY:${AVG_LATENCY:-0}"
else
    echo "PING_STATUS:FAILED"
    echo "PACKET_LOSS:100"
    cat /tmp/ping_result
fi

# Check WireGuard peer status
echo ""
echo "WIREGUARD_STATUS:"
wg show $INTERFACE peers | while read -r line; do
    echo "$line"
done

# Check if we have a handshake with any peer
if wg show $INTERFACE latest-handshakes | grep -q "$TARGET_IP"; then
    echo "HANDSHAKE:ACTIVE"
else
    echo "HANDSHAKE:NONE"
fi

# Test TCP connectivity (optional - for services)
echo ""
echo "TCP_TEST:"
if timeout 2 nc -zv $TARGET_IP 22 2>&1; then
    echo "SSH_PORT:OPEN"
else
    echo "SSH_PORT:CLOSED"
fi

# Check routing table
echo ""
echo "ROUTING:"
ip route | grep $TARGET_IP || echo "No specific route for $TARGET_IP"

echo ""
echo "=== Check Complete ==="
`, targetIP)
}

// parseConnectivityOutput parses the output from connectivity check
func (v *VPNConnectivityChecker) parseConnectivityOutput(output string, status *ConnectionStatus) {
	// Parse ping status
	if contains(output, "PING_STATUS:SUCCESS") {
		status.IsConnected = true

		// Parse packet loss
		if match := extractValue(output, "PACKET_LOSS:"); match != "" {
			fmt.Sscanf(match, "%f", &status.PacketLoss)
		}

		// Parse latency
		if match := extractValue(output, "AVG_LATENCY:"); match != "" {
			if duration, err := time.ParseDuration(match + "ms"); err == nil {
				status.Latency = duration
			}
		}
	} else {
		status.IsConnected = false
		status.PacketLoss = 100
	}

	// Parse WireGuard stats
	if contains(output, "HANDSHAKE:ACTIVE") {
		status.WireGuardStats = &WireGuardStats{
			Interface: "wg0",
		}
	}
}

// updateWireGuardStats updates WireGuard statistics for an existing connection
func (v *VPNConnectivityChecker) updateWireGuardStats(source, target *providers.NodeOutput, conn *ConnectionStatus) {
	// Build a lightweight stats update script
	statsScript := fmt.Sprintf(`#!/bin/bash
wg show wg0 endpoints | grep -A 5 "%s" || echo "No peer info"
`, target.WireGuardIP)

	cmd, err := remote.NewCommand(v.ctx,
		fmt.Sprintf("vpn-stats-%s-%d", source.Name, time.Now().Unix()),
		&remote.CommandArgs{
			Connection: &remote.ConnectionArgs{
				Host:       source.PublicIP,
				Port:       pulumi.Float64(22),
				User:       pulumi.String(source.SSHUser),
				PrivateKey: pulumi.String(v.getSSHPrivateKey()),
			},
			Create: pulumi.String(statsScript),
		},
		pulumi.IgnoreChanges([]string{"create"}),
	)

	if err != nil {
		conn.Error = err
		return
	}

	cmd.Stdout.ApplyT(func(output string) string {
		// Update stats based on output
		conn.LastCheck = time.Now()
		return output
	})
}

// reportConnectivityStatus periodically reports the connectivity status
func (v *VPNConnectivityChecker) reportConnectivityStatus(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			v.mu.RLock()

			totalConnections := 0
			establishedConnections := 0

			statusMessages := []string{}
			for nodeName, result := range v.results {
				nodeEstablished := 0
				for _, conn := range result.Connections {
					totalConnections++
					if conn.IsConnected {
						establishedConnections++
						nodeEstablished++
					}
				}

				emoji := "✗"
				if result.AllConnected {
					emoji = "✓"
				}

				statusMessages = append(statusMessages,
					fmt.Sprintf("%s %s: %d/%d", emoji, nodeName, nodeEstablished, len(v.nodes)-1))
			}

			v.mu.RUnlock()

			if totalConnections > 0 {
				v.ctx.Log.Info("VPN connectivity status", nil)
			}
		}
	}
}

// countConnections counts established connections for a result
func (v *VPNConnectivityChecker) countConnections(result *ConnectivityResult) int {
	count := 0
	for _, conn := range result.Connections {
		if conn.IsConnected {
			count++
		}
	}
	return count
}

// GetConnectivityMatrix returns the full connectivity matrix
func (v *VPNConnectivityChecker) GetConnectivityMatrix() map[string]map[string]bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	matrix := make(map[string]map[string]bool)

	for sourceName, result := range v.results {
		matrix[sourceName] = make(map[string]bool)
		for targetName, conn := range result.Connections {
			matrix[sourceName][targetName] = conn.IsConnected
		}
	}

	return matrix
}

// PrintConnectivityMatrix prints a visual representation of connectivity
func (v *VPNConnectivityChecker) PrintConnectivityMatrix() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	v.ctx.Log.Info("VPN Connectivity Matrix", nil)
	v.ctx.Log.Info("=======================", nil)

	// Print header
	header := "Source\\Target\t"
	for _, node := range v.nodes {
		header += fmt.Sprintf("%s\t", node.Name[:6])
	}
	v.ctx.Log.Info(header, nil)

	// Print each row
	for _, sourceNode := range v.nodes {
		row := fmt.Sprintf("%s\t", sourceNode.Name[:6])

		if result, exists := v.results[sourceNode.Name]; exists {
			for _, targetNode := range v.nodes {
				if sourceNode.Name == targetNode.Name {
					row += "---\t"
				} else if conn, exists := result.Connections[targetNode.Name]; exists {
					if conn.IsConnected {
						row += fmt.Sprintf("✓(%.0fms)\t", conn.Latency.Seconds()*1000)
					} else {
						row += "✗\t"
					}
				} else {
					row += "?\t"
				}
			}
		}

		v.ctx.Log.Info(row, nil)
	}
}

// WaitForTunnelEstablishment waits for all WireGuard tunnels to be established
func (v *VPNConnectivityChecker) WaitForTunnelEstablishment() error {
	v.ctx.Log.Info("Waiting for WireGuard tunnels to establish", nil)

	ctx, cancel := context.WithTimeout(context.Background(), v.timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for WireGuard tunnels")

		case <-ticker.C:
			// Check if all tunnels are established
			allEstablished := true

			for _, node := range v.nodes {
				if !v.isWireGuardReady(node) {
					allEstablished = false
					break
				}
			}

			if allEstablished {
				v.ctx.Log.Info("All WireGuard tunnels established", nil)
				return nil
			}

			v.ctx.Log.Info("Waiting for WireGuard tunnels...", nil)
		}
	}
}

// isWireGuardReady checks if WireGuard is ready on a node
func (v *VPNConnectivityChecker) isWireGuardReady(node *providers.NodeOutput) bool {
	checkScript := `#!/bin/bash
if [ -f /etc/wireguard/wg0.conf ] && wg show wg0 &>/dev/null; then
    echo "WIREGUARD:READY"
    wg show wg0 peers | wc -l | sed 's/^/PEER_COUNT:/'
else
    echo "WIREGUARD:NOT_READY"
fi
`

	cmd, err := remote.NewCommand(v.ctx,
		fmt.Sprintf("wg-ready-check-%s-%d", node.Name, time.Now().Unix()),
		&remote.CommandArgs{
			Connection: &remote.ConnectionArgs{
				Host:       node.PublicIP,
				Port:       pulumi.Float64(22),
				User:       pulumi.String(node.SSHUser),
				PrivateKey: pulumi.String(v.getSSHPrivateKey()),
			},
			Create: pulumi.String(checkScript),
		},
		pulumi.IgnoreChanges([]string{"create"}),
	)

	if err != nil {
		return false
	}

	ready := false
	cmd.Stdout.ApplyT(func(output string) string {
		if contains(output, "WIREGUARD:READY") {
			ready = true
		}
		return output
	})

	return ready
}

// getSSHPrivateKey retrieves the SSH private key
func (v *VPNConnectivityChecker) getSSHPrivateKey() string {
	// In production, read the actual key from v.sshKeyPath
	return "SSH_PRIVATE_KEY_CONTENT"
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func extractValue(s, prefix string) string {
	idx := 0
	for i := 0; i <= len(s)-len(prefix); i++ {
		if s[i:i+len(prefix)] == prefix {
			idx = i + len(prefix)
			break
		}
	}

	if idx > 0 {
		end := idx
		for end < len(s) && s[end] != '\n' && s[end] != ' ' {
			end++
		}
		return s[idx:end]
	}

	return ""
}
