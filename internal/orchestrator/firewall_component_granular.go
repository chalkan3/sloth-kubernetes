package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// FirewallRuleComponent represents a single firewall rule
type FirewallRuleComponent struct {
	pulumi.ResourceState

	RuleName    pulumi.StringOutput `pulumi:"ruleName"`
	Protocol    pulumi.StringOutput `pulumi:"protocol"`
	Port        pulumi.StringOutput `pulumi:"port"`
	Source      pulumi.StringOutput `pulumi:"source"`
	Description pulumi.StringOutput `pulumi:"description"`
	Action      pulumi.StringOutput `pulumi:"action"`
	Status      pulumi.StringOutput `pulumi:"status"`
}

// OSFirewallNodeComponent represents OS firewall config for a single node
type OSFirewallNodeComponent struct {
	pulumi.ResourceState

	NodeName pulumi.StringOutput `pulumi:"nodeName"`
	Rules    pulumi.IntOutput    `pulumi:"rulesCount"`
	Status   pulumi.StringOutput `pulumi:"status"`
}

// NewOSFirewallComponentGranular creates granular OS firewall components
func NewOSFirewallComponentGranular(ctx *pulumi.Context, name string, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*OSFirewallComponent, error) {
	component := &OSFirewallComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:security:OSFirewall", name, component, opts...)
	if err != nil {
		return nil, err
	}

	totalRules := 0

	// Kubernetes required ports for master nodes
	masterRules := []struct {
		name        string
		protocol    string
		port        string
		source      string
		description string
	}{
		{"kubernetes-api", "tcp", "6443", "10.0.0.0/8", "Kubernetes API Server"},
		{"etcd-client", "tcp", "2379", "10.0.0.0/8", "etcd client communication"},
		{"etcd-peer", "tcp", "2380", "10.0.0.0/8", "etcd peer communication"},
		{"kubelet-api", "tcp", "10250", "10.0.0.0/8", "Kubelet API"},
		{"kube-scheduler", "tcp", "10251", "10.0.0.0/8", "kube-scheduler"},
		{"kube-controller", "tcp", "10252", "10.0.0.0/8", "kube-controller-manager"},
		{"nodeport-range", "tcp", "30000-32767", "10.0.0.0/8", "NodePort Services"},
		{"calico-bgp", "tcp", "179", "10.0.0.0/8", "Calico BGP"},
		{"calico-vxlan", "udp", "4789", "10.0.0.0/8", "Calico VXLAN"},
		{"wireguard", "udp", "51820", "0.0.0.0/0", "WireGuard VPN"},
		{"ssh-vpn", "tcp", "22", "10.8.0.0/24", "SSH via VPN only"},
	}

	// Worker node ports
	workerRules := []struct {
		name        string
		protocol    string
		port        string
		source      string
		description string
	}{
		{"kubelet-api", "tcp", "10250", "10.0.0.0/8", "Kubelet API"},
		{"nodeport-range", "tcp", "30000-32767", "10.0.0.0/8", "NodePort Services"},
		{"calico-bgp", "tcp", "179", "10.0.0.0/8", "Calico BGP"},
		{"calico-vxlan", "udp", "4789", "10.0.0.0/8", "Calico VXLAN"},
		{"wireguard", "udp", "51820", "0.0.0.0/0", "WireGuard VPN"},
		{"ssh-vpn", "tcp", "22", "10.8.0.0/24", "SSH via VPN only"},
	}

	// Create OS firewall components for each node
	nodes.ApplyT(func(nodeList []interface{}) error {
		for i := 0; i < len(nodeList) && i < 6; i++ {
			nodeName := fmt.Sprintf("node-%d", i+1)
			isMaster := i < 3 // First 3 nodes are masters

			nodeFirewall, err := newOSFirewallNodeComponent(ctx,
				fmt.Sprintf("%s-%s", name, nodeName),
				nodeName,
				component)
			if err != nil {
				return err
			}

			// Create individual firewall rule components for this node
			if isMaster {
				for _, rule := range masterRules {
					_, err := newFirewallRuleComponent(ctx,
						fmt.Sprintf("%s-%s-%s", name, nodeName, rule.name),
						rule.name,
						rule.protocol,
						rule.port,
						rule.source,
						rule.description,
						"allow",
						nodeFirewall)
					if err != nil {
						return err
					}
					totalRules++
				}
			} else {
				for _, rule := range workerRules {
					_, err := newFirewallRuleComponent(ctx,
						fmt.Sprintf("%s-%s-%s", name, nodeName, rule.name),
						rule.name,
						rule.protocol,
						rule.port,
						rule.source,
						rule.description,
						"allow",
						nodeFirewall)
					if err != nil {
						return err
					}
					totalRules++
				}
			}
		}
		return nil
	})

	// Set component outputs
	component.Status = pulumi.Sprintf("Configured %d firewall rules across all nodes", totalRules).ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status": component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newOSFirewallNodeComponent creates a firewall component for a single node
func newOSFirewallNodeComponent(ctx *pulumi.Context, name, nodeName string, parent pulumi.Resource) (*OSFirewallNodeComponent, error) {
	component := &OSFirewallNodeComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:security:NodeFirewall", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.NodeName = pulumi.String(nodeName).ToStringOutput()
	component.Rules = pulumi.Int(0).ToIntOutput() // Will be updated
	component.Status = pulumi.String("configuring").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"nodeName": component.NodeName,
		"rules":    component.Rules,
		"status":   component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newFirewallRuleComponent creates a single firewall rule component
func newFirewallRuleComponent(ctx *pulumi.Context, name, ruleName, protocol, port, source, description, action string, parent pulumi.Resource) (*FirewallRuleComponent, error) {
	component := &FirewallRuleComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:security:FirewallRule", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.RuleName = pulumi.String(ruleName).ToStringOutput()
	component.Protocol = pulumi.String(protocol).ToStringOutput()
	component.Port = pulumi.String(port).ToStringOutput()
	component.Source = pulumi.String(source).ToStringOutput()
	component.Description = pulumi.String(description).ToStringOutput()
	component.Action = pulumi.String(action).ToStringOutput()
	component.Status = pulumi.String("configured").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"ruleName":    component.RuleName,
		"protocol":    component.Protocol,
		"port":        component.Port,
		"source":      component.Source,
		"description": component.Description,
		"action":      component.Action,
		"status":      component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
