package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// RealNodeProvisioningComponent provisions a node with Docker and Kubernetes prerequisites via SSH
type RealNodeProvisioningComponent struct {
	pulumi.ResourceState
	NodeName       pulumi.StringOutput `pulumi:"nodeName"`
	InstallOutput  pulumi.StringOutput `pulumi:"installOutput"`
	DockerVersion  pulumi.StringOutput `pulumi:"dockerVersion"`
	Status         pulumi.StringOutput `pulumi:"status"`
}

// NewRealNodeProvisioningComponent creates real provisioning via SSH using remote.Command
func NewRealNodeProvisioningComponent(ctx *pulumi.Context, name string, nodeIP pulumi.StringOutput, nodeName string, sshPrivateKey pulumi.StringOutput, parent pulumi.Resource) (*RealNodeProvisioningComponent, error) {
	component := &RealNodeProvisioningComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:provisioning:NodeProvisioning", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.NodeName = pulumi.String(nodeName).ToStringOutput()

	// Script de instalação de pré-requisitos do Kubernetes
	installScript := `#!/bin/bash
set -e

echo "=== Installing Kubernetes prerequisites on $(hostname) ==="

# Function to wait for apt-get lock
wait_for_apt_lock() {
    local MAX_WAIT=300  # 5 minutes max
    local ELAPSED=0
    echo "Waiting for apt-get lock to be released..."
    while fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1 || fuser /var/lib/apt/lists/lock >/dev/null 2>&1 || fuser /var/lib/dpkg/lock >/dev/null 2>&1; do
        if [ $ELAPSED -ge $MAX_WAIT ]; then
            echo "ERROR: apt-get lock still held after ${MAX_WAIT}s, killing processes..."
            killall -9 apt-get apt dpkg unattended-upgrades || true
            sleep 5
            rm -f /var/lib/dpkg/lock-frontend /var/lib/dpkg/lock /var/lib/apt/lists/lock 2>/dev/null || true
            dpkg --configure -a || true
            break
        fi
        echo "  Waiting for apt lock... (${ELAPSED}s elapsed)"
        sleep 5
        ELAPSED=$((ELAPSED + 5))
    done
    echo "✅ apt-get lock released"
}

# Function to run apt-get commands with retry
apt_get_with_retry() {
    local MAX_RETRIES=5
    local RETRY_COUNT=0

    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        wait_for_apt_lock

        if "$@"; then
            return 0
        else
            RETRY_COUNT=$((RETRY_COUNT + 1))
            if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
                echo "Command failed, retrying in 10 seconds... (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)"
                sleep 10
            else
                echo "Command failed after $MAX_RETRIES attempts"
                return 1
            fi
        fi
    done
}

# Disable unattended-upgrades to prevent conflicts
echo "Disabling unattended-upgrades..."
systemctl stop unattended-upgrades || true
systemctl disable unattended-upgrades || true
killall -9 unattended-upgrades || true

# Initial lock wait
wait_for_apt_lock

# Update system
export DEBIAN_FRONTEND=noninteractive
apt_get_with_retry apt-get update
apt_get_with_retry apt-get upgrade -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"

# Install Docker
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."

    # Wait for lock and kill unattended-upgrades if needed
    wait_for_apt_lock

    # Download Docker install script
    curl -fsSL https://get.docker.com -o /tmp/get-docker.sh

    # Export retry function to subshell and run docker install
    export -f wait_for_apt_lock apt_get_with_retry

    # Run docker install with multiple retry attempts
    MAX_INSTALL_RETRIES=3
    INSTALL_ATTEMPT=0
    while [ $INSTALL_ATTEMPT -lt $MAX_INSTALL_RETRIES ]; do
        wait_for_apt_lock

        if sh /tmp/get-docker.sh 2>&1; then
            echo "✅ Docker installation succeeded"
            break
        else
            INSTALL_ATTEMPT=$((INSTALL_ATTEMPT + 1))
            if [ $INSTALL_ATTEMPT -lt $MAX_INSTALL_RETRIES ]; then
                echo "⚠️  Docker installation failed, waiting 30s and retrying... (attempt $((INSTALL_ATTEMPT + 1))/$MAX_INSTALL_RETRIES)"
                sleep 30
                # Kill any remaining apt processes
                killall -9 apt-get apt dpkg unattended-upgrades 2>/dev/null || true
                sleep 5
            else
                echo "❌ Docker installation failed after $MAX_INSTALL_RETRIES attempts"
                exit 1
            fi
        fi
    done

    systemctl enable docker
    systemctl start docker
    echo "✅ Docker installed and running"
else
    echo "✅ Docker already installed"
fi

# Install required packages
echo "Installing additional packages..."
apt_get_with_retry apt-get install -y apt-transport-https ca-certificates curl software-properties-common nfs-common

# Disable swap (required for Kubernetes)
echo "Disabling swap..."
swapoff -a
sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

# Enable kernel modules
echo "Enabling kernel modules..."
modprobe br_netfilter 2>/dev/null || true
modprobe overlay 2>/dev/null || true
cat > /etc/modules-load.d/k8s.conf << EOF
br_netfilter
overlay
EOF

# Configure sysctl for Kubernetes
echo "Configuring sysctl..."
cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sysctl --system >/dev/null 2>&1

# Install WireGuard
if ! command -v wg &> /dev/null; then
    echo "Installing WireGuard..."
    apt_get_with_retry apt-get install -y wireguard wireguard-tools
    echo "✅ WireGuard installed"
else
    echo "✅ WireGuard already installed"
fi

echo ""
echo "✅ ✅ ✅ All prerequisites installed successfully!"
docker --version
wg --version 2>&1 | head -1 || echo "WireGuard tools ready"
`

	// Execute installation via SSH using remote.Command
	// Note: DialErrorLimit controls SSH connection retries (default is 10)
	// New VMs can take 2-3 minutes to boot, so we increase this
	installCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-install", name), &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:           nodeIP,
			User:           pulumi.String("root"),
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30), // Retry SSH connection up to 30 times
		},
		Create: pulumi.String(installScript),
	}, pulumi.Parent(component), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "15m", // Increased timeout for slow boot + installation
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create install command: %w", err)
	}

	component.InstallOutput = installCmd.Stdout

	// CRITICAL: Validate that all dependencies were actually installed before proceeding
	// This prevents the issue where remote commands show "success" but installations didn't actually run
	ctx.Log.Info(fmt.Sprintf("🔍 Validating dependencies on %s...", nodeName), nil)

	validationScript := `#!/bin/bash
set -e

echo "🔍 Validating dependencies on $(hostname)..."

# Check Docker
if ! docker --version | grep -q "Docker version"; then
	echo "❌ VALIDATION FAILED: Docker not found"
	exit 1
fi
echo "✅ Docker: $(docker --version)"

# Check WireGuard tools
if ! wg --version 2>&1 | grep -q "wireguard-tools"; then
	echo "❌ VALIDATION FAILED: WireGuard tools not found"
	exit 1
fi
echo "✅ WireGuard: $(wg --version 2>&1 | head -1)"

# Check IP forwarding
if ! sysctl net.ipv4.ip_forward | grep -q "= 1"; then
	echo "⚠️  IP forwarding not enabled, enabling now..."
	sysctl -w net.ipv4.ip_forward=1
	echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
fi
echo "✅ IP forwarding: $(sysctl net.ipv4.ip_forward)"

# Check kernel modules
if ! lsmod | grep -q br_netfilter; then
	echo "⚠️  br_netfilter not loaded"
	modprobe br_netfilter
fi
echo "✅ Kernel modules: br_netfilter loaded"

# Check disk space
USED=$(df -h / | tail -1 | awk '{print $5}' | sed 's/%//')
if [ "$USED" -gt 85 ]; then
	echo "⚠️  WARNING: Disk usage is ${USED}% (> 85%)"
else
	echo "✅ Disk space: ${USED}% used"
fi

echo ""
echo "✅ ✅ ✅ ALL DEPENDENCY VALIDATIONS PASSED on $(hostname)"
echo "Ready for WireGuard and RKE2 installation"
`

	validateCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-validate", name), &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:           nodeIP,
			User:           pulumi.String("root"),
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(5),
		},
		Create: pulumi.String(validationScript),
	}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{installCmd}), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "3m",
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create validation command: %w", err)
	}

	// Get Docker version to confirm installation
	component.DockerVersion = validateCmd.Stdout.ApplyT(func(stdout string) string {
		return "Dependencies validated successfully"
	}).(pulumi.StringOutput)

	component.Status = pulumi.Sprintf("✅ Provisioned and validated %s", nodeName)

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"nodeName":      component.NodeName,
		"installOutput": component.InstallOutput,
		"dockerVersion": component.DockerVersion,
		"status":        component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
