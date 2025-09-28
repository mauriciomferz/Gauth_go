# GAuth Demo Application Makefile

.PHONY: help build-all test-all clean run-web run-cli install-deps docker-build

# Default target
help:
	@echo "GAuth Demo Application"
	@echo "====================="
	@echo ""
	@echo "Available targets:"
	@echo "  help          - Show this help message"
	@echo "  install-deps  - Install all dependencies"
	@echo "  build-all     - Build all components"
	@echo "  test-all      - Run all tests"
	@echo "  run-web       - Start web application"
	@echo "  run-cli       - Build and run CLI demo"
	@echo "  docker-build  - Build Docker images"
	@echo "  clean         - Clean build artifacts"

# Install dependencies
install-deps: install-go-deps install-node-deps install-python-deps

install-go-deps:
	@echo "Installing Go dependencies..."
	cd web/backend && go mod tidy
	cd cli && go mod tidy

install-node-deps:
	@echo "Installing Node.js dependencies..."
	cd web/frontend && npm install

install-python-deps:
	@echo "Installing Python dependencies..."
	cd python-sdk && pip install -e .
	cd python-sdk && pip install -r requirements-dev.txt

# Build all components
build-all: build-backend build-frontend build-cli build-python

build-backend:
	@echo "Building Go backend..."
	cd web/backend && go build -o gauth-web-server .

build-frontend:
	@echo "Building React frontend..."
	cd web/frontend && npm run build

build-cli:
	@echo "Building CLI tool..."
	cd cli && go build -o gauth-cli .

build-python:
	@echo "Building Python SDK..."
	cd python-sdk && python setup.py build

# Test all components
test-all: test-go test-python test-web test-integration

test-go:
	@echo "Running Go tests..."
	cd web/backend && go test ./...
	cd cli && go test ./...
	cd shared && go test ./...

test-python:
	@echo "Running Python tests..."
	cd python-sdk && python -m pytest tests/

test-web:
	@echo "Running web frontend tests..."
	cd web/frontend && npm test

test-integration:
	@echo "Running integration tests..."
	cd tests && go test ./...

# Run applications
run-web:
	@echo "Starting web application..."
	@echo "Backend will be available at http://localhost:8080"
	@echo "Frontend will be available at http://localhost:3000"
	cd web && make run

run-cli:
	@echo "Running CLI demo..."
	cd cli && make demo

# Docker operations
docker-build:
	@echo "Building Docker images..."
	docker build -t gauth-web -f web/Dockerfile web/
	docker build -t gauth-cli -f cli/Dockerfile cli/
	docker build -t gauth-python -f python-sdk/Dockerfile python-sdk/

docker-run:
	@echo "Starting Docker containers..."
	docker-compose up -d

docker-stop:
	@echo "Stopping Docker containers..."
	docker-compose down

# Development tools
dev-setup:
	@echo "Setting up development environment..."
	@make install-deps
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest
	npm install -g @typescript-eslint/parser @typescript-eslint/eslint-plugin

dev-run:
	@echo "Starting development servers..."
	cd web && make dev

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	cd web/backend && go clean
	cd cli && go clean
	cd web/frontend && npm run clean || rm -rf build/ dist/
	cd python-sdk && rm -rf build/ dist/ *.egg-info/
	rm -f web/backend/gauth-web-server
	rm -f cli/gauth-cli

# Format code
fmt:
	@echo "Formatting code..."
	cd web/backend && go fmt ./...
	cd cli && go fmt ./...
	cd shared && go fmt ./...
	cd web/frontend && npm run format
	cd python-sdk && black pygauth/ tests/

# Lint code
lint:
	@echo "Linting code..."
	cd web/backend && golangci-lint run
	cd cli && golangci-lint run
	cd web/frontend && npm run lint
	cd python-sdk && pylint pygauth/

# Generate documentation
docs:
	@echo "Generating documentation..."
	cd web/backend && godoc -http=:6060 &
	cd python-sdk && sphinx-build -b html docs/ docs/_build/

# Benchmark tests
benchmark:
	@echo "Running benchmarks..."
	cd web/backend && go test -bench=. ./...
	cd python-sdk && python -m pytest tests/ --benchmark-only