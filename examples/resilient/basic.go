package resilient

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/circuit"
	"github.com/mauriciomferz/AgentAuth/internal/monitoring"
	"github.com/mauriciomferz/AgentAuth/pkg/gauth"
)

// ResilientService combines circuit breaker and monitoring
type ResilientService struct {
	auth    *gauth.Service
	server  *gauth.ResourceServer
	breaker *circuit.Breaker
	metrics *monitoring.DefaultMetricsCollector
}

func NewResilientService(auth *gauth.Service) *ResilientService {
	metrics := monitoring.NewMetricsCollector()
	breaker := circuit.NewBreaker(circuit.Options{
		Name:             "auth-service",
		FailureThreshold: 5,
		ResetTimeout:     10 * time.Second,
		HalfOpenLimit:    2,
	})
	return &ResilientService{
		auth:    auth,
		server:  gauth.NewResourceServer("resilient-service", auth),
		breaker: breaker,
		metrics: metrics,
	}
}

func (s *ResilientService) ProcessRequest(tx gauth.TransactionDetails, token string) error {
	return s.breaker.Execute(context.Background(), func() error {
		start := time.Now()

		result, err := s.server.ProcessTransaction(tx, token)
		duration := time.Since(start).Seconds()

		txType := fmt.Sprintf("%v", tx.Type)
		labels := map[string]string{
			"type": txType,
		}
		if err == nil {
			labels["status"] = "success"
			s.metrics.IncrementWithLabels("transactions_total", labels)
			s.metrics.GaugeWithLabels("response_time_seconds", duration, labels)
			log.Printf("Transaction processed successfully: %v", result)
			return nil
		} else {
			labels["status"] = "error"
			s.metrics.IncrementWithLabels("transactions_total", labels)
			s.metrics.IncrementWithLabels("transaction_errors_total", labels)
			return err
		}
	})
}

// runBasicExample demonstrates basic resilient service usage
//
//nolint:unused // Example function for documentation purposes
func runBasicExample() {
	// This is a stub for demonstration. See NewResilientService for usage.
}
