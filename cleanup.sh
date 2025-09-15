#!/bin/bash
# Script to clean and fix common issues in the GAuth codebase

echo "Starting GAuth codebase cleanup..."

# Fix permissions for go modules directory
echo "Fixing Go module permissions..."
sudo chown -R $(whoami) $HOME/go/pkg/mod
chmod -R u+w $HOME/go/pkg/mod

# Clean up Go module cache
echo "Cleaning Go module cache..."
go clean -cache -modcache

# Run go mod tidy to fix dependencies
echo "Running go mod tidy..."
go mod tidy

# Run gofmt on all Go files
echo "Running gofmt on all files..."
find . -name "*.go" -type f -exec gofmt -w {} \;

# Run go vet to check for issues
echo "Running go vet..."
go vet ./...

# Build the project
echo "Building the project..."
go build ./...

echo "Cleanup complete!"