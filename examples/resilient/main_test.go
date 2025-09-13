package resilient

import (
	"fmt"
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
	"github.com/Gimel-Foundation/gauth/internal/monitoring"
	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

// Helper functions for testing metrics
func getMetricValue(metrics map[string]monitoring.Metric, name string, labels map[string]string) float64 {
	for _, metric := range metrics {
		if metric.Name == name && matchLabels(metric.Labels, labels) {
			return metric.Value
		}
	}
	return 0
}

func hasMetric(metrics map[string]monitoring.Metric, name string, labels map[string]string) bool {
	for _, metric := range metrics {
		if metric.Name == name && matchLabels(metric.Labels, labels) {
			return true
		}
	}
	return false
}

func matchLabels(a, b map[string]string) bool {
	if len(b) == 0 {
		return true
	}
	for k, v := range b {
		if a[k] != v {
			return false
		}
	}
	return true
}

func TestResilientService(t *testing.T) {
	// Create a GAuth instance for testing
	config := gauth.Config{
		AuthServerURL:     "https://test.example.com",
		ClientID:          "test-client",
		ClientSecret:      "test-secret",
		Scopes:            []string{"transaction:execute"},
		AccessTokenExpiry: time.Hour,
	}

	auth, err := gauth.New(config)
	if err != nil {
		t.Fatalf("Failed to create GAuth instance: %v", err)
	}

	service := NewResilientService(auth)

	// Test successful transaction
	t.Run("SuccessfulTransaction", func(t *testing.T) {
		tx := gauth.TransactionDetails{
			Type:   "payment",
			Amount: 100.0,
			Metadata: map[string]string{
				"test": "true",
			},
		}

		// Get a test token
		grant, err := auth.InitiateAuthorization(gauth.AuthorizationRequest{
			ClientID:        "test-client",
			ClientOwnerID:   "test-owner",
			ResourceOwnerID: "test-resource",
			Scopes:          []string{"transaction:execute"},
		})
		if err != nil {
			t.Fatalf("Failed to initiate authorization: %v", err)
		}

		tokenResp, err := auth.RequestToken(gauth.TokenRequest{
			GrantID: grant.GrantID,
			Scope:   grant.Scope,
		})
		if err != nil {
			t.Fatalf("Failed to request token: %v", err)
		}

		err = service.ProcessRequest(tx, tokenResp.Token)
		if err != nil {
			t.Errorf("Expected successful transaction, got error: %v", err)
		}

		// Verify metrics
		metrics := service.metrics.GetAllMetrics()
		successCount := getMetricValue(metrics, "transactions_total", map[string]string{
			"type":   "payment",
			"status": "success",
		})
		if successCount != 1 {
			t.Errorf("Expected 1 successful transaction, got %.0f", successCount)
		}
	})

	// Test circuit breaker behavior
	t.Run("CircuitBreakerTrip", func(t *testing.T) {
		tx := gauth.TransactionDetails{
			ID:     "test-tx-2",
			Type:   "failing",
			Amount: 100,
		}

		// Force multiple failures
		failureCount := 0
		for i := 0; i < 10; i++ {
			err := service.ProcessRequest(tx, "invalid-token")
			if err != nil {
				failureCount++
			}
		}

		if failureCount < 5 {
			t.Errorf("Expected at least 5 failures before circuit opens")
		}

		// Verify circuit breaker state
		if service.breaker.GetState() != circuit.StateOpen {
			t.Error("Expected circuit breaker to be open")
		}

		// Wait for reset timeout
		time.Sleep(11 * time.Second)

		// Try a successful request with a proper token
		tx.Type = "payment"
		grant, err := auth.InitiateAuthorization(gauth.AuthorizationRequest{
			ClientID:        "test-client",
			ClientOwnerID:   "test-owner",
			ResourceOwnerID: "test-resource",
			Scopes:          []string{"transaction:execute"},
		})
		if err != nil {
			t.Fatalf("Failed to initiate authorization: %v", err)
		}

		tokenResp, err := auth.RequestToken(gauth.TokenRequest{
			GrantID: grant.GrantID,
			Scope:   grant.Scope,
		})
		if err != nil {
			t.Fatalf("Failed to request token: %v", err)
		}

		err = service.ProcessRequest(tx, tokenResp.Token)
		if err != nil {
			t.Error("Expected success after circuit reset")
		}
	})

	// Test metrics collection
	t.Run("MetricsCollection", func(t *testing.T) {
		// Reset metrics
		service.metrics = monitoring.NewMetricsCollector()

		// Get a valid token
		grant, err := auth.InitiateAuthorization(gauth.AuthorizationRequest{
			ClientID:        "test-client",
			ClientOwnerID:   "test-owner",
			ResourceOwnerID: "test-resource",
			Scopes:          []string{"transaction:execute"},
		})
		if err != nil {
			t.Fatalf("Failed to initiate authorization: %v", err)
		}

		tokenResp, err := auth.RequestToken(gauth.TokenRequest{
			GrantID: grant.GrantID,
			Scope:   grant.Scope,
		})
		if err != nil {
			t.Fatalf("Failed to request token: %v", err)
		}

		// Perform mixed transactions
		transactions := []struct {
			tx    gauth.TransactionDetails
			token string
		}{
			{
				tx: gauth.TransactionDetails{
					Type:     "payment",
					Amount:   100,
					Metadata: map[string]string{"test": "1"},
				},
				token: tokenResp.Token,
			},
			{
				tx: gauth.TransactionDetails{
					Type:     "refund",
					Amount:   50,
					Metadata: map[string]string{"test": "2"},
				},
				token: "invalid-token",
			},
			{
				tx: gauth.TransactionDetails{
					Type:     "payment",
					Amount:   75,
					Metadata: map[string]string{"test": "3"},
				},
				token: tokenResp.Token,
			},
		}

		for _, tc := range transactions {
			service.ProcessRequest(tc.tx, tc.token)
		}

		metrics := service.metrics.GetAllMetrics()

		// Verify transaction counts
		paymentSuccess := getMetricValue(metrics, string(monitoring.MetricTransactions), map[string]string{
			"type":   "payment",
			"status": "success",
		})
		if paymentSuccess != 2 {
			t.Errorf("Expected 2 successful payment transactions, got %.0f", paymentSuccess)
		}

		refundError := getMetricValue(metrics, string(monitoring.MetricTransactionErrors), map[string]string{
			"type": "refund",
		})
		if refundError != 1 {
			t.Errorf("Expected 1 failed refund transaction, got %.0f", refundError)
		}

		// Verify response time metrics exist
		if !hasMetric(metrics, string(monitoring.MetricResponseTime), map[string]string{"type": "payment"}) {
			t.Error("Expected response time metrics to be present")
		}
	})

	// Test concurrent requests
	t.Run("ConcurrentRequests", func(t *testing.T) {
		const numRequests = 100
		errors := make(chan error, numRequests)
		start := time.Now()

		// Get a valid token for all requests
		grant, err := auth.InitiateAuthorization(gauth.AuthorizationRequest{
			ClientID:        "test-client",
			ClientOwnerID:   "test-owner",
			ResourceOwnerID: "test-resource",
			Scopes:          []string{"transaction:execute"},
		})
		if err != nil {
			t.Fatalf("Failed to initiate authorization: %v", err)
		}

		tokenResp, err := auth.RequestToken(gauth.TokenRequest{
			GrantID: grant.GrantID,
			Scope:   grant.Scope,
		})
		if err != nil {
			t.Fatalf("Failed to request token: %v", err)
		}

		for i := 0; i < numRequests; i++ {
			go func(id int) {
				tx := gauth.TransactionDetails{
					Type:   "payment",
					Amount: float64(id),
					Metadata: map[string]string{
						"concurrent": "true",
						"id":         fmt.Sprintf("%d", id),
					},
				}
				errors <- service.ProcessRequest(tx, tokenResp.Token)
			}(i)
		}

		// Collect errors
		var errorCount int
		for i := 0; i < numRequests; i++ {
			if err := <-errors; err != nil {
				errorCount++
			}
		}

		duration := time.Since(start)
		t.Logf("Processed %d concurrent requests in %v with %d errors",
			numRequests, duration, errorCount)

		if errorCount > 50 {
			t.Errorf("Too many errors: %d", errorCount)
		}

		// Verify metrics under load
		metrics := service.metrics.GetAllMetrics()
		if !hasMetric(metrics, string(monitoring.MetricResponseTime), map[string]string{"type": "payment"}) {
			t.Error("Expected response time metrics under load")
		}
	})
}

func labelsMatch(a, b map[string]string) bool {
	if len(b) == 0 {
		return true
	}
	for k, v := range b {
		if a[k] != v {
			return false
		}
	}
	return true
}
