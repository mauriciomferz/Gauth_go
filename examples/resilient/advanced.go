package resilient

import (
	"context"
	"fmt"
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
		tracing.AttributeTransactionType.String(tx.Type),
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

func main() {
	// Initialize GAuth
	config := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "resilient-client",
		ClientSecret:      "resilient-secret",
		Scopes:            []string{"transaction:execute"},
		AccessTokenExpiry: time.Hour,
	}

	auth, err := gauth.New(config)
	if err != nil {
		log.Fatalf("Failed to initialize GAuth: %v", err)
	}

	// Create highly resilient service
	service, err := NewHighlyResilientService(auth)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Get authorization and token
	ctx := context.Background()
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

	// Process multiple requests with full resilience
	for i := 0; i < 20; i++ {
		tx := gauth.TransactionDetails{
			Type:   "payment",
			Amount: float64(100 + i),
			Metadata: map[string]string{
				"request_id": fmt.Sprintf("req-%d", i),
			},
		}

		err := service.ProcessRequest(ctx, tx, tokenResp.Token)
		if err != nil {
			log.Printf("Request %d failed: %v", i, err)
			time.Sleep(time.Second)
			continue
		}
		log.Printf("Request %d succeeded", i)
	}

	// Graceful shutdown
	if err := service.tracer.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down tracer: %v", err)
	}
}
