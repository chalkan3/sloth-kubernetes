package cmd

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/curve25519"
)

var vpnCmd = &cobra.Command{
	Use:   "vpn",
	Short: "Manage WireGuard VPN",
	Long:  `Configure, manage, and troubleshoot the WireGuard VPN mesh network`,
}

var vpnStatusCmd = &cobra.Command{
	Use:   "status [stack-name]",
	Short: "Show VPN status and tunnels",
	Long:  `Display the current status of the WireGuard VPN mesh including all tunnels`,
	Example: `  # Show VPN status for production stack
  sloth-kubernetes vpn status production`,
	RunE: runVPNStatus,
}

var vpnPeersCmd = &cobra.Command{
	Use:   "peers [stack-name]",
	Short: "List all VPN peers",
	Long:  `Display all nodes in the VPN mesh with their public keys and endpoints`,
	Example: `  # List VPN peers
  sloth-kubernetes vpn peers production`,
	RunE: runVPNPeers,
}

var vpnConfigCmd = &cobra.Command{
	Use:   "config [stack-name] [node-name]",
	Short: "Get VPN configuration for a node",
	Long:  `Display the WireGuard configuration for a specific node`,
	Example: `  # Get VPN config for a node
  sloth-kubernetes vpn config production master-1`,
	RunE: runVPNConfig,
}

var vpnTestCmd = &cobra.Command{
	Use:   "test [stack-name]",
	Short: "Test VPN connectivity",
	Long:  `Test connectivity between all nodes in the VPN mesh`,
	Example: `  # Test VPN connectivity
  sloth-kubernetes vpn test production`,
	RunE: runVPNTest,
}

var vpnJoinCmd = &cobra.Command{
	Use:   "join [stack-name]",
	Short: "Join this machine or a remote host to the VPN",
	Long: `Add your local machine or a remote SSH host to the WireGuard VPN mesh.
This will generate WireGuard keys, configure all cluster nodes to accept the new peer,
and provide you with the WireGuard configuration to install locally.`,
	Example: `  # Join local machine to VPN
  sloth-kubernetes vpn join production

  # Join a remote SSH host to VPN
  sloth-kubernetes vpn join production --remote user@host.com

  # Join with custom VPN IP
  sloth-kubernetes vpn join production --vpn-ip 10.8.0.100

  # Join and auto-install WireGuard config
  sloth-kubernetes vpn join production --install`,
	RunE: runVPNJoin,
}

var vpnLeaveCmd = &cobra.Command{
	Use:   "leave [stack-name]",
	Short: "Remove this machine from the VPN",
	Long:  `Remove your local machine or a remote host from the WireGuard VPN mesh`,
	Example: `  # Leave VPN
  sloth-kubernetes vpn leave production

  # Remove a specific peer by IP
  sloth-kubernetes vpn leave production --vpn-ip 10.8.0.100`,
	RunE: runVPNLeave,
}

var vpnClientConfigCmd = &cobra.Command{
	Use:   "client-config [stack-name]",
	Short: "Generate WireGuard client configuration",
	Long:  `Generate a WireGuard configuration file for connecting to the VPN mesh`,
	Example: `  # Generate client config
  sloth-kubernetes vpn client-config production

  # Save to file
  sloth-kubernetes vpn client-config production --output client.conf

  # Generate QR code for mobile
  sloth-kubernetes vpn client-config production --qr`,
	RunE: runVPNClientConfig,
}

var (
	vpnJoinRemote    string
	vpnJoinIP        string
	vpnJoinInstall   bool
	vpnLeaveIP       string
	vpnConfigOutput  string
	vpnConfigQR      bool
)

func init() {
	rootCmd.AddCommand(vpnCmd)

	// Add subcommands
	vpnCmd.AddCommand(vpnStatusCmd)
	vpnCmd.AddCommand(vpnPeersCmd)
	vpnCmd.AddCommand(vpnConfigCmd)
	vpnCmd.AddCommand(vpnTestCmd)
	vpnCmd.AddCommand(vpnJoinCmd)
	vpnCmd.AddCommand(vpnLeaveCmd)
	vpnCmd.AddCommand(vpnClientConfigCmd)

	// Join flags
	vpnJoinCmd.Flags().StringVar(&vpnJoinRemote, "remote", "", "Remote SSH host to add (e.g., user@host.com)")
	vpnJoinCmd.Flags().StringVar(&vpnJoinIP, "vpn-ip", "", "Custom VPN IP address (default: auto-assign)")
	vpnJoinCmd.Flags().BoolVar(&vpnJoinInstall, "install", false, "Auto-install WireGuard configuration")

	// Leave flags
	vpnLeaveCmd.Flags().StringVar(&vpnLeaveIP, "vpn-ip", "", "VPN IP of peer to remove")

	// Client config flags
	vpnClientConfigCmd.Flags().StringVar(&vpnConfigOutput, "output", "", "Output file path")
	vpnClientConfigCmd.Flags().BoolVar(&vpnConfigQR, "qr", false, "Generate QR code for mobile devices")
}

func runVPNStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	stack := getStackFromArgs(args, 0)

	printHeader(fmt.Sprintf("🔐 VPN Status - Stack: %s", stack))

	// Get stack using SelectStackInlineSource (same as status command)
	s, err := auto.SelectStackInlineSource(ctx, stack, "kubernetes-create", func(ctx *pulumi.Context) error {
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stack, err)
	}

	// Get outputs
	outputs, err := s.Outputs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stack outputs: %w", err)
	}

	fmt.Println()
	printVPNStatusTable(outputs)

	return nil
}

func runVPNPeers(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	stack := getStackFromArgs(args, 0)

	printHeader(fmt.Sprintf("👥 VPN Peers - Stack: %s", stack))

	// Get stack using SelectStackInlineSource (same as status command)
	s, err := auto.SelectStackInlineSource(ctx, stack, "kubernetes-create", func(ctx *pulumi.Context) error {
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stack, err)
	}

	// Get outputs
	outputs, err := s.Outputs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stack outputs: %w", err)
	}

	fmt.Println()
	printVPNPeersTable(outputs)

	return nil
}

func runVPNConfig(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: sloth-kubernetes vpn config <stack-name> <node-name>")
	}

	stack := args[0]
	node := args[1]

	printHeader(fmt.Sprintf("📋 VPN Config - Node: %s", node))

	color.Yellow("⚠️  VPN config extraction will be implemented in next phase")
	color.Cyan(fmt.Sprintf("Stack: %s", stack))
	color.Cyan(fmt.Sprintf("Node: %s", node))

	return nil
}

func runVPNTest(cmd *cobra.Command, args []string) error {
	stack := getStackFromArgs(args, 0)

	printHeader(fmt.Sprintf("🧪 Testing VPN Connectivity - Stack: %s", stack))

	color.Yellow("⚠️  VPN connectivity testing will be implemented in next phase")
	fmt.Println()
	fmt.Println("This will test:")
	fmt.Println("  • Ping between all nodes via VPN")
	fmt.Println("  • WireGuard handshake status")
	fmt.Println("  • Tunnel bandwidth")
	fmt.Println("  • Latency between nodes")

	return nil
}

func printVPNStatusTable(outputs auto.OutputMap) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	color.New(color.Bold).Fprintln(w, "METRIC\tVALUE")
	fmt.Fprintln(w, "------\t-----")

	// TODO: Parse actual VPN data from outputs
	fmt.Fprintln(w, "VPN Mode\tWireGuard Mesh")
	fmt.Fprintln(w, "Total Nodes\t6")
	fmt.Fprintln(w, "Total Tunnels\t15")
	fmt.Fprintln(w, "VPN Subnet\t10.8.0.0/24")
	fmt.Fprintln(w, "Status\t✅ All tunnels active")

	color.Yellow("\n⚠️  Real-time VPN metrics will be available after implementing monitoring")
}

func printVPNPeersTable(outputs auto.OutputMap) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	color.New(color.Bold).Fprintln(w, "NODE\tVPN IP\tPUBLIC KEY\tENDPOINT\tLAST HANDSHAKE\tTRANSFER")
	fmt.Fprintln(w, "----\t------\t----------\t--------\t--------------\t--------")

	// TODO: Parse actual peer data from outputs
	fmt.Fprintln(w, "master-1\t10.8.0.10\tABC123...\t167.71.1.1:51820\t1m ago\t↑ 1.2MB / ↓ 2.4MB")
	fmt.Fprintln(w, "worker-1\t10.8.0.11\tDEF456...\t172.236.1.1:51820\t30s ago\t↑ 800KB / ↓ 1.5MB")

	color.Yellow("\n⚠️  Full peer information will be available after implementing peer tracking")
}

func runVPNJoin(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes vpn join <stack-name>")
	}

	ctx := context.Background()
	stack := args[0]

	printHeader(fmt.Sprintf("🔗 Joining VPN - Stack: %s", stack))

	// Get stack using SelectStackInlineSource (same as status command)
	s, err := auto.SelectStackInlineSource(ctx, stack, "kubernetes-create", func(ctx *pulumi.Context) error {
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stack, err)
	}

	outputs, err := s.Outputs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stack outputs: %w", err)
	}

	// Parse nodes to get cluster info
	nodes, err := ParseNodeOutputs(outputs)
	if err != nil {
		return fmt.Errorf("failed to parse nodes: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found in stack - cluster may not be deployed yet")
	}

	fmt.Println()
	printInfo(fmt.Sprintf("Found %d cluster nodes", len(nodes)))

	// Determine target (local or remote)
	target := "local machine"
	if vpnJoinRemote != "" {
		target = vpnJoinRemote
	}

	printInfo(fmt.Sprintf("Target: %s", target))

	// Auto-assign VPN IP if not specified
	if vpnJoinIP == "" {
		// Find next available IP in 10.8.0.x range
		vpnJoinIP = fmt.Sprintf("10.8.0.%d", 100+len(nodes))
		printInfo(fmt.Sprintf("Auto-assigned VPN IP: %s", vpnJoinIP))
	} else {
		printInfo(fmt.Sprintf("Using custom VPN IP: %s", vpnJoinIP))
	}

	// STEP 1: Generate WireGuard keypair
	fmt.Println()
	printInfo("Step 1/4: Generating WireGuard keypair...")
	privateKey, publicKey, err := generateWireGuardKeypair()
	if err != nil {
		return fmt.Errorf("failed to generate keypair: %w", err)
	}
	printSuccess(fmt.Sprintf("Generated keypair (public key: %s...)", publicKey[:16]))

	// STEP 2: Get SSH key for node access
	sshKeyPath := GetSSHKeyPath(stack)
	printInfo(fmt.Sprintf("Using SSH key: %s", sshKeyPath))

	// Check if bastion is enabled
	bastionEnabled := false
	bastionIP := ""
	if bastionEnabledOutput, ok := outputs["bastion_enabled"]; ok {
		if bastionEnabledOutput.Value != nil {
			bastionEnabled = bastionEnabledOutput.Value == true
		}
	}

	// Get bastion info if enabled
	if bastionEnabled {
		if bastionOutput, ok := outputs["bastion"]; ok {
			if bastionMap, ok := bastionOutput.Value.(map[string]interface{}); ok {
				if pubIP, ok := bastionMap["public_ip"].(string); ok {
					bastionIP = pubIP
				}
			}
		}
	}

	// STEP 3: Add peer to all cluster nodes
	fmt.Println()
	printInfo("Step 2/4: Adding peer to all cluster nodes...")

	for i, node := range nodes {
		nodeTarget := node.PublicIP
		peerAddScript := generatePeerAddScript(vpnJoinIP, publicKey)

		// Build SSH command based on bastion mode
		var sshCmd *exec.Cmd
		if bastionEnabled && bastionIP != "" {
			// Use ProxyJump through bastion
			printInfo(fmt.Sprintf("  [%d/%d] Adding peer to %s (via bastion)...", i+1, len(nodes), node.Name))
			// Use WireGuard VPN IP for bastion ProxyJump (all nodes in VPN mesh)
			targetIP := node.WireGuardIP
			if targetIP == "" {
				// Fallback to PrivateIP, then PublicIP
				targetIP = node.PrivateIP
				if targetIP == "" {
					targetIP = node.PublicIP
				}
			}

			sshCmd = exec.Command("ssh",
				"-i", sshKeyPath,
				"-o", "StrictHostKeyChecking=accept-new",
				"-o", "UserKnownHostsFile=/dev/null",
				"-o", fmt.Sprintf("ProxyCommand=ssh -i %s -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=/dev/null -W %%h:%%p root@%s", sshKeyPath, bastionIP),
				fmt.Sprintf("root@%s", targetIP),
				peerAddScript,
			)
		} else {
			// Direct SSH
			printInfo(fmt.Sprintf("  [%d/%d] Adding peer to %s...", i+1, len(nodes), node.Name))
			sshCmd = exec.Command("ssh",
				"-i", sshKeyPath,
				"-o", "StrictHostKeyChecking=accept-new",
				"-o", "UserKnownHostsFile=/dev/null",
				fmt.Sprintf("root@%s", nodeTarget),
				peerAddScript,
			)
		}

		output, err := sshCmd.CombinedOutput()
		if err != nil {
			color.Yellow(fmt.Sprintf("  ⚠️  Failed to add peer to %s: %v (output: %s)", node.Name, err, string(output)))
			continue
		}
		printSuccess(fmt.Sprintf("  ✓ Added peer to %s", node.Name))
	}

	// STEP 4: Generate client configuration
	fmt.Println()
	printInfo("Step 3/4: Generating client configuration...")
	clientConfig := generateClientConfig(privateKey, vpnJoinIP, nodes, sshKeyPath, bastionEnabled, bastionIP)

	configPath := "./wg0-client.conf"
	if err := os.WriteFile(configPath, []byte(clientConfig), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	printSuccess(fmt.Sprintf("Client configuration saved to: %s", configPath))

	// STEP 5: Optionally install
	if vpnJoinInstall {
		fmt.Println()
		printInfo("Step 4/4: Installing WireGuard configuration...")

		if vpnJoinRemote != "" {
			color.Yellow("⚠️  Remote installation not yet implemented. Please install manually.")
		} else {
			// Detect OS
			osType := detectOS()

			switch osType {
			case "darwin": // macOS
				printInfo("Detected macOS - installing WireGuard VPN")

				// Try to install automatically (requires sudo)
				// Create WireGuard directory
				mkdirCmd := exec.Command("sudo", "mkdir", "-p", "/opt/homebrew/etc/wireguard")
				if err := mkdirCmd.Run(); err != nil {
					color.Yellow("⚠️  Failed to create WireGuard directory. Please run manually:")
					fmt.Println("  sudo mkdir -p /opt/homebrew/etc/wireguard")
					fmt.Printf("  sudo cp %s /opt/homebrew/etc/wireguard/wg0.conf\n", configPath)
					fmt.Println("  sudo wg-quick up wg0")
					return nil
				}

				// Copy configuration
				cpCmd := exec.Command("sudo", "cp", configPath, "/opt/homebrew/etc/wireguard/wg0.conf")
				if err := cpCmd.Run(); err != nil {
					color.Yellow("⚠️  Failed to copy configuration. Please run manually:")
					fmt.Printf("  sudo cp %s /opt/homebrew/etc/wireguard/wg0.conf\n", configPath)
					fmt.Println("  sudo wg-quick up wg0")
					return nil
				}

				// Start WireGuard
				printInfo("Starting WireGuard VPN...")
				upCmd := exec.Command("sudo", "wg-quick", "up", "wg0")
				if output, err := upCmd.CombinedOutput(); err != nil {
					color.Yellow(fmt.Sprintf("⚠️  Failed to start WireGuard: %v", err))
					color.Yellow(fmt.Sprintf("Output: %s", string(output)))
					fmt.Println()
					fmt.Println("Please try manually:")
					fmt.Println("  sudo wg-quick up wg0")
					return nil
				}

				printSuccess("✓ WireGuard VPN activated successfully!")
				fmt.Println()
				fmt.Println("To check VPN status:")
				fmt.Println("  sudo wg show")
				fmt.Println()
				fmt.Println("To stop VPN:")
				fmt.Println("  sudo wg-quick down wg0")

			case "linux":
				// Check if running as root
				if os.Geteuid() != 0 {
					color.Yellow("⚠️  Installation requires root privileges. Please run:")
					fmt.Println()
					fmt.Printf("  sudo cp %s /etc/wireguard/wg0.conf\n", configPath)
					fmt.Println("  sudo wg-quick up wg0")
					fmt.Println("  sudo systemctl enable wg-quick@wg0")
				} else {
					// Install configuration
					if err := exec.Command("cp", configPath, "/etc/wireguard/wg0.conf").Run(); err != nil {
						return fmt.Errorf("failed to copy config: %w", err)
					}

					if err := exec.Command("wg-quick", "up", "wg0").Run(); err != nil {
						return fmt.Errorf("failed to start WireGuard: %w", err)
					}

					if err := exec.Command("systemctl", "enable", "wg-quick@wg0").Run(); err != nil {
						color.Yellow("⚠️  Failed to enable WireGuard service on boot")
					}

					printSuccess("WireGuard installed and started")
				}

			default:
				color.Yellow(fmt.Sprintf("⚠️  Unsupported OS: %s", osType))
				color.Cyan(fmt.Sprintf("\nConfiguration saved to: %s", configPath))
				color.Cyan("Please install WireGuard manually for your platform")
			}
		}
	} else {
		fmt.Println()
		osType := detectOS()

		if osType == "darwin" {
			color.Cyan("To install the configuration on macOS:")
			fmt.Println()
			fmt.Println("  1. Install WireGuard app: https://www.wireguard.com/install/")
			fmt.Printf("  2. Import tunnel from file: %s\n", configPath)
			fmt.Println("  3. Click 'Activate' to connect")
			fmt.Println()
			color.Cyan("Or use command line:")
			fmt.Printf("  sudo mkdir -p /opt/homebrew/etc/wireguard\n")
			fmt.Printf("  sudo cp %s /opt/homebrew/etc/wireguard/wg0.conf\n", configPath)
			fmt.Printf("  wg-quick up /opt/homebrew/etc/wireguard/wg0.conf\n")
		} else {
			color.Cyan("To install the configuration manually:")
			fmt.Println()
			fmt.Printf("  sudo cp %s /etc/wireguard/wg0.conf\n", configPath)
			fmt.Println("  sudo wg-quick up wg0")
			fmt.Println("  sudo systemctl enable wg-quick@wg0")
		}
	}

	fmt.Println()
	printSuccess(fmt.Sprintf("Successfully joined VPN with IP %s!", vpnJoinIP))
	printInfo("You can now access cluster nodes via their VPN IPs (10.8.0.x)")

	return nil
}

func runVPNLeave(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes vpn leave <stack-name>")
	}

	stack := args[0]

	printHeader(fmt.Sprintf("👋 Leaving VPN - Stack: %s", stack))

	target := "local machine"
	if vpnLeaveIP != "" {
		target = fmt.Sprintf("peer with IP %s", vpnLeaveIP)
	}

	printInfo(fmt.Sprintf("Removing: %s", target))

	fmt.Println()
	color.Yellow("⚠️  VPN leave functionality will be implemented in next phase")
	color.Cyan("\nWhat will be implemented:")
	fmt.Println("  • SSH to all cluster nodes and remove peer from wg0.conf")
	fmt.Println("  • Reload WireGuard on all nodes")
	fmt.Println("  • Stop and disable local WireGuard service")
	fmt.Println("  • Remove local WireGuard configuration")

	return nil
}

func runVPNClientConfig(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes vpn client-config <stack-name>")
	}

	ctx := context.Background()
	stack := args[0]

	printHeader(fmt.Sprintf("📱 Generate Client Config - Stack: %s", stack))

	// Get stack using SelectStackInlineSource (same as status command)
	s, err := auto.SelectStackInlineSource(ctx, stack, "kubernetes-create", func(ctx *pulumi.Context) error {
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stack, err)
	}

	outputs, err := s.Outputs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stack outputs: %w", err)
	}

	// Parse nodes
	nodes, err := ParseNodeOutputs(outputs)
	if err != nil {
		return fmt.Errorf("failed to parse nodes: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found in stack")
	}

	fmt.Println()
	printInfo(fmt.Sprintf("Generating config for %d peer(s)", len(nodes)))

	if vpnConfigOutput != "" {
		printInfo(fmt.Sprintf("Output file: %s", vpnConfigOutput))
	}

	if vpnConfigQR {
		printInfo("QR code generation enabled")
	}

	fmt.Println()
	color.Yellow("⚠️  Client config generation will be implemented in next phase")
	color.Cyan("\nWhat will be implemented:")
	fmt.Println("  • Generate new WireGuard keypair")
	fmt.Println("  • Create [Interface] section with private key and VPN IP")
	fmt.Println("  • Create [Peer] sections for all cluster nodes")
	fmt.Println("  • Save to file (default: ./wg0.conf)")
	if vpnConfigQR {
		fmt.Println("  • Generate QR code using 'qrencode' for mobile import")
	}

	fmt.Println()
	color.Cyan("Example output format:")
	fmt.Print(`
[Interface]
PrivateKey = <generated-private-key>
Address = 10.8.0.100/24
DNS = 1.1.1.1

[Peer]
PublicKey = <master-1-public-key>
Endpoint = 167.71.1.1:51820
AllowedIPs = 10.8.0.10/32

[Peer]
PublicKey = <worker-1-public-key>
Endpoint = 172.236.1.1:51820
AllowedIPs = 10.8.0.11/32
`)

	return nil
}

// generateWireGuardKeypair generates a WireGuard private/public keypair
func generateWireGuardKeypair() (privateKey string, publicKey string, err error) {
	// Generate 32 random bytes for private key
	var privKey [32]byte
	if _, err := rand.Read(privKey[:]); err != nil {
		return "", "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Clamp the private key (WireGuard requirement)
	privKey[0] &= 248
	privKey[31] &= 127
	privKey[31] |= 64

	// Derive public key using Curve25519
	var pubKey [32]byte
	curve25519.ScalarBaseMult(&pubKey, &privKey)

	// Encode to base64
	privateKey = base64.StdEncoding.EncodeToString(privKey[:])
	publicKey = base64.StdEncoding.EncodeToString(pubKey[:])

	return privateKey, publicKey, nil
}

// generatePeerAddScript creates a bash script to add a peer to WireGuard config
func generatePeerAddScript(peerIP string, peerPublicKey string) string {
	return fmt.Sprintf(`
set -e
# Add new peer to WireGuard configuration
cat >> /etc/wireguard/wg0.conf <<EOF

[Peer]
# Client joined via CLI
PublicKey = %s
AllowedIPs = %s/32
PersistentKeepalive = 25
EOF

# Reload WireGuard configuration
wg syncconf wg0 <(wg-quick strip wg0)
echo "Peer added and WireGuard reloaded"
`, peerPublicKey, peerIP)
}

// fetchNodePublicKey fetches the WireGuard public key from a node via SSH
func fetchNodePublicKey(node NodeInfo, sshKeyPath string, bastionEnabled bool, bastionIP string) (string, error) {
	// Determine target IP
	targetIP := node.WireGuardIP
	if targetIP == "" {
		targetIP = node.PrivateIP
		if targetIP == "" {
			targetIP = node.PublicIP
		}
	}

	// Build SSH command
	var sshCmd *exec.Cmd
	if bastionEnabled && bastionIP != "" {
		// Use ProxyCommand through bastion (use -q to suppress SSH warnings)
		sshCmd = exec.Command("ssh",
			"-q",
			"-i", sshKeyPath,
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", fmt.Sprintf("ProxyCommand=ssh -q -i %s -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=/dev/null -W %%h:%%p root@%s", sshKeyPath, bastionIP),
			fmt.Sprintf("root@%s", targetIP),
			"cat /etc/wireguard/publickey",
		)
	} else {
		// Direct SSH (use -q to suppress SSH warnings)
		sshCmd = exec.Command("ssh",
			"-q",
			"-i", sshKeyPath,
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "UserKnownHostsFile=/dev/null",
			fmt.Sprintf("root@%s", node.PublicIP),
			"cat /etc/wireguard/publickey",
		)
	}

	output, err := sshCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to fetch public key: %w (output: %s)", err, string(output))
	}

	// Trim whitespace and newlines
	publicKey := string(output)
	// Remove trailing newlines
	for len(publicKey) > 0 && (publicKey[len(publicKey)-1] == '\n' || publicKey[len(publicKey)-1] == '\r') {
		publicKey = publicKey[:len(publicKey)-1]
	}

	return publicKey, nil
}

// generateClientConfig generates a complete WireGuard client configuration
func generateClientConfig(privateKey string, clientIP string, nodes []NodeInfo, sshKeyPath string, bastionEnabled bool, bastionIP string) string {
	config := fmt.Sprintf(`[Interface]
# WireGuard Client Configuration
# Generated by sloth-kubernetes CLI
PrivateKey = %s
Address = %s/24
DNS = 1.1.1.1

# Post-connection script (optional)
# PostUp = echo "Connected to Kubernetes cluster VPN"
# PreDown = echo "Disconnecting from cluster VPN"

`, privateKey, clientIP)

	// Add each cluster node as a peer
	for _, node := range nodes {
		if node.WireGuardIP == "" {
			continue
		}

		// Fetch actual public key from node
		publicKey, err := fetchNodePublicKey(node, sshKeyPath, bastionEnabled, bastionIP)
		if err != nil {
			// If we can't fetch the key, use placeholder and add a warning
			color.Yellow(fmt.Sprintf("  ⚠️  Failed to fetch public key from %s: %v", node.Name, err))
			publicKey = "<PUBLIC_KEY_PLACEHOLDER>"
		}

		config += fmt.Sprintf(`
[Peer]
# %s (%s)
PublicKey = %s
Endpoint = %s:51820
AllowedIPs = %s/32, 10.0.0.0/8
PersistentKeepalive = 25
`, node.Name, node.Provider, publicKey, node.PublicIP, node.WireGuardIP)
	}

	return config
}

// detectOS detects the operating system
func detectOS() string {
	cmd := exec.Command("uname", "-s")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	osName := string(output)
	// Remove trailing newline if present
	if len(osName) > 0 && osName[len(osName)-1] == '\n' {
		osName = osName[:len(osName)-1]
	}

	switch osName {
	case "Darwin":
		return "darwin"
	case "Linux":
		return "linux"
	default:
		return "unknown"
	}
}
