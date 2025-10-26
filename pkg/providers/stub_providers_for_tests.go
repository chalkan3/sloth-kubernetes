//go:build !ignore
// +build !ignore

package providers

import (
	"fmt"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Stub implementations for incomplete providers (AWS, Azure, GCP)
// These are only used when the real implementations are excluded from build

// StubAWSProvider is a stub for AWS provider
type StubAWSProvider struct{}

func NewAWSProvider() *StubAWSProvider {
	return &StubAWSProvider{}
}

func (p *StubAWSProvider) GetName() string { return "aws" }
func (p *StubAWSProvider) Initialize(ctx *pulumi.Context, config *config.ClusterConfig) error {
	return fmt.Errorf("AWS provider not available in this build")
}
func (p *StubAWSProvider) CreateNode(ctx *pulumi.Context, node *config.NodeConfig) (*NodeOutput, error) {
	return nil, fmt.Errorf("AWS provider not available")
}
func (p *StubAWSProvider) CreateNodePool(ctx *pulumi.Context, pool *config.NodePool) ([]*NodeOutput, error) {
	return nil, fmt.Errorf("AWS provider not available")
}
func (p *StubAWSProvider) CreateNetwork(ctx *pulumi.Context, network *config.NetworkConfig) (*NetworkOutput, error) {
	return nil, fmt.Errorf("AWS provider not available")
}
func (p *StubAWSProvider) CreateFirewall(ctx *pulumi.Context, firewall *config.FirewallConfig, nodeIds []pulumi.IDOutput) error {
	return fmt.Errorf("AWS provider not available")
}
func (p *StubAWSProvider) CreateLoadBalancer(ctx *pulumi.Context, lb *config.LoadBalancerConfig) (*LoadBalancerOutput, error) {
	return nil, fmt.Errorf("AWS provider not available")
}
func (p *StubAWSProvider) GetRegions() []string              { return []string{} }
func (p *StubAWSProvider) GetSizes() []string                { return []string{} }
func (p *StubAWSProvider) Cleanup(ctx *pulumi.Context) error { return nil }

// Note: Azure provider is now fully implemented in azure.go

// StubGCPProvider is a stub for GCP provider
type StubGCPProvider struct{}

func NewGCPProvider() *StubGCPProvider {
	return &StubGCPProvider{}
}

func (p *StubGCPProvider) GetName() string { return "gcp" }
func (p *StubGCPProvider) Initialize(ctx *pulumi.Context, config *config.ClusterConfig) error {
	return fmt.Errorf("GCP provider not available in this build")
}
func (p *StubGCPProvider) CreateNode(ctx *pulumi.Context, node *config.NodeConfig) (*NodeOutput, error) {
	return nil, fmt.Errorf("GCP provider not available")
}
func (p *StubGCPProvider) CreateNodePool(ctx *pulumi.Context, pool *config.NodePool) ([]*NodeOutput, error) {
	return nil, fmt.Errorf("GCP provider not available")
}
func (p *StubGCPProvider) CreateNetwork(ctx *pulumi.Context, network *config.NetworkConfig) (*NetworkOutput, error) {
	return nil, fmt.Errorf("GCP provider not available")
}
func (p *StubGCPProvider) CreateFirewall(ctx *pulumi.Context, firewall *config.FirewallConfig, nodeIds []pulumi.IDOutput) error {
	return fmt.Errorf("GCP provider not available")
}
func (p *StubGCPProvider) CreateLoadBalancer(ctx *pulumi.Context, lb *config.LoadBalancerConfig) (*LoadBalancerOutput, error) {
	return nil, fmt.Errorf("GCP provider not available")
}
func (p *StubGCPProvider) GetRegions() []string              { return []string{} }
func (p *StubGCPProvider) GetSizes() []string                { return []string{} }
func (p *StubGCPProvider) Cleanup(ctx *pulumi.Context) error { return nil }
