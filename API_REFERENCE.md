# GAuth API Reference

**Version**: 1.0.0  
**Base URL**: `http://localhost:8080`  
**Last Updated**: November 29, 2025

## Table of Contents

- [Authentication](#authentication)
- [Model Context Protocol (MCP)](#model-context-protocol-mcp)
- [Admin APIs](#admin-apis)
- [Revocation APIs](#revocation-apis)
- [Authorization APIs](#authorization-apis)
- [Audit APIs](#audit-apis)
- [Response Formats](#response-formats)

---

## Authentication

### Frontend Authentication Flow

#### 1. Initiate Login
```http
POST /api/v1/gauth/auth/login/init
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "userId": "user-admin",
  "requiresMFA": true,
  "mfaMethods": ["totp", "sms", "email"],
  "sessionChallenge": "challenge-admin-20251129120000"
}
```

**Response (401 Unauthorized)**:
```json
{
  "success": false,
  "error": "Invalid username or password"
}
```

#### 2. Verify MFA
```http
POST /api/v1/gauth/auth/login/mfa
Content-Type: application/json

{
  "challengeId": "challenge-admin-20251129120000",
  "code": "123456",
  "method": "totp"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": "2025-11-29T13:00:00Z",
  "tokenType": "Bearer"
}
```

**Token Details**:
- **Access Token**: Valid for 1 hour
- **Refresh Token**: Valid for 7 days
- **Usage**: Include in `Authorization: Bearer <token>` header

#### 3. Refresh Token
```http
POST /api/v1/gauth/auth/token/refresh
Content-Type: application/json

{
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": "2025-11-29T14:00:00Z"
}
```

#### 4. Logout
```http
POST /api/v1/gauth/auth/logout
Content-Type: application/json

{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Logout successful"
}
```

---

## Model Context Protocol (MCP)

### Health Check
```http
GET /api/v1/gauth/mcp/health
```

**Response (200 OK)**:
```json
{
  "success": true,
  "status": "healthy",
  "servers": {
    "total": 3,
    "connected": 2,
    "disconnected": 1
  },
  "timestamp": "2025-11-29T12:00:00Z"
}
```

### List All Servers
```http
GET /api/v1/gauth/mcp/servers
```

**Response (200 OK)**:
```json
{
  "success": true,
  "servers": [
    {
      "id": "filesystem-server",
      "name": "Filesystem MCP Server",
      "description": "Access local filesystem resources",
      "transport_type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/Users/data"],
      "url": "",
      "status": "connected"
    },
    {
      "id": "github-server",
      "name": "GitHub MCP Server",
      "description": "Access GitHub repositories",
      "transport_type": "https",
      "command": "",
      "args": [],
      "url": "https://api.github.com/mcp",
      "status": "connected"
    }
  ]
}
```

### Register New Server
```http
POST /api/v1/gauth/mcp/servers
Content-Type: application/json

{
  "id": "custom-server",
  "name": "Custom MCP Server",
  "description": "Custom server implementation",
  "transport_type": "stdio",
  "command": "node",
  "args": ["./server.js"],
  "require_auth": false,
  "allowed_scopes": ["read", "write"]
}
```

**Transport Types**:
- `stdio`: Standard input/output (requires `command`)
- `http`: HTTP protocol (requires `url`)
- `https`: HTTPS protocol (requires `url`)

**Response (201 Created)**:
```json
{
  "success": true,
  "server": {
    "id": "custom-server",
    "name": "Custom MCP Server",
    "status": "connected"
  }
}
```

**Response (400 Bad Request)**:
```json
{
  "success": false,
  "error": "invalid_transport_type",
  "message": "Transport type must be 'stdio', 'http', or 'https'"
}
```

### Get Server Status
```http
GET /api/v1/gauth/mcp/servers/:id/status
```

**Response (200 OK)**:
```json
{
  "success": true,
  "server": {
    "id": "filesystem-server",
    "name": "Filesystem MCP Server",
    "description": "Access local filesystem resources",
    "transport_type": "stdio",
    "status": "connected"
  }
}
```

### Disconnect Server
```http
DELETE /api/v1/gauth/mcp/servers/:id
```

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Server disconnected successfully",
  "server_id": "filesystem-server"
}
```

**Response (504 Gateway Timeout)**:
```json
{
  "success": false,
  "error": "timeout",
  "message": "Server disconnection timed out"
}
```

### List Server Resources
```http
GET /api/v1/gauth/mcp/servers/:id/resources
```

**Response (200 OK)**:
```json
{
  "success": true,
  "resources": [
    {
      "uri": "file:///Users/data/config.json",
      "name": "config.json",
      "mimeType": "application/json",
      "description": "Application configuration"
    },
    {
      "uri": "file:///Users/data/logs/app.log",
      "name": "app.log",
      "mimeType": "text/plain",
      "description": "Application logs"
    }
  ]
}
```

### Read Resource Content
```http
POST /api/v1/gauth/mcp/servers/:id/resources/read
Content-Type: application/json

{
  "uri": "file:///Users/data/config.json"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "contents": [
    {
      "uri": "file:///Users/data/config.json",
      "mimeType": "application/json",
      "text": "{\n  \"app\": \"gauth\",\n  \"version\": \"1.0.0\"\n}"
    }
  ]
}
```

### List Server Tools
```http
GET /api/v1/gauth/mcp/servers/:id/tools
```

**Response (200 OK)**:
```json
{
  "success": true,
  "tools": [
    {
      "name": "read_file",
      "description": "Read contents of a file",
      "inputSchema": {
        "type": "object",
        "properties": {
          "path": {
            "type": "string",
            "description": "File path to read"
          }
        },
        "required": ["path"]
      }
    },
    {
      "name": "write_file",
      "description": "Write content to a file",
      "inputSchema": {
        "type": "object",
        "properties": {
          "path": {
            "type": "string",
            "description": "File path to write"
          },
          "content": {
            "type": "string",
            "description": "Content to write"
          }
        },
        "required": ["path", "content"]
      }
    }
  ]
}
```

### Call Tool
```http
POST /api/v1/gauth/mcp/servers/:id/tools/call
Content-Type: application/json

{
  "name": "read_file",
  "arguments": {
    "path": "/Users/data/config.json"
  }
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "content": [
    {
      "type": "text",
      "text": "{\n  \"app\": \"gauth\",\n  \"version\": \"1.0.0\"\n}"
    }
  ]
}
```

**Response (500 Internal Server Error)**:
```json
{
  "success": false,
  "error": "tool_call_failed",
  "message": "File not found: /Users/data/config.json"
}
```

---

## Admin APIs

### Admin Login
```http
POST /api/admin/auth/login
Content-Type: application/json

{
  "email": "admin@example.com",
  "password": "password"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "admin-1",
    "email": "admin@example.com",
    "role": "admin",
    "name": "Admin User"
  },
  "expiresAt": "2025-11-30T12:00:00Z"
}
```

---

## Revocation APIs

### Initiate Revocation
```http
POST /api/v1/poa/:id/revocation/initiate
Content-Type: application/json

{
  "initiator": "alice",
  "reason": "security_risk"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "revocation_id": "rev-123456",
  "status": "pending",
  "initiated_at": "2025-11-29T12:00:00Z"
}
```

### Approve Revocation
```http
POST /api/v1/poa/:id/revocation/approve
Content-Type: application/json

{
  "approver": "controller1"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "revocation_id": "rev-123456",
  "status": "approved",
  "approved_at": "2025-11-29T12:05:00Z"
}
```

### Cancel Revocation
```http
POST /api/v1/poa/:id/revocation/cancel
Content-Type: application/json

{
  "actor": "alice"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "revocation_id": "rev-123456",
  "status": "cancelled",
  "cancelled_at": "2025-11-29T12:10:00Z"
}
```

---

## Authorization APIs

### Authorize POA
```http
POST /api/v1/poa/authorize
Content-Type: application/json

{
  "grantor": "alice",
  "grantee": "bob",
  "scope": "read:documents",
  "duration": 3600
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "poa_id": "poa-789012",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-11-29T13:00:00Z"
}
```

### Get POA Metrics
```http
GET /api/v1/poa/metrics
```

**Response (200 OK)**:
```json
{
  "total_active": 156,
  "total_revoked": 23,
  "total_expired": 45,
  "avg_duration_hours": 24,
  "most_common_scope": "read:documents"
}
```

---

## Audit APIs

### List Audit Logs
```http
GET /api/v1/audit/logs?limit=100&offset=0
```

**Response (200 OK)**:
```json
{
  "success": true,
  "logs": [
    {
      "id": "audit-001",
      "timestamp": "2025-11-29T12:00:00Z",
      "action": "poa.authorize",
      "actor": "alice",
      "resource": "poa-789012",
      "status": "success"
    }
  ],
  "total": 1543,
  "limit": 100,
  "offset": 0
}
```

### Record Audit Event
```http
POST /api/v1/audit/record
Content-Type: application/json

{
  "action": "login.success",
  "actor": "admin",
  "resource": "admin-portal",
  "metadata": {
    "ip": "192.168.1.100",
    "user_agent": "Mozilla/5.0"
  }
}
```

**Response (201 Created)**:
```json
{
  "success": true,
  "audit_id": "audit-002",
  "timestamp": "2025-11-29T12:01:00Z"
}
```

---

## Response Formats

### Success Response
```json
{
  "success": true,
  "data": { },
  "message": "Operation completed successfully"
}
```

### Error Response
```json
{
  "success": false,
  "error": "error_code",
  "message": "Human-readable error message",
  "details": {
    "field": "Additional error details"
  }
}
```

### Common HTTP Status Codes

| Code | Meaning | Usage |
|------|---------|-------|
| 200 | OK | Successful GET, PUT, or DELETE |
| 201 | Created | Successful POST that creates a resource |
| 400 | Bad Request | Invalid request format or parameters |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Valid auth but insufficient permissions |
| 404 | Not Found | Resource does not exist |
| 409 | Conflict | Resource already exists or state conflict |
| 500 | Internal Server Error | Server-side error |
| 504 | Gateway Timeout | Upstream service timeout |

### Common Error Codes

| Error Code | Description |
|------------|-------------|
| `invalid_request` | Request validation failed |
| `invalid_credentials` | Username or password incorrect |
| `invalid_token` | JWT token invalid or expired |
| `server_not_found` | MCP server not registered |
| `timeout` | Operation exceeded time limit |
| `registration_failed` | MCP server registration failed |
| `disconnect_failed` | Server disconnection failed |
| `tool_call_failed` | MCP tool execution failed |
| `missing_command` | Required command not provided |
| `missing_url` | Required URL not provided |
| `invalid_transport_type` | Transport type not supported |

---

## Rate Limiting

All API endpoints are subject to rate limiting:

- **Default**: 100 requests per minute per IP
- **Authenticated**: 1000 requests per minute per user
- **MCP Tool Calls**: 50 requests per minute per server

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 998
X-RateLimit-Reset: 1732881600
```

---

## Pagination

List endpoints support pagination:

**Query Parameters**:
- `limit`: Number of results per page (default: 50, max: 1000)
- `offset`: Number of results to skip (default: 0)

**Example**:
```http
GET /api/v1/audit/logs?limit=100&offset=200
```

---

## Authentication Headers

Include JWT token in requests:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

Optional tenant header for multi-tenant deployments:
```http
X-Tenant-ID: tenant-123
```

---

## Webhooks

Configure webhooks for event notifications:

### Supported Events
- `auth.login.success`
- `auth.login.failed`
- `poa.created`
- `poa.revoked`
- `audit.critical`

### Webhook Payload
```json
{
  "event": "poa.revoked",
  "timestamp": "2025-11-29T12:00:00Z",
  "data": {
    "poa_id": "poa-789012",
    "reason": "security_risk"
  }
}
```

---

## Development Credentials

**Frontend Auth**:
- Username: `admin`
- Password: `password`
- MFA Code: `123456`

**Admin Portal**:
- Email: `admin@example.com`
- Password: `password`

⚠️ **Warning**: Change these credentials in production!

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/mauriciomferz/Gauth_go/issues
- Documentation: https://github.com/mauriciomferz/Gauth_go/wiki

---

**Last Updated**: November 29, 2025  
**API Version**: 1.0.0
