package validation

import (
	"fmt"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
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

	// If auto-creating VPN, validate creation parameters
	if cfg.Network.WireGuard.Create {
		if cfg.Network.WireGuard.Provider == "" {
			return fmt.Errorf("WireGuard provider is required when auto-creating VPN server")
		}
		if cfg.Network.WireGuard.Region == "" {
			return fmt.Errorf("WireGuard region is required when auto-creating VPN server")
		}
		// Endpoint and public key will be generated during deployment
		return nil
	}

	// If using existing VPN, validate endpoint and key
	if cfg.Network.WireGuard.ServerEndpoint == "" {
		return fmt.Errorf("WireGuard server endpoint is required when using existing VPN server")
	}

	if cfg.Network.WireGuard.ServerPublicKey == "" {
		return fmt.Errorf("WireGuard server public key is required when using existing VPN server")
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

	// Verify DigitalOcean token if enabled
	if doEnabled && cfg.Providers.DigitalOcean.Token == "" {
		return fmt.Errorf("DigitalOcean API token is required")
	}

	// Verify Linode token if enabled
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
