# Manual Testing Guide

This guide provides instructions for manually testing and exploring the functionality of the Gauth library.

## Prerequisites

1. Go 1.19 or later
2. Redis (optional, for distributed features)
3. Docker (optional, for running examples)

## Quick Start

1. Clone the repository:
```bash
git clone https://github.com/Gimel-Foundation/gauth.git
cd gauth
```

2. Run basic examples:
```bash
make run-examples
```

This will run a series of examples demonstrating core functionality.

## Core Features Testing

### 1. Token Management

Test token creation and validation:

```bash
cd examples/basic
go run main.go
```

This will:
- Create a token
- Validate the token
- Attempt to use an expired token
- Demonstrate token revocation

### 2. Authorization

Test authorization policies:

```bash
cd examples/authz
go run main.go
```

This demonstrates:
- Policy creation
- Access evaluation
- Role-based access
- Attribute-based access

### 3. Resilience Patterns

Test resilience features:

```bash
cd examples/resilience/comprehensive
go run main.go
```

Then in another terminal:
```bash
# Test successful request
curl http://localhost:8080/resilient

# Test circuit breaker
for i in {1..10}; do curl http://localhost:8080/resilient; done

# Test bulkhead
ab -n 100 -c 10 http://localhost:8080/resilient
```

### 4. Distributed Features

Prerequisites:
- Running Redis instance
- Multiple terminal windows

```bash
# Terminal 1: Start first node
cd examples/distributed
go run main.go -port 8081

# Terminal 2: Start second node
go run main.go -port 8082

# Terminal 3: Test distributed token validation
curl -X POST http://localhost:8081/token
curl http://localhost:8082/validate/{token}
```

## Common Testing Scenarios

### Authentication Flow

1. Start the auth server:
```bash
cd examples/auth
go run main.go
```

2. Test various authentication methods:
```bash
# Basic auth
curl -u user:pass http://localhost:8080/login

# Bearer token
TOKEN=$(curl -u user:pass http://localhost:8080/login | jq -r .token)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/secure
```

### Policy Testing

1. Create test policies:
```bash
cd examples/authz
go run main.go -setup
```

2. Test policy evaluation:
```bash
# Test allowed access
curl -X POST http://localhost:8080/evaluate \
  -d '{"subject": {"id": "user1", "roles": ["admin"]}, 
       "action": "read",
       "resource": "document1"}'

# Test denied access
curl -X POST http://localhost:8080/evaluate \
  -d '{"subject": {"id": "user2", "roles": ["user"]}, 
       "action": "write",
       "resource": "document1"}'
```

### Resilience Testing

1. Start resilience demo:
```bash
cd examples/resilience/comprehensive
go run main.go
```

2. Test patterns:
```bash
# Normal operation
curl http://localhost:8080/resilient

# Circuit breaker
for i in {1..10}; do curl http://localhost:8080/resilient; done

# Timeout
curl http://localhost:8080/resilient/slow

# Retry
curl http://localhost:8080/resilient/flaky
```

## Monitoring and Debugging

### Metrics

1. Start monitoring example:
```bash
cd examples/monitoring
go run main.go
```

2. View metrics:
```bash
curl http://localhost:8080/metrics
```

### Tracing

1. Start tracing example:
```bash
cd examples/tracing
go run main.go
```

2. Generate traces:
```bash
curl http://localhost:8080/traced-operation
```

3. View traces in Jaeger UI (http://localhost:16686)

## Performance Testing

Run benchmarks:
```bash
make bench
```

This runs benchmarks for:
- Token operations
- Policy evaluation
- Authentication
- Resilience patterns

## Common Issues and Solutions

### Token Validation Fails
- Check token expiration
- Verify signing key configuration
- Check if token is revoked

### Authorization Denied
- Review policy configuration
- Check subject roles and attributes
- Verify resource permissions

### Circuit Breaker Opens
- Check service health
- Review failure thresholds
- Monitor recovery time

### Redis Connection Issues
- Verify Redis is running
- Check connection string
- Review network configuration

## Next Steps

- Review example code for implementation patterns
- Check documentation for detailed API information
- Join community discussions for support
- Consider contributing improvements