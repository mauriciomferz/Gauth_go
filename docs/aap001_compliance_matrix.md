---
title: Rfc0111 Compliance Matrix
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AAP-001 / 0115 Compliance Matrix (Post Remediation: Version & Embedded Weights, Strict Authenticity, Mandatory JTI)

Generated: 2025-10-25
Status Categories: Implemented, Partial, Missing
Legend:
- Implemented: Clause requirements met with tests & evidence.
- Partial: Core behavior present; gaps remain (listed).
- Missing: Not yet implemented; roadmap required.

## Summary
- Clauses mapped (current harness): 8 (from `conformance/report.json` summary)
- Symbols resolved: 24 / 24
- Test coverage for mapped clauses: 100%
- Gap Items Total: 43 (Implemented: 6, Partial: 17, Missing: 20)
- Recent Remediation Impact:
  - Canonical digest now binds multi‑signature threshold & weights via automatic V2 domain (no env indirection).
  - Embedded `Version` field and deterministic `weights` object included in canonical JSON (future evolution ready).
  - Strict authenticity default (missing public key is integrity failure unless explicitly disabled).
  - Mandatory `jti` claim (fail closed unless override flag `AGENTAUTH_ALLOW_MISSING_JTI=1`).
  - Weighted multi‑signature structural validation (positive ints, subset of signers, cumulative weight >= threshold).

## Clause Coverage (Mapped Set)
| Clause | Title | Status | Notes / Gaps | Evidence |
|--------|-------|--------|--------------|----------|
| 0111:2 | Policy Bundle Integrity | Partial | Bundle integrity & chain verification present; needs stronger cryptographic transparency (signed bundle manifests). | `AddBundle`, `VerifyChain`; `pkg/policy/engine.go` |
| 0111:3 | Delegation & Revocation | Partial | Creation, validation, revocation chains implemented; lacks depth limits & suspension semantics. | `CreateDelegation`, `RevokeDelegation`, `RevocationChain` |
| 0111:4 | Audit Logging | Partial | Memory/File loggers + hash chain; external notarization optional; lacks tamper‑evident signature on each entry. | `AuditEvents`, `FileLogger`, `VerifyChain` |
| 0111:5 | Replay Protection | Partial | JTI enforced & replay store option; needs durable persistence + TTL eviction strategy. | `WithReplayProtection`, `VerifyToken` |
| 0111:6 | Cryptographic Requirements | Partial | Canonical digest + multi‑sig domain separation done; still single Ed25519 algorithm & no detached verification for tokens. | `CanonicalPOADigest`, `verifyPOASignature` |
| 0111:10 | Detached Signatures | Partial | Detached PoA signature path present; needs standard envelope negotiation + third‑party verifiers doc. | `aap001_detached_signature_test.go` |
| 0111:11 | Multi‑Signature Threshold | Implemented | Threshold + embedded weights; deterministic canonicalization & domain V2 separation; property tests updated. | `verifyMultiSignatures`, `ValidateMultiSignature`, `canonical_prop_test.go` |
| 0115:1 | PoA Structure | Partial | Structure includes Version & Weights; advanced joint conditions & conditional clauses missing. | `PowerOfAttorney`, `ValidateMultiSignature` |
| 0115:3 | Validity Period | Implemented | UTC normalized RFC3339, canonical digest excludes mutable fields. | `CanonicalPOADigest`, `validateDelegationRequest` |
| 0115:8 | Joint Signatures | Implemented (threshold/weighted) | Joint threshold semantics enforced; still lacks aggregated signature compression optimization. | `verifyMultiSignatures` |
| 0115:9 | Canonical Serialization | Implemented | Deterministic JSON (sorted scope, restrictions, weights) + domain separation. | `canonical.go`, property tests |
| 0115:10 | Revocation Semantics | Partial | Basic revocation chain; advanced partial revocation & conditional revocation absent. | `delegation/revocation_chain.go` |

(Other unmapped clauses in AAP-001/0115 exist beyond current harness scope; see Gap Matrix.)

## Remediation Delta
| Feature | Previous State | Current State | Security Gain |
|---------|----------------|---------------|---------------|
| Multi‑sig weights binding | External env var injection (`AGENTAUTH_MULTI_SIG_WEIGHTS`) | Embedded in PoA & canonical digest domain | Removes config race; prevents threshold confusion & replay of single-sig digest |
| Domain separation | Manual env toggle V2 | Automatic when `Threshold > 1` | Eliminates misconfiguration risk, ensures semantic binding |
| Canonical JSON | No version / weights | Includes `version`, `weights` (sorted) | Future-proof evolution; integrity across weighted signers |
| Authenticity default | Soft skip missing public key | Integrity failure unless `AGENTAUTH_STRICT_AUTHENTICITY=0` | Reduces silent downgrade of signature assurance |
| Replay claim enforcement | JTI optional outside replay store | JTI mandatory unless explicit override | Eliminates trivial replay vectors |

## Gap Matrix Highlights (Condensed)
| Category | Key Missing / Partial Items | Priority |
|----------|-----------------------------|----------|
| Crypto & Authenticity | Algorithm agility (ECDSA/BLS for PoA), external detached token signature | P0 |
| Authorization Engine | Obligations/advice execution, distributed PDP caching | P1/P2 |
| PoA Lifecycle | Suspension, partial revocation, chaining depth limits | P2 |
| Persistence | External anchoring notarization, index/pruning strategies | P2 |
| Observability | Tracing, granular semantic rejection metrics, OTEL exporter | P2/P3 |
| Secret Management | Vault/HSM integration, tenant isolation | P0/P1 |
| Interoperability | OpenAPI/Discovery spec completion | P1 |
| Testing | Load/stress harness, expanded property tests (validators) | P2 |
| Risk Modeling | Mitigations matrix, residual risk register | P2/P3 |

## New Verification Artifacts
- Updated property tests: `canonical_prop_test.go` (weight order invariance, threshold domain separation)
- Rotation & strict authenticity tests adjusted to reflect default strict mode: `aap001_rotation_test.go`, `aap001_strict_auth_test.go`
- Automatic domain V2 test: `canonical_domain_v2_test.go`

## Roadmap (Next Steps)
1. Algorithm Agility: Introduce interface for PoA signature algorithms (Ed25519, ECDSA-P256, BLS aggregate). (P0)
2. External Anchoring: Signed ledger entries + optional Merkle root publication. (P1)
3. Partial Revocation & Suspension States: Extend `POAStatus` with `suspended`, `partially_revoked`. (P2)
4. OpenAPI & Discovery: Emit `/well-known/agentauth/config` with canonical digest algorithm list & JTI constraints. (P1)
5. Tracing & Metrics: OTEL spans around create/validate; Prometheus collector registration for violation counters. (P2)
6. Replay Durability: Persistent JTI bloom filter or append-only WAL + snapshot/compaction. (P2)
7. Validator Property Tests: Add semantic validator corpus; fuzz special conditions once interpreter added. (P2)
8. Risk Register: YAML/JSON doc linking threat IDs -> mitigations -> residual risk. (P3)

## Verification Checklist (Post-Change)
- [x] Digest difference when threshold elevated (domain V2) without JSON mutation.
- [x] Insertion order of weights invariant.
- [x] Weight/threshold modifications produce digest change.
- [x] Missing public key raises integrity failure under default strict mode.
- [x] JTI required unless explicit override.
- [x] Structural weight validation (subset + positive + cumulative >= threshold).

## Compatibility & Migration Notes
- Existing single-sig PoAs (Threshold=1) continue using V1 domain (digest unchanged).
- Multi-sig PoAs issued before remediation that relied solely on env weight mapping will produce different digest if re-issued with embedded weights; consider transitional re-issuance.
- Strict authenticity may cause new integrity failures for previously accepted delegations missing public key entries—mitigate by setting `AGENTAUTH_STRICT_AUTHENTICITY=0` temporarily during migration.

## Evidence References
- `conformance/clause_map.json`
- `conformance/report.json`
- `pkg/aap001/canonical.go`
- Tests: `pkg/aap001/*_test.go` (domain, rotation, strict auth, canonical properties)

---
Maintainer Action: Review roadmap items, prioritize P0/P1 for next sprint; attach this matrix to release notes.
