# 📊 Node Management - Guia Completo

## Visão Geral

O comando `kubernetes-create nodes` permite gerenciar completamente os nodes do seu cluster Kubernetes, incluindo listar, adicionar, remover, acessar via SSH e fazer upgrade.

---

## 🎯 Comandos Disponíveis

```bash
kubernetes-create nodes list              # Listar todos os nodes
kubernetes-create nodes add --count 2     # Adicionar workers
kubernetes-create nodes remove <name>     # Remover node
kubernetes-create nodes ssh <name>        # SSH no node
kubernetes-create nodes upgrade           # Upgrade Kubernetes
```

---

## 📋 1. Listar Nodes

### Uso Básico

```bash
# Listar todos os nodes do cluster
kubernetes-create nodes list
```

### Output Exemplo

```
📊 Cluster Nodes

Nodes:

NAME             ROLE     PROVIDER       REGION    STATUS      IP ADDRESS
----             ----     --------       ------    ------      ----------
master-1         master   DigitalOcean   nyc3      ✅ Ready    167.71.1.1
master-2         master   Linode         us-east   ✅ Ready    172.105.1.1
master-3         master   Linode         us-east   ✅ Ready    172.105.1.2
worker-1         worker   DigitalOcean   nyc3      ✅ Ready    167.71.1.2
worker-2         worker   DigitalOcean   nyc3      ✅ Ready    167.71.1.3
worker-3         worker   Linode         us-east   ✅ Ready    172.105.1.3

📊 Summary:

  • Total Nodes: 6
  • Masters: 3 (HA)
  • Workers: 3

  • DigitalOcean: 3 nodes
  • Linode: 3 nodes

  ✅ All nodes healthy
```

### Com Stack Específico

```bash
# Listar nodes de um stack específico
kubernetes-create nodes list --stack production
kubernetes-create nodes list --stack staging
```

### Informações Mostradas

- ✅ **Nome do node**
- ✅ **Role** (master/worker)
- ✅ **Provider** (DigitalOcean/Linode)
- ✅ **Região**
- ✅ **Status** (Ready/NotReady)
- ✅ **Endereço IP público**
- ✅ **Resumo do cluster**

---

## ➕ 2. Adicionar Nodes

### Adicionar Workers (Simples)

```bash
# Adicionar 2 workers com configuração padrão
kubernetes-create nodes add --count 2
```

### Adicionar com Configurações Específicas

```bash
# Adicionar 3 workers com size específico
kubernetes-create nodes add \
  --count 3 \
  --size s-4vcpu-8gb \
  --role worker

# Adicionar em provider específico
kubernetes-create nodes add \
  --count 2 \
  --provider linode \
  --region us-east

# Adicionar a um pool específico
kubernetes-create nodes add \
  --count 1 \
  --pool do-workers
```

### Flags Disponíveis

| Flag | Descrição | Padrão |
|------|-----------|--------|
| `--count` | Número de nodes | 1 |
| `--role` | Role (master/worker) | worker |
| `--size` | Tamanho do node | Do config |
| `--provider` | Provider (digitalocean/linode) | Do config |
| `--region` | Região | Do config |
| `--pool` | Nome do pool | Auto-detect |

### Workflow Recomendado

```bash
# 1. Ver configuração atual
kubernetes-create nodes list

# 2. Editar cluster config para aumentar count
vim cluster.yaml
# nodePools:
#   - name: workers
#     count: 5  # Era 3, agora 5

# 3. Preview das mudanças
kubernetes-create deploy --config cluster.yaml --dry-run

# 4. Aplicar as mudanças
kubernetes-create deploy --config cluster.yaml

# 5. Verificar novos nodes
kubernetes-create nodes list
```

### ⚠️ Importante

- **Masters**: Sempre mantenha número ímpar (1, 3, 5) para HA
- **Workers**: Pode adicionar qualquer quantidade
- **Custos**: Cada node tem custo mensal no provider
- **Tempo**: Provisionamento leva ~5-10 minutos

---

## ➖ 3. Remover Nodes

### Remover um Worker

```bash
# Remover worker específico
kubernetes-create nodes remove worker-3
```

### Remover com Force (sem confirmação)

```bash
# Remover sem pedir confirmação
kubernetes-create nodes remove worker-3 --force
```

### Processo de Remoção

O comando executa:

1. **Drain** - Move todos os pods para outros nodes
2. **Delete** - Remove o node do Kubernetes
3. **Destroy** - Destroi o recurso no cloud provider

### Output Exemplo

```
➖ Removing Node: worker-3

⚠️  WARNING: This will:
  1. Drain all pods from the node
  2. Remove the node from Kubernetes
  3. Destroy the cloud resource

Are you sure you want to remove worker-3? (y/N): y

⏳ Draining node...
✅ Node drained successfully

⏳ Deleting from Kubernetes...
✅ Node deleted from cluster

⏳ Destroying cloud resource...
✅ Resource destroyed

✅ Node worker-3 removed successfully!
```

### Workflow Recomendado

```bash
# 1. Ver nodes atuais
kubernetes-create nodes list

# 2. Editar config para reduzir count
vim cluster.yaml
# nodePools:
#   - name: workers
#     count: 2  # Era 3, agora 2

# 3. Preview
kubernetes-create deploy --config cluster.yaml --dry-run

# 4. Aplicar
kubernetes-create deploy --config cluster.yaml

# 5. Verificar
kubernetes-create nodes list
```

### ⚠️  Avisos

- **Dados**: Certifique-se que não há dados importantes no node
- **Pods**: Pods com PersistentVolumes podem ter problemas
- **Masters**: NUNCA remova masters sem backup
- **HA**: Mantenha pelo menos 3 masters para HA

---

## 🔐 4. SSH nos Nodes

### SSH em um Node Específico

```bash
# SSH no master-1
kubernetes-create nodes ssh master-1

# SSH no worker-2
kubernetes-create nodes ssh worker-2
```

### O Que Acontece

1. CLI busca o IP do node
2. Recupera a chave SSH do stack
3. Abre sessão SSH interativa

### Output Exemplo

```
🔐 SSH to Node: master-1

⏳ Fetching node information...
✅ Node found: 167.71.1.1

🔑 Using SSH key from cluster deployment

Connecting to root@167.71.1.1...

root@master-1:~#
```

### Comandos Úteis no Node

```bash
# Ver status do RKE2
systemctl status rke2-server  # Master
systemctl status rke2-agent   # Worker

# Ver logs do RKE2
journalctl -u rke2-server -f
journalctl -u rke2-agent -f

# Ver containers rodando
crictl ps

# Ver uso de recursos
top
htop
df -h

# Ver configuração RKE2
cat /etc/rancher/rke2/config.yaml

# Ver kubeconfig (apenas master)
cat /etc/rancher/rke2/rke2.yaml
```

### SSH Manual (Alternativa)

Se o comando `nodes ssh` não funcionar:

```bash
# 1. Pegar IP do node
kubernetes-create nodes list

# 2. Pegar chave SSH (no primeiro deploy)
# A chave está em ~/.ssh/ ou no output do Pulumi

# 3. SSH manual
ssh -i ~/.ssh/cluster-key root@<node-ip>
```

---

## ⬆️ 5. Upgrade do Kubernetes

### Upgrade para Versão Específica

```bash
# Upgrade para versão específica
kubernetes-create nodes upgrade --version v1.29.0+rke2r1
```

### Upgrade para Latest Stable

```bash
# Upgrade para última versão estável
kubernetes-create nodes upgrade
```

### Preview do Upgrade (Dry-Run)

```bash
# Ver o que será feito sem executar
kubernetes-create nodes upgrade --version v1.29.0+rke2r1 --dry-run
```

### Processo de Upgrade

```
⬆️  Upgrade Kubernetes

📋 Upgrade Plan:
  • Current Version: v1.28.5+rke2r1
  • Target Version: v1.29.0+rke2r1
  • Upgrade Order:
    1. Master nodes (one by one)
    2. Worker nodes (rolling update)

⚠️  IMPORTANT:
  • Backup your cluster before upgrading
  • Test in staging environment first
  • Expect brief downtime during master upgrades

Do you want to proceed with the upgrade? (y/N): y

🔄 Upgrading master-1...
✅ master-1 upgraded successfully

🔄 Upgrading master-2...
✅ master-2 upgraded successfully

🔄 Upgrading master-3...
✅ master-3 upgraded successfully

🔄 Upgrading workers (rolling)...
✅ All workers upgraded successfully

✅ Cluster upgraded to v1.29.0+rke2r1!
```

### Workflow Recomendado

```bash
# 1. BACKUP PRIMEIRO!
kubernetes-create backup create

# 2. Testar em staging
kubernetes-create nodes upgrade \
  --version v1.29.0+rke2r1 \
  --stack staging

# 3. Verificar staging está OK
kubernetes-create nodes list --stack staging
kubectl get nodes --stack staging

# 4. Fazer upgrade em produção
kubernetes-create nodes upgrade \
  --version v1.29.0+rke2r1 \
  --stack production

# 5. Verificar produção
kubernetes-create nodes list
kubectl get nodes
```

### Workflow via Config (Alternativo)

```bash
# 1. Editar cluster.yaml
vim cluster.yaml
# kubernetes:
#   version: v1.29.0+rke2r1  # Atualizar versão

# 2. Preview
kubernetes-create deploy --config cluster.yaml --dry-run

# 3. Aplicar
kubernetes-create deploy --config cluster.yaml

# 4. Verificar
kubernetes-create nodes list
```

### ⚠️ Precauções

1. **Backup**: SEMPRE faça backup antes de upgrade
2. **Staging**: Teste em staging primeiro
3. **Documentação**: Leia release notes da nova versão
4. **Compatibilidade**: Verifique compatibilidade de addons
5. **Rollback Plan**: Tenha plano de rollback pronto
6. **Maintenance Window**: Faça em horário de baixo uso
7. **Monitoring**: Monitore durante e após upgrade

### Versões Suportadas

```bash
# RKE2 Stable
v1.28.x+rke2r1
v1.29.x+rke2r1
v1.30.x+rke2r1

# Verificar versões disponíveis
curl -s https://update.rke2.io/v1-release/channels | jq
```

---

## 📊 Casos de Uso Comuns

### 1. Escalar Workers para Atender Demanda

```bash
# Situação: Aplicação precisa de mais recursos

# Ver uso atual
kubectl top nodes

# Adicionar 2 workers
vim cluster.yaml  # Aumentar count de workers
kubernetes-create deploy --config cluster.yaml

# Verificar
kubernetes-create nodes list
kubectl get nodes
```

### 2. Remover Node Com Problema

```bash
# Situação: Node com problema de hardware

# Identificar node problemático
kubernetes-create nodes list
# worker-2 está com status NotReady

# Remover node problemático
kubernetes-create nodes remove worker-2

# Node será recriado automaticamente no próximo deploy
```

### 3. Maintenance de um Node

```bash
# Situação: Precisa fazer manutenção no node

# 1. SSH no node
kubernetes-create nodes ssh worker-1

# 2. Fazer manutenção
# ... comandos de manutenção ...

# 3. Sair
exit

# 4. Verificar se voltou ao normal
kubernetes-create nodes list
```

### 4. Investigar Problema

```bash
# Situação: Pods falhando em um node específico

# 1. SSH no node
kubernetes-create nodes ssh worker-1

# 2. Ver logs
journalctl -u rke2-agent -f

# 3. Ver containers
crictl ps
crictl logs <container-id>

# 4. Ver recursos
top
df -h

# 5. Se necessário, remover e recriar
exit
kubernetes-create nodes remove worker-1
kubernetes-create deploy --config cluster.yaml
```

### 5. Upgrade Seguro

```bash
# Situação: Nova versão do Kubernetes disponível

# 1. Backup
kubernetes-create backup create

# 2. Testar em staging
kubernetes-create nodes upgrade \
  --version v1.29.0+rke2r1 \
  --stack staging

# 3. Verificar staging
kubernetes-create nodes list --stack staging
# Rodar testes da aplicação

# 4. Se OK, fazer em produção
kubernetes-create nodes upgrade \
  --version v1.29.0+rke2r1 \
  --stack production

# 5. Monitorar
watch -n 5 'kubernetes-create nodes list'
```

---

## 🎯 Boas Práticas

### 1. Always Backup Before Changes

```bash
# Antes de qualquer operação destrutiva
kubernetes-create backup create
```

### 2. Use Dry-Run

```bash
# Sempre faça preview primeiro
kubernetes-create deploy --config cluster.yaml --dry-run
```

### 3. Maintain HA

```bash
# Masters sempre em número ímpar
# Mínimo 3 para HA
nodePools:
  - name: masters
    count: 3  # ✅ Ímpar
    # count: 4  # ❌ Par
```

### 4. Label Your Nodes

```yaml
# No cluster.yaml
nodePools:
  - name: workers
    labels:
      workload: web
      environment: production
```

### 5. Monitor After Changes

```bash
# Após adicionar/remover/upgrade
watch -n 5 'kubernetes-create nodes list'
kubectl get nodes -w
kubectl get pods --all-namespaces -w
```

---

## 🚨 Troubleshooting

### Node Não Aparece

```bash
# Verificar se foi criado no provider
# DigitalOcean: https://cloud.digitalocean.com/droplets
# Linode: https://cloud.linode.com/linodes

# Ver logs do Pulumi
kubernetes-create deploy --config cluster.yaml --verbose
```

### Node em NotReady

```bash
# SSH no node
kubernetes-create nodes ssh <node-name>

# Ver logs
journalctl -u rke2-server -f  # Master
journalctl -u rke2-agent -f   # Worker

# Ver se RKE2 está rodando
systemctl status rke2-server
systemctl status rke2-agent

# Reiniciar se necessário
systemctl restart rke2-server
systemctl restart rke2-agent
```

### SSH Não Funciona

```bash
# 1. Verificar IP
kubernetes-create nodes list

# 2. Testar conectividade
ping <node-ip>

# 3. Verificar firewall
# Permitir SSH (porta 22) no firewall do provider

# 4. SSH manual
ssh -i ~/.ssh/cluster-key root@<node-ip>
```

### Upgrade Falhou

```bash
# 1. Verificar status
kubernetes-create nodes list

# 2. Ver logs
kubernetes-create nodes ssh master-1
journalctl -u rke2-server -f

# 3. Rollback se necessário
# Editar cluster.yaml com versão anterior
kubernetes-create deploy --config cluster.yaml

# 4. Restaurar backup se crítico
kubernetes-create backup restore <backup-id>
```

---

## 📚 Resumo dos Comandos

```bash
# Listar
kubernetes-create nodes list
kubernetes-create nodes list --stack production

# Adicionar
kubernetes-create nodes add --count 2
kubernetes-create nodes add --count 3 --size s-4vcpu-8gb
kubernetes-create nodes add --provider linode --region us-east

# Remover
kubernetes-create nodes remove worker-3
kubernetes-create nodes remove worker-3 --force

# SSH
kubernetes-create nodes ssh master-1
kubernetes-create nodes ssh worker-2

# Upgrade
kubernetes-create nodes upgrade
kubernetes-create nodes upgrade --version v1.29.0+rke2r1
kubernetes-create nodes upgrade --dry-run
```

---

## 🎉 Conclusão

O gerenciamento de nodes é essencial para manter um cluster saudável e escalável. Use estes comandos para:

✅ **Monitorar** - Ver status de todos os nodes
✅ **Escalar** - Adicionar/remover conforme necessidade
✅ **Debugar** - SSH para investigar problemas
✅ **Atualizar** - Manter Kubernetes atualizado

**Sempre lembre:**
- 🔒 Backup antes de mudanças
- 👀 Dry-run para preview
- 📊 Monitor após operações
- 🧪 Teste em staging primeiro

Happy Node Management! 🚀
