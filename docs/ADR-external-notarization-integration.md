# ADR: External Notarization / Timestamp Integration for Capability & Audit Chains

Status: Proposed
Date: 2025-10-20
Decision Drivers:

Summary: Introduces external notarization for capability registry and audit chain tips using TSA and transparency log integration, metrics, verification endpoints, and security considerations. Implementation in beta-refactor branch.

References: See GAP_MATRIX Section 13, implementation in `internal/notary/provider.go`, tests in `web/notarization_test.go`.

## Context
The system emits:
- Capability registry anchor artifacts (hash + timestamp + optional Ed25519 signature) at interval.
- Audit ledger chain tips (hash chain in BoltDB) with integrity verification endpoints.
Gaps called out in `GAP_MATRIX.md`: missing external timestamp/notarization (revocation anchoring, immutable ledger external anchor, capability anchoring external proof).

## Goals
1. Provide cryptographically verifiable, append-only external timestamp / transparency proof for:
   - Capability registry periodic anchor artifacts
   - Audit ledger chain tip hashes (issuance/revocation events)
   - (Future) Revocation chain tip & semantic anomaly snapshots
2. Enable retrieval & verification endpoints returning inclusion/consistency proofs.
3. Integrate latency, success/failure, age metrics (extending existing Prometheus collectors).
4. Preserve minimal operational overhead (pluggable provider abstraction).

## Non-Goals
- Building a full transparency log from scratch.
- On-chain settlement or blockchain integration.
- Formal SLA enforcement (first iteration).

## Solution Overview
Introduce a `NotaryProvider` interface with methods:
```
Submit(hash []byte, kind ArtifactKind) (Receipt, error)
Get(receiptID string) (Receipt, error)
Verify(hash []byte, receipt Receipt) (VerificationResult, error)
```
Artifact kinds: CapabilityAnchor, AuditLedgerTip, RevocationChainTip, SemanticSnapshot.

Provider implementations (phased):
1. RFC Phase 1: RFC3161 TSA via HTTP (timestamp request, DER token parse).
2. Phase 2: Transparency Log (e.g. Sigstore, Trillian) using add-entry + inclusion proof.
3. Phase 3: Dual notarization (TSA + transparency) with cross-proof correlation.

Receipt structure (JSON + binary fields):
```
{
  "id": "ts-<uuid>",
  "hash": "base64", // original artifact hash
  "provider": "tsa-rfc3161" | "transparency" | "dual",
  "submitted_at": "RFC3339",
  "notarized_at": "RFC3339",
  "latency_ms": <number>,
  "proof_type": "rfc3161" | "inclusion" | "dual",
  "proof": "base64", // DER token or inclusion proof bundle
  "chain_head_hash": "base64" // optional for ledger submissions
}
```

## Metrics Additions
- `gauth_capability_anchor_notarization_latency_seconds` (already present) extended with provider label.
- `gauth_notarization_failures_total` (counter, labeled by kind/provider).
- `gauth_notarization_age_seconds` (gauge per kind: capability, audit, revocation).
- `gauth_notarization_pending_total` (current in-flight submissions).
- `capability_anchor_notarization_receipts_integrity` (gauge – receipt hash-chain integrity: ok=1 mismatch=0 unconfigured/legacy/empty=-1)

### Prototype Provider Selection (Implemented)

Environment variables now enable choosing a stub external provider:

| Env | Purpose | Values | Default |
| --- | --- | --- | --- |
| `GAUTH_CAP_ANCHOR_NOTARIZE` | Enable capability anchor notarization path | `1` to enable | unset |
| `GAUTH_CAP_ANCHOR_NOTARY_PROVIDER` | Select provider implementation | `memory`, `external_stub` | `memory` |
| `GAUTH_NOTARY_STUB_MIN_LATENCY_MS` | Minimum simulated latency (ms) | 0-1000 | 40 |
| `GAUTH_NOTARY_STUB_MAX_LATENCY_MS` | Maximum simulated latency (ms) | >=min, <=5000 | 250 |
| `GAUTH_NOTARY_STUB_FAIL_PROB` | Failure probability (0.0-1.0) | float | 0 |
| `GAUTH_NOTARY_STUB_PROVIDER_NAME` | Override receipt provider field | string | `external_stub` |

The `external_stub` provider offers randomized latency and probabilistic failures to exercise metrics & error paths. It does NOT provide cryptographic guarantees and must be replaced before marking GAP_MATRIX external anchoring items as fully implemented.

## Endpoint Extensions
- `/api/v1/beta/capabilities/anchor/status` add `last_notarization_receipt_id`, `notarization_provider`, `notarization_verified`.
- New: `/api/v1/beta/audit/anchor/notarize` triggers external submission of latest ledger tip.
- New: `/api/v1/beta/notarization/receipt/:id` returns receipt JSON.
- New: `/api/v1/beta/notarization/verify` POST { hash, receipt_id } → verification result.
- Prototype Persistence Verification: `GET /api/v1/beta/notarization/receipts/verify` returns hash-chain integrity status for persisted receipt chain (ok|mismatch|empty|unconfigured) and chain head.

## Verification Flow
1. Client fetches artifact + receipt ID.
2. Calls verification endpoint or locally verifies (transparency proof or TSA token).
3. Compares hash equality + signature/time validity (policy: max skew, not-after constraints).

## Security Considerations
- Ensure provider responses are authenticated (TLS + response signature if available).
- Prevent hash substitution: include artifact kind & size metadata in receipt.
- Rate limit notarization to avoid provider DoS.
- Store receipts durably (BoltDB bucket) with hash chain of receipt entries for secondary integrity.

## Data Model Changes
- New Bolt bucket: `notary_receipts` (key=receipt_id, value=JSON).
- Optional `receipt_chain` (append-only hash chain of receipt JSON digests).

## Migration Plan
1. Implement `NotaryProvider` adapter with memory + TSA implementation stub.
2. Add provider configuration env vars: `GAUTH_NOTARY_PROVIDER`, `GAUTH_NOTARY_TSA_URL`.
3. Register metrics & extend status endpoint.
4. Add receipt persistence layer + tests (inclusion of hash, consistent retrieval).
5. Build verification endpoint with TSA token checks (timestamp validity, imprint match).
6. Documentation updates (OBSERVABILITY.md, COMPLETE_API_REFERENCE.md, GAP_MATRIX).
7. Add conformance tests for receipt fetch & verify (happy path + negative imprint mismatch).

## Alternatives Considered
- Direct blockchain anchoring: higher cost & latency; deferred.
- Rolling local Merkle accumulator only: lacks external trust root.

## Risks
- Provider latency impacting anchor emission path (mitigate with async queue).
- Receipt storage growth (periodic pruning strategy needed later).

## Open Questions
- Standardizing dual proof format (TSA + transparency) bundling.
- Inclusion of policy hash in capability anchor receipts.

## Success Criteria
- External receipt stored & retrievable for capability and audit chain tip.
- Verification endpoint returns success for valid receipt + hash.
- Metrics show latency and age progression; failure counter increments on simulated provider errors.
- GAP_MATRIX entries for external anchoring moved from Partial to Implemented (with residual advanced gaps noted).

## Follow-Up
- Phase 2 integration with transparency log service.
- Dual notarization ADR.
