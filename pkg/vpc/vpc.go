package vpc

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi-linode/sdk/v4/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
)

// VPCManager handles VPC creation across providers
type VPCManager struct {
	ctx *pulumi.Context
}

// NewVPCManager creates a new VPC manager
func NewVPCManager(ctx *pulumi.Context) *VPCManager {
	return &VPCManager{ctx: ctx}
}

// VPCResult contains created VPC information
type VPCResult struct {
	Provider string
	ID       pulumi.IDOutput
	Name     string
	CIDR     string
	Region   string
}

// CreateDigitalOceanVPC creates a VPC on DigitalOcean
func (m *VPCManager) CreateDigitalOceanVPC(cfg *config.DigitalOceanProvider) (*VPCResult, error) {
	if cfg.VPC == nil || !cfg.VPC.Create {
		return nil, nil // No VPC to create
	}

	vpcCfg := cfg.VPC

	// Use provider region if VPC region is not specified
	region := vpcCfg.Region
	if region == "" {
		region = cfg.Region
	}

	// Create VPC
	vpc, err := digitalocean.NewVpc(m.ctx, vpcCfg.Name, &digitalocean.VpcArgs{
		Name:        pulumi.String(vpcCfg.Name),
		Region:      pulumi.String(region),
		IpRange:     pulumi.String(vpcCfg.CIDR),
		Description: pulumi.String(fmt.Sprintf("VPC for Kubernetes cluster - %s", vpcCfg.Name)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DigitalOcean VPC: %w", err)
	}

	// Export VPC information
	secrets.Export(m.ctx, "digitalocean_vpc_id", vpc.ID())
	secrets.Export(m.ctx, "digitalocean_vpc_urn", vpc.URN())
	secrets.Export(m.ctx, "digitalocean_vpc_ip_range", vpc.IpRange)

	return &VPCResult{
		Provider: "digitalocean",
		ID:       vpc.ID(),
		Name:     vpcCfg.Name,
		CIDR:     vpcCfg.CIDR,
		Region:   region,
	}, nil
}

// CreateLinodeVPC creates a VPC on Linode
func (m *VPCManager) CreateLinodeVPC(cfg *config.LinodeProvider) (*VPCResult, error) {
	if cfg.VPC == nil || !cfg.VPC.Create {
		return nil, nil // No VPC to create
	}

	vpcCfg := cfg.VPC

	// Use provider region if VPC region is not specified
	region := vpcCfg.Region
	if region == "" {
		region = cfg.Region
	}

	// Create VPC
	vpcArgs := &linode.VpcArgs{
		Label:       pulumi.String(vpcCfg.Name),
		Region:      pulumi.String(region),
		Description: pulumi.String(fmt.Sprintf("VPC for Kubernetes cluster - %s", vpcCfg.Name)),
	}

	vpc, err := linode.NewVpc(m.ctx, vpcCfg.Name, vpcArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create Linode VPC: %w", err)
	}

	// Export VPC information
	secrets.Export(m.ctx, "linode_vpc_id", vpc.ID())
	secrets.Export(m.ctx, "linode_vpc_label", vpc.Label)

	return &VPCResult{
		Provider: "linode",
		ID:       vpc.ID(),
		Name:     vpcCfg.Name,
		CIDR:     vpcCfg.CIDR,
		Region:   region,
	}, nil
}

// CreateAllVPCs creates VPCs for all enabled providers
func (m *VPCManager) CreateAllVPCs(cfg *config.ProvidersConfig) (map[string]*VPCResult, error) {
	results := make(map[string]*VPCResult)

	// Create DigitalOcean VPC
	if cfg.DigitalOcean != nil && cfg.DigitalOcean.Enabled {
		result, err := m.CreateDigitalOceanVPC(cfg.DigitalOcean)
		if err != nil {
			return nil, err
		}
		if result != nil {
			results["digitalocean"] = result
			m.ctx.Log.Info(fmt.Sprintf("✅ Created DigitalOcean VPC: %s (%s)", result.Name, result.CIDR), nil)
		}
	}

	// Create Linode VPC
	if cfg.Linode != nil && cfg.Linode.Enabled {
		result, err := m.CreateLinodeVPC(cfg.Linode)
		if err != nil {
			return nil, err
		}
		if result != nil {
			results["linode"] = result
			m.ctx.Log.Info(fmt.Sprintf("✅ Created Linode VPC: %s (%s)", result.Name, result.CIDR), nil)
		}
	}

	return results, nil
}

// GetOrCreateVPC gets existing VPC or creates a new one
func (m *VPCManager) GetOrCreateVPC(ctx context.Context, provider string, cfg interface{}) (*VPCResult, error) {
	switch provider {
	case "digitalocean":
		if doCfg, ok := cfg.(*config.DigitalOceanProvider); ok {
			if doCfg.VPC != nil && doCfg.VPC.ID != "" {
				// Use existing VPC
				return &VPCResult{
					Provider: "digitalocean",
					ID:       pulumi.ID(doCfg.VPC.ID).ToIDOutput(),
					Name:     doCfg.VPC.Name,
					CIDR:     doCfg.VPC.CIDR,
					Region:   doCfg.VPC.Region,
				}, nil
			}
			// Create new VPC
			return m.CreateDigitalOceanVPC(doCfg)
		}

	case "linode":
		if linodeCfg, ok := cfg.(*config.LinodeProvider); ok {
			if linodeCfg.VPC != nil && linodeCfg.VPC.ID != "" {
				// Use existing VPC
				return &VPCResult{
					Provider: "linode",
					ID:       pulumi.ID(linodeCfg.VPC.ID).ToIDOutput(),
					Name:     linodeCfg.VPC.Name,
					CIDR:     linodeCfg.VPC.CIDR,
					Region:   linodeCfg.VPC.Region,
				}, nil
			}
			// Create new VPC
			return m.CreateLinodeVPC(linodeCfg)
		}
	}

	return nil, fmt.Errorf("unsupported provider or invalid configuration: %s", provider)
}
