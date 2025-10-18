package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"kubernetes-create/pkg/config"
)

// SimpleRealOrchestratorComponent orchestrates REAL cluster with WireGuard, RKE2, and DNS
type SimpleRealOrchestratorComponent struct {
	pulumi.ResourceState

	ClusterName    pulumi.StringOutput `pulumi:"clusterName"`
	KubeConfig     pulumi.StringOutput `pulumi:"kubeConfig"`
	SSHPrivateKey  pulumi.StringOutput `pulumi:"sshPrivateKey"`
	SSHPublicKey   pulumi.StringOutput `pulumi:"sshPublicKey"`
	APIEndpoint    pulumi.StringOutput `pulumi:"apiEndpoint"`
	Status         pulumi.StringOutput `pulumi:"status"`
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
	sshKeyComponent, err := NewSSHKeyComponent(ctx, fmt.Sprintf("%s-ssh-keys", name), cfg, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH keys: %w", err)
	}

	// Phase 2: Node Deployment (6 real VMs)
	ctx.Log.Info("💻 Phase 2: Creating 6 real cloud VMs...", nil)
	nodeComponent, realNodes, err := NewRealNodeDeploymentComponent(
		ctx,
		fmt.Sprintf("%s-nodes", name),
		cfg,
		sshKeyComponent.PublicKey,
		sshKeyComponent.PrivateKey,
		pulumi.String(cfg.Providers.DigitalOcean.Token),
		pulumi.String(cfg.Providers.Linode.Token),
		pulumi.Parent(component),
		pulumi.DependsOn([]pulumi.Resource{sshKeyComponent}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy nodes: %w", err)
	}

	ctx.Log.Info(fmt.Sprintf("✅ Created %d real nodes", len(realNodes)), nil)

	// Phase 3: WireGuard Mesh VPN (REAL)
	ctx.Log.Info("🔐 Phase 3: Setting up WireGuard mesh VPN...", nil)
	wgComponent, err := NewWireGuardMeshComponent(
		ctx,
		fmt.Sprintf("%s-wireguard", name),
		realNodes,
		sshKeyComponent.PrivateKey,
		pulumi.Parent(component),
		pulumi.DependsOn([]pulumi.Resource{nodeComponent}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup WireGuard: %w", err)
	}

	ctx.Log.Info("✅ WireGuard mesh VPN configured", nil)

	// Phase 4: RKE2 Kubernetes Cluster (REAL)
	ctx.Log.Info("☸️  Phase 4: Installing RKE2 Kubernetes cluster...", nil)
	rkeComponent, err := NewRKE2RealComponent(
		ctx,
		fmt.Sprintf("%s-rke2", name),
		realNodes,
		sshKeyComponent.PrivateKey,
		pulumi.Parent(component),
		pulumi.DependsOn([]pulumi.Resource{wgComponent}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to install RKE2: %w", err)
	}

	ctx.Log.Info("✅ RKE2 cluster installed", nil)

	// Phase 5: DNS Records (REAL)
	ctx.Log.Info("🌐 Phase 5: Creating DNS records...", nil)
	dnsComponent, err := NewDNSRealComponent(
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
