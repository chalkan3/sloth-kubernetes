package providers

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// Provider defines the interface for cloud providers
type Provider interface {
	// GetName returns the provider name
	GetName() string

	// Initialize initializes the provider with configuration
	Initialize(ctx *pulumi.Context, config *config.ClusterConfig) error

	// CreateNode creates a compute instance
	CreateNode(ctx *pulumi.Context, node *config.NodeConfig) (*NodeOutput, error)

	// CreateNodePool creates a pool of nodes
	CreateNodePool(ctx *pulumi.Context, pool *config.NodePool) ([]*NodeOutput, error)

	// CreateNetwork creates network infrastructure
	CreateNetwork(ctx *pulumi.Context, network *config.NetworkConfig) (*NetworkOutput, error)

	// CreateFirewall creates firewall rules
	CreateFirewall(ctx *pulumi.Context, firewall *config.FirewallConfig, nodeIds []pulumi.IDOutput) error

	// CreateLoadBalancer creates a load balancer
	CreateLoadBalancer(ctx *pulumi.Context, lb *config.LoadBalancerConfig) (*LoadBalancerOutput, error)

	// GetRegions returns available regions
	GetRegions() []string

	// GetSizes returns available instance sizes
	GetSizes() []string

	// Cleanup performs cleanup operations
	Cleanup(ctx *pulumi.Context) error
}

// NodeOutput represents the output of a created node
type NodeOutput struct {
	ID           pulumi.IDOutput
	Name         string
	PublicIP     pulumi.StringOutput
	PrivateIP    pulumi.StringOutput
	Provider     string
	Region       string
	Size         string
	Status       pulumi.StringOutput
	Labels       map[string]string
	WireGuardIP  string
	WireGuardKey pulumi.StringOutput
	SSHUser      string
	SSHKeyPath   string
}

// NetworkOutput represents network creation output
type NetworkOutput struct {
	ID      pulumi.IDOutput
	Name    string
	CIDR    string
	Region  string
	Subnets []SubnetOutput
}

// SubnetOutput represents subnet information
type SubnetOutput struct {
	ID   pulumi.IDOutput
	CIDR string
	Zone string
}

// LoadBalancerOutput represents load balancer output
type LoadBalancerOutput struct {
	ID       pulumi.IDOutput
	IP       pulumi.StringOutput
	Hostname pulumi.StringOutput
	Status   pulumi.StringOutput
}

// ProviderRegistry manages available providers
type ProviderRegistry struct {
	providers map[string]Provider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register registers a provider
func (r *ProviderRegistry) Register(name string, provider Provider) {
	r.providers[name] = provider
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	provider, ok := r.providers[name]
	return provider, ok
}

// GetAll returns all registered providers
func (r *ProviderRegistry) GetAll() map[string]Provider {
	return r.providers
}

// InitializeAll initializes all registered providers
func (r *ProviderRegistry) InitializeAll(ctx *pulumi.Context, config *config.ClusterConfig) error {
	for name, provider := range r.providers {
		if err := provider.Initialize(ctx, config); err != nil {
			return err
		}
		ctx.Log.Info(fmt.Sprintf("Provider %s initialized successfully", name), nil)
	}
	return nil
}
