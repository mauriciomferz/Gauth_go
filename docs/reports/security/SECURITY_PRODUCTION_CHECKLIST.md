---
title: Security Production Checklist
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GAuth Server - Production Security Checklist

**CRITICAL**: This checklist MUST be completed before deploying to production. The GAuth server includes development shortcuts that create severe security vulnerabilities if deployed as-is.

## ⚠️ Critical Security Vulnerabilities (MUST FIX)

### 1. JWT Signing Key Forgery Risk 🔴 CRITICAL

**Vulnerability**: The default `GAUTH_JWT_SIGNING_KEY` is set to `dev-please-change` in examples and documentation.

**Impact**: If deployed with this default key, attackers can:
- Forge any "Proof of Authorization" (PoA) token
- Bypass all Policy Decision Point (PDP) checks  
- Impersonate any user or service
- Grant themselves arbitrary permissions

**Remediation**:
```bash
# Generate a strong random key (minimum 32 bytes)
export GAUTH_JWT_SIGNING_KEY=$(openssl rand -base64 32)

# Or use a Hardware Security Module (HSM) / Cloud KMS
# AWS KMS, Azure Key Vault, GCP Cloud KMS, etc.
```

**Validation**: Server now validates on startup and **BLOCKS startup** if:
- `GAUTH_JWT_SIGNING_KEY` is unset
- Key matches known weak values: `dev-please-change`, `dev-signing-key-change-in-production`, `changeme`, `secret`, `test`
- Key is shorter than 32 bytes in production mode

---

### 2. MCP Protocol SSRF Attack Vector 🔴 HIGH

**Vulnerability**: The Model Context Protocol (MCP) allows AI Agents to "read resources" via URIs. Without scheme validation, an attacker-controlled Agent could request:
- `file:///etc/passwd` - Local file disclosure
- `http://169.254.169.254/latest/meta-data/` - AWS credentials theft
- `http://localhost:6379/` - Redis access
- `http://192.168.1.1/admin` - Internal network scanning

**Impact**: 
- Steal cloud provider credentials (AWS/Azure/GCP metadata)
- Read sensitive local files
- Access internal services not exposed to internet
- Pivot to internal network

**Remediation**: MCP client now includes automatic URI validation:

```go
// Automatically applied in pkg/mcp/client.go ReadResource()
validator := security.NewURIValidatorWithSchemes([]string{"mcp", "https"})
if err := validator.ValidateURI(uri); err != nil {
    return nil, fmt.Errorf("URI validation failed (SSRF protection): %w", err)
}
```

**Blocked**:
- `file://` scheme - local file access
- `localhost`, `127.0.0.1`, `::1` - loopback
- `169.254.169.254` - AWS/Azure/GCP metadata
- `metadata.google.internal` - GCP metadata  
- Private IP ranges: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`

**Allowed**:
- `https://` - External HTTPS resources
- `mcp://` - MCP protocol resources

---

### 3. Mock Identity Verification 🟡 MEDIUM

**Vulnerability**: The `/api/v1/beta/pvp/verify` endpoint uses a mock PowerVerificationPoint (PVP) that accepts any identity proof without real verification.

**Impact**:
- Anyone can claim any identity
- No real document validation  
- Compliance violations (KYC/AML requirements)

**Remediation**: Configure real identity verification provider:

```bash
# Option 1: Stripe Identity
export GAUTH_PVP_PROVIDER=stripe
export STRIPE_API_KEY=sk_live_...

# Option 2: Veriff
export GAUTH_PVP_PROVIDER=veriff  
export VERIFF_API_KEY=...
export VERIFF_API_SECRET=...

# Option 3: Idemia
export GAUTH_PVP_PROVIDER=idemia
export IDEMIA_API_KEY=...

# Option 4: Onfido
export GAUTH_PVP_PROVIDER=onfido
export ONFIDO_API_TOKEN=...

# Option 5: Jumio
export GAUTH_PVP_PROVIDER=jumio
export JUMIO_API_TOKEN=...
export JUMIO_API_SECRET=...
```

**Validation**: Server now warns on startup if `GAUTH_PVP_PROVIDER` is unset or set to `mock` in production mode.

**Implementation Status**:
- ✅ Framework and validation complete
- ⚠️ Provider integrations require implementation (see `pkg/gauth/pvp_factory.go`)

---

### 4. Debug UI Exposure 🟢 LOW  

**Vulnerability**: `GAUTH_DEV_INDEX=1` exposes developer UI and debug endpoints.

**Impact**: Information disclosure, potential admin interface access.

**Remediation**:
```bash
# Disable debug features
unset GAUTH_DEV_INDEX
unset GAUTH_DEV_MODE

# Or explicitly set to 0
export GAUTH_DEV_INDEX=0
export GAUTH_DEV_MODE=false
```

**Validation**: Server now **BLOCKS startup** if these are enabled in production mode.

---

## Production Environment Variables Checklist

### Required (Server blocks startup if missing/weak)

- [ ] `GAUTH_JWT_SIGNING_KEY` - Strong random key (min 32 bytes) - **CRITICAL**
- [ ] `GAUTH_ENV=production` or `GAUTH_MODE=production` - Enables production mode

### Required for Compliance

- [ ] `GAUTH_PVP_PROVIDER` - Real identity verification (stripe, veriff, idemia, onfido, jumio)
- [ ] Provider-specific credentials (see section 3 above)

### Security Configuration  

- [ ] `GAUTH_CORS_ALLOW` - Restrict to specific domains (NOT `*`)
- [ ] `GAUTH_RATE_LIMIT_ENABLED=true` - Enable rate limiting
- [ ] `GAUTH_RATE_LIMIT_REQUESTS=100` - Requests per window
- [ ] `GAUTH_RATE_LIMIT_WINDOW=60s` - Time window

### Database Security

- [ ] `GAUTH_DB_PASSWORD` - Strong password (NOT `dev-password-please-change`)
- [ ] `GAUTH_DB_HOST` - Database hostname
- [ ] `GAUTH_DB_PORT` - Database port (default 5432)
- [ ] `GAUTH_DB_USER` - Database user
- [ ] `GAUTH_DB_NAME` - Database name

### Disable Development Features

- [ ] `GAUTH_DEV_INDEX=0` or unset
- [ ] `GAUTH_DEV_MODE=false` or unset
- [ ] `GAUTH_ENABLE_PPROF=0` or unset - Disable profiling endpoints

### Monitoring (Recommended)

- [ ] `GAUTH_METRICS_ENABLED=true`
- [ ] `GAUTH_TRACING_ENABLED=true`
- [ ] `GAUTH_LOG_LEVEL=info` (NOT `debug`)

---

## Automated Validation

The server now performs automatic security validation on startup:

```go
// In cmd/web-server/main.go
productionMode := security.ProductionModeDetector()
validator := security.NewStartupValidator(productionMode)

if err := validator.ValidateAll(); err != nil {
    log.Fatalf("[SECURITY] FATAL: %v\n\nSERVER STARTUP BLOCKED.", err)
}
```

**Production Mode Detection**: Triggered by:
- `GAUTH_ENV=production` or `GAUTH_ENV=prod`
- `GAUTH_MODE=production` or `GAUTH_MODE=prod`  
- Port binding to 443 or 8443 without dev flags

**Validation Failures Block Startup**: Server will refuse to start if:
- JWT signing key is missing or weak
- `GAUTH_DEV_INDEX=1` in production
- `GAUTH_DEV_MODE=true` in production
- `GAUTH_CORS_ALLOW=*` in production
- Database password matches known weak values

**Warnings Logged**: Non-blocking warnings for:
- Short JWT key (<32 bytes) in production
- Mock PVP provider in production
- Rate limiting disabled in production
- CORS allowing localhost in production

---

## Testing Security Remediations

Run the security test suite:

```bash
# Run all security tests
go test ./internal/security/... -v

# Run SSRF protection tests
go test ./internal/security/ -run TestSSRFPrevention -v

# Run token forgery tests  
go test ./internal/security/ -run TestForgedTokenRejection -v

# Run production mode tests
go test ./internal/security/ -run TestDebugEndpointsBlocked -v
```

Expected output:
```
=== RUN   TestForgedTokenRejection
=== RUN   TestSSRFPrevention
=== RUN   TestIdentityVerificationEnforcement
=== RUN   TestDebugEndpointsBlocked
--- PASS: TestForgedTokenRejection (0.00s)
--- PASS: TestSSRFPrevention (0.01s)
--- PASS: TestIdentityVerificationEnforcement (0.00s)
--- PASS: TestDebugEndpointsBlocked (0.00s)
PASS
```

---

## Deployment Workflow

### 1. Pre-Deployment

```bash
# Review this checklist
cat SECURITY_PRODUCTION_CHECKLIST.md

# Set production environment variables
export GAUTH_ENV=production
export GAUTH_JWT_SIGNING_KEY=$(openssl rand -base64 32)
export GAUTH_PVP_PROVIDER=stripe
export STRIPE_API_KEY=sk_live_...
# ... (see checklist above)

# Run security tests
go test ./internal/security/... -v

# Build production binary
go build -o gauth-server ./cmd/web-server
```

### 2. Startup (Automatic Validation)

```bash
./gauth-server
```

Expected output:
```
[SECURITY] Production mode detected - enforcing security validations
[SECURITY] All security validations passed ✓
[Server] Starting GAuth Server on :8080
```

If validation fails:
```
[SECURITY] FATAL: security validation failed:
GAUTH_JWT_SIGNING_KEY is set to known weak value 'dev-please-change'
GAUTH_DEV_INDEX=1 exposes debug UI - MUST be disabled in production

SERVER STARTUP BLOCKED. Fix the above security issues and restart.
exit status 1
```

### 3. Post-Deployment Verification

```bash
# Verify no debug endpoints
curl http://your-server:8080/debug/pprof/
# Should return 404

# Verify real PVP provider
curl http://your-server:8080/api/v1/beta/health
# Check logs for "Using PVP provider: stripe" (not "mock")

# Test MCP SSRF protection (should be blocked)
# This would require MCP client access - validate in logs
```

---

## Migration from Development to Production

### Step 1: Inventory Current Configuration

```bash
# Check current environment
env | grep GAUTH

# Check for weak defaults
grep -r "dev-please-change" .env* || echo "No weak defaults in .env files"
```

### Step 2: Generate Production Secrets

```bash
# JWT signing key
export GAUTH_JWT_SIGNING_KEY=$(openssl rand -base64 48)

# Database password  
export GAUTH_DB_PASSWORD=$(openssl rand -base64 24)

# Store in secrets management
# AWS Secrets Manager, HashiCorp Vault, Azure Key Vault, etc.
```

### Step 3: Configure External Services

```bash
# Sign up for identity verification provider
# Example: Stripe Identity
# 1. Create Stripe account at https://stripe.com
# 2. Enable Stripe Identity in dashboard
# 3. Get API key from https://dashboard.stripe.com/apikeys

export GAUTH_PVP_PROVIDER=stripe
export STRIPE_API_KEY=sk_live_...
```

### Step 4: Update Infrastructure

```yaml
# docker-compose.production.yml
services:
  gauth:
    environment:
      - GAUTH_ENV=production
      - GAUTH_JWT_SIGNING_KEY=${GAUTH_JWT_SIGNING_KEY}
      - GAUTH_PVP_PROVIDER=stripe
      - STRIPE_API_KEY=${STRIPE_API_KEY}
      # NO GAUTH_DEV_INDEX or GAUTH_DEV_MODE
```

### Step 5: Deploy and Validate

```bash
# Deploy
docker-compose -f docker-compose.production.yml up -d

# Check logs for security validation
docker-compose logs gauth | grep SECURITY

# Should see:
# [SECURITY] Production mode detected - enforcing security validations  
# [SECURITY] All security validations passed ✓
```

---

## Additional Security Recommendations

### Network Security

- [ ] Use TLS/HTTPS in production (terminate at load balancer or use `GAUTH_TLS_CERT` and `GAUTH_TLS_KEY`)
- [ ] Implement Web Application Firewall (WAF)
- [ ] Use private subnets for backend services
- [ ] Restrict database access to application subnet only

### Key Rotation

- [ ] Rotate JWT signing keys periodically (suggest 90 days)
- [ ] Use key versioning (e.g., `GAUTH_JWT_SIGNING_KEY_VERSION=v2`)
- [ ] Implement zero-downtime key rotation

### Monitoring

- [ ] Set up alerts for:
  - Failed authentication attempts (rate > threshold)
  - Invalid token signatures (potential forgery attempts)
  - MCP SSRF attempts (blocked URIs)
  - PVP verification failures

### Compliance

- [ ] Review PVP provider compliance certifications (SOC 2, ISO 27001)
- [ ] Document identity verification procedures for audits
- [ ] Implement audit logging for all PoA issuance/verification

---

## Support and Resources

- **Security Issue Reporting**: security@example.com (REPLACE WITH ACTUAL CONTACT)
- **Documentation**: See `PRODUCTION_DEPLOYMENT_GUIDE.md` for detailed deployment instructions
- **Identity Verification Providers**:
  - Stripe Identity: https://stripe.com/docs/identity
  - Veriff: https://developers.veriff.com/
  - Idemia: https://www.idemia.com/identity-verification
  - Onfido: https://documentation.onfido.com/
  - Jumio: https://www.jumio.com/developers/

---

## Summary

| Vulnerability | Severity | Status | Remediation |
|---------------|----------|--------|-------------|
| Weak JWT Key | 🔴 CRITICAL | ✅ Fixed | Startup validation blocks weak keys |
| MCP SSRF | 🔴 HIGH | ✅ Fixed | URI validation in `pkg/mcp/client.go` |
| Mock PVP | 🟡 MEDIUM | ⚠️ Partial | Framework ready, providers need implementation |
| Debug UI | 🟢 LOW | ✅ Fixed | Startup validation blocks in production |

**Action Required**: 
1. ✅ Review this checklist
2. ✅ Set production environment variables  
3. ⚠️ Implement real PVP provider integration (or use mock with explicit production override if compliance not required)
4. ✅ Run security tests
5. ✅ Deploy and verify startup validation passes

---

**Last Updated**: November 2025
**Version**: 1.0
**Maintainer**: GAuth Security Team
