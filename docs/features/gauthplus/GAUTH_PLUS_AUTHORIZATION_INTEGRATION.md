# GAuth+ Integration with Authorization Chain

## Overview

This document describes how GAuth+ features (successor management, AI-to-AI delegation, dual control approvals, capability assessment, and fiduciary duty tracking) are integrated into the RFC-0111 authorization chain validation process.

## Architecture

The GAuth+ integration adds an additional layer of policy enforcement on top of the existing RFC-0111 authorization chain validation. When enabled, GAuth+ policies are checked during:

1. **Request Compliance Validation** (`ComplianceValidator.ValidateRequestCompliance`)
2. **Grant Compliance Validation** (`ComplianceValidator.ValidateGrantCompliance`)
3. **PDP Authorization Decisions** (`SimplePDP.MakeDecision`)

### Key Components

#### 1. GAuthPlusValidator (`pkg/gauth/gauthplus_integration.go`)

Central integration point that coordinates validation across all GAuth+ services:

- **PostgreSQLSuccessorService**: Checks for active successor AI takeovers
- **PostgreSQLDelegationService**: Validates AI-to-AI delegation chains
- **PostgreSQLDualControlService**: Verifies multi-approver requirements
- **PostgreSQLCapabilityAssessmentService**: Ensures AI meets capability requirements
- **PostgreSQLFiduciaryDutyService**: Checks for unresolved fiduciary violations

**Main Method**:
```go
func (v *GAuthPlusValidator) ValidatePoAWithGAuthPlus(
    ctx context.Context,
    poaID string,
    poaDef *poa.PoADefinition,
    agentID string,
    actionType string,
) (*GAuthPlusValidationResult, error)
```

**Validation Flow**:
1. Check if successor AI is active → Use successor agent ID for subsequent checks
2. Validate delegation chain depth and policies
3. Check dual control approval requirements (if enabled)
4. Verify AI capability meets requirements (if enabled)
5. Check for critical fiduciary violations (if enabled)

#### 2. ComplianceValidator Extensions

**Added Fields**:
```go
type ComplianceValidator struct {
    chainValidator     *AuthorizationChainValidator
    gauthPlusValidator *GAuthPlusValidator  // NEW
    pipClient          PIPClient
    pdpClient          PDPClient
    strictMode         bool
    enforceGAuthPlus   bool                 // NEW
}
```

**Setup Methods**:
```go
// Enable GAuth+ enforcement
validator.SetGAuthPlusValidator(gauthPlusValidator)
validator.SetEnforceGAuthPlus(true)
```

**Integration Points**:
- `ValidateRequestCompliance`: Step 4a calls `validatePoAWithGAuthPlus`
- `ValidateGrantCompliance`: Step 5a calls `validatePoAWithGAuthPlus`
- Results include `GAuthPlusValidation *GAuthPlusValidationResult`

#### 3. SimplePDP Extensions

**Added Fields**:
```go
type SimplePDP struct {
    pap                *PowerAdministrationPoint
    gauthPlusValidator *GAuthPlusValidator  // NEW
    enforceGAuthPlus   bool                 // NEW
}
```

**Setup Methods**:
```go
// Enable GAuth+ in PDP
pdp.SetGAuthPlusValidator(gauthPlusValidator)
pdp.SetEnforceGAuthPlus(true)
```

**Integration Point**:
- `evaluateRequest`: Step 3 validates GAuth+ policies before checking action authorization

## Configuration

### Basic Setup

```go
import (
    "database/sql"
    "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
    "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauthplus"
)

// Initialize GAuth+ services
successorService := gauthplus.NewPostgreSQLSuccessorService(db)
delegationService := gauthplus.NewPostgreSQLDelegationService(db)
dualControlService := gauthplus.NewPostgreSQLDualControlService(db)
fiduciaryService := gauthplus.NewPostgreSQLFiduciaryDutyService(db)
capabilityService := gauthplus.NewPostgreSQLCapabilityAssessmentService(db)

// Create GAuth+ validator
gauthPlusValidator := gauth.NewGAuthPlusValidator(
    successorService,
    delegationService,
    dualControlService,
    fiduciaryService,
    capabilityService,
)

// Configure enforcement levels
gauthPlusValidator.SetEnforceCapabilities(true)   // Enforce capability requirements
gauthPlusValidator.SetEnforceDualControl(true)    // Enforce dual control approvals
gauthPlusValidator.SetEnforceFiduciary(true)      // Block on critical violations

// Integrate with ComplianceValidator
complianceValidator.SetGAuthPlusValidator(gauthPlusValidator)
complianceValidator.SetEnforceGAuthPlus(true)

// Integrate with PDP
pdp.SetGAuthPlusValidator(gauthPlusValidator)
pdp.SetEnforceGAuthPlus(true)
```

### Enforcement Modes

**1. Fully Permissive (Default)**
```go
validator.SetEnforceGAuthPlus(false)
// GAuth+ checks are skipped entirely
```

**2. Advisory Mode**
```go
validator.SetEnforceGAuthPlus(true)
validator.SetEnforceCapabilities(false)
validator.SetEnforceDualControl(false)
validator.SetEnforceFiduciary(false)
// GAuth+ checks run but only produce warnings, don't block authorization
```

**3. Strict Mode**
```go
validator.SetEnforceGAuthPlus(true)
validator.SetEnforceCapabilities(true)
validator.SetEnforceDualControl(true)
validator.SetEnforceFiduciary(true)
// All GAuth+ policy violations block authorization
```

**4. Custom Mode**
```go
validator.SetEnforceGAuthPlus(true)
validator.SetEnforceCapabilities(true)   // Block on capability issues
validator.SetEnforceDualControl(false)   // Don't enforce dual control
validator.SetEnforceFiduciary(true)      // Block on fiduciary violations
```

## Validation Details

### 1. Successor Status Check

**Purpose**: Detect if primary AI has failed and successor has taken over

**Process**:
1. Query `successor_activations` table for active successor
2. If successor active, replace primary agent ID with successor agent ID
3. All subsequent checks use effective agent ID (successor or primary)

**Result Fields**:
```go
type SuccessorCheckResult struct {
    SuccessorActive  bool
    ActiveSuccessor  *SuccessorActivation
    EffectiveAgentID string  // Primary or successor agent ID
}
```

**Failure Conditions**: None (successor takeover is automatic, not a failure)

**Warnings**:
- "Successor AI {id} is active, taking over from primary AI {id}"

### 2. Delegation Chain Validation

**Purpose**: Ensure AI-to-AI delegations comply with depth limits and policies

**Process**:
1. Retrieve delegation chain from `ai_delegations` table (recursive query)
2. Check current depth against max allowed depth
3. Validate each delegation against source agent's policy
4. Verify scope restrictions and time validity

**Result Fields**:
```go
type DelegationCheckResult struct {
    DelegationValid   bool
    CurrentDepth      int
    MaxAllowedDepth   int
    DelegationChain   []*AIDelegation
    DepthExceeded     bool
}
```

**Failure Conditions**:
- Delegation depth exceeds maximum allowed
- Delegation violates source agent policy (can_delegate=false)
- Target agent not in allowed delegates list
- Scope exceeds source agent's authorized scope

**Warnings**:
- Individual delegation policy violations in chain

### 3. Dual Control Approval Check

**Purpose**: Ensure high-risk actions have required approvals

**Process**:
1. Query `dual_control_approvals` for matching PoA and action type
2. Check approval status (pending/approved/rejected/expired)
3. Count approvers against required threshold
4. Validate approval logic (all/majority/quorum/weighted)

**Result Fields**:
```go
type DualControlCheckResult struct {
    RequiresApproval  bool
    ApprovalObtained  bool
    PendingApproval   *DualControlApproval
    ApprovedAction    *DualControlApproval
    RequiredApprovers int
    CurrentApprovers  int
}
```

**Failure Conditions** (when enforcement enabled):
- Action requires approval but none obtained
- Approval threshold not met
- Approval expired

**Current Limitation**:
- Dual control checking requires additional service methods to query by PoA + action type
- Currently set to permissive (doesn't block) pending enhancement

### 4. Capability Assessment

**Purpose**: Verify AI meets minimum capability requirements

**Process**:
1. Retrieve latest assessment from `ai_capability_assessments`
2. Compare overall capability level (L0-L5) against required minimum
3. Check domain-specific scores against thresholds
4. Verify assessment not expired (ValidUntil)
5. Check required certifications present

**Result Fields**:
```go
type CapabilityCheckResult struct {
    CapabilityMet      bool
    LatestAssessment   *AICapabilityAssessment
    RequiredLevel      string
    ActualLevel        string
    DomainMatches      map[string]bool
    AssessmentExpired  bool
}
```

**Failure Conditions** (when enforcement enabled):
- No capability assessment exists
- Actual level below required level
- Domain scores below thresholds
- Required certifications missing

**Warnings**:
- "Capability assessment is expired, re-assessment recommended"

### 5. Fiduciary Duty Violations

**Purpose**: Block authorization when AI has unresolved duty breaches

**Process**:
1. Query `fiduciary_duty_violations` for agent and PoA
2. Filter to unresolved violations (status=open or investigating)
3. Count violations by severity (minor/moderate/major/critical)
4. Block if any critical violations exist (when enforcement enabled)

**Result Fields**:
```go
type FiduciaryCheckResult struct {
    HasViolations        bool
    UnresolvedViolations []*FiduciaryDutyViolation
    CriticalViolations   int
    BlockingAction       bool
}
```

**Failure Conditions** (when enforcement enabled):
- One or more critical unresolved violations exist

**Warnings**:
- "Agent has {n} unresolved fiduciary violations"

## Result Structure

```go
type GAuthPlusValidationResult struct {
    Valid            bool
    SuccessorCheck   *SuccessorCheckResult
    DelegationCheck  *DelegationCheckResult
    DualControlCheck *DualControlCheckResult
    CapabilityCheck  *CapabilityCheckResult
    FiduciaryCheck   *FiduciaryCheckResult
    Warnings         []string
    FailureReason    string
}
```

This result is embedded in:
- `RequestComplianceResult.GAuthPlusValidation`
- `GrantComplianceResult.GAuthPlusValidation`

## Usage Examples

### Example 1: Request Validation with GAuth+ Enforcement

```go
request := &gauth.ExtendedAuthorizationRequest{
    AuthorizationRequest: &baseRequest,
    PowerOfAttorney:      poaDefinition,
    AuthorizationChain:   authChain,
    RequestedActions:     []string{"transfer"},
    // Note: Include PoA ID in future for GAuth+ to work properly
}

result, err := complianceValidator.ValidateRequestCompliance(ctx, request)
if err != nil {
    log.Printf("Validation error: %v", err)
    return err
}

if !result.Valid {
    log.Printf("Validation failed: %s", result.FailureReason)
    
    // Check GAuth+ specific failures
    if result.GAuthPlusValidation != nil {
        gauthPlus := result.GAuthPlusValidation
        
        if gauthPlus.SuccessorCheck.SuccessorActive {
            log.Printf("Successor active: %s", gauthPlus.SuccessorCheck.EffectiveAgentID)
        }
        
        if gauthPlus.DelegationCheck.DepthExceeded {
            log.Printf("Delegation depth %d exceeds max %d",
                gauthPlus.DelegationCheck.CurrentDepth,
                gauthPlus.DelegationCheck.MaxAllowedDepth)
        }
        
        if !gauthPlus.CapabilityCheck.CapabilityMet {
            log.Printf("Capability %s below required %s",
                gauthPlus.CapabilityCheck.ActualLevel,
                gauthPlus.CapabilityCheck.RequiredLevel)
        }
        
        if gauthPlus.FiduciaryCheck.CriticalViolations > 0 {
            log.Printf("Agent has %d critical violations",
                gauthPlus.FiduciaryCheck.CriticalViolations)
        }
    }
    
    return fmt.Errorf("authorization denied: %s", result.FailureReason)
}

log.Printf("Validation passed with %d warnings", len(result.Warnings))
```

### Example 2: PDP Decision with GAuth+ Policies

```go
decisionRequest := &gauth.AuthorizationDecisionRequest{
    PowerOfAttorney:    poaDefinition,
    AuthorizationChain: authChain,
    ActionType:         "high-value-transaction",
    ResourceID:         "account-12345",
}

decision, err := pdp.MakeDecision(ctx, decisionRequest)
if err != nil {
    return err
}

if !decision.Authorized {
    // Reason will include GAuth+ policy violations:
    // - "GAuth+ policy violation: delegation depth 5 exceeds maximum 3"
    // - "GAuth+ policy violation: agent has 2 critical unresolved fiduciary violations"
    // - "GAuth+ policy violation: agent agent-001 does not meet capability requirements"
    log.Printf("Decision denied: %s", decision.Reason)
    return fmt.Errorf("authorization denied")
}

log.Printf("Decision authorized: %s", decision.Reason)
```

## Database Dependencies

GAuth+ integration requires the following database migrations:

**Migration 009**: GAuth+ Enhancement Tables
- `successor_activations`
- `ai_delegations`
- `dual_control_approvals`
- `fiduciary_duty_violations`
- `ai_capability_assessments`

**Migration 010**: Schema Fixes
- Fixed column names in `ai_capability_assessments`
- Added `risk_profile` and `notes` columns

## Known Limitations

### 1. PoA ID Tracking

`PoADefinition` doesn't have an ID field. Current implementation uses agent identity as placeholder:
```go
poaID := agentID // TODO: Get actual PoA ID from request metadata
```

**Recommendation**: Extend `ExtendedAuthorizationRequest` and `ExtendedAuthorizationGrant` with explicit `PoAID` field.

### 2. Dual Control Querying

`CheckApprovalStatus` only takes approval ID, not PoA + action type. Current implementation is permissive.

**Recommendation**: Add service method:
```go
func FindApprovalsByPoAAndAction(ctx context.Context, poaID, actionType string) ([]*DualControlApproval, error)
```

### 3. Delegation Chain Scope Validation

Delegation chain validation checks policies but doesn't validate scope inheritance strictly.

**Recommendation**: Enhance `ValidateDelegation` to perform recursive scope checking up the chain.

### 4. Capability Domain Requirements

PoA doesn't specify required capability domains. Currently uses default minimum level (L2).

**Recommendation**: Add `CapabilityRequirements` to `AuthorizationScope`:
```go
type AuthorizationScope struct {
    // ... existing fields
    RequiredCapabilities *CapabilityRequirements `json:"required_capabilities,omitempty"`
}
```

## Testing

Integration tests should cover:

1. **Successor Takeover**:
   - Primary fails, successor activates
   - Authorization request uses successor identity
   - Successor deactivation returns to primary

2. **Delegation Depth Enforcement**:
   - Create delegation chain exceeding max depth
   - Verify authorization blocked when depth exceeded
   - Test scope reduction through delegation chain

3. **Dual Control Workflow**:
   - Request high-risk action
   - Verify authorization blocked without approval
   - Submit approvals until threshold met
   - Verify authorization succeeds

4. **Capability Requirements**:
   - AI with insufficient capability level attempts action
   - Verify authorization blocked
   - Update capability assessment
   - Verify authorization succeeds

5. **Fiduciary Violations**:
   - Record critical fiduciary violation
   - Verify authorization blocked
   - Resolve violation
   - Verify authorization succeeds

## Performance Considerations

Each GAuth+ validation adds database queries:
- Successor check: 1 query
- Delegation chain: 1 recursive query
- Dual control: 1 query (if implemented)
- Capability assessment: 1 query
- Fiduciary violations: 1 query

**Total**: ~5 additional queries per authorization request

**Optimization strategies**:
1. **Caching**: Cache capability assessments and delegation chains
2. **Batch validation**: Validate multiple requests in single transaction
3. **Lazy evaluation**: Only check policies relevant to action type
4. **Read replicas**: Route GAuth+ queries to read replicas

## Migration Path

For existing deployments without GAuth+ tables:

**Phase 1: Deploy Schema**
```bash
psql -U postgres -d gauth < database/migrations/009_gauthplus_enhancements.sql
psql -U postgres -d gauth < database/migrations/010_fix_capability_assessments_schema.sql
```

**Phase 2: Deploy Code (Disabled)**
```go
// GAuth+ enforcement disabled by default
complianceValidator := gauth.NewComplianceValidator(chainValidator, pipClient, pdpClient)
// enforceGAuthPlus defaults to false
```

**Phase 3: Enable Advisory Mode**
```go
complianceValidator.SetGAuthPlusValidator(gauthPlusValidator)
complianceValidator.SetEnforceGAuthPlus(true)
// But disable blocking
gauthPlusValidator.SetEnforceCapabilities(false)
gauthPlusValidator.SetEnforceDualControl(false)
gauthPlusValidator.SetEnforceFiduciary(false)
// Warnings logged but authorization not blocked
```

**Phase 4: Enable Enforcement**
```go
gauthPlusValidator.SetEnforceCapabilities(true)
gauthPlusValidator.SetEnforceDualControl(true)
gauthPlusValidator.SetEnforceFiduciary(true)
// Full enforcement mode
```

## Compliance Mapping

GAuth+ features map to regulatory requirements:

| Feature | Regulatory Requirement |
|---------|----------------------|
| Successor Management | Business continuity, disaster recovery |
| Delegation Policies | Segregation of duties, least privilege |
| Dual Control | High-risk transaction approval, fraud prevention |
| Capability Assessment | Competency requirements, risk-based controls |
| Fiduciary Duties | Director liability, duty of care |

## Future Enhancements

1. **Dynamic Policy Loading**: Load GAuth+ policies from PAP instead of database
2. **Policy Conflict Resolution**: Handle conflicts between PoA and GAuth+ policies
3. **Audit Trail**: Enhanced logging of all GAuth+ policy decisions
4. **Real-time Monitoring**: Dashboard for GAuth+ policy violations
5. **Machine Learning**: Anomaly detection in delegation patterns
6. **Multi-tenant Isolation**: Separate GAuth+ policies per organization
7. **API Rate Limiting**: Prevent policy check abuse
8. **Webhook Notifications**: Alert on critical violations
9. **Policy Simulation**: Test policy changes before deployment
10. **Compliance Reports**: Automated GAuth+ compliance reporting

## Support

For questions or issues with GAuth+ integration:
1. Check logs for GAuth+ warnings and errors
2. Verify database migrations applied correctly
3. Confirm GAuth+ services initialized properly
4. Review enforcement mode configuration
5. Test with advisory mode first before strict enforcement

## References

- [RFC-0111: GAuth Authorization Framework](../docs/RFC_IMPLEMENTATION_COVERAGE.md)
- [GAuth+ Phase 1 Completion](../GAUTH_PLUS_PHASE1_COMPLETION.md)
- [GAuth+ Phase 2 Completion](../GAUTH_PLUS_PHASE2_COMPLETION.md)
- [GAuth+ Integration Test Report](../GAUTH_PLUS_INTEGRATION_TEST_REPORT.md)
- [Database Migration 009](../database/migrations/009_gauthplus_enhancements.sql)
- [Database Migration 010](../database/migrations/010_fix_capability_assessments_schema.sql)
