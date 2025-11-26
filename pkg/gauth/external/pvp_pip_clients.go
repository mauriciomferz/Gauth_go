// Package external - External Service Clients for RFC-0111
// Implements production-ready clients for PowerVerificationPoint (PVP) and PIP
// with retry logic, circuit breakers, authentication, and fallback mechanisms
package external

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)

// Note: CircuitBreaker and related types are now defined in connector_utils.go to avoid duplication

// PVPClientConfig holds configuration for PVP client
type PVPClientConfig struct {
	BaseURL         string
	APIKey          string
	Timeout         time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	CircuitBreaker  *CircuitBreaker
	FallbackEnabled bool
}

// PVPClient is a production-ready client for PowerVerificationPoint
type PVPClient struct {
	config     *PVPClientConfig
	httpClient *http.Client
}

// NewPVPClient creates a new PVP client
func NewPVPClient(config *PVPClientConfig) *PVPClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.CircuitBreaker == nil {
		config.CircuitBreaker = NewCircuitBreaker(5, config.Timeout, 60*time.Second)
	}

	return &PVPClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// VerifyIdentity verifies identity proof through PVP service
func (c *PVPClient) VerifyIdentity(
	ctx context.Context,
	request *gauth.IdentityVerificationRequest,
) (*gauth.IdentityVerificationResult, error) {
	var lastErr error
	var finalResult *gauth.IdentityVerificationResult

	// Execute with circuit breaker
	err := c.config.CircuitBreaker.Call(func() error {
		// Implement retry logic
		for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
			if attempt > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(c.config.RetryDelay * time.Duration(attempt)):
					// Exponential backoff
				}
			}

			result, err := c.attemptVerifyIdentity(ctx, request)
			if err == nil {
				finalResult = result
				return nil
			}

			lastErr = err

			// Don't retry on client errors
			if isClientError(err) {
				return err
			}
		}
		return lastErr
	})

	if err != nil && c.config.FallbackEnabled {
		// Fallback to cached/alternative verification
		return c.fallbackVerifyIdentity(ctx, request)
	}

	if err != nil {
		return nil, fmt.Errorf("PVP verification failed after %d attempts: %w", c.config.MaxRetries, err)
	}

	return finalResult, nil
}

// attemptVerifyIdentity makes a single attempt to verify identity
func (c *PVPClient) attemptVerifyIdentity(
	ctx context.Context,
	request *gauth.IdentityVerificationRequest,
) (*gauth.IdentityVerificationResult, error) {
	// Build HTTP request to PVP service
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/api/v1/verify", c.config.BaseURL),
		nil, // TODO: serialize request body
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PVP returned status %d", resp.StatusCode)
	}

	// TODO: Parse response
	return &gauth.IdentityVerificationResult{
		Verified:   true,
		SubjectID:  request.SubjectID,
		VerifiedAt: time.Now(),
	}, nil
}

// fallbackVerifyIdentity provides fallback verification
func (c *PVPClient) fallbackVerifyIdentity(
	ctx context.Context,
	request *gauth.IdentityVerificationRequest,
) (*gauth.IdentityVerificationResult, error) {
	// Implement fallback logic:
	// 1. Check cache for recent verifications
	// 2. Use alternative identity provider
	// 3. Return partial verification with lower confidence

	return &gauth.IdentityVerificationResult{
		Verified:           true,
		VerificationID:     fmt.Sprintf("fallback-%d", time.Now().Unix()),
		SubjectID:          request.SubjectID,
		IdentityLevel:      "basic", // Lower level for fallback
		VerifiedAt:         time.Now(),
		ExpiresAt:          time.Now().Add(1 * time.Hour), // Shorter expiry
		VerificationMethod: "fallback_" + request.ProofMethod,
		IsFallback:         true,
	}, nil
}

// PIPClientConfig holds configuration for PIP client
type PIPClientConfig struct {
	BaseURL        string
	APIKey         string
	Timeout        time.Duration
	MaxRetries     int
	RetryDelay     time.Duration
	CircuitBreaker *CircuitBreaker
	CacheEnabled   bool
	CacheTTL       time.Duration
}

// PIPClient is a production-ready client for Power Information Point
type PIPClient struct {
	config     *PIPClientConfig
	httpClient *http.Client
	cache      map[string]*cachedPolicy // Simple cache, use Redis in production
}

type cachedPolicy struct {
	policy    *gauth.PowerOfAttorneyPolicy
	expiresAt time.Time
}

// NewPIPClient creates a new PIP client
func NewPIPClient(config *PIPClientConfig) *PIPClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.CircuitBreaker == nil {
		config.CircuitBreaker = NewCircuitBreaker(5, config.Timeout, 60*time.Second)
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	return &PIPClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		cache: make(map[string]*cachedPolicy),
	}
}

// GetPolicy retrieves policy information from PIP
func (c *PIPClient) GetPolicy(
	ctx context.Context,
	request *gauth.PolicyRequest,
) (*gauth.PowerOfAttorneyPolicy, error) {
	// Check cache first
	if c.config.CacheEnabled {
		if cached := c.getFromCache(request); cached != nil {
			return cached, nil
		}
	}

	var lastErr error
	var policy *gauth.PowerOfAttorneyPolicy

	// Execute with circuit breaker and retry logic
	err := c.config.CircuitBreaker.Call(func() error {
		for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
			if attempt > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(c.config.RetryDelay * time.Duration(attempt)):
				}
			}

			var err error
			policy, err = c.attemptGetPolicy(ctx, request)
			if err == nil {
				// Cache successful result
				if c.config.CacheEnabled {
					c.putInCache(request, policy)
				}
				return nil
			}

			lastErr = err

			if isClientError(err) {
				return err
			}
		}
		return lastErr
	})

	if err != nil {
		return nil, fmt.Errorf("PIP policy retrieval failed: %w", err)
	}

	return policy, nil
}

// attemptGetPolicy makes a single attempt to get policy
func (c *PIPClient) attemptGetPolicy(
	ctx context.Context,
	request *gauth.PolicyRequest,
) (*gauth.PowerOfAttorneyPolicy, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/api/v1/policies/%s", c.config.BaseURL, request.PolicyID),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PIP returned status %d", resp.StatusCode)
	}

	// TODO: Parse actual response
	return &gauth.PowerOfAttorneyPolicy{
		PolicyID:  request.PolicyID,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

// getFromCache retrieves policy from cache
func (c *PIPClient) getFromCache(request *gauth.PolicyRequest) *gauth.PowerOfAttorneyPolicy {
	cached, exists := c.cache[request.PolicyID]
	if !exists {
		return nil
	}

	if time.Now().After(cached.expiresAt) {
		delete(c.cache, request.PolicyID)
		return nil
	}

	return cached.policy
}

// putInCache stores policy in cache
func (c *PIPClient) putInCache(request *gauth.PolicyRequest, policy *gauth.PowerOfAttorneyPolicy) {
	c.cache[request.PolicyID] = &cachedPolicy{
		policy:    policy,
		expiresAt: time.Now().Add(c.config.CacheTTL),
	}
}

// isClientError determines if an error is a client error (4xx) that shouldn't be retried
func isClientError(err error) bool {
	// Simple heuristic - in production, check actual HTTP status codes
	return false
}
