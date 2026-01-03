---
title: AgentAuth + RFC 9396 (Rich Authorization Requests) Integration Guide
category: architecture
status: proposed
lastUpdated: 2025-11-19
owners: architecture-team
---

# AgentAuth + RFC 9396 (Rich Authorization Requests) Integration Guide

## Purpose

This document provides a practical implementation guide for integrating RFC 9396 (Rich Authorization Requests) into AgentAuth to combine:
- **AgentAuth**: Legal delegation chains and Proof of Authorization framework
- **RFC 9396**: Fine-grained resource-level permissions

---

## Integration Benefits

### Combined Strengths

| Capability | AgentAuth Provides | RFC 9396 Provides |
|:-----------|:---------------|:------------------|
| **Legal Authority** | ✅ Proof of Authorization | - |
| **Authorization Chains** | ✅ Multi-level validation | - |
| **Commercial Register** | ✅ Corporate authority | - |
| **Identity Verification** | ✅ PVP (18 countries) | - |
| **Fine-Grained Permissions** | - | ✅ Resource details |
| **Action Granularity** | - | ✅ Specific actions |
| **Data Type Filtering** | - | ✅ Specific data types |
| **Constraints** | ⚠️ Value limits | ✅ Full constraints |

**Result**: First authorization system with both legal framework AND fine-grained permissions.

---

## Implementation Strategy

### Phase 1: Data Model Extension

#### 1.1 Add `AuthorizationDetail` Type

```go
// pkg/agentauth/rar_support.go

package agentauth

// RFC 9396 Authorization Detail
type AuthorizationDetail struct {
    Type        string                 `json:"type"`
    Actions     []string               `json:"actions,omitempty"`
    Locations   []string               `json:"locations,omitempty"`
    DataTypes   []string               `json:"datatypes,omitempty"`
    Identifier  string                 `json:"identifier,omitempty"`
    Privileges  []string               `json:"privileges,omitempty"`
    
    // Resource-specific fields
    InstructedAmount *Amount            `json:"instructedAmount,omitempty"`
    CreditorAccount  *Account           `json:"creditorAccount,omitempty"`
    
    // Metadata for extensibility
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type Amount struct {
    Currency string `json:"currency"`
    Amount   string `json:"amount"`
}

type Account struct {
    IBAN string `json:"iban,omitempty"`
    BIC  string `json:"bic,omitempty"`
}
```

#### 1.2 Extend Token Request

```go
// pkg/agentauth/token_request.go

type RFCCompliantAuthorizationRequest struct {
    // Existing AgentAuth fields
    SubscriptionID   string
    RequestedScope   map[string]interface{}
    PoACredentialRef string
    
    // RFC 9396 extension
    AuthorizationDetails []AuthorizationDetail `json:"authorization_details,omitempty"`
}
```

#### 1.3 Extend Extended Token

```go
// pkg/agentauth/extended_token.go

type ExtendedToken struct {
    // Existing fields
    AccessToken       string
    TokenType         string
    ExpiresIn         int64
    PowerOfAttorney   *poa.PoADefinition
    AuthorizationChain []*AuthorizationChainEntity
    VerificationProof *VerificationProof
    ComplianceLevel   string
    
    // RFC 9396 extension
    AuthorizationDetails []AuthorizationDetail `json:"authorization_details,omitempty"`
}
```

---

### Phase 2: Validation Logic

#### 2.1 Validate Authorization Details Against PoA

```go
// pkg/agentauth/rar_validator.go

package agentauth

import (
    "fmt"
    "github.com/mauriciomferz/AgentAuth/pkg/poa"
)

type RARValidator struct {
    poaValidator *poa.Validator
}

func NewRARValidator() *RARValidator {
    return &RARValidator{
        poaValidator: poa.NewValidator(),
    }
}

// ValidateAuthorizationDetails ensures requested details are within PoA scope
func (v *RARValidator) ValidateAuthorizationDetails(
    poaDef *poa.PoADefinition,
    details []AuthorizationDetail,
) error {
    for _, detail := range details {
        // Validate actions are authorized
        if err := v.validateActions(poaDef, detail); err != nil {
            return fmt.Errorf("action validation failed: %w", err)
        }
        
        // Validate locations match authorized resources
        if err := v.validateLocations(poaDef, detail); err != nil {
            return fmt.Errorf("location validation failed: %w", err)
        }
        
        // Validate constraints (e.g., amounts) against PoA restrictions
        if err := v.validateConstraints(poaDef, detail); err != nil {
            return fmt.Errorf("constraint validation failed: %w", err)
        }
        
        // Validate data types if specified
        if err := v.validateDataTypes(poaDef, detail); err != nil {
            return fmt.Errorf("data type validation failed: %w", err)
        }
    }
    
    return nil
}

func (v *RARValidator) validateActions(
    poaDef *poa.PoADefinition,
    detail AuthorizationDetail,
) error {
    // Check if requested actions are in PoA scope
    authorizedActions := poaDef.Scope.Actions
    
    for _, requestedAction := range detail.Actions {
        if !contains(authorizedActions, requestedAction) {
            return fmt.Errorf("action '%s' not authorized in PoA", requestedAction)
        }
    }
    
    return nil
}

func (v *RARValidator) validateLocations(
    poaDef *poa.PoADefinition,
    detail AuthorizationDetail,
) error {
    // Validate locations match authorized resources
    authorizedLocations := poaDef.Scope.Locations
    
    for _, requestedLocation := range detail.Locations {
        if !matchesPattern(authorizedLocations, requestedLocation) {
            return fmt.Errorf("location '%s' not authorized", requestedLocation)
        }
    }
    
    return nil
}

func (v *RARValidator) validateConstraints(
    poaDef *poa.PoADefinition,
    detail AuthorizationDetail,
) error {
    // Validate amount constraints
    if detail.InstructedAmount != nil {
        if poaDef.Restrictions.ValueLimits.Max > 0 {
            amount, err := parseAmount(detail.InstructedAmount.Amount)
            if err != nil {
                return fmt.Errorf("invalid amount: %w", err)
            }
            
            if amount > poaDef.Restrictions.ValueLimits.Max {
                return fmt.Errorf("amount %.2f exceeds PoA limit %.2f", 
                    amount, poaDef.Restrictions.ValueLimits.Max)
            }
        }
    }
    
    return nil
}

func (v *RARValidator) validateDataTypes(
    poaDef *poa.PoADefinition,
    detail AuthorizationDetail,
) error {
    // Validate data types if PoA specifies them
    if len(poaDef.Scope.DataTypes) > 0 {
        for _, requestedType := range detail.DataTypes {
            if !contains(poaDef.Scope.DataTypes, requestedType) {
                return fmt.Errorf("data type '%s' not authorized", requestedType)
            }
        }
    }
    
    return nil
}
```

---

### Phase 3: Protocol Flow Integration

#### 3.1 Enhanced Request Flow

```
Client Request
      ↓
┌─────────────────────────────────────────┐
│ Step (a): Authorization Request         │
│ • Subscription ID                       │
│ • Proof of Authorization reference           │
│ • authorization_details (RFC 9396) ←NEW │
└─────────────────────────────────────────┘
      ↓
┌─────────────────────────────────────────┐
│ Step (b): Request Compliance Validation │
│ • Validate PoA scope                    │
│ • Validate authorization_details ←NEW   │
│   - Actions in PoA scope?               │
│   - Locations authorized?               │
│   - Amounts within limits?              │
│   - Data types permitted?               │
└─────────────────────────────────────────┘
      ↓
Extended Token with authorization_details
```

#### 3.2 Modified Token Issuance

```go
// pkg/agentauth/protocol_orchestrator.go

func (o *ProtocolOrchestrator) ExecuteRFCCompliantFlow(
    ctx context.Context,
    req *RFCCompliantAuthorizationRequest,
) (*RFCCompliantTokenResponse, error) {
    
    // Step (a): Verify subscription
    subscription, err := o.subscriptionStore.GetSubscription(req.SubscriptionID)
    if err != nil {
        return nil, fmt.Errorf("subscription not found: %w", err)
    }
    
    // Get PoA definition
    poaDef := subscription.PowerOfAttorney
    
    // NEW: Validate authorization_details against PoA
    if len(req.AuthorizationDetails) > 0 {
        rarValidator := NewRARValidator()
        if err := rarValidator.ValidateAuthorizationDetails(poaDef, req.AuthorizationDetails); err != nil {
            return nil, fmt.Errorf("authorization_details validation failed: %w", err)
        }
    }
    
    // Step (b): Request compliance validation
    complianceResult, err := o.complianceValidator.ValidateRequestCompliance(/* ... */)
    if err != nil {
        return nil, err
    }
    
    // Step (c): Issue grant
    grant, err := o.IssueAuthorizationGrant(/* ... */)
    if err != nil {
        return nil, err
    }
    
    // Step (e): Create extended token
    extendedToken, err := o.extendedTokenService.CreateExtendedToken(ctx, &ExtendedTokenParams{
        Grant:                grant,
        PowerOfAttorney:      poaDef,
        AuthorizationChain:   subscription.AuthorizationChain,
        AuthorizationDetails: req.AuthorizationDetails, // NEW: Include RAR details
    })
    if err != nil {
        return nil, err
    }
    
    // ... rest of flow
    
    return &RFCCompliantTokenResponse{
        ExtendedToken: extendedToken,
        // ...
    }, nil
}
```

---

### Phase 4: Resource Server Validation

#### 4.1 Extended Token Validation

```go
// Resource Server validation logic

func (rs *ResourceServer) ValidateExtendedToken(
    ctx context.Context,
    token *ExtendedToken,
    requestedResource string,
    requestedAction string,
) error {
    
    // 1. AgentAuth validation (existing)
    if err := rs.validatePowerOfAttorney(token.PowerOfAttorney); err != nil {
        return err
    }
    
    if err := rs.validateAuthorizationChain(token.AuthorizationChain); err != nil {
        return err
    }
    
    // 2. RFC 9396 validation (NEW)
    if len(token.AuthorizationDetails) > 0 {
        if err := rs.validateAuthorizationDetails(
            token.AuthorizationDetails,
            requestedResource,
            requestedAction,
        ); err != nil {
            return err
        }
    }
    
    return nil
}

func (rs *ResourceServer) validateAuthorizationDetails(
    details []AuthorizationDetail,
    resourceURL string,
    action string,
) error {
    
    // Find matching authorization detail
    for _, detail := range details {
        // Check if resource matches any location
        if matchesLocation(detail.Locations, resourceURL) {
            // Check if action is authorized
            if contains(detail.Actions, action) {
                // Additional checks based on detail type
                if err := rs.validateDetailConstraints(detail); err != nil {
                    return err
                }
                return nil // Authorized
            }
        }
    }
    
    return fmt.Errorf("no matching authorization_detail for resource '%s' and action '%s'", 
        resourceURL, action)
}
```

---

## Real-World Use Cases

### Use Case 1: Healthcare AI with Fine-Grained Access

**Scenario**: AI diagnostic assistant with legal guardian authorization needs specific patient data.

```json
{
  "subscription_id": "sub_healthcare_123",
  
  "power_of_attorney": {
    "poa_id": "poa_guardian_456",
    "authorization_chain": [
      {"entity": "Legal Guardian", "authority": "Parental Rights"},
      {"entity": "Patient", "authority": "Consented"},
      {"entity": "Hospital", "authority": "Healthcare Provider"},
      {"entity": "AI Diagnostic System", "authority": "Granted"}
    ]
  },
  
  "authorization_details": [
    {
      "type": "patient_record_access",
      "actions": ["read", "analyze"],
      "locations": ["https://ehr.hospital.com/patients/789"],
      "datatypes": ["lab_results", "imaging", "vital_signs"],
      "purpose": "ai_diagnostic_assistance",
      "constraints": {
        "max_records": 100,
        "time_range": "2024-01-01/2025-12-31",
        "exclude_sensitive": ["psychiatric", "genetic"]
      }
    }
  ]
}
```

**Validation**:
- ✅ AgentAuth validates legal guardian authority chain
- ✅ RFC 9396 ensures only authorized data types are accessed
- ✅ Time-based and content-based constraints enforced

---

### Use Case 2: Corporate AI with Financial Transactions

**Scenario**: Board-authorized AI needs to execute specific financial transactions.

```json
{
  "subscription_id": "sub_corporate_ai_789",
  
  "power_of_attorney": {
    "poa_id": "poa_board_resolution_123",
    "authorization_chain": [
      {"entity": "Board of Directors", "authority": "Corporate Governance"},
      {"entity": "CEO", "authority": "Executive"},
      {"entity": "CFO", "authority": "Financial Officer"},
      {"entity": "AI Treasury Agent", "authority": "Granted"}
    ]
  },
  
  "authorization_details": [
    {
      "type": "payment_initiation",
      "actions": ["initiate", "cancel"],
      "locations": ["https://bank.example.com/payments"],
      "instructedAmount": {
        "currency": "EUR",
        "amount": "50000.00"
      },
      "creditorAccount": {
        "iban": "DE89370400440532013000"
      },
      "constraints": {
        "max_daily_amount": 100000,
        "requires_dual_approval": true,
        "geographic_restrictions": ["EU", "US"]
      }
    }
  ]
}
```

**Validation**:
- ✅ AgentAuth validates board authorization chain
- ✅ Commercial register verification
- ✅ RFC 9396 ensures amount limits and geographic restrictions
- ✅ Dual approval enforcement

---

## Migration Guide

### For Existing AgentAuth Implementations

#### Step 1: Add RAR Support (Backward Compatible)

```go
// Keep existing token requests working
type TokenRequest struct {
    GrantID string
    Scope   []string
    
    // Optional: new field
    AuthorizationDetails []AuthorizationDetail `json:"authorization_details,omitempty"`
}

// If authorization_details is empty, use existing AgentAuth flow
// If populated, validate against PoA and include in token
```

#### Step 2: Update Extended Token Schema

```sql
-- Add authorization_details column to extended_tokens table
ALTER TABLE extended_tokens 
ADD COLUMN authorization_details JSONB;

-- Create index for querying
CREATE INDEX idx_authorization_details_type 
ON extended_tokens ((authorization_details->>'type'));
```

#### Step 3: Update Validation Logic

```go
// Extend existing validators
func (v *ComplianceValidator) ValidateRequest(
    req *TokenRequest,
    poa *poa.PoADefinition,
) error {
    // Existing validation
    if err := v.validateScope(req.Scope, poa); err != nil {
        return err
    }
    
    // NEW: RAR validation if present
    if len(req.AuthorizationDetails) > 0 {
        rarValidator := NewRARValidator()
        if err := rarValidator.ValidateAuthorizationDetails(poa, req.AuthorizationDetails); err != nil {
            return err
        }
    }
    
    return nil
}
```

---

## API Examples

### Request with RAR

```http
POST /v1/token/rfc HTTP/1.1
Host: agentauth.as.example.com
Content-Type: application/json

{
  "subscription_id": "sub_123",
  "poa_credential_ref": "poa_456",
  "authorization_details": [
    {
      "type": "financial_transaction",
      "actions": ["transfer"],
      "locations": ["https://bank.example.com/accounts/789"],
      "instructedAmount": {
        "currency": "USD",
        "amount": "10000.00"
      }
    }
  ]
}
```

### Response with RAR

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "extended_token": {
    "access_token": "agentauth_at_...",
    "token_type": "AgentAuth-Extended",
    "expires_in": 3600,
    "power_of_attorney": {
      "poa_id": "poa_456",
      "status": "active"
    },
    "authorization_chain": [...],
    "authorization_details": [
      {
        "type": "financial_transaction",
        "actions": ["transfer"],
        "locations": ["https://bank.example.com/accounts/789"],
        "instructedAmount": {
          "currency": "USD",
          "amount": "10000.00"
        }
      }
    ]
  }
}
```

---

## Testing

### Unit Tests

```go
// pkg/agentauth/rar_validator_test.go

func TestValidateAuthorizationDetails_Success(t *testing.T) {
    validator := NewRARValidator()
    
    poa := &poa.PoADefinition{
        Scope: poa.Scope{
            Actions:   []string{"read", "write"},
            Locations: []string{"https://api.example.com/*"},
            DataTypes: []string{"patient_record"},
        },
        Restrictions: poa.Restrictions{
            ValueLimits: poa.ValueLimits{Max: 10000},
        },
    }
    
    details := []AuthorizationDetail{
        {
            Type:      "patient_record_access",
            Actions:   []string{"read"},
            Locations: []string{"https://api.example.com/patients/123"},
            DataTypes: []string{"patient_record"},
        },
    }
    
    err := validator.ValidateAuthorizationDetails(poa, details)
    assert.NoError(t, err)
}

func TestValidateAuthorizationDetails_UnauthorizedAction(t *testing.T) {
    validator := NewRARValidator()
    
    poa := &poa.PoADefinition{
        Scope: poa.Scope{
            Actions: []string{"read"},
        },
    }
    
    details := []AuthorizationDetail{
        {
            Actions: []string{"delete"}, // Not authorized
        },
    }
    
    err := validator.ValidateAuthorizationDetails(poa, details)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not authorized")
}
```

---

## Benefits Summary

### For AI Systems
✅ Legal authority validation (AgentAuth)  
✅ Fine-grained resource control (RFC 9396)  
✅ Reduced over-privileging  
✅ Clear audit trail  

### For Resource Servers
✅ Self-contained tokens (AgentAuth)  
✅ Detailed permission validation (RFC 9396)  
✅ No need for scope guessing  
✅ Compliance enforcement  

### For Developers
✅ Standard OAuth 2.0 extension (RFC 9396)  
✅ Clear authorization semantics  
✅ Backward compatible  
✅ Enhanced security  

---

## Conclusion

Integrating RFC 9396 (Rich Authorization Requests) into AgentAuth creates **the first authorization framework combining legal delegation chains with fine-grained resource permissions**.

**Implementation is straightforward**:
1. Add `authorization_details` field to token requests
2. Validate details against Proof of Authorization
3. Embed details in Extended Tokens
4. Validate at Resource Server

This makes AgentAuth suitable for **regulated AI systems** requiring both legal authority and precise resource control.

---

## References

- [RFC 9396 - Rich Authorization Requests](https://datatracker.ietf.org/doc/rfc9396/)
- [AAP-001 - AgentAuth Authorization Framework](../AAP_AAP-001.md)
- [AgentAuth Architecture](../../ARCHITECTURE_SOLUTION.md)
- [RFC 9396 vs AgentAuth Comparison](RFC9396_RAR_AGENTAUTH_COMPARISON.md)
