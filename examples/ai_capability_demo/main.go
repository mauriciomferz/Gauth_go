package main

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/examples/ai_capability_demo/ledger"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/ai"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc0111"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "modernc.org/sqlite"

	// OpenTelemetry minimal imports (stdout fallback)
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)


// Global instrumentation & auth variables
var (
    // Metrics (initialized when GAUTH_AI_DEMO_METRICS=1)
    pruneCounter      prometheus.Counter
    decisionRowsGauge prometheus.Gauge
    jwksFetchCounter  *prometheus.CounterVec
    jwksKeysGauge     prometheus.Gauge
	jwksNegHitsCounter prometheus.Counter
	jwksNegEntriesGauge prometheus.Gauge
	jwksBgRefreshCounter prometheus.Counter
	jwksCacheTtlRemainingGauge prometheus.Gauge
	jwksNegEvictionsCounter prometheus.Counter
	poaValidationCounter *prometheus.CounterVec
	poaRevocationsCounter prometheus.Counter
	poaIntegrityFailuresCounter *prometheus.CounterVec
	poaDigestMismatchCounter prometheus.Counter
	poaVersionMismatchCounter prometheus.Counter
	poaMultisigSignaturesCounter *prometheus.CounterVec
	poaMultisigFinalizationsCounter prometheus.Counter
	ledgerAppendsCounter *prometheus.CounterVec
	ledgerRootEmissionsCounter prometheus.Counter
	ledgerEventsCounter *prometheus.CounterVec
	ledgerProofRequestsCounter prometheus.Counter
	poaIssuanceCounter prometheus.Counter
	poaRevocationReasonCounter *prometheus.CounterVec
	auditLedgerAppendsCounter *prometheus.CounterVec
	auditLedgerSizeGauge prometheus.Gauge
	poaStatusTransitionsCounter *prometheus.CounterVec
    // JWKS cache state
    jwksCache  map[string]*rsa.PublicKey
    jwksExpiry time.Time
	// negative KID cache (kid -> expiry timestamp)
	jwksNegative map[string]time.Time
	// PoA repository (in-memory for demo) & mutex
	poaRepoMu sync.RWMutex
	poaRepo = make(map[string]*rfc0111.PowerOfAttorney)
	boltPOARepo *rfc0111.BoltRepository
	// Simple ledger instance (Week2 anchoring; may be replaced by BoltDB if persistence configured)
	ledgerInstance = ledger.New()
	boltLedgerInstance *ledger.BoltLedger
	auditLedgerInstance *ledger.AuditLedger
)

// JWKS response structure
type jwksResp struct {
    Keys []struct {
        Kid string `json:"kid"`
        Kty string `json:"kty"`
        Alg string `json:"alg"`
        Use string `json:"use"`
        N   string `json:"n"`
        E   string `json:"e"`
    } `json:"keys"`
}
// fetchJWKS retrieves JWKS and populates in-memory cache with TTL
func fetchJWKS(url string, cacheSeconds int) error {
		if url == "" { return errors.New("jwks_url_empty") }
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(url)
		if err != nil { return err }
		defer resp.Body.Close()
		if resp.StatusCode != 200 { return fmt.Errorf("jwks_http_status_%d", resp.StatusCode) }
		var parsed jwksResp
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil { return err }
		tmp := make(map[string]*rsa.PublicKey)
		for _, k := range parsed.Keys {
			if k.Kty != "RSA" { continue }
			modBytes, errN := base64.RawURLEncoding.DecodeString(k.N)
			if errN != nil { continue }
			expBytes, errE := base64.RawURLEncoding.DecodeString(k.E)
			if errE != nil { continue }
			n := new(big.Int).SetBytes(modBytes)
			// Convert exponent bytes (big-endian) to int
			exp := 0
			for _, b := range expBytes { exp = exp*256 + int(b) }
			if exp <= 0 { continue }
			pub := &rsa.PublicKey{N: n, E: exp}
			tmp[k.Kid] = pub
		}
		jwksCache = tmp
		jwksExpiry = time.Now().Add(time.Duration(cacheSeconds) * time.Second)
		// Metrics update if enabled
		if jwksFetchCounter != nil { jwksFetchCounter.WithLabelValues("success").Inc() }
		if jwksKeysGauge != nil { jwksKeysGauge.Set(float64(len(jwksCache))) }
		// Set initial TTL remaining gauge to full TTL
		if jwksCacheTtlRemainingGauge != nil { jwksCacheTtlRemainingGauge.Set(float64(cacheSeconds)) }
		return nil
	}

func getJWKSKey(kid string) (*rsa.PublicKey, error) {
		url := os.Getenv("GAUTH_AI_DEMO_JWKS_URL")
		if url == "" { return nil, errors.New("jwks_disabled") }
		cacheSecs := 300
		if v := os.Getenv("GAUTH_AI_DEMO_JWKS_CACHE_SECONDS"); v != "" { if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 { cacheSecs = parsed } }
			// Negative cache check first
			if jwksNegative != nil {
				if exp, exists := jwksNegative[kid]; exists {
					if time.Now().Before(exp) {
						if jwksNegHitsCounter != nil { jwksNegHitsCounter.Inc() }
						return nil, errors.New("kid_not_found")
					}
					// expired negative entry -> remove and update gauge
					delete(jwksNegative, kid)
					if jwksNegEntriesGauge != nil { jwksNegEntriesGauge.Set(float64(len(jwksNegative))) }
				}
			}
		if jwksCache == nil || time.Now().After(jwksExpiry) {
				if err := fetchJWKS(url, cacheSecs); err != nil {
					if jwksFetchCounter != nil { jwksFetchCounter.WithLabelValues("error").Inc() }
					return nil, fmt.Errorf("jwks_fetch_error:%v", err.Error())
				}
		}
		pub, ok := jwksCache[kid]
			if !ok {
				// add to negative cache with TTL if configured
				if ttlStr := os.Getenv("GAUTH_AI_DEMO_JWKS_NEGATIVE_TTL_SECONDS"); ttlStr != "" {
					if ttl, err := strconv.Atoi(ttlStr); err == nil && ttl > 0 {
						if jwksNegative == nil { jwksNegative = make(map[string]time.Time) }
						// Enforce max entries if configured
						if maxStr := os.Getenv("GAUTH_AI_DEMO_JWKS_NEGATIVE_MAX_ENTRIES"); maxStr != "" {
							if max, errM := strconv.Atoi(maxStr); errM == nil && max > 0 {
								if len(jwksNegative) >= max {
									// Evict oldest expiry
									var oldestKid string
									var oldestTime time.Time
									for k, expT := range jwksNegative {
										if oldestKid == "" || expT.Before(oldestTime) { oldestKid = k; oldestTime = expT }
									}
									delete(jwksNegative, oldestKid)
									if jwksNegEvictionsCounter != nil { jwksNegEvictionsCounter.Inc() }
								}
							}
						}
						jwksNegative[kid] = time.Now().Add(time.Duration(ttl) * time.Second)
						if jwksNegEntriesGauge != nil { jwksNegEntriesGauge.Set(float64(len(jwksNegative))) }
					}
				}
				return nil, errors.New("kid_not_found")
			}
		return pub, nil
	}

	// startJWKSBackgroundRefresh launches the JWKS background refresh loop (used by server and tests)
	func startJWKSBackgroundRefresh() {
		jwksURL := os.Getenv("GAUTH_AI_DEMO_JWKS_URL")
		if jwksURL == "" { return }
		go func(){
			for {
				cacheSecs := 300
				if v := os.Getenv("GAUTH_AI_DEMO_JWKS_CACHE_SECONDS"); v != "" { if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 { cacheSecs = parsed } }
				bgFactor := 0.5
				if fStr := os.Getenv("GAUTH_AI_DEMO_JWKS_BG_REFRESH_FACTOR"); fStr != "" { if f, err := strconv.ParseFloat(fStr, 64); err == nil && f > 0 && f < 1 { bgFactor = f } }
				jitterSeconds := 0
				if jStr := os.Getenv("GAUTH_AI_DEMO_JWKS_BG_REFRESH_JITTER_SECONDS"); jStr != "" { if j, err := strconv.Atoi(jStr); err == nil && j >= 0 { jitterSeconds = j } }
				if jwksCache == nil || time.Now().After(jwksExpiry) {
					_ = fetchJWKS(jwksURL, cacheSecs)
				} else {
					remaining := time.Until(jwksExpiry)
					refreshDelay := time.Duration(float64(cacheSecs)*bgFactor) * time.Second
					if refreshDelay > remaining { refreshDelay = remaining / 2 }
					// Apply +/- jitter if configured (uniform)
					if jitterSeconds > 0 {
						j := time.Duration(rand.Intn(jitterSeconds*1000)) * time.Millisecond // 0..jitterSeconds
						if rand.Intn(2) == 0 { refreshDelay += j } else { refreshDelay -= j }
						if refreshDelay < 250*time.Millisecond { refreshDelay = 250 * time.Millisecond } // floor to avoid thrash
					}
					time.Sleep(refreshDelay)
					_ = fetchJWKS(jwksURL, cacheSecs)
					if jwksBgRefreshCounter != nil { jwksBgRefreshCounter.Inc() }
				}
			}
		}()
		// Gauge updater (TTL remaining) if metric enabled
		if jwksCacheTtlRemainingGauge != nil {
			go func(){
				for {
					if !jwksExpiry.IsZero() {
						rem := time.Until(jwksExpiry).Seconds()
						if rem < 0 { rem = 0 }
						jwksCacheTtlRemainingGauge.Set(rem)
					}
					time.Sleep(1 * time.Second)
				}
			}()
		}
	}

func verifyRS256(headerPayload, signature string, pub *rsa.PublicKey) bool {
		sigBytes, err := base64.RawURLEncoding.DecodeString(signature)
		if err != nil { return false }
		h := sha256.New(); h.Write([]byte(headerPayload)); hashed := h.Sum(nil)
		if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed, sigBytes); err != nil { return false }
		return true
	}

func main() {
	// Prevent unused global metric variable compile errors prior to metrics initialization
	_ = jwksFetchCounter
	_ = jwksKeysGauge
	fmt.Println("🤖 AI Capability Matrix Enforcement Demo")
	fmt.Println("========================================")

	// Create AI capability integration
	integration := ai.NewServerIntegration()
	integration.EnableEnforcement(true)

	// Set up audit and metrics callbacks
	integration.SetAuditCallback(func(action string, metadata map[string]any) {
		fmt.Printf("📋 AUDIT: %s - %v\n", action, metadata["decision"])
	})

	integration.SetMetricsCallback(func(metric string) {
		// Metrics callback currently acts as a hook for jurisdiction_conflict increments.
		// Ensure in-memory PoA repo exists when BoltDB persistence is not enabled.
		poaRepoMu.Lock()
		defer poaRepoMu.Unlock()
		if boltPOARepo == nil {
			if poaRepo == nil || len(poaRepo) == 0 {
				poaRepo = make(map[string]*rfc0111.PowerOfAttorney)
			}
		}
	})

	// Demo scenarios
	fmt.Println("\n🎭 Demo Scenarios:")
	fmt.Println("================")

	// Scenario 1: Human User (should be allowed for everything)
	fmt.Println("\n1. 👤 Human User Access:")
	testHumanAccess(integration)

	// Scenario 2: AI Assistant (restricted access)
	fmt.Println("\n2. 🤖 AI Assistant Access:")
	testAIAssistantAccess(integration)

	// Scenario 3: AI Agent with proper compliance (should be allowed with restrictions)
	fmt.Println("\n3. 🤖 AI Agent with Compliance:")
	testAIAgentAccess(integration)

	// Scenario 4: EU AI Agent (stricter compliance)
	fmt.Println("\n4. 🇪🇺 EU AI Agent (EU AI Act):")
	testEUAIAgentAccess(integration)

	// Scenario 5: Healthcare AI (HIPAA compliance)
	fmt.Println("\n5. 🏥 Healthcare AI (HIPAA):")
	testHealthcareAIAccess(integration)

	// Scenario 6: Financial AI (SOX compliance)
	fmt.Println("\n6. 💰 Financial AI (SOX):")
	testFinancialAIAccess(integration)

	// Show governance policies
	fmt.Println("\n📋 Loaded Governance Policies:")
	fmt.Println("=============================")
	policies := integration.GetGovernancePolicies()
	for _, policy := range policies {
		fmt.Printf("- %s (%s, %s)\n", policy.PolicyID, policy.Jurisdiction, policy.ComplianceFramework)
	}

	// Environment controlled server behavior
	noServer := os.Getenv("GAUTH_AI_DEMO_NO_SERVER") == "1"
	port := os.Getenv("GAUTH_AI_DEMO_PORT")
	if port == "" { port = "8080" }

	// Initialize optional SQLite persistence
	dbPath := os.Getenv("GAUTH_AI_DEMO_DB_PATH")
	var db *sql.DB
	if dbPath != "" {
		var err error
		db, err = sql.Open("sqlite", dbPath)
		if err != nil { log.Printf("sqlite open error: %v", err) } else {
			if err := initSchema(db); err != nil { log.Printf("sqlite schema error: %v", err) }
		}
	}

	if noServer {
		fmt.Println("\n🚫 GAUTH_AI_DEMO_NO_SERVER=1 set; skipping HTTP server start. Scenarios complete.")
		return
	}

	// Initialize OpenTelemetry tracer if enabled
	var tracer trace.Tracer
	var tp *sdktrace.TracerProvider
	if os.Getenv("GAUTH_AI_DEMO_OTEL") == "1" {
		tracer, tp = initTracer()
		fmt.Println("[otel] tracing enabled")
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := tp.Shutdown(ctx); err != nil { fmt.Println("[otel] shutdown error:", err) }
		}()
	}

	fmt.Println("\n🌐 Starting AI Capability API Server on :" + port + "...")
	fmt.Println("======================================")
	// Optional BoltDB ledger persistence initialization
	if lp := os.Getenv("GAUTH_AI_DEMO_LEDGER_DB_PATH"); lp != "" {
		if bl, err := ledger.NewBoltLedger(lp); err == nil {
			boltLedgerInstance = bl
			fmt.Println("[ledger] bolt persistence enabled at", lp)
		} else {
			fmt.Println("[ledger] bolt init error:", err)
		}
	}
	// Week2: Optional audit hash-chain ledger persistence (single global instance)
	if ap := os.Getenv("GAUTH_AI_DEMO_AUDIT_DB_PATH"); ap != "" {
		if al, err := ledger.NewAuditLedger(ap); err == nil {
			auditLedgerInstance = al
			fmt.Println("[audit-ledger] persistence enabled at", ap, "head=", al.HeadHash(), "size=", al.Size())
		} else { fmt.Println("[audit-ledger] init error:", err) }
	}
	startAPIServer(integration, port, db, tracer)
}

// (rest of code omitted for patch brevity)
func testHumanAccess(integration *ai.ServerIntegration) {
	claims := map[string]any{
		"user_id": "human-123",
		"name":    "John Doe",
	}

	allowed, missing, metadata := integration.EnforceAICapabilities("admin:delete", claims)
	fmt.Printf("   Action: admin:delete | Allowed: %v | Missing: %v | Entity: %v\n",
		allowed, missing, metadata["entity_type"])
}

func testAIAssistantAccess(integration *ai.ServerIntegration) {
	claims := map[string]any{
		"ai_entity_type":             "assistant",
		"system_id":                  "chatgpt-4",
		"jurisdiction":               "US",
		"ai_entity_verified":         true,
		"algorithmic_accountability": true,
	}

	// Test allowed action
	allowed, missing, metadata := integration.EnforceAICapabilities("transaction:read", claims)
	fmt.Printf("   Action: transaction:read | Allowed: %v | Missing: %v | Human Auth: %v\n",
		allowed, missing, metadata["required_human_auth"])

	// Test forbidden action
	allowed, missing, metadata = integration.EnforceAICapabilities("transaction:execute", claims)
	fmt.Printf("   Action: transaction:execute | Allowed: %v | Missing: %v | Reason: %v\n",
		allowed, missing, metadata["reason"])
}

func testAIAgentAccess(integration *ai.ServerIntegration) {
	claims := map[string]any{
		"ai_entity_type":             "agent",
		"system_id":                  "autonomous-agent-1",
		"jurisdiction":               "US",
		"risk_level":                 "medium",
		"ai_entity_verified":         true,
		"ai_agent_registered":        true,
		"algorithmic_accountability": true,
	}

	allowed, missing, metadata := integration.EnforceAICapabilities("transaction:execute", claims)
	fmt.Printf("   Action: transaction:execute | Allowed: %v | Missing: %v | Human Auth: %v | Audit: %v\n",
		allowed, missing, metadata["required_human_auth"], metadata["audit_level"])
}

func testEUAIAgentAccess(integration *ai.ServerIntegration) {
	claims := map[string]any{
		"ai_entity_type":       "agent",
		"system_id":            "eu-agent-1",
		"jurisdiction":         "EU",
		"risk_level":           "high",
		"ai_entity_verified":   true,
		"ai_agent_registered":  true,
		// Missing EU compliance claims
	}

	allowed, missing, metadata := integration.EnforceAICapabilities("transaction:execute", claims)
	fmt.Printf("   Action: transaction:execute | Allowed: %v | Missing: %v | Policies: %v\n",
		allowed, missing, metadata["applied_policies"])

	// Now with EU compliance
	claims["eu_ai_conformity"] = true
	claims["human_oversight"] = true
	claims["ai_risk_assessment"] = true

	allowed, _, metadata = integration.EnforceAICapabilities("transaction:read", claims)
	fmt.Printf("   Action: transaction:read (with compliance) | Allowed: %v | Human Auth: %v\n",
		allowed, metadata["required_human_auth"])
}

func testHealthcareAIAccess(integration *ai.ServerIntegration) {
	claims := map[string]any{
		"ai_entity_type":             "analytics",
		"system_id":                  "healthcare-ai-1",
		"jurisdiction":               "US",
		"industry_context":           "healthcare",
		"risk_level":                 "critical",
		"ai_analytics_approved":      true,
		"ai_entity_verified":         true,
		"hipaa_compliance":           true,
		"phi_protection":             true,
		"healthcare_cert":            true,
		"de_identification":          true,
		"algorithmic_accountability": true,
	}

	allowed, _, metadata := integration.EnforceAICapabilities("transaction:read", claims)
	fmt.Printf("   Action: transaction:read | Allowed: %v | Human Auth: %v | Audit: %v | Policies: %v\n",
		allowed, metadata["required_human_auth"], metadata["audit_level"], metadata["applied_policies"])
}

func testFinancialAIAccess(integration *ai.ServerIntegration) {
	claims := map[string]any{
		"ai_entity_type":             "automation",
		"system_id":                  "finance-ai-1",
		"jurisdiction":               "US",
		"industry_context":           "finance",
		"risk_level":                 "high",
		"ai_automation_certified":    true,
		"ai_entity_verified":         true,
		"sox_compliance":             true,
		"financial_cert":             true,
		"model_validation":           true,
		"algorithmic_accountability": true,
	}

	allowed, _, metadata := integration.EnforceAICapabilities("transaction:read", claims)
	fmt.Printf("   Action: transaction:read | Allowed: %v | Human Auth: %v | Policies: %v\n",
		allowed, metadata["required_human_auth"], metadata["applied_policies"])

	// Test forbidden action
	allowed, _, metadata = integration.EnforceAICapabilities("transaction:pay", claims)
	fmt.Printf("   Action: transaction:pay | Allowed: %v | Reason: %v\n",
		allowed, metadata["reason"])
}

// initSchema creates decisions table if not exists.
func initSchema(db *sql.DB) error {
	if db == nil { return errors.New("db nil") }
	ddl := `CREATE TABLE IF NOT EXISTS decisions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		action TEXT NOT NULL,
		allowed INTEGER NOT NULL,
		entity_type TEXT,
		jurisdictions TEXT,
		missing TEXT,
		applied_policies TEXT,
		poa_id TEXT,
		poa_digest TEXT,
		poa_version INTEGER,
		created_at TEXT NOT NULL
	);`
	if _, err := db.Exec(ddl); err != nil { return err }
	// Auto-migration for legacy schema (ensure poa_id, poa_digest columns exist)
	rows, err := db.Query(`PRAGMA table_info(decisions)`)
	if err != nil { return nil } // non-critical
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var cid int; var name, ctype string; var notnull, pk int; var dflt sql.NullString
		_ = rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
		cols[strings.ToLower(name)] = true
	}
	for _, col := range []string{"poa_id", "poa_digest", "poa_version"} {
		if !cols[col] {
			ctype := "TEXT"
			if col == "poa_version" { ctype = "INTEGER" }
			alter := fmt.Sprintf("ALTER TABLE decisions ADD COLUMN %s %s", col, ctype)
			if _, e := db.Exec(alter); e != nil { log.Printf("schema migration: add column %s failed: %v", col, e) }
		}
	}
	return nil
}

func persistDecision(db *sql.DB, action string, allowed bool, meta map[string]any, missing []string) {
	if db == nil { return }
	entity := fmt.Sprintf("%v", meta["entity_type"])
	applied := fmt.Sprintf("%v", meta["applied_policies"])
	juris := fmt.Sprintf("%v", meta["jurisdiction"]) // single jurisdiction
	if v, ok := meta["jurisdictions"].([]string); ok { juris = strings.Join(v, ",") }
	miss := strings.Join(missing, ",")
	poaID := fmt.Sprintf("%v", meta["poa_id"])
	poaDigest := fmt.Sprintf("%v", meta["poa_digest"])
	poaVersion := -1
	if v, ok := meta["poa_version"].(int); ok { poaVersion = v } else if vf, ok2 := meta["poa_version"].(float64); ok2 { poaVersion = int(vf) }
	_, err := db.Exec(`INSERT INTO decisions(action,allowed,entity_type,jurisdictions,missing,applied_policies,poa_id,poa_digest,poa_version,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, action, boolToInt(allowed), entity, juris, miss, applied, poaID, poaDigest, poaVersion, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil { log.Printf("persist error: %v", err) }
	// Optional retention pruning (max rows)
	if maxStr := os.Getenv("GAUTH_AI_DEMO_DB_MAX_ROWS"); maxStr != "" {
		if max, errP := strconv.Atoi(maxStr); errP == nil && max > 0 {
			var count int
			if errC := db.QueryRow(`SELECT COUNT(*) FROM decisions`).Scan(&count); errC == nil && count > max {
				res, errD := db.Exec(`DELETE FROM decisions WHERE id NOT IN (SELECT id FROM decisions ORDER BY id DESC LIMIT ?)`, max)
				if errD != nil { log.Printf("prune error: %v", errD) } else if pruneCounter != nil { if n, _ := res.RowsAffected(); n > 0 { pruneCounter.Inc() } }
			}
		}
	}
	// Optional age-based pruning (max age days)
	if ageStr := os.Getenv("GAUTH_AI_DEMO_DB_MAX_AGE_DAYS"); ageStr != "" {
		if days, errA := strconv.Atoi(ageStr); errA == nil && days > 0 {
			cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour).Format(time.RFC3339Nano)
			res, errDel := db.Exec(`DELETE FROM decisions WHERE created_at < ?`, cutoff)
			if errDel != nil { log.Printf("age prune error: %v", errDel) } else if pruneCounter != nil { if n, _ := res.RowsAffected(); n > 0 { pruneCounter.Inc() } }
		}
	}
	// Update gauge with final row count (post pruning)
	if decisionRowsGauge != nil {
		var count int
		if errCnt := db.QueryRow(`SELECT COUNT(*) FROM decisions`).Scan(&count); errCnt == nil {
			decisionRowsGauge.Set(float64(count))
		}
	}
}

func boolToInt(b bool) int { if b { return 1 }; return 0 }

// authMiddleware validates API key or JWT (HS256) based on env configuration.
func authMiddleware() gin.HandlerFunc {
	// Test bypass: if GAUTH_AI_DEMO_TEST_BYPASS_AUTH=1, skip all auth checks (facilitates unit tests for new endpoints)
	if os.Getenv("GAUTH_AI_DEMO_TEST_BYPASS_AUTH") == "1" {
		return func(c *gin.Context) { c.Next() }
	}
	apiKey := os.Getenv("GAUTH_AI_DEMO_API_KEY")
	jwtSecret := os.Getenv("GAUTH_AI_DEMO_JWT_SECRET")
	jwksURL := os.Getenv("GAUTH_AI_DEMO_JWKS_URL")
	expectAlg := os.Getenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG")
	if apiKey == "" && jwtSecret == "" && jwksURL == "" { // no auth configured
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		unauth := func(reason string) { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "reason": reason}) }
		// API Key path (optional)
		if apiKey != "" {
			if hdr := c.GetHeader("X-API-Key"); hdr != "" && hdr == apiKey { c.Next(); return }
			// Continue to JWT check if configured
			if jwtSecret == "" && jwksURL == "" { unauth("missing_api_key"); return }
		}
		// JWT verification (supports HS256 via shared secret or RS256 via JWKS)
		bearer := c.GetHeader("Authorization")
		if strings.HasPrefix(strings.ToLower(bearer), "bearer ") {
			token := strings.TrimSpace(bearer[7:])
			parts := strings.Split(token, ".")
			if len(parts) == 3 {
				// decode header for alg + kid
				hdrBytes, errHdr := base64.RawURLEncoding.DecodeString(parts[0])
				if errHdr != nil { unauth("malformed_token"); return }
				var hdr map[string]any
				if err := json.Unmarshal(hdrBytes, &hdr); err != nil { unauth("malformed_token"); return }
				alg, _ := hdr["alg"].(string)
				kid, _ := hdr["kid"].(string)
				if expectAlg != "" && alg != expectAlg { unauth("unsupported_alg"); return }
				headerPayload := parts[0] + "." + parts[1]
				switch alg {
				case "HS256":
					if jwtSecret == "" { unauth("unsupported_alg"); return }
					computed := hmacSHA256Base64URL(headerPayload, jwtSecret)
					if subtleConstantTimeEq(computed, parts[2]) {
						payloadRaw, errDec := base64.RawURLEncoding.DecodeString(parts[1])
						if errDec != nil { unauth("malformed_token"); return }
						valid, reason := validateJWTClaimsDetailed(payloadRaw)
						if valid {
							// Optional PoA claim validation (poa_id embedded in token claims) – lightweight check
							var claims map[string]any
							_ = json.Unmarshal(payloadRaw, &claims)
							if poaID, ok := claims["poa_id"].(string); ok && poaID != "" {
								poaRepoMu.RLock(); var poa *rfc0111.PowerOfAttorney; var exists bool
								if boltPOARepo != nil { poa, exists = boltPOARepo.Get(poaID) } else { poa, exists = poaRepo[poaID] }
								poaRepoMu.RUnlock()
								if !exists { unauth("poa_not_found"); return }
								now := time.Now().UTC()
								if poa.Status == rfc0111.POAStatusRevoked { unauth("poa_revoked"); return }
								if now.Before(poa.ValidFrom) || now.After(poa.ValidUntil) { unauth("poa_expired"); return }
								// Digest/version integrity checks
								if digClaim, okD := claims["poa_digest"].(string); okD && digClaim != "" {
									if calcDig, _, errDig := rfc0111.CanonicalPOADigest(poa); errDig == nil {
										if calcDig != digClaim {
											if poaIntegrityFailuresCounter != nil { poaIntegrityFailuresCounter.WithLabelValues("digest_mismatch").Inc() }
											if poaDigestMismatchCounter != nil { poaDigestMismatchCounter.Inc() }
											unauth("poa_digest_mismatch"); return
										}
									}
								}
								if verClaim, okV := claims["poa_version"].(float64); okV {
									if int(verClaim) != poa.Version {
										if poaIntegrityFailuresCounter != nil { poaIntegrityFailuresCounter.WithLabelValues("version_mismatch").Inc() }
										if poaVersionMismatchCounter != nil { poaVersionMismatchCounter.Inc() }
										unauth("poa_version_mismatch"); return
									}
								}
							}
							c.Next(); return
						}
						unauth(reason); return
					}
					unauth("invalid_jwt_signature"); return
				case "RS256":
					if jwksURL == "" { unauth("unsupported_alg"); return }
					pub, errKey := getJWKSKey(kid)
					if errKey != nil {
						errMsg := errKey.Error()
						if strings.HasPrefix(errMsg, "jwks_fetch_error") { unauth("jwks_fetch_error"); return }
						if errMsg == "kid_not_found" { unauth("kid_not_found"); return }
						unauth("jwks_fetch_error"); return
					}
					if verifyRS256(headerPayload, parts[2], pub) {
						payloadRaw, errDec := base64.RawURLEncoding.DecodeString(parts[1])
						if errDec != nil { unauth("malformed_token"); return }
						valid, reason := validateJWTClaimsDetailed(payloadRaw)
						if valid {
							var claims map[string]any
							_ = json.Unmarshal(payloadRaw, &claims)
							if poaID, ok := claims["poa_id"].(string); ok && poaID != "" {
								poaRepoMu.RLock(); var poa *rfc0111.PowerOfAttorney; var exists bool
								if boltPOARepo != nil { poa, exists = boltPOARepo.Get(poaID) } else { poa, exists = poaRepo[poaID] }
								poaRepoMu.RUnlock()
								if !exists { unauth("poa_not_found"); return }
								now := time.Now().UTC()
								if poa.Status == rfc0111.POAStatusRevoked { unauth("poa_revoked"); return }
								if now.Before(poa.ValidFrom) || now.After(poa.ValidUntil) { unauth("poa_expired"); return }
								if digClaim, okD := claims["poa_digest"].(string); okD && digClaim != "" {
									if calcDig, _, errDig := rfc0111.CanonicalPOADigest(poa); errDig == nil {
										if calcDig != digClaim {
											if poaIntegrityFailuresCounter != nil { poaIntegrityFailuresCounter.WithLabelValues("digest_mismatch").Inc() }
											if poaDigestMismatchCounter != nil { poaDigestMismatchCounter.Inc() }
											unauth("poa_digest_mismatch"); return
										}
									}
								}
								if verClaim, okV := claims["poa_version"].(float64); okV {
									if int(verClaim) != poa.Version {
										if poaIntegrityFailuresCounter != nil { poaIntegrityFailuresCounter.WithLabelValues("version_mismatch").Inc() }
										if poaVersionMismatchCounter != nil { poaVersionMismatchCounter.Inc() }
										unauth("poa_version_mismatch"); return
									}
								}
							}
							c.Next(); return
						}
						unauth(reason); return
					}
					unauth("rsa_verification_failed"); return
				default:
					unauth("unsupported_alg"); return
				}
			}
			unauth("malformed_token"); return
		}
		unauth("missing_api_key")
	}
}

// hmacSHA256Base64URL returns base64url (no padding) encoded HMAC-SHA256 signature.
func hmacSHA256Base64URL(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	raw := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(raw)
}

// subtleConstantTimeEq compares two strings in constant-ish time.
func subtleConstantTimeEq(a, b string) bool {
	if len(a) != len(b) { return false }
	var diff byte
	for i := 0; i < len(a); i++ { diff |= a[i] ^ b[i] }
	return diff == 0
}

// validateJWTClaims parses JSON and validates standard claims if present.
// Enforced: exp (future), nbf (past), iat (reasonable skew), iss/aud match expected env values, alg header=HS256.
// Clock skew allowed via GAUTH_AI_DEMO_JWT_CLOCK_SKEW_SECONDS (default 60).
// Expected issuer/audience via GAUTH_AI_DEMO_JWT_EXPECT_ISS / GAUTH_AI_DEMO_JWT_EXPECT_AUD.
func validateJWTClaims(payload []byte) bool {
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil { return false }
	skew := int64(60)
	if v := os.Getenv("GAUTH_AI_DEMO_JWT_CLOCK_SKEW_SECONDS"); v != "" { if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed >= 0 { skew = parsed } }
	now := time.Now().Unix()
	// exp
	if v, ok := claims["exp"].(float64); ok { if now > int64(v)+skew { return false } }
	// nbf
	if v, ok := claims["nbf"].(float64); ok { if now+skew < int64(v) { return false } }
	// iat (allow within skew future/leeway)
	if v, ok := claims["iat"].(float64); ok { if int64(v) > now+skew { return false } }
	// iss
	if expectedIss := os.Getenv("GAUTH_AI_DEMO_JWT_EXPECT_ISS"); expectedIss != "" {
		if v, ok := claims["iss"].(string); !ok || v != expectedIss { return false }
	}
	// aud (string or array)
	if expectedAud := os.Getenv("GAUTH_AI_DEMO_JWT_EXPECT_AUD"); expectedAud != "" {
		audValid := false
		switch v := claims["aud"].(type) {
		case string:
			audValid = v == expectedAud
		case []any:
			for _, item := range v { if s, ok := item.(string); ok && s == expectedAud { audValid = true; break } }
		}
		if !audValid { return false }
	}
	return true
}

// validateJWTClaimsDetailed returns (valid,bool) and reason code if invalid.
func validateJWTClaimsDetailed(payload []byte) (bool,string) {
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil { return false, "malformed_token" }
	skew := int64(60)
	if v := os.Getenv("GAUTH_AI_DEMO_JWT_CLOCK_SKEW_SECONDS"); v != "" { if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed >= 0 { skew = parsed } }
	now := time.Now().Unix()
	if v, ok := claims["exp"].(float64); ok { if now > int64(v)+skew { return false, "expired_token" } }
	if v, ok := claims["nbf"].(float64); ok { if now+skew < int64(v) { return false, "not_before_violation" } }
	if v, ok := claims["iat"].(float64); ok { if int64(v) > now+skew { return false, "future_iat" } }
	if expectedIss := os.Getenv("GAUTH_AI_DEMO_JWT_EXPECT_ISS"); expectedIss != "" {
		if v, ok := claims["iss"].(string); !ok || v != expectedIss { return false, "issuer_mismatch" }
	}
	if expectedAud := os.Getenv("GAUTH_AI_DEMO_JWT_EXPECT_AUD"); expectedAud != "" {
		audValid := false
		switch v := claims["aud"].(type) {
		case string:
			audValid = v == expectedAud
		case []any:
			for _, item := range v { if s, ok := item.(string); ok && s == expectedAud { audValid = true; break } }
		}
		if !audValid { return false, "audience_mismatch" }
	}
	return true, ""
}

func initTracer() (trace.Tracer, *sdktrace.TracerProvider) {
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil { fmt.Println("[otel] exporter error:", err); return otel.Tracer("gauth-ai-demo"), nil }
	res, _ := resource.New(context.Background(), resource.WithAttributes(
		semconv.ServiceName("gauth-ai-capability-demo"),
	))
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	return tp.Tracer("gauth-ai-demo"), tp
}

func startAPIServer(integration *ai.ServerIntegration, port string, db *sql.DB, tracer trace.Tracer) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(authMiddleware())

	// Optional Prometheus metrics (GAUTH_AI_DEMO_METRICS=1)
	var decisionCounter *prometheus.CounterVec
	var conflictCounter prometheus.Counter
	var enforceDuration *prometheus.HistogramVec
	var conflictDuration prometheus.Histogram
	if os.Getenv("GAUTH_AI_DEMO_METRICS") == "1" {
		decisionCounter = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_demo_decisions_total",
			Help: "Total number of AI capability enforcement decisions (allowed dimension).",
		}, []string{"action", "allowed"})
		conflictCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_demo_conflicts_total",
			Help: "Total number of jurisdiction conflicts detected in multi-jurisdiction evaluation.",
		})
		enforceDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ai_demo_enforcement_duration_seconds",
			Help:    "Duration of single enforcement decisions.",
			Buckets: prometheus.ExponentialBuckets(0.0005, 2, 15),
		}, []string{"action"})
		conflictDuration = promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "ai_demo_conflict_batch_duration_seconds",
			Help:    "Duration of multi-jurisdiction conflict simulation batches.",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
		})
		pruneCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_demo_prune_operations_total",
			Help: "Total number of prune delete operations (rows removed in retention or age pruning).",
		})
		decisionRowsGauge = promauto.NewGauge(prometheus.GaugeOpts{
			Name: "ai_demo_decisions_store_rows",
			Help: "Current number of rows in the decisions persistence store.",
		})
		jwksFetchCounter = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_demo_jwks_fetch_total",
			Help: "Total JWKS fetch attempts labeled by result (success|error).",
			}, []string{"result"})
		jwksKeysGauge = promauto.NewGauge(prometheus.GaugeOpts{
			Name: "ai_demo_jwks_keys_loaded",
			Help: "Current number of JWKS public keys cached for RS256 verification.",
		})
		jwksNegHitsCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_demo_jwks_negative_hits_total",
			Help: "Total number of negative JWKS kid cache hits (prevented redundant fetch attempts).",
		})
		jwksNegEntriesGauge = promauto.NewGauge(prometheus.GaugeOpts{
			Name: "ai_demo_jwks_negative_entries",
			Help: "Current number of entries in the negative JWKS kid cache.",
		})
		jwksBgRefreshCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_demo_jwks_bg_refresh_total",
			Help: "Total number of proactive background JWKS refresh operations performed before cache expiry.",
		})
		jwksCacheTtlRemainingGauge = promauto.NewGauge(prometheus.GaugeOpts{
			Name: "ai_demo_jwks_cache_ttl_remaining_seconds",
			Help: "Seconds remaining until current JWKS cache expiry (updates every second).",
		})
		jwksNegEvictionsCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_demo_jwks_negative_evictions_total",
			Help: "Total number of evictions performed on the negative JWKS kid cache due to max entries limit.",
		})
		poaValidationCounter = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_demo_poa_validations_total",
			Help: "Total number of PoA validation attempts labeled by result (success|revoked|expired|scope_mismatch|not_found)",
		}, []string{"result"})
		poaRevocationsCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_demo_poa_revocations_total",
			Help: "Total number of PoA revocations processed in demo.",
		})
		poaIntegrityFailuresCounter = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_demo_poa_integrity_failures_total",
			Help: "Total number of PoA integrity validation failures (digest/version).",
		}, []string{"reason"})
		poaDigestMismatchCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_demo_poa_digest_mismatch_total",
			Help: "Total number of tokens failing PoA digest integrity binding",
		})
		poaVersionMismatchCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_demo_poa_version_mismatch_total",
			Help: "Total number of tokens failing PoA version integrity binding",
		})
		poaMultisigSignaturesCounter = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_demo_poa_multisig_signatures_total",
			Help: "Total number of multi-signature submissions for PoA drafts (result label indicates status).",
		}, []string{"result"})
		poaMultisigFinalizationsCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_demo_poa_multisig_finalizations_total",
			Help: "Total number of multi-signature PoA finalizations transitioning draft -> active.",
		})
		ledgerAppendsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_demo_ledger_appends_total",
			Help: "Total number of ledger append operations labeled by type.",
		}, []string{"type"})
		ledgerRootEmissionsCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_demo_ledger_root_emissions_total",
			Help: "Total number of ledger Merkle root emissions (after append).",
		})
		ledgerEventsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_demo_ledger_events_total",
			Help: "Total number of anchored ledger events labeled by event type.",
		}, []string{"event"})
		poaIssuanceCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "ai_demo_poa_issuance_total",
			Help: "Total number of PoA issuance operations (active + draft).",
		})
		poaRevocationReasonCounter = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_demo_poa_revocation_reason_total",
			Help: "PoA revocations labeled by reason (empty -> none).",
		}, []string{"reason"})
		auditLedgerAppendsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_demo_audit_ledger_appends_total",
			Help: "Audit hash-chain ledger append operations labeled by type.",
		}, []string{"type"})
		auditLedgerSizeGauge = promauto.NewGauge(prometheus.GaugeOpts{
			Name: "ai_demo_audit_ledger_size",
			Help: "Current number of entries in the audit hash-chain ledger.",
		})
		poaStatusTransitionsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_demo_poa_status_transitions_total",
			Help: "Lifecycle status transitions labeled by from/to.",
		}, []string{"from","to"})
		// Expose /metrics endpoint
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// Background JWKS refresh goroutine (optional)
	startJWKSBackgroundRefresh()

	// Helper: append to audit ledger (if enabled) with metrics updates
	appendAudit := func(entryType, actor, poaID string, metadata map[string]any) {
		if auditLedgerInstance == nil { return }
		if metadata == nil { metadata = map[string]any{} }
		if _, err := auditLedgerInstance.Append(entryType, actor, poaID, metadata); err == nil {
			if auditLedgerAppendsCounter != nil { auditLedgerAppendsCounter.WithLabelValues(entryType).Inc() }
			if auditLedgerSizeGauge != nil { auditLedgerSizeGauge.Set(float64(auditLedgerInstance.Size())) }
		} else {
			// silent failure is okay for demo; could add log.Printf
		}
	}

	// Add AI capability API routes
	apiHandler := ai.NewAPIHandler(integration)
	apiHandler.RegisterRoutes(router)

	// Add demo routes
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "GAuth AI Capability Matrix Demo",
			"version": "1.0.0",
			"endpoints": []string{
				"GET /api/v1/ai/capabilities/status",
				"GET /api/v1/ai/capabilities/entity-types",
				"GET /api/v1/ai/capabilities/policies",
				"POST /api/v1/ai/capabilities/test/enforcement",
				"POST /api/v1/ai/capabilities/simulate/decision",
				"GET /api/v1/ai/health",
			},
		})
	})

	// Demo enforcement endpoint (persist + optional jurisdictions conflict demo)
	router.POST("/demo/enforce", func(c *gin.Context) {
		var request struct {
			Action string         `json:"action"`
			Claims map[string]any `json:"claims"`
			Jurisdictions []string `json:"jurisdictions"` // optional multi-jurisdiction context
			POAID string `json:"poa_id"` // optional PoA reference for delegated action
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Basic conflict simulation: if multiple jurisdictions provided and decision differs per jurisdiction, mark conflict.
		var conflict bool
		ctx := context.Background()
		var span trace.Span
		if tracer != nil { ctx, span = tracer.Start(ctx, "demo.enforce") }
		var timer *prometheus.Timer
		if enforceDuration != nil { timer = prometheus.NewTimer(enforceDuration.WithLabelValues(request.Action)) }
		allowed, missing, metadata := integration.EnforceAICapabilities(request.Action, request.Claims)
		// Optional PoA validation if poa_id provided
		if request.POAID != "" {
			result := "not_found"
				poaRepoMu.RLock(); var poa *rfc0111.PowerOfAttorney; var ok bool
				if boltPOARepo != nil { poa, ok = boltPOARepo.Get(request.POAID) } else { poa, ok = poaRepo[request.POAID] }
				poaRepoMu.RUnlock()
			if ok {
				// Basic status/time/scope validation
				now := time.Now().UTC()
				if poa.Status == rfc0111.POAStatusRevoked {
					result = "revoked"
					allowed = false
				} else if now.After(poa.ValidUntil) || now.Before(poa.ValidFrom) {
					result = "expired"
					allowed = false
				} else if len(poa.Scope) > 0 {
					// scope match: action must be in poa.Scope
					match := false
					for _, s := range poa.Scope { if s == request.Action { match = true; break } }
					if !match {
						result = "scope_mismatch"
						allowed = false
					} else {
						result = "success"
					}
				} else {
					result = "success" // empty scope treated as wildcard for demo
				}
				metadata["poa_id"] = poa.ID
				if dig, canon, err := rfc0111.CanonicalPOADigest(poa); err == nil { metadata["poa_digest"] = dig; _ = canon }
			} else {
				allowed = false // cannot allow delegated action without known PoA
			}
			if poaValidationCounter != nil { poaValidationCounter.WithLabelValues(result).Inc() }
		}
		if timer != nil { timer.ObserveDuration() }
		if decisionCounter != nil {
			decisionCounter.WithLabelValues(request.Action, strconv.FormatBool(allowed)).Inc()
		}
		if span != nil {
			span.SetAttributes(
				attribute.String("action", request.Action),
				attribute.Bool("allowed", allowed),
				attribute.Int("missing_count", len(missing)),
			)
		}
		if len(request.Jurisdictions) > 1 {
			// naive approach: re-run per jurisdiction with claim override
			decisions := map[string]bool{}
			origJur := fmt.Sprintf("%v", request.Claims["jurisdiction"])
			for _, j := range request.Jurisdictions {
				request.Claims["jurisdiction"] = j
				ajAllowed, _, _ := integration.EnforceAICapabilities(request.Action, request.Claims)
				decisions[j] = ajAllowed
			}
			request.Claims["jurisdiction"] = origJur
			var firstVal *bool
			for _, v := range decisions { if firstVal == nil { first := v; firstVal = &first; continue }; if v != *firstVal { conflict = true; break } }
			metadata["jurisdictions"] = request.Jurisdictions
			metadata["jurisdiction_conflict"] = conflict
			if conflict {
				integration.GetMetricsCallback()("jurisdiction_conflict")
				if conflictCounter != nil { conflictCounter.Inc() }
			}
		}
		persistDecision(db, request.Action, allowed, metadata, missing)
		// Audit ledger entry for enforcement decision
		appendAudit("decision", fmt.Sprintf("%v", metadata["entity_type"]), fmt.Sprintf("%v", metadata["poa_id"]), map[string]any{"action": request.Action, "allowed": allowed, "missing": missing, "jurisdictions": metadata["jurisdictions"], "conflict": metadata["jurisdiction_conflict"]})
		// Automatic ledger anchoring for decisions (type=decision)
		decPayload := map[string]any{"action": request.Action, "allowed": allowed, "missing": missing, "poa_id": metadata["poa_id"], "jurisdictions": metadata["jurisdictions"], "conflict": metadata["jurisdiction_conflict"], "timestamp": time.Now().UTC().Format(time.RFC3339Nano)}
		if b, err := json.Marshal(decPayload); err == nil {
			ledgerInstance.Append("decision", string(b), "")
			if ledgerEventsCounter != nil { ledgerEventsCounter.WithLabelValues("decision").Inc() }
		}
		if span != nil { span.End() }

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"result": map[string]any{
				"action":               request.Action,
				"allowed":              allowed,
				"missing_capabilities": missing,
				"metadata":             metadata,
				"timestamp":            time.Now().Format(time.RFC3339),
			},
		})
	})

	// Minimal PoA issuance endpoint (demo-only; not a full RFC0111 implementation)
	router.POST("/demo/poa/issue", func(c *gin.Context) {
		var req struct {
			ID string `json:"id"`
			Grantor string `json:"grantor"`
			Grantee string `json:"grantee"`
			Scope []string `json:"scope"`
			ValidForSeconds int `json:"valid_for_seconds"`
			Jurisdiction string `json:"jurisdiction"`
			Witnesses []string `json:"witnesses"`
			Attestations []string `json:"attestations"`
		}
		if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
		if req.ID == "" { req.ID = fmt.Sprintf("poa-%d", time.Now().UnixNano()) }
		poa := &rfc0111.PowerOfAttorney{
			ID: req.ID,
			Version: 1,
			Grantor: req.Grantor,
			Grantee: req.Grantee,
			Scope: req.Scope,
			Restrictions: map[string]string{},
			ValidFrom: time.Now().UTC(),
			ValidUntil: time.Now().UTC().Add(time.Duration(req.ValidForSeconds) * time.Second),
			Status: rfc0111.POAStatusActive,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			Jurisdiction: req.Jurisdiction,
			Witnesses: req.Witnesses,
			Attestations: req.Attestations,
		}
		poaRepoMu.Lock(); if boltPOARepo != nil { _ = boltPOARepo.Create(poa) } else { poaRepo[poa.ID] = poa }; poaRepoMu.Unlock()
		if poaIssuanceCounter != nil { poaIssuanceCounter.Inc() }
		if poaStatusTransitionsCounter != nil { poaStatusTransitionsCounter.WithLabelValues("new", string(poa.Status)).Inc() }
		// Ledger anchoring for PoA issuance
		if b, err := json.Marshal(map[string]any{"event": "poa_issue", "poa_id": poa.ID, "grantor": poa.Grantor, "grantee": poa.Grantee, "scope": poa.Scope, "jurisdiction": poa.Jurisdiction, "valid_until": poa.ValidUntil}); err == nil {
			ledgerInstance.Append("poa_issue", string(b), "")
			if ledgerEventsCounter != nil { ledgerEventsCounter.WithLabelValues("poa_issue").Inc() }
		}
		appendAudit("poa_issue", poa.Grantor, poa.ID, map[string]any{"scope": poa.Scope, "jurisdiction": poa.Jurisdiction})
		digest, _, _ := rfc0111.CanonicalPOADigest(poa)
		c.JSON(200, gin.H{"poa": poa, "canonical_digest": digest})
	})

	// Multi-signature PoA preparation endpoint (creates draft PoA requiring threshold signatures)
	router.POST("/demo/poa/prepare", func(c *gin.Context) {
		var req struct {
			ID string `json:"id"`
			Grantor string `json:"grantor"`
			Grantee string `json:"grantee"`
			Scope []string `json:"scope"`
			Jurisdiction string `json:"jurisdiction"`
			Signers []string `json:"signers"`
			Threshold int `json:"threshold"`
			ValidForSeconds int `json:"valid_for_seconds"`
		}
		if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
		if req.Threshold <= 0 { req.Threshold = 1 }
		if req.ID == "" { req.ID = fmt.Sprintf("poa-draft-%d", time.Now().UnixNano()) }
		if len(req.Signers) < req.Threshold { c.JSON(400, gin.H{"error": "insufficient_signers", "have": len(req.Signers), "need": req.Threshold}); return }
		now := time.Now().UTC()
		poa := &rfc0111.PowerOfAttorney{ID: req.ID, Version: 1, Grantor: req.Grantor, Grantee: req.Grantee, Scope: req.Scope, Restrictions: map[string]string{}, ValidFrom: now, ValidUntil: now.Add(time.Duration(req.ValidForSeconds) * time.Second), Status: rfc0111.POAStatusDraft, CreatedAt: now, UpdatedAt: now, Jurisdiction: req.Jurisdiction, Signers: req.Signers, Threshold: req.Threshold, MultiSignatures: map[string]*rfc0111.POASignature{}}
		poaRepoMu.Lock(); if boltPOARepo != nil { _ = boltPOARepo.Create(poa) } else { poaRepo[poa.ID] = poa }; poaRepoMu.Unlock()
		if poaIssuanceCounter != nil { poaIssuanceCounter.Inc() }
		if poaStatusTransitionsCounter != nil { poaStatusTransitionsCounter.WithLabelValues("new", string(poa.Status)).Inc() }
		// Ledger anchoring for PoA draft preparation
		if b, err := json.Marshal(map[string]any{"event": "poa_prepare", "poa_id": poa.ID, "signers": poa.Signers, "threshold": poa.Threshold}); err == nil {
			ledgerInstance.Append("poa_prepare", string(b), "")
			if ledgerEventsCounter != nil { ledgerEventsCounter.WithLabelValues("poa_prepare").Inc() }
		}
		appendAudit("poa_prepare", poa.Grantor, poa.ID, map[string]any{"signers": poa.Signers, "threshold": poa.Threshold})
		c.JSON(200, gin.H{"poa": poa})
	})

	// Multi-signature signature submission endpoint
	router.POST("/demo/poa/:id/sign", func(c *gin.Context) {
		id := c.Param("id")
		var req struct { Signer string `json:"signer"` }
		if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
		poaRepoMu.Lock(); var poa *rfc0111.PowerOfAttorney; var ok bool
		if boltPOARepo != nil { poa, ok = boltPOARepo.Get(id) } else { poa, ok = poaRepo[id] }
		if !ok { poaRepoMu.Unlock(); c.JSON(404, gin.H{"error": "poa_not_found"}); return }
		if poa.Status != rfc0111.POAStatusDraft { poaRepoMu.Unlock(); c.JSON(400, gin.H{"error": "poa_not_draft", "status": poa.Status}); return }
		found := false
		for _, s := range poa.Signers { if s == req.Signer { found = true; break } }
		if !found { poaRepoMu.Unlock(); c.JSON(400, gin.H{"error": "unknown_signer"}); return }
		if poa.MultiSignatures == nil { poa.MultiSignatures = map[string]*rfc0111.POASignature{} }
		if _, exists := poa.MultiSignatures[req.Signer]; exists { poaRepoMu.Unlock(); c.JSON(409, gin.H{"error": "duplicate_signature"}); return }
		digest, canon, _ := rfc0111.CanonicalPOADigest(poa)
		// For demo we generate a dummy Ed25519-like signature placeholder (no crypto key resolution).
		randomSig := base64.StdEncoding.EncodeToString([]byte("demo-" + req.Signer + "-" + digest))
		poa.MultiSignatures[req.Signer] = &rfc0111.POASignature{Algorithm: "ed25519", KeyID: req.Signer + "-kid", DigestHex: digest, SigBase64: randomSig, Canonical: canon}
		poa.UpdatedAt = time.Now().UTC()
		// Update repo
		if boltPOARepo != nil { _ = boltPOARepo.Update(poa) } else { poaRepo[id] = poa }
		poaRepoMu.Unlock()
		if poaMultisigSignaturesCounter != nil { poaMultisigSignaturesCounter.WithLabelValues("accepted").Inc() }
		// Ledger anchoring for signature submission
		if b, err := json.Marshal(map[string]any{"event": "poa_sign", "poa_id": id, "signer": req.Signer, "current_signatures": len(poa.MultiSignatures)}); err == nil {
			ledgerInstance.Append("poa_sign", string(b), "")
			if ledgerEventsCounter != nil { ledgerEventsCounter.WithLabelValues("poa_sign").Inc() }
		}
		appendAudit("poa_sign", req.Signer, id, map[string]any{"current_signatures": len(poa.MultiSignatures)})
		c.JSON(200, gin.H{"signed": true, "poa_id": id, "signatures": len(poa.MultiSignatures)})
	})

	// Multi-signature PoA status endpoint
	router.GET("/demo/poa/:id/status", func(c *gin.Context) {
		id := c.Param("id")
		poaRepoMu.RLock(); var poa *rfc0111.PowerOfAttorney; var ok bool
		if boltPOARepo != nil { poa, ok = boltPOARepo.Get(id) } else { poa, ok = poaRepo[id] }
		poaRepoMu.RUnlock()
		if !ok { c.JSON(404, gin.H{"error": "poa_not_found"}); return }
		c.JSON(200, gin.H{"poa_id": id, "status": poa.Status, "signatures": len(poa.MultiSignatures), "threshold": poa.Threshold})
	})

	// Multi-signature finalize endpoint (activates draft if threshold signatures collected)
	router.POST("/demo/poa/:id/finalize", func(c *gin.Context) {
		id := c.Param("id")
		poaRepoMu.Lock(); var poa *rfc0111.PowerOfAttorney; var ok bool
		if boltPOARepo != nil { poa, ok = boltPOARepo.Get(id) } else { poa, ok = poaRepo[id] }
		if !ok { poaRepoMu.Unlock(); c.JSON(404, gin.H{"error": "poa_not_found"}); return }
		if poa.Status != rfc0111.POAStatusDraft { poaRepoMu.Unlock(); c.JSON(400, gin.H{"error": "not_draft", "status": poa.Status}); return }
		if len(poa.MultiSignatures) < poa.Threshold { poaRepoMu.Unlock(); c.JSON(400, gin.H{"error": "threshold_not_met", "current": len(poa.MultiSignatures), "need": poa.Threshold}); return }
		// Mark active
		oldStatus := poa.Status
		poa.Status = rfc0111.POAStatusActive
		poa.UpdatedAt = time.Now().UTC()
		if boltPOARepo != nil { _ = boltPOARepo.Update(poa) } else { poaRepo[id] = poa }
		poaRepoMu.Unlock()
		if poaMultisigFinalizationsCounter != nil { poaMultisigFinalizationsCounter.Inc() }
		if poaStatusTransitionsCounter != nil { poaStatusTransitionsCounter.WithLabelValues(string(oldStatus), string(poa.Status)).Inc() }
		if poaIssuanceCounter != nil { poaIssuanceCounter.Inc() } // treat finalization as issuance
		// Ledger anchoring for PoA finalization
		if b, err := json.Marshal(map[string]any{"event": "poa_finalize", "poa_id": id, "status": poa.Status}); err == nil {
			ledgerInstance.Append("poa_finalize", string(b), "")
			if ledgerEventsCounter != nil { ledgerEventsCounter.WithLabelValues("poa_finalize").Inc() }
		}
		appendAudit("poa_finalize", poa.Grantor, id, map[string]any{"status": poa.Status})
		c.JSON(200, gin.H{"finalized": true, "poa_id": id, "status": poa.Status})
	})

	// --- Ledger endpoints (Merkle) ---
	router.POST("/demo/ledger/append", func(c *gin.Context) {
		var req struct { Type string `json:"type"`; Payload string `json:"payload"`; Digest string `json:"digest"` }
		if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
		if req.Type == "" { c.JSON(400, gin.H{"error": "missing_type"}); return }
		var id int; var root string; var errA error
		if boltLedgerInstance != nil { id, root, errA = boltLedgerInstance.Append(req.Type, req.Payload, req.Digest); if errA != nil { c.JSON(500, gin.H{"error": errA.Error()}); return } } else { id, root = ledgerInstance.Append(req.Type, req.Payload, req.Digest) }
		if ledgerAppendsCounter != nil { ledgerAppendsCounter.WithLabelValues(req.Type).Inc() }
		if ledgerRootEmissionsCounter != nil { ledgerRootEmissionsCounter.Inc() }
		c.JSON(200, gin.H{"appended": true, "entry_id": id, "root": root})
	})

	router.GET("/demo/ledger/root/latest", func(c *gin.Context) {
		var root string
		if boltLedgerInstance != nil { roots := boltLedgerInstance.HistoricalRoots(); if len(roots) > 0 { root = roots[len(roots)-1] } } else { root = ledgerInstance.LatestRoot() }
		c.JSON(200, gin.H{"root": root})
	})

	router.GET("/demo/ledger/entry/:id/proof", func(c *gin.Context) {
		idStr := c.Param("id")
		i, err := strconv.Atoi(idStr)
		if err != nil { c.JSON(400, gin.H{"error": "invalid_id"}); return }
		var proof ledger.Proof; var perr error
		if boltLedgerInstance != nil { proof, perr = boltLedgerInstance.Proof(i) } else { proof, perr = ledgerInstance.Proof(i) }
		if perr != nil { c.JSON(404, gin.H{"error": perr.Error()}); return }
		if ledgerProofRequestsCounter != nil { ledgerProofRequestsCounter.Inc() }
		c.JSON(200, gin.H{"proof": proof})
	})

	// --- Audit Ledger (hash-chain) endpoints (global instance) ---
	if auditLedgerInstance != nil {
		router.GET("/demo/audit/ledger", func(c *gin.Context) {
			limit := 100
			if ls := c.Query("limit"); ls != "" { if v, err := strconv.Atoi(ls); err == nil && v > 0 && v <= 10000 { limit = v } }
			entries, err := auditLedgerInstance.List(limit)
			if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
			if auditLedgerSizeGauge != nil { auditLedgerSizeGauge.Set(float64(auditLedgerInstance.Size())) }
			c.JSON(200, gin.H{"entries": entries, "count": len(entries), "limit": limit, "head": auditLedgerInstance.HeadHash(), "size": auditLedgerInstance.Size()})
		})
		router.GET("/demo/audit/ledger/:id", func(c *gin.Context) {
			idStr := c.Param("id"); i, err := strconv.Atoi(idStr); if err != nil { c.JSON(400, gin.H{"error": "invalid_id"}); return }
			entry, ok := auditLedgerInstance.Get(i); if !ok { c.JSON(404, gin.H{"error": "not_found"}); return }
			c.JSON(200, gin.H{"entry": entry})
		})
		router.GET("/demo/audit/ledger/export", func(c *gin.Context) {
			format := strings.ToLower(c.Query("format")); if format == "" { format = "ndjson" }
			limit := 1000
			if ls := c.Query("limit"); ls != "" { if v, err := strconv.Atoi(ls); err == nil && v > 0 && v <= 50000 { limit = v } }
			entries, err := auditLedgerInstance.List(limit)
			if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
			switch format {
			case "csv":
				c.Header("Content-Type", "text/csv; charset=utf-8")
				c.Writer.Write([]byte("id,at,type,poa_id,actor,prev_hash,entry_hash\n"))
				for _, e := range entries {
					line := fmt.Sprintf("%d,%s,%s,%s,%s,%s,%s\n", e.ID, e.At.Format(time.RFC3339Nano), e.Type, escapeCSV(e.POAID), escapeCSV(e.Actor), e.PrevHash, e.EntryHash)
					c.Writer.Write([]byte(line))
				}
			case "ndjson":
				c.Header("Content-Type", "application/x-ndjson")
				for _, e := range entries { b, _ := json.Marshal(e); c.Writer.Write(b); c.Writer.Write([]byte("\n")) }
			default:
				c.JSON(400, gin.H{"error": "unsupported_format", "supported": []string{"ndjson","csv"}}); return
			}
		})
		router.GET("/demo/audit/ledger/verify", func(c *gin.Context) {
			idx, err := auditLedgerInstance.VerifyChain()
			status := "ok"
			if err != nil { status = err.Error() }
			c.JSON(200, gin.H{"status": status, "mismatch_entry_id": idx, "head": auditLedgerInstance.HeadHash(), "size": auditLedgerInstance.Size()})
		})
	}

	// PoA revocation endpoint
	router.POST("/demo/poa/:id/revoke", func(c *gin.Context) {
		id := c.Param("id")
		reason := c.Query("reason")
		poaRepoMu.Lock(); var poa *rfc0111.PowerOfAttorney; var ok bool
		if boltPOARepo != nil { poa, ok = boltPOARepo.Get(id) } else { poa, ok = poaRepo[id] }
		if ok {
			if poa.Status != rfc0111.POAStatusRevoked {
				oldStatus := poa.Status
				poa.Status = rfc0111.POAStatusRevoked
				now := time.Now().UTC(); poa.RevokedAt = &now; poa.RevocationReason = reason
				poa.UpdatedAt = now
				if boltPOARepo != nil { _ = boltPOARepo.Update(poa) } else { poaRepo[id] = poa }
				if poaRevocationsCounter != nil { poaRevocationsCounter.Inc() }
				if poaRevocationReasonCounter != nil { poaRevocationReasonCounter.WithLabelValues(reason).Inc() }
				if poaStatusTransitionsCounter != nil { poaStatusTransitionsCounter.WithLabelValues(string(oldStatus), string(poa.Status)).Inc() }
				// Ledger anchoring for revocation
				if b, err := json.Marshal(map[string]any{"event": "poa_revoke", "poa_id": id, "reason": reason}); err == nil {
					ledgerInstance.Append("poa_revoke", string(b), "")
					if ledgerEventsCounter != nil { ledgerEventsCounter.WithLabelValues("poa_revoke").Inc() }
				}
				appendAudit("poa_revoke", poa.Grantor, id, map[string]any{"reason": reason})
			}
		}
		poaRepoMu.Unlock()
		if !ok { c.JSON(404, gin.H{"error": "poa_not_found"}); return }
		c.JSON(200, gin.H{"revoked": true, "poa_id": id, "status": poa.Status, "revoked_at": poa.RevokedAt, "reason": poa.RevocationReason})
	})

	// PoA suspend endpoint (temporary hold; only active PoAs may be suspended)
	router.POST("/demo/poa/:id/suspend", func(c *gin.Context) {
		id := c.Param("id")
		reason := c.Query("reason")
		poaRepoMu.Lock(); var poa *rfc0111.PowerOfAttorney; var ok bool
		if boltPOARepo != nil { poa, ok = boltPOARepo.Get(id) } else { poa, ok = poaRepo[id] }
		if ok {
			if poa.Status == rfc0111.POAStatusActive { // only allow suspend from active
				oldStatus := poa.Status
				poa.Status = rfc0111.POAStatusSuspended
				poa.UpdatedAt = time.Now().UTC()
				if boltPOARepo != nil { _ = boltPOARepo.Update(poa) } else { poaRepo[id] = poa }
				if poaStatusTransitionsCounter != nil { poaStatusTransitionsCounter.WithLabelValues(string(oldStatus), string(poa.Status)).Inc() }
				// Ledger anchoring
				if b, err := json.Marshal(map[string]any{"event": "poa_suspend", "poa_id": id, "reason": reason, "from": oldStatus, "to": poa.Status}); err == nil {
					ledgerInstance.Append("poa_suspend", string(b), "")
					if ledgerEventsCounter != nil { ledgerEventsCounter.WithLabelValues("poa_suspend").Inc() }
				}
				appendAudit("poa_suspend", poa.Grantor, id, map[string]any{"reason": reason, "from": oldStatus, "to": poa.Status})
			} // else ignore if not active
		}
		poaRepoMu.Unlock()
		if !ok { c.JSON(404, gin.H{"error": "poa_not_found"}); return }
		c.JSON(200, gin.H{"suspended": poa.Status == rfc0111.POAStatusSuspended, "poa_id": id, "status": poa.Status, "reason": reason})
	})

	// PoA terminate endpoint (permanent; allow from active or suspended; cannot change revoked/expired/terminated)
	router.POST("/demo/poa/:id/terminate", func(c *gin.Context) {
		id := c.Param("id")
		reason := c.Query("reason")
		poaRepoMu.Lock(); var poa *rfc0111.PowerOfAttorney; var ok bool
		if boltPOARepo != nil { poa, ok = boltPOARepo.Get(id) } else { poa, ok = poaRepo[id] }
		if ok {
			if poa.Status == rfc0111.POAStatusActive || poa.Status == rfc0111.POAStatusSuspended {
				oldStatus := poa.Status
				poa.Status = rfc0111.POAStatusTerminated
				poa.UpdatedAt = time.Now().UTC()
				if boltPOARepo != nil { _ = boltPOARepo.Update(poa) } else { poaRepo[id] = poa }
				if poaStatusTransitionsCounter != nil { poaStatusTransitionsCounter.WithLabelValues(string(oldStatus), string(poa.Status)).Inc() }
				if b, err := json.Marshal(map[string]any{"event": "poa_terminate", "poa_id": id, "reason": reason, "from": oldStatus, "to": poa.Status}); err == nil {
					ledgerInstance.Append("poa_terminate", string(b), "")
					if ledgerEventsCounter != nil { ledgerEventsCounter.WithLabelValues("poa_terminate").Inc() }
				}
				appendAudit("poa_terminate", poa.Grantor, id, map[string]any{"reason": reason, "from": oldStatus, "to": poa.Status})
			}
		}
		poaRepoMu.Unlock()
		if !ok { c.JSON(404, gin.H{"error": "poa_not_found"}); return }
		c.JSON(200, gin.H{"terminated": poa.Status == rfc0111.POAStatusTerminated, "poa_id": id, "status": poa.Status, "reason": reason})
	})

	// Minimal extended token issuance referencing a PoA (HS256 signed JWT). Requires GAUTH_AI_DEMO_JWT_SECRET.
	router.POST("/demo/poa/:id/token", func(c *gin.Context) {
		secret := os.Getenv("GAUTH_AI_DEMO_JWT_SECRET")
		if secret == "" { c.JSON(400, gin.H{"error": "jwt_secret_not_configured"}); return }
		id := c.Param("id")
		poaRepoMu.RLock(); var poa *rfc0111.PowerOfAttorney; var ok bool
		if boltPOARepo != nil { poa, ok = boltPOARepo.Get(id) } else { poa, ok = poaRepo[id] }
		poaRepoMu.RUnlock()
		if !ok { c.JSON(404, gin.H{"error": "poa_not_found"}); return }
		if poa.Status != rfc0111.POAStatusActive { c.JSON(400, gin.H{"error": "poa_not_active", "status": poa.Status}); return }
		now := time.Now().UTC()
		exp := poa.ValidUntil
		if exp.Sub(now) > 2*time.Hour { // cap token lifetime to 2h for demo safety
			exp = now.Add(2 * time.Hour)
		}
		digest, _, _ := rfc0111.CanonicalPOADigest(poa)
		claims := map[string]any{
			"sub": poa.Grantee,
			"iss": "gauth-demo",
			"aud": "gauth-demo-api",
			"iat": now.Unix(),
			"nbf": now.Unix(),
			"exp": exp.Unix(),
			"poa_id": poa.ID,
			"poa_digest": digest,
			"poa_version": poa.Version,
			"token_version": "et_v1",
		}
		// Encode header + payload
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		payloadBytes, _ := json.Marshal(claims)
		payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
		toSign := header + "." + payload
		sig := hmacSHA256Base64URL(toSign, secret)
		token := toSign + "." + sig
		// Ledger anchoring + audit ledger for token issuance
		if b, err := json.Marshal(map[string]any{"event": "poa_token_issue", "poa_id": poa.ID, "token_version": "et_v1", "poa_version": poa.Version, "expires_at": exp.Format(time.RFC3339)}); err == nil {
			ledgerInstance.Append("poa_token_issue", string(b), "")
			if ledgerEventsCounter != nil { ledgerEventsCounter.WithLabelValues("poa_token_issue").Inc() }
		}
		appendAudit("poa_token_issue", poa.Grantee, poa.ID, map[string]any{"expires_at": exp.Format(time.RFC3339), "token_version": "et_v1"})
		c.JSON(200, gin.H{"token": token, "poa_id": poa.ID, "poa_digest": digest, "poa_version": poa.Version, "token_version": "et_v1", "expires_at": exp.Format(time.RFC3339)})
	})

	// List persisted decisions (if DB enabled)
	router.GET("/demo/decisions", func(c *gin.Context) {
		if db == nil { c.JSON(200, gin.H{"decisions": []any{}, "persistence": "disabled"}); return }
		limitStr := c.Query("limit")
		offsetStr := c.Query("offset")
		actionFilter := c.Query("action")
		entityFilter := c.Query("entity_type")
		limit := 50
		if limitStr != "" { if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 500 { limit = v } }
		offset := 0
		if offsetStr != "" { if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 { offset = v } }
		base := `SELECT id, action, allowed, entity_type, jurisdictions, missing, applied_policies, poa_id, poa_digest, poa_version, created_at FROM decisions`
		var where []string
		var args []any
		if actionFilter != "" { where = append(where, "action = ?"); args = append(args, actionFilter) }
		if entityFilter != "" { where = append(where, "entity_type = ?"); args = append(args, entityFilter) }
		if len(where) > 0 { base += " WHERE " + strings.Join(where, " AND ") }
		base += " ORDER BY id DESC LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
		rows, err := db.Query(base, args...)
		if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
		defer rows.Close()
		var out []map[string]any
		for rows.Next() {
			var id, allowed int
			var action, entityType, jurisdictions, missing, applied, poaID, poaDigest, created string
			var poaVersion sql.NullInt64
			if err := rows.Scan(&id, &action, &allowed, &entityType, &jurisdictions, &missing, &applied, &poaID, &poaDigest, &poaVersion, &created); err != nil { continue }
			out = append(out, map[string]any{
				"id": id,
				"action": action,
				"allowed": allowed == 1,
				"entity_type": entityType,
				"jurisdictions": jurisdictions,
				"missing": strings.Split(strings.TrimSpace(missing), ","),
				"applied_policies": applied,
				"poa_id": poaID,
				"poa_digest": poaDigest,
				"created_at": created,
			})
		}
		c.JSON(200, gin.H{"decisions": out, "count": len(out), "limit": limit, "offset": offset, "action_filter": actionFilter, "entity_type_filter": entityFilter})
	})

	// Export decisions as NDJSON or CSV
	router.GET("/demo/decisions/export", func(c *gin.Context) {
		if db == nil { c.JSON(200, gin.H{"error": "persistence_disabled"}); return }
		format := strings.ToLower(c.Query("format"))
		if format == "" { format = "ndjson" }
		limitStr := c.Query("limit")
		limit := 1000
		if limitStr != "" { if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100000 { limit = v } }
		rows, err := db.Query(`SELECT id, action, allowed, entity_type, jurisdictions, missing, applied_policies, poa_id, poa_digest, poa_version, created_at FROM decisions ORDER BY id DESC LIMIT ?`, limit)
		if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
		defer rows.Close()
			type rec struct {
				ID int `json:"id"`
				Action string `json:"action"`
				Allowed bool `json:"allowed"`
				EntityType string `json:"entity_type"`
				Jurisdictions string `json:"jurisdictions"`
				Missing string `json:"missing"`
				AppliedPolicies string `json:"applied_policies"`
				POAID string `json:"poa_id"`
				POADigest string `json:"poa_digest"`
				POAVersion int `json:"poa_version"`
				CreatedAt string `json:"created_at"`
			}
		var records []rec
		for rows.Next() {
			var r rec; var allowedInt int
			if err := rows.Scan(&r.ID, &r.Action, &allowedInt, &r.EntityType, &r.Jurisdictions, &r.Missing, &r.AppliedPolicies, &r.POAID, &r.POADigest, &r.POAVersion, &r.CreatedAt); err == nil {
				r.Allowed = allowedInt == 1
				records = append(records, r)
			}
		}
		switch format {
		case "csv":
			c.Header("Content-Type", "text/csv; charset=utf-8")
			c.Writer.Write([]byte("id,action,allowed,entity_type,jurisdictions,missing,applied_policies,poa_id,poa_digest,poa_version,created_at\n"))
			for _, r := range records {
				line := fmt.Sprintf("%d,%s,%t,%s,%s,%s,%s,%s,%s,%d,%s\n", r.ID, r.Action, r.Allowed, r.EntityType, escapeCSV(r.Jurisdictions), escapeCSV(r.Missing), escapeCSV(r.AppliedPolicies), escapeCSV(r.POAID), escapeCSV(r.POADigest), r.POAVersion, r.CreatedAt)
				c.Writer.Write([]byte(line))
			}
		case "ndjson":
			c.Header("Content-Type", "application/x-ndjson")
			for _, r := range records {
				b, _ := json.Marshal(r)
				c.Writer.Write(b); c.Writer.Write([]byte("\n"))
			}
		default:
			c.JSON(400, gin.H{"error": "unsupported_format", "supported": []string{"ndjson","csv"}}); return
		}
	})

	// Decision log statistics endpoint
	router.GET("/demo/decisions/stats", func(c *gin.Context) {
		if db == nil { c.JSON(200, gin.H{"persistence": "disabled"}); return }
		var total int
		_ = db.QueryRow(`SELECT COUNT(*) FROM decisions`).Scan(&total)
		var oldest, newest string
		_ = db.QueryRow(`SELECT created_at FROM decisions ORDER BY id ASC LIMIT 1`).Scan(&oldest)
		_ = db.QueryRow(`SELECT created_at FROM decisions ORDER BY id DESC LIMIT 1`).Scan(&newest)
		rows, err := db.Query(`SELECT action, COUNT(*) as c FROM decisions GROUP BY action ORDER BY c DESC LIMIT 5`)
		var top []map[string]any
		if err == nil {
			defer rows.Close()
			for rows.Next() { var a string; var cCount int; if err2 := rows.Scan(&a, &cCount); err2 == nil { top = append(top, map[string]any{"action": a, "count": cCount}) } }
		}
		c.JSON(200, gin.H{"total": total, "oldest": oldest, "newest": newest, "top_actions": top})
	})

	// Conflict simulation dedicated endpoint
	router.POST("/api/v1/ai/capabilities/simulate/conflict", func(c *gin.Context) {
		var req struct {
			Action string `json:"action"`
			Claims map[string]any `json:"claims"`
			Jurisdictions []string `json:"jurisdictions"`
		}
		if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
		if len(req.Jurisdictions) < 2 { c.JSON(400, gin.H{"error": "need >=2 jurisdictions"}); return }
		origJur := fmt.Sprintf("%v", req.Claims["jurisdiction"])
		decisions := map[string]bool{}
		ctx := context.Background(); var parentSpan trace.Span
		if tracer != nil { ctx, parentSpan = tracer.Start(ctx, "simulate.conflict.batch") }
		batchStart := time.Now()
		for _, j := range req.Jurisdictions {
			req.Claims["jurisdiction"] = j
			var childSpan trace.Span
			if tracer != nil { ctx, childSpan = tracer.Start(ctx, "simulate.conflict.jurisdiction", trace.WithAttributes(attribute.String("jurisdiction", j))) }
			ok, _, _ := integration.EnforceAICapabilities(req.Action, req.Claims)
			decisions[j] = ok
			if childSpan != nil {
				childSpan.SetAttributes(attribute.Bool("allowed", ok))
				childSpan.End()
			}
			if parentSpan != nil { parentSpan.SetAttributes(attribute.String("jurisdiction", j), attribute.Bool("allowed", ok)) }
		}
		req.Claims["jurisdiction"] = origJur
		var firstVal *bool
		conflict := false
		for _, v := range decisions { if firstVal == nil { first := v; firstVal = &first; continue }; if v != *firstVal { conflict = true; break } }
		if conflict { integration.GetMetricsCallback()("jurisdiction_conflict") }
		if conflictDuration != nil { conflictDuration.Observe(time.Since(batchStart).Seconds()) }
		if conflict && conflictCounter != nil { conflictCounter.Inc() }
		if parentSpan != nil { parentSpan.SetAttributes(attribute.Bool("conflict", conflict)); parentSpan.End() }
		c.JSON(200, gin.H{"success": true, "action": req.Action, "decisions": decisions, "conflict": conflict})
	})

	// API documentation
	router.GET("/api/docs", func(c *gin.Context) {
		docs := apiHandler.GetAPIDocumentation()
		c.JSON(http.StatusOK, docs)
	})

	fmt.Println("🚀 Server starting on http://localhost:" + port)
	fmt.Println("\nExample API calls:")
	fmt.Println("==================")
	fmt.Println("curl http://localhost:8080/api/v1/ai/capabilities/status")
	fmt.Println("curl http://localhost:8080/api/v1/ai/capabilities/entity-types")
	fmt.Println("curl http://localhost:8080/api/v1/ai/health")
	fmt.Println("")
	fmt.Println("Demo enforcement:")
	fmt.Println("curl -X POST http://localhost:8080/demo/enforce \\")
	fmt.Println("  -H 'Content-Type: application/json' \\")
	fmt.Println("  -d '{\"action\":\"transaction:read\",\"claims\":{\"ai_entity_type\":\"assistant\",\"ai_entity_verified\":true,\"algorithmic_accountability\":true}}'")
	fmt.Println("")
	fmt.Println("Press Ctrl+C to stop the server")

	log.Fatal(http.ListenAndServe(":"+port, router))
}

// escapeCSV performs minimal escaping (wrap in quotes if comma present, replace quotes with doubled quotes)
func escapeCSV(s string) string {
	// characters requiring quoting: comma, newline, double quote
	if strings.ContainsAny(s, ",\n\"") {
		esc := strings.ReplaceAll(s, "\"", "\"\"")
		return "\"" + esc + "\""
	}
	return s
}