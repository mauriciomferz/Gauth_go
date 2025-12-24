package scim

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for SCIM users
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new SCIM repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// UserDB represents the database model
type UserDB struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	ExternalID string    `json:"external_id"`
	UserName   string    `json:"username"`
	GivenName  string    `json:"given_name"`
	FamilyName string    `json:"family_name"`
	Email      string    `json:"email"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CreateUser inserts a user
func (r *Repository) CreateUser(ctx context.Context, u *UserDB) error {
	if r.db == nil {
		return fmt.Errorf("database unavailable")
	}
	query := `
		INSERT INTO users (
			id, tenant_id, external_id, username, given_name, family_name, email, active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		u.ID, u.TenantID, u.ExternalID, u.UserName, u.GivenName, u.FamilyName, u.Email, u.Active,
	).Scan(&u.CreatedAt, &u.UpdatedAt)
}

// GetUser retrieves a user by ID
func (r *Repository) GetUser(ctx context.Context, tenantID, userID string) (*UserDB, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	query := `
		SELECT id, tenant_id, external_id, username, given_name, family_name, email, active, created_at, updated_at
		FROM users WHERE id = $1 AND tenant_id = $2
	`
	var u UserDB
	err := r.db.QueryRow(ctx, query, userID, tenantID).Scan(
		&u.ID, &u.TenantID, &u.ExternalID, &u.UserName, &u.GivenName, &u.FamilyName, &u.Email, &u.Active, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ListUsers lists users (simple pagination)
func (r *Repository) ListUsers(ctx context.Context, tenantID string, limit, offset int) ([]UserDB, int, error) {
	if r.db == nil {
		return nil, 0, fmt.Errorf("database unavailable")
	}
	countQuery := `SELECT COUNT(*) FROM users WHERE tenant_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, tenant_id, external_id, username, given_name, family_name, email, active, created_at, updated_at
		FROM users WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []UserDB
	for rows.Next() {
		var u UserDB
		if err := rows.Scan(
			&u.ID, &u.TenantID, &u.ExternalID, &u.UserName, &u.GivenName, &u.FamilyName, &u.Email, &u.Active, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, total, nil
}

// ---------------------------------------------------------------------
// SCIM Client Repository Methods
// ---------------------------------------------------------------------

// CreateClient creates a new SCIM client
func (r *Repository) CreateClient(ctx context.Context, c *SCIMClient) error {
	if r.db == nil {
		return fmt.Errorf("database unavailable")
	}
	query := `
		INSERT INTO scim_clients (
			id, tenant_id, client_name, token_id, scim_base_url, is_active, permissions
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`
	// Simple JSON scan for permissions (mock for now if struct scan fails, but pgx handles []string -> jsonb usually via driver)
	// We might need to manually Marshal if pgx doesn't infer. Assuming standard behavior for now.
	return r.db.QueryRow(ctx, query,
		c.ID, c.TenantID, c.ClientName, c.TokenID, c.SCIMBaseURL, c.IsActive, c.Permissions,
	).Scan(&c.CreatedAt)
}

// ListClients returns all SCIM clients for a tenant
func (r *Repository) ListClients(ctx context.Context, tenantID string) ([]SCIMClient, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	query := `
		SELECT id, tenant_id, client_name, token_id, scim_base_url, is_active, permissions, created_at, last_used_at
		FROM scim_clients
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []SCIMClient
	for rows.Next() {
		var c SCIMClient
		var lastUsedAt *time.Time // Handle nullable

		err := rows.Scan(
			&c.ID, &c.TenantID, &c.ClientName, &c.TokenID, &c.SCIMBaseURL, &c.IsActive, &c.Permissions, &c.CreatedAt, &lastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scim client: %w", err)
		}
		if lastUsedAt != nil {
			c.LastUsedAt = *lastUsedAt
		}
		clients = append(clients, c)
	}
	return clients, nil
}

// DeleteClient deletes a SCIM client
func (r *Repository) DeleteClient(ctx context.Context, tenantID, id string) error {
	if r.db == nil {
		return fmt.Errorf("database unavailable")
	}
	query := `DELETE FROM scim_clients WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, id, tenantID)
	return err
}
