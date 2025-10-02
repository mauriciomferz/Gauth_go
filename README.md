# GAuth: Go Authorization Framework

**🏗️ Development Prototype** | ✅ **Basic Tests Passing** | ⚠️ **Not Ready for Production** | 📚 **Security Research Project**

Official Go implementation of the combined Gimel Foundation gGmbH i.G. authorization specifications.

**Copyright (c) 2025 Gimel Foundation gGmbH i.G.**  
Licensed under Apache 2.0

---

## 🏢 Gimel Foundation Details

**Gimel Foundation gGmbH i.G.**, www.GimelFoundation.com  
**Operated by**: Gimel Technologies GmbH  
**Leadership**: MD: Bjørn Baunbæk, Dr. Götz G. Wehberg | Chairman of the Board: Daniel Hartert  
**Location**: Hardtweg 31, D-53639 Königswinter, Germany  
**Registration**: Siegburg HRB 18660  
**Additional Information**: www.GimelID.com

---

## 🎯 Combined RFC Implementation Status

| RFC Standard | Implementation Status | Documentation |
|--------------|----------------------|---------------|
| **🔥 Combined RFC-0111 + RFC-0115** | ✅ **UNIFIED IMPLEMENTATION** | [Combined Demo](examples/combined_rfc_demo/) |
| **GiFo-RFC-0111** | ✅ **COMPLETE** - GAuth 1.0 Authorization Framework | [Individual Implementation](examples/official_rfc0111_implementation/) |
| **GiFo-RFC-0115** | ✅ **COMPLETE** - PoA-Definition | [Individual Implementation](examples/rfc_0115_poa_definition/) |

### 🚀 **NEW: Combined RFC-0111 & RFC-0115 Implementation** ⭐

✅ **Unified Framework**: Single API combining both RFC specifications  
✅ **Complete Integration**: GAuth 1.0 + PoA-Definition in one comprehensive system  
✅ **Enhanced AI Governance**: Power-of-Attorney for AI systems with legal framework  
✅ **Full Compliance**: Both RFC specifications with mandatory exclusions enforced  
✅ **Type-Safe**: Comprehensive Go type system for enterprise deployment  
✅ **Complete Architecture**: OAuth 2.0, OpenID Connect, MCP integration  

```bash
cd examples/combined_rfc_demo
go run main.go
```

### RFC-0111 GAuth 1.0 Authorization Framework Features

✅ **Complete P*P Architecture**: Power Decision/Information/Administration/Verification Points  
✅ **Extended Token System**: Comprehensive authorization scope and duration management  
✅ **AI Client Support**: Digital agents, agentic AI, humanoid robots  
✅ **Mandatory Exclusions**: Web3, AI operators, DNA identities excluded (Section 2)  
✅ **Official Compliance**: ISBN 978-3-00-084039-5, Standards Track Document  

### RFC-0115 PoA-Definition Features

✅ **Section 3.A - Parties**: Principal, Representative, AuthorizedClient  
✅ **Section 3.B - Authorization Scope**: Types, Sectors, Regions, Actions  
✅ **Section 3.C - Requirements**: Validity, Formal Requirements, Power Limits, Security Compliance  
✅ **Legal Framework**: Multi-jurisdiction support with quantum resistance  
✅ **Working Demo**: [examples/rfc_0115_poa_definition/](examples/rfc_0115_poa_definition/)

---

## 📋 Current Project Status

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org/doc/devel/release.html)
[![RFC Compliance](https://img.shields.io/badge/RFC-0115%20Complete-green.svg)](./examples/rfc_0115_poa_definition/)
[![Build Status](https://img.shields.io/badge/Build-✅%20Passing-brightgreen.svg)](#quick-start)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](./LICENSE)
[![Gimel Foundation](https://img.shields.io/badge/Gimel%20Foundation-Official%20Implementation-gold.svg)](https://www.GimelFoundation.com)

### **What This Project Represents:**

## 🎯 What You Get

- 🎯 **RFC-0111 Implementation**: Complete GAuth 1.0 framework with P*P architecture
- 🎯 **RFC-0115 Implementation**: Complete PoA-Definition structure implementation
- 🎯 **Combined Framework**: Unified API for comprehensive AI authorization
- 🏗️ **Architecture Design**: Well-designed authentication system architecture
- 📚 **Educational Value**: Comprehensive example of RFC compliance implementation
- ⚠️ **Development Status**: Prototype with honest security disclaimers

### **Production Readiness:**

- ✅ **RFC-0115 Compliance**: Complete PoA-Definition implementation
- ✅ **Type Safety**: Full Go type system enforcement  
- ✅ **Documentation**: Comprehensive examples and guides
- ⚠️ **Security**: Requires real cryptography for production use
- ⚠️ **Authentication**: Mock implementations need production replacement

---

## 🚀 Quick Start

### 1. Clone and Setup

```bash
git clone https://github.com/Gimel-Foundation/gauth.git
cd gauth
go mod tidy
```

### 2. RFC-0115 PoA-Definition Demo

```bash
cd examples/rfc_0115_poa_definition
go run main.go
```

This demonstrates the complete RFC-0115 PoA-Definition structure with:
- Gimel Foundation gGmbH i.G. as Principal (Non-profit organization)
- Daniel Hartert as Representative (Chairman with registered PoA)
- AI Client authorization with comprehensive security requirements
- Full compliance with German Federal Law and EU regulations

### 3. Web Application Demo

```bash
# Start backend (Terminal 1)
cd gauth-demo-app/web/backend
go run main.go

# Start frontend (Terminal 2)  
cd gauth-demo-app/web
python3 -m http.server 3000
```

**Access**: http://localhost:3000 (Frontend) | http://localhost:8080 (Backend API)

---

## 📖 Documentation

### Core Documentation
- [**Getting Started**](docs/GETTING_STARTED.md) - Complete setup and usage guide
- [**Architecture**](docs/ARCHITECTURE.md) - System design and structure  
- [**RFC Architecture**](docs/RFC_ARCHITECTURE.md) - RFC-0111 & RFC-0115 compliance
- [**Library Usage**](LIBRARY.md) - Integration as a Go library
- [**Security**](SECURITY.md) - Security model and limitations

### Implementation Guides
- [**RFC-0115 Implementation**](docs/RFC_0115_IMPLEMENTATION_SUMMARY.md) - Complete PoA-Definition guide
- [**Examples**](docs/EXAMPLES.md) - Usage examples and patterns
- [**Testing Guide**](docs/TESTING.md) - Testing strategies and validation
- [**Troubleshooting**](docs/TROUBLESHOOTING.md) - Common issues and solutions

### Technical Reference
- [**API Reference**](docs/API_REFERENCE.md) - Complete API documentation
- [**Performance**](docs/PERFORMANCE.md) - Performance characteristics
- [**Benchmarks**](docs/BENCHMARKS.md) - Performance benchmarks

---

## 🔧 Core Features

### RFC-0115 PoA-Definition Implementation
```go
import "github.com/Gimel-Foundation/gauth/pkg/poa"

// Complete PoA-Definition structure
poaDefinition := &poa.PoADefinition{
    Parties: poa.Parties{
        Principal: poa.Principal{
            Type: poa.PrincipalTypeOrganization,
            Organization: &poa.Organization{
                Type: poa.OrgTypeNonProfit,
                Name: "Gimel Foundation gGmbH i.G.",
                // ... complete structure
            },
        },
        // ... Representatives and AuthorizedClient
    },
    Authorization: poa.AuthorizationScope{
        // ... Complete authorization scope
    },
    Requirements: poa.Requirements{
        // ... Complete requirements structure
    },
}
```

### Authentication & Authorization
```go
import "github.com/Gimel-Foundation/gauth/pkg/gauth"

// Create service with comprehensive configuration
service := gauth.NewService(gauth.Config{
    TokenStore:      store.NewMemoryStore(),
    AuditLogger:     audit.NewStructuredLogger(),
    EventPublisher:  events.NewPublisher(),
    // ... additional configuration
})

// Token operations with full audit trail
token, err := service.GrantToken(ctx, request)
```

### Event System
```go
import "github.com/Gimel-Foundation/gauth/pkg/events"

// Type-safe event handling
publisher := events.NewPublisher()
publisher.Subscribe(func(event *events.TokenGrantedEvent) {
    // Handle token granted event
})
```

---

## 🏗️ Architecture

### Modular Package Structure
```
pkg/
├── auth/          # Authentication primitives
├── authz/         # Authorization logic  
├── poa/           # RFC-0115 PoA-Definition ✅
├── token/         # Token management
├── events/        # Event system
├── audit/         # Audit and logging
├── store/         # Pluggable storage
├── rate/          # Rate limiting
├── resilience/    # Circuit breakers
└── monitoring/    # Metrics and monitoring
```

### Key Design Principles
- **RFC Compliance**: Strict adherence to GiFo-RFC-0111 & RFC-0115
- **Type Safety**: Comprehensive Go type system usage
- **Modularity**: Independent, reusable packages
- **Extensibility**: Plugin architecture for customization
- **Observability**: Comprehensive logging, metrics, tracing

---

## ⚠️ Security Notice

**This is a development prototype with the following limitations:**

### 🏗️ Development Status
- **Cryptography**: All JWT signing uses stub implementations
- **Authentication**: Mock user verification and password handling  
- **Key Management**: No secure key storage or rotation
- **Authorization**: Basic RBAC without policy enforcement
- **Audit**: Logging without tamper protection

### ✅ Production Requirements
For production deployment, implement:

1. **Real Cryptography**: Replace stub JWT implementations with production libraries
2. **Secure Authentication**: Multi-factor authentication, secure password hashing
3. **Key Management**: HSM or secure key management service integration  
4. **Authorization**: Full RBAC with policy engines (OPA, Cedar, etc.)
5. **Compliance**: Real regulatory compliance validation
6. **Infrastructure**: Rate limiting, DDoS protection, security monitoring

**Estimated Production Implementation**: 6-15 months, $6-15M budget, 15-25 specialists

---

## 🧪 Testing

### Run All Tests
```bash
# Unit tests
go test ./...

# Integration tests  
make test-integration

# RFC-0115 compliance test
cd examples/rfc_0115_poa_definition && go run main.go
```

### Test Coverage
- ✅ **Unit Tests**: Core package functionality
- ✅ **Integration Tests**: End-to-end workflows
- ✅ **RFC Compliance**: RFC-0115 structure validation
- ⚠️ **Security Tests**: Mock implementations only

---

## 📈 Performance

### Benchmarks
- **Token Operations**: ~100k/sec (in-memory store)
- **Event Processing**: ~50k events/sec  
- **Memory Usage**: ~10MB baseline
- **Latency**: <1ms (95th percentile)

See [BENCHMARKS.md](docs/BENCHMARKS.md) for detailed performance analysis.

---

## 🤝 Contributing

1. **Read**: [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines
2. **RFC Compliance**: Ensure all changes maintain RFC-0111 & RFC-0115 compliance
3. **Documentation**: Update relevant documentation
4. **Testing**: Add comprehensive tests
5. **Security**: Follow security best practices

### Development Setup
```bash
# Install tools
make install-tools

# Run linting
make lint

# Run all tests
make test

# Build documentation
make docs
```

---

## 📄 License

**Apache License 2.0**

Copyright (c) 2025 Gimel Foundation gGmbH i.G.

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

---

## 🔗 Links

- **Gimel Foundation**: https://www.GimelFoundation.com
- **GimelID**: https://www.GimelID.com  
- **Repository**: https://github.com/Gimel-Foundation/gauth
- **Issues**: https://github.com/Gimel-Foundation/gauth/issues
- **RFC Documentation**: [docs/RFC_ARCHITECTURE.md](docs/RFC_ARCHITECTURE.md)

---

## 📞 Support

For questions, issues, or contributions:

- **GitHub Issues**: [Create an issue](https://github.com/Gimel-Foundation/gauth/issues)
- **Documentation**: [docs/](docs/)
- **RFC Questions**: Refer to official GiFo-RFC-0111 & GiFo-RFC-0115 specifications

**Gimel Foundation gGmbH i.G.**  
Hardtweg 31, D-53639 Königswinter, Germany  
Registration: Siegburg HRB 18660