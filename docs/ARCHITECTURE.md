<!-- trunk-ignore-all(prettier) -->
---
title: AgentAuth 1.0 Architecture Documentation
category: architecture
status: active
lastUpdated: 2025-12-07
owners: architecture-team
source: manual-curation
refreshCadence: quarterly
---
# AgentAuth 1.0 - Architecture Documentation

**Version**: 1.0  
**Last Updated**: December 7, 2025  
**Status**: Production Ready

---

## Table of Contents

1. [Overview](#overview)
2. [System Architecture](#system-architecture)
3. [Web Handler Architecture](#web-handler-architecture)
4. [Core Components](#core-components)
5. [Data Flow](#data-flow)
6. [Security Architecture](#security-architecture)
7. [Scalability & Performance](#scalability--performance)
8. [Deployment Models](#deployment-models)
9. [Integration Patterns](#integration-patterns)

---

## Overview

AgentAuth 1.0 is a comprehensive authorization framework implementing AAP-001 (Core Authorization Protocol) and AAP-002 (Proof-of-Authorization Tokens). It provides:

- **Delegated Authorization** - Chain of authority with cryptographic proofs
- **Policy-Based Access Control** - ABAC, RBAC, and hybrid models
- **Multi-Signature Support** - Threshold signatures for high-security operations
- **Revocation Transparency** - Merkle tree-based revocation with inclusion proofs
- **Audit & Compliance** - Cryptographic audit trails with external anchoring

### Design Principles

1. **Security First** - All operations cryptographically verified
2. **Zero Trust** - Verify every authorization request
3. **Auditability** - Complete audit trail for all operations
4. **Extensibility** - Plugin architecture for custom policies
5. **Performance** - Sub-millisecond authorization decisions
6. **Standards Compliance** - AAP-001/0115 conformance

---

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Applications                      │
└────────────┬─────────────────────────────────────┬──────────────┘
             │                                     │
             ▼                                     ▼
┌────────────────────────┐            ┌────────────────────────┐
│   Policy Enforcement   │            │    Web Interface       │
│   Point (PEP)          │            │    (Demo/Admin)        │
└────────────┬───────────┘            └────────┬───────────────┘
             │                                 │
             │   Authorization Requests        │
             ▼                                 ▼
┌────────────────────────────────────────────────────────────────┐
│                      Core Services                             │
│                                                                │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────┐        │
│  │   Auth       │  │   Authz      │  │  Delegation    │        │
│  │ (pkg/auth)   │  │ (pkg/authz)  │  │(pkg/delegation)│        │
│  │              │  │              │  │                │        │
│  │ - Validate   │  │ - Policies   │  │ - Chains       │        │
│  │ - Verify     │  │ - PDP        │  │ - Constraints. │        │
│  │ - Extract    │  │ - Evaluate   │  │ - Revoke       │        │
│  └──────┬───────┘  └──────┬───────┘  └──────┬─────────┘        │
│         │                 │                 │                  │
│         └─────────────────┴─────────────────┘                  │
│                           │                                    │
│  ┌────────────────────────┴─────────────────────────┐          │
│  │              Core Auth Engine                    │          │
│  │              (pkg/auth)                          │          │
│  │  - POA Token Management (pkg/poa)                │          │
│  │  - Multi-Signature Validation                    │          │
│  │  - Revocation Management                         │          │
│  │  - Replay Protection                             │          │
│  └─────────────────────────────────────────────────┬┘          │
│                                                    │           │
└────────────────────────────────────────────────────┼───────────┘
                                                     │
                                                     │
                                                     │
                                                     │
                                                     ▼  
                    ┌──────────────────────────────────────────┐
                    │         Storage Layer                    │
                    │                                          │
                    │  ┌──────────────┐   ┌──────────────────┐ │
                    │  │ Policy Store │   │  Key Provider    │ │
                    │  │(pkg/policy)  │   │ (pkg/crypto)     │ │
                    │  └──────────────┘   └──────────────────┘ │
                    │                                          │ 
                    │  ┌──────────────┐   ┌──────────────────┐ │
                    │  │ Replay Store │   │ Revocation Store │ │
                    │  │(pkg/replay)  │   │  (Merkle Tree)   │ │
                    │  └──────────────┘   └──────────────────┘ │
                    │                                          │
                    │  ┌──────────────┐   ┌──────────────────┐ │
                    │  │ Audit Ledger │   │ Attestation Store│ │
                    │  │(pkg/ledger)  │   │ (pkg/compliance) │ │
                    │  └──────────────┘   └──────────────────┘ │
                    └──────────────────────────────────────────┘
```

### Component Layers

#### 1. Interface Layer
- **Web Interface**: Admin UI for token management, policy testing
- **API Layer**: REST/gRPC endpoints for programmatic access
- **PEP Integration**: Policy enforcement points in applications

#### 2. Authorization Layer
- **Authentication** (`pkg/auth`): Token validation, signature verification
- **Authorization** (`pkg/authz`): Policy evaluation, access control
- **Delegation** (`pkg/delegation`): Delegation chain management

#### 3. Core Engine Layer
- **AgentAuth Engine** (`pkg/agentauth`): Central orchestration
- **POA Management** (`pkg/poa`): Proof-of-authorization tokens
- **Policy Decision Point** (`pkg/pdp`): Advanced policy evaluation

#### 4. Storage Layer
- **Policy Store**: Policy persistence and retrieval
- **Key Provider**: Cryptographic key management
- **Replay Store**: Replay attack prevention
- **Revocation Store**: Token revocation tracking
- **Audit Ledger**: Cryptographic audit trail

---

## Web Handler Architecture

The web server (`web/server_clean.go`) has been modularized into dedicated handler packages for better separation of concerns, testability, and maintainability.

### Handler Package Structure

```
web/handlers/
├── admin/          # Admin portal authentication
├── anchor/         # External anchoring operations
├── audit/          # Audit trail API (entries, capabilities, stream)
├── auth/           # Frontend authentication endpoints
├── authz/          # Authorization API (evaluation, metrics, decisions)
├── beta/           # Beta feature handlers
├── capability_anchor/ # Capability registry anchoring
├── delegation/       # Delegation lifecycle/status handlers
├── events/           # Server-Sent Events (SSE) hub & stream handlers
├── lifecycle/        # Lifecycle event adapters & transition logic
├── mcp/            # Model Context Protocol handlers
├── modellimits/    # Model usage limits and quotas
├── notary/         # Notarization receipts and combined anchors
├── poa/            # Power of Attorney authorization
├── policy/         # Policy CRUD, chain, bundles, provenance
├── semantic/       # Semantic anomaly detection
├── token/          # Token store and JWKS endpoints
└── violations/     # Violation tracking and metrics
```

### Handler Interface Pattern

Each handler package follows a consistent pattern:

```go
// 1. Define dependencies via interfaces (for testability)
type Deps interface {
    GetStore() StoreProvider
    GetMetrics() MetricsProvider
}

// 2. Create handler struct
type Handler struct {
    deps Deps
    // ... fields
}

// 3. Constructor
func NewHandler(deps Deps) *Handler {
    return &Handler{deps: deps}
}

// 4. Route registration
func (h *Handler) RegisterRoutes(r *gin.Engine) {
    r.GET("/api/v1/resource", h.List)
    r.POST("/api/v1/resource", h.Create)
}

// 5. HTTP handlers
func (h *Handler) List(c *gin.Context) { ... }
func (h *Handler) Create(c *gin.Context) { ... }
```

### Key Handler Packages

#### Policy Handler (`web/handlers/policy`)
- **18 API methods** for policy management
- Chain-based version control with cryptographic hashes
- Bundle management (add, rollback, provenance)
- Prometheus metrics exposition

#### Audit Handler (`web/handlers/audit`)
- **5 API methods** for audit trail access
- `Entry` and `Provider` interfaces for abstraction
- SSE streaming for real-time audit events
- CSV export with pagination

#### Notary Handler (`web/handlers/notary`)
- **6 API methods** for notarization
- Combined anchor emission (capability + rotation digest)
- Receipt chain integrity verification
- Interface-based metrics delegation

#### Events Handler (`web/handlers/events`)
- **2 API methods** for event system
- In-memory pub/sub with `Hub` type
- SSE streaming for real-time events
- `HubProvider` interface for compatibility

### Metrics Integration

Handlers integrate with the metrics system via interface assertions:

```go
// Adapter pattern for metrics
type notaryMetricsAdapter struct {
    m metrics.Metrics
}

func (a *notaryMetricsAdapter) IncCombinedAnchorEmitted() {
    if inc, ok := a.m.(interface{ IncCombinedAnchorEmitted() }); ok {
        inc.IncCombinedAnchorEmitted()
    }
}
```


## Core Components

### 1. Authentication Service (`pkg/auth`)

**Responsibilities**:
- Token validation and signature verification
- Principal extraction
- Expiration checking
- Revocation status validation
- Replay attack prevention

**Test Coverage**: 97.8% ✅

**Key Algorithms Supported**:
- EdDSA (Ed25519) - Recommended
- ECDSA (P-256, P-384, P-521)
- RSA (2048, 3072, 4096 bits)

**Performance**:
- Token validation: <1ms
- Signature verification: 1-5ms (algorithm dependent)
- Revocation check: <1ms (cached)

### 2. Authorization Service (`pkg/authz`)

**Responsibilities**:
- Policy-based access control
- ABAC/RBAC evaluation
- Decision caching
- Obligation execution
- Pattern matching

**Test Coverage**: 84.3% ✅

**Policy Evaluation**:
```
Request → Target Matching → Condition Evaluation → Effect Determination
                                                           ↓
                                              Obligations & Advice
```

**Performance**:
- Simple policy: <1ms
- Complex ABAC policy: 1-5ms
- Cached decision: <0.1ms

### 3. Delegation Service (`pkg/delegation`)

**Responsibilities**:
- Delegation chain creation
- Chain validation
- Authority narrowing
- Constraint enforcement
- Revocation propagation

**Delegation Chain Example**:
```
Alice (Root Authority)
  │
  ├─ delegates to → Bob (Document:*, read/write)
  │                  │
  │                  └─ delegates to → Charlie (Document:123, read)
  │                                     │
  │                                     └─ Final Authorization
  │
  └─ Each hop: Signature verified, authority checked, constraints enforced
```

### 4. POA Token Management (`pkg/poa`)

**Responsibilities**:
- POA token issuance
- Multi-signature coordination
- CBOR encoding/decoding
- AAP-002 compliance validation
- Chain serialization

**Test Coverage**: 49.1% (practical limit) ✅

**Token Structure**:
```json
{
  "jti": "unique-token-id",
  "sub": "user:alice",
  "res": "document:123",
  "act": "read",
  "nbf": 1699564800,
  "exp": 1699651200,
  "chain": [
    {
      "delegator": "user:alice",
      "delegate": "user:bob",
      "signature": "...",
      "algorithm": "EdDSA"
    }
  ],
  "constraints": {
    "ip_range": "192.168.1.0/24"
  }
}
```

### 5. Policy Management (`pkg/policy`)

**Responsibilities**:
- Policy CRUD operations
- File-based persistence
- Policy validation
- Version management
- Registry coordination

**Test Coverage**: 76.9% ✅

**Policy Structure**:
```json
{
  "id": "read-documents",
  "version": 1,
  "effect": "Allow",
  "subjects": ["role:employee"],
  "resources": ["document:*"],
  "actions": ["read"],
  "conditions": [
    {
      "type": "time-range",
      "start": "09:00",
      "end": "17:00"
    }
  ],
  "obligations": [
    {
      "type": "audit-log",
      "level": "INFO"
    }
  ]
}
```

### 6. Key Management (`pkg/crypto`)

**Responsibilities**:
- Key generation and rotation
- Signing and verification
- Key storage and retrieval
- Algorithm management
- HSM integration

**Key Rotation**:
```
Current Key (Active)
  ↓ (rotation triggered)
New Key Generated
  ↓
Grace Period (both keys valid)
  ↓
Old Key Deprecated
  ↓
Old Key Archived (verification only)
```

### 7. Revocation Management

**Responsibilities**:
- Token revocation
- Merkle tree maintenance
- Inclusion proof generation
- Revocation transparency
- Batch updates

**Merkle Tree Structure**:
```
                    Root Hash
                    /       \
                  /           \
             H(AB)              H(CD)
            /    \              /    \
          /        \          /        \
      H(A)        H(B)      H(C)      H(D)
       |           |         |         |
    Token1      Token2    Token3    Token4
   (revoked)   (active)  (revoked) (active)
```

---

## Data Flow

### Authorization Request Flow

```
1. Client Request
   ↓
2. PEP Intercepts
   │
   ├─ Extract: Subject, Resource, Action, Context
   └─ Create Authorization Request
   ↓
3. Authentication (pkg/auth)
   │
   ├─ Validate Token Signature
   ├─ Check Expiration
   ├─ Verify Revocation Status
   └─ Prevent Replay Attacks
   ↓
4. Authorization (pkg/authz)
   │
   ├─ Retrieve Applicable Policies
   ├─ Evaluate Policy Conditions
   ├─ Apply Combining Algorithm
   └─ Collect Obligations
   ↓
5. Decision
   │
   ├─ Allow: Execute Obligations → Grant Access
   ├─ Deny: Log Decision → Reject Access
   └─ NotApplicable: Default Deny → Reject Access
   ↓
6. Audit
   │
   └─ Log Decision to Audit Ledger
```

### Delegation Creation Flow

```
1. Delegation Request
   │
   ├─ Delegator: user:alice
   ├─ Delegate: user:bob
   ├─ Resource: document:123
   └─ Actions: [read]
   ↓
2. Authority Check
   │
   ├─ Verify Alice has authority
   ├─ Check delegation constraints
   └─ Validate resource/action subset
   ↓
3. Token Generation
   │
   ├─ Create POA token
   ├─ Sign with Alice's key
   └─ Embed delegation chain
   ↓
4. Persistence
   │
   ├─ Store token metadata
   ├─ Update delegation registry
   └─ Log to audit trail
   ↓
5. Return Token
   └─ Bob can now use delegation token
```

### Revocation Propagation Flow

```
1. Revocation Request
   │
   └─ Token ID: abc123
   ↓
2. Revocation Manager
   │
   ├─ Mark token as revoked
   ├─ Update Merkle tree
   └─ Generate new root hash
   ↓
3. Chain Revocation
   │
   ├─ Find child delegations
   └─ Revoke entire subtree
   ↓
4. Cache Invalidation
   │
   ├─ Clear decision cache entries
   └─ Notify distributed caches
   ↓
5. Audit
   │
   └─ Log revocation event
```

---

## Security Architecture

### Defense in Depth

1. **Cryptographic Verification**
   - All tokens signed with asymmetric keys
   - Signature verification on every use
   - Support for HSM/KMS integration

2. **Replay Attack Prevention**
   - JTI (JWT ID) uniqueness enforcement
   - Time-bound token validity
   - Nonce-based challenge-response

3. **Revocation Transparency**
   - Merkle tree-based revocation
   - Inclusion proofs for verification
   - Real-time revocation checks

4. **Audit Trail**
   - Cryptographic audit ledger
   - External timestamp anchoring
   - Tamper-evident logging

5. **Zero Trust**
   - Verify every request
   - No implicit trust
   - Principle of least privilege

### Threat Model

| Threat | Mitigation |
|--------|------------|
| Token forgery | Asymmetric cryptography, signature verification |
| Replay attacks | JTI tracking, expiration, nonce |
| Man-in-the-middle | TLS encryption, certificate pinning |
| Privilege escalation | Authority narrowing, policy enforcement |
| Revocation bypass | Merkle tree proofs, real-time checks |
| Audit tampering | Cryptographic ledger, external anchoring |
| Key compromise | Key rotation, HSM storage, multi-sig |
| DoS attacks | Rate limiting, caching, timeouts |

---

## Scalability & Performance

### Performance Characteristics

| Operation | Latency | Throughput |
|-----------|---------|------------|
| Token validation | <1ms | 100K req/s |
| Policy evaluation | 1-5ms | 50K req/s |
| Signature verification | 1-5ms | 20K req/s |
| Revocation check | <1ms | 200K req/s (cached) |
| Delegation creation | 5-10ms | 10K req/s |

### Scaling Strategies

1. **Horizontal Scaling**
   - Stateless authorization services
   - Distributed caching (Redis)
   - Load balancing across instances

2. **Caching**
   - Decision caching (5-10 min TTL)
   - Public key caching
   - Policy compilation caching
   - Revocation status caching

3. **Database Optimization**
   - Indexed policy lookups
   - Read replicas for policy store
   - Write-optimized audit log
   - Partitioned revocation store

4. **Asynchronous Processing**
   - Background obligation execution
   - Batch audit log writes
   - Async revocation propagation

### Resource Requirements

**Small Deployment** (< 1K users):
- 2 CPU cores
- 4 GB RAM
- 20 GB storage

**Medium Deployment** (1K - 100K users):
- 8 CPU cores
- 16 GB RAM
- 200 GB storage
- Redis cache

**Large Deployment** (> 100K users):
- 32+ CPU cores (distributed)
- 64+ GB RAM (distributed)
- 1+ TB storage (distributed)
- Redis cluster
- CDN for policy distribution

---

## Deployment Models

### 1. Standalone Service

```
┌─────────────────────────────────┐
│      AgentAuth Standalone           │
│                                 │
│  - All-in-one binary            │
│  - SQLite/BoltDB storage        │
│  - Single instance              │
│  - Development/testing          │
└─────────────────────────────────┘
```

**Use Case**: Development, testing, small deployments

### 2. Microservices Architecture

```
┌──────────┐  ┌──────────┐  ┌──────────┐
│  Auth    │  │  Authz   │  │Delegation│
│ Service  │  │ Service  │  │ Service  │
└────┬─────┘  └────┬─────┘  └────┬─────┘
     │             │             │
     └─────────────┴─────────────┘
                   │
           ┌───────┴────────┐
           │   Shared DB    │
           └────────────────┘
```

**Use Case**: Production, high availability

### 3. Sidecar Pattern

```
┌─────────────────────────┐
│   Application Pod       │
│                         │
│  ┌─────────┐            │
│  │  App    │            │
│  └────┬────┘            │
│       │                 │
│  ┌────┴────┐            │
│  │ AgentAuth   │            │
│  │ Sidecar │            │
│  └─────────┘            │
└─────────────────────────┘
```

**Use Case**: Kubernetes, service mesh

---

## Integration Patterns

### 1. REST API Integration

```go
// PEP in application code
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    // Extract authorization token
    token := r.Header.Get("Authorization")
    
    // Call AgentAuth service
    resp, err := agentauthClient.Authorize(&AuthRequest{
        Token:    token,
        Resource: "/api/documents/123",
        Action:   "read",
        Context:  map[string]interface{}{
            "ip": r.RemoteAddr,
            "time": time.Now(),
        },
    })
    
    if err != nil || resp.Effect != "Allow" {
        http.Error(w, "Forbidden", 403)
        return
    }
    
    // Continue with authorized request
}
```

### 2. Middleware Integration

```go
// AgentAuth authorization middleware
func AuthzMiddleware(agentauthClient *agentauth.Client) func(next http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Authorize request
            if !authorize(r, agentauthClient) {
                http.Error(w, "Forbidden", 403)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### 3. Service Mesh Integration

```yaml
# Envoy filter for AgentAuth
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: agentauth-filter
spec:
  filters:
  - filterName: envoy.ext_authz
    filterConfig:
      grpc_service:
        envoy_grpc:
          cluster_name: agentauth-service
```

---

## Conclusion

AgentAuth 1.0 provides a comprehensive, production-ready authorization framework with:

- ✅ **Security**: Cryptographic verification, zero trust
- ✅ **Performance**: Sub-millisecond authorization decisions
- ✅ **Scalability**: Horizontal scaling, distributed caching
- ✅ **Compliance**: Audit trails, RFC conformance
- ✅ **Flexibility**: ABAC, RBAC, delegation chains

For implementation details, see package documentation in `pkg/`.
