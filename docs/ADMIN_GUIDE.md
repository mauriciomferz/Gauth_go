# GAuth+ Admin User Guide

Quick reference guide for GAuth+ administrators.

---

## Getting Started

### Prerequisites
- Backend running on http://localhost:8080
- PostgreSQL database connected
- Admin credentials (if authentication required)

### First Steps
1. Verify backend health:
   ```bash
   curl http://localhost:8080/api/v1/beta/health
   ```

2. Check you have admin access:
   ```bash
   curl http://localhost:8080/api/admin/audit/metrics
   ```

---

## Common Operations

### Exporting Audit Logs

**Use Case**: Monthly compliance reporting

```bash
# 1. Create export for last 30 days
curl -X POST http://localhost:8080/api/admin/audit/export \
  -H "Content-Type: application/json" \
  -d '{
    "format": "csv",
    "dateRange": "last-30d",
    "compressed": true
  }' | jq -r '.jobId'

# Save the job ID
JOB_ID="<from-above>"

# 2. Wait a minute, then check status
curl http://localhost:8080/api/admin/audit/export/$JOB_ID | jq '.status'

# 3. When status is "completed", download
curl -o compliance-report-$(date +%Y%m).csv.gz \
  http://localhost:8080/api/admin/audit/export/$JOB_ID/download
```

### Managing API Keys

**Use Case**: Creating keys for service accounts

```bash
# Create API key for a service
curl -X POST http://localhost:8080/api/admin/api-keys \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "production",
    "keyName": "Payment Service",
    "description": "Production payment processing service",
    "scopes": ["poa:read", "poa:write"],
    "rateLimitPerMinute": 200,
    "rateLimitPerHour": 10000,
    "createdBy": "admin@company.com"
  }'

# IMPORTANT: Save the secretKey from response immediately!
```

### Monitoring System Health

**Daily monitoring routine**:

```bash
# 1. Check health
curl http://localhost:8080/api/v1/beta/health | jq .

# 2. Check audit metrics
curl http://localhost:8080/api/admin/audit/metrics | jq .

# 3. Check for critical events (last hour)
curl "http://localhost:8080/api/admin/audit/events?severity=critical&limit=10" | jq .
```

### Investigating Security Incidents

**Use Case**: Reviewing suspicious activity

```bash
# 1. Search for events by user
curl "http://localhost:8080/api/admin/audit/events?actor=suspicious@user.com&limit=50" | jq .

# 2. Filter by time and severity
curl "http://localhost:8080/api/admin/audit/events?severity=high&category=security" | jq .

# 3. Check event correlations
curl http://localhost:8080/api/admin/audit/correlations | jq .
```

---

## Troubleshooting

### Export Job Stuck in "Pending"

**Symptoms**: Export job status remains "pending" for > 5 minutes

**Solutions**:
1. Check backend logs:
   ```bash
   docker logs gauth-backend-new --tail=50 | grep -i export
   ```

2. Verify database connectivity:
   ```bash
   docker exec gauth-postgres pg_isready
   ```

3. Check disk space for export directory:
   ```bash
   df -h /tmp/gauth-audit-exports
   ```

### API Key Not Working

**Symptoms**: Getting 401/403 errors with API key

**Solutions**:
1. Verify key hasn't expired:
   ```bash
   curl http://localhost:8080/api/admin/api-keys/<key-id> | jq '.expiresAt'
   ```

2. Check key hasn't been revoked:
   ```bash
   curl http://localhost:8080/api/admin/api-keys/<key-id> | jq '.status'
   ```

3. Verify rate limits aren't exceeded:
   ```bash
   curl http://localhost:8080/api/admin/api-keys/<key-id>/usage | jq .
   ```

### High Memory Usage

**Symptoms**: Backend using > 2GB RAM

**Solutions**:
1. Check for large export jobs:
   ```bash
   ls -lh /tmp/gauth-audit-exports/
   ```

2. Clean up old export jobs:
   ```bash
   # Old jobs auto-delete after 24h, or manually:
   curl -X DELETE http://localhost:8080/api/admin/audit/export/<old-job-id>
   ```

3. Restart backend if needed:
   ```bash
   docker restart gauth-backend-new
   ```

---

## Best Practices

### Audit Log Management
- Export logs monthly for compliance
- Keep exports for required retention period
- Monitor siem_integrations for failures
- Review high/critical severity events daily

### API Key Security
- Rotate keys every 90 days
- Use minimum required scopes
- Set appropriate rate limits
- Monitor usage regularly
- Revoke unused keys immediately

### Performance Tuning
- Limit audit event queries to < 1000 results
- Use compressed exports for large datasets
- Clean up completed export jobs
- Monitor database connection pool

---

## Quick Reference

### Useful Commands

```bash
# Health check
curl http://localhost:8080/api/v1/beta/health | jq .

# Recent events
curl "http://localhost:8080/api/admin/audit/events?limit=10" | jq .

# System metrics
curl http://localhost:8080/api/admin/audit/metrics | jq .

# Create export
curl -X POST http://localhost:8080/api/admin/audit/export \
  -H "Content-Type: application/json" \
  -d '{"format":"json","dateRange":"last-24h"}' | jq .

# List API keys
curl http://localhost:8080/api/admin/api-keys | jq .
```

### Log Locations

- **Backend logs**: `docker logs gauth-backend-new`
- **PostgreSQL logs**: `docker logs gauth-postgres`
- **Export files**: `/tmp/gauth-audit-exports/`

### Important URLs

- Backend: http://localhost:8080
- Health: http://localhost:8080/api/v1/beta/health
- Swagger: http://localhost:8080/api/docs/swagger
- Prometheus (if configured): http://localhost:9090

---

## Emergency Procedures

### Backend Crash

```bash
# 1. Check logs
docker logs gauth-backend-new --tail=100

# 2. Restart backend
docker restart gauth-backend-new

# 3. Verify health
curl http://localhost:8080/api/v1/beta/health

# 4. If still failing, redeploy
cd /path/to/Gauth_go
./deploy-standalone.sh
```

### Database Connection Loss

```bash
# 1. Check PostgreSQL
docker ps | grep gauth-postgres

# 2. Restart if needed
docker restart gauth-postgres

# 3. Wait for healthy
docker exec gauth-postgres pg_isready

# 4. Restart backend
docker restart gauth-backend-new
```

---

## Support

For issues not covered in this guide:
1. Check deployment guide: `DEPLOYMENT_GUIDE.md`
2. Review API examples: `docs/API_EXAMPLES.md`
3. Check backend logs for errors
4. Review GitHub issues

**Current Version**: 1.0.0-beta  
**Last Updated**: December 29, 2025
