---
title: Hierarchical Digest Design
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Hierarchical Digest Domain Design (Draft)

Date: 2025-10-30

## Purpose
Introduce a structured digest domain version that safely incorporates hierarchical delegation context (parent linkage, depth, and advanced inheritance patterns) without breaking integrity for existing PoAs.

## Current State
- Digest versions: V1 (single-sig base), V2 (multi-sig domain separation), V3 (taxonomy enrichment).
- Hierarchical fields (`parent_poa_id`, `depth`) are excluded from canonical digest to avoid retroactive invalidation of existing signatures.
- Advanced scope inheritance (regex/prefix) now supported behind `GAUTH_ENABLE_ADVANCED_SCOPE`.

## Goals
1. Preserve existing signatures for legacy PoAs (V1–V3 remain valid).
2. Provide replay-resistant inclusion of hierarchical attributes for new PoAs opting into hierarchical domain.
3. Support deterministic digest computation when parent hierarchy changes (prevent silent privilege elevation).
4. Avoid digest instability from derived fields (e.g., `depth`) changing due to migration or re-parenting.

## Non-Goals
- Retroactive re-issuing digests for existing PoAs.
- Encoding full ancestor chain inline (size concerns). We prefer anchored root hash or parent ID only.

## Proposed Version: V4 ("hier:v1")
New domain format activates when either:
- `parent_poa_id` is non-empty, OR
- Advanced inheritance patterns (regex) are present in `scope` entries, OR
- Explicit environment `GAUTH_FORCE_HIER_DIGEST=1` set.

### Canonical Fields for V4
Included (ordered JSON keys to maintain stable hashing):
1. `id`
2. `version` (structural numeric)
3. `grantor`
4. `grantee`
5. `scope` (full array) – regex entries maintained literally.
6. `restrictions` (sorted by key)
7. `agent_type`, `sector`, `action_class` (taxonomy)
8. `valid_from`, `valid_until` (RFC3339Nano UTC)
9. `parent_poa_id` (string) – direct linkage only.
10. `parent_digest` (optional) – canonical digest of parent if available and parent is already persisted; absence represented as null.
11. `hier_depth` (integer) – stable copy of derived depth; computed once at issuance, immutable thereafter.

Excluded:
- Dynamic runtime revocation workflow fields.
- Witnesses/Attestations (remain evidentiary, not structural).
- `revoked_at`, `revocation_reason`.

### Stability Rules
- `hier_depth` becomes immutable after issuance; re-parenting is not supported (must revoke and re-issue child).
- If parent is revoked post issuance, child digest remains unchanged (authorization logic handles cascade separately if needed).

### Parent Digest Inclusion
Pros:
- Tamper-evident chain: changing parent body invalidates child signatures.
Cons:
- Parent updates (e.g., restrictions) would invalidate descendants.
Mitigation: Parent structural mutation requires revocation + new issuance; treat PoA records as immutable except status changes.
Decision: INCLUDE `parent_digest` for V4 to bind subtree integrity.

### Upgrade Path
1. Add conditional digest constructor path for hierarchical activation.
2. Gate activation behind environment until sufficient test coverage (`GAUTH_ENABLE_HIER_DIGEST=1`).
3. Introduce metrics: ✅ **COMPLETED**
   - `hier_digest_issued_total` (dedicated Prometheus counter with fqName registration)
   - `hier_digest_parent_digest_missing_total` (dedicated counter for parent retrieval failures)
   - `hier_digest_version_mismatch_total` (dedicated counter for version validation failures)
4. Add unit tests:
   - Root PoA (no hierarchy) stays V3.
   - Child PoA with parent -> V4 digest includes parent digest.
   - Regex scope triggers V4 even without parent.
   - Attempted depth mutation (simulate re-parent) fails validation.

### Verification Impacts
- Validation must check if parent digest referenced matches stored parent canonical digest when verifying child.
- Revocation cascade optional: If parent revoked, child may either remain valid or be auto-marked `suspended`; configurable via `GAUTH_CASCADE_PARENT_REVOCATION=1`.

### Security Considerations
- Binding parent digest prevents stealth widening by swapping parent object.
- Regex patterns in scope are frozen in digest to prevent later reinterpretation.
- Depth integer inclusion guards against silent depth re-assignment.

### Cascade Revocation Semantics

When `GAUTH_CASCADE_PARENT_REVOCATION=1` is enabled, parent PoA revocation triggers automatic status changes for descendant PoAs to prevent unauthorized privilege retention.

#### Implementation Options

**Option 1: Immediate Revocation Cascade (Strict)**
- Parent revocation immediately marks all descendants as `revoked`
- Recursively processes entire subtree in single transaction
- Pros: Strong security guarantee, clear semantics
- Cons: Potential performance impact for deep hierarchies, irreversible

**Option 2: Suspension Cascade (Conservative)**
- Parent revocation marks descendants as `suspended` pending review
- Allows manual restoration if parent revocation was accidental
- Requires administrative intervention to finalize revocation or restore
- Pros: Recoverable, operational flexibility
- Cons: Suspended state requires careful handling in authorization logic

**Option 3: Configurable Cascade Depth**
- Environment variable `GAUTH_CASCADE_MAX_DEPTH=N` limits cascade scope
- Prevents runaway cascades in very deep hierarchies
- Combined with either immediate or suspension semantics
- Pros: Performance protection, operational control
- Cons: Security gap if depth limit too restrictive

#### Recommended Implementation (Phase 2)

```bash
GAUTH_CASCADE_PARENT_REVOCATION=1          # Enable cascade processing
GAUTH_CASCADE_MODE=suspend                 # suspend|revoke|notify
GAUTH_CASCADE_MAX_DEPTH=10                # Max cascade depth (0=unlimited)
GAUTH_CASCADE_BATCH_SIZE=100              # Process descendants in batches
```

#### Cascade Processing Algorithm

1. **Parent Revocation Initiated**: Standard revocation workflow completes
2. **Descendant Discovery**: Query repository for `parent_poa_id` matches
3. **Batch Processing**: Process descendants in configurable batch sizes
4. **Status Update**: Apply cascade mode (suspend/revoke) with audit logging
5. **Recursive Processing**: Apply to child descendants up to max depth
6. **Metrics Emission**: Track cascade events, depth reached, processing time

#### Metrics for Cascade Operations

```go
// New metrics for cascade revocation tracking
IncCascadeRevocationTriggered()              // Parent revocation initiated cascade
IncCascadeDescendantsProcessed(depth int)    // Descendants affected by depth level
ObserveCascadeProcessingLatency(duration)    // Time to complete cascade operation
IncCascadeDepthLimitReached()               // Hit configured max depth limit
IncCascadeBatchProcessed(count int)         // Batch processing progress
```

#### Security Considerations

- **Audit Trail**: All cascade operations logged with full lineage context
- **Idempotency**: Repeated cascade operations safe (already suspended/revoked)
- **Authorization**: Cascade inherits parent revocation authorization context
- **Recovery**: Suspended descendants can be restored via administrative endpoint

#### Error Handling

- **Partial Failure**: Continue processing remaining descendants, log failures
- **Repository Errors**: Retry with exponential backoff, surface metrics
- **Depth Limits**: Log warning when cascade truncated by depth limit
- **Performance**: Async processing for large subtrees with progress tracking

### Open Questions
- Should multi-signature domain (V2) automatically roll into V4 for hierarchical PoAs, or remain distinct? (Proposal: V4 supersedes V2/V3; threshold semantics coexist.)
- Performance impact of fetching parent digest during issuance (minimal; single Get + canonical recompute).
- Need Merkle subtree hashing for large hierarchies? (Future optimization.)
- **Cascade Questions**:
  - Should cascade operations require additional authorization beyond parent revocation rights?
  - How to handle cycles in delegation chains (malformed data protection)?
  - Integration with dual-control revocation workflow (cascade during suspension vs after approval)?

## Implementation Checklist
- [x] Add env gate `GAUTH_ENABLE_HIER_DIGEST`.
- [x] Extend canonical digest builder with V4 branch.
- [x] Include parent digest fetch + error handling metric (missing parent counter wired).
- [x] Emit hierarchical digest counters (issued, parent_digest_missing, version_mismatch) - **FULLY IMPLEMENTED** with dedicated Prometheus counters.
- [x] Validation path: verify parent digest on child validation (fails on mismatch/missing canonicalization).
- [x] Tests covering activation, tamper, disabled path, validation mismatch.
- [ ] Documentation updates (README + OpenAPI + assessment cross-reference cascade semantics).
- [ ] **Cascade revocation implementation** (Phase 2):
  - [ ] Environment flags: `GAUTH_CASCADE_PARENT_REVOCATION`, `GAUTH_CASCADE_MODE`, `GAUTH_CASCADE_MAX_DEPTH`
  - [ ] Descendant discovery via `parent_poa_id` repository queries
  - [ ] Batch processing with configurable size limits
  - [ ] Cascade-specific metrics (triggered, processed, latency, depth limits)
  - [ ] Administrative restoration endpoint for suspended descendants
  - [ ] Integration testing with deep delegation hierarchies

## Rollback Plan
Retain V3 path; if hierarchical domain causes issues, disable via env and re-issue affected hierarchical PoAs using legacy approach (revocation + issuance).

---
Generated: 2025-10-30

## Activation & Rollout Status (Post Partial Implementation)

Implemented (Phase 1):
* Env flags wired: `GAUTH_ENABLE_HIER_DIGEST`, `GAUTH_FORCE_HIER_DIGEST`.
* Structural field `parent_digest` added to PoA; canonical JSON emits hierarchy object for Version>=4.
* V4 domain sentinel: `GAUTH_RFC0111_POA_V4|hier=1`.
* **Metrics fully implemented**: Dedicated Prometheus counters with proper registration (`hierDigestIssued`, `hierDigestParentDigestMissing`, `hierDigestVersionMismatch`) in both in-memory and Prometheus adapter implementations.
* Unit tests: root & child issuance (Version=4), disabled flag path retains Version<4, parent digest tamper changes canonical digest.

Pending (Phase 2):
* Validation-time parent digest verification for V4 PoAs.
* Version mismatch counter surfacing when expectations diverge.
* README / API reference updates & OpenAPI schema augmentation for hierarchy object.
* Optional cascade semantics via `GAUTH_CASCADE_PARENT_REVOCATION`.

Operational Guidance:
* Enable gradually with `GAUTH_ENABLE_HIER_DIGEST=1`; monitor hierarchical issuance vs total.
* Use `GAUTH_FORCE_HIER_DIGEST=1` after confirming parent retrieval reliability.
* Rollback by unsetting flags; existing Version=4 PoAs remain until revoked.

Monitoring & Alerting:
* Alert if `hier_digest_parent_digest_missing_total` increases rapidly (parent fetch failures).
* Monitor `hier_digest_version_mismatch_total` for validation failures indicating inconsistent expectations.
* Track `hier_digest_issued_total` for hierarchical adoption rate.
* Future ratio gauge: hierarchical adoption percentage.

Security Gains:
* Parent tamper cascades to child signature invalidation (improved lineage integrity).
* Depth immutability blocks stealth chain restructuring.

## Cascade Revocation Implementation Plan

### Phase 2a: Core Infrastructure (Week 4)
1. **Environment Configuration**
   - Add flags: `GAUTH_CASCADE_PARENT_REVOCATION`, `GAUTH_CASCADE_MODE`, `GAUTH_CASCADE_MAX_DEPTH`, `GAUTH_CASCADE_BATCH_SIZE`
   - Configuration validation and defaults
   - Documentation in environment variables reference

2. **Repository Extensions**
   - Add `ListDescendants(parentID string, maxDepth int) ([]*PowerOfAttorney, error)` method
   - Optimize query performance with proper indexing on `parent_poa_id`
   - Batch processing support for large result sets

3. **Cascade Engine**
   - Core cascade processor with depth-limited traversal
   - Batch processing with configurable sizes
   - Error handling and partial failure recovery
   - Audit logging for all cascade operations

### Phase 2b: Metrics & Observability (Week 4)
1. **Cascade Metrics**
   - Extend metrics interface with cascade-specific counters
   - Prometheus adapter implementation with proper labels
   - Dashboard queries for cascade operation visibility

2. **Testing Framework**
   - Unit tests for cascade engine with mock hierarchies
   - Integration tests with realistic delegation trees
   - Performance tests for deep hierarchies (depth 10+)
   - Error injection tests for partial failure scenarios

### Phase 2c: API & Integration (Week 5)
1. **Administrative Endpoints**
   - `POST /api/v1/poa/{id}/restore` for suspended descendants
   - `GET /api/v1/poa/{id}/descendants` for impact assessment
   - Proper authorization checks and audit logging

2. **Dual-Control Integration**
   - Cascade processing during dual-control revocation workflow
   - Suspension during pending approval, cascade on final approval
   - Clear semantics for cancel operations with cascaded descendants

### Rollout Strategy
1. **Testing Phase**: Enable in development with synthetic hierarchies
2. **Canary Phase**: Enable with `cascade_mode=notify` (metrics only, no status changes)
3. **Limited Rollout**: Enable `cascade_mode=suspend` for low-impact use cases
4. **Full Production**: Enable `cascade_mode=revoke` after validation

### Success Metrics
- Zero cascade operation failures in production
- Cascade processing latency < 100ms for depth <= 5
- Complete audit trail for all cascade operations
- Administrative restoration success rate > 99%

---
