---
title: Auditor Enhancements
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Auditor Enhancements (Beta Roadmap)

This document defines planned extensions to the auditor tooling to strengthen assurance for AAP-001 / AAP-002 artifacts and forthcoming multi-sig & algorithm agility features.

## Objectives

1. Provide end-to-end reproducible verification of revocation transparency (inclusion & consistency proofs, chain growth semantics).
2. Validate capability registry anchoring and external notarization freshness.
3. Assess replay protection durability (WAL + external backend) with spot integrity checks.
4. Verify domain signature correctness across attestation types and report drift.
5. Support weighted multi-sig rotation verification when feature lands.
6. Enable algorithm agility readiness audits (enumerate supported algorithms, detect monoculture risk).

## New Auditor Modes

| Mode | Command Flag | Description | Inputs | Outputs |
|------|--------------|-------------|--------|---------|
| Revocation Inclusion | `--revocation-inclusion` | Fetch event by id/hash and verify Merkle path vs head root. | event id/hash | PASS/FAIL + path length + root digest |
| Revocation Consistency | `--revocation-consistency` | Verify logarithmic consistency proof between two sizes. | older size, newer size | PASS/FAIL + proof segments count |
| Capabilities Anchor | `--capability-anchor` | Fetch latest anchor artifact and recompute registry hash locally. | registry source path | PASS/FAIL + mismatch diff |
| Replay WAL Integrity | `--replay-wal-integrity` | Reconstruct in-memory index from WAL; compare counts & ordering; sample TTL expirations. | WAL path, sample count | PASS/FAIL + discrepancies list |
| Domain Signature | `--domain-signature` | Recompute canonical attestation digest and verify domain signature set. | attestation payload | PASS/FAIL + missing/extra signers |
| Rotation Summary | `--rotation-summary` | Fetch rotation summary; verify threshold vs signatures weight (post weighted multi-sig). | summary endpoint | PASS/FAIL + effective weight |
| Algorithm Agility | `--alg-agility` | Enumerate supported algorithms vs configured policy; flag monoculture. | discovery endpoint | PASS/FAIL + diversification score |

## Data Flow Diagrams (High-Level)

### Revocation Inclusion / Consistency
```
auditor -> GET /revocation/head -> head_root
auditor -> GET /revocation/proof?id=... -> proof[]
auditor: rebuild path with sibling hashes -> compare root == head_root -> emit result
```

### Replay WAL Integrity
```
auditor: open WAL file
auditor: sequentially parse entries -> build index
auditor: fetch live adapter stats (optional endpoint TBD)
auditor: compare counts, latest timestamp, sample TTL expirations
auditor: emit discrepancies
```

## Metrics & Reporting

Each mode emits structured JSON:
```json
{
  "mode": "revocation-inclusion",
  "success": true,
  "details": {
    "event_hash": "...",
    "path_length": 12,
    "head_root": "..."
  }
}
```

Aggregate summary (multi-run) will compute:
- success_ratio
- average_latency_ms
- anomaly_counts by category

## Implementation Plan

1. CLI Flag Scaffolding: Extend existing auditor command (in `cmd/auditor/`) with new flags and dispatch.
2. Shared Utilities: Create internal helper package `pkg/auditor/helpers` for:
   - HTTP client with trace correlation
   - Merkle path verification
   - WAL parsing primitives
   - Rotation weight computation (post multi-sig feature)
3. Mode Implementations (order): revocation inclusion, consistency, capability anchor, domain signature, replay WAL integrity, algorithm agility, rotation summary.
4. JSON Output Standardization: Introduce `ResultEnvelope` struct.
5. Add Tests: Unit tests for Merkle verification, WAL parsing edge cases (truncated file, corrupted entry), rotation weight calculation.
6. Documentation: Embed usage examples in README or `--help` output.

## Edge Cases & Risks

- Truncated WAL: Mode returns partial integrity result with `"success": false` and lists last valid offset.
- Large Consistency Proof: Ensure logarithmic proof generation avoids memory blow-up.
- Algorithm Monoculture: Flag if only a single algorithm present and policy demands diversity.
- Threshold Drift (multi-sig): Detect if effective weight < declared threshold.

## Future Extensions

- Streaming audit mode (continuous revocation monitoring).
- gRPC support for low-latency verification.
- Signed auditor reports (attest results for external verifiers).

## Acceptance Criteria

| Criterion | Definition |
|-----------|------------|
| Mode Coverage | All listed modes implemented with help output and JSON result envelope. |
| Test Coverage | >80% line coverage for auditor helper utilities. |
| Performance | Revocation inclusion/consistency < 150ms median locally. |
| Documentation | Each mode documented with examples. |
| Stability | Handles corrupted WAL without panic. |
