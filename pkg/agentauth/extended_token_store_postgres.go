package agentauth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// PostgresExtendedTokenStore implements ExtendedTokenStore using PostgreSQL
type PostgresExtendedTokenStore struct {
	db *sql.DB
}

// NewPostgresExtendedTokenStore creates a new PostgreSQL-backed token store
func NewPostgresExtendedTokenStore(db *sql.DB) *PostgresExtendedTokenStore {
	return &PostgresExtendedTokenStore{
		db: db,
	}
}

// NewPostgresExtendedTokenStoreFromDSN creates a new store from a connection string
func NewPostgresExtendedTokenStoreFromDSN(dsn string) (*PostgresExtendedTokenStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	return NewPostgresExtendedTokenStore(db), nil
}

// SaveToken saves a new token or updates an existing one
func (s *PostgresExtendedTokenStore) SaveToken(ctx context.Context, token *ExtendedToken) error {
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}
	if token.AccessToken == "" {
		return fmt.Errorf("access token is required")
	}

	// Extract grant ID from token
	grantID := token.GrantID

	// Marshal complex structures to JSONB
	poaJSON, err := json.Marshal(token.PowerOfAttorney)
	if err != nil {
		return fmt.Errorf("failed to marshal power of attorney: %w", err)
	}

	authChainJSON, err := json.Marshal(token.AuthorizationChain)
	if err != nil {
		return fmt.Errorf("failed to marshal authorization chain: %w", err)
	}

	legalFrameworkJSON, err := json.Marshal(token.LegalFramework)
	if err != nil {
		return fmt.Errorf("failed to marshal legal framework: %w", err)
	}

	verificationProofJSON, err := json.Marshal(token.VerificationProof)
	if err != nil {
		return fmt.Errorf("failed to marshal verification proof: %w", err)
	}

	auditTrailJSON, err := json.Marshal(token.AuditTrail)
	if err != nil {
		return fmt.Errorf("failed to marshal audit trail: %w", err)
	}

	authDetailsJSON, err := json.Marshal(token.AuthorizationDetails)
	if err != nil {
		return fmt.Errorf("failed to marshal authorization details: %w", err)
	}

	// Map ExtendedToken fields to table columns
	// Note: compliance_level is required but not in ExtendedToken struct
	complianceLevel := "AAP-0111-compliant" // default value

	// Insert or update token
	query := `
		INSERT INTO extended_tokens (
			access_token, token_type, expires_in, refresh_token, scope, issued_at,
			power_of_attorney, authorization_chain, legal_framework, verification_proof,
			audit_trail, grant_id, compliance_level, authorization_details,
			created_at, use_count
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14,
			NOW(), 0
		)
		ON CONFLICT (access_token) DO UPDATE SET
			token_type = EXCLUDED.token_type,
			expires_in = EXCLUDED.expires_in,
			refresh_token = EXCLUDED.refresh_token,
			scope = EXCLUDED.scope,
			power_of_attorney = EXCLUDED.power_of_attorney,
			authorization_chain = EXCLUDED.authorization_chain,
			legal_framework = EXCLUDED.legal_framework,
			verification_proof = EXCLUDED.verification_proof,
			audit_trail = EXCLUDED.audit_trail,
			authorization_details = EXCLUDED.authorization_details
	`

	_, err = s.db.ExecContext(ctx, query,
		token.AccessToken,
		token.TokenType,
		token.ExpiresIn,
		token.RefreshToken,
		pq.Array(token.Scope), // Convert Go slice to PostgreSQL array
		token.IssuedAt,
		poaJSON,
		authChainJSON,
		legalFrameworkJSON,
		verificationProofJSON,
		auditTrailJSON,
		grantID,
		complianceLevel,
		authDetailsJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

// GetToken retrieves a token by access token
func (s *PostgresExtendedTokenStore) GetToken(ctx context.Context, accessToken string) (*ExtendedToken, error) {
	query := `
		SELECT 
			access_token, token_type, expires_in, refresh_token, scope, issued_at,
			power_of_attorney, authorization_chain, legal_framework, verification_proof,
			audit_trail, authorization_details,
			created_at, revoked_at, last_used_at, use_count
		FROM extended_tokens
		WHERE access_token = $1 AND revoked_at IS NULL
	`

	var token ExtendedToken
	var metadata TokenMetadata
	var scope []string
	var poaJSON, authChainJSON, legalFrameworkJSON, verificationProofJSON []byte
	var auditTrailJSON, authDetailsJSON []byte
	var revokedAt, lastUsedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, accessToken).Scan(
		&token.AccessToken,
		&token.TokenType,
		&token.ExpiresIn,
		&token.RefreshToken,
		pq.Array(&scope), // Scan PostgreSQL array to Go slice
		&token.IssuedAt,
		&poaJSON,
		&authChainJSON,
		&legalFrameworkJSON,
		&verificationProofJSON,
		&auditTrailJSON,
		&authDetailsJSON,
		&metadata.CreatedAt,
		&revokedAt,
		&lastUsedAt,
		&metadata.UseCount,
	)

	if err == sql.ErrNoRows {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Check if token is expired
	expiresAt := token.IssuedAt.Unix() + token.ExpiresIn
	if time.Now().Unix() > expiresAt {
		return nil, ErrTokenExpired
	}

	// Unmarshal JSONB fields
	if len(poaJSON) > 0 {
		if err := json.Unmarshal(poaJSON, &token.PowerOfAttorney); err != nil {
			return nil, fmt.Errorf("failed to unmarshal power of attorney: %w", err)
		}
	}

	if err := json.Unmarshal(authChainJSON, &token.AuthorizationChain); err != nil {
		return nil, fmt.Errorf("failed to unmarshal authorization chain: %w", err)
	}

	if err := json.Unmarshal(legalFrameworkJSON, &token.LegalFramework); err != nil {
		return nil, fmt.Errorf("failed to unmarshal legal framework: %w", err)
	}

	if err := json.Unmarshal(verificationProofJSON, &token.VerificationProof); err != nil {
		return nil, fmt.Errorf("failed to unmarshal verification proof: %w", err)
	}

	if len(auditTrailJSON) > 0 {
		if err := json.Unmarshal(auditTrailJSON, &token.AuditTrail); err != nil {
			return nil, fmt.Errorf("failed to unmarshal audit trail: %w", err)
		}
	}

	if len(authDetailsJSON) > 0 {
		if err := json.Unmarshal(authDetailsJSON, &token.AuthorizationDetails); err != nil {
			return nil, fmt.Errorf("failed to unmarshal authorization details: %w", err)
		}
	}

	token.Scope = scope

	// Update last used timestamp and use count
	updateQuery := `
		UPDATE extended_tokens 
		SET last_used_at = NOW(), use_count = use_count + 1
		WHERE access_token = $1
	`
	_, _ = s.db.ExecContext(ctx, updateQuery, accessToken)

	return &token, nil
}

// GetTokenByRefreshToken retrieves a token by refresh token
func (s *PostgresExtendedTokenStore) GetTokenByRefreshToken(ctx context.Context, refreshToken string) (*ExtendedToken, error) {
	query := `
		SELECT 
			access_token, token_type, expires_in, refresh_token, scope, issued_at,
			power_of_attorney, authorization_chain, legal_framework, verification_proof,
			audit_trail, authorization_details,
			created_at, revoked_at, last_used_at, use_count
		FROM extended_tokens
		WHERE refresh_token = $1 AND revoked_at IS NULL
	`

	var token ExtendedToken
	var metadata TokenMetadata
	var scope []string
	var poaJSON, authChainJSON, legalFrameworkJSON, verificationProofJSON []byte
	var auditTrailJSON, authDetailsJSON []byte
	var revokedAt, lastUsedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, refreshToken).Scan(
		&token.AccessToken,
		&token.TokenType,
		&token.ExpiresIn,
		&token.RefreshToken,
		pq.Array(&scope), // Scan PostgreSQL array to Go slice
		&token.IssuedAt,
		&poaJSON,
		&authChainJSON,
		&legalFrameworkJSON,
		&verificationProofJSON,
		&auditTrailJSON,
		&authDetailsJSON,
		&metadata.CreatedAt,
		&revokedAt,
		&lastUsedAt,
		&metadata.UseCount,
	)

	if err == sql.ErrNoRows {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token by refresh token: %w", err)
	}

	// Check if token is expired
	expiresAt := token.IssuedAt.Unix() + token.ExpiresIn
	if time.Now().Unix() > expiresAt {
		return nil, ErrTokenExpired
	}

	// Unmarshal JSONB fields
	if len(poaJSON) > 0 {
		if err := json.Unmarshal(poaJSON, &token.PowerOfAttorney); err != nil {
			return nil, fmt.Errorf("failed to unmarshal power of attorney: %w", err)
		}
	}

	if err := json.Unmarshal(authChainJSON, &token.AuthorizationChain); err != nil {
		return nil, fmt.Errorf("failed to unmarshal authorization chain: %w", err)
	}

	if err := json.Unmarshal(legalFrameworkJSON, &token.LegalFramework); err != nil {
		return nil, fmt.Errorf("failed to unmarshal legal framework: %w", err)
	}

	if err := json.Unmarshal(verificationProofJSON, &token.VerificationProof); err != nil {
		return nil, fmt.Errorf("failed to unmarshal verification proof: %w", err)
	}

	if len(auditTrailJSON) > 0 {
		if err := json.Unmarshal(auditTrailJSON, &token.AuditTrail); err != nil {
			return nil, fmt.Errorf("failed to unmarshal audit trail: %w", err)
		}
	}

	if len(authDetailsJSON) > 0 {
		if err := json.Unmarshal(authDetailsJSON, &token.AuthorizationDetails); err != nil {
			return nil, fmt.Errorf("failed to unmarshal authorization details: %w", err)
		}
	}

	token.Scope = scope

	return &token, nil
}

// RevokeToken marks a token as revoked (RFC 7009 compliant - idempotent)
func (s *PostgresExtendedTokenStore) RevokeToken(ctx context.Context, accessToken string) error {
	query := `
		UPDATE extended_tokens 
		SET revoked_at = NOW()
		WHERE access_token = $1 AND revoked_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, accessToken)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// RFC 7009: Token revocation should be idempotent
	// Even if no rows affected, return success
	_, _ = result.RowsAffected()

	return nil
}

// IsRevoked checks if a token has been revoked
func (s *PostgresExtendedTokenStore) IsRevoked(ctx context.Context, accessToken string) (bool, error) {
	query := `
		SELECT revoked_at IS NOT NULL
		FROM extended_tokens
		WHERE access_token = $1
	`

	var isRevoked bool
	err := s.db.QueryRowContext(ctx, query, accessToken).Scan(&isRevoked)

	if err == sql.ErrNoRows {
		// Token doesn't exist - treat as revoked
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check revocation status: %w", err)
	}

	return isRevoked, nil
}

// DeleteExpiredTokens removes all expired tokens from the store
func (s *PostgresExtendedTokenStore) DeleteExpiredTokens(ctx context.Context) (int, error) {
	// Use the computed expires_at column for efficient cleanup
	// Implementing grace periods:
	// 1. Expired tokens: Delete 1 hour after expiration (to handle clock skew)
	// 2. Revoked tokens: Delete 24 hours after revocation (to maintain audit trail)
	query := `
		DELETE FROM extended_tokens
		WHERE 
			(expires_at < NOW() - INTERVAL '1 hour')
			OR
			(revoked_at < NOW() - INTERVAL '24 hours')
	`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return int(count), nil
}

// ListTokensByClient retrieves all tokens for a specific client
func (s *PostgresExtendedTokenStore) ListTokensByClient(ctx context.Context, clientID string) ([]*ExtendedToken, error) {
	query := `
		SELECT 
			access_token, token_type, expires_in, refresh_token, scope, issued_at,
			power_of_attorney, authorization_chain, legal_framework, verification_proof,
			audit_trail, authorization_details
		FROM extended_tokens
		WHERE client_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens by client: %w", err)
	}
	defer rows.Close()

	var tokens []*ExtendedToken

	for rows.Next() {
		var token ExtendedToken
		var scope []string
		var poaJSON, authChainJSON, legalFrameworkJSON, verificationProofJSON []byte
		var auditTrailJSON, authDetailsJSON []byte

		err := rows.Scan(
			&token.AccessToken,
			&token.TokenType,
			&token.ExpiresIn,
			&token.RefreshToken,
			pq.Array(&scope), // Scan PostgreSQL array to Go slice
			&token.IssuedAt,
			&poaJSON,
			&authChainJSON,
			&legalFrameworkJSON,
			&verificationProofJSON,
			&auditTrailJSON,
			&authDetailsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token row: %w", err)
		}

		// Unmarshal JSONB fields
		if len(poaJSON) > 0 {
			if err := json.Unmarshal(poaJSON, &token.PowerOfAttorney); err != nil {
				return nil, fmt.Errorf("failed to unmarshal power of attorney: %w", err)
			}
		}

		if err := json.Unmarshal(authChainJSON, &token.AuthorizationChain); err != nil {
			return nil, fmt.Errorf("failed to unmarshal authorization chain: %w", err)
		}

		if err := json.Unmarshal(legalFrameworkJSON, &token.LegalFramework); err != nil {
			return nil, fmt.Errorf("failed to unmarshal legal framework: %w", err)
		}

		if err := json.Unmarshal(verificationProofJSON, &token.VerificationProof); err != nil {
			return nil, fmt.Errorf("failed to unmarshal verification proof: %w", err)
		}

		if len(auditTrailJSON) > 0 {
			if err := json.Unmarshal(auditTrailJSON, &token.AuditTrail); err != nil {
				return nil, fmt.Errorf("failed to unmarshal audit trail: %w", err)
			}
		}

		if len(authDetailsJSON) > 0 {
			if err := json.Unmarshal(authDetailsJSON, &token.AuthorizationDetails); err != nil {
				return nil, fmt.Errorf("failed to unmarshal authorization details: %w", err)
			}
		}

		token.Scope = scope
		tokens = append(tokens, &token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating token rows: %w", err)
	}

	return tokens, nil
}

// ListTokensByResourceOwner returns all active tokens for a specific resource owner
func (s *PostgresExtendedTokenStore) ListTokensByResourceOwner(ctx context.Context, ownerID string) ([]*ExtendedToken, error) {
	// Query for active (non-revoked, non-expired) tokens for the resource owner
	// We need to extract owner_id from the authorization_chain JSONB field
	query := `
		SELECT 
			access_token, token_type, expires_in, refresh_token, scope, issued_at,
			power_of_attorney, authorization_chain, legal_framework, verification_proof,
			audit_trail, grant_id, compliance_level, authorization_details
		FROM extended_tokens
		WHERE 
			revoked_at IS NULL
			AND (issued_at + (expires_in * interval '1 second') > NOW()
			AND authorization_chain->>'resource_owner_id' = $1
		ORDER BY issued_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tokens by resource owner: %w", err)
	}
	defer rows.Close()

	var tokens []*ExtendedToken
	for rows.Next() {
		var token ExtendedToken
		var scope pq.StringArray
		var poaJSON, authChainJSON, legalFrameworkJSON, verificationProofJSON, auditTrailJSON, authDetailsJSON []byte

		err := rows.Scan(
			&token.AccessToken,
			&token.TokenType,
			&token.ExpiresIn,
			&token.RefreshToken,
			&scope,
			&token.IssuedAt,
			&poaJSON,
			&authChainJSON,
			&legalFrameworkJSON,
			&verificationProofJSON,
			&auditTrailJSON,
			&token.GrantID,
			&token.ComplianceLevel,
			&authDetailsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token row: %w", err)
		}

		// Unmarshal JSONB fields
		if len(poaJSON) > 0 {
			if err := json.Unmarshal(poaJSON, &token.PowerOfAttorney); err != nil {
				return nil, fmt.Errorf("failed to unmarshal power of attorney: %w", err)
			}
		}

		if err := json.Unmarshal(authChainJSON, &token.AuthorizationChain); err != nil {
			return nil, fmt.Errorf("failed to unmarshal authorization chain: %w", err)
		}

		if err := json.Unmarshal(legalFrameworkJSON, &token.LegalFramework); err != nil {
			return nil, fmt.Errorf("failed to unmarshal legal framework: %w", err)
		}

		if err := json.Unmarshal(verificationProofJSON, &token.VerificationProof); err != nil {
			return nil, fmt.Errorf("failed to unmarshal verification proof: %w", err)
		}

		if len(auditTrailJSON) > 0 {
			if err := json.Unmarshal(auditTrailJSON, &token.AuditTrail); err != nil {
				return nil, fmt.Errorf("failed to unmarshal audit trail: %w", err)
			}
		}

		if len(authDetailsJSON) > 0 {
			if err := json.Unmarshal(authDetailsJSON, &token.AuthorizationDetails); err != nil {
				return nil, fmt.Errorf("failed to unmarshal authorization details: %w", err)
			}
		}

		token.Scope = scope
		tokens = append(tokens, &token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating token rows: %w", err)
	}

	return tokens, nil
}

// RevokeTokenWithReason marks a token as revoked with a specific reason
func (s *PostgresExtendedTokenStore) RevokeTokenWithReason(ctx context.Context, accessToken string, reason string) error {
	// First, revoke the token
	if err := s.RevokeToken(ctx, accessToken); err != nil {
		return err
	}

	// Then, add an audit entry to the token's audit trail
	query := `
		UPDATE extended_tokens
		SET audit_trail = COALESCE(audit_trail, '[]'::jsonb) || $2::jsonb
		WHERE access_token = $1
	`

	auditEntry := AuditEntry{
		Timestamp: time.Now(),
		Action:    "token_revoked",
		Actor:     "resource_owner", // Could be enhanced to get from context
		Result:    "success",
		Details: map[string]interface{}{
			"reason":       reason,
			"access_token": accessToken,
		},
	}

	auditJSON, err := json.Marshal(auditEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query, accessToken, auditJSON)
	if err != nil {
		return fmt.Errorf("failed to add audit entry: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *PostgresExtendedTokenStore) Close() error {
	return s.db.Close()
}
