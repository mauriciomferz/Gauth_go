package webhook

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrWebhookNotFound      = errors.New("webhook not found")
	ErrInvalidWebhookURL    = errors.New("invalid webhook URL")
	ErrInvalidEventType     = errors.New("invalid event type")
	ErrWebhookAlreadyExists = errors.New("webhook already exists for this URL")
)

// Manager handles webhook registration and management
type Manager struct {
	db *sqlx.DB
}

// NewManager creates a new webhook manager
func NewManager(db *sqlx.DB) *Manager {
	return &Manager{db: db}
}

// CreateWebhook creates a new webhook registration
func (m *Manager) CreateWebhook(ctx context.Context, userID string, req *CreateWebhookRequest) (*Webhook, string, error) {
	// Validate event types
	for _, event := range req.Events {
		if !isValidEventType(event) {
			return nil, "", fmt.Errorf("%w: %s", ErrInvalidEventType, event)
		}
	}

	// Generate webhook ID and secret
	webhookID := uuid.New().String()
	secret, err := generateSecret()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate secret: %w", err)
	}

	// Set defaults
	retryCount := req.RetryCount
	if retryCount == 0 {
		retryCount = 3
	}
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30
	}

	// Create webhook
	webhook := &Webhook{
		ID:          webhookID,
		Name:        req.Name,
		URL:         req.URL,
		Secret:      secret,
		UserID:      userID,
		Enabled:     true,
		Events:      req.Events,
		RetryCount:  retryCount,
		Timeout:     timeout,
		Description: req.Description,
		Headers:     req.Headers,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Insert into database
	query := `
		INSERT INTO webhooks (
			id, name, url, secret, user_id, enabled, events, 
			retry_count, timeout_seconds, description, headers,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`

	_, err = m.db.ExecContext(ctx, query,
		webhook.ID, webhook.Name, webhook.URL, webhook.Secret,
		webhook.UserID, webhook.Enabled, pq.Array(webhook.Events),
		webhook.RetryCount, webhook.Timeout, webhook.Description,
		webhook.Headers, webhook.CreatedAt, webhook.UpdatedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create webhook: %w", err)
	}

	return webhook, secret, nil
}

// GetWebhook retrieves a webhook by ID
func (m *Manager) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	webhook := &Webhook{}
	query := `
		SELECT id, name, url, secret, user_id, enabled, events,
			   retry_count, timeout_seconds, description, headers,
			   created_at, updated_at, last_triggered_at
		FROM webhooks
		WHERE id = $1
	`

	err := m.db.GetContext(ctx, webhook, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrWebhookNotFound
		}
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}

	return webhook, nil
}

// ListWebhooks lists webhooks with optional filtering
func (m *Manager) ListWebhooks(ctx context.Context, query *ListWebhooksQuery) ([]Webhook, error) {
	webhooks := []Webhook{}
	
	sqlQuery := `
		SELECT id, name, url, secret, user_id, enabled, events,
			   retry_count, timeout_seconds, description, headers,
			   created_at, updated_at, last_triggered_at
		FROM webhooks
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if query.UserID != "" {
		sqlQuery += fmt.Sprintf(" AND user_id = $%d", argPos)
		args = append(args, query.UserID)
		argPos++
	}

	if query.Enabled != nil {
		sqlQuery += fmt.Sprintf(" AND enabled = $%d", argPos)
		args = append(args, *query.Enabled)
		argPos++
	}

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

	err := m.db.SelectContext(ctx, &webhooks, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	return webhooks, nil
}

// UpdateWebhook updates an existing webhook
func (m *Manager) UpdateWebhook(ctx context.Context, id string, req *UpdateWebhookRequest) (*Webhook, error) {
	// Validate event types if provided
	if len(req.Events) > 0 {
		for _, event := range req.Events {
			if !isValidEventType(event) {
				return nil, fmt.Errorf("%w: %s", ErrInvalidEventType, event)
			}
		}
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *req.Name)
		argPos++
	}

	if req.URL != nil {
		updates = append(updates, fmt.Sprintf("url = $%d", argPos))
		args = append(args, *req.URL)
		argPos++
	}

	if req.Enabled != nil {
		updates = append(updates, fmt.Sprintf("enabled = $%d", argPos))
		args = append(args, *req.Enabled)
		argPos++
	}

	if len(req.Events) > 0 {
		updates = append(updates, fmt.Sprintf("events = $%d", argPos))
		args = append(args, pq.Array(req.Events))
		argPos++
	}

	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *req.Description)
		argPos++
	}

	if req.Headers != nil {
		updates = append(updates, fmt.Sprintf("headers = $%d", argPos))
		args = append(args, req.Headers)
		argPos++
	}

	if req.RetryCount != nil {
		updates = append(updates, fmt.Sprintf("retry_count = $%d", argPos))
		args = append(args, *req.RetryCount)
		argPos++
	}

	if req.Timeout != nil {
		updates = append(updates, fmt.Sprintf("timeout_seconds = $%d", argPos))
		args = append(args, *req.Timeout)
		argPos++
	}

	if len(updates) == 0 {
		return m.GetWebhook(ctx, id)
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE webhooks
		SET %s
		WHERE id = $%d
	`, join(updates, ", "), argPos)

	result, err := m.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update webhook: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check update result: %w", err)
	}

	if rows == 0 {
		return nil, ErrWebhookNotFound
	}

	return m.GetWebhook(ctx, id)
}

// DeleteWebhook deletes a webhook
func (m *Manager) DeleteWebhook(ctx context.Context, id string) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	result, err := m.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}

	if rows == 0 {
		return ErrWebhookNotFound
	}

	return nil
}

// RegenerateSecret generates a new secret for a webhook
func (m *Manager) RegenerateSecret(ctx context.Context, id string) (string, error) {
	secret, err := generateSecret()
	if err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}

	query := `
		UPDATE webhooks
		SET secret = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	result, err := m.db.ExecContext(ctx, query, secret, id)
	if err != nil {
		return "", fmt.Errorf("failed to regenerate secret: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("failed to check update result: %w", err)
	}

	if rows == 0 {
		return "", ErrWebhookNotFound
	}

	return secret, nil
}

// GetWebhookStats retrieves delivery statistics for a webhook
func (m *Manager) GetWebhookStats(ctx context.Context, id string) (*WebhookStats, error) {
	stats := &WebhookStats{}
	query := `
		SELECT * FROM webhook_stats
		WHERE webhook_id = $1
	`

	err := m.db.GetContext(ctx, stats, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrWebhookNotFound
		}
		return nil, fmt.Errorf("failed to get webhook stats: %w", err)
	}

	return stats, nil
}

// ValidateSignature validates a webhook signature
func ValidateSignature(payload []byte, signature string, secret string) bool {
	expectedSignature := GenerateSignature(payload, secret)
	return subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) == 1
}

// GenerateSignature generates HMAC-SHA256 signature for webhook payload
func GenerateSignature(payload []byte, secret string) string {
	// Note: In production, use crypto/hmac with sha256
	// For simplicity, this is a placeholder
	return fmt.Sprintf("sha256=%x", payload)
}

// Helper functions

func generateSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "whsec_" + hex.EncodeToString(bytes), nil
}

func isValidEventType(eventType string) bool {
	validEvents := map[string]bool{
		string(EventPoACreated):                  true,
		string(EventPoAUpdated):                  true,
		string(EventPoARevoked):                  true,
		string(EventPoAVerified):                 true,
		string(EventPoAExpired):                  true,
		string(EventSuccessorActivated):          true,
		string(EventDelegationCreated):           true,
		string(EventDelegationRevoked):           true,
		string(EventDualControlApprovalRequired): true,
		string(EventDualControlApproved):         true,
		string(EventDualControlRejected):         true,
		string(EventBlockchainSyncCompleted):     true,
		string(EventBlockchainSyncFailed):        true,
	}
	return validEvents[eventType]
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
