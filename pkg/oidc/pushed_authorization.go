// Package oidc - Pushed Authorization Requests (PAR)
// Implements RFC 9126 OAuth 2.0 Pushed Authorization Requests
// Enhances security by allowing clients to push authorization request parameters directly to the AS
package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// PushedAuthorizationRequest represents a PAR request per RFC 9126 Section 2.1.
type PushedAuthorizationRequest struct {
	// Standard OAuth 2.0 authorization parameters
	ClientID            string `json:"client_id"`
	ResponseType        string `json:"response_type"`
	RedirectURI         string `json:"redirect_uri,omitempty"`
	Scope               string `json:"scope,omitempty"`
	State               string `json:"state,omitempty"`
	CodeChallenge       string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`

	// OIDC specific parameters
	Nonce        string   `json:"nonce,omitempty"`
	ResponseMode string   `json:"response_mode,omitempty"`
	Display      string   `json:"display,omitempty"`
	Prompt       string   `json:"prompt,omitempty"`
	MaxAge       int      `json:"max_age,omitempty"`
	UILocales    []string `json:"ui_locales,omitempty"`
	IDTokenHint  string   `json:"id_token_hint,omitempty"`
	LoginHint    string   `json:"login_hint,omitempty"`
	ACRValues    []string `json:"acr_values,omitempty"`

	// Additional custom parameters
	CustomParameters map[string]string `json:"-"`
}

// PushedAuthorizationResponse represents a PAR response per RFC 9126 Section 2.2.
type PushedAuthorizationResponse struct {
	// RequestURI is the identifier for the pushed request
	RequestURI string `json:"request_uri"`

	// ExpiresIn is seconds until the request URI expires
	ExpiresIn int `json:"expires_in"`
}

// PushedAuthorizationError represents a PAR error response.
type PushedAuthorizationError struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// Error implements the error interface.
func (e *PushedAuthorizationError) Error() string {
	if e.ErrorDescription != "" {
		return fmt.Sprintf("%s: %s", e.ErrorCode, e.ErrorDescription)
	}
	return e.ErrorCode
}

// RequestURIEntry stores a pushed authorization request.
type RequestURIEntry struct {
	RequestURI string
	ClientID   string
	Request    *PushedAuthorizationRequest
	CreatedAt  time.Time
	ExpiresAt  time.Time
	Used       bool
	UsedAt     time.Time
}

// PARConfig configures PAR behavior.
type PARConfig struct {
	// RequestURIPrefix is the prefix for request URIs (e.g., "urn:ietf:params:oauth:request_uri:")
	RequestURIPrefix string

	// RequestURILifetime is how long request URIs are valid
	RequestURILifetime time.Duration

	// RequirePKCE enforces PKCE for PAR requests
	RequirePKCE bool

	// RequireRedirectURI enforces redirect_uri in requests
	RequireRedirectURI bool

	// AllowedResponseTypes restricts response_type values
	AllowedResponseTypes []string

	// MaxRequestSize limits request size in bytes (0 = no limit)
	MaxRequestSize int

	// SingleUse enforces one-time use of request URIs
	SingleUse bool
}

// DefaultPARConfig returns secure default PAR configuration.
func DefaultPARConfig() *PARConfig {
	return &PARConfig{
		RequestURIPrefix:     "urn:ietf:params:oauth:request_uri:",
		RequestURILifetime:   5 * time.Minute, // Short-lived per RFC 9126
		RequirePKCE:          true,
		RequireRedirectURI:   true,
		AllowedResponseTypes: []string{"code", "code id_token", "id_token token"},
		MaxRequestSize:       10240, // 10 KB
		SingleUse:            true,
	}
}

// PARService manages pushed authorization requests.
type PARService struct {
	config       *PARConfig
	requests     map[string]*RequestURIEntry // requestURI -> entry
	mu           sync.RWMutex
	cleanupTimer *time.Ticker
	stopCleanup  chan struct{}
}

// NewPARService creates a new PAR service.
func NewPARService(config *PARConfig) *PARService {
	if config == nil {
		config = DefaultPARConfig()
	}

	service := &PARService{
		config:      config,
		requests:    make(map[string]*RequestURIEntry),
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine (runs every minute)
	service.cleanupTimer = time.NewTicker(time.Minute)
	go service.cleanupLoop()

	return service
}

// PushAuthorizationRequest processes a PAR request per RFC 9126 Section 2.
func (s *PARService) PushAuthorizationRequest(
	ctx context.Context,
	req *PushedAuthorizationRequest,
) (*PushedAuthorizationResponse, error) {
	// Validate request
	if err := s.validateRequest(req); err != nil {
		return nil, err
	}

	// Generate request URI
	requestURI, err := s.generateRequestURI()
	if err != nil {
		return nil, &PushedAuthorizationError{
			ErrorCode:        ErrorServerError,
			ErrorDescription: "failed to generate request URI",
		}
	}

	// Store request
	now := time.Now()
	entry := &RequestURIEntry{
		RequestURI: requestURI,
		ClientID:   req.ClientID,
		Request:    req,
		CreatedAt:  now,
		ExpiresAt:  now.Add(s.config.RequestURILifetime),
		Used:       false,
	}

	s.mu.Lock()
	s.requests[requestURI] = entry
	s.mu.Unlock()

	// Build response per RFC 9126 Section 2.2
	response := &PushedAuthorizationResponse{
		RequestURI: requestURI,
		ExpiresIn:  int(s.config.RequestURILifetime.Seconds()),
	}

	return response, nil
}

// GetAuthorizationRequest retrieves a stored authorization request by request_uri.
func (s *PARService) GetAuthorizationRequest(
	ctx context.Context, requestURI string, clientID string,
) (*PushedAuthorizationRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.requests[requestURI]
	if !exists {
		return nil, &PushedAuthorizationError{
			ErrorCode:        ErrorInvalidRequest,
			ErrorDescription: "request_uri not found or expired",
		}
	}

	// Verify client ID matches
	if entry.ClientID != clientID {
		return nil, &PushedAuthorizationError{
			ErrorCode:        ErrorInvalidClient,
			ErrorDescription: "request_uri does not belong to this client",
		}
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		delete(s.requests, requestURI)
		return nil, &PushedAuthorizationError{
			ErrorCode:        ErrorInvalidRequest,
			ErrorDescription: "request_uri has expired",
		}
	}

	// Check single-use enforcement
	if s.config.SingleUse && entry.Used {
		return nil, &PushedAuthorizationError{
			ErrorCode:        ErrorInvalidRequest,
			ErrorDescription: "request_uri has already been used",
		}
	}

	// Mark as used if single-use is enforced
	if s.config.SingleUse {
		entry.Used = true
		entry.UsedAt = time.Now()
	}

	// Return a copy to avoid external modifications
	requestCopy := *entry.Request
	return &requestCopy, nil
}

// RevokeRequestURI revokes a request URI (makes it unusable).
func (s *PARService) RevokeRequestURI(ctx context.Context, requestURI string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.requests[requestURI]
	if !exists {
		return fmt.Errorf("request_uri not found")
	}

	// Mark as used to prevent further use
	entry.Used = true
	entry.UsedAt = time.Now()

	return nil
}

// validateRequest validates a pushed authorization request.
func (s *PARService) validateRequest(req *PushedAuthorizationRequest) error {
	// Validate client_id
	if req.ClientID == "" {
		return &PushedAuthorizationError{
			ErrorCode:        ErrorInvalidRequest,
			ErrorDescription: "client_id is required",
		}
	}

	// Validate response_type
	if req.ResponseType == "" {
		return &PushedAuthorizationError{
			ErrorCode:        ErrorInvalidRequest,
			ErrorDescription: "response_type is required",
		}
	}

	// Check allowed response types
	if len(s.config.AllowedResponseTypes) > 0 {
		allowed := false
		for _, allowedType := range s.config.AllowedResponseTypes {
			if req.ResponseType == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return &PushedAuthorizationError{
				ErrorCode:        ErrorUnsupportedResponseType,
				ErrorDescription: fmt.Sprintf("response_type '%s' not allowed", req.ResponseType),
			}
		}
	}

	// Validate redirect_uri if required
	if s.config.RequireRedirectURI && req.RedirectURI == "" {
		return &PushedAuthorizationError{
			ErrorCode:        ErrorInvalidRequest,
			ErrorDescription: "redirect_uri is required",
		}
	}

	// Validate redirect_uri format
	if req.RedirectURI != "" {
		if _, err := url.Parse(req.RedirectURI); err != nil {
			return &PushedAuthorizationError{
				ErrorCode:        ErrorInvalidRequest,
				ErrorDescription: "redirect_uri is not a valid URI",
			}
		}
	}

	// Validate PKCE if required
	if s.config.RequirePKCE {
		if req.CodeChallenge == "" {
			return &PushedAuthorizationError{
				ErrorCode:        ErrorInvalidRequest,
				ErrorDescription: "code_challenge is required (PKCE)",
			}
		}
		if req.CodeChallengeMethod == "" {
			req.CodeChallengeMethod = "plain" // Default per RFC 7636
		}
		if req.CodeChallengeMethod != "plain" && req.CodeChallengeMethod != "S256" {
			return &PushedAuthorizationError{
				ErrorCode:        ErrorInvalidRequest,
				ErrorDescription: "code_challenge_method must be 'plain' or 'S256'",
			}
		}
	}

	// Validate scope format
	if req.Scope != "" {
		scopes := strings.Fields(req.Scope)
		if len(scopes) == 0 {
			return &PushedAuthorizationError{
				ErrorCode:        ErrorInvalidScope,
				ErrorDescription: "scope format is invalid",
			}
		}
	}

	return nil
}

// generateRequestURI generates a secure request URI per RFC 9126 Section 2.2.
func (s *PARService) generateRequestURI() (string, error) {
	// Generate 32 bytes of randomness (256 bits)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// Encode as base64 URL-safe
	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b)

	// Add prefix per RFC 9126
	return s.config.RequestURIPrefix + encoded, nil
}

// cleanupLoop periodically removes expired request URIs.
func (s *PARService) cleanupLoop() {
	for {
		select {
		case <-s.cleanupTimer.C:
			s.cleanupExpiredRequests()
		case <-s.stopCleanup:
			s.cleanupTimer.Stop()
			return
		}
	}
}

// cleanupExpiredRequests removes expired request URIs from storage.
func (s *PARService) cleanupExpiredRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for requestURI, entry := range s.requests {
		if now.After(entry.ExpiresAt) {
			delete(s.requests, requestURI)
		}
	}
}

// Stop stops the cleanup goroutine.
func (s *PARService) Stop() {
	close(s.stopCleanup)
}

// GetRequestURIStats returns statistics about request URIs.
func (s *PARService) GetRequestURIStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalRequests := len(s.requests)
	usedCount := 0
	expiredCount := 0
	now := time.Now()

	for _, entry := range s.requests {
		if entry.Used {
			usedCount++
		}
		if now.After(entry.ExpiresAt) {
			expiredCount++
		}
	}

	return map[string]interface{}{
		"total_requests":   totalRequests,
		"used_requests":    usedCount,
		"expired_requests": expiredCount,
		"active_requests":  totalRequests - usedCount - expiredCount,
	}
}

// ValidateRequestURI validates that a request_uri has the correct format.
func (s *PARService) ValidateRequestURI(requestURI string) error {
	if !strings.HasPrefix(requestURI, s.config.RequestURIPrefix) {
		return fmt.Errorf("request_uri does not have the correct prefix")
	}

	// Extract the identifier part
	identifier := strings.TrimPrefix(requestURI, s.config.RequestURIPrefix)
	if identifier == "" {
		return fmt.Errorf("request_uri identifier is empty")
	}

	// Validate base64 URL encoding
	if _, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(identifier); err != nil {
		return fmt.Errorf("request_uri identifier is not valid base64url: %w", err)
	}

	return nil
}

// BuildAuthorizationURL constructs the authorization URL with request_uri.
// This is what the client would use to redirect the user to the authorization endpoint.
func BuildAuthorizationURL(authorizationEndpoint string, requestURI string, clientID string, state string) (string, error) {
	u, err := url.Parse(authorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf("invalid authorization endpoint: %w", err)
	}

	// Add query parameters per RFC 9126 Section 4
	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("request_uri", requestURI)

	if state != "" {
		q.Set("state", state)
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ExtractPARParameters converts a PushedAuthorizationRequest to URL parameters.
// This is useful for logging, debugging, or converting to other formats.
func ExtractPARParameters(req *PushedAuthorizationRequest) url.Values {
	params := url.Values{}

	if req.ClientID != "" {
		params.Set("client_id", req.ClientID)
	}
	if req.ResponseType != "" {
		params.Set("response_type", req.ResponseType)
	}
	if req.RedirectURI != "" {
		params.Set("redirect_uri", req.RedirectURI)
	}
	if req.Scope != "" {
		params.Set("scope", req.Scope)
	}
	if req.State != "" {
		params.Set("state", req.State)
	}
	if req.CodeChallenge != "" {
		params.Set("code_challenge", req.CodeChallenge)
	}
	if req.CodeChallengeMethod != "" {
		params.Set("code_challenge_method", req.CodeChallengeMethod)
	}
	if req.Nonce != "" {
		params.Set("nonce", req.Nonce)
	}
	if req.ResponseMode != "" {
		params.Set("response_mode", req.ResponseMode)
	}
	if req.Display != "" {
		params.Set("display", req.Display)
	}
	if req.Prompt != "" {
		params.Set("prompt", req.Prompt)
	}
	if req.MaxAge > 0 {
		params.Set("max_age", fmt.Sprintf("%d", req.MaxAge))
	}
	if len(req.UILocales) > 0 {
		params.Set("ui_locales", strings.Join(req.UILocales, " "))
	}
	if req.IDTokenHint != "" {
		params.Set("id_token_hint", req.IDTokenHint)
	}
	if req.LoginHint != "" {
		params.Set("login_hint", req.LoginHint)
	}
	if len(req.ACRValues) > 0 {
		params.Set("acr_values", strings.Join(req.ACRValues, " "))
	}

	// Add custom parameters
	for key, value := range req.CustomParameters {
		params.Set(key, value)
	}

	return params
}

// ParsePARParameters converts URL parameters to a PushedAuthorizationRequest.
// This is useful for handling incoming PAR requests.
func ParsePARParameters(params url.Values) *PushedAuthorizationRequest {
	req := &PushedAuthorizationRequest{
		ClientID:            params.Get("client_id"),
		ResponseType:        params.Get("response_type"),
		RedirectURI:         params.Get("redirect_uri"),
		Scope:               params.Get("scope"),
		State:               params.Get("state"),
		CodeChallenge:       params.Get("code_challenge"),
		CodeChallengeMethod: params.Get("code_challenge_method"),
		Nonce:               params.Get("nonce"),
		ResponseMode:        params.Get("response_mode"),
		Display:             params.Get("display"),
		Prompt:              params.Get("prompt"),
		IDTokenHint:         params.Get("id_token_hint"),
		LoginHint:           params.Get("login_hint"),
		CustomParameters:    make(map[string]string),
	}

	// Parse max_age
	if maxAgeStr := params.Get("max_age"); maxAgeStr != "" {
		if maxAge, err := parseIntSafe(maxAgeStr); err == nil {
			req.MaxAge = maxAge
		}
	}

	// Parse ui_locales
	if uiLocales := params.Get("ui_locales"); uiLocales != "" {
		req.UILocales = strings.Fields(uiLocales)
	}

	// Parse acr_values
	if acrValues := params.Get("acr_values"); acrValues != "" {
		req.ACRValues = strings.Fields(acrValues)
	}

	// Store any custom parameters
	standardParams := map[string]bool{
		"client_id": true, "response_type": true, "redirect_uri": true,
		"scope": true, "state": true, "code_challenge": true,
		"code_challenge_method": true, "nonce": true, "response_mode": true,
		"display": true, "prompt": true, "max_age": true, "ui_locales": true,
		"id_token_hint": true, "login_hint": true, "acr_values": true,
	}

	for key := range params {
		if !standardParams[key] {
			req.CustomParameters[key] = params.Get(key)
		}
	}

	return req
}

// parseIntSafe safely parses an integer string.
func parseIntSafe(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
