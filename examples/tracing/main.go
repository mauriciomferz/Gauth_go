package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/Gimel-Foundation/gauth/internal/tracing"
	"github.com/Gimel-Foundation/gauth/pkg/auth"
	"github.com/Gimel-Foundation/gauth/pkg/authz"
	"github.com/Gimel-Foundation/gauth/pkg/rate"
)

func main() {
	// Initialize Authorizer (in-memory)
	var authorizer = authz.NewMemoryAuthorizer()

	// Add a default allow policy so all requests are authorized for demo purposes
	var err error
	err = authorizer.AddPolicy(context.Background(), &authz.Policy{
		ID:        "default-allow",
		Version:   "1.0",
		Name:      "Allow all",
		Effect:    authz.Allow,
		Subjects:  []authz.Subject{{ID: "*"}},
		Resources: []authz.Resource{{ID: "*"}},
		Actions:   []authz.Action{{Name: "*"}},
		Priority:  1,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		log.Fatalf("failed to add default policy: %v", err)
	}
	// Initialize tracer
	tracer, err := tracing.NewTracerProvider(tracing.Config{
		ServiceName:    "gauth-demo",
		ServiceVersion: "1.0.0",
		Environment:    "development",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer tracer.Shutdown(context.Background())

	// Initialize Authenticator (JWT)
	authService, err := auth.NewAuthenticator(auth.Config{
		Type:              auth.TypeJWT,
		AccessTokenExpiry: time.Hour,
	})
	if err != nil {
		log.Fatalf("failed to create authenticator: %v", err)
	}

	// Initialize Rate Limiter
	rateLimiter := rate.NewTokenBucket(rate.Config{
		Rate:      100,
		Window:    time.Minute,
		BurstSize: 100,
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, rootSpan := tracer.StartSpan(r.Context(), "handle_request",
			attribute.String("path", r.URL.Path),
			attribute.String("method", r.Method),
		)
		defer rootSpan.End()

		clientIP := r.RemoteAddr
		userID := r.Header.Get("X-User-ID")
		token := r.Header.Get("Authorization")

		// 1. Rate limiting
		ctx, rateSpan := tracer.StartSpan(ctx, "rate_limit_check")
		err := rateLimiter.Allow(ctx, clientIP)
		remaining := rateLimiter.GetRemainingRequests(clientIP)
		if err != nil {
			rateSpan.SetAttributes(
				attribute.String("error", err.Error()),
				attribute.Int64("remaining", remaining),
			)
			rateSpan.End()
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		rateSpan.SetAttributes(attribute.Int64("remaining", remaining))
		rateSpan.End()

		// 2. Authentication
		ctx, authSpan := tracer.StartSpan(ctx, "authenticate")
		claims, err := authService.ValidateToken(ctx, token)
		if err != nil {
			authSpan.SetAttributes(attribute.String("error", err.Error()))
			authSpan.End()
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		authSpan.SetAttributes(
			attribute.String("user_id", claims.Subject),
		)
		authSpan.End()

		// 3. Authorization
		ctx, authzSpan := tracer.StartSpan(ctx, "authorize")
		accessReq := authz.NewAccessRequest(
			authz.Subject{ID: userID},
			authz.Resource{ID: r.URL.Path},
			authz.Action{Name: r.Method},
		)
		decision, err := authorizer.Authorize(ctx, accessReq.Subject, accessReq.Action, accessReq.Resource)
		if err != nil {
			authzSpan.SetAttributes(attribute.String("error", err.Error()))
			authzSpan.End()
			http.Error(w, "Authorization error", http.StatusInternalServerError)
			return
		}
		if !decision.Allowed {
			authzSpan.SetAttributes(attribute.Bool("allowed", false))
			authzSpan.End()
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		authzSpan.SetAttributes(attribute.Bool("allowed", true))
		authzSpan.End()

		// Process request
		ctx, processSpan := tracer.StartSpan(ctx, "process_request")
		// ... process request ...
		processSpan.SetAttributes(
			attribute.String("status", "success"),
			attribute.String("operation", "example"),
		)
		processSpan.End()

		fmt.Fprintf(w, "Request processed successfully\n")
	})

	// Add trace ID middleware
	traced := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.StartSpan(r.Context(), "request")
			defer span.End()

			traceID := tracing.TraceID(ctx)
			w.Header().Set("X-Trace-ID", traceID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Start server with timeouts
	http.Handle("/api", traced(handler))
	
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	log.Println("Starting server on :8080...")
	log.Fatal(server.ListenAndServe())
}
