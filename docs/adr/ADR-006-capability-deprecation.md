# ADR-006: Capability Deprecation and Rollover Policy

**Status**: Accepted  
**Date**: 2025-12-27  
**Authors**: AgentAuth Security Team  
**Stakeholders**: Platform Team, Security Engineering, Compliance

---

## Context

The AgentAuth capability registry allows service providers to register cryptographic capabilities (e.g., signature algorithms, key IDs) for use in PoA token issuance. Over time, capabilities may need to be deprecated due to:
- **Algorithm deprecation** (e.g., RSA-2048 → RSA-4096, SHA-256 → SHA-512)
- **Key rotation** (scheduled or emergency)
- **Security vulnerabilities** (e.g., algorithm breaks, key compromise)
- **Compliance changes** (regulatory requirements for stronger cryptography)

Without a systematic deprecation and rollover policy, we risk:
- Breaking existing PoA tokens mid-lifecycle
- Security vulnerabilities from using deprecated capabilities
- Compliance violations from outdated cryptographic standards

---

## Decision

We adopt a **phased deprecation and rollover policy** with explicit grace periods and sunset timelines.

### Capability Lifecycle States

1. **Active** (`active`): Capability is fully supported for new token issuance and validation
2. **Deprecated** (`deprecated`): Capability is discouraged for new tokens but still validates existing tokens
3. **Sunset** (`sunset`): Capability only validates existing tokens; new issuance rejected
4. **Retired** (`retired`): Capability fully retired; validation fails (emergency override only)

### Deprecation Process

#### Normal Deprecation (Planned)
1. **Announcement** (T-90 days):
   - Publish deprecation notice in capability registry metadata
   - Notify all service providers via API webhook + email
   - Update discovery endpoint with deprecation timeline

2. **Deprecation Phase** (T-90 to T-30 days):
   - Mark capability as `deprecated` in registry
   - New token issuance receives warning (not blocked)
   - Existing tokens continue to validate normally
   - Monitoring alerts on deprecated capability usage

3. **Sunset Phase** (T-30 to T-0 days):
   - Mark capability as `sunset`
   - New token issuance **blocked** with error code `capability_sunset`
   - Existing tokens continue to validate until expiration
   - Grace period accounts for longest-lived PoA token (typically 30 days)

4. **Retirement** (T+0):
   - Mark capability as `retired`
   - All validation fails (except emergency override with audit log)
   - Capability can be purged from registry after retention period

#### Emergency Deprecation (Security Incident)
1. **Immediate Sunset** (T-0):
   - Skip deprecation phase entirely
   - Immediately move to `sunset` or `retired` based on severity
   - Broadcast emergency notification via all channels

2. **Grace Period** (0-7 days):
   - For critical vulnerabilities: 0-day grace period (immediate retirement)
   - For high severity: 7-day grace period (sunset → retired)
   - All service providers must rotate to new capability

3. **Emergency Override**:
   - Admin API endpoint: `POST /api/v1/admin/capabilities/{id}/emergency-override`
   - Temporarily allows validation for specific tokens during incident response
   - Requires multi-party approval + audit log entry

### Rollover Process

#### Scheduled Rollover (e.g., Annual Key Rotation)
1. **Pre-Rollover** (T-60 days):
   - Generate new capability (e.g., new key ID)
   - Register in capability registry as `active`
   - Publish rollover schedule via discovery endpoint

2. **Dual-Operation Phase** (T-60 to T-0 days):
   - Both old and new capabilities are `active`
   - Service providers gradually migrate token issuance to new capability
   - Validation accepts both old and new signatures

3. **Deprecation of Old Capability** (T-0):
   - Follow normal deprecation process for old capability
   - New capability becomes primary

#### Emergency Rollover (Key Compromise)
1. **Immediate Rollover**:
   - Generate and register new capability immediately
   - Old capability marked `retired` with 0-day grace period
   - All service providers notified to rotate immediately

2. **Incident Response**:
   - Root cause analysis documented in ADR
   - Post-mortem includes rollover timeline
   - Update key management procedures if needed

---

## Consequences

### Positive
- **Predictability**: Service providers have 90+ days notice for planned deprecations
- **Security**: Emergency deprecation process enables rapid response to vulnerabilities
- **Compliance**: Automated enforcement of cryptographic standards
- **Auditability**: All deprecations logged with justification and timeline

### Negative
- **Operational Overhead**: Requires monitoring for deprecated capability usage
- **Client Coordination**: Service providers must implement capability refresh logic
- **Grace Period Management**: Needs careful calculation based on token TTL

### Mitigations
- **Automated Monitoring**: Alerts when deprecated capabilities exceed usage threshold
- **Client SDK Support**: Provide SDK helpers for automatic capability refresh
- **Discovery Endpoint**: Publish deprecation timelines for client planning

---

## Implementation

### Capability Registry Schema Enhancement
```json
{
  "id": "cap_abc123",
  "algorithm": "ES384",
  "key_id": "2024-Q4-prod",
  "status": "deprecated",
  "lifecycle": {
    "created_at": "2024-10-01T00:00:00Z",
    "deprecated_at": "2025-01-01T00:00:00Z",
    "sunset_at": "2025-03-01T00:00:00Z",
    "retired_at": "2025-04-01T00:00:00Z"
  },
  "deprecation_reason": "algorithm_upgrade",
  "replacement_capability_id": "cap_xyz789"
}
```

### API Endpoints
- `GET /api/v1/capabilities?status=deprecated` - List deprecated capabilities
- `POST /api/v1/admin/capabilities/{id}/deprecate` - Initiate deprecation
- `POST /api/v1/admin/capabilities/{id}/emergency-retire` - Emergency retirement
- Discovery endpoint includes `capability_lifecycle` section

### Migration Playbook
See [docs/playbooks/capability-rollover.md](../playbooks/capability-rollover.md) (to be created) for step-by-step rollover procedures.

---

## References
- [Capability Registry Implementation](../../internal/capability/registry.go)
- [AAP-001 § Capability Management](https://example.com/AAP-001#capabilities)
- [NIST SP 800-57: Key Management Guidelines](https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final)

---

## Review History
- **2025-12-27**: Initial ADR approved
- **Future**: Quarterly reviews by Security Engineering team
