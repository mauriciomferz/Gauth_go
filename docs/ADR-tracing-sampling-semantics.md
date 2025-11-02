# ADR: Tracing Sampling Semantics (RB9 Phase 1)

Status: Proposed (Implemented)
Date: 2025-10-27
Decision Owners: Observability Working Group / Sprint 2 Team

## Context
Phase RB9 introduced in-repo tracing spans for core flows (`token.issue`, `token.validate`, `attestation.verify`, `rotation.perform`). A sampling control was required to:
- Limit overhead in production.
- Allow full capture in development/testing.
- Provide deterministic behavior for unit tests.

Environment variables:
- `GAUTH_TRACING_ENABLED` (new primary flag) and legacy `GAUTH_OTEL_ENABLE` (backward compatibility).
- `GAUTH_TRACING_SAMPLE_RATIO` numeric string in [0,1].

## Decision Drivers
1. Simplicity: Single float ratio without multi-parameter configuration.
2. Low friction for dev: Full sampling by default (unset ratio -> 1.0).
3. Backward compatibility: Preserve `GAUTH_OTEL_ENABLE` for existing deployments.
4. Testability: Ability to force deterministic span presence.
5. Performance: Avoid allocation when spans are skipped.

## Options Considered
| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| A | Conventional semantics (ratio=0 => no spans; ratio=1 => all spans) | Familiar, intuitive | Needs explicit handling for disabled state beyond env flag; zero value ambiguous if unset vs explicit 0 |
| B | Current implemented semantics (ratio <= 0 => always sample; ratio in (0,1] probabilistic) | Simplifies dev defaults; tests use `0` for deterministic full sampling | Counterintuitive for operators; surprises when trying to disable via setting 0 |
| C | Dual env approach (`ENABLED` boolean + `SAMPLE_RATIO` conventional) | Clear separation of enable vs sample rate | More configuration surface; added complexity |
| D | Token bucket / rate-per-second sampling | Precise control under burst load | Over-engineered for Phase 1; increases code complexity |

## Decision Outcome
Selected Option B (ratio <= 0 interpreted as "always sample"). This reduces initial complexity and allows using `GAUTH_TRACING_SAMPLE_RATIO=0` in tests to guarantee span emission. The disabled state is governed solely by absence of `GAUTH_TRACING_ENABLED`/`GAUTH_OTEL_ENABLE`.

## Implementation Summary
Code snippet (conceptual):
```go
if tracerProvider != nil && (sampleRatio <= 0 || rand.Float64() < sampleRatio) {
    _, span = tracerProvider.StartSpan(ctx, "token.issue")
}
```
- Ratio parsing clamps to [0,1]; invalid values ignored (default 1.0).
- Spans hold minimal tags to reduce cardinality.
- Rotation tracing provider wired to same instance for consolidated capture.

## Consequences
### Positive
- Straightforward deterministic testing with ratio=0.
- Minimal branching logic.
- Reduced accidental partial sampling during early development.

### Negative / Risks
- Operators might intuitively set ratio=0 expecting to disable sampling; they instead enable full sampling.
- Future exporter integration may require user education or semantic switch.

## Mitigations & Future Work
- Documentation: Explicit note in `OBSERVABILITY.md` highlighting current semantics and pending change consideration.
- Transition Plan (if semantics inverted later):
  1. Introduce `GAUTH_TRACING_SAMPLE_MODE` with values `legacy` (current) / `standard`.
  2. Default new deployments to `standard` (0 => no sample) while honoring legacy when unset.
  3. After 2 sprints, deprecate `legacy` mode.
- Add mid-ratio statistical test (0.5) with multi-iteration to assert distribution before any semantic change.

## Alternatives Rejected
- Immediate adoption of conventional semantics (Option A) due to desire for guaranteed deterministic unit test sampling without additional enable flags.
- External configurable policy engine for sampling (too heavy for Phase 1).

## Security & Integrity Considerations
- Sampling choice does not alter correctness of token, attestation, or rotation logic.
- Span tags avoid sensitive values (no raw payloads, keys, or secrets stored).

## Operations
- To fully disable tracing: ensure neither `GAUTH_TRACING_ENABLED` nor `GAUTH_OTEL_ENABLE` are set.
- To force all spans for debugging: set either enable flag plus `GAUTH_TRACING_SAMPLE_RATIO=0`.

## Acceptance Criteria
- Unit tests: enabled (ratio=1) emits spans, disabled (no env) emits none, ratio=0 emits spans.
- Documentation updated with semantics.
- No performance degradation observed under disabled state (span gate short-circuits).

## Future Enhancements
- OTLP exporter bridge with resource attributes and W3C propagation.
- Configurable attribute filters for high cardinality prevention.
- Span buffering with size/time flush policy when exporter added.

## References
- Sprint 2 Plan (`SPRINT2_PLAN.md`) – RB9 section.
- `web/tracing_basic_test.go` – sampling tests.
- `server_clean.go` – tracer provider initialization and sampling gate.

---
