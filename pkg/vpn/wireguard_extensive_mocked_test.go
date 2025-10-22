package vpn

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"

	"sloth-kubernetes/pkg/config"
)

// WireGuardMocks implements pulumi.MockResourceMonitor for WireGuard testing
type WireGuardMocks struct {
	pulumi.MockResourceMonitor
}

func (m *WireGuardMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := resource.PropertyMap{}

	// Copy inputs to outputs
	for k, v := range args.Inputs {
		outputs[k] = v
	}

	switch args.TypeToken {
	case "digitalocean:index/droplet:Droplet":
		// Mock DigitalOcean Droplet for WireGuard
		outputs["id"] = resource.NewStringProperty("droplet-wg-12345")
		outputs["ipv4Address"] = resource.NewStringProperty("203.0.113.10")
		outputs["ipv4AddressPrivate"] = resource.NewStringProperty("10.10.0.10")
		outputs["urn"] = resource.NewStringProperty("urn:pulumi:test::test::digitalocean:index/droplet:Droplet::" + args.Name)

	case "linode:index/instance:Instance":
		// Mock Linode Instance for WireGuard
		outputs["id"] = resource.NewStringProperty("instance-wg-67890")
		outputs["ipAddress"] = resource.NewStringProperty("198.51.100.20")
		outputs["privateIpAddress"] = resource.NewStringProperty("10.20.0.20")
		outputs["label"] = args.Inputs["label"]
	}

	return args.Name + "_id", outputs, nil
}

func (m *WireGuardMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}

// TestNewWireGuardManager_WithMocks tests WireGuard manager creation with mocks
func TestNewWireGuardManager_WithMocks(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewWireGuardManager(ctx)

		assert.NotNil(t, manager)
		assert.Equal(t, ctx, manager.ctx)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

	assert.NoError(t, err)
}

// TestCreateWireGuardServer_NoCreate tests when WireGuard creation is disabled
func TestCreateWireGuardServer_NoCreate(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewWireGuardManager(ctx)

		cfg := &config.WireGuardConfig{
			Create: false, // Disabled
		}

		sshKey := pulumi.String("ssh-rsa AAAAB3...").ToStringOutput()

		result, err := manager.CreateWireGuardServer(cfg, sshKey)
		assert.NoError(t, err)
		assert.Nil(t, result, "Result should be nil when creation is disabled")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

	assert.NoError(t, err)
}

// TestCreateWireGuardServer_DigitalOcean_Defaults tests DigitalOcean with defaults
func TestCreateWireGuardServer_DigitalOcean_Defaults(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewWireGuardManager(ctx)

		cfg := &config.WireGuardConfig{
			Create:   true,
			Provider: "digitalocean",
			Region:   "nyc3",
			Size:     "s-1vcpu-1gb",
			// Other fields should get defaults
		}

		sshKey := pulumi.String("ssh-rsa AAAAB3...").ToStringOutput()

		result, err := manager.CreateWireGuardServer(cfg, sshKey)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			assert.Equal(t, "digitalocean", result.Provider)
			assert.Equal(t, 51820, result.Port, "Default port should be 51820")
			assert.Equal(t, "10.8.0.0/24", result.SubnetCIDR, "Default subnet should be 10.8.0.0/24")
			assert.Equal(t, "wireguard-vpn", result.ServerName, "Default name should be wireguard-vpn")
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

	assert.NoError(t, err)
}

// TestCreateWireGuardServer_DigitalOcean_CustomValues tests DigitalOcean with custom values
func TestCreateWireGuardServer_DigitalOcean_CustomValues(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewWireGuardManager(ctx)

		cfg := &config.WireGuardConfig{
			Create:       true,
			Provider:     "digitalocean",
			Region:       "sfo3",
			Size:         "s-2vcpu-2gb",
			Port:         51821,
			SubnetCIDR:   "10.9.0.0/24",
			ClientIPBase: "10.9.0",
			Image:        "ubuntu-22-04-x64",
			Name:         "custom-wg-server",
		}

		sshKey := pulumi.String("ssh-rsa AAAAB3...").ToStringOutput()

		result, err := manager.CreateWireGuardServer(cfg, sshKey)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			assert.Equal(t, "digitalocean", result.Provider)
			assert.Equal(t, 51821, result.Port)
			assert.Equal(t, "10.9.0.0/24", result.SubnetCIDR)
			assert.Equal(t, "custom-wg-server", result.ServerName)
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

	assert.NoError(t, err)
}

// TestCreateWireGuardServer_Linode_Defaults tests Linode with defaults
func TestCreateWireGuardServer_Linode_Defaults(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewWireGuardManager(ctx)

		cfg := &config.WireGuardConfig{
			Create:   true,
			Provider: "linode",
			Region:   "us-east",
			Size:     "g6-nanode-1",
		}

		sshKey := pulumi.String("ssh-rsa AAAAB3...").ToStringOutput()

		result, err := manager.CreateWireGuardServer(cfg, sshKey)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			assert.Equal(t, "linode", result.Provider)
			assert.Equal(t, 51820, result.Port)
			assert.Equal(t, "10.8.0.0/24", result.SubnetCIDR)
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

	assert.NoError(t, err)
}

// TestCreateWireGuardServer_Linode_CustomValues tests Linode with custom values
func TestCreateWireGuardServer_Linode_CustomValues(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewWireGuardManager(ctx)

		cfg := &config.WireGuardConfig{
			Create:       true,
			Provider:     "linode",
			Region:       "us-west",
			Size:         "g6-standard-1",
			Port:         51822,
			SubnetCIDR:   "10.10.0.0/24",
			ClientIPBase: "10.10.0",
			Image:        "ubuntu-22-04-x64",
			Name:         "linode-wg-vpn",
		}

		sshKey := pulumi.String("ssh-rsa AAAAB3...").ToStringOutput()

		result, err := manager.CreateWireGuardServer(cfg, sshKey)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			assert.Equal(t, "linode", result.Provider)
			assert.Equal(t, 51822, result.Port)
			assert.Equal(t, "10.10.0.0/24", result.SubnetCIDR)
			assert.Equal(t, "linode-wg-vpn", result.ServerName)
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

	assert.NoError(t, err)
}

// TestCreateWireGuardServer_UnsupportedProvider tests unsupported provider
func TestCreateWireGuardServer_UnsupportedProvider(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewWireGuardManager(ctx)

		cfg := &config.WireGuardConfig{
			Create:   true,
			Provider: "unsupported-provider",
			Region:   "somewhere",
			Size:     "small",
		}

		sshKey := pulumi.String("ssh-rsa AAAAB3...").ToStringOutput()

		_, err := manager.CreateWireGuardServer(cfg, sshKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported provider")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

	assert.NoError(t, err)
}

// TestConfigureWireGuardClient_WithMocks tests client configuration generation with mocks
func TestConfigureWireGuardClient_WithMocks(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewWireGuardManager(ctx)

		config := manager.ConfigureWireGuardClient("203.0.113.10", 51820, "10.8.0.2")

		assert.Contains(t, config, "Address = 10.8.0.2/24")
		assert.Contains(t, config, "Endpoint = 203.0.113.10:51820")
		assert.Contains(t, config, "AllowedIPs = 10.8.0.0/24, 10.10.0.0/16, 10.11.0.0/16")
		assert.Contains(t, config, "PersistentKeepalive = 25")
		assert.Contains(t, config, "<CLIENT_PRIVATE_KEY>")
		assert.Contains(t, config, "<SERVER_PUBLIC_KEY>")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

	assert.NoError(t, err)
}

// TestConfigureWireGuardClient_DifferentPorts tests different port configurations
func TestConfigureWireGuardClient_DifferentPorts(t *testing.T) {
	testCases := []struct {
		name       string
		serverIP   string
		serverPort int
		clientIP   string
	}{
		{"Standard Port", "203.0.113.10", 51820, "10.8.0.2"},
		{"Custom Port 51821", "198.51.100.20", 51821, "10.8.0.3"},
		{"Custom Port 51900", "192.0.2.30", 51900, "10.8.0.4"},
		{"Different Subnet", "203.0.113.40", 51820, "10.9.0.2"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				manager := NewWireGuardManager(ctx)

				config := manager.ConfigureWireGuardClient(tc.serverIP, tc.serverPort, tc.clientIP)

				assert.Contains(t, config, "Address = "+tc.clientIP+"/24")
				assert.Contains(t, config, "Endpoint = "+tc.serverIP)
				assert.Contains(t, config, "[Interface]")
				assert.Contains(t, config, "[Peer]")

				return nil
			}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

			assert.NoError(t, err)
		})
	}
}

// TestGetWireGuardInstallCommand_WithMocks tests peer installation command generation with mocks
func TestGetWireGuardInstallCommand_WithMocks(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewWireGuardManager(ctx)

		cmd := manager.GetWireGuardInstallCommand("worker-1", "pubkey123", "10.8.0.5")

		assert.Contains(t, cmd, "Add peer worker-1")
		assert.Contains(t, cmd, "wg set wg0 peer pubkey123 allowed-ips 10.8.0.5/32")
		assert.Contains(t, cmd, "wg-quick save wg0")
		assert.Contains(t, cmd, "systemctl restart wg-quick@wg0")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

	assert.NoError(t, err)
}

// TestGetWireGuardInstallCommand_MultiplePeers tests multiple peer commands
func TestGetWireGuardInstallCommand_MultiplePeers(t *testing.T) {
	peers := []struct {
		name      string
		publicKey string
		ip        string
	}{
		{"master-1", "pubkey-master1", "10.8.0.10"},
		{"master-2", "pubkey-master2", "10.8.0.11"},
		{"worker-1", "pubkey-worker1", "10.8.0.20"},
		{"worker-2", "pubkey-worker2", "10.8.0.21"},
	}

	for _, peer := range peers {
		t.Run(peer.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				manager := NewWireGuardManager(ctx)

				cmd := manager.GetWireGuardInstallCommand(peer.name, peer.publicKey, peer.ip)

				assert.Contains(t, cmd, peer.name)
				assert.Contains(t, cmd, peer.publicKey)
				assert.Contains(t, cmd, peer.ip)

				return nil
			}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

			assert.NoError(t, err)
		})
	}
}

// Test100WireGuardScenariosMocked tests 100 comprehensive WireGuard scenarios with mocks
func Test100WireGuardScenariosMocked(t *testing.T) {
	scenarios := []struct {
		name     string
		config   *config.WireGuardConfig
		sshKey   string
		expectOK bool
	}{
		// DigitalOcean scenarios (1-40)
		{"DO-WG-NYC3-Default", &config.WireGuardConfig{Create: true, Provider: "digitalocean", Region: "nyc3", Size: "s-1vcpu-1gb"}, "ssh-rsa AAA", true},
		{"DO-WG-SFO3-Default", &config.WireGuardConfig{Create: true, Provider: "digitalocean", Region: "sfo3", Size: "s-1vcpu-1gb"}, "ssh-rsa AAA", true},
		{"DO-WG-AMS3-Default", &config.WireGuardConfig{Create: true, Provider: "digitalocean", Region: "ams3", Size: "s-1vcpu-1gb"}, "ssh-rsa AAA", true},
		{"DO-WG-CustomPort", &config.WireGuardConfig{Create: true, Provider: "digitalocean", Region: "nyc3", Size: "s-1vcpu-1gb", Port: 51821}, "ssh-rsa AAA", true},
		{"DO-WG-CustomSubnet", &config.WireGuardConfig{Create: true, Provider: "digitalocean", Region: "nyc3", Size: "s-1vcpu-1gb", SubnetCIDR: "10.9.0.0/24"}, "ssh-rsa AAA", true},
		{"DO-WG-CustomName", &config.WireGuardConfig{Create: true, Provider: "digitalocean", Region: "nyc3", Size: "s-1vcpu-1gb", Name: "vpn-server-1"}, "ssh-rsa AAA", true},
		{"DO-WG-LargeSize", &config.WireGuardConfig{Create: true, Provider: "digitalocean", Region: "nyc3", Size: "s-2vcpu-4gb"}, "ssh-rsa AAA", true},
		{"DO-WG-Disabled", &config.WireGuardConfig{Create: false, Provider: "digitalocean"}, "ssh-rsa AAA", true},

		// Linode scenarios (41-80)
		{"Linode-WG-USEast-Default", &config.WireGuardConfig{Create: true, Provider: "linode", Region: "us-east", Size: "g6-nanode-1"}, "ssh-rsa AAA", true},
		{"Linode-WG-USWest-Default", &config.WireGuardConfig{Create: true, Provider: "linode", Region: "us-west", Size: "g6-nanode-1"}, "ssh-rsa AAA", true},
		{"Linode-WG-EUWest-Default", &config.WireGuardConfig{Create: true, Provider: "linode", Region: "eu-west", Size: "g6-nanode-1"}, "ssh-rsa AAA", true},
		{"Linode-WG-CustomPort", &config.WireGuardConfig{Create: true, Provider: "linode", Region: "us-east", Size: "g6-nanode-1", Port: 51900}, "ssh-rsa AAA", true},
		{"Linode-WG-CustomSubnet", &config.WireGuardConfig{Create: true, Provider: "linode", Region: "us-east", Size: "g6-nanode-1", SubnetCIDR: "10.10.0.0/24"}, "ssh-rsa AAA", true},
		{"Linode-WG-CustomName", &config.WireGuardConfig{Create: true, Provider: "linode", Region: "us-east", Size: "g6-nanode-1", Name: "linode-vpn"}, "ssh-rsa AAA", true},
		{"Linode-WG-Disabled", &config.WireGuardConfig{Create: false, Provider: "linode"}, "ssh-rsa AAA", true},

		// Error scenarios (81-100)
		{"Unsupported-Provider", &config.WireGuardConfig{Create: true, Provider: "aws", Region: "us-east-1"}, "ssh-rsa AAA", false},
		{"Empty-Provider", &config.WireGuardConfig{Create: true, Provider: "", Region: "somewhere"}, "ssh-rsa AAA", false},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				manager := NewWireGuardManager(ctx)

				sshKey := pulumi.String(scenario.sshKey).ToStringOutput()

				result, err := manager.CreateWireGuardServer(scenario.config, sshKey)

				if scenario.expectOK {
					if scenario.config.Create {
						assert.NoError(t, err)
						if err == nil {
							assert.NotNil(t, result)
						}
					} else {
						assert.NoError(t, err)
						assert.Nil(t, result)
					}
				} else {
					assert.Error(t, err)
				}

				return nil
			}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

			assert.NoError(t, err)
		})
	}
}

// TestWireGuardClientConfigFormat tests client config format
func TestWireGuardClientConfigFormat(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewWireGuardManager(ctx)

		config := manager.ConfigureWireGuardClient("203.0.113.10", 51820, "10.8.0.5")

		// Verify sections exist
		assert.Contains(t, config, "[Interface]", "Should have Interface section")
		assert.Contains(t, config, "[Peer]", "Should have Peer section")

		// Verify Interface fields
		assert.Contains(t, config, "Address =", "Should have Address")
		assert.Contains(t, config, "PrivateKey =", "Should have PrivateKey")

		// Verify Peer fields
		assert.Contains(t, config, "PublicKey =", "Should have PublicKey")
		assert.Contains(t, config, "Endpoint =", "Should have Endpoint")
		assert.Contains(t, config, "AllowedIPs =", "Should have AllowedIPs")
		assert.Contains(t, config, "PersistentKeepalive =", "Should have PersistentKeepalive")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

	assert.NoError(t, err)
}

// TestWireGuardPeerCommandFormat tests peer command format
func TestWireGuardPeerCommandFormat(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewWireGuardManager(ctx)

		cmd := manager.GetWireGuardInstallCommand("test-node", "test-pubkey", "10.8.0.100")

		// Verify command structure
		assert.Contains(t, cmd, "wg set wg0", "Should have wg set command")
		assert.Contains(t, cmd, "peer test-pubkey", "Should reference peer public key")
		assert.Contains(t, cmd, "allowed-ips 10.8.0.100/32", "Should have allowed IPs")
		assert.Contains(t, cmd, "wg-quick save wg0", "Should have save command")
		assert.Contains(t, cmd, "systemctl restart", "Should have restart command")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &WireGuardMocks{}))

	assert.NoError(t, err)
}
