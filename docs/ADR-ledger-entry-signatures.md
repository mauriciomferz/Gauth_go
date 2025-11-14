---
title: ADR Ledger Entry Signatures (Rotation & Revocation Chains)
category: adr
status: proposed
lastUpdated: 2025-11-12
owners: architecture-team
source: internal
refreshCadence: on-change
---
# ADR: Ledger Entry Signatures (Rotation & Revocation Chains)

Date: 2025-10-27
Status: Proposed (Implemented for Rotation Ledger; Revocation chain entry signatures optional via global key registry)
Author: Sprint 3 Integrity Track

## Context

The rotation ledger records each active key rotation descriptor with a hash-chained sequence (prev_hash -> hash) providing tamper-evident continuity. Prior to Sprint 3 (RB5) only the aggregate ledger summary (periodic SignedTreeHead style snapshot) could be externally signed and anchored. Individual ledger entries lacked a first-party cryptographic authentication binding the raw descriptor fields to the append action. This reduced forensic resolution in the presence of partial compromise (e.g. attacker appends a forged rotation and quickly anchors a summary) and weakened selective replay defenses.

Similarly, the revocation chain introduced append-only hash chaining of `RevocationEvent` objects with Merkle accumulation and Signed Tree Heads. Per-event signatures enhance auditability and enable fine-grained authenticity validation without requiring full chain traversal.

## Problem

Without per-entry signatures:
1. An attacker with write access (but not private key material) could inject a forged rotation record referencing an existing key ID, relying on later summaries to appear legitimate until deep hash chain verification occurs.
2. External verification tooling must recompute hashes and rely solely on hash chain integrity—unable to assert origin authenticity for each append.
3. Partial ledger disclosure (subset of entries) cannot be cryptographically authenticated individually; consumers forced to request entire ledger or recent summaries.
4. Future multi-party rotation governance (threshold signing) requires a signature slot at the entry granularity; retrofitting later increases migration complexity.

## Decision

Introduce per-entry Ed25519 signatures for both rotation ledger and revocation chain entries when an active key manager (`internal/crypto.GlobalEdDSARegistry`) is present.

Signed payload: canonical JSON over the event/descriptor fields including the computed linkage hash but excluding signature fields themselves to avoid recursion.

Rotation Ledger Entry Canonical Fields:
```
{
  "id": <string>,
  "kid": <current key id>,
  "prev_hash": <string>,
  "timestamp": <RFC3339 UTC>,
  "hash": <sha256(prev_hash || canonical_descriptor_without_signature)>
}
```
Revocation Event Canonical Fields:
```
{
  "id": <revocation id>,
  "delegation_id": <optional>,
  "delegation_hash": <optional>,
  "reason": <string>,
  "revoked_at": <RFC3339 UTC>,
  "prev_hash": <string>,
  "hash": <sha256(...)>
}
```

Domain Separation:
* Rotation ledger signature domain: implicit via JSON shape + inclusion of `hash` (already domain separated during hash computation). Future: add explicit prefix `GAUTH_ROTATION_LEDGER_ENTRY:` before JSON bytes if cross-structure collision surfaces.
* Revocation event signature domain: same strategy; optional future prefix `GAUTH_REVOCATION_EVENT_V1:`.

Verification Path:
* Chain verification recomputes per-entry hash, checks prev linkage, then if `Signature` present validates using `SigKid` against key manager public keys.
* Missing signature tolerated (legacy mode) to preserve backward compatibility.
* External CLI (`cmd/ledger-verify`) performs end-to-end verification reporting mismatches & invalid signatures.

Failure Modes:
* `sig_kid` not found -> counted as public key missing (soft failure) if strict mode not enabled.
* Invalid signature bytes or verification failure -> chain verification fails early (hard failure).
* Hash mismatch triggers integrity failure regardless of signature validity.

## Alternatives Considered

1. Summary-Only Signing
   - Pros: Single signature per period; minimal overhead.
   - Cons: Cannot isolate tampering at entry granularity; requires entire chain for forensic validation; weaker partial disclosure trust.

2. Batched Entry Signatures
   - Sign a batch of N entries together (aggregation). Pros: amortizes signature overhead. Cons: increases blast radius of compromised batch; complex partial batch validation; requires buffering before exposure.

3. Merkle Leaf Signatures Only
   - Sign the Merkle leaf digests instead of full event JSON. Pros: smaller payload. Cons: loses direct binding to field semantics; requires leaf->event mapping; complicates multi-field tamper attribution.

4. Detached Log Signed Checkpoints
   - Rely on periodic STHs only; store unsigned entries. Similar drawbacks to (1) plus delayed detection window.

Decision rationale: direct per-entry signatures offer strongest authenticity with manageable overhead (Ed25519 ~64 bytes signature, fast verification). Hash chain + signature pairing yields immediate rejection of forged appends without walking future summaries.

## Consequences

Positive:
* Fine-grained authenticity: each entry independently verifiable.
* Enables incremental disclosure: small slices can be shared & verified without full chain.
* Facilitates future multi-sig (threshold) expansion by reusing per-entry signature vector.
* Simplifies external tooling—no need to reconstruct multi-level commitments for per-entry validation.

Negative / Costs:
* Increased storage (signature + key id per entry).
* Slight CPU overhead during append (Ed25519 sign) and verification (Ed25519 verify per entry).
* Dependency on global key registry; if unavailable entries remain unsigned (must be documented for auditors).

## Migration & Backward Compatibility

Existing ledgers without signatures continue to verify hash chaining. Verification logic treats absence of `Signature` as legacy entries (reports `signature_present=false` & `signature_valid=false` + `signature_error="unsigned"`).

Auditors should interpret mixed ledgers (partial signature coverage) as transitional; once signing activated all subsequent entries MUST include signatures. Consider alerting if unsigned entries appear after activation timestamp.

## Operational Considerations

* Key Rotation: Active signing key changes seamlessly; new entries use new `SigKid`. Historical entries remain valid under previous keys as long as public keys retained for verification window.
* Key Pruning: Safe only after verification horizon passes (e.g. retain public keys for at least M days); pruning early invalidates historical entry verification.
* Monitoring: Add counters for invalid entry signatures & missing public keys; alert on sudden spikes.

## Future Work

1. Explicit domain prefixes for signature canonicalization (hardening).
2. Threshold / weighted multi-sig entry support (vector of signatures + cumulative weight).
3. Detached batch signature optimization mode for high-throughput ledgers (configurable toggle).
4. Inclusion of algorithm agility (ecdsa-p256, bls aggregated) behind Signer interface.
5. External anchor emission per entry (optional) with grouping to reduce upstream load.

## Acceptance Criteria (Implemented)
* Append path signs each entry when key manager active.
* Verification fails on any signature mismatch or hash chain break.
* CLI tool outputs JSON summarizing mismatch & invalid signature counts.
* Tests cover: signature presence, tamper detection (modified descriptor triggers mismatch), unsigned legacy acceptance.

## References
* Ed25519: RFC 8032
* Hash Chain Integrity Patterns: Schneier & Kelsey (Secure Audit Logs) – per-entry MAC analogy
* Merkle Tree Append-Only Logs: RFC 6962 (conceptual consistency proofs inspiring future expansion)

---
Record ID: ADR-ledger-entry-signatures
Linked Sprint Backlog Items: RB5 (rotation ledger entry signatures), RB10 (revocation proofs – complementary authenticity)
