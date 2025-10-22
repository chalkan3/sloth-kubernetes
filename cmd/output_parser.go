package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"gopkg.in/yaml.v3"
)

// NodeInfo represents a single node's information parsed from outputs
type NodeInfo struct {
	Name        string   `json:"name" yaml:"name"`
	Provider    string   `json:"provider" yaml:"provider"`
	Region      string   `json:"region" yaml:"region"`
	Size        string   `json:"size" yaml:"size"`
	PublicIP    string   `json:"publicIP" yaml:"publicIP"`
	PrivateIP   string   `json:"privateIP" yaml:"privateIP"`
	WireGuardIP string   `json:"wireGuardIP" yaml:"wireGuardIP"`
	Roles       []string `json:"roles" yaml:"roles"`
	Status      string   `json:"status" yaml:"status"`
}

// VPNPeerInfo represents a VPN peer (external client)
type VPNPeerInfo struct {
	PublicKey  string `json:"publicKey"`
	VPNAddress string `json:"vpnAddress"`
}

// ClusterInfo represents overall cluster information
type ClusterInfo struct {
	Name        string     `json:"name" yaml:"name"`
	Nodes       []NodeInfo `json:"nodes" yaml:"nodes"`
	KubeConfig  string     `json:"kubeConfig,omitempty" yaml:"kubeConfig,omitempty"`
	APIEndpoint string     `json:"apiEndpoint,omitempty" yaml:"apiEndpoint,omitempty"`
	Status      string     `json:"status" yaml:"status"`
}

// ParseNodeOutputs extracts node information from Pulumi stack outputs
func ParseNodeOutputs(outputs auto.OutputMap) ([]NodeInfo, error) {
	nodes := []NodeInfo{}

	// Check if we have the structured "nodes" output
	nodesOutput, ok := outputs["nodes"]
	if !ok {
		// Try to parse from legacy individual outputs
		return parseLegacyNodeOutputs(outputs)
	}

	// The nodes output is a map like:
	// {
	//   "node_0": { "name": "...", "public_ip": "...", ...},
	//   "node_1": { "name": "...", "public_ip": "...", ...},
	// }
	nodesMap, ok := nodesOutput.Value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("nodes output is not a map")
	}

	for _, nodeData := range nodesMap {
		nodeMap, ok := nodeData.(map[string]interface{})
		if !ok {
			continue
		}

		node := NodeInfo{}

		if name, ok := nodeMap["name"].(string); ok {
			node.Name = name
		}
		if publicIP, ok := nodeMap["public_ip"].(string); ok {
			node.PublicIP = publicIP
		}
		if privateIP, ok := nodeMap["private_ip"].(string); ok {
			node.PrivateIP = privateIP
		}
		if vpnIP, ok := nodeMap["vpn_ip"].(string); ok {
			node.WireGuardIP = vpnIP
		}
		if provider, ok := nodeMap["provider"].(string); ok {
			node.Provider = provider
		}
		if region, ok := nodeMap["region"].(string); ok {
			node.Region = region
		}
		if size, ok := nodeMap["size"].(string); ok {
			node.Size = size
		}
		if status, ok := nodeMap["status"].(string); ok {
			node.Status = status
		}

		// Parse roles array
		if rolesData, ok := nodeMap["roles"].([]interface{}); ok {
			for _, role := range rolesData {
				if roleStr, ok := role.(string); ok {
					node.Roles = append(node.Roles, roleStr)
				}
			}
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// parseLegacyNodeOutputs parses node outputs from individual keys (fallback)
func parseLegacyNodeOutputs(outputs auto.OutputMap) ([]NodeInfo, error) {
	// For legacy outputs or as fallback, return empty list
	// This would need to parse outputs like "do-masters-1_public_ip"
	return []NodeInfo{}, nil
}

// ParseClusterOutputs extracts cluster-level information
func ParseClusterOutputs(outputs auto.OutputMap) (*ClusterInfo, error) {
	cluster := &ClusterInfo{
		Status: "Unknown",
	}

	// Extract cluster name
	if nameOutput, ok := outputs["clusterName"]; ok {
		if str, ok := nameOutput.Value.(string); ok {
			cluster.Name = str
		}
	}

	// Extract kubeconfig
	if kubeconfigOutput, ok := outputs["kubeConfig"]; ok {
		if str, ok := kubeconfigOutput.Value.(string); ok {
			cluster.KubeConfig = str
		}
	}

	// Extract API endpoint
	if apiOutput, ok := outputs["apiEndpoint"]; ok {
		if str, ok := apiOutput.Value.(string); ok {
			cluster.APIEndpoint = str
		}
	}

	// Extract status
	if statusOutput, ok := outputs["status"]; ok {
		if str, ok := statusOutput.Value.(string); ok {
			cluster.Status = str
		}
	}

	// Parse nodes
	nodes, err := ParseNodeOutputs(outputs)
	if err != nil {
		return nil, err
	}
	cluster.Nodes = nodes

	return cluster, nil
}

// FormatNodesAsJSON formats node information as JSON
func FormatNodesAsJSON(nodes []NodeInfo) (string, error) {
	data, err := json.MarshalIndent(nodes, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// FormatClusterAsJSON formats cluster information as JSON
func FormatClusterAsJSON(cluster *ClusterInfo) (string, error) {
	data, err := json.MarshalIndent(cluster, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// FormatNodesAsYAML formats node information as YAML
func FormatNodesAsYAML(nodes []NodeInfo) (string, error) {
	data, err := yaml.Marshal(nodes)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return string(data), nil
}

// FormatClusterAsYAML formats cluster information as YAML
func FormatClusterAsYAML(cluster *ClusterInfo) (string, error) {
	data, err := yaml.Marshal(cluster)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return string(data), nil
}

// GetSSHKeyPath returns the SSH private key path for a stack
func GetSSHKeyPath(stackName string) string {
	// Expand home directory to absolute path
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "~"
	}
	return fmt.Sprintf("%s/.ssh/kubernetes-clusters/%s.pem", homeDir, stackName)
}
