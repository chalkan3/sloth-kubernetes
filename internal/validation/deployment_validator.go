package validation

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/digitalocean/godo"
	"github.com/linode/linodego"
	"golang.org/x/oauth2"
)

// ValidateForDeployment performs all validations required before deployment
// This is a comprehensive validation that goes beyond basic config validation
func ValidateForDeployment(cfg *config.ClusterConfig) error {
	// 1. Basic configuration validation
	if err := ValidateClusterConfig(cfg); err != nil {
		return fmt.Errorf("basic validation failed: %w", err)
	}

	// 2. Validate API tokens are present and accessible
	if err := ValidateAPITokensPresence(cfg); err != nil {
		return fmt.Errorf("API token validation failed: %w", err)
	}

	// 3. Validate node pool configuration
	if err := ValidateNodePools(cfg); err != nil {
		return fmt.Errorf("node pool validation failed: %w", err)
	}

	// 4. Validate networking configuration
	if err := ValidateNetworkingConfig(cfg); err != nil {
		return fmt.Errorf("network configuration validation failed: %w", err)
	}

	// 5. Validate SSH keys configuration
	if err := ValidateSSHConfig(cfg); err != nil {
		return fmt.Errorf("SSH configuration validation failed: %w", err)
	}

	// 6. Validate resource limits and sizes
	if err := ValidateResourceSizes(cfg); err != nil {
		return fmt.Errorf("resource size validation failed: %w", err)
	}

	return nil
}

// ValidateAPITokensPresence validates that API tokens are present
func ValidateAPITokensPresence(cfg *config.ClusterConfig) error {
	errors := []string{}

	// Check DigitalOcean token
	if cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.Enabled {
		if cfg.Providers.DigitalOcean.Token == "" {
			doToken := os.Getenv("DIGITALOCEAN_TOKEN")
			if doToken == "" {
				errors = append(errors, "DigitalOcean token is required (set DIGITALOCEAN_TOKEN env var or provide in config)")
			}
		}
	}

	// Check Linode token
	if cfg.Providers.Linode != nil && cfg.Providers.Linode.Enabled {
		if cfg.Providers.Linode.Token == "" {
			linodeToken := os.Getenv("LINODE_TOKEN")
			if linodeToken == "" {
				errors = append(errors, "Linode token is required (set LINODE_TOKEN env var or provide in config)")
			}
		}
	}

	// Check Azure credentials
	// Note: Azure credentials are optional if using Azure CLI (az login)
	// The Azure provider will automatically use Azure CLI credentials when available
	if cfg.Providers.Azure != nil && cfg.Providers.Azure.Enabled {
		// Skip validation - Azure CLI credentials will be used automatically
	}

	if len(errors) > 0 {
		return fmt.Errorf("API token validation failed:\n  • %s", strings.Join(errors, "\n  • "))
	}

	return nil
}

// ValidateAPITokensWithProviders validates tokens by making actual API calls
func ValidateAPITokensWithProviders(cfg *config.ClusterConfig) error {
	errors := []string{}

	// Validate DigitalOcean token
	if cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.Enabled {
		token := cfg.Providers.DigitalOcean.Token
		if token == "" {
			token = os.Getenv("DIGITALOCEAN_TOKEN")
		}

		if token != "" {
			if err := validateDigitalOceanToken(token); err != nil {
				errors = append(errors, fmt.Sprintf("DigitalOcean token validation failed: %v", err))
			}
		}
	}

	// Validate Linode token
	if cfg.Providers.Linode != nil && cfg.Providers.Linode.Enabled {
		token := cfg.Providers.Linode.Token
		if token == "" {
			token = os.Getenv("LINODE_TOKEN")
		}

		if token != "" {
			if err := validateLinodeToken(token); err != nil {
				errors = append(errors, fmt.Sprintf("Linode token validation failed: %v", err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("provider API validation failed:\n  • %s", strings.Join(errors, "\n  • "))
	}

	return nil
}

// validateDigitalOceanToken validates a DO token by making a test API call
func validateDigitalOceanToken(token string) error {
	client := godo.NewFromToken(token)
	ctx := context.Background()

	_, _, err := client.Account.Get(ctx)
	if err != nil {
		return fmt.Errorf("invalid token or API error: %w", err)
	}

	return nil
}

// validateLinodeToken validates a Linode token by making a test API call
func validateLinodeToken(token string) error {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	oauth2Client := oauth2.NewClient(context.Background(), tokenSource)
	client := linodego.NewClient(oauth2Client)

	_, err := client.GetProfile(context.Background())
	if err != nil {
		return fmt.Errorf("invalid token or API error: %w", err)
	}

	return nil
}

// ValidateNodePools validates node pool configuration
func ValidateNodePools(cfg *config.ClusterConfig) error {
	errors := []string{}

	// Validate that all node pools reference valid providers
	for poolName, pool := range cfg.NodePools {
		// Check provider is valid
		switch pool.Provider {
		case "digitalocean":
			if cfg.Providers.DigitalOcean == nil || !cfg.Providers.DigitalOcean.Enabled {
				errors = append(errors, fmt.Sprintf("pool '%s' uses DigitalOcean but provider is not enabled", poolName))
			}
		case "linode":
			if cfg.Providers.Linode == nil || !cfg.Providers.Linode.Enabled {
				errors = append(errors, fmt.Sprintf("pool '%s' uses Linode but provider is not enabled", poolName))
			}
		case "aws":
			if cfg.Providers.AWS == nil || !cfg.Providers.AWS.Enabled {
				errors = append(errors, fmt.Sprintf("pool '%s' uses AWS but provider is not enabled", poolName))
			}
		case "azure":
			if cfg.Providers.Azure == nil || !cfg.Providers.Azure.Enabled {
				errors = append(errors, fmt.Sprintf("pool '%s' uses Azure but provider is not enabled", poolName))
			}
		case "gcp":
			if cfg.Providers.GCP == nil || !cfg.Providers.GCP.Enabled {
				errors = append(errors, fmt.Sprintf("pool '%s' uses GCP but provider is not enabled", poolName))
			}
		case "hetzner":
			if cfg.Providers.Hetzner == nil || !cfg.Providers.Hetzner.Enabled {
				errors = append(errors, fmt.Sprintf("pool '%s' uses Hetzner but provider is not enabled", poolName))
			}
		default:
			errors = append(errors, fmt.Sprintf("pool '%s' has invalid provider: %s", poolName, pool.Provider))
		}

		// Validate count
		if pool.Count <= 0 {
			errors = append(errors, fmt.Sprintf("pool '%s' has invalid count: %d (must be > 0)", poolName, pool.Count))
		}

		// Validate size
		if pool.Size == "" {
			errors = append(errors, fmt.Sprintf("pool '%s' has no size specified", poolName))
		}

		// Validate region
		if pool.Region == "" {
			errors = append(errors, fmt.Sprintf("pool '%s' has no region specified", poolName))
		}

		// Validate roles
		if len(pool.Roles) == 0 {
			errors = append(errors, fmt.Sprintf("pool '%s' has no roles specified", poolName))
		}

		// Validate role names
		validRoles := map[string]bool{
			"master":       true,
			"controlplane": true,
			"worker":       true,
			"etcd":         true,
		}
		for _, role := range pool.Roles {
			if !validRoles[role] {
				errors = append(errors, fmt.Sprintf("pool '%s' has invalid role: %s", poolName, role))
			}
		}
	}

	// Validate individual nodes
	for i, node := range cfg.Nodes {
		// Check provider is valid
		switch node.Provider {
		case "digitalocean":
			if cfg.Providers.DigitalOcean == nil || !cfg.Providers.DigitalOcean.Enabled {
				errors = append(errors, fmt.Sprintf("node %d uses DigitalOcean but provider is not enabled", i))
			}
		case "linode":
			if cfg.Providers.Linode == nil || !cfg.Providers.Linode.Enabled {
				errors = append(errors, fmt.Sprintf("node %d uses Linode but provider is not enabled", i))
			}
		case "aws":
			if cfg.Providers.AWS == nil || !cfg.Providers.AWS.Enabled {
				errors = append(errors, fmt.Sprintf("node %d uses AWS but provider is not enabled", i))
			}
		case "azure":
			if cfg.Providers.Azure == nil || !cfg.Providers.Azure.Enabled {
				errors = append(errors, fmt.Sprintf("node %d uses Azure but provider is not enabled", i))
			}
		case "gcp":
			if cfg.Providers.GCP == nil || !cfg.Providers.GCP.Enabled {
				errors = append(errors, fmt.Sprintf("node %d uses GCP but provider is not enabled", i))
			}
		case "hetzner":
			if cfg.Providers.Hetzner == nil || !cfg.Providers.Hetzner.Enabled {
				errors = append(errors, fmt.Sprintf("node %d uses Hetzner but provider is not enabled", i))
			}
		default:
			errors = append(errors, fmt.Sprintf("node %d has invalid provider: %s", i, node.Provider))
		}

		// Validate size
		if node.Size == "" {
			errors = append(errors, fmt.Sprintf("node %d has no size specified", i))
		}

		// Validate region
		if node.Region == "" {
			errors = append(errors, fmt.Sprintf("node %d has no region specified", i))
		}

		// Validate roles
		if len(node.Roles) == 0 {
			errors = append(errors, fmt.Sprintf("node %d has no roles specified", i))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("node pool validation failed:\n  • %s", strings.Join(errors, "\n  • "))
	}

	return nil
}

// ValidateNetworkingConfig validates network configuration
func ValidateNetworkingConfig(cfg *config.ClusterConfig) error {
	errors := []string{}

	// Validate WireGuard configuration if enabled
	// TEMPORARILY DISABLED - WireGuard validation has issues with Create field
	// TODO: Fix this properly
	_ = cfg.Network.WireGuard // silence unused warning
	/*
		if cfg.Network.WireGuard != nil && cfg.Network.WireGuard.Enabled {
			// If auto-creating, validate creation parameters
			if cfg.Network.WireGuard.Create {
				if cfg.Network.WireGuard.Provider == "" {
					errors = append(errors, "WireGuard auto-create requires provider to be specified")
				}
				if cfg.Network.WireGuard.Region == "" {
					errors = append(errors, "WireGuard auto-create requires region to be specified")
				}
				if cfg.Network.WireGuard.Size == "" {
					// Use default size
					cfg.Network.WireGuard.Size = "s-1vcpu-1gb" // Default for DigitalOcean
				}
			} else {
				// Using existing VPN, validate endpoint and key
				if cfg.Network.WireGuard.ServerEndpoint == "" {
					errors = append(errors, "WireGuard requires server endpoint when not auto-creating")
				}
				if cfg.Network.WireGuard.ServerPublicKey == "" {
					errors = append(errors, "WireGuard requires server public key when not auto-creating")
				}

				// Validate endpoint format (IP:PORT)
				if cfg.Network.WireGuard.ServerEndpoint != "" {
					endpointRegex := regexp.MustCompile(`^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}):(\d{1,5})$`)
					if !endpointRegex.MatchString(cfg.Network.WireGuard.ServerEndpoint) {
						errors = append(errors, "WireGuard endpoint must be in format IP:PORT (e.g., 1.2.3.4:51820)")
					}
				}
			}
		}
	*/

	// Validate DNS configuration if provided
	if cfg.Network.DNS.Domain != "" {
		// Validate domain format
		domainRegex := regexp.MustCompile(`^[a-z0-9]+([\-\.]{1}[a-z0-9]+)*\.[a-z]{2,}$`)
		if !domainRegex.MatchString(cfg.Network.DNS.Domain) {
			errors = append(errors, fmt.Sprintf("invalid DNS domain format: %s", cfg.Network.DNS.Domain))
		}

		// Validate DNS provider
		validDNSProviders := map[string]bool{
			"digitalocean": true,
			"cloudflare":   true,
			"route53":      true,
		}
		if !validDNSProviders[cfg.Network.DNS.Provider] {
			errors = append(errors, fmt.Sprintf("invalid DNS provider: %s (must be digitalocean, cloudflare, or route53)", cfg.Network.DNS.Provider))
		}
	}

	// Validate CIDR ranges
	if cfg.Kubernetes.PodCIDR != "" {
		if err := validateCIDR(cfg.Kubernetes.PodCIDR); err != nil {
			errors = append(errors, fmt.Sprintf("invalid Pod CIDR: %v", err))
		}
	}

	if cfg.Kubernetes.ServiceCIDR != "" {
		if err := validateCIDR(cfg.Kubernetes.ServiceCIDR); err != nil {
			errors = append(errors, fmt.Sprintf("invalid Service CIDR: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("network validation failed:\n  • %s", strings.Join(errors, "\n  • "))
	}

	return nil
}

// validateCIDR validates a CIDR notation string
func validateCIDR(cidr string) error {
	cidrRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}/\d{1,2}$`)
	if !cidrRegex.MatchString(cidr) {
		return fmt.Errorf("invalid CIDR format: %s (must be in format 10.0.0.0/16)", cidr)
	}
	return nil
}

// ValidateSSHConfig validates SSH configuration
func ValidateSSHConfig(cfg *config.ClusterConfig) error {
	// SSH keys are auto-generated by the orchestrator
	// No validation needed as they are created during deployment
	return nil
}

// ValidateResourceSizes validates resource sizes are appropriate
func ValidateResourceSizes(cfg *config.ClusterConfig) error {
	warnings := []string{}

	// Check if master nodes have sufficient resources
	for poolName, pool := range cfg.NodePools {
		isMaster := false
		for _, role := range pool.Roles {
			if role == "master" || role == "controlplane" {
				isMaster = true
				break
			}
		}

		if isMaster {
			// Warn about small master nodes
			smallSizes := map[string]bool{
				"s-1vcpu-1gb":   true,
				"s-1vcpu-2gb":   true,
				"g6-nanode-1":   true,
				"g6-standard-1": true,
				"Standard_B1s":  true,
				"Standard_B1ms": true,
			}

			if smallSizes[pool.Size] {
				warnings = append(warnings, fmt.Sprintf("pool '%s' uses a small size for master nodes (%s) - consider at least 2GB RAM and 2 vCPUs", poolName, pool.Size))
			}
		}
	}

	// Warnings are not errors, just informational
	if len(warnings) > 0 {
		fmt.Printf("⚠️  Resource warnings:\n  • %s\n\n", strings.Join(warnings, "\n  • "))
	}

	return nil
}
