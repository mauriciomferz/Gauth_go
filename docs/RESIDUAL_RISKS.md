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

| Risk ID | Risk Description | Severity | Likelihood | Risk Score | Mitigation Strategy | Target Date | Owner |
|---------|-----------------|----------|------------|------------|---------------------|-------------|-------|
| RR-001 | **Distributed PDP cache stampede** | Medium | Medium | 4 | Implement distributed cache with TTL + invalidation pub/sub | Q2 2025 | Architecture Team |
| RR-002 | **Obligations execution failures** (no retry) | High | Medium | 6 | Add distributed task queue (RabbitMQ/Kafka) with retry logic | Q3 2025 | Backend Team |
| RR-003 | **Conditional DSL injection attacks** | High | Low | 3 | Security review + sandboxed expression evaluation | Q2 2025 | Security Team |
| RR-004 | **Evidence storage integrity** (no cryptographic proof) | Medium | Low | 2 | Integrate IPFS/Arweave with signature verification | Q4 2025 | Compliance Team |
| RR-005 | **Key rotation downtime** (no zero-downtime rotation) | Medium | Medium | 4 | Implement dual-key validation during rotation windows | Q3 2025 | Crypto Team |
| RR-006 | **Replay store exhaustion** (unbounded growth) | Medium | High | 6 | Automatic eviction with configurable retention policies | Q1 2025 | **URGENT** |
| RR-007 | **Multi-signature coordinator SPOF** | High | Medium | 6 | Distributed signature collection with consensus protocol | Q4 2025 | Architecture Team |
| RR-008 | **Policy conflict deadlocks** (circular dependencies) | Low | Low | 1 | Policy dependency graph validation | Q3 2025 | Policy Team |
| RR-009 | **Audit ledger size growth** (no pruning) | Medium | High | 6 | Implement ledger archival with external cold storage | Q2 2025 | Storage Team |
| RR-010 | **AI model cost tracking drift** (no persistent state) | Medium | Medium | 4 | Add time-series DB for daily token/cost tracking | Q3 2025 | AI Governance Team |
| RR-011 | **Jurisdiction policy staleness** (no auto-update) | Medium | Low | 2 | Automated policy sync with regulatory databases | Q4 2025 | Compliance Team |
| RR-012 | **Metrics collector outage** (no fallback) | Low | Medium | 2 | Secondary metrics backend with automatic failover | Q3 2025 | Observability Team |
| RR-013 | **Detached signature key compromise** | Critical | Low | 4 | HSM integration for signing keys + automatic rotation | Q2 2025 | **HIGH PRIORITY** |
| RR-014 | **Cascade revocation performance** (large trees) | Medium | Medium | 4 | Optimize depth-first traversal + background processing | Q2 2025 | Performance Team |
| RR-015 | **Temporal validation clock skew** (>5min drift) | Medium | Low | 2 | NTP enforcement + clock drift monitoring | Q1 2025 | Infrastructure Team |

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

---

## Risk Prioritization (By Risk Score)

### Urgent (Score ≥ 6)
1. **RR-006**: Replay store exhaustion (Medium × High = 6) - **Q1 2025 DEADLINE**
2. **RR-002**: Obligations execution failures (High × Medium = 6)
3. **RR-007**: Multi-signature coordinator SPOF (High × Medium = 6)
4. **RR-009**: Audit ledger size growth (Medium × High = 6)

### High Priority (Score 4-5)
1. **RR-013**: Detached signature key compromise (Critical × Low = 4) - **HIGH PRIORITY**
2. **RR-001**: Distributed PDP cache stampede (Medium × Medium = 4)
3. **RR-005**: Key rotation downtime (Medium × Medium = 4)
4. **RR-010**: AI model cost tracking drift (Medium × Medium = 4)
5. **RR-014**: Cascade revocation performance (Medium × Medium = 4)

### Medium Priority (Score 2-3)
1. **RR-003**: Conditional DSL injection (High × Low = 3)
2. **RR-004**: Evidence storage integrity (Medium × Low = 2)
3. **RR-011**: Jurisdiction policy staleness (Medium × Low = 2)
4. **RR-012**: Metrics collector outage (Low × Medium = 2)
5. **RR-015**: Temporal validation clock skew (Medium × Low = 2)

### Low Priority (Score 1)
- **RR-008**: Policy conflict deadlocks (Low × Low = 1)

---

## Mitigation Implementation Plan

### Q1 2025 (Current Quarter)
- [x] ~~Create residual risk register~~ (RR documentation)
- [ ] **RR-006**: Implement replay store eviction policy (URGENT)
- [ ] **RR-015**: Add NTP enforcement and clock drift monitoring

### Q2 2025
- [ ] **RR-013**: HSM integration for signing keys (HIGH PRIORITY)
- [ ] **RR-001**: Distributed cache with pub/sub invalidation
- [ ] **RR-003**: Security review of conditional DSL
- [ ] **RR-009**: Ledger archival implementation
- [ ] **RR-014**: Cascade revocation optimization

### Q3 2025
- [ ] **RR-002**: Distributed task queue for obligations
- [ ] **RR-005**: Zero-downtime key rotation
- [ ] **RR-008**: Policy dependency validation
- [ ] **RR-010**: Time-series DB for AI cost tracking
- [ ] **RR-012**: Secondary metrics backend

### Q4 2025
- [ ] **RR-004**: IPFS/Arweave evidence storage
- [ ] **RR-007**: Distributed multi-signature coordinator
- [ ] **RR-011**: Automated policy sync

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
