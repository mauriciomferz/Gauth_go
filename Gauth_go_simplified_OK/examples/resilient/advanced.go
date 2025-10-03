package resilient

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
	"github.com/Gimel-Foundation/gauth/internal/resilience"
	"github.com/Gimel-Foundation/gauth/internal/tracing"
	"github.com/Gimel-Foundation/gauth/pkg/gauth"
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
	tracerProvider, err := tracing.NewTracerProvider(tracing.Config{
		ServiceName:    "resilient-gauth",
		ServiceVersion: "1.0",
		Environment:    "production",
		OTLPEndpoint:   "localhost:4317",
	})
	if err != nil {
		return nil, err
	}

	// Configure resilience patterns
	composite := resilience.NewComposite(resilience.CompositeOptions{
		CircuitOptions: circuit.Options{
			Name:             "auth-service",
			FailureThreshold: 5,
			ResetTimeout:     10 * time.Second,
			HalfOpenLimit:    2,
		},
		MaxConcurrent: 100,
		RetryStrategy: resilience.RetryStrategy{
			MaxAttempts:     3,
			InitialInterval: time.Second,
			MaxInterval:     5 * time.Second,
			Multiplier:      2.0,
		},
		RateLimit: 100, // requests per second
		BurstSize: 20,
	})

	return &HighlyResilientService{
		auth:      auth,
		server:    gauth.NewResourceServer("resilient-service", auth),
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
			attribute.Int64("amount", int64(tx.Amount)))

		log.Printf("Transaction processed: %v", result)
		return nil
	})

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
