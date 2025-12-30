# Audit Log Export - API Reference

## Overview

The Audit Log Export API provides comprehensive export functionality for audit events with support for multiple formats, filtering, compression, and async job processing.

**Base Path**: `/api/v1/admin/audit`

---

## Authentication

All endpoints require authentication via Bearer token:
```
Authorization: Bearer <jwt_token>
```

---

## Endpoints

### 1. Create Export Job

**Endpoint**: `POST /audit/export`

**Description**: Initiates an async export job for audit events.

**Request Body**:
```json
{
  "format": "json",           // Required: json|csv|syslog|cef
  "dateRange": "last-24h",    // Required: last-1h|last-24h|last-7d|last-30d|all|custom
  "category": "access",       // Optional: Filter by category
  "severity": "high",         // Optional: critical|high|medium|low|info
  "actor": "user-123",        // Optional: Filter by user ID
  "action": "poa.create",     // Optional: Filter by action
  "resourceType": "poa",      // Optional: Filter by resource type
  "compressed": true          // Optional: Enable gzip compression (default: false)
}
```

**Custom Date Range Format**:
```json
{
  "format": "csv",
  "dateRange": "2025-11-01,2025-11-26"  // YYYY-MM-DD,YYYY-MM-DD
}
```

**Response** (202 Accepted):
```json
{
  "jobId": "abc-123-def-456",
  "status": "pending",
  "format": "json",
  "createdAt": "2025-11-26T10:00:00Z",
  "expiresAt": "2025-11-27T10:00:00Z"
}
```

**Status Codes**:
- `202 Accepted` - Export job created successfully
- `400 Bad Request` - Invalid format or date range
- `401 Unauthorized` - Missing or invalid token
- `500 Internal Server Error` - Failed to create job

---

### 2. Get Export Status

**Endpoint**: `GET /audit/export/:id`

**Description**: Retrieves the status of an export job.

**Path Parameters**:
- `id` - Export job ID (from create response)

**Response** (200 OK):
```json
{
  "jobId": "abc-123-def-456",
  "status": "completed",      // pending|processing|completed|failed
  "format": "json",
  "compressed": true,
  "totalEvents": 1234,
  "fileSize": 524288,         // bytes
  "createdAt": "2025-11-26T10:00:00Z",
  "completedAt": "2025-11-26T10:01:30Z",
  "expiresAt": "2025-11-27T10:00:00Z"
}
```

**Status Codes**:
- `200 OK` - Job found
- `404 Not Found` - Job ID not found
- `401 Unauthorized` - Missing or invalid token

---

### 3. Download Export File

**Endpoint**: `GET /audit/export/:id/download`

**Description**: Downloads the completed export file.

**Path Parameters**:
- `id` - Export job ID

**Response Headers**:
```
Content-Disposition: attachment; filename=audit-export-20251126-100000.json.gz
Content-Type: application/gzip
Content-Length: 524288
```

**Status Codes**:
- `200 OK` - File download started
- `400 Bad Request` - Export not ready (still processing)
- `404 Not Found` - Job ID not found
- `401 Unauthorized` - Missing or invalid token

---

### 4. Delete Export Job

**Endpoint**: `DELETE /audit/export/:id`

**Description**: Deletes an export job and its file.

**Path Parameters**:
- `id` - Export job ID

**Response** (200 OK):
```json
{
  "message": "export job deleted"
}
```

**Status Codes**:
- `200 OK` - Job deleted successfully
- `404 Not Found` - Job ID not found
- `401 Unauthorized` - Missing or invalid token

---

## Export Formats

### JSON Format

**Structure**:
```json
{
  "exported_at": "2025-11-26T10:00:00Z",
  "total": 2,
  "events": [
    {
      "id": "evt-123",
      "tenantId": "tenant-1",
      "timestamp": "2025-11-26T10:00:00Z",
      "eventType": "audit.event",
      "category": "access",
      "severity": "medium",
      "userId": "user-123",
      "userName": "John Doe",
      "userRole": "admin",
      "action": "poa.create",
      "resourceType": "poa",
      "resourceId": "poa-456",
      "resourceName": "Power of Attorney #456",
      "status": "success",
      "statusCode": 201,
      "ipAddress": "192.168.1.1",
      "userAgent": "Mozilla/5.0...",
      "requestId": "req-789",
      "sessionId": "sess-abc",
      "correlationId": "corr-xyz",
      "beforeState": {},
      "afterState": {"status": "active"},
      "changes": {"status": "created"},
      "complianceFramework": "SOC2",
      "riskLevel": "low",
      "requiresReview": false,
      "hash": "sha256:abcdef...",
      "previousHash": "sha256:123456..."
    }
  ]
}
```

**Content-Type**: `application/json` (or `application/gzip` if compressed)

---

### CSV Format

**Structure**:
```csv
ID,Timestamp,TenantID,UserID,Action,ResourceID,ResourceType,Status,Category,Severity,IPAddress,UserAgent
evt-123,2025-11-26T10:00:00Z,tenant-1,user-123,poa.create,poa-456,poa,success,access,medium,192.168.1.1,Mozilla/5.0...
evt-124,2025-11-26T10:05:00Z,tenant-1,user-456,poa.revoke,poa-789,poa,success,access,high,192.168.1.2,Mozilla/5.0...
```

**Content-Type**: `text/csv` (or `application/gzip` if compressed)

**Features**:
- Header row included
- Comma-separated values
- Quoted fields for safety
- RFC 4180 compliant

---

### Syslog Format (RFC 5424)

**Structure**:
```
<18>1 2025-11-26T10:00:00Z agentauth-audit - - - [tenant="tenant-1" user="user-123" action="poa.create" resource="poa-456" status="success"] poa.create
<19>1 2025-11-26T10:05:00Z agentauth-audit - - - [tenant="tenant-1" user="user-456" action="poa.revoke" resource="poa-789" status="success"] poa.revoke
```

**Content-Type**: `text/plain` (or `application/gzip` if compressed)

**Format**:
- Priority: `<18>` (facility 2, severity based on event severity)
- Version: `1` (RFC 5424)
- Timestamp: ISO 8601 format
- Hostname: `agentauth-audit`
- Structured data: Key audit fields
- Message: Action performed

**Priority Mapping**:
- Critical: 18 (facility 2, severity 2)
- High: 19 (facility 2, severity 3)
- Medium: 20 (facility 2, severity 4)
- Low: 21 (facility 2, severity 5)
- Info: 22 (facility 2, severity 6)

---

### CEF Format (Common Event Format)

**Structure**:
```
CEF:0|AgentAuth Community|AgentAuth|1.0|access|poa.create|5|rt=1732618800000 tenantId=tenant-1 suser=user-123 act=poa.create src=192.168.1.1 outcome=success cat=access
CEF:0|AgentAuth Community|AgentAuth|1.0|access|poa.revoke|8|rt=1732619100000 tenantId=tenant-1 suser=user-456 act=poa.revoke src=192.168.1.2 outcome=success cat=access
```

**Content-Type**: `text/plain` (or `application/gzip` if compressed)

**Format**:
- Version: `CEF:0`
- Device Vendor: `AgentAuth Community`
- Device Product: `AgentAuth`
- Device Version: `1.0`
- Signature ID: Event category
- Name: Action performed
- Severity: 0-10 (mapped from audit severity)
- Extension: Key-value pairs

**Severity Mapping**:
- Critical: 10
- High: 8
- Medium: 5
- Low: 3
- Info: 1

**Extension Fields**:
- `rt`: Receipt time (milliseconds since epoch)
- `tenantId`: Tenant identifier
- `suser`: Source user
- `act`: Action performed
- `src`: Source IP address
- `outcome`: Event outcome (success/failure)
- `cat`: Event category

---

## Date Range Options

### Predefined Ranges

| Value | Description | Example |
|-------|-------------|---------|
| `last-1h` | Last hour | Last 60 minutes |
| `last-24h` | Last 24 hours | Last day |
| `last-7d` | Last 7 days | Last week |
| `last-30d` | Last 30 days | Last month |
| `all` | All time | From 2020-01-01 |

### Custom Range

**Format**: `YYYY-MM-DD,YYYY-MM-DD`

**Examples**:
- `2025-11-01,2025-11-26` - November 2025
- `2025-01-01,2025-12-31` - Year 2025
- `2025-11-26,2025-11-26` - Single day

**Notes**:
- Start date is inclusive (00:00:00)
- End date is inclusive (23:59:59)
- Dates are in UTC timezone

---

## Filtering Options

### Available Filters

| Filter | Type | Values | Example |
|--------|------|--------|---------|
| `category` | string | access, security, compliance, governance, admin | `"category": "access"` |
| `severity` | string | critical, high, medium, low, info | `"severity": "critical"` |
| `actor` | string | User ID | `"actor": "user-123"` |
| `action` | string | Action type | `"action": "poa.create"` |
| `resourceType` | string | Resource type | `"resourceType": "poa"` |

### Combining Filters

All filters are combined with AND logic:

```json
{
  "format": "json",
  "dateRange": "last-7d",
  "category": "security",
  "severity": "critical",
  "action": "poa.revoke"
}
```

This returns only events that match ALL conditions:
- Last 7 days
- Security category
- Critical severity
- Revoke action

---

## Compression

### Gzip Compression

**Enable**: Set `"compressed": true` in request

**Benefits**:
- 40-70% size reduction
- Faster downloads
- Reduced bandwidth

**File Extension**: `.gz` appended (e.g., `audit-export-20251126.json.gz`)

**Decompression**:
```bash
# Linux/Mac
gunzip audit-export-20251126.json.gz

# Or extract directly
gzip -dc audit-export-20251126.json.gz > audit-export.json
```

---

## Limits and Quotas

### Per Export
- **Max Events**: 10,000 per export
- **Max File Size**: ~50 MB (uncompressed)
- **Job Expiration**: 24 hours
- **Processing Time**: 1-5 seconds (for 10,000 events)

### Rate Limits
- **Create Job**: 10 requests/minute
- **Status Check**: 60 requests/minute
- **Download**: 5 downloads/minute
- **Delete**: 10 requests/minute

---

## Error Responses

### 400 Bad Request
```json
{
  "error": "invalid date range: invalid format"
}
```

### 404 Not Found
```json
{
  "error": "export job not found"
}
```

### 500 Internal Server Error
```json
{
  "error": "failed to create export job: database connection error"
}
```

---

## Usage Examples

### Example 1: Export Last 24 Hours (JSON)

```bash
# Create job
curl -X POST http://localhost:8080/api/v1/admin/audit/export \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "json",
    "dateRange": "last-24h",
    "compressed": true
  }'

# Response: {"jobId":"abc-123",...}

# Check status
curl http://localhost:8080/api/v1/admin/audit/export/abc-123 \
  -H "Authorization: Bearer $TOKEN"

# Download
curl -O http://localhost:8080/api/v1/admin/audit/export/abc-123/download \
  -H "Authorization: Bearer $TOKEN"
```

### Example 2: Export Critical Security Events (CSV)

```bash
curl -X POST http://localhost:8080/api/v1/admin/audit/export \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "csv",
    "dateRange": "last-7d",
    "category": "security",
    "severity": "critical"
  }'
```

### Example 3: Export for SIEM (Syslog)

```bash
curl -X POST http://localhost:8080/api/v1/admin/audit/export \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "syslog",
    "dateRange": "last-30d",
    "compressed": true
  }'
```

### Example 4: Export Custom Date Range (CEF)

```bash
curl -X POST http://localhost:8080/api/v1/admin/audit/export \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "cef",
    "dateRange": "2025-11-01,2025-11-26",
    "actor": "admin-123"
  }'
```

---

## SIEM Integration

### Splunk
```bash
# Export as CEF
# Import into Splunk using CEF add-on
```

### Elastic (ELK)
```bash
# Export as JSON
# Use Logstash to ingest JSON files
```

### IBM QRadar
```bash
# Export as Syslog
# Configure QRadar log source
```

### Azure Sentinel
```bash
# Export as CEF
# Use CEF connector in Sentinel
```

---

## Best Practices

### 1. Polling for Status
```javascript
async function waitForExport(jobId) {
  let attempts = 0;
  const maxAttempts = 60; // 5 minutes
  
  while (attempts < maxAttempts) {
    const status = await checkExportStatus(jobId);
    
    if (status === 'completed') {
      return true;
    }
    if (status === 'failed') {
      throw new Error('Export failed');
    }
    
    await sleep(5000); // Poll every 5 seconds
    attempts++;
  }
  
  throw new Error('Export timeout');
}
```

### 2. Error Handling
```javascript
try {
  const job = await createExport(params);
  await waitForExport(job.jobId);
  await downloadExport(job.jobId);
} catch (error) {
  if (error.status === 400) {
    console.error('Invalid request:', error.message);
  } else if (error.status === 404) {
    console.error('Job not found');
  } else {
    console.error('Export failed:', error);
  }
}
```

### 3. Cleanup
```javascript
// Always delete exports after download
try {
  await downloadExport(jobId);
} finally {
  await deleteExport(jobId);
}
```

---

## Performance

### Expected Times
- **100 events**: < 1 second
- **1,000 events**: 1-2 seconds
- **10,000 events**: 3-5 seconds

### File Sizes (Compressed)
- **JSON**: 50-100 KB per 1,000 events
- **CSV**: 30-60 KB per 1,000 events
- **Syslog**: 20-40 KB per 1,000 events
- **CEF**: 25-50 KB per 1,000 events

---

**Version**: 1.0  
**Last Updated**: November 26, 2025  
**Status**: Production Ready
