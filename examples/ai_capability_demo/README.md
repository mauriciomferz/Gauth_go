---
title: "AI Capability Matrix Enforcement Demo"
category: example
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# AI Capability Matrix Enforcement Demo

This example (`ai_capability_demo`) showcases how to integrate and enforce AI capability governance using the AgentAuth AI Capability Matrix. It spins up a local Gin server exposing a small API surface plus interactive scenarios demonstrating compliance gates, human authorization requirements, audit levels, risk tiers, and jurisdiction / industry frameworks.

## Key Concepts

The demo walks through multiple entity archetypes:

| Scenario | Entity Type | Jurisdiction / Context | Goal |
|----------|-------------|------------------------|------|
| 1. Human User | (human) | US | Full administrative action allowed |
| 2. AI Assistant | assistant | US | Limited transactional read, blocked execute |
| 3. AI Agent (US) | agent | US | Execute allowed with human auth + audit |
| 4. EU AI Agent | agent | EU | Initially blocked (missing EU compliance), then allowed after adding claims |
| 5. Healthcare AI | analytics | US / healthcare industry | Read allowed with stricter audit & policy application (HIPAA) |
| 6. Financial AI | automation | US / finance industry | Pay action blocked; read allowed with SOX compliance |

Capability enforcement returns:
- `allowed`: boolean decision
- `missing_capabilities`: list of gating capabilities not satisfied
- `metadata`: rich context (entity type, required human auth, applied policies, audit level, reason, required compliance frameworks)

## Folder Structure
```
examples/ai_capability_demo/
  main.go              # Runs scenarios then starts HTTP server
  README.md            # This file
```

## Running the Demo
### Prerequisites
- Go 1.21+
- Working clone of the repository (from project root: `./`)

### Start Demo
From project root:
```bash
go run ./examples/ai_capability_demo
```
You will see:
- Printed scenario evaluations
- Loaded governance policies list
- Startup banner with example curl commands
### Environment Variables
| Variable | Purpose | Default |
|----------|---------|---------|
| `AGENTAUTH_AI_DEMO_PORT` | Port for HTTP server | `8080` |
| `AGENTAUTH_AI_DEMO_NO_SERVER` | If set to `1`, skip starting HTTP server (print scenarios only) | unset |
| `AGENTAUTH_AI_DEMO_DB_PATH` | Path to SQLite database file for decision persistence (creates `decisions` table) | unset |
| `AGENTAUTH_AI_DEMO_DB_MAX_ROWS` | If set (integer), prune oldest rows to keep at most this many decisions | unset |
| `AGENTAUTH_AI_DEMO_DB_MAX_AGE_DAYS` | If set (integer), delete decisions older than this many days after each insert | unset |
| `AGENTAUTH_AI_DEMO_API_KEY` | If set, require matching `X-API-Key` header for protected endpoints | unset |
| `AGENTAUTH_AI_DEMO_JWT_SECRET` | If set (with or without API key), accept Bearer JWT (HMAC-SHA256 signature + optional `exp` claim validation) | unset |
| `AGENTAUTH_AI_DEMO_JWT_EXPECT_ISS` | Expected JWT issuer (`iss`) if provided | unset |
| `AGENTAUTH_AI_DEMO_JWT_EXPECT_AUD` | Expected JWT audience (`aud`) (string or element in array) | unset |
| `AGENTAUTH_AI_DEMO_JWT_CLOCK_SKEW_SECONDS` | Allowed clock skew for `exp`/`nbf`/`iat` validation (default 60) | unset |
| `AGENTAUTH_AI_DEMO_JWKS_URL` | JWKS endpoint URL for RS256 JWT public keys (enables RS256 verification) | unset |
| `AGENTAUTH_AI_DEMO_JWKS_CACHE_SECONDS` | JWKS cache TTL seconds (default 300) | unset |
| `AGENTAUTH_AI_DEMO_JWT_EXPECT_ALG` | Enforce specific JWT alg (e.g. RS256 or HS256); rejects mismatches if set | unset |
| `AGENTAUTH_AI_DEMO_JWKS_BG_REFRESH_FACTOR` | Background refresh fraction of TTL (0<factor<1). Fetches JWKS proactively after `factor * JWKS_CACHE_SECONDS` (default 0.5). | unset |
| `AGENTAUTH_AI_DEMO_JWKS_BG_REFRESH_JITTER_SECONDS` | Max +/- jitter (seconds) applied to background refresh delay to de-synchronize replicas (default 0 = disabled). | unset |
| `AGENTAUTH_AI_DEMO_JWKS_NEGATIVE_TTL_SECONDS` | TTL (seconds) to cache missing `kid` lookups and avoid repeated JWKS fetch attempts. | unset |
| `AGENTAUTH_AI_DEMO_JWKS_NEGATIVE_MAX_ENTRIES` | Maximum number of negative (missing kid) cache entries before oldest eviction. | unset |
| `AGENTAUTH_AI_DEMO_OTEL` | If `1`, enable OpenTelemetry stdout tracing for enforcement & conflict simulation spans | unset |
| `AGENTAUTH_AI_DEMO_METRICS` | If `1`, enable Prometheus counters and expose `/metrics` endpoint | unset |
| `AGENTAUTH_AI_DEMO_POA_DB_PATH` | Path to BoltDB file for persistent PoA storage (falls back to in-memory if unset) | unset |

### Start Demo
From project root (default port):
```bash
go run ./examples/ai_capability_demo
```
Custom port:
```bash
export AGENTAUTH_AI_DEMO_PORT=9090
go run ./examples/ai_capability_demo
```
Run scenarios only (no server):
```bash
export AGENTAUTH_AI_DEMO_NO_SERVER=1
go run ./examples/ai_capability_demo
```
You will see:
- Printed scenario evaluations
- Loaded governance policies list
- Startup banner with example curl commands (unless server disabled)

### Example Console Output (abridged)
```
🤖 AI Capability Matrix Enforcement Demo
========================================

🎭 Demo Scenarios:
================
1. 👤 Human User Access:
   Action: admin:delete | Allowed: true | Missing: [] | Entity: human
2. 🤖 AI Assistant Access:
   Action: transaction:read | Allowed: true | Missing: [] | Human Auth: false
   Action: transaction:execute | Allowed: false | Missing: [cap_transaction_execute] | Reason: restricted_entity_type
...
🚀 Server starting on http://localhost:8080
Example API calls:
curl http://localhost:8080/api/v1/ai/capabilities/status
curl http://localhost:8080/api/v1/ai/capabilities/entity-types
curl http://localhost:8080/api/v1/ai/health
```

## Exposed Endpoints
After startup the server provides:
- `GET /api/v1/ai/capabilities/status` – Health / config status of capability matrix
- `GET /api/v1/ai/capabilities/entity-types` – Enumerates supported AI entity types
- `GET /api/v1/ai/capabilities/policies` – Lists loaded governance policies
- `POST /api/v1/ai/capabilities/test/enforcement` – Test an arbitrary action + claims payload
- `POST /api/v1/ai/capabilities/simulate/decision` – Simulated decision for experimentation
- `GET /api/v1/ai/health` – Basic health signal
- `POST /demo/enforce` – Demo wrapper that returns decision + metadata for provided action/claims
- `POST /api/v1/ai/capabilities/simulate/conflict` – Run the same action across multiple jurisdictions & detect divergence
- `GET /demo/decisions` – List persisted decisions (requires `AGENTAUTH_AI_DEMO_DB_PATH`); supports `?limit=` (<=500) and `?offset=` for pagination
- Filters supported: `?action=transaction:read` and/or `?entity_type=assistant` can be combined with pagination.
- `GET /demo/decisions/export` – Bulk export decisions (requires persistence) with `?format=ndjson|csv` (default ndjson) and optional `?limit=` (up to 100000).
- `GET /demo/decisions/stats` – Summary statistics (total rows, oldest/newest timestamps, top 5 actions) when persistence enabled.
- `GET /metrics` – Prometheus metrics (enabled when `AGENTAUTH_AI_DEMO_METRICS=1`) including decision and conflict counters.
- `GET /` – Root descriptor listing available endpoints
- **PoA (Beta MVP)**:
  - `POST /demo/poa/issue` – Issue a minimal PoA (fields: grantor, grantee, scope, valid_for_seconds, jurisdiction, witnesses, attestations)
  - `POST /demo/poa/:id/revoke` – Revoke an existing PoA (optional `?reason=` query)
  - `POST /demo/poa/:id/token` – Issue a short‑lived HS256 JWT extended token embedding PoA integrity claims (`poa_id`, `poa_digest`, `poa_version`, `token_version=et_v1`) (requires `AGENTAUTH_AI_DEMO_JWT_SECRET`)
  - `/demo/enforce` – If `poa_id` present, validates status, temporal validity, scope membership; attaches `poa_id` & canonical digest to persisted decision row.
  - Persistent storage: if `AGENTAUTH_AI_DEMO_POA_DB_PATH` set, PoAs stored durably in BoltDB with primary bucket and secondary principal index (falls back to in-memory map otherwise).
  - **Multi-Signature Draft Workflow (Week 2)**:
    - `POST /demo/poa/prepare` – Create a draft PoA requiring threshold signatures (`signers[]`, `threshold`, optional `valid_for_seconds`). Status starts as `draft`.
    - `POST /demo/poa/:id/sign` – Submit a signer signature (demo uses deterministic placeholder; real flow would supply cryptographic signature). Rejects unknown or duplicate signer.
    - `GET /demo/poa/:id/status` – View draft progress (current signature count vs threshold).
    - `POST /demo/poa/:id/finalize` – Activate draft once threshold signatures collected; transitions status from `draft` to `active`.
    - Draft status constant: `POAStatusDraft` introduced for quorum gating.

## Example Demo Enforcement Request (`POST /demo/enforce`)
```bash
curl -X POST http://localhost:8080/demo/enforce \
  -H 'Content-Type: application/json' \
  -d '{"action":"transaction:read","claims":{"ai_entity_type":"assistant","ai_entity_verified":true,"algorithmic_accountability":true}}'
```
Response (illustrative):
```json
{
  "success": true,
  "result": {
    "action": "transaction:read",
    "allowed": true,
    "missing_capabilities": [],
    "metadata": {
      "entity_type": "assistant",
      "required_human_auth": false,
      "audit_level": "standard",
      "applied_policies": ["POLICY-US-AI-BASE"],
      "timestamp": "2025-10-28T07:42:17Z"
    },
    "timestamp": "2025-10-28T07:42:17Z"
  }
}
```

  ## Example Simulation Request (`POST /api/v1/ai/capabilities/simulate/decision`)
  Use the simulation endpoint to obtain a decision object without invoking the demo wrapper. This mirrors internal evaluation and returns structured decision metadata directly.

  ```bash
  curl -X POST http://localhost:8080/api/v1/ai/capabilities/simulate/decision \
    -H 'Content-Type: application/json' \
    -d '{
          "action":"transaction:execute",
          "claims":{
            "ai_entity_type":"assistant",
            "ai_entity_verified":true,
            "algorithmic_accountability":true
          }
        }'
  ```
  Illustrative response (fields may evolve):
  ```json
  {
    "success": true,
    "decision": {
      "action": "transaction:execute",
      "allowed": false,
      "missing_capabilities": ["cap_transaction_execute"],
      "metadata": {
        "entity_type": "assistant",
        "required_human_auth": true,
        "audit_level": "elevated",
        "reason": "restricted_entity_type",
        "applied_policies": ["POLICY-US-AI-BASE"]
      },
      "timestamp": "2025-10-28T07:42:55Z"
    }
  }
  ```

## Multi-Jurisdiction Conflict Simulation (`POST /api/v1/ai/capabilities/simulate/conflict`)
Detect whether an action evaluates differently across jurisdictions for the same (or minimally adjusted) claim set.

```bash
curl -X POST http://localhost:8080/api/v1/ai/capabilities/simulate/conflict \
  -H 'Content-Type: application/json' \
  -d '{
        "action":"transaction:read",
        "claims":{
          "ai_entity_type":"assistant",
          "ai_entity_verified":true,
          "algorithmic_accountability":true
        },
        "jurisdictions":["US","EU","UK"]
      }'
```
Illustrative response:
```json
{
  "success": true,
  "action": "transaction:read",
  "decisions": {
    "US": true,
    "EU": true,
    "UK": true
  },
  "conflict": false
}
```
If divergence occurs (`conflict: true`), inspect compliance claims for missing regional requirements.

## Audit Ledger & Delegation Integrity Additions
To provide verifiable anchoring of governance and delegation events the demo now includes an in-memory Merkle ledger (see `README-ledger.md` for deep details):

### Automatic Anchoring Events
The following lifecycle operations automatically append a ledger entry and increment the unified events metric:
- Decision enforcement (`decision_enforce`)
- PoA issuance (`poa_issue`)
- PoA draft preparation (`poa_prepare`)
- PoA signer submission (`poa_sign`)
- PoA finalization (`poa_finalize`)
- PoA revocation (`poa_revoke`)
- PoA token issuance (`poa_token_issue`) – HS256 JWT with embedded `poa_id`, `poa_digest`, `poa_version`, `token_version`
- Manual append (`ledger_append_manual`)
- Root emission (`ledger_root_emit`)

### Ledger Endpoints
- `POST /demo/ledger/append` – Manual append (specify `type`, `payload`, optional `parent_root`).
- `GET /demo/ledger/root/latest` – Latest Merkle root + total entries.
- `GET /demo/ledger/entry/:id/proof` – Inclusion proof with sibling path.
- `GET /demo/ledger/roots` – Full historical root list for external auditors to verify time evolution and detect tampering.

### Metrics
Unified events counter consolidates all anchoring into one dimensional series:
```
ai_demo_ledger_events_total{event="<label>"}
```
Supplementary metrics remain:
- `ai_demo_ledger_appends_total{type}`
- `ai_demo_ledger_root_emissions_total`

### Proof Verification
Client can recompute root from `entry_hash` + ordered `siblings` using deterministic left/right pairing (duplicate-last strategy when odd). Orientation bits will be added in a future enhancement for explicit structural provenance.

### Why Unified Metric?
Provides a single high-cardinality-safe dimension for dashboards/time-series anomaly detection (e.g. unexpected spike in `poa_revoke`). Easier to alert on aggregate ledger activity without joining multiple counters.

### Future Roadmap
- Persistent ledger backend (BoltDB/SQLite)
- Batch append & streaming emission (Kafka)
- Cross-ledger stitching via `parent_root`
- Orientation bits + compressed proof format
- Optionally SNARK/STARK verification primitives

Refer to `examples/ai_capability_demo/README-ledger.md` for full specification and test references.

## Decision Persistence (SQLite)
Filtering examples:
```bash
curl 'http://localhost:8080/demo/decisions?action=transaction:read&entity_type=assistant&limit=20'
```

Export examples:
```bash
# NDJSON (default)
curl 'http://localhost:8080/demo/decisions/export?limit=200' -o decisions.ndjson

# CSV
curl 'http://localhost:8080/demo/decisions/export?format=csv&limit=200' -o decisions.csv
```
You can retrieve recent decisions:
```bash
curl 'http://localhost:8080/demo/decisions?limit=10'
```
Response excerpt:
```json
{
  "decisions": [
    {
      "id": 42,
      "action": "transaction:read",
      "allowed": true,
      "entity_type": "assistant",
      "jurisdictions": "US",
      "missing": [""],
      "applied_policies": "[us_ai_governance_2025]",
      "created_at": "2025-10-29T10:01:22.123456Z"
    }
  ],
  "count": 1
}
```
If `AGENTAUTH_AI_DEMO_DB_PATH` is set (e.g. `export AGENTAUTH_AI_DEMO_DB_PATH=/tmp/agentauth_demo.db`), each `/demo/enforce` call records a row:

Retention:
Set `AGENTAUTH_AI_DEMO_DB_MAX_ROWS` to automatically prune oldest rows beyond the limit (keeps most recent by `id`). Example:
```bash
export AGENTAUTH_AI_DEMO_DB_MAX_ROWS=5000
```
Pruning occurs after each insert; deletion uses a single subquery selecting most recent IDs.

Age-based pruning:
Set `AGENTAUTH_AI_DEMO_DB_MAX_AGE_DAYS` to remove decisions older than N days automatically:
```bash
export AGENTAUTH_AI_DEMO_DB_MAX_AGE_DAYS=30
```
Age pruning runs after row-based pruning; both can be combined.

Statistics endpoint:
```bash
curl http://localhost:8080/demo/decisions/stats
```
Example response:
```json
{
  "total": 1284,
  "oldest": "2025-09-29T11:02:15.123456Z",
  "newest": "2025-10-29T10:44:07.987654Z",
  "top_actions": [
    {"action":"transaction:read","count":840},
    {"action":"transaction:execute","count":320},
    {"action":"transaction:pay","count":124}
  ]
}
```

Schema (updated with PoA linkage & integrity columns):
```sql
CREATE TABLE decisions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  action TEXT NOT NULL,
  allowed INTEGER NOT NULL,
  entity_type TEXT,
  jurisdictions TEXT,
  missing TEXT,
  applied_policies TEXT,
  poa_id TEXT,
  poa_digest TEXT,
  poa_version INTEGER,
  created_at TEXT NOT NULL
);
```
Quick inspection:
```bash
sqlite3 /tmp/agentauth_demo.db 'SELECT id,action,allowed,entity_type,jurisdictions,missing,created_at FROM decisions LIMIT 5';
```

## Authentication Middleware
Set an API key:
```bash
export AGENTAUTH_AI_DEMO_API_KEY=secret123
curl -X POST http://localhost:8080/demo/enforce \
  -H 'X-API-Key: secret123' -H 'Content-Type: application/json' \
  -d '{"action":"transaction:read","claims":{"ai_entity_type":"assistant","ai_entity_verified":true}}'
```
Without the header you'll receive `401 {"error":"unauthorized"}`.

Add JWT support (HS256 verified):
```bash
export AGENTAUTH_AI_DEMO_JWT_SECRET=devjwt
curl -X POST http://localhost:8080/demo/enforce \
  -H 'Authorization: Bearer header.payload.signature' \
  -H 'Content-Type: application/json' \
  -d '{"action":"transaction:read","claims":{"ai_entity_type":"assistant","ai_entity_verified":true}}'
```
The signature is recomputed with HMAC-SHA256 over `header.payload` using `AGENTAUTH_AI_DEMO_JWT_SECRET` and compared constant-time. Claims enforced if present: `exp` (future), `nbf` (not in future beyond skew), `iat` (not excessively future), plus optional `iss` / `aud` matching when corresponding environment variables are set. Adjust skew via `AGENTAUTH_AI_DEMO_JWT_CLOCK_SKEW_SECONDS`.

### JWT Error Reasons
When authentication fails a structured JSON error is returned:
```json
{"error":"unauthorized","reason":"expired_token"}
```
Possible `reason` codes:
| Reason | Meaning |
|--------|---------|
| `missing_api_key` | API key required but absent (and no valid JWT provided) |
| `invalid_jwt_signature` | HMAC-SHA256 signature mismatch |
| `malformed_token` | Token structure or base64 decoding failed |
| `expired_token` | `exp` claim is in the past beyond allowed skew |
| `not_before_violation` | `nbf` claim is in the future beyond skew |
| `future_iat` | `iat` claim too far in the future |
| `issuer_mismatch` | `iss` claim does not match expected issuer |
| `audience_mismatch` | `aud` claim missing expected audience |
| `unsupported_alg` | Algorithm not allowed or does not match `AGENTAUTH_AI_DEMO_JWT_EXPECT_ALG` |
| `jwks_fetch_error` | JWKS retrieval failed or parse error |
| `kid_not_found` | Token `kid` header not present in JWKS set |
| `rsa_verification_failed` | RS256 signature verification failed |
| `poa_not_found` | `poa_id` claim references unknown PoA |
| `poa_revoked` | Referenced PoA has been revoked |
| `poa_expired` | Referenced PoA is outside its validity window |
| `poa_digest_mismatch` | Token embedded digest does not match canonical PoA digest (integrity failure) |
| `poa_version_mismatch` | Token embedded version does not match stored PoA version |
| `unsupported_token_version` | Token `token_version` claim not recognized (future compatibility) |

### RS256 / JWKS Verification
Enable RS256 tokens signed by an issuer's private key verified against a JWKS URL.
```bash
export AGENTAUTH_AI_DEMO_JWKS_URL=https://issuer.example.com/.well-known/jwks.json
export AGENTAUTH_AI_DEMO_JWT_EXPECT_ALG=RS256   # optional but recommended
```
Optionally tune cache TTL:
```bash
export AGENTAUTH_AI_DEMO_JWKS_CACHE_SECONDS=600
```
Flow:
- Decode JWT header to extract `alg` and `kid`.
- Enforce expected `alg` if `AGENTAUTH_AI_DEMO_JWT_EXPECT_ALG` set.
- Fetch JWKS (cached) if cache expired or not populated.
- Resolve RSA public key by `kid`.
- Verify RS256 signature (PKCS#1 v1.5 + SHA-256).
- Validate claims (`exp`, `nbf`, `iat`, `iss`, `aud`).

### JWKS Background Refresh & Negative KID Caching
To reduce latency spikes from on-demand JWKS fetch right at expiry and to suppress repeated network calls for unknown key IDs, two optional behaviors can be enabled:

1. Background Refresh (`AGENTAUTH_AI_DEMO_JWKS_BG_REFRESH_FACTOR`):
  - A goroutine wakes early (fraction of original TTL) and refreshes JWKS before expiry.
  - Default factor: `0.5` (refresh halfway through TTL).
  - Configure a different fraction (e.g. `0.3`) for earlier refresh: `export AGENTAUTH_AI_DEMO_JWKS_BG_REFRESH_FACTOR=0.3`.
  - Must be between `0` and `1` (exclusive). Values outside range are ignored and default used.

2. Negative KID Cache (`AGENTAUTH_AI_DEMO_JWKS_NEGATIVE_TTL_SECONDS`):
  - When a `kid` is not found in the current JWKS set, the missing `kid` is cached for the specified TTL.
  - Subsequent requests for the same `kid` during TTL immediately return `kid_not_found` without triggering a JWKS re-fetch, reducing unnecessary fetch attempts.
  - Example: `export AGENTAUTH_AI_DEMO_JWKS_NEGATIVE_TTL_SECONDS=60` (cache missing KIDs for 60s).

These optimizations improve resilience under burst traffic and noisy / misconfigured clients repeatedly sending invalid `kid` values.

Failure reasons map to structured error codes above for observability.

## OpenTelemetry Tracing
Tracer provider shuts down gracefully (3s timeout) to flush spans.
## Jurisdiction Conflict Metric
Whenever a divergence is detected in multi-jurisdiction evaluation (`/demo/enforce` with jurisdictions or the conflict simulation endpoint), a metric named `jurisdiction_conflict` is incremented via the metrics callback. Hook this into real telemetry to monitor regulatory friction zones.
If `AGENTAUTH_AI_DEMO_METRICS=1` is set, an additional Prometheus counter `ai_demo_conflicts_total` tracks conflict occurrences, and decisions are counted in `ai_demo_decisions_total{action,allowed}`.
Latency histograms exposed:
- `ai_demo_enforcement_duration_seconds{action}` – single enforcement evaluation time.
- `ai_demo_conflict_batch_duration_seconds` – total batch duration for multi-jurisdiction conflict simulations.
Additional gauges/counters:
- `ai_demo_decisions_store_rows` – current number of persisted decisions (updates on insert/prune) when DB enabled.
- `ai_demo_prune_operations_total` – number of prune delete operations (not rows) triggered by retention or age pruning.
- `ai_demo_jwks_fetch_total{result}` – JWKS fetch attempts labeled `success` or `error`.
- `ai_demo_jwks_keys_loaded` – current count of cached JWKS RSA public keys for RS256 verification.
 - `ai_demo_jwks_negative_hits_total` – number of times a previously missing `kid` was served from the negative cache (reducing network fetch attempts).
 - `ai_demo_jwks_negative_entries` – current number of entries in the negative KID cache awaiting expiry.
 - `ai_demo_jwks_negative_evictions_total` – total number of evictions from the negative KID cache due to `AGENTAUTH_AI_DEMO_JWKS_NEGATIVE_MAX_ENTRIES` limit.
 - `ai_demo_jwks_bg_refresh_total` – number of proactive background JWKS refresh operations before cache expiry.
 - `ai_demo_jwks_cache_ttl_remaining_seconds` – live gauge of seconds until JWKS cache expiry (updates each second).
- **PoA Metrics**:
  - `ai_demo_poa_validations_total{result}` – Attempts to validate PoA during `/demo/enforce` (results: success|revoked|expired|scope_mismatch|not_found)
  - `ai_demo_poa_revocations_total` – Count of processed revocations.
  - `ai_demo_poa_integrity_failures_total{reason}` – Integrity binding failures for extended token PoA claims (reasons: `digest_mismatch`, `version_mismatch`).
  - `ai_demo_poa_digest_mismatch_total` – Count of digest mismatch failures (redundant dimensional breakdown for dashboards).
  - `ai_demo_poa_version_mismatch_total` – Count of version mismatch failures.
  - **Multi-Signature Metrics**:
    - `ai_demo_poa_multisig_signatures_total{result}` – Signature submissions (currently only `accepted`; future labels may include `duplicate`, `unknown_signer`).
    - `ai_demo_poa_multisig_finalizations_total` – Successful finalizations of draft PoAs meeting quorum.
Note: With negative KID caching enabled, repeated invalid `kid` values won't inflate error fetch counts; only the initial miss prior to caching triggers a potential fetch (if cache refresh needed).
Monitoring guidance:
 - A rising `ai_demo_jwks_negative_hits_total` with stable `ai_demo_jwks_keys_loaded` may indicate misconfigured or stale clients sending nonexistent `kid` values.
 - Large sustained `ai_demo_jwks_negative_entries` suggests many distinct invalid `kid`s — consider alerting or adding request validation.
 - Frequent increases in `ai_demo_jwks_negative_evictions_total` mean the negative cache max size is being hit; consider raising `AGENTAUTH_AI_DEMO_JWKS_NEGATIVE_MAX_ENTRIES` or investigating malicious random kid spraying.
 - A low `ai_demo_jwks_bg_refresh_total` combined with frequent on-demand fetch errors may indicate refresh factor misconfiguration (increase `AGENTAUTH_AI_DEMO_JWKS_BG_REFRESH_FACTOR`).
 - Sudden drops in `ai_demo_jwks_cache_ttl_remaining_seconds` without corresponding `ai_demo_jwks_bg_refresh_total` increments may signal manual interference or clock skew issues.
 - High churn in negative entries hitting eviction threshold (see `AGENTAUTH_AI_DEMO_JWKS_NEGATIVE_MAX_ENTRIES`) could point to malicious spraying of random `kid` values—consider rate limiting.
Enable tracing:
```bash
export AGENTAUTH_AI_DEMO_OTEL=1
go run ./examples/ai_capability_demo
```
Spans emitted:
- `demo.enforce` for each enforcement via `/demo/enforce`
- `simulate.conflict.batch` aggregating jurisdiction decisions for conflict simulation
- `simulate.conflict.jurisdiction` child spans per jurisdiction with `jurisdiction` + `allowed` attributes

Attributes include `action`, `allowed`, `missing_count`, and per-jurisdiction `allowed` flags plus `conflict` outcome.
The stdout exporter prints spans in prettified JSON to the console.

## Customizing Scenarios
Edit `main.go` and adjust claim maps:
- Add compliance flags (e.g., `eu_ai_conformity`, `human_oversight`)
- Change `risk_level` to test escalated audit requirements
- Inject or omit framework booleans (e.g., `hipaa_compliance`, `sox_compliance`)

## Audit & Metrics Hooks
The integration sets callbacks:
- `SetAuditCallback(action, metadata)` – prints audit decisions
- `SetMetricsCallback(metric)` – prints metric increments
These can be redirected to real systems (SIEM, Prometheus push gateway, etc.).

## Extending the Demo
Ideas:
- Persist decisions to a lightweight SQLite DB
- Add JWT / API key auth for enforcement endpoints
- Simulate multi-jurisdiction conflict resolution across policy sets
- Emit OpenTelemetry spans for capability evaluation
- Expose Prometheus metrics for decisions & jurisdiction conflicts (`AGENTAUTH_AI_DEMO_METRICS=1`)
- Integrate Power of Attorney (delegation) records and revocation lifecycle
- Issue extended tokens referencing PoA for delegated actions
- Bind extended tokens to PoA integrity via canonical digest and version (implemented)
- Add persistent BoltDB PoA storage (implemented via `AGENTAUTH_AI_DEMO_POA_DB_PATH`)
- Expose integrity failure metrics (implemented)

## Troubleshooting
| Symptom | Cause | Fix |
|---------|-------|-----|
| 404 on capability endpoints | Routes not registered | Ensure `apiHandler.RegisterRoutes(router)` runs before `ListenAndServe` |
| Missing policies list | Governance loader returned empty | Check policy source configuration if integration evolves |
| All actions denied | Required base claims absent | Add `ai_entity_verified` and domain-specific compliance flags |
| PoA validation failing | PoA expired or scope mismatch | Check `valid_for_seconds`, ensure action is listed in PoA `scope` |
| Extended token rejected | `poa_id` missing or revoked | Re-issue PoA or verify revocation status |
| Multi-sig finalize rejected | Threshold not met | Add required signatures via `/demo/poa/:id/sign` before finalize |
| Multi-sig signature rejected | Duplicate or unknown signer | Ensure signer is in original `signers[]` list and not already signed |
| Server not starting | `AGENTAUTH_AI_DEMO_NO_SERVER=1` set | Unset env var or run without it |
| /metrics not found | Metrics disabled | Set `AGENTAUTH_AI_DEMO_METRICS=1` and restart |

## License / Usage
This demo is intended for evaluation and integration guidance. Do not use the hard-coded claims or simplistic compliance markers for production risk decisions. Replace with real attestation / evidence pipeline.

---
_Last updated: 2025-10-29 (Adds Multi-Signature Draft Workflow + Metrics)_
