package subscribers

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles subscriber database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new subscriber repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Subscriber represents a tenant/subscriber in the system
type Subscriber struct {
	ID                     uuid.UUID
	TenantName             string
	TenantID               string
	Status                 string
	Tier                   string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	CreatedBy              *string
	OIDCProvider           *string
	OIDCIssuer             *string
	OIDCClientID           *string
	OIDCClientSecret       *string
	OIDCScopes             []string
	OIDCDiscoveryURL       *string
	KeyType                *string
	PublicKey              *string
	PrivateKeyID           *string
	KeyGeneratedAt         *time.Time
	KeyExpiresAt           *time.Time
	PolicyTemplate         *string
	LegalFramework         *string
	NotificationChannels   []string
	NotificationEmail      *string
	NotificationWebhookURL *string
	ContactEmail           *string
	ContactName            *string
	Domain                 *string
	MaxUsers               int
	MaxTokens              int
	Metadata               map[string]interface{}

	// Computed fields from aggregations
	TotalTokens    int64
	ActiveUsers    int64
	LastActivityAt *time.Time
}

// SubscriberFilters defines query filters for listing subscribers
type SubscriberFilters struct {
	Status string // active, suspended, pending, disabled
	Tier   string // free, standard, premium, enterprise
	Search string // Search in tenant_name, tenant_id, contact_email
	Limit  int
	Offset int
}

// CreateSubscriber inserts a new subscriber
func (r *Repository) CreateSubscriber(ctx context.Context, subscriber *Subscriber) error {
	if r.db == nil {
		subscriber.ID = uuid.New()
		subscriber.CreatedAt = time.Now()
		subscriber.UpdatedAt = time.Now()
		return nil
	}
	query := `
		INSERT INTO subscribers (
			tenant_name, tenant_id, status, tier, created_by,
			oidc_provider, oidc_issuer, oidc_client_id, oidc_client_secret, 
			oidc_scopes, oidc_discovery_url,
			key_type, public_key, private_key_id, key_generated_at, key_expires_at,
			policy_template, legal_framework,
			notification_channels, notification_email, notification_webhook_url,
			contact_email, contact_name, domain, max_users, max_tokens, metadata
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18,
			$19, $20, $21,
			$22, $23, $24, $25, $26, $27
		) RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		subscriber.TenantName, subscriber.TenantID, subscriber.Status, subscriber.Tier, subscriber.CreatedBy,
		subscriber.OIDCProvider, subscriber.OIDCIssuer, subscriber.OIDCClientID, subscriber.OIDCClientSecret,
		subscriber.OIDCScopes, subscriber.OIDCDiscoveryURL,
		subscriber.KeyType, subscriber.PublicKey, subscriber.PrivateKeyID, subscriber.KeyGeneratedAt, subscriber.KeyExpiresAt,
		subscriber.PolicyTemplate, subscriber.LegalFramework,
		subscriber.NotificationChannels, subscriber.NotificationEmail, subscriber.NotificationWebhookURL,
		subscriber.ContactEmail, subscriber.ContactName, subscriber.Domain, subscriber.MaxUsers, subscriber.MaxTokens, subscriber.Metadata,
	).Scan(&subscriber.ID, &subscriber.CreatedAt, &subscriber.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}

	return nil
}

// ListSubscribers retrieves subscribers with optional filtering and pagination
func (r *Repository) ListSubscribers(ctx context.Context, filters SubscriberFilters) ([]Subscriber, int, error) {
	if r.db == nil {
		return []Subscriber{}, 0, nil
	}
	whereClauses := []string{}
	args := []interface{}{}
	argPos := 1

	// Build WHERE clause dynamically
	if filters.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("s.status = $%d", argPos))
		args = append(args, filters.Status)
		argPos++
	}

	if filters.Tier != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("s.tier = $%d", argPos))
		args = append(args, filters.Tier)
		argPos++
	}

	if filters.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(
			"(s.tenant_name ILIKE $%d OR s.tenant_id ILIKE $%d OR s.contact_email ILIKE $%d)",
			argPos, argPos, argPos,
		))
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total matching records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM subscribers s %s", whereSQL)
	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count subscribers: %w", err)
	}

	// Get paginated results with aggregations
	query := fmt.Sprintf(`
		SELECT 
			s.id, s.tenant_name, s.tenant_id, s.status, s.tier,
			s.created_at, s.updated_at, s.subscriber_id, s.subscriber_name,
			COALESCE(token_stats.total_tokens, 0) AS total_tokens,
			COALESCE(user_stats.active_users, 0) AS active_users,
			COALESCE(token_stats.last_activity, NULL) AS last_activity_at
		FROM subscribers s
		LEFT JOIN (
			SELECT 
				tenant_id,
				COUNT(*) AS total_tokens,
				MAX(last_used_at) AS last_activity
			FROM tokens
			WHERE revoked_at IS NULL
			GROUP BY tenant_id
		) token_stats ON s.tenant_id = token_stats.tenant_id
		LEFT JOIN (
			SELECT 
				tenant_id,
				COUNT(DISTINCT subject) AS active_users
			FROM tokens
			WHERE revoked_at IS NULL
			GROUP BY tenant_id
		) user_stats ON s.tenant_id = user_stats.tenant_id
		%s
		ORDER BY s.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argPos, argPos+1)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list subscribers: %w", err)
	}
	defer rows.Close()

	subscribers := make([]Subscriber, 0)
	for rows.Next() {
		var s Subscriber
		var subscriberID, subscriberName sql.NullString
		err := rows.Scan(
			&s.ID, &s.TenantName, &s.TenantID, &s.Status, &s.Tier,
			&s.CreatedAt, &s.UpdatedAt, &subscriberID, &subscriberName,
			&s.TotalTokens, &s.ActiveUsers, &s.LastActivityAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan subscriber: %w", err)
		}

		// Handle nullable fields
		if subscriberID.Valid {
			s.TenantID = subscriberID.String
		}
		if subscriberName.Valid {
			name := subscriberName.String
			s.ContactName = &name
		}

		subscribers = append(subscribers, s)
	}

	return subscribers, total, nil
}

// GetSubscriber retrieves a single subscriber by ID or tenant_id
func (r *Repository) GetSubscriber(ctx context.Context, idOrTenantID string) (*Subscriber, error) {
	if r.db == nil {
		return nil, sql.ErrNoRows
	}
	query := `
		SELECT 
			s.id, s.tenant_name, s.tenant_id, s.status, s.tier,
			s.created_at, s.updated_at, s.created_by,
			s.oidc_provider, s.oidc_issuer, s.oidc_client_id, s.oidc_client_secret,
			s.oidc_scopes, s.oidc_discovery_url,
			s.key_type, s.public_key, s.private_key_id, s.key_generated_at, s.key_expires_at,
			s.policy_template, s.legal_framework,
			s.notification_channels, s.notification_email, s.notification_webhook_url,
			s.contact_email, s.contact_name, s.domain, s.max_users, s.max_tokens, s.metadata,
			COALESCE(token_stats.total_tokens, 0) AS total_tokens,
			COALESCE(token_stats.last_activity, NULL) AS last_activity_at
		FROM subscribers s
		LEFT JOIN (
			SELECT 
				tenant_id,
				COUNT(*) AS total_tokens,
				MAX(last_used_at) AS last_activity
			FROM tokens
			GROUP BY tenant_id
		) token_stats ON s.tenant_id = token_stats.tenant_id
		WHERE s.id::text = $1 OR s.tenant_id = $1
	`

	var s Subscriber
	err := r.db.QueryRow(ctx, query, idOrTenantID).Scan(
		&s.ID, &s.TenantName, &s.TenantID, &s.Status, &s.Tier,
		&s.CreatedAt, &s.UpdatedAt, &s.CreatedBy,
		&s.OIDCProvider, &s.OIDCIssuer, &s.OIDCClientID, &s.OIDCClientSecret,
		&s.OIDCScopes, &s.OIDCDiscoveryURL,
		&s.KeyType, &s.PublicKey, &s.PrivateKeyID, &s.KeyGeneratedAt, &s.KeyExpiresAt,
		&s.PolicyTemplate, &s.LegalFramework,
		&s.NotificationChannels, &s.NotificationEmail, &s.NotificationWebhookURL,
		&s.ContactEmail, &s.ContactName, &s.Domain, &s.MaxUsers, &s.MaxTokens, &s.Metadata,
		&s.TotalTokens, &s.LastActivityAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriber: %w", err)
	}

	return &s, nil
}

// UpdateSubscriber updates subscriber configuration
func (r *Repository) UpdateSubscriber(ctx context.Context, idOrTenantID string, subscriber *Subscriber) error {
	if r.db == nil {
		return sql.ErrNoRows
	}
	query := `
		UPDATE subscribers SET
			tenant_name = $1,
			status = $2,
			tier = $3,
			oidc_provider = $4,
			oidc_issuer = $5,
			oidc_client_id = $6,
			oidc_client_secret = $7,
			oidc_scopes = $8,
			oidc_discovery_url = $9,
			key_type = $10,
			public_key = $11,
			private_key_id = $12,
			key_generated_at = $13,
			key_expires_at = $14,
			policy_template = $15,
			legal_framework = $16,
			notification_channels = $17,
			notification_email = $18,
			notification_webhook_url = $19,
			contact_email = $20,
			contact_name = $21,
			domain = $22,
			max_users = $23,
			max_tokens = $24,
			metadata = $25,
			updated_at = NOW()
		WHERE id::text = $26 OR tenant_id = $26
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		subscriber.TenantName, subscriber.Status, subscriber.Tier,
		subscriber.OIDCProvider, subscriber.OIDCIssuer, subscriber.OIDCClientID, subscriber.OIDCClientSecret,
		subscriber.OIDCScopes, subscriber.OIDCDiscoveryURL,
		subscriber.KeyType, subscriber.PublicKey, subscriber.PrivateKeyID, subscriber.KeyGeneratedAt, subscriber.KeyExpiresAt,
		subscriber.PolicyTemplate, subscriber.LegalFramework,
		subscriber.NotificationChannels, subscriber.NotificationEmail, subscriber.NotificationWebhookURL,
		subscriber.ContactEmail, subscriber.ContactName, subscriber.Domain, subscriber.MaxUsers, subscriber.MaxTokens, subscriber.Metadata,
		idOrTenantID,
	).Scan(&subscriber.UpdatedAt)

	if err == sql.ErrNoRows {
		return sql.ErrNoRows
	}
	if err != nil {
		return fmt.Errorf("failed to update subscriber: %w", err)
	}

	return nil
}

// DeleteSubscriber soft-deletes a subscriber by setting status to 'disabled'
func (r *Repository) DeleteSubscriber(ctx context.Context, idOrTenantID string) error {
	if r.db == nil {
		return sql.ErrNoRows
	}
	query := `
		UPDATE subscribers 
		SET status = 'disabled', updated_at = NOW()
		WHERE (id::text = $1 OR tenant_id = $1) AND status != 'disabled'
	`

	result, err := r.db.Exec(ctx, query, idOrTenantID)
	if err != nil {
		return fmt.Errorf("failed to delete subscriber: %w", err)
	}

	if result.RowsAffected() == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UpdateKeyMetadata updates key generation metadata for a subscriber
func (r *Repository) UpdateKeyMetadata(ctx context.Context, tenantID string, keyType, publicKey, privateKeyID string, expiresAt time.Time) error {
	if r.db == nil {
		return sql.ErrNoRows
	}
	query := `
		UPDATE subscribers
		SET 
			key_type = $1,
			public_key = $2,
			private_key_id = $3,
			key_generated_at = NOW(),
			key_expires_at = $4,
			updated_at = NOW()
		WHERE tenant_id = $5
	`

	result, err := r.db.Exec(ctx, query, keyType, publicKey, privateKeyID, expiresAt, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update key metadata: %w", err)
	}

	if result.RowsAffected() == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetSubscriberMetrics retrieves aggregated metrics for a specific subscriber
func (r *Repository) GetSubscriberMetrics(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	if r.db == nil {
		return map[string]interface{}{
			"tenant_id":      tenantID,
			"total_tokens":   0,
			"active_tokens":  0,
			"revoked_tokens": 0,
			"total_requests": 0,
			"last_activity":  nil,
			"max_users":      0,
			"max_tokens":     0,
			"status":         "unknown",
			"tier":           "unknown",
		}, nil
	}
	// Get token statistics
	tokenQuery := `
		SELECT 
			COUNT(*) AS total_tokens,
			COUNT(*) FILTER (WHERE revoked_at IS NULL AND expires_at > NOW()) AS active_tokens,
			COUNT(*) FILTER (WHERE revoked_at IS NOT NULL) AS revoked_tokens,
			MAX(last_used_at) AS last_activity
		FROM tokens
		WHERE tenant_id = $1
	`

	var totalTokens, activeTokens, revokedTokens int64
	var lastActivity *time.Time
	err := r.db.QueryRow(ctx, tokenQuery, tenantID).Scan(&totalTokens, &activeTokens, &revokedTokens, &lastActivity)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get token metrics: %w", err)
	}

	// Get audit event count
	auditQuery := `
		SELECT COUNT(*) 
		FROM audit_events
		WHERE tenant_id = $1 AND timestamp >= NOW() - INTERVAL '30 days'
	`
	var totalRequests int64
	err = r.db.QueryRow(ctx, auditQuery, tenantID).Scan(&totalRequests)
	if err != nil && err != sql.ErrNoRows {
		totalRequests = 0
	}

	// Get subscriber info
	subscriber, err := r.GetSubscriber(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriber: %w", err)
	}

	metrics := map[string]interface{}{
		"tenant_id":      tenantID,
		"total_tokens":   totalTokens,
		"active_tokens":  activeTokens,
		"revoked_tokens": revokedTokens,
		"total_requests": totalRequests,
		"last_activity":  lastActivity,
		"max_users":      subscriber.MaxUsers,
		"max_tokens":     subscriber.MaxTokens,
		"status":         subscriber.Status,
		"tier":           subscriber.Tier,
	}

	return metrics, nil
}

// CheckTenantIDExists checks if a tenant_id is already in use
func (r *Repository) CheckTenantIDExists(ctx context.Context, tenantID string) (bool, error) {
	if r.db == nil {
		return false, nil
	}
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM subscribers WHERE tenant_id = $1)`
	err := r.db.QueryRow(ctx, query, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check tenant ID: %w", err)
	}
	return exists, nil
}
