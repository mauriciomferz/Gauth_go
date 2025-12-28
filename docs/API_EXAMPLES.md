# GAuth+ API Usage Examples

Complete examples for using the GAuth+ admin API endpoints.

---

## Audit Export Examples

### Export Last 24 Hours (JSON)
```bash
curl -X POST http://localhost:8080/api/admin/audit/export \
  -H "Content-Type: application/json" \
  -d '{
    "format": "json",
    "dateRange": "last-24h",
    "compressed": false
  }'

# Response:
{
  "jobId": "uuid-here",
  "status": "pending",
  "format": "json",
  "createdAt": "2025-12-29T00:00:00Z",
  "expiresAt": "2025-12-30T00:00:00Z"
}
```

### Export Last Week (CSV, Compressed)
```bash
curl -X POST http://localhost:8080/api/admin/audit/export \
  -H "Content-Type: application/json" \
  -d '{
    "format": "csv",
    "dateRange": "last-7d",
    "compressed": true,
    "category": "security",
    "severity": "high"
  }'
```

### Check Export Status
```bash
JOB_ID="your-job-id-here"
curl http://localhost:8080/api/admin/audit/export/$JOB_ID | jq .

# Response when complete:
{
  "jobId": "...",
  "status": "completed",
  "format": "json",
  "fileSize": 1024,
  "totalEvents": 150,
  "completedAt": "2025-12-29T00:05:00Z"
}
```

### Download Completed Export
```bash
JOB_ID="your-job-id-here"
curl -o audit-export.json \
  http://localhost:8080/api/admin/audit/export/$JOB_ID/download
```

### Delete Export Job
```bash
curl -X DELETE http://localhost:8080/api/admin/audit/export/$JOB_ID
```

---

## API Key Management Examples

### Create API Key
```bash
curl -X POST http://localhost:8080/api/admin/api-keys \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "acme-corp",
    "keyName": "Production Service",
    "description": "Main backend service API key",
    "scopes": ["poa:read", "poa:write", "delegation:create"],
    "rateLimitPerMinute": 100,
    "rateLimitPerHour": 5000,
    "createdBy": "admin@acme.com"
  }'

# Response (secretKey only shown once!):
{
  "apiKey": {
    "id": "key-123",
    "keyId": "gauth_sk_...",
    "name": "Production Service",
    "scopes": ["poa:read", "poa:write"],
    "createdAt": "2025-12-29T00:00:00Z"
  },
  "secretKey": "gauth_sk_live_abc123...",
  "message": "Save this key securely - it won't be shown again"
}
```

### List API Keys
```bash
curl "http://localhost:8080/api/admin/api-keys?tenant_id=acme-corp" | jq .
```

### Get API Key Details
```bash
curl http://localhost:8080/api/admin/api-keys/key-123 | jq .
```

### Get API Key Usage Statistics
```bash
curl "http://localhost:8080/api/admin/api-keys/key-123/usage?tenant_id=acme-corp" | jq .

# Response:
{
  "totalRequests": 1500,
  "successRate": 98.5,
  "avgResponseTime": 45,
  "requestsByEndpoint": {
    "/api/v1/poa": 800,
    "/api/v1/delegation": 700
  }
}
```

### Revoke API Key
```bash
curl -X POST http://localhost:8080/api/admin/api-keys/key-123/revoke \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "acme-corp"}'
```

---

## Audit Events Examples

### List Recent Events
```bash
curl "http://localhost:8080/api/admin/audit/events?limit=10&offset=0" | jq .
```

### Filter by Category
```bash
curl "http://localhost:8080/api/admin/audit/events?category=security&severity=high" | jq .
```

### Filter by User and Action
```bash
curl "http://localhost:8080/api/admin/audit/events?actor=user@example.com&action=poa_created" | jq .
```

---

## Compliance Reports Examples

### Get All Compliance Reports
```bash
curl http://localhost:8080/api/admin/audit/compliance | jq .

# Response:
{
  "reports": [
    {
      "id": "report-1",
      "framework": "GDPR",
      "standard": "General Data Protection Regulation",
      "status": "compliant",
      "coverage": 95,
      "violations": 2,
      "lastAudit": "2025-12-29T00:00:00Z"
    }
  ]
}
```

---

## Metrics Examples

### Get Audit Metrics
```bash
curl http://localhost:8080/api/admin/audit/metrics | jq .

# Response:
{
  "total_events": 5000,
  "events_by_category": {
    "auth": 1500,
    "admin": 800,
    "system": 2700
  },
  "events_by_severity": {
    "critical": 5,
    "high": 50,
    "medium": 500,
    "low": 4445
  },
  "siem_integrations": {
    "total": 2,
    "active": 2,
    "events_sent": 4950
  }
}
```

---

## Complete Workflow Example

### Complete Audit Export Workflow
```bash
#!/bin/bash

# 1. Create export job
echo "Creating export job..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/admin/audit/export \
  -H "Content-Type: application/json" \
  -d '{
    "format": "json",
    "dateRange": "last-7d",
    "compressed": false
  }')

JOB_ID=$(echo $RESPONSE | jq -r '.jobId')
echo "Job ID: $JOB_ID"

# 2. Wait for completion
echo "Waiting for export to complete..."
while true; do
  STATUS=$(curl -s http://localhost:8080/api/admin/audit/export/$JOB_ID | jq -r '.status')
  echo "Status: $STATUS"
  
  if [ "$STATUS" = "completed" ]; then
    break
  fi
  
  sleep 2
done

# 3. Download export
echo "Downloading export..."
curl -o audit-export-$(date +%Y%m%d).json \
  http://localhost:8080/api/admin/audit/export/$JOB_ID/download

echo "Export complete!"

# 4. Optional: Delete job
curl -X DELETE http://localhost:8080/api/admin/audit/export/$JOB_ID
```

---

## Python Examples

### Python: Export Audit Logs
```python
import requests
import time
import json

BASE_URL = "http://localhost:8080"

# Create export job
response = requests.post(
    f"{BASE_URL}/api/admin/audit/export",
    json={
        "format": "json",
        "dateRange": "last-24h",
        "compressed": False
    }
)
job_id = response.json()["jobId"]
print(f"Job ID: {job_id}")

# Wait for completion
while True:
    status_response = requests.get(
        f"{BASE_URL}/api/admin/audit/export/{job_id}"
    )
    status = status_response.json()["status"]
    
    if status == "completed":
        break
    
    time.sleep(2)

# Download export
download_response = requests.get(
    f"{BASE_URL}/api/admin/audit/export/{job_id}/download"
)

with open("audit-export.json", "wb") as f:
    f.write(download_response.content)

print("Export downloaded successfully")
```

### Python: Create API Key
```python
import requests

response = requests.post(
    "http://localhost:8080/api/admin/api-keys",
    json={
        "tenantId": "my-tenant",
        "keyName": "Python Service",
        "description": "API key for Python application",
        "scopes": ["poa:read"],
        "rateLimitPerMinute": 60,
        "createdBy": "admin"
    }
)

result = response.json()
print(f"API Key ID: {result['apiKey']['id']}")
print(f"Secret Key: {result['secretKey']}")
print("SAVE THIS KEY - it won't be shown again!")
```
