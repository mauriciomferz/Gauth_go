---
title: Replay Protection
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Replay Protection Architecture

This document describes the layered replay protection mechanisms implemented in AgentAuth for delegation artifacts, tokens, and attestation-like signatures.

## Layers

1. Token JTI (JWT `jti` claim): Prevents reuse of extended PoA-bound tokens within their lifetime. The JTI store enforces single consumption semantics (or short-lived uniqueness window depending on configuration).
2. Attestation Nonce: Domain-separated attestation verification requires a fresh nonce for each challenge/response cycle, blocking replay of previously valid attestations in a new verification context.
3. Signature Digest Replay Store (NEW): Prevents re-submission of identical signatures over the same canonical PoA digest by the same key identifier. A compound key `digestHex|keyID` is recorded on first acceptance. Subsequent attempts return a `409` with reason `signature_replay_detected`.

## Canonical Digest Boundary

The canonical digest excludes mutable or observer-specific fields ensuring any substantive PoA mutation yields a different digest. This prevents signature replays across divergent PoA states and limits replay detection scope to identical artifacts.

## Store Interface

`SignatureReplayStore` exposes:
```
SeenSignature(ctx, digestHex, keyID) (bool, error)
RecordSignature(ctx, digestHex, keyID) error
```
Compound key uniqueness ensures O(1) detection for both in-memory and persistent backends (BoltDB, Redis). WAL-backed adapters may be used for durability.

## Fail-Open vs Fail-Closed Behavior

An environment configuration controls whether backend store errors cause:
- Fail-Open: Allow issuance/signature (logs error; metrics `replay_store_errors_total` increment).
- Fail-Closed: Reject operation to preserve anti-replay guarantees (returns `500` or mapped conflict depending on path). This is recommended for high-assurance deployments where store availability is critical.

## Metrics

The following metrics are emitted:
- `gauth_replay_hits_total`: Count of detected signature replay attempts (blocked).
- `gauth_replay_misses_total`: Count of first-time valid signatures accepted (new digest+keyID).
- `gauth_replay_store_errors_total`: Backend errors encountered (whether fail-open or fail-closed).
- `gauth_replay_store_latency_seconds`: Histogram/summary (depending on backend) measuring Seen/Record operations latency.

Prometheus exposition names may differ (adapter-specific); reference internal metrics implementation for exact naming if scraping.

## Error Semantics

On replay detection the API returns HTTP 409 with an error envelope:
```json
{
  "success": false,
  "reason": "signature_replay_detected",
  "message": "Signature already recorded for canonical digest and key"
}
```

Clients SHOULD treat this as a terminal condition for that signature PoA pair. Generating a fresh PoA (changed scope, validity, jurisdiction, etc.) yields a new canonical digest allowing a new signature to proceed.

## Operational Guidance

| Scenario | Recommended Mode |
|----------|------------------|
| Development / Demo | Fail-Open (availability prioritized) |
| Staging | Fail-Open initially; monitor error rate, then evaluate Fail-Closed |
| Production (regulated) | Fail-Closed with persistent store + alerting on error spikes |

Set capacity/TTL policies for persistent stores to avoid unbounded growth. Since digest values are collision-resistant hashes of canonical artifacts, storage growth aligns with unique PoA issuance volume.

## Testing Summary

Unit tests cover:
- First signature acceptance (miss path)
- Second identical signature rejection (hit path)
- Backend error scenarios (fail-open vs fail-closed) verifying metrics and returned error.

## Future Enhancements

- Time-window pruning (rolling horizon) for very high-volume issuance.
- Cross-region replication of replay keys for multi-cluster deployments.
- Adaptive toggling to fail-closed when error rate below threshold and revert on sustained backend failures.

---
Updated: 2025-10-30