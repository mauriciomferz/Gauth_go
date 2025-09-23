package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/redis/go-redis/v9"
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