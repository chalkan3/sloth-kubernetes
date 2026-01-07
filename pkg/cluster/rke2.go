package cluster

import (
	"fmt"
	"strings"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// RKE2Manager manages RKE2 cluster deployment
type RKE2Manager struct {
	config        *config.KubernetesConfig
	rke2Config    *config.RKE2Config
	nodes         []*providers.NodeOutput
	ctx           *pulumi.Context
	kubeconfig    pulumi.StringOutput
	sshPrivateKey string
	firstMasterIP string
}

// NewRKE2Manager creates a new RKE2 manager
func NewRKE2Manager(ctx *pulumi.Context, k8sConfig *config.KubernetesConfig) *RKE2Manager {
	// Merge user RKE2 config with defaults
	rke2Config := config.MergeRKE2Config(k8sConfig.RKE2, k8sConfig.Version)

	return &RKE2Manager{
		ctx:        ctx,
		config:     k8sConfig,
		rke2Config: rke2Config,
		nodes:      make([]*providers.NodeOutput, 0),
	}
}

// SetSSHPrivateKey sets the SSH private key for connecting to nodes
func (r *RKE2Manager) SetSSHPrivateKey(key string) {
	r.sshPrivateKey = key
}

// AddNode adds a node to the RKE2 cluster
func (r *RKE2Manager) AddNode(node *providers.NodeOutput) {
	r.nodes = append(r.nodes, node)
}

// GetNodes returns all nodes in the cluster
func (r *RKE2Manager) GetNodes() []*providers.NodeOutput {
	return r.nodes
}

// DeployCluster deploys the RKE2 cluster
func (r *RKE2Manager) DeployCluster() error {
	// Separate masters and workers
	masters := r.getMasterNodes()
	workers := r.getWorkerNodes()

	if len(masters) == 0 {
		return fmt.Errorf("no master nodes found")
	}

	// Deploy first master (initializes the cluster)
	firstMaster := masters[0]
	if err := r.deployFirstMaster(firstMaster); err != nil {
		return fmt.Errorf("failed to deploy first master: %w", err)
	}

	// Store first master IP for other nodes to join
	r.firstMasterIP = firstMaster.WireGuardIP
	if r.firstMasterIP == "" {
		// Fallback to public IP if WireGuard IP not available
		r.firstMasterIP = "FIRST_MASTER_IP" // Will be resolved via Pulumi
	}

	// Deploy additional masters
	for i := 1; i < len(masters); i++ {
		if err := r.deployAdditionalMaster(masters[i], firstMaster); err != nil {
			return fmt.Errorf("failed to deploy master %d: %w", i+1, err)
		}
	}

	// Deploy workers
	for i, worker := range workers {
		if err := r.deployWorker(worker, firstMaster); err != nil {
			return fmt.Errorf("failed to deploy worker %d: %w", i+1, err)
		}
	}

	// Store kubeconfig
	r.storeKubeconfig(firstMaster)

	return nil
}

// deployFirstMaster deploys the first master node (initializes the cluster)
func (r *RKE2Manager) deployFirstMaster(node *providers.NodeOutput) error {
	nodeIP := node.WireGuardIP
	if nodeIP == "" {
		nodeIP = "NODE_IP" // Will be resolved
	}

	// Generate RKE2 server config
	serverConfig := config.BuildRKE2ServerConfig(r.rke2Config, nodeIP, node.Name, true, "", r.config)

	// Get install command
	installCmd := config.GetRKE2InstallCommand(r.rke2Config, true)

	script := fmt.Sprintf(`#!/bin/bash
set -e

echo "=== Installing RKE2 Server (First Master) ==="

# Disable swap
swapoff -a
sed -i '/swap/d' /etc/fstab

# Load required kernel modules
cat > /etc/modules-load.d/k8s.conf << EOF
overlay
br_netfilter
EOF
modprobe overlay
modprobe br_netfilter

# Kernel parameters
cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sysctl --system

# Create RKE2 config directory
mkdir -p /etc/rancher/rke2

# Write RKE2 server config
cat > /etc/rancher/rke2/config.yaml << 'RKECONFIG'
%s
RKECONFIG

# Install RKE2
echo "Installing RKE2..."
%s

# Enable and start RKE2 server
systemctl enable rke2-server.service
systemctl start rke2-server.service

# Wait for RKE2 to be ready
echo "Waiting for RKE2 to be ready..."
sleep 30

# Configure kubectl
mkdir -p /root/.kube
ln -sf /etc/rancher/rke2/rke2.yaml /root/.kube/config
chmod 600 /root/.kube/config

# Add kubectl and crictl to PATH
cat >> /root/.bashrc << 'EOF'
export PATH=$PATH:/var/lib/rancher/rke2/bin
export KUBECONFIG=/etc/rancher/rke2/rke2.yaml
alias k=kubectl
EOF

export PATH=$PATH:/var/lib/rancher/rke2/bin
export KUBECONFIG=/etc/rancher/rke2/rke2.yaml

# Wait for node to be ready
echo "Waiting for node to be ready..."
for i in {1..60}; do
    if kubectl get nodes 2>/dev/null | grep -q "Ready"; then
        echo "Node is ready!"
        kubectl get nodes
        break
    fi
    echo "Waiting... ($i/60)"
    sleep 10
done

echo "=== RKE2 First Master Deployment Complete ==="
`, serverConfig, installCmd)

	_, err := remote.NewCommand(r.ctx, fmt.Sprintf("rke2-first-master-%s", node.Name), &remote.CommandArgs{
		Connection: r.getConnection(node),
		Create:     pulumi.String(script),
		Delete: pulumi.String(`#!/bin/bash
systemctl stop rke2-server.service || true
systemctl disable rke2-server.service || true
/usr/local/bin/rke2-uninstall.sh || true
rm -rf /etc/rancher/rke2
rm -rf /var/lib/rancher/rke2
echo "RKE2 server removed"
`),
	})

	return err
}

// deployAdditionalMaster deploys additional master nodes
func (r *RKE2Manager) deployAdditionalMaster(node *providers.NodeOutput, firstMaster *providers.NodeOutput) error {
	nodeIP := node.WireGuardIP
	if nodeIP == "" {
		nodeIP = "NODE_IP"
	}

	firstMasterIP := firstMaster.WireGuardIP
	if firstMasterIP == "" {
		firstMasterIP = "FIRST_MASTER_IP"
	}

	// Generate RKE2 server config for additional master
	serverConfig := config.BuildRKE2ServerConfig(r.rke2Config, nodeIP, node.Name, false, firstMasterIP, r.config)
	installCmd := config.GetRKE2InstallCommand(r.rke2Config, true)

	script := fmt.Sprintf(`#!/bin/bash
set -e

echo "=== Installing RKE2 Server (Additional Master) ==="

# Disable swap
swapoff -a
sed -i '/swap/d' /etc/fstab

# Load required kernel modules
cat > /etc/modules-load.d/k8s.conf << EOF
overlay
br_netfilter
EOF
modprobe overlay
modprobe br_netfilter

# Kernel parameters
cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sysctl --system

# Create RKE2 config directory
mkdir -p /etc/rancher/rke2

# Write RKE2 server config
cat > /etc/rancher/rke2/config.yaml << 'RKECONFIG'
%s
RKECONFIG

# Install RKE2
echo "Installing RKE2..."
%s

# Enable and start RKE2 server
systemctl enable rke2-server.service
systemctl start rke2-server.service

# Wait for RKE2 to join the cluster
echo "Waiting to join cluster..."
sleep 60

# Configure kubectl
mkdir -p /root/.kube
ln -sf /etc/rancher/rke2/rke2.yaml /root/.kube/config
chmod 600 /root/.kube/config

export PATH=$PATH:/var/lib/rancher/rke2/bin
export KUBECONFIG=/etc/rancher/rke2/rke2.yaml

echo "=== RKE2 Additional Master Deployment Complete ==="
`, serverConfig, installCmd)

	_, err := remote.NewCommand(r.ctx, fmt.Sprintf("rke2-master-%s", node.Name), &remote.CommandArgs{
		Connection: r.getConnection(node),
		Create:     pulumi.String(script),
		Delete: pulumi.String(`#!/bin/bash
systemctl stop rke2-server.service || true
systemctl disable rke2-server.service || true
/usr/local/bin/rke2-uninstall.sh || true
rm -rf /etc/rancher/rke2
rm -rf /var/lib/rancher/rke2
echo "RKE2 server removed"
`),
	})

	return err
}

// deployWorker deploys a worker node
func (r *RKE2Manager) deployWorker(node *providers.NodeOutput, firstMaster *providers.NodeOutput) error {
	nodeIP := node.WireGuardIP
	if nodeIP == "" {
		nodeIP = "NODE_IP"
	}

	firstMasterIP := firstMaster.WireGuardIP
	if firstMasterIP == "" {
		firstMasterIP = "FIRST_MASTER_IP"
	}

	// Generate RKE2 agent config
	agentConfig := config.BuildRKE2AgentConfig(r.rke2Config, nodeIP, node.Name, firstMasterIP)
	installCmd := config.GetRKE2InstallCommand(r.rke2Config, false)

	script := fmt.Sprintf(`#!/bin/bash
set -e

echo "=== Installing RKE2 Agent (Worker) ==="

# Disable swap
swapoff -a
sed -i '/swap/d' /etc/fstab

# Load required kernel modules
cat > /etc/modules-load.d/k8s.conf << EOF
overlay
br_netfilter
EOF
modprobe overlay
modprobe br_netfilter

# Kernel parameters
cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sysctl --system

# Create RKE2 config directory
mkdir -p /etc/rancher/rke2

# Write RKE2 agent config
cat > /etc/rancher/rke2/config.yaml << 'RKECONFIG'
%s
RKECONFIG

# Install RKE2 Agent
echo "Installing RKE2 Agent..."
%s

# Enable and start RKE2 agent
systemctl enable rke2-agent.service
systemctl start rke2-agent.service

echo "Waiting for agent to join cluster..."
sleep 30

echo "=== RKE2 Worker Deployment Complete ==="
`, agentConfig, installCmd)

	_, err := remote.NewCommand(r.ctx, fmt.Sprintf("rke2-worker-%s", node.Name), &remote.CommandArgs{
		Connection: r.getConnection(node),
		Create:     pulumi.String(script),
		Delete: pulumi.String(`#!/bin/bash
systemctl stop rke2-agent.service || true
systemctl disable rke2-agent.service || true
/usr/local/bin/rke2-agent-uninstall.sh || true
rm -rf /etc/rancher/rke2
rm -rf /var/lib/rancher/rke2
echo "RKE2 agent removed"
`),
	})

	return err
}

// getConnection returns the SSH connection for a node
func (r *RKE2Manager) getConnection(node *providers.NodeOutput) *remote.ConnectionArgs {
	// Use PublicIP for SSH connection (WireGuard IP is for internal cluster communication)
	return &remote.ConnectionArgs{
		Host:       node.PublicIP,
		Port:       pulumi.Float64(22),
		User:       pulumi.String(node.SSHUser),
		PrivateKey: pulumi.String(r.sshPrivateKey),
	}
}

// getMasterNodes returns all master nodes
func (r *RKE2Manager) getMasterNodes() []*providers.NodeOutput {
	var masters []*providers.NodeOutput
	for _, node := range r.nodes {
		if r.isMasterNode(node) {
			masters = append(masters, node)
		}
	}
	return masters
}

// getWorkerNodes returns all worker nodes
func (r *RKE2Manager) getWorkerNodes() []*providers.NodeOutput {
	var workers []*providers.NodeOutput
	for _, node := range r.nodes {
		if !r.isMasterNode(node) {
			workers = append(workers, node)
		}
	}
	return workers
}

// isMasterNode checks if a node is a master
func (r *RKE2Manager) isMasterNode(node *providers.NodeOutput) bool {
	// Check labels
	if role, ok := node.Labels["role"]; ok {
		if role == "master" || role == "controlplane" || role == "control-plane" {
			return true
		}
	}

	// Check node name
	name := strings.ToLower(node.Name)
	return strings.Contains(name, "master") || strings.Contains(name, "control")
}

// storeKubeconfig retrieves and stores the kubeconfig from the first master
func (r *RKE2Manager) storeKubeconfig(masterNode *providers.NodeOutput) {
	r.kubeconfig = pulumi.All(masterNode.PublicIP).ApplyT(func(args []interface{}) string {
		// The actual kubeconfig will be retrieved via SSH in production
		// This is a placeholder that will be replaced by the actual retrieval
		return fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://%s:6443
    certificate-authority-data: BASE64_CA_DATA
  name: rke2-cluster
contexts:
- context:
    cluster: rke2-cluster
    user: rke2-admin
  name: rke2-context
current-context: rke2-context
users:
- name: rke2-admin
  user:
    client-certificate-data: BASE64_CERT_DATA
    client-key-data: BASE64_KEY_DATA
`, args[0].(string))
	}).(pulumi.StringOutput)

	secrets.Export(r.ctx, "kubeconfig", r.kubeconfig)
}

// GetKubeconfig returns the kubeconfig output
func (r *RKE2Manager) GetKubeconfig() pulumi.StringOutput {
	return r.kubeconfig
}

// InstallAddons installs additional components on the cluster
func (r *RKE2Manager) InstallAddons() error {
	masterNode := r.getMasterNodes()[0]
	if masterNode == nil {
		return fmt.Errorf("no master node found")
	}

	// Install Helm
	_, err := remote.NewCommand(r.ctx, "rke2-install-helm", &remote.CommandArgs{
		Connection: r.getConnection(masterNode),
		Create: pulumi.String(`#!/bin/bash
set -e

export PATH=$PATH:/var/lib/rancher/rke2/bin
export KUBECONFIG=/etc/rancher/rke2/rke2.yaml

# Install Helm
if ! command -v helm &> /dev/null; then
    echo "Installing Helm..."
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi

# Add common Helm repos
helm repo add stable https://charts.helm.sh/stable || true
helm repo add bitnami https://charts.bitnami.com/bitnami || true
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts || true
helm repo add grafana https://grafana.github.io/helm-charts || true
helm repo add jetstack https://charts.jetstack.io || true
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
func (r *RKE2Manager) installMonitoring(masterNode *providers.NodeOutput) error {
	_, err := remote.NewCommand(r.ctx, "rke2-install-monitoring", &remote.CommandArgs{
		Connection: r.getConnection(masterNode),
		Create: pulumi.String(`#!/bin/bash
set -e

export PATH=$PATH:/var/lib/rancher/rke2/bin
export KUBECONFIG=/etc/rancher/rke2/rke2.yaml

# Create monitoring namespace
kubectl create namespace monitoring || true

# Install Prometheus Operator
helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --set prometheus.prometheusSpec.retention=30d \
  --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.accessModes[0]=ReadWriteOnce \
  --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=50Gi \
  --set grafana.adminPassword=admin \
  --wait --timeout 10m

echo "Monitoring stack installed"
`),
	})

	return err
}

// ExportClusterInfo exports cluster information
func (r *RKE2Manager) ExportClusterInfo() {
	secrets.Export(r.ctx, "cluster_name", pulumi.String(r.ctx.Stack()))
	secrets.Export(r.ctx, "kubernetes_distribution", pulumi.String("rke2"))
	secrets.Export(r.ctx, "kubernetes_version", pulumi.String(r.rke2Config.Version))
	secrets.Export(r.ctx, "rke2_channel", pulumi.String(r.rke2Config.Channel))
	secrets.Export(r.ctx, "network_plugin", pulumi.String(r.config.NetworkPlugin))
	secrets.Export(r.ctx, "pod_cidr", pulumi.String(r.config.PodCIDR))
	secrets.Export(r.ctx, "service_cidr", pulumi.String(r.config.ServiceCIDR))
	secrets.Export(r.ctx, "cluster_dns", pulumi.String(r.config.ClusterDNS))
	secrets.Export(r.ctx, "cluster_domain", pulumi.String(r.config.ClusterDomain))

	// Export node information
	nodeInfo := make(map[string]interface{})
	for _, node := range r.nodes {
		role := "worker"
		if r.isMasterNode(node) {
			role = "master"
		}
		nodeInfo[node.Name] = map[string]interface{}{
			"wireguard_ip": node.WireGuardIP,
			"role":         role,
			"provider":     node.Provider,
			"region":       node.Region,
		}
	}
	secrets.Export(r.ctx, "nodes", pulumi.ToMap(nodeInfo))
}

// UpgradeCluster upgrades the RKE2 cluster to a new version
func (r *RKE2Manager) UpgradeCluster(newVersion string) error {
	// Update the config version
	r.rke2Config.Version = newVersion

	// Upgrade masters first (one at a time for HA)
	masters := r.getMasterNodes()
	for i, master := range masters {
		if err := r.upgradeNode(master, true); err != nil {
			return fmt.Errorf("failed to upgrade master %d: %w", i+1, err)
		}
	}

	// Then upgrade workers
	workers := r.getWorkerNodes()
	for i, worker := range workers {
		if err := r.upgradeNode(worker, false); err != nil {
			return fmt.Errorf("failed to upgrade worker %d: %w", i+1, err)
		}
	}

	return nil
}

// upgradeNode upgrades a single node
func (r *RKE2Manager) upgradeNode(node *providers.NodeOutput, isServer bool) error {
	installCmd := config.GetRKE2InstallCommand(r.rke2Config, isServer)
	serviceType := "agent"
	if isServer {
		serviceType = "server"
	}

	script := fmt.Sprintf(`#!/bin/bash
set -e

echo "=== Upgrading RKE2 on %s ==="

# Cordon the node (for workers)
export PATH=$PATH:/var/lib/rancher/rke2/bin
export KUBECONFIG=/etc/rancher/rke2/rke2.yaml

if [ "%s" = "agent" ]; then
    kubectl cordon %s || true
    kubectl drain %s --ignore-daemonsets --delete-emptydir-data --force || true
fi

# Stop the service
systemctl stop rke2-%s.service

# Upgrade RKE2
%s

# Start the service
systemctl start rke2-%s.service

# Wait for node to be ready
sleep 30

# Uncordon the node (for workers)
if [ "%s" = "agent" ]; then
    kubectl uncordon %s || true
fi

echo "=== RKE2 Upgrade Complete ==="
`, node.Name, serviceType, node.Name, node.Name, serviceType, installCmd, serviceType, serviceType, node.Name)

	_, err := remote.NewCommand(r.ctx, fmt.Sprintf("rke2-upgrade-%s", node.Name), &remote.CommandArgs{
		Connection: r.getConnection(node),
		Create:     pulumi.String(script),
	})

	return err
}

// BackupEtcd creates an etcd backup
func (r *RKE2Manager) BackupEtcd() error {
	masterNode := r.getMasterNodes()[0]
	if masterNode == nil {
		return fmt.Errorf("no master node found")
	}

	_, err := remote.NewCommand(r.ctx, "rke2-backup-etcd", &remote.CommandArgs{
		Connection: r.getConnection(masterNode),
		Create: pulumi.String(`#!/bin/bash
set -e

echo "=== Creating etcd Snapshot ==="

# RKE2 automatic snapshot
/var/lib/rancher/rke2/bin/rke2 etcd-snapshot save --name manual-backup-$(date +%Y%m%d-%H%M%S)

echo "Snapshot created successfully"
ls -la /var/lib/rancher/rke2/server/db/snapshots/
`),
	})

	return err
}

// RestoreEtcd restores etcd from a backup
func (r *RKE2Manager) RestoreEtcd(snapshotName string) error {
	masterNode := r.getMasterNodes()[0]
	if masterNode == nil {
		return fmt.Errorf("no master node found")
	}

	script := fmt.Sprintf(`#!/bin/bash
set -e

echo "=== Restoring etcd from Snapshot ==="

# Stop RKE2 on all servers first (must be done manually on other masters)
systemctl stop rke2-server.service

# Restore from snapshot
/var/lib/rancher/rke2/bin/rke2 server --cluster-reset --cluster-reset-restore-path=/var/lib/rancher/rke2/server/db/snapshots/%s

# Start RKE2
systemctl start rke2-server.service

echo "=== Restore Complete ==="
echo "NOTE: You must restart rke2-server on all other master nodes!"
`, snapshotName)

	_, err := remote.NewCommand(r.ctx, "rke2-restore-etcd", &remote.CommandArgs{
		Connection: r.getConnection(masterNode),
		Create:     pulumi.String(script),
	})

	return err
}
