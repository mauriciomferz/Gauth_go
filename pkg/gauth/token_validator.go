package gauth

import (
	"strings"
	"sync"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/observability"
)

// TokenSignatureValidator validates tokens using a specific strategy
type TokenSignatureValidator interface {
	// ValidateSignature validates the token signature and extracts claims
	// Returns claims map if signature is valid, error otherwise
	ValidateSignature(token string) (claims map[string]any, err error)

	// Name returns the validator name for logging and debugging
	Name() string
}

// CommonClaimValidator handles validation logic shared across all strategies
type CommonClaimValidator struct {
	config     Config
	replay     ReplayStore
	metrics    Metrics
	seenJTI    map[string]time.Time
	jtiMutex   sync.RWMutex
	jtiTTL     time.Duration
	violations *observability.ViolationCounters
}

// NewCommonClaimValidator creates a new common claim validator
func NewCommonClaimValidator(cfg Config, replay ReplayStore, metrics Metrics, jtiTTL time.Duration, violations *observability.ViolationCounters) *CommonClaimValidator {
	return &CommonClaimValidator{
		config:     cfg,
		replay:     replay,
		metrics:    metrics,
		seenJTI:    make(map[string]time.Time),
		jtiTTL:     jtiTTL,
		violations: violations,
	}
}

// ValidateClaims validates common JWT claims (exp, nbf, iss, jti, aud, scope, sub)
func (v *CommonClaimValidator) ValidateClaims(claims map[string]any) (*TokenValidationResult, error) {
	// Validate expiration
	if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		v.failMetric(observability.Expired)
		return nil, ErrTokenExpired
	}

	// Validate not-before
	if nbf, ok := claims["nbf"].(float64); ok && time.Now().Unix() < int64(nbf) {
		v.failMetric(observability.NotYetValid)
		return nil, ErrInvalidToken
	}

	// Validate issuer
	if iss, ok := claims["iss"].(string); ok && iss != "" && v.config.AuthServerURL != "" && iss != v.config.AuthServerURL {
		v.failMetric(observability.IssuerMismatch)
		return nil, ErrInvalidToken
	}

	// Validate JTI and check for replay
	if jti, ok := claims["jti"].(string); ok {
		if jti == "" {
			v.failMetric(observability.MissingClaim)
			return nil, ErrInvalidToken
		}
		if v.replay != nil {
			if err := v.replay.CheckAndStore(jti); err != nil {
				v.failMetric(observability.ReplayDetected)
				return nil, ErrInvalidToken
			}
		} else {
			if v.isReplay(jti) {
				v.failMetric(observability.ReplayDetected)
				return nil, ErrInvalidToken
			}
			v.storeJTI(jti)
		}
	} else {
		v.failMetric(observability.MissingClaim)
		return nil, ErrInvalidToken
	}

	// Validate audience
	if audVal, ok := claims["aud"]; ok {
		if !audClaimMatches(audVal, v.config.Audience) {
			v.failMetric(observability.AudienceMismatch)
			return nil, ErrInvalidToken
		}
	}

	// Extract scope and subject
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

	if v.metrics != nil {
		v.metrics.IncTokenValidations()
	}

	return &TokenValidationResult{ClientID: clientID, Scope: scopes, Valid: true}, nil
}

// isReplay checks if JTI has been seen before
func (v *CommonClaimValidator) isReplay(jti string) bool {
	v.jtiMutex.RLock()
	defer v.jtiMutex.RUnlock()
	_, seen := v.seenJTI[jti]
	return seen
}

// storeJTI stores JTI for replay protection
func (v *CommonClaimValidator) storeJTI(jti string) {
	v.jtiMutex.Lock()
	defer v.jtiMutex.Unlock()
	v.seenJTI[jti] = time.Now()

	// Clean up old JTIs
	cutoff := time.Now().Add(-v.jtiTTL)
	for id, ts := range v.seenJTI {
		if ts.Before(cutoff) {
			delete(v.seenJTI, id)
		}
	}
}

// failMetric records a validation failure metric
func (v *CommonClaimValidator) failMetric(reason observability.ViolationCategory) {
	if v.violations != nil {
		v.violations.Inc(reason)
	}
	if v.metrics != nil {
		v.metrics.IncTokenValidationFailures()
	}
}
