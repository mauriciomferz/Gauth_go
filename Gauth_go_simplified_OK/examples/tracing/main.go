package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

func main() {
	fmt.Println("🔍 GAuth Tracing Integration Demo")
	fmt.Println("=================================")

	// Initialize OpenTelemetry
	fmt.Println("\n1. Initializing OpenTelemetry tracing...")
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := exporter.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down exporter: %v", err)
		}
	}()

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
	)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	otel.SetTracerProvider(tp)
	tracer := otel.Tracer("gauth-demo")

	// Initialize RFC Compliant Service
	fmt.Println("\n2. Initializing RFC Compliant Service...")
	authService, err := auth.NewRFCCompliantService("tracing-issuer", "tracing-audience")
	if err != nil {
		log.Fatalf("failed to create RFC service: %v", err)
	}

	// Create a traced context
	ctx, span := tracer.Start(context.Background(), "gauth-authorization-flow")
	defer span.End()

	// Demonstrate traced GAuth operations
	fmt.Println("\n3. Executing traced GAuth operations...")

	// Create a Power of Attorney request with tracing
	_, childSpan := tracer.Start(ctx, "create-poa-request")
	poaRequest := auth.PowerOfAttorneyRequest{
		ClientID:     "traced-client-123",
		ResponseType: "code",
		Scope:        []string{"traced_operations", "audit_logging"},
		RedirectURI:  "https://traced-app.example.com/callback",
		State:        "traced-state-456",
		PowerType:    "traced_power_of_attorney",
		PrincipalID:  "traced-principal",
		AIAgentID:    "traced-ai-agent",
		Jurisdiction: "US",
		LegalBasis:   "tracing_compliance_act",
	}
	childSpan.End()

	// Execute authorization with tracing
	_, authSpan := tracer.Start(ctx, "authorize-gauth")
	gauthResponse, err := authService.AuthorizeGAuth(ctx, poaRequest)
	authSpan.End()

	if err != nil {
		fmt.Printf("❌ Authorization failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Authorization successful with tracing!\n")
	fmt.Printf("   Authorization Code: %s...\n", gauthResponse.AuthorizationCode[:20])
	fmt.Printf("   Legal Compliance: %s\n", gauthResponse.LegalCompliance)
	fmt.Printf("   Audit Record ID: %s\n", gauthResponse.AuditRecordID)

	// Demonstrate traced token operations
	fmt.Println("\n4. Demonstrating traced token lifecycle...")

	_, tokenSpan := tracer.Start(ctx, "token-operations")
	defer tokenSpan.End()

	// Simulate token operations with tracing context
	fmt.Printf("   🔍 Trace ID: %s\n", span.SpanContext().TraceID().String())
	fmt.Printf("   📊 Span ID: %s\n", span.SpanContext().SpanID().String())

	// Add artificial delay to show tracing duration
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n✅ Tracing Integration Demo Completed!")
	fmt.Println("📈 Check the console output above for detailed trace information.")
}
