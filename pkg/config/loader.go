package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	pulumiconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"gopkg.in/yaml.v3"
)

// Loader handles configuration loading and validation
type Loader struct {
	configPath string
	config     *ClusterConfig
	overrides  map[string]interface{}
	validators []Validator
}

// Validator interface for config validation
type Validator interface {
	Validate(config *ClusterConfig) error
}

// NewLoader creates a new configuration loader
func NewLoader(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
		overrides:  make(map[string]interface{}),
		validators: []Validator{},
	}
}

// Load loads the configuration from file
func (l *Loader) Load() (*ClusterConfig, error) {
	// Check if config file exists
	if _, err := os.Stat(l.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", l.configPath)
	}

	// Read file content
	data, err := ioutil.ReadFile(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Determine file format and parse
	config := &ClusterConfig{}
	ext := strings.ToLower(filepath.Ext(l.configPath))

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON configuration: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported configuration format: %s", ext)
	}

	// Apply environment variable overrides
	if err := l.applyEnvironmentOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Apply explicit overrides
	if err := l.applyOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply overrides: %w", err)
	}

	// Set defaults
	if err := l.setDefaults(config); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	// Validate configuration
	if err := l.validate(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	l.config = config
	return config, nil
}

// LoadFromPulumiConfig loads configuration from Pulumi config
func (l *Loader) LoadFromPulumiConfig(ctx *pulumi.Context) (*ClusterConfig, error) {
	cfg := pulumiconfig.New(ctx, "")

	// Load base config from file if specified
	configFile := cfg.Get("configFile")
	if configFile != "" {
		l.configPath = configFile
		baseConfig, err := l.Load()
		if err != nil {
			return nil, err
		}
		l.config = baseConfig
	} else {
		// Initialize with empty config
		l.config = &ClusterConfig{}
		l.setDefaults(l.config)
	}

	// Override with Pulumi config values
	l.applyPulumiOverrides(ctx, l.config)

	// Validate final configuration
	if err := l.validate(l.config); err != nil {
		return nil, err
	}

	return l.config, nil
}

// SetOverride sets a configuration override
func (l *Loader) SetOverride(key string, value interface{}) {
	l.overrides[key] = value
}

// AddValidator adds a configuration validator
func (l *Loader) AddValidator(v Validator) {
	l.validators = append(l.validators, v)
}

// GetConfig returns the loaded configuration
func (l *Loader) GetConfig() *ClusterConfig {
	return l.config
}

// SaveConfig saves the configuration to a file
func (l *Loader) SaveConfig(path string) error {
	if l.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Marshal based on file extension
	ext := strings.ToLower(filepath.Ext(path))
	var data []byte
	var err error

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(l.config)
	case ".json":
		data, err = json.MarshalIndent(l.config, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration: %w", err)
	}

	return nil
}

// applyEnvironmentOverrides applies environment variable overrides
func (l *Loader) applyEnvironmentOverrides(config *ClusterConfig) error {
	// Check for environment variables with prefix CLUSTER_
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "CLUSTER_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.ToLower(strings.TrimPrefix(parts[0], "CLUSTER_"))
			key = strings.ReplaceAll(key, "_", ".")
			value := parts[1]

			// Apply override based on key path
			if err := l.setConfigValue(config, key, value); err != nil {
				return fmt.Errorf("failed to apply environment override %s: %w", key, err)
			}
		}
	}

	return nil
}

// applyOverrides applies explicit overrides
func (l *Loader) applyOverrides(config *ClusterConfig) error {
	for key, value := range l.overrides {
		if err := l.setConfigValue(config, key, value); err != nil {
			return fmt.Errorf("failed to apply override %s: %w", key, err)
		}
	}
	return nil
}

// applyPulumiOverrides applies Pulumi config overrides
func (l *Loader) applyPulumiOverrides(ctx *pulumi.Context, clusterCfg *ClusterConfig) {
	cfg := pulumiconfig.New(ctx, "")

	// Apply common overrides
	if val := cfg.Get("clusterName"); val != "" {
		clusterCfg.Metadata.Name = val
	}
	if val := cfg.Get("environment"); val != "" {
		clusterCfg.Metadata.Environment = val
	}
	if val := cfg.Get("region"); val != "" {
		// Apply to default provider
		if clusterCfg.Providers.DigitalOcean != nil {
			clusterCfg.Providers.DigitalOcean.Region = val
		}
		if clusterCfg.Providers.Linode != nil {
			clusterCfg.Providers.Linode.Region = val
		}
	}

	// WireGuard specific overrides
	if val := cfg.Get("wireguardServerEndpoint"); val != "" {
		if clusterCfg.Network.WireGuard == nil {
			clusterCfg.Network.WireGuard = &WireGuardConfig{}
		}
		clusterCfg.Network.WireGuard.ServerEndpoint = val
		clusterCfg.Network.WireGuard.Enabled = true
	}
	if val := cfg.Get("wireguardServerPublicKey"); val != "" {
		if clusterCfg.Network.WireGuard == nil {
			clusterCfg.Network.WireGuard = &WireGuardConfig{}
		}
		clusterCfg.Network.WireGuard.ServerPublicKey = val
	}
}

// setConfigValue sets a value in the config using dot notation path
func (l *Loader) setConfigValue(config *ClusterConfig, path string, value interface{}) error {
	// This is a simplified implementation
	// In production, you would use reflection to navigate the struct
	parts := strings.Split(path, ".")

	switch parts[0] {
	case "metadata":
		if len(parts) > 1 {
			switch parts[1] {
			case "name":
				config.Metadata.Name = fmt.Sprintf("%v", value)
			case "environment":
				config.Metadata.Environment = fmt.Sprintf("%v", value)
			case "owner":
				config.Metadata.Owner = fmt.Sprintf("%v", value)
			case "team":
				config.Metadata.Team = fmt.Sprintf("%v", value)
			}
		}
	case "cluster":
		if len(parts) > 1 {
			switch parts[1] {
			case "type":
				config.Cluster.Type = fmt.Sprintf("%v", value)
			case "version":
				config.Cluster.Version = fmt.Sprintf("%v", value)
			case "highavailability":
				config.Cluster.HighAvailability = value.(bool)
			}
		}
	}

	return nil
}

// setDefaults sets default values for the configuration
func (l *Loader) setDefaults(config *ClusterConfig) error {
	// Set metadata defaults
	if config.Metadata.Name == "" {
		config.Metadata.Name = "kubernetes-cluster"
	}
	if config.Metadata.Environment == "" {
		config.Metadata.Environment = "development"
	}
	if config.Metadata.Version == "" {
		config.Metadata.Version = "1.0.0"
	}

	// Set cluster defaults
	if config.Cluster.Type == "" {
		config.Cluster.Type = "rke"
	}
	if config.Cluster.Version == "" {
		config.Cluster.Version = "v1.28.7-rancher1-1"
	}

	// Set network defaults
	if config.Network.Mode == "" {
		config.Network.Mode = "vpc"
	}
	if config.Network.CIDR == "" {
		config.Network.CIDR = "10.0.0.0/16"
	}

	// Set Kubernetes defaults
	if config.Kubernetes.NetworkPlugin == "" {
		config.Kubernetes.NetworkPlugin = "canal"
	}
	if config.Kubernetes.PodCIDR == "" {
		config.Kubernetes.PodCIDR = "10.42.0.0/16"
	}
	if config.Kubernetes.ServiceCIDR == "" {
		config.Kubernetes.ServiceCIDR = "10.43.0.0/16"
	}
	if config.Kubernetes.ClusterDNS == "" {
		config.Kubernetes.ClusterDNS = "10.43.0.10"
	}
	if config.Kubernetes.ClusterDomain == "" {
		config.Kubernetes.ClusterDomain = "cluster.local"
	}

	// Set security defaults
	if config.Security.SSHConfig.Port == 0 {
		config.Security.SSHConfig.Port = 22
	}

	// Set WireGuard defaults if enabled
	if config.Network.WireGuard != nil && config.Network.WireGuard.Enabled {
		if config.Network.WireGuard.Port == 0 {
			config.Network.WireGuard.Port = 51820
		}
		if config.Network.WireGuard.PersistentKeepalive == 0 {
			config.Network.WireGuard.PersistentKeepalive = 25
		}
		if config.Network.WireGuard.MTU == 0 {
			config.Network.WireGuard.MTU = 1420
		}
		if len(config.Network.WireGuard.DNS) == 0 {
			config.Network.WireGuard.DNS = []string{"1.1.1.1", "8.8.8.8"}
		}
		if len(config.Network.WireGuard.AllowedIPs) == 0 {
			config.Network.WireGuard.AllowedIPs = []string{"10.0.0.0/8", "172.16.0.0/12"}
		}
	}

	return nil
}

// validate validates the configuration
func (l *Loader) validate(config *ClusterConfig) error {
	// Basic validation
	if config.Metadata.Name == "" {
		return fmt.Errorf("cluster name is required")
	}

	// Check that at least one provider is enabled
	hasProvider := false
	if config.Providers.DigitalOcean != nil && config.Providers.DigitalOcean.Enabled {
		hasProvider = true
	}
	if config.Providers.Linode != nil && config.Providers.Linode.Enabled {
		hasProvider = true
	}
	if config.Providers.AWS != nil && config.Providers.AWS.Enabled {
		hasProvider = true
	}
	if config.Providers.Azure != nil && config.Providers.Azure.Enabled {
		hasProvider = true
	}
	if config.Providers.GCP != nil && config.Providers.GCP.Enabled {
		hasProvider = true
	}

	if !hasProvider {
		return fmt.Errorf("at least one cloud provider must be enabled")
	}

	// Validate node configuration
	if len(config.Nodes) == 0 && len(config.NodePools) == 0 {
		return fmt.Errorf("at least one node or node pool must be configured")
	}

	// Check for master nodes
	hasMaster := false
	for _, node := range config.Nodes {
		for _, role := range node.Roles {
			if role == "controlplane" || role == "master" {
				hasMaster = true
				break
			}
		}
	}
	for _, pool := range config.NodePools {
		for _, role := range pool.Roles {
			if role == "controlplane" || role == "master" {
				hasMaster = true
				break
			}
		}
	}

	if !hasMaster {
		return fmt.Errorf("at least one control plane node is required")
	}

	// Run custom validators
	for _, validator := range l.validators {
		if err := validator.Validate(config); err != nil {
			return err
		}
	}

	return nil
}

// MergeConfigs merges multiple configurations
func MergeConfigs(configs ...*ClusterConfig) (*ClusterConfig, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("no configurations to merge")
	}

	// Start with the first config as base
	result := configs[0]

	// Merge subsequent configs
	for i := 1; i < len(configs); i++ {
		if err := mergeConfig(result, configs[i]); err != nil {
			return nil, fmt.Errorf("failed to merge configuration %d: %w", i, err)
		}
	}

	return result, nil
}

// mergeConfig merges source into target
func mergeConfig(target, source *ClusterConfig) error {
	// This is a simplified merge implementation
	// In production, you would use a library like mergo for deep merging

	// Merge metadata
	if source.Metadata.Name != "" {
		target.Metadata.Name = source.Metadata.Name
	}
	if source.Metadata.Environment != "" {
		target.Metadata.Environment = source.Metadata.Environment
	}

	// Merge nodes
	target.Nodes = append(target.Nodes, source.Nodes...)

	// Merge node pools
	for k, v := range source.NodePools {
		target.NodePools[k] = v
	}

	return nil
}