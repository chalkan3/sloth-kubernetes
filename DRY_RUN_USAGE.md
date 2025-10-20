# 🔍 Dry-Run Mode - Preview Changes Before Deployment

## O Que é Dry-Run?

O modo **dry-run** (`--dry-run`) permite **visualizar** todas as mudanças que serão aplicadas **sem realmente executá-las**. É como um "preview" ou "plano de execução" do Terraform.

---

## 🎯 Quando Usar Dry-Run

### ✅ Use dry-run quando:

1. **Primeira implantação** - Ver o que será criado antes de gastar dinheiro
2. **Mudanças na configuração** - Validar alterações antes de aplicar
3. **Produção** - Sempre fazer preview antes de mudar produção
4. **Aprender** - Entender o que a ferramenta faz sem riscos
5. **Debugging** - Identificar problemas antes de executar
6. **Aprovações** - Mostrar para a equipe antes de aplicar

### ⚠️ Não é necessário para:

- Comandos de leitura (`status`, `kubeconfig`)
- Quando você tem certeza absoluta
- Ambientes de desenvolvimento descartáveis

---

## 🚀 Como Usar

### Sintaxe Básica

```bash
kubernetes-create deploy --config cluster.yaml --dry-run
```

### Com Flags Adicionais

```bash
# Dry-run com tokens diferentes
kubernetes-create deploy \
  --config cluster.yaml \
  --do-token test-token \
  --dry-run

# Dry-run com stack específico
kubernetes-create deploy \
  --config production.yaml \
  --stack production \
  --dry-run
```

### Workflow Recomendado

```bash
# 1. Fazer dry-run primeiro
kubernetes-create deploy --config cluster.yaml --dry-run

# 2. Revisar output

# 3. Se estiver OK, aplicar de verdade
kubernetes-create deploy --config cluster.yaml

# Ou em uma linha com confirmação
kubernetes-create deploy --config cluster.yaml --yes
```

---

## 📋 O Que o Dry-Run Mostra

### Output Exemplo

```
📄 Generating Configuration File

✓ Configuration loaded
✓ Configuration validated

📋 Deployment Summary:
  • Cluster Name: production
  • Providers: DigitalOcean + Linode
  • Total Nodes: 6 (3 masters + 3 workers)
  • Kubernetes: RKE2
  • Network: WireGuard VPN Mesh

🔧 Setting up Pulumi stack...
✓ Pulumi stack configured

🔄 Refreshing stack state...

📋 Previewing changes (dry-run mode)...

📋 Preview Summary (Dry-Run Mode)

Resources to be created: 25
  → New resources will be provisioned

Resources to be updated: 0

Resources to be deleted: 0

Resources unchanged: 0

💡 What will happen when you run without --dry-run:

  1. SSH keys will be generated
  2. Droplets/Linodes will be created across providers
  3. WireGuard VPN mesh will be configured
  4. RKE2 Kubernetes will be installed and configured
  5. DNS records will be created
  6. Kubeconfig will be generated and available

⚠️  This was a DRY-RUN. No actual changes were made.

To apply these changes, run without --dry-run flag:
  kubernetes-create deploy --config cluster.yaml
```

### Informações Mostradas

O dry-run exibe:

✅ **Configuração** - Resumo do que será deployado
✅ **Recursos a criar** - Número de novos recursos
✅ **Recursos a atualizar** - Mudanças em recursos existentes
✅ **Recursos a deletar** - O que será destruído
✅ **Recursos inalterados** - O que permanece igual
✅ **Próximos passos** - O que acontecerá no deploy real

---

## 🔄 Cenários de Uso

### 1. Nova Implantação (Create)

```bash
# Primeira vez - tudo será criado
kubernetes-create deploy --config new-cluster.yaml --dry-run
```

**Output:**
```
Resources to be created: 25
Resources to be updated: 0
Resources to be deleted: 0
```

### 2. Atualização (Update)

```bash
# Mudou algo no cluster.yaml e quer ver o impacto
kubernetes-create deploy --config cluster-updated.yaml --dry-run
```

**Output:**
```
Resources to be created: 3    # Novos nodes adicionados
Resources to be updated: 5    # Configurações alteradas
Resources to be deleted: 1    # Algo removido
```

### 3. Nenhuma Mudança (No-Op)

```bash
# Deploy novamente sem mudar nada
kubernetes-create deploy --config cluster.yaml --dry-run
```

**Output:**
```
Resources to be created: 0
Resources to be updated: 0
Resources to be deleted: 0
Resources unchanged: 25
```

### 4. Mudanças Destrutivas

```bash
# Reduzir número de workers de 3 para 2
kubernetes-create deploy --config cluster-reduced.yaml --dry-run
```

**Output:**
```
Resources to be created: 0
Resources to be updated: 2    # Configurações ajustadas
Resources to be deleted: 3    # 1 worker + recursos relacionados

⚠️  Some resources will be DESTROYED!
```

---

## 💰 Economia de Custos

### Evitar Gastos Desnecessários

```bash
# ERRADO: Deploy direto sem verificar
kubernetes-create deploy --config huge-cluster.yaml  # 💸 $$$

# CERTO: Dry-run primeiro
kubernetes-create deploy --config huge-cluster.yaml --dry-run

# Output: "25 droplets x $40/month = $1000/month"
# Você: "Ops, era pra ser 5 droplets!" 😅
```

### Validar Configuração

```bash
# Dry-run detecta erros antes de gastar
kubernetes-create deploy --config typo-cluster.yaml --dry-run

# Output: Error: Invalid node size "s-2vcpu-4gb-TYPO"
# Você corrige antes de criar recursos reais
```

---

## 🔐 Segurança

### Validar Mudanças em Produção

```bash
# Em produção, SEMPRE faça dry-run primeiro
kubernetes-create deploy \
  --config production.yaml \
  --stack production \
  --dry-run > changes-review.txt

# Envie changes-review.txt para equipe revisar

# Após aprovação:
kubernetes-create deploy \
  --config production.yaml \
  --stack production \
  --yes
```

### Pipeline CI/CD com Dry-Run

```yaml
# .github/workflows/deploy.yml
name: Deploy Cluster

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  preview:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Preview Changes (Dry-Run)
        run: |
          kubernetes-create deploy \
            --config cluster.yaml \
            --dry-run \
            > preview.txt

      - name: Comment PR with Preview
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const preview = fs.readFileSync('preview.txt', 'utf8');
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `## 🔍 Deployment Preview\n\n\`\`\`\n${preview}\n\`\`\``
            });

  deploy:
    if: github.event_name == 'push'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Deploy Cluster
        run: |
          kubernetes-create deploy \
            --config cluster.yaml \
            --yes
```

---

## 📊 Diferença: Dry-Run vs Deploy Real

| Aspecto | Dry-Run | Deploy Real |
|---------|---------|-------------|
| **Cria recursos** | ❌ Não | ✅ Sim |
| **Gasta dinheiro** | ❌ Não | ✅ Sim |
| **Modifica estado** | ❌ Não | ✅ Sim |
| **Mostra preview** | ✅ Sim | ❌ Não (só faz) |
| **Validação** | ✅ Sim | ✅ Sim |
| **Tempo** | ⚡ Rápido (segundos) | 🐢 Lento (minutos) |
| **Reversível** | ✅ Sim (nada foi feito) | ⚠️ Requer destroy |
| **Seguro** | ✅ 100% seguro | ⚠️ Cria recursos reais |

---

## 🎓 Boas Práticas

### 1. Sempre Dry-Run em Produção

```bash
# SEMPRE faça isso antes de tocar produção
kubernetes-create deploy \
  --config production.yaml \
  --stack production \
  --dry-run
```

### 2. Salvar Preview para Documentação

```bash
# Salvar output do dry-run
kubernetes-create deploy --config cluster.yaml --dry-run \
  > deployment-plan-$(date +%Y%m%d).txt

# Commitar no Git (sem secrets!)
git add deployment-plan-*.txt
git commit -m "Add deployment plan for review"
```

### 3. Automação com Dry-Run

```bash
#!/bin/bash
# Script inteligente que sempre faz dry-run primeiro

echo "🔍 Running dry-run first..."
kubernetes-create deploy --config $1 --dry-run

echo ""
read -p "Apply these changes? (y/N): " confirm

if [ "$confirm" = "y" ]; then
    echo "🚀 Deploying..."
    kubernetes-create deploy --config $1 --yes
else
    echo "❌ Deployment cancelled"
fi
```

### 4. Code Review com Dry-Run

```bash
# Pull Request com preview
git checkout feature/add-workers
kubernetes-create deploy --config cluster.yaml --dry-run

# Colar output no PR para revisão
# Time revisa ANTES de mergear
```

---

## 🔄 Fluxo Recomendado

### Workflow Completo

```bash
# 1. Editar configuração
vim cluster.yaml

# 2. Validar sintaxe (opcional)
yamllint cluster.yaml

# 3. DRY-RUN - Ver o que será feito
kubernetes-create deploy --config cluster.yaml --dry-run

# 4. Revisar output cuidadosamente
#    - Número de recursos
#    - Tipos de mudanças
#    - Recursos que serão destruídos

# 5. Se tudo OK, aplicar
kubernetes-create deploy --config cluster.yaml

# 6. Monitorar deployment
watch -n 5 'kubernetes-create status'

# 7. Verificar resultado
kubernetes-create kubeconfig -o ~/.kube/config
kubectl get nodes
```

---

## 🚨 Limitações do Dry-Run

### O Que Dry-Run NÃO Faz

❌ **Não valida credenciais** - Só valida sintaxe
❌ **Não testa conectividade** - Não faz chamadas de API reais
❌ **Não verifica quotas** - Pode falhar no deploy real por falta de quota
❌ **Não detecta conflitos** - Ex: IP já em uso
❌ **Não testa DNS** - Não valida se domínio está configurado

### Erros que Só Aparecem no Deploy Real

- **Quota excedida**: "You've reached your droplet limit"
- **Recurso em uso**: "IP address already allocated"
- **Permissões**: "Insufficient permissions to create resource"
- **Conflitos**: "Resource name already exists"
- **Rede**: "Unable to connect to API"

### Recomendação

```bash
# Dry-run mostra O QUE, mas não GARANTE sucesso
kubernetes-create deploy --config cluster.yaml --dry-run  # Preview

# Sempre monitore o deploy real
kubernetes-create deploy --config cluster.yaml            # Aplicar
```

---

## 💡 Dicas Pro

### 1. Combinar com Verbose

```bash
# Ver mais detalhes durante dry-run
kubernetes-create deploy \
  --config cluster.yaml \
  --dry-run \
  --verbose
```

### 2. Different Stacks

```bash
# Preview para staging
kubernetes-create deploy \
  --config staging.yaml \
  --stack staging \
  --dry-run

# Preview para production
kubernetes-create deploy \
  --config production.yaml \
  --stack production \
  --dry-run
```

### 3. Comparar Mudanças

```bash
# Dry-run antes
kubernetes-create deploy --config cluster.yaml --dry-run \
  > before.txt

# Editar cluster.yaml

# Dry-run depois
kubernetes-create deploy --config cluster.yaml --dry-run \
  > after.txt

# Comparar
diff before.txt after.txt
```

---

## 📚 Resumo

| Comando | O Que Faz |
|---------|-----------|
| `kubernetes-create deploy --dry-run` | Preview completo |
| `kubernetes-create deploy` | Deploy real |
| `kubernetes-create deploy --yes` | Deploy sem confirmação |
| `kubernetes-create deploy --dry-run --verbose` | Preview detalhado |

**Regra de Ouro:**

> 🔍 **Sempre faça dry-run em produção antes de deploy real!**

```bash
# ✅ CERTO
kubernetes-create deploy --config prod.yaml --dry-run  # Revisar
kubernetes-create deploy --config prod.yaml             # Aplicar

# ❌ ERRADO
kubernetes-create deploy --config prod.yaml --yes      # YOLO! 💥
```

---

## 🎯 Conclusão

O **dry-run** é sua ferramenta de segurança para:

✅ **Evitar surpresas** - Ver antes de fazer
✅ **Economizar dinheiro** - Não criar recursos por engano
✅ **Validar mudanças** - Garantir que está fazendo o que quer
✅ **Documentar** - Registrar o que será mudado
✅ **Aprovar** - Mostrar para equipe antes de aplicar

**Use sempre em produção!** 🛡️
