# GAuth API Examples & Tutorials

Complete guide with examples in **curl**, **JavaScript**, **Python**, and **Go** for common workflows.

---

## Table of Contents

1. [RFC-0111 Subscription Flow](#rfc-0111-subscription-flow)
2. [Power of Attorney Management](#power-of-attorney-management)
3. [Token Management](#token-management)
4. [Authorization Evaluation](#authorization-evaluation)
5. [Identity Verification (PVP)](#identity-verification-pvp)
6. [Commercial Registry](#commercial-registry)
7. [Policy Management](#policy-management)

---

## RFC-0111 Subscription Flow

### Complete 8-Step Subscription Flow

#### curl

```bash
# Step I: Create Subscription
SUB_RESPONSE=$(curl -X POST http://localhost:8080/api/v1/rfc0111/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "my-app",
    "scope": "read write"
  }')

SUB_ID=$(echo $SUB_RESPONSE | jq -r '.id')
echo "Subscription ID: $SUB_ID"

# Step II: Authorizer Authentication (PVP)
curl -X POST "http://localhost:8080/api/v1/rfc0111/subscriptions/$SUB_ID/step-ii" \
  -H "Content-Type: application/json" \
  -d '{
    "proof_type": "document",
    "proof_data": "base64_encoded_document"
  }'

# Step III: Client Owner Identification
curl -X POST "http://localhost:8080/api/v1/rfc0111/subscriptions/$SUB_ID/step-iii"

# Step IV: Client Owner Authorization
curl -X POST "http://localhost:8080/api/v1/rfc0111/subscriptions/$SUB_ID/step-iv"

# Step V: Client Authorization
curl -X POST "http://localhost:8080/api/v1/rfc0111/subscriptions/$SUB_ID/step-v"

# Step VI: Resource Owner Identification
curl -X POST "http://localhost:8080/api/v1/rfc0111/subscriptions/$SUB_ID/step-vi"

# Step VII: Resource Owner Authorization
curl -X POST "http://localhost:8080/api/v1/rfc0111/subscriptions/$SUB_ID/step-vii"

# Step VIII: Resource Server Verification (Get Token)
TOKEN_RESPONSE=$(curl -X POST "http://localhost:8080/api/v1/rfc0111/subscriptions/$SUB_ID/step-viii")
TOKEN=$(echo $TOKEN_RESPONSE | jq -r '.token')
echo "Access Token: $TOKEN"
```

#### JavaScript/TypeScript

```typescript
import { GAuthClient } from './gauth-client';

async function subscriptionFlow() {
  const client = new GAuthClient({
    baseURL: 'http://localhost:8080'
  });

  // Complete entire flow automatically
  const result = await client.completeSubscriptionFlow({
    client_id: 'my-app',
    scope: 'read write'
  });

  console.log('Subscription ID:', result.subscription.id);
  console.log('Access Token:', result.token);
  
  // Client now has the access token set automatically
  const health = await client.health();
  console.log('Health:', health);
}

subscriptionFlow().catch(console.error);
```

#### Python

```python
from gauth_client import GAuthClient

def subscription_flow():
    client = GAuthClient(base_url='http://localhost:8080')
    
    # Complete entire flow automatically
    result = client.complete_subscription_flow(
        client_id='my-app',
        scope='read write'
    )
    
    print(f"Subscription ID: {result['subscription'].id}")
    print(f"Access Token: {result['token']}")
    
    # Client now has the access token set automatically
    health = client.health()
    print(f"Health: {health}")

if __name__ == '__main__':
    subscription_flow()
```

#### Go

```go
package main

import (
    "fmt"
    "github.com/yourorg/gauth-go-sdk"
)

func main() {
    client := gauth.NewClient("http://localhost:8080")
    
    // Create subscription
    sub, err := client.CreateSubscription(&gauth.SubscriptionCreate{
        ClientID: "my-app",
        Scope:    "read write",
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("Subscription ID: %s\n", sub.ID)
    
    // Execute steps II-VIII
    if err := client.ExecuteStepII(sub.ID, "document", "base64_data"); err != nil {
        panic(err)
    }
    if err := client.ExecuteStepIII(sub.ID); err != nil {
        panic(err)
    }
    if err := client.ExecuteStepIV(sub.ID); err != nil {
        panic(err)
    }
    if err := client.ExecuteStepV(sub.ID); err != nil {
        panic(err)
    }
    if err := client.ExecuteStepVI(sub.ID); err != nil {
        panic(err)
    }
    if err := client.ExecuteStepVII(sub.ID); err != nil {
        panic(err)
    }
    
    // Get token
    result, err := client.ExecuteStepVIII(sub.ID)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Access Token: %s\n", result.Token)
    client.SetAccessToken(result.Token)
}
```

---

## Power of Attorney Management

### Create, Validate, and Revoke PoA

#### curl

```bash
# Create PoA
POA_RESPONSE=$(curl -X POST http://localhost:8080/api/v1/beta/poa \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "grantor": "alice@example.com",
    "grantee": "bob@example.com",
    "scope": ["read:documents", "write:reports"],
    "valid_from": "2025-01-01T00:00:00Z",
    "valid_until": "2025-12-31T23:59:59Z",
    "resource_pattern": "/api/documents/*"
  }')

POA_ID=$(echo $POA_RESPONSE | jq -r '.id')
echo "PoA ID: $POA_ID"

# Validate PoA
curl -X POST "http://localhost:8080/api/v1/beta/poa/$POA_ID/validate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "action": "read:documents",
    "resource": "/api/documents/report.pdf"
  }' | jq

# List PoAs
curl -X GET "http://localhost:8080/api/v1/beta/poa?grantor=alice@example.com" \
  -H "Authorization: Bearer $TOKEN" | jq

# Update PoA
curl -X PUT "http://localhost:8080/api/v1/beta/poa/$POA_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "scope": ["read:documents", "write:reports", "delete:documents"],
    "valid_until": "2026-12-31T23:59:59Z"
  }' | jq

# Revoke PoA
curl -X DELETE "http://localhost:8080/api/v1/beta/poa/$POA_ID" \
  -H "Authorization: Bearer $TOKEN" | jq
```

#### JavaScript/TypeScript

```typescript
async function poaWorkflow(client: GAuthClient) {
  // Create PoA
  const poa = await client.createPoA({
    grantor: 'alice@example.com',
    grantee: 'bob@example.com',
    scope: ['read:documents', 'write:reports'],
    valid_from: '2025-01-01T00:00:00Z',
    valid_until: '2025-12-31T23:59:59Z',
    resource_pattern: '/api/documents/*'
  });
  console.log('PoA created:', poa.id);
  
  // Validate PoA
  const validation = await client.validatePoA(poa.id, {
    action: 'read:documents',
    resource: '/api/documents/report.pdf'
  });
  console.log('Valid:', validation.valid);
  
  // List PoAs
  const list = await client.listPoAs({
    grantor: 'alice@example.com'
  });
  console.log('Total PoAs:', list.total);
  
  // Update PoA
  const updated = await client.updatePoA(poa.id, {
    scope: ['read:documents', 'write:reports', 'delete:documents']
  });
  console.log('Updated scope:', updated.scope);
  
  // Revoke PoA
  await client.revokePoA(poa.id);
  console.log('PoA revoked');
}
```

#### Python

```python
def poa_workflow(client: GAuthClient):
    # Create PoA
    poa = client.create_poa(
        grantor='alice@example.com',
        grantee='bob@example.com',
        scope=['read:documents', 'write:reports'],
        valid_from='2025-01-01T00:00:00Z',
        valid_until='2025-12-31T23:59:59Z',
        resource_pattern='/api/documents/*'
    )
    print(f'PoA created: {poa.id}')
    
    # Validate PoA
    validation = client.validate_poa(
        poa_id=poa.id,
        action='read:documents',
        resource='/api/documents/report.pdf'
    )
    print(f"Valid: {validation['valid']}")
    
    # List PoAs
    poas = client.list_poas(grantor='alice@example.com')
    print(f"Total PoAs: {poas['total']}")
    
    # Update PoA
    updated = client.update_poa(
        poa_id=poa.id,
        scope=['read:documents', 'write:reports', 'delete:documents']
    )
    print(f'Updated scope: {updated.scope}')
    
    # Revoke PoA
    client.revoke_poa(poa.id)
    print('PoA revoked')
```

---

## Token Management

### Create, Validate, and Revoke Tokens

#### curl

```bash
# Create token
TOKEN_RESP=$(curl -X POST http://localhost:8080/api/v1/token/create \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "my-app",
    "scope": "read write",
    "expires_in": 3600
  }')

TOKEN=$(echo $TOKEN_RESP | jq -r '.access_token')
echo "Token: $TOKEN"

# Validate token
curl -X POST http://localhost:8080/api/v1/token/validate \
  -H "Content-Type: application/json" \
  -d "{\"token\": \"$TOKEN\"}" | jq

# Revoke token
curl -X POST http://localhost:8080/api/v1/token/revoke \
  -H "Content-Type: application/json" \
  -d "{\"token\": \"$TOKEN\"}" | jq
```

#### JavaScript/TypeScript

```typescript
async function tokenWorkflow(client: GAuthClient) {
  // Create token
  const token = await client.createToken({
    client_id: 'my-app',
    scope: 'read write',
    expires_in: 3600
  });
  console.log('Token:', token.access_token);
  
  // Validate token
  const validation = await client.validateToken(token.access_token);
  console.log('Valid:', validation.valid);
  console.log('Claims:', validation.claims);
  
  // Revoke token
  await client.revokeToken(token.access_token);
  console.log('Token revoked');
}
```

#### Python

```python
def token_workflow(client: GAuthClient):
    # Create token
    token = client.create_token(
        client_id='my-app',
        scope='read write',
        expires_in=3600
    )
    print(f'Token: {token.access_token}')
    
    # Validate token
    validation = client.validate_token(token.access_token)
    print(f"Valid: {validation['valid']}")
    print(f"Claims: {validation['claims']}")
    
    # Revoke token
    client.revoke_token(token.access_token)
    print('Token revoked')
```

---

## Authorization Evaluation

### Evaluate Policy-Based Authorization

#### curl

```bash
# Evaluate authorization
curl -X POST http://localhost:8080/api/v1/beta/authz/evaluate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "subject": "alice@example.com",
    "action": "read",
    "resource": "/api/documents/report.pdf",
    "context": {
      "ip_address": "192.168.1.100",
      "department": "engineering"
    }
  }' | jq

# Get authorization metrics
curl -X GET http://localhost:8080/api/v1/beta/authz/metrics \
  -H "Authorization: Bearer $TOKEN" | jq
```

#### JavaScript/TypeScript

```typescript
async function authzWorkflow(client: GAuthClient) {
  // Evaluate authorization
  const decision = await client.evaluateAuthorization({
    subject: 'alice@example.com',
    action: 'read',
    resource: '/api/documents/report.pdf',
    context: {
      ip_address: '192.168.1.100',
      department: 'engineering'
    }
  });
  
  console.log('Decision:', decision.decision); // 'permit' or 'deny'
  console.log('Policy ID:', decision.policy_id);
  console.log('Reason:', decision.reason);
  
  // Get metrics
  const metrics = await client.getAuthzMetrics();
  console.log('Cache hit rate:', metrics.cache.hit_rate);
  console.log('Total decisions:', metrics.decisions.total);
}
```

#### Python

```python
def authz_workflow(client: GAuthClient):
    # Evaluate authorization
    decision = client.evaluate_authorization(
        subject='alice@example.com',
        action='read',
        resource='/api/documents/report.pdf',
        context={
            'ip_address': '192.168.1.100',
            'department': 'engineering'
        }
    )
    
    print(f"Decision: {decision['decision']}")  # 'permit' or 'deny'
    print(f"Policy ID: {decision['policy_id']}")
    print(f"Reason: {decision['reason']}")
    
    # Get metrics
    metrics = client.get_authz_metrics()
    print(f"Cache hit rate: {metrics['cache']['hit_rate']}")
    print(f"Total decisions: {metrics['decisions']['total']}")
```

---

## Identity Verification (PVP)

### Verify Person Identity

#### curl

```bash
# Verify with document proof
curl -X POST http://localhost:8080/api/v1/beta/pvp/verify \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "subject": "alice@example.com",
    "proof_type": "document",
    "proof_data": "base64_encoded_passport_scan"
  }' | jq

# Verify with biometric proof
curl -X POST http://localhost:8080/api/v1/beta/pvp/verify \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "subject": "bob@example.com",
    "proof_type": "biometric",
    "proof_data": "base64_encoded_fingerprint"
  }' | jq
```

#### JavaScript/TypeScript

```typescript
async function pvpWorkflow(client: GAuthClient) {
  // Verify with document
  const result = await client.verifyPVP({
    subject: 'alice@example.com',
    proof_type: 'document',
    proof_data: 'base64_encoded_passport_scan'
  });
  
  console.log('Valid:', result.valid);
  console.log('Confidence score:', result.confidence_score);
  console.log('Verified at:', result.verified_at);
}
```

#### Python

```python
def pvp_workflow(client: GAuthClient):
    # Verify with document
    result = client.verify_pvp(
        subject='alice@example.com',
        proof_type='document',
        proof_data='base64_encoded_passport_scan'
    )
    
    print(f"Valid: {result['valid']}")
    print(f"Confidence score: {result['confidence_score']}")
    print(f"Verified at: {result['verified_at']}")
```

---

## Commercial Registry

### Verify Entity and Signatory

#### curl

```bash
# Verify entity
curl -X POST http://localhost:8080/api/v1/beta/registry/verify-entity \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "entity_id": "HRB12345-DE",
    "registry": "germany_hrb"
  }' | jq

# Verify signatory
curl -X POST http://localhost:8080/api/v1/beta/registry/verify-signatory \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "signatory_id": "12345678-GB",
    "entity_id": "HRB12345-DE",
    "role": "director"
  }' | jq
```

#### JavaScript/TypeScript

```typescript
async function registryWorkflow(client: GAuthClient) {
  // Verify entity
  const entity = await client.verifyEntity({
    entity_id: 'HRB12345-DE',
    registry: 'germany_hrb'
  });
  console.log('Entity valid:', entity.valid);
  console.log('Entity name:', entity.entity_name);
  console.log('Status:', entity.status);
  
  // Verify signatory
  const signatory = await client.verifySignatory({
    signatory_id: '12345678-GB',
    entity_id: 'HRB12345-DE',
    role: 'director'
  });
  console.log('Signatory valid:', signatory.valid);
  console.log('Authority level:', signatory.authority_level);
}
```

#### Python

```python
def registry_workflow(client: GAuthClient):
    # Verify entity
    entity = client.verify_entity(
        entity_id='HRB12345-DE',
        registry='germany_hrb'
    )
    print(f"Entity valid: {entity['valid']}")
    print(f"Entity name: {entity['entity_name']}")
    print(f"Status: {entity['status']}")
    
    # Verify signatory
    signatory = client.verify_signatory(
        signatory_id='12345678-GB',
        entity_id='HRB12345-DE',
        role='director'
    )
    print(f"Signatory valid: {signatory['valid']}")
    print(f"Authority level: {signatory['authority_level']}")
```

---

## Policy Management

### Add and Evaluate Policies

#### curl

```bash
# Add policy bundle
curl -X POST http://localhost:8080/api/v1/beta/policy/bundles \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "version": "v1.0.0",
    "policies": [
      {
        "id": "policy1",
        "effect": "allow",
        "subjects": ["alice@example.com"],
        "actions": ["read", "write"],
        "resources": ["/api/documents/*"]
      }
    ]
  }' | jq

# List active policies
curl -X GET http://localhost:8080/api/v1/beta/policy/head/policies \
  -H "Authorization: Bearer $TOKEN" | jq

# Evaluate policy
curl -X POST http://localhost:8080/api/v1/beta/policy/evaluate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "policy": {
      "id": "test-policy",
      "effect": "allow",
      "subjects": ["alice@example.com"],
      "actions": ["read"],
      "resources": ["/api/documents/*"]
    },
    "input": {
      "subject": "alice@example.com",
      "action": "read",
      "resource": "/api/documents/report.pdf"
    }
  }' | jq
```

---

## Complete End-to-End Example

### Full Workflow: Subscription → PoA → Authorization

```bash
#!/bin/bash

# 1. Complete RFC-0111 subscription flow
echo "=== Step 1: RFC-0111 Subscription Flow ==="
SUB_RESP=$(curl -s -X POST http://localhost:8080/api/v1/rfc0111/subscriptions \
  -H "Content-Type: application/json" \
  -d '{"client_id": "my-app", "scope": "read write"}')
SUB_ID=$(echo $SUB_RESP | jq -r '.id')
echo "Subscription ID: $SUB_ID"

# Execute steps II-VIII
curl -s -X POST "http://localhost:8080/api/v1/rfc0111/subscriptions/$SUB_ID/step-ii" \
  -H "Content-Type: application/json" \
  -d '{"proof_type": "document", "proof_data": "auto"}' > /dev/null
  
for STEP in iii iv v vi vii; do
  curl -s -X POST "http://localhost:8080/api/v1/rfc0111/subscriptions/$SUB_ID/step-$STEP" > /dev/null
done

TOKEN_RESP=$(curl -s -X POST "http://localhost:8080/api/v1/rfc0111/subscriptions/$SUB_ID/step-viii")
TOKEN=$(echo $TOKEN_RESP | jq -r '.token')
echo "Access Token: ${TOKEN:0:20}..."

# 2. Create Power of Attorney
echo -e "\n=== Step 2: Create PoA ==="
POA_RESP=$(curl -s -X POST http://localhost:8080/api/v1/beta/poa \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "grantor": "alice@example.com",
    "grantee": "bob@example.com",
    "scope": ["read:documents"],
    "valid_from": "2025-01-01T00:00:00Z",
    "valid_until": "2025-12-31T23:59:59Z"
  }')
POA_ID=$(echo $POA_RESP | jq -r '.id')
echo "PoA ID: $POA_ID"

# 3. Validate PoA
echo -e "\n=== Step 3: Validate PoA ==="
curl -s -X POST "http://localhost:8080/api/v1/beta/poa/$POA_ID/validate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"action": "read:documents"}' | jq '.valid'

# 4. Evaluate authorization
echo -e "\n=== Step 4: Evaluate Authorization ==="
curl -s -X POST http://localhost:8080/api/v1/beta/authz/evaluate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "subject": "bob@example.com",
    "action": "read",
    "resource": "/api/documents/report.pdf"
  }' | jq '.decision'

echo -e "\n=== Workflow Complete ==="_"
```

---

## Error Handling Examples

### JavaScript/TypeScript

```typescript
import { GAuthClient, GAuthError } from './gauth-client';

async function handleErrors() {
  const client = new GAuthClient({ baseURL: 'http://localhost:8080' });
  
  try {
    const poa = await client.createPoA({
      grantor: 'alice@example.com',
      grantee: 'bob@example.com',
      scope: ['read:documents'],
      valid_from: '2025-01-01T00:00:00Z',
      valid_until: '2025-12-31T23:59:59Z'
    });
  } catch (error) {
    if (error instanceof GAuthError) {
      console.error('GAuth Error:', error.error);
      console.error('Description:', error.error_description);
      console.error('Timestamp:', error.timestamp);
      
      // Handle specific errors
      if (error.error === 'rate_limit_exceeded') {
        console.log('Rate limit hit, retrying after delay...');
        await new Promise(resolve => setTimeout(resolve, 5000));
      }
    } else {
      console.error('Unexpected error:', error);
    }
  }
}
```

### Python

```python
from gauth_client import GAuthClient, GAuthError, GAuthHTTPError

def handle_errors():
    client = GAuthClient(base_url='http://localhost:8080')
    
    try:
        poa = client.create_poa(
            grantor='alice@example.com',
            grantee='bob@example.com',
            scope=['read:documents'],
            valid_from='2025-01-01T00:00:00Z',
            valid_until='2025-12-31T23:59:59Z'
        )
    except GAuthHTTPError as e:
        print(f'HTTP {e.status_code}: {e.error}')
        print(f'Description: {e.error_description}')
        
        # Handle specific errors
        if e.error == 'rate_limit_exceeded':
            print('Rate limit hit, retrying after delay...')
            time.sleep(5)
    except GAuthError as e:
        print(f'GAuth Error: {e.error}')
        print(f'Description: {e.error_description}')
```

---

## Best Practices

1. **Always use HTTPS in production**
2. **Store tokens securely** (never in client-side storage)
3. **Implement exponential backoff** for rate limits
4. **Validate tokens on every request**
5. **Use appropriate scopes** (principle of least privilege)
6. **Monitor token expiration** and refresh proactively
7. **Log all security events** for audit trails
8. **Handle errors gracefully** with user-friendly messages

---

## Next Steps

- **Quick Start Guide**: See [QUICKSTART.md](./QUICKSTART.md)
- **API Reference**: Visit http://localhost:8080/api/docs
- **SDK Documentation**: Check individual SDK folders
- **Production Deployment**: See [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md)
