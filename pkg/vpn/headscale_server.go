package vpn

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
)

// HeadscaleManager handles Headscale coordination server creation
type HeadscaleManager struct {
	ctx *pulumi.Context
}

// NewHeadscaleManager creates a new Headscale manager
func NewHeadscaleManager(ctx *pulumi.Context) *HeadscaleManager {
	return &HeadscaleManager{ctx: ctx}
}

// HeadscaleResult contains created Headscale server information
type HeadscaleResult struct {
	Provider   string
	ServerID   pulumi.IDOutput
	ServerIP   pulumi.StringOutput
	ServerName string
	APIURL     pulumi.StringOutput
	APIKey     pulumi.StringOutput
	AuthKey    pulumi.StringOutput
	Namespace  string
	Port       int
}

// CreateHeadscaleServer creates a Headscale coordination server
func (m *HeadscaleManager) CreateHeadscaleServer(cfg *config.TailscaleConfig, sshKeyPair *ec2.KeyPair, securityGroup *ec2.SecurityGroup, subnetID pulumi.StringOutput) (*HeadscaleResult, error) {
	if !cfg.Create {
		return nil, nil // Not creating a server
	}

	// Default values
	namespace := cfg.Namespace
	if namespace == "" {
		namespace = "kubernetes"
	}

	serverName := "headscale-server"
	if cfg.Domain != "" {
		serverName = fmt.Sprintf("headscale-%s", cfg.Domain)
	}

	// Headscale installation and configuration script
	installScript := m.generateHeadscaleCloudInit(namespace, cfg.Domain)

	switch cfg.Provider {
	case "aws":
		return m.createAWSHeadscale(cfg, serverName, namespace, sshKeyPair, securityGroup, subnetID, installScript)
	default:
		return nil, fmt.Errorf("unsupported provider for Headscale: %s (supported: aws)", cfg.Provider)
	}
}

func (m *HeadscaleManager) generateHeadscaleCloudInit(namespace, domain string) pulumi.StringOutput {
	// Note: serverURL is dynamically set using PUBLIC_IP in the script
	// If domain is provided, it would be used for HTTPS setup (not implemented yet)
	_ = domain // Reserved for future HTTPS/Let's Encrypt support

	script := fmt.Sprintf(`#!/bin/bash
set -e

echo "=== Starting Headscale Installation ==="

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

# Get public IP
PUBLIC_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 || curl -s ifconfig.me)
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

# Wait for Headscale to be ready
echo "Waiting for Headscale to start..."
sleep 10

# Create namespace
echo "Creating namespace: %s"
headscale namespaces create %s || true

# Create API key and save it
echo "Creating API key..."
API_KEY=$(headscale apikeys create --expiration 365d 2>/dev/null | tail -1)
echo "$API_KEY" > /root/headscale-api-key

# Create pre-auth key for nodes (reusable, 30 day expiration)
echo "Creating pre-auth key..."
AUTH_KEY=$(headscale preauthkeys create --namespace %s --reusable --expiration 720h 2>/dev/null | tail -1)
echo "$AUTH_KEY" > /root/headscale-auth-key

# Save server info
echo "http://${PUBLIC_IP}:8080" > /root/headscale-url

# Create a script to get credentials
cat > /root/get-headscale-credentials.sh <<'SCRIPT'
#!/bin/bash
echo "=== Headscale Credentials ==="
echo "Server URL: $(cat /root/headscale-url)"
echo "API Key: $(cat /root/headscale-api-key)"
echo "Auth Key: $(cat /root/headscale-auth-key)"
echo "Namespace: %s"
SCRIPT
chmod +x /root/get-headscale-credentials.sh

echo "=== Headscale Installation Complete ==="
echo "Server URL: http://${PUBLIC_IP}:8080"
echo "Namespace: %s"
echo "Credentials saved to /root/"
`, namespace, namespace, namespace, namespace, namespace)

	return pulumi.String(script).ToStringOutput()
}

func (m *HeadscaleManager) createAWSHeadscale(cfg *config.TailscaleConfig, serverName, namespace string, sshKeyPair *ec2.KeyPair, securityGroup *ec2.SecurityGroup, subnetID pulumi.StringOutput, installScript pulumi.StringOutput) (*HeadscaleResult, error) {
	// Default size if not specified
	instanceType := cfg.Size
	if instanceType == "" {
		instanceType = "t3.small"
	}

	// Default region
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	// Create security group for Headscale if not provided
	var sgID pulumi.StringOutput
	if securityGroup != nil {
		sgID = securityGroup.ID().ToStringOutput()
	} else {
		// Create a dedicated security group for Headscale
		headscaleSG, err := ec2.NewSecurityGroup(m.ctx, "headscale-sg", &ec2.SecurityGroupArgs{
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
				"Name": pulumi.String("headscale-sg"),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create Headscale security group: %w", err)
		}
		sgID = headscaleSG.ID().ToStringOutput()
	}

	// Find Ubuntu 22.04 AMI
	ami, err := ec2.LookupAmi(m.ctx, &ec2.LookupAmiArgs{
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
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find Ubuntu AMI: %w", err)
	}

	// Create EC2 instance args
	instanceArgs := &ec2.InstanceArgs{
		InstanceType:             pulumi.String(instanceType),
		Ami:                      pulumi.String(ami.Id),
		KeyName:                  sshKeyPair.KeyName,
		VpcSecurityGroupIds:      pulumi.StringArray{sgID},
		AssociatePublicIpAddress: pulumi.Bool(true),
		UserData:                 installScript,
		Tags: pulumi.StringMap{
			"Name": pulumi.String(serverName),
			"Role": pulumi.String("headscale"),
		},
		RootBlockDevice: &ec2.InstanceRootBlockDeviceArgs{
			VolumeSize: pulumi.Int(20),
			VolumeType: pulumi.String("gp3"),
		},
	}

	// Add subnet if provided (subnetID is set by the caller)
	// Note: Empty StringOutput is fine, Pulumi handles it gracefully
	instanceArgs.SubnetId = subnetID

	// Create Headscale EC2 instance
	instance, err := ec2.NewInstance(m.ctx, serverName, instanceArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create Headscale instance: %w", err)
	}

	// Build the API URL
	apiURL := instance.PublicIp.ApplyT(func(ip string) string {
		return fmt.Sprintf("http://%s:8080", ip)
	}).(pulumi.StringOutput)

	// Export server information
	secrets.Export(m.ctx, "headscale_server_id", instance.ID())
	secrets.Export(m.ctx, "headscale_server_ip", instance.PublicIp)
	secrets.Export(m.ctx, "headscale_server_name", pulumi.String(serverName))
	secrets.Export(m.ctx, "headscale_api_url", apiURL)
	secrets.Export(m.ctx, "headscale_namespace", pulumi.String(namespace))

	return &HeadscaleResult{
		Provider:   "aws",
		ServerID:   instance.ID(),
		ServerIP:   instance.PublicIp,
		ServerName: serverName,
		APIURL:     apiURL,
		Namespace:  namespace,
		Port:       8080,
	}, nil
}

// GetHeadscaleCredentials returns a command to retrieve Headscale credentials from the server
func (m *HeadscaleManager) GetHeadscaleCredentials() string {
	return `#!/bin/bash
# Run this on the Headscale server to get credentials
echo "=== Headscale Credentials ==="
echo "Server URL: $(cat /root/headscale-url)"
echo "API Key: $(cat /root/headscale-api-key)"
echo "Auth Key: $(cat /root/headscale-auth-key)"
`
}

// GetTailscaleJoinCommand returns the command for a node to join the Tailscale network
func (m *HeadscaleManager) GetTailscaleJoinCommand(headscaleURL, authKey, hostname string) string {
	return fmt.Sprintf(`#!/bin/bash
# Install Tailscale
curl -fsSL https://tailscale.com/install.sh | sh

# Join the network via Headscale
tailscale up --login-server=%s --authkey=%s --hostname=%s --accept-routes
`, headscaleURL, authKey, hostname)
}
