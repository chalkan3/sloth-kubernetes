package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chalkan3/sloth-kubernetes/pkg/vpn"
)

func init() {
	vpn.RegisterProvider(vpn.ProviderTailscale, func(config interface{}) vpn.VPNProvider {
		return NewProvider(config)
	})
}

// TailscaleConfig holds Tailscale/Headscale-specific configuration
type TailscaleConfig struct {
	HeadscaleURL string   // Headscale coordination server URL
	AuthKey      string   // Pre-auth key for automatic registration
	Namespace    string   // Headscale namespace/tailnet
	Tags         []string // ACL tags to apply to nodes
	AcceptRoutes bool     // Accept routes advertised by other nodes
	ExitNode     string   // Optional exit node
	Hostname     string   // Override hostname in tailnet
	APIKey       string   // Headscale API key for management operations
}

// DefaultTailscaleConfig returns default Tailscale configuration
func DefaultTailscaleConfig() *TailscaleConfig {
	return &TailscaleConfig{
		AcceptRoutes: true,
		Namespace:    "default",
	}
}

// Provider implements the VPNProvider interface for Tailscale
type Provider struct {
	config           *TailscaleConfig
	headscaleManager *HeadscaleManager
}

// NewProvider creates a new Tailscale provider
func NewProvider(cfg interface{}) *Provider {
	var config *TailscaleConfig
	if cfg != nil {
		if c, ok := cfg.(*TailscaleConfig); ok {
			config = c
		}
	}
	if config == nil {
		config = DefaultTailscaleConfig()
	}

	provider := &Provider{
		config: config,
	}

	// Initialize Headscale manager if API key is provided
	if config.HeadscaleURL != "" && config.APIKey != "" {
		provider.headscaleManager = NewHeadscaleManager(HeadscaleConfig{
			APIURL:    config.HeadscaleURL,
			APIKey:    config.APIKey,
			Namespace: config.Namespace,
		})
	}

	return provider
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "Tailscale (Headscale)"
}

// Type returns the provider type
func (p *Provider) Type() vpn.ProviderType {
	return vpn.ProviderTailscale
}

// Install installs Tailscale on a node
func (p *Provider) Install(ctx context.Context, conn *vpn.SSHConnection) error {
	script := `#!/bin/bash
set -e

# Check if already installed
if command -v tailscale &> /dev/null; then
    echo "TAILSCALE_ALREADY_INSTALLED"
    exit 0
fi

# Install Tailscale using official script
curl -fsSL https://tailscale.com/install.sh | sh

# Enable and start tailscaled
sudo systemctl enable tailscaled
sudo systemctl start tailscaled

# Wait for daemon to be ready
sleep 2

# Verify installation
if tailscale status &> /dev/null || tailscale status 2>&1 | grep -q "Logged out"; then
    echo "TAILSCALE_INSTALLED"
else
    echo "TAILSCALE_INSTALL_FAILED"
    exit 1
fi
`
	output, err := conn.ExecuteScript(script)
	if err != nil {
		return fmt.Errorf("failed to install Tailscale: %w", err)
	}

	if strings.Contains(output, "TAILSCALE_INSTALL_FAILED") {
		return fmt.Errorf("Tailscale installation verification failed")
	}

	return nil
}

// Configure joins the node to the tailnet via Headscale
func (p *Provider) Configure(ctx context.Context, conn *vpn.SSHConnection, config interface{}) error {
	cfg := p.config
	if config != nil {
		if c, ok := config.(*TailscaleConfig); ok {
			cfg = c
		}
	}

	if cfg.HeadscaleURL == "" {
		return fmt.Errorf("HeadscaleURL is required")
	}

	if cfg.AuthKey == "" {
		return fmt.Errorf("AuthKey is required for automatic registration")
	}

	// Build tailscale up command
	cmd := fmt.Sprintf("sudo tailscale up --login-server=%s --authkey=%s",
		cfg.HeadscaleURL, cfg.AuthKey)

	if cfg.Hostname != "" {
		cmd += fmt.Sprintf(" --hostname=%s", cfg.Hostname)
	}

	if len(cfg.Tags) > 0 {
		cmd += fmt.Sprintf(" --advertise-tags=%s", strings.Join(cfg.Tags, ","))
	}

	if cfg.AcceptRoutes {
		cmd += " --accept-routes"
	}

	if cfg.ExitNode != "" {
		cmd += fmt.Sprintf(" --exit-node=%s", cfg.ExitNode)
	}

	// Add timeout and reset flag for clean configuration
	cmd += " --timeout=60s --reset"

	output, err := conn.Execute(cmd)
	if err != nil {
		return fmt.Errorf("tailscale up failed: %w (output: %s)", err, output)
	}

	// Verify connection
	verifyCmd := "tailscale status --json"
	statusOutput, err := conn.Execute(verifyCmd)
	if err != nil {
		return fmt.Errorf("failed to verify Tailscale status: %w", err)
	}

	var status tailscaleStatusJSON
	if err := json.Unmarshal([]byte(statusOutput), &status); err != nil {
		return fmt.Errorf("failed to parse Tailscale status: %w", err)
	}

	if status.BackendState != "Running" {
		return fmt.Errorf("Tailscale not in running state: %s", status.BackendState)
	}

	return nil
}

// AddPeer - Tailscale manages peers via the coordination server
// This is effectively a no-op since peers are auto-discovered
func (p *Provider) AddPeer(ctx context.Context, conn *vpn.SSHConnection, peer vpn.PeerConfig) error {
	// Tailscale handles peer discovery automatically through the coordination server
	// No manual peer addition is needed
	return nil
}

// RemovePeer removes a node from the tailnet via Headscale API
func (p *Provider) RemovePeer(ctx context.Context, conn *vpn.SSHConnection, peerID string) error {
	if p.headscaleManager == nil {
		return fmt.Errorf("Headscale API not configured - cannot remove peer")
	}

	// Delete node from Headscale
	return p.headscaleManager.DeleteNode(ctx, peerID)
}

// ListPeers returns all peers from tailscale status
func (p *Provider) ListPeers(ctx context.Context, conn *vpn.SSHConnection) ([]vpn.PeerInfo, error) {
	output, err := conn.Execute("tailscale status --json")
	if err != nil {
		return nil, fmt.Errorf("failed to get Tailscale status: %w", err)
	}

	var status tailscaleStatusJSON
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		return nil, fmt.Errorf("failed to parse Tailscale status: %w", err)
	}

	var peers []vpn.PeerInfo

	// Add self as a peer
	if status.Self != nil {
		self := vpn.PeerInfo{
			ID:           status.Self.ID,
			PublicKey:    status.Self.PublicKey,
			Hostname:     status.Self.HostName,
			Online:       status.Self.Online,
			ProviderType: vpn.ProviderTailscale,
		}
		if len(status.Self.TailscaleIPs) > 0 {
			self.VPNIP = status.Self.TailscaleIPs[0]
		}
		peers = append(peers, self)
	}

	// Add other peers
	for _, peer := range status.Peer {
		peerInfo := vpn.PeerInfo{
			ID:           peer.ID,
			PublicKey:    peer.PublicKey,
			Hostname:     peer.HostName,
			Online:       peer.Online,
			LastSeen:     peer.LastSeen,
			ProviderType: vpn.ProviderTailscale,
		}
		if len(peer.TailscaleIPs) > 0 {
			peerInfo.VPNIP = peer.TailscaleIPs[0]
		}
		if peer.CurAddr != "" {
			peerInfo.Endpoint = peer.CurAddr
		}

		// Parse transfer stats
		peerInfo.TransferRx = peer.RxBytes
		peerInfo.TransferTx = peer.TxBytes

		peers = append(peers, peerInfo)
	}

	return peers, nil
}

// GetStatus returns Tailscale connection status on a node
func (p *Provider) GetStatus(ctx context.Context, conn *vpn.SSHConnection) (*vpn.VPNStatus, error) {
	status := &vpn.VPNStatus{
		Provider:       vpn.ProviderTailscale,
		InterfaceName:  "tailscale0",
		CoordinatorURL: p.config.HeadscaleURL,
		Namespace:      p.config.Namespace,
	}

	output, err := conn.Execute("tailscale status --json")
	if err != nil {
		status.Connected = false
		status.Error = fmt.Sprintf("failed to get status: %v", err)
		return status, nil
	}

	var tsStatus tailscaleStatusJSON
	if err := json.Unmarshal([]byte(output), &tsStatus); err != nil {
		status.Connected = false
		status.Error = fmt.Sprintf("failed to parse status: %v", err)
		return status, nil
	}

	status.Connected = tsStatus.BackendState == "Running"

	if tsStatus.Self != nil {
		status.PublicKey = tsStatus.Self.PublicKey
		if len(tsStatus.Self.TailscaleIPs) > 0 {
			status.VPNIP = tsStatus.Self.TailscaleIPs[0]
		}
	}

	status.Peers = len(tsStatus.Peer)

	return status, nil
}

// IsHealthy checks if Tailscale is connected and healthy
func (p *Provider) IsHealthy(ctx context.Context, conn *vpn.SSHConnection) (bool, error) {
	output, err := conn.Execute("tailscale status --json 2>/dev/null")
	if err != nil {
		return false, err
	}

	var status tailscaleStatusJSON
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		return false, err
	}

	return status.BackendState == "Running", nil
}

// GenerateClientConfig generates instructions for joining the tailnet
// Tailscale doesn't use config files like WireGuard - it uses the CLI
func (p *Provider) GenerateClientConfig(params vpn.ClientConfigParams) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Tailscale Join Instructions\n")
	sb.WriteString("# ===========================\n\n")

	sb.WriteString("# 1. Install Tailscale:\n")
	sb.WriteString("curl -fsSL https://tailscale.com/install.sh | sh\n\n")

	sb.WriteString("# 2. Join the tailnet:\n")

	cmd := fmt.Sprintf("sudo tailscale up --login-server=%s", params.CoordinatorURL)

	if params.AuthKey != "" {
		cmd += fmt.Sprintf(" --authkey=%s", params.AuthKey)
	}

	if params.Hostname != "" {
		cmd += fmt.Sprintf(" --hostname=%s", params.Hostname)
	}

	if len(params.Tags) > 0 {
		cmd += fmt.Sprintf(" --advertise-tags=%s", strings.Join(params.Tags, ","))
	}

	sb.WriteString(cmd + "\n\n")

	sb.WriteString("# 3. Verify connection:\n")
	sb.WriteString("tailscale status\n\n")

	sb.WriteString("# Available cluster nodes:\n")
	for _, node := range params.NodeEndpoints {
		sb.WriteString(fmt.Sprintf("#   %s: %s\n", node.Name, node.VPNIP))
	}

	return sb.String(), nil
}

// RequiresCoordinator returns true for Tailscale (needs Headscale)
func (p *Provider) RequiresCoordinator() bool {
	return true
}

// GetInterfaceName returns the Tailscale interface name
func (p *Provider) GetInterfaceName() string {
	return "tailscale0"
}

// GetHeadscaleManager returns the Headscale manager for advanced operations
func (p *Provider) GetHeadscaleManager() *HeadscaleManager {
	return p.headscaleManager
}

// CreateAuthKey creates a new pre-auth key via Headscale API
func (p *Provider) CreateAuthKey(ctx context.Context, opts AuthKeyOptions) (string, error) {
	if p.headscaleManager == nil {
		return "", fmt.Errorf("Headscale API not configured")
	}
	return p.headscaleManager.CreateAuthKey(ctx, opts)
}

// Tailscale status JSON structures

type tailscaleStatusJSON struct {
	BackendState   string                         `json:"BackendState"`
	Self           *tailscalePeerStatus           `json:"Self"`
	Peer           map[string]tailscalePeerStatus `json:"Peer"`
	MagicDNSSuffix string                         `json:"MagicDNSSuffix"`
}

type tailscalePeerStatus struct {
	ID           string    `json:"ID"`
	PublicKey    string    `json:"PublicKey"`
	HostName     string    `json:"HostName"`
	DNSName      string    `json:"DNSName"`
	OS           string    `json:"OS"`
	TailscaleIPs []string  `json:"TailscaleIPs"`
	AllowedIPs   []string  `json:"AllowedIPs"`
	CurAddr      string    `json:"CurAddr"`
	Relay        string    `json:"Relay"`
	RxBytes      int64     `json:"RxBytes"`
	TxBytes      int64     `json:"TxBytes"`
	Created      time.Time `json:"Created"`
	LastSeen     time.Time `json:"LastSeen"`
	LastWrite    time.Time `json:"LastWrite"`
	Online       bool      `json:"Online"`
	Active       bool      `json:"Active"`
}
