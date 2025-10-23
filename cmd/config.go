package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

var (
	outputPath string
	format     string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage cluster configuration",
	Long: `Manage cluster configuration files.

Generate example configuration files in Kubernetes-style YAML format.`,
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate example configuration file",
	Long: `Generate an example cluster configuration file in Kubernetes-style YAML format.

The generated file uses the Kubernetes API convention with:
  - apiVersion: kubernetes-create.io/v1
  - kind: Cluster
  - metadata: name, labels, annotations
  - spec: providers, network, kubernetes, nodePools

You can use environment variables in the YAML using ${VAR_NAME} syntax.`,
	Example: `  # Generate example config
  kubernetes-create config generate

  # Generate to specific file
  kubernetes-create config generate -o cluster.yaml

  # Generate minimal config
  kubernetes-create config generate --format minimal`,
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&outputPath, "output", "o", "cluster-config.yaml", "Output file path")
	generateCmd.Flags().StringVar(&format, "format", "full", "Config format: full|minimal")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	printHeader("📄 Generating Configuration File")

	var cfg *config.KubernetesStyleConfig

	switch format {
	case "minimal":
		cfg = generateMinimalConfig()
	case "full":
		cfg = config.GenerateK8sStyleConfig()
	default:
		return fmt.Errorf("unknown format: %s (use 'full' or 'minimal')", format)
	}

	// Save to file
	if err := config.SaveK8sStyleConfig(cfg, outputPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	printSuccess(fmt.Sprintf("Configuration saved to %s", outputPath))
	fmt.Println()

	// Print usage instructions
	printUsageInstructions(outputPath)

	return nil
}

func generateMinimalConfig() *config.KubernetesStyleConfig {
	return &config.KubernetesStyleConfig{
		APIVersion: "kubernetes-create.io/v1",
		Kind:       "Cluster",
		Metadata: config.K8sMetadata{
			Name: "my-cluster",
			Labels: map[string]string{
				"env": "production",
			},
		},
		Spec: config.ClusterSpec2{
			Providers: config.ProvidersSpec{
				DigitalOcean: &config.DigitalOceanSpec{
					Enabled: true,
					Token:   "${DIGITALOCEAN_TOKEN}",
					Region:  "nyc3",
				},
				Linode: &config.LinodeSpec{
					Enabled:      true,
					Token:        "${LINODE_TOKEN}",
					Region:       "us-east",
					RootPassword: "${LINODE_ROOT_PASSWORD}",
				},
			},
			Network: config.NetworkSpec{
				DNS: config.DNSSpec{
					Domain:   "example.com",
					Provider: "digitalocean",
				},
				WireGuard: &config.WireGuardSpec{
					Enabled:         true,
					ServerEndpoint:  "${WIREGUARD_ENDPOINT}",
					ServerPublicKey: "${WIREGUARD_PUBKEY}",
				},
			},
			Kubernetes: config.KubernetesSpec{
				Distribution:  "rke2",
				Version:       "v1.28.5+rke2r1",
				NetworkPlugin: "calico",
				PodCIDR:       "10.42.0.0/16",
				ServiceCIDR:   "10.43.0.0/16",
				ClusterDNS:    "10.43.0.10",
			},
			NodePools: []config.NodePoolSpec{
				{
					Name:     "masters",
					Provider: "digitalocean",
					Count:    3,
					Roles:    []string{"master"},
					Size:     "s-2vcpu-4gb",
					Image:    "ubuntu-22-04-x64",
					Region:   "nyc3",
				},
				{
					Name:     "workers",
					Provider: "digitalocean",
					Count:    3,
					Roles:    []string{"worker"},
					Size:     "s-2vcpu-4gb",
					Image:    "ubuntu-22-04-x64",
					Region:   "nyc3",
				},
			},
		},
	}
}

func printUsageInstructions(filePath string) {
	color.Cyan("📋 Next Steps:")
	fmt.Println()
	fmt.Println("1. Edit the configuration file:")
	fmt.Printf("   vim %s\n", filePath)
	fmt.Println()
	fmt.Println("2. Set your credentials:")
	fmt.Println("   export DIGITALOCEAN_TOKEN=\"your-token\"")
	fmt.Println("   export LINODE_TOKEN=\"your-token\"")
	fmt.Println("   export LINODE_ROOT_PASSWORD=\"secure-password\"")
	fmt.Println("   export WIREGUARD_ENDPOINT=\"vpn.example.com:51820\"")
	fmt.Println("   export WIREGUARD_PUBKEY=\"your-wireguard-public-key\"")
	fmt.Println()
	fmt.Println("3. Deploy the cluster:")
	fmt.Printf("   kubernetes-create deploy --config %s\n", filePath)
	fmt.Println()
	color.Yellow("💡 Tip: You can also pass credentials via CLI flags to override the config file")
	fmt.Println("   kubernetes-create deploy --config cluster.yaml --do-token xxx --linode-token yyy")
}
