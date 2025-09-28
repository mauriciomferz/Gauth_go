# Getting Started with GAuth

Welcome! This guide will help you get up and running with GAuth quickly, whether you want to try the demo or integrate it into your application.

## 🚀 Quick Start (30 seconds)

### Option 1: Try the Demo Immediately
```bash
# Clone the repository
git clone https://github.com/mauriciomferz/Gauth_go.git
cd Gauth_go

# Run the basic console demo
./gauth-server

# OR run the web interface
./gauth-http-server
# Then open http://localhost:8080
```

### Option 2: Build from Source
```bash
# Ensure dependencies are up to date
go mod tidy

# Build all executables
make build

# Run the demo
./gauth-server
```

## 🎯 What to Try First

### 1. **Console Demo** - Understanding the Protocol
```bash
./gauth-server
```
This shows the complete GAuth authorization flow:
- Authorization request and grant
- Token issuance and validation  
- Transaction processing with token
- Audit logging

### 2. **Web Interface** - Interactive Experience  
```bash
./gauth-http-server
# Open http://localhost:8080
```
Features to explore:
- Create JWT tokens with custom claims
- Real-time token validation
- Live system metrics
- API endpoint testing

### 3. **Code Examples** - Integration Patterns
```bash
# Basic authentication flow
cd examples/basic && go run main.go

# Advanced patterns  
cd examples/advanced && go run main.go

# Error handling
cd examples/errors && go run main.go
```

## 📚 Next Steps

### For Developers
- **Library Integration**: See [LIBRARY.md](./LIBRARY.md) for API usage
- **Package Documentation**: Each package has a `doc.go` file with detailed docs
- **Type Safety**: Review [docs/TYPE_SAFETY.md](./docs/TYPE_SAFETY.md) for best practices

### For Architects  
- **Architecture Overview**: [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)
- **Production Deployment**: [PRODUCTION_DEPLOYMENT.md](./PRODUCTION_DEPLOYMENT.md)  
- **Security Considerations**: [SECURITY.md](./SECURITY.md)

### For Contributors
- **Contributing Guidelines**: [CONTRIBUTING.md](./CONTRIBUTING.md)
- **Development Setup**: [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md)
- **Testing Guide**: [docs/TESTING.md](./docs/TESTING.md)

## 🧪 Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
make coverage

# Run specific package tests
go test ./pkg/gauth -v
go test ./pkg/auth -v
```

## 🔧 Build Options

```bash
# Build all binaries
make build

# Build specific targets
make build-server    # Just the console demo
make build-web      # Just the web server

# Clean and rebuild
make clean && make build
```

## 🌐 Web Application Features

The web interface (`./gauth-http-server`) includes:

- **Token Management**: Create and validate JWT/PASETO tokens
- **Real-time Dashboard**: Live metrics and system status  
- **API Explorer**: Test all available endpoints
- **Legal Framework**: RFC111/RFC115 compliance demonstrations
- **Audit Trail**: Complete event logging and analysis

## 🆘 Troubleshooting

### Build Issues
```bash
# Clean module cache and rebuild
go clean -modcache
go mod download
make clean && make build
```

### Runtime Issues  
```bash
# Check if executable exists
ls -la gauth-*

# Run with verbose output
./gauth-server -v

# Check server logs
tail -f server.log
```

## 📖 More Resources

- **Main Documentation**: [README.md](./README.md)
- **API Reference**: [docs/](./docs/)
- **Examples**: [examples/](./examples/)  
- **Demo Applications**: [gauth-demo-app/](./gauth-demo-app/)
- **Release Notes**: [RELEASE_NOTES_v1.0.5.md](./RELEASE_NOTES_v1.0.5.md)
