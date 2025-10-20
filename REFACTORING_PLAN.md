# Plano de Refatoração do Projeto Kubernetes Multi-Cloud

## Análise Atual

### Problemas Identificados

#### 1. Arquivos Duplicados com Sufixos
O projeto contém múltiplas versões de componentes com sufixos confusos:
- **`*_real.go`**: Implementações reais (9 arquivos)
- **`*_granular.go`**: Versões granulares (7 arquivos)
- **`*_stub.go`**: Stubs/mocks (1 arquivo)
- **`*.bak`**: Backups desnecessários (4 arquivos)

#### 2. Nomenclatura Inconsistente
- Arquivos: `wireguard_real.go`, `wireguard_mesh_simple.go`, `wireguard_setup_real.go`
- Função `loadConfigFromPulumi` vs método de estrutura
- Componentes com nomes genéricos: `component`, `orchestrator`, `manager`

#### 3. Código Duplicado
- Lógica de SSH repetida em múltiplos componentes
- Validação de nós duplicada
- Configuração de WireGuard em 3 arquivos diferentes
- Lógica de DNS em 2 implementações

---

## Plano de Refatoração

### Fase 1: Limpeza de Arquivos

#### Remover Arquivos Obsoletos
```bash
# Arquivos .bak
rm internal/orchestrator/*.bak

# Arquivos Python de correção temporária
rm fix_*.py

# Arquivos duplicados/desatualizados
rm main.go.new
```

#### Consolidar Implementações
Manter apenas uma versão de cada componente:

**Manter:**
- `simple_real_orchestrator.go` → Renomear para `cluster_orchestrator.go`
- `node_deployment_real.go` → Renomear para `node_deployment.go`
- `wireguard_mesh_simple.go` → Renomear para `wireguard_mesh.go`
- `rke2_real.go` → Renomear para `rke2_installer.go`
- `dns_real.go` → Renomear para `dns_manager.go`
- `node_provisioning_real.go` → Renomear para `node_provisioner.go`

**Remover (migrar código útil antes):**
- `orchestrator.go` (versão antiga complexa)
- `wireguard_real.go`
- `wireguard_setup_real.go`
- `dns_component_real.go`
- `rke_install_real.go`
- `node_deployment_component.go`
- Todos os `*_granular.go`
- `component_stubs.go`

---

### Fase 2: Reorganização da Estrutura

#### Estrutura de Diretórios Proposta

```
internal/
├── orchestrator/
│   ├── cluster_orchestrator.go      # Orquestrador principal
│   ├── ssh_keys.go                  # Geração de chaves SSH
│   └── components/                  # Componentes do cluster
│       ├── node_deployment.go       # Deploy de nós
│       ├── node_provisioner.go      # Provisionamento (Docker, etc)
│       ├── wireguard_mesh.go        # Configuração VPN mesh
│       ├── rke2_installer.go        # Instalação RKE2
│       └── dns_manager.go           # Gerenciamento DNS
│
├── validation/                      # Validações centralizadas
│   ├── config_validator.go         # Validação de configuração
│   └── node_validator.go           # Validação de distribuição de nós
│
└── common/                          # Código compartilhado
    ├── ssh_executor.go              # Execução SSH reutilizável
    ├── remote_commands.go           # Comandos remotos comuns
    └── retry_logic.go               # Lógica de retry

pkg/
├── config/
│   ├── loader.go                    # Carregamento de config
│   ├── pulumi_loader.go            # Específico do Pulumi
│   └── types.go                     # Tipos de configuração
│
├── providers/                       # Providers cloud
│   ├── interface.go
│   ├── digitalocean.go
│   └── linode.go
│
└── [manter estrutura existente dos demais packages]
```

---

### Fase 3: Melhorias de Nomenclatura

#### Renomeação de Arquivos

| Arquivo Atual | Novo Nome | Justificativa |
|---------------|-----------|---------------|
| `simple_real_orchestrator.go` | `cluster_orchestrator.go` | Mais descritivo, remove "simple/real" |
| `node_deployment_real.go` | `node_deployment.go` | Remove sufixo redundante |
| `node_provisioning_real.go` | `node_provisioner.go` | Mais conciso |
| `wireguard_mesh_simple.go` | `wireguard_mesh.go` | Remove "simple" |
| `rke2_real.go` | `rke2_installer.go` | Mais descritivo |
| `dns_real.go` | `dns_manager.go` | Mais descritivo |

#### Renomeação de Funções/Componentes

| Função/Tipo Atual | Novo Nome | Justificativa |
|-------------------|-----------|---------------|
| `SimpleRealOrchestratorComponent` | `ClusterOrchestrator` | Mais simples e claro |
| `NewSimpleRealOrchestratorComponent` | `NewClusterOrchestrator` | Segue novo nome |
| `RealNodeDeploymentComponent` | `NodeDeployment` | Remove "Real" |
| `NewRealNodeDeploymentComponent` | `NewNodeDeployment` | Consistente |
| `WireGuardMeshComponent` | `WireGuardMesh` | Remove "Component" |
| `RKE2RealComponent` | `RKE2Installer` | Mais descritivo |
| `loadConfigFromPulumi` | `LoadPulumiConfig` | Exportável, segue padrão Go |
| `validateConfig` | `ValidateClusterConfig` | Mais específico |

---

### Fase 4: Extração de Código Duplicado

#### 1. Utilitários SSH Comuns

**Criar:** `internal/common/ssh_executor.go`

```go
package common

import (
    "github.com/pulumi/pulumi-command/sdk/go/command/remote"
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// SSHExecutor centraliza execução de comandos SSH
type SSHExecutor struct {
    ctx           *pulumi.Context
    privateKey    pulumi.StringOutput
}

func NewSSHExecutor(ctx *pulumi.Context, privateKey pulumi.StringOutput) *SSHExecutor {
    return &SSHExecutor{
        ctx:        ctx,
        privateKey: privateKey,
    }
}

func (e *SSHExecutor) Execute(name string, host pulumi.StringOutput, command string, opts ...pulumi.ResourceOption) (*remote.Command, error) {
    return remote.NewCommand(e.ctx, name, &remote.CommandArgs{
        Connection: remote.ConnectionArgs{
            Host:       host,
            User:       pulumi.String("root"),
            PrivateKey: e.privateKey,
        },
        Create: pulumi.String(command),
    }, opts...)
}
```

#### 2. Validação de Nós Centralizada

**Criar:** `internal/validation/node_validator.go`

```go
package validation

import (
    "fmt"
    "kubernetes-create/pkg/config"
)

type NodeDistribution struct {
    Total   int
    Masters int
    Workers int
    ByProvider map[string]int
}

func ValidateNodeDistribution(cfg *config.ClusterConfig) error {
    dist := CalculateDistribution(cfg)

    if dist.Total != 6 {
        return fmt.Errorf("expected 6 nodes, got %d", dist.Total)
    }

    if dist.Masters != 3 {
        return fmt.Errorf("expected 3 masters, got %d", dist.Masters)
    }

    if dist.Workers != 3 {
        return fmt.Errorf("expected 3 workers, got %d", dist.Workers)
    }

    return nil
}

func CalculateDistribution(cfg *config.ClusterConfig) NodeDistribution {
    // Implementação compartilhada
    // (código extraído de main.go e orchestrator.go)
}
```

#### 3. Comandos Remotos Comuns

**Criar:** `internal/common/remote_commands.go`

```go
package common

// Comandos SSH reutilizáveis
const (
    InstallDocker = `
        apt-get update && apt-get install -y docker.io
        systemctl enable docker
        systemctl start docker
    `

    InstallWireGuard = `
        apt-get update && apt-get install -y wireguard
    `

    GenerateWireGuardKeys = `
        wg genkey | tee /etc/wireguard/private.key | wg pubkey > /etc/wireguard/public.key
        cat /etc/wireguard/public.key
    `
)

func BuildAptRetryCommand(packages ...string) string {
    // Lógica de retry do apt-get
    // (extraída de node_provisioning_real.go)
}
```

---

### Fase 5: Melhorias no main.go

#### Separar Responsabilidades

**Criar:** `pkg/config/pulumi_loader.go`

```go
package config

import (
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// LoadPulumiConfig carrega configuração do Pulumi
func LoadPulumiConfig(ctx *pulumi.Context) (*ClusterConfig, error) {
    conf := config.New(ctx, "")

    // ... implementação atual de loadConfigFromPulumi
}
```

**Criar:** `internal/validation/config_validator.go`

```go
package validation

import "kubernetes-create/pkg/config"

// ValidateClusterConfig valida configuração do cluster
func ValidateClusterConfig(cfg *config.ClusterConfig) error {
    if err := ValidateNodeDistribution(cfg); err != nil {
        return err
    }

    if err := ValidateWireGuardConfig(cfg); err != nil {
        return err
    }

    if err := ValidateProviders(cfg); err != nil {
        return err
    }

    return nil
}

func ValidateWireGuardConfig(cfg *config.ClusterConfig) error {
    // Implementação extraída de main.go
}

func ValidateProviders(cfg *config.ClusterConfig) error {
    // Implementação extraída de main.go
}
```

**Novo main.go simplificado:**

```go
package main

import (
    "fmt"
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
    "kubernetes-create/internal/orchestrator"
    "kubernetes-create/internal/validation"
    "kubernetes-create/pkg/config"
)

func main() {
    pulumi.Run(func(ctx *pulumi.Context) error {
        ctx.Log.Info("Starting Kubernetes cluster deployment", nil)

        // Carregar configuração
        cfg, err := config.LoadPulumiConfig(ctx)
        if err != nil {
            return fmt.Errorf("failed to load configuration: %w", err)
        }

        // Validar configuração
        if err := validation.ValidateClusterConfig(cfg); err != nil {
            return fmt.Errorf("configuration validation failed: %w", err)
        }

        // Criar orquestrador
        clusterOrch, err := orchestrator.NewClusterOrchestrator(ctx, "kubernetes-cluster", cfg)
        if err != nil {
            return fmt.Errorf("failed to create orchestrator: %w", err)
        }

        // Exportar outputs
        exportClusterOutputs(ctx, clusterOrch, cfg)

        return nil
    })
}

func exportClusterOutputs(ctx *pulumi.Context, orch *orchestrator.ClusterOrchestrator, cfg *config.ClusterConfig) {
    ctx.Export("status", orch.Status)
    ctx.Export("clusterName", orch.ClusterName)
    ctx.Export("kubeConfig", orch.KubeConfig)
    ctx.Export("sshPrivateKey", orch.SSHPrivateKey)
    ctx.Export("sshPublicKey", orch.SSHPublicKey)
    ctx.Export("apiEndpoint", orch.APIEndpoint)

    ctx.Export("connectionInstructions", pulumi.String(fmt.Sprintf(`
=== KUBERNETES CLUSTER DEPLOYED ===

Cluster: %s
API Endpoint: https://api.chalkan3.com.br:6443

1. Save kubeconfig:
   pulumi stack output kubeConfig --show-secrets > ~/.kube/config

2. Test cluster:
   kubectl get nodes

3. SSH to nodes:
   pulumi stack output sshPrivateKey --show-secrets > ~/.ssh/k8s.pem
   chmod 600 ~/.ssh/k8s.pem
   ssh -i ~/.ssh/k8s.pem root@<node-ip>
`, cfg.Metadata.Name)))
}
```

---

## Checklist de Execução

### Fase 1: Preparação
- [ ] Criar backup do projeto
- [ ] Documentar decisões de refatoração
- [ ] Revisar código para identificar dependências

### Fase 2: Limpeza
- [ ] Remover arquivos .bak
- [ ] Remover scripts Python temporários
- [ ] Remover arquivos duplicados não utilizados

### Fase 3: Consolidação
- [ ] Criar estrutura de diretórios nova
- [ ] Consolidar implementações WireGuard
- [ ] Consolidar implementações DNS
- [ ] Consolidar implementações RKE2
- [ ] Atualizar imports em todos os arquivos

### Fase 4: Extração de Código Comum
- [ ] Criar SSHExecutor
- [ ] Criar RemoteCommands
- [ ] Criar NodeValidator
- [ ] Criar ConfigValidator
- [ ] Refatorar componentes para usar código comum

### Fase 5: Renomeação
- [ ] Renomear arquivos conforme tabela
- [ ] Renomear tipos/structs
- [ ] Renomear funções
- [ ] Atualizar todos os imports
- [ ] Atualizar comentários e documentação

### Fase 6: Testes
- [ ] Verificar compilação: `go build`
- [ ] Executar testes se existirem
- [ ] Validar com `pulumi preview`
- [ ] Testar deploy em ambiente de teste

### Fase 7: Documentação
- [ ] Atualizar README.md
- [ ] Atualizar documentação em docs/
- [ ] Adicionar comentários GoDoc
- [ ] Atualizar diagramas se necessário

---

## Benefícios Esperados

### Manutenibilidade
- ✅ Código mais fácil de entender
- ✅ Estrutura clara e organizada
- ✅ Menos duplicação = menos bugs

### Legibilidade
- ✅ Nomes descritivos e consistentes
- ✅ Hierarquia lógica de arquivos
- ✅ Separação clara de responsabilidades

### Extensibilidade
- ✅ Fácil adicionar novos providers
- ✅ Componentes reutilizáveis
- ✅ Código comum bem organizado

### Performance de Desenvolvimento
- ✅ Menos tempo procurando código
- ✅ Mais rápido adicionar features
- ✅ Menos código para manter

---

## Considerações

### Compatibilidade
- Manter compatibilidade com Pulumi state
- Não alterar nomes de recursos do Pulumi
- Manter outputs existentes

### Migração Gradual
- Pode ser feito em etapas
- Testar após cada fase
- Manter git commits organizados

### Riscos
- ⚠️ Possíveis imports quebrados temporariamente
- ⚠️ Necessário teste completo após refatoração
- ⚠️ Revisar cuidadosamente antes de deploy em produção

---

## Comandos Úteis

```bash
# Verificar compilação
go build ./...

# Verificar imports não utilizados
go mod tidy

# Formatar código
go fmt ./...

# Verificar erros
go vet ./...

# Preview Pulumi (sem aplicar mudanças)
pulumi preview

# Testar build do binário
go build -o bin/kubernetes-create main.go
```

---

## Próximos Passos

1. Revisar este plano com o time
2. Priorizar fases mais críticas
3. Executar refatoração em branch separada
4. Code review detalhado
5. Testar em ambiente de desenvolvimento
6. Merge para main após validação completa
