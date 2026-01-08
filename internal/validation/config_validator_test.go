package validation

import (
	"strings"
	"testing"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

func TestValidateClusterConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.ClusterConfig
		wantErr       bool
		errorContains string
	}{
		{
			name: "Valid configuration",
			config: &config.ClusterConfig{
				Metadata: config.Metadata{
					Name: "test-cluster",
				},
				Providers: config.ProvidersConfig{
					DigitalOcean: &config.DigitalOceanProvider{
						Enabled: true,
						Token:   "test-token",
						Region:  "nyc3",
					},
				},
				Network: config.NetworkConfig{
					WireGuard: &config.WireGuardConfig{
						Enabled:         true,
						Create:          true,
						Provider:        "digitalocean",
						Region:          "nyc3",
						ServerEndpoint:  "",
						ServerPublicKey: "",
					},
				},
				Nodes: []config.NodeConfig{
					{
						Name:     "master-1",
						Provider: "digitalocean",
						Roles:    []string{"master"},
					},
					{
						Name:     "worker-1",
						Provider: "digitalocean",
						Roles:    []string{"worker"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid - VPN disabled",
			config: &config.ClusterConfig{
				Metadata: config.Metadata{
					Name: "test-cluster",
				},
				Providers: config.ProvidersConfig{
					DigitalOcean: &config.DigitalOceanProvider{
						Enabled: true,
						Token:   "test-token",
					},
				},
				Network: config.NetworkConfig{
					WireGuard: &config.WireGuardConfig{
						Enabled: false,
					},
				},
				Nodes: []config.NodeConfig{
					{Name: "master-1", Roles: []string{"master"}},
				},
			},
			wantErr:       true,
			errorContains: "VPN (WireGuard or Tailscale) must be enabled",
		},
		{
			name: "Invalid - No provider enabled",
			config: &config.ClusterConfig{
				Metadata: config.Metadata{
					Name: "test-cluster",
				},
				Providers: config.ProvidersConfig{
					DigitalOcean: &config.DigitalOceanProvider{
						Enabled: false,
					},
				},
				Network: config.NetworkConfig{
					WireGuard: &config.WireGuardConfig{
						Enabled:  true,
						Create:   true,
						Provider: "digitalocean",
						Region:   "nyc3",
					},
				},
				Nodes: []config.NodeConfig{
					{Name: "master-1", Roles: []string{"master"}},
				},
			},
			wantErr:       true,
			errorContains: "at least one cloud provider must be enabled",
		},
		{
			name: "Invalid - Missing token",
			config: &config.ClusterConfig{
				Metadata: config.Metadata{
					Name: "test-cluster",
				},
				Providers: config.ProvidersConfig{
					DigitalOcean: &config.DigitalOceanProvider{
						Enabled: true,
						Token:   "",
					},
				},
				Network: config.NetworkConfig{
					WireGuard: &config.WireGuardConfig{
						Enabled:  true,
						Create:   true,
						Provider: "digitalocean",
						Region:   "nyc3",
					},
				},
				Nodes: []config.NodeConfig{
					{Name: "master-1", Roles: []string{"master"}},
				},
			},
			wantErr:       true,
			errorContains: "API token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClusterConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error '%v' does not contain '%s'", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateWireGuardConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.ClusterConfig
		wantErr       bool
		errorContains string
	}{
		{
			name: "Valid - Auto-create VPN",
			config: &config.ClusterConfig{
				Network: config.NetworkConfig{
					WireGuard: &config.WireGuardConfig{
						Enabled:  true,
						Create:   true,
						Provider: "digitalocean",
						Region:   "nyc3",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid - Existing VPN",
			config: &config.ClusterConfig{
				Network: config.NetworkConfig{
					WireGuard: &config.WireGuardConfig{
						Enabled:         true,
						Create:          false,
						ServerEndpoint:  "1.2.3.4:51820",
						ServerPublicKey: "test-public-key",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid - WireGuard not enabled but using existing",
			config: &config.ClusterConfig{
				Network: config.NetworkConfig{
					WireGuard: &config.WireGuardConfig{
						Enabled: false,
						Create:  false,
					},
				},
			},
			wantErr:       true,
			errorContains: "WireGuard server endpoint is required",
		},
		{
			name: "Invalid - WireGuard not enabled but creating",
			config: &config.ClusterConfig{
				Network: config.NetworkConfig{
					WireGuard: &config.WireGuardConfig{
						Enabled: false,
						Create:  true,
					},
				},
			},
			wantErr:       true,
			errorContains: "WireGuard provider is required",
		},
		{
			name: "Invalid - Auto-create without provider",
			config: &config.ClusterConfig{
				Network: config.NetworkConfig{
					WireGuard: &config.WireGuardConfig{
						Enabled:  true,
						Create:   true,
						Provider: "",
						Region:   "nyc3",
					},
				},
			},
			wantErr:       true,
			errorContains: "WireGuard provider is required",
		},
		{
			name: "Invalid - Auto-create without region",
			config: &config.ClusterConfig{
				Network: config.NetworkConfig{
					WireGuard: &config.WireGuardConfig{
						Enabled:  true,
						Create:   true,
						Provider: "digitalocean",
						Region:   "",
					},
				},
			},
			wantErr:       true,
			errorContains: "WireGuard region is required",
		},
		{
			name: "Invalid - Existing VPN without endpoint",
			config: &config.ClusterConfig{
				Network: config.NetworkConfig{
					WireGuard: &config.WireGuardConfig{
						Enabled:         true,
						Create:          false,
						ServerEndpoint:  "",
						ServerPublicKey: "test-key",
					},
				},
			},
			wantErr:       true,
			errorContains: "WireGuard server endpoint is required",
		},
		{
			name: "Invalid - Existing VPN without public key",
			config: &config.ClusterConfig{
				Network: config.NetworkConfig{
					WireGuard: &config.WireGuardConfig{
						Enabled:         true,
						Create:          false,
						ServerEndpoint:  "1.2.3.4:51820",
						ServerPublicKey: "",
					},
				},
			},
			wantErr:       true,
			errorContains: "WireGuard server public key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWireGuardConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error '%v' does not contain '%s'", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateProviders(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.ClusterConfig
		wantErr       bool
		errorContains string
	}{
		{
			name: "Valid - DigitalOcean only",
			config: &config.ClusterConfig{
				Providers: config.ProvidersConfig{
					DigitalOcean: &config.DigitalOceanProvider{
						Enabled: true,
						Token:   "do-token",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid - Linode only",
			config: &config.ClusterConfig{
				Providers: config.ProvidersConfig{
					Linode: &config.LinodeProvider{
						Enabled: true,
						Token:   "linode-token",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid - Both providers",
			config: &config.ClusterConfig{
				Providers: config.ProvidersConfig{
					DigitalOcean: &config.DigitalOceanProvider{
						Enabled: true,
						Token:   "do-token",
					},
					Linode: &config.LinodeProvider{
						Enabled: true,
						Token:   "linode-token",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid - No providers enabled",
			config: &config.ClusterConfig{
				Providers: config.ProvidersConfig{
					DigitalOcean: &config.DigitalOceanProvider{
						Enabled: false,
					},
					Linode: &config.LinodeProvider{
						Enabled: false,
					},
				},
			},
			wantErr:       true,
			errorContains: "at least one cloud provider must be enabled",
		},
		{
			name: "Invalid - DigitalOcean enabled but no token",
			config: &config.ClusterConfig{
				Providers: config.ProvidersConfig{
					DigitalOcean: &config.DigitalOceanProvider{
						Enabled: true,
						Token:   "",
					},
				},
			},
			wantErr:       true,
			errorContains: "DigitalOcean API token is required",
		},
		{
			name: "Invalid - Linode enabled but no token",
			config: &config.ClusterConfig{
				Providers: config.ProvidersConfig{
					Linode: &config.LinodeProvider{
						Enabled: true,
						Token:   "",
					},
				},
			},
			wantErr:       true,
			errorContains: "Linode API token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProviders(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error '%v' does not contain '%s'", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateDNSConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.ClusterConfig
		wantErr       bool
		errorContains string
	}{
		{
			name: "Valid DNS config",
			config: &config.ClusterConfig{
				Network: config.NetworkConfig{
					DNS: config.DNSConfig{
						Domain:   "example.com",
						Provider: "digitalocean",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid - Missing domain",
			config: &config.ClusterConfig{
				Network: config.NetworkConfig{
					DNS: config.DNSConfig{
						Domain:   "",
						Provider: "digitalocean",
					},
				},
			},
			wantErr:       true,
			errorContains: "DNS domain is required",
		},
		{
			name: "Invalid - Missing provider",
			config: &config.ClusterConfig{
				Network: config.NetworkConfig{
					DNS: config.DNSConfig{
						Domain:   "example.com",
						Provider: "",
					},
				},
			},
			wantErr:       true,
			errorContains: "DNS provider is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDNSConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error '%v' does not contain '%s'", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateMetadata(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.ClusterConfig
		wantErr       bool
		errorContains string
	}{
		{
			name: "Valid metadata",
			config: &config.ClusterConfig{
				Metadata: config.Metadata{
					Name: "my-cluster",
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid - Empty cluster name",
			config: &config.ClusterConfig{
				Metadata: config.Metadata{
					Name: "",
				},
			},
			wantErr:       true,
			errorContains: "cluster name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMetadata(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error '%v' does not contain '%s'", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
