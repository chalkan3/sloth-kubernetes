package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// KubernetesStyleConfig represents a Kubernetes-style configuration file
type KubernetesStyleConfig struct {
	APIVersion string                 `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                 `yaml:"kind" json:"kind"`
	Metadata   K8sMetadata            `yaml:"metadata" json:"metadata"`
	Spec       ClusterSpec2           `yaml:"spec" json:"spec"`
	Status     map[string]interface{} `yaml:"status,omitempty" json:"status,omitempty"`
}

// K8sMetadata follows Kubernetes metadata structure
type K8sMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Namespace   string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// ClusterSpec2 defines the cluster specification (Kubernetes-style)
type ClusterSpec2 struct {
	Providers  ProvidersSpec  `yaml:"providers" json:"providers"`
	Network    NetworkSpec    `yaml:"network" json:"network"`
	Kubernetes KubernetesSpec `yaml:"kubernetes" json:"kubernetes"`
	NodePools  []NodePoolSpec `yaml:"nodePools" json:"nodePools"`
}

// ProvidersSpec defines cloud providers
type ProvidersSpec struct {
	DigitalOcean *DigitalOceanSpec `yaml:"digitalocean,omitempty" json:"digitalocean,omitempty"`
	Linode       *LinodeSpec       `yaml:"linode,omitempty" json:"linode,omitempty"`
	AWS          *AWSSpec          `yaml:"aws,omitempty" json:"aws,omitempty"`
	GCP          *GCPSpec          `yaml:"gcp,omitempty" json:"gcp,omitempty"`
	Azure        *AzureSpec        `yaml:"azure,omitempty" json:"azure,omitempty"`
}

// DigitalOceanSpec provider configuration
type DigitalOceanSpec struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Token   string   `yaml:"token,omitempty" json:"token,omitempty"`
	Region  string   `yaml:"region" json:"region"`
	Tags    []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// LinodeSpec provider configuration
type LinodeSpec struct {
	Enabled      bool     `yaml:"enabled" json:"enabled"`
	Token        string   `yaml:"token,omitempty" json:"token,omitempty"`
	Region       string   `yaml:"region" json:"region"`
	RootPassword string   `yaml:"rootPassword,omitempty" json:"rootPassword,omitempty"`
	Tags         []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// AWSSpec provider configuration
type AWSSpec struct {
	Enabled         bool     `yaml:"enabled" json:"enabled"`
	AccessKeyID     string   `yaml:"accessKeyId,omitempty" json:"accessKeyId,omitempty"`
	SecretAccessKey string   `yaml:"secretAccessKey,omitempty" json:"secretAccessKey,omitempty"`
	Region          string   `yaml:"region" json:"region"`
	KeyPair         string   `yaml:"keyPair,omitempty" json:"keyPair,omitempty"`
	IAMRole         string   `yaml:"iamRole,omitempty" json:"iamRole,omitempty"`
}

// GCPSpec provider configuration
type GCPSpec struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	ProjectID   string `yaml:"projectId,omitempty" json:"projectId,omitempty"`
	Credentials string `yaml:"credentials,omitempty" json:"credentials,omitempty"`
	Region      string `yaml:"region" json:"region"`
	Zone        string `yaml:"zone,omitempty" json:"zone,omitempty"`
}

// AzureSpec provider configuration
type AzureSpec struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	SubscriptionID string `yaml:"subscriptionId,omitempty" json:"subscriptionId,omitempty"`
	TenantID       string `yaml:"tenantId,omitempty" json:"tenantId,omitempty"`
	ClientID       string `yaml:"clientId,omitempty" json:"clientId,omitempty"`
	ClientSecret   string `yaml:"clientSecret,omitempty" json:"clientSecret,omitempty"`
	ResourceGroup  string `yaml:"resourceGroup,omitempty" json:"resourceGroup,omitempty"`
	Location       string `yaml:"location" json:"location"`
}

// NetworkSpec defines network configuration
type NetworkSpec struct {
	DNS       DNSSpec        `yaml:"dns" json:"dns"`
	WireGuard *WireGuardSpec `yaml:"wireguard,omitempty" json:"wireguard,omitempty"`
}

// DNSSpec configuration
type DNSSpec struct {
	Domain   string `yaml:"domain" json:"domain"`
	Provider string `yaml:"provider" json:"provider"`
}

// WireGuardSpec VPN configuration
type WireGuardSpec struct {
	Enabled             bool   `yaml:"enabled" json:"enabled"`
	ServerEndpoint      string `yaml:"serverEndpoint" json:"serverEndpoint"`
	ServerPublicKey     string `yaml:"serverPublicKey" json:"serverPublicKey"`
	ClientIPBase        string `yaml:"clientIPBase,omitempty" json:"clientIPBase,omitempty"`
	Port                int    `yaml:"port,omitempty" json:"port,omitempty"`
	MTU                 int    `yaml:"mtu,omitempty" json:"mtu,omitempty"`
	PersistentKeepalive int    `yaml:"persistentKeepalive,omitempty" json:"persistentKeepalive,omitempty"`
}

// KubernetesSpec defines Kubernetes configuration
type KubernetesSpec struct {
	Distribution  string    `yaml:"distribution" json:"distribution"`
	Version       string    `yaml:"version" json:"version"`
	NetworkPlugin string    `yaml:"networkPlugin" json:"networkPlugin"`
	PodCIDR       string    `yaml:"podCIDR" json:"podCIDR"`
	ServiceCIDR   string    `yaml:"serviceCIDR" json:"serviceCIDR"`
	ClusterDNS    string    `yaml:"clusterDNS" json:"clusterDNS"`
	ClusterDomain string    `yaml:"clusterDomain,omitempty" json:"clusterDomain,omitempty"`
	RKE2          *RKE2Spec `yaml:"rke2,omitempty" json:"rke2,omitempty"`
}

// RKE2Spec RKE2-specific configuration
type RKE2Spec struct {
	Channel              string            `yaml:"channel,omitempty" json:"channel,omitempty"`
	ClusterToken         string            `yaml:"clusterToken" json:"clusterToken"`
	TLSSan               []string          `yaml:"tlsSan,omitempty" json:"tlsSan,omitempty"`
	DisableComponents    []string          `yaml:"disableComponents,omitempty" json:"disableComponents,omitempty"`
	SnapshotScheduleCron string            `yaml:"snapshotScheduleCron,omitempty" json:"snapshotScheduleCron,omitempty"`
	SnapshotRetention    int               `yaml:"snapshotRetention,omitempty" json:"snapshotRetention,omitempty"`
	SecretsEncryption    bool              `yaml:"secretsEncryption,omitempty" json:"secretsEncryption,omitempty"`
	WriteKubeconfigMode  string            `yaml:"writeKubeconfigMode,omitempty" json:"writeKubeconfigMode,omitempty"`
	ExtraServerArgs      map[string]string `yaml:"extraServerArgs,omitempty" json:"extraServerArgs,omitempty"`
	ExtraAgentArgs       map[string]string `yaml:"extraAgentArgs,omitempty" json:"extraAgentArgs,omitempty"`
}

// NodePoolSpec defines a node pool
type NodePoolSpec struct {
	Name     string            `yaml:"name" json:"name"`
	Provider string            `yaml:"provider" json:"provider"`
	Count    int               `yaml:"count" json:"count"`
	Roles    []string          `yaml:"roles" json:"roles"`
	Size     string            `yaml:"size" json:"size"`
	Image    string            `yaml:"image" json:"image"`
	Region   string            `yaml:"region" json:"region"`
	Labels   map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Taints   []TaintSpec       `yaml:"taints,omitempty" json:"taints,omitempty"`
}

// TaintSpec defines node taints
type TaintSpec struct {
	Key    string `yaml:"key" json:"key"`
	Value  string `yaml:"value" json:"value"`
	Effect string `yaml:"effect" json:"effect"`
}

// LoadFromK8sYAML loads a Kubernetes-style YAML configuration
func LoadFromK8sYAML(filePath string) (*ClusterConfig, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	dataStr := expandEnvVars(string(data))

	// Parse Kubernetes-style YAML
	var k8sConfig KubernetesStyleConfig
	if err := yaml.Unmarshal([]byte(dataStr), &k8sConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// DEBUG: Check how many pools were parsed
	fmt.Printf("🔍 DEBUG [k8s_style.go]: Parsed %d node pools from YAML\n", len(k8sConfig.Spec.NodePools))
	for i, pool := range k8sConfig.Spec.NodePools {
		fmt.Printf("🔍 DEBUG [k8s_style.go]: Pool[%d] = name=%s, provider=%s, count=%d\n", i, pool.Name, pool.Provider, pool.Count)
	}

	// Validate API version and kind
	if k8sConfig.APIVersion != "kubernetes-create.io/v1" && k8sConfig.APIVersion != "sloth-kubernetes.io/v1" && k8sConfig.APIVersion != "v1" {
		return nil, fmt.Errorf("unsupported apiVersion: %s (expected: sloth-kubernetes.io/v1 or kubernetes-create.io/v1)", k8sConfig.APIVersion)
	}
	if k8sConfig.Kind != "Cluster" {
		return nil, fmt.Errorf("unsupported kind: %s (expected: Cluster)", k8sConfig.Kind)
	}

	// Convert to internal ClusterConfig
	cfg := convertFromK8sStyle(&k8sConfig)

	// DEBUG: Check how many pools after conversion
	fmt.Printf("🔍 DEBUG [k8s_style.go]: After conversion: %d node pools\n", len(cfg.NodePools))
	for poolName, pool := range cfg.NodePools {
		fmt.Printf("🔍 DEBUG [k8s_style.go]: Converted pool '%s' - provider=%s, count=%d\n", poolName, pool.Provider, pool.Count)
	}

	// Apply defaults
	applyDefaults(cfg)

	return cfg, nil
}

// convertFromK8sStyle converts Kubernetes-style config to internal format
func convertFromK8sStyle(k8s *KubernetesStyleConfig) *ClusterConfig {
	cfg := &ClusterConfig{
		Metadata: Metadata{
			Name:        k8s.Metadata.Name,
			Labels:      k8s.Metadata.Labels,
			Annotations: k8s.Metadata.Annotations,
		},
		Providers:  ProvidersConfig{},
		Network:    NetworkConfig{},
		Kubernetes: KubernetesConfig{},
		NodePools:  make(map[string]NodePool),
	}

	// Providers
	if k8s.Spec.Providers.DigitalOcean != nil {
		cfg.Providers.DigitalOcean = &DigitalOceanProvider{
			Enabled: k8s.Spec.Providers.DigitalOcean.Enabled,
			Token:   k8s.Spec.Providers.DigitalOcean.Token,
			Region:  k8s.Spec.Providers.DigitalOcean.Region,
			Tags:    k8s.Spec.Providers.DigitalOcean.Tags,
		}
	}
	if k8s.Spec.Providers.Linode != nil {
		cfg.Providers.Linode = &LinodeProvider{
			Enabled:      k8s.Spec.Providers.Linode.Enabled,
			Token:        k8s.Spec.Providers.Linode.Token,
			Region:       k8s.Spec.Providers.Linode.Region,
			RootPassword: k8s.Spec.Providers.Linode.RootPassword,
			Tags:         k8s.Spec.Providers.Linode.Tags,
		}
	}
	if k8s.Spec.Providers.AWS != nil {
		cfg.Providers.AWS = &AWSProvider{
			Enabled:         k8s.Spec.Providers.AWS.Enabled,
			AccessKeyID:     k8s.Spec.Providers.AWS.AccessKeyID,
			SecretAccessKey: k8s.Spec.Providers.AWS.SecretAccessKey,
			Region:          k8s.Spec.Providers.AWS.Region,
			KeyPair:         k8s.Spec.Providers.AWS.KeyPair,
			IAMRole:         k8s.Spec.Providers.AWS.IAMRole,
		}
	}
	if k8s.Spec.Providers.GCP != nil {
		cfg.Providers.GCP = &GCPProvider{
			Enabled:     k8s.Spec.Providers.GCP.Enabled,
			ProjectID:   k8s.Spec.Providers.GCP.ProjectID,
			Credentials: k8s.Spec.Providers.GCP.Credentials,
			Region:      k8s.Spec.Providers.GCP.Region,
			Zone:        k8s.Spec.Providers.GCP.Zone,
		}
	}
	if k8s.Spec.Providers.Azure != nil {
		cfg.Providers.Azure = &AzureProvider{
			Enabled:        k8s.Spec.Providers.Azure.Enabled,
			SubscriptionID: k8s.Spec.Providers.Azure.SubscriptionID,
			TenantID:       k8s.Spec.Providers.Azure.TenantID,
			ClientID:       k8s.Spec.Providers.Azure.ClientID,
			ClientSecret:   k8s.Spec.Providers.Azure.ClientSecret,
			ResourceGroup:  k8s.Spec.Providers.Azure.ResourceGroup,
			Location:       k8s.Spec.Providers.Azure.Location,
		}
	}

	// Network
	cfg.Network.DNS = DNSConfig{
		Domain:   k8s.Spec.Network.DNS.Domain,
		Provider: k8s.Spec.Network.DNS.Provider,
	}
	if k8s.Spec.Network.WireGuard != nil {
		cfg.Network.WireGuard = &WireGuardConfig{
			Enabled:             k8s.Spec.Network.WireGuard.Enabled,
			ServerEndpoint:      k8s.Spec.Network.WireGuard.ServerEndpoint,
			ServerPublicKey:     k8s.Spec.Network.WireGuard.ServerPublicKey,
			ClientIPBase:        k8s.Spec.Network.WireGuard.ClientIPBase,
			Port:                k8s.Spec.Network.WireGuard.Port,
			MTU:                 k8s.Spec.Network.WireGuard.MTU,
			PersistentKeepalive: k8s.Spec.Network.WireGuard.PersistentKeepalive,
		}
	}

	// Kubernetes
	cfg.Kubernetes = KubernetesConfig{
		Distribution:  k8s.Spec.Kubernetes.Distribution,
		Version:       k8s.Spec.Kubernetes.Version,
		NetworkPlugin: k8s.Spec.Kubernetes.NetworkPlugin,
		PodCIDR:       k8s.Spec.Kubernetes.PodCIDR,
		ServiceCIDR:   k8s.Spec.Kubernetes.ServiceCIDR,
		ClusterDNS:    k8s.Spec.Kubernetes.ClusterDNS,
		ClusterDomain: k8s.Spec.Kubernetes.ClusterDomain,
	}
	if k8s.Spec.Kubernetes.RKE2 != nil {
		cfg.Kubernetes.RKE2 = &RKE2Config{
			Channel:              k8s.Spec.Kubernetes.RKE2.Channel,
			ClusterToken:         k8s.Spec.Kubernetes.RKE2.ClusterToken,
			TLSSan:               k8s.Spec.Kubernetes.RKE2.TLSSan,
			DisableComponents:    k8s.Spec.Kubernetes.RKE2.DisableComponents,
			SnapshotScheduleCron: k8s.Spec.Kubernetes.RKE2.SnapshotScheduleCron,
			SnapshotRetention:    k8s.Spec.Kubernetes.RKE2.SnapshotRetention,
			SecretsEncryption:    k8s.Spec.Kubernetes.RKE2.SecretsEncryption,
			WriteKubeconfigMode:  k8s.Spec.Kubernetes.RKE2.WriteKubeconfigMode,
			ExtraServerArgs:      k8s.Spec.Kubernetes.RKE2.ExtraServerArgs,
			ExtraAgentArgs:       k8s.Spec.Kubernetes.RKE2.ExtraAgentArgs,
		}
	}

	// Node pools
	for _, pool := range k8s.Spec.NodePools {
		taints := make([]TaintConfig, len(pool.Taints))
		for i, t := range pool.Taints {
			taints[i] = TaintConfig{
				Key:    t.Key,
				Value:  t.Value,
				Effect: t.Effect,
			}
		}

		cfg.NodePools[pool.Name] = NodePool{
			Name:     pool.Name,
			Provider: pool.Provider,
			Count:    pool.Count,
			Roles:    pool.Roles,
			Size:     pool.Size,
			Image:    pool.Image,
			Region:   pool.Region,
			Labels:   pool.Labels,
			Taints:   taints,
		}
	}

	return cfg
}

// GenerateK8sStyleConfig generates a Kubernetes-style configuration example
func GenerateK8sStyleConfig() *KubernetesStyleConfig {
	return &KubernetesStyleConfig{
		APIVersion: "kubernetes-create.io/v1",
		Kind:       "Cluster",
		Metadata: K8sMetadata{
			Name: "production-cluster",
			Labels: map[string]string{
				"env":        "production",
				"managed-by": "kubernetes-create",
			},
			Annotations: map[string]string{
				"description": "Multi-cloud Kubernetes cluster",
			},
		},
		Spec: ClusterSpec2{
			Providers: ProvidersSpec{
				DigitalOcean: &DigitalOceanSpec{
					Enabled: true,
					Token:   "${DIGITALOCEAN_TOKEN}",
					Region:  "nyc3",
					Tags:    []string{"kubernetes", "production"},
				},
				Linode: &LinodeSpec{
					Enabled:      true,
					Token:        "${LINODE_TOKEN}",
					Region:       "us-east",
					RootPassword: "${LINODE_ROOT_PASSWORD}",
					Tags:         []string{"kubernetes", "production"},
				},
			},
			Network: NetworkSpec{
				DNS: DNSSpec{
					Domain:   "example.com",
					Provider: "digitalocean",
				},
				WireGuard: &WireGuardSpec{
					Enabled:             true,
					ServerEndpoint:      "${WIREGUARD_ENDPOINT}",
					ServerPublicKey:     "${WIREGUARD_PUBKEY}",
					ClientIPBase:        "10.100.0.0/24",
					Port:                51820,
					MTU:                 1420,
					PersistentKeepalive: 25,
				},
			},
			Kubernetes: KubernetesSpec{
				Distribution:  "rke2",
				Version:       "v1.28.5+rke2r1",
				NetworkPlugin: "calico",
				PodCIDR:       "10.42.0.0/16",
				ServiceCIDR:   "10.43.0.0/16",
				ClusterDNS:    "10.43.0.10",
				ClusterDomain: "cluster.local",
				RKE2: &RKE2Spec{
					Channel:              "stable",
					ClusterToken:         "your-secure-cluster-token-here",
					TLSSan:               []string{"api.example.com", "kubernetes.example.com"},
					DisableComponents:    []string{"rke2-ingress-nginx"},
					SnapshotScheduleCron: "0 */12 * * *",
					SnapshotRetention:    5,
					SecretsEncryption:    true,
					WriteKubeconfigMode:  "0600",
				},
			},
			NodePools: []NodePoolSpec{
				{
					Name:     "do-masters",
					Provider: "digitalocean",
					Count:    1,
					Roles:    []string{"master"},
					Size:     "s-2vcpu-4gb",
					Image:    "ubuntu-22-04-x64",
					Region:   "nyc3",
					Labels: map[string]string{
						"node-role.kubernetes.io/master": "true",
					},
				},
				{
					Name:     "linode-masters",
					Provider: "linode",
					Count:    2,
					Roles:    []string{"master"},
					Size:     "g6-standard-2",
					Image:    "linode/ubuntu22.04",
					Region:   "us-east",
					Labels: map[string]string{
						"node-role.kubernetes.io/master": "true",
					},
				},
				{
					Name:     "do-workers",
					Provider: "digitalocean",
					Count:    2,
					Roles:    []string{"worker"},
					Size:     "s-2vcpu-4gb",
					Image:    "ubuntu-22-04-x64",
					Region:   "nyc3",
					Labels: map[string]string{
						"node-role.kubernetes.io/worker": "true",
					},
				},
				{
					Name:     "linode-workers",
					Provider: "linode",
					Count:    1,
					Roles:    []string{"worker"},
					Size:     "g6-standard-2",
					Image:    "linode/ubuntu22.04",
					Region:   "us-east",
					Labels: map[string]string{
						"node-role.kubernetes.io/worker": "true",
					},
				},
			},
		},
	}
}

// SaveK8sStyleConfig saves a Kubernetes-style configuration to YAML
func SaveK8sStyleConfig(cfg *KubernetesStyleConfig, filePath string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// expandEnvVars expands environment variables in the format ${VAR_NAME} or $VAR_NAME
func expandEnvVars(s string) string {
	// Match ${VAR_NAME} pattern
	re := regexp.MustCompile(`\$\{([A-Za-z0-9_]+)\}`)
	result := re.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name (remove ${ and })
		varName := match[2 : len(match)-1]
		// Get environment variable value
		if val := os.Getenv(varName); val != "" {
			return val
		}
		// Keep original if not found
		return match
	})

	// Also match $VAR_NAME pattern (without braces)
	re2 := regexp.MustCompile(`\$([A-Za-z0-9_]+)`)
	result = re2.ReplaceAllStringFunc(result, func(match string) string {
		// Extract variable name (remove $)
		varName := match[1:]
		// Don't replace if it was already in ${} format
		if strings.Contains(s, "${"+varName+"}") {
			return match
		}
		// Get environment variable value
		if val := os.Getenv(varName); val != "" {
			return val
		}
		// Keep original if not found
		return match
	})

	return result
}
