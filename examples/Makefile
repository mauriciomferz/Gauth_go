# Example Makefile targets

.PHONY: examples test bench coverage docs clean

# Default target
all: test examples bench

# Run all examples
examples: example-basic example-auth example-rate example-token

# Basic example
example-basic:
	@echo "Running basic example..."
	cd examples/basic && go run main.go

# Authentication example
example-auth:
	@echo "Running authentication example..."
	cd examples/auth && go run main.go

# Rate limiting example
example-rate:
	@echo "Running rate limiting example..."
	cd examples/rate && go run main.go

# Token management example
example-token:
	@echo "Running token management example..."
	cd examples/token_management && go run main.go

# Run tests
test:
	go test ./...

# Run tests with race detection
test-race:
	go test -race ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./test/benchmarks/...

# Run specific benchmark
bench-%:
	go test -bench=$* -benchmem ./test/benchmarks/...

# Generate test coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Generate documentation
docs:
	@mkdir -p docs/api
	godoc -http=:6060

# Clean build artifacts
clean:
	go clean
	rm -f coverage.out

# Install dependencies
deps:
	go mod download

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golint ./...

# Check for race conditions
race:
	go test -race ./...

# Generate profiling data
profile:
	go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./test/benchmarks/...

# Example help
help:
	@echo "Available targets:"
	@echo "  make examples      - Run all examples"
	@echo "  make test         - Run tests"
	@echo "  make bench        - Run benchmarks"
	@echo "  make coverage     - Generate test coverage"
	@echo "  make docs         - Generate documentation"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make deps         - Install dependencies"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Lint code"
	@echo "  make race         - Check for race conditions"
	@echo "  make profile      - Generate profiling data"
	@echo "  make help         - Show this help"