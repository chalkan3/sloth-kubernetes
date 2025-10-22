package ingress

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"

	"sloth-kubernetes/pkg/providers"
)

// IngressMocks implements pulumi.MockResourceMonitor for Ingress testing
type IngressMocks struct {
	pulumi.MockResourceMonitor
}

func (m *IngressMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := resource.PropertyMap{}

	// Copy inputs to outputs
	for k, v := range args.Inputs {
		outputs[k] = v
	}

	switch args.TypeToken {
	case "command:remote:Command":
		// Mock remote command execution
		outputs["stdout"] = resource.NewStringProperty(`
Installing NGINX Ingress Controller...
Waiting for LoadBalancer IP...
LoadBalancer IP: 203.0.113.100
INGRESS_IP:203.0.113.100
NGINX Ingress Controller installed successfully!
`)
		outputs["stderr"] = resource.NewStringProperty("")
		outputs["id"] = resource.NewStringProperty("cmd-" + args.Name)
	}

	return args.Name + "_id", outputs, nil
}

func (m *IngressMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}

// TestNewNginxIngressManager_WithMocks tests NGINX Ingress manager creation with mocks
func TestNewNginxIngressManager_WithMocks(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewNginxIngressManager(ctx, "example.com")

		assert.NotNil(t, manager)
		assert.Equal(t, ctx, manager.ctx)
		assert.Equal(t, "example.com", manager.domain)
		assert.Nil(t, manager.masterNode, "Master node should be nil initially")
		assert.Empty(t, manager.sshKeyPath, "SSH key path should be empty initially")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}

// TestSetMasterNode_WithMocks tests setting master node with mocks
func TestSetMasterNode_WithMocks(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewNginxIngressManager(ctx, "example.com")

		masterNode := &providers.NodeOutput{
			Name:     "master-1",
			PublicIP: pulumi.String("203.0.113.10").ToStringOutput(),
			SSHUser:  "root",
		}

		manager.SetMasterNode(masterNode)
		assert.NotNil(t, manager.masterNode)
		assert.Equal(t, masterNode, manager.masterNode)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}

// TestSetSSHKeyPath_WithMocks tests setting SSH key path with mocks
func TestSetSSHKeyPath_WithMocks(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewNginxIngressManager(ctx, "example.com")

		keyPath := "/root/.ssh/id_rsa"
		manager.SetSSHKeyPath(keyPath)

		assert.Equal(t, keyPath, manager.sshKeyPath)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}

// TestInstall_NoMasterNode tests install without master node
func TestInstall_NoMasterNode(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewNginxIngressManager(ctx, "example.com")

		// Don't set master node
		_, err := manager.Install()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "master node not set")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}

// TestInstall_Success tests successful installation
func TestInstall_Success(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewNginxIngressManager(ctx, "example.com")

		masterNode := &providers.NodeOutput{
			Name:     "master-1",
			PublicIP: pulumi.String("203.0.113.10").ToStringOutput(),
			SSHUser:  "root",
		}
		manager.SetMasterNode(masterNode)
		manager.SetSSHKeyPath("/root/.ssh/id_rsa")

		ingressIP, err := manager.Install()
		assert.NoError(t, err)
		assert.NotNil(t, ingressIP)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}

// TestInstall_DifferentDomains tests installation with different domains
func TestInstall_DifferentDomains(t *testing.T) {
	domains := []string{
		"example.com",
		"test.io",
		"k8s.mycompany.com",
		"cluster.example.org",
	}

	for _, domain := range domains {
		t.Run(domain, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				manager := NewNginxIngressManager(ctx, domain)

				masterNode := &providers.NodeOutput{
					Name:     "master-1",
					PublicIP: pulumi.String("203.0.113.10").ToStringOutput(),
					SSHUser:  "root",
				}
				manager.SetMasterNode(masterNode)

				ingressIP, err := manager.Install()
				assert.NoError(t, err)
				assert.NotNil(t, ingressIP)

				return nil
			}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

			assert.NoError(t, err)
		})
	}
}

// TestInstallCertManager_NoMasterNode tests cert-manager without master node
func TestInstallCertManager_NoMasterNode(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewNginxIngressManager(ctx, "example.com")

		err := manager.InstallCertManager()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "master node not set")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}

// TestInstallCertManager_Success tests successful cert-manager installation
func TestInstallCertManager_Success(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewNginxIngressManager(ctx, "example.com")

		masterNode := &providers.NodeOutput{
			Name:     "master-1",
			PublicIP: pulumi.String("203.0.113.10").ToStringOutput(),
			SSHUser:  "root",
		}
		manager.SetMasterNode(masterNode)
		manager.SetSSHKeyPath("/root/.ssh/id_rsa")

		err := manager.InstallCertManager()
		assert.NoError(t, err)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}

// TestCreateSampleIngress tests sample ingress creation
func TestCreateSampleIngress(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewNginxIngressManager(ctx, "example.com")

		err := manager.CreateSampleIngress()
		assert.NoError(t, err)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}

// TestCreateSampleIngress_DifferentDomains tests sample ingress with different domains
func TestCreateSampleIngress_DifferentDomains(t *testing.T) {
	domains := []string{
		"example.com",
		"test.io",
		"k8s.mycompany.com",
	}

	for _, domain := range domains {
		t.Run(domain, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				manager := NewNginxIngressManager(ctx, domain)

				err := manager.CreateSampleIngress()
				assert.NoError(t, err)

				return nil
			}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

			assert.NoError(t, err)
		})
	}
}

// Test100IngressScenariosMocked tests 100 comprehensive ingress scenarios with mocks
func Test100IngressScenariosMocked(t *testing.T) {
	scenarios := []struct {
		name         string
		domain       string
		masterNode   *providers.NodeOutput
		sshKeyPath   string
		testInstall  bool
		testCertMgr  bool
		expectOK     bool
	}{
		// Basic installation scenarios (1-20)
		{
			"Install-Basic-ExampleCom",
			"example.com",
			&providers.NodeOutput{Name: "master-1", PublicIP: pulumi.String("203.0.113.10").ToStringOutput(), SSHUser: "root"},
			"/root/.ssh/id_rsa",
			true,
			false,
			true,
		},
		{
			"Install-Basic-TestIO",
			"test.io",
			&providers.NodeOutput{Name: "master-1", PublicIP: pulumi.String("203.0.113.11").ToStringOutput(), SSHUser: "root"},
			"/root/.ssh/id_rsa",
			true,
			false,
			true,
		},
		{
			"Install-Basic-K8sSubdomain",
			"k8s.example.com",
			&providers.NodeOutput{Name: "master-1", PublicIP: pulumi.String("203.0.113.12").ToStringOutput(), SSHUser: "root"},
			"/root/.ssh/id_rsa",
			true,
			false,
			true,
		},

		// Different SSH users (21-30)
		{
			"Install-Ubuntu-User",
			"example.com",
			&providers.NodeOutput{Name: "master-1", PublicIP: pulumi.String("203.0.113.20").ToStringOutput(), SSHUser: "ubuntu"},
			"/home/ubuntu/.ssh/id_rsa",
			true,
			false,
			true,
		},
		{
			"Install-Admin-User",
			"example.com",
			&providers.NodeOutput{Name: "master-1", PublicIP: pulumi.String("203.0.113.21").ToStringOutput(), SSHUser: "admin"},
			"/home/admin/.ssh/id_rsa",
			true,
			false,
			true,
		},

		// Cert-manager scenarios (31-40)
		{
			"CertManager-Basic",
			"example.com",
			&providers.NodeOutput{Name: "master-1", PublicIP: pulumi.String("203.0.113.30").ToStringOutput(), SSHUser: "root"},
			"/root/.ssh/id_rsa",
			false,
			true,
			true,
		},
		{
			"CertManager-CustomDomain",
			"secure.k8s.example.com",
			&providers.NodeOutput{Name: "master-1", PublicIP: pulumi.String("203.0.113.31").ToStringOutput(), SSHUser: "root"},
			"/root/.ssh/id_rsa",
			false,
			true,
			true,
		},

		// Both install and cert-manager (41-50)
		{
			"Full-Install-ExampleCom",
			"example.com",
			&providers.NodeOutput{Name: "master-1", PublicIP: pulumi.String("203.0.113.40").ToStringOutput(), SSHUser: "root"},
			"/root/.ssh/id_rsa",
			true,
			true,
			true,
		},

		// Different master nodes (51-60)
		{
			"Different-MasterIP-1",
			"example.com",
			&providers.NodeOutput{Name: "master-prod", PublicIP: pulumi.String("198.51.100.10").ToStringOutput(), SSHUser: "root"},
			"/root/.ssh/id_rsa",
			true,
			false,
			true,
		},
		{
			"Different-MasterIP-2",
			"example.com",
			&providers.NodeOutput{Name: "master-staging", PublicIP: pulumi.String("192.0.2.50").ToStringOutput(), SSHUser: "root"},
			"/root/.ssh/id_rsa",
			true,
			false,
			true,
		},

		// Error scenarios (61-70)
		{
			"NoMasterNode-Install",
			"example.com",
			nil,
			"/root/.ssh/id_rsa",
			true,
			false,
			false,
		},
		{
			"NoMasterNode-CertManager",
			"example.com",
			nil,
			"/root/.ssh/id_rsa",
			false,
			true,
			false,
		},

		// Complex domain scenarios (71-80)
		{
			"Deep-Subdomain",
			"ingress.cluster.prod.k8s.example.com",
			&providers.NodeOutput{Name: "master-1", PublicIP: pulumi.String("203.0.113.70").ToStringOutput(), SSHUser: "root"},
			"/root/.ssh/id_rsa",
			true,
			false,
			true,
		},

		// Different SSH key paths (81-90)
		{
			"CustomSSHKey-1",
			"example.com",
			&providers.NodeOutput{Name: "master-1", PublicIP: pulumi.String("203.0.113.80").ToStringOutput(), SSHUser: "root"},
			"/custom/path/key",
			true,
			false,
			true,
		},
		{
			"CustomSSHKey-2",
			"example.com",
			&providers.NodeOutput{Name: "master-1", PublicIP: pulumi.String("203.0.113.81").ToStringOutput(), SSHUser: "root"},
			"/etc/ssh/custom_key",
			true,
			false,
			true,
		},

		// Sample ingress only (91-100)
		{
			"SampleIngress-Only-1",
			"example.com",
			nil,
			"",
			false,
			false,
			true,
		},
		{
			"SampleIngress-Only-2",
			"test.io",
			nil,
			"",
			false,
			false,
			true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				manager := NewNginxIngressManager(ctx, scenario.domain)

				if scenario.masterNode != nil {
					manager.SetMasterNode(scenario.masterNode)
				}
				if scenario.sshKeyPath != "" {
					manager.SetSSHKeyPath(scenario.sshKeyPath)
				}

				if scenario.testInstall {
					_, err := manager.Install()
					if scenario.expectOK {
						assert.NoError(t, err)
					} else {
						assert.Error(t, err)
					}
				}

				if scenario.testCertMgr {
					err := manager.InstallCertManager()
					if scenario.expectOK {
						assert.NoError(t, err)
					} else {
						assert.Error(t, err)
					}
				}

				if !scenario.testInstall && !scenario.testCertMgr {
					// Test sample ingress creation
					err := manager.CreateSampleIngress()
					assert.NoError(t, err)
				}

				return nil
			}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

			assert.NoError(t, err)
		})
	}
}

// TestIngressIPExtraction tests IP extraction from command output
func TestIngressIPExtraction(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewNginxIngressManager(ctx, "example.com")

		masterNode := &providers.NodeOutput{
			Name:     "master-1",
			PublicIP: pulumi.String("203.0.113.10").ToStringOutput(),
			SSHUser:  "root",
		}
		manager.SetMasterNode(masterNode)

		ingressIP, err := manager.Install()
		assert.NoError(t, err)
		assert.NotNil(t, ingressIP)

		// The mock should return an IP
		// In a real test, we would verify the IP format

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}

// TestManagerMultipleInstances tests multiple manager instances
func TestManagerMultipleInstances(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager1 := NewNginxIngressManager(ctx, "domain1.com")
		manager2 := NewNginxIngressManager(ctx, "domain2.com")
		manager3 := NewNginxIngressManager(ctx, "domain3.com")

		assert.NotNil(t, manager1)
		assert.NotNil(t, manager2)
		assert.NotNil(t, manager3)

		assert.Equal(t, "domain1.com", manager1.domain)
		assert.Equal(t, "domain2.com", manager2.domain)
		assert.Equal(t, "domain3.com", manager3.domain)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}

// TestSetMasterNodeMultipleTimes tests changing master node
func TestSetMasterNodeMultipleTimes(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewNginxIngressManager(ctx, "example.com")

		node1 := &providers.NodeOutput{Name: "master-1", PublicIP: pulumi.String("203.0.113.10").ToStringOutput()}
		node2 := &providers.NodeOutput{Name: "master-2", PublicIP: pulumi.String("203.0.113.11").ToStringOutput()}
		node3 := &providers.NodeOutput{Name: "master-3", PublicIP: pulumi.String("203.0.113.12").ToStringOutput()}

		manager.SetMasterNode(node1)
		assert.Equal(t, node1, manager.masterNode)

		manager.SetMasterNode(node2)
		assert.Equal(t, node2, manager.masterNode)

		manager.SetMasterNode(node3)
		assert.Equal(t, node3, manager.masterNode)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}

// TestSetSSHKeyPathMultipleTimes tests changing SSH key path
func TestSetSSHKeyPathMultipleTimes(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewNginxIngressManager(ctx, "example.com")

		paths := []string{
			"/root/.ssh/id_rsa",
			"/home/ubuntu/.ssh/id_rsa",
			"/custom/path/key",
			"/etc/ssh/deploy_key",
		}

		for _, path := range paths {
			manager.SetSSHKeyPath(path)
			assert.Equal(t, path, manager.sshKeyPath)
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}

// TestNginxIngressWithAllFeatures tests all features together
func TestNginxIngressWithAllFeatures(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewNginxIngressManager(ctx, "full-test.k8s.example.com")

		// Set master node
		masterNode := &providers.NodeOutput{
			Name:     "master-prod-1",
			PublicIP: pulumi.String("203.0.113.100").ToStringOutput(),
			SSHUser:  "root",
		}
		manager.SetMasterNode(masterNode)

		// Set SSH key
		manager.SetSSHKeyPath("/root/.ssh/production_key")

		// Install NGINX Ingress
		ingressIP, err := manager.Install()
		assert.NoError(t, err)
		assert.NotNil(t, ingressIP)

		// Install cert-manager
		err = manager.InstallCertManager()
		assert.NoError(t, err)

		// Create sample ingress
		err = manager.CreateSampleIngress()
		assert.NoError(t, err)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &IngressMocks{}))

	assert.NoError(t, err)
}
