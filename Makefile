rfc_clause_index: ## Generate RFC clause index JSON from placeholder spec files
	@echo "📘 Generating RFC clause index..."; \
	go run ./cmd/rfcscan -out docs/rfc/rfc_clause_index.json > /dev/null || exit 1; \
	CLAUSES=$$(jq '.clauses | length' docs/rfc/rfc_clause_index.json); \
	echo "✅ RFC clause index generated (clauses=$$CLAUSES)";

crypto-rotate: ## Manually rotate Ed25519 key (requires GAUTH_TOKEN_SIG_MODE=eddsa); prints new kid
	@echo "🔄 Rotating Ed25519 key..."; \
	GAUTH_TOKEN_SIG_MODE=eddsa go test -run TestEdDSA -count=1 ./pkg/gauth >/dev/null || true; \
	go run ./scripts/rotate_key.go || echo "(rotate script missing - future implementation)";

crypto-test: ## Run EdDSA-focused tests only
	@echo "🧪 Running EdDSA test subset..."; \
	GAUTH_TOKEN_SIG_MODE=eddsa $(GOTEST) -run TestEdDSA ./pkg/gauth -count=1; \
	echo "✅ EdDSA tests passed";

## Revocation System Targets
test-revocation: ## Run revocation system tests (requires Redis on localhost:6379)
	@echo "🔒 Running revocation system tests..."; \
	$(GOTEST) -v -count=1 -timeout=5m ./pkg/revocation/...; \
	echo "✅ Revocation tests passed (77 tests)";

test-revocation-race: ## Run revocation tests with race detector (requires Redis)
	@echo "🏁 Running revocation tests with race detector..."; \
	$(GOTEST) -race -v -count=1 -timeout=10m ./pkg/revocation/...; \
	echo "✅ Revocation race tests passed";

test-revocation-coverage: ## Generate coverage report for revocation system
	@echo "📊 Generating revocation test coverage..."; \
	$(GOTEST) -coverprofile=coverage-revocation.out -covermode=atomic ./pkg/revocation/...; \
	go tool cover -html=coverage-revocation.out -o coverage-revocation.html; \
	COVERAGE=$$(go tool cover -func=coverage-revocation.out | grep total | awk '{print $$3}'); \
	echo "✅ Revocation coverage: $$COVERAGE (report: coverage-revocation.html)";

bench-revocation: ## Run revocation system benchmarks
	@echo "⚡ Running revocation benchmarks..."; \
	$(GOTEST) -bench=. -benchmem -benchtime=5s ./pkg/revocation/...; \
	echo "✅ Benchmarks complete (baseline: 67k ops/sec)";

.PHONY: all build test test-all clean lint coverage docs help security deps format ci verify-csp verify-build-env debug-ci-env build-ci-adaptive build-fallback ci-build build-server-adaptive
.PHONY: gap-matrix gap-matrix-check openapi-guard
.PHONY: js-lint js-test js-build
.PHONY: js-bundle
.PHONY: coverage-rfc0111 tidy-all
.PHONY: test-revocation test-revocation-race test-revocation-coverage bench-revocation

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

ci: ## Run full CI suite locally (format check, vet, lint, race tests)
	@echo "🏗  Running local CI pipeline..."
	@echo "▶ Format check"; \
	gofmt -l . | grep . && echo "Formatting issues found (run 'make format')" && exit 1 || echo "Format OK"; \
	go vet ./...; \
	command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	golangci-lint run ./...; \
	$(GOTEST) -race -count=1 ./...; \
	echo "✅ CI suite passed"

## Build targets
verify-build-env: ## Verify build environment and directory structure
	@echo "🔍 Verifying build environment..."
	@echo "📍 Current working directory: $$(pwd)"
	@echo "📂 Repository structure verification:"
	@test -d "./cmd" || (echo "❌ ./cmd directory missing" && exit 1)
	@test -d "./cmd/gauth-server" || (echo "❌ ./cmd/gauth-server directory missing" && exit 1)
	@test -f "./cmd/gauth-server/main.go" || (echo "❌ ./cmd/gauth-server/main.go missing" && exit 1)
	@test -f "./go.mod" || (echo "❌ ./go.mod missing" && exit 1)
	@echo "✅ Build environment verified successfully"

debug-ci-env: ## Debug CI environment (detailed diagnostics)
	@echo "🐛 CI Environment Debug Information"
	@echo "=================================="
	@echo "📍 Working Directory: $$(pwd)"
	@echo "🏠 Home Directory: $$HOME"
	@echo "👤 Current User: $$(whoami)"
	@echo "💻 System: $$(uname -a)"
	@echo ""
	@echo "📁 Directory Structure:"
	@ls -la . | head -20
	@echo ""
	@echo "📂 cmd/ Directory:"
	@if [ -d "./cmd" ]; then ls -la ./cmd/; else echo "❌ ./cmd directory not found"; fi
	@echo ""
	@echo "🔍 Go Environment:"
	@go version || echo "❌ Go not found"
	@echo "GOPATH: $$GOPATH"
	@echo "GOROOT: $$GOROOT"
	@echo ""
	@echo "📋 go.mod Information:"
	@if [ -f "./go.mod" ]; then head -5 ./go.mod; else echo "❌ ./go.mod not found"; fi

build-ci-adaptive: ## CI-adaptive build that handles nested directory structures
	@echo "🔧 CI-Adaptive Build for AgentAuth Server"
	@echo "======================================"
	@echo "📍 Working Directory: $$(pwd)"
	@echo ""
	@echo "🔍 Searching for source files..."
	@if [ -f "./cmd/gauth-server/main.go" ]; then \
		echo "✅ Method 1: Found ./cmd/gauth-server/main.go"; \
		SOURCE_PATH="./cmd/gauth-server"; \
	elif [ -f "cmd/gauth-server/main.go" ]; then \
		echo "✅ Method 2: Found cmd/gauth-server/main.go"; \
		SOURCE_PATH="cmd/gauth-server"; \
	else \
		echo "🔍 Method 3: Searching filesystem..."; \
		MAIN_GO_PATH=$$(find . -name "main.go" -path "*/cmd/gauth-server/*" 2>/dev/null | head -1); \
		if [ -n "$$MAIN_GO_PATH" ]; then \
			SOURCE_PATH=$$(dirname "$$MAIN_GO_PATH"); \
			echo "✅ Method 3: Found $$MAIN_GO_PATH"; \
			echo "✅ Using source path: $$SOURCE_PATH"; \
		else \
			echo "❌ FAILED: Cannot find gauth-server/main.go anywhere"; \
			echo "📋 Available main.go files:"; \
			find . -name "main.go" 2>/dev/null | head -10 || echo "No main.go files found"; \
			exit 1; \
		fi; \
	fi; \
	echo "🏗️ Building with source path: $$SOURCE_PATH"; \
	mkdir -p $(BINARY_DIR); \
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-server "$$SOURCE_PATH"

build: verify-build-env build-server-adaptive build-security-test ## Build all binaries

build-fallback: ## Fallback build method for CI environments with path issues
	@echo "🔧 Fallback Build Method"
	@echo "========================"
	@if $(MAKE) build-server 2>/dev/null; then \
		echo "✅ Standard build succeeded"; \
	else \
		echo "⚠️ Standard build failed, trying adaptive method..."; \
		$(MAKE) build-ci-adaptive; \
	fi

ci-build: ## Recommended build target for CI/CD environments  
	@echo "🚀 CI/CD Build Process"
	@echo "======================"
	@echo "📊 Environment Check:"
	@$(MAKE) debug-ci-env
	@echo ""
	@echo "🏗️ Building with adaptive method:"
	@$(MAKE) build-ci-adaptive

build-server-adaptive: ## Build gauth-server with adaptive path detection (recommended)
	@echo "🔧 Building AgentAuth demo server (adaptive method)..."
	@echo "📍 Current working directory: $$(pwd)"
	@echo "� Searching for gauth-server source..."
	@SOURCE_PATH=""; \
	if [ -f "./cmd/gauth-server/main.go" ]; then \
		SOURCE_PATH="./cmd/gauth-server"; \
		echo "✅ Method 1: Found ./cmd/gauth-server/main.go"; \
	elif [ -f "cmd/gauth-server/main.go" ]; then \
		SOURCE_PATH="cmd/gauth-server"; \
		echo "✅ Method 2: Found cmd/gauth-server/main.go"; \
	else \
		FOUND_PATH=$$(find . -name "main.go" -path "*/cmd/gauth-server/*" 2>/dev/null | head -1); \
		if [ -n "$$FOUND_PATH" ]; then \
			SOURCE_PATH=$$(dirname "$$FOUND_PATH"); \
			echo "✅ Method 3: Found $$FOUND_PATH"; \
		fi; \
	fi; \
	if [ -n "$$SOURCE_PATH" ]; then \
		echo "🏗️ Building with source path: $$SOURCE_PATH"; \
		mkdir -p $(BINARY_DIR); \
		$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-server "$$SOURCE_PATH"; \
		echo "✅ Build completed successfully"; \
	else \
		echo "❌ ERROR: Cannot find gauth-server source code"; \
		echo "📋 Current directory contents:"; \
		ls -la . | head -10; \
		echo "📋 cmd directory contents:"; \
		ls -la cmd/ 2>/dev/null || echo "No cmd directory found"; \
		echo "📋 Search results for gauth-server directories:"; \
		find . -name "gauth-server" -type d 2>/dev/null | head -5 || echo "No gauth-server directories found"; \
		exit 1; \
	fi

build-server: ## Build the CLI demo server (legacy - calls adaptive version)
	@echo "🔄 Redirecting to adaptive build method..."
	@$(MAKE) build-server-adaptive

build-web: js-build ## Build the web demo server (cmd/web-server) (includes JS bundle embed)
	@echo "🌐 Building web demo server..."
	mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-web ./cmd/web-server

build-security-test: ## Build the RFC demo
	@echo "🔐 Building RFC demo..."
	mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-rfc-demo ./cmd/examples/rfc-demo

## Test targets
test: ## Run all tests (CI-safe, skips performance tests)
	@echo "🧪 Running test suite (skipping performance tests)..."
	$(GOCLEAN) -testcache
	$(GOTEST) -short -p=1 -v -timeout=60s ./...

test-all: ## Run all tests including performance/load tests
	@echo "🧪 Running complete test suite (including performance tests)..."
	$(GOCLEAN) -testcache
	$(GOTEST) -v -timeout=5m ./...

test-race: ## Run all tests with race detection (skips performance tests)
	@echo "🏁 Running test suite with race detection..."
	$(GOCLEAN) -testcache
	$(GOTEST) -short -v -race -timeout=60s ./...

test-coverage: ## Run tests with coverage
	@echo "📊 Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

test-integration: ## Run integration tests
	@echo "🔗 Running integration tests..."
	$(GOTEST) -v -tags=integration ./test/integration/...

test-rfc0111-example: ## Run AAP-001 example tests (F=Regex to filter, default runs both)
	@echo "🧪 Running AAP-001 example tests..."
	@if [ -n "$(F)" ]; then \
	  echo "▶ Filter: $(F)"; \
	  $(GOTEST) -v ./examples/official_rfc0111_implementation -run $(F) -count=1; \
	else \
	  $(GOTEST) -v ./examples/official_rfc0111_implementation -run TestDelegationLifecycle -count=1; \
	  $(GOTEST) -v ./examples/official_rfc0111_implementation -run TestUnauthorizedRevocation -count=1; \
	fi

coverage-rfc0111: ## Run coverage for AAP-001 example (outputs coverage.out.rfc0111 and summary)
	@echo "📊 AAP-001 example coverage..."; \
	$(GOTEST) -cover -coverprofile=coverage.out.rfc0111 ./examples/official_rfc0111_implementation -count=1 > /dev/null; \
	go tool cover -func=coverage.out.rfc0111 | tee coverage.rfc0111.txt | grep total || true; \
	echo "✅ Coverage artifacts: coverage.out.rfc0111, coverage.rfc0111.txt";

tidy-all: ## Run go mod tidy across root and any referenced modules (go.work aware)
	@echo "🧹 Tidying modules..."; \
	$(GOMOD) tidy; \
	if [ -f go.work ]; then \
		awk '/use / {print $$2}' go.work | while read m; do echo "→ tidy $$m"; (cd $$m && go mod tidy); done; \
	fi; \
	echo "✅ Modules tidied"

tidy-fast: ## Fast hygiene (format-enhanced + vet + minimal lint) without tests
	@echo "⚡ Fast tidy (imports, format, vet, minimal lint)"; \
	$(MAKE) format-enhanced; \
	go vet ./...; \
	command -v golangci-lint >/dev/null 2>&1 && golangci-lint run --disable-all --enable=ineffassign --enable=errcheck || echo "(golangci-lint skipped)"; \
	echo "✅ Fast tidy complete"

## Code quality targets
lint: ## Run default golangci-lint (project .golangci.yml)
	@echo "🔍 Running golangci-lint (standard config)..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint (local)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run --config .golangci.yml ./...
	@echo "✅ Lint passed (standard)"

lint-minimal: ## Run fast minimal lint (formatting + essential static checks)
	@echo "⚡ Running minimal lint (.golangci-minimal.yml)..."; \
	if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint (local)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi; \
	gofmt -l . | grep . && echo "❌ Formatting issues (run 'make format')" && exit 1 || true; \
	golangci-lint run --config .golangci-minimal.yml ./... || exit 1; \
	echo "✅ Minimal lint passed";

lint-strict: ## Run strict lint (treat warnings as errors, fail on revive warnings)
	@echo "🧪 Running strict lint (warnings elevated)..."; \
	if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint (local)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi; \
	golangci-lint run --config .golangci.yml --issues-exit-code=1 --max-issues-per-linter=0 --max-same-issues=0 ./... || { echo "❌ Strict lint failed"; exit 1; }; \
	echo "✅ Strict lint passed";

vet: ## Run go vet
	@echo "🔎 Running go vet..."
	go vet ./...
	@echo "✅ Vet passed"

tidy: ## Run formatting, module tidy, vet (fast hygiene). Set DRY=1 to preview changes only.
	@echo "🧹 Running tidy (format + mod tidy + vet)..."; \
	if [ "$${DRY}" = "1" ]; then \
	  echo "(Dry run) Listing files that would change from gofmt:"; \
	  gofmt -l . | sed 's/^/  • /'; \
	else \
	  go fmt ./...; \
	fi; \
	if [ "$${DRY}" = "1" ]; then \
	  echo "(Dry run) Skipping go mod tidy"; \
	else \
	  go mod tidy; \
	fi; \
	go vet ./... || { echo "❌ vet issues detected"; exit 1; }; \
	if command -v golangci-lint >/dev/null 2>&1; then \
	  echo "▶ golangci-lint (fast subset)"; \
	  golangci-lint run --disable-all --enable=ineffassign --enable=errcheck || { echo "❌ lint issues"; exit 1; }; \
	else \
	  echo "(golangci-lint not installed - skipping fast subset)"; \
	fi; \
	echo "✅ tidy complete";

todo-report: ## Generate CODE_TODO_REPORT.md from current TODO/FIXME markers (filters out vendor/build/git)
	@echo "📝 Generating TODO report..."; \
	TMP=$$(mktemp); \
	grep -R "TODO\|FIXME" -n --exclude-dir=node_modules --exclude-dir=bin --exclude-dir=build --exclude-dir=.git --exclude='*.map' . > $$TMP || true; \
	echo "# Code TODO / FIXME Report" > docs/CODE_TODO_REPORT.md; \
	echo "Generated via 'make todo-report'." >> docs/CODE_TODO_REPORT.md; \
	echo "" >> docs/CODE_TODO_REPORT.md; \
	echo "\n## Raw Matches" >> docs/CODE_TODO_REPORT.md; \
	echo '\n```' >> docs/CODE_TODO_REPORT.md; \
	cat $$TMP >> docs/CODE_TODO_REPORT.md; \
	echo '\n```' >> docs/CODE_TODO_REPORT.md; \
	echo "✅ TODO report updated (docs/CODE_TODO_REPORT.md)"; \
	rm -f $$TMP;

hygiene: ## Run tidy plus TODO report (fast combined hygiene task)
	@$(MAKE) tidy
	@$(MAKE) todo-report
	@echo "✅ hygiene complete"

lint-fast: ## Run fast subset of linters (no build tags, fewer checks)
	@echo "⚡ Fast lint (format + vet + ineffassign)"; \
	gofmt -l . | grep . && echo "Please run 'make format' to apply formatting" && exit 1 || true; \
	go vet ./...; \
	command -v golangci-lint >/dev/null 2>&1 && golangci-lint run --disable-all --enable=ineffassign --enable=errcheck || echo "(Skipped extended lint)";

format: ## Format code
	@echo "📝 Formatting code..."
	$(GOFMT) ./...
	$(GOCMD) mod tidy

format-enhanced: ## Enhanced formatting (goimports + gofmt + tidy via scripts/format.sh)
	@echo "🛠  Enhanced formatting (imports + canonical style + tidy)"
	@if [ ! -x scripts/format.sh ]; then chmod +x scripts/format.sh; fi; \
		scripts/format.sh

format-dry: ## Preview formatting changes without modifying files (DRY=1)
	@echo "🔍 Dry-run format preview"
	@if [ ! -x scripts/format.sh ]; then chmod +x scripts/format.sh; fi; \
		DRY=1 scripts/format.sh || true

lint-fix: ## Apply formatting & import fixes (gofmt + goimports; install tools if missing)
	@echo "🛠  Auto-fixing common lint issues (format + imports)..."; \
	if ! command -v goimports >/dev/null 2>&1; then \
		 echo "Installing goimports..."; \
		 go install golang.org/x/tools/cmd/goimports@latest; \
	fi; \
	goimports -w ./; \
	gofmt -w ./; \
	$(GOMOD) tidy; \
	echo "✅ lint-fix complete (run 'make lint-minimal' to verify)";

security: ## Run security scans
	@echo "🛡️  Running security scan..."
	gosec ./...

security-full: ## Run extended security tooling (educational, non-blocking)
	@echo "🛡️  Running extended security checks (vet, lint, gosec, govulncheck)"
	go vet ./... || echo "vet warnings present"
	golangci-lint run ./... || echo "golangci-lint warnings present"
	gosec ./... || echo "gosec findings present"
	go install golang.org/x/vuln/cmd/govulncheck@latest >/dev/null 2>&1 || true
	govulncheck ./... || echo "govulncheck findings present"

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

docker-build-minimal: ## Build minimal beta web image (Dockerfile.minimal)
	@echo "🐳 Building minimal beta web image..."
	docker build -f Dockerfile.minimal -t gauth:minimal .

docker-smoke-minimal: ## Build & run smoke test against minimal image
	@echo "🧪 Running minimal image smoke test..."; \
	if [ ! -x scripts/smoke-minimal.sh ]; then chmod +x scripts/smoke-minimal.sh; fi; \
	scripts/smoke-minimal.sh

docker-run: ## Run Docker container
	@echo "🚀 Running Docker container..."
	docker run -p 8080:8080 gauth:latest

docker-dev-build: ## Build developer hot-reload image (Dockerfile.dev) (set GO_VERSION / AIR_VERSION / INSTALL_AIR)
	@echo "🐳 Building dev image (Dockerfile.dev)..."; \
	BUILD_ARGS=""; \
	[ -n "$$GO_VERSION" ] && BUILD_ARGS="$$BUILD_ARGS --build-arg GO_VERSION=$$GO_VERSION"; \
	[ -n "$$AIR_VERSION" ] && BUILD_ARGS="$$BUILD_ARGS --build-arg AIR_VERSION=$$AIR_VERSION"; \
	[ -n "$$INSTALL_AIR" ] && BUILD_ARGS="$$BUILD_ARGS --build-arg INSTALL_AIR=$$INSTALL_AIR"; \
	docker build $$BUILD_ARGS -f Dockerfile.dev -t gauth-dev .

docker-dev-run: ## Run dev hot-reload container (mount source, exposes :8080)
	@echo "🌀 Starting dev hot-reload container (Ctrl+C to stop)..."; \
	docker run --rm -it -p $${GAUTH_DEV_PORT:-8080}:8080 -v "$(PWD)":/app -e GAUTH_BETA=1 gauth-dev

## Utility targets
run-server: build-server ## Build and run CLI demo server
	./$(BINARY_DIR)/$(BINARY_NAME)-server

run-web: build-web ## Build and run web demo server (serves CSP-hardened UI)
	./$(BINARY_DIR)/$(BINARY_NAME)-web

run-rfc-demo: build-security-test ## Build and run RFC demo
	./$(BINARY_DIR)/$(BINARY_NAME)-rfc-demo

## Documentation
docs: ## Generate aggregated API documentation (scripts/gen-docs.sh -> docs/GENERATED_API.md)
	@echo "📖 Generating aggregated API docs..."
	@if [ ! -x scripts/gen-docs.sh ]; then chmod +x scripts/gen-docs.sh; fi; \
	./scripts/gen-docs.sh

docs-validate: ## Validate documentation metadata headers (scripts/docs_index.sh --validate)
	@echo "🧪 Validating documentation headers..."; \
	if [ ! -x scripts/docs_index.sh ]; then chmod +x scripts/docs_index.sh; fi; \
	bash scripts/docs_index.sh --validate

docs-summary: ## Show category counts for documentation corpus
	@echo "📊 Documentation category summary"; \
	if [ ! -x scripts/docs_index.sh ]; then chmod +x scripts/docs_index.sh; fi; \
	bash scripts/docs_index.sh --summary

docs-meta-validate: ## Run Go-based validator for documentation front matter (tools/docvalidate)
	@echo "🧪 Running Go docs validator..."; \
	go run ./tools/docvalidate || exit 1; \
	echo "✅ Docs validation passed"

docs-meta-index: ## Generate taxonomy index (docs/TAXONOMY_INDEX.auto.md) via Go validator
	@echo "🧪 Generating taxonomy index..."; \
	go run ./tools/docvalidate --write-index || exit 1; \
	echo "✅ Taxonomy index written to docs/TAXONOMY_INDEX.auto.md"

help: ## Show this help message
	@echo "AgentAuth Makefile Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make build               # Build all binaries"
	@echo "  make test                # Run all tests"
	@echo "  make run-server          # Build and run CLI demo server"
	@echo "  make run-web             # Build and run web demo server (UI)"
	@echo "  make run-rfc-demo        # Build and run RFC demo"
	@echo "  make docker-build        # Build Docker image"
	@echo "  make web-start           # Start web demo (scripts/start-web-demo.sh)"
	@echo "  make web-stop            # Stop web demo (scripts/stop-web-demo.sh)"
	@echo "  make web-restart         # Restart web demo (stop+start)"
	@echo "  make web-logs            # Tail web demo logs (gauth-web.log)"
	@echo "  make web-health           # Health check for web demo (scripts/health.sh)"
	@echo "  make web-tail-logs        # Tail web demo logs (scripts/tail-logs.sh)"
	@echo "  make web-integration-test  # Run integration tests for web demo (test/integration, integration tag)"
	@echo "  make compose-build         # Build Docker Compose stack (full demo)"
	@echo "  make compose-up            # Start Docker Compose stack (full demo)"
	@echo "  make compose-down          # Stop Docker Compose stack"
	@echo "  make js-lint              # Lint JavaScript (basic syntax via node --check or eslint if installed)"
	@echo "  make js-test              # Run lightweight JS syntax tests (placeholder)"
	@echo "  make js-build             # Bundle or process JS modules (placeholder)"
	@echo "  make gap-matrix          # Generate GAP_MATRIX.auto.md (ignores drift exit)"
	@echo "  make gap-matrix-check    # Enforce drift-free gap matrix (fails on mismatch)"
	@echo "  make openapi-guard       # Minimal structural guard for docs/openapi.yaml"

## Gap Matrix Automation
gap-matrix: ## Generate GAP_MATRIX.auto.md (ignores drift for local review)
	@echo "🧩 Generating gap matrix (non-enforcing)..."; \
	go run ./scripts/gen_gap_matrix.go || echo "(drift detected - see docs/GAP_MATRIX.auto.md for details)"; \
	if [ -f docs/GAP_MATRIX.auto.md ]; then echo "✅ Generated docs/GAP_MATRIX.auto.md"; else echo "❌ Generation failed"; fi;

gap-matrix-check: ## Generate and enforce no drift between CSV and curated markdown
	@echo "🧪 Checking gap matrix drift..."; \
	go run ./scripts/gen_gap_matrix.go; \
	echo "✅ Gap matrix in sync";

openapi-guard: ## Validate minimal OpenAPI spec structure (presence of openapi: & paths:)
	@echo "🔍 Guarding OpenAPI spec..."; \
	go run ./scripts/openapi_guard/main.go || exit $$?;
	echo "✅ OpenAPI guard passed";

JS_FILES=$(shell find frontend/ui-react/src -name '*.js' -o -name '*.jsx' -o -name '*.ts' -o -name '*.tsx' 2>/dev/null || echo "")

js-lint: ## Lint JavaScript/TypeScript (tries eslint, falls back to tsc --noEmit)
	@echo "🔎 Linting JS/TS (front-end)"; \
	if [ -d "frontend/ui-react" ]; then \
	  if command -v eslint >/dev/null 2>&1 && [ -f "frontend/ui-react/.eslintrc.cjs" ]; then \
		(cd frontend/ui-react && eslint --max-warnings=0 src); \
	  elif command -v tsc >/dev/null 2>&1 && [ -f "frontend/ui-react/tsconfig.json" ]; then \
		echo "(eslint not found, running tsc --noEmit for type checking)"; \
		(cd frontend/ui-react && tsc --noEmit); \
	  else \
	    echo "⚠️  Neither eslint nor tsc configured for frontend/ui-react - skipping lint"; \
	  fi; \
	  echo "✅ JS/TS lint passed"; \
	else \
	  echo "⚠️  frontend/ui-react not found - skipping"; \
	fi

js-test: ## Run frontend tests (jest/vitest)
	@echo "🧪 Running frontend tests..."; \
	if [ -f "frontend/ui-react/package.json" ]; then \
	  (cd frontend/ui-react && npm test); \
	  echo "✅ Frontend tests complete"; \
	else \
	  echo "⚠️  frontend/ui-react/package.json not found - skipping tests"; \
	fi

js-build: ## Bundle front-end assets using Vite
	@if [ "$$SKIP_WEB_ASSETS" = "1" ]; then \
	  echo "⏭️  SKIP_WEB_ASSETS=1 -> skipping front-end bundling"; \
	else \
	  echo "📦 Bundling front-end assets with Vite..."; \
	  if [ -f "frontend/ui-react/package.json" ]; then \
	    if [ ! -d "frontend/ui-react/node_modules" ]; then \
	      echo "⬇️  Installing JS dev dependencies (one-time)..."; \
	      (cd frontend/ui-react && npm install --no-audit --no-fund); \
	    fi; \
	    (cd frontend/ui-react && npm run build); \
	    if [ ! -s "frontend/ui-react/dist/index.html" ]; then \
	      echo "❌ Build output missing - check Vite build process"; exit 1; \
	    else \
	      echo "✅ Vite build complete (output in frontend/ui-react/dist)"; \
		  echo "🔄 Syncing to backend static assets..."; \
		  mkdir -p web/static/assets; \
		  rm -rf web/static/assets/*; \
		  cp -r frontend/ui-react/dist/assets/* web/static/assets/; \
		  mkdir -p web/templates; \
		  cp frontend/ui-react/dist/index.html web/templates/index.html; \
		  echo "✅ Assets synced to web/static/assets"; \
	    fi; \
	  else \
	    echo "⚠️  No package.json in frontend/ui-react - skipping JS bundling"; \
	  fi; \
	fi

js-bundle: js-build ## Alias (deprecated) - use 'make js-build'
	@true
	@echo "  make compose-logs          # Tail logs for Docker Compose stack"
	@echo "  make bench                # Run focused benchmarks (set B=Regex to filter)"
	@echo "  make verify-csp           # Run CSP verification script (scripts/verify_csp.sh)"
verify-sse: ## Verify SSE log stream contract (server must be running on :8080)
	@echo "📡 Verifying SSE log stream..."; \
	if [ ! -x scripts/test_sse.sh ]; then chmod +x scripts/test_sse.sh; fi; \
	scripts/test_sse.sh || (echo "❌ SSE verification failed" && exit 1); \
	echo "✅ SSE verification passed"
## Docker Compose stack management
.PHONY: compose-build compose-up compose-down compose-logs

compose-build: ## Build Docker Compose stack (full demo)
	docker compose -f deployments/docker/docker-compose.yml build

compose-up: ## Start Docker Compose stack (full demo)
	docker compose -f deployments/docker/docker-compose.yml up -d

compose-down: ## Stop Docker Compose stack
	docker compose -f deployments/docker/docker-compose.yml down

compose-logs: ## Tail logs for Docker Compose stack
	docker compose -f deployments/docker/docker-compose.yml logs -f
web-integration-test: ## Run integration tests for web demo (integration tag)
	@echo "🧪 Running web demo integration tests..."
	$(GOTEST) -v -tags=integration ./test/integration/...
web-health: ## Health check for web demo (scripts/health.sh)
	@echo "🩺 Checking web demo health..."
	./scripts/health.sh

web-tail-logs: ## Tail web demo logs using scripts/tail-logs.sh
	@echo "📜 Tailing web demo logs (Ctrl+C to stop)..."
	./scripts/tail-logs.sh
## Web demo management
.PHONY: web-start web-stop web-restart web-logs

web-start: ## Start the web demo using scripts/start-web-demo.sh
	@echo "🚦 Starting web demo..."
	./scripts/start-web-demo.sh

web-stop: ## Stop the web demo using scripts/stop-web-demo.sh
	@echo "🛑 Stopping web demo..."
	./scripts/stop-web-demo.sh

web-restart: ## Restart the web demo (stop then start)
	@echo "🔄 Restarting web demo..."
	-./scripts/stop-web-demo.sh || true
	./scripts/start-web-demo.sh

web-logs: ## Tail the web demo logs
	@echo "📜 Tailing web demo logs (Ctrl+C to stop)..."
	tail -f gauth-web.log

web-beta: ## Run embedded beta UI server (optionally set GAUTH_WEB_PORT)
	@echo "🧪 Starting beta web UI (Ctrl+C to stop)..."
	@if [ -z "$$GAUTH_WEB_PORT" ]; then \\
	  echo "(Tip) export GAUTH_WEB_PORT=9090 to change port"; \\
	fi; \\
	GAUTH_BETA=1 go run ./cmd/web-server

web-educational: web-beta ## (Deprecated) Alias for legacy educational UI target
	@echo "⚠️  'web-educational' is deprecated. Use 'make web-beta' instead." >&2

dev-web: ## Run web server directly (no build) with log redirection (GAUTH_WEB_PORT optional)
	@echo "🧪 Starting dev web server (Ctrl+C to stop)..."; \
	PORT=$${GAUTH_WEB_PORT:-8080}; \
	# Normalize to digits only then rely on main.go adding colon
	PORT=$$(echo $$PORT | sed 's/^://'); \
	LOG=$${GAUTH_DEV_LOG:-/tmp/gauth_web.log}; \
	[ -z "$$LOG" ] && LOG=/tmp/gauth_web.log; \
	echo "→ Port: $$PORT"; \
	echo "→ Log: $$LOG"; \
	GAUTH_WEB_PORT=$$PORT go run ./cmd/web-server > "$$LOG" 2>&1 & echo $$! > /tmp/gauth_web.pid; \
	echo "✅ Server PID $$(cat /tmp/gauth_web.pid) (logs: $$LOG)"; \
	echo "Tip: tail -f $$LOG";

dev-web-stop: ## Stop dev web server started via dev-web target
	@if [ -f /tmp/gauth_web.pid ]; then \
		PID=$$(cat /tmp/gauth_web.pid); \
		echo "🛑 Stopping dev web server PID $$PID"; \
		kill $$PID 2>/dev/null || echo "(process already gone)"; \
		rm -f /tmp/gauth_web.pid; \
	else \
		echo "No dev web PID file found"; \
	fi

dev-web-restart: ## Restart dev web server (stop then start)
	-$(MAKE) dev-web-stop || true
	$(MAKE) dev-web

## Benchmarking
.PHONY: bench
bench: ## Run benchmarks with memory stats (B=regex to filter, default '^Benchmark(Token|Rate|Sliding)')
	@echo "🚀 Running benchmarks..."; \
	PATTERN=$${B:-Benchmark(Token|Rate|Sliding)}; \
	$(GOTEST) -run=^$$ -bench=$$PATTERN -benchmem ./test/benchmarks -count=1 | tee bench.out; \
	grep -E "^Benchmark" bench.out > bench_summary.txt || true; \
	echo "📄 Raw output: bench.out"; echo "📝 Summary: bench_summary.txt"

fuzz-cbor: ## Fuzz CBOR codec for PoA (FUZZ_TIME=10s to extend)
	@echo "🧪 Fuzzing CBOR codec..."; \
	FUZZ_TIME=$${FUZZ_TIME:-5s}; \
	$(GOTEST) -run=^$ -fuzz=FuzzCBORCodec -fuzztime=$$FUZZ_TIME ./pkg/poa; \
	echo "✅ Fuzz run complete"

verify-csp: ## Run CSP verification script (fails if violations found) 
	@echo "🔐 Verifying Content Security Policy..."; \
	if [ ! -x scripts/verify_csp.sh ]; then chmod +x scripts/verify_csp.sh; fi; \
	scripts/verify_csp.sh http://localhost:8080/ || (echo "❌ CSP verification failed" && exit 1); \
	echo "✅ CSP verification passed"

openapi_coverage: ## Generate OpenAPI coverage report (fails if <100%)
	@echo "📘 Generating OpenAPI coverage report..."; \
	go run ./cmd/specgen -spec docs/openapi.yaml -out docs/openapi.coverage.json > /dev/null || exit 1; \
	RATIO=$$(awk -F ':' '/"covered_ratio"/ {gsub(/[ ,]/,""); print $$2}' docs/openapi.coverage.json); \
	if [ "$$RATIO" != "1" ]; then \
	  echo "❌ OpenAPI route coverage below 100% (ratio=$$RATIO). Update docs/openapi.yaml."; \
	  exit 1; \
	else \
	  echo "✅ OpenAPI route coverage 100% (ratio=$$RATIO)"; \
	fi; \
	if grep -q '"missing_paths": \[' docs/openapi.coverage.json; then \
	  echo "⚠️  Unexpected missing_paths array present"; \
	  exit 1; \
	fi

openapi_param_coverage: ## Enforce path parameter description coverage (fails if any path param lacks description)
	@echo "📝 Checking OpenAPI path parameter description coverage..."; \
	go run ./cmd/specgen -spec docs/openapi.yaml -out docs/openapi.coverage.json > /dev/null || exit 1; \
	PARAM_RATIO=$$(awk -F ':' '/"parameter_description_coverage"/ {gsub(/[ ,]/,"\n"); print $$2}' docs/openapi.coverage.json | head -n1); \
	MISSING_COUNT=$$(awk -F ':' '/"missing_param_descriptions"/ {getline;};'); \
	if grep -q '"missing_param_descriptions": \[' docs/openapi.coverage.json; then \
	  # Extract count of missing entries
	  COUNT=$$(jq '.missing_param_descriptions | length' docs/openapi.coverage.json); \
	  if [ "$$COUNT" -gt 0 ]; then \
	    echo "❌ Missing path parameter descriptions ($$COUNT). See docs/openapi.coverage.json"; \
	    jq '.missing_param_descriptions' docs/openapi.coverage.json; \
	    exit 1; \
	  fi; \
	fi; \
	if [ "$$PARAM_RATIO" != "1" ]; then \
	  echo "❌ Path parameter description coverage below 100% (ratio=$$PARAM_RATIO)"; \
	  exit 1; \
	else \
	  echo "✅ Path parameter description coverage 100%"; \
	fi

openapi_query_param_coverage: ## Enforce query parameter description coverage (fails if any query param lacks description)
	@echo "🔍 Checking OpenAPI query parameter description coverage..."; \
	go run ./cmd/specgen -spec docs/openapi.yaml -out docs/openapi.coverage.json > /dev/null || exit 1; \
	QUERY_PARAM_RATIO=$$(awk -F ':' '/"query_parameter_description_coverage"/ {gsub(/[ ,]/,"\n"); print $$2}' docs/openapi.coverage.json | head -n1); \
	if grep -q '"missing_query_param_descriptions": \[' docs/openapi.coverage.json; then \
	  COUNT=$$(jq '.missing_query_param_descriptions | length' docs/openapi.coverage.json); \
	  if [ "$$COUNT" -gt 0 ]; then \
	    echo "❌ Missing query parameter descriptions ($$COUNT). See docs/openapi.coverage.json"; \
	    jq '.missing_query_param_descriptions' docs/openapi.coverage.json; \
	    exit 1; \
	  fi; \
	fi; \
	if [ "$$QUERY_PARAM_RATIO" != "1" ]; then \
	  echo "❌ Query parameter description coverage below 100% (ratio=$$QUERY_PARAM_RATIO)"; \
	  exit 1; \
	else \
	  echo "✅ Query parameter description coverage 100%"; \
	fi

spec-contract: openapi_coverage openapi_param_coverage openapi_query_param_coverage ## Run all OpenAPI contract enforcement checks

openapi_schema_prop_coverage: ## Enforce schema property description coverage
	@echo "🧬 Checking schema property description coverage..."; \
	go run ./cmd/specgen -spec docs/openapi.yaml -out docs/openapi.coverage.json > /dev/null || exit 1; \
	SCHEMA_RATIO=$$(awk -F ':' '/"schema_property_description_coverage"/ {gsub(/[ ,]/,"\n"); print $$2}' docs/openapi.coverage.json | head -n1); \
	if grep -q '"missing_schema_prop_descriptions": \[' docs/openapi.coverage.json; then \
	  COUNT=$$(jq '.missing_schema_prop_descriptions | length' docs/openapi.coverage.json); \
	  if [ "$$COUNT" -gt 0 ]; then \
	    echo "❌ Missing schema property descriptions ($$COUNT)."; \
	    jq '.missing_schema_prop_descriptions' docs/openapi.coverage.json; \
	    exit 1; \
	  fi; \
	fi; \
	if [ "$$SCHEMA_RATIO" != "1" ]; then \
	  echo "❌ Schema property description coverage below 100% (ratio=$$SCHEMA_RATIO)"; \
	  exit 1; \
	else \
	  echo "✅ Schema property description coverage 100%"; \
	fi

spec-contract: openapi_schema_prop_coverage ## Extend spec-contract with schema property coverage

openapi_example_coverage: ## Enforce operation example coverage (each operation must have >=1 example in request or response)
	@echo "🧪 Checking operation example coverage..."; \
	go run ./cmd/specgen -spec docs/openapi.yaml -out docs/openapi.coverage.json > /dev/null || exit 1; \
	OP_RATIO=$$(awk -F ':' '/"operation_example_coverage"/ {gsub(/[ ,]/,"\n"); print $$2}' docs/openapi.coverage.json | head -n1); \
	if grep -q '"missing_operation_examples": \[' docs/openapi.coverage.json; then \
	  COUNT=$$(jq '.missing_operation_examples | length' docs/openapi.coverage.json); \
	  if [ "$$COUNT" -gt 0 ]; then \
	    echo "❌ Missing operation examples ($$COUNT)."; \
	    jq '.missing_operation_examples' docs/openapi.coverage.json; \
	    exit 1; \
	  fi; \
	fi; \
	if [ "$$OP_RATIO" != "1" ]; then \
	  echo "❌ Operation example coverage below 100% (ratio=$$OP_RATIO)"; \
	  exit 1; \
	else \
	  echo "✅ Operation example coverage 100%"; \
	fi

spec-contract: openapi_example_coverage ## Extend spec-contract with operation example coverage

openapi_error_example_coverage: ## Enforce error (4xx/5xx) response example coverage
	@echo "🚨 Checking error response example coverage..."; \
	go run ./cmd/specgen -spec docs/openapi.yaml -out docs/openapi.coverage.json > /dev/null || exit 1; \
	ERR_RATIO=$$(awk -F ':' '/"error_response_example_coverage"/ {gsub(/[ ,]/,"\n"); print $$2}' docs/openapi.coverage.json | head -n1); \
	if grep -q '"missing_error_response_examples": \[' docs/openapi.coverage.json; then \
	  COUNT=$$(jq '.missing_error_response_examples | length' docs/openapi.coverage.json); \
	  if [ "$$COUNT" -gt 0 ]; then \
	    echo "❌ Missing error response examples ($$COUNT)."; \
	    jq '.missing_error_response_examples' docs/openapi.coverage.json; \
	    exit 1; \
	  fi; \
	fi; \
	if [ "$$ERR_RATIO" != "1" ]; then \
	  echo "❌ Error response example coverage below 100% (ratio=$$ERR_RATIO)"; \
	  exit 1; \
	else \
	  echo "✅ Error response example coverage 100%"; \
	fi

spec-contract: openapi_error_example_coverage ## Extend spec-contract with error response example coverage