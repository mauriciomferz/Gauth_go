package resilient

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/circuit"
	"github.com/mauriciomferz/Gauth_go/internal/monitoring"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
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

	if !strings.Contains(output, "Resilience demo consolidated") {
		t.Errorf("Expected consolidated resilience demo notice, got: %s", output)
	}
}

func hasMetric(metrics map[string]monitoring.MetricValue, name string, labels map[string]string) bool {
	mv, ok := metrics[name]
	if !ok {
		return false
	}
	return matchLabels(mv.Labels, labels)
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

	t.Run("SuccessfulTransaction", func(t *testing.T) {
		tx := gauth.TransactionDetails{Type: gauth.PaymentTransaction, Amount: 100.0, CustomMetadata: map[string]interface{}{"test": "true"}}
		grant, err := auth.InitiateAuthorization(gauth.AuthorizationRequest{ClientID: "test-client", Scopes: []string{"transaction:execute"}})
		if err != nil {
			t.Fatalf("grant error: %v", err)
		}
		tokenResp, err := auth.RequestToken(gauth.TokenRequest{GrantID: grant.GrantID, Scope: grant.Scope})
		if err != nil {
			t.Fatalf("token error: %v", err)
		}
		if err := service.ProcessRequest(tx, tokenResp.Token); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		metrics := service.metrics.GetAllMetrics()
		if mv, ok := metrics["transactions_total"]; !ok || mv.Value < 1 {
			t.Errorf("Expected transactions_total >=1 after success, got %.0f", mv.Value)
		}
	})

	t.Run("CircuitBreakerTrip", func(t *testing.T) {
		tx := gauth.TransactionDetails{ID: "test-tx-2", Type: gauth.PaymentTransaction, Amount: 100}
		failures := 0
		for i := 0; i < 6; i++ {
			if err := service.ProcessRequest(tx, "invalid-token"); err != nil {
				failures++
			}
		}
		if failures < 5 {
			t.Errorf("Expected at least 5 failures got %d", failures)
		}
		time.Sleep(100 * time.Millisecond)
		if service.breaker.GetState() != circuit.StateOpen {
			t.Errorf("expected circuit open, got %v", service.breaker.GetState())
		}
		time.Sleep(11 * time.Second)
		grant, _ := auth.InitiateAuthorization(gauth.AuthorizationRequest{ClientID: "test-client", Scopes: []string{"transaction:execute"}})
		tokenResp, _ := auth.RequestToken(gauth.TokenRequest{GrantID: grant.GrantID, Scope: grant.Scope})
		if err := service.ProcessRequest(tx, tokenResp.Token); err != nil {
			t.Error("expected success after reset")
		}
	})

	t.Run("MetricsCollection", func(t *testing.T) {
		service.metrics = monitoring.NewMetricsCollector()
		grant, _ := auth.InitiateAuthorization(gauth.AuthorizationRequest{ClientID: "test-client", Scopes: []string{"transaction:execute"}})
		tokenResp, _ := auth.RequestToken(gauth.TokenRequest{GrantID: grant.GrantID, Scope: grant.Scope})
		cases := []struct {
			tx    gauth.TransactionDetails
			token string
		}{
			{gauth.TransactionDetails{Type: gauth.PaymentTransaction, Amount: 100, CustomMetadata: map[string]interface{}{"test": "1"}}, tokenResp.Token},
			{gauth.TransactionDetails{Type: gauth.PaymentTransaction, Amount: 50, CustomMetadata: map[string]interface{}{"test": "2"}}, "invalid-token"},
			{gauth.TransactionDetails{Type: gauth.PaymentTransaction, Amount: 75, CustomMetadata: map[string]interface{}{"test": "3"}}, tokenResp.Token},
			{gauth.TransactionDetails{Type: "refund", Amount: 25, CustomMetadata: map[string]interface{}{"test": "refund"}}, "invalid-token"},
		}
		for _, c := range cases {
			_ = service.ProcessRequest(c.tx, c.token)
			time.Sleep(5 * time.Millisecond)
		}
		metrics := service.metrics.GetAllMetrics()
		for name, mv := range metrics {
			t.Logf("Metric: %s=%.2f labels=%+v", name, mv.Value, mv.Labels)
		}
		if mv, ok := metrics["transactions_total"]; !ok || mv.Value < 1 {
			t.Errorf("expected transactions_total to be incremented, got %.0f", mv.Value)
		}
		if !hasMetric(metrics, monitoring.MetricResponseTime, map[string]string{}) && len(metrics) == 0 {
			t.Logf("Warning: no response time metric present")
		}
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		const n = 50
		errs := make(chan error, n)
		grant, _ := auth.InitiateAuthorization(gauth.AuthorizationRequest{ClientID: "test-client", Scopes: []string{"transaction:execute"}})
		for i := 0; i < n; i++ {
			go func(id int) {
				// Issue a fresh token per goroutine to avoid replay detection failures skewing resilience measurement.
				tr, terr := auth.RequestToken(gauth.TokenRequest{GrantID: grant.GrantID, Scope: grant.Scope})
				if terr != nil {
					errs <- terr
					return
				}
				tx := gauth.TransactionDetails{Type: gauth.PaymentTransaction, Amount: float64(id + 1)}
				errs <- service.ProcessRequest(tx, tr.Token)
			}(i)
		}
		errorCount := 0
		for i := 0; i < n; i++ {
			if e := <-errs; e != nil {
				errorCount++
			}
		}
		if errorCount > n/2 {
			t.Errorf("too many errors: %d", errorCount)
		}
		metrics := service.metrics.GetAllMetrics()
		if !hasMetric(metrics, monitoring.MetricResponseTime, map[string]string{}) && len(metrics) == 0 {
			t.Log("Warning: missing response time metrics after concurrency test")
		}
	})
}
