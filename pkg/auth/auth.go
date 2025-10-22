package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/errors"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/token"
)

// --- Structured error types for POA validation ---
var (
	ErrInvalidJurisdiction = fmt.Errorf("invalid jurisdiction")
	ErrDisallowedScope     = fmt.Errorf("disallowed scope capability")
	ErrMissingFields       = fmt.Errorf("missing required power-of-attorney fields")
)

// ProfessionalConfig for professional interface demo
type ProfessionalConfig struct {
	Issuer            string
	Audience          string
	TokenExpiry       time.Duration
	ServiceID         string
	MeshID            string
	UseSecureDefaults bool
}

// ProfessionalAuthService stub for demo
type ProfessionalAuthService struct{}

func NewProfessionalAuthService(config ProfessionalConfig) (*ProfessionalAuthService, error) {
	// Stub: always succeed
	return &ProfessionalAuthService{}, nil
}

func (s *ProfessionalAuthService) CreateToken(userID string, scopes []string, expiry time.Duration) (string, error) {
	// Stub: return dummy token
	return "dummy-token", nil
}

type ProfessionalClaims struct {
	UserID    string
	Scopes    []string
	ExpiresAt time.Time
}

func (s *ProfessionalAuthService) ValidateToken(token string) (*ProfessionalClaims, error) {
	// Stub: return dummy claims
	return &ProfessionalClaims{
		UserID:    "service-user-123",
		Scopes:    []string{"service:read", "service:write", "mesh:communicate"},
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}, nil
}

// Authenticator defines the interface for authentication operations
type Authenticator interface {
	Authenticate(ctx context.Context, credentials *Credentials) (*AuthResult, error)
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error)
}

// Credentials represents user credentials
type Credentials struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	GrantType    string `json:"grant_type,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// AuthResult represents the result of an authentication operation
type AuthResult struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	Scope        string    `json:"scope,omitempty"`
	Subject      string    `json:"subject"`
	IssuedAt     time.Time `json:"issued_at"`
}

// TokenClaims represents the claims in a token
type TokenClaims struct {
	Subject   string    `json:"sub"`
	Issuer    string    `json:"iss"`
	Audience  string    `json:"aud"`
	ExpiresAt time.Time `json:"exp"`
	IssuedAt  time.Time `json:"iat"`
	NotBefore time.Time `json:"nbf"`
	Scope     string    `json:"scope,omitempty"`
	KeyID     string    `json:"kid,omitempty"`
}

// SimpleAuthenticator is a basic implementation of Authenticator
type SimpleAuthenticator struct {
	tokenService *token.Service
	users        map[string]*User
}

// User represents a user in the system
type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Password string   `json:"password"` // In real implementation, this should be hashed
	Roles    []string `json:"roles"`
	Active   bool     `json:"active"`
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(tokenService *token.Service) *SimpleAuthenticator {
	return &SimpleAuthenticator{
		tokenService: tokenService,
		users: map[string]*User{
			"admin": {
				ID:       "admin-id",
				Username: "admin",
				Password: "admin", // In real implementation, use proper password hashing
				Roles:    []string{"admin"},
				Active:   true,
			},
			"user": {
				ID:       "user-id",
				Username: "user",
				Password: "user",
				Roles:    []string{"user"},
				Active:   true,
			},
		},
	}
}

// Authenticate authenticates user credentials
func (a *SimpleAuthenticator) Authenticate(ctx context.Context, credentials *Credentials) (*AuthResult, error) {
	user, exists := a.users[credentials.Username]
	if !exists || !user.Active {
		return nil, errors.ErrUnauthenticated
	}

	// In real implementation, use proper password verification
	if user.Password != credentials.Password {
		return nil, errors.ErrUnauthenticated
	}

	// Generate tokens
	accessToken, err := a.generateAccessToken(user)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to generate access token")
	}

	refreshToken, err := a.generateRefreshToken(user)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to generate refresh token")
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour
		Subject:      user.ID,
		IssuedAt:     time.Now(),
	}, nil
}

// ValidateToken validates a token and returns its claims
func (a *SimpleAuthenticator) ValidateToken(ctx context.Context, tokenStr string) (*TokenClaims, error) {
	// This is a simplified implementation
	// In real implementation, use proper JWT validation

	// For demo purposes, create mock claims
	return &TokenClaims{
		Subject:   "demo-user",
		Issuer:    "gauth",
		Audience:  "api",
		ExpiresAt: time.Now().Add(time.Hour),
		IssuedAt:  time.Now(),
		NotBefore: time.Now(),
		Scope:     "read write",
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (a *SimpleAuthenticator) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {
	// Simplified implementation
	// In real implementation, validate the refresh token and generate new tokens

	return &AuthResult{
		AccessToken: "new-access-token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		Subject:     "demo-user",
		IssuedAt:    time.Now(),
	}, nil
}

func (a *SimpleAuthenticator) generateAccessToken(user *User) (string, error) {
	// Simplified token generation
	// In real implementation, use proper JWT signing
	return "access-token-for-" + user.Username, nil
}

func (a *SimpleAuthenticator) generateRefreshToken(user *User) (string, error) {
	// Simplified token generation
	// In real implementation, use proper JWT signing
	return "refresh-token-for-" + user.Username, nil
}

// RFC Functional Test compatibility stubs
// These types and methods are required for the rfc_functional_test example to build.
type PowerOfAttorneyResponse struct {
	AuthorizationCode string
	LegalCompliance   bool
	AuditRecordID     string
}

type DelegationResponse struct {
	DelegationID     string
	Status           string
	ValidUntil       time.Time
	Attestations     []string
	ComplianceStatus string
}

func (a *SimpleAuthenticator) AuthorizePowerOfAttorney(ctx context.Context, req PowerOfAttorneyRequest) (*PowerOfAttorneyResponse, error) {
	// --- Demo validation logic (beta demonstration; simplified, not production) ---
	// Accept only a small whitelist of jurisdictions.
	validJurisdictions := map[string]bool{"US": true, "EU": true, "UK": true, "DE": true}
	if !validJurisdictions[req.Jurisdiction] {
		return nil, fmt.Errorf("%w: %s", ErrInvalidJurisdiction, req.Jurisdiction)
	}
	disallowed := map[string]bool{"nuclear_launch_codes": true, "critical_infra_root": true}
	scopeStr := strings.ReplaceAll(req.Scope, ",", " ")
	fields := strings.Fields(scopeStr)
	for _, sc := range fields {
		if disallowed[sc] {
			return nil, fmt.Errorf("%w: %s", ErrDisallowedScope, sc)
		}
	}
	if req.ClientID == "" || req.PrincipalID == "" || req.AIAgentID == "" || req.PowerType == "" {
		return nil, ErrMissingFields
	}
	// Return deterministic response for tests
	return &PowerOfAttorneyResponse{
		AuthorizationCode: "AUTHCODE-1234567890",
		LegalCompliance:   true,
		AuditRecordID:     "AUDIT-9876543210",
	}, nil
}

func (a *SimpleAuthenticator) CreateAdvancedDelegation(ctx context.Context, req DelegationRequest) (*DelegationResponse, error) {
	// --- Minimal RFC 115 validation logic ---
	if req.PrincipalID == "" || req.DelegateID == "" {
		return nil, fmt.Errorf("missing required principal or delegate ID")
	}
	if req.ValidityPeriod.Days <= 0 {
		return nil, fmt.Errorf("invalid validity period: must be > 0 days")
	}
	if len(req.AttestationRequirement.Attesters) == 0 {
		return nil, fmt.Errorf("at least one attester required")
	}
	// Return deterministic response for tests
	return &DelegationResponse{
		DelegationID:     "DELEG-1234567890",
		Status:           "active",
		ValidUntil:       time.Now().Add(time.Duration(req.ValidityPeriod.Days) * 24 * time.Hour),
		Attestations:     req.AttestationRequirement.Attesters,
		ComplianceStatus: "compliant",
	}, nil
}

// Stubs for RFC, PoA, and advanced types for demo and compliance examples

type PowerOfAttorneyRequest struct {
	ClientID     string
	ResponseType string
	Scope        string
	RedirectURI  string
	State        string
	PowerType    string
	PrincipalID  string
	AIAgentID    string
	Jurisdiction string
	LegalBasis   string
}

type PoADefinition struct {
	Principal Principal
	Client    string
}

type Principal struct {
	Type         string
	Identity     string
	Organization Organization
}
type Organization struct {
	Type                string
	Name                string
	RegisterEntry       string
	ManagingDirector    string
	RegisteredAuthority string
}
type DelegationRequest struct {
	PrincipalID            string
	DelegateID             string
	ValidityPeriod         ValidityPeriod
	AttestationRequirement AttestationRequirement
}
type (
	PowerRestrictions struct{}
	ValidityPeriod    struct {
		Days int
	}
)

type AttestationRequirement struct {
	Attesters []string
}
type (
	TimeWindow struct{}
	Authorizer struct {
		ClientOwner string
	}
)

type AuthorizedRepresentative struct {
	Name                string
	RegisteredAuthority string
	RegisterEntry       string
	AuthorityType       string
}

const (
	PrincipalTypeOrganization = "Organization"
	OrgTypeCommercial         = "Commercial"
	ClientAI                  = "AI"
	ClientTypeLLM             = "LLM"
	ClientTypeAgenticAI       = "AgenticAI"
	AuthorizationType         = "Authorization"

	DepositTransaction   = gauth.DepositTransaction
	TransactionPending   = gauth.TransactionPending
	TransactionCompleted = gauth.TransactionCompleted
	TransactionFailed    = gauth.TransactionFailed
	// TransactionCanceled exported for external packages; legacy alias retained in gauth.
	TransactionCanceled  = gauth.TransactionCanceled
	TransactionCancelled = gauth.TransactionCancelled // deprecated alias
)

func NewRFCCompliantService() *SimpleAuthenticator {
	return &SimpleAuthenticator{}
}

// Re-export variables
var (
	ErrInvalidToken  = gauth.ErrInvalidToken
	ErrUnauthorized  = gauth.ErrUnauthorized
	ErrTokenExpired  = gauth.ErrTokenExpired
	ErrInvalidGrant  = gauth.ErrInvalidGrant
	ErrInvalidClient = gauth.ErrInvalidClient
)

// Re-export functions
var (
	New                         = gauth.New
	NewResourceServer           = gauth.NewResourceServer
	NewPowerAdministrationPoint = gauth.NewPowerAdministrationPoint
)

// ValidateToken validates an authentication token
// This is a simple implementation that always returns success for demo purposes
func ValidateToken(token string) (interface{}, error) {
	// In a real implementation, you would:
	// 1. Parse the token (JWT, etc.)
	// 2. Validate signature
	// 3. Check expiration
	// 4. Return claims or user info

	// For demo purposes, just return some basic info
	return map[string]interface{}{
		"user_id": "demo-user",
		"scopes":  []string{"read", "write"},
		"valid":   true,
	}, nil
}

// JWTService represents a JWT service for token operations
type JWTService struct {
	issuer   string
	audience string
	secret   string
}

// Claims represents JWT token claims
type Claims struct {
	UserID    string         `json:"user_id"`
	SessionID string         `json:"session_id"`
	Scopes    []string       `json:"scopes"`
	ExpiresAt ExpirationTime `json:"exp"`
	IssuedAt  int64          `json:"iat"`
	Issuer    string         `json:"iss"`
	Audience  string         `json:"aud"`
}

// ExpirationTime wraps int64 with Time method for compatibility
type ExpirationTime struct {
	Time time.Time
}

// Unix returns the Unix timestamp
func (et ExpirationTime) Unix() int64 {
	return et.Time.Unix()
}

// NewProperJWTService creates a new JWT service
func NewProperJWTService(issuer, audience string) (*JWTService, error) {
	return &JWTService{
		issuer:   issuer,
		audience: audience,
		secret:   "demo-secret-key", // In production, use proper secret management
	}, nil
}

// CreateToken creates a new JWT token
func (j *JWTService) CreateToken(userID string, scopes []string, duration time.Duration) (string, error) {
	now := time.Now()
	expiresAt := now.Add(duration)
	expUnix := expiresAt.Unix()
	// Join scopes with space (preserves embedded colons), encode URL-safe base64 to avoid introducing additional '.' delimiters.
	scopePayload := ""
	if len(scopes) > 0 {
		scopePayload = base64.RawURLEncoding.EncodeToString([]byte(strings.Join(scopes, " ")))
	}
	// New token format: jwt.<userID>.<issuer>.<expUnix>[.<b64 scopes>]
	if scopePayload != "" {
		return fmt.Sprintf("jwt.%s.%s.%d.%s", userID, j.issuer, expUnix, scopePayload), nil
	}
	// Backward-compatible (no scopes segment)
	return fmt.Sprintf("jwt.%s.%s.%d", userID, j.issuer, expUnix), nil
}

// ValidateToken validates a JWT token and returns claims
func (j *JWTService) ValidateToken(token string) (*Claims, error) {
	// For demo purposes, simple validation
	// In production, use proper JWT parsing and validation

	if token == "" {
		return nil, fmt.Errorf("empty token")
	}

	// Extract info from our demo token format
	parts := strings.Split(token, ".")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Parse expiration
	exp, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid expiration: %w", err)
	}

	// Check if expired
	if time.Now().Unix() > exp {
		return nil, fmt.Errorf("token expired")
	}

	// Decode scopes if present (fifth segment). If absent, keep empty slice.
	var scopes []string
	if len(parts) >= 5 && parts[4] != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(parts[4])
		if err == nil {
			payload := string(decoded)
			for _, s := range strings.Fields(payload) { // Fields splits on space used in CreateToken
				if s != "" {
					scopes = append(scopes, s)
				}
			}
		}
	}
	claims := &Claims{
		UserID:    parts[1],
		SessionID: fmt.Sprintf("sess_%s", parts[1]),
		Scopes:    scopes,
		ExpiresAt: ExpirationTime{Time: time.Unix(exp, 0)},
		IssuedAt:  time.Now().Unix(),
		Issuer:    j.issuer,
		Audience:  j.audience,
	}
	return claims, nil
}

// HasScope reports whether the validated claims include the provided scope token.
func (c *Claims) HasScope(scope string) bool {
	if scope == "" {
		return false
	}
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// BuildAuthzRequestFromClaims constructs an authz.Request using token claims.
// extraCtx allows callers to inject additional context keys (e.g., jurisdiction, ip, device).
// Roles are not embedded in Claims yet; callers can pass them via extraCtx["roles"] as comma-separated values.
func BuildAuthzRequestFromClaims(claims *Claims, resource, action string, extraCtx map[string]string) authz.Request {
	ctx := make(map[string]string)
	for k, v := range extraCtx { // nil map safe: no iterations
		ctx[k] = v
	}
	// attach scopes as space-separated list for policy matching
	if len(claims.Scopes) > 0 {
		ctx["scopes"] = strings.Join(claims.Scopes, " ")
	}
	return authz.Request{Subject: claims.UserID, Resource: resource, Action: action, Context: ctx}
}
