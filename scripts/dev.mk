# dev.mk - Include-style Makefile fragment for local development.
# Usage: from repo root run `make -f scripts/dev.mk <target>` if root Makefile inaccessible.

.PHONY: web key-rotation smoke test fmt lint coverage help

web:
	go run ./cmd/web-server

key-rotation:
	go run ./examples/key_rotation

smoke:
	bash scripts/smoke_web.sh

# Run all tests
test:
	go test ./...

fmt:
	go fmt ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else go vet ./...; fi

coverage:
	go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | grep total:

help:
	@echo "Available dev.mk targets:"; \
	echo "  web           - Run beta web server"; \
	echo "  key-rotation  - Run key rotation example"; \
	echo "  smoke         - Smoke test web server"; \
	echo "  test          - Run all tests"; \
	echo "  fmt           - Format sources"; \
	echo "  lint          - Lint sources"; \
	echo "  coverage      - Show total coverage"; \
	echo "  help          - Show this help";
