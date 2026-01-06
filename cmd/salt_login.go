package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chalkan3/sloth-kubernetes/internal/common"
	"github.com/chalkan3/sloth-kubernetes/pkg/operations"
	"github.com/chalkan3/sloth-kubernetes/pkg/salt"
	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/spf13/cobra"
)

var (
	saltConfigPath string
	skipVerify     bool
)

// SaltConfig stores the Salt API connection information
type SaltConfig struct {
	APIURL     string `json:"api_url"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	BastionIP  string `json:"bastion_ip"`
	StackName  string `json:"stack_name"`
	ConfigFile string `json:"config_file"`
}

var saltLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Salt API using Pulumi stack information",
	Long: `Automatically retrieves bastion host information from the Pulumi stack
and configures Salt API access. This eliminates the need to manually
set environment variables or flags for each Salt command.

The command:
  1. Reads the current Pulumi stack
  2. Retrieves bastion host IP from stack outputs
  3. Tests connection to Salt API
  4. Saves configuration for future use

Configuration is saved to ~/.sloth-kubernetes/salt-config.json

After running this once, all 'sloth-kubernetes salt' commands will
automatically use the saved configuration.`,
	Example: `  # Login using current stack
  sloth-kubernetes salt login

  # Login to specific stack
  sloth-kubernetes salt login --stack cluster-prod

  # Login with custom config file
  sloth-kubernetes salt login --config cluster.lisp

  # Skip connection verification
  sloth-kubernetes salt login --skip-verify`,
	RunE: runSaltLogin,
}

func init() {
	saltCmd.AddCommand(saltLoginCmd)

	saltLoginCmd.Flags().StringVar(&saltConfigPath, "config", "", "Path to cluster config file")
	saltLoginCmd.Flags().StringVar(&stackName, "stack", "", "Pulumi stack name (defaults to current stack)")
	saltLoginCmd.Flags().BoolVar(&skipVerify, "skip-verify", false, "Skip connection verification")
}

func runSaltLogin(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	printHeader("🔐 Salt API Login")

	// Create workspace
	printInfo("📦 Loading Pulumi workspace...")
	ws, err := createWorkspaceForSalt(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Select or get stack
	var stack auto.Stack
	if stackName != "" {
		printInfo(fmt.Sprintf("📚 Selecting stack: %s", stackName))
		stack, err = auto.SelectStack(ctx, stackName, ws)
		if err != nil {
			return fmt.Errorf("failed to select stack %q: %w", stackName, err)
		}
	} else {
		printInfo("📚 Using current stack")
		stack, err = auto.SelectStack(ctx, "dev", ws)
		if err != nil {
			// Try to get default stack
			stacks, _ := ws.ListStacks(ctx)
			if len(stacks) == 0 {
				return fmt.Errorf("no stacks found. Please deploy a cluster first with 'sloth-kubernetes deploy'")
			}
			stackName = string(stacks[0].Name)
			stack, err = auto.SelectStack(ctx, stackName, ws)
			if err != nil {
				return fmt.Errorf("failed to select stack: %w", err)
			}
		} else {
			stackName = "dev"
		}
	}

	printSuccess(fmt.Sprintf("✓ Using stack: %s", stackName))

	// Refresh to get latest outputs
	printInfo("🔄 Refreshing stack outputs...")
	_, err = stack.Refresh(ctx, optrefresh.ShowSecrets(true))
	if err != nil {
		printWarning("⚠️  Could not refresh stack, using cached outputs")
	}

	// Get stack outputs
	printInfo("📊 Retrieving bastion information...")
	outputs, err := stack.Outputs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stack outputs: %w", err)
	}

	// Look for bastion in outputs
	bastionOutput, hasBastionOutput := outputs["bastion"]
	if !hasBastionOutput {
		// Try nodes output (older format)
		nodesOutput, hasNodes := outputs["nodes"]
		if !hasNodes {
			return fmt.Errorf("no bastion or nodes found in stack outputs. Ensure your cluster has a bastion host enabled")
		}

		// Parse nodes to find bastion
		bastionOutput = nodesOutput
	}

	// Extract bastion IP
	bastionIP, err := extractBastionIP(bastionOutput.Value)
	if err != nil {
		return fmt.Errorf("failed to extract bastion IP: %w", err)
	}

	printSuccess(fmt.Sprintf("✓ Found bastion host: %s", bastionIP))

	// Build Salt API URL
	saltAPIURL := fmt.Sprintf("http://%s:8000", bastionIP)

	// Default credentials (as documented in bastion setup)
	saltUsername := getEnvOrDefault("SALT_USERNAME", "saltapi")
	saltPassword := getEnvOrDefault("SALT_PASSWORD", "saltapi123")

	printInfo(fmt.Sprintf("🌐 Salt API URL: %s", saltAPIURL))
	printInfo(fmt.Sprintf("👤 Username: %s", saltUsername))

	// Create Salt client
	client := salt.NewClient(saltAPIURL, saltUsername, saltPassword)

	// Test connection (unless skipped)
	if !skipVerify {
		printInfo("🔌 Testing connection to Salt API...")

		if err := client.Login(); err != nil {
			return fmt.Errorf("failed to connect to Salt API: %w\n\nTroubleshooting:\n  • Ensure the cluster is fully deployed\n  • Check that bastion host is running\n  • Verify Salt Master and Salt API are installed\n  • Try: ssh root@%s 'systemctl status salt-master salt-api'", err, bastionIP)
		}

		printSuccess("✓ Successfully authenticated to Salt API")

		// Test ping
		printInfo("📡 Testing minion connectivity...")
		resp, err := client.Ping("*")
		if err != nil {
			printWarning("⚠️  Could not ping minions (this is normal if nodes are still provisioning)")
		} else {
			minionCount := len(resp)
			if minionCount > 0 {
				printSuccess(fmt.Sprintf("✓ Connected to %d minion(s)", minionCount))
			} else {
				printWarning("⚠️  No minions responded (cluster may still be provisioning)")
			}
		}
	}

	// Save credentials to Pulumi state for auto-login
	printInfo("💾 Saving credentials to Pulumi state...")
	saltCreds := &operations.SaltCredentials{
		APIURL:      saltAPIURL,
		Username:    saltUsername,
		Password:    saltPassword,
		BastionIP:   bastionIP,
		AuthMethod:  "pam",
		InstalledAt: time.Now().UTC(),
	}

	if err := operations.SaveSaltCredentials(stackName, saltCreds); err != nil {
		printWarning(fmt.Sprintf("⚠️  Could not save to Pulumi state: %v", err))
	} else {
		printSuccess("✓ Credentials saved to Pulumi state (auto-login enabled)")
	}

	// Save configuration to local file as backup
	config := SaltConfig{
		APIURL:     saltAPIURL,
		Username:   saltUsername,
		Password:   saltPassword,
		BastionIP:  bastionIP,
		StackName:  stackName,
		ConfigFile: saltConfigPath,
	}

	if err := saveSaltConfig(config); err != nil {
		printWarning(fmt.Sprintf("⚠️  Could not save local configuration: %v", err))
		printInfo("You can still use Salt commands by setting environment variables:")
		fmt.Printf("\n  export SALT_API_URL=%s\n", saltAPIURL)
		fmt.Printf("  export SALT_USERNAME=%s\n", saltUsername)
		fmt.Printf("  export SALT_PASSWORD=%s\n", saltPassword)
	} else {
		printSuccess("✓ Configuration saved to ~/.sloth-kubernetes/salt-config.json")
	}

	printHeader("✅ Login Complete!")
	fmt.Println()
	printInfo("You can now use Salt commands without additional configuration:")
	fmt.Println()
	fmt.Println("  " + color.CyanString("sloth-kubernetes salt ping"))
	fmt.Println("  " + color.CyanString("sloth-kubernetes salt cmd \"uptime\""))
	fmt.Println("  " + color.CyanString("sloth-kubernetes salt system disk"))
	fmt.Println()
	printInfo("For more examples, see: sloth-kubernetes salt --help")

	return nil
}

// extractBastionIP extracts the bastion IP from Pulumi output
func extractBastionIP(value interface{}) (string, error) {
	// Try direct string
	if str, ok := value.(string); ok {
		return str, nil
	}

	// Try map format (bastion: {public_ip: "x.x.x.x", ...})
	if m, ok := value.(map[string]interface{}); ok {
		if pubIP, ok := m["public_ip"].(string); ok && pubIP != "" {
			return pubIP, nil
		}

		// Try other possible keys
		for _, key := range []string{"ip", "ipv4", "address", "public_address"} {
			if ip, ok := m[key].(string); ok && ip != "" {
				return ip, nil
			}
		}
	}

	// Try array format (look for bastion in nodes array)
	if arr, ok := value.([]interface{}); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				// Check if this is bastion
				if name, ok := m["name"].(string); ok && name == "bastion" {
					if pubIP, ok := m["public_ip"].(string); ok && pubIP != "" {
						return pubIP, nil
					}
				}
				// Or if it has "bastion" role
				if roles, ok := m["roles"].([]interface{}); ok {
					for _, role := range roles {
						if roleStr, ok := role.(string); ok && roleStr == "bastion" {
							if pubIP, ok := m["public_ip"].(string); ok && pubIP != "" {
								return pubIP, nil
							}
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("could not extract bastion IP from output (found type: %T)", value)
}

// saveSaltConfig saves the Salt configuration to disk
func saveSaltConfig(config SaltConfig) error {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create config directory
	configDir := filepath.Join(homeDir, ".sloth-kubernetes")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	configFile := filepath.Join(configDir, "salt-config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// loadSaltConfig loads the saved Salt configuration
func loadSaltConfig() (*SaltConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configFile := filepath.Join(homeDir, ".sloth-kubernetes", "salt-config.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config SaltConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// createWorkspaceForSalt creates a Pulumi workspace for Salt operations
func createWorkspaceForSalt(ctx context.Context) (auto.Workspace, error) {
	// Load saved S3 backend configuration
	_ = common.LoadSavedConfig()

	projectName := "sloth-kubernetes"
	workspaceOpts := []auto.LocalWorkspaceOption{
		auto.Project(workspace.Project{
			Name:    tokens.PackageName(projectName),
			Runtime: workspace.NewProjectRuntimeInfo("go", nil),
		}),
	}

	// Collect all AWS/S3 environment variables
	envVars := make(map[string]string)
	awsEnvKeys := []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
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
	if len(envVars) > 0 {
		workspaceOpts = append(workspaceOpts, auto.EnvVars(envVars))
	}

	// If PULUMI_BACKEND_URL is set, use passphrase secrets provider
	if backendURL := os.Getenv("PULUMI_BACKEND_URL"); backendURL != "" {
		workspaceOpts = append(workspaceOpts, auto.SecretsProvider("passphrase"))
		if os.Getenv("PULUMI_CONFIG_PASSPHRASE") == "" {
			os.Setenv("PULUMI_CONFIG_PASSPHRASE", "")
		}
	}

	// Create workspace
	ws, err := auto.NewLocalWorkspace(ctx, workspaceOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	return ws, nil
}
