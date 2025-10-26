package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/pkg/salt"
)

var (
	saltAPIURL      string
	saltUsername    string
	saltPassword    string
	saltTarget      string
	saltOutputJSON  bool
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
	if saltAPIURL == "" {
		return nil, fmt.Errorf(`Salt API URL is required.

Please run one of the following:

  1. Login to Salt using your stack:
     %s

  2. Set environment variables:
     export SALT_API_URL="http://bastion-ip:8000"
     export SALT_USERNAME="saltapi"
     export SALT_PASSWORD="saltapi123"

  3. Use command-line flags:
     --url "http://bastion-ip:8000" --username saltapi --password saltapi123`,
			color.CyanString("sloth-kubernetes salt login"))
	}

	client := salt.NewClient(saltAPIURL, saltUsername, saltPassword)
	return client, nil
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
		return err
	}

	if saltOutputJSON {
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	if len(resp.Return) == 0 || len(resp.Return[0]) == 0 {
		color.Yellow("⚠️  No results returned")
		return nil
	}

	color.Green("✅ Results:")
	fmt.Println()
	for minion, result := range resp.Return[0] {
		color.Cyan("Minion: %s", minion)
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("%v\n", result)
		fmt.Println()
	}

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
		return err
	}

	if saltOutputJSON {
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	color.Green("✅ State applied successfully")
	fmt.Println()

	if len(resp.Return) > 0 {
		for minion, result := range resp.Return[0] {
			color.Cyan("Minion: %s", minion)
			fmt.Println(strings.Repeat("-", 60))
			jsonData, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonData))
			fmt.Println()
		}
	}

	return nil
}

func runSaltHighstate(cmd *cobra.Command, args []string) error {
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
		return err
	}

	if saltOutputJSON {
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	color.Green("✅ Highstate completed")
	fmt.Println()

	if len(resp.Return) > 0 {
		for minion, result := range resp.Return[0] {
			color.Cyan("Minion: %s", minion)
			fmt.Println(strings.Repeat("-", 60))
			jsonData, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonData))
			fmt.Println()
		}
	}

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
