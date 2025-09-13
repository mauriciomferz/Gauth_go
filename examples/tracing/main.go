package tracing
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
	"github.com/Gimel-Foundation/gauth/pkg/authz"
	"github.com/Gimel-Foundation/gauth/pkg/rate"
	"github.com/Gimel-Foundation/gauth/internal/tracing"
)

func main() {
	// Initialize tracer
	tracer, err := tracing.NewTracerProvider(tracing.Config{
		ServiceName:    "gauth-demo",
		ServiceVersion: "1.0.0",
		Environment:   "development",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer tracer.Shutdown(context.Background())

	// Initialize services
	authService := auth.New(auth.Config{
		TokenType: auth.JWT,
		TTL:      time.Hour,
	})

	authzService := authz.New(authz.Config{
		PolicyStore: authz.NewMemoryPolicyStore(),
	})

	rateLimiter := rate.NewTokenBucket(rate.Config{
		Limit:  100,
		Window: time.Minute,
	})

	// Create handler with tracing
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Start root span
		ctx, rootSpan := tracer.StartSpan(r.Context(), "handle_request",
			attribute.String("path", r.URL.Path),
			attribute.String("method", r.Method),
		)
		defer rootSpan.End()

		// Extract client info
		clientIP := r.RemoteAddr
		userID := r.Header.Get("X-User-ID")
		token := r.Header.Get("Authorization")

		// 1. Rate limiting
		ctx, rateSpan := tracer.StartSpan(ctx, "rate_limit_check")
		quota, err := rateLimiter.Allow(ctx, clientIP)
		if err != nil {
			rateSpan.SetAttributes(
				attribute.String("error", err.Error()),
				attribute.Int64("remaining", quota.Remaining),
			)
			rateSpan.End()
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		rateSpan.SetAttributes(attribute.Int64("remaining", quota.Remaining))
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
			attribute.String("token_id", claims.ID),
		)
		authSpan.End()

		// 3. Authorization
		ctx, authzSpan := tracer.StartSpan(ctx, "authorize")
		allowed, err := authzService.IsAllowed(ctx, authz.Request{
			Subject:  userID,
			Resource: r.URL.Path,
			Action:   r.Method,
		})
		if err != nil {
			authzSpan.SetAttributes(attribute.String("error", err.Error()))
			authzSpan.End()
			http.Error(w, "Authorization error", http.StatusInternalServerError)
			return
		}
		if !allowed {
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
			// Get trace ID
			ctx, span := tracer.StartSpan(r.Context(), "request")
			defer span.End()

			traceID := tracing.TraceID(ctx)
			w.Header().Set("X-Trace-ID", traceID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Start server
	http.Handle("/api", traced(handler))
	log.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}