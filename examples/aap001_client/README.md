---
title: "RFC-0111 Client Examples"
category: example
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# RFC-0111 Client Examples

This directory contains examples demonstrating how to use the RFC-0111 REST API.

## Quick Demo (Bash)

Fast demonstration of Steps I-III using curl:

```bash
./scripts/demo_rfc0111_quick.sh
```

**Output:**
- Creates a new subscription (Step I)
- Verifies authorizer authorization (Step II)
- Verifies client owner identity (Step III)
- Shows final subscription status

## Go Client Example

Complete programmatic example in Go:

```bash
go run examples/rfc0111_client/main.go
```

**Features:**
- Type-safe API client
- Error handling
- JSON request/response handling
- Demonstrates Steps I-III

## Integration Test

Comprehensive test of all 8 subscription steps:

```bash
./scripts/test_rfc0111_subscription_flow.sh
```

**Features:**
- Tests all Steps I-VIII
- Validates prerequisites
- Error handling tests
- Colored output with pass/fail indicators
- Summary report

## Prerequisites

1. **Start the server with RFC-0111 enabled:**
   ```bash
   GAUTH_AAP-001_ENABLED=1 go run ./cmd/web-server
   ```

2. **Verify server is running:**
   ```bash
   curl http://localhost:8080/api/v1/rfc0111/subscriptions?client_id=test
   ```

## API Documentation

For complete API reference, see:
- **[AAP-001_API_GUIDE.md](../AAP-001_API_GUIDE.md)** - Complete API documentation with examples
- **[AAP-001_WEB_INTEGRATION_GUIDE.md](../AAP-001_WEB_INTEGRATION_GUIDE.md)** - Integration guide
- **[AAP-001_SESSION_SUMMARY.md](../AAP-001_SESSION_SUMMARY.md)** - Implementation summary

## Example Output

### Quick Demo
```
========================================
RFC-0111 API Quick Demo
========================================

Step I: Owner's Authorizer Identity Proof
✓ Created subscription: sub_1762827332502984000

Step II: Owner's Authorizer Authorization Proof
✓ Step II completed

Step III: Client Owner Identity Proof
✓ Step III completed

Final Subscription Status: awaiting_client
========================================
Demo completed successfully!
========================================
```

### Go Client
```
=== RFC-0111 API Go Client Example ===

Step I: Creating subscription...
✓ Subscription created: sub_1762827409479317000

Step II: Verifying authorizer authorization...
✓ Step II completed

Step III: Verifying client owner identity...
✓ Step III completed

Checking subscription status...
✓ Subscription status: awaiting_client

=== Demo completed successfully! ===
```

## Next Steps

After running these examples:

1. **Complete remaining steps (IV-VIII)** - See API guide for curl examples
2. **Implement authorization flow** - Use completed subscription for token requests
3. **Build your own client** - Use the Go example as a template

## Available Endpoints

| Endpoint | Method | Description | Status |
|----------|--------|-------------|--------|
| `/subscriptions` | POST | Step I: Create subscription | ✅ |
| `/subscriptions/:id/step-ii` | POST | Step II: Authorization proof | ✅ |
| `/subscriptions/:id/step-iii` | POST | Step III: Client owner identity | ✅ |
| `/subscriptions/:id/step-iv` | POST | Step IV: Client owner auth | ✅ |
| `/subscriptions/:id/step-v` | POST | Step V: Client authorization | ✅ |
| `/subscriptions/:id/step-vi` | POST | Step VI: Resource owner identity | ✅ |
| `/subscriptions/:id/step-vii` | POST | Step VII: Resource owner auth | ✅ |
| `/subscriptions/:id/step-viii` | POST | Step VIII: Resource server | ✅ |
| `/subscriptions/:id` | GET | Get subscription | ✅ |
| `/subscriptions` | GET | List subscriptions | ✅ |

## Troubleshooting

### Server not responding
```bash
# Check if server is running
lsof -ti:8080

# If not running, start it
GAUTH_AAP-001_ENABLED=1 go run ./cmd/web-server
```

### 404 errors
```bash
# Ensure RFC-0111 is enabled
export GAUTH_AAP-001_ENABLED=1
go run ./cmd/web-server
```

### Step execution errors
- Ensure steps are executed in order (I → II → III → ... → VIII)
- Check that subscription ID is correct
- Verify request format matches examples in API guide

## Support

For issues or questions:
- See **[AAP-001_API_GUIDE.md](../AAP-001_API_GUIDE.md)** troubleshooting section
- Check server logs for detailed error messages
- Verify environment variables are set correctly
