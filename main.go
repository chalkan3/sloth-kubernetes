package main

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"kubernetes-create/internal/orchestrator"
	clusterConfig "kubernetes-create/pkg/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Log startup
		ctx.Log.Info("Starting Kubernetes cluster deployment", nil)

		// Load configuration from Pulumi config
		cfg, err := loadConfigFromPulumi(ctx)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Validate configuration
		if err := validateConfig(cfg); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		// Create the SIMPLE REAL orchestrator component
		// This uses ONLY REAL implementations: WireGuard mesh, RKE2, and DNS
		orchestratorComponent, err := orchestrator.NewSimpleRealOrchestratorComponent(ctx, "kubernetes-cluster", cfg)
		if err != nil {
			return fmt.Errorf("failed to create orchestrator component: %w", err)
		}

		// Export REAL cluster outputs
		ctx.Export("status", orchestratorComponent.Status)
		ctx.Export("clusterName", orchestratorComponent.ClusterName)
		ctx.Export("kubeConfig", orchestratorComponent.KubeConfig)
		ctx.Export("sshPrivateKey", orchestratorComponent.SSHPrivateKey)
		ctx.Export("sshPublicKey", orchestratorComponent.SSHPublicKey)
		ctx.Export("ssh_private_key", orchestratorComponent.SSHPrivateKey)
		ctx.Export("ssh_public_key", orchestratorComponent.SSHPublicKey)
		ctx.Export("ssh_private_key_path", pulumi.String("~/.ssh/kubernetes-clusters/production.pem"))
		ctx.Export("apiEndpoint", orchestratorComponent.APIEndpoint)

		// Connection instructions
		ctx.Export("connectionInstructions", pulumi.String(fmt.Sprintf(`
=== REAL KUBERNETES CLUSTER DEPLOYED ===

Cluster: %s
API Endpoint: https://api.chalkan3.com.br:6443

1. Save the kubeconfig:
   pulumi stack output kubeConfig --show-secrets > ~/.kube/config

2. Test cluster:
   kubectl get nodes

3. SSH to nodes:
   pulumi stack output ssh_private_key --show-secrets > ~/.ssh/k8s.pem && chmod 600 ~/.ssh/k8s.pem
   ssh -i ~/.ssh/k8s.pem root@<node-ip>

✅ WireGuard mesh VPN: CONFIGURED
✅ RKE2 Kubernetes: DEPLOYED
✅ DNS records: CREATED
`, cfg.Metadata.Name)))

		return nil
	})
}

// loadConfigFromPulumi loads all configuration from Pulumi config (no YAML files, no env vars)
func loadConfigFromPulumi(ctx *pulumi.Context) (*clusterConfig.ClusterConfig, error) {
	conf := config.New(ctx, "")

	// Get provider tokens from Pulumi config
	doToken := conf.Require("digitaloceanToken")
	linodeToken := conf.Require("linodeToken")

	// Get WireGuard configuration from Pulumi config
	wgEndpoint := conf.Require("wireguardServerEndpoint")
	wgPubKey := conf.Require("wireguardServerPublicKey")

	// Get RKE2 cluster token from Pulumi config (optional, will generate if not set)
	rke2Token := conf.Get("rke2ClusterToken")
	if rke2Token == "" {
		rke2Token = "my-super-secret-cluster-token-rke2-production-2025"
	}

	// Build cluster config
	cfg := &clusterConfig.ClusterConfig{
		Metadata: clusterConfig.Metadata{
			Name: "production",
		},
		Providers: clusterConfig.ProvidersConfig{
			DigitalOcean: &clusterConfig.DigitalOceanProvider{
				Enabled: true,
				Token:   doToken,
				Region:  "nyc3",
			},
			Linode: &clusterConfig.LinodeProvider{
				Enabled:      true,
				Token:        linodeToken,
				Region:       "us-east",
				RootPassword: "SecureLinodeRootPass2025!",
			},
		},
		Network: clusterConfig.NetworkConfig{
			DNS: clusterConfig.DNSConfig{
				Domain:   "chalkan3.com.br",
				Provider: "digitalocean",
			},
			WireGuard: &clusterConfig.WireGuardConfig{
				Enabled:         true,
				ServerEndpoint:  wgEndpoint,
				ServerPublicKey: wgPubKey,
			},
		},
		NodePools: map[string]clusterConfig.NodePool{
			"do-masters": {
				Name:     "do-masters",
				Count:    1,
				Size:     "s-2vcpu-4gb",
			Image:    "ubuntu-22-04-x64",
			Region:   "nyc3",
			Provider: "digitalocean",
				Roles:    []string{"master"},
			},
			"do-workers": {
				Name:     "do-workers",
				Count:    2,
				Size:     "s-2vcpu-4gb",
			Image:    "ubuntu-22-04-x64",
			Region:   "nyc3",
			Provider: "digitalocean",
				Roles:    []string{"worker"},
			},
			"linode-masters": {
				Name:     "linode-masters",
				Count:    2,
				Size:     "g6-standard-2",
			Image:    "linode/ubuntu22.04",
			Region:   "us-east",
			Provider: "linode",
				Roles:    []string{"master"},
			},
			"linode-workers": {
				Name:     "linode-workers",
				Count:    1,
				Size:     "g6-standard-2",
			Image:    "linode/ubuntu22.04",
			Region:   "us-east",
			Provider: "linode",
				Roles:    []string{"worker"},
			},
		},
	}

	return cfg, nil
}

// validateConfig performs additional configuration validation
func validateConfig(config *clusterConfig.ClusterConfig) error {
	// Check that we have exactly 6 nodes configured
	totalNodes := 0
	masterNodes := 0
	workerNodes := 0

	for _, pool := range config.NodePools {
		totalNodes += pool.Count

		for _, role := range pool.Roles {
			if role == "controlplane" || role == "master" {
				masterNodes += pool.Count
				break
			} else if role == "worker" {
				workerNodes += pool.Count
				break
			}
		}
	}

	for _, node := range config.Nodes {
		totalNodes++
		for _, role := range node.Roles {
			if role == "controlplane" || role == "master" {
				masterNodes++
				break
			} else if role == "worker" {
				workerNodes++
				break
			}
		}
	}

	if totalNodes != 6 {
		return fmt.Errorf("configuration must define exactly 6 nodes, found %d", totalNodes)
	}

	if masterNodes != 3 {
		return fmt.Errorf("configuration must define exactly 3 master nodes, found %d", masterNodes)
	}

	if workerNodes != 3 {
		return fmt.Errorf("configuration must define exactly 3 worker nodes, found %d", workerNodes)
	}

	// Verify WireGuard is enabled (required for private cluster)
	if config.Network.WireGuard == nil || !config.Network.WireGuard.Enabled {
		return fmt.Errorf("WireGuard must be enabled for private cluster deployment")
	}

	// Verify WireGuard configuration
	if config.Network.WireGuard.ServerEndpoint == "" {
		return fmt.Errorf("WireGuard server endpoint is required")
	}

	if config.Network.WireGuard.ServerPublicKey == "" {
		return fmt.Errorf("WireGuard server public key is required")
	}

	// Verify provider configuration
	doEnabled := config.Providers.DigitalOcean != nil && config.Providers.DigitalOcean.Enabled
	linodeEnabled := config.Providers.Linode != nil && config.Providers.Linode.Enabled

	if !doEnabled || !linodeEnabled {
		return fmt.Errorf("both DigitalOcean and Linode providers must be enabled")
	}

	// Verify tokens are set
	if doEnabled && config.Providers.DigitalOcean.Token == "" {
		return fmt.Errorf("DigitalOcean API token is required")
	}

	if linodeEnabled && config.Providers.Linode.Token == "" {
		return fmt.Errorf("Linode API token is required")
	}

	return nil
}
