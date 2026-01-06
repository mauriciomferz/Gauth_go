package auth

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TrustedIssuer represents a configured external Identity Provider (e.g., Azure AD)
// that is trusted to issue identity assertions for OBO flows.
type TrustedIssuer struct {
	// Issuer URL exactly as it appears in the 'iss' claim (e.g., https://sts.windows.net/tenant-id/)
	Issuer string `json:"issuer"`

	// JWKSURI is the endpoint to fetch public keys (e.g., https://login.microsoftonline.com/common/discovery/v2.0/keys)
	JWKSURI string `json:"jwks_uri"`

	// Audience is the required 'aud' claim value for tokens accepted from this issuer
	Audience string `json:"audience"`

	// ClaimsMapping allows remapping external claims to internal context
	// Key: External Claim Name (e.g., "oid"), Value: Internal Context Key (e.g., "subject_id")
	ClaimsMapping map[string]string `json:"claims_mapping"`

	// CacheTTL for JWKS keys
	CacheTTL time.Duration `json:"-"`

	jwksClient *JWKSClient
	once       sync.Once
}

// IssuerRegistry manages the set of TrustedIssuer configurations.
type IssuerRegistry struct {
	mu      sync.RWMutex
	issuers map[string]*TrustedIssuer
}

// NewIssuerRegistry creates a new registry.
func NewIssuerRegistry() *IssuerRegistry {
	return &IssuerRegistry{
		issuers: make(map[string]*TrustedIssuer),
	}
}

// AddRegister registers a trusted issuer.
func (r *IssuerRegistry) Register(issuer *TrustedIssuer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if issuer.CacheTTL == 0 {
		issuer.CacheTTL = 1 * time.Hour // Default 1 hour JWKS cache
	}
	r.issuers[issuer.Issuer] = issuer
}

// Get returns a trusted issuer by its issuer URL.
func (r *IssuerRegistry) Get(issuerURL string) (*TrustedIssuer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if issuer, ok := r.issuers[issuerURL]; ok {
		return issuer, nil
	}
	return nil, fmt.Errorf("issuer not trusted: %s", issuerURL)
}

// GetKey retrieves a key from the issuer's JWKS (dynamic fetching).
func (i *TrustedIssuer) GetKey(ctx context.Context, kid string) (interface{}, error) {
	i.initClient()
	return i.jwksClient.GetKey(ctx, kid)
}

func (i *TrustedIssuer) initClient() {
	i.once.Do(func() {
		i.jwksClient = NewJWKSClient(i.JWKSURI, i.CacheTTL)
	})
}

// GlobalRegistry is the default singleton registry.
var GlobalRegistry = NewIssuerRegistry()

// ContextKey for passing issuer information in request context
type ContextKey string

const IssuerContextKey ContextKey = "trusted_issuer"
