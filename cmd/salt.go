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
	saltAPIURL     string
	saltUsername   string
	saltPassword   string
	saltTarget     string
	saltOutputJSON bool
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

	// State subcommands
	saltStateCmd.AddCommand(saltStateApplyCmd)
	saltStateCmd.AddCommand(saltStateHighstateCmd)

	// Keys subcommands
	keysCmd.AddCommand(keysListCmd)
	keysCmd.AddCommand(keysAcceptCmd)

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
	// If no URL provided, try to auto-login from stack
	if saltAPIURL == "" {
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

	client := salt.NewClient(saltAPIURL, saltUsername, saltPassword)
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
		targetStack = "production"
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

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
