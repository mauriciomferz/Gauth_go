# CHANGELOG

## 2025-10-25 (Post Multi-Sig Remediation)
### Security & Integrity Enhancements
- Canonical digest upgraded: automatic domain V2 when `Threshold > 1`, binding `thr` and sorted embedded weights to prevent replay/confusion attacks between single vs aggregated signature contexts.
- Embedded `Version` and deterministic `weights` object into canonical JSON (future evolution + integrity across weighted signer sets).
- Multi-signature weights now stored inside `PowerOfAttorney` (no env injection), eliminating configuration race and increasing auditability.
- Structural validation for weighted threshold signatures: positive weights, subset of signers, cumulative weight ≥ threshold.
- Strict authenticity enabled by default (missing signature public key -> integrity failure). Override only via `GAUTH_STRICT_AUTHENTICITY=0` for transitional migration.
- Mandatory `jti` claim enforced unless `GAUTH_ALLOW_MISSING_JTI=1` set, closing trivial replay gaps without a replay store.

### Test Suite Updates
- Refactored canonical property tests (`canonical_prop_test.go`) to use embedded weights instead of env variables.
- Added domain transition test (`canonical_domain_v2_test.go`) verifying digest changes without canonical JSON mutation.
- Introduced version/weights presence test (`canonical_version_weights_test.go`).
- Adjusted rotation & strict authenticity tests to reflect default strict mode (`rfc0111_rotation_test.go`, `rfc0111_strict_auth_test.go`).

### Documentation
- New compliance matrix: `docs/rfc0111_compliance_matrix.md` summarizing Implemented / Partial / Missing clauses & gap roadmap.
- API README updated with compliance summary and remediation highlights.

### Backward Compatibility Notes
- Single-sig PoAs (Threshold=1) retain V1 domain (digests unchanged).
- Multi-sig PoAs previously relying on `GAUTH_MULTI_SIG_WEIGHTS` will produce different digests when re-issued with embedded weights; re-issue recommended.
- Strict authenticity may surface new integrity failures for legacy delegations missing public key—use env override during phased rollout.

### Roadmap (Next Targets)
- Algorithm agility (add ECDSA/BLS PoA signature provider abstraction).
- External audit ledger anchoring + signed entries.
- Partial revocation & suspension states; delegation depth limits.
- OpenAPI/Discovery contract & OTEL tracing integration.
- Durable replay store with snapshot/compaction.

---
