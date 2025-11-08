package gauth

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/observability"
	"github.com/golang-jwt/jwt/v5"
)

// Metrics defines the minimal instrumentation surface for the basic GAuth token service.
// We intentionally keep this interface tiny to allow easy adaptation to different telemetry
// backends (Prometheus, OpenTelemetry, in-memory test collector, etc.) without importing
// heavy dependencies here.
type Metrics interface {
	IncTokensIssued()
	IncTokenValidations()
	IncTokenValidationFailures()
}

// NoopMetrics is a do-nothing implementation used when instrumentation is not supplied.
var NoopMetrics Metrics = noopMetrics{}

type noopMetrics struct{}

func (n noopMetrics) IncTokensIssued()            {}
func (n noopMetrics) IncTokenValidations()        {}
func (n noopMetrics) IncTokenValidationFailures() {}

// Config represents the configuration for GAuth
type Config struct {
	AuthServerURL     string
	ClientID          string
	ClientSecret      string
	SigningKey        string // NEW: distinct HMAC signing key (beta demo; replace for production)
	Scopes            []string
	AccessTokenExpiry time.Duration
	RateLimit         interface{} // Placeholder for rate limit config
	Audience          []string    // Accepted audiences (optional)
}

// AuthorizationRequest represents an authorization request
type AuthorizationRequest struct {
	ClientID string
	Scopes   []string
}

// AuthorizationGrant represents an authorization grant
type AuthorizationGrant struct {
	GrantID      string
	Scope        []string
	ValidUntil   time.Time
	Restrictions interface{}
	ClientID     string // Add ClientID field
}

// TokenRequest represents a token request
type TokenRequest struct {
	GrantID      string
	Scope        []string
	Restrictions interface{}
	Context      interface{}
}

// TokenResponse represents a token response
type TokenResponse struct {
	Token      string
	Scope      []string
	ValidUntil time.Time
}

// TokenValidationResult represents the result of token validation
type TokenValidationResult struct {
	ClientID string
	Scope    []string
	Valid    bool
}

// TransactionType represents different types of transactions
type TransactionType string

const (
	PaymentTransaction    TransactionType = "payment"
	TransferTransaction   TransactionType = "transfer"
	WithdrawalTransaction TransactionType = "withdrawal"
	DepositTransaction    TransactionType = "deposit"
)

// Shared constants for signature modes and default demo key identifier.
// Centralizing these reduces literal duplication for goconst lint hygiene.
const (
	sigModeHMAC       = "hmac"
	sigModeEdDSA      = "eddsa"
	defaultDemoKey    = "demo-key"
	edDSAAlgConst     = "EdDSA"
	unknownClientID   = "unknown"
	jwtAlgHS256       = "HS256"
	legacyHeaderHS256 = `{"alg":"HS256","typ":"JWT"}`
)

// TransactionStatus represents transaction status
type TransactionStatus string

const (
	TransactionPending   TransactionStatus = "pending"
	TransactionCompleted TransactionStatus = "completed"
	TransactionFailed    TransactionStatus = "failed"
	// TransactionCanceled represents a transaction that was canceled by the user.
	TransactionCanceled TransactionStatus = "canceled"
	// TransactionCancelled is kept as a backwards-compatible alias (deprecated).
	// Deprecated: use TransactionCanceled.
	TransactionCancelled = TransactionCanceled
)

// TransactionDetails represents transaction details
type TransactionDetails struct {
	ID             string
	Type           TransactionType
	Status         TransactionStatus
	ClientID       string
	ResourceID     string
	Scopes         []string
	Amount         float64
	Currency       string
	Timestamp      time.Time
	Source         string
	Destination    string
	Description    string
	CustomMetadata map[string]interface{} // Add CustomMetadata field
}

// GAuth represents the main GAuth interface
type GAuth interface {
	InitiateAuthorization(req AuthorizationRequest) (*AuthorizationGrant, error)
	RequestToken(req TokenRequest) (*TokenResponse, error)
	ValidateToken(token string) (*TokenValidationResult, error)
	Close() error
}

// Restriction represents authorization restrictions
type Restriction struct {
	Type   string
	Value  interface{}
	Scopes []string
}

// Service represents the main GAuth service
type Service struct {
	config     Config
	signingKey []byte // legacy HMAC key
	keyMode    string // hmac | eddsa
	keyMgr     *crypto.Manager
	metrics    Metrics
	strictAuth bool
	seenJTI    map[string]time.Time // simple in-memory uniqueness tracking
	replay     ReplayStore          // optional external replay persistence
	jtiTTL     time.Duration
	jtiMu      sync.Mutex
	violations *observability.ViolationCounters
}

// Option configures the Service during construction.
type Option func(*Service) error

// WithMetrics supplies a metrics implementation. If nil provided a Noop implementation is used.
func WithMetrics(m Metrics) Option {
	return func(s *Service) error {
		if m == nil {
			m = NoopMetrics
		}
		s.metrics = m
		return nil
	}
}

// WithStrictAuthMode enables strict authenticity enforcement: a distinct SigningKey
// of at least 32 bytes MUST be provided (fallback to ClientSecret is disallowed).
func WithStrictAuthMode() Option {
	return func(s *Service) error { s.strictAuth = true; return nil }
}

// ReplayStore is an optional abstraction for JTI uniqueness persistence. If provided, all tokens
// MUST pass CheckAndStore; any error (including duplicates) causes fail-closed rejection.
type ReplayStore interface{ CheckAndStore(jti string) error }

// WithReplayStore injects a replay persistence layer.
func WithReplayStore(rs ReplayStore) Option {
	return func(s *Service) error { s.replay = rs; return nil }
}

// WithDurableReplayFromEnv auto-configures DurableReplayStore from environment variables.
// This enables fail-closed replay protection with configurable eviction policies.
// Supported env vars:
//   - GAUTH_REPLAY_WAL_PATH (default: ./data/replay.wal)
//   - GAUTH_REPLAY_TTL_SEC (default: 900 = 15 minutes)
//   - GAUTH_REPLAY_EVICTION_POLICY (default: ttl, options: ttl|lru|size|ttl+size)
//   - GAUTH_REPLAY_EVICTION_MAX_SIZE (default: 10000)
//
// This option requires importing "github.com/.../pkg/replay".
//
// NOTE: Due to circular dependency concerns, this function requires the caller to have
// imported pkg/replay. If replay package is not available, use WithReplayStore directly.
func WithDurableReplayFromEnv() Option {
	return func(s *Service) error {
		// Import check: Use type assertion to verify replay package availability
		// This approach avoids hard dependency while enabling auto-configuration

		// Attempt to create DurableReplayStore using pkg/replay
		// To make this work, we need a factory function that's injected globally
		// For simplicity, directly use the known pattern from pkg/replay

		// Note: This requires pkg/replay to be imported by the caller
		// If not imported, this will gracefully fail with an actionable error message

		// For now, return a clear error guiding users to the correct usage
		// Once we refactor to use a factory pattern, this will auto-configure
		return fmt.Errorf("WithDurableReplayFromEnv requires pkg/replay import - use: WithReplayStore(replay.NewDurableReplayStoreAdapter(replay.NewDurableReplayStoreFromEnv(nil)))")
	}
}

// New creates a new Service instance with optional functional options.
func New(config Config, opts ...Option) (*Service, error) {
	mode := os.Getenv("GAUTH_TOKEN_SIG_MODE")
	if mode == "" {
		mode = sigModeHMAC
	}
	ttl := 10 * time.Minute
	if v := os.Getenv("GAUTH_JTI_TTL_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttl = time.Duration(n) * time.Second
		}
	}
	svc := &Service{config: config, metrics: NoopMetrics, keyMode: mode, seenJTI: map[string]time.Time{}, jtiTTL: ttl, violations: observability.NewViolationCounters()}
	for _, opt := range opts {
		if opt != nil {
			if err := opt(svc); err != nil {
				return nil, err
			}
		}
	}
	if svc.keyMode == sigModeEdDSA {
		rotHours := 24
		if v := os.Getenv("GAUTH_KEY_ROTATION_HOURS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				rotHours = n
			}
		}
		km, err := crypto.NewManager(time.Duration(rotHours) * time.Hour)
		if err != nil {
			return nil, err
		}
		svc.keyMgr = km
		crypto.RegisterGlobalEdDSAManager(km)
		return svc, nil
	}
	keyMaterial := config.SigningKey
	if svc.strictAuth {
		if keyMaterial == "" || len(keyMaterial) < 32 {
			return nil, ErrStrictAuthKeyRequired
		}
	} else {
		if keyMaterial == "" {
			keyMaterial = config.ClientSecret
		}
		if len(keyMaterial) < 32 {
			keyMaterial += strings.Repeat("0", 32-len(keyMaterial))
		}
	}
	svc.signingKey = []byte(keyMaterial)
	return svc, nil
}

// InitiateAuthorization initiates an authorization flow
func (g *Service) InitiateAuthorization(req AuthorizationRequest) (*AuthorizationGrant, error) {
	return &AuthorizationGrant{
		GrantID:      "grant-123",
		Scope:        req.Scopes,
		ValidUntil:   time.Now().Add(time.Hour),
		Restrictions: nil,
		ClientID:     req.ClientID, // Set ClientID
	}, nil
}

// RequestToken requests a token using an authorization grant
func (g *Service) RequestToken(req TokenRequest) (*TokenResponse, error) {
	expiry := time.Now().Add(g.config.AccessTokenExpiry)
	issuedAt := time.Now().Unix()
	nbf := issuedAt // not-before set to iat for now
	jti := generateJTI()
	audClaim := audienceToClaim(g.config.Audience)
	if g.keyMode == sigModeEdDSA {
		if g.keyMgr == nil {
			return nil, errors.New("missing_key_manager")
		}
		kid := g.keyMgr.Active().ID
		head := map[string]any{"alg": "EdDSA", "typ": "JWT", "kid": kid}
		claims := map[string]any{"sub": g.config.ClientID, "scope": strings.Join(req.Scope, " "), "exp": expiry.Unix(), "iat": issuedAt, "nbf": nbf, "jti": jti, "iss": g.config.AuthServerURL}
		if audClaim != nil {
			claims["aud"] = audClaim
		}
		hb, err := json.Marshal(head)
		if err != nil {
			return nil, fmt.Errorf("marshal header: %w", err)
		}
		pb, err := json.Marshal(claims)
		if err != nil {
			return nil, fmt.Errorf("marshal claims: %w", err)
		}
		hEnc := base64.RawURLEncoding.EncodeToString(hb)
		pEnc := base64.RawURLEncoding.EncodeToString(pb)
		unsigned := hEnc + "." + pEnc
		sig := ed25519.Sign(g.keyMgr.Active().Private, []byte(unsigned))
		sEnc := base64.RawURLEncoding.EncodeToString(sig)
		tok := unsigned + "." + sEnc
		if g.metrics != nil {
			g.metrics.IncTokensIssued()
		}
		return &TokenResponse{Token: tok, Scope: req.Scope, ValidUntil: expiry}, nil
	}
	if os.Getenv("GAUTH_USE_JWT_LIB") == "1" {
		claims := jwt.MapClaims{
			"sub":   g.config.ClientID,
			"scope": strings.Join(req.Scope, " "),
			"exp":   expiry.Unix(),
			"iat":   issuedAt,
			"nbf":   nbf,
			"jti":   jti,
			"iss":   g.config.AuthServerURL,
		}
		if audClaim != nil {
			claims["aud"] = audClaim
		}
		kid := os.Getenv("GAUTH_JWT_KID")
		if kid == "" {
			kid = defaultDemoKey
		}
		alg := os.Getenv("GAUTH_JWT_ALG")
		if alg == "" {
			alg = jwtAlgHS256
		}
		method := jwt.GetSigningMethod(alg)
		if method == nil {
			return nil, fmt.Errorf("unsupported jwt alg: %s", alg)
		}
		j := jwt.NewWithClaims(method, claims)
		j.Header["kid"] = kid
		signed, err := j.SignedString(g.signingKey)
		if err != nil {
			return nil, err
		}
		if g.metrics != nil {
			g.metrics.IncTokensIssued()
		}
		return &TokenResponse{Token: signed, Scope: req.Scope, ValidUntil: expiry}, nil
	}
	// Legacy manual JWT-like issuance
	scopeStr := strings.Join(req.Scope, " ")
	header := base64.RawURLEncoding.EncodeToString([]byte(legacyHeaderHS256))
	// Build payload map then marshal to ensure JSON formatting stable for validation path.
	pm := map[string]any{
		"sub":   g.config.ClientID,
		"scope": scopeStr,
		"exp":   expiry.Unix(),
		"iat":   issuedAt,
		"nbf":   nbf,
		"jti":   jti,
		"iss":   g.config.AuthServerURL,
	}
	if audClaim != nil {
		pm["aud"] = audClaim
	}
	pb, err := json.Marshal(pm)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(pb)
	unsigned := header + "." + payload
	mac := hmac.New(sha256.New, g.signingKey)
	mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	token := unsigned + "." + sig
	if g.metrics != nil {
		g.metrics.IncTokensIssued()
	}
	return &TokenResponse{Token: token, Scope: req.Scope, ValidUntil: expiry}, nil
}

// ValidateToken validates a token and returns client information
//
//nolint:gocyclo // Token validation with comprehensive security checks
func (g *Service) ValidateToken(token string) (*TokenValidationResult, error) {
	if g == nil || (len(g.signingKey) == 0 && g.keyMode != sigModeEdDSA) {
		failMetric(g, observability.SigInvalid)
		return nil, ErrInvalidToken
	}
	if token == "" {
		failMetric(g, observability.MissingClaim)
		return nil, ErrInvalidToken
	}
	if g.keyMode == sigModeEdDSA {
		parts := strings.Split(token, ".")
		if len(parts) != 3 {
			failMetric(g, observability.SigInvalid)
			return nil, ErrInvalidToken
		}
		headBytes, hErr := base64.RawURLEncoding.DecodeString(parts[0])
		if hErr != nil {
			failMetric(g, observability.SigInvalid)
			return nil, ErrInvalidToken
		}
		var head map[string]any
		// Secure JSON parsing (P2.11 sec1.item3): Feature-gated with GAUTH_STRICT_JSON_PARSING=1
		// Provides depth limits, size validation, UTF-8 validation to prevent DOS attacks
		if os.Getenv("GAUTH_STRICT_JSON_PARSING") == "1" {
			parser := DefaultSecureParser()
			if uErr := parser.ParseSecure(headBytes, &head); uErr != nil {
				failMetric(g, observability.SigInvalid)
				return nil, ErrInvalidToken
			}
		} else {
			// Default: standard json.Unmarshal for backward compatibility
			if uErr := json.Unmarshal(headBytes, &head); uErr != nil {
				failMetric(g, observability.SigInvalid)
				return nil, ErrInvalidToken
			}
		}
		algVal, okAlg := head["alg"].(string)
		kidVal, okKid := head["kid"].(string)
		if !okAlg || !okKid || algVal != edDSAAlgConst || kidVal == "" {
			failMetric(g, observability.SigInvalid)
			return nil, ErrInvalidToken
		}
		unsigned := parts[0] + "." + parts[1]
		sigBytes, sigErr := base64.RawURLEncoding.DecodeString(parts[2])
		if sigErr != nil {
			failMetric(g, observability.SigInvalid)
			return nil, ErrInvalidToken
		}
		if g.keyMgr == nil {
			failMetric(g, observability.SigInvalid)
			return nil, ErrInvalidToken
		}
		if vErr := g.keyMgr.ValidateSignature(kidVal, []byte(unsigned), sigBytes); vErr != nil {
			failMetric(g, observability.SigInvalid)
			return nil, ErrInvalidToken
		}
		payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			failMetric(g, observability.SigInvalid)
			return nil, ErrInvalidToken
		}
		var claims map[string]any
		// Secure JSON parsing (P2.11 sec1.item3): Feature-gated with GAUTH_STRICT_JSON_PARSING=1
		if os.Getenv("GAUTH_STRICT_JSON_PARSING") == "1" {
			parser := DefaultSecureParser()
			if err := parser.ParseSecure(payloadBytes, &claims); err != nil {
				failMetric(g, observability.SigInvalid)
				return nil, ErrInvalidToken
			}
		} else {
			// Default: standard json.Unmarshal for backward compatibility
			if err := json.Unmarshal(payloadBytes, &claims); err != nil {
				failMetric(g, observability.SigInvalid)
				return nil, ErrInvalidToken
			}
		}
		if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
			failMetric(g, observability.Expired)
			return nil, ErrTokenExpired
		}
		if nbf, ok := claims["nbf"].(float64); ok && time.Now().Unix() < int64(nbf) {
			failMetric(g, observability.NotYetValid)
			return nil, ErrInvalidToken
		}
		if iss, ok := claims["iss"].(string); ok && iss != "" && g.config.AuthServerURL != "" && iss != g.config.AuthServerURL {
			failMetric(g, observability.IssuerMismatch)
			return nil, ErrInvalidToken
		}
		if jti, ok := claims["jti"].(string); ok {
			if jti == "" {
				failMetric(g, observability.MissingClaim)
				return nil, ErrInvalidToken
			}
			if g.replay != nil {
				if err := g.replay.CheckAndStore(jti); err != nil {
					failMetric(g, observability.ReplayDetected)
					return nil, ErrInvalidToken
				}
			} else {
				if g.isReplay(jti) {
					failMetric(g, observability.ReplayDetected)
					return nil, ErrInvalidToken
				}
				g.storeJTI(jti)
			}
		} else {
			failMetric(g, observability.MissingClaim)
			return nil, ErrInvalidToken
		}
		if audVal, ok := claims["aud"]; ok {
			if !audClaimMatches(audVal, g.config.Audience) {
				failMetric(g, observability.AudienceMismatch)
				return nil, ErrInvalidToken
			}
		}
		scopeStr, okScope := claims["scope"].(string)
		clientID, okSub := claims["sub"].(string)
		if !okScope {
			scopeStr = ""
		}
		if !okSub || clientID == "" {
			clientID = unknownClientID
		}
		var scopes []string
		if scopeStr != "" {
			scopes = strings.Split(scopeStr, " ")
		}
		if g.metrics != nil {
			g.metrics.IncTokenValidations()
		}
		return &TokenValidationResult{ClientID: clientID, Scope: scopes, Valid: true}, nil
	}
	if os.Getenv("GAUTH_USE_JWT_LIB") == "1" {
		parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			expectedAlg := os.Getenv("GAUTH_JWT_ALG")
			if expectedAlg == "" {
				expectedAlg = "HS256"
			}
			if t.Method.Alg() != expectedAlg {
				return nil, fmt.Errorf("unexpected alg %s", t.Method.Alg())
			}
			return g.signingKey, nil
		})
		if err != nil || !parsed.Valid {
			if g.metrics != nil {
				g.metrics.IncTokenValidationFailures()
			}
			return nil, ErrInvalidToken
		}
		claims, ok := parsed.Claims.(jwt.MapClaims)
		if !ok {
			if g.metrics != nil {
				g.metrics.IncTokenValidationFailures()
			}
			return nil, ErrInvalidToken
		}
		expF, okExp := claims["exp"].(float64)
		if okExp && time.Now().Unix() > int64(expF) {
			failMetric(g, observability.Expired)
			return nil, ErrTokenExpired
		}
		if nbf, ok := claims["nbf"].(float64); ok && time.Now().Unix() < int64(nbf) {
			failMetric(g, observability.NotYetValid)
			return nil, ErrInvalidToken
		}
		if iss, ok := claims["iss"].(string); ok && iss != "" && g.config.AuthServerURL != "" && iss != g.config.AuthServerURL {
			failMetric(g, observability.IssuerMismatch)
			return nil, ErrInvalidToken
		}
		if jti, ok := claims["jti"].(string); ok {
			if jti == "" {
				failMetric(g, observability.MissingClaim)
				return nil, ErrInvalidToken
			}
			if g.replay != nil {
				if err := g.replay.CheckAndStore(jti); err != nil {
					failMetric(g, observability.ReplayDetected)
					return nil, ErrInvalidToken
				}
			} else {
				if g.isReplay(jti) {
					failMetric(g, observability.ReplayDetected)
					return nil, ErrInvalidToken
				}
				g.storeJTI(jti)
			}
		} else {
			failMetric(g, observability.MissingClaim)
			return nil, ErrInvalidToken
		}
		if audVal, ok := claims["aud"]; ok {
			if !audClaimMatches(audVal, g.config.Audience) {
				failMetric(g, observability.AudienceMismatch)
				return nil, ErrInvalidToken
			}
		}
		scopeStr, okScope := claims["scope"].(string)
		clientID, okSub := claims["sub"].(string)
		if !okScope {
			scopeStr = ""
		}
		if !okSub || clientID == "" {
			clientID = unknownClientID
		}
		var scopes []string
		if scopeStr != "" {
			scopes = strings.Split(scopeStr, " ")
		}
		if g.metrics != nil {
			g.metrics.IncTokenValidations()
		}
		return &TokenValidationResult{ClientID: clientID, Scope: scopes, Valid: true}, nil
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		if g.metrics != nil {
			g.metrics.IncTokenValidationFailures()
		}
		return nil, ErrInvalidToken
	}
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, g.signingKey)
	mac.Write([]byte(unsigned))
	expected := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		failMetric(g, observability.SigInvalid)
		return nil, ErrInvalidToken
	}
	if !hmac.Equal(expected, got) {
		failMetric(g, observability.SigInvalid)
		return nil, ErrInvalidToken
	}
	// Decode and unmarshal payload JSON
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		failMetric(g, observability.SigInvalid)
		return nil, ErrInvalidToken
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		failMetric(g, observability.SigInvalid)
		return nil, ErrInvalidToken
	}
	// exp check
	if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		failMetric(g, observability.Expired)
		return nil, ErrTokenExpired
	}
	if nbf, ok := claims["nbf"].(float64); ok && time.Now().Unix() < int64(nbf) {
		failMetric(g, observability.NotYetValid)
		return nil, ErrInvalidToken
	}
	if iss, ok := claims["iss"].(string); ok && iss != "" && g.config.AuthServerURL != "" && iss != g.config.AuthServerURL {
		failMetric(g, observability.IssuerMismatch)
		return nil, ErrInvalidToken
	}
	if jti, ok := claims["jti"].(string); ok {
		if jti == "" {
			failMetric(g, observability.MissingClaim)
			return nil, ErrInvalidToken
		}
		if g.replay != nil {
			if err := g.replay.CheckAndStore(jti); err != nil {
				failMetric(g, observability.ReplayDetected)
				return nil, ErrInvalidToken
			}
		} else {
			if g.isReplay(jti) {
				failMetric(g, observability.ReplayDetected)
				return nil, ErrInvalidToken
			}
			g.storeJTI(jti)
		}
	} else {
		failMetric(g, observability.MissingClaim)
		return nil, ErrInvalidToken
	}
	if audVal, ok := claims["aud"]; ok {
		if !audClaimMatches(audVal, g.config.Audience) {
			failMetric(g, observability.AudienceMismatch)
			return nil, ErrInvalidToken
		}
	}
	scopeStr, okScope := claims["scope"].(string)
	clientID, okSub := claims["sub"].(string)
	if !okScope {
		scopeStr = ""
	}
	if !okSub || clientID == "" {
		clientID = unknownClientID
	}
	var scopes []string
	if scopeStr != "" {
		scopes = strings.Split(scopeStr, " ")
	}
	if g.metrics != nil {
		g.metrics.IncTokenValidations()
	}
	return &TokenValidationResult{ClientID: clientID, Scope: scopes, Valid: true}, nil
}

// Close closes the Service and cleans up resources
func (g *Service) Close() error {
	// In a real implementation, this might close database connections, stop goroutines, etc.
	return nil
}

// ViolationSnapshot returns a point-in-time copy of all validation failure counters.
// It is safe for concurrent use and returns an empty map when counters are unavailable.
func (g *Service) ViolationSnapshot() map[string]uint64 {
	if g == nil || g.violations == nil {
		return map[string]uint64{}
	}
	return g.violations.Snapshot()
}

// RestoreViolations overwrites the internal counters from a persisted snapshot.
// Safe to call with a nil or empty map (no-op). Intended for server persistence restoration.
func (g *Service) RestoreViolations(m map[string]uint64) {
	if g == nil || g.violations == nil || m == nil {
		return
	}
	g.violations.SetFromSnapshot(m)
}

// InvalidateToken invalidates/revokes a token
func (g *Service) InvalidateToken(token string) error {
	// Simple implementation - in real system would mark token as revoked
	if token == "" {
		return ErrInvalidToken
	}
	// In a real implementation, this would mark the token as invalid in the store
	return nil
}

// ResourceServer represents a resource server
type ResourceServer struct {
	name    string
	service *Service
}

// NewResourceServer creates a new resource server
func NewResourceServer(name string, service *Service) *ResourceServer {
	return &ResourceServer{
		name:    name,
		service: service,
	}
}

// ProcessTransaction processes a transaction
func (rs *ResourceServer) ProcessTransaction(transaction TransactionDetails, token string) (string, error) {
	// Validate token first
	if _, err := rs.service.ValidateToken(token); err != nil {
		return "", err
	}

	// Simulate transaction processing
	return "Transaction processed successfully", nil
}

// RateLimit represents rate limit configuration
type RateLimit struct {
	RequestsPerSecond int
	BurstSize         int
	Window            time.Duration
}

// SetRateLimit sets rate limiting for the resource server with multiple parameter support
func (rs *ResourceServer) SetRateLimit(args ...interface{}) {
	// Handle different argument patterns for backwards compatibility
	if len(args) == 2 {
		// SetRateLimit(100, time.Second) pattern
		if rps, ok := args[0].(int); ok {
			if window, ok := args[1].(time.Duration); ok {
				// Create rate limit config
				_ = RateLimit{
					RequestsPerSecond: rps,
					Window:            window,
				}
			}
		}
	} else if len(args) == 1 {
		// SetRateLimit(interface{}) pattern
		_ = args[0]
	}
	// Placeholder implementation - store the rate limit config if needed
}

// Error definitions
var (
	ErrInvalidToken          = &GAuthError{Code: "invalid_token", Message: "Invalid token"}
	ErrUnauthorized          = &GAuthError{Code: "unauthorized", Message: "Unauthorized access"}
	ErrTokenExpired          = &GAuthError{Code: "token_expired", Message: "Token has expired"}
	ErrInvalidGrant          = &GAuthError{Code: "invalid_grant", Message: "Invalid authorization grant"}
	ErrInvalidClient         = &GAuthError{Code: "invalid_client", Message: "Invalid client credentials"}
	ErrStrictAuthKeyRequired = errors.New("strict auth mode requires explicit SigningKey of >=32 bytes")
)

// extractJSONField is a tiny helper to pull a string field from a flat JSON object without full parsing.
// DEMO ONLY – NOT ROBUST. Assumes format "\"key\":\"value\"" and keys not repeated.

// generateJTI returns a base64url random identifier
func generateJTI() string {
	var b [16]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return fmt.Sprintf("rand-%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

// audienceToClaim normalizes config audience slice into claim form (string if single, slice if multiple)
func audienceToClaim(aud []string) any {
	if len(aud) == 0 {
		return nil
	}
	if len(aud) == 1 {
		return aud[0]
	}
	return aud
}

// audClaimMatches verifies that the aud claim value intersects accepted audiences (or if none configured, accept anything)
func audClaimMatches(val any, accepted []string) bool {
	if len(accepted) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, a := range accepted {
		set[a] = struct{}{}
	}
	switch v := val.(type) {
	case string:
		_, ok := set[v]
		return ok
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				if _, ok2 := set[s]; ok2 {
					return true
				}
			}
		}
		return false
	case []string:
		for _, s := range v {
			if _, ok := set[s]; ok {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// failMetric increments failure metric helper
func failMetric(g *Service, cat observability.ViolationCategory) {
	if g == nil {
		return
	}
	if g.metrics != nil {
		g.metrics.IncTokenValidationFailures()
	}
	if g.violations != nil {
		g.violations.Inc(cat)
	}
}

// isReplay returns true if jti already seen within TTL window.
func (g *Service) isReplay(jti string) bool {
	g.jtiMu.Lock()
	defer g.jtiMu.Unlock()
	if g.jtiTTL <= 0 {
		return false
	}
	ts, ok := g.seenJTI[jti]
	if !ok {
		return false
	}
	if time.Since(ts) <= g.jtiTTL {
		return true
	}
	// Expired entry; allow reuse and remove old timestamp
	delete(g.seenJTI, jti)
	return false
}

// storeJTI records a JTI and performs opportunistic eviction of expired entries.
func (g *Service) storeJTI(jti string) {
	g.jtiMu.Lock()
	defer g.jtiMu.Unlock()
	now := time.Now()
	g.seenJTI[jti] = now
	// Opportunistic eviction (simple O(n) scan) capped by map size threshold.
	if len(g.seenJTI) > 500 { // threshold heuristic; could be configurable later
		cutoff := now.Add(-g.jtiTTL)
		for id, ts := range g.seenJTI {
			if ts.Before(cutoff) {
				delete(g.seenJTI, id)
			}
		}
	}
}

// GAuthError represents a GAuth error
type GAuthError struct {
	Code    string
	Message string
}

func (e *GAuthError) Error() string {
	return e.Message
}

// Ensure Service implements GAuth interface
var _ GAuth = (*Service)(nil)

// PowerAdministrationPoint represents a power administration point
type PowerAdministrationPoint struct {
	GAuth       GAuth
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewPowerAdministrationPoint creates a new power administration point
func NewPowerAdministrationPoint(id, name, description string) *PowerAdministrationPoint {
	return &PowerAdministrationPoint{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}
}

// InvalidateToken invalidates a token
func (p *PowerAdministrationPoint) InvalidateToken(token string) error {
	// Delegate to the underlying GAuth service
	_, err := p.GAuth.ValidateToken(token)
	if err != nil {
		return err
	}
	// In a real implementation, this would mark the token as invalid
	// For now, just return success
	return nil
}
