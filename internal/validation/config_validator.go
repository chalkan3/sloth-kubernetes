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

	// Validate VPN configuration (WireGuard or Tailscale)
	if err := ValidateVPNConfig(cfg); err != nil {
		return fmt.Errorf("VPN validation failed: %w", err)
	}

	// Validate provider configuration
	if err := ValidateProviders(cfg); err != nil {
		return fmt.Errorf("provider validation failed: %w", err)
	}

	return nil
}

// ValidateVPNConfig validates VPN configuration (WireGuard or Tailscale)
func ValidateVPNConfig(cfg *config.ClusterConfig) error {
	wireGuardEnabled := cfg.Network.WireGuard != nil && cfg.Network.WireGuard.Enabled
	tailscaleEnabled := cfg.Network.Tailscale != nil && cfg.Network.Tailscale.Enabled

	// At least one VPN must be enabled for private cluster deployment
	if !wireGuardEnabled && !tailscaleEnabled {
		return fmt.Errorf("VPN (WireGuard or Tailscale) must be enabled for private cluster deployment")
	}

	// Cannot have both enabled
	if wireGuardEnabled && tailscaleEnabled {
		return fmt.Errorf("cannot enable both WireGuard and Tailscale - choose one VPN solution")
	}

	// Validate the specific VPN configuration
	if wireGuardEnabled {
		return ValidateWireGuardConfig(cfg)
	}

	return ValidateTailscaleConfig(cfg)
}

// ValidateWireGuardConfig validates WireGuard configuration
func ValidateWireGuardConfig(cfg *config.ClusterConfig) error {
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

// ValidateTailscaleConfig validates Tailscale/Headscale configuration
func ValidateTailscaleConfig(cfg *config.ClusterConfig) error {
	// If auto-creating Headscale server, validate creation parameters
	if cfg.Network.Tailscale.Create {
		if cfg.Network.Tailscale.Provider == "" {
			return fmt.Errorf("Tailscale provider is required when auto-creating Headscale server")
		}
		if cfg.Network.Tailscale.Region == "" {
			return fmt.Errorf("Tailscale region is required when auto-creating Headscale server")
		}
		// HeadscaleURL and AuthKey will be generated during deployment
		return nil
	}

	// If using existing Headscale, validate URL and auth key
	if cfg.Network.Tailscale.HeadscaleURL == "" {
		return fmt.Errorf("Headscale URL is required when using existing Headscale server")
	}

	if cfg.Network.Tailscale.AuthKey == "" {
		return fmt.Errorf("Tailscale auth key is required when using existing Headscale server")
	}

	return nil
}

// ValidateProviders validates cloud provider configuration
func ValidateProviders(cfg *config.ClusterConfig) error {
	doEnabled := cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.Enabled
	linodeEnabled := cfg.Providers.Linode != nil && cfg.Providers.Linode.Enabled
	awsEnabled := cfg.Providers.AWS != nil && cfg.Providers.AWS.Enabled
	azureEnabled := cfg.Providers.Azure != nil && cfg.Providers.Azure.Enabled
	gcpEnabled := cfg.Providers.GCP != nil && cfg.Providers.GCP.Enabled

	// Verify at least one provider is enabled
	if !doEnabled && !linodeEnabled && !awsEnabled && !azureEnabled && !gcpEnabled {
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

	// AWS credentials are validated via environment variables or config
	if awsEnabled && cfg.Providers.AWS.Region == "" {
		return fmt.Errorf("AWS region is required")
	}

	// Azure credentials are validated via environment variables or config
	if azureEnabled && cfg.Providers.Azure.Location == "" {
		return fmt.Errorf("Azure location is required")
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
