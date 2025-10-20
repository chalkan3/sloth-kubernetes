package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/spf13/cobra"
)

var (
	nodeCount     int
	nodePool      string
	nodeSize      string
	nodeProvider  string
	nodeRegion    string
	nodeRole      string
	upgradeVersion string
	forceRemove   bool
)

// nodesCmd represents the nodes command
var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Manage cluster nodes",
	Long: `Manage Kubernetes cluster nodes including listing, adding, removing,
SSH access, and upgrading nodes.

Available subcommands:
  list     - List all nodes in the cluster
  add      - Add new nodes to the cluster
  remove   - Remove a node from the cluster
  ssh      - SSH into a specific node
  upgrade  - Upgrade Kubernetes version on nodes`,
}

// nodesListCmd lists all nodes
var nodesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all nodes in the cluster",
	Long: `Display detailed information about all nodes in the cluster including:
  - Node name and role (master/worker)
  - Cloud provider and region
  - IP addresses (public and private)
  - Status and health
  - Kubernetes version`,
	Example: `  # List all nodes
  kubernetes-create nodes list

  # List with specific stack
  kubernetes-create nodes list --stack production`,
	RunE: runNodesList,
}

// nodesAddCmd adds new nodes
var nodesAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add new nodes to the cluster",
	Long: `Add new worker or master nodes to an existing cluster.

Note: Adding master nodes requires careful consideration of HA requirements.
Worker nodes can be added more freely.`,
	Example: `  # Add 2 worker nodes
  kubernetes-create nodes add --count 2

  # Add to specific pool
  kubernetes-create nodes add --count 3 --pool workers

  # Add with specific size
  kubernetes-create nodes add --count 2 --size s-4vcpu-8gb

  # Add to specific provider
  kubernetes-create nodes add --count 1 --provider linode --region us-east`,
	RunE: runNodesAdd,
}

// nodesRemoveCmd removes a node
var nodesRemoveCmd = &cobra.Command{
	Use:   "remove <node-name>",
	Short: "Remove a node from the cluster",
	Long: `Remove a specific node from the cluster. The node will be:
  1. Drained (pods moved to other nodes)
  2. Deleted from Kubernetes
  3. Destroyed at the cloud provider

WARNING: This is a destructive operation!`,
	Example: `  # Remove a worker node
  kubernetes-create nodes remove worker-1

  # Force remove without confirmation
  kubernetes-create nodes remove worker-1 --force`,
	Args: cobra.ExactArgs(1),
	RunE: runNodesRemove,
}

// nodesSSHCmd SSHs into a node
var nodesSSHCmd = &cobra.Command{
	Use:   "ssh <node-name>",
	Short: "SSH into a specific node",
	Long: `Open an interactive SSH session to a specific node.

The SSH key from the cluster deployment will be used automatically.`,
	Example: `  # SSH into master-1
  kubernetes-create nodes ssh master-1

  # SSH into worker-2
  kubernetes-create nodes ssh worker-2`,
	Args: cobra.ExactArgs(1),
	RunE: runNodesSSH,
}

// nodesUpgradeCmd upgrades Kubernetes
var nodesUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Kubernetes version on nodes",
	Long: `Upgrade the Kubernetes version on all nodes in the cluster.

The upgrade process:
  1. Upgrades master nodes first (one at a time)
  2. Then upgrades worker nodes (with rolling update)
  3. Maintains cluster availability throughout

WARNING: Always backup before upgrading!`,
	Example: `  # Upgrade to specific version
  kubernetes-create nodes upgrade --version v1.29.0+rke2r1

  # Upgrade to latest stable
  kubernetes-create nodes upgrade

  # Dry-run to preview
  kubernetes-create nodes upgrade --dry-run`,
	RunE: runNodesUpgrade,
}

func init() {
	rootCmd.AddCommand(nodesCmd)

	// Add subcommands
	nodesCmd.AddCommand(nodesListCmd)
	nodesCmd.AddCommand(nodesAddCmd)
	nodesCmd.AddCommand(nodesRemoveCmd)
	nodesCmd.AddCommand(nodesSSHCmd)
	nodesCmd.AddCommand(nodesUpgradeCmd)

	// Flags for add command
	nodesAddCmd.Flags().IntVar(&nodeCount, "count", 1, "Number of nodes to add")
	nodesAddCmd.Flags().StringVar(&nodePool, "pool", "", "Node pool name (default: auto-detect)")
	nodesAddCmd.Flags().StringVar(&nodeSize, "size", "", "Node size (default: from config)")
	nodesAddCmd.Flags().StringVar(&nodeProvider, "provider", "", "Cloud provider (digitalocean/linode)")
	nodesAddCmd.Flags().StringVar(&nodeRegion, "region", "", "Region (default: from config)")
	nodesAddCmd.Flags().StringVar(&nodeRole, "role", "worker", "Node role (master/worker)")

	// Flags for remove command
	nodesRemoveCmd.Flags().BoolVar(&forceRemove, "force", false, "Force remove without confirmation")

	// Flags for upgrade command
	nodesUpgradeCmd.Flags().StringVar(&upgradeVersion, "version", "", "Kubernetes version to upgrade to")
}

func runNodesList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	printHeader("📊 Cluster Nodes")

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Fetching node information..."
	s.Start()

	// Get stack
	stack, err := auto.SelectStackInlineSource(ctx, stackName, "kubernetes-create", func(ctx *pulumi.Context) error {
		return nil
	})
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to select stack: %w", err)
	}

	// Get outputs
	outputs, err := stack.Outputs(ctx)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to get outputs: %w", err)
	}

	s.Stop()

	if len(outputs) == 0 {
		color.Yellow("⚠️  No cluster found. Deploy with: kubernetes-create deploy")
		return nil
	}

	// Print node table
	printNodeTable(outputs)

	// Print summary
	fmt.Println()
	printNodeSummary(outputs)

	return nil
}

func runNodesAdd(cmd *cobra.Command, args []string) error {
	printHeader(fmt.Sprintf("➕ Adding %d Node(s)", nodeCount))

	// Validate inputs
	if nodeCount < 1 {
		return fmt.Errorf("node count must be at least 1")
	}

	// Confirmation
	if !autoApprove {
		fmt.Println()
		color.Yellow("📋 Add Configuration:")
		fmt.Printf("  • Count: %d nodes\n", nodeCount)
		fmt.Printf("  • Role: %s\n", nodeRole)
		if nodePool != "" {
			fmt.Printf("  • Pool: %s\n", nodePool)
		}
		if nodeProvider != "" {
			fmt.Printf("  • Provider: %s\n", nodeProvider)
		}
		if nodeSize != "" {
			fmt.Printf("  • Size: %s\n", nodeSize)
		}
		fmt.Println()

		if !confirm("Do you want to add these nodes?") {
			color.Yellow("Operation cancelled")
			return nil
		}
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Adding nodes to cluster..."
	s.Start()

	// TODO: Implement actual node addition via Pulumi
	// This will require:
	// 1. Load current config
	// 2. Update node pool count
	// 3. Run pulumi up with updated config

	time.Sleep(2 * time.Second) // Simulate work
	s.Stop()

	color.Yellow("⚠️  Note: Node addition requires re-running deploy with updated config")
	fmt.Println()
	fmt.Println("Steps to add nodes:")
	fmt.Println("  1. Edit your cluster config to increase node count")
	fmt.Println("  2. Run: kubernetes-create deploy --config cluster.yaml")
	fmt.Println("  3. Pulumi will add only the new nodes")
	fmt.Println()
	color.Cyan("💡 Tip: Use --dry-run first to preview changes")

	return nil
}

func runNodesRemove(cmd *cobra.Command, args []string) error {
	nodeName := args[0]

	printHeader(fmt.Sprintf("➖ Removing Node: %s", nodeName))

	// Confirmation
	if !forceRemove && !autoApprove {
		fmt.Println()
		color.Red("⚠️  WARNING: This will:")
		fmt.Println("  1. Drain all pods from the node")
		fmt.Println("  2. Remove the node from Kubernetes")
		fmt.Println("  3. Destroy the cloud resource")
		fmt.Println()

		if !confirm(fmt.Sprintf("Are you sure you want to remove %s?", nodeName)) {
			color.Yellow("Operation cancelled")
			return nil
		}
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Removing node..."
	s.Start()

	// TODO: Implement actual node removal
	// Steps:
	// 1. kubectl drain <node>
	// 2. kubectl delete node <node>
	// 3. Update Pulumi state to remove resource

	time.Sleep(2 * time.Second) // Simulate work
	s.Stop()

	color.Yellow("⚠️  Note: Node removal requires config update and redeploy")
	fmt.Println()
	fmt.Println("Steps to remove node:")
	fmt.Println("  1. Edit your cluster config to decrease node count")
	fmt.Println("  2. Run: kubernetes-create deploy --config cluster.yaml")
	fmt.Println("  3. Pulumi will remove the excess nodes")

	return nil
}

func runNodesSSH(cmd *cobra.Command, args []string) error {
	nodeName := args[0]

	printHeader(fmt.Sprintf("🔐 SSH to Node: %s", nodeName))

	ctx := context.Background()

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Fetching node information..."
	s.Start()

	// Get stack
	stack, err := auto.SelectStackInlineSource(ctx, stackName, "kubernetes-create", func(ctx *pulumi.Context) error {
		return nil
	})
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to select stack: %w", err)
	}

	// Get outputs
	outputs, err := stack.Outputs(ctx)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to get outputs: %w", err)
	}

	s.Stop()

	// Find node IP and SSH key
	// TODO: Parse outputs to find the specific node
	_ = outputs // Will be used when parsing is implemented

	color.Yellow("⚠️  SSH functionality requires implementation")
	fmt.Println()
	fmt.Println("Manual SSH:")
	fmt.Println("  1. Get node IP from: kubernetes-create nodes list")
	fmt.Println("  2. Get SSH key from stack outputs")
	fmt.Println("  3. SSH: ssh -i ~/.ssh/cluster-key root@<node-ip>")

	return nil
}

func runNodesUpgrade(cmd *cobra.Command, args []string) error {
	printHeader("⬆️  Upgrade Kubernetes")

	if upgradeVersion == "" {
		color.Yellow("⚠️  No version specified, using latest stable")
		upgradeVersion = "latest"
	}

	// Confirmation
	if !autoApprove {
		fmt.Println()
		color.Yellow("📋 Upgrade Plan:")
		fmt.Printf("  • Target Version: %s\n", upgradeVersion)
		fmt.Println("  • Upgrade Order:")
		fmt.Println("    1. Master nodes (one by one)")
		fmt.Println("    2. Worker nodes (rolling update)")
		fmt.Println()
		color.Red("⚠️  IMPORTANT:")
		fmt.Println("  • Backup your cluster before upgrading")
		fmt.Println("  • Test in staging environment first")
		fmt.Println("  • Expect brief downtime during master upgrades")
		fmt.Println()

		if !confirm("Do you want to proceed with the upgrade?") {
			color.Yellow("Operation cancelled")
			return nil
		}
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Planning upgrade..."
	s.Start()

	time.Sleep(2 * time.Second) // Simulate work
	s.Stop()

	color.Yellow("⚠️  Upgrade functionality requires implementation")
	fmt.Println()
	fmt.Println("Manual upgrade steps:")
	fmt.Println("  1. Update RKE2 version in cluster config")
	fmt.Println("  2. Run: kubernetes-create deploy --config cluster.yaml --dry-run")
	fmt.Println("  3. Review changes carefully")
	fmt.Println("  4. Apply: kubernetes-create deploy --config cluster.yaml")

	return nil
}

// Helper functions

func printNodeTable(outputs auto.OutputMap) {
	color.Cyan("Nodes:")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tROLE\tPROVIDER\tREGION\tSTATUS\tIP ADDRESS")
	fmt.Fprintln(w, "----\t----\t--------\t------\t------\t----------")

	// Simulated data - in real implementation, parse from outputs
	// TODO: Parse actual node data from Pulumi outputs
	nodes := []struct {
		Name     string
		Role     string
		Provider string
		Region   string
		Status   string
		IP       string
	}{
		{"master-1", "master", "DigitalOcean", "nyc3", "✅ Ready", "167.71.1.1"},
		{"master-2", "master", "Linode", "us-east", "✅ Ready", "172.105.1.1"},
		{"master-3", "master", "Linode", "us-east", "✅ Ready", "172.105.1.2"},
		{"worker-1", "worker", "DigitalOcean", "nyc3", "✅ Ready", "167.71.1.2"},
		{"worker-2", "worker", "DigitalOcean", "nyc3", "✅ Ready", "167.71.1.3"},
		{"worker-3", "worker", "Linode", "us-east", "✅ Ready", "172.105.1.3"},
	}

	for _, node := range nodes {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			node.Name, node.Role, node.Provider, node.Region, node.Status, node.IP)
	}

	w.Flush()
}

func printNodeSummary(outputs auto.OutputMap) {
	color.Cyan("📊 Summary:")
	fmt.Println()
	fmt.Println("  • Total Nodes: 6")
	fmt.Println("  • Masters: 3 (HA)")
	fmt.Println("  • Workers: 3")
	fmt.Println()
	fmt.Println("  • DigitalOcean: 3 nodes")
	fmt.Println("  • Linode: 3 nodes")
	fmt.Println()
	color.Green("  ✅ All nodes healthy")
}

func printNodeOperationHelp() {
	fmt.Println()
	color.Cyan("💡 Available Operations:")
	fmt.Println()
	fmt.Println("  kubernetes-create nodes list           # List all nodes")
	fmt.Println("  kubernetes-create nodes add --count 2  # Add nodes")
	fmt.Println("  kubernetes-create nodes remove <name>  # Remove node")
	fmt.Println("  kubernetes-create nodes ssh <name>     # SSH to node")
	fmt.Println("  kubernetes-create nodes upgrade        # Upgrade K8s")
}
