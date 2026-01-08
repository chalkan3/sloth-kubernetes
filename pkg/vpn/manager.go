package vpn

import (
	"context"
	"fmt"
	"time"
)

// Manager provides a high-level API for VPN operations
type Manager struct {
	connMgr      *ConnectionManager
	healthChk    *HealthChecker
	configMgr    *ConfigManager // Legacy WireGuard config manager
	provider     VPNProvider    // VPN provider (WireGuard or Tailscale)
	peerRegistry *PeerRegistry
	retryPolicy  *RetryPolicy
	providerType ProviderType
}

// ManagerConfig holds configuration for the VPN manager
type ManagerConfig struct {
	SSHKeyPath     string
	DataDir        string        // For peer registry persistence
	RetryPolicy    *RetryPolicy  // Optional custom retry policy
	ConnectTimeout time.Duration // SSH connection timeout
	ProviderType   ProviderType  // VPN provider type (default: wireguard)
	ProviderConfig interface{}   // Provider-specific configuration
}

// NewManager creates a new VPN Manager
func NewManager(cfg ManagerConfig) (*Manager, error) {
	if cfg.SSHKeyPath == "" {
		return nil, fmt.Errorf("SSH key path is required")
	}

	retryPolicy := cfg.RetryPolicy
	if retryPolicy == nil {
		retryPolicy = NewDefaultRetryPolicy()
	}

	timeout := cfg.ConnectTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	healthChecker := NewHealthChecker(timeout)

	connMgr, err := NewConnectionManager(cfg.SSHKeyPath, retryPolicy, healthChecker)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	peerRegistry, err := NewPeerRegistry(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer registry: %w", err)
	}

	// Default to WireGuard if not specified
	providerType := cfg.ProviderType
	if providerType == "" {
		providerType = ProviderWireGuard
	}

	// Create VPN provider if registered
	var provider VPNProvider
	provider, err = NewProvider(providerType, cfg.ProviderConfig)
	if err != nil {
		// Fall back to legacy config manager for backward compatibility
		provider = nil
	}

	// Legacy WireGuard config manager (for backward compatibility)
	configMgr := NewConfigManager(connMgr)

	return &Manager{
		connMgr:      connMgr,
		healthChk:    healthChecker,
		configMgr:    configMgr,
		provider:     provider,
		peerRegistry: peerRegistry,
		retryPolicy:  retryPolicy,
		providerType: providerType,
	}, nil
}

// NewManagerWithProvider creates a Manager with a specific provider
func NewManagerWithProvider(cfg ManagerConfig, provider VPNProvider) (*Manager, error) {
	if cfg.SSHKeyPath == "" {
		return nil, fmt.Errorf("SSH key path is required")
	}

	retryPolicy := cfg.RetryPolicy
	if retryPolicy == nil {
		retryPolicy = NewDefaultRetryPolicy()
	}

	timeout := cfg.ConnectTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	healthChecker := NewHealthChecker(timeout)

	connMgr, err := NewConnectionManager(cfg.SSHKeyPath, retryPolicy, healthChecker)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	peerRegistry, err := NewPeerRegistry(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer registry: %w", err)
	}

	// Legacy WireGuard config manager
	configMgr := NewConfigManager(connMgr)

	providerType := ProviderWireGuard
	if provider != nil {
		providerType = provider.Type()
	}

	return &Manager{
		connMgr:      connMgr,
		healthChk:    healthChecker,
		configMgr:    configMgr,
		provider:     provider,
		peerRegistry: peerRegistry,
		retryPolicy:  retryPolicy,
		providerType: providerType,
	}, nil
}

// JoinConfig holds configuration for joining the VPN
type JoinConfig struct {
	StackName   string
	VPNIP       string     // Desired VPN IP (empty for auto-assign)
	Label       string     // Human-readable label
	PublicKey   string     // WireGuard public key
	Nodes       []NodeInfo // Cluster nodes to add peer to
	BastionIP   string     // Bastion host IP (if using bastion)
	BastionUser string     // Bastion SSH user
	SubnetCIDR  string     // VPN subnet for IP assignment (e.g., "10.8.0.0/24")
	ReservedIPs []string   // IPs reserved for cluster nodes
}

// JoinResult contains the result of a join operation
type JoinResult struct {
	VPNIP           string
	PublicKey       string
	NodesConfigured int
	NodesFailed     int
	Duration        time.Duration
	Errors          []string
}

// Join adds a peer to the VPN mesh
func (m *Manager) Join(ctx context.Context, cfg JoinConfig) (*JoinResult, error) {
	startTime := time.Now()
	result := &JoinResult{
		PublicKey: cfg.PublicKey,
	}

	// Auto-assign IP if not specified
	vpnIP := cfg.VPNIP
	if vpnIP == "" {
		// Check if peer with this label already exists (stable IP)
		if cfg.Label != "" {
			if existing, err := m.peerRegistry.GetByLabel(cfg.StackName, cfg.Label); err == nil {
				vpnIP = existing.VPNIP
			}
		}

		// If still no IP, assign next available
		if vpnIP == "" {
			var err error
			vpnIP, err = m.peerRegistry.NextAvailableIP(cfg.StackName, cfg.SubnetCIDR, cfg.ReservedIPs)
			if err != nil {
				return nil, fmt.Errorf("failed to assign VPN IP: %w", err)
			}
		}
	}
	result.VPNIP = vpnIP

	// Pre-check: verify bastion is reachable if configured
	if cfg.BastionIP != "" {
		if err := m.healthChk.CheckBastion(ctx, cfg.BastionIP); err != nil {
			return nil, fmt.Errorf("bastion unreachable: %w", err)
		}
	}

	// Build peer config
	peer := PeerConfig{
		PublicKey:  cfg.PublicKey,
		AllowedIPs: []string{vpnIP + "/32"},
		Keepalive:  25,
		Label:      cfg.Label,
	}

	// Add peer to all nodes
	for _, node := range cfg.Nodes {
		connCfg := ConnectionConfig{
			Host:        node.PublicIP,
			User:        getSSHUserForProvider(node.Provider),
			UseBastion:  cfg.BastionIP != "",
			BastionHost: cfg.BastionIP,
			BastionUser: cfg.BastionUser,
		}

		// Connect to node
		conn, err := m.connMgr.Connect(ctx, connCfg)
		if err != nil {
			result.NodesFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: connection failed: %v", node.Name, err))
			continue
		}

		// Add peer using provider if available, otherwise use legacy config manager
		if m.provider != nil {
			if err := m.provider.AddPeer(ctx, conn, peer); err != nil {
				result.NodesFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("%s: add peer failed: %v", node.Name, err))
				conn.Close()
				continue
			}
		} else {
			if err := m.configMgr.AddPeer(ctx, conn, peer); err != nil {
				result.NodesFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("%s: add peer failed: %v", node.Name, err))
				conn.Close()
				continue
			}
		}

		conn.Close()
		result.NodesConfigured++
	}

	// Register peer in registry
	registeredPeer := RegisteredPeer{
		PublicKey:  cfg.PublicKey,
		VPNIP:      vpnIP,
		Label:      cfg.Label,
		AllowedIPs: []string{vpnIP + "/32"},
	}

	if err := m.peerRegistry.Register(cfg.StackName, registeredPeer); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("registry: %v", err))
	}

	result.Duration = time.Since(startTime)

	if result.NodesConfigured == 0 {
		return result, fmt.Errorf("failed to configure any nodes")
	}

	return result, nil
}

// LeaveConfig holds configuration for leaving the VPN
type LeaveConfig struct {
	StackName   string
	PublicKey   string // Peer's public key to remove
	VPNIP       string // Alternative: identify by VPN IP
	Nodes       []NodeInfo
	BastionIP   string
	BastionUser string
}

// LeaveResult contains the result of a leave operation
type LeaveResult struct {
	NodesUpdated int
	NodesFailed  int
	Duration     time.Duration
	Errors       []string
}

// Leave removes a peer from the VPN mesh
func (m *Manager) Leave(ctx context.Context, cfg LeaveConfig) (*LeaveResult, error) {
	startTime := time.Now()
	result := &LeaveResult{}

	// Determine public key if only VPN IP is provided
	publicKey := cfg.PublicKey
	if publicKey == "" && cfg.VPNIP != "" {
		// Look up in registry
		if peer, err := m.peerRegistry.GetByIP(cfg.StackName, cfg.VPNIP); err == nil {
			publicKey = peer.PublicKey
		}
	}

	if publicKey == "" {
		return nil, fmt.Errorf("peer not found - specify public key or VPN IP")
	}

	// Remove peer from all nodes
	for _, node := range cfg.Nodes {
		connCfg := ConnectionConfig{
			Host:        node.PublicIP,
			User:        getSSHUserForProvider(node.Provider),
			UseBastion:  cfg.BastionIP != "",
			BastionHost: cfg.BastionIP,
			BastionUser: cfg.BastionUser,
		}

		conn, err := m.connMgr.Connect(ctx, connCfg)
		if err != nil {
			result.NodesFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: connection failed: %v", node.Name, err))
			continue
		}

		// Remove peer using provider if available, otherwise use legacy config manager
		if m.provider != nil {
			if err := m.provider.RemovePeer(ctx, conn, publicKey); err != nil {
				result.NodesFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("%s: remove peer failed: %v", node.Name, err))
				conn.Close()
				continue
			}
		} else {
			if err := m.configMgr.RemovePeer(ctx, conn, publicKey); err != nil {
				result.NodesFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("%s: remove peer failed: %v", node.Name, err))
				conn.Close()
				continue
			}
		}

		conn.Close()
		result.NodesUpdated++
	}

	// Remove from registry
	if err := m.peerRegistry.Unregister(cfg.StackName, publicKey); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("registry: %v", err))
	}

	result.Duration = time.Since(startTime)

	if result.NodesUpdated == 0 {
		return result, fmt.Errorf("failed to update any nodes")
	}

	return result, nil
}

// CheckHealth performs health checks on all nodes
func (m *Manager) CheckHealth(ctx context.Context, nodes []NodeInfo, bastionIP, bastionUser string) *MultiNodeHealthResult {
	baseCfg := ConnectionConfig{
		UseBastion:  bastionIP != "",
		BastionHost: bastionIP,
		BastionUser: bastionUser,
	}

	return m.healthChk.CheckMultipleNodes(ctx, nodes, m.connMgr, baseCfg)
}

// ListPeers returns all registered peers for a stack
func (m *Manager) ListPeers(stackName string) ([]RegisteredPeer, error) {
	return m.peerRegistry.List(stackName)
}

// GetPeerByLabel retrieves a peer by label
func (m *Manager) GetPeerByLabel(stackName, label string) (*RegisteredPeer, error) {
	return m.peerRegistry.GetByLabel(stackName, label)
}

// GetConnectionManager returns the underlying connection manager
func (m *Manager) GetConnectionManager() *ConnectionManager {
	return m.connMgr
}

// GetConfigManager returns the underlying config manager
func (m *Manager) GetConfigManager() *ConfigManager {
	return m.configMgr
}

// GetHealthChecker returns the underlying health checker
func (m *Manager) GetHealthChecker() *HealthChecker {
	return m.healthChk
}

// GetPeerRegistry returns the underlying peer registry
func (m *Manager) GetPeerRegistry() *PeerRegistry {
	return m.peerRegistry
}

// GetProvider returns the VPN provider
func (m *Manager) GetProvider() VPNProvider {
	return m.provider
}

// GetProviderType returns the provider type
func (m *Manager) GetProviderType() ProviderType {
	return m.providerType
}

// SetProvider sets the VPN provider
func (m *Manager) SetProvider(provider VPNProvider) {
	m.provider = provider
	if provider != nil {
		m.providerType = provider.Type()
	}
}

// GetVPNStatus gets the VPN status from a specific node
func (m *Manager) GetVPNStatus(ctx context.Context, node NodeInfo, bastionIP, bastionUser string) (*VPNStatus, error) {
	if m.provider == nil {
		return nil, fmt.Errorf("no VPN provider configured")
	}

	connCfg := ConnectionConfig{
		Host:        node.PublicIP,
		User:        getSSHUserForProvider(node.Provider),
		UseBastion:  bastionIP != "",
		BastionHost: bastionIP,
		BastionUser: bastionUser,
	}

	conn, err := m.connMgr.Connect(ctx, connCfg)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	return m.provider.GetStatus(ctx, conn)
}

// ListPeersFromNode lists all VPN peers from a specific node
func (m *Manager) ListPeersFromNode(ctx context.Context, node NodeInfo, bastionIP, bastionUser string) ([]PeerInfo, error) {
	if m.provider == nil {
		return nil, fmt.Errorf("no VPN provider configured")
	}

	connCfg := ConnectionConfig{
		Host:        node.PublicIP,
		User:        getSSHUserForProvider(node.Provider),
		UseBastion:  bastionIP != "",
		BastionHost: bastionIP,
		BastionUser: bastionUser,
	}

	conn, err := m.connMgr.Connect(ctx, connCfg)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	return m.provider.ListPeers(ctx, conn)
}

// getSSHUserForProvider returns the SSH user for a cloud provider
func getSSHUserForProvider(provider string) string {
	switch provider {
	case "azure":
		return "azureuser"
	case "aws":
		return "ubuntu"
	case "gcp":
		return "ubuntu"
	default:
		return "root"
	}
}

// ExecuteOnNode executes a command on a specific node
func (m *Manager) ExecuteOnNode(ctx context.Context, node NodeInfo, bastionIP, bastionUser, cmd string) (string, error) {
	connCfg := ConnectionConfig{
		Host:        node.PublicIP,
		User:        getSSHUserForProvider(node.Provider),
		UseBastion:  bastionIP != "",
		BastionHost: bastionIP,
		BastionUser: bastionUser,
	}

	conn, err := m.connMgr.Connect(ctx, connCfg)
	if err != nil {
		return "", fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	return conn.Execute(cmd)
}

// ExecuteOnAllNodes executes a command on all nodes
func (m *Manager) ExecuteOnAllNodes(ctx context.Context, nodes []NodeInfo, bastionIP, bastionUser, cmd string) map[string]string {
	results := make(map[string]string)

	for _, node := range nodes {
		output, err := m.ExecuteOnNode(ctx, node, bastionIP, bastionUser, cmd)
		if err != nil {
			results[node.Name] = fmt.Sprintf("ERROR: %v", err)
		} else {
			results[node.Name] = output
		}
	}

	return results
}
