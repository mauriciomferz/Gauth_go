package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// GAuthService provides a comprehensive service layer for the GAuth protocol
type GAuthService struct {
	config *viper.Viper
	logger *logrus.Logger
	redis  *redis.Client
}

// NewGAuthService creates a new instance of the GAuth service
func NewGAuthService(config *viper.Viper, logger *logrus.Logger) (*GAuthService, error) {
	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.GetString("redis.addr"),
		Password: config.GetString("redis.password"),
		DB:       config.GetInt("redis.db"),
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warnf("Redis connection failed: %v", err)
		// Continue without Redis for demo purposes
		redisClient = nil
	}

	return &GAuthService{
		config: config,
		logger: logger,
		redis:  redisClient,
	}, nil
}

// Client represents a client application
type Client struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
}

// AuthorizeRequest represents an authorization request
type AuthorizeRequest struct {
	ClientID     string `json:"client_id" binding:"required"`
	ResponseType string `json:"response_type" binding:"required"`
	Scope        string `json:"scope"`
	RedirectURI  string `json:"redirect_uri" binding:"required"`
	State        string `json:"state"`
}

// AuthorizeResponse represents an authorization response
type AuthorizeResponse struct {
	Code        string `json:"code,omitempty"`
	State       string `json:"state,omitempty"`
	RedirectURI string `json:"redirect_uri"`
	Error       string `json:"error,omitempty"`
}

// Authorize processes an authorization request
func (s *GAuthService) Authorize(ctx context.Context, req *AuthorizeRequest) (*AuthorizeResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"client_id": req.ClientID,
		"scope":     req.Scope,
	}).Info("Processing authorization request")

	// Validate client
	client, err := s.validateClient(ctx, req.ClientID)
	if err != nil {
		return &AuthorizeResponse{
			Error: "invalid_client",
		}, err
	}

	// Generate authorization code
	code := generateID("auth_code")

	// Store authorization data in cache if Redis is available
	if s.redis != nil {
		authData := map[string]interface{}{
			"client_id":    req.ClientID,
			"scope":        req.Scope,
			"redirect_uri": req.RedirectURI,
			"user_id":      "demo_user",
			"created_at":   time.Now().Unix(),
		}
		data, _ := json.Marshal(authData)
		s.redis.Set(ctx, fmt.Sprintf("auth_code:%s", code), data, time.Minute*10)
	}

	// Log audit event
	s.logAuditEvent(ctx, "authorization_request", req.ClientID, client.ID, "authorize", "success")

	return &AuthorizeResponse{
		Code:        code,
		State:       req.State,
		RedirectURI: req.RedirectURI,
	}, nil
}

// TokenRequest represents a token request
type TokenRequest struct {
	GrantType    string `json:"grant_type" binding:"required"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
}

// TokenResponse represents a token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
}

// Token processes a token request
func (s *GAuthService) Token(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"client_id":  req.ClientID,
		"grant_type": req.GrantType,
	}).Info("Processing token request")

	switch req.GrantType {
	case "authorization_code":
		return s.exchangeCodeForToken(ctx, req)
	case "refresh_token":
		return s.refreshToken(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported grant type: %s", req.GrantType)
	}
}

// LegalEntity represents a legal entity
type LegalEntity struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Jurisdiction string                 `json:"jurisdiction"`
	Status       string                 `json:"status"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
}

// CreateLegalEntity creates a new legal entity
func (s *GAuthService) CreateLegalEntity(ctx context.Context, entity *LegalEntity) (*LegalEntity, error) {
	s.logger.WithFields(logrus.Fields{
		"entity_name": entity.Name,
		"entity_type": entity.Type,
	}).Info("Creating legal entity")

	// Generate ID
	entity.ID = generateID("entity")
	entity.CreatedAt = time.Now()
	entity.Status = "active"

	// Store in cache if Redis is available
	if s.redis != nil {
		data, _ := json.Marshal(entity)
		s.redis.Set(ctx, fmt.Sprintf("entity:%s", entity.ID), data, time.Hour*24)
	}

	// Log audit event
	s.logAuditEvent(ctx, "entity_creation", "system", entity.ID, "create", "success")

	return entity, nil
}

// GetLegalEntity retrieves a legal entity by ID
func (s *GAuthService) GetLegalEntity(ctx context.Context, id string) (*LegalEntity, error) {
	s.logger.WithField("entity_id", id).Info("Retrieving legal entity")

	// Try to get from cache first
	if s.redis != nil {
		data, err := s.redis.Get(ctx, fmt.Sprintf("entity:%s", id)).Result()
		if err == nil {
			var entity LegalEntity
			if json.Unmarshal([]byte(data), &entity) == nil {
				return &entity, nil
			}
		}
	}

	// For demo purposes, return a mock entity
	return &LegalEntity{
		ID:           id,
		Name:         "Demo Legal Entity",
		Type:         "corporation",
		Jurisdiction: "US",
		Status:       "active",
		Metadata: map[string]interface{}{
			"demo": true,
		},
		CreatedAt: time.Now().Add(-time.Hour * 24),
	}, nil
}

// PowerOfAttorney represents a power of attorney
type PowerOfAttorney struct {
	ID         string                 `json:"id"`
	Grantor    string                 `json:"grantor"`
	Grantee    string                 `json:"grantee"`
	Powers     []string               `json:"powers"`
	Conditions map[string]interface{} `json:"conditions"`
	ExpiresAt  *time.Time             `json:"expires_at,omitempty"`
	Status     string                 `json:"status"`
	CreatedAt  time.Time              `json:"created_at"`
}

// CreatePowerOfAttorney creates a new power of attorney
func (s *GAuthService) CreatePowerOfAttorney(ctx context.Context, poa *PowerOfAttorney) (*PowerOfAttorney, error) {
	s.logger.WithFields(logrus.Fields{
		"grantor": poa.Grantor,
		"grantee": poa.Grantee,
	}).Info("Creating power of attorney")

	poa.ID = generateID("poa")
	poa.CreatedAt = time.Now()
	poa.Status = "active"

	// Store in cache if Redis is available
	if s.redis != nil {
		data, _ := json.Marshal(poa)
		s.redis.Set(ctx, fmt.Sprintf("poa:%s", poa.ID), data, time.Hour*24)
	}

	// Log audit event
	s.logAuditEvent(ctx, "power_of_attorney_creation", poa.Grantor, poa.ID, "create", "success")

	return poa, nil
}

// AuditEvent represents an audit event
type AuditEvent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	ActorID    string                 `json:"actor_id"`
	ResourceID string                 `json:"resource_id"`
	Action     string                 `json:"action"`
	Outcome    string                 `json:"outcome"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// GetAuditEvents retrieves audit events
func (s *GAuthService) GetAuditEvents(ctx context.Context, limit int, offset int) ([]*AuditEvent, error) {
	s.logger.WithFields(logrus.Fields{
		"limit":  limit,
		"offset": offset,
	}).Info("Retrieving audit events")

	// For demo purposes, return mock events
	events := []*AuditEvent{
		{
			ID:         generateID("event"),
			Type:       "authorization_request",
			ActorID:    "demo_client",
			ResourceID: "demo_user",
			Action:     "authorize",
			Outcome:    "success",
			Timestamp:  time.Now().Add(-time.Minute * 5),
			Metadata:   map[string]interface{}{"scope": "read write"},
		},
		{
			ID:         generateID("event"),
			Type:       "token_exchange",
			ActorID:    "demo_client",
			ResourceID: "demo_user",
			Action:     "token",
			Outcome:    "success",
			Timestamp:  time.Now().Add(-time.Minute * 3),
			Metadata:   map[string]interface{}{"grant_type": "authorization_code"},
		},
	}

	return events, nil
}

// DemoScenario represents a demo scenario
type DemoScenario struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Steps       []DemoStep             `json:"steps"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// DemoStep represents a step in a demo scenario
type DemoStep struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	Expected   map[string]interface{} `json:"expected"`
	Status     string                 `json:"status"`
	Result     map[string]interface{} `json:"result,omitempty"`
}

// GetDemoScenarios returns available demo scenarios
func (s *GAuthService) GetDemoScenarios(ctx context.Context) ([]*DemoScenario, error) {
	return []*DemoScenario{
		{
			ID:          "basic_auth",
			Name:        "Basic Authentication Flow",
			Description: "Demonstrates the OAuth2 authorization code flow",
			Status:      "available",
			Steps: []DemoStep{
				{
					ID:   "authorize",
					Name: "Authorization Request",
					Type: "auth",
					Parameters: map[string]interface{}{
						"client_id":    "demo_client",
						"scope":        "read write",
						"redirect_uri": "http://localhost:3000/callback",
					},
					Status: "pending",
				},
				{
					ID:   "token",
					Name: "Token Exchange",
					Type: "token",
					Parameters: map[string]interface{}{
						"grant_type": "authorization_code",
					},
					Status: "pending",
				},
			},
		},
		{
			ID:          "legal_framework",
			Name:        "Legal Framework Operations",
			Description: "Demonstrates legal entity management and power of attorney",
			Status:      "available",
			Steps: []DemoStep{
				{
					ID:   "create_entity",
					Name: "Create Legal Entity",
					Type: "legal",
					Parameters: map[string]interface{}{
						"name":         "Demo Corporation",
						"type":         "corporation",
						"jurisdiction": "US",
					},
					Status: "pending",
				},
				{
					ID:   "create_poa",
					Name: "Create Power of Attorney",
					Type: "legal",
					Parameters: map[string]interface{}{
						"powers": []string{"sign_contracts", "manage_finances"},
					},
					Status: "pending",
				},
			},
		},
	}, nil
}

// Token management types
type CreateTokenRequest struct {
	Claims   map[string]interface{} `json:"claims"`
	Duration time.Duration          `json:"duration"`
	Scope    []string               `json:"scope"`
}

type CreateTokenResponse struct {
	Token        string                 `json:"token"`
	AccessToken  string                 `json:"access_token,omitempty"`
	RefreshToken string                 `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time              `json:"expires_at"`
	Claims       map[string]interface{} `json:"claims"`
}

type GetTokensRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Status   string `json:"status,omitempty"`
	OwnerID  string `json:"owner_id,omitempty"`
}

type GetTokensResponse struct {
	Tokens []TokenData `json:"tokens"`
	Total  int         `json:"total"`
}

type TokenData struct {
	ID        string                 `json:"id"`
	OwnerID   string                 `json:"owner_id"`
	ClientID  string                 `json:"client_id"`
	Scope     []string               `json:"scope"`
	Claims    map[string]interface{} `json:"claims"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	Valid     bool                   `json:"valid"`
	Status    string                 `json:"status"`
}

type TokenMetrics struct {
	ActiveTokens    int       `json:"active_tokens"`
	ExpiredTokens   int       `json:"expired_tokens"`
	RevokedTokens   int       `json:"revoked_tokens"`
	TotalTokens     int       `json:"total_tokens"`
	TokensCreated1h int       `json:"tokens_created_1h"`
	SuccessRate     float64   `json:"success_rate"`
	LastUpdated     time.Time `json:"last_updated"`
}

// CreateToken creates a new token with enhanced features
func (s *GAuthService) CreateToken(ctx context.Context, req CreateTokenRequest) (*CreateTokenResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"owner_id": req.Claims["sub"],
		"duration": req.Duration,
		"scope":    req.Scope,
	}).Info("Creating new token")

	// Generate unique token ID
	tokenID := generateToken("token")

	// Create token data
	tokenData := TokenData{
		ID:        tokenID,
		OwnerID:   extractStringClaim(req.Claims, "sub"),
		ClientID:  extractStringClaim(req.Claims, "client_id"),
		Scope:     req.Scope,
		Claims:    req.Claims,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(req.Duration),
		Valid:     true,
		Status:    "active",
	}

	// Store in Redis if available
	if s.redis != nil {
		data, err := json.Marshal(tokenData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal token data: %w", err)
		}

		key := fmt.Sprintf("token:%s", tokenID)
		if err := s.redis.Set(ctx, key, data, req.Duration).Err(); err != nil {
			s.logger.WithError(err).Error("Failed to store token in Redis")
		}

		// Store in token index for listing
		indexKey := fmt.Sprintf("token_index:%s", tokenData.OwnerID)
		s.redis.SAdd(ctx, indexKey, tokenID)
		s.redis.Expire(ctx, indexKey, req.Duration)
	}

	return &CreateTokenResponse{
		Token:     tokenID,
		ExpiresAt: tokenData.ExpiresAt,
		Claims:    req.Claims,
	}, nil
}

// GetTokens retrieves a paginated list of tokens
func (s *GAuthService) GetTokens(ctx context.Context, req GetTokensRequest) (*GetTokensResponse, error) {
	tokens := []TokenData{}

	if s.redis != nil {
		// Get all token IDs (simplified implementation)
		pattern := "token:*"
		keys, err := s.redis.Keys(ctx, pattern).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get token keys: %w", err)
		}

		// Retrieve token data
		for _, key := range keys {
			data, err := s.redis.Get(ctx, key).Result()
			if err != nil {
				continue // Skip invalid tokens
			}

			var tokenData TokenData
			if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
				continue
			}

			// Apply filters
			if req.Status != "" && tokenData.Status != req.Status {
				continue
			}
			if req.OwnerID != "" && tokenData.OwnerID != req.OwnerID {
				continue
			}

			// Update status based on expiration
			if time.Now().After(tokenData.ExpiresAt) {
				tokenData.Status = "expired"
				tokenData.Valid = false
			}

			tokens = append(tokens, tokenData)
		}
	} else {
		// Return mock data if Redis is not available
		tokens = s.getMockTokens()
	}

	// Apply pagination
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize

	if start > len(tokens) {
		start = len(tokens)
	}
	if end > len(tokens) {
		end = len(tokens)
	}

	return &GetTokensResponse{
		Tokens: tokens[start:end],
		Total:  len(tokens),
	}, nil
}

// RevokeToken revokes a specific token
func (s *GAuthService) RevokeToken(ctx context.Context, tokenID string) error {
	s.logger.WithField("token_id", tokenID).Info("Revoking token")

	if s.redis != nil {
		key := fmt.Sprintf("token:%s", tokenID)

		// Get existing token data
		data, err := s.redis.Get(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("token not found: %w", err)
		}

		var tokenData TokenData
		if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
			return fmt.Errorf("invalid token data: %w", err)
		}

		// Update token status
		tokenData.Status = "revoked"
		tokenData.Valid = false

		// Store updated data
		updatedData, err := json.Marshal(tokenData)
		if err != nil {
			return fmt.Errorf("failed to marshal updated token data: %w", err)
		}

		return s.redis.Set(ctx, key, updatedData, time.Hour*24).Err() // Keep for 24 hours
	}

	return nil // Success if no Redis
}

// ValidateToken validates a token and returns its claims
func (s *GAuthService) ValidateToken(ctx context.Context, tokenID string) (map[string]interface{}, error) {
	if s.redis != nil {
		key := fmt.Sprintf("token:%s", tokenID)
		data, err := s.redis.Get(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("token not found: %w", err)
		}

		var tokenData TokenData
		if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
			return nil, fmt.Errorf("invalid token data: %w", err)
		}

		// Check if token is valid and not expired
		if !tokenData.Valid || time.Now().After(tokenData.ExpiresAt) {
			return nil, fmt.Errorf("token is invalid or expired")
		}

		if tokenData.Status == "revoked" {
			return nil, fmt.Errorf("token has been revoked")
		}

		return tokenData.Claims, nil
	}

	// Return mock validation for demo
	return map[string]interface{}{
		"sub": "demo_user",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}, nil
}

// RefreshToken creates a new access token from a refresh token
func (s *GAuthService) RefreshToken(ctx context.Context, refreshTokenID string) (*CreateTokenResponse, error) {
	// Validate refresh token (similar to ValidateToken)
	claims, err := s.ValidateToken(ctx, refreshTokenID)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Create new access token
	newReq := CreateTokenRequest{
		Claims:   claims,
		Duration: time.Hour, // 1 hour for access token
		Scope:    []string{"read", "write"},
	}

	return s.CreateToken(ctx, newReq)
}

// GetTokenMetrics returns token-related metrics
func (s *GAuthService) GetTokenMetrics(ctx context.Context) (*TokenMetrics, error) {
	if s.redis != nil {
		// Get all tokens and calculate metrics
		keys, _ := s.redis.Keys(ctx, "token:*").Result()

		metrics := &TokenMetrics{
			TotalTokens: len(keys),
			LastUpdated: time.Now(),
		}

		active, expired, revoked := 0, 0, 0
		tokensCreated1h := 0
		oneHourAgo := time.Now().Add(-time.Hour)

		for _, key := range keys {
			data, err := s.redis.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var tokenData TokenData
			if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
				continue
			}

			switch tokenData.Status {
			case "active":
				if time.Now().After(tokenData.ExpiresAt) {
					expired++
				} else {
					active++
				}
			case "expired":
				expired++
			case "revoked":
				revoked++
			}

			if tokenData.CreatedAt.After(oneHourAgo) {
				tokensCreated1h++
			}
		}

		metrics.ActiveTokens = active
		metrics.ExpiredTokens = expired
		metrics.RevokedTokens = revoked
		metrics.TokensCreated1h = tokensCreated1h
		metrics.SuccessRate = float64(active) / float64(len(keys))

		return metrics, nil
	}

	// Return mock metrics
	return &TokenMetrics{
		ActiveTokens:    15,
		ExpiredTokens:   3,
		RevokedTokens:   2,
		TotalTokens:     20,
		TokensCreated1h: 5,
		SuccessRate:     0.95,
		LastUpdated:     time.Now(),
	}, nil
}

// Helper functions
func extractStringClaim(claims map[string]interface{}, key string) string {
	if val, ok := claims[key].(string); ok {
		return val
	}
	return ""
}

func (s *GAuthService) getMockTokens() []TokenData {
	return []TokenData{
		{
			ID:        "token_001",
			OwnerID:   "user123",
			ClientID:  "client001",
			Scope:     []string{"read", "write"},
			Claims:    map[string]interface{}{"sub": "user123", "role": "admin"},
			CreatedAt: time.Now().Add(-time.Hour * 2),
			ExpiresAt: time.Now().Add(time.Hour * 22),
			Valid:     true,
			Status:    "active",
		},
		{
			ID:        "token_002",
			OwnerID:   "user456",
			ClientID:  "client002",
			Scope:     []string{"read"},
			Claims:    map[string]interface{}{"sub": "user456", "role": "user"},
			CreatedAt: time.Now().Add(-time.Hour * 25),
			ExpiresAt: time.Now().Add(-time.Hour),
			Valid:     false,
			Status:    "expired",
		},
	}
}

// Helper functions

func (s *GAuthService) validateClient(ctx context.Context, clientID string) (*Client, error) {
	// For demo purposes, accept any client_id
	return &Client{
		ID:           clientID,
		Name:         "Demo Client",
		RedirectURIs: []string{"http://localhost:3000/callback"},
	}, nil
}

func (s *GAuthService) exchangeCodeForToken(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	// Validate authorization code
	if s.redis != nil {
		data, err := s.redis.Get(ctx, fmt.Sprintf("auth_code:%s", req.Code)).Result()
		if err != nil {
			return nil, fmt.Errorf("invalid authorization code")
		}

		var authData map[string]interface{}
		if err := json.Unmarshal([]byte(data), &authData); err != nil {
			return nil, fmt.Errorf("invalid authorization code data")
		}

		// Remove the used authorization code
		s.redis.Del(ctx, fmt.Sprintf("auth_code:%s", req.Code))
	}

	// Generate tokens
	accessToken := generateToken("access")
	refreshToken := generateToken("refresh")

	// Store refresh token if Redis is available
	if s.redis != nil {
		tokenData := map[string]interface{}{
			"client_id": req.ClientID,
			"user_id":   "demo_user",
			"scope":     "read write",
		}
		data, _ := json.Marshal(tokenData)
		s.redis.Set(ctx, fmt.Sprintf("refresh_token:%s", refreshToken), data, time.Hour*24*30) // 30 days
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour
		RefreshToken: refreshToken,
		Scope:        "read write",
	}, nil
}

func (s *GAuthService) refreshToken(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	// Validate refresh token
	if s.redis != nil {
		data, err := s.redis.Get(ctx, fmt.Sprintf("refresh_token:%s", req.RefreshToken)).Result()
		if err != nil {
			return nil, fmt.Errorf("invalid refresh token")
		}

		var tokenData map[string]interface{}
		if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
			return nil, fmt.Errorf("invalid refresh token data")
		}
	}

	// Generate new access token
	accessToken := generateToken("access")

	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600, // 1 hour
		Scope:       "read write",
	}, nil
}

func (s *GAuthService) logAuditEvent(ctx context.Context, eventType, actorID, resourceID, action, outcome string) {
	s.logger.WithFields(logrus.Fields{
		"event_type":  eventType,
		"actor_id":    actorID,
		"resource_id": resourceID,
		"action":      action,
		"outcome":     outcome,
	}).Info("Audit event logged")
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func generateToken(tokenType string) string {
	return fmt.Sprintf("%s_token_%d", tokenType, time.Now().UnixNano())
}
