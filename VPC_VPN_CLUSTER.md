# 🌐 VPC + VPN + Cluster - Fluxo Completo

## Visão Geral

O `kubernetes-create` agora suporta criação **automática e integrada** de toda a infraestrutura de rede:

```
1️⃣ VPC (Virtual Private Cloud)
        ↓
2️⃣ VPN (WireGuard Server)
        ↓
3️⃣ Cluster (Kubernetes)
```

**Tudo configurado em um único YAML e deployado em um único comando!**

---

## 🎯 O Que É Criado

### 1. VPCs (Virtual Private Clouds)

- **DigitalOcean**: VPC com CIDR configurável (ex: `10.10.0.0/16`)
- **Linode**: VPC com subnets configuráveis (ex: `10.11.0.0/16`)
- **Isolamento**: Cada provider tem sua própria VPC privada
- **DNS**: DNS interno habilitado automaticamente

### 2. VPN (WireGuard Server)

- **Servidor**: VM dedicada rodando WireGuard
- **Provider**: Escolha onde criar (DigitalOcean ou Linode)
- **Configuração**: Totalmente automática
- **Mesh**: Conecta todos os nodes via túneis criptografados
- **Cross-Cloud**: Permite comunicação entre providers

### 3. Kubernetes Cluster

- **RKE2**: Kubernetes production-ready
- **Multi-Cloud**: Nodes em múltiplos providers
- **HA**: 3+ masters para alta disponibilidade
- **Networking**: Pod e Service networking via Calico

---

## 📋 Configuração no YAML

### Configuração Completa

```yaml
apiVersion: kubernetes-create.io/v1
kind: ClusterConfig
metadata:
  name: production-cluster

# 1️⃣ PROVIDERS com VPC
providers:
  digitalocean:
    enabled: true
    token: ${DIGITALOCEAN_TOKEN}
    region: nyc3

    # VPC Configuration
    vpc:
      create: true                    # Auto-criar VPC
      name: k8s-vpc-do
      cidr: 10.10.0.0/16
      region: nyc3
      enableDns: true
      enableDnsHostname: true
      internetGateway: true
      tags:
        - kubernetes
        - production

  linode:
    enabled: true
    token: ${LINODE_TOKEN}
    region: us-east

    # VPC Configuration
    vpc:
      create: true                    # Auto-criar VPC
      name: k8s-vpc-linode
      cidr: 10.11.0.0/16
      region: us-east
      enableDns: true
      linode:
        label: k8s-vpc-linode
        subnets:
          - label: k8s-subnet-1
            ipv4: 10.11.1.0/24

# 2️⃣ NETWORK com VPN
network:
  mode: wireguard
  cidr: 10.8.0.0/16
  crossProviderNetworking: true

  # WireGuard VPN Server
  wireguard:
    # Auto-create settings
    create: true                      # Auto-criar VPN server
    provider: digitalocean            # Onde criar
    region: nyc3
    size: s-1vcpu-1gb                # Tamanho do server
    name: wireguard-vpn-server

    # VPN settings
    enabled: true
    port: 51820
    subnetCidr: 10.8.0.0/24
    clientIpBase: 10.8.0
    mtu: 1420
    persistentKeepalive: 25
    meshNetworking: true
    autoConfig: true

    # Rotas permitidas
    allowedIps:
      - 10.8.0.0/24      # VPN subnet
      - 10.10.0.0/16     # DO VPC
      - 10.11.0.0/16     # Linode VPC

# 3️⃣ KUBERNETES
kubernetes:
  version: v1.28.5+rke2r1
  cni: calico

# 4️⃣ NODE POOLS
nodePools:
  do-masters:
    provider: digitalocean
    count: 1
    size: s-2vcpu-4gb
    role: master

  do-workers:
    provider: digitalocean
    count: 2
    size: s-2vcpu-4gb
    role: worker

  linode-masters:
    provider: linode
    count: 2
    size: g6-standard-2
    role: master

  linode-workers:
    provider: linode
    count: 1
    size: g6-standard-2
    role: worker
```

### Configuração Mínima

```yaml
apiVersion: kubernetes-create.io/v1
kind: ClusterConfig
metadata:
  name: simple-cluster

providers:
  digitalocean:
    enabled: true
    token: ${DIGITALOCEAN_TOKEN}
    region: nyc3
    vpc:
      create: true                    # VPC automática
      name: k8s-vpc
      cidr: 10.10.0.0/16

network:
  mode: wireguard
  wireguard:
    create: true                      # VPN automática
    provider: digitalocean
    region: nyc3

kubernetes:
  version: v1.28.5+rke2r1

nodePools:
  masters:
    provider: digitalocean
    count: 1
    size: s-2vcpu-4gb
    role: master
  workers:
    provider: digitalocean
    count: 2
    size: s-2vcpu-4gb
    role: worker
```

---

## 🚀 Deploy Completo

### 1. Criar Arquivo de Configuração

```bash
# Usar exemplo completo
cp examples/cluster-with-vpc-vpn.yaml my-cluster.yaml

# Ou usar mínimo
cp examples/cluster-minimal-with-vpn.yaml my-cluster.yaml

# Editar conforme necessário
vim my-cluster.yaml
```

### 2. Configurar Variáveis de Ambiente

```bash
# Tokens dos providers
export DIGITALOCEAN_TOKEN="dop_v1_xxxxxxxxxxxxx"
export LINODE_TOKEN="xxxxxxxxxxxxx"
```

### 3. Deploy!

```bash
# Preview primeiro (dry-run)
kubernetes-create deploy --config my-cluster.yaml --dry-run

# Deploy real
kubernetes-create deploy --config my-cluster.yaml
```

### 4. O Que Acontece

```
⏳ Starting deployment...

📊 Phase 1: VPC Creation
  ✅ Creating DigitalOcean VPC (10.10.0.0/16)...
  ✅ Creating Linode VPC (10.11.0.0/16)...
  ✅ VPCs created successfully!

📊 Phase 2: VPN Server Creation
  ✅ Creating WireGuard server on DigitalOcean...
  ✅ Installing WireGuard software...
  ✅ Generating WireGuard keys...
  ✅ Configuring WireGuard interfaces...
  ✅ VPN server ready at 167.99.1.1:51820

📊 Phase 3: Cluster Nodes Creation
  ✅ Creating 3 master nodes (1 DO + 2 Linode)...
  ✅ Creating 3 worker nodes (2 DO + 1 Linode)...
  ✅ All nodes created!

📊 Phase 4: WireGuard Client Configuration
  ✅ Installing WireGuard on all nodes...
  ✅ Generating client keys...
  ✅ Configuring peer connections...
  ✅ Establishing VPN mesh...
  ✅ All nodes connected via VPN!

📊 Phase 5: Kubernetes Installation
  ✅ Installing RKE2 on master nodes...
  ✅ Bootstrapping etcd cluster...
  ✅ Installing RKE2 on worker nodes...
  ✅ Joining workers to cluster...
  ✅ Configuring Calico CNI...
  ✅ Kubernetes cluster ready!

🎉 Deployment complete!

📝 Cluster Information:
  • Nodes: 6 (3 masters + 3 workers)
  • VPCs: 2 (DigitalOcean + Linode)
  • VPN: WireGuard at 167.99.1.1:51820
  • Kubeconfig: ./kubeconfig

🔍 Next Steps:
  • kubectl get nodes
  • kubernetes-create status
  • kubernetes-create addons bootstrap --repo <gitops-repo>
```

---

## 🔧 Casos de Uso

### Caso 1: Cluster Simples Single-Cloud

```yaml
providers:
  digitalocean:
    enabled: true
    token: ${DO_TOKEN}
    vpc:
      create: true
      name: k8s-vpc
      cidr: 10.10.0.0/16

network:
  wireguard:
    create: true
    provider: digitalocean

nodePools:
  masters:
    provider: digitalocean
    count: 3
    role: master
  workers:
    provider: digitalocean
    count: 3
    role: worker
```

**Resultado**: Cluster de 6 nodes em uma VPC DigitalOcean conectados via VPN.

### Caso 2: Cluster Multi-Cloud

```yaml
providers:
  digitalocean:
    enabled: true
    token: ${DO_TOKEN}
    vpc:
      create: true
      cidr: 10.10.0.0/16

  linode:
    enabled: true
    token: ${LINODE_TOKEN}
    vpc:
      create: true
      cidr: 10.11.0.0/16

network:
  mode: wireguard
  crossProviderNetworking: true
  wireguard:
    create: true
    provider: digitalocean
    allowedIps:
      - 10.8.0.0/24
      - 10.10.0.0/16
      - 10.11.0.0/16

nodePools:
  do-masters:
    provider: digitalocean
    count: 1
    role: master
  linode-masters:
    provider: linode
    count: 2
    role: master
  do-workers:
    provider: digitalocean
    count: 2
    role: worker
  linode-workers:
    provider: linode
    count: 1
    role: worker
```

**Resultado**: Cluster multi-cloud com nodes em 2 VPCs diferentes conectados via VPN mesh.

### Caso 3: VPC Existente + VPN Nova

```yaml
providers:
  digitalocean:
    enabled: true
    token: ${DO_TOKEN}
    vpc:
      create: false               # Não criar VPC
      id: "existing-vpc-id"       # Usar VPC existente
      cidr: 10.10.0.0/16

network:
  wireguard:
    create: true                  # Criar apenas VPN
    provider: digitalocean
```

**Resultado**: Usa VPC existente mas cria nova VPN.

### Caso 4: VPN Existente + VPC Nova

```yaml
providers:
  digitalocean:
    enabled: true
    token: ${DO_TOKEN}
    vpc:
      create: true                # Criar VPC
      cidr: 10.10.0.0/16

network:
  wireguard:
    create: false                 # Não criar VPN
    enabled: true
    serverEndpoint: "1.2.3.4:51820"
    serverPublicKey: "existing-key"
```

**Resultado**: Cria nova VPC mas usa VPN existente.

---

## 📊 Arquitetura de Rede

### Single Cloud

```
┌─────────────────────────────────────────────────┐
│  DigitalOcean VPC (10.10.0.0/16)                │
│                                                  │
│  ┌────────────────┐                             │
│  │ WireGuard VPN  │                             │
│  │  10.8.0.1      │                             │
│  └────────────────┘                             │
│          │                                       │
│  ┌───────┴────────────────────┐                 │
│  │         VPN Mesh           │                 │
│  │                            │                 │
│  │  ┌──────┐  ┌──────┐       │                 │
│  │  │Master│  │Worker│  ...  │                 │
│  │  │10.8.2│  │10.8.3│       │                 │
│  │  └──────┘  └──────┘       │                 │
│  └────────────────────────────┘                 │
│                                                  │
└─────────────────────────────────────────────────┘
```

### Multi-Cloud

```
┌──────────────────────────────┐    ┌──────────────────────────────┐
│ DO VPC (10.10.0.0/16)        │    │ Linode VPC (10.11.0.0/16)    │
│                              │    │                              │
│  ┌────────────────┐          │    │  ┌──────┐  ┌──────┐         │
│  │ WireGuard VPN  │          │    │  │Master│  │Worker│         │
│  │   10.8.0.1     │◄─────────┼────┼──┤10.8.5├──┤10.8.6│         │
│  └────────────────┘          │    │  └──────┘  └──────┘         │
│          │                   │    │      ▲         ▲             │
│  ┌───────┴────────┐          │    │      │         │             │
│  │     Master     │          │    │      └─────────┘             │
│  │     10.8.2     │          │    │     VPN Tunnels              │
│  └────────────────┘          │    │                              │
│                              │    │                              │
└──────────────────────────────┘    └──────────────────────────────┘
            ▲                                      ▲
            └──────────────────────────────────────┘
                    Encrypted VPN Mesh
```

---

## 🔐 Segurança

### VPC Isolation

- Cada provider tem sua própria VPC privada
- Nodes não são acessíveis diretamente pela internet
- Apenas portas necessárias expostas (API, SSH via VPN)

### VPN Encryption

- Todos os dados criptografados com WireGuard
- Modern cryptography (ChaCha20, Poly1305)
- Perfect Forward Secrecy
- Autenticação por chave pública

### Network Policies

```yaml
security:
  networkPolicies: true
```

Habilita Kubernetes Network Policies para isolar pods.

---

## 📝 Configurações Avançadas

### VPC Customizada

```yaml
providers:
  digitalocean:
    vpc:
      create: true
      name: custom-vpc
      cidr: 172.16.0.0/12           # CIDR customizado
      enableDns: true
      enableDnsHostname: true
      internetGateway: true
      natGateway: true
      subnets:
        - 172.16.1.0/24
        - 172.16.2.0/24
      tags:
        - production
        - k8s
      digitalocean:
        ipRange: 172.16.0.0/12
        description: "Production Kubernetes VPC"
```

### VPN Customizada

```yaml
network:
  wireguard:
    create: true
    provider: linode              # VPN no Linode
    region: us-west
    size: g6-nanode-1             # Smallest instance
    name: vpn-gateway
    image: linode/ubuntu22.04

    # Network config
    port: 51820
    subnetCidr: 192.168.100.0/24
    clientIpBase: 192.168.100
    mtu: 1420
    persistentKeepalive: 25

    # Routes
    allowedIps:
      - 192.168.100.0/24
      - 10.10.0.0/16
      - 10.11.0.0/16
      - 10.244.0.0/16              # Pod CIDR
      - 10.96.0.0/12               # Service CIDR
```

### Multi-Region

```yaml
nodePools:
  nyc-masters:
    provider: digitalocean
    region: nyc3
    count: 1
    role: master

  sfo-masters:
    provider: digitalocean
    region: sfo3
    count: 1
    role: master

  eu-masters:
    provider: digitalocean
    region: ams3
    count: 1
    role: master
```

---

## 🚨 Troubleshooting

### VPC Não Foi Criada

```bash
# Verificar outputs do Pulumi
pulumi stack output

# Ver logs
kubernetes-create deploy --config cluster.yaml --verbose
```

**Possíveis causas:**
- Token inválido
- Região não suportada
- CIDR inválido
- Limite de VPCs atingido

### VPN Não Conecta

```bash
# SSH no servidor VPN
kubernetes-create nodes ssh wireguard-vpn-server

# Verificar status
sudo systemctl status wg-quick@wg0
sudo wg show

# Ver logs
sudo journalctl -u wg-quick@wg0 -f
```

**Possíveis causas:**
- Firewall bloqueando porta 51820
- Chaves incorretas
- Endpoint errado

### Nodes Não Se Comunicam

```bash
# Testar conectividade
kubectl exec -it <pod> -- ping 10.8.0.2

# Verificar rotas
kubectl exec -it <pod> -- ip route

# Ver WireGuard status no node
ssh root@<node-ip> wg show
```

---

## 💰 Custos

### VPC

- **DigitalOcean**: Gratuito
- **Linode**: Gratuito

### VPN Server

- **DigitalOcean s-1vcpu-1gb**: ~$6/mês
- **Linode g6-nanode-1**: ~$5/mês

### Cluster Nodes

Depende do tamanho e quantidade de nodes escolhidos.

**Exemplo (configuração mínima)**:
- 1 VPN server: $5/mês
- 1 master (s-2vcpu-4gb): $18/mês
- 2 workers (s-2vcpu-4gb): $36/mês
- **Total**: ~$59/mês

---

## 🎉 Resumo

### Antes (Manual)

```bash
# 1. Criar VPC no DigitalOcean web UI
# 2. Criar VPC no Linode web UI
# 3. Criar servidor WireGuard
# 4. Configurar WireGuard manualmente
# 5. Criar VMs
# 6. Configurar WireGuard clients
# 7. Instalar Kubernetes
# 8. Configurar networking
# ... Horas de trabalho manual
```

### Agora (Automático)

```bash
# 1. Criar YAML
vim cluster.yaml

# 2. Deploy!
kubernetes-create deploy --config cluster.yaml

# Pronto! ✅
```

---

## 📚 Exemplos Completos

### Exemplo 1: Cluster Produção Multi-Cloud

Ver: [examples/cluster-with-vpc-vpn.yaml](./examples/cluster-with-vpc-vpn.yaml)

### Exemplo 2: Cluster Dev Single-Cloud

Ver: [examples/cluster-minimal-with-vpn.yaml](./examples/cluster-minimal-with-vpn.yaml)

---

## 🔗 Recursos

- [VPC Configuration Reference](./docs/VPC_CONFIGURATION.md)
- [WireGuard Configuration Reference](./docs/WIREGUARD_CONFIGURATION.md)
- [Network Architecture](./docs/NETWORK_ARCHITECTURE.md)
- [Security Best Practices](./docs/SECURITY.md)

---

**Infraestrutura completa em um único YAML. Deploy em um único comando. 🚀**
