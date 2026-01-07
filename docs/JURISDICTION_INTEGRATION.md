---
title: Jurisdiction Integration
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Jurisdiction-Specific Runtime Enforcement Integration (P1.3)

## Overview

P1.3 integrates the existing jurisdiction enforcement infrastructure (`internal/jurisdiction/`) with AAP-001 token operations (delegation creation and token verification). This enables **opt-in** compliance with GDPR, CCPA, cross-border data transfer rules, data residency requirements, and jurisdiction-specific blocked actions at the token level.

**Key Features:**
- ✅ **GDPR Consent Validation** (EU): Requires `gdpr_consent:true` for data processing actions
- ✅ **CCPA Opt-Out Enforcement** (US): Denies data processing when `ccpa_opt_out:true`
- ✅ **Cross-Border Data Transfer Rules**: EU personal data restricted to adequacy countries (UK only)
- ✅ **Data Residency Enforcement**: EU personal/health data must stay in EU jurisdiction
- ✅ **Blocked Actions**: Jurisdiction-specific action blocks (e.g., EU blocks `unrestricted_data_export`)
- ✅ **Opt-In Design**: Backward compatible, disabled by default

## Architecture

### Integration Points

```
CreateDelegationCtx()                    VerifyToken()
        |                                       |
        v                                       v
+-------+-------+                    +----------+---------+
| Jurisdiction  |                    | Jurisdiction       |
| Enforcement   |                    | Enforcement        |
| (Issuance)    |                    | (Verification)     |
+-------+-------+                    +----------+---------+
        |                                       |
        v                                       v
   EnforceJurisdiction()              EnforceJurisdiction()
        |                                       |
        +---------------+---------------+
                        |
                        v
            +---------------------+
            | EnforcementEngine   |
            | - GDPR rules        |
            | - CCPA rules        |
            | - Cross-border      |
            | - Data residency    |
            | - Blocked actions   |
            +---------------------+
```

### Files Added/Modified

1. **pkg/aap001/jurisdiction_integration.go** (NEW, 265 lines)
   - `WithJurisdictionEnforcement()` option
   - `enforceJurisdictionOnIssuance()` - gates delegation creation
   - `enforceJurisdictionOnVerification()` - validates token usage
   - `ExtractJurisdictionFromPOA()` - jurisdiction extraction helper
   - `ValidateJurisdictionCompliance()` - standalone validation

2. **pkg/aap001/jurisdiction_integration_test.go** (NEW, 380 lines)
   - 9 comprehensive integration tests
   - EU GDPR consent scenarios
   - US CCPA opt-out scenarios
   - Cross-border transfer validation
   - Data residency enforcement
   - Blocked actions testing

3. **pkg/aap001/aap001.go** (MODIFIED)
   - Added `jurisdictionEnforcement *JurisdictionEnforcement` field to `Service` struct (line 1288)
   - Added `Claims map[string]interface{}` field to `DelegationRequest` struct (line 459)
   - Added jurisdiction enforcement call in `CreateDelegationCtx()` (line 1747)
   - Added jurisdiction enforcement call in `VerifyToken()` (line 913)

## Usage

### 1. Enable Jurisdiction Enforcement

```go
import (
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/aap001"
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/internal/jurisdiction"
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/audit"
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/authz"
)

// Create jurisdiction enforcement integration
integration := jurisdiction.NewServerIntegration()

// Create AAP-001 service with jurisdiction enforcement
logger := audit.NewMemoryLogger(nil)
authorizer := authz.NewMemoryAuthorizer()

svc := aap001.NewService(
    logger,
    authorizer,
    aap001.WithJurisdictionEnforcement(integration),
)
```

### 2. EU GDPR Compliance Example

```go
// Create delegation requiring GDPR consent
req := aap001.DelegationRequest{
    Grantor:  "alice@eubank.com",
    Grantee:  "bob@eubank.com",
    Scope:    []string{"gdpr_data_processing"},
    Duration: 1 * time.Hour,
    Claims: map[string]interface{}{
        "jurisdiction":  "EU",
        "entity_type":   "corporation",
        "gdpr_consent":  true, // Required for EU data processing
    },
}

resp, err := svc.CreateDelegationCtx(ctx, req)
if err != nil {
    // Denied if gdpr_consent is false or missing
    log.Fatalf("GDPR enforcement denied: %v", err)
}
```

### 3. US CCPA Opt-Out Example

```go
// Create delegation with CCPA opt-out check
req := aap001.DelegationRequest{
    Grantor:  "alice@usbank.com",
    Grantee:  "bob@usbank.com",
    Scope:    []string{"ccpa_data_processing"},
    Duration: 1 * time.Hour,
    Claims: map[string]interface{}{
        "jurisdiction": "US",
        "entity_type":  "corporation",
        "ccpa_opt_out": false, // If true, processing is denied
    },
}

resp, err := svc.CreateDelegationCtx(ctx, req)
if err != nil {
    // Denied if ccpa_opt_out is true
    log.Fatalf("CCPA opt-out enforced: %v", err)
}
```

### 4. Cross-Border Data Transfer Example

```go
// EU to UK cross-border transfer (adequacy country)
req := aap001.DelegationRequest{
    Grantor:  "alice@eubank.com",
    Grantee:  "bob@ukbank.com",
    Scope:    []string{"personal_data_transfer"},
    Duration: 1 * time.Hour,
    Claims: map[string]interface{}{
        "jurisdiction":             "EU",
        "entity_type":              "corporation",
        "destination_jurisdiction": "UK", // UK is adequacy country
    },
}

resp, err := svc.CreateDelegationCtx(ctx, req)
// ✅ ALLOWED: UK is in EU adequacy list

// EU to US cross-border transfer (NOT adequacy country)
reqBlocked := req
reqBlocked.Claims["destination_jurisdiction"] = "US"

_, err = svc.CreateDelegationCtx(ctx, reqBlocked)
// ❌ DENIED: US not in EU adequacy list for personal data
```

### 5. Data Residency Enforcement Example

```go
// EU personal data must stay in EU
req := aap001.DelegationRequest{
    Grantor:  "alice@eubank.com",
    Grantee:  "bob@usbank.com",
    Scope:    []string{"data_export"},
    Duration: 1 * time.Hour,
    Claims: map[string]interface{}{
        "jurisdiction":             "EU",
        "entity_type":              "corporation",
        "destination_jurisdiction": "US",           // Leaving EU
        "data_type":                "personal_data", // Residency rule applies
    },
}

_, err := svc.CreateDelegationCtx(ctx, req)
// ❌ DENIED: Data residency violation (EU personal data leaving EU)
```

### 6. Blocked Actions Example

```go
// EU blocks "unrestricted_data_export" action
req := aap001.DelegationRequest{
    Grantor:  "alice@eubank.com",
    Grantee:  "bob@eubank.com",
    Scope:    []string{"unrestricted_data_export"}, // Blocked in EU
    Duration: 1 * time.Hour,
    Claims: map[string]interface{}{
        "jurisdiction": "EU",
        "entity_type":  "corporation",
    },
}

_, err := svc.CreateDelegationCtx(ctx, req)
// ❌ DENIED: Action blocked in EU jurisdiction
```

## Supported Jurisdictions

| Jurisdiction | Strict Mode | GDPR/CCPA | Cross-Border Rules | Data Residency | Key Blocked Actions |
|---|---|---|---|---|---|
| **EU** | ✅ | GDPR consent required | Personal: EU, UK only | Personal/health must stay in EU | unrestricted_data_export, automated_profiling, bulk_data_transfer |
| **US** | ❌ | CCPA opt-out enforced | Permissive (all major) | No restrictions | autonomous_high_risk_decision |
| **UK** | ✅ | UK GDPR | Personal: UK, EU only | Personal/health must stay in UK | unrestricted_data_export |
| **CA** | ❌ | PIPEDA | Permissive | Health data local only | - |
| **AU** | ❌ | Privacy Act | Permissive | No restrictions | - |
| **JP** | ✅ | APPI | Restrictive (whitelist) | Personal/health must stay in JP | - |

## Configuration

### Environment Variables

Jurisdiction enforcement can be configured via external rules:

```bash
# Point to external jurisdiction rules JSON file
export AGENTAUTH_JURISDICTION_RULES_PATH=/path/to/jurisdiction_rules.json
```

### jurisdiction_rules.json Example

```json
{
  "jurisdictions": [
    {
      "jurisdiction": "EU",
      "strict_mode": true,
      "allowed_actions": ["transfer", "pay"],
      "blocked_actions": ["unrestricted_data_export"],
      "cross_border_rules": {
        "personal_data_transfer": ["EU", "UNITED_KINGDOM"],
        "financial_data_export": ["EU", "UK", "US"]
      },
      "data_residency_rules": {
        "personal_data": true,
        "health_data": true,
        "financial_data": false
      }
    }
  ]
}
```

### Default Behavior

- **Enforcement Disabled**: By default, jurisdiction enforcement is **disabled** (backward compatible)
- **Opt-In Activation**: Enable by passing `WithJurisdictionEnforcement()` option to `NewService()`
- **Fail-Open on Errors**: If enforcement engine fails, errors are logged but requests proceed (fail-open behavior)

## Migration Guide

### Phase 1: Assessment (Week 1)

1. **Review Compliance Requirements**
   - Identify jurisdictions where your application operates
   - Document GDPR, CCPA, and other regulatory requirements
   - Map business actions to compliance rules

2. **Analyze Token Flows**
   - Identify delegation creation points requiring jurisdiction enforcement
   - Determine which token verification points need runtime validation
   - Audit existing jurisdiction field usage in PowerOfAttorney objects

### Phase 2: Pilot (Weeks 2-3)

1. **Enable Enforcement in Test Environment**
   ```go
   integration := jurisdiction.NewServerIntegration()
   svc := aap001.NewService(logger, authz, 
       aap001.WithJurisdictionEnforcement(integration),
   )
   ```

2. **Add Jurisdiction Claims to Delegation Requests**
   ```go
   req.Claims = map[string]interface{}{
       "jurisdiction": "EU",
       "entity_type": "corporation",
       "gdpr_consent": true,
   }
   ```

3. **Monitor Enforcement Metrics**
   - Track allowed vs denied operations
   - Identify unexpected enforcement failures
   - Validate compliance rules accuracy

### Phase 3: Production Rollout (Weeks 4-6)

1. **Gradual Enablement**
   - Enable jurisdiction enforcement for non-critical operations first
   - Monitor error rates and user impact
   - Adjust rules based on real-world data

2. **Full Activation**
   - Enable enforcement for all delegation operations
   - Configure external jurisdiction rules via `AGENTAUTH_JURISDICTION_RULES_PATH`
   - Set up alerting for enforcement denials

3. **Documentation & Training**
   - Update API documentation with jurisdiction requirements
   - Train development teams on compliance rules
   - Provide troubleshooting guides for common enforcement failures

## Testing

### Running Integration Tests

```bash
cd <repo-root>
go test -v ./pkg/aap001 -run "TestJurisdiction"
```

### Test Coverage

- ✅ Enforcement disabled by default (backward compatibility)
- ✅ EU GDPR consent validation (allow/deny)
- ✅ US CCPA opt-out enforcement (allow/deny)
- ✅ Cross-border data transfer (EU→UK allow, EU→US deny)
- ✅ Data residency enforcement (EU personal data leaving EU denied)
- ✅ Blocked actions (EU unrestricted_data_export denied)
- ✅ Token verification enforcement
- ✅ Jurisdiction extraction helpers
- ✅ Standalone compliance validation

### Test Results (All Passing)

```
=== RUN   TestJurisdictionIntegration_Disabled
--- PASS: TestJurisdictionIntegration_Disabled (0.00s)
=== RUN   TestJurisdictionIntegration_EUGDPRConsent
--- PASS: TestJurisdictionIntegration_EUGDPRConsent (0.00s)
=== RUN   TestJurisdictionIntegration_USCCPAOptOut
--- PASS: TestJurisdictionIntegration_USCCPAOptOut (0.00s)
=== RUN   TestJurisdictionIntegration_CrossBorderDataTransfer
--- PASS: TestJurisdictionIntegration_CrossBorderDataTransfer (0.00s)
=== RUN   TestJurisdictionIntegration_DataResidency
--- PASS: TestJurisdictionIntegration_DataResidency (0.00s)
=== RUN   TestJurisdictionIntegration_BlockedActions
--- PASS: TestJurisdictionIntegration_BlockedActions (0.00s)
=== RUN   TestJurisdictionIntegration_VerifyTokenEnforcement
--- PASS: TestJurisdictionIntegration_VerifyTokenEnforcement (0.00s)
=== RUN   TestExtractJurisdictionFromPOA
--- PASS: TestExtractJurisdictionFromPOA (0.00s)
=== RUN   TestValidateJurisdictionCompliance
--- PASS: TestValidateJurisdictionCompliance (0.00s)
PASS
```

## Security Considerations

### 1. Fail-Closed vs Fail-Open

**Current Behavior**: Fail-open (enforcement errors logged but requests proceed)

**Recommendation**: Consider fail-closed mode for production:
```go
// TODO(P1.3.1): Add fail-closed configuration option
// When enabled, enforcement errors deny requests instead of logging warnings
```

### 2. Jurisdiction Spoofing

- Claims-based jurisdiction detection is vulnerable to spoofing
- **Mitigation**: Validate jurisdiction against trusted identity providers or IP geolocation
- **Future Enhancement**: Add jurisdiction verification against external authority

### 3. Enforcement Bypass

- Jurisdiction enforcement is opt-in (disabled by default)
- **Mitigation**: Mandate enforcement in security-sensitive environments via deployment policy
- **Recommendation**: Add runtime enforcement status metrics

### 4. Cross-Border Evasion

- Attackers may route data through adequacy countries to bypass restrictions
- **Mitigation**: Log all cross-border transfers for audit trail
- **Recommendation**: Add destination chain validation (detect multi-hop transfers)

## API Reference

### `WithJurisdictionEnforcement(integration *jurisdiction.ServerIntegration) Option`

Enables jurisdiction enforcement for AAP-001 Service operations.

**Parameters:**
- `integration`: ServerIntegration instance from `internal/jurisdiction`

**Returns:** Service configuration option

**Example:**
```go
integration := jurisdiction.NewServerIntegration()
svc := aap001.NewService(logger, authz, aap001.WithJurisdictionEnforcement(integration))
```

### `ExtractJurisdictionFromPOA(poa *PowerOfAttorney) compliance.Jurisdiction`

Extracts jurisdiction from PowerOfAttorney object.

**Parameters:**
- `poa`: PowerOfAttorney object (can be nil)

**Returns:** Jurisdiction enum (defaults to `JurisdictionUS`)

**Priority:**
1. `poa.Jurisdiction` field
2. `poa.Restrictions["jurisdiction"]` map value
3. Default: `compliance.JurisdictionUS`

### `ValidateJurisdictionCompliance(ctx, poa) error`

Standalone validation of PowerOfAttorney jurisdiction compliance (without creating/verifying tokens).

**Parameters:**
- `ctx`: Context
- `poa`: PowerOfAttorney to validate

**Returns:** nil if compliant, error with violation details otherwise

**Example:**
```go
if err := svc.ValidateJurisdictionCompliance(ctx, poa); err != nil {
    log.Printf("Compliance violation: %v", err)
}
```

## Troubleshooting

### Enforcement Denials

**Symptom**: Delegation creation fails with "jurisdiction enforcement denied"

**Diagnosis:**
1. Check enforcement decision violations in error message
2. Verify jurisdiction claim matches expected value
3. Confirm GDPR consent/CCPA opt-out fields present
4. Validate destination_jurisdiction for cross-border operations
5. Check data_type for residency rules

**Resolution:**
- Ensure required claims present in `DelegationRequest.Claims`
- Verify jurisdiction rules via `AGENTAUTH_JURISDICTION_RULES_PATH` configuration
- Check enforcement engine logs for detailed denial reasons

### Missing Jurisdiction Data

**Symptom**: Enforcement uses default US jurisdiction unexpectedly

**Diagnosis:**
1. PowerOfAttorney.Jurisdiction field empty
2. DelegationRequest.Claims["jurisdiction"] missing
3. Restrictions map lacks jurisdiction key

**Resolution:**
- Set `poa.Jurisdiction` explicitly
- Add "jurisdiction" to request claims
- Configure jurisdiction in restrictions map

### Performance Impact

**Symptom**: Delegation creation slower after enabling enforcement

**Diagnosis:**
- Jurisdiction enforcement adds 1-5ms latency per operation
- Complex cross-border rules may increase latency

**Resolution:**
- Monitor enforcement latency via metrics
- Use caching for repeated enforcement decisions
- Consider async enforcement for non-critical operations

## Metrics (TODO: Requires metrics interface extension)

Future metrics to be added:

```go
// Enforcement counters
s.metrics.IncJurisdictionEnforcementAllows()  // Successful enforcements
s.metrics.IncJurisdictionEnforcementDenials() // Denied operations
s.metrics.IncJurisdictionEnforcementErrors()  // Engine errors

// Jurisdiction breakdown
s.metrics.ObserveJurisdictionEnforcementLatency(d time.Duration)
s.metrics.IncJurisdictionEnforcementByType(jurisdiction string)
```

## Roadmap

### P1.3.1: Enhanced Enforcement (Future)

- [ ] Fail-closed configuration option
- [ ] Jurisdiction verification against external authority
- [ ] Multi-hop cross-border transfer detection
- [ ] Enforcement decision caching
- [ ] Jurisdiction metrics integration

### P1.3.2: Additional Jurisdictions (Future)

- [ ] Singapore (PDPA)
- [ ] Brazil (LGPD)
- [ ] China (PIPL)
- [ ] India (DPDPA)

### P1.3.3: Advanced Compliance (Future)

- [ ] Automated compliance reporting
- [ ] Jurisdiction-specific audit trails
- [ ] Real-time enforcement alerts
- [ ] Compliance dashboard UI

## References

- **P1 Implementation Plan**: `docs/P1_IMPLEMENTATION_PLAN.md` (lines 115-195)
- **Jurisdiction Infrastructure**: `internal/jurisdiction/enforcement.go`
- **Existing Documentation**: `JURISDICTION_IMPLEMENTATION_SUMMARY.md`
- **GAP Matrix**: sec4.item1 (Jurisdiction-specific enforcement)

## License

Same as AgentAuth project (see root LICENSE file)
