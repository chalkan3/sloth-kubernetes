package cloudinit

import "fmt"

// GenerateUserDataWithHostname generates cloud-init user data with hostname configuration
// K3s installation is handled by remote commands AFTER WireGuard is configured
func GenerateUserDataWithHostname(hostname string) string {
	return GenerateUserDataWithHostnameAndSalt(hostname, "")
}

// GenerateUserDataWithHostnameAndSalt generates cloud-init user data with hostname and Salt Minion
// If saltMasterIP is provided, Salt Minion will be installed and configured to connect to that master
func GenerateUserDataWithHostnameAndSalt(hostname string, saltMasterIP string) string {
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

	// Build write_files section for Salt config
	writeFiles := ""
	if saltMasterIP != "" {
		writeFiles = fmt.Sprintf(`
# Salt Minion configuration files
write_files:
  - path: /etc/salt/minion.d/master.conf
    content: |
      master: %s
    owner: root:root
    permissions: '0644'
  - path: /etc/salt/minion.d/minion_id.conf
    content: |
      id: %s
    owner: root:root
    permissions: '0644'
`, saltMasterIP, hostname)
	}

	// Build runcmd section
	runcmds := `# Enable IP forwarding for Kubernetes networking
runcmd:
  - sysctl -w net.ipv4.ip_forward=1
  - echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
  - sysctl -w net.ipv6.conf.all.forwarding=1
  - echo "net.ipv6.conf.all.forwarding=1" >> /etc/sysctl.conf`

	// Add Salt Minion installation if master IP is provided
	if saltMasterIP != "" {
		runcmds += `
  # Install Salt Minion
  - echo "Installing Salt Minion..."
  - curl -o /tmp/bootstrap-salt.sh -L https://github.com/saltstack/salt-bootstrap/releases/latest/download/bootstrap-salt.sh
  - chmod +x /tmp/bootstrap-salt.sh
  - sh /tmp/bootstrap-salt.sh stable
  # Restart Salt Minion to apply configuration
  - systemctl restart salt-minion
  - systemctl enable salt-minion
  - echo "Salt Minion installed and configured"`
	}

	cloudConfig := fmt.Sprintf(`#cloud-config
%s%s
# Package installation (runs during boot)
# Only install prerequisites - K3s will be installed later via remote commands
packages:
  - curl
  - wget
  - git
  - wireguard
  - wireguard-tools
  - net-tools

%s
`, hostnameConfig, writeFiles, runcmds)

	return cloudConfig
}
