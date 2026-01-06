package apikey

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mauriciomferz/AgentAuth/pkg/database"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAPIKeyNotFound   = errors.New("api key not found")
	ErrAPIKeyInvalid    = errors.New("invalid api key")
	ErrAPIKeyExpired    = errors.New("api key expired")
	ErrAPIKeyDisabled   = errors.New("api key disabled")
	ErrQuotaExceeded    = errors.New("quota exceeded")
	ErrInvalidIPAddress = errors.New("request from unauthorized IP address")
	ErrInvalidEndpoint  = errors.New("endpoint not allowed for this API key")
)

// Manager handles API key operations
type Manager struct {
	db *database.DB
}

// NewManager creates a new API key manager
func NewManager(db *database.DB) *Manager {
	return &Manager{db: db}
}

// CreateAPIKey creates a new API key
func (m *Manager) CreateAPIKey(ctx context.Context, userID string, req *CreateAPIKeyRequest) (*APIKeyWithSecret, error) {
	// Generate key ID and secret
	keyID, err := generateKeyID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key ID: %w", err)
	}

	secretKey, err := generateSecretKey(keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret key: %w", err)
	}

	// Hash the secret for storage
	keyHash, err := hashSecret(secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to hash secret: %w", err)
	}

	// Set defaults
	rateLimitPerMinute := 60
	if req.RateLimitPerMinute != nil {
		rateLimitPerMinute = *req.RateLimitPerMinute
	}

	rateLimitPerHour := 1000
	if req.RateLimitPerHour != nil {
		rateLimitPerHour = *req.RateLimitPerHour
	}

	rateLimitPerDay := 10000
	if req.RateLimitPerDay != nil {
		rateLimitPerDay = *req.RateLimitPerDay
	}

	// Create API key
	apiKey := &APIKey{
		KeyID:              keyID,
		KeyHash:            keyHash,
		Name:               req.Name,
		Description:        req.Description,
		UserID:             userID,
		RateLimitPerMinute: rateLimitPerMinute,
		RateLimitPerHour:   rateLimitPerHour,
		RateLimitPerDay:    rateLimitPerDay,
		QuotaRequestsTotal: req.QuotaRequestsTotal,
		QuotaRequestsUsed:  0,
		Enabled:            true,
		IPWhitelist:        req.IPWhitelist,
		AllowedEndpoints:   req.AllowedEndpoints,
		Metadata:           req.Metadata,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		ExpiresAt:          req.ExpiresAt,
	}

	// Insert into database
	query := `
		INSERT INTO api_keys (
			key_id, key_hash, name, description, user_id,
			rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
			quota_requests_total, quota_requests_used, enabled,
			ip_whitelist, allowed_endpoints, metadata,
			created_at, updated_at, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		) RETURNING id
	`

	err = m.db.Pool.QueryRow(ctx, query,
		apiKey.KeyID, apiKey.KeyHash, apiKey.Name, apiKey.Description, apiKey.UserID,
		apiKey.RateLimitPerMinute, apiKey.RateLimitPerHour, apiKey.RateLimitPerDay,
		apiKey.QuotaRequestsTotal, apiKey.QuotaRequestsUsed, apiKey.Enabled,
		apiKey.IPWhitelist, apiKey.AllowedEndpoints, apiKey.Metadata,
		apiKey.CreatedAt, apiKey.UpdatedAt, apiKey.ExpiresAt,
	).Scan(&apiKey.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return &APIKeyWithSecret{
		APIKey:    apiKey,
		SecretKey: secretKey,
	}, nil
}

// GetAPIKey retrieves an API key by key_id
func (m *Manager) GetAPIKey(ctx context.Context, keyID string) (*APIKey, error) {
	query := `
		SELECT id, key_id, key_hash, name, description, user_id,
			   rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
			   quota_requests_total, quota_requests_used, enabled,
			   ip_whitelist, allowed_endpoints, metadata,
			   created_at, updated_at, last_used_at, expires_at
		FROM api_keys
		WHERE key_id = $1
	`

	rows, err := m.db.Pool.Query(ctx, query, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query API key: %w", err)
	}
	defer rows.Close()

	apiKey, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[APIKey])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to collect API key: %w", err)
	}

	return &apiKey, nil
}

// ListAPIKeys lists API keys with optional filtering
func (m *Manager) ListAPIKeys(ctx context.Context, query *ListAPIKeysQuery) ([]APIKey, int, error) {
	apiKeys := []APIKey{}

	sqlQuery := `
		SELECT id, key_id, key_hash, name, description, user_id,
			   rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
			   quota_requests_total, quota_requests_used, enabled,
			   ip_whitelist, allowed_endpoints, metadata,
			   created_at, updated_at, last_used_at, expires_at
		FROM api_keys
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM api_keys WHERE 1=1`

	args := []interface{}{}
	argPos := 1

	// Apply filters
	if query.UserID != "" {
		filter := fmt.Sprintf(" AND user_id = $%d", argPos)
		sqlQuery += filter
		countQuery += filter
		args = append(args, query.UserID)
		argPos++
	}

	if query.Enabled != nil {
		filter := fmt.Sprintf(" AND enabled = $%d", argPos)
		sqlQuery += filter
		countQuery += filter
		args = append(args, *query.Enabled)
		argPos++
	}

	// Get total count
	var total int
	err := m.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count API keys: %w", err)
	}

	// Apply pagination
	sqlQuery += " ORDER BY created_at DESC"

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, query.Limit)
		argPos++
	}

	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, query.Offset)
	}

	rows, err := m.db.Pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	apiKeys, err = pgx.CollectRows(rows, pgx.RowToStructByName[APIKey])
	if err != nil {
		return nil, 0, fmt.Errorf("failed to collect API keys: %w", err)
	}

	return apiKeys, total, nil
}

// UpdateAPIKey updates an existing API key
func (m *Manager) UpdateAPIKey(ctx context.Context, keyID string, req *UpdateAPIKeyRequest) (*APIKey, error) {
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *req.Name)
		argPos++
	}

	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *req.Description)
		argPos++
	}

	if req.Enabled != nil {
		updates = append(updates, fmt.Sprintf("enabled = $%d", argPos))
		args = append(args, *req.Enabled)
		argPos++
	}

	if req.RateLimitPerMinute != nil {
		updates = append(updates, fmt.Sprintf("rate_limit_per_minute = $%d", argPos))
		args = append(args, *req.RateLimitPerMinute)
		argPos++
	}

	if req.RateLimitPerHour != nil {
		updates = append(updates, fmt.Sprintf("rate_limit_per_hour = $%d", argPos))
		args = append(args, *req.RateLimitPerHour)
		argPos++
	}

	if req.RateLimitPerDay != nil {
		updates = append(updates, fmt.Sprintf("rate_limit_per_day = $%d", argPos))
		args = append(args, *req.RateLimitPerDay)
		argPos++
	}

	if req.QuotaRequestsTotal != nil {
		updates = append(updates, fmt.Sprintf("quota_requests_total = $%d", argPos))
		args = append(args, *req.QuotaRequestsTotal)
		argPos++
	}

	if req.IPWhitelist != nil {
		updates = append(updates, fmt.Sprintf("ip_whitelist = $%d", argPos))
		args = append(args, req.IPWhitelist) // pgx handles slices
		argPos++
	}

	if req.AllowedEndpoints != nil {
		updates = append(updates, fmt.Sprintf("allowed_endpoints = $%d", argPos))
		args = append(args, req.AllowedEndpoints) // pgx handles slices
		argPos++
	}

	if req.Metadata != nil {
		updates = append(updates, fmt.Sprintf("metadata = $%d", argPos))
		args = append(args, req.Metadata)
		argPos++
	}

	if len(updates) == 0 {
		return m.GetAPIKey(ctx, keyID)
	}

	args = append(args, keyID)
	updateQuery := fmt.Sprintf(`
		UPDATE api_keys
		SET %s
		WHERE key_id = $%d
	`, join(updates, ", "), argPos)

	result, err := m.db.Pool.Exec(ctx, updateQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil, ErrAPIKeyNotFound
	}

	return m.GetAPIKey(ctx, keyID)
}

// DeleteAPIKey deletes an API key
func (m *Manager) DeleteAPIKey(ctx context.Context, keyID string) error {
	query := `DELETE FROM api_keys WHERE key_id = $1`
	result, err := m.db.Pool.Exec(ctx, query, keyID)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

// RegenerateSecret generates a new secret for an API key
func (m *Manager) RegenerateSecret(ctx context.Context, keyID string) (*RegenerateSecretResponse, error) {
	// Generate new secret
	newSecret, err := generateSecretKey(keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new secret: %w", err)
	}

	// Hash the new secret
	keyHash, err := hashSecret(newSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to hash new secret: %w", err)
	}

	// Update in database
	query := `
		UPDATE api_keys
		SET key_hash = $1
		WHERE key_id = $2
	`
	result, err := m.db.Pool.Exec(ctx, query, keyHash, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to update secret: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil, ErrAPIKeyNotFound
	}

	return &RegenerateSecretResponse{
		KeyID:         keyID,
		SecretKey:     newSecret,
		RegeneratedAt: time.Now(),
	}, nil
}

// ValidateAPIKey validates an API key and returns the associated API key record
func (m *Manager) ValidateAPIKey(ctx context.Context, secretKey string) (*APIKey, error) {
	// Extract key ID from secret (format: keyID.randomSecret)
	keyID, err := extractKeyIDFromSecret(secretKey)
	if err != nil {
		return nil, ErrAPIKeyInvalid
	}

	// Get API key from database
	apiKey, err := m.GetAPIKey(ctx, keyID)
	if err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			return nil, ErrAPIKeyInvalid
		}
		return nil, err
	}

	// Check if expired
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, ErrAPIKeyExpired
	}

	// Check if disabled
	if !apiKey.Enabled {
		return nil, ErrAPIKeyDisabled
	}

	// Verify secret hash
	err = bcrypt.CompareHashAndPassword([]byte(apiKey.KeyHash), []byte(secretKey))
	if err != nil {
		return nil, ErrAPIKeyInvalid
	}

	// Check quota
	if apiKey.QuotaRequestsTotal != nil && apiKey.QuotaRequestsUsed >= *apiKey.QuotaRequestsTotal {
		return nil, ErrQuotaExceeded
	}

	return apiKey, nil
}

// RecordUsage records API key usage for analytics
func (m *Manager) RecordUsage(ctx context.Context, usage *APIKeyUsage) error {
	query := `
		INSERT INTO api_key_usage (
			key_id, endpoint, method, status_code, response_time_ms,
			request_ip, user_agent, error_message, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := m.db.Pool.Exec(ctx, query,
		usage.KeyID, usage.Endpoint, usage.Method, usage.StatusCode, usage.ResponseTimeMs,
		usage.RequestIP, usage.UserAgent, usage.ErrorMessage, usage.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to record usage: %w", err)
	}

	// Increment usage counter
	updateQuery := `
		UPDATE api_keys
		SET quota_requests_used = quota_requests_used + 1,
		    last_used_at = $1
		WHERE key_id = $2
	`

	_, err = m.db.Pool.Exec(ctx, updateQuery, time.Now(), usage.KeyID)
	if err != nil {
		return fmt.Errorf("failed to update usage counter: %w", err)
	}

	return nil
}

// GetAPIKeyStats retrieves statistics for an API key
func (m *Manager) GetAPIKeyStats(ctx context.Context, keyID string) (*APIKeyStats, error) {
	query := `SELECT * FROM api_key_stats WHERE key_id = $1`

	rows, err := m.db.Pool.Query(ctx, query, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query API key stats: %w", err)
	}
	defer rows.Close()

	stats, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[APIKeyStats])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to collect stats: %w", err)
	}

	return &stats, nil
}

// Helper functions

func generateKeyID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "pk_live_" + hex.EncodeToString(bytes), nil
}

func generateSecretKey(keyID string) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Embed KeyID in secret: <KeyID>.<RandomHex>
	return fmt.Sprintf("%s.sk_live_%s", keyID, hex.EncodeToString(bytes)), nil
}

func hashSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func extractKeyIDFromSecret(secretKey string) (string, error) {
	// Expected format: <KeyID>.sk_live_<UniquePart>
	// We split by first dot to get KeyID
	parts := strings.SplitN(secretKey, ".", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid secret key format")
	}
	return parts[0], nil
}

func join(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
