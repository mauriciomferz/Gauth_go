// Package gnap implements RFC 9635 Grant Negotiation and Authorization Protocol.
// This provides modern authorization with grant negotiation, flexible interaction
// modes, and token management, extended with AgentAuth's Power of Attorney support.
package gnap

import (
	"crypto"
	"encoding/json"
	"time"
)

// GrantState represents the current state of a grant request.
type GrantState string

const (
	GrantStateProcessing GrantState = "processing" // Grant request is being processed
	GrantStatePending    GrantState = "pending"    // Waiting for interaction
	GrantStateApproved   GrantState = "approved"   // Grant approved, tokens available
	GrantStateFinalized  GrantState = "finalized"  // Grant complete, no more changes
	GrantStateDenied     GrantState = "denied"     // Grant denied
)

// GrantRequest represents a client's request for authorization (RFC 9635 §2).
type GrantRequest struct {
	// AccessToken describes the access being requested (§2.1)
	AccessToken *AccessTokenRequest `json:"access_token,omitempty"`

	// Subject describes information about the end-user (§2.2)
	Subject *SubjectRequest `json:"subject,omitempty"`

	// Client identifies the client instance (§2.3)
	Client *ClientInstance `json:"client,omitempty"`

	// User identifies the end-user (§2.4)
	User *UserInfo `json:"user,omitempty"`

	// Interact describes interaction capabilities (§2.5)
	Interact *InteractionRequest `json:"interact,omitempty"`

	// --- AgentAuth Extensions ---

	// PoACredentialRef references a Power of Attorney credential
	PoACredentialRef string `json:"poa_credential_ref,omitempty"`

	// SubscriptionID links to existing AgentAuth subscription (migration path)
	SubscriptionID string `json:"subscription_id,omitempty"`
}

// AccessTokenRequest describes requested access token parameters (§2.1).
type AccessTokenRequest struct {
	// Access describes the rights requested
	Access []AccessRight `json:"access,omitempty"`

	// Label for this token (when requesting multiple)
	Label string `json:"label,omitempty"`

	// Flags for token behavior
	Flags []TokenFlag `json:"flags,omitempty"`
}

// AccessRight describes a single access right (§8).
type AccessRight struct {
	// Type of resource (string reference)
	Type string `json:"type,omitempty"`

	// Actions permitted
	Actions []string `json:"actions,omitempty"`

	// Locations (URIs) where access applies
	Locations []string `json:"locations,omitempty"`

	// DataTypes that can be accessed
	DataTypes []string `json:"datatypes,omitempty"`

	// Identifier for specific resource
	Identifier string `json:"identifier,omitempty"`

	// Privileges requested
	Privileges []string `json:"privileges,omitempty"`
}

// TokenFlag represents access token flags (§2.1.1).
type TokenFlag string

const (
	TokenFlagBearer  TokenFlag = "bearer"  // Token can be used as bearer
	TokenFlagDurable TokenFlag = "durable" // Token survives rotation
)

// SubjectRequest describes requested subject information (§2.2).
type SubjectRequest struct {
	// SubIDs requested (RFC 9493 Subject Identifiers)
	SubIDs []string `json:"sub_ids,omitempty"`

	// Assertions requested
	Assertions []string `json:"assertions,omitempty"`
}

// ClientInstance identifies a client instance (§2.3).
type ClientInstance struct {
	// Key for this client instance
	Key *ClientKey `json:"key,omitempty"`

	// ClassID for client software class
	ClassID string `json:"class_id,omitempty"`

	// Display information
	Display *ClientDisplay `json:"display,omitempty"`

	// InstanceID assigned by AS (returned in response)
	InstanceID string `json:"instance_id,omitempty"`
}

// ClientKey describes the client's key material (§2.3.1).
type ClientKey struct {
	// Proof method for key binding
	Proof ProofMethod `json:"proof"`

	// JWK for the key (inline)
	JWK json.RawMessage `json:"jwk,omitempty"`

	// JWKS URI for key retrieval
	JWKS string `json:"jwks,omitempty"`

	// Cert for certificate chain
	Cert string `json:"cert,omitempty"`

	// CertS256 thumbprint
	CertS256 string `json:"cert#S256,omitempty"`
}

// ProofMethod identifies key proofing mechanism.
type ProofMethod string

const (
	ProofHTTPSig ProofMethod = "httpsig" // RFC 9421 HTTP Message Signatures
	ProofMTLS    ProofMethod = "mtls"    // Mutual TLS
	ProofJWSD    ProofMethod = "jwsd"    // Detached JWS
	ProofDPoP    ProofMethod = "dpop"    // DPoP (RFC 9449)
)

// ClientDisplay provides human-readable client info (§2.3.2).
type ClientDisplay struct {
	Name string `json:"name,omitempty"`
	URI  string `json:"uri,omitempty"`
	Logo string `json:"logo_uri,omitempty"`
}

// UserInfo identifies the end-user (§2.4).
type UserInfo struct {
	// SubIDs for the user
	SubIDs []SubjectID `json:"sub_ids,omitempty"`

	// Assertions about the user
	Assertions []string `json:"assertions,omitempty"`
}

// SubjectID represents a subject identifier (RFC 9493).
type SubjectID struct {
	Format string `json:"format"`
	Email  string `json:"email,omitempty"`
	Phone  string `json:"phone,omitempty"`
	Issuer string `json:"iss,omitempty"`
	Sub    string `json:"sub,omitempty"`
}

// InteractionRequest describes interaction capabilities (§2.5).
type InteractionRequest struct {
	// Start modes the client can initiate
	Start []InteractionStartMode `json:"start,omitempty"`

	// Finish describes how client receives interaction result
	Finish *InteractionFinish `json:"finish,omitempty"`

	// Hints for interaction preferences
	Hints *InteractionHints `json:"hints,omitempty"`
}

// InteractionStartMode identifies how to start interaction.
type InteractionStartMode string

const (
	InteractionStartRedirect    InteractionStartMode = "redirect"      // Browser redirect
	InteractionStartApp         InteractionStartMode = "app"           // Native app launch
	InteractionStartUserCode    InteractionStartMode = "user_code"     // Short code display
	InteractionStartUserCodeURI InteractionStartMode = "user_code_uri" // Code + URI
)

// InteractionFinish describes how to complete interaction (§2.5.2).
type InteractionFinish struct {
	// Method for finish callback
	Method InteractionFinishMethod `json:"method"`

	// URI to call back to
	URI string `json:"uri"`

	// Nonce for hash calculation
	Nonce string `json:"nonce"`

	// HashMethod for interaction hash (default SHA-256)
	HashMethod string `json:"hash_method,omitempty"`
}

// InteractionFinishMethod identifies finish callback type.
type InteractionFinishMethod string

const (
	InteractionFinishRedirect InteractionFinishMethod = "redirect" // Browser redirect
	InteractionFinishPush     InteractionFinishMethod = "push"     // Direct POST
)

// InteractionHints provides preferences (§2.5.3).
type InteractionHints struct {
	UILocales []string `json:"ui_locales,omitempty"`
}

// GrantResponse is the AS response to a grant request (§3).
type GrantResponse struct {
	// Continue provides continuation information (§3.1)
	Continue *ContinuationInfo `json:"continue,omitempty"`

	// AccessToken is the issued token (§3.2)
	AccessToken *AccessToken `json:"access_token,omitempty"`

	// Interact provides interaction instructions (§3.3)
	Interact *InteractionResponse `json:"interact,omitempty"`

	// Subject provides subject information (§3.4)
	Subject *SubjectInfo `json:"subject,omitempty"`

	// InstanceID assigned to client (§3.5)
	InstanceID string `json:"instance_id,omitempty"`

	// Error if request failed (§3.6)
	Error *GrantError `json:"error,omitempty"`

	// --- AgentAuth Extensions ---

	// PowerOfAttorney embedded in response
	PowerOfAttorney *PowerOfAttorneyRef `json:"power_of_attorney,omitempty"`

	// AuthorizationChain for delegation proof
	AuthorizationChain []ChainLink `json:"authorization_chain,omitempty"`

	// ComplianceLevel achieved
	ComplianceLevel string `json:"compliance_level,omitempty"`
}

// ContinuationInfo for multi-step grants (§3.1).
type ContinuationInfo struct {
	// URI for continuation requests
	URI string `json:"uri"`

	// AccessToken for authorization
	AccessToken *ContinuationToken `json:"access_token"`

	// Wait seconds before polling
	Wait int `json:"wait,omitempty"`
}

// ContinuationToken for accessing continuation endpoint.
type ContinuationToken struct {
	Value string `json:"value"`
}

// AccessToken represents an issued access token (§3.2).
type AccessToken struct {
	// Value is the token string
	Value string `json:"value"`

	// Label if multiple tokens
	Label string `json:"label,omitempty"`

	// Manage provides token management endpoint
	Manage *TokenManagement `json:"manage,omitempty"`

	// Access rights granted
	Access []AccessRight `json:"access,omitempty"`

	// ExpiresIn seconds until expiration
	ExpiresIn int64 `json:"expires_in,omitempty"`

	// Key bound to this token (if different from client key)
	Key *ClientKey `json:"key,omitempty"`

	// Flags for this token
	Flags []TokenFlag `json:"flags,omitempty"`

	// --- AgentAuth Extensions ---

	// PoAID references embedded Power of Attorney
	PoAID string `json:"poa_id,omitempty"`

	// IssuedAt timestamp
	IssuedAt time.Time `json:"issued_at,omitempty"`
}

// TokenManagement endpoint info (§3.2.1).
type TokenManagement struct {
	URI         string             `json:"uri"`
	AccessToken *ContinuationToken `json:"access_token,omitempty"`
}

// InteractionResponse from AS (§3.3).
type InteractionResponse struct {
	// Redirect URI for redirect mode
	Redirect string `json:"redirect,omitempty"`

	// App URI for app mode
	App string `json:"app,omitempty"`

	// UserCode for code modes
	UserCode string `json:"user_code,omitempty"`

	// UserCodeURI for user_code_uri mode
	UserCodeURI *UserCodeURIResponse `json:"user_code_uri,omitempty"`

	// Finish nonce for hash calculation
	Finish string `json:"finish,omitempty"`

	// ExpiresIn seconds for interaction timeout
	ExpiresIn int `json:"expires_in,omitempty"`
}

// UserCodeURIResponse for device-style flow.
type UserCodeURIResponse struct {
	Code string `json:"code"`
	URI  string `json:"uri"`
}

// SubjectInfo returned by AS (§3.4).
type SubjectInfo struct {
	SubIDs     []SubjectID `json:"sub_ids,omitempty"`
	Assertions []string    `json:"assertions,omitempty"`
	UpdatedAt  time.Time   `json:"updated_at,omitempty"`
}

// GrantError represents an error response (§3.6).
type GrantError struct {
	Code        string `json:"error"`
	Description string `json:"error_description,omitempty"`
}

// Error codes (§3.6).
const (
	ErrorInvalidRequest     = "invalid_request"
	ErrorInvalidClient      = "invalid_client"
	ErrorInvalidInteraction = "invalid_interaction"
	ErrorInvalidFlag        = "invalid_flag"
	ErrorUnknownUser        = "unknown_user"
	ErrorUnknownRequest     = "unknown_request"
	ErrorUserDenied         = "user_denied"
	ErrorRequestDenied      = "request_denied"
	ErrorTooFast            = "too_fast"
	ErrorTooSlow            = "too_slow"
)

// PowerOfAttorneyRef references AgentAuth PoA in GNAP context.
type PowerOfAttorneyRef struct {
	PoAID   string `json:"poa_id"`
	Issuer  string `json:"issuer"`
	Grantee string `json:"grantee"`
	Scope   any    `json:"scope,omitempty"`
}

// ChainLink represents one hop in authorization chain.
type ChainLink struct {
	Entity     string `json:"entity"`
	EntityType string `json:"entity_type,omitempty"` // "human", "ai_agent"
	Authority  string `json:"authority"`
	Verified   bool   `json:"verified"`
}

// --- Key Material Helpers ---

// Signer interface for signing requests.
type Signer interface {
	Sign(payload []byte) ([]byte, error)
	KeyID() string
	Algorithm() string
	PublicKey() crypto.PublicKey
}

// Verifier interface for verifying signatures.
type Verifier interface {
	Verify(payload, signature []byte, keyID string) error
}

// --- RFC 9767 Resource Server Extensions ---

// ResourceServerRequest for dynamic registration (RFC 9767).
type ResourceServerRequest struct {
	// Client details (RS acts as client to AS)
	Client *ClientInstance `json:"client"`

	// Resources managed by this RS
	Resources []string `json:"resource_uris"`

	// Name for display
	Name string `json:"name"`
}

// ResourceServerResponse for registration response.
type ResourceServerResponse struct {
	// InstanceID assigned to the RS
	InstanceID string `json:"instance_id"`

	// Key bound to the RS
	Key *ClientKey `json:"key"`
}

// IntrospectionRequest per RFC 9767.
type IntrospectionRequest struct {
	// Token to introspect
	Token string `json:"token"`

	// Specific access rights to check (optional)
	Access []string `json:"access,omitempty"`
}

// IntrospectionResponse per RFC 9767.
type IntrospectionResponse struct {
	// Active status of the token
	Active bool `json:"active"`

	// Access rights granted associated with this token
	Access []AccessRight `json:"access,omitempty"`

	// PoA information (AgentAuth extension)
	PoA *PowerOfAttorneyRef `json:"poa,omitempty"`

	// Subject information associated with the token
	Subject *SubjectInfo `json:"subject,omitempty"`
}
