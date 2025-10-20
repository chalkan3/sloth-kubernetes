# Configuration Examples

This directory contains example configuration files for kubernetes-create.

## Files

### cluster-basic.yaml
Complete configuration example with all available options including:
- Multi-cloud setup (DigitalOcean + Linode)
- WireGuard VPN mesh networking
- RKE2 with custom configuration
- High availability (3 masters + 3 workers)
- Full metadata and labels

### cluster-minimal.yaml
Minimal configuration example for quick start:
- Single cloud provider (DigitalOcean)
- Simple HA setup (3 masters + 3 workers)
- Default RKE2 settings

## Configuration Format

All configuration files follow the Kubernetes API convention:

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: cluster-name
  labels: {}
  annotations: {}
spec:
  providers: {}
  network: {}
  kubernetes: {}
  nodePools: []
```

## Environment Variables

Configuration files support environment variable expansion using `${VAR_NAME}` syntax:

```yaml
providers:
  digitalocean:
    token: ${DIGITALOCEAN_TOKEN}  # Will be replaced with env var value
```

Required environment variables:
- `DIGITALOCEAN_TOKEN` - DigitalOcean API token
- `LINODE_TOKEN` - Linode API token (if using Linode)
- `LINODE_ROOT_PASSWORD` - Root password for Linode instances
- `WIREGUARD_ENDPOINT` - WireGuard server endpoint (if using WireGuard)
- `WIREGUARD_PUBKEY` - WireGuard server public key (if using WireGuard)

## Usage

### 1. Generate a new configuration file

```bash
# Generate full example
kubernetes-create config generate

# Generate minimal example
kubernetes-create config generate --format minimal -o my-cluster.yaml
```

### 2. Edit the configuration

```bash
vim cluster-config.yaml
```

### 3. Set environment variables

```bash
export DIGITALOCEAN_TOKEN="dop_xxxxx"
export LINODE_TOKEN="xxxxx"
export LINODE_ROOT_PASSWORD="secure-password"
export WIREGUARD_ENDPOINT="vpn.example.com:51820"
export WIREGUARD_PUBKEY="xxxxx="
```

### 4. Deploy the cluster

```bash
# Using config file
kubernetes-create deploy --config cluster-config.yaml

# Override specific values with flags
kubernetes-create deploy --config cluster-config.yaml \
  --do-token different-token \
  --wireguard-endpoint different-endpoint
```

## Configuration Sections

### Providers

Define which cloud providers to use:

```yaml
spec:
  providers:
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}
      region: nyc3
      tags: [kubernetes, production]

    linode:
      enabled: true
      token: ${LINODE_TOKEN}
      region: us-east
      rootPassword: ${LINODE_ROOT_PASSWORD}
      tags: [kubernetes, production]
```

### Network

Configure DNS and VPN:

```yaml
spec:
  network:
    dns:
      domain: example.com
      provider: digitalocean

    wireguard:
      enabled: true
      serverEndpoint: ${WIREGUARD_ENDPOINT}
      serverPublicKey: ${WIREGUARD_PUBKEY}
      clientIPBase: 10.100.0.0/24
      port: 51820
      mtu: 1420
      persistentKeepalive: 25
```

### Kubernetes

Configure Kubernetes settings:

```yaml
spec:
  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1
    networkPlugin: calico
    podCIDR: 10.42.0.0/16
    serviceCIDR: 10.43.0.0/16
    clusterDNS: 10.43.0.10
    clusterDomain: cluster.local

    rke2:
      channel: stable
      clusterToken: your-secure-token
      tlsSan:
        - api.example.com
      disableComponents:
        - rke2-ingress-nginx
      snapshotScheduleCron: "0 */12 * * *"
      snapshotRetention: 5
      secretsEncryption: true
      writeKubeconfigMode: "0600"
```

### Node Pools

Define node pools:

```yaml
spec:
  nodePools:
    - name: masters
      provider: digitalocean
      count: 3
      roles: [master]
      size: s-2vcpu-4gb
      image: ubuntu-22-04-x64
      region: nyc3
      labels:
        node-role.kubernetes.io/master: "true"

    - name: workers
      provider: digitalocean
      count: 3
      roles: [worker]
      size: s-2vcpu-4gb
      image: ubuntu-22-04-x64
      region: nyc3
      labels:
        node-role.kubernetes.io/worker: "true"
```

## RKE2 Configuration Options

Available RKE2 configuration options:

- `channel`: Release channel (stable, latest, testing)
- `clusterToken`: Shared secret for cluster authentication
- `tlsSan`: Additional Subject Alternative Names for API server certificate
- `disableComponents`: List of components to disable
- `snapshotScheduleCron`: Etcd snapshot schedule (cron format)
- `snapshotRetention`: Number of snapshots to retain
- `secretsEncryption`: Enable secrets encryption at rest
- `writeKubeconfigMode`: File permissions for kubeconfig
- `extraServerArgs`: Additional arguments for RKE2 server
- `extraAgentArgs`: Additional arguments for RKE2 agent

## Tips

1. **Use environment variables for sensitive data**: Never commit tokens to Git
2. **Start with minimal config**: Use `cluster-minimal.yaml` as a starting point
3. **Validate before deploy**: The CLI validates configuration before deployment
4. **Flags override config**: CLI flags take precedence over config file values
5. **Keep backups**: Save your configuration files in version control (without secrets)

## Further Reading

- [RKE2 Documentation](https://docs.rke2.io/)
- [WireGuard Documentation](https://www.wireguard.com/)
- [DigitalOcean Kubernetes](https://www.digitalocean.com/products/kubernetes)
- [Linode Kubernetes](https://www.linode.com/products/kubernetes/)
