package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// RKE2RealComponent represents a real RKE2 Kubernetes cluster
type RKE2RealComponent struct {
	pulumi.ResourceState

	Status         pulumi.StringOutput `pulumi:"status"`
	KubeConfig     pulumi.StringOutput `pulumi:"kubeConfig"`
	MasterCount    pulumi.IntOutput    `pulumi:"masterCount"`
	WorkerCount    pulumi.IntOutput    `pulumi:"workerCount"`
	ClusterToken   pulumi.StringOutput `pulumi:"clusterToken"`
	FirstMasterIP  pulumi.StringOutput `pulumi:"firstMasterIP"`
}

// NewRKE2RealComponent deploys a REAL RKE2 cluster
func NewRKE2RealComponent(ctx *pulumi.Context, name string, nodes []*RealNodeComponent, sshPrivateKey pulumi.StringOutput, opts ...pulumi.ResourceOption) (*RKE2RealComponent, error) {
	component := &RKE2RealComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:cluster:RKE2Real", name, component, opts...)
	if err != nil {
		return nil, err
	}

	// Separate nodes: first 3 are masters, last 3 are workers
	// NOTE: This works because node_deployment_real.go now creates pools in deterministic order:
	// 1. do-masters (1 node) → masters[0]
	// 2. linode-masters (2 nodes) → masters[1], masters[2]
	// 3. do-workers (2 nodes) → workers[0], workers[1]
	// 4. linode-workers (1 node) → workers[2]
	var masters []*RealNodeComponent
	var workers []*RealNodeComponent

	ctx.Log.Info("🔍 Separating nodes (first 3 = masters, last 3 = workers)...", nil)

	for i, node := range nodes {
		if i < 3 {
			masters = append(masters, node)
		} else {
			workers = append(workers, node)
		}
	}

	ctx.Log.Info(fmt.Sprintf("🚀 Installing RKE2: %d masters, %d workers", len(masters), len(workers)), nil)

	// Generate cluster token - use simple password format (RKE2 will generate proper K10 format internally)
	clusterToken := pulumi.String("my-super-secret-cluster-token-rke2-production-2025").ToStringOutput()

	// STEP 1: Install RKE2 on first master node (this becomes the cluster leader)
	firstMaster := masters[0]

	ctx.Log.Info("📦 Installing RKE2 on first master (cluster init)...", nil)

	firstMasterInstall, err := remote.NewCommand(ctx, fmt.Sprintf("%s-master-0-install", name), &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:           firstMaster.PublicIP,
			User:           pulumi.String("root"),
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
		},
		Create: pulumi.Sprintf(`#!/bin/bash
set -e

echo "🔧 Installing RKE2 server on first master..."

# Install RKE2 server
curl -sfL https://get.rke2.io | INSTALL_RKE2_TYPE=server sh -

# Create config directory
mkdir -p /etc/rancher/rke2

# Configure RKE2 server (first master)
cat > /etc/rancher/rke2/config.yaml << 'EOF'
token: %s
tls-san:
  - api.chalkan3.com.br
  - %s
cluster-cidr: 10.42.0.0/16
service-cidr: 10.43.0.0/16
cluster-dns: 10.43.0.10
cni:
  - calico
disable:
  - rke2-ingress-nginx
node-name: master-1
node-ip: %s
advertise-address: %s
EOF

# Start RKE2 server
systemctl enable rke2-server.service
systemctl start rke2-server.service

# Wait for RKE2 to be ready
echo "⏳ Waiting for RKE2 server to start..."
timeout=300
elapsed=0
while [ ! -f /var/lib/rancher/rke2/server/node-token ] && [ $elapsed -lt $timeout ]; do
  sleep 5
  elapsed=$((elapsed + 5))
  echo "Waited ${elapsed}s..."
done

if [ ! -f /var/lib/rancher/rke2/server/node-token ]; then
  echo "❌ RKE2 server failed to start after ${timeout}s"
  exit 1
fi

# Make kubectl accessible
mkdir -p /root/.kube
ln -sf /etc/rancher/rke2/rke2.yaml /root/.kube/config
ln -sf /var/lib/rancher/rke2/bin/kubectl /usr/local/bin/kubectl
export KUBECONFIG=/etc/rancher/rke2/rke2.yaml

echo "✅ RKE2 first master installed and running"
kubectl get nodes
cat /etc/rancher/rke2/rke2.yaml
`, clusterToken, firstMaster.PublicIP, firstMaster.WireGuardIP, firstMaster.WireGuardIP),
	}, pulumi.Parent(component), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "20m",
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to install RKE2 on first master: %w", err)
	}

	// Extract kubeconfig from first master - SIMPLIFIED
	kubeConfig := firstMasterInstall.Stdout.ApplyT(func(stdout string) string {
		// Find the kubeconfig YAML in the output (starts with "apiVersion:")
		startIdx := -1
		for i := 0; i < len(stdout)-11; i++ {
			if stdout[i:i+11] == "apiVersion:" {
				startIdx = i
				break
			}
		}
		if startIdx == -1 {
			return "# Kubeconfig not found in output"
		}
		return stdout[startIdx:]
	}).(pulumi.StringOutput)

	// STEP 2: Install RKE2 on additional master nodes (join cluster)
	var masterInstallDeps []pulumi.Resource
	masterInstallDeps = append(masterInstallDeps, firstMasterInstall)

	for i := 1; i < len(masters); i++ {
		master := masters[i]

		ctx.Log.Info(fmt.Sprintf("📦 Installing RKE2 on master %d (join cluster)...", i+1), nil)

		masterInstall, err := remote.NewCommand(ctx, fmt.Sprintf("%s-master-%d-install", name, i), &remote.CommandArgs{
			Connection: remote.ConnectionArgs{
				Host:           master.PublicIP,
				User:           pulumi.String("root"),
				PrivateKey:     sshPrivateKey,
				DialErrorLimit: pulumi.Int(30),
			},
			Create: pulumi.All(clusterToken, firstMaster.WireGuardIP, master.WireGuardIP).ApplyT(func(args []interface{}) string {
				token := args[0].(string)
				firstMasterWgIP := args[1].(string)
				myWgIP := args[2].(string)
				masterNum := i + 1

				return fmt.Sprintf(`#!/bin/bash
set -e

echo "🔧 Installing RKE2 server on master %d..."

# Install RKE2 server
curl -sfL https://get.rke2.io | INSTALL_RKE2_TYPE=server sh -

# Create config directory
mkdir -p /etc/rancher/rke2

# Configure RKE2 server (additional master - join cluster)
cat > /etc/rancher/rke2/config.yaml << 'EOF'
token: %s
server: https://%s:9345
tls-san:
  - api.chalkan3.com.br
cluster-cidr: 10.42.0.0/16
service-cidr: 10.43.0.0/16
cluster-dns: 10.43.0.10
cni:
  - calico
disable:
  - rke2-ingress-nginx
node-name: master-%d
node-ip: %s
EOF

# Start RKE2 server
systemctl enable rke2-server.service
systemctl start rke2-server.service

# Wait for RKE2 to be ready
echo "⏳ Waiting for RKE2 server to join cluster..."
sleep 60

# Make kubectl accessible
mkdir -p /root/.kube
ln -sf /etc/rancher/rke2/rke2.yaml /root/.kube/config
ln -sf /var/lib/rancher/rke2/bin/kubectl /usr/local/bin/kubectl

echo "✅ RKE2 master %d joined cluster"
`, masterNum, token, firstMasterWgIP, masterNum, myWgIP, masterNum)
			}).(pulumi.StringOutput),
		}, pulumi.Parent(component), pulumi.DependsOn(masterInstallDeps), pulumi.Timeouts(&pulumi.CustomTimeouts{
			Create: "20m",
		}))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to install RKE2 on master %d: %v", i+1, err), nil)
		} else {
			masterInstallDeps = append(masterInstallDeps, masterInstall)
		}
	}

	// STEP 3: Install RKE2 on worker nodes
	for i, worker := range workers {
		ctx.Log.Info(fmt.Sprintf("📦 Installing RKE2 on worker %d...", i+1), nil)

		_, err := remote.NewCommand(ctx, fmt.Sprintf("%s-worker-%d-install", name, i), &remote.CommandArgs{
			Connection: remote.ConnectionArgs{
				Host:           worker.PublicIP,
				User:           pulumi.String("root"),
				PrivateKey:     sshPrivateKey,
				DialErrorLimit: pulumi.Int(30),
			},
			Create: pulumi.All(clusterToken, firstMaster.WireGuardIP, worker.WireGuardIP).ApplyT(func(args []interface{}) string {
				token := args[0].(string)
				firstMasterWgIP := args[1].(string)
				myWgIP := args[2].(string)
				workerNum := i + 1

				return fmt.Sprintf(`#!/bin/bash
set -e

echo "🔧 Installing RKE2 agent on worker %d..."

# Install RKE2 agent
curl -sfL https://get.rke2.io | INSTALL_RKE2_TYPE=agent sh -

# Create config directory
mkdir -p /etc/rancher/rke2

# Configure RKE2 agent
cat > /etc/rancher/rke2/config.yaml << 'EOF'
token: %s
server: https://%s:9345
node-name: worker-%d
node-ip: %s
EOF

# Start RKE2 agent
systemctl enable rke2-agent.service
systemctl start rke2-agent.service

echo "✅ RKE2 worker %d joined cluster"
sleep 30
`, workerNum, token, firstMasterWgIP, workerNum, myWgIP, workerNum)
			}).(pulumi.StringOutput),
		}, pulumi.Parent(component), pulumi.DependsOn(masterInstallDeps), pulumi.Timeouts(&pulumi.CustomTimeouts{
			Create: "20m",
		}))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to install RKE2 on worker %d: %v", i+1, err), nil)
		}
	}

	component.Status = pulumi.Sprintf("RKE2 cluster deployed: %d masters, %d workers", len(masters), len(workers))
	component.KubeConfig = kubeConfig
	component.MasterCount = pulumi.Int(len(masters)).ToIntOutput()
	component.WorkerCount = pulumi.Int(len(workers)).ToIntOutput()
	component.ClusterToken = clusterToken
	component.FirstMasterIP = firstMaster.PublicIP

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":        component.Status,
		"kubeConfig":    component.KubeConfig,
		"masterCount":   component.MasterCount,
		"workerCount":   component.WorkerCount,
		"clusterToken":  component.ClusterToken,
		"firstMasterIP": component.FirstMasterIP,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info(fmt.Sprintf("✅ RKE2 cluster DEPLOYED: %d masters, %d workers", len(masters), len(workers)), nil)

	return component, nil
}
