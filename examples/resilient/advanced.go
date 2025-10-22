package resilient

import (
	"context"
	"log"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/tracing"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/resilience"
)

// HighlyResilientService combines tracing and multiple resilience patterns
type HighlyResilientService struct {
	auth      *gauth.GAuth
	server    *gauth.ResourceServer
	composite *resilience.Composite
	tracer    *tracing.TracerProvider
}

func NewHighlyResilientService(auth *gauth.GAuth) (*HighlyResilientService, error) {
	// Initialize tracer
	tracerProvider, _ := tracing.NewTracerProvider(tracing.Config{
		ServiceName:    "resilient-gauth",
		ServiceVersion: "1.0",
		Environment:    "production",
		OTLPEndpoint:   "localhost:4317",
	})

	// Configure resilience patterns (use available stub fields)
	composite := resilience.NewComposite(resilience.CompositeOptions{
		CircuitOptions: nil, // stub, as circuit.Options is not defined
		MaxConcurrent:  100,
		RetryStrategy: resilience.RetryStrategy{
			MaxAttempts:     3,
			InitialInterval: time.Second,
			MaxInterval:     5 * time.Second,
			Multiplier:      2.0,
		},
		RateLimit: 100,
		BurstSize: 20,
	})

	// gauth.NewResourceServer expects (name string, service *Service)
	return &HighlyResilientService{
		auth:      auth,
		server:    gauth.NewResourceServer("resilient-service", nil),
		composite: composite,
		tracer:    tracerProvider,
	}, nil
}

func (s *HighlyResilientService) ProcessRequest(ctx context.Context, tx gauth.TransactionDetails, token string) error {
	// Start tracing span
	ctx, span := s.tracer.StartSpan(ctx, tracing.SpanTransaction,
		tracing.AttributeTransactionType.String(string(tx.Type)),
		tracing.AttributeResourceID.String(tx.ResourceID),
	)
	defer span.End()

	// Execute with resilience patterns
	err := s.composite.Execute(ctx, func() error {
		result, err := s.server.ProcessTransaction(tx, token)
		if err != nil {
			tracing.AddEvent(span, "transaction_error",
				tracing.AttributeError.String(err.Error()))
			return err
		}

		tracing.AddEvent(span, "transaction_success",
			tracing.Attribute{Key: "amount", Value: int64(tx.Amount)})

		log.Printf("Transaction processed: %v", result)
		return nil
	})
	if err != nil {
		span.SetStatus(tracing.StatusError, err.Error())
		return err
	}

	span.SetStatus(tracing.StatusOK, "")
	return nil
}
