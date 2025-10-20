# Design do CLI para Kubernetes Multi-Cloud

## Visão Geral

Transformar o projeto em uma CLI poderosa e intuitiva usando **Cobra** (framework CLI do Kubernetes).

---

## Estrutura de Comandos

```bash
kubernetes-create [command] [flags]

Commands:
  deploy      Deploy a new Kubernetes cluster
  destroy     Destroy an existing cluster
  status      Show cluster status
  config      Manage cluster configuration
  nodes       Manage cluster nodes
  kubeconfig  Get kubeconfig for cluster access
  version     Show CLI version

Global Flags:
  --config        Config file (default: ./cluster-config.yaml)
  --stack         Pulumi stack name (default: production)
  --verbose       Verbose output
  --dry-run       Show what would be done without executing
```

---

## Comandos Detalhados

### 1. `deploy` - Criar Cluster

```bash
kubernetes-create deploy [flags]

Deploy a new multi-cloud Kubernetes cluster with RKE2 and WireGuard VPN.

Flags:
  -c, --config string              Config file path (default "./cluster-config.yaml")
  -s, --stack string               Pulumi stack name (default "production")
      --do-token string            DigitalOcean API token
      --linode-token string        Linode API token
      --wireguard-endpoint string  WireGuard server endpoint
      --wireguard-pubkey string    WireGuard server public key
  -y, --yes                        Auto-approve deployment
      --dry-run                    Preview changes without applying
  -v, --verbose                    Verbose output

Examples:
  # Deploy using config file
  kubernetes-create deploy --config production.yaml

  # Deploy with inline tokens
  kubernetes-create deploy \
    --do-token xxx \
    --linode-token yyy \
    --wireguard-endpoint 1.2.3.4:51820 \
    --wireguard-pubkey "xxx="

  # Dry run to preview
  kubernetes-create deploy --dry-run

  # Auto-approve
  kubernetes-create deploy --yes
```

### 2. `destroy` - Destruir Cluster

```bash
kubernetes-create destroy [flags]

Destroy an existing Kubernetes cluster and all resources.

Flags:
  -s, --stack string    Pulumi stack name (default "production")
  -y, --yes             Auto-approve destruction
      --force           Force destroy even if resources have dependencies
  -v, --verbose         Verbose output

Examples:
  # Destroy with confirmation
  kubernetes-create destroy --stack production

  # Force destroy
  kubernetes-create destroy --force --yes
```

### 3. `status` - Ver Status

```bash
kubernetes-create status [flags]

Show cluster status and health information.

Flags:
  -s, --stack string    Pulumi stack name (default "production")
      --format string   Output format: table|json|yaml (default "table")
  -w, --watch           Watch for changes (refresh every 5s)

Examples:
  # Show status
  kubernetes-create status

  # JSON output
  kubernetes-create status --format json

  # Watch mode
  kubernetes-create status --watch
```

### 4. `config` - Gerenciar Configuração

```bash
kubernetes-create config <subcommand> [flags]

Manage cluster configuration.

Subcommands:
  generate    Generate a sample config file
  validate    Validate config file
  show        Show current configuration
  set         Set a configuration value
  get         Get a configuration value

Examples:
  # Generate sample config
  kubernetes-create config generate > cluster-config.yaml

  # Validate config
  kubernetes-create config validate --config production.yaml

  # Set a value
  kubernetes-create config set --stack production digitaloceanToken xxx

  # Get a value
  kubernetes-create config get --stack production clusterName
```

### 5. `nodes` - Gerenciar Nós

```bash
kubernetes-create nodes <subcommand> [flags]

Manage cluster nodes.

Subcommands:
  list        List all nodes
  add         Add nodes to cluster
  remove      Remove nodes from cluster
  ssh         SSH into a node

Examples:
  # List nodes
  kubernetes-create nodes list

  # SSH to a node
  kubernetes-create nodes ssh master-1

  # Add worker nodes
  kubernetes-create nodes add --count 2 --role worker --provider linode
```

### 6. `kubeconfig` - Obter Kubeconfig

```bash
kubernetes-create kubeconfig [flags]

Get kubeconfig for kubectl access.

Flags:
  -s, --stack string     Pulumi stack name (default "production")
  -o, --output string    Output file (default: stdout)
      --merge            Merge with existing ~/.kube/config

Examples:
  # Print to stdout
  kubernetes-create kubeconfig

  # Save to file
  kubernetes-create kubeconfig -o ~/.kube/config

  # Merge with existing
  kubernetes-create kubeconfig --merge
```

### 7. `version` - Versão

```bash
kubernetes-create version

Show CLI and cluster version information.

Examples:
  kubernetes-create version
```

---

## Estrutura de Código

```
cmd/
├── root.go              # Comando raiz e flags globais
├── deploy.go            # Comando deploy
├── destroy.go           # Comando destroy
├── status.go            # Comando status
├── config/              # Subcomandos config
│   ├── config.go
│   ├── generate.go
│   ├── validate.go
│   ├── show.go
│   ├── set.go
│   └── get.go
├── nodes/               # Subcomandos nodes
│   ├── nodes.go
│   ├── list.go
│   ├── add.go
│   ├── remove.go
│   └── ssh.go
├── kubeconfig.go        # Comando kubeconfig
└── version.go           # Comando version

internal/cli/            # Lógica CLI
├── output.go            # Formatação de output (table, json, yaml)
├── spinner.go           # Spinners e progress bars
├── prompt.go            # Prompts interativos
└── colors.go            # Output colorido

internal/pulumi/         # Integração Pulumi
├── runner.go            # Executa operações Pulumi
├── preview.go           # Preview de mudanças
└── stack.go             # Gerenciamento de stacks

main.go                  # Entry point
```

---

## Experiência do Usuário

### Deploy Interativo

```bash
$ kubernetes-create deploy

🚀 Kubernetes Multi-Cloud Deployment

✓ Loading configuration from cluster-config.yaml
✓ Validating configuration
✓ Checking cloud provider credentials

📋 Deployment Summary:
  • Cluster Name: production
  • Providers: DigitalOcean (3 nodes) + Linode (3 nodes)
  • Total Nodes: 6 (3 masters + 3 workers)
  • Kubernetes: RKE2
  • Network: WireGuard VPN Mesh

❓ Do you want to proceed? (y/N): y

🔧 Deploying cluster...

[1/5] ⏳ Generating SSH keys...                  ✓ Done (2s)
[2/5] ⏳ Creating cloud VMs (6 nodes)...          ⏳ In progress...
      • do-master-1: Creating...                 ✓ Done
      • do-worker-1: Creating...                 ✓ Done
      • do-worker-2: Creating...                 ✓ Done
      • linode-master-1: Creating...             ✓ Done
      • linode-master-2: Creating...             ✓ Done
      • linode-worker-1: Creating...             ✓ Done
                                                  ✓ Done (45s)
[3/5] ⏳ Setting up WireGuard VPN mesh...        ✓ Done (30s)
[4/5] ⏳ Installing RKE2 Kubernetes...           ✓ Done (120s)
[5/5] ⏳ Creating DNS records...                 ✓ Done (5s)

✅ Cluster deployed successfully!

📊 Cluster Information:
  • Name: production
  • API Endpoint: https://api.chalkan3.com.br:6443
  • Nodes: 6 (all ready)
  • Kubeconfig: Run 'kubernetes-create kubeconfig'

🎯 Next Steps:
  1. Get kubeconfig: kubernetes-create kubeconfig -o ~/.kube/config
  2. Check status: kubernetes-create status
  3. List nodes: kubectl get nodes
```

### Status com Tabela

```bash
$ kubernetes-create status

📊 Cluster Status: production

Overall Health: ✅ Healthy

Nodes:
┌──────────────────┬────────────┬──────────────┬──────────────┬────────────┬────────┐
│ NAME             │ PROVIDER   │ ROLE         │ STATUS       │ PUBLIC IP  │ REGION │
├──────────────────┼────────────┼──────────────┼──────────────┼────────────┼────────┤
│ do-master-1      │ DigitalOce │ master       │ ✅ Ready     │ 1.2.3.4    │ nyc3   │
│ linode-master-1  │ Linode     │ master       │ ✅ Ready     │ 5.6.7.8    │ us-eas │
│ linode-master-2  │ Linode     │ master       │ ✅ Ready     │ 9.10.11.12 │ us-eas │
│ do-worker-1      │ DigitalOce │ worker       │ ✅ Ready     │ 13.14.15.1 │ nyc3   │
│ do-worker-2      │ DigitalOce │ worker       │ ✅ Ready     │ 17.18.19.2 │ nyc3   │
│ linode-worker-1  │ Linode     │ worker       │ ✅ Ready     │ 21.22.23.2 │ us-eas │
└──────────────────┴────────────┴──────────────┴──────────────┴────────────┴────────┘

VPN Status: ✅ All nodes connected
RKE2 Status: ✅ Cluster operational
DNS Status: ✅ All records configured

API Endpoint: https://api.chalkan3.com.br:6443
```

### Config Generate

```bash
$ kubernetes-create config generate

# Generated sample configuration
metadata:
  name: production
  environment: production

providers:
  digitalocean:
    enabled: true
    region: nyc3
  linode:
    enabled: true
    region: us-east

network:
  dns:
    domain: example.com
    provider: digitalocean
  wireguard:
    enabled: true

node_pools:
  - name: do-masters
    provider: digitalocean
    count: 1
    size: s-2vcpu-4gb
    roles: [master]

  - name: do-workers
    provider: digitalocean
    count: 2
    size: s-2vcpu-4gb
    roles: [worker]

  - name: linode-masters
    provider: linode
    count: 2
    size: g6-standard-2
    roles: [master]

  - name: linode-workers
    provider: linode
    count: 1
    size: g6-standard-2
    roles: [worker]
```

---

## Dependências

```bash
# Adicionar ao go.mod
github.com/spf13/cobra        # Framework CLI
github.com/spf13/viper        # Configuração
github.com/fatih/color        # Output colorido
github.com/briandowns/spinner # Spinners
github.com/olekukonko/tablewriter # Tabelas
github.com/manifoldco/promptui # Prompts interativos
gopkg.in/yaml.v3              # YAML parsing
```

---

## Fluxo de Implementação

1. **Instalar Cobra CLI**
2. **Criar estrutura base** (`cmd/root.go`)
3. **Implementar comandos principais** (deploy, destroy, status)
4. **Adicionar helpers** (output, spinners, prompts)
5. **Integrar com Pulumi** (usar SDK programático)
6. **Adicionar subcomandos** (config, nodes)
7. **Testes e refinamento**

---

## Vantagens da CLI

✅ **Facilidade de Uso**
- Comandos intuitivos
- Help integrado
- Auto-complete

✅ **Experiência Melhorada**
- Output colorido
- Progress bars
- Confirmações interativas

✅ **Flexibilidade**
- Flags para todas as opções
- Múltiplos formatos de output
- Dry-run mode

✅ **Automação**
- Scriptável
- CI/CD friendly
- Flags `--yes` para auto-approve

---

## Exemplo de Uso Completo

```bash
# 1. Gerar config
kubernetes-create config generate > prod.yaml

# 2. Editar config
vim prod.yaml

# 3. Validar
kubernetes-create config validate --config prod.yaml

# 4. Preview deployment
kubernetes-create deploy --config prod.yaml --dry-run

# 5. Deploy
kubernetes-create deploy --config prod.yaml

# 6. Get kubeconfig
kubernetes-create kubeconfig -o ~/.kube/config

# 7. Check status
kubernetes-create status

# 8. Access cluster
kubectl get nodes

# 9. Destroy when done
kubernetes-create destroy --yes
```

---

## Pronto para Implementar! 🚀
