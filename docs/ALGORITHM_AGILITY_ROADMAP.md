# Algorithm Agility Roadmap

Date: 2025-10-28
Status: Draft

## Motivation

Long-lived cryptographic systems must gracefully evolve algorithms (resisting advances in cryptanalysis, preparing for post-quantum, meeting ecosystem interoperability). Current implementation relies primarily on ED25519 & ECDSA (limited) and will introduce BLS for aggregation benefits. This roadmap outlines staged integration ensuring backward compatibility and verifiability.

## Target Algorithms

| Algorithm | Purpose | Properties | Initial Use |
|-----------|---------|------------|-------------|
| ED25519 | Existing signatures | Fast, deterministic | Tokens, attestations |
| ECDSA P-256 | Regulatory / ecosystem compliance | Widely supported, FIPS path | Rotation multi-sig (weighted) |
| BLS12-381 | Aggregation & succinct multi-sig | Enables compressed multi-sig | Future aggregated attestations |
| (Future) Post-Quantum Hybrid (Dilithium + classical) | PQ preparedness | Hybrid resilience | Transitional artifact signing |

## Phased Plan

### Phase 1: Capability Enumeration & Discovery
1. Extend discovery endpoint to list `supported_algorithms` (done partially) with metadata (stability tier: stable, beta, experimental).
2. Add policy manifest fields specifying required minimum algorithm diversity (e.g., `min_distinct_algorithms: 2`).

### Phase 2: Weighted Multi-Sig ECDSA Integration
1. Implement rotation V2 artifact (see ADR_WEIGHTED_MULTISIG) with ECDSA signer option.
2. Ensure deterministic encoding for ECDSA (DER -> canonical normalization) before digest.
3. Metrics: per-alg signature counts & latency.

### Phase 3: BLS Introduction (Experimental)
1. Add BLS key management module (`internal/crypto/bls.go` TBD).
2. Provide experimental attestation multi-sig aggregation path: combine N signer shares into single BLS signature.
3. Extend canonical attestation digest with `aggregation_version` when BLS used.
4. Auditor mode: verify BLS aggregate vs individual public keys set.
5. Fallback path: if aggregator failure, revert to individual signatures (graceful degradation).

### Phase 4: Dual / Hybrid Artifacts
1. For rotation & attestation, allow embedding both classical (ED25519/ECDSA) and aggregated BLS signature sets.
2. Clients verify at least one acceptable chain (policy-driven selection).
3. Introduce `alg_selection_policy` in discovery (e.g., prefer_aggregate, prefer_classical).

### Phase 5: PQ Readiness Scaffolding
1. Abstract signing interface to allow pluggable PQ implementations.
2. Add hybrid signature field structure `{ "hybrid": { "classical": <sig>, "pq": <pq_sig>, "alg": "dilithium2" } }`.
3. Provide negotiation endpoint to query PQ readiness levels.

## Canonical Digest Considerations

- Algorithm identifiers (`alg`) MUST be included for each signer in preimage ordering.
- For aggregated signatures, include participating signer IDs sorted before the aggregate signature bytes to prevent rogue key attacks.
- Hybrid signatures include both classical & PQ bytes; digest ordering: signer_id | alg_classical | alg_pq | classical_sig | pq_sig.

## Policy & Governance

- Minimum algorithm diversity enforced on rotation (effective weight must span >= configured distinct algorithms).
- Deprecation process: mark algorithm as `deprecated` in discovery; continue acceptance for 2 minor releases; auditor flags usage.
- Emergency algorithm sunset (e.g., severe vuln): flip `sunset_active` flag -> system rejects new artifacts unless they exclude deprecated algorithm; auditor emits critical alert.

## Observability

Metrics:
- `signature_latency_ms{alg="ed25519"}`
- `signature_latency_ms{alg="ecdsa_p256"}`
- `signature_latency_ms{alg="bls12381"}`
- `algorithm_diversity_effective` (count of distinct algorithms in current rotation).
- `aggregate_signature_size_bytes`.

Traces:
- `rotation.sign` span attribute `alg`, `weight`.
- `attestation.aggregate` span with participant count.

## Risk Analysis

| Risk | Impact | Mitigation |
|------|--------|------------|
| BLS implementation bug | Invalid aggregates compromise trust | Use vetted library, cross-verify with individual signatures tests |
| Algorithm monoculture persists | Reduced resilience | Enforce diversity via policy manifest & auditor checks |
| PQ premature adoption complexity | Operational burden | Stage PQ hybrid only after stability criteria met |
| Signature size inflation | Performance & latency | Measure metrics; use aggregation where beneficial |

## Acceptance Criteria

- Discovery lists at least ED25519 & ECDSA with stability tiers.
- Rotation V2 artifact supports ECDSA weight semantics.
- Auditor detects monoculture and reports diversification score.
- BLS experimental path verifies aggregated signature with fallback.
- Documentation updated for each phase.

## Open Questions

- Minimum distinct algorithms for compliance? (Candidate: 2)
- PQ timeline dependency on external library maturity.

## Next Steps

1. Implement rotation V2 (weights + alg metadata).
2. Extend discovery & policy manifest fields.
3. Prototype BLS key management & aggregation path behind feature flag.
