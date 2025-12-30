---
title: OIDC Integration Design
category: design-spec
status: active
lastUpdated: 2025-11-12
owners: platform-eng
source: internal
refreshCadence: on-change
---

# OpenID Connect Integration Architecture Design
## AAP-001 Building Block Implementation

**Document Version**: 1.0  
**Date**: November 12, 2025  
**Status**: Design Phase  
**Priority**: P1 - RFC REQUIREMENT

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [AAP-001 Requirements](#rfc-0111-requirements)
3. [Current State Analysis](#current-state-analysis)
4. [Design Goals](#design-goals)
5. [Architecture Overview](#architecture-overview)
6. [Component Design](#component-design)
7. [Integration Strategy](#integration-strategy)
8. [Implementation Phases](#implementation-phases)
9. [Security Considerations](#security-considerations)
10. [Testing Strategy](#testing-strategy)
11. [Migration Path](#migration-path)
12. [References](#references)

---

## Executive Summary

### Purpose

This document defines the architecture for integrating **OpenID Connect (OIDC)** as a required building block into the AgentAuth 1.0 implementation, ensuring full compliance with AAP-001 Section 1 (Scope) requirements.

### Current Gap

**Finding**: AgentAuth currently uses custom `IdentityProofRequest/Result` structures for identity verification instead of standard OIDC ID tokens.

**Evidence from Audit**:
```
❌ No OpenID Connect Implementation
- No ID tokens
- No UserInfo endpoint  
- No OIDC Discovery
- No Dynamic Client Registration
- No Session Management
- OpenID Connect Compliance: 0%
```

**Impact**: 
- Cannot interoperate with OIDC-compliant identity providers
- Violates AAP-001 building block requirement
- Limits adoption by enterprises using standard OIDC infrastructure

### Solution Approach

**Hybrid Integration Model**: Extend existing AgentAuth structures with OIDC compatibility while maintaining backward compatibility with current AAP-001 implementation.

**Key Strategy**:
1. Implement OIDC as identity verification layer
2. Map OIDC ID tokens to AgentAuth identity structures
3. Support standard OIDC flows alongside AgentAuth subscription flow
4. Enable interoperability with major OIDC providers (Google, Okta, Auth0, Keycloak)

### Expected Outcomes

- ✅ AAP-001 OIDC requirement satisfied (0% → 90%)
- ✅ Overall compliance increase (+6%): 62% → 68%
- ✅ Enterprise-grade identity verification
- ✅ Interoperability with standard OIDC ecosystem
- ✅ Backward compatibility maintained

---

## AAP-001 Requirements

### Section 1: Scope - Building Blocks

**Direct Quote from AAP-001**:
> "AgentAuth builds on the following standards as building blocks:
> 
> **OpenID Connect or its alternatives, including but not limited to:**
> - OpenID Connect Discovery 1.0
> - OpenID Connect Dynamic Client Registration
> - OpenID Connect Session Management"

### Compliance Interpretation

**MUST Implement**:
1. **OIDC Discovery** - `.well-known/openid-configuration` endpoint
2. **ID Token Support** - JWT-based identity tokens with standard claims
3. **UserInfo Endpoint** - Retrieve authenticated user information
4. **Dynamic Client Registration** (Optional but recommended)
5. **Session Management** (Optional but recommended)

**Integration Points**:
- AAP-001 Step I: Owner's Authorizer Identity Proof → **Use OIDC ID Token**
- AAP-001 Step III: Client Owner Identity Proof → **Use OIDC ID Token**
- AAP-001 Step VI: Resource Owner Identity Proof → **Use OIDC ID Token**

---

## Current State Analysis

### Existing Identity Verification

**Current Implementation** (`subscription_flow.go`):

```go
// PowerVerificationPoint interface
type PowerVerificationPoint interface {
    VerifyIdentityProof(ctx context.Context, 
        request *IdentityProofRequest) (*IdentityProofResult, error)
}

// Custom identity structures
type IdentityProofRequest struct {
    SubjectID      string
    IdentityType   string // "natural_person", "legal_entity"
    ProofMethod    string // "eIDAS", "government_id", "commercial_register"
    ProofData      map[string]interface{}
    RequiredLevel  string // "substantial", "high"
}

type IdentityProofResult struct {
    Valid          bool
    SubjectID      string
    Identity       string
    VerifiedAt     time.Time
    TrustLevel     string
    FailureReason  string
}
```

**Strengths**:
- ✅ Well-structured interface for identity verification
- ✅ Supports multiple proof methods (eIDAS, government ID, commercial register)
- ✅ Trust level concept maps well to OIDC ACR (Authentication Context Class Reference)
- ✅ Already integrated into subscription flow Steps I, III, VI

**Gaps**:
- ❌ No standard OIDC token format
- ❌ Cannot consume ID tokens from external OIDC providers
- ❌ No OIDC Discovery support
- ❌ No interoperability with OIDC ecosystem

---

## Design Goals

### Primary Goals

1. **AAP-001 Compliance**: Satisfy OIDC building block requirement
2. **Interoperability**: Work with major OIDC providers (Google, Microsoft Azure AD, Okta, Auth0, Keycloak)
3. **Backward Compatibility**: Don't break existing AgentAuth subscription flow
4. **Enterprise-Grade**: Support enterprise identity requirements (SSO, MFA, ACR levels)

### Non-Goals

- ❌ Replace existing AgentAuth authorization chain (keep AgentAuth's unique PoA model)
- ❌ Implement full OAuth 2.0 server (focus on OIDC as identity layer)
- ❌ Support all optional OIDC features (focus on core + Discovery)

---

## Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    AgentAuth Authorization Server                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │            OIDC Identity Layer (NEW)                         │ │
│  ├──────────────────────────────────────────────────────────────┤ │
│  │ • Discovery Endpoint (.well-known/openid-configuration)      │ │
│  │ • Authorization Endpoint (/authorize)                        │ │
│  │ • Token Endpoint (/token) → Issues ID Tokens                 │ │
│  │ • UserInfo Endpoint (/userinfo)                              │ │
│  │ • JWKS Endpoint (/jwks) → Public keys for signature verify   │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                            ↓                                        │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │         Identity Bridge (NEW)                                │ │
│  ├──────────────────────────────────────────────────────────────┤ │
│  │ • ID Token → IdentityProofResult converter                   │ │
│  │ • ACR → TrustLevel mapper                                    │ │
│  │ • External OIDC provider client (Google, Okta, etc.)         │ │
│  │ • Identity claims aggregation                                │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                            ↓                                        │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │  Existing AgentAuth Subscription Flow (Steps I-VIII)             │ │
│  ├──────────────────────────────────────────────────────────────┤ │
│  │ • Step I: Owner's Authorizer Identity (uses OIDC)            │ │
│  │ • Step III: Client Owner Identity (uses OIDC)                │ │
│  │ • Step VI: Resource Owner Identity (uses OIDC)               │ │
│  │ • Authorization Chain Validation                             │ │
│  │ • Extended Token Issuance                                    │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘

     External OIDC Providers (Federation)
     ┌────────────┐  ┌────────────┐  ┌────────────┐
     │   Google   │  │    Okta    │  │   Azure    │
     │  Identity  │  │  Identity  │  │     AD     │
     └────────────┘  └────────────┘  └────────────┘
```

### Component Layers

1. **OIDC Protocol Layer** - Standard OIDC endpoints (Discovery, Token, UserInfo, JWKS)
2. **Identity Bridge Layer** - Converts OIDC tokens to AgentAuth identity structures
3. **AgentAuth Application Layer** - Existing subscription flow and authorization logic

---

## Component Design

### 1. OIDC Discovery Service

**Purpose**: Provide `.well-known/openid-configuration` endpoint for OIDC discovery.

**File**: `pkg/oidc/discovery.go`

```go
package oidc

import (
    "encoding/json"
    "net/http"
)

// OIDCConfiguration represents OIDC discovery metadata
type OIDCConfiguration struct {
    Issuer                string   `json:"issuer"`
    AuthorizationEndpoint string   `json:"authorization_endpoint"`
    TokenEndpoint         string   `json:"token_endpoint"`
    UserInfoEndpoint      string   `json:"userinfo_endpoint"`
    JWKSUri               string   `json:"jwks_uri"`
    
    // Supported features
    ResponseTypesSupported []string `json:"response_types_supported"`
    SubjectTypesSupported  []string `json:"subject_types_supported"`
    IDTokenSigningAlgValues []string `json:"id_token_signing_alg_values_supported"`
    ScopesSupported        []string `json:"scopes_supported"`
    ClaimsSupported        []string `json:"claims_supported"`
    
    // OPTIONAL: Dynamic registration
    RegistrationEndpoint   string   `json:"registration_endpoint,omitempty"`
    
    // OPTIONAL: ACR support (trust levels)
    ACRValuesSupported     []string `json:"acr_values_supported,omitempty"`
}

// DiscoveryService manages OIDC discovery
type DiscoveryService struct {
    issuerURL string
    config    *OIDCConfiguration
}

// NewDiscoveryService creates OIDC discovery service
func NewDiscoveryService(issuerURL string) *DiscoveryService {
    return &DiscoveryService{
        issuerURL: issuerURL,
        config: &OIDCConfiguration{
            Issuer:                issuerURL,
            AuthorizationEndpoint: issuerURL + "/authorize",
            TokenEndpoint:         issuerURL + "/token",
            UserInfoEndpoint:      issuerURL + "/userinfo",
            JWKSUri:               issuerURL + "/jwks",
            
            ResponseTypesSupported: []string{"code", "id_token", "token id_token"},
            SubjectTypesSupported:  []string{"public"},
            IDTokenSigningAlgValues: []string{"RS256", "HS256"},
            
            // AgentAuth-specific scopes
            ScopesSupported: []string{
                "openid", "profile", "email",
                "agentauth:owner", "agentauth:client", "agentauth:resource",
            },
            
            // Standard OIDC claims + AgentAuth extensions
            ClaimsSupported: []string{
                "sub", "name", "email", "email_verified",
                "given_name", "family_name", "picture",
                // AgentAuth extensions
                "entity_type", "entity_id", "commercial_register",
                "legal_entity_name", "jurisdiction",
            },
            
            // Trust levels (maps to AgentAuth TrustLevel)
            ACRValuesSupported: []string{
                "substantial", "high", "loa-2", "loa-3", "loa-4",
            },
        },
    }
}

// ServeHTTP handles discovery requests
func (d *DiscoveryService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(d.config)
}
```

**Endpoints**:
- `GET /.well-known/openid-configuration` → Returns discovery metadata

---

### 2. ID Token Service

**Purpose**: Issue and validate OIDC ID tokens.

**File**: `pkg/oidc/id_token.go`

```go
package oidc

import (
    "context"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
)

// IDTokenClaims represents standard OIDC ID token claims
type IDTokenClaims struct {
    jwt.RegisteredClaims
    
    // Standard OIDC claims
    Name              string `json:"name,omitempty"`
    GivenName         string `json:"given_name,omitempty"`
    FamilyName        string `json:"family_name,omitempty"`
    Email             string `json:"email,omitempty"`
    EmailVerified     bool   `json:"email_verified,omitempty"`
    Picture           string `json:"picture,omitempty"`
    
    // Authentication context
    ACR               string `json:"acr,omitempty"` // Authentication Context Class Reference
    AMR               []string `json:"amr,omitempty"` // Authentication Methods References
    AuthTime          int64  `json:"auth_time,omitempty"`
    
    // AgentAuth extensions (for legal entities)
    EntityType        string `json:"entity_type,omitempty"` // "natural_person", "legal_entity"
    EntityID          string `json:"entity_id,omitempty"` // Commercial register ID
    LegalEntityName   string `json:"legal_entity_name,omitempty"`
    CommercialRegister string `json:"commercial_register,omitempty"` // "DE-HRB", "UK-CH", etc.
    Jurisdiction      string `json:"jurisdiction,omitempty"`
    
    // Nonce for replay protection
    Nonce             string `json:"nonce,omitempty"`
}

// IDTokenService manages ID token lifecycle
type IDTokenService struct {
    issuerURL  string
    signingKey []byte // For HMAC or RSA private key
    algorithm  jwt.SigningMethod
    validity   time.Duration
}

// NewIDTokenService creates ID token service
func NewIDTokenService(issuerURL string, signingKey []byte) *IDTokenService {
    return &IDTokenService{
        issuerURL:  issuerURL,
        signingKey: signingKey,
        algorithm:  jwt.SigningMethodRS256, // RS256 recommended for OIDC
        validity:   15 * time.Minute, // ID tokens are short-lived
    }
}

// IssueIDToken creates a new ID token
func (s *IDTokenService) IssueIDToken(
    ctx context.Context,
    subject string,
    audience string,
    claims *IDTokenClaims,
) (string, error) {
    now := time.Now()
    
    claims.Issuer = s.issuerURL
    claims.Subject = subject
    claims.Audience = jwt.ClaimStrings{audience}
    claims.ExpiresAt = jwt.NewNumericDate(now.Add(s.validity))
    claims.IssuedAt = jwt.NewNumericDate(now)
    claims.NotBefore = jwt.NewNumericDate(now)
    
    if claims.AuthTime == 0 {
        claims.AuthTime = now.Unix()
    }
    
    token := jwt.NewWithClaims(s.algorithm, claims)
    return token.SignedString(s.signingKey)
}

// ValidateIDToken verifies and parses ID token
func (s *IDTokenService) ValidateIDToken(
    ctx context.Context,
    tokenString string,
    expectedAudience string,
) (*IDTokenClaims, error) {
    claims := &IDTokenClaims{}
    
    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
        // Verify signing algorithm
        if token.Method != s.algorithm {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Method)
        }
        return s.signingKey, nil
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to parse ID token: %w", err)
    }
    
    if !token.Valid {
        return nil, fmt.Errorf("invalid ID token")
    }
    
    // Verify audience
    if !claims.VerifyAudience(expectedAudience, true) {
        return nil, fmt.Errorf("invalid audience")
    }
    
    // Verify issuer
    if !claims.VerifyIssuer(s.issuerURL, true) {
        return nil, fmt.Errorf("invalid issuer")
    }
    
    // Verify expiration
    if !claims.VerifyExpiresAt(time.Now(), true) {
        return nil, fmt.Errorf("token expired")
    }
    
    return claims, nil
}
```

---

### 3. Identity Bridge

**Purpose**: Convert OIDC ID tokens to AgentAuth `IdentityProofResult`.

**File**: `pkg/oidc/identity_bridge.go`

```go
package oidc

import (
    "context"
    "fmt"
    "time"
    
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/agentauth"
)

// IdentityBridge converts OIDC identity to AgentAuth identity structures
type IdentityBridge struct {
    idTokenService *IDTokenService
    trustMapper    *TrustLevelMapper
}

// NewIdentityBridge creates identity bridge
func NewIdentityBridge(idTokenService *IDTokenService) *IdentityBridge {
    return &IdentityBridge{
        idTokenService: idTokenService,
        trustMapper:    NewTrustLevelMapper(),
    }
}

// ConvertIDTokenToIdentityProof converts OIDC ID token to AgentAuth identity proof
func (b *IdentityBridge) ConvertIDTokenToIdentityProof(
    ctx context.Context,
    idToken string,
    expectedAudience string,
) (*agentauth.IdentityProofResult, error) {
    // Validate ID token
    claims, err := b.idTokenService.ValidateIDToken(ctx, idToken, expectedAudience)
    if err != nil {
        return &agentauth.IdentityProofResult{
            Valid:         false,
            FailureReason: fmt.Sprintf("ID token validation failed: %v", err),
        }, nil
    }
    
    // Extract identity information
    identity := claims.Subject
    if claims.Name != "" {
        identity = claims.Name
    }
    if claims.LegalEntityName != "" {
        identity = claims.LegalEntityName
    }
    
    // Map ACR to AgentAuth trust level
    trustLevel := b.trustMapper.MapACRToTrustLevel(claims.ACR)
    
    return &agentauth.IdentityProofResult{
        Valid:      true,
        SubjectID:  claims.Subject,
        Identity:   identity,
        VerifiedAt: time.Now(),
        TrustLevel: trustLevel,
    }, nil
}

// TrustLevelMapper maps OIDC ACR to AgentAuth trust levels
type TrustLevelMapper struct {
    acrMappings map[string]string
}

// NewTrustLevelMapper creates trust level mapper
func NewTrustLevelMapper() *TrustLevelMapper {
    return &TrustLevelMapper{
        acrMappings: map[string]string{
            // OIDC ACR → AgentAuth TrustLevel
            "loa-1":        "low",
            "loa-2":        "medium",
            "loa-3":        "substantial",
            "loa-4":        "high",
            "substantial":  "substantial",
            "high":         "high",
        },
    }
}

// MapACRToTrustLevel converts OIDC ACR to AgentAuth trust level
func (m *TrustLevelMapper) MapACRToTrustLevel(acr string) string {
    if level, ok := m.acrMappings[acr]; ok {
        return level
    }
    return "medium" // Default trust level
}

// MapTrustLevelToACR converts AgentAuth trust level to OIDC ACR
func (m *TrustLevelMapper) MapTrustLevelToACR(trustLevel string) string {
    // Reverse mapping
    for acr, level := range m.acrMappings {
        if level == trustLevel {
            return acr
        }
    }
    return "loa-2" // Default ACR
}
```

---

### 4. External OIDC Provider Client

**Purpose**: Federate with external OIDC providers (Google, Okta, etc.).

**File**: `pkg/oidc/provider_client.go`

```go
package oidc

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

// ExternalOIDCProvider represents external OIDC provider configuration
type ExternalOIDCProvider struct {
    Name          string
    IssuerURL     string
    ClientID      string
    ClientSecret  string
    RedirectURI   string
    Scopes        []string
    
    // Discovered endpoints (from .well-known/openid-configuration)
    AuthEndpoint  string
    TokenEndpoint string
    UserInfoEndpoint string
    JWKSUri       string
}

// OIDCProviderClient manages external OIDC provider integration
type OIDCProviderClient struct {
    provider       *ExternalOIDCProvider
    httpClient     *http.Client
    idTokenService *IDTokenService
}

// NewOIDCProviderClient creates external OIDC provider client
func NewOIDCProviderClient(provider *ExternalOIDCProvider) (*OIDCProviderClient, error) {
    client := &OIDCProviderClient{
        provider:   provider,
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
    
    // Discover provider configuration
    if err := client.Discover(context.Background()); err != nil {
        return nil, fmt.Errorf("failed to discover provider: %w", err)
    }
    
    return client, nil
}

// Discover fetches OIDC provider configuration
func (c *OIDCProviderClient) Discover(ctx context.Context) error {
    discoveryURL := c.provider.IssuerURL + "/.well-known/openid-configuration"
    
    req, err := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
    if err != nil {
        return err
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("discovery failed: %s", resp.Status)
    }
    
    var config OIDCConfiguration
    if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
        return err
    }
    
    // Update provider endpoints
    c.provider.AuthEndpoint = config.AuthorizationEndpoint
    c.provider.TokenEndpoint = config.TokenEndpoint
    c.provider.UserInfoEndpoint = config.UserInfoEndpoint
    c.provider.JWKSUri = config.JWKSUri
    
    return nil
}

// GetAuthorizationURL generates authorization URL for OAuth 2.0 flow
func (c *OIDCProviderClient) GetAuthorizationURL(state, nonce string) string {
    params := url.Values{}
    params.Set("client_id", c.provider.ClientID)
    params.Set("redirect_uri", c.provider.RedirectURI)
    params.Set("response_type", "code")
    params.Set("scope", strings.Join(c.provider.Scopes, " "))
    params.Set("state", state)
    params.Set("nonce", nonce)
    
    return c.provider.AuthEndpoint + "?" + params.Encode()
}

// ExchangeCodeForToken exchanges authorization code for tokens
func (c *OIDCProviderClient) ExchangeCodeForToken(
    ctx context.Context,
    code string,
) (*TokenResponse, error) {
    // Implementation: POST to token endpoint with authorization code
    // Returns access_token, id_token, refresh_token
    // ... (standard OAuth 2.0 token exchange)
}

// VerifyIDToken validates ID token from external provider
func (c *OIDCProviderClient) VerifyIDToken(
    ctx context.Context,
    idToken string,
) (*IDTokenClaims, error) {
    // Implementation: Validate ID token signature using JWKS
    // Verify issuer, audience, expiration
    // Parse claims
    // ... (standard OIDC ID token validation)
}
```

---

### 5. OIDC-Enabled PowerVerificationPoint

**Purpose**: Extend existing PowerVerificationPoint to support OIDC.

**File**: `pkg/agentauth/pvp_oidc.go`

```go
package agentauth

import (
    "context"
    "fmt"
    
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/oidc"
)

// OIDCPowerVerificationPoint implements PowerVerificationPoint with OIDC support
type OIDCPowerVerificationPoint struct {
    identityBridge  *oidc.IdentityBridge
    providerClients map[string]*oidc.OIDCProviderClient
}

// NewOIDCPowerVerificationPoint creates OIDC-enabled PVP
func NewOIDCPowerVerificationPoint(
    identityBridge *oidc.IdentityBridge,
) *OIDCPowerVerificationPoint {
    return &OIDCPowerVerificationPoint{
        identityBridge:  identityBridge,
        providerClients: make(map[string]*oidc.OIDCProviderClient),
    }
}

// RegisterProvider adds external OIDC provider
func (pvp *OIDCPowerVerificationPoint) RegisterProvider(
    name string,
    provider *oidc.ExternalOIDCProvider,
) error {
    client, err := oidc.NewOIDCProviderClient(provider)
    if err != nil {
        return fmt.Errorf("failed to register provider %s: %w", name, err)
    }
    pvp.providerClients[name] = client
    return nil
}

// VerifyIdentityProof implements PowerVerificationPoint interface
func (pvp *OIDCPowerVerificationPoint) VerifyIdentityProof(
    ctx context.Context,
    request *IdentityProofRequest,
) (*IdentityProofResult, error) {
    // Check proof method
    switch request.ProofMethod {
    case "oidc_id_token":
        return pvp.verifyOIDCIDToken(ctx, request)
    case "oidc_external":
        return pvp.verifyExternalOIDC(ctx, request)
    default:
        return nil, fmt.Errorf("unsupported proof method: %s", request.ProofMethod)
    }
}

// verifyOIDCIDToken validates AgentAuth-issued ID token
func (pvp *OIDCPowerVerificationPoint) verifyOIDCIDToken(
    ctx context.Context,
    request *IdentityProofRequest,
) (*IdentityProofResult, error) {
    // Extract ID token from proof data
    idToken, ok := request.ProofData["id_token"].(string)
    if !ok {
        return &IdentityProofResult{
            Valid:         false,
            FailureReason: "missing id_token in proof data",
        }, nil
    }
    
    // Convert ID token to identity proof
    return pvp.identityBridge.ConvertIDTokenToIdentityProof(
        ctx,
        idToken,
        request.SubjectID,
    )
}

// verifyExternalOIDC validates ID token from external provider
func (pvp *OIDCPowerVerificationPoint) verifyExternalOIDC(
    ctx context.Context,
    request *IdentityProofRequest,
) (*IdentityProofResult, error) {
    // Extract provider name and ID token
    providerName, ok := request.ProofData["provider"].(string)
    if !ok {
        return &IdentityProofResult{
            Valid:         false,
            FailureReason: "missing provider in proof data",
        }, nil
    }
    
    idToken, ok := request.ProofData["id_token"].(string)
    if !ok {
        return &IdentityProofResult{
            Valid:         false,
            FailureReason: "missing id_token in proof data",
        }, nil
    }
    
    // Get provider client
    client, ok := pvp.providerClients[providerName]
    if !ok {
        return &IdentityProofResult{
            Valid:         false,
            FailureReason: fmt.Sprintf("unknown provider: %s", providerName),
        }, nil
    }
    
    // Verify ID token with provider
    claims, err := client.VerifyIDToken(ctx, idToken)
    if err != nil {
        return &IdentityProofResult{
            Valid:         false,
            FailureReason: fmt.Sprintf("ID token verification failed: %v", err),
        }, nil
    }
    
    // Convert to AgentAuth identity proof
    return pvp.identityBridge.ConvertIDTokenToIdentityProof(
        ctx,
        idToken,
        request.SubjectID,
    )
}
```

---

## Integration Strategy

### Integration with AAP-001 Subscription Flow

**Modified Subscription Flow** (Steps I-VIII):

```go
// Step I: Owner's Authorizer Identity Proof (WITH OIDC)
func (m *SubscriptionFlowManager) ExecuteStepI(
    ctx context.Context,
    subscriptionID string,
    request *IdentityProofRequest,
) error {
    // NEW: Support OIDC ID token as proof method
    if request.ProofMethod == "oidc_id_token" || request.ProofMethod == "oidc_external" {
        // Use OIDC-enabled PVP
        result, err := m.pvpClient.VerifyIdentityProof(ctx, request)
        if err != nil {
            return NewAgentAuthError(ErrCodeIdentityVerificationFailed, err.Error())
        }
        
        if !result.Valid {
            return NewAgentAuthError(ErrCodeInvalidIdentityProof, result.FailureReason)
        }
        
        // Store identity proof result
        return m.subscriptionStore.UpdateOwnersAuthorizerIdentity(
            ctx, subscriptionID, result,
        )
    }
    
    // FALLBACK: Existing proof methods (eIDAS, government_id, etc.)
    // ... existing implementation
}

// Step III: Client Owner Identity Proof (WITH OIDC)
// Similar OIDC integration...

// Step VI: Resource Owner Identity Proof (WITH OIDC)
// Similar OIDC integration...
```

### Provider Configuration

**Example Configuration** (`config/oidc_providers.yaml`):

```yaml
oidc:
  issuer_url: "https://agentauth.example.com"
  
  external_providers:
    - name: "google"
      issuer_url: "https://accounts.google.com"
      client_id: "your-google-client-id"
      client_secret: "your-google-client-secret"
      redirect_uri: "https://agentauth.example.com/callback/google"
      scopes:
        - "openid"
        - "profile"
        - "email"
    
    - name: "okta"
      issuer_url: "https://your-domain.okta.com"
      client_id: "your-okta-client-id"
      client_secret: "your-okta-client-secret"
      redirect_uri: "https://agentauth.example.com/callback/okta"
      scopes:
        - "openid"
        - "profile"
        - "email"
    
    - name: "azure_ad"
      issuer_url: "https://login.microsoftonline.com/{tenant-id}/v2.0"
      client_id: "your-azure-client-id"
      client_secret: "your-azure-client-secret"
      redirect_uri: "https://agentauth.example.com/callback/azure"
      scopes:
        - "openid"
        - "profile"
        - "email"
```

---

## Implementation Phases

### Phase 1: Core OIDC Infrastructure (Week 1)

**Deliverables**:
- ✅ OIDC Discovery Service (`pkg/oidc/discovery.go`)
- ✅ ID Token Service (`pkg/oidc/id_token.go`)
- ✅ Identity Bridge (`pkg/oidc/identity_bridge.go`)
- ✅ Unit tests for each component

**Acceptance Criteria**:
- Discovery endpoint returns valid OIDC configuration
- ID tokens can be issued and validated
- ID tokens convert correctly to `IdentityProofResult`

**Effort**: 3-4 days

---

### Phase 2: PowerVerificationPoint Integration (Week 2)

**Deliverables**:
- ✅ OIDC-enabled PVP (`pkg/agentauth/pvp_oidc.go`)
- ✅ Integration into subscription flow Steps I, III, VI
- ✅ Configuration loader for OIDC providers
- ✅ Integration tests

**Acceptance Criteria**:
- Subscription flow accepts OIDC ID tokens
- Identity verification works with OIDC proof method
- Backward compatibility maintained (existing proof methods still work)

**Effort**: 4-5 days

---

### Phase 3: External Provider Federation (Week 3)

**Deliverables**:
- ✅ External OIDC Provider Client (`pkg/oidc/provider_client.go`)
- ✅ Google, Okta, Azure AD integration
- ✅ OAuth 2.0 authorization code flow
- ✅ E2E tests with test providers

**Acceptance Criteria**:
- Can authenticate users via Google/Okta/Azure
- ID tokens from external providers validated correctly
- External identities map to AgentAuth subscription flow

**Effort**: 5-6 days

---

### Phase 4: Documentation & Production Hardening (Week 4)

**Deliverables**:
- ✅ API documentation (OpenAPI spec update)
- ✅ Integration guide for OIDC
- ✅ Security audit
- ✅ Performance testing
- ✅ Production deployment guide

**Acceptance Criteria**:
- Documentation complete and reviewed
- Security vulnerabilities addressed
- Performance benchmarks meet SLAs
- Production-ready configuration

**Effort**: 4-5 days

---

**Total Estimated Effort**: **3-4 weeks** (16-20 business days)

---

## Security Considerations

### 1. ID Token Validation

**Critical Checks**:
- ✅ Signature verification (RS256/HS256)
- ✅ Issuer validation (`iss` claim)
- ✅ Audience validation (`aud` claim)
- ✅ Expiration check (`exp` claim)
- ✅ Not-before check (`nbf` claim)
- ✅ Issued-at check (`iat` claim)
- ✅ Nonce validation (for replay protection)

**Implementation**:
```go
// All validation handled by jwt.ParseWithClaims + custom checks
token, err := jwt.ParseWithClaims(tokenString, claims, keyFunc)
if err != nil {
    return nil, err
}
if !token.Valid {
    return nil, errors.New("invalid token")
}
// Additional checks...
```

---

### 2. JWKS Key Rotation

**Strategy**:
- Cache public keys from external provider JWKS endpoints
- Refresh keys periodically (every 24 hours)
- Support multiple key IDs (kid) for rotation overlap
- Fallback to fresh JWKS fetch if key not found

**Implementation**:
```go
type JWKSCache struct {
    keys       map[string]*rsa.PublicKey
    jwksURL    string
    lastUpdate time.Time
    mutex      sync.RWMutex
}

func (c *JWKSCache) GetKey(kid string) (*rsa.PublicKey, error) {
    // Check cache
    // Refresh if expired
    // Fetch from JWKS endpoint
}
```

---

### 3. Client Secret Protection

**Best Practices**:
- Store client secrets in secure configuration (environment variables, secret manager)
- Use `client_secret_post` or `private_key_jwt` for client authentication
- Rotate client secrets periodically
- Never log client secrets

---

### 4. Nonce Management

**Purpose**: Prevent replay attacks on ID tokens.

**Implementation**:
```go
type NonceStore interface {
    GenerateNonce(ctx context.Context) (string, error)
    ValidateNonce(ctx context.Context, nonce string) (bool, error)
}

// Use Redis or similar for distributed nonce tracking
// Nonces expire after 15 minutes (ID token validity period)
```

---

### 5. ACR Requirements

**Enforce Trust Levels**:
```go
func (pvp *OIDCPowerVerificationPoint) VerifyIdentityProof(
    ctx context.Context,
    request *IdentityProofRequest,
) (*IdentityProofResult, error) {
    result, err := pvp.verifyOIDCIDToken(ctx, request)
    if err != nil {
        return nil, err
    }
    
    // Enforce minimum trust level
    requiredLevel := request.RequiredLevel
    if !meetsRequirement(result.TrustLevel, requiredLevel) {
        return &IdentityProofResult{
            Valid:         false,
            FailureReason: fmt.Sprintf("insufficient trust level: got %s, need %s",
                result.TrustLevel, requiredLevel),
        }, nil
    }
    
    return result, nil
}
```

---

## Testing Strategy

### Unit Tests

**Coverage**: 80%+ for all OIDC components

**Test Cases**:
1. **Discovery Service**:
   - Returns valid OIDC configuration
   - Contains all required fields
   
2. **ID Token Service**:
   - Issues valid ID tokens
   - Validates signatures correctly
   - Rejects expired tokens
   - Rejects invalid audience
   - Rejects invalid issuer

3. **Identity Bridge**:
   - Converts ID tokens to identity proofs
   - Maps ACR to trust levels correctly
   - Handles missing claims gracefully

4. **Provider Client**:
   - Discovers provider configuration
   - Generates authorization URLs
   - Exchanges codes for tokens
   - Validates external ID tokens

---

### Integration Tests

**Test Scenarios**:
1. **AgentAuth OIDC Flow**:
   - User authenticates via AgentAuth OIDC
   - Receives ID token
   - Uses ID token in Step I/III/VI
   - Subscription flow completes

2. **External Provider Flow**:
   - User redirects to Google/Okta
   - Authenticates with provider
   - Returns with authorization code
   - Exchanges for ID token
   - Uses ID token in subscription flow

3. **Mixed Flow**:
   - Step I uses Google OIDC
   - Step III uses AgentAuth OIDC
   - Step VI uses Okta OIDC
   - Subscription completes with federated identities

---

### E2E Tests

**Test Infrastructure**:
- Mock OIDC provider (Keycloak test instance)
- Test client application
- Automated browser testing (Playwright)

**Test Flow**:
1. Start AgentAuth server with OIDC enabled
2. User initiates subscription flow
3. System redirects to mock OIDC provider
4. User authenticates (automated)
5. System receives ID token
6. Subscription flow uses ID token for identity verification
7. Verify complete subscription created
8. Verify authorization chain includes OIDC identity

---

## Migration Path

### Backward Compatibility

**Strategy**: Support both OIDC and legacy proof methods simultaneously.

**Implementation**:
```go
// IdentityProofRequest.ProofMethod can be:
// - "eIDAS" (legacy)
// - "government_id" (legacy)
// - "commercial_register" (legacy)
// - "oidc_id_token" (NEW)
// - "oidc_external" (NEW)

// PowerVerificationPoint router
func (pvp *OIDCPowerVerificationPoint) VerifyIdentityProof(
    ctx context.Context,
    request *IdentityProofRequest,
) (*IdentityProofResult, error) {
    switch request.ProofMethod {
    case "oidc_id_token", "oidc_external":
        return pvp.verifyOIDC(ctx, request)
    default:
        return pvp.legacyPVP.VerifyIdentityProof(ctx, request)
    }
}
```

---

### Gradual Rollout

**Phase 1**: OIDC optional (default: legacy)
```yaml
oidc:
  enabled: false  # Opt-in
```

**Phase 2**: OIDC recommended
```yaml
oidc:
  enabled: true
  fallback_to_legacy: true  # Backward compatibility
```

**Phase 3**: OIDC required (3-6 months after Phase 2)
```yaml
oidc:
  enabled: true
  fallback_to_legacy: false
```

---

## References

### OIDC Specifications

1. **OpenID Connect Core 1.0**  
   https://openid.net/specs/openid-connect-core-1_0.html

2. **OpenID Connect Discovery 1.0**  
   https://openid.net/specs/openid-connect-discovery-1_0.html

3. **OpenID Connect Dynamic Client Registration 1.0**  
   https://openid.net/specs/openid-connect-registration-1_0.html

4. **OpenID Connect Session Management 1.0**  
   https://openid.net/specs/openid-connect-session-1_0.html

### OAuth 2.0 Specifications

1. **RFC 6749 - The OAuth 2.0 Authorization Framework**  
   https://datatracker.ietf.org/doc/html/rfc6749

2. **RFC 7636 - Proof Key for Code Exchange (PKCE)**  
   https://datatracker.ietf.org/doc/html/rfc7636

### Security Best Practices

1. **OAuth 2.0 Security Best Current Practice**  
   https://datatracker.ietf.org/doc/html/draft-ietf-oauth-security-topics

2. **OpenID Connect Security Considerations**  
   https://openid.net/specs/openid-connect-core-1_0.html#Security

### Libraries

1. **golang-jwt/jwt** (JWT implementation)  
   https://github.com/golang-jwt/jwt

2. **coreos/go-oidc** (OIDC client library)  
   https://github.com/coreos/go-oidc

---

## Next Steps

### Immediate Actions

1. **Review & Approval** - Stakeholder review of this design document
2. **Dependency Audit** - Check all OIDC libraries for vulnerabilities
3. **Development Environment** - Set up OIDC test infrastructure (Keycloak)
4. **Resource Allocation** - Assign development team (1-2 engineers)

### Success Metrics

- ✅ OIDC Discovery endpoint operational
- ✅ ID tokens issued and validated
- ✅ External provider federation working (Google/Okta/Azure)
- ✅ AAP-001 compliance increased: 62% → 68% (+6%)
- ✅ All tests passing (unit, integration, E2E)
- ✅ Documentation complete
- ✅ Security audit passed

---

**Document Status**: Ready for Implementation  
**Next Review**: After Phase 1 completion (Week 1)  
**Owner**: AgentAuth Development Team  
**Stakeholders**: AAP-001 Compliance Team, Security Team, Architecture Team
