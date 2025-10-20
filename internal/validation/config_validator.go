package validation

import (
	"fmt"

	"sloth-kubernetes/pkg/config"
)

// ValidateClusterConfig performs comprehensive validation of cluster configuration
func ValidateClusterConfig(cfg *config.ClusterConfig) error {
	// Validate node distribution
	if err := ValidateNodeDistribution(cfg); err != nil {
		return fmt.Errorf("node distribution validation failed: %w", err)
	}

	// Validate WireGuard configuration
	if err := ValidateWireGuardConfig(cfg); err != nil {
		return fmt.Errorf("WireGuard validation failed: %w", err)
	}

	// Validate provider configuration
	if err := ValidateProviders(cfg); err != nil {
		return fmt.Errorf("provider validation failed: %w", err)
	}

	return nil
}

// ValidateWireGuardConfig validates WireGuard configuration
func ValidateWireGuardConfig(cfg *config.ClusterConfig) error {
	// Verify WireGuard is enabled (required for private cluster)
	if cfg.Network.WireGuard == nil || !cfg.Network.WireGuard.Enabled {
		return fmt.Errorf("WireGuard must be enabled for private cluster deployment")
	}

	// Verify WireGuard endpoint
	if cfg.Network.WireGuard.ServerEndpoint == "" {
		return fmt.Errorf("WireGuard server endpoint is required")
	}

	// Verify WireGuard public key
	if cfg.Network.WireGuard.ServerPublicKey == "" {
		return fmt.Errorf("WireGuard server public key is required")
	}

	return nil
}

// ValidateProviders validates cloud provider configuration
func ValidateProviders(cfg *config.ClusterConfig) error {
	doEnabled := cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.Enabled
	linodeEnabled := cfg.Providers.Linode != nil && cfg.Providers.Linode.Enabled

	// Verify at least one provider is enabled
	if !doEnabled && !linodeEnabled {
		return fmt.Errorf("at least one cloud provider must be enabled")
	}

	// For this specific deployment, both must be enabled
	if !doEnabled || !linodeEnabled {
		return fmt.Errorf("both DigitalOcean and Linode providers must be enabled")
	}

	// Verify DigitalOcean token
	if doEnabled && cfg.Providers.DigitalOcean.Token == "" {
		return fmt.Errorf("DigitalOcean API token is required")
	}

	// Verify Linode token
	if linodeEnabled && cfg.Providers.Linode.Token == "" {
		return fmt.Errorf("Linode API token is required")
	}

	return nil
}

// ValidateDNSConfig validates DNS configuration
func ValidateDNSConfig(cfg *config.ClusterConfig) error {
	if cfg.Network.DNS.Domain == "" {
		return fmt.Errorf("DNS domain is required")
	}

	if cfg.Network.DNS.Provider == "" {
		return fmt.Errorf("DNS provider is required")
	}

	return nil
}

// ValidateMetadata validates cluster metadata
func ValidateMetadata(cfg *config.ClusterConfig) error {
	if cfg.Metadata.Name == "" {
		return fmt.Errorf("cluster name is required")
	}

	return nil
}
