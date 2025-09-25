.PHONY: all build test clean lint coverage examples docs help security deps bench

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=gauth

# Build flags
LDFLAGS=-ldflags="-s -w"

all: test build

build: ## Build all binaries
	@echo "Building GAuth binaries..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-server -v ./cmd/demo
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-demo -v ./demo
	@echo "✅ Build completed successfully!"

test: ## Run all tests
	$(GOTEST) -v ./pkg/... ./internal/... ./examples/cascade/pkg/gauth ./test/...

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)-server
	rm -f $(BINARY_NAME)-demo
	rm -f coverage.out
	rm -f *.html

lint: ## Run linter
	golangci-lint run ./...

coverage: ## Generate test coverage report
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

deps: ## Download and tidy dependencies
	$(GOMOD) download
	$(GOMOD) tidy
	$(GOMOD) verify

examples: ## Build example binaries (note: many examples don't have main packages)
	@echo "Building available example binaries..."
	@for dir in examples/*/; do \
		if [ -f "$$dir/main.go" ]; then \
			name=$$(basename "$$dir"); \
			echo "Building example: $$name"; \
			$(GOBUILD) -o "$$dir/$$name" "./$$dir" || echo "⚠️  Failed to build $$name"; \
		fi \
	done

bench: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

docs: ## Start documentation server
	godoc -http=:6060

security: ## Run security scan
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Generate mocks for testing
mocks:
	mockgen -destination internal/mocks/tokenstore_mock.go -package mocks github.com/Gimel-Foundation/gauth/internal/tokenstore Store
	mockgen -destination internal/mocks/audit_mock.go -package mocks github.com/Gimel-Foundation/gauth/internal/audit Logger

# Run all checks before committing
pre-commit: lint test security

# Install development tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/godoc@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/golang/mock/mockgen@latest

# Format code
fmt:
	gofmt -s -w .
	goimports -w .

# Version management
VERSION ?= $(shell git describe --tags --always --dirty)
version:
	@echo $(VERSION)

# Create a new release tag
tag:
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)