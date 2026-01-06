package providers

import (
	"fmt"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ProviderFactory creates and registers all available cloud providers
type ProviderFactory struct {
	registry *ProviderRegistry
}

// NewProviderFactory creates a new provider factory with all providers registered
func NewProviderFactory() *ProviderFactory {
	factory := &ProviderFactory{
		registry: NewProviderRegistry(),
	}

	// Register all 5 cloud providers
	factory.registerAllProviders()

	return factory
}

// registerAllProviders registers all supported cloud providers
func (f *ProviderFactory) registerAllProviders() {
	// DigitalOcean provider
	f.registry.Register("digitalocean", NewDigitalOceanProvider())

	// Linode provider
	f.registry.Register("linode", NewLinodeProvider())

	// AWS provider
	f.registry.Register("aws", NewAWSProvider())

	// GCP provider
	f.registry.Register("gcp", NewGCPProvider())

	// Azure provider
	f.registry.Register("azure", NewAzureProvider())

	// Hetzner provider
	f.registry.Register("hetzner", NewHetznerProvider())
}

// GetRegistry returns the provider registry
func (f *ProviderFactory) GetRegistry() *ProviderRegistry {
	return f.registry
}

// GetProvider returns a specific provider by name
func (f *ProviderFactory) GetProvider(name string) (Provider, error) {
	provider, ok := f.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// GetEnabledProviders returns only the providers that are enabled in the config
func (f *ProviderFactory) GetEnabledProviders(cfg *config.ClusterConfig) ([]Provider, error) {
	enabledProviders := make([]Provider, 0)

	// Check DigitalOcean
	if cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.Enabled {
		provider, err := f.GetProvider("digitalocean")
		if err != nil {
			return nil, err
		}
		enabledProviders = append(enabledProviders, provider)
	}

	// Check Linode
	if cfg.Providers.Linode != nil && cfg.Providers.Linode.Enabled {
		provider, err := f.GetProvider("linode")
		if err != nil {
			return nil, err
		}
		enabledProviders = append(enabledProviders, provider)
	}

	// Check AWS
	if cfg.Providers.AWS != nil && cfg.Providers.AWS.Enabled {
		provider, err := f.GetProvider("aws")
		if err != nil {
			return nil, err
		}
		enabledProviders = append(enabledProviders, provider)
	}

	// Check GCP
	if cfg.Providers.GCP != nil && cfg.Providers.GCP.Enabled {
		provider, err := f.GetProvider("gcp")
		if err != nil {
			return nil, err
		}
		enabledProviders = append(enabledProviders, provider)
	}

	// Check Azure
	if cfg.Providers.Azure != nil && cfg.Providers.Azure.Enabled {
		provider, err := f.GetProvider("azure")
		if err != nil {
			return nil, err
		}
		enabledProviders = append(enabledProviders, provider)
	}

	// Check Hetzner
	if cfg.Providers.Hetzner != nil && cfg.Providers.Hetzner.Enabled {
		provider, err := f.GetProvider("hetzner")
		if err != nil {
			return nil, err
		}
		enabledProviders = append(enabledProviders, provider)
	}

	if len(enabledProviders) == 0 {
		return nil, fmt.Errorf("no providers enabled in configuration")
	}

	return enabledProviders, nil
}

// InitializeEnabledProviders initializes only the enabled providers
func (f *ProviderFactory) InitializeEnabledProviders(ctx *pulumi.Context, cfg *config.ClusterConfig) ([]Provider, error) {
	enabledProviders, err := f.GetEnabledProviders(cfg)
	if err != nil {
		return nil, err
	}

	for _, provider := range enabledProviders {
		if err := provider.Initialize(ctx, cfg); err != nil {
			return nil, fmt.Errorf("failed to initialize provider %s: %w", provider.GetName(), err)
		}
		ctx.Log.Info(fmt.Sprintf("Provider %s initialized successfully", provider.GetName()), nil)
	}

	return enabledProviders, nil
}

// GetProviderForNodePool returns the provider for a specific node pool
func (f *ProviderFactory) GetProviderForNodePool(pool *config.NodePool) (Provider, error) {
	if pool.Provider == "" {
		return nil, fmt.Errorf("node pool %s has no provider specified", pool.Name)
	}

	provider, err := f.GetProvider(pool.Provider)
	if err != nil {
		return nil, fmt.Errorf("provider %s for node pool %s not found: %w", pool.Provider, pool.Name, err)
	}

	return provider, nil
}

// ValidateProviderConfig validates provider configuration
func (f *ProviderFactory) ValidateProviderConfig(cfg *config.ClusterConfig) error {
	var errors []string

	// Validate DigitalOcean config if enabled
	if cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.Enabled {
		if cfg.Providers.DigitalOcean.Token == "" {
			errors = append(errors, "DigitalOcean: token is required")
		}
	}

	// Validate Linode config if enabled
	if cfg.Providers.Linode != nil && cfg.Providers.Linode.Enabled {
		if cfg.Providers.Linode.Token == "" {
			errors = append(errors, "Linode: token is required")
		}
	}

	// Validate AWS config if enabled
	if cfg.Providers.AWS != nil && cfg.Providers.AWS.Enabled {
		if cfg.Providers.AWS.Region == "" {
			errors = append(errors, "AWS: region is required")
		}
		// Access keys can be provided via environment variables or IAM roles
	}

	// Validate GCP config if enabled
	if cfg.Providers.GCP != nil && cfg.Providers.GCP.Enabled {
		if cfg.Providers.GCP.ProjectID == "" {
			errors = append(errors, "GCP: projectID is required")
		}
		if cfg.Providers.GCP.Region == "" {
			errors = append(errors, "GCP: region is required")
		}
		// Credentials can be provided via environment variables or application default credentials
	}

	// Validate Azure config if enabled
	if cfg.Providers.Azure != nil && cfg.Providers.Azure.Enabled {
		if cfg.Providers.Azure.Location == "" {
			errors = append(errors, "Azure: location is required")
		}
		// Subscription ID and credentials can be provided via environment variables or managed identity
	}

	// Validate Hetzner config if enabled
	if cfg.Providers.Hetzner != nil && cfg.Providers.Hetzner.Enabled {
		if cfg.Providers.Hetzner.Token == "" {
			errors = append(errors, "Hetzner: token is required")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("provider configuration validation failed:\n  - %s", joinErrors(errors))
	}

	return nil
}

// GetSupportedProviders returns a list of all supported provider names
func (f *ProviderFactory) GetSupportedProviders() []string {
	return []string{
		"digitalocean",
		"linode",
		"aws",
		"gcp",
		"azure",
		"hetzner",
	}
}

// GetProviderInfo returns information about all providers
func (f *ProviderFactory) GetProviderInfo() map[string]ProviderInfo {
	return map[string]ProviderInfo{
		"digitalocean": {
			Name:        "DigitalOcean",
			Code:        "digitalocean",
			Description: "DigitalOcean Droplets - Cost-effective cloud compute",
			Regions:     NewDigitalOceanProvider().GetRegions(),
			Sizes:       NewDigitalOceanProvider().GetSizes(),
		},
		"linode": {
			Name:        "Linode",
			Code:        "linode",
			Description: "Linode Compute Instances - High-performance cloud",
			Regions:     NewLinodeProvider().GetRegions(),
			Sizes:       NewLinodeProvider().GetSizes(),
		},
		"aws": {
			Name:        "Amazon Web Services",
			Code:        "aws",
			Description: "AWS EC2 - Global cloud infrastructure",
			Regions:     NewAWSProvider().GetRegions(),
			Sizes:       NewAWSProvider().GetSizes(),
		},
		"gcp": {
			Name:        "Google Cloud Platform",
			Code:        "gcp",
			Description: "GCP Compute Engine - Google's cloud infrastructure",
			Regions:     NewGCPProvider().GetRegions(),
			Sizes:       NewGCPProvider().GetSizes(),
		},
		"azure": {
			Name:        "Microsoft Azure",
			Code:        "azure",
			Description: "Azure Virtual Machines - Microsoft's cloud platform",
			Regions:     NewAzureProvider().GetRegions(),
			Sizes:       NewAzureProvider().GetSizes(),
		},
		"hetzner": {
			Name:        "Hetzner Cloud",
			Code:        "hetzner",
			Description: "Hetzner Cloud - High-performance European cloud infrastructure",
			Regions:     NewHetznerProvider().GetRegions(),
			Sizes:       NewHetznerProvider().GetSizes(),
		},
	}
}

// ProviderInfo contains information about a cloud provider
type ProviderInfo struct {
	Name        string
	Code        string
	Description string
	Regions     []string
	Sizes       []string
}

// Helper function to join errors
func joinErrors(errors []string) string {
	result := ""
	for i, err := range errors {
		if i > 0 {
			result += "\n  - "
		}
		result += err
	}
	return result
}
