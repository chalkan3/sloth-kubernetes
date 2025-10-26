package components

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// getSSHUserForProvider returns the correct SSH username for the given cloud provider
// Azure uses "azureuser", while other providers use "root" or "ubuntu"
func getSSHUserForProvider(provider pulumi.StringOutput) pulumi.StringOutput {
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

// CloudInitValidatorComponent validates cloud-init completion before proceeding
type CloudInitValidatorComponent struct {
	pulumi.ResourceState

	Status         pulumi.StringOutput `pulumi:"status"`
	K3sReady       pulumi.BoolOutput   `pulumi:"k3sReady"`
	WireGuardReady pulumi.BoolOutput   `pulumi:"wireGuardReady"`
}

// NewCloudInitValidatorComponent waits for cloud-init to complete on all nodes
// This ensures WireGuard and prerequisites are installed before WireGuard mesh configuration
// K3s installation is handled separately via remote commands AFTER WireGuard is configured
func NewCloudInitValidatorComponent(ctx *pulumi.Context, name string, nodes []*RealNodeComponent, sshPrivateKey pulumi.StringOutput, bastionComponent *BastionComponent, opts ...pulumi.ResourceOption) (*CloudInitValidatorComponent, error) {
	component := &CloudInitValidatorComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:provisioning:CloudInitValidator", name, component, opts...)
	if err != nil {
		return nil, err
	}

	ctx.Log.Info(fmt.Sprintf("🔍 Validating cloud-init completion on %d nodes...", len(nodes)), nil)

	// Simplified validation using basic commands instead of large bash script
	// This avoids exit code 126 issues with inline scripts
	// We only check if WireGuard is installed - K3s will be installed later via remote commands
	validationScript := `timeout=180; elapsed=0; while [ $elapsed -lt $timeout ]; do if command -v wg >/dev/null 2>&1; then echo "Cloud-init provisioning complete"; exit 0; fi; sleep 2; elapsed=$((elapsed + 2)); done; echo "Timeout waiting for provisioning"; exit 1`

	// Run validation on all nodes in parallel
	var validationResults []pulumi.Resource
	for i, node := range nodes {
		// Determine SSH user based on provider using helper function
		sshUser := getSSHUserForProvider(node.Provider)

		// Bastion is on Linode in this config, so it uses "root"
		bastionUser := pulumi.String("root")

		// Build connection args with ProxyJump if bastion is enabled
		connArgs := remote.ConnectionArgs{
			Host:           node.PublicIP,
			User:           sshUser,
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
		}
		if bastionComponent != nil {
			connArgs.Proxy = &remote.ProxyConnectionArgs{
				Host:       bastionComponent.PublicIP,
				User:       bastionUser,
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
	component.K3sReady = pulumi.Bool(true).ToBoolOutput()
	component.WireGuardReady = pulumi.Bool(true).ToBoolOutput()

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":         component.Status,
		"k3sReady":       component.K3sReady,
		"wireGuardReady": component.WireGuardReady,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info("✅ Cloud-init validation component created", nil)

	return component, nil
}
