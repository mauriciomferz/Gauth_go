# ADR: Taxonomy Extension Governance (agent_type, sector, action_class)

Status: Proposed
Date: 2025-10-28
RFC References: RFC-0111 (taxonomy), RFC-0115 (authorization semantics)

## Context
Current enumerations in `pkg/rfc0111/taxonomy.go` are static:
- AllowedAgentTypes: human, service, team, automation, robot, llm
- AllowedSectors: finance, health, legal, it, operations, security, research
- AllowedActionClasses: read_ops, write_ops, admin, transfer, decision, audit

Expansion is needed for domain specificity (e.g. manufacturing, energy) and future agent modalities (e.g. autonomous_vehicle). Uncontrolled addition risks semantic dilution and digest instability.

## Decision
Establish a controlled extension workflow ensuring backward compatibility and auditability:

1. Proposal PR includes:
   - Justification (use cases, avoiding overlap)
   - Collision analysis (similar existing term?)
   - Security review (no ambiguous or misleading term)
   - Migration notes (client impact, monitoring adjustments)
2. Each new value appended only (no reordering) to preserve index stability; removal prohibited—values may be deprecated.
3. Deprecation process: mark value in a `DeprecatedTaxonomyValues` map with sunset date; validation continues accepting value until removal threshold passes and major version bump occurs.
4. Canonical digest domain stays unchanged unless structural taxonomy format changes (e.g. nested categories). Adding values alone does not change domain string.
5. Discovery endpoint (`/api/v1/beta/discovery`) extended to return enumerations + deprecated flags enabling client pre-flight validation.
6. Metrics: add counter `taxonomy_extension_events_total` incremented when server boots with new enumerations compared to previous registry snapshot; gauge `taxonomy_deprecated_active_total` for deprecated in-use counts.
7. Security gating: values referencing prohibited tech (e.g. blockchain_operator) rejected outright to align with exclusion policies.

## Alternatives Considered
- Free-form strings (Rejected: undermines interop & audit consistency)
- Versioned taxonomy schema per POA (Deferred: complexity not warranted yet)

## Consequences
Positive:
- Predictable digest stability
- Operational visibility for extension events
Negative:
- Requires governance overhead (review cycle)

## Implementation Steps
1. Add deprecated map & validation hook.
2. Discovery endpoint enumeration expansion.
3. Metrics instrumentation additions.
4. Documentation update (this ADR, compliance assessment cross-link).
5. Add tests covering deprecated acceptance and rejection of prohibited terms.

## Rollback Plan
If enumeration addition causes issues, mark value deprecated immediately; avoid removal until next major version. Clients relying on new term may need patch instructions.

## References
`pkg/rfc0111/taxonomy.go`, `web/discovery_endpoint.go`
