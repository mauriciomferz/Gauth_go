---
title: Observability Plan
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Observability Enhancement Plan (Phase 1 Beta)

Objective: Achieve actionable visibility for security & governance operations (replay protection, PoA issuance, attestation verification, ledger rotation) while laying groundwork for adaptive anomaly detection.

## Scope (RB9 & RB11)
- Tracing: Token issue/validate, PoA issue, attestation verify, revocation action, rotation append.
- Metrics Additions: WAL flush latency histogram, snapshot duration gauge, pending WAL entries, attestation verify latency histogram.
- Logging Hygiene: Structured fields (event, poa_id, jti, nonce, rotation_hash, error_code). No secret material.
- Alerting: Basic Prometheus rules for replay spike, WAL flush latency, rotation failure count.

## OpenTelemetry (OTEL) Tracing
Spans:
- `token.issue` attributes: poa_version, signer_threshold, jti.
- `token.validate` attributes: validation_outcome (ok|replay|expired|invalid_sig), algorithm.
- `poa.issue` attributes: weights_json, digest_version.
- `attestation.verify` attributes: sig_mode, nonce_replay(boolean), signer_count.
- `revocation.apply` attributes: poa_id, leaf_index, tree_size.
- `rotation.append` attributes: new_key_set_size, prev_hash, new_hash.
 - `rotation.perform` attributes: prev_kid, new_kid, ttl_hours, history_size.
Exporter: OTLP (stdout fallback). Sampling: ParentBased(TraceIDRatio=0.25) initial.

## Prometheus Metrics (Additions)
- Counter: `agentauth_replay_wal_writes_total` labels: result(success|fail).
- Histogram: `agentauth_replay_wal_flush_latency_ms` (buckets: 1,2,4,8,16,32,64).
- Gauge: `agentauth_replay_wal_pending_entries`.
- Histogram: `agentauth_attestation_verify_latency_ms` (1,2,4,8,16,32,64,128,256).
- Counter: `agentauth_rotation_signature_failures_total`.
- Gauge: `agentauth_revocation_tree_size`.
 - Counter: `agentauth_attestation_verify_total` labels: outcome(success|failure), soft_invalid(true|false).

## Alert Rules (monitoring/)
1. Replay Spike:
```
ALERT ReplaySpike
  IF increase(agentauth_replay_detected_total[5m]) > 50
  FOR 2m
  LABELS { severity = "warning" }
  ANNOTATIONS { summary = "High replay detection volume" }
```
2. WAL Flush Latency:
```
ALERT WALFlushHigh
  IF histogram_quantile(0.95, sum(rate(agentauth_replay_wal_flush_latency_ms_bucket[5m]) by (le) > 32
  FOR 2m
  LABELS { severity = "critical" }
  ANNOTATIONS { summary = "WAL flush p95 latency high" }
```
3. Rotation Failure:
```
ALERT RotationSignatureFailures
  IF increase(agentauth_rotation_signature_failures_total[15m]) > 0
  FOR 1m
  LABELS { severity = "critical" }
  ANNOTATIONS { summary = "Rotation signature failures detected" }
```

## Logging Conventions
Use JSON lines: `{"ts":"RFC3339","level":"info","event":"token.validate","jti":"...","outcome":"replay"}`.
Error Path: include `error_code` and `rfc_ref` matching taxonomy.

## Instrumentation Checklist
- Wrap handler functions with tracing start/end helpers.
- WAL implementation exposes flush/snapshot events (emit span + metrics).
- Rotation append function increments failure counter on signature mismatch.
- Attestation verify path records latency histogram regardless of outcome.

## Privacy & Security
- Exclude raw PoA signer public keys from logs (only counts & digest).
- Do not log JWTs or nonce values after replay detection (log first 8 hex chars hash only).
- Ensure OTEL resource attributes exclude secrets.

## Phase 2 (Post-Beta Preview)
- Adaptive anomaly scoring (e.g. unusual replay slope) exported as gauge.
- Trace linking to external anchor receipts via `anchor.receipt_id`.
- PromQL recording rules for daily digest of security events.

---
Revise after WAL implementation and handler modularization (RB1, RB8).
