package gauth_rfc_001

import (
	"crypto/rand"
	"math/big"
	"time"
)

// randomString generates an ASCII alphanumeric string of length n.
func randomString(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		idxBig, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		b[i] = alphabet[idxBig.Int64()]
	}
	return string(b)
}

// buildRandomPOA constructs a PowerOfAttorney with randomized scope & restrictions for testing.
func buildRandomPOA(scopeSize, restrSize int) *PowerOfAttorney {
	scope := make([]string, scopeSize)
	for i := range scope {
		scope[i] = randomString(6)
	}
	restrictions := map[string]string{}
	for i := 0; i < restrSize; i++ {
		restrictions[randomString(5)] = randomString(8)
	}
	now := time.Now().UTC().Add(-time.Minute)
	return &PowerOfAttorney{
		ID:           randomString(10),
		Grantor:      randomString(8),
		Grantee:      randomString(8),
		Scope:        scope,
		Restrictions: restrictions,
		ValidFrom:    now,
		ValidUntil:   now.Add(time.Hour),
		CreatedAt:    now.Add(-5 * time.Second),
		Status:       "active",
		UpdatedAt:    now, // mutable (excluded from digest)
	}
}
