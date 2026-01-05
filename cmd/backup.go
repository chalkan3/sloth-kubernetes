package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/pkg/backup"
	"github.com/chalkan3/sloth-kubernetes/pkg/operations"
)

var (
	backupKubeconfig      string
	backupNamespaces      []string
	backupExcludeNS       []string
	backupResources       []string
	backupExcludeRes      []string
	backupTTL             string
	backupStorageLocation string
	backupLabels          []string
	backupSnapshotVolumes bool
	backupWait            bool
	backupTimeout         time.Duration
	backupOutputJSON      bool
	backupVeleroNS        string
	backupDryRun          bool

	// Restore flags
	restoreFromBackup string
	restorePVs        bool
	restorePreserveNP bool

	// Schedule flags
	scheduleExpression string

	// Install flags
	installProvider   string
	installBucket     string
	installRegion     string
	installSecretFile string
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage cluster backups with Velero",
	Long: `Manage Kubernetes cluster backups using Velero.

This command provides comprehensive backup and restore capabilities:
  - Create full cluster backups
  - Create namespace-specific backups
  - Restore from backups
  - Schedule automated backups
  - Manage backup storage locations

Velero must be installed in your cluster for backup operations to work.`,
	Example: `  # Create a full cluster backup
  sloth-kubernetes backup create my-backup

  # Create a namespace-specific backup
  sloth-kubernetes backup create my-backup --namespaces default,app

  # List all backups
  sloth-kubernetes backup list

  # Restore from a backup
  sloth-kubernetes backup restore --from-backup my-backup

  # Create a scheduled backup (daily at midnight)
  sloth-kubernetes backup schedule create daily-backup --schedule "0 0 * * *"`,
}

var backupCreateCmd = &cobra.Command{
	Use:   "create [stack-name] [backup-name]",
	Short: "Create a new backup",
	Long:  `Create a new Velero backup of the cluster or specific namespaces.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runBackupCreate,
}

var backupListCmd = &cobra.Command{
	Use:   "list [stack-name]",
	Short: "List all backups",
	Long:  `List all Velero backups in the cluster.`,
	RunE:  runBackupList,
}

var backupDescribeCmd = &cobra.Command{
	Use:   "describe [stack-name] [backup-name]",
	Short: "Describe a backup",
	Long:  `Show detailed information about a specific backup.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runBackupDescribe,
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete [stack-name] [backup-name]",
	Short: "Delete a backup",
	Long:  `Delete a Velero backup.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runBackupDelete,
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore [stack-name] [restore-name]",
	Short: "Restore from a backup",
	Long:  `Create a restore from an existing backup.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runBackupRestore,
}

var backupRestoreListCmd = &cobra.Command{
	Use:   "restore-list [stack-name]",
	Short: "List all restores",
	Long:  `List all Velero restores in the cluster.`,
	RunE:  runRestoreList,
}

var backupScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage backup schedules",
	Long:  `Manage Velero backup schedules for automated backups.`,
}

var backupScheduleCreateCmd = &cobra.Command{
	Use:   "create [stack-name] [schedule-name]",
	Short: "Create a backup schedule",
	Long: `Create a new backup schedule using cron expressions.

Cron expression format: "minute hour day-of-month month day-of-week"
Examples:
  "0 0 * * *"     - Daily at midnight
  "0 */6 * * *"   - Every 6 hours
  "0 0 * * 0"     - Weekly on Sunday at midnight
  "0 0 1 * *"     - Monthly on the 1st at midnight`,
	Args: cobra.ExactArgs(2),
	RunE: runScheduleCreate,
}

var backupScheduleListCmd = &cobra.Command{
	Use:   "list [stack-name]",
	Short: "List backup schedules",
	Long:  `List all Velero backup schedules.`,
	RunE:  runScheduleList,
}

var backupScheduleDeleteCmd = &cobra.Command{
	Use:   "delete [stack-name] [schedule-name]",
	Short: "Delete a backup schedule",
	Long:  `Delete a Velero backup schedule.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runScheduleDelete,
}

var backupSchedulePauseCmd = &cobra.Command{
	Use:   "pause [stack-name] [schedule-name]",
	Short: "Pause a backup schedule",
	Long:  `Pause a Velero backup schedule.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runSchedulePause,
}

var backupScheduleUnpauseCmd = &cobra.Command{
	Use:   "unpause [stack-name] [schedule-name]",
	Short: "Unpause a backup schedule",
	Long:  `Unpause a paused Velero backup schedule.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runScheduleUnpause,
}

var backupLocationsCmd = &cobra.Command{
	Use:   "locations [stack-name]",
	Short: "List backup storage locations",
	Long:  `List all configured backup storage locations.`,
	RunE:  runBackupLocations,
}

var backupInstallCmd = &cobra.Command{
	Use:   "install [stack-name]",
	Short: "Install Velero",
	Long: `Install Velero in the cluster with the specified storage provider.

Supported providers:
  - aws       Amazon S3
  - gcp       Google Cloud Storage
  - azure     Azure Blob Storage
  - minio     MinIO (S3-compatible)`,
	RunE: runBackupInstall,
}

var backupStatusCmd = &cobra.Command{
	Use:   "status [stack-name]",
	Short: "Show Velero status",
	Long:  `Check if Velero is installed and show its status.`,
	RunE:  runBackupStatus,
}

func init() {
	rootCmd.AddCommand(backupCmd)

	// Subcommands
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupDescribeCmd)
	backupCmd.AddCommand(backupDeleteCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupRestoreListCmd)
	backupCmd.AddCommand(backupScheduleCmd)
	backupCmd.AddCommand(backupLocationsCmd)
	backupCmd.AddCommand(backupInstallCmd)
	backupCmd.AddCommand(backupStatusCmd)

	// Schedule subcommands
	backupScheduleCmd.AddCommand(backupScheduleCreateCmd)
	backupScheduleCmd.AddCommand(backupScheduleListCmd)
	backupScheduleCmd.AddCommand(backupScheduleDeleteCmd)
	backupScheduleCmd.AddCommand(backupSchedulePauseCmd)
	backupScheduleCmd.AddCommand(backupScheduleUnpauseCmd)

	// Global backup flags
	backupCmd.PersistentFlags().StringVar(&backupKubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	backupCmd.PersistentFlags().StringVar(&backupVeleroNS, "velero-namespace", "velero", "Velero namespace")
	backupCmd.PersistentFlags().BoolVar(&backupOutputJSON, "json", false, "Output in JSON format")

	// Create flags
	backupCreateCmd.Flags().StringSliceVar(&backupNamespaces, "namespaces", nil, "Namespaces to include in backup")
	backupCreateCmd.Flags().StringSliceVar(&backupExcludeNS, "exclude-namespaces", nil, "Namespaces to exclude from backup")
	backupCreateCmd.Flags().StringSliceVar(&backupResources, "resources", nil, "Resources to include in backup")
	backupCreateCmd.Flags().StringSliceVar(&backupExcludeRes, "exclude-resources", nil, "Resources to exclude from backup")
	backupCreateCmd.Flags().StringVar(&backupTTL, "ttl", "720h", "Backup retention period (default 30 days)")
	backupCreateCmd.Flags().StringVar(&backupStorageLocation, "storage-location", "", "Backup storage location name")
	backupCreateCmd.Flags().StringSliceVar(&backupLabels, "labels", nil, "Labels to apply to backup (key=value)")
	backupCreateCmd.Flags().BoolVar(&backupSnapshotVolumes, "snapshot-volumes", true, "Take snapshots of PVs")
	backupCreateCmd.Flags().BoolVar(&backupWait, "wait", false, "Wait for backup to complete")
	backupCreateCmd.Flags().DurationVar(&backupTimeout, "timeout", 30*time.Minute, "Timeout when using --wait")
	backupCreateCmd.Flags().BoolVar(&backupDryRun, "dry-run", false, "Show what would be backed up without creating backup")

	// Delete flags
	backupDeleteCmd.Flags().BoolVar(&backupDryRun, "dry-run", false, "Show what would be deleted without deleting")

	// Restore flags
	backupRestoreCmd.Flags().StringVar(&restoreFromBackup, "from-backup", "", "Backup name to restore from (required)")
	backupRestoreCmd.Flags().StringSliceVar(&backupNamespaces, "namespaces", nil, "Namespaces to restore")
	backupRestoreCmd.Flags().StringSliceVar(&backupExcludeNS, "exclude-namespaces", nil, "Namespaces to exclude from restore")
	backupRestoreCmd.Flags().StringSliceVar(&backupResources, "resources", nil, "Resources to restore")
	backupRestoreCmd.Flags().StringSliceVar(&backupExcludeRes, "exclude-resources", nil, "Resources to exclude from restore")
	backupRestoreCmd.Flags().BoolVar(&restorePVs, "restore-volumes", true, "Restore persistent volumes")
	backupRestoreCmd.Flags().BoolVar(&restorePreserveNP, "preserve-nodeports", false, "Preserve original NodePort values")
	backupRestoreCmd.Flags().BoolVar(&backupWait, "wait", false, "Wait for restore to complete")
	backupRestoreCmd.Flags().DurationVar(&backupTimeout, "timeout", 30*time.Minute, "Timeout when using --wait")
	backupRestoreCmd.Flags().BoolVar(&backupDryRun, "dry-run", false, "Show what would be restored without restoring")

	// Schedule create flags
	backupScheduleCreateCmd.Flags().StringVar(&scheduleExpression, "schedule", "", "Cron expression for schedule (required)")
	backupScheduleCreateCmd.Flags().StringSliceVar(&backupNamespaces, "namespaces", nil, "Namespaces to include")
	backupScheduleCreateCmd.Flags().StringSliceVar(&backupExcludeNS, "exclude-namespaces", nil, "Namespaces to exclude")
	backupScheduleCreateCmd.Flags().StringVar(&backupTTL, "ttl", "720h", "Backup retention period")
	backupScheduleCreateCmd.Flags().BoolVar(&backupSnapshotVolumes, "snapshot-volumes", true, "Take snapshots of PVs")
	backupScheduleCreateCmd.Flags().BoolVar(&backupDryRun, "dry-run", false, "Show what schedule would be created without creating")
	backupScheduleCreateCmd.MarkFlagRequired("schedule")

	// Schedule delete flags
	backupScheduleDeleteCmd.Flags().BoolVar(&backupDryRun, "dry-run", false, "Show what would be deleted without deleting")

	// Install flags
	backupInstallCmd.Flags().StringVar(&installProvider, "provider", "", "Storage provider (aws, gcp, azure, minio)")
	backupInstallCmd.Flags().StringVar(&installBucket, "bucket", "", "Storage bucket name")
	backupInstallCmd.Flags().StringVar(&installRegion, "region", "", "Storage region")
	backupInstallCmd.Flags().StringVar(&installSecretFile, "secret-file", "", "Path to credentials file")
	backupInstallCmd.MarkFlagRequired("provider")
	backupInstallCmd.MarkFlagRequired("bucket")
	backupInstallCmd.MarkFlagRequired("secret-file")
}

func createBackupManager(targetStack string) (*backup.Manager, error) {
	// Get kubeconfig from stack
	kubeconfigPath, err := GetKubeconfigFromStack(targetStack)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig from stack '%s': %w", targetStack, err)
	}

	manager := backup.NewManager(kubeconfigPath)
	manager.SetNamespace(backupVeleroNS)
	return manager, nil
}

func runBackupCreate(cmd *cobra.Command, args []string) error {
	printHeader("Create Backup")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	// Check Velero is installed
	installed, err := manager.CheckVeleroInstalled()
	if err != nil || !installed {
		return fmt.Errorf("velero is not installed. Run 'sloth-kubernetes backup install %s' first", targetStack)
	}

	config := backup.BackupConfig{
		IncludedNamespaces: backupNamespaces,
		ExcludedNamespaces: backupExcludeNS,
		IncludedResources:  backupResources,
		ExcludedResources:  backupExcludeRes,
		TTL:                backupTTL,
		StorageLocation:    backupStorageLocation,
		SnapshotVolumes:    backupSnapshotVolumes,
		Wait:               backupWait,
		Timeout:            backupTimeout,
	}

	// Backup name is second argument (optional)
	if len(args) > 1 {
		config.Name = args[1]
	}

	// Parse labels
	if len(backupLabels) > 0 {
		config.Labels = make(map[string]string)
		for _, label := range backupLabels {
			parts := strings.SplitN(label, "=", 2)
			if len(parts) == 2 {
				config.Labels[parts[0]] = parts[1]
			}
		}
	}

	// Dry-run mode: show what would be backed up without creating
	if backupDryRun {
		fmt.Println()
		color.Yellow("[DRY-RUN] Would create backup with the following configuration:")
		fmt.Println()
		fmt.Printf("  Name:              %s\n", config.Name)
		if len(config.IncludedNamespaces) > 0 {
			fmt.Printf("  Namespaces:        %s\n", strings.Join(config.IncludedNamespaces, ", "))
		} else {
			fmt.Printf("  Namespaces:        All\n")
		}
		if len(config.ExcludedNamespaces) > 0 {
			fmt.Printf("  Excluded NS:       %s\n", strings.Join(config.ExcludedNamespaces, ", "))
		}
		if len(config.IncludedResources) > 0 {
			fmt.Printf("  Resources:         %s\n", strings.Join(config.IncludedResources, ", "))
		}
		if len(config.ExcludedResources) > 0 {
			fmt.Printf("  Excluded Resources: %s\n", strings.Join(config.ExcludedResources, ", "))
		}
		fmt.Printf("  TTL:               %s\n", config.TTL)
		fmt.Printf("  Snapshot Volumes:  %v\n", config.SnapshotVolumes)
		if config.StorageLocation != "" {
			fmt.Printf("  Storage Location:  %s\n", config.StorageLocation)
		}
		if len(config.Labels) > 0 {
			fmt.Printf("  Labels:            %v\n", config.Labels)
		}
		fmt.Println()
		color.Cyan("No backup was created (dry-run mode)")
		return nil
	}

	fmt.Println()
	color.Cyan("Creating backup...")

	startTime := time.Now()
	result, err := manager.CreateBackup(config)
	if err != nil {
		// Record failed backup
		operations.RecordBackupOperation(targetStack, "create", config.Name, "failed", config.IncludedNamespaces, time.Since(startTime), err)
		return err
	}

	if backupWait {
		color.Cyan("Waiting for backup to complete...")
		result, err = manager.WaitForBackup(result.Name, backupTimeout)
		if err != nil {
			operations.RecordBackupOperation(targetStack, "create", result.Name, "failed", config.IncludedNamespaces, time.Since(startTime), err)
			return err
		}
	}

	// Record successful backup
	status := "success"
	if result.Status == "PartiallyFailed" {
		status = "partial"
	}
	operations.RecordBackupOperation(targetStack, "create", result.Name, status, result.IncludedNamespaces, time.Since(startTime), nil)

	if backupOutputJSON {
		return outputBackupJSON(result)
	}

	fmt.Println()
	printBackupDetails(result)

	return nil
}

func runBackupList(cmd *cobra.Command, args []string) error {
	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	backups, err := manager.ListBackups()
	if err != nil {
		return err
	}

	if backupOutputJSON {
		return outputBackupJSON(backups)
	}

	printHeader("Backups")
	fmt.Println()

	if len(backups) == 0 {
		color.Yellow("No backups found")
		return nil
	}

	fmt.Printf("%-30s %-15s %-10s %-10s %-20s\n", "NAME", "STATUS", "ERRORS", "WARNINGS", "CREATED")
	fmt.Println(strings.Repeat("-", 90))

	for _, b := range backups {
		statusColor := getBackupStatusColor(string(b.Status))
		created := b.StartTimestamp.Format("2006-01-02 15:04:05")
		if b.StartTimestamp.IsZero() {
			created = "-"
		}
		fmt.Printf("%-30s ", b.Name)
		statusColor.Printf("%-15s ", b.Status)
		fmt.Printf("%-10d %-10d %-20s\n", b.Errors, b.Warnings, created)
	}

	fmt.Printf("\nTotal: %d backups\n", len(backups))
	return nil
}

func runBackupDescribe(cmd *cobra.Command, args []string) error {
	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	// Backup name is second argument
	b, err := manager.GetBackup(args[1])
	if err != nil {
		return err
	}

	if backupOutputJSON {
		return outputBackupJSON(b)
	}

	printHeader("Backup Details")
	printBackupDetails(b)

	return nil
}

func runBackupDelete(cmd *cobra.Command, args []string) error {
	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	// Backup name is second argument
	backupName := args[1]

	// Dry-run mode: show what would be deleted without deleting
	if backupDryRun {
		// Try to get backup details to show
		b, err := manager.GetBackup(backupName)
		if err != nil {
			return fmt.Errorf("backup %s not found: %w", backupName, err)
		}

		fmt.Println()
		color.Yellow("[DRY-RUN] Would delete the following backup:")
		fmt.Println()
		fmt.Printf("  Name:       %s\n", b.Name)
		fmt.Printf("  Status:     %s\n", b.Status)
		if !b.StartTimestamp.IsZero() {
			fmt.Printf("  Created:    %s\n", b.StartTimestamp.Format("2006-01-02 15:04:05"))
		}
		if len(b.IncludedNamespaces) > 0 {
			fmt.Printf("  Namespaces: %s\n", strings.Join(b.IncludedNamespaces, ", "))
		}
		fmt.Println()
		color.Cyan("No backup was deleted (dry-run mode)")
		return nil
	}

	color.Yellow("Deleting backup %s...", backupName)

	startTime := time.Now()
	if err := manager.DeleteBackup(backupName); err != nil {
		operations.RecordBackupOperation(targetStack, "delete", backupName, "failed", nil, time.Since(startTime), err)
		return err
	}

	operations.RecordBackupOperation(targetStack, "delete", backupName, "success", nil, time.Since(startTime), nil)
	color.Green("Backup %s deleted successfully", backupName)
	return nil
}

func runBackupRestore(cmd *cobra.Command, args []string) error {
	printHeader("Restore from Backup")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	if restoreFromBackup == "" {
		return fmt.Errorf("--from-backup flag is required")
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	// Check Velero is installed
	installed, err := manager.CheckVeleroInstalled()
	if err != nil || !installed {
		return fmt.Errorf("velero is not installed")
	}

	config := backup.RestoreConfig{
		BackupName:         restoreFromBackup,
		IncludedNamespaces: backupNamespaces,
		ExcludedNamespaces: backupExcludeNS,
		IncludedResources:  backupResources,
		ExcludedResources:  backupExcludeRes,
		RestorePVs:         restorePVs,
		PreserveNodePorts:  restorePreserveNP,
		Wait:               backupWait,
		Timeout:            backupTimeout,
	}

	// Restore name is second argument (optional)
	if len(args) > 1 {
		config.Name = args[1]
	}

	// Dry-run mode: show what would be restored without restoring
	if backupDryRun {
		// Get backup details to show what would be restored
		b, err := manager.GetBackup(restoreFromBackup)
		if err != nil {
			return fmt.Errorf("backup %s not found: %w", restoreFromBackup, err)
		}

		fmt.Println()
		color.Yellow("[DRY-RUN] Would restore from backup with the following configuration:")
		fmt.Println()
		fmt.Printf("  Backup:            %s\n", restoreFromBackup)
		fmt.Printf("  Backup Status:     %s\n", b.Status)
		if !b.StartTimestamp.IsZero() {
			fmt.Printf("  Backup Created:    %s\n", b.StartTimestamp.Format("2006-01-02 15:04:05"))
		}
		if len(config.IncludedNamespaces) > 0 {
			fmt.Printf("  Namespaces:        %s\n", strings.Join(config.IncludedNamespaces, ", "))
		} else if len(b.IncludedNamespaces) > 0 {
			fmt.Printf("  Namespaces:        %s (from backup)\n", strings.Join(b.IncludedNamespaces, ", "))
		} else {
			fmt.Printf("  Namespaces:        All\n")
		}
		if len(config.ExcludedNamespaces) > 0 {
			fmt.Printf("  Excluded NS:       %s\n", strings.Join(config.ExcludedNamespaces, ", "))
		}
		if len(config.IncludedResources) > 0 {
			fmt.Printf("  Resources:         %s\n", strings.Join(config.IncludedResources, ", "))
		}
		if len(config.ExcludedResources) > 0 {
			fmt.Printf("  Excluded Resources: %s\n", strings.Join(config.ExcludedResources, ", "))
		}
		fmt.Printf("  Restore PVs:       %v\n", config.RestorePVs)
		fmt.Printf("  Preserve NodePorts: %v\n", config.PreserveNodePorts)
		fmt.Println()
		color.Cyan("No restore was created (dry-run mode)")
		return nil
	}

	fmt.Println()
	color.Cyan("Creating restore from backup %s...", restoreFromBackup)

	startTime := time.Now()
	result, err := manager.CreateRestore(config)
	if err != nil {
		operations.RecordBackupOperation(targetStack, "restore", restoreFromBackup, "failed", config.IncludedNamespaces, time.Since(startTime), err)
		return err
	}

	if backupWait {
		color.Cyan("Waiting for restore to complete...")
		result, err = manager.WaitForRestore(result.Name, backupTimeout)
		if err != nil {
			operations.RecordBackupOperation(targetStack, "restore", restoreFromBackup, "failed", config.IncludedNamespaces, time.Since(startTime), err)
			return err
		}
	}

	// Record successful restore
	status := "success"
	if result.Status == "PartiallyFailed" {
		status = "partial"
	}
	operations.RecordBackupOperation(targetStack, "restore", result.Name, status, result.IncludedNamespaces, time.Since(startTime), nil)

	if backupOutputJSON {
		return outputBackupJSON(result)
	}

	fmt.Println()
	printRestoreDetails(result)

	return nil
}

func runRestoreList(cmd *cobra.Command, args []string) error {
	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	restores, err := manager.ListRestores()
	if err != nil {
		return err
	}

	if backupOutputJSON {
		return outputBackupJSON(restores)
	}

	printHeader("Restores")
	fmt.Println()

	if len(restores) == 0 {
		color.Yellow("No restores found")
		return nil
	}

	fmt.Printf("%-30s %-25s %-15s %-10s %-10s\n", "NAME", "BACKUP", "STATUS", "ERRORS", "WARNINGS")
	fmt.Println(strings.Repeat("-", 95))

	for _, r := range restores {
		statusColor := getBackupStatusColor(string(r.Status))
		fmt.Printf("%-30s %-25s ", r.Name, r.BackupName)
		statusColor.Printf("%-15s ", r.Status)
		fmt.Printf("%-10d %-10d\n", r.Errors, r.Warnings)
	}

	fmt.Printf("\nTotal: %d restores\n", len(restores))
	return nil
}

func runScheduleCreate(cmd *cobra.Command, args []string) error {
	printHeader("Create Backup Schedule")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	// Check Velero is installed
	installed, err := manager.CheckVeleroInstalled()
	if err != nil || !installed {
		return fmt.Errorf("velero is not installed")
	}

	// Schedule name is second argument
	config := backup.ScheduleConfig{
		Name:               args[1],
		Schedule:           scheduleExpression,
		IncludedNamespaces: backupNamespaces,
		ExcludedNamespaces: backupExcludeNS,
		TTL:                backupTTL,
		SnapshotVolumes:    backupSnapshotVolumes,
	}

	// Dry-run mode: show what schedule would be created without creating
	if backupDryRun {
		fmt.Println()
		color.Yellow("[DRY-RUN] Would create backup schedule with the following configuration:")
		fmt.Println()
		fmt.Printf("  Name:              %s\n", config.Name)
		fmt.Printf("  Schedule:          %s\n", config.Schedule)
		if len(config.IncludedNamespaces) > 0 {
			fmt.Printf("  Namespaces:        %s\n", strings.Join(config.IncludedNamespaces, ", "))
		} else {
			fmt.Printf("  Namespaces:        All\n")
		}
		if len(config.ExcludedNamespaces) > 0 {
			fmt.Printf("  Excluded NS:       %s\n", strings.Join(config.ExcludedNamespaces, ", "))
		}
		fmt.Printf("  TTL:               %s\n", config.TTL)
		fmt.Printf("  Snapshot Volumes:  %v\n", config.SnapshotVolumes)
		fmt.Println()
		color.Cyan("No schedule was created (dry-run mode)")
		return nil
	}

	fmt.Println()
	color.Cyan("Creating backup schedule...")

	startTime := time.Now()
	result, err := manager.CreateSchedule(config)
	if err != nil {
		operations.RecordBackupOperation(targetStack, "schedule-create", config.Name, "failed", config.IncludedNamespaces, time.Since(startTime), err)
		return err
	}

	operations.RecordBackupOperation(targetStack, "schedule-create", result.Name, "success", config.IncludedNamespaces, time.Since(startTime), nil)

	if backupOutputJSON {
		return outputBackupJSON(result)
	}

	fmt.Println()
	color.Green("Schedule created successfully!")
	fmt.Printf("\n  Name:     %s\n", result.Name)
	fmt.Printf("  Schedule: %s\n", result.Schedule)

	return nil
}

func runScheduleList(cmd *cobra.Command, args []string) error {
	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	schedules, err := manager.ListSchedules()
	if err != nil {
		return err
	}

	if backupOutputJSON {
		return outputBackupJSON(schedules)
	}

	printHeader("Backup Schedules")
	fmt.Println()

	if len(schedules) == 0 {
		color.Yellow("No schedules found")
		return nil
	}

	fmt.Printf("%-25s %-20s %-10s %-25s\n", "NAME", "SCHEDULE", "PAUSED", "LAST BACKUP")
	fmt.Println(strings.Repeat("-", 85))

	for _, s := range schedules {
		paused := "No"
		if s.Paused {
			paused = "Yes"
		}
		lastBackup := s.LastBackup.Format("2006-01-02 15:04:05")
		if s.LastBackup.IsZero() {
			lastBackup = "-"
		}
		fmt.Printf("%-25s %-20s %-10s %-25s\n", s.Name, s.Schedule, paused, lastBackup)
	}

	fmt.Printf("\nTotal: %d schedules\n", len(schedules))
	return nil
}

func runScheduleDelete(cmd *cobra.Command, args []string) error {
	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	// Schedule name is second argument
	scheduleName := args[1]

	// Dry-run mode: show what would be deleted without deleting
	if backupDryRun {
		// Try to get schedule details
		schedules, err := manager.ListSchedules()
		if err != nil {
			return fmt.Errorf("failed to get schedules: %w", err)
		}

		var found *backup.Schedule
		for i, s := range schedules {
			if s.Name == scheduleName {
				found = &schedules[i]
				break
			}
		}

		if found == nil {
			return fmt.Errorf("schedule %s not found", scheduleName)
		}

		fmt.Println()
		color.Yellow("[DRY-RUN] Would delete the following schedule:")
		fmt.Println()
		fmt.Printf("  Name:       %s\n", found.Name)
		fmt.Printf("  Schedule:   %s\n", found.Schedule)
		fmt.Printf("  Paused:     %v\n", found.Paused)
		if !found.LastBackup.IsZero() {
			fmt.Printf("  Last Backup: %s\n", found.LastBackup.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
		color.Cyan("No schedule was deleted (dry-run mode)")
		return nil
	}

	color.Yellow("Deleting schedule %s...", scheduleName)

	startTime := time.Now()
	if err := manager.DeleteSchedule(scheduleName); err != nil {
		operations.RecordBackupOperation(targetStack, "schedule-delete", scheduleName, "failed", nil, time.Since(startTime), err)
		return err
	}

	operations.RecordBackupOperation(targetStack, "schedule-delete", scheduleName, "success", nil, time.Since(startTime), nil)
	color.Green("Schedule %s deleted successfully", scheduleName)
	return nil
}

func runSchedulePause(cmd *cobra.Command, args []string) error {
	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	// Schedule name is second argument
	scheduleName := args[1]

	if err := manager.PauseSchedule(scheduleName); err != nil {
		return err
	}

	color.Green("Schedule %s paused", scheduleName)
	return nil
}

func runScheduleUnpause(cmd *cobra.Command, args []string) error {
	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	// Schedule name is second argument
	scheduleName := args[1]

	if err := manager.UnpauseSchedule(scheduleName); err != nil {
		return err
	}

	color.Green("Schedule %s unpaused", scheduleName)
	return nil
}

func runBackupLocations(cmd *cobra.Command, args []string) error {
	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	locations, err := manager.GetBackupLocations()
	if err != nil {
		return err
	}

	if backupOutputJSON {
		return outputBackupJSON(locations)
	}

	printHeader("Backup Storage Locations")
	fmt.Println()

	if len(locations) == 0 {
		color.Yellow("No backup storage locations found")
		return nil
	}

	fmt.Printf("%-20s %-15s %-25s %-15s %-10s\n", "NAME", "PROVIDER", "BUCKET", "REGION", "DEFAULT")
	fmt.Println(strings.Repeat("-", 90))

	for _, l := range locations {
		isDefault := "No"
		if l.Default {
			isDefault = "Yes"
		}
		fmt.Printf("%-20s %-15s %-25s %-15s %-10s\n", l.Name, l.Provider, l.Bucket, l.Region, isDefault)
	}

	return nil
}

func runBackupInstall(cmd *cobra.Command, args []string) error {
	printHeader("Install Velero")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	// Check if already installed
	installed, _ := manager.CheckVeleroInstalled()
	if installed {
		return fmt.Errorf("velero is already installed")
	}

	fmt.Println()
	color.Cyan("Installing Velero with %s provider...", installProvider)
	fmt.Printf("  Bucket: %s\n", installBucket)
	fmt.Printf("  Region: %s\n", installRegion)
	fmt.Println()

	if err := manager.InstallVelero(installProvider, installBucket, installRegion, installSecretFile); err != nil {
		return err
	}

	color.Green("Velero installed successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Create a backup: sloth-kubernetes backup create %s my-backup\n", targetStack)
	fmt.Printf("  2. Create a schedule: sloth-kubernetes backup schedule create %s daily --schedule '0 0 * * *'\n", targetStack)

	return nil
}

func runBackupStatus(cmd *cobra.Command, args []string) error {
	printHeader("Velero Status")

	// Get stack name from args
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	manager, err := createBackupManager(targetStack)
	if err != nil {
		return err
	}

	installed, err := manager.CheckVeleroInstalled()
	if err != nil {
		return fmt.Errorf("failed to check Velero status: %w", err)
	}

	fmt.Println()
	if installed {
		color.Green("[OK] Velero is installed")

		// Get backup locations
		locations, err := manager.GetBackupLocations()
		if err == nil && len(locations) > 0 {
			fmt.Printf("\nBackup Locations: %d configured\n", len(locations))
			for _, l := range locations {
				fmt.Printf("  - %s (%s)\n", l.Name, l.Provider)
			}
		}

		// Get backup count
		backups, err := manager.ListBackups()
		if err == nil {
			fmt.Printf("\nBackups: %d total\n", len(backups))
		}

		// Get schedule count
		schedules, err := manager.ListSchedules()
		if err == nil {
			fmt.Printf("Schedules: %d configured\n", len(schedules))
		}
	} else {
		color.Red("[NOT INSTALLED] Velero is not installed")
		fmt.Println()
		fmt.Println("To install Velero, run:")
		fmt.Println("  sloth-kubernetes backup install --provider <provider> --bucket <bucket> --region <region> --secret-file <file>")
	}

	return nil
}

// Helper functions

func printBackupDetails(b *backup.Backup) {
	statusColor := getBackupStatusColor(string(b.Status))

	fmt.Printf("  Name:       %s\n", b.Name)
	fmt.Printf("  Status:     ")
	statusColor.Printf("%s\n", b.Status)
	fmt.Printf("  Phase:      %s\n", b.Phase)

	if len(b.IncludedNamespaces) > 0 {
		fmt.Printf("  Namespaces: %s\n", strings.Join(b.IncludedNamespaces, ", "))
	} else {
		fmt.Printf("  Namespaces: All\n")
	}

	if !b.StartTimestamp.IsZero() {
		fmt.Printf("  Started:    %s\n", b.StartTimestamp.Format("2006-01-02 15:04:05"))
	}
	if !b.CompletionTimestamp.IsZero() {
		fmt.Printf("  Completed:  %s\n", b.CompletionTimestamp.Format("2006-01-02 15:04:05"))
		duration := b.CompletionTimestamp.Sub(b.StartTimestamp)
		fmt.Printf("  Duration:   %s\n", duration.Round(time.Second))
	}
	if !b.Expiration.IsZero() {
		fmt.Printf("  Expires:    %s\n", b.Expiration.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("  Errors:     %d\n", b.Errors)
	fmt.Printf("  Warnings:   %d\n", b.Warnings)

	if b.StorageLocation != "" {
		fmt.Printf("  Location:   %s\n", b.StorageLocation)
	}
}

func printRestoreDetails(r *backup.Restore) {
	statusColor := getBackupStatusColor(string(r.Status))

	fmt.Printf("  Name:       %s\n", r.Name)
	fmt.Printf("  Backup:     %s\n", r.BackupName)
	fmt.Printf("  Status:     ")
	statusColor.Printf("%s\n", r.Status)
	fmt.Printf("  Phase:      %s\n", r.Phase)

	if len(r.IncludedNamespaces) > 0 {
		fmt.Printf("  Namespaces: %s\n", strings.Join(r.IncludedNamespaces, ", "))
	}

	if !r.StartTimestamp.IsZero() {
		fmt.Printf("  Started:    %s\n", r.StartTimestamp.Format("2006-01-02 15:04:05"))
	}
	if !r.CompletionTimestamp.IsZero() {
		fmt.Printf("  Completed:  %s\n", r.CompletionTimestamp.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("  Errors:     %d\n", r.Errors)
	fmt.Printf("  Warnings:   %d\n", r.Warnings)
}

func getBackupStatusColor(status string) *color.Color {
	switch status {
	case "Completed":
		return color.New(color.FgGreen)
	case "InProgress", "New":
		return color.New(color.FgCyan)
	case "Failed":
		return color.New(color.FgRed)
	case "PartiallyFailed":
		return color.New(color.FgYellow)
	default:
		return color.New(color.FgWhite)
	}
}

func outputBackupJSON(data interface{}) error {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}
