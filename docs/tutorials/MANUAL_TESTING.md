# Manual Testing & Familiarization Guide

This comprehensive guide helps you manually test and explore GAuth's functionality using type-safe, modular APIs.

## 🚀 Quick Testing (5 minutes)

### 1. Basic Console Demo
```bash
# Run the pre-built console demo
./gauth-server

# Expected output:
# ✓ Authorization granted  
# ✓ Token issued
# ✓ Transaction succeeded
# ✓ Demo completed successfully!
```

### 2. Interactive Web Interface
```bash
# Start the web server
./gauth-http-server

# Open browser: http://localhost:8080
# Test features:
# - Create JWT tokens
# - Validate tokens  
# - View live metrics
# - Explore API endpoints
```

## 🧪 Comprehensive Testing Scenarios

### 1. Token Lifecycle Testing
```bash
cd examples/basic
go run main.go
```
**Test Cases:**
- ✅ Token creation with different scopes
- ✅ Token validation (valid/invalid/expired)  
- ✅ Token refresh mechanisms
- ✅ Scope verification and enforcement

### 2. Authorization Flow Testing  
```bash
cd examples/advanced
go run main.go
```
**Test Cases:**
- ✅ Multi-scope authorization requests
- ✅ Grant approval workflows
- ✅ Delegation chains
- ✅ Resource access control

### 3. Error Handling Testing
```bash
cd examples/errors
go run main.go
```
**Test Cases:**
- ❌ Invalid token formats
- ❌ Expired tokens  
- ❌ Insufficient scopes
- ❌ Network failures
- ❌ Rate limit violations

### 4. Rate Limiting Testing
```bash  
cd examples/rate
go run main.go
```
**Test Cases:**
- 🚦 Burst request handling
- 🚦 Sustained traffic patterns  
- 🚦 Multi-client scenarios
- 🚦 Rate limit recovery

### 5. Audit & Compliance Testing
```bash
cd examples/audit  
go run main.go
```
**Test Cases:**
- 📝 Complete event logging
- 📝 Audit trail generation
- 📝 Compliance reporting
- 📝 Forensic analysis

## 🔧 Advanced Testing Scenarios

### 1. Custom Configuration Testing
```bash
# Test with custom configs
export GAUTH_CLIENT_SECRET="test-secret-123"
export GAUTH_SERVER_READ_TIMEOUT="30s"
./gauth-server
```

### 2. Resilience Testing
```bash
cd examples/resilient
go run main.go
```
**Test Cases:**
- 🛡️ Circuit breaker activation
- 🛡️ Retry mechanisms  
- 🛡️ Graceful degradation
- 🛡️ Recovery behaviors

### 3. Performance Testing
```bash
cd examples/monitoring
go run main.go
```
**Test Cases:**
- ⚡ High-throughput token validation
- ⚡ Concurrent request handling
- ⚡ Memory usage patterns
- ⚡ Response time analysis

### 4. Integration Testing
```bash
cd examples/gateway
go run main.go
```
**Test Cases:**
- 🔌 API gateway integration
- 🔌 Microservices communication
- 🔌 Event-driven workflows
- 🔌 Cross-service authentication

## 🌐 Web Interface Testing

### Token Management Testing
1. **Create Token**:
   - Subject: `test-user`
   - Role: `admin` 
   - Verify JWT structure and claims

2. **Validate Token**:
   - Copy generated token to validation field
   - Verify successful validation
   - Test with invalid token

3. **Live Metrics**:
   - Watch real-time counter updates
   - Verify WebSocket connectivity
   - Test auto-reconnection

### API Endpoint Testing
```bash
# Health check
curl http://localhost:8080/health

# Create token
curl -X POST http://localhost:8080/api/v1/tokens \
  -H "Content-Type: application/json" \
  -d '{"subject":"test","role":"user"}'

# Validate token  
curl -X POST http://localhost:8080/api/v1/tokens/validate \
  -H "Content-Type: application/json" \
  -d '{"token":"YOUR_TOKEN_HERE"}'

# System metrics
curl http://localhost:8080/api/v1/metrics/system
```

## 🔍 Testing Custom Extensions

### 1. Custom Token Store
```go
// Implement your own token store
type MyTokenStore struct {
    // Your implementation
}

func (s *MyTokenStore) Store(token string) error {
    // Your logic
    return nil
}
```

### 2. Custom Event Types
```go
// Define custom event metadata
type CustomEventMetadata struct {
    BusinessUnit string    `json:"business_unit"`
    CostCenter   string    `json:"cost_center"`
    Project      string    `json:"project"`
}
```

### 3. Custom Audit Logger
```go
// Implement custom audit functionality
type MyAuditLogger struct {
    // Your implementation
}

func (a *MyAuditLogger) LogEvent(event interface{}) error {
    // Your logging logic
    return nil
}
```

## 📊 Testing Checklist

### Core Functionality
- [ ] Authorization flow completes successfully
- [ ] Tokens are generated with correct format
- [ ] Token validation works for valid/invalid tokens
- [ ] Scopes are properly enforced
- [ ] Audit events are logged correctly

### Error Handling  
- [ ] Invalid tokens are rejected properly
- [ ] Expired tokens return appropriate errors
- [ ] Network failures are handled gracefully
- [ ] Rate limiting activates correctly
- [ ] Error messages are clear and actionable

### Performance
- [ ] Response times are acceptable (<100ms for token ops)
- [ ] Memory usage is stable under load
- [ ] No memory leaks in long-running operations
- [ ] Concurrent operations handle properly

### Security
- [ ] Tokens cannot be forged or modified
- [ ] Sensitive data is not logged
- [ ] Rate limiting prevents abuse
- [ ] Authentication is properly enforced

### Web Interface
- [ ] All features work in modern browsers
- [ ] Mobile responsiveness functions correctly  
- [ ] WebSocket connections are stable
- [ ] Real-time updates work as expected
- [ ] API endpoints return correct responses

## 🆘 Troubleshooting Test Issues

### Build Problems
```bash
# Clean and rebuild
make clean && make build

# Check Go module status
go mod tidy
go mod verify
```

### Runtime Problems
```bash
# Check executable permissions
chmod +x gauth-*

# Run with debug output
./gauth-server -v

# Check for port conflicts
lsof -i :8080
```

### Web Interface Problems
```bash
# Clear browser cache
# Try different browser
# Check browser developer console for errors
# Verify server is running on correct port
```

## 📈 Performance Benchmarking

```bash
# Run performance tests
go test -bench=. ./pkg/gauth

# Memory profiling
go test -memprofile=mem.prof ./pkg/gauth
go tool pprof mem.prof

# CPU profiling  
go test -cpuprofile=cpu.prof ./pkg/gauth
go tool pprof cpu.prof
```

## 📝 Test Results Documentation

After testing, document:
- ✅ Successful test scenarios
- ❌ Failed test cases and root causes
- 🐛 Bugs discovered and reported
- 💡 Improvement suggestions
- ⚡ Performance observations
- 🛡️ Security considerations noted

---

For more detailed information, see:
- [README.md](./README.md) - Main project documentation
- [docs/TESTING.md](./docs/TESTING.md) - Automated testing guide  
- [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) - System architecture
- [CONTRIBUTING.md](./CONTRIBUTING.md) - Contributing guidelines
