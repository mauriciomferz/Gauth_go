# Staging Deployment & CI/CD Verification Plan

## 1. CI Pipeline Simulation
- [x] **Core Tests**: `go test -short -count=1 -timeout=15m -parallel=4 ./pkg/...`
- [x] **Internal Tests**: `go test -short -count=1 -timeout=15m -parallel=4 ./internal/...`
- [x] **Web Tests**: `go test -count=1 -timeout=20m -parallel=2 ./web/...` (No DB)
- [x] **Revocation Tests**: `go test -v -count=1 -timeout=5m ./pkg/revocation/...`
- [x] **Race Detection**: `go test -race -v -count=1 -timeout=10m ./pkg/revocation/...`
- [x] **Linting**: `golangci-lint run` (if available)

## 2. Build Verification
- [x] **Binary Build**: `go build -ldflags="-s -w" -o build/bin/gauth-server ./cmd/gauth-server`
- [x] **Docker Build**: `docker build -f Dockerfile.production -t gauth:staging .` (Dry run check)

## 3. Staging Configuration Check
- [x] Verify `deployments/k8s/staging` manifests exist.
- [x] Verify `docker-compose.staging.yml` (if applicable) or verify `deployments/docker/docker-compose.yml` works for local staging sim.

## 4. Gap Matrix Update
- [x] Add "CI/CD & Deployment" section to `GAP_MATRIX.md`.
- [x] Mark "Staging Deployment" as "Ready" in `production_readiness.md`.

