package gauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/rfc"
)

func TestUsageLedgerPersistence(t *testing.T) {
	s, ctx := setupUsageTestService(t)

	// 1. Create PoA with daily limit of 1000 USD
	req := DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    []string{"payment"},
		Duration: 24 * time.Hour,
		Restrictions: map[string]string{
			"max_daily_amount": "1000",
			"default_currency": "USD",
		},
	}
	resp, err := s.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create delegation: %v", err)
	}
	token := resp.AuthToken

	bobCtx := WithSubject(context.Background(), "bob")

	// 2. First transaction: 400 USD
	tx1Ctx := context.WithValue(bobCtx, ctxKeyRequestedAmount, 400.0)
	_, err = s.VerifyToken(tx1Ctx, token)
	if err != nil {
		t.Fatalf("First verification failed: %v", err)
	}

	// 3. Second transaction: 500 USD
	tx2Ctx := context.WithValue(bobCtx, ctxKeyRequestedAmount, 500.0)
	_, err = s.VerifyToken(tx2Ctx, token)
	if err != nil {
		t.Fatalf("Second verification failed: %v", err)
	}

	// 4. Third transaction: 200 USD -> Should fail (400+500+200 = 1100 > 1000)
	tx3Ctx := context.WithValue(bobCtx, ctxKeyRequestedAmount, 200.0)
	_, err = s.VerifyToken(tx3Ctx, token)
	if err == nil {
		t.Error("Third verification should have failed due to daily limit")
	} else {
		rfcErr, ok := err.(rfc.RFCError)
		if !ok || rfcErr.Code != rfc.ErrInvalidRequest {
			t.Errorf("Expected ErrInvalidRequest, got %v", err)
		}
	}
}

func TestCurrencyNormalization(t *testing.T) {
	s, ctx := setupUsageTestService(t)
	// StaticCurrencyConverter: 1 EUR = 1.1 USD

	// 1. Create PoA with daily limit of 100 USD
	req := DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    []string{"payment"},
		Duration: 24 * time.Hour,
		Restrictions: map[string]string{
			"max_daily_amount": "100",
			"default_currency": "USD",
		},
	}
	resp, err := s.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create delegation: %v", err)
	}
	token := resp.AuthToken

	bobCtx := WithSubject(context.Background(), "bob")

	// 2. Transaction for 90 EUR -> Should be ~99 USD -> Success
	tx1Ctx := context.WithValue(bobCtx, ctxKeyRequestedAmount, 90.0)
	tx1Ctx = context.WithValue(tx1Ctx, "currency", "EUR")
	_, err = s.VerifyToken(tx1Ctx, token)
	if err != nil {
		t.Fatalf("EUR verification failed: %v", err)
	}

	// 3. Transaction for 2 EUR -> Should be ~2.2 USD -> Total ~101.2 USD -> Failure
	tx2Ctx := context.WithValue(bobCtx, ctxKeyRequestedAmount, 2.0)
	tx2Ctx = context.WithValue(tx2Ctx, "currency", "EUR")
	_, err = s.VerifyToken(tx2Ctx, token)
	if err == nil {
		t.Error("Second EUR verification should have failed due to daily limit in USD")
	}
}

func setupUsageTestService(t *testing.T) (*Service, context.Context) {
	s, _ := setupTestServiceForStatus(t) // Reusing existing setup helper

	// Inject enhanced validator with store and converter
	store, _ := NewBoltDailyLimitStore("") // In-memory bolt for testing
	converter := NewStaticCurrencyConverter()

	s.enhancedValidator = NewEnhancedPoAValidator(
		WithLimitStore(store),
		WithCurrencyConverter(converter),
	)

	ctx := WithSubject(context.Background(), "alice")
	return s, ctx
}
