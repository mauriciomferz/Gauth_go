# RFC-0111 / RFC-0115 Compliance Gaps & Remediation Plan

Generated: 2025-10-26
Scope: Fact-based from current repository state (no external speculation).

## Classification Legend
| Status | Meaning |
|--------|---------|
| Implemented | Meets clause intent with code + tests + artifacts |
| Partial | Core mechanism present; notable sub-features or robustness missing |
| Missing | Not present or only placeholder comments |

## Gap Summary (High-Level)
- Strong foundations: rotation ledger integrity, multi-signature threshold, semantic anomaly throttle, basic revocation Merkle root, PoA structural model & validation.
- Partial domains: weighted multi-signature for PoA (structure present, enforcement incomplete), attestation domain separation consistency, external anchoring breadth, revocation proof API, comprehensive clause matrix for RFC-0115.
- Missing elements: algorithm agility (BLS/ECDSA for PoA), replay nonce for attestation artifact, persistent replay protection storage, PoA suspension / partial revocation flows, formal OpenAPI discovery surface, weight-based signature aggregation for rotation (currently simple count), risk register & formal external auditor tooling.

## Detailed Gaps Table
| Area | Clause Ref(s) | Current State | Gap | Impact | Priority | Suggested Remediation / Status |
|------|---------------|---------------|-----|--------|----------|-------------------------------|
| PoA Multi-Signature Weights | RFC-0111 §5.4 (Governance Signatures) | `Weights` & `SatisfiedWeight` fields exist; structural validation only | No aggregated weight calculation / threshold weight check | Governance trust can be overstated | P0 | Implement cumulative weight verification & digest binding tests |
| Attestation Domain Separation | RFC-0111 §6.2 (Attestation Integrity) | Prefix implemented (`GAUTH_MODEL_LIMIT_ATTEST:`) in signing & verification | (Remediated) Previously missing domain prefix | Prevents cross-context signature replay/confusion | DONE | Monitor rollout; add nonce + replay cache next |
| Revocation Proof Retrieval | Merkle recompute test exists; inclusion proof artifact example only | No endpoint to fetch proof for specific event hash | Demo of transparency less compelling | P1 | Add `/revocation/proof/:hash` returning siblings + verification status |
| Algorithm Agility (PoA & Rotation) | Ed25519 only | Lack alternate algorithms (ECDSA, BLS aggregate) | Limits interop & resilience | P1 | Abstract signature interface + plug BLS lib (already vendored for PoP) |
| Replay Protection Persistence | In-memory nonce/JTI store | No durable / bounded eviction strategy | Replay risk after restart | P1 | File-backed bloom filter or rolling WAL + compaction |
| Weighted Rotation Multi-Sig | Rotation threshold counts signatures only | Weight semantics absent (all equal) | Cannot express governance tiers | P2 | Add optional `weights` map, cumulative weight logic |
| External Anchoring Coverage | Capability + optional revocation root anchoring | Rotation summary & model limits not externally anchored | Reduced external audit confidence | P2 | Anchor rotation head & model limits snapshot hash periodically |
| PoA Lifecycle States | `active`, `revoked`, `expired` | No `suspended`, `partially_revoked` states | Reduced expressiveness for partial delegation | P2 | Extend enum + update validation & canonical digest |
| OpenAPI / Discovery | Placeholder directories | Missing published API spec / config discovery | Harder third-party integration | P2 | Generate minimal OpenAPI + `/well-known/gauth/config` |
| Observability Depth | Prometheus counters/gauges + anomaly scores | No latency histograms or signature verification metrics | Limited performance introspection | P2 | Add histograms (`signature_verify_latency_seconds`, `poa_issue_latency_seconds`) |
| Attestation Replay Nonce | RFC-0111 §6.3 (Replay Mitigation) | Timestamp only | No nonce/sequential counter | Replay indistinguishable from legitimate | P2 | Add `nonce` field & monotonic sequence cache |
| Risk Register | THREAT_MODEL doc present | Missing structured YAML/JSON register mapping threats → mitigations | Hard to present risk posture | P3 | Create `docs/RISK_REGISTER.yaml` referencing controls |
| Auditor Tooling | Internal tests & manual verification | No CLI for offline chain + Merkle + anchoring verification | External audit friction | P3 | Implement `cmd/auditor` with verify subcommands |

## Prioritized Remediation Roadmap (Next Sprint)
1. P0: PoA weight verification & attestation domain prefix.
2. P1: Revocation proof endpoint + signature algorithm abstraction + durable replay store.
3. P2: Rotation weight support, extended lifecycle states, latency histograms, external anchoring for rotation/model limits.
4. P3: Risk register formalization + auditor CLI.

## Security Impact Overview
| Remediation | Security Gain |
|-------------|---------------|
| Weight verification | Prevents threshold inflation attacks (claiming multi-signature without cumulative trust) |
| Domain prefix | Eliminates cross-context signature confusion / prefix collision |
| Durable replay store | Preserves replay protection across restarts |
| Proof endpoint | Enables third-party independent revocation inclusion verification |
| Algorithm agility | Future-proofs cryptography & enables aggregate signature compression |

## Testing Additions Needed
- New property tests: canonical digest invariance when adding weights, ordering stability.
- Negative tests: invalid cumulative weight (< threshold) should fail verification.
- Attestation prefix mismatch test (tamper raw payload prefix).
- Revocation proof round-trip (server siblings → client recompute root).
- Replay persistence integration test (restart & attempt original JTI).

## Demo Risk Mitigation
Prior to demo, ensure:
1. Threshold & multisig functioning with at least two ephemeral keys.
2. Semantic anomaly throttle configured (set realistic Z threshold, e.g. 2.5) + trigger scenario scripted.
3. Merkle recompute test passing (proof endpoint if implemented).
4. Rotation continuity negative test ready (inject gap) for observability narrative.

## Acceptance Criteria for “Clean Beta” Label
| Criterion | Must Have |
|-----------|-----------|
| Integrity Artifacts | Rotation summary (multi-sig), model limits attestation, revocation root |
| Transparency Proofs | Merkle recompute, continuity validation, proof endpoint (or documented artifact) |
| Governance | PoA issuance/validation + delegation + revocation chain |
| Reactive Controls | Semantic throttle activation & structured denial |
| Observability | Prometheus metrics for counters + anomaly scores + at least one latency measurement |
| Documentation | Compliance gaps, demo guide, release notes with multi-sig + threshold explained |

---
Maintainer: Update this file after each remediation iteration; keep historical snapshots in `artifacts/` if needed.
