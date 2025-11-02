# ADR: Weighted Multi-Signature Rotation & Attestation Support (Public Key Embedding Extension)

Date: 2025-10-28
Status: Accepted (Core weighted model); Public Key Embedding: Adopted as Optional Feature Flag
Decision Drivers: Threshold flexibility, future algorithm agility, scalable signer governance.

## Context

Current rotation & attestation model assumes uniform signer weight (each signature counts as 1). As platform grows, certain domains (e.g., higher assurance HSM-backed keys, external notary signers) should contribute greater effective trust weight while still enabling smaller contributors. Additionally, algorithm diversity (ECDSA + BLS) may introduce varying trust semantics.

## Goals

Accepted (Verified-weight enforcement implemented; initial multi-algorithm schema + metrics added; full ECDSA/BLS verification & auditor tooling pending)
- Preserve deterministic canonical digest while incorporating weight semantics for verification.
- Enable threshold definition in terms of aggregate weight instead of number of signatures.
* Phase 2 (CURRENT): Verified-weight enforcement using Ed25519 signatures. Threshold must be satisfied by sum of weights for signatures passing cryptographic verification. Metrics: `gauth_rotation_v2_verified_weight`, `gauth_rotation_v2_signature_failures_total{reason}`, `gauth_rotation_v2_threshold_violations_total`.
* Phase 2a (Incremental): Multi-algorithm schema extension. Signer entries now persist an `alg` field (defaulting to `ED25519` if omitted for backward compatibility). OpenAPI updated to expose `alg` per signer. Additional metrics introduced: `gauth_rotation_v2_verified_weight_alg{alg}` and `gauth_rotation_v2_signature_failures_by_alg_total{alg,reason}` to support per-algorithm observability.
* Phase 2b (Current): Optional public key embedding behind environment flag `GAUTH_ROTATIONS_V2_EMBED_PUBS=1`, enabling offline artifact signature verification (auditor CLI) without relying on server-side key registries.

## Non-Goals
* Adding new algorithms requires extending verification logic and possibly resolver interfaces without altering canonical digest (which already binds signer algorithm identifiers). The digest preimage includes each signer's `id|alg|weight` tuple, so algorithm substitution is a detectable change.
* Per-algorithm metrics enable early detection of asymmetric failure patterns (e.g., only ECDSA signatures failing due to curve parameter misconfiguration) before threshold violations occur.
- Delegation chain weighting (out of scope for initial implementation).
- Dynamic runtime re-weighting based on external telemetry.
* Implement full multi-algorithm verification (ECDSA P-256, BLS12-381 aggregate) with corresponding resolver expansion (storing multiple key types) and hybrid threshold semantics (e.g., minimum weight AND diversity constraints).
## Proposed Design

### Canonical Rotation Artifact Changes

Add a `signers` array with explicit weights:
```json
{
  "version": 2,
  "active_key_id": "kid-123",
  "previous_rotation_hash": "abc...",
  "threshold_weight": 100,
  "signers": [
    { "id": "hsm-a", "alg": "ECDSA_P256", "weight": 60, "signature": "..." },
    { "id": "soft-b", "alg": "ED25519", "weight": 20, "signature": "..." },
    { "id": "notary-c", "alg": "ECDSA_P256", "weight": 40, "signature": "..." }
  ]
}
```

Canonical digest input (ordered normalization):
1. version
2. active_key_id
3. previous_rotation_hash
4. threshold_weight
5. algorithm_suite (CSV of sorted algorithms)
6. for each signer (sorted by id asc): `<id>|<alg>|<weight>` (NOTE: Signatures and embedded public keys are EXCLUDED from the digest to preserve digest stability irrespective of signature timing and to permit post-build signature attachment.)

Rationale for excluding signatures & public keys from digest:
* Prevents digest mutation when adding or re-attaching a signature (avoid circular dependency between digest preimage and signature bytes).
* Allows embedding of public keys as a transport optimization without altering the artifact integrity root.
* Mitigates replay or swap attacks through domain-separated signature preimage: `GAUTH_ROTATION_V2:<canonical_digest>` binds signatures to the digest even though signature bytes are not part of the digest calculation.

Integrity considerations:
Public Key Encoding Standardization:
* Ed25519: raw 32-byte public key encoded base64url.
* ECDSA P-256: uncompressed point 0x04||X||Y with X,Y left-padded to 32 bytes each, encoded base64url (helper: `EncodeECDSAP256Uncompressed`).
* Future algorithms (BLS12-381): plan to use compressed form (e.g., 48 or 96 bytes) base64url.
* PQC (future): adopt NIST-recommended binary form with explicit alg identifier.
* Signature exclusion means an attacker cannot forge a valid digest+signature pair without controlling the signer private key; digest enumerates signer identity + algorithm + weight, so modifying any signer property changes preimage and invalidates existing signatures.
* Embedded public keys are informational; verification still requires signature correspondence to digest preimage. Tampering with a public key changes no digest but causes signature verification failure.

### Verification Logic

1. Parse artifact; recompute canonical digest from non-signature, non-public-key fields.
2. Construct domain-separated preimage: `GAUTH_ROTATION_V2:<canonical_digest>`.
3. Verify each signature using appropriate algorithm path (Ed25519; ECDSA-P256; future BLS) with resolver/embedded public key.
4. Accumulate weights for signatures that verify; compare sum to `threshold_weight`.
5. Threshold enforcement performed server-side (artifact withheld if verified weight insufficient) and externally verifiable via auditor when public keys embedded.

### Attestation Multi-Sig (Future Extension)

Introduce optional `attestation_signers` with weights; threshold may differ from rotation threshold. Reuse same verification pattern. For initial rollout, rotation only.

### Backward Compatibility & Feature Flags

- If `version` absent or <2: fallback to legacy uniform model (count signatures; threshold interpreted as count).
- Provide dual endpoints for summary: existing `/rotation/summary` and new `/rotation/summary/v2`.
- Public key embedding is controlled by `GAUTH_ROTATIONS_V2_EMBED_PUBS=1`; when disabled, auditor signature verification requires an external key source (future enhancement) or is limited to digest/continuity.

### Configuration

Add `config/multisig_weights.json` or integrate into existing config loader:
```json
{
  "signers": [
    { "id": "hsm-a", "weight": 60, "alg": "ECDSA_P256" },
    { "id": "soft-b", "weight": 20, "alg": "ED25519" },
    { "id": "notary-c", "weight": 40, "alg": "ECDSA_P256" }
  ],
  "threshold_weight": 100
}
```

### Metrics & Observability

Expose:
- `gauth_rotation_v2_threshold_weight` gauge
- `gauth_rotation_v2_effective_weight` gauge (configured total)
- `gauth_rotation_v2_signer_weight{signer}` gauge
- `gauth_rotation_v2_verified_weight` gauge (verified weight for latest artifact)
- `gauth_rotation_v2_verified_weight_alg{alg}` gauge (algorithm breakdown)
- `gauth_rotation_v2_signature_failures_total{reason}` & `gauth_rotation_v2_signature_failures_by_alg_total{alg,reason}` counters
- `gauth_rotation_v2_threshold_violations_total` counter
- Planned: continuity break counter, embedding success metric
* Added continuity metrics:
  - `gauth_rotation_v2_chain_starts_total` (chain initialization events)
  - `gauth_rotation_v2_continuity_updates_total` (successful advancement of previous hash)
* Added embedding metrics:
  - `gauth_rotation_v2_public_keys_embedded_total` (artifacts built with embedding flag enabled)
  - `gauth_rotation_v2_embedded_public_key_count` (gauge of embedded keys in latest artifact)

### Security Considerations

* Weight file integrity (signed/anchored) prevents silent weight inflation.
* Duplicate signer ID rejection avoids double-counting weight.
* Chain continuity via `previous_artifact_hash` binds artifact sequence; auditor can externally verify continuity.
* Public key embedding trade-offs:
  - Pros: Enables offline verification; simplifies auditor; aids transparency.
  - Cons: Increases artifact size; exposes raw key bytes (acceptable for most public keys but consider stealth signer scenarios); requires operational discipline for key rotation (new key appears as changed `public` field while digest shifts due to signer ID/alg changes if any).
* Embedded public keys are not part of digest; tampering triggers signature verification failures, surfaced in auditor output & metrics.
* Algorithm agility preserved: additional algorithms can embed public keys using standardized encodings (e.g., base64url uncompressed EC point, future BLS G1/G2 serialization).

### Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Risk | Mitigation |
|------|------------|
| Weight config drift | Signed/anchored weight config; auditor comparison of artifact signers vs config. |
| Signature ordering manipulation | Deterministic sort by `id` before digest generation. |
| Algorithm mismatch exploitation | Validate `alg` against allowed set; include `alg` in digest preimage. |
| Threshold misconfiguration | Startup validation + auditor reporting of verified vs threshold weight. |
| Public key tampering | Signatures fail verification; failures logged & metered. |
| Artifact bloat | Embedding flag can be disabled; keys remain resolvable via registry. |
| Inconsistent key rotations | Auditor continuity + digest change surfaces unexpected key material shifts. |

### Alternatives Considered

- Fractional weights (e.g., float): Rejected; integer weights simpler, avoids encoding divergence.
- Excluding signatures from digest: Rejected; binding signature bytes prevents swap attacks.

## Implementation Steps

1. Config Loader: Parse weight config; validate uniqueness & positive integers.
2. Artifact Struct V2: Add fields + JSON tags; maintain legacy struct for V1.
3. Digest Builder: New function computing ordered preimage for V2.
4. Signing Workflow: Generate artifact without signatures; iterate signers to produce signature bytes; reassemble final artifact.
5. Verification Path: Accept both V1 & V2; compute effective weight; emit metrics.
6. Auditor Extension: Implement rotation weight mode (see Auditor Enhancements doc).
7. OpenAPI Update: Add `RotationSummaryV2` schema; document content negotiation.
8. Tests: Weight summation, threshold enforcement, duplicate ID rejection, downgrade fallback.

### Current Status (2025-10-28)

Implemented:
* OpenAPI schema for Rotation V2 (includes optional `public` signer field).
* Canonical digest builder excluding signatures/public keys.
* Ed25519 + ECDSA-P256 signature attachment & verification.
* Weighted verification metrics (threshold, effective, per-signer, verified, failures, per-alg breakdown).
* Auditor CLI modes for V2 remote/file digest + continuity + signature verification (Ed25519 via embedded public keys).
* Continuity chaining using `previous_artifact_hash` with server state tracking.
* Optional public key embedding via `GAUTH_ROTATIONS_V2_EMBED_PUBS`.
* Tests: config loading, mixed algorithm verification success/failure, continuity, embedded public key verification.

Pending / Planned:
* Additional algorithms (BLS12-381 aggregate, PQC) with encoding & verification paths.
* Auditor support for non-embedded key resolution (registry fetch / external key files).
* Continuity break metric & embedding success gauge.
* Formal artifact JSON canonicalization path (if migration from current digest strategy is desired).

## Acceptance Criteria

Core (Met):
* Rotation V2 artifact considered valid when verified weight >= threshold.
* Legacy rotation summaries unaffected.
* Metrics reflect configured and verified weights + failure taxonomy.
* Auditor outputs digest_match, continuity_ok, verified_weight, threshold_met, failure reasons.

Embedding (Met):
* Public key embedding does not alter canonical digest.
* Auditor performs offline Ed25519 verification using embedded keys without external resolver.
* Failures surface with clear reason taxonomy (public_missing, public_decode, signature_decode, signature_invalid, alg_unsupported).

## Open Questions

* Should attestation multi-sig share thresholds or have independent domain-specific thresholds? (Deferred)
* Governance requirement for minimum distinct algorithms besides aggregate weight? (Under consideration.)
* Artifact JSON canonicalization vs current plain-text digest preimage long-term direction? (Evaluation pending.)
* Compression or merkle-ization of signer set for very large signer counts? (Future scalability.)

## Decision

Weighted multi-signature rotation (V2) with optional public key embedding is Accepted. Future enhancements tracked in roadmap; embedding remains optional and can be disabled in environments preferring registry lookups.
