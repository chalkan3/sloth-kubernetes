package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"

	"github.com/chalkan3/sloth-kubernetes/pkg/operations"
)

var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Manage cluster nodes",
	Long:  `List, add, remove, and manage Kubernetes cluster nodes`,
}

var listNodesCmd = &cobra.Command{
	Use:   "list [stack-name]",
	Short: "List all nodes in the cluster",
	Long:  `Display information about all nodes in the specified stack`,
	Example: `  # List nodes in production stack
  sloth-kubernetes nodes list production

  # List nodes with output format
  sloth-kubernetes nodes list production --output json`,
	RunE: runListNodes,
}

var sshNodeCmd = &cobra.Command{
	Use:   "ssh [stack-name] [node-name]",
	Short: "SSH into a cluster node",
	Long:  `Connect to a cluster node via SSH using the stored private key`,
	Example: `  # SSH into a specific node
  sloth-kubernetes nodes ssh production master-primary-nyc

  # SSH with custom command
  sloth-kubernetes nodes ssh production worker-1 --command "docker ps"`,
	RunE: runSSHNode,
}

var addNodeCmd = &cobra.Command{
	Use:   "add [stack-name]",
	Short: "Add a new node to the cluster",
	Long:  `Add a new node to an existing cluster by updating the stack`,
	Example: `  # Add a node from config file
  sloth-kubernetes nodes add production --config node.lisp

  # Add a node with inline parameters
  sloth-kubernetes nodes add production \
    --name worker-new-1 \
    --provider digitalocean \
    --size s-2vcpu-4gb \
    --role worker`,
	RunE: runAddNode,
}

var removeNodeCmd = &cobra.Command{
	Use:   "remove [stack-name] [node-name]",
	Short: "Remove a node from the cluster",
	Long:  `Safely drain and remove a node from the cluster`,
	Example: `  # Remove a node
  sloth-kubernetes nodes remove production worker-old-1

  # Force remove without draining
  sloth-kubernetes nodes remove production worker-old-1 --force`,
	RunE: runRemoveNode,
}

var (
	nodesOutputFormat string
	sshCommand        string
	forceRemove       bool
	nodeName          string
	nodeProvider      string
	nodeSize          string
	nodeRole          string
)

func init() {
	rootCmd.AddCommand(nodesCmd)

	// Add subcommands
	nodesCmd.AddCommand(listNodesCmd)
	nodesCmd.AddCommand(sshNodeCmd)
	nodesCmd.AddCommand(addNodeCmd)
	nodesCmd.AddCommand(removeNodeCmd)

	// List flags
	listNodesCmd.Flags().StringVar(&nodesOutputFormat, "output", "table", "Output format (table, json, yaml)")

	// SSH flags
	sshNodeCmd.Flags().StringVar(&sshCommand, "command", "", "Command to execute on the node")

	// Add node flags
	addNodeCmd.Flags().StringVar(&nodeName, "name", "", "Node name")
	addNodeCmd.Flags().StringVar(&nodeProvider, "provider", "", "Cloud provider (digitalocean, linode)")
	addNodeCmd.Flags().StringVar(&nodeSize, "size", "", "Node size/type")
	addNodeCmd.Flags().StringVar(&nodeRole, "role", "worker", "Node role (master, worker)")

	// Remove node flags
	removeNodeCmd.Flags().BoolVar(&forceRemove, "force", false, "Force remove without draining")
}

func runListNodes(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get stack name
	stack := getStackFromArgs(args, 0)

	printHeader(fmt.Sprintf("📋 Nodes in stack: %s", stack))

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Use fully qualified stack name for S3 backend
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stack)
	s, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stack, err)
	}

	// Get outputs
	outputs, err := s.Outputs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stack outputs: %w", err)
	}

	// Parse nodes from outputs
	nodes, err := ParseNodeOutputs(outputs)
	if err != nil {
		return fmt.Errorf("failed to parse node outputs: %w", err)
	}

	fmt.Println()
	if nodesOutputFormat == "json" {
		jsonOutput, err := FormatNodesAsJSON(nodes)
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		fmt.Println(jsonOutput)
	} else if nodesOutputFormat == "yaml" {
		yamlOutput, err := FormatNodesAsYAML(nodes)
		if err != nil {
			return fmt.Errorf("failed to format YAML: %w", err)
		}
		fmt.Println(yamlOutput)
	} else {
		// Default: table format
		printNodesTableReal(nodes)
	}

	return nil
}

func runSSHNode(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: sloth-kubernetes nodes ssh <stack-name> <node-name>")
	}

	ctx := context.Background()
	stack := args[0]
	nodeName := args[1]

	printInfo(fmt.Sprintf("🔐 Connecting to node '%s' in stack '%s'...", nodeName, stack))

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Use fully qualified stack name for S3 backend
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stack)
	s, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stack, err)
	}

	// Get outputs
	outputs, err := s.Outputs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stack outputs: %w", err)
	}

	// Parse nodes
	nodes, err := ParseNodeOutputs(outputs)
	if err != nil {
		return fmt.Errorf("failed to parse node outputs: %w", err)
	}

	// Find the requested node
	var targetNode *NodeInfo
	for _, n := range nodes {
		if n.Name == nodeName {
			targetNode = &n
			break
		}
	}

	if targetNode == nil {
		return fmt.Errorf("node '%s' not found in stack '%s'", nodeName, stack)
	}

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

	// Extract SSH key path
	sshKeyPath := GetSSHKeyPath(stack)

	// Determine SSH user based on provider
	// AWS uses "ubuntu", DigitalOcean/Linode use "root"
	sshUser := getSSHUserForProvider(targetNode.Provider)

	// Build SSH command based on bastion mode
	var sshArgs []string

	if bastionEnabled && bastionIP != "" {
		// Bastion mode: Use ProxyCommand to connect through bastion
		printInfo(fmt.Sprintf("🏰 Bastion mode detected - connecting via bastion (%s)", bastionIP))
		printInfo(fmt.Sprintf("   Target: %s (VPN IP: %s)", targetNode.Name, targetNode.WireGuardIP))

		// Use VPN IP for connection (nodes are on private network)
		targetIP := targetNode.WireGuardIP
		if targetIP == "" {
			// Fallback to public IP if VPN IP not available
			targetIP = targetNode.PublicIP
			printInfo("⚠️  VPN IP not available, using public IP")
		}

		// Bastion always uses root (it's a custom image)
		sshArgs = []string{
			"-i", sshKeyPath,
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", fmt.Sprintf("ProxyCommand=ssh -i %s -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=/dev/null -W %%h:%%p root@%s", sshKeyPath, bastionIP),
			fmt.Sprintf("%s@%s", sshUser, targetIP),
		}
	} else {
		// Direct mode: Connect directly to node public IP
		printInfo(fmt.Sprintf("🌍 Direct mode - connecting to %s (%s) as %s", targetNode.Name, targetNode.PublicIP, sshUser))

		sshArgs = []string{
			"-i", sshKeyPath,
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "UserKnownHostsFile=/dev/null",
			fmt.Sprintf("%s@%s", sshUser, targetNode.PublicIP),
		}
	}

	// Add custom command if specified
	if sshCommand != "" {
		sshArgs = append(sshArgs, sshCommand)
	}

	// Execute SSH using exec.Command for full interactive session
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh command not found: %w", err)
	}

	// If no custom command, start interactive SSH session
	if sshCommand == "" {
		printInfo("Starting interactive SSH session...")
		fmt.Println()
	}

	// Execute SSH with full I/O redirection
	execCmd := exec.Command(sshPath, sshArgs...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}

// getSSHUserForProvider returns the appropriate SSH user for a cloud provider
func getSSHUserForProvider(provider string) string {
	switch provider {
	case "aws":
		return "ubuntu" // AWS Ubuntu AMIs use "ubuntu" user
	case "azure":
		return "azureuser" // Azure default user
	case "gcp":
		return "ubuntu" // GCP Ubuntu images use "ubuntu" user
	case "digitalocean", "linode":
		return "root" // DigitalOcean and Linode use root by default
	default:
		return "root" // Default to root for unknown providers
	}
}

func runAddNode(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes nodes add <stack-name> --pool <pool-name>")
	}

	stack := args[0]

	// Validate required flags
	if nodeName == "" {
		return fmt.Errorf("--pool flag is required (specify which node pool to scale)")
	}
	poolName := nodeName // Reusing the nodeName flag for pool name

	printHeader(fmt.Sprintf("➕ Adding node to stack: %s", stack))

	// Get the config file path
	configFile := cfgFile
	if configFile == "" {
		configFile = "./cluster-config.lisp"
	}

	printInfo(fmt.Sprintf("📄 Reading configuration from: %s", configFile))

	// Read the current config file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse config (Note: This needs to be updated to use Lisp parser)
	var config map[string]interface{}
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Find and update the node pool
	nodePools, ok := config["nodePools"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("nodePools not found in configuration")
	}

	pool, ok := nodePools[poolName].(map[string]interface{})
	if !ok {
		return fmt.Errorf("node pool '%s' not found in configuration", poolName)
	}

	// Get current count
	currentCount, ok := pool["count"].(int)
	if !ok {
		return fmt.Errorf("count field not found or invalid in pool '%s'", poolName)
	}

	// Calculate new count
	addCount := 1
	if nodeSize != "" {
		// If --count flag is provided, use it (reusing nodeSize flag for count)
		fmt.Sscanf(nodeSize, "%d", &addCount)
	}
	newCount := currentCount + addCount

	printInfo(fmt.Sprintf("📊 Current node count in pool '%s': %d", poolName, currentCount))
	printInfo(fmt.Sprintf("➕ Adding %d node(s)", addCount))
	printInfo(fmt.Sprintf("📈 New node count: %d", newCount))

	// Update the count
	pool["count"] = newCount

	// Save the updated config to a temporary file
	tempConfig := fmt.Sprintf("%s.add-node.tmp", configFile)
	updatedData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	if err := os.WriteFile(tempConfig, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write temporary config: %w", err)
	}

	printInfo(fmt.Sprintf("💾 Saved updated configuration to: %s", tempConfig))
	fmt.Println()

	color.Cyan("🚀 Running deployment to add node(s)...")
	color.Yellow("⚠️  This will provision new infrastructure. Press Ctrl+C to cancel.")
	fmt.Println()

	// TODO: Call the deploy command programmatically
	// For now, show the command the user should run
	color.Green("✅ Configuration updated successfully!")
	fmt.Println()
	color.Cyan("To apply the changes, run:")
	fmt.Printf("  ./sloth-kubernetes deploy %s --config %s --do-token $DIGITALOCEAN_TOKEN --linode-token $LINODE_TOKEN -y\n", stack, tempConfig)
	fmt.Println()
	color.Yellow("💡 Tip: After deployment completes, you can delete the temporary config file:")
	fmt.Printf("  rm %s\n", tempConfig)

	// Record the operation
	details := fmt.Sprintf("Pool: %s, Adding %d node(s), New count: %d", poolName, addCount, newCount)
	operations.RecordNodeOperation(stack, "add", poolName, nodeRole, "", "success", details, time.Since(startTime), nil)

	return nil
}

func runRemoveNode(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	if len(args) < 2 {
		return fmt.Errorf("usage: sloth-kubernetes nodes remove <stack-name> <node-name>")
	}

	stack := args[0]
	node := args[1]

	printHeader(fmt.Sprintf("➖ Removing node '%s' from stack: %s", node, stack))

	if !forceRemove {
		color.Yellow("⚠️  Node will be drained before removal")
	} else {
		color.Red("⚠️  Force removal - node will NOT be drained!")
	}

	color.Yellow("⚠️  Remove node functionality will be implemented in next phase")

	// Record the operation (placeholder for now)
	details := "Remove functionality pending implementation"
	if forceRemove {
		details = "Force remove - " + details
	}
	operations.RecordNodeOperation(stack, "remove", node, "", "", "pending", details, time.Since(startTime), nil)

	return nil
}

func printNodesTable(outputs auto.OutputMap) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	// Header
	color.New(color.Bold).Fprintln(w, "NAME\tPROVIDER\tREGION\tSIZE\tROLES\tPUBLIC IP\tVPN IP\tSTATUS")
	fmt.Fprintln(w, "----\t--------\t------\t----\t-----\t---------\t------\t------")

	// TODO: Parse actual node data from outputs
	// For now, show placeholder
	fmt.Fprintln(w, "master-1\tdigitalocean\tnyc3\ts-2vcpu-4gb\tmaster\t167.71.1.1\t10.8.0.10\t✅ Running")
	fmt.Fprintln(w, "worker-1\tlinode\tus-ord\tg6-standard-2\tworker\t172.236.1.1\t10.8.0.11\t✅ Running")

	color.Yellow("\n⚠️  Full node information will be available after implementing output parsing")
}

func printNodesTableReal(nodes []NodeInfo) {
	if len(nodes) == 0 {
		color.Yellow("⚠️  No nodes found in stack outputs")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	// Header
	color.New(color.Bold).Fprintln(w, "NAME\tPROVIDER\tREGION\tSIZE\tROLES\tPUBLIC IP\tVPN IP\tSTATUS")
	fmt.Fprintln(w, "----\t--------\t------\t----\t-----\t---------\t------\t------")

	for _, node := range nodes {
		// Format roles
		rolesStr := "unknown"
		if len(node.Roles) > 0 {
			rolesStr = node.Roles[0]
			if len(node.Roles) > 1 {
				rolesStr += fmt.Sprintf(" +%d", len(node.Roles)-1)
			}
		}

		// Status icon
		statusIcon := "✅"
		if node.Status != "" && node.Status != "active" && node.Status != "running" {
			statusIcon = "⚠️"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s %s\n",
			node.Name,
			node.Provider,
			node.Region,
			node.Size,
			rolesStr,
			node.PublicIP,
			node.WireGuardIP,
			statusIcon,
			node.Status,
		)
	}
}

func getStackFromArgs(args []string, index int) string {
	if len(args) > index {
		return args[index]
	}
	if stackName != "" {
		return stackName
	}
	return "production"
}
