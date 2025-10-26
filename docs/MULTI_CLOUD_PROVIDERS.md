# Multi-Cloud Provider Support

## 🌐 Overview

Sloth Kubernetes now supports **5 major cloud providers**, enabling true multi-cloud Kubernetes deployments with seamless cross-cloud networking via WireGuard VPN mesh.

### Supported Providers

| Provider | Status | Compute | Network | Load Balancer | Auto-Scaling |
|----------|--------|---------|---------|---------------|--------------|
| **DigitalOcean** | ✅ Production Ready | Droplets | VPC | ✅ | Planned |
| **Linode** | ✅ Production Ready | Instances | VPC | ✅ | Planned |
| **AWS** | ✅ Production Ready | EC2 | VPC | 🚧 In Progress | Planned |
| **Google Cloud** | ✅ Production Ready | Compute Engine | VPC Network | 🚧 In Progress | Planned |
| **Microsoft Azure** | ✅ Production Ready | Virtual Machines | Virtual Network | 🚧 In Progress | Planned |

---

## 🚀 Quick Start

### 1. **Single Provider Deployment**

Deploy to AWS only:

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: aws-cluster
spec:
  providers:
    aws:
      enabled: true
      accessKeyId: ${AWS_ACCESS_KEY_ID}
      secretAccessKey: ${AWS_SECRET_ACCESS_KEY}
      region: us-east-1

  nodePools:
    - name: masters
      provider: aws
      count: 3
      roles: [master]
      size: t3.medium
      region: us-east-1
```

### 2. **Multi-Cloud Deployment**

Deploy across multiple providers:

```yaml
spec:
  providers:
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}
      region: nyc3

    aws:
      enabled: true
      region: us-west-2

    gcp:
      enabled: true
      projectId: my-project
      region: us-central1

  network:
    wireguard:
      create: true
      enabled: true
      meshNetworking: true

  nodePools:
    - name: do-masters
      provider: digitalocean
      count: 2
      roles: [master]

    - name: aws-workers
      provider: aws
      count: 3
      roles: [worker]

    - name: gcp-workers
      provider: gcp
      count: 2
      roles: [worker]
```

---

## 📋 Provider Configuration

### **AWS (Amazon Web Services)**

```yaml
providers:
  aws:
    enabled: true
    accessKeyId: ${AWS_ACCESS_KEY_ID}
    secretAccessKey: ${AWS_SECRET_ACCESS_KEY}
    region: us-east-1
    vpc:
      create: true
      cidr: 10.12.0.0/16
      enableDns: true
      enableDnsHostname: true
      internetGateway: true
    keyPair: my-ssh-key  # Optional: auto-generated if not provided
    iamRole: k8s-node-role  # Optional: IAM instance profile
    securityGroups:
      - sg-xxxxx
```

**Supported Regions:**
- North America: `us-east-1`, `us-east-2`, `us-west-1`, `us-west-2`, `ca-central-1`
- Europe: `eu-west-1`, `eu-west-2`, `eu-central-1`, `eu-north-1`
- Asia Pacific: `ap-southeast-1`, `ap-southeast-2`, `ap-northeast-1`, `ap-south-1`
- South America: `sa-east-1`

**Instance Types:**
- General Purpose: `t3.micro`, `t3.small`, `t3.medium`, `t3.large`, `m5.large`, `m5.xlarge`
- Compute Optimized: `c5.large`, `c5.xlarge`, `c5.2xlarge`
- Memory Optimized: `r5.large`, `r5.xlarge`

**AMIs:** Ubuntu 22.04 LTS (auto-selected per region)

---

### **Google Cloud Platform**

```yaml
providers:
  gcp:
    enabled: true
    projectId: ${GCP_PROJECT_ID}
    credentials: ${GCP_CREDENTIALS}  # Path to service account JSON
    region: us-central1
    zone: us-central1-a  # Optional: defaults to {region}-a
    network:
      create: true
      cidr: 10.13.0.0/16
```

**Authentication:**
```bash
# Create service account
gcloud iam service-accounts create k8s-sloth \
  --display-name="Sloth Kubernetes Service Account"

# Grant permissions
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="serviceAccount:k8s-sloth@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/compute.admin"

# Create and download key
gcloud iam service-accounts keys create ~/gcp-key.json \
  --iam-account=k8s-sloth@${PROJECT_ID}.iam.gserviceaccount.com

export GCP_CREDENTIALS=$(cat ~/gcp-key.json)
```

**Supported Regions:**
- Americas: `us-central1`, `us-east1`, `us-west1`, `southamerica-east1`
- Europe: `europe-west1`, `europe-west2`, `europe-north1`
- Asia: `asia-east1`, `asia-southeast1`, `asia-northeast1`

**Machine Types:**
- Shared Core: `e2-micro`, `e2-small`, `e2-medium`
- General Purpose: `e2-standard-2`, `n2-standard-2`, `n2-standard-4`
- Compute Optimized: `c2-standard-4`, `c2-standard-8`

---

### **Microsoft Azure**

```yaml
providers:
  azure:
    enabled: true
    subscriptionId: ${AZURE_SUBSCRIPTION_ID}
    tenantId: ${AZURE_TENANT_ID}
    clientId: ${AZURE_CLIENT_ID}
    clientSecret: ${AZURE_CLIENT_SECRET}
    location: eastus
    resourceGroup: k8s-production-rg
    virtualNetwork:
      create: true
      cidr: 10.14.0.0/16
```

**Authentication:**
```bash
# Login to Azure
az login

# Create service principal
az ad sp create-for-rbac \
  --name sloth-kubernetes \
  --role Contributor \
  --scopes /subscriptions/${SUBSCRIPTION_ID}

# Output will contain:
# - appId (CLIENT_ID)
# - password (CLIENT_SECRET)
# - tenant (TENANT_ID)
```

**Supported Locations:**
- North America: `eastus`, `eastus2`, `westus`, `westus2`, `centralus`
- Europe: `northeurope`, `westeurope`, `uksouth`, `francecentral`
- Asia: `eastasia`, `southeastasia`, `japaneast`, `australiaeast`
- Other: `brazilsouth`, `southafricanorth`

**VM Sizes:**
- General Purpose: `Standard_B2s`, `Standard_D2s_v3`, `Standard_D4s_v3`
- Compute Optimized: `Standard_F2s_v2`, `Standard_F4s_v2`
- Memory Optimized: `Standard_E2s_v3`, `Standard_E4s_v3`

---

## 🔐 Environment Variables

Set credentials via environment variables:

```bash
# DigitalOcean
export DIGITALOCEAN_TOKEN="dop_v1_xxxxx"

# Linode
export LINODE_TOKEN="xxxxx"
export LINODE_ROOT_PASSWORD="SecurePassword123!"

# AWS
export AWS_ACCESS_KEY_ID="AKIAxxxxx"
export AWS_SECRET_ACCESS_KEY="xxxxx"

# Google Cloud
export GCP_PROJECT_ID="my-project-id"
export GCP_CREDENTIALS="path/to/service-account.json"

# Azure
export AZURE_SUBSCRIPTION_ID="xxxxx"
export AZURE_TENANT_ID="xxxxx"
export AZURE_CLIENT_ID="xxxxx"
export AZURE_CLIENT_SECRET="xxxxx"
```

---

## 🌐 Cross-Cloud Networking

### **WireGuard VPN Mesh**

All nodes across all providers are connected via encrypted WireGuard VPN:

```yaml
network:
  wireguard:
    create: true              # Auto-create VPN server
    provider: digitalocean    # Host on cheapest provider
    region: nyc3
    size: s-1vcpu-1gb
    enabled: true
    port: 51820
    clientIpBase: 10.8.0
    subnetCidr: 10.8.0.0/24
    meshNetworking: true
    allowedIps:
      - 10.10.0.0/16  # DigitalOcean VPC
      - 10.11.0.0/16  # Linode VPC
      - 10.12.0.0/16  # AWS VPC
      - 10.13.0.0/16  # GCP VPC
      - 10.14.0.0/16  # Azure VNet
```

**Benefits:**
- ✅ Encrypted communication between all nodes
- ✅ Flat network topology across clouds
- ✅ No egress charges for inter-node traffic
- ✅ Works with Kubernetes CNI (Calico/Cilium)
- ✅ Low latency (~5-10ms overhead)

---

## 💰 Cost Optimization

### **Multi-Cloud Cost Comparison**

| Provider | Master (2vCPU, 4GB) | Worker (4vCPU, 8GB) | Egress (per GB) |
|----------|---------------------|---------------------|-----------------|
| DigitalOcean | $24/mo | $48/mo | $0.01 |
| Linode | $24/mo | $48/mo | $0.01 |
| AWS | $30/mo | $60/mo | $0.09 |
| GCP | $28/mo | $56/mo | $0.12 |
| Azure | $30/mo | $55/mo | $0.087 |

### **Cost-Effective Configuration**

```yaml
# 3 Masters, 6 Workers = ~$400/month
nodePools:
  # Masters on cheapest providers
  - name: do-masters
    provider: digitalocean
    count: 2
    roles: [master]
    size: s-2vcpu-4gb  # $24/mo each

  - name: linode-master
    provider: linode
    count: 1
    roles: [master]
    size: g6-standard-2  # $24/mo

  # Workers distributed
  - name: do-workers
    provider: digitalocean
    count: 3
    roles: [worker]
    size: s-4vcpu-8gb  # $48/mo each

  - name: aws-workers
    provider: aws
    count: 2
    roles: [worker]
    size: t3.large  # $60/mo each

  - name: gcp-worker
    provider: gcp
    count: 1
    roles: [worker]
    size: n2-standard-4  # $56/mo
```

**Estimated Total:** ~$408/month

---

## 🎯 Use Cases

### **1. High Availability**
Distribute nodes across providers to survive cloud provider outages:

```yaml
nodePools:
  - name: do-masters
    provider: digitalocean
    count: 2
    roles: [master]

  - name: aws-masters
    provider: aws
    count: 2
    roles: [master]

  - name: gcp-master
    provider: gcp
    count: 1
    roles: [master]
```

**Resilience:** Cluster survives even if 2 providers fail simultaneously.

### **2. Cost Optimization**
Use cheapest providers for control plane, powerful providers for workloads:

```yaml
nodePools:
  # Masters on cheap providers
  - name: linode-masters
    provider: linode
    count: 3
    size: g6-standard-2  # $24/mo

  # GPU workloads on GCP
  - name: gcp-gpu-workers
    provider: gcp
    count: 2
    size: n1-standard-8
    # Add GPU config
```

### **3. Geographic Distribution**
Deploy close to users globally:

```yaml
nodePools:
  # US East
  - name: us-workers
    provider: aws
    region: us-east-1
    count: 3

  # Europe
  - name: eu-workers
    provider: azure
    location: westeurope
    count: 2

  # Asia
  - name: asia-workers
    provider: gcp
    region: asia-southeast1
    count: 2
```

### **4. Vendor Lock-in Avoidance**
No dependency on single cloud provider's managed Kubernetes:

- ✅ Standard Kubernetes (RKE2)
- ✅ Full control over cluster
- ✅ Portable across any provider
- ✅ No vendor-specific APIs

---

## 🔧 Advanced Configuration

### **Node Pool per Provider**

```yaml
nodePools:
  # DigitalOcean
  - name: do-pool
    provider: digitalocean
    count: 3
    size: s-2vcpu-4gb
    region: nyc3
    labels:
      cloud: digitalocean
      zone: nyc3

  # AWS
  - name: aws-pool
    provider: aws
    count: 2
    size: t3.medium
    region: us-east-1
    zones:
      - us-east-1a
      - us-east-1b
    labels:
      cloud: aws
      availability-zone: multi

  # GCP
  - name: gcp-pool
    provider: gcp
    count: 2
    size: e2-standard-2
    region: us-central1
    zones:
      - us-central1-a
      - us-central1-b
    labels:
      cloud: gcp
      preemptible: "false"

  # Azure
  - name: azure-pool
    provider: azure
    count: 2
    size: Standard_D2s_v3
    location: eastus
    labels:
      cloud: azure
      vm-family: Dsv3
```

### **Provider-Specific Taints**

Prevent pods from scheduling on certain clouds:

```yaml
nodePools:
  - name: expensive-cloud
    provider: azure
    count: 2
    taints:
      - key: cloud
        value: azure
        effect: NoSchedule
```

Then schedule specific pods:

```yaml
apiVersion: v1
kind: Pod
spec:
  tolerations:
    - key: cloud
      operator: Equal
      value: azure
      effect: NoSchedule
  nodeSelector:
    cloud: azure
```

---

## 📊 Monitoring & Observability

### **Provider Metrics**

Sloth Kubernetes exports metrics for each provider:

```promql
# Nodes by provider
count(kube_node_labels{label_cloud_provider="aws"})

# Pod distribution
sum(kube_pod_info) by (node)

# Cross-cloud traffic (WireGuard)
wireguard_bytes_received{peer="10.8.0.x"}
```

---

## 🐛 Troubleshooting

### **Provider Not Connecting**

1. Check credentials:
   ```bash
   sloth-kubernetes config validate
   ```

2. Test provider API:
   ```bash
   # AWS
   aws ec2 describe-regions

   # GCP
   gcloud compute regions list

   # Azure
   az account list
   ```

3. Verify networking:
   ```bash
   sloth-kubernetes vpn test
   ```

### **Cross-Cloud Communication Issues**

1. Check WireGuard status:
   ```bash
   sloth-kubernetes vpn status
   ```

2. Test connectivity:
   ```bash
   # From any node
   ping 10.8.0.11  # VPN IP of another node
   ```

3. Verify firewall rules:
   ```bash
   # Check if UDP 51820 is open
   nc -zvu <node-ip> 51820
   ```

---

## 🚀 Roadmap

### **Q1 2025**
- ✅ AWS Support
- ✅ GCP Support
- ✅ Azure Support

### **Q2 2025**
- 🔲 AWS ALB/NLB Integration
- 🔲 GCP Load Balancer
- 🔲 Azure Load Balancer
- 🔲 Auto-scaling per provider

### **Q3 2025**
- 🔲 Spot/Preemptible Instance Support
- 🔲 Oracle Cloud Infrastructure
- 🔲 Hetzner Cloud
- 🔲 Cost Analytics Dashboard

---

## 📚 Examples

See full examples in `examples/`:

- **`cluster-multi-cloud-all-providers.yaml`** - 5-provider production cluster
- **`cluster-aws-only.yaml`** - AWS-only deployment
- **`cluster-gcp-only.yaml`** - GCP-only deployment
- **`cluster-azure-only.yaml`** - Azure-only deployment

---

## 🤝 Contributing

We welcome contributions for additional providers! See [CONTRIBUTING.md](../CONTRIBUTING.md).

### **Adding a New Provider**

1. Create `pkg/providers/{provider}.go`
2. Implement the `Provider` interface
3. Add config types to `pkg/config/types.go`
4. Update `pkg/config/k8s_style.go`
5. Add tests
6. Create example configuration

---

## 📖 Documentation

- [Configuration Reference](../docs/configuration/)
- [Network Architecture](../docs/architecture/network.md)
- [Security Best Practices](../docs/security/)
- [Examples](../examples/)

---

Built with ❤️ by the Sloth Kubernetes team
