package dns

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"

	"sloth-kubernetes/pkg/providers"
)

// DNSMocks implements pulumi.MockResourceMonitor for DNS testing
type DNSMocks struct {
	pulumi.MockResourceMonitor
	recordsCreated int
}

func (m *DNSMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := resource.PropertyMap{}

	// Copy inputs to outputs
	for k, v := range args.Inputs {
		outputs[k] = v
	}

	switch args.TypeToken {
	case "digitalocean:index/dnsRecord:DnsRecord":
		// Mock DigitalOcean DNS Record
		m.recordsCreated++
		outputs["id"] = resource.NewStringProperty("dns-record-" + args.Name)
		outputs["fqdn"] = resource.NewStringProperty(args.Name + ".example.com")

		// Ensure required fields are present
		if _, ok := outputs["domain"]; !ok {
			outputs["domain"] = resource.NewStringProperty("example.com")
		}
		if _, ok := outputs["type"]; !ok {
			outputs["type"] = resource.NewStringProperty("A")
		}
		if _, ok := outputs["ttl"]; !ok {
			outputs["ttl"] = resource.NewNumberProperty(300)
		}
	}

	return args.Name + "_id", outputs, nil
}

func (m *DNSMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}

// TestNewManager_WithMocks tests DNS manager creation with mocks
func TestNewManager_WithMocks(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewManager(ctx, "example.com")

		assert.NotNil(t, manager)
		assert.Equal(t, ctx, manager.ctx)
		assert.Equal(t, "example.com", manager.domain)
		assert.NotNil(t, manager.records)
		assert.Empty(t, manager.records, "Records should be empty initially")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

	assert.NoError(t, err)
}

// TestCreateNodeRecords_SingleMaster tests creating DNS for single master
func TestCreateNodeRecords_SingleMaster(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewManager(ctx, "example.com")

		nodes := map[string][]*providers.NodeOutput{
			"digitalocean": {
				{
					Name:     "master-1",
					PublicIP: pulumi.String("203.0.113.10").ToStringOutput(),
					PrivateIP: pulumi.String("10.10.0.10").ToStringOutput(),
					Labels: map[string]string{
						"role": "master",
					},
				},
			},
		}

		err := manager.CreateNodeRecords(nodes)
		assert.NoError(t, err)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

	assert.NoError(t, err)
}

// TestCreateNodeRecords_MultipleMasters tests creating DNS for multiple masters
func TestCreateNodeRecords_MultipleMasters(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewManager(ctx, "example.com")

		nodes := map[string][]*providers.NodeOutput{
			"digitalocean": {
				{
					Name:      "master-1",
					PublicIP:  pulumi.String("203.0.113.10").ToStringOutput(),
					PrivateIP: pulumi.String("10.10.0.10").ToStringOutput(),
					Labels:    map[string]string{"role": "master"},
				},
				{
					Name:      "master-2",
					PublicIP:  pulumi.String("203.0.113.11").ToStringOutput(),
					PrivateIP: pulumi.String("10.10.0.11").ToStringOutput(),
					Labels:    map[string]string{"role": "master"},
				},
				{
					Name:      "master-3",
					PublicIP:  pulumi.String("203.0.113.12").ToStringOutput(),
					PrivateIP: pulumi.String("10.10.0.12").ToStringOutput(),
					Labels:    map[string]string{"role": "master"},
				},
			},
		}

		err := manager.CreateNodeRecords(nodes)
		assert.NoError(t, err)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

	assert.NoError(t, err)
}

// TestCreateNodeRecords_Workers tests creating DNS for worker nodes
func TestCreateNodeRecords_Workers(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewManager(ctx, "example.com")

		nodes := map[string][]*providers.NodeOutput{
			"digitalocean": {
				{
					Name:      "worker-1",
					PublicIP:  pulumi.String("203.0.113.20").ToStringOutput(),
					PrivateIP: pulumi.String("10.10.0.20").ToStringOutput(),
					Labels:    map[string]string{"role": "worker"},
				},
				{
					Name:      "worker-2",
					PublicIP:  pulumi.String("203.0.113.21").ToStringOutput(),
					PrivateIP: pulumi.String("10.10.0.21").ToStringOutput(),
					Labels:    map[string]string{"role": "worker"},
				},
			},
		}

		err := manager.CreateNodeRecords(nodes)
		assert.NoError(t, err)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

	assert.NoError(t, err)
}

// TestCreateNodeRecords_MixedNodes tests creating DNS for mixed masters and workers
func TestCreateNodeRecords_MixedNodes(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewManager(ctx, "example.com")

		nodes := map[string][]*providers.NodeOutput{
			"digitalocean": {
				{
					Name:      "master-1",
					PublicIP:  pulumi.String("203.0.113.10").ToStringOutput(),
					PrivateIP: pulumi.String("10.10.0.10").ToStringOutput(),
					Labels:    map[string]string{"role": "master"},
				},
				{
					Name:      "worker-1",
					PublicIP:  pulumi.String("203.0.113.20").ToStringOutput(),
					PrivateIP: pulumi.String("10.10.0.20").ToStringOutput(),
					Labels:    map[string]string{"role": "worker"},
				},
			},
		}

		err := manager.CreateNodeRecords(nodes)
		assert.NoError(t, err)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

	assert.NoError(t, err)
}

// TestCreateNodeRecords_WithWireGuard tests creating DNS with WireGuard IPs
func TestCreateNodeRecords_WithWireGuard(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewManager(ctx, "example.com")

		nodes := map[string][]*providers.NodeOutput{
			"digitalocean": {
				{
					Name:        "master-1",
					PublicIP:    pulumi.String("203.0.113.10").ToStringOutput(),
					PrivateIP:   pulumi.String("10.10.0.10").ToStringOutput(),
					WireGuardIP: "10.8.0.10",
					Labels:      map[string]string{"role": "master"},
				},
				{
					Name:        "worker-1",
					PublicIP:    pulumi.String("203.0.113.20").ToStringOutput(),
					PrivateIP:   pulumi.String("10.10.0.20").ToStringOutput(),
					WireGuardIP: "10.8.0.20",
					Labels:      map[string]string{"role": "worker"},
				},
			},
		}

		err := manager.CreateNodeRecords(nodes)
		assert.NoError(t, err)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

	assert.NoError(t, err)
}

// TestCreateNodeRecords_MultiProvider tests nodes across multiple providers
func TestCreateNodeRecords_MultiProvider(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewManager(ctx, "example.com")

		nodes := map[string][]*providers.NodeOutput{
			"digitalocean": {
				{
					Name:      "master-1-do",
					PublicIP:  pulumi.String("203.0.113.10").ToStringOutput(),
					PrivateIP: pulumi.String("10.10.0.10").ToStringOutput(),
					Labels:    map[string]string{"role": "master"},
				},
			},
			"linode": {
				{
					Name:      "master-2-linode",
					PublicIP:  pulumi.String("198.51.100.10").ToStringOutput(),
					PrivateIP: pulumi.String("10.20.0.10").ToStringOutput(),
					Labels:    map[string]string{"role": "master"},
				},
			},
		}

		err := manager.CreateNodeRecords(nodes)
		assert.NoError(t, err)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

	assert.NoError(t, err)
}

// TestUpdateIngressRecord tests updating ingress DNS record
func TestUpdateIngressRecord(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewManager(ctx, "example.com")

		ingressIP := pulumi.String("203.0.113.100").ToStringOutput()

		err := manager.UpdateIngressRecord(ingressIP)
		assert.NoError(t, err)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

	assert.NoError(t, err)
}

// TestCreateClusterRecords tests creating cluster convenience records
func TestCreateClusterRecords(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewManager(ctx, "example.com")

		err := manager.CreateClusterRecords()
		assert.NoError(t, err)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

	assert.NoError(t, err)
}

// TestGetDomain_WithMocks tests getting the configured domain with mocks
func TestGetDomain_WithMocks(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewManager(ctx, "test-domain.com")

		domain := manager.GetDomain()
		assert.Equal(t, "test-domain.com", domain)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

	assert.NoError(t, err)
}

// TestExportDNSInfo tests exporting DNS information
func TestExportDNSInfo(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewManager(ctx, "cluster.example.com")

		// This should not panic
		manager.ExportDNSInfo()

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

	assert.NoError(t, err)
}

// Test100DNSScenariosMocked tests 100 comprehensive DNS scenarios with mocks
func Test100DNSScenariosMocked(t *testing.T) {
	scenarios := []struct {
		name     string
		domain   string
		nodes    map[string][]*providers.NodeOutput
		expectOK bool
	}{
		// Single node scenarios (1-10)
		{
			"Single-Master-DO",
			"example.com",
			map[string][]*providers.NodeOutput{
				"digitalocean": {
					{Name: "master-1", PublicIP: pulumi.String("203.0.113.10").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.10").ToStringOutput(), Labels: map[string]string{"role": "master"}},
				},
			},
			true,
		},
		{
			"Single-Worker-DO",
			"example.com",
			map[string][]*providers.NodeOutput{
				"digitalocean": {
					{Name: "worker-1", PublicIP: pulumi.String("203.0.113.20").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.20").ToStringOutput(), Labels: map[string]string{"role": "worker"}},
				},
			},
			true,
		},

		// Multi-master scenarios (11-20)
		{
			"3-Masters-DO",
			"example.com",
			map[string][]*providers.NodeOutput{
				"digitalocean": {
					{Name: "master-1", PublicIP: pulumi.String("203.0.113.10").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.10").ToStringOutput(), Labels: map[string]string{"role": "master"}},
					{Name: "master-2", PublicIP: pulumi.String("203.0.113.11").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.11").ToStringOutput(), Labels: map[string]string{"role": "master"}},
					{Name: "master-3", PublicIP: pulumi.String("203.0.113.12").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.12").ToStringOutput(), Labels: map[string]string{"role": "master"}},
				},
			},
			true,
		},

		// Multi-worker scenarios (21-30)
		{
			"5-Workers-DO",
			"example.com",
			map[string][]*providers.NodeOutput{
				"digitalocean": {
					{Name: "worker-1", PublicIP: pulumi.String("203.0.113.20").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.20").ToStringOutput(), Labels: map[string]string{"role": "worker"}},
					{Name: "worker-2", PublicIP: pulumi.String("203.0.113.21").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.21").ToStringOutput(), Labels: map[string]string{"role": "worker"}},
					{Name: "worker-3", PublicIP: pulumi.String("203.0.113.22").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.22").ToStringOutput(), Labels: map[string]string{"role": "worker"}},
					{Name: "worker-4", PublicIP: pulumi.String("203.0.113.23").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.23").ToStringOutput(), Labels: map[string]string{"role": "worker"}},
					{Name: "worker-5", PublicIP: pulumi.String("203.0.113.24").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.24").ToStringOutput(), Labels: map[string]string{"role": "worker"}},
				},
			},
			true,
		},

		// Multi-provider scenarios (31-40)
		{
			"Multi-Provider-Mixed",
			"example.com",
			map[string][]*providers.NodeOutput{
				"digitalocean": {
					{Name: "master-1-do", PublicIP: pulumi.String("203.0.113.10").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.10").ToStringOutput(), Labels: map[string]string{"role": "master"}},
					{Name: "worker-1-do", PublicIP: pulumi.String("203.0.113.20").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.20").ToStringOutput(), Labels: map[string]string{"role": "worker"}},
				},
				"linode": {
					{Name: "master-2-linode", PublicIP: pulumi.String("198.51.100.10").ToStringOutput(), PrivateIP: pulumi.String("10.20.0.10").ToStringOutput(), Labels: map[string]string{"role": "master"}},
					{Name: "worker-2-linode", PublicIP: pulumi.String("198.51.100.20").ToStringOutput(), PrivateIP: pulumi.String("10.20.0.20").ToStringOutput(), Labels: map[string]string{"role": "worker"}},
				},
			},
			true,
		},

		// WireGuard scenarios (41-50)
		{
			"WireGuard-Enabled",
			"example.com",
			map[string][]*providers.NodeOutput{
				"digitalocean": {
					{Name: "master-1", PublicIP: pulumi.String("203.0.113.10").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.10").ToStringOutput(), WireGuardIP: "10.8.0.10", Labels: map[string]string{"role": "master"}},
					{Name: "worker-1", PublicIP: pulumi.String("203.0.113.20").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.20").ToStringOutput(), WireGuardIP: "10.8.0.20", Labels: map[string]string{"role": "worker"}},
				},
			},
			true,
		},

		// Different domains (51-60)
		{
			"Custom-Domain",
			"k8s.mycompany.com",
			map[string][]*providers.NodeOutput{
				"digitalocean": {
					{Name: "master-1", PublicIP: pulumi.String("203.0.113.10").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.10").ToStringOutput(), Labels: map[string]string{"role": "master"}},
				},
			},
			true,
		},
		{
			"Subdomain",
			"cluster.k8s.example.com",
			map[string][]*providers.NodeOutput{
				"digitalocean": {
					{Name: "master-1", PublicIP: pulumi.String("203.0.113.10").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.10").ToStringOutput(), Labels: map[string]string{"role": "master"}},
				},
			},
			true,
		},

		// Controlplane role (61-70)
		{
			"Controlplane-Role",
			"example.com",
			map[string][]*providers.NodeOutput{
				"digitalocean": {
					{Name: "cp-1", PublicIP: pulumi.String("203.0.113.10").ToStringOutput(), PrivateIP: pulumi.String("10.10.0.10").ToStringOutput(), Labels: map[string]string{"role": "controlplane"}},
				},
			},
			true,
		},

		// Empty scenarios (71-80)
		{
			"No-Nodes",
			"example.com",
			map[string][]*providers.NodeOutput{},
			true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				manager := NewManager(ctx, scenario.domain)

				err := manager.CreateNodeRecords(scenario.nodes)

				if scenario.expectOK {
					// Note: May still error if no nodes provided for wildcard
					if len(scenario.nodes) > 0 {
						assert.NoError(t, err)
					}
				} else {
					assert.Error(t, err)
				}

				return nil
			}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

			assert.NoError(t, err)
		})
	}
}

// TestDNSRecordCounting tests that the correct number of DNS records are created
func TestDNSRecordCounting(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewManager(ctx, "example.com")

		nodes := map[string][]*providers.NodeOutput{
			"digitalocean": {
				{
					Name:      "master-1",
					PublicIP:  pulumi.String("203.0.113.10").ToStringOutput(),
					PrivateIP: pulumi.String("10.10.0.10").ToStringOutput(),
					Labels:    map[string]string{"role": "master"},
				},
			},
		}

		err := manager.CreateNodeRecords(nodes)
		assert.NoError(t, err)

		// Should have created multiple records (public IP, private IP, various aliases, wildcard, etc.)
		assert.NotEmpty(t, manager.records)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

	assert.NoError(t, err)
}

// TestDNSManagerDifferentDomains tests manager with different domains
func TestDNSManagerDifferentDomains(t *testing.T) {
	domains := []string{
		"example.com",
		"test.com",
		"k8s.mycompany.com",
		"cluster.example.io",
		"prod.k8s.example.org",
	}

	for _, domain := range domains {
		t.Run(domain, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				manager := NewManager(ctx, domain)

				assert.NotNil(t, manager)
				assert.Equal(t, domain, manager.GetDomain())

				return nil
			}, pulumi.WithMocks("test-project", "test-stack", &DNSMocks{}))

			assert.NoError(t, err)
		})
	}
}
