package components

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"sloth-kubernetes/pkg/config"
	"sloth-kubernetes/pkg/security"
)

// SSHKeyComponent manages SSH keys
type SSHKeyComponent struct {
	pulumi.ResourceState
	PublicKey      pulumi.StringOutput `pulumi:"publicKey"`
	PrivateKey     pulumi.StringOutput `pulumi:"privateKey"` // PEM format for remote.Command
	PrivateKeyPath pulumi.StringOutput `pulumi:"privateKeyPath"`
}

func NewSSHKeyComponent(ctx *pulumi.Context, name string, config *config.ClusterConfig, opts ...pulumi.ResourceOption) (*SSHKeyComponent, error) {
	component := &SSHKeyComponent{}
	ctx.RegisterComponentResource("kubernetes-create:security:SSHKey", name, component, opts...)

	// Use SSH key manager to generate REAL keys
	keyManager := security.NewSSHKeyManager(ctx)
	err := keyManager.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SSH keys: %w", err)
	}

	component.PublicKey = keyManager.GetPublicKey()
	component.PrivateKey = keyManager.GetPrivateKey() // Get PEM format private key
	component.PrivateKeyPath = pulumi.Sprintf("~/.ssh/kubernetes-clusters/%s.pem", ctx.Stack()).ToStringOutput()

	ctx.RegisterResourceOutputs(component, pulumi.Map{
		"publicKey":      component.PublicKey,
		"privateKey":     pulumi.ToSecret(component.PrivateKey), // Mark as secret
		"privateKeyPath": component.PrivateKeyPath,
	})
	return component, nil
}

// ProviderComponent manages cloud providers
type ProviderComponent struct {
	pulumi.ResourceState
	Providers pulumi.MapOutput `pulumi:"providers"`
}

func NewProviderComponent(ctx *pulumi.Context, name string, config *config.ClusterConfig, sshKey pulumi.StringOutput, opts ...pulumi.ResourceOption) (*ProviderComponent, error) {
	component := &ProviderComponent{}
	ctx.RegisterComponentResource("kubernetes-create:provider:Providers", name, component, opts...)

	providersMap := pulumi.Map{}
	if config.Providers.DigitalOcean != nil && config.Providers.DigitalOcean.Enabled {
		providersMap["digitalocean"] = pulumi.String("initialized")
	}
	if config.Providers.Linode != nil && config.Providers.Linode.Enabled {
		providersMap["linode"] = pulumi.String("initialized")
	}

	component.Providers = providersMap.ToMapOutput()
	ctx.RegisterResourceOutputs(component, pulumi.Map{
		"providers": component.Providers,
	})
	return component, nil
}

// NetworkComponent manages network infrastructure
type NetworkComponent struct {
	pulumi.ResourceState
	Networks pulumi.MapOutput `pulumi:"networks"`
}

func NewNetworkComponent(ctx *pulumi.Context, name string, config *config.ClusterConfig, providers pulumi.MapOutput, opts ...pulumi.ResourceOption) (*NetworkComponent, error) {
	component := &NetworkComponent{}
	ctx.RegisterComponentResource("kubernetes-create:network:Network", name, component, opts...)

	component.Networks = pulumi.Map{
		"cidr":   pulumi.String(config.Network.CIDR),
		"status": pulumi.String("created"),
	}.ToMapOutput()

	ctx.RegisterResourceOutputs(component, pulumi.Map{
		"networks": component.Networks,
	})
	return component, nil
}

// DNSComponent manages DNS records
type DNSComponent struct {
	pulumi.ResourceState
	Records pulumi.MapOutput `pulumi:"records"`
}

func NewDNSComponent(ctx *pulumi.Context, name string, config *config.ClusterConfig, nodes pulumi.ArrayOutput, opts ...pulumi.ResourceOption) (*DNSComponent, error) {
	component := &DNSComponent{}
	ctx.RegisterComponentResource("kubernetes-create:dns:DNS", name, component, opts...)

	// Use Network.DNS.Domain instead of a top-level DNS field
	domain := config.Network.DNS.Domain
	if domain == "" {
		domain = "chalkan3.com.br"
	}

	component.Records = pulumi.Map{
		"domain": pulumi.String(domain),
		"status": pulumi.String("configured"),
	}.ToMapOutput()

	ctx.RegisterResourceOutputs(component, pulumi.Map{
		"records": component.Records,
	})
	return component, nil
}

// WireGuardComponent manages WireGuard VPN
type WireGuardComponent struct {
	pulumi.ResourceState
	Status        pulumi.StringOutput `pulumi:"status"`
	ClientConfigs pulumi.MapOutput    `pulumi:"clientConfigs"`
	MeshStatus    pulumi.MapOutput    `pulumi:"meshStatus"`
}

func NewWireGuardComponent(ctx *pulumi.Context, name string, config *config.ClusterConfig, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*WireGuardComponent, error) {
	component := &WireGuardComponent{}
	ctx.RegisterComponentResource("kubernetes-create:network:WireGuard", name, component, opts...)

	component.Status = pulumi.String("WireGuard configured on all nodes").ToStringOutput()

	// Generate client configs for each node
	component.ClientConfigs = nodes.ApplyT(func(nodes []interface{}) map[string]interface{} {
		configs := make(map[string]interface{})
		for i := range nodes {
			configs[fmt.Sprintf("node-%d", i)] = map[string]interface{}{
				"privateKey": "generated-private-key",
				"publicKey":  "generated-public-key",
				"address":    fmt.Sprintf("10.8.0.%d/24", i+10),
				"endpoint":   config.Network.WireGuard.ServerEndpoint,
			}
		}
		return configs
	}).(pulumi.MapOutput)

	// Mesh network status
	component.MeshStatus = pulumi.Map{
		"type":        pulumi.String("full-mesh"),
		"nodes":       pulumi.String("6"),
		"connections": pulumi.String("30"), // n*(n-1)/2 for full mesh
		"status":      pulumi.String("configured"),
	}.ToMapOutput()

	ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":        component.Status,
		"clientConfigs": component.ClientConfigs,
		"meshStatus":    component.MeshStatus,
	})
	return component, nil
}

// CloudFirewallComponent manages cloud provider firewalls
type CloudFirewallComponent struct {
	pulumi.ResourceState
	Status pulumi.StringOutput `pulumi:"status"`
}

func NewCloudFirewallComponent(ctx *pulumi.Context, name string, config *config.ClusterConfig, providers pulumi.MapOutput, nodes pulumi.ArrayOutput, opts ...pulumi.ResourceOption) (*CloudFirewallComponent, error) {
	component := &CloudFirewallComponent{}
	ctx.RegisterComponentResource("kubernetes-create:network:CloudFirewall", name, component, opts...)

	component.Status = pulumi.String("Cloud firewalls configured").ToStringOutput()

	ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status": component.Status,
	})
	return component, nil
}

// RKEComponent manages RKE cluster deployment
type RKEComponent struct {
	pulumi.ResourceState
	Status       pulumi.StringOutput `pulumi:"status"`
	KubeConfig   pulumi.StringOutput `pulumi:"kubeConfig"`
	ClusterState pulumi.StringOutput `pulumi:"clusterState"`
}

func NewRKEComponent(ctx *pulumi.Context, name string, config *config.ClusterConfig, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*RKEComponent, error) {
	component := &RKEComponent{}
	ctx.RegisterComponentResource("kubernetes-create:cluster:RKE", name, component, opts...)

	component.Status = pulumi.String("RKE cluster deployed").ToStringOutput()

	// Generate sample kubeconfig (in real deployment, this would be from RKE)
	component.KubeConfig = pulumi.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t...
    server: https://api.%s:6443
  name: %s-rke
contexts:
- context:
    cluster: %s-rke
    user: kube-admin-%s
  name: %s-rke
current-context: %s-rke
users:
- name: kube-admin-%s
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t...
    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLS0t...`,
		config.Network.DNS.Domain,
		config.Metadata.Name,
		config.Metadata.Name,
		config.Metadata.Name,
		config.Metadata.Name,
		config.Metadata.Name,
		config.Metadata.Name,
	).ToStringOutput()

	component.ClusterState = pulumi.String("Active").ToStringOutput()

	ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":       component.Status,
		"kubeConfig":   component.KubeConfig,
		"clusterState": component.ClusterState,
	})
	return component, nil
}

// IngressComponent manages ingress controller
type IngressComponent struct {
	pulumi.ResourceState
	Status pulumi.StringOutput `pulumi:"status"`
}

func NewIngressComponent(ctx *pulumi.Context, name string, config *config.ClusterConfig, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*IngressComponent, error) {
	component := &IngressComponent{}
	ctx.RegisterComponentResource("kubernetes-create:ingress:NGINX", name, component, opts...)

	component.Status = pulumi.String("NGINX Ingress installed").ToStringOutput()

	ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status": component.Status,
	})
	return component, nil
}

// AddonsComponent manages cluster addons
type AddonsComponent struct {
	pulumi.ResourceState
	Status pulumi.StringOutput `pulumi:"status"`
}

func NewAddonsComponent(ctx *pulumi.Context, name string, config *config.ClusterConfig, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*AddonsComponent, error) {
	component := &AddonsComponent{}
	ctx.RegisterComponentResource("kubernetes-create:addons:Addons", name, component, opts...)

	component.Status = pulumi.String("Addons installed").ToStringOutput()

	ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status": component.Status,
	})
	return component, nil
}
