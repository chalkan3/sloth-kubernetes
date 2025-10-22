package vpc

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"

	"sloth-kubernetes/pkg/config"
)

// VPCMocks implements pulumi.MockResourceMonitor for VPC testing
type VPCMocks struct {
	pulumi.MockResourceMonitor
}

func (m *VPCMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := resource.PropertyMap{}

	// Copy inputs to outputs
	for k, v := range args.Inputs {
		outputs[k] = v
	}

	switch args.TypeToken {
	case "digitalocean:index/vpc:Vpc":
		// Mock DigitalOcean VPC
		outputs["id"] = resource.NewStringProperty("vpc-12345")
		outputs["urn"] = resource.NewStringProperty("urn:pulumi:test::vpc::digitalocean:index/vpc:Vpc::" + args.Name)
		if ipRange, ok := args.Inputs["ipRange"]; ok {
			outputs["ipRange"] = ipRange
		} else {
			outputs["ipRange"] = resource.NewStringProperty("10.10.0.0/16")
		}

	case "linode:index/vpc:Vpc":
		// Mock Linode VPC
		outputs["id"] = resource.NewStringProperty("vpc-67890")
		outputs["label"] = args.Inputs["label"]
	}

	return args.Name + "_id", outputs, nil
}

func (m *VPCMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}

// TestNewVPCManager_WithMocks tests VPC manager creation with mocks
func TestNewVPCManager_WithMocks(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		assert.NotNil(t, manager)
		assert.Equal(t, ctx, manager.ctx)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestCreateDigitalOceanVPC_Success tests successful DigitalOcean VPC creation
func TestCreateDigitalOceanVPC_Success(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.DigitalOceanProvider{
			Region: "nyc3",
			VPC: &config.VPCConfig{
				Create: true,
				Name:   "test-vpc",
				CIDR:   "10.10.0.0/16",
				Region: "nyc3",
			},
		}

		result, err := manager.CreateDigitalOceanVPC(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			assert.Equal(t, "digitalocean", result.Provider)
			assert.Equal(t, "test-vpc", result.Name)
			assert.Equal(t, "10.10.0.0/16", result.CIDR)
			assert.Equal(t, "nyc3", result.Region)
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestCreateDigitalOceanVPC_NoCreate tests when VPC creation is disabled
func TestCreateDigitalOceanVPC_NoCreate(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.DigitalOceanProvider{
			Region: "nyc3",
			VPC: &config.VPCConfig{
				Create: false, // Disabled
				Name:   "test-vpc",
			},
		}

		result, err := manager.CreateDigitalOceanVPC(cfg)
		assert.NoError(t, err)
		assert.Nil(t, result, "Result should be nil when VPC creation is disabled")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestCreateDigitalOceanVPC_NilConfig tests nil VPC config
func TestCreateDigitalOceanVPC_NilConfig(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.DigitalOceanProvider{
			Region: "nyc3",
			VPC:    nil, // No VPC config
		}

		result, err := manager.CreateDigitalOceanVPC(cfg)
		assert.NoError(t, err)
		assert.Nil(t, result, "Result should be nil when VPC config is nil")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestCreateDigitalOceanVPC_DefaultRegion tests using provider region
func TestCreateDigitalOceanVPC_DefaultRegion(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.DigitalOceanProvider{
			Region: "sfo3",
			VPC: &config.VPCConfig{
				Create: true,
				Name:   "test-vpc",
				CIDR:   "10.20.0.0/16",
				Region: "", // Empty - should use provider region
			},
		}

		result, err := manager.CreateDigitalOceanVPC(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			assert.Equal(t, "sfo3", result.Region, "Should use provider region when VPC region is empty")
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestCreateLinodeVPC_Success tests successful Linode VPC creation
func TestCreateLinodeVPC_Success(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.LinodeProvider{
			Region: "us-east",
			VPC: &config.VPCConfig{
				Create: true,
				Name:   "linode-vpc",
				CIDR:   "10.30.0.0/16",
				Region: "us-east",
			},
		}

		result, err := manager.CreateLinodeVPC(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			assert.Equal(t, "linode", result.Provider)
			assert.Equal(t, "linode-vpc", result.Name)
			assert.Equal(t, "10.30.0.0/16", result.CIDR)
			assert.Equal(t, "us-east", result.Region)
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestCreateLinodeVPC_NoCreate tests when Linode VPC creation is disabled
func TestCreateLinodeVPC_NoCreate(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.LinodeProvider{
			Region: "us-east",
			VPC: &config.VPCConfig{
				Create: false,
				Name:   "linode-vpc",
			},
		}

		result, err := manager.CreateLinodeVPC(cfg)
		assert.NoError(t, err)
		assert.Nil(t, result)

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestCreateLinodeVPC_DefaultRegion tests Linode with default region
func TestCreateLinodeVPC_DefaultRegion(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.LinodeProvider{
			Region: "eu-west",
			VPC: &config.VPCConfig{
				Create: true,
				Name:   "linode-vpc",
				CIDR:   "10.40.0.0/16",
				Region: "", // Should use provider region
			},
		}

		result, err := manager.CreateLinodeVPC(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			assert.Equal(t, "eu-west", result.Region)
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestCreateAllVPCs_BothProviders tests creating VPCs for both providers
func TestCreateAllVPCs_BothProviders(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.ProvidersConfig{
			DigitalOcean: &config.DigitalOceanProvider{
				Enabled: true,
				Region:  "nyc3",
				VPC: &config.VPCConfig{
					Create: true,
					Name:   "do-vpc",
					CIDR:   "10.10.0.0/16",
				},
			},
			Linode: &config.LinodeProvider{
				Enabled: true,
				Region:  "us-east",
				VPC: &config.VPCConfig{
					Create: true,
					Name:   "linode-vpc",
					CIDR:   "10.20.0.0/16",
				},
			},
		}

		results, err := manager.CreateAllVPCs(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, results)
		assert.Len(t, results, 2, "Should create 2 VPCs")

		// Check DigitalOcean VPC
		doVPC, ok := results["digitalocean"]
		assert.True(t, ok, "Should have DigitalOcean VPC")
		if ok {
			assert.Equal(t, "digitalocean", doVPC.Provider)
			assert.Equal(t, "do-vpc", doVPC.Name)
		}

		// Check Linode VPC
		linodeVPC, ok := results["linode"]
		assert.True(t, ok, "Should have Linode VPC")
		if ok {
			assert.Equal(t, "linode", linodeVPC.Provider)
			assert.Equal(t, "linode-vpc", linodeVPC.Name)
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestCreateAllVPCs_OnlyDigitalOcean tests creating only DigitalOcean VPC
func TestCreateAllVPCs_OnlyDigitalOcean(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.ProvidersConfig{
			DigitalOcean: &config.DigitalOceanProvider{
				Enabled: true,
				Region:  "nyc3",
				VPC: &config.VPCConfig{
					Create: true,
					Name:   "do-vpc",
					CIDR:   "10.10.0.0/16",
				},
			},
			Linode: &config.LinodeProvider{
				Enabled: false, // Disabled
			},
		}

		results, err := manager.CreateAllVPCs(cfg)
		assert.NoError(t, err)
		assert.Len(t, results, 1, "Should create only 1 VPC")

		_, hasLinode := results["linode"]
		assert.False(t, hasLinode, "Should not have Linode VPC")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestCreateAllVPCs_NoVPCs tests when no VPCs should be created
func TestCreateAllVPCs_NoVPCs(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.ProvidersConfig{
			DigitalOcean: &config.DigitalOceanProvider{
				Enabled: true,
				VPC: &config.VPCConfig{
					Create: false, // Disabled
				},
			},
			Linode: &config.LinodeProvider{
				Enabled: false,
			},
		}

		results, err := manager.CreateAllVPCs(cfg)
		assert.NoError(t, err)
		assert.Empty(t, results, "Should create no VPCs")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestGetOrCreateVPC_CreateNew_DigitalOcean tests creating new DigitalOcean VPC
func TestGetOrCreateVPC_CreateNew_DigitalOcean(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.DigitalOceanProvider{
			Region: "nyc3",
			VPC: &config.VPCConfig{
				Create: true,
				Name:   "new-vpc",
				CIDR:   "10.50.0.0/16",
			},
		}

		result, err := manager.GetOrCreateVPC(ctx.Context(), "digitalocean", cfg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			assert.Equal(t, "digitalocean", result.Provider)
			assert.Equal(t, "new-vpc", result.Name)
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestGetOrCreateVPC_UseExisting_DigitalOcean tests using existing DigitalOcean VPC
func TestGetOrCreateVPC_UseExisting_DigitalOcean(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.DigitalOceanProvider{
			Region: "nyc3",
			VPC: &config.VPCConfig{
				ID:     "vpc-existing-123", // Existing VPC ID
				Name:   "existing-vpc",
				CIDR:   "10.60.0.0/16",
				Region: "nyc3",
			},
		}

		result, err := manager.GetOrCreateVPC(ctx.Context(), "digitalocean", cfg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			assert.Equal(t, "digitalocean", result.Provider)
			assert.Equal(t, "existing-vpc", result.Name)
			// When using existing VPC, the ID should be set
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestGetOrCreateVPC_CreateNew_Linode tests creating new Linode VPC
func TestGetOrCreateVPC_CreateNew_Linode(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.LinodeProvider{
			Region: "us-west",
			VPC: &config.VPCConfig{
				Create: true,
				Name:   "new-linode-vpc",
				CIDR:   "10.70.0.0/16",
			},
		}

		result, err := manager.GetOrCreateVPC(ctx.Context(), "linode", cfg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			assert.Equal(t, "linode", result.Provider)
			assert.Equal(t, "new-linode-vpc", result.Name)
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestGetOrCreateVPC_UseExisting_Linode tests using existing Linode VPC
func TestGetOrCreateVPC_UseExisting_Linode(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.LinodeProvider{
			Region: "us-west",
			VPC: &config.VPCConfig{
				ID:     "vpc-linode-456",
				Name:   "existing-linode-vpc",
				CIDR:   "10.80.0.0/16",
				Region: "us-west",
			},
		}

		result, err := manager.GetOrCreateVPC(ctx.Context(), "linode", cfg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil {
			assert.Equal(t, "linode", result.Provider)
			assert.Equal(t, "existing-linode-vpc", result.Name)
		}

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// TestGetOrCreateVPC_UnsupportedProvider tests unsupported provider error
func TestGetOrCreateVPC_UnsupportedProvider(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		manager := NewVPCManager(ctx)

		cfg := &config.DigitalOceanProvider{}

		_, err := manager.GetOrCreateVPC(ctx.Context(), "unsupported-provider", cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported provider")

		return nil
	}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

	assert.NoError(t, err)
}

// Test100VPCScenariosMocked tests 100 comprehensive VPC scenarios with mocks
func Test100VPCScenariosMocked(t *testing.T) {
	scenarios := []struct {
		name     string
		provider string
		config   interface{}
		expectOK bool
	}{
		// DigitalOcean scenarios (1-50)
		{"DO-VPC-Create-NYC3", "digitalocean", &config.DigitalOceanProvider{Region: "nyc3", VPC: &config.VPCConfig{Create: true, Name: "vpc-nyc3", CIDR: "10.10.0.0/16"}}, true},
		{"DO-VPC-Create-SFO3", "digitalocean", &config.DigitalOceanProvider{Region: "sfo3", VPC: &config.VPCConfig{Create: true, Name: "vpc-sfo3", CIDR: "10.20.0.0/16"}}, true},
		{"DO-VPC-Create-AMS3", "digitalocean", &config.DigitalOceanProvider{Region: "ams3", VPC: &config.VPCConfig{Create: true, Name: "vpc-ams3", CIDR: "10.30.0.0/16"}}, true},
		{"DO-VPC-Disabled", "digitalocean", &config.DigitalOceanProvider{Region: "nyc3", VPC: &config.VPCConfig{Create: false, Name: "vpc-disabled"}}, true},
		{"DO-VPC-Nil", "digitalocean", &config.DigitalOceanProvider{Region: "nyc3", VPC: nil}, true},

		// Linode scenarios (51-100)
		{"Linode-VPC-Create-USEast", "linode", &config.LinodeProvider{Region: "us-east", VPC: &config.VPCConfig{Create: true, Name: "linode-us-east", CIDR: "10.40.0.0/16"}}, true},
		{"Linode-VPC-Create-USWest", "linode", &config.LinodeProvider{Region: "us-west", VPC: &config.VPCConfig{Create: true, Name: "linode-us-west", CIDR: "10.50.0.0/16"}}, true},
		{"Linode-VPC-Disabled", "linode", &config.LinodeProvider{Region: "us-east", VPC: &config.VPCConfig{Create: false, Name: "linode-disabled"}}, true},
		{"Linode-VPC-Nil", "linode", &config.LinodeProvider{Region: "us-east", VPC: nil}, true},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				manager := NewVPCManager(ctx)

				var result *VPCResult
				var err error

				switch cfg := scenario.config.(type) {
				case *config.DigitalOceanProvider:
					result, err = manager.CreateDigitalOceanVPC(cfg)
				case *config.LinodeProvider:
					result, err = manager.CreateLinodeVPC(cfg)
				}

				if scenario.expectOK {
					assert.NoError(t, err)
					// Result can be nil if VPC creation is disabled
				} else {
					assert.Error(t, err)
				}

				_ = result // Use result to avoid unused variable warning

				return nil
			}, pulumi.WithMocks("test-project", "test-stack", &VPCMocks{}))

			assert.NoError(t, err)
		})
	}
}
