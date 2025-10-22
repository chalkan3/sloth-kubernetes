# 🦥 FAQ

Frequently Asked Questions. Slowly answered!

---

## General Questions

### What is Sloth Kubernetes?

Sloth Kubernetes is a single-binary CLI tool that deploys production-ready Kubernetes clusters across multiple cloud providers (DigitalOcean and Linode) with zero external dependencies. No Pulumi CLI, no Terraform, no kubectl required for deployment! 🦥

### Why "Sloth"?

Because we believe in doing things slowly and correctly! Like a sloth, we take our time to ensure your cluster is deployed properly, securely, and reliably. Good clusters are deployed slowly and surely! 🦥

### Is it really free of external dependencies?

Yes! Unlike other tools that require:
- Pulumi CLI
- Terraform
- kubectl
- Multiple provider CLIs

Sloth Kubernetes embeds everything in one binary using the Pulumi Automation API. Just download and run! 🦥

### What cloud providers are supported?

Currently:
- ✅ **DigitalOcean** (Droplets, VPC, DNS, Load Balancers)
- ✅ **Linode** (Instances, VPC, DNS, NodeBalancers)

Coming soon:
- 🔜 AWS
- 🔜 Azure
- 🔜 GCP
- 🔜 Hetzner

### Can I use just one cloud provider?

Absolutely! While Sloth Kubernetes excels at multi-cloud deployments, you can use a single provider if you prefer. Just enable one provider in your config:

```yaml
spec:
  providers:
    digitalocean:
      enabled: true
    linode:
      enabled: false  # 🦥 Single cloud is fine!
```

---

## Technical Questions

### Does Sloth Kubernetes require Pulumi CLI?

**No!** This is a common question. Sloth Kubernetes uses the **Pulumi Automation API** which is embedded directly in the binary. You never need to install the Pulumi CLI separately. 🦥

### Where is cluster state stored?

By default, state is stored locally in `~/.sloth/stacks/`. Each cluster has its own state directory:

```
~/.sloth/
└── stacks/
    └── my-cluster/
        └── .pulumi/
            └── stacks/
                └── my-cluster.json  # 🦥 Your state
```

You can also use remote backends like S3, Azure Blob, or GCS for team collaboration.

### What Kubernetes distribution is used?

**RKE2** (Rancher Kubernetes Engine 2) by default. RKE2 is:
- Security-focused
- CIS benchmark compliant
- Highly available
- Production-ready
- Actively maintained by SUSE/Rancher 🦥

### Can I use a different Kubernetes distribution?

Currently only RKE2 is supported. We chose RKE2 for its security features and CIS compliance. Other distributions may be added in the future based on community demand! 🦥

### How does the VPN mesh work?

Sloth Kubernetes automatically deploys a **WireGuard VPN** mesh:

1. Creates a VPN server node
2. Generates encryption keys for each node
3. Configures WireGuard on all nodes
4. Sets up routing between VPCs
5. All nodes communicate over encrypted tunnels 🔐

The VPN allows nodes across different clouds to communicate securely as if they were on the same network!

### What ports need to be open?

Minimal ports:

| Port | Protocol | Purpose |
|------|----------|---------|
| 22 | TCP | SSH (from bastion only) |
| 51820 | UDP | WireGuard VPN |
| 6443 | TCP | Kubernetes API (via VPN) |
| 9345 | TCP | RKE2 supervisor API (internal) |

All other communication happens over the encrypted VPN mesh! 🦥

### How long does deployment take?

Typical times:

| Cluster Size | Time | Details |
|--------------|------|---------|
| **Simple (1 node)** | 3-5 min | Single master+worker |
| **Small (3 nodes)** | 5-8 min | 1 master, 2 workers |
| **HA (5+ nodes)** | 8-12 min | 3 masters, 2+ workers |
| **Large (10+ nodes)** | 12-20 min | Multi-cloud HA |

Remember, we're sloths - we take our time! 🦥

---

## Cost Questions

### How much does it cost?

Sloth Kubernetes itself is **free and open source**! You only pay for:

1. **Cloud provider resources** (nodes, VPCs, load balancers)
2. **Bandwidth** (typically included in node pricing)

Example costs:

| Cluster Type | Monthly Cost | Details |
|--------------|--------------|---------|
| **Dev** | $15-30 | 1-3 small nodes |
| **Staging** | $50-100 | 3-5 medium nodes |
| **Production** | $200-500 | 5-10 nodes, HA |

Actual costs vary by provider and region. Check our [examples](configuration/examples.md) for detailed breakdowns! 🦥

### Which provider is cheaper?

Generally:
- **Linode** tends to be slightly cheaper for compute
- **DigitalOcean** has simpler, more predictable pricing
- Both offer free VPCs and bandwidth allowances

Our recommendation: Use both! Multi-cloud diversity is worth the tiny price difference. 🦥

### Are there any hidden costs?

No hidden costs! Watch out for:
- ✅ Bandwidth overage (rare, most providers include 1-5TB free)
- ✅ Load balancers ($10-15/month if you use them)
- ✅ DNS hosting (usually $1-2/month or free)
- ✅ Snapshots/backups (optional, $0.05/GB/month)

### Can I save money?

Yes! Tips:
1. Use smaller node sizes for non-production
2. Share dev/staging clusters across teams
3. Use spot/preemptible instances (coming soon)
4. Enable cluster autoscaling (coming soon)
5. Destroy non-production clusters when not in use 🦥

---

## Security Questions

### Is it secure?

Yes! Security features include:
- ✅ WireGuard VPN mesh (encrypted node communication)
- ✅ RKE2 with CIS benchmarks
- ✅ Secrets encryption at rest
- ✅ Private VPCs (nodes not directly exposed)
- ✅ Bastion host for access control
- ✅ Automatic firewall rules
- ✅ Pod security policies 🦥

### Should I use this in production?

Yes! Sloth Kubernetes is designed for production use. We recommend:
- Use HA configuration (3+ masters)
- Enable secrets encryption
- Use bastion host
- Apply CIS profiles
- Enable monitoring
- Regular backups 🦥

### How do I rotate credentials?

```bash
# Update API tokens
export DIGITALOCEAN_TOKEN="new_token"
export LINODE_TOKEN="new_token"

# Re-deploy (won't recreate nodes)
sloth-kubernetes deploy --config cluster.yaml  # 🦥

# Rotate SSH keys
sloth-kubernetes nodes rotate-keys --pool all
```

### What about compliance (HIPAA, PCI, etc.)?

RKE2 with CIS profiles provides a strong foundation for compliance. Additional requirements:
- Enable secrets encryption ✅
- Enable audit logging ✅
- Use private networking only ✅
- Implement network policies ✅
- Regular backups and retention ✅

See our [Compliance Example](configuration/examples.md#compliance-first-cluster) 🦥

---

## Operational Questions

### How do I add more nodes?

```bash
# Edit config to increase count
nodePools:
  - name: workers
    count: 5  # 🦥 Was 3, now 5

# Deploy (only adds new nodes)
sloth-kubernetes deploy --config cluster.yaml
```

Or use the nodes command:

```bash
sloth-kubernetes nodes add --pool workers --count 2  # 🦥
```

### How do I upgrade Kubernetes?

```bash
# Update version in config
kubernetes:
  version: v1.29.0+rke2r1  # 🦥 New version

# Deploy (rolling upgrade)
sloth-kubernetes deploy --config cluster.yaml
```

Sloth Kubernetes performs rolling upgrades automatically - no downtime! 🦥

### How do I backup my cluster?

RKE2 includes automatic etcd snapshots:

```yaml
kubernetes:
  rke2:
    snapshotScheduleCron: "0 */6 * * *"  # 🦥 Every 6 hours
    snapshotRetention: 30  # Keep 30 snapshots
```

Backups stored on master nodes at `/var/lib/rancher/rke2/server/db/snapshots/`

### How do I restore from backup?

```bash
# SSH to first master
ssh -J bastion-ip master-1

# List snapshots
ls /var/lib/rancher/rke2/server/db/snapshots/

# Restore 🦥
rke2 server --cluster-reset --cluster-reset-restore-path=/var/lib/rancher/rke2/server/db/snapshots/snapshot-name
```

### Can I use kubectl?

Yes! After deployment:

```bash
# Get kubeconfig
sloth-kubernetes kubeconfig > ~/.kube/config

# Use kubectl normally 🦥
kubectl get nodes
kubectl get pods -A
kubectl apply -f manifest.yaml
```

### How do I destroy a cluster?

```bash
# Destroy everything
sloth-kubernetes destroy --config cluster.yaml  # 🦥

# This removes:
# - All nodes
# - VPCs
# - VPN server
# - DNS records
# - Load balancers
```

**Warning:** This is permanent! Make sure you've backed up any data first! 🦥

---

## Troubleshooting

### Deployment is stuck

Check the logs:

```bash
# Enable debug mode
sloth-kubernetes deploy --config cluster.yaml --debug  # 🦥

# Common issues:
# - API rate limits (wait a few minutes)
# - Network connectivity
# - Incorrect API tokens
# - Region not available
```

### Nodes not joining cluster

Verify:

```bash
# Check VPN connectivity
sloth-kubernetes vpn status  # 🦥

# SSH to node and check RKE2
ssh -J bastion-ip node-ip
systemctl status rke2-server  # or rke2-agent
journalctl -u rke2-server -f
```

### Can't connect to cluster

```bash
# Regenerate kubeconfig
sloth-kubernetes kubeconfig > ~/.kube/config  # 🦥

# Verify API server is running
ssh -J bastion-ip master-1
systemctl status rke2-server
```

### Need more help?

- 📖 [Troubleshooting Guide](advanced/troubleshooting.md)
- 💬 [Community Slack](https://sloth-kubernetes.slack.com)
- 🐛 [GitHub Issues](https://github.com/yourusername/sloth-kubernetes/issues)
- 📧 [Email Support](mailto:support@sloth-kubernetes.io)

---

## Contributing

### How can I contribute?

We love contributions! 🦥

- 🐛 Report bugs
- 💡 Suggest features
- 📝 Improve docs
- 🔧 Submit PRs
- 🌟 Star the repo

See our [Contributing Guide](contributing/development.md)

### Can I add a new cloud provider?

Yes! We welcome provider contributions. See [Provider Development Guide](contributing/development.md#adding-providers) 🦥

### How do I request a feature?

[Open an issue](https://github.com/yourusername/sloth-kubernetes/issues/new?template=feature_request.md) with:
- Feature description
- Use case
- Example configuration
- Why it's important

We prioritize features by community demand! 🦥

---

## Philosophy

### Why build this?

Existing tools are complex, require many dependencies, and have steep learning curves. We wanted something simple:
- One binary
- One config file
- Works out of the box 🦥

### What's next for Sloth Kubernetes?

Roadmap:
- 🔜 More cloud providers (AWS, Azure, GCP)
- 🔜 Cluster autoscaling
- 🔜 Cost optimization tools
- 🔜 Multi-region support
- 🔜 Disaster recovery features
- 🔜 Web UI for management

Follow our [Roadmap](https://github.com/yourusername/sloth-kubernetes/projects/1) 🦥

---

!!! quote "Ancient Sloth Proverb 🦥"
    *"Questions are the path to knowledge. Ask slowly, learn surely!"*

**Still have questions?** Join our [community Slack](https://sloth-kubernetes.slack.com) - we're always happy to help! 🦥
