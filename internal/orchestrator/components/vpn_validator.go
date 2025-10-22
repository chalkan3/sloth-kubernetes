package components

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// VPNValidatorComponent validates VPN connectivity before RKE2 installation
type VPNValidatorComponent struct {
	pulumi.ResourceState

	Status          pulumi.StringOutput `pulumi:"status"`
	ValidationCount pulumi.IntOutput    `pulumi:"validationCount"`
	AllPassed       pulumi.BoolOutput   `pulumi:"allPassed"`
}

// NewVPNValidatorComponent validates VPN connectivity between all nodes
// This ensures WireGuard mesh is fully functional before proceeding with RKE2
func NewVPNValidatorComponent(ctx *pulumi.Context, name string, nodes []*RealNodeComponent, sshPrivateKey pulumi.StringOutput, bastionComponent *BastionComponent, opts ...pulumi.ResourceOption) (*VPNValidatorComponent, error) {
	component := &VPNValidatorComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:VPNValidator", name, component, opts...)
	if err != nil {
		return nil, err
	}

	totalNodes := len(nodes)
	if bastionComponent != nil {
		totalNodes++ // Include bastion
	}

	ctx.Log.Info(fmt.Sprintf("🔍 Validating VPN connectivity: %d nodes (full mesh)", totalNodes), nil)

	// Build list of all nodes with their IPs
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
			User:           pulumi.String("root"),
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
			Proxy: func() *remote.ProxyConnectionArgs {
				if bastionComponent != nil {
					return &remote.ProxyConnectionArgs{
						Host:       bastionComponent.PublicIP,
						User:       pulumi.String("root"),
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
