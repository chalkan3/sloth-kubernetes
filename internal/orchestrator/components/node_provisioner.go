package components

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// RealNodeProvisioningComponent provisions a node with Docker and Kubernetes prerequisites via SSH
type RealNodeProvisioningComponent struct {
	pulumi.ResourceState
	NodeName      pulumi.StringOutput `pulumi:"nodeName"`
	InstallOutput pulumi.StringOutput `pulumi:"installOutput"`
	DockerVersion pulumi.StringOutput `pulumi:"dockerVersion"`
	Status        pulumi.StringOutput `pulumi:"status"`
}

// NewRealNodeProvisioningComponent creates real provisioning via SSH using remote.Command
// If bastionIP is provided (non-zero value), SSH connections will use ProxyJump through the bastion
func NewRealNodeProvisioningComponent(ctx *pulumi.Context, name string, nodeIP pulumi.StringOutput, nodeName string, sshPrivateKey pulumi.StringOutput, bastionIP *pulumi.StringOutput, opts ...pulumi.ResourceOption) (*RealNodeProvisioningComponent, error) {
	component := &RealNodeProvisioningComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:provisioning:NodeProvisioning", name, component, opts...)
	if err != nil {
		return nil, err
	}

	component.NodeName = pulumi.String(nodeName).ToStringOutput()

	// Script de instalação de pré-requisitos do Kubernetes
	installScript := `#!/bin/bash
set -e

echo "=========================================="
echo "[$(date +%H:%M:%S)] 🖥️  NODE PROVISIONING: $(hostname)"
echo "=========================================="
echo "[$(date +%H:%M:%S)] Started at: $(date)"
echo ""

# Function to wait for apt-get lock
wait_for_apt_lock() {
    local MAX_WAIT=300  # 5 minutes max
    local ELAPSED=0
    echo "[$(date +%H:%M:%S)] Checking for apt-get lock..."
    while fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1 || fuser /var/lib/apt/lists/lock >/dev/null 2>&1 || fuser /var/lib/dpkg/lock >/dev/null 2>&1; do
        if [ $ELAPSED -ge $MAX_WAIT ]; then
            echo "[$(date +%H:%M:%S)] ❌ ERROR: apt-get lock still held after ${MAX_WAIT}s"
            killall -9 apt-get apt dpkg unattended-upgrades || true
            sleep 5
            rm -f /var/lib/dpkg/lock-frontend /var/lib/dpkg/lock /var/lib/apt/lists/lock 2>/dev/null || true
            dpkg --configure -a || true
            break
        fi
        echo "[$(date +%H:%M:%S)]   ⏳ Waiting for apt lock... (${ELAPSED}s elapsed)"
        sleep 5
        ELAPSED=$((ELAPSED + 5))
    done
    echo "[$(date +%H:%M:%S)] ✅ apt-get lock released"
}

# Function to run apt-get commands with retry and mirror fallback
apt_get_with_retry() {
    local MAX_RETRIES=5
    local RETRY_COUNT=0
    local SWITCHED_MIRROR=false

    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        wait_for_apt_lock

        echo "[$(date +%H:%M:%S)] 🔄 Executing: $@"
        if "$@"; then
            echo "[$(date +%H:%M:%S)] ✅ Command succeeded"
            return 0
        else
            RETRY_COUNT=$((RETRY_COUNT + 1))

            # On 3rd failure, try switching to Ubuntu official mirrors
            if [ $RETRY_COUNT -eq 3 ] && [ "$SWITCHED_MIRROR" = "false" ] && grep -q "mirrors.digitalocean.com" /etc/apt/sources.list 2>/dev/null; then
                echo "[$(date +%H:%M:%S)] ⚠️  Switching to official Ubuntu mirrors..."
                sed -i.bak 's|http://mirrors.digitalocean.com/ubuntu|http://archive.ubuntu.com/ubuntu|g' /etc/apt/sources.list
                SWITCHED_MIRROR=true
                echo "[$(date +%H:%M:%S)] 📝 Mirror switched, retrying..."
            fi

            if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
                echo "[$(date +%H:%M:%S)] ⚠️  Retrying in 10s... (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)"
                sleep 10
            else
                echo "[$(date +%H:%M:%S)] ❌ Command failed after $MAX_RETRIES attempts"
                return 1
            fi
        fi
    done
}

# Disable unattended-upgrades to prevent conflicts
echo ""
echo "[$(date +%H:%M:%S)] STEP 1: Preparing system"
echo "[$(date +%H:%M:%S)] Disabling unattended-upgrades..."
systemctl stop unattended-upgrades || true
systemctl disable unattended-upgrades || true
killall -9 unattended-upgrades || true

# Initial lock wait
wait_for_apt_lock

# Update system
echo ""
echo "[$(date +%H:%M:%S)] STEP 2: Updating system packages"
export DEBIAN_FRONTEND=noninteractive
apt_get_with_retry apt-get update
apt_get_with_retry apt-get upgrade -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"

# CRITICAL: Wait 30 seconds after apt-get upgrade to ensure all locks are released
echo "[$(date +%H:%M:%S)] ⏳ Waiting 30s for all apt locks to be released..."
sleep 30

# Install Docker
echo ""
echo "[$(date +%H:%M:%S)] STEP 3: Installing Docker"
if ! command -v docker &> /dev/null; then
    echo "[$(date +%H:%M:%S)] Docker not found, starting installation..."

    # Download Docker install script first (before any cleanup)
    echo "[$(date +%H:%M:%S)] Downloading Docker installation script..."
    curl -fsSL https://get.docker.com -o /tmp/get-docker.sh || {
        echo "[$(date +%H:%M:%S)] ❌ Failed to download Docker script"
        exit 1
    }

    # Run docker install with multiple retry attempts
    MAX_INSTALL_RETRIES=5
    INSTALL_ATTEMPT=0
    while [ $INSTALL_ATTEMPT -lt $MAX_INSTALL_RETRIES ]; do
        echo ""
        echo "[$(date +%H:%M:%S)] 🐋 Docker installation attempt $((INSTALL_ATTEMPT + 1))/$MAX_INSTALL_RETRIES..."

        # Aggressive cleanup before each attempt
        echo "  → Performing aggressive apt cleanup..."
        killall -9 apt-get apt dpkg unattended-upgrades 2>/dev/null || true
        sleep 3
        rm -f /var/lib/dpkg/lock-frontend /var/lib/dpkg/lock /var/lib/apt/lists/lock 2>/dev/null || true
        rm -f /var/cache/apt/archives/lock 2>/dev/null || true
        dpkg --configure -a 2>&1 || true

        echo "  → Waiting for apt locks to be released..."
        wait_for_apt_lock

        # Run Docker installation (NOTE: get-docker.sh may return 0 even on failure!)
        # We MUST verify Docker is actually installed afterwards
        echo "  → Running Docker installation script..."
        DEBIAN_FRONTEND=noninteractive sh /tmp/get-docker.sh 2>&1 | tee /tmp/docker-install.log || true

        # CRITICAL: Check if Docker was ACTUALLY installed (don't trust exit code!)
        sleep 5  # Give Docker binaries time to be installed
        if command -v docker &> /dev/null; then
            echo "  ✅ Docker binary found! Verifying installation..."

            # Start Docker daemon if not running
            systemctl enable docker 2>/dev/null || true
            systemctl start docker 2>/dev/null || true
            sleep 3

            # Verify daemon is running
            if systemctl is-active --quiet docker; then
                echo "  ✅ Docker daemon is running"
                docker --version
                break
            else
                echo "  ⚠️  Docker binary exists but daemon failed to start"
                systemctl status docker || true
            fi
        else
            echo "  ❌ Docker binary NOT found after installation script ran"
        fi

        # If we reach here, installation failed
        INSTALL_ATTEMPT=$((INSTALL_ATTEMPT + 1))
        if [ $INSTALL_ATTEMPT -lt $MAX_INSTALL_RETRIES ]; then
            echo ""
            echo "  ⚠️  Installation failed, will retry in 60s... (attempt $((INSTALL_ATTEMPT + 1))/$MAX_INSTALL_RETRIES)"
            echo "  Last install log (last 30 lines):"
            tail -30 /tmp/docker-install.log 2>/dev/null || echo "  No log available"

            # More aggressive cleanup
            killall -9 apt-get apt dpkg unattended-upgrades 2>/dev/null || true
            rm -f /var/lib/dpkg/lock-frontend /var/lib/dpkg/lock /var/lib/apt/lists/lock 2>/dev/null || true
            apt-get clean 2>/dev/null || true
            sleep 60  # Increased from 45s to 60s
        else
            echo ""
            echo "  ❌ Docker installation failed after $MAX_INSTALL_RETRIES attempts"
            echo "  === Full installation log ==="
            cat /tmp/docker-install.log 2>/dev/null || echo "  No log available"
            exit 1
        fi
    done

    # Final verification that Docker was actually installed
    if ! command -v docker &> /dev/null; then
        echo "❌ Docker command not found after installation loop completed"
        exit 1
    fi

    systemctl enable docker
    systemctl start docker
    sleep 3

    # Verify Docker daemon is running
    if ! systemctl is-active --quiet docker; then
        echo "❌ Docker daemon failed to start"
        systemctl status docker || true
        exit 1
    fi

    echo "✅ Docker installed and running"
else
    echo "✅ Docker already installed"
fi

# Install required packages
echo ""
echo "[$(date +%H:%M:%S)] STEP 4: Installing Kubernetes prerequisites"
echo "[$(date +%H:%M:%S)] Installing: apt-transport-https, ca-certificates, curl, nfs-common..."
apt_get_with_retry apt-get install -y apt-transport-https ca-certificates curl software-properties-common nfs-common

# Disable swap (required for Kubernetes)
echo ""
echo "[$(date +%H:%M:%S)] STEP 5: Configuring system for Kubernetes"
echo "[$(date +%H:%M:%S)] Disabling swap..."
swapoff -a
sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

# Enable kernel modules
echo "[$(date +%H:%M:%S)] Enabling kernel modules (br_netfilter, overlay)..."
modprobe br_netfilter 2>/dev/null || true
modprobe overlay 2>/dev/null || true
cat > /etc/modules-load.d/k8s.conf << EOF
br_netfilter
overlay
EOF

# Configure sysctl for Kubernetes
echo "[$(date +%H:%M:%S)] Configuring sysctl parameters..."
cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sysctl --system >/dev/null 2>&1
echo "[$(date +%H:%M:%S)] ✅ Kubernetes system configuration complete"

# Install WireGuard
echo ""
echo "[$(date +%H:%M:%S)] STEP 6: Installing WireGuard"
if ! command -v wg &> /dev/null; then
    echo "[$(date +%H:%M:%S)] Installing WireGuard tools..."
    apt_get_with_retry apt-get install -y wireguard wireguard-tools
    echo "[$(date +%H:%M:%S)] ✅ WireGuard installed"
else
    echo "[$(date +%H:%M:%S)] ✅ WireGuard already installed"
fi

echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] ✅ NODE PROVISIONING COMPLETE!"
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] All prerequisites installed successfully!"
docker --version
wg --version 2>&1 | head -1 || echo "WireGuard tools ready"
`

	// Add UFW firewall configuration if bastion is enabled
	// NOTE: We need to do this AFTER the initial script because bastionIP is a Pulumi Output
	// We'll create a separate firewall configuration command that depends on the bastion

	// Execute installation via SSH using remote.Command
	// Note: DialErrorLimit controls SSH connection retries (default is 10)
	// New VMs can take 2-3 minutes to boot, some cloud providers are slower
	// CRITICAL: Linode instances in particular can take 3-5 minutes to be SSH-ready

	// Build connection args - use ProxyJump if bastion is provided
	connectionArgs := remote.ConnectionArgs{
		Host:           nodeIP,
		User:           pulumi.String("root"),
		PrivateKey:     sshPrivateKey,
		DialErrorLimit: pulumi.Int(60), // Retry SSH connection up to 60 times (increased from 30)
	}

	// If bastion is provided, use ProxyJump
	if bastionIP != nil {
		connectionArgs.Proxy = &remote.ProxyConnectionArgs{
			Host:       *bastionIP,
			User:       pulumi.String("root"),
			PrivateKey: sshPrivateKey,
		}
		ctx.Log.Info(fmt.Sprintf("🏰 Using bastion ProxyJump for %s", nodeName), nil)
	}

	ctx.Log.Info(fmt.Sprintf("⏳ Provisioning node %s (this may take 5-10 minutes)...", nodeName), nil)
	ctx.Log.Info("   → Installing Docker", nil)
	ctx.Log.Info("   → Installing WireGuard", nil)
	ctx.Log.Info("   → Configuring Kubernetes prerequisites", nil)
	ctx.Log.Info("", nil)
	ctx.Log.Info("💡 Note: Pulumi doesn't show real-time output from remote commands.", nil)
	ctx.Log.Info("   The process is still running - please wait...", nil)

	installCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-install", name), &remote.CommandArgs{
		Connection: connectionArgs,
		Create:     pulumi.String(installScript),
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("failed to create install command: %w", err)
	}

	ctx.Log.Info(fmt.Sprintf("✅ Node %s provisioning command completed", nodeName), nil)

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

	// Build connection args for validation (reuse bastion proxy if provided)
	validationConnectionArgs := remote.ConnectionArgs{
		Host:           nodeIP,
		User:           pulumi.String("root"),
		PrivateKey:     sshPrivateKey,
		DialErrorLimit: pulumi.Int(20), // Increased from 5 for more robust validation
	}

	// Use ProxyJump for validation too
	if bastionIP != nil {
		validationConnectionArgs.Proxy = &remote.ProxyConnectionArgs{
			Host:       *bastionIP,
			User:       pulumi.String("root"),
			PrivateKey: sshPrivateKey,
		}
	}

	validateCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-validate", name), &remote.CommandArgs{
		Connection: validationConnectionArgs,
		Create:     pulumi.String(validationScript),
	}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{installCmd}))
	if err != nil {
		return nil, fmt.Errorf("failed to create validation command: %w", err)
	}

	// Get Docker version to confirm installation
	component.DockerVersion = validateCmd.Stdout.ApplyT(func(stdout string) string {
		return "Dependencies validated successfully"
	}).(pulumi.StringOutput)

	// If bastion is enabled, configure UFW firewall to restrict SSH access
	if bastionIP != nil {
		ctx.Log.Info(fmt.Sprintf("🔒 Configuring UFW firewall on %s (restricting SSH to bastion only)...", nodeName), nil)

		// Create firewall configuration script
		// This will be executed with the bastion IP injected dynamically using Pulumi.Apply
		_, err := remote.NewCommand(ctx, fmt.Sprintf("%s-firewall", name), &remote.CommandArgs{
			Connection: validationConnectionArgs,
			Create: bastionIP.ToStringOutput().ApplyT(func(bIP string) string {
				return fmt.Sprintf(`#!/bin/bash
set -e

echo "=========================================="
echo "[$(date +%%H:%%M:%%S)] 🔒 UFW FIREWALL CONFIGURATION"
echo "=========================================="
echo "[$(date +%%H:%%M:%%S)] Node: $(hostname)"
echo "[$(date +%%H:%%M:%%S)] Bastion IP: %s"
echo ""

# Install UFW if not already installed
if ! command -v ufw &> /dev/null; then
    echo "[$(date +%%H:%%M:%%S)] Installing UFW..."
    apt-get update -qq
    apt-get install -y ufw
fi

echo "[$(date +%%H:%%M:%%S)] Configuring UFW firewall rules..."

# Disable UFW first to configure safely
ufw --force disable

# Reset to default state
echo "y" | ufw --force reset

# Set default policies
ufw default deny incoming
ufw default allow outgoing

# CRITICAL: Allow SSH from bastion IP ONLY
# This prevents direct SSH access from the internet
echo "[$(date +%%H:%%M:%%S)] → Allowing SSH (port 22) from bastion IP: %s"
ufw allow from %s to any port 22 proto tcp comment 'SSH from bastion only'

# Allow Kubernetes API Server (required for kubectl access)
echo "[$(date +%%H:%%M:%%S)] → Allowing Kubernetes API (port 6443)"
ufw allow 6443/tcp comment 'Kubernetes API Server'

# Allow WireGuard VPN (required for mesh networking)
echo "[$(date +%%H:%%M:%%S)] → Allowing WireGuard VPN (port 51820)"
ufw allow 51820/udp comment 'WireGuard VPN'

# Allow HTTP/HTTPS for ingress traffic
echo "[$(date +%%H:%%M:%%S)] → Allowing HTTP/HTTPS (ports 80, 443)"
ufw allow 80/tcp comment 'HTTP Ingress'
ufw allow 443/tcp comment 'HTTPS Ingress'

# Allow traffic from WireGuard subnet (10.8.0.0/24)
echo "[$(date +%%H:%%M:%%S)] → Allowing all traffic from WireGuard subnet (10.8.0.0/24)"
ufw allow from 10.8.0.0/24 comment 'WireGuard mesh traffic'

# Allow traffic from VPC subnets (10.10.0.0/16 and 10.21.0.0/24)
echo "[$(date +%%H:%%M:%%S)] → Allowing traffic from VPC subnets"
ufw allow from 10.10.0.0/16 comment 'DigitalOcean VPC'
ufw allow from 10.21.0.0/24 comment 'Linode VPC'

# Enable IP forwarding in UFW (required for Kubernetes)
sed -i 's/DEFAULT_FORWARD_POLICY="DROP"/DEFAULT_FORWARD_POLICY="ACCEPT"/' /etc/default/ufw || true

# Enable UFW
echo "[$(date +%%H:%%M:%%S)] Enabling UFW..."
echo "y" | ufw --force enable

# Show status
echo ""
echo "[$(date +%%H:%%M:%%S)] =========================================="
echo "[$(date +%%H:%%M:%%S)] UFW Firewall Status:"
echo "[$(date +%%H:%%M:%%S)] =========================================="
ufw status verbose

echo ""
echo "[$(date +%%H:%%M:%%S)] ✅ UFW FIREWALL CONFIGURED SUCCESSFULLY"
echo "[$(date +%%H:%%M:%%S)] → Direct SSH from internet: BLOCKED"
echo "[$(date +%%H:%%M:%%S)] → SSH from bastion (%s): ALLOWED"
echo "[$(date +%%H:%%M:%%S)] → Kubernetes API, WireGuard, HTTP/HTTPS: ALLOWED"
echo "[$(date +%%H:%%M:%%S)] =========================================="
`, bIP, bIP, bIP, bIP)
			}).(pulumi.StringOutput),
		}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{validateCmd}))

		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("Failed to create firewall configuration for %s: %v", nodeName, err), nil)
		} else {
			ctx.Log.Info(fmt.Sprintf("✅ UFW firewall configured on %s", nodeName), nil)
		}
	}

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
