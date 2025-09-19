package token

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"
)

func TestTokenServiceSigningKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	config := &Config{
		SigningKey:     key,
		SigningMethod:  RS256,
		ValidityPeriod: time.Hour,
	}
	ts := NewService(config, NewMemoryStore())
	tok := &Token{Subject: "test", Type: Access}
	_, err = ts.Issue(context.Background(), tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
