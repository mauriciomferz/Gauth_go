# Staging Deployment & CI/CD Verification Plan

## 1. CI Pipeline Simulation
- [ ] **Core Tests**: `go test -short -count=1 -timeout=15m -parallel=4 ./pkg/...`
- [ ] **Internal Tests**: `go test -short -count=1 -timeout=15m -parallel=4 ./internal/...`
- [ ] **Web Tests**: `go test -count=1 -timeout=20m -parallel=2 ./web/...` (No DB)
- [ ] **Revocation Tests**: `go test -v -count=1 -timeout=5m ./pkg/revocation/...`
- [ ] **Race Detection**: `go test -race -v -count=1 -timeout=10m ./pkg/revocation/...`
- [ ] **Linting**: `golangci-lint run` (if available)

## 2. Build Verification
- [ ] **Binary Build**: `go build -ldflags="-s -w" -o build/bin/gauth-server ./cmd/gauth-server`
- [ ] **Docker Build**: `docker build -f Dockerfile.production -t gauth:staging .` (Dry run check)

## 3. Staging Configuration Check
- [ ] Verify `deployments/k8s/staging` manifests exist.
- [ ] Verify `docker-compose.staging.yml` (if applicable) or verify `deployments/docker/docker-compose.yml` works for local staging sim.

## 4. Gap Matrix Update
- [ ] Add "CI/CD & Deployment" section to `GAP_MATRIX.md`.
- [ ] Mark "Staging Deployment" as "Ready" in `production_readiness.md`.
