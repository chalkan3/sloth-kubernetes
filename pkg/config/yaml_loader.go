package config

import (
	"fmt"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"
)

// LoadFromYAML loads cluster configuration from a YAML file
// Automatically detects Kubernetes-style (with apiVersion/kind) or legacy format
func LoadFromYAML(filePath string) (*ClusterConfig, error) {
	// Expand home directory if needed
	if len(filePath) > 0 && filePath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		filePath = filepath.Join(home, filePath[1:])
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Detect format by checking for apiVersion field
	var detector struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(data, &detector); err == nil && detector.APIVersion != "" {
		// Kubernetes-style format detected
		return LoadFromK8sYAML(filePath)
	}

	// Legacy format - parse directly
	var cfg ClusterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// DEBUG: Check how many pools were parsed from legacy YAML
	fmt.Printf("🔍 DEBUG [yaml_loader.go LEGACY]: Parsed %d node pools from YAML\n", len(cfg.NodePools))
	for poolName, pool := range cfg.NodePools {
		fmt.Printf("🔍 DEBUG [yaml_loader.go LEGACY]: Pool '%s' - provider=%s, count=%d\n", poolName, pool.Provider, pool.Count)
	}

	// DEBUG: Check bastion configuration
	if cfg.Security.Bastion == nil {
		fmt.Printf("🔍 DEBUG [yaml_loader.go LEGACY]: cfg.Security.Bastion is NIL after parsing\n")
	} else {
		fmt.Printf("🔍 DEBUG [yaml_loader.go LEGACY]: cfg.Security.Bastion.Enabled = %v\n", cfg.Security.Bastion.Enabled)
		fmt.Printf("🔍 DEBUG [yaml_loader.go LEGACY]: cfg.Security.Bastion.Provider = %s\n", cfg.Security.Bastion.Provider)
	}

	// Apply defaults
	applyDefaults(&cfg)

	return &cfg, nil
}

// SaveToYAML saves cluster configuration to a YAML file
func SaveToYAML(cfg *ClusterConfig, filePath string) error {
	// Expand home directory if needed
	if len(filePath) > 0 && filePath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		filePath = filepath.Join(home, filePath[1:])
	}

	// Create directory if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// applyDefaults applies default values to configuration
func applyDefaults(cfg *ClusterConfig) {
	// Kubernetes defaults
	if cfg.Kubernetes.Distribution == "" {
		cfg.Kubernetes.Distribution = "rke2"
	}
	if cfg.Kubernetes.Version == "" {
		cfg.Kubernetes.Version = "v1.28.5+rke2r1"
	}
	if cfg.Kubernetes.NetworkPlugin == "" {
		cfg.Kubernetes.NetworkPlugin = "calico"
	}
	if cfg.Kubernetes.PodCIDR == "" {
		cfg.Kubernetes.PodCIDR = "10.42.0.0/16"
	}
	if cfg.Kubernetes.ServiceCIDR == "" {
		cfg.Kubernetes.ServiceCIDR = "10.43.0.0/16"
	}
	if cfg.Kubernetes.ClusterDNS == "" {
		cfg.Kubernetes.ClusterDNS = "10.43.0.10"
	}
	if cfg.Kubernetes.ClusterDomain == "" {
		cfg.Kubernetes.ClusterDomain = "cluster.local"
	}

	// RKE2 defaults
	if cfg.Kubernetes.Distribution == "rke2" && cfg.Kubernetes.RKE2 == nil {
		cfg.Kubernetes.RKE2 = GetRKE2Defaults()
	} else if cfg.Kubernetes.RKE2 != nil {
		cfg.Kubernetes.RKE2 = MergeRKE2Config(cfg.Kubernetes.RKE2, cfg.Kubernetes.Version)
	}

	// WireGuard defaults
	if cfg.Network.WireGuard != nil && cfg.Network.WireGuard.Enabled {
		if cfg.Network.WireGuard.ClientIPBase == "" {
			cfg.Network.WireGuard.ClientIPBase = "10.100.0.0/24"
		}
		if cfg.Network.WireGuard.Port == 0 {
			cfg.Network.WireGuard.Port = 51820
		}
		if cfg.Network.WireGuard.MTU == 0 {
			cfg.Network.WireGuard.MTU = 1420
		}
		if cfg.Network.WireGuard.PersistentKeepalive == 0 {
			cfg.Network.WireGuard.PersistentKeepalive = 25
		}
	}

	// Metadata defaults
	if cfg.Metadata.Name == "" {
		cfg.Metadata.Name = "kubernetes-cluster"
	}
	if cfg.Metadata.Environment == "" {
		cfg.Metadata.Environment = "production"
	}
}

// ValidateConfig validates the configuration
func ValidateConfig(cfg *ClusterConfig) error {
	// Required fields
	if cfg.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	// Validate providers - at least one must be enabled
	providersEnabled := false
	if cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.Enabled {
		if cfg.Providers.DigitalOcean.Token == "" {
			return fmt.Errorf("digitalocean token is required when provider is enabled")
		}
		providersEnabled = true
	}
	if cfg.Providers.Linode != nil && cfg.Providers.Linode.Enabled {
		if cfg.Providers.Linode.Token == "" {
			return fmt.Errorf("linode token is required when provider is enabled")
		}
		providersEnabled = true
	}
	if cfg.Providers.Azure != nil && cfg.Providers.Azure.Enabled {
		// Azure credentials are validated via Azure CLI or environment variables
		// No token validation needed here
		providersEnabled = true
	}
	if !providersEnabled {
		return fmt.Errorf("at least one cloud provider must be enabled")
	}

	// Validate node pools
	if len(cfg.NodePools) == 0 {
		return fmt.Errorf("at least one node pool is required")
	}

	// Count masters and workers
	masterCount := 0
	workerCount := 0
	for _, pool := range cfg.NodePools {
		for _, role := range pool.Roles {
			if role == "master" {
				masterCount += pool.Count
			} else if role == "worker" {
				workerCount += pool.Count
			}
		}
	}

	if masterCount == 0 {
		return fmt.Errorf("at least one master node is required")
	}
	if masterCount%2 == 0 {
		return fmt.Errorf("master count must be odd for HA (got %d)", masterCount)
	}
	if workerCount == 0 {
		return fmt.Errorf("at least one worker node is required")
	}

	// Validate WireGuard if enabled
	// NOTE: This validation is now handled by deployment_validator.go which considers the Create field
	// if cfg.Network.WireGuard != nil && cfg.Network.WireGuard.Enabled {
	// 	if cfg.Network.WireGuard.ServerEndpoint == "" {
	// 		return fmt.Errorf("wireguard.serverEndpoint is required when WireGuard is enabled")
	// 	}
	// 	if cfg.Network.WireGuard.ServerPublicKey == "" {
	// 		return fmt.Errorf("wireguard.serverPublicKey is required when WireGuard is enabled")
	// 	}
	// }

	// Validate Kubernetes
	if cfg.Kubernetes.Distribution == "" {
		return fmt.Errorf("kubernetes.distribution is required")
	}
	if cfg.Kubernetes.Distribution != "rke2" && cfg.Kubernetes.Distribution != "k3s" {
		return fmt.Errorf("only rke2 and k3s distributions are supported")
	}

	return nil
}

// GenerateExampleConfig generates an example configuration
func GenerateExampleConfig() *ClusterConfig {
	return &ClusterConfig{
		Metadata: Metadata{
			Name:        "production-cluster",
			Environment: "production",
			Description: "Multi-cloud Kubernetes cluster",
			Owner:       "devops-team",
			Labels: map[string]string{
				"managed-by": "kubernetes-create",
				"env":        "production",
			},
		},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled: true,
				Token:   "YOUR_DO_TOKEN",
				Region:  "nyc3",
				Tags:    []string{"kubernetes", "production"},
			},
			Linode: &LinodeProvider{
				Enabled:      true,
				Token:        "YOUR_LINODE_TOKEN",
				Region:       "us-east",
				RootPassword: "YOUR_SECURE_PASSWORD",
				Tags:         []string{"kubernetes", "production"},
			},
		},
		Network: NetworkConfig{
			DNS: DNSConfig{
				Domain:   "example.com",
				Provider: "digitalocean",
			},
			WireGuard: &WireGuardConfig{
				Enabled:             true,
				ServerEndpoint:      "vpn.example.com:51820",
				ServerPublicKey:     "YOUR_WIREGUARD_PUBLIC_KEY",
				ClientIPBase:        "10.100.0.0/24",
				Port:                51820,
				MTU:                 1420,
				PersistentKeepalive: 25,
			},
		},
		Kubernetes: KubernetesConfig{
			Distribution:  "rke2",
			Version:       "v1.28.5+rke2r1",
			NetworkPlugin: "calico",
			PodCIDR:       "10.42.0.0/16",
			ServiceCIDR:   "10.43.0.0/16",
			ClusterDNS:    "10.43.0.10",
			ClusterDomain: "cluster.local",
			RKE2: &RKE2Config{
				Channel:              "stable",
				ClusterToken:         "your-secure-cluster-token",
				TLSSan:               []string{"api.example.com"},
				DisableComponents:    []string{"rke2-ingress-nginx"},
				SnapshotScheduleCron: "0 */12 * * *",
				SnapshotRetention:    5,
				SecretsEncryption:    true,
				WriteKubeconfigMode:  "0600",
			},
		},
		NodePools: map[string]NodePool{
			"do-masters": {
				Name:     "do-masters",
				Provider: "digitalocean",
				Count:    1,
				Roles:    []string{"master"},
				Size:     "s-2vcpu-4gb",
				Image:    "ubuntu-22-04-x64",
				Region:   "nyc3",
			},
			"linode-masters": {
				Name:     "linode-masters",
				Provider: "linode",
				Count:    2,
				Roles:    []string{"master"},
				Size:     "g6-standard-2",
				Image:    "linode/ubuntu22.04",
				Region:   "us-east",
			},
			"do-workers": {
				Name:     "do-workers",
				Provider: "digitalocean",
				Count:    2,
				Roles:    []string{"worker"},
				Size:     "s-2vcpu-4gb",
				Image:    "ubuntu-22-04-x64",
				Region:   "nyc3",
			},
			"linode-workers": {
				Name:     "linode-workers",
				Provider: "linode",
				Count:    1,
				Roles:    []string{"worker"},
				Size:     "g6-standard-2",
				Image:    "linode/ubuntu22.04",
				Region:   "us-east",
			},
		},
	}
}
