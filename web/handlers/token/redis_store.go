package token

import (
	"context"
	"encoding/json"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/mauriciomferz/AgentAuth/pkg/cache"
)

type RedisTokenStore struct {
	cache  cache.Cache
	tracer trace.Tracer
}

func NewRedisTokenStore(c cache.Cache) *RedisTokenStore {
	return &RedisTokenStore{
		cache:  c,
		tracer: otel.Tracer("agentauth.redis.token_store"),
	}
}

// Ensure interface compliance
var _ TokenStorer = (*RedisTokenStore)(nil)

func (s *RedisTokenStore) Create(ttlSeconds int, meta any) *Token {
	if ttlSeconds <= 0 {
		ttlSeconds = 300
	}
	// Start span (detached context since interface doesn't support it yet)
	ctx, span := s.tracer.Start(context.Background(), "CreateToken")
	defer span.End()

	t := &Token{
		ID:        randomNonce(10),
		Value:     randomNonce(24),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Duration(ttlSeconds) * time.Second),
		Meta:      meta,
		Status:    "active",
	}

	span.SetAttributes(attribute.String("token.id", t.ID))

	// Store ID -> Token
	data, _ := json.Marshal(t)
	err := s.cache.Set(ctx, "token:"+t.ID, data, time.Duration(ttlSeconds)*time.Second)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	// Store Value -> ID mapping
	_ = s.cache.Set(ctx, "token_val:"+t.Value, []byte(t.ID), time.Duration(ttlSeconds)*time.Second)

	return t
}

func (s *RedisTokenStore) Validate(idOrVal string) (string, *Token) {
	ctx, span := s.tracer.Start(context.Background(), "ValidateToken")
	defer span.End()

	// Try direct lookup (ID)
	data, err := s.cache.Get(ctx, "token:"+idOrVal)
	if err != nil || len(data) == 0 {
		// Try lookup by value
		valData, valErr := s.cache.Get(ctx, "token_val:"+idOrVal)
		if valErr == nil && len(valData) > 0 {
			tokenID := string(valData)
			dataByID, getErr := s.cache.Get(ctx, "token:"+tokenID)
			if getErr == nil {
				data = dataByID
			}
		}
	}

	if len(data) == 0 {
		span.SetAttributes(attribute.String("validation.result", "not_found"))
		return TokenStatusNotFound, nil
	}

	var t Token
	if unmarshalErr := json.Unmarshal(data, &t); unmarshalErr != nil {
		span.RecordError(unmarshalErr)
		return TokenStatusNotFound, nil
	}

	span.SetAttributes(attribute.String("token.id", t.ID))

	now := time.Now()
	if t.RevokedAt != nil || t.Status == StatusTerminated {
		span.SetAttributes(attribute.String("validation.result", "revoked"))
		return TokenStatusRevoked, &t
	}
	if t.Status == StatusSuspended {
		span.SetAttributes(attribute.String("validation.result", "suspended"))
		return StatusSuspended, &t
	}
	if now.After(t.ExpiresAt) {
		span.SetAttributes(attribute.String("validation.result", "expired"))
		return "expired", &t
	}

	span.SetAttributes(attribute.String("validation.result", "valid"))
	return TokenStatusValid, &t
}

func (s *RedisTokenStore) Revoke(id string) string {
	ctx, span := s.tracer.Start(context.Background(), "RevokeToken")
	defer span.End()
	span.SetAttributes(attribute.String("token.id", id))

	data, err := s.cache.Get(ctx, "token:"+id)
	if err != nil || len(data) == 0 {
		span.SetAttributes(attribute.String("revoke.result", "not_found"))
		return TokenStatusNotFound
	}

	var t Token
	if unmarshalErr := json.Unmarshal(data, &t); unmarshalErr != nil {
		span.RecordError(unmarshalErr)
		return TokenStatusNotFound
	}

	if t.RevokedAt != nil {
		span.SetAttributes(attribute.String("revoke.result", "already_revoked"))
		return TokenStatusAlreadyRevoked
	}

	now := time.Now()
	t.RevokedAt = &now

	// Update
	newData, _ := json.Marshal(t)
	// TTL remaining
	ttl := time.Until(t.ExpiresAt)
	if ttl < 0 {
		ttl = 1 * time.Second
	}

	err = s.cache.Set(ctx, "token:"+id, newData, ttl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update revocation")
	}

	span.SetAttributes(attribute.String("revoke.result", "success"))
	return TokenStatusRevoked
}

func (s *RedisTokenStore) Metrics() map[string]any {
	// Return real stats from Redis
	stats, _ := s.cache.GetStats(context.Background())
	if stats != nil {
		return map[string]any{
			"redis_keys":        stats.Keys,
			"redis_hits":        stats.Hits,
			"redis_misses":      stats.Misses,
			"redis_hit_rate":    stats.HitRate,
			"redis_pool_active": stats.PoolActive,
			"redis_pool_idle":   stats.PoolIdle,
		}
	}
	return map[string]any{
		"error": "stats_unavailable",
	}
}

func (s *RedisTokenStore) UpdateStatus(id, newStatus string) (bool, string, *Token) {
	ctx, span := s.tracer.Start(context.Background(), "UpdateTokenStatus")
	defer span.End()
	span.SetAttributes(attribute.String("token.id", id), attribute.String("status.new", newStatus))

	data, err := s.cache.Get(ctx, "token:"+id)
	if err != nil || len(data) == 0 {
		return false, "not_found", nil
	}

	var t Token
	if unmarshalErr := json.Unmarshal(data, &t); unmarshalErr != nil {
		span.RecordError(unmarshalErr)
		return false, "error", nil
	}

	old := t.Status
	// Terminated is a terminal state
	if old == StatusTerminated && newStatus != StatusTerminated {
		return false, "invalid_transition", &t
	}

	// No-op
	if old == newStatus {
		return true, "noop", &t
	}

	t.Status = newStatus

	// Write back
	newData, _ := json.Marshal(t)
	ttl := time.Until(t.ExpiresAt)
	if ttl < 0 {
		ttl = 1 * time.Second
	}
	err = s.cache.Set(ctx, "token:"+id, newData, ttl)
	if err != nil {
		span.RecordError(err)
		return false, "persist_error", &t
	}

	return true, "success", &t
}
