# GAuth: AI Power-of-Attorney Authorization Framework

**🚀 Enhanced with Interactive Demo Webapp & Modern API**

GAuth enables AI systems to act on behalf of humans or organizations, with explicit, verifiable, and auditable power-of-attorney flows. Built on OAuth, OpenID Connect, and MCP, GAuth is designed for open source, extensibility, and compliance with RFC111.

## 🌟 **NEW: Enhanced Interactive Webapp with Real-time Features**

Experience GAuth in action with our cutting-edge web application featuring modern UI and live updates:

### 🎫 **Interactive Token Management**
- **Real-time Token Creation**: Generate JWT tokens with custom claims and instant feedback
- **Live Validation**: Instant token validation with comprehensive claim inspection
- **Auto-workflow**: Created tokens automatically populate validation fields
- **Visual Feedback**: Beautiful loading states, success animations, and error handling

### 📊 **Live System Dashboard**
- **Real-time Metrics**: Auto-updating system statistics (users, transactions, success rate)
- **WebSocket Integration**: Live event streaming with automatic reconnection
- **Health Monitoring**: Visual API and WebSocket connection status indicators
- **Performance Tracking**: Response times and system performance metrics

### 🎨 **Modern User Experience**
- **Glassmorphism Design**: Beautiful modern UI with backdrop blur effects
- **Mobile Responsive**: Perfect experience on mobile, tablet, and desktop
- **Keyboard Shortcuts**: Ctrl/Cmd+Enter for quick actions and workflow efficiency
- **Toast Notifications**: Real-time user feedback with professional animations
- **Dark Mode**: Elegant gradient backgrounds with high contrast readability

### 🌐 **Enhanced Technical Features**
- **WebSocket Real-time Events**: Live updates for token operations and system events
- **Advanced Error Handling**: Comprehensive error states with recovery suggestions
- **Auto-reconnection**: Smart WebSocket reconnection with exponential backoff
- **Progressive Enhancement**: Core functionality works even with JavaScript disabled

### 🚀 Quick Start with Enhanced Webapp
```bash
# Start the enhanced backend server with WebSocket support
cd gauth-demo-app/web/backend
go build -o gauth-enhanced-server main.go
./gauth-enhanced-server

# Access the interactive demo with real-time features
open http://localhost:8080
```

**✨ Try these features immediately:**
- Create tokens with custom subject and role
- See real-time validation with detailed claim inspection  
- Watch live system metrics auto-update every 5 seconds
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

---

## Getting Started

### **🌐 Option 1: Interactive Webapp (Recommended)**
```bash
# Clone the repository
git clone https://github.com/mauriciomferz/Gauth_go.git
cd Gauth_go

# Start the enhanced backend
cd gauth-demo-app/web/backend
go run main.go

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

Explore the `/demo` and `/cmd/demo` directories for runnable demonstration applications and usage examples:

- `/demo/main.go`: End-to-end protocol demo (authorization, token, transaction, audit)
- `/cmd/demo/main.go` and `/cmd/demo/improved_main.go`: CLI demo apps showing advanced flows

To run a demo:

```sh
# Run the main demo
cd demo
go run main.go

# Or run the CLI demo
cd cmd/demo
go run main.go
```

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