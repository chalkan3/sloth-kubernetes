# Cloud Provider Requirements

This document details all resources created and requirements for each cloud provider in Sloth Kubernetes.

## Table of Contents

- [Overview](#overview)
- [DigitalOcean](#digitalocean)
- [Linode](#linode)
- [Amazon Web Services (AWS)](#amazon-web-services-aws)
- [Google Cloud Platform (GCP)](#google-cloud-platform-gcp)
- [Microsoft Azure](#microsoft-azure)
- [Multi-Cloud Considerations](#multi-cloud-considerations)
- [Troubleshooting](#troubleshooting)

## Overview

Sloth Kubernetes creates the following resources in each cloud provider:

1. **Network Infrastructure**: VPC/VNet, Subnets, Internet Gateway/NAT
2. **Security**: Security Groups/Firewall Rules/NSG
3. **Compute**: Virtual Machines/Instances/Droplets
4. **Networking**: Public IPs, Network Interfaces
5. **Storage**: Boot disks (100GB per node)

### Resource Creation Order

**CRITICAL**: Resources are created in this specific order to avoid dependency errors:

```
1. Network (VPC/VNet)
2. Subnets
3. Internet Gateway (AWS only)
4. Route Tables (AWS only)
5. Security Groups/Firewall Rules
6. Virtual Machines
```

## DigitalOcean

### Required Resources

| Resource | Quantity | Purpose |
|----------|----------|---------|
| Droplets | Per node pool | Compute instances |
| VPC | 1 per cluster | Private network (CIDR: 10.10.0.0/16) |
| Firewall | 1 per cluster | Security rules |
| SSH Keys | 1 per cluster | Authentication |

### Firewall Rules Created

```yaml
Inbound:
  - Port 22 (SSH): 0.0.0.0/0
  - Port 6443 (K8s API): 0.0.0.0/0
  - Port 51820 (WireGuard): 0.0.0.0/0
  - Ports 30000-32767 (NodePorts): 0.0.0.0/0
  - All traffic from VPC: 10.10.0.0/16

Outbound:
  - All traffic: 0.0.0.0/0
```

### Configuration Example

```yaml
providers:
  digitalocean:
    enabled: true
    token: "${DO_TOKEN}"  # Required
    region: "nyc3"        # Required
    vpc:
      cidr: "10.10.0.0/16"
```

### Environment Variables

```bash
export DO_TOKEN="your-digitalocean-token"
export DIGITALOCEAN_TOKEN="${DO_TOKEN}"  # Alternative
```

### Cost Estimate

- **Masters**: $12/month per node (s-2vcpu-4gb)
- **Workers**: $12/month per node (s-2vcpu-4gb)
- **3 master + 3 worker cluster**: ~$72/month

## Linode

### Required Resources

| Resource | Quantity | Purpose |
|----------|----------|---------|
| Linodes | Per node pool | Compute instances |
| VPC | 1 per cluster | Private network (CIDR: 10.11.0.0/16) |
| Firewall | 1 per cluster | Security rules |
| SSH Keys | 1 per cluster | Authentication |

### Firewall Rules Created

```yaml
Inbound:
  - Port 22 (SSH): 0.0.0.0/0
  - Port 6443 (K8s API): 0.0.0.0/0
  - Port 51820 (WireGuard): 0.0.0.0/0
  - Ports 30000-32767 (NodePorts): 0.0.0.0/0
  - All traffic from VPC: 10.11.0.0/16

Outbound:
  - All traffic: 0.0.0.0/0
```

### Configuration Example

```yaml
providers:
  linode:
    enabled: true
    token: "${LINODE_TOKEN}"  # Required
    region: "us-east"         # Required
    vpc:
      cidr: "10.11.0.0/16"
```

### Environment Variables

```bash
export LINODE_TOKEN="your-linode-token"
```

### Cost Estimate

- **Masters**: $12/month per node (g6-standard-2)
- **Workers**: $12/month per node (g6-standard-2)
- **3 master + 3 worker cluster**: ~$72/month

## Amazon Web Services (AWS)

### Required Resources

| Resource | Quantity | Purpose |
|----------|----------|---------|
| VPC | 1 per cluster | Virtual Private Cloud (CIDR: 10.12.0.0/16) |
| Subnet | 1 per VPC | Subnet within VPC (CIDR: 10.12.1.0/24) |
| Internet Gateway | 1 per VPC | Internet connectivity |
| Route Table | 1 per VPC | Routing rules |
| Route Table Association | 1 per subnet | Link subnet to route table |
| Security Group | 1 per cluster | Firewall rules |
| EC2 Instances | Per node pool | Virtual machines |
| Key Pairs | 1 per cluster | SSH authentication |

### Security Group Rules Created

```yaml
Ingress:
  - Port 22 (SSH): 0.0.0.0/0
  - Port 6443 (K8s API): 0.0.0.0/0
  - Port 51820 (WireGuard UDP): 0.0.0.0/0
  - Ports 2379-2380 (etcd): VPC CIDR only
  - Ports 10250-10259 (K8s components): VPC CIDR only
  - Ports 30000-32767 (NodePorts): 0.0.0.0/0
  - All traffic from VPC: 10.12.0.0/16

Egress:
  - All traffic: 0.0.0.0/0
```

### Configuration Example

```yaml
providers:
  aws:
    enabled: true
    region: "us-east-1"                    # Required
    accessKeyId: "${AWS_ACCESS_KEY_ID}"    # Optional (can use IAM role)
    secretAccessKey: "${AWS_SECRET_KEY}"   # Optional (can use IAM role)
    vpc:
      cidr: "10.12.0.0/16"
      enableDnsHostnames: true
      enableDnsSupport: true
```

### Environment Variables

```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
```

### Alternative: IAM Role

If running on EC2 or using AWS CLI with configured credentials:

```bash
aws configure
# Or attach IAM role to EC2 instance
```

### Cost Estimate

- **Masters**: $33.60/month per node (t3.medium: 2 vCPU, 4GB RAM)
- **Workers**: $33.60/month per node (t3.medium)
- **VPC, Internet Gateway**: Free
- **Data Transfer**: ~$0.09/GB outbound
- **3 master + 3 worker cluster**: ~$201.60/month + data transfer

## Google Cloud Platform (GCP)

### Required Resources

| Resource | Quantity | Purpose |
|----------|----------|---------|
| VPC Network | 1 per cluster | Virtual network (auto-mode disabled) |
| Subnetwork | 1 per region | Subnet within VPC (CIDR: 10.13.1.0/24) |
| Firewall Rules | 4 per cluster | Security rules |
| Compute Instances | Per node pool | Virtual machines |

### Firewall Rules Created

```yaml
Rule 1 - SSH:
  - Protocol: TCP
  - Port: 22
  - Source: 0.0.0.0/0

Rule 2 - Kubernetes API:
  - Protocol: TCP
  - Port: 6443
  - Source: 0.0.0.0/0

Rule 3 - WireGuard VPN:
  - Protocol: UDP
  - Port: 51820
  - Source: 0.0.0.0/0

Rule 4 - Internal Cluster:
  - Protocol: All
  - Source: 10.13.0.0/16 (VPC CIDR)
```

### Configuration Example

```yaml
providers:
  gcp:
    enabled: true
    projectId: "my-project-123456"        # Required
    region: "us-central1"                  # Required
    zone: "us-central1-a"                  # Optional (defaults to region-a)
    credentials: "${GOOGLE_CREDENTIALS}"   # Optional (can use ADC)
    network:
      cidr: "10.13.0.0/16"
      autoCreateSubnetworks: false
```

### Environment Variables

```bash
export GOOGLE_CREDENTIALS="/path/to/service-account.json"
export GOOGLE_PROJECT="my-project-123456"
export GOOGLE_REGION="us-central1"
```

### Alternative: Application Default Credentials (ADC)

```bash
gcloud auth application-default login
gcloud config set project my-project-123456
```

### Cost Estimate

- **Masters**: $24.27/month per node (e2-medium: 2 vCPU, 4GB RAM)
- **Workers**: $24.27/month per node (e2-medium)
- **Networking**: Free (within same region)
- **Data Transfer**: ~$0.12/GB outbound
- **3 master + 3 worker cluster**: ~$145.62/month + data transfer

### Important Notes

- **Labels**: GCP requires labels to be lowercase, alphanumeric + hyphens only
- **Subnet**: Created automatically in the specified region
- **Service Account**: Instances run with default compute service account

## Microsoft Azure

### Required Resources

| Resource | Quantity | Purpose |
|----------|----------|---------|
| Resource Group | 1 per cluster | Container for all resources |
| Virtual Network | 1 per cluster | VNet (CIDR: 10.14.0.0/16) |
| Subnet | 1 per VNet | Subnet (CIDR: 10.14.1.0/24) |
| Network Security Group | 1 per cluster | Firewall rules |
| Public IP Addresses | 1 per node | Static public IPs |
| Network Interfaces | 1 per node | VM network adapters |
| Virtual Machines | Per node pool | Compute instances |

### NSG Rules Created

```yaml
Priority 100 - Allow VNet Inbound:
  - Protocol: All
  - Source: VirtualNetwork
  - Destination: VirtualNetwork

Priority 110 - Allow SSH:
  - Protocol: TCP
  - Port: 22
  - Source: Internet

Priority 120 - Allow WireGuard:
  - Protocol: UDP
  - Port: 51820
  - Source: Internet

Priority 130 - Allow Kubernetes API:
  - Protocol: TCP
  - Port: 6443
  - Source: Internet

Priority 140 - Allow etcd:
  - Protocol: TCP
  - Ports: 2379-2380
  - Source: VirtualNetwork

Priority 150 - Allow NodePorts:
  - Protocol: TCP
  - Ports: 30000-32767
  - Source: Internet

Priority 160 - Allow HTTP/HTTPS:
  - Protocol: TCP
  - Ports: 80, 443
  - Source: Internet
```

### Configuration Example

```yaml
providers:
  azure:
    enabled: true
    location: "eastus"                           # Required
    subscriptionId: "${AZURE_SUBSCRIPTION_ID}"   # Optional (can use Azure CLI)
    tenantId: "${AZURE_TENANT_ID}"               # Optional
    clientId: "${AZURE_CLIENT_ID}"               # Optional
    clientSecret: "${AZURE_CLIENT_SECRET}"       # Optional
    resourceGroup: "sloth-k8s-rg"                # Optional (auto-created)
    virtualNetwork:
      cidr: "10.14.0.0/16"
```

### Environment Variables

```bash
export AZURE_SUBSCRIPTION_ID="your-subscription-id"
export AZURE_TENANT_ID="your-tenant-id"
export AZURE_CLIENT_ID="your-client-id"
export AZURE_CLIENT_SECRET="your-client-secret"
```

### Alternative: Azure CLI

```bash
az login
az account set --subscription "your-subscription-id"
```

### Cost Estimate

- **Masters**: $30.37/month per node (Standard_B2ms: 2 vCPU, 8GB RAM)
- **Workers**: $15.18/month per node (Standard_B2s: 2 vCPU, 4GB RAM)
- **Public IPs**: $3.65/month per IP
- **Networking**: Free within VNet
- **3 master + 3 worker cluster**: ~$136.83/month

### Important Notes

- **Resource Group**: All resources are created within a single Resource Group
- **Public IPs**: Standard SKU with Static allocation
- **OS**: Ubuntu 22.04 LTS (Jammy)
- **Admin User**: `azureuser` (SSH key-based authentication)

## Multi-Cloud Considerations

### WireGuard VPN Mesh

All nodes across all clouds are connected via WireGuard VPN mesh:

| Provider | WireGuard IP Range |
|----------|-------------------|
| DigitalOcean | 10.8.0.1 - 10.8.0.5 |
| Linode | 10.8.0.6 - 10.8.0.10 |
| AWS | 10.8.0.11 - 10.8.0.14 |
| GCP | 10.8.0.15 - 10.8.0.19 |
| Azure | 10.8.0.20 - 10.8.0.25 |

### Network Architecture

```
┌─────────────────────────────────────────────────────────┐
│              WireGuard Mesh (10.8.0.0/24)               │
├─────────────┬──────────┬──────────┬──────────┬──────────┤
│             │          │          │          │          │
│ DigitalOcean│  Linode  │   AWS    │   GCP    │  Azure   │
│ 10.10.0.0/16│10.11.0.0 │10.12.0.0 │10.13.0.0 │10.14.0.0 │
│             │    /16   │    /16   │    /16   │    /16   │
└─────────────┴──────────┴──────────┴──────────┴──────────┘
```

### Cross-Cloud Communication

1. **Encrypted**: All cross-cloud traffic encrypted via WireGuard
2. **Direct**: Nodes communicate directly using public IPs
3. **Automatic**: VPN mesh configured automatically during deployment
4. **Resilient**: Survives provider failures

## Troubleshooting

### Common Issues

#### 1. AWS: Security Group Created After Nodes

**Symptom**: Nodes fail to create with "security group not found"

**Cause**: Security Group must be created BEFORE nodes

**Solution**: Already fixed! Security Group is now created in `CreateNetwork()` before nodes.

#### 2. GCP: Label Validation Failed

**Symptom**: "Invalid label format" error

**Cause**: GCP labels must be lowercase, alphanumeric + hyphens, max 63 chars

**Solution**: Already fixed! Labels are sanitized with `sanitizeGCPLabel()` function.

#### 3. Azure: NSG Not Attached to NICs

**Symptom**: Nodes have no firewall rules

**Cause**: NSG must exist before creating Network Interfaces

**Solution**: Already fixed! NSG is created in `CreateNetwork()` before nodes.

#### 4. Network Not Created Before Nodes

**Symptom**: "network infrastructure incomplete" error

**Cause**: `CreateNode()` called before `CreateNetwork()`

**Solution**: Ensure orchestration calls `CreateNetwork()` first for each provider.

#### 5. Credential Errors

**Symptom**: "authentication failed" or "unauthorized"

**Solutions**:
- **DigitalOcean**: Check `DO_TOKEN` environment variable
- **Linode**: Check `LINODE_TOKEN` environment variable
- **AWS**: Verify `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`, or IAM role
- **GCP**: Check `GOOGLE_CREDENTIALS` path or run `gcloud auth application-default login`
- **Azure**: Verify Azure CLI login with `az login` or check service principal credentials

### Validation Checklist

Before deployment, verify:

- [ ] All provider credentials configured
- [ ] Required environment variables set
- [ ] SSH public key exists at `~/.ssh/id_rsa.pub`
- [ ] Provider regions/locations are valid
- [ ] CIDR ranges don't overlap (unless intentional)
- [ ] At least 1 provider is enabled in config
- [ ] Each node pool specifies a valid provider

### Testing Individual Providers

Test each provider independently before multi-cloud deployment:

```yaml
# Test DigitalOcean only
providers:
  digitalocean:
    enabled: true
    token: "${DO_TOKEN}"
    region: "nyc3"

nodePools:
  - name: do-test
    provider: digitalocean
    count: 1
    roles: [master]
```

### Resource Cleanup

To clean up all resources:

```bash
# Using Pulumi
sloth-kubernetes destroy --config cluster.yaml

# Or manually via cloud consoles
# AWS: Delete VPC (cascades to all resources)
# GCP: Delete VPC Network
# Azure: Delete Resource Group
```

## Cost Optimization Tips

1. **Use Cheaper Providers for Masters**: DigitalOcean/Linode ($12/mo) vs AWS ($33/mo)
2. **Spot Instances**: Use AWS/GCP spot instances for non-critical workers (Q1 2025)
3. **Right-Size VMs**: Don't over-provision - K8s masters need minimal resources
4. **Regional Selection**: Some regions are cheaper (e.g., GCP us-central1)
5. **Reserved Instances**: AWS/GCP offer discounts for 1-3 year commitments

## Security Best Practices

1. **Rotate Credentials**: Regularly rotate API tokens and access keys
2. **Least Privilege**: Use service accounts with minimal required permissions
3. **Network Segmentation**: Use VPN mesh, don't expose etcd publicly
4. **SSH Keys**: Use different SSH keys per cluster
5. **Audit Logs**: Enable CloudTrail (AWS), Cloud Audit Logs (GCP), Activity Log (Azure)
6. **Secrets Management**: Use environment variables, never commit credentials to git

## Summary Table

| Provider | Monthly Cost | Setup Time | Regions | Special Requirements |
|----------|--------------|------------|---------|---------------------|
| DigitalOcean | $12/node | 2-3 min | 14 | API token |
| Linode | $12/node | 2-3 min | 11 | API token |
| AWS | $33/node | 4-5 min | 25 | IAM permissions |
| GCP | $24/node | 3-4 min | 35 | Project ID, billing enabled |
| Azure | $15-30/node | 4-5 min | 60+ | Subscription ID |

**Total for 3+3 cluster across all 5 providers**: ~$408/month

---

**Generated with Sloth Kubernetes v1.0.0**

For more information, see:
- [Multi-Cloud Providers Guide](./MULTI_CLOUD_PROVIDERS.md)
- [README](../README.md)
- [Project Repository](https://github.com/chalkan3/sloth-kubernetes)
