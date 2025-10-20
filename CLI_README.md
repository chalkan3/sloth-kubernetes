# 🚀 Kubernetes-Create CLI

**CLI standalone para criar clusters Kubernetes multi-cloud** - Sem precisar do Pulumi CLI instalado!

---

## ✨ O Que Foi Criado

Uma **CLI completa em Go** que usa o **Pulumi Automation API** para gerenciar clusters Kubernetes sem precisar ter a CLI do Pulumi instalada.

### Características

✅ **Standalone** - Não precisa de Pulumi CLI
✅ **Automation API** - Usa Pulumi programaticamente
✅ **Interface Interativa** - Spinners, cores, progress bars
✅ **Multi-Cloud** - DigitalOcean + Linode
✅ **Kubernetes RKE2** - Distribuição production-ready
✅ **WireGuard VPN** - Mesh network privada
✅ **Comandos Simples** - Fácil de usar

---

## 📦 Instalação

### Opção 1: Compilar do Código

```bash
# Clone o repositório
git clone https://github.com/your-username/kubernetes-create.git
cd kubernetes-create

# Compile
go build -o kubernetes-create main.go

# Instale
sudo mv kubernetes-create /usr/local/bin/

# Teste
kubernetes-create --help
```

### Opção 2: Download do Binário

```bash
# Download (quando disponível)
curl -LO https://github.com/your-username/kubernetes-create/releases/latest/download/kubernetes-create

# Permissão de execução
chmod +x kubernetes-create

# Instale
sudo mv kubernetes-create /usr/local/bin/
```

---

## 🎯 Comandos Disponíveis

```bash
kubernetes-create [command] [flags]

Comandos:
  deploy      Deploy a new Kubernetes cluster
  destroy     Destroy an existing cluster
  status      Show cluster status
  kubeconfig  Get kubeconfig for kubectl access
  version     Show CLI version
  help        Help about any command
```

---

## 📚 Guia de Uso

### 1. Deploy de um Cluster

#### Modo Interativo

```bash
kubernetes-create deploy \
  --do-token YOUR_DO_TOKEN \
  --linode-token YOUR_LINODE_TOKEN \
  --wireguard-endpoint 1.2.3.4:51820 \
  --wireguard-pubkey "xxxxx="
```

**Output:**
```
🚀 Kubernetes Multi-Cloud Deployment

✓ Configuration loaded
✓ Configuration validated

📋 Deployment Summary:
  • Cluster Name: production
  • Providers: DigitalOcean + Linode
  • Total Nodes: 6 (3 masters + 3 workers)
  • Kubernetes: RKE2
  • Network: WireGuard VPN Mesh

❓ Do you want to proceed with deployment? (y/N): y

🔧 Setting up Pulumi stack...
✓ Pulumi stack configured

🚀 Deploying cluster...

[Creates resources with progress...]

✅ Cluster deployed successfully!

📊 Cluster Information:
  • Name: production
  • API Endpoint: https://api.chalkan3.com.br:6443

🎯 Next Steps:
  1. Get kubeconfig: kubernetes-create kubeconfig -o ~/.kube/config
  2. Check status: kubernetes-create status
  3. List nodes: kubectl get nodes
```

#### Preview (Dry-Run)

```bash
# Ver o que será criado sem aplicar
kubernetes-create deploy --dry-run
```

#### Auto-Approve (Para CI/CD)

```bash
# Deploy sem confirmação
kubernetes-create deploy --yes \
  --do-token $DO_TOKEN \
  --linode-token $LINODE_TOKEN \
  --wireguard-endpoint $WG_ENDPOINT \
  --wireguard-pubkey $WG_PUBKEY
```

### 2. Verificar Status

```bash
kubernetes-create status
```

**Output:**
```
📊 Cluster Status: production

Overall Health: ✅ Healthy

Cluster Name: production
API Endpoint: https://api.chalkan3.com.br:6443

Nodes:
NAME              PROVIDER      ROLE    STATUS     REGION
----              --------      ----    ------     ------
do-master-1       DigitalOcean  master  ✅ Ready  nyc3
linode-master-1   Linode        master  ✅ Ready  us-east
linode-master-2   Linode        master  ✅ Ready  us-east
do-worker-1       DigitalOcean  worker  ✅ Ready  nyc3
do-worker-2       DigitalOcean  worker  ✅ Ready  nyc3
linode-worker-1   Linode        worker  ✅ Ready  us-east

VPN Status: ✅ All nodes connected
RKE2 Status: ✅ Cluster operational
DNS Status: ✅ All records configured
```

### 3. Obter Kubeconfig

```bash
# Salvar em arquivo
kubernetes-create kubeconfig -o ~/.kube/config

# Ou imprimir no terminal
kubernetes-create kubeconfig
```

### 4. Acessar o Cluster

```bash
# Configurar kubectl
export KUBECONFIG=~/.kube/config

# Verificar nodes
kubectl get nodes

# Ver pods
kubectl get pods --all-namespaces
```

### 5. Destruir o Cluster

```bash
# Com confirmação
kubernetes-create destroy

# Sem confirmação (cuidado!)
kubernetes-create destroy --yes
```

---

## 🔧 Variáveis de Ambiente

Em vez de passar flags, você pode usar variáveis de ambiente:

```bash
export DIGITALOCEAN_TOKEN="dop_xxx"
export LINODE_TOKEN="xxx"
export WIREGUARD_ENDPOINT="1.2.3.4:51820"
export WIREGUARD_PUBKEY="xxxxx="

# Deploy sem flags
kubernetes-create deploy
```

---

## ⚙️ Configuração Avançada

### Stacks Múltiplos

```bash
# Deploy para staging
kubernetes-create deploy --stack staging

# Deploy para production
kubernetes-create deploy --stack production

# Status de staging
kubernetes-create status --stack staging
```

### Verbose Mode

```bash
# Ver mais detalhes
kubernetes-create deploy --verbose

# Ou
kubernetes-create -v deploy
```

---

## 🎨 Recursos da Interface

### Spinners Animados
```
⠋ Loading configuration...
⠙ Validating configuration...
⠹ Setting up Pulumi stack...
```

### Cores e Emojis
- 🚀 Headers importantes
- ✓ Operações bem-sucedidas
- ⚠️ Avisos
- ❌ Erros
- 📊 Informações
- 🎯 Próximos passos

### Progress Bars
Durante o deploy, você verá o progresso em tempo real:
```
Deploying kubernetes-cluster...
 + 15 resources created
   4 resources updated
   0 resources deleted
```

---

## 🔒 Segurança

### Tokens Sensíveis

Os tokens são marcados como secrets no Pulumi:
```go
configs := map[string]auto.ConfigValue{
    "digitaloceanToken": {Value: token, Secret: true},
    "linodeToken":       {Value: token, Secret: true},
}
```

### Kubeconfig

O kubeconfig é salvo com permissões seguras:
```go
os.WriteFile(outputFile, []byte(kubeConfig), 0600)  // rw-------
```

---

## 📖 Exemplos Completos

### Deploy Completo

```bash
#!/bin/bash
# deploy-cluster.sh

# Configurar variáveis
export DIGITALOCEAN_TOKEN="dop_xxxxx"
export LINODE_TOKEN="xxxxxx"
export WIREGUARD_ENDPOINT="1.2.3.4:51820"
export WIREGUARD_PUBKEY="xxxxx="

# Deploy
kubernetes-create deploy --yes --stack production

# Aguardar alguns minutos para cluster ficar pronto...
sleep 60

# Obter kubeconfig
kubernetes-create kubeconfig -o ~/.kube/config

# Verificar nodes
kubectl get nodes

# Deploy de aplicação exemplo
kubectl apply -f https://k8s.io/examples/application/deployment.yaml

echo "✅ Cluster pronto!"
```

### CI/CD Pipeline

```yaml
# .github/workflows/deploy-k8s.yml
name: Deploy Kubernetes

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Setup Go
        uses: actions/setup-go@v2
        with:
          go-version: '1.21'

      - name: Build CLI
        run: go build -o kubernetes-create main.go

      - name: Deploy Cluster
        env:
          DIGITALOCEAN_TOKEN: ${{ secrets.DO_TOKEN }}
          LINODE_TOKEN: ${{ secrets.LINODE_TOKEN }}
          WIREGUARD_ENDPOINT: ${{ secrets.WG_ENDPOINT }}
          WIREGUARD_PUBKEY: ${{ secrets.WG_PUBKEY }}
        run: |
          ./kubernetes-create deploy --yes --stack production

      - name: Get Kubeconfig
        run: ./kubernetes-create kubeconfig -o kubeconfig.yaml

      - name: Deploy App
        run: |
          export KUBECONFIG=kubeconfig.yaml
          kubectl apply -f manifests/
```

---

## 🐛 Troubleshooting

### "Failed to select stack"

```bash
# Verifique se o stack existe
ls -la ~/.pulumi/stacks/kubernetes-create/

# Crie manualmente se necessário
mkdir -p ~/.pulumi/stacks/kubernetes-create/production/
```

### "Configuration validation failed"

```bash
# Verifique as variáveis
echo $DIGITALOCEAN_TOKEN
echo $LINODE_TOKEN
echo $WIREGUARD_ENDPOINT
echo $WIREGUARD_PUBKEY

# Ou use flags explicitamente
kubernetes-create deploy --do-token xxx --linode-token yyy ...
```

### "No cluster found"

```bash
# Verifique o nome do stack
kubernetes-create status --stack production

# Liste todos os stacks
ls ~/.pulumi/stacks/kubernetes-create/
```

---

## 🔄 Comparação: CLI vs Pulumi Direto

### Antes (Pulumi CLI)

```bash
# Necessário ter Pulumi CLI instalado
pulumi login
pulumi stack select production
pulumi config set digitaloceanToken xxx --secret
pulumi config set linodeToken yyy --secret
pulumi config set wireguardServerEndpoint zzz
pulumi config set wireguardServerPublicKey www
pulumi up

# Para cada comando...
pulumi stack output kubeConfig > ~/.kube/config
```

### Agora (kubernetes-create CLI)

```bash
# Tudo em um comando!
kubernetes-create deploy \
  --do-token xxx \
  --linode-token yyy \
  --wireguard-endpoint zzz \
  --wireguard-pubkey www

# Kubeconfig em um comando
kubernetes-create kubeconfig -o ~/.kube/config
```

**Benefícios:**
- ✅ Sem dependência externa
- ✅ Interface mais amigável
- ✅ Menos comandos
- ✅ Melhor UX
- ✅ Um binário standalone

---

## 📦 Arquitetura Interna

### Como Funciona

```
kubernetes-create CLI
        │
        ├─> Cobra (CLI Framework)
        │
        ├─> Pulumi Automation API
        │   ├─> auto.UpsertStackInlineSource()
        │   ├─> stack.Up(ctx)
        │   └─> stack.Outputs(ctx)
        │
        └─> internal/orchestrator
            ├─> Components (node_deployment, wireguard, rke2, dns)
            └─> Providers (DigitalOcean, Linode)
```

### Diferença vs Pulumi CLI

| Aspecto | Pulumi CLI | kubernetes-create |
|---------|-----------|-------------------|
| **Instalação** | Separada | Tudo incluído |
| **Dependências** | Pulumi CLI + Go | Apenas o binário |
| **Comandos** | `pulumi up`, `pulumi destroy` | `kubernetes-create deploy`, etc |
| **UX** | Terminal genérico | Interface customizada |
| **Automação** | Possível mas complexo | Simples e direto |

---

## 🚀 Próximas Melhorias

Possíveis adições futuras:

- [ ] **Config YAML** - Suporte a arquivo de configuração
- [ ] **Nodes add/remove** - Adicionar/remover nós dinamicamente
- [ ] **SSH command** - SSH direto nos nós via CLI
- [ ] **Logs** - Ver logs dos pods via CLI
- [ ] **Scale** - Escalar workers
- [ ] **Backup** - Backup do etcd
- [ ] **Restore** - Restaurar backup
- [ ] **Upgrade** - Upgrade do Kubernetes
- [ ] **Auto-complete** - Completação de comandos
- [ ] **Plugins** - Sistema de plugins

---

## 📝 Notas Importantes

1. **Pulumi State** - O state é armazenado localmente em `~/.pulumi/`
2. **Secrets** - Tokens são criptografados no state
3. **Idempotência** - Pode rodar `deploy` múltiplas vezes
4. **Destruição** - `destroy` remove TUDO (cuidado!)
5. **Stacks** - Múltiplos ambientes via stacks

---

## 🎉 Conclusão

Você agora tem uma **CLI standalone completa** que:

✅ **Não precisa de Pulumi CLI**
✅ **Interface amigável** com cores e spinners
✅ **Deploy multi-cloud** em um comando
✅ **Automation API** integrada
✅ **Production-ready**

**Comandos principais:**
```bash
kubernetes-create deploy    # Criar cluster
kubernetes-create status    # Ver estado
kubernetes-create kubeconfig # Obter acesso
kubernetes-create destroy   # Destruir
```

🚀 **Happy clustering!**
