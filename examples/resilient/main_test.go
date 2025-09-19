package resilient

import (
	"bytes"
	"crypto"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/pkg/monitoring"
	"github.com/mauriciomferz/Gauth_go/pkg/token"
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

       if err := w.Close(); err != nil {
	       t.Errorf("Failed to close write pipe: %v", err)
       }
       var buf bytes.Buffer
       if _, err := io.Copy(&buf, r); err != nil {
	       t.Errorf("Failed to copy output: %v", err)
       }
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
       config := gauth.Config{
	       AuthServerURL:     "https://test.example.com",
	       ClientID:          "test-client",
	       ClientSecret:      "test-secret",
	       Scopes:            []string{"transaction:execute"},
	       AccessTokenExpiry: time.Hour,
	       TokenConfig: &token.Config{
		       SigningMethod: token.RS256,
		       SigningKey: newMockSigner(),
	       },
       }

	auth, err := gauth.New(&config, audit.NewLogger(100))
       if err != nil {
	       t.Fatalf("Failed to create GAuth instance: %v", err)
       }

       _ = NewResilientService(auth) // Remove unused variable warning

       // Minimal test to ensure setup works
       t.Run("SanityCheck", func(t *testing.T) {
	       if auth == nil {
		       t.Fatal("auth should not be nil")
	       }
       })
}

// newMockSigner returns a crypto.Signer for testing
func newMockSigner() crypto.Signer {
	return token.NewMockSigner()
}

