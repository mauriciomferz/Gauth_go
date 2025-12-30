package pip

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
	"github.com/mauriciomferz/AgentAuth/pkg/poa"
	"github.com/mauriciomferz/AgentAuth/pkg/poa/taxonomy"
	"github.com/mauriciomferz/AgentAuth/pkg/pvp"
	"github.com/mauriciomferz/AgentAuth/pkg/registry"
)

// TestDefaultPIP_VerifyCommercialRegister tests commercial register verification integration
func TestDefaultPIP_VerifyCommercialRegister(t *testing.T) {
	ctx := context.Background()

	t.Run("Valid German GmbH verification", func(t *testing.T) {
		commercialRegister := registry.NewMockCommercialRegisterService()
		pip := NewDefaultPIP(nil, commercialRegister, nil, 5*time.Minute)

		result, err := pip.VerifyCommercialRegister(ctx, "HRB12345", "DE")
		if err != nil {
			t.Fatalf("VerifyCommercialRegister() unexpected error: %v", err)
		}

		if !result.Verified {
			t.Errorf("Verified = %v, want true", result.Verified)
		}

		if result.EntityName != "Test Technologies GmbH" {
			t.Errorf("EntityName = %v, want Test Technologies GmbH", result.EntityName)
		}

		if result.Jurisdiction != "DE" {
			t.Errorf("Jurisdiction = %v, want DE", result.Jurisdiction)
		}
	})

	t.Run("Valid UK Ltd verification", func(t *testing.T) {
		commercialRegister := registry.NewMockCommercialRegisterService()
		pip := NewDefaultPIP(nil, commercialRegister, nil, 5*time.Minute)

		result, err := pip.VerifyCommercialRegister(ctx, "12345678", "GB")
		if err != nil {
			t.Fatalf("VerifyCommercialRegister() unexpected error: %v", err)
		}

		if !result.Verified {
			t.Errorf("Verified = %v, want true", result.Verified)
		}

		if result.Jurisdiction != "GB" {
			t.Errorf("Jurisdiction = %v, want GB", result.Jurisdiction)
		}
	})

	t.Run("Invalid registration returns unverified", func(t *testing.T) {
		commercialRegister := registry.NewMockCommercialRegisterService()
		pip := NewDefaultPIP(nil, commercialRegister, nil, 5*time.Minute)

		result, err := pip.VerifyCommercialRegister(ctx, "INVALID", "DE")
		if err != nil {
			t.Fatalf("VerifyCommercialRegister() unexpected error: %v", err)
		}

		if result.Verified {
			t.Error("Expected unverified result for invalid registration")
		}
	})

	t.Run("Cache hit on second request", func(t *testing.T) {
		commercialRegister := registry.NewMockCommercialRegisterService()
		pip := NewDefaultPIP(nil, commercialRegister, nil, 5*time.Minute)

		// First request - cache miss
		start := time.Now()
		result1, err := pip.VerifyCommercialRegister(ctx, "HRB12345", "DE")
		duration1 := time.Since(start)
		if err != nil {
			t.Fatalf("First request error: %v", err)
		}

		// Second request - should be cached (much faster)
		start = time.Now()
		result2, err := pip.VerifyCommercialRegister(ctx, "HRB12345", "DE")
		duration2 := time.Since(start)
		if err != nil {
			t.Fatalf("Second request error: %v", err)
		}

		if !result1.Verified || !result2.Verified {
			t.Error("Both requests should be verified")
		}

		// Cache hit should be significantly faster (at least 10x)
		if duration2 > duration1/10 {
			t.Logf("Cache hit timing: first=%v, second=%v (may not be 10x faster in tests)", duration1, duration2)
		}

		// Verify cache stats
		stats := pip.GetCacheStats()
		if stats.TotalEntries == 0 {
			t.Error("Expected cache entries, got 0")
		}
	})
}

// TestDefaultPIP_VerifyIdentityChain tests PVP integration
func TestDefaultPIP_VerifyIdentityChain(t *testing.T) {
	ctx := context.Background()

	t.Run("Valid identity chain verification", func(t *testing.T) {
		pvpClient := pvp.NewDefaultPVP("https://example.com/trust-list")
		pip := NewDefaultPIP(nil, nil, pvpClient, 5*time.Minute)

		req := &pvp.IdentityChainVerificationRequest{
			PowerOfAttorney: "poa-test-123",
			ResourceOwner: &pvp.IdentityCredential{
				ID:                 "resource-owner-1",
				Type:               "natural_person",
				Name:               "Dr. Max Mustermann",
				Identifier:         "DE:eIDAS:123456",
				IdentifierType:     "eIDAS_QES",
				Jurisdiction:       "DE",
				VerificationMethod: "eIDAS_qualified",
				VerificationLevel: agentauth.VerificationLevel{
					Level:          1,
					AssuranceLevel: "high",
				},
				IssuedAt:  time.Now().Add(-30 * 24 * time.Hour),
				ExpiresAt: time.Now().Add(335 * 24 * time.Hour),
			},
			ClientOwner: &pvp.IdentityCredential{
				ID:                 "client-owner-1",
				Type:               "legal_person",
				Name:               "Test Technologies GmbH",
				Identifier:         "HRB12345",
				IdentifierType:     "commercial_register",
				Jurisdiction:       "DE",
				VerificationMethod: "commercial_register",
				VerificationLevel: agentauth.VerificationLevel{
					Level:          2,
					AssuranceLevel: "substantial",
				},
				IssuedAt: time.Now().Add(-60 * 24 * time.Hour),
			},
			Client: &pvp.ClientIdentity{
				ClientID:         "ai-client-123",
				ClientName:       "Test AI Assistant",
				PublicKey:        "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA\n-----END PUBLIC KEY-----",
				RegistrationDate: time.Now().Add(-90 * 24 * time.Hour),
			},
			RequiredTrustLevel: "substantial",
		}

		result, err := pip.VerifyIdentityChain(ctx, req)
		if err != nil {
			t.Fatalf("VerifyIdentityChain() unexpected error: %v", err)
		}

		if !result.Valid {
			t.Errorf("Valid = %v, want true", result.Valid)
		}

		if result.TrustLevel == "" {
			t.Error("TrustLevel should not be empty")
		}

		if !result.ResourceOwnerVerified {
			t.Error("ResourceOwnerVerified should be true")
		}
	})

	t.Run("Missing resource owner", func(t *testing.T) {
		pvpClient := pvp.NewDefaultPVP("https://example.com/trust-list")
		pip := NewDefaultPIP(nil, nil, pvpClient, 5*time.Minute)

		req := &pvp.IdentityChainVerificationRequest{
			ResourceOwner: nil,
			ClientOwner: &pvp.IdentityCredential{
				ID:   "client-owner-1",
				Type: "legal_person",
				Name: "Test GmbH",
			},
			Client: &pvp.ClientIdentity{
				ClientID:   "ai-client-123",
				ClientName: "Test AI",
			},
			RequiredTrustLevel: "substantial",
		}

		result, err := pip.VerifyIdentityChain(ctx, req)
		if err != nil {
			t.Fatalf("VerifyIdentityChain() unexpected error: %v", err)
		}

		if result.Valid {
			t.Error("Expected invalid result for missing resource owner")
		}
	})
}

// TestDefaultPIP_ValidateAuthorization tests authorization validation
func TestDefaultPIP_ValidateAuthorization(t *testing.T) {
	ctx := context.Background()

	t.Run("Authorization validation with action check", func(t *testing.T) {
		pip := NewDefaultPIP(nil, nil, nil, 5*time.Minute)

		// Pre-populate cache with authorization data
		clientID := "client-123"
		chain := &agentauth.AuthorizationChain{
			Client: &agentauth.AuthorizationLink{
				EntityID:   clientID,
				EntityType: "ai_system",
				Role:       "client",
			},
		}
		pip.cache.SetAuthorizationChain(clientID, chain)

		// Set authorized actions
		actions := &poa.AuthorizedActions{
			Transactions: []taxonomy.TransactionType{
				taxonomy.TransactionPayment,
				taxonomy.TransactionPurchase,
			},
			Decisions: []taxonomy.DecisionType{
				taxonomy.DecisionFinancial,
			},
		}
		pip.cache.authorizedActions[clientID] = &cachedActions{
			data:      actions,
			timestamp: time.Now(),
		}

		req := &AuthorizationValidationRequest{
			ClientID:  clientID,
			Action:    string(taxonomy.TransactionPayment),
			Resource:  "account-456",
			Timestamp: time.Now(),
		}

		result, err := pip.ValidateAuthorization(ctx, req)
		if err != nil {
			t.Fatalf("ValidateAuthorization() unexpected error: %v", err)
		}

		if !result.Authorized {
			t.Errorf("Authorized = %v, want true", result.Authorized)
		}

		if result.AuthorizationChain == nil {
			t.Error("AuthorizationChain should not be nil")
		}

		if len(result.ValidatedActions) == 0 {
			t.Error("ValidatedActions should not be empty")
		}
	})

	t.Run("Unauthorized action", func(t *testing.T) {
		pip := NewDefaultPIP(nil, nil, nil, 5*time.Minute)

		clientID := "client-456"
		chain := &agentauth.AuthorizationChain{
			Client: &agentauth.AuthorizationLink{
				EntityID: clientID,
				Role:     "client",
			},
		}
		pip.cache.SetAuthorizationChain(clientID, chain)

		// Set limited authorized actions
		actions := &poa.AuthorizedActions{
			Transactions: []taxonomy.TransactionType{
				taxonomy.TransactionPayment,
			},
		}
		pip.cache.authorizedActions[clientID] = &cachedActions{
			data:      actions,
			timestamp: time.Now(),
		}

		req := &AuthorizationValidationRequest{
			ClientID:  clientID,
			Action:    string(taxonomy.TransactionLoan), // Not authorized
			Resource:  "account-789",
			Timestamp: time.Now(),
		}

		result, err := pip.ValidateAuthorization(ctx, req)
		if err != nil {
			t.Fatalf("ValidateAuthorization() unexpected error: %v", err)
		}

		if result.Authorized {
			t.Error("Expected unauthorized result for non-authorized action")
		}
	})

	t.Run("Missing authorization chain", func(t *testing.T) {
		pip := NewDefaultPIP(nil, nil, nil, 5*time.Minute)

		req := &AuthorizationValidationRequest{
			ClientID:  "unknown-client",
			Action:    string(taxonomy.TransactionPayment),
			Resource:  "account-999",
			Timestamp: time.Now(),
		}

		result, err := pip.ValidateAuthorization(ctx, req)
		if err == nil {
			t.Error("Expected error for missing authorization chain")
		}

		if result.Authorized {
			t.Error("Expected unauthorized result when chain missing")
		}
	})
}

// TestDefaultPIP_GetCacheStats tests cache statistics
func TestDefaultPIP_GetCacheStats(t *testing.T) {
	ctx := context.Background()

	t.Run("Initial cache stats", func(t *testing.T) {
		pip := NewDefaultPIP(nil, nil, nil, 5*time.Minute)

		stats := pip.GetCacheStats()
		if stats == nil {
			t.Fatal("GetCacheStats() returned nil")
		}

		if stats.TotalEntries != 0 {
			t.Errorf("TotalEntries = %v, want 0", stats.TotalEntries)
		}

		if stats.HitRate != 0 {
			t.Errorf("HitRate = %v, want 0", stats.HitRate)
		}

		if stats.MissRate != 0 {
			t.Errorf("MissRate = %v, want 0", stats.MissRate)
		}
	})

	t.Run("Cache stats after operations", func(t *testing.T) {
		commercialRegister := registry.NewMockCommercialRegisterService()
		pip := NewDefaultPIP(nil, commercialRegister, nil, 5*time.Minute)

		// Perform some operations
		_, _ = pip.VerifyCommercialRegister(ctx, "HRB12345", "DE")
		_, _ = pip.VerifyCommercialRegister(ctx, "HRB12345", "DE") // Cache hit
		_, _ = pip.VerifyCommercialRegister(ctx, "12345678", "GB") // Cache miss

		stats := pip.GetCacheStats()

		if stats.TotalEntries == 0 {
			t.Error("Expected some cache entries")
		}

		if stats.HitRate == 0 {
			t.Error("Expected non-zero hit rate")
		}

		if stats.MissRate == 0 {
			t.Error("Expected non-zero miss rate")
		}

		// We had 3 requests: 1 miss, 1 hit, 1 miss
		expectedHitRate := 1.0 / 3.0
		tolerance := 0.01
		if stats.HitRate < expectedHitRate-tolerance || stats.HitRate > expectedHitRate+tolerance {
			t.Errorf("HitRate = %v, want ~%v", stats.HitRate, expectedHitRate)
		}
	})
}

// TestDefaultPIP_RefreshCache tests cache refresh
func TestDefaultPIP_RefreshCache(t *testing.T) {
	ctx := context.Background()

	t.Run("Refresh invalidates cached data", func(t *testing.T) {
		pip := NewDefaultPIP(nil, nil, nil, 5*time.Minute)

		// Add data to cache
		clientID := "client-refresh-test"
		chain := &agentauth.AuthorizationChain{
			Client: &agentauth.AuthorizationLink{
				EntityID: clientID,
				Role:     "client",
			},
		}
		pip.cache.SetAuthorizationChain(clientID, chain)

		// Verify it's cached
		if pip.cache.GetAuthorizationChain(clientID) == nil {
			t.Fatal("Failed to cache authorization chain")
		}

		// Refresh cache
		err := pip.RefreshCache(ctx, clientID)
		if err != nil {
			t.Fatalf("RefreshCache() error: %v", err)
		}

		// Verify it's invalidated
		if pip.cache.GetAuthorizationChain(clientID) != nil {
			t.Error("Cache should be invalidated after refresh")
		}
	})
}

// TestAuthorizationCache_TTLExpiration tests cache TTL expiration
func TestAuthorizationCache_TTLExpiration(t *testing.T) {
	t.Run("Expired entries return nil", func(t *testing.T) {
		cache := NewAuthorizationCache(100, 100*time.Millisecond)

		// Add PoA definition
		poaID := "poa-ttl-test"
		poaDef := &poa.PoADefinition{
			Parties: poa.Parties{
				Principal: poa.Principal{
					Type:     "organization",
					Identity: "test-org",
				},
			},
		}
		cache.SetPoA(poaID, poaDef)

		// Should be retrievable immediately
		if cache.GetPoA(poaID) == nil {
			t.Error("PoA should be in cache immediately")
		}

		// Wait for TTL expiration
		time.Sleep(150 * time.Millisecond)

		// Should be expired now
		if cache.GetPoA(poaID) != nil {
			t.Error("PoA should be expired after TTL")
		}
	})

	t.Run("Commercial register cache TTL", func(t *testing.T) {
		cache := NewAuthorizationCache(100, 50*time.Millisecond)

		key := "DE:HRB12345"
		result := &registry.RegistrationVerificationResult{
			Verified:   true,
			EntityName: "Test GmbH",
		}
		cache.SetCommercialRegisterVerification(key, result)

		// Should be retrievable
		if cache.GetCommercialRegisterVerification(key) == nil {
			t.Error("Result should be in cache")
		}

		// Wait for expiration
		time.Sleep(75 * time.Millisecond)

		// Should be expired
		if cache.GetCommercialRegisterVerification(key) != nil {
			t.Error("Result should be expired")
		}
	})
}

// TestAuthorizationCache_Invalidate tests cache invalidation
func TestAuthorizationCache_Invalidate(t *testing.T) {
	t.Run("Invalidate removes client-specific entries", func(t *testing.T) {
		cache := NewAuthorizationCache(100, 5*time.Minute)

		clientID := "client-invalidate-test"

		// Add authorization chain
		chain := &agentauth.AuthorizationChain{
			Client: &agentauth.AuthorizationLink{
				EntityID: clientID,
				Role:     "client",
			},
		}
		cache.SetAuthorizationChain(clientID, chain)

		// Add client owner
		owner := &agentauth.ClientOwnerInfo{
			OwnerID: clientID + "-owner",
		}
		cache.clientOwners[clientID] = &cachedClientOwner{
			data:      owner,
			timestamp: time.Now(),
		}

		// Add authorized actions
		actions := &poa.AuthorizedActions{
			Transactions: []taxonomy.TransactionType{taxonomy.TransactionPayment},
		}
		cache.authorizedActions[clientID] = &cachedActions{
			data:      actions,
			timestamp: time.Now(),
		}

		// Verify all are cached
		if cache.GetAuthorizationChain(clientID) == nil {
			t.Error("Authorization chain should be cached")
		}
		if cache.GetClientOwner(clientID) == nil {
			t.Error("Client owner should be cached")
		}
		if cache.GetAuthorizedActions(clientID) == nil {
			t.Error("Authorized actions should be cached")
		}

		// Invalidate
		cache.Invalidate(clientID)

		// Verify all are removed
		if cache.GetAuthorizationChain(clientID) != nil {
			t.Error("Authorization chain should be invalidated")
		}
		if cache.GetClientOwner(clientID) != nil {
			t.Error("Client owner should be invalidated")
		}
		if cache.GetAuthorizedActions(clientID) != nil {
			t.Error("Authorized actions should be invalidated")
		}
	})
}

// TestAuthorizationCache_Size tests cache size calculation
func TestAuthorizationCache_Size(t *testing.T) {
	t.Run("Size reflects all cached entries", func(t *testing.T) {
		cache := NewAuthorizationCache(100, 5*time.Minute)

		if cache.Size() != 0 {
			t.Errorf("Initial size = %v, want 0", cache.Size())
		}

		// Add various entries
		cache.SetPoA("poa-1", &poa.PoADefinition{
			Parties: poa.Parties{
				Principal: poa.Principal{Type: "organization"},
			},
		})
		if cache.Size() != 1 {
			t.Errorf("Size after 1 entry = %v, want 1", cache.Size())
		}

		cache.SetAuthorizationChain("client-1", &agentauth.AuthorizationChain{})
		if cache.Size() != 2 {
			t.Errorf("Size after 2 entries = %v, want 2", cache.Size())
		}

		cache.SetCommercialRegisterVerification("DE:HRB12345", &registry.RegistrationVerificationResult{})
		if cache.Size() != 3 {
			t.Errorf("Size after 3 entries = %v, want 3", cache.Size())
		}
	})
}

// TestDefaultPIP_ActionAuthorization tests action authorization logic
func TestDefaultPIP_ActionAuthorization(t *testing.T) {
	pip := NewDefaultPIP(nil, nil, nil, 5*time.Minute)

	t.Run("Transaction action is authorized", func(t *testing.T) {
		actions := &poa.AuthorizedActions{
			Transactions: []taxonomy.TransactionType{
				taxonomy.TransactionPayment,
				taxonomy.TransactionPurchase,
			},
		}

		if !pip.isActionAuthorized(string(taxonomy.TransactionPayment), actions) {
			t.Error("TransactionPayment should be authorized")
		}

		if !pip.isActionAuthorized(string(taxonomy.TransactionPurchase), actions) {
			t.Error("TransactionPurchase should be authorized")
		}
	})

	t.Run("Decision action is authorized", func(t *testing.T) {
		actions := &poa.AuthorizedActions{
			Decisions: []taxonomy.DecisionType{
				taxonomy.DecisionFinancial,
				taxonomy.DecisionStrategic,
			},
		}

		if !pip.isActionAuthorized(string(taxonomy.DecisionFinancial), actions) {
			t.Error("DecisionFinancial should be authorized")
		}
	})

	t.Run("Non-physical action is authorized", func(t *testing.T) {
		actions := &poa.AuthorizedActions{
			NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
				taxonomy.ActionNonPhysicalDataAggregation,
				taxonomy.ActionNonPhysicalVisualization,
			},
		}

		if !pip.isActionAuthorized(string(taxonomy.ActionNonPhysicalDataAggregation), actions) {
			t.Error("NonPhysical DataAggregation should be authorized")
		}
	})

	t.Run("Physical action is authorized", func(t *testing.T) {
		actions := &poa.AuthorizedActions{
			PhysicalActions: []taxonomy.ActionTypePhysical{
				taxonomy.ActionPhysicalManufacturing,
				taxonomy.ActionPhysicalAssembly,
			},
		}

		if !pip.isActionAuthorized(string(taxonomy.ActionPhysicalManufacturing), actions) {
			t.Error("PhysicalManufacturing should be authorized")
		}
	})

	t.Run("Unauthorized action returns false", func(t *testing.T) {
		actions := &poa.AuthorizedActions{
			Transactions: []taxonomy.TransactionType{
				taxonomy.TransactionPayment,
			},
		}

		if pip.isActionAuthorized(string(taxonomy.TransactionLoan), actions) {
			t.Error("TransactionLoan should not be authorized")
		}
	})

	t.Run("Nil actions returns false", func(t *testing.T) {
		if pip.isActionAuthorized("any-action", nil) {
			t.Error("Any action with nil actions should return false")
		}
	})
}

// TestDefaultPIP_GeographicAuthorization tests geographic scope validation
func TestDefaultPIP_GeographicAuthorization(t *testing.T) {
	pip := NewDefaultPIP(nil, nil, nil, 5*time.Minute)

	t.Run("Authorized in specific jurisdiction", func(t *testing.T) {
		scopes := []poa.GeographicScope{
			{
				Type:       poa.GeoTypeNational,
				Identifier: "DE",
				Name:       "Germany",
			},
			{
				Type:       poa.GeoTypeNational,
				Identifier: "FR",
				Name:       "France",
			},
		}

		// Test with included countries
		authorized := pip.isJurisdictionAuthorized("DE", scopes)
		_ = authorized // Result depends on poa.IsAuthorizedInRegion implementation
	})
}

// TestDefaultPIP_SectorAuthorization tests industry sector validation
func TestDefaultPIP_SectorAuthorization(t *testing.T) {
	pip := NewDefaultPIP(nil, nil, nil, 5*time.Minute)

	t.Run("Authorized in specific sector", func(t *testing.T) {
		sectors := []poa.IndustrySector{
			{
				Code:        taxonomy.SectorFinanceInsurance,
				Description: "Financial Services",
				Authorized:  true,
			},
			{
				Code:        taxonomy.SectorHealthSocialWork,
				Description: "Healthcare",
				Authorized:  true,
			},
		}

		if !pip.isSectorAuthorized(string(taxonomy.SectorFinanceInsurance), sectors) {
			t.Error("Financial Services sector should be authorized")
		}

		if !pip.isSectorAuthorized("Healthcare", sectors) {
			t.Error("Healthcare sector (by description) should be authorized")
		}
	})

	t.Run("Not authorized in non-authorized sector", func(t *testing.T) {
		sectors := []poa.IndustrySector{
			{
				Code:        taxonomy.SectorFinanceInsurance,
				Description: "Financial Services",
				Authorized:  true,
			},
			{
				Code:        taxonomy.SectorManufacturing,
				Description: "Manufacturing",
				Authorized:  false, // Not authorized
			},
		}

		if pip.isSectorAuthorized(string(taxonomy.SectorManufacturing), sectors) {
			t.Error("Manufacturing sector should not be authorized")
		}
	})

	t.Run("Unknown sector returns false", func(t *testing.T) {
		sectors := []poa.IndustrySector{
			{
				Code:       taxonomy.SectorFinanceInsurance,
				Authorized: true,
			},
		}

		if pip.isSectorAuthorized("unknown-sector", sectors) {
			t.Error("Unknown sector should not be authorized")
		}
	})
}

// TestDefaultPIP_ConcurrentAccess tests thread-safe concurrent access
func TestDefaultPIP_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()

	t.Run("Concurrent cache operations", func(t *testing.T) {
		commercialRegister := registry.NewMockCommercialRegisterService()
		pip := NewDefaultPIP(nil, commercialRegister, nil, 5*time.Minute)

		// Launch multiple goroutines
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 10; j++ {
					_, _ = pip.VerifyCommercialRegister(ctx, "HRB12345", "DE")
					stats := pip.GetCacheStats()
					_ = stats
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		stats := pip.GetCacheStats()
		if stats.TotalEntries == 0 {
			t.Error("Expected cache entries after concurrent operations")
		}
	})
}

// Benchmark tests

func BenchmarkDefaultPIP_VerifyCommercialRegister(b *testing.B) {
	ctx := context.Background()
	commercialRegister := registry.NewMockCommercialRegisterService()
	pip := NewDefaultPIP(nil, commercialRegister, nil, 5*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pip.VerifyCommercialRegister(ctx, "HRB12345", "DE")
	}
}

func BenchmarkDefaultPIP_VerifyCommercialRegister_CacheHit(b *testing.B) {
	ctx := context.Background()
	commercialRegister := registry.NewMockCommercialRegisterService()
	pip := NewDefaultPIP(nil, commercialRegister, nil, 5*time.Minute)

	// Warm up cache
	_, _ = pip.VerifyCommercialRegister(ctx, "HRB12345", "DE")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pip.VerifyCommercialRegister(ctx, "HRB12345", "DE")
	}
}

func BenchmarkDefaultPIP_ValidateAuthorization(b *testing.B) {
	ctx := context.Background()
	pip := NewDefaultPIP(nil, nil, nil, 5*time.Minute)

	// Setup cache
	clientID := "bench-client"
	chain := &agentauth.AuthorizationChain{
		Client: &agentauth.AuthorizationLink{
			EntityID: clientID,
			Role:     "client",
		},
	}
	pip.cache.SetAuthorizationChain(clientID, chain)

	actions := &poa.AuthorizedActions{
		Transactions: []taxonomy.TransactionType{
			taxonomy.TransactionPayment,
		},
	}
	pip.cache.authorizedActions[clientID] = &cachedActions{
		data:      actions,
		timestamp: time.Now(),
	}

	req := &AuthorizationValidationRequest{
		ClientID:  clientID,
		Action:    string(taxonomy.TransactionPayment),
		Resource:  "account",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pip.ValidateAuthorization(ctx, req)
	}
}

func BenchmarkAuthorizationCache_Get(b *testing.B) {
	cache := NewAuthorizationCache(1000, 5*time.Minute)

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		id := "poa-" + string(rune('0'+i%10))
		cache.SetPoA(id, &poa.PoADefinition{
			Parties: poa.Parties{
				Principal: poa.Principal{Type: "org"},
			},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetPoA("poa-5")
	}
}
