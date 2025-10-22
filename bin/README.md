# Binary Directory

> Last Updated: 2025-10-17
> Status: Active

This directory contains compiled binaries and executables for the GAuth implementation.

## Current Status: 🎓 BETA BINARIES

> **⚠️ Beta Purpose Only**: These binaries are for learning and demonstration. They are NOT production ready. Do NOT use for real security, production, or commercial deployment.

All binaries build successfully for beta and testing purposes.

## Directory Structure

```
bin/
├── gauth-server          # ✅ Main GAuth server binary (working)
├── examples/             # ✅ Example binaries and demos
│   └── rfc-demo          # ✅ RFC compliance demonstration
└── README.md             # This file
```

## Working Binaries

### 🚀 Main Server Binary
- **File**: `gauth-server`
- **Status**: ✅ **BUILDS AND RUNS SUCCESSFULLY**
- **Size**: ~2.4MB (optimized Go binary)
- **Purpose**: Complete GAuth server implementation

```bash
# Build the server (WORKS ✅)
go build -o bin/gauth-server ./cmd/gauth-server

# Run the server (FUNCTIONAL ✅)
./bin/gauth-server
```

### 📋 Example Binaries
- **Directory**: `examples/`
- **Status**: 🎓 **BETA EXAMPLES**
- **Purpose**: Demonstration and testing binaries

## Build Instructions

### Development Build
```bash
# Build main server
make build-server
# OR
go build -o bin/gauth-server ./cmd/gauth-server

# Build all examples
make build-examples
# OR
go build -o bin/examples/rfc-demo ./cmd/examples/rfc-demo
```

### Verification
```bash
# Verify server binary
./bin/gauth-server --version    # Should show version info
./bin/gauth-server --help       # Should show usage help

# Test functionality
go test ./cmd/gauth-server/...   # All tests pass ✅
```

## Binary Features

### GAuth Server Binary
- ✅ **Token Management**: Complete lifecycle with revocation
- ✅ **Event System**: Typed events with structured handlers
- ✅ **Resilience Patterns**: Circuit breaker, rate limiting, retry
- ✅ **Authorization Flow**: Full GAuth protocol implementation
- ✅ **Audit Logging**: Transaction and authorization tracking
- ✅ **Configuration**: File and environment variable support

### Build Quality
- ✅ **Zero compilation errors** across all binaries
- ✅ **Proper Go modules** with dependency management
- ✅ **Optimized builds** with appropriate binary sizes
- ✅ **Cross-platform** compatibility (Linux, macOS, Windows)

## Deployment

### Development Deployment
```bash
# Local development
./bin/gauth-server --config=configs/development.yaml

# Docker deployment
docker build -t gauth:latest .
docker run -p 8080:8080 gauth:latest
```

### Configuration
The server binary supports:
- Configuration files (YAML/JSON)
- Environment variables
- Command-line flags
- Default development settings

## Troubleshooting

### Build Issues
```bash
# Clean and rebuild
make clean
make build

# Check Go version (requires Go 1.24+)
go version

# Verify dependencies
go mod tidy && go mod verify
```

### Runtime Issues
```bash
# Check binary
./bin/gauth-server --version

# Run with verbose output
./bin/gauth-server --verbose --config=configs/debug.yaml
```

## Beta Use

The binaries are designed for:
- 🎓 **Learning environments** - Beta testing and exploration
- 📚 **Demonstration purposes** - Showing GAuth concepts
- 🔬 **Experimentation** - Testing ideas and understanding flows
- 🧪 **Development learning** - Understanding build and deployment concepts

---

*Implementation by [Mauricio Fernandez](https://github.com/mauriciomferz)*

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
