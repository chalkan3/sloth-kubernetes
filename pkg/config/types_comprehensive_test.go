package config

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestClusterConfig_JSONMarshaling(t *testing.T) {
	config := &ClusterConfig{
		Metadata: Metadata{
			Name:        "test-cluster",
			Environment: "production",
			Owner:       "platform-team",
		},
		Cluster: ClusterSpec{
			Type:    "rke2",
			Version: "v1.27.0",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	// Unmarshal back
	var decoded ClusterConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal from JSON: %v", err)
	}

	if decoded.Metadata.Name != config.Metadata.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Metadata.Name, config.Metadata.Name)
	}

	if decoded.Cluster.Type != config.Cluster.Type {
		t.Errorf("Type mismatch: got %q, want %q", decoded.Cluster.Type, config.Cluster.Type)
	}
}

func TestClusterConfig_YAMLMarshaling(t *testing.T) {
	config := &ClusterConfig{
		Metadata: Metadata{
			Name:        "test-cluster",
			Environment: "staging",
		},
		Cluster: ClusterSpec{
			Type:             "k3s",
			HighAvailability: true,
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal to YAML: %v", err)
	}

	// Unmarshal back
	var decoded ClusterConfig
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal from YAML: %v", err)
	}

	if decoded.Metadata.Name != config.Metadata.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Metadata.Name, config.Metadata.Name)
	}

	if decoded.Cluster.HighAvailability != config.Cluster.HighAvailability {
		t.Error("HighAvailability mismatch")
	}
}

func TestMetadata_AllFields(t *testing.T) {
	metadata := Metadata{
		Name:        "prod-cluster",
		Environment: "production",
		Version:     "1.0.0",
		Description: "Production Kubernetes cluster",
		Owner:       "devops@example.com",
		Team:        "Platform Team",
		Labels: map[string]string{
			"cost-center": "engineering",
			"project":     "microservices",
		},
		Annotations: map[string]string{
			"created-by": "terraform",
			"managed-by": "sloth-kubernetes",
		},
	}

	if metadata.Name == "" {
		t.Error("Name should not be empty")
	}
	if len(metadata.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(metadata.Labels))
	}
	if len(metadata.Annotations) != 2 {
		t.Errorf("Expected 2 annotations, got %d", len(metadata.Annotations))
	}
}

func TestClusterSpec_Types(t *testing.T) {
	types := []string{"rke", "rke2", "k3s", "eks", "gke", "aks"}

	for _, clusterType := range types {
		spec := ClusterSpec{
			Type:    clusterType,
			Version: "v1.27.0",
		}

		if spec.Type != clusterType {
			t.Errorf("Expected type %q, got %q", clusterType, spec.Type)
		}
	}
}

func TestDigitalOceanProvider_Structure(t *testing.T) {
	provider := &DigitalOceanProvider{
		Enabled:    true,
		Token:      "test-token",
		Region:     "nyc3",
		SSHKeys:    []string{"key1", "key2"},
		Tags:       []string{"production", "kubernetes"},
		Monitoring: true,
		IPv6:       true,
		Custom: map[string]interface{}{
			"droplet_size": "s-2vcpu-4gb",
		},
	}

	if !provider.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if provider.Region != "nyc3" {
		t.Errorf("Expected region 'nyc3', got %q", provider.Region)
	}
	if len(provider.SSHKeys) != 2 {
		t.Errorf("Expected 2 SSH keys, got %d", len(provider.SSHKeys))
	}
	if len(provider.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(provider.Tags))
	}
}

func TestLinodeProvider_Structure(t *testing.T) {
	provider := &LinodeProvider{
		Enabled:        true,
		Token:          "test-token",
		Region:         "us-east",
		PrivateIP:      true,
		AuthorizedKeys: []string{"ssh-rsa AAAA..."},
		Tags:           []string{"k8s", "production"},
	}

	if !provider.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if !provider.PrivateIP {
		t.Error("Expected PrivateIP to be true")
	}
	if len(provider.AuthorizedKeys) != 1 {
		t.Errorf("Expected 1 authorized key, got %d", len(provider.AuthorizedKeys))
	}
}

func TestAWSProvider_Structure(t *testing.T) {
	provider := &AWSProvider{
		Enabled:         true,
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "secret",
		Region:          "us-west-2",
		SecurityGroups:  []string{"sg-123", "sg-456"},
		KeyPair:         "my-keypair",
		IAMRole:         "arn:aws:iam::123456789012:role/k8s-role",
	}

	if !provider.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if provider.Region != "us-west-2" {
		t.Errorf("Expected region 'us-west-2', got %q", provider.Region)
	}
	if len(provider.SecurityGroups) != 2 {
		t.Errorf("Expected 2 security groups, got %d", len(provider.SecurityGroups))
	}
}

func TestAzureProvider_Structure(t *testing.T) {
	provider := &AzureProvider{
		Enabled:        true,
		SubscriptionID: "sub-123",
		TenantID:       "tenant-123",
		ClientID:       "client-123",
		ClientSecret:   "secret",
		Location:       "eastus",
	}

	if !provider.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if provider.Location != "eastus" {
		t.Errorf("Expected location 'eastus', got %q", provider.Location)
	}
}

func TestGCPProvider_Structure(t *testing.T) {
	provider := &GCPProvider{
		Enabled:     true,
		ProjectID:   "my-project",
		Credentials: "credentials.json",
		Region:      "us-central1",
		Zone:        "us-central1-a",
	}

	if !provider.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if provider.Region != "us-central1" {
		t.Errorf("Expected region 'us-central1', got %q", provider.Region)
	}
}

func TestProvidersConfig_MultiCloud(t *testing.T) {
	providers := ProvidersConfig{
		DigitalOcean: &DigitalOceanProvider{
			Enabled: true,
			Region:  "nyc3",
		},
		Linode: &LinodeProvider{
			Enabled: true,
			Region:  "us-east",
		},
	}

	if providers.DigitalOcean == nil {
		t.Error("DigitalOcean provider should not be nil")
	}
	if providers.Linode == nil {
		t.Error("Linode provider should not be nil")
	}
	if !providers.DigitalOcean.Enabled {
		t.Error("DigitalOcean should be enabled")
	}
	if !providers.Linode.Enabled {
		t.Error("Linode should be enabled")
	}
}

func TestClusterConfig_WithNodePools(t *testing.T) {
	config := &ClusterConfig{
		Metadata: Metadata{Name: "test"},
		NodePools: map[string]NodePool{
			"masters": {
				Name:     "masters",
				Provider: "digitalocean",
				Count:    3,
				Roles:    []string{"master"},
			},
			"workers": {
				Name:     "workers",
				Provider: "digitalocean",
				Count:    5,
				Roles:    []string{"worker"},
			},
		},
	}

	if len(config.NodePools) != 2 {
		t.Errorf("Expected 2 node pools, got %d", len(config.NodePools))
	}

	if config.NodePools["masters"].Count != 3 {
		t.Errorf("Expected 3 masters, got %d", config.NodePools["masters"].Count)
	}

	if config.NodePools["workers"].Count != 5 {
		t.Errorf("Expected 5 workers, got %d", config.NodePools["workers"].Count)
	}
}

func TestClusterConfig_WithAddons(t *testing.T) {
	config := &ClusterConfig{
		Metadata: Metadata{Name: "test"},
		Addons: map[string]interface{}{
			"ingress-nginx": map[string]interface{}{
				"enabled":  true,
				"replicas": 3,
			},
			"cert-manager": map[string]interface{}{
				"enabled": true,
				"version": "v1.12.0",
			},
		},
	}

	if len(config.Addons) != 2 {
		t.Errorf("Expected 2 addons, got %d", len(config.Addons))
	}

	nginx, ok := config.Addons["ingress-nginx"].(map[string]interface{})
	if !ok {
		t.Error("ingress-nginx should be a map")
	}

	if nginx["enabled"] != true {
		t.Error("ingress-nginx should be enabled")
	}
}

func TestClusterSpec_HighAvailability(t *testing.T) {
	tests := []struct {
		name string
		spec ClusterSpec
		want bool
	}{
		{
			name: "HA enabled",
			spec: ClusterSpec{HighAvailability: true},
			want: true,
		},
		{
			name: "HA disabled",
			spec: ClusterSpec{HighAvailability: false},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.spec.HighAvailability != tt.want {
				t.Errorf("HighAvailability = %v, want %v", tt.spec.HighAvailability, tt.want)
			}
		})
	}
}

func TestClusterSpec_MultiCloud(t *testing.T) {
	spec := ClusterSpec{
		Type:       "rke2",
		MultiCloud: true,
	}

	if !spec.MultiCloud {
		t.Error("Expected MultiCloud to be true")
	}
}

func TestMetadata_Labels(t *testing.T) {
	metadata := Metadata{
		Name: "test",
		Labels: map[string]string{
			"env":     "production",
			"team":    "platform",
			"project": "microservices",
		},
	}

	if len(metadata.Labels) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(metadata.Labels))
	}

	if metadata.Labels["env"] != "production" {
		t.Errorf("Expected env=production, got %q", metadata.Labels["env"])
	}
}

func TestMetadata_Annotations(t *testing.T) {
	metadata := Metadata{
		Name: "test",
		Annotations: map[string]string{
			"created-at":  "2024-01-01",
			"created-by":  "admin",
			"description": "Test cluster",
		},
	}

	if len(metadata.Annotations) != 3 {
		t.Errorf("Expected 3 annotations, got %d", len(metadata.Annotations))
	}
}

func TestDigitalOceanProvider_DefaultValues(t *testing.T) {
	provider := &DigitalOceanProvider{}

	if provider.Enabled {
		t.Error("Default Enabled should be false")
	}
	if provider.Monitoring {
		t.Error("Default Monitoring should be false")
	}
	if provider.IPv6 {
		t.Error("Default IPv6 should be false")
	}
}

func TestLinodeProvider_DefaultValues(t *testing.T) {
	provider := &LinodeProvider{}

	if provider.Enabled {
		t.Error("Default Enabled should be false")
	}
	if provider.PrivateIP {
		t.Error("Default PrivateIP should be false")
	}
}

func TestClusterConfig_EmptyNodePools(t *testing.T) {
	config := &ClusterConfig{
		Metadata:  Metadata{Name: "test"},
		NodePools: make(map[string]NodePool),
	}

	if config.NodePools == nil {
		t.Error("NodePools should not be nil")
	}

	if len(config.NodePools) != 0 {
		t.Errorf("Expected 0 node pools, got %d", len(config.NodePools))
	}
}

func TestClusterConfig_EmptyAddons(t *testing.T) {
	config := &ClusterConfig{
		Metadata: Metadata{Name: "test"},
		Addons:   make(map[string]interface{}),
	}

	if config.Addons == nil {
		t.Error("Addons should not be nil")
	}

	if len(config.Addons) != 0 {
		t.Errorf("Expected 0 addons, got %d", len(config.Addons))
	}
}

func TestProvidersConfig_AllProviders(t *testing.T) {
	providers := ProvidersConfig{
		DigitalOcean: &DigitalOceanProvider{Enabled: true},
		Linode:       &LinodeProvider{Enabled: true},
		AWS:          &AWSProvider{Enabled: true},
		Azure:        &AzureProvider{Enabled: true},
		GCP:          &GCPProvider{Enabled: true},
	}

	enabledCount := 0
	if providers.DigitalOcean != nil && providers.DigitalOcean.Enabled {
		enabledCount++
	}
	if providers.Linode != nil && providers.Linode.Enabled {
		enabledCount++
	}
	if providers.AWS != nil && providers.AWS.Enabled {
		enabledCount++
	}
	if providers.Azure != nil && providers.Azure.Enabled {
		enabledCount++
	}
	if providers.GCP != nil && providers.GCP.Enabled {
		enabledCount++
	}

	if enabledCount != 5 {
		t.Errorf("Expected 5 enabled providers, got %d", enabledCount)
	}
}

func TestMetadata_EmptyLabels(t *testing.T) {
	metadata := Metadata{
		Name:   "test",
		Labels: make(map[string]string),
	}

	if metadata.Labels == nil {
		t.Error("Labels should not be nil")
	}

	if len(metadata.Labels) != 0 {
		t.Errorf("Expected 0 labels, got %d", len(metadata.Labels))
	}
}

func TestMetadata_EmptyAnnotations(t *testing.T) {
	metadata := Metadata{
		Name:        "test",
		Annotations: make(map[string]string),
	}

	if metadata.Annotations == nil {
		t.Error("Annotations should not be nil")
	}

	if len(metadata.Annotations) != 0 {
		t.Errorf("Expected 0 annotations, got %d", len(metadata.Annotations))
	}
}

func TestDigitalOceanProvider_Tags(t *testing.T) {
	provider := &DigitalOceanProvider{
		Tags: []string{"production", "kubernetes", "us-east"},
	}

	if len(provider.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(provider.Tags))
	}

	expectedTags := map[string]bool{
		"production": true,
		"kubernetes": true,
		"us-east":    true,
	}

	for _, tag := range provider.Tags {
		if !expectedTags[tag] {
			t.Errorf("Unexpected tag: %s", tag)
		}
	}
}

func TestLinodeProvider_Tags(t *testing.T) {
	provider := &LinodeProvider{
		Tags: []string{"staging", "test"},
	}

	if len(provider.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(provider.Tags))
	}
}

func TestAWSProvider_SecurityGroups(t *testing.T) {
	provider := &AWSProvider{
		SecurityGroups: []string{"sg-123", "sg-456", "sg-789"},
	}

	if len(provider.SecurityGroups) != 3 {
		t.Errorf("Expected 3 security groups, got %d", len(provider.SecurityGroups))
	}
}

func TestClusterConfig_CompleteStructure(t *testing.T) {
	config := &ClusterConfig{
		Metadata: Metadata{
			Name:        "production-cluster",
			Environment: "production",
			Owner:       "platform@example.com",
			Labels: map[string]string{
				"cost-center": "engineering",
			},
		},
		Cluster: ClusterSpec{
			Type:             "rke2",
			Version:          "v1.27.0",
			HighAvailability: true,
			MultiCloud:       true,
		},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled:    true,
				Region:     "nyc3",
				Monitoring: true,
			},
			Linode: &LinodeProvider{
				Enabled: true,
				Region:  "us-east",
			},
		},
		NodePools: map[string]NodePool{
			"masters": {
				Name:     "masters",
				Provider: "digitalocean",
				Count:    3,
				Roles:    []string{"master"},
			},
		},
	}

	// Validate structure
	if config.Metadata.Name != "production-cluster" {
		t.Error("Invalid cluster name")
	}
	if !config.Cluster.HighAvailability {
		t.Error("HA should be enabled")
	}
	if !config.Cluster.MultiCloud {
		t.Error("MultiCloud should be enabled")
	}
	if len(config.NodePools) != 1 {
		t.Error("Should have 1 node pool")
	}
}
