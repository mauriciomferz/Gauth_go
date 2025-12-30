package notarization

import (
	"testing"
	"time"
)

func TestBlockchainNotary_Notarize(t *testing.T) {
	config := &BlockchainConfig{
		ChainType:    "ethereum",
		ContractAddr: "0x1234567890abcdef",
		Endpoint:     "https://eth.example.com",
	}

	notary, err := NewBlockchainNotary(config)
	if err != nil {
		t.Fatalf("Failed to create notary: %v", err)
	}

	event := &RevocationEvent{
		DelegationID: "del-001",
		Subject:      "user:alice",
		Delegate:     "user:bob",
		Reason:       "policy violation",
		RevokedAt:    time.Now(),
		RevokedBy:    "admin",
	}

	proof, err := notary.Notarize(event)
	if err != nil {
		t.Fatalf("Failed to notarize: %v", err)
	}

	if proof == nil {
		t.Fatal("Proof is nil")
	}

	if proof.EventHash == "" {
		t.Error("Event hash is empty")
	}

	if proof.TransactionID == "" {
		t.Error("Transaction ID is empty")
	}

	if proof.BlockNumber == 0 {
		t.Error("Block number is zero")
	}

	if proof.NotaryProvider != "blockchain:ethereum" {
		t.Errorf("Expected provider blockchain:ethereum, got %s", proof.NotaryProvider)
	}
}

func TestBlockchainNotary_Verify(t *testing.T) {
	config := &BlockchainConfig{
		ChainType: "polygon",
	}

	notary, _ := NewBlockchainNotary(config)

	event := &RevocationEvent{
		DelegationID: "del-002",
		Subject:      "user:charlie",
		Delegate:     "user:david",
		Reason:       "expired",
		RevokedAt:    time.Now(),
		RevokedBy:    "system",
	}

	proof, _ := notary.Notarize(event)

	valid, err := notary.Verify(event, proof)
	if err != nil {
		t.Fatalf("Verification failed: %v", err)
	}

	if !valid {
		t.Error("Proof should be valid")
	}
}

func TestBlockchainNotary_VerifyInvalidProof(t *testing.T) {
	config := &BlockchainConfig{
		ChainType: "avalanche",
	}

	notary, _ := NewBlockchainNotary(config)

	event := &RevocationEvent{
		DelegationID: "del-003",
		Subject:      "user:eve",
		Delegate:     "user:frank",
		Reason:       "test",
		RevokedAt:    time.Now(),
		RevokedBy:    "admin",
	}

	// Create proof for different event
	differentEvent := &RevocationEvent{
		DelegationID: "del-999",
		Subject:      "user:different",
		Delegate:     "user:other",
		Reason:       "other",
		RevokedAt:    time.Now(),
		RevokedBy:    "admin",
	}

	proof, _ := notary.Notarize(differentEvent)

	// Verify should fail
	valid, _ := notary.Verify(event, proof)
	if valid {
		t.Error("Proof should be invalid for different event")
	}
}

func TestRFC3161TimestampNotary(t *testing.T) {
	notary, err := NewRFC3161TimestampNotary("https://tsa.example.com")
	if err != nil {
		t.Fatalf("Failed to create notary: %v", err)
	}

	event := &RevocationEvent{
		DelegationID: "del-004",
		Subject:      "user:george",
		Delegate:     "user:helen",
		Reason:       "policy change",
		RevokedAt:    time.Now(),
		RevokedBy:    "compliance",
	}

	proof, err := notary.Notarize(event)
	if err != nil {
		t.Fatalf("Failed to notarize: %v", err)
	}

	if proof.TimestampToken == "" {
		t.Error("Timestamp token is empty")
	}

	if proof.NotaryProvider != "rfc3161" {
		t.Errorf("Expected provider rfc3161, got %s", proof.NotaryProvider)
	}

	// Verify
	valid, err := notary.Verify(event, proof)
	if err != nil {
		t.Fatalf("Verification failed: %v", err)
	}

	if !valid {
		t.Error("RFC 3161 proof should be valid")
	}
}

func TestMultiNotary(t *testing.T) {
	blockchain, _ := NewBlockchainNotary(&BlockchainConfig{ChainType: "ethereum"})
	rfc3161, _ := NewRFC3161TimestampNotary("https://tsa.example.com")
	multiNotary, err := NewMultiNotary(blockchain, rfc3161)
	if err != nil {
		t.Fatalf("Failed to create multi-notary: %v", err)
	}

	event := &RevocationEvent{
		DelegationID: "del-005",
		Subject:      "user:ivan",
		Delegate:     "user:julia",
		Reason:       "security",
		RevokedAt:    time.Now(),
		RevokedBy:    "security-team",
	}

	proof, err := multiNotary.Notarize(event)
	if err != nil {
		t.Fatalf("Failed to notarize with multi-notary: %v", err)
	}

	// Should have primary proof + additional proofs
	if proof == nil {
		t.Fatal("Proof is nil")
	}

	if len(proof.AdditionalProofs) == 0 {
		t.Error("Multi-notary should have additional proofs")
	}

	// Verify
	valid, err := multiNotary.Verify(event, proof)
	if err != nil {
		t.Fatalf("Verification failed: %v", err)
	}

	if !valid {
		t.Error("Multi-notary proof should be valid")
	}
}

func TestNotarizationRegistry(t *testing.T) {
	notary, _ := NewBlockchainNotary(&BlockchainConfig{ChainType: "ethereum"})
	registry := NewNotarizationRegistry(notary)

	event := &RevocationEvent{
		DelegationID: "del-006",
		Subject:      "user:kevin",
		Delegate:     "user:linda",
		Reason:       "audit",
		RevokedAt:    time.Now(),
		RevokedBy:    "auditor",
	}

	// Notarize and store
	proof, err := registry.NotarizeRevocation(event)
	if err != nil {
		t.Fatalf("Failed to notarize revocation: %v", err)
	}

	if proof == nil {
		t.Fatal("Proof is nil")
	}

	// Retrieve proofs
	proofs := registry.GetProofs("del-006")
	if len(proofs) != 1 {
		t.Errorf("Expected 1 proof, got %d", len(proofs))
	}

	// Verify
	valid, err := registry.VerifyRevocation(event, proof)
	if err != nil {
		t.Fatalf("Verification failed: %v", err)
	}

	if !valid {
		t.Error("Registry proof should be valid")
	}

	// Check stats
	stats := registry.GetStats()
	if stats["total_delegations"].(int) != 1 {
		t.Errorf("Expected 1 delegation in registry, got %d", stats["total_delegations"])
	}
}

func TestNotarizationRegistry_MultipleRevocations(t *testing.T) {
	notary, _ := NewBlockchainNotary(&BlockchainConfig{ChainType: "polygon"})
	registry := NewNotarizationRegistry(notary)

	// Notarize same delegation multiple times
	delegationID := "del-007"

	for i := 0; i < 3; i++ {
		event := &RevocationEvent{
			DelegationID: delegationID,
			Subject:      "user:mike",
			Delegate:     "user:nancy",
			Reason:       "test revocation",
			RevokedAt:    time.Now().Add(time.Duration(i) * time.Hour),
			RevokedBy:    "tester",
		}

		_, err := registry.NotarizeRevocation(event)
		if err != nil {
			t.Fatalf("Failed to notarize: %v", err)
		}
	}

	// Should have 3 proofs for the same delegation
	proofs := registry.GetProofs(delegationID)
	if len(proofs) != 3 {
		t.Errorf("Expected 3 proofs, got %d", len(proofs))
	}
}

func TestBlockchainNotary_GetStats(t *testing.T) {
	config := &BlockchainConfig{
		ChainType: "ethereum",
	}

	notary, _ := NewBlockchainNotary(config)

	// Notarize a few events
	for i := 0; i < 5; i++ {
		event := &RevocationEvent{
			DelegationID: "del-" + string(rune('0'+i)),
			Subject:      "user:test",
			Delegate:     "user:test2",
			Reason:       "test",
			RevokedAt:    time.Now(),
			RevokedBy:    "admin",
		}
		_, _ = notary.Notarize(event) //nolint:errcheck
	}

	stats := notary.GetStats()

	if stats["total_notarized"].(int64) != 5 {
		t.Errorf("Expected 5 notarizations, got %d", stats["total_notarized"])
	}

	if stats["chain_type"].(string) != "ethereum" {
		t.Errorf("Expected chain_type ethereum, got %s", stats["chain_type"])
	}
}

func TestNewBlockchainNotary_Defaults(t *testing.T) {
	config := &BlockchainConfig{} // Empty config

	notary, err := NewBlockchainNotary(config)
	if err != nil {
		t.Fatalf("Failed to create notary with defaults: %v", err)
	}

	if notary.chainType != "ethereum" {
		t.Errorf("Expected default chain type ethereum, got %s", notary.chainType)
	}

	if notary.batchSize != 100 {
		t.Errorf("Expected default batch size 100, got %d", notary.batchSize)
	}

	if notary.batchTimeout != 5*time.Minute {
		t.Errorf("Expected default batch timeout 5m, got %v", notary.batchTimeout)
	}
}

func TestNewRFC3161TimestampNotary_InvalidURL(t *testing.T) {
	_, err := NewRFC3161TimestampNotary("")
	if err == nil {
		t.Error("Should fail with empty TSA URL")
	}
}

func TestNewMultiNotary_NoProviders(t *testing.T) {
	_, err := NewMultiNotary()
	if err == nil {
		t.Error("Should fail with no providers")
	}
}

func TestBlockchainNotary_NilEvent(t *testing.T) {
	config := &BlockchainConfig{ChainType: "ethereum"}
	notary, _ := NewBlockchainNotary(config)

	_, err := notary.Notarize(nil)
	if err == nil {
		t.Error("Should fail with nil event")
	}
}

func BenchmarkBlockchainNotary_Notarize(b *testing.B) {
	config := &BlockchainConfig{ChainType: "ethereum"}
	notary, _ := NewBlockchainNotary(config)

	event := &RevocationEvent{
		DelegationID: "del-bench",
		Subject:      "user:bench",
		Delegate:     "user:bench2",
		Reason:       "benchmark",
		RevokedAt:    time.Now(),
		RevokedBy:    "system",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = notary.Notarize(event) //nolint:errcheck
	}
}

func BenchmarkRFC3161_Notarize(b *testing.B) {
	notary, _ := NewRFC3161TimestampNotary("https://tsa.example.com")

	event := &RevocationEvent{
		DelegationID: "del-bench",
		Subject:      "user:bench",
		Delegate:     "user:bench2",
		Reason:       "benchmark",
		RevokedAt:    time.Now(),
		RevokedBy:    "system",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = notary.Notarize(event) //nolint:errcheck
	}
}
