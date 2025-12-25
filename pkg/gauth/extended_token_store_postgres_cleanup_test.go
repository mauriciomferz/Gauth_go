package gauth

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestPostgresExtendedTokenStore_Cleanup verifies the cleanup logic with grace periods
// This test requires a running Postgres instance and GAUTH_POSTGRES_DSN env var
func TestPostgresExtendedTokenStore_Cleanup(t *testing.T) {
	dsn := os.Getenv("GAUTH_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("Skipping Postgres cleanup test: GAUTH_POSTGRES_DSN not set")
	}

	// Setup DB connection
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	// Verify connection
	err = db.Ping()
	require.NoError(t, err)

	// Clean up previous runs
	_, err = db.Exec("DELETE FROM extended_tokens WHERE compliance_level = 'test-cleanup'")
	require.NoError(t, err)

	store := NewPostgresExtendedTokenStore(db)
	ctx := context.Background()

	// Helpers to create token with specific timings
	createToken := func(accessToken string, expiresAt, revokedAt *time.Time) {
		query := `
			INSERT INTO extended_tokens (
				access_token, token_type, expires_in, refresh_token, scope, 
				issued_at, created_at, revoked_at, compliance_level, use_count
			) VALUES (
				$1, 'Bearer', 3600, 'refresh_' || $1, '{}',
				$2, $2, $3, 'test-cleanup', 0
			)`

		issuedAt := time.Now()
		if expiresAt != nil {
			issuedAt = expiresAt.Add(-1 * time.Hour)
		}

		_, err := db.Exec(query, accessToken, issuedAt, revokedAt)
		require.NoError(t, err)

		// Manually override expires_at if needed (since it's usually computed generated column or calculated)
		// Assuming extended_tokens uses a computed column based on issued_at + expires_in,
		// we might need to adjust issued_at to simulate expiration.
		// If expires_at is a real column, we set it. If it is computed, we rely on issued_at.
		// Based on previous file view, logic usually relies on `expires_at < NOW()`.
		// Let's force update timestamps to ensure precise control.

		if expiresAt != nil {
			// If we want it to be expired at 'expiresAt', issued_at should be expiresAt - expires_in
			// The INSERT above sets issued_at.
			// Let's manually set timestamps to be sure about "valid", "expired grace", "expired old"
		}
	}

	// We need 5 tokens:
	now := time.Now()

	// 1. Active Token (Valid)
	// Issued now, expires in 1h.
	createToken("cleanup_test_valid", nil, nil) // Default logic makes it valid

	// 2. Just Expired (Within Grace Period) -> SHOULD NOT BE DELETED
	// Expired 30 mins ago. (Grace is 1h)
	// To be expired 30 mins ago, issued_at = now - 1h 30m (assuming 1h lifetime)
	expiredGraceTime := now.Add(-30 * time.Minute)
	issuedGrace := expiredGraceTime.Add(-1 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO extended_tokens (access_token, token_type, expires_in, scope, issued_at, compliance_level, use_count)
		VALUES ($1, 'Bearer', 3600, '{}', $2, 'test-cleanup', 0)
	`, "cleanup_test_expired_grace", issuedGrace)
	require.NoError(t, err)

	// 3. Old Expired (Outside Grace Period) -> SHOULD BE DELETED
	// Expired 2 hours ago.
	expiredOldTime := now.Add(-2 * time.Hour)
	issuedOld := expiredOldTime.Add(-1 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO extended_tokens (access_token, token_type, expires_in, scope, issued_at, compliance_level, use_count)
		VALUES ($1, 'Bearer', 3600, '{}', $2, 'test-cleanup', 0)
	`, "cleanup_test_expired_old", issuedOld)
	require.NoError(t, err)

	// 4. Just Revoked (Within Audit Grace Period) -> SHOULD NOT BE DELETED
	// Revoked 23 hours ago. (Grace is 24h)
	revokedGraceTime := now.Add(-23 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO extended_tokens (access_token, token_type, expires_in, scope, issued_at, revoked_at, compliance_level, use_count)
		VALUES ($1, 'Bearer', 3600, '{}', $2, $3, 'test-cleanup', 0)
	`, "cleanup_test_revoked_grace", now, revokedGraceTime)
	require.NoError(t, err)

	// 5. Old Revoked (Outside Audit Grace Period) -> SHOULD BE DELETED
	// Revoked 25 hours ago.
	revokedOldTime := now.Add(-25 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO extended_tokens (access_token, token_type, expires_in, scope, issued_at, revoked_at, compliance_level, use_count)
		VALUES ($1, 'Bearer', 3600, '{}', $2, $3, 'test-cleanup', 0)
	`, "cleanup_test_revoked_old", now, revokedOldTime)
	require.NoError(t, err)

	// Run Cleanup
	count, err := store.DeleteExpiredTokens(ctx)
	require.NoError(t, err)

	// Expect 2 deletions (expired_old, revoked_old)
	if count != 2 {
		// Let's diagnose what happened
		var remaining []string
		rows, _ := db.Query("SELECT access_token FROM extended_tokens WHERE compliance_level = 'test-cleanup'")
		defer rows.Close()
		for rows.Next() {
			var s string
			rows.Scan(&s)
			remaining = append(remaining, s)
		}
		t.Errorf("Expected 2 tokens cleaned, got %d. Remaining: %v", count, remaining)
	}

	// Verify specific tokens
	verifyExists := func(token string, shouldExist bool) {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM extended_tokens WHERE access_token = $1)", token).Scan(&exists)
		require.NoError(t, err)
		if exists != shouldExist {
			t.Errorf("Token %s existence check failed. Expected: %v, Got: %v", token, shouldExist, exists)
		}
	}

	verifyExists("cleanup_test_valid", true)
	verifyExists("cleanup_test_expired_grace", true)
	verifyExists("cleanup_test_expired_old", false)
	verifyExists("cleanup_test_revoked_grace", true)
	verifyExists("cleanup_test_revoked_old", false)
}
