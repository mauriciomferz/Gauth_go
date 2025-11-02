# Beta Remediation Plan (RFC111 / RFC115)

Date: 2025-10-26
Goal: Achieve demonstrable RFC alignment for public beta (security, determinism, observability, multi-sig readiness).

## Prioritization Buckets
- P1 (Critical demo blockers) – Must land before beta tag.
- P2 (Strong compliance / credibility) – Land if feasible; otherwise scheduled early post-beta patch.
- P3 (Documentation & polish) – Improves adoption & clarity; can trail by days.

## P1 Items
| ID | Task | Description | Acceptance Criteria | Owner | Status |
|----|------|-------------|---------------------|-------|--------|
| P1-1 | Full error taxonomy coverage | Replace ad-hoc gin.H errors in delegation, token, audit endpoints with `respondError` + RFC refs | All non-2xx responses carry `success=false, code, error, rfc_ref` | TBD | Pending |
| P1-2 | Scope pattern narrowing | Implement prefix (`payment:*`), wildcard `*`, regex opt-in (`regex:` prefix), numeric range (`amount:[min,max]`) | Validation rejects out-of-pattern actions; tests cover each mode | TBD | Pending |
| P1-3 | External revocation anchor emission | Add endpoint to emit revocation chain tip anchor & verify chain integrity | `/api/v1/anchor/revocation/emit` returns receipt; chain verifies | TBD | Pending |
| P1-4 | Semantic counters diagnostics | Add `/api/v1/diagnostics/semantic` returning counters snapshot | Endpoint returns counters, tests compare increments | TBD | Pending |
| P1-5 | Multi-sig POA issuance endpoint | Endpoint to collect individual signer signatures, produce aggregated metadata | `/api/v1/poa/multi/issue` returns canonical digest + signer checklist | TBD | Pending |
| P1-6 | Rights & obligations serialization | Extend POA schema with `obligations`, `advice` arrays and record evaluation in validation | Canonical digest includes arrays (empty omitted) + basic enforcement stub | TBD | Pending |
| P1-7 | OpenAPI spec upgrades (skew & integrity codes) | Document `GAUTH_CLOCK_SKEW_SECONDS`, multi-sig integrity failure codes, mismatch reasons | Spec includes parameter + enumerated error codes | TBD | Pending |

## P2 Items
| ID | Task | Description | Acceptance Criteria | Status |
|----|------|-------------|---------------------|--------|
| P2-1 | Replay store error counter | Increment metric on distributed store failures, expose total | Counter visible in Prometheus metrics | Pending |
| P2-2 | Weighted multi-sig metrics detail | Distinct metrics for `satisfied_weight` vs threshold | Prometheus gauge/histogram present | Pending |
| P2-3 | Anchor observer test coverage | Unit test exercising external anchor callback hash chain | Test verifies callback invoked with expected tip hash | Pending |
| P2-4 | Mismatch reason diagnostics endpoint | `/api/v1/diagnostics/digest_mismatch` aggregated counts | Endpoint returns per-reason counts | Pending |
| P2-5 | Canonical digest fuzz harness | Lightweight fuzz test (Go 1.20+ `testing.F`) | Fuzz passes stable digest invariants | Pending |

## P3 Items
| ID | Task | Description | Acceptance Criteria | Status |
|----|------|-------------|---------------------|--------|
| P3-1 | Sequence diagrams | Mermaid diagrams for PoP, multi-sig, delegation validation | Diagrams in `docs/diagrams/*.md` | Pending |
| P3-2 | Architecture diagram | High-level component & data flow (audit, anchor, revocation, replay) | PNG/SVG + source mermaid committed | Pending |
| P3-3 | Metrics reference page | Table of metrics names, types, semantic meaning | `docs/metrics_reference.md` present | Pending |
| P3-4 | Demo script | Step-by-step multi-sig + PoP + revocation anchor demonstration | Script validated end-to-end locally | Pending |

## Implementation Sequencing
1. P1-1 taxonomy (touches many handlers) – minimizes rebase friction later.
2. P1-2 scope narrowing (introduces parser + tests) – dependent on standardized errors for clean failure modes.
3. P1-3 revocation anchor endpoint – builds on existing combined anchor code structure.
4. P1-4 semantic counters endpoint – low risk, purely read-only.
5. P1-5 multi-sig issuance – extends POA creation pipeline; integrate canonical digest output.
6. P1-6 rights & obligations – schema + digest changes; coordinate with canonical tests update.
7. P1-7 OpenAPI augment (after new endpoints to avoid churn).

## Risk & Mitigation
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Canonical digest change after adding obligations | Signature invalidation | Bump POA version and migrate tests; keep obligations optional for existing POAs. |
| Multi-sig issuance races (concurrent signer submissions) | Inconsistent aggregated state | Use per-POA in-memory signer collection with mutex; optional TTL. |
| Regex scope injection / performance | ReDoS potential | Precompile with timeout; limit pattern length; reject catastrophic patterns. |
| Anchor endpoint misuse (excessive calls) | Resource exhaustion | Rate-limit via simple counter & sliding window; metrics. |

## Acceptance Gate (Beta Ready)
All P1 items implemented + passing tests + OpenAPI updated + sequence diagrams present (at least PoP + multi-sig). Metrics counters exposed and no critical failing tests.

## Tracking
Use labels `rfc111`, `rfc115`, `beta` in issue tracker. Each P1 item gets an issue referencing this plan.

---
Will refine as features land. Auto-generated baseline.
