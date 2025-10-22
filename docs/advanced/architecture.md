# 🦥 Architecture

Deep dive into how Sloth Kubernetes works under the hood. For the curious sloths!

---

## System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                  🦥 Sloth Kubernetes CLI                     │
│                    (Single Binary)                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌────────────────┐  ┌──────────────┐  ┌────────────────┐  │
│  │ Config Parser  │  │ Orchestrator │  │ State Manager  │  │
│  │   (YAML)   🦥  │→ │   (Pulumi)🦥 │→ │   (Local)  🦥  │  │
│  └────────────────┘  └──────────────┘  └────────────────┘  │
│           │                  │                   │          │
│           └──────────────────┴───────────────────┘          │
│                              ↓                               │
│              ┌──────────────────────────────┐               │
│              │   Pulumi Automation API      │               │
│              │   (Embedded, No CLI)    🦥   │               │
│              └──────────────────────────────┘               │
│                              ↓                               │
│         ┌────────────────────┴────────────────────┐         │
│         ↓                                          ↓         │
│  ┌─────────────┐                          ┌─────────────┐   │
│  │ Cloud APIs  │                          │   SSH/RKE2  │   │
│  │  (DO/Linode)│                          │  Installer  │   │
│  │     🦥      │                          │     🦥      │   │
│  └─────────────┘                          └─────────────┘   │
└─────────────────────────────────────────────────────────────┘
                 ↓                                ↓
    ┌────────────────────────┐      ┌──────────────────────┐
    │  Cloud Resources       │      │  Kubernetes Cluster  │
    │  • VPCs                │      │  • RKE2 Installed    │
    │  • Droplets/Instances  │      │  • WireGuard VPN     │
    │  • DNS Records     🦥  │      │  • Encrypted     🦥  │
    └────────────────────────┘      └──────────────────────┘
```

---

## Core Components

### 1. Configuration Parser

**Location:** `pkg/config/`

Parses YAML config files into Go structs:

```go
// Main config structure
type ClusterConfig struct {
    APIVersion string                  `yaml:"apiVersion"`
    Kind       string                  `yaml:"kind"`
    Metadata   ClusterMetadata         `yaml:"metadata"`
    Spec       ClusterSpec             `yaml:"spec"`
}

// Providers config
type ProvidersConfig struct {
    DigitalOcean *DigitalOceanProvider `yaml:"digitalocean"`
    Linode       *LinodeProvider       `yaml:"linode"`
}

// Node pool config
type NodePool struct {
    Name     string            `yaml:"name"`
    Provider string            `yaml:"provider"`
    Count    int               `yaml:"count"`
    Roles    []string          `yaml:"roles"`
    Size     string            `yaml:"size"`
    Labels   map[string]string `yaml:"labels"`
    Taints   []Taint           `yaml:"taints"`
}
```

**Validation:**

```go
// Validates configuration
func (c *ClusterConfig) Validate() error {
    // Check providers
    if !c.Spec.Providers.DigitalOcean.Enabled &&
       !c.Spec.Providers.Linode.Enabled {
        return errors.New("at least one provider must be enabled")
    }

    // Validate node pools
    masterCount := 0
    for _, pool := range c.Spec.NodePools {
        if contains(pool.Roles, "master") {
            masterCount += pool.Count
        }
    }

    // 🦥 Odd number of masters for etcd quorum
    if masterCount > 1 && masterCount%2 == 0 {
        return errors.New("master count must be odd for HA")
    }

    return nil
}
```

---

### 2. Orchestrator

**Location:** `internal/orchestrator/`

Coordinates the deployment process:

```go
type ClusterOrchestrator struct {
    config        *config.ClusterConfig
    pulumiStack   *auto.Stack
    components    []Component
}

// Main deployment flow
func (o *ClusterOrchestrator) Deploy(ctx context.Context) error {
    // 1. Initialize Pulumi stack 🦥
    if err := o.initStack(ctx); err != nil {
        return err
    }

    // 2. Create cloud resources (VPCs, nodes)
    if err := o.provisionInfrastructure(ctx); err != nil {
        return err
    }

    // 3. Setup WireGuard VPN
    if o.config.Spec.Network.WireGuard.Create {
        if err := o.setupVPN(ctx); err != nil {
            return err
        }
    }

    // 4. Install RKE2 on nodes
    if err := o.installKubernetes(ctx); err != nil {
        return err
    }

    // 5. Configure cluster
    if err := o.configureCluster(ctx); err != nil {
        return err
    }

    // 6. Bootstrap addons (GitOps, monitoring, etc)
    if err := o.bootstrapAddons(ctx); err != nil {
        return err
    }

    return nil
}
```

**Component Architecture:**

```go
// All infrastructure components implement this interface
type Component interface {
    Create(ctx context.Context) error
    Update(ctx context.Context) error
    Delete(ctx context.Context) error
    Status(ctx context.Context) (ComponentStatus, error)
}

// Components
var components = []Component{
    &VPCComponent{},           // 🦥 Create VPCs
    &BastionComponent{},       // 🦥 Create bastion host
    &VPNComponent{},           // 🦥 Deploy WireGuard
    &NodeProvisioningComponent{}, // 🦥 Provision nodes
    &RKE2InstallerComponent{}, // 🦥 Install Kubernetes
    &DNSComponent{},           // 🦥 Configure DNS
    &FirewallComponent{},      // 🦥 Setup firewall rules
}
```

---

### 3. State Management

**Location:** `internal/orchestrator/state_manager.go`

Uses Pulumi for infrastructure state:

```go
type StateManager struct {
    stackName   string
    projectName string
    stateDir    string
}

// Initialize Pulumi stack (no CLI required!)
func (sm *StateManager) InitStack(ctx context.Context) (*auto.Stack, error) {
    // Create local backend (file-based state)
    backend := fmt.Sprintf("file://%s", sm.stateDir)

    // Create stack using Pulumi Automation API 🦥
    stack, err := auto.UpsertStackLocalSource(ctx, sm.stackName,
        sm.projectName, backend)
    if err != nil {
        return nil, err
    }

    return stack, nil
}

// Get current state
func (sm *StateManager) GetState(ctx context.Context) (*StateSnapshot, error) {
    export, err := sm.stack.Export(ctx)
    if err != nil {
        return nil, err
    }

    // Parse state JSON
    var snapshot StateSnapshot
    if err := json.Unmarshal(export.Deployment, &snapshot); err != nil {
        return nil, err
    }

    return &snapshot, nil
}
```

**State Directory Structure:**

```
~/.sloth/
├── stacks/
│   ├── my-cluster/
│   │   ├── .pulumi/
│   │   │   ├── stacks/
│   │   │   │   └── my-cluster.json  # 🦥 Stack state
│   │   │   └── backups/
│   │   │       └── *.json.bak       # 🦥 Automatic backups
│   │   └── Pulumi.yaml
│   └── staging-cluster/
│       └── ...
└── config/
    └── credentials.json  # 🦥 Encrypted API tokens (optional)
```

---

### 4. Cloud Providers

#### DigitalOcean Integration

**Location:** `pkg/providers/digitalocean/`

```go
type DigitalOceanProvider struct {
    client *godo.Client
    config *config.DigitalOceanProvider
}

// Create VPC
func (p *DigitalOceanProvider) CreateVPC(ctx *pulumi.Context) error {
    vpc, err := digitalocean.NewVpc(ctx, "vpc", &digitalocean.VpcArgs{
        Name:   pulumi.String(p.config.VPC.Name),
        Region: pulumi.String(p.config.Region),
        IpRange: pulumi.String(p.config.VPC.CIDR),
    })
    // ... 🦥
}

// Create Droplet
func (p *DigitalOceanProvider) CreateDroplet(ctx *pulumi.Context,
    pool *config.NodePool) error {

    droplet, err := digitalocean.NewDroplet(ctx, pool.Name,
        &digitalocean.DropletArgs{
            Name:   pulumi.String(pool.Name),
            Size:   pulumi.String(pool.Size),
            Image:  pulumi.String("ubuntu-22-04-x64"),
            Region: pulumi.String(p.config.Region),
            VpcUuid: vpc.ID(), // 🦥 Attach to VPC
            SshKeys: pulumi.StringArray{sshKeyID},
        })
    // ... 🦥
}
```

#### Linode Integration

**Location:** `pkg/providers/linode/`

```go
type LinodeProvider struct {
    client *linodego.Client
    config *config.LinodeProvider
}

// Create Instance
func (p *LinodeProvider) CreateInstance(ctx *pulumi.Context,
    pool *config.NodePool) error {

    instance, err := linode.NewInstance(ctx, pool.Name,
        &linode.InstanceArgs{
            Label:  pulumi.String(pool.Name),
            Type:   pulumi.String(pool.Size),
            Image:  pulumi.String("linode/ubuntu22.04"),
            Region: pulumi.String(p.config.Region),
            // ... 🦥
        })
    // ...
}
```

---

### 5. WireGuard VPN

**Location:** `pkg/vpn/wireguard.go`

Automatic VPN mesh networking:

```go
type WireGuardManager struct {
    serverNode  *Node
    clientNodes []*Node
    config      *config.WireGuardConfig
}

// Deploy VPN server
func (wg *WireGuardManager) DeployServer(ctx context.Context) error {
    // 1. Generate server keys 🦥
    privateKey, publicKey, err := wg.generateKeyPair()

    // 2. Install WireGuard on server node
    installScript := `
        apt-get update
        apt-get install -y wireguard

        # Configure interface 🦥
        cat > /etc/wireguard/wg0.conf <<EOF
[Interface]
PrivateKey = ` + privateKey + `
Address = 10.8.0.1/24
ListenPort = 51820
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT
EOF

        # Start VPN 🦥
        systemctl enable wg-quick@wg0
        systemctl start wg-quick@wg0
    `

    // 3. Execute via SSH
    if err := wg.executeSSH(ctx, wg.serverNode, installScript); err != nil {
        return err
    }

    // 4. Configure firewall
    if err := wg.configureFirewall(ctx); err != nil {
        return err
    }

    return nil
}

// Configure VPN client on each node
func (wg *WireGuardManager) ConfigureClient(ctx context.Context,
    node *Node) error {

    // Generate client keys
    privateKey, publicKey, _ := wg.generateKeyPair()

    // Assign VPN IP
    vpnIP := wg.getNextAvailableIP()

    // Configure client
    clientConfig := fmt.Sprintf(`
[Interface]
PrivateKey = %s
Address = %s/24

[Peer]
PublicKey = %s
Endpoint = %s:51820
AllowedIPs = 10.8.0.0/24, %s, %s
PersistentKeepalive = 25
`, privateKey, vpnIP, wg.serverPublicKey,
   wg.serverNode.PublicIP, vpc1CIDR, vpc2CIDR)

    // Install on node 🦥
    return wg.installClientConfig(ctx, node, clientConfig)
}
```

**VPN Mesh Topology:**

```
                  🦥 VPN Server 🦥
                   (10.8.0.1)
                  203.0.113.10
                       │
       ┌───────────────┼───────────────┐
       │               │               │
   DO Master 1     DO Master 2    Linode Master 1
   (10.8.0.2)      (10.8.0.3)      (10.8.0.4)
   10.10.1.5       10.10.1.6       10.11.1.5
       │               │               │
       └───────────────┴───────────────┘
                       │
              All nodes communicate
           via encrypted VPN tunnel 🔐
```

---

### 6. RKE2 Installer

**Location:** `pkg/kubernetes/rke2_installer.go`

Installs and configures RKE2:

```go
type RKE2Installer struct {
    config    *config.KubernetesConfig
    nodes     []*Node
    masterIPs []string
}

// Install on master nodes
func (r *RKE2Installer) InstallMasters(ctx context.Context) error {
    for i, master := range r.getMasterNodes() {
        if i == 0 {
            // First master initializes cluster 🦥
            err := r.installFirstMaster(ctx, master)
        } else {
            // Other masters join existing cluster
            err := r.installAdditionalMaster(ctx, master)
        }

        if err != nil {
            return err
        }
    }
    return nil
}

// Install first master
func (r *RKE2Installer) installFirstMaster(ctx context.Context,
    node *Node) error {

    config := fmt.Sprintf(`
# RKE2 Server Config 🦥
write-kubeconfig-mode: "0644"
tls-san:
  - %s
  - %s
cluster-cidr: "10.42.0.0/16"
service-cidr: "10.43.0.0/16"
`, node.PublicIP, node.PrivateIP)

    // Add secrets encryption if enabled
    if r.config.RKE2.SecretsEncryption {
        config += `
secrets-encryption: true
`
    }

    // Add CIS profile if configured
    if len(r.config.RKE2.Profiles) > 0 {
        config += fmt.Sprintf(`
profile: %s
`, r.config.RKE2.Profiles[0])
    }

    // Write config and install 🦥
    installScript := fmt.Sprintf(`
        # Write config
        mkdir -p /etc/rancher/rke2
        cat > /etc/rancher/rke2/config.yaml <<'EOF'
%s
EOF

        # Install RKE2
        curl -sfL https://get.rke2.io | INSTALL_RKE2_VERSION=%s sh -

        # Enable and start 🦥
        systemctl enable rke2-server
        systemctl start rke2-server

        # Wait for startup
        until systemctl is-active rke2-server; do
            sleep 5
        done

        # Get join token for other nodes
        cat /var/lib/rancher/rke2/server/node-token
    `, config, r.config.Version)

    output, err := r.executeSSH(ctx, node, installScript)
    if err != nil {
        return err
    }

    // Save join token 🦥
    r.joinToken = strings.TrimSpace(output)

    return nil
}

// Install worker nodes
func (r *RKE2Installer) InstallWorkers(ctx context.Context) error {
    workerConfig := fmt.Sprintf(`
server: https://%s:9345
token: %s
`, r.masterIPs[0], r.joinToken)

    installScript := fmt.Sprintf(`
        mkdir -p /etc/rancher/rke2
        cat > /etc/rancher/rke2/config.yaml <<'EOF'
%s
EOF

        curl -sfL https://get.rke2.io | INSTALL_RKE2_TYPE=agent \
            INSTALL_RKE2_VERSION=%s sh -

        systemctl enable rke2-agent
        systemctl start rke2-agent  # 🦥
    `, workerConfig, r.config.Version)

    // Install on all workers in parallel
    errChan := make(chan error, len(r.getWorkerNodes()))
    for _, worker := range r.getWorkerNodes() {
        go func(node *Node) {
            errChan <- r.executeSSH(ctx, node, installScript)
        }(worker)
    }

    // Wait for all 🦥
    for range r.getWorkerNodes() {
        if err := <-errChan; err != nil {
            return err
        }
    }

    return nil
}
```

---

### 7. DNS Management

**Location:** `pkg/dns/manager.go`

Automatic DNS record creation:

```go
type DNSManager struct {
    provider string
    domain   string
    records  []*config.DNSRecord
}

// Create DNS records for nodes
func (d *DNSManager) CreateRecords(ctx *pulumi.Context,
    nodes []*Node) error {

    for _, node := range nodes {
        // Create A record for each node 🦥
        _, err := digitalocean.NewDnsRecord(ctx,
            fmt.Sprintf("%s-record", node.Name),
            &digitalocean.DnsRecordArgs{
                Domain: pulumi.String(d.domain),
                Type:   pulumi.String("A"),
                Name:   pulumi.String(node.Name),
                Value:  pulumi.String(node.PublicIP),
                Ttl:    pulumi.Int(300),
            })

        if err != nil {
            return err
        }
    }

    // Create wildcard record for ingress 🦥
    if d.records.IngressWildcard {
        ingressIP := d.getIngressIP(ctx)
        _, err := digitalocean.NewDnsRecord(ctx, "wildcard-ingress",
            &digitalocean.DnsRecordArgs{
                Domain: pulumi.String(d.domain),
                Type:   pulumi.String("A"),
                Name:   pulumi.String("*"),
                Value:  pulumi.String(ingressIP),
                Ttl:    pulumi.Int(300),
            })
    }

    return nil
}
```

---

## Deployment Flow

Detailed step-by-step process:

```
1. Parse Configuration 🦥
   ├─ Read YAML file
   ├─ Validate structure
   ├─ Expand environment variables
   └─ Validate semantics

2. Initialize Pulumi Stack 🦥
   ├─ Create local backend
   ├─ Load or create stack
   └─ Configure providers

3. Create VPCs 🦥
   ├─ DigitalOcean VPC (if enabled)
   └─ Linode VPC (if enabled)

4. Deploy Bastion (if enabled) 🦥
   ├─ Create bastion host
   ├─ Configure SSH access
   └─ Setup port forwarding

5. Deploy WireGuard VPN (if enabled) 🦥
   ├─ Provision VPN server
   ├─ Generate keys
   ├─ Configure firewall
   └─ Start VPN service

6. Provision Nodes 🦥
   ├─ Create master nodes
   ├─ Create worker nodes
   ├─ Attach to VPCs
   ├─ Configure SSH keys
   └─ Wait for nodes to be ready

7. Configure VPN Clients 🦥
   ├─ Generate client keys
   ├─ Assign VPN IPs
   ├─ Install WireGuard on nodes
   └─ Connect to VPN mesh

8. Install Kubernetes 🦥
   ├─ Install first master
   ├─ Get join token
   ├─ Install additional masters
   ├─ Install workers
   └─ Wait for cluster ready

9. Configure DNS (if enabled) 🦥
   ├─ Create node A records
   └─ Create wildcard for ingress

10. Bootstrap Addons 🦥
    ├─ Install NGINX Ingress
    ├─ Install cert-manager
    ├─ Bootstrap ArgoCD (if enabled)
    └─ Apply custom manifests

11. Generate Kubeconfig 🦥
    ├─ Fetch from master
    ├─ Update server address
    └─ Save locally

12. Deployment Complete! 🦥
    └─ Print summary and next steps
```

---

## Security Architecture

### Secrets Encryption

RKE2 can encrypt secrets at rest:

```yaml
# Enabled in config
kubernetes:
  rke2:
    secretsEncryption: true
```

**What happens:**

1. RKE2 generates encryption key
2. Stores key in `/var/lib/rancher/rke2/server/cred/encryption-config.json`
3. All Secrets encrypted before writing to etcd
4. Transparent decryption on read

### Network Security

```
┌─────────────────────────────────────────────────────┐
│              Internet (Untrusted)                    │
└───────────────────┬─────────────────────────────────┘
                    │
                    ↓
         ┌──────────────────────┐
         │  Bastion Host (SSH)  │  🦥 Only entry point
         │    203.0.113.5       │
         └──────────┬───────────┘
                    │
                    ↓
         ┌──────────────────────┐
         │  WireGuard VPN       │  🔐 Encrypted tunnel
         │  10.8.0.1 (Server)   │
         └──────────┬───────────┘
                    │
       ┌────────────┴────────────┐
       ↓                         ↓
┌─────────────┐           ┌─────────────┐
│  VPC (DO)   │           │ VPC (Linode)│
│ 10.10.0.0/16│◄─────────►│ 10.11.0.0/16│  🦥 Private
└─────────────┘   VPN     └─────────────┘
    │                          │
    ↓                          ↓
Master/Worker              Master/Worker
  Nodes                      Nodes
```

### Firewall Rules

Automatically configured:

```go
// Inbound rules
var firewallRules = []FirewallRule{
    // SSH from bastion only
    {
        Port:     22,
        Protocol: "tcp",
        Sources:  []string{bastionIP},
    },
    // WireGuard VPN
    {
        Port:     51820,
        Protocol: "udp",
        Sources:  []string{"0.0.0.0/0"},  // Public
    },
    // Kubernetes API (from VPN only)
    {
        Port:     6443,
        Protocol: "tcp",
        Sources:  []string{"10.8.0.0/24"},  // VPN subnet
    },
    // All traffic within VPC
    {
        Port:     "all",
        Protocol: "all",
        Sources:  []string{vpcCIDR},
    },
}
```

---

## Performance Considerations

### Parallel Operations

Many operations run in parallel:

```go
// Provision nodes in parallel 🦥
func (o *Orchestrator) provisionNodes(pools []*NodePool) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(pools))

    for _, pool := range pools {
        wg.Add(1)
        go func(p *NodePool) {
            defer wg.Done()
            if err := o.createNodePool(p); err != nil {
                errChan <- err
            }
        }(pool)
    }

    wg.Wait()
    close(errChan)

    // Check for errors
    for err := range errChan {
        if err != nil {
            return err
        }
    }

    return nil
}
```

### Resource Limits

Default limits (configurable):

- Max parallel node provisions: 10
- Max parallel RKE2 installs: 5
- SSH connection timeout: 5 minutes
- Total deployment timeout: 30 minutes

---

## Extensibility

### Custom Components

Add your own components:

```go
// pkg/components/custom.go
type CustomComponent struct {
    config *CustomConfig
}

func (c *CustomComponent) Create(ctx context.Context) error {
    // Your custom logic 🦥
    return nil
}

// Register in orchestrator
orchestrator.RegisterComponent(&CustomComponent{})
```

### Hooks

Run custom code at specific points:

```go
// Pre-deployment hook
orchestrator.OnPreDeploy(func(ctx context.Context) error {
    log.Info("🦥 Running custom pre-deployment checks...")
    return validateCustomRequirements()
})

// Post-deployment hook
orchestrator.OnPostDeploy(func(ctx context.Context) error {
    log.Info("🦥 Running custom post-deployment tasks...")
    return setupCustomMonitoring()
})
```

---

## Troubleshooting

### Debug Mode

Enable detailed logging:

```bash
sloth-kubernetes deploy --debug
```

Shows:

- Pulumi resource operations
- SSH commands executed
- API calls to cloud providers
- State changes

### State Inspection

View current state:

```bash
# List all resources
sloth-kubernetes stacks state list

# Export state for debugging
sloth-kubernetes stacks export > state.json
```

### Manual Recovery

If deployment fails partway:

```bash
# Check what was created
sloth-kubernetes stacks state list

# Remove specific failed resource
sloth-kubernetes stacks state delete <urn>

# Resume deployment
sloth-kubernetes deploy --config cluster.yaml
```

---

!!! quote "Sloth Wisdom 🦥"
    *"Understanding the architecture makes you a better operator. Take time to learn!"*

**Want to contribute?** Check out [Contributing Guide](../contributing/development.md) 🦥
