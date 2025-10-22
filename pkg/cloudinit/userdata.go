package cloudinit

import "fmt"

// GenerateUserDataWithHostname generates cloud-init user data with hostname configuration
func GenerateUserDataWithHostname(k3sVersion string, hostname string) string {
	k3sInstallCmd := "curl -sfL https://get.k3s.io | sh -"
	if k3sVersion != "" {
		k3sInstallCmd = fmt.Sprintf("curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=%s sh -", k3sVersion)
	}

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
packages:
  - curl
  - wget
  - git

# Commands to run after packages are installed
runcmd:
  - %s
  - systemctl enable k3s || systemctl enable k3s-agent
  - systemctl start k3s || systemctl start k3s-agent
`, hostnameConfig, k3sInstallCmd)

	return cloudConfig
}
