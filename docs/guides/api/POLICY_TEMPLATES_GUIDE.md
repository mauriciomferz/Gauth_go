# Policy Templates Guide

## Overview
AgentAuth provides three pre-configured policy templates for authorization management, each designed for different security and usability requirements. These templates can be selected during tenant/subscriber onboarding and are used to configure the authorization policy framework.

**Location:** Subscribers Management → Step 4: Policy Templates  
**Purpose:** Simplify policy configuration by providing pre-defined authorization rules

---

## Template Types

### 1. 🛡️ Standard (Balanced Security and Usability)
**Recommended for:** Production environments, most business applications

**Description:**  
The Standard template provides a balanced approach between security and usability, suitable for most production deployments. It implements industry-standard security practices while maintaining good user experience.

**Characteristics:**
- **Security Level:** Medium-High
- **Use Case:** General business applications, SaaS platforms
- **Flexibility:** Balanced
- **Recommended For:** 
  - Production applications
  - Business-to-business (B2B) platforms
  - Enterprise applications with standard security requirements
  - Multi-tenant SaaS applications

**Security Features:**
- ✅ Token validation with reasonable expiration times
- ✅ Standard rate limiting
- ✅ Basic geo-restriction capabilities
- ✅ Scope-based access control
- ✅ Role-based authorization
- ✅ Audit logging enabled
- ✅ MFA support (optional)

**Policy Configuration:**
```yaml
template: standard
features:
  - token_validation: enabled
  - token_expiry: 3600s (1 hour)
  - refresh_token: enabled
  - rate_limiting: standard (100 req/min)
  - geo_restriction: configurable
  - scope_enforcement: enabled
  - audit_logging: full
  - mfa: optional
  - session_timeout: 30 minutes
```

**Example Use Cases:**
- Corporate web applications
- Business management systems
- Customer portals
- API services for partners
- Internal tools with external access

---

### 2. 🔒 Strict (Maximum Security)
**Recommended for:** High-security environments, sensitive data applications

**Description:**  
The Strict template implements maximum security controls with stringent validation, monitoring, and access restrictions. This template prioritizes security over convenience and is designed for applications handling highly sensitive data or operating in regulated industries.

**Characteristics:**
- **Security Level:** Very High
- **Use Case:** Financial services, healthcare, government, defense
- **Flexibility:** Low (security-first)
- **Recommended For:**
  - Financial institutions
  - Healthcare applications (HIPAA-compliant)
  - Government systems
  - Defense/military applications
  - PCI-DSS compliant systems
  - Applications handling PII/PHI

**Security Features:**
- ✅ Strict token validation with short expiration
- ✅ Aggressive rate limiting
- ✅ Mandatory geo-restrictions
- ✅ Strict scope enforcement
- ✅ Mandatory role-based authorization
- ✅ Comprehensive audit logging
- ✅ MFA required for all operations
- ✅ IP whitelisting support
- ✅ Time-based access restrictions
- ✅ Device fingerprinting
- ✅ Anomaly detection

**Policy Configuration:**
```yaml
template: strict
features:
  - token_validation: strict
  - token_expiry: 900s (15 minutes)
  - refresh_token: disabled
  - rate_limiting: aggressive (20 req/min)
  - geo_restriction: mandatory
  - scope_enforcement: strict
  - audit_logging: comprehensive
  - mfa: required
  - session_timeout: 10 minutes
  - ip_whitelist: enabled
  - time_restrictions: enabled
  - device_fingerprint: required
  - anomaly_detection: enabled
  - failed_attempts_lockout: 3 attempts
```

**Additional Restrictions:**
- No concurrent sessions allowed
- Mandatory re-authentication for sensitive operations
- Stricter CORS policies
- Enhanced input validation
- Mandatory encryption at rest and in transit
- Regular security audits required

**Example Use Cases:**
- Banking applications
- Electronic health record (EHR) systems
- Payment processing platforms
- Government portals
- Defense systems
- Compliance-critical applications

---

### 3. 🔓 Relaxed (Development/Testing)
**Recommended for:** Development, testing, and demo environments

**Description:**  
The Relaxed template provides minimal security restrictions to facilitate rapid development, testing, and demonstrations. This template should **NEVER** be used in production environments.

**Characteristics:**
- **Security Level:** Low
- **Use Case:** Development, testing, demos, proof-of-concepts
- **Flexibility:** Very High
- **Recommended For:**
  - Local development environments
  - Integration testing
  - Automated testing suites
  - Demonstrations and prototypes
  - Internal training systems

**Security Features:**
- ⚠️ Basic token validation
- ⚠️ Extended token expiration
- ⚠️ Minimal rate limiting
- ⚠️ No geo-restrictions
- ⚠️ Relaxed scope enforcement
- ⚠️ Basic audit logging
- ⚠️ MFA disabled
- ⚠️ Permissive CORS

**Policy Configuration:**
```yaml
template: relaxed
features:
  - token_validation: basic
  - token_expiry: 86400s (24 hours)
  - refresh_token: enabled
  - rate_limiting: lenient (1000 req/min)
  - geo_restriction: disabled
  - scope_enforcement: lenient
  - audit_logging: basic
  - mfa: disabled
  - session_timeout: 8 hours
  - cors: permissive
  - validation: minimal
```

**⚠️ Important Warnings:**
- **DO NOT USE IN PRODUCTION**
- Suitable only for development and testing
- No compliance guarantees
- Minimal security protections
- Increased vulnerability to attacks
- Should be restricted to development networks

**Example Use Cases:**
- Local development environments (localhost)
- Continuous Integration/Continuous Deployment (CI/CD) pipelines
- Automated testing frameworks
- Demo environments for sales/marketing
- Training and educational platforms (sandbox)
- Proof-of-concept development

---

## 4. ⚙️ Custom (Define Your Own)
**Recommended for:** Organizations with specific security requirements

**Description:**  
The Custom template allows you to define your own authorization policies using YAML configuration. This provides maximum flexibility to implement organization-specific security policies.

**Use Cases:**
- Unique compliance requirements
- Hybrid security models
- Industry-specific regulations
- Multi-level security clearances
- Complex organizational policies

**Configuration Method:**
When selecting "Custom", you can define policies in YAML format:

```yaml
policies:
  - name: admin
    actions: [read, write, delete]
    resources: ["*"]
    conditions:
      - ip_range: 10.0.0.0/8
      - mfa_required: true
      
  - name: readonly
    actions: [read]
    resources: ["documents/*", "reports/*"]
    conditions:
      - time_range: "09:00-17:00"
      
  - name: api_access
    actions: [read, write]
    resources: ["api/*"]
    conditions:
      - rate_limit: 100
      - scope: ["api:read", "api:write"]
```

---

## Template Selection Guide

### Decision Matrix

| Requirement | Standard | Strict | Relaxed | Custom |
|-------------|----------|--------|---------|--------|
| **Production Ready** | ✅ Yes | ✅ Yes | ❌ No | ⚠️ Depends |
| **Compliance** | ⚠️ Moderate | ✅ High | ❌ None | ⚠️ Configurable |
| **User Experience** | ✅ Good | ⚠️ Moderate | ✅ Excellent | ⚠️ Varies |
| **Development Speed** | ✅ Good | ⚠️ Slower | ✅ Fast | ⚠️ Varies |
| **Security Level** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐ | ⭐⭐⭐⭐ |
| **Setup Complexity** | Low | Low | Low | High |
| **Maintenance** | Low | Medium | Low | High |

### Industry Recommendations

| Industry | Recommended Template | Reason |
|----------|---------------------|---------|
| **Financial Services** | Strict | Regulatory compliance (PCI-DSS, SOX) |
| **Healthcare** | Strict | HIPAA compliance, PHI protection |
| **E-commerce** | Standard | Balance security and UX |
| **SaaS Platforms** | Standard | General business needs |
| **Government** | Strict | Security clearances, data sensitivity |
| **Education** | Standard/Relaxed | Lower security requirements |
| **Media/Content** | Standard | Content access control |
| **Development Tools** | Relaxed | Developer experience priority |
| **IoT Platforms** | Standard/Custom | Device-specific requirements |

---

## Implementation Details

### Where Templates Are Used

1. **Subscriber Creation** (`POST /api/admin/subscribers`)
   - Selected during tenant onboarding wizard
   - Step 4 of 8-step process
   - Stored in `policy_template` field

2. **Database Storage** (`subscribers` table)
   ```sql
   policy_template VARCHAR(100)
   ```

3. **Frontend Component** (`Subscribers.tsx`)
   - Radio button selection
   - Line 417-428
   - Custom YAML editor for custom template

4. **Backend Handler** (`subscriber_handler.go`)
   - Validation of template selection
   - Integration with authorization engine

### Template Application Flow

```
User Selects Template
    ↓
Frontend Validation
    ↓
POST /api/admin/subscribers
    ↓
Backend Validation
    ↓
Store in Database
    ↓
Apply to Authorization Engine
    ↓
Tenant Configured
```

---

## Migration Between Templates

### Upgrading Security (Relaxed → Standard → Strict)

**Considerations:**
- Review existing tokens (may need re-issuance)
- Communicate changes to users
- Enable MFA gradually (if moving to Strict)
- Update rate limits incrementally
- Test thoroughly before production rollout

**Process:**
1. Backup current configuration
2. Update `policy_template` in database
3. Apply new template configuration
4. Notify affected users
5. Monitor for issues
6. Roll back if necessary

### Downgrading Security (Not Recommended)

**⚠️ Warning:** Downgrading security templates can expose vulnerabilities and should be avoided in production environments.

---

## Best Practices

### ✅ Do's
- Use **Standard** for most production applications
- Use **Strict** for sensitive data or regulated industries
- Use **Relaxed** only for development/testing
- Review and update templates regularly
- Monitor security events and adjust as needed
- Document template selection rationale
- Test thoroughly before production deployment

### ❌ Don'ts
- Don't use Relaxed in production
- Don't mix templates across environments
- Don't skip security reviews when using Custom
- Don't ignore compliance requirements
- Don't disable MFA in Strict template
- Don't modify templates without testing

---

## Future Enhancements

### Planned Features
- [ ] Template versioning
- [ ] Template cloning/inheritance
- [ ] Dynamic template switching based on context
- [ ] Template analytics and recommendations
- [ ] Pre-configured industry-specific templates
- [ ] Template testing and validation tools
- [ ] Template marketplace for community sharing

---

## Related Documentation

- **Subscriber Management**: `ADMIN_PORTAL_IMPLEMENTATION_STATUS.md`
- **Authorization Engine**: `docs/RFC_IMPLEMENTATION_COVERAGE.md`
- **Security Settings**: `API_KEYS_SECURITY_IMPLEMENTATION.md`
- **Policy Administration**: `docs/P_STAR_P_USER_GUIDE.md`
- **Compliance Guide**: `SECURITY_COMPLIANCE_GUIDE.md`

---

## Support and Questions

For questions about policy template selection or custom configuration:
- Review the security requirements for your application
- Consult with your security team
- Reference compliance requirements for your industry
- Test templates in staging environment first

**Last Updated:** November 24, 2025  
**Version:** 1.0.0
