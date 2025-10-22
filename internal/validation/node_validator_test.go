package validation

import (
	"strings"
	"testing"

	"sloth-kubernetes/pkg/config"
)

func TestValidateNodeDistribution(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.ClusterConfig
		wantErr       bool
		errorContains string
	}{
		{
			name: "Valid - 1 master, 2 workers",
			config: &config.ClusterConfig{
				NodePools: map[string]config.NodePool{
					"masters": {
						Name:     "masters",
						Provider: "digitalocean",
						Count:    1,
						Roles:    []string{"master"},
					},
					"workers": {
						Name:     "workers",
						Provider: "digitalocean",
						Count:    2,
						Roles:    []string{"worker"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid - 3 masters (HA)",
			config: &config.ClusterConfig{
				NodePools: map[string]config.NodePool{
					"masters": {
						Name:     "masters",
						Provider: "digitalocean",
						Count:    3,
						Roles:    []string{"master"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid - 5 masters (HA)",
			config: &config.ClusterConfig{
				NodePools: map[string]config.NodePool{
					"masters": {
						Name:     "masters",
						Provider: "digitalocean",
						Count:    5,
						Roles:    []string{"master"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid - No nodes",
			config: &config.ClusterConfig{
				NodePools: map[string]config.NodePool{},
				Nodes:     []config.NodeConfig{},
			},
			wantErr:       true,
			errorContains: "must define at least 1 node, found 0",
		},
		{
			name: "Invalid - No master nodes",
			config: &config.ClusterConfig{
				NodePools: map[string]config.NodePool{
					"workers": {
						Name:     "workers",
						Provider: "digitalocean",
						Count:    3,
						Roles:    []string{"worker"},
					},
				},
			},
			wantErr:       true,
			errorContains: "must define at least 1 master node, found 0",
		},
		{
			name: "Invalid - Even number of masters (2)",
			config: &config.ClusterConfig{
				NodePools: map[string]config.NodePool{
					"masters": {
						Name:     "masters",
						Provider: "digitalocean",
						Count:    2,
						Roles:    []string{"master"},
					},
				},
			},
			wantErr:       true,
			errorContains: "master nodes must be an odd number",
		},
		{
			name: "Invalid - Even number of masters (4)",
			config: &config.ClusterConfig{
				NodePools: map[string]config.NodePool{
					"masters": {
						Name:     "masters",
						Provider: "digitalocean",
						Count:    4,
						Roles:    []string{"master"},
					},
				},
			},
			wantErr:       true,
			errorContains: "master nodes must be an odd number",
		},
		{
			name: "Valid - Mixed nodes and nodepools",
			config: &config.ClusterConfig{
				Nodes: []config.NodeConfig{
					{
						Name:     "master-1",
						Provider: "digitalocean",
						Roles:    []string{"master"},
					},
				},
				NodePools: map[string]config.NodePool{
					"workers": {
						Name:     "workers",
						Provider: "linode",
						Count:    2,
						Roles:    []string{"worker"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid - Controlplane role (synonym for master)",
			config: &config.ClusterConfig{
				NodePools: map[string]config.NodePool{
					"masters": {
						Name:     "masters",
						Provider: "digitalocean",
						Count:    3,
						Roles:    []string{"controlplane"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNodeDistribution(tt.config)

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

func TestCalculateDistribution(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.ClusterConfig
		expectedTotal  int
		expectedMasters int
		expectedWorkers int
		expectedByProvider map[string]int
	}{
		{
			name: "Simple config with NodePools",
			config: &config.ClusterConfig{
				NodePools: map[string]config.NodePool{
					"masters": {
						Name:     "masters",
						Provider: "digitalocean",
						Count:    3,
						Roles:    []string{"master"},
					},
					"workers": {
						Name:     "workers",
						Provider: "digitalocean",
						Count:    5,
						Roles:    []string{"worker"},
					},
				},
			},
			expectedTotal:  8,
			expectedMasters: 3,
			expectedWorkers: 5,
			expectedByProvider: map[string]int{
				"digitalocean": 8,
			},
		},
		{
			name: "Multi-cloud config",
			config: &config.ClusterConfig{
				NodePools: map[string]config.NodePool{
					"do-masters": {
						Name:     "do-masters",
						Provider: "digitalocean",
						Count:    1,
						Roles:    []string{"master"},
					},
					"linode-masters": {
						Name:     "linode-masters",
						Provider: "linode",
						Count:    2,
						Roles:    []string{"master"},
					},
					"do-workers": {
						Name:     "do-workers",
						Provider: "digitalocean",
						Count:    2,
						Roles:    []string{"worker"},
					},
					"linode-workers": {
						Name:     "linode-workers",
						Provider: "linode",
						Count:    3,
						Roles:    []string{"worker"},
					},
				},
			},
			expectedTotal:  8,
			expectedMasters: 3,
			expectedWorkers: 5,
			expectedByProvider: map[string]int{
				"digitalocean": 3,
				"linode":       5,
			},
		},
		{
			name: "Config with individual nodes",
			config: &config.ClusterConfig{
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
					{
						Name:     "worker-2",
						Provider: "linode",
						Roles:    []string{"worker"},
					},
				},
			},
			expectedTotal:  3,
			expectedMasters: 1,
			expectedWorkers: 2,
			expectedByProvider: map[string]int{
				"digitalocean": 2,
				"linode":       1,
			},
		},
		{
			name: "Mixed nodes and nodepools",
			config: &config.ClusterConfig{
				Nodes: []config.NodeConfig{
					{
						Name:     "master-1",
						Provider: "digitalocean",
						Roles:    []string{"master"},
					},
				},
				NodePools: map[string]config.NodePool{
					"workers": {
						Name:     "workers",
						Provider: "linode",
						Count:    4,
						Roles:    []string{"worker"},
					},
				},
			},
			expectedTotal:  5,
			expectedMasters: 1,
			expectedWorkers: 4,
			expectedByProvider: map[string]int{
				"digitalocean": 1,
				"linode":       4,
			},
		},
		{
			name: "Empty config",
			config: &config.ClusterConfig{
				NodePools: map[string]config.NodePool{},
				Nodes:     []config.NodeConfig{},
			},
			expectedTotal:  0,
			expectedMasters: 0,
			expectedWorkers: 0,
			expectedByProvider: map[string]int{},
		},
		{
			name: "Controlplane role",
			config: &config.ClusterConfig{
				NodePools: map[string]config.NodePool{
					"control": {
						Name:     "control",
						Provider: "digitalocean",
						Count:    3,
						Roles:    []string{"controlplane"},
					},
				},
			},
			expectedTotal:  3,
			expectedMasters: 3,
			expectedWorkers: 0,
			expectedByProvider: map[string]int{
				"digitalocean": 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := CalculateDistribution(tt.config)

			if dist.Total != tt.expectedTotal {
				t.Errorf("expected total %d, got %d", tt.expectedTotal, dist.Total)
			}
			if dist.Masters != tt.expectedMasters {
				t.Errorf("expected masters %d, got %d", tt.expectedMasters, dist.Masters)
			}
			if dist.Workers != tt.expectedWorkers {
				t.Errorf("expected workers %d, got %d", tt.expectedWorkers, dist.Workers)
			}

			for provider, expected := range tt.expectedByProvider {
				if dist.ByProvider[provider] != expected {
					t.Errorf("expected %d nodes for provider %s, got %d",
						expected, provider, dist.ByProvider[provider])
				}
			}
		})
	}
}

func TestGetDistributionSummary(t *testing.T) {
	config := &config.ClusterConfig{
		NodePools: map[string]config.NodePool{
			"do-masters": {
				Name:     "do-masters",
				Provider: "digitalocean",
				Count:    3,
				Roles:    []string{"master"},
			},
			"linode-workers": {
				Name:     "linode-workers",
				Provider: "linode",
				Count:    5,
				Roles:    []string{"worker"},
			},
		},
	}

	summary := GetDistributionSummary(config)

	// Check that summary contains expected information
	expectedStrings := []string{
		"Total Nodes: 8",
		"Masters: 3",
		"Workers: 5",
		"By Provider:",
		"digitalocean: 3 nodes",
		"linode: 5 nodes",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(summary, expected) {
			t.Errorf("summary does not contain expected string '%s'\nGot: %s", expected, summary)
		}
	}
}

func TestGetDistributionSummary_EmptyConfig(t *testing.T) {
	config := &config.ClusterConfig{
		NodePools: map[string]config.NodePool{},
		Nodes:     []config.NodeConfig{},
	}

	summary := GetDistributionSummary(config)

	expectedStrings := []string{
		"Total Nodes: 0",
		"Masters: 0",
		"Workers: 0",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(summary, expected) {
			t.Errorf("summary does not contain expected string '%s'\nGot: %s", expected, summary)
		}
	}
}
