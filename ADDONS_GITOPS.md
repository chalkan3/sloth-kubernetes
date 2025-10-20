# 📦 Addons via GitOps - Guia Completo

## Visão Geral

O sistema de addons do `kubernetes-create` usa uma abordagem **GitOps-first**, onde um repositório Git é a fonte única da verdade para todos os addons do cluster.

### Como Funciona

```
┌─────────────────────────────────────────────────────────────┐
│  1. Você cria um repositório Git com manifests             │
│     └── addons/                                             │
│         ├── argocd/          (ArgoCD manifests)            │
│         ├── ingress-nginx/   (NGINX Ingress)               │
│         ├── cert-manager/    (Cert Manager)                │
│         └── prometheus/      (Monitoring)                  │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  2. Bootstrap: kubernetes-create addons bootstrap           │
│     • Clona o repositório                                   │
│     • Aplica ArgoCD via kubectl                            │
│     • Configura ArgoCD para assistir o repo                │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  3. ArgoCD assume o controle                               │
│     • Monitora o repositório Git                           │
│     • Auto-sync de todos os addons                         │
│     • Self-heal quando há drift                            │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  4. Você apenas faz commits                                │
│     • git add addons/new-addon/                            │
│     • git commit -m "Add new addon"                        │
│     • git push                                              │
│     • ArgoCD aplica automaticamente! ✅                     │
└─────────────────────────────────────────────────────────────┘
```

---

## 🎯 Comandos Disponíveis

```bash
kubernetes-create addons bootstrap --repo <url>  # Bootstrap ArgoCD
kubernetes-create addons list                    # Listar addons
kubernetes-create addons status                  # Status do ArgoCD
kubernetes-create addons sync                    # Force sync
kubernetes-create addons template                # Template de repo
```

---

## 🚀 Quick Start

### 1. Criar Repositório GitOps

```bash
# Gerar template
kubernetes-create addons template --output my-gitops-repo

cd my-gitops-repo
tree
```

**Estrutura gerada:**

```
my-gitops-repo/
├── README.md
├── addons/
│   ├── argocd/
│   │   ├── namespace.yaml
│   │   └── install.yaml
│   ├── ingress-nginx/
│   ├── cert-manager/
│   ├── prometheus/
│   └── longhorn/
└── apps/
```

### 2. Customizar Addons

```bash
# Adicionar NGINX Ingress
cat > addons/ingress-nginx/namespace.yaml <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: ingress-nginx
EOF

cat > addons/ingress-nginx/helmrelease.yaml <<EOF
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: ingress-nginx
  namespace: kube-system
spec:
  chart: ingress-nginx
  repo: https://kubernetes.github.io/ingress-nginx
  targetNamespace: ingress-nginx
  version: 4.8.3
EOF
```

### 3. Commit e Push

```bash
git init
git add .
git commit -m "Initial GitOps setup"
git remote add origin git@github.com:you/gitops-repo.git
git push -u origin main
```

### 4. Bootstrap no Cluster

```bash
# Repositório público
kubernetes-create addons bootstrap \
  --repo https://github.com/you/gitops-repo

# Repositório privado (SSH)
kubernetes-create addons bootstrap \
  --repo git@github.com:you/private-repo.git \
  --private-key ~/.ssh/id_rsa

# Com branch e path específicos
kubernetes-create addons bootstrap \
  --repo https://github.com/you/gitops-repo \
  --branch main \
  --path addons/
```

### 5. Acessar ArgoCD UI

```bash
# Pegar senha do admin
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath='{.data.password}' | base64 -d

# Port-forward
kubectl port-forward svc/argocd-server -n argocd 8080:443

# Abrir no browser
open https://localhost:8080
# Username: admin
# Password: (output do comando acima)
```

---

## 📋 Comandos Detalhados

### 1. Bootstrap

```bash
kubernetes-create addons bootstrap --repo <url>
```

**O que acontece:**

1. **Clone do repo** - Repositório é clonado temporariamente
2. **Instala ArgoCD** - Aplica manifests do ArgoCD via kubectl
3. **Configura Application** - Cria ArgoCD Application apontando para o repo
4. **Auto-sync** - ArgoCD começa a monitorar e aplicar addons

**Flags:**

| Flag | Descrição | Padrão |
|------|-----------|--------|
| `--repo` | URL do repositório Git (required) | - |
| `--branch` | Branch a usar | main |
| `--path` | Path dentro do repo | addons/ |
| `--private-key` | Chave SSH para repos privados | - |

**Exemplos:**

```bash
# Repositório público GitHub
kubernetes-create addons bootstrap \
  --repo https://github.com/myorg/k8s-gitops

# Repositório privado com SSH
kubernetes-create addons bootstrap \
  --repo git@github.com:myorg/private-gitops.git \
  --private-key ~/.ssh/deploy_key

# Branch e path customizados
kubernetes-create addons bootstrap \
  --repo https://gitlab.com/myorg/infra \
  --branch production \
  --path clusters/prod/addons/

# Sem confirmação (CI/CD)
kubernetes-create addons bootstrap \
  --repo https://github.com/myorg/gitops \
  --yes
```

**Output:**

```
🚀 Bootstrap ArgoCD via GitOps

📋 Bootstrap Configuration:
  • Repository: https://github.com/you/gitops-repo
  • Branch: main
  • Path: addons/

⚠️  This will:
  1. Clone your GitOps repository
  2. Install ArgoCD from the repo
  3. Configure ArgoCD to watch the repo
  4. ArgoCD will auto-sync all addons

Do you want to proceed? (y/N): y

✅ Cluster found
✅ Repository cloned
✅ ArgoCD bootstrapped successfully!

🎉 Success! ArgoCD is now watching your repository

📝 Next Steps:

  1. Get ArgoCD admin password:
     kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d

  2. Port-forward to ArgoCD UI:
     kubectl port-forward svc/argocd-server -n argocd 8080:443

  3. Access ArgoCD:
     https://localhost:8080
     Username: admin
     Password: (from step 1)

  4. Add more addons by committing to your repo:
     https://github.com/you/gitops-repo/addons/

💡 View addon status:
   kubernetes-create addons status
```

### 2. List

```bash
kubernetes-create addons list
```

Lista todos os addons instalados no cluster.

**Output:**

```
📦 Installed Addons

Addons:

NAME            CATEGORY     STATUS       VERSION    NAMESPACE
----            --------     ------       -------    ---------
argocd          CD           ✅ Running   v2.9.3     argocd
ingress-nginx   Ingress      ✅ Running   v4.8.3     ingress-nginx
cert-manager    Security     ✅ Running   v1.13.3    cert-manager
prometheus      Monitoring   ✅ Running   v55.5.0    monitoring
longhorn        Storage      ✅ Running   v1.5.3     longhorn-system

📊 Summary:
  • Total Addons: 5
  • Running: 5
  • Failed: 0

  ✅ All addons healthy
```

### 3. Status

```bash
kubernetes-create addons status
```

Mostra status detalhado do ArgoCD e todas as Applications.

**Output:**

```
📊 ArgoCD & Addon Status

ArgoCD Server:
  Status: ✅ Running
  Version: v2.9.3
  Namespace: argocd

Applications:

NAME             SYNC STATUS   HEALTH       NAMESPACE        REPO
----             -----------   ------       ---------        ----
cluster-addons   ✅ Synced     ✅ Healthy   argocd           github.com/user/gitops
ingress-nginx    ✅ Synced     ✅ Healthy   ingress-nginx    Auto-synced
cert-manager     ✅ Synced     ✅ Healthy   cert-manager     Auto-synced

💡 View in ArgoCD UI:
   kubectl port-forward svc/argocd-server -n argocd 8080:443
   https://localhost:8080
```

### 4. Sync

```bash
kubernetes-create addons sync
```

Força sincronização manual de todas as Applications.

**Quando usar:**

- Após fazer um commit no repo GitOps
- Quando auto-sync está desabilitado
- Para forçar re-aplicação imediata

**Flags:**

| Flag | Descrição |
|------|-----------|
| `--app` | Sync apenas uma Application específica |

**Exemplos:**

```bash
# Sync tudo
kubernetes-create addons sync

# Sync apenas cert-manager
kubernetes-create addons sync --app cert-manager
```

### 5. Template

```bash
kubernetes-create addons template
```

Gera estrutura de exemplo para repositório GitOps.

**Flags:**

| Flag | Descrição |
|------|-----------|
| `-o, --output` | Diretório de saída |

**Exemplos:**

```bash
# Imprimir estrutura
kubernetes-create addons template

# Gerar diretório
kubernetes-create addons template --output my-gitops-repo

# Gerar e inicializar repo
kubernetes-create addons template -o gitops-repo
cd gitops-repo
git init
git add .
git commit -m "Initial commit"
```

---

## 📁 Estrutura do Repositório GitOps

### Estrutura Recomendada

```
gitops-repo/
├── README.md                    # Documentação
├── addons/                      # Cluster addons
│   ├── argocd/                  # ArgoCD (bootstrapped first)
│   │   ├── namespace.yaml
│   │   └── install.yaml
│   │
│   ├── ingress-nginx/           # Ingress Controller
│   │   ├── namespace.yaml
│   │   ├── helmchart.yaml
│   │   └── values.yaml
│   │
│   ├── cert-manager/            # Certificate Management
│   │   ├── namespace.yaml
│   │   ├── helmchart.yaml
│   │   └── clusterissuer.yaml
│   │
│   ├── prometheus/              # Monitoring
│   │   ├── namespace.yaml
│   │   ├── helmchart.yaml
│   │   └── values.yaml
│   │
│   └── longhorn/                # Storage
│       ├── namespace.yaml
│       ├── helmchart.yaml
│       └── storageclass.yaml
│
└── apps/                        # Your applications
    ├── backend/
    ├── frontend/
    └── database/
```

### Exemplo de Addon: NGINX Ingress

**addons/ingress-nginx/namespace.yaml:**

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: ingress-nginx
```

**addons/ingress-nginx/helmchart.yaml:**

```yaml
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: ingress-nginx
  namespace: kube-system
spec:
  chart: ingress-nginx
  repo: https://kubernetes.github.io/ingress-nginx
  targetNamespace: ingress-nginx
  version: 4.8.3
  valuesContent: |-
    controller:
      service:
        type: LoadBalancer
      replicaCount: 2
      resources:
        requests:
          cpu: 100m
          memory: 128Mi
```

### Exemplo de Addon: Cert Manager

**addons/cert-manager/namespace.yaml:**

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: cert-manager
```

**addons/cert-manager/helmchart.yaml:**

```yaml
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: cert-manager
  namespace: kube-system
spec:
  chart: cert-manager
  repo: https://charts.jetstack.io
  targetNamespace: cert-manager
  version: v1.13.3
  valuesContent: |-
    installCRDs: true
    global:
      leaderElection:
        namespace: cert-manager
```

**addons/cert-manager/clusterissuer.yaml:**

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
```

---

## 🔄 Workflow Completo

### 1. Setup Inicial

```bash
# 1. Gerar template
kubernetes-create addons template -o k8s-gitops

# 2. Customizar addons
cd k8s-gitops
# Editar manifests conforme necessário

# 3. Init repo
git init
git add .
git commit -m "Initial GitOps setup"

# 4. Push para GitHub/GitLab
git remote add origin git@github.com:you/k8s-gitops.git
git push -u origin main

# 5. Bootstrap no cluster
kubernetes-create addons bootstrap --repo git@github.com:you/k8s-gitops.git
```

### 2. Adicionar Novo Addon

```bash
# 1. Criar diretório do addon
mkdir -p addons/redis/

# 2. Adicionar manifests
cat > addons/redis/namespace.yaml <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: redis
EOF

cat > addons/redis/helmchart.yaml <<EOF
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: redis
  namespace: kube-system
spec:
  chart: redis
  repo: https://charts.bitnami.com/bitnami
  targetNamespace: redis
  version: 18.4.0
  valuesContent: |-
    auth:
      enabled: true
      password: changeme
    master:
      persistence:
        enabled: true
        size: 8Gi
EOF

# 3. Commit e push
git add addons/redis/
git commit -m "Add Redis addon"
git push

# 4. ArgoCD aplica automaticamente!
# Verificar status
kubernetes-create addons status
```

### 3. Atualizar Addon Existente

```bash
# 1. Editar manifest
vim addons/prometheus/values.yaml
# Mudar configurações...

# 2. Commit e push
git add addons/prometheus/
git commit -m "Update Prometheus resources"
git push

# 3. ArgoCD auto-sync (ou force sync)
kubernetes-create addons sync --app prometheus

# 4. Verificar
kubectl get pods -n monitoring
```

### 4. Remover Addon

```bash
# 1. Remover diretório
git rm -r addons/redis/
git commit -m "Remove Redis addon"
git push

# 2. ArgoCD prune automaticamente (se configurado)
# Ou manualmente:
kubectl delete namespace redis
```

---

## 🎨 Customizações Avançadas

### Multi-Environment

Organize por ambiente:

```
gitops-repo/
├── clusters/
│   ├── production/
│   │   └── addons/
│   │       ├── ingress-nginx/
│   │       ├── cert-manager/
│   │       └── prometheus/
│   │
│   ├── staging/
│   │   └── addons/
│   │       ├── ingress-nginx/
│   │       └── cert-manager/
│   │
│   └── development/
│       └── addons/
│           └── ingress-nginx/
```

Bootstrap por ambiente:

```bash
# Production
kubernetes-create addons bootstrap \
  --repo https://github.com/you/gitops \
  --path clusters/production/addons/

# Staging
kubernetes-create addons bootstrap \
  --repo https://github.com/you/gitops \
  --path clusters/staging/addons/
```

### Kustomize

Use Kustomize para overlays:

```
addons/ingress-nginx/
├── base/
│   ├── kustomization.yaml
│   └── helmchart.yaml
├── overlays/
│   ├── production/
│   │   ├── kustomization.yaml
│   │   └── patches.yaml
│   └── staging/
│       ├── kustomization.yaml
│       └── patches.yaml
```

### Helm Values por Ambiente

```yaml
# addons/prometheus/helmchart.yaml
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: prometheus
  namespace: kube-system
spec:
  chart: kube-prometheus-stack
  repo: https://prometheus-community.github.io/helm-charts
  targetNamespace: monitoring
  version: 55.5.0
  valuesContent: |-
    prometheus:
      prometheusSpec:
        # Production: 30 dias retenção
        retention: 30d
        storageSpec:
          volumeClaimTemplate:
            spec:
              resources:
                requests:
                  storage: 100Gi
    grafana:
      persistence:
        enabled: true
        size: 10Gi
```

---

## 🔐 Repositórios Privados

### SSH Key

```bash
# 1. Gerar deploy key
ssh-keygen -t ed25519 -C "argocd-deploy" -f ~/.ssh/argocd-deploy

# 2. Adicionar no GitHub/GitLab
# Settings -> Deploy Keys -> Add key
cat ~/.ssh/argocd-deploy.pub

# 3. Bootstrap com private key
kubernetes-create addons bootstrap \
  --repo git@github.com:myorg/private-gitops.git \
  --private-key ~/.ssh/argocd-deploy
```

### HTTPS com Token

```bash
# GitHub Personal Access Token
kubernetes-create addons bootstrap \
  --repo https://github-token:ghp_xxxxxxxxxxxx@github.com/myorg/private-repo
```

### ArgoCD Secret Manual

```bash
# Criar secret com SSH key
kubectl create secret generic private-repo-creds \
  -n argocd \
  --from-file=sshPrivateKey=~/.ssh/argocd-deploy

# Configurar no ArgoCD Application
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: private-repo
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: repository
stringData:
  type: git
  url: git@github.com:myorg/private-repo.git
  sshPrivateKey: |
    $(cat ~/.ssh/argocd-deploy)
EOF
```

---

## 📊 Monitoramento

### Ver Status via kubectl

```bash
# Ver Applications
kubectl get applications -n argocd

# Ver Application específica
kubectl get application cluster-addons -n argocd -o yaml

# Ver sync status
kubectl get applications -n argocd -o custom-columns=\
NAME:.metadata.name,\
SYNC:.status.sync.status,\
HEALTH:.status.health.status

# Logs do ArgoCD
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-server -f
```

### ArgoCD CLI

```bash
# Instalar ArgoCD CLI
brew install argocd  # macOS
# ou
curl -sSL -o /usr/local/bin/argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
chmod +x /usr/local/bin/argocd

# Login
argocd login localhost:8080

# List apps
argocd app list

# Sync app
argocd app sync cluster-addons

# Get app details
argocd app get cluster-addons
```

---

## 🚨 Troubleshooting

### ArgoCD Não Sincroniza

```bash
# 1. Verificar Application
kubectl get application cluster-addons -n argocd -o yaml

# 2. Ver eventos
kubectl describe application cluster-addons -n argocd

# 3. Forçar refresh
kubectl patch application cluster-addons -n argocd \
  --type merge -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'

# 4. Sync manual
kubernetes-create addons sync
```

### Addon Não Aparece

```bash
# 1. Verificar se ArgoCD está vendo o addon
argocd app get cluster-addons --show-operation

# 2. Verificar se path está correto
# ArgoCD procura em addons/ por padrão

# 3. Verificar se manifests são válidos
kubectl apply --dry-run=client -f addons/myaddon/

# 4. Ver logs do ArgoCD
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-application-controller -f
```

### Sync Falha

```bash
# 1. Ver erro na Application
kubectl get application <name> -n argocd -o yaml | grep -A 10 status

# 2. Ver último sync operation
argocd app get <name> --show-operation

# 3. Tentar sync com prune
argocd app sync <name> --prune

# 4. Reset se necessário
argocd app delete <name>
# Recriar application
```

### Repositório Não Conecta

```bash
# SSH: Verificar key
ssh -T git@github.com

# Verificar secret do ArgoCD
kubectl get secret -n argocd | grep repo

# Testar clone manual
git clone <repo-url> /tmp/test-clone

# Ver logs de connection
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-repo-server
```

---

## 💡 Boas Práticas

### 1. Organize por Layers

```
addons/
├── 00-bootstrap/        # ArgoCD
├── 01-infrastructure/   # Ingress, Storage, CNI
├── 02-security/         # Cert-manager, Sealed Secrets
├── 03-monitoring/       # Prometheus, Loki
└── 04-applications/     # App-specific addons
```

### 2. Use Annotations

```yaml
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: ingress-nginx
  namespace: kube-system
  annotations:
    # Desabilitar auto-sync para este addon
    argocd.argoproj.io/sync-options: "Prune=false"
```

### 3. Health Checks

```yaml
# Custom health check para addon
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-addon
spec:
  ignoreDifferences:
  - group: apps
    kind: Deployment
    jsonPointers:
    - /spec/replicas  # Ignorar mudanças em replicas (HPA)
```

### 4. Sync Waves

```yaml
# Aplicar em ordem específica
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"  # Aplica primeiro
```

### 5. Notifications

Configure notificações do ArgoCD:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-notifications-cm
  namespace: argocd
data:
  service.slack: |
    token: $slack-token
  trigger.on-deployed: |
    - send: [app-deployed]
  template.app-deployed: |
    message: |
      Application {{.app.metadata.name}} deployed!
```

---

## 📚 Recursos Úteis

### Documentação

- [ArgoCD Docs](https://argo-cd.readthedocs.io/)
- [GitOps Guide](https://www.gitops.tech/)
- [Kubernetes Docs](https://kubernetes.io/docs/)

### Addons Populares

| Addon | Descrição | Repo |
|-------|-----------|------|
| ingress-nginx | NGINX Ingress Controller | https://kubernetes.github.io/ingress-nginx |
| cert-manager | Certificate Management | https://charts.jetstack.io |
| prometheus | Monitoring Stack | https://prometheus-community.github.io/helm-charts |
| longhorn | Distributed Storage | https://charts.longhorn.io |
| sealed-secrets | Encrypted Secrets | https://bitnami-labs.github.io/sealed-secrets |
| external-dns | DNS Management | https://kubernetes-sigs.github.io/external-dns |
| velero | Backup/Restore | https://vmware-tanzu.github.io/helm-charts |

---

## 🎉 Resumo

### Workflow GitOps

1. **Criar repo** → `kubernetes-create addons template`
2. **Customizar** → Editar manifests
3. **Bootstrap** → `kubernetes-create addons bootstrap --repo <url>`
4. **Commit & Push** → ArgoCD auto-sync! ✅

### Vantagens

✅ **Git como fonte da verdade** - Tudo versionado
✅ **Declarativo** - Apenas descreva o estado desejado
✅ **Auto-sync** - ArgoCD mantém cluster sincronizado
✅ **Rollback fácil** - `git revert` e pronto
✅ **Auditoria** - Git log mostra todas as mudanças
✅ **Multi-cluster** - Um repo, vários clusters

Happy GitOps! 🚀
