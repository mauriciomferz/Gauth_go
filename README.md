---
title: GAuth 1.0 README
category: overview
status: beta-ready
lastUpdated: 2025-11-19
owners: core-maintainers
---
# GAuth 1.0 - Go Implementation

[![CI Build](https://github.com/mauriciomferz/Gauth_go/actions/workflows/ci.yml/badge.svg)](https://github.com/mauriciomferz/Gauth_go/actions/workflows/ci.yml)
[![Deploy Staging](https://github.com/mauriciomferz/Gauth_go/actions/workflows/deploy-staging.yml/badge.svg)](https://github.com/mauriciomferz/Gauth_go/actions/workflows/deploy-staging.yml)
[![Gap Matrix](docs/badges/gap-matrix.svg)](docs/GAP_MATRIX.auto.md)

> **🚀 BETA-READY** - Comprehensive security audit, extensive testing (689+ test cases), complete documentation. Suitable for testing and evaluation.
> **Last Updated:** 2025-11-19 (📚 Documentation refresh, architecture updates, configuration accuracy improvements)

**✨ Latest Updates (Nov 29, 2025):**
- **🔐 Enhanced Auth API:** Refresh token support with 7-day expiration, improved JWT flow with userId, username, and role claims
- **📚 API Documentation:** Comprehensive API_REFERENCE.md with all authentication and MCP endpoints documented
- **✅ All Tests Passing:** 76 revocation race tests passing with race detector enabled (26.4s runtime)
- **🔧 Improved Error Handling:** Better validation and standardized error responses across MCP and auth endpoints
- **🔐 Login Tab Added:** Multi-step authentication with credentials and MFA verification (TOTP, SMS, Email)
- **🤖 MCP Tab Added:** Model Context Protocol server management with stdio, WebSocket, and HTTP-SSE transport support
- **Phase 2A Complete:** 11 backend API endpoints implemented and tested (PVP, Registry, PoA CRUD)
- **18-Country Identity Verification:** Complete identity connectors for US, DE, UK, FR, IT, ES, SE, NL, AE, SA, JP, AU, SG, KR, IN, NZ, BR, CA, MX, ZA, NG, KE

A complete Go implementation of the GAuth authorization framework (RFC 0111/0115) with delegated authorization, proof-of-authorization tokens, and comprehensive security features.

**Previous Updates (Nov 14, 2025):**
- CI/CD Hardening, Enhanced Protocol Flow (30 substeps), Authorization Chain Validation
- Extended JWT Tokens, PIP Integration, Commercial Register, Updated UI

> **Quick Dev Start**
> ```bash
> # Backend (IMPORTANT: Set GAUTH_RFC0111_ENABLED=1 for Phase 2A endpoints)
> GAUTH_DEV_INDEX=1 GAUTH_RFC0111_ENABLED=1 GAUTH_USE_JWT_LIB=1 go run ./cmd/web-server
> # Frontend (React)
> cd web/ui-react && npm install && npm run dev
> ```
> **Access Points:**
> - 🌐 **API Server:** http://localhost:8080
> - ⚛️ **React UI:** http://localhost:3000 (full SPA with routing)
> - 📄 **Static HTML UI:** http://localhost:8080/gauth1.html (includes Login & MCP tabs)
> - 📚 **API Documentation:**
>   - **Swagger UI:** http://localhost:8080/api/docs/swagger
>   - **ReDoc:** http://localhost:8080/api/docs/redoc
>   - **Default:** http://localhost:8080/api/docs
> - 📋 **OpenAPI Spec:** http://localhost:8080/api/openapi/gauth.yaml
>
> **Features:** RFC-0111 8-step subscription wizard, Login with MFA (refresh tokens), MCP server management, 11 Phase 2A backend endpoints (PVP, Registry, PoA)
> 
> **📖 API Reference:** See [API_REFERENCE.md](API_REFERENCE.md) for complete API documentation with examples

**Status**: Beta (November 29, 2025)
- ✅ **CI/CD:** Resilient workflows with graceful degradation for missing infrastructure
- ✅ **Security:** All HIGH severity issues resolved, 215 gosec findings documented
- ✅ **Testing:** 49-97% coverage across core packages (689+ test cases)
- ✅ **Dependencies:** All current, zero known CVEs
- ✅ **Documentation:** 1,633+ lines (7 packages + architecture)
- ✅ **UI Features:** Login with MFA, MCP server management, RFC-0111 8-step subscription wizard

See [Dockerfiles Summary](DOCKERFILES_SUMMARY.md) for container build details. See also `CODEOWNERS` for ownership mapping.

## 🎉 100% RFC Conformance Achieved

**Status**: 45/45 requirements implemented with enhanced protocol flow (November 2025)

- ✅ P0 (Critical): 11/11 complete (100%)
- ✅ P1 (High): 10/10 complete (100%)
- ✅ P2 (Medium): 19/19 complete (100%)
- ✅ P3 (Low): 5/5 complete (100%)

**Enhanced Protocol Implementation (30 substeps):**
- **Subscription**: 3 substeps (registration, configuration, credentials)
- **Matching**: 7 substeps - ✨ Added: Authorization Chain, Commercial Register, Formal Requirements
- **Subset/Request**: 7 substeps - ✨ Added: Request Compliance, PIP Query, Extended Tokens, Grant Compliance
- **Enforcement**: 4 substeps (supply PEP, demand PEP, disclosure, audit)
- **Verification**: 6 substeps - ✨ Added: Extended Token validation, PVP Identity, Authorization Chain verify
- **Audit**: 3 substeps (logging, compliance check, reporting)

See [Gap Matrix](docs/GAP_MATRIX.auto.md) for detailed implementation status.

## 🎯 Quality Status

**GAuth 1.0 is in beta** with comprehensive quality assurance:

### Security ✅
- All HIGH severity issues resolved (215 gosec findings documented, non-blocking)
- Zero known CVEs in production dependencies
- Cryptographic operations use crypto/rand
- Integer overflow protection on critical paths
- CI/CD pipelines hardened with continue-on-error patterns
- Full security policy: [SECURITY.md](SECURITY.md)

**🔒 IMPORTANT SECURITY NOTICE (CV-2025-005):**
- **BoltDB DEPRECATED** for production containerized deployments (ephemeral storage vulnerability)
- **Redis REQUIRED** for production Kubernetes/Docker deployments
- See [REPLAY_STORE_MIGRATION_GUIDE.md](REPLAY_STORE_MIGRATION_GUIDE.md) for migration instructions
- See [SECURITY_AUDIT_CRITICAL_REVIEW.md](SECURITY_AUDIT_CRITICAL_REVIEW.md) for full details

### Testing ✅
- **pkg/auth**: 97.8% coverage (325 test cases)
- **pkg/authz**: 84.3% coverage (131+ test cases)
- **pkg/policy**: 76.9% coverage (33 test cases)
- **pkg/poa**: 49.1% coverage (200 test cases)
- Total: 689+ test cases across all packages
- All tests passing on every commit

### Documentation ✅
- 7 core packages with comprehensive godoc documentation (1,133 lines)
- Architecture documentation with system design: [ARCHITECTURE_SOLUTION.md](ARCHITECTURE_SOLUTION.md)
- 20+ runnable code examples
- Integration patterns and deployment models
- Performance characteristics documented

### Dependencies ✅
- All core dependencies on latest versions
- Zero deprecated dependencies
- Regular security updates
- 6 packages updated in latest cycle

**See**: [Docs Index](docs/INDEX.md) for complete documentation navigation.

## Quick Start

### Prerequisites

- **Go 1.25.3 or later** (required for security patches - see [Security Advisory](#security-advisory))
- Make (optional)

### Installation / Local Run

```bash
# Clone the repository
git clone https://github.com/mauriciomferz/Gauth_go.git
cd Gauth_go

# Run tests
go test ./...

# Run the web server (with Phase 2A endpoints enabled)
GAUTH_DEV_INDEX=1 GAUTH_RFC0111_ENABLED=1 go run ./cmd/web-server

# Or build and run
go build -o bin/web-server ./cmd/web-server
GAUTH_DEV_INDEX=1 GAUTH_RFC0111_ENABLED=1 ./bin/web-server
```

Visit http://localhost:8080 (API) and http://localhost:3000 (UI) to explore the interactive system.

### Modern React Web Interface

The webapp provides a comprehensive UI for testing and demonstrating GAuth capabilities:

**Features:**
- 🔐 **Secure Login** - Multi-step authentication with credentials and MFA verification (TOTP, SMS)
- 🎯 **Token Management** - Create, validate, and manage authorization tokens with JWT/JWE support
- 👤 **Identity Verification (PVP)** - ✅ Real backend endpoint (`/beta/pvp/verify`)
- 🏢 **Registry Integration** - ✅ Real backend endpoints (`/beta/registry/*`)
- 🔐 **Authorization (PIP)** - Policy-based authorization with ABAC/RBAC support
- 📜 **Power of Attorney (PoA)** - ✅ Full CRUD API (`/beta/poa/*`) - create, validate, manage delegations
- 🤖 **MCP Integration** - Model Context Protocol server management with resource reading and tool execution
- 📊 **Real-time Metrics** - Live system performance, latency, cache statistics, and component health
- 🧪 **E2E Testing** - Integrated test suite with 13 automated tests and detailed reporting
- 📈 **Dynamic Analytics** - Interactive charts showing request volume and latency trends

**✨ Phase 2A Complete**: All UI mocks replaced with real backend APIs (11 endpoints)

**Quick Start:**
```bash
# Backend (Go server on port 8080) - REQUIRED: Set GAUTH_RFC0111_ENABLED=1
GAUTH_DEV_INDEX=1 GAUTH_RFC0111_ENABLED=1 go run ./cmd/web-server

# Frontend (React + Vite on port 3000)
cd web/ui-react
npm install
npm run dev
```

**Important**: Without `GAUTH_RFC0111_ENABLED=1`, Phase 2A endpoints will return 404. See [Phase 2A Quick Start](docs/PHASE_2A_QUICK_START.md).

**Expose to Internet (VS Code):**
1. Press `Cmd+J` (Mac) or `Ctrl+J` (Windows/Linux) to open bottom panel
2. Click **"PORTS"** tab
3. Click **"Forward a Port"** → Enter `3000`
4. Right-click port 3000 → **Port Visibility** → **Public**
5. Copy the generated URL (e.g., `https://xxx-3000.app.github.dev`)

### Unified Dev Environment (Backend + Frontend)

Use the helper script to start both services:

```bash
./scripts/dev-up.sh
```

Outputs backend & UI PIDs and log paths. Stop via:

```bash
kill $(pgrep -f web-server) $(pgrep -f vite)
```

Backend environment defaults are documented in `.env.backend.example` (copy and adjust for local needs).

## 🌐 Interactive Web Demo

A modern **React + TypeScript** web interface showcasing all GAuth capabilities with real-time testing:

- **Secure Login** - Multi-factor authentication with credentials and MFA verification (TOTP/SMS/Email)
- **Token Operations** - Create, validate, and manage tokens with scope-based authorization
- **Identity Verification (PVP)** - Mock identity verification with entity type detection
- **Registry Services** - Entity lookup with jurisdictional compliance checks
- **Authorization (PIP)** - Policy evaluation with geographic and sector-based rules
- **Power of Attorney** - Create and validate delegation chains with action restrictions
- **MCP Integration** - Model Context Protocol server management supporting stdio, WebSocket, and HTTP-SSE transports
- **Live Metrics** - Real-time system performance with auto-refresh (request/sec, latency, cache hit rate)
- **E2E Testing** - Automated test suite with 13 tests covering all major flows
- **Dynamic Analytics** - Interactive charts and component health monitoring

**Access:**
- React UI: http://localhost:3000 (run `npm run dev` in `web/ui-react/`)
- Static HTML UI: http://localhost:8080/gauth1.html (includes Login and MCP tabs)

## Core Features

### Authorization & Delegation
- **Delegated Authorization** - Chain of authority with proof-of-authorization tokens
- **Policy Decision Point (PDP)** - Advanced policy evaluation engine with ABAC/RBAC
- **Multi-signature Support** - Threshold signatures for delegation chains
- **Revocation Transparency** - Merkle tree-based revocation with inclusion proofs

### Security & Cryptography
- **Multi-Algorithm Support** - EdDSA, ECDSA (P-256/P-384/P-521), RSA
- **Key Rotation** - Automated rotation with audit trails and multi-tenant support
- **External Anchoring** - Timestamp authority integration for audit logs
- **Replay Protection** - BoltDB-based replay attack prevention

### Compliance & Auditing
- **Attestation Store** - Evidence ingestion for compliance proof
- **Audit Ledger** - Cryptographic audit trail with external anchoring
- **Jurisdiction Support** - Multi-jurisdiction compliance (US/EU/UK/CA)
- **AI Governance** - AI capability matrix with risk-level enforcement

### Developer Experience
- **Interactive Web UI** - Live token creation, validation, and revocation with React SPA and static HTML
- **Secure Authentication** - Multi-step login with MFA support (TOTP, SMS, Email)
- **MCP Integration** - Model Context Protocol server management with multiple transport types
- **RFC-0111 Wizard** - 8-step subscription flow with visual progress tracking
- **OpenAPI Specification** - Complete API documentation
- **Comprehensive Testing** - Property tests, fuzz tests, integration tests (689+ test cases)
- **Observability** - Prometheus metrics, distributed tracing support

### Persistence & Scalability
- **PostgreSQL Integration** - Production-ready token persistence with JSONB storage
- **Token Store Abstraction** - Pluggable storage backends (memory, PostgreSQL)
- **Subscription Management** - Complete RFC-0111 subscription flow persistence
- **Docker Compose** - Ready-to-use development environment
- **Connection Pooling** - Optimized database connection management
- **Efficient Indexing** - Fast queries on authorization chains and metadata

**See**: [PostgreSQL Setup Guide](docs/POSTGRESQL_SETUP.md) for database configuration.

## Architecture

```
cmd/                    # 20+ executable commands
├── web-server/         # Main web demo server
├── conformance/        # RFC conformance testing
├── coverage/           # Test coverage tools
├── gauth-server/       # Standalone auth server
├── auditor/           # Audit log analysis
├── mock-server/       # Mock external services
└── ...                # See cmd/ directory for complete list

pkg/                    # 40+ packages organized by domain
# Core Authorization
├── gauth/             # Core authorization logic & PAP
├── auth/              # Authentication primitives
├── authz/             # Authorization engine
├── delegation/        # Delegation chain management
├── pdp/               # Policy decision point
├── pip/               # Policy information point
├── poa/               # Power of attorney
├── policy/            # Policy management

# Security & Compliance
├── crypto/            # Cryptographic operations
├── enforcement/       # Rule-based enforcement (NEW)
├── gagent/            # G-Agent API for AI (NEW)
├── compliance/        # Attestation & compliance
├── audit/             # Audit logging

# External Integration
├── oidc/              # OpenID Connect (NEW)
├── registry/          # Entity registry (NEW)
├── rfc0111/           # RFC-0111 implementation (NEW)
├── mcp/               # Model Context Protocol (NEW)

# Infrastructure
├── ledger/            # Audit ledger
├── notarization/      # External anchoring
├── cache/             # Caching layer
├── metrics/           # Metrics collection
├── events/            # Event system
├── replay/            # Replay protection
├── store/             # Storage abstraction
└── token/             # Token utilities

internal/              # 36 internal packages
├── anchor/            # Anchoring utilities
├── attestation/       # Attestation logic
├── authorization/     # Authorization internals
├── capability/        # Capability-based access
├── cascade/           # Cascade deletion
├── circuit/           # Circuit breaker
├── conditions/        # Condition evaluation
├── config/            # Configuration management
├── crypto/            # Internal crypto utilities
├── jurisdiction/      # Multi-jurisdiction support
├── metrics/           # Metrics internals
├── monitoring/        # Monitoring utilities
├── pdp/               # PDP cache & distributed support
├── policy/            # Policy internals
├── secrets/           # Secret management
├── security/          # Security utilities
└── ...               # See internal/ directory for complete list
```

## API Examples (Selected)

### Create a Delegation Token

```go
import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/delegation"

// Create delegation manager
mgr := delegation.NewManager(keyProvider, replayStore)

// Create delegation context
ctx := &delegation.DelegationContext{
    Subject:     "user:alice",
    Delegate:    "user:bob",
    Resource:    "document:123",
    Actions:     []string{"read", "write"},
    NotBefore:   time.Now(),
    NotAfter:    time.Now().Add(24 * time.Hour),
    Constraints: map[string]interface{}{
        "ip_range": "192.168.1.0/24",
    },
}

// Generate token
token, err := mgr.CreateDelegationCtx(ctx)
```

### Validate Authorization

```go
import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/pdp"

// Create PDP engine
engine := pdp.NewEngine(policyStore, keyProvider)

// Create authorization request
req := &pdp.Request{
    Subject:  "user:bob",
    Resource: "document:123",
    Action:   "read",
    Context: map[string]interface{}{
        "time":       time.Now(),
        "ip_address": "192.168.1.100",
    },
}

// Evaluate
result := engine.Evaluate(req, token)
if result.Decision == pdp.Allow {
    // Authorization granted
}
```

## Web Interface Features

The included web demo (`cmd/web-server`) provides:

- **Secure Login** - Multi-step authentication with MFA support (available in static HTML UI)
- **Token Management** - Create, validate, and revoke tokens with RFC-0111 8-step subscription wizard
- **Delegation Chains** - Visualize delegation hierarchies
- **Policy Testing** - Test policies against requests
- **MCP Integration** - Model Context Protocol server management (stdio/WebSocket/HTTP-SSE transports)
- **Metrics Dashboard** - Real-time authorization metrics
- **Audit Log Viewer** - Browse cryptographic audit trails
- **Key Rotation** - Manual and automatic key rotation controls

## CI/CD Configuration

The project includes production-ready CI/CD workflows with graceful degradation:

### GitHub Actions Workflows

**Main CI Pipeline** (`.github/workflows/ci.yml`):
- ✅ Go 1.25.4 with comprehensive testing (-race, -short flags)
- ✅ Security scanning with Gosec (215 findings documented, non-blocking)
- ✅ Dependency vulnerability checks with Trivy
- ✅ Docker image building with multi-platform support
- ✅ CodeQL security analysis with SARIF uploads

**Staging Deployment** (`.github/workflows/deploy-staging.yml`):
- ✅ Automated Docker builds with registry conditionals
- ✅ Kubernetes deployment with kubeconfig validation
- ✅ Health checks and smoke tests
- ✅ Slack notifications (optional)
- ✅ Automatic rollback on deployment failures

**Resilience Features:**
- Workflows continue on missing credentials (Docker, Kubernetes, Slack)
- Conditional step execution based on secret availability
- Non-blocking security scans with detailed reporting
- Comprehensive error handling with continue-on-error patterns

### Required Secrets (Optional)

For full CI/CD functionality, configure these secrets in GitHub:

```bash
# Docker Registry (optional - skipped if not configured)
DOCKER_REGISTRY=ghcr.io
DOCKER_USERNAME=your-username
DOCKER_PASSWORD=your-token

# Kubernetes Deployment (optional - skipped if not configured)
KUBE_CONFIG_STAGING=base64-encoded-kubeconfig

# Slack Notifications (optional - skipped if not configured)
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
```

**Note:** All workflows will run successfully even without these secrets configured.

## Configuration (Selected Environment Variables)

Configuration via environment variables:

```bash
# Core Server
GAUTH_PORT=8080
GAUTH_HOST=localhost

# CORS Configuration
GAUTH_CORS_ALLOW=http://localhost:3000,http://localhost:5173  # Development origins (restrict in production)

# Feature Flags
GAUTH_DEV_INDEX=1            # Enable development index page
GAUTH_RFC0111_ENABLED=1      # Enable RFC-0111 subscription endpoints (required for Phase 2A)
GAUTH_USE_JWT_LIB=1          # Use JWT library for token operations
GAUTH_BETA=1                 # Enable beta features

# Security & Tokens
GAUTH_JWT_SIGNING_KEY=dev-please-change    # JWT signing key (change in production)
GAUTH_ENABLE_REPLAY_PROTECTION=true
GAUTH_TOKEN_SIG_MODE=eddsa                 # Token signature algorithm

# Key Rotation
GAUTH_KEY_ROTATION_INTERVAL=720h
GAUTH_KEY_ROTATION_AUTO=true

# External Anchoring / Timestamp Authority
GAUTH_ANCHOR_ROTATIONS=false
GAUTH_TSA_ENDPOINT=                        # Optional TSA endpoint

# Observability
GAUTH_METRICS_ENABLED=true
GAUTH_METRICS_PERSIST_PATH=./data/metrics.db
GAUTH_TRACING_ENABLED=false
GAUTH_ENABLE_PROMETHEUS=true
GAUTH_ENABLE_OPENAPI=true

# Database (PostgreSQL - optional)
GAUTH_DB_HOST=localhost
GAUTH_DB_PORT=5432
GAUTH_DB_USER=gauth
GAUTH_DB_PASSWORD=dev-password-change-me
GAUTH_DB_NAME=gauth
```

**Note:** MFA and MCP features are implemented in the UI but environment variable configuration is handled internally. See `.env.backend.example` for complete configuration including database persistence.

## Testing (Backend & Frontend)

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run specific package
go test ./pkg/delegation

# Run conformance tests
go test ./conformance/...

# Generate coverage report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Frontend (from web/ui-react)
npm run test         # Run unit tests with Vitest
npm run test:unit    # Run unit tests once
npm run test:e2e     # Run Playwright end-to-end tests
npm run test:e2e:ui  # Run E2E tests with UI mode
```

## Performance

- **Authorization Latency**: < 5ms (p95)
- **Token Creation**: < 10ms (p95)
- **Revocation Check**: < 2ms (p95)
- **Throughput**: > 10,000 requests/sec (single instance)

## Documentation & Index

- [Gap Matrix](docs/GAP_MATRIX.auto.md) - RFC compliance tracking
- [API Documentation](api/openapi/gauth-api.yaml) - OpenAPI specification
- [P*P Architecture User Guide](docs/P_STAR_P_USER_GUIDE.md) - Complete guide to Policy Administration, Decision, Information, Verification, and Enforcement Points
- [Threat Model](docs/THREAT_MITIGATIONS_MATRIX.yaml) - Security analysis
- [Architecture](ARCHITECTURE_SOLUTION.md) - System architecture
- [Docs Index](docs/INDEX.md) - Curated navigation of gap, compliance & performance reports
- [Contributing](CONTRIBUTORS.md) - Contribution guidelines
- [Security](SECURITY.md) - Security policy
- [Disclaimer](DISCLAIMER.md) - Usage limitations

## Project Status

This implementation achieves **100% conformance** with RFC 0111/0115 specifications (45/45 requirements implemented):

- ✅ All core authorization primitives
- ✅ Complete delegation chain support
- ✅ Full cryptographic audit trails
- ✅ Multi-jurisdiction compliance
- ✅ AI governance capabilities
- ✅ Production-grade security patterns

## Security Advisory

**CRITICAL**: Go 1.25.3+ Required

This project requires **Go 1.25.3 or later** to address 8 known vulnerabilities in the Go standard library (Go 1.25.1):
- 2 HIGH severity DoS vulnerabilities (crypto/x509 panic, net/http memory exhaustion)
- 5 MEDIUM severity issues (ASN.1/PEM parsing, IPv6 validation, x509 constraints)
- 1 LOW severity information disclosure

All vulnerabilities are patched in Go 1.25.3. See [Pre-Production Audit Report](artifacts/preproduction_audit_week1_day2.md) for details.

**Before deploying**:
```bash
# Verify Go version
go version  # Must show go1.25.3 or higher

# Run vulnerability check
govulncheck ./...  # Must show 0 vulnerabilities
```

## Dockerfile Variants

Multiple Dockerfiles exist for targeted scenarios:
- `Dockerfile` (default multi-stage)
- `Dockerfile.production` (optimized production image)
- `Dockerfile.minimal` (static, smallest surface)
- `Dockerfile.simple` / `Dockerfile.simple-prod` (reference builds)
- `Dockerfile.local-arm64` (ARM64 dev convenience)
- `Dockerfile.single-stage` (debug simplicity)
- `Dockerfile.mock` (mock services instrumentation)

See `DOCKERFILES_SUMMARY.md` for consolidation guidance.

## License

See [LICENSE](LICENSE) for details.

## Disclaimer

**This is a beta demonstration implementation for learning and testing purposes only.**

Do NOT use for:
- ❌ Production systems
- ❌ Real security requirements
- ❌ Regulated data
- ❌ Commercial applications

See [DISCLAIMER.md](DISCLAIMER.md) for complete details on intentionally missing production safeguards.

## Contributing & Formatting

Contributions welcome! See [CONTRIBUTORS.md](CONTRIBUTORS.md) for guidelines.

### Development Tooling

- `make hygiene` – run tidy plus TODO report (fast combined hygiene)
- `make ci` – full CI suite locally (format check, vet, lint, race tests)
- `make test` – run all tests (CI-safe, without race detection)
- `make test-race` – run tests with race detection (for development)
- `make build` – build all binaries with adaptive path detection
- `scripts/dev-up.sh` – start backend + frontend together
- `make gap-matrix` – regenerate GAP_MATRIX.auto.md
- `make spec-contract` – enforce OpenAPI coverage/metadata completeness

For ownership clarity see [docs/STRUCTURE.md](docs/STRUCTURE.md) (proposed CODEOWNERS mapping).

## Acknowledgments & Future Hardening

Upcoming hardening focus:
- Distributed PDP cache invalidation & eviction metrics
- Attestation chain verification depth optimizations
- Metrics persistence compaction & Prometheus registry hygiene
- Structured error catalog & automated example coverage
- Fuzzing (CBOR codec, policy evaluator) & negative security tests

Built following the GAuth authorization framework specifications (RFC 0111/0115) by the Gimel Foundation. For structure overview and ownership, see [docs/STRUCTURE.md](docs/STRUCTURE.md) and `CODEOWNERS`.
