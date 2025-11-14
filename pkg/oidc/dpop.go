package oidc

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RFC 9449: OAuth 2.0 Demonstrating Proof-of-Possession at the Application Layer (DPoP)
//
// DPoP enables sender-constrained access tokens by binding them to a cryptographic key pair.
// This prevents token theft and replay attacks since the token can only be used by the holder
// of the corresponding private key.
//
// Key Features:
// - DPoP proof JWT signed with client's private key
// - Access tokens bound to the DPoP key
// - Replay attack prevention via nonce and jti
// - Token type: "DPoP" instead of "Bearer"

// DPoPProof represents a DPoP proof JWT as defined in RFC 9449 Section 4.2.
type DPoPProof struct {
	// JWT Header (JOSE)
	Type      string          `json:"typ"` // Must be "dpop+jwt"
	Algorithm string          `json:"alg"` // RS256, ES256, or EdDSA
	JWK       json.RawMessage `json:"jwk"` // Public key in JWK format

	// JWT Claims
	JTI             string `json:"jti"`             // Unique identifier
	HTTPMethod      string `json:"htm"`             // HTTP method (e.g., "POST")
	HTTPURI         string `json:"htu"`             // HTTP URI (without query/fragment)
	IssuedAt        int64  `json:"iat"`             // Unix timestamp
	Nonce           string `json:"nonce,omitempty"` // Server-provided nonce (if required)
	AccessTokenHash string `json:"ath,omitempty"`   // Access token hash (for resource server)
}

// DPoPConfig configures DPoP validation behavior.
type DPoPConfig struct {
	// RequireNonce indicates whether nonce is required in DPoP proofs
	RequireNonce bool

	// MaxAge is the maximum age of a DPoP proof (default: 60 seconds)
	MaxAge time.Duration

	// NonceLifetime is how long a nonce is valid (default: 300 seconds)
	NonceLifetime time.Duration

	// AllowedAlgorithms specifies which signing algorithms are accepted
	// Default: RS256, ES256, EdDSA
	AllowedAlgorithms []string

	// ClockSkew allows some time difference for iat validation
	ClockSkew time.Duration
}

// DefaultDPoPConfig returns default DPoP configuration.
func DefaultDPoPConfig() *DPoPConfig {
	return &DPoPConfig{
		RequireNonce:      false,
		MaxAge:            60 * time.Second,
		NonceLifetime:     300 * time.Second,
		AllowedAlgorithms: []string{"RS256", "ES256", "EdDSA"},
		ClockSkew:         5 * time.Second,
	}
}

// DPoPManager manages DPoP proof validation and nonce generation.
type DPoPManager struct {
	config *DPoPConfig

	// nonces stores active nonces and their creation time
	nonces map[string]time.Time

	// usedJTIs stores used JTI values to prevent replay
	usedJTIs map[string]time.Time

	// dpopKeys stores DPoP public keys bound to access tokens
	// key: access token ID, value: JWK thumbprint
	dpopKeys map[string]string

	mu sync.RWMutex

	// cleanup goroutine
	stopCleanup chan struct{}
	wg          sync.WaitGroup
}

// NewDPoPManager creates a new DPoP manager.
func NewDPoPManager(config *DPoPConfig) *DPoPManager {
	if config == nil {
		config = DefaultDPoPConfig()
	}

	dm := &DPoPManager{
		config:      config,
		nonces:      make(map[string]time.Time),
		usedJTIs:    make(map[string]time.Time),
		dpopKeys:    make(map[string]string),
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	dm.wg.Add(1)
	go dm.cleanupLoop()

	return dm
}

// GenerateNonce generates a new nonce for DPoP proof challenge.
func (dm *DPoPManager) GenerateNonce() (string, error) {
	// Generate 32 bytes of randomness
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	nonce := base64.RawURLEncoding.EncodeToString(b)

	dm.mu.Lock()
	dm.nonces[nonce] = time.Now()
	dm.mu.Unlock()

	return nonce, nil
}

// ValidateDPoPProof validates a DPoP proof JWT.
//
// Parameters:
//   - ctx: Context for the operation
//   - dpopHeader: The "DPoP" HTTP header value (the JWT)
//   - httpMethod: The HTTP method of the request (e.g., "POST")
//   - httpURI: The HTTP URI without query/fragment
//   - accessToken: Optional access token for "ath" validation
//
// Returns:
//   - *DPoPProof: Validated proof with extracted claims
//   - string: JWK thumbprint (for token binding)
//   - error: Validation error
func (dm *DPoPManager) ValidateDPoPProof(ctx context.Context, dpopHeader, httpMethod, httpURI, accessToken string) (*DPoPProof, string, error) {
	if dpopHeader == "" {
		return nil, "", &OIDCError{
			ErrorCode:        ErrorInvalidRequest,
			ErrorDescription: "missing DPoP proof",
		}
	}

	// Parse JWT without verification first to extract header
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(dpopHeader, jwt.MapClaims{})
	if err != nil {
		return nil, "", &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: fmt.Sprintf("invalid DPoP JWT format: %v", err),
		}
	}

	// Validate JWT header
	if err := dm.validateDPoPHeader(token.Header); err != nil {
		return nil, "", err
	}

	// Extract public key from JWK
	jwkRaw, ok := token.Header["jwk"].(map[string]interface{})
	if !ok {
		return nil, "", &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: "missing or invalid 'jwk' in header",
		}
	}

	publicKey, err := parseJWKPublicKey(jwkRaw)
	if err != nil {
		return nil, "", &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: fmt.Sprintf("invalid JWK: %v", err),
		}
	}

	// Verify JWT signature
	token, err = parser.Parse(dpopHeader, func(t *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	if err != nil {
		return nil, "", &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: fmt.Sprintf("signature verification failed: %v", err),
		}
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, "", &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: "invalid claims format",
		}
	}

	// Build DPoPProof struct
	proof := &DPoPProof{}
	if typ, ok := token.Header["typ"].(string); ok {
		proof.Type = typ
	}
	if alg, ok := token.Header["alg"].(string); ok {
		proof.Algorithm = alg
	}

	// Extract string claims
	if jti, ok := claims["jti"].(string); ok {
		proof.JTI = jti
	}
	if htm, ok := claims["htm"].(string); ok {
		proof.HTTPMethod = htm
	}
	if htu, ok := claims["htu"].(string); ok {
		proof.HTTPURI = htu
	}
	if nonce, ok := claims["nonce"].(string); ok {
		proof.Nonce = nonce
	}
	if ath, ok := claims["ath"].(string); ok {
		proof.AccessTokenHash = ath
	}

	// Extract iat (issued at)
	if iat, ok := claims["iat"].(float64); ok {
		proof.IssuedAt = int64(iat)
	}

	// Validate claims
	if err := dm.validateDPoPClaims(proof, httpMethod, httpURI, accessToken); err != nil {
		return nil, "", err
	}

	// Calculate JWK thumbprint for token binding
	thumbprint, err := calculateJWKThumbprint(jwkRaw)
	if err != nil {
		return nil, "", &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: fmt.Sprintf("failed to calculate JWK thumbprint: %v", err),
		}
	}

	return proof, thumbprint, nil
}

// validateDPoPHeader validates the DPoP JWT header.
func (dm *DPoPManager) validateDPoPHeader(header map[string]interface{}) error {
	// Validate "typ" header
	typ, ok := header["typ"].(string)
	if !ok || typ != "dpop+jwt" {
		return &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: "invalid 'typ' header, must be 'dpop+jwt'",
		}
	}

	// Validate "alg" header
	alg, ok := header["alg"].(string)
	if !ok {
		return &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: "missing 'alg' header",
		}
	}

	// Check if algorithm is allowed
	allowedAlg := false
	for _, allowed := range dm.config.AllowedAlgorithms {
		if alg == allowed {
			allowedAlg = true
			break
		}
	}
	if !allowedAlg {
		return &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: fmt.Sprintf("unsupported algorithm '%s'", alg),
		}
	}

	return nil
}

// validateDPoPClaims validates the DPoP JWT claims.
func (dm *DPoPManager) validateDPoPClaims(proof *DPoPProof, httpMethod, httpURI, accessToken string) error {
	// Validate jti (unique identifier)
	if proof.JTI == "" {
		return &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: "missing 'jti' claim",
		}
	}

	// Check for JTI replay
	dm.mu.Lock()
	if _, used := dm.usedJTIs[proof.JTI]; used {
		dm.mu.Unlock()
		return &OIDCError{
			ErrorCode:        ErrorUseDPoPNonce,
			ErrorDescription: "DPoP proof replay detected (jti already used)",
		}
	}
	dm.usedJTIs[proof.JTI] = time.Now()
	dm.mu.Unlock()

	// Validate htm (HTTP method)
	if proof.HTTPMethod == "" {
		return &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: "missing 'htm' claim",
		}
	}
	if proof.HTTPMethod != httpMethod {
		return &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: fmt.Sprintf("'htm' mismatch: expected '%s', got '%s'", httpMethod, proof.HTTPMethod),
		}
	}

	// Validate htu (HTTP URI)
	if proof.HTTPURI == "" {
		return &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: "missing 'htu' claim",
		}
	}
	// Normalize URIs for comparison (remove query and fragment)
	expectedURI := normalizeHTTPURI(httpURI)
	actualURI := normalizeHTTPURI(proof.HTTPURI)
	if actualURI != expectedURI {
		return &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: fmt.Sprintf("'htu' mismatch: expected '%s', got '%s'", expectedURI, actualURI),
		}
	}

	// Validate iat (issued at)
	if proof.IssuedAt == 0 {
		return &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: "missing 'iat' claim",
		}
	}

	now := time.Now()
	issuedAt := time.Unix(proof.IssuedAt, 0)

	// Check if proof is too old
	age := now.Sub(issuedAt)
	if age > dm.config.MaxAge+dm.config.ClockSkew {
		return &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: fmt.Sprintf("DPoP proof too old (age: %v, max: %v)", age, dm.config.MaxAge),
		}
	}

	// Check if proof is from the future
	if issuedAt.After(now.Add(dm.config.ClockSkew)) {
		return &OIDCError{
			ErrorCode:        ErrorInvalidDPoPProof,
			ErrorDescription: "DPoP proof from the future",
		}
	}

	// Validate nonce (if required)
	if dm.config.RequireNonce {
		if proof.Nonce == "" {
			// Return use_dpop_nonce error with a new nonce
			return &OIDCError{
				ErrorCode:        ErrorUseDPoPNonce,
				ErrorDescription: "nonce required",
			}
		}

		dm.mu.Lock()
		nonceTime, exists := dm.nonces[proof.Nonce]
		dm.mu.Unlock()

		if !exists {
			return &OIDCError{
				ErrorCode:        ErrorUseDPoPNonce,
				ErrorDescription: "invalid or expired nonce",
			}
		}

		// Check nonce age
		if now.Sub(nonceTime) > dm.config.NonceLifetime {
			dm.mu.Lock()
			delete(dm.nonces, proof.Nonce)
			dm.mu.Unlock()

			return &OIDCError{
				ErrorCode:        ErrorUseDPoPNonce,
				ErrorDescription: "nonce expired",
			}
		}

		// Remove used nonce (single-use)
		dm.mu.Lock()
		delete(dm.nonces, proof.Nonce)
		dm.mu.Unlock()
	}

	// Validate ath (access token hash) if access token provided
	if accessToken != "" {
		expectedATH := calculateAccessTokenHash(accessToken)
		if proof.AccessTokenHash != expectedATH {
			return &OIDCError{
				ErrorCode:        ErrorInvalidDPoPProof,
				ErrorDescription: "access token hash mismatch",
			}
		}
	}

	return nil
}

// BindAccessToken binds an access token to a DPoP key.
func (dm *DPoPManager) BindAccessToken(tokenID, jwkThumbprint string) {
	dm.mu.Lock()
	dm.dpopKeys[tokenID] = jwkThumbprint
	dm.mu.Unlock()
}

// ValidateTokenBinding checks if an access token is bound to the correct DPoP key.
func (dm *DPoPManager) ValidateTokenBinding(tokenID, jwkThumbprint string) error {
	dm.mu.RLock()
	boundThumbprint, exists := dm.dpopKeys[tokenID]
	dm.mu.RUnlock()

	if !exists {
		return &OIDCError{
			ErrorCode:        ErrorInvalidToken,
			ErrorDescription: "access token not DPoP-bound",
		}
	}

	if boundThumbprint != jwkThumbprint {
		return &OIDCError{
			ErrorCode:        ErrorInvalidToken,
			ErrorDescription: "DPoP key binding mismatch",
		}
	}

	return nil
}

// cleanupLoop removes expired nonces, JTIs, and token bindings.
func (dm *DPoPManager) cleanupLoop() {
	defer dm.wg.Done()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dm.cleanup()
		case <-dm.stopCleanup:
			return
		}
	}
}

// cleanup removes expired entries.
func (dm *DPoPManager) cleanup() {
	now := time.Now()

	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Remove expired nonces
	for nonce, createdAt := range dm.nonces {
		if now.Sub(createdAt) > dm.config.NonceLifetime {
			delete(dm.nonces, nonce)
		}
	}

	// Remove old JTIs (keep for MaxAge + some buffer)
	maxJTIAge := dm.config.MaxAge + 5*time.Minute
	for jti, usedAt := range dm.usedJTIs {
		if now.Sub(usedAt) > maxJTIAge {
			delete(dm.usedJTIs, jti)
		}
	}
}

// Close stops the cleanup goroutine.
func (dm *DPoPManager) Close() error {
	close(dm.stopCleanup)
	dm.wg.Wait()
	return nil
}

// parseJWKPublicKey parses a JWK and returns the public key.
func parseJWKPublicKey(jwk map[string]interface{}) (interface{}, error) {
	kty, ok := jwk["kty"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'kty' in JWK")
	}

	switch kty {
	case "RSA":
		return parseRSAPublicKey(jwk)
	case "EC":
		return parseECPublicKey(jwk)
	case "OKP":
		return parseOKPPublicKey(jwk)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", kty)
	}
}

// parseRSAPublicKey parses an RSA JWK.
func parseRSAPublicKey(jwk map[string]interface{}) (*rsa.PublicKey, error) {
	n, ok := jwk["n"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'n' in RSA JWK")
	}
	e, ok := jwk["e"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'e' in RSA JWK")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, fmt.Errorf("invalid 'n': %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, fmt.Errorf("invalid 'e': %w", err)
	}

	// Convert e bytes to int
	var eInt int
	for _, b := range eBytes {
		eInt = eInt<<8 | int(b)
	}

	// Create big.Int from n bytes
	nBig := new(big.Int).SetBytes(nBytes)

	return &rsa.PublicKey{
		N: nBig,
		E: eInt,
	}, nil
}

// parseECPublicKey parses an EC JWK.
func parseECPublicKey(jwk map[string]interface{}) (*ecdsa.PublicKey, error) {
	// This is a simplified implementation
	// In production, use a proper JWK parsing library
	return nil, fmt.Errorf("EC key parsing not fully implemented")
}

// parseOKPPublicKey parses an OKP (Ed25519) JWK.
func parseOKPPublicKey(jwk map[string]interface{}) (ed25519.PublicKey, error) {
	x, ok := jwk["x"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'x' in OKP JWK")
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(x)
	if err != nil {
		return nil, fmt.Errorf("invalid 'x': %w", err)
	}

	if len(xBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public key size")
	}

	return ed25519.PublicKey(xBytes), nil
}

// calculateJWKThumbprint calculates the JWK thumbprint per RFC 7638.
func calculateJWKThumbprint(jwk map[string]interface{}) (string, error) {
	// Create canonical JSON representation
	kty, _ := jwk["kty"].(string)

	var canonical map[string]string
	switch kty {
	case "RSA":
		canonical = map[string]string{
			"e":   jwk["e"].(string),
			"kty": kty,
			"n":   jwk["n"].(string),
		}
	case "EC":
		canonical = map[string]string{
			"crv": jwk["crv"].(string),
			"kty": kty,
			"x":   jwk["x"].(string),
			"y":   jwk["y"].(string),
		}
	case "OKP":
		canonical = map[string]string{
			"crv": jwk["crv"].(string),
			"kty": kty,
			"x":   jwk["x"].(string),
		}
	default:
		return "", fmt.Errorf("unsupported key type for thumbprint: %s", kty)
	}

	// JSON encode in canonical form
	jsonBytes, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}

	// SHA-256 hash
	hash := sha256.Sum256(jsonBytes)
	return base64.RawURLEncoding.EncodeToString(hash[:]), nil
}

// calculateAccessTokenHash calculates the "ath" value for a DPoP proof.
func calculateAccessTokenHash(accessToken string) string {
	hash := sha256.Sum256([]byte(accessToken))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// normalizeHTTPURI removes query parameters and fragments from a URI.
func normalizeHTTPURI(uri string) string {
	// Remove query
	if idx := strings.Index(uri, "?"); idx >= 0 {
		uri = uri[:idx]
	}
	// Remove fragment
	if idx := strings.Index(uri, "#"); idx >= 0 {
		uri = uri[:idx]
	}
	return uri
}

// ExtractDPoPProof extracts the DPoP proof from an HTTP request.
func ExtractDPoPProof(r *http.Request) string {
	return r.Header.Get("DPoP")
}

// Additional error codes for DPoP
const (
	ErrorInvalidDPoPProof = "invalid_dpop_proof"
	ErrorUseDPoPNonce     = "use_dpop_nonce"
)
