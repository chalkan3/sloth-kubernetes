package config

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// LoadPulumiConfig loads cluster configuration from Pulumi config
// This replaces the loadConfigFromPulumi function in main.go
func LoadPulumiConfig(ctx *pulumi.Context) (*ClusterConfig, error) {
	conf := config.New(ctx, "")

	// Get provider tokens from Pulumi config
	doToken := conf.Require("digitaloceanToken")
	linodeToken := conf.Require("linodeToken")

	// Get WireGuard configuration from Pulumi config
	wgEndpoint := conf.Require("wireguardServerEndpoint")
	wgPubKey := conf.Require("wireguardServerPublicKey")

	// Get RKE2 cluster token from Pulumi config (optional, will generate if not set)
	rke2Token := conf.Get("rke2ClusterToken")
	if rke2Token == "" {
		rke2Token = "my-super-secret-cluster-token-rke2-production-2025"
	}

	// Build cluster config
	cfg := &ClusterConfig{
		Metadata: Metadata{
			Name: "production",
		},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled: true,
				Token:   doToken,
				Region:  "nyc3",
			},
			Linode: &LinodeProvider{
				Enabled:      true,
				Token:        linodeToken,
				Region:       "us-east",
				RootPassword: "SecureLinodeRootPass2025!",
			},
		},
		Network: NetworkConfig{
			DNS: DNSConfig{
				Domain:   "chalkan3.com.br",
				Provider: "digitalocean",
			},
			WireGuard: &WireGuardConfig{
				Enabled:         true,
				ServerEndpoint:  wgEndpoint,
				ServerPublicKey: wgPubKey,
			},
		},
		NodePools: map[string]NodePool{
			"do-masters": {
				Name:     "do-masters",
				Count:    1,
				Size:     "s-2vcpu-4gb",
				Image:    "ubuntu-22-04-x64",
				Region:   "nyc3",
				Provider: "digitalocean",
				Roles:    []string{"master"},
			},
			"do-workers": {
				Name:     "do-workers",
				Count:    2,
				Size:     "s-2vcpu-4gb",
				Image:    "ubuntu-22-04-x64",
				Region:   "nyc3",
				Provider: "digitalocean",
				Roles:    []string{"worker"},
			},
			"linode-masters": {
				Name:     "linode-masters",
				Count:    2,
				Size:     "g6-standard-2",
				Image:    "linode/ubuntu22.04",
				Region:   "us-east",
				Provider: "linode",
				Roles:    []string{"master"},
			},
			"linode-workers": {
				Name:     "linode-workers",
				Count:    1,
				Size:     "g6-standard-2",
				Image:    "linode/ubuntu22.04",
				Region:   "us-east",
				Provider: "linode",
				Roles:    []string{"worker"},
			},
		},
	}

	return cfg, nil
}
