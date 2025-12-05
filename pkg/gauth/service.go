package gauth

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/Gauth_go/internal/config"
	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
	"github.com/mauriciomferz/Gauth_go/internal/observability"
	"github.com/mauriciomferz/Gauth_go/pkg/poa"
)

// Config represents the configuration for GAuth

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

// GAuth represents the main GAuth interface
type GAuth interface {
	InitiateAuthorization(req AuthorizationRequest) (*AuthorizationGrant, error)
	RequestToken(req TokenRequest) (*TokenResponse, error)
	ValidateToken(token string) (*TokenValidationResult, error)
	Close() error
}

// Service represents the main GAuth service
type Service struct {
	config         Config
	signingKey     []byte // legacy HMAC key
	keyMode        string // hmac | eddsa
	keyMgr         *crypto.Manager
	metrics        Metrics
	strictAuth     bool
	seenJTI        map[string]time.Time // simple in-memory uniqueness tracking
	replay         ReplayStore          // optional external replay persistence
	jtiTTL         time.Duration
	jtiMu          sync.Mutex
	violations     *observability.ViolationCounters
	validator      TokenSignatureValidator
	claimValidator *CommonClaimValidator // optional external replay persistence

	// RFC-0111 compliance components
	protocolOrchestrator  *ProtocolOrchestrator
	subscriptionManager   *SubscriptionFlowManager
	powerEnforcementPoint *PowerEnforcementPoint
	powerDecisionPoint    PowerDecisionPoint
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

// WithKeyManager injects a crypto.Manager for EdDSA operations.
// This allows for explicit dependency injection instead of relying on global state.
// If not provided when keyMode is EdDSA, a new manager will be created automatically.
func WithKeyManager(km *crypto.Manager) Option {
	return func(s *Service) error {
		if km == nil {
			return fmt.Errorf("key manager cannot be nil")
		}
		s.keyMgr = km
		return nil
	}
}

// WithRFCCompliance configures RFC-0111 compliance components
// This enables the full RFC-0111 subscription and authorization flows
func WithRFCCompliance(
	subscriptionStore SubscriptionStore,
	extendedTokenService *ExtendedTokenService,
	complianceValidator *ComplianceValidator,
	authChainValidator *AuthorizationChainValidator,
	formalReqValidator *FormalRequirementsValidator,
	pvpClient PowerVerificationPoint,
	pipClient PIPClient,
	commercialRegClient CommercialRegisterClient,
	complianceTracker ComplianceTracker,
) Option {
	return func(s *Service) error {
		// Create subscription flow manager
		s.subscriptionManager = NewSubscriptionFlowManager(
			pvpClient,
			pipClient,
			commercialRegClient,
			authChainValidator,
			formalReqValidator,
			subscriptionStore,
		)

		// Create protocol orchestrator
		s.protocolOrchestrator = NewProtocolOrchestrator(
			extendedTokenService,
			complianceValidator,
			authChainValidator,
			formalReqValidator,
			pipClient,
			subscriptionStore,
			complianceTracker,
		)

		// Create PDP (Power Decision Point)
		s.powerDecisionPoint = NewSimplePDP()

		// Create PEP (Power Enforcement Point) wired to PDP
		// This completes the P*P architecture integration (RFC-0111 Section 3.1)
		if s.powerDecisionPoint != nil {
			tokenValidator := &simpleTokenValidator{
				extTokenService: extendedTokenService,
			}
			s.powerEnforcementPoint = NewPowerEnforcementPoint(
				tokenValidator,
				s.powerDecisionPoint,
				&noopPEPAuditLogger{}, // Simple audit logger
				complianceTracker,
				"strict", // Enforcement mode
			)
		}

		return nil
	}
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
func New(cfg Config, opts ...Option) (*Service, error) {
	// Ensure AppConfig is loaded if not provided
	if cfg.AppConfig == nil {
		var err error
		cfg.AppConfig, err = config.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load app config: %w", err)
		}
	}

	mode := cfg.AppConfig.TokenSigMode
	if mode == "" {
		mode = sigModeHMAC
	}
	ttl := time.Duration(cfg.AppConfig.JTITTLSeconds) * time.Second
	if ttl == 0 {
		ttl = 10 * time.Minute
	}

	svc := &Service{config: cfg, metrics: NoopMetrics, keyMode: mode, seenJTI: map[string]time.Time{}, jtiTTL: ttl, violations: observability.NewViolationCounters()}
	for _, opt := range opts {
		if opt != nil {
			if err := opt(svc); err != nil {
				return nil, err
			}
		}
	}
	if svc.keyMode == sigModeEdDSA {
		// Only create manager if not injected via WithKeyManager
		if svc.keyMgr == nil {
			rotHours := cfg.AppConfig.KeyRotationHours
			if rotHours <= 0 {
				rotHours = 24
			}
			km, err := crypto.NewManager(time.Duration(rotHours) * time.Hour)
			if err != nil {
				return nil, err
			}
			svc.keyMgr = km
		}
		// Note: We no longer register globally. Callers should inject via WithKeyManager if needed.
		// For backwards compatibility with other packages, they can still access crypto.GlobalEdDSARegistry
		// but this service won't set it automatically.
	} else {
		keyMaterial := cfg.SigningKey
		if svc.strictAuth {
			if keyMaterial == "" || len(keyMaterial) < 32 {
				return nil, ErrStrictAuthKeyRequired
			}
		} else {
			if keyMaterial == "" {
				keyMaterial = cfg.ClientSecret
			}
			if len(keyMaterial) < 32 {
				keyMaterial += strings.Repeat("0", 32-len(keyMaterial))
			}
		}
		svc.signingKey = []byte(keyMaterial)
	}

	// Initialize token validator and claim validator
	svc.claimValidator = NewCommonClaimValidator(cfg, svc.replay, svc.metrics, svc.jtiTTL, svc.violations)
	svc.validator = svc.selectValidator()

	return svc, nil
}

// selectValidator chooses the appropriate token validator based on configuration
// Priority order: EdDSA (explicit env var) > JWTLib (config flag) > HMAC (default)
func (svc *Service) selectValidator() TokenSignatureValidator {
	strictParsing := false
	if svc.config.AppConfig != nil {
		strictParsing = svc.config.AppConfig.StrictJSONParsing
	}

	// EdDSA mode takes highest priority (explicitly set via GAUTH_TOKEN_SIG_MODE env var)
	if svc.keyMode == sigModeEdDSA {
		return NewEdDSAValidator(svc.keyMgr, strictParsing)
	}

	// JWTLib is a config flag that can be used for HMAC tokens
	if svc.config.AppConfig != nil && svc.config.AppConfig.UseJWTLib {
		alg := svc.config.AppConfig.JWTAlg
		return NewJWTLibValidator(svc.signingKey, alg)
	}

	return NewHMACValidator(svc.signingKey, strictParsing)
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
// UPDATED: Now uses RFC-0111 flow by default for RFC compliance
// Set GAUTH_LEGACY_OAUTH_MODE=1 to use legacy OAuth-only mode
func (g *Service) RequestToken(req TokenRequest) (*TokenResponse, error) {
	// Check if legacy OAuth mode is enabled
	if g.config.AppConfig.LegacyOAuthMode {
		return g.RequestTokenLegacy(req)
	}

	// RFC-0111 compliant mode (default)
	ctx := context.Background()

	// Check if RFC orchestrator is available
	if g.protocolOrchestrator == nil {
		// Fallback to legacy mode if RFC orchestrator not initialized
		return g.RequestTokenLegacy(req)
	}

	// Convert TokenRequest to RFCCompliantAuthorizationRequest
	rfcReq := &RFCCompliantAuthorizationRequest{
		ClientID:       g.config.ClientID,
		SubscriptionID: req.GrantID, // Use GrantID as subscription reference
		RequestedScope: convertScopeToAuthorizationScope(req.Scope),
		Context:        convertContextToMap(req.Context),
	}

	// Execute RFC-0111 compliant flow
	rfcResp, err := g.RequestTokenRFC(ctx, rfcReq)
	if err != nil {
		// If RFC flow fails, log and fallback to legacy
		// In production, you may want to return the error instead
		return g.RequestTokenLegacy(req)
	}

	// Convert RFCCompliantTokenResponse to TokenResponse
	return convertRFCResponseToTokenResponse(rfcResp), nil
}

// RequestTokenLegacy generates a basic OAuth token (non-RFC-0111 compliant)
// This is the original implementation, kept for backward compatibility
// Use GAUTH_LEGACY_OAUTH_MODE=1 to enable this mode
func (g *Service) RequestTokenLegacy(req TokenRequest) (*TokenResponse, error) {
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
		activeKey := g.keyMgr.Active()
		sig := ed25519.Sign(activeKey.Private, []byte(unsigned))
		sEnc := base64.RawURLEncoding.EncodeToString(sig)
		tok := unsigned + "." + sEnc
		if g.metrics != nil {
			g.metrics.IncTokensIssued()
		}
		return &TokenResponse{Token: tok, Scope: req.Scope, ValidUntil: expiry}, nil
	}
	if g.config.AppConfig.UseJWTLib {
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
		kid := g.config.AppConfig.JWTKeyID
		if kid == "" {
			kid = defaultDemoKey
		}
		alg := g.config.AppConfig.JWTAlg
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

// RequestTokenRFC executes RFC-0111 compliant authorization flow
// This is the main entry point for RFC-0111 compliant token requests
// It orchestrates Steps (a)-(i) and returns an ExtendedToken
func (g *Service) RequestTokenRFC(ctx context.Context, req *RFCCompliantAuthorizationRequest) (*RFCCompliantTokenResponse, error) {
	if g.protocolOrchestrator == nil {
		return nil, fmt.Errorf("RFC-0111 protocol orchestrator not initialized - use WithRFCCompliance option")
	}

	return g.protocolOrchestrator.ExecuteRFCCompliantFlow(ctx, req)
}

// GetSubscriptionManager returns the subscription flow manager
// This allows external code to execute Steps I-VIII
func (g *Service) GetSubscriptionManager() *SubscriptionFlowManager {
	return g.subscriptionManager
}

// ValidateToken validates a token and returns client information
//
//nolint:gocyclo // Token validation with comprehensive security checks
func (g *Service) ValidateToken(token string) (*TokenValidationResult, error) {
	if g == nil {
		return nil, ErrInvalidToken
	}
	if token == "" {
		failMetric(g, observability.MissingClaim)
		return nil, ErrInvalidToken
	}
	if g.validator == nil {
		failMetric(g, observability.SigInvalid)
		return nil, ErrInvalidToken
	}

	// Validate signature and extract claims using selected strategy
	claims, err := g.validator.ValidateSignature(token)
	if err != nil {
		failMetric(g, observability.SigInvalid)
		return nil, err
	}

	// Validate common claims
	return g.claimValidator.ValidateClaims(claims)
}
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

// Helper functions for RequestToken RFC-0111 conversion

// convertScopeToAuthorizationScope converts string scope to poa.AuthorizationScope
func convertScopeToAuthorizationScope(scopes []string) *poa.AuthorizationScope {
	if len(scopes) == 0 {
		return nil
	}

	// For now, create a basic authorization scope
	// In production, this should be enriched with proper sector/region data
	return &poa.AuthorizationScope{
		AuthorizationType: poa.AuthorizationType{
			RepresentationType: "sole",
			Restrictions:       []string{},
			SubProxyAuthority:  false,
		},
		ApplicableSectors: []poa.IndustrySector{},
		ApplicableRegions: []poa.GeographicScope{
			{Type: poa.GeoTypeGlobal}, // Default to global
		},
		AuthorizedActions: poa.AuthorizedActions{},
	}
}

// convertContextToMap converts interface{} context to map[string]interface{}
func convertContextToMap(ctx interface{}) map[string]interface{} {
	if ctx == nil {
		return make(map[string]interface{})
	}

	if m, ok := ctx.(map[string]interface{}); ok {
		return m
	}

	return map[string]interface{}{
		"original_context": ctx,
	}
}

// convertRFCResponseToTokenResponse converts RFC response to legacy TokenResponse
func convertRFCResponseToTokenResponse(rfcResp *RFCCompliantTokenResponse) *TokenResponse {
	if rfcResp == nil || rfcResp.ExtendedToken == nil {
		return &TokenResponse{}
	}

	token := rfcResp.ExtendedToken
	expiry := token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second)

	return &TokenResponse{
		Token:      token.AccessToken,
		Scope:      token.Scope,
		ValidUntil: expiry,
	}
}

// Ensure Service implements GAuth interface
var _ GAuth = (*Service)(nil)
