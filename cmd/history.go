package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/pkg/operations"
)

var historyCmd = &cobra.Command{
	Use:   "history <stack-name> [type]",
	Short: "View operation history for a stack",
	Long: `Display the history of CLI operations recorded in the Pulumi stack state.

Supported operation types:
  - backups     Backup create/restore/delete operations
  - upgrades    Cluster upgrade operations
  - health      Health check results
  - benchmarks  Benchmark runs
  - nodes       Node add/remove/drain operations
  - vpn         VPN join/leave/test operations
  - argocd      ArgoCD install/sync operations
  - addons      Addons bootstrap/install operations
  - salt        Salt command executions
  - validation  Validation check results

The history command shows the last 50 records of each operation type.`,
	Example: `  # View all operation history
  sloth-kubernetes history my-cluster

  # View backup history only
  sloth-kubernetes history my-cluster backups

  # View upgrade history only
  sloth-kubernetes history my-cluster upgrades

  # View health check history
  sloth-kubernetes history my-cluster health

  # View benchmark history
  sloth-kubernetes history my-cluster benchmarks

  # View node operations
  sloth-kubernetes history my-cluster nodes

  # View VPN operations
  sloth-kubernetes history my-cluster vpn

  # View ArgoCD operations
  sloth-kubernetes history my-cluster argocd

  # View addons operations
  sloth-kubernetes history my-cluster addons

  # View Salt operations
  sloth-kubernetes history my-cluster salt

  # View validation results
  sloth-kubernetes history my-cluster validation

  # Output as JSON
  sloth-kubernetes history my-cluster --json`,
	RunE: runHistory,
}

var (
	historyJSON  bool
	historyLimit int
)

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.Flags().BoolVar(&historyJSON, "json", false, "Output in JSON format")
	historyCmd.Flags().IntVar(&historyLimit, "limit", 10, "Number of records to show per type")
}

func runHistory(cmd *cobra.Command, args []string) error {
	// Require stack name as first argument
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	// Get operation type if specified
	operationType := ""
	if len(args) > 1 {
		operationType = args[1]
	}

	// Get history from stack
	history, err := operations.GetOperationsHistory(targetStack)
	if err != nil {
		return fmt.Errorf("failed to get operations history: %w", err)
	}

	if historyJSON {
		return printHistoryJSON(history, operationType)
	}

	printHeader(fmt.Sprintf("📜 Operations History: %s", targetStack))
	fmt.Println()

	totalOps := history.TotalOperations()
	if totalOps == 0 {
		color.Yellow("No operations recorded yet")
		fmt.Println()
		color.Cyan("Operations are recorded automatically when you run:")
		fmt.Println("  • sloth-kubernetes backup create/restore/delete")
		fmt.Println("  • sloth-kubernetes upgrade apply/rollback")
		fmt.Println("  • sloth-kubernetes health")
		fmt.Println("  • sloth-kubernetes benchmark run")
		fmt.Println("  • sloth-kubernetes nodes add/remove/drain")
		fmt.Println("  • sloth-kubernetes vpn join/leave/test")
		fmt.Println("  • sloth-kubernetes argocd install/sync")
		fmt.Println("  • sloth-kubernetes addons bootstrap/install")
		fmt.Println("  • sloth-kubernetes salt cmd/state")
		fmt.Println("  • sloth-kubernetes validate")
		return nil
	}

	fmt.Printf("Last updated: %s\n", history.LastUpdated.Format(time.RFC3339))
	fmt.Printf("Total operations: %d\n\n", totalOps)

	switch operationType {
	case "backups", "backup":
		printBackupHistory(history.BackupHistory)
	case "upgrades", "upgrade":
		printUpgradeHistory(history.UpgradeHistory)
	case "health":
		printHealthHistory(history.HealthHistory)
	case "benchmarks", "benchmark":
		printBenchmarkHistory(history.BenchmarkHistory)
	case "nodes", "node":
		printNodeHistory(history.NodeHistory)
	case "vpn":
		printVPNHistory(history.VPNHistory)
	case "argocd", "argo":
		printArgoCDHistory(history.ArgoCDHistory)
	case "addons", "addon":
		printAddonsHistory(history.AddonsHistory)
	case "salt":
		printSaltHistory(history.SaltHistory)
	case "validation", "validate":
		printValidationHistory(history.ValidationHistory)
	case "":
		// Show all types
		printAllHistory(history)
	default:
		return fmt.Errorf("unknown operation type: %s (valid: backups, upgrades, health, benchmarks, nodes, vpn, argocd, addons, salt, validation)", operationType)
	}

	return nil
}

func printHistoryJSON(history *operations.OperationsHistory, operationType string) error {
	var output interface{}

	switch operationType {
	case "backups", "backup":
		output = history.BackupHistory
	case "upgrades", "upgrade":
		output = history.UpgradeHistory
	case "health":
		output = history.HealthHistory
	case "benchmarks", "benchmark":
		output = history.BenchmarkHistory
	case "nodes", "node":
		output = history.NodeHistory
	case "vpn":
		output = history.VPNHistory
	case "argocd", "argo":
		output = history.ArgoCDHistory
	case "addons", "addon":
		output = history.AddonsHistory
	case "salt":
		output = history.SaltHistory
	case "validation", "validate":
		output = history.ValidationHistory
	case "":
		output = history
	default:
		return fmt.Errorf("unknown operation type: %s", operationType)
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func printAllHistory(history *operations.OperationsHistory) {
	// Summary counts
	color.New(color.Bold).Println("Summary:")
	fmt.Printf("  Backups:     %d records\n", len(history.BackupHistory))
	fmt.Printf("  Upgrades:    %d records\n", len(history.UpgradeHistory))
	fmt.Printf("  Health:      %d records\n", len(history.HealthHistory))
	fmt.Printf("  Benchmarks:  %d records\n", len(history.BenchmarkHistory))
	fmt.Printf("  Nodes:       %d records\n", len(history.NodeHistory))
	fmt.Printf("  VPN:         %d records\n", len(history.VPNHistory))
	fmt.Printf("  ArgoCD:      %d records\n", len(history.ArgoCDHistory))
	fmt.Printf("  Addons:      %d records\n", len(history.AddonsHistory))
	fmt.Printf("  Salt:        %d records\n", len(history.SaltHistory))
	fmt.Printf("  Validation:  %d records\n", len(history.ValidationHistory))
	fmt.Println()

	// Show recent from each category
	if len(history.BackupHistory) > 0 {
		printBackupHistory(limitSlice(history.BackupHistory, historyLimit))
	}

	if len(history.UpgradeHistory) > 0 {
		printUpgradeHistory(limitSlice(history.UpgradeHistory, historyLimit))
	}

	if len(history.HealthHistory) > 0 {
		printHealthHistory(limitSlice(history.HealthHistory, historyLimit))
	}

	if len(history.BenchmarkHistory) > 0 {
		printBenchmarkHistory(limitSlice(history.BenchmarkHistory, historyLimit))
	}

	if len(history.NodeHistory) > 0 {
		printNodeHistory(limitSlice(history.NodeHistory, historyLimit))
	}

	if len(history.VPNHistory) > 0 {
		printVPNHistory(limitSlice(history.VPNHistory, historyLimit))
	}

	if len(history.ArgoCDHistory) > 0 {
		printArgoCDHistory(limitSlice(history.ArgoCDHistory, historyLimit))
	}

	if len(history.AddonsHistory) > 0 {
		printAddonsHistory(limitSlice(history.AddonsHistory, historyLimit))
	}

	if len(history.SaltHistory) > 0 {
		printSaltHistory(limitSlice(history.SaltHistory, historyLimit))
	}

	if len(history.ValidationHistory) > 0 {
		printValidationHistory(limitSlice(history.ValidationHistory, historyLimit))
	}
}

func printBackupHistory(entries []operations.BackupEntry) {
	if len(entries) == 0 {
		color.Yellow("No backup operations recorded")
		fmt.Println()
		return
	}

	color.New(color.Bold).Printf("Backup Operations (%d):\n", len(entries))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "TIMESTAMP\tOPERATION\tNAME\tSTATUS\tDURATION")
	fmt.Fprintln(w, "---------\t---------\t----\t------\t--------")

	// Show most recent first (reverse order)
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		status := formatStatus(e.Status)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			formatTimestamp(e.Timestamp),
			e.Operation,
			truncateString(e.BackupName, 30),
			status,
			e.Duration,
		)
	}
	fmt.Println()
}

func printUpgradeHistory(entries []operations.UpgradeEntry) {
	if len(entries) == 0 {
		color.Yellow("No upgrade operations recorded")
		fmt.Println()
		return
	}

	color.New(color.Bold).Printf("Upgrade Operations (%d):\n", len(entries))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "TIMESTAMP\tOPERATION\tFROM\tTO\tSTATUS\tNODES")
	fmt.Fprintln(w, "---------\t---------\t----\t--\t------\t-----")

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		status := formatStatus(e.Status)
		nodes := fmt.Sprintf("%d/%d", e.NodesOK, e.NodesTotal)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			formatTimestamp(e.Timestamp),
			e.Operation,
			e.FromVersion,
			e.ToVersion,
			status,
			nodes,
		)
	}
	fmt.Println()
}

func printHealthHistory(entries []operations.HealthEntry) {
	if len(entries) == 0 {
		color.Yellow("No health check operations recorded")
		fmt.Println()
		return
	}

	color.New(color.Bold).Printf("Health Check History (%d):\n", len(entries))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "TIMESTAMP\tSTATUS\tPASSED\tWARN\tFAIL\tDURATION")
	fmt.Fprintln(w, "---------\t------\t------\t----\t----\t--------")

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		status := formatHealthStatus(e.OverallStatus)
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%s\n",
			formatTimestamp(e.Timestamp),
			status,
			e.ChecksPassed,
			e.ChecksWarning,
			e.ChecksFailed,
			e.Duration,
		)
	}
	fmt.Println()
}

func printBenchmarkHistory(entries []operations.BenchmarkEntry) {
	if len(entries) == 0 {
		color.Yellow("No benchmark operations recorded")
		fmt.Println()
		return
	}

	color.New(color.Bold).Printf("Benchmark History (%d):\n", len(entries))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "TIMESTAMP\tTYPE\tSCORE\tGRADE\tNETWORK\tSTORAGE\tCPU\tMEMORY")
	fmt.Fprintln(w, "---------\t----\t-----\t-----\t-------\t-------\t---\t------")

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		grade := formatGrade(e.Grade)
		fmt.Fprintf(w, "%s\t%s\t%.1f\t%s\t%.1f\t%.1f\t%.1f\t%.1f\n",
			formatTimestamp(e.Timestamp),
			e.BenchmarkType,
			e.OverallScore,
			grade,
			e.NetworkScore,
			e.StorageScore,
			e.CPUScore,
			e.MemoryScore,
		)
	}
	fmt.Println()
}

func formatTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

func formatStatus(status string) string {
	switch status {
	case "success", "completed", "passed":
		return color.GreenString("[OK]")
	case "failed", "error":
		return color.RedString("[FAIL]")
	case "warning", "partial":
		return color.YellowString("[WARN]")
	case "in-progress", "running":
		return color.CyanString("[...]")
	default:
		return status
	}
}

func formatHealthStatus(status string) string {
	switch status {
	case "healthy":
		return color.GreenString("HEALTHY")
	case "degraded":
		return color.YellowString("DEGRADED")
	case "unhealthy":
		return color.RedString("UNHEALTHY")
	default:
		return status
	}
}

func formatGrade(grade string) string {
	switch {
	case grade == "A" || grade == "A+":
		return color.GreenString(grade)
	case grade == "B" || grade == "B+":
		return color.GreenString(grade)
	case grade == "C" || grade == "C+":
		return color.YellowString(grade)
	case grade == "D":
		return color.YellowString(grade)
	case grade == "F":
		return color.RedString(grade)
	default:
		return grade
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func limitSlice[T any](slice []T, limit int) []T {
	if len(slice) <= limit {
		return slice
	}
	// Return the most recent entries
	return slice[len(slice)-limit:]
}

func printNodeHistory(entries []operations.NodeEntry) {
	if len(entries) == 0 {
		color.Yellow("No node operations recorded")
		fmt.Println()
		return
	}

	color.New(color.Bold).Printf("Node Operations (%d):\n", len(entries))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "TIMESTAMP\tOPERATION\tNODE\tROLE\tSTATUS\tDURATION")
	fmt.Fprintln(w, "---------\t---------\t----\t----\t------\t--------")

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		status := formatStatus(e.Status)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			formatTimestamp(e.Timestamp),
			e.Operation,
			truncateString(e.NodeName, 25),
			e.NodeRole,
			status,
			e.Duration,
		)
	}
	fmt.Println()
}

func printVPNHistory(entries []operations.VPNEntry) {
	if len(entries) == 0 {
		color.Yellow("No VPN operations recorded")
		fmt.Println()
		return
	}

	color.New(color.Bold).Printf("VPN Operations (%d):\n", len(entries))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "TIMESTAMP\tOPERATION\tNODE\tNETWORK\tSTATUS\tNODES")
	fmt.Fprintln(w, "---------\t---------\t----\t-------\t------\t-----")

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		status := formatStatus(e.Status)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
			formatTimestamp(e.Timestamp),
			e.Operation,
			truncateString(e.NodeName, 20),
			truncateString(e.NetworkID, 15),
			status,
			e.NodesCount,
		)
	}
	fmt.Println()
}

func printArgoCDHistory(entries []operations.ArgoCDEntry) {
	if len(entries) == 0 {
		color.Yellow("No ArgoCD operations recorded")
		fmt.Println()
		return
	}

	color.New(color.Bold).Printf("ArgoCD Operations (%d):\n", len(entries))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "TIMESTAMP\tOPERATION\tAPP\tNAMESPACE\tSTATUS\tSYNC\tHEALTH")
	fmt.Fprintln(w, "---------\t---------\t---\t---------\t------\t----\t------")

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		status := formatStatus(e.Status)
		syncStatus := formatSyncStatus(e.SyncStatus)
		healthState := formatArgoCDHealth(e.HealthState)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			formatTimestamp(e.Timestamp),
			e.Operation,
			truncateString(e.AppName, 20),
			truncateString(e.Namespace, 15),
			status,
			syncStatus,
			healthState,
		)
	}
	fmt.Println()
}

func printAddonsHistory(entries []operations.AddonsEntry) {
	if len(entries) == 0 {
		color.Yellow("No addons operations recorded")
		fmt.Println()
		return
	}

	color.New(color.Bold).Printf("Addons Operations (%d):\n", len(entries))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "TIMESTAMP\tOPERATION\tADDON\tTYPE\tSTATUS\tAPPLIED\tFAILED")
	fmt.Fprintln(w, "---------\t---------\t-----\t----\t------\t-------\t------")

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		status := formatStatus(e.Status)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\n",
			formatTimestamp(e.Timestamp),
			e.Operation,
			truncateString(e.AddonName, 20),
			truncateString(e.AddonType, 15),
			status,
			e.AddonsApplied,
			e.AddonsFailed,
		)
	}
	fmt.Println()
}

func printSaltHistory(entries []operations.SaltEntry) {
	if len(entries) == 0 {
		color.Yellow("No Salt operations recorded")
		fmt.Println()
		return
	}

	color.New(color.Bold).Printf("Salt Operations (%d):\n", len(entries))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "TIMESTAMP\tOPERATION\tTARGET\tFUNCTION\tSTATUS\tOK\tFAIL")
	fmt.Fprintln(w, "---------\t---------\t------\t--------\t------\t--\t----")

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		status := formatStatus(e.Status)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\n",
			formatTimestamp(e.Timestamp),
			e.Operation,
			truncateString(e.Target, 15),
			truncateString(e.Function, 20),
			status,
			e.NodesSuccess,
			e.NodesFailed,
		)
	}
	fmt.Println()
}

func printValidationHistory(entries []operations.ValidationEntry) {
	if len(entries) == 0 {
		color.Yellow("No validation operations recorded")
		fmt.Println()
		return
	}

	color.New(color.Bold).Printf("Validation History (%d):\n", len(entries))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "TIMESTAMP\tTYPE\tSTATUS\tPASSED\tFAILED\tWARN\tDURATION")
	fmt.Fprintln(w, "---------\t----\t------\t------\t------\t----\t--------")

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		status := formatValidationStatus(e.OverallStatus)
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\t%s\n",
			formatTimestamp(e.Timestamp),
			e.ValidationType,
			status,
			e.PassedChecks,
			e.FailedChecks,
			e.WarningChecks,
			e.Duration,
		)
	}
	fmt.Println()
}

func formatSyncStatus(status string) string {
	switch status {
	case "Synced":
		return color.GreenString("Synced")
	case "OutOfSync":
		return color.YellowString("OutOfSync")
	case "Unknown":
		return color.CyanString("Unknown")
	default:
		return status
	}
}

func formatArgoCDHealth(status string) string {
	switch status {
	case "Healthy":
		return color.GreenString("Healthy")
	case "Progressing":
		return color.CyanString("Progress")
	case "Degraded":
		return color.YellowString("Degraded")
	case "Suspended":
		return color.YellowString("Suspend")
	case "Missing":
		return color.RedString("Missing")
	default:
		return status
	}
}

func formatValidationStatus(status string) string {
	switch status {
	case "passed":
		return color.GreenString("PASSED")
	case "failed":
		return color.RedString("FAILED")
	case "warning":
		return color.YellowString("WARNING")
	default:
		return status
	}
}
