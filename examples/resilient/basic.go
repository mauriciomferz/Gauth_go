package resilient

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
	"github.com/Gimel-Foundation/gauth/internal/monitoring"
	"github.com/Gimel-Foundation/gauth/internal/monitoring/prometheus"
	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

// ResilientService combines circuit breaker and monitoring
type ResilientService struct {
	auth         *gauth.GAuth
	server       *gauth.ResourceServer
	breaker      *circuit.CircuitBreaker
	metrics      *monitoring.MetricsCollector
	promExporter *prometheus.PrometheusExporter
}

func NewResilientService(auth *gauth.GAuth) *ResilientService {
	metrics := monitoring.NewMetricsCollector()

	service := &ResilientService{
		auth:         auth,
		server:       gauth.NewResourceServer("resilient-service", auth),
		metrics:      metrics,
		promExporter: prometheus.NewPrometheusExporter(metrics),
		breaker: circuit.NewCircuitBreaker(circuit.Options{
			Name:             "auth-service",
			FailureThreshold: 5,
			ResetTimeout:     10 * time.Second,
			HalfOpenLimit:    2,
			OnStateChange: func(name string, from, to circuit.State) {
				log.Printf("Circuit state changed from %v to %v", from, to)
				metrics.Counter("circuit_state_changes_total", 1, map[string]string{
					"name": name,
					"from": from.String(),
					"to":   to.String(),
				})
			},
		}),
	}

	// Start metrics export
	go service.exportMetrics()

	return service
}

func (s *ResilientService) ProcessRequest(tx gauth.TransactionDetails, token string) error {
	return s.breaker.Execute(func() error {
		start := time.Now()

		result, err := s.server.ProcessTransaction(tx, token)

		duration := time.Since(start).Seconds()
		s.metrics.Counter(string(monitoring.MetricTransactions), 1, map[string]string{
			"type":   tx.Type,
			"status": map[bool]string{true: "success", false: "error"}[err == nil],
		})

		if err == nil {
			s.metrics.Gauge(string(monitoring.MetricResponseTime), duration, map[string]string{
				"type": tx.Type,
			})
		} else {
			s.metrics.Counter(string(monitoring.MetricTransactionErrors), 1, map[string]string{
				"type":  tx.Type,
				"error": err.Error(),
			})
			return err
		}

		log.Printf("Transaction processed successfully: %v", result)
		return nil
	})
}

func (s *ResilientService) exportMetrics() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.promExporter.Export()
	}
}

func runBasicExample() {
	var (
		config = gauth.Config{
			AuthServerURL:     "https://auth.example.com",
			ClientID:          "resilient-client",
			ClientSecret:      "resilient-secret",
			Scopes:            []string{"transaction:execute"},
			AccessTokenExpiry: time.Hour,
		}
		err error
	)

	auth, err := gauth.New(config)
	if err != nil {
		log.Fatalf("Failed to initialize GAuth: %v", err)
	}

	// Create resilient service
	service := NewResilientService(auth)

	// Set up HTTP server with metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Start server
	go func() {
		log.Printf("Starting server on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Example usage
	tx := gauth.TransactionDetails{
		Type:   "payment",
		Amount: 100.0,
		Metadata: map[string]string{
			"customer_id": "cust-123",
		},
	}

	// Get a token
	authReq := gauth.AuthorizationRequest{
		ClientID:        "resilient-client",
		ClientOwnerID:   "owner-1",
		ResourceOwnerID: "resource-1",
		Scopes:          []string{"transaction:execute"},
	}

	grant, err := auth.InitiateAuthorization(authReq)
	if err != nil {
		log.Fatalf("Authorization failed: %v", err)
	}

	tokenResp, err := auth.RequestToken(gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   grant.Scope,
	})
	if err != nil {
		log.Fatalf("Token request failed: %v", err)
	}

	// Process requests with circuit breaker and monitoring
	for i := 0; i < 10; i++ {
		if err := service.ProcessRequest(tx, tokenResp.Token); err != nil {
			log.Printf("Request %d failed: %v", i, err)
			time.Sleep(time.Second)
			continue
		}
		log.Printf("Request %d succeeded", i)
	}

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gracefully...")
}
