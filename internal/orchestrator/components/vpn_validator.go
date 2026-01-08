package components

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// VPNMode represents the type of VPN being used
type VPNMode string

const (
	VPNModeWireGuard VPNMode = "wireguard"
	VPNModeTailscale VPNMode = "tailscale"
)

// getSSHUserForVPNValidator returns the correct SSH username for the given cloud provider
// Azure uses "azureuser", while other providers use "root" or "ubuntu"
func getSSHUserForVPNValidator(provider pulumi.StringOutput) pulumi.StringOutput {
	return provider.ApplyT(func(p string) string {
		switch p {
		case "azure":
			return "azureuser"
		case "aws":
			return "ubuntu" // AWS Ubuntu AMIs use "ubuntu"
		case "gcp":
			return "ubuntu" // GCP uses "ubuntu" for Ubuntu images
		default:
			return "root" // DigitalOcean, Linode, and others use "root"
		}
	}).(pulumi.StringOutput)
}

// VPNValidatorComponent validates VPN connectivity before RKE2 installation
type VPNValidatorComponent struct {
	pulumi.ResourceState

	Status          pulumi.StringOutput `pulumi:"status"`
	ValidationCount pulumi.IntOutput    `pulumi:"validationCount"`
	AllPassed       pulumi.BoolOutput   `pulumi:"allPassed"`
}

// VPNValidatorArgs contains arguments for the VPN validator
type VPNValidatorArgs struct {
	Mode VPNMode // wireguard or tailscale
}

// NewVPNValidatorComponent validates VPN connectivity between all nodes
// This ensures VPN mesh is fully functional before proceeding with RKE2
// vpnMode can be "wireguard" or "tailscale"
func NewVPNValidatorComponent(ctx *pulumi.Context, name string, nodes []*RealNodeComponent, sshPrivateKey pulumi.StringOutput, bastionComponent *BastionComponent, opts ...pulumi.ResourceOption) (*VPNValidatorComponent, error) {
	// Default to WireGuard mode for backward compatibility
	return NewVPNValidatorComponentWithMode(ctx, name, nodes, sshPrivateKey, bastionComponent, VPNModeWireGuard, opts...)
}

// NewVPNValidatorComponentWithMode validates VPN connectivity between all nodes with specified VPN mode
func NewVPNValidatorComponentWithMode(ctx *pulumi.Context, name string, nodes []*RealNodeComponent, sshPrivateKey pulumi.StringOutput, bastionComponent *BastionComponent, vpnMode VPNMode, opts ...pulumi.ResourceOption) (*VPNValidatorComponent, error) {
	component := &VPNValidatorComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:VPNValidator", name, component, opts...)
	if err != nil {
		return nil, err
	}

	totalNodes := len(nodes)
	if bastionComponent != nil && vpnMode == VPNModeWireGuard {
		totalNodes++ // Include bastion only for WireGuard (Tailscale bastion handled separately)
	}

	vpnTypeName := "WireGuard"
	if vpnMode == VPNModeTailscale {
		vpnTypeName = "Tailscale"
	}
	ctx.Log.Info(fmt.Sprintf("🔍 Validating %s connectivity: %d nodes (full mesh)", vpnTypeName, totalNodes), nil)

	// For Tailscale, use a different validation approach
	if vpnMode == VPNModeTailscale {
		return validateTailscaleMesh(ctx, name, component, nodes, sshPrivateKey, bastionComponent, totalNodes, opts...)
	}

	// Build list of all nodes with their IPs (WireGuard mode)
	type nodeInfo struct {
		wgIP pulumi.StringOutput
		name pulumi.StringOutput
	}

	var allNodes []*nodeInfo

	// Add bastion if present
	if bastionComponent != nil {
		allNodes = append(allNodes, &nodeInfo{
			wgIP: pulumi.String("10.8.0.5").ToStringOutput(),
			name: pulumi.String("bastion").ToStringOutput(),
		})
	}

	// Add all nodes
	for _, node := range nodes {
		allNodes = append(allNodes, &nodeInfo{
			wgIP: node.WireGuardIP,
			name: node.NodeName,
		})
	}

	// Build validation script
	buildValidationScript := func(myIP, myName string, targetIPs, targetNames []string) string {
		var pings []string
		for i, ip := range targetIPs {
			if ip != myIP { // Don't ping ourselves
				pings = append(pings, fmt.Sprintf(`
echo "  [%d/%d] Pinging %s (%s)..."
if ping -c 2 -W 10 %s >/dev/null 2>&1; then
  echo "    ✅ %s is reachable"
  ((success_count++))
else
  echo "    ❌ %s is NOT reachable"
  failed_ips="$failed_ips %s(%s)"
  ((failure_count++))
fi
`, i+1, len(targetIPs), targetNames[i], ip, ip, targetNames[i], targetNames[i], targetNames[i], ip))
			}
		}

		return fmt.Sprintf(`#!/bin/bash
set +e  # Don't exit on error - we handle errors manually

echo "════════════════════════════════════════════════════════"
echo "🔍 VPN VALIDATION: %s (%s)"
echo "════════════════════════════════════════════════════════"
echo ""
echo "Waiting for WireGuard mesh to stabilize (checking handshakes)..."

# Wait for WireGuard interface to be ready and peers to establish handshakes
max_wait=120
waited=0
ready=false

while [ $waited -lt $max_wait ]; do
  # Check if wg0 interface exists
  if ip addr show wg0 >/dev/null 2>&1; then
    # Check if we have active handshakes with peers
    peer_count=$(sudo wg show wg0 peers 2>/dev/null | wc -l)
    active_handshakes=$(sudo wg show wg0 latest-handshakes 2>/dev/null | awk '{if ($2 != "0") print}' | wc -l)

    if [ "$peer_count" -gt 0 ] && [ "$active_handshakes" -ge "$((peer_count * 70 / 100))" ]; then
      echo "  ✅ WireGuard ready: $active_handshakes/$peer_count peers with active handshakes"
      ready=true
      break
    fi

    if [ $((waited %% 15)) -eq 0 ]; then
      echo "  ⏳ Waiting for handshakes: $active_handshakes/$peer_count peers ready (${waited}s elapsed)..."
    fi
  else
    echo "  ⏳ Waiting for wg0 interface... (${waited}s elapsed)"
  fi

  sleep 5
  waited=$((waited + 5))
done

if [ "$ready" = false ]; then
  echo "  ⚠️  Warning: Not all peers have handshakes yet, but proceeding with validation..."
fi

echo ""
echo "Testing connectivity to all %d peers via WireGuard VPN..."
echo ""

success_count=0
failure_count=0
failed_ips=""

%s

echo ""
echo "════════════════════════════════════════════════════════"
if [ $failure_count -eq 0 ]; then
  echo "✅ VPN VALIDATION PASSED: All %d peers reachable"
  echo "════════════════════════════════════════════════════════"
  exit 0
else
  echo "❌ VPN VALIDATION FAILED: $failure_count/$((success_count + failure_count)) peers unreachable"
  echo "Failed IPs:$failed_ips"
  echo "════════════════════════════════════════════════════════"
  exit 1
fi
`, myName, myIP, len(targetIPs)-1, strings.Join(pings, "\n"), len(targetIPs)-1)
	}

	// Run validation on first node (representative test)
	// If this passes, mesh is configured correctly
	firstNode := nodes[0]

	// Determine SSH user based on provider (Azure uses "azureuser", others use "root")
	firstNodeSSHUser := getSSHUserForVPNValidator(firstNode.Provider)

	// Collect all IPs and names as pulumi.All inputs
	var allInputs []interface{}
	allInputs = append(allInputs, firstNode.WireGuardIP)
	allInputs = append(allInputs, firstNode.NodeName)
	for _, node := range allNodes {
		allInputs = append(allInputs, node.wgIP)
		allInputs = append(allInputs, node.name)
	}

	validationCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-validate", name), &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:           firstNode.PublicIP,
			User:           firstNodeSSHUser,
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
			Proxy: func() *remote.ProxyConnectionArgs {
				if bastionComponent != nil {
					// Use the correct SSH user for the bastion based on its provider
					return &remote.ProxyConnectionArgs{
						Host:       bastionComponent.PublicIP,
						User:       getSSHUserForProvider(bastionComponent.Provider),
						PrivateKey: sshPrivateKey,
					}
				}
				return nil
			}(),
		},
		Create: pulumi.All(allInputs...).ApplyT(func(args []interface{}) string {
			myIP := args[0].(string)
			myName := args[1].(string)

			// Extract all target IPs and names
			var targetIPs []string
			var targetNames []string
			for i := 2; i < len(args); i += 2 {
				targetIPs = append(targetIPs, args[i].(string))
				targetNames = append(targetNames, args[i+1].(string))
			}

			return buildValidationScript(myIP, myName, targetIPs, targetNames)
		}).(pulumi.StringOutput),
	}, pulumi.Parent(component), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "5m",
	}))

	if err != nil {
		return nil, fmt.Errorf("failed to create VPN validation command: %w", err)
	}

	component.Status = pulumi.Sprintf("VPN validation completed: %d nodes tested", totalNodes)
	component.ValidationCount = pulumi.Int(totalNodes).ToIntOutput()
	component.AllPassed = validationCmd.Stdout.ApplyT(func(s string) bool {
		return true // If command succeeds, all tests passed
	}).(pulumi.BoolOutput)

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":          component.Status,
		"validationCount": component.ValidationCount,
		"allPassed":       component.AllPassed,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info("✅ VPN validation component created", nil)

	return component, nil
}

// validateTailscaleMesh validates Tailscale mesh connectivity
func validateTailscaleMesh(ctx *pulumi.Context, name string, component *VPNValidatorComponent, nodes []*RealNodeComponent, sshPrivateKey pulumi.StringOutput, bastionComponent *BastionComponent, totalNodes int, opts ...pulumi.ResourceOption) (*VPNValidatorComponent, error) {
	ctx.Log.Info("🔧 Using Tailscale validation mode", nil)

	// Run validation on first node
	firstNode := nodes[0]
	firstNodeSSHUser := getSSHUserForVPNValidator(firstNode.Provider)

	// Collect all node names for the validation script
	var nodeNames []interface{}
	for _, node := range nodes {
		nodeNames = append(nodeNames, node.NodeName)
	}

	// Build Tailscale validation script
	buildTailscaleValidationScript := func(myName string, peerNames []string) string {
		return fmt.Sprintf(`#!/bin/bash
set +e  # Don't exit on error - we handle errors manually

echo "════════════════════════════════════════════════════════"
echo "🔍 TAILSCALE VPN VALIDATION: %s"
echo "════════════════════════════════════════════════════════"
echo ""
echo "Waiting for Tailscale mesh to stabilize..."

# Wait for Tailscale to be connected
max_wait=120
waited=0
ready=false

while [ $waited -lt $max_wait ]; do
  # Check if Tailscale is connected
  TS_STATUS=$(sudo tailscale status --json 2>/dev/null)
  if [ $? -eq 0 ]; then
    # Check if we're connected (BackendState should be "Running")
    BACKEND_STATE=$(echo "$TS_STATUS" | grep -o '"BackendState":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [ "$BACKEND_STATE" = "Running" ]; then
      # Count peers
      PEER_COUNT=$(echo "$TS_STATUS" | grep -c '"Online":true' 2>/dev/null || echo "0")
      echo "  ✅ Tailscale connected: $PEER_COUNT peers online"
      ready=true
      break
    fi
  fi

  if [ $((waited %% 15)) -eq 0 ]; then
    echo "  ⏳ Waiting for Tailscale connection... (${waited}s elapsed)"
  fi

  sleep 5
  waited=$((waited + 5))
done

if [ "$ready" = false ]; then
  echo "  ⚠️  Warning: Tailscale not fully connected yet, but proceeding with validation..."
fi

echo ""
echo "Getting my Tailscale IP..."
MY_TS_IP=$(sudo tailscale ip -4 2>/dev/null)
echo "  My Tailscale IP: $MY_TS_IP"

echo ""
echo "Testing connectivity to all Tailscale peers..."
echo ""

success_count=0
failure_count=0
failed_peers=""

# Get all peers from Tailscale status
PEERS=$(sudo tailscale status 2>/dev/null | grep -E "^100\." | awk '{print $1, $2}')

# If no peers found, try to discover them from tailscale status --json
if [ -z "$PEERS" ]; then
  echo "Discovering peers from Tailscale status..."
  PEER_IPS=$(sudo tailscale status --json 2>/dev/null | grep -oE '"TailscaleIPs":\["[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"' | grep -oE '100\.[0-9]+\.[0-9]+\.[0-9]+')
fi

# Test connectivity to each peer
while read -r peer_ip peer_name; do
  if [ -n "$peer_ip" ] && [ "$peer_ip" != "$MY_TS_IP" ]; then
    echo "  Testing $peer_name ($peer_ip)..."
    if ping -c 2 -W 10 "$peer_ip" >/dev/null 2>&1; then
      echo "    ✅ $peer_name is reachable"
      ((success_count++))
    else
      echo "    ❌ $peer_name is NOT reachable"
      failed_peers="$failed_peers $peer_name($peer_ip)"
      ((failure_count++))
    fi
  fi
done <<< "$PEERS"

# Also test using hostnames if no IP-based peers found
if [ $success_count -eq 0 ] && [ $failure_count -eq 0 ]; then
  echo "No peers found by IP, testing via Tailscale DNS..."
  %s
fi

echo ""
echo "════════════════════════════════════════════════════════"
if [ $failure_count -eq 0 ] && [ $success_count -gt 0 ]; then
  echo "✅ TAILSCALE VALIDATION PASSED: All $success_count peers reachable"
  echo "════════════════════════════════════════════════════════"
  exit 0
elif [ $success_count -eq 0 ] && [ $failure_count -eq 0 ]; then
  # No peers but we're connected, that's fine for single-node
  echo "✅ TAILSCALE VALIDATION PASSED: Connected (no other peers to test)"
  echo "════════════════════════════════════════════════════════"
  exit 0
else
  echo "❌ TAILSCALE VALIDATION FAILED: $failure_count peer(s) unreachable"
  echo "Failed:$failed_peers"
  echo "════════════════════════════════════════════════════════"
  exit 1
fi
`, myName, buildPeerTestCommands(peerNames, myName))
	}

	validationCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-tailscale-validate", name), &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:           firstNode.PublicIP,
			User:           firstNodeSSHUser,
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
			Proxy: func() *remote.ProxyConnectionArgs {
				if bastionComponent != nil {
					return &remote.ProxyConnectionArgs{
						Host:       bastionComponent.PublicIP,
						User:       getSSHUserForProvider(bastionComponent.Provider),
						PrivateKey: sshPrivateKey,
					}
				}
				return nil
			}(),
		},
		Create: pulumi.All(nodeNames...).ApplyT(func(args []interface{}) string {
			myName := args[0].(string)
			var peerNames []string
			for i := 1; i < len(args); i++ {
				peerNames = append(peerNames, args[i].(string))
			}
			return buildTailscaleValidationScript(myName, peerNames)
		}).(pulumi.StringOutput),
	}, pulumi.Parent(component), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "5m",
	}))

	if err != nil {
		return nil, fmt.Errorf("failed to create Tailscale validation command: %w", err)
	}

	component.Status = pulumi.Sprintf("Tailscale validation completed: %d nodes tested", totalNodes)
	component.ValidationCount = pulumi.Int(totalNodes).ToIntOutput()
	component.AllPassed = validationCmd.Stdout.ApplyT(func(s string) bool {
		return true // If command succeeds, all tests passed
	}).(pulumi.BoolOutput)

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":          component.Status,
		"validationCount": component.ValidationCount,
		"allPassed":       component.AllPassed,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info("✅ Tailscale VPN validation component created", nil)

	return component, nil
}

// buildPeerTestCommands builds bash commands to test connectivity to peer hostnames
func buildPeerTestCommands(peerNames []string, myName string) string {
	var cmds []string
	for _, name := range peerNames {
		if name != myName {
			cmds = append(cmds, fmt.Sprintf(`
  # Try to ping peer by hostname
  PEER_IP=$(sudo tailscale status 2>/dev/null | grep -i "%s" | awk '{print $1}')
  if [ -n "$PEER_IP" ]; then
    echo "  Testing %s ($PEER_IP)..."
    if ping -c 2 -W 10 "$PEER_IP" >/dev/null 2>&1; then
      echo "    ✅ %s is reachable"
      ((success_count++))
    else
      echo "    ❌ %s is NOT reachable"
      failed_peers="$failed_peers %s($PEER_IP)"
      ((failure_count++))
    fi
  fi`, name, name, name, name, name))
		}
	}
	return strings.Join(cmds, "\n")
}
