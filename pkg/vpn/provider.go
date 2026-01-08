package vpn

import (
	"context"
	"fmt"
	"time"
)

// ProviderType identifies the VPN provider
type ProviderType string

const (
	ProviderWireGuard ProviderType = "wireguard"
	ProviderTailscale ProviderType = "tailscale"
)

// VPNProvider defines the interface that all VPN providers must implement
type VPNProvider interface {
	// Name returns the provider name
	Name() string

	// Type returns the provider type
	Type() ProviderType

	// Install installs the VPN software on a node
	Install(ctx context.Context, conn *SSHConnection) error

	// Configure configures the VPN on a node with provider-specific settings
	Configure(ctx context.Context, conn *SSHConnection, config interface{}) error

	// AddPeer adds a peer to the VPN mesh
	// For self-managed VPNs (WireGuard), this adds the peer directly
	// For coordination-based VPNs (Tailscale), this may be a no-op
	AddPeer(ctx context.Context, conn *SSHConnection, peer PeerConfig) error

	// RemovePeer removes a peer from the VPN mesh
	RemovePeer(ctx context.Context, conn *SSHConnection, peerID string) error

	// ListPeers returns all peers visible from a node
	ListPeers(ctx context.Context, conn *SSHConnection) ([]PeerInfo, error)

	// GetStatus returns the VPN status on a node
	GetStatus(ctx context.Context, conn *SSHConnection) (*VPNStatus, error)

	// IsHealthy checks if the VPN interface is healthy on a node
	IsHealthy(ctx context.Context, conn *SSHConnection) (bool, error)

	// GenerateClientConfig generates client configuration for joining the VPN
	GenerateClientConfig(params ClientConfigParams) (string, error)

	// RequiresCoordinator returns true if the provider requires a coordination server
	// (e.g., Tailscale needs Headscale/tailscale.com)
	RequiresCoordinator() bool

	// GetInterfaceName returns the VPN interface name (e.g., "wg0", "tailscale0")
	GetInterfaceName() string
}

// PeerInfo contains information about a VPN peer
type PeerInfo struct {
	ID           string       `json:"id"`
	PublicKey    string       `json:"publicKey"`
	VPNIP        string       `json:"vpnIP"`
	Hostname     string       `json:"hostname"`
	Online       bool         `json:"online"`
	LastSeen     time.Time    `json:"lastSeen"`
	Endpoint     string       `json:"endpoint,omitempty"`
	TransferRx   int64        `json:"transferRx,omitempty"`
	TransferTx   int64        `json:"transferTx,omitempty"`
	ProviderType ProviderType `json:"providerType"`
}

// VPNStatus contains status information for a VPN connection
type VPNStatus struct {
	Provider  ProviderType  `json:"provider"`
	Connected bool          `json:"connected"`
	VPNIP     string        `json:"vpnIP"`
	PublicKey string        `json:"publicKey,omitempty"`
	Peers     int           `json:"peers"`
	Latency   time.Duration `json:"latency,omitempty"`
	Error     string        `json:"error,omitempty"`

	// Provider-specific fields
	CoordinatorURL string `json:"coordinatorUrl,omitempty"` // For Tailscale
	Namespace      string `json:"namespace,omitempty"`      // For Headscale
	InterfaceName  string `json:"interfaceName"`
}

// ClientConfigParams contains parameters for generating client configuration
type ClientConfigParams struct {
	VPNIP               string         // Client's VPN IP
	PrivateKey          string         // Client's private key (for WireGuard)
	PublicKey           string         // Client's public key
	NodeEndpoints       []NodeEndpoint // Cluster node endpoints
	DNS                 []string       // DNS servers
	MTU                 int            // MTU setting
	PersistentKeepalive int            // Keepalive interval

	// Tailscale-specific
	CoordinatorURL string   // Headscale URL
	AuthKey        string   // Pre-auth key
	Hostname       string   // Desired hostname in tailnet
	Tags           []string // ACL tags
}

// NodeEndpoint contains endpoint information for a cluster node
type NodeEndpoint struct {
	Name      string `json:"name"`
	PublicIP  string `json:"publicIP"`
	VPNIP     string `json:"vpnIP"`
	PublicKey string `json:"publicKey"`
	Port      int    `json:"port"`
}

// ProviderConstructor is a function that creates a VPN provider
type ProviderConstructor func(config interface{}) VPNProvider

// providerRegistry holds registered provider constructors
var providerRegistry = make(map[ProviderType]ProviderConstructor)

// RegisterProvider registers a provider constructor for a given type
// This should be called in init() of each provider package
func RegisterProvider(providerType ProviderType, constructor ProviderConstructor) {
	providerRegistry[providerType] = constructor
}

// NewProvider creates a provider of the specified type using registered constructors
func NewProvider(providerType ProviderType, config interface{}) (VPNProvider, error) {
	constructor, ok := providerRegistry[providerType]
	if !ok {
		return nil, fmt.Errorf("provider type '%s' not registered - import the provider package first", providerType)
	}
	return constructor(config), nil
}

// ProviderFactory creates VPN providers based on type
type ProviderFactory struct {
	wireGuardConfig interface{}
	tailscaleConfig interface{}
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// WithWireGuardConfig sets WireGuard configuration
func (f *ProviderFactory) WithWireGuardConfig(config interface{}) *ProviderFactory {
	f.wireGuardConfig = config
	return f
}

// WithTailscaleConfig sets Tailscale configuration
func (f *ProviderFactory) WithTailscaleConfig(config interface{}) *ProviderFactory {
	f.tailscaleConfig = config
	return f
}

// CreateProvider creates a provider of the specified type
func (f *ProviderFactory) CreateProvider(providerType ProviderType) (VPNProvider, error) {
	var config interface{}
	switch providerType {
	case ProviderWireGuard:
		config = f.wireGuardConfig
	case ProviderTailscale:
		config = f.tailscaleConfig
	}
	return NewProvider(providerType, config)
}

// ParseProviderType converts a string to ProviderType
func ParseProviderType(s string) (ProviderType, error) {
	switch s {
	case "wireguard", "wg":
		return ProviderWireGuard, nil
	case "tailscale", "ts":
		return ProviderTailscale, nil
	default:
		return "", fmt.Errorf("unknown provider type: %s", s)
	}
}
