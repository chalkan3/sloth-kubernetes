package validation

import (
	"fmt"

	"sloth-kubernetes/pkg/config"
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

	if dist.Total != 6 {
		return fmt.Errorf("configuration must define exactly 6 nodes, found %d", dist.Total)
	}

	if dist.Masters != 3 {
		return fmt.Errorf("configuration must define exactly 3 master nodes, found %d", dist.Masters)
	}

	if dist.Workers != 3 {
		return fmt.Errorf("configuration must define exactly 3 worker nodes, found %d", dist.Workers)
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
