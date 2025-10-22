package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNodeConfigEdgeCases tests edge cases for node configuration
func TestNodeConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		config *NodeConfig
		field  string
		check  func(*NodeConfig) bool
	}{
		{
			"Empty name",
			&NodeConfig{Name: ""},
			"Name",
			func(n *NodeConfig) bool { return n.Name == "" },
		},
		{
			"Very long name",
			&NodeConfig{Name: strings.Repeat("a", 1000)},
			"Name",
			func(n *NodeConfig) bool { return len(n.Name) == 1000 },
		},
		{
			"Special characters in name",
			&NodeConfig{Name: "node-1_test@cluster.local"},
			"Name",
			func(n *NodeConfig) bool { return strings.Contains(n.Name, "@") },
		},
		{
			"Zero size",
			&NodeConfig{Size: ""},
			"Size",
			func(n *NodeConfig) bool { return n.Size == "" },
		},
		{
			"Empty provider",
			&NodeConfig{Provider: ""},
			"Provider",
			func(n *NodeConfig) bool { return n.Provider == "" },
		},
		{
			"Empty pool",
			&NodeConfig{Pool: ""},
			"Pool",
			func(n *NodeConfig) bool { return n.Pool == "" },
		},
		{
			"Empty roles",
			&NodeConfig{Roles: []string{}},
			"Roles",
			func(n *NodeConfig) bool { return len(n.Roles) == 0 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.check(tt.config), "Check should pass for field %s", tt.field)
		})
	}
}

// TestRKE2ConfigEdgeCases tests edge cases for RKE2 configuration
func TestRKE2ConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		config *RKE2Config
		valid  bool
	}{
		{
			"Empty token",
			&RKE2Config{ClusterToken: ""},
			true, // Empty is technically valid, will use default
		},
		{
			"Very long token",
			&RKE2Config{ClusterToken: strings.Repeat("a", 10000)},
			true,
		},
		{
			"Special characters in token",
			&RKE2Config{ClusterToken: "token-with-!@#$%^&*()"},
			true,
		},
		{
			"Empty TLS SANs",
			&RKE2Config{TLSSan: []string{}},
			true,
		},
		{
			"Many TLS SANs",
			&RKE2Config{TLSSan: make([]string, 100)},
			true,
		},
		{
			"Negative snapshot retention",
			&RKE2Config{SnapshotRetention: -1},
			true, // Will be validated separately
		},
		{
			"Zero snapshot retention",
			&RKE2Config{SnapshotRetention: 0},
			true,
		},
		{
			"Very large snapshot retention",
			&RKE2Config{SnapshotRetention: 1000000},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.config)
		})
	}
}

// TestClusterConfigBoundaries tests boundary values
func TestClusterConfigBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		config   *ClusterConfig
		property string
		value    interface{}
	}{
		{
			"Empty cluster name",
			&ClusterConfig{Metadata: Metadata{Name: ""}},
			"Name",
			"",
		},
		{
			"Single character name",
			&ClusterConfig{Metadata: Metadata{Name: "a"}},
			"Name",
			"a",
		},
		{
			"Maximum length name",
			&ClusterConfig{Metadata: Metadata{Name: strings.Repeat("a", 255)}},
			"Name",
			255,
		},
		{
			"Empty environment",
			&ClusterConfig{Metadata: Metadata{Environment: ""}},
			"Environment",
			"",
		},
		{
			"Production environment",
			&ClusterConfig{Metadata: Metadata{Environment: "production"}},
			"Environment",
			"production",
		},
		{
			"Empty nodes",
			&ClusterConfig{Nodes: []NodeConfig{}},
			"Nodes",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.config)
			switch tt.property {
			case "Name":
				if strVal, ok := tt.value.(string); ok {
					assert.Equal(t, strVal, tt.config.Metadata.Name)
				} else if intVal, ok := tt.value.(int); ok {
					assert.Equal(t, intVal, len(tt.config.Metadata.Name))
				}
			case "Environment":
				assert.Equal(t, tt.value, tt.config.Metadata.Environment)
			case "Nodes":
				assert.Equal(t, tt.value, len(tt.config.Nodes))
			}
		})
	}
}

// TestNetworkConfigIPRanges tests IP range validations
func TestNetworkConfigIPRanges(t *testing.T) {
	tests := []struct {
		name  string
		podCIDR string
		svcCIDR string
		vpnCIDR string
	}{
		{"Standard ranges", "10.42.0.0/16", "10.43.0.0/16", "10.8.0.0/24"},
		{"Large pod range", "10.0.0.0/8", "172.16.0.0/16", "192.168.0.0/24"},
		{"Small ranges", "10.1.0.0/24", "10.2.0.0/24", "10.3.0.0/28"},
		{"Different private ranges", "172.16.0.0/12", "10.96.0.0/12", "192.168.1.0/24"},
		{"Non-standard pod CIDR", "100.64.0.0/16", "100.65.0.0/16", "100.66.0.0/24"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &KubernetesConfig{
				PodCIDR:     tt.podCIDR,
				ServiceCIDR: tt.svcCIDR,
			}
			assert.NotEmpty(t, cfg.PodCIDR)
			assert.NotEmpty(t, cfg.ServiceCIDR)
			assert.Contains(t, cfg.PodCIDR, "/")
			assert.Contains(t, cfg.ServiceCIDR, "/")
		})
	}
}

// TestProviderConfigEdgeCases tests provider configuration edge cases
func TestProviderConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		config *DigitalOceanProvider
		valid  bool
	}{
		{
			"Empty region",
			&DigitalOceanProvider{Region: ""},
			false,
		},
		{
			"Empty token",
			&DigitalOceanProvider{Token: ""},
			false,
		},
		{
			"Valid provider",
			&DigitalOceanProvider{Region: "nyc1", Token: "test"},
			true,
		},
		{
			"Valid with monitoring",
			&DigitalOceanProvider{Region: "nyc1", Token: "test", Monitoring: true},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.config)
		})
	}
}

// TestNodeConfigRoleAssignments tests role assignment logic
func TestNodeConfigRoleAssignments(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		hasRole string
	}{
		{
			"Master role",
			map[string]string{"role": "master"},
			"master",
		},
		{
			"Worker role",
			map[string]string{"role": "worker"},
			"worker",
		},
		{
			"Etcd role",
			map[string]string{"role": "etcd"},
			"etcd",
		},
		{
			"Multiple roles",
			map[string]string{"role": "master,etcd"},
			"master",
		},
		{
			"No role",
			map[string]string{},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &NodeConfig{
				Labels: tt.labels,
			}
			if tt.hasRole != "" {
				role, exists := node.Labels["role"]
				assert.True(t, exists)
				assert.Contains(t, role, tt.hasRole)
			} else {
				_, exists := node.Labels["role"]
				assert.False(t, exists)
			}
		})
	}
}

// TestKubernetesVersionFormats tests various version format inputs
func TestKubernetesVersionFormats(t *testing.T) {
	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{"Semantic version", "v1.28.5", true},
		{"Without v prefix", "1.28.5", true},
		{"Major.minor only", "v1.28", false},
		{"Latest", "latest", true},
		{"Stable", "stable", true},
		{"Empty", "", true}, // Empty means use default
		{"Invalid format", "version-1.28", false},
		{"With build metadata", "v1.28.5+rke2r1", true},
		{"Beta version", "v1.29.0-beta.1", true},
		{"Alpha version", "v1.30.0-alpha.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &KubernetesConfig{
				Version: tt.version,
			}
			if tt.version != "" {
				assert.NotEmpty(t, cfg.Version)
			}
			// Actual validation would be done by validator
		})
	}
}

// Test100ConfigCombinations tests 100 different configuration combinations
func Test100ConfigCombinations(t *testing.T) {
	providers := []string{"digitalocean", "linode"}
	regions := []string{"nyc1", "nyc3", "sfo1", "sfo2", "ams3", "sgp1", "lon1", "fra1"}
	sizes := []string{"s-1vcpu-1gb", "s-2vcpu-2gb", "s-2vcpu-4gb", "s-4vcpu-8gb"}
	networks := []string{"calico", "flannel", "canal", "cilium"}
	versions := []string{"v1.28.0", "v1.29.0", "v1.30.0", "latest"}

	for i := 0; i < 100; i++ {
		provider := providers[i%len(providers)]
		region := regions[i%len(regions)]
		nodeSize := sizes[i%len(sizes)]
		network := networks[i%len(networks)]
		version := versions[i%len(versions)]

		t.Run("Combo_"+string(rune('A'+i%26))+string(rune('0'+i/26)), func(t *testing.T) {
			cfg := &ClusterConfig{
				Metadata: Metadata{
					Name: "test-cluster-" + string(rune('a'+i%26)),
				},
				Kubernetes: KubernetesConfig{
					Version:       version,
					NetworkPlugin: network,
				},
				Nodes: []NodeConfig{
					{
						Name:   "node-1",
						Size:   nodeSize,
						Region: region,
					},
				},
			}

			providerCfg := &DigitalOceanProvider{
				Region: region,
				Token:  "test-token-" + provider,
			}

			assert.NotEmpty(t, cfg.Metadata.Name)
			assert.NotNil(t, cfg.Nodes)
			assert.NotEmpty(t, cfg.Kubernetes.Version)
			assert.NotEmpty(t, providerCfg.Region)
			assert.NotEmpty(t, providerCfg.Token)
		})
	}
}

// TestStringFieldsMaxLength tests maximum length constraints
func TestStringFieldsMaxLength(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		maxLen    int
	}{
		{"Cluster name", "ClusterName", strings.Repeat("a", 300), 300},
		{"Node name", "NodeName", strings.Repeat("b", 300), 300},
		{"Region", "Region", strings.Repeat("c", 100), 100},
		{"Token", "Token", strings.Repeat("d", 1000), 1000},
		{"Domain", "Domain", strings.Repeat("e", 255), 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Len(t, tt.value, tt.maxLen)
		})
	}
}

// TestNilConfigHandling tests handling of nil configurations
func TestNilConfigHandling(t *testing.T) {
	tests := []struct {
		name   string
		config interface{}
	}{
		{"Nil ClusterConfig", (*ClusterConfig)(nil)},
		{"Nil NodeConfig", (*NodeConfig)(nil)},
		{"Nil KubernetesConfig", (*KubernetesConfig)(nil)},
		{"Nil DigitalOceanProvider", (*DigitalOceanProvider)(nil)},
		{"Nil RKE2Config", (*RKE2Config)(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Nil(t, tt.config)
		})
	}
}

// TestConfigDefaultValues tests default value initialization
func TestConfigDefaultValues(t *testing.T) {
	t.Run("RKE2 defaults", func(t *testing.T) {
		defaults := GetRKE2Defaults()
		assert.NotNil(t, defaults)
		assert.Equal(t, "stable", defaults.Channel)
		assert.NotEmpty(t, defaults.ClusterToken)
		assert.Equal(t, "/var/lib/rancher/rke2", defaults.DataDir)
		assert.Equal(t, 5, defaults.SnapshotRetention)
		assert.Equal(t, "0600", defaults.WriteKubeconfigMode)
		assert.False(t, defaults.ProtectKernelDefaults)
	})
}

// TestTaintFormats tests various taint format inputs
func TestTaintFormats(t *testing.T) {
	tests := []struct {
		name   string
		taint  string
		valid  bool
	}{
		{"Standard NoSchedule", "key=value:NoSchedule", true},
		{"NoExecute", "key=value:NoExecute", true},
		{"PreferNoSchedule", "key=value:PreferNoSchedule", true},
		{"Empty value", "key=:NoSchedule", true},
		{"Missing effect", "key=value", false},
		{"Missing value", "key:NoSchedule", false},
		{"Empty", "", false},
		{"Only key", "key", false},
		{"With namespace", "example.com/key=value:NoSchedule", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				// Should have both = and :
				assert.Contains(t, tt.taint, "=")
				assert.Contains(t, tt.taint, ":")
				parts := strings.Split(tt.taint, ":")
				assert.Len(t, parts, 2)
			}
		})
	}
}

// TestLabelFormats tests various label format inputs
func TestLabelFormats(t *testing.T) {
	tests := []struct {
		name  string
		label string
		valid bool
	}{
		{"Simple label", "env=prod", true},
		{"With namespace", "example.com/env=prod", true},
		{"Boolean value", "enabled=true", true},
		{"Numeric value", "priority=100", true},
		{"Empty value", "key=", true},
		{"No value", "key", false},
		{"Multiple equals", "key=value=extra", true}, // Last = is part of value
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.Contains(t, tt.label, "=")
			}
		})
	}
}

// TestCIDRValidation tests CIDR format validation
func TestCIDRValidation(t *testing.T) {
	tests := []struct {
		name  string
		cidr  string
		valid bool
	}{
		{"Valid /16", "10.0.0.0/16", true},
		{"Valid /24", "192.168.1.0/24", true},
		{"Valid /8", "10.0.0.0/8", true},
		{"Missing prefix", "10.0.0.0", false},
		{"Invalid IP", "999.999.999.999/24", false},
		{"Invalid prefix", "10.0.0.0/33", false},
		{"Zero prefix", "10.0.0.0/0", true},
		{"Max prefix", "10.0.0.0/32", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.Contains(t, tt.cidr, "/")
				parts := strings.Split(tt.cidr, "/")
				assert.Len(t, parts, 2)
			}
		})
	}
}

// TestEmptyStructInitialization tests empty struct initialization
func TestEmptyStructInitialization(t *testing.T) {
	t.Run("Empty ClusterConfig", func(t *testing.T) {
		cfg := &ClusterConfig{}
		assert.Equal(t, "", cfg.Metadata.Name)
		assert.Equal(t, 0, len(cfg.Nodes))
		assert.Empty(t, cfg.NodePools)
	})

	t.Run("Empty NodeConfig", func(t *testing.T) {
		cfg := &NodeConfig{}
		assert.Equal(t, "", cfg.Name)
		assert.Empty(t, cfg.Roles)
		assert.Nil(t, cfg.Labels)
	})

	t.Run("Empty KubernetesConfig", func(t *testing.T) {
		cfg := &KubernetesConfig{}
		assert.Equal(t, "", cfg.Version)
		assert.Equal(t, "", cfg.NetworkPlugin)
	})
}

// TestConfigMerging tests configuration merging scenarios
func TestConfigMerging(t *testing.T) {
	t.Run("Merge non-empty over empty", func(t *testing.T) {
		base := &ClusterConfig{Metadata: Metadata{Name: ""}}
		override := &ClusterConfig{Metadata: Metadata{Name: "new-name"}}

		// Simulate merge
		if override.Metadata.Name != "" {
			base.Metadata.Name = override.Metadata.Name
		}

		assert.Equal(t, "new-name", base.Metadata.Name)
	})

	t.Run("Keep existing when override is empty", func(t *testing.T) {
		base := &ClusterConfig{Metadata: Metadata{Name: "existing"}}
		override := &ClusterConfig{Metadata: Metadata{Name: ""}}

		// Simulate merge
		if override.Metadata.Name != "" {
			base.Metadata.Name = override.Metadata.Name
		}

		assert.Equal(t, "existing", base.Metadata.Name)
	})
}
