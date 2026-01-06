---
title: ADR Envelope V1 Deprecation & Sunset Plan
category: adr
status: draft
lastUpdated: 2025-11-12
owners: architecture-team
source: internal
refreshCadence: on-change
---
# ADR: Envelope V1 Deprecation & Sunset Plan

Date: 2025-10-20
Status: Draft
Authors: Automated Assistant
Context: Introduction of Envelope V2 (`ver="agentauth-aap001-env2"`) adds canonical digest and multi-signature satisfaction metadata. A structured lifecycle is required to retire Envelope V1 safely while preserving verification compatibility and minimizing operational risk.

Summary: Outlines metrics-driven multi-phase migration from Envelope V1 to V2, operational playbooks, rollback, and communication matrix. Implementation in beta-refactor branch.

References: See GAP_MATRIX Section 12, implementation in `internal/envelope/manager.go`, tests in `web/envelope_migration_test.go`.

## 1. Problem Statement
Running two envelope formats indefinitely increases maintenance overhead, complicates observability interpretation, and dilutes security posture (legacy format lacks embedded canonical digest). We need a formal deprecation cadence tied to measurable adoption and integrity signals.

## 2. Goals
- Predictable multi-phase transition (no sudden breaking change).
- Metrics-driven decision gates (objective, reproducible).
- Rapid rollback path if regression or consumer incompatibility emerges.
- Communication artifacts for internal teams and integrators.
- Integrity assurance: mismatch anomalies below threshold pre-sunset.

## 3. Non-Goals
- Changing canonical digest algorithm or domain separation rules (handled by separate flag `AGENTAUTH_MULTI_SIG_DOMAIN_V2`).
- Introducing labels to adoption ratio gauge (keep cardinality low).

## 4. Terminology
- Adoption Ratio: Gauge `agentauth_aap001_envelope_v2_adoption_ratio` (0-1).
- Mismatch Counter: `agentauth_aap001_envelope_digest_mismatch_total`.
- Issuance Counters: `envelope_v1_issued_total`, `envelope_v2_issued_total`.

## 5. Lifecycle Phases & Gates
| Phase | Target Window | Entry Criteria | Exit Criteria | Actions |
|-------|---------------|----------------|---------------|---------|
| Pilot | 1-3 days | Adoption Ratio >=0.05 any 30m window | Adoption Ratio >=0.20 sustained 6h | Canary subset only; monitor mismatch spikes |
| Broad Adoption | 2-7 days | Adoption Ratio >=0.20 sustained 6h | Adoption Ratio >=0.70 sustained 12h; mismatch ratio <0.01 | Expand to majority replicas; publish internal readiness email |
| Stabilization | 3-10 days | Adoption Ratio >=0.70 sustained 12h | Adoption Ratio >=0.90 sustained 24h; mismatch ratio <0.005 | Freeze feature changes affecting canonicalization; prep deprecation notice |
| Soft Deprecation | 7-14 days | Adoption Ratio >=0.90 sustained 24h | Adoption Ratio >=0.95 sustained 7d; mismatch ratio <0.005; no unresolved high-sev issues | Announce V1 issuance disable date; update GAP_MATRIX status to "Deprecated" |
| Sunset | Scheduled (post soft deprecation) | Adoption Ratio >=0.95 sustained 7d; mismatch ratio <0.005 | Completion: V1 issuance disabled | Remove V1 issuance path (verification acceptance retained) |
| Post-Sunset Verification Support | 1-2 release cycles | V1 issuance disabled | Removal of V1 verification after cycles OR earlier if <2% verifications | Announce final verification removal plan |

Mismatch Ratio Formula:
```
mismatch_ratio_5m = increase(agentauth_aap001_envelope_digest_mismatch_total[5m]) / (increase(agentauth_aap001_envelope_v1_issued_total[5m]) + increase(agentauth_aap001_envelope_v2_issued_total[5m]))
```

## 6. Metrics Acceptance Criteria
Before disabling V1 issuance:
- Adoption Ratio >=0.95 (avg_over_time over 7d) AND min_over_time over 7d >=0.90.
- Mismatch Ratio <0.005 across all consecutive 1h windows in last 48h.
- No critical alerts for `EnvelopeDigestMismatchSpike` in last 72h.

## 7. Operational Playbooks
### Rollout
1. Enable `AGENTAUTH_POA_ENVELOPE_V2=1` on canary pods only.
2. Validate dashboards: adoption ratio updates; mismatch counter remains stable.
3. Incrementally update remaining pods (batch size 10-20%).
4. Record progression timestamps in change log; tag release after Broad Adoption gate.

### Rollback
If Adoption Ratio drops >15 percentage points within 30m OR mismatch ratio exceeds 0.01:
- Immediately halt further V2 flag enablement.
- Re-enable V1 issuance on last updated batch (set env back to 0) while retaining V2 verification.
- Capture diagnostic bundle: recent deployment diff, canonicalization code commits, sample mismatched tokens.
- Open incident ticket linking PromQL evidence.

### Sunset Execution
1. Merge PR removing V1 issuance conditional branch in `generateAuthToken` (keep parser fallback for verification).
2. Tag release and deploy gradually (10% -> 50% -> 100%).
3. Confirm `envelope_v1_issued_total` remains static post deployment across all pods.
4. Archive final adoption & mismatch metrics snapshot in `docs/operational/adoption_snapshots/`.

### Post-Sunset Verification Retirement
Criteria:
- share_of_v1_verifications = increase(v1_verification_count[24h]) / increase(total_verification_count[24h]) <0.02 for 30 consecutive days.
- Publish removal notice; schedule removal in next minor version.

## 8. Communication Matrix
| Audience | Trigger | Channel | Artifact |
|----------|---------|---------|----------|
| Internal Eng | Phase transitions | Slack #agentauth-migration | Phase summary w/ metrics screenshot |
| SRE | Rollback initiation | PagerDuty | Incident runbook + PromQL queries |
| Integrators | Soft Deprecation start | Email + Portal Announcement | Deprecation notice (FAQ) |
| Security | Mismatch spike | Security incident workflow | Digest diff + token samples (sanitized) |
| Compliance | Sunset completion | Quarterly report | Metrics snapshot + GAP_MATRIX update |

## 9. Risks & Mitigations
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Flag misconfiguration across replicas | Distorted adoption ratio | Automated config audit + deployment validation step |
| Undetected canonicalization regression | Elevated mismatch late | Mandatory code review checklist + mismatch alert gating |
| Premature sunset before client readiness | Client breakage | Require integrator confirmation receipt prior soft deprecation exit |
| Alert fatigue from minor mismatch blips | Slow incident response | Use rate-based AND absolute count threshold in alert rules |

## 10. Alternatives Considered
- Dual issuance (emit both V1 & V2 tokens): Rejected (security & complexity overhead).
- Hard cutover without ratio gating: Rejected (consumer risk).
- Labeling issuance counters by service ID: Deferred (cardinality concerns first).

## 11. Future Enhancements
- Automatic phased rollout controller reading adoption gauge and adjusting pod flag distribution.
- Digest mismatch reason labeling + structured anomaly endpoint.
- Recording rule for 7d moving adoption average with min/max.
- Signed attestation artifact for sunset completion.

## 12. Implementation Checklist
- [ ] Dashboard panels (ratio, v1/v2 rates, mismatch rate).
- [ ] Alert rules committed to infra repo.
- [ ] Runbook page created / updated.
- [ ] GAP_MATRIX.md status row updated to Deprecated when entering Soft Deprecation.
- [ ] Change log entry referencing this ADR at sunset completion.

## 13. Open Questions
- Should we require external notarization witness attestation before final sunset? (Pending security review.)
- Do we compress historical adoption data after archiving? (Pending observability retention policy.)

### Appendix: PromQL Snippets
```
# 7d adoption average & min
adoption_avg_7d = avg_over_time(agentauth_aap001_envelope_v2_adoption_ratio[7d])
adoption_min_7d = min_over_time(agentauth_aap001_envelope_v2_adoption_ratio[7d])

# Mismatch ratio 1h window
mismatch_ratio_1h = increase(agentauth_aap001_envelope_digest_mismatch_total[1h]) / (increase(agentauth_aap001_envelope_v1_issued_total[1h]) + increase(agentauth_aap001_envelope_v2_issued_total[1h]))

# V1 issuance stalled post sunset (should remain flat)
rate(agentauth_aap001_envelope_v1_issued_total[30m]) == 0
```

---
