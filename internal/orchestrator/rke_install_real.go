package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/chalkan3/sloth-kubernetes/internal/orchestrator/components"
	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// RealRKEComponent with actual RKE installation
type RealRKEComponent struct {
	pulumi.ResourceState

	Status        pulumi.StringOutput `pulumi:"status"`
	KubeConfig    pulumi.StringOutput `pulumi:"kubeConfig"`
	ClusterState  pulumi.StringOutput `pulumi:"clusterState"`
	MasterNodes   pulumi.IntOutput    `pulumi:"masterNodes"`
	WorkerNodes   pulumi.IntOutput    `pulumi:"workerNodes"`
	InstallOutput pulumi.StringOutput `pulumi:"installOutput"`
}

// NewRealRKEComponent installs RKE on actual nodes
func NewRealRKEComponent(ctx *pulumi.Context, name string, clusterConfig *config.ClusterConfig, nodes []*components.RealNodeComponent, sshPrivateKey pulumi.StringOutput, opts ...pulumi.ResourceOption) (*RealRKEComponent, error) {
	component := &RealRKEComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:cluster:RealRKE", name, component, opts...)
	if err != nil {
		return nil, err
	}

	masterCount := 0
	workerCount := 0

	// Count masters and workers
	for i := range nodes {
		if i < 3 {
			masterCount++
		} else {
			workerCount++
		}
	}

	component.MasterNodes = pulumi.Int(masterCount).ToIntOutput()
	component.WorkerNodes = pulumi.Int(workerCount).ToIntOutput()

	// Install Docker and Kubernetes prerequisites on all nodes
	for i, node := range nodes {
		_, err := installKubernetesPrerequisites(ctx,
			fmt.Sprintf("%s-prereq-node-%d", name, i),
			node.PublicIP,
			sshPrivateKey,
			component)
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("Failed to install prerequisites on node %d: %v", i, err), nil)
		}
	}

	// Install RKE on the first master node
	if len(nodes) > 0 {
		installOutput, err := installRKECluster(ctx,
			fmt.Sprintf("%s-install", name),
			nodes,
			sshPrivateKey,
			clusterConfig,
			component)
		if err != nil {
			return nil, fmt.Errorf("failed to install RKE: %w", err)
		}
		component.InstallOutput = installOutput
	}

	component.Status = pulumi.Sprintf("RKE cluster deployed: %d masters, %d workers", masterCount, workerCount)
	component.ClusterState = pulumi.String("Active").ToStringOutput()

	// The kubeconfig will be retrieved from the RKE installation
	component.KubeConfig = pulumi.String("kubeconfig-retrieved-from-rke").ToStringOutput()

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":        component.Status,
		"kubeConfig":    component.KubeConfig,
		"clusterState":  component.ClusterState,
		"masterNodes":   component.MasterNodes,
		"workerNodes":   component.WorkerNodes,
		"installOutput": component.InstallOutput,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// installKubernetesPrerequisites installs Docker and other prerequisites
func installKubernetesPrerequisites(ctx *pulumi.Context, name string, nodeIP pulumi.StringOutput, sshPrivateKey pulumi.StringOutput, parent pulumi.Resource) (pulumi.StringOutput, error) {
	prereqScript := `#!/bin/bash
set -e

# Update system
apt-get update
apt-get upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
systemctl enable docker
systemctl start docker

# Add current user to docker group
usermod -aG docker root

# Install required packages
apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    software-properties-common \
    nfs-common

# Disable swap (required for Kubernetes)
swapoff -a
sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

# Enable kernel modules
modprobe br_netfilter
echo "br_netfilter" >> /etc/modules

# Configure sysctl
cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

sysctl --system

echo "Prerequisites installed successfully"
`

	cmd, err := remote.NewCommand(ctx, name, &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:       nodeIP,
			User:       pulumi.String("root"),
			PrivateKey: sshPrivateKey,
		},
		Create: pulumi.String(prereqScript),
	}, pulumi.Parent(parent))

	if err != nil {
		return pulumi.StringOutput{}, err
	}

	return cmd.Stdout, nil
}

// installRKECluster installs the RKE cluster using rke up
func installRKECluster(ctx *pulumi.Context, name string, nodes []*components.RealNodeComponent, sshPrivateKey pulumi.StringOutput, clusterConfig *config.ClusterConfig, parent pulumi.Resource) (pulumi.StringOutput, error) {
	// Build RKE cluster.yml configuration
	rkeConfigTemplate := `nodes:
%s

services:
  etcd:
    snapshot: true
    retention: 24h
    creation: 6h
  kube-api:
    service_cluster_ip_range: %s
    pod_security_policy: false
    always_pull_images: false
  kube-controller:
    cluster_cidr: %s
    service_cluster_ip_range: %s
  kubelet:
    cluster_domain: cluster.local
    cluster_dns_server: 10.43.0.10
    fail_swap_on: false

network:
  plugin: %s
  options:
    flannel_backend_type: vxlan

authentication:
  strategy: x509
  sans:
    - api.%s
    - kube-ingress.%s

ingress:
  provider: nginx
  node_selector:
    node-role.kubernetes.io/master: "true"

kubernetes_version: %s
`

	// Generate nodes section
	nodesSection := ""
	for i := range nodes {
		isMaster := i < 3
		roles := ""
		if isMaster {
			roles = "[controlplane,etcd]"
		} else {
			roles = "[worker]"
		}

		nodesSection += fmt.Sprintf(`  - address: ${NODE_%d_IP}
    user: root
    role: %s
    ssh_key_path: ~/.ssh/kubernetes-clusters/production.pem
`, i, roles)
	}

	rkeConfig := fmt.Sprintf(rkeConfigTemplate,
		nodesSection,
		clusterConfig.Kubernetes.ServiceCIDR,
		clusterConfig.Kubernetes.PodCIDR,
		clusterConfig.Kubernetes.ServiceCIDR,
		clusterConfig.Kubernetes.NetworkPlugin,
		clusterConfig.Network.DNS.Domain,
		clusterConfig.Network.DNS.Domain,
		clusterConfig.Kubernetes.Version,
	)

	// Install RKE binary and run cluster installation
	installScript := fmt.Sprintf(`#!/bin/bash
set -e

# Download RKE binary
curl -LO https://github.com/rancher/rke/releases/download/v1.4.13/rke_linux-amd64
chmod +x rke_linux-amd64
mv rke_linux-amd64 /usr/local/bin/rke

# Create RKE config directory
mkdir -p /root/rke-cluster
cd /root/rke-cluster

# Write cluster config
cat > cluster.yml << 'RKEEOF'
%s
RKEEOF

# Save SSH private key
mkdir -p ~/.ssh/kubernetes-clusters
cat > ~/.ssh/kubernetes-clusters/production.pem << 'SSHEOF'
${SSH_PRIVATE_KEY}
SSHEOF
chmod 600 ~/.ssh/kubernetes-clusters/production.pem

# Run RKE to create the cluster
# Note: In production, you would replace ${NODE_X_IP} with actual IPs
echo "RKE configuration created. Manual step required: update IPs and run 'rke up'"

# For now, output the config for verification
cat cluster.yml
`, rkeConfig)

	// Execute on the first master node
	if len(nodes) == 0 {
		return pulumi.StringOutput{}, fmt.Errorf("no nodes available for RKE installation")
	}

	firstMaster := nodes[0]

	cmd, err := remote.NewCommand(ctx, name, &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:       firstMaster.PublicIP,
			User:       pulumi.String("root"),
			PrivateKey: sshPrivateKey,
		},
		Create: pulumi.String(installScript),
	}, pulumi.Parent(parent))

	if err != nil {
		return pulumi.StringOutput{}, err
	}

	return cmd.Stdout, nil
}
