---
title: Getting Started
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Getting Started with AgentAuth RFC Implementation

> Last Updated: 2025-12-30
> Status: Active

> NOTE: Beta demonstration only – NOT production ready. See `../DISCLAIMER.md` for missing security controls.

**🏗️ DEVELOPMENT PROTOTYPE** | **🏆 AAP-002 COMPLETE** | **🏢 OPEN SOURCE COMMUNITY**

**Copyright (c) 2025 AgentAuth Authors**
Licensed under MIT License

This guide will help you get started with the official AgentAuth implementation.

## 🚀 Quick Installation

### 1. **Install the Package**

```bash
go get github.com/mauriciomferz/AgentAuth
```

### 2. **Build and Test**
curl -N http://localhost:8080/api/v1/educational/examples/run/$JOB_ID/logs

# 🌐 Beta Web UI & Examples API (Optional Quick Exploration)

The repository includes a lightweight embedded beta web interface showcasing Power-of-Attorney (PoA) flows, example execution, and live log streaming via Server-Sent Events (SSE).

Run it directly:
```bash
go run ./cmd/web-server
# Visit http://localhost:8080 (set AGENTAUTH_WEB_PORT to override port)
```

Key endpoints:
| Endpoint | Purpose |
|----------|---------|
| `/index.html` | Full embedded UI (HTML + JS + CSS via go:embed) |
| `/api/v1/beta/examples/catalog` | List runnable examples |
| `/api/v1/beta/examples/run` (POST) | Start an example job (returns job_id) |
| `/api/v1/beta/examples/run/{id}/status` | Poll job status |
| `/api/v1/beta/examples/run/{id}/logs` | SSE stream of job logs |
| `/api/v1/beta/examples/run/jobs/{id}/cancel` (POST) | Cancel job |

Example workflow:
```bash
# List examples
curl -s http://localhost:8080/api/v1/beta/examples/catalog | jq

# Start one
JOB_ID=$(curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"id":"agentauth_protocol_basics:minimal_poa"}' \
    http://localhost:8080/api/v1/beta/examples/run | jq -r .job_id)

# Check status
curl -s http://localhost:8080/api/v1/beta/examples/run/$JOB_ID/status | jq

# Stream logs (SSE)
curl -N http://localhost:8080/api/v1/beta/examples/run/$JOB_ID/logs
```

Cancel (if still running):
```bash
curl -X POST http://localhost:8080/api/v1/beta/examples/run/jobs/$JOB_ID/cancel
```

Implementation notes:
- All assets embedded with `go:embed` (no external file dependency at runtime)
- CSP with per-request nonce restricts script execution
- Job manager is in-memory (ephemeral) and capacity-limited

---


```bash
# Clone the repository
git clone https://github.com/mauriciomferz/AgentAuth
cd AgentAuth

# Build the package
go build ./pkg/auth

# Run compliance tests
go run examples/aap_functional_test/main.go
```

## 🎯 **First AAP-001 Implementation**

### **Basic AgentAuth Authorization**

Create your first AAP-001 compliant authorization:

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/mauriciomferz/AgentAuth/pkg/auth"
)

func main() {
    // 1. Create RFC-compliant service
    service, err := auth.NewRFCCompliantService("my-company", "ai-authorization")
    if err != nil {
        panic(err)
    }
    
    // 2. Create basic PoA Definition
    poa := auth.PoADefinition{
        Principal: auth.Principal{
            Type:     auth.PrincipalTypeIndividual,
            Identity: "john_doe_ceo",
        },
        Client: auth.ClientAI{
            Type:              auth.ClientTypeAgent,
            Identity:          "my_ai_assistant",
            Version:           "1.0.0",
            OperationalStatus: "active",
        },
        ScopeDefinition: auth.ScopeDefinition{
            ApplicableSectors: []auth.IndustrySector{auth.SectorBusiness},
            ApplicableRegions: []auth.GeographicScope{
                {Type: auth.GeoTypeNational, Identifier: "US"},
            },
            AuthorizedActions: auth.AuthorizedActions{
                Decisions: []auth.DecisionType{auth.DecisionInformation},
            },
        },
        Requirements: auth.Requirements{
            ValidityPeriod: auth.ValidityPeriod{
                StartTime: time.Now(),
                EndTime:   time.Now().Add(30 * 24 * time.Hour), // 30 days
            },
            JurisdictionLaw: auth.JurisdictionLaw{
                GoverningLaw:       "US_Federal_Law",
                PlaceOfJurisdiction: "US",
            },
        },
    }
    
    // 3. Create AgentAuth request
    request := auth.AgentAuthRequest{
        ClientID:     "my_ai_assistant",
        ResponseType: "code",
        Scope:        []string{"information_sharing"},
        PowerType:    "data_management",
        PrincipalID:  "john_doe_ceo",
        AIAgentID:    "my_ai_assistant",
        Jurisdiction: "US",
        PoADefinition: poa,
    }
    
    // 4. Authorize with RFC validation
    response, err := service.AuthorizeAgentAuth(context.Background(), request)
    if err != nil {
        fmt.Printf("❌ Authorization failed: %v\n", err)
        return
    }
    
    fmt.Printf("✅ Authorization successful!\n")
    fmt.Printf("Authorization Code: %s\n", response.AuthorizationCode[:20]+"...")
    fmt.Printf("Compliance Level: %s\n", response.PoAValidation.ComplianceLevel)
    fmt.Printf("Legal Compliance: %v\n", response.LegalCompliance)
}
```

## 🏢 **Corporate Implementation Example**
cd examples/basic
go run main.go
```

This demonstrates:
- Authorization request and grant
- JWT token issuance
- Token validation
- Transaction processing

### 2. **Test Rate Limiting**

Run the rate limiting example:
```bash
cd examples/rate
go run main.go
```

Watch how different patterns affect the rate limits:
- Burst requests
- Steady traffic
- Multiple clients

### 3. **Explore Token Management**

Try the token management example:
```bash
cd examples/token
go run main.go
```

See how tokens are:
- Created and validated
- Stored and retrieved
- Automatically cleaned up

## 🔐 Capability Governance (Enforcement Optional)

To restrict sensitive actions (token issuance, delegation create/revoke, capability reload, audit export) you can enable capability enforcement:

1. Provide a versioned capability file (see `docs/examples/capabilities.v1.json`).
2. Set environment variable `AGENTAUTH_CAPABILITIES_PATH` to its absolute path.
3. Set `AGENTAUTH_CAPABILITY_ENFORCE=1` to activate enforcement.
4. (Optional) Reload at runtime via `POST /api/v1/beta/capabilities/reload`.

Discovery endpoint `/.well-known/agentauth-configuration` will expose:
- `capability_registry_schema_version`
- `capability_registry_hash` (canonical SHA256 for integrity monitoring)
- Current `capability_registry` and `action_capabilities`.

Audit capability decisions can be paginated and filtered via:
`GET /api/v1/audit/capabilities?limit=50&cursor=...&action=delegation_create&outcome=denied`

See architectural rationale in `docs/ADR-capability-governance.md`.

Recommended next hardening steps (future releases): external hash anchoring, multi-version negotiation, and fuzz/property tests for loader stability.

## 🧾 External Capability Anchor Receipt Persistence (Beta)

To enable tamper-evident persistence of external capability anchoring receipts (hash-chain):

1. Set `AGENTAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER` (e.g. `memory` or `tsa_stub`).
2. Provide a writable file path via `AGENTAUTH_CAP_EXTERNAL_ANCHOR_RECEIPT_PATH` (will be created if absent).
3. (Optional) Set `AGENTAUTH_CAP_EXTERNAL_ANCHOR_RECEIPT_VERIFY_INTERVAL` (seconds, default disabled) to run a background integrity verification loop.

Endpoints exposed when persistence is configured and at least one receipt exists:
| Endpoint | Purpose |
|----------|---------|
| `/api/v1/beta/capabilities/anchor/external/receipts/latest` | Latest persisted external anchor receipt with linkage fields. |
| `/api/v1/beta/capabilities/anchor/external/receipts` | Full receipt chain (chronological append-only). |
| `/api/v1/beta/capabilities/anchor/external/receipts/verify` | Chain integrity verification (reports `ok`, `mismatch`, or `empty`). |

Metrics (Prometheus):
| Metric | Meaning |
|--------|---------|
| `capability_external_anchor_receipts_integrity` | 1=ok, 0=mismatch, -1=unconfigured/empty |
| `capability_external_anchor_receipts_last_verify_age_seconds` | Seconds since last chain verification |
| `capability_external_anchor_receipts_total` | Count of persisted external anchor receipts |

Integrity Model:
- Each entry stores `prev_hash` (prior chain head) and `chain_hash = sha256(prev_hash || canonical_receipt_json)`.
- Verification recomputes and compares linkage sequentially; first divergence reported.
- Background verifier updates integrity gauge and last-verify-age gauge.

Alerting Suggestion:
- Alert if integrity gauge transitions 1→0 for two consecutive scrapes.
- Monitor verify age to ensure background loop operating (< 2x configured interval).

## Manual Testing

1. **Authentication Flow**
```bash
# Request a token
curl -X POST http://localhost:8080/auth \
  -d '{"username": "test", "password": "test123"}'

# Use the token
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/protected
```

2. **Rate Limiting**
```bash
# Make rapid requests to see rate limiting
for i in {1..10}; do
  curl -H "Authorization: Bearer <token>" \
    http://localhost:8080/protected
done
```

3. **Token Management**
```bash
# Create a token
curl -X POST http://localhost:8080/token/create

# Validate a token
curl -X POST http://localhost:8080/token/validate \
  -d '{"token": "<token>"}'

# Revoke a token
curl -X POST http://localhost:8080/token/revoke \
  -d '{"token": "<token>"}'
```

## Configuration Examples

1. **Basic Setup**
```go
auth := agentauth.New(agentauth.Config{
    AuthServerURL: "https://auth.example.com",
    ClientID:     "client-123",
    ClientSecret: "secret-456",
})
```

2. **With Rate Limiting**
```go
auth := agentauth.New(agentauth.Config{
    // ... basic config ...
    RateLimit: agentauth.RateLimitConfig{
        RequestsPerSecond: 100,
        WindowSize:       60,
        BurstSize:       10,
    },
})
```

3. **With Custom Token Store**
```go
auth := agentauth.New(agentauth.Config{
    // ... basic config ...
    TokenStore: myCustomStore,
})
```

## Monitoring

1. **Check Rate Limit Status**
```bash
curl http://localhost:8080/metrics | grep rate_limit
```

2. **View Token Statistics**
```bash
curl http://localhost:8080/metrics | grep token
```

3. **Monitor Authentication Events**
```bash
curl http://localhost:8080/metrics | grep auth
```

## Troubleshooting

1. **Rate Limit Issues**
- Check the current rate limit status
- Verify client identification
- Review window size settings

2. **Token Problems**
- Verify token format
- Check expiration times
- Confirm scope configuration

3. **Authentication Failures**
- Review credentials
- Check server connectivity
- Verify client configuration

## Next Steps

1. Read the [Development Guide](DEVELOPMENT.md) for implementation details
2. Explore the [API Documentation](pkg/agentauth/doc.go)
3. Try the [Advanced Examples](examples/advanced/)

## Community Resources

- GitHub Issues: Report bugs and request features
- Discussions: Ask questions and share ideas
- Wiki: Additional documentation and guides

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
