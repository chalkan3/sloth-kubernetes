package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/pkg/salt"
)

// Package Management Commands

var pkgCmd = &cobra.Command{
	Use:   "pkg",
	Short: "Package management operations",
	Long:  `Manage packages on minions (install, remove, upgrade)`,
}

var pkgInstallCmd = &cobra.Command{
	Use:   "install <package...>",
	Short: "Install packages",
	Example: `  sloth-kubernetes salt pkg install nginx
  sloth-kubernetes salt pkg install htop vim git --target "worker*"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runPkgInstall,
}

var pkgRemoveCmd = &cobra.Command{
	Use:   "remove <package...>",
	Short: "Remove packages",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runPkgRemove,
}

var pkgUpgradeCmd = &cobra.Command{
	Use:   "upgrade [package...]",
	Short: "Upgrade packages (all if none specified)",
	RunE:  runPkgUpgrade,
}

var pkgListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	RunE:  runPkgList,
}

// Service Management Commands

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Service management operations",
	Long:  `Manage services on minions (start, stop, restart, status)`,
}

var serviceStartCmd = &cobra.Command{
	Use:   "start <service>",
	Short: "Start a service",
	Example: `  sloth-kubernetes salt service start nginx
  sloth-kubernetes salt service start k3s --target "master*"`,
	Args: cobra.ExactArgs(1),
	RunE: runServiceStart,
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop <service>",
	Short: "Stop a service",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceStop,
}

var serviceRestartCmd = &cobra.Command{
	Use:   "restart <service>",
	Short: "Restart a service",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceRestart,
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status <service>",
	Short: "Check service status",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceStatus,
}

var serviceEnableCmd = &cobra.Command{
	Use:   "enable <service>",
	Short: "Enable service on boot",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceEnable,
}

var serviceDisableCmd = &cobra.Command{
	Use:   "disable <service>",
	Short: "Disable service on boot",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceDisable,
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all services",
	RunE:  runServiceList,
}

// File Management Commands

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "File management operations",
	Long:  `Manage files on minions (read, write, remove, copy)`,
}

var fileReadCmd = &cobra.Command{
	Use:   "read <path>",
	Short: "Read file content",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileRead,
}

var fileWriteCmd = &cobra.Command{
	Use:   "write <path> <content>",
	Short: "Write content to file",
	Args:  cobra.ExactArgs(2),
	RunE:  runFileWrite,
}

var fileRemoveCmd = &cobra.Command{
	Use:   "remove <path>",
	Short: "Remove a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileRemove,
}

var fileExistsCmd = &cobra.Command{
	Use:   "exists <path>",
	Short: "Check if file exists",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileExists,
}

var fileChmodCmd = &cobra.Command{
	Use:     "chmod <path> <mode>",
	Short:   "Change file permissions",
	Example: `  sloth-kubernetes salt file chmod /tmp/test.sh 755`,
	Args:    cobra.ExactArgs(2),
	RunE:    runFileChmod,
}

// System Commands

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "System management operations",
	Long:  `System-level operations (reboot, uptime, disk, memory)`,
}

var systemRebootCmd = &cobra.Command{
	Use:   "reboot",
	Short: "Reboot minions",
	Long:  `⚠️  WARNING: This will reboot the target minions!`,
	RunE:  runSystemReboot,
}

var systemUptimeCmd = &cobra.Command{
	Use:   "uptime",
	Short: "Get system uptime",
	RunE:  runSystemUptime,
}

var systemDiskCmd = &cobra.Command{
	Use:   "disk",
	Short: "Get disk usage",
	RunE:  runSystemDisk,
}

var systemMemoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Get memory usage",
	RunE:  runSystemMemory,
}

var systemCPUCmd = &cobra.Command{
	Use:   "cpu",
	Short: "Get CPU information",
	RunE:  runSystemCPU,
}

var systemNetworkCmd = &cobra.Command{
	Use:   "network",
	Short: "Get network interfaces",
	RunE:  runSystemNetwork,
}

// User Management Commands

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User management operations",
	Long:  `Manage users on minions`,
}

var userAddCmd = &cobra.Command{
	Use:   "add <username>",
	Short: "Add a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserAdd,
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <username>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserDelete,
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	RunE:  runUserList,
}

var userInfoCmd = &cobra.Command{
	Use:   "info <username>",
	Short: "Get user information",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserInfo,
}

// Docker Commands

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Docker container management",
	Long:  `Manage Docker containers on minions`,
}

var dockerPSCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running containers",
	RunE:  runDockerPS,
}

var dockerStartCmd = &cobra.Command{
	Use:   "start <container>",
	Short: "Start a container",
	Args:  cobra.ExactArgs(1),
	RunE:  runDockerStart,
}

var dockerStopCmd = &cobra.Command{
	Use:   "stop <container>",
	Short: "Stop a container",
	Args:  cobra.ExactArgs(1),
	RunE:  runDockerStop,
}

var dockerRestartCmd = &cobra.Command{
	Use:   "restart <container>",
	Short: "Restart a container",
	Args:  cobra.ExactArgs(1),
	RunE:  runDockerRestart,
}

// Job Management Commands

var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "Job management operations",
	Long:  `Manage Salt jobs (list, kill, sync)`,
}

var jobListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent jobs",
	RunE:  runJobList,
}

var jobKillCmd = &cobra.Command{
	Use:   "kill <jid>",
	Short: "Kill a running job",
	Args:  cobra.ExactArgs(1),
	RunE:  runJobKill,
}

var jobSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync all modules to minions",
	RunE:  runJobSync,
}

func init() {
	// Package commands
	saltCmd.AddCommand(pkgCmd)
	pkgCmd.AddCommand(pkgInstallCmd)
	pkgCmd.AddCommand(pkgRemoveCmd)
	pkgCmd.AddCommand(pkgUpgradeCmd)
	pkgCmd.AddCommand(pkgListCmd)

	// Service commands
	saltCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceRestartCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	serviceCmd.AddCommand(serviceEnableCmd)
	serviceCmd.AddCommand(serviceDisableCmd)
	serviceCmd.AddCommand(serviceListCmd)

	// File commands
	saltCmd.AddCommand(fileCmd)
	fileCmd.AddCommand(fileReadCmd)
	fileCmd.AddCommand(fileWriteCmd)
	fileCmd.AddCommand(fileRemoveCmd)
	fileCmd.AddCommand(fileExistsCmd)
	fileCmd.AddCommand(fileChmodCmd)

	// System commands
	saltCmd.AddCommand(systemCmd)
	systemCmd.AddCommand(systemRebootCmd)
	systemCmd.AddCommand(systemUptimeCmd)
	systemCmd.AddCommand(systemDiskCmd)
	systemCmd.AddCommand(systemMemoryCmd)
	systemCmd.AddCommand(systemCPUCmd)
	systemCmd.AddCommand(systemNetworkCmd)

	// User commands
	saltCmd.AddCommand(userCmd)
	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userDeleteCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userInfoCmd)

	// Docker commands
	saltCmd.AddCommand(dockerCmd)
	dockerCmd.AddCommand(dockerPSCmd)
	dockerCmd.AddCommand(dockerStartCmd)
	dockerCmd.AddCommand(dockerStopCmd)
	dockerCmd.AddCommand(dockerRestartCmd)

	// Job commands
	saltCmd.AddCommand(jobCmd)
	jobCmd.AddCommand(jobListCmd)
	jobCmd.AddCommand(jobKillCmd)
	jobCmd.AddCommand(jobSyncCmd)
}

// Implementation functions

func runPkgInstall(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("📦 Installing packages: %s", strings.Join(args, ", "))
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.PackageInstall(saltTarget, args...)
	if err != nil {
		color.Red("❌ Installation failed: %v", err)
		return err
	}

	printSaltResponse(resp, "Package installation")
	return nil
}

func runPkgRemove(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("🗑️  Removing packages: %s", strings.Join(args, ", "))
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.PackageRemove(saltTarget, args...)
	if err != nil {
		color.Red("❌ Removal failed: %v", err)
		return err
	}

	printSaltResponse(resp, "Package removal")
	return nil
}

func runPkgUpgrade(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	if len(args) == 0 {
		color.Cyan("⬆️  Upgrading all packages")
	} else {
		color.Cyan("⬆️  Upgrading packages: %s", strings.Join(args, ", "))
	}
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.PackageUpgrade(saltTarget, args...)
	if err != nil {
		color.Red("❌ Upgrade failed: %v", err)
		return err
	}

	printSaltResponse(resp, "Package upgrade")
	return nil
}

func runPkgList(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("📋 Listing installed packages")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.PackageList(saltTarget)
	if err != nil {
		color.Red("❌ Failed to list packages: %v", err)
		return err
	}

	printSaltResponse(resp, "Installed packages")
	return nil
}

func runServiceStart(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	service := args[0]

	fmt.Println()
	color.Cyan("▶️  Starting service: %s", service)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ServiceStart(saltTarget, service)
	if err != nil {
		color.Red("❌ Failed to start service: %v", err)
		return err
	}

	printSaltResponse(resp, "Service start")
	return nil
}

func runServiceStop(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	service := args[0]

	fmt.Println()
	color.Cyan("⏹️  Stopping service: %s", service)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ServiceStop(saltTarget, service)
	if err != nil {
		color.Red("❌ Failed to stop service: %v", err)
		return err
	}

	printSaltResponse(resp, "Service stop")
	return nil
}

func runServiceRestart(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	service := args[0]

	fmt.Println()
	color.Cyan("🔄 Restarting service: %s", service)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ServiceRestart(saltTarget, service)
	if err != nil {
		color.Red("❌ Failed to restart service: %v", err)
		return err
	}

	printSaltResponse(resp, "Service restart")
	return nil
}

func runServiceStatus(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	service := args[0]

	fmt.Println()
	color.Cyan("📊 Checking service status: %s", service)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ServiceStatus(saltTarget, service)
	if err != nil {
		color.Red("❌ Failed to get status: %v", err)
		return err
	}

	printSaltResponse(resp, "Service status")
	return nil
}

func runServiceEnable(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	service := args[0]

	fmt.Println()
	color.Cyan("✅ Enabling service: %s", service)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ServiceEnable(saltTarget, service)
	if err != nil {
		color.Red("❌ Failed to enable service: %v", err)
		return err
	}

	printSaltResponse(resp, "Service enable")
	return nil
}

func runServiceDisable(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	service := args[0]

	fmt.Println()
	color.Cyan("❌ Disabling service: %s", service)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ServiceDisable(saltTarget, service)
	if err != nil {
		color.Red("❌ Failed to disable service: %v", err)
		return err
	}

	printSaltResponse(resp, "Service disable")
	return nil
}

func runServiceList(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("📋 Listing all services")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ServiceGetAll(saltTarget)
	if err != nil {
		color.Red("❌ Failed to list services: %v", err)
		return err
	}

	printSaltResponse(resp, "Services")
	return nil
}

func runFileRead(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	path := args[0]

	fmt.Println()
	color.Cyan("📄 Reading file: %s", path)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.FileRead(saltTarget, path)
	if err != nil {
		color.Red("❌ Failed to read file: %v", err)
		return err
	}

	printSaltResponse(resp, "File content")
	return nil
}

func runFileWrite(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	path, content := args[0], args[1]

	fmt.Println()
	color.Cyan("✍️  Writing to file: %s", path)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.FileWrite(saltTarget, path, content)
	if err != nil {
		color.Red("❌ Failed to write file: %v", err)
		return err
	}

	printSaltResponse(resp, "File write")
	return nil
}

func runFileRemove(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	path := args[0]

	fmt.Println()
	color.Cyan("🗑️  Removing file: %s", path)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.FileRemove(saltTarget, path)
	if err != nil {
		color.Red("❌ Failed to remove file: %v", err)
		return err
	}

	printSaltResponse(resp, "File removal")
	return nil
}

func runFileExists(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	path := args[0]

	fmt.Println()
	color.Cyan("🔍 Checking if file exists: %s", path)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.FileExists(saltTarget, path)
	if err != nil {
		color.Red("❌ Failed to check file: %v", err)
		return err
	}

	printSaltResponse(resp, "File exists")
	return nil
}

func runFileChmod(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	path, mode := args[0], args[1]

	fmt.Println()
	color.Cyan("🔒 Changing permissions: %s -> %s", path, mode)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.FileChmod(saltTarget, path, mode)
	if err != nil {
		color.Red("❌ Failed to change permissions: %v", err)
		return err
	}

	printSaltResponse(resp, "Chmod")
	return nil
}

func runSystemReboot(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Yellow("⚠️  WARNING: This will reboot the target minions!")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	if !autoApprove {
		if !confirm("Are you sure you want to reboot?") {
			color.Yellow("Reboot cancelled")
			return nil
		}
	}

	resp, err := client.SystemReboot(saltTarget)
	if err != nil {
		color.Red("❌ Reboot failed: %v", err)
		return err
	}

	printSaltResponse(resp, "System reboot")
	return nil
}

func runSystemUptime(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("⏱️  Getting system uptime")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.SystemUptime(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get uptime: %v", err)
		return err
	}

	printSaltResponse(resp, "System uptime")
	return nil
}

func runSystemDisk(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("💾 Getting disk usage")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.DiskUsage(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get disk usage: %v", err)
		return err
	}

	printSaltResponse(resp, "Disk usage")
	return nil
}

func runSystemMemory(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("🧠 Getting memory usage")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.MemoryUsage(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get memory usage: %v", err)
		return err
	}

	printSaltResponse(resp, "Memory usage")
	return nil
}

func runSystemCPU(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("⚙️  Getting CPU information")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.CPUInfo(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get CPU info: %v", err)
		return err
	}

	printSaltResponse(resp, "CPU information")
	return nil
}

func runSystemNetwork(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("🌐 Getting network interfaces")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.NetworkInterfaces(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get network info: %v", err)
		return err
	}

	printSaltResponse(resp, "Network interfaces")
	return nil
}

func runUserAdd(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	username := args[0]

	fmt.Println()
	color.Cyan("👤 Adding user: %s", username)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.UserAdd(saltTarget, username)
	if err != nil {
		color.Red("❌ Failed to add user: %v", err)
		return err
	}

	printSaltResponse(resp, "User add")
	return nil
}

func runUserDelete(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	username := args[0]

	fmt.Println()
	color.Cyan("🗑️  Deleting user: %s", username)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.UserDelete(saltTarget, username)
	if err != nil {
		color.Red("❌ Failed to delete user: %v", err)
		return err
	}

	printSaltResponse(resp, "User delete")
	return nil
}

func runUserList(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("📋 Listing users")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.UserList(saltTarget)
	if err != nil {
		color.Red("❌ Failed to list users: %v", err)
		return err
	}

	printSaltResponse(resp, "Users")
	return nil
}

func runUserInfo(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	username := args[0]

	fmt.Println()
	color.Cyan("👤 Getting user info: %s", username)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.UserInfo(saltTarget, username)
	if err != nil {
		color.Red("❌ Failed to get user info: %v", err)
		return err
	}

	printSaltResponse(resp, "User information")
	return nil
}

func runDockerPS(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("🐳 Listing Docker containers")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.DockerPS(saltTarget)
	if err != nil {
		color.Red("❌ Failed to list containers: %v", err)
		return err
	}

	printSaltResponse(resp, "Docker containers")
	return nil
}

func runDockerStart(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	container := args[0]

	fmt.Println()
	color.Cyan("▶️  Starting container: %s", container)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.DockerStart(saltTarget, container)
	if err != nil {
		color.Red("❌ Failed to start container: %v", err)
		return err
	}

	printSaltResponse(resp, "Docker start")
	return nil
}

func runDockerStop(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	container := args[0]

	fmt.Println()
	color.Cyan("⏹️  Stopping container: %s", container)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.DockerStop(saltTarget, container)
	if err != nil {
		color.Red("❌ Failed to stop container: %v", err)
		return err
	}

	printSaltResponse(resp, "Docker stop")
	return nil
}

func runDockerRestart(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	container := args[0]

	fmt.Println()
	color.Cyan("🔄 Restarting container: %s", container)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.DockerRestart(saltTarget, container)
	if err != nil {
		color.Red("❌ Failed to restart container: %v", err)
		return err
	}

	printSaltResponse(resp, "Docker restart")
	return nil
}

func runJobList(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("📋 Listing Salt jobs")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.JobsList(saltTarget)
	if err != nil {
		color.Red("❌ Failed to list jobs: %v", err)
		return err
	}

	printSaltResponse(resp, "Salt jobs")
	return nil
}

func runJobKill(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	jid := args[0]

	fmt.Println()
	color.Cyan("🔪 Killing job: %s", jid)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.JobKill(saltTarget, jid)
	if err != nil {
		color.Red("❌ Failed to kill job: %v", err)
		return err
	}

	printSaltResponse(resp, "Job kill")
	return nil
}

func runJobSync(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("🔄 Syncing all modules to minions")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.SyncAll(saltTarget)
	if err != nil {
		color.Red("❌ Sync failed: %v", err)
		return err
	}

	printSaltResponse(resp, "Module sync")
	return nil
}

// Helper function to print Salt responses
func printSaltResponse(resp *salt.CommandResponse, title string) {
	if saltOutputJSON {
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(jsonData))
		return
	}

	if len(resp.Return) == 0 || len(resp.Return[0]) == 0 {
		color.Yellow("⚠️  No results returned")
		return
	}

	color.Green("✅ %s:", title)
	fmt.Println()
	for minion, result := range resp.Return[0] {
		color.Cyan("Minion: %s", minion)
		fmt.Println(strings.Repeat("-", 60))

		// Try to format as JSON if it's a complex object
		if resultMap, ok := result.(map[string]interface{}); ok {
			jsonData, _ := json.MarshalIndent(resultMap, "  ", "  ")
			fmt.Printf("  %s\n", string(jsonData))
		} else {
			fmt.Printf("  %v\n", result)
		}
		fmt.Println()
	}
}
