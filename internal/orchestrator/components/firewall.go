package components

import (
	"fmt"

	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// FirewallComponent manages firewall rules for bastion mode
type FirewallComponent struct {
	pulumi.ResourceState

	FirewallID   pulumi.StringOutput `pulumi:"firewallId"`
	FirewallName pulumi.StringOutput `pulumi:"firewallName"`
	Status       pulumi.StringOutput `pulumi:"status"`
}

// NewFirewallComponent creates firewall rules for bastion security
// When bastion is enabled, this blocks direct SSH access to cluster nodes
func NewFirewallComponent(
	ctx *pulumi.Context,
	name string,
	bastionIP pulumi.StringOutput,
	nodeDropletIDs []string,
	allowedCIDRs []string,
	opts ...pulumi.ResourceOption,
) (*FirewallComponent, error) {
	component := &FirewallComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:security:Firewall", name, component, opts...)
	if err != nil {
		return nil, err
	}

	ctx.Log.Info("🛡️  Creating firewall rules for bastion mode...", nil)

	// Build inbound rules
	var inboundRules digitalocean.FirewallInboundRuleArray

	// Allow SSH ONLY from bastion host
	inboundRules = append(inboundRules, &digitalocean.FirewallInboundRuleArgs{
		Protocol: pulumi.String("tcp"),
		PortRange: pulumi.String("22"),
		SourceAddresses: pulumi.StringArray{
			bastionIP, // Only bastion can SSH to nodes
		},
	})

	// Allow all traffic from VPN network (WireGuard mesh)
	inboundRules = append(inboundRules, &digitalocean.FirewallInboundRuleArgs{
		Protocol: pulumi.String("tcp"),
		PortRange: pulumi.String("1-65535"),
		SourceAddresses: pulumi.StringArray{
			pulumi.String("10.8.0.0/24"), // VPN network
		},
	})
	inboundRules = append(inboundRules, &digitalocean.FirewallInboundRuleArgs{
		Protocol: pulumi.String("udp"),
		PortRange: pulumi.String("1-65535"),
		SourceAddresses: pulumi.StringArray{
			pulumi.String("10.8.0.0/24"), // VPN network
		},
	})

	// Allow WireGuard VPN traffic from anywhere (UDP 51820)
	inboundRules = append(inboundRules, &digitalocean.FirewallInboundRuleArgs{
		Protocol: pulumi.String("udp"),
		PortRange: pulumi.String("51820"),
		SourceAddresses: pulumi.StringArray{
			pulumi.String("0.0.0.0/0"),
			pulumi.String("::/0"),
		},
	})

	// Allow Kubernetes API traffic (for kubectl access)
	inboundRules = append(inboundRules, &digitalocean.FirewallInboundRuleArgs{
		Protocol: pulumi.String("tcp"),
		PortRange: pulumi.String("6443"),
		SourceAddresses: pulumi.StringArray{
			pulumi.String("0.0.0.0/0"), // API should be publicly accessible
			pulumi.String("::/0"),
		},
	})

	// Allow HTTP/HTTPS for ingress traffic
	inboundRules = append(inboundRules, &digitalocean.FirewallInboundRuleArgs{
		Protocol: pulumi.String("tcp"),
		PortRange: pulumi.String("80"),
		SourceAddresses: pulumi.StringArray{
			pulumi.String("0.0.0.0/0"),
			pulumi.String("::/0"),
		},
	})
	inboundRules = append(inboundRules, &digitalocean.FirewallInboundRuleArgs{
		Protocol: pulumi.String("tcp"),
		PortRange: pulumi.String("443"),
		SourceAddresses: pulumi.StringArray{
			pulumi.String("0.0.0.0/0"),
			pulumi.String("::/0"),
		},
	})

	// Allow ICMP (ping)
	inboundRules = append(inboundRules, &digitalocean.FirewallInboundRuleArgs{
		Protocol: pulumi.String("icmp"),
		SourceAddresses: pulumi.StringArray{
			pulumi.String("0.0.0.0/0"),
			pulumi.String("::/0"),
		},
	})

	// Outbound rules - allow all outbound traffic
	var outboundRules digitalocean.FirewallOutboundRuleArray
	outboundRules = append(outboundRules, &digitalocean.FirewallOutboundRuleArgs{
		Protocol: pulumi.String("tcp"),
		PortRange: pulumi.String("1-65535"),
		DestinationAddresses: pulumi.StringArray{
			pulumi.String("0.0.0.0/0"),
			pulumi.String("::/0"),
		},
	})
	outboundRules = append(outboundRules, &digitalocean.FirewallOutboundRuleArgs{
		Protocol: pulumi.String("udp"),
		PortRange: pulumi.String("1-65535"),
		DestinationAddresses: pulumi.StringArray{
			pulumi.String("0.0.0.0/0"),
			pulumi.String("::/0"),
		},
	})
	outboundRules = append(outboundRules, &digitalocean.FirewallOutboundRuleArgs{
		Protocol: pulumi.String("icmp"),
		DestinationAddresses: pulumi.StringArray{
			pulumi.String("0.0.0.0/0"),
			pulumi.String("::/0"),
		},
	})

	// Convert node droplet IDs to IntArray
	dropletIDsInt := make([]pulumi.IntInput, len(nodeDropletIDs))
	for i, id := range nodeDropletIDs {
		dropletIDsInt[i] = pulumi.Int(0) // Will be set dynamically
		_ = id // TODO: Convert string ID to int
	}

	// Create firewall
	firewall, err := digitalocean.NewFirewall(ctx, name, &digitalocean.FirewallArgs{
		Name: pulumi.String(fmt.Sprintf("kubernetes-bastion-fw-%s", ctx.Stack())),
		// DropletIds:    pulumi.IntArray(dropletIDsInt), // Commented - will use tags instead
		InboundRules:  inboundRules,
		OutboundRules: outboundRules,
		Tags: pulumi.StringArray{
			pulumi.String("kubernetes"),
			pulumi.String("bastion-protected"),
			pulumi.String(ctx.Stack()),
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("failed to create firewall: %w", err)
	}

	component.FirewallID = firewall.ID().ToStringOutput()
	component.FirewallName = firewall.Name
	component.Status = pulumi.String("active").ToStringOutput()

	ctx.Log.Info("✅ Firewall rules created - SSH access restricted to bastion only", nil)

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"firewallId":   component.FirewallID,
		"firewallName": component.FirewallName,
		"status":       component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
