# 💾 State Management - Como Funciona

## ✅ Sim, o Estado é Salvo Automaticamente!

O **Pulumi Automation API** (usado pela CLI) salva o estado automaticamente em `~/.pulumi/`. Você **não precisa se preocupar** com gerenciamento de estado - tudo é automático!

---

## 📂 Onde o Estado é Armazenado

### Local Backend (Padrão)

Por padrão, o estado é salvo localmente em:

```
~/.pulumi/
├── stacks/
│   └── kubernetes-create/
│       ├── production.json          # Estado do stack "production"
│       ├── production.json.bak      # Backup automático
│       ├── staging.json             # Estado do stack "staging"
│       └── staging.json.bak         # Backup automático
├── backups/                         # Backups históricos
├── history/                         # Histórico de operações
└── workspaces/                      # Workspaces locais
```

### Estrutura do Estado

Cada stack tem seu próprio arquivo JSON que contém:
- **Recursos criados** (droplets, DNS, chaves SSH, etc.)
- **Outputs** (kubeconfig, IPs, endpoints)
- **Configurações secretas** (tokens criptografados)
- **Dependências** entre recursos
- **Metadata** (timestamps, versões)

---

## 🔄 Como Funciona na Prática

### 1. Deploy (Criar Cluster)

```bash
kubernetes-create deploy --config cluster.yaml --stack production
```

**O que acontece:**
1. ✅ CLI lê o estado existente de `~/.pulumi/stacks/kubernetes-create/production.json`
2. ✅ Compara com a configuração desejada
3. ✅ Cria/atualiza apenas o que mudou
4. ✅ Salva novo estado automaticamente
5. ✅ Cria backup do estado anterior

### 2. Status (Ver Estado)

```bash
kubernetes-create status --stack production
```

**O que acontece:**
1. ✅ CLI lê o estado de `~/.pulumi/stacks/kubernetes-create/production.json`
2. ✅ Mostra informações dos recursos criados
3. ✅ Exibe outputs (IPs, kubeconfig, etc.)

### 3. Destroy (Destruir Cluster)

```bash
kubernetes-create destroy --stack production
```

**O que acontece:**
1. ✅ CLI lê o estado para saber o que precisa destruir
2. ✅ Destrói recursos na ordem correta (respeitando dependências)
3. ✅ Remove o estado após destruição completa
4. ✅ Mantém backup do estado anterior

### 4. Update (Atualizar Cluster)

```bash
# Edite o cluster.yaml (ex: adicione mais workers)
kubernetes-create deploy --config cluster.yaml --stack production
```

**O que acontece:**
1. ✅ CLI compara estado atual com nova configuração
2. ✅ Identifica diferenças (diff)
3. ✅ Aplica apenas as mudanças necessárias
4. ✅ Salva estado atualizado

---

## 🌐 Backends Remotos (Opcional)

Para ambientes de equipe ou CI/CD, você pode usar backends remotos:

### Pulumi Cloud (Gratuito até 3 membros)

```bash
# Login no Pulumi Cloud
pulumi login

# Deploy (estado vai para Pulumi Cloud)
kubernetes-create deploy --config cluster.yaml
```

**Vantagens:**
- ✅ Estado centralizado e seguro
- ✅ Colaboração em equipe
- ✅ Histórico completo de mudanças
- ✅ Secrets criptografados
- ✅ Interface web para visualizar recursos

### S3 Backend (AWS)

```bash
# Login no S3
pulumi login s3://my-pulumi-state-bucket

# Deploy (estado vai para S3)
kubernetes-create deploy --config cluster.yaml
```

### Azure Blob Storage

```bash
pulumi login azblob://my-container
```

### Google Cloud Storage

```bash
pulumi login gs://my-pulumi-state-bucket
```

---

## 🔒 Segurança do Estado

### Secrets Criptografados

Todos os valores sensíveis são **criptografados** no estado:

```json
{
  "digitaloceanToken": {
    "secret": "[ciphertext]v1:xxxxx..."
  },
  "linodeToken": {
    "secret": "[ciphertext]v1:yyyyy..."
  }
}
```

### Chave de Criptografia

Por padrão, usa uma chave baseada em senha. Para mais segurança:

```bash
# Usar chave específica
export PULUMI_CONFIG_PASSPHRASE="my-super-secure-passphrase"

# Ou usar AWS KMS
pulumi stack init production --secrets-provider="awskms://alias/pulumi-secrets"

# Ou usar Google Cloud KMS
pulumi stack init production --secrets-provider="gcpkms://projects/my-project/locations/us-central1/keyRings/pulumi/cryptoKeys/pulumi"
```

---

## 📊 Múltiplos Ambientes (Stacks)

Você pode ter múltiplos stacks (ambientes) com estados independentes:

```bash
# Criar stack de staging
kubernetes-create deploy --config staging.yaml --stack staging

# Criar stack de production
kubernetes-create deploy --config production.yaml --stack production

# Criar stack de development
kubernetes-create deploy --config dev.yaml --stack dev
```

Cada stack tem seu próprio estado:
```
~/.pulumi/stacks/kubernetes-create/
├── staging.json
├── production.json
└── dev.json
```

---

## 🔍 Inspecionar o Estado

### Ver estado bruto (JSON)

```bash
cat ~/.pulumi/stacks/kubernetes-create/production.json | jq
```

### Ver recursos via Pulumi CLI (se instalado)

```bash
cd ~/.projects/do-droplet-create
pulumi stack ls                    # Listar stacks
pulumi stack select production     # Selecionar stack
pulumi stack export                # Exportar estado (JSON)
pulumi stack graph                 # Gerar grafo de dependências
```

### Ver outputs

```bash
kubernetes-create status --stack production
```

---

## 🔄 Backup e Restore

### Backups Automáticos

A cada deploy, Pulumi cria backup automático:

```
~/.pulumi/stacks/kubernetes-create/
├── production.json          # Estado atual
├── production.json.bak      # Backup imediato anterior
```

Backups históricos em:
```
~/.pulumi/backups/kubernetes-create/production/
├── 2025-10-20-10-30-00.json
├── 2025-10-20-09-15-00.json
└── 2025-10-19-16-45-00.json
```

### Restore Manual

```bash
# Copiar backup para restaurar
cp ~/.pulumi/stacks/kubernetes-create/production.json.bak \
   ~/.pulumi/stacks/kubernetes-create/production.json

# Ou restaurar de backup histórico
cp ~/.pulumi/backups/kubernetes-create/production/2025-10-20-09-15-00.json \
   ~/.pulumi/stacks/kubernetes-create/production.json
```

### Export/Import de Estado

```bash
# Exportar estado (backup manual)
pulumi stack export --file backup-production.json

# Importar estado (restore)
pulumi stack import --file backup-production.json
```

---

## 🚨 Problemas Comuns

### Estado Corrompido

Se o estado ficar corrompido:

```bash
# Restaurar do backup
cp ~/.pulumi/stacks/kubernetes-create/production.json.bak \
   ~/.pulumi/stacks/kubernetes-create/production.json

# Tentar refresh
kubernetes-create deploy --config cluster.yaml --refresh
```

### Estado Dessincronizado

Se recursos foram modificados fora do Pulumi:

```bash
# Refresh para sincronizar estado com realidade
pulumi refresh --stack production

# Ou forçar re-deploy
kubernetes-create deploy --config cluster.yaml --yes
```

### Perda de Estado

Se você perder o estado local:

1. **Com backup**: Restaurar do backup
2. **Sem backup**: Você pode:
   - Importar recursos manualmente (avançado)
   - Destruir recursos manualmente e recriar
   - Usar `pulumi import` para importar recursos existentes

---

## 💡 Boas Práticas

### 1. Use Backend Remoto em Produção

```bash
# Em produção, sempre use backend remoto
pulumi login
# ou
pulumi login s3://production-state-bucket
```

### 2. Backups Regulares

```bash
# Script de backup diário
#!/bin/bash
DATE=$(date +%Y-%m-%d)
mkdir -p ~/backups/kubernetes-create
pulumi stack export --file ~/backups/kubernetes-create/production-$DATE.json
```

### 3. Versionamento do Estado

Se usar Git (cuidado com secrets!):

```bash
# .gitignore
*.json         # Não commitear estado
*.json.bak

# Mas você pode versionar configs
cluster.yaml   # OK para versionar
```

### 4. Proteção do Estado em Produção

```bash
# Permissões restritas
chmod 600 ~/.pulumi/stacks/kubernetes-create/production.json

# Criptografia
export PULUMI_CONFIG_PASSPHRASE="strong-passphrase"
```

### 5. CI/CD com Estado Remoto

```yaml
# .github/workflows/deploy.yml
- name: Deploy Cluster
  env:
    PULUMI_ACCESS_TOKEN: ${{ secrets.PULUMI_ACCESS_TOKEN }}
  run: |
    pulumi login
    kubernetes-create deploy --config cluster.yaml --yes
```

---

## 📚 Resumo

| Aspecto | Detalhes |
|---------|----------|
| **Armazenamento padrão** | `~/.pulumi/stacks/kubernetes-create/<stack>.json` |
| **Backup automático** | ✅ Sim, a cada deploy |
| **Criptografia** | ✅ Sim, secrets criptografados |
| **Múltiplos ambientes** | ✅ Via stacks (`--stack production`) |
| **Backends remotos** | ✅ Pulumi Cloud, S3, Azure Blob, GCS |
| **Sincronização** | ✅ Automática via Pulumi |
| **Restore** | ✅ Via backups automáticos ou `pulumi import` |

---

## 🎯 Comandos Úteis

```bash
# Ver stacks disponíveis
ls ~/.pulumi/stacks/kubernetes-create/

# Ver tamanho do estado
du -h ~/.pulumi/stacks/kubernetes-create/production.json

# Ver outputs
kubernetes-create status --stack production

# Backup manual
cp ~/.pulumi/stacks/kubernetes-create/production.json \
   ~/production-backup-$(date +%Y%m%d).json

# Listar recursos no estado (com Pulumi CLI)
pulumi stack --show-urns
```

---

## 🚀 Conclusão

**Sim, o estado é salvo automaticamente!** Você não precisa se preocupar com isso. O Pulumi cuida de:

✅ Salvar estado após cada operação
✅ Criar backups automáticos
✅ Criptografar secrets
✅ Gerenciar dependências
✅ Sincronizar com recursos reais

Você só precisa usar a CLI normalmente:

```bash
# Deploy
kubernetes-create deploy --config cluster.yaml

# Estado salvo automaticamente! ✅

# Ver estado
kubernetes-create status

# Atualizar
kubernetes-create deploy --config cluster-updated.yaml

# Estado atualizado automaticamente! ✅

# Destruir
kubernetes-create destroy

# Estado removido automaticamente! ✅
```

**É simples assim!** 🎉
