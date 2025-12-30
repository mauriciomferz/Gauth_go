---
title: Gnap Deployment
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GNAP Production Deployment Guide

Deploy GNAP (RFC 9635) in production with this guide.

## Prerequisites

- AgentAuth backend running
- PostgreSQL (for persistent stores - optional, in-memory by default)
- HTTPS/TLS termination (required for security)

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GAUTH_GNAP_BASE_URL` | `http://localhost:8080` | Base URL for continuation URIs |
| `GAUTH_JWT_SIGNING_KEY` | - | JWT signing key (required) |

## Configuration

### 1. Set Base URL

```bash
export GAUTH_GNAP_BASE_URL=https://your-domain.com
```

### 2. Enable GNAP Endpoints

GNAP endpoints are registered automatically when the server starts.

### 3. Configure Token Expiration (Optional)

Default: 1 hour. Customize via handler initialization in `server_factory.go`.

## Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/.well-known/gnap-as-rs` | GET | Discovery metadata |
| `/gnap/tx` | POST | Grant request |
| `/gnap/continue/:id` | POST/PATCH/DELETE | Continuation |
| `/gnap/token/:id` | POST/DELETE | Token management |

## Security Checklist

- [ ] Use HTTPS for all GNAP endpoints
- [ ] Configure `GAUTH_GNAP_BASE_URL` with HTTPS
- [ ] Enable HTTP Message Signatures for client authentication
- [ ] Set up audit logging for compliance
- [ ] Configure rate limiting

## Client Configuration

Clients should:

1. **Discover endpoints**: `GET /.well-known/gnap-as-rs`
2. **Request grants**: `POST /gnap/tx` with access requirements
3. **Use tokens**: `Authorization: GNAP <token>`

### Example Client Request

```bash
curl -X POST https://your-domain.com/gnap/tx \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": {
      "access": [{"type": "api", "actions": ["read", "write"]}]
    },
    "client": {
      "display": {"name": "My App"}
    }
  }'
```

## Monitoring

GNAP audit events:
- `gnap_grant_request`
- `gnap_grant_approved`
- `gnap_grant_denied`
- `gnap_token_issue`
- `gnap_token_rotate`
- `gnap_token_revoke`

Query via audit API or logs.

## Persistent Storage (Production)

Replace in-memory stores with database-backed implementations:

```go
// In server_factory.go
grantStore := gnap.NewPostgresGrantStore(db)  // TODO: Implement
tokenStore := gnap.NewPostgresTokenStore(db)  // TODO: Implement
```

## High Availability

For HA deployments:
- Use shared grant/token stores (Redis or PostgreSQL)
- Ensure continuation tokens work across instances
- Use sticky sessions or shared state

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Token rejected | Check expiration, verify signature |
| Continuation fails | Verify continue token matches grant |
| Discovery returns wrong URL | Check `GAUTH_GNAP_BASE_URL` |
