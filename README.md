---
title: GAuth 1.0 README
category: overview
status: beta-ready
lastUpdated: 2025-12-01
owners: core-maintainers
---
# GAuth 1.0 - Go Implementation

[![CI Build](https://github.com/mauriciomferz/Gauth_go/actions/workflows/ci.yml/badge.svg)](https://github.com/mauriciomferz/Gauth_go/actions/workflows/ci.yml)
[![Deploy Staging](https://github.com/mauriciomferz/Gauth_go/actions/workflows/deploy-staging.yml/badge.svg)](https://github.com/mauriciomferz/Gauth_go/actions/workflows/deploy-staging.yml)
[![Gap Matrix](docs/badges/gap-matrix.svg)](docs/GAP_MATRIX.auto.md)

## What is GAuth?

**GAuth is a comprehensive authorization framework implementing GiFo-RFC-0111/0115 specifications by the Gimel Foundation.**

### 🔴 NOT an OAuth 2.0 Server

**Important:** GAuth is **NOT** an OAuth 2.0 (RFC 6749) authorization server. While it shares some concepts with OAuth 2.0 (tokens, scopes, delegation), it implements a fundamentally different authorization model designed for **legal Power of Attorney delegation and AI governance**.

### Key Distinctions from OAuth 2.0

| Aspect | OAuth 2.0 (RFC 6749) | GAuth (GiFo-RFC-0111/0115) |
|--------|---------------------|---------------------------|
| **Standards Body** | IETF (Internet Engineering Task Force) | Gimel Foundation (Proprietary) |
| **Primary Purpose** | API access delegation for web/mobile apps | Legal Power of Attorney + AI governance authorization |
| **Core Use Case** | "Login with Google", third-party API access | Legal delegation chains, fiduciary duty compliance, AI capability assessments |
| **Token Type** | Access tokens, refresh tokens | Proof-of-Authorization (PoA) tokens with delegation chains |
| **Authorization Model** | Resource owner consent for client apps | Legal authority delegation with multi-level chains |
| **Compliance Focus** | OAuth 2.0 Security BCP (RFC 8252) | Multi-jurisdiction legal compliance, AI governance |

### What GAuth Adds Beyond OAuth 2.0

GAuth extends traditional authorization with legal and AI governance capabilities:

1. **📜 Legal Delegation Chains** - Multi-level Power of Attorney with cryptographic proof and revocation transparency
2. **🤖 AI Governance** - Capability assessments, risk-level enforcement, and fiduciary duty monitoring for AI agents
3. **✅ Dual Control Approvals** - Multi-signature threshold requirements for high-risk operations
4. **👥 Successor Management** - Authority transfer workflows with automated succession planning
5. **🌍 Multi-Jurisdiction Compliance** - 18+ country identity verification, commercial register integration
6. **🔐 Cryptographic Audit Trails** - Merkle trees, external anchoring, tamper-proof logging with legal validity
7. **🎯 Policy-Based Authorization** - Advanced ABAC/RBAC with Policy Decision/Information/Enforcement Points

### Interoperability with OAuth 2.0

While GAuth is not an OAuth 2.0 server, the [OAuth 2.0 Migration Feasibility Study](docs/OAUTH_2_MIGRATION_FEASIBILITY_STUDY.md) recommends a **hybrid approach** for organizations needing both:

- **Keep GAuth** for legal delegation, AI governance, and fiduciary capabilities
- **Add RFC 8693 Token Exchange** to enable OAuth 2.0 interoperability when needed
- **Use Case Split**: OAuth 2.0 for API access, GAuth for legal authorization and AI governance

---

> **🚀 BETA-READY** - Comprehensive security audit, extensive testing (689+ test cases), complete documentation. Suitable for testing and evaluation.
> **Last Updated:** 2025-12-03 (📁 Directory reorganization, PolicyStore interface implementation)

**✨ Latest Updates (Dec 3, 2025):**
- **🔐 Enhanced Signature Validation**: New extensible validator registry supports multiple concurrent signature algorithms (HS256, RS256, ES256) with detailed logging and Prometheus metrics.
- **📁 Directory Reorganization:** Frontend moved to `frontend/`, integration tests to `test/integration/` for better organization
- **🗄️ PolicyStore Interface:** Pluggable policy storage with in-memory and PostgreSQL implementations for production scalability
- **♻️ Cleaner Architecture:** Separation of concerns - backend (web/), frontend (frontend/), tests (test/)
- **📚 Migration Guide:** Comprehensive [DIRECTORY_REORGANIZATION.md](DIRECTORY_REORGANIZATION.md) documentation

**Previous Updates (Dec 1, 2025):**
- **✅ 100% RFC Compliance:** Achieved 45/45 RFC-0111/0115 conformance requirements (updated gap-matrix badge)
- **🔍 OAuth 2.0 Clarification:** Added comprehensive distinction - GAuth implements GiFo-RFC-0111/0115, NOT OAuth 2.0 RFC 6749
- **🧪 Test Stability:** Fixed probabilistic test tolerance in TestCachePressureGenerator for reliable CI builds
- **🚀 GAuth+ Features Activated:** All 27 advanced authorization endpoints now operational (Successor Management, Delegation Chains, Dual Control, Fiduciary Duty, AI Capabilities)
- **⚙️ Easy Activation:** Set `GAUTH_GAUTHPLUS_ENABLED=1` to enable all GAuth+ features (now included in default dev tasks)
- **📊 Production Ready:** 5 advanced features with database persistence, caching, and advisory/enforcement modes
- **🔐 Enhanced Auth API:** Refresh token support with 7-day expiration, improved JWT flow with userId, username, and role claims
- **📚 API Documentation:** Comprehensive API_REFERENCE.md with all authentication and MCP endpoints documented
- **✅ All Tests Passing:** 76 revocation race tests passing with race detector enabled (26.4s runtime)
- **🔧 Improved Error Handling:** Better validation and standardized error responses across MCP and auth endpoints
- **🔐 Login Tab Added:** Multi-step authentication with credentials and MFA verification (TOTP, SMS, Email)
- **🤖 MCP Tab Added:** Model Context Protocol server management with stdio, WebSocket, and HTTP-SSE transport support
- **Phase 2A Complete:** 11 backend API endpoints implemented and tested (PVP, Registry, PoA CRUD)
- **18-Country Identity Verification:** Complete identity connectors for US, DE, UK, FR, IT, ES, SE, NL, AE, SA, JP, AU, SG, KR, IN, NZ, BR, CA, MX, ZA, NG, KE

A complete Go implementation of the **GAuth authorization framework (GiFo-RFC-0111/0115)** with delegated legal authorization, proof-of-authorization tokens, AI governance, and comprehensive security features.

**Previous Updates (Nov 14, 2025):**
- CI/CD Hardening, Enhanced Protocol Flow (30 substeps), Authorization Chain Validation
- Extended JWT Tokens, PIP Integration, Commercial Register, Updated UI
## 🎯 Project Status

- **RFC Compliance**: 45/45 requirements (100%) ✅
- **Test Coverage**: 689+ test cases across 66 packages
- **Architecture**: Microservices-ready with PostgreSQL, Redis, monitoring
- **Frontend**: React-based admin portal with MCP integration
- **API**: RESTful + WebSocket with comprehensive OpenAPI 3.0 spec
- **Security**: Multi-algorithm crypto, key rotation, audit trails
- **Identity Verification**: 18+ country connectors (US, EU, UK, DE, FR, IT, ES, etc.)

## 🏗️ Architecture Overview

### Core Components

**Backend Services** (\`cmd/web-server/\`)
- **RFC-0111 Authorization Server**: Complete GAuth protocol implementation
- **GAuth+ Advanced Features**: AI governance, delegation chains, successor management
- **Model Context Protocol (MCP)**: AI agent integration layer
- **Admin APIs**: User management, policy configuration, system monitoring

**Package Structure** (\`pkg/\`)

| Package | Description |
|---------|-------------|
| `gauth/` | Core authorization engine (RFC-0111/0115) |
| `gauthplus/` | Advanced features (AI governance, fiduciary duty) |
| `authz/` | Policy Decision Point (PDP) with ABAC/RBAC |
| `crypto/` | Multi-algorithm signing (Ed25519, ES256, ES384, RS256) |
| `token/` | JWT/PASETO token service with refresh support |
| `revocation/` | Revocation transparency with Merkle trees |
| `audit/` | Immutable audit ledger with external anchoring |
| `mcp/` | Model Context Protocol server management |
| `compliance/` | Legal framework validators (GDPR, CCPA, HIPAA) |
| `verification/` | 18-country identity verification system |
| `database/` | PostgreSQL repositories and migrations |
\`\`\`

**Frontend** (\`frontend/ui-react/\`)
- React 18 + TypeScript admin portal
- Material-UI components  
- Real-time WebSocket updates
- MCP server management UI

**Infrastructure**
- **Database**: PostgreSQL 15+ with comprehensive migration system
- **Caching**: Redis for session and policy caching
- **Monitoring**: Prometheus metrics, OpenTelemetry tracing
- **CI/CD**: GitHub Actions with automated testing and deployment
- **Policy Storage**: Pluggable storage interface (in-memory for dev, PostgreSQL for production)

## 🚀 Key Features

### Authorization & Security
- ✅ **Multi-Algorithm Crypto**: Ed25519, ES256, ES384, ES512, RS256
- ✅ **Key Rotation**: Automated scheduling with Vault/KMS integration
- ✅ **JWT/PASETO Tokens**: Full RFC compliance with refresh tokens (7-day expiry)
- ✅ **Replay Protection**: Durable persistence with WAL and eviction policies
- ✅ **Revocation Transparency**: Merkle tree proofs with external anchoring
- ✅ **Audit Trails**: Cryptographic audit ledger with tamper detection

### Policy Engine
- ✅ **ABAC/RBAC**: Attribute and role-based access control
- ✅ **Policy Versioning**: Hash chain with integrity verification
- ✅ **Distributed PDP**: Cache invalidation with clustering support
- ✅ **Obligations & Advice**: Executor framework with metrics
- ✅ **Expression Engine**: 40+ functions with regex support

### GAuth+ Advanced Features
- ✅ **Successor Management**: AI agent takeover scenarios
- ✅ **Delegation Chains**: Multi-level depth limits and validation
- ✅ **Dual Control**: Multi-signature threshold requirements
- ✅ **Capability Assessment**: AI capability level enforcement (L1/L2/L3)
- ✅ **Fiduciary Duties**: Violation detection and blocking

### Identity & Compliance
- ✅ **18+ Country Identity Verification**: US, DE, UK, FR, IT, ES, SE, NL, AE, SA, JP, AU, SG, KR, IN, NZ, BR, CA, MX, ZA, NG, KE
- ✅ **Commercial Register Integration**: Multi-jurisdiction business verification
- ✅ **Legal Framework Validators**: GDPR, CCPA, HIPAA, PCI-DSS support
- ✅ **Compliance Attestation**: Evidence ingestion with dual-backend storage

### Developer Experience
- ✅ **Comprehensive API**: 100+ RESTful endpoints with OpenAPI 3.0 spec
- ✅ **Interactive Documentation**: Swagger UI and ReDoc interfaces
- ✅ **MCP Integration**: Model Context Protocol for AI agents
- ✅ **Admin Portal**: Full-featured React frontend
- ✅ **Development Tools**: Docker Compose, hot reload, seeding scripts

## 📦 Quick Start

### Prerequisites

| Requirement | Version | Purpose |
|-------------|---------|---------|
| **Go** | 1.21+ | Backend runtime |
| **PostgreSQL** | 15+ | Primary database |
| **Redis** | 7+ | Caching (optional) |
| **Node.js** | 18+ | Frontend development |

### Installation Steps

#### Step 1: Clone Repository

```bash
git clone https://github.com/mauriciomferz/Gauth_go.git
cd Gauth_go
```

#### Step 2: Setup Database

```bash
# Start PostgreSQL with Docker
docker-compose -f docker-compose.database.yml up -d

# Run migrations
./setup-database.sh
```

> ✅ **Database Ready**: PostgreSQL running on `localhost:5432`

#### Step 3: Start Backend

**Option A: Using VS Code Tasks** (Recommended)
```
Press Cmd+Shift+P → "Tasks: Run Task" → "Start GAuth Backend with JWT"
```

**Option B: Manual Start**
```bash
GAUTH_RFC0111_ENABLED=1 \
GAUTH_GAUTHPLUS_ENABLED=1 \
DB_HOST=localhost \
DB_PORT=5432 \
DB_USER=gauth_admin \
DB_PASSWORD=gauth_dev_password \
DB_NAME=gauth \
GAUTH_JWT_SIGNING_KEY=dev-secret-change-in-production \
go run ./cmd/web-server
```

> 🚀 **Backend Running**: http://localhost:8080

#### Step 4: Start Frontend (Optional)

```bash
cd web/ui-react
npm install
npm run dev
```

> 🎨 **Frontend Running**: http://localhost:5173

### Quick Verification

Test your installation with these commands:

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Discovery endpoint
curl http://localhost:8080/.well-known/gauth-configuration

# GAuth+ feature example
curl http://localhost:8080/api/v1/gauthplus/successors/active/00000000-0000-0000-0000-000000000001
```

**Expected Response**: JSON with `{"status": "ok"}` for health check

---

## 📚 Documentation

### Essential Guides
- **[Getting Started](docs/GETTING_STARTED.md)** - Installation and first steps
- **[API Reference](API_REFERENCE.md)** - Complete endpoint documentation
- **[Database Setup](docs/POSTGRESQL_SETUP.md)** - PostgreSQL configuration
- **[Security Guide](SECURITY.md)** - Security best practices

### Architecture & Design
- **[Architecture Overview](docs/ARCHITECTURE.md)** - System architecture
- **[RFC Compliance Matrix](docs/GAP_MATRIX.auto.md)** - 45/45 requirements coverage
- **[OAuth 2.0 Migration Study](docs/OAUTH_2_MIGRATION_FEASIBILITY_STUDY.md)** - Interoperability guidance
- **[Cryptography Implementation](docs/CRYPTOGRAPHY_IMPLEMENTATION.md)** - Crypto details

### Advanced Features
- **[GAuth+ README](GAUTHPLUS_README.md)** - AI governance features
- **[MCP Integration](docs/MCP_INTEGRATION_PLAN.md)** - Model Context Protocol
- **[Identity Verification](docs/US_IDENTITY_VERIFICATION_ARCHITECTURE.md)** - Multi-country verification
- **[Policy Engine](docs/POLICY_ENGINE.md)** - ABAC/RBAC policies

### Development
- **[Contributing Guide](CONTRIBUTORS.md)** - How to contribute
- **[Testing Guide](docs/TESTING.md)** - Test strategies
- **[Code Style](docs/CODE_STYLE.md)** - Coding standards
- **[Changelog](CHANGELOG.md)** - Version history

## �� Configuration

### Environment Variables

Configure GAuth using environment variables. Create a `.env` file or export them in your shell.

#### Core Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `GAUTH_RFC0111_ENABLED` | `0` | Enable RFC-0111 compliance mode |
| `GAUTH_GAUTHPLUS_ENABLED` | `0` | Enable GAuth+ advanced features |
| `GAUTH_USE_JWT_LIB` | `0` | Use JWT token library |
| `GAUTH_DEV_INDEX` | `0` | Enable development mode with debug endpoints |

#### Database Configuration

| Variable | Example | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host address |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `gauth_admin` | Database username |
| `DB_PASSWORD` | `***` | Database password (use secrets manager in production) |
| `DB_NAME` | `gauth` | Database name |
| `DB_SSLMODE` | `require` | SSL mode (`disable`, `require`, `verify-ca`, `verify-full`) |

> ⚠️ **Production**: Always use `DB_SSLMODE=require` or higher in production environments

#### Security Settings

| Variable | Example | Description |
|----------|---------|-------------|
| `GAUTH_JWT_SIGNING_KEY` | `***` | JWT signing key (min 32 chars, use secrets manager) |
| `JWT_SECRET` | `***` | Additional JWT secret for legacy support |

> 🔒 **Security**: Never commit secrets to version control. Use environment-specific secret management.

#### Optional: Redis Cache

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_HOST` | `localhost` | Redis host address |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | - | Redis password (if authentication enabled) |

#### Example Configuration

**Development** (`.env.development`)
```bash
GAUTH_RFC0111_ENABLED=1
GAUTH_GAUTHPLUS_ENABLED=1
GAUTH_USE_JWT_LIB=1
GAUTH_DEV_INDEX=1
DB_HOST=localhost
DB_PORT=5432
DB_USER=gauth_admin
DB_PASSWORD=gauth_dev_password
DB_NAME=gauth
DB_SSLMODE=disable
GAUTH_JWT_SIGNING_KEY=dev-secret-change-in-production
```

**Production** (`.env.production`)
```bash
GAUTH_RFC0111_ENABLED=1
GAUTH_GAUTHPLUS_ENABLED=1
GAUTH_USE_JWT_LIB=1
GAUTH_DEV_INDEX=0
DB_HOST=${DB_HOST}
DB_PORT=5432
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=gauth
DB_SSLMODE=require
GAUTH_JWT_SIGNING_KEY=${VAULT_JWT_KEY}
```

### VS Code Tasks

The project includes pre-configured tasks in `.vscode/tasks.json`:

| Task Name | Purpose | Environment |
|-----------|---------|-------------|
| **Start GAuth Backend with JWT** | Full backend server with database | Development |
| **Start Vite Dev Server** | React frontend development server | Development |
| **Start GAuth With Admin Handlers** | Backend with admin API endpoints | Development |

**Usage**: `Cmd+Shift+P` → `Tasks: Run Task` → Select task

### Configuration Files

| File | Purpose |
|------|---------|
| `.env.development` | Local development settings |
| `.env.production` | Production environment template |
| `.env.staging` | Staging environment settings |
| `config/` | YAML configuration files for advanced settings |


## 🧪 Testing

### Running Tests

| Test Type | Command | Description |
|-----------|---------|-------------|
| All Tests | `go test ./...` | Run complete test suite across all packages |
| Race Detection | `go test -race ./...` | Run tests with race condition detection |
| Specific Package | `go test -v ./pkg/gauth` | Run tests for a specific package with verbose output |
| Coverage Report | `go test -cover ./...` | Generate test coverage report |
| Benchmarks | `go test -bench=. ./pkg/authz` | Run performance benchmarks |

### Test Suite Statistics

| Metric | Value |
|--------|-------|
| Total Test Cases | 689+ |
| Packages Tested | 66 |
| Integration Tests | 100+ |
| Race Detection | ✅ Enabled |

### Test Categories

- **Unit Tests**: Core functionality and business logic validation
- **Integration Tests**: Component interaction and end-to-end workflows
- **RFC Compliance Tests**: GiFo-RFC-0111/0115 conformance validation
- **Performance Tests**: Benchmarks for authorization decisions and policy evaluation
- **Security Tests**: Cryptographic operations and token validation

### Test Organization

```bash
# Run tests by category
go test ./pkg/gauth/...        # Core authorization tests
go test ./pkg/gauthplus/...    # GAuth+ successor tests
go test ./pkg/authz/...        # Policy engine tests
go test ./pkg/crypto/...       # Cryptographic tests
go test ./pkg/compliance/...   # RFC compliance tests
```

> 💡 **Tip**: Use `-v` flag for verbose output and `-count=1` to disable test caching during development.


## 📊 Monitoring & Observability

### Metrics Endpoints
- \`/api/v1/beta/metrics\` - Prometheus metrics
- \`/api/v1/beta/metrics/revocation/auto-sign/prometheus\` - Revocation metrics
- \`/api/v1/health\` - Health check
- \`/api/v1/info\` - System information

### Supported Collectors
- **Prometheus** - Primary metrics collection
- **OpenTelemetry** - Distributed tracing

## 🚢 Deployment

### Docker Deployment

#### Quick Start

| Step | Command | Description |
|------|---------|-------------|
| Build Image | `docker build -t gauth:latest .` | Build production Docker image |
| Run Container | `docker run -p 8080:8080 gauth:latest` | Run standalone container |
| Docker Compose | `docker-compose up -d` | Start full stack with database |

#### Available Docker Compose Profiles

| Profile | File | Purpose |
|---------|------|---------|
| Development | `docker-compose.yml` | Full stack with hot-reload |
| Production | `docker-compose.production.yml` | Optimized production deployment |
| Database Only | `docker-compose.database.yml` | PostgreSQL for local development |

### Kubernetes Deployment

#### Deployment Commands

```bash
# Deploy to staging environment
kubectl apply -f k8s/staging/

# Deploy to production environment
kubectl apply -f k8s/production/

# Deploy monitoring stack
kubectl apply -f k8s-monitoring-stack.yaml

# Verify deployment status
kubectl rollout status deployment/gauth
```

#### Kubernetes Resources

| Resource | Location | Description |
|----------|----------|-------------|
| Deployments | `k8s/staging/` | Application deployment manifests |
| Services | `k8s/production/` | Service definitions and load balancers |
| ConfigMaps | `k8s/configmaps/` | Configuration data |
| Secrets | `k8s/secrets/` | Sensitive configuration (use sealed secrets) |
| Monitoring | `k8s-monitoring-stack.yaml` | Prometheus, Grafana, and alerting |

### Production Readiness Checklist

#### Security

- [ ] Generate and configure strong JWT signing keys (min 256-bit)
- [ ] Enable PostgreSQL SSL mode (`DB_SSLMODE=require` or higher)
- [ ] Integrate external secret management (HashiCorp Vault, AWS KMS, Azure Key Vault)
- [ ] Rotate secrets regularly using automated processes
- [ ] Review and apply `SECURITY.md` security guidelines
- [ ] Enable HTTPS/TLS for all external endpoints
- [ ] Configure secure CORS policies

#### Database

- [ ] Set up PostgreSQL automated backups (daily minimum)
- [ ] Configure point-in-time recovery (PITR)
- [ ] Test database restore procedures
- [ ] Enable connection pooling (e.g., PgBouncer)
- [ ] Monitor database performance and set up alerts
- [ ] Configure replication for high availability

#### Monitoring & Operations

- [ ] Deploy Prometheus for metrics collection
- [ ] Configure Grafana dashboards for GAuth metrics
- [ ] Set up log aggregation (ELK, Splunk, or CloudWatch)
- [ ] Configure alerting rules for critical metrics
- [ ] Enable distributed tracing with OpenTelemetry
- [ ] Set up synthetic monitoring for health checks
- [ ] Document runbooks for common operational scenarios

#### Reliability

- [ ] Test disaster recovery procedures (RPO/RTO targets)
- [ ] Configure horizontal pod autoscaling (HPA) in Kubernetes
- [ ] Set up health checks and readiness probes
- [ ] Implement circuit breakers for external dependencies
- [ ] Configure rate limiting and throttling
- [ ] Test failover scenarios
- [ ] Establish SLO/SLA targets and measure compliance

### Environment Variables for Production

Refer to the **🔧 Configuration** section for complete environment variable documentation. Critical production settings:

- `GAUTH_RFC0111_ENABLED=1` - Enable RFC-0111 compliance mode
- `DB_SSLMODE=require` - Force SSL/TLS for database connections
- `GAUTH_JWT_SIGNING_KEY` - Strong signing key (never commit to version control)
- `REDIS_HOST` - Redis cache host for session management (optional but recommended)

### Deployment Guides

For detailed deployment instructions, see:
- **Docker**: `DEPLOYMENT_GUIDE.md`
- **Kubernetes**: `docs/kubernetes-deployment.md`
- **Production**: `docs/production-deployment.md`
- **Security**: `SECURITY.md`


\`\`\`

### Production Checklist
- [ ] Use strong JWT signing keys
- [ ] Enable DB SSL mode
- [ ] Configure external secret management (Vault/KMS)
- [ ] Set up PostgreSQL backups
- [ ] Enable Prometheus monitoring
- [ ] Configure log aggregation
- [ ] Review SECURITY.md guidelines
- [ ] Test disaster recovery procedures

## 🤝 Contributing

We welcome contributions! Please see:
- [Contributing Guidelines](CONTRIBUTORS.md)
- [Code of Conduct](CONTRIBUTORS.md#code-of-conduct)
- [Development Guide](docs/DEVELOPMENT.md)

## 📝 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

**Copyright (c) 2025 Gimel Foundation gGmbH i.G.**

## 🔗 Links

- **Repository**: https://github.com/mauriciomferz/Gauth_go
- **Gimel Foundation**: https://www.GimelFoundation.com
- **Documentation**: https://mauriciomferz.github.io/Gauth_go/
- **Issue Tracker**: https://github.com/mauriciomferz/Gauth_go/issues

## 📞 Support

For questions or issues:
1. Check the [documentation](docs/)
2. Review [existing issues](https://github.com/mauriciomferz/Gauth_go/issues)
3. Open a [new issue](https://github.com/mauriciomferz/Gauth_go/issues/new)

---
**Last Comprehensive Update**: December 1, 2025
