# Model Limits Enforcement (sec11.item2)

## Overview
The model limits subsystem provides runtime safety and governance by enforcing configured constraints on model usage dimensions:

- Input token limit (`max_input_tokens`)
- Output token limit (`max_output_tokens`)
- Request rate per model (`max_requests_per_minute`)

Limits are loaded from a JSON file referenced by `GAUTH_MODEL_LIMITS_PATH` at server startup:

```json
{
  "model_limits": {
    "demo-model": {
      "max_input_tokens": 200,
      "max_output_tokens": 150,
      "max_requests_per_minute": 5
    },
    "other-model": {
      "max_input_tokens": 8192
    }
  }
}
```

Per-user scoped quotas extend the schema with an optional `user_limits` object:

```json
{
  "model_limits": { "demo-model": { "max_input_tokens": 200 } },
  "user_limits": {
    "demo-model": {
      "alice": { "max_input_tokens": 100, "max_output_tokens": 80, "max_requests_per_minute": 2 },
      "bob":   { "max_input_tokens": 50 }
    }
  }
}
```
If present, per-user values override the model-level limits for that user only (dimension by dimension). Dimensions omitted or set <=0 fall back to the model-level limit (or disable enforcement if also unset globally).

Fields are optional; absent or non-positive values disable enforcement for that dimension.

## Enforcement Endpoint
`POST /api/v1/model/validate`
Request body:
```json
{
  "model_id": "demo-model",
  "input_tokens": 123,
  "output_tokens": 45
}
```
Responses:
- Success (limits satisfied or unknown model): `200 {"success":true,...}`
- Input limit exceeded: `400 {"error":"model_limit_exceeded"}`
- Output limit exceeded: `400 {"error":"model_output_limit_exceeded"}`
- Rate limit exceeded: `429 {"error":"model_rate_limit_exceeded"}`

Unknown models currently pass-through (`limit_enforced=false` implicit) — future strict mode flag will invert behavior.

## Metrics
Prometheus & memory metrics counters:
- `model_limit_exceeded_total` – input token limit violations
- `model_output_limit_exceeded_total` – output token limit violations
- `model_rate_limit_exceeded_total` – per-minute rate limit violations
- `model_user_input_limit_exceeded_total` – per-user input token limit violations
- `model_user_output_limit_exceeded_total` – per-user output token limit violations
- `model_user_rate_limit_exceeded_total` – per-user per-minute rate limit violations

Decision labeling also records `decision_total{action="model_validate",resource=<model_id>,outcome=allow|deny}` for all paths.

## Internal Data Structures
Added to `BetaServer`:
- `modelLimits map[string]int` (input tokens)
- `modelOutputLimits map[string]int` (output tokens)
- `modelRateLimits map[string]int` (per-minute rate)
- `modelRateState map[string]{WindowStart, Count}` per-model rolling 60s window
- `modelUserLimits map[string]map[string]{InputLimit, OutputLimit, RateLimit}` compound model+user governance map
- `modelUserRateState map[string]map[string]{WindowStart, Count}` per-user rolling 60s windows

Concurrency protected by fine-grained mutexes (`modelOutputLimitsMu`, `modelRateMu`, `modelRateStateMu`).

## Remaining Gaps (Partial Status)
Implemented:
- Multi-dimension model limits (input/output/rate)
- Per-user scoped quotas (input/output/rate overrides) with new metrics counters
- Audit chain (GAUTH_MODEL_LIMIT_AUDIT_PATH) + verification endpoint `/api/v1/model/limits/audit/verify`
- Periodic anchor chain (GAUTH_MODEL_LIMIT_ANCHOR_PATH + GAUTH_MODEL_LIMIT_ANCHOR_INTERVAL) + verification endpoint `/api/v1/model/limits/audit/anchor/verify`
- Dynamic reload (GAUTH_MODEL_LIMITS_RELOAD_INTERVAL seconds polling) for live config adjustments
 - Snapshot endpoint `/api/v1/model/limits/snapshot` exposing ordered limits + canonical hash for drift detection
 - Strict unknown-model rejection mode (GAUTH_MODEL_LIMITS_STRICT_UNKNOWN=1)

Still pending:
- External notarization / transparency publication of anchor records
- Fuzz/property tests for loader, rate window rollover, audit chain tamper cases
- Discovery document hash exposure (exposed via snapshot endpoint hash; external publication pending)
- Anomaly detection (surge in exceed events) integration
- Metrics toggle for strict unknown-mode (currently only decision_total increments)

## Roadmap
1. Introduce `model_limit_audit.jsonl` append-only log with hash chain `prev_hash -> hash`.
2. Export `/api/v1/model/limits` snapshot with canonical hash for governance drift detection.
3. Implement per-user quota plugin (interface `ModelQuotaProvider`) with default memory provider.
4. Add fuzz tests: malformed JSON, extreme values, negative numbers, concurrent loads.
5. Integrate anomaly detector (EWMA) for rate & token exceed events; surface z-score metrics.

## Testing
Current tests:
- `web/model_limits_test.go` – basic input limit and unknown model pass-through.
- `web/model_limit_metric_test.go` – input exceed increments dedicated metric.
- `web/model_limits_extended_test.go` – output & rate limit enforcement.
- `web/model_user_limits_test.go` – per-user quota overrides and metrics counters.
- `web/model_limit_anchor_test.go` – periodic anchor chain verification.
- `web/model_limits_reload_test.go` – dynamic reload tightening behavior.
 - `web/fuzz_model_limits_parse_test.go` – fuzz harness for JSON parsing robustness (run with `go test -fuzz=FuzzModelLimitsParse`).
 - `web/fuzz_model_limit_audit_chain_test.go` – fuzz harness for audit chain writer & verifier (run with `go test -fuzz=FuzzModelLimitAuditChain`).

## Security Considerations
- Enforced limits mitigate runaway cost / exposure.
- Pending persistence & anchoring needed for tamper-evident governance.
- Rate limiting currently naive (single window); future sliding window / leaky bucket recommended.

## Extensibility
Future dimensions may include:
- `max_daily_requests`
- `max_concurrent_requests`
- `max_output_tokens_per_minute`
- Per-user adaptive budgets using anomaly feedback loops.

## Environment Variables Summary
| Variable | Purpose |
|----------|---------|
| `GAUTH_MODEL_LIMITS_PATH` | Path to limits JSON file loaded at startup (and reloaded if interval set). |
| `GAUTH_MODEL_LIMITS_RELOAD_INTERVAL` | Seconds between polling for modified limits file (mtime change triggers atomic swap). Omit or 0 to disable. |
| `GAUTH_MODEL_LIMIT_AUDIT_PATH` | Path to audit chain JSONL for exceed events. |
| `GAUTH_MODEL_LIMIT_ANCHOR_PATH` | Path to periodic anchor chain JSONL. |
| `GAUTH_MODEL_LIMIT_ANCHOR_INTERVAL` | Emit anchor every N audit entries (default 100). |

## Snapshot Endpoint
`GET /api/v1/model/limits/snapshot`
Response:
```
{
  "success": true,
  "hash": "sha256:...",
  "generated_at": "2025-10-22T12:34:56.789Z",
  "model_limits": [
    {"model_id":"demo-model","max_input_tokens":200,"max_output_tokens":150,"max_requests_per_minute":5}
  ],
  "user_limits": [
    {"model_id":"demo-model","user_id":"alice","max_input_tokens":100}
  ]
}
```
Canonical hash is computed over a deterministic JSON (sorted model IDs then user IDs) enabling drift detection and external attestation alignment with anchor chain.

## Strict Unknown-Model Mode
Set `GAUTH_MODEL_LIMITS_STRICT_UNKNOWN=1` to reject any validation request referencing a model ID not present (or with non-positive limits) in the current configuration. Response:

`400 {"success":false,"error":"model_unknown","model_id":"<id>"}`

Metrics: the request is recorded as a `decision_total{action="model_validate",resource="<id>",outcome="deny"}`. No dedicated counter yet (future enhancement if observability gap identified).
As of current build, a dedicated Prometheus counter `gauth_rfc0111_model_unknown_total` is emitted (incremented per denied request) enabling direct alerting on unexpected model traffic.

## Attestation Endpoint
Endpoint: `GET /api/v1/model/limits/attestation`

Combines governance evidence components into a single JSON payload for external verifiers:

```
{
  "success": true,
  "configured": true,
  "snapshot": {"hash": "sha256:...", "generated_at": "2025-10-22T20:00:00Z"},
  "audit": {"head_hash": "sha256:...", "entries": 42},
  "anchor": {"latest_hash": "sha256:...", "entries": 5, "interval": 100},
  "strict_unknown": true
}
```

Field semantics:
- `snapshot.hash`: Deterministic hash of current limits & per-user overrides.
- `audit.head_hash` / `audit.entries`: Head hash & line count of exceed audit hash-chain.
- `anchor.latest_hash` / `anchor.entries`: Head hash & line count of anchor chain (commitments every N entries).
- `anchor.interval`: Configured anchor cadence (entries between anchors).
- `strict_unknown`: Indicates enforcement of unknown-model rejection.

If audit / anchor disabled: `configured` false with `reason` field.

Integrity model: Attestors record `(snapshot.hash, audit.head_hash, anchor.latest_hash)` periodically along with timestamps or external notarization receipts enabling later drift or tamper detection.

### Optional Cryptographic Signature
Enable attestation signing by setting:

```
GAUTH_TOKEN_SIG_MODE=eddsa
GAUTH_MODEL_LIMIT_ATTEST_SIGN=1
```

When active, three additional root fields are emitted:

```
"signature": "<base64_raw_ed25519>",
"sig_kid": "<key_id>",
"sig_mode": "eddsa"
```

Signature coverage: raw canonical JSON bytes of the attestation object with the three signature fields omitted (stable ordering via struct fields; no maps used). Verification procedure:

1. Fetch attestation JSON.
2. Parse into a struct (or map), capture `signature`, `sig_kid`, `sig_mode`.
3. Reconstruct an identical JSON serialization excluding `signature`, `sig_kid`, `sig_mode` (exact field ordering must match original struct order; easiest is to define the same struct layout locally and marshal after zeroing those fields).
4. Decode `signature` (Base64 Raw URL / Std? Uses Raw Std Encoding without padding) and verify with Ed25519 public key referenced by `sig_kid`.

Example signed payload (truncated hashes):

```
{
  "success": true,
  "configured": true,
  "snapshot": {"hash": "sha256:abc...", "generated_at": "2025-10-22T20:00:00Z"},
  "audit": {"head_hash": "sha256:def...", "entries": 42},
  "anchor": {"latest_hash": "sha256:ghi...", "entries": 5, "interval": 100},
  "strict_unknown": true,
  "signature": "3J1k...",
  "sig_kid": "AbCdEf12",
  "sig_mode": "eddsa"
}
```

Tamper detection: modifying any governed field (e.g., `audit.head_hash`) invalidates the signature.

Future enhancements:
- External timestamping / notarization of attestation hash triple.
- Multi-signature threshold (M-of-N key support).
- Streaming of new signed attestations via SSE / WebSocket.
 - Public publication pipeline for attestation history (append-only receipt log exposed).

### Surge & Notarization Fields
When enabled with `GAUTH_MODEL_LIMIT_SURGE_FACTOR` / `GAUTH_MODEL_LIMIT_SURGE_MIN_EVENTS`, the attestation may include a `surge` object if a surge was detected in the last few seconds:

```
"surge": {
  "model_id": "demo",
  "last_10s_exceed_events": 18,
  "avg_active_seconds": 3.2,
  "factor": 3.0,
  "min_events": 5,
  "triggered": true,
  "triggered_at": "2025-10-22T21:11:05Z"
}
```

Notarization of the attestation triple (snapshot hash, audit head, anchor head) is enabled with:

```
GAUTH_MODEL_LIMIT_ATTEST_NOTARIZE=1
GAUTH_CAP_ANCHOR_NOTARIZE=1   # ensures prototype notarizer initialization
```

Produces a `notarization` object:

```
"notarization": {
  "provider": "memory",
  "timestamp": "2025-10-22T21:11:05Z",
  "latency_seconds": 0.00012,
  "success": true
}
```

Hash submitted = `sha256` over `attest|<snapshot_hash>|<audit_head>|<anchor_head>` (domain separation). External verifiers can re-hash and compare against trusted receipt log.

### Verification & Key Discovery
Endpoints:

- `GET /api/v1/model/limits/attestation/keys` — lists active + historical Ed25519 public keys used for signing attestations. Response:
  `{"success":true,"keys":[{"kid":"AbCd...","public_b64":"<base64>"}]}`

- `POST /api/v1/model/limits/attestation/verify` — submit a full attestation JSON (as previously returned) to obtain signature validation and recomputed combined hash triple digest.
  Response example:
  `{"success":true,"valid":true,"kid":"AbCd...","sig_mode":"eddsa","combined_hash":"sha256:..."}`

Client-side verification (manual):
1. Fetch attestation and keys endpoints.
2. Build unsigned canonical JSON excluding `signature`, `sig_kid`, `sig_mode` maintaining field order.
3. Ed25519 verify using `public_b64` for matching `kid`.
4. Recompute combined hash seed `attest|snapshot.hash|audit.head_hash|anchor.latest_hash` and `sha256` it; compare with local expectation or notarized receipt record.

Security considerations:
- Key rotation continuity handled via `kid` stability; historical keys remain available until expiry.
- Combined hash domain separation (`attest|`) prevents collision with other hash contexts.
- Verification endpoint is optional convenience; offline re-verification preferred for air-gapped auditors.

### Attestation Streaming (SSE)

Endpoint: `GET /api/v1/model/limits/attestation/stream` (gated by `GAUTH_ATTEST_STREAM_ENABLE=1`).

Behavior:
- Connection establishes a Server-Sent Events channel (`Content-Type: text/event-stream`).
- Immediately emits an `attestation` event containing a freshly built attestation JSON (including signature / notarization / surge fields if enabled by respective environment flags).
- Heartbeat comment (`: ping`) every ~15s keeps intermediaries from timing out idle streams; each heartbeat triggers a lightweight background refresh emission so auditors receive updated heads even without explicit triggers.
- Clients auto-reconnect with suggested 5s backoff (`retry: 5000`).

Event format:
```
event: attestation
data: {"success":true,"configured":true,"snapshot":{...},"audit":{...},"anchor":{...},"strict_unknown":true,"signature":"..."}
```

Fields match the REST attestation endpoint. A transient `reason` field may be added in future (e.g., `open`, `heartbeat`, `surge_trigger`, `audit_append`) to annotate emission causality.

Current trigger reasons (when `GAUTH_ATTEST_STREAM_ENABLE=1`):
- `open` – initial connection emission
- `heartbeat` – periodic refresh (every ~15s)
- `audit_append` – a new exceed audit chain entry appended (updates audit head)
- `anchor_commit` – anchor chain commit at configured interval boundary
- `surge_trigger` – surge detection condition met (exceed event spike)

Metric:
`attestation_stream_emissions_total{reason="<reason>"}` – monotonic counter exposed alongside semantic metrics (Prometheus textual endpoint) tracking total SSE attestation emissions by reason.

Enabling:
```
GAUTH_ATTEST_STREAM_ENABLE=1
GAUTH_MODEL_LIMIT_ATTEST_SIGN=1            # optional signing
GAUTH_MODEL_LIMIT_ATTEST_NOTARIZE=1        # optional notarization (requires prototype notarizer)
```

Planned trigger expansion (future):
- Emit immediately on audit chain append (exceed event) to surface new head hash.
- Emit on anchor chain commit.
- Emit on surge detection trigger with `reason=surge_trigger`.
- Optional rate-based throttling (coalesce multiple changes within short window).

Client Guidance:
- Use browser EventSource or CLI tools (curl) to consume; verify each `attestation` event independently (signature + combined hash) to maintain a local timeline.
- Maintain last seen `(snapshot.hash, audit.head_hash, anchor.latest_hash)` triplet to detect retroactive tampering (unexpected head regressions).

Security & Integrity:
- Stream uses same canonical JSON serialization path as REST endpoint (shared builder functions) ensuring identical signature verification semantics.
- Backpressure: per-subscriber buffered channel prevents global stalls; overflowing a slow subscriber drops intermediate events (client should reconnect promptly).
- No sensitive secret material is streamed; only governance evidence and optional signatures / notarization receipts.

Monitoring:
- Future metric `attestation_stream_emissions_total` will count emission reasons by label.
- Current build: no dedicated stream metric (roadmap item).


## Surge Detection (Experimental)
Environment variables:
- `GAUTH_MODEL_LIMIT_SURGE_FACTOR` (default 3.0): multiplier threshold. A surge triggers when last10s exceeds (average_nonzero * factor) and minimum events met.
- `GAUTH_MODEL_LIMIT_SURGE_MIN_EVENTS` (default 5): minimum exceed events in the last 10s window to evaluate surge condition.

Metric:
- `model_limit_exceed_surge_total` (Prometheus) increments when a surge condition triggers (cool-down 15s between triggers).

Windowing: maintains per-model per-second counts over a 60s rolling window. Last 10s slice compared to average of non-zero seconds to reduce cold-start influence.

Limitations / Future:
- Single global cool-down; per-model cool-down and severity levels (moderate/critical) pending.
- No external alert hook yet; integrate with anomaly subsystem / SSE stream next.
\n+As of current build, a dedicated Prometheus counter `gauth_rfc0111_model_unknown_total` is emitted (incremented per denied request) enabling direct alerting on unexpected model traffic.

---
Status: Partial (multi-dimension enforcement active, persistence & advanced governance pending)
