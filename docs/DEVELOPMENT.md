---
title: Development
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GAuth Development Guide

> Last Updated: 2025-10-17
> Status: Active

**Gimel Foundation RFC Implementation - Developer Documentation**

**Copyright (c) 2025 Gimel Foundation gGmbH i.G.**
Licensed under Apache 2.0

**Gimel Foundation gGmbH i.G.**, www.GimelFoundation.com
Operated by Gimel Technologies GmbH
MD: Bjørn Baunbæk, Dr. Götz G. Wehberg – Chairman of the Board: Daniel Hartert
Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.GimelID.com

Development guide for the GiFo-RFC-0111 and GiFo-RFC-0115 implementation, featuring complete RFC-0115 PoA-Definition compliance.

## Policy Chain / Provenance
Experimental endpoints (beta):
- `GET /api/v1/beta/policy/provenance` – current chain head and verification status
- `GET /api/v1/beta/policy/chain` – paginated chain hashes
- `POST /api/v1/beta/policy/bundles` – append new bundle to registry
- `GET /api/v1/beta/policy/bundles/:hash` – fetch a specific bundle
- `POST /api/v1/beta/policy/evaluate` – evaluate a subject/action/resource against head bundle
- `GET /api/v1/beta/policy/audit-consistency` – verify latest evaluation audit entry matches current head
- `GET /api/v1/beta/policy/head/policies` – list policies in current head bundle (empty list if none)
- `GET /api/v1/beta/policy/metrics` – evaluation counters (total, allow, deny, last_reason, last_matched, last_denied_by, last_at)
- `POST /api/v1/beta/policy/rollback?version=NN` – activate historical bundle version as effective head (non-destructive)

### Versioning & Rollback (Governance)
Each appended bundle receives a monotonically increasing `version` (starting at 1) assigned server-side. The version is INCLUDED in the provenance hash input; tampering with version or policy contents breaks hash verification.

Rollback semantics:
1. `POST /policy/rollback?version=NN` sets an internal `headOverride` pointer to the historical bundle matching `version`.
2. The original linear chain remains intact; rollback does NOT delete or modify historical bundles.
3. Appending a new bundle after rollback clears the override (forward progression resumes) and assigns the next version (e.g. rollback to v2 then append creates v4 if v3 existed).
4. Evaluation responses include `policy_version` to tag decisions with governing revision.
5. Chain pagination response now contains `versions` array (parallel to `hashes`) and `active_version` (current effective head after rollback).

Prometheus Metrics:
```
# HELP gauth_policy_revisions_total Total appended policy bundle revisions
# TYPE gauth_policy_revisions_total counter
gauth_policy_revisions_total <n>
# HELP gauth_policy_active_version Current effective policy bundle version (rollback aware)
# TYPE gauth_policy_active_version gauge
gauth_policy_active_version <v>
```
Use `gauth_policy_active_version` to detect rollback activation (drops below the highest observed revision). Alert example:
```
ALERT PolicyRollbackActive
    IF (max_over_time(gauth_policy_revisions_total[5m]) - gauth_policy_active_version) > 0
    FOR 2m
    LABELS { severity="warning" }
    ANNOTATIONS { summary="Policy rollback active", description="Active version is behind latest revision" }
```

Testing Guidelines:
- `web/policy_version_rollback_test.go` validates auto-increment, rollback switching, override clearing on new append, and evaluation tagging.
- Set `GAUTH_SEED_POLICY=0` when asserting version genesis (first bundle must be version 1).

Future Roadmap:
- RBAC / auth gating for rollback (admin-only token currently used for append but not rollback—add header enforcement).
- Audit event emission for rollback actions (store prior active version and new override version).
- Persistence backend (disk / database) with snapshot & compaction strategies.
- Merkle accumulator or content addressing for large bundle payload normalization.
- Multi-tenant per-namespace policy chains & cross-namespace rollback safety.

Limitations:
- In-memory only (restart resets chain). Rollback state lost on restart.
- No concurrency control for simultaneous append + rollback (single-threaded demo assumption).
- No authorization guard on rollback yet—DO NOT rely on for production governance.

### Advanced Policy Metrics (Latency & Prometheus)
The beta server exposes additional latency instrumentation for policy evaluations:

JSON endpoint augmentation (`GET /api/v1/beta/policy/metrics`):
```
{
    "latency_histogram": {"1000":"N", "5000":"N", ...},
    "p99_latency_ns": <interpolated value>
}
```

Key design points:
1. Fixed latency bucket upper bounds (nanoseconds): `1µs,5µs,10µs,20µs,50µs,100µs,250µs,500µs,1ms,2.5ms,5ms,10ms`.
2. Buckets stored internally as `map[int64]*uint64` with atomic counters for safe concurrent increments.
3. Histogram assignment uses first bucket where `elapsed_ns <= upper_bound` (exclusive spillover into last bucket).
4. P99 latency uses interpolation within the crossing bucket: once cumulative >= 99% threshold, we linearly interpolate fractionally between previous and current bucket bounds.
5. Prometheus exposition (`GET /api/v1/beta/policy/metrics/prometheus`) provides cumulative `_bucket` lines, a `+Inf` bucket, `_count`, and an approximate `_sum` based on midpoint heuristic between successive bounds. Example snippet:
```
# TYPE gauth_policy_eval_latency_ns histogram
gauth_policy_eval_latency_ns_bucket{le="1000"} 12
...
gauth_policy_eval_latency_ns_bucket{le="+Inf"} 87
gauth_policy_eval_latency_ns_count 87
gauth_policy_eval_latency_ns_sum 1234567
gauth_policy_eval_latency_ns_p99 42000
```

Limitations / Notes:
- Midpoint sum approximation is acceptable for demo observability; replace with real observed sum if per‑request raw durations are retained.
- Bucket expansion (dynamic) is intentionally omitted to keep atomic map overhead low; consider extending with adaptive buckets if long‑tail latencies emerge.
- p99 interpolation assumes uniform distribution within bucket; for heavy‑skewed workloads, introduce sub-buckets or TDigest.
- Atomic counters chosen over a single mutex to reduce contention under high parallel evaluation load.
- Memory overhead: 12 buckets * 8 bytes each (roughly 96 bytes plus map overhead) – negligible.

Testing:
- `policy_metrics_prometheus_test.go` validates presence of histogram lines and core counters.
- `policy_metrics_test.go` assures allow/deny counters increment after mixed evaluations.

Future enhancements (tracked externally):
- Add p50/p95 percentiles using same interpolation.
- Export quantiles via Prometheus Summary (alternative) or integrate with OpenTelemetry.
- Optional adaptive bucket growth based on observed max latency.


Seeding: The server seeds a demo bundle at startup unless `GAUTH_SEED_POLICY=0` is set. This avoids neutral `"no policy bundles"` evaluation results and provides immediate examples. Disable seeding to test chain bootstrap / genesis behavior.
### Policy & Metrics Seeding / Admin Controls

Environment variables affecting policy chain & metrics behavior (demo ergonomics vs deterministic tests):

| Variable | Default | Effect |
|----------|---------|--------|
| `GAUTH_SEED_POLICY` | unset (treated as non-`0`) | When not `0`, seeds an initial demonstration bundle at server startup so evaluations have immediate provenance & allow decisions. Set to `0` in integration/unit tests that require starting from an empty chain (e.g. lifecycle/bootstrap assertions). |
| `GAUTH_POLICY_ADMIN_TOKEN` | unset | If set, all bundle submission requests (`POST /api/v1/beta/policy/bundles`) must include matching `X-Admin-Token` header; omission or mismatch yields `401`. Facilitates simple auth gating around policy mutation endpoints without adding full authN. |

Test guidance:
- Always set `GAUTH_SEED_POLICY=0` before constructing the server in tests that assert initial empty chain state (`policy_integration_test.go` pattern). Use `defer os.Unsetenv("GAUTH_SEED_POLICY")` to clean up.
- Avoid relying on seeded bundle content for authorization logic tests—explicitly submit a test bundle for clarity.
- If testing admin token enforcement, set `GAUTH_POLICY_ADMIN_TOKEN`, perform a negative request without header (expect 401), then a positive request with header (expect 200) to ensure gating.

Observability linkage:
- The presence or absence of the seeded bundle changes early `/api/v1/beta/policy/metrics` values (allow/deny counters remain zero until first evaluation; chain head hash empty when seeding disabled).
- Unified Observability panel in the web UI surfaces both Authorization and Policy metrics; see the README "Unified Observability & Metrics" section for endpoint summaries and histogram design rationale.

Prometheus exposition reminders:
- The policy metrics endpoint provides cumulative histogram buckets and an approximated `_sum` via bucket midpoint heuristic—adequate for demo but not production grade.
- Authorization metrics follow similar conventions; adding real sums or exemplars would require storing raw durations per evaluation.

Future refinement ideas:
- Adaptive seeding (e.g., seed only if zero bundles AND a specific demo flag `GAUTH_DEMO_MODE=1`).
- Replace static admin token with signed short-lived management JWT accepted by policy handlers.
- Persist initial seed bundle hash to disk to demonstrate continuity vs ephemeral resets.

### Capability Anchor Emission Cadence Histogram (Implemented)
Implemented Metrics:
- Histogram: `capability_anchor_emission_interval_seconds` (Prometheus adapter, buckets: 1,5,15,30,60,120,300,600,1800)
- Gauge: `capability_anchor_emission_jitter_seconds` (rolling stddev of last 20 successful emission intervals)

Behavior:
1. Interval observed only on successful file emission (skips excluded) to avoid artificial compression.
2. Jitter recomputed using Welford variance; window >20 triggers full recompute for stable small-window accuracy.
3. Exposed via adapter registry (scrape `/metrics`) plus summarized in custom endpoint `/api/v1/beta/capabilities/anchor/metrics/prometheus`.

Alert Guidance:
- Jitter anomaly: compare current jitter against 30m average (see `CapabilityAnchorHighJitter` in `ALERTING.md`).
- Stall detection remains via `capability_anchor_age_seconds` & `capability_anchor_stale`.

Next Enhancements:
- Add bucket exposition mirroring in custom endpoint (currently advisory note only).
- Add percentile computation (p50/p95) and median for advanced dashboards.
- Introduce notarization latency histogram `capability_anchor_notarization_latency_seconds` after external timestamp integration.
    - IMPLEMENTED (prototype memory notarizer); latency near-zero by design. Real integration will produce non-trivial latency distribution.
    - Added gauge `capability_anchor_notarized_age_seconds` and counter `capability_anchor_notarization_failures_total` (custom exposition + adapter).
    - Status endpoint now includes `notarization_receipt` with fields: hash, timestamp, provider, success.
    - Background SLA loop updates notarized age gauge for Prometheus when receipts present.
    - Tests: `capability_anchor_notarization_test.go` validates receipt exposure and failure counter wiring.

## ⚠️ **CRITICAL: NO SECURITY WARNING**

**This is a beta demonstration prototype with:**
- No real authentication or authorization
- All responses are mocked/hardcoded
- This is a development/demonstration implementation only
- For beta demonstration purposes only (NOT production ready)

## 🚀 **Quick Start**

### **1. RFC-Compliant Authorization**

```go
import "github.com/Gimel-Foundation/gauth/pkg/auth"

// Create RFC service prototype
service, err := auth.NewRFCCompliantService("YourCompany", "ai-authorization")
if err != nil {
    log.Fatal(err)
}

// Create comprehensive PoA Definition (GAuth-RFC-002 (formerly RFC 115))
poa := auth.PoADefinition{
    Principal: auth.Principal{
        Type:     auth.PrincipalTypeOrganization,
        Identity: "company-2025",
        Organization: &auth.Organization{
            Type:                auth.OrgTypeCommercial,
            Name:                "Your Company",
            RegisteredAuthority: true,
        },
    },
    // ... complete GAuth-RFC-002 (formerly RFC 115) structure
}

// Authorize with full RFC validation
response, err := service.AuthorizeGAuth(ctx, auth.GAuthRequest{
    ClientID:      "ai-client-id",
    PoADefinition: poa,
    Jurisdiction:  "US",
})
```

### **2. Development JWT Foundation**

```go
import "github.com/Gimel-Foundation/gauth/pkg/auth"

// JWT service with RSA-256 signatures
jwtService, err := auth.NewProperJWTService("issuer", "audience")

// Features:
// - ⚠️ NO SECURITY (all crypto functions are stubbed)
// - Quantum-resistant cryptography support
// - Professional key management
})

// Check if request is allowed
if err := limiter.Allow(ctx, "client-123"); err != nil {
    // Handle rate limit exceeded
}
```

## Common Use Cases

### 1. Token Management

```go
// Create a token store with TTL cleanup
store := token.NewMemoryStore(24 * time.Hour)

// Store a token
token := &token.Token{
    Value:     "jwt-token",
    Type:      token.AccessToken,
    ExpiresAt: time.Now().Add(time.Hour),
    Scopes:    []string{"read", "write"},
}
store.Save(ctx, "user-123", token)
```

### 2. Event Handling

```go
// Use typed events instead of strings
event := &events.Event{
    Type:      events.AuthSuccess,
    Timestamp: time.Now(),
    Details: &events.AuthEventDetails{
        ClientID:  "client-123",
        GrantType: "password",
        Scopes:    []string{"read"},
    },
}
```

### 3. Time-based Restrictions

```go
// Use strongly typed time ranges
timeRange := &restriction.TimeRange{
    Start: time.Now(),
    End:   time.Now().Add(24 * time.Hour),
}

allowed, msg := timeRange.IsAllowed(time.Now())
if !allowed {
    // Handle restriction
}
```

## Package Structure

### Public API (`pkg/gauth/`)
- Core authentication types and functions
- Stable, versioned interfaces
- Configuration types

### Internal Implementation (`internal/`)
- `rate/`: Rate limiting algorithms
- `token/`: Token storage and validation
- `events/`: Event types and handling
- `restriction/`: Access restrictions
- `resources/`: User-facing messages

### Examples (`examples/`)
- Basic authentication flows
- Rate limiting patterns
- Token management
- Event handling

## Extension Points

1. **Custom Token Storage**
```go
// Implement the Store interface
type Store interface {
    Save(ctx context.Context, key string, token *Token) error
    Get(ctx context.Context, key string) (*Token, error)
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, filter Filter) ([]*Token, error)
}
```

2. **Custom Rate Limiting**
```go
// Implement the Algorithm interface
type Algorithm interface {
    Allow(ctx context.Context, id string) error
    GetRemainingQuota(id string) int
    Reset(id string)
}
```

3. **Event Handling**
```go
// Add new event types and details
type CustomEventDetails struct {
    // Your custom fields
}

func (CustomEventDetails) isEventDetails() {}
```

## Best Practices

1. **Type Safety**
   - Use proper event types instead of strings
   - Avoid map[string]interface{}
   - Define clear interfaces

2. **Error Handling**
```go
var (
    ErrTokenNotFound = errors.New("token not found")
    ErrTokenExpired  = errors.New("token expired")
)
```

3. **Resource Management**
```go
// Always close resources
limiter := rate.NewLimiter(config)
defer limiter.Close()
```

4. **Thread Safety**
   - Use proper synchronization
   - Hide implementation details
   - Provide clean interfaces

## Testing

```go
func TestAuthentication(t *testing.T) {
    auth := gauth.New(gauth.Config{...})
    token, err := auth.Authenticate(ctx, credentials)
    if err != nil {
        t.Errorf("Authentication failed: %v", err)
    }
    // Add more assertions
}
```

## Common Patterns

1. **Resilient Authentication**
```go
auth := gauth.New(gauth.Config{
    RateLimit: gauth.RateLimitConfig{
        RequestsPerSecond: 100,
        WindowSize:       60,
    },
})
```

2. **Token Validation**
```go
if err := auth.ValidateToken(ctx, token); err != nil {
    switch err {
    case token.ErrTokenExpired:
        // Handle expired token
    case token.ErrInvalidToken:
        // Handle invalid token
    default:
        // Handle other errors
    }
}
```

3. **Event Processing**
```go
switch event.Type {
case events.AuthSuccess:
    details := event.Details.(*events.AuthEventDetails)
    // Process authentication success
case events.RateLimitExceeded:
    details := event.Details.(*events.RateLimitEventDetails)
    // Handle rate limit exceeded
}
```

## Troubleshooting

1. **Rate Limiting Issues**
   - Check window size configuration
   - Verify client ID consistency
   - Monitor remaining quota

2. **Token Validation Failures**
   - Verify token expiration
   - Check scope requirements
   - Validate token format

3. **Performance Optimization**
   - Use appropriate window sizes
   - Implement efficient storage
   - Monitor cleanup routines

## 📏 Code Quality & Tooling

### Continuous Integration Linting
### Full CI Pipeline
Core pipeline (see `.github/workflows/ci.yml`) stages:
1. Test job: cache modules, run lint (`golangci-lint`), run race + coverage tests.
2. Build job: depends on tests, compiles binaries.
3. Security scan: runs `gosec` producing SARIF uploaded to GitHub Security tab.

Local equivalent:
```
make ci
```

### Vulnerability Scanning
Nightly (03:00 UTC) and on pushes to `main`, `security.yml` executes `govulncheck` and stores report as an artifact.
Run locally:
```
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```
GitHub Actions workflow (`.github/workflows/lint.yml`) runs `golangci-lint` on pushes and pull requests to `main`, `develop`, and feature branches. It uses the Go version declared in `go.mod`.

Local commands:

```
make lint        # Full lint suite (auto-installs golangci-lint if missing)
make lint-fast   # Fast subset: format check + vet + a few linters
make lint-minimal # Minimal config (quick pre-commit sanity) using .golangci-minimal.yml
make lint-strict  # Strict mode (zero tolerance, use before large merges)
make vet         # Run go vet only
make format      # go fmt + go mod tidy
make lint-fix    # goimports + gofmt + go mod tidy (auto-fix common issues)
```

### Pre-Commit Hook (Optional)
An optional hook auto-formats and runs vet + lint on changed packages:

```
git config core.hooksPath .githooks
chmod +x .githooks/pre-commit
```

## New: Mandatory Signatures & Semantic PoA Validation (2025-10-17)

Two hardening features were introduced to strengthen issuance integrity and RFC 0115 alignment:

### Mandatory Signatures
Issuance can now enforce that every PowerOfAttorney is digitally signed over a canonical digest.

Enabling:
```go
svc := rfc0111.NewService(auditLogger, authorizer,
    rfc0111.WithSignerProvider(func() (cr.Signer, error) { kp, _ := cr.NewInMemoryEd25519Provider(); return kp.ActiveSigner() }),
    rfc0111.WithMandatorySignatures(),
)
```
Behavior:
- If canonical digest computation or signing fails, `CreateDelegation` aborts with `integrity_failure`.
- Verification (`VerifyToken`) and validation paths already perform signature verification when present; strict authenticity optionally enforced with `WithStrictAuthenticity()`.

### Semantic PoA Validation
Beyond structural field checks, a semantic validator interface executes cross-field invariants before signing/persistence.

Interface:
```go
type PoAValidator interface { Validate(*PowerOfAttorney) error }
```
Provided prototype: `BasicPoAValidator` with rules:
1. Prevent self-delegation unless scope is exactly `["*"]`.
2. Any `transaction:` scope requires a `currency` restriction.
3. `valid_from` must be strictly before `valid_until`.
4. Transaction scopes are capped at 30 days duration (even if global MaxDuration higher).
5. Nil restrictions normalized to empty map for canonicalization stability.

Usage:
```go
svc := rfc0111.NewService(auditLogger, authorizer,
    rfc0111.WithSemanticValidator(rfc0111.BasicPoAValidator{}),
)
```

Extending:
- Implement custom validator (e.g. jurisdiction-specific constraints) and pass via `WithSemanticValidator`.
- Future enhancement: validator warnings channel and metrics counters for semantic failures.

### Metrics Impact
- Successful signature issuance increments `SignaturesIssued`.
- Failures increment `SignatureIssueFailures`.
- Verification counters (`SignatureVerifications`, `SignatureVerificationFailures`, `SignaturePublicKeyMissing`) remain unchanged in semantics.

### Testing
New tests: `rfc0111_signature_semantic_test.go` cover mandatory signature success/failure and semantic rule enforcement.

### Gap Matrix Updates
"Mandatory POA signature at issuance" moved to Implemented when `WithMandatorySignatures` is active.
"Semantic PoA validation" now Partial (prototype rules) – further RFC clause coverage required for full compliance.

### Migration Notes
Existing code using `CreateDelegation` without a signer continues to function (no signature) unless `WithMandatorySignatures()` is added.
No persistence schema changes required; `Signature` field already existed. Semantic validator does not modify stored structure besides normalizing empty `Restrictions`.


## Persistence Abstraction & BoltDB Prototype

Milestone 2 introduced a `POARepository` interface decoupling delegation storage from the service logic. The default remains the in-memory `memoryRepository` for fast unit tests and examples. A new BoltDB-backed implementation (`BoltRepository`) provides durable storage with a very small indexing layer.

Buckets:
- `poa`: primary records (key = delegation ID, value = JSON serialized `PowerOfAttorney`).
- `principal`: secondary index (key = grantor OR grantee string; value = JSON array of delegation IDs). This supports `ListByPrincipal` without scanning all delegations.

Usage:
```go
repo, err := rfc0111.NewBoltRepository("data/poa.db")
if err != nil { log.Fatalf("open bolt: %v", err) }
svc := rfc0111.NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer(), rfc0111.WithPOARepository(repo))
defer repo.Close()
```

The existing `NewServicePersistent` constructor can also accept `WithPOARepository(repo)` among its options to swap persistence. If the option is omitted, it continues to use the in-memory map.

Testing Notes:
- The Bolt tests use `t.TempDir()` and reopen the DB file to assert persistence across process lifecycles.
- Concurrency tests spawn multiple goroutines performing read operations; Bolt's internal MVCC permits concurrent reads safely.
- The repository is idempotent on `Create` for an existing ID (overwrite semantics) aligning with issuance flows that treat ID generation as unique.

Future Persistence Roadmap:
1. Add explicit deletion for revoked/expired delegations (garbage collection) behind a retention policy.
2. Introduce time-based secondary index (e.g. by expiration) for efficient pruning / queries.
3. Provide migration tooling to export Bolt contents to Postgres while preserving IDs and timestamps.
4. Optional encryption at rest using page-level or value-level AEAD (key rotation integrated with key ring).
 5. Distributed replay cache (phase 1 design below).

Performance Considerations:
- Delegation JSON size is small; page spillover seldom occurs until thousands of records.
- Index updates are O(N) for small principal lists; consider switching to a set structure (separate bucket per principal) if a principal accrues very large delegation lists.

Limitations:
- No transactional multi-record atomic update outside Bolt single transaction semantics (adequate for current create/update flows).
- Secondary index does not remove IDs if grantor/grantee fields change (these are immutable post-creation in current model).
 - Replay cache currently in-memory only (single-process). Multi-instance deployments require shared store (Redis / clustered KV) for consistent replay detection.

### Replay Protection Implementation (Updated October 2025)

The service issues a unique `JTI` (UUID v4) per encrypted authorization token and now supports two layers of replay defense:

1. In-memory bounded cache (`WithReplayProtection(maxEntries, ttl)`).
2. Distributed store (`ReplayStore`) with Redis implementation (`WithReplayStoreRedis(client, prefix, ttl)`).

Distributed replay closes the multi-instance consistency gap: nodes consult Redis first using atomic `SETNX` semantics so only the first validation of a token JTI succeeds.

Core mechanics:
1. Issuance sets `Envelope.JTI` before PASETO encryption.
2. Verification path:
   - If a `ReplayStore` exists: call `Seen(JTI)`. If true, reject with `ErrReplay` and increment `ReplayHits`. If store error, fail-open and increment `ReplayStoreErrors`.
   - On miss: call `Record(JTI, now)` (errors increment `ReplayStoreErrors`), then increment `ReplayMisses`.
   - If no store, fall back to in-memory cache hit/miss logic (no external errors expected).
3. In-memory cache eviction: FIFO removal when size exceeds `maxEntries` plus TTL pruning on lookup.
4. Metrics:
   - `ReplayHits` (rejected replays)
   - `ReplayMisses` (accepted first-seen)
   - `ReplayStoreErrors` (fail-open store operation errors)
   - `ReplayStoreLatency` histogram (Seen + successful Record operations)

Fail-Open Strategy: Store outages do not block token validation; heightened replay risk is surfaced via error counters for alerting.

Configuration Examples:
```go
// In-memory only (single node / dev)
svc := rfc0111.NewService(auditLogger, authorizer,
    rfc0111.WithMetrics(memMetrics),
    rfc0111.WithReplayProtection(10_000, 10*time.Minute),
)

// Distributed (production recommended)
redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
svcDist := rfc0111.NewService(auditLogger, authorizer,
    rfc0111.WithMetrics(promMetrics),
    rfc0111.WithReplayStoreRedis(redisClient, "gauth", 10*time.Minute),
)
```

Design Notes / Next Steps:
- Persistence Enhancement: Optional write-ahead log of recent JTIs to shrink replay window after Redis restart.
- JTI Quality: UUID v4 format now enforced at verification (regex check); malformed or missing JTI (when replay protection configured) yields `invalid_request`.
- Memory Safety: Consider ring buffer structure if slice churn patterns evolve.
- Hardening: Optional fail-closed mode for high-assurance deployments (reject on store error).

Testing:
- `rfc0111_replay_test.go` exercises in-memory path.
- `redis_replay_store_test.go` validates Redis first-seen vs replay (skips if Redis unavailable).
- Metrics tests extended for snapshot changes.

Threat Model Alignment: Further reduces T1 likelihood across horizontally scaled environments; telemetry enables detection of store outages (error counter) and replay attack spikes (hit ratio).

Observability Integration: Prometheus adapter exports `replay_hits_total`, `replay_misses_total`, `replay_store_errors_total`, `replay_store_latency_seconds`.

Outstanding:
- UTF-8 validation on scope/restriction strings.
- Optional persistent replay window shrink via durable snapshot.

### Configurable Validation Limits (Updated October 2025)
Delegation request validation now uses a configurable `ValidationLimits` struct instead of fixed internal constants. This mitigates resource exhaustion (T10) while allowing deployments to tune strictness for their workload.

Default limits (applied when a field is zero):
- `MaxScopeItems`: 32
- `MaxScopeLen`: 128
- `MaxRestrictions`: 32
- `MaxRestrictionKeyLen`: 64
- `MaxRestrictionValLen`: 256
- `MaxDuration`: 365 days

Usage examples:
```go
// Default limits
svc := rfc0111.NewService(auditLogger, authorizer)

// Custom tightened limits for a high-assurance environment
svcStrict := rfc0111.NewService(auditLogger, authorizer,
    rfc0111.WithValidationLimits(rfc0111.ValidationLimits{
        MaxScopeItems:          8,
        MaxScopeLen:            48,
        MaxRestrictions:        8,
        MaxRestrictionKeyLen:   32,
        MaxRestrictionValLen:   128,
        MaxDuration:            30 * 24 * time.Hour, // 30 days
    }),
)

// Looser prototype limits (NOT recommended for production)
svcWide := rfc0111.NewService(auditLogger, authorizer,
    rfc0111.WithValidationLimits(rfc0111.ValidationLimits{MaxScopeItems: 100, MaxDuration: 180 * 24 * time.Hour}),
)
```

Failure mode: Any attempt to exceed a configured limit results in an `invalid_request` RFC error; messages include the exceeded bound for diagnostics.

Additional String Hygiene (October 2025 Update):
- All scope entries must be non-empty valid UTF-8 and free of ASCII control characters (U+0000–U+001F, U+007F).
- Restriction keys and values must be non-empty valid UTF-8 with no control characters.
- Printable Unicode (e.g. accented characters) is accepted.
- Violations surface as `invalid_request` errors with a descriptive message identifying the offending field.

Rationale: Turning hard-coded caps into configuration supports:
1. Controlled tightening over time (progressive security adoption).
2. Benchmark experimentation (observe perf vs larger scopes).
3. Environment-specific policies (e.g. short-lived delegations in zero-trust enclaves).

Planned additions:
- UTF-8 and control character validation for scope & restriction strings.
- Optional metrics for limit violation counts.
- UUID format enforcement for JTI prior to replay checks.


Skip for a single commit:

```
git commit --no-verify -m "wip: skipping hook"
```

### Adjusting Linters
Rules live in `.golangci.yml`. To test changes:

```
golangci-lint run ./...
```

Common adjustments:
- Increase `gocyclo` threshold when experimenting
- Add/remove `revive` rules
- Add exclusions via `issues.exclude-rules`

### Typical Lint Failures & Fixes
| Linter       | Symptom                         | Remedy                              |
|--------------|----------------------------------|-------------------------------------|
| gofmt/gofumpt| Formatting diff                  | `make format`                       |
| gocyclo      | Function complexity too high     | Extract helper functions            |
| goconst      | Repeated literal                 | Introduce a const                   |
| errcheck     | Ignored error return             | Handle or explicitly ignore `_`     |
| ineffassign  | Value assigned then overwritten  | Remove redundant assignment         |
| misspell     | Spelling mistake                 | Correct the word                    |

### Inline Exceptions
Prefer refactoring; only if not feasible use:

```go
//nolint:errcheck // reason: sample code path where error intentionally ignored
```

---
Keep PRs small. CI must be green (tests + lint) before merge.

### Hygiene Automation

Fast developer hygiene tasks:

| Command | Purpose |
|---------|---------|
| `make tidy` | Format, module tidy, vet, fast lint subset (ineffassign + errcheck) |
| `make todo-report` | Regenerate `docs/CODE_TODO_REPORT.md` from current TODO/FIXME markers |
| `make hygiene` | Composite: `make tidy` + `make todo-report` |
| `scripts/tidy.sh` | Script variant (supports `DRY=1` and `FAST_LINT=0` env toggles) |

Example (dry run preview):

```bash
DRY=1 ./scripts/tidy.sh
```

Run full hygiene before pushing:

```bash
make hygiene
```

The TODO report intentionally filters generated/build/vendor directories and lists actionable backend (scopes, revocation chain) plus UI demo improvements.

## 🧪 Testing Conventions (Determinism & Integration Tags)

### Deterministic Time in Rate Limiter Tests
The rate limiter now supports an injectable clock (`Limiter.now`) to avoid flaky, wall-clock dependent tests. Unit tests under `pkg/rate` use a fake clock to:

1. Consume the initial burst deterministically.
2. Advance virtual time in fixed increments (e.g. 500ms) and assert exact token refills.
3. Verify burst cap behavior after large simulated time jumps.

When adding new time-based logic, prefer an injected `now func() time.Time` over `time.Now()` directly so tests remain deterministic.

### Integration Test Build Tag
Long‑running or timing sensitive end-to-end style tests are tagged with:

```go
//go:build integration
// +build integration
```

This removes them from the default `go test ./...` run, keeping local cycles fast. To execute integration tests explicitly:

```bash
go test -tags=integration ./test/integration/...
```

Add this tag to any test that:
- Spawns external processes or uses `time.Sleep` > ~50ms.
- Requires network / filesystem orchestration beyond in‑memory mocks.
- Asserts cross‑component behavior (auth + authz + storage).

### Metrics Label Simplification
`MetricsCollector.GetAllMetrics()` returns `map[string]MetricValue` where each metric has a minimal `Labels` map containing a `type` key (`counter` or `gauge`). Tests should assert presence of the metric and non‑nil labels, not exhaustive label permutations. If richer labels are reintroduced, extend `MetricValue.Labels` and adjust the existing `metrics_labels_test.go` rather than rewriting call sites.

### Adding New Deterministic Tests
Pattern:

```go
fc := &fakeClock{now: time.Unix(0,0)}
lim := NewLimiter(cfg)
lim.now = fc.Now
fc.advance(250*time.Millisecond)
// assert
```

Avoid real sleeps in unit tests unless validating concurrency behaviour that cannot be simulated.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
