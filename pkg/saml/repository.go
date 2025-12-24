package saml

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for SAML providers
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new SAML repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// SAMLProvider represents a SAML Identity Provider configuration
type SAMLProvider struct {
	ID                   string                 `json:"id"`
	TenantID             string                 `json:"tenantId"`
	ProviderName         string                 `json:"providerName"`
	DisplayName          string                 `json:"displayName"`
	EntityID             string                 `json:"entityId"`
	SSOURL               string                 `json:"ssoUrl"`
	SLOURL               *string                `json:"sloUrl"`
	Certificate          string                 `json:"certificate"`
	RequestSigningCert   *string                `json:"requestSigningCert,omitempty"`
	RequestSigningKey    *string                `json:"requestSigningKey,omitempty"`
	SignRequests         bool                   `json:"signRequests"`
	WantAssertionsSigned bool                   `json:"wantAssertionsSigned"`
	WantResponseSigned   bool                   `json:"wantResponseSigned"`
	AttributeMapping     map[string]interface{} `json:"attributeMapping"`
	Status               string                 `json:"status"`
	CreatedAt            time.Time              `json:"createdAt"`
	UpdatedAt            time.Time              `json:"updatedAt"`
	CreatedBy            *string                `json:"createdBy"`
	UpdatedBy            *string                `json:"updatedBy"`
}

// ListProviders returns all SAML providers for a tenant
func (r *Repository) ListProviders(ctx context.Context, tenantID string) ([]SAMLProvider, error) {
	if r.db == nil {
		return []SAMLProvider{}, nil
	}
	query := `
		SELECT 
			id, tenant_id, provider_name, display_name,
			entity_id, sso_url, slo_url, certificate,
			request_signing_cert, request_signing_key,
			sign_requests, want_assertions_signed, want_response_signed,
			attribute_mapping, status,
			created_at, updated_at, created_by, updated_by
		FROM saml_providers
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query saml providers: %w", err)
	}
	defer rows.Close()

	var providers []SAMLProvider
	for rows.Next() {
		var p SAMLProvider
		err := rows.Scan(
			&p.ID, &p.TenantID, &p.ProviderName, &p.DisplayName,
			&p.EntityID, &p.SSOURL, &p.SLOURL, &p.Certificate,
			&p.RequestSigningCert, &p.RequestSigningKey,
			&p.SignRequests, &p.WantAssertionsSigned, &p.WantResponseSigned,
			&p.AttributeMapping, &p.Status,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan saml provider: %w", err)
		}
		// Redact private key in list
		p.RequestSigningKey = nil
		providers = append(providers, p)
	}

	return providers, nil
}

// GetProvider returns a specific SAML provider
func (r *Repository) GetProvider(ctx context.Context, tenantID, providerID string) (*SAMLProvider, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	query := `
		SELECT 
			id, tenant_id, provider_name, display_name,
			entity_id, sso_url, slo_url, certificate,
			request_signing_cert, request_signing_key,
			sign_requests, want_assertions_signed, want_response_signed,
			attribute_mapping, status,
			created_at, updated_at, created_by, updated_by
		FROM saml_providers
		WHERE id = $1 AND tenant_id = $2
	`

	var p SAMLProvider
	err := r.db.QueryRow(ctx, query, providerID, tenantID).Scan(
		&p.ID, &p.TenantID, &p.ProviderName, &p.DisplayName,
		&p.EntityID, &p.SSOURL, &p.SLOURL, &p.Certificate,
		&p.RequestSigningCert, &p.RequestSigningKey,
		&p.SignRequests, &p.WantAssertionsSigned, &p.WantResponseSigned,
		&p.AttributeMapping, &p.Status,
		&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// CreateProvider creates a new SAML provider
func (r *Repository) CreateProvider(ctx context.Context, p *SAMLProvider) error {
	if r.db == nil {
		return fmt.Errorf("database unavailable")
	}
	query := `
		INSERT INTO saml_providers (
			id, tenant_id, provider_name, display_name,
			entity_id, sso_url, slo_url, certificate,
			request_signing_cert, request_signing_key,
			sign_requests, want_assertions_signed, want_response_signed,
			attribute_mapping, status, created_by
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10,
			$11, $12, $13,
			$14, $15, $16
		)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		p.ID, p.TenantID, p.ProviderName, p.DisplayName,
		p.EntityID, p.SSOURL, p.SLOURL, p.Certificate,
		p.RequestSigningCert, p.RequestSigningKey,
		p.SignRequests, p.WantAssertionsSigned, p.WantResponseSigned,
		p.AttributeMapping, p.Status, p.CreatedBy,
	).Scan(&p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create saml provider: %w", err)
	}

	return nil
}

// UpdateProvider updates a SAML provider
func (r *Repository) UpdateProvider(ctx context.Context, p *SAMLProvider) error {
	if r.db == nil {
		return fmt.Errorf("database unavailable")
	}
	// Note: This is a partial update implementation for brevity/reference
	query := `
		UPDATE saml_providers
		SET display_name = $1, entity_id = $2, sso_url = $3, slo_url = $4,
		    certificate = $5, sign_requests = $6, updated_at = NOW(), updated_by = $7
		WHERE id = $8 AND tenant_id = $9
	`

	_, err := r.db.Exec(ctx, query,
		p.DisplayName, p.EntityID, p.SSOURL, p.SLOURL,
		p.Certificate, p.SignRequests, p.UpdatedBy,
		p.ID, p.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update saml provider: %w", err)
	}

	return nil
}

// DeleteProvider deletes a SAML provider
func (r *Repository) DeleteProvider(ctx context.Context, tenantID, providerID string) error {
	if r.db == nil {
		return fmt.Errorf("database unavailable")
	}
	query := `DELETE FROM saml_providers WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.Exec(ctx, query, providerID, tenantID)
	return err
}
