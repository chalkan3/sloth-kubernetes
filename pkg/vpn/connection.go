package vpn

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// ConnectionConfig holds SSH connection parameters
type ConnectionConfig struct {
	Host        string        // Target host IP or hostname
	User        string        // SSH user (default: root)
	Port        int           // SSH port (default: 22)
	Timeout     time.Duration // Connection timeout (default: 30s)
	UseBastion  bool          // Whether to connect through bastion
	BastionHost string        // Bastion host IP
	BastionUser string        // Bastion SSH user (default: root)
	BastionPort int           // Bastion SSH port (default: 22)
}

// SSHConnection represents an established SSH connection
type SSHConnection struct {
	client      *ssh.Client
	bastionConn *ssh.Client
	host        string
	user        string
	connectedAt time.Time
	mu          sync.Mutex
	closed      bool
}

// ConnectionManager manages SSH connections with retry and health checking
type ConnectionManager struct {
	sshKeyPath    string
	retryPolicy   *RetryPolicy
	healthChecker *HealthChecker
	signer        ssh.Signer
}

// NewConnectionManager creates a new ConnectionManager
func NewConnectionManager(sshKeyPath string, retryPolicy *RetryPolicy, healthChecker *HealthChecker) (*ConnectionManager, error) {
	// Load SSH key
	keyData, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key: %w", err)
	}

	if retryPolicy == nil {
		retryPolicy = NewDefaultRetryPolicy()
	}

	if healthChecker == nil {
		healthChecker = NewHealthChecker(30 * time.Second)
	}

	return &ConnectionManager{
		sshKeyPath:    sshKeyPath,
		retryPolicy:   retryPolicy,
		healthChecker: healthChecker,
		signer:        signer,
	}, nil
}

// Connect establishes an SSH connection with retry logic
func (c *ConnectionManager) Connect(ctx context.Context, cfg ConnectionConfig) (*SSHConnection, error) {
	// Apply defaults
	if cfg.User == "" {
		cfg.User = "root"
	}
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.UseBastion && cfg.BastionUser == "" {
		cfg.BastionUser = "root"
	}
	if cfg.UseBastion && cfg.BastionPort == 0 {
		cfg.BastionPort = 22
	}

	// Health check bastion first if using it
	if cfg.UseBastion {
		if err := c.healthChecker.CheckBastion(ctx, cfg.BastionHost); err != nil {
			return nil, fmt.Errorf("bastion health check failed: %w", err)
		}
	}

	// Connect with retry
	result, err := c.retryPolicy.Execute(ctx, func() (any, error) {
		return c.dial(ctx, cfg)
	})

	if err != nil {
		return nil, err
	}

	return result.(*SSHConnection), nil
}

// dial performs the actual SSH connection
func (c *ConnectionManager) dial(ctx context.Context, cfg ConnectionConfig) (*SSHConnection, error) {
	sshConfig := &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(c.signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         cfg.Timeout,
	}

	conn := &SSHConnection{
		host:        cfg.Host,
		user:        cfg.User,
		connectedAt: time.Now(),
	}

	var targetConn net.Conn
	var err error

	if cfg.UseBastion {
		// Connect to bastion first
		bastionConfig := &ssh.ClientConfig{
			User: cfg.BastionUser,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(c.signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         cfg.Timeout,
		}

		bastionAddr := fmt.Sprintf("%s:%d", cfg.BastionHost, cfg.BastionPort)
		conn.bastionConn, err = ssh.Dial("tcp", bastionAddr, bastionConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to bastion %s: %w", bastionAddr, err)
		}

		// Connect to target through bastion
		targetAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		targetConn, err = conn.bastionConn.Dial("tcp", targetAddr)
		if err != nil {
			conn.bastionConn.Close()
			return nil, fmt.Errorf("failed to dial target %s through bastion: %w", targetAddr, err)
		}

		// Establish SSH connection over the tunneled connection
		ncc, chans, reqs, err := ssh.NewClientConn(targetConn, targetAddr, sshConfig)
		if err != nil {
			targetConn.Close()
			conn.bastionConn.Close()
			return nil, fmt.Errorf("failed to establish SSH to target %s: %w", targetAddr, err)
		}

		conn.client = ssh.NewClient(ncc, chans, reqs)
	} else {
		// Direct connection
		targetAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		conn.client, err = ssh.Dial("tcp", targetAddr, sshConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s: %w", targetAddr, err)
		}
	}

	return conn, nil
}

// Host returns the connected host
func (c *SSHConnection) Host() string {
	return c.host
}

// User returns the connected user
func (c *SSHConnection) User() string {
	return c.user
}

// ConnectedAt returns when the connection was established
func (c *SSHConnection) ConnectedAt() time.Time {
	return c.connectedAt
}

// Execute runs a command on the remote host and returns the output
func (c *SSHConnection) Execute(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return "", fmt.Errorf("connection is closed")
	}

	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		// Include stderr in error for debugging
		if stderr.Len() > 0 {
			return stdout.String(), fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
		}
		return stdout.String(), fmt.Errorf("command failed: %w", err)
	}

	return stdout.String(), nil
}

// ExecuteWithStdin runs a command with stdin input
func (c *SSHConnection) ExecuteWithStdin(cmd string, stdin io.Reader) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return "", fmt.Errorf("connection is closed")
	}

	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	session.Stdin = stdin

	if err := session.Run(cmd); err != nil {
		if stderr.Len() > 0 {
			return stdout.String(), fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
		}
		return stdout.String(), fmt.Errorf("command failed: %w", err)
	}

	return stdout.String(), nil
}

// ExecuteScript runs a multi-line script on the remote host
func (c *SSHConnection) ExecuteScript(script string) (string, error) {
	// Wrap script in bash heredoc for safe execution
	cmd := fmt.Sprintf("bash -s << 'EOFSCRIPT'\n%s\nEOFSCRIPT", script)
	return c.Execute(cmd)
}

// Close closes the SSH connection
func (c *SSHConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	var errs []string

	if c.client != nil {
		if err := c.client.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("client close: %v", err))
		}
	}

	if c.bastionConn != nil {
		if err := c.bastionConn.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("bastion close: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// IsAlive checks if the connection is still alive
func (c *SSHConnection) IsAlive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false
	}

	// Try to create a session to verify connection is alive
	session, err := c.client.NewSession()
	if err != nil {
		return false
	}
	session.Close()

	return true
}

// ExecuteWithRetry executes a command with retry logic
func (m *ConnectionManager) ExecuteWithRetry(ctx context.Context, conn *SSHConnection, cmd string) (string, error) {
	result, err := m.retryPolicy.Execute(ctx, func() (any, error) {
		return conn.Execute(cmd)
	})

	if err != nil {
		return "", err
	}

	return result.(string), nil
}

// ConnectAndExecute connects to a host, executes a command, and closes the connection
func (m *ConnectionManager) ConnectAndExecute(ctx context.Context, cfg ConnectionConfig, cmd string) (string, error) {
	conn, err := m.Connect(ctx, cfg)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	return conn.Execute(cmd)
}
