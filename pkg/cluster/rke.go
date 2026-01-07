package cluster

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// RKEManager manages RKE cluster deployment
type RKEManager struct {
	config     *config.KubernetesConfig
	nodes      []*providers.NodeOutput
	ctx        *pulumi.Context
	clusterYML pulumi.StringOutput
	kubeconfig pulumi.StringOutput
}

// NewRKEManager creates a new RKE manager
func NewRKEManager(ctx *pulumi.Context, config *config.KubernetesConfig) *RKEManager {
	return &RKEManager{
		ctx:    ctx,
		config: config,
		nodes:  make([]*providers.NodeOutput, 0),
	}
}

// AddNode adds a node to the RKE cluster
func (r *RKEManager) AddNode(node *providers.NodeOutput) {
	r.nodes = append(r.nodes, node)
}

// GenerateClusterConfig generates RKE cluster.yml configuration
func (r *RKEManager) GenerateClusterConfig() pulumi.StringOutput {
	return pulumi.All(r.gatherNodeInfo()...).ApplyT(func(args []interface{}) string {
		nodes := make([]map[string]interface{}, len(args))
		for i, arg := range args {
			nodes[i] = arg.(map[string]interface{})
		}

		clusterConfig := map[string]interface{}{
			"cluster_name":       r.ctx.Stack(),
			"kubernetes_version": r.config.Version,
			"nodes":              nodes,
			"network": map[string]interface{}{
				"plugin": r.config.NetworkPlugin,
				"options": map[string]interface{}{
					"flannel_backend_type": "vxlan",
				},
			},
			"services": r.generateServicesConfig(),
			"ingress": map[string]interface{}{
				"provider": "nginx",
				"options": map[string]interface{}{
					"use-forwarded-headers": "true",
				},
			},
			"authentication": map[string]interface{}{
				"strategy": "x509",
				"options":  map[string]interface{}{},
			},
			"authorization": map[string]interface{}{
				"mode": "rbac",
			},
			"system_images":         r.generateSystemImages(),
			"ssh_key_path":          "~/.ssh/id_rsa",
			"ssh_agent_auth":        false,
			"ignore_docker_version": true,
			"private_registries":    []map[string]interface{}{},
			"addon_job_timeout":     60,
			"restore": map[string]interface{}{
				"restore": false,
			},
			"rotate_certificates": map[string]interface{}{
				"enabled": true,
			},
		}

		// Add cluster-level addons if configured
		if len(r.config.Addons) > 0 {
			clusterConfig["addons_include"] = r.config.Addons
		}

		// Convert to YAML
		yamlBytes, _ := json.MarshalIndent(clusterConfig, "", "  ")
		return string(yamlBytes)
	}).(pulumi.StringOutput)
}

// gatherNodeInfo gathers information about nodes for RKE config
func (r *RKEManager) gatherNodeInfo() []interface{} {
	nodeInfos := make([]interface{}, len(r.nodes))

	for i, node := range r.nodes {
		// Use WireGuard IP for internal communication
		nodeInfos[i] = pulumi.All(node.PublicIP, node.PrivateIP).ApplyT(func(args []interface{}) map[string]interface{} {
			publicIP := args[0].(string)
			privateIP := args[1].(string)

			// Use WireGuard IP if available
			internalIP := privateIP
			if node.WireGuardIP != "" {
				internalIP = node.WireGuardIP
			}

			nodeConfig := map[string]interface{}{
				"address":           node.WireGuardIP, // Use WireGuard IP for SSH
				"internal_address":  internalIP,
				"hostname_override": node.Name,
				"user":              node.SSHUser,
				"ssh_key_path":      node.SSHKeyPath,
				"role":              r.getNodeRoles(node),
				"labels":            node.Labels,
			}

			// Add taints if configured
			if taints := r.getNodeTaints(node); len(taints) > 0 {
				nodeConfig["taints"] = taints
			}

			// For debugging - remove in production
			nodeConfig["public_ip"] = publicIP

			return nodeConfig
		})
	}

	return nodeInfos
}

// getNodeRoles determines RKE roles for a node
func (r *RKEManager) getNodeRoles(node *providers.NodeOutput) []string {
	roles := []string{}

	// Check node labels for roles
	if role, ok := node.Labels["role"]; ok {
		switch role {
		case "master", "controlplane":
			roles = append(roles, "controlplane", "etcd")
		case "worker":
			roles = append(roles, "worker")
		case "etcd":
			roles = append(roles, "etcd")
		}
	}

	// Default roles based on node name
	if len(roles) == 0 {
		if strings.Contains(node.Name, "master") || strings.Contains(node.Name, "control") {
			roles = append(roles, "controlplane", "etcd")
		} else if strings.Contains(node.Name, "worker") {
			roles = append(roles, "worker")
		} else {
			// Default to worker if no specific role
			roles = append(roles, "worker")
		}
	}

	return roles
}

// getNodeTaints gets taints for a node
func (r *RKEManager) getNodeTaints(node *providers.NodeOutput) []map[string]interface{} {
	taints := []map[string]interface{}{}

	// Add taints from node labels
	if taintStr, ok := node.Labels["taints"]; ok {
		// Parse taint string format: key=value:effect
		parts := strings.Split(taintStr, ":")
		if len(parts) == 2 {
			keyValue := strings.Split(parts[0], "=")
			if len(keyValue) == 2 {
				taints = append(taints, map[string]interface{}{
					"key":    keyValue[0],
					"value":  keyValue[1],
					"effect": parts[1],
				})
			}
		}
	}

	return taints
}

// generateServicesConfig generates RKE services configuration
func (r *RKEManager) generateServicesConfig() map[string]interface{} {
	return map[string]interface{}{
		"etcd": map[string]interface{}{
			"creation":  "12h",
			"retention": "72h",
			"snapshot":  true,
			"backup_config": map[string]interface{}{
				"enabled":        true,
				"interval_hours": 12,
				"retention":      6,
			},
			"extra_args": map[string]interface{}{
				"heartbeat-interval":        "500",
				"election-timeout":          "5000",
				"snapshot-count":            "10000",
				"quota-backend-bytes":       "8589934592",
				"max-request-bytes":         "10485760",
				"auto-compaction-mode":      "periodic",
				"auto-compaction-retention": "1",
			},
		},
		"kube-api": map[string]interface{}{
			"service_cluster_ip_range": r.config.ServiceCIDR,
			"pod_security_policy":      false,
			"always_pull_images":       false,
			"event_rate_limit": map[string]interface{}{
				"enabled": true,
			},
			"audit_log": map[string]interface{}{
				"enabled": r.config.AuditLog,
				"configuration": map[string]interface{}{
					"max_age":    30,
					"max_backup": 10,
					"max_size":   100,
					"path":       "/var/log/kube-audit/audit-log.json",
					"format":     "json",
				},
			},
			"secrets_encryption_config": map[string]interface{}{
				"enabled": r.config.EncryptSecrets,
			},
			"extra_args": map[string]interface{}{
				"enable-admission-plugins":       "NodeRestriction,ResourceQuota,ServiceAccount",
				"max-requests-inflight":          "1000",
				"max-mutating-requests-inflight": "500",
				"default-watch-cache-size":       "1000",
			},
		},
		"kube-controller": map[string]interface{}{
			"cluster_cidr":             r.config.PodCIDR,
			"service_cluster_ip_range": r.config.ServiceCIDR,
			"extra_args": map[string]interface{}{
				"node-monitor-grace-period":       "40s",
				"node-monitor-period":             "5s",
				"pod-eviction-timeout":            "5m0s",
				"terminated-pod-gc-threshold":     "12500",
				"use-service-account-credentials": "true",
				"feature-gates":                   "RotateKubeletServerCertificate=true",
			},
		},
		"scheduler": map[string]interface{}{
			"extra_args": map[string]interface{}{
				"leader-elect": "true",
			},
		},
		"kubelet": map[string]interface{}{
			"cluster_domain":               r.config.ClusterDomain,
			"cluster_dns_server":           r.config.ClusterDNS,
			"fail_swap_on":                 false,
			"generate_serving_certificate": true,
			"extra_args": map[string]interface{}{
				"max-pods":                          "110",
				"serialize-image-pulls":             "false",
				"registry-pull-qps":                 "10",
				"registry-burst":                    "20",
				"event-qps":                         "5",
				"event-burst":                       "10",
				"cgroups-per-qos":                   "true",
				"cgroup-driver":                     "systemd",
				"feature-gates":                     "RotateKubeletServerCertificate=true",
				"protect-kernel-defaults":           "true",
				"streaming-connection-idle-timeout": "30m",
				"make-iptables-util-chains":         "true",
			},
			"extra_binds": []string{
				"/var/lib/docker:/var/lib/docker:rshared",
			},
		},
		"kubeproxy": map[string]interface{}{
			"extra_args": map[string]interface{}{
				"proxy-mode":           "ipvs",
				"ipvs-strict-arp":      "true",
				"ipvs-scheduler":       "lc",
				"ipvs-sync-period":     "30s",
				"ipvs-min-sync-period": "2s",
			},
		},
	}
}

// generateSystemImages generates system images configuration
func (r *RKEManager) generateSystemImages() map[string]interface{} {
	// Use default RKE images for the specified Kubernetes version
	// These can be overridden in config
	return map[string]interface{}{
		"kubernetes":                  fmt.Sprintf("rancher/hyperkube:%s", r.config.Version),
		"etcd":                        "rancher/mirrored-coreos-etcd:v3.5.9",
		"alpine":                      "rancher/rke-tools:v0.1.96",
		"nginx_proxy":                 "rancher/rke-tools:v0.1.96",
		"cert_downloader":             "rancher/rke-tools:v0.1.96",
		"kubernetes_services_sidecar": "rancher/rke-tools:v0.1.96",
		"kubedns":                     "rancher/mirrored-k8s-dns-kube-dns:1.22.20",
		"dnsmasq":                     "rancher/mirrored-k8s-dns-dnsmasq-nanny:1.22.20",
		"kubedns_sidecar":             "rancher/mirrored-k8s-dns-sidecar:1.22.20",
		"kubedns_autoscaler":          "rancher/cluster-proportional-autoscaler:1.8.5",
		"coredns":                     "rancher/mirrored-coredns-coredns:1.10.1",
		"coredns_autoscaler":          "rancher/cluster-proportional-autoscaler:1.8.5",
		"nodelocal":                   "rancher/mirrored-k8s-dns-node-cache:1.22.20",
		"kubernetes_external_dns":     "rancher/external-dns:v0.7.3",
		"flannel":                     "rancher/mirrored-flannel-flannel:v0.21.5",
		"flannel_cni":                 "rancher/flannel-cni:v0.3.0",
		"calico_node":                 "rancher/mirrored-calico-node:v3.22.5",
		"calico_cni":                  "rancher/calico-cni:v3.22.5",
		"calico_controllers":          "rancher/mirrored-calico-kube-controllers:v3.22.5",
		"calico_ctl":                  "rancher/mirrored-calico-ctl:v3.22.5",
		"calico_flexvol":              "rancher/mirrored-calico-pod2daemon:v3.22.5",
		"canal_node":                  "rancher/mirrored-calico-node:v3.22.5",
		"canal_cni":                   "rancher/calico-cni:v3.22.5",
		"canal_controllers":           "rancher/mirrored-calico-kube-controllers:v3.22.5",
		"canal_flannel":               "rancher/mirrored-flannel-flannel:v0.21.5",
		"canal_flexvol":               "rancher/mirrored-calico-pod2daemon:v3.22.5",
		"weave_node":                  "weaveworks/weave-kube:2.8.1",
		"weave_cni":                   "weaveworks/weave-npc:2.8.1",
		"pod_infra_container":         "rancher/mirrored-pause:3.7",
		"ingress":                     "rancher/nginx-ingress-controller:nginx-1.7.1-rancher1",
		"ingress_backend":             "rancher/mirrored-nginx-ingress-controller-defaultbackend:1.5-rancher1",
		"ingress_webhook":             "rancher/mirrored-ingress-nginx-kube-webhook-certgen:v1.1.1",
		"metrics_server":              "rancher/mirrored-metrics-server:v0.6.3",
		"windows_pod_infra_container": "rancher/mirrored-pause:3.7",
		"aci_cni_deploy_container":    "noiro/cnideploy:5.2.7.1.81c2369",
		"aci_host_container":          "noiro/aci-containers-host:5.2.7.1.81c2369",
		"aci_opflex_container":        "noiro/opflex:5.2.7.1.81c2369",
		"aci_mcast_container":         "noiro/opflex:5.2.7.1.81c2369",
		"aci_ovs_container":           "noiro/openvswitch:5.2.7.1.81c2369",
		"aci_controller_container":    "noiro/aci-containers-controller:5.2.7.1.81c2369",
		"aci_gbp_server_container":    "noiro/gbp-server:5.2.7.1.81c2369",
		"aci_opflex_server_container": "noiro/opflex-server:5.2.7.1.81c2369",
	}
}

// DeployCluster deploys the RKE cluster
func (r *RKEManager) DeployCluster() error {
	// First, ensure all nodes are ready
	if err := r.waitForNodes(); err != nil {
		return fmt.Errorf("nodes not ready: %w", err)
	}

	// Generate cluster configuration
	clusterConfig := r.GenerateClusterConfig()

	// Get the first master node to run RKE from
	masterNode := r.getMasterNode()
	if masterNode == nil {
		return fmt.Errorf("no master node found")
	}

	// Install RKE on master node and deploy cluster
	_, err := remote.NewCommand(r.ctx, "rke-deploy", &remote.CommandArgs{
		Connection: &remote.ConnectionArgs{
			Host:       masterNode.PublicIP,
			Port:       pulumi.Float64(22),
			User:       pulumi.String(masterNode.SSHUser),
			PrivateKey: pulumi.String(r.getSSHPrivateKey()),
		},
		Create: clusterConfig.ApplyT(func(config string) string {
			return fmt.Sprintf(`
#!/bin/bash
set -e

# Install RKE
if ! command -v rke &> /dev/null; then
    echo "Installing RKE..."
    curl -LO https://github.com/rancher/rke/releases/download/v1.4.11/rke_linux-amd64
    chmod +x rke_linux-amd64
    sudo mv rke_linux-amd64 /usr/local/bin/rke
fi

# Create cluster directory
mkdir -p ~/rke-cluster
cd ~/rke-cluster

# Write cluster configuration
cat > cluster.yml << 'EOF'
%s
EOF

# Deploy cluster
echo "Deploying RKE cluster..."
rke up --config cluster.yml

# Install kubectl if not present
if ! command -v kubectl &> /dev/null; then
    echo "Installing kubectl..."
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/
fi

# Configure kubectl
mkdir -p ~/.kube
cp kube_config_cluster.yml ~/.kube/config

# Verify cluster
kubectl get nodes
kubectl get pods --all-namespaces

echo "RKE cluster deployed successfully!"
`, config)
		}).(pulumi.StringOutput),
		Delete: pulumi.String(`
#!/bin/bash
cd ~/rke-cluster
if [ -f cluster.yml ]; then
    rke remove --config cluster.yml --force || true
fi
rm -rf ~/rke-cluster
echo "RKE cluster removed"
`),
	})

	if err != nil {
		return fmt.Errorf("failed to deploy RKE cluster: %w", err)
	}

	// Store kubeconfig
	r.storeKubeconfig(masterNode)

	return nil
}

// waitForNodes waits for all nodes to be ready
func (r *RKEManager) waitForNodes() error {
	// Implementation would check node readiness via SSH
	return nil
}

// getMasterNode returns the first master node
func (r *RKEManager) getMasterNode() *providers.NodeOutput {
	for _, node := range r.nodes {
		roles := r.getNodeRoles(node)
		for _, role := range roles {
			if role == "controlplane" {
				return node
			}
		}
	}
	return nil
}

// getSSHPrivateKey gets the SSH private key for connecting to nodes
func (r *RKEManager) getSSHPrivateKey() string {
	// This should be configured in the security config
	return "SSH_PRIVATE_KEY_CONTENT"
}

// storeKubeconfig stores the kubeconfig from the cluster
func (r *RKEManager) storeKubeconfig(masterNode *providers.NodeOutput) {
	r.kubeconfig = pulumi.All(masterNode.PublicIP).ApplyT(func(args []interface{}) string {
		// In production, this would retrieve the actual kubeconfig
		return "KUBECONFIG_CONTENT"
	}).(pulumi.StringOutput)

	secrets.Export(r.ctx, "kubeconfig", r.kubeconfig)
}

// InstallAddons installs additional components on the cluster
func (r *RKEManager) InstallAddons() error {
	masterNode := r.getMasterNode()
	if masterNode == nil {
		return fmt.Errorf("no master node found")
	}

	// Install Helm
	_, err := remote.NewCommand(r.ctx, "install-helm", &remote.CommandArgs{
		Connection: &remote.ConnectionArgs{
			Host:       masterNode.PublicIP,
			Port:       pulumi.Float64(22),
			User:       pulumi.String(masterNode.SSHUser),
			PrivateKey: pulumi.String(r.getSSHPrivateKey()),
		},
		Create: pulumi.String(`
#!/bin/bash
set -e

# Install Helm
if ! command -v helm &> /dev/null; then
    echo "Installing Helm..."
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi

# Add common Helm repos
helm repo add stable https://charts.helm.sh/stable
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

echo "Helm installed and configured"
`),
	})

	if err != nil {
		return fmt.Errorf("failed to install Helm: %w", err)
	}

	// Install monitoring if configured
	if r.config.Monitoring {
		if err := r.installMonitoring(masterNode); err != nil {
			return fmt.Errorf("failed to install monitoring: %w", err)
		}
	}

	return nil
}

// installMonitoring installs Prometheus and Grafana
func (r *RKEManager) installMonitoring(masterNode *providers.NodeOutput) error {
	_, err := remote.NewCommand(r.ctx, "install-monitoring", &remote.CommandArgs{
		Connection: &remote.ConnectionArgs{
			Host:       masterNode.PublicIP,
			Port:       pulumi.Float64(22),
			User:       pulumi.String(masterNode.SSHUser),
			PrivateKey: pulumi.String(r.getSSHPrivateKey()),
		},
		Create: pulumi.String(`
#!/bin/bash
set -e

# Create monitoring namespace
kubectl create namespace monitoring || true

# Install Prometheus Operator
helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --set prometheus.prometheusSpec.retention=30d \
  --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.accessModes[0]=ReadWriteOnce \
  --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=50Gi \
  --set grafana.adminPassword=admin

echo "Monitoring stack installed"
`),
	})

	return err
}

// ExportClusterInfo exports cluster information
func (r *RKEManager) ExportClusterInfo() {
	secrets.Export(r.ctx, "cluster_name", pulumi.String(r.ctx.Stack()))
	secrets.Export(r.ctx, "kubernetes_version", pulumi.String(r.config.Version))
	secrets.Export(r.ctx, "network_plugin", pulumi.String(r.config.NetworkPlugin))
	secrets.Export(r.ctx, "pod_cidr", pulumi.String(r.config.PodCIDR))
	secrets.Export(r.ctx, "service_cidr", pulumi.String(r.config.ServiceCIDR))
	secrets.Export(r.ctx, "cluster_dns", pulumi.String(r.config.ClusterDNS))
	secrets.Export(r.ctx, "cluster_domain", pulumi.String(r.config.ClusterDomain))

	// Export node information
	nodeInfo := make(map[string]interface{})
	for _, node := range r.nodes {
		nodeInfo[node.Name] = map[string]interface{}{
			"wireguard_ip": node.WireGuardIP,
			"roles":        r.getNodeRoles(node),
			"provider":     node.Provider,
			"region":       node.Region,
		}
	}
	secrets.Export(r.ctx, "nodes", pulumi.ToMap(nodeInfo))
}
