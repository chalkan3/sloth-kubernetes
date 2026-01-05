package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/pkg/health"
	"github.com/chalkan3/sloth-kubernetes/pkg/operations"
)

var (
	healthVerbose bool
	healthCompact bool
	healthChecks  []string
)

var healthCmd = &cobra.Command{
	Use:   "health [stack-name]",
	Short: "Check cluster health status",
	Long: `Check the health status of your Kubernetes cluster.

This command runs comprehensive health checks including:
  - Node health and readiness
  - System pods status (kube-system namespace)
  - CoreDNS availability
  - Certificate expiration
  - Etcd cluster health
  - API server responsiveness
  - Persistent volume claims status
  - CNI/networking status
  - Memory pressure on nodes
  - Disk pressure on nodes

The kubeconfig is automatically retrieved from the specified Pulumi stack.`,
	Example: `  # Check cluster health
  sloth-kubernetes health my-cluster

  # Run only specific checks
  sloth-kubernetes health my-cluster --checks nodes,pods,dns

  # Verbose output with all details
  sloth-kubernetes health my-cluster --verbose

  # Compact output (only show issues)
  sloth-kubernetes health my-cluster --compact`,
	RunE: runHealthCheck,
}

var healthSummaryCmd = &cobra.Command{
	Use:   "summary [stack-name]",
	Short: "Quick health summary",
	Long:  `Show a quick one-line health summary of the cluster.`,
	RunE:  runHealthSummary,
}

var healthNodesCmd = &cobra.Command{
	Use:   "nodes [stack-name]",
	Short: "Check node health only",
	Long:  `Check only the health status of cluster nodes.`,
	RunE:  runHealthNodes,
}

var healthPodsCmd = &cobra.Command{
	Use:   "pods [stack-name]",
	Short: "Check system pods health only",
	Long:  `Check only the health status of system pods in kube-system namespace.`,
	RunE:  runHealthPods,
}

var healthCertsCmd = &cobra.Command{
	Use:   "certs [stack-name]",
	Short: "Check certificate expiration",
	Long:  `Check only the certificate expiration status.`,
	RunE:  runHealthCerts,
}

func init() {
	rootCmd.AddCommand(healthCmd)
	healthCmd.AddCommand(healthSummaryCmd)
	healthCmd.AddCommand(healthNodesCmd)
	healthCmd.AddCommand(healthPodsCmd)
	healthCmd.AddCommand(healthCertsCmd)

	// Main health command flags
	healthCmd.Flags().BoolVarP(&healthVerbose, "verbose", "V", false, "Show verbose output with all details")
	healthCmd.Flags().BoolVar(&healthCompact, "compact", false, "Show compact output (only issues)")
	healthCmd.Flags().StringSliceVar(&healthChecks, "checks", []string{}, "Specific checks to run (nodes,pods,dns,certs,etcd,api,storage,network,memory,disk)")

	// Subcommand flags
	healthNodesCmd.Flags().BoolVarP(&healthVerbose, "verbose", "V", false, "Show verbose output")
	healthPodsCmd.Flags().BoolVarP(&healthVerbose, "verbose", "V", false, "Show verbose output")
}

func runHealthCheck(cmd *cobra.Command, args []string) error {
	printHeader("Cluster Health Check")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	checker, clusterName, err := createHealthCheckerFromStack(targetStack)
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("Running health checks...")
	fmt.Println()

	// Run all checks
	startTime := time.Now()
	report, err := checker.RunAllChecks(clusterName)
	if err != nil {
		operations.RecordHealthCheck(targetStack, "error", 0, 0, 0, 0, err.Error(), time.Since(startTime), err)
		return fmt.Errorf("health check failed: %w", err)
	}

	// Filter checks if specific ones requested
	if len(healthChecks) > 0 {
		report = filterChecks(report, healthChecks)
	}

	// Record health check
	summary := fmt.Sprintf("%d healthy, %d warnings, %d critical", report.Summary.HealthyChecks, report.Summary.WarningChecks, report.Summary.CriticalChecks)
	operations.RecordHealthCheck(targetStack, string(report.OverallStatus), report.Summary.TotalChecks, report.Summary.HealthyChecks, report.Summary.WarningChecks, report.Summary.CriticalChecks, summary, time.Since(startTime), nil)

	// Print report based on output mode
	if healthCompact {
		report.PrintCompact()
	} else {
		report.PrintReport()
	}

	// Return error if cluster is unhealthy (for CI/CD integration)
	if report.OverallStatus == health.StatusCritical {
		return fmt.Errorf("cluster health check failed: %d critical issues found", report.Summary.CriticalChecks)
	}

	return nil
}

func runHealthSummary(cmd *cobra.Command, args []string) error {
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	checker, clusterName, err := createHealthCheckerFromStack(targetStack)
	if err != nil {
		return err
	}

	report, err := checker.RunAllChecks(clusterName)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// Print one-line summary
	statusIcon := getHealthIcon(report.OverallStatus)
	statusColor := getHealthColor(report.OverallStatus)

	statusColor.Printf("%s %s: %s (%d healthy, %d warning, %d critical)\n",
		statusIcon,
		clusterName,
		strings.ToUpper(string(report.OverallStatus)),
		report.Summary.HealthyChecks,
		report.Summary.WarningChecks,
		report.Summary.CriticalChecks)

	return nil
}

func runHealthNodes(cmd *cobra.Command, args []string) error {
	printHeader("Node Health Check")

	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	checker, _, err := createHealthCheckerFromStack(targetStack)
	if err != nil {
		return err
	}

	result := checker.CheckNodes()
	printCheckResult(result)

	return nil
}

func runHealthPods(cmd *cobra.Command, args []string) error {
	printHeader("System Pods Health Check")

	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	checker, _, err := createHealthCheckerFromStack(targetStack)
	if err != nil {
		return err
	}

	result := checker.CheckSystemPods()
	printCheckResult(result)

	return nil
}

func runHealthCerts(cmd *cobra.Command, args []string) error {
	printHeader("Certificate Health Check")

	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	checker, _, err := createHealthCheckerFromStack(targetStack)
	if err != nil {
		return err
	}

	result := checker.CheckCertificates()
	printCheckResult(result)

	return nil
}

// Helper functions

func createHealthCheckerFromStack(targetStack string) (*health.Checker, string, error) {
	// Get kubeconfig from stack
	kubeconfigPath, err := GetKubeconfigFromStack(targetStack)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get kubeconfig from stack '%s': %w", targetStack, err)
	}

	checker := health.NewChecker("", "")
	checker.SetKubeconfig(kubeconfigPath)
	checker.SetVerbose(healthVerbose)

	return checker, targetStack, nil
}

func filterChecks(report *health.HealthReport, checks []string) *health.HealthReport {
	checkMap := map[string]string{
		"nodes":   "Node Health",
		"pods":    "System Pods",
		"dns":     "CoreDNS",
		"certs":   "Certificates",
		"etcd":    "Etcd Cluster",
		"api":     "API Server",
		"storage": "Storage (PVCs)",
		"network": "Networking",
		"memory":  "Memory Pressure",
		"disk":    "Disk Pressure",
	}

	// Build set of requested check names
	requestedNames := make(map[string]bool)
	for _, check := range checks {
		if name, ok := checkMap[strings.ToLower(check)]; ok {
			requestedNames[name] = true
		}
	}

	// Filter checks
	var filteredChecks []health.CheckResult
	for _, check := range report.Checks {
		if requestedNames[check.Name] {
			filteredChecks = append(filteredChecks, check)
		}
	}

	// Recalculate summary
	report.Checks = filteredChecks
	report.Summary = health.Summary{}
	for _, check := range filteredChecks {
		report.Summary.TotalChecks++
		switch check.Status {
		case health.StatusHealthy:
			report.Summary.HealthyChecks++
		case health.StatusWarning:
			report.Summary.WarningChecks++
		case health.StatusCritical:
			report.Summary.CriticalChecks++
		default:
			report.Summary.UnknownChecks++
		}
	}

	// Recalculate overall status
	report.OverallStatus = health.StatusHealthy
	for _, check := range filteredChecks {
		if check.Status == health.StatusCritical {
			report.OverallStatus = health.StatusCritical
			break
		} else if check.Status == health.StatusWarning && report.OverallStatus != health.StatusCritical {
			report.OverallStatus = health.StatusWarning
		}
	}

	return report
}

func printCheckResult(result health.CheckResult) {
	fmt.Println()
	icon := getHealthIcon(result.Status)
	statusColor := getHealthColor(result.Status)

	statusColor.Printf("%s %s\n", icon, result.Name)
	fmt.Printf("   Status:  %s\n", result.Status)
	fmt.Printf("   Message: %s\n", result.Message)
	fmt.Printf("   Duration: %s\n", result.Duration)

	if len(result.Details) > 0 && healthVerbose {
		fmt.Println("   Details:")
		for _, detail := range result.Details {
			fmt.Printf("     - %s\n", detail)
		}
	}

	if result.Remediation != "" && (result.Status == health.StatusWarning || result.Status == health.StatusCritical) {
		fmt.Println()
		color.Yellow("   Remediation: %s", result.Remediation)
	}

	fmt.Println()
}

func getHealthIcon(status health.CheckStatus) string {
	switch status {
	case health.StatusHealthy:
		return "[OK]"
	case health.StatusWarning:
		return "[WARN]"
	case health.StatusCritical:
		return "[FAIL]"
	default:
		return "[?]"
	}
}

func getHealthColor(status health.CheckStatus) *color.Color {
	switch status {
	case health.StatusHealthy:
		return color.New(color.FgGreen)
	case health.StatusWarning:
		return color.New(color.FgYellow)
	case health.StatusCritical:
		return color.New(color.FgRed)
	default:
		return color.New(color.FgWhite)
	}
}
