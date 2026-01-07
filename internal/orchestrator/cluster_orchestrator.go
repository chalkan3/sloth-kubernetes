package orchestrator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chalkan3/sloth-kubernetes/internal/orchestrator/components"
	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	"github.com/chalkan3/sloth-kubernetes/pkg/versioning"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// DeploymentMetadata contains timestamps and tracking info for scale operations
type DeploymentMetadata struct {
	// Timestamps
	CreatedAt      string `json:"createdAt"`      // First deployment timestamp
	UpdatedAt      string `json:"updatedAt"`      // Last update timestamp
	LastDeployedAt string `json:"lastDeployedAt"` // Current deployment timestamp

	// Deployment tracking
	DeploymentID    string `json:"deploymentId"`    // Unique ID for this deployment
	DeploymentCount int    `json:"deploymentCount"` // Number of times deployed/updated

	// Scale operation tracking
	LastScaleOperation string `json:"lastScaleOperation,omitempty"` // "scale-up", "scale-down", "initial"
	PreviousNodeCount  int    `json:"previousNodeCount"`            // Node count before this operation
	CurrentNodeCount   int    `json:"currentNodeCount"`             // Current node count

	// Node pool changes
	NodePoolsAdded   []string `json:"nodePoolsAdded,omitempty"`   // Pools added in this deployment
	NodePoolsRemoved []string `json:"nodePoolsRemoved,omitempty"` // Pools removed in this deployment
	NodePoolsScaled  []string `json:"nodePoolsScaled,omitempty"`  // Pools with count changes

	// Version info
	SlothVersion   string `json:"slothVersion"`   // Sloth-kubernetes version
	PulumiVersion  string `json:"pulumiVersion"`  // Pulumi version used
	ConfigChecksum string `json:"configChecksum"` // SHA256 of config for change detection

	// Enhanced versioning (new fields)
	ConfigVersion *versioning.ConfigVersion `json:"configVersion,omitempty"` // Full config version info with migrations
	SchemaVersion string                    `json:"schemaVersion,omitempty"` // Current schema version (e.g., "2.0")

	// Manifest tracking (new fields)
	ManifestCount    int                    `json:"manifestCount,omitempty"`    // Number of tracked manifests
	ManifestRegistry *ManifestRegistryState `json:"manifestRegistry,omitempty"` // Serialized manifest registry state
	ManifestHashes   map[string]string      `json:"manifestHashes,omitempty"`   // Map of manifest name to hash

	// Deployment history (new fields)
	PreviousDeployments []DeploymentHistoryEntry `json:"previousDeployments,omitempty"` // History of past deployments
	RollbackTarget      string                   `json:"rollbackTarget,omitempty"`      // Target deployment ID for rollback
	RollbackReason      string                   `json:"rollbackReason,omitempty"`      // Reason for rollback if applicable

	// Change tracking (new fields)
	ChangeLog     []ChangeLogEntry `json:"changeLog,omitempty"`     // Detailed change log
	DriftDetected bool             `json:"driftDetected,omitempty"` // Whether drift was detected from desired state
	DriftDetails  string           `json:"driftDetails,omitempty"`  // Details about detected drift

	// State management (new fields)
	StateSnapshotID string `json:"stateSnapshotId,omitempty"` // ID of the state snapshot for this deployment
	ParentStateID   string `json:"parentStateId,omitempty"`   // ID of the parent state snapshot
}

// ManifestRegistryState holds serialized manifest registry information
type ManifestRegistryState struct {
	Version    string            `json:"version"`
	Hash       string            `json:"hash"`
	Manifests  []ManifestSummary `json:"manifests,omitempty"`
	ExportedAt string            `json:"exportedAt"`
}

// ManifestSummary provides a summary of a manifest in the registry
type ManifestSummary struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
	Status  string `json:"status"`
}

// DeploymentHistoryEntry represents a historical deployment record
type DeploymentHistoryEntry struct {
	DeploymentID   string `json:"deploymentId"`
	Timestamp      string `json:"timestamp"`
	NodeCount      int    `json:"nodeCount"`
	ConfigChecksum string `json:"configChecksum"`
	SchemaVersion  string `json:"schemaVersion,omitempty"`
	Success        bool   `json:"success"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
}

// ChangeLogEntry represents a single change in the deployment
type ChangeLogEntry struct {
	Timestamp   string `json:"timestamp"`
	ChangeType  string `json:"changeType"`  // "node_added", "node_removed", "config_updated", "manifest_applied", etc.
	ResourceID  string `json:"resourceId"`  // ID of the affected resource
	Description string `json:"description"` // Human-readable description
	OldValue    string `json:"oldValue,omitempty"`
	NewValue    string `json:"newValue,omitempty"`
	Actor       string `json:"actor,omitempty"` // Who/what initiated the change
}

// SimpleRealOrchestratorComponent orchestrates REAL cluster with WireGuard, RKE2, and DNS
type SimpleRealOrchestratorComponent struct {
	pulumi.ResourceState

	ClusterName   pulumi.StringOutput `pulumi:"clusterName"`
	KubeConfig    pulumi.StringOutput `pulumi:"kubeConfig"`
	SSHPrivateKey pulumi.StringOutput `pulumi:"sshPrivateKey"`
	SSHPublicKey  pulumi.StringOutput `pulumi:"sshPublicKey"`
	APIEndpoint   pulumi.StringOutput `pulumi:"apiEndpoint"`
	Status        pulumi.StringOutput `pulumi:"status"`

	// Manifest storage for config regeneration
	ConfigJSON   pulumi.StringOutput `pulumi:"configJson"`
	LispManifest pulumi.StringOutput `pulumi:"lispManifest"`

	// Deployment metadata for scale tracking
	DeploymentMeta pulumi.StringOutput `pulumi:"deploymentMeta"`
}

// NewSimpleRealOrchestratorComponent creates a simple orchestrator with REAL implementations only
// lispManifest is the raw Lisp configuration file content for storage in Pulumi state
// previousMeta is the previous deployment metadata (empty string for initial deployment)
func NewSimpleRealOrchestratorComponent(ctx *pulumi.Context, name string, cfg *config.ClusterConfig, lispManifest string, previousMeta string, opts ...pulumi.ResourceOption) (*SimpleRealOrchestratorComponent, error) {
	component := &SimpleRealOrchestratorComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:orchestrator:SimpleReal", name, component, opts...)
	if err != nil {
		return nil, err
	}

	ctx.Log.Info("🚀 Starting REAL Kubernetes deployment (WireGuard + K3s + DNS)", nil)

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

	// DEBUG: Check bastion configuration
	if cfg.Security.Bastion == nil {
		ctx.Log.Info("🔍 DEBUG: cfg.Security.Bastion is NIL", nil)
	} else {
		ctx.Log.Info(fmt.Sprintf("🔍 DEBUG: cfg.Security.Bastion.Enabled = %v", cfg.Security.Bastion.Enabled), nil)
		ctx.Log.Info(fmt.Sprintf("🔍 DEBUG: cfg.Security.Bastion.Provider = %s", cfg.Security.Bastion.Provider), nil)
		ctx.Log.Info(fmt.Sprintf("🔍 DEBUG: cfg.Security.Bastion.Name = %s", cfg.Security.Bastion.Name), nil)
	}

	if cfg.Security.Bastion != nil && cfg.Security.Bastion.Enabled {
		ctx.Log.Info("", nil)
		ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
		ctx.Log.Info("🏰 Phase 1.5: BASTION HOST PROVISIONING", nil)
		ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
		ctx.Log.Info("⚠️  IMPORTANT: Nodes will ONLY be created AFTER bastion is 100% validated", nil)
		ctx.Log.Info("", nil)

		// Safely get provider tokens (empty if provider is not configured)
		doToken := ""
		if cfg.Providers.DigitalOcean != nil {
			doToken = cfg.Providers.DigitalOcean.Token
		}
		linodeToken := ""
		if cfg.Providers.Linode != nil {
			linodeToken = cfg.Providers.Linode.Token
		}

		bastionComponent, err = components.NewBastionComponent(
			ctx,
			fmt.Sprintf("%s-bastion", name),
			cfg.Security.Bastion,
			sshKeyComponent.PublicKey,
			sshKeyComponent.PrivateKey,
			pulumi.String(doToken),
			pulumi.String(linodeToken),
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

	// Safely get provider tokens (empty if provider is not configured)
	doTokenForNodes := ""
	if cfg.Providers.DigitalOcean != nil {
		doTokenForNodes = cfg.Providers.DigitalOcean.Token
	}
	linodeTokenForNodes := ""
	if cfg.Providers.Linode != nil {
		linodeTokenForNodes = cfg.Providers.Linode.Token
	}

	nodeComponent, realNodes, err := components.NewRealNodeDeploymentComponent(
		ctx,
		fmt.Sprintf("%s-nodes", name),
		cfg,
		sshKeyComponent.PublicKey,
		sshKeyComponent.PrivateKey,
		pulumi.String(doTokenForNodes),
		pulumi.String(linodeTokenForNodes),
		vpcComponent,     // Pass VPC component (nil if bastion disabled)
		bastionComponent, // Pass bastion for ProxyJump SSH connections
		pulumi.Parent(component),
		pulumi.DependsOn(nodeDependencies), // WAIT for bastion to be validated (or SSH keys if no bastion)
	)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy nodes: %w", err)
	}

	ctx.Log.Info(fmt.Sprintf("✅ Created %d real nodes", len(realNodes)), nil)

	// Phase 2.5: Cloud-init Validation (wait for Docker + WireGuard to be installed)
	ctx.Log.Info("", nil)
	ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
	ctx.Log.Info("🔍 Phase 2.5: CLOUD-INIT VALIDATION", nil)
	ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
	ctx.Log.Info("⏳ Waiting for cloud-init to complete (Docker + WireGuard installation)...", nil)
	ctx.Log.Info("", nil)

	cloudInitValidator, err := components.NewCloudInitValidatorComponent(
		ctx,
		fmt.Sprintf("%s-cloudinit-validator", name),
		realNodes,
		sshKeyComponent.PrivateKey,
		bastionComponent,
		pulumi.Parent(component),
		pulumi.DependsOn([]pulumi.Resource{nodeComponent}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to validate cloud-init: %w", err)
	}

	ctx.Log.Info("✅ Cloud-init validation passed - Docker and WireGuard installed on all nodes", nil)

	// Phase 3: WireGuard Mesh VPN (REAL) - includes bastion if enabled
	ctx.Log.Info("", nil)
	ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
	ctx.Log.Info("🔐 Phase 3: WIREGUARD MESH VPN CONFIGURATION", nil)
	ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
	ctx.Log.Info("", nil)

	// Build dependency list - must wait for cloud-init validation
	// CRITICAL: WireGuard must be installed (via cloud-init) before we configure the mesh
	var wgDependencies []pulumi.Resource
	wgDependencies = append(wgDependencies, cloudInitValidator)
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

	// Phase 3.5: Validate VPN connectivity before RKE2
	ctx.Log.Info("🔍 Phase 3.5: Validating VPN connectivity...", nil)
	vpnValidator, err := components.NewVPNValidatorComponent(
		ctx,
		fmt.Sprintf("%s-vpn-validator", name),
		realNodes,
		sshKeyComponent.PrivateKey,
		bastionComponent,
		pulumi.Parent(component),
		pulumi.DependsOn([]pulumi.Resource{wgComponent}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to validate VPN: %w", err)
	}

	ctx.Log.Info("✅ VPN validation passed - all nodes reachable", nil)

	// Phase 4: Kubernetes Cluster Installation (K3s or RKE2)
	var kubeConfig pulumi.StringOutput
	var clusterInstallResource pulumi.Resource

	distribution := cfg.Kubernetes.Distribution
	if distribution == "" {
		distribution = "k3s" // Default to K3s
	}

	if distribution == "rke2" {
		// Install RKE2
		ctx.Log.Info("☸️  Phase 4: Installing RKE2 Kubernetes cluster...", nil)
		rke2Component, err := components.NewRKE2RealComponent(
			ctx,
			fmt.Sprintf("%s-rke2", name),
			realNodes,
			sshKeyComponent.PrivateKey,
			cfg,
			bastionComponent,
			pulumi.Parent(component),
			pulumi.DependsOn([]pulumi.Resource{vpnValidator}),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to install RKE2: %w", err)
		}
		kubeConfig = rke2Component.KubeConfig
		clusterInstallResource = rke2Component
		ctx.Log.Info("✅ RKE2 cluster installed", nil)
	} else {
		// Install K3s (default)
		ctx.Log.Info("☸️  Phase 4: Installing K3s Kubernetes cluster...", nil)
		k3sComponent, err := components.NewK3sRealComponent(
			ctx,
			fmt.Sprintf("%s-k3s", name),
			realNodes,
			sshKeyComponent.PrivateKey,
			cfg,
			bastionComponent,
			pulumi.Parent(component),
			pulumi.DependsOn([]pulumi.Resource{vpnValidator}),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to install K3s: %w", err)
		}
		kubeConfig = k3sComponent.KubeConfig
		clusterInstallResource = k3sComponent
		ctx.Log.Info("✅ K3s cluster installed", nil)
	}

	// Use rkeComponent as alias for compatibility
	rkeComponent := clusterInstallResource
	_ = rkeComponent // Silence unused variable warning

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

	// Phase 5.5: Salt Master Installation (only if enabled in config)
	var saltMasterComponent *components.SaltMasterComponent
	var saltMinionComponent *components.SaltMinionJoinComponent

	if cfg.Addons.Salt != nil && cfg.Addons.Salt.Enabled {
		ctx.Log.Info("", nil)
		ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
		ctx.Log.Info("🧂 Phase 5.5: SALT MASTER INSTALLATION", nil)
		ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
		ctx.Log.Info("🔐 Secure hash-based authentication enabled", nil)
		ctx.Log.Info("", nil)

		// Find the Salt Master node - use first node (typically first master)
		// The MasterNode config can specify an index like "0", "1", etc.
		var saltMasterNode *components.RealNodeComponent
		masterNodeIndex := 0 // Default to first node

		if cfg.Addons.Salt.MasterNode != "" {
			// Parse index if specified (e.g., "1" for second node)
			if idx, err := fmt.Sscanf(cfg.Addons.Salt.MasterNode, "%d", &masterNodeIndex); err != nil || idx != 1 {
				masterNodeIndex = 0 // Default to first
			}
		}

		if len(realNodes) > masterNodeIndex {
			saltMasterNode = realNodes[masterNodeIndex]
		} else if len(realNodes) > 0 {
			saltMasterNode = realNodes[0]
		}

		if saltMasterNode != nil {
			ctx.Log.Info(fmt.Sprintf("📍 Installing Salt Master (API port: %d)...", cfg.Addons.Salt.APIPort), nil)

			saltMasterComponent, err = components.NewSaltMasterComponent(
				ctx,
				fmt.Sprintf("%s-salt-master", name),
				saltMasterNode,
				sshKeyComponent.PrivateKey,
				bastionComponent,
				pulumi.Parent(component),
				pulumi.DependsOn([]pulumi.Resource{clusterInstallResource, dnsComponent}),
			)
			if err != nil {
				ctx.Log.Warn(fmt.Sprintf("⚠️  Salt Master installation failed: %v", err), nil)
				ctx.Log.Warn("   Cluster is ready but Salt Master was not installed", nil)
			} else {
				ctx.Log.Info("✅ Salt Master installed successfully", nil)

				// Phase 5.6: Salt Minion Join (if auto-join enabled)
				if cfg.Addons.Salt.AutoJoin {
					ctx.Log.Info("", nil)
					ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
					ctx.Log.Info("🧂 Phase 5.6: SALT MINION CONFIGURATION", nil)
					ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
					ctx.Log.Info("", nil)

					saltMinionComponent, err = components.NewSaltMinionJoinComponent(
						ctx,
						fmt.Sprintf("%s-salt-minions", name),
						realNodes,
						saltMasterComponent,
						sshKeyComponent.PrivateKey,
						bastionComponent,
						pulumi.Parent(component),
						pulumi.DependsOn([]pulumi.Resource{saltMasterComponent}),
					)
					if err != nil {
						ctx.Log.Warn(fmt.Sprintf("⚠️  Salt Minion configuration failed: %v", err), nil)
					} else {
						ctx.Log.Info("✅ All nodes configured as Salt Minions", nil)
					}
				}
			}
		} else {
			ctx.Log.Warn("⚠️  No node found for Salt Master installation", nil)
		}
	}

	// Phase 6: ArgoCD Installation (if enabled)
	var argoCDComponent *components.ArgoCDInstallerComponent
	if cfg.Addons.ArgoCD != nil && cfg.Addons.ArgoCD.Enabled {
		ctx.Log.Info("", nil)
		ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
		ctx.Log.Info("🚀 Phase 6: ARGOCD GITOPS INSTALLATION", nil)
		ctx.Log.Info("════════════════════════════════════════════════════════════", nil)
		ctx.Log.Info("", nil)

		argoCDComponent, err = components.NewArgoCDInstallerComponent(
			ctx,
			fmt.Sprintf("%s-argocd", name),
			cfg.Addons.ArgoCD,
			realNodes,
			bastionComponent,
			sshKeyComponent.PrivateKey,
			pulumi.Parent(component),
			pulumi.DependsOn([]pulumi.Resource{rkeComponent, dnsComponent}),
		)
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  ArgoCD installation failed: %v", err), nil)
			ctx.Log.Warn("   Cluster is ready but ArgoCD was not installed", nil)
		} else {
			ctx.Log.Info("✅ ArgoCD installed successfully", nil)
		}
	}

	// Set outputs
	component.ClusterName = pulumi.String(cfg.Metadata.Name).ToStringOutput()
	component.KubeConfig = kubeConfig
	component.SSHPrivateKey = sshKeyComponent.PrivateKeyPath
	component.SSHPublicKey = sshKeyComponent.PublicKey
	component.APIEndpoint = dnsComponent.APIEndpoint
	component.Status = pulumi.String("✅ REAL Kubernetes cluster deployed successfully!").ToStringOutput()

	// Store manifest for config regeneration (Pulumi state as database)
	// Create a sanitized copy of config without sensitive data for JSON storage
	sanitizedCfg := sanitizeConfigForStorage(cfg)
	configJSON, err := json.MarshalIndent(sanitizedCfg, "", "  ")
	if err != nil {
		ctx.Log.Warn(fmt.Sprintf("Failed to serialize config to JSON: %v", err), nil)
		configJSON = []byte("{}")
	}
	component.ConfigJSON = pulumi.String(string(configJSON)).ToStringOutput()
	component.LispManifest = pulumi.String(lispManifest).ToStringOutput()

	// Create secret exporter - ALL outputs are encrypted with passphrase
	secretExporter := secrets.NewSecretExporter(ctx)

	// Export manifests for retrieval (encrypted)
	secretExporter.ExportString("configJson", string(configJSON))
	secretExporter.ExportString("lispManifest", lispManifest)

	// Generate deployment metadata for scale tracking
	deployMeta := generateDeploymentMetadata(cfg, previousMeta, len(realNodes), lispManifest)
	deployMetaJSON, err := json.MarshalIndent(deployMeta, "", "  ")
	if err != nil {
		ctx.Log.Warn(fmt.Sprintf("Failed to serialize deployment metadata: %v", err), nil)
		deployMetaJSON = []byte("{}")
	}
	component.DeploymentMeta = pulumi.String(string(deployMetaJSON)).ToStringOutput()
	secretExporter.ExportString("deploymentMeta", string(deployMetaJSON))

	// Export detailed node information as a structured map for CLI commands (encrypted)
	nodesMap := pulumi.Map{}
	for i, node := range realNodes {
		nodeKey := fmt.Sprintf("node_%d", i)
		nodesMap[nodeKey] = pulumi.Map{
			"name":       node.NodeName,
			"public_ip":  node.PublicIP,
			"private_ip": node.PrivateIP,
			"vpn_ip":     node.WireGuardIP,
			"provider":   node.Provider,
			"region":     node.Region,
			"size":       node.Size,
			"roles":      node.Roles,
			"status":     node.Status,
		}
	}
	secretExporter.ExportMap("nodes", nodesMap)
	secretExporter.ExportInt("node_count", len(realNodes))

	// Export kubeconfig for kubectl access (encrypted)
	secretExporter.Export("kubeConfig", kubeConfig)

	// Initialize operations history for CLI operations tracking (encrypted)
	secretExporter.ExportString("operationsHistory", "{}")

	// Export bastion information if enabled (encrypted)
	if bastionComponent != nil {
		secretExporter.ExportMap("bastion", pulumi.Map{
			"name":       bastionComponent.BastionName,
			"public_ip":  bastionComponent.PublicIP,
			"private_ip": bastionComponent.PrivateIP,
			"vpn_ip":     bastionComponent.WireGuardIP,
			"provider":   bastionComponent.Provider,
			"region":     bastionComponent.Region,
			"ssh_port":   bastionComponent.SSHPort,
			"status":     bastionComponent.Status,
		})
		secretExporter.ExportBool("bastion_enabled", true)
	} else {
		secretExporter.ExportBool("bastion_enabled", false)
	}

	// Export ArgoCD information if installed (encrypted - contains admin password)
	if argoCDComponent != nil {
		secretExporter.Export("argocd_admin_password", argoCDComponent.AdminPassword)
		secretExporter.Export("argocd_status", argoCDComponent.Status)
	}

	// Export Salt Master information if installed (encrypted - contains credentials)
	if saltMasterComponent != nil {
		secretExporter.ExportMap("salt_master", pulumi.Map{
			"master_ip":     saltMasterComponent.MasterIP,
			"api_url":       saltMasterComponent.APIURL,
			"api_username":  saltMasterComponent.APIUsername,
			"api_password":  saltMasterComponent.APIPassword,
			"cluster_token": saltMasterComponent.ClusterToken,
			"status":        saltMasterComponent.Status,
		})
		secretExporter.ExportBool("salt_enabled", true)
	} else {
		secretExporter.ExportBool("salt_enabled", false)
	}

	// Export Salt Minion information if configured (encrypted)
	if saltMinionComponent != nil {
		secretExporter.ExportMap("salt_minions", pulumi.Map{
			"joined_count": saltMinionComponent.JoinedMinions,
			"status":       saltMinionComponent.Status,
		})
	}

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"clusterName":    component.ClusterName,
		"kubeConfig":     component.KubeConfig,
		"sshPrivateKey":  component.SSHPrivateKey,
		"sshPublicKey":   component.SSHPublicKey,
		"apiEndpoint":    component.APIEndpoint,
		"status":         component.Status,
		"configJson":     component.ConfigJSON,
		"lispManifest":   component.LispManifest,
		"deploymentMeta": component.DeploymentMeta,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info("🎉 REAL Kubernetes cluster deployment COMPLETE!", nil)

	return component, nil
}

// sanitizeConfigForStorage creates a copy of the config with sensitive data removed
// This is stored in Pulumi state for config regeneration
func sanitizeConfigForStorage(cfg *config.ClusterConfig) *config.ClusterConfig {
	// Create a deep copy by marshaling and unmarshaling
	data, err := json.Marshal(cfg)
	if err != nil {
		return cfg // Return original if copy fails
	}

	var sanitized config.ClusterConfig
	if err := json.Unmarshal(data, &sanitized); err != nil {
		return cfg
	}

	// Remove sensitive provider tokens
	if sanitized.Providers.DigitalOcean != nil {
		sanitized.Providers.DigitalOcean.Token = "${DIGITALOCEAN_TOKEN}"
	}
	if sanitized.Providers.Linode != nil {
		sanitized.Providers.Linode.Token = "${LINODE_TOKEN}"
		sanitized.Providers.Linode.RootPassword = "${LINODE_ROOT_PASSWORD}"
	}
	if sanitized.Providers.AWS != nil {
		sanitized.Providers.AWS.AccessKeyID = "${AWS_ACCESS_KEY_ID}"
		sanitized.Providers.AWS.SecretAccessKey = "${AWS_SECRET_ACCESS_KEY}"
	}
	if sanitized.Providers.Azure != nil {
		sanitized.Providers.Azure.ClientID = "${AZURE_CLIENT_ID}"
		sanitized.Providers.Azure.ClientSecret = "${AZURE_CLIENT_SECRET}"
		sanitized.Providers.Azure.TenantID = "${AZURE_TENANT_ID}"
		sanitized.Providers.Azure.SubscriptionID = "${AZURE_SUBSCRIPTION_ID}"
	}
	if sanitized.Providers.GCP != nil {
		sanitized.Providers.GCP.Credentials = "${GCP_CREDENTIALS}"
	}

	return &sanitized
}

// generateDeploymentMetadata creates metadata for tracking scale operations
func generateDeploymentMetadata(cfg *config.ClusterConfig, previousMetaJSON string, currentNodeCount int, lispManifest string) *DeploymentMetadata {
	now := time.Now().UTC().Format(time.RFC3339)
	configChecksum := generateSHA256Checksum(lispManifest)

	meta := &DeploymentMetadata{
		LastDeployedAt:   now,
		UpdatedAt:        now,
		CurrentNodeCount: currentNodeCount,
		SlothVersion:     "1.0.0", // TODO: get from version constant
		PulumiVersion:    "3.x",   // Pulumi SDK version
		ConfigChecksum:   configChecksum,
		SchemaVersion:    string(versioning.CurrentSchema),
		ManifestHashes:   make(map[string]string),
		ChangeLog:        make([]ChangeLogEntry, 0),
	}

	// Create config version
	vm := versioning.NewVersionManager()
	configMap := map[string]interface{}{
		"cluster_name": cfg.Metadata.Name,
		"node_count":   currentNodeCount,
		"checksum":     configChecksum,
	}
	if configVersion, err := vm.CreateVersion(configMap, ""); err == nil {
		meta.ConfigVersion = configVersion
	}

	// Parse previous metadata if exists
	var prevMeta DeploymentMetadata
	if previousMetaJSON != "" {
		if err := json.Unmarshal([]byte(previousMetaJSON), &prevMeta); err == nil {
			// Preserve creation timestamp
			meta.CreatedAt = prevMeta.CreatedAt
			meta.DeploymentCount = prevMeta.DeploymentCount + 1
			meta.PreviousNodeCount = prevMeta.CurrentNodeCount
			meta.DeploymentID = fmt.Sprintf("deploy-%d-%s", meta.DeploymentCount, now[:10])

			// Link to parent state
			if prevMeta.StateSnapshotID != "" {
				meta.ParentStateID = prevMeta.StateSnapshotID
			}

			// Update config version with parent hash
			if prevMeta.ConfigVersion != nil && meta.ConfigVersion != nil {
				meta.ConfigVersion.ParentHash = prevMeta.ConfigVersion.ConfigHash
			}

			// Determine scale operation type and create change log entries
			if currentNodeCount > prevMeta.CurrentNodeCount {
				meta.LastScaleOperation = "scale-up"
				meta.ChangeLog = append(meta.ChangeLog, ChangeLogEntry{
					Timestamp:   now,
					ChangeType:  "scale_up",
					ResourceID:  cfg.Metadata.Name,
					Description: fmt.Sprintf("Scaled cluster from %d to %d nodes", prevMeta.CurrentNodeCount, currentNodeCount),
					OldValue:    fmt.Sprintf("%d", prevMeta.CurrentNodeCount),
					NewValue:    fmt.Sprintf("%d", currentNodeCount),
					Actor:       "sloth-kubernetes",
				})
			} else if currentNodeCount < prevMeta.CurrentNodeCount {
				meta.LastScaleOperation = "scale-down"
				meta.ChangeLog = append(meta.ChangeLog, ChangeLogEntry{
					Timestamp:   now,
					ChangeType:  "scale_down",
					ResourceID:  cfg.Metadata.Name,
					Description: fmt.Sprintf("Scaled cluster from %d to %d nodes", prevMeta.CurrentNodeCount, currentNodeCount),
					OldValue:    fmt.Sprintf("%d", prevMeta.CurrentNodeCount),
					NewValue:    fmt.Sprintf("%d", currentNodeCount),
					Actor:       "sloth-kubernetes",
				})
			} else {
				meta.LastScaleOperation = "update"
			}

			// Check for config changes
			if prevMeta.ConfigChecksum != configChecksum {
				meta.ChangeLog = append(meta.ChangeLog, ChangeLogEntry{
					Timestamp:   now,
					ChangeType:  "config_updated",
					ResourceID:  cfg.Metadata.Name,
					Description: "Configuration updated",
					OldValue:    prevMeta.ConfigChecksum,
					NewValue:    configChecksum,
					Actor:       "sloth-kubernetes",
				})
			}

			// Track node pool changes
			meta.NodePoolsAdded, meta.NodePoolsRemoved, meta.NodePoolsScaled = detectPoolChanges(cfg, &prevMeta)

			// Add pool changes to changelog
			for _, poolName := range meta.NodePoolsAdded {
				meta.ChangeLog = append(meta.ChangeLog, ChangeLogEntry{
					Timestamp:   now,
					ChangeType:  "pool_added",
					ResourceID:  poolName,
					Description: fmt.Sprintf("Node pool '%s' added", poolName),
					Actor:       "sloth-kubernetes",
				})
			}
			for _, poolName := range meta.NodePoolsRemoved {
				meta.ChangeLog = append(meta.ChangeLog, ChangeLogEntry{
					Timestamp:   now,
					ChangeType:  "pool_removed",
					ResourceID:  poolName,
					Description: fmt.Sprintf("Node pool '%s' removed", poolName),
					Actor:       "sloth-kubernetes",
				})
			}

			// Build deployment history (keep last 10 entries)
			meta.PreviousDeployments = append([]DeploymentHistoryEntry{{
				DeploymentID:   prevMeta.DeploymentID,
				Timestamp:      prevMeta.LastDeployedAt,
				NodeCount:      prevMeta.CurrentNodeCount,
				ConfigChecksum: prevMeta.ConfigChecksum,
				SchemaVersion:  prevMeta.SchemaVersion,
				Success:        true, // Previous deployment was successful if we have its metadata
			}}, prevMeta.PreviousDeployments...)

			// Trim history to last 10 entries
			if len(meta.PreviousDeployments) > 10 {
				meta.PreviousDeployments = meta.PreviousDeployments[:10]
			}
		}
	}

	// Initial deployment
	if meta.CreatedAt == "" {
		meta.CreatedAt = now
		meta.DeploymentCount = 1
		meta.DeploymentID = fmt.Sprintf("deploy-1-%s", now[:10])
		meta.LastScaleOperation = "initial"
		meta.PreviousNodeCount = 0

		// All pools are "added" on initial deployment
		for poolName := range cfg.NodePools {
			meta.NodePoolsAdded = append(meta.NodePoolsAdded, poolName)
			meta.ChangeLog = append(meta.ChangeLog, ChangeLogEntry{
				Timestamp:   now,
				ChangeType:  "pool_added",
				ResourceID:  poolName,
				Description: fmt.Sprintf("Initial node pool '%s' created", poolName),
				Actor:       "sloth-kubernetes",
			})
		}

		// Initial deployment change log entry
		meta.ChangeLog = append([]ChangeLogEntry{{
			Timestamp:   now,
			ChangeType:  "cluster_created",
			ResourceID:  cfg.Metadata.Name,
			Description: fmt.Sprintf("Initial cluster deployment with %d nodes", currentNodeCount),
			NewValue:    fmt.Sprintf("%d", currentNodeCount),
			Actor:       "sloth-kubernetes",
		}}, meta.ChangeLog...)
	}

	// Generate state snapshot ID for this deployment
	meta.StateSnapshotID = fmt.Sprintf("state-%s-%s", meta.DeploymentID, configChecksum[:8])

	return meta
}

// generateSHA256Checksum creates a cryptographic checksum for change detection
func generateSHA256Checksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// generateChecksum creates a simple checksum for change detection (legacy, kept for compatibility)
func generateChecksum(content string) string {
	// Simple hash for change detection (not cryptographic)
	var hash uint64
	for i, c := range content {
		hash = hash*31 + uint64(c) + uint64(i)
	}
	return fmt.Sprintf("%x", hash)
}

// detectPoolChanges compares current config with previous to find changes
func detectPoolChanges(cfg *config.ClusterConfig, prevMeta *DeploymentMetadata) (added, removed, scaled []string) {
	// This is a simplified version - in production you'd compare against stored pool configs
	// For now, we just return empty slices since we don't have the previous pool configuration
	// stored in metadata. This could be enhanced to store pool info in DeploymentMetadata.
	return nil, nil, nil
}
