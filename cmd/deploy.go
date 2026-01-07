package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/internal/common"
	"github.com/chalkan3/sloth-kubernetes/internal/orchestrator"
	"github.com/chalkan3/sloth-kubernetes/internal/validation"
	"github.com/chalkan3/sloth-kubernetes/pkg/addons"
	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	"github.com/chalkan3/sloth-kubernetes/pkg/vpc"
)

var (
	doToken           string
	linodeToken       string
	wireguardEndpoint string
	wireguardPubKey   string
	dryRun            bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy [stack-name]",
	Short: "Deploy a new Kubernetes cluster",
	Long: `Deploy a multi-cloud Kubernetes cluster with:
  • 6 nodes across DigitalOcean and Linode
  • RKE2 Kubernetes distribution
  • WireGuard VPN mesh for private networking
  • Automated DNS configuration
  • High availability setup (3 masters + 3 workers)

Stack-based deployment allows you to manage multiple clusters independently.
Each stack maintains its own state file, enabling cluster updates and parallel deployments.`,
	Example: `  # Deploy a cluster with stack name
  sloth-kubernetes deploy my-cluster --config production.lisp

  # Deploy production and staging clusters
  sloth-kubernetes deploy production --config prod.lisp
  sloth-kubernetes deploy staging --config staging.lisp

  # Update an existing cluster
  sloth-kubernetes deploy production --config prod.lisp

  # Preview without applying
  sloth-kubernetes deploy my-cluster --config test.lisp --dry-run`,
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

	// IMPORTANT: Load saved S3 backend configuration FIRST,
	// before any Pulumi API calls that might initialize AWS SDK
	_ = common.LoadSavedConfig()

	// Parse stack name from args (first positional argument)
	if len(args) > 0 {
		stackName = args[0]
		printInfo(fmt.Sprintf("📦 Using stack: %s", stackName))
	} else {
		// Use default stack name if not provided
		if stackName == "" {
			stackName = "production"
		}
		printInfo(fmt.Sprintf("📦 Using default stack: %s", stackName))
	}

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

	// Comprehensive validation before deployment
	fmt.Println()
	printHeader("🔍 Pre-Deployment Validation")
	fmt.Println()

	// Step 1: Basic configuration validation
	s.Suffix = " Validating configuration structure..."
	s.Start()
	if err := validation.ValidateClusterConfig(cfg); err != nil {
		s.Stop()
		color.Red("❌ Configuration validation failed")
		fmt.Println()
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	s.Stop()
	color.Green("✅ Configuration structure is valid")

	// Step 2: Validate API tokens presence
	s.Suffix = " Validating API tokens..."
	s.Start()
	if err := validation.ValidateAPITokensPresence(cfg); err != nil {
		s.Stop()
		color.Red("❌ API token validation failed")
		fmt.Println()
		return fmt.Errorf("API token validation failed: %w", err)
	}
	s.Stop()
	color.Green("✅ API tokens are configured")

	// Step 3: Validate with actual provider APIs (optional but recommended)
	if !dryRun {
		s.Suffix = " Validating API tokens with cloud providers..."
		s.Start()
		if err := validation.ValidateAPITokensWithProviders(cfg); err != nil {
			s.Stop()
			color.Yellow("⚠️  Warning: API token verification failed")
			color.Yellow("   %v", err)
			fmt.Println()
			if !autoApprove {
				if !confirm("Continue anyway?") {
					return fmt.Errorf("deployment cancelled due to API validation failure")
				}
			}
		} else {
			s.Stop()
			color.Green("✅ API tokens verified with providers")
		}
	}

	// Step 4: Validate node pools
	s.Suffix = " Validating node pools..."
	s.Start()
	if err := validation.ValidateNodePools(cfg); err != nil {
		s.Stop()
		color.Red("❌ Node pool validation failed")
		fmt.Println()
		return fmt.Errorf("node pool validation failed: %w", err)
	}
	s.Stop()
	color.Green("✅ Node pools are valid")

	// Step 5: Validate networking
	s.Suffix = " Validating network configuration..."
	s.Start()
	if err := validation.ValidateNetworkingConfig(cfg); err != nil {
		s.Stop()
		color.Red("❌ Network validation failed")
		fmt.Println()
		return fmt.Errorf("network validation failed: %w", err)
	}
	s.Stop()
	color.Green("✅ Network configuration is valid")

	// Step 6: Validate SSH configuration
	s.Suffix = " Validating SSH configuration..."
	s.Start()
	if err := validation.ValidateSSHConfig(cfg); err != nil {
		s.Stop()
		color.Red("❌ SSH configuration validation failed")
		fmt.Println()
		return fmt.Errorf("SSH configuration validation failed: %w", err)
	}
	s.Stop()
	color.Green("✅ SSH configuration is valid")

	// Step 7: Validate resource sizes
	s.Suffix = " Validating resource sizes..."
	s.Start()
	if err := validation.ValidateResourceSizes(cfg); err != nil {
		s.Stop()
		color.Red("❌ Resource validation failed")
		fmt.Println()
		return fmt.Errorf("resource validation failed: %w", err)
	}
	s.Stop()
	color.Green("✅ Resource sizes validated")

	fmt.Println()
	color.Green("✅ All pre-deployment validations passed!")
	fmt.Println()

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

		// Phase 2: Create cluster orchestrator FIRST (to generate SSH keys)
		ctx.Log.Info("📊 Phase 2: WireGuard VPN Server Creation", nil)
		ctx.Log.Info("📊 Phase 3: Kubernetes Cluster Creation", nil)
		clusterOrch, err := orchestrator.NewSimpleRealOrchestratorComponent(ctx, "kubernetes-cluster", cfg, lispManifestContent, previousDeploymentMeta)
		if err != nil {
			return fmt.Errorf("failed to create orchestrator: %w", err)
		}

		// Export outputs (all encrypted with passphrase)
		secretExporter := secrets.NewSecretExporter(ctx)
		secretExporter.Export("clusterName", clusterOrch.ClusterName)
		secretExporter.Export("kubeConfig", clusterOrch.KubeConfig)
		secretExporter.Export("sshPrivateKey", clusterOrch.SSHPrivateKey)
		secretExporter.Export("apiEndpoint", clusterOrch.APIEndpoint)

		// Export VPC information (encrypted)
		for provider, vpcResult := range vpcs {
			secretExporter.Export(fmt.Sprintf("vpc_%s_id", provider), vpcResult.ID)
			secretExporter.ExportString(fmt.Sprintf("vpc_%s_cidr", provider), vpcResult.CIDR)
		}

		ctx.Log.Info("✅ All phases completed successfully!", nil)

		return nil
	}

	// Setup Pulumi Automation API stack
	fmt.Println()
	printInfo("🔧 Setting up Pulumi stack...")

	// Create workspace with backend URL from environment
	// Note: LoadSavedConfig() already set all environment variables at line 74
	// For S3 backend, we need to set the project name
	projectName := "sloth-kubernetes"
	workspaceOpts := []auto.LocalWorkspaceOption{
		auto.Program(program),
		auto.Project(workspace.Project{
			Name:    tokens.PackageName(projectName),
			Runtime: workspace.NewProjectRuntimeInfo("go", nil),
		}),
	}

	// Collect all AWS/S3 environment variables to pass to Pulumi subprocess
	envVars := make(map[string]string)
	awsEnvKeys := []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"AWS_REGION",
		"AWS_S3_ENDPOINT",
		"AWS_S3_USE_PATH_STYLE",
		"AWS_S3_FORCE_PATH_STYLE",
		"PULUMI_BACKEND_URL",
		"PULUMI_CONFIG_PASSPHRASE",
	}
	for _, key := range awsEnvKeys {
		if val := os.Getenv(key); val != "" {
			envVars[key] = val
		}
	}

	// Add environment variables to workspace options
	// This ensures Pulumi subprocess gets the S3 credentials
	if len(envVars) > 0 {
		workspaceOpts = append(workspaceOpts, auto.EnvVars(envVars))
	}

	// If PULUMI_BACKEND_URL is set, use it
	if backendURL := os.Getenv("PULUMI_BACKEND_URL"); backendURL != "" {
		workspaceOpts = append(workspaceOpts, auto.SecretsProvider("passphrase"))
		// Set PULUMI_CONFIG_PASSPHRASE if not set
		if os.Getenv("PULUMI_CONFIG_PASSPHRASE") == "" {
			os.Setenv("PULUMI_CONFIG_PASSPHRASE", "")
			envVars["PULUMI_CONFIG_PASSPHRASE"] = ""
		}
	}

	ws, err := auto.NewLocalWorkspace(ctx, workspaceOpts...)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// For S3 backend, we need to use fully qualified stack name: organization/project/stack
	// We use "organization" as the organization name (self-managed backend doesn't need real org)
	fullyQualifiedStackName := fmt.Sprintf("organization/%s/%s", projectName, stackName)

	stack, err := auto.UpsertStack(ctx, fullyQualifiedStackName, ws)
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

	// Fetch previous deployment metadata for scale tracking
	outputs, err := stack.Outputs(ctx)
	if err == nil {
		if metaOutput, ok := outputs["deploymentMeta"]; ok && metaOutput.Value != nil {
			if metaStr, ok := metaOutput.Value.(string); ok {
				previousDeploymentMeta = metaStr
				printInfo("📊 Found previous deployment metadata (tracking scale operations)")
			}
		}
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

// lispManifestContent stores the raw Lisp file content for Pulumi state storage
var lispManifestContent string

// previousDeploymentMeta stores the previous deployment metadata for scale tracking
var previousDeploymentMeta string

func loadConfiguration() (*config.ClusterConfig, error) {
	var cfg *config.ClusterConfig
	var err error

	// Try to load from config file first
	if cfgFile != "" {
		fmt.Printf("🔍 DEBUG [cmd/deploy.go]: Loading config from file: %s\n", cfgFile)

		// Read raw Lisp content for storage in Pulumi state
		rawContent, readErr := os.ReadFile(cfgFile)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read config file: %w", readErr)
		}
		lispManifestContent = string(rawContent)

		cfg, err = config.LoadFromLisp(cfgFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}

		// DEBUG: Log pools immediately after loading
		fmt.Printf("🔍 DEBUG [cmd/deploy.go]: Loaded %d node pools from Lisp\n", len(cfg.NodePools))
		for poolName, pool := range cfg.NodePools {
			fmt.Printf("🔍 DEBUG [cmd/deploy.go]: Pool '%s' - provider=%s, count=%d\n", poolName, pool.Provider, pool.Count)
		}

		// DEBUG: Check bastion configuration
		if cfg.Security.Bastion == nil {
			fmt.Printf("🔍 DEBUG [cmd/deploy.go]: cfg.Security.Bastion is NIL after LoadFromLisp\n")
		} else {
			fmt.Printf("🔍 DEBUG [cmd/deploy.go]: cfg.Security.Bastion.Enabled = %v\n", cfg.Security.Bastion.Enabled)
		}
	} else {
		fmt.Printf("🔍 DEBUG [cmd/deploy.go]: No config file, using default config\n")
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
	configs := map[string]auto.ConfigValue{}

	// DigitalOcean token
	if cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.Token != "" {
		configs["digitaloceanToken"] = auto.ConfigValue{Value: cfg.Providers.DigitalOcean.Token, Secret: true}
	}

	// Linode token
	if cfg.Providers.Linode != nil && cfg.Providers.Linode.Token != "" {
		configs["linodeToken"] = auto.ConfigValue{Value: cfg.Providers.Linode.Token, Secret: true}
	}

	// AWS region (credentials come from environment)
	if cfg.Providers.AWS != nil && cfg.Providers.AWS.Region != "" {
		configs["awsRegion"] = auto.ConfigValue{Value: cfg.Providers.AWS.Region}
	}

	// WireGuard configuration
	if cfg.Network.WireGuard != nil {
		if cfg.Network.WireGuard.ServerEndpoint != "" {
			configs["wireguardServerEndpoint"] = auto.ConfigValue{Value: cfg.Network.WireGuard.ServerEndpoint}
		}
		if cfg.Network.WireGuard.ServerPublicKey != "" {
			configs["wireguardServerPublicKey"] = auto.ConfigValue{Value: cfg.Network.WireGuard.ServerPublicKey}
		}
	}

	if len(configs) == 0 {
		return nil
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
	if cfg.Providers.Azure != nil && cfg.Providers.Azure.Enabled {
		providers = append(providers, "Azure")
	}
	if len(providers) > 0 {
		fmt.Printf("  • Providers: %s\n", joinStrings(providers, " + "))
	}

	fmt.Printf("  • Kubernetes: K3s %s\n", cfg.Kubernetes.Version)
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

// installArgoCDIfEnabled installs ArgoCD if enabled in the configuration
func installArgoCDIfEnabled(cfg *config.ClusterConfig, outputs auto.OutputMap) error {
	// Check if ArgoCD is enabled
	if cfg.Addons.ArgoCD == nil || !cfg.Addons.ArgoCD.Enabled {
		return nil // ArgoCD not enabled, skip
	}

	// Get master node IP from outputs
	// The nodes are exported as a map in the format: {"node_0": {"public_ip": "...", ...}, ...}
	nodesOutput, ok := outputs["nodes"]
	if !ok {
		return fmt.Errorf("nodes output not found")
	}

	nodesMap, ok := nodesOutput.Value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("nodes output is not a map")
	}

	// Find the first master node
	var masterNodeIP string
	for _, nodeData := range nodesMap {
		nodeMap, ok := nodeData.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this node has master role
		roles, ok := nodeMap["roles"]
		if !ok {
			continue
		}

		rolesStr, ok := roles.(string)
		if !ok {
			continue
		}

		if strings.Contains(rolesStr, "master") || strings.Contains(rolesStr, "control-plane") {
			// Get the public IP
			publicIP, ok := nodeMap["public_ip"]
			if !ok {
				continue
			}

			masterNodeIP, ok = publicIP.(string)
			if !ok {
				continue
			}

			break
		}
	}

	if masterNodeIP == "" {
		return fmt.Errorf("no master node found in cluster outputs")
	}

	// Get SSH private key from outputs
	sshKeyOutput, ok := outputs["sshPrivateKey"]
	if !ok {
		return fmt.Errorf("sshPrivateKey output not found")
	}

	sshPrivateKey, ok := sshKeyOutput.Value.(string)
	if !ok {
		return fmt.Errorf("sshPrivateKey output is not a string")
	}

	// Install ArgoCD
	return addons.InstallArgoCD(cfg, masterNodeIP, sshPrivateKey)
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
	fmt.Printf("  kubernetes-create deploy --config <your-config>.lisp\n")
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

func printWarning(text string) {
	color.Yellow(text)
}

func confirm(question string) bool {
	fmt.Printf("\n%s (y/N): ", color.YellowString("❓ "+question))
	var response string
	fmt.Scanln(&response)
	return response == "y" || response == "Y" || response == "yes"
}
