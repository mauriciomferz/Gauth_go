# API Directory

> Last Updated: 2025-10-17
> Status: Active

This directory contains the API definitions and specifications for the GAuth implementation.

## Current Status: 🎓 BETA IMPLEMENTATION

> **⚠️ Beta Purpose Only**: This implementation is designed for learning and demonstration. It is NOT production ready. Do NOT use for real security, production, or commercial deployment.

The GAuth beta implementation provides a demonstration Go API through the `pkg/gauth` package for learning purposes.

## Directory Structure

```
api/
├── openapi/          # OpenAPI/Swagger specifications (placeholder)
├── proto/            # Protocol Buffer definitions (placeholder) 
└── README.md         # This file
```

authService, err := gauth.New(config)
authGrant, err := authService.InitiateAuthorization(authReq)
tokenResp, err := authService.RequestToken(tokenReq)
tokenStore := tokenstore.NewMemoryStore()
eventBus := events.NewBus()
eventBus.Subscribe("token.revoked", handler)
eventBus.Publish("token.created", eventData)
## Beta GAuth API

The beta GAuth API demonstrates authorization, token, tracing, metrics, and RFC compliance concepts in Go. Key surfaces:

### Core Service API
```go
import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"

cfg := gauth.Config{AuthServerURL: "https://auth.example.com", ClientID: "demo", ClientSecret: "secret"}
svc, err := gauth.New(cfg)

// Authorization grant issuance
grant, err := svc.InitiateAuthorization(gauth.AuthorizationRequest{ClientID: cfg.ClientID, Scopes: []string{"transaction:execute"}})

// Token request
tok, err := svc.RequestToken(gauth.TokenRequest{GrantID: grant.GrantID, Scope: grant.Scope, Restrictions: grant.Restrictions})

// Token validation now returns a structured result
vr, err := svc.ValidateToken(tok.Token)
if err != nil { /* handle error */ }
if vr != nil && vr.Valid { /* token ok */ }

// Revocation
_ = svc.InvalidateToken(tok.Token)
```

### Token Store (RFC‑0115 style features)
```go
import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/token"

store := token.NewMemoryStore()
ctx := context.Background()
t := &token.Token{ID: "id-1", Value: "opaque", ExpiresAt: time.Now().Add(time.Hour)}
_ = store.Save(ctx, t.ID, t)
got, _ := store.Get(ctx, t.ID)
_ = store.Revoke(ctx, t.ID, "demo-reason")
```

### Event System
```go
import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/events"

bus := events.NewBus()
bus.Subscribe("token.revoked", func(e events.Event){ /* ... */ })
bus.Publish("token.revoked", events.Event{Type: "token.revoked", Subject: "id-1"})
```

### Unified Rate Limiter API
All limiters expose a single method signature: `Allow(ctx context.Context, key string) error`.
```go
import rl "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/ratelimit"

lim := rl.WrapTokenBucket(&rl.Config{RequestsPerSecond: 100, WindowSize: 1, BurstSize: 20})
if err := lim.Allow(ctx, "user:123"); err != nil { /* rate limited */ }
```

### Tracing (Lightweight Demo Layer)
```go
import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/tracing"

tp, _ := tracing.NewTracerProvider(tracing.Config{ServiceName: "demo"})
ctx, span := tp.StartSpan(ctx, "operation", tracing.AttributeTransactionType.String("txn"))
defer span.End()
tracing.AddEvent(span, "step", tracing.Attribute{Key: "phase", Value: 1})
```

### Metrics Collector (Enhanced)
```go
import m "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/monitoring"
mc := m.NewMetricsCollector()
mc.IncrementWithLabels("transactions_total", map[string]string{"status": "success"})
mc.GaugeWithLabels("response_time_seconds", 0.123, map[string]string{"endpoint": "/token"})
```

### RFC Compliance Helpers
Lightweight summaries are exposed under `pkg/rfc`:
```go
info := rfc.GetComplianceInfo()
_ = rfc.ValidateCompliance("RFC-0111")

combined := rfc.CreateCombinedRFCConfig() // detailed combined config
_ = rfc.ValidateCombinedRFCConfig(combined)
```

Legacy example compatibility (official RFC‑0111 demo) uses `pkg/rfc0111.RFC0111Config` + `ValidateRFC0111Compliance` shim.

### PoA Definition (RFC‑0115 Demo)
```go
import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"

def := poa.PoADefinition{ /* fill Parties, Authorization, Requirements */ }
if err := poa.ValidatePoADefinition(def); err != nil { /* handle */ }
```

### Error Handling Compatibility
`pkg/errors` provides legacy code constants plus HTTP/status fields for bridging older examples. Use wrapped errors with additional context.

### Migration Notes (Recent Refactors)
| Area | Old Pattern | New Pattern |
|------|-------------|-------------|
| Rate Limiter | `ok := limiter.Allow()` | `err := limiter.Allow(ctx, key)` (nil => allowed) |
| Token Validation | `valid, err := ValidateToken(t)` | `res, err := ValidateToken(t)` (check `res.Valid`) |
| Token Store | `SaveToken/GetToken` | `Save/Get/Revoke(ctx,id,reason)` |
| RFC Config | Multiple scattered helpers | `rfc.CreateCombinedRFCConfig()` central source |
| Tracing | (absent / OTEL direct) | Lightweight `internal/tracing` shim + attributes + `AddEvent` |
| Metrics | Basic counters | Enhanced collector `IncrementWithLabels/GaugeWithLabels` |
| PoA Types | Missing / ad-hoc | `poa.PoADefinition` + `ValidatePoADefinition` |

If upgrading existing examples, adapt the signatures per the table above.

## Usage Examples

See the working examples in:
- `cmd/gauth-server/main.go` - Complete demo application ✅
- `examples/typed_structures_demo/` - Event system usage ✅
- `examples/token/advanced_revocation_flow/` - Token management ✅

## Testing

```bash
# Test the working API
cd examples/token/advanced_revocation_flow && go test -v  # PASSES ✅
cd examples/typed_structures_demo && go build            # BUILDS ✅
```

## Future Enhancements

While the current Go API is fully functional, future work may include:
- REST API endpoints over HTTP
- OpenAPI/Swagger specifications
- gRPC Protocol Buffer services
- GraphQL schema definitions
 - RFC 0111/0115 compliance maturation (see `docs/rfc0111_compliance_matrix.md`)

## Beta Use

This API is designed for:
- 🎓 **Learning environments** - Demonstration of GAuth concepts
- 📚 **Beta scenarios** - Understanding authorization flows
- 🔬 **Experimentation** - Testing ideas and patterns
- 🧪 **Prototype development** - Exploring extension possibilities

---

*Implementation by [Mauricio Fernandez](https://github.com/mauriciomferz)*

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md

### Compliance Summary (2025-10-25)
Recent security & integrity improvements:
1. Automatic multi-signature domain separation (threshold >1 triggers V2; binds sorted weights).
2. Embedded deterministic `weights` + `version` fields in canonical JSON (future evolution ready).
3. Strict authenticity default (missing signature key => integrity failure unless `GAUTH_STRICT_AUTHENTICITY=0`).
4. Mandatory `jti` claim baseline (replay fail-closed by default).

Status snapshot: Multi-signature threshold & canonical serialization Implemented; several authorization engine, lifecycle, and observability items remain Partial/Missing. Full matrix in `docs/rfc0111_compliance_matrix.md`.
