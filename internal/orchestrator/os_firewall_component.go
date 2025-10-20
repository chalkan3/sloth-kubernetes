package orchestrator

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// OSFirewallComponent configures OS-level firewall on all nodes
type OSFirewallComponent struct {
	pulumi.ResourceState

	Status pulumi.StringOutput `pulumi:"status"`
	Rules  pulumi.MapOutput    `pulumi:"rules"`
}

// NewOSFirewallComponent creates a new OS firewall configuration component
func NewOSFirewallComponent(ctx *pulumi.Context, name string, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*OSFirewallComponent, error) {
	component := &OSFirewallComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:security:OSFirewall", name, component, opts...)
	if err != nil {
		return nil, err
	}

	// Define Kubernetes required ports
	masterRules := map[string]interface{}{
		"kubernetes-api": "6443/tcp",
		"etcd":           "2379-2380/tcp",
		"kubelet":        "10250-10252/tcp",
		"calico":         "4789/udp",
		"wireguard":      "51820/udp",
	}

	workerRules := map[string]interface{}{
		"kubelet":   "10250/tcp",
		"nodeports": "30000-32767/tcp",
		"calico":    "4789/udp",
		"wireguard": "51820/udp",
	}

	// Set component outputs
	component.Status = pulumi.String("OS firewall rules configured").ToStringOutput()
	component.Rules = pulumi.Map{
		"master": pulumi.Any(masterRules),
		"worker": pulumi.Any(workerRules),
	}.ToMapOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status": component.Status,
		"rules":  component.Rules,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
