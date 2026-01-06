---
title: ADR Multi-Signature / Threshold Enforcement for Proof of Authorization
category: adr
status: proposed
lastUpdated: 2025-11-12
owners: architecture-team
source: internal
refreshCadence: on-change
---
# ADR: Multi-Signature / Threshold Enforcement for Proof of Authorization

Status: Proposed
Date: 2025-10-20
Drivers:

Summary: Proposes N-of-M multi-signature/threshold enforcement for Proof of Authorization, verification API, metrics, and audit log integration. Implementation in beta-refactor branch.

References: See GAP_MATRIX Section 15, implementation in `internal/poa/multisig.go`, tests in `internal/poa/multisig_test.go`.

## Problem Statement
Delegations / PoAs needing multiple principals (e.g., dual control) lack cryptographic enforcement. Without threshold signatures, policy evaluation cannot assert collective approval, weakening compliance and auditability.

## Goals
1. Support N-of-M threshold signing for a PoA.
2. Preserve deterministic canonical digest prior to aggregation.
3. Provide verification API & metrics for multi-signature events.
4. Allow heterogeneous key types (initially Ed25519 only; extensible to ECDSA/P256).

## Non-Goals
- Implement advanced MPC / aggregated Schnorr protocols (initial phase).
- Dynamic threshold changes mid-lifecycle.

## Design Overview
Two models evaluated:
1. Simple Multi-Signature Set: Store array of individual signatures over identical canonical digest. Threshold satisfied when count >= required.
2. Aggregated Threshold Signature (future): Compress multiple signatures into a single aggregate (e.g., Ed25519-BLS hybrid). Deferred.

Initial approach: Multi-Signature Set.

### Data Model Extensions
```
PowerOfAttorney {
  ...existing fields,
  required_signers: [SignerID],   // ordered for digest stability
  threshold: int,                 // <= len(required_signers)
  signatures: [SignatureRecord]   // collected signatures
}
SignatureRecord {
  signer_id: string,
  public_key_kid: string,
  sig: base64, // raw Ed25519 signature
  signed_at: RFC3339
}
```
Canonical digest excludes `signatures` array; includes `required_signers` & `threshold` to prevent replay downgrade.

### Signing Flow
1. Issuance creates PoA with empty `signatures`.
2. Authorized signers call `/api/v1/beta/poa/sign` providing signer_id + signature.
3. Server verifies signature over canonical digest; appends record if unique.
4. Once `len(signatures) >= threshold`, PoA status transitions to `active_collective`.

### Verification
`ValidateMultiSignature(poA)` ensures:
- `threshold <= len(required_signers)`.
- Each signature unique by signer_id.
- All signatures valid over canonical digest.
- Count >= threshold → returns success.
Metrics: `multi_signature_verifications_total`, `multi_signature_verification_failures_total` already exist; extend with `multi_signature_threshold_failures_total` (implemented) & new gauge `multi_signature_completion_ratio` (signatures_collected/threshold).

### API Additions
- `POST /api/v1/beta/poa/sign` (body: poa_id, signer_id, signature).
- `GET /api/v1/beta/poa/:id/multisig/status` → { collected, threshold, required_signers, remaining }.

### Security Considerations
- Prevent signature replay: enforce signer uniqueness.
- Protect against threshold downgrade: canonical digest ties threshold to signed content.
- Rate limit signature submission endpoint.
- Audit log each signature append (hash chain).

### Migration
- Add fields with backward compatibility (threshold=1 default ensures current behavior unaffected).
- Recompute canonical digest only when required_signers/threshold set.

### Open Questions
- Do we need per-signer capability constraints? (Future extension.)
- Aggregated signature optimization timeline vs. performance needs.

### Alternatives
- Use BLS for native aggregation (requires new key infrastructure; higher complexity).
- Offload to external signature orchestration service (latency/risk of external dependency).

### Success Criteria
- Threshold PoA can be constructed, partially signed, and activated when criteria met.
- Verification endpoint returns correct status & metrics reflect completions.
- GAP_MATRIX entries for joint signatures move from Missing to Partial (later Implemented after aggregator or advanced features).

### Follow-Up
- ADR for aggregated signature compression.
- Add property tests ensuring canonical digest stability with multi-sig fields.
