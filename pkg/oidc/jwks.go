package oidc

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWKSFetcher fetches and caches JSON Web Key Sets from OIDC providers
type JWKSFetcher interface {
	// GetKey retrieves a public key by key ID from a provider's JWKS endpoint
	GetKey(ctx context.Context, jwksURI, kid string) (interface{}, error)

	// RefreshKeys forces a refresh of the JWKS for a given URI
	RefreshKeys(ctx context.Context, jwksURI string) error

	// ClearCache clears all cached JWKS
	ClearCache()
}

// CachedJWKS represents a cached JWKS with metadata
type CachedJWKS struct {
	JWKS      *JWKS
	FetchedAt time.Time
	ExpiresAt time.Time
}

// InMemoryJWKSFetcher is an in-memory implementation of JWKSFetcher
type InMemoryJWKSFetcher struct {
	cache      map[string]*CachedJWKS
	mu         sync.RWMutex
	httpClient *http.Client
	cacheTTL   time.Duration
}

// NewInMemoryJWKSFetcher creates a new in-memory JWKS fetcher
func NewInMemoryJWKSFetcher(cacheTTL time.Duration) *InMemoryJWKSFetcher {
	if cacheTTL == 0 {
		cacheTTL = 24 * time.Hour // Default 24 hour cache
	}

	return &InMemoryJWKSFetcher{
		cache: make(map[string]*CachedJWKS),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cacheTTL: cacheTTL,
	}
}

// GetKey retrieves a public key by key ID from a provider's JWKS endpoint
func (f *InMemoryJWKSFetcher) GetKey(ctx context.Context, jwksURI, kid string) (interface{}, error) {
	// Check cache first
	jwks, err := f.getCachedJWKS(jwksURI)
	if err != nil {
		// Cache miss or expired, fetch from endpoint
		jwks, err = f.fetchJWKS(ctx, jwksURI)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
		}
		f.cacheJWKS(jwksURI, jwks)
	}

	// Find the key with matching kid
	for _, key := range jwks.Keys {
		if key.KID == kid {
			return f.parseJWK(&key)
		}
	}

	return nil, fmt.Errorf("key with kid %s not found in JWKS", kid)
}

// RefreshKeys forces a refresh of the JWKS for a given URI
func (f *InMemoryJWKSFetcher) RefreshKeys(ctx context.Context, jwksURI string) error {
	jwks, err := f.fetchJWKS(ctx, jwksURI)
	if err != nil {
		return fmt.Errorf("failed to refresh JWKS: %w", err)
	}

	f.cacheJWKS(jwksURI, jwks)
	return nil
}

// ClearCache clears all cached JWKS
func (f *InMemoryJWKSFetcher) ClearCache() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cache = make(map[string]*CachedJWKS)
}

// getCachedJWKS retrieves a cached JWKS if available and not expired
func (f *InMemoryJWKSFetcher) getCachedJWKS(jwksURI string) (*JWKS, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	cached, exists := f.cache[jwksURI]
	if !exists {
		return nil, fmt.Errorf("JWKS not found in cache")
	}

	if time.Now().After(cached.ExpiresAt) {
		return nil, fmt.Errorf("cached JWKS expired")
	}

	return cached.JWKS, nil
}

// fetchJWKS fetches a JWKS from the provider's endpoint
func (f *InMemoryJWKSFetcher) fetchJWKS(ctx context.Context, jwksURI string) (*JWKS, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", jwksURI, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var jwks JWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	return &jwks, nil
}

// cacheJWKS stores a JWKS in the cache
func (f *InMemoryJWKSFetcher) cacheJWKS(jwksURI string, jwks *JWKS) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	f.cache[jwksURI] = &CachedJWKS{
		JWKS:      jwks,
		FetchedAt: now,
		ExpiresAt: now.Add(f.cacheTTL),
	}
}

// parseJWK converts a JWKSKey to a crypto public key
func (f *InMemoryJWKSFetcher) parseJWK(jwk *JWKSKey) (interface{}, error) {
	switch jwk.Kty {
	case "RSA":
		return f.parseRSAKey(jwk)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}
}

// parseRSAKey converts a JWKSKey to an RSA public key
func (f *InMemoryJWKSFetcher) parseRSAKey(jwk *JWKSKey) (*rsa.PublicKey, error) {
	// Decode base64url encoded modulus
	nBytes, err := jwt.NewParser().DecodeSegment(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode base64url encoded exponent
	eBytes, err := jwt.NewParser().DecodeSegment(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert bytes to big.Int
	n := new(big.Int).SetBytes(nBytes)

	// Convert exponent bytes to int
	var e int
	if len(eBytes) > 4 {
		return nil, fmt.Errorf("exponent too large")
	}
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

// ExternalTokenValidator validates external OIDC tokens using JWKS
type ExternalTokenValidator struct {
	jwksFetcher    JWKSFetcher
	discoveryCache DiscoveryCache
}

// NewExternalTokenValidator creates a new external token validator
func NewExternalTokenValidator(jwksFetcher JWKSFetcher, discoveryCache DiscoveryCache) *ExternalTokenValidator {
	return &ExternalTokenValidator{
		jwksFetcher:    jwksFetcher,
		discoveryCache: discoveryCache,
	}
}

// ValidateToken validates an external OIDC token
func (v *ExternalTokenValidator) ValidateToken(
	ctx context.Context,
	tokenString, issuer, audience string,
) (*IDTokenClaims, error) {
	// Parse token without verification to get header
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &IDTokenClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Get kid from header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("token missing kid header")
	}

	// Get JWKS URI from discovery document
	doc, err := v.discoveryCache.Get(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to get discovery document: %w", err)
	}

	if doc.JWKSUri == "" {
		return nil, fmt.Errorf("discovery document missing jwks_uri")
	}

	// Get public key from JWKS
	publicKey, err := v.jwksFetcher.GetKey(ctx, doc.JWKSUri, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Parse and verify token with public key
	claims := &IDTokenClaims{}
	parsedToken, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	if !parsedToken.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	// Verify issuer
	if claims.Issuer != issuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", issuer, claims.Issuer)
	}

	// Verify audience
	audienceMatched := false
	for _, aud := range claims.Audience {
		if aud == audience {
			audienceMatched = true
			break
		}
	}
	if !audienceMatched {
		return nil, fmt.Errorf("invalid audience: token not intended for %s", audience)
	}

	// Verify expiration
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("token has expired")
	}

	// Verify not before
	if claims.NotBefore != nil && time.Now().Before(claims.NotBefore.Time) {
		return nil, fmt.Errorf("token not yet valid")
	}

	return claims, nil
}

// ValidateTokenForProvider validates a token for a specific provider configuration
func (v *ExternalTokenValidator) ValidateTokenForProvider(
	ctx context.Context,
	tokenString string,
	provider ProviderConfig,
) (*IDTokenClaims, error) {
	return v.ValidateToken(ctx, tokenString, provider.IssuerURL, provider.ClientID)
}
