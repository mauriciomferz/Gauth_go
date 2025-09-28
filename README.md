# GAuth: Go Authorization Framework

**🚀 Production-Ready Authorization Framework** | ✅ **All Tests Passing** | 📊 **Prometheus Monitoring** | 🛡️ **Zero Vulnerabilities**

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

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test suites
go test ./pkg/gauth/...
go test ./examples/resilient/...
```

## 📊 Monitoring

GAuth includes comprehensive Prometheus metrics:

- **Business Metrics**: Authorization attempts, token operations, user activities
- **HTTP Metrics**: Request duration, response sizes, status codes
- **System Metrics**: Active connections, error rates, performance indicators

Access metrics at `/metrics` endpoint when running the HTTP server.

## 📚 Documentation

- **[Getting Started](docs/development/GETTING_STARTED.md)**: Quick introduction and setup
- **[Architecture](docs/ARCHITECTURE.md)**: System design and components
- **[API Reference](docs/api/)**: Complete API documentation
- **[Examples](examples/)**: Code samples and tutorials
- **[Reports](docs/reports/)**: Technical reports and analysis

## 🏛️ RFC Compliance

GAuth implements RFC-111 (GAuth) standard for AI power-of-attorney:
- Explicit authorization flows
- Auditable delegation chains
- Compliance verification
- Multi-jurisdictional support

## 🔧 Development

### Prerequisites
- Go 1.23+
- Docker (optional)
- Make

### Building
```bash
# Build all components
make build

# Run tests
make test

# Run linters
make lint

# Build Docker images
make docker
```

### Contributing
See [CONTRIBUTING.md](docs/development/CONTRIBUTING.md) for guidelines.

## 🚀 Deployment

### Docker
```bash
docker build -t gauth .
docker run -p 8080:8080 gauth
```

### Kubernetes
```bash
kubectl apply -f k8s/
```

### Binary Release
Download pre-built binaries from the releases page.

## 📄 License

Apache 2.0 - see [LICENSE](LICENSE) file for details.

## 🤝 Support

- **Issues**: [GitHub Issues](https://github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/issues)
- **Documentation**: [Project Wiki](https://github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/wiki)
- **Community**: [Discussions](https://github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/discussions)

---

**GAuth** - Secure, scalable, and production-ready authorization for modern applications.