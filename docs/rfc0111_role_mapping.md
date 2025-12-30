---
title: Rfc0111 Role Mapping
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# RFC0111 Role Mapping (Beta MVP)

Date: 2025-10-29

This document maps conceptual RFC0111 actors & functional planes (traditional OAuth-style roles + AgentAuth P*P components) to current code artifacts or identifies gaps.

## Legend
| Role | Description |
|------|-------------|
| Resource Owner | Entity that controls protected resource / action space |
| Client | Software agent requesting authorization |
| Authorization Server | Issues delegation & tokens; evaluates PoA validity |
| Resource Server | Validates tokens & enforces capability / PoA constraints |
| PDP | Policy Decision Point (makes allow/deny decisions) |
| PEP | Policy Enforcement Point (intercepts requests, invokes PDP) |
| PAP | Policy Administration Point (manages policies/governance rules) |
| PIP | Policy Information Point (provides attributes/claims/context) |
| PVP | Policy Validation Point (integrity, signature & multi-sig checks) |

## Role → Implementation Mapping
| RFC Role / Plane | Current Implementation | File / Package References | Status |
|------------------|------------------------|----------------------------|--------|
| Resource Owner | Represented implicitly by `grantor` in PoA | `pkg/rfc0111/rfc0111.go` (PowerOfAttorney.Grantor) | Implemented |
| Client | External caller using `/demo/enforce` or requesting token | `examples/ai_capability_demo/main.go` endpoints | Implemented |
| Authorization Server | Token issuance & PoA management endpoints | `/demo/poa/issue`, `/demo/poa/:id/token`, `/demo/poa/:id/revoke` in `main.go` | Partial (minimal issuance logic) |
| Resource Server | Enforcement of action with PoA validation | `authMiddleware` + `/demo/enforce` | Implemented |
| PDP | Capability evaluation engine | `internal/ai` (ServerIntegration.EnforceAICapabilities) | Implemented |
| PEP | HTTP handlers wrapping enforcement | `/demo/enforce` handler; Gin middleware | Implemented |
| PAP | Policy definitions loading (governance policies set) | `internal/ai` policy loader (GetGovernancePolicies) | Partial (no admin API) |
| PIP | Claims source (request JSON + JWT claims) | `main.go` request parsing + JWT claim extraction | Implemented |
| PVP | Signature/multi-sign enforcement & canonical digest | `pkg/rfc0111/canonical.go`, `verifyMultiSignatures`, multisig tests | Implemented |
| Delegation Repository | In-memory PoA map | `main.go` (poaRepo) | Partial (no persistent backend) |
| Audit Trail | Decision persistence + stdout audit callback | `persistDecision`, audit callback in `main.go` | Partial (no structured ledger) |
| Revocation Controller | Endpoint updates status + metrics | `/demo/poa/:id/revoke` | Implemented |
| Extended Token Processor | Issues JWT referencing PoA | `/demo/poa/:id/token` | Implemented (no digest embedding) |
| Multi-Signature Orchestrator | Threshold verification logic & tests | `pkg/rfc0111/*multi_signature*` | Implemented (no live collection API in demo) |

## Gaps & Remediation (Role Perspective)
| Gap | Role(s) Affected | Remediation |
|-----|------------------|------------|
| Missing persistent PoA storage | Authorization Server / Repository | Introduce BoltDB/Postgres implementation behind `POARepository` |
| No admin interface for policy updates | PAP | Add secured endpoints for policy CRUD and versioning |
| Lack of audit ledger with integrity chain | Authorization Server / Audit | Implement append-only ledger w/ hash chaining + export API |
| No sub-delegation validation | Authorization Server / PDP | Add parent-child linkage + depth enforcement |
| Extended token missing embedded digest/version | Authorization Server / Resource Server | Embed `poa_digest` & `poa_version` claims; verify in middleware |
| Multi-sig live collection not wired | Authorization Server / PVP | Integrate `internal/multisig` manager endpoints into demo |
| Suspension / termination states absent | Authorization Server / Resource Server | Extend `POAStatus` enum + enforcement logic |

## Sequenced Role-Focused Roadmap
1. Persistence & Ledger (Repository + Audit chain)
2. Extended Token Integrity (Digest + Version claims)
3. Sub-Delegation Support (Parent linkage + depth checks)
4. Multi-Sig Collection API Integration (Sign, status, activate flows)
5. Policy Admin Endpoints (Secure PAP management)
6. Lifecycle Expansion (Suspension / termination semantics)

---
_Generated: 2025-10-29_