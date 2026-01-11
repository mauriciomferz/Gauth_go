package pip

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// =============================================================================
// Database PIP Constants
// =============================================================================

// AttributeType represents the type of attribute
type AttributeType string

const (
	AttributeTypeString AttributeType = "string"
	AttributeTypeInt    AttributeType = "int"
	AttributeTypeFloat  AttributeType = "float"
	AttributeTypeBool   AttributeType = "bool"
	AttributeTypeJSON   AttributeType = "json"
	AttributeTypeTime   AttributeType = "time"
)

// AttributeCategory represents the category of attribute
type AttributeCategory string

const (
	CategoryIdentity     AttributeCategory = "identity"
	CategoryDocument     AttributeCategory = "document"
	CategoryVerification AttributeCategory = "verification"
	CategoryContact      AttributeCategory = "contact"
	CategoryAddress      AttributeCategory = "address"
	CategoryFinancial    AttributeCategory = "financial"
	CategoryCustom       AttributeCategory = "custom"
)

// =============================================================================
// Data Models
// =============================================================================

// Attribute represents a user attribute in the PIP
type Attribute struct {
	ID         string            `json:"id"`
	UserID     string            `json:"user_id"`
	Name       string            `json:"name"`
	Value      interface{}       `json:"value"`
	Type       AttributeType     `json:"type"`
	Category   AttributeCategory `json:"category"`
	Source     string            `json:"source"` // e.g., "us_verifier", "de_eid", "uk_verify"
	Verified   bool              `json:"verified"`
	VerifiedAt *time.Time        `json:"verified_at,omitempty"`
	VerifiedBy string            `json:"verified_by,omitempty"` // Verifier ID
	ExpiresAt  *time.Time        `json:"expires_at,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// AttributeQuery represents a query for attributes
type AttributeQuery struct {
	UserID     string              `json:"user_id,omitempty"`
	Names      []string            `json:"names,omitempty"`
	Categories []AttributeCategory `json:"categories,omitempty"`
	Verified   *bool               `json:"verified,omitempty"`
	Source     string              `json:"source,omitempty"`
	NotExpired bool                `json:"not_expired,omitempty"`
	Limit      int                 `json:"limit,omitempty"`
	Offset     int                 `json:"offset,omitempty"`
}

// AttributeUpdate represents an update to an attribute
type AttributeUpdate struct {
	Value      interface{}       `json:"value,omitempty"`
	Verified   *bool             `json:"verified,omitempty"`
	VerifiedAt *time.Time        `json:"verified_at,omitempty"`
	VerifiedBy string            `json:"verified_by,omitempty"`
	ExpiresAt  *time.Time        `json:"expires_at,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// UserAttributeSet represents a collection of attributes for a user
type UserAttributeSet struct {
	UserID     string       `json:"user_id"`
	Attributes []*Attribute `json:"attributes"`
	Count      int          `json:"count"`
	FetchedAt  time.Time    `json:"fetched_at"`
}

// =============================================================================
// Database PIP Configuration
// =============================================================================

// DatabasePIPConfig contains configuration for the database PIP
type DatabasePIPConfig struct {
	// Database connection
	DSN             string        `json:"dsn"` // Data Source Name
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`

	// Connection retry
	MaxRetries   int           `json:"max_retries"`
	RetryDelay   time.Duration `json:"retry_delay"`
	RetryBackoff float64       `json:"retry_backoff"`

	// Caching
	CacheEnabled bool          `json:"cache_enabled"`
	CacheTTL     time.Duration `json:"cache_ttl"`
	CacheMaxSize int           `json:"cache_max_size"`

	// Query timeout
	QueryTimeout time.Duration `json:"query_timeout"`

	// Auto-cleanup
	CleanupEnabled   bool          `json:"cleanup_enabled"`
	CleanupInterval  time.Duration `json:"cleanup_interval"`
	ExpiredRetention time.Duration `json:"expired_retention"` // How long to keep expired attributes
}

// =============================================================================
// Database PIP Implementation
// =============================================================================

// DatabasePIP implements a PostgreSQL-backed Policy Information Point
type DatabasePIP struct {
	config    *DatabasePIPConfig
	db        *sql.DB
	cache     map[string]*cachedAttributeSet
	cacheMu   sync.RWMutex
	cleanupCh chan struct{}
	wg        sync.WaitGroup
}

// cachedAttributeSet stores cached attributes
type cachedAttributeSet struct {
	Attributes []*Attribute
	Timestamp  time.Time
}

// NewDatabasePIP creates a new database-backed PIP
func NewDatabasePIP(config *DatabasePIPConfig) (*DatabasePIP, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if config.DSN == "" {
		return nil, errors.New("DSN is required")
	}

	// Set defaults
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 25
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 5
	}
	if config.ConnMaxLifetime == 0 {
		config.ConnMaxLifetime = 5 * time.Minute
	}
	if config.ConnMaxIdleTime == 0 {
		config.ConnMaxIdleTime = 2 * time.Minute
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.RetryBackoff == 0 {
		config.RetryBackoff = 2.0
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}
	if config.CacheMaxSize == 0 {
		config.CacheMaxSize = 1000
	}
	if config.QueryTimeout == 0 {
		config.QueryTimeout = 10 * time.Second
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Hour
	}
	if config.ExpiredRetention == 0 {
		config.ExpiredRetention = 30 * 24 * time.Hour // 30 days
	}

	// Open database connection
	db, err := sql.Open("postgres", config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pip := &DatabasePIP{
		config:    config,
		db:        db,
		cache:     make(map[string]*cachedAttributeSet),
		cleanupCh: make(chan struct{}),
	}

	// Start cleanup routine if enabled
	if config.CleanupEnabled {
		pip.wg.Add(1)
		go pip.cleanupRoutine()
	}

	return pip, nil
}

// Close closes the database connection and stops background routines
func (p *DatabasePIP) Close() error {
	// Stop cleanup routine
	if p.config.CleanupEnabled {
		close(p.cleanupCh)
		p.wg.Wait()
	}

	// Close database
	return p.db.Close()
}

// InitSchema initializes the database schema
func (p *DatabasePIP) InitSchema(ctx context.Context) error {
	schema := `
		CREATE TABLE IF NOT EXISTS pip_attributes (
			id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			value TEXT NOT NULL,
			type VARCHAR(50) NOT NULL,
			category VARCHAR(50) NOT NULL,
			source VARCHAR(255) NOT NULL,
			verified BOOLEAN NOT NULL DEFAULT FALSE,
			verified_at TIMESTAMPTZ,
			verified_by VARCHAR(255),
			expires_at TIMESTAMPTZ,
			metadata JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		
		CREATE INDEX IF NOT EXISTS idx_pip_attributes_user_id ON pip_attributes(user_id);
		CREATE INDEX IF NOT EXISTS idx_pip_attributes_name ON pip_attributes(name);
		CREATE INDEX IF NOT EXISTS idx_pip_attributes_category ON pip_attributes(category);
		CREATE INDEX IF NOT EXISTS idx_pip_attributes_verified ON pip_attributes(verified);
		CREATE INDEX IF NOT EXISTS idx_pip_attributes_source ON pip_attributes(source);
		CREATE INDEX IF NOT EXISTS idx_pip_attributes_expires_at ON pip_attributes(expires_at);
		CREATE INDEX IF NOT EXISTS idx_pip_attributes_user_name ON pip_attributes(user_id, name);
		
		CREATE TABLE IF NOT EXISTS pip_audit_log (
			id SERIAL PRIMARY KEY,
			attribute_id VARCHAR(255) NOT NULL,
			user_id VARCHAR(255) NOT NULL,
			action VARCHAR(50) NOT NULL,
			old_value TEXT,
			new_value TEXT,
			changed_by VARCHAR(255),
			changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			metadata JSONB
		);
		
		CREATE INDEX IF NOT EXISTS idx_pip_audit_log_attribute_id ON pip_audit_log(attribute_id);
		CREATE INDEX IF NOT EXISTS idx_pip_audit_log_user_id ON pip_audit_log(user_id);
		CREATE INDEX IF NOT EXISTS idx_pip_audit_log_action ON pip_audit_log(action);
		CREATE INDEX IF NOT EXISTS idx_pip_audit_log_changed_at ON pip_audit_log(changed_at);
	`

	ctx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	_, err := p.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}

// =============================================================================
// CRUD Operations
// =============================================================================

// CreateAttribute creates a new attribute
func (p *DatabasePIP) CreateAttribute(ctx context.Context, attr *Attribute) error {
	if attr == nil {
		return errors.New("attribute cannot be nil")
	}

	// Generate ID if not provided
	if attr.ID == "" {
		attr.ID = fmt.Sprintf("%s_%s_%d", attr.UserID, attr.Name, time.Now().UnixNano())
	}

	// Set timestamps
	now := time.Now()
	attr.CreatedAt = now
	attr.UpdatedAt = now

	// Serialize value
	valueJSON, err := json.Marshal(attr.Value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Serialize metadata
	metadataJSON, err := json.Marshal(attr.Metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	query := `
		INSERT INTO pip_attributes (
			id, user_id, name, value, type, category, source,
			verified, verified_at, verified_by, expires_at, metadata,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	ctx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	_, err = p.db.ExecContext(ctx, query,
		attr.ID, attr.UserID, attr.Name, string(valueJSON), attr.Type, attr.Category,
		attr.Source, attr.Verified, attr.VerifiedAt, attr.VerifiedBy, attr.ExpiresAt,
		string(metadataJSON), attr.CreatedAt, attr.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create attribute: %w", err)
	}

	// Invalidate cache
	p.invalidateCache(attr.UserID)

	// Audit log
	p.logAudit(ctx, attr.ID, attr.UserID, "CREATE", "", string(valueJSON), "")

	return nil
}

// GetAttribute retrieves an attribute by ID
func (p *DatabasePIP) GetAttribute(ctx context.Context, id string) (*Attribute, error) {
	query := `
		SELECT id, user_id, name, value, type, category, source,
		       verified, verified_at, verified_by, expires_at, metadata,
		       created_at, updated_at
		FROM pip_attributes
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	var attr Attribute
	var valueJSON, metadataJSON string

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&attr.ID, &attr.UserID, &attr.Name, &valueJSON, &attr.Type, &attr.Category,
		&attr.Source, &attr.Verified, &attr.VerifiedAt, &attr.VerifiedBy, &attr.ExpiresAt,
		&metadataJSON, &attr.CreatedAt, &attr.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("attribute not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attribute: %w", err)
	}

	// Deserialize value
	if err := json.Unmarshal([]byte(valueJSON), &attr.Value); err != nil {
		return nil, fmt.Errorf("failed to deserialize value: %w", err)
	}

	// Deserialize metadata
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &attr.Metadata); err != nil {
			return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
		}
	}

	return &attr, nil
}

// GetUserAttributes retrieves all attributes for a user based on query
func (p *DatabasePIP) GetUserAttributes(ctx context.Context, query *AttributeQuery) ([]*Attribute, error) {
	if query == nil || query.UserID == "" {
		return nil, errors.New("user_id is required")
	}

	// Check cache
	if p.config.CacheEnabled {
		cacheKey := p.getCacheKey(query.UserID)
		if cached := p.getFromCache(cacheKey); cached != nil {
			return p.filterAttributes(cached, query), nil
		}
	}

	// Build SQL query
	sqlQuery, args := p.buildQuery(query)

	ctx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	rows, err := p.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query attributes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var attributes []*Attribute

	for rows.Next() {
		var attr Attribute
		var valueJSON, metadataJSON string

		err := rows.Scan(
			&attr.ID, &attr.UserID, &attr.Name, &valueJSON, &attr.Type, &attr.Category,
			&attr.Source, &attr.Verified, &attr.VerifiedAt, &attr.VerifiedBy, &attr.ExpiresAt,
			&metadataJSON, &attr.CreatedAt, &attr.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attribute: %w", err)
		}

		// Deserialize value
		if err := json.Unmarshal([]byte(valueJSON), &attr.Value); err != nil {
			return nil, fmt.Errorf("failed to deserialize value: %w", err)
		}

		// Deserialize metadata
		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &attr.Metadata); err != nil {
				return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
			}
		}

		attributes = append(attributes, &attr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attributes: %w", err)
	}

	// Cache results
	if p.config.CacheEnabled && len(attributes) > 0 {
		p.addToCache(p.getCacheKey(query.UserID), attributes)
	}

	return attributes, nil
}

// UpdateAttribute updates an existing attribute
func (p *DatabasePIP) UpdateAttribute(ctx context.Context, id string, update *AttributeUpdate) error {
	if update == nil {
		return errors.New("update cannot be nil")
	}

	// Get current attribute for audit
	current, err := p.GetAttribute(ctx, id)
	if err != nil {
		return err
	}

	// Build update query dynamically
	query := "UPDATE pip_attributes SET updated_at = NOW()"
	args := []interface{}{}
	argPos := 1

	if update.Value != nil {
		valueJSON, marshalErr := json.Marshal(update.Value)
		if marshalErr != nil {
			return fmt.Errorf("failed to serialize value: %w", marshalErr)
		}
		query += fmt.Sprintf(", value = $%d", argPos)
		args = append(args, string(valueJSON))
		argPos++
	}

	if update.Verified != nil {
		query += fmt.Sprintf(", verified = $%d", argPos)
		args = append(args, *update.Verified)
		argPos++
	}

	if update.VerifiedAt != nil {
		query += fmt.Sprintf(", verified_at = $%d", argPos)
		args = append(args, *update.VerifiedAt)
		argPos++
	}

	if update.VerifiedBy != "" {
		query += fmt.Sprintf(", verified_by = $%d", argPos)
		args = append(args, update.VerifiedBy)
		argPos++
	}

	if update.ExpiresAt != nil {
		query += fmt.Sprintf(", expires_at = $%d", argPos)
		args = append(args, *update.ExpiresAt)
		argPos++
	}

	if update.Metadata != nil {
		metadataJSON, marshalErr := json.Marshal(update.Metadata)
		if marshalErr != nil {
			return fmt.Errorf("failed to serialize metadata: %w", marshalErr)
		}
		query += fmt.Sprintf(", metadata = $%d", argPos)
		args = append(args, string(metadataJSON))
		argPos++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argPos)
	args = append(args, id)

	ctx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	result, err := p.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update attribute: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("attribute not found: %s", id)
	}

	// Invalidate cache
	p.invalidateCache(current.UserID)

	// Audit log
	oldValue, _ := json.Marshal(current.Value)
	newValue, _ := json.Marshal(update.Value)
	p.logAudit(ctx, id, current.UserID, "UPDATE", string(oldValue), string(newValue), update.VerifiedBy)

	return nil
}

// DeleteAttribute deletes an attribute
func (p *DatabasePIP) DeleteAttribute(ctx context.Context, id string) error {
	// Get attribute for audit
	attr, err := p.GetAttribute(ctx, id)
	if err != nil {
		return err
	}

	query := "DELETE FROM pip_attributes WHERE id = $1"

	ctx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete attribute: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("attribute not found: %s", id)
	}

	// Invalidate cache
	p.invalidateCache(attr.UserID)

	// Audit log
	value, _ := json.Marshal(attr.Value)
	p.logAudit(ctx, id, attr.UserID, "DELETE", string(value), "", "")

	return nil
}

// DeleteUserAttributes deletes all attributes for a user
func (p *DatabasePIP) DeleteUserAttributes(ctx context.Context, userID string) error {
	query := "DELETE FROM pip_attributes WHERE user_id = $1"

	ctx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	_, err := p.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user attributes: %w", err)
	}

	// Invalidate cache
	p.invalidateCache(userID)

	return nil
}

// =============================================================================
// Helper Methods
// =============================================================================

func (p *DatabasePIP) buildQuery(query *AttributeQuery) (string, []interface{}) {
	sql := `
		SELECT id, user_id, name, value, type, category, source,
		       verified, verified_at, verified_by, expires_at, metadata,
		       created_at, updated_at
		FROM pip_attributes
		WHERE user_id = $1
	`
	args := []interface{}{query.UserID}
	argPos := 2

	if len(query.Names) > 0 {
		sql += fmt.Sprintf(" AND name = ANY($%d)", argPos)
		args = append(args, query.Names)
		argPos++
	}

	if len(query.Categories) > 0 {
		sql += fmt.Sprintf(" AND category = ANY($%d)", argPos)
		args = append(args, query.Categories)
		argPos++
	}

	if query.Verified != nil {
		sql += fmt.Sprintf(" AND verified = $%d", argPos)
		args = append(args, *query.Verified)
		argPos++
	}

	if query.Source != "" {
		sql += fmt.Sprintf(" AND source = $%d", argPos)
		args = append(args, query.Source)
		argPos++
	}

	if query.NotExpired {
		sql += " AND (expires_at IS NULL OR expires_at > NOW())"
	}

	sql += " ORDER BY created_at DESC"

	if query.Limit > 0 {
		sql += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, query.Limit)
		argPos++
	}

	if query.Offset > 0 {
		sql += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, query.Offset)
	}

	return sql, args
}

func (p *DatabasePIP) filterAttributes(attrs []*Attribute, query *AttributeQuery) []*Attribute {
	var filtered []*Attribute

	for _, attr := range attrs {
		// Apply filters
		if len(query.Names) > 0 {
			found := false
			for _, name := range query.Names {
				if attr.Name == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if len(query.Categories) > 0 {
			found := false
			for _, cat := range query.Categories {
				if attr.Category == cat {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if query.Verified != nil && attr.Verified != *query.Verified {
			continue
		}

		if query.Source != "" && attr.Source != query.Source {
			continue
		}

		if query.NotExpired && attr.ExpiresAt != nil && time.Now().After(*attr.ExpiresAt) {
			continue
		}

		filtered = append(filtered, attr)
	}

	// Apply limit and offset
	if query.Offset > 0 && query.Offset < len(filtered) {
		filtered = filtered[query.Offset:]
	}

	if query.Limit > 0 && query.Limit < len(filtered) {
		filtered = filtered[:query.Limit]
	}

	return filtered
}

func (p *DatabasePIP) logAudit(ctx context.Context, attrID, userID, action, oldValue, newValue, changedBy string) {
	query := `
		INSERT INTO pip_audit_log (attribute_id, user_id, action, old_value, new_value, changed_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	ctx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	_, _ = p.db.ExecContext(ctx, query, attrID, userID, action, oldValue, newValue, changedBy)
}

// =============================================================================
// Cache Management
// =============================================================================

func (p *DatabasePIP) getCacheKey(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

func (p *DatabasePIP) getFromCache(key string) []*Attribute {
	if !p.config.CacheEnabled {
		return nil
	}

	p.cacheMu.RLock()
	defer p.cacheMu.RUnlock()

	cached, exists := p.cache[key]
	if !exists {
		return nil
	}

	if time.Since(cached.Timestamp) > p.config.CacheTTL {
		return nil
	}

	return cached.Attributes
}

func (p *DatabasePIP) addToCache(key string, attributes []*Attribute) {
	if !p.config.CacheEnabled {
		return
	}

	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	// Check cache size
	if len(p.cache) >= p.config.CacheMaxSize {
		// Remove oldest entry
		var oldestKey string
		var oldestTime time.Time
		first := true

		for k, v := range p.cache {
			if first || v.Timestamp.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.Timestamp
				first = false
			}
		}

		delete(p.cache, oldestKey)
	}

	p.cache[key] = &cachedAttributeSet{
		Attributes: attributes,
		Timestamp:  time.Now(),
	}
}

func (p *DatabasePIP) invalidateCache(userID string) {
	if !p.config.CacheEnabled {
		return
	}

	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	delete(p.cache, p.getCacheKey(userID))
}

// =============================================================================
// Background Cleanup
// =============================================================================

func (p *DatabasePIP) cleanupRoutine() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), p.config.QueryTimeout)
			p.cleanupExpiredAttributes(ctx)
			cancel()
		case <-p.cleanupCh:
			return
		}
	}
}

func (p *DatabasePIP) cleanupExpiredAttributes(ctx context.Context) {
	cutoff := time.Now().Add(-p.config.ExpiredRetention)

	query := "DELETE FROM pip_attributes WHERE expires_at IS NOT NULL AND expires_at < $1"

	_, _ = p.db.ExecContext(ctx, query, cutoff)
}
