.PHONY: help test test-coverage test-race lint fmt vet build clean install-tools ci

# Variables
BINARY_NAME=sloth-kubernetes
COVERAGE_FILE=coverage.txt
COVERAGE_HTML=coverage.html
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run tests
	@echo "Running tests..."
	go test -v -short ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@echo ""
	@echo "Coverage summary:"
	go tool cover -func=$(COVERAGE_FILE) | grep total

test-coverage-html: test-coverage ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	go test -v -race ./...

lint: ## Run golangci-lint
	@echo "Running linter..."
	golangci-lint run --timeout=5m

fmt: ## Format Go code
	@echo "Formatting code..."
	gofmt -s -w $(GO_FILES)
	goimports -w $(GO_FILES)

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	go build -v -ldflags="-s -w" -o $(BINARY_NAME) ./main.go
	@echo "Binary built: $(BINARY_NAME)"

build-all: ## Build binaries for all platforms
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build -v -ldflags="-s -w" -o $(BINARY_NAME)-linux-amd64 ./main.go
	GOOS=linux GOARCH=arm64 go build -v -ldflags="-s -w" -o $(BINARY_NAME)-linux-arm64 ./main.go
	GOOS=darwin GOARCH=amd64 go build -v -ldflags="-s -w" -o $(BINARY_NAME)-darwin-amd64 ./main.go
	GOOS=darwin GOARCH=arm64 go build -v -ldflags="-s -w" -o $(BINARY_NAME)-darwin-arm64 ./main.go
	GOOS=windows GOARCH=amd64 go build -v -ldflags="-s -w" -o $(BINARY_NAME)-windows-amd64.exe ./main.go
	@echo "All binaries built successfully"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f $(COVERAGE_FILE)
	rm -f $(COVERAGE_HTML)
	go clean -cache -testcache -modcache
	@echo "Clean complete"

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Tools installed successfully"

ci: vet lint test-coverage ## Run CI checks locally
	@echo ""
	@echo "============================================"
	@echo "All CI checks passed! ✅"
	@echo "============================================"

mod-tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	go mod tidy

mod-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	go mod verify

deps: mod-tidy mod-verify ## Download and verify dependencies
	@echo "Downloading dependencies..."
	go mod download
