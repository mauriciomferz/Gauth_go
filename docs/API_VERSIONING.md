# GAuth API Versioning & Deprecation Policy

**Version**: 1.0  
**Effective Date**: November 2025  
**Last Updated**: November 15, 2025

---

## Table of Contents

1. [Overview](#overview)
2. [Versioning Strategy](#versioning-strategy)
3. [Current Versions](#current-versions)
4. [Deprecation Policy](#deprecation-policy)
5. [Breaking Changes](#breaking-changes)
6. [Migration Process](#migration-process)
7. [Backward Compatibility](#backward-compatibility)
8. [API Lifecycle](#api-lifecycle)
9. [Version Support Timeline](#version-support-timeline)
10. [Best Practices](#best-practices)

---

## Overview

GAuth follows a clear and predictable versioning strategy to ensure stability for integrators while allowing the API to evolve. This document outlines our commitment to backward compatibility and provides guidance on managing API changes.

### Key Principles

1. **Stability**: Existing integrations should continue to work without modification
2. **Predictability**: Changes are announced well in advance with clear timelines
3. **Transparency**: All changes are documented with migration guides
4. **Developer-First**: Decisions prioritize developer experience and ease of migration

---

## Versioning Strategy

### URL-Based Versioning

GAuth uses **URL-based versioning** as the primary versioning mechanism:

```
https://api.gauth.example.com/api/{version}/{endpoint}
```

**Examples:**
- `/api/v1/beta/health` - Version 1 (beta)
- `/api/v1/rfc0111/subscriptions` - Version 1 (stable)
- `/api/v2/subscriptions` - Version 2 (future)

### Version Header Support

Optionally, clients can specify versions using headers:

```http
GET /api/health HTTP/1.1
X-API-Version: v1
```

If both URL and header versions are present, the **URL version takes precedence**.

### Semantic Versioning for SDKs

Client SDKs follow [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** (1.0.0 → 2.0.0): Breaking changes
- **MINOR** (1.0.0 → 1.1.0): New features, backward compatible
- **PATCH** (1.0.0 → 1.0.1): Bug fixes, backward compatible

---

## Current Versions

### Version 1 (v1) - CURRENT

**Status**: Beta → Stable (transition planned Q1 2026)  
**Base Path**: `/api/v1`  
**Release Date**: November 2025  
**End of Life**: TBD (minimum 2 years from stable release)

**Endpoints:**
- System: `/api/v1/beta/health`, `/api/v1/beta/info`, `/api/v1/beta/ping`
- RFC-0111: `/api/v1/rfc0111/subscriptions/*`
- PoA: `/api/v1/beta/poa/*`
- Authorization: `/api/v1/beta/authz/*`
- PVP: `/api/v1/beta/pvp/*`
- Registry: `/api/v1/beta/registry/*`
- Policy: `/api/v1/beta/policy/*`
- Tokens: `/api/v1/token/*`
- Metrics: `/api/v1/beta/metrics/*`

**Beta Notice:**  
Endpoints marked with `/beta/` may receive breaking changes with shorter notice periods (minimum 3 months). Once stable, they will follow the full deprecation policy.

---

## Deprecation Policy

### Deprecation Timeline

When an API version or endpoint is deprecated:

1. **T+0 (Announcement)**: Deprecation is announced via:
   - API response headers (`X-API-Deprecated: true`)
   - Changelog entry
   - Email to registered developers
   - Blog post
   - Documentation update

2. **T+6 Months (Warning Period)**: 
   - Deprecation warnings in API responses
   - Migration guide published
   - New version available for testing

3. **T+12 Months (Removal)**:
   - Deprecated version is removed
   - Requests return 410 Gone with migration instructions

### Beta Endpoints

Beta endpoints have a **shorter timeline**:

1. **T+0 (Announcement)**: Deprecation announced
2. **T+3 Months (Warning)**: Migration guide available
3. **T+6 Months (Removal)**: Endpoint removed

### Deprecation Headers

Deprecated endpoints return the following headers:

```http
HTTP/1.1 200 OK
X-API-Deprecated: true
X-API-Deprecated-Date: 2025-11-15T00:00:00Z
X-API-Sunset-Date: 2026-11-15T00:00:00Z
X-API-Migration-Guide: https://docs.gauth.example.com/migration/v1-to-v2
Link: <https://api.gauth.example.com/api/v2/subscriptions>; rel="successor-version"
```

### Deprecation Response Body

Deprecated endpoints include a warning in the response:

```json
{
  "data": { /* response data */ },
  "deprecated": {
    "deprecated_at": "2025-11-15T00:00:00Z",
    "sunset_at": "2026-11-15T00:00:00Z",
    "message": "This endpoint is deprecated and will be removed on 2026-11-15",
    "migration_guide": "https://docs.gauth.example.com/migration/v1-to-v2",
    "successor": "/api/v2/subscriptions"
  }
}
```

---

## Breaking Changes

### What Constitutes a Breaking Change?

Breaking changes include:

1. **Removing endpoints or fields** from responses
2. **Renaming fields** in requests or responses
3. **Changing field types** (string → number, etc.)
4. **Changing authentication mechanisms**
5. **Removing or changing error codes**
6. **Changing HTTP methods** (POST → PUT)
7. **Making optional fields required**
8. **Changing URL structure**

### What is NOT a Breaking Change?

Non-breaking changes include:

1. **Adding new endpoints**
2. **Adding new optional fields** to requests
3. **Adding new fields** to responses
4. **Adding new error codes** (clients should handle unknown codes gracefully)
5. **Changing internal implementation** without API changes
6. **Improving performance**
7. **Bug fixes** that restore documented behavior

### Handling Breaking Changes

Breaking changes are **only introduced in major versions**:

- v1 → v2: Breaking changes allowed
- v1.0 → v1.1: No breaking changes
- v1.0.0 → v1.0.1 (SDK): No breaking changes

---

## Migration Process

### Step-by-Step Migration Guide

When migrating from one version to another:

#### Step 1: Review the Changelog

```bash
# Check the API changelog
curl https://api.gauth.example.com/api/changelog
```

#### Step 2: Test Against New Version

```bash
# Use the new version in a test environment
curl https://api-staging.gauth.example.com/api/v2/subscriptions
```

#### Step 3: Update Your Code

**Before (v1):**
```javascript
const response = await fetch('/api/v1/beta/poa', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}` },
  body: JSON.stringify({ grantor, grantee, scope })
});
```

**After (v2):**
```javascript
const response = await fetch('/api/v2/authorizations', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}` },
  body: JSON.stringify({ 
    principal: grantor, 
    delegate: grantee, 
    permissions: scope 
  })
});
```

#### Step 4: Update SDK Version

```bash
# JavaScript/TypeScript
npm install @gauth/client@2.0.0

# Python
pip install gauth-client==2.0.0
```

#### Step 5: Deploy and Monitor

- Deploy to staging first
- Run integration tests
- Monitor error rates
- Gradually roll out to production

---

## Backward Compatibility

### Compatibility Guarantees

Within a major version (e.g., v1):

✅ **Guaranteed Compatible:**
- Existing endpoints will not be removed
- Existing fields will not be renamed
- Existing field types will not change
- Authentication mechanisms remain stable

✅ **May Be Added (Non-Breaking):**
- New optional request fields
- New response fields
- New endpoints
- New error codes

❌ **Not Guaranteed:**
- Exact error messages (use error codes instead)
- Response field ordering
- Undocumented behavior

### Forward Compatibility

Design your clients to be forward-compatible:

1. **Ignore unknown fields** in responses
2. **Handle new error codes** gracefully (treat as generic errors)
3. **Don't rely on field ordering** in JSON responses
4. **Parse timestamps** as ISO 8601, not specific formats
5. **Follow redirects** (3xx responses)

**Example: Forward-Compatible Client**

```python
def parse_subscription(data):
    # Extract only the fields we know about
    return {
        'id': data['id'],
        'status': data.get('status', 'unknown'),
        'client_id': data.get('client_id')
        # Unknown fields are ignored automatically
    }
```

---

## API Lifecycle

### Lifecycle Stages

```
Experimental → Beta → Stable → Deprecated → Retired
```

#### 1. Experimental (Optional)

- **Duration**: 1-3 months
- **Path**: `/api/experimental/*`
- **Guarantees**: None - may change at any time
- **Notice**: Not for production use

#### 2. Beta

- **Duration**: 3-12 months
- **Path**: `/api/v1/beta/*`
- **Guarantees**: 3-month deprecation notice
- **Notice**: Use cautiously in production

#### 3. Stable

- **Duration**: 2+ years
- **Path**: `/api/v1/*` (no `/beta/`)
- **Guarantees**: 12-month deprecation notice
- **Notice**: Production-ready

#### 4. Deprecated

- **Duration**: 6-12 months
- **Headers**: `X-API-Deprecated: true`
- **Guarantees**: Continues to work during deprecation period
- **Notice**: Migrate to new version

#### 5. Retired

- **Status**: 410 Gone
- **Response**: Migration instructions
- **Notice**: No longer available

---

## Version Support Timeline

### Active Support

- **Full Support**: Bug fixes, security patches, feature updates
- **Duration**: Current major version + previous major version

### Security Support

- **Security Patches Only**: Critical security vulnerabilities
- **Duration**: Previous major version only (1 year after new major release)

### End of Life

- **No Support**: No updates or patches
- **Status**: Retired versions return 410 Gone

### Example Timeline

```
v1 Released:        Nov 2025  ━━━━━━━━━━━━━━━ Full Support ━━━━━━━━━━━━━━━┓
v2 Released:                  Nov 2026  ━━━━━━━━━ Full Support ━━━━━━━━━━┃
v1 Deprecated:                Nov 2026  ━━━━ Security Only ━━━━━━┓        ┃
v1 Retired:                             Nov 2027  ✗ EOL          ┃        ┃
v3 Released:                                      Nov 2027  ━━━━━┃━━━━━━━━┛
v2 Deprecated:                                    Nov 2027  ━━━━━┛━━━━━━┓
v2 Retired:                                                 Nov 2028  ✗ EOL
```

---

## Best Practices

### For API Clients

1. **Always specify version** in URLs (don't rely on defaults)
2. **Check deprecation headers** in responses
3. **Subscribe to changelog** notifications
4. **Test against new versions** early
5. **Use official SDKs** for automatic version handling
6. **Implement graceful degradation** for unknown fields
7. **Monitor API status** page for announcements

### For API Developers

1. **Document all changes** in CHANGELOG.md
2. **Announce deprecations** 6+ months in advance
3. **Provide migration guides** with code examples
4. **Version SDKs** according to semver
5. **Maintain OpenAPI specs** for each version
6. **Run breaking change detection** in CI/CD
7. **Support previous version** for 12 months minimum

---

## Migration Guides

### v1-beta → v1-stable (Planned Q1 2026)

**Changes:**
- Remove `/beta/` from stable endpoints
- Finalize request/response schemas
- Lock in field names and types

**Action Required:**
- Update URLs: `/api/v1/beta/poa` → `/api/v1/poa`
- Test integration with stable endpoints

**Migration Timeline:**
- **Dec 2025**: v1-stable available for testing
- **Feb 2026**: v1-beta deprecated (6-month notice)
- **Aug 2026**: v1-beta retired

---

## Changelog Access

### API Changelog Endpoint

```bash
# Get changelog for current version
curl https://api.gauth.example.com/api/changelog

# Get changelog for specific version
curl https://api.gauth.example.com/api/changelog?version=v1
```

### GitHub Releases

All versions are tagged and released on GitHub:

https://github.com/mauriciomferz/Gauth_go/releases

### RSS Feed

Subscribe to changelog updates:

https://api.gauth.example.com/api/changelog.rss

---

## Support and Contact

### Questions About Versioning?

- **Email**: api-support@gauth.example.com
- **GitHub Discussions**: https://github.com/mauriciomferz/Gauth_go/discussions
- **Slack**: #gauth-api-versioning

### Report Issues

- **GitHub Issues**: https://github.com/mauriciomferz/Gauth_go/issues
- **Security Issues**: security@gauth.example.com

---

## References

- [Semantic Versioning](https://semver.org/)
- [RFC 8594 - Sunset HTTP Header](https://tools.ietf.org/html/rfc8594)
- [API Evolution Best Practices](https://tools.ietf.org/html/draft-ietf-httpapi-api-evolution)

---

**Last Updated**: November 15, 2025  
**Next Review**: February 2026
