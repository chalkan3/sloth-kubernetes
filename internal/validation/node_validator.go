package validation

import (
	"fmt"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// NodeDistribution holds information about node distribution
type NodeDistribution struct {
	Total      int
	Masters    int
	Workers    int
	ByProvider map[string]int
}

// ValidateNodeDistribution validates that the cluster has the correct node distribution
func ValidateNodeDistribution(cfg *config.ClusterConfig) error {
	dist := CalculateDistribution(cfg)

	if dist.Total == 0 {
		return fmt.Errorf("configuration must define at least 1 node, found 0")
	}

	if dist.Masters == 0 {
		return fmt.Errorf("configuration must define at least 1 master node, found 0")
	}

	// Validate odd number of masters for HA
	if dist.Masters > 1 && dist.Masters%2 == 0 {
		return fmt.Errorf("for HA, master nodes must be an odd number (1, 3, 5, ...), found %d", dist.Masters)
	}

	return nil
}

// CalculateDistribution calculates node distribution from configuration
func CalculateDistribution(cfg *config.ClusterConfig) NodeDistribution {
	dist := NodeDistribution{
		ByProvider: make(map[string]int),
	}

	// Count nodes from NodePools
	for _, pool := range cfg.NodePools {
		dist.Total += pool.Count
		dist.ByProvider[pool.Provider] += pool.Count

		for _, role := range pool.Roles {
			if role == "controlplane" || role == "master" {
				dist.Masters += pool.Count
				break
			} else if role == "worker" {
				dist.Workers += pool.Count
				break
			}
		}
	}

	// Count nodes from individual Nodes
	for _, node := range cfg.Nodes {
		dist.Total++
		dist.ByProvider[node.Provider]++

		for _, role := range node.Roles {
			if role == "controlplane" || role == "master" {
				dist.Masters++
				break
			} else if role == "worker" {
				dist.Workers++
				break
			}
		}
	}

	return dist
}

// GetDistributionSummary returns a human-readable summary of node distribution
func GetDistributionSummary(cfg *config.ClusterConfig) string {
	dist := CalculateDistribution(cfg)

	summary := fmt.Sprintf("Total Nodes: %d\n", dist.Total)
	summary += fmt.Sprintf("- Masters: %d\n", dist.Masters)
	summary += fmt.Sprintf("- Workers: %d\n", dist.Workers)
	summary += "\nBy Provider:\n"

	for provider, count := range dist.ByProvider {
		summary += fmt.Sprintf("- %s: %d nodes\n", provider, count)
	}

	return summary
}
