# Release Notes – 2025-10-25

## Overview
This release focuses on strengthening multi-signature integrity, canonical serialization, authenticity defaults, and replay protection. It introduces embedded weights, an automatic domain V2 for threshold signatures, mandatory JTI enforcement, and a structured compliance matrix.

## Key Changes
| Area | Change | Benefit |
|------|--------|---------|
| Canonical Digest | Automatic domain V2 when `Threshold > 1` (includes `thr` + sorted weights) | Prevents digest confusion between single vs multi-sig contexts |
| Multi-Signature | Embedded `Weights` map + structural validation (positive, subset, cumulative >= threshold) | Eliminates env-driven ambiguity; ensures deterministic binding |
| Canonical JSON | Added `version` and serialized `weights` (when provided) | Future evolution support; integrity across signer sets |
| Authenticity | Strict authenticity now default (missing public key -> integrity failure) | Reduces silent authenticity downgrade risk |
| Replay Protection | JTI claim mandatory unless `GAUTH_ALLOW_MISSING_JTI=1` | Closes trivial replay channel for tokens without store |
| Tests | Updated property & domain tests; added version/weights presence test | Ensures determinism & correctness under new model |
| Docs | New `rfc0111_compliance_matrix.md`, API README compliance summary, CHANGELOG entry | Transparent compliance and roadmap communication |
| Algorithm Agility | Added ECDSA P-256 + BLS12-381 single signature support via registry | Enables phased adoption of alternative crypto primitives |

## Security Impact
- Stronger signature context binding (domain separation tied to threshold & weights).
- Reduced reliance on mutable environment for cryptographic semantics.
- Fail-closed replay behavior improves baseline token resilience.

## Migration Guidance
1. Legacy multi-sig PoAs (pre-embedded weights) should be re-issued to gain domain V2 differentiation.
2. If existing deployments depend on soft authenticity skip, set `GAUTH_STRICT_AUTHENTICITY=0` temporarily during transition.
3. Audit storage systems should verify digest divergence for threshold PoAs after upgrade.

## Backward Compatibility
- Single-signer (`Threshold=1`) PoAs keep V1 domain; digests unchanged.
- Existing tokens with missing JTI will now fail unless override is set.

## Testing Summary
- All updated tests pass (`canonical_prop`, `canonical_domain_v2`, `rotation`, `strict_auth`, `canonical_version_weights`).
- Property assertions confirm weight order invariance and digest variation on threshold/weight changes.

## Compliance Snapshot
See `docs/rfc0111_compliance_matrix.md` for full matrix. Highlights:
- Implemented: Multi-signature threshold, canonical serialization, validity period.
- Partial: Audit logging, replay protection, cryptographic requirements (algorithm agility pending).
- Missing: OpenAPI export, partial revocation, external anchoring.

## Roadmap (Excerpt)
1. Algorithm agility (Ed25519 + newly added ECDSA P-256 abstraction; BLS planned).
2. External audit ledger anchoring with signed entries/Merkle roots.
3. Partial revocation & suspension states; depth limits.
4. OpenAPI/Discovery contract + OTEL tracing integration.
5. Durable replay store (persistent JTI index with snapshot).

## References
- CHANGELOG: `docs/CHANGELOG.md`
- Compliance Matrix: `docs/rfc0111_compliance_matrix.md`
- Canonical Implementation: `pkg/rfc0111/canonical.go`
- Crypto Abstraction: `pkg/crypto/signature.go`, `pkg/crypto/ecdsa_provider.go`

## Algorithm Agility Usage
Configure service with a specific algorithm using functional options:

- Ed25519 (default): existing in-memory provider or external KMS.
- ECDSA P-256: `WithInMemoryAlgorithm("ecdsa-p256")`
- BLS12-381: construct provider manually (future convenience option planned) or use registry for verification.

Signatures record the algorithm name without altering canonical digest. Digest invariance maintains structural semantics across algorithms.

Planned: aggregated BLS signatures (multi-signer compression) and algorithm introspection endpoint.

## Checksums (Informational)
The canonical digest domain prefix variants:
- V1: `GAUTH_RFC0111_POA_V1` (single-sig)
- V2: `GAUTH_RFC0111_POA_V2|thr=<T>|w=<sorted weights>` (multi-sig)

## Acknowledgments
Thanks to contributors refining threshold signatures, authenticity defaults, and compliance mapping.

---
End of release notes.
