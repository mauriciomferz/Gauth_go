package oidc

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresStorage implements StorageBackend using PostgreSQL.
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage creates a new PostgreSQL storage backend.
// connectionString format: "postgres://user:password@host:port/database?sslmode=disable"
func NewPostgresStorage(connectionString string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := &PostgresStorage{db: db}

	// Initialize schema
	if err := storage.initSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema creates the required database tables if they don't exist.
func (s *PostgresStorage) initSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS refresh_tokens (
		refresh_token VARCHAR(512) PRIMARY KEY,
		provider_id VARCHAR(255) NOT NULL,
		subject VARCHAR(255) NOT NULL,
		audience VARCHAR(255),
		scopes TEXT, -- JSON array
		issued_at TIMESTAMP NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		last_used TIMESTAMP NOT NULL,
		use_count INTEGER DEFAULT 0,
		revoked BOOLEAN DEFAULT FALSE,
		email VARCHAR(255),
		email_verified BOOLEAN DEFAULT FALSE,
		name VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_refresh_tokens_subject (subject),
		INDEX idx_refresh_tokens_provider (provider_id),
		INDEX idx_refresh_tokens_expires_at (expires_at)
	);

	CREATE TABLE IF NOT EXISTS revoked_tokens (
		token_id VARCHAR(512) PRIMARY KEY,
		revoked_at TIMESTAMP NOT NULL,
		reason TEXT,
		revoked_by VARCHAR(255),
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_revoked_tokens_expires_at (expires_at)
	);

	CREATE TABLE IF NOT EXISTS device_codes (
		device_code VARCHAR(512) PRIMARY KEY,
		user_code VARCHAR(64) UNIQUE NOT NULL,
		client_id VARCHAR(255) NOT NULL,
		scope TEXT,
		issued_at TIMESTAMP NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		last_polled TIMESTAMP,
		poll_count INTEGER DEFAULT 0,
		status VARCHAR(50) DEFAULT 'pending',
		authorized_by VARCHAR(255),
		authorized_at TIMESTAMP,
		access_token TEXT,
		refresh_token TEXT,
		id_token TEXT,
		token_issued_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_device_codes_user_code (user_code),
		INDEX idx_device_codes_expires_at (expires_at)
	);

	CREATE TABLE IF NOT EXISTS par_requests (
		request_uri VARCHAR(512) PRIMARY KEY,
		client_id VARCHAR(255) NOT NULL,
		response_type VARCHAR(100),
		redirect_uri TEXT,
		scope TEXT,
		state VARCHAR(255),
		nonce VARCHAR(255),
		code_challenge VARCHAR(255),
		code_challenge_method VARCHAR(10),
		response_mode VARCHAR(50),
		display VARCHAR(50),
		prompt VARCHAR(100),
		max_age INTEGER,
		ui_locales VARCHAR(255),
		id_token_hint TEXT,
		login_hint VARCHAR(255),
		acr_values VARCHAR(255),
		custom_parameters TEXT, -- JSON object
		expires_at TIMESTAMP NOT NULL,
		used BOOLEAN DEFAULT FALSE,
		used_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_par_requests_expires_at (expires_at),
		INDEX idx_par_requests_client_id (client_id)
	);
	`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// RefreshToken operations

func (s *PostgresStorage) StoreRefreshToken(ctx context.Context, token *RefreshTokenEntry) error {
	scopesJSON, err := json.Marshal(token.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	query := `
		INSERT INTO refresh_tokens 
		(refresh_token, provider_id, subject, audience, scopes, issued_at, expires_at, 
		 last_used, use_count, revoked, email, email_verified, name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (refresh_token) 
		DO UPDATE SET
			last_used = $8,
			use_count = $9,
			revoked = $10
	`

	_, err = s.db.ExecContext(ctx, query,
		token.RefreshToken, token.ProviderID, token.Subject, token.Audience,
		scopesJSON, token.IssuedAt, token.ExpiresAt, token.LastUsed,
		token.UseCount, token.Revoked, token.Email, token.EmailVerified, token.Name,
	)

	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

func (s *PostgresStorage) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenEntry, error) {
	query := `
		SELECT refresh_token, provider_id, subject, audience, scopes, issued_at, 
		       expires_at, last_used, use_count, revoked, email, email_verified, name
		FROM refresh_tokens
		WHERE refresh_token = $1
	`

	var entry RefreshTokenEntry
	var scopesJSON []byte

	err := s.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&entry.RefreshToken, &entry.ProviderID, &entry.Subject, &entry.Audience,
		&scopesJSON, &entry.IssuedAt, &entry.ExpiresAt, &entry.LastUsed,
		&entry.UseCount, &entry.Revoked, &entry.Email, &entry.EmailVerified, &entry.Name,
	)

	if err == sql.ErrNoRows {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	if err := json.Unmarshal(scopesJSON, &entry.Scopes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
	}

	return &entry, nil
}

func (s *PostgresStorage) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	query := "DELETE FROM refresh_tokens WHERE refresh_token = $1"
	_, err := s.db.ExecContext(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}

func (s *PostgresStorage) ListRefreshTokensByUser(ctx context.Context, userID string) ([]*RefreshTokenEntry, error) {
	query := `
		SELECT refresh_token, provider_id, subject, audience, scopes, issued_at, 
		       expires_at, last_used, use_count, revoked, email, email_verified, name
		FROM refresh_tokens
		WHERE subject = $1
		ORDER BY issued_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list refresh tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*RefreshTokenEntry
	for rows.Next() {
		var entry RefreshTokenEntry
		var scopesJSON []byte

		err := rows.Scan(
			&entry.RefreshToken, &entry.ProviderID, &entry.Subject, &entry.Audience,
			&scopesJSON, &entry.IssuedAt, &entry.ExpiresAt, &entry.LastUsed,
			&entry.UseCount, &entry.Revoked, &entry.Email, &entry.EmailVerified, &entry.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan refresh token: %w", err)
		}

		if err := json.Unmarshal(scopesJSON, &entry.Scopes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
		}

		tokens = append(tokens, &entry)
	}

	return tokens, rows.Err()
}

func (s *PostgresStorage) ListRefreshTokensByClient(ctx context.Context, clientID string) ([]*RefreshTokenEntry, error) {
	query := `
		SELECT refresh_token, provider_id, subject, audience, scopes, issued_at, 
		       expires_at, last_used, use_count, revoked, email, email_verified, name
		FROM refresh_tokens
		WHERE provider_id = $1
		ORDER BY issued_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list refresh tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*RefreshTokenEntry
	for rows.Next() {
		var entry RefreshTokenEntry
		var scopesJSON []byte

		err := rows.Scan(
			&entry.RefreshToken, &entry.ProviderID, &entry.Subject, &entry.Audience,
			&scopesJSON, &entry.IssuedAt, &entry.ExpiresAt, &entry.LastUsed,
			&entry.UseCount, &entry.Revoked, &entry.Email, &entry.EmailVerified, &entry.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan refresh token: %w", err)
		}

		if err := json.Unmarshal(scopesJSON, &entry.Scopes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
		}

		tokens = append(tokens, &entry)
	}

	return tokens, rows.Err()
}

func (s *PostgresStorage) CleanupExpiredRefreshTokens(ctx context.Context) (int, error) {
	query := "DELETE FROM refresh_tokens WHERE expires_at < $1"
	result, err := s.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired refresh tokens: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(count), nil
}

// Revoked token operations

func (s *PostgresStorage) StoreRevokedToken(ctx context.Context, entry *RevokedTokenEntry) error {
	query := `
		INSERT INTO revoked_tokens (token_id, revoked_at, reason, revoked_by, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (token_id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query,
		entry.TokenID, entry.RevokedAt, entry.Reason, entry.RevokedBy, entry.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store revoked token: %w", err)
	}

	return nil
}

func (s *PostgresStorage) IsTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM revoked_tokens WHERE token_id = $1)"

	var exists bool
	err := s.db.QueryRowContext(ctx, query, tokenHash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check token revocation: %w", err)
	}

	return exists, nil
}

func (s *PostgresStorage) CleanupExpiredRevocations(ctx context.Context) (int, error) {
	query := "DELETE FROM revoked_tokens WHERE expires_at < $1"
	result, err := s.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired revocations: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(count), nil
}

// Device code operations

func (s *PostgresStorage) StoreDeviceCode(ctx context.Context, entry *DeviceCodeEntry) error {
	scopeJSON, err := json.Marshal(entry.Scope)
	if err != nil {
		return fmt.Errorf("failed to marshal scope: %w", err)
	}

	query := `
		INSERT INTO device_codes 
		(device_code, user_code, client_id, scope, issued_at, expires_at, last_polled,
		 poll_count, status, authorized_by, authorized_at, access_token, refresh_token, 
		 id_token, token_issued_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (device_code)
		DO UPDATE SET
			last_polled = $7,
			poll_count = $8,
			status = $9,
			authorized_by = $10,
			authorized_at = $11,
			access_token = $12,
			refresh_token = $13,
			id_token = $14,
			token_issued_at = $15
	`

	_, err = s.db.ExecContext(ctx, query,
		entry.DeviceCode, entry.UserCode, entry.ClientID, scopeJSON,
		entry.IssuedAt, entry.ExpiresAt, entry.LastPolled, entry.PollCount,
		entry.Status, entry.AuthorizedBy, entry.AuthorizedAt,
		entry.AccessToken, entry.RefreshToken, entry.IDToken, entry.TokenIssuedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store device code: %w", err)
	}

	return nil
}

func (s *PostgresStorage) GetDeviceCodeByDeviceCode(ctx context.Context, deviceCode string) (*DeviceCodeEntry, error) {
	query := `
		SELECT device_code, user_code, client_id, scope, issued_at, expires_at, 
		       last_polled, poll_count, status, authorized_by, authorized_at,
		       access_token, refresh_token, id_token, token_issued_at
		FROM device_codes
		WHERE device_code = $1
	`

	var entry DeviceCodeEntry
	var scopeJSON []byte
	var lastPolled, authorizedAt, tokenIssuedAt sql.NullTime
	var authorizedBy sql.NullString

	err := s.db.QueryRowContext(ctx, query, deviceCode).Scan(
		&entry.DeviceCode, &entry.UserCode, &entry.ClientID, &scopeJSON,
		&entry.IssuedAt, &entry.ExpiresAt, &lastPolled, &entry.PollCount,
		&entry.Status, &authorizedBy, &authorizedAt,
		&entry.AccessToken, &entry.RefreshToken, &entry.IDToken, &tokenIssuedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrDeviceCodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device code: %w", err)
	}

	if err := json.Unmarshal(scopeJSON, &entry.Scope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scope: %w", err)
	}

	if lastPolled.Valid {
		entry.LastPolled = lastPolled.Time
	}
	if authorizedBy.Valid {
		entry.AuthorizedBy = authorizedBy.String
	}
	if authorizedAt.Valid {
		entry.AuthorizedAt = authorizedAt.Time
	}
	if tokenIssuedAt.Valid {
		entry.TokenIssuedAt = tokenIssuedAt.Time
	}

	return &entry, nil
}

func (s *PostgresStorage) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*DeviceCodeEntry, error) {
	query := `
		SELECT device_code, user_code, client_id, scope, issued_at, expires_at, 
		       last_polled, poll_count, status, authorized_by, authorized_at,
		       access_token, refresh_token, id_token, token_issued_at
		FROM device_codes
		WHERE user_code = $1
	`

	var entry DeviceCodeEntry
	var scopeJSON []byte
	var lastPolled, authorizedAt, tokenIssuedAt sql.NullTime
	var authorizedBy sql.NullString

	err := s.db.QueryRowContext(ctx, query, userCode).Scan(
		&entry.DeviceCode, &entry.UserCode, &entry.ClientID, &scopeJSON,
		&entry.IssuedAt, &entry.ExpiresAt, &lastPolled, &entry.PollCount,
		&entry.Status, &authorizedBy, &authorizedAt,
		&entry.AccessToken, &entry.RefreshToken, &entry.IDToken, &tokenIssuedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserCodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device code by user code: %w", err)
	}

	if err := json.Unmarshal(scopeJSON, &entry.Scope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scope: %w", err)
	}

	if lastPolled.Valid {
		entry.LastPolled = lastPolled.Time
	}
	if authorizedBy.Valid {
		entry.AuthorizedBy = authorizedBy.String
	}
	if authorizedAt.Valid {
		entry.AuthorizedAt = authorizedAt.Time
	}
	if tokenIssuedAt.Valid {
		entry.TokenIssuedAt = tokenIssuedAt.Time
	}

	return &entry, nil
}

func (s *PostgresStorage) UpdateDeviceCodeStatus(ctx context.Context, deviceCode string, entry *DeviceCodeEntry) error {
	query := `
		UPDATE device_codes
		SET status = $2, authorized_by = $3, authorized_at = $4, access_token = $5, 
		    refresh_token = $6, id_token = $7, token_issued_at = $8, last_polled = $9, poll_count = $10
		WHERE device_code = $1
	`

	result, err := s.db.ExecContext(ctx, query,
		deviceCode, entry.Status, entry.AuthorizedBy, entry.AuthorizedAt,
		entry.AccessToken, entry.RefreshToken, entry.IDToken, entry.TokenIssuedAt,
		entry.LastPolled, entry.PollCount,
	)

	if err != nil {
		return fmt.Errorf("failed to update device code status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrDeviceCodeNotFound
	}

	return nil
}

func (s *PostgresStorage) DeleteDeviceCode(ctx context.Context, deviceCode string) error {
	query := "DELETE FROM device_codes WHERE device_code = $1"
	_, err := s.db.ExecContext(ctx, query, deviceCode)
	if err != nil {
		return fmt.Errorf("failed to delete device code: %w", err)
	}
	return nil
}

func (s *PostgresStorage) CleanupExpiredDeviceCodes(ctx context.Context) (int, error) {
	query := "DELETE FROM device_codes WHERE expires_at < $1"
	result, err := s.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired device codes: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(count), nil
}

// PAR request URI operations

func (s *PostgresStorage) StorePARRequest(ctx context.Context, requestURI string, entry *RequestURIEntry) error {
	customParamsJSON, err := json.Marshal(entry.Request.CustomParameters)
	if err != nil {
		return fmt.Errorf("failed to marshal custom parameters: %w", err)
	}

	query := `
		INSERT INTO par_requests 
		(request_uri, client_id, response_type, redirect_uri, scope, state, nonce,
		 code_challenge, code_challenge_method, response_mode, display, prompt,
		 max_age, ui_locales, id_token_hint, login_hint, acr_values, custom_parameters,
		 expires_at, used, used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		ON CONFLICT (request_uri)
		DO UPDATE SET used = $20, used_at = $21
	`

	_, err = s.db.ExecContext(ctx, query,
		requestURI, entry.Request.ClientID, entry.Request.ResponseType,
		entry.Request.RedirectURI, entry.Request.Scope, entry.Request.State,
		entry.Request.Nonce, entry.Request.CodeChallenge, entry.Request.CodeChallengeMethod,
		entry.Request.ResponseMode, entry.Request.Display, entry.Request.Prompt,
		entry.Request.MaxAge, entry.Request.UILocales, entry.Request.IDTokenHint,
		entry.Request.LoginHint, entry.Request.ACRValues, customParamsJSON,
		entry.ExpiresAt, entry.Used, entry.UsedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store PAR request: %w", err)
	}

	return nil
}

func (s *PostgresStorage) GetPARRequest(ctx context.Context, requestURI string) (*RequestURIEntry, error) {
	query := `
		SELECT client_id, response_type, redirect_uri, scope, state, nonce,
		       code_challenge, code_challenge_method, response_mode, display, prompt,
		       max_age, ui_locales, id_token_hint, login_hint, acr_values, custom_parameters,
		       expires_at, used, used_at, created_at
		FROM par_requests
		WHERE request_uri = $1
	`

	var entry RequestURIEntry
	entry.Request = &PushedAuthorizationRequest{}
	var customParamsJSON []byte
	var usedAt, createdAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, requestURI).Scan(
		&entry.Request.ClientID, &entry.Request.ResponseType, &entry.Request.RedirectURI,
		&entry.Request.Scope, &entry.Request.State, &entry.Request.Nonce,
		&entry.Request.CodeChallenge, &entry.Request.CodeChallengeMethod,
		&entry.Request.ResponseMode, &entry.Request.Display, &entry.Request.Prompt,
		&entry.Request.MaxAge, &entry.Request.UILocales, &entry.Request.IDTokenHint,
		&entry.Request.LoginHint, &entry.Request.ACRValues, &customParamsJSON,
		&entry.ExpiresAt, &entry.Used, &usedAt, &createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrRequestURINotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get PAR request: %w", err)
	}

	if usedAt.Valid {
		entry.UsedAt = usedAt.Time
	}
	if createdAt.Valid {
		entry.CreatedAt = createdAt.Time
	}

	if err := json.Unmarshal(customParamsJSON, &entry.Request.CustomParameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal custom parameters: %w", err)
	}

	return &entry, nil
}

func (s *PostgresStorage) DeletePARRequest(ctx context.Context, requestURI string) error {
	query := "DELETE FROM par_requests WHERE request_uri = $1"
	_, err := s.db.ExecContext(ctx, query, requestURI)
	if err != nil {
		return fmt.Errorf("failed to delete PAR request: %w", err)
	}
	return nil
}

func (s *PostgresStorage) MarkPARRequestUsed(ctx context.Context, requestURI string) error {
	query := "UPDATE par_requests SET used = true, used_at = $2 WHERE request_uri = $1"
	result, err := s.db.ExecContext(ctx, query, requestURI, time.Now())
	if err != nil {
		return fmt.Errorf("failed to mark PAR request as used: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrRequestURINotFound
	}

	return nil
}

func (s *PostgresStorage) CleanupExpiredPARRequests(ctx context.Context) (int, error) {
	query := "DELETE FROM par_requests WHERE expires_at < $1"
	result, err := s.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired PAR requests: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(count), nil
}

// Health check

func (s *PostgresStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}
