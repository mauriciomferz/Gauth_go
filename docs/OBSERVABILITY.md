---
title: Observability
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

## Decision Metrics
## Tracing (RB9) & Latency Percentiles (NEW)
The RB9 observability phase introduces lightweight in-repo tracing plus a consolidated latency percentile endpoint.

Enabling Tracing:
- Set `GAUTH_TRACING_ENABLED=1` (preferred) OR legacy `GAUTH_OTEL_ENABLE=1`.
- Optional sampling ratio: `GAUTH_TRACING_SAMPLE_RATIO` in `[0,1]`. A value of `0` or unset defaults to always sample; any value in `(0,1]` applies probabilistic sampling (uniform `rand.Float64()<ratio`).

Emitted Span Operations:
| Operation | Description | Key Tags |
|-----------|-------------|----------|
| `token.issue` | Token creation endpoint | `kid`, `subject`, `scopes_count` |
| `token.validate` | Token validation endpoint | `kid`, `subject`, `digest_version`, `replay_hit` (bool) |
| `attestation.verify` | Model limits attestation verification | `valid` (bool), `failure_code`, `kid` |
| `rotation.perform` | Ed25519 key rotation (manual or scheduled) | `prev_kid`, `new_kid`, `ttl_hours`, `history_size`, `error` (on failure) |
| `rotation.append` | Immutable rotation ledger append emission | `prev_kid`, `new_kid`, `new_key_set_size`, `error` (on failure) |

Error Path Coverage:
- Body read failure and JSON parse failure in attestation verify set `error` tag and end span early.
- Rotation failures tag `error` before span end.

Latency Percentiles Endpoint:
`GET /api/v1/beta/metrics/latency` returns approximate p50/p95/p99 for selected Prometheus histograms using bucket scans (upper bound of first bucket exceeding the quantile threshold). Shape:
```json
{
	"success": true,
	"generated_at": "2025-10-27T02:55:00.000000Z",
	"histograms": {
		"attestation_verify": {"p50": 0.0005, "p95": 0.002, "p99": 0.005, "count": 42},
		"rotation_summary": {"p50": -1, "p95": -1, "p99": -1, "count": 0},
		"rfc0111_validation": {"p50": 0.001, "p95": 0.006, "p99": 0.010, "count": 310}
	}
}
```
- `-1` indicates no observations yet (histogram empty).
- Percentiles are approximate (no interpolation within bucket). For precise Prometheus calculations use `histogram_quantile()` over `_bucket` series.

Alerting Example (PromQL):
```
histogram_quantile(0.95, sum by (le)(rate(gauth_attestation_verify_latency_seconds_bucket[5m]))) > 0.050
```
Use the JSON endpoint for quick UI snapshots; prefer PromQL for dashboards / SLO tracking.

Future Enhancements:
- Add token validation latency histogram (`gauth_token_validation_latency_seconds`) once implemented and include it automatically.
- Add richer attestation outcome counters (`gauth_attestation_verify_total`) sliced by soft_invalid classification for governance dashboards.
- Emit trace-to-metric correlation IDs for deeper latency root cause drill-down.

Backward Compatibility: The new endpoint is additive and does not alter existing metric names or semantics.

## WAL Metrics (Replay Durability)

The replay nonce store (token issuance & attestation replay protection) supports optional write-ahead log (WAL) durability.

Environment Variables:
* `GAUTH_REPLAY_WAL` – enable WAL for token issuance replay store.
* `GAUTH_ATTEST_REPLAY_WAL_PATH` – enable WAL for attestation replay store.

Durable Path Behavior:
1. On startup, existing WAL file is replayed; malformed lines are skipped (counted via `replay_store_errors_total`).
2. Active nonces are held in memory; TTL expiration pruning occurs lazily on access.
3. Compaction (`SnapshotAndCompact()`): writes snapshot `<wal>.snapshot`, rotates (truncates) WAL, re-appends active entries.

Metrics Surface (Memory adapter):
* `replay_wal_pending` (gauge) – number of active entries considered pending during flush (Prometheus adapter: noop placeholder today).
* `replay_wal_flush_latency` – recorded as part of replay store latency histogram until a dedicated histogram is added.

Prometheus (current state):
* Flush latency samples appear under `gauth_rfc0111_replay_store_latency_seconds` (multiplexed). Future version will add `gauth_rfc0111_replay_wal_flush_latency_seconds` and `gauth_rfc0111_replay_wal_pending` gauge.

Alert Rule Examples:
```
ALERT ReplayStoreHighFlushLatency
	IF histogram_quantile(0.95, sum(rate(gauth_rfc0111_replay_store_latency_seconds_bucket[5m])) by (le)) > 0.25
	FOR 5m
	LABELS { severity = "warning" }
	ANNOTATIONS { summary = "High WAL flush latency", description = "95th percentile flush latency >250ms" }

ALERT ReplayStoreErrorSurge
	IF increase(gauth_rfc0111_replay_store_errors_total[5m]) > 25
	FOR 2m
	LABELS { severity = "critical" }
	ANNOTATIONS { summary = "Replay store error surge", description = "Errors exceed 25 in 5m window" }
```

Operational Guidance:
* Investigate high flush latency for disk IO contention or large active set; consider increasing snapshot cadence.
* Error surges often indicate WAL corruption; rotate WAL (archive & recreate) then monitor error count stabilization.
* Snapshot Age (future metric) should remain < configured RPO (e.g. 6h). Until exposed, derive externally from file mtime.

Roadmap:
* Dedicated Prometheus gauge + histogram.
* Snapshot age & last flush timestamp metrics.
* Optional automatic periodic compaction based on size or age thresholds.

## Delegation Depth Enforcement (RB12)

RB12 introduces an optional maximum delegation chain length enforced at append time.

Environment Variable:
* `GAUTH_MAX_DELEGATION_DEPTH` – positive integer; if set and >0, attempts to append a delegation that would increase chain length beyond this value are rejected with error code `delegation_depth_exceeded`.

Metrics:
* `delegation_depth_exceeded_total` (counter) – increments for each rejected append due to depth.
* Optional future gauge `delegation_max_depth_observed` – highest depth seen since start (not yet implemented).

Discovery Surface:
`GET /api/v1/discovery` now includes `max_delegation_depth` when the environment variable is set (absent or null when disabled). Clients can proactively bound delegation creation attempts without trial failures.

Alert Examples (PromQL):
```
ALERT DelegationDepthExceededSpike
	IF increase(gauth_delegation_depth_exceeded_total[30m]) > 5
	FOR 10m
	LABELS {severity="warning"}
	ANNOTATIONS {summary="Delegation depth limit exceed spike", description="More than 5 exceed events in 30m"}
```

Operational Notes:
* Depth counting uses chain length (genesis = 1). Clarify off‑by‑one semantics in any consumer dashboards.
* To stage rollout: leave env unset (disabled) while observing natural depths, then set limit slightly above p99 observed depth.
* Set to a high temporary value (e.g. 100) to validate error taxonomy path before lowering.

Roadmap:
* Add max observed depth gauge.
* Include depth statistics in a governance diagnostics endpoint.
* Soft warning mode (`GAUTH_DELEGATION_DEPTH_ENFORCE_STRICT=0`) prior to hard enforcement.


### Decision Counters
Existing decision counters track total decisions, allows, denies, and expression evaluation errors. To enable richer dimensional analysis (gap: action/resource labeling) the system now records labeled decision counters via the metrics interface (`RecordDecision(action, resource, outcome)`) and optional reason-enriched counters (`RecordDecisionWithReason(action, resource, outcome, reason)`).

Prometheus:

```text
# HELP gauth_decisions_total Total authorization decisions by action/resource/outcome.
# TYPE gauth_decisions_total counter
gauth_decisions_total{action="read",resource="report:finance",outcome="allow"} 42
gauth_decisions_total{action="write",resource="report:finance",outcome="deny"} 5

# HELP gauth_decision_reasons_total Authorization decisions labeled with reason taxonomy.
# TYPE gauth_decision_reasons_total counter
gauth_decision_reasons_total{action="write",resource="report:finance",outcome="deny",reason="default_deny"} 5
```

JSON snapshot (excerpt) may expose aggregated decision sets (future expansion) but labeled counts are primarily for Prometheus & OTEL. Empty strings normalized to `_` sentinel.

OpenTelemetry: if enabled (GAUTH_OTEL_METRICS_ENABLE=1) the adapter emits `gauth.decisions` and `gauth.decision.reasons` counters with identical label dimensions.
## Violation Counters (Beta)

The token validation path now categorizes rejection reasons and increments in-memory counters:

Categories: `sig_invalid`, `expired`, `not_yet_valid`, `issuer_mismatch`, `replay_detected`, `audience_mismatch`, `missing_claim`, `unknown`.

Implementation: `internal/observability/violations.go` provides atomic counters. They are wired into `pkg/gauth/gauth.go` via `failMetric` helper. Tests: `pkg/gauth/gauth_violation_metrics_test.go` exercises several failure categories.

Implemented Endpoint:
`GET /api/v1/beta/metrics/violations` returns a JSON payload:
```json
{
	"success": true,
	"timestamp": "2025-10-19T01:13:14.123456Z",
	"counters": {
		"sig_invalid": 12,
		"expired": 3,
		"not_yet_valid": 0,
		"issuer_mismatch": 1,
		"replay_detected": 4,
		"audience_mismatch": 2,
		"missing_claim": 7,
		"unknown": 0
	},
	"total": 29,
	"categories": ["sig_invalid","expired","not_yet_valid","issuer_mismatch","replay_detected","audience_mismatch","missing_claim","unknown"]
}
```
All category keys are always present (zero when unused). `total` is the sum of all counters. The endpoint is lightweight and safe to poll frequently (no locking contention beyond atomic loads).


	## External Anchoring Forced Failures (NEW)

	Deterministic external anchoring tests and controlled chaos experiments sometimes need to guarantee a fixed number of initial failures before allowing the normal probabilistic model to apply. The environment variable:

	```
	GAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS=<N>
	```

	forces the first `N` external anchoring attempts to fail ("stub failure (forced)") regardless of `GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB`. After the forced failures are consumed, the configured probability model resumes.

	To distinguish these deterministic injected failures from organic (probability-driven) failures, two new Prometheus counters are exposed:

	```
	# HELP gauth_rfc0111_external_anchor_forced_failures_total Forced external capability anchoring failures (deterministic override before probabilistic model)
	# TYPE gauth_rfc0111_external_anchor_forced_failures_total counter
	gauth_rfc0111_external_anchor_forced_failures_total 1

	# HELP gauth_rfc0111_external_anchor_forced_failures_provider_total Forced external capability anchoring failures labeled by provider
	# TYPE gauth_rfc0111_external_anchor_forced_failures_provider_total counter
	gauth_rfc0111_external_anchor_forced_failures_provider_total{provider="tsa-stub"} 1
	```

	Behavioral Relationships:
	- Forced failures are a strict subset of total failures (`external_anchor_failures_total`). Each forced failure increments both counters so existing failure alerting continues to work without changes.
	- Provider-labeled forced failures mirror the provider dimension used for attempts/failures/latency.

	Suggested PromQL to isolate only organic failures (excluding deterministic injects):

	```
	organic_failures = gauth_rfc0111_external_anchor_failures_total - gauth_rfc0111_external_anchor_forced_failures_total
	rate_organic_failures_5m = increase(gauth_rfc0111_external_anchor_failures_total[5m]) - increase(gauth_rfc0111_external_anchor_forced_failures_total[5m])
	```

	Provider Breakdown (organic only):
	```
	sum by (provider) (increase(gauth_rfc0111_external_anchor_failures_provider_total[5m])) -
	sum by (provider) (increase(gauth_rfc0111_external_anchor_forced_failures_provider_total[5m]))
	```

	Alert Example (ignore forced failures spikes used in test pipelines):
	```
	ALERT ExternalAnchorOrganicFailureSpike
	  IF (increase(gauth_rfc0111_external_anchor_failures_total[10m]) - increase(gauth_rfc0111_external_anchor_forced_failures_total[10m])) > 25
	  FOR 5m
	  LABELS {severity="high"}
	  ANNOTATIONS {summary="External anchoring organic failure spike", description="Organic external anchor failures >25 in 10m (forced failures excluded)."}
	```

	Testing Guidance:
	- Set `GAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS=1` with `GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB=0` to validate success path latency + forced failure counters deterministically.
	- Combine with `GAUTH_CAP_EXTERNAL_ANCHOR_RAND_SEED` to stabilize latency histogram sample counts.

	Backward Compatibility:
	- Existing consumers of `gauth_rfc0111_external_anchor_failures_total` require no changes; forced failures seamlessly appear in aggregate totals.
	- The Memory metrics adapter exposes only aggregate counts; forced failure separation is Prometheus‑only at this stage (future memory separation can be added if needed for JSON endpoints).

	Roadmap:
	- Expose `organic` failures as a computed gauge to remove the need for subtraction in common dashboards.
	- Add a ratio metric `forced_failure_ratio` for quick detection of excessive test configuration leakage into production.

Prometheus Exposition (NEW):
`GET /api/v1/beta/metrics/violations/prometheus` returns text format metrics:
```
# HELP gauth_validation_total Total token validation failure events across all categories
# TYPE gauth_validation_total counter
gauth_validation_total 29
# HELP gauth_validation_category_token_failures Token validation failures by category
# TYPE gauth_validation_category_token_failures counter
gauth_validation_sig_invalid_total 12
gauth_validation_expired_total 3
gauth_validation_not_yet_valid_total 0
gauth_validation_issuer_mismatch_total 1
gauth_validation_replay_detected_total 4
gauth_validation_audience_mismatch_total 2
gauth_validation_missing_claim_total 7
gauth_validation_unknown_total 0
```
Metric Naming Pattern: `gauth_validation_<category>_total`. A rolled-up aggregate `gauth_validation_total` is also provided for quick ratio calculations.

### Anomaly Detection (Rolling Window)

`GET /api/v1/beta/metrics/violations` now includes an `anomaly` object:

```json
"anomaly": {
	"rate_per_minute_60s": 42.7,
	"rate_per_minute_300s": 18.4,
	"delta_60s": 45,
	"delta_300s": 92,
	"window_60s_seconds": 53,
	"window_300s_seconds": 210,
	"surge_60s": true,
	"surge_threshold_per_minute": 40
}
```
Fields:
- `rate_per_minute_60s` / `rate_per_minute_300s`: Approximate failure event rates normalized per minute using earliest counter inside the window.
- `delta_60s` / `delta_300s`: Raw counter growth over the window.
- `window_*_seconds`: Effective duration since earliest sample used (can be < nominal window immediately after startup).
- `surge_60s`: Boolean flag when 60s rate exceeds configured threshold.
- `surge_threshold_per_minute`: Threshold value (default 100 unless overridden).

Configuration:
- `GAUTH_VIOLATION_SURGE_THRESHOLD` sets custom per-minute surge threshold.
- `GAUTH_VIOLATION_PERSIST_PATH` enables persistence (JSON file). On restart, counters + recent history (<5m) are restored.
- `GAUTH_VIOLATION_AUTOSAVE_SEC` enables periodic autosave (>=10 seconds). Manual saves occur on validation attempts (throttled 5s).
- `GAUTH_VIOLATION_PERSIST_NO_THROTTLE=1` disables the 5s write throttle (use only for tests / debugging).

Persistence File Schema (Hash Chain Wrapped – NEW):
```
## Policy Governance Metrics (NEW)

Two new counters expose governance operations and inspection activity. These complement existing revision & active version metrics.

Prometheus Names:

```text
# HELP gauth_policy_rollback_total Total successful policy rollback operations
# TYPE gauth_policy_rollback_total counter
gauth_policy_rollback_total 5

# HELP gauth_policy_diff_requests_total Total successful diff requests
# TYPE gauth_policy_diff_requests_total counter
gauth_policy_diff_requests_total 42
```

JSON Endpoint: `GET /api/v1/beta/policy/metrics`
Fields added:
```json
{
	"revisions": 17,
	"active_version": 16,
	"rollback_count": 5,
	"diff_requests": 42
}
```

### Operational Dashboards

| Objective | PromQL Example | Notes |
|-----------|----------------|-------|
| Excessive rollbacks | increase(gauth_policy_rollback_total[1h]) > 3 | Indicates instability or high correction rate |
| Diff usage rate | rate(gauth_policy_diff_requests_total[5m]) | Correlate with change windows |
| Rollbacks per revision | gauth_policy_rollback_total / gauth_policy_revisions_total | Ratio > 0.2 may warrant review |
| Pre‑change inspection efficacy | rate(gauth_policy_diff_requests_total[30m]) / increase(gauth_policy_revisions_total[30m]) | Low ratio suggests insufficient diff review |

### Alert Suggestions

```text
ALERT PolicyRollbackSpike
	IF increase(gauth_policy_rollback_total[30m]) > 2
	FOR 10m
	LABELS {severity="warning"}
	ANNOTATIONS {summary="Policy rollback spike", description="More than 2 rollbacks in 30m"}

ALERT MissingDiffReview
	IF (increase(gauth_policy_revisions_total[2h]) > 5) AND (increase(gauth_policy_diff_requests_total[2h]) < 3)
	FOR 15m
	LABELS {severity="info"}
	ANNOTATIONS {summary="Low diff review activity", description="Fewer than 3 diff inspections for >5 revisions in 2h"}
```

### Forensic Correlation
Sequence a suspected incident by combining:
1. `gauth_policy_revisions_total` increments (policy changes)
2. `gauth_policy_diff_requests_total` bursts (inspection)
3. `gauth_policy_rollback_total` increments (remediation)
4. Audit entries (`/api/v1/beta/audit`) for authoritative timeline.

### OTEL Roadmap
Future instrumentation will add Int64 counters `gauth.policy.rollback` and `gauth.policy.diff.requests` when `GAUTH_OTEL_METRICS_ENABLE=1`. This enables unified tracing + metrics correlation (e.g., rollback spans annotated with diff inspection count preceding the action).

Cross‑Reference: See `artifacts/report.md` Governance Feature Report for broader context and implementation details.
{
	"payload": {
		"counters": { "sig_invalid": 12, "expired": 3, ... },
		"history": [ { "at": "2025-10-19T01:27:05.123456Z", "total": 29 }, ... ]
	},
	"prev_hash": "<previous file hash or empty on first>",
	"hash": "<sha256(prev_hash || payload-bytes)>",
	"timestamp": "2025-10-19T01:27:05.999999Z"
}
```
Only the last ~120 history checkpoints are retained for compactness. Older entries (>5m) are pruned on load. Legacy (pre-chain) files *without* the wrapper are still accepted on restore for backward compatibility.

Hash Computation:
```
chain_input = prev_hash + raw_payload_json_bytes
hash = sha256(chain_input)
```
This forms a simple append-only hash chain across successive persistence snapshots enabling tamper detection.

Integrity Verification Endpoints (NEW):

| Endpoint | Purpose | Responses |
|----------|---------|-----------|
| `GET /api/v1/beta/metrics/violations/verify` | Verify violation counters persistence integrity | `{integrity:"ok"|"mismatch"|"legacy"}` |
| `GET /api/v1/beta/metrics/poa/semantics/verify` | Verify semantic counters persistence integrity | Same schema |

Sample OK Response:
```json
{
	"success": true,
	"configured": true,
	"integrity": "ok",
	"details": {
		"expected": "804ba1425be9...",
		"recomputed": "804ba1425be9...",
		"prev_hash": "0e3558e7470d..."
	}
}
```
Sample Tamper (Mismatch) Response:
```json
{
	"success": true,
	"configured": true,
	"integrity": "mismatch",
	"details": {
		"expected": "804ba1425be9...",
		"recomputed": "6c1d4ef9921c...",
		"prev_hash": "0e3558e7470d..."
	}
}
```
Legacy files (without wrapper) return `integrity:"legacy"` and SHOULD be rotated forward (write a new snapshot) to enroll into the chain.

Prometheus (Anomaly Metrics Additions):
```
# HELP gauth_validation_rate_per_minute_60s Approximate failure events per minute over last 60s
gauth_validation_rate_per_minute_60s 42.70
# HELP gauth_validation_rate_per_minute_300s Approximate failure events per minute over last 300s
gauth_validation_rate_per_minute_300s 18.40
# HELP gauth_validation_surge_60s Surge indicator (1 if 60s rate exceeds threshold)
gauth_validation_surge_60s 1
# HELP gauth_validation_surge_threshold_per_minute Configured surge threshold per minute
gauth_validation_surge_threshold_per_minute 40
# HELP gauth_poa_semantic_anomaly_score EWMA-based standardized anomaly score for 60s semantic rejection rate (z-score)
gauth_poa_semantic_anomaly_score{category="amount_limit_exceeded"} 2.4
gauth_poa_semantic_anomaly_score{category="daily_amount_limit_exceeded"} 0
gauth_poa_semantic_anomaly_score{category="currency_mismatch"} -0.1
gauth_poa_semantic_anomaly_score{category="scope_violation"} 0.3
gauth_poa_semantic_anomaly_score{category="restriction_mismatch"} 0
```

Alert Hint (Prototype):
```
ALERT ViolationSurge60s IF gauth_validation_surge_60s == 1 FOR 2m LABELS {severity="high"}
```

### OpenTelemetry Metrics Exporter (NEW)
An optional OpenTelemetry metrics exporter now emits violation counters and rolling rates as Observable Gauges when `GAUTH_OTEL_METRICS_ENABLE=1` is set. A stdout exporter is used for the demo; production deployments should swap in OTLP exporters.

Environment Toggle:
```
GAUTH_OTEL_METRICS_ENABLE=1 ./gauth-server
```

Instrumented Gauge Names:
- `gauth_violation_counter_sig_invalid`
- `gauth_violation_counter_expired`
- `gauth_violation_counter_not_yet_valid`
- `gauth_violation_counter_issuer_mismatch`
- `gauth_violation_counter_replay_detected`
- `gauth_violation_counter_audience_mismatch`
- `gauth_violation_counter_missing_claim`
- `gauth_violation_counter_unknown`
- `gauth_violation_rate_60s` (per-minute normalized rate over ~60s window)
- `gauth_violation_rate_300s` (per-minute normalized rate over ~300s window)
- `gauth_poa_semantic_counter_<category>` (semantic validation rejection counters)
- `gauth_poa_semantic_rate_60s_<category>` / `gauth_poa_semantic_rate_300s_<category>` (per-category per-minute rejection rates)
- `gauth_poa_semantic_anomaly_score{category="<category>"}` (EWMA-based z-score of 60s rate; baseline adapts online)

All gauges are observed during the exporter callback without additional locking (atomic snapshot reads). Rate computation reuses in-memory rolling history (`violationHistory`).

Sample Stdout Exporter Emission (truncated):
```
gauth_violation_counter_sig_invalid 12
gauth_violation_counter_missing_claim 7
gauth_violation_rate_60s 42.7
gauth_violation_rate_300s 18.4
gauth_poa_semantic_counter_amount_limit_exceeded 5
gauth_poa_semantic_rate_60s_amount_limit_exceeded 12.0
gauth_poa_semantic_anomaly_score{category="amount_limit_exceeded"} 2.1
```

### Semantic Anomaly Score (NEW)

The semantic anomaly score is a per-category standardized z-score computed from an online EWMA/Welford variance of the 60s per-minute rejection rate.

Computation:
```
mean_t+1 = mean_t + (x - mean_t)/count
M2_t+1   = M2_t + (x - mean_t)*(x - mean_t+1)
variance = M2/(count-1)   (count >= 2)
score    = (x - mean) / (sqrt(variance)+epsilon)   (count >=5 else 0)
```
Where `x` is the current 60s per-minute rate sample; `epsilon=1e-6` avoids division by zero.

Interpretation Guidelines:
- `≈0`: Within adaptive baseline noise band.
- `>1`: Mild elevation.
- `>2`: Significant deviation (suggest investigate).
- `>3`: High severity spike (candidate alert).
- `<-1`: Unusual drop (might indicate suppression or upstream outage).

## Policy Revision Governance Metrics (NEW)

Two Prometheus metrics expose policy version lifecycle state:

```
# HELP gauth_policy_revisions_total Total appended policy bundle revisions
# TYPE gauth_policy_revisions_total counter
gauth_policy_revisions_total 7

# HELP gauth_policy_active_version Current effective policy bundle version (rollback aware)
# TYPE gauth_policy_active_version gauge
gauth_policy_active_version 5
```

Interpretation:
- `gauth_policy_revisions_total` increments only when a new bundle is appended (immutable history length).
- `gauth_policy_active_version` reflects the version of the bundle currently governing decisions. During rollback it may be lower than the highest revision.
- A rollback does NOT decrement `gauth_policy_revisions_total`; instead the difference `(max_over_time(gauth_policy_revisions_total[5m]) - gauth_policy_active_version)` becomes >0.

Alert Examples:
```
ALERT PolicyRollbackActive
	IF (max_over_time(gauth_policy_revisions_total[10m]) - gauth_policy_active_version) > 0
	FOR 3m
	LABELS {severity="warning"}
	ANNOTATIONS {summary="Policy rollback active", description="Active version behind latest appended revision"}

ALERT PolicyFrequentRollbacks
	IF increase(gauth_policy_active_version[1h]) < increase(gauth_policy_revisions_total[1h]) AND (max_over_time(gauth_policy_revisions_total[1h]) - gauth_policy_active_version) > 0
	FOR 5m
	LABELS {severity="medium"}
	ANNOTATIONS {summary="Frequent rollback activity", description="Multiple new revisions with persistent rollback state"}
```

Dashboard Tips:
- Plot both metrics; annotate rollback windows when gauge dips below counter.
- Derive rollback ratio: `rollback_ratio = (gauth_policy_revisions_total - gauth_policy_active_version) / gauth_policy_revisions_total` (0 when no rollback).

Roadmap:
- Add audit event metric `gauth_policy_rollbacks_total` to distinguish intentional governance actions vs temporary exploration.
- Tag evaluation latency histogram buckets with version labels for longitudinal performance drift analysis across revisions.

Suggested PromQL Alert (prototype):
```
ALERT SemanticAnomalyHigh
	IF max_over_time(gauth_poa_semantic_anomaly_score[2m]) > 3
	FOR 2m
	LABELS {severity="high"}
	ANNOTATIONS {
		summary="High semantic rejection anomaly for GAuth",
		description="Semantic anomaly score exceeded 3 for >2m. Investigate recent PoA rejection surge categories."}
```

Follow-up (Roadmap): incorporate seasonality and time-of-day baselines and add persistence for EWMA state.

Background Sampling:
An optional background anomaly sampler runs when `GAUTH_SEMANTIC_ANOMALY_BG_SEC` is set (minimum 5). It periodically (every N seconds) appends semantic counter snapshots (if changed or >30s elapsed) and recomputes 60s rates feeding the EWMA anomaly model.

Example:
```
GAUTH_SEMANTIC_ANOMALY_BG_SEC=15 ./gauth-server

Hash‑chain wrapped payload example:
```jsonc
{
			"amount_limit_exceeded": 17,
			"daily_amount_limit_exceeded": 2,
			"currency_mismatch": 5,
			"scope_violation": 0,
			"restriction_mismatch": 3
		},
		"anomaly": {
			"amount_limit_exceeded": { "mean": 4.2, "m2": 11.86, "count": 9 },
			"currency_mismatch":      { "mean": 0.8, "m2": 1.27,  "count": 6 }
			// categories absent simply had no samples yet
		},
		"timestamp": "2025-10-19T02:11:45.901234Z"
	},
	"prev_hash": "804ba1425be9...",
	"hash": "6c1d4ef9921c...",
	"timestamp": "2025-10-19T02:11:45.901234Z"
}
```

Fields under `anomaly`:
- `mean`: Running mean of the 60s per‑minute rate samples.
- `m2`: Welford aggregate (sum of squared deviations) for variance reconstruction.
- `count`: Number of samples incorporated.

On restore the server rehydrates EWMA stats before new samples arrive, so early anomaly scores reflect historical baseline immediately (avoiding cold‑start false positives).

Backward Compatibility: Files without `anomaly` are treated as legacy; anomaly stats begin fresh.

Integrity: The entire payload including `anomaly` participates in the same hash chain computation (`sha256(prev_hash || raw_payload_bytes)`). Tampering with EWMA state is detectable via `/api/v1/beta/metrics/poa/semantics/verify`.

### Remaining Gaps (Updated – Post Hash Chain & Semantic Rates)
1. Category-specific surge indicators (currently only aggregate `surge_60s`).
2. Adaptive baseline (EWMA / seasonality) replacing static threshold env knob.
3. Persistence rotation policy (periodic rollover + archival) & multi-file ledger verification.
4. Validation path latency & payload size histograms (token validator) + quantile sketches.
5. Optional per-category semantic surge flags (rate thresholding per semantic counter).
6. Chain hash external anchoring (publishing latest hash to audit/anchor subsystem).

Deferrable Low-Risk Enhancements:
1. Add ratio gauges (e.g., specific category / total) for dashboards.
2. Export build/version and strict mode as OTEL Info metric for annotation.
3. Provide CLI tool for offline verification across rotated snapshot set.

Evidence references included in `docs/conformance_gaps.json` (sec7.item3 now Partial).

# GAuth Observability & Metrics Guide

> Last Updated: 2025-10-19

This document describes the available metrics, recommended PromQL queries, alerting thresholds, and SLO considerations for the GAuth reference implementation.

## 1. Metrics Surface

All counters and latency histogram are exposed under the Prometheus namespace `gauth` and subsystem `rfc0111` when using the `PrometheusMetrics` adapter.

| Metric Name | Type | Description | Increment Source |
|-------------|------|-------------|------------------|
| gauth_rfc0111_delegations_created_total | counter | Successfully created delegations (POAs). | `CreateDelegationCtx` after persistence & audit success |
| gauth_rfc0111_signatures_issued_total | counter | Canonical POA signatures successfully issued. | Sign path (`signerProvider.Sign`) success |
| gauth_rfc0111_signature_issue_failures_total | counter | Failed attempts to issue a POA signature (digest or sign error). | Any signature attempt error |
| gauth_rfc0111_signature_verifications_total | counter | Successful signature verifications during validation. | `ValidateDelegationCtx` authenticity path |
| gauth_rfc0111_signature_public_key_missing_total | counter | Signature present but public key not found (soft skip path). | `ValidateDelegationCtx` when key id absent and NOT strict mode |
| gauth_rfc0111_envelope_v1_issued_total | counter | Tokens issued using legacy envelope version 1 format. | `generateAuthToken` when GAUTH_POA_ENVELOPE_V2 != 1 |
| gauth_rfc0111_envelope_v2_issued_total | counter | Tokens issued using envelope version 2 (canonical digest + multi-sig metadata). | `generateAuthToken` when GAUTH_POA_ENVELOPE_V2 = 1 |
| gauth_rfc0111_envelope_issuance_cadence_seconds | histogram | Distribution of inter-issuance intervals (seconds between consecutive token issuances). Buckets: 0.1,0.25,0.5,1,2,5,10,30,60,120,300. | Recorded after each issuance when a previous issuance timestamp exists |
| gauth_rfc0111_envelope_v2_adoption_ratio | gauge | Ratio of V2 issuance count to total issuance count (V1+V2). | Updated on every issuance |
| gauth_rfc0111_envelope_v1_sunset_phase | gauge | Current integer sunset lifecycle phase for Envelope V1 (Pilot=1,Broad=2,Stabilization=3,SoftDeprecation=4,Sunset=5,PostVerification=6). | Manually/automatically set via migration controller |
| gauth_rfc0111_anchor_attempt_total | counter | Attempts to externally anchor issuance or revocation chain tip. | After chain append (issuance & revocation) when `anchorClient` not nil |
| gauth_rfc0111_anchor_failure_total | counter | External anchoring failures. | When `anchorClient.Anchor` returns error |

### External Anchor Deterministic Seeding (NEW)
### Envelope Versioning (NEW)

Delegation auth tokens now support a versioned envelope structure controlled by the environment flag:

```
GAUTH_POA_ENVELOPE_V2=1
```

When unset (default), issuance uses Envelope V1 (`ver="gauth-rfc0111-env1"`), containing the core delegation fields. When set to `1`, issuance switches to Envelope V2 (`ver="gauth-rfc0111-env2"`), adding:

Fields Added in V2:
- `canonical_digest`: Hex digest of canonical POA representation (best-effort; empty if digest computation failed during issuance fallback).
- `satisfied_weight`: Cumulative verified weight (for weighted multi-signature scenarios) captured at issuance time (if already known — may be 0 when signatures occur post issuance path).
- `satisfied_signatures`: Count of valid signatures contributing to threshold.

Metrics:
### Issuance Cadence Histogram (NEW)

Purpose:
Tracks pacing of envelope issuance to identify abnormally rapid spikes (possible runaway job or traffic surge) and assess rollout stability during migration.

Metric:
```
# HELP gauth_rfc0111_envelope_issuance_cadence_seconds Seconds between consecutive token issuances (inter-arrival interval)
# TYPE gauth_rfc0111_envelope_issuance_cadence_seconds histogram
gauth_rfc0111_envelope_issuance_cadence_seconds_bucket{le="0.1"} 12
...
gauth_rfc0111_envelope_issuance_cadence_seconds_sum 542.3
gauth_rfc0111_envelope_issuance_cadence_seconds_count 275
```

Buckets:
`[0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60, 120, 300]` seconds — tuned to cover sub‑second bursts through multi‑minute issuance gaps.

Computation:
On each issuance the service loads the previous issuance UNIX timestamp; if present, computes `delta = now - prev` and observes histogram. Then updates last issuance timestamp.

Example PromQL:
- P95 issuance cadence (fast issuance implies small value):
```
histogram_quantile(0.95, sum by (le)(rate(gauth_rfc0111_envelope_issuance_cadence_seconds_bucket[5m])))
```
- Detect sustained rapid issuance (<500ms inter-arrival) over 10 minutes:
```
histogram_quantile(0.50, sum by (le)(rate(gauth_rfc0111_envelope_issuance_cadence_seconds_bucket[10m]))) < 0.5
```
- Average issuance rate derived from cadence (approx tokens/sec):
```
1 / ( (gauth_rfc0111_envelope_issuance_cadence_seconds_sum / gauth_rfc0111_envelope_issuance_cadence_seconds_count) )
```

Alert Examples:
```
ALERT EnvelopeIssuanceBurst
	IF histogram_quantile(0.90, sum by (le)(rate(gauth_rfc0111_envelope_issuance_cadence_seconds_bucket[5m]))) < 0.25
	FOR 2m
	LABELS {severity="high"}
	ANNOTATIONS {summary="High-speed token issuance burst", description="P90 issuance cadence <250ms for >2m; investigate potential runaway issuer."}
```

Operational Guidance:
- Sudden collapse of cadence metrics (extremely low intervals) during migration phases may indicate uncontrolled batch issuance ignoring adoption safeguards.
- Elevated intervals (multi-minute gaps) combined with low adoption ratio plateau could signal stalled migration.

SLO Consideration:
Track `P95 cadence > 2s` as a stability objective if issuance is expected to be moderately paced outside known batch windows.

### Sunset Phase Gauge (NEW)

Purpose:
Represents governance lifecycle for Envelope V1 deprecation. Instrumentation enables dashboards & alerting aligned with ADR `ADR-envelope-v1-sunset.md`.

Metric:
```
# HELP gauth_rfc0111_envelope_v1_sunset_phase Envelope V1 sunset lifecycle phase
# TYPE gauth_rfc0111_envelope_v1_sunset_phase gauge
gauth_rfc0111_envelope_v1_sunset_phase 3
```

Phase Mapping:
| Phase | Value | Description |
|-------|-------|-------------|
| Pilot | 1 | Limited controlled rollout, validate basic integrity |
| Broad | 2 | Wider audience, monitor adoption + mismatch closely |
| Stabilization | 3 | Majority traffic on V2; address residual issues |
| Soft Deprecation | 4 | Communicate deprecation, throttle new V1 issuance |
| Sunset | 5 | Block V1 issuance; only existing tokens valid until expiry |
| Post-Verification | 6 | Verification-only monitoring period, ensure no unexpected V1 issuance |

Example PromQL Readiness Check (Adoption + Digest Health):
```
sunset_ready = (gauth_rfc0111_envelope_v2_adoption_ratio > 0.95) and \
							 ((increase(gauth_rfc0111_envelope_digest_mismatch_total[1h]) / increase(gauth_rfc0111_envelope_v2_issued_total[1h])) < 0.002)
```

Automated Controller (Roadmap):
- Observe adoption & mismatch ratios; when thresholds satisfied for N hours promote phase sequentially.
- Manual override via ops tooling: set gauge directly for emergency rollback.

Alert Example (Unexpected Legacy Issuance Post Sunset):
```
ALERT UnexpectedEnvelopeV1IssuancePostSunset
	IF gauth_rfc0111_envelope_v1_sunset_phase == 5 AND increase(gauth_rfc0111_envelope_v1_issued_total[5m]) > 0
	FOR 1m
	LABELS {severity="critical"}
	ANNOTATIONS {summary="Legacy envelope issuance detected after Sunset", description="Envelope V1 issuance occurred while in Sunset phase (5)."}
```

Dashboard Suggestions:
- Phase timeline (annotated) vs adoption ratio.
- Cadence p95 overlayed with adoption ratio to show stability while approaching Sunset.
- Digest mismatch rate trend lines gating phase promotion.

Governance Notes:
- Gauge should never decrement between phases except emergency rollback; track changes in audit log.
- Post-Verification phase maintained for minimum 7 days per ADR to detect latent mismatches.

### Digest Mismatch Reason Labeling (NEW)

Purpose:
Adds attribution to envelope digest mismatches enabling root cause triage and blocking sunset promotion when systemic issues appear.

Metrics:
```
# HELP gauth_rfc0111_envelope_digest_mismatch_total Canonical digest mismatches during verification
# TYPE gauth_rfc0111_envelope_digest_mismatch_total counter

# HELP gauth_rfc0111_envelope_digest_mismatch_reason_total Envelope digest mismatches labeled by reason
# TYPE gauth_rfc0111_envelope_digest_mismatch_reason_total counter
gauth_rfc0111_envelope_digest_mismatch_reason_total{reason="tamper_suspected"} 3
gauth_rfc0111_envelope_digest_mismatch_reason_total{reason="domain_conflict"} 1
gauth_rfc0111_envelope_digest_mismatch_reason_total{reason="other"} 2
```

Reason Semantics:
| Reason | Heuristic | Interpretation | Suggested Action |
|--------|-----------|----------------|------------------|
| tamper_suspected | Stored digest length differs from recomputed | Possible envelope manipulation or corruption | Investigate signing pipeline & storage layer integrity |
| domain_conflict | Equal length but different value | Potential domain separation / versioning inconsistency (e.g., digest prefix rules changed) | Validate canonicalization algorithm & domain separation configuration |
| other | Fallback classification | Unclassified mismatch | Add additional labeling heuristics / logging |

Roadmap: Introduce explicit `canonicalization_error` label when digest computation fails pre-compare; currently such failures are counted separately and do not raise mismatch reason metrics.

Alert Examples:
```
ALERT DigestTamperSuspectedSpike
	IF increase(gauth_rfc0111_envelope_digest_mismatch_reason_total{reason="tamper_suspected"}[15m]) > 10
	FOR 5m
	LABELS {severity="high"}
	ANNOTATIONS {summary="Spike in tamper-suspected digest mismatches", description=">10 tamper_suspected mismatches in 15m window."}
```

Promotion Guard (Sunset Controller):
Sunset phase automation uses mismatch ratio (mismatches / V2 issued) and adoption ratio thresholds. Elevated tamper_suspected counts temporarily reset the satisfaction window preventing phase advancement.

### Sunset Phase Automation Controller (NEW)

Path: `internal/sunset/controller.go`

Behavior:
- Periodically (default 30s) evaluates adoption ratio and mismatch ratio against configured thresholds.
- Requires continuous satisfaction for a window (default 15m) before promoting to next phase.
- Never decrements phase unless rollback is explicitly allowed (manual emergency override).

Threshold Defaults:
| Transition | Adoption >= | Mismatch Ratio <= |
|------------|------------|-------------------|
| Pilot -> Broad | 0.60 | 0.005 |
| Broad -> Stabilization | 0.80 | 0.005 |
| Stabilization -> SoftDep | 0.90 | 0.005 |
| SoftDep -> Sunset | 0.95 | 0.005 |

Configuration (example env-driven pseudo-code — integrate with actual config loader later):
```
ControllerConfig{
	Enable: true,
	Interval: 30 * time.Second,
	Window: 15 * time.Minute,
	PilotToBroadAdoption: 0.6,
	BroadToStabilizeAdoption: 0.8,
	StabilizeToSoftDepAdoption: 0.9,
	SoftDepToSunsetAdoption: 0.95,
	MaxMismatchRatio: 0.005,
}
```

Readiness Composite (reused from recording rules concept):
```
sunset_ready = (gauth_rfc0111_envelope_v2_adoption_ratio > 0.95) and \
							 ((increase(gauth_rfc0111_envelope_digest_mismatch_total[1h]) / increase(gauth_rfc0111_envelope_v2_issued_total[1h])) < 0.002)
```

Dashboard Widgets:
- Phase gauge vs adoption ratio line & mismatch ratio area chart.
- Reason breakdown stacked bar for mismatches (tamper vs domain vs other) over time.
- Controller satisfaction progress (elapsed / required window) as burn-down indicator.

Rollback Guidance:
If urgent issues discovered post promotion, set `GAUTH_ENVELOPE_SUNSET_FORCE_PHASE=<prev>` (future implementation) and record audit event; avoid toggling repeatedly.


```
# HELP gauth_rfc0111_envelope_v1_issued_total Delegation tokens issued using envelope version 1
# HELP gauth_rfc0111_envelope_v2_issued_total Delegation tokens issued using envelope version 2
```
These allow tracking migration rollout and alerting on unexpected reversion (e.g. V2 drop to zero).

Backward Compatibility:
- Verification auto-detects envelope version by inspecting `ver` suffix; both formats normalize into `TokenVerificationResult` identically.
- No client changes required for legacy consumers; V2 fields are additive and ignorable for parsers that only require grantor/grantee/scope/expiry.

Suggested PromQL Migration Dashboard:
```
sum(rate(gauth_rfc0111_envelope_v1_issued_total[5m])) as v1_rate
sum(rate(gauth_rfc0111_envelope_v2_issued_total[5m])) as v2_rate
v2_adoption_pct = v2_rate / (v1_rate + v2_rate)
```

Alert Example (Detect Envelope V2 Regression for >15m):
```
ALERT EnvelopeV2Regression
	IF sum(increase(gauth_rfc0111_envelope_v2_issued_total[15m])) == 0
	FOR 5m
	LABELS {severity="medium"}
	ANNOTATIONS {summary="Envelope V2 issuance stalled", description="No V2 tokens issued in the last 15m (possible misconfiguration or rollback)."}
```

Domain Separation Note:
- Envelope versioning does not modify canonical POA digest domain separation logic (controlled independently by `GAUTH_MULTI_SIG_DOMAIN_V2`).
- Envelope V2 embedding of `canonical_digest` is a mirror value; digest computation rules remain unchanged and unaffected by the envelope toggle.

Rollout Strategy:
1. Deploy with `GAUTH_POA_ENVELOPE_V2=0` and observe baseline V1 issuance rate.
2. Enable flag in staging; monitor `envelope_v2_issued_total` growth and downstream consumer compatibility.
3. Gradually enable in production (canary subset) using deployment slice environment overrides.
4. Set global flag once V2 adoption >90% over a sustained window.
5. (Future) Deprecate V1 after two minor release cycles; maintain verification support until official sunset date.

### Rollout Monitoring & Adoption Ratio (NEW)

To simplify migration dashboards we expose a direct gauge:

Metric:
```
# HELP gauth_rfc0111_envelope_v2_adoption_ratio Ratio (0-1) of envelope V2 issuance vs total issuance
# TYPE gauth_rfc0111_envelope_v2_adoption_ratio gauge
```
Computation (performed after each issuance):
```
adoption_ratio = envelope_v2_issued_total / (envelope_v1_issued_total + envelope_v2_issued_total)
```
Edge Cases:
- Division by zero guarded: ratio remains 0 until first issuance.
- Gauge updates only after issuance events (not on verification).
- Large rapid oscillations typically indicate flag flapping between replicas.

Suggested PromQL (raw ratio + smoothing):
```promql
gauth_rfc0111_envelope_v2_adoption_ratio
avg_over_time(gauth_rfc0111_envelope_v2_adoption_ratio[15m])
```
Threshold Mapping (recommended policy):
| Phase | Ratio (15m avg) | Action |
|-------|-----------------|--------|
| Pilot | 0.05 - 0.20 | Validate consumer compatibility; monitor mismatch counter |
| Broad Adoption | 0.20 - 0.70 | Continue staged rollout, watch for regressions |
| Stabilization | 0.70 - 0.90 | Prepare deprecation comms for V1 |
| Soft Deprecation | >=0.90 sustained 24h | Announce impending V1 sunset window |
| Sunset Eligible | >=0.95 sustained 7d | Remove V1 issuance except emergency override |

Regression Alert Example (ratio drop below 0.80 for 10m):
```yaml
ALERT EnvelopeV2AdoptionRegression
	IF avg_over_time(gauth_rfc0111_envelope_v2_adoption_ratio[10m]) < 0.80
	FOR 5m
	LABELS {severity="high"}
	ANNOTATIONS {summary="Envelope V2 adoption ratio regression", description="Ratio <80% for 10m (possible rollback or misconfiguration)."}
```

Canary Completion Alert (reach stabilization threshold):
```yaml
ALERT EnvelopeV2StabilizationReached
	IF avg_over_time(gauth_rfc0111_envelope_v2_adoption_ratio[30m]) > 0.70
	FOR 15m
	LABELS {severity="info"}
	ANNOTATIONS {summary="Envelope V2 stabilization threshold crossed", description="Adoption >70% sustained for 30m; begin soft deprecation planning."}
```

Dashboard Widgets:
- Single stat: current `gauth_rfc0111_envelope_v2_adoption_ratio`.
- Sparkline: 6h window of ratio.
- Phase annotation lines at 0.2, 0.7, 0.9, 0.95.
- Overlay lines for `rate(gauth_rfc0111_envelope_v1_issued_total[5m])` vs `rate(gauth_rfc0111_envelope_v2_issued_total[5m])`.

Operational Runbook Notes:
- Sudden ratio plunge: check environment variable drift or a subset of pods deployed without the flag.
- Plateau <0.2 in pilot longer than 24h: investigate consumer incompatibilities or silent parsing failures (look for elevated `envelope_digest_mismatch_total`).
- Oscillation pattern (sawtooth): rolling restarts toggling flag; stabilize configuration management.

### Digest Mismatch Counter (NEW)

Metric:
```
# HELP gauth_rfc0111_envelope_digest_mismatch_total Canonical digest mismatch detected during envelope verification
# TYPE gauth_rfc0111_envelope_digest_mismatch_total counter
```
Definition: Incremented when verification recomputes the canonical delegation digest and it differs from the envelope's embedded digest (V2) or expected reconstruction (V1 path). This signals potential integrity issues: mutation, canonicalization drift, or code regression.

Expected Behavior:
- Zero or near-zero during normal operation.
- Non-zero transient bumps during active deployments that modify canonicalization logic (should be explicitly communicated in release notes).

Alerting:
```yaml
ALERT EnvelopeDigestMismatchSpike
	IF increase(gauth_rfc0111_envelope_digest_mismatch_total[10m]) > 3
	FOR 5m
	LABELS {severity="critical"}
	ANNOTATIONS {summary="Envelope digest mismatch spike", description=">3 mismatches in 10m; possible integrity regression or tampering."}
```
Ratio (mismatch per issuance) for context:
```promql
increase(gauth_rfc0111_envelope_digest_mismatch_total[5m]) / (increase(gauth_rfc0111_envelope_v1_issued_total[5m]) + increase(gauth_rfc0111_envelope_v2_issued_total[5m]))
```
If ratio >0.01 (1%) sustained => investigate canonicalization path or potential malicious token modifications.

Troubleshooting Steps:
1. Capture sample mismatched token envelope (ensure redaction of sensitive fields) and recompute digest manually using the canonicalization helper.
2. Diff canonical JSON representation vs stored envelope fields.
3. Verify multi-signature domain separation flag consistency (`GAUTH_MULTI_SIG_DOMAIN_V2`).
4. Check recent deployment diff for changes in ordering, omitted fields, or added optional metadata.
5. Inspect logs for fallback digest computation warnings during issuance.
6. If tampering suspected, enable heightened audit event sampling and consider revoking affected delegations.

False Positives:
- Tests intentionally tampering with digest to validate counter increments.
- Race during immediate post-issuance signature augmentation (should be rare; review issuance path ordering).

Hardening Roadmap:
- Add labeled mismatch counter with reason codes (canonicalization_error, tamper_suspected, domain_conflict).
- Expose last mismatch timestamp & moving window gauge for UI quick-glance.
- Integrate with anomaly detection subsystem for automatic surge classification.

Sunset Governance Tie-In:
During deprecation planning for Envelope V1, mismatch anomaly rate must remain below 0.5% for 7 consecutive days before disabling V1 issuance. Include `increase(gauth_rfc0111_envelope_digest_mismatch_total[1h])` in ADR metrics acceptance criteria.

Security Considerations:
- Elevated mismatch counts can signal canonicalization drift enabling signature replay across versions.
- Continuous monitoring is required during multi-signature domain separation migrations.

Data Retention:
- Raw counter only; long-term trend derived through remote write or recording rules.
- Consider recording rule:
```promql
record: gauth_rfc0111_envelope_digest_mismatch_rate_5m
expr: rate(gauth_rfc0111_envelope_digest_mismatch_total[5m])
```


Troubleshooting:
- If `envelope_v2_issued_total` remains zero after enabling flag, confirm environment variable visibility for the process and ensure no wrapper scripts override it.
- Unexpected spikes in V1 after migration completion may indicate a rollback or misconfigured replica.


Flaky probability-driven tests for the external anchoring stub (`tsa_stub`) are eliminated via a deterministic RNG seed.

Environment Variable:
```
GAUTH_CAP_EXTERNAL_ANCHOR_RAND_SEED=<int64 seed>
```

Behavior:
- When set and `GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER=tsa_stub`, the stub provider uses a fixed `math/rand` source seeded with the provided value.
- This makes latency and success/failure sequences reproducible across test runs and CI.
- Omitted or empty value falls back to non-deterministic time-based seeding.

Use Cases:
- Stabilizing retry logic tests (avoids asserting on statistically rare patterns).
- Replaying specific failure/success sequences for debugging.

Example:
```
GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER=tsa_stub \
GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB=0.6 \
GAUTH_CAP_EXTERNAL_ANCHOR_RETRIES=4 \
GAUTH_CAP_EXTERNAL_ANCHOR_RAND_SEED=1700001 \
./gauth-server
```

Test Guidance:
- Prefer asserting on presence of attempts + latency sample rather than insisting on a failure occurrence for moderate probabilities.
- Use `GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB=1` in separate tests to exercise failure metrics deterministically.

### Forced Initial Failures (NEW)

### Multi-Signature Verification Metrics (NEW)

Multi-signature delegation (POA) validation now emits granular counters and a dedicated latency histogram. These metrics differentiate structural, cryptographic, and threshold weight failures, enabling precise alerting and optimization transparency.

Prometheus Counters:
```
# HELP gauth_rfc0111_multi_signature_verifications_total Successful multi-signature threshold verifications
# TYPE gauth_rfc0111_multi_signature_verifications_total counter
gauth_rfc0111_multi_signature_verifications_total 7

# HELP gauth_rfc0111_multi_signature_verification_failures_total Failed multi-signature threshold verifications (generic)
gauth_rfc0111_multi_signature_verification_failures_total 3

# HELP gauth_rfc0111_multi_signature_structural_failures_total Multi-signature structural precondition failures
gauth_rfc0111_multi_signature_structural_failures_total 1

# HELP gauth_rfc0111_multi_signature_digest_failures_total Multi-signature canonical digest failures
gauth_rfc0111_multi_signature_digest_failures_total 0

# HELP gauth_rfc0111_multi_signature_public_key_missing_failures_total Multi-signature public key missing events
gauth_rfc0111_multi_signature_public_key_missing_failures_total 0

# HELP gauth_rfc0111_multi_signature_invalid_signature_failures_total Multi-signature invalid signature cryptographic failures
gauth_rfc0111_multi_signature_invalid_signature_failures_total 1

# HELP gauth_rfc0111_multi_signature_threshold_failures_total Multi-signature count-based threshold failures
gauth_rfc0111_multi_signature_threshold_failures_total 1

# HELP gauth_rfc0111_multi_signature_weight_failures_total Multi-signature weight threshold failures
gauth_rfc0111_multi_signature_weight_failures_total 0
```

Latency Histogram:
```
# HELP gauth_rfc0111_multi_signature_verification_latency_seconds Latency of multi-signature verification operations
# TYPE gauth_rfc0111_multi_signature_verification_latency_seconds histogram
```
Buckets inherit the adapter's generic bucket configuration unless overridden (default sub‑millisecond through 100ms). Use P95/P99 via PromQL:
```
histogram_quantile(0.95, sum by (le) (rate(gauth_rfc0111_multi_signature_verification_latency_seconds_bucket[5m])))
```

Failure Taxonomy:
- Structural: malformed signer list (duplicates, empty, insufficient signers), invalid weight map (duplicate keys, unknown signer, non-positive or overflow weight, total possible weight < threshold).
- Digest: canonical digest computation failure (rare; indicates internal bug or unexpected mutation).
- Public Key Missing: signature present but key material not found (strict mode elevates to integrity failure path).
- Invalid Signature: cryptographic verification failed (mismatched digest, wrong algorithm, or bad Ed25519 signature bytes).
- Threshold: valid signature count below configured threshold in count-based mode.
- Weight: cumulative valid weights below threshold in weighted mode.

Gauge-Like Fields (in responses / internal POA struct):
- `satisfied_signatures`: number of valid signatures contributing to threshold.
- `satisfied_weight`: sum of valid weights (weighted mode only).

Suggested Alerts:
```
ALERT MultiSignatureStructuralAnomaly
	IF increase(gauth_rfc0111_multi_signature_structural_failures_total[10m]) > 5
	FOR 5m

ALERT MultiSignatureCryptoErrorSpike
	IF (increase(gauth_rfc0111_multi_signature_invalid_signature_failures_total[5m]) + increase(gauth_rfc0111_multi_signature_public_key_missing_failures_total[5m])) > 10
	FOR 3m
```

### Multi-Signature Domain Separation V2 (NEW)

To prevent digest collisions between single-signature and threshold / weighted multi-signature modes, an optional domain prefix variant can be enabled:
```
GAUTH_MULTI_SIG_DOMAIN_V2=1
```
When active and `Threshold>1`, the canonical digest preimage domain changes from:
```
GAUTH_RFC0111_POA_V1\n
```
to:
```
GAUTH_RFC0111_POA_V2|thr=<threshold>|w=<sorted-weight-map>\n
GAUTH_RFC0111_POA_V3|tax=1\n  (Introduced with taxonomy expansion RB2; used for single-signer / non multi-sig PoAs when Version >=3. Multi-sig PoAs continue to use V2 domain to minimize downstream changes. Canonical JSON gains optional taxonomy object: {"taxonomy":{"agent_type":...,"sector":...,"action_class":...}} only when non-empty values are provided.)
```
`<sorted-weight-map>` is a comma-separated `signer=weight` list sorted lexicographically. If no valid weight map is configured, only `thr=<threshold>` is embedded (weights segment empty).

Backward Compatibility:
- Existing signatures produced under V1 remain valid when V2 disabled (default).
- Enabling V2 invalidates prior multi-signature artifacts (expected, migration requires re-issuance); single-signature delegations unaffected (threshold ≤1).
- Tests ensure canonical JSON payload remains unchanged across domain versions while digest differs (domain separation only).

Migration Guidance:
1. Enable V2 in staging, re-issue multi-signature delegations, validate threshold and weight metrics stability.
2. Roll out to production with dual-path acceptance if needed (temporary logic to accept both domains) – deferred feature.
3. Monitor structural + crypto failure counters for unexpected spikes post switch.

PromQL Digest Change Detection (optional):
If previous digest set cardinality known:
```
sum(increase(gauth_rfc0111_multi_signature_verifications_total[30m])) BY ()
```
Should reflect re-verification volume after re-issuance; structural failures should be near zero.

Roadmap:
- Dual-domain acceptance window for smoother migration.
- Weight map hash embedding to reduce prefix length for very large signer sets.
- Distinct latency bucket profile for high-signer cardinality scenarios.


To guarantee a failure-before-success sequence without relying on probability draws, configure:
```
GAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS=<n>
```
When `n>0` and the provider is `tsa_stub`, the first `n` anchoring attempts return a forced failure (`stub failure (forced)`), after which normal probabilistic behavior resumes.

Interaction Order:
1. Forced failures (decrementing counter) take precedence.
2. If no forced failures remain, probabilistic failure (`rnd.Float64() < failProb`) applies.
3. On success, latency and receipt metrics record normally.

Recommended Usage:
- Set `GAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS=1` in tests asserting presence of at least one failure and subsequent success metrics.
- Combine with `GAUTH_CAP_EXTERNAL_ANCHOR_RAND_SEED` for fully reproducible multi-attempt sequences when `retries > 0`.

Example (failure then success deterministic):
```
GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER=tsa_stub \
GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB=0.6 \
GAUTH_CAP_EXTERNAL_ANCHOR_RETRIES=4 \
GAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS=1 \
GAUTH_CAP_EXTERNAL_ANCHOR_RAND_SEED=1700001 \
./gauth-server
```

Metrics Impact:
- Failure counters increment for forced failures like probabilistic ones; no separate label is added (intent: keep surface stable).
- Latency histogram excludes failed attempts (observed latency recorded only on success path); forced failures do not contribute latency samples.
| gauth_rfc0111_validation_latency_seconds | histogram | End-to-end delegation validation latency. | Observation in `ValidateDelegationCtx` |

### Cardinality
All current metrics are global (no labels). Add labels conservatively (e.g. `result`, `algorithm`) only when required for dashboards. A label explosion will degrade performance and increase scrape size.

## 2. PromQL Reference

Common queries:

```promql
# Delegations per 5m
sum(increase(gauth_rfc0111_delegations_created_total[5m]))

# Validation latency p95
(sum(increase(gauth_rfc0111_anchor_failure_total[15m])) / clamp_min(sum(increase(gauth_rfc0111_anchor_attempt_total[15m])), 1))

# Revocation integrity failures (bursts)
increase(gauth_rfc0111_revocation_integrity_failures_total[1h])
```

|-------|------------|-----------------------|-----------|
| High Validation Latency | `histogram_quantile(0.95, sum by (le)(rate(gauth_rfc0111_validation_latency_seconds_bucket[10m]))) > 0.050` | 50ms p95 sustained 10m | Indicates unexpected slow path (I/O, locking) |
| Anchor Failure Spike | `sum(increase(gauth_rfc0111_anchor_failure_total[15m])) > 0 and (sum(increase(gauth_rfc0111_anchor_failure_total[15m])) / sum(increase(gauth_rfc0111_anchor_attempt_total[15m]))) > 0.25` | >25% failures | External anchoring dependency degradation |
| Signature Verification Failures | `increase(gauth_rfc0111_signature_verification_failures_total[10m]) > 10` | >10 failures / 10m | Possible tampering or key mismatch event |
| Revocation Integrity Failure | `increase(gauth_rfc0111_revocation_integrity_failures_total[5m]) > 0` | Any occurrence | Chain integrity compromise should page immediately |

## 4. SLO & SLIs

| SLI | Measurement | Target | Notes |
|-----|------------|--------|------|
| Validation Latency p95 | histogram_quantile(0.95, rate(validation_latency_seconds_bucket[5m])) | < 40ms | In-memory baseline ~47.5µs for crypto verification; includes audit overhead |
| Signature Issue Success Rate | 1 - failure ratio | > 99.5% | Failures should be rare (digest/sign errors) |
| Revocation Integrity Fail | Count per 30d | 0 | Any failure triggers immediate investigation |
| Anchor Success Rate | Attempts - failures | > 90% | Until external service matures; adjust after production data |

## 5. Dashboard Layout

Recommended panels:
- Delegations Created (rate) line
- Validation Latency p50/p95/p99 stacked or multi-series
- Signature Issue / Verification Success Ratios (single-stat)
- Anchor Attempts vs Failures (dual line + ratio)
- Revocation Integrity Failures (single-stat, shows last 30d)
- Missing Public Key Counter (gauge) with strict mode toggle annotation

## 6. Strict Authenticity Mode Impact

When strict authenticity is enabled (`WithStrictAuthenticity()`):
- `signature_public_key_missing_total` SHOULD flatline at zero (missing key escalates to integrity failure and request abort).
- Add an annotation on dashboards when strict mode active (derive from config flag exported as env or Info metric when added).

## 7. Future Enhancements

| Enhancement | Description |
|-------------|-------------|
| Info Metric | Export build version and strict mode flag for dashboard annotations. |
| Signer Algorithm Label | Optional `algorithm` label on signature counters if multiple algorithms supported. |
| Anchor Latency | Histogram for anchoring latency once external service integrated. |
| Per-Action Validation Latency | Label `action` (careful: cardinality constraint) for high-volume actions only. |

## 8. Operational Runbooks

### Anchor Failure Burst
1. Check external anchoring service health (HTTP 5xx rates, latency). 
2. Verify DNS / network connectivity from GAuth pods. 
3. Inspect application logs for timeout or TLS errors. 
4. Consider circuit breaker: temporarily disable anchoring (swap client to `NoopAnchorClient`) if causing cascading latency.

### Revocation Integrity Failure
1. Capture current revocation chain tip hash (`chainTipRev`).
2. Run offline verification tool (planned) against repository snapshot.
3. Compare aggregate hash with last anchored hash (if available). 
4. If mismatch persists: initiate key compromise procedure (rotate keys, invalidate affected delegations).

### Elevated Missing Public Key Metric (non-strict mode)
1. Confirm key rotation event occurred; previous key still present? 
2. Check if signature KeyID matches an unexpectedly removed key. 
3. Audit for unauthorized key deletions.

## 9. Local Diagnostic Usage (Memory Metrics)

```go
mem := metrics.NewMemory()
// ... perform validations
snap := mem.SnapshotStruct()
fmt.Printf("p95 validation latency: %v\n", snap.P90) // approximate (use p90 if limited samples)
```

For richer analysis dump JSON:
```go
enc, _ := json.MarshalIndent(mem.SnapshotEx(), "", "  ")
fmt.Println(string(enc))
```

## 10. Benchmark Correlation

Correlate latency histogram shifts with benchmark regression signals in `BENCHMARKS.md`. Set automated guardrail to alert if p95 grows >25% versus rolling 7-day average.

---
Maintainers: Update this doc when introducing new metrics or labels. Every new metric must include an owner, purpose, and expected stability period.

## PoA Semantic Counters (NEW)

Semantic counters provide fine-grained visibility into domain-level delegation (PoA) validation rejections beyond generic scope or expiration errors. They are incremented in `ValidateDelegationRich` and surfaced via JSON, Prometheus, and OpenTelemetry gauges.

| Counter Key | Meaning | Trigger Condition | Notes |
|-------------|---------|-------------------|-------|
| `scope_violation` | Action outside granted delegation scope | Requested action not contained in `poa.Scope` | Also increments legacy scope metrics (if any) |
| `amount_limit_exceeded` | Requested amount surpassed configured maximum | `max_amount` restriction present and `requested_amount` > limit | Only enforced for `transaction:` prefixed actions |
| `currency_mismatch` | Currency in request metadata differs from delegation restriction | `currency` restriction present and metadata `currency` mismatch | Only evaluated for `transaction:` prefixed actions |
| `restriction_mismatch` | Arbitrary non-special restriction key mismatch | Metadata provides key present in `poa.Restrictions` with different value (excluding `currency`, `max_amount`) | First mismatch aborts validation |

### JSON Endpoint
`GET /api/v1/beta/metrics/poa/semantics`

Sample:
```json
{
	"success": true,
	"wired": true,
	"timestamp": "2025-10-19T01:45:12.345678Z",
	"counters": {
		"scope_violation": 7,
		"amount_limit_exceeded": 2,
		"daily_amount_limit_exceeded": 1,
		"currency_mismatch": 1,
		"restriction_mismatch": 0
	}
}
```

### Prometheus Exposition
`GET /api/v1/beta/metrics/poa/semantics/prometheus`

Format:
```
# HELP gauth_poa_semantic_counter Semantic PoA validation rejection counters.
# TYPE gauth_poa_semantic_counter counter
gauth_poa_semantic_counter_scope_violation 7
gauth_poa_semantic_counter_amount_limit_exceeded 2
gauth_poa_semantic_counter_daily_amount_limit_exceeded 1
gauth_poa_semantic_counter_currency_mismatch 1
gauth_poa_semantic_counter_restriction_mismatch 0
```

Metric Naming Pattern: `gauth_poa_semantic_counter_<key>`.

### OpenTelemetry Gauges
Enabled when `GAUTH_OTEL_METRICS_ENABLE=1`.

Gauge Names:
- `gauth_poa_semantic_counter_scope_violation`
- `gauth_poa_semantic_counter_amount_limit_exceeded`
- `gauth_poa_semantic_counter_daily_amount_limit_exceeded` (cumulative daily limit violations)
- `gauth_poa_semantic_counter_currency_mismatch`
- `gauth_poa_semantic_counter_restriction_mismatch`

All gauges are Int64 observable and collected alongside violation counters.

### Persistence
Set `GAUTH_SEMANTIC_PERSIST_PATH` to enable snapshot persistence (JSON file with schema):
```json
{
	"counters": {
		"scope_violation": 7,
		"amount_limit_exceeded": 2,
		"currency_mismatch": 1,
		"restriction_mismatch": 0
	},
	"timestamp": "2025-10-19T01:45:12.345678Z"
}
```
Autosave interval via `GAUTH_SEMANTIC_AUTOSAVE_SEC` (>=10). Throttle override: `GAUTH_SEMANTIC_PERSIST_NO_THROTTLE=1`.

Restore Behavior: On startup if `GAUTH_SEMANTIC_PERSIST_PATH` is set, the server calls `SetSemanticSnapshot` on the RFC0111 service to fully restore counters (missing keys default to zero). This replaces earlier advisory-only logging.

### Example PromQL
```promql
# 5m rate of amount limit exceeded events
sum(increase(gauth_poa_semantic_counter_amount_limit_exceeded[5m]))

# Scope violation ratio vs all semantic events
sum(increase(gauth_poa_semantic_counter_scope_violation[5m])) /
clamp_min(sum(increase(gauth_poa_semantic_counter_scope_violation[5m]) + increase(gauth_poa_semantic_counter_amount_limit_exceeded[5m]) + increase(gauth_poa_semantic_counter_daily_amount_limit_exceeded[5m]) + increase(gauth_poa_semantic_counter_currency_mismatch[5m]) + increase(gauth_poa_semantic_counter_restriction_mismatch[5m])), 1)
```

### Alert Suggestions
| Alert | Expression | Threshold | Rationale |
|-------|------------|-----------|-----------|
| Excessive Scope Violations | `sum(increase(gauth_poa_semantic_counter_scope_violation[10m])) > 50` | >50 / 10m | Indicates potential misuse or missing client-side caching of allowed actions |
| Amount Limit Abuse | `sum(increase(gauth_poa_semantic_counter_amount_limit_exceeded[5m])) > 10` | >10 / 5m | Detects repeated attempts above allowed financial thresholds |
| Daily Limit Abuse | `sum(increase(gauth_poa_semantic_counter_daily_amount_limit_exceeded[15m])) > 5` | >5 / 15m | Detects cumulative attempts to exceed daily authorized monetary ceilings |
| Currency Drift | `sum(increase(gauth_poa_semantic_counter_currency_mismatch[15m])) > 5` | >5 / 15m | Possible incorrect client region/currency configuration |
| Restriction Mismatch Spike | `sum(increase(gauth_poa_semantic_counter_restriction_mismatch[10m])) > 20` | >20 / 10m | Suggests outdated delegation metadata in consumers |

### Roadmap for Semantic Metrics
1. Add per-restriction labeled counters (guard cardinality via allowlist) for top 3 financial restriction keys.
2. Integrate semantic counters into anomaly window computation (`violationRatesForWindows` analogue) for rate-based alerts.
3. Implement full restoration support (`SetSemanticSnapshot`).
4. Add histogram for requested amounts vs delegated limit utilization.
5. Provide ratio dashboards (e.g., scope violations / total validation attempts).

### Semantic Anomaly Rates (NEW)
Per-category per-minute rates over 60s and 300s rolling windows are now computed using an in-memory history of semantic counter snapshots.

JSON Extension Example (`GET /api/v1/beta/metrics/poa/semantics`):
```json
{
	"success": true,
	"wired": true,
	"timestamp": "2025-10-19T02:05:12.345678Z",
	"counters": {
		"scope_violation": 7,
		"amount_limit_exceeded": 2,
		"currency_mismatch": 1,
		"restriction_mismatch": 0
	},
	"anomaly": {
		"rate_per_minute_60s": {"scope_violation": 3.2, "amount_limit_exceeded": 0.4, "currency_mismatch": 0.2, "restriction_mismatch": 0.0},
		"rate_per_minute_300s": {"scope_violation": 1.1, "amount_limit_exceeded": 0.15, "currency_mismatch": 0.07, "restriction_mismatch": 0.0}
	}
}
```

Prometheus Metrics:
```
# HELP gauth_poa_semantic_rate_60s Per-minute semantic rejection rate over trailing ~60s window.
# TYPE gauth_poa_semantic_rate_60s gauge
gauth_poa_semantic_rate_60s{category="scope_violation"} 3.2
gauth_poa_semantic_rate_60s{category="amount_limit_exceeded"} 0.4
...
# HELP gauth_poa_semantic_rate_300s Per-minute semantic rejection rate over trailing ~300s window.
# TYPE gauth_poa_semantic_rate_300s gauge
gauth_poa_semantic_rate_300s{category="scope_violation"} 1.1
...
```

OpenTelemetry Gauges:
```
gauth_poa_semantic_rate_60s_scope_violation
gauth_poa_semantic_rate_300s_scope_violation
gauth_poa_semantic_rate_60s_amount_limit_exceeded
gauth_poa_semantic_rate_300s_amount_limit_exceeded
... (one pair per semantic category)
```

Usage Hints:
1. Alert on sustained high scope_violation rate (e.g., >20/min for 5m) instead of raw counts.
2. Combine amount_limit_exceeded rate with business hour window label (future) to detect bursts.
3. Use ratio: `rate_60s_amount_limit_exceeded / (rate_60s_scope_violation + rate_60s_amount_limit_exceeded + ...)` to normalize across traffic volume.

### Known Gaps (Semantic – Revised)
Remaining (post implementation of anomaly rates & full restoration):
- Adaptive per-category thresholds absent (static manual interpretation only).
- No per-currency breakdown (avoids cardinality increase for now).
- Restriction mismatch still first-hit only; multi-key mismatch reporting deferred.
- No historical rate persistence (rates recomputed from in-memory snapshot history only).

Removed Gaps (Addressed):
- Per-category anomaly rate computation (implemented).
- Restoration setter for semantic counters (`SetSemanticSnapshot`).
- Hash chain integrity for semantic persistence file.

Owner: Authorization/Delegation Subsystem Maintainers.
Stability: alpha (subject to naming and payload changes pre-1.0).

### Detached Signature Missing Metric (NEW)

When strict detached signature enforcement is enabled (`GAUTH_REQUIRE_DETACHED_SIGNATURE=1`), requests missing the required detached signature artifact increment a dedicated counter:

```
# HELP gauth_rfc0111_crypto_signature_missing_total Missing detached signature artifact events when strict enforcement is enabled
# TYPE gauth_rfc0111_crypto_signature_missing_total counter
gauth_rfc0111_crypto_signature_missing_total 0
```

Purpose:
- Distinguish genuine signature verification failures from client integration issues (missing artifact).
- Provide rollout visibility when enabling detached signature enforcement gradually.
- Enable alerting before elevated missing rates begin to impact availability.

Suggested PromQL:
```promql
# 10m rate of missing detached signatures
sum(increase(gauth_rfc0111_crypto_signature_missing_total[10m]))

# Ratio of missing events vs total attestation verifications
sum(increase(gauth_rfc0111_crypto_signature_missing_total[5m])) /
clamp_min(sum(increase(gauth_rfc0111_attestation_proof_verifications_total[5m])), 1)
```

Alert Example:
```yaml
ALERT DetachedSignatureMissingSpike
	IF increase(gauth_rfc0111_crypto_signature_missing_total[15m]) > 25
	FOR 5m
	LABELS {severity="medium"}
	ANNOTATIONS {summary="Detached signature missing spike", description=">25 missing detached signature events in 15m window (strict mode)."}
```

Runbook:
1. Confirm strict mode flag enabled on all pods.
2. Inspect client request construction—ensure detached signature attached (header or claim as per integration contract).
3. Sample several denied requests; verify absence of `detached_signature` claim.
4. Cross-check recent deployment of client libraries for regression.
5. If spike coincides with key rotation, validate clients didn't drop signature field during error handling fallback.

SLO Draft:
- Missing Detached Signature Ratio < 0.5% of attestation verifications per rolling 1h.

Roadmap:
- Add labeled variant `crypto_signature_missing_total{client="<id>"}` with small allowlisted client IDs (guard cardinality).
- Histogram of detached signature verification latency (to correlate slow path vs missing path).
- Audit event enrichment (include missing signature reason) for rapid forensic pivot.


### Attestation Proof Trust Anchor Metrics (NEW)

Granular counters now distinguish specific trust anchor enforcement failure modes during attestation proof verification. These augment the generic `attestation_proof_verification_failures_total` counter to enable precise alerting and configuration drift detection.

Prometheus Counters:
```
# HELP gauth_rfc0111_attestation_proof_trust_anchor_missing_total Attestation proof verification failures due to missing trust anchor
# TYPE gauth_rfc0111_attestation_proof_trust_anchor_missing_total counter
gauth_rfc0111_attestation_proof_trust_anchor_missing_total 0

# HELP gauth_rfc0111_attestation_proof_trust_anchor_algorithm_mismatch_total Attestation proof verification failures due to algorithm mismatch with trust anchor
# TYPE gauth_rfc0111_attestation_proof_trust_anchor_algorithm_mismatch_total counter
gauth_rfc0111_attestation_proof_trust_anchor_algorithm_mismatch_total 0

# HELP gauth_rfc0111_attestation_proof_trust_anchor_key_mismatch_total Attestation proof verification failures due to key mismatch with trust anchor
# TYPE gauth_rfc0111_attestation_proof_trust_anchor_key_mismatch_total counter
gauth_rfc0111_attestation_proof_trust_anchor_key_mismatch_total 0
```

Failure Taxonomy:
| Counter | Condition | Typical Cause | Action |
|---------|-----------|---------------|--------|
| missing | Issuer not present in trust anchor registry | Registry drift, stale config, mis-deployed anchor JSON | Reload anchors, verify distribution channel |
| algorithm_mismatch | Attestation proof signature algorithm differs from registered anchor algorithm | Unexpected algorithm rotation, misconfiguration | Confirm intended rotation; update anchor metadata |
| key_mismatch | Key ID found but public key bytes differ | Stale key material, partial rotation, possible tampering | Reconcile key rotation logs; rotate & re-issue proofs |

Environment Toggle:
`GAUTH_ATTEST_REQUIRE_TRUST_ANCHOR=1` enables strict enforcement. When unset, proofs still verify cryptographically but trust anchor counters remain zero (soft mode) unless failures occur in branches guarded by strict mode.

Suggested PromQL:
```promql
# 5m rate of total trust anchor failures
sum(increase(gauth_rfc0111_attestation_proof_trust_anchor_missing_total[5m])) +
sum(increase(gauth_rfc0111_attestation_proof_trust_anchor_algorithm_mismatch_total[5m])) +
sum(increase(gauth_rfc0111_attestation_proof_trust_anchor_key_mismatch_total[5m]))

# Failure ratio vs all verifications (guard divide-by-zero)
(sum(increase(gauth_rfc0111_attestation_proof_trust_anchor_missing_total[5m])) +
 sum(increase(gauth_rfc0111_attestation_proof_trust_anchor_algorithm_mismatch_total[5m])) +
 sum(increase(gauth_rfc0111_attestation_proof_trust_anchor_key_mismatch_total[5m]))) /
 clamp_min(sum(increase(gauth_rfc0111_attestation_proof_verifications_total[5m])), 1)

# Algorithm mismatch spike detection (configuration drift)
increase(gauth_rfc0111_attestation_proof_trust_anchor_algorithm_mismatch_total[15m]) > 3
```

Alert Examples:
```
ALERT AttestationTrustAnchorFailureSpike
	IF (sum(increase(gauth_rfc0111_attestation_proof_trust_anchor_missing_total[10m])) +
			sum(increase(gauth_rfc0111_attestation_proof_trust_anchor_algorithm_mismatch_total[10m])) +
			sum(increase(gauth_rfc0111_attestation_proof_trust_anchor_key_mismatch_total[10m]))) > 10
	FOR 5m
	LABELS {severity="high"}
	ANNOTATIONS {summary="Attestation trust anchor failure spike", description=">10 trust anchor verification failures in 10m window."}

ALERT AttestationAlgorithmDrift
	IF increase(gauth_rfc0111_attestation_proof_trust_anchor_algorithm_mismatch_total[30m]) > 5
	FOR 10m
	LABELS {severity="medium"}
	ANNOTATIONS {summary="Attestation algorithm mismatch surge", description=">5 algorithm mismatches in 30m; verify anchor algorithm metadata."}

ALERT AttestationKeyMismatchPersistent
	IF increase(gauth_rfc0111_attestation_proof_trust_anchor_key_mismatch_total[1h]) > 0
		AND increase(gauth_rfc0111_attestation_proof_trust_anchor_key_mismatch_total[1h]) == increase(gauth_rfc0111_attestation_proof_trust_anchor_key_mismatch_total[2h])
	FOR 15m
	LABELS {severity="warning"}
	ANNOTATIONS {summary="Persistent attestation key mismatch", description="Key mismatch failures sustained across consecutive hours."}
```

Operational Runbook:
1. Missing Anchor: Confirm `capabilities.json` or anchor registry source; diff currently loaded anchors vs expected artifact.
2. Algorithm Mismatch: Inspect recent anchor rotation change logs; ensure dependent services updated; verify no mixed algorithm issuance.
3. Key Mismatch: Retrieve current anchor public key bytes; compare against issuance key ring; rotate and reissue if compromised.
4. All: Correlate with deployment events; check audit trail for unauthorized anchor modifications.

Hardening Roadmap:
- Add labeled counter for `trust_anchor_expired` (future) when anchor timestamps supported.
- Emit gauge `attestation_trust_anchor_last_refresh_age_seconds` to alert on stale registry refresh.
- Integrate anomaly detection (EWMA) for mismatch ratio vs baseline.
- Provide CLI `gauth attestation anchors verify` to offline validate registry consistency.

Testing Guidance:
- Unit tests inject memory metrics and simulate each failure path asserting only the relevant counter increments.
- Integration tests set `GAUTH_ATTEST_REQUIRE_TRUST_ANCHOR=1` and rotate algorithms/keys to ensure mismatch counters fire.
- Use ephemeral anchor with deliberate wrong algorithm to validate algorithm mismatch counter before production rotations.

SLO Considerations (initial draft):
- Trust Anchor Failure Ratio < 0.1% of total attestation proof verifications per rolling 1h.
- Algorithm Mismatch Count = 0 outside planned rotation windows.
- Key Mismatch Count = 0 (any occurrence treated as potential security event).

Owner: Cryptography / Attestation Subsystem Maintainers.
Stability: beta (metrics names considered stable; additional labels may be added post 1.0).

