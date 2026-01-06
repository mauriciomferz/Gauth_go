# AgentAuth+ Security Documentation

Security best practices, vulnerability status, and incident response guide.

---

## Dependabot Security Status ✅

**Last Checked**: December 29, 2025

### Alert #1: quic-go (CVE-2025-59530) - FIXED ✅
- **Status**: Fixed on October 24, 2025
- **Severity**: High (CVSS 7.5)
- **Issue**: Denial-of-service vulnerability
- **Fix**: Upgraded to patched version
- **Impact**: Client-side DoS protection

### Alert #2: esbuild - FIXED ✅
- **Status**: Fixed on November 16, 2025
- **Severity**: Medium (CVSS 5.3)
- **Issue**: Origin validation error
- **Fix**: Upgraded to version 0.25.0+
- **Impact**: Build tool security

**Current Status**: All known vulnerabilities resolved ✅

---

## Security Best Practices

### JWT Configuration

**Current Setup**:
- JWT signing key: Randomly generated (256-bit)
- Storage: Environment variable `AGENTAUTH_JWT_SIGNING_KEY`
- Algorithm: EdDSA (configurable)

**Recommendations**:
1. Keep JWT key in secure secret manager:
   ```bash
   # Never commit to git
   echo "AGENTAUTH_JWT_SIGNING_KEY=..." >> .gitignore
   
   # Use environment-specific keys
   # Dev: Different from staging/prod
   ```

2. Rotate keys every 90 days:
   ```bash
   # Generate new key
   NEW_KEY=$(openssl rand -hex 32)
   
   # Update environment
   # Restart backend with new key
   ```

3. Monitor key usage in logs

### Database Security

**Current Setup**:
- PostgreSQL with password authentication
- Connection limited to Docker network
- SSL mode: disabled (development)

**Production Recommendations**:
```bash
# Enable SSL
DB_SSLMODE=require

# Use strong passwords
DB_PASSWORD=$(openssl rand -base64 32)

# Limit connections
# PostgreSQL: max_connections = 100
# Connection pool: max 20

# Regular backups
pg_dump -U agentauth agentauth > backup-$(date +%Y%m%d).sql
```

### API Key Security

**Best Practices**:
1. **Never log secret keys**:
   ```go
   // Good: Log key ID only
   log.Info("API key created", "keyId", key.ID)
   
   // Bad: Don't log secret
   // log.Info("Secret", "secret", key.Secret)
   ```

2. **Use scoped permissions**:
   ```bash
   # Minimum required scopes
   curl -X POST /api/admin/api-keys -d '{
     "scopes": ["poa:read"],  # Read-only
     ...
   }'
   ```

3. **Set rate limits**:
   ```bash
   # Reasonable limits
   "rateLimitPerMinute": 60,
   "rateLimitPerHour": 1000
   ```

4. **Monitor usage**:
   ```bash
   # Regular audits
   curl /api/admin/api-keys/KEY_ID/usage | jq .
   ```

### Network Security

**Current Setup**:
- Backend: port 8080
- PostgreSQL: port 5432 (Docker network only)
- Redis: port 6379 (Docker network only)

**Production Recommendations**:
```bash
# Use firewall rules
ufw allow 443/tcp  # HTTPS only
ufw allow 8080/tcp from 10.0.0.0/8  # Internal only

# Enable TLS
AGENTAUTH_TLS_ENABLED=true
AGENTAUTH_TLS_CERT_PATH=/etc/agentauth/tls/cert.pem
AGENTAUTH_TLS_KEY_PATH=/etc/agentauth/tls/key.pem
```

---

## Incident Response Guide

### Security Incident Detected

**Immediate Steps**:

1. **Assess Severity**:
   ```bash
   # Check recent critical events
   curl "http://localhost:8080/api/admin/audit/events?severity=critical&limit=50" | jq .
   ```

2. **Isolate if necessary**:
   ```bash
   # Revoke compromised API key
   curl -X POST /api/admin/api-keys/COMPROMISED_ID/revoke
   
   # Block IP if needed (firewall level)
   ufw deny from ATTACKER_IP
   ```

3. **Export audit logs**:
   ```bash
   # Full export for investigation
   curl -X POST /api/admin/audit/export \
     -d '{"format":"json","dateRange":"last-24h"}' | jq -r '.jobId'
   
   # Download immediately
   curl /api/admin/audit/export/JOB_ID/download > incident-$(date +%Y%m%d).json
   ```

### Data Breach Response

1. **Stop the breach**:
   - Revoke all API keys
   - Change JWT signing key
   - Restart backend

2. **Assess impact**:
   ```bash
   # Check affected users
   curl "/api/admin/audit/events?action=poa_created&limit=1000" | jq .
   
   # Check data accessed
   curl "/api/admin/audit/events?category=data_access" | jq .
   ```

3. **Notification**:
   - Document timeline
   - Notify affected users
   - Report to authorities if required

### Suspicious Activity

**Indicators**:
- Multiple failed auth attempts
- Unusual request patterns
- High rate limit violations
- Access from unknown IPs

**Investigation**:
```bash
# Check user activity
curl "/api/admin/audit/events?actor=SUSPICIOUS_USER@example.com" | jq .

# Check correlations
curl /api/admin/audit/correlations | jq .

# Export for analysis
curl -X POST /api/admin/audit/export \
  -d '{"format":"csv","dateRange":"last-7d","actor":"SUSPICIOUS_USER"}' | jq .
```

---

## Security Checklist

### Daily
- [ ] Check critical/high severity events
- [ ] Monitor API key usage
- [ ] Review failed authentication attempts

### Weekly
- [ ] Export audit logs for compliance
- [ ] Review API key expiration dates
- [ ] Check for unusual patterns

### Monthly
- [ ] Rotate JWT signing keys
- [ ] Review and revoke unused API keys
- [ ] Update dependencies (`go get -u ./...`)
- [ ] Security scan (`gosec ./...`)

### Quarterly
- [ ] Full security audit
- [ ] Penetration testing
- [ ] Disaster recovery drill
- [ ] Update incident response plan

---

## Dependency Management

### Checking for Updates

```bash
# List outdated packages
go list -m -u all | grep '\['

# Update all dependencies
go get -u ./...
go mod tidy

# Security scan
gosec ./...
```

### Update Process

1. **Test in development**:
   ```bash
   go test ./...
   ```

2. **Deploy to staging**:
   ```bash
   ./deploy-standalone.sh
   ```

3. **Verify functionality**:
   ```bash
   curl http://localhost:8080/api/v1/beta/health
   ```

4. **Deploy to production** (if successful)

---

## Audit Log Security

**Tamper Protection**:
- Events include hash chain
- Previous event hash stored
- Blockchain anchoring (optional)

**Retention**:
```bash
# Export old logs before cleanup
curl -X POST /api/admin/audit/export \
  -d '{"format":"json","dateRange":"2024-01-01,2024-12-31","compressed":true}'

# Store securely offsite
```

**Access Control**:
- Admin endpoints require authentication
- Audit log viewing logged itself
- No modification of historical events allowed

---

## Compliance

### GDPR
- Export user data on request
- Delete user data on request (right to erasure)
- Maintain audit trail of data access
- Encrypt data at rest and in transit

### SOC 2
- Comprehensive audit logging
- Access controls implemented
- Regular security reviews
- Incident response procedures

---

## Emergency Contacts

### Security Team
- Primary: security@company.com
- On-call: +1-XXX-XXX-XXXX

### Escalation
1. DevOps lead
2. Security officer
3. CTO
4. Legal (for data breaches)

---

## Security Resources

- **OWASP Top 10**: https://owasp.org/Top10/
- **Go Security**: https://go.dev/security/
- **PostgreSQL Security**: https://www.postgresql.org/docs/current/security.html
- **Docker Security**: https://docs.docker.com/engine/security/

**Last Security Audit**: December 29, 2025  
**Next Scheduled Audit**: March 29, 2026
