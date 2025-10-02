// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// ConcurrentTokenIssuer prevents race conditions in token issuance
type ConcurrentTokenIssuer struct {
	mu              sync.RWMutex
	issuanceQueue   map[string]*IssuanceRequest
	rateLimiter     map[string]*RateLimit
	noncePrevention map[string]time.Time
	sequentialNonce uint64
	sequentialMutex sync.Mutex
}

// IssuanceRequest represents a token issuance request
type IssuanceRequest struct {
	RequestID  string
	ClientID   string
	Scopes     []string
	Timestamp  time.Time
	Status     string
	ResultChan chan *IssuanceResult
}

// IssuanceResult contains the result of token issuance
type IssuanceResult struct {
	Token *token.Token
	Error error
}

// RateLimit tracks rate limiting per client
type RateLimit struct {
	Count       int
	WindowStart time.Time
	MaxRequests int
	WindowSize  time.Duration
}

// NewConcurrentTokenIssuer creates a race-condition-safe token issuer
func NewConcurrentTokenIssuer() *ConcurrentTokenIssuer {
	return &ConcurrentTokenIssuer{
		issuanceQueue:   make(map[string]*IssuanceRequest),
		rateLimiter:     make(map[string]*RateLimit),
		noncePrevention: make(map[string]time.Time),
	}
}

// IssueTokenSafe issues tokens with race condition prevention
func (c *ConcurrentTokenIssuer) IssueTokenSafe(
	ctx context.Context, clientID string, scopes []string,
) (*token.Token, error) {
	// Generate unique request ID with nonce
	requestID, err := c.generateSecureRequestID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate request ID: %w", err)
	}

	// Check for replay attacks
	if err := c.checkNonceReplay(requestID); err != nil {
		return nil, fmt.Errorf("nonce replay detected: %w", err)
	}

	// Apply rate limiting
	if err := c.checkRateLimit(clientID); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Create issuance request
	request := &IssuanceRequest{
		RequestID:  requestID,
		ClientID:   clientID,
		Scopes:     scopes,
		Timestamp:  time.Now(),
		Status:     "pending",
		ResultChan: make(chan *IssuanceResult, 1),
	}

	// Serialize token issuance
	c.mu.Lock()

	// Check for duplicate requests
	if existing, exists := c.issuanceQueue[clientID]; exists {
		c.mu.Unlock()
		if time.Since(existing.Timestamp) < time.Second*30 {
			return nil, fmt.Errorf("duplicate token request in progress")
		}
		// Clean up stale request
		delete(c.issuanceQueue, clientID)
		c.mu.Lock()
	}

	c.issuanceQueue[clientID] = request
	c.mu.Unlock()

	// Process request asynchronously but wait for result
	go c.processIssuanceRequest(ctx, request)

	// Wait for result with timeout
	select {
	case result := <-request.ResultChan:
		c.cleanupRequest(clientID)
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Token, nil
	case <-time.After(time.Second * 10):
		c.cleanupRequest(clientID)
		return nil, fmt.Errorf("token issuance timeout")
	case <-ctx.Done():
		c.cleanupRequest(clientID)
		return nil, fmt.Errorf("token issuance cancelled: %w", ctx.Err())
	}
}

// processIssuanceRequest processes the token issuance request
func (c *ConcurrentTokenIssuer) processIssuanceRequest(
	ctx context.Context, request *IssuanceRequest,
) {
	// Simulate token creation with timing attack prevention
	start := time.Now()

	token, err := c.createToken(ctx, request.ClientID, request.Scopes)

	// Constant-time response to prevent timing attacks
	elapsed := time.Since(start)
	if elapsed < time.Millisecond*100 {
		time.Sleep(time.Millisecond*100 - elapsed)
	}

	result := &IssuanceResult{
		Token: token,
		Error: err,
	}

	select {
	case request.ResultChan <- result:
	default:
		// Channel full or closed
	}
}

// createToken creates the actual token
func (c *ConcurrentTokenIssuer) createToken(
	ctx context.Context, clientID string, scopes []string,
) (*token.Token, error) {
	// Generate secure token value
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &token.Token{
		ID:        c.generateTokenID(),
		Value:     hex.EncodeToString(tokenBytes),
		Type:      token.Access,
		Subject:   clientID,
		Scopes:    scopes,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Audience:  []string{"gauth-system"},
		Issuer:    "gauth-server",
	}, nil
}

// generateSecureRequestID generates a cryptographically secure request ID
func (c *ConcurrentTokenIssuer) generateSecureRequestID() (string, error) {
	// Use sequential nonce for uniqueness
	c.sequentialMutex.Lock()
	c.sequentialNonce++
	nonce := c.sequentialNonce
	c.sequentialMutex.Unlock()

	// Add random component
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d-%s-%d",
		nonce, hex.EncodeToString(randomBytes), time.Now().UnixNano()), nil
}

// generateTokenID generates unique token ID
func (c *ConcurrentTokenIssuer) generateTokenID() string {
	c.sequentialMutex.Lock()
	defer c.sequentialMutex.Unlock()

	c.sequentialNonce++
	return fmt.Sprintf("token-%d-%d", c.sequentialNonce, time.Now().UnixNano())
}

// checkNonceReplay prevents nonce replay attacks
func (c *ConcurrentTokenIssuer) checkNonceReplay(requestID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if lastSeen, exists := c.noncePrevention[requestID]; exists {
		return fmt.Errorf("nonce already used at %v", lastSeen)
	}

	c.noncePrevention[requestID] = time.Now()

	// Clean up old nonces (older than 1 hour)
	cutoff := time.Now().Add(-time.Hour)
	for nonce, timestamp := range c.noncePrevention {
		if timestamp.Before(cutoff) {
			delete(c.noncePrevention, nonce)
		}
	}

	return nil
}

// checkRateLimit enforces rate limiting per client
func (c *ConcurrentTokenIssuer) checkRateLimit(clientID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	limit, exists := c.rateLimiter[clientID]
	if !exists {
		limit = &RateLimit{
			Count:       0,
			WindowStart: now,
			MaxRequests: 10, // 10 requests per minute
			WindowSize:  time.Minute,
		}
		c.rateLimiter[clientID] = limit
	}

	// Reset window if expired
	if now.Sub(limit.WindowStart) > limit.WindowSize {
		limit.Count = 0
		limit.WindowStart = now
	}

	// Check limit
	if limit.Count >= limit.MaxRequests {
		return fmt.Errorf("rate limit exceeded: %d requests in %v",
			limit.Count, limit.WindowSize)
	}

	limit.Count++
	return nil
}

// cleanupRequest removes request from queue
func (c *ConcurrentTokenIssuer) cleanupRequest(clientID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.issuanceQueue, clientID)
}
