package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/pkg/operations"
	"github.com/chalkan3/sloth-kubernetes/pkg/salt"
)

var (
	saltAPIURL       string
	saltUsername     string
	saltPassword     string
	saltTarget       string
	saltOutputJSON   bool
	pushApplyAfter   bool
	pushRefreshAfter bool
	pushDryRun       bool
	pushExcludes     []string
)

var saltCmd = &cobra.Command{
	Use:   "salt",
	Short: "Manage cluster nodes with SaltStack",
	Long: `Interact with cluster nodes using SaltStack API.

SaltStack provides powerful remote execution and configuration management
capabilities for your cluster nodes. This command allows you to execute
commands, apply states, and manage minions through the Salt API.

The Salt Master is automatically installed on the bastion host during deployment.

Configuration:
  Set these environment variables or use flags:
  • SALT_API_URL - Salt API endpoint (default: http://bastion-ip:8000)
  • SALT_USERNAME - Salt API username (default: saltapi)
  • SALT_PASSWORD - Salt API password (default: saltapi123)`,
	Example: `  # Ping all minions
  sloth-kubernetes salt ping

  # List all connected minions
  sloth-kubernetes salt minions

  # Execute command on all minions
  sloth-kubernetes salt cmd "uptime"

  # Execute command on specific target
  sloth-kubernetes salt cmd "df -h" --target "web*"

  # Get system information
  sloth-kubernetes salt grains --target "master*"

  # Apply a Salt state
  sloth-kubernetes salt state apply webserver

  # List minion keys
  sloth-kubernetes salt keys list

  # Accept pending minion keys
  sloth-kubernetes salt keys accept node-1`,
}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping all or specific minions",
	Long:  `Test connectivity to Salt minions using test.ping`,
	Example: `  # Ping all minions
  sloth-kubernetes salt ping

  # Ping specific minions
  sloth-kubernetes salt ping --target "master*"`,
	RunE: runSaltPing,
}

var minionsCmd = &cobra.Command{
	Use:   "minions",
	Short: "List all connected minions",
	Long:  `List all minions currently connected to the Salt Master`,
	RunE:  runSaltMinions,
}

var cmdCmd = &cobra.Command{
	Use:   "cmd <command>",
	Short: "Execute shell command on minions",
	Long:  `Execute a shell command on target minions using cmd.run`,
	Example: `  # Run command on all minions
  sloth-kubernetes salt cmd "uptime"

  # Run on specific target
  sloth-kubernetes salt cmd "systemctl status k3s" --target "master*"

  # Get disk usage
  sloth-kubernetes salt cmd "df -h"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSaltCmd,
}

var grainsCmd = &cobra.Command{
	Use:   "grains",
	Short: "Get system information (grains) from minions",
	Long:  `Retrieve grain data (system information) from minions`,
	Example: `  # Get all grains from all minions
  sloth-kubernetes salt grains

  # Get grains from specific minions
  sloth-kubernetes salt grains --target "worker*"`,
	RunE: runSaltGrains,
}

var saltStateCmd = &cobra.Command{
	Use:   "state",
	Short: "Manage Salt states",
	Long:  `Apply Salt states to configure minions`,
}

var saltStateApplyCmd = &cobra.Command{
	Use:   "apply <state>",
	Short: "Apply a Salt state to minions",
	Long:  `Apply a specific Salt state to target minions`,
	Example: `  # Apply state to all minions
  sloth-kubernetes salt state apply webserver

  # Apply to specific target
  sloth-kubernetes salt state apply nginx --target "web*"`,
	Args: cobra.ExactArgs(1),
	RunE: runSaltStateApply,
}

var saltStateHighstateCmd = &cobra.Command{
	Use:   "highstate",
	Short: "Apply full highstate to minions",
	Long:  `Apply the complete highstate (all configured states) to minions`,
	Example: `  # Apply highstate to all minions
  sloth-kubernetes salt state highstate

  # Apply to specific target
  sloth-kubernetes salt state highstate --target "master*"`,
	RunE: runSaltHighstate,
}

var saltStatePushCmd = &cobra.Command{
	Use:   "push <path>",
	Short: "Push local Salt states to the Salt master",
	Long: `Upload local Salt state files to /srv/salt/ on the Salt master.

This command transfers your local Salt state files to the Salt master node
using SCP (via bastion if configured). The files will be extracted to
/srv/salt/ on the master.`,
	Example: `  # Push states from local directory
  sloth-kubernetes salt state push ./salt/states/

  # Push and apply highstate immediately
  sloth-kubernetes salt state push ./salt/states/ --apply

  # Dry run - show what would be transferred
  sloth-kubernetes salt state push ./salt/states/ --dry-run

  # Exclude certain patterns
  sloth-kubernetes salt state push ./salt/states/ --exclude "*.pyc" --exclude "__pycache__"`,
	Args: cobra.ExactArgs(1),
	RunE: runStatePush,
}

var saltPillarCmd = &cobra.Command{
	Use:   "pillar",
	Short: "Manage Salt pillars",
	Long:  `Manage Salt pillar data for minion configuration`,
}

var saltPillarPushCmd = &cobra.Command{
	Use:   "push <path>",
	Short: "Push local Salt pillars to the Salt master",
	Long: `Upload local Salt pillar files to /srv/pillar/ on the Salt master.

This command transfers your local pillar files to the Salt master node
using SCP (via bastion if configured). The files will be extracted to
/srv/pillar/ on the master.`,
	Example: `  # Push pillars from local directory
  sloth-kubernetes salt pillar push ./salt/pillars/

  # Push and refresh pillars on all minions
  sloth-kubernetes salt pillar push ./salt/pillars/ --refresh

  # Dry run - show what would be transferred
  sloth-kubernetes salt pillar push ./salt/pillars/ --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runPillarPush,
}

var saltPillarRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh pillars on all minions",
	Long:  `Refresh pillar data on all minions to pick up changes`,
	RunE:  runPillarRefresh,
}

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage minion keys",
	Long:  `Manage Salt minion authentication keys`,
}

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all minion keys",
	Long:  `List all minion keys (accepted, pending, rejected, denied)`,
	RunE:  runSaltKeysList,
}

var keysAcceptCmd = &cobra.Command{
	Use:   "accept <minion-id>",
	Short: "Accept a pending minion key",
	Long:  `Accept a minion's authentication key to allow it to connect`,
	Example: `  # Accept specific minion
  sloth-kubernetes salt keys accept node-1

  # Accept all pending keys
  sloth-kubernetes salt keys accept "*"`,
	Args: cobra.ExactArgs(1),
	RunE: runSaltKeysAccept,
}

func init() {
	rootCmd.AddCommand(saltCmd)

	// Add subcommands
	saltCmd.AddCommand(pingCmd)
	saltCmd.AddCommand(minionsCmd)
	saltCmd.AddCommand(cmdCmd)
	saltCmd.AddCommand(grainsCmd)
	saltCmd.AddCommand(saltStateCmd)
	saltCmd.AddCommand(keysCmd)
	saltCmd.AddCommand(saltPillarCmd)

	// State subcommands
	saltStateCmd.AddCommand(saltStateApplyCmd)
	saltStateCmd.AddCommand(saltStateHighstateCmd)
	saltStateCmd.AddCommand(saltStatePushCmd)

	// Pillar subcommands
	saltPillarCmd.AddCommand(saltPillarPushCmd)
	saltPillarCmd.AddCommand(saltPillarRefreshCmd)

	// Keys subcommands
	keysCmd.AddCommand(keysListCmd)
	keysCmd.AddCommand(keysAcceptCmd)

	// State push flags
	saltStatePushCmd.Flags().BoolVar(&pushApplyAfter, "apply", false, "Apply highstate after pushing states")
	saltStatePushCmd.Flags().BoolVar(&pushDryRun, "dry-run", false, "Show what would be transferred without actually transferring")
	saltStatePushCmd.Flags().StringSliceVar(&pushExcludes, "exclude", nil, "Patterns to exclude from transfer (can be specified multiple times)")

	// Pillar push flags
	saltPillarPushCmd.Flags().BoolVar(&pushRefreshAfter, "refresh", false, "Refresh pillars on all minions after pushing")
	saltPillarPushCmd.Flags().BoolVar(&pushDryRun, "dry-run", false, "Show what would be transferred without actually transferring")

	// Load saved configuration if available
	defaultURL := getEnvOrDefault("SALT_API_URL", "")
	defaultUser := getEnvOrDefault("SALT_USERNAME", "saltapi")
	defaultPass := getEnvOrDefault("SALT_PASSWORD", "saltapi123")

	// Try to load from saved config file
	if savedConfig, err := loadSaltConfig(); err == nil {
		if defaultURL == "" {
			defaultURL = savedConfig.APIURL
		}
		if defaultUser == "saltapi" {
			defaultUser = savedConfig.Username
		}
		if defaultPass == "saltapi123" {
			defaultPass = savedConfig.Password
		}
	}

	// Persistent flags for all salt commands
	saltCmd.PersistentFlags().StringVar(&saltAPIURL, "url", defaultURL, "Salt API URL (e.g., http://bastion-ip:8000)")
	saltCmd.PersistentFlags().StringVar(&saltUsername, "username", defaultUser, "Salt API username")
	saltCmd.PersistentFlags().StringVar(&saltPassword, "password", defaultPass, "Salt API password")
	saltCmd.PersistentFlags().StringVarP(&saltTarget, "target", "t", "*", "Target minions (glob, grain, list, etc.)")
	saltCmd.PersistentFlags().BoolVar(&saltOutputJSON, "json", false, "Output raw JSON response")
}

func getSaltClient() (*salt.Client, error) {
	// If no URL provided, try to auto-login from saved credentials
	if saltAPIURL == "" {
		// First, try to load credentials from Pulumi state
		targetStack := stackName
		if targetStack == "" {
			return nil, fmt.Errorf(`stack name is required for Salt API auto-login

Specify a stack with:
  sloth-kubernetes salt <command> --stack <stack-name>

Or set Salt API credentials manually:
  export SALT_API_URL="http://master-ip:8000"
  export SALT_USERNAME="saltapi"
  export SALT_PASSWORD="your-password"`)
		}

		creds, err := operations.GetSaltCredentialsFromStack(targetStack)
		if err == nil && creds != nil {
			saltAPIURL = creds.APIURL
			saltUsername = creds.Username
			saltPassword = creds.Password
			color.Green("🔐 Auto-login: Using Salt credentials from Pulumi state")
		} else {
			// Fallback to auto-login from stack outputs
			if err := autoLoginFromStack(); err != nil {
				return nil, fmt.Errorf(`Salt API URL is required.

Please run one of the following:

  1. Use with stack flag (auto-login):
     %s

  2. Login to Salt using your stack:
     %s

  3. Set environment variables:
     export SALT_API_URL="http://master-ip:8000"
     export SALT_USERNAME="saltapi"
     export SALT_PASSWORD="saltapi123"

  4. Use command-line flags:
     --url "http://master-ip:8000" --username saltapi --password saltapi123

Auto-login error: %v`,
					color.CyanString("sloth-kubernetes salt ping -s <stack-name>"),
					color.CyanString("sloth-kubernetes salt login"),
					err)
			}
		}
	}

	client := salt.NewClient(saltAPIURL, saltUsername, saltPassword)

	// Try to set SSH config for push operations
	targetStack := stackName
	if targetStack != "" {
		sshKeyPath := GetSSHKeyPath(targetStack)
		if sshKeyPath != "" {
			if _, err := os.Stat(sshKeyPath); err == nil {
				// Extract master IP from Salt API URL
				masterIP := extractIPFromURL(saltAPIURL)
				if masterIP != "" {
					client.SetSSHConfig(&salt.SSHConfig{
						Host:    masterIP,
						User:    "root",
						KeyPath: sshKeyPath,
					})
				}
			}
		}
	}

	return client, nil
}

// autoLoginFromStack automatically fetches Salt API credentials from the Pulumi stack
// and ensures VPN connectivity if the Salt API is on a VPN IP
func autoLoginFromStack() error {
	ctx := context.Background()

	// Create workspace
	ws, err := createWorkspaceForSalt(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Get stack name from global flag
	targetStack := stackName
	if targetStack == "" {
		return fmt.Errorf("stack name is required for Salt API auto-login")
	}

	// Try to select the stack
	stack, err := auto.SelectStack(ctx, fmt.Sprintf("organization/sloth-kubernetes/%s", targetStack), ws)
	if err != nil {
		// Try without organization prefix
		stack, err = auto.SelectStack(ctx, targetStack, ws)
		if err != nil {
			return fmt.Errorf("failed to select stack %q: %w", targetStack, err)
		}
	}

	// Get stack outputs
	outputs, err := stack.Outputs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stack outputs: %w", err)
	}

	// Try to find Salt API URL from outputs
	var saltIP string

	// First check for salt_master output
	if saltMasterOutput, ok := outputs["salt_master"]; ok {
		if saltMaster, ok := saltMasterOutput.Value.(map[string]interface{}); ok {
			if apiURL, ok := saltMaster["api_url"].(string); ok && apiURL != "" {
				saltAPIURL = apiURL
				// Extract IP from URL
				saltIP = extractIPFromURL(apiURL)

				// Get credentials from salt_master output if available
				// Try different naming conventions for username
				if user, ok := saltMaster["api_username"].(string); ok && user != "" {
					saltUsername = user
				} else if user, ok := saltMaster["api_user"].(string); ok && user != "" {
					saltUsername = user
				}
				// Get password (shared secret for sharedsecret auth)
				if pass, ok := saltMaster["api_password"].(string); ok && pass != "" {
					saltPassword = pass
				}
			}
		}
	}

	// Fallback: Check for bastion output
	if saltAPIURL == "" {
		if bastionOutput, ok := outputs["bastion"]; ok {
			bastionIP, err := extractBastionIP(bastionOutput.Value)
			if err == nil && bastionIP != "" {
				saltAPIURL = fmt.Sprintf("http://%s:8000", bastionIP)
				saltIP = bastionIP
			}
		}
	}

	// Fallback: Check nodes output for master node
	if saltAPIURL == "" {
		if nodesOutput, ok := outputs["nodes"]; ok {
			if nodes, ok := nodesOutput.Value.([]interface{}); ok {
				for _, node := range nodes {
					if nodeMap, ok := node.(map[string]interface{}); ok {
						// Check for master role
						if roles, ok := nodeMap["roles"].([]interface{}); ok {
							for _, role := range roles {
								if roleStr, ok := role.(string); ok && (roleStr == "master" || roleStr == "control-plane") {
									// Use VPN IP if available, otherwise public IP
									if vpnIP, ok := nodeMap["vpn_ip"].(string); ok && vpnIP != "" {
										saltAPIURL = fmt.Sprintf("http://%s:8000", vpnIP)
										saltIP = vpnIP
										break
									}
									if pubIP, ok := nodeMap["public_ip"].(string); ok && pubIP != "" {
										saltAPIURL = fmt.Sprintf("http://%s:8000", pubIP)
										saltIP = pubIP
										break
									}
								}
							}
						}
					}
				}
			}
		}
	}

	if saltAPIURL == "" {
		return fmt.Errorf("no Salt API URL found in stack outputs")
	}

	// Save credentials to operations history for future auto-login
	// This makes subsequent calls faster by avoiding stack output lookup
	saltCreds := &operations.SaltCredentials{
		APIURL:      saltAPIURL,
		Username:    saltUsername,
		Password:    saltPassword,
		BastionIP:   saltIP,
		AuthMethod:  "sharedsecret",
		InstalledAt: time.Now().UTC(),
	}
	if err := operations.SaveSaltCredentials(targetStack, saltCreds); err != nil {
		// Non-fatal: just log warning
		color.Yellow("⚠️  Could not cache credentials to state: %v", err)
	}

	// Check if Salt IP is on VPN network (10.8.0.x)
	if isVPNIP(saltIP) {
		color.Cyan("🔐 Salt API is on VPN network: %s", saltAPIURL)

		// Check if we can reach the VPN IP
		if !canReachIP(saltIP, 8000) {
			color.Yellow("⚠️  Cannot reach Salt API - VPN connection required")

			// Check if WireGuard is running locally
			if !isWireGuardRunning() {
				color.Cyan("🔗 Attempting to join VPN automatically...")

				// Try to join VPN
				if err := autoJoinVPN(targetStack, outputs); err != nil {
					return fmt.Errorf("failed to auto-join VPN: %w\n\nPlease join the VPN manually:\n  sloth-kubernetes vpn join %s --install", err, targetStack)
				}

				// Wait for VPN to establish
				color.Cyan("⏳ Waiting for VPN connection to establish...")
				time.Sleep(3 * time.Second)

				// Verify we can now reach the Salt API
				if !canReachIP(saltIP, 8000) {
					return fmt.Errorf("VPN connected but still cannot reach Salt API at %s", saltAPIURL)
				}
			} else {
				// WireGuard is running but can't reach the IP - might be wrong network
				return fmt.Errorf("WireGuard is running but cannot reach Salt API at %s. Check your VPN configuration", saltAPIURL)
			}
		}

		color.Green("✅ VPN connected - Salt API reachable at %s", saltAPIURL)
	} else {
		color.Green("🔐 Auto-login: Using Salt API from stack %q: %s", targetStack, saltAPIURL)
	}

	return nil
}

// extractIPFromURL extracts the IP address from a URL like "http://10.8.0.10:8000"
func extractIPFromURL(url string) string {
	// Remove protocol
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	// Remove port
	if idx := strings.Index(url, ":"); idx != -1 {
		url = url[:idx]
	}

	return url
}

// isVPNIP checks if an IP is in the VPN range (10.8.0.0/24)
func isVPNIP(ip string) bool {
	return strings.HasPrefix(ip, "10.8.0.")
}

// canReachIP checks if we can establish a TCP connection to an IP:port
func canReachIP(ip string, port int) bool {
	address := net.JoinHostPort(ip, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// isWireGuardRunning checks if WireGuard is running on the local machine
func isWireGuardRunning() bool {
	// Try to detect WireGuard by running 'wg show' which works on both Linux and macOS
	cmd := exec.Command("sh", "-c", "sudo wg show 2>/dev/null | head -1")
	output, err := cmd.CombinedOutput()
	return err == nil && len(strings.TrimSpace(string(output))) > 0
}

// autoJoinVPN automatically joins the VPN for the given stack
func autoJoinVPN(stackName string, outputs auto.OutputMap) error {
	// Parse nodes from outputs
	nodes, err := ParseNodeOutputs(outputs)
	if err != nil {
		return fmt.Errorf("failed to parse nodes: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found in stack")
	}

	// Get SSH key path
	sshKeyPath := GetSSHKeyPath(stackName)

	// Check if SSH key exists
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key not found at %s. Please ensure the cluster was deployed from this machine", sshKeyPath)
	}

	// Get bastion info if enabled
	bastionEnabled := false
	bastionIP := ""

	if bastionEnabledOutput, ok := outputs["bastion_enabled"]; ok {
		if bastionEnabledOutput.Value != nil {
			bastionEnabled = bastionEnabledOutput.Value == true
		}
	}

	if bastionEnabled {
		if bastionOutput, ok := outputs["bastion"]; ok {
			if bastionMap, ok := bastionOutput.Value.(map[string]interface{}); ok {
				if pubIP, ok := bastionMap["public_ip"].(string); ok {
					bastionIP = pubIP
				}
			}
		}
	}

	// Generate WireGuard keypair
	color.Cyan("🔑 Generating WireGuard keypair...")
	privateKey, publicKey, err := generateWireGuardKeypair()
	if err != nil {
		return fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Auto-assign VPN IP (100-254 range for clients)
	vpnIP := "10.8.0.100"
	for i := 100; i < 255; i++ {
		candidateIP := fmt.Sprintf("10.8.0.%d", i)
		// Simple assignment - could check for conflicts in production
		vpnIP = candidateIP
		break
	}

	color.Cyan("📡 VPN IP: %s", vpnIP)

	// Add peer to cluster nodes
	color.Cyan("🔗 Adding peer to cluster nodes...")
	for _, node := range nodes {
		targetIP := node.PublicIP
		peerAddScript := generatePeerAddScript(vpnIP, publicKey, "cli-auto-join")

		var sshCmd *exec.Cmd
		if bastionEnabled && bastionIP != "" {
			nodeTargetIP := node.WireGuardIP
			if nodeTargetIP == "" {
				nodeTargetIP = node.PrivateIP
				if nodeTargetIP == "" {
					nodeTargetIP = node.PublicIP
				}
			}
			sshUser := getSSHUserForNode(node.Provider)
			sshCmd = exec.Command("ssh",
				"-i", sshKeyPath,
				"-o", "StrictHostKeyChecking=accept-new",
				"-o", "UserKnownHostsFile=/dev/null",
				"-o", "ConnectTimeout=10",
				"-o", fmt.Sprintf("ProxyCommand=ssh -i %s -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=/dev/null -W %%h:%%p root@%s", sshKeyPath, bastionIP),
				fmt.Sprintf("%s@%s", sshUser, nodeTargetIP),
				"bash", "-s",
			)
		} else {
			sshUser := getSSHUserForNode(node.Provider)
			sshCmd = exec.Command("ssh",
				"-i", sshKeyPath,
				"-o", "StrictHostKeyChecking=accept-new",
				"-o", "UserKnownHostsFile=/dev/null",
				"-o", "ConnectTimeout=10",
				fmt.Sprintf("%s@%s", sshUser, targetIP),
				"bash", "-s",
			)
		}
		sshCmd.Stdin = strings.NewReader(peerAddScript)

		if _, err := sshCmd.CombinedOutput(); err != nil {
			color.Yellow("  ⚠️  Failed to add peer to %s: %v", node.Name, err)
		} else {
			color.Green("  ✓ Added peer to %s", node.Name)
		}
	}

	// Generate and install client config
	color.Cyan("📝 Generating WireGuard configuration...")
	clientConfig := generateClientConfig(privateKey, vpnIP, "cli-auto-join", nodes, nil, sshKeyPath, bastionEnabled, bastionIP)

	// Detect OS and install
	osType := detectOS()

	switch osType {
	case "darwin":
		// macOS installation
		mkdirCmd := exec.Command("sudo", "mkdir", "-p", "/opt/homebrew/etc/wireguard")
		if err := mkdirCmd.Run(); err != nil {
			// Try alternative path
			mkdirCmd = exec.Command("sudo", "mkdir", "-p", "/usr/local/etc/wireguard")
			if err := mkdirCmd.Run(); err != nil {
				return fmt.Errorf("failed to create WireGuard directory: %w", err)
			}
		}

		// Write config to temp file first
		tmpFile := "/tmp/wg0-auto.conf"
		if err := os.WriteFile(tmpFile, []byte(clientConfig), 0600); err != nil {
			return fmt.Errorf("failed to write temp config: %w", err)
		}

		// Copy to WireGuard directory
		cpCmd := exec.Command("sudo", "cp", tmpFile, "/opt/homebrew/etc/wireguard/wg0.conf")
		if err := cpCmd.Run(); err != nil {
			cpCmd = exec.Command("sudo", "cp", tmpFile, "/usr/local/etc/wireguard/wg0.conf")
			if err := cpCmd.Run(); err != nil {
				return fmt.Errorf("failed to install config: %w", err)
			}
		}

		// Start WireGuard
		color.Cyan("🚀 Starting WireGuard VPN...")
		upCmd := exec.Command("sudo", "wg-quick", "up", "wg0")
		if output, err := upCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to start WireGuard: %w (output: %s)", err, string(output))
		}

	case "linux":
		// Linux installation
		mkdirCmd := exec.Command("sudo", "mkdir", "-p", "/etc/wireguard")
		if err := mkdirCmd.Run(); err != nil {
			return fmt.Errorf("failed to create WireGuard directory: %w", err)
		}

		// Write config
		tmpFile := "/tmp/wg0-auto.conf"
		if err := os.WriteFile(tmpFile, []byte(clientConfig), 0600); err != nil {
			return fmt.Errorf("failed to write temp config: %w", err)
		}

		cpCmd := exec.Command("sudo", "cp", tmpFile, "/etc/wireguard/wg0.conf")
		if err := cpCmd.Run(); err != nil {
			return fmt.Errorf("failed to install config: %w", err)
		}

		// Start WireGuard
		color.Cyan("🚀 Starting WireGuard VPN...")
		upCmd := exec.Command("sudo", "wg-quick", "up", "wg0")
		if output, err := upCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to start WireGuard: %w (output: %s)", err, string(output))
		}

	default:
		return fmt.Errorf("unsupported OS: %s. Please install WireGuard manually", osType)
	}

	color.Green("✅ VPN connected successfully!")
	return nil
}

func runSaltPing(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("🔍 Pinging Salt minions...")
	fmt.Println()

	results, err := client.Ping(saltTarget)
	if err != nil {
		color.Red("❌ Ping failed: %v", err)
		return err
	}

	if len(results) == 0 {
		color.Yellow("⚠️  No minions responded to ping")
		return nil
	}

	color.Green("✅ Connected minions:")
	for minion, responsive := range results {
		if responsive {
			color.Green("  • %s: online", minion)
		} else {
			color.Red("  • %s: offline", minion)
		}
	}

	fmt.Println()
	return nil
}

func runSaltMinions(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("📋 Listing Salt minions...")
	fmt.Println()

	minions, err := client.GetMinions()
	if err != nil {
		color.Red("❌ Failed to list minions: %v", err)
		return err
	}

	if len(minions) == 0 {
		color.Yellow("⚠️  No minions found")
		return nil
	}

	color.Green("✅ Connected minions (%d):", len(minions))
	for _, minion := range minions {
		fmt.Printf("  • %s\n", minion)
	}

	fmt.Println()
	return nil
}

func runSaltCmd(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	command := strings.Join(args, " ")

	fmt.Println()
	color.Cyan("🔧 Executing command: %s", command)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.RunShellCommand(saltTarget, command)
	if err != nil {
		color.Red("❌ Command execution failed: %v", err)
		operations.RecordSaltOperation(stackName, "cmd", saltTarget, "cmd.run", command, "failed", "", 0, 0, 0, time.Since(startTime), err)
		return err
	}

	if saltOutputJSON {
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	if len(resp.Return) == 0 || len(resp.Return[0]) == 0 {
		color.Yellow("⚠️  No results returned")
		operations.RecordSaltOperation(stackName, "cmd", saltTarget, "cmd.run", command, "success", "No results returned", 0, 0, 0, time.Since(startTime), nil)
		return nil
	}

	nodesTargeted := len(resp.Return[0])
	color.Green("✅ Results:")
	fmt.Println()
	for minion, result := range resp.Return[0] {
		color.Cyan("Minion: %s", minion)
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("%v\n", result)
		fmt.Println()
	}

	// Record the operation
	operations.RecordSaltOperation(stackName, "cmd", saltTarget, "cmd.run", command, "success", "", nodesTargeted, nodesTargeted, 0, time.Since(startTime), nil)

	return nil
}

func runSaltGrains(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("📊 Retrieving grain data...")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.GetGrains(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get grains: %v", err)
		return err
	}

	if saltOutputJSON {
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	if len(resp.Return) == 0 || len(resp.Return[0]) == 0 {
		color.Yellow("⚠️  No grains data returned")
		return nil
	}

	color.Green("✅ Grains:")
	fmt.Println()
	for minion, grains := range resp.Return[0] {
		color.Cyan("Minion: %s", minion)
		fmt.Println(strings.Repeat("-", 60))

		if grainsMap, ok := grains.(map[string]interface{}); ok {
			// Show key information
			if os, ok := grainsMap["os"].(string); ok {
				fmt.Printf("  OS: %s\n", os)
			}
			if osVersion, ok := grainsMap["osrelease"].(string); ok {
				fmt.Printf("  OS Version: %s\n", osVersion)
			}
			if kernel, ok := grainsMap["kernel"].(string); ok {
				fmt.Printf("  Kernel: %s\n", kernel)
			}
			if cpuArch, ok := grainsMap["cpuarch"].(string); ok {
				fmt.Printf("  CPU Arch: %s\n", cpuArch)
			}
			if numCPUs, ok := grainsMap["num_cpus"]; ok {
				fmt.Printf("  CPUs: %v\n", numCPUs)
			}
			if mem, ok := grainsMap["mem_total"]; ok {
				fmt.Printf("  Memory: %v MB\n", mem)
			}
		} else {
			jsonData, _ := json.MarshalIndent(grains, "  ", "  ")
			fmt.Printf("%s\n", string(jsonData))
		}
		fmt.Println()
	}

	return nil
}

func runSaltStateApply(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	state := args[0]

	fmt.Println()
	color.Cyan("⚙️  Applying state: %s", state)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ApplyState(saltTarget, state)
	if err != nil {
		color.Red("❌ State apply failed: %v", err)
		operations.RecordSaltOperation(stackName, "state", saltTarget, "state.apply", state, "failed", "", 0, 0, 0, time.Since(startTime), err)
		return err
	}

	if saltOutputJSON {
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	color.Green("✅ State applied successfully")
	fmt.Println()

	nodesTargeted := 0
	if len(resp.Return) > 0 {
		nodesTargeted = len(resp.Return[0])
		for minion, result := range resp.Return[0] {
			color.Cyan("Minion: %s", minion)
			fmt.Println(strings.Repeat("-", 60))
			jsonData, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonData))
			fmt.Println()
		}
	}

	// Record the operation
	operations.RecordSaltOperation(stackName, "state", saltTarget, "state.apply", state, "success", "", nodesTargeted, nodesTargeted, 0, time.Since(startTime), nil)

	return nil
}

func runSaltHighstate(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("⚙️  Applying highstate...")
	color.Cyan("Target: %s", saltTarget)
	color.Yellow("⚠️  This may take several minutes...")
	fmt.Println()

	resp, err := client.HighState(saltTarget)
	if err != nil {
		color.Red("❌ Highstate failed: %v", err)
		operations.RecordSaltOperation(stackName, "highstate", saltTarget, "state.highstate", "", "failed", "", 0, 0, 0, time.Since(startTime), err)
		return err
	}

	if saltOutputJSON {
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	color.Green("✅ Highstate completed")
	fmt.Println()

	nodesTargeted := 0
	if len(resp.Return) > 0 {
		nodesTargeted = len(resp.Return[0])
		for minion, result := range resp.Return[0] {
			color.Cyan("Minion: %s", minion)
			fmt.Println(strings.Repeat("-", 60))
			jsonData, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonData))
			fmt.Println()
		}
	}

	// Record the operation
	operations.RecordSaltOperation(stackName, "highstate", saltTarget, "state.highstate", "", "success", "", nodesTargeted, nodesTargeted, 0, time.Since(startTime), nil)

	return nil
}

func runSaltKeysList(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("🔑 Listing minion keys...")
	fmt.Println()

	keys, err := client.KeyList()
	if err != nil {
		color.Red("❌ Failed to list keys: %v", err)
		return err
	}

	if saltOutputJSON {
		jsonData, _ := json.MarshalIndent(keys, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	// Display keys by category
	if accepted, ok := keys["minions"]; ok && len(accepted) > 0 {
		color.Green("✅ Accepted keys (%d):", len(accepted))
		for _, key := range accepted {
			fmt.Printf("  • %s\n", key)
		}
		fmt.Println()
	}

	if pending, ok := keys["minions_pre"]; ok && len(pending) > 0 {
		color.Yellow("⏳ Pending keys (%d):", len(pending))
		for _, key := range pending {
			fmt.Printf("  • %s\n", key)
		}
		fmt.Println()
		color.Yellow("💡 Accept pending keys with: sloth-kubernetes salt keys accept <minion-id>")
		fmt.Println()
	}

	if rejected, ok := keys["minions_rejected"]; ok && len(rejected) > 0 {
		color.Red("❌ Rejected keys (%d):", len(rejected))
		for _, key := range rejected {
			fmt.Printf("  • %s\n", key)
		}
		fmt.Println()
	}

	if denied, ok := keys["minions_denied"]; ok && len(denied) > 0 {
		color.Red("🚫 Denied keys (%d):", len(denied))
		for _, key := range denied {
			fmt.Printf("  • %s\n", key)
		}
		fmt.Println()
	}

	return nil
}

func runSaltKeysAccept(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	minionID := args[0]

	fmt.Println()
	color.Cyan("🔑 Accepting minion key: %s", minionID)
	fmt.Println()

	if err := client.KeyAccept(minionID); err != nil {
		color.Red("❌ Failed to accept key: %v", err)
		return err
	}

	color.Green("✅ Key accepted successfully")
	fmt.Println()

	return nil
}

// PushConnectionInfo holds connection info for push operations
type PushConnectionInfo struct {
	SSHKeyPath     string
	MasterIP       string
	BastionIP      string
	BastionEnabled bool
}

// getPushConnectionInfo retrieves connection info from stack for push operations
func getPushConnectionInfo() (*PushConnectionInfo, error) {
	targetStack := stackName
	if targetStack == "" {
		return nil, fmt.Errorf("stack name is required for Salt push operations\n\nSpecify a stack with:\n  sloth-kubernetes salt state push <path> --stack <stack-name>")
	}

	ctx := context.Background()

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Use fully qualified stack name
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", targetStack)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		return nil, fmt.Errorf("stack '%s' not found: %w", targetStack, err)
	}

	// Get outputs
	outputs, err := stack.Outputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stack outputs: %w", err)
	}

	info := &PushConnectionInfo{}

	// Get SSH key path
	sshKeyPath := GetSSHKeyPath(targetStack)
	if sshKeyPath == "" {
		return nil, fmt.Errorf("SSH key path not found in stack outputs")
	}
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("SSH key not found at %s. Please ensure the cluster was deployed from this machine", sshKeyPath)
	}
	info.SSHKeyPath = sshKeyPath

	// Get bastion info
	if bastionEnabledOutput, ok := outputs["bastion_enabled"]; ok {
		if bastionEnabledOutput.Value != nil {
			info.BastionEnabled = bastionEnabledOutput.Value == true
		}
	}

	if info.BastionEnabled {
		if bastionOutput, ok := outputs["bastion"]; ok {
			if bastionMap, ok := bastionOutput.Value.(map[string]interface{}); ok {
				if pubIP, ok := bastionMap["public_ip"].(string); ok {
					info.BastionIP = pubIP
				}
			}
		}
	}

	// Check if Salt is enabled and where the master is
	// By default, Salt Master is installed on the bastion host
	saltOnBastion := true // Default: Salt Master is on bastion
	if saltEnabledOutput, ok := outputs["salt_enabled"]; ok {
		if saltEnabledOutput.Value == true {
			// Salt is enabled, check if master is on bastion (default) or separate node
			// For now, assume Salt Master is always on bastion when bastion is enabled
			saltOnBastion = info.BastionEnabled
		}
	}

	if saltOnBastion && info.BastionIP != "" {
		// Salt Master is on bastion - connect directly to bastion
		info.MasterIP = info.BastionIP
		// Clear bastion IP so we don't use ProxyCommand (direct connection)
		info.BastionIP = ""
	} else {
		// Salt Master is on a separate node - find it from nodes
		nodes, _ := ParseNodeOutputs(outputs)
		for _, node := range nodes {
			for _, role := range node.Roles {
				if role == "master" || role == "controlplane" {
					// Prefer WireGuard IP, then private IP, then public IP
					if node.WireGuardIP != "" {
						info.MasterIP = node.WireGuardIP
					} else if node.PrivateIP != "" {
						info.MasterIP = node.PrivateIP
					} else if node.PublicIP != "" {
						info.MasterIP = node.PublicIP
					}
					break
				}
			}
			if info.MasterIP != "" {
				break
			}
		}
	}

	if info.MasterIP == "" {
		return nil, fmt.Errorf("no Salt master found in stack outputs")
	}

	return info, nil
}

func runStatePush(cmd *cobra.Command, args []string) error {
	localPath := args[0]

	// Validate local path exists
	fileInfo, err := os.Stat(localPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", localPath)
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("path must be a directory: %s", localPath)
	}

	// Get Salt API client
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	if pushDryRun {
		color.Yellow("🔍 Dry run - showing what would be transferred:")
	} else {
		color.Cyan("📦 Pushing Salt states via Salt API")
	}
	fmt.Printf("   Source: %s\n", localPath)
	fmt.Printf("   Destination: /srv/salt\n")
	fmt.Println()

	// Configure API push
	cfg := salt.APIPushConfig{
		Client:     client,
		LocalPath:  localPath,
		RemotePath: "/srv/salt",
		Excludes:   pushExcludes,
		DryRun:     pushDryRun,
	}

	// Execute push via API
	result, err := salt.PushDirectoryViaAPI(cfg)
	if err != nil {
		color.Red("❌ Push failed: %v", err)
		return err
	}

	// Show results
	if pushDryRun {
		color.Cyan("📋 Files that would be transferred (%d files, %s):",
			result.FilesTransferred, formatBytes(result.BytesTransferred))
		for _, file := range result.Files {
			fmt.Printf("   • %s\n", file)
		}
	} else {
		color.Green("✅ Transferred %d files (%s) in %s",
			result.FilesTransferred,
			formatBytes(result.BytesTransferred),
			result.Duration.Round(time.Millisecond))

		// Show any errors
		if len(result.Errors) > 0 {
			color.Yellow("⚠️  Some files had errors:")
			for _, e := range result.Errors {
				fmt.Printf("   • %s\n", e)
			}
		}

		// Track in history
		if err := trackSaltPushAPI("state", localPath, "/srv/salt", result, pushApplyAfter); err != nil {
			color.Yellow("⚠️  Failed to save push history: %v", err)
		}
	}
	fmt.Println()

	// Optionally apply highstate
	if pushApplyAfter && !pushDryRun {
		color.Cyan("🔄 Applying highstate...")
		fmt.Println()
		return runSaltHighstate(cmd, []string{})
	}

	return nil
}

func runPillarPush(cmd *cobra.Command, args []string) error {
	localPath := args[0]

	// Validate local path exists
	fileInfo, err := os.Stat(localPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", localPath)
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("path must be a directory: %s", localPath)
	}

	// Get Salt API client
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	if pushDryRun {
		color.Yellow("🔍 Dry run - showing what would be transferred:")
	} else {
		color.Cyan("📦 Pushing Salt pillars via Salt API")
	}
	fmt.Printf("   Source: %s\n", localPath)
	fmt.Printf("   Destination: /srv/pillar\n")
	fmt.Println()

	// Configure API push
	cfg := salt.APIPushConfig{
		Client:     client,
		LocalPath:  localPath,
		RemotePath: "/srv/pillar",
		Excludes:   pushExcludes,
		DryRun:     pushDryRun,
	}

	// Execute push via API
	result, err := salt.PushDirectoryViaAPI(cfg)
	if err != nil {
		color.Red("❌ Push failed: %v", err)
		return err
	}

	// Show results
	if pushDryRun {
		color.Cyan("📋 Files that would be transferred (%d files, %s):",
			result.FilesTransferred, formatBytes(result.BytesTransferred))
		for _, file := range result.Files {
			fmt.Printf("   • %s\n", file)
		}
	} else {
		color.Green("✅ Transferred %d files (%s) in %s",
			result.FilesTransferred,
			formatBytes(result.BytesTransferred),
			result.Duration.Round(time.Millisecond))

		// Show any errors
		if len(result.Errors) > 0 {
			color.Yellow("⚠️  Some files had errors:")
			for _, e := range result.Errors {
				fmt.Printf("   • %s\n", e)
			}
		}

		// Track in history
		if err := trackSaltPushAPI("pillar", localPath, "/srv/pillar", result, pushRefreshAfter); err != nil {
			color.Yellow("⚠️  Failed to save push history: %v", err)
		}
	}
	fmt.Println()

	// Optionally refresh pillars
	if pushRefreshAfter && !pushDryRun {
		color.Cyan("🔄 Refreshing pillars on all minions...")
		fmt.Println()
		return runPillarRefresh(cmd, []string{})
	}

	return nil
}

func runPillarRefresh(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("🔄 Refreshing pillars on all minions...")
	fmt.Println()

	resp, err := client.RunCommand(saltTarget, "saltutil.refresh_pillar", nil)
	if err != nil {
		color.Red("❌ Failed to refresh pillars: %v", err)
		return err
	}

	// Parse and display results
	if saltOutputJSON {
		jsonBytes, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		for minion, result := range resp.Return[0] {
			if result == true {
				color.Green("  ✅ %s: refreshed", minion)
			} else {
				color.Yellow("  ⚠️  %s: %v", minion, result)
			}
		}
	}

	fmt.Println()
	color.Green("✅ Pillar refresh complete")
	return nil
}

func trackSaltPush(pushType, localPath, remotePath string, result *salt.PushResult, applied bool) error {
	targetStack := stackName
	if targetStack == "" {
		return nil // Skip tracking if no stack
	}

	history, err := operations.GetOperationsHistory(targetStack)
	if err != nil {
		return err
	}

	entry := operations.SaltEntry{
		ID:        fmt.Sprintf("push-%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Operation: fmt.Sprintf("%s-push", pushType),
		Target:    remotePath,
		Function:  fmt.Sprintf("push %s -> %s", localPath, remotePath),
		Status:    "success",
		Duration:  result.Duration.String(),
	}

	history.AddSalt(entry)
	return operations.SaveOperationsHistory(targetStack, history)
}

func trackSaltPushAPI(pushType, localPath, remotePath string, result *salt.APIPushResult, applied bool) error {
	targetStack := stackName
	if targetStack == "" {
		return nil // Skip tracking if no stack
	}

	history, err := operations.GetOperationsHistory(targetStack)
	if err != nil {
		return err
	}

	status := "success"
	if len(result.Errors) > 0 {
		status = "partial"
	}

	entry := operations.SaltEntry{
		ID:           fmt.Sprintf("push-%d", time.Now().UnixNano()),
		Timestamp:    time.Now(),
		Operation:    fmt.Sprintf("%s-push-api", pushType),
		Target:       remotePath,
		Function:     fmt.Sprintf("push %s -> %s (%d files)", localPath, remotePath, result.FilesTransferred),
		Status:       status,
		NodesSuccess: result.FilesTransferred - len(result.Errors),
		NodesFailed:  len(result.Errors),
		Duration:     result.Duration.String(),
	}

	history.AddSalt(entry)
	return operations.SaveOperationsHistory(targetStack, history)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
