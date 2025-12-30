# US Identity Verification Architecture

**Project**: AgentAuth AAP-001 External Connectors Enhancement  
**Component**: US Identity Verification System  
**Date**: November 12, 2025  
**Status**: Design Phase  
**Priority**: P1 - CRITICAL (Geographic Expansion)

---

## Executive Summary

This document defines the architecture for US identity verification integration within the AgentAuth system. The implementation extends the existing PVP (Power Verification Point) framework to support multiple US identity document types and verification methods, addressing the gap in non-European identity verification capabilities.

### Objectives

1. **Geographic Expansion**: Enable AgentAuth to support US-based identity verification
2. **Multi-Document Support**: Verify US passports, driver's licenses, state IDs, and SSN
3. **Compliance**: Meet AAP-001 PVP requirements with US-specific adaptations
4. **Integration**: Seamlessly integrate with existing PVPClient framework

### Scope

**In Scope**:
- ✅ US Passport verification (US State Department)
- ✅ State driver's license verification (50 states + DC, territories)
- ✅ Social Security Number (SSN) validation
- ✅ State-issued ID card verification
- ✅ Commercial API provider integration (Trulioo, Persona, or Jumio)
- ✅ Circuit breaker, retry logic, fallback mechanisms (existing framework)

**Out of Scope**:
- ❌ US business entity verification (commercial registers) - separate component
- ❌ Credit bureau integration (Equifax, Experian, TransUnion)
- ❌ Real-time biometric verification (facial recognition, liveness detection)
- ❌ Background checks or criminal record verification

---

## 1. System Architecture

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        AgentAuth Application                        │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │            Identity Verification Service                  │  │
│  │  (pkg/agentauth/service/identity_verification.go)             │  │
│  └──────────────────┬────────────────────────────────────────┘  │
│                     │                                           │
│                     ▼                                           │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              PVPClient Framework                          │  │
│  │       (pkg/agentauth/external/pvp_pip_clients.go)             │  │
│  │                                                           │  │
│  │  • Circuit Breaker    • Retry Logic                       │  │
│  │  • Fallback Handling  • Metrics Tracking                  │  │
│  └──────────────────┬────────────────────────────────────────┘  │
│                     │                                           │
│                     ▼                                           │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │          US Identity Verifier (NEW)                       │  │
│  │      (pkg/agentauth/external/us_identity_verifier.go)         │  │
│  │                                                           │  │
│  │  ┌──────────────────┐  ┌──────────────────┐               │  │
│  │  │ Passport Verifier│  │  DL Verifier     │               │  │
│  │  └──────────────────┘  └──────────────────┘               │  │
│  │  ┌──────────────────┐  ┌──────────────────┐               │  │
│  │  │  SSN Validator   │  │ State ID Verifier│               │  │
│  │  └──────────────────┘  └──────────────────┘               │  │
│  └──────────────────┬────────────────────────────────────────┘  │
│                     │                                           │
└─────────────────────┼───────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│              External API Provider Abstraction                  │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Trulioo   │  │   Persona   │  │    Jumio    │              │
│  │GlobalGateway│  │    API      │  │  Netverify  │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Component Overview

#### USIdentityVerifier
**Location**: `pkg/agentauth/external/us_identity_verifier.go`  
**Responsibility**: US-specific identity verification logic  
**Methods**:
- `VerifyPassport(ctx, *PassportVerificationRequest) (*IdentityVerificationResult, error)`
- `VerifyDriverLicense(ctx, *DLVerificationRequest) (*IdentityVerificationResult, error)`
- `VerifySSN(ctx, *SSNValidationRequest) (*SSNValidationResult, error)`
- `VerifyStateID(ctx, *StateIDVerificationRequest) (*IdentityVerificationResult, error)`

#### API Provider Adapter
**Location**: `pkg/agentauth/external/us_api_providers.go`  
**Responsibility**: Abstract API provider implementations  
**Interface**:
```go
type USIdentityAPIProvider interface {
    VerifyDocument(ctx context.Context, req *DocumentVerificationRequest) (*DocumentVerificationResponse, error)
    ValidateSSN(ctx context.Context, req *SSNValidationRequest) (*SSNValidationResponse, error)
    GetProviderName() string
    GetSupportedDocumentTypes() []string
}
```

**Implementations**:
- `TruliooProvider` - Trulioo GlobalGateway API
- `PersonaProvider` - Persona Identity API
- `JumioProvider` - Jumio Netverify API

---

## 2. API Provider Evaluation

### 2.1 Provider Comparison

| Criteria | Trulioo | Persona | Jumio | Recommendation |
|----------|---------|---------|-------|----------------|
| **US Coverage** | Excellent (all 50 states) | Excellent | Excellent | All adequate |
| **Document Types** | Passport, DL, State ID, SSN | Passport, DL, State ID | Passport, DL, State ID | Trulioo (SSN included) |
| **API Quality** | REST, JSON, well-documented | GraphQL, modern | REST, comprehensive | Persona (modern) |
| **Pricing** | $0.75-$2.00 per verification | $0.50-$1.50 | $1.00-$3.00 | Persona (cost-effective) |
| **Compliance** | KYC/AML, FCRA compliant | GDPR, SOC 2 | GDPR, CCPA, SOC 2 | All compliant |
| **Latency** | 1-3 seconds | 0.5-2 seconds | 1-3 seconds | Persona (fastest) |
| **Global Support** | 195+ countries | 200+ countries | 200+ countries | All global |
| **Integration Complexity** | Medium | Low | Medium | Persona (easiest) |
| **Developer Experience** | Good | Excellent | Good | Persona (best docs) |
| **Sandbox/Testing** | ✅ Free sandbox | ✅ Free test mode | ✅ Free test mode | All adequate |

### 2.2 Recommended Primary Provider: **Persona**

**Rationale**:
1. **Modern API**: GraphQL with strong typing, excellent error handling
2. **Cost-Effective**: Lower per-verification cost ($0.50-$1.50)
3. **Fast**: Sub-second verification for many document types
4. **Developer-Friendly**: Excellent documentation, SDKs, sandbox
5. **Compliance**: SOC 2 Type II, GDPR, strong security posture

**Backup Provider**: **Trulioo** (for SSN validation and fallback)

### 2.3 Provider Configuration

```yaml
# config/external_connectors.yaml
us_identity_verification:
  primary_provider: persona
  fallback_provider: trulioo
  
  providers:
    persona:
      api_key: ${PERSONA_API_KEY}
      api_secret: ${PERSONA_API_SECRET}
      base_url: https://api.withpersona.com/v1
      timeout: 5s
      max_retries: 3
      
    trulioo:
      api_key: ${TRULIOO_API_KEY}
      base_url: https://api.globaldatacompany.com
      timeout: 5s
      max_retries: 3
      
  document_types:
    - us_passport
    - us_driver_license
    - us_state_id
    - ssn
    
  state_specific_config:
    california:
      dl_format: "^[A-Z]\\d{7}$"
      enhanced_verification: true
    texas:
      dl_format: "^\\d{8}$"
      enhanced_verification: false
    # ... additional states
```

---

## 3. Data Models

### 3.1 Request Models

```go
// PassportVerificationRequest for US passport verification
type PassportVerificationRequest struct {
    PassportNumber       string    `json:"passport_number" validate:"required"`
    FirstName            string    `json:"first_name" validate:"required"`
    LastName             string    `json:"last_name" validate:"required"`
    DateOfBirth          time.Time `json:"date_of_birth" validate:"required"`
    IssueDate            time.Time `json:"issue_date" validate:"required"`
    ExpirationDate       time.Time `json:"expiration_date" validate:"required"`
    Nationality          string    `json:"nationality" validate:"required,eq=US"`
    
    // Optional fields for enhanced verification
    PlaceOfBirth         string    `json:"place_of_birth,omitempty"`
    PassportImageFront   []byte    `json:"passport_image_front,omitempty"`
    PassportImageBack    []byte    `json:"passport_image_back,omitempty"`
    FaceImage            []byte    `json:"face_image,omitempty"`
    
    // Metadata
    RequestID            string    `json:"request_id"`
    Timestamp            time.Time `json:"timestamp"`
}

// DLVerificationRequest for driver's license verification
type DLVerificationRequest struct {
    LicenseNumber        string    `json:"license_number" validate:"required"`
    State                string    `json:"state" validate:"required,len=2"` // Two-letter state code
    FirstName            string    `json:"first_name" validate:"required"`
    LastName             string    `json:"last_name" validate:"required"`
    DateOfBirth          time.Time `json:"date_of_birth" validate:"required"`
    IssueDate            time.Time `json:"issue_date" validate:"required"`
    ExpirationDate       time.Time `json:"expiration_date" validate:"required"`
    
    // Optional fields
    Address              *Address  `json:"address,omitempty"`
    LicenseClass         string    `json:"license_class,omitempty"` // e.g., "C", "M"
    Endorsements         []string  `json:"endorsements,omitempty"`
    Restrictions         []string  `json:"restrictions,omitempty"`
    LicenseImageFront    []byte    `json:"license_image_front,omitempty"`
    LicenseImageBack     []byte    `json:"license_image_back,omitempty"`
    
    // Metadata
    RequestID            string    `json:"request_id"`
    Timestamp            time.Time `json:"timestamp"`
}

// SSNValidationRequest for Social Security Number validation
type SSNValidationRequest struct {
    SSN                  string    `json:"ssn" validate:"required,len=9"` // 9 digits without dashes
    FirstName            string    `json:"first_name" validate:"required"`
    LastName             string    `json:"last_name" validate:"required"`
    DateOfBirth          time.Time `json:"date_of_birth" validate:"required"`
    
    // Optional fields
    MiddleName           string    `json:"middle_name,omitempty"`
    Address              *Address  `json:"address,omitempty"`
    
    // Validation level
    ValidationLevel      SSNValidationLevel `json:"validation_level"` // basic, standard, comprehensive
    
    // Metadata
    RequestID            string    `json:"request_id"`
    Timestamp            time.Time `json:"timestamp"`
}

// StateIDVerificationRequest for state-issued ID cards
type StateIDVerificationRequest struct {
    IDNumber             string    `json:"id_number" validate:"required"`
    State                string    `json:"state" validate:"required,len=2"`
    FirstName            string    `json:"first_name" validate:"required"`
    LastName             string    `json:"last_name" validate:"required"`
    DateOfBirth          time.Time `json:"date_of_birth" validate:"required"`
    IssueDate            time.Time `json:"issue_date" validate:"required"`
    ExpirationDate       time.Time `json:"expiration_date" validate:"required"`
    
    // Optional fields
    Address              *Address  `json:"address,omitempty"`
    IDImageFront         []byte    `json:"id_image_front,omitempty"`
    IDImageBack          []byte    `json:"id_image_back,omitempty"`
    
    // Metadata
    RequestID            string    `json:"request_id"`
    Timestamp            time.Time `json:"timestamp"`
}

// Address for all verification types
type Address struct {
    StreetAddress1       string `json:"street_address_1"`
    StreetAddress2       string `json:"street_address_2,omitempty"`
    City                 string `json:"city"`
    State                string `json:"state" validate:"len=2"`
    ZipCode              string `json:"zip_code" validate:"required"`
    Country              string `json:"country" default:"US"`
}
```

### 3.2 Response Models

```go
// IdentityVerificationResult for passport, DL, state ID
type IdentityVerificationResult struct {
    // Verification status
    Verified             bool                   `json:"verified"`
    VerificationLevel    VerificationLevel      `json:"verification_level"` // basic, standard, enhanced
    ConfidenceScore      float64                `json:"confidence_score"` // 0.0 to 1.0
    
    // Document details
    DocumentType         DocumentType           `json:"document_type"`
    DocumentNumber       string                 `json:"document_number"`
    DocumentState        string                 `json:"document_state,omitempty"`
    IssuingAuthority     string                 `json:"issuing_authority"`
    
    // Identity details
    VerifiedIdentity     *VerifiedIdentity      `json:"verified_identity"`
    
    // Verification checks
    Checks               *VerificationChecks    `json:"checks"`
    
    // Warnings and errors
    Warnings             []string               `json:"warnings,omitempty"`
    Errors               []VerificationError    `json:"errors,omitempty"`
    
    // Provider information
    ProviderName         string                 `json:"provider_name"`
    ProviderTransactionID string                `json:"provider_transaction_id"`
    
    // Metadata
    RequestID            string                 `json:"request_id"`
    VerificationTimestamp time.Time             `json:"verification_timestamp"`
    ProcessingTimeMs     int64                  `json:"processing_time_ms"`
}

// SSNValidationResult for SSN validation
type SSNValidationResult struct {
    // Validation status
    Valid                bool                   `json:"valid"`
    ValidationLevel      SSNValidationLevel     `json:"validation_level"`
    ConfidenceScore      float64                `json:"confidence_score"`
    
    // SSN details
    SSN                  string                 `json:"ssn"` // Masked: XXX-XX-1234
    IssuanceState        string                 `json:"issuance_state,omitempty"`
    IssuanceYearRange    string                 `json:"issuance_year_range,omitempty"`
    
    // Validation checks
    FormatValid          bool                   `json:"format_valid"`
    NotDeceased          bool                   `json:"not_deceased"`
    NameMatch            *NameMatchResult       `json:"name_match,omitempty"`
    DOBMatch             *DOBMatchResult        `json:"dob_match,omitempty"`
    
    // Warnings and errors
    Warnings             []string               `json:"warnings,omitempty"`
    Errors               []VerificationError    `json:"errors,omitempty"`
    
    // Provider information
    ProviderName         string                 `json:"provider_name"`
    ProviderTransactionID string                `json:"provider_transaction_id"`
    
    // Metadata
    RequestID            string                 `json:"request_id"`
    ValidationTimestamp  time.Time              `json:"validation_timestamp"`
    ProcessingTimeMs     int64                  `json:"processing_time_ms"`
}

// VerifiedIdentity contains verified identity information
type VerifiedIdentity struct {
    FirstName            string    `json:"first_name"`
    MiddleName           string    `json:"middle_name,omitempty"`
    LastName             string    `json:"last_name"`
    DateOfBirth          time.Time `json:"date_of_birth"`
    Address              *Address  `json:"address,omitempty"`
    Nationality          string    `json:"nationality,omitempty"`
}

// VerificationChecks contains detailed verification check results
type VerificationChecks struct {
    DocumentAuthenticity *CheckResult `json:"document_authenticity"` // Is document genuine?
    DocumentExpiration   *CheckResult `json:"document_expiration"`   // Is document valid (not expired)?
    NameMatch            *CheckResult `json:"name_match"`            // Does name match?
    DOBMatch             *CheckResult `json:"dob_match"`             // Does DOB match?
    AddressMatch         *CheckResult `json:"address_match,omitempty"`
    FaceMatch            *CheckResult `json:"face_match,omitempty"`  // If biometric provided
    LivenessCheck        *CheckResult `json:"liveness_check,omitempty"`
}

// CheckResult for individual verification checks
type CheckResult struct {
    Status               CheckStatus `json:"status"` // passed, failed, warning, not_performed
    Details              string      `json:"details,omitempty"`
    Score                float64     `json:"score,omitempty"` // 0.0 to 1.0
}

// Enums
type DocumentType string
const (
    DocumentTypePassport       DocumentType = "passport"
    DocumentTypeDriverLicense  DocumentType = "driver_license"
    DocumentTypeStateID        DocumentType = "state_id"
)

type VerificationLevel string
const (
    VerificationLevelBasic     VerificationLevel = "basic"     // Format validation only
    VerificationLevelStandard  VerificationLevel = "standard"  // API verification
    VerificationLevelEnhanced  VerificationLevel = "enhanced"  // API + biometric
)

type SSNValidationLevel string
const (
    SSNValidationLevelBasic        SSNValidationLevel = "basic"        // Format only
    SSNValidationLevelStandard     SSNValidationLevel = "standard"     // SSN validity
    SSNValidationLevelComprehensive SSNValidationLevel = "comprehensive" // Full match
)

type CheckStatus string
const (
    CheckStatusPassed       CheckStatus = "passed"
    CheckStatusFailed       CheckStatus = "failed"
    CheckStatusWarning      CheckStatus = "warning"
    CheckStatusNotPerformed CheckStatus = "not_performed"
)
```

---

## 4. Integration with PVP Framework

### 4.1 PVPClient Integration

The USIdentityVerifier integrates with the existing `PVPClient` framework:

```go
// Example: Using USIdentityVerifier with PVPClient
func (s *IdentityVerificationService) VerifyUSPassport(
    ctx context.Context,
    req *agentauth.IdentityVerificationRequest,
) (*agentauth.IdentityVerificationResult, error) {
    
    // Convert to US-specific request format
    passportReq := &external.PassportVerificationRequest{
        PassportNumber: req.ProofData["passport_number"],
        FirstName:      req.SubjectClaims["given_name"],
        LastName:       req.SubjectClaims["family_name"],
        DateOfBirth:    parseDate(req.SubjectClaims["birthdate"]),
        // ... additional fields
    }
    
    // Execute through PVPClient framework (circuit breaker, retry, fallback)
    result, err := s.pvpClient.Execute(ctx, func() (interface{}, error) {
        return s.usVerifier.VerifyPassport(ctx, passportReq)
    })
    
    if err != nil {
        // Fallback to manual verification if available
        if s.pvpClient.config.FallbackEnabled {
            return s.fallbackVerifyPassport(ctx, passportReq)
        }
        return nil, err
    }
    
    // Convert to AgentAuth format
    return convertToAgentAuthResult(result.(*external.IdentityVerificationResult)), nil
}
```

### 4.2 Circuit Breaker Configuration

```go
// Circuit breaker settings for US identity verification
circuitBreakerConfig := &external.CircuitBreakerConfig{
    MaxFailures:  5,                 // Open after 5 consecutive failures
    Timeout:      30 * time.Second,  // Request timeout
    ResetTimeout: 60 * time.Second,  // Try half-open after 60 seconds
}

usVerifierConfig := &external.USIdentityVerifierConfig{
    PrimaryProvider:   persona.NewPersonaProvider(personaConfig),
    FallbackProvider:  trulioo.NewTruliooProvider(truliooConfig),
    CircuitBreaker:    external.NewCircuitBreaker(circuitBreakerConfig),
    MaxRetries:        3,
    RetryDelay:        1 * time.Second,
    BackoffMultiplier: 2.0,
}
```

---

## 5. State-Specific Handling

### 5.1 Driver's License Format Validation

Each US state has different driver's license number formats:

```go
// State-specific DL formats (samples)
var stateDLFormats = map[string]*regexp.Regexp{
    "CA": regexp.MustCompile(`^[A-Z]\d{7}$`),              // California: A1234567
    "TX": regexp.MustCompile(`^\d{8}$`),                   // Texas: 12345678
    "FL": regexp.MustCompile(`^[A-Z]\d{12}$`),             // Florida: A123456789012
    "NY": regexp.MustCompile(`^(\d{9}|\d{16})$`),          // New York: 9 or 16 digits
    "PA": regexp.MustCompile(`^\d{8}$`),                   // Pennsylvania: 12345678
    "IL": regexp.MustCompile(`^[A-Z]\d{11}$`),             // Illinois: A12345678901
    "OH": regexp.MustCompile(`^[A-Z]{2}\d{6}$`),           // Ohio: AB123456
    "GA": regexp.MustCompile(`^\d{9}$`),                   // Georgia: 123456789
    "NC": regexp.MustCompile(`^\d{12}$`),                  // North Carolina: 123456789012
    "MI": regexp.MustCompile(`^[A-Z]\d{12}$`),             // Michigan: A123456789012
    // ... additional states
}

func (v *USIdentityVerifier) validateDLFormat(state, licenseNumber string) bool {
    format, exists := stateDLFormats[strings.ToUpper(state)]
    if !exists {
        // Unknown state format, rely on API provider validation
        return true
    }
    return format.MatchString(licenseNumber)
}
```

### 5.2 Enhanced Verification States

Some states offer enhanced verification capabilities:

```go
// States with enhanced verification (REAL ID Act compliant)
var enhancedVerificationStates = map[string]bool{
    "CA": true,  // California: REAL ID compliant
    "TX": true,  // Texas: REAL ID compliant
    "FL": true,  // Florida: REAL ID compliant
    "NY": true,  // New York: Enhanced Driver License (EDL)
    "WA": true,  // Washington: Enhanced Driver License
    "MI": true,  // Michigan: Enhanced Driver License
    "VT": true,  // Vermont: Enhanced Driver License
    // ... additional states
}
```

---

## 6. Security and Privacy

### 6.1 Data Handling

**Sensitive Data Protection**:
1. **SSN Masking**: Always mask SSN in responses (XXX-XX-1234)
2. **Document Images**: Store encrypted, delete after verification
3. **Biometric Data**: Process in memory, never persist
4. **Audit Logging**: Log verification attempts without PII

**Implementation**:
```go
// Mask SSN for logging and responses
func maskSSN(ssn string) string {
    if len(ssn) != 9 {
        return "XXX-XX-XXXX"
    }
    return fmt.Sprintf("XXX-XX-%s", ssn[5:])
}

// Secure document image handling
func (v *USIdentityVerifier) processDocumentImage(image []byte) error {
    // Process image in memory
    defer func() {
        // Zero out image data
        for i := range image {
            image[i] = 0
        }
    }()
    
    // ... image processing logic
    
    return nil
}
```

### 6.2 Compliance

**Regulatory Compliance**:
- ✅ **FCRA (Fair Credit Reporting Act)**: Not using for credit decisions, no FCRA requirements
- ✅ **GDPR**: Data minimization, purpose limitation, right to erasure
- ✅ **CCPA (California Consumer Privacy Act)**: Disclosure, opt-out, data deletion
- ✅ **PIPEDA (Canada)**: For Canadian users accessing US verification

**Data Retention**:
- Verification results: 90 days (configurable)
- Document images: Delete immediately after verification
- Audit logs: 7 years (compliance requirement)

---

## 7. Error Handling and Fallback

### 7.1 Error Categories

```go
type VerificationError struct {
    Code        string `json:"code"`
    Message     string `json:"message"`
    Severity    string `json:"severity"` // error, warning, info
    Recoverable bool   `json:"recoverable"`
}

const (
    // Document errors
    ErrDocumentExpired          = "DOCUMENT_EXPIRED"
    ErrDocumentInvalid          = "DOCUMENT_INVALID"
    ErrDocumentNotFound         = "DOCUMENT_NOT_FOUND"
    ErrDocumentFormatInvalid    = "DOCUMENT_FORMAT_INVALID"
    
    // Identity mismatch errors
    ErrNameMismatch             = "NAME_MISMATCH"
    ErrDOBMismatch              = "DOB_MISMATCH"
    ErrAddressMismatch          = "ADDRESS_MISMATCH"
    
    // API provider errors
    ErrProviderUnavailable      = "PROVIDER_UNAVAILABLE"
    ErrProviderTimeout          = "PROVIDER_TIMEOUT"
    ErrProviderAuthFailure      = "PROVIDER_AUTH_FAILURE"
    ErrProviderRateLimited      = "PROVIDER_RATE_LIMITED"
    
    // SSN-specific errors
    ErrSSNInvalid               = "SSN_INVALID"
    ErrSSNDeceased              = "SSN_DECEASED"
    
    // System errors
    ErrCircuitBreakerOpen       = "CIRCUIT_BREAKER_OPEN"
    ErrMaxRetriesExceeded       = "MAX_RETRIES_EXCEEDED"
)
```

### 7.2 Fallback Strategy

```go
func (v *USIdentityVerifier) VerifyPassportWithFallback(
    ctx context.Context,
    req *PassportVerificationRequest,
) (*IdentityVerificationResult, error) {
    
    // Try primary provider (Persona)
    result, err := v.primaryProvider.VerifyDocument(ctx, convertToProviderRequest(req))
    if err == nil {
        return convertToVerificationResult(result), nil
    }
    
    // Log primary failure
    log.Warn("Primary provider failed, attempting fallback",
        "provider", v.primaryProvider.GetProviderName(),
        "error", err)
    
    // Check if fallback is appropriate
    if !shouldFallback(err) {
        return nil, err
    }
    
    // Try fallback provider (Trulioo)
    if v.fallbackProvider != nil {
        result, err := v.fallbackProvider.VerifyDocument(ctx, convertToProviderRequest(req))
        if err == nil {
            result.Warnings = append(result.Warnings, "Verified using fallback provider")
            return convertToVerificationResult(result), nil
        }
    }
    
    // Both providers failed, return manual verification result
    return v.manualVerificationFallback(ctx, req)
}

func shouldFallback(err error) bool {
    // Fallback for transient errors only
    switch {
    case errors.Is(err, ErrProviderUnavailable):
        return true
    case errors.Is(err, ErrProviderTimeout):
        return true
    case errors.Is(err, ErrCircuitBreakerOpen):
        return true
    default:
        return false
    }
}
```

---

## 8. Performance and Scalability

### 8.1 Performance Targets

| Metric | Target | Rationale |
|--------|--------|-----------|
| **P50 Latency** | < 2 seconds | User experience |
| **P95 Latency** | < 5 seconds | Acceptable wait time |
| **P99 Latency** | < 10 seconds | Edge cases |
| **Throughput** | 100 req/sec | Initial scale |
| **Error Rate** | < 1% | Reliability |
| **Circuit Breaker** | Open after 5 failures | Fault isolation |

### 8.2 Caching Strategy

```go
// Cache verified documents to reduce API calls
type VerificationCache struct {
    cache *lru.Cache // LRU cache (size: 10,000 entries)
    ttl   time.Duration // 24 hours
}

func (c *VerificationCache) Get(req *PassportVerificationRequest) (*IdentityVerificationResult, bool) {
    key := generateCacheKey(req)
    if val, found := c.cache.Get(key); found {
        cached := val.(*cachedVerification)
        if time.Since(cached.Timestamp) < c.ttl {
            return cached.Result, true
        }
        // Expired, remove from cache
        c.cache.Remove(key)
    }
    return nil, false
}

func generateCacheKey(req *PassportVerificationRequest) string {
    // Hash sensitive data for cache key
    data := fmt.Sprintf("%s|%s|%s|%s",
        req.PassportNumber,
        req.FirstName,
        req.LastName,
        req.DateOfBirth.Format("2006-01-02"))
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}
```

### 8.3 Rate Limiting

```go
// Per-provider rate limiting
type ProviderRateLimiter struct {
    limiter *rate.Limiter
    config  *RateLimitConfig
}

type RateLimitConfig struct {
    RequestsPerSecond int           // 10 req/sec
    BurstSize         int           // 20 requests
    BackoffOnLimit    time.Duration // 1 second
}

func (r *ProviderRateLimiter) Wait(ctx context.Context) error {
    return r.limiter.Wait(ctx)
}
```

---

## 9. Testing Strategy

### 9.1 Unit Tests

```go
// Test passport verification
func TestVerifyPassport_Valid(t *testing.T) {
    verifier := setupTestVerifier(t)
    
    req := &PassportVerificationRequest{
        PassportNumber: "123456789",
        FirstName:      "John",
        LastName:       "Doe",
        DateOfBirth:    parseDate("1990-01-01"),
        IssueDate:      parseDate("2020-01-01"),
        ExpirationDate: parseDate("2030-01-01"),
        Nationality:    "US",
    }
    
    result, err := verifier.VerifyPassport(context.Background(), req)
    
    assert.NoError(t, err)
    assert.True(t, result.Verified)
    assert.Equal(t, DocumentTypePassport, result.DocumentType)
    assert.GreaterOrEqual(t, result.ConfidenceScore, 0.8)
}
```

### 9.2 Integration Tests

```go
// Test with real API provider (sandbox mode)
func TestVerifyPassport_PersonaSandbox(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    config := loadTestConfig(t)
    provider := persona.NewPersonaProvider(config.PersonaConfig)
    verifier := external.NewUSIdentityVerifier(&external.USIdentityVerifierConfig{
        PrimaryProvider: provider,
    })
    
    req := &PassportVerificationRequest{
        // Use Persona sandbox test data
        PassportNumber: "SANDBOX123456789",
        FirstName:      "Test",
        LastName:       "User",
        DateOfBirth:    parseDate("1990-01-01"),
        // ...
    }
    
    result, err := verifier.VerifyPassport(context.Background(), req)
    
    assert.NoError(t, err)
    assert.True(t, result.Verified)
}
```

### 9.3 State-Specific Tests

```go
// Table-driven tests for driver's license formats
func TestValidateDLFormat_StateVariations(t *testing.T) {
    tests := []struct {
        state          string
        licenseNumber  string
        expectedValid  bool
    }{
        {"CA", "A1234567", true},
        {"CA", "1234567", false},  // Missing letter
        {"TX", "12345678", true},
        {"TX", "A1234567", false}, // Should be digits only
        {"FL", "A123456789012", true},
        {"NY", "123456789", true},
        {"NY", "1234567890123456", true},
        // ... additional test cases
    }
    
    verifier := setupTestVerifier(t)
    
    for _, tt := range tests {
        t.Run(fmt.Sprintf("%s_%s", tt.state, tt.licenseNumber), func(t *testing.T) {
            valid := verifier.validateDLFormat(tt.state, tt.licenseNumber)
            assert.Equal(t, tt.expectedValid, valid)
        })
    }
}
```

---

## 10. Implementation Plan

### 10.1 Phase 1: Core Infrastructure (Week 1-2)

**Deliverables**:
1. ✅ Create `pkg/agentauth/external/us_identity_verifier.go`
2. ✅ Implement data models (request/response types)
3. ✅ Create API provider interface (`USIdentityAPIProvider`)
4. ✅ Implement state-specific validation (DL formats)
5. ✅ Integrate with PVPClient framework (circuit breaker, retry)
6. ✅ Create configuration system

**Estimated Effort**: 40-50 hours

### 10.2 Phase 2: API Provider Integration (Week 2-3)

**Deliverables**:
1. ✅ Implement Persona provider (`pkg/agentauth/external/providers/persona_provider.go`)
2. ✅ Implement Trulioo provider (fallback)
3. ✅ Create provider abstraction layer
4. ✅ Add caching layer
5. ✅ Implement rate limiting
6. ✅ Add metrics and monitoring

**Estimated Effort**: 40-50 hours

### 10.3 Phase 3: Testing and Validation (Week 3-4)

**Deliverables**:
1. ✅ Unit tests (80%+ coverage)
2. ✅ Integration tests (sandbox mode)
3. ✅ State-specific tests (all 50 states + DC)
4. ✅ Performance tests (load testing)
5. ✅ Security audit (PII handling, encryption)
6. ✅ Documentation

**Estimated Effort**: 30-40 hours

### 10.4 Phase 4: Production Deployment (Week 4)

**Deliverables**:
1. ✅ Production API keys (Persona, Trulioo)
2. ✅ Production configuration
3. ✅ Monitoring and alerting setup
4. ✅ Deployment to staging
5. ✅ Production deployment
6. ✅ Post-deployment validation

**Estimated Effort**: 20-30 hours

**Total Estimated Effort**: **130-170 hours (3-4 weeks)**

---

## 11. Monitoring and Observability

### 11.1 Metrics

```go
// Prometheus metrics for US identity verification
var (
    verificationAttempts = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "us_identity_verification_attempts_total",
            Help: "Total number of identity verification attempts",
        },
        []string{"document_type", "provider", "status"},
    )
    
    verificationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "us_identity_verification_duration_seconds",
            Help:    "Time taken for identity verification",
            Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
        },
        []string{"document_type", "provider"},
    )
    
    verificationErrors = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "us_identity_verification_errors_total",
            Help: "Total number of verification errors",
        },
        []string{"error_code", "provider"},
    )
    
    providerFallbacks = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "us_identity_provider_fallbacks_total",
            Help: "Number of times fallback provider was used",
        },
        []string{"primary_provider", "fallback_provider"},
    )
)
```

### 11.2 Logging

```go
// Structured logging (no PII)
log.Info("Identity verification attempt",
    "request_id", req.RequestID,
    "document_type", "passport",
    "provider", "persona",
    "has_image", req.PassportImageFront != nil,
)

// Success
log.Info("Identity verification succeeded",
    "request_id", req.RequestID,
    "verified", true,
    "confidence_score", result.ConfidenceScore,
    "processing_time_ms", result.ProcessingTimeMs,
)

// Failure
log.Warn("Identity verification failed",
    "request_id", req.RequestID,
    "error_code", ErrDocumentExpired,
    "provider", "persona",
)
```

### 11.3 Alerting

**Critical Alerts**:
- Error rate > 5% (5 minutes)
- Circuit breaker open for > 10 minutes
- API provider unavailable
- P95 latency > 10 seconds

**Warning Alerts**:
- Fallback provider usage > 10%
- Error rate > 2% (15 minutes)
- P95 latency > 5 seconds

---

## 12. Cost Analysis

### 12.1 API Provider Costs

**Persona** (Primary):
- Passport verification: $1.00 per verification
- Driver's license verification: $0.50 per verification
- State ID verification: $0.50 per verification
- Estimated monthly volume: 10,000 verifications
- **Estimated monthly cost**: $5,000 - $7,500

**Trulioo** (Fallback):
- All verification types: $1.50 per verification
- Estimated fallback usage: 5% (500 verifications)
- **Estimated monthly cost**: $750

**Total Estimated Monthly Cost**: **$5,750 - $8,250**

### 12.2 Infrastructure Costs

- Caching (Redis): $50/month
- Monitoring (Prometheus, Grafana): $100/month
- Storage (verification logs): $20/month
- **Total Infrastructure**: $170/month

**Total Estimated Cost**: **$5,920 - $8,420/month**

---

## 13. Success Criteria

### 13.1 Functional Requirements

- ✅ US passport verification functional
- ✅ Driver's license verification for all 50 states + DC
- ✅ SSN validation (format + death master file check)
- ✅ State ID verification
- ✅ Circuit breaker and retry logic functional
- ✅ Fallback provider working
- ✅ Integration with PVPClient framework

### 13.2 Non-Functional Requirements

- ✅ P95 latency < 5 seconds
- ✅ Error rate < 1%
- ✅ 80%+ test coverage
- ✅ Zero PII in logs
- ✅ Compliance with GDPR, CCPA
- ✅ Documentation complete

### 13.3 Compliance Metrics

- External connector compliance: 20% → 40% (+20%)
- US identity verification support: 0% → 100%
- AAP-001 PVP compliance: Enhanced with US support

---

## Appendices

### Appendix A: State Driver's License Formats (Complete List)

| State | Format | Example | Notes |
|-------|--------|---------|-------|
| AL | 1-8 digits | 1234567 | |
| AK | 1-7 digits | 1234567 | |
| AZ | 1 letter + 8 digits OR 2 letters + 2-5 digits | A12345678 | |
| AR | 4-9 digits | 123456789 | |
| CA | 1 letter + 7 digits | A1234567 | Most common format |
| CO | 9 digits OR 1-2 letters + 3-6 digits | 123456789 | |
| CT | 9 digits | 123456789 | |
| DE | 1-7 digits | 1234567 | |
| DC | 7 digits OR 9 digits | 1234567 | |
| FL | 1 letter + 12 digits | A123456789012 | |
| GA | 7-9 digits | 123456789 | |
| HI | 1 letter + 8 digits OR 9 digits | H12345678 | |
| ID | 2 letters + 6 digits + 1 letter OR 9 digits | AB123456C | |
| IL | 1 letter + 11 digits | A12345678901 | |
| IN | 1 letter + 9 digits OR 10 digits | A123456789 | |
| IA | 9 digits OR 3 digits + 2 letters + 4 digits | 123AB4567 | |
| KS | 1 letter + 8 digits OR 9 digits | K12345678 | |
| KY | 1 letter + 8 digits OR 9 digits | A12345678 | |
| LA | 1-9 digits | 123456789 | |
| ME | 7 digits OR 7 digits + X | 1234567 | X for extension |
| MD | 1 letter + 12 digits | A123456789012 | |
| MA | 1 letter + 8 digits OR 9 digits | S12345678 | S prefix common |
| MI | 1 letter + 12 digits | A123456789012 | |
| MN | 1 letter + 12 digits | A123456789012 | |
| MS | 9 digits | 123456789 | |
| MO | 1 letter + 5-9 digits OR 9 digits | A123456789 | |
| MT | 9 digits OR 13 digits | 123456789 | |
| NE | 1 letter + 3-8 digits | A12345678 | |
| NV | 9-10 digits OR X + 8 digits | X12345678 | |
| NH | 3 letters + 5 digits | ABC12345 | |
| NJ | 1 letter + 14 digits | A12345678901234 | Longest format |
| NM | 8-9 digits | 123456789 | |
| NY | 1 letter + 7 digits OR 1 letter + 18 digits OR 8 digits OR 9 digits OR 16 digits | A1234567 | Multiple formats |
| NC | 1-12 digits | 123456789012 | |
| ND | 3 letters + 6 digits OR 9 digits | ABC123456 | |
| OH | 2 letters + 6 digits OR 8 digits | AB123456 | |
| OK | 1 letter + 9 digits OR 9 digits | A123456789 | |
| OR | 1-9 digits | 123456789 | |
| PA | 8 digits | 12345678 | |
| RI | 7 digits OR 1 letter + 6 digits | 1234567 | |
| SC | 5-11 digits | 12345678901 | |
| SD | 6-10 digits | 1234567890 | |
| TN | 7-9 digits | 123456789 | |
| TX | 7-8 digits | 12345678 | |
| UT | 4-10 digits | 1234567890 | |
| VT | 8 digits OR 7 digits + A | 12345678 | |
| VA | 1 letter + 8 digits OR 9 digits | A12345678 | |
| WA | 1-7 letters + numbers (12 chars) | ABC1234DEF56 | Complex format |
| WV | 1-2 letters + 5-6 digits | AB12345 | |
| WI | 1 letter + 13 digits | A1234567890123 | |
| WY | 9-10 digits | 1234567890 | |

### Appendix B: SSN Validation Rules

**Format Validation**:
- 9 digits (no dashes in API)
- Cannot be all zeros
- Cannot be 000-XX-XXXX
- Cannot be XXX-00-XXXX
- Cannot be XXX-XX-0000
- Cannot be 666-XX-XXXX (never issued)
- Cannot be 900-999-XXXX (never issued for individuals)

**Death Master File (DMF)**:
- Check against Social Security Administration Death Master File
- Trulioo provides DMF checking

**Issuance Validation**:
- Area number (first 3 digits) indicates issuance location
- Group number (middle 2 digits) indicates issuance year range
- Serial number (last 4 digits) is sequential

---

## Conclusion

This architecture provides a comprehensive, secure, and scalable solution for US identity verification within the AgentAuth system. By leveraging commercial API providers (Persona primary, Trulioo fallback), integrating with the existing PVP framework, and implementing state-specific validation logic, the system will support all major US identity document types while maintaining high performance, reliability, and compliance with data protection regulations.

**Next Steps**:
1. Review and approve architecture
2. Select API provider (recommendation: Persona primary, Trulioo fallback)
3. Begin implementation (Phase 1: Core Infrastructure)
4. Iterate based on testing and feedback

**Estimated Timeline**: 3-4 weeks (130-170 hours)  
**Estimated Cost**: $5,920 - $8,420/month (API + infrastructure)
