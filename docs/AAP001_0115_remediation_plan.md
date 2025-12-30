---
title: Rfc0111 0115 Remediation Plan
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AAP-001 & AAP-002 Remediation Plan

Date: 2025-10-20
Scope: Close high-priority (P0/P1) gaps identified in GAP_MATRIX for Authorization (AAP-001) and Power of Attorney (AAP-002).

Update (2025-10-21): Envelope V2 migration instrumentation (adoption ratio gauge `envelope_v2_adoption_ratio` + digest mismatch counter `envelope_digest_mismatch_total`) live; sunset lifecycle defined in `ADR-envelope-v1-sunset.md`. Backlog adjusted to include sunset execution epic and cadence metrics.

## Prioritized Backlog (Sequenced)

| Order | Epic | RFC Coverage | Priority | Outcome / Acceptance Criteria |
|-------|------|--------------|----------|-------------------------------|
| 1 | External Notarization & Timestamping | 0111 Audit Logging, 0115 Revocation Semantics | P0 | Capability & audit chain tips externally notarized; receipts retrievable & verifiable; latency/failure/age metrics present; GAP_MATRIX external anchoring status -> Implemented. |
| 2 | Multi / Joint Signature Threshold | 0115 Joint Signatures | P1 | PoA supports required_signers + threshold & optional weights; granular verification metrics & latency histogram; domain separation v2 flag; satisfaction metadata exposed; tests cover success & failure taxonomy. |
| 3 | Key Rotation Scheduler + Secure Storage | 0111 Cryptographic Requirements | P1 | Automatic key rotation with Vault/KMS backend; discovery advertises schedule; rotation log chain externally anchored; metrics reflect active key age & rotation counts. |
| 4 | Obligations & Advice Execution Layer | 0111 Policy Bundle Integrity / Combining Algorithms | P1 | Engine executes obligations post-decision; advice returned to caller; tests simulate allow/deny with side-effect obligations; GAP entry moves from Missing -> Partial/Implemented. |
| 5 | Full PoA Embedding & Canonical Expansion | 0115 Power-of-Attorney Structure | P1 | Token envelope embeds full canonical PoA serialization (JSON/CBOR) incl. limits/conditions; canonical digest stable; backward compatibility preserved; fuzz/property tests for serialization stability. |
| 6 | Policy Versioning & Rollback Metadata | 0111 Delegation & Revocation / Policy Integrity | P1 | Policies carry version ids; evaluator stores historical versions; rollback endpoint; audit log includes version transitions. |
| 7 | Public Verifiable Token Mode (Detached Sig / Public PASETO) | 0111 Cryptographic Requirements | P1 | Tokens carry detached signature (Ed25519) or PASETO public mode; verification endpoint updated; metrics expanded; digest mismatch counter validated across modes. |
| 8 | Envelope V1 Sunset Execution | 0111/0115 Governance & Integrity | P1 | Adoption ratio >=0.95 sustained 7d & mismatch ratio <0.005; V1 issuance disabled (verification retained); issuance cadence histogram + phase controller implemented; CHANGELOG & GAP_MATRIX updated; archival snapshot stored. |
| 9 | Conditional / Special Conditions Interpreter | 0115 Special Conditions | P2 | Interpreter evaluates conditions at validation time; failure surfaces reason metrics; property tests around expression classification. |
| 10 | Replay Persistence WAL & Recovery | 0111 Replay Protection | P2 | Write-ahead log for JTI store; recovery harness; tests simulate crash & replay; GAP moves Missing -> Partial. |
| 11 | OpenAPI Expansion & Discovery Enrichment | 0111 & 0115 Interop | P1 | Spec includes multi-sig, notarization, rotation endpoints; discovery doc adds deprecation & sunset schedule, rotation & notarization metadata, adoption ratio exposure. |
| 12 | Secure Secret Storage (Vault/HSM) Hardening | 0111 Cryptographic Requirements | P0 | Active keys & secrets stored only in secure backend; fallback file mode flagged; GAP moves Missing -> Partial/Implemented. |
| 13 | Fuzz & Property Test Expansion | 0111/0115 Robustness | P2 | Fuzz tests for loader, validator, signature verify, anchor artifact, envelope parser; property tests for canonical stability across multi-sig, embedded PoA & sunset removal commit. |
| 14 | Distributed PDP & Caching | 0111 Scalability | P2 | Cache layer with invalidation; cluster metadata; latency metrics vs baseline. |
| 15 | Conditional Numeric Multi-Period Limits Persistence | 0115 Power Limits | P2 | Weekly/monthly limit tracking in persistent ledger; exceedance metrics. |

## Cross-Cutting Implementation Guidelines
- Use feature flags for incremental rollout (`GAUTH_NOTARY_PROVIDER`, `GAUTH_MULTISIG_ENABLED`, `GAUTH_KEY_ROTATION_ENABLED`).
- Each epic produces an ADR (3 already drafted: notarization, multi-signature, key rotation).
- Update GAP_MATRIX after merging epic branch; add evidence references.
- Ensure metrics integration early (avoid retrofitting later).
- Provide migration notes & backward compatibility shims (e.g., threshold default=1). 

## Metrics Additions (Planned / Adjusted)
- `gauth_notarization_failures_total{kind,provider}`
- `gauth_multisig_completion_ratio` (gauge)
- `gauth_key_rotation_total` / `gauth_key_rotation_failures_total`
- `gauth_token_public_verify_failures_total`
- `gauth_obligation_execution_failures_total`
- `gauth_poacondition_evaluation_latency_seconds` (histogram)
- `gauth_replay_wal_recovery_total`
- `gauth_envelope_issuance_cadence_seconds` (histogram) – interval between consecutive issuance events
- `gauth_envelope_v1_sunset_phase` (gauge enum) – 0 Pilot,1 Broad,2 Stabilization,3 SoftDep,4 Sunset,5 PostVerify
- Future: labeled digest mismatch counter `gauth_envelope_digest_mismatch_total{reason}` (canonicalization_error|tamper_suspected|domain_conflict)

## Testing Strategy Summary
- Unit: Each interface (NotaryProvider, RotationManager, MultiSignatureVerifier)
- Integration: End-to-end notarization receipt retrieval & verification
- Property: Canonical digest ignoring volatile fields
- Fuzz: PoA loader & capability registry transaction parser
- Load: Threshold signature collection under concurrent signers

## Risk Mitigations
- External dependency outages: circuit breaker + deferred retry queue for notarization.
- Key rotation failure: atomic rollback; alert on repeated failures.
- Multi-sig partial state: status endpoint shows remaining signers; TTL to force re-issuance if stalled.

## Success Measurement (Extended)
- Reduction of P0/P1 Missing items by >70% within two milestone cycles.
- External verifiability demonstrated via automated verification tests.
- Mean notarization latency within defined SLO (e.g., p95 < 2s) after integration.
- Envelope adoption ratio >=0.95 & mismatch ratio <0.005 sustained pre-sunset.
- Zero unexpected V1 issuance events 24h post sunset execution.
- Stable issuance cadence (no >20% variance spike) during stabilization phase.

## Milestone Cut Suggestions (Updated)
Milestone A (Integrity Core): Epics 1–3, 12.
Milestone B (Semantic, Governance & Sunset): Epics 4–8, 11, 15.
Milestone C (Scalability & Robustness): Epics 5 (embedding stabilization), 9–10, 13–14.

## Follow-Up Tracking
Create GitHub issues per epic with labels: `rfc0111`, `rfc0115`, `integrity`, `sunset`, `priority:P0/P1`.
Dashboard panels: adoption ratio progression, mismatch ratio trend, issuance cadence histogram, sunset phase gauge.
Link issues to ADR references (`ADR-envelope-v1-sunset.md`).
