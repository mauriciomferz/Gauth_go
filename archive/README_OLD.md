# GAuth: Go Authorization Framework

**🚀 Production-Ready Authorization Framework** | ✅ **All Tests Passing** | � **Prometheus Monitoring** | 🛡️ **Zero Vulnerabilities**

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/doc/devel/release.html)
[![Security Status](https://img.shields.io/badge/Security-🔒%20Zero%20Vulnerabilities-brightgreen.svg)](./docs/reports/)
[![Build Status](https://img.shields.io/badge/Build-✅%20All%20Tests%20Passing-green.svg)](#testing)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](./LICENSE)

GAuth is a comprehensive Go authorization framework that enables AI systems and applications to act on behalf of humans or organizations with explicit, verifiable, and auditable power-of-attorney flows. Built with modern Go practices, comprehensive monitoring, and production-ready architecture.

## ✨ Key Features

- **🔐 Secure Authorization**: RFC-compliant OAuth and OpenID Connect implementation
- **📊 Prometheus Monitoring**: Complete observability with business and HTTP metrics
- **🏗️ Clean Architecture**: Well-organized pkg/, internal/, examples/ structure
- **🧪 Comprehensive Testing**: 100% test coverage with integration tests
- **🚀 Production Ready**: Zero vulnerabilities, full CI/CD pipeline
- **📖 Rich Documentation**: Complete API docs, guides, and examples

## 🏗️ Project Structure

```
├── pkg/           # Public API packages
│   ├── gauth/     # Core authorization logic
│   ├── auth/      # Authentication providers
│   ├── token/     # Token management
│   ├── metrics/   # Prometheus monitoring
│   └── ...
├── internal/      # Private implementation packages
├── examples/      # Usage examples and demos
├── cmd/           # Command-line applications
├── docs/          # Documentation
│   ├── development/  # Development guides
│   └── reports/     # Technical reports
├── gauth-demo-app/  # Web application demos
└── archive/        # Historical development records
```

## 🚀 Quick Start

### Installation
```bash
go get github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0
```

### Basic Usage
```go
package main

import (
    "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
)

func main() {
    // Initialize GAuth service
    config := gauth.Config{
        AuthServerURL: "https://auth.example.com",
        ClientID:      "your-client-id",
        TokenExpiry:   3600,
    }
    
    service, err := gauth.New(config)
    if err != nil {
        panic(err)
    }
    
    // Use the service for authorization
    token, err := service.Authorize("user123", []string{"read", "write"})
    if err != nil {
        panic(err)
    }
    
    println("Token created:", token.AccessToken)
}
```

### Demo Applications
```bash
# Run the web demo
cd gauth-demo-app/web
go run main.go
# Access at http://localhost:8080

# Run examples
go run examples/basic/main.go
go run examples/resilient/main.go
```
- Experience WebSocket real-time event notifications
- Test mobile responsiveness on different devices

---

## Who is this for?
- Developers integrating AI with sensitive actions or decisions
- Security architects and compliance teams  
- Anyone needing transparent, auditable AI authorization

## Where to Start
- **🌐 Interactive Demo**: `gauth-demo-app/web/` - Full-featured webapp with modern UI
- **📚 Library Code**: `pkg/gauth/` - Core authentication and authorization APIs
- **💡 Examples**: `examples/` - Runnable code samples and tutorials
- **📖 Documentation**: `docs/` - Architecture, patterns, and guides

## RFC111 Compliance
- Implements the GiFo-RfC 0111 (GAuth) standard for AI power-of-attorney, delegation, and auditability.
- All protocol roles, flows, and exclusions are respected. See https://gimelfoundation.com for the full RFC.

## 🚀 Enhanced Features

### **Core Framework**
- **Type Safety**: All public APIs use explicit, strongly-typed structures
- **Authentication**: Token-based authentication with JWT and PASETO support
- **Authorization**: Fine-grained, policy-based access control (RBAC, ABAC, PBAC)
- **Audit Logging**: Comprehensive, structured audit trails for all actions
- **Extensible Storage**: Memory, Redis, and database backends
- **Resilience**: Circuit breaking and retry mechanisms
- **Observability**: Metrics, distributed tracing, and event-driven architecture

### **🆕 Modern Web Application**
- **Interactive Token Management**: Real-time JWT creation and validation
- **Live System Dashboard**: Auto-updating metrics and statistics
- **Legal Framework Integration**: Power-of-attorney and compliance demos
- **Professional API**: RESTful endpoints with comprehensive error handling
- **WebSocket Support**: Real-time event streaming and notifications
- **Mobile-Responsive**: Works beautifully on all device sizes

### **🔧 Developer Experience**
- **Modern Tech Stack**: Go + Gin + React + TypeScript + Material-UI
- **Docker Ready**: Complete containerization support
- **API Documentation**: Swagger/OpenAPI specifications
- **Comprehensive Testing**: Integration tests with >90% coverage
- **Production Ready**: Structured logging, error handling, monitoring

### **🌟 GAuth+ Commercial Register** ✨ **NEW**
- **Blockchain Registry**: First commercial register for AI systems with cryptographic verification
- **Comprehensive Authorization**: Answers WHO, WHAT, TRANSACTIONS, and ACTIONS for AI power-of-attorney
- **Dual Control Principle**: Second-level approval for sensitive operations with human accountability
- **Global Verification**: Any relying party can verify AI authority against blockchain registry
- **Legal Framework**: Full power-of-attorney integration with multi-jurisdictional compliance
- **Authorization Cascade**: Ensures human authority at top of every delegation chain

---

## Getting Started

### **🌐 Option 1: Interactive Webapp (Recommended)**
```bash
# Clone the repository
git clone https://github.com/mauriciomferz/Gauth_go.git
cd Gauth_go

# Use pre-built executable (fastest)
./gauth-http-server

# OR build from source
make build
./gauth-web

# Open the interactive demo
open http://localhost:8080
```

### **📚 Option 2: Library Integration**
1. Review the runnable examples in `examples/` or `examples/<category>/cmd/` for minimal and advanced integrations.
2. See `pkg/gauth/` for core types and APIs.
3. Extend or customize by implementing your own token store, audit logger, or event types.

---

## 🏗️ **Webapp Architecture**

### **🔧 Enhanced Backend (Go + Gin)**
- **🔐 Advanced Authentication**: OAuth2, JWT with key rotation, comprehensive token lifecycle
- **⚖️ Legal Framework**: Full RFC111/RFC115 compliance with multi-jurisdiction support  
- **📊 Real-time Monitoring**: Live system metrics with WebSocket streaming
- **🎭 Interactive Scenarios**: Dynamic demo flows with real-time feedback
- **🔄 WebSocket Server**: Real-time event broadcasting with auto-reconnection
- **🛡️ Security Features**: Structured logging, CORS, comprehensive error handling

### **🎨 Modern Frontend (HTML5 + ES6+ JavaScript)**
- **💎 Glassmorphism UI**: Beautiful modern design with backdrop blur effects
- **📱 Fully Responsive**: Optimized experience across all device sizes  
- **⚡ Real-time WebSocket**: Live updates with automatic reconnection and status indicators
- **🌐 Advanced API Integration**: Intelligent error handling and retry logic
- **⌨️ Enhanced UX**: Keyboard shortcuts, toast notifications, loading animations
- **🎯 Progressive Enhancement**: Works beautifully with and without JavaScript

### **Key API Endpoints**
```
GET  /health                           # Health check
POST /api/v1/tokens                    # Create JWT token
POST /api/v1/tokens/validate           # Validate token
GET  /api/v1/metrics/system            # System metrics
GET  /api/v1/demo/scenarios            # Demo scenarios
GET  /api/v1/legal/jurisdictions       # Legal framework
POST /api/v1/auth/authorize            # OAuth2 authorization
POST /api/v1/audit/events              # Audit events
```

---

## Manual Testing
See `MANUAL_TESTING.md` for suggestions.

---

## Breaking Changes (2025 Migration)

- All example logic with a `main` function is now in its own `main.go` under `examples/` or `examples/<category>/cmd/`.
- All obsolete and duplicate example files have been removed.
- All examples are refactored for the new API and type-safe signatures.
- See `docs/IMPROVEMENTS.md` for a summary of codebase modernization.

## License
- GAuth: Apache 2.0 (see LICENSE)
- OAuth, OpenID Connect: Apache 2.0
- MCP: MIT

---
For more, see the RFC111 specification and the package-level documentation in `doc.go`.

## Documentation

- [Getting Started](docs/GETTING_STARTED.md): Quick introduction to GAuth
- [Code Organization](docs/CODE_ORGANIZATION.md): Understanding the codebase structure
- [Type Safety](docs/TYPE_SAFETY.md): How GAuth ensures type safety
- [Architecture](docs/ARCHITECTURE.md): Design and architecture of GAuth
- [Development](docs/DEVELOPMENT.md): Contributing to GAuth
- [Patterns Guide](docs/PATTERNS_GUIDE.md): Common patterns and best practices
- [Event System](docs/EVENT_SYSTEM.md): Understanding the event system
- [Testing](docs/TESTING.md): Testing GAuth and your application

## Package Structure

- `pkg/`: Public API packages
  - `auth/`: Authentication functionality
  - `authz/`: Authorization and policy enforcement
  - `token/`: Token creation and validation
  - `store/`: Token and session storage backends
  - `events/`: Event system with typed metadata
  - `errors/`: Error handling with typed details
  - `audit/`: Audit logging with compliance features
  - `rate/`: Rate limiting with multiple algorithms
  - `resilience/`: Circuit breaking and retry patterns
  - `resources/`: Resource management system
  - `monitoring/`: Metrics and distributed tracing
  - `metrics/`: Prometheus and custom metrics
  - `mesh/`: Service mesh integration
  - `util/`: Common utilities and helpers
  - `gauth/`: Main integration package

- `internal/`: Implementation details
  - `resource/`: Resource management with typed configurations
  - `ratelimit/`: Rate limiting implementation
  - `circuit/`: Circuit breaking patterns
  - `errors/`: Internal error types

- `examples/`: Usage examples
  - `basic/`: Simple authentication flow
  - `rate/`: Rate limiting examples
  - `resilient/`: Resilience patterns
  - `advanced/`: Advanced usage patterns
  - `typed_events/`: Using strongly typed event structures
  - `errors/`: Error handling and propagation
  - `audit/`: Audit logging implementation
  - `cache/`: Token caching strategies
  - `gateway/`: API gateway integration
  - `events/`: Event system with filtering and subscription

## Demos and Examples

### Ready-to-Use Executables (No Build Required)
```bash
# Basic console demo - Shows GAuth protocol flow
./gauth-server

# Full HTTP server with web interface
./gauth-http-server
# Access at http://localhost:8080
```

### Build from Source
```bash
# Build all binaries
make build

# Run the CLI demo
./gauth-server

# Run the web server
./gauth-web
```

### Code Examples
Explore runnable code examples in:
- `examples/basic/`: Simple authentication flows  
- `examples/advanced/`: Complex integration patterns
- `cmd/demo/`: Command-line demonstration apps
- `gauth-demo-app/`: Full-featured web application

These demos use only the public, type-safe APIs and are a great starting point for learning and extension.

## Design Philosophy

1. **Clear Entry Points**: Each package has well-defined interfaces and documentation.
2. **Extensibility**: Core functionality is interface-based for easy customization.
3. **Type Safety**: Strong typing with typed structures instead of `interface{}`.
4. **Resilience**: Built-in patterns for reliable operation.
5. **Developer Experience**: Comprehensive examples and documentation.
6. **Modularity**: Common utilities are centralized and reusable across packages.

## Typed Structures

GAuth uses strongly typed structures instead of `map[string]interface{}` for better type safety, IDE support, and code clarity:

```go
// Instead of this:
metadata := map[string]interface{}{
    "user_id": "user123",
    "method": "password",
    "timestamp": time.Now(),
}

// Use this:
authEvent := &AuthenticationEvent{
    User: UserMetadata{
        UserID:   "user123",
        Username: "johndoe",
        Email:    "john@example.com",
    },
    Auth: AuthenticationMetadata{
        Method:    "password",
        Timestamp: time.Now(),
        SourceIP:  "192.168.1.1",
    },
}

// Type-safe access without type assertions
fmt.Println(authEvent.User.Username)  // No type assertions needed
```

See the [typed_events example](examples/typed_events) for a complete demonstration.
7. **Code Organization**: Library code and examples are clearly separated.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

1. Clone the repository
2. Install dependencies: `go mod download`
3. Run tests: `go test ./...`
4. Try examples: `cd examples/basic && go run main.go`

### Code Organization

Each package follows these principles:
- `doc.go`: Package documentation
- `interfaces.go`: Public interfaces
- Implementation files: Focused, single-responsibility
- `_test.go`: Comprehensive tests

## Type Safety

GAuth emphasizes type safety throughout the codebase. Instead of using `map[string]interface{}`, we provide strongly-typed structures with helper methods:

```go
// Instead of using map[string]interface{}:
metadata := make(map[string]interface{})
metadata["user_id"] = "user123"
metadata["logged_in"] = true
metadata["login_count"] = 42

// We use typed structures with helper methods:
metadata := events.NewMetadata()
metadata.SetString("user_id", "user123")
metadata.SetBool("logged_in", true)
metadata.SetInt("login_count", 42)

// Type-safe access:
if userID, ok := metadata.GetString("user_id"); ok {
    // Use userID safely
}
```

Read more about our approach to type safety in [Type Safety](docs/TYPE_SAFETY.md).

## Examples

See the `examples/` directory for:
- Basic authentication flows
- Rate limiting implementations
- Resilience patterns
- Custom extensions
- Typed structures demo (showing type-safe alternatives to map[string]interface{})

## Testing

Run the full test suite:
```bash
go test ./...
```

Run specific tests:
```bash
go test ./pkg/gauth -run TestAuth
go test ./internal/rate -run TestRateLimit
```

## License

MIT License - see [LICENSE](LICENSE)

---
*Workflow updated: September 25, 2025 - CodeQL Action v3 implemented*
