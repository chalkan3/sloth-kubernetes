package config

import (
	"fmt"
	"strings"
)

// GetRKE2Defaults returns default RKE2 configuration
func GetRKE2Defaults() *RKE2Config {
	return &RKE2Config{
		Version:             "", // Empty means latest stable
		Channel:             "stable",
		ClusterToken:        "my-super-secret-cluster-token-rke2-production-2025",
		TLSSan:              []string{},
		DisableComponents:   []string{"rke2-ingress-nginx"},
		DataDir:             "/var/lib/rancher/rke2",
		NodeTaint:           []string{},
		NodeLabel:           []string{},
		SnapshotScheduleCron: "0 */12 * * *", // Every 12 hours
		SnapshotRetention:   5,
		WriteKubeconfigMode: "0600",
		ProtectKernelDefaults: false,
		SeLinux:             false,
		SecretsEncryption:   false,
		ExtraServerArgs:     make(map[string]string),
		ExtraAgentArgs:      make(map[string]string),
	}
}

// BuildRKE2ServerConfig generates the RKE2 server config file content
func BuildRKE2ServerConfig(cfg *RKE2Config, nodeIP, nodeName string, isFirstMaster bool, firstMasterIP string, k8sConfig *KubernetesConfig) string {
	var builder strings.Builder

	// Basic configuration
	builder.WriteString(fmt.Sprintf("token: %s\n", cfg.ClusterToken))

	// If not first master, join existing cluster
	if !isFirstMaster && firstMasterIP != "" {
		builder.WriteString(fmt.Sprintf("server: https://%s:9345\n", firstMasterIP))
	}

	// TLS SANs
	if len(cfg.TLSSan) > 0 {
		builder.WriteString("tls-san:\n")
		for _, san := range cfg.TLSSan {
			builder.WriteString(fmt.Sprintf("  - %s\n", san))
		}
	}

	// Network configuration
	if k8sConfig.PodCIDR != "" {
		builder.WriteString(fmt.Sprintf("cluster-cidr: %s\n", k8sConfig.PodCIDR))
	}
	if k8sConfig.ServiceCIDR != "" {
		builder.WriteString(fmt.Sprintf("service-cidr: %s\n", k8sConfig.ServiceCIDR))
	}
	if k8sConfig.ClusterDNS != "" {
		builder.WriteString(fmt.Sprintf("cluster-dns: %s\n", k8sConfig.ClusterDNS))
	}

	// CNI
	if k8sConfig.NetworkPlugin != "" {
		builder.WriteString("cni:\n")
		builder.WriteString(fmt.Sprintf("  - %s\n", k8sConfig.NetworkPlugin))
	}

	// Disable components
	if len(cfg.DisableComponents) > 0 {
		builder.WriteString("disable:\n")
		for _, component := range cfg.DisableComponents {
			builder.WriteString(fmt.Sprintf("  - %s\n", component))
		}
	}

	// Node configuration
	builder.WriteString(fmt.Sprintf("node-name: %s\n", nodeName))
	builder.WriteString(fmt.Sprintf("node-ip: %s\n", nodeIP))

	// CRITICAL FIX: Add bind-address for API server to listen on the correct IP
	// Without this, API server only listens on 127.0.0.1 and kubectl cannot connect
	// This is essential when using WireGuard VPN IPs for node-to-node communication
	builder.WriteString(fmt.Sprintf("bind-address: %s\n", nodeIP))

	// Node taints
	if len(cfg.NodeTaint) > 0 {
		builder.WriteString("node-taint:\n")
		for _, taint := range cfg.NodeTaint {
			builder.WriteString(fmt.Sprintf("  - %s\n", taint))
		}
	}

	// Node labels
	if len(cfg.NodeLabel) > 0 {
		builder.WriteString("node-label:\n")
		for _, label := range cfg.NodeLabel {
			builder.WriteString(fmt.Sprintf("  - %s\n", label))
		}
	}

	// Data directory
	if cfg.DataDir != "" && cfg.DataDir != "/var/lib/rancher/rke2" {
		builder.WriteString(fmt.Sprintf("data-dir: %s\n", cfg.DataDir))
	}

	// Etcd snapshots
	if isFirstMaster {
		if cfg.SnapshotScheduleCron != "" {
			builder.WriteString(fmt.Sprintf("etcd-snapshot-schedule-cron: %s\n", cfg.SnapshotScheduleCron))
		}
		if cfg.SnapshotRetention > 0 {
			builder.WriteString(fmt.Sprintf("etcd-snapshot-retention: %d\n", cfg.SnapshotRetention))
		}
	}

	// Security
	if cfg.SeLinux {
		builder.WriteString("selinux: true\n")
	}
	if cfg.SecretsEncryption {
		builder.WriteString("secrets-encryption: true\n")
	}
	if cfg.ProtectKernelDefaults {
		builder.WriteString("protect-kernel-defaults: true\n")
	}
	if cfg.WriteKubeconfigMode != "" {
		builder.WriteString(fmt.Sprintf("write-kubeconfig-mode: %s\n", cfg.WriteKubeconfigMode))
	}

	// System default registry
	if cfg.SystemDefaultRegistry != "" {
		builder.WriteString(fmt.Sprintf("system-default-registry: %s\n", cfg.SystemDefaultRegistry))
	}

	// CIS profiles
	if len(cfg.Profiles) > 0 {
		builder.WriteString("profile:\n")
		for _, profile := range cfg.Profiles {
			builder.WriteString(fmt.Sprintf("  - %s\n", profile))
		}
	}

	return builder.String()
}

// BuildRKE2AgentConfig generates the RKE2 agent (worker) config file content
func BuildRKE2AgentConfig(cfg *RKE2Config, nodeIP, nodeName, serverIP string) string {
	var builder strings.Builder

	// Basic configuration
	builder.WriteString(fmt.Sprintf("token: %s\n", cfg.ClusterToken))
	builder.WriteString(fmt.Sprintf("server: https://%s:9345\n", serverIP))

	// Node configuration
	builder.WriteString(fmt.Sprintf("node-name: %s\n", nodeName))
	builder.WriteString(fmt.Sprintf("node-ip: %s\n", nodeIP))

	// Node taints
	if len(cfg.NodeTaint) > 0 {
		builder.WriteString("node-taint:\n")
		for _, taint := range cfg.NodeTaint {
			builder.WriteString(fmt.Sprintf("  - %s\n", taint))
		}
	}

	// Node labels
	if len(cfg.NodeLabel) > 0 {
		builder.WriteString("node-label:\n")
		for _, label := range cfg.NodeLabel {
			builder.WriteString(fmt.Sprintf("  - %s\n", label))
		}
	}

	// Data directory
	if cfg.DataDir != "" && cfg.DataDir != "/var/lib/rancher/rke2" {
		builder.WriteString(fmt.Sprintf("data-dir: %s\n", cfg.DataDir))
	}

	// Security
	if cfg.SeLinux {
		builder.WriteString("selinux: true\n")
	}
	if cfg.ProtectKernelDefaults {
		builder.WriteString("protect-kernel-defaults: true\n")
	}

	// System default registry
	if cfg.SystemDefaultRegistry != "" {
		builder.WriteString(fmt.Sprintf("system-default-registry: %s\n", cfg.SystemDefaultRegistry))
	}

	// CIS profiles
	if len(cfg.Profiles) > 0 {
		builder.WriteString("profile:\n")
		for _, profile := range cfg.Profiles {
			builder.WriteString(fmt.Sprintf("  - %s\n", profile))
		}
	}

	return builder.String()
}

// GetRKE2InstallCommand returns the installation command for RKE2
func GetRKE2InstallCommand(cfg *RKE2Config, isServer bool) string {
	var builder strings.Builder

	builder.WriteString("curl -sfL https://get.rke2.io | ")

	// Type (server or agent)
	if isServer {
		builder.WriteString("INSTALL_RKE2_TYPE=server ")
	} else {
		builder.WriteString("INSTALL_RKE2_TYPE=agent ")
	}

	// Version
	if cfg.Version != "" {
		builder.WriteString(fmt.Sprintf("INSTALL_RKE2_VERSION=%s ", cfg.Version))
	}

	// Channel
	if cfg.Channel != "" && cfg.Version == "" {
		builder.WriteString(fmt.Sprintf("INSTALL_RKE2_CHANNEL=%s ", cfg.Channel))
	}

	builder.WriteString("sh -")

	return builder.String()
}

// MergeRKE2Config merges user config with defaults
func MergeRKE2Config(user *RKE2Config, k8sVersion string) *RKE2Config {
	defaults := GetRKE2Defaults()

	if user == nil {
		// If user config is nil, use k8s version if provided
		if k8sVersion != "" {
			defaults.Version = k8sVersion
		}
		return defaults
	}

	// Merge fields (user values override defaults)
	if user.Version != "" {
		defaults.Version = user.Version
	} else if k8sVersion != "" {
		// Fall back to kubernetes.version if rke2.version is not set
		defaults.Version = k8sVersion
	}
	if user.Channel != "" {
		defaults.Channel = user.Channel
	}
	if user.ClusterToken != "" {
		defaults.ClusterToken = user.ClusterToken
	}
	if len(user.TLSSan) > 0 {
		defaults.TLSSan = user.TLSSan
	}
	if len(user.DisableComponents) > 0 {
		defaults.DisableComponents = user.DisableComponents
	}
	if user.DataDir != "" {
		defaults.DataDir = user.DataDir
	}
	if len(user.NodeTaint) > 0 {
		defaults.NodeTaint = user.NodeTaint
	}
	if len(user.NodeLabel) > 0 {
		defaults.NodeLabel = user.NodeLabel
	}
	if user.ContainerRuntimeEndpoint != "" {
		defaults.ContainerRuntimeEndpoint = user.ContainerRuntimeEndpoint
	}
	if user.SnapshotScheduleCron != "" {
		defaults.SnapshotScheduleCron = user.SnapshotScheduleCron
	}
	if user.SnapshotRetention > 0 {
		defaults.SnapshotRetention = user.SnapshotRetention
	}
	if user.SystemDefaultRegistry != "" {
		defaults.SystemDefaultRegistry = user.SystemDefaultRegistry
	}
	if len(user.Profiles) > 0 {
		defaults.Profiles = user.Profiles
	}
	if user.SeLinux {
		defaults.SeLinux = user.SeLinux
	}
	if user.SecretsEncryption {
		defaults.SecretsEncryption = user.SecretsEncryption
	}
	if user.WriteKubeconfigMode != "" {
		defaults.WriteKubeconfigMode = user.WriteKubeconfigMode
	}
	if user.ProtectKernelDefaults {
		defaults.ProtectKernelDefaults = user.ProtectKernelDefaults
	}
	if len(user.ExtraServerArgs) > 0 {
		defaults.ExtraServerArgs = user.ExtraServerArgs
	}
	if len(user.ExtraAgentArgs) > 0 {
		defaults.ExtraAgentArgs = user.ExtraAgentArgs
	}

	return defaults
}
