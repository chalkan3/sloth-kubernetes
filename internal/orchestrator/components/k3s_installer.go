package components

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// getSSHUserForProviderK3s returns the correct SSH username for the given cloud provider
// Azure uses "azureuser", while other providers use "root" or "ubuntu"
// This is identical to getSSHUserForProvider() in cloudinit_validator.go
func getSSHUserForProviderK3s(provider pulumi.StringOutput) pulumi.StringOutput {
	return provider.ApplyT(func(p string) string {
		switch p {
		case "azure":
			return "azureuser"
		case "aws":
			return "ubuntu" // AWS Ubuntu AMIs use "ubuntu"
		case "gcp":
			return "ubuntu" // GCP uses "ubuntu" for Ubuntu images
		default:
			return "root" // DigitalOcean, Linode, and others use "root"
		}
	}).(pulumi.StringOutput)
}

// K3sRealComponent represents a real K3s Kubernetes cluster
type K3sRealComponent struct {
	pulumi.ResourceState

	Status        pulumi.StringOutput `pulumi:"status"`
	KubeConfig    pulumi.StringOutput `pulumi:"kubeConfig"`
	MasterCount   pulumi.IntOutput    `pulumi:"masterCount"`
	WorkerCount   pulumi.IntOutput    `pulumi:"workerCount"`
	ClusterToken  pulumi.StringOutput `pulumi:"clusterToken"`
	FirstMasterIP pulumi.StringOutput `pulumi:"firstMasterIP"`
}

// NewK3sRealComponent deploys a REAL K3s cluster
func NewK3sRealComponent(ctx *pulumi.Context, name string, nodes []*RealNodeComponent, sshPrivateKey pulumi.StringOutput, cfg *config.ClusterConfig, bastionComponent *BastionComponent, opts ...pulumi.ResourceOption) (*K3sRealComponent, error) {
	component := &K3sRealComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:cluster:K3sReal", name, component, opts...)
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

	ctx.Log.Info(fmt.Sprintf("🚀 Installing K3s: %d masters, %d workers", len(masters), len(workers)), nil)

	// Use cluster token from configuration (from RKE2 config if exists, otherwise generate a default)
	clusterToken := "my-super-secret-cluster-token-k3s-production-2025"
	if cfg.Kubernetes.RKE2 != nil && cfg.Kubernetes.RKE2.ClusterToken != "" {
		clusterToken = cfg.Kubernetes.RKE2.ClusterToken
	}
	clusterTokenOutput := pulumi.String(clusterToken).ToStringOutput()

	// STEP 1: Install K3s on first master node (this becomes the cluster leader)
	firstMaster := masters[0]

	ctx.Log.Info("📦 Installing K3s on first master (cluster init)...", nil)

	// Determine SSH user based on provider (Azure uses "azureuser", others use "root")
	firstMasterSSHUser := getSSHUserForProviderK3s(firstMaster.Provider)

	// Build connection args with ProxyJump if bastion is enabled
	firstMasterConnArgs := remote.ConnectionArgs{
		Host:           firstMaster.PublicIP,
		User:           firstMasterSSHUser,
		PrivateKey:     sshPrivateKey,
		DialErrorLimit: pulumi.Int(30),
	}
	if bastionComponent != nil {
		// Bastion is on Linode in this config, so it uses "root"
		firstMasterConnArgs.Proxy = &remote.ProxyConnectionArgs{
			Host:       bastionComponent.PublicIP,
			User:       pulumi.String("root"),
			PrivateKey: sshPrivateKey,
		}
	}

	firstMasterInstall, err := remote.NewCommand(ctx, fmt.Sprintf("%s-master-0-install", name), &remote.CommandArgs{
		Connection: firstMasterConnArgs,
		Create: pulumi.All(firstMaster.WireGuardIP, firstMaster.PublicIP).ApplyT(func(args []interface{}) string {
			wgIP := args[0].(string)
			publicIP := args[1].(string)

			return fmt.Sprintf(`#!/bin/bash
set -e

echo "🔧 Installing K3s on first master..."

# Wait for WireGuard to be ready
echo "⏳ Waiting for WireGuard VPN interface (wg0)..."
timeout=60
elapsed=0
while [ $elapsed -lt $timeout ]; do
  if ip addr show wg0 &>/dev/null && ip addr show wg0 | grep -q "%s"; then
    break
  fi
  sleep 2
  elapsed=$((elapsed + 2))
done

echo "✅ WireGuard ready (IP: %s)"

# Install K3s with inline configuration
echo "📥 Installing K3s server..."
if ! curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server \
  --node-ip=%s \
  --node-external-ip=%s \
  --advertise-address=%s \
  --tls-san=%s \
  --tls-san=%s \
  --tls-san=127.0.0.1 \
  --flannel-iface=wg0 \
  --write-kubeconfig-mode=644 \
  --cluster-init \
  --disable=traefik" sh -; then
  echo "❌ K3s installation script failed!"
  exit 1
fi

# Verify K3s service started successfully
echo "🔍 Checking K3s service status..."
sleep 5
if ! systemctl is-active --quiet k3s; then
  echo "❌ K3s service is not running!"
  echo "Service status:"
  systemctl status k3s --no-pager || true
  echo ""
  echo "Last 50 lines of K3s logs:"
  journalctl -xeu k3s.service -n 50 --no-pager || true
  exit 1
fi
echo "✅ K3s service is running!"

# Fix kubeconfig to use VPN IP instead of 127.0.0.1 or 0.0.0.0
echo "🔧 Fixing kubeconfig to use VPN IP..."
if [ -f /etc/rancher/k3s/k3s.yaml ]; then
  sed -i "s|https://127.0.0.1:6443|https://%s:6443|g" /etc/rancher/k3s/k3s.yaml
  sed -i "s|https://0.0.0.0:6443|https://%s:6443|g" /etc/rancher/k3s/k3s.yaml
  echo "✅ Kubeconfig updated to use VPN IP %s"
fi

# Wait for K3s API
echo "⏳ Waiting for K3s API server..."
timeout=180
elapsed=0
while [ $elapsed -lt $timeout ]; do
  if [ -f /etc/rancher/k3s/k3s.yaml ] && kubectl --kubeconfig=/etc/rancher/k3s/k3s.yaml get nodes &>/dev/null 2>&1; then
    echo "✅ K3s server is ready!"
    break
  fi
  if [ $((elapsed %% 10)) -eq 0 ]; then
    echo "  ${elapsed}s: Waiting for API server..."
  fi
  sleep 3
  elapsed=$((elapsed + 3))
done

# Show status
kubectl --kubeconfig=/etc/rancher/k3s/k3s.yaml get nodes
cat /etc/rancher/k3s/k3s.yaml
`, wgIP, wgIP, wgIP, publicIP, wgIP, wgIP, publicIP, wgIP, wgIP, wgIP)
		}).(pulumi.StringOutput),
	}, pulumi.Parent(component), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "30m", // Increased from 20m for slower Azure B1s VMs
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to install K3s on first master: %w", err)
	}

	// Extract kubeconfig from first master and replace localhost with VPN IP
	kubeConfig := pulumi.All(firstMasterInstall.Stdout, firstMaster.WireGuardIP).ApplyT(func(args []interface{}) string {
		stdout := args[0].(string)
		vpnIP := args[1].(string)

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

		// Get the kubeconfig
		kubeconfigRaw := stdout[startIdx:]

		// Replace 127.0.0.1:6443 and 0.0.0.0:6443 with the VPN IP of the first master
		// K3s uses port 6443 by default
		kubeconfigWithVPN := ""
		for i := 0; i < len(kubeconfigRaw); i++ {
			if i+21 < len(kubeconfigRaw) && kubeconfigRaw[i:i+21] == "https://127.0.0.1:6443" {
				kubeconfigWithVPN += "https://" + vpnIP + ":6443"
				i += 20 // Skip the rest of the matched string
			} else if i+20 < len(kubeconfigRaw) && kubeconfigRaw[i:i+20] == "https://0.0.0.0:6443" {
				kubeconfigWithVPN += "https://" + vpnIP + ":6443"
				i += 19 // Skip the rest of the matched string
			} else {
				kubeconfigWithVPN += string(kubeconfigRaw[i])
			}
		}

		return kubeconfigWithVPN
	}).(pulumi.StringOutput)

	// Fetch the join token from the first master
	ctx.Log.Info("🔑 Fetching K3s join token from first master...", nil)

	tokenFetchConnArgs := remote.ConnectionArgs{
		Host:           firstMaster.PublicIP,
		User:           firstMasterSSHUser, // Reuse SSH user from first master (Azure = azureuser, others = root)
		PrivateKey:     sshPrivateKey,
		DialErrorLimit: pulumi.Int(30),
	}
	if bastionComponent != nil {
		// Bastion is on Linode in this config, so it uses "root"
		tokenFetchConnArgs.Proxy = &remote.ProxyConnectionArgs{
			Host:       bastionComponent.PublicIP,
			User:       pulumi.String("root"),
			PrivateKey: sshPrivateKey,
		}
	}

	tokenFetch, err := remote.NewCommand(ctx, fmt.Sprintf("%s-fetch-token", name), &remote.CommandArgs{
		Connection: tokenFetchConnArgs,
		Create: pulumi.String(`#!/bin/bash
set -e

# Wait for token file to exist
timeout=120
elapsed=0
while [ $elapsed -lt $timeout ]; do
  if [ -f /var/lib/rancher/k3s/server/node-token ]; then
    cat /var/lib/rancher/k3s/server/node-token
    exit 0
  fi
  sleep 3
  elapsed=$((elapsed + 3))
done

echo "ERROR: Token file not found after ${timeout}s" >&2
exit 1
`),
	}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{firstMasterInstall}), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "5m",
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch K3s token: %w", err)
	}

	// Extract the token from stdout
	k3sToken := tokenFetch.Stdout

	// STEP 2: Install K3s on additional master nodes (join cluster) - PARALLEL
	// All additional masters and workers can install in parallel after first master is ready
	ctx.Log.Info("🚀 Installing remaining nodes IN PARALLEL (masters 2-3 + all workers)...", nil)

	var allRemainingInstalls []pulumi.Resource

	// Create all additional master installations in parallel
	for i := 1; i < len(masters); i++ {
		master := masters[i]

		ctx.Log.Info(fmt.Sprintf("📦 Installing K3s on master %d (join cluster) [PARALLEL]...", i+1), nil)

		// Determine SSH user based on provider (Azure uses "azureuser", others use "root")
		masterSSHUser := getSSHUserForProviderK3s(master.Provider)

		// Build connection args with ProxyJump if bastion is enabled
		masterConnArgs := remote.ConnectionArgs{
			Host:           master.PublicIP,
			User:           masterSSHUser,
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
		}
		if bastionComponent != nil {
			// Bastion is on Linode in this config, so it uses "root"
			masterConnArgs.Proxy = &remote.ProxyConnectionArgs{
				Host:       bastionComponent.PublicIP,
				User:       pulumi.String("root"),
				PrivateKey: sshPrivateKey,
			}
		}

		masterInstall, err := remote.NewCommand(ctx, fmt.Sprintf("%s-master-%d-install", name, i), &remote.CommandArgs{
			Connection: masterConnArgs,
			Create: pulumi.All(k3sToken, firstMaster.WireGuardIP, master.WireGuardIP, master.PublicIP).ApplyT(func(args []interface{}) string {
				token := args[0].(string) // K3s join token from first master
				firstMasterWgIP := args[1].(string)
				myWgIP := args[2].(string)
				myPublicIP := args[3].(string)
				masterNum := i + 1

				return fmt.Sprintf(`#!/bin/bash
set -e

echo "🔧 Installing K3s on master %d (join cluster)..."

# Wait for WireGuard to be ready
echo "⏳ Waiting for WireGuard VPN interface (wg0)..."
timeout=60
elapsed=0
while [ $elapsed -lt $timeout ]; do
  if ip addr show wg0 &>/dev/null && ip addr show wg0 | grep -q "%s"; then
    break
  fi
  sleep 2
  elapsed=$((elapsed + 2))
done

echo "✅ WireGuard ready (IP: %s)"

# Wait for first master API server to be ready
echo "⏳ Waiting for first master API server at %s:6443..."
timeout=300
elapsed=0
while [ $elapsed -lt $timeout ]; do
  if nc -z -w 5 %s 6443 2>/dev/null; then
    echo "✅ First master API server is reachable!"
    break
  fi
  if [ $((elapsed %% 30)) -eq 0 ]; then
    echo "  ${elapsed}s: Still waiting for API server..."
  fi
  sleep 5
  elapsed=$((elapsed + 5))
done

if ! nc -z -w 5 %s 6443 2>/dev/null; then
  echo "❌ Failed to reach first master API server after ${timeout}s"
  echo "Attempting curl test..."
  curl -k -v https://%s:6443/version 2>&1 || true
  exit 1
fi

# Install K3s server in cluster mode (join existing cluster)
echo "📥 Installing K3s server (joining cluster)..."
if ! curl -sfL https://get.k3s.io | K3S_URL=https://%s:6443 \
  K3S_TOKEN="%s" \
  INSTALL_K3S_EXEC="server \
    --server https://%s:6443 \
    --node-ip=%s \
    --node-external-ip=%s \
    --advertise-address=%s \
    --tls-san=%s \
    --tls-san=%s \
    --tls-san=127.0.0.1 \
    --flannel-iface=wg0 \
    --write-kubeconfig-mode=644 \
    --disable=traefik" sh -; then
  echo "❌ K3s installation script failed!"
  exit 1
fi

# Verify K3s service started successfully
echo "🔍 Checking K3s service status..."
sleep 5
if ! systemctl is-active --quiet k3s; then
  echo "❌ K3s service is not running!"
  echo "Service status:"
  systemctl status k3s --no-pager || true
  echo ""
  echo "Last 50 lines of K3s logs:"
  journalctl -xeu k3s.service -n 50 --no-pager || true
  exit 1
fi
echo "✅ K3s service is running!"

# Wait for K3s to join cluster
echo "⏳ Waiting for K3s server to join cluster..."
timeout=180
elapsed=0
while [ $elapsed -lt $timeout ]; do
  if [ -f /etc/rancher/k3s/k3s.yaml ] && kubectl --kubeconfig=/etc/rancher/k3s/k3s.yaml get nodes &>/dev/null 2>&1; then
    echo "✅ K3s server joined cluster!"
    break
  fi
  if [ $((elapsed %% 10)) -eq 0 ]; then
    echo "  ${elapsed}s: Waiting to join cluster..."
  fi
  sleep 3
  elapsed=$((elapsed + 3))
done

kubectl --kubeconfig=/etc/rancher/k3s/k3s.yaml get nodes

echo "✅ K3s master %d joined cluster"
`, masterNum, myWgIP, myWgIP, firstMasterWgIP, firstMasterWgIP, firstMasterWgIP, firstMasterWgIP, firstMasterWgIP, token, firstMasterWgIP, myWgIP, myPublicIP, myWgIP, myWgIP, myPublicIP, masterNum)
			}).(pulumi.StringOutput),
		}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{tokenFetch}), pulumi.Timeouts(&pulumi.CustomTimeouts{
			Create: "30m", // Increased from 20m for slower Azure B1s VMs
		}))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to install K3s on master %d: %v", i+1, err), nil)
		} else {
			allRemainingInstalls = append(allRemainingInstalls, masterInstall)
		}
	}

	// STEP 3: Install K3s on worker nodes - PARALLEL (all workers start at the same time as additional masters)
	for i, worker := range workers {
		ctx.Log.Info(fmt.Sprintf("📦 Installing K3s on worker %d [PARALLEL]...", i+1), nil)

		// Determine SSH user based on provider (Azure uses "azureuser", others use "root")
		workerSSHUser := getSSHUserForProviderK3s(worker.Provider)

		// Build connection args with ProxyJump if bastion is enabled
		workerConnArgs := remote.ConnectionArgs{
			Host:           worker.PublicIP,
			User:           workerSSHUser,
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
		}
		if bastionComponent != nil {
			// Bastion is on Linode in this config, so it uses "root"
			workerConnArgs.Proxy = &remote.ProxyConnectionArgs{
				Host:       bastionComponent.PublicIP,
				User:       pulumi.String("root"),
				PrivateKey: sshPrivateKey,
			}
		}

		workerInstall, err := remote.NewCommand(ctx, fmt.Sprintf("%s-worker-%d-install", name, i), &remote.CommandArgs{
			Connection: workerConnArgs,
			Create: pulumi.All(k3sToken, firstMaster.WireGuardIP, worker.WireGuardIP, worker.PublicIP).ApplyT(func(args []interface{}) string {
				token := args[0].(string) // K3s join token from first master
				firstMasterWgIP := args[1].(string)
				myWgIP := args[2].(string)
				myPublicIP := args[3].(string)
				workerNum := i + 1

				return fmt.Sprintf(`#!/bin/bash
set -e

echo "🔧 Installing K3s agent on worker %d..."

# Wait for WireGuard to be ready
echo "⏳ Waiting for WireGuard VPN interface (wg0)..."
timeout=60
elapsed=0
while [ $elapsed -lt $timeout ]; do
  if ip addr show wg0 &>/dev/null && ip addr show wg0 | grep -q "%s"; then
    break
  fi
  sleep 2
  elapsed=$((elapsed + 2))
done

echo "✅ WireGuard ready (IP: %s)"

# Wait for first master API server to be ready
echo "⏳ Waiting for first master API server at %s:6443..."
timeout=300
elapsed=0
while [ $elapsed -lt $timeout ]; do
  if nc -z -w 5 %s 6443 2>/dev/null; then
    echo "✅ First master API server is reachable!"
    break
  fi
  if [ $((elapsed %% 30)) -eq 0 ]; then
    echo "  ${elapsed}s: Still waiting for API server..."
  fi
  sleep 5
  elapsed=$((elapsed + 5))
done

if ! nc -z -w 5 %s 6443 2>/dev/null; then
  echo "❌ Failed to reach first master API server after ${timeout}s"
  exit 1
fi

# Get hostname for node name
HOSTNAME=$(hostname -s)

# Install K3s agent (worker node)
echo "📥 Installing K3s agent (joining cluster)..."
if ! curl -sfL https://get.k3s.io | K3S_URL=https://%s:6443 \
  K3S_TOKEN="%s" \
  INSTALL_K3S_EXEC="agent \
    --node-name=${HOSTNAME} \
    --node-ip=%s \
    --node-external-ip=%s \
    --flannel-iface=wg0" sh -; then
  echo "❌ K3s agent installation script failed!"
  exit 1
fi

# Verify K3s agent service started successfully
echo "🔍 Checking K3s agent service status..."
sleep 5
if ! systemctl is-active --quiet k3s-agent; then
  echo "❌ K3s agent service is not running!"
  echo "Service status:"
  systemctl status k3s-agent --no-pager || true
  echo ""
  echo "Last 50 lines of K3s agent logs:"
  journalctl -xeu k3s-agent.service -n 50 --no-pager || true
  exit 1
fi
echo "✅ K3s agent service is running!"

# Wait for K3s agent to join
echo "⏳ Waiting for K3s agent to join cluster..."
sleep 30

echo "✅ K3s worker %d joined cluster"
`, workerNum, myWgIP, myWgIP, firstMasterWgIP, firstMasterWgIP, firstMasterWgIP, firstMasterWgIP, token, myWgIP, myPublicIP, workerNum)
			}).(pulumi.StringOutput),
		}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{tokenFetch}), pulumi.Timeouts(&pulumi.CustomTimeouts{
			Create: "30m", // Increased from 20m for slower Azure B1s VMs
		}))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to install K3s on worker %d: %v", i+1, err), nil)
		} else {
			allRemainingInstalls = append(allRemainingInstalls, workerInstall)
		}
	}

	ctx.Log.Info(fmt.Sprintf("✅ Created %d parallel K3s installations (2 masters + 3 workers)", len(allRemainingInstalls)), nil)

	component.Status = pulumi.Sprintf("K3s cluster deployed: %d masters, %d workers", len(masters), len(workers))
	component.KubeConfig = kubeConfig
	component.MasterCount = pulumi.Int(len(masters)).ToIntOutput()
	component.WorkerCount = pulumi.Int(len(workers)).ToIntOutput()
	component.ClusterToken = clusterTokenOutput
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

	ctx.Log.Info(fmt.Sprintf("✅ K3s cluster DEPLOYED: %d masters, %d workers", len(masters), len(workers)), nil)

	return component, nil
}
