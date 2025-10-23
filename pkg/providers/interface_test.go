package providers

import (
	"testing"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Mock Provider for testing
type MockProvider struct {
	name     string
	regions  []string
	sizes    []string
	initErr  error
	cleanErr error
}

func (m *MockProvider) GetName() string {
	return m.name
}

func (m *MockProvider) Initialize(ctx *pulumi.Context, cfg *config.ClusterConfig) error {
	return m.initErr
}

func (m *MockProvider) CreateNode(ctx *pulumi.Context, node *config.NodeConfig) (*NodeOutput, error) {
	return nil, nil
}

func (m *MockProvider) CreateNodePool(ctx *pulumi.Context, pool *config.NodePool) ([]*NodeOutput, error) {
	return nil, nil
}

func (m *MockProvider) CreateNetwork(ctx *pulumi.Context, network *config.NetworkConfig) (*NetworkOutput, error) {
	return nil, nil
}

func (m *MockProvider) CreateFirewall(ctx *pulumi.Context, firewall *config.FirewallConfig, nodeIds []pulumi.IDOutput) error {
	return nil
}

func (m *MockProvider) CreateLoadBalancer(ctx *pulumi.Context, lb *config.LoadBalancerConfig) (*LoadBalancerOutput, error) {
	return nil, nil
}

func (m *MockProvider) GetRegions() []string {
	return m.regions
}

func (m *MockProvider) GetSizes() []string {
	return m.sizes
}

func (m *MockProvider) Cleanup(ctx *pulumi.Context) error {
	return m.cleanErr
}

func TestNodeOutput_Structure(t *testing.T) {
	node := &NodeOutput{
		Name:        "test-node",
		Provider:    "digitalocean",
		Region:      "nyc3",
		Size:        "s-2vcpu-4gb",
		Labels:      map[string]string{"role": "master"},
		WireGuardIP: "10.8.0.1",
		SSHUser:     "root",
		SSHKeyPath:  "/root/.ssh/id_rsa",
	}

	if node.Name != "test-node" {
		t.Errorf("Expected name 'test-node', got '%s'", node.Name)
	}

	if node.Provider != "digitalocean" {
		t.Errorf("Expected provider 'digitalocean', got '%s'", node.Provider)
	}

	if node.Region != "nyc3" {
		t.Errorf("Expected region 'nyc3', got '%s'", node.Region)
	}

	if len(node.Labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(node.Labels))
	}

	if node.Labels["role"] != "master" {
		t.Errorf("Expected role 'master', got '%s'", node.Labels["role"])
	}
}

func TestNetworkOutput_Structure(t *testing.T) {
	network := &NetworkOutput{
		Name:   "test-network",
		CIDR:   "10.0.0.0/16",
		Region: "nyc3",
		Subnets: []SubnetOutput{
			{CIDR: "10.0.1.0/24", Zone: "nyc3-a"},
			{CIDR: "10.0.2.0/24", Zone: "nyc3-b"},
		},
	}

	if network.Name != "test-network" {
		t.Errorf("Expected name 'test-network', got '%s'", network.Name)
	}

	if network.CIDR != "10.0.0.0/16" {
		t.Errorf("Expected CIDR '10.0.0.0/16', got '%s'", network.CIDR)
	}

	if len(network.Subnets) != 2 {
		t.Errorf("Expected 2 subnets, got %d", len(network.Subnets))
	}

	if network.Subnets[0].CIDR != "10.0.1.0/24" {
		t.Errorf("Expected subnet CIDR '10.0.1.0/24', got '%s'", network.Subnets[0].CIDR)
	}
}

func TestSubnetOutput_Structure(t *testing.T) {
	subnet := SubnetOutput{
		CIDR: "192.168.1.0/24",
		Zone: "us-east-1a",
	}

	if subnet.CIDR != "192.168.1.0/24" {
		t.Errorf("Expected CIDR '192.168.1.0/24', got '%s'", subnet.CIDR)
	}

	if subnet.Zone != "us-east-1a" {
		t.Errorf("Expected zone 'us-east-1a', got '%s'", subnet.Zone)
	}
}

func TestLoadBalancerOutput_Structure(t *testing.T) {
	lb := &LoadBalancerOutput{}

	// Just verify structure exists
	if lb == nil {
		t.Error("LoadBalancerOutput should not be nil")
	}
}

func TestNewProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	if registry == nil {
		t.Fatal("NewProviderRegistry should not return nil")
	}

	if registry.providers == nil {
		t.Error("providers map should be initialized")
	}

	if len(registry.providers) != 0 {
		t.Errorf("New registry should be empty, got %d providers", len(registry.providers))
	}
}

func TestProviderRegistry_Register(t *testing.T) {
	registry := NewProviderRegistry()

	mockProvider := &MockProvider{
		name:    "test-provider",
		regions: []string{"us-east-1", "us-west-2"},
		sizes:   []string{"small", "medium", "large"},
	}

	registry.Register("test", mockProvider)

	if len(registry.providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(registry.providers))
	}
}

func TestProviderRegistry_Get(t *testing.T) {
	registry := NewProviderRegistry()

	mockProvider := &MockProvider{
		name: "test-provider",
	}

	registry.Register("test", mockProvider)

	// Test successful get
	provider, ok := registry.Get("test")
	if !ok {
		t.Error("Provider 'test' should exist")
	}

	if provider == nil {
		t.Fatal("Retrieved provider should not be nil")
	}

	if provider.GetName() != "test-provider" {
		t.Errorf("Expected provider name 'test-provider', got '%s'", provider.GetName())
	}

	// Test non-existent provider
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("Provider 'nonexistent' should not exist")
	}
}

func TestProviderRegistry_GetAll(t *testing.T) {
	registry := NewProviderRegistry()

	// Register multiple providers
	registry.Register("provider1", &MockProvider{name: "Provider 1"})
	registry.Register("provider2", &MockProvider{name: "Provider 2"})
	registry.Register("provider3", &MockProvider{name: "Provider 3"})

	all := registry.GetAll()

	if len(all) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(all))
	}

	if _, ok := all["provider1"]; !ok {
		t.Error("provider1 should exist")
	}

	if _, ok := all["provider2"]; !ok {
		t.Error("provider2 should exist")
	}

	if _, ok := all["provider3"]; !ok {
		t.Error("provider3 should exist")
	}
}

func TestProviderRegistry_MultipleRegistrations(t *testing.T) {
	registry := NewProviderRegistry()

	provider1 := &MockProvider{name: "Provider 1"}
	provider2 := &MockProvider{name: "Provider 2"}

	registry.Register("test", provider1)
	registry.Register("test", provider2) // Overwrite

	retrieved, ok := registry.Get("test")
	if !ok {
		t.Fatal("Provider should exist")
	}

	// Should have the second provider (overwritten)
	if retrieved.GetName() != "Provider 2" {
		t.Errorf("Expected 'Provider 2', got '%s'", retrieved.GetName())
	}
}

func TestMockProvider_GetName(t *testing.T) {
	provider := &MockProvider{
		name: "my-test-provider",
	}

	if provider.GetName() != "my-test-provider" {
		t.Errorf("Expected 'my-test-provider', got '%s'", provider.GetName())
	}
}

func TestMockProvider_GetRegions(t *testing.T) {
	provider := &MockProvider{
		regions: []string{"us-east-1", "us-west-2", "eu-west-1"},
	}

	regions := provider.GetRegions()

	if len(regions) != 3 {
		t.Errorf("Expected 3 regions, got %d", len(regions))
	}

	expectedRegions := map[string]bool{
		"us-east-1": true,
		"us-west-2": true,
		"eu-west-1": true,
	}

	for _, region := range regions {
		if !expectedRegions[region] {
			t.Errorf("Unexpected region: %s", region)
		}
	}
}

func TestMockProvider_GetSizes(t *testing.T) {
	provider := &MockProvider{
		sizes: []string{"s-1vcpu-1gb", "s-2vcpu-4gb", "s-4vcpu-8gb"},
	}

	sizes := provider.GetSizes()

	if len(sizes) != 3 {
		t.Errorf("Expected 3 sizes, got %d", len(sizes))
	}

	if sizes[0] != "s-1vcpu-1gb" {
		t.Errorf("Expected 's-1vcpu-1gb', got '%s'", sizes[0])
	}
}

func TestNodeOutput_Labels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   int
	}{
		{
			name:   "No labels",
			labels: map[string]string{},
			want:   0,
		},
		{
			name: "Single label",
			labels: map[string]string{
				"role": "master",
			},
			want: 1,
		},
		{
			name: "Multiple labels",
			labels: map[string]string{
				"role":        "worker",
				"environment": "production",
				"team":        "platform",
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &NodeOutput{
				Labels: tt.labels,
			}

			if len(node.Labels) != tt.want {
				t.Errorf("Expected %d labels, got %d", tt.want, len(node.Labels))
			}
		})
	}
}

func TestNodeOutput_WireGuardConfig(t *testing.T) {
	node := &NodeOutput{
		Name:        "vpn-node",
		WireGuardIP: "10.8.0.5",
	}

	if node.WireGuardIP != "10.8.0.5" {
		t.Errorf("Expected WireGuardIP '10.8.0.5', got '%s'", node.WireGuardIP)
	}
}

func TestNodeOutput_SSHConfig(t *testing.T) {
	node := &NodeOutput{
		SSHUser:    "ubuntu",
		SSHKeyPath: "/home/user/.ssh/id_rsa",
	}

	if node.SSHUser != "ubuntu" {
		t.Errorf("Expected SSHUser 'ubuntu', got '%s'", node.SSHUser)
	}

	if node.SSHKeyPath != "/home/user/.ssh/id_rsa" {
		t.Errorf("Expected SSHKeyPath '/home/user/.ssh/id_rsa', got '%s'", node.SSHKeyPath)
	}
}

func TestNetworkOutput_Subnets(t *testing.T) {
	network := &NetworkOutput{
		Name: "test",
		Subnets: []SubnetOutput{
			{CIDR: "10.0.1.0/24", Zone: "a"},
			{CIDR: "10.0.2.0/24", Zone: "b"},
			{CIDR: "10.0.3.0/24", Zone: "c"},
		},
	}

	if len(network.Subnets) != 3 {
		t.Errorf("Expected 3 subnets, got %d", len(network.Subnets))
	}

	for i, subnet := range network.Subnets {
		if subnet.CIDR == "" {
			t.Errorf("Subnet %d CIDR should not be empty", i)
		}
		if subnet.Zone == "" {
			t.Errorf("Subnet %d Zone should not be empty", i)
		}
	}
}

func TestProviderRegistry_EmptyRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	all := registry.GetAll()
	if len(all) != 0 {
		t.Errorf("Empty registry should return 0 providers, got %d", len(all))
	}

	_, ok := registry.Get("anything")
	if ok {
		t.Error("Empty registry should not find any provider")
	}
}

func TestNodeOutput_Providers(t *testing.T) {
	tests := []struct {
		provider string
		region   string
	}{
		{"digitalocean", "nyc3"},
		{"linode", "us-east"},
		{"aws", "us-west-2"},
		{"azure", "eastus"},
		{"gcp", "us-central1"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			node := &NodeOutput{
				Provider: tt.provider,
				Region:   tt.region,
			}

			if node.Provider != tt.provider {
				t.Errorf("Expected provider %q, got %q", tt.provider, node.Provider)
			}

			if node.Region != tt.region {
				t.Errorf("Expected region %q, got %q", tt.region, node.Region)
			}
		})
	}
}

func TestNodeOutput_Sizes(t *testing.T) {
	sizes := []string{
		"s-1vcpu-1gb",
		"s-2vcpu-4gb",
		"s-4vcpu-8gb",
		"g6-standard-2",
		"t2.micro",
		"t3.medium",
	}

	for _, size := range sizes {
		t.Run(size, func(t *testing.T) {
			node := &NodeOutput{
				Size: size,
			}

			if node.Size != size {
				t.Errorf("Expected size %q, got %q", size, node.Size)
			}
		})
	}
}
