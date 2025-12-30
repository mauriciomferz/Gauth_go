---
title: Lifecycle Status
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Lifecycle Status Model

This document defines the lifecycle status taxonomy and transition semantics for AgentAuth delegations (Power of Attorney) and internal tokens.

## Delegation Statuses

Supported values:
- active: Delegation is usable for authorization decisions and validation.
- suspended: Temporarily disabled; may be reactivated or terminated. Validation treats suspended delegations as revoked for denial semantics.
- terminated: Permanently disabled; terminal state. Cannot transition back to active or suspended.
- revoked (implicit via revocation chain): Represented separately by revocation events; when a revocation is recorded the delegation status is synchronized to revoked.
- expired (derived): Set automatically when ValidUntil timestamp passes during validation.
- pending (future use): Issued but not yet active (e.g., start date in future).

### Transition Rules (Prototype)
| From       | To          | Allowed | Notes |
|------------|-------------|---------|-------|
| active     | suspended   | yes     | Operational pause |
| active     | terminated  | yes     | Immediate permanent disable |
| active     | revoked     | via revocation chain | Use RevokeDelegation API |
| active     | expired     | automatic | Time-based transition |
| suspended  | active      | yes     | Reactivation |
| suspended  | terminated  | yes     | Permanent disable |
| suspended  | revoked     | via revocation chain | |
| terminated | any other   | no      | Terminal state |
| revoked    | any other   | no      | Terminal state |
| expired    | any other   | no      | Terminal state |

Validation currently treats any non-active status as denied with an RFC error (revoked or expired). Future versions will differentiate suspended vs terminated errors.

### Enforcement Points
- Token issuance: Sets initial Status=active.
- Validation path (`ValidateDelegationCtx`): Converts expired -> expired status and denies suspended/terminated/revoked.
- Revocation path (`RevokeDelegationCtx`): Sets Status=revoked and appends revocation chain event.
- Status update endpoint (prototype): Allows active<->suspended and active/suspended->terminated transitions; enforces terminal immutability of terminated.

## Token Statuses (Internal Token Store)
Supported values:
- active
- suspended
- terminated

Additional derived statuses:
- revoked (RevokedAt timestamp set)
- expired (ExpiresAt passed)

### Token Transition Rules
| From      | To         | Allowed |
|-----------|------------|---------|
| active    | suspended  | yes |
| active    | terminated | yes |
| suspended | active     | yes |
| suspended | terminated | yes |
| terminated| any other  | no |

Revocation sets RevokedAt and implicitly treats token as revoked regardless of Status.

## Metrics
New counters added (memory & Prometheus):
- multi_signature_weight_failures_total: Cumulative weight shortfall under weighted threshold semantics.
- delegation_status_transitions_total / delegation_status_transition_failures_total (reserved for future when server integrates persistent RFC0111 service).
- token_status_transitions_total / token_status_transition_failures_total (reserved; prototype audit-only implementation).

## Weighted Multi-Signature Threshold
If `GAUTH_MULTI_SIG_WEIGHTS` is set (format signer=weight,...), the multi-signature verification path interprets `Threshold` as required cumulative `weight` of valid signatures instead of raw count. Failures increment `multi_signature_weight_failures_total`.

## Future Enhancements
- Persistent delegation status updates wired to RFC0111 service repository.
- Distinct error codes for suspended vs terminated.
- Administrative audit trail enrichment (actor roles & reasons).
- Bulk status transition endpoint with concurrency safeguards.
- Formal schema version bump when pending status enters enforcement.
