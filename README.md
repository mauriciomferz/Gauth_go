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
> **Last Updated:** 2025-12-01 (✅ 100% RFC compliance, OAuth 2.0 clarification, test stability improvements)

**✨ Latest Updates (Dec 1, 2025):**
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

**Frontend** (\`web/ui-react/\`)
- React 18 + TypeScript admin portal
- Material-UI components  
- Real-time WebSocket updates
- MCP server management UI

**Infrastructure**
- **Database**: PostgreSQL 15+ with comprehensive migration system
- **Caching**: Redis for session and policy caching
- **Monitoring**: Prometheus metrics, OpenTelemetry tracing
- **CI/CD**: GitHub Actions with automated testing and deployment

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
- **Go**: 1.21 or later
- **PostgreSQL**: 15+ 
- **Redis**: 7+ (optional, for caching)
- **Node.js**: 18+ (for frontend development)

### 1. Clone and Setup Database
\`\`\`bash
git clone https://github.com/mauriciomferz/Gauth_go.git
cd Gauth_go

# Start PostgreSQL with Docker
docker-compose -f docker-compose.database.yml up -d

# Run migrations
./setup-database.sh
\`\`\`

### 2. Start the Backend
\`\`\`bash
# Using VS Code tasks (recommended)
# Press Cmd+Shift+P -> "Tasks: Run Task" -> "Start GAuth Backend with JWT"

# Or manually with environment variables
GAUTH_RFC0111_ENABLED=1 \\
GAUTH_GAUTHPLUS_ENABLED=1 \\
DB_HOST=localhost \\
DB_PORT=5432 \\
DB_USER=gauth_admin \\
DB_PASSWORD=gauth_dev_password \\
DB_NAME=gauth \\
GAUTH_JWT_SIGNING_KEY=dev-secret-change-in-production \\
go run ./cmd/web-server
\`\`\`

Server starts on \`http://localhost:8080\`

### 3. Start the Frontend (Optional)
\`\`\`bash
cd web/ui-react
npm install
npm run dev
\`\`\`

Frontend starts on \`http://localhost:5173\`

### 4. Quick Test
\`\`\`bash
# Health check
curl http://localhost:8080/api/v1/health

# Discovery endpoint
curl http://localhost:8080/.well-known/gauth-configuration

# GAuth+ successor status (example)
curl http://localhost:8080/api/v1/gauthplus/successors/active/00000000-0000-0000-0000-000000000001
\`\`\`

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

## 🔧 Configuration

### Environment Variables
\`\`\`bash
# Core Settings
GAUTH_RFC0111_ENABLED=1              # Enable RFC-0111 compliance
GAUTH_GAUTHPLUS_ENABLED=1            # Enable GAuth+ features
GAUTH_USE_JWT_LIB=1                  # Use JWT library
GAUTH_DEV_INDEX=1                    # Enable development mode

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=gauth_admin
DB_PASSWORD=your-secure-password
DB_NAME=gauth
DB_SSLMODE=require                   # Use in production

# Security
GAUTH_JWT_SIGNING_KEY=change-this-in-production
JWT_SECRET=your-jwt-secret

# Optional: Redis
REDIS_HOST=localhost
REDIS_PORT=6379
\`\`\`

### VS Code Tasks
The project includes pre-configured VS Code tasks:
- **Start GAuth Backend with JWT** - Full backend with database
- **Start Vite Dev Server** - React frontend development
- **Start GAuth With Admin Handlers** - Backend with admin APIs

## 🧪 Testing

\`\`\`bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run specific package
go test -v ./pkg/gauth

# Run with coverage
go test -cover ./...

# Performance benchmarks
go test -bench=. ./pkg/authz
\`\`\`

**Test Statistics**:
- 689+ test cases
- 66 packages tested
- 100+ integration tests
- Race condition testing enabled

## 📊 Monitoring & Observability

### Metrics Endpoints
- \`/api/v1/beta/metrics\` - Prometheus metrics
- \`/api/v1/beta/metrics/revocation/auto-sign/prometheus\` - Revocation metrics
- \`/api/v1/health\` - Health check
- \`/api/v1/info\` - System information

### Supported Collectors
- **Prometheus** - Primary metrics collection
- **OpenTelemetry** - Distributed tracing
- **StatsD** - Alternative metrics backend

### Monitoring Features
- Decision metrics with action/resource/reason taxonomy
- Revocation transparency metrics
- Key rotation audit trails
- Policy evaluation provenance
- Latency histograms with SLO support

## 🚢 Deployment

### Docker
\`\`\`bash
# Build image
docker build -t gauth:latest .

# Run with Docker Compose
docker-compose up -d
\`\`\`

### Kubernetes
\`\`\`bash
# Deploy to staging
kubectl apply -f k8s/staging/

# Deploy monitoring stack
kubectl apply -f k8s-monitoring-stack.yaml
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
