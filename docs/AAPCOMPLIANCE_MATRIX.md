---
title: Rfc Compliance Matrix
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# RFC Compliance Matrix (Initial Draft)

> Last Updated: 2025-10-17
> Status: Active

This document maps (placeholder form) intended AgentAuth AgentAuth-RFC-001 (formerly RFC 111) / AgentAuth-RFC-002 (formerly RFC 115) requirement areas to current implementation artifacts. It is **incomplete** and serves as a planning scaffold.

| Area | RFC Reference (Placeholder) | Current Implementation Component | Status | Gaps / TODO |
|------|-----------------------------|----------------------------------|--------|-------------|
| Policy Bundle Chaining | RFC111 §Chain Integrity | `pkg/policy` (hashing & AddBundle) | Partial | Formal spec hash algorithm & tamper test referencing RFC section missing |
| Bundle Substitution Detection | RFC111 §Substitution | `test/policy_bundle_substitution_test.go` | Partial | Detection via hash mismatch implemented; need formal RFC error codes & negative multi-hop scenarios |
| Provenance Endpoint | RFC111 §Provenance Query | `web/server_clean.go` `/policy/provenance` | Partial | Need response schema alignment with RFC format & negative cases |
| Pagination of Chain | RFC111 §Chain Listing | `/policy/chain` endpoint | Partial | Need total count & consistent ordering invariants documented |
| Audit Chain Integrity | RFC111 §Audit Linking | `audit` package (hash chain) | Partial | Cross-reference RFC integrity MUSTs & add collision simulation test |
| Policy Evaluation Provenance | RFC111 §Decision Trace | Audit append w/ `bundle_hash` + `chain_head` | Partial | Need multi-bundle historical evaluation replay tests |
| Delegation / POA Creation | RFC115 §Delegation Artifacts | `pkg/delegation/delegation.go` + `web/apiAuthorizePOA` + tests (`test/delegation_chain_test.go`, `web/delegation_authorize_test.go`) | Partial | Missing signatures, canonical serialization, enforcement binding to policy/authorization (metadata only) |
| Delegation Scope Validation | RFC115 §Scope Limits | `delegation.ValidateScopeNarrowing` + chain tests + authorization widening rejection | Partial | Only equality narrowing; need advanced semantics (range/regex), integrate into token issuance scope reduction |
| Revocation Handling | RFC115 §Revocation | Placeholder (not implemented) | Missing | Add Revocation struct, chain, tests & integration with verification |
| Expiry Enforcement | RFC115 §Temporal Validity | Delegation expiry check in `VerifyChain()` + test | Partial | Need evaluation-time denial responses & clock skew handling |
| Chain Anchor Externalization | RFC111 §External Anchor | Anchor callback in audit logger | Partial | Persist anchor & prove external verification procedure |
| Cryptographic Algorithms | RFC111/115 §Crypto | Hash only (algorithm unspecified) | Missing | Document chosen hash & key scheme, add signature verification |
| Error Codes & Semantics | RFC111/115 §Errors | Generic HTTP JSON errors | Missing | Define structured error taxonomy mapping to RFC clauses |
| Interoperability (Cross-Impl) | RFC111/115 §Interoperability | None | Missing | Add harness testing against second independent implementation |
| Compliance Test Suite | RFC111/115 §Conformance | Sparse / placeholders | Missing | Build table-driven tests enumerating MUST/SHOULD items |

## Notes
- Sections labeled "RFC Reference" are placeholders pending precise clause citations.
- "Partial" denotes functionality exists but lacks formal RFC-aligned documentation, tests, or negative scenarios.
- This matrix will evolve; each row should eventually link to specific test file names and line references.

## Next Steps
1. Fill RFC section citations once spec text is ingested.
2. Expand bundle substitution tests (multi-index tamper, replay scenarios).
3. Enhance delegation artifacts (signature, canonical JSON, parent chain cross-link).
4. Introduce revocation semantics (Revocation chain + evaluation filtering) and extend expiry handling to authorization path.
5. Formalize error taxonomy with mapped codes.
6. Define cryptographic algorithm choices (hash + signatures) and enforce them.
7. Build interoperability harness.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
