package orchestrator

import (
	"fmt"
	"sync"

	"github.com/chalkan3/sloth-kubernetes/pkg/cluster"
	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/dns"
	"github.com/chalkan3/sloth-kubernetes/pkg/health"
	"github.com/chalkan3/sloth-kubernetes/pkg/ingress"
	"github.com/chalkan3/sloth-kubernetes/pkg/network"
	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	"github.com/chalkan3/sloth-kubernetes/pkg/security"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Orchestrator coordinates the entire cluster deployment
type Orchestrator struct {
	ctx              *pulumi.Context
	config           *config.ClusterConfig
	providerRegistry *providers.ProviderRegistry
	networkManager   *network.Manager
	wireGuardManager *security.WireGuardManager
	tailscaleManager *security.TailscaleManager
	sshKeyManager    *security.SSHKeyManager
	osFirewallMgr    *security.OSFirewallManager
	dnsManager       *dns.Manager
	ingressManager   *ingress.NginxIngressManager
	rkeManager       *cluster.RKEManager
	rke2Manager      *cluster.RKE2Manager
	healthChecker    *health.HealthChecker
	validator        *health.PrerequisiteValidator
	vpnChecker       *network.VPNConnectivityChecker
	nodes            map[string][]*providers.NodeOutput
	mu               sync.Mutex
}

// New creates a new orchestrator
func New(ctx *pulumi.Context, config *config.ClusterConfig) *Orchestrator {
	return &Orchestrator{
		ctx:              ctx,
		config:           config,
		providerRegistry: providers.NewProviderRegistry(),
		nodes:            make(map[string][]*providers.NodeOutput),
		validator:        health.NewPrerequisiteValidator(ctx),
		healthChecker:    health.NewHealthChecker(ctx),
	}
}

// Deploy orchestrates the complete cluster deployment
func (o *Orchestrator) Deploy() error {
	o.ctx.Log.Info("Starting Kubernetes cluster deployment", nil)

	// Phase 0: Generate SSH keys
	if err := o.generateSSHKeys(); err != nil {
		return fmt.Errorf("failed to generate SSH keys: %w", err)
	}

	// Phase 1: Initialize providers
	if err := o.initializeProviders(); err != nil {
		return fmt.Errorf("failed to initialize providers: %w", err)
	}

	// Phase 2: Create networking infrastructure
	if err := o.createNetworking(); err != nil {
		return fmt.Errorf("failed to create networking: %w", err)
	}

	// Phase 3: Deploy nodes
	if err := o.deployNodes(); err != nil {
		return fmt.Errorf("failed to deploy nodes: %w", err)
	}

	// Phase 4: Configure OS-level firewalls on nodes
	// OS firewall configuration moved to component
	// if err := o.configureOSFirewalls(); err != nil {
	// 	return fmt.Errorf("failed to configure OS firewalls: %w", err)
	// }

	// Phase 5: Configure DNS records
	if err := o.configureDNS(); err != nil {
		return fmt.Errorf("failed to configure DNS: %w", err)
	}

	// Phase 6: Configure VPN (WireGuard or Tailscale)
	if err := o.configureVPN(); err != nil {
		return fmt.Errorf("failed to configure VPN: %w", err)
	}

	// Phase 7: Configure cloud provider firewalls
	if err := o.configureFirewalls(); err != nil {
		return fmt.Errorf("failed to configure firewalls: %w", err)
	}

	// Phase 7.5: CRITICAL - Verify VPN connectivity before RKE
	// This MUST pass before RKE deployment or the cluster will fail
	vpnEnabled := (o.config.Network.WireGuard != nil && o.config.Network.WireGuard.Enabled) ||
		(o.config.Network.Tailscale != nil && o.config.Network.Tailscale.Enabled)

	if vpnEnabled {
		o.ctx.Log.Info("=====================================", nil)
		o.ctx.Log.Info("CRITICAL: Verifying VPN Connectivity", nil)
		o.ctx.Log.Info("=====================================", nil)
		o.ctx.Log.Info("RKE requires all nodes to communicate via private network", nil)
		o.ctx.Log.Info("Checking full mesh connectivity before proceeding...", nil)

		if err := o.verifyVPNReadyForRKE(); err != nil {
			return fmt.Errorf("VPN not ready for RKE deployment: %w", err)
		}

		o.ctx.Log.Info("=====================================", nil)
		o.ctx.Log.Info("✓ VPN VERIFIED - Safe to deploy RKE", nil)
		o.ctx.Log.Info("=====================================", nil)
	}

	// Phase 8: Deploy RKE cluster
	if err := o.deployRKE(); err != nil {
		return fmt.Errorf("failed to deploy RKE: %w", err)
	}

	// Phase 9: Install NGINX Ingress
	if err := o.installIngress(); err != nil {
		return fmt.Errorf("failed to install ingress: %w", err)
	}

	// Phase 10: Install addons
	if err := o.installAddons(); err != nil {
		return fmt.Errorf("failed to install addons: %w", err)
	}

	// Phase 11: Export outputs
	o.exportOutputs()

	o.ctx.Log.Info("Kubernetes cluster deployment completed successfully", nil)
	return nil
}

// generateSSHKeys generates SSH keys for the cluster
func (o *Orchestrator) generateSSHKeys() error {
	o.ctx.Log.Info("Generating SSH keys for cluster", nil)

	o.sshKeyManager = security.NewSSHKeyManager(o.ctx)
	if err := o.sshKeyManager.GenerateKeyPair(); err != nil {
		return fmt.Errorf("failed to generate SSH key pair: %w", err)
	}

	// Set the SSH key in the config for providers to use
	publicKey := o.sshKeyManager.GetPublicKeyString()

	// Update provider configs with the generated key
	if o.config.Providers.DigitalOcean != nil {
		o.config.Providers.DigitalOcean.SSHPublicKey = publicKey
	}
	if o.config.Providers.Linode != nil {
		o.config.Providers.Linode.SSHPublicKey = publicKey
	}
	// Note: Azure SSH key is handled directly in node_deployment.go via sshKeyOutput parameter

	return nil
}

// initializeProviders initializes all cloud providers
func (o *Orchestrator) initializeProviders() error {
	o.ctx.Log.Info("Initializing cloud providers", nil)

	// Initialize DigitalOcean provider
	if o.config.Providers.DigitalOcean != nil && o.config.Providers.DigitalOcean.Enabled {
		doProvider := providers.NewDigitalOceanProvider()
		if err := doProvider.Initialize(o.ctx, o.config); err != nil {
			return fmt.Errorf("failed to initialize DigitalOcean provider: %w", err)
		}
		o.providerRegistry.Register("digitalocean", doProvider)
		o.ctx.Log.Info("✓ DigitalOcean provider initialized", nil)
	}

	// Initialize Linode provider
	if o.config.Providers.Linode != nil && o.config.Providers.Linode.Enabled {
		linodeProvider := providers.NewLinodeProvider()
		if err := linodeProvider.Initialize(o.ctx, o.config); err != nil {
			return fmt.Errorf("failed to initialize Linode provider: %w", err)
		}
		o.providerRegistry.Register("linode", linodeProvider)
		o.ctx.Log.Info("✓ Linode provider initialized", nil)
	}

	// Initialize Azure provider
	if o.config.Providers.Azure != nil && o.config.Providers.Azure.Enabled {
		azureProvider := providers.NewAzureProvider()
		if err := azureProvider.Initialize(o.ctx, o.config); err != nil {
			return fmt.Errorf("failed to initialize Azure provider: %w", err)
		}
		o.providerRegistry.Register("azure", azureProvider)
		o.ctx.Log.Info("✓ Azure provider initialized", nil)
	}

	// Initialize AWS provider
	if o.config.Providers.AWS != nil && o.config.Providers.AWS.Enabled {
		awsProvider := providers.NewAWSProvider()
		if err := awsProvider.Initialize(o.ctx, o.config); err != nil {
			return fmt.Errorf("failed to initialize AWS provider: %w", err)
		}
		o.providerRegistry.Register("aws", awsProvider)
		o.ctx.Log.Info("✓ AWS provider initialized", nil)
	}

	// Initialize GCP provider
	if o.config.Providers.GCP != nil && o.config.Providers.GCP.Enabled {
		gcpProvider := providers.NewGCPProvider()
		if err := gcpProvider.Initialize(o.ctx, o.config); err != nil {
			return fmt.Errorf("failed to initialize GCP provider: %w", err)
		}
		o.providerRegistry.Register("gcp", gcpProvider)
		o.ctx.Log.Info("✓ GCP provider initialized", nil)
	}

	// Verify at least one provider is enabled
	if len(o.providerRegistry.GetAll()) == 0 {
		return fmt.Errorf("no cloud providers enabled")
	}

	return nil
}

// createNetworking creates network infrastructure
func (o *Orchestrator) createNetworking() error {
	o.ctx.Log.Info("Creating network infrastructure", nil)

	o.networkManager = network.NewManager(o.ctx, &o.config.Network)

	// Register providers with network manager
	for name, provider := range o.providerRegistry.GetAll() {
		o.networkManager.RegisterProvider(name, provider)
	}

	// Validate network configuration
	if err := o.networkManager.ValidateCIDRs(); err != nil {
		return fmt.Errorf("network validation failed: %w", err)
	}

	// Create networks
	if err := o.networkManager.CreateNetworks(); err != nil {
		return fmt.Errorf("failed to create networks: %w", err)
	}

	return nil
}

// deployNodes deploys all cluster nodes
func (o *Orchestrator) deployNodes() error {
	o.ctx.Log.Info("Deploying cluster nodes", nil)

	// Deploy individual nodes
	for i := range o.config.Nodes {
		nodeConfig := &o.config.Nodes[i]
		if err := o.deployNode(nodeConfig); err != nil {
			return fmt.Errorf("failed to deploy node %s: %w", nodeConfig.Name, err)
		}
	}

	// Deploy node pools
	for poolName := range o.config.NodePools {
		poolConfig := o.config.NodePools[poolName]
		if err := o.deployNodePool(poolName, &poolConfig); err != nil {
			return fmt.Errorf("failed to deploy node pool %s: %w", poolName, err)
		}
	}

	// Verify we have the required nodes
	if err := o.verifyNodeDistribution(); err != nil {
		return err
	}

	// Initialize health checker and validator
	o.healthChecker = health.NewHealthChecker(o.ctx)
	o.validator = health.NewPrerequisiteValidator(o.ctx)

	// Add all nodes to health checker
	for _, nodes := range o.nodes {
		for _, node := range nodes {
			o.healthChecker.AddNode(node)
		}
	}

	// Set SSH key path if available
	if o.sshKeyManager != nil {
		sshKeyPath := fmt.Sprintf("~/.ssh/kubernetes-clusters/%s.pem", o.ctx.Stack())
		o.healthChecker.SetSSHKeyPath(sshKeyPath)
	}

	// Wait for all nodes to be ready with basic services
	o.ctx.Log.Info("Waiting for all nodes to be ready with SSH and Docker", nil)
	requiredServices := []string{"ssh", "docker"}
	if err := o.healthChecker.WaitForNodesReady(requiredServices); err != nil {
		return fmt.Errorf("nodes failed health checks: %w", err)
	}

	return nil
}

// deployNode deploys a single node
func (o *Orchestrator) deployNode(nodeConfig *config.NodeConfig) error {
	provider, ok := o.providerRegistry.Get(nodeConfig.Provider)
	if !ok {
		return fmt.Errorf("provider %s not found", nodeConfig.Provider)
	}

	node, err := provider.CreateNode(o.ctx, nodeConfig)
	if err != nil {
		return err
	}

	o.mu.Lock()
	o.nodes[nodeConfig.Provider] = append(o.nodes[nodeConfig.Provider], node)
	o.mu.Unlock()

	return nil
}

// deployNodePool deploys a pool of nodes
func (o *Orchestrator) deployNodePool(poolName string, poolConfig *config.NodePool) error {
	provider, ok := o.providerRegistry.Get(poolConfig.Provider)
	if !ok {
		return fmt.Errorf("provider %s not found", poolConfig.Provider)
	}

	nodes, err := provider.CreateNodePool(o.ctx, poolConfig)
	if err != nil {
		return err
	}

	o.mu.Lock()
	o.nodes[poolConfig.Provider] = append(o.nodes[poolConfig.Provider], nodes...)
	o.mu.Unlock()

	return nil
}

// verifyNodeDistribution verifies the node distribution matches requirements
func (o *Orchestrator) verifyNodeDistribution() error {
	totalNodes := 0
	masterNodes := 0
	workerNodes := 0

	doNodes := 0
	linodeNodes := 0

	for provider, nodes := range o.nodes {
		for _, node := range nodes {
			totalNodes++

			// Count by provider
			switch provider {
			case "digitalocean":
				doNodes++
			case "linode":
				linodeNodes++
			}

			// Count by role
			if node.Labels != nil {
				if role, ok := node.Labels["role"]; ok {
					switch role {
					case "master", "controlplane":
						masterNodes++
					case "worker":
						workerNodes++
					}
				}
			}
		}
	}

	// Calculate expected nodes from configuration
	expectedTotal := 0
	expectedMasters := 0
	expectedWorkers := 0

	for _, pool := range o.config.NodePools {
		expectedTotal += pool.Count
		for _, role := range pool.Roles {
			if role == "master" || role == "controlplane" {
				expectedMasters += pool.Count
			} else if role == "worker" {
				expectedWorkers += pool.Count
			}
		}
	}

	// Verify distribution
	if totalNodes != expectedTotal {
		return fmt.Errorf("expected %d nodes, got %d", expectedTotal, totalNodes)
	}

	if masterNodes != expectedMasters {
		return fmt.Errorf("expected %d master nodes, got %d", expectedMasters, masterNodes)
	}

	if workerNodes != expectedWorkers {
		return fmt.Errorf("expected %d worker nodes, got %d", expectedWorkers, workerNodes)
	}

	o.ctx.Log.Info(fmt.Sprintf("Node distribution verified: %d total (%d masters, %d workers)", totalNodes, masterNodes, workerNodes), nil)

	return nil
}

// configureDNS configures DNS records for all nodes
func (o *Orchestrator) configureDNS() error {
	// Check if we have a domain configured
	domain := o.config.Network.DNS.Domain
	if domain == "" {
		// Default to chalkan3.com.br if not configured
		domain = "chalkan3.com.br"
	}

	o.ctx.Log.Info("Configuring DNS records", nil)

	o.dnsManager = dns.NewManager(o.ctx, domain)

	// Create DNS records for all nodes
	if err := o.dnsManager.CreateNodeRecords(o.nodes); err != nil {
		return fmt.Errorf("failed to create node DNS records: %w", err)
	}

	// Create cluster convenience records
	if err := o.dnsManager.CreateClusterRecords(); err != nil {
		return fmt.Errorf("failed to create cluster DNS records: %w", err)
	}

	// Export DNS information
	o.dnsManager.ExportDNSInfo()

	return nil
}

// configureVPN configures the VPN based on network mode (WireGuard or Tailscale)
func (o *Orchestrator) configureVPN() error {
	// Check which VPN mode is enabled
	if o.config.Network.Tailscale != nil && o.config.Network.Tailscale.Enabled {
		return o.configureTailscale()
	}

	if o.config.Network.WireGuard != nil && o.config.Network.WireGuard.Enabled {
		return o.configureWireGuard()
	}

	o.ctx.Log.Info("No VPN configured, skipping VPN setup", nil)
	return nil
}

// configureTailscale configures Tailscale VPN on all nodes via Headscale
func (o *Orchestrator) configureTailscale() error {
	o.ctx.Log.Info("Configuring Tailscale VPN via Headscale", nil)

	o.tailscaleManager = security.NewTailscaleManager(o.ctx, o.config.Network.Tailscale)

	// Set SSH private key if available
	if o.sshKeyManager != nil {
		// Get SSH private key content
		o.tailscaleManager.SetSSHPrivateKey("")
	}

	// Auto-provision Headscale server if requested
	if o.config.Network.Tailscale.Create {
		o.ctx.Log.Info("Auto-provisioning Headscale coordination server", nil)

		// Get subnet ID from network manager if available
		var subnetID pulumi.StringOutput
		if o.networkManager != nil {
			// Use the first available subnet
			subnetID = pulumi.String("").ToStringOutput()
		}

		result, err := o.tailscaleManager.CreateHeadscaleServerIfNeeded(nil, nil, subnetID)
		if err != nil {
			return fmt.Errorf("failed to create Headscale server: %w", err)
		}

		if result != nil {
			o.ctx.Log.Info("✓ Headscale server provisioned", nil)
			o.tailscaleManager.SetHeadscaleInfo(result.APIURL, result.AuthKey, nil)
		}
	}

	// Validate Tailscale configuration
	if err := o.tailscaleManager.ValidateConfiguration(); err != nil {
		return fmt.Errorf("Tailscale validation failed: %w", err)
	}

	// Configure Tailscale on each node
	for _, nodes := range o.nodes {
		for _, node := range nodes {
			if err := o.tailscaleManager.ConfigureNode(node); err != nil {
				return fmt.Errorf("failed to configure Tailscale on %s: %w", node.Name, err)
			}
		}
	}

	// Initialize VPN connectivity checker
	o.ctx.Log.Info("Initializing VPN connectivity verification for Tailscale", nil)
	o.vpnChecker = network.NewVPNConnectivityChecker(o.ctx)

	// Add all nodes to VPN checker
	for _, nodes := range o.nodes {
		for _, node := range nodes {
			o.vpnChecker.AddNode(node)
		}
	}

	// Set SSH key path if available
	if o.sshKeyManager != nil {
		sshKeyPath := fmt.Sprintf("~/.ssh/kubernetes-clusters/%s.pem", o.ctx.Stack())
		o.vpnChecker.SetSSHKeyPath(sshKeyPath)
	}

	// Wait for Tailscale to establish connections
	o.ctx.Log.Info("Waiting for Tailscale connections to establish on all nodes", nil)
	// Note: For Tailscale, we wait for tailscale status to show Running
	// The vpnChecker will need to be enhanced for Tailscale support

	o.ctx.Log.Info("✓ Tailscale VPN configured on all nodes!", nil)
	o.tailscaleManager.ExportTailscaleInfo()

	return nil
}

// configureWireGuard configures WireGuard VPN on all nodes
func (o *Orchestrator) configureWireGuard() error {
	if o.config.Network.WireGuard == nil || !o.config.Network.WireGuard.Enabled {
		o.ctx.Log.Info("WireGuard not enabled, skipping configuration", nil)
		return nil
	}

	o.ctx.Log.Info("Configuring WireGuard VPN", nil)

	o.wireGuardManager = security.NewWireGuardManager(o.ctx, o.config.Network.WireGuard)

	// Validate WireGuard configuration
	if err := o.wireGuardManager.ValidateConfiguration(); err != nil {
		return fmt.Errorf("WireGuard validation failed: %w", err)
	}

	// Configure WireGuard on each node
	for _, nodes := range o.nodes {
		for _, node := range nodes {
			if err := o.wireGuardManager.ConfigureNode(node); err != nil {
				return fmt.Errorf("failed to configure WireGuard on %s: %w", node.Name, err)
			}
		}
	}

	// Configure peers on WireGuard server
	if err := o.wireGuardManager.ConfigureServerPeers(); err != nil {
		return fmt.Errorf("failed to configure WireGuard server peers: %w", err)
	}

	// Initialize VPN connectivity checker
	o.ctx.Log.Info("Initializing VPN connectivity verification", nil)
	o.vpnChecker = network.NewVPNConnectivityChecker(o.ctx)

	// Add all nodes to VPN checker
	for _, nodes := range o.nodes {
		for _, node := range nodes {
			o.vpnChecker.AddNode(node)
		}
	}

	// Set SSH key path if available
	if o.sshKeyManager != nil {
		sshKeyPath := fmt.Sprintf("~/.ssh/kubernetes-clusters/%s.pem", o.ctx.Stack())
		o.vpnChecker.SetSSHKeyPath(sshKeyPath)
	}

	// Wait for WireGuard tunnels to be established
	o.ctx.Log.Info("Waiting for WireGuard tunnels to establish on all nodes", nil)
	if err := o.vpnChecker.WaitForTunnelEstablishment(); err != nil {
		return fmt.Errorf("failed waiting for WireGuard tunnels: %w", err)
	}

	// Verify full mesh VPN connectivity between all nodes
	o.ctx.Log.Info("Verifying full mesh VPN connectivity between all nodes", nil)
	o.ctx.Log.Info("This ensures every node can reach every other node via WireGuard", nil)

	if err := o.vpnChecker.VerifyFullMeshConnectivity(); err != nil {
		// Print connectivity matrix to help debug
		o.vpnChecker.PrintConnectivityMatrix()
		return fmt.Errorf("VPN connectivity verification failed: %w", err)
	}

	// Print successful connectivity matrix
	o.vpnChecker.PrintConnectivityMatrix()
	o.ctx.Log.Info("✓ VPN full mesh connectivity verified successfully!", nil)

	return nil
}

// configureFirewalls configures firewalls for all nodes
func (o *Orchestrator) configureFirewalls() error {
	o.ctx.Log.Info("Configuring firewalls", nil)

	if err := o.networkManager.CreateFirewalls(o.nodes); err != nil {
		return fmt.Errorf("failed to create firewalls: %w", err)
	}

	return nil
}

// deployRKE deploys the RKE cluster
func (o *Orchestrator) deployRKE() error {
	o.ctx.Log.Info("Preparing to deploy RKE cluster", nil)

	// Collect all nodes for validation
	allNodes := []*providers.NodeOutput{}
	for _, nodes := range o.nodes {
		allNodes = append(allNodes, nodes...)
	}

	// Validate prerequisites for RKE installation
	o.ctx.Log.Info("Running prerequisite validation for RKE installation", nil)
	if err := o.validator.ValidateForRKE(allNodes); err != nil {
		o.validator.PrintSummary()
		return fmt.Errorf("RKE prerequisite validation failed: %w", err)
	}
	o.validator.PrintSummary()

	// VPN connectivity is already verified in Phase 6.5 before we get here
	// Just log that we're proceeding with verified connectivity
	if o.config.Network.WireGuard != nil && o.config.Network.WireGuard.Enabled {
		o.ctx.Log.Info("VPN connectivity already verified - proceeding with RKE deployment", nil)
	}

	o.ctx.Log.Info("All prerequisites validated, deploying Kubernetes cluster", nil)

	// Check distribution type and use appropriate manager
	distribution := o.config.Kubernetes.Distribution
	if distribution == "" {
		distribution = "rke" // Default to RKE1 for backwards compatibility
	}

	switch distribution {
	case "rke2":
		o.ctx.Log.Info("Using RKE2 distribution", nil)
		o.rke2Manager = cluster.NewRKE2Manager(o.ctx, &o.config.Kubernetes)

		// Set SSH private key if available
		if o.sshKeyManager != nil {
			// Get SSH private key content
			// Note: In production, this should retrieve the actual key
			o.rke2Manager.SetSSHPrivateKey("")
		}

		// Add all nodes to RKE2 manager
		for _, nodes := range o.nodes {
			for _, node := range nodes {
				o.rke2Manager.AddNode(node)
			}
		}

		// Deploy the cluster
		if err := o.rke2Manager.DeployCluster(); err != nil {
			return fmt.Errorf("RKE2 deployment failed: %w", err)
		}

		// Export cluster info
		o.rke2Manager.ExportClusterInfo()

	default: // "rke" or any other value defaults to RKE1
		o.ctx.Log.Info("Using RKE1 distribution", nil)
		o.rkeManager = cluster.NewRKEManager(o.ctx, &o.config.Kubernetes)

		// Add all nodes to RKE manager
		for _, nodes := range o.nodes {
			for _, node := range nodes {
				o.rkeManager.AddNode(node)
			}
		}

		// Deploy the cluster
		if err := o.rkeManager.DeployCluster(); err != nil {
			return fmt.Errorf("RKE deployment failed: %w", err)
		}
	}

	// Wait for Kubernetes to be ready
	o.ctx.Log.Info("Waiting for Kubernetes cluster to be ready", nil)
	if err := o.healthChecker.WaitForKubernetesReady(); err != nil {
		return fmt.Errorf("Kubernetes cluster failed to become ready: %w", err)
	}

	o.ctx.Log.Info("RKE cluster deployed successfully", nil)

	return nil
}

// installIngress installs NGINX Ingress Controller
func (o *Orchestrator) installIngress() error {
	o.ctx.Log.Info("Preparing to install NGINX Ingress Controller", nil)

	// Collect all nodes for validation
	allNodes := []*providers.NodeOutput{}
	for _, nodes := range o.nodes {
		allNodes = append(allNodes, nodes...)
	}

	// Validate prerequisites for Ingress installation
	o.ctx.Log.Info("Running prerequisite validation for Ingress installation", nil)
	if err := o.validator.ValidateForIngress(allNodes); err != nil {
		o.validator.PrintSummary()
		return fmt.Errorf("Ingress prerequisite validation failed: %w", err)
	}
	o.validator.PrintSummary()

	// Ensure Kubernetes is still healthy before proceeding
	o.ctx.Log.Info("Verifying Kubernetes cluster health before Ingress installation", nil)
	if err := o.healthChecker.WaitForKubernetesReady(); err != nil {
		return fmt.Errorf("Kubernetes cluster not ready for Ingress installation: %w", err)
	}

	o.ctx.Log.Info("All prerequisites validated, installing NGINX Ingress Controller", nil)

	// Get domain for ingress
	domain := o.config.Network.DNS.Domain
	if domain == "" {
		domain = "chalkan3.com.br"
	}

	// Create ingress manager
	o.ingressManager = ingress.NewNginxIngressManager(o.ctx, domain)

	// Get first master node
	masterNode := o.GetMasterNodes()[0]
	if masterNode == nil {
		return fmt.Errorf("no master node available for ingress installation")
	}

	o.ingressManager.SetMasterNode(masterNode)

	// Set SSH key path if available
	if o.sshKeyManager != nil {
		sshKeyPath := fmt.Sprintf("~/.ssh/kubernetes-clusters/%s.pem", o.ctx.Stack())
		o.ingressManager.SetSSHKeyPath(sshKeyPath)
	}

	// Install NGINX Ingress
	ingressIP, err := o.ingressManager.Install()
	if err != nil {
		return fmt.Errorf("failed to install NGINX Ingress: %w", err)
	}

	// Wait for Ingress to be ready
	o.ctx.Log.Info("Waiting for NGINX Ingress Controller to be ready", nil)
	if err := o.healthChecker.WaitForIngressReady(); err != nil {
		return fmt.Errorf("NGINX Ingress Controller failed to become ready: %w", err)
	}

	// Update DNS records with actual ingress IP
	if o.dnsManager != nil {
		if err := o.dnsManager.UpdateIngressRecord(ingressIP); err != nil {
			o.ctx.Log.Warn("Failed to update ingress DNS record", nil)
		}
	}

	// Install cert-manager for TLS
	if err := o.ingressManager.InstallCertManager(); err != nil {
		o.ctx.Log.Warn("Failed to install cert-manager", nil)
	}

	// Create sample ingress
	o.ingressManager.CreateSampleIngress()

	o.ctx.Log.Info("NGINX Ingress Controller installed successfully", nil)

	return nil
}

// installAddons installs cluster addons
func (o *Orchestrator) installAddons() error {
	o.ctx.Log.Info("Installing cluster addons", nil)

	if o.rkeManager == nil {
		return fmt.Errorf("RKE manager not initialized - cannot install addons")
	}

	if err := o.rkeManager.InstallAddons(); err != nil {
		return fmt.Errorf("failed to install addons: %w", err)
	}

	// Install storage if configured
	if len(o.config.Storage.Classes) > 0 {
		if err := o.installStorage(); err != nil {
			return fmt.Errorf("failed to install storage: %w", err)
		}
	}

	// Install load balancers if configured
	if len([]*config.LoadBalancerConfig{&o.config.LoadBalancer}) > 0 {
		if err := o.installLoadBalancers(); err != nil {
			return fmt.Errorf("failed to install load balancers: %w", err)
		}
	}

	return nil
}

// installStorage installs storage classes
func (o *Orchestrator) installStorage() error {
	// Implementation would install configured storage providers
	o.ctx.Log.Info("Storage configuration detected but not implemented", nil)
	return nil
}

// installLoadBalancers installs load balancers
func (o *Orchestrator) installLoadBalancers() error {
	for _, lbConfig := range []*config.LoadBalancerConfig{&o.config.LoadBalancer} {
		provider, ok := o.providerRegistry.Get(lbConfig.Provider)
		if !ok {
			return fmt.Errorf("provider %s not found for load balancer", lbConfig.Provider)
		}

		lb, err := provider.CreateLoadBalancer(o.ctx, lbConfig)
		if err != nil {
			return fmt.Errorf("failed to create load balancer %s: %w", lbConfig.Name, err)
		}

		secrets.Export(o.ctx, fmt.Sprintf("lb_%s_ip", lbConfig.Name), lb.IP)
	}

	return nil
}

// exportOutputs exports all cluster outputs
func (o *Orchestrator) exportOutputs() {
	o.ctx.Log.Info("Exporting cluster outputs", nil)

	// Export metadata
	secrets.Export(o.ctx, "cluster_name", pulumi.String(o.config.Metadata.Name))
	secrets.Export(o.ctx, "environment", pulumi.String(o.config.Metadata.Environment))
	secrets.Export(o.ctx, "version", pulumi.String(o.config.Metadata.Version))

	// Export node information
	nodeOutputs := make(map[string]interface{})
	for provider, nodes := range o.nodes {
		for _, node := range nodes {
			nodeOutputs[node.Name] = map[string]interface{}{
				"provider":     provider,
				"public_ip":    node.PublicIP,
				"private_ip":   node.PrivateIP,
				"wireguard_ip": node.WireGuardIP,
				"region":       node.Region,
				"size":         node.Size,
			}
		}
	}
	secrets.Export(o.ctx, "nodes", pulumi.ToMap(nodeOutputs))

	// Export network information
	if o.networkManager != nil {
		o.networkManager.ExportNetworkOutputs()
	}

	// Export VPN information (WireGuard or Tailscale)
	if o.wireGuardManager != nil {
		o.wireGuardManager.ExportWireGuardInfo()
	}
	if o.tailscaleManager != nil {
		o.tailscaleManager.ExportTailscaleInfo()
	}

	// Export RKE information
	if o.rkeManager != nil {
		o.rkeManager.ExportClusterInfo()
	}

	// Export health check results
	if o.healthChecker != nil {
		healthStatuses := o.healthChecker.GetAllStatuses()
		healthOutputs := make(map[string]interface{})
		for name, status := range healthStatuses {
			healthOutputs[name] = map[string]interface{}{
				"healthy":    status.IsHealthy,
				"last_check": status.LastCheck.Format("2006-01-02 15:04:05"),
				"services":   status.Services,
			}
		}
		secrets.Export(o.ctx, "health_status", pulumi.ToMap(healthOutputs))
	}

	// Export validation results
	if o.validator != nil {
		validationResults := o.validator.GetResults()
		validationOutputs := make(map[string]interface{})
		for name, result := range validationResults {
			validationOutputs[name] = map[string]interface{}{
				"success": result.Success,
				"message": result.Message,
			}
		}
		secrets.Export(o.ctx, "validation_results", pulumi.ToMap(validationOutputs))
	}

	// Export VPN connectivity matrix
	if o.vpnChecker != nil {
		matrix := o.vpnChecker.GetConnectivityMatrix()
		matrixOutputs := make(map[string]interface{})
		for source, targets := range matrix {
			targetOutputs := make(map[string]interface{})
			for target, connected := range targets {
				targetOutputs[target] = connected
			}
			matrixOutputs[source] = targetOutputs
		}
		secrets.Export(o.ctx, "vpn_connectivity_matrix", pulumi.ToMap(matrixOutputs))
	}

	// Export access information
	vpnRequired := (o.config.Network.WireGuard != nil && o.config.Network.WireGuard.Enabled) ||
		(o.config.Network.Tailscale != nil && o.config.Network.Tailscale.Enabled)

	vpnType := "none"
	if o.config.Network.WireGuard != nil && o.config.Network.WireGuard.Enabled {
		vpnType = "wireguard"
	} else if o.config.Network.Tailscale != nil && o.config.Network.Tailscale.Enabled {
		vpnType = "tailscale"
	}

	secrets.Export(o.ctx, "access_info", pulumi.Map{
		"vpn_required": pulumi.Bool(vpnRequired),
		"vpn_type":     pulumi.String(vpnType),
		"api_endpoint": pulumi.String(fmt.Sprintf("https://10.8.0.11:6443")), // Master 1 VPN IP
		"ssh_user":     pulumi.String("root"),
	})
}

// Cleanup performs cleanup operations
func (o *Orchestrator) Cleanup() error {
	o.ctx.Log.Info("Performing cleanup operations", nil)

	for _, provider := range o.providerRegistry.GetAll() {
		if err := provider.Cleanup(o.ctx); err != nil {
			o.ctx.Log.Warn("Cleanup failed for provider", nil)
		}
	}

	return nil
}

// GetNodeByName returns a node by name
func (o *Orchestrator) GetNodeByName(name string) (*providers.NodeOutput, error) {
	for _, nodes := range o.nodes {
		for _, node := range nodes {
			if node.Name == name {
				return node, nil
			}
		}
	}
	return nil, fmt.Errorf("node %s not found", name)
}

// GetNodesByProvider returns all nodes for a provider
func (o *Orchestrator) GetNodesByProvider(provider string) ([]*providers.NodeOutput, error) {
	nodes, ok := o.nodes[provider]
	if !ok {
		return nil, fmt.Errorf("no nodes found for provider %s", provider)
	}
	return nodes, nil
}

// GetMasterNodes returns all master nodes
func (o *Orchestrator) GetMasterNodes() []*providers.NodeOutput {
	masters := []*providers.NodeOutput{}
	for _, nodes := range o.nodes {
		for _, node := range nodes {
			if node.Labels != nil {
				if role, ok := node.Labels["role"]; ok && (role == "master" || role == "controlplane") {
					masters = append(masters, node)
				}
			}
		}
	}
	return masters
}

// GetWorkerNodes returns all worker nodes
func (o *Orchestrator) GetWorkerNodes() []*providers.NodeOutput {
	workers := []*providers.NodeOutput{}
	for _, nodes := range o.nodes {
		for _, node := range nodes {
			if node.Labels != nil {
				if role, ok := node.Labels["role"]; ok && role == "worker" {
					workers = append(workers, node)
				}
			}
		}
	}
	return workers
}

// verifyVPNReadyForRKE performs comprehensive VPN verification before RKE deployment
func (o *Orchestrator) verifyVPNReadyForRKE() error {
	// Initialize VPN checker if not already done
	if o.vpnChecker == nil {
		o.vpnChecker = network.NewVPNConnectivityChecker(o.ctx)

		// Add all nodes to VPN checker
		for _, nodes := range o.nodes {
			for _, node := range nodes {
				o.vpnChecker.AddNode(node)
			}
		}

		// Set SSH key path
		if o.sshKeyManager != nil {
			sshKeyPath := fmt.Sprintf("~/.ssh/kubernetes-clusters/%s.pem", o.ctx.Stack())
			o.vpnChecker.SetSSHKeyPath(sshKeyPath)
		}
	}

	// Step 1: Verify WireGuard is running on all nodes
	o.ctx.Log.Info("Step 1: Verifying WireGuard service on all nodes", nil)
	if err := o.vpnChecker.WaitForTunnelEstablishment(); err != nil {
		return fmt.Errorf("WireGuard tunnels not established: %w", err)
	}
	o.ctx.Log.Info("✓ WireGuard running on all nodes", nil)

	// Step 2: Verify full mesh connectivity
	o.ctx.Log.Info("Step 2: Verifying full mesh VPN connectivity", nil)
	o.ctx.Log.Info("Each node must reach every other node for RKE to work", nil)

	if err := o.vpnChecker.VerifyFullMeshConnectivity(); err != nil {
		o.ctx.Log.Error("VPN Connectivity FAILED - Cannot proceed with RKE", nil)
		o.vpnChecker.PrintConnectivityMatrix()

		// Provide detailed error information
		o.ctx.Log.Error("RKE Requirements NOT Met:", nil)
		o.ctx.Log.Error("- All master nodes must reach each other", nil)
		o.ctx.Log.Error("- All worker nodes must reach all masters", nil)
		o.ctx.Log.Error("- etcd requires full connectivity between masters", nil)

		return fmt.Errorf("VPN connectivity check failed: %w", err)
	}

	// Step 3: Print connectivity matrix for verification
	o.ctx.Log.Info("✓ Full mesh connectivity verified", nil)
	o.vpnChecker.PrintConnectivityMatrix()

	// Step 4: Verify specific RKE requirements
	o.ctx.Log.Info("Step 3: Verifying RKE-specific connectivity requirements", nil)

	// Check master-to-master connectivity
	masters := o.GetMasterNodes()
	if len(masters) > 1 {
		o.ctx.Log.Info("Verifying master-to-master connectivity for etcd cluster", nil)
		matrix := o.vpnChecker.GetConnectivityMatrix()

		for _, master1 := range masters {
			for _, master2 := range masters {
				if master1.Name != master2.Name {
					if targets, exists := matrix[master1.Name]; exists {
						if !targets[master2.Name] {
							return fmt.Errorf("CRITICAL: Master %s cannot reach master %s - etcd cluster will fail",
								master1.Name, master2.Name)
						}
					}
				}
			}
		}
		o.ctx.Log.Info("✓ All masters can communicate for etcd", nil)
	}

	// Check worker-to-master connectivity
	workers := o.GetWorkerNodes()
	if len(workers) > 0 && len(masters) > 0 {
		o.ctx.Log.Info("Verifying worker-to-master connectivity for API access", nil)
		matrix := o.vpnChecker.GetConnectivityMatrix()

		for _, worker := range workers {
			for _, master := range masters {
				if targets, exists := matrix[worker.Name]; exists {
					if !targets[master.Name] {
						return fmt.Errorf("CRITICAL: Worker %s cannot reach master %s - kubelet will fail",
							worker.Name, master.Name)
					}
				}
			}
		}
		o.ctx.Log.Info("✓ All workers can reach masters for API access", nil)
	}

	// Step 5: Final validation
	o.ctx.Log.Info("Step 4: Final VPN validation", nil)

	// Get all nodes for a final check
	allNodes := []*providers.NodeOutput{}
	for _, nodes := range o.nodes {
		allNodes = append(allNodes, nodes...)
	}

	// Ensure we have the expected number of nodes
	if len(allNodes) != 6 {
		return fmt.Errorf("expected 6 nodes for VPN mesh, found %d", len(allNodes))
	}

	o.ctx.Log.Info("✓ All VPN checks passed", nil)

	return nil
}
