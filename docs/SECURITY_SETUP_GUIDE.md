---
title: Security Setup Guide
category: security-setup-guide
status: active
lastUpdated: 2025-11-12
owners: security-team
source: internal
refreshCadence: quarterly
---

# Security Setup Guide

> **Status:** Beta Demonstration  
> **Last Updated:** October 24, 2025

## Overview

This guide helps you enable and configure security features for the GAuth Go Beta repository, including Code Scanning, Secret Scanning, and Dependency Review.

---

## 1. Enable GitHub Advanced Security Features

### For Public Repositories (Free)

GitHub Advanced Security features are **free for public repositories**. Enable them in repository settings:

1. Navigate to: `https://github.com/mauriciomferz/Gauth_go/settings`
2. Click **Code security and analysis** in the left sidebar
3. Enable the following:
   - ✅ **Dependency graph** (should be auto-enabled)
   - ✅ **Dependabot alerts**
   - ✅ **Dependabot security updates**
   - ✅ **Code scanning** → Click "Set up" → Choose "Default" or "Advanced"
   - ✅ **Secret scanning** (auto-enabled for public repos)

### For Private Repositories

Advanced Security requires GitHub Enterprise or a trial:
- Contact your organization admin to enable GitHub Advanced Security
- Or request a trial: https://github.com/enterprise/trial

---

## 2. CodeQL Configuration

### Current Setup

The repository has CodeQL configured in `.github/workflows/codeql.yml` with:

**Analyzed Languages:**
- ✅ **Go** (autobuild mode)
- ✅ **JavaScript/TypeScript** (no build)
- ✅ **Python** (no build)
- ✅ **GitHub Actions** (no build)

**Scan Triggers:**
- Push to `main` branch
- Pull requests to `main` branch
- Weekly schedule (Wednesday at 20:22 UTC)

**Coverage Status:**
- Current: 287 out of 656 Go files (43.8%)
- Target: 80%+ coverage recommended

### Improve Coverage

To analyze more files, ensure all packages are included:

```yaml
# In .github/workflows/codeql.yml
- name: Initialize CodeQL
  uses: github/codeql-action/init@v3
  with:
    languages: ${{ matrix.language }}
    build-mode: ${{ matrix.build-mode }}
    # Add custom queries for better security coverage
    queries: security-extended,security-and-quality
    # Specify paths to include
    config-file: ./.github/codeql/codeql-config.yml
```

Create `.github/codeql/codeql-config.yml`:

```yaml
name: "GAuth CodeQL Config"

paths:
  - 'cmd/**'
  - 'internal/**'
  - 'pkg/**'
  - 'web/**'
  - 'examples/**'

paths-ignore:
  - 'test/**'
  - 'node_modules/**'
  - '**/*_test.go'

queries:
  - uses: security-extended
  - uses: security-and-quality
```

---

## 3. Current Workflow Status

### CodeQL Workflow Results

From the latest run:

```
✅ CodeQL scanned 287 out of 656 Go files
⚠️  Warning: Code scanning is not enabled in repository settings
❌ Error: Could not upload SARIF results
```

**Action Required:** Enable Code Scanning in repository settings (see Section 1)

### CI/CD Workflow

The main CI workflow (`.github/workflows/ci.yml`) includes:
- ✅ Go 1.25 testing
- ✅ Redis integration tests
- ✅ PostgreSQL integration tests
- ✅ Comprehensive test suite with timeout controls
- ✅ Package-level test isolation

---

## 4. Security Best Practices

### Secrets Management

**Current Implementation:**
- ✅ Vault integration for key storage (`internal/crypto/vault_keystore.go`)
- ✅ KMS abstraction layer (`internal/crypto/kms_keystore.go`)
- ✅ HSM-ready architecture

**GitHub Secrets:**
Store sensitive values as GitHub secrets:

```bash
# Using GitHub CLI
gh secret set VAULT_ADDR --body "https://vault.example.com"
gh secret set VAULT_TOKEN --body "<your-token>"
gh secret set KMS_KEY_ID --body "<your-key-id>"
```

Then reference in workflows:

```yaml
env:
  VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
  VAULT_TOKEN: ${{ secrets.VAULT_TOKEN }}
```

### Dependency Security

**Automated Scanning:**
- Dependabot alerts for vulnerable dependencies
- Automated security updates
- CodeQL dependency analysis

**Manual Review:**
```bash
# Check for known vulnerabilities
go list -json -m all | nancy sleuth

# Audit Go modules
go mod verify
go mod tidy

# Use govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### Code Scanning Alerts

**View Current Alerts:**
- Navigate to: `https://github.com/mauriciomferz/Gauth_go/security/code-scanning`
- Filter by severity: Critical, High, Medium, Low
- Review and remediate

**Common Security Issues to Watch:**
1. **SQL Injection** - Use parameterized queries
2. **Path Traversal** - Validate file paths (see docs route in `web/server_clean.go`)
3. **Hardcoded Credentials** - Use environment variables or secret stores
4. **Weak Cryptography** - Use Ed25519, avoid MD5/SHA1
5. **Unvalidated Redirects** - Sanitize redirect URLs

---

## 5. Beta Security Considerations

### Current Beta Limitations

⚠️ **This is a BETA demonstration - NOT for production use**

**Known Beta Limitations:**
1. **Authentication:** Demo-grade token validation
2. **Rate Limiting:** Basic implementation, needs hardening
3. **Input Validation:** Enhanced for demo, requires production-grade sanitization
4. **Error Handling:** Verbose for debugging (should be minimal in production)
5. **Logging:** May expose sensitive data in debug mode

### Hardening Checklist (Pre-Production)

Before production deployment:

- [ ] Enable strict CSP (Content Security Policy)
- [ ] Implement rate limiting per-endpoint
- [ ] Add request size limits
- [ ] Enable HTTPS/TLS only (no HTTP)
- [ ] Implement proper CORS policies
- [ ] Add DDoS protection
- [ ] Enable audit logging for all mutations
- [ ] Implement session management
- [ ] Add 2FA/MFA support
- [ ] Perform security penetration testing
- [ ] Complete OWASP Top 10 review
- [ ] Run static analysis (gosec, staticcheck)
- [ ] Implement secrets rotation
- [ ] Add intrusion detection

---

## 6. Security Monitoring

### GitHub Security Dashboard

Monitor security at:
- **Overview:** `https://github.com/mauriciomferz/Gauth_go/security`
- **Code Scanning:** `https://github.com/mauriciomferz/Gauth_go/security/code-scanning`
- **Secret Scanning:** `https://github.com/mauriciomferz/Gauth_go/security/secret-scanning`
- **Dependabot:** `https://github.com/mauriciomferz/Gauth_go/security/dependabot`

### Runtime Security (Beta Demo)

The webapp includes security metrics:
- Violation counters: `http://localhost:8080/api/v1/beta/metrics/violations`
- Policy metrics: `http://localhost:8080/api/v1/beta/policy/metrics`
- Audit logs: `http://localhost:8080/api/v1/audit/logs`
- Token metrics: `http://localhost:8080/api/v1/token/metrics`

### Prometheus Metrics

Export security metrics:
```bash
curl http://localhost:8080/api/v1/beta/metrics/violations/prometheus
curl http://localhost:8080/api/v1/beta/policy/metrics/prometheus
curl http://localhost:8080/api/v1/beta/authz/metrics/prometheus
```

---

## 7. Incident Response

### Security Issue Reporting

**For Security Vulnerabilities:**
- **DO NOT** open public GitHub issues
- Email: [Add security contact email]
- Use GitHub Security Advisories: `https://github.com/mauriciomferz/Gauth_go/security/advisories/new`

**Response SLA (Beta):**
- Critical: 24 hours
- High: 72 hours
- Medium: 1 week
- Low: Best effort

### Emergency Response

If a security incident occurs:

1. **Assess Impact:** Determine scope and severity
2. **Contain:** Disable affected endpoints/features
3. **Investigate:** Review logs, audit trail
4. **Remediate:** Apply fixes, deploy patches
5. **Document:** Create post-mortem report
6. **Communicate:** Notify users if data exposed

---

## 8. Compliance & Standards

### Implemented Standards

- ✅ **RFC-0150** - GAuth Protocol Compliance
- ✅ **RFC-0111** - Semantic Validation
- ✅ **RFC-0115** - Authorization Framework
- ✅ **OWASP** - Security best practices (partial)
- ⚠️ **SOC 2** - Not certified (beta demonstration)
- ⚠️ **ISO 27001** - Not certified (beta demonstration)
- ⚠️ **GDPR** - Not compliant (beta demonstration)

### Audit Trail

The system implements comprehensive audit logging:
- Multi-tenant key rotation audit (`internal/crypto/multitenant_manager.go`)
- External audit anchor integration (`pkg/ledger/external_anchor.go`)
- Hash-chained audit events
- Prometheus metrics export (8 rotation metrics)

**Audit Trail Features:**
```go
// Rotation events with tenant segregation
type RotationEvent struct {
    ID               string
    Timestamp        time.Time
    Tenant           string  // Multi-tenant isolation
    Type             string
    OldKeyID         string
    NewKeyID         string
    Backend          string
    RotationDuration time.Duration
    Success          bool
    Error            string
    Metadata         map[string]interface{}
}
```

---

## 9. Next Steps

### Immediate Actions

1. **Enable Code Scanning** in GitHub repository settings
2. **Review CodeQL alerts** once scanning is enabled
3. **Enable Dependabot alerts** and security updates
4. **Configure branch protection** rules for `main`
5. **Set up GitHub Secrets** for sensitive values

### Short Term (1-2 weeks)

1. Increase CodeQL coverage from 43.8% to 80%+
2. Add custom CodeQL queries for GAuth-specific patterns
3. Implement automated security testing in CI
4. Review and remediate all High/Critical alerts
5. Document security architecture

### Long Term (Production Readiness)

1. Complete OWASP Top 10 security review
2. Perform penetration testing
3. Implement production-grade authentication
4. Add comprehensive input validation
5. Enable strict CSP and security headers
6. Implement rate limiting and DDoS protection
7. Add intrusion detection and monitoring
8. Complete compliance certifications (if required)

---

## 10. Resources

### Documentation

- [GitHub Code Scanning Docs](https://docs.github.com/en/code-security/code-scanning)
- [CodeQL Documentation](https://codeql.github.com/docs/)
- [Dependabot Documentation](https://docs.github.com/en/code-security/dependabot)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Checklist](https://github.com/Checkmarx/Go-SCP)

### Internal Documentation

- `docs/P1_KEY_ROTATION_COMPLETION_REPORT.md` - Key rotation implementation
- `docs/ENHANCED_POA_VALIDATOR_SUMMARY.md` - PoA validation
- `docs/EXTERNAL_AUDIT_ANCHOR.md` - Audit anchoring
- `docs/GAP_MATRIX.md` - Implementation status
- `web/server_clean.go` - API security implementation

### Tools

```bash
# Static analysis
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...

# Vulnerability scanning
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Dependency auditing
go install github.com/sonatype-nexus-community/nancy@latest
go list -json -m all | nancy sleuth

# Code quality
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

---

## Support

For security questions or concerns:
- GitHub Issues (non-security): `https://github.com/mauriciomferz/Gauth_go/issues`
- Security Advisories: `https://github.com/mauriciomferz/Gauth_go/security/advisories`
- Documentation: `http://localhost:8080/docs/SECURITY_SETUP_GUIDE.md`

---

**Remember:** This is a BETA demonstration. Always perform thorough security reviews before any production deployment.
