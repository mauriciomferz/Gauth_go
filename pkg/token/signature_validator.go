package token

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	validationAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gauthtoken_validation_attempts_total",
			Help: "Total number of token validation attempts.",
		},
		[]string{"algorithm", "status"},
	)
	validationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gauthtoken_validation_duration_seconds",
			Help:    "Histogram of token validation durations.",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 15),
		},
		[]string{"algorithm"},
	)
)

// TokenSignatureValidator defines the interface for validating a token's signature.
type TokenSignatureValidator interface {
	Validate(ctx context.Context, tokenString string) error
}

// HMACValidator validates HMAC-based JWT signatures (e.g., HS256).
type HMACValidator struct {
	key []byte
}

// NewHMACValidator creates a new HMACValidator.
func NewHMACValidator(key []byte) *HMACValidator {
	return &HMACValidator{key: key}
}

// Validate checks the token signature.
func (v *HMACValidator) Validate(ctx context.Context, tokenString string) error {
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.key, nil
	})
	if err != nil {
		return &ValidationError{Code: "invalid_signature", Message: err.Error()}
	}
	return nil
}

// RSAVerifier validates RSA-based JWT signatures (e.g., RS256).
type RSAVerifier struct {
	publicKey *rsa.PublicKey
}

// NewRSAVerifier creates a new RSAVerifier.
func NewRSAVerifier(publicKey *rsa.PublicKey) *RSAVerifier {
	return &RSAVerifier{publicKey: publicKey}
}

// Validate checks the token signature.
func (v *RSAVerifier) Validate(ctx context.Context, tokenString string) error {
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.publicKey, nil
	})
	if err != nil {
		return &ValidationError{Code: "invalid_signature", Message: err.Error()}
	}
	return nil
}

// ECDSAValidator validates ECDSA-based JWT signatures (e.g., ES256).
type ECDSAValidator struct {
	publicKey *ecdsa.PublicKey
}

// NewECDSAValidator creates a new ECDSAValidator.
func NewECDSAValidator(publicKey *ecdsa.PublicKey) *ECDSAValidator {
	return &ECDSAValidator{publicKey: publicKey}
}

// Validate checks the token signature.
func (v *ECDSAValidator) Validate(ctx context.Context, tokenString string) error {
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.publicKey, nil
	})
	if err != nil {
		return &ValidationError{Code: "invalid_signature", Message: err.Error()}
	}
	return nil
}

// ValidatorRegistry holds multiple validators and selects one based on the token's algorithm.
type ValidatorRegistry struct {
	mu         sync.RWMutex
	validators map[string]TokenSignatureValidator
	logger     *zap.Logger
	metrics    *Monitor
}

// NewValidatorRegistry creates a new ValidatorRegistry.
func NewValidatorRegistry() *ValidatorRegistry {
	logger, _ := zap.NewProduction()
	return &ValidatorRegistry{
		validators: make(map[string]TokenSignatureValidator),
		logger:     logger,
		metrics:    NewMonitor(),
	}
}

// Register adds a validator for a given algorithm.
func (r *ValidatorRegistry) Register(alg string, validator TokenSignatureValidator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.validators[alg] = validator
}

// Validate parses the token to determine the algorithm and uses the corresponding validator.
func (r *ValidatorRegistry) Validate(ctx context.Context, tokenString string) error {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		r.logger.Warn("Failed to parse token unverified", zap.Error(err))
		validationAttempts.WithLabelValues("unknown", "malformed").Inc()
		return &ValidationError{Code: "malformed_token", Message: err.Error()}
	}

	alg, ok := token.Header["alg"].(string)
	if !ok {
		validationAttempts.WithLabelValues("unknown", "missing_alg").Inc()
		return &ValidationError{Code: "missing_alg", Message: "missing 'alg' header"}
	}

	r.mu.RLock()
	validator, exists := r.validators[alg]
	r.mu.RUnlock()

	if !exists {
		r.logger.Error("Unsupported signing method", zap.String("algorithm", alg))
		validationAttempts.WithLabelValues(alg, "unsupported").Inc()
		return &ValidationError{Code: "unsupported_algorithm", Message: fmt.Sprintf("unsupported signing method: %s", alg)}
	}

	startTime := time.Now()
	err = validator.Validate(ctx, tokenString)
	duration := time.Since(startTime)
	validationDuration.WithLabelValues(alg).Observe(duration.Seconds())

	if err != nil {
		r.logger.Warn("Signature validation failed",
			zap.String("algorithm", alg),
			zap.Error(err),
			zap.Duration("duration", duration),
		)
		validationAttempts.WithLabelValues(alg, "failed").Inc()
		r.metrics.IncRevoked() // Using revoked as a proxy for failed validation
	} else {
		r.logger.Info("Signature validation successful",
			zap.String("algorithm", alg),
			zap.Duration("duration", duration),
		)
		validationAttempts.WithLabelValues(alg, "success").Inc()
		r.metrics.IncValidated()
	}

	return err
}
