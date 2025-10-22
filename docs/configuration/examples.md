# 🦥 Configuration Examples

Real-world cluster configurations for every scenario. Copy, paste, and customize at your own pace!

---

## Simple Single-Cloud Cluster

Perfect for development or small projects. 🦥

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: simple-dev
  labels:
    environment: development

spec:
  providers:
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}  # 🦥 From environment
      region: nyc3

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1

  nodePools:
    - name: all-in-one
      provider: digitalocean
      count: 1  # 🦥 Single node for dev
      roles: [master, worker]
      size: s-2vcpu-4gb
```

**What you get:**

- 1 node serving as both master and worker
- No VPN (single node doesn't need it)
- Perfect for testing
- Cost: ~$24/month

---

## Production HA Multi-Cloud

High availability across multiple clouds. 🦥

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: production-ha
  labels:
    environment: production
    tier: critical

spec:
  providers:
    # DigitalOcean for masters
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}
      region: nyc3
      vpc:
        create: true
        cidr: 10.10.0.0/16

    # Linode for masters and workers
    linode:
      enabled: true
      token: ${LINODE_TOKEN}
      region: us-east
      vpc:
        create: true
        cidr: 10.11.0.0/16

  # 🦥 Secure VPN mesh
  network:
    wireguard:
      create: true
      meshNetworking: true
      subnet: 10.8.0.0/24
      port: 51820

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1
    rke2:
      secretsEncryption: true  # 🦥 Encrypt at rest
      snapshotScheduleCron: "0 */6 * * *"  # Backup every 6 hours
      profiles:
        - cis-1.6  # CIS security benchmark

  nodePools:
    # Masters across clouds for HA
    - name: do-masters
      provider: digitalocean
      count: 1
      roles: [master]
      size: s-2vcpu-4gb
      tags: [master, production]

    - name: linode-masters
      provider: linode
      count: 2  # 🦥 3 total masters (quorum)
      roles: [master]
      size: g6-standard-2
      tags: [master, production]

    # Workers for application workloads
    - name: do-workers
      provider: digitalocean
      count: 2
      roles: [worker]
      size: s-4vcpu-8gb  # 🦥 More resources for apps
      tags: [worker, production]

    - name: linode-workers
      provider: linode
      count: 2
      roles: [worker]
      size: g6-standard-4
      tags: [worker, production]

  # 🦥 Bastion for secure access
  security:
    bastion:
      enabled: true
      provider: digitalocean
      size: s-1vcpu-1gb
      allowedIPs:
        - "203.0.113.0/24"  # Your office IP range

  # 🦥 GitOps with ArgoCD
  addons:
    gitops:
      enabled: true
      repository: https://github.com/yourorg/k8s-gitops
      branch: main
```

**What you get:**

- 3 master nodes (1 DO + 2 Linode) for HA
- 4 worker nodes across both clouds
- WireGuard VPN mesh
- Encrypted secrets
- CIS security benchmarks
- Automated backups every 6 hours
- Bastion host for secure access
- ArgoCD for GitOps
- Cost: ~$240/month

---

## Cost-Optimized Cluster

Maximum value for minimum spend. 🦥

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: budget-friendly
  labels:
    environment: staging
    cost-optimized: "true"

spec:
  providers:
    # Linode (generally cheaper)
    linode:
      enabled: true
      token: ${LINODE_TOKEN}
      region: us-east
      vpc:
        create: true
        cidr: 10.20.0.0/16

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1

  nodePools:
    # Single master (not HA, but cheap!)
    - name: master
      provider: linode
      count: 1
      roles: [master]
      size: g6-nanode-1  # 🦥 Smallest size: $5/month
      tags: [master, staging]

    # 2 small workers
    - name: workers
      provider: linode
      count: 2
      roles: [worker]
      size: g6-nanode-1  # 🦥 Also $5/month each
      tags: [worker, staging]
```

**What you get:**

- 1 master + 2 workers
- Single cloud (no VPN overhead)
- Basic Kubernetes functionality
- Perfect for staging/testing
- Cost: ~$15/month

---

## GPU Workloads Cluster

For ML/AI and GPU-intensive workloads. 🦥

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: gpu-cluster
  labels:
    environment: ml-training
    workload: gpu

spec:
  providers:
    # DigitalOcean for control plane
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}
      region: nyc3
      vpc:
        create: true
        cidr: 10.30.0.0/16

    # Linode for GPU nodes
    linode:
      enabled: true
      token: ${LINODE_TOKEN}
      region: us-east
      vpc:
        create: true
        cidr: 10.31.0.0/16

  network:
    wireguard:
      create: true
      meshNetworking: true

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1

  nodePools:
    # Control plane on DO
    - name: masters
      provider: digitalocean
      count: 3
      roles: [master]
      size: s-2vcpu-4gb

    # CPU workers for system services
    - name: cpu-workers
      provider: digitalocean
      count: 2
      roles: [worker]
      size: s-4vcpu-8gb
      labels:
        node-type: cpu  # 🦥 Label for scheduling

    # GPU workers for ML workloads
    - name: gpu-workers
      provider: linode
      count: 2
      roles: [worker]
      size: g1-gpu-rtx6000-1  # 🦥 RTX 6000 GPU
      labels:
        node-type: gpu
        nvidia.com/gpu: "true"
      taints:
        - key: nvidia.com/gpu
          value: "true"
          effect: NoSchedule  # 🦥 Only GPU pods here
```

**Example GPU pod:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-training
spec:
  nodeSelector:
    node-type: gpu  # 🦥 Schedule on GPU nodes
  tolerations:
    - key: nvidia.com/gpu
      operator: Exists
  containers:
    - name: pytorch
      image: pytorch/pytorch:latest
      resources:
        limits:
          nvidia.com/gpu: 1  # 🦥 Request GPU
```

---

## Edge Computing Cluster

Distributed edge locations. 🦥

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: edge-distributed
  labels:
    environment: edge
    topology: distributed

spec:
  providers:
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}
      regions:  # 🦥 Multiple regions!
        - nyc3
        - sfo3
        - ams3

    linode:
      enabled: true
      token: ${LINODE_TOKEN}
      regions:
        - us-east
        - us-west
        - eu-central

  network:
    wireguard:
      create: true
      meshNetworking: true
      subnet: 10.8.0.0/24

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1

  nodePools:
    # Masters in primary region
    - name: central-masters
      provider: digitalocean
      region: nyc3
      count: 3
      roles: [master]
      size: s-2vcpu-4gb

    # Edge workers in NYC
    - name: nyc-edge
      provider: digitalocean
      region: nyc3
      count: 2
      roles: [worker]
      size: s-2vcpu-4gb
      labels:
        edge-location: nyc  # 🦥 Location-aware scheduling

    # Edge workers in SF
    - name: sfo-edge
      provider: digitalocean
      region: sfo3
      count: 2
      roles: [worker]
      size: s-2vcpu-4gb
      labels:
        edge-location: sfo

    # Edge workers in Amsterdam
    - name: ams-edge
      provider: digitalocean
      region: ams3
      count: 2
      roles: [worker]
      size: s-2vcpu-4gb
      labels:
        edge-location: ams

    # Edge workers in Asia
    - name: asia-edge
      provider: linode
      region: ap-south
      count: 2
      roles: [worker]
      size: g6-standard-2
      labels:
        edge-location: asia
```

**Deploy geographically:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cdn-cache
spec:
  replicas: 8
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: app
                      operator: In
                      values: [cdn-cache]
                topologyKey: edge-location  # 🦥 Spread across locations
```

---

## Development Team Cluster

For collaborative development teams. 🦥

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: dev-team
  labels:
    environment: development
    team: engineering

spec:
  providers:
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}
      region: nyc3
      vpc:
        create: true
        cidr: 10.50.0.0/16

  network:
    wireguard:
      create: true
      meshNetworking: true
      clients:  # 🦥 VPN for developers
        - name: alice-laptop
          allowedIPs: [10.8.0.100/32]
        - name: bob-laptop
          allowedIPs: [10.8.0.101/32]
        - name: charlie-laptop
          allowedIPs: [10.8.0.102/32]

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1
    rke2:
      profiles:
        - cis-1.6  # 🦥 Security even in dev!

  nodePools:
    - name: masters
      provider: digitalocean
      count: 1  # Single master for dev
      roles: [master]
      size: s-2vcpu-4gb

    - name: workers
      provider: digitalocean
      count: 3
      roles: [worker]
      size: s-4vcpu-8gb  # 🦥 Enough for multiple apps
      labels:
        node-type: general

  # 🦥 Pre-install development tools
  addons:
    gitops:
      enabled: true
      repository: https://github.com/yourorg/dev-cluster-config
      applications:
        - name: dev-namespace-creator
          path: namespaces/
        - name: ingress-nginx
          path: ingress/
        - name: cert-manager
          path: cert-manager/
        - name: monitoring
          path: monitoring/

  # 🦥 DNS for easy access
  dns:
    enabled: true
    domain: dev.yourcompany.com
    provider: digitalocean
    records:
      - name: "*.dev"
        type: A
        value: "${INGRESS_IP}"
```

**Team members can:**

- Connect via VPN to access internal services
- Deploy to their own namespaces
- Use `*.dev.yourcompany.com` domains
- Share the cluster without conflicts

---

## Compliance-First Cluster

For regulated industries (healthcare, finance). 🦥

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: compliance-cluster
  labels:
    environment: production
    compliance: hipaa-pci

spec:
  providers:
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}
      region: nyc3
      vpc:
        create: true
        cidr: 10.100.0.0/16

  network:
    wireguard:
      create: true
      meshNetworking: true
      # 🦥 Strong encryption
      allowedCipherSuites:
        - TLS_AES_256_GCM_SHA384

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1
    rke2:
      secretsEncryption: true  # 🦥 Required for compliance
      snapshotScheduleCron: "0 */4 * * *"  # Backup every 4 hours
      snapshotRetention: 72  # Keep 72 backups (2 weeks)
      auditLogEnabled: true  # 🦥 Audit all API calls
      profiles:
        - cis-1.6  # CIS benchmarks
      podSecurityPolicy: restricted  # Strictest policy

  nodePools:
    - name: masters
      provider: digitalocean
      count: 3  # 🦥 HA required
      roles: [master]
      size: s-4vcpu-8gb
      encrypted: true  # Encrypted volumes
      tags: [master, compliance]

    - name: workers
      provider: digitalocean
      count: 4
      roles: [worker]
      size: s-8vcpu-16gb
      encrypted: true  # 🦥 All volumes encrypted
      tags: [worker, compliance]

  # 🦥 Strict security controls
  security:
    bastion:
      enabled: true
      provider: digitalocean
      size: s-1vcpu-1gb
      allowedIPs:
        - "203.0.113.0/24"  # Only from office
      mfaRequired: true

    networkPolicies:
      enabled: true
      defaultDeny: true  # 🦥 Deny all, allow explicitly

    podSecurityStandards:
      enforce: restricted
      audit: restricted
      warn: restricted

  # 🦥 Monitoring and alerting
  addons:
    monitoring:
      enabled: true
      retentionDays: 90  # Keep logs for compliance
      alerts:
        - unauthorizedAccess
        - configChanges
        - podSecurityViolations
```

**Compliance features:**

- ✅ Encrypted volumes
- ✅ Encrypted secrets at rest
- ✅ Audit logging enabled
- ✅ CIS benchmarks enforced
- ✅ Network policies (default deny)
- ✅ Pod security standards (restricted)
- ✅ Regular automated backups
- ✅ Bastion with MFA
- ✅ Monitoring and alerting

---

## Template Variables

You can use environment variables in your configs:

```yaml
spec:
  providers:
    digitalocean:
      token: ${DIGITALOCEAN_TOKEN}  # 🦥 From environment
      region: ${DO_REGION:-nyc3}    # 🦥 Default to nyc3

  network:
    wireguard:
      port: ${VPN_PORT:-51820}      # 🦥 Default to 51820
```

Set before deploying:

```bash
export DIGITALOCEAN_TOKEN="dop_v1_..."
export DO_REGION="sfo3"
export VPN_PORT="51821"

sloth-kubernetes deploy --config cluster.yaml  # 🦥
```

---

## Tips for Writing Configs

!!! tip "Start Small 🦥"
    Begin with a simple config and add features gradually. Don't rush!

!!! warning "Test in Dev First 🦥"
    Always test new configurations in development before production.

!!! success "Version Control 🦥"
    Keep your configs in Git for tracking and rollback capability.

```bash
# Good structure
k8s-clusters/
├── production.yaml
├── staging.yaml
├── development.yaml
└── examples/
    ├── simple.yaml
    ├── ha.yaml
    └── multi-cloud.yaml
```

---

!!! quote "Sloth Wisdom 🦥"
    *"A well-configured cluster is worth the wait. Take your time, get it right!"*

**Need more examples?** Check out the [examples directory](https://github.com/yourusername/sloth-kubernetes/tree/main/examples) in the repo! 🦥
