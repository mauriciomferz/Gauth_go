---
title: Dockerfiles Summary and Consolidation Plan
category: build-artifacts-guide
status: active
lastUpdated: 2025-11-12
owners: platform-eng
source: internal
refreshCadence: quarterly
---

# Dockerfiles Summary & Consolidation Plan

This repository contains multiple Dockerfiles tailored for different build/runtime scenarios. This document classifies them, highlights differences, and proposes a consolidation path.

## Inventory
| File | Purpose | Build Style | CGO | Base Runtime | Assets Included | Multi-Arch Ready | Healthcheck | Non-root |
|------|---------|------------|-----|--------------|-----------------|------------------|-------------|----------|
| `Dockerfile` | Generic multi-stage prod (legacy gauth-server path) | Multi-stage | No (CGO_DISABLED=0 but static) | `alpine:3.18.4` | No web assets copied | amd64 | wget /health | Yes |
| `Dockerfile.production` | Full prod with templates/static & CGO (BLS) | Multi-stage | Yes | `alpine:3.19` | templates/static/static_ui | amd64 | binary flag | Yes |
| `Dockerfile.minimal` | Ultra-minimal scratch debug | Two-stage -> scratch | No | `scratch` | No (binary only) | amd64 | binary flag | Yes (uid) |
| `Dockerfile.simple` | Runtime only (pre-built binary) | Single-stage runtime | N/A | `alpine:3.18.4` | None | amd64 | curl /healthz | Yes |
| `Dockerfile.simple-prod` | Runtime with assets (pre-built) | Single-stage runtime | N/A | `alpine:3.19` | templates/static/static_ui | amd64 | binary flag | Yes |
| `Dockerfile.single-stage` | Combined build+run (debug) | Single-stage | Yes | golang:alpine | templates/static/static_ui (via copy .) | amd64 | binary flag | Yes |
| `Dockerfile.dev` | Hot reload (Air) dev container | Single-stage (builder) | No | golang:alpine | All source | amd64 | (dev) | Root (runs as dev) |
| `Dockerfile.mock` | Mock server minimal | Multi-stage | No | `alpine:3.19` | Mock binary only | amd64 | None | No |
| `Dockerfile.kind` | Kind cluster test image | Multi-stage | No | `alpine:3.19` | Binary only | amd64 | binary flag | No |
| `Dockerfile.local-arm64` | Native ARM64 CGO build | Multi-stage | Yes | `alpine:3.19` | templates/static/static_ui | arm64 | binary flag | Yes |
| `Dockerfile.ui` | React UI build + nginx serve | Multi-stage (node+nginx) | N/A | `nginx:alpine` | Built dist/ | amd64 | HTTP probe | N/A |
| `deployments/docker/Dockerfile.jwe` | JWE-focused variant | Multi-stage | No | `alpine:latest` | None | amd64 | wget /health | Yes |

## Overlaps & Redundancies
- Runtime-only variants: `Dockerfile.simple`, `Dockerfile.simple-prod`, `Dockerfile.simple-prod` (duplicate name appears twice in repo listing) overlap with `Dockerfile.production` and could be replaced by parameterized build args.
- Specialized architecture variants (`local-arm64`) could be replaced with a buildx multi-arch pipeline producing both amd64 and arm64 from `production`.
- `single-stage` and `minimal` partially duplicate goals (debug vs smallest surface). Keep one well-documented minimal path.
- `kind` differs only in static build flags (could use `minimal` plus ARG toggles).
- `mock` stands alone (different entrypoint) but might live under `examples/mock/` with its Dockerfile.
- JWE-specific `Dockerfile.jwe` differs only by env defaults; could be replaced by helm chart / env config.

## Proposed Consolidation (Phased)
1. Phase 1 (Low Risk):
   - Keep: `Dockerfile.production` (rename to `Dockerfile`), `Dockerfile.dev`, `Dockerfile.ui`, `Dockerfile.minimal`.
   - Deprecate: `simple`, `simple-prod`, `single-stage` (document migration), `kind` (use minimal + build args), `mock` (move to examples), `local-arm64` (replace with buildx), `Dockerfile.jwe` (use base image + env).
2. Phase 2 (Refinement):
   - Introduce build args in main Dockerfile:
     - `ARG ENABLE_CGO=1`
     - `ARG INCLUDE_UI=0` (to copy assets if available)
     - `ARG TARGETARCH` for multi-arch matrix.
   - Add make targets: `make image-prod`, `make image-minimal`, `make image-dev`.
3. Phase 3 (Security & Slimming):
   - Switch runtime to `distroless/static:nonroot` (if CGO not required) or `distroless/base` when CGO needed.
   - Add SBOM generation (e.g. `syft packages dir:`) in CI.
   - Add vulnerability scan (Grype/Trivy) quality gate.

## Immediate Actions Suggested
- Create buildx workflow producing: `gauth:{version}-{arch}`, `gauth:{version}` multi-arch manifest.
- Add `LABEL org.opencontainers.image.source`, `description`, `licenses` to production image.
- Standardize health endpoint to `/api/v1/beta/health` across all Dockerfiles.
- Replace duplicate pre-built binary Dockerfiles with docs instructing: `GOOS=linux GOARCH=$arch go build ...` then single runtime stage.

## Example Unified Production Dockerfile (Concept)
```dockerfile
# syntax=docker/dockerfile:1
ARG GO_VERSION=1.25.3
FROM golang:${GO_VERSION}-alpine AS build
ARG ENABLE_CGO=1
RUN apk add --no-cache gcc g++ musl-dev git ca-certificates tzdata
WORKDIR /src
COPY go.mod go.sum ./
ENV GOWORK=off
RUN go mod download
COPY . .
RUN CGO_ENABLED=${ENABLE_CGO} GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags="-w -s" -o /out/web-server ./cmd/web-server

FROM alpine:3.19 AS runtime
RUN apk add --no-cache ca-certificates tzdata libstdc++ libgcc
WORKDIR /app
COPY --from=build /out/web-server ./web-server
# Optional UI assets (only if built/copied externally)
COPY --from=build /src/web/templates ./web/templates
COPY --from=build /src/web/static ./web/static
COPY --from=build /src/web/static_ui ./web/static_ui
USER 1000:1000
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/app/web-server", "-healthcheck"]
ENTRYPOINT ["/app/web-server"]
```

## Decision Matrix
| Keep | Reason |
|------|--------|
| production (as unified) | Full-featured CGO+assets build |
| dev | Hot reload + tools |
| minimal | Troubleshooting & base image security diff |
| ui | Standalone static UI delivery via CDN/Edge |

| Deprecate | Rationale |
|-----------|-----------|
| simple / simple-prod | Pre-built binary path adds maintenance; unify via build args |
| single-stage | Superseded by unified Dockerfile with ARGs |
| kind | Use minimal + build args or Helm chart values |
| mock | Move to example to reduce root clutter |
| local-arm64 | Buildx multi-arch replaces |
| jwe | Use env vars + unified Dockerfile |

## Next Steps
1. Implement unified Dockerfile with build args.
2. Add CI matrix (linux/amd64, linux/arm64) using buildx.
3. Remove deprecated Dockerfiles in a breaking-change release (note in CHANGELOG).
4. Add SBOM + vulnerability scan to pipeline.
5. Publish image hardening guidelines.

---
*Generated automatically as part of cleanup initiative. Update when Dockerfile set changes.*
