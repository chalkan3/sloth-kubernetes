package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"sloth-kubernetes/internal/orchestrator/components"
	"sloth-kubernetes/pkg/config"
)

// SimpleRealOrchestratorComponent orchestrates REAL cluster with WireGuard, RKE2, and DNS
type SimpleRealOrchestratorComponent struct {
	pulumi.ResourceState

	ClusterName   pulumi.StringOutput `pulumi:"clusterName"`
	KubeConfig    pulumi.StringOutput `pulumi:"kubeConfig"`
	SSHPrivateKey pulumi.StringOutput `pulumi:"sshPrivateKey"`
	SSHPublicKey  pulumi.StringOutput `pulumi:"sshPublicKey"`
	APIEndpoint   pulumi.StringOutput `pulumi:"apiEndpoint"`
	Status        pulumi.StringOutput `pulumi:"status"`
}

// NewSimpleRealOrchestratorComponent creates a simple orchestrator with REAL implementations only
func NewSimpleRealOrchestratorComponent(ctx *pulumi.Context, name string, cfg *config.ClusterConfig, opts ...pulumi.ResourceOption) (*SimpleRealOrchestratorComponent, error) {
	component := &SimpleRealOrchestratorComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:orchestrator:SimpleReal", name, component, opts...)
	if err != nil {
		return nil, err
	}

	ctx.Log.Info("🚀 Starting REAL Kubernetes deployment (WireGuard + RKE2 + DNS)", nil)

	// Phase 1: SSH Keys
	ctx.Log.Info("🔑 Phase 1: Generating SSH keys...", nil)
	sshKeyComponent, err := components.NewSSHKeyComponent(ctx, fmt.Sprintf("%s-ssh-keys", name), cfg, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH keys: %w", err)
	}

	// Phase 1.5: Bastion Host (if enabled)
	// CRITICAL: Bastion must be FULLY provisioned and validated BEFORE any node creation
	var bastionComponent *components.BastionComponent
	var vpcComponent *components.VPCComponent
	var nodeDependencies []pulumi.Resource

	if cfg.Security.Bastion != nil && cfg.Security.Bastion.Enabled {
		ctx.Log.Info("", nil)
		ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
		ctx.Log.Info("🏰 Phase 1.5: BASTION HOST PROVISIONING", nil)
		ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
		ctx.Log.Info("⚠️  IMPORTANT: Nodes will ONLY be created AFTER bastion is 100% validated", nil)
		ctx.Log.Info("", nil)

		bastionComponent, err = components.NewBastionComponent(
			ctx,
			fmt.Sprintf("%s-bastion", name),
			cfg.Security.Bastion,
			sshKeyComponent.PublicKey,
			sshKeyComponent.PrivateKey,
			pulumi.String(cfg.Providers.DigitalOcean.Token),
			pulumi.String(cfg.Providers.Linode.Token),
			pulumi.Parent(component),
			pulumi.DependsOn([]pulumi.Resource{sshKeyComponent}),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create bastion: %w", err)
		}

		ctx.Log.Info("", nil)
		ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
		ctx.Log.Info("✅ BASTION PROVISIONING COMPLETE AND VALIDATED", nil)
		ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
		ctx.Log.Info("", nil)
		ctx.Log.Info("📋 Now proceeding to cluster node creation...", nil)
		ctx.Log.Info("", nil)

		// CRITICAL: Add bastion to dependencies so nodes wait for it
		nodeDependencies = append(nodeDependencies, bastionComponent)

		// NOTE: VPC creation is handled per-provider in the YAML configuration
		// The per-provider VPC configuration (providers.digitalocean.vpc, providers.linode.vpc)
		// is more flexible for multi-cloud deployments
		// This component-based VPC creation is commented out to avoid conflicts
		/*
		// Phase 1.6: Create VPC for private networking
		ctx.Log.Info("🌐 Phase 1.6: Creating VPC for private cluster networking...", nil)
		// Use first node's region for VPC (or bastion region if no nodes)
		vpcRegion := cfg.Security.Bastion.Region
		if len(cfg.Nodes) > 0 {
			vpcRegion = cfg.Nodes[0].Region
		}
		vpcComponent, err = components.NewVPCComponent(
			ctx,
			fmt.Sprintf("%s-vpc", name),
			vpcRegion,
			"10.0.0.0/16", // Private network range
			pulumi.Parent(component),
			pulumi.DependsOn([]pulumi.Resource{sshKeyComponent}),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create VPC: %w", err)
		}
		ctx.Log.Info("✅ VPC created for private networking", nil)
		*/
	} else {
		// No bastion - nodes can start immediately after SSH keys
		nodeDependencies = append(nodeDependencies, sshKeyComponent)
	}

	// Phase 2: Node Deployment (real VMs - private if bastion enabled)
	ctx.Log.Info("", nil)
	ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
	ctx.Log.Info("💻 Phase 2: CLUSTER NODE CREATION", nil)
	ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
	ctx.Log.Info("", nil)

	nodeComponent, realNodes, err := components.NewRealNodeDeploymentComponent(
		ctx,
		fmt.Sprintf("%s-nodes", name),
		cfg,
		sshKeyComponent.PublicKey,
		sshKeyComponent.PrivateKey,
		pulumi.String(cfg.Providers.DigitalOcean.Token),
		pulumi.String(cfg.Providers.Linode.Token),
		vpcComponent,     // Pass VPC component (nil if bastion disabled)
		bastionComponent, // Pass bastion for ProxyJump SSH connections
		pulumi.Parent(component),
		pulumi.DependsOn(nodeDependencies), // WAIT for bastion to be validated (or SSH keys if no bastion)
	)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy nodes: %w", err)
	}

	ctx.Log.Info(fmt.Sprintf("✅ Created %d real nodes", len(realNodes)), nil)

	// Phase 3: WireGuard Mesh VPN (REAL) - includes bastion if enabled
	ctx.Log.Info("🔐 Phase 3: Setting up WireGuard mesh VPN...", nil)

	// Build dependency list - must wait for bastion provisioning if enabled
	// CRITICAL: WireGuard mesh setup generates keys on bastion, so bastion must have
	// WireGuard installed BEFORE we attempt keygen. Wait for bastion provisioning to complete.
	var wgDependencies []pulumi.Resource
	wgDependencies = append(wgDependencies, nodeComponent)
	if bastionComponent != nil {
		ctx.Log.Info("🏰 WireGuard mesh will wait for bastion provisioning to complete...", nil)
		wgDependencies = append(wgDependencies, bastionComponent)
	}

	wgComponent, err := components.NewWireGuardMeshComponent(
		ctx,
		fmt.Sprintf("%s-wireguard", name),
		realNodes,
		sshKeyComponent.PrivateKey,
		bastionComponent, // Pass bastion to be included in VPN mesh
		pulumi.Parent(component),
		pulumi.DependsOn(wgDependencies),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup WireGuard: %w", err)
	}

	ctx.Log.Info("✅ WireGuard mesh VPN configured", nil)

	// Phase 4: RKE2 Kubernetes Cluster (REAL)
	ctx.Log.Info("☸️  Phase 4: Installing RKE2 Kubernetes cluster...", nil)
	rkeComponent, err := components.NewRKE2RealComponent(
		ctx,
		fmt.Sprintf("%s-rke2", name),
		realNodes,
		sshKeyComponent.PrivateKey,
		cfg,
		bastionComponent, // Pass bastion for ProxyJump SSH connections
		pulumi.Parent(component),
		pulumi.DependsOn([]pulumi.Resource{wgComponent}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to install RKE2: %w", err)
	}

	ctx.Log.Info("✅ RKE2 cluster installed", nil)

	// Phase 5: DNS Records (REAL)
	ctx.Log.Info("🌐 Phase 5: Creating DNS records...", nil)
	dnsComponent, err := components.NewDNSRealComponent(
		ctx,
		fmt.Sprintf("%s-dns", name),
		cfg.Network.DNS.Domain,
		realNodes,
		pulumi.Parent(component),
		pulumi.DependsOn([]pulumi.Resource{rkeComponent}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS: %w", err)
	}

	ctx.Log.Info("✅ DNS records created", nil)

	// Set outputs
	component.ClusterName = pulumi.String(cfg.Metadata.Name).ToStringOutput()
	component.KubeConfig = rkeComponent.KubeConfig
	component.SSHPrivateKey = sshKeyComponent.PrivateKeyPath
	component.SSHPublicKey = sshKeyComponent.PublicKey
	component.APIEndpoint = dnsComponent.APIEndpoint
	component.Status = pulumi.String("✅ REAL Kubernetes cluster deployed successfully!").ToStringOutput()

	// Export detailed node information as a structured map for CLI commands
	nodesMap := pulumi.Map{}
	for i, node := range realNodes {
		nodeKey := fmt.Sprintf("node_%d", i)
		nodesMap[nodeKey] = pulumi.Map{
			"name":        node.NodeName,
			"public_ip":   node.PublicIP,
			"private_ip":  node.PrivateIP,
			"vpn_ip":      node.WireGuardIP,
			"provider":    node.Provider,
			"region":      node.Region,
			"size":        node.Size,
			"roles":       node.Roles,
			"status":      node.Status,
		}
	}
	ctx.Export("nodes", nodesMap)
	ctx.Export("node_count", pulumi.Int(len(realNodes)))

	// Export bastion information if enabled
	if bastionComponent != nil {
		ctx.Export("bastion", pulumi.Map{
			"name":       bastionComponent.BastionName,
			"public_ip":  bastionComponent.PublicIP,
			"private_ip": bastionComponent.PrivateIP,
			"vpn_ip":     bastionComponent.WireGuardIP,
			"provider":   bastionComponent.Provider,
			"region":     bastionComponent.Region,
			"ssh_port":   bastionComponent.SSHPort,
			"status":     bastionComponent.Status,
		})
		ctx.Export("bastion_enabled", pulumi.Bool(true))
	} else {
		ctx.Export("bastion_enabled", pulumi.Bool(false))
	}

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"clusterName":   component.ClusterName,
		"kubeConfig":    component.KubeConfig,
		"sshPrivateKey": component.SSHPrivateKey,
		"sshPublicKey":  component.SSHPublicKey,
		"apiEndpoint":   component.APIEndpoint,
		"status":        component.Status,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info("🎉 REAL Kubernetes cluster deployment COMPLETE!", nil)

	return component, nil
}
