package common

import "strings"

// Common remote commands used across the project

const (
	// InstallDocker installs and starts Docker
	InstallDocker = `
apt-get update && apt-get install -y docker.io
systemctl enable docker
systemctl start docker
`

	// InstallWireGuard installs WireGuard
	InstallWireGuard = `apt-get update && apt-get install -y wireguard wireguard-tools`

	// GenerateWireGuardKeys generates WireGuard keypair and outputs public key
	GenerateWireGuardKeys = `
wg genkey | tee /etc/wireguard/private.key | wg pubkey > /etc/wireguard/public.key
chmod 600 /etc/wireguard/private.key
cat /etc/wireguard/public.key
`

	// EnableIPForwarding enables IP forwarding for VPN
	EnableIPForwarding = `
echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf
sysctl -p
`

	// CheckDockerStatus checks if Docker is running
	CheckDockerStatus = `systemctl is-active docker`

	// CheckWireGuardStatus checks if WireGuard is active
	CheckWireGuardStatus = `wg show`
)

// BuildAptInstallCommand creates an apt-get install command with retry logic
// This handles the common issue of unattended-upgrades holding the lock
func BuildAptInstallCommand(packages ...string) string {
	packageList := strings.Join(packages, " ")

	return `
# Wait for apt locks to be released (handles unattended-upgrades)
while fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1 || fuser /var/lib/apt/lists/lock >/dev/null 2>&1; do
  echo "Waiting for apt locks to be released..."
  sleep 5
done

# Update and install packages
apt-get update -y
DEBIAN_FRONTEND=noninteractive apt-get install -y ` + packageList
}

// BuildSystemdEnableStart enables and starts a systemd service
func BuildSystemdEnableStart(serviceName string) string {
	return "systemctl enable " + serviceName + " && systemctl start " + serviceName
}

// BuildFileWrite creates a command to write content to a file
func BuildFileWrite(path string, content string) string {
	return "cat > " + path + " <<'EOF'\n" + content + "\nEOF"
}

// BuildDirectoryCreate creates a directory with proper permissions
func BuildDirectoryCreate(path string, mode string) string {
	return "mkdir -p " + path + " && chmod " + mode + " " + path
}
