# RFC 0111/0115 Conformance - Remaining Gaps Analysis

## Date: 2025
## Status: For Architectural Discussion

This document analyzes the remaining conformance gaps that require architectural decisions or major new subsystems before implementation.

---

## P2 Priority Gaps (Requires Architecture)

### sec2.item3: Obligations & Advice Processing
**Status**: Deferred - Requires distributed execution framework
**Gap**: "Concept only, not executed"

**Current State**:
- Authorization engine returns simple allow/deny decisions
- No post-decision obligation execution
- No advice/warning collection beyond validation warnings

**Minimal Implementation Path**:
1. Extend `authz.Decision` with `Obligations []Obligation` and `Advice []Advice`
2. Create `pkg/authz/obligations.go` with:
   - `Obligation` struct (action, target, params, deadline)
   - `Advice` struct (suggestion, severity, metadata)
   - `ObligationExecutor` interface
3. Implement basic executors:
   - Logging obligation (audit trail)
   - Notification obligation (webhook/email)
   - Temporal obligation (scheduled action)

**Production Requirements**:
- Distributed task queue (RabbitMQ/Kafka)
- Retry/compensation logic
- Obligation state persistence
- Monitoring/alerting for failed obligations

**Recommendation**: Implement minimal logging obligations for P1, defer distributed execution to post-MVP.

---

### sec2.item5: Distributed PDP & Caching
**Status**: Deferred - Requires clustering & cache invalidation
**Gap**: "No clustering or cache invalidation"

**Current State**:
- Single-instance PDP (internal/policy/pdp.go)
- No policy cache
- No distributed coordination

**Minimal Implementation Path**:
1. Add in-memory policy cache with TTL
2. Implement cache invalidation on policy updates
3. Add `PolicyCache` interface for pluggable backends

**Production Requirements**:
- Redis/Memcached for shared cache
- Pub/sub for cache invalidation (Redis PUBLISH)
- Consistent hashing for multi-node PDP
- Circuit breaker for cache failures

**Recommendation**: Add TTL cache for P1, defer distributed clustering to horizontal scaling phase.

---

### sec3.item4: Conditional/Special Conditions Evaluation
**Status**: Deferred - Requires DSL interpreter
**Gap**: "No runtime interpreter"

**Current State**:
- Basic restriction checks (max_amount, currency, time windows)
- No dynamic conditional evaluation
- No expression language

**Minimal Implementation Path**:
1. Define simple expression DSL:
   ```
   "context.time >= 09:00 AND context.time <= 17:00"
   "delegation.scope CONTAINS 'finance' AND user.department == 'accounting'"
   ```
2. Implement parser using `github.com/antonmedv/expr` or similar
3. Add `Condition` field to PowerOfAttorney
4. Evaluate conditions in ValidateDelegation()

**Production Requirements**:
- Security review of expression language (injection risks)
- Performance testing (complex conditions)
- Audit logging of condition evaluations

**Recommendation**: Use existing restriction framework, add expression support in v2.

---

### sec4.item2: Compliance Attestation Proof
**Status**: Deferred - Requires evidence management
**Gap**: "No evidence ingestion"

**Current State**:
- PowerOfAttorney has `EvidenceHashes []string` field (unused)
- No evidence storage or verification

**Minimal Implementation Path**:
1. Create `pkg/evidence/` package:
   - `Evidence` struct (hash, type, timestamp, source)
   - `EvidenceStore` interface (Get/Store/Verify)
2. Implement basic file-based store
3. Add evidence attachment to delegation creation
4. Validate evidence integrity on verification

**Production Requirements**:
- IPFS/Arweave for immutable storage
- Evidence cryptographic verification (signatures)
- Compliance reporting (GDPR/HIPAA evidence trails)

**Recommendation**: Implement hash-based evidence references for P1, defer full management to compliance phase.

---

## P3 Priority Gaps (Low Priority / Documentation)

### sec4.item3: Arbitration / Dispute Hooks
**Status**: Documentation stub
**Gap**: "No code path"

**Description**: Webhook integration for external dispute resolution systems.

**Minimal Implementation**:
- Add `ArbitrationWebhook string` to PowerOfAttorney
- Trigger webhook on revocation disputes
- Log dispute metadata for audit

**Recommendation**: Document interface, implement when legal disputes arise.

---

### sec7.item4: Distributed Tracing
**Status**: Deferred - Low priority for V1
**Gap**: "No span linking"

**Description**: OpenTelemetry span propagation across service boundaries.

**Minimal Implementation**:
- Import `go.opentelemetry.io/otel`
- Add span creation in key functions
- Propagate context with trace IDs

**Recommendation**: Add in service mesh deployment, not critical for single-instance.

---

### sec14.item2: Residual Risk Register
**Status**: Documentation task
**Gap**: "No tracking of remaining exposures"

**Description**: Maintain document of unmitigated risks with severity/likelihood.

**Implementation**: Create `docs/RESIDUAL_RISKS.md` with:
- Risk ID, description, severity, likelihood
- Mitigation strategy (when planned)
- Acceptance criteria

**Recommendation**: Create document as part of security audit preparation.

---

## Partial Implementation Gap

### sec14.item1: Threat Model Synchronization
**Status**: Needs mitigations matrix
**Gap**: "No mitigations matrix"

**Current State**:
- Threat model exists conceptually
- No structured threat-to-mitigation mapping

**Implementation**: Create `docs/THREAT_MITIGATIONS_MATRIX.md`:
```
| Threat ID | Threat | Mitigation | Implementation | Status |
|-----------|--------|------------|----------------|--------|
| T1 | Token replay | JTI tracking + expiry | pkg/replay/durable_replay_store.go | Complete |
| T2 | Unauthorized delegation | PDP enforcement | internal/policy/pdp.go | Complete |
| T3 | Scope escalation | Scope inheritance validation | pkg/rfc0111/rfc0111.go:1885 | Complete |
```

**Recommendation**: Create matrix document, mark as Implemented after review.

---

## Summary

**Implementation Strategy**:

1. **Implement Immediately (closes gaps)**:
   - [ ] Threat mitigations matrix document (sec14.item1)
   - [ ] Residual risk register document (sec14.item2)
   - [ ] Minimal obligations logging (sec2.item3 partial)
   - [ ] Evidence hash references (sec4.item2 partial)

2. **Document for Future Implementation**:
   - [ ] Distributed PDP architecture (sec2.item5)
   - [ ] Conditional DSL specification (sec3.item4)
   - [ ] Arbitration webhook interface (sec4.item3)
   - [ ] Distributed tracing integration (sec7.item4)

3. **Gap Matrix Updates**:
   - sec14.item1: Missing → Implemented (after matrix creation)
   - sec14.item2: Missing → Documented (after register creation)
   - sec2.item3: Missing → Partial (with logging obligations)
   - sec4.item2: Missing → Partial (with hash references)
   - Others remain "Documented" with architecture notes
