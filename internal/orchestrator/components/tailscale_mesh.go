package components

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// TailscaleMeshComponent configures Tailscale VPN mesh with Headscale coordination server
type TailscaleMeshComponent struct {
	pulumi.ResourceState

	Status       pulumi.StringOutput `pulumi:"status"`
	PeerCount    pulumi.IntOutput    `pulumi:"peerCount"`
	HeadscaleURL pulumi.StringOutput `pulumi:"headscaleUrl"`
	HeadscaleIP  pulumi.StringOutput `pulumi:"headscaleIp"`
	APIKey       pulumi.StringOutput `pulumi:"apiKey"`
	Namespace    pulumi.StringOutput `pulumi:"namespace"`
}

// TailscaleMeshArgs contains the arguments for creating a Tailscale mesh
type TailscaleMeshArgs struct {
	Config        *config.TailscaleConfig
	SSHPrivateKey pulumi.StringOutput
	SSHPublicKey  pulumi.StringOutput
	ClusterName   string
}

// NewTailscaleMeshComponent sets up Tailscale mesh between nodes via Headscale
func NewTailscaleMeshComponent(ctx *pulumi.Context, name string, nodes []*RealNodeComponent, args *TailscaleMeshArgs, bastionComponent *BastionComponent, opts ...pulumi.ResourceOption) (*TailscaleMeshComponent, error) {
	component := &TailscaleMeshComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:TailscaleMesh", name, component, opts...)
	if err != nil {
		return nil, err
	}

	peerCount := len(nodes)
	ctx.Log.Info(fmt.Sprintf("🔧 Configuring Tailscale mesh: %d nodes", peerCount), nil)

	// Default namespace
	namespace := args.Config.Namespace
	if namespace == "" {
		namespace = "kubernetes"
	}

	// STEP 1: Create Headscale coordination server if create=true
	var headscaleIP pulumi.StringOutput
	var headscaleURL pulumi.StringOutput
	var headscaleInstance *ec2.Instance
	var headscaleReady pulumi.Resource

	if args.Config.Create {
		ctx.Log.Info("🏗️  Creating Headscale coordination server...", nil)

		// Get region from config
		region := args.Config.Region
		if region == "" {
			region = "us-east-1" // Default region
		}

		// Create an explicit AWS provider for Headscale resources
		awsProvider, err := aws.NewProvider(ctx, fmt.Sprintf("%s-headscale-aws-provider", name), &aws.ProviderArgs{
			Region: pulumi.String(region),
		}, pulumi.Parent(component))
		if err != nil {
			return nil, fmt.Errorf("failed to create AWS provider for Headscale: %w", err)
		}

		// Create SSH key pair for Headscale server
		headscaleKeyPair, err := ec2.NewKeyPair(ctx, fmt.Sprintf("%s-headscale-keypair", name), &ec2.KeyPairArgs{
			KeyName:   pulumi.Sprintf("%s-headscale-key", args.ClusterName),
			PublicKey: args.SSHPublicKey,
			Tags: pulumi.StringMap{
				"Name": pulumi.String(fmt.Sprintf("%s-headscale-keypair", name)),
			},
		}, pulumi.Parent(component), pulumi.Provider(awsProvider))
		if err != nil {
			return nil, fmt.Errorf("failed to create Headscale SSH key pair: %w", err)
		}

		// Create Headscale security group
		headscaleSG, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-headscale-sg", name), &ec2.SecurityGroupArgs{
			Description: pulumi.String("Security group for Headscale coordination server"),
			Ingress: ec2.SecurityGroupIngressArray{
				// SSH
				&ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(22),
					ToPort:     pulumi.Int(22),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				// Headscale HTTP
				&ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(8080),
					ToPort:     pulumi.Int(8080),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				// Headscale gRPC
				&ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(50443),
					ToPort:     pulumi.Int(50443),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				// HTTPS (for future TLS)
				&ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(443),
					ToPort:     pulumi.Int(443),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String(fmt.Sprintf("%s-headscale-sg", name)),
			},
		}, pulumi.Parent(component), pulumi.Provider(awsProvider))
		if err != nil {
			return nil, fmt.Errorf("failed to create Headscale security group: %w", err)
		}

		// Find Ubuntu 22.04 AMI (use invoke options with provider)
		ami, err := ec2.LookupAmi(ctx, &ec2.LookupAmiArgs{
			MostRecent: pulumi.BoolRef(true),
			Owners:     []string{"099720109477"}, // Canonical
			Filters: []ec2.GetAmiFilter{
				{
					Name:   "name",
					Values: []string{"ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"},
				},
				{
					Name:   "virtualization-type",
					Values: []string{"hvm"},
				},
			},
		}, pulumi.Provider(awsProvider))
		if err != nil {
			return nil, fmt.Errorf("failed to find Ubuntu AMI: %w", err)
		}

		// Instance type
		instanceType := args.Config.Size
		if instanceType == "" {
			instanceType = "t3.small"
		}

		// Generate Headscale cloud-init script
		installScript := generateHeadscaleCloudInit(namespace)

		// Create Headscale EC2 instance
		headscaleInstance, err = ec2.NewInstance(ctx, fmt.Sprintf("%s-headscale", name), &ec2.InstanceArgs{
			InstanceType:             pulumi.String(instanceType),
			Ami:                      pulumi.String(ami.Id),
			KeyName:                  headscaleKeyPair.KeyName,
			VpcSecurityGroupIds:      pulumi.StringArray{headscaleSG.ID()},
			AssociatePublicIpAddress: pulumi.Bool(true),
			UserData:                 pulumi.String(installScript),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(fmt.Sprintf("%s-headscale", name)),
				"Role": pulumi.String("headscale"),
			},
			RootBlockDevice: &ec2.InstanceRootBlockDeviceArgs{
				VolumeSize: pulumi.Int(20),
				VolumeType: pulumi.String("gp3"),
			},
		}, pulumi.Parent(component), pulumi.Provider(awsProvider))
		if err != nil {
			return nil, fmt.Errorf("failed to create Headscale instance: %w", err)
		}

		headscaleIP = headscaleInstance.PublicIp
		headscaleURL = headscaleIP.ApplyT(func(ip string) string {
			return fmt.Sprintf("http://%s:8080", ip)
		}).(pulumi.StringOutput)

		ctx.Log.Info("✅ Headscale server created", nil)

		// Wait for Headscale to be ready and get auth key
		ctx.Log.Info("⏳ Waiting for Headscale server to initialize...", nil)
		waitAndGetAuthCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-headscale-wait", name), &remote.CommandArgs{
			Connection: remote.ConnectionArgs{
				Host:           headscaleIP,
				User:           pulumi.String("ubuntu"),
				PrivateKey:     args.SSHPrivateKey,
				DialErrorLimit: pulumi.Int(60), // More retries for cloud-init to complete
			},
			Create: pulumi.String(`#!/bin/bash
set -e

# Wait for cloud-init to complete (up to 10 minutes)
echo "Waiting for cloud-init to complete..."
for i in {1..120}; do
    if sudo [ -f /root/headscale-auth-key ] && sudo [ -f /root/headscale-url ]; then
        echo "Headscale is ready!"
        break
    fi
    echo "Waiting... ($i/120)"
    sleep 5
done

# Check if files exist
if ! sudo [ -f /root/headscale-auth-key ]; then
    echo "ERROR: Auth key not found after 10 minutes"
    # Show cloud-init status for debugging
    echo "Cloud-init status:"
    cloud-init status --long || true
    echo "Checking headscale service:"
    sudo systemctl status headscale || true
    exit 1
fi

# Output credentials
echo "=== HEADSCALE_CREDENTIALS ==="
echo "URL=$(sudo cat /root/headscale-url)"
echo "AUTH_KEY=$(sudo cat /root/headscale-auth-key)"
echo "=== END_CREDENTIALS ==="
`),
		}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{headscaleInstance}), pulumi.Timeouts(&pulumi.CustomTimeouts{
			Create: "15m",
		}))
		if err != nil {
			return nil, fmt.Errorf("failed to wait for Headscale: %w", err)
		}

		headscaleReady = waitAndGetAuthCmd
		ctx.Log.Info("✅ Headscale server ready", nil)

		// Fetch API key from Headscale server and store as secret
		fetchAPIKeyCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-headscale-fetch-apikey", name), &remote.CommandArgs{
			Connection: remote.ConnectionArgs{
				Host:       headscaleIP,
				User:       pulumi.String("ubuntu"),
				PrivateKey: args.SSHPrivateKey,
			},
			Create: pulumi.String(`sudo cat /root/headscale-api-key 2>/dev/null || echo ""`),
		}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{waitAndGetAuthCmd}))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to fetch API key: %v", err), nil)
		}

		// Store API key as secret output
		if fetchAPIKeyCmd != nil {
			component.APIKey = pulumi.ToSecret(fetchAPIKeyCmd.Stdout).(pulumi.StringOutput)
		} else {
			component.APIKey = pulumi.ToSecret(pulumi.String("")).(pulumi.StringOutput)
		}

	} else {
		// Using existing Headscale server
		if args.Config.HeadscaleURL == "" {
			return nil, fmt.Errorf("headscale-url is required when create=false")
		}
		headscaleURL = pulumi.String(args.Config.HeadscaleURL).ToStringOutput()
		headscaleIP = pulumi.String("").ToStringOutput() // Not applicable for external Headscale

		// Use API key from config (stored as secret)
		if args.Config.APIKey != "" {
			component.APIKey = pulumi.ToSecret(pulumi.String(args.Config.APIKey)).(pulumi.StringOutput)
		} else {
			component.APIKey = pulumi.ToSecret(pulumi.String("")).(pulumi.StringOutput)
		}
	}

	// Set namespace (always store for later use)
	component.Namespace = pulumi.String(namespace).ToStringOutput()

	// STEP 2: Install Tailscale on each node and join to Headscale
	ctx.Log.Info("🔧 Installing Tailscale on cluster nodes...", nil)

	var installDependencies []pulumi.Resource
	if headscaleReady != nil {
		installDependencies = append(installDependencies, headscaleReady)
	}

	// Build the auth key retrieval - either from newly created Headscale or from config
	var authKeySource pulumi.StringOutput
	if args.Config.Create && headscaleInstance != nil {
		// Get auth key from the Headscale server we created
		authKeySource = pulumi.All(headscaleIP, args.SSHPrivateKey).ApplyT(func(args []interface{}) string {
			// This is a placeholder - actual key is fetched via SSH command
			// The real auth key is fetched in the join command
			return "FETCH_FROM_SERVER"
		}).(pulumi.StringOutput)
	} else if args.Config.AuthKey != "" {
		// Use provided auth key
		authKeySource = pulumi.String(args.Config.AuthKey).ToStringOutput()
	} else {
		return nil, fmt.Errorf("auth-key is required when using external Headscale server")
	}

	// Install Tailscale on each node
	for i, node := range nodes {
		hostname := node.NodeName.ApplyT(func(name string) string {
			return name
		}).(pulumi.StringOutput)

		// Build Tailscale installation and join script
		joinScript := pulumi.All(headscaleURL, hostname, headscaleIP, args.SSHPrivateKey, authKeySource).ApplyT(func(args []interface{}) string {
			url := args[0].(string)
			host := args[1].(string)
			hsIP := args[2].(string)
			sshKey := args[3].(string)
			authKeySrc := args[4].(string)

			// If we need to fetch auth key from Headscale server
			fetchAuthKeyCmd := ""
			if authKeySrc == "FETCH_FROM_SERVER" && hsIP != "" {
				// Create a temp SSH key file and fetch auth key
				fetchAuthKeyCmd = fmt.Sprintf(`
# Fetch auth key from Headscale server
echo "Fetching auth key from Headscale server..."
mkdir -p ~/.ssh
cat > ~/.ssh/headscale_key << 'SSHKEYEOF'
%s
SSHKEYEOF
chmod 600 ~/.ssh/headscale_key

# Try to get auth key via SSH
for i in {1..30}; do
    AUTH_KEY=$(ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/headscale_key ubuntu@%s 'sudo cat /root/headscale-auth-key 2>/dev/null' 2>/dev/null)
    if [ -n "$AUTH_KEY" ]; then
        echo "Got auth key from Headscale server"
        break
    fi
    echo "Waiting for auth key... ($i/30)"
    sleep 10
done

if [ -z "$AUTH_KEY" ]; then
    echo "ERROR: Could not get auth key from Headscale server"
    exit 1
fi

rm -f ~/.ssh/headscale_key
`, sshKey, hsIP)
			} else {
				fetchAuthKeyCmd = fmt.Sprintf(`AUTH_KEY="%s"`, authKeySrc)
			}

			return fmt.Sprintf(`#!/bin/bash
set -e

echo "=== Installing Tailscale on %s ==="

%s

# Install Tailscale
echo "Installing Tailscale..."
if ! command -v tailscale &> /dev/null; then
    curl -fsSL https://tailscale.com/install.sh | sh
fi

# Enable tailscaled service
sudo systemctl enable --now tailscaled

# Wait for tailscaled to be ready
sleep 5

# Join the Tailscale network via Headscale
echo "Joining Tailscale network via Headscale..."
echo "Headscale URL: %s"
echo "Hostname: %s"

sudo tailscale up --login-server=%s --authkey="$AUTH_KEY" --hostname=%s --accept-routes --reset

# Verify connection
echo "Verifying Tailscale connection..."
sleep 3
sudo tailscale status

# Get assigned IP
TAILSCALE_IP=$(sudo tailscale ip -4 2>/dev/null || echo "pending")
echo "Tailscale IP: $TAILSCALE_IP"

echo "=== Tailscale installation complete ==="
`, host, fetchAuthKeyCmd, url, host, url, host)
		}).(pulumi.StringOutput)

		// Use provider-specific SSH user
		sshUser := getSSHUserForWireGuard(node.Provider)
		sudoPrefix := getSudoPrefixForUser(node.Provider)
		_ = sudoPrefix // Reserved for future use

		connectionArgs := remote.ConnectionArgs{
			Host:           node.PublicIP,
			User:           sshUser,
			PrivateKey:     args.SSHPrivateKey,
			DialErrorLimit: pulumi.Int(30),
		}

		// Add ProxyJump when bastion is enabled
		if bastionComponent != nil {
			connectionArgs.Proxy = &remote.ProxyConnectionArgs{
				Host:       bastionComponent.PublicIP,
				User:       getSSHUserForProvider(bastionComponent.Provider),
				PrivateKey: args.SSHPrivateKey,
			}
		}

		_, err := remote.NewCommand(ctx, fmt.Sprintf("%s-tailscale-join-%d", name, i), &remote.CommandArgs{
			Connection: connectionArgs,
			Create:     joinScript,
		}, pulumi.Parent(component), pulumi.DependsOn(installDependencies), pulumi.Timeouts(&pulumi.CustomTimeouts{
			Create: "15m",
		}))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to install Tailscale on node %d: %v", i, err), nil)
		} else {
			ctx.Log.Info(fmt.Sprintf("✅ Tailscale installed on node %d", i), nil)
		}
	}

	// Set component outputs
	component.Status = pulumi.Sprintf("Tailscale mesh: %d nodes connected via Headscale", peerCount)
	component.PeerCount = pulumi.Int(peerCount).ToIntOutput()
	component.HeadscaleURL = headscaleURL
	if headscaleInstance != nil {
		component.HeadscaleIP = headscaleIP
	} else {
		component.HeadscaleIP = pulumi.String("").ToStringOutput()
	}

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":       component.Status,
		"peerCount":    component.PeerCount,
		"headscaleUrl": component.HeadscaleURL,
		"headscaleIp":  component.HeadscaleIP,
		"apiKey":       component.APIKey,
		"namespace":    component.Namespace,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info(fmt.Sprintf("✅ Tailscale mesh COMPLETE: %d nodes connected", peerCount), nil)

	return component, nil
}

// generateHeadscaleCloudInit generates the cloud-init script for Headscale installation
func generateHeadscaleCloudInit(namespace string) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

exec > >(tee /var/log/headscale-install.log) 2>&1

echo "=== Starting Headscale Installation ==="
echo "Timestamp: $(date)"

# Update system
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get upgrade -y

# Install dependencies
apt-get install -y curl wget jq

# Get the latest Headscale version
HEADSCALE_VERSION=$(curl -s https://api.github.com/repos/juanfont/headscale/releases/latest | jq -r '.tag_name' | sed 's/v//')
echo "Installing Headscale version: $HEADSCALE_VERSION"

# Download and install Headscale
wget -q "https://github.com/juanfont/headscale/releases/download/v${HEADSCALE_VERSION}/headscale_${HEADSCALE_VERSION}_linux_amd64.deb"
dpkg -i "headscale_${HEADSCALE_VERSION}_linux_amd64.deb"

# Get public IP (with retry)
for i in {1..10}; do
    PUBLIC_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 2>/dev/null || curl -s ifconfig.me 2>/dev/null)
    if [ -n "$PUBLIC_IP" ]; then
        break
    fi
    echo "Waiting for public IP... ($i/10)"
    sleep 5
done
echo "Public IP: $PUBLIC_IP"

# Create Headscale configuration
mkdir -p /etc/headscale
cat > /etc/headscale/config.yaml <<EOF
server_url: http://${PUBLIC_IP}:8080
listen_addr: 0.0.0.0:8080
metrics_listen_addr: 127.0.0.1:9090
grpc_listen_addr: 0.0.0.0:50443
grpc_allow_insecure: true

private_key_path: /var/lib/headscale/private.key
noise:
  private_key_path: /var/lib/headscale/noise_private.key

prefixes:
  v4: 100.64.0.0/10
  v6: fd7a:115c:a1e0::/48

derp:
  server:
    enabled: false
  urls:
    - https://controlplane.tailscale.com/derpmap/default
  auto_update_enabled: true
  update_frequency: 24h

disable_check_updates: false
ephemeral_node_inactivity_timeout: 30m

database:
  type: sqlite
  sqlite:
    path: /var/lib/headscale/db.sqlite

log:
  format: text
  level: info

dns:
  magic_dns: true
  base_domain: headscale.local
  nameservers:
    global:
      - 1.1.1.1
      - 8.8.8.8

policy:
  mode: file
  path: ""
EOF

# Create data directory
mkdir -p /var/lib/headscale
chown -R headscale:headscale /var/lib/headscale

# Create systemd service
cat > /etc/systemd/system/headscale.service <<EOF
[Unit]
Description=headscale coordination server
After=network.target

[Service]
Type=simple
User=headscale
Group=headscale
ExecStart=/usr/bin/headscale serve
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Enable and start Headscale
systemctl daemon-reload
systemctl enable headscale
systemctl start headscale

# Wait for Headscale to be ready (check if it's listening)
echo "Waiting for Headscale to start..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health >/dev/null 2>&1 || netstat -tlnp 2>/dev/null | grep -q 8080; then
        echo "Headscale is listening on port 8080"
        break
    fi
    echo "Waiting for Headscale... ($i/30)"
    sleep 2
done

# Additional wait for service stabilization
sleep 10

# Create user (new Headscale uses 'users' instead of 'namespaces')
echo "Creating user: %s"
headscale users create %s 2>/dev/null || headscale namespaces create %s 2>/dev/null || true

# Wait a bit for user creation
sleep 5

# Create API key and save it
echo "Creating API key..."
API_KEY=$(headscale apikeys create --expiration 365d 2>&1 | grep -v "^API" | tail -1)
echo "$API_KEY" > /root/headscale-api-key
echo "API Key created: ${API_KEY:0:10}..."

# Create pre-auth key for nodes (reusable, 30 day expiration)
# Headscale v0.26+ requires user ID (numeric), not username string
echo "Creating pre-auth key..."

# First, get the user ID for our user
echo "Getting user ID for %s..."
USER_ID=""

# Try to get user ID using JSON output (works in v0.23+)
if command -v jq &> /dev/null; then
    USER_ID=$(headscale users list -o json 2>/dev/null | jq -r '.[] | select(.name=="%s") | .id' | head -1)
fi

# Fallback: parse text output to get user ID
if [[ -z "$USER_ID" ]]; then
    # headscale users list output format varies, try to extract first user ID
    USER_ID=$(headscale users list 2>/dev/null | grep -E "^[0-9]+" | head -1 | awk '{print $1}')
fi

# If still no ID, assume user ID 1 (first user created)
if [[ -z "$USER_ID" ]]; then
    echo "Could not determine user ID, assuming ID 1"
    USER_ID="1"
fi

echo "User ID: $USER_ID"

# Try to create preauthkey with numeric user ID (v0.26+ syntax)
AUTH_KEY=$(headscale preauthkeys create --user "$USER_ID" --reusable --expiration 720h 2>&1)
AUTH_KEY_RESULT=$?

# Check if we got a valid key
if [[ "$AUTH_KEY_RESULT" -ne 0 ]] || [[ "$AUTH_KEY" == *"invalid"* ]] || [[ "$AUTH_KEY" == *"error"* ]] || [[ "$AUTH_KEY" == *"Error"* ]]; then
    echo "Numeric user ID syntax failed (exit code: $AUTH_KEY_RESULT), trying username syntax..."
    echo "Previous output: $AUTH_KEY"

    # Try with username (older versions)
    AUTH_KEY=$(headscale preauthkeys create --user %s --reusable --expiration 720h 2>&1)

    # If that fails too, try namespace syntax (very old versions)
    if [[ "$AUTH_KEY" == *"invalid"* ]] || [[ "$AUTH_KEY" == *"error"* ]]; then
        echo "Username syntax failed, trying --namespace syntax..."
        AUTH_KEY=$(headscale preauthkeys create --namespace %s --reusable --expiration 720h 2>&1)
    fi
fi

# Extract just the key if there's extra output
if [[ "$AUTH_KEY" == *$'\n'* ]]; then
    # Get the last non-empty line that looks like a key
    AUTH_KEY=$(echo "$AUTH_KEY" | grep -v "^Pre" | grep -v "^$" | tail -1)
fi

# Trim whitespace
AUTH_KEY=$(echo "$AUTH_KEY" | xargs)

echo "Auth Key result: ${AUTH_KEY:0:20}..."

# Validate the key looks correct (should be alphanumeric/base64-like)
if [[ -z "$AUTH_KEY" ]] || [[ "$AUTH_KEY" == *"invalid"* ]] || [[ "$AUTH_KEY" == *"error"* ]] || [[ "$AUTH_KEY" == *"Error"* ]] || [[ "$AUTH_KEY" == *"strconv"* ]]; then
    echo "ERROR: Failed to create valid auth key"
    echo "Auth key output was: $AUTH_KEY"
    echo "Headscale version info:"
    headscale version 2>/dev/null || true
    echo "Available users:"
    headscale users list 2>/dev/null || headscale namespaces list 2>/dev/null || true
    echo "Trying to list preauthkeys..."
    headscale preauthkeys list --user "$USER_ID" 2>/dev/null || true
    # Set empty to indicate failure
    AUTH_KEY=""
else
    echo "Auth key created successfully!"
fi

echo "$AUTH_KEY" > /root/headscale-auth-key

# Save server info
echo "http://${PUBLIC_IP}:8080" > /root/headscale-url

# Verify files
echo "=== Verification ==="
echo "URL file contents: $(cat /root/headscale-url)"
echo "Auth key file exists: $(test -f /root/headscale-auth-key && echo yes || echo no)"
echo "Auth key file size: $(wc -c < /root/headscale-auth-key 2>/dev/null || echo 0)"

echo "=== Headscale Installation Complete ==="
echo "Server URL: http://${PUBLIC_IP}:8080"
echo "Namespace/User: %s"
`, namespace, namespace, namespace, namespace, namespace, namespace, namespace, namespace)
}
