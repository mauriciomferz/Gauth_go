# AgentAuth Hygiene Metrics

AgentAuth implements strict string hygiene for all delegation and token fields to prevent injection attacks and ensure interoperability across diverse clients.

## Metrics Overview

The following metrics are tracked in-memory and exported via Prometheus/OpenTelemetry:

| Metric Name | Category | Description |
| ----------- | -------- | ----------- |
| `gauth_violation_scope_utf8_invalid_total` | Hygiene | Incremented when a `scope` entry contains invalid UTF-8 sequences. |
| `gauth_violation_scope_control_char_total` | Hygiene | Incremented when a `scope` entry contains ASCII control characters (0x00-0x1F, 0x7F). |
| `gauth_violation_restriction_utf8_invalid_total` | Hygiene | Incremented when a `restriction` key or value contains invalid UTF-8. |
| `gauth_violation_restriction_control_char_total` | Hygiene | Incremented when a `restriction` key or value contains ASCII control characters. |

## Enforcement

Hygiene checks are performed during:
1.  **Delegation Creation**: `CreateDelegation` calls reject requests with `invalid_request` if any hygiene rule is violated.
2.  **Token Verification**: `VerifyToken` checks the hygiene of claims extracted from the envelope.

## Anchoring

To ensure the integrity of these security indicators, a snapshot of the violation counters is periodically included in the **System Integrity Anchor**. This prevents an attacker from masking a surge in violations by tampering with the metrics export.

## Remediation

Surges in hygiene violations typically indicate:
-   Malformed client implementations.
-   Attempted injection attacks (e.g., trying to sneak control characters into scopes).
-   Encoding mismatches between systems.

Monitoring systems should alert on high rates of these violations.
