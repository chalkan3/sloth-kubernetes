package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/internal/validation"
	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/operations"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate cluster configuration file",
	Long: `Validate that your cluster configuration Lisp file is correct and ready for deployment.

This command performs comprehensive validation including:
  • Lisp S-expression syntax and structure
  • Required fields and metadata
  • Node distribution (masters/workers)
  • Provider configuration and credentials
  • Network and WireGuard VPN settings
  • DNS configuration
  • Resource limits and quotas

Use this before 'deploy' to catch configuration errors early.`,
	Example: `  # Validate configuration file
  sloth-kubernetes validate --config cluster.lisp

  # Validate with detailed output
  sloth-kubernetes validate --config production.lisp --verbose

  # Validate and show node distribution
  sloth-kubernetes validate -c staging.lisp`,
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	fmt.Println()
	printHeader("🔍 Validating Cluster Configuration")
	fmt.Println()

	// Use default config file if not specified
	configPath := cfgFile
	if configPath == "" {
		configPath = "./cluster-config.lisp"
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		color.Red("❌ Config file not found: %s", configPath)
		fmt.Println()
		color.Yellow("💡 Specify config file with: --config <path>")
		fmt.Println()
		return fmt.Errorf("config file not found: %s", configPath)
	}

	color.Cyan("📄 Loading configuration: %s", configPath)
	fmt.Println()

	// Load configuration
	cfg, err := config.LoadFromLisp(configPath)
	if err != nil {
		color.Red("❌ Failed to parse Lisp configuration")
		fmt.Println()
		color.Yellow("Error details:")
		fmt.Printf("  %v\n", err)
		fmt.Println()
		color.Yellow("💡 Common issues:")
		fmt.Println("  • Check S-expression syntax (matching parentheses)")
		fmt.Println("  • Ensure all required fields are present")
		fmt.Println("  • Verify quotes around strings")
		fmt.Println()
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	color.Green("✅ Lisp syntax is valid")
	fmt.Println()

	// Validate metadata
	printHeader("📋 Validating Metadata")
	fmt.Println()

	if err := validation.ValidateMetadata(cfg); err != nil {
		color.Red("❌ Metadata validation failed")
		fmt.Printf("  %v\n", err)
		fmt.Println()
		return err
	}

	color.Green("✅ Cluster name: %s", cfg.Metadata.Name)
	if cfg.Metadata.Environment != "" {
		fmt.Printf("  Environment: %s\n", cfg.Metadata.Environment)
	}
	fmt.Println()

	// Validate providers
	printHeader("☁️  Validating Cloud Providers")
	fmt.Println()

	if err := validation.ValidateProviders(cfg); err != nil {
		color.Red("❌ Provider validation failed")
		fmt.Printf("  %v\n", err)
		fmt.Println()
		color.Yellow("💡 Make sure to:")
		fmt.Println("  • Enable at least one provider (DigitalOcean, Linode, Azure)")
		fmt.Println("  • Provide valid API tokens")
		fmt.Println("  • Check token environment variables")
		fmt.Println()
		return err
	}

	// Show enabled providers
	if cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.Enabled {
		color.Green("✅ DigitalOcean: enabled")
		if cfg.Providers.DigitalOcean.Token != "" {
			fmt.Println("  Token: configured ✓")
		}
	}
	if cfg.Providers.Linode != nil && cfg.Providers.Linode.Enabled {
		color.Green("✅ Linode: enabled")
		if cfg.Providers.Linode.Token != "" {
			fmt.Println("  Token: configured ✓")
		}
	}
	if cfg.Providers.Azure != nil && cfg.Providers.Azure.Enabled {
		color.Green("✅ Azure: enabled")
		fmt.Printf("  Location: %s\n", cfg.Providers.Azure.Location)
		fmt.Printf("  Resource Group: %s\n", cfg.Providers.Azure.ResourceGroup)
	}
	fmt.Println()

	// Validate node distribution
	printHeader("🖥️  Validating Node Distribution")
	fmt.Println()

	if err := validation.ValidateNodeDistribution(cfg); err != nil {
		color.Red("❌ Node distribution validation failed")
		fmt.Printf("  %v\n", err)
		fmt.Println()
		color.Yellow("💡 Requirements:")
		fmt.Println("  • At least 1 master node")
		fmt.Println("  • Master nodes must be odd number for HA (1, 3, 5, ...)")
		fmt.Println("  • At least 1 node in total")
		fmt.Println()
		return err
	}

	// Show node distribution
	dist := validation.CalculateDistribution(cfg)
	color.Green("✅ Node distribution is valid")
	fmt.Println()
	fmt.Printf("  Total Nodes: %d\n", dist.Total)
	fmt.Printf("  - Masters: %d\n", dist.Masters)
	fmt.Printf("  - Workers: %d\n", dist.Workers)
	fmt.Println()

	if len(dist.ByProvider) > 0 {
		color.Cyan("  By Provider:")
		for provider, count := range dist.ByProvider {
			fmt.Printf("    • %s: %d nodes\n", provider, count)
		}
		fmt.Println()
	}

	// Validate network configuration
	printHeader("🌐 Validating Network Configuration")
	fmt.Println()

	// Validate VPN (WireGuard or Tailscale)
	if err := validation.ValidateVPNConfig(cfg); err != nil {
		color.Red("❌ VPN validation failed")
		fmt.Printf("  %v\n", err)
		fmt.Println()
		color.Yellow("💡 VPN (WireGuard or Tailscale) is required for:")
		fmt.Println("  • Private cluster networking")
		fmt.Println("  • Secure node-to-node communication")
		fmt.Println("  • Cross-provider mesh networking")
		fmt.Println()
		return err
	}

	if cfg.Network.WireGuard != nil && cfg.Network.WireGuard.Enabled {
		color.Green("✅ WireGuard VPN: enabled")
		if cfg.Network.WireGuard.Create {
			fmt.Println("  Mode: Auto-create VPN server")
			if cfg.Network.WireGuard.Provider != "" {
				fmt.Printf("  Provider: %s\n", cfg.Network.WireGuard.Provider)
			}
			if cfg.Network.WireGuard.Region != "" {
				fmt.Printf("  Region: %s\n", cfg.Network.WireGuard.Region)
			}
		} else {
			fmt.Println("  Mode: Using existing VPN server")
			if cfg.Network.WireGuard.ServerEndpoint != "" {
				fmt.Printf("  Endpoint: %s\n", cfg.Network.WireGuard.ServerEndpoint)
			}
		}
	}

	if cfg.Network.Tailscale != nil && cfg.Network.Tailscale.Enabled {
		color.Green("✅ Tailscale VPN: enabled")
		if cfg.Network.Tailscale.Create {
			fmt.Println("  Mode: Auto-create Headscale server")
			if cfg.Network.Tailscale.Provider != "" {
				fmt.Printf("  Provider: %s\n", cfg.Network.Tailscale.Provider)
			}
			if cfg.Network.Tailscale.Region != "" {
				fmt.Printf("  Region: %s\n", cfg.Network.Tailscale.Region)
			}
			if cfg.Network.Tailscale.Namespace != "" {
				fmt.Printf("  Namespace: %s\n", cfg.Network.Tailscale.Namespace)
			}
		} else {
			fmt.Println("  Mode: Using existing Headscale server")
			if cfg.Network.Tailscale.HeadscaleURL != "" {
				fmt.Printf("  Headscale URL: %s\n", cfg.Network.Tailscale.HeadscaleURL)
			}
		}
	}
	fmt.Println()

	// Validate DNS if configured
	if cfg.Network.DNS.Domain != "" {
		if err := validation.ValidateDNSConfig(cfg); err != nil {
			color.Yellow("⚠️  DNS validation warning")
			fmt.Printf("  %v\n", err)
			fmt.Println()
		} else {
			color.Green("✅ DNS configuration: valid")
			fmt.Printf("  Domain: %s\n", cfg.Network.DNS.Domain)
			fmt.Printf("  Provider: %s\n", cfg.Network.DNS.Provider)
			fmt.Println()
		}
	}

	// Validate Kubernetes version
	printHeader("☸️  Validating Kubernetes Configuration")
	fmt.Println()

	if cfg.Kubernetes.Version != "" {
		color.Green("✅ Kubernetes version: %s", cfg.Kubernetes.Version)
	} else {
		color.Yellow("⚠️  No Kubernetes version specified (will use default)")
	}
	fmt.Println()

	// Overall validation
	printHeader("✨ Overall Validation")
	fmt.Println()

	if err := validation.ValidateClusterConfig(cfg); err != nil {
		color.Red("❌ Configuration validation failed")
		fmt.Printf("  %v\n", err)
		fmt.Println()
		return err
	}

	color.Green("✅ Configuration is valid and ready for deployment!")
	fmt.Println()

	// Show next steps
	color.Cyan("📋 Next Steps:")
	fmt.Println()
	fmt.Printf("  1. Deploy cluster:\n")
	color.Cyan("     sloth-kubernetes deploy --config %s\n", configPath)
	fmt.Println()
	fmt.Printf("  2. Or preview changes first:\n")
	color.Cyan("     sloth-kubernetes pulumi preview --stack <name>\n")
	fmt.Println()

	// Show warnings if any
	warnings := collectWarnings(cfg)
	if len(warnings) > 0 {
		color.Yellow("⚠️  Warnings:")
		fmt.Println()
		for _, warning := range warnings {
			fmt.Printf("  • %s\n", warning)
		}
		fmt.Println()
	}

	// Record the validation (6 checks: syntax, metadata, providers, nodes, network, kubernetes)
	// Note: validate command doesn't require a stack, but we try to use it if available for recording
	totalChecks := 6
	passedChecks := totalChecks
	warningChecks := len(warnings)
	if stackName != "" {
		operations.RecordValidation(stackName, "config", "passed", totalChecks, passedChecks, 0, warningChecks, time.Since(startTime), nil)
	}

	return nil
}

// collectWarnings collects non-critical warnings about the configuration
func collectWarnings(cfg *config.ClusterConfig) []string {
	var warnings []string

	// Check if only one master (no HA)
	dist := validation.CalculateDistribution(cfg)
	if dist.Masters == 1 {
		warnings = append(warnings, "Single master node - no high availability")
	}

	// Check if no workers
	if dist.Workers == 0 {
		warnings = append(warnings, "No dedicated worker nodes - masters will run workloads")
	}

	// Check if DNS is not configured
	if cfg.Network.DNS.Domain == "" {
		warnings = append(warnings, "DNS not configured - nodes will use IP addresses")
	}

	// Check if using single provider
	enabledProviders := 0
	if cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.Enabled {
		enabledProviders++
	}
	if cfg.Providers.Linode != nil && cfg.Providers.Linode.Enabled {
		enabledProviders++
	}
	if cfg.Providers.Azure != nil && cfg.Providers.Azure.Enabled {
		enabledProviders++
	}
	if enabledProviders == 1 {
		warnings = append(warnings, "Single cloud provider - consider multi-cloud for redundancy")
	}

	return warnings
}
