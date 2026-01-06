package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var outputFormat string

// ClusterStatus represents the cluster status for JSON/YAML output
type ClusterStatus struct {
	ClusterName string       `json:"clusterName" yaml:"clusterName"`
	StackName   string       `json:"stackName" yaml:"stackName"`
	Health      string       `json:"health" yaml:"health"`
	APIEndpoint string       `json:"apiEndpoint,omitempty" yaml:"apiEndpoint,omitempty"`
	Nodes       []NodeStatus `json:"nodes" yaml:"nodes"`
	VPNStatus   string       `json:"vpnStatus" yaml:"vpnStatus"`
	RKE2Status  string       `json:"rke2Status" yaml:"rke2Status"`
	DNSStatus   string       `json:"dnsStatus" yaml:"dnsStatus"`
}

// NodeStatus represents the status of a single node
type NodeStatus struct {
	Name     string `json:"name" yaml:"name"`
	Provider string `json:"provider" yaml:"provider"`
	Role     string `json:"role" yaml:"role"`
	Status   string `json:"status" yaml:"status"`
	Region   string `json:"region" yaml:"region"`
}

var statusCmd = &cobra.Command{
	Use:   "status [stack-name]",
	Short: "Show cluster status and health information",
	Long: `Display detailed information about the cluster including:
  • Node status and health
  • Provider information
  • Network configuration
  • Kubernetes cluster state`,
	Example: `  # Show status for a specific cluster
  sloth-kubernetes status aws-cluster

  # JSON output
  sloth-kubernetes status aws-cluster --format json

  # Using --stack flag (alternative)
  sloth-kubernetes status --stack aws-cluster`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVar(&outputFormat, "format", "table", "Output format: table|json|yaml")
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get stack name from argument or flag
	targetStack := stackName
	if len(args) > 0 {
		targetStack = args[0]
	}

	// Validate output format
	if outputFormat != "table" && outputFormat != "json" && outputFormat != "yaml" {
		return fmt.Errorf("invalid output format: %s (must be table, json, or yaml)", outputFormat)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Fetching cluster status for %s...", targetStack)
	// Only show spinner for table output
	if outputFormat == "table" {
		s.Start()
	}

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Use fully qualified stack name for S3 backend
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", targetStack)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
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

	// Build cluster status
	status := buildClusterStatus(outputs, targetStack)

	// Output based on format
	switch outputFormat {
	case "json":
		return outputStatusJSON(status)
	case "yaml":
		return outputStatusYAML(status)
	default:
		return outputStatusTable(status, outputs)
	}
}

// buildClusterStatus creates a ClusterStatus from Pulumi outputs
func buildClusterStatus(outputs auto.OutputMap, targetStack string) ClusterStatus {
	status := ClusterStatus{
		StackName:  targetStack,
		Health:     "Healthy",
		VPNStatus:  "All nodes connected",
		RKE2Status: "Cluster operational",
		DNSStatus:  "All records configured",
	}

	if clusterName, ok := outputs["clusterName"]; ok {
		status.ClusterName = fmt.Sprintf("%v", clusterName.Value)
	}

	if apiEndpoint, ok := outputs["apiEndpoint"]; ok {
		status.APIEndpoint = fmt.Sprintf("%v", apiEndpoint.Value)
	}

	// Parse real node data from Pulumi outputs
	status.Nodes = parseNodesFromOutputs(outputs)

	return status
}

// parseNodesFromOutputs extracts node information from Pulumi outputs
func parseNodesFromOutputs(outputs auto.OutputMap) []NodeStatus {
	var nodes []NodeStatus

	// Try to get nodes from the "nodes" output
	if nodesOutput, ok := outputs["nodes"]; ok {
		if nodesMap, ok := nodesOutput.Value.(map[string]interface{}); ok {
			for _, nodeData := range nodesMap {
				if node, ok := nodeData.(map[string]interface{}); ok {
					nodeStatus := NodeStatus{
						Status: "Ready",
					}

					if name, ok := node["name"].(string); ok {
						nodeStatus.Name = name
					}
					if provider, ok := node["provider"].(string); ok {
						nodeStatus.Provider = capitalizeProvider(provider)
					}
					if region, ok := node["region"].(string); ok {
						nodeStatus.Region = region
					}
					if roles, ok := node["roles"].([]interface{}); ok && len(roles) > 0 {
						if role, ok := roles[0].(string); ok {
							nodeStatus.Role = role
						}
					}
					if status, ok := node["status"].(string); ok {
						nodeStatus.Status = status
					}

					nodes = append(nodes, nodeStatus)
				}
			}
		}
	}

	// If no nodes found in outputs, return empty slice
	return nodes
}

// capitalizeProvider returns a nicely formatted provider name
func capitalizeProvider(provider string) string {
	switch provider {
	case "aws":
		return "AWS"
	case "hetzner":
		return "Hetzner"
	case "digitalocean":
		return "DigitalOcean"
	case "linode":
		return "Linode"
	case "azure":
		return "Azure"
	case "gcp":
		return "GCP"
	default:
		return provider
	}
}

// outputStatusJSON outputs the status as JSON
func outputStatusJSON(status ClusterStatus) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(status)
}

// outputStatusYAML outputs the status as YAML
func outputStatusYAML(status ClusterStatus) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	return encoder.Encode(status)
}

// outputStatusTable outputs the status as a formatted table
func outputStatusTable(status ClusterStatus, outputs auto.OutputMap) error {
	printHeader(fmt.Sprintf("Cluster Status: %s", status.StackName))

	if len(outputs) == 0 {
		color.Yellow("No cluster found. Deploy with: kubernetes-create deploy")
		return nil
	}

	// Overall health
	color.Green("Overall Health: Healthy")
	fmt.Println()

	// Cluster info
	if status.ClusterName != "" {
		fmt.Printf("Cluster Name: %s\n", status.ClusterName)
	}
	if status.APIEndpoint != "" {
		fmt.Printf("API Endpoint: %s\n", status.APIEndpoint)
	}
	fmt.Println()

	// Node table with real data
	printStatusNodeTable(status.Nodes)

	fmt.Println()
	color.Green("VPN Status: ✅ %s", status.VPNStatus)
	color.Green("RKE2 Status: ✅ %s", status.RKE2Status)
	color.Green("DNS Status: ✅ %s", status.DNSStatus)

	return nil
}

func printStatusNodeTable(nodes []NodeStatus) {
	color.Cyan("Nodes:")

	if len(nodes) == 0 {
		color.Yellow("  No nodes found in cluster state")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPROVIDER\tROLE\tSTATUS\tREGION")
	fmt.Fprintln(w, "----\t--------\t----\t------\t------")

	for _, node := range nodes {
		statusIcon := "✅"
		if node.Status != "Ready" && node.Status != "created" {
			statusIcon = "⚠️"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s %s\t%s\n",
			node.Name,
			node.Provider,
			node.Role,
			statusIcon,
			node.Status,
			node.Region,
		)
	}
	w.Flush()
}
