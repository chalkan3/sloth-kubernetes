package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/pkg/operations"
	"github.com/chalkan3/sloth-kubernetes/pkg/upgrade"
)

var (
	upgradeTargetVersion string
	upgradeStrategy      string
	upgradeDryRun        bool
	upgradeVerbose       bool
	upgradeForce         bool
	upgradeKubeconfig    string
	upgradeNodeFilter    []string
	upgradeBackupDir     string
	upgradeTimeout       int
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade cluster Kubernetes version",
	Long: `Upgrade your Kubernetes cluster to a new version.

This command supports multiple upgrade strategies:
  - rolling: Upgrade one node at a time (safest, default)
  - blue-green: Create new nodes, migrate workloads, remove old
  - canary: Upgrade a subset first, then proceed
  - in-place: Upgrade all nodes simultaneously (fastest, riskiest)

The upgrade process includes:
  - Pre-flight checks (cluster health, etcd backup, node readiness)
  - Node cordoning and draining
  - Kubernetes component upgrade
  - Post-upgrade validation
  - Automatic rollback on failure`,
	Example: `  # Plan an upgrade to version 1.29.0
  sloth-kubernetes upgrade plan --to 1.29.0

  # Execute upgrade with rolling strategy
  sloth-kubernetes upgrade apply --to 1.29.0 --strategy rolling

  # Dry-run to see what would happen
  sloth-kubernetes upgrade apply --to 1.29.0 --dry-run

  # Rollback to previous version
  sloth-kubernetes upgrade rollback

  # Check available versions
  sloth-kubernetes upgrade versions

  # Upgrade specific nodes only
  sloth-kubernetes upgrade apply --to 1.29.0 --nodes master-1,worker-1`,
}

var upgradePlanCmd = &cobra.Command{
	Use:   "plan [stack-name]",
	Short: "Create an upgrade plan",
	Long: `Create a detailed upgrade plan without executing it.

The plan shows:
  - Current cluster state
  - Nodes to be upgraded and their order
  - Pre-flight checks to be performed
  - Estimated downtime per node
  - Potential risks identified`,
	RunE: runUpgradePlan,
}

var upgradeApplyCmd = &cobra.Command{
	Use:   "apply [stack-name]",
	Short: "Execute cluster upgrade",
	Long: `Execute the upgrade plan on the cluster.

Use --dry-run to simulate the upgrade without making changes.
Use --force to skip confirmation prompts.`,
	RunE: runUpgradeApply,
}

var upgradeRollbackCmd = &cobra.Command{
	Use:   "rollback [stack-name]",
	Short: "Rollback to previous version",
	Long: `Rollback the cluster to the previous Kubernetes version.

This command uses the rollback information stored from the last upgrade.
Only the most recent upgrade can be rolled back.`,
	RunE: runUpgradeRollback,
}

var upgradeVersionsCmd = &cobra.Command{
	Use:   "versions [stack-name]",
	Short: "List available versions",
	Long:  `List current cluster version and available upgrade targets.`,
	RunE:  runUpgradeVersions,
}

var upgradeStatusCmd = &cobra.Command{
	Use:   "status [stack-name]",
	Short: "Show upgrade status",
	Long:  `Show the status of an ongoing or recent upgrade.`,
	RunE:  runUpgradeStatus,
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.AddCommand(upgradePlanCmd)
	upgradeCmd.AddCommand(upgradeApplyCmd)
	upgradeCmd.AddCommand(upgradeRollbackCmd)
	upgradeCmd.AddCommand(upgradeVersionsCmd)
	upgradeCmd.AddCommand(upgradeStatusCmd)

	// Upgrade plan flags
	upgradePlanCmd.Flags().StringVar(&upgradeTargetVersion, "to", "", "Target Kubernetes version (required)")
	upgradePlanCmd.Flags().StringVar(&upgradeStrategy, "strategy", "rolling", "Upgrade strategy (rolling, blue-green, canary, in-place)")
	upgradePlanCmd.Flags().StringVar(&upgradeKubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	upgradePlanCmd.MarkFlagRequired("to")

	// Upgrade apply flags
	upgradeApplyCmd.Flags().StringVar(&upgradeTargetVersion, "to", "", "Target Kubernetes version (required)")
	upgradeApplyCmd.Flags().StringVar(&upgradeStrategy, "strategy", "rolling", "Upgrade strategy (rolling, blue-green, canary, in-place)")
	upgradeApplyCmd.Flags().BoolVar(&upgradeDryRun, "dry-run", false, "Simulate upgrade without making changes")
	upgradeApplyCmd.Flags().BoolVarP(&upgradeVerbose, "verbose", "v", false, "Show verbose output")
	upgradeApplyCmd.Flags().BoolVar(&upgradeForce, "force", false, "Skip confirmation prompts")
	upgradeApplyCmd.Flags().StringVar(&upgradeKubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	upgradeApplyCmd.Flags().StringSliceVar(&upgradeNodeFilter, "nodes", []string{}, "Specific nodes to upgrade (comma-separated)")
	upgradeApplyCmd.Flags().StringVar(&upgradeBackupDir, "backup-dir", "/var/lib/sloth-kubernetes/backups", "Directory for etcd backups")
	upgradeApplyCmd.Flags().IntVar(&upgradeTimeout, "timeout", 600, "Timeout in seconds for each node upgrade")
	upgradeApplyCmd.MarkFlagRequired("to")

	// Rollback flags
	upgradeRollbackCmd.Flags().BoolVar(&upgradeForce, "force", false, "Skip confirmation prompts")
	upgradeRollbackCmd.Flags().BoolVarP(&upgradeVerbose, "verbose", "v", false, "Show verbose output")
	upgradeRollbackCmd.Flags().StringVar(&upgradeKubeconfig, "kubeconfig", "", "Path to kubeconfig file")

	// Versions flags
	upgradeVersionsCmd.Flags().StringVar(&upgradeKubeconfig, "kubeconfig", "", "Path to kubeconfig file")

	// Status flags
	upgradeStatusCmd.Flags().StringVar(&upgradeKubeconfig, "kubeconfig", "", "Path to kubeconfig file")
}

func createUpgradeManager(targetStack string) (*upgrade.Manager, error) {
	// Get stack info including SSH credentials
	stackInfo, err := GetStackInfo(targetStack)
	if err != nil {
		return nil, fmt.Errorf("failed to get stack info: %w", err)
	}

	// Get kubeconfig from stack
	kubeconfigPath, err := GetKubeconfigFromStack(targetStack)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig from stack '%s': %w", targetStack, err)
	}

	manager := upgrade.NewManager(stackInfo.MasterIP, "", kubeconfigPath)
	manager.SetDryRun(upgradeDryRun)
	manager.SetVerbose(upgradeVerbose)

	return manager, nil
}

func runUpgradePlan(cmd *cobra.Command, args []string) error {
	printHeader("Upgrade Plan")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createUpgradeManager(targetStack)
	if err != nil {
		return err
	}

	// Get current version
	currentVersion, err := manager.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	fmt.Println()
	color.Cyan("Creating upgrade plan...")
	fmt.Println()

	strategy := upgrade.UpgradeStrategy(upgradeStrategy)
	plan, err := manager.CreatePlan(upgradeTargetVersion, strategy)
	if err != nil {
		return fmt.Errorf("failed to create upgrade plan: %w", err)
	}

	// Print plan summary
	printUpgradePlanSummary(plan, currentVersion)

	return nil
}

func runUpgradeApply(cmd *cobra.Command, args []string) error {
	printHeader("Cluster Upgrade")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createUpgradeManager(targetStack)
	if err != nil {
		return err
	}

	strategy := upgrade.UpgradeStrategy(upgradeStrategy)
	plan, err := manager.CreatePlan(upgradeTargetVersion, strategy)
	if err != nil {
		return fmt.Errorf("failed to create upgrade plan: %w", err)
	}

	// Filter nodes if specified
	if len(upgradeNodeFilter) > 0 {
		plan = filterUpgradeNodes(plan, upgradeNodeFilter)
	}

	// Show plan and confirm
	fmt.Println()
	printUpgradePlanSummary(plan, plan.CurrentVersion)

	if !upgradeForce && !upgradeDryRun {
		fmt.Println()
		color.Yellow("This will upgrade your cluster. Are you sure? [y/N]: ")
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
			color.Yellow("Upgrade cancelled.")
			return nil
		}
	}

	if upgradeDryRun {
		fmt.Println()
		color.Cyan("[DRY-RUN] Would execute the following upgrade:")
		printUpgradeSteps(plan)
		return nil
	}

	// Execute upgrade
	fmt.Println()
	color.Cyan("Starting upgrade...")
	fmt.Println()

	startTime := time.Now()
	result, err := manager.Execute(plan)
	if err != nil {
		// Record failed upgrade
		operations.RecordUpgradeOperation(targetStack, "upgrade", plan.CurrentVersion, plan.TargetVersion, string(plan.Strategy), "failed", len(plan.Nodes), 0, len(plan.Nodes), time.Since(startTime), err)
		return fmt.Errorf("upgrade failed: %w", err)
	}

	// Print results
	printUpgradeResult(result)

	// Record upgrade result
	status := "success"
	if result.Status == upgrade.StatusFailed {
		status = "failed"
	} else if result.Status == upgrade.StatusRolledBack {
		status = "rollback"
	} else if result.NodesUpgraded < result.TotalNodes {
		status = "partial"
	}

	nodesFailed := result.TotalNodes - result.NodesUpgraded
	operations.RecordUpgradeOperation(targetStack, "upgrade", result.PreviousVersion, result.NewVersion, string(plan.Strategy), status, result.TotalNodes, result.NodesUpgraded, nodesFailed, time.Since(startTime), nil)

	if result.Status == upgrade.StatusFailed {
		return fmt.Errorf("upgrade completed with failures")
	}

	return nil
}

func runUpgradeRollback(cmd *cobra.Command, args []string) error {
	printHeader("Cluster Rollback")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createUpgradeManager(targetStack)
	if err != nil {
		return err
	}

	// Confirm rollback
	if !upgradeForce {
		fmt.Println()
		color.Yellow("This will rollback your cluster to the previous version. Are you sure? [y/N]: ")
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
			color.Yellow("Rollback cancelled.")
			return nil
		}
	}

	fmt.Println()
	color.Cyan("Starting rollback...")
	fmt.Println()

	// Get current version before rollback
	currentVersion, _ := manager.GetCurrentVersion()

	startTime := time.Now()
	err = manager.Rollback()
	if err != nil {
		operations.RecordUpgradeOperation(targetStack, "rollback", currentVersion, "", "", "failed", 0, 0, 0, time.Since(startTime), err)
		return fmt.Errorf("rollback failed: %w", err)
	}

	// Get new version after rollback
	newVersion, _ := manager.GetCurrentVersion()
	operations.RecordUpgradeOperation(targetStack, "rollback", currentVersion, newVersion, "", "success", 0, 0, 0, time.Since(startTime), nil)

	color.Green("[OK] Rollback completed successfully!")
	return nil
}

func runUpgradeVersions(cmd *cobra.Command, args []string) error {
	printHeader("Available Versions")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createUpgradeManager(targetStack)
	if err != nil {
		return err
	}

	// Get current version
	currentVersion, err := manager.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Get available versions
	availableVersions, err := manager.GetAvailableVersions()
	if err != nil {
		return fmt.Errorf("failed to get available versions: %w", err)
	}

	fmt.Println()
	color.Cyan("Current Version:")
	fmt.Printf("  %s\n", currentVersion)

	fmt.Println()
	color.Cyan("Available Upgrade Targets:")
	for _, v := range availableVersions {
		if v > currentVersion {
			fmt.Printf("  %s\n", v)
		}
	}

	fmt.Println()
	color.Cyan("All Supported Versions:")
	for _, v := range availableVersions {
		indicator := ""
		if v == currentVersion {
			indicator = " (current)"
			color.Green("  %s%s", v, indicator)
		} else if v > currentVersion {
			indicator = " (upgrade available)"
			fmt.Printf("  %s%s\n", v, indicator)
		} else {
			indicator = " (downgrade)"
			color.White("  %s%s", v, indicator)
		}
	}

	return nil
}

func runUpgradeStatus(cmd *cobra.Command, args []string) error {
	printHeader("Upgrade Status")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createUpgradeManager(targetStack)
	if err != nil {
		return err
	}

	// Get node versions
	nodes, err := manager.GetNodes()
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	// Get current version for comparison
	currentVersion, err := manager.GetCurrentVersion()
	if err != nil {
		currentVersion = "unknown"
	}

	fmt.Println()
	color.Cyan("Cluster Version: %s", currentVersion)
	fmt.Println()

	color.Cyan("Node Status:")
	fmt.Printf("%-30s %-15s %-15s %-10s\n", "NODE", "VERSION", "STATUS", "ROLE")
	fmt.Println(strings.Repeat("-", 75))

	for _, node := range nodes {
		statusColor := color.New(color.FgGreen)
		if node.Status != "Ready" {
			statusColor = color.New(color.FgYellow)
		}
		if node.Version != currentVersion {
			statusColor = color.New(color.FgYellow)
		}

		statusColor.Printf("%-30s %-15s %-15s %-10s\n",
			node.Name,
			node.Version,
			node.Status,
			node.Role)
	}

	return nil
}

// Helper functions

func printUpgradePlanSummary(plan *upgrade.UpgradePlan, currentVersion string) {
	color.Cyan("Upgrade Plan Summary")
	fmt.Println(strings.Repeat("=", 50))

	fmt.Printf("Cluster:          %s\n", plan.ClusterName)
	fmt.Printf("Current Version:  %s\n", currentVersion)
	fmt.Printf("Target Version:   %s\n", plan.TargetVersion)
	fmt.Printf("Strategy:         %s\n", plan.Strategy)
	fmt.Printf("Total Nodes:      %d\n", len(plan.Nodes))

	// Count by role
	masters := 0
	workers := 0
	for _, node := range plan.Nodes {
		if node.Role == "master" || node.Role == "control-plane" {
			masters++
		} else {
			workers++
		}
	}
	fmt.Printf("  - Masters:      %d\n", masters)
	fmt.Printf("  - Workers:      %d\n", workers)

	// Show pre-checks
	fmt.Println()
	color.Cyan("Pre-flight Checks:")
	for i, check := range plan.PreChecks {
		fmt.Printf("  %d. %s\n", i+1, check.Name)
	}

	// Show risks
	if len(plan.Risks) > 0 {
		fmt.Println()
		color.Yellow("Identified Risks:")
		for _, risk := range plan.Risks {
			color.Yellow("  [!] %s", risk)
		}
	}

	// Show node order
	fmt.Println()
	color.Cyan("Upgrade Order:")
	for i, node := range plan.Nodes {
		fmt.Printf("  %d. %s (%s)\n", i+1, node.NodeName, node.Role)
	}
}

func printUpgradeSteps(plan *upgrade.UpgradePlan) {
	fmt.Println()
	color.Cyan("Planned Steps:")

	step := 1
	fmt.Printf("  %d. Run pre-flight checks\n", step)
	step++
	fmt.Printf("  %d. Create etcd backup\n", step)
	step++

	for _, node := range plan.Nodes {
		fmt.Printf("  %d. Cordon node %s\n", step, node.NodeName)
		step++
		fmt.Printf("  %d. Drain node %s\n", step, node.NodeName)
		step++
		fmt.Printf("  %d. Upgrade node %s to %s\n", step, node.NodeName, plan.TargetVersion)
		step++
		fmt.Printf("  %d. Uncordon node %s\n", step, node.NodeName)
		step++
		fmt.Printf("  %d. Wait for node %s to be ready\n", step, node.NodeName)
		step++
	}

	fmt.Printf("  %d. Run post-upgrade validation\n", step)
}

func printUpgradeResult(result *upgrade.UpgradeResult) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 50))

	switch result.Status {
	case upgrade.StatusCompleted:
		color.Green("[OK] Upgrade completed successfully!")
	case upgrade.StatusFailed:
		color.Red("[FAIL] Upgrade failed!")
	case upgrade.StatusRolledBack:
		color.Yellow("[ROLLBACK] Upgrade was rolled back")
	default:
		color.Yellow("[%s] Upgrade status: %s", result.Status, result.Status)
	}

	fmt.Println()
	fmt.Printf("Duration: %s\n", result.Duration)
	fmt.Printf("Nodes Upgraded: %d/%d\n", result.NodesUpgraded, result.TotalNodes)

	if result.PreviousVersion != "" {
		fmt.Printf("Previous Version: %s\n", result.PreviousVersion)
	}
	if result.NewVersion != "" {
		fmt.Printf("New Version: %s\n", result.NewVersion)
	}

	// Show node results
	if len(result.NodeResults) > 0 {
		fmt.Println()
		color.Cyan("Node Results:")
		for _, nr := range result.NodeResults {
			var statusIcon string
			var statusColor *color.Color
			switch nr.Status {
			case upgrade.NodeStatusCompleted:
				statusIcon = "[OK]"
				statusColor = color.New(color.FgGreen)
			case upgrade.NodeStatusFailed:
				statusIcon = "[FAIL]"
				statusColor = color.New(color.FgRed)
			case upgrade.NodeStatusSkipped:
				statusIcon = "[SKIP]"
				statusColor = color.New(color.FgYellow)
			default:
				statusIcon = "[?]"
				statusColor = color.New(color.FgWhite)
			}
			statusColor.Printf("  %s %s (%s)\n", statusIcon, nr.NodeName, nr.Duration)
			if nr.Error != "" {
				color.Red("      Error: %s", nr.Error)
			}
		}
	}

	// Show errors
	if len(result.Errors) > 0 {
		fmt.Println()
		color.Red("Errors:")
		for _, err := range result.Errors {
			color.Red("  - %s", err)
		}
	}

	// Show warnings
	if len(result.Warnings) > 0 {
		fmt.Println()
		color.Yellow("Warnings:")
		for _, warn := range result.Warnings {
			color.Yellow("  - %s", warn)
		}
	}
}

func filterUpgradeNodes(plan *upgrade.UpgradePlan, nodeFilter []string) *upgrade.UpgradePlan {
	filterSet := make(map[string]bool)
	for _, n := range nodeFilter {
		filterSet[n] = true
	}

	var filteredNodes []upgrade.NodeUpgradePlan
	for _, node := range plan.Nodes {
		if filterSet[node.NodeName] {
			filteredNodes = append(filteredNodes, node)
		}
	}

	plan.Nodes = filteredNodes
	return plan
}
