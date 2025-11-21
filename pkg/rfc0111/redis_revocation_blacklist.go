package rfc0111

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RevocationBlacklistStore provides real-time revocation status checking to prevent "zombie tokens".
//
// Security Context: Addresses Medium Vulnerability - Revocation Latency (The "Zombie Token" Window)
//
// Problem:
// Current implementation checks PoA status during token issuance (VerifyToken at T=0), but:
//   1. Issues short-lived JWT/PASETO access token (valid for 1 hour)
//   2. If PoA is revoked at T=5min, the access token remains valid until T=60min
//   3. API middleware validates JWT signature (stateless), NOT underlying PoA status
//
// Attack Scenario:
//   T=0:    Agent exchanges PoA for Access Token (status check passes)
//   T=5min: Principal revokes PoA in database
//   T=10min: Agent uses Access Token to access protected resources
//   Result: Unauthorized access for 55 minutes after revocation
//
// Solution:
// Maintain a Redis-backed blacklist of revoked PoA IDs with TTL matching longest token lifetime.
// API middleware MUST check blacklist on every request: !isRevoked(token.poa_id)
//
// Architecture:
//   - Revocation event: Service adds PoA ID to blacklist with TTL = max_token_lifetime
//   - Token validation: Middleware checks blacklist before processing request
//   - Performance: O(1) Redis GET operation (~1ms latency)
//   - Memory: Bounded by (revocations_per_hour × max_token_lifetime_hours)
type RevocationBlacklistStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewRevocationBlacklistStore constructs a new Redis-backed revocation blacklist.
//
// Parameters:
//   - client: Redis client (must be connected)
//   - prefix: namespace prefix (e.g., "gauth:revoked")
//   - ttl: how long to keep revocation entries (should be >= max token lifetime, e.g., 24h)
//
// Returns error if Redis connection fails.
func NewRevocationBlacklistStore(client *redis.Client, prefix string, ttl time.Duration) (*RevocationBlacklistStore, error) {
	if client == nil {
		return nil, fmt.Errorf("nil redis client")
	}
	if prefix == "" {
		prefix = "gauth:revoked"
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour // Default: keep revocations for 24 hours
	}

	// Validate connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RevocationBlacklistStore{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}, nil
}

// key formats the full Redis key for a revoked PoA.
// Format: <prefix>:poa:<poa_id>
// Example: "gauth:revoked:poa:poa-abc123"
func (r *RevocationBlacklistStore) key(poaID string) string {
	return fmt.Sprintf("%s:poa:%s", r.prefix, poaID)
}

// AddRevocation marks a PoA as revoked in the blacklist.
// This should be called immediately when a PoA is revoked via InitiateRevocation or ApproveRevocation.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//   - poaID: ID of the revoked PoA
//   - revokedAt: timestamp of revocation (stored as metadata)
//   - reason: revocation reason (stored as metadata)
//
// Returns error if Redis operation fails.
func (r *RevocationBlacklistStore) AddRevocation(ctx context.Context, poaID string, revokedAt time.Time, reason string) error {
	if poaID == "" {
		return fmt.Errorf("empty poaID")
	}

	// Store revocation metadata as JSON for audit trail
	metadata := map[string]interface{}{
		"revoked_at": revokedAt.UTC().Format(time.RFC3339Nano),
		"reason":     reason,
		"added_at":   time.Now().UTC().Format(time.RFC3339Nano),
	}

	// Use SET with EX (expiration) in seconds
	err := r.client.Set(ctx, r.key(poaID), fmt.Sprintf("%v", metadata), r.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to add revocation for %s: %w", poaID, err)
	}

	return nil
}

// IsRevoked checks if a PoA is in the revocation blacklist.
// This is the critical security check that MUST be called on every API request.
//
// Parameters:
//   - ctx: context for cancellation/timeout (should have short timeout, e.g., 100ms)
//   - poaID: ID of the PoA to check
//
// Returns:
//   - revoked (bool): true if PoA is revoked, false otherwise
//   - error: non-nil if Redis operation fails
//
// Performance: O(1) Redis GET operation, typically <1ms
// Failure Mode: If Redis is unavailable, caller should decide:
//   - Fail-open: allow request (risk unauthorized access)
//   - Fail-closed: deny request (risk false positive denials)
func (r *RevocationBlacklistStore) IsRevoked(ctx context.Context, poaID string) (revoked bool, err error) {
	if poaID == "" {
		return false, fmt.Errorf("empty poaID")
	}

	// Check if key exists (GET returns error redis.Nil if not found)
	_, err = r.client.Get(ctx, r.key(poaID)).Result()
	if err != nil {
		if err == redis.Nil {
			// Key does not exist = not revoked
			return false, nil
		}
		// Redis error (network, timeout, etc.)
		return false, fmt.Errorf("failed to check revocation for %s: %w", poaID, err)
	}

	// Key exists = revoked
	return true, nil
}

// RemoveRevocation removes a PoA from the blacklist.
// This should RARELY be used - typically only for testing or correcting errors.
// Production use case: PoA was incorrectly revoked and needs to be reinstated.
func (r *RevocationBlacklistStore) RemoveRevocation(ctx context.Context, poaID string) error {
	if poaID == "" {
		return fmt.Errorf("empty poaID")
	}

	err := r.client.Del(ctx, r.key(poaID)).Err()
	if err != nil {
		return fmt.Errorf("failed to remove revocation for %s: %w", poaID, err)
	}

	return nil
}

// GetRevocationMetadata retrieves the metadata for a revoked PoA.
// Useful for audit logging and debugging.
func (r *RevocationBlacklistStore) GetRevocationMetadata(ctx context.Context, poaID string) (metadata string, err error) {
	if poaID == "" {
		return "", fmt.Errorf("empty poaID")
	}

	metadata, err = r.client.Get(ctx, r.key(poaID)).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("PoA %s not in revocation blacklist", poaID)
		}
		return "", fmt.Errorf("failed to get revocation metadata for %s: %w", poaID, err)
	}

	return metadata, nil
}

// CountRevocations returns the approximate number of revoked PoAs in the blacklist.
// Useful for monitoring and capacity planning.
//
// Note: This scans Redis keys matching the prefix, which can be expensive for large datasets.
// Use sparingly (e.g., only in admin/monitoring endpoints).
func (r *RevocationBlacklistStore) CountRevocations(ctx context.Context) (count int, err error) {
	pattern := fmt.Sprintf("%s:poa:*", r.prefix)
	iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()

	count = 0
	for iter.Next(ctx) {
		count++
	}

	if err := iter.Err(); err != nil {
		return 0, fmt.Errorf("failed to count revocations: %w", err)
	}

	return count, nil
}

// WithRevocationBlacklistStore is a functional option to inject RevocationBlacklistStore into Service.
func WithRevocationBlacklistStore(client *redis.Client, prefix string, ttl time.Duration) Option {
	return func(s *Service) {
		store, err := NewRevocationBlacklistStore(client, prefix, ttl)
		if err != nil {
			// Log error but don't fail service construction
			// Service will fall back to database status checks (with latency window)
			return
		}
		s.revocationBlacklistStore = store
	}
}
