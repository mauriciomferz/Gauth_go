---
title: Technical Debt Inventory
category: technical-debt
status: active
lastUpdated: 2025-11-12
owners: engineering
---
# Technical Debt Inventory

Curated from TODO/FIXME markers and observed gaps. Categorized with priority (P0 critical → P3 low).

## P0 (Critical)
- Distributed PDP cache invalidation strategy (stale authorization decisions)
- Attestation chain depth verification & truncation safeguards
- CBOR encoding/decoding for PoA artifacts (interoperability requirement)

## P1 (High)
- Metrics persistence compaction & rotation to prevent unbounded growth
- Formal error catalog normalization (consistent codes, HTTP mapping)
- Jurisdiction enforcement metrics completeness (add missing counters)
- Revocation chain incremental hashing performance optimization

## P2 (Medium)
- Policy adapter: richer tracing spans for decision branches
- Extended token introspection detail standardization
- PoA validation: structured restriction expressions (beyond simple map[string]any)
- Anchor provider abstraction for external timestamping providers

## P3 (Low)
- Demo examples drift (update to latest struct shapes)
- README cross-links for all compliance reports
- Consolidate Dockerfiles (see DOCKERFILES_SUMMARY.md)
- Expand error boundary UI component & health badge visual states

## Proposed Remediation Sequence
1. Implement CBOR codec + tests (P0) → unlock interoperability scenarios.
2. Add PDP cache invalidation hooks & eviction metrics (P0/P1).
3. Formalize error catalog (generate JSON + validation CI step) (P1).
4. Add metrics compaction job (P1) with size threshold env variables.
5. Implement attestation chain safe depth enforcement (P0) + metrics.
6. Introduce structured restriction language for PoA (P2).
7. Dockerfile consolidation & buildx multi-arch (P3, fast win). 

## Tracking Template
| Item | Priority | Owner | Status | Notes |
|------|----------|-------|--------|-------|
| CBOR PoA codec | P0 | TBD | Pending | Define canonical schema first |
| PDP cache invalidation | P0 | TBD | Pending | Consider TTL + event broadcast |
| Error catalog normalization | P1 | TBD | Pending | Add generator script |
| Metrics compaction | P1 | TBD | Pending | Requires persistence path check |
| Attestation chain depth | P0 | TBD | Pending | Add MAX_CHAIN_DEPTH env |
| Revocation hash perf | P1 | TBD | Pending | Benchmark large chains |
| Anchor provider abstraction | P2 | TBD | Pending | Interface + registry |
| Dockerfile consolidation | P3 | TBD | Pending | See summary doc |

---
*Update priorities as context evolves; integrate with issue tracker for accountability.*
