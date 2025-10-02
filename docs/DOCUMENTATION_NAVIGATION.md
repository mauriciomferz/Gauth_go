# GAuth Documentation Navigation Guide

**Official Gimel Foundation RFC Implementation - Documentation Index**

This guide helps you navigate the GAuth documentation and choose the right API for your needs.

## 📚 **Documentation Overview**

The GAuth project provides **two main API surfaces**:

1. **🏗️ Go Library API** - Full RFC-compliant implementation
2. **🌐 Web Demo API** - REST endpoints for demonstration

## 🎯 **Choose Your API**

### **For Production Go Applications**

**Use the Go Library API:**

| Document | Purpose | Status |
|----------|---------|--------|
| [API_REFERENCE.md](./API_REFERENCE.md) | Complete Go library API reference | ✅ Current |
| [GETTING_STARTED.md](./GETTING_STARTED.md) | Quick start guide | ✅ Current |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | System architecture overview | ✅ Current |

**Key Package:**
```go
import "github.com/Gimel-Foundation/gauth/pkg/auth"

// Create RFC-compliant service
service, err := auth.NewRFCCompliantService("issuer", "audience")
```

### **For Web/REST Integration**

**Use the Web Demo API:**

| Document | Purpose | Status |
|----------|---------|--------|
| [COMPLETE_API_REFERENCE.md](./COMPLETE_API_REFERENCE.md) | REST API endpoints documentation | ✅ Excellent |

**Base URL:** `http://localhost:8080`

**Available Endpoints:**
- `GET /scenarios` - List demo scenarios
- `POST /authenticate` - Mock authentication  
- `POST /validate` - Token validation
- `POST /rfc0111/config` - RFC-0111 configuration
- `POST /rfc0115/poa` - RFC-0115 power of attorney
- `POST /combined/demo` - Combined RFC demo

## 🚀 **Quick Start Paths**

### **Path 1: Go Library Development**
1. Read [GETTING_STARTED.md](./GETTING_STARTED.md)
2. Review [API_REFERENCE.md](./API_REFERENCE.md)
3. Run example: `go run examples/official_rfc_compliance_test/main.go`

### **Path 2: Web API Integration**
1. Read [COMPLETE_API_REFERENCE.md](./COMPLETE_API_REFERENCE.md)
2. Start demo server: `./gauth-demo-app/web/start.sh`
3. Test endpoints: `curl http://localhost:8080/scenarios`

### **Path 3: Understanding the System**
1. Read [ARCHITECTURE.md](./ARCHITECTURE.md) - System overview
2. Review [RFC_ARCHITECTURE.md](./RFC_ARCHITECTURE.md) - RFC compliance details
3. Check [PERFORMANCE.md](./PERFORMANCE.md) - Performance characteristics

## 📋 **All Documentation Files**

### **Core Documentation**
- [API_REFERENCE.md](./API_REFERENCE.md) - Go library API reference
- [COMPLETE_API_REFERENCE.md](./COMPLETE_API_REFERENCE.md) - Web API reference
- [GETTING_STARTED.md](./GETTING_STARTED.md) - Quick start guide
- [ARCHITECTURE.md](./ARCHITECTURE.md) - System architecture

### **Implementation Guides**
- [AUTHORIZATION_IMPLEMENTATION.md](./AUTHORIZATION_IMPLEMENTATION.md) - Authorization patterns
- [COMPLIANCE_IMPLEMENTATION.md](./COMPLIANCE_IMPLEMENTATION.md) - RFC compliance details
- [CRYPTOGRAPHY_IMPLEMENTATION.md](./CRYPTOGRAPHY_IMPLEMENTATION.md) - Cryptographic implementation
- [INFRASTRUCTURE_IMPLEMENTATION.md](./INFRASTRUCTURE_IMPLEMENTATION.md) - Infrastructure setup

### **Reference Documentation**
- [RFC_0115_IMPLEMENTATION_SUMMARY.md](./RFC_0115_IMPLEMENTATION_SUMMARY.md) - RFC-0115 details
- [RFC_ARCHITECTURE.md](./RFC_ARCHITECTURE.md) - RFC architecture overview
- [EVENT_SYSTEM.md](./EVENT_SYSTEM.md) - Event system documentation
- [PATTERNS_GUIDE.md](./PATTERNS_GUIDE.md) - Implementation patterns

### **Development & Testing**
- [DEVELOPMENT.md](./DEVELOPMENT.md) - Development setup
- [TESTING.md](./TESTING.md) - Testing guide
- [BENCHMARKS.md](./BENCHMARKS.md) - Performance benchmarks
- [PERFORMANCE.md](./PERFORMANCE.md) - Performance analysis
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - Common issues

### **Examples & Guides**
- [EXAMPLES.md](./EXAMPLES.md) - Code examples
- [REAL_SECURITY_PLAN.md](./REAL_SECURITY_PLAN.md) - Security considerations
- [LEGAL_BUSINESS_REVIEW_CHECKLIST.md](./LEGAL_BUSINESS_REVIEW_CHECKLIST.md) - Legal compliance

### **Reports & Summaries**
- [reports/](./reports/) - Implementation reports and summaries
- [COMPREHENSIVE_DOCUMENTATION_REVIEW.md](./COMPREHENSIVE_DOCUMENTATION_REVIEW.md) - Documentation review
- [DOCUMENTATION_UPDATE_SUMMARY.md](./DOCUMENTATION_UPDATE_SUMMARY.md) - Update history

## ⚠️ **Important Notes**

### **Module Path**
All Go imports should use:
```go
import "github.com/Gimel-Foundation/gauth/pkg/auth"
```

### **Development Status**
- **Go Library**: RFC-compliant implementation (development prototype)
- **Web Demo**: Educational demonstration only
- **Security**: Both are development/demo implementations - not production ready

### **Support**

- **Issues**: GitHub repository issue tracker
- **Documentation**: This docs/ folder
- **Examples**: `/examples/` directory with working code

## 🎯 **Recommended Reading Order**

**For New Users:**
1. This index file (you are here)
2. [GETTING_STARTED.md](./GETTING_STARTED.md)
3. [COMPLETE_API_REFERENCE.md](./COMPLETE_API_REFERENCE.md) or [API_REFERENCE.md](./API_REFERENCE.md)

**For Implementers:**
1. [ARCHITECTURE.md](./ARCHITECTURE.md)
2. [API_REFERENCE.md](./API_REFERENCE.md)
3. [COMPLIANCE_IMPLEMENTATION.md](./COMPLIANCE_IMPLEMENTATION.md)
4. Working examples in `/examples/`

**For System Architects:**
1. [ARCHITECTURE.md](./ARCHITECTURE.md)
2. [RFC_ARCHITECTURE.md](./RFC_ARCHITECTURE.md)
3. [PERFORMANCE.md](./PERFORMANCE.md)
4. [REAL_SECURITY_PLAN.md](./REAL_SECURITY_PLAN.md)