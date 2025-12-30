# OIDC Phase 3: External Providers - Design Document

**Status:** 🎯 In Progress  
**Phase:** 3 of 4  
**Started:** November 12, 2025  
**Estimated Duration:** 5-6 days  
**Compliance Impact:** 65% → 68%

---

## 📋 Executive Summary

OIDC Phase 3 extends our internal OIDC implementation (Phases 1 & 2) to support external identity providers (Google, Okta, Azure AD). This enables AgentAuth to act as both an OIDC provider (internal) and an OIDC client/relying party (external), facilitating federated authentication across multiple identity domains.

**Key Goals:**
1. Support major external OIDC providers (Google, Okta, Azure AD)
2. Multi-tenant provider configuration management
3. Discovery document caching with TTL and refresh
4. Provider-specific claim mapping and normalization
5. Token exchange between provider formats
6. Comprehensive integration testing

---

## 🎯 Objectives

### Primary Objectives
- ✅ **Multi-Provider Support:** Register and manage multiple OIDC providers
- ✅ **Provider Discovery:** Auto-discover provider endpoints via OIDC Discovery
- ✅ **Claim Mapping:** Normalize provider-specific claims to AgentAuth format
- ✅ **Token Exchange:** Handle provider-specific token formats
- ✅ **Configuration Management:** Support provider-specific configurations
- ✅ **Caching:** Cache discovery documents to reduce latency

### Secondary Objectives
- ⏳ **Provider Templates:** Pre-configured templates for common providers
- ⏳ **Dynamic Registration:** Support dynamic client registration (RFC 7591)
- ⏳ **Token Revocation:** Support token revocation (RFC 7009)
- ⏳ **Provider Health Checks:** Monitor provider availability

---

## 🏗️ Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        AgentAuth System                          │
│                                                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │             OIDC Phase 3: External Providers          │  │
│  │                                                        │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────┐ │  │
│  │  │   Provider   │  │   Discovery  │  │   Token    │ │  │
│  │  │    Config    │  │    Cache     │  │  Exchange  │ │  │
│  │  └──────┬───────┘  └──────┬───────┘  └─────┬──────┘ │  │
│  │         │                  │                 │        │  │
│  │         └──────────────────┼─────────────────┘        │  │
│  │                            │                          │  │
│  │  ┌─────────────────────────┴──────────────────────┐  │  │
│  │  │           Provider Registry                    │  │  │
│  │  │  - Google OIDC                                 │  │  │
│  │  │  - Okta OIDC                                   │  │  │
│  │  │  - Azure AD OIDC                               │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │        OIDC Phase 1 & 2 (Existing)                   │  │
│  │  - Discovery Service                                  │  │
│  │  - ID Token Service                                   │  │
│  │  - Identity Bridge (ACR ↔ AgentAuth Trust)              │  │
│  │  - PowerVerificationPoint (PVP)                      │  │
│  │  - PVP Router                                        │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

#### 1. Provider Configuration (`pkg/oidc/provider_config.go`)
**Purpose:** Manage external provider configurations

**Responsibilities:**
- Register new OIDC providers
- Store provider metadata (client_id, client_secret, issuer URL, etc.)
- Validate provider configurations
- Support multi-tenant provider instances
- Provide provider lookup by ID/name

**Key Types:**
```go
type ProviderConfig struct {
    ID             string            // Unique provider identifier
    Name           string            // Human-readable name (e.g., "Google")
    IssuerURL      string            // Provider's issuer URL
    ClientID       string            // OAuth2 client ID
    ClientSecret   string            // OAuth2 client secret
    Scopes         []string          // Required scopes (openid, profile, email)
    ClaimMappings  map[string]string // Provider claim → AgentAuth claim mapping
    TenantID       string            // For multi-tenant providers (Azure AD)
    Metadata       map[string]any    // Provider-specific metadata
    Enabled        bool              // Provider enabled/disabled
}

type ProviderRegistry interface {
    Register(cfg ProviderConfig) error
    Get(id string) (*ProviderConfig, error)
    List() []ProviderConfig
    Update(id string, cfg ProviderConfig) error
    Delete(id string) error
}
```

#### 2. Discovery Cache (`pkg/oidc/discovery_cache.go`)
**Purpose:** Cache provider discovery documents

**Responsibilities:**
- Fetch and cache OIDC discovery documents
- Respect cache TTL and max-age headers
- Auto-refresh expired documents
- Handle cache misses gracefully
- Provide thread-safe access

**Key Types:**
```go
type CachedDiscovery struct {
    Document   *OIDCConfiguration  // Discovery document
    FetchedAt  time.Time           // When cached
    ExpiresAt  time.Time           // When to refresh
    ETag       string              // ETag for conditional requests
}

type DiscoveryCache interface {
    Get(issuerURL string) (*OIDCConfiguration, error)
    Set(issuerURL string, doc *OIDCConfiguration, ttl time.Duration) error
    Invalidate(issuerURL string) error
    Clear() error
}
```

#### 3. Provider-Specific Integrations (`pkg/oidc/providers/`)

##### Google (`google.go`)
**Provider URL:** `https://accounts.google.com`  
**Discovery:** `https://accounts.google.com/.well-known/openid-configuration`

**Key Characteristics:**
- Standard OIDC implementation
- Claims: `sub`, `email`, `email_verified`, `name`, `picture`, `hd` (hosted domain)
- Trust Level: Maps to `substantial` by default
- Scopes: `openid`, `profile`, `email`

**Claim Mappings:**
```go
var GoogleClaimMappings = map[string]string{
    "sub":            "user_id",
    "email":          "email",
    "email_verified": "email_verified",
    "name":           "full_name",
    "picture":        "avatar_url",
    "hd":             "organization",
}
```

##### Okta (`okta.go`)
**Provider URL:** `https://{domain}.okta.com` or `https://{domain}.oktapreview.com`  
**Discovery:** `https://{domain}.okta.com/.well-known/openid-configuration`

**Key Characteristics:**
- Enterprise-grade OIDC
- Claims: `sub`, `email`, `email_verified`, `name`, `preferred_username`, `groups`
- Trust Level: Maps based on MFA status → `substantial` (MFA) or `low` (no MFA)
- Scopes: `openid`, `profile`, `email`, `groups`

**Claim Mappings:**
```go
var OktaClaimMappings = map[string]string{
    "sub":                "user_id",
    "email":              "email",
    "email_verified":     "email_verified",
    "name":               "full_name",
    "preferred_username": "username",
    "groups":             "roles",
}
```

##### Azure AD (`azure.go`)
**Provider URL:** `https://login.microsoftonline.com/{tenant}/v2.0`  
**Discovery:** `https://login.microsoftonline.com/{tenant}/v2.0/.well-known/openid-configuration`

**Key Characteristics:**
- Multi-tenant support required
- Claims: `sub`, `email`, `name`, `preferred_username`, `tid` (tenant ID), `roles`
- Trust Level: Maps based on `amr` (authentication method) → `high` (MFA), `substantial` (password)
- Scopes: `openid`, `profile`, `email`, `User.Read`

**Claim Mappings:**
```go
var AzureADClaimMappings = map[string]string{
    "sub":                "user_id",
    "email":              "email",
    "name":               "full_name",
    "preferred_username": "username",
    "tid":                "tenant_id",
    "roles":              "roles",
}
```

#### 4. Token Exchange (`pkg/oidc/token_exchange.go`)
**Purpose:** Exchange tokens between provider formats

**Responsibilities:**
- Validate external provider ID tokens
- Extract and normalize claims
- Map trust levels based on provider ACR/AMR
- Convert to AgentAuth internal format
- Handle provider-specific quirks

**Key Functions:**
```go
func ExchangeExternalToken(
    providerID string,
    idToken string,
    nonce string,
) (*auth.ExtendedToken, error)

func NormalizeClaims(
    providerID string,
    claims map[string]any,
) (*auth.Identity, error)

func MapTrustLevel(
    providerID string,
    acr string,
    amr []string,
) (string, error)
```

---

## 🔐 Security Considerations

### 1. Provider Validation
- ✅ Validate provider issuer matches discovery document
- ✅ Verify HTTPS for all provider communications
- ✅ Check token signature with provider's JWKS
- ✅ Validate audience claim matches our client_id

### 2. Secret Management
- ⚠️ **CRITICAL:** Never log client secrets
- ⚠️ Store secrets encrypted at rest
- ⚠️ Support secret rotation without downtime
- ⚠️ Use environment variables or secret managers (not config files)

### 3. Token Validation
- ✅ Verify token signature (RS256/384/512)
- ✅ Check expiration (`exp` claim)
- ✅ Validate issuer (`iss` claim)
- ✅ Validate audience (`aud` claim)
- ✅ Check nonce (if provided in auth request)
- ✅ Validate trust level claims (ACR, AMR)

### 4. Rate Limiting
- ⚠️ Implement rate limiting for discovery requests
- ⚠️ Cache discovery documents aggressively (1-24 hours)
- ⚠️ Implement backoff on provider errors

---

## 🧪 Testing Strategy

### Unit Tests
**Coverage Target:** 85%+

**Test Coverage:**
1. Provider configuration CRUD operations
2. Discovery cache TTL and expiration
3. Claim mapping for each provider
4. Token validation (valid/invalid/expired)
5. Trust level mapping logic
6. Error handling (network errors, invalid tokens, etc.)

### Integration Tests
**File:** `test/integration/external_providers_test.go`

**Test Scenarios:**
1. **Google OIDC Flow:**
   - Mock Google discovery endpoint
   - Mock token validation
   - Verify claim mapping
   - Test trust level → `substantial`

2. **Okta OIDC Flow:**
   - Mock Okta discovery endpoint
   - Test MFA vs non-MFA trust levels
   - Verify group claim mapping
   - Test multi-tenant scenarios

3. **Azure AD OIDC Flow:**
   - Mock Azure AD discovery endpoint (with tenant ID)
   - Test AMR-based trust level mapping
   - Verify role claim mapping
   - Test tenant isolation

4. **Multi-Provider Scenario:**
   - Register all 3 providers
   - Test switching between providers
   - Verify no cross-contamination of configs

5. **Discovery Cache:**
   - Test cache hit/miss
   - Test TTL expiration
   - Test cache invalidation
   - Test concurrent access

### Mock Provider
**Purpose:** Test external provider integration without live credentials

**Implementation:**
- Create mock HTTP server for discovery endpoint
- Return valid discovery document
- Mock JWKS endpoint with test keys
- Support configurable responses (success, error, timeout)

---

## 📊 Success Criteria

### Functional Requirements
- [ ] Register and manage 3+ external providers
- [ ] Fetch and cache provider discovery documents
- [ ] Validate ID tokens from external providers
- [ ] Map provider claims to AgentAuth format
- [ ] Convert provider trust levels to AgentAuth trust levels
- [ ] Support Google, Okta, Azure AD out-of-the-box

### Quality Requirements
- [ ] 85%+ test coverage on new code
- [ ] All integration tests passing
- [ ] Clean compilation (no errors/warnings)
- [ ] Documentation complete (API docs, configuration guide)

### Performance Requirements
- [ ] Discovery cache reduces latency by 90%+
- [ ] Token validation < 50ms (excluding network)
- [ ] Support 1000+ providers in registry
- [ ] Concurrent provider operations (thread-safe)

### Security Requirements
- [ ] Client secrets never logged
- [ ] All provider communication over HTTPS
- [ ] Token signature verification mandatory
- [ ] Audience validation mandatory

---

## 📐 Implementation Plan

### Day 1: Foundation
**Duration:** 6-8 hours

**Tasks:**
1. Create `pkg/oidc/provider_config.go`
   - ProviderConfig struct
   - ProviderRegistry interface + in-memory implementation
   - CRUD operations
   - Validation logic

2. Create `pkg/oidc/discovery_cache.go`
   - CachedDiscovery struct
   - DiscoveryCache interface + in-memory implementation
   - TTL and expiration logic
   - Thread-safe access (sync.RWMutex)

3. Write unit tests for both components
   - Provider registration/lookup
   - Cache operations (get, set, invalidate)

**Deliverables:**
- Provider configuration management ✅
- Discovery caching ✅
- 50+ unit tests

### Day 2: Google Provider
**Duration:** 6-8 hours

**Tasks:**
1. Create `pkg/oidc/providers/google.go`
   - Google-specific configuration
   - Claim mappings
   - Trust level logic (default: substantial)

2. Integrate with discovery cache
   - Fetch Google discovery document
   - Cache with 24h TTL

3. Write Google integration tests
   - Mock Google discovery endpoint
   - Test claim mapping
   - Test token validation

**Deliverables:**
- Google OIDC support ✅
- 20+ tests

### Day 3: Okta Provider
**Duration:** 6-8 hours

**Tasks:**
1. Create `pkg/oidc/providers/okta.go`
   - Okta-specific configuration
   - Claim mappings (including groups)
   - Trust level logic (MFA-aware)

2. Integrate with discovery cache
   - Support custom Okta domains

3. Write Okta integration tests
   - Test MFA vs non-MFA flows
   - Test group claim mapping

**Deliverables:**
- Okta OIDC support ✅
- 20+ tests

### Day 4: Azure AD Provider
**Duration:** 6-8 hours

**Tasks:**
1. Create `pkg/oidc/providers/azure.go`
   - Azure AD-specific configuration
   - Multi-tenant support
   - Claim mappings (including roles)
   - AMR-based trust level logic

2. Integrate with discovery cache
   - Support tenant-specific discovery

3. Write Azure AD integration tests
   - Test multi-tenant scenarios
   - Test AMR-based trust levels
   - Test role claim mapping

**Deliverables:**
- Azure AD OIDC support ✅
- 20+ tests

### Day 5: Token Exchange & Integration
**Duration:** 6-8 hours

**Tasks:**
1. Create `pkg/oidc/token_exchange.go`
   - ExchangeExternalToken function
   - NormalizeClaims function
   - MapTrustLevel function

2. Integrate with existing PVP Router
   - Add external provider support
   - Test with Steps I, III, VI

3. Write comprehensive integration tests
   - Test all 3 providers end-to-end
   - Test multi-provider scenarios
   - Test discovery cache effectiveness

**Deliverables:**
- Token exchange logic ✅
- Full integration with AgentAuth ✅
- 30+ integration tests

### Day 6: Documentation & Polish
**Duration:** 4-6 hours

**Tasks:**
1. Write `OIDC_PHASE3_EXTERNAL_PROVIDERS_REPORT.md`
   - Implementation details
   - Configuration guide
   - Provider specifications
   - Testing results

2. Update `EXECUTIVE_SUMMARY_GAP_REMEDIATION.md`
   - Add Phase 3 achievements
   - Update compliance (65% → 68%)

3. Code review and polish
   - Address any TODO comments
   - Improve error messages
   - Add inline documentation

**Deliverables:**
- Comprehensive documentation ✅
- Clean, production-ready code ✅
- Updated compliance tracking ✅

---

## 🎯 Compliance Impact

**Before Phase 3:** 65%  
**After Phase 3:** 68% (+3%)

**RFC-0111 Requirements Addressed:**
- ✅ External Identity Provider Integration (P1)
- ✅ Federated Authentication Support (P1)
- ✅ Multi-Tenant Provider Support (P2)
- ✅ Trust Level Propagation (P1)

---

## 📋 Configuration Example

```yaml
# config/oidc_providers.yaml

providers:
  - id: google
    name: Google
    issuer_url: https://accounts.google.com
    client_id: ${GOOGLE_CLIENT_ID}
    client_secret: ${GOOGLE_CLIENT_SECRET}
    scopes:
      - openid
      - profile
      - email
    claim_mappings:
      sub: user_id
      email: email
      name: full_name
    default_trust_level: substantial
    enabled: true

  - id: okta-prod
    name: Okta Production
    issuer_url: https://mycompany.okta.com
    client_id: ${OKTA_CLIENT_ID}
    client_secret: ${OKTA_CLIENT_SECRET}
    scopes:
      - openid
      - profile
      - email
      - groups
    claim_mappings:
      sub: user_id
      email: email
      groups: roles
    trust_mapping:
      mfa_enabled: substantial
      password_only: low
    enabled: true

  - id: azure-ad
    name: Azure AD
    issuer_url: https://login.microsoftonline.com/{tenant}/v2.0
    tenant_id: ${AZURE_TENANT_ID}
    client_id: ${AZURE_CLIENT_ID}
    client_secret: ${AZURE_CLIENT_SECRET}
    scopes:
      - openid
      - profile
      - email
      - User.Read
    claim_mappings:
      sub: user_id
      email: email
      roles: roles
    trust_mapping:
      mfa: high
      password: substantial
    enabled: true

discovery_cache:
  ttl: 24h              # Cache discovery documents for 24 hours
  max_entries: 100      # Maximum cached providers
  refresh_before: 1h    # Refresh 1 hour before expiration
```

---

## 🚀 Next Steps

**After Phase 3 Completion:**
1. **OIDC Phase 4:** Production Hardening (4-5 days)
   - Security audit
   - Performance optimization
   - Secret management (Vault, KMS)
   - Health checks and monitoring
   - Configuration validation
   - Compliance: 68% → 72%

2. **MCP Integration:** Model Context Protocol (2-3 weeks)
   - Study MCP specification
   - Design server/client architecture
   - Implement AI context management
   - Compliance: 72% → 75%

3. **E2E Testing:** Full system validation (1-2 weeks)
   - Complete end-to-end test suite
   - Load testing
   - Security testing
   - Compliance: 75% → 80%

---

## 📚 References

**Standards:**
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html)
- [RFC 7591 - OAuth 2.0 Dynamic Client Registration](https://tools.ietf.org/html/rfc7591)
- [RFC 7009 - OAuth 2.0 Token Revocation](https://tools.ietf.org/html/rfc7009)

**Provider Documentation:**
- [Google OIDC](https://developers.google.com/identity/protocols/oauth2/openid-connect)
- [Okta OIDC](https://developer.okta.com/docs/reference/api/oidc/)
- [Azure AD OIDC](https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-protocols-oidc)

**Related Documents:**
- `OIDC_PHASE1_IMPLEMENTATION_REPORT.md` - Core OIDC infrastructure
- `OIDC_PHASE2_PVP_INTEGRATION_REPORT.md` - PVP integration
- `EXECUTIVE_SUMMARY_GAP_REMEDIATION.md` - Overall compliance tracking

---

**Document Status:** 🎯 Active  
**Last Updated:** November 12, 2025  
**Next Review:** After Day 2 implementation
