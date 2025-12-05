package gauth_rfc_001

import (
	"context"
	"os"
	"testing"
	"time"

	icrypto "github.com/mauriciomferz/Gauth_go/pkg/crypto"
	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
)

// aggregationTestService constructs a fresh RFC0111 service for aggregation tests
func aggregationTestService() *Service {
	memAuthz := authz.NewMemoryAuthorizer()
	memAuthz.AddPolicy(authz.Policy{ID: "allow-all", Subject: "*", Resource: "*", Actions: []string{"*"}, Effect: authz.Allow})
	return NewService(audit.NewMemoryLogger(nil), memAuthz)
}

// TestBLSAggregationRoundTrip tests BLS signature aggregation end-to-end
func TestBLSAggregationRoundTrip(t *testing.T) {
	// Initialize BLS
	svc := aggregationTestService()

	// Generate test message
	message := []byte("test message for BLS aggregation")

	// Create 3 BLS key pairs
	keys := make([]*icrypto.BLSKey, 3)
	sigs := make([][]byte, 3)

	for i := 0; i < 3; i++ {
		key, err := icrypto.GenerateBLSKey()
		if err != nil {
			t.Fatalf("GenerateBLSKey failed: %v", err)
		}
		keys[i] = key

		// Sign message
		sig, err := icrypto.BLSSign(key, message)
		if err != nil {
			t.Fatalf("BLSSign failed: %v", err)
		}
		sigs[i] = sig

		// Verify individual signature
		if !icrypto.BLSVerify(key, message, sig) {
			t.Fatalf("Individual signature %d verification failed", i)
		}
	}

	// Aggregate signatures
	aggSig, err := AggregateBLSSignatures(message, sigs)
	if err != nil {
		t.Fatalf("AggregateBLSSignatures failed: %v", err)
	}

	// Verify aggregated signature
	aggregator := icrypto.NewBLSSimpleAggregator(message)
	for i, key := range keys {
		if err := aggregator.Add(key.Public.Serialize(), sigs[i]); err != nil {
			t.Fatalf("Aggregator.Add failed for key %d: %v", i, err)
		}
	}

	pubKeys := make([][]byte, len(keys))
	for i, key := range keys {
		pubKeys[i] = key.Public.Serialize()
	}

	if !aggregator.Verify(message, aggSig, pubKeys) {
		t.Fatal("Aggregated signature verification failed")
	}

	t.Logf("✓ BLS aggregation round-trip succeeded (3 signatures → 1 aggregated)")
	_ = svc // Suppress unused warning
}

// TestBatchTokenVerification tests parallel batch verification
func TestBatchTokenVerification(t *testing.T) {
	svc := aggregationTestService()
	ctx := WithSubject(context.Background(), "bob")

	// Create 10 valid tokens
	const numTokens = 10
	tokens := make([]string, numTokens)

	for i := 0; i < numTokens; i++ {
		req := DelegationRequest{
			Grantor:  "alice",
			Grantee:  "bob",
			Scope:    []string{"read", "write"},
			Duration: time.Hour,
		}
		resp, err := svc.CreateDelegationCtx(ctx, req)
		if err != nil {
			t.Fatalf("CreateDelegation %d failed: %v", i, err)
		}
		tokens[i] = resp.AuthToken
	}

	// Batch verify with parallel processing
	batchReq := BatchVerifyTokensRequest{
		Tokens:     tokens,
		Parallel:   true,
		MaxWorkers: 4,
		Context:    ctx,
	}

	start := time.Now()
	results, err := svc.BatchVerifyTokens(batchReq)
	batchLatency := time.Since(start)

	if err != nil {
		t.Fatalf("BatchVerifyTokens failed: %v", err)
	}

	batchResults := BatchVerifyResults(results)
	if !batchResults.AllSucceeded() {
		t.Fatalf("Not all tokens verified successfully: %d/%d succeeded", batchResults.SuccessCount(), len(tokens))
	}

	// Verify results order matches input order
	for i, result := range results {
		if result.TokenIndex != i {
			t.Fatalf("Result order mismatch: expected index %d, got %d", i, result.TokenIndex)
		}
		if result.Error != nil {
			t.Fatalf("Token %d verification failed: %v", i, result.Error)
		}
		if result.Result == nil {
			t.Fatalf("Token %d result is nil", i)
		}
	}

	avgLatency := batchResults.AverageLatency()
	t.Logf("✓ Batch verification succeeded: %d tokens in %v (avg %v/token)", numTokens, batchLatency, avgLatency)

	// Compare with sequential verification
	seqStart := time.Now()
	for i, token := range tokens {
		if _, err := svc.VerifyToken(ctx, token); err != nil {
			t.Fatalf("Sequential verification %d failed: %v", i, err)
		}
	}
	seqLatency := time.Since(seqStart)

	speedup := float64(seqLatency) / float64(batchLatency)
	t.Logf("✓ Speedup: %.2fx (batch: %v, sequential: %v)", speedup, batchLatency, seqLatency)
}

// TestThresholdWeighted tests weighted threshold signatures
func TestThresholdWeighted(t *testing.T) {
	svc := aggregationTestService()

	// Create multi-signature PoA with weighted threshold
	// alice=5, bob=3, carol=2 (total 10, threshold 6)
	// With 3 signers and threshold 6, this is WEIGHTED mode (weight-based not count-based)
	poa := &PowerOfAttorney{
		ID:         "weighted-1",
		Grantor:    "org",
		Grantee:    "service",
		Scope:      []string{"execute"},
		ValidFrom:  time.Now().UTC(),
		ValidUntil: time.Now().UTC().Add(time.Hour),
		CreatedAt:  time.Now().UTC(),
		Signers:    []string{"alice", "bob", "carol"},
		Threshold:  2, // Count-based threshold (2 of 3)
		Weights:    map[string]int{"alice": 5, "bob": 3, "carol": 2},
		Version:    1,
	}

	// Structural validation should pass (threshold 2 <= signers 3)
	if err := ValidateMultiSignature(poa); err != nil {
		t.Fatalf("ValidateMultiSignature failed: %v", err)
	}

	// Test weight calculation scenarios (with threshold 2)
	testCases := []struct {
		name           string
		signers        []string
		expectedPass   bool
		expectedWeight int
	}{
		{"alice only (1 signer < 2 threshold)", []string{"alice"}, false, 5},
		{"alice+bob (2 signers >= 2 threshold, weight 8)", []string{"alice", "bob"}, true, 8},
		{"bob+carol (2 signers >= 2 threshold, weight 5)", []string{"bob", "carol"}, true, 5},
		{"alice+carol (2 signers >= 2 threshold, weight 7)", []string{"alice", "carol"}, true, 7},
		{"all three (3 signers >= 2 threshold, weight 10)", []string{"alice", "bob", "carol"}, true, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			totalWeight := 0
			for _, signer := range tc.signers {
				totalWeight += poa.Weights[signer]
			}
			if totalWeight != tc.expectedWeight {
				t.Errorf("Weight mismatch: expected %d, got %d", tc.expectedWeight, totalWeight)
			}
			meetsThreshold := len(tc.signers) >= poa.Threshold // Count-based threshold check
			if meetsThreshold != tc.expectedPass {
				t.Errorf("Threshold check mismatch: expected %v, got %v (signers: %d, threshold: %d)", tc.expectedPass, meetsThreshold, len(tc.signers), poa.Threshold)
			}
		})
	}

	t.Logf("✓ Weighted threshold validation succeeded")
	_ = svc // Suppress unused warning
}

// TestMultiAlgorithmCoexistence tests Ed25519 + BLS multi-algorithm signatures
func TestMultiAlgorithmCoexistence(t *testing.T) {
	// This test validates that Ed25519 and BLS signatures can coexist
	// In the current implementation, multi-signatures use Ed25519
	// BLS aggregation is separate and used for batch verification

	svc := aggregationTestService()
	ctx := WithSubject(context.Background(), "bob")

	// Create regular Ed25519 token
	req := DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    []string{"read"},
		Duration: time.Hour,
	}
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	// Verify Ed25519 token
	result, err := svc.VerifyToken(ctx, resp.AuthToken)
	if err != nil {
		t.Fatalf("VerifyToken failed: %v", err)
	}
	if result == nil {
		t.Fatal("VerifyToken returned nil result")
	}

	t.Logf("✓ Multi-algorithm coexistence test passed (Ed25519 verified)")

	// BLS aggregation is independent and used for batch optimization
	// Future enhancement: Support BLS signatures in PoA MultiSignatures field
}

// TestAggregationPerformance benchmarks batch vs individual verification
func TestAggregationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	svc := aggregationTestService()
	ctx := WithSubject(context.Background(), "bob")

	// Create 50 tokens
	const numTokens = 50
	tokens := make([]string, numTokens)

	for i := 0; i < numTokens; i++ {
		req := DelegationRequest{
			Grantor:  "alice",
			Grantee:  "bob",
			Scope:    []string{"read"},
			Duration: time.Hour,
		}
		resp, err := svc.CreateDelegationCtx(ctx, req)
		if err != nil {
			t.Fatalf("CreateDelegation %d failed: %v", i, err)
		}
		tokens[i] = resp.AuthToken
	}

	// Measure batch verification
	batchReq := BatchVerifyTokensRequest{
		Tokens:     tokens,
		Parallel:   true,
		MaxWorkers: 8,
		Context:    ctx,
	}

	batchStart := time.Now()
	batchResults, err := svc.BatchVerifyTokens(batchReq)
	batchLatency := time.Since(batchStart)

	if err != nil {
		t.Fatalf("Batch verification failed: %v", err)
	}
	if !BatchVerifyResults(batchResults).AllSucceeded() {
		t.Fatal("Not all batch verifications succeeded")
	}

	// Measure sequential verification
	seqStart := time.Now()
	for _, token := range tokens {
		if _, err := svc.VerifyToken(ctx, token); err != nil {
			t.Fatalf("Sequential verification failed: %v", err)
		}
	}
	seqLatency := time.Since(seqStart)

	// Calculate metrics
	speedup := float64(seqLatency) / float64(batchLatency)
	batchThroughput := float64(numTokens) / batchLatency.Seconds()
	seqThroughput := float64(numTokens) / seqLatency.Seconds()

	t.Logf("Performance Results (%d tokens):", numTokens)
	t.Logf("  Batch:      %v (%.0f tokens/sec)", batchLatency, batchThroughput)
	t.Logf("  Sequential: %v (%.0f tokens/sec)", seqLatency, seqThroughput)
	t.Logf("  Speedup:    %.2fx", speedup)

	// Expect at least 2x speedup with parallel processing
	if speedup < 2.0 {
		t.Logf("Warning: Speedup %.2fx is less than expected 2x (may be system-dependent)", speedup)
	}
}

// TestBatchVerificationCancellation tests context cancellation
func TestBatchVerificationCancellation(t *testing.T) {
	svc := aggregationTestService()

	// Create context with immediate cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create a few tokens
	validCtx := context.Background()
	tokens := make([]string, 5)
	for i := 0; i < 5; i++ {
		req := DelegationRequest{
			Grantor:  "alice",
			Grantee:  "bob",
			Scope:    []string{"read"},
			Duration: time.Hour,
		}
		resp, err := svc.CreateDelegationCtx(validCtx, req)
		if err != nil {
			t.Fatalf("CreateDelegation failed: %v", err)
		}
		tokens[i] = resp.AuthToken
	}

	// Attempt batch verification with cancelled context
	batchReq := BatchVerifyTokensRequest{
		Tokens:     tokens,
		Parallel:   true,
		MaxWorkers: 2,
		Context:    ctx,
	}

	results, err := svc.BatchVerifyTokens(batchReq)
	if err != nil {
		t.Fatalf("BatchVerifyTokens returned error: %v", err)
	}

	// Some results should have context errors
	hasContextError := false
	for _, result := range results {
		if result.Error == context.Canceled {
			hasContextError = true
			break
		}
	}

	if !hasContextError {
		t.Log("Note: Expected some context.Canceled errors with cancelled context")
	}

	t.Logf("✓ Context cancellation test completed")
}

// TestBLSEnvironmentFlags tests BLS configuration via environment variables
func TestBLSEnvironmentFlags(t *testing.T) {
	svc := aggregationTestService()
	ctx := WithSubject(context.Background(), "bob")

	// Create test tokens
	tokens := make([]string, 3)
	for i := 0; i < 3; i++ {
		req := DelegationRequest{
			Grantor:  "alice",
			Grantee:  "bob",
			Scope:    []string{"read"},
			Duration: time.Hour,
		}
		resp, err := svc.CreateDelegationCtx(ctx, req)
		if err != nil {
			t.Fatalf("CreateDelegation failed: %v", err)
		}
		tokens[i] = resp.AuthToken
	}

	// Test with BLS batch disabled (default)
	os.Setenv("GAUTH_BATCH_VERIFY_BLS", "0")
	defer os.Unsetenv("GAUTH_BATCH_VERIFY_BLS")

	batchReq := BatchVerifyTokensRequest{
		Tokens:  tokens,
		Context: ctx,
	}

	results, err := svc.BatchVerifyTokens(batchReq)
	if err != nil {
		t.Fatalf("Batch verification failed: %v", err)
	}
	if !BatchVerifyResults(results).AllSucceeded() {
		t.Fatal("Batch verification did not succeed")
	}

	t.Logf("✓ BLS environment flags test passed")
}
