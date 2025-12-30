---
title: Demo Readiness
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Demo Readiness Checklist (Beta)

This checklist enumerates remaining polish items to ensure a clear, compelling live demonstration of AgentAuth’s protocol features (AAP-001) and semantic diagnostics (AAP-002).

## Core Demonstration Flow
1. Issue Delegation (PoA) with scoped restrictions.
2. Validate JWT / Replay protection (strict JTI mode) showing `token_replay_detected` on duplicate.
3. Multi-Signature issuance & verification (BLS aggregate) with participant count enforcement.
4. Model limits validation (trigger a limit_exceeded response tagged `AAP-001:model_limits`).
5. Capability + Rotation anchoring (external & registry) showing anchor receipts.
6. Revocation events appended → emit revocation Merkle root anchor.
7. Semantic diagnostics: induce semantic counter growth → show anomaly scores and hash chain evolution.

## Environment Preparation
- Set `AGENTAUTH_REPLAY_STRICT=1` for replay demo.
- Export `AGENTAUTH_CAPABILITIES_PATH` to a small curated capability registry file.
- Ensure anchor client memory persistence path writable for signed tree heads and capability audit chain tips.
- (Optional) Enable OTEL metrics: `AGENTAUTH_OTEL_METRICS_ENABLE=1`.

## Scripts / Automation (Recommended)
- Add a `scripts/demo_sequence.sh` to orchestrate a timed curl sequence for endpoints (future task).
- Provide a `Makefile` alias: `make demo` executing script above.

## Observability Enhancements
- Add one Prometheus scrape example for semantic counters (`/api/v1/beta/diagnostics/semantic` & metrics exposition).
- Include anomaly score gauge names in README (future addition).

## UX / Documentation Improvements
- Link diagrams in `docs/diagrams.md` directly from README with anchors.
- Add short “RFC Mapping” section in README linking to `docs/rfc_endpoint_mapping.md` and compliance matrix.
- Include sample JSON for a capability anchor and revocation root in `examples/`.

## Testing Gaps to Consider (Optional pre-demo)
- Attestation verification deep cryptographic path (beyond JSON shape) test.
- Rotation ledger invariant tests (sequence continuity).
- Negative semantic integrity test (force mismatch of prev_hash chain in a controlled scenario).

## Risk & Mitigation
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Lack of live semantic variation | Low audience engagement | Pre-generate varying counter snapshots via scripted requests before opening diagnostics page. |
| Replay strict mode unset | Replay demo fails silently | Assert `AGENTAUTH_REPLAY_STRICT=1` early; script check environment variable. |
| Capability registry hash unchanged | Anchor looks trivial | Introduce a small modification (add capability) mid-demo before re-anchoring. |
| Revocation chain empty | Anchor emission returns 404 | Pre-seed 2 revocation events early in sequence. |
| Anomaly scores all zero | Diminished AAP-002 showcase | Rapid bursts (≥3 snapshots) with increasing semantic counts to elevate z-score. |

## Immediate Actionable Items
1. Add demo sequence script (`scripts/demo_sequence.sh`).
2. Provide example artifacts (`examples/capability_anchor_example.json`, `examples/revocation_root_example.json`).
3. README additions: RFC mapping links, integrity chain explanation.
4. Optional: Add consistency proof endpoint (future) to highlight revocation auditability.

## Completion Criteria
- All referenced endpoints returning expected structured payloads.
- Compliance docs (`compliance_AAP-001_AAP-002.md`, `rfc_endpoint_mapping.md`) accessible and linked.
- Demo script runs end-to-end without manual intervention.
- Anomaly scores non-zero in at least one semantic counter during live session.
