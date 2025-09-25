# GAuth Demo Application

A comprehensive demonstration of the GAuth (AI Power-of-Attorney Authorization Framework) protocol with web interface, command-line tools, and Python SDK.

## 🚀 Features

### Web Application
- **Interactive Dashboard**: Visual representation of all GAuth capabilities
- **Legal Framework Demo**: Complete RFC111 authorization flow
- **Power of Attorney Management**: Create, delegate, and manage AI powers
- **Real-time Audit Trail**: Live monitoring of authorization events
- **Compliance Dashboard**: Jurisdiction-specific regulatory compliance
- **Token Management**: JWT/PASETO token lifecycle management
- **Rate Limiting Controls**: Demonstration of various rate limiting strategies

### Command Line Interface
- **Complete Protocol Demo**: Run full GAuth authorization flows
- **Batch Operations**: Bulk authorization and token management
- **Configuration Management**: Setup and manage GAuth instances
- **Testing Tools**: Validate implementations and compliance

### Python SDK
- **Native Python API**: Full Python bindings for GAuth protocol
- **Integration Examples**: FastAPI, Django, Flask integrations
- **Type Safety**: Pydantic models for all GAuth types
- **Async Support**: Full async/await support for modern Python

## 🏗️ Architecture

```
gauth-demo-app/
├── web/                    # React + TypeScript frontend
│   ├── frontend/          # React application
│   └── backend/           # Go HTTP server
├── cli/                   # Command-line tools
│   ├── main.go           # Main CLI application
│   └── commands/         # CLI command implementations
├── python-sdk/           # Python SDK and bindings
│   ├── pygauth/          # Python package
│   ├── examples/         # Python usage examples
│   └── tests/            # Python test suite
└── shared/               # Shared types and utilities
    ├── models/           # Common data models
    └── config/           # Configuration management
```

## 🎯 Demonstrated Capabilities

### 1. Legal Framework Operations
- ✅ **Entity Verification**: Legal capacity validation
- ✅ **Power of Attorney**: Creation and delegation chains
- ✅ **Jurisdiction Compliance**: Multi-jurisdiction authorization
- ✅ **Fiduciary Duties**: Automated compliance checking
- ✅ **Approval Workflows**: Multi-level approval processes

### 2. Authentication & Authorization
- ✅ **Token Issuance**: JWT and PASETO token generation
- ✅ **Token Validation**: Signature verification and expiration
- ✅ **Scope Management**: Fine-grained permission control
- ✅ **Resource Protection**: RBAC/ABAC policy enforcement
- ✅ **Delegation**: Power delegation between entities

### 3. Audit & Compliance
- ✅ **Comprehensive Logging**: All protocol events logged
- ✅ **Compliance Tracking**: Regulatory requirement validation
- ✅ **Event Streaming**: Real-time event notifications
- ✅ **Forensic Analysis**: Detailed audit trail analysis

### 4. Resilience & Performance
- ✅ **Rate Limiting**: Multiple rate limiting strategies
- ✅ **Circuit Breaking**: Fault tolerance mechanisms
- ✅ **Caching**: Redis-based token and data caching
- ✅ **Observability**: Metrics, tracing, and monitoring

### 5. Integration Patterns
- ✅ **REST API**: Complete HTTP API implementation
- ✅ **gRPC Support**: High-performance RPC interface
- ✅ **Event-Driven**: Pub/sub event architecture
- ✅ **Microservices**: Distributed deployment patterns

## 🚀 Quick Start

### Prerequisites
- Go 1.23+
- Node.js 18+
- Python 3.10+
- Docker (optional)

### Web Application
```bash
# Quick start with pre-built executable (from project root)
cd ..
./gauth-http-server
# Access at http://localhost:8080

# OR build from source
cd web
make run
# Access at http://localhost:3000
```

### Command Line Interface
```bash
# Use pre-built executables from project root
cd ..
./gauth-server          # Basic console demo
./gauth-http-server     # Web server demo

# OR build CLI from source
cd cli
go build -o gauth-cli .
./gauth-cli demo --scenario legal-framework
```

### Python SDK
```bash
# Install Python SDK
cd python-sdk
pip install -e .

# Run Python examples
python examples/basic_usage.py
python examples/legal_framework.py
python examples/async_operations.py
```

## 📚 Documentation

- [Web Application Guide](web/README.md)
- [CLI Reference](cli/README.md)
- [Python SDK Documentation](python-sdk/README.md)
- [API Reference](docs/api-reference.md)
- [Integration Examples](docs/integration-examples.md)

## 🔧 Development

### Running Tests
```bash
# Go tests
make test-go

# Python tests
make test-python

# Web frontend tests
make test-web

# Integration tests
make test-integration
```

### Building for Production
```bash
# Build all components
make build-all

# Build Docker images
make docker-build

# Deploy to production
make deploy
```

## 🤝 Contributing

Please read our [Contributing Guide](../CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](../LICENSE) file for details.

## 🔗 Related Links

- [GAuth RFC111 Specification](https://gimelfoundation.com)
- [Main GAuth Repository](../README.md)
- [Examples Collection](../examples/README.md)