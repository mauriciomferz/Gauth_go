package resilient

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
	"github.com/Gimel-Foundation/gauth/internal/monitoring"
	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

func TestMainDemoOutput(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = origStdout
		if r := recover(); r != nil {
			t.Errorf("MainDemo panicked: %v", r)
		}
	}()

	MainDemo()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Scenario 1: Token Bucket Rate Limiting") {
		t.Errorf("Expected output to contain 'Scenario 1: Token Bucket Rate Limiting', got: %s", output)
	}
	if !strings.Contains(output, "Scenario 2: Sliding Window Rate Limiting") {
		t.Errorf("Expected output to contain 'Scenario 2: Sliding Window Rate Limiting', got: %s", output)
	}
	if !strings.Contains(output, "Scenario 3: Circuit Breaker with Retry") {
		t.Errorf("Expected output to contain 'Scenario 3: Circuit Breaker with Retry', got: %s", output)
	}
	if !strings.Contains(output, "Scenario 4: All Patterns Combined") {
		t.Errorf("Expected output to contain 'Scenario 4: All Patterns Combined', got: %s", output)
	}
}

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
			Type:   gauth.PaymentTransaction,
			Amount: 100.0,
			CustomMetadata: map[string]string{
				"test": "true",
			},
		}

		// Get a test token
		grant, err := auth.InitiateAuthorization(gauth.AuthorizationRequest{
			ClientID: "test-client",
			Scopes:   []string{"transaction:execute"},
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
			Type:   gauth.PaymentTransaction,
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

		// Wait for circuit breaker reset timeout (10s + buffer for CI stability)
		resetTimeout := 11 * time.Second
		if os.Getenv("CI") == "true" {
			resetTimeout = 12 * time.Second // Extra buffer for CI environment
		}
		t.Logf("Waiting %v for circuit breaker reset...", resetTimeout)
		time.Sleep(resetTimeout)

		// Try a successful request with a proper token
		tx.Type = "payment"
		grant, err := auth.InitiateAuthorization(gauth.AuthorizationRequest{
			ClientID: "test-client",
			Scopes:   []string{"transaction:execute"},
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
		// Reset metrics and circuit breaker state
		freshService := NewResilientService(auth)

		// Get a valid token
		grant, err := auth.InitiateAuthorization(gauth.AuthorizationRequest{
			ClientID: "test-client",
			Scopes:   []string{"transaction:execute"},
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

		// Perform mixed transactions, including a failed refund
		transactions := []struct {
			tx    gauth.TransactionDetails
			token string
		}{
			{
				tx: gauth.TransactionDetails{
					Type:           gauth.PaymentTransaction,
					Amount:         100,
					CustomMetadata: map[string]string{"test": "1"},
				},
				token: tokenResp.Token,
			},
			{
				tx: gauth.TransactionDetails{
					Type:           gauth.PaymentTransaction,
					Amount:         50,
					CustomMetadata: map[string]string{"test": "2"},
				},
				token: "invalid-token",
			},
			{
				tx: gauth.TransactionDetails{
					Type:           gauth.PaymentTransaction,
					Amount:         75,
					CustomMetadata: map[string]string{"test": "3"},
				},
				token: tokenResp.Token,
			},
			{
				tx: gauth.TransactionDetails{
					Type:           gauth.RefundTransaction,
					Amount:         25,
					CustomMetadata: map[string]string{"test": "refund"},
				},
				token: "invalid-token",
			},
		}

		for _, tc := range transactions {
			_ = freshService.ProcessRequest(tc.tx, tc.token)
		}

		metrics := freshService.metrics.GetAllMetrics()

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
		// Create fresh service to avoid circuit breaker state from previous tests
		freshService := NewResilientService(auth)
		
		numRequests := 100
		if os.Getenv("CI") == "true" {
			numRequests = 50 // Reduce load for CI environment
		}
		errors := make(chan error, numRequests)
		start := time.Now()

		// Get a valid token for all requests
		grant, err := auth.InitiateAuthorization(gauth.AuthorizationRequest{
			ClientID: "test-client",
			Scopes:   []string{"transaction:execute"},
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
					Type:   gauth.PaymentTransaction,
					Amount: float64(id),
					CustomMetadata: map[string]string{
						"concurrent": "true",
						"id":         fmt.Sprintf("%d", id),
					},
				}
				errors <- freshService.ProcessRequest(tx, tokenResp.Token)
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

		// Allow up to 50% error rate (reasonable for resilience testing)
		maxErrors := numRequests / 2
		if errorCount > maxErrors {
			t.Errorf("Too many errors: %d (max allowed: %d out of %d requests)", errorCount, maxErrors, numRequests)
		}

		// Verify metrics under load
		metrics := freshService.metrics.GetAllMetrics()
		if !hasMetric(metrics, string(monitoring.MetricResponseTime), map[string]string{"type": "payment"}) {
			t.Error("Expected response time metrics under load")
		}
	})
}
