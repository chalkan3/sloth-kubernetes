# CI/CD Pipeline Documentation

Este projeto utiliza GitHub Actions para automação de CI/CD, garantindo qualidade de código através de testes, linting e builds automatizados.

## Workflows Disponíveis

### 1. CI Workflow (`.github/workflows/ci.yml`)

**Triggers:**
- Push para branch `main`
- Pull Requests para branch `main`

**Jobs:**

#### Test Job
- Executa todos os testes unitários com race detector
- Gera relatório de coverage
- Verifica threshold mínimo de coverage (15%)
- Upload de coverage para Codecov (opcional)

#### Lint Job
- Executa `golangci-lint` com configuração customizada
- Verifica qualidade e padrões de código
- Timeout de 5 minutos

#### Build Job
- Depende dos jobs de Test e Lint
- Compila binários para múltiplas plataformas:
  - Linux (amd64, arm64)
  - macOS/Darwin (amd64, arm64)
  - Windows (amd64, arm64)
- Armazena artefatos por 7 dias

### 2. Release Workflow (`.github/workflows/release.yml`)

**Triggers:**
- Tags com formato `v*.*.*` (ex: v1.0.0, v2.1.3)

**Jobs:**

#### Test Job
- Executa testes antes do release
- Verifica coverage mínimo
- Garante que apenas código testado seja liberado

#### GoReleaser Job
- Depende do job de Test
- Compila binários para todas as plataformas
- Cria release no GitHub
- Gera changelog automaticamente
- Publica artefatos e checksums

## Executando Localmente

### Pré-requisitos

```bash
# Instalar ferramentas de desenvolvimento
make install-tools
```

### Comandos Disponíveis

```bash
# Executar todos os checks de CI
make ci

# Executar apenas testes
make test

# Executar testes com coverage
make test-coverage

# Gerar relatório HTML de coverage
make test-coverage-html

# Executar linter
make lint

# Formatar código
make fmt

# Executar go vet
make vet

# Build local
make build

# Build para todas as plataformas
make build-all
```

## Configurações

### Coverage Threshold

O threshold mínimo de coverage está configurado em **15%**. Isso significa que:
- PRs com coverage abaixo de 15% não passarão no CI
- Releases não serão criados se o coverage estiver abaixo deste valor

Para ajustar o threshold, edite o valor `THRESHOLD` em:
- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`

### Linter Configuration

O golangci-lint está configurado em `.golangci.yml` com os seguintes linters habilitados:
- errcheck
- gosimple
- govet
- ineffassign
- staticcheck
- unused
- gofmt
- goimports
- misspell
- revive
- gosec
- unconvert
- unparam
- gocritic

### GoReleaser Hooks

O GoReleaser está configurado para executar antes do build:
- `go mod tidy`
- `go mod download`
- `go test -race -coverprofile=coverage.txt -covermode=atomic ./...`
- `go vet ./...`

## Badges de Status

Adicione estes badges ao README principal:

```markdown
[![CI](https://github.com/chalkan3/sloth-kubernetes/actions/workflows/ci.yml/badge.svg)](https://github.com/chalkan3/sloth-kubernetes/actions/workflows/ci.yml)
[![Release](https://github.com/chalkan3/sloth-kubernetes/actions/workflows/release.yml/badge.svg)](https://github.com/chalkan3/sloth-kubernetes/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/chalkan3/sloth-kubernetes/branch/main/graph/badge.svg)](https://codecov.io/gh/chalkan3/sloth-kubernetes)
```

## Criando um Release

1. Certifique-se de que todos os testes estão passando
2. Crie e push uma tag com formato semântico:

```bash
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

3. O workflow de release será acionado automaticamente
4. O release será publicado em: https://github.com/chalkan3/sloth-kubernetes/releases

## Troubleshooting

### Testes Falhando Localmente

```bash
# Limpar cache de testes
go clean -testcache

# Executar testes com verbose
go test -v ./...
```

### Linter Reportando Erros

```bash
# Executar linter localmente
golangci-lint run

# Auto-fix quando possível
golangci-lint run --fix
```

### Build Falhando

```bash
# Limpar artefatos
make clean

# Verificar dependências
go mod verify
go mod tidy
```

## Contribuindo

Ao contribuir com código:
1. Certifique-se de que `make ci` passa localmente
2. Escreva testes para novas funcionalidades
3. Mantenha ou melhore o coverage atual
4. Siga os padrões de código (executar `make fmt` antes de commitar)
