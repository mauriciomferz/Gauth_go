package cache

import (
	"fmt"
)

// Cache key prefixes for different types of data
const (
	PrefixVerification = "agentauth:verification:"
	PrefixPoA          = "agentauth:poa:"
	PrefixUser         = "agentauth:user:"
	PrefixStats        = "agentauth:stats:"
	PrefixBlockchain   = "agentauth:blockchain:"
	PrefixSession      = "agentauth:session:"
)

// KeyBuilder provides helper methods for building cache keys
type KeyBuilder struct{}

// NewKeyBuilder creates a new key builder instance
func NewKeyBuilder() *KeyBuilder {
	return &KeyBuilder{}
}

// VerificationKey builds a cache key for verification results
func (k *KeyBuilder) VerificationKey(poaID string) string {
	return fmt.Sprintf("%s%s", PrefixVerification, poaID)
}

// PoAKey builds a cache key for PoA metadata
func (k *KeyBuilder) PoAKey(poaID string) string {
	return fmt.Sprintf("%s%s", PrefixPoA, poaID)
}

// PoAListKey builds a cache key for PoA lists by user
func (k *KeyBuilder) PoAListKey(userID string) string {
	return fmt.Sprintf("%slist:%s", PrefixPoA, userID)
}

// UserKey builds a cache key for user data
func (k *KeyBuilder) UserKey(userID string) string {
	return fmt.Sprintf("%s%s", PrefixUser, userID)
}

// StatsKey builds a cache key for statistics
func (k *KeyBuilder) StatsKey(statType string) string {
	return fmt.Sprintf("%s%s", PrefixStats, statType)
}

// BlockchainSyncKey builds a cache key for blockchain sync status
func (k *KeyBuilder) BlockchainSyncKey(poaID string) string {
	return fmt.Sprintf("%ssync:%s", PrefixBlockchain, poaID)
}

// BlockchainVerifyKey builds a cache key for blockchain verification
func (k *KeyBuilder) BlockchainVerifyKey(poaID string) string {
	return fmt.Sprintf("%sverify:%s", PrefixBlockchain, poaID)
}

// SessionKey builds a cache key for user sessions
func (k *KeyBuilder) SessionKey(sessionID string) string {
	return fmt.Sprintf("%s%s", PrefixSession, sessionID)
}

// InvalidatePoAPattern returns a pattern to invalidate all PoA-related cache entries
func (k *KeyBuilder) InvalidatePoAPattern(poaID string) string {
	return fmt.Sprintf("agentauth:*:%s", poaID)
}

// InvalidateUserPattern returns a pattern to invalidate all user-related cache entries
func (k *KeyBuilder) InvalidateUserPattern(userID string) string {
	return fmt.Sprintf("%s*%s*", PrefixPoA, userID)
}
