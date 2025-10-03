# Getting Started with GAuth RFC Implementation

**🏗️ DEVELOPMENT PROTOTYPE** | **🏆 RFC-0115 COMPLETE** | **🏢 GIMEL FOUNDATION**

**Copyright (c) 2025 Gimel Foundation gGmbH i.G.**  
Licensed under Apache 2.0

**Gimel Foundation gGmbH i.G.**, www.GimelFoundation.com  
Operated by Gimel Technologies GmbH  
MD: Bjørn Baunbæk, Dr. Götz G. Wehberg – Chairman of the Board: Daniel Hartert  
Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.GimelID.com

This guide will help you get started with the official Gimel Foundation GAuth implementation, featuring complete GiFo-RFC-0115 PoA-Definition compliance.

## 🚀 Quick Installation

### 1. **Install the Package**

```bash
go get github.com/Gimel-Foundation/gauth
```

### 2. **Build and Test**

```bash
# Clone the repository
git clone https://github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0
cd GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0

# Build the package
go build ./pkg/auth

# Run RFC compliance tests
go run examples/official_rfc_compliance_test/main.go
```

## 🎯 **First RFC 111 Implementation**

### **Basic GAuth Authorization**

Create your first RFC-compliant authorization:

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/Gimel-Foundation/gauth/pkg/auth"
)

func main() {
    // 1. Create RFC-compliant service
    service, err := auth.NewRFCCompliantService("my-company", "ai-authorization")
    if err != nil {
        panic(err)
    }
    
    // 2. Create basic PoA Definition
    poa := auth.PoADefinition{
        Principal: auth.Principal{
            Type:     auth.PrincipalTypeIndividual,
            Identity: "john_doe_ceo",
        },
        Client: auth.ClientAI{
            Type:              auth.ClientTypeAgent,
            Identity:          "my_ai_assistant",
            Version:           "1.0.0",
            OperationalStatus: "active",
        },
        ScopeDefinition: auth.ScopeDefinition{
            ApplicableSectors: []auth.IndustrySector{auth.SectorBusiness},
            ApplicableRegions: []auth.GeographicScope{
                {Type: auth.GeoTypeNational, Identifier: "US"},
            },
            AuthorizedActions: auth.AuthorizedActions{
                Decisions: []auth.DecisionType{auth.DecisionInformation},
            },
        },
        Requirements: auth.Requirements{
            ValidityPeriod: auth.ValidityPeriod{
                StartTime: time.Now(),
                EndTime:   time.Now().Add(30 * 24 * time.Hour), // 30 days
            },
            JurisdictionLaw: auth.JurisdictionLaw{
                GoverningLaw:       "US_Federal_Law",
                PlaceOfJurisdiction: "US",
            },
        },
    }
    
    // 3. Create GAuth request
    request := auth.GAuthRequest{
        ClientID:     "my_ai_assistant",
        ResponseType: "code",
        Scope:        []string{"information_sharing"},
        PowerType:    "data_management",
        PrincipalID:  "john_doe_ceo",
        AIAgentID:    "my_ai_assistant",
        Jurisdiction: "US",
        PoADefinition: poa,
    }
    
    // 4. Authorize with RFC validation
    response, err := service.AuthorizeGAuth(context.Background(), request)
    if err != nil {
        fmt.Printf("❌ Authorization failed: %v\n", err)
        return
    }
    
    fmt.Printf("✅ Authorization successful!\n")
    fmt.Printf("Authorization Code: %s\n", response.AuthorizationCode[:20]+"...")
    fmt.Printf("Compliance Level: %s\n", response.PoAValidation.ComplianceLevel)
    fmt.Printf("Legal Compliance: %v\n", response.LegalCompliance)
}
```

## 🏢 **Corporate Implementation Example**
cd examples/basic
go run main.go
```

This demonstrates:
- Authorization request and grant
- JWT token issuance
- Token validation
- Transaction processing

### 2. **Test Rate Limiting**

Run the rate limiting example:
```bash
cd examples/rate
go run main.go
```

Watch how different patterns affect the rate limits:
- Burst requests
- Steady traffic
- Multiple clients

### 3. **Explore Token Management**

Try the token management example:
```bash
cd examples/token
go run main.go
```

See how tokens are:
- Created and validated
- Stored and retrieved
- Automatically cleaned up

## Manual Testing

1. **Authentication Flow**
```bash
# Request a token
curl -X POST http://localhost:8080/auth \
  -d '{"username": "test", "password": "test123"}'

# Use the token
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/protected
```

2. **Rate Limiting**
```bash
# Make rapid requests to see rate limiting
for i in {1..10}; do
  curl -H "Authorization: Bearer <token>" \
    http://localhost:8080/protected
done
```

3. **Token Management**
```bash
# Create a token
curl -X POST http://localhost:8080/token/create

# Validate a token
curl -X POST http://localhost:8080/token/validate \
  -d '{"token": "<token>"}'

# Revoke a token
curl -X POST http://localhost:8080/token/revoke \
  -d '{"token": "<token>"}'
```

## Configuration Examples

1. **Basic Setup**
```go
auth := gauth.New(gauth.Config{
    AuthServerURL: "https://auth.example.com",
    ClientID:     "client-123",
    ClientSecret: "secret-456",
})
```

2. **With Rate Limiting**
```go
auth := gauth.New(gauth.Config{
    // ... basic config ...
    RateLimit: gauth.RateLimitConfig{
        RequestsPerSecond: 100,
        WindowSize:       60,
        BurstSize:       10,
    },
})
```

3. **With Custom Token Store**
```go
auth := gauth.New(gauth.Config{
    // ... basic config ...
    TokenStore: myCustomStore,
})
```

## Monitoring

1. **Check Rate Limit Status**
```bash
curl http://localhost:8080/metrics | grep rate_limit
```

2. **View Token Statistics**
```bash
curl http://localhost:8080/metrics | grep token
```

3. **Monitor Authentication Events**
```bash
curl http://localhost:8080/metrics | grep auth
```

## Troubleshooting

1. **Rate Limit Issues**
- Check the current rate limit status
- Verify client identification
- Review window size settings

2. **Token Problems**
- Verify token format
- Check expiration times
- Confirm scope configuration

3. **Authentication Failures**
- Review credentials
- Check server connectivity
- Verify client configuration

## Next Steps

1. Read the [Development Guide](DEVELOPMENT.md) for implementation details
2. Explore the [API Documentation](pkg/gauth/doc.go)
3. Try the [Advanced Examples](examples/advanced/)

## Community Resources

- GitHub Issues: Report bugs and request features
- Discussions: Ask questions and share ideas
- Wiki: Additional documentation and guides