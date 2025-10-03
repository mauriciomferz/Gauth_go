# GAuth RFC API Reference

**Official Gimel Foundation Implementation - Go Library API Documentation**

Complete Go library API reference for GiFo-RFC-0111 (GAuth 1.0) and GiFo-RFC-0115 (PoA Definition) implementation.

> **📝 Note**: This document covers the Go library API. For the web demonstration API documentation, see [COMPLETE_API_REFERENCE.md](./COMPLETE_API_REFERENCE.md) which includes both library and web API documentation.

## 📋 **Table of Contents**

1. [Core Service API](#core-service-api)
2. [RFC 111 Authorization API](#rfc-111-authorization-api)
3. [RFC 115 PoA Definition API](#rfc-115-poa-definition-api)
4. [Professional Foundation API](#professional-foundation-api)
5. [Data Types Reference](#data-types-reference)
6. [Error Handling](#error-handling)
7. [Examples](#examples)

> **🌐 Web API**: For REST API endpoints used by the webapp demo, see [COMPLETE_API_REFERENCE.md](./COMPLETE_API_REFERENCE.md)

## 🏗️ **Core Service API**

### **RFCCompliantService**

The main service implementing both RFC 111 and RFC 115 specifications.

```go
type RFCCompliantService struct {
    jwtService         *ProperJWTService
    legalValidator     *LegalFrameworkValidator
    delegationManager  *DelegationManager
    attestationService *AttestationService
}
```

#### **Constructor**

```go
func NewRFCCompliantService(issuer, audience string) (*RFCCompliantService, error)
```

**Parameters:**
- `issuer` (string): The issuer identifier for JWT tokens
- `audience` (string): The intended audience for JWT tokens

**Returns:**
- `*RFCCompliantService`: Configured service instance
- `error`: Configuration or initialization error

**Example:**
```go
service, err := auth.NewRFCCompliantService("my-company", "ai-authorization")
if err != nil {
    return fmt.Errorf("service creation failed: %w", err)
}
```

## 🎯 **RFC 111 Authorization API**

### **AuthorizeGAuth**

Main authorization method implementing complete RFC 111 flow with RFC 115 PoA Definition validation.

```go
func (s *RFCCompliantService) AuthorizeGAuth(ctx context.Context, req GAuthRequest) (*GAuthResponse, error)
```

**Parameters:**
- `ctx` (context.Context): Request context for cancellation and timeout
- `req` (GAuthRequest): Complete RFC-compliant authorization request

**Returns:**
- `*GAuthResponse`: Authorization response with compliance validation
- `error`: Authorization or validation error

**Process Flow:**
1. Validates PoA Definition (RFC 115)
2. Validates principal capacity (RFC 111)
3. Validates AI client capabilities
4. Validates legal compliance
5. Generates authorization code
6. Creates comprehensive audit record

**Example:**
```go
response, err := service.AuthorizeGAuth(ctx, auth.GAuthRequest{
    ClientID:     "ai_agent_v1",
    ResponseType: "code",
    Scope:        []string{"financial_advisory"},
    PowerType:    "financial_advisory_powers",
    PrincipalID:  "corp_ceo_123",
    AIAgentID:    "ai_financial_advisor",
    Jurisdiction: "US",
    PoADefinition: poaDefinition, // Complete RFC 115 structure
})
```

### **CreateGAuthToken** (Future Enhancement)

Exchange authorization code for extended token with comprehensive metadata.

```go
func (s *RFCCompliantService) CreateGAuthToken(ctx context.Context, authCode string) (*GAuthToken, error)
```

## 📋 **RFC 115 PoA Definition API**

### **PoA Definition Structure**

Complete implementation of RFC 115 Power-of-Attorney Credential Definition.

```go
type PoADefinition struct {
    Principal         Principal         `json:"principal"`          // Section 3.A
    Authorizer        Authorizer        `json:"authorizer"`         // Section 3.A
    Client           ClientAI          `json:"client"`             // Section 3.A
    AuthorizationType AuthorizationType `json:"authorization_type"` // Section 3.B
    ScopeDefinition   ScopeDefinition   `json:"scope_definition"`   // Section 3.B
    Requirements     Requirements      `json:"requirements"`       // Section 3.C
}
```

### **Section A: Parties**

#### **Principal**

```go
type Principal struct {
    Type         PrincipalType `json:"type"`                    // "individual" or "organization"
    Identity     string        `json:"identity"`                // Unique identifier
    Organization *Organization `json:"organization,omitempty"`  // Required if type is "organization"
}

type PrincipalType string
const (
    PrincipalTypeIndividual   PrincipalType = "individual"
    PrincipalTypeOrganization PrincipalType = "organization"
)
```

#### **Organization**

```go
type Organization struct {
    Type                OrganizationType `json:"type"`                 // Commercial, public, non-profit, etc.
    Name                string           `json:"name"`                 // Organization name
    RegisterEntry       string           `json:"register_entry"`       // Commercial register entry
    ManagingDirector    string           `json:"managing_director"`    // Current managing director
    RegisteredAuthority bool             `json:"registered_authority"` // Has registered authority
}

type OrganizationType string
const (
    OrgTypeCommercial   OrganizationType = "commercial_enterprise"    // AG, Ltd., partnership
    OrgTypePublic       OrganizationType = "public_authority"         // Federal, state, municipal
    OrgTypeNonProfit    OrganizationType = "non_profit_organization"  // Foundation, gGmbH
    OrgTypeAssociation  OrganizationType = "association"              // Non-profit or non-charitable
    OrgTypeOther        OrganizationType = "other"                    // Church, cooperative, etc.
)
```

#### **ClientAI**

```go
type ClientAI struct {
    Type              ClientType `json:"type"`               // LLM, agent, agentic AI, robot
    Identity          string     `json:"identity"`           // Unique AI identifier
    Version           string     `json:"version"`            // AI version
    OperationalStatus string     `json:"operational_status"` // "active", "revoked", etc.
}

type ClientType string
const (
    ClientTypeLLM       ClientType = "llm"            // Large Language Model
    ClientTypeAgent     ClientType = "digital_agent"  // Single digital agent
    ClientTypeAgenticAI ClientType = "agentic_ai"     // Team of agents
    ClientTypeRobot     ClientType = "humanoid_robot" // Physical humanoid robot
    ClientTypeOther     ClientType = "other"          // Other AI types
)
```

### **Section B: Authorization Type & Scope**

#### **AuthorizationType**

```go
type AuthorizationType struct {
    RepresentationType    RepresentationType `json:"representation_type"`     // Sole or joint
    RestrictionsExclusions []string          `json:"restrictions_exclusions"` // Specific exclusions
    SubProxyAuthority     bool              `json:"sub_proxy_authority"`     // Can delegate further
    SignatureType         SignatureType     `json:"signature_type"`          // Single, joint, collective
}

type RepresentationType string
const (
    RepresentationSole  RepresentationType = "sole"  // Sole representation
    RepresentationJoint RepresentationType = "joint" // Joint representation
)

type SignatureType string
const (
    SignatureSingle     SignatureType = "single"     // Single signature authority
    SignatureJoint      SignatureType = "joint"      // Joint signature required  
    SignatureCollective SignatureType = "collective" // Collective signature required
)
```

#### **ScopeDefinition**

```go
type ScopeDefinition struct {
    ApplicableSectors  []IndustrySector   `json:"applicable_sectors"`  // ISIC/NACE industry codes
    ApplicableRegions  []GeographicScope  `json:"applicable_regions"`  // Geographic constraints
    AuthorizedActions  AuthorizedActions  `json:"authorized_actions"`  // Permitted actions
}
```

#### **Industry Sectors (ISIC/NACE)**

```go
type IndustrySector string
const (
    SectorAgriculture     IndustrySector = "agriculture_forestry_fishing"
    SectorMining          IndustrySector = "mining_quarrying"
    SectorManufacturing   IndustrySector = "manufacturing"
    SectorEnergy          IndustrySector = "energy_supply"
    SectorWater           IndustrySector = "water_supply"
    SectorWaste           IndustrySector = "waste_management"
    SectorConstruction    IndustrySector = "construction"
    SectorTrade           IndustrySector = "trade"
    SectorTransport       IndustrySector = "transport_storage"
    SectorHospitality     IndustrySector = "hospitality"
    SectorICT             IndustrySector = "information_communication"
    SectorFinancial       IndustrySector = "financial_insurance"
    SectorRealEstate      IndustrySector = "real_estate"
    SectorProfessional    IndustrySector = "professional_scientific"
    SectorBusiness        IndustrySector = "business_services"
    SectorPublicAdmin     IndustrySector = "public_administration"
    SectorEducation       IndustrySector = "education"
    SectorHealth          IndustrySector = "health_social_work"
    SectorArts            IndustrySector = "arts_entertainment"
    SectorOtherServices   IndustrySector = "other_services"
)
```

#### **Geographic Scope**

```go
type GeographicScope struct {
    Type        GeographicType `json:"type"`                   // Global, national, regional, etc.
    Identifier  string         `json:"identifier"`             // Country code, region name, etc.
    Description string         `json:"description,omitempty"`  // Human-readable description
}

type GeographicType string
const (
    GeoTypeGlobal      GeographicType = "global"           // Global operations
    GeoTypeNational    GeographicType = "national"         // Specific country (specify identifier)
    GeoTypeRegional    GeographicType = "regional"         // DACH, Benelux, NAFTA, etc.
    GeoTypeSubnational GeographicType = "subnational"      // States, provinces, municipalities
    GeoTypeSpecific    GeographicType = "specific_location" // Specific branches or locations
)
```

#### **Authorized Actions**

```go
type AuthorizedActions struct {
    Transactions      []TransactionType   `json:"transactions"`        // Financial transactions
    Decisions         []DecisionType      `json:"decisions"`           // Business decisions
    NonPhysicalActions []NonPhysicalAction `json:"non_physical_actions"` // Information actions
    PhysicalActions   []PhysicalAction    `json:"physical_actions"`    // Physical world actions
}

// Transaction Types
type TransactionType string
const (
    TransactionLoan     TransactionType = "loan_transactions"
    TransactionPurchase TransactionType = "purchase_transactions"  
    TransactionSale     TransactionType = "sale_transactions"
    TransactionLease    TransactionType = "leasing_rental"
)

// Decision Types
type DecisionType string
const (
    DecisionPersonnel    DecisionType = "personnel_decisions"    // Hiring, dismissal, development
    DecisionFinancial    DecisionType = "financial_commitments"  // Contracts, expenses, investments
    DecisionBuySell      DecisionType = "buy_sell_transactions"  // Asset acquisition/disposal
    DecisionConceptual   DecisionType = "conceptual_determinations" // Business models, concepts
    DecisionDesign       DecisionType = "design_decisions"       // Branding, architecture
    DecisionInformation  DecisionType = "information_sharing"    // Data disclosure, PR
    DecisionStrategic    DecisionType = "strategic_decisions"    // M&A, partnerships, strategy
    DecisionLegal        DecisionType = "legal_proceedings"      // Legal actions
    DecisionAsset        DecisionType = "asset_management"       // Asset management decisions
)

// Non-Physical Actions
type NonPhysicalAction string
const (
    ActionSharing      NonPhysicalAction = "sharing_presenting"
    ActionBrainstorm   NonPhysicalAction = "brainstorming"
    ActionResearch     NonPhysicalAction = "researching_rag"    // Including RAG operations
)

// Physical Actions (primarily for humanoid robots)
type PhysicalAction string
const (
    ActionShipment     PhysicalAction = "shipments"      // Ocean, air, truck shipments
    ActionProduction   PhysicalAction = "production"     // Manufacturing processes
    ActionRecycling    PhysicalAction = "recycling"      // Recycling operations
    ActionStorage      PhysicalAction = "storage"        // Physical storage operations
    ActionCustomization PhysicalAction = "customization" // Product customization
    ActionPackage      PhysicalAction = "package"        // Packaging operations
    ActionClean        PhysicalAction = "clean"          // Cleaning operations
)
```

### **Section C: Requirements**

#### **Requirements Structure**

```go
type Requirements struct {
    ValidityPeriod    ValidityPeriod    `json:"validity_period"`     // Time constraints
    FormalRequirements FormalRequirements `json:"formal_requirements"` // Legal formalities
    PowerLimits       PowerLimits       `json:"power_limits"`        // Authority limitations
    SpecificRights    SpecificRights    `json:"specific_rights"`     // Rights and obligations
    SpecialConditions SpecialConditions `json:"special_conditions"`  // Special conditions
    DeathIncapacity   DeathIncapacity   `json:"death_incapacity"`    // Death/incapacity rules
    SecurityCompliance SecurityCompliance `json:"security_compliance"` // Security requirements
    JurisdictionLaw   JurisdictionLaw   `json:"jurisdiction_law"`    // Legal framework
    ConflictResolution ConflictResolution `json:"conflict_resolution"` // Dispute resolution
}
```

#### **ValidityPeriod**

```go
type ValidityPeriod struct {
    StartTime       time.Time      `json:"start_time"`        // Authorization start time
    EndTime         time.Time      `json:"end_time"`          // Authorization end time (max 1 year)
    TimeWindows     []TimeWindow   `json:"time_windows"`      // Operational time windows
    GeoConstraints  []string       `json:"geo_constraints"`   // Geographic restrictions
    SuspensionRules []string       `json:"suspension_rules"`  // Automatic suspension conditions
}

type TimeWindow struct {
    Start    string   `json:"start"`    // Start time (HH:MM format)
    End      string   `json:"end"`      // End time (HH:MM format)
    Timezone string   `json:"timezone"` // Timezone identifier
    Days     []string `json:"days"`     // Days of week (Mon, Tue, etc.)
}
```

#### **PowerLimits**

```go
type PowerLimits struct {
    PowerLevels        []PowerLevel     `json:"power_levels"`         // Amount and transaction limits
    InteractionBoundaries []string      `json:"interaction_boundaries"` // Data access, collaboration limits
    ToolLimitations    []string         `json:"tool_limitations"`     // Permitted tools and APIs
    OutcomeLimitations []string         `json:"outcome_limitations"`  // Intended outcome constraints
    ModelLimits        []ModelLimit     `json:"model_limits"`         // AI model restrictions
    BehavioralLimits   []string         `json:"behavioral_limits"`    // Action restrictions
    QuantumResistance  bool             `json:"quantum_resistance"`   // Require quantum-resistant crypto
    ExplicitExclusions []string         `json:"explicit_exclusions"`  // Explicitly forbidden actions
}

type PowerLevel struct {
    Type        string  `json:"type"`        // "amount", "transaction_type", etc.
    Limit       float64 `json:"limit"`       // Numerical limit value
    Currency    string  `json:"currency"`    // Currency code (if applicable)
    Description string  `json:"description"` // Human-readable description
}

type ModelLimit struct {
    ParameterCount   int64    `json:"parameter_count"`   // Maximum model parameters
    ReasoningMethods []string `json:"reasoning_methods"` // Permitted reasoning methods
    TrainingMethods  []string `json:"training_methods"`  // Permitted training approaches
    Description      string   `json:"description"`       // Limit description
}
```

#### **JurisdictionLaw**

```go
type JurisdictionLaw struct {
    Language           string   `json:"language"`             // Contract language
    GoverningLaw       string   `json:"governing_law"`        // Applicable legal framework
    PlaceOfJurisdiction string   `json:"place_of_jurisdiction"` // Legal jurisdiction
    AttachedDocuments  []string `json:"attached_documents"`   // Referenced legal documents
}
```

## 🔐 **Professional Foundation API**

### **ProperJWTService**

Development JWT implementation with RSA-256 signatures.

```go
type ProperJWTService struct {
    privateKey *rsa.PrivateKey
    publicKey  *rsa.PublicKey
    issuer     string
    audience   string
}

func NewProperJWTService(issuer, audience string) (*ProperJWTService, error)
func (s *ProperJWTService) CreateToken(subject string, scope []string, expiry time.Duration) (string, error)
func (s *ProperJWTService) ValidateToken(tokenString string) (*CustomClaims, error)
```

### **LegalFrameworkValidator**

Multi-jurisdiction legal validation.

```go
type LegalFrameworkValidator struct {
    supportedJurisdictions map[string]bool
    supportedEntityTypes   map[string]bool
    complianceRules       map[string][]string
}

func (v *LegalFrameworkValidator) ValidateFramework(ctx context.Context, framework LegalFramework) error
```

## 📊 **Response Types**

### **GAuthResponse**

```go
type GAuthResponse struct {
    AuthorizationCode string              `json:"code"`              // OAuth authorization code
    State            string              `json:"state"`             // CSRF protection state
    ExtendedToken    string              `json:"extended_token"`    // Optional immediate token
    LegalCompliance  bool                `json:"legal_compliance"`  // Legal validation result
    AuditRecordID    string              `json:"audit_record_id"`   // Audit trail identifier
    ExpiresIn        int                 `json:"expires_in"`        // Code expiration (seconds)
    Scope           []string            `json:"scope"`             // Granted scopes
    PoAValidation   PoAValidationResult `json:"poa_validation"`    // PoA validation details
}
```

### **PoAValidationResult**

```go
type PoAValidationResult struct {
    Valid             bool     `json:"valid"`              // Overall validation result
    ValidationErrors  []string `json:"validation_errors"`  // Specific validation errors
    ComplianceLevel   string   `json:"compliance_level"`   // "rfc115_compliant", etc.
    AttestationStatus string   `json:"attestation_status"` // "validated", "pending", etc.
}
```

### **GAuthToken**

```go
type GAuthToken struct {
    AccessToken      string           `json:"access_token"`      // JWT access token
    TokenType        string           `json:"token_type"`        // "bearer"
    ExpiresIn        int              `json:"expires_in"`        // Token expiration (seconds)
    Scope           []string         `json:"scope"`             // Token scopes
    ExtendedMetadata ExtendedMetadata `json:"extended_metadata"` // PoA metadata
}

type ExtendedMetadata struct {
    PowerType        string        `json:"power_type"`         // Type of power granted
    PrincipalID      string        `json:"principal_id"`       // Principal identifier
    AIAgentID        string        `json:"ai_agent_id"`        // AI agent identifier
    PoADefinition    PoADefinition `json:"poa_definition"`     // Complete PoA definition
    AttestationLevel string        `json:"attestation_level"`  // Attestation level achieved
    ComplianceProof  []string      `json:"compliance_proof"`   // Compliance validation proof
}
```

## 🚨 **Error Handling**

### **Error Types**

```go
// RFC validation errors
type RFCValidationError struct {
    Code    string `json:"code"`    // Error code (e.g., "invalid_principal")
    Message string `json:"message"` // Human-readable message
    Field   string `json:"field"`   // Field that caused the error
}

// Legal compliance errors
type LegalComplianceError struct {
    Jurisdiction string `json:"jurisdiction"` // Jurisdiction where error occurred
    Regulation   string `json:"regulation"`   // Specific regulation violated
    Description  string `json:"description"`  // Error description
}

// AI capability errors
type AICapabilityError struct {
    ClientID   string `json:"client_id"`   // AI client identifier
    Capability string `json:"capability"`  // Missing capability
    PowerType  string `json:"power_type"`  // Power type requiring capability
}
```

### **Common Error Codes**

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| `invalid_request` | Malformed request structure | 400 |
| `invalid_principal` | Principal validation failed | 400 |
| `invalid_client` | AI client validation failed | 400 |
| `unsupported_jurisdiction` | Jurisdiction not supported | 400 |
| `insufficient_capabilities` | AI lacks required capabilities | 403 |
| `excessive_power_limits` | Requested powers exceed limits | 403 |
| `legal_compliance_failure` | Legal framework validation failed | 403 |
| `invalid_poa_definition` | PoA Definition validation failed | 400 |

## 📖 **Usage Examples**

### **Complete RFC Implementation Example**

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/Gimel-Foundation/gauth/pkg/auth"
)

func main() {
    // Create service
    service, err := auth.NewRFCCompliantService("GlobalTech", "ai-authorization")
    if err != nil {
        panic(err)
    }
    
    // Create comprehensive PoA Definition
    poa := auth.PoADefinition{
        // A. Parties
        Principal: auth.Principal{
            Type:     auth.PrincipalTypeOrganization,
            Identity: "GlobalTech-Corp-2025",
            Organization: &auth.Organization{
                Type:                auth.OrgTypeCommercial,
                Name:                "GlobalTech Corporation",
                RegisterEntry:       "HRB-123456",
                ManagingDirector:    "Dr. Sarah Johnson",
                RegisteredAuthority: true,
            },
        },
        Client: auth.ClientAI{
            Type:              auth.ClientTypeAgenticAI,
            Identity:          "ai_financial_advisor_v3",
            Version:           "3.2.1-prod",
            OperationalStatus: "active",
        },
        
        // B. Authorization Type & Scope
        AuthorizationType: auth.AuthorizationType{
            RepresentationType: auth.RepresentationSole,
            SignatureType:      auth.SignatureSingle,
            SubProxyAuthority:  false,
        },
        ScopeDefinition: auth.ScopeDefinition{
            ApplicableSectors: []auth.IndustrySector{
                auth.SectorFinancial, auth.SectorICT,
            },
            ApplicableRegions: []auth.GeographicScope{
                {Type: auth.GeoTypeNational, Identifier: "US"},
                {Type: auth.GeoTypeRegional, Identifier: "EU"},
            },
            AuthorizedActions: auth.AuthorizedActions{
                Transactions: []auth.TransactionType{
                    auth.TransactionPurchase, auth.TransactionSale,
                },
                Decisions: []auth.DecisionType{
                    auth.DecisionFinancial, auth.DecisionInformation,
                },
                NonPhysicalActions: []auth.NonPhysicalAction{
                    auth.ActionResearch, auth.ActionSharing,
                },
            },
        },
        
        // C. Requirements
        Requirements: auth.Requirements{
            ValidityPeriod: auth.ValidityPeriod{
                StartTime: time.Now(),
                EndTime:   time.Now().Add(90 * 24 * time.Hour),
                TimeWindows: []auth.TimeWindow{
                    {Start: "09:00", End: "17:00", Timezone: "EST"},
                },
            },
            PowerLimits: auth.PowerLimits{
                PowerLevels: []auth.PowerLevel{
                    {Type: "transaction_value", Limit: 500000.0, Currency: "USD"},
                    {Type: "daily_limit", Limit: 1000000.0, Currency: "USD"},
                },
                QuantumResistance: true,
                ExplicitExclusions: []string{"cryptocurrency_trading"},
            },
            JurisdictionLaw: auth.JurisdictionLaw{
                Language:           "English",
                GoverningLaw:       "Delaware_Corporate_Law",
                PlaceOfJurisdiction: "US",
            },
        },
    }
    
    // Create GAuth request
    request := auth.GAuthRequest{
        ClientID:     "ai_financial_advisor_v3",
        ResponseType: "code",
        Scope:        []string{"financial_advisory", "asset_management"},
        State:        "secure_state_token_2025",
        PowerType:    "financial_advisory_powers",
        PrincipalID:  "GlobalTech-Corp-2025",
        AIAgentID:    "ai_financial_advisor_v3",
        Jurisdiction: "US",
        PoADefinition: poa,
    }
    
    // Authorize with full RFC validation
    response, err := service.AuthorizeGAuth(context.Background(), request)
    if err != nil {
        fmt.Printf("❌ Authorization failed: %v\n", err)
        return
    }
    
    fmt.Printf("✅ Authorization successful!\n")
    fmt.Printf("Authorization Code: %s\n", response.AuthorizationCode[:20]+"...")
    fmt.Printf("Legal Compliance: %v\n", response.LegalCompliance)
    fmt.Printf("Compliance Level: %s\n", response.PoAValidation.ComplianceLevel)
    fmt.Printf("Attestation Status: %s\n", response.PoAValidation.AttestationStatus)
    fmt.Printf("Audit Record: %s\n", response.AuditRecordID)
}
```

---

*This API reference provides complete documentation for the official Gimel Foundation GAuth RFC implementation. For additional examples and guides, see the [Getting Started Guide](../docs/GETTING_STARTED.md) and [RFC Architecture Documentation](../docs/RFC_ARCHITECTURE.md).*