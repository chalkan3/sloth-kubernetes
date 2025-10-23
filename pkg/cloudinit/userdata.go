package cloudinit

import "fmt"

// GenerateUserDataWithHostname generates cloud-init user data with hostname configuration
// K3s installation is handled by remote commands AFTER WireGuard is configured
func GenerateUserDataWithHostname(hostname string) string {
	// Add hostname configuration if provided
	hostnameConfig := ""
	if hostname != "" {
		hostnameConfig = fmt.Sprintf(`
# Set unique hostname for this node
hostname: %s
fqdn: %s.cluster.local
manage_etc_hosts: true
`, hostname, hostname)
	}

	cloudConfig := fmt.Sprintf(`#cloud-config
%s
# Package installation (runs during boot)
# Only install prerequisites - K3s will be installed later via remote commands
packages:
  - curl
  - wget
  - git
  - wireguard
  - wireguard-tools
  - net-tools

# Enable IP forwarding for Kubernetes networking
runcmd:
  - sysctl -w net.ipv4.ip_forward=1
  - echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
  - sysctl -w net.ipv6.conf.all.forwarding=1
  - echo "net.ipv6.conf.all.forwarding=1" >> /etc/sysctl.conf
`, hostnameConfig)

	return cloudConfig
}
