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