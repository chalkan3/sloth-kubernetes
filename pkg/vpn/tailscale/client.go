package tailscale

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"tailscale.com/tsnet"
)

// EmbeddedClient provides an embedded Tailscale client for local machine connectivity
type EmbeddedClient struct {
	server        *tsnet.Server
	stateDir      string
	clusterName   string
	config        *EmbeddedClientConfig
	mu            sync.Mutex
	connected     bool
	proxyListener net.Listener
	proxyPort     int
}

// EmbeddedClientConfig holds configuration for the embedded client
type EmbeddedClientConfig struct {
	HeadscaleURL string   `json:"headscaleUrl"`
	AuthKey      string   `json:"authKey"`
	Hostname     string   `json:"hostname"`
	Tags         []string `json:"tags,omitempty"`
	StateDir     string   `json:"stateDir"`
}

// EmbeddedClientStatus represents the current status of the embedded client
type EmbeddedClientStatus struct {
	Connected    bool      `json:"connected"`
	Hostname     string    `json:"hostname"`
	TailscaleIP  string    `json:"tailscaleIp,omitempty"`
	ClusterName  string    `json:"clusterName"`
	HeadscaleURL string    `json:"headscaleUrl"`
	PeerCount    int       `json:"peerCount"`
	ConnectedAt  time.Time `json:"connectedAt,omitempty"`
}

// NewEmbeddedClient creates a new embedded Tailscale client
func NewEmbeddedClient(clusterName string, config *EmbeddedClientConfig) (*EmbeddedClient, error) {
	if config.HeadscaleURL == "" {
		return nil, fmt.Errorf("HeadscaleURL is required")
	}

	// Default state directory
	if config.StateDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		config.StateDir = filepath.Join(homeDir, ".sloth", "vpn", clusterName)
	}

	// Ensure state directory exists
	if err := os.MkdirAll(config.StateDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Default hostname
	if config.Hostname == "" {
		hostname, _ := os.Hostname()
		config.Hostname = fmt.Sprintf("sloth-client-%s", hostname)
	}

	return &EmbeddedClient{
		stateDir:    config.StateDir,
		clusterName: clusterName,
		config:      config,
	}, nil
}

// Connect joins the Tailscale mesh network
func (c *EmbeddedClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected && c.server != nil {
		return fmt.Errorf("already connected")
	}

	// Create tsnet server
	c.server = &tsnet.Server{
		Dir:          c.stateDir,
		Hostname:     c.config.Hostname,
		AuthKey:      c.config.AuthKey,
		ControlURL:   c.config.HeadscaleURL,
		Ephemeral:    false,
		RunWebClient: false,
	}

	// Start the server
	status, err := c.server.Up(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to tailnet: %w", err)
	}

	c.connected = true

	// Save connection state
	if err := c.saveState(); err != nil {
		// Non-fatal, just log
		fmt.Fprintf(os.Stderr, "Warning: failed to save connection state: %v\n", err)
	}

	// Log success
	if len(status.TailscaleIPs) > 0 {
		fmt.Printf("Connected to tailnet with IP: %s\n", status.TailscaleIPs[0])
	}

	return nil
}

// Disconnect leaves the Tailscale mesh network
func (c *EmbeddedClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.server == nil {
		return fmt.Errorf("not connected")
	}

	if err := c.server.Close(); err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	c.connected = false
	c.server = nil

	// Remove state file
	stateFile := filepath.Join(c.stateDir, "connection.json")
	os.Remove(stateFile)

	return nil
}

// Status returns the current connection status
func (c *EmbeddedClient) Status(ctx context.Context) (*EmbeddedClientStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	status := &EmbeddedClientStatus{
		Connected:    c.connected,
		ClusterName:  c.clusterName,
		HeadscaleURL: c.config.HeadscaleURL,
		Hostname:     c.config.Hostname,
	}

	if c.connected && c.server != nil {
		lc, err := c.server.LocalClient()
		if err == nil {
			tsStatus, err := lc.Status(ctx)
			if err == nil {
				if tsStatus.Self != nil && len(tsStatus.Self.TailscaleIPs) > 0 {
					status.TailscaleIP = tsStatus.Self.TailscaleIPs[0].String()
				}
				status.PeerCount = len(tsStatus.Peer)
			}
		}
	}

	return status, nil
}

// GetTailscaleIP returns the Tailscale IP of the local client
func (c *EmbeddedClient) GetTailscaleIP(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.server == nil {
		return "", fmt.Errorf("not connected")
	}

	lc, err := c.server.LocalClient()
	if err != nil {
		return "", fmt.Errorf("failed to get local client: %w", err)
	}

	status, err := lc.Status(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	if status.Self != nil && len(status.Self.TailscaleIPs) > 0 {
		return status.Self.TailscaleIPs[0].String(), nil
	}

	return "", fmt.Errorf("no Tailscale IP assigned")
}

// ListPeers returns all peers in the tailnet
func (c *EmbeddedClient) ListPeers(ctx context.Context) ([]PeerStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.server == nil {
		return nil, fmt.Errorf("not connected")
	}

	lc, err := c.server.LocalClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get local client: %w", err)
	}

	status, err := lc.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var peers []PeerStatus
	for _, peer := range status.Peer {
		p := PeerStatus{
			ID:       string(peer.ID),
			Hostname: peer.HostName,
			Online:   peer.Online,
			OS:       peer.OS,
			LastSeen: peer.LastSeen,
		}
		if len(peer.TailscaleIPs) > 0 {
			p.TailscaleIP = peer.TailscaleIPs[0].String()
		}
		peers = append(peers, p)
	}

	return peers, nil
}

// PeerStatus represents a peer in the tailnet
type PeerStatus struct {
	ID          string    `json:"id"`
	Hostname    string    `json:"hostname"`
	TailscaleIP string    `json:"tailscaleIp"`
	Online      bool      `json:"online"`
	OS          string    `json:"os"`
	LastSeen    time.Time `json:"lastSeen"`
}

// Dial creates a network connection to a peer
func (c *EmbeddedClient) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	c.mu.Lock()
	server := c.server
	connected := c.connected
	c.mu.Unlock()

	if !connected || server == nil {
		return nil, fmt.Errorf("not connected")
	}

	return server.Dial(ctx, network, addr)
}

// HTTPClient returns an HTTP client that routes through the tailnet
func (c *EmbeddedClient) HTTPClient() (*http.Client, error) {
	c.mu.Lock()
	server := c.server
	connected := c.connected
	c.mu.Unlock()

	if !connected || server == nil {
		return nil, fmt.Errorf("not connected")
	}

	return &http.Client{
		Transport: &http.Transport{
			DialContext: server.Dial,
		},
	}, nil
}

// saveState saves the connection state to disk
func (c *EmbeddedClient) saveState() error {
	state := struct {
		ClusterName  string    `json:"clusterName"`
		HeadscaleURL string    `json:"headscaleUrl"`
		Hostname     string    `json:"hostname"`
		ConnectedAt  time.Time `json:"connectedAt"`
	}{
		ClusterName:  c.clusterName,
		HeadscaleURL: c.config.HeadscaleURL,
		Hostname:     c.config.Hostname,
		ConnectedAt:  time.Now(),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	stateFile := filepath.Join(c.stateDir, "connection.json")
	return os.WriteFile(stateFile, data, 0600)
}

// LoadExistingClient attempts to load an existing client from saved state
func LoadExistingClient(clusterName string) (*EmbeddedClient, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".sloth", "vpn", clusterName)
	stateFile := filepath.Join(stateDir, "connection.json")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("no existing connection found: %w", err)
	}

	var state struct {
		ClusterName  string `json:"clusterName"`
		HeadscaleURL string `json:"headscaleUrl"`
		Hostname     string `json:"hostname"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state: %w", err)
	}

	return NewEmbeddedClient(clusterName, &EmbeddedClientConfig{
		HeadscaleURL: state.HeadscaleURL,
		Hostname:     state.Hostname,
		StateDir:     stateDir,
	})
}

// IsConnected checks if there's an active connection for the given cluster
func IsConnected(clusterName string) bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	stateDir := filepath.Join(homeDir, ".sloth", "vpn", clusterName)
	stateFile := filepath.Join(stateDir, "connection.json")

	_, err = os.Stat(stateFile)
	return err == nil
}

// CleanupState removes all state for a cluster connection
func CleanupState(clusterName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".sloth", "vpn", clusterName)
	return os.RemoveAll(stateDir)
}

// GetPIDFile returns the path to the PID file for daemon mode
func GetPIDFile(clusterName string) string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".sloth", "vpn", clusterName, "daemon.pid")
}

// GetDaemonPID returns the PID of the running daemon, or 0 if not running
func GetDaemonPID(clusterName string) int {
	pidFile := GetPIDFile(clusterName)
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}

	var pid int
	fmt.Sscanf(string(data), "%d", &pid)
	return pid
}

// IsDaemonRunning checks if the VPN daemon is running for the given cluster
func IsDaemonRunning(clusterName string) bool {
	pid := GetDaemonPID(clusterName)
	if pid == 0 {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0 to check
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// StartSOCKS5Proxy starts a SOCKS5 proxy server that routes through the tailnet
func (c *EmbeddedClient) StartSOCKS5Proxy(port int) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.server == nil {
		return 0, fmt.Errorf("not connected")
	}

	if c.proxyListener != nil {
		return c.proxyPort, nil // Already running
	}

	// Listen on localhost
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("failed to start SOCKS5 proxy: %w", err)
	}

	c.proxyListener = listener
	c.proxyPort = listener.Addr().(*net.TCPAddr).Port

	// Start accepting connections
	go c.runSOCKS5Proxy()

	return c.proxyPort, nil
}

// StopSOCKS5Proxy stops the SOCKS5 proxy server
func (c *EmbeddedClient) StopSOCKS5Proxy() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.proxyListener != nil {
		c.proxyListener.Close()
		c.proxyListener = nil
		c.proxyPort = 0
	}
}

// GetProxyPort returns the SOCKS5 proxy port if running
func (c *EmbeddedClient) GetProxyPort() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.proxyPort
}

// runSOCKS5Proxy accepts and handles SOCKS5 connections
func (c *EmbeddedClient) runSOCKS5Proxy() {
	for {
		conn, err := c.proxyListener.Accept()
		if err != nil {
			return // Listener closed
		}
		go c.handleSOCKS5Connection(conn)
	}
}

// handleSOCKS5Connection handles a single SOCKS5 connection
func (c *EmbeddedClient) handleSOCKS5Connection(conn net.Conn) {
	defer conn.Close()

	// Set read deadline for handshake
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Read version and auth methods
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return
	}

	if buf[0] != 0x05 { // SOCKS5
		return
	}

	nMethods := int(buf[1])
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return
	}

	// No auth required
	conn.Write([]byte{0x05, 0x00})

	// Read connect request
	buf = make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return
	}

	if buf[0] != 0x05 || buf[1] != 0x01 { // SOCKS5, CONNECT
		conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Command not supported
		return
	}

	// Parse address
	var destAddr string
	switch buf[3] {
	case 0x01: // IPv4
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return
		}
		destAddr = net.IP(addr).String()
	case 0x03: // Domain
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return
		}
		domain := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return
		}
		destAddr = string(domain)
	case 0x04: // IPv6
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return
		}
		destAddr = net.IP(addr).String()
	default:
		conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Address type not supported
		return
	}

	// Read port
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(portBuf)

	// Clear deadline for actual connection
	conn.SetReadDeadline(time.Time{})

	// Connect through tailnet
	target := fmt.Sprintf("%s:%d", destAddr, port)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c.mu.Lock()
	server := c.server
	c.mu.Unlock()

	if server == nil {
		conn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // General failure
		return
	}

	remote, err := server.Dial(ctx, "tcp", target)
	if err != nil {
		conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Connection refused
		return
	}
	defer remote.Close()

	// Success response
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 127, 0, 0, 1, 0, 0})

	// Bidirectional copy
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(remote, conn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(conn, remote)
		done <- struct{}{}
	}()
	<-done
}

// GetProxyFile returns the path to the proxy port file
func GetProxyFile(clusterName string) string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".sloth", "vpn", clusterName, "proxy.port")
}

// SaveProxyPort saves the proxy port to a file
func SaveProxyPort(clusterName string, port int) error {
	portFile := GetProxyFile(clusterName)
	return os.WriteFile(portFile, []byte(fmt.Sprintf("%d", port)), 0600)
}

// GetSavedProxyPort returns the saved proxy port for a cluster
func GetSavedProxyPort(clusterName string) int {
	portFile := GetProxyFile(clusterName)
	data, err := os.ReadFile(portFile)
	if err != nil {
		return 0
	}
	var port int
	fmt.Sscanf(string(data), "%d", &port)
	return port
}
