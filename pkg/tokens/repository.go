package tokens

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles token database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new token repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Token represents a token from the database
type Token struct {
	ID               string     `json:"id"`
	TokenID          string     `json:"tokenId"`
	TenantID         string     `json:"tenantId"`
	TokenType        string     `json:"tokenType"`
	Subject          string     `json:"subject"`
	Audience         []string   `json:"audience"`
	Issuer           string     `json:"issuer"`
	Scope            []string   `json:"scope"`
	IssuedAt         time.Time  `json:"issuedAt"`
	ExpiresAt        time.Time  `json:"expiresAt"`
	LastUsedAt       *time.Time `json:"lastUsedAt"`
	RevokedAt        *time.Time `json:"revokedAt"`
	RevokedBy        *string    `json:"revokedBy"`
	RevocationReason *string    `json:"revocationReason"`
	IPAddress        *string    `json:"ipAddress"`
	UserAgent        *string    `json:"userAgent"`
	DeviceID         *string    `json:"deviceId"`
	UsageCount       int        `json:"usageCount"`
	SubscriberName   string     `json:"subscriberName"` // Joined from subscribers table
}

// BlacklistEntry represents a blacklisted token
type BlacklistEntry struct {
	ID        string    `json:"id"`
	TokenID   string    `json:"tokenId"`
	TenantID  string    `json:"tenantId"`
	Reason    *string   `json:"reason"`
	RevokedAt time.Time `json:"revokedAt"`
	RevokedBy *string   `json:"revokedBy"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// TokenFilters for querying tokens
type TokenFilters struct {
	TenantID     string
	SubscriberID string
	TokenType    string
	Status       string // active, expired, revoked
	Subject      string
	Limit        int
	Offset       int
}

// CreateToken inserts a new token into the database
func (r *Repository) CreateToken(ctx context.Context, token *Token) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}
	query := `
		INSERT INTO tokens (
			token_id, tenant_id, token_type, subject, audience, issuer, scope,
			issued_at, expires_at, ip_address, user_agent, device_id, usage_count
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING id
	`

	err := r.db.QueryRow(
		ctx, query,
		token.TokenID, token.TenantID, token.TokenType, token.Subject,
		token.Audience, token.Issuer, token.Scope,
		token.IssuedAt, token.ExpiresAt,
		token.IPAddress, token.UserAgent, token.DeviceID, token.UsageCount,
	).Scan(&token.ID)

	if err != nil {
		return fmt.Errorf("failed to create token: %w", err)
	}

	return nil
}

// ListTokens retrieves tokens with filtering and pagination
func (r *Repository) ListTokens(ctx context.Context, filters TokenFilters) ([]Token, int, error) {
	if r.db == nil {
		return []Token{}, 0, nil
	}
	// Build dynamic WHERE clause
	whereClauses := []string{"t.tenant_id = $1"}
	args := []interface{}{filters.TenantID}
	paramIndex := 2

	if filters.SubscriberID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("s.subscriber_id = $%d", paramIndex))
		args = append(args, filters.SubscriberID)
		paramIndex++
	}

	if filters.TokenType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.token_type = $%d", paramIndex))
		args = append(args, filters.TokenType)
		paramIndex++
	}

	if filters.Subject != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.subject = $%d", paramIndex))
		args = append(args, filters.Subject)
		paramIndex++
	}

	// Status filtering
	if filters.Status == "active" {
		whereClauses = append(whereClauses, "t.revoked_at IS NULL AND t.expires_at > NOW()")
	} else if filters.Status == "expired" {
		whereClauses = append(whereClauses, "t.revoked_at IS NULL AND t.expires_at <= NOW()")
	} else if filters.Status == "revoked" {
		whereClauses = append(whereClauses, "t.revoked_at IS NOT NULL")
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + whereClauses[0]
		for i := 1; i < len(whereClauses); i++ {
			whereClause += " AND " + whereClauses[i]
		}
	}

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM tokens t
		LEFT JOIN subscribers s ON t.tenant_id = s.tenant_id
		%s
	`, whereClause)

	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tokens: %w", err)
	}

	// Query tokens
	query := fmt.Sprintf(`
		SELECT 
			t.id, t.token_id, t.tenant_id, t.token_type, t.subject,
			t.audience, t.issuer, t.scope,
			t.issued_at, t.expires_at, t.last_used_at,
			t.revoked_at, t.revoked_by, t.revocation_reason,
			t.ip_address, t.user_agent, t.device_id, t.usage_count,
			COALESCE(s.subscriber_name, 'Unknown') as subscriber_name
		FROM tokens t
		LEFT JOIN subscribers s ON t.tenant_id = s.tenant_id
		%s
		ORDER BY t.issued_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, paramIndex, paramIndex+1)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query tokens: %w", err)
	}
	defer rows.Close()

	var tokens []Token
	for rows.Next() {
		var t Token
		err := rows.Scan(
			&t.ID, &t.TokenID, &t.TenantID, &t.TokenType, &t.Subject,
			&t.Audience, &t.Issuer, &t.Scope,
			&t.IssuedAt, &t.ExpiresAt, &t.LastUsedAt,
			&t.RevokedAt, &t.RevokedBy, &t.RevocationReason,
			&t.IPAddress, &t.UserAgent, &t.DeviceID, &t.UsageCount,
			&t.SubscriberName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan token: %w", err)
		}
		tokens = append(tokens, t)
	}

	return tokens, total, nil
}

// GetToken retrieves a single token by token_id
func (r *Repository) GetToken(ctx context.Context, tenantID, tokenID string) (*Token, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	query := `
		SELECT 
			t.id, t.token_id, t.tenant_id, t.token_type, t.subject,
			t.audience, t.issuer, t.scope,
			t.issued_at, t.expires_at, t.last_used_at,
			t.revoked_at, t.revoked_by, t.revocation_reason,
			t.ip_address, t.user_agent, t.device_id, t.usage_count,
			COALESCE(s.subscriber_name, 'Unknown') as subscriber_name
		FROM tokens t
		LEFT JOIN subscribers s ON t.tenant_id = s.tenant_id
		WHERE t.tenant_id = $1 AND t.token_id = $2
	`

	var t Token
	err := r.db.QueryRow(ctx, query, tenantID, tokenID).Scan(
		&t.ID, &t.TokenID, &t.TenantID, &t.TokenType, &t.Subject,
		&t.Audience, &t.Issuer, &t.Scope,
		&t.IssuedAt, &t.ExpiresAt, &t.LastUsedAt,
		&t.RevokedAt, &t.RevokedBy, &t.RevocationReason,
		&t.IPAddress, &t.UserAgent, &t.DeviceID, &t.UsageCount,
		&t.SubscriberName,
	)

	if err == pgx.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return &t, nil
}

// RevokeToken marks a token as revoked
func (r *Repository) RevokeToken(ctx context.Context, tenantID, tokenID, revokedBy, reason string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}
	query := `
		UPDATE tokens
		SET revoked_at = NOW(), revoked_by = $3, revocation_reason = $4
		WHERE tenant_id = $1 AND token_id = $2 AND revoked_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, tenantID, tokenID, revokedBy, reason)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	if result.RowsAffected() == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// AddToBlacklist adds a token to the blacklist (for Redis sync)
func (r *Repository) AddToBlacklist(ctx context.Context, entry *BlacklistEntry) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}
	query := `
		INSERT INTO token_blacklist (token_id, tenant_id, reason, revoked_at, revoked_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (token_id, tenant_id) DO UPDATE
		SET reason = EXCLUDED.reason, revoked_at = EXCLUDED.revoked_at, revoked_by = EXCLUDED.revoked_by
		RETURNING id
	`

	err := r.db.QueryRow(
		ctx, query,
		entry.TokenID, entry.TenantID, entry.Reason, entry.RevokedAt, entry.RevokedBy, entry.ExpiresAt,
	).Scan(&entry.ID)

	if err != nil {
		return fmt.Errorf("failed to add to blacklist: %w", err)
	}

	return nil
}

// IsBlacklisted checks if a token is blacklisted
func (r *Repository) IsBlacklisted(ctx context.Context, tenantID, tokenID string) (bool, error) {
	if r.db == nil {
		return false, nil // Assume safe in degraded mode
	}
	query := `
		SELECT EXISTS(
			SELECT 1 FROM token_blacklist
			WHERE tenant_id = $1 AND token_id = $2 AND expires_at > NOW()
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, tenantID, tokenID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}

	return exists, nil
}

// UpdateLastUsed updates the last_used_at timestamp and increments usage_count
func (r *Repository) UpdateLastUsed(ctx context.Context, tenantID, tokenID string) error {
	if r.db == nil {
		return nil // No-op in degraded mode
	}
	query := `
		UPDATE tokens
		SET last_used_at = NOW(), usage_count = usage_count + 1
		WHERE tenant_id = $1 AND token_id = $2
	`

	_, err := r.db.Exec(ctx, query, tenantID, tokenID)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}

	return nil
}

// GetTokenMetrics retrieves aggregated token statistics
func (r *Repository) GetTokenMetrics(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	if r.db == nil {
		// Return empty metrics in degraded mode
		return map[string]interface{}{
			"total_tokens":    0,
			"active_tokens":   0,
			"expired_tokens":  0,
			"revoked_tokens":  0,
			"tokens_per_day":  0,
			"top_subscribers": []map[string]interface{}{},
			"token_types": map[string]int{
				"access":  0,
				"refresh": 0,
				"api_key": 0,
			},
			"recent_activity": []map[string]interface{}{},
		}, nil
	}
	query := `
		SELECT
			COUNT(*) as total_tokens,
			COUNT(*) FILTER (WHERE revoked_at IS NULL AND expires_at > NOW() as active_tokens,
			COUNT(*) FILTER (WHERE revoked_at IS NULL AND expires_at <= NOW() as expired_tokens,
			COUNT(*) FILTER (WHERE revoked_at IS NOT NULL) as revoked_tokens,
			COUNT(*) FILTER (WHERE token_type = 'access') as access_tokens,
			COUNT(*) FILTER (WHERE token_type = 'refresh') as refresh_tokens,
			COUNT(*) FILTER (WHERE token_type = 'api_key') as api_key_tokens,
			COUNT(*) FILTER (WHERE issued_at >= NOW() - INTERVAL '24 hours') as tokens_last_24h
		FROM tokens
		WHERE tenant_id = $1
	`

	var metrics struct {
		Total, Active, Expired, Revoked        int
		Access, Refresh, APIKey, TokensLast24h int
	}

	err := r.db.QueryRow(ctx, query, tenantID).Scan(
		&metrics.Total, &metrics.Active, &metrics.Expired, &metrics.Revoked,
		&metrics.Access, &metrics.Refresh, &metrics.APIKey, &metrics.TokensLast24h,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get token metrics: %w", err)
	}

	// Get top subscribers
	topSubsQuery := `
		SELECT s.subscriber_id, s.subscriber_name, COUNT(*) as token_count
		FROM tokens t
		JOIN subscribers s ON t.tenant_id = s.tenant_id
		WHERE t.tenant_id = $1
		GROUP BY s.subscriber_id, s.subscriber_name
		ORDER BY token_count DESC
		LIMIT 5
	`

	rows, err := r.db.Query(ctx, topSubsQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get top subscribers: %w", err)
	}
	defer rows.Close()

	var topSubscribers []map[string]interface{}
	for rows.Next() {
		var subID, subName string
		var count int
		if err := rows.Scan(&subID, &subName, &count); err != nil {
			continue
		}
		topSubscribers = append(topSubscribers, map[string]interface{}{
			"id":          subID,
			"name":        subName,
			"token_count": count,
		})
	}

	// Get recent activity
	activityQuery := `
		SELECT issued_at, 'created' as action, COALESCE(s.subscriber_name, 'Unknown') as subscriber
		FROM tokens t
		LEFT JOIN subscribers s ON t.tenant_id = s.tenant_id
		WHERE t.tenant_id = $1
		ORDER BY issued_at DESC
		LIMIT 10
	`

	rows2, err := r.db.Query(ctx, activityQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent activity: %w", err)
	}
	defer rows2.Close()

	var recentActivity []map[string]interface{}
	for rows2.Next() {
		var timestamp time.Time
		var action, subscriber string
		if err := rows2.Scan(&timestamp, &action, &subscriber); err != nil {
			continue
		}
		recentActivity = append(recentActivity, map[string]interface{}{
			"timestamp":  timestamp,
			"action":     action,
			"subscriber": subscriber,
		})
	}

	return map[string]interface{}{
		"total_tokens":    metrics.Total,
		"active_tokens":   metrics.Active,
		"expired_tokens":  metrics.Expired,
		"revoked_tokens":  metrics.Revoked,
		"tokens_per_day":  metrics.TokensLast24h,
		"top_subscribers": topSubscribers,
		"token_types": map[string]int{
			"access":  metrics.Access,
			"refresh": metrics.Refresh,
			"api_key": metrics.APIKey,
		},
		"recent_activity": recentActivity,
	}, nil
}

// DeleteExpiredTokens removes tokens that have expired beyond retention period
func (r *Repository) DeleteExpiredTokens(ctx context.Context, retentionDays int) (int64, error) {
	query := `
		DELETE FROM tokens
		WHERE expires_at < NOW() - INTERVAL '1 day' * $1
		AND revoked_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	return result.RowsAffected(), nil
}

// CleanupBlacklist removes expired entries from blacklist
func (r *Repository) CleanupBlacklist(ctx context.Context) (int64, error) {
	query := `DELETE FROM token_blacklist WHERE expires_at < NOW()`

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup blacklist: %w", err)
	}

	return result.RowsAffected(), nil
}
