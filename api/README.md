# API Directory

> Last Updated: 2025-11-05
> Status: Active - Enhanced with Complete OpenAPI Specification

This directory contains the API definitions and specifications for the GAuth implementation.

## Current Status: 🎓 BETA IMPLEMENTATION - Production-Grade Specifications

> **⚠️ Beta Purpose Only**: This implementation is designed for learning and demonstration. It is NOT production ready. Do NOT use for real security, production, or commercial deployment.

The GAuth beta implementation provides a demonstration Go API through the `pkg/gauth` package for learning purposes, now with complete OpenAPI specifications for all endpoints.

## Directory Structure

```
api/
├── openapi/          # Complete OpenAPI/Swagger specifications
│   └── openapi.yaml  # Full API documentation (issue, validate, status, delegation, metrics, provenance)
├── proto/            # Protocol Buffer definitions (placeholder) 
└── README.md         # This file
```

## Recent Updates (November 2025)

### OpenAPI Specification - Now Complete! ✅

The OpenAPI specification has been upgraded from Partial to **Implemented** status:

**Documented Endpoints:**
- ✅ POST `/api/v1/poa/issue` - Issue new Power of Attorney
- ✅ POST `/api/v1/beta/policy/evaluate` - Evaluate authorization policies
- ✅ GET `/api/v1/poa/status/{id}` - Get PoA delegation status
- ✅ POST `/api/v1/delegation/create` - Create delegation
- ✅ GET `/api/v1/metrics` - System metrics and observability
- ✅ GET `/api/v1/provenance` - Audit trail and provenance
- ✅ GET `/.well-known/gauth-configuration` - Discovery endpoint

**Documentation Quality:**
- Complete request/response schemas
- Error code documentation
- Authentication requirements
- Example payloads
- Status code specifications

**Locations:**
- `api/openapi/openapi.yaml` (comprehensive specification)
- `docs/openapi.yaml` (documentation copy)
- Inline annotations in `web/server_clean.go`

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

### Rotation Summary Artifact (Ledger Transparency & Multisig)
The endpoint `GET /api/v1/beta/rotations/summary` exposes an integrity artifact for Ed25519 key rotations.

JSON Shape (multisig enabled):
```jsonc
{
	"success": true,
	"configured": true,
	"generated_at": "2025-10-26T04:00:00.000000000Z",
	"summary": {
		"chain_length": 42,
		"head_hash": "sha256:...",          // hash of latest rotation record
		"aggregate_hash": "sha256:...",     // stable hash over entire rotation chain
		"generated_at": "RFC3339Nano",
		"kid": "ed25519:abcdef12",          // legacy single-signature KID (omitted when pure multisig mode)
		"signature": "base64url...",        // legacy single signature (omitted in pure multisig mode)
		"mode": "EdDSA",                    // signature mode (currently EdDSA only)
		"threshold": 2,                      // required signature quorum (only if GAUTH_ROTATIONS_MULTISIG=1)
		"satisfied_weight": 2,               // number of valid signatures collected
		"signatures": [                      // multisig entries
			{ "kid": "ed25519:abcdef12", "mode": "EdDSA", "signature": "base64url..." },
			{ "kid": "ed25519:98765432", "mode": "EdDSA", "signature": "base64url..." }
		]
	}
}
```

Legacy single-signature path (multisig disabled) includes only: `kid`, `signature`, `mode` and omits `threshold`, `satisfied_weight`, `signatures`.

Environment Flags:
| Flag | Purpose |
|------|---------|
| `GAUTH_ROTATIONS_SIGN=1` | Enable rotation summary signing. |
| `GAUTH_ROTATIONS_MULTISIG=1` | Aggregate signatures from all current Ed25519 keys. |
| `GAUTH_ROTATIONS_THRESHOLD` | Optional integer quorum; error if greater than available signatures. |

Error Codes (rotation summary path):
| Code | Condition |
|------|-----------|
| `rotation_chain_empty` | Ledger has no rotation entries. |
| `rotation_continuity_gap` | Hash continuity broken between entries. |
| `rotation_signature_registry_unavailable` | Signing required but EdDSA registry missing. |
| `rotation_signature_missing` | Signature fields absent in required single-sign path. |
| `rotation_signature_invalid` | Signature verification failure (legacy mode). |
| `rotation_threshold_unsatisfied` | Requested threshold exceeds available signatures. |
| `rotation_ledger_unavailable` | Rotation ledger not wired. |
| `rotation_ledger_type_mismatch` | Internal type assertion failed (should never occur in normal config). |

Client Verification:
Use `pkg/verification.VerifyRotationSummarySignature(sum)` which:
- Iterates all `signatures[]` if present; every must verify.
- Enforces `satisfied_weight >= threshold` when `threshold > 0`.
- Falls back to legacy single signature (`kid` + `signature`).

Canonical Signing Payload:
Only the minimal fields `{chain_length, head_hash, aggregate_hash, generated_at}` are serialized and prefixed with `GAUTH_ROTATION_SUMMARY:` before Ed25519 signing. This keeps signatures stable across future schema extensions.

Example Go verification snippet:
```go
resp, _ := verification.FetchRotationSummary(httpClient, baseURL)
if err := verification.VerifyRotationSummarySignature(resp.Summary); err != nil {
		// handle integrity failure (invalid signature, threshold unsatisfied, etc.)
}
```

Planned Extensions:
- Weighted signatures (governance tiers) – future `weights[]` field.
- Alternative algorithms (BLS) – potential `mode` variants.
- External anchoring of head hash – integrating with capability anchors.

### Model Limits Attestation Integrity & Signing

Endpoint: `GET /api/v1/model/limits/attestation` produces a governance attestation describing current model limits, audit chain head, optional anchor chain head, surge detection status, and strict unknown-model enforcement.

Signature (when `GAUTH_MODEL_LIMIT_ATTEST_SIGN=1`):
 - Includes `nonce` (random base64url) for replay protection; unique per signed attestation

Verification (`POST /api/v1/model/limits/attestation/verify`):
1. Reconstruct unsigned object preserving field order.
2. Prepend identical prefix `GAUTH_MODEL_LIMIT_ATTEST:`.
3. Verify Ed25519 signature using provided `sig_kid` key.
4. Return `combined_hash = sha256(attest|snapshot.hash|audit.head_hash|anchor.latest_hash)` for external anchoring / linkage.

Environment Flags:
| Flag | Purpose |
|------|---------|
| `GAUTH_MODEL_LIMIT_ATTEST_SIGN=1` | Enable attestation signing with domain-separated payload. |
| `GAUTH_MODEL_LIMIT_ATTEST_NOTARIZE=1` | Attach external notarization receipt over combined hash seed. |
| `GAUTH_ATTEST_STREAM_ENABLE=1` | Enable SSE stream of periodic attestation updates. |
| `GAUTH_ATTEST_NONCE_TTL=30m` | Optional TTL (Go duration string) to evict cached replay nonces (default 1h). |

Error Codes (attestation verify path):
| Code | Condition |
|------|-----------|
| `attestation_body_read_failed` | Request body unreadable. |
| `attestation_invalid_json` | JSON malformed. |
| `attestation_signature_fields_missing` | Missing signature mode/KID fields. |
| `attestation_key_registry_unavailable` | EdDSA key registry not wired. |
| `attestation_unknown_kid` | `sig_kid` not found. |
| `attestation_signature_base64_invalid` | Signature not valid base64url. |
| `attestation_signature_invalid` | Signature cryptographic verification failed. |
| `attestation_notarization_inconsistent` | Notarization block present but marked unsuccessful. |
| `attestation_nonce_missing` | Nonce absent in signed attestation. |
| `attestation_nonce_replay` | Nonce already observed (replay attempt). |

Canonical Signing Payload Stability:
The prefix ensures signatures cannot be replayed or confused with rotation summaries. Future schema extensions that add optional fields will not alter past signatures because optional fields are excluded when empty.

Example Verification (client-side pseudocode):
```go
rawUnsigned := rebuildUnsigned(att) // drop signature fields
msg := append([]byte("GAUTH_MODEL_LIMIT_ATTEST:"), rawUnsigned...)
ok := ed25519.Verify(pubKey, msg, sigBytes)
if nonceCache.Seen(att.Nonce) { /* reject replay */ }
```

Planned Attestation Extensions:

#### Observability Metrics (Rotation & Attestation Verification)

The beta surfaces Prometheus metrics for cryptographic integrity operations.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `gauth_rotation_signature_verify_latency_seconds` | histogram | (none) | Latency per rotation summary signature verification (each signature). |
| `gauth_rotation_signature_verify_failures_total` | counter | `reason` | Count of rotation signature verification failures by reason (e.g. `missing_signature`, `kid_mismatch`). |
| `gauth_attestation_verify_latency_seconds` | histogram | (none) | Latency of full attestation verification (reconstruction + signature + replay check). |
| `gauth_attestation_verify_failures_total` | counter | `reason` | Count of attestation verification failures by reason (e.g. `invalid_json`, `signature_invalid`, `nonce_replay`). |
| `gauth_attestation_verify_total` | counter | `outcome`, `soft_invalid` | Total attestation verification attempts classified by outcome (success|failure) and whether failure was soft-invalid. |
| `gauth_attestation_nonce_cache_size` | gauge | (none) | Current size of replay nonce cache after periodic TTL prune. |
| `attestation_domain_signature_failures_total` | counter | `reason` | Domain signature soft invalid verification failures (invalid, prefix_missing, base64_invalid). |
| `attestation_domain_signature_success_total` | counter | (none) | Count of attestations where optional domain signature verified successfully. |

Example PromQL dashboards:

```promql
# 95th percentile latency (5m window)
histogram_quantile(0.95, sum(rate(gauth_rotation_signature_verify_latency_seconds_bucket[5m])) by (le))
histogram_quantile(0.95, sum(rate(gauth_attestation_verify_latency_seconds_bucket[5m])) by (le))

# Failure ratios
sum(increase(gauth_rotation_signature_verify_failures_total[5m])) / sum(increase(gauth_rotation_signature_verify_latency_seconds_count[5m]))
sum(increase(gauth_attestation_verify_failures_total[5m])) / sum(increase(gauth_attestation_verify_latency_seconds_count[5m]))

# Top attestation failure reasons (5m)
topk(5, sum(increase(gauth_attestation_verify_failures_total[5m])) by (reason))
```

Operational Notes:
- A consistently elevated `nonce_replay` rate may indicate upstream caching / replay issues.
- Rotation signature failures for `serialization_error` imply schema drift between signer & verifier.
- Use latency histograms to size alert thresholds (e.g. P99 > 50ms for rotation verification under normal load is atypical in this beta).

Planned metric enhancements:
- Add success/failure outcome counter for attestation similar to rotation aggregate.
- Expose eviction gauge for nonce replay map once TTL/LRU added.


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
