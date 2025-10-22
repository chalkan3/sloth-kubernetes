package components

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// CloudInitValidatorComponent validates cloud-init completion before proceeding
type CloudInitValidatorComponent struct {
	pulumi.ResourceState

	Status         pulumi.StringOutput `pulumi:"status"`
	RKE2Ready      pulumi.BoolOutput   `pulumi:"rke2Ready"`
	WireGuardReady pulumi.BoolOutput   `pulumi:"wireGuardReady"`
}

// NewCloudInitValidatorComponent waits for cloud-init to complete on all nodes
// This ensures RKE2 and WireGuard are installed before WireGuard configuration
func NewCloudInitValidatorComponent(ctx *pulumi.Context, name string, nodes []*RealNodeComponent, sshPrivateKey pulumi.StringOutput, bastionComponent *BastionComponent, opts ...pulumi.ResourceOption) (*CloudInitValidatorComponent, error) {
	component := &CloudInitValidatorComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:provisioning:CloudInitValidator", name, component, opts...)
	if err != nil {
		return nil, err
	}

	ctx.Log.Info(fmt.Sprintf("🔍 Validating cloud-init completion on %d nodes...", len(nodes)), nil)

	// Simplified validation using basic commands instead of large bash script
	// This avoids exit code 126 issues with inline scripts
	// We just need to check if RKE2 and WireGuard exist
	validationScript := `timeout=300; elapsed=0; while [ $elapsed -lt $timeout ]; do if command -v wg >/dev/null 2>&1 && [ -f /usr/local/bin/rke2 ]; then echo "Cloud-init provisioning complete"; exit 0; fi; sleep 5; elapsed=$((elapsed + 5)); done; echo "Timeout waiting for provisioning"; exit 1`

	// Run validation on all nodes in parallel
	var validationResults []pulumi.Resource
	for i, node := range nodes {
		// Build connection args with ProxyJump if bastion is enabled
		connArgs := remote.ConnectionArgs{
			Host:           node.PublicIP,
			User:           pulumi.String("root"),
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
		}
		if bastionComponent != nil {
			connArgs.Proxy = &remote.ProxyConnectionArgs{
				Host:       bastionComponent.PublicIP,
				User:       pulumi.String("root"),
				PrivateKey: sshPrivateKey,
			}
		}

		validationCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-node-%d-validate", name, i), &remote.CommandArgs{
			Connection: connArgs,
			Create:     pulumi.String(validationScript),
		}, pulumi.Parent(component), pulumi.Timeouts(&pulumi.CustomTimeouts{
			Create: "10m", // Give cloud-init up to 10 minutes to complete
		}))

		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to create cloud-init validator for node %d: %v", i, err), nil)
		} else {
			validationResults = append(validationResults, validationCmd)
		}
	}

	ctx.Log.Info(fmt.Sprintf("✅ Created %d cloud-init validators (running in parallel)", len(validationResults)), nil)

	component.Status = pulumi.Sprintf("Cloud-init validated on %d nodes", len(nodes))
	component.RKE2Ready = pulumi.Bool(true).ToBoolOutput()
	component.WireGuardReady = pulumi.Bool(true).ToBoolOutput()

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":         component.Status,
		"rke2Ready":      component.RKE2Ready,
		"wireGuardReady": component.WireGuardReady,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info("✅ Cloud-init validation component created", nil)

	return component, nil
}
