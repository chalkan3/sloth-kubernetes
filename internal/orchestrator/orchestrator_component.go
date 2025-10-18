package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"kubernetes-create/pkg/config"
)

// OrchestratorComponent is a Pulumi component that orchestrates the entire cluster
type OrchestratorComponent struct {
	pulumi.ResourceState

	// Exported outputs
	ClusterName    pulumi.StringOutput   `pulumi:"clusterName"`
	Nodes          pulumi.MapOutput      `pulumi:"nodes"`
	VPNStatus      pulumi.StringOutput   `pulumi:"vpnStatus"`
	RKEStatus      pulumi.StringOutput   `pulumi:"rkeStatus"`
	KubeConfig     pulumi.StringOutput   `pulumi:"kubeConfig"`
	SSHPrivateKey  pulumi.StringOutput   `pulumi:"sshPrivateKey"`
	SSHPublicKey   pulumi.StringOutput   `pulumi:"sshPublicKey"`
	WireGuardConfigs pulumi.MapOutput    `pulumi:"wireGuardConfigs"`
	DNSRecords     pulumi.MapOutput      `pulumi:"dnsRecords"`
	LoadBalancers  pulumi.ArrayOutput    `pulumi:"loadBalancers"`
	IngressURL     pulumi.StringOutput   `pulumi:"ingressURL"`
	APIEndpoint    pulumi.StringOutput   `pulumi:"apiEndpoint"`
	ClusterInfo    pulumi.MapOutput      `pulumi:"clusterInfo"`
}

// NewOrchestratorComponent creates a new orchestrator component
func NewOrchestratorComponent(ctx *pulumi.Context, name string, config *config.ClusterConfig, opts ...pulumi.ResourceOption) (*OrchestratorComponent, error) {
	component := &OrchestratorComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:orchestrator:Orchestrator", name, component, opts...)
	if err != nil {
		return nil, err
	}

	// Create child components in sequence

	// Phase 1: SSH Key Generation Component
	sshKeyComponent, err := NewSSHKeyComponent(ctx, fmt.Sprintf("%s-ssh-keys", name), config, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH key component: %w", err)
	}

	// Phase 2: Provider Initialization Component (Granular - separate DO and Linode components)
	providerComponent, err := NewProviderComponentGranular(ctx, fmt.Sprintf("%s-providers", name), config, sshKeyComponent.PublicKey, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{sshKeyComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create provider component: %w", err)
	}

	// Phase 3: Network Infrastructure Component
	networkComponent, err := NewNetworkComponent(ctx, fmt.Sprintf("%s-network", name), config, providerComponent.Providers, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{providerComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create network component: %w", err)
	}

	// Phase 4: Node Deployment Component - REAL CLOUD RESOURCES
	// Pass both public key (for cloud provider) and private key (for provisioning)
	nodeComponent, realNodes, err := NewRealNodeDeploymentComponent(ctx, fmt.Sprintf("%s-nodes", name), config, sshKeyComponent.PublicKey, sshKeyComponent.PrivateKey, pulumi.String(config.Providers.DigitalOcean.Token), pulumi.String(config.Providers.Linode.Token), pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{networkComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create node deployment component: %w", err)
	}

	ctx.Log.Info(fmt.Sprintf("Created %d real nodes for WireGuard and RKE setup", len(realNodes)), nil)

	// Phase 4.5: WireGuard Mesh VPN Setup - REAL IMPLEMENTATION
	// This configures full mesh VPN between all nodes using WireGuard
	ctx.Log.Info("Setting up WireGuard mesh VPN between all nodes...", nil)
	wgMeshComponent, err := NewWireGuardMeshComponent(ctx, fmt.Sprintf("%s-wireguard-mesh", name), realNodes, sshKeyComponent.PrivateKey, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{nodeComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create WireGuard mesh component: %w", err)
	}
	ctx.Log.Info("WireGuard mesh VPN configured successfully", nil)

	// Phase 5: Node Health Check Component (Granular - individual health checks per node)
	// Pass the PRIVATE KEY CONTENT (not path) for remote SSH provisioning
	healthCheckComponent, err := NewHealthCheckComponentGranular(ctx, fmt.Sprintf("%s-health-check", name), nodeComponent.Nodes, sshKeyComponent.PrivateKey, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{nodeComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create health check component: %w", err)
	}

	// Phase 6: OS Firewall Configuration Component (Granular - individual rules per node)
	osFirewallComponent, err := NewOSFirewallComponentGranular(ctx, fmt.Sprintf("%s-os-firewall", name), nodeComponent.Nodes, sshKeyComponent.PrivateKeyPath, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{healthCheckComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create OS firewall component: %w", err)
	}

	// Phase 7: DNS Configuration Component (Granular - individual record components)
	dnsComponent, err := NewDNSComponentGranular(ctx, fmt.Sprintf("%s-dns", name), config, nodeComponent.Nodes, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{osFirewallComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS component: %w", err)
	}

	// Phase 8: WireGuard VPN Component (Granular - individual peer and tunnel components)
	wireGuardComponent, err := NewWireGuardComponentGranular(ctx, fmt.Sprintf("%s-wireguard", name), config, nodeComponent.Nodes, sshKeyComponent.PrivateKeyPath, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{dnsComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create WireGuard component: %w", err)
	}

	// Phase 9: Cloud Provider Firewall Component
	cloudFirewallComponent, err := NewCloudFirewallComponent(ctx, fmt.Sprintf("%s-cloud-firewall", name), config, providerComponent.Providers, nodeComponent.Nodes, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{wireGuardComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud firewall component: %w", err)
	}

	// Phase 10: VPN Connectivity Verification Component (Granular - ping tests for each node pair)
	vpnVerifyComponent, err := NewVPNVerificationComponentGranular(ctx, fmt.Sprintf("%s-vpn-verify", name), nodeComponent.Nodes, sshKeyComponent.PrivateKeyPath, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{cloudFirewallComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create VPN verification component: %w", err)
	}

	// Phase 11: RKE Deployment Component (Granular - separate master/worker components)
	rkeComponent, err := NewRKEComponentGranular(ctx, fmt.Sprintf("%s-rke", name), config, nodeComponent.Nodes, sshKeyComponent.PrivateKeyPath, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{vpnVerifyComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create RKE component: %w", err)
	}

	// Phase 12: Ingress Installation Component (Granular - controller, class, cert-manager)
	ingressComponent, err := NewIngressComponentGranular(ctx, fmt.Sprintf("%s-ingress", name), config, nodeComponent.Nodes, sshKeyComponent.PrivateKeyPath, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{rkeComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create ingress component: %w", err)
	}

	// Phase 13: Addons Installation Component (Granular - individual addon components)
	_, err = NewAddonsComponentGranular(ctx, fmt.Sprintf("%s-addons", name), config, nodeComponent.Nodes, sshKeyComponent.PrivateKeyPath, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{ingressComponent}))
	if err != nil {
		return nil, fmt.Errorf("failed to create addons component: %w", err)
	}

	// Set basic outputs
	component.ClusterName = pulumi.String(config.Metadata.Name).ToStringOutput()
	component.VPNStatus = vpnVerifyComponent.Status
	component.RKEStatus = rkeComponent.Status

	// Set SSH key outputs
	component.SSHPrivateKey = sshKeyComponent.PrivateKeyPath
	component.SSHPublicKey = sshKeyComponent.PublicKey

	// Set node information with details
	component.Nodes = nodeComponent.Nodes.ApplyT(func(nodes []interface{}) map[string]interface{} {
		nodeMap := make(map[string]interface{})
		for i, node := range nodes {
			nodeMap[fmt.Sprintf("node-%d", i)] = node
		}
		return map[string]interface{}{
			"count": len(nodes),
			"nodes": nodeMap,
		}
	}).(pulumi.MapOutput)

	// Set WireGuard configurations with detailed info
	component.WireGuardConfigs = pulumi.All(
		wireGuardComponent.Status,
		wireGuardComponent.ClientConfigs,
		wireGuardComponent.MeshStatus,
		wgMeshComponent.Status,
		wgMeshComponent.PeerCount,
		wgMeshComponent.TunnelCount,
	).ApplyT(func(args []interface{}) map[string]interface{} {
		return map[string]interface{}{
			"status": args[0],
			"endpoint": config.Network.WireGuard.ServerEndpoint,
			"port": config.Network.WireGuard.Port,
			"publicKey": config.Network.WireGuard.ServerPublicKey,
			"clientConfigs": args[1],
			"meshStatus": args[2],
			"realMeshStatus": args[3],
			"realMeshPeerCount": args[4],
			"realMeshTunnelCount": args[5],
		}
	}).(pulumi.MapOutput)

	// Set DNS records
	component.DNSRecords = dnsComponent.Records

	// Set load balancers (empty for now as per config)
	component.LoadBalancers = pulumi.ToArrayOutput([]pulumi.Output{})

	// Set ingress URL
	component.IngressURL = pulumi.Sprintf("https://kube-ingress.%s", config.Network.DNS.Domain)

	// Set API endpoint
	component.APIEndpoint = pulumi.Sprintf("https://api.%s:6443", config.Network.DNS.Domain)

	// Use kubeconfig from RKE component
	component.KubeConfig = rkeComponent.KubeConfig

	// Set comprehensive cluster information
	component.ClusterInfo = pulumi.ToMap(map[string]interface{}{
		"name": config.Metadata.Name,
		"environment": config.Metadata.Environment,
		"version": config.Cluster.Version,
		"kubernetes_version": config.Kubernetes.Version,
		"network_plugin": config.Kubernetes.NetworkPlugin,
		"pod_cidr": config.Kubernetes.PodCIDR,
		"service_cidr": config.Kubernetes.ServiceCIDR,
		"cluster_dns": config.Kubernetes.ClusterDNS,
		"high_availability": config.Cluster.HighAvailability,
		"providers": map[string]interface{}{
			"digitalocean": config.Providers.DigitalOcean != nil && config.Providers.DigitalOcean.Enabled,
			"linode": config.Providers.Linode != nil && config.Providers.Linode.Enabled,
		},
		"wireguard_enabled": config.Network.WireGuard != nil && config.Network.WireGuard.Enabled,
		"monitoring_enabled": config.Monitoring.Enabled,
		"ingress_controller": "nginx",
		"storage_class": config.Storage.DefaultClass,
		"total_nodes": 6,
		"master_nodes": 3,
		"worker_nodes": 3,
	}).ToMapOutput()

	// Register all outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"clusterName": component.ClusterName,
		"nodes": component.Nodes,
		"vpnStatus": component.VPNStatus,
		"rkeStatus": component.RKEStatus,
		"kubeConfig": component.KubeConfig,
		"sshPrivateKey": component.SSHPrivateKey,
		"sshPublicKey": component.SSHPublicKey,
		"wireGuardConfigs": component.WireGuardConfigs,
		"dnsRecords": component.DNSRecords,
		"loadBalancers": component.LoadBalancers,
		"ingressURL": component.IngressURL,
		"apiEndpoint": component.APIEndpoint,
		"clusterInfo": component.ClusterInfo,
	}); err != nil {
		return nil, err
	}

	return component, nil
}