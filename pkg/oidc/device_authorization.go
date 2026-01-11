// Package oidc - Device Authorization Grant
// Implements RFC 8628 OAuth 2.0 Device Authorization Grant
// For input-constrained devices (smart TVs, IoT devices, CLI tools)
package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"sync"
	"time"
)

// DeviceAuthorizationRequest represents a device authorization request per RFC 8628.
type DeviceAuthorizationRequest struct {
	// ClientID identifies the client
	ClientID string `json:"client_id"`

	// Scope is the requested scope (optional)
	Scope string `json:"scope,omitempty"`
}

// DeviceAuthorizationResponse represents the device authorization response per RFC 8628 Section 3.2.
type DeviceAuthorizationResponse struct {
	// DeviceCode is the device verification code
	DeviceCode string `json:"device_code"`

	// UserCode is the end-user verification code (human-readable)
	UserCode string `json:"user_code"`

	// VerificationURI is where the user should visit
	VerificationURI string `json:"verification_uri"`

	// VerificationURIComplete includes the user code (optional)
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`

	// ExpiresIn is seconds until device code expires
	ExpiresIn int `json:"expires_in"`

	// Interval is the minimum polling interval in seconds
	Interval int `json:"interval"`
}

// DeviceTokenRequest represents a token request using device code per RFC 8628 Section 3.4.
type DeviceTokenRequest struct {
	// GrantType must be "urn:ietf:params:oauth:grant-type:device_code"
	GrantType string `json:"grant_type"`

	// DeviceCode from the authorization response
	DeviceCode string `json:"device_code"`

	// ClientID identifies the client
	ClientID string `json:"client_id"`
}

// DeviceCodeEntry stores device code information.
type DeviceCodeEntry struct {
	DeviceCode string
	UserCode   string
	ClientID   string
	Scope      []string
	IssuedAt   time.Time
	ExpiresAt  time.Time
	LastPolled time.Time
	PollCount  int

	// Authorization status
	Status       DeviceCodeStatus
	AuthorizedBy string // User ID who authorized
	AuthorizedAt time.Time

	// Token information (after authorization)
	AccessToken   string
	RefreshToken  string
	IDToken       string
	TokenIssuedAt time.Time
}

// DeviceCodeStatus represents the authorization status.
type DeviceCodeStatus string

const (
	DeviceCodePending    DeviceCodeStatus = "pending"
	DeviceCodeAuthorized DeviceCodeStatus = "authorized"
	DeviceCodeDenied     DeviceCodeStatus = "denied"
	DeviceCodeExpired    DeviceCodeStatus = "expired"
)

// RFC 8628 specific error codes (in addition to standard OAuth2 errors in types.go)
const (
	ErrorAuthorizationPending = "authorization_pending"
	ErrorSlowDown             = "slow_down"
	ErrorDeviceCodeExpired    = "expired_token"
	// ErrorAccessDenied already defined in types.go
)

// DeviceAuthorizationConfig configures device authorization behavior.
type DeviceAuthorizationConfig struct {
	// VerificationURI is the base URI where users verify codes
	VerificationURI string

	// DeviceCodeLifetime is how long device codes are valid
	DeviceCodeLifetime time.Duration

	// PollingInterval is the minimum time between token requests
	PollingInterval time.Duration

	// UserCodeLength is the length of user codes (default: 8)
	UserCodeLength int

	// UserCodeCharset defines allowed characters in user codes
	UserCodeCharset string

	// MaxPollAttempts limits polling attempts (0 = unlimited)
	MaxPollAttempts int

	// SlowDownPenalty adds extra wait time for slow_down errors
	SlowDownPenalty time.Duration
}

// DefaultDeviceAuthorizationConfig returns secure default configuration.
func DefaultDeviceAuthorizationConfig() *DeviceAuthorizationConfig {
	return &DeviceAuthorizationConfig{
		VerificationURI:    "https://example.com/device",
		DeviceCodeLifetime: 15 * time.Minute,
		PollingInterval:    5 * time.Second,
		UserCodeLength:     8,
		UserCodeCharset:    "BCDFGHJKLMNPQRSTVWXZ", // Removes ambiguous characters
		MaxPollAttempts:    100,
		SlowDownPenalty:    5 * time.Second,
	}
}

// DeviceAuthorizationService manages device authorization flow.
type DeviceAuthorizationService struct {
	config       *DeviceAuthorizationConfig
	deviceCodes  map[string]*DeviceCodeEntry // deviceCode -> entry
	userCodes    map[string]*DeviceCodeEntry // userCode -> entry
	mu           sync.RWMutex
	cleanupTimer *time.Ticker
	stopCleanup  chan struct{}
	wg           sync.WaitGroup
}

// NewDeviceAuthorizationService creates a new device authorization service.
func NewDeviceAuthorizationService(config *DeviceAuthorizationConfig) *DeviceAuthorizationService {
	if config == nil {
		config = DefaultDeviceAuthorizationConfig()
	}

	service := &DeviceAuthorizationService{
		config:      config,
		deviceCodes: make(map[string]*DeviceCodeEntry),
		userCodes:   make(map[string]*DeviceCodeEntry),
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine (runs every minute)
	service.cleanupTimer = time.NewTicker(time.Minute)
	service.wg.Add(1)
	go service.cleanupLoop()

	return service
}

// AuthorizeDevice initiates device authorization flow per RFC 8628 Section 3.1-3.2.
func (s *DeviceAuthorizationService) AuthorizeDevice(
	ctx context.Context,
	req *DeviceAuthorizationRequest,
) (*DeviceAuthorizationResponse, error) {
	if err := s.validateAuthorizationRequest(req); err != nil {
		return nil, err
	}

	// Generate device code (long, secure, opaque)
	deviceCode, err := s.generateDeviceCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate device code: %w", err)
	}

	// Generate user code (short, human-readable)
	userCode, err := s.generateUserCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate user code: %w", err)
	}

	// Parse scopes
	scopes := []string{}
	if req.Scope != "" {
		scopes = strings.Fields(req.Scope)
	}

	// Store device code entry
	now := time.Now()
	entry := &DeviceCodeEntry{
		DeviceCode: deviceCode,
		UserCode:   userCode,
		ClientID:   req.ClientID,
		Scope:      scopes,
		IssuedAt:   now,
		ExpiresAt:  now.Add(s.config.DeviceCodeLifetime),
		Status:     DeviceCodePending,
		PollCount:  0,
	}

	s.mu.Lock()
	s.deviceCodes[deviceCode] = entry
	s.userCodes[userCode] = entry
	s.mu.Unlock()

	// Build response per RFC 8628 Section 3.2
	response := &DeviceAuthorizationResponse{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		VerificationURI: s.config.VerificationURI,
		ExpiresIn:       int(s.config.DeviceCodeLifetime.Seconds()),
		Interval:        int(s.config.PollingInterval.Seconds()),
	}

	// Add complete verification URI if possible
	response.VerificationURIComplete = fmt.Sprintf("%s?user_code=%s", s.config.VerificationURI, userCode)

	return response, nil
}

// PollToken polls for token using device code per RFC 8628 Section 3.4-3.5.
func (s *DeviceAuthorizationService) PollToken(ctx context.Context, req *DeviceTokenRequest) (*DeviceTokenResponse, error) {
	if err := s.validateTokenRequest(req); err != nil {
		return nil, err
	}

	s.mu.Lock()
	entry, exists := s.deviceCodes[req.DeviceCode]
	if !exists {
		s.mu.Unlock()
		return nil, &DeviceAuthorizationError{
			ErrorCode:        ErrorDeviceCodeExpired,
			ErrorDescription: "device code not found or expired",
		}
	}

	// Update poll tracking
	now := time.Now()
	entry.PollCount++
	timeSinceLastPoll := now.Sub(entry.LastPolled)
	entry.LastPolled = now

	// Check expiration
	if now.After(entry.ExpiresAt) {
		entry.Status = DeviceCodeExpired
		s.mu.Unlock()
		return nil, &DeviceAuthorizationError{
			ErrorCode:        ErrorDeviceCodeExpired,
			ErrorDescription: "device code has expired",
		}
	}

	// Check max poll attempts
	if s.config.MaxPollAttempts > 0 && entry.PollCount > s.config.MaxPollAttempts {
		s.mu.Unlock()
		return nil, &DeviceAuthorizationError{
			ErrorCode:        ErrorAccessDenied,
			ErrorDescription: "maximum poll attempts exceeded",
		}
	}

	// Check polling interval (slow_down)
	minInterval := s.config.PollingInterval
	if timeSinceLastPoll < minInterval && entry.PollCount > 1 {
		s.mu.Unlock()
		return nil, &DeviceAuthorizationError{
			ErrorCode:        ErrorSlowDown,
			ErrorDescription: fmt.Sprintf("polling too fast, wait at least %d seconds", int(minInterval.Seconds())),
		}
	}

	// Check authorization status
	switch entry.Status {
	case DeviceCodePending:
		s.mu.Unlock()
		return nil, &DeviceAuthorizationError{
			ErrorCode:        ErrorAuthorizationPending,
			ErrorDescription: "user has not yet authorized the device",
		}

	case DeviceCodeDenied:
		s.mu.Unlock()
		return nil, &DeviceAuthorizationError{
			ErrorCode:        ErrorAccessDenied,
			ErrorDescription: "user denied the authorization request",
		}

	case DeviceCodeAuthorized:
		// Return tokens
		response := &DeviceTokenResponse{
			AccessToken:  entry.AccessToken,
			TokenType:    "Bearer",
			ExpiresIn:    3600, // 1 hour
			RefreshToken: entry.RefreshToken,
			IDToken:      entry.IDToken,
			Scope:        strings.Join(entry.Scope, " "),
		}
		s.mu.Unlock()
		return response, nil

	default:
		s.mu.Unlock()
		return nil, fmt.Errorf("unknown device code status: %s", entry.Status)
	}
}

// VerifyUserCode retrieves device code information by user code.
func (s *DeviceAuthorizationService) VerifyUserCode(ctx context.Context, userCode string) (*DeviceCodeEntry, error) {
	userCode = strings.ToUpper(strings.TrimSpace(userCode))

	s.mu.RLock()
	entry, exists := s.userCodes[userCode]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("invalid user code")
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		return nil, fmt.Errorf("user code has expired")
	}

	// Return a copy to avoid race conditions
	entryCopy := *entry
	return &entryCopy, nil
}

// ApproveAuthorization approves a device authorization request.
func (s *DeviceAuthorizationService) ApproveAuthorization(
	ctx context.Context,
	userCode string,
	userID string,
	accessToken string,
	refreshToken string,
	idToken string,
) error {
	userCode = strings.ToUpper(strings.TrimSpace(userCode))

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.userCodes[userCode]
	if !exists {
		return fmt.Errorf("invalid user code")
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		return fmt.Errorf("user code has expired")
	}

	// Check if already processed
	if entry.Status != DeviceCodePending {
		return fmt.Errorf("authorization request already processed")
	}

	// Approve authorization
	entry.Status = DeviceCodeAuthorized
	entry.AuthorizedBy = userID
	entry.AuthorizedAt = time.Now()
	entry.AccessToken = accessToken
	entry.RefreshToken = refreshToken
	entry.IDToken = idToken
	entry.TokenIssuedAt = time.Now()

	return nil
}

// DenyAuthorization denies a device authorization request.
func (s *DeviceAuthorizationService) DenyAuthorization(ctx context.Context, userCode string, userID string) error {
	userCode = strings.ToUpper(strings.TrimSpace(userCode))

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.userCodes[userCode]
	if !exists {
		return fmt.Errorf("invalid user code")
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		return fmt.Errorf("user code has expired")
	}

	// Check if already processed
	if entry.Status != DeviceCodePending {
		return fmt.Errorf("authorization request already processed")
	}

	// Deny authorization
	entry.Status = DeviceCodeDenied
	entry.AuthorizedBy = userID
	entry.AuthorizedAt = time.Now()

	return nil
}

// validateAuthorizationRequest validates the device authorization request.
func (s *DeviceAuthorizationService) validateAuthorizationRequest(req *DeviceAuthorizationRequest) error {
	if req.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	return nil
}

// validateTokenRequest validates the device token request.
func (s *DeviceAuthorizationService) validateTokenRequest(req *DeviceTokenRequest) error {
	if req.GrantType != "urn:ietf:params:oauth:grant-type:device_code" {
		return fmt.Errorf("invalid grant_type, must be 'urn:ietf:params:oauth:grant-type:device_code'")
	}
	if req.DeviceCode == "" {
		return fmt.Errorf("device_code is required")
	}
	if req.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	return nil
}

// generateDeviceCode generates a cryptographically secure device code.
func (s *DeviceAuthorizationService) generateDeviceCode() (string, error) {
	// Generate 32 bytes of randomness (256 bits)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// Encode as base32 (URL-safe, case-insensitive)
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)), nil
}

// generateUserCode generates a human-readable user code.
func (s *DeviceAuthorizationService) generateUserCode() (string, error) {
	charset := s.config.UserCodeCharset
	if charset == "" {
		charset = "BCDFGHJKLMNPQRSTVWXZ" // Default: remove vowels to avoid words
	}

	length := s.config.UserCodeLength
	if length == 0 {
		length = 8
	}

	// Generate random user code
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// Map to charset
	code := make([]byte, length)
	for i := 0; i < length; i++ {
		code[i] = charset[int(b[i])%len(charset)]
	}

	// Format with dash for readability (e.g., "BCDF-GHJK")
	if length == 8 {
		return fmt.Sprintf("%s-%s", string(code[:4]), string(code[4:])), nil
	}

	return string(code), nil
}

// cleanupLoop periodically removes expired device codes.
func (s *DeviceAuthorizationService) cleanupLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.cleanupTimer.C:
			s.cleanupExpiredCodes()
		case <-s.stopCleanup:
			s.cleanupTimer.Stop()
			return
		}
	}
}

// cleanupExpiredCodes removes expired device codes from storage.
func (s *DeviceAuthorizationService) cleanupExpiredCodes() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for deviceCode, entry := range s.deviceCodes {
		if now.After(entry.ExpiresAt) {
			delete(s.deviceCodes, deviceCode)
			delete(s.userCodes, entry.UserCode)
		}
	}
}

// Stop stops the cleanup goroutine.
func (s *DeviceAuthorizationService) Stop() {
	close(s.stopCleanup)
	s.wg.Wait()
}

// DeviceTokenResponse represents the token response for device flow.
type DeviceTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// DeviceAuthorizationError represents an error in device authorization flow.
type DeviceAuthorizationError struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// Error implements the error interface.
func (e *DeviceAuthorizationError) Error() string {
	if e.ErrorDescription != "" {
		return fmt.Sprintf("%s: %s", e.ErrorCode, e.ErrorDescription)
	}
	return e.ErrorCode
}

// GetDeviceCodeStats returns statistics about device codes.
func (s *DeviceAuthorizationService) GetDeviceCodeStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_codes": len(s.deviceCodes),
		"by_status":   make(map[DeviceCodeStatus]int),
	}

	statusCounts := make(map[DeviceCodeStatus]int)
	for _, entry := range s.deviceCodes {
		statusCounts[entry.Status]++
	}
	stats["by_status"] = statusCounts

	return stats
}
