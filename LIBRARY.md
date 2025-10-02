# GAuth Library Documentation

**Official Gimel Foundation RFC Implementation - Library Guide**

**Copyright (c) 2025 Gimel Foundation gGmbH i.G.**  
Licensed under Apache 2.0

**Gimel Foundation gGmbH i.G.**, www.GimelFoundation.com  
Operated by Gimel Technologies GmbH  
MD: Bjørn Baunbæk, Dr. Götz G. Wehberg – Chairman of the Board: Daniel Hartert  
Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.GimelID.com

Complete library documentation for GiFo-RFC-0111 (GAuth 1.0) and GiFo-RFC-0115 (PoA-Definition) implementation. 

**🏗️ Development Prototype** - RFC-0115 implementation complete, security implementations are mock/stub for demonstration.

## 📋 **Table of Contents**

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Core Components](#core-components)
4. [RFC 115 PoA Definition](#rfc-115-poa-definition)
5. [Authorization Flow](#authorization-flow)
6. [Advanced Usage](#advanced-usage)
7. [Error Handling](#error-handling)
8. [Best Practices](#best-practices)

## 🚀 **Installation**

```bash
go get github.com/mauriciomferz/Gauth_go
```

**Requirements:**
- Go 1.21+
- Professional-grade security standards
- Production environment ready

## ⚡ **Quick Start**

### **Basic RFC-Compliant Authorization**

```go
package main

import (
    "context"
    "log"
    "time"
    "github.com/mauriciomferz/Gauth_go/pkg/auth"
)

func main() {
    // 1. Create RFC-compliant service
    service, err := auth.NewRFCCompliantService("YourCompany", "ai-authorization")
    if err != nil {
        log.Fatalf("Service creation failed: %v", err)
    }

    // 2. Define PoA Definition (RFC 115 compliant)
    poa := auth.PoADefinition{
        // A. Parties
        Principal: auth.Principal{
            Type:     auth.PrincipalTypeOrganization,
            Identity: "your-company-2025",
            Organization: &auth.Organization{
                Type:                auth.OrgTypeCommercial,
                Name:                "Your Company Inc.",
                RegisterEntry:       "DE-123456789",
                ManagingDirector:    "John Smith",
                RegisteredAuthority: true,
            },
        },
        Client: auth.ClientAI{
            Type:              auth.ClientTypeAgent,
            Identity:          "ai_assistant_v1",
            Version:           "1.0.0",
            OperationalStatus: "active",
        },
        
        // B. Authorization Type & Scope
        AuthorizationType: auth.AuthorizationType{
            RepresentationType: auth.RepresentationSole,
            SignatureType:      auth.SignatureSingle,
            SubProxyAuthority:  false,
        },
        ScopeDefinition: auth.ScopeDefinition{
            ApplicableSectors: []auth.IndustrySector{auth.SectorProfessional},
            ApplicableRegions: []auth.GeographicScope{
                {Type: auth.GeoTypeNational, Identifier: "US"},
            },
            AuthorizedActions: auth.AuthorizedActions{
                NonPhysicalActions: []auth.NonPhysicalAction{
                    auth.ActionResearch, 
                    auth.ActionSharing,
                },
                Decisions: []auth.DecisionType{
                    auth.DecisionInformation,
                },
            },
        },
        
        // C. Requirements
        Requirements: auth.Requirements{
            ValidityPeriod: auth.ValidityPeriod{
                StartTime: time.Now(),
                EndTime:   time.Now().Add(24 * time.Hour),
                TimeWindows: []auth.TimeWindow{
                    {Start: "09:00", End: "17:00", Timezone: "UTC"},
                },
            },
            PowerLimits: auth.PowerLimits{
                PowerLevels: []auth.PowerLevel{
                    {Type: "document_access", Limit: 1000},
                },
                QuantumResistance: true,
                ExplicitExclusions: []string{"financial_transactions"},
            },
            JurisdictionLaw: auth.JurisdictionLaw{
                Language:           "English",
                GoverningLaw:       "US_Federal_Law",
                PlaceOfJurisdiction: "US",
            },
        },
    }

    // 3. Create authorization request
    request := auth.GAuthRequest{
        ClientID:      "ai_assistant_v1",
        ResponseType:  "code",
        Scope:         []string{"research", "document_analysis"},
        State:         "secure-random-state-123",
        PowerType:     "research_assistant",
        PrincipalID:   "your-company-2025",
        AIAgentID:     "ai_assistant_v1",
        Jurisdiction:  "US",
        PoADefinition: poa,
    }

    // 4. Authorize with full RFC validation
    response, err := service.AuthorizeGAuth(context.Background(), request)
    if err != nil {
        log.Fatalf("Authorization failed: %v", err)
    }

    // 5. Success! Use the authorization
    log.Printf("✅ RFC Authorization successful!")
    log.Printf("Authorization Code: %s", response.AuthorizationCode[:16]+"...")
    log.Printf("Legal Compliance: %v", response.LegalCompliance)
    log.Printf("Compliance Level: %s", response.PoAValidation.ComplianceLevel)
    log.Printf("Attestation Status: %s", response.PoAValidation.AttestationStatus)
    log.Printf("Audit Record: %s", response.AuditRecordID)
}
```

## 🏗️ **Core Components**

### **RFCCompliantService**

Main service implementing both RFC 111 and RFC 115 specifications:

```go
// Create service
service, err := auth.NewRFCCompliantService("issuer", "audience")

// Features:
// - Complete RFC 111 authorization flow
// - RFC 115 PoA Definition validation
// - Professional JWT foundation (RSA-256)
// - Multi-jurisdiction legal compliance
// - Comprehensive audit logging
// - ⚠️ NO SECURITY - Mock implementation only
```

### **Professional JWT Foundation**

Built on mock JWT implementation (no real security):

```go
// Automatic features:
// - RSA-256 signatures
// - Argon2id password hashing
// - ChaCha20-Poly1305 encryption
// - Quantum-resistant crypto options
// - Professional key management
```

### **Legal Framework Validator**

Multi-jurisdiction compliance validation:

```go
// Supported jurisdictions:
// - United States (US Federal Law)
// - European Union (GDPR, AI Act)
// - Canada (PIPEDA, AI regulations)
// - United Kingdom (Data Protection Act)
// - Australia (Privacy Act, AI Ethics)
```

## 📋 **RFC 115 PoA Definition**

Complete implementation of RFC 115 Power-of-Attorney Credential Definition with 3-section structure per official specification.

### **Section A: Parties**

```go
// Principal (who grants authority)
Principal: auth.Principal{
    Type:     auth.PrincipalTypeOrganization, // or PrincipalTypeIndividual
    Identity: "unique-principal-id",
    Organization: &auth.Organization{
        Type:                auth.OrgTypeCommercial,  // AG, Ltd., partnership
        Name:                "Company Name",
        RegisterEntry:       "Commercial-Register-123",
        ManagingDirector:    "Director Name",
        RegisteredAuthority: true,
    },
}

// AI Client (who receives authority)
Client: auth.ClientAI{
    Type:              auth.ClientTypeAgent,     // LLM, agent, agentic AI, robot
    Identity:          "ai-system-identifier",
    Version:           "1.0.0",
    OperationalStatus: "active",                 // active, revoked, suspended
}
```

### **Section B: Authorization Type & Scope**

```go
// Authorization type configuration
AuthorizationType: auth.AuthorizationType{
    RepresentationType:    auth.RepresentationSole,  // Sole or joint
    RestrictionsExclusions: []string{"crypto_trading"},
    SubProxyAuthority:     false,                    // Can delegate further
    SignatureType:         auth.SignatureSingle,     // Single, joint, collective
}

// Scope definition with industry sectors (ISIC/NACE codes)
ScopeDefinition: auth.ScopeDefinition{
    ApplicableSectors: []auth.IndustrySector{
        auth.SectorFinancial,    // Financial & insurance
        auth.SectorICT,          // Information & communication
        auth.SectorProfessional, // Professional services
        // ... 20 industry sectors supported
    },
    ApplicableRegions: []auth.GeographicScope{
        {Type: auth.GeoTypeNational, Identifier: "US"},
        {Type: auth.GeoTypeRegional, Identifier: "EU"},
        // Global, national, regional, subnational, specific
    },
    AuthorizedActions: auth.AuthorizedActions{
        Transactions: []auth.TransactionType{
            auth.TransactionPurchase, // Purchase transactions
            auth.TransactionSale,     // Sale transactions
        },
        Decisions: []auth.DecisionType{
            auth.DecisionFinancial,   // Financial commitments
            auth.DecisionPersonnel,   // Personnel decisions
            auth.DecisionStrategic,   // Strategic decisions
        },
        NonPhysicalActions: []auth.NonPhysicalAction{
            auth.ActionResearch,      // Research & RAG operations
            auth.ActionSharing,       // Information sharing
            auth.ActionBrainstorm,    // Brainstorming sessions
        },
        PhysicalActions: []auth.PhysicalAction{
            auth.ActionProduction,    // Manufacturing (for robots)
            auth.ActionShipment,      // Logistics operations
        },
    },
}
```

### **Section C: Requirements**

```go
Requirements: auth.Requirements{
    // Validity period (max 1 year)
    ValidityPeriod: auth.ValidityPeriod{
        StartTime:   time.Now(),
        EndTime:     time.Now().Add(365 * 24 * time.Hour),
        TimeWindows: []auth.TimeWindow{
            {Start: "09:00", End: "17:00", Timezone: "UTC", Days: []string{"Mon", "Tue", "Wed", "Thu", "Fri"}},
        },
        GeoConstraints: []string{"US", "CA"},
    },
    
    // Power limits and restrictions
    PowerLimits: auth.PowerLimits{
        PowerLevels: []auth.PowerLevel{
            {Type: "transaction_value", Limit: 100000.0, Currency: "USD"},
            {Type: "daily_transactions", Limit: 50},
        },
        ModelLimits: []auth.ModelLimit{
            {ParameterCount: 1000000000, ReasoningMethods: []string{"chain_of_thought"}},
        },
        QuantumResistance:  true,
        ExplicitExclusions: []string{"nuclear_operations", "weapons_systems"},
    },
    
    // Legal jurisdiction
    JurisdictionLaw: auth.JurisdictionLaw{
        Language:           "English",
        GoverningLaw:       "Delaware_Corporate_Law",
        PlaceOfJurisdiction: "US",
        AttachedDocuments:  []string{"corporate_bylaws.pdf"},
    },
}
```

## 🔄 **Authorization Flow**

### **1. Service Creation**

```go
service, err := auth.NewRFCCompliantService("YourCompany", "ai-systems")
if err != nil {
    return fmt.Errorf("service setup failed: %w", err)
}
```

### **2. PoA Definition Creation**

```go
poa := auth.PoADefinition{
    // Complete 3-section structure per RFC 115
    Principal:         /* Section A */,
    Client:           /* Section A */,
    AuthorizationType: /* Section B */,
    ScopeDefinition:   /* Section B */,
    Requirements:     /* Section C */,
}
```

### **3. Authorization Request**

```go
request := auth.GAuthRequest{
    ClientID:      "ai-client-id",
    ResponseType:  "code",
    Scope:         []string{"required", "scopes"},
    State:         "csrf-protection-token",
    PowerType:     "power-category",
    PrincipalID:   "principal-identifier",
    AIAgentID:     "ai-agent-identifier",
    Jurisdiction:  "US",
    PoADefinition: poa,
}
```

### **4. Authorization Execution**

```go
response, err := service.AuthorizeGAuth(ctx, request)
if err != nil {
    // Handle RFC validation errors
    return fmt.Errorf("authorization failed: %w", err)
}

// Success - use response.AuthorizationCode
```

### **5. Response Validation**

```go
if !response.LegalCompliance {
    return errors.New("legal compliance failed")
}

if response.PoAValidation.ComplianceLevel != "rfc115_compliant" {
    return errors.New("RFC 115 compliance failed")
}

// Authorization successful - proceed with AI operations
```

## 📚 **Additional Resources**

- **[API Reference](docs/API_REFERENCE.md)** - Complete API documentation
- **[Getting Started Guide](docs/GETTING_STARTED.md)** - Detailed tutorials
- **[RFC Architecture](docs/RFC_ARCHITECTURE.md)** - Technical architecture
- **[Examples Directory](examples/)** - Working code examples
- **[Security Guide](SECURITY.md)** - Security implementation details

---

*This library provides a development implementation of Gimel Foundation RFCs 111 and 115, with basic security and compliance validation for demonstration purposes. For official RFC specifications, visit the Gimel Foundation documentation.*
