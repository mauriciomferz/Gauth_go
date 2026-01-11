package poa

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles Proof of Authorization data operations
type Repository struct {
	db          *pgxpool.Pool
	authChecker AuthorizationChecker
}

// NewRepository creates a new PoA repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db:          db,
		authChecker: nil, // Authorization checking disabled by default for backward compatibility
	}
}

// SetAuthorizationChecker enables privilege escalation protection by validating
// that the grantor (principal) actually holds the permissions being delegated.
//
// SECURITY RECOMMENDATION: Enable this in production to prevent CVE-2025-AGENTAUTH-005.
// Without this, a user with "Editor" role can delegate "Admin" permissions if they
// can forge a valid signature (e.g., compromised key, social engineering).
//
// Example:
//
//	repo := NewRepository(db)
//	authChecker := NewDefaultAuthorizationChecker(repo)
//	repo.SetAuthorizationChecker(authChecker)
func (r *Repository) SetAuthorizationChecker(checker AuthorizationChecker) {
	r.authChecker = checker
}

// PoARecord represents a power of attorney delegation in the database
type PoARecord struct {
	ID                 string           `json:"id"`
	TenantID           string           `json:"tenantId"`
	PoAName            string           `json:"poaName"`
	GrantorID          string           `json:"grantorId"`
	GrantorName        string           `json:"grantorName"`
	RepresentativeID   string           `json:"representativeId"`
	RepresentativeName string           `json:"representativeName"`
	RepresentativeType string           `json:"representativeType"`
	ScopeType          string           `json:"scopeType"`
	Actions            []string         `json:"actions"`
	GeographicRegions  []string         `json:"geographicRegions"`
	Status             string           `json:"status"`
	CreatedAt          time.Time        `json:"createdAt"`
	ApprovedAt         *time.Time       `json:"approvedAt,omitempty"`
	ApprovedBy         *string          `json:"approvedBy,omitempty"`
	RevokedAt          *time.Time       `json:"revokedAt,omitempty"`
	RevokedBy          *string          `json:"revokedBy,omitempty"`
	RevocationReason   *string          `json:"revocationReason,omitempty"`
	ValidFrom          time.Time        `json:"validFrom"`
	ValidUntil         time.Time        `json:"validUntil"`
	Conditions         *json.RawMessage `json:"conditions,omitempty"`
	Metadata           *json.RawMessage `json:"metadata,omitempty"`
	MultiSignatures    *json.RawMessage `json:"multiSignatures,omitempty"`
	UpdatedAt          *time.Time       `json:"updatedAt,omitempty"`
}

// PoATemplate represents a power of attorney template
type PoATemplate struct {
	ID                  string           `json:"id"`
	TenantID            *string          `json:"tenantId,omitempty"`
	TemplateName        string           `json:"templateName"`
	Description         *string          `json:"description,omitempty"`
	ScopeType           string           `json:"scopeType"`
	DefaultActions      []string         `json:"defaultActions"`
	DefaultDurationDays *int             `json:"defaultDurationDays,omitempty"`
	ConditionsSchema    *json.RawMessage `json:"conditionsSchema,omitempty"`
	CreatedAt           time.Time        `json:"createdAt"`
	CreatedBy           *string          `json:"createdBy,omitempty"`
	IsSystemTemplate    bool             `json:"isSystemTemplate"`
}

// PoAStats represents aggregate statistics for power of attorneys
type PoAStats struct {
	TotalPoAs       int                      `json:"totalPoas"`
	ActivePoAs      int                      `json:"activePoas"`
	PendingPoAs     int                      `json:"pendingPoas"`
	ExpiredPoAs     int                      `json:"expiredPoas"`
	RevokedPoAs     int                      `json:"revokedPoas"`
	ByRepType       map[string]int           `json:"byRepresentativeType"`
	TopActions      []ActionCount            `json:"topActions"`
	GeoDistribution []GeographicDistribution `json:"geographicDistribution"`
}

// ActionCount represents count of a specific action
type ActionCount struct {
	Action string `json:"action"`
	Count  int    `json:"count"`
}

// GeographicDistribution represents count by region
type GeographicDistribution struct {
	Region string `json:"region"`
	Count  int    `json:"count"`
}

// CreatePoA creates a new power of attorney record
func (r *Repository) CreatePoA(ctx context.Context, poa *PoARecord) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}
	query := `
		INSERT INTO power_of_attorney (
			tenant_id, poa_name, grantor_id, grantor_name,
			representative_id, representative_name, representative_type,
			scope_type, actions, geographic_regions,
			status, valid_from, valid_until, conditions, metadata, multi_signatures, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		poa.TenantID, poa.PoAName, poa.GrantorID, poa.GrantorName,
		poa.RepresentativeID, poa.RepresentativeName, poa.RepresentativeType,
		poa.ScopeType, poa.Actions, poa.GeographicRegions,
		poa.Status, poa.ValidFrom, poa.ValidUntil, poa.Conditions, poa.Metadata, poa.MultiSignatures, time.Now(),
	).Scan(&poa.ID, &poa.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create PoA: %w", err)
	}

	return nil
}

// ListPoAs retrieves all power of attorney records for a tenant with pagination
func (r *Repository) ListPoAs(ctx context.Context, tenantID string, limit, offset int) ([]PoARecord, int, error) {
	if r.db == nil {
		return nil, 0, nil
	}
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM power_of_attorney WHERE tenant_id = $1`
	if err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count PoAs: %w", err)
	}

	// Get paginated records
	query := `
		SELECT 
			id, tenant_id, poa_name, grantor_id, grantor_name,
			representative_id, representative_name, representative_type,
			scope_type, actions, geographic_regions,
			status, created_at, approved_at, approved_by,
			revoked_at, revoked_by, revocation_reason,
			valid_from, valid_until, conditions, metadata, multi_signatures, updated_at
		FROM power_of_attorney
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list PoAs: %w", err)
	}
	var poas []PoARecord
	for rows.Next() {
		var poa PoARecord
		var conditionsJSON, metadataJSON, multiSignaturesJSON *[]byte
		err := rows.Scan(
			&poa.ID, &poa.TenantID, &poa.PoAName, &poa.GrantorID, &poa.GrantorName,
			&poa.RepresentativeID, &poa.RepresentativeName, &poa.RepresentativeType,
			&poa.ScopeType, &poa.Actions, &poa.GeographicRegions,
			&poa.Status, &poa.CreatedAt, &poa.ApprovedAt, &poa.ApprovedBy,
			&poa.RevokedAt, &poa.RevokedBy, &poa.RevocationReason,
			&poa.ValidFrom, &poa.ValidUntil, &conditionsJSON, &metadataJSON, &multiSignaturesJSON, &poa.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan PoA: %w", err)
		}
		if conditionsJSON != nil && len(*conditionsJSON) > 0 {
			raw := json.RawMessage(*conditionsJSON)
			poa.Conditions = &raw
		}
		if metadataJSON != nil && len(*metadataJSON) > 0 {
			raw := json.RawMessage(*metadataJSON)
			poa.Metadata = &raw
		}
		if multiSignaturesJSON != nil && len(*multiSignaturesJSON) > 0 {
			raw := json.RawMessage(*multiSignaturesJSON)
			poa.MultiSignatures = &raw
		}
		poas = append(poas, poa)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating PoAs: %w", err)
	}

	return poas, total, nil
}

// GetPoA retrieves a specific power of attorney record
func (r *Repository) GetPoA(ctx context.Context, tenantID, poaID string) (*PoARecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	query := `
		SELECT 
			id, tenant_id, poa_name, grantor_id, grantor_name,
			representative_id, representative_name, representative_type,
			scope_type, actions, geographic_regions,
			status, created_at, approved_at, approved_by,
			revoked_at, revoked_by, revocation_reason,
			valid_from, valid_until, conditions, metadata, multi_signatures, updated_at
		FROM power_of_attorney
		WHERE tenant_id = $1 AND id = $2
	`

	var poa PoARecord
	err := r.db.QueryRow(ctx, query, tenantID, poaID).Scan(
		&poa.ID, &poa.TenantID, &poa.PoAName, &poa.GrantorID, &poa.GrantorName,
		&poa.RepresentativeID, &poa.RepresentativeName, &poa.RepresentativeType,
		&poa.ScopeType, &poa.Actions, &poa.GeographicRegions,
		&poa.Status, &poa.CreatedAt, &poa.ApprovedAt, &poa.ApprovedBy,
		&poa.RevokedAt, &poa.RevokedBy, &poa.RevocationReason,
		&poa.ValidFrom, &poa.ValidUntil, &poa.Conditions, &poa.Metadata, &poa.MultiSignatures, &poa.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("PoA not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get PoA: %w", err)
	}

	return &poa, nil
}

// RevokePoA revokes a power of attorney
func (r *Repository) RevokePoA(ctx context.Context, tenantID, poaID, revokedBy, reason string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}
	query := `
		UPDATE power_of_attorney
		SET status = 'revoked',
		    revoked_at = CURRENT_TIMESTAMP,
		    revoked_by = $3,
		    revocation_reason = $4
		WHERE tenant_id = $1 AND id = $2 AND status != 'revoked'
		RETURNING id
	`

	var returnedID string
	err := r.db.QueryRow(ctx, query, tenantID, poaID, revokedBy, reason).Scan(&returnedID)

	if err == pgx.ErrNoRows {
		return fmt.Errorf("PoA not found or already revoked")
	}
	if err != nil {
		return fmt.Errorf("failed to revoke PoA: %w", err)
	}

	return nil
}

// ApprovePoA approves a pending power of attorney
func (r *Repository) ApprovePoA(ctx context.Context, tenantID, poaID, approvedBy string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}
	query := `
		UPDATE power_of_attorney
		SET status = 'active',
		    approved_at = CURRENT_TIMESTAMP,
		    approved_by = $3
		WHERE tenant_id = $1 AND id = $2 AND status = 'pending'
		RETURNING id
	`

	var returnedID string
	err := r.db.QueryRow(ctx, query, tenantID, poaID, approvedBy).Scan(&returnedID)

	if err == pgx.ErrNoRows {
		return fmt.Errorf("PoA not found or not in pending status")
	}
	if err != nil {
		return fmt.Errorf("failed to approve PoA: %w", err)
	}

	return nil
}

// RejectPoA rejects a pending power of attorney
func (r *Repository) RejectPoA(ctx context.Context, tenantID, poaID, rejectedBy, reason string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}
	// Store rejection as revocation with reason
	query := `
		UPDATE power_of_attorney
		SET status = 'revoked',
		    revoked_at = CURRENT_TIMESTAMP,
		    revoked_by = $3,
		    revocation_reason = $4
		WHERE tenant_id = $1 AND id = $2 AND status = 'pending'
		RETURNING id
	`

	var returnedID string
	err := r.db.QueryRow(ctx, query, tenantID, poaID, rejectedBy, "Rejected: "+reason).Scan(&returnedID)

	if err == pgx.ErrNoRows {
		return fmt.Errorf("PoA not found or not in pending status")
	}
	if err != nil {
		return fmt.Errorf("failed to reject PoA: %w", err)
	}

	return nil
}

// ValidatePoA checks if a representative can perform an action for a grantor.
//
// Security Enhancement (CVE-2025-AGENTAUTH-005): Now performs TWO-LEVEL validation:
//  1. Database check: PoA exists, is active, contains requested action (EXISTING)
//  2. Authorization check: Grantor actually holds permissions being delegated (NEW)
//
// This prevents privilege escalation attacks where a delegate requests scopes
// beyond what the principal (grantor) actually possesses in the system.
//
// To enable authorization checking, inject an AuthorizationChecker via SetAuthorizationChecker().
// If no checker is configured, only database validation is performed (legacy behavior).
func (r *Repository) ValidatePoA(
	ctx context.Context,
	tenantID, grantorID, representativeID, action, resource string,
) (*PoARecord, bool, string) {
	if r.db == nil {
		return nil, false, "database unavailable"
	}
	query := `
		SELECT 
			id, tenant_id, poa_name, grantor_id, grantor_name,
			representative_id, representative_name, representative_type,
			scope_type, actions, geographic_regions,
			status, created_at, approved_at, approved_by,
			revoked_at, revoked_by, revocation_reason,
			valid_from, valid_until, conditions, metadata, multi_signatures, updated_at
		FROM power_of_attorney
		WHERE tenant_id = $1 
		  AND grantor_id = $2 
		  AND representative_id = $3
		  AND status = 'active'
		  AND valid_from <= CURRENT_TIMESTAMP
		  AND valid_until >= CURRENT_TIMESTAMP
		  AND $4 = ANY(actions)
		ORDER BY created_at DESC
		LIMIT 1
	`

	var poa PoARecord
	err := r.db.QueryRow(ctx, query, tenantID, grantorID, representativeID, action).Scan(
		&poa.ID, &poa.TenantID, &poa.PoAName, &poa.GrantorID, &poa.GrantorName,
		&poa.RepresentativeID, &poa.RepresentativeName, &poa.RepresentativeType,
		&poa.ScopeType, &poa.Actions, &poa.GeographicRegions,
		&poa.Status, &poa.CreatedAt, &poa.ApprovedAt, &poa.ApprovedBy,
		&poa.RevokedAt, &poa.RevokedBy, &poa.RevocationReason,
		&poa.ValidFrom, &poa.ValidUntil, &poa.Conditions, &poa.Metadata, &poa.MultiSignatures,
	)

	if err == pgx.ErrNoRows {
		return nil, false, "No valid Proof of Authorization found"
	}
	if err != nil {
		return nil, false, fmt.Sprintf("Error validating PoA: %v", err)
	}

	// SECURITY FIX: Verify grantor actually holds the permissions being delegated
	// This prevents privilege escalation where a delegate requests scopes beyond
	// what the principal possesses (e.g., "Editor" delegating "Admin" rights)
	if r.authChecker != nil {
		valid, unauthorized, err := ValidateScopeAuthorization(
			ctx,
			r.authChecker,
			tenantID,
			grantorID,
			poa.Actions, // All actions in the PoA must be authorized
		)

		if err != nil {
			// Fail-closed: Authorization check failure = reject
			fmt.Printf("[POA-SECURITY] Authorization check failed for grantor=%s: %v\n", grantorID, err)
			return nil, false, fmt.Sprintf("Authorization verification failed: %v", err)
		}

		if !valid {
			// Privilege escalation detected
			fmt.Printf("[POA-SECURITY] Privilege escalation blocked: grantor=%s lacks permissions %v\n",
				grantorID, unauthorized)
			return nil, false, fmt.Sprintf("Grantor does not hold required permissions: %v", unauthorized)
		}
	}

	return &poa, true, "Valid Proof of Authorization found"
}

// AddMultiSignature adds a signature to the PoA and transitions to active if threshold matched.
// It uses row-level locking (SELECT FOR UPDATE) to ensure concurrency safety (RR-007).
func (r *Repository) AddMultiSignature(
	ctx context.Context,
	tenantID, poaID string,
	signerID string,
	signature map[string]interface{},
	threshold int,
) (*PoARecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Lock the row
	query := `
		SELECT multi_signatures, status
		FROM power_of_attorney
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`

	var currentSigsRaw []byte
	var status string
	err = tx.QueryRow(ctx, query, tenantID, poaID).Scan(&currentSigsRaw, &status)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("PoA not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to lock PoA: %w", err)
	}

	// 2. Parse existing signatures
	signatures := make(map[string]interface{})
	if len(currentSigsRaw) > 0 {
		if unmarshalErr := json.Unmarshal(currentSigsRaw, &signatures); unmarshalErr != nil {
			return nil, fmt.Errorf("corrupt multi_signatures data: %w", unmarshalErr)
		}
	}

	// 3. Add new signature
	signatures[signerID] = signature

	// 4. Check threshold
	if len(signatures) >= threshold && status == "pending" {
		status = "active"
	}

	// 5. Update row
	updateQuery := `
		UPDATE power_of_attorney
		SET multi_signatures = $1, status = $2, updated_at = $3
		WHERE tenant_id = $4 AND id = $5
		RETURNING 
			id, tenant_id, poa_name, grantor_id, grantor_name,
			representative_id, representative_name, representative_type,
			scope_type, actions, geographic_regions,
			status, created_at, approved_at, approved_by,
			revoked_at, revoked_by, revocation_reason,
			valid_from, valid_until, conditions, metadata, multi_signatures, updated_at
	`

	newSigsJSON, err := json.Marshal(signatures)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal signatures: %w", err)
	}

	var updatedPoa PoARecord
	var conditionsJSON, metadataJSON, multiSignaturesJSON *[]byte

	err = tx.QueryRow(ctx, updateQuery, newSigsJSON, status, time.Now(), tenantID, poaID).Scan(
		&updatedPoa.ID, &updatedPoa.TenantID, &updatedPoa.PoAName, &updatedPoa.GrantorID, &updatedPoa.GrantorName,
		&updatedPoa.RepresentativeID, &updatedPoa.RepresentativeName, &updatedPoa.RepresentativeType,
		&updatedPoa.ScopeType, &updatedPoa.Actions, &updatedPoa.GeographicRegions,
		&updatedPoa.Status, &updatedPoa.CreatedAt, &updatedPoa.ApprovedAt, &updatedPoa.ApprovedBy,
		&updatedPoa.RevokedAt, &updatedPoa.RevokedBy, &updatedPoa.RevocationReason,
		&updatedPoa.ValidFrom, &updatedPoa.ValidUntil, &conditionsJSON, &metadataJSON, &multiSignaturesJSON, &updatedPoa.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update PoA signatures: %w", err)
	}

	if conditionsJSON != nil && len(*conditionsJSON) > 0 {
		raw := json.RawMessage(*conditionsJSON)
		updatedPoa.Conditions = &raw
	}
	if metadataJSON != nil && len(*metadataJSON) > 0 {
		raw := json.RawMessage(*metadataJSON)
		updatedPoa.Metadata = &raw
	}
	if multiSignaturesJSON != nil && len(*multiSignaturesJSON) > 0 {
		raw := json.RawMessage(*multiSignaturesJSON)
		updatedPoa.MultiSignatures = &raw
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &updatedPoa, nil
}

// GetPoAStats retrieves aggregate statistics for power of attorneys
func (r *Repository) GetPoAStats(ctx context.Context, tenantID string) (*PoAStats, error) {
	stats := &PoAStats{
		ByRepType:       make(map[string]int),
		TopActions:      []ActionCount{},
		GeoDistribution: []GeographicDistribution{},
	}

	// Get status counts
	statusQuery := `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'active') as active,
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'expired') as expired,
			COUNT(*) FILTER (WHERE status = 'revoked') as revoked,
			COUNT(*) as total
		FROM power_of_attorney
		WHERE tenant_id = $1
	`

	err := r.db.QueryRow(ctx, statusQuery, tenantID).Scan(
		&stats.ActivePoAs, &stats.PendingPoAs, &stats.ExpiredPoAs, &stats.RevokedPoAs, &stats.TotalPoAs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}

	// Get counts by representative type
	repTypeQuery := `
		SELECT representative_type, COUNT(*) as count
		FROM power_of_attorney
		WHERE tenant_id = $1
		GROUP BY representative_type
	`

	rows, err := r.db.Query(ctx, repTypeQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rep type counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var repType string
		var count int
		if scanErr := rows.Scan(&repType, &count); scanErr != nil {
			return nil, fmt.Errorf("failed to scan rep type: %w", scanErr)
		}
		stats.ByRepType[repType] = count
	}

	// Get top actions (unnest array and count)
	actionsQuery := `
		SELECT action, COUNT(*) as count
		FROM (
			SELECT UNNEST(actions) as action
			FROM power_of_attorney
			WHERE tenant_id = $1
		) actions_expanded
		GROUP BY action
		ORDER BY count DESC
		LIMIT 10
	`

	rows, err = r.db.Query(ctx, actionsQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get action counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ac ActionCount
		if scanErr := rows.Scan(&ac.Action, &ac.Count); scanErr != nil {
			return nil, fmt.Errorf("failed to scan action count: %w", scanErr)
		}
		stats.TopActions = append(stats.TopActions, ac)
	}

	// Get geographic distribution
	geoQuery := `
		SELECT region, COUNT(*) as count
		FROM (
			SELECT UNNEST(geographic_regions) as region
			FROM power_of_attorney
			WHERE tenant_id = $1
		) regions_expanded
		GROUP BY region
		ORDER BY count DESC
	`

	rows, err = r.db.Query(ctx, geoQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get geo distribution: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var gd GeographicDistribution
		if err := rows.Scan(&gd.Region, &gd.Count); err != nil {
			return nil, fmt.Errorf("failed to scan geo distribution: %w", err)
		}
		stats.GeoDistribution = append(stats.GeoDistribution, gd)
	}

	return stats, nil
}

// CreateTemplate creates a new PoA template
func (r *Repository) CreateTemplate(ctx context.Context, template *PoATemplate) error {
	query := `
		INSERT INTO poa_templates (
			tenant_id, template_name, description, scope_type,
			default_actions, default_duration_days, conditions_schema,
			created_by, is_system_template
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		template.TenantID, template.TemplateName, template.Description, template.ScopeType,
		template.DefaultActions, template.DefaultDurationDays, template.ConditionsSchema,
		template.CreatedBy, template.IsSystemTemplate,
	).Scan(&template.ID, &template.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	return nil
}

// ListTemplates retrieves all PoA templates for a tenant
func (r *Repository) ListTemplates(ctx context.Context, tenantID *string) ([]PoATemplate, error) {
	query := `
		SELECT 
			id, tenant_id, template_name, description, scope_type,
			default_actions, default_duration_days, conditions_schema,
			created_at, created_by, is_system_template
		FROM poa_templates
		WHERE tenant_id = $1 OR tenant_id IS NULL OR is_system_template = true
		ORDER BY is_system_template DESC, template_name
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []PoATemplate
	for rows.Next() {
		var t PoATemplate
		err := rows.Scan(
			&t.ID, &t.TenantID, &t.TemplateName, &t.Description, &t.ScopeType,
			&t.DefaultActions, &t.DefaultDurationDays, &t.ConditionsSchema,
			&t.CreatedAt, &t.CreatedBy, &t.IsSystemTemplate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating templates: %w", err)
	}

	return templates, nil
}
