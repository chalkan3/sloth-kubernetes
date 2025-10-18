package security

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"kubernetes-create/pkg/providers"
)

// OSFirewallManager manages operating system level firewall rules
type OSFirewallManager struct {
	ctx        *pulumi.Context
	nodes      []*providers.NodeOutput
	sshKeyPath string
	results    map[string]*FirewallResult
	mu         sync.RWMutex
	timeout    time.Duration
}

// FirewallResult represents the result of firewall configuration
type FirewallResult struct {
	NodeName      string
	Success       bool
	RulesApplied  []FirewallRule
	Error         error
	Timestamp     time.Time
	FirewallType  string // ufw, iptables, firewalld
}

// FirewallRule represents a single firewall rule
type FirewallRule struct {
	Port        string
	Protocol    string
	Source      string
	Direction   string // inbound/outbound
	Action      string // allow/deny
	Description string
}

// KubernetesFirewallPorts defines all required ports for Kubernetes components
var KubernetesFirewallPorts = struct {
	// Master/Control Plane Ports
	APIServer          FirewallRule
	ETCD               FirewallRule
	ETCDPeer           FirewallRule
	Scheduler          FirewallRule
	ControllerManager  FirewallRule
	Kubelet            FirewallRule
	KubeProxy          FirewallRule

	// Worker Ports
	NodePortServices   FirewallRule

	// Network Plugin Ports
	Flannel            FirewallRule
	Calico             FirewallRule
	CanalBGP           FirewallRule
	Weave              FirewallRule

	// Additional Ports
	SSH                FirewallRule
	WireGuard          FirewallRule
	DockerRegistry     FirewallRule
	MetricsServer      FirewallRule

	// Ingress Controller
	HTTPIngress        FirewallRule
	HTTPSIngress       FirewallRule
}{
	// Master/Control Plane Ports
	APIServer: FirewallRule{
		Port:        "6443",
		Protocol:    "tcp",
		Source:      "0.0.0.0/0",
		Direction:   "inbound",
		Action:      "allow",
		Description: "Kubernetes API Server",
	},
	ETCD: FirewallRule{
		Port:        "2379",
		Protocol:    "tcp",
		Source:      "10.0.0.0/8",
		Direction:   "inbound",
		Action:      "allow",
		Description: "etcd client API",
	},
	ETCDPeer: FirewallRule{
		Port:        "2380",
		Protocol:    "tcp",
		Source:      "10.0.0.0/8",
		Direction:   "inbound",
		Action:      "allow",
		Description: "etcd peer API",
	},
	Scheduler: FirewallRule{
		Port:        "10259",
		Protocol:    "tcp",
		Source:      "10.0.0.0/8",
		Direction:   "inbound",
		Action:      "allow",
		Description: "kube-scheduler",
	},
	ControllerManager: FirewallRule{
		Port:        "10257",
		Protocol:    "tcp",
		Source:      "10.0.0.0/8",
		Direction:   "inbound",
		Action:      "allow",
		Description: "kube-controller-manager",
	},
	Kubelet: FirewallRule{
		Port:        "10250",
		Protocol:    "tcp",
		Source:      "10.0.0.0/8",
		Direction:   "inbound",
		Action:      "allow",
		Description: "Kubelet API",
	},
	KubeProxy: FirewallRule{
		Port:        "10256",
		Protocol:    "tcp",
		Source:      "10.0.0.0/8",
		Direction:   "inbound",
		Action:      "allow",
		Description: "kube-proxy",
	},

	// Worker Ports
	NodePortServices: FirewallRule{
		Port:        "30000:32767",
		Protocol:    "tcp",
		Source:      "0.0.0.0/0",
		Direction:   "inbound",
		Action:      "allow",
		Description: "NodePort Services",
	},

	// Network Plugin Ports
	Flannel: FirewallRule{
		Port:        "8472",
		Protocol:    "udp",
		Source:      "10.0.0.0/8",
		Direction:   "inbound",
		Action:      "allow",
		Description: "Flannel VXLAN",
	},
	Calico: FirewallRule{
		Port:        "179",
		Protocol:    "tcp",
		Source:      "10.0.0.0/8",
		Direction:   "inbound",
		Action:      "allow",
		Description: "Calico BGP",
	},
	CanalBGP: FirewallRule{
		Port:        "179",
		Protocol:    "tcp",
		Source:      "10.0.0.0/8",
		Direction:   "inbound",
		Action:      "allow",
		Description: "Canal BGP",
	},
	Weave: FirewallRule{
		Port:        "6783",
		Protocol:    "tcp",
		Source:      "10.0.0.0/8",
		Direction:   "inbound",
		Action:      "allow",
		Description: "Weave Net",
	},

	// Additional Ports
	SSH: FirewallRule{
		Port:        "22",
		Protocol:    "tcp",
		Source:      "10.8.0.0/24", // Only from WireGuard
		Direction:   "inbound",
		Action:      "allow",
		Description: "SSH via WireGuard",
	},
	WireGuard: FirewallRule{
		Port:        "51820",
		Protocol:    "udp",
		Source:      "0.0.0.0/0",
		Direction:   "inbound",
		Action:      "allow",
		Description: "WireGuard VPN",
	},
	DockerRegistry: FirewallRule{
		Port:        "5000",
		Protocol:    "tcp",
		Source:      "10.0.0.0/8",
		Direction:   "inbound",
		Action:      "allow",
		Description: "Docker Registry",
	},
	MetricsServer: FirewallRule{
		Port:        "10255",
		Protocol:    "tcp",
		Source:      "10.0.0.0/8",
		Direction:   "inbound",
		Action:      "allow",
		Description: "Metrics Server",
	},

	// Ingress Controller
	HTTPIngress: FirewallRule{
		Port:        "80",
		Protocol:    "tcp",
		Source:      "0.0.0.0/0",
		Direction:   "inbound",
		Action:      "allow",
		Description: "HTTP Ingress",
	},
	HTTPSIngress: FirewallRule{
		Port:        "443",
		Protocol:    "tcp",
		Source:      "0.0.0.0/0",
		Direction:   "inbound",
		Action:      "allow",
		Description: "HTTPS Ingress",
	},
}

// NewOSFirewallManager creates a new OS firewall manager
func NewOSFirewallManager(ctx *pulumi.Context) *OSFirewallManager {
	return &OSFirewallManager{
		ctx:     ctx,
		nodes:   make([]*providers.NodeOutput, 0),
		results: make(map[string]*FirewallResult),
		timeout: 5 * time.Minute,
	}
}

// AddNode adds a node to be configured
func (m *OSFirewallManager) AddNode(node *providers.NodeOutput) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nodes = append(m.nodes, node)
	m.results[node.Name] = &FirewallResult{
		NodeName:     node.Name,
		RulesApplied: []FirewallRule{},
	}
}

// SetSSHKeyPath sets the SSH key path
func (m *OSFirewallManager) SetSSHKeyPath(path string) {
	m.sshKeyPath = path
}

// ConfigureAllNodesFirewall configures firewall on all nodes using goroutines
func (m *OSFirewallManager) ConfigureAllNodesFirewall() error {
	m.ctx.Log.Info("Starting OS firewall configuration on all nodes", nil)

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	// Channel to collect results
	resultChan := make(chan *FirewallResult, len(m.nodes))
	errorChan := make(chan error, len(m.nodes))

	// WaitGroup to track all goroutines
	var wg sync.WaitGroup

	// Launch a goroutine for each node
	for _, node := range m.nodes {
		wg.Add(1)
		go m.configureNodeFirewall(ctx, &wg, node, resultChan, errorChan)
	}

	// Monitor goroutine
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Collect results
	allSuccess := true
	failedNodes := []string{}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout configuring OS firewalls")

		case err := <-errorChan:
			if err != nil {
				m.ctx.Log.Warn("Firewall configuration error", nil)
			}

		case result, ok := <-resultChan:
			if !ok {
				// All results collected
				if !allSuccess {
					return fmt.Errorf("firewall configuration failed on nodes: %v", failedNodes)
				}

				m.ctx.Log.Info("✓ OS firewall configured successfully on all nodes!", nil)
				m.printFirewallSummary()
				return nil
			}

			// Update results
			m.mu.Lock()
			m.results[result.NodeName] = result
			m.mu.Unlock()

			if !result.Success {
				allSuccess = false
				failedNodes = append(failedNodes, result.NodeName)
				m.ctx.Log.Error("Firewall configuration failed", nil)
			} else {
				m.ctx.Log.Info("Firewall configured", nil)
			}
		}
	}
}

// configureNodeFirewall configures firewall on a single node
func (m *OSFirewallManager) configureNodeFirewall(
	ctx context.Context,
	wg *sync.WaitGroup,
	node *providers.NodeOutput,
	resultChan chan<- *FirewallResult,
	errorChan chan<- error,
) {
	defer wg.Done()

	result := &FirewallResult{
		NodeName:     node.Name,
		Timestamp:    time.Now(),
		RulesApplied: []FirewallRule{},
	}

	// Determine which rules to apply based on node role
	rules := m.getRulesForNode(node)

	// Generate firewall configuration script
	script := m.generateFirewallScript(node, rules)

	// Execute the script on the node
	cmd, err := remote.NewCommand(m.ctx,
		fmt.Sprintf("os-firewall-%s-%d", node.Name, time.Now().Unix()),
		&remote.CommandArgs{
			Connection: &remote.ConnectionArgs{
				Host:       node.PublicIP,
				Port:       pulumi.Float64(22),
				User:       pulumi.String(node.SSHUser),
				PrivateKey: pulumi.String(m.getSSHPrivateKey()),
			},
			Create: pulumi.String(script),
		},
		pulumi.IgnoreChanges([]string{"create"}),
	)

	if err != nil {
		result.Error = err
		result.Success = false
		resultChan <- result
		return
	}

	// Parse the output
	cmd.Stdout.ApplyT(func(output string) string {
		if strings.Contains(output, "FIREWALL_TYPE:") {
			result.FirewallType = extractValue(output, "FIREWALL_TYPE:")
		}
		if strings.Contains(output, "SUCCESS") {
			result.Success = true
			result.RulesApplied = rules
		}
		return output
	})

	resultChan <- result
}

// getRulesForNode returns the appropriate firewall rules based on node role
func (m *OSFirewallManager) getRulesForNode(node *providers.NodeOutput) []FirewallRule {
	rules := []FirewallRule{}

	// Common rules for all nodes
	rules = append(rules,
		KubernetesFirewallPorts.SSH,
		KubernetesFirewallPorts.WireGuard,
		KubernetesFirewallPorts.Kubelet,
		KubernetesFirewallPorts.KubeProxy,
	)

	// Check if this is a master/control plane node
	isMaster := false
	isWorker := false

	if node.Labels != nil {
		if role, ok := node.Labels["role"]; ok {
			if role == "master" || role == "controlplane" {
				isMaster = true
			}
			if role == "worker" {
				isWorker = true
			}
		}
	}

	// Master-specific rules
	if isMaster {
		rules = append(rules,
			KubernetesFirewallPorts.APIServer,
			KubernetesFirewallPorts.ETCD,
			KubernetesFirewallPorts.ETCDPeer,
			KubernetesFirewallPorts.Scheduler,
			KubernetesFirewallPorts.ControllerManager,
		)
	}

	// Worker-specific rules
	if isWorker {
		rules = append(rules,
			KubernetesFirewallPorts.NodePortServices,
		)
	}

	// Network plugin rules (Canal for this cluster)
	rules = append(rules,
		KubernetesFirewallPorts.CanalBGP,
		KubernetesFirewallPorts.Flannel,
	)

	// Metrics and monitoring
	rules = append(rules,
		KubernetesFirewallPorts.MetricsServer,
	)

	// Ingress controller (if this is a worker node)
	if isWorker {
		rules = append(rules,
			KubernetesFirewallPorts.HTTPIngress,
			KubernetesFirewallPorts.HTTPSIngress,
		)
	}

	return rules
}

// generateFirewallScript generates the firewall configuration script
func (m *OSFirewallManager) generateFirewallScript(node *providers.NodeOutput, rules []FirewallRule) string {
	script := `#!/bin/bash
set -e

echo "=== OS Firewall Configuration ==="
echo "Node: ` + node.Name + `"
echo "Timestamp: $(date)"
echo ""

# Detect firewall type
FIREWALL_TYPE=""
if command -v ufw &> /dev/null; then
    FIREWALL_TYPE="ufw"
elif command -v firewall-cmd &> /dev/null; then
    FIREWALL_TYPE="firewalld"
elif command -v iptables &> /dev/null; then
    FIREWALL_TYPE="iptables"
else
    echo "ERROR: No supported firewall found"
    exit 1
fi

echo "FIREWALL_TYPE:$FIREWALL_TYPE"
echo "Detected firewall: $FIREWALL_TYPE"
echo ""

# Function to configure UFW
configure_ufw() {
    echo "Configuring UFW firewall..."

    # Disable and reset UFW
    ufw --force disable
    echo "y" | ufw --force reset

    # Set default policies
    ufw default deny incoming
    ufw default allow outgoing
    ufw default allow routed

    # Enable forwarding for Kubernetes
    sed -i 's/DEFAULT_FORWARD_POLICY="DROP"/DEFAULT_FORWARD_POLICY="ACCEPT"/' /etc/default/ufw

    # Allow established connections
    ufw allow in on lo
    ufw allow out on lo

    # Apply rules
`

	// Add UFW rules
	for _, rule := range rules {
		if rule.Source == "0.0.0.0/0" {
			script += fmt.Sprintf("    ufw allow %s/%s comment '%s'\n",
				rule.Port, rule.Protocol, rule.Description)
		} else {
			script += fmt.Sprintf("    ufw allow from %s to any port %s proto %s comment '%s'\n",
				rule.Source, rule.Port, rule.Protocol, rule.Description)
		}
	}

	script += `

    # Enable UFW
    echo "y" | ufw enable

    # Show status
    ufw status verbose

    echo "UFW configuration complete"
}

# Function to configure firewalld
configure_firewalld() {
    echo "Configuring firewalld..."

    # Start firewalld if not running
    systemctl start firewalld || true
    systemctl enable firewalld || true

    # Set default zone
    firewall-cmd --set-default-zone=public

    # Enable masquerading for Kubernetes
    firewall-cmd --add-masquerade --permanent

    # Apply rules
`

	// Add firewalld rules
	for _, rule := range rules {
		if rule.Source == "0.0.0.0/0" {
			script += fmt.Sprintf("    firewall-cmd --add-port=%s/%s --permanent\n",
				rule.Port, rule.Protocol)
		} else {
			script += fmt.Sprintf("    firewall-cmd --add-rich-rule='rule family=ipv4 source address=%s port port=%s protocol=%s accept' --permanent\n",
				rule.Source, rule.Port, rule.Protocol)
		}
	}

	script += `

    # Reload firewalld
    firewall-cmd --reload

    # Show configuration
    firewall-cmd --list-all

    echo "firewalld configuration complete"
}

# Function to configure iptables
configure_iptables() {
    echo "Configuring iptables..."

    # Save existing rules
    iptables-save > /tmp/iptables.backup

    # Set default policies
    iptables -P INPUT DROP
    iptables -P FORWARD ACCEPT
    iptables -P OUTPUT ACCEPT

    # Allow loopback
    iptables -A INPUT -i lo -j ACCEPT
    iptables -A OUTPUT -o lo -j ACCEPT

    # Allow established connections
    iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

    # Apply rules
`

	// Add iptables rules
	for _, rule := range rules {
		proto := strings.ToUpper(rule.Protocol)
		if rule.Source == "0.0.0.0/0" {
			script += fmt.Sprintf("    iptables -A INPUT -p %s --dport %s -j ACCEPT -m comment --comment '%s'\n",
				proto, rule.Port, rule.Description)
		} else {
			script += fmt.Sprintf("    iptables -A INPUT -s %s -p %s --dport %s -j ACCEPT -m comment --comment '%s'\n",
				rule.Source, proto, rule.Port, rule.Description)
		}
	}

	script += `

    # Enable IP forwarding for Kubernetes
    echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
    echo "net.bridge.bridge-nf-call-iptables=1" >> /etc/sysctl.conf
    echo "net.bridge.bridge-nf-call-ip6tables=1" >> /etc/sysctl.conf
    sysctl -p

    # Save iptables rules
    if command -v iptables-save &> /dev/null; then
        iptables-save > /etc/iptables/rules.v4 2>/dev/null || \
        iptables-save > /etc/sysconfig/iptables 2>/dev/null || \
        iptables-save > /etc/iptables.rules
    fi

    # Show rules
    iptables -L -n -v

    echo "iptables configuration complete"
}

# Configure based on detected firewall
case "$FIREWALL_TYPE" in
    ufw)
        configure_ufw
        ;;
    firewalld)
        configure_firewalld
        ;;
    iptables)
        configure_iptables
        ;;
    *)
        echo "ERROR: Unsupported firewall type"
        exit 1
        ;;
esac

echo ""
echo "=== Firewall Configuration Complete ==="
echo "SUCCESS"
`

	return script
}

// printFirewallSummary prints a summary of firewall configuration
func (m *OSFirewallManager) printFirewallSummary() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.ctx.Log.Info("Firewall Configuration Summary", nil)
	m.ctx.Log.Info("==============================", nil)

	successCount := 0
	failedCount := 0

	for nodeName, result := range m.results {
		if result.Success {
			successCount++
			m.ctx.Log.Info(fmt.Sprintf("✓ %s: %s firewall configured with %d rules",
				nodeName, result.FirewallType, len(result.RulesApplied)), nil)
		} else {
			failedCount++
			m.ctx.Log.Warn(fmt.Sprintf("✗ %s: Failed - %v",
				nodeName, result.Error), nil)
		}
	}

	m.ctx.Log.Info(fmt.Sprintf("Total: %d successful, %d failed",
		successCount, failedCount), nil)
}

// GetResults returns all firewall configuration results
func (m *OSFirewallManager) GetResults() map[string]*FirewallResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]*FirewallResult)
	for k, v := range m.results {
		results[k] = v
	}

	return results
}

// getSSHPrivateKey retrieves the SSH private key
func (m *OSFirewallManager) getSSHPrivateKey() string {
	// In production, read from m.sshKeyPath
	return "SSH_PRIVATE_KEY_CONTENT"
}

// Helper function to extract value from output
func extractValue(s, prefix string) string {
	idx := strings.Index(s, prefix)
	if idx == -1 {
		return ""
	}

	start := idx + len(prefix)
	end := start
	for end < len(s) && s[end] != '\n' && s[end] != ' ' {
		end++
	}

	return s[start:end]
}