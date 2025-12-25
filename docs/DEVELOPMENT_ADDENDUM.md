---
title: Development Addendum
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Development Addendum

> Last Updated: 2025-10-18
> Status: Active

Refer to [`CODE_STYLE.md`](CODE_STYLE.md) for canonical code formatting, lint target usage, constants guidelines, and error handling patterns. This supplements sections in `DEVELOPMENT.md`.

## Multi-Signature Verification Enhancements

Default count mode: number of valid signatures must be >= `GAUTH_MULTI_SIG_THRESHOLD`.

Weighted mode (enabled by `GAUTH_MULTI_SIG_WEIGHTS` – comma-separated integers matching signer order):
* Structural count pre-check relaxed (fewer signers than threshold permitted if cumulative weight >= threshold).
* Cumulative weight of valid signatures compared directly to threshold.
* Metric: `multi_signature_weight_failures_total` increments on insufficient cumulative weight.
* Regression tests: `multi_signature_weight_test.go` (success + insufficient weight failure).

## Token & Delegation Lifecycle Status

Statuses: `active`, `suspended`, `terminated`.

Allowed transitions:
* active -> suspended | terminated
* suspended -> active | terminated
* terminated -> terminated (idempotent only)

Terminal guard prevents non-idempotent transitions out of `terminated` (HTTP 409).

Validation mapping:
* Suspended tokens validate as `suspended`.
* Terminated tokens validate as `revoked` (unified downstream revocation semantics).

Persistence (prototype):
* Tokens: status field on in-memory token entries.
* Delegations: `BetaServer.delegationStatus` map (mutex guarded).

Tests:
* `token_status_test.go` – transitions, reactivation, terminal guard, validation mapping.
* `delegation_status_test.go` – initialization, cycling, termination, terminal guard, unsupported status rejection, no-op.
* `delegation_lifecycle_metrics_test.go` & lifecycle metrics token test validate counters.

## Metrics (wired)
Counters now active (memory & Prometheus):
* `token_status_transitions_total`, `token_status_transition_failures_total`
* `delegation_status_transitions_total`, `delegation_status_transition_failures_total`
* `multi_signature_weight_failures_total`
* Decisions labeled via `decision_total{action,resource,outcome}` for lifecycle updates.

### Labeled Lifecycle Transition Metrics (Implemented)

Both in-memory and Prometheus collectors now support fine-grained lifecycle transition labeling:

Prometheus:
* `token_lifecycle_transition_total{old="<old>",new="<new>",outcome="<success|failure|noop>"}`
* `delegation_lifecycle_transition_total{old="<old>",new="<new>",outcome="<success|failure|noop>"}`

In-memory snapshot enrichment:
* `LifecycleBreakdown` map keyed by `entity|old|new|outcome` (underscore `_` denotes initialization when no prior state existed).
* `DecisionBreakdown` map keyed by `action|resource|outcome`.

Label normalization rules:
* Empty old status -> `_` for delegation initialization (prevents ambiguous empty string).
* No-op transitions (requested status == current) recorded with `outcome="noop"` (currently used internally; may be exposed later if needed for thrash detection).

Failure semantics:
* Terminal guard (e.g., `terminated -> active`) increments failure labels (`outcome="failure"`).
* Unsupported status values produce failure entries without state mutation.

Primary consumers:
* Audit / compliance correlation (failed resurrection attempts).
* Policy engine tuning (frequency of suspension cycles).
* Operational SLO design (later latency histograms can join labels).

Events emitted:
* `token_status_changed` (payload: token_id, old_status, new_status)
* `delegation_status_changed` (payload: delegation_id, old_status, new_status, initialized?)

### Decision Reason Codes & Lifecycle Latency (Implemented)

Reason codes now accompany lifecycle status updates (audit + events + metrics):
* `init` – first delegation status creation
* `status_change` – successful mutation
* `invalid_transition` – rejected terminal or unsupported change
* `unsupported_status` – non-whitelisted status value
* `invalid_payload` – schema / required field failure
* `not_found` – missing token identifier
* `noop` – idempotent request (state unchanged)

Metrics additions:
* Prometheus `decision_reason_total{action,resource,outcome,reason}` CounterVec.
* Memory snapshot `decision_reason_breakdown` map keyed `action|resource|outcome|reason`.

Lifecycle transition latency instrumentation:
* Prometheus `lifecycle_transition_seconds{entity="token"|"delegation",outcome="success"|"failure"|"noop"}` histogram (outcome label added for differential latency analysis).
* Memory aggregates (outcome-labeled): `lifecycle_latency_totals_ns`, `lifecycle_latency_counts`, `lifecycle_latency_max_ns` keyed by `entity|outcome` (e.g. `token|success`).
* Memory percentile estimates (ring-buffer reservoir, last 64 samples) per `entity|outcome`: `lifecycle_latency_p50_ns`, `lifecycle_latency_p90_ns`, `lifecycle_latency_p99_ns`.

Intended usage:
* Detect handler regression (spikes in max / p99 for `success` path).
* Compare `noop` vs `success` latency to uncover unnecessary work.
* Highlight pathological failure handling if `failure` p99 diverges.
* Provide groundwork for SLOs (e.g., 99% of successful lifecycle transitions < 5ms).

Outcome-labeled latency completed; percentiles implemented. Reservoir size kept intentionally small (64) for low overhead; can be tuned later.

Persistence (metrics prototype):
* Environment variable `GAUTH_METRICS_PERSIST_PATH` enables file-backed persistence.
* On startup: if file exists, labeled counters (`decision_counts`, `decision_reason_counts`, `lifecycle_counts`) and status transition counters restored.
* On graceful shutdown (signal or server close): snapshot saved (JSON) to path.
* Use cases: retain counters across deployments; cold-start dashboards without losing historical transition volume.
* NOT persisted: latency aggregates & reservoirs (ephemeral, runtime quality signal only).

## Cross-References
* `POLICY_ENGINE.md` – policy interpretation for suspended vs revoked.
* `EVENT_SYSTEM.md` – lifecycle events & token revocation event.
* `CRYPTOGRAPHY_IMPLEMENTATION.md` – multi-signature semantics.

## Future Work
1. Durable storage layer for delegation lifecycle states (beyond current in-memory map).
2. Extended reason taxonomy (policy_violation, rate_limited, maintenance) & mapping logic.
3. Introspection endpoint enrichment with lifecycle & reason history timeline.
4. Periodic autosave of metrics (crash resilience) & optional latency persistence.
5. Optional OpenTelemetry span emission for lifecycle transitions (attributes: entity, old_status, new_status, outcome, reason, latency_ns).
6. Prometheus / OTEL bridging for percentile export (client-side aggregation or exemplar linkage).
7. Pluggable metrics backend selection (env: memory | prometheus | otel-direct).

### Validation Failure Counters (2025-10-18)

Introduced dedicated atomic counters for lifecycle validation failure reasons to separate structural hygiene issues from general transition failures:

Reasons tracked:
* invalid_payload – Malformed JSON or missing required fields in status update request.
* unsupported_status – Provided `new_status` not in allowed set (`active|suspended|terminated`).
* invalid_transition – State machine violation (e.g., attempting transition out of terminal `terminated`).
* not_found – Referenced token or delegation ID does not exist for update.

Implementation details:
* Backing fields (unexported) in `metrics.Memory` with accessor methods (e.g., `InvalidPayloadFailures()`).
* Persistence extended to include these counters in the metrics snapshot JSON.
* Prometheus exposition (`apiPolicyMetricsPrometheus`) now emits HELP/TYPE and counter lines:
	- `gauth_validation_invalid_payload_total`
	- `gauth_validation_unsupported_status_total`
	- `gauth_validation_invalid_transition_total`
	- `gauth_validation_not_found_total`
* Handlers (`apiTokenStatusUpdate`, `apiDelegationStatusUpdate`) increment counters in failure branches in addition to existing lifecycle failure counters.

Rationale & Usage:
* Enables dashboards to distinguish correctness hygiene issues from legitimate transition failures (e.g., business rule violations) and to set error budgets.
* Counters complement lifecycle breakdown keys (which already label failure reasons) by providing direct aggregate series for alerting (e.g., spike in invalid_payload failures suggests client regressions).
* Next expansion will include semantic PoA validator rejection reasons (scope_violation, restriction_violation) into same family for unified hygiene monitoring.

Testing:
* `validation_failure_counters_test.go` exercises each failure path for both token and delegation endpoints and asserts counter increments.

### Alerting & SLO Guidance (2025-10-18)

Suggested initial alert rules (Prometheus-style pseudo syntax) leveraging new counters:

1. Spike in malformed client requests (potential client regression):
	- Expr: increase(gauth_validation_invalid_payload_total[5m]) > 25
	- Action: Page if sustained 3 intervals; annotate deployment timeline.

2. Unexpected growth in unsupported status usage (possible abuse / outdated clients):
	- Expr: increase(gauth_validation_unsupported_status_total[15m]) > 5
	- Action: Create low-priority ticket; correlate with user agent logs.

3. Invalid transition anomaly (potential resurrection attempts or logic bug):
	- Expr: rate(gauth_validation_invalid_transition_total[10m]) > 0.1
	- Action: Security review if coupled with authz deny spikes.

4. Not-found churn (guessing IDs / enumeration):
	- Expr: rate(gauth_validation_not_found_total[5m]) / rate(token_status_transitions_total[5m]) > 0.3
	- Action: Trigger rate limiting review; consider IP-based throttling.

5. Lifecycle latency SLO (success transitions):
	- SLO: 99% < 5ms
	- Burn alert (fast-burn): lifecycle_latency_p99_ns{entity|outcome="token|success"} > 8e6 for 5m.

Dashboards:
* Panel group: "Lifecycle Hygiene" showing stacked area of the four failure counters.
* Ratio panels: invalid_transition / total_failures; not_found / total_requests.
* Latency correlation: overlay p99 success vs failure to catch pathological failures.

Capacity Planning:
* Use cumulative successful transitions + invalid transitions to approximate write path pressure when introducing persistent backend; size queues accordingly.

### Metrics Interface Parity Decision (2025-10-18)

Decision: Do NOT add explicit getter or increment methods for validation failure counters to the generic `Metrics` interface at this time.

Rationale:
* Current generic interface consumed primarily for increment side-effects; read paths already branch on concrete `*metrics.Memory` for snapshots.
* Adding four more Inc* methods would expand interface surface (risking churn) before confirming production need in Prometheus adapter (which can export direct counters independently if/when added).
* Keeps alternative backends (e.g., OTEL direct) simpler during experimental phase.

Future Revisit Criteria:
* Need to increment these counters from packages outside web handler layer.
* Requirement to swap Memory with Prometheus adapter while preserving increments without type assertions.
* Standardization of extended validation taxonomy (semantic validator reasons) across components.

If criteria met, evolve interface using additive versioning (e.g., `type ExtendedMetrics interface { Metrics; IncInvalidPayloadFailure(); ... }`) to preserve backward compatibility.

