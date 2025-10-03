.PHONY: all build test clean lint coverage docs help security deps format

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build configuration
BINARY_NAME=gauth
BINARY_DIR=build/bin
LDFLAGS=-ldflags="-s -w"

# Default target
all: deps format test build

## Build targets
build: build-server build-security-test ## Build all binaries

build-server: ## Build the demo server
	@echo "🔧 Building GAuth demo server..."
	mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-server ./cmd/demo

build-security-test: ## Build the security test tool
	@echo "🔐 Building GAuth security test tool..."
	mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-security-test ./cmd/security-test

## Test targets
test: ## Run all tests
	@echo "🧪 Running test suite..."
	$(GOCLEAN) -testcache
	$(GOTEST) -v -race -timeout=30s ./...

test-coverage: ## Run tests with coverage
	@echo "📊 Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

test-integration: ## Run integration tests
	@echo "🔗 Running integration tests..."
	$(GOTEST) -v -tags=integration ./test/integration/...

## Code quality targets
lint: ## Run linters
	@echo "🔍 Running linters..."
	golangci-lint run ./...

format: ## Format code
	@echo "📝 Formatting code..."
	$(GOFMT) ./...
	$(GOCMD) mod tidy

security: ## Run security scans
	@echo "🛡️  Running security scan..."
	gosec ./...

## Development targets
deps: ## Install dependencies
	@echo "📦 Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

## Docker targets
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t gauth:latest .

docker-run: ## Run Docker container
	@echo "🚀 Running Docker container..."
	docker run -p 8080:8080 gauth:latest

## Utility targets
run-server: build-server ## Build and run demo server
	./$(BINARY_DIR)/$(BINARY_NAME)-server

run-security-test: build-security-test ## Build and run security test tool
	./$(BINARY_DIR)/$(BINARY_NAME)-security-test

## Documentation
docs: ## Generate documentation
	@echo "📖 Generating documentation..."
	@echo "Documentation generated from source code comments"
	$(GOCMD) doc -all ./pkg/auth
	$(GOCMD) doc -all ./pkg/rfc
	$(GOCMD) doc -all ./internal

help: ## Show this help message
	@echo "GAuth Makefile Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make build               # Build all binaries"
	@echo "  make test                # Run all tests"
	@echo "  make run-server          # Build and run demo server"
	@echo "  make run-security-test   # Build and run security test tool"
	@echo "  make docker-build        # Build Docker image"