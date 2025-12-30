package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/config"
	imetrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "Perform a lightweight health probe against the beta health endpoint and exit")
	flag.Parse()

	if *healthcheck {
		// Allow endpoint override via env for flexibility
		url := config.Get("GAUTH_HEALTH_URL", "http://localhost:8080/api/v1/beta/health")
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			fmt.Println("healthcheck request failed:", err)
			os.Exit(1)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			_ = resp.Body.Close()
			fmt.Println("OK")
			os.Exit(0)
		}
		_ = resp.Body.Close()
		fmt.Printf("unhealthy status: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	fmt.Println("AgentAuth Demo Application")
	fmt.Println("======================")
	enableProm := config.Get("GAUTH_METRICS", "") == "prometheus"
	if enableProm {
		fmt.Println("[metrics] Prometheus metrics enabled (endpoint /metrics)")
	} else {
		fmt.Println("[metrics] Prometheus metrics disabled (set GAUTH_METRICS=prometheus to enable)")
	}

	// Configuration via environment (educational – NOT production secure)
	authURL := config.Get("GAUTH_AUTH_SERVER_URL", "https://auth.example.com")
	clientID := config.Get("GAUTH_CLIENT_ID", "demo-client")
	clientSecret, generatedClient, warnClient := config.EphemeralSecret("GAUTH_CLIENT_SECRET", 24)
	if warnClient != "" {
		fmt.Println("⚠️  WARNING:", warnClient)
	}
	if generatedClient {
		if config.IsProduction() {
			fmt.Println("❌ GAUTH_MODE=production but GAUTH_CLIENT_SECRET not provided. Refusing to start with ephemeral secret.")
			os.Exit(1)
		}
		fmt.Println("(Ephemeral client secret generated – will change each run; set GAUTH_CLIENT_SECRET for stability)")
	}
	signingKey, generatedSign, warnSign := config.EphemeralSecret("GAUTH_SIGNING_KEY", 32)
	if warnSign != "" {
		fmt.Println("⚠️  WARNING:", warnSign)
	}
	if generatedSign {
		if config.IsProduction() {
			fmt.Println("❌ GAUTH_MODE=production but GAUTH_SIGNING_KEY not provided. Refusing to start with ephemeral signing key.")
			os.Exit(1)
		}
		fmt.Println("(Ephemeral signing key generated – tokens will be invalid across restarts; set GAUTH_SIGNING_KEY for stability)")
	}
	scopes := []string{"transaction:execute", "read", "write"}
	expiry := config.GetDurationSeconds("GAUTH_TOKEN_EXPIRY_SECONDS", 3600)

	gconfig := gauth.Config{
		AuthServerURL:     authURL,
		ClientID:          clientID,
		ClientSecret:      clientSecret,
		SigningKey:        signingKey,
		Scopes:            scopes,
		AccessTokenExpiry: time.Duration(expiry) * time.Second,
		RateLimit:         gauth.Config{}.RateLimit, // Placeholder
	}
	// NOTE: Current gauth.New does not accept instrumentation options; Prometheus adapter
	// is initialized solely to expose process metrics (no internal hook here yet).
	if enableProm {
		_ = imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "core"})
	}
	authService, err := gauth.New(gconfig)
	if err != nil {
		fmt.Println("Error creating AgentAuth instance:", err)
		return
	}

	// Simulate an authorization request and grant
	authReq := gauth.AuthorizationRequest{
		ClientID: "demo-client",
		Scopes:   []string{"transaction:execute"},
	}

	fmt.Println("\n1. Requesting Authorization")
	authGrant, err := authService.InitiateAuthorization(authReq)
	if err != nil {
		fmt.Println("Error requesting authorization:", err)
		return
	}
	fmt.Println("✓ Authorization granted")
	fmt.Printf("  - Grant ID: %s\n", authGrant.GrantID)
	fmt.Printf("  - Scopes: %v\n", authGrant.Scope)
	fmt.Printf("  - Expires: %v\n", authGrant.ValidUntil.Format(time.RFC3339))

	// Issue a token using the grant
	fmt.Println("\n2. Requesting Token")
	tokenReq := gauth.TokenRequest{
		GrantID:      authGrant.GrantID,
		Scope:        authGrant.Scope,
		Restrictions: authGrant.Restrictions,
		Context:      nil, // For demo, context is nil
	}
	tokenResp, err := authService.RequestToken(tokenReq)
	if err != nil {
		fmt.Println("Error issuing token:", err)
		return
	}
	fmt.Println("✓ Token issued")
	fmt.Printf("  - Token: %s\n", tokenResp.Token)
	fmt.Printf("  - Scopes: %v\n", tokenResp.Scope)
	fmt.Printf("  - Expires: %v\n", tokenResp.ValidUntil.Format(time.RFC3339))

	// Create a transaction
	fmt.Println("\n3. Creating Transaction")
	transaction := gauth.TransactionDetails{
		ID:          "tx-12345",
		Type:        gauth.PaymentTransaction,
		Status:      gauth.TransactionPending,
		ClientID:    "demo-client",
		ResourceID:  "resource-1",
		Scopes:      []string{"transaction:execute"},
		Amount:      50.0,
		Currency:    "USD",
		Timestamp:   time.Now(),
		Source:      "account-1",
		Destination: "account-2",
		Description: "Demo payment transaction",
	}
	fmt.Println("✓ Transaction created")
	fmt.Printf("  - ID: %s\n", transaction.ID)
	fmt.Printf("  - Type: %s\n", transaction.Type)
	fmt.Printf("  - Amount: %.2f\n", transaction.Amount)

	// Create a resource server
	fmt.Println("\n4. Initializing Resource Server")
	resourceServer := gauth.NewResourceServer("demo-resource", authService)
	fmt.Println("✓ Resource server initialized")

	// Process the transaction
	fmt.Println("\n5. Processing Transaction")
	resultMsg, err := resourceServer.ProcessTransaction(transaction, tokenResp.Token)
	if err != nil {
		fmt.Println("✗ Transaction failed:", err)
	} else {
		fmt.Println("✓ Transaction succeeded")
		fmt.Printf("  - Message: %s\n", resultMsg)
	}

	// Start metrics endpoint if enabled. Keep demo behavior synchronous otherwise.
	if enableProm {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		go func() {
			addr := os.Getenv("GAUTH_METRICS_ADDR")
			if addr == "" {
				addr = ":9095"
			}
			fmt.Println("[metrics] Serving /metrics on", addr)
			srv := &http.Server{Addr: addr, Handler: mux, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
			if err := srv.ListenAndServe(); err != nil {
				fmt.Println("metrics server error:", err)
			}
		}()
	}

	// (Audit log not available in current API)
	fmt.Println("\n6. Testing Token Expiration (simulated)")
	fmt.Println("(Token expiration handling would be tested here if API allowed direct manipulation)")
	fmt.Println("Demo completed successfully! Press Ctrl+C to exit (metrics server may be running)...")
	if enableProm {
		select {} // keep process alive for scraping
	}
}
