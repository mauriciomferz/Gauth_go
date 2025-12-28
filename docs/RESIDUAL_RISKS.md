# Residual Risk Register
## GAuth RFC 0111/0115 Implementation

**Document Version**: 1.1  
**Last Updated**: 2025-12-27  
**Status**: Active

This document tracks unmitigated security risks, their severity, likelihood, and planned mitigation strategies, fulfilling sec14.item2 (Residual Risk Register).

---

## Risk Assessment Criteria

### Severity Levels
- **Critical**: System compromise, data breach, regulatory violations
- **High**: Significant service degradation, partial data exposure
- **Medium**: Limited impact, workarounds available
- **Low**: Minimal impact, cosmetic issues

### Likelihood Levels
- **High**: Expected to occur frequently (>10% probability per year)
- **Medium**: May occur occasionally (1-10% probability per year)
- **Low**: Unlikely but possible (<1% probability per year)

### Risk Score
Risk Score = Severity × Likelihood (scale: Critical=4, High=3, Medium=2, Low=1)

---

## Active Residual Risks

(No active risks remaining)

---

## Accepted Risks (Require No Immediate Action)

| Risk ID | Risk Description | Severity | Likelihood | Risk Score | Acceptance Rationale | Review Date |
|---------|-----------------|----------|------------|------------|---------------------|-------------|
| AR-001 | **Arbitration webhook failures** | Low | Low | 1 | Webhooks are optional, manual arbitration available | Q4 2025 |
| AR-002 | **Distributed tracing overhead** | Low | Medium | 2 | Single-instance deployment doesn't require tracing | Q3 2025 |
| AR-003 | **Property test corpus incompleteness** | Low | Low | 1 | Fuzzing provides adequate coverage for MVP | Q2 2025 |
| AR-004 | **Load testing at scale** (>100K ops/sec) | Medium | Low | 2 | Current baseline (66K ops/sec) meets SLA requirements | Q3 2025 |
| AR-005 | **Policy versioning race conditions** | Low | Low | 1 | Approval workflow prevents concurrent updates | Q4 2025 |

---

## Mitigated Risks (Archived)

| Risk ID | Original Risk | Severity | Mitigation Implemented | Completion Date |
|---------|--------------|----------|------------------------|-----------------|
| MR-001 | Token replay attacks | Critical | Durable JTI tracking with fail-closed mode | 2025-01 |
| MR-002 | Scope escalation | Critical | Scope inheritance validation | 2025-01 |
| MR-003 | Audit log tampering | Critical | Immutable hash-chained ledger + external TSA | 2025-01 |
| MR-004 | UTF-8 injection | Medium | UTF-8 validation with metrics | 2025-01 |
| MR-005 | Delegation depth bombs | High | Configurable max depth enforcement | 2025-01 |
| MR-006 | AI resource exhaustion | High | Model metadata limits (token/cost/rate) | 2025-01 |
| MR-007 | Cross-border data violations | Critical | Jurisdiction enforcement | 2025-01 |
| MR-008 | Multi-signature bypass | Critical | M-of-N threshold verification | 2025-01 |
| MR-009 | Capability registry tampering | High | RFC-3161 TSA anchoring with CMS verification | 2025-12-27 |
| MR-010 | Discovery metadata spoofing | Medium | JWKS signatures and deprecation schedules | 2025-12-27 |
| MR-011 | Cascade revocation performance | Medium | Optimized recursive traversal + background processing | 2025-12-28 |
| MR-012 | Replay store exhaustion | Medium | O(1) LRU eviction + configurable TTL | 2025-12-28 |
| MR-013 | Temporal validation clock skew | Medium | NTP enforcement + clock drift monitoring | 2025-12-28 |
| MR-014 | Audit ledger size growth | Medium | Automated archival with compression and pruning | 2025-12-28 |
| MR-015 | Key Material Exposure | High | AWS KMS Integration for detached signing keys | 2025-12-28 |
| MR-016 | Distributed PDP cache stampede | Medium | Request coalescing (singleflight) + pub/sub invalidation | 2025-12-28 |
| MR-017 | Key rotation downtime | Medium | Dual-key validation (LookupPublicKey) for zero-downtime rotation | 2025-12-28 |
| MR-018 | AI model cost tracking drift | Medium | Persistent daily cost tracking (CostTracker) to JSON/disk | 2025-12-28 |
| MR-019 | Conditional DSL injection attacks | High | Regex caching + strict ExprLimits enforcement | 2025-12-28 |
| MR-020 | Metrics collector outage (no fallback) | Low | Secondary backend (LoggingMetrics) + automatic failover | 2025-12-28 |
| MR-021 | Obligations execution failures (no retry) | High | Sync/Async retry logic (RetryingObligationExecutor) + Backoff | 2025-12-28 |
| MR-022 | Multi-signature coordinator SPOF | High | DB-backed concurrency-safe signature collection (AddMultiSignature) | 2025-12-28 |
| MR-023 | Evidence storage integrity | Medium | Local Content-Addressable Storage (CAS) with SHA-256 verification | 2025-12-28 |
| MR-024 | Jurisdiction policy staleness | Medium | Automated policy sync via PolicySyncer/FileSource | 2025-12-28 |
| MR-025 | Policy conflict deadlocks | Low | Role Hierarchy Cycle Detection (Graph Validation) | 2025-12-28 |
| MR-026 | MCP unauthorized resource access | Critical | MCP Authorization Bridge with mandatory PDP checks | 2025-12-28 |
| MR-027 | MCP audit trail missingness | High | Enforced MCP auditing via AuditLogger integration | 2025-12-28 |
| MR-028 | MCP server identity spoofing | Medium | Strict token-based identity propagation in MCPHandler | 2025-12-28 |

---

## Risk Prioritization (By Risk Score)

### Urgent (Score ≥ 6)
(No urgent risks remaining)

### High Priority (Score 4-5)
(No high priority risks remaining)

### Medium Priority (Score 2-3)
(No medium priority risks remaining)

### Low Priority (Score 1)
(No low priority risks remaining)

---

## Mitigation Implementation Plan

### Q1 2025 (Current Quarter)
- [x] ~~Create residual risk register~~ (RR documentation)
- [x] **RR-006**: Implement replay store eviction policy (DONE 2025-12-28)
- [x] **RR-015**: Add NTP enforcement and clock drift monitoring (DONE 2025-12-28)
- [x] **RR-009**: Ledger archival implementation (DONE 2025-12-28)
- [x] **RR-013**: Key Management Abstraction (Phase 1 & 2 Completed - 2025-12-28)

### Q2 2025
- [ ] (No pending risks this quarter)

- [x] **RR-014**: Cascade revocation optimization (DONE 2025-12-28)

### Q3 2025
- [x] **RR-008**: Policy dependency validation (DONE 2025-12-28)

### Q4 2025
- [x] **RR-004**: IPFS/Arweave evidence storage (CAS implemented 2025-12-28)
- [x] **RR-007**: Distributed multi-signature coordinator (DONE 2025-12-28)
- [x] **RR-011**: Automated policy sync (DONE 2025-12-28)
- [x] **MCP Security Hardening**: Authz and Auditing for MCP endpoints (DONE 2025-12-28)

---

## Risk Review Schedule

- **Quarterly Review**: All residual risks reassessed for severity/likelihood
- **Monthly**: Critical/High risks monitored for changes
- **Incident-Triggered**: Immediate review when related incidents occur

**Next Review Date**: 2025-04-01

---

## Risk Reporting

**Escalation Path**:
1. Engineering Team → Risk Owner
2. Risk Owner → Architecture Team (Score ≥ 6)
3. Architecture Team → CTO (Critical Severity)

**Reporting Frequency**:
- Critical risks: Weekly status updates
- High risks: Bi-weekly updates
- Medium/Low risks: Monthly updates

---

## References

- [THREAT_MITIGATIONS_MATRIX.md](THREAT_MITIGATIONS_MATRIX.md): Threat-to-mitigation mapping (sec14.item1)
- [REMAINING_GAPS_ANALYSIS.md](REMAINING_GAPS_ANALYSIS.md): Architectural gaps discussion
- [GAP_MATRIX.md](GAP_MATRIX.md): Conformance tracking
- RFC 0111/0115: Protocol specifications
