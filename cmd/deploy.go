package cmd

import (
	"context"
	"fmt"
	"os"

	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/spf13/cobra"

	"sloth-kubernetes/internal/orchestrator"
	"sloth-kubernetes/internal/validation"
	"sloth-kubernetes/pkg/config"
	"sloth-kubernetes/pkg/vpc"
	"sloth-kubernetes/pkg/vpn"
)

var (
	doToken           string
	linodeToken       string
	wireguardEndpoint string
	wireguardPubKey   string
	dryRun            bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a new Kubernetes cluster",
	Long: `Deploy a multi-cloud Kubernetes cluster with:
  • 6 nodes across DigitalOcean and Linode
  • RKE2 Kubernetes distribution
  • WireGuard VPN mesh for private networking
  • Automated DNS configuration
  • High availability setup (3 masters + 3 workers)`,
	Example: `  # Deploy using config file
  kubernetes-create deploy --config production.yaml

  # Deploy with inline credentials
  kubernetes-create deploy \
    --do-token xxx \
    --linode-token yyy \
    --wireguard-endpoint 1.2.3.4:51820 \
    --wireguard-pubkey "xxx="

  # Preview without applying
  kubernetes-create deploy --dry-run`,
	RunE: runDeploy,
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVar(&doToken, "do-token", "", "DigitalOcean API token")
	deployCmd.Flags().StringVar(&linodeToken, "linode-token", "", "Linode API token")
	deployCmd.Flags().StringVar(&wireguardEndpoint, "wireguard-endpoint", "", "WireGuard server endpoint (e.g., 1.2.3.4:51820)")
	deployCmd.Flags().StringVar(&wireguardPubKey, "wireguard-pubkey", "", "WireGuard server public key")
	deployCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Print header
	printHeader("🚀 Kubernetes Multi-Cloud Deployment")

	// Load configuration
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Loading configuration..."
	s.Start()

	cfg, err := loadConfiguration()
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	s.Stop()
	printSuccess("Configuration loaded")

	// Validate configuration
	s.Suffix = " Validating configuration..."
	s.Start()
	if err := validation.ValidateClusterConfig(cfg); err != nil {
		s.Stop()
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	s.Stop()
	printSuccess("Configuration validated")

	// Print summary
	printDeploymentSummary(cfg)

	// Confirm deployment
	if !autoApprove && !dryRun {
		if !confirm("Do you want to proceed with deployment?") {
			color.Yellow("Deployment cancelled")
			return nil
		}
	}

	// Create Pulumi program
	program := func(ctx *pulumi.Context) error {
		// Phase 1: Create VPCs if configured
		ctx.Log.Info("📊 Phase 1: VPC Creation", nil)
		vpcManager := vpc.NewVPCManager(ctx)
		vpcs, err := vpcManager.CreateAllVPCs(&cfg.Providers)
		if err != nil {
			return fmt.Errorf("failed to create VPCs: %w", err)
		}

		if len(vpcs) > 0 {
			ctx.Log.Info(fmt.Sprintf("✅ Created %d VPC(s)", len(vpcs)), nil)
		}

		// Phase 2: Create WireGuard VPN server if configured
		var wgResult *vpn.WireGuardResult
		if cfg.Network.WireGuard != nil && cfg.Network.WireGuard.Create {
			ctx.Log.Info("📊 Phase 2: WireGuard VPN Server Creation", nil)

			// Generate SSH key for VPN server
			sshKeyOutput := pulumi.String("dummy-key").ToStringOutput() // Will be replaced by actual key

			wgManager := vpn.NewWireGuardManager(ctx)
			wgResult, err = wgManager.CreateWireGuardServer(cfg.Network.WireGuard, sshKeyOutput)
			if err != nil {
				return fmt.Errorf("failed to create WireGuard server: %w", err)
			}

			if wgResult != nil {
				ctx.Log.Info("✅ WireGuard VPN server created", nil)

				// Update config with VPN server info (will be resolved by Pulumi)
				// The actual IP will be available in outputs after deployment
			}
		}

		// Phase 3: Create cluster orchestrator
		ctx.Log.Info("📊 Phase 3: Kubernetes Cluster Creation", nil)
		clusterOrch, err := orchestrator.NewSimpleRealOrchestratorComponent(ctx, "kubernetes-cluster", cfg)
		if err != nil {
			return fmt.Errorf("failed to create orchestrator: %w", err)
		}

		// Export outputs
		ctx.Export("clusterName", clusterOrch.ClusterName)
		ctx.Export("kubeConfig", clusterOrch.KubeConfig)
		ctx.Export("sshPrivateKey", clusterOrch.SSHPrivateKey)
		ctx.Export("apiEndpoint", clusterOrch.APIEndpoint)

		// Export VPC information
		for provider, vpcResult := range vpcs {
			ctx.Export(fmt.Sprintf("vpc_%s_id", provider), vpcResult.ID)
			ctx.Export(fmt.Sprintf("vpc_%s_cidr", provider), pulumi.String(vpcResult.CIDR))
		}

		// Export VPN information
		if wgResult != nil {
			ctx.Export("vpn_server_id", wgResult.ServerID)
			ctx.Export("vpn_server_ip", wgResult.ServerIP)
			ctx.Export("vpn_port", pulumi.Int(wgResult.Port))
			ctx.Export("vpn_subnet", pulumi.String(wgResult.SubnetCIDR))
		}

		ctx.Log.Info("✅ All phases completed successfully!", nil)

		return nil
	}

	// Setup Pulumi Automation API stack
	fmt.Println()
	printInfo("🔧 Setting up Pulumi stack...")

	stack, err := auto.UpsertStackInlineSource(ctx, stackName, "kubernetes-create", program)
	if err != nil {
		return fmt.Errorf("failed to create or select stack: %w", err)
	}

	// Set configuration
	if err := setStackConfig(ctx, stack, cfg); err != nil {
		return fmt.Errorf("failed to set stack config: %w", err)
	}

	printSuccess("Pulumi stack configured")

	// Refresh stack
	fmt.Println()
	printInfo("🔄 Refreshing stack state...")
	_, err = stack.Refresh(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh stack: %w", err)
	}

	if dryRun {
		// Preview mode
		fmt.Println()
		printInfo("📋 Previewing changes (dry-run mode)...")

		prev, err := stack.Preview(ctx)
		if err != nil {
			return fmt.Errorf("failed to preview: %w", err)
		}

		printPreviewSummary(prev)
		return nil
	}

	// Deploy!
	fmt.Println()
	printHeader("🚀 Deploying cluster...")
	fmt.Println()

	// Setup progress streams
	stdoutStreamer := optup.ProgressStreams(os.Stdout)

	res, err := stack.Up(ctx, stdoutStreamer)
	if err != nil {
		return fmt.Errorf("failed to deploy: %w", err)
	}

	// Print success
	fmt.Println()
	printSuccess("✅ Cluster deployed successfully!")
	fmt.Println()

	// Print outputs
	printClusterOutputs(res.Outputs)

	return nil
}

func loadConfiguration() (*config.ClusterConfig, error) {
	var cfg *config.ClusterConfig
	var err error

	// Try to load from config file first
	if cfgFile != "" {
		cfg, err = config.LoadFromYAML(cfgFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	} else {
		// Use default configuration with flag overrides
		cfg = &config.ClusterConfig{
			Metadata: config.Metadata{
				Name: "production",
			},
			Providers: config.ProvidersConfig{
				DigitalOcean: &config.DigitalOceanProvider{
					Enabled: true,
					Token:   getEnvOrFlag("DIGITALOCEAN_TOKEN", doToken),
					Region:  "nyc3",
				},
				Linode: &config.LinodeProvider{
					Enabled:      true,
					Token:        getEnvOrFlag("LINODE_TOKEN", linodeToken),
					Region:       "us-east",
					RootPassword: "SecureLinodeRootPass2025!",
				},
			},
			Network: config.NetworkConfig{
				DNS: config.DNSConfig{
					Domain:   "chalkan3.com.br",
					Provider: "digitalocean",
				},
				WireGuard: &config.WireGuardConfig{
					Enabled:         true,
					ServerEndpoint:  getEnvOrFlag("WIREGUARD_ENDPOINT", wireguardEndpoint),
					ServerPublicKey: getEnvOrFlag("WIREGUARD_PUBKEY", wireguardPubKey),
				},
			},
			Kubernetes: config.KubernetesConfig{
				Distribution:  "rke2",
				Version:       "v1.28.5+rke2r1",
				NetworkPlugin: "calico",
				PodCIDR:       "10.42.0.0/16",
				ServiceCIDR:   "10.43.0.0/16",
				ClusterDNS:    "10.43.0.10",
				ClusterDomain: "cluster.local",
				RKE2:          config.GetRKE2Defaults(),
			},
			NodePools: map[string]config.NodePool{
				"do-masters": {
					Name:     "do-masters",
					Count:    1,
					Size:     "s-2vcpu-4gb",
					Image:    "ubuntu-22-04-x64",
					Region:   "nyc3",
					Provider: "digitalocean",
					Roles:    []string{"master"},
				},
				"do-workers": {
					Name:     "do-workers",
					Count:    2,
					Size:     "s-2vcpu-4gb",
					Image:    "ubuntu-22-04-x64",
					Region:   "nyc3",
					Provider: "digitalocean",
					Roles:    []string{"worker"},
				},
				"linode-masters": {
					Name:     "linode-masters",
					Count:    2,
					Size:     "g6-standard-2",
					Image:    "linode/ubuntu22.04",
					Region:   "us-east",
					Provider: "linode",
					Roles:    []string{"master"},
				},
				"linode-workers": {
					Name:     "linode-workers",
					Count:    1,
					Size:     "g6-standard-2",
					Image:    "linode/ubuntu22.04",
					Region:   "us-east",
					Provider: "linode",
					Roles:    []string{"worker"},
				},
			},
		}
	}

	// Override with flags if provided (flags take precedence over config file)
	if doToken != "" {
		if cfg.Providers.DigitalOcean == nil {
			cfg.Providers.DigitalOcean = &config.DigitalOceanProvider{}
		}
		cfg.Providers.DigitalOcean.Token = doToken
	}
	if linodeToken != "" {
		if cfg.Providers.Linode == nil {
			cfg.Providers.Linode = &config.LinodeProvider{}
		}
		cfg.Providers.Linode.Token = linodeToken
	}
	if wireguardEndpoint != "" {
		if cfg.Network.WireGuard == nil {
			cfg.Network.WireGuard = &config.WireGuardConfig{}
		}
		cfg.Network.WireGuard.ServerEndpoint = wireguardEndpoint
	}
	if wireguardPubKey != "" {
		if cfg.Network.WireGuard == nil {
			cfg.Network.WireGuard = &config.WireGuardConfig{}
		}
		cfg.Network.WireGuard.ServerPublicKey = wireguardPubKey
	}

	return cfg, nil
}

func setStackConfig(ctx context.Context, stack auto.Stack, cfg *config.ClusterConfig) error {
	// Set configuration values for Pulumi
	configs := map[string]auto.ConfigValue{
		"digitaloceanToken": {Value: cfg.Providers.DigitalOcean.Token, Secret: true},
		"linodeToken":       {Value: cfg.Providers.Linode.Token, Secret: true},
		"wireguardServerEndpoint":  {Value: cfg.Network.WireGuard.ServerEndpoint},
		"wireguardServerPublicKey": {Value: cfg.Network.WireGuard.ServerPublicKey},
	}

	return stack.SetAllConfig(ctx, configs)
}

func getEnvOrFlag(envKey, flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envKey)
}

func printDeploymentSummary(cfg *config.ClusterConfig) {
	fmt.Println()
	color.Cyan("📋 Deployment Summary:")
	fmt.Printf("  • Cluster Name: %s\n", cfg.Metadata.Name)

	// VPC Information
	fmt.Println()
	color.Cyan("🌐 Network Infrastructure:")
	vpcCount := 0
	if cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.VPC != nil && cfg.Providers.DigitalOcean.VPC.Create {
		fmt.Printf("  • DigitalOcean VPC: %s (%s)\n", cfg.Providers.DigitalOcean.VPC.Name, cfg.Providers.DigitalOcean.VPC.CIDR)
		vpcCount++
	}
	if cfg.Providers.Linode != nil && cfg.Providers.Linode.VPC != nil && cfg.Providers.Linode.VPC.Create {
		fmt.Printf("  • Linode VPC: %s (%s)\n", cfg.Providers.Linode.VPC.Name, cfg.Providers.Linode.VPC.CIDR)
		vpcCount++
	}
	if vpcCount == 0 {
		fmt.Printf("  • VPCs: Using existing networks\n")
	}

	// VPN Information
	if cfg.Network.WireGuard != nil && cfg.Network.WireGuard.Create {
		fmt.Printf("  • WireGuard VPN: Auto-create on %s (%s)\n", cfg.Network.WireGuard.Provider, cfg.Network.WireGuard.SubnetCIDR)
		fmt.Printf("    → Port: %d\n", cfg.Network.WireGuard.Port)
		fmt.Printf("    → Mesh Networking: %v\n", cfg.Network.WireGuard.MeshNetworking)
	} else if cfg.Network.WireGuard != nil && cfg.Network.WireGuard.Enabled {
		fmt.Printf("  • WireGuard VPN: Using existing server (%s)\n", cfg.Network.WireGuard.ServerEndpoint)
	}

	// Node Information
	fmt.Println()
	color.Cyan("🖥️  Cluster Nodes:")
	totalNodes := 0
	masters := 0
	workers := 0
	for _, pool := range cfg.NodePools {
		totalNodes += pool.Count
		for _, role := range pool.Roles {
			if role == "master" {
				masters += pool.Count
			} else if role == "worker" {
				workers += pool.Count
			}
		}
	}

	fmt.Printf("  • Total Nodes: %d (%d masters + %d workers)\n", totalNodes, masters, workers)

	// Providers
	providers := []string{}
	if cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.Enabled {
		providers = append(providers, "DigitalOcean")
	}
	if cfg.Providers.Linode != nil && cfg.Providers.Linode.Enabled {
		providers = append(providers, "Linode")
	}
	if len(providers) > 0 {
		fmt.Printf("  • Providers: %s\n", joinStrings(providers, " + "))
	}

	fmt.Printf("  • Kubernetes: RKE2 %s\n", cfg.Kubernetes.Version)
	fmt.Println()

	// Deployment phases
	color.Cyan("📊 Deployment Phases:")
	phaseNum := 1
	if vpcCount > 0 {
		fmt.Printf("  %d. Create VPCs (%d)\n", phaseNum, vpcCount)
		phaseNum++
	}
	if cfg.Network.WireGuard != nil && cfg.Network.WireGuard.Create {
		fmt.Printf("  %d. Create WireGuard VPN server\n", phaseNum)
		phaseNum++
	}
	fmt.Printf("  %d. Provision %d nodes\n", phaseNum, totalNodes)
	phaseNum++
	if cfg.Network.WireGuard != nil && cfg.Network.WireGuard.Enabled {
		fmt.Printf("  %d. Configure VPN mesh networking\n", phaseNum)
		phaseNum++
	}
	fmt.Printf("  %d. Install Kubernetes\n", phaseNum)
	fmt.Println()
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

func printClusterOutputs(outputs auto.OutputMap) {
	// VPC Information
	hasVPC := false
	for key := range outputs {
		if len(key) > 4 && key[:4] == "vpc_" {
			if !hasVPC {
				color.Cyan("🌐 VPC Information:")
				hasVPC = true
			}
			if key[len(key)-3:] == "_id" {
				provider := key[4 : len(key)-3]
				if id, ok := outputs[key]; ok {
					cidr := ""
					if cidrVal, ok := outputs[fmt.Sprintf("vpc_%s_cidr", provider)]; ok {
						cidr = fmt.Sprintf(" (%v)", cidrVal.Value)
					}
					fmt.Printf("  • %s VPC: %v%s\n", provider, id.Value, cidr)
				}
			}
		}
	}
	if hasVPC {
		fmt.Println()
	}

	// VPN Information
	if vpnIP, ok := outputs["vpn_server_ip"]; ok {
		color.Cyan("🔐 VPN Information:")
		fmt.Printf("  • Server IP: %v\n", vpnIP.Value)
		if port, ok := outputs["vpn_port"]; ok {
			fmt.Printf("  • Port: %v\n", port.Value)
		}
		if subnet, ok := outputs["vpn_subnet"]; ok {
			fmt.Printf("  • Subnet: %v\n", subnet.Value)
		}
		fmt.Println()
	}

	// Cluster Information
	color.Cyan("📊 Cluster Information:")
	if name, ok := outputs["clusterName"]; ok {
		fmt.Printf("  • Name: %v\n", name.Value)
	}

	if endpoint, ok := outputs["apiEndpoint"]; ok {
		fmt.Printf("  • API Endpoint: %v\n", endpoint.Value)
	}

	fmt.Println()
	color.Green("🎯 Next Steps:")
	fmt.Println("  1. Get kubeconfig: kubernetes-create kubeconfig -o ~/.kube/config")
	fmt.Println("  2. Check status: kubernetes-create status")
	fmt.Println("  3. List nodes: kubectl get nodes")
	fmt.Println("  4. Bootstrap addons: kubernetes-create addons bootstrap --repo <gitops-repo>")
}

func printPreviewSummary(prev auto.PreviewResult) {
	fmt.Println()
	color.Cyan("📋 Preview Summary (Dry-Run Mode)")
	fmt.Println()

	// Count changes
	creates := prev.ChangeSummary["create"]
	updates := prev.ChangeSummary["update"]
	deletes := prev.ChangeSummary["delete"]
	same := prev.ChangeSummary["same"]

	// Print summary
	color.Green("Resources to be created: %d", creates)
	if creates > 0 {
		fmt.Println("  → New resources will be provisioned")
	}

	color.Yellow("Resources to be updated: %d", updates)
	if updates > 0 {
		fmt.Println("  → Existing resources will be modified")
	}

	color.Red("Resources to be deleted: %d", deletes)
	if deletes > 0 {
		fmt.Println("  → Resources will be destroyed")
	}

	color.Blue("Resources unchanged: %d", same)

	fmt.Println()

	// Print what would happen
	fmt.Println()
	color.Cyan("💡 What will happen when you run without --dry-run:")
	fmt.Println()

	if creates > 0 {
		fmt.Println("  1. SSH keys will be generated")
		fmt.Println("  2. Droplets/Linodes will be created across providers")
		fmt.Println("  3. WireGuard VPN mesh will be configured")
		fmt.Println("  4. RKE2 Kubernetes will be installed and configured")
		fmt.Println("  5. DNS records will be created")
		fmt.Println("  6. Kubeconfig will be generated and available")
	} else if updates > 0 {
		fmt.Println("  1. Existing resources will be updated in-place where possible")
		fmt.Println("  2. Some resources may need to be replaced (destroy + recreate)")
		fmt.Println("  3. Cluster may experience brief downtime during updates")
	}

	fmt.Println()
	color.Yellow("⚠️  This was a DRY-RUN. No actual changes were made.")
	fmt.Println()
	color.Green("To apply these changes, run without --dry-run flag:")
	fmt.Printf("  kubernetes-create deploy --config <your-config>.yaml\n")
	fmt.Println()
}

func printHeader(text string) {
	fmt.Println()
	color.New(color.Bold, color.FgCyan).Println(text)
	fmt.Println()
}

func printSuccess(text string) {
	color.Green("✓ " + text)
}

func printInfo(text string) {
	color.Cyan(text)
}

func confirm(question string) bool {
	fmt.Printf("\n%s (y/N): ", color.YellowString("❓ "+question))
	var response string
	fmt.Scanln(&response)
	return response == "y" || response == "Y" || response == "yes"
}
