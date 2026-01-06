# AgentAuth Quick Start Guide

**Get started with AgentAuth in under 15 minutes!**

This guide will walk you through setting up AgentAuth, making your first API call, and implementing common workflows.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Your First API Call](#your-first-api-call)
4. [Authentication Setup](#authentication-setup)
5. [Common Workflows](#common-workflows)
6. [SDK Setup](#sdk-setup)
7. [Troubleshooting](#troubleshooting)
8. [Next Steps](#next-steps)

---

## Prerequisites

- **Runtime**: Go 1.21+ (for running the server)
- **Tools**: curl, git, or HTTP client of choice
- **Optional**: Node.js 18+, Python 3.9+ (for SDKs)

---

## Installation

### Option 1: Run with Docker (Recommended)

```bash
# Pull and run AgentAuth
docker run -p 8080:8080 ghcr.io/mauriciomferz/agentauth:latest

# Verify it's running
curl http://localhost:8080/api/v1/beta/health
```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/mauriciomferz/AgentAuth.git
cd AgentAuth

# Install dependencies
go mod download

# Run the server
go run ./cmd/web-server

# Server starts on http://localhost:8080
```

### Option 3: Use Pre-built Binary

```bash
# Download latest release
wget https://github.com/mauriciomferz/AgentAuth/releases/latest/download/agentauth-linux-amd64

# Make executable
chmod +x agentauth-linux-amd64

# Run
./agentauth-linux-amd64
```

---

## Your First API Call

### Check Server Health

```bash
curl http://localhost:8080/api/v1/beta/health
```

**Expected Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-15T10:30:00Z",
  "version": "1.0.0-beta"
}
```

✅ **Success!** Your AgentAuth server is running.

---

## Authentication Setup

### Step 1: Create a Subscription (Get Access Token)

AgentAuth uses the AAP-001 subscription flow to issue access tokens. Let's complete the flow automatically:

```bash
# Create subscription (Step I)
SUB_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/aap001/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "quickstart-app",
    "scope": "read write"
  }')

# Extract subscription ID
SUB_ID=$(echo $SUB_RESPONSE | jq -r '.id')
echo "Subscription ID: $SUB_ID"

# Execute steps II-VII automatically
for STEP in ii iii iv v vi vii; do
  curl -s -X POST "http://localhost:8080/api/v1/aap001/subscriptions/$SUB_ID/step-$STEP" \
    -H "Content-Type: application/json" \
    -d '{"proof_type": "document", "proof_data": "quickstart"}' > /dev/null
done

# Get access token (Step VIII)
TOKEN_RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/aap001/subscriptions/$SUB_ID/step-viii")
TOKEN=$(echo $TOKEN_RESPONSE | jq -r '.token')

echo "✅ Access Token: $TOKEN"
```

### Step 2: Use Your Token

```bash
# Make an authenticated request
curl http://localhost:8080/api/v1/beta/authz/metrics \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Expected Response:**
```json
{
  "cache": {
    "hits": 0,
    "misses": 0,
    "size": 0,
    "hit_rate": 0
  },
  "decisions": {
    "total": 0,
    "permit": 0,
    "deny": 0
  }
}
```

---

## Common Workflows

### Workflow 1: Create and Validate a Proof of Authorization

```bash
# Create PoA
POA_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/beta/poa \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "grantor": "alice@example.com",
    "grantee": "bob@example.com",
    "scope": ["read:documents", "write:reports"],
    "valid_from": "2025-01-01T00:00:00Z",
    "valid_until": "2025-12-31T23:59:59Z"
  }')

POA_ID=$(echo $POA_RESPONSE | jq -r '.id')
echo "PoA ID: $POA_ID"

# Validate the PoA
curl -s -X POST "http://localhost:8080/api/v1/beta/poa/$POA_ID/validate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "action": "read:documents",
    "resource": "/api/documents/report.pdf"
  }' | jq
```

**Expected Response:**
```json
{
  "valid": true,
  "poa_id": "poa_abc123",
  "action": "read:documents",
  "validated_at": "2025-11-15T10:35:00Z"
}
```

### Workflow 2: Evaluate Authorization

```bash
curl -s -X POST http://localhost:8080/api/v1/beta/authz/evaluate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "subject": "alice@example.com",
    "action": "read",
    "resource": "/api/documents/report.pdf"
  }' | jq
```

**Expected Response:**
```json
{
  "decision": "permit",
  "policy_id": "policy_v1_abc123",
  "reason": "matched_allow_rule",
  "evaluated_at": "2025-11-15T10:36:00Z",
  "cache_hit": false
}
```

### Workflow 3: Verify Identity (PVP)

```bash
curl -s -X POST http://localhost:8080/api/v1/beta/pvp/verify \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "subject": "alice@example.com",
    "proof_type": "document",
    "proof_data": "base64_encoded_passport"
  }' | jq
```

**Expected Response:**
```json
{
  "valid": true,
  "subject": "alice@example.com",
  "verified_at": "2025-11-15T10:37:00Z",
  "proof_type": "document",
  "confidence_score": 0.95
}
```

---

## SDK Setup

### JavaScript/TypeScript

```bash
# Install the SDK
npm install @agentauth/client
# or
yarn add @agentauth/client
```

**Usage:**
```typescript
import { AgentAuthClient } from '@agentauth/client';

const client = new AgentAuthClient({
  baseURL: 'http://localhost:8080'
});

// Complete subscription flow and get token
const result = await client.completeSubscriptionFlow({
  client_id: 'my-app',
  scope: 'read write'
});

console.log('Token:', result.token);

// Create PoA
const poa = await client.createPoA({
  grantor: 'alice@example.com',
  grantee: 'bob@example.com',
  scope: ['read:documents'],
  valid_from: '2025-01-01T00:00:00Z',
  valid_until: '2025-12-31T23:59:59Z'
});

console.log('PoA created:', poa.id);
```

### Python

```bash
# Install the SDK
pip install agentauth-client
```

**Usage:**
```python
from agentauth_client import AgentAuthClient

client = AgentAuthClient(base_url='http://localhost:8080')

# Complete subscription flow and get token
result = client.complete_subscription_flow(
    client_id='my-app',
    scope='read write'
)

print(f'Token: {result["token"]}')

# Create PoA
poa = client.create_poa(
    grantor='alice@example.com',
    grantee='bob@example.com',
    scope=['read:documents'],
    valid_from='2025-01-01T00:00:00Z',
    valid_until='2025-12-31T23:59:59Z'
)

print(f'PoA created: {poa.id}')
```

---

## Troubleshooting

### Problem: Server won't start

**Solution:**
- Check if port 8080 is already in use:
  ```bash
  lsof -i :8080
  ```
- Use a different port:
  ```bash
  PORT=8081 go run ./cmd/web-server
  ```

### Problem: "401 Unauthorized" errors

**Solution:**
- Verify your access token is valid:
  ```bash
  curl -X POST http://localhost:8080/api/v1/token/validate \
    -H "Content-Type: application/json" \
    -d "{\"token\": \"$TOKEN\"}" | jq
  ```
- Get a fresh token by completing the subscription flow again

### Problem: "429 Rate Limit Exceeded"

**Solution:**
- Wait for the rate limit window to reset (check `X-RateLimit-Reset` header)
- Implement exponential backoff in your client
- Request a higher rate limit for production use

### Problem: Connection refused

**Solution:**
- Ensure the server is running:
  ```bash
  curl http://localhost:8080/api/v1/beta/health
  ```
- Check firewall settings
- Verify Docker container is running (if using Docker)

### Problem: Invalid subscription flow

**Solution:**
- Make sure to execute steps in order (I → II → III → ... → VIII)
- Don't skip steps
- Use the same subscription ID for all steps

---

## Interactive API Documentation

Visit the interactive API documentation to explore all endpoints:

- **Swagger UI**: http://localhost:8080/api/docs/swagger
- **ReDoc**: http://localhost:8080/api/docs/redoc
- **OpenAPI Spec**: http://localhost:8080/api/docs/openapi.yaml

Try out endpoints directly from your browser!

---

## Configuration

### Environment Variables

```bash
# Server Configuration
export PORT=8080
export AGENTAUTH_ENV=development

# Security
export AGENTAUTH_CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001
export AGENTAUTH_RATE_LIMIT_RPS=100

# Features
export AGENTAUTH_AAP-001_ENABLED=1

# Logging
export AGENTAUTH_AUDIT_LOG_FILE=/var/log/agentauth/audit.log
export AGENTAUTH_AUDIT_LOG_STDOUT=1
```

### Configuration File

Create `config.yaml`:

```yaml
server:
  port: 8080
  host: 0.0.0.0

security:
  cors_allowed_origins:
    - http://localhost:3000
    - http://localhost:3001
  rate_limit_rps: 100
  hsts_max_age: 31536000

features:
  aap001_enabled: true
  metrics_enabled: true

logging:
  audit_log_file: /var/log/agentauth/audit.log
  audit_log_stdout: true
```

---

## Performance Tips

1. **Use connection pooling** in your HTTP client
2. **Cache authorization decisions** when appropriate
3. **Implement retry logic** with exponential backoff
4. **Monitor rate limits** and adjust request patterns
5. **Use batch operations** when available

---

## Security Checklist

- [ ] Use HTTPS in production
- [ ] Store tokens securely (server-side only)
- [ ] Implement CORS properly
- [ ] Enable audit logging
- [ ] Set up monitoring and alerts
- [ ] Rotate keys regularly
- [ ] Use appropriate token expiration times
- [ ] Validate all inputs
- [ ] Implement rate limiting on your client

---

## Next Steps

### 🚀 Build Something

- **OAuth 2.0 Flow**: Implement full OAuth flow in your app
- **PoA System**: Build a delegation management system
- **Policy Engine**: Create custom authorization policies
- **Identity Platform**: Integrate PVP for identity verification

### 📚 Learn More

- **API Examples**: [API_EXAMPLES.md](./API_EXAMPLES.md)
- **Full API Reference**: http://localhost:8080/api/docs
- **Architecture**: [ARCHITECTURE_SOLUTION.md](../ARCHITECTURE_SOLUTION.md)
- **Security Guide**: [SECURITY_COMPLIANCE_GUIDE.md](../SECURITY_COMPLIANCE_GUIDE.md)

### 🤝 Get Help

- **GitHub Issues**: https://github.com/mauriciomferz/AgentAuth/issues
- **Discussions**: https://github.com/mauriciomferz/AgentAuth/discussions
- **Email**: support@agentauth.example.com

---

## Sample Application

Check out our sample applications:

- **React + AgentAuth**: [examples/react-app](../examples/react-app)
- **Python Flask**: [examples/flask-app](../examples/flask-app)
- **Go Service**: [examples/go-service](../examples/go-service)

---

## FAQ

**Q: How do I get an API key?**  
A: Use the AAP-001 subscription flow to obtain an access token. API keys are for service-to-service authentication (contact support).

**Q: Can I use AgentAuth without the subscription flow?**  
A: The subscription flow is recommended for full AAP-001 compliance, but you can also use direct token creation for simpler scenarios.

**Q: How long do tokens last?**  
A: Default token lifetime is 1 hour (3600 seconds). This can be configured during subscription.

**Q: Is AgentAuth production-ready?**  
A: AgentAuth is currently in beta. Review the [security guide](../SECURITY_COMPLIANCE_GUIDE.md) before production deployment.

**Q: Can I self-host AgentAuth?**  
A: Yes! AgentAuth is open source and can be self-hosted. See the [deployment guide](../DEPLOYMENT_GUIDE.md).

---

**🎉 Congratulations!** You're now ready to build with AgentAuth.

For more examples and advanced use cases, see [API_EXAMPLES.md](./API_EXAMPLES.md).
