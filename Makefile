.PHONY: all build test clean lint coverage examples docs

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=gauth

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

lint:
	golangci-lint run ./...

coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

deps:
	$(GOMOD) download
	$(GOMOD) tidy

examples: build
	$(GOBUILD) -o examples/basic/basic ./examples/basic
	$(GOBUILD) -o examples/advanced/advanced ./examples/advanced

bench:
	$(GOTEST) -bench=. -benchmem ./...

docs:
	godoc -http=:6060

# Security scanning
security:
	gosec ./...

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