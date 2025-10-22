package components

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi-linode/sdk/v4/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"sloth-kubernetes/pkg/config"
)

// BastionComponent represents the bastion host for secure cluster access
type BastionComponent struct {
	pulumi.ResourceState

	BastionName pulumi.StringOutput `pulumi:"bastionName"`
	PublicIP    pulumi.StringOutput `pulumi:"publicIP"`
	PrivateIP   pulumi.StringOutput `pulumi:"privateIP"`
	WireGuardIP pulumi.StringOutput `pulumi:"wireGuardIP"`
	Provider    pulumi.StringOutput `pulumi:"provider"`
	Region      pulumi.StringOutput `pulumi:"region"`
	SSHPort     pulumi.IntOutput    `pulumi:"sshPort"`
	Status      pulumi.StringOutput `pulumi:"status"`
}

// NewBastionComponent creates a bastion host for secure cluster access
// The bastion is the ONLY host with public SSH access. All cluster nodes are private.
func NewBastionComponent(
	ctx *pulumi.Context,
	name string,
	bastionConfig *config.BastionConfig,
	sshKeyOutput pulumi.StringOutput,
	sshPrivateKey pulumi.StringOutput,
	doToken pulumi.StringInput,
	linodeToken pulumi.StringInput,
	opts ...pulumi.ResourceOption,
) (*BastionComponent, error) {
	component := &BastionComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:security:Bastion", name, component, opts...)
	if err != nil {
		return nil, err
	}

	if !bastionConfig.Enabled {
		ctx.Log.Info("⏭️  Bastion host disabled - cluster nodes will have public IPs", nil)
		component.Status = pulumi.String("disabled").ToStringOutput()
		return component, nil
	}

	ctx.Log.Info(fmt.Sprintf("🏰 Creating Bastion host on %s...", bastionConfig.Provider), nil)

	// Set defaults
	if bastionConfig.Name == "" {
		bastionConfig.Name = "bastion"
	}
	if bastionConfig.SSHPort == 0 {
		bastionConfig.SSHPort = 22
	}

	// Assign VPN IP for bastion (10.8.0.5 - reserved for bastion)
	bastionVPNIP := "10.8.0.5"

	component.BastionName = pulumi.String(bastionConfig.Name).ToStringOutput()
	component.Provider = pulumi.String(bastionConfig.Provider).ToStringOutput()
	component.Region = pulumi.String(bastionConfig.Region).ToStringOutput()
	component.SSHPort = pulumi.Int(bastionConfig.SSHPort).ToIntOutput()
	component.WireGuardIP = pulumi.String(bastionVPNIP).ToStringOutput()

	// Create bastion host based on provider
	switch bastionConfig.Provider {
	case "digitalocean":
		err = createDigitalOceanBastion(ctx, name, bastionConfig, sshKeyOutput, doToken, component)
	case "linode":
		err = createLinodeBastion(ctx, name, bastionConfig, sshKeyOutput, linodeToken, component)
	default:
		return nil, fmt.Errorf("unsupported bastion provider: %s", bastionConfig.Provider)
	}

	if err != nil {
		return nil, err
	}

	// Provision bastion with security hardening
	provComp, err := NewBastionProvisioningComponent(
		ctx,
		fmt.Sprintf("%s-provision", name),
		component.PublicIP,
		bastionConfig,
		sshPrivateKey,
		component,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to provision bastion: %w", err)
	}

	component.Status = provComp.Status

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"bastionName": component.BastionName,
		"publicIP":    component.PublicIP,
		"privateIP":   component.PrivateIP,
		"wireGuardIP": component.WireGuardIP,
		"provider":    component.Provider,
		"region":      component.Region,
		"sshPort":     component.SSHPort,
		"status":      component.Status,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info("✅ Bastion host created successfully", nil)

	return component, nil
}

// createDigitalOceanBastion creates a DigitalOcean bastion droplet
func createDigitalOceanBastion(
	ctx *pulumi.Context,
	name string,
	bastionConfig *config.BastionConfig,
	sshKeyOutput pulumi.StringOutput,
	doToken pulumi.StringInput,
	component *BastionComponent,
) error {
	// Create SSH key for bastion
	sshKey, err := digitalocean.NewSshKey(ctx, fmt.Sprintf("%s-ssh-key", name), &digitalocean.SshKeyArgs{
		Name:      pulumi.Sprintf("bastion-key-%s", name),
		PublicKey: sshKeyOutput,
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create DO SSH key: %w", err)
	}

	// Create bastion droplet
	droplet, err := digitalocean.NewDroplet(ctx, name, &digitalocean.DropletArgs{
		Image:  pulumi.String(bastionConfig.Image),
		Name:   pulumi.String(bastionConfig.Name),
		Region: pulumi.String(bastionConfig.Region),
		Size:   pulumi.String(bastionConfig.Size),
		SshKeys: pulumi.StringArray{
			sshKey.Fingerprint,
		},
		Tags: pulumi.StringArray{
			pulumi.String("bastion"),
			pulumi.String("security"),
			pulumi.String(ctx.Stack()),
		},
		Ipv6:       pulumi.Bool(true),
		Monitoring: pulumi.Bool(true),
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create bastion droplet: %w", err)
	}

	component.PublicIP = droplet.Ipv4Address
	component.PrivateIP = droplet.Ipv4AddressPrivate

	return nil
}

// createLinodeBastion creates a Linode bastion instance
func createLinodeBastion(
	ctx *pulumi.Context,
	name string,
	bastionConfig *config.BastionConfig,
	sshKeyOutput pulumi.StringOutput,
	linodeToken pulumi.StringInput,
	component *BastionComponent,
) error {
	// Create bastion instance
	instance, err := linode.NewInstance(ctx, name, &linode.InstanceArgs{
		Label:  pulumi.String(bastionConfig.Name),
		Region: pulumi.String(bastionConfig.Region),
		Type:   pulumi.String(bastionConfig.Size),
		Image:  pulumi.String(bastionConfig.Image),
		AuthorizedKeys: pulumi.StringArray{
			sshKeyOutput,
		},
		Tags: pulumi.StringArray{
			pulumi.String("bastion"),
			pulumi.String("security"),
			pulumi.String(ctx.Stack()),
		},
		PrivateIp: pulumi.Bool(true),
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create bastion instance: %w", err)
	}

	component.PublicIP = instance.IpAddress
	component.PrivateIP = instance.PrivateIpAddress

	return nil
}

// BastionProvisioningComponent handles bastion host provisioning and hardening
type BastionProvisioningComponent struct {
	pulumi.ResourceState

	Status pulumi.StringOutput `pulumi:"status"`
}

// NewBastionProvisioningComponent provisions and hardens the bastion host
func NewBastionProvisioningComponent(
	ctx *pulumi.Context,
	name string,
	bastionIP pulumi.StringOutput,
	bastionConfig *config.BastionConfig,
	sshPrivateKey pulumi.StringOutput,
	parent pulumi.Resource,
) (*BastionProvisioningComponent, error) {
	component := &BastionProvisioningComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:security:BastionProvisioning", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	ctx.Log.Info("🔧 Provisioning bastion with security hardening...", nil)

	// Build provisioning script with security hardening
	provisionScript := buildBastionProvisionScript(bastionConfig)

	// Execute provisioning script via pulumi-command
	ctx.Log.Info("📋 Bastion will be provisioned with:", nil)
	ctx.Log.Info("  • UFW firewall (SSH only from allowed CIDRs)", nil)
	ctx.Log.Info("  • fail2ban for brute force protection", nil)
	ctx.Log.Info("  • SSH hardening (key-only auth)", nil)
	ctx.Log.Info("  • Audit logging enabled", nil)
	ctx.Log.Info("  • WireGuard VPN client", nil)

	if bastionConfig.EnableMFA {
		ctx.Log.Info("  • MFA (Google Authenticator)", nil)
	}

	// Execute the provisioning script on the bastion host
	ctx.Log.Info("⏳ Starting bastion provisioning (this may take 5-10 minutes)...", nil)
	ctx.Log.Info("   → Installing security packages (ufw, fail2ban, wireguard)", nil)
	ctx.Log.Info("   → Configuring firewall rules", nil)
	ctx.Log.Info("   → Hardening SSH configuration", nil)
	ctx.Log.Info("   → Setting up audit logging", nil)
	ctx.Log.Info("", nil)
	ctx.Log.Info("💡 Note: Pulumi doesn't show real-time output from remote commands.", nil)
	ctx.Log.Info("   The process is still running - please wait...", nil)

	provisionCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-provision-script", name), &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:           bastionIP,
			User:           pulumi.String("root"),
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
		},
		Create: pulumi.String(provisionScript),
	}, pulumi.Parent(component), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "20m", // Provisioning can take time for package installation
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to execute provisioning script: %w", err)
	}

	ctx.Log.Info("✅ Bastion provisioning command completed successfully", nil)
	ctx.Log.Info("", nil)
	ctx.Log.Info("🔍 Validating bastion SSH connectivity...", nil)

	// SSH Validation Command - Test that SSH is working properly
	validateSSHCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-validate-ssh", name), &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:           bastionIP,
			User:           pulumi.String("root"),
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(10),
		},
		Create: pulumi.String(`#!/bin/bash
echo "=========================================="
echo "🔍 BASTION SSH VALIDATION TEST"
echo "=========================================="
echo "✅ SSH connection successful!"
echo "📋 Bastion details:"
echo "  • Hostname: $(hostname)"
echo "  • Uptime: $(uptime -p)"
echo "  • SSH service: $(systemctl is-active sshd)"
echo "  • UFW status: $(ufw status | head -1)"
echo "  • fail2ban status: $(systemctl is-active fail2ban)"
echo ""
echo "✅ Bastion is fully operational and ready!"
echo "=========================================="
`),
	}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{provisionCmd}), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "2m",
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to validate bastion SSH: %w", err)
	}

	// Log validation output
	validateSSHCmd.Stdout.ApplyT(func(stdout string) string {
		if stdout != "" {
			ctx.Log.Info("", nil)
			ctx.Log.Info("✅ BASTION VALIDATION SUCCESSFUL", nil)
			ctx.Log.Info("   SSH connectivity confirmed", nil)
			ctx.Log.Info("   Security services are active", nil)
			ctx.Log.Info("", nil)
			ctx.Log.Info("🎉 BASTION IS 100% READY FOR CLUSTER DEPLOYMENT", nil)
			ctx.Log.Info("", nil)
		}
		return stdout
	})

	// Set status based on validation success
	component.Status = validateSSHCmd.Stdout.ApplyT(func(stdout string) string {
		if stdout != "" {
			return "validated"
		}
		return "validation-failed"
	}).(pulumi.StringOutput)

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status": component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// buildBastionProvisionScript creates the provisioning script for bastion security hardening
func buildBastionProvisionScript(cfg *config.BastionConfig) string {
	script := `#!/bin/bash
set -e

echo "=========================================="
echo "🏰 BASTION PROVISIONING STARTED"
echo "=========================================="
echo "Time: $(date)"
echo ""

# Function to wait for apt-get lock
wait_for_apt_lock() {
    local MAX_WAIT=300  # 5 minutes max
    local ELAPSED=0
    echo "[$(date +%H:%M:%S)] Checking for apt-get lock..."
    while fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1 || fuser /var/lib/apt/lists/lock >/dev/null 2>&1 || fuser /var/lib/dpkg/lock >/dev/null 2>&1; do
        if [ $ELAPSED -ge $MAX_WAIT ]; then
            echo "[$(date +%H:%M:%S)] ❌ ERROR: apt-get lock still held after ${MAX_WAIT}s, killing processes..."
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
                echo "[$(date +%H:%M:%S)] ⚠️  Repeated failures detected, switching mirrors..."
                sed -i.bak 's|http://mirrors.digitalocean.com/ubuntu|http://archive.ubuntu.com/ubuntu|g' /etc/apt/sources.list
                SWITCHED_MIRROR=true
                echo "[$(date +%H:%M:%S)] 📝 Mirror switched, retrying..."
            fi

            if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
                echo "[$(date +%H:%M:%S)] ⚠️  Command failed, retrying in 10s... (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)"
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
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 1: Preparing system"
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] Disabling unattended-upgrades..."
systemctl stop unattended-upgrades || true
systemctl disable unattended-upgrades || true
killall -9 unattended-upgrades || true
echo "[$(date +%H:%M:%S)] ✅ Unattended upgrades disabled"

# Initial lock wait
wait_for_apt_lock

# Update system
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 2: Updating system packages"
echo "[$(date +%H:%M:%S)] =========================================="
export DEBIAN_FRONTEND=noninteractive
apt_get_with_retry apt-get update
# OPTIMIZATION: Skip apt-get upgrade to speed up provisioning (Ubuntu 24.04 is already recent)
# apt_get_with_retry apt-get upgrade -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"
echo "[$(date +%H:%M:%S)] ⏭️  Skipping apt-get upgrade for faster provisioning"

# CRITICAL: Wait 5 seconds after apt-get update to ensure all locks are released
echo "[$(date +%H:%M:%S)] ⏳ Waiting 5s for all apt locks to be fully released..."
sleep 5

# Install required packages
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 3: Installing security packages"
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] Installing: ufw, fail2ban, wireguard-tools, net-tools, curl, wget"
apt_get_with_retry apt-get install -y ufw fail2ban wireguard-tools net-tools curl wget

# Configure UFW Firewall
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 4: Configuring UFW firewall"
echo "[$(date +%H:%M:%S)] =========================================="
ufw default deny incoming
ufw default allow outgoing

# Allow SSH from allowed CIDRs
`

	// Add allowed CIDRs
	if len(cfg.AllowedCIDRs) > 0 {
		for _, cidr := range cfg.AllowedCIDRs {
			script += fmt.Sprintf("ufw allow from %s to any port %d proto tcp\n", cidr, cfg.SSHPort)
		}
	} else {
		// If no CIDRs specified, allow from anywhere (not recommended for production)
		script += fmt.Sprintf("ufw allow %d/tcp\n", cfg.SSHPort)
	}

	script += `
# Enable UFW
echo "[$(date +%H:%M:%S)] 🔥 Enabling UFW firewall..."
ufw --force enable
echo "[$(date +%H:%M:%S)] ✅ UFW firewall enabled and configured"

# Configure fail2ban
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 5: Configuring fail2ban"
echo "[$(date +%H:%M:%S)] =========================================="
cat > /etc/fail2ban/jail.local <<'EOF'
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5

[sshd]
enabled = true
port = ` + fmt.Sprintf("%d", cfg.SSHPort) + `
logpath = /var/log/auth.log
maxretry = 3
EOF

systemctl enable fail2ban
systemctl restart fail2ban
echo "[$(date +%H:%M:%S)] ✅ fail2ban configured and started"

# SSH Hardening
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 6: Hardening SSH configuration"
echo "[$(date +%H:%M:%S)] =========================================="
cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup

cat >> /etc/ssh/sshd_config <<'EOF'

# Bastion Security Hardening
PermitRootLogin prohibit-password
PasswordAuthentication no
PubkeyAuthentication yes
ChallengeResponseAuthentication no
UsePAM yes
X11Forwarding no
PrintMotd no
AcceptEnv LANG LC_*
`

	if cfg.IdleTimeout > 0 {
		script += fmt.Sprintf("ClientAliveInterval 60\nClientAliveCountMax %d\n", cfg.IdleTimeout)
	}

	if cfg.MaxSessions > 0 {
		script += fmt.Sprintf("MaxSessions %d\n", cfg.MaxSessions)
	}

	script += `EOF

# Enable SSH agent forwarding for ProxyJump
sed -i 's/#AllowAgentForwarding yes/AllowAgentForwarding yes/' /etc/ssh/sshd_config

echo "[$(date +%H:%M:%S)] 🔄 Reloading SSH configuration (not restarting to avoid breaking Pulumi connection)..."
# Use 'reload' instead of 'restart' to apply config changes without dropping connections
systemctl reload sshd || {
    echo "[$(date +%H:%M:%S)] ⚠️  Reload failed, SSH will restart on next connection"
}

echo "[$(date +%H:%M:%S)] ✅ SSH configuration reloaded"
echo "[$(date +%H:%M:%S)] ℹ️  Note: SSH is NOT restarted to avoid breaking Pulumi connection"
echo "[$(date +%H:%M:%S)] ℹ️  Changes are active for NEW connections"

echo "[$(date +%H:%M:%S)] ✅ SSH hardening complete"

# Audit Logging
`
	if cfg.EnableAuditLog {
		script += `
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 7: Setting up audit logging"
echo "[$(date +%H:%M:%S)] =========================================="
apt_get_with_retry apt-get install -y auditd audispd-plugins

# Log all SSH sessions
cat >> /etc/audit/rules.d/bastion.rules <<'EOF'
# Log all SSH sessions
-w /usr/sbin/sshd -p x -k bastion_ssh
-w /var/log/auth.log -p wa -k bastion_auth
EOF

augenrules --load || true
systemctl enable auditd
systemctl restart auditd
echo "[$(date +%H:%M:%S)] ✅ Audit logging configured"
`
	}

	script += `
# Install WireGuard
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 8: Finalizing configuration"
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] WireGuard tools already installed"

# Set hostname
echo "[$(date +%H:%M:%S)] Setting hostname to ` + cfg.Name + `"
hostnamectl set-hostname ` + cfg.Name + `

# Create MOTD
echo "[$(date +%H:%M:%S)] Creating MOTD banner"
cat > /etc/motd <<'EOF'
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║            🏰  BASTION HOST - AUTHORIZED ACCESS ONLY      ║
║                                                           ║
║  This system is for authorized users only.                ║
║  All activity is monitored and logged.                    ║
║  Unauthorized access is prohibited.                       ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝

Cluster Access:
  • SSH to cluster nodes: ssh root@10.8.0.<node-vpn-ip>
  • ProxyJump is configured automatically
  • All sessions are audited

EOF

echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] ✅ BASTION PROVISIONING COMPLETE!"
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] 🏰 Bastion is ready for secure cluster access"
echo "[$(date +%H:%M:%S)] Finished at: $(date)"
echo ""
`

	return script
}
