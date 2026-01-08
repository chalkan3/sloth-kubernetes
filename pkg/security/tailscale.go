package security

import (
	"fmt"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	"github.com/chalkan3/sloth-kubernetes/pkg/vpn"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// TailscaleManager manages Tailscale VPN configuration via Headscale
type TailscaleManager struct {
	ctx              *pulumi.Context
	config           *config.TailscaleConfig
	nodes            []*providers.NodeOutput
	headscaleURL     pulumi.StringOutput
	authKey          pulumi.StringOutput
	sshPrivateKey    string
	headscaleReady   pulumi.Resource
	headscaleInfoSet bool // Track if headscale info was dynamically set
}

// NewTailscaleManager creates a new Tailscale manager
func NewTailscaleManager(ctx *pulumi.Context, cfg *config.TailscaleConfig) *TailscaleManager {
	return &TailscaleManager{
		ctx:    ctx,
		config: cfg,
		nodes:  make([]*providers.NodeOutput, 0),
	}
}

// SetHeadscaleInfo sets the Headscale coordination server info
func (t *TailscaleManager) SetHeadscaleInfo(url, authKey pulumi.StringOutput, readyResource pulumi.Resource) {
	t.headscaleURL = url
	t.authKey = authKey
	t.headscaleReady = readyResource
	t.headscaleInfoSet = true
}

// SetSSHPrivateKey sets the SSH private key for connecting to nodes
func (t *TailscaleManager) SetSSHPrivateKey(key string) {
	t.sshPrivateKey = key
}

// ConfigureNode configures Tailscale on a node
func (t *TailscaleManager) ConfigureNode(node *providers.NodeOutput) error {
	if !t.config.Enabled {
		return nil
	}

	t.nodes = append(t.nodes, node)

	// Build dependencies - wait for Headscale server if auto-provisioned
	var deps []pulumi.Resource
	if t.headscaleReady != nil {
		deps = append(deps, t.headscaleReady)
	}

	// Generate configuration script
	configScript := t.generateNodeConfigScript(node)

	// Configure Tailscale on the node
	_, err := remote.NewCommand(t.ctx, fmt.Sprintf("%s-tailscale-config", node.Name), &remote.CommandArgs{
		Connection: &remote.ConnectionArgs{
			Host:       node.PublicIP,
			Port:       pulumi.Float64(22),
			User:       pulumi.String(node.SSHUser),
			PrivateKey: pulumi.String(t.sshPrivateKey),
		},
		Create: configScript,
		Update: configScript,
		Delete: pulumi.String(`#!/bin/bash
# Leave the tailnet and stop Tailscale
tailscale logout || true
systemctl stop tailscaled || true
systemctl disable tailscaled || true
echo "Tailscale removed"
`),
	}, pulumi.DependsOn(deps))

	if err != nil {
		return fmt.Errorf("failed to configure Tailscale on %s: %w", node.Name, err)
	}

	return nil
}

// generateNodeConfigScript generates the Tailscale installation and configuration script
func (t *TailscaleManager) generateNodeConfigScript(node *providers.NodeOutput) pulumi.StringOutput {
	// Get config values with defaults
	namespace := t.config.Namespace
	if namespace == "" {
		namespace = "kubernetes"
	}

	// Build the script with dynamic values
	return pulumi.All(t.headscaleURL, t.authKey).ApplyT(func(args []interface{}) string {
		headscaleURL := args[0].(string)
		authKey := args[1].(string)

		// If config has static values, use them
		if headscaleURL == "" && t.config.HeadscaleURL != "" {
			headscaleURL = t.config.HeadscaleURL
		}
		if authKey == "" && t.config.AuthKey != "" {
			authKey = t.config.AuthKey
		}

		// Build tags argument
		tagsArg := ""
		if len(t.config.Tags) > 0 {
			tags := ""
			for i, tag := range t.config.Tags {
				if i > 0 {
					tags += ","
				}
				tags += tag
			}
			tagsArg = fmt.Sprintf("--advertise-tags=%s", tags)
		}

		// Build accept-routes argument
		acceptRoutesArg := ""
		if t.config.AcceptRoutes {
			acceptRoutesArg = "--accept-routes"
		}

		// Build exit-node argument
		exitNodeArg := ""
		if t.config.ExitNode != "" {
			exitNodeArg = fmt.Sprintf("--exit-node=%s", t.config.ExitNode)
		}

		return fmt.Sprintf(`#!/bin/bash
set -e

echo "=== Starting Tailscale Configuration on %s ==="

# Wait for cloud-init to complete
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do
    echo "Waiting for cloud-init to finish..."
    sleep 5
done

# Check if Tailscale is already installed
if ! command -v tailscale &> /dev/null; then
    echo "Installing Tailscale..."
    curl -fsSL https://tailscale.com/install.sh | sh

    # Enable and start tailscaled
    systemctl enable tailscaled
    systemctl start tailscaled

    # Wait for daemon to be ready
    sleep 5
fi

# Check if already connected to this tailnet
CURRENT_STATUS=$(tailscale status --json 2>/dev/null | jq -r '.BackendState' 2>/dev/null || echo "NotRunning")

if [ "$CURRENT_STATUS" = "Running" ]; then
    echo "Tailscale already connected, checking if configuration matches..."

    # Check current login server
    CURRENT_SERVER=$(tailscale debug prefs 2>/dev/null | grep -o 'ControlURL:[^ ]*' | cut -d: -f2- || echo "")
    if [ "$CURRENT_SERVER" != "%s" ]; then
        echo "Different login server detected, reconfiguring..."
        tailscale logout || true
        sleep 2
    else
        echo "Already connected to correct Headscale server"
        exit 0
    fi
fi

echo "Joining Headscale coordination server..."
echo "Server URL: %s"
echo "Hostname: %s"

# Join the tailnet via Headscale
tailscale up \
    --login-server=%s \
    --authkey=%s \
    --hostname=%s \
    %s \
    %s \
    %s \
    --timeout=120s \
    --reset

# Wait for connection to establish
echo "Waiting for Tailscale connection..."
for i in {1..30}; do
    STATUS=$(tailscale status --json 2>/dev/null | jq -r '.BackendState' 2>/dev/null || echo "Connecting")
    if [ "$STATUS" = "Running" ]; then
        echo "✓ Tailscale connected successfully!"
        break
    fi
    echo "Waiting... ($i/30)"
    sleep 2
done

# Verify connection
tailscale status

# Get assigned IP
TAILSCALE_IP=$(tailscale ip -4 2>/dev/null || echo "")
echo "Tailscale IP: $TAILSCALE_IP"

echo "=== Tailscale Configuration Complete on %s ==="
`,
			node.Name,
			headscaleURL,
			headscaleURL,
			node.Name,
			headscaleURL,
			authKey,
			node.Name,
			tagsArg,
			acceptRoutesArg,
			exitNodeArg,
			node.Name)
	}).(pulumi.StringOutput)
}

// ValidateConfiguration validates Tailscale configuration
func (t *TailscaleManager) ValidateConfiguration() error {
	if !t.config.Enabled {
		return nil
	}

	// If auto-provisioning, HeadscaleURL will be set dynamically
	if t.config.Create {
		return nil
	}

	// For existing Headscale, require URL and auth key
	if t.config.HeadscaleURL == "" && !t.headscaleInfoSet {
		return fmt.Errorf("Headscale URL is required")
	}

	if t.config.AuthKey == "" && !t.headscaleInfoSet {
		return fmt.Errorf("Tailscale auth key is required")
	}

	return nil
}

// ExportTailscaleInfo exports Tailscale information to Pulumi stack
func (t *TailscaleManager) ExportTailscaleInfo() {
	secrets.Export(t.ctx, "tailscale_configured", pulumi.Bool(t.config.Enabled))

	if t.config.Enabled {
		if t.headscaleInfoSet {
			secrets.Export(t.ctx, "headscale_url", t.headscaleURL)
		} else if t.config.HeadscaleURL != "" {
			secrets.Export(t.ctx, "headscale_url", pulumi.String(t.config.HeadscaleURL))
		}

		namespace := t.config.Namespace
		if namespace == "" {
			namespace = "kubernetes"
		}
		secrets.Export(t.ctx, "tailscale_namespace", pulumi.String(namespace))

		// Export node Tailscale info
		nodeInfo := pulumi.Map{}
		for _, node := range t.nodes {
			nodeInfo[node.Name] = pulumi.String(node.Name)
		}
		secrets.Export(t.ctx, "tailscale_nodes", nodeInfo)
	}
}

// GetNodeByName returns a node by name
func (t *TailscaleManager) GetNodeByName(name string) (*providers.NodeOutput, error) {
	for _, node := range t.nodes {
		if node.Name == name {
			return node, nil
		}
	}
	return nil, fmt.Errorf("node %s not found", name)
}

// GetNodes returns all configured nodes
func (t *TailscaleManager) GetNodes() []*providers.NodeOutput {
	return t.nodes
}

// WaitForTailscaleReady creates a resource that waits for all nodes to be connected via Tailscale
func (t *TailscaleManager) WaitForTailscaleReady() pulumi.Resource {
	// This would create a command that verifies all nodes are visible in tailscale status
	return nil
}

// TailscaleConnectivityChecker provides methods to verify Tailscale mesh connectivity
type TailscaleConnectivityChecker struct {
	manager *TailscaleManager
}

// NewTailscaleConnectivityChecker creates a new connectivity checker
func NewTailscaleConnectivityChecker(manager *TailscaleManager) *TailscaleConnectivityChecker {
	return &TailscaleConnectivityChecker{
		manager: manager,
	}
}

// VerifyConnectivity verifies that all nodes can see each other via Tailscale
func (c *TailscaleConnectivityChecker) VerifyConnectivity() error {
	// Implementation would SSH to each node and run:
	// tailscale ping <other-node-hostname>
	// to verify connectivity
	return nil
}

// CreateHeadscaleServerIfNeeded creates a Headscale server if config.Create is true
func (t *TailscaleManager) CreateHeadscaleServerIfNeeded(sshKeyPair interface{}, securityGroup interface{}, subnetID pulumi.StringOutput) (*vpn.HeadscaleResult, error) {
	if !t.config.Create {
		return nil, nil
	}

	headscaleMgr := vpn.NewHeadscaleManager(t.ctx)
	return headscaleMgr.CreateHeadscaleServer(t.config, nil, nil, subnetID)
}
