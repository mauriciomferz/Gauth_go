# ADR: Capability Deprecation & Sunset Lifecycle

Date: 2025-10-19
Status: Proposed
Context: Capability governance now supports `deprecated_after` and `sunset_after` metadata fields. A formal lifecycle policy is required for consistent enforcement, client negotiation behavior, and external communication.

Summary: Defines a two-phase lifecycle for capability versions (deprecation and sunset), enforcement flags, audit chain integration, anchoring, and rollback plan. Implementation in beta-refactor branch.

References: See GAP_MATRIX Section 11, implementation in `web/server_clean.go`, tests in `web/capability_persistence_test.go`.

## Problem Statement
Without a standardized lifecycle, introducing new capability versions or retiring old ones risks:
- Inconsistent client negotiation results.
- Silent removal without advance warning.
- Difficulty auditing historical capability sets in relation to deprecation timelines.

## Decision
Define a two-phase lifecycle for each capability version:
1. Deprecation Phase (`deprecated_after`): Capability remains functional; discovery surfaces advisory warnings. Negotiation continues to include deprecated versions unless strict mode is enabled.
2. Sunset Phase (`sunset_after`): Capability is scheduled for removal. After this timestamp passes, enforcement treats the capability as invalid unless an override flag is set for emergency extension.

### Enforcement Modes
- `GAUTH_CAP_LIFECYCLE_STRICT=1`: API negotiation omits versions past `deprecated_after` (proactive migration) and rejects requests listing only deprecated versions.
- `GAUTH_CAP_LIFECYCLE_SUNSET_ENFORCE=1`: Capability usage past `sunset_after` yields `capability_enforce` denial with outcome `sunset`.

### Discovery & Info Exposure
`/api/v1/beta/info` and the well-known configuration will expose:
```jsonc
capabilities_meta: {
  capabilities: [
    {"id":"cap.transfer","versions":["1.0","1.1"],"stable":true,"deprecated_after":"2025-12-01T00:00:00Z","sunset_after":"2026-03-01T00:00:00Z"}
  ]
}
```
A `lifecycle` object will be added summarizing strict/sunset enforcement flags and upcoming milestones.

### Audit Chain Integration
When a capability is enforced (create/revoke or denial), if the denial reason is lifecycle-based (`deprecated` or `sunset`) the audit entry includes `{"lifecycle_reason":"deprecated"}` or `{"lifecycle_reason":"sunset"}` in metadata, and is hash-chained for integrity.

### Anchoring & External Verification
- The capability registry hash already anchors structural governance.
- We will anchor the capability audit chain tip periodically via `/api/v1/beta/capabilities/audit/anchor` reusing the existing `AnchorClient` abstraction (memory provider).
- Anchoring metadata will include the *latest tip hash* and *timestamp*, enabling external witnesses to correlate lifecycle transitions with governance events.

## Alternatives Considered
- Single timestamp (remove deprecated_after, keep only sunset_after): reduces communication granularity; rejected.
- Multi-stage (introducing warning_after, deprecated_after, sunset_after): increases complexity early; deferred until scaling needs.
- Immediate auto-removal on sunset timestamp without enforce flag: reduces operator control; rejected.

## Consequences
Positive:
- Predictable migration path for clients.
- Auditable lifecycle decisions with hash-chain provenance.
- Clear separation of advisory vs enforcement phases.

Negative / Risks:
- Additional complexity in negotiation logic (version filtering).
- Need for clear operator runbook (extending sunset window).

## Implementation Outline (Phased)
1. Add enforcement flags env parsing in `NewBetaServer`.
2. Extend negotiation endpoint to optionally filter deprecated (strict mode).
3. Add lifecycle denial logic in `enforceCapabilities` (check timestamps, produce denial reason).
4. Hash-chain metadata enrichment for lifecycle denials.
5. Add anchoring endpoint for audit chain tip.
6. Extend discovery with lifecycle summary.
7. Add tests: strict negotiation exclusion, deprecated denial when strict+sunset flags active, anchoring of audit chain tip.

## Rollback Plan
- Flags disabled => lifecycle enforcement paths return to advisory only.
- Removing fields leaves prior versions intact; clients see trimmed metadata.

## Future Work
- External timestamp attestation for audit chain tip anchors.
- Merkle subtree commitment for batch audit events.
- Automated reminder system for approaching sunset milestones.

---
Author: Automated Assistant
