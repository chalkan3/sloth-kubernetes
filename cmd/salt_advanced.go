package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Network Commands

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Advanced network operations",
	Long:  `Network diagnostics and management (ping, traceroute, netstat, routes)`,
}

var networkPingCmd = &cobra.Command{
	Use:   "ping <host> [count]",
	Short: "Ping a host",
	Example: `  sloth-kubernetes salt network ping 8.8.8.8
  sloth-kubernetes salt network ping google.com 10`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runNetworkPing,
}

var networkTracerouteCmd = &cobra.Command{
	Use:   "traceroute <host>",
	Short: "Trace route to host",
	Args:  cobra.ExactArgs(1),
	RunE:  runNetworkTraceroute,
}

var networkNetstatCmd = &cobra.Command{
	Use:   "netstat",
	Short: "Show network statistics",
	RunE:  runNetworkNetstat,
}

var networkConnectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "Show active TCP connections",
	RunE:  runNetworkConnections,
}

var networkRoutesCmd = &cobra.Command{
	Use:   "routes",
	Short: "Show routing table",
	RunE:  runNetworkRoutes,
}

var networkARPCmd = &cobra.Command{
	Use:   "arp",
	Short: "Show ARP table",
	RunE:  runNetworkARP,
}

// Process Commands

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Process management",
	Long:  `Manage processes (list, top, kill, info)`,
}

var processListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all processes",
	RunE:  runProcessList,
}

var processTopCmd = &cobra.Command{
	Use:   "top",
	Short: "Show top processes",
	RunE:  runProcessTop,
}

var processKillCmd = &cobra.Command{
	Use:   "kill <pid> [signal]",
	Short: "Kill a process",
	Example: `  sloth-kubernetes salt process kill 1234
  sloth-kubernetes salt process kill 1234 SIGTERM`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runProcessKill,
}

var processInfoCmd = &cobra.Command{
	Use:   "info <pid>",
	Short: "Get process information",
	Args:  cobra.ExactArgs(1),
	RunE:  runProcessInfo,
}

// Cron Commands

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Cron job management",
	Long:  `Manage cron jobs (list, add, remove)`,
}

var cronListCmd = &cobra.Command{
	Use:   "list <user>",
	Short: "List cron jobs for user",
	Args:  cobra.ExactArgs(1),
	RunE:  runCronList,
}

var cronAddCmd = &cobra.Command{
	Use:   "add <user> <minute> <hour> <day> <month> <weekday> <command>",
	Short: "Add a cron job",
	Example: `  # Run every day at 2am
  sloth-kubernetes salt cron add root "0" "2" "*" "*" "*" "/usr/bin/backup.sh"`,
	Args: cobra.ExactArgs(7),
	RunE: runCronAdd,
}

var cronRemoveCmd = &cobra.Command{
	Use:   "remove <user> <command>",
	Short: "Remove a cron job",
	Args:  cobra.ExactArgs(2),
	RunE:  runCronRemove,
}

// Archive Commands

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Archive operations",
	Long:  `Create and extract archives (tar, zip)`,
}

var archiveTarCmd = &cobra.Command{
	Use:     "tar <source> <destination>",
	Short:   "Create tar.gz archive",
	Example: `  sloth-kubernetes salt archive tar /var/log /tmp/logs.tar.gz`,
	Args:    cobra.ExactArgs(2),
	RunE:    runArchiveTar,
}

var archiveUntarCmd = &cobra.Command{
	Use:   "untar <source> <destination>",
	Short: "Extract tar.gz archive",
	Args:  cobra.ExactArgs(2),
	RunE:  runArchiveUntar,
}

var archiveZipCmd = &cobra.Command{
	Use:   "zip <source> <destination>",
	Short: "Create zip archive",
	Args:  cobra.ExactArgs(2),
	RunE:  runArchiveZip,
}

var archiveUnzipCmd = &cobra.Command{
	Use:   "unzip <source> <destination>",
	Short: "Extract zip archive",
	Args:  cobra.ExactArgs(2),
	RunE:  runArchiveUnzip,
}

// Monitor Commands

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitoring and statistics",
	Long:  `System monitoring (load, iostat, network stats)`,
}

var monitorLoadCmd = &cobra.Command{
	Use:   "load",
	Short: "Show load average",
	RunE:  runMonitorLoad,
}

var monitorIOCmd = &cobra.Command{
	Use:   "iostat",
	Short: "Show disk I/O statistics",
	RunE:  runMonitorIO,
}

var monitorNetIOCmd = &cobra.Command{
	Use:   "netstats",
	Short: "Show network I/O statistics",
	RunE:  runMonitorNetIO,
}

var monitorInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show comprehensive system info",
	RunE:  runMonitorInfo,
}

// SSH Commands

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "SSH key management",
	Long:  `Manage SSH keys and authorized_keys`,
}

var sshKeyGenCmd = &cobra.Command{
	Use:   "keygen <user> <type>",
	Short: "Generate SSH key",
	Example: `  sloth-kubernetes salt ssh keygen root rsa
  sloth-kubernetes salt ssh keygen user ed25519`,
	Args: cobra.ExactArgs(2),
	RunE: runSSHKeyGen,
}

var sshAuthKeysCmd = &cobra.Command{
	Use:   "authkeys <user>",
	Short: "List authorized keys",
	Args:  cobra.ExactArgs(1),
	RunE:  runSSHAuthKeys,
}

var sshSetKeyCmd = &cobra.Command{
	Use:   "setkey <user> <key>",
	Short: "Add authorized key",
	Args:  cobra.ExactArgs(2),
	RunE:  runSSHSetKey,
}

// Git Commands

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git operations",
	Long:  `Git repository management (clone, pull)`,
}

var gitCloneCmd = &cobra.Command{
	Use:     "clone <repo> <destination>",
	Short:   "Clone a git repository",
	Example: `  sloth-kubernetes salt git clone https://github.com/user/repo.git /opt/repo`,
	Args:    cobra.ExactArgs(2),
	RunE:    runGitClone,
}

var gitPullCmd = &cobra.Command{
	Use:   "pull <path>",
	Short: "Pull latest changes",
	Args:  cobra.ExactArgs(1),
	RunE:  runGitPull,
}

// Kubernetes Commands

var k8sCmd = &cobra.Command{
	Use:     "k8s",
	Aliases: []string{"kubectl"},
	Short:   "Kubernetes operations",
	Long:    `Execute kubectl commands on cluster nodes`,
}

var k8sGetCmd = &cobra.Command{
	Use:   "get <resource>",
	Short: "Get Kubernetes resources",
	Example: `  sloth-kubernetes salt k8s get nodes
  sloth-kubernetes salt k8s get pods -n kube-system
  sloth-kubernetes salt k8s get deployments --target "master*"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runK8sGet,
}

var k8sApplyCmd = &cobra.Command{
	Use:   "apply <manifest>",
	Short: "Apply Kubernetes manifest",
	Args:  cobra.ExactArgs(1),
	RunE:  runK8sApply,
}

var k8sDeleteCmd = &cobra.Command{
	Use:   "delete <resource> <name>",
	Short: "Delete Kubernetes resource",
	Args:  cobra.ExactArgs(2),
	RunE:  runK8sDelete,
}

// Pillar Commands

var pillarCmd = &cobra.Command{
	Use:   "pillar",
	Short: "Salt pillar management",
	Long:  `Manage Salt pillar data`,
}

var pillarGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get pillar value",
	Args:  cobra.ExactArgs(1),
	RunE:  runPillarGet,
}

var pillarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all pillar items",
	RunE:  runPillarList,
}

// Mount Commands

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Filesystem mount management",
	Long:  `Manage filesystem mounts`,
}

var mountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List mounted filesystems",
	RunE:  runMountList,
}

var mountMountCmd = &cobra.Command{
	Use:     "mount <device> <mountpoint> <fstype>",
	Short:   "Mount a filesystem",
	Example: `  sloth-kubernetes salt mount mount /dev/sdb1 /mnt/data ext4`,
	Args:    cobra.ExactArgs(3),
	RunE:    runMountMount,
}

var mountUnmountCmd = &cobra.Command{
	Use:   "umount <mountpoint>",
	Short: "Unmount a filesystem",
	Args:  cobra.ExactArgs(1),
	RunE:  runMountUnmount,
}

func init() {
	// Network commands
	saltCmd.AddCommand(networkCmd)
	networkCmd.AddCommand(networkPingCmd)
	networkCmd.AddCommand(networkTracerouteCmd)
	networkCmd.AddCommand(networkNetstatCmd)
	networkCmd.AddCommand(networkConnectionsCmd)
	networkCmd.AddCommand(networkRoutesCmd)
	networkCmd.AddCommand(networkARPCmd)

	// Process commands
	saltCmd.AddCommand(processCmd)
	processCmd.AddCommand(processListCmd)
	processCmd.AddCommand(processTopCmd)
	processCmd.AddCommand(processKillCmd)
	processCmd.AddCommand(processInfoCmd)

	// Cron commands
	saltCmd.AddCommand(cronCmd)
	cronCmd.AddCommand(cronListCmd)
	cronCmd.AddCommand(cronAddCmd)
	cronCmd.AddCommand(cronRemoveCmd)

	// Archive commands
	saltCmd.AddCommand(archiveCmd)
	archiveCmd.AddCommand(archiveTarCmd)
	archiveCmd.AddCommand(archiveUntarCmd)
	archiveCmd.AddCommand(archiveZipCmd)
	archiveCmd.AddCommand(archiveUnzipCmd)

	// Monitor commands
	saltCmd.AddCommand(monitorCmd)
	monitorCmd.AddCommand(monitorLoadCmd)
	monitorCmd.AddCommand(monitorIOCmd)
	monitorCmd.AddCommand(monitorNetIOCmd)
	monitorCmd.AddCommand(monitorInfoCmd)

	// SSH commands
	saltCmd.AddCommand(sshCmd)
	sshCmd.AddCommand(sshKeyGenCmd)
	sshCmd.AddCommand(sshAuthKeysCmd)
	sshCmd.AddCommand(sshSetKeyCmd)

	// Git commands
	saltCmd.AddCommand(gitCmd)
	gitCmd.AddCommand(gitCloneCmd)
	gitCmd.AddCommand(gitPullCmd)

	// Kubernetes commands
	saltCmd.AddCommand(k8sCmd)
	k8sCmd.AddCommand(k8sGetCmd)
	k8sCmd.AddCommand(k8sApplyCmd)
	k8sCmd.AddCommand(k8sDeleteCmd)

	// Pillar commands
	saltCmd.AddCommand(pillarCmd)
	pillarCmd.AddCommand(pillarGetCmd)
	pillarCmd.AddCommand(pillarListCmd)

	// Mount commands
	saltCmd.AddCommand(mountCmd)
	mountCmd.AddCommand(mountListCmd)
	mountCmd.AddCommand(mountMountCmd)
	mountCmd.AddCommand(mountUnmountCmd)
}

// Implementation functions

func runNetworkPing(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	host := args[0]
	count := 4
	if len(args) > 1 {
		count, _ = strconv.Atoi(args[1])
	}

	fmt.Println()
	color.Cyan("🏓 Pinging %s (%d times)", host, count)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.NetworkPing(saltTarget, host, count)
	if err != nil {
		color.Red("❌ Ping failed: %v", err)
		return err
	}

	printSaltResponse(resp, "Ping results")
	return nil
}

func runNetworkTraceroute(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	host := args[0]

	fmt.Println()
	color.Cyan("🗺️  Tracing route to %s", host)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.NetworkTraceroute(saltTarget, host)
	if err != nil {
		color.Red("❌ Traceroute failed: %v", err)
		return err
	}

	printSaltResponse(resp, "Traceroute")
	return nil
}

func runNetworkNetstat(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("📊 Getting network statistics")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.NetworkNetstat(saltTarget)
	if err != nil {
		color.Red("❌ Netstat failed: %v", err)
		return err
	}

	printSaltResponse(resp, "Network statistics")
	return nil
}

func runNetworkConnections(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("🔌 Getting active TCP connections")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.NetworkActiveConnections(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get connections: %v", err)
		return err
	}

	printSaltResponse(resp, "Active connections")
	return nil
}

func runNetworkRoutes(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("🛣️  Getting routing table")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.NetworkRoutes(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get routes: %v", err)
		return err
	}

	printSaltResponse(resp, "Routing table")
	return nil
}

func runNetworkARP(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("📋 Getting ARP table")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.NetworkARP(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get ARP table: %v", err)
		return err
	}

	printSaltResponse(resp, "ARP table")
	return nil
}

func runProcessList(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("📋 Listing processes")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ProcessList(saltTarget)
	if err != nil {
		color.Red("❌ Failed to list processes: %v", err)
		return err
	}

	printSaltResponse(resp, "Processes")
	return nil
}

func runProcessTop(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("⚡ Getting top processes")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ProcessTop(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get top: %v", err)
		return err
	}

	printSaltResponse(resp, "Top processes")
	return nil
}

func runProcessKill(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	pid := args[0]
	signal := "SIGTERM"
	if len(args) > 1 {
		signal = args[1]
	}

	fmt.Println()
	color.Cyan("🔪 Killing process %s with signal %s", pid, signal)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ProcessKill(saltTarget, pid, signal)
	if err != nil {
		color.Red("❌ Failed to kill process: %v", err)
		return err
	}

	printSaltResponse(resp, "Process kill")
	return nil
}

func runProcessInfo(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	pid := args[0]

	fmt.Println()
	color.Cyan("ℹ️  Getting process info for PID %s", pid)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ProcessInfo(saltTarget, pid)
	if err != nil {
		color.Red("❌ Failed to get process info: %v", err)
		return err
	}

	printSaltResponse(resp, "Process information")
	return nil
}

func runCronList(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	user := args[0]

	fmt.Println()
	color.Cyan("📋 Listing cron jobs for user: %s", user)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.CronList(saltTarget, user)
	if err != nil {
		color.Red("❌ Failed to list cron jobs: %v", err)
		return err
	}

	printSaltResponse(resp, "Cron jobs")
	return nil
}

func runCronAdd(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	user, minute, hour, daymonth, month, dayweek, command := args[0], args[1], args[2], args[3], args[4], args[5], args[6]

	fmt.Println()
	color.Cyan("➕ Adding cron job")
	color.Cyan("User: %s", user)
	color.Cyan("Schedule: %s %s %s %s %s", minute, hour, daymonth, month, dayweek)
	color.Cyan("Command: %s", command)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.CronAdd(saltTarget, user, minute, hour, daymonth, month, dayweek, command)
	if err != nil {
		color.Red("❌ Failed to add cron job: %v", err)
		return err
	}

	printSaltResponse(resp, "Cron add")
	return nil
}

func runCronRemove(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	user, command := args[0], args[1]

	fmt.Println()
	color.Cyan("🗑️  Removing cron job")
	color.Cyan("User: %s", user)
	color.Cyan("Command: %s", command)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.CronRemove(saltTarget, user, command)
	if err != nil {
		color.Red("❌ Failed to remove cron job: %v", err)
		return err
	}

	printSaltResponse(resp, "Cron remove")
	return nil
}

func runArchiveTar(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	source, dest := args[0], args[1]

	fmt.Println()
	color.Cyan("📦 Creating tar archive")
	color.Cyan("Source: %s", source)
	color.Cyan("Destination: %s", dest)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ArchiveTar(saltTarget, source, dest)
	if err != nil {
		color.Red("❌ Failed to create archive: %v", err)
		return err
	}

	printSaltResponse(resp, "Archive created")
	return nil
}

func runArchiveUntar(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	source, dest := args[0], args[1]

	fmt.Println()
	color.Cyan("📂 Extracting tar archive")
	color.Cyan("Source: %s", source)
	color.Cyan("Destination: %s", dest)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ArchiveUntar(saltTarget, source, dest)
	if err != nil {
		color.Red("❌ Failed to extract archive: %v", err)
		return err
	}

	printSaltResponse(resp, "Archive extracted")
	return nil
}

func runArchiveZip(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	source, dest := args[0], args[1]

	fmt.Println()
	color.Cyan("🤐 Creating zip archive")
	color.Cyan("Source: %s", source)
	color.Cyan("Destination: %s", dest)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ArchiveZip(saltTarget, source, dest)
	if err != nil {
		color.Red("❌ Failed to create zip: %v", err)
		return err
	}

	printSaltResponse(resp, "Zip created")
	return nil
}

func runArchiveUnzip(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	source, dest := args[0], args[1]

	fmt.Println()
	color.Cyan("🔓 Extracting zip archive")
	color.Cyan("Source: %s", source)
	color.Cyan("Destination: %s", dest)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.ArchiveUnzip(saltTarget, source, dest)
	if err != nil {
		color.Red("❌ Failed to extract zip: %v", err)
		return err
	}

	printSaltResponse(resp, "Zip extracted")
	return nil
}

func runMonitorLoad(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("📊 Getting load average")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.LoadAverage(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get load: %v", err)
		return err
	}

	printSaltResponse(resp, "Load average")
	return nil
}

func runMonitorIO(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("💾 Getting disk I/O statistics")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.DiskIOStats(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get I/O stats: %v", err)
		return err
	}

	printSaltResponse(resp, "Disk I/O statistics")
	return nil
}

func runMonitorNetIO(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("🌐 Getting network I/O statistics")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.NetworkIOStats(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get network stats: %v", err)
		return err
	}

	printSaltResponse(resp, "Network I/O statistics")
	return nil
}

func runMonitorInfo(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("📊 Getting comprehensive system information")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.SystemInfo(saltTarget)
	if err != nil {
		color.Red("❌ Failed to get system info: %v", err)
		return err
	}

	printSaltResponse(resp, "System information")
	return nil
}

func runSSHKeyGen(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	user, keytype := args[0], args[1]

	fmt.Println()
	color.Cyan("🔑 Generating SSH key")
	color.Cyan("User: %s", user)
	color.Cyan("Type: %s", keytype)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.SSHKeyGen(saltTarget, user, keytype)
	if err != nil {
		color.Red("❌ Failed to generate key: %v", err)
		return err
	}

	printSaltResponse(resp, "SSH key generation")
	return nil
}

func runSSHAuthKeys(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	user := args[0]

	fmt.Println()
	color.Cyan("🔑 Getting authorized keys for: %s", user)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.SSHAuth(saltTarget, user)
	if err != nil {
		color.Red("❌ Failed to get authorized keys: %v", err)
		return err
	}

	printSaltResponse(resp, "Authorized keys")
	return nil
}

func runSSHSetKey(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	user, key := args[0], args[1]

	fmt.Println()
	color.Cyan("🔑 Adding authorized key for: %s", user)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.SSHSetAuth(saltTarget, user, key)
	if err != nil {
		color.Red("❌ Failed to set authorized key: %v", err)
		return err
	}

	printSaltResponse(resp, "Authorized key added")
	return nil
}

func runGitClone(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	repo, dest := args[0], args[1]

	fmt.Println()
	color.Cyan("📥 Cloning repository")
	color.Cyan("Repository: %s", repo)
	color.Cyan("Destination: %s", dest)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.GitClone(saltTarget, repo, dest)
	if err != nil {
		color.Red("❌ Failed to clone repository: %v", err)
		return err
	}

	printSaltResponse(resp, "Git clone")
	return nil
}

func runGitPull(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	repo := args[0]

	fmt.Println()
	color.Cyan("🔄 Pulling latest changes")
	color.Cyan("Repository: %s", repo)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.GitPull(saltTarget, repo)
	if err != nil {
		color.Red("❌ Failed to pull: %v", err)
		return err
	}

	printSaltResponse(resp, "Git pull")
	return nil
}

func runK8sGet(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	resource := strings.Join(args, " ")

	fmt.Println()
	color.Cyan("☸️  Getting Kubernetes resources: %s", resource)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.KubectlGet(saltTarget, resource)
	if err != nil {
		color.Red("❌ kubectl get failed: %v", err)
		return err
	}

	printSaltResponse(resp, "Kubernetes resources")
	return nil
}

func runK8sApply(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	manifest := args[0]

	fmt.Println()
	color.Cyan("☸️  Applying manifest: %s", manifest)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.KubectlApply(saltTarget, manifest)
	if err != nil {
		color.Red("❌ kubectl apply failed: %v", err)
		return err
	}

	printSaltResponse(resp, "Manifest applied")
	return nil
}

func runK8sDelete(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	resource, name := args[0], args[1]

	fmt.Println()
	color.Cyan("☸️  Deleting %s/%s", resource, name)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.KubectlDelete(saltTarget, resource, name)
	if err != nil {
		color.Red("❌ kubectl delete failed: %v", err)
		return err
	}

	printSaltResponse(resp, "Resource deleted")
	return nil
}

func runPillarGet(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	key := args[0]

	fmt.Println()
	color.Cyan("🗂️  Getting pillar: %s", key)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.PillarGet(saltTarget, key)
	if err != nil {
		color.Red("❌ Failed to get pillar: %v", err)
		return err
	}

	printSaltResponse(resp, "Pillar data")
	return nil
}

func runPillarList(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("🗂️  Listing all pillar items")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.PillarItems(saltTarget)
	if err != nil {
		color.Red("❌ Failed to list pillars: %v", err)
		return err
	}

	printSaltResponse(resp, "Pillar items")
	return nil
}

func runMountList(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("💿 Listing mounted filesystems")
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.MountList(saltTarget)
	if err != nil {
		color.Red("❌ Failed to list mounts: %v", err)
		return err
	}

	printSaltResponse(resp, "Mounted filesystems")
	return nil
}

func runMountMount(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	device, mountpoint, fstype := args[0], args[1], args[2]

	fmt.Println()
	color.Cyan("💿 Mounting filesystem")
	color.Cyan("Device: %s", device)
	color.Cyan("Mountpoint: %s", mountpoint)
	color.Cyan("Type: %s", fstype)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.MountFS(saltTarget, device, mountpoint, fstype)
	if err != nil {
		color.Red("❌ Failed to mount: %v", err)
		return err
	}

	printSaltResponse(resp, "Mount")
	return nil
}

func runMountUnmount(cmd *cobra.Command, args []string) error {
	client, err := getSaltClient()
	if err != nil {
		return err
	}

	mountpoint := args[0]

	fmt.Println()
	color.Cyan("💿 Unmounting: %s", mountpoint)
	color.Cyan("Target: %s", saltTarget)
	fmt.Println()

	resp, err := client.UnmountFS(saltTarget, mountpoint)
	if err != nil {
		color.Red("❌ Failed to unmount: %v", err)
		return err
	}

	printSaltResponse(resp, "Unmount")
	return nil
}
