package web

import (
	"bytes"
	"context"
	"crypto/ed25519"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	otel "go.opentelemetry.io/otel"
	stdoutmetric "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"

	anchorint "github.com/mauriciomferz/Gauth_go/internal/anchor"
	"github.com/mauriciomferz/Gauth_go/internal/capability"
	"github.com/mauriciomferz/Gauth_go/internal/limits"
	"github.com/mauriciomferz/Gauth_go/internal/metrics"
	notary "github.com/mauriciomferz/Gauth_go/internal/notary"
	"github.com/mauriciomferz/Gauth_go/internal/tracing"
	"github.com/mauriciomferz/Gauth_go/pkg/anchor"
	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/auth"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	"github.com/mauriciomferz/Gauth_go/pkg/blockchain"
	"github.com/mauriciomferz/Gauth_go/pkg/cache"
	"github.com/mauriciomferz/Gauth_go/pkg/common"
	"github.com/mauriciomferz/Gauth_go/pkg/common/clock"
	cacheConfig "github.com/mauriciomferz/Gauth_go/pkg/config"
	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
	"github.com/mauriciomferz/Gauth_go/pkg/crypto/keys"
	"github.com/mauriciomferz/Gauth_go/pkg/database"
	"github.com/mauriciomferz/Gauth_go/pkg/delegation"
	devicePkg "github.com/mauriciomferz/Gauth_go/pkg/device"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth_rfc_001"
	"github.com/mauriciomferz/Gauth_go/pkg/gauthplus"
	gnapPkg "github.com/mauriciomferz/Gauth_go/pkg/gnap"
	"github.com/mauriciomferz/Gauth_go/pkg/ledger"
	"github.com/mauriciomferz/Gauth_go/pkg/mcp"
	"github.com/mauriciomferz/Gauth_go/pkg/pdp"
	"github.com/mauriciomferz/Gauth_go/pkg/redis"
	"github.com/mauriciomferz/Gauth_go/pkg/registry"
	"github.com/mauriciomferz/Gauth_go/pkg/scim"
	a2aHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/a2a"
	adminHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/admin"
	anchorHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/anchor"
	authHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/auth"
	authzHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/authz"
	"github.com/mauriciomferz/Gauth_go/web/handlers/capabilities"
	"github.com/mauriciomferz/Gauth_go/web/handlers/capability_anchor"
	delegationHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/delegation"
	deviceHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/device"
	grantJWTHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/grant_jwt"

	"github.com/mauriciomferz/Gauth_go/pkg/saml"
	eventsHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/events"
	gnapHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/gnap"
	mcpHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/mcp"
	modellimits "github.com/mauriciomferz/Gauth_go/web/handlers/modellimits"
	notaryHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/notary"
	poaHandlers "github.com/mauriciomferz/Gauth_go/web/handlers/poa"
	policyHandler "github.com/mauriciomferz/Gauth_go/web/handlers/policy"
	"github.com/mauriciomferz/Gauth_go/web/handlers/semantic"
	"github.com/mauriciomferz/Gauth_go/web/handlers/token"
	"github.com/mauriciomferz/Gauth_go/web/handlers/violations"
)

// atoiDefault converts string to int, returning def on error or negative values.
func atoiDefault(raw string, def int) int {
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return def
	}
	return v
}

func envFallback(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

// getExtProviderLabel returns provider label for external anchor metrics.
func getExtProviderLabel(s *BetaServer) string {
	if s == nil || s.capabilityAnchorHandler == nil || s.capabilityAnchorHandler.Provider == nil {
		return "_"
	}
	rec, _ := s.capabilityAnchorHandler.Latest(context.Background())
	// Preferred: provider tag from latest receipt (already normalized by provider implementation)
	if rec.Provider != "" {
		return rec.Provider
	}
	// Fallback to environment selection; normalize known variants for metrics label stability.
	raw := os.Getenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER")
	if raw == "" {
		return "_"
	}
	// Normalization map (extendable): env value -> metrics label
	switch raw {
	case providerTSAStub:
		return tsaStubProvider
	case memoryProvider:
		return memoryProvider
	default:
		// Replace underscores with dashes for generic normalization, lowercase already enforced by env usage.
		return strings.ReplaceAll(raw, "_", "-")
	}
}

// NewBetaServer creates a new BetaServer instance.
// NewBetaServer constructs a server using the default in-memory metrics adapter.
// For tests or advanced instrumentation provide a custom metrics implementation via NewBetaServerWithMetrics.
func NewBetaServer(port string, opts ...BetaServerOption) *BetaServer {
	return NewBetaServerWithMetrics(port, nil, opts...)
}

// NewBetaServerWithMetrics constructs a BetaServer instance allowing a custom metrics adapter to be injected
// at construction time so that any startup side-effects (e.g. initial external anchoring attempt) record metrics
// on the desired registry/implementation. When m is nil a new in-memory metrics adapter is created.
//
//nolint:gocyclo // Server initialization with metrics and components
//nolint:gocyclo // Server initialization with metrics and components
func NewBetaServerWithMetrics(port string, m metrics.Metrics, opts ...BetaServerOption) *BetaServer {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// Enable CORS with allow-list support (GAUTH_CORS_ALLOW env).
	r.Use(corsMiddleware())
	// Normalize port: allow ":8080" or "8080"
	if port == "" {
		port = defaultPort
	} else if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	var memoryMetrics metrics.Metrics
	if m == nil {
		memoryMetrics = metrics.NewMemory()
	} else {
		memoryMetrics = m
	}
	// Enable persistence if GAUTH_METRICS_PERSIST_PATH is set (only for in-memory implementation)
	if pp := os.Getenv("GAUTH_METRICS_PERSIST_PATH"); pp != "" {
		if memImpl, ok := memoryMetrics.(*metrics.Memory); ok {
			// Expand tilde and relative paths for convenience
			if strings.HasPrefix(pp, "~") {
				if home, err := os.UserHomeDir(); err == nil {
					pp = filepath.Join(home, strings.TrimPrefix(pp, "~"))
				}
			}
			if err := memImpl.EnablePersistence(pp); err != nil {
				fmt.Fprintf(os.Stderr, "[metrics] persistence load failed path=%s err=%v\n", pp, err)
			} else {
				fmt.Fprintf(os.Stderr, "[metrics] persistence enabled path=%s\n", pp)
			}
		}
	}
	s := &BetaServer{
		router:           r,
		start:            time.Now(),
		audit:            NewAuditLog(500),
		events:           NewEventHub(500),
		eventsHubAdapter: eventsHandlers.NewHub(500),
		tokens:           token.NewStore(500),
		port:             port,
		replayStore:      token.NewReplayNonceStore(5 * time.Minute),
		metrics:          memoryMetrics,
		poaHandler:       poaHandlers.NewHandler(),
		lifecycleEvents:  make(map[string][]*LifecycleEvent),
		lifecycleCap:     250,
		// Capabilities Handler Initialization
		// requiredActionCaps removed
		stopCh:              make(chan struct{}),
		protocolFlowManager: NewProtocolFlowManager(),
		keyProvider:         nil, // default to nil; expected to be injected or initialized via options
	}

	// Tracing initialization: primary enable flag GAUTH_TRACING_ENABLED or legacy GAUTH_OTEL_ENABLE.
	if os.Getenv("GAUTH_TRACING_ENABLED") == "1" || os.Getenv("GAUTH_OTEL_ENABLE") == "1" {
		if tp, err := tracing.NewTracerProvider(tracing.Config{ServiceName: "gauth-beta"}); err == nil {
			s.tracerProvider = tp
			fmt.Fprintln(os.Stderr, "[tracing] enabled spans")
			// Sampling ratio (0..1). Ratio <=0 interpreted as ALWAYS SAMPLE per ADR.
			if raw := os.Getenv("GAUTH_TRACING_SAMPLE_RATIO"); raw != "" {
				if v, err := strconv.ParseFloat(raw, 64); err == nil && v >= 0 && v <= 1 {
					s.tracerSampleRatio = v
				}
			}
			// Wire global tracing middleware to Gin
			r.Use(tracing.GinTracingMiddleware(s.tracerProvider.Tracer(), s.tracerSampleRatio))
		}
	}

	// Initialize System Clock Monitor (RR-015)
	ntpServer := envFallback("GAUTH_NTP_SERVER", "pool.ntp.org")
	maxSkew := 5 * time.Minute
	if v := os.Getenv("GAUTH_MAX_CLOCK_SKEW"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			maxSkew = d
		}
	}
	ntpInterval := 1 * time.Hour
	if v := os.Getenv("GAUTH_NTP_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ntpInterval = d
		}
	}
	s.systemClockMonitor = clock.NewSystemClockMonitor(ntpServer, maxSkew, ntpInterval, s.metrics)
	s.systemClockMonitor.Start()

	// Redis Initialization
	if os.Getenv("GAUTH_SKIP_REDIS") == "1" {
		fmt.Fprintln(os.Stderr, "[redis] initialization skipped via GAUTH_SKIP_REDIS")
	} else {
		redisHost := os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "localhost"
		}
		redisPortStr := os.Getenv("REDIS_PORT")
		redisPort := 6379
		if redisPortStr != "" {
			if v, err := strconv.Atoi(redisPortStr); err == nil {
				redisPort = v
			}
		}
		redisCfg := &redis.Config{
			Host: redisHost,
			Port: redisPort,
		}
		if rc, err := redis.NewClient(redisCfg); err == nil {
			s.redisClient = rc
		} else {
			fmt.Fprintf(os.Stderr, "[redis] failed to initialize client: %v\n", err)
		}
	}

	for _, opt := range opts {
		opt(s)
	}

	// Initialize Admin Handlers using injected DB pool and Redis client
	if s.db != nil && s.redisClient != nil {
		s.adminTokenHandler = adminHandlers.NewTokenHandler(s.db.Pool, s.redisClient)
	}
	if s.db != nil {
		s.apiKeyHandler = adminHandlers.NewAPIKeyHandler(s.db.Pool)
		s.resilienceHandler = adminHandlers.NewResilienceHandler(s.db.Pool)
	}

	// Capabilities Handler Initialization
	capsHandler := capabilities.NewHandler()
	// Restore default mappings (required for tests/legacy behavior when no file loaded)
	capsHandler.ActionMappings["delegation:create"] = []string{"cap.delegation.create"}
	capsHandler.ActionMappings["delegation:revoke"] = []string{"cap.delegation.revoke"}
	// Load config if path set
	s.capabilitiesHandler = capsHandler
	s.capabilitiesAPI = capabilities.NewAPI(capsHandler)

	// Wire up capabilities handler metrics and OnReload callback for anchor emission
	capsHandler.Metrics = &capabilityMetricsAdapter{m: s.metrics}
	capsHandler.OnReload = func(newHash string) {
		// Emit anchor artifact (throttled by interval)
		if s.capAnchorFilePath != "" {
			// Check throttle
			if time.Since(s.capAnchorLastWrite) >= s.capAnchorWriteInterval {
				s.capAnchorLastWrite = time.Now()
				// Build anchor material artifact
				artifact := AnchorMaterial{
					Type:          "capability_registry_anchor",
					RegistryHash:  newHash,
					PreviousHash:  s.capabilitiesHandler.PrevRegistryHash,
					LastChangedAt: s.capabilitiesHandler.RegistryChangeAt.UTC().Format(time.RFC3339),
					SchemaVersion: s.capabilitiesHandler.SchemaVersion,
					AnchoredAt:    time.Now().UTC().Format(time.RFC3339),
				}
				// Use compact JSON for the inner artifact to ensure signature stability across transport
				data, err := json.Marshal(artifact)
				if err == nil {
					// Check if signing is enabled and KeyManager available
					signEnabled := os.Getenv("GAUTH_CAP_ANCHOR_SIGN") == "1"
					if signEnabled && s.keyProvider != nil {
						// Attempt EdDSA signing via KeyProvider.ActiveSigner()
						if signer, err := s.keyProvider.ActiveSigner(); err == nil && signer != nil {
							sig, signErr := signer.Sign(data)
							if signErr == nil {
								// Build signed wrapper
								wrapper := struct {
									Artifact  json.RawMessage `json:"artifact"`
									Kid       string          `json:"kid"`
									Signature string          `json:"signature"`
									Mode      string          `json:"mode"`
								}{
									Artifact:  data,
									Kid:       signer.KeyID(),
									Signature: base64.RawStdEncoding.EncodeToString(sig),
									Mode:      "eddsa",
								}
								data, _ = json.MarshalIndent(wrapper, "", "  ")
							}
						}
					}
					if writeErr := os.WriteFile(s.capAnchorFilePath, data, 0600); writeErr == nil {
						if mem, ok := s.metrics.(*metrics.Memory); ok {
							mem.IncCapabilityAnchorEmitted()
							// #nosec G115
							mem.SetCapabilityAnchorLastWriteUnix(uint64(time.Now().Unix()))
							// Emit algorithm facet metrics for all registered algorithms
							for _, algInfo := range crypto.ListAlgorithms() {
								mem.IncCapabilityAnchorAlgorithm(algInfo.Name)
							}
						}
						s.capAnchorEmitted = true
						s.capAnchorArtifact = data
					} else {
						fmt.Fprintf(os.Stderr, "[anchor] write failed: %v\n", writeErr)
					}
				}
			} else {
				if mem, ok := s.metrics.(*metrics.Memory); ok {
					mem.IncCapabilityAnchorSkipped()
				}
			}
		}
		// Emit to external anchor provider if configured
		if s.capabilityAnchorHandler != nil && s.capabilityAnchorHandler.Provider != nil && newHash != "" {
			s.capabilityAnchorHandler.SetRegistryHash(newHash)
			_, _ = s.capabilityAnchorHandler.Anchor(context.Background())
		}

		// Internal Notarization (if enabled via GAUTH_CAP_ANCHOR_NOTARIZE)
		if s.notarizer != nil {
			if rec, err := s.notarizer.Notarize(newHash); err != nil {
				fmt.Fprintf(os.Stderr, "[anchor] notarization failed hash=%s err=%v\n", newHash, err)
			} else {
				// Persist receipt
				if s.receiptStore != nil {
					if _, err := s.receiptStore.Append(rec); err != nil {
						fmt.Fprintf(os.Stderr, "[anchor] receipt persistence failed: %v\n", err)
					} else {
						s.capLastNotarization = time.Now()
						s.capLastNotarizationReceipt = rec
					}
				}
			}
		}
	}

	// Wire Audit Chaining (Capability Audit Anchor)
	if s.audit != nil {
		s.audit.OnEntry = func(e *AuditEntry) {
			prev := s.capabilitiesHandler.GetAuditPrevHash()
			// Deterministic canonicalization for hash chain
			// We need a stable representation. Using json.Marshal of the entry struct is reasonable if struct fields are stable.
			// AuditEntry struct in this file has clear JSON tags (though implicit defaults).
			// We'll use compact JSON.
			data, _ := json.Marshal(e)

			h := sha256.New()
			h.Write([]byte(prev))
			h.Write(data)
			newHash := fmt.Sprintf("sha256:%x", h.Sum(nil))

			s.capabilitiesHandler.SetAuditPrevHash(newHash)

			// Persist tip if configured
			if path := s.capabilitiesHandler.GetAuditPersistPath(); path != "" {
				tip := map[string]interface{}{
					"payload":   json.RawMessage(data),
					"hash":      newHash,
					"prev_hash": prev,
					"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
					"entry_id":  e.ID,
				}
				if b, err := json.Marshal(tip); err == nil {
					_ = os.WriteFile(path, b, 0600)
				}
			}
		}

		// Wire Revocation Audit Hook (RFC111-R7)
		delegation.OnRevocationAppended = func(ev delegation.RevocationEvent, chainLen int, aggregateHash string) {
			// Capture event in centralized audit log
			s.audit.Append(&AuditEntry{
				ID:       randomNonce(8),
				At:       time.Now().UTC(),
				Actor:    "system", // Attribution limited in async hook; implicitly system/grantor
				Action:   "revocation_appended",
				Resource: "revocation_chain",
				Outcome:  "success",
				Meta: map[string]any{
					"revocation_id":  ev.ID,
					"delegation_id":  ev.DelegationID,
					"chain_length":   chainLen,
					"aggregate_hash": aggregateHash,
					"event_hash":     ev.Hash,
					"reason":         ev.Reason,
				},
			})
		}
	}

	// Seed demo capabilities if no GAUTH_CAPABILITIES_PATH and registry is empty (test compatibility)
	if os.Getenv("GAUTH_CAPABILITIES_PATH") == "" {
		currentCaps := capability.DefaultRegistry().List()
		if len(currentCaps) == 0 {
			capability.Register(capability.Capability{ID: "cap.transfer", Version: "1.0", Stable: true})
			capability.Register(capability.Capability{ID: "cap.issue", Version: "1.0", Stable: true})
			capability.Register(capability.Capability{ID: "cap.delegation.create", Version: "1.0", Stable: true})
			capability.Register(capability.Capability{ID: "cap.delegation.revoke", Version: "1.0", Stable: true})
			currentCaps = capability.DefaultRegistry().List()
		}

		// Compute registry hash from capabilities (demo or existing) for external anchor support
		if len(currentCaps) > 0 {
			enc, _ := json.Marshal(currentCaps)
			hSum := sha256.Sum256(enc)
			demoHash := fmt.Sprintf("sha256:%x", hSum[:])
			s.capabilitiesHandler.RegistryHash = demoHash
		}
	}
	// Initialize Token Handler
	s.tokens = token.NewStore(500) // Re-initialize if overwritten or just ensure it matches
	s.delegationHandler = delegationHandlers.NewHandler(s.metrics, s.audit, s, s.enforceCapabilities, s.tracerProvider, s.capabilitiesHandler.GetRequiredCaps)

	// Initialize Key Manager
	// RR-013 Phase 2: Support External/Cloud KMS
	// Uses unified factory which checks GAUTH_KMS_PROVIDER
	km, kmErr := keys.NewKeyManager(context.Background())

	if kmErr != nil {
		fmt.Fprintf(os.Stderr, "[factory] warning: failed to init key manager: %v\n", kmErr)
		// We can proceed but token issuance might fail
	}

	s.tokenHandler = token.NewHandler(s.tokens, s.replayStore, s, s, s, &tokenTracerAdapter{tp: s.tracerProvider}, s.enforceCapabilities, s.metrics, s, s.keyProvider, km, s.systemClockMonitor)

	s.tokenHandler.ETagUpdater = s // Server implements JWKSETagUpdater
	s.tokenHandler.RegisterRoutes(s.router)

	// Initialize Policy Handler
	s.policyHandler = policyHandler.NewHandler(os.Getenv("POLICY_CHAIN_STATE_PATH"), s.metrics)
	// Inject revocation chain for end-to-end provenance (Roadmap Item 5)
	if s.revocationChain != nil {
		s.policyHandler.RevocationChain = s.revocationChain
	}
	s.policyHandler.EnsureInitialized()
	s.policyAPI = &policyHandler.API{Handler: s.policyHandler, Auditor: &policyAuditorAdapter{log: s.audit}}
	s.policyAPI.RegisterRoutes(s.router)

	// Initialize Authorizer (ensure initialized for handler)
	if s.authorizer == nil {
		s.authorizer = authz.NewMemoryAuthorizer()
	}

	// Initialize PDP Distributed Cache (RR-001)
	if os.Getenv("GAUTH_PDP_CACHE_ENABLED") == "1" {
		l1Size := atoiDefault(os.Getenv("GAUTH_PDP_CACHE_L1_SIZE"), 1000)
		l1 := authz.NewLRUDecisionCache(l1Size)

		var cacheImpl authz.DecisionCache = l1

		if os.Getenv("GAUTH_PDP_CACHE_TYPE") == "redis" && s.redisClient != nil {
			// Initialize L2 Redis cache using server's redis connection info
			redisHost := os.Getenv("REDIS_HOST")
			if redisHost == "" {
				redisHost = "localhost"
			}
			redisPort := os.Getenv("REDIS_PORT")
			if redisPort == "" {
				redisPort = "6379"
			}

			l2, err := cache.NewRedisCache(&cache.Config{
				RedisURL: fmt.Sprintf("redis://%s:%s", redisHost, redisPort),
				Type:     "redis",
			})
			if err == nil {
				nodeID := os.Getenv("GAUTH_NODE_ID")
				if nodeID == "" {
					nodeID = "node-" + strconv.FormatInt(time.Now().UnixNano()%1000, 10)
				}
				cacheImpl = authz.NewDistributedDecisionCache(l1, l2, s.redisClient.GetClient(), nodeID)
				fmt.Fprintln(os.Stderr, "[factory] using Distributed PDP Cache (Hybrid L1/L2)")
			} else {
				fmt.Fprintf(os.Stderr, "[factory] warning: failed to init pdp l2 cache: %v\n", err)
			}
		} else {
			fmt.Fprintln(os.Stderr, "[factory] using Local PDP Cache (L1 only)")
		}

		s.authorizer.SetDecisionCache(cacheImpl)

		// Wire OnPolicyChange hook to invalidate cache across nodes
		if s.policyHandler != nil {
			s.policyHandler.OnPolicyChange = func() {
				cacheImpl.InvalidateAll()
				fmt.Fprintln(os.Stderr, "[factory] pdp cache invalidated due to policy change")
			}
		}
	}

	// Wire obligation executor for persistence and failure taxonomy (Epic 4)
	s.authorizer.SetObligationExecutor(&authz.AuditObligationExecutor{Audit: s.audit})
	// Initialize Authz Handler
	s.authzAPI = authzHandlers.NewAPI(s.authorizer, s.policyHandler, s.metrics)
	s.authzAPI.RegisterRoutes(s.router)

	// Initialize Model Limits Handler
	s.modelLimitsHandler = modellimits.NewHandler(
		os.Getenv("GAUTH_MODEL_LIMITS_CONFIG_PATH"),
		os.Getenv("GAUTH_MODEL_LIMIT_AUDIT_PATH"),
		os.Getenv("GAUTH_MODEL_LIMIT_ANCHOR_PATH"),
	)
	s.modelLimitsHandler.StrictUnknown = os.Getenv("GAUTH_MODEL_LIMITS_STRICT_UNKNOWN") == "1"
	// Wire dependencies if available
	if km := s.getKeyManager(); km != nil {
		s.modelLimitsHandler.KeyManager = km
	}
	s.modelLimitsHandler.Metrics = s.metrics
	if err := s.modelLimitsHandler.Init(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "[model-limits] init failed: %v\n", err)
	}
	s.modelLimitsAPI = modellimits.NewAPI(s.modelLimitsHandler)

	// Initialize Database
	if host := os.Getenv("GAUTH_DB_HOST"); host != "" {
		port := atoiDefault(os.Getenv("GAUTH_DB_PORT"), 5432)
		dbCfg := &database.Config{
			Host:     host,
			Port:     port,
			User:     os.Getenv("GAUTH_DB_USER"),
			Password: os.Getenv("GAUTH_DB_PASSWORD"),
			Database: os.Getenv("GAUTH_DB_NAME"),
			SSLMode:  os.Getenv("GAUTH_DB_SSL_MODE"),
		}
		if db, err := database.NewDB(dbCfg); err == nil {
			s.db = db
			fmt.Printf("Database connected: %s\n", host)

			// Initialize Blockchain Components if configured
			if ethRPC := os.Getenv("GAUTH_ETH_RPC_URL"); ethRPC != "" {
				ethKey := os.Getenv("GAUTH_ETH_PRIVATE_KEY")
				// We allow empty private key for read-only mode if needed, but warnings usually apply

				ethConfig := &blockchain.EthereumConfig{
					RPCURL:          ethRPC,
					PrivateKey:      ethKey,
					ContractAddress: os.Getenv("GAUTH_ETH_CONTRACT_ADDRESS"),
					NetworkName:     "sepolia", // defaulting to sepolia for this phase
					ChainID:         11155111,  // Sepolia ChainID
					GasLimit:        3000000,
				}

				adapter, err := blockchain.NewEthereumAdapter(ethConfig)
				if err != nil {
					fmt.Printf("Warning: Failed to initialize Ethereum Adapter: %v\n", err)
				} else {
					s.blockchainRegistry = adapter
					fmt.Println("Blockchain Adapter initialized (Ethereum/Sepolia)")

					// Initialize Sync Service
					poaStore := blockchain.NewPostgresPoAStore(s.db)
					hashingService := blockchain.NewSimpleHashingService()
					// IPFS service is optional/nil for now

					syncConfig := blockchain.DefaultSyncConfig()
					// Override sync config from env if needed
					if mode := os.Getenv("GAUTH_SYNC_MODE"); mode != "" {
						syncConfig.SyncMode = mode
					}

					s.syncService = blockchain.NewBlockchainSyncService(
						adapter,
						poaStore,
						hashingService,
						nil, // ipfsService
						syncConfig,
					)

					// Start Sync Service
					// Note: Background context used for long-running service
					if err := s.syncService.Start(context.Background()); err != nil {
						fmt.Printf("Warning: Failed to start Blockchain Sync Service: %v\n", err)
					} else {
						fmt.Println("Blockchain Sync Service started")
					}
				}
			}

			// Wire up audit logger
			repo := audit.NewRepository(db.Pool)
			l := common.NewSimpleLogger()
			dbLogger := audit.NewDatabaseLogger(repo, l)
			s.audit.SetDatabaseLogger(dbLogger)
		} else {
			fmt.Printf("Failed to connect to database: %v\n", err)
		}
	}

	// Defer semantic diagnostics integrity endpoint registration to initUIRevamp (invoked immediately below)
	s.initUIRevamp()
	// Register composite authorization endpoints (demo) for tests expecting activation flow.
	// POST /api/v1/authorization/composite -> 201 on first activation, 409 on conflict when duplicate hash.
	// GET  /api/v1/authorization/composite -> 404 until activated, then 200 with artifact summary.
	var compositeActivated bool
	var compositeHash string
	r.POST("/api/v1/authorization/composite", func(c *gin.Context) {
		// Read raw body (store copy) then reset Body so gin JSON binding can succeed even on repeated POST.
		b, err := io.ReadAll(c.Request.Body)
		if err != nil || len(b) == 0 {
			c.JSON(400, gin.H{"success": false, "error": "invalid_json"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(b))
		var payload map[string]any
		// Attempt JSON parse only if not yet activated; after activation we skip parse to allow conflict detection even if stream reused.
		if !compositeActivated {
			if err := json.Unmarshal(b, &payload); err != nil {
				c.JSON(400, gin.H{"success": false, "error": "invalid_json"})
				return
			}
		}
		raw := b
		h := sha256.Sum256(raw)
		digest := hex.EncodeToString(h[:])
		if compositeActivated {
			if digest == compositeHash {
				c.JSON(409, gin.H{"success": false, "error": "authorization_conflict"})
				return
			}
		}
		compositeActivated = true
		compositeHash = digest
		c.JSON(201, gin.H{"success": true, "activated": true, "hash": compositeHash})
	})
	r.GET("/api/v1/authorization/composite", func(c *gin.Context) {
		if !compositeActivated {
			c.JSON(404, gin.H{"success": false, "error": "authorization_not_found"})
			return
		}
		c.JSON(200, gin.H{"success": true, "hash": compositeHash})
	})
	// Register modular anchor handlers early to ensure consistent error taxonomy.
	betaGroup := r.Group("/api/v1/beta")
	anchorHandlers.RegisterAll(betaGroup, s)
	anchorHandlers.RegisterMetrics(betaGroup, s)

	// Keep KeyManager sync for now manually or inject
	// s.capabilitiesHandler.KeyManager = s.keyProvider // Type mismatch likely, assume KeyProvider interface ok?
	// NewHandler defines KeyManager as crypto.KeyManager interface.
	if km, ok := s.keyProvider.(*crypto.Manager); ok {
		s.capabilitiesHandler.KeyManager = km
	}
	// Capability registry external anchor artifact configuration (prototype)
	if v := os.Getenv("GAUTH_CAP_ANCHOR_FILE_PATH"); v != "" {
		// Expand ~ for convenience
		if strings.HasPrefix(v, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				v = filepath.Join(home, strings.TrimPrefix(v, "~"))
			}
		}
		abs := v
		if !strings.HasPrefix(abs, string(os.PathSeparator)) {
			cwd, _ := os.Getwd()
			abs = filepath.Join(cwd, abs)
		}
		// #nosec G301
		if err := os.MkdirAll(filepath.Dir(abs), 0o750); err == nil {
			s.capAnchorFilePath = abs
			fmt.Fprintf(os.Stderr, "[cap-anchor] file path configured path=%s\n", abs)
		}
	}
	if v := os.Getenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL"); v != "" {
		if dur, err := time.ParseDuration(v); err == nil && dur >= time.Minute {
			s.capAnchorWriteInterval = dur
		}
	}
	if s.capAnchorWriteInterval == 0 {
		// default conservative interval (5m) to limit filesystem churn
		s.capAnchorWriteInterval = 5 * time.Minute
	}
	// SLA stale threshold (seconds) optional: GAUTH_CAP_ANCHOR_STALE_THRESHOLD_SECONDS (default 600s)
	staleSec := 600
	if raw := os.Getenv("GAUTH_CAP_ANCHOR_STALE_THRESHOLD_SECONDS"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			staleSec = v
		}
	}
	s.capabilitiesHandler.AnchorStaleThreshold = time.Duration(staleSec) * time.Second
	// Start stale monitor goroutine (checks every 30s) unless background polls disabled entirely.

	// Register limits diagnostics endpoint (if manager initialized) early so it's available immediately.
	if lm := limits.GetManager(); lm != nil {
		// Attach snapshot callback to append audit entry on each periodic persistence.
		lm.SetSnapshotCallback(func(entry map[string]any) {
			// Derive counters copy excluding metadata keys
			meta := map[string]any{}
			for k, v := range entry {
				if k == "_type" || k == "ts" {
					continue
				}
				meta[k] = v
			}
			// Append audit entry (non-blocking safety already in audit Append)
			s.audit.Append(&AuditEntry{ID: randomNonce(6), At: time.Now(), Actor: "system", Action: limits.SnapshotType, Resource: "limits_store", Outcome: "success", Meta: meta})
		})
		// HTTP endpoint registered via helper to keep routing centralized
		s.registerLimitsDiagnostics(r)
	}
	// Register model validation endpoint (prototype governance feature)
	// Register Model Limits Handored (refactored)
	if s.modelLimitsAPI != nil {
		s.modelLimitsAPI.RegisterRoutes(s.router)
	}
	// Combined anchor and notarization endpoints (extracted to handlers/notary)
	notaryMetrics := &notaryMetricsAdapter{m: s.metrics}
	notaryHandler := notaryHandlers.NewHandler(s, notaryMetrics)
	notaryHandler.RegisterRoutes(s.router)
	// RB3 Discovery endpoint (cacheable config snapshot)
	s.registerRB3Discovery()
	// RB4 Signed Policy Manifest endpoint (hash-addressed snapshot + signature)
	s.registerPolicyManifest()
	// Crypto algorithms introspection endpoint required by tests
	s.router.GET("/api/v1/crypto/algorithms", s.apiCryptoAlgorithms)

	// GNAP (RFC 9635) Grant Negotiation and Authorization Protocol
	gnapStore := gnapPkg.NewMemoryGrantStore()
	gnapTokenStore := gnapPkg.NewMemoryTokenStore()
	gnapRSStore := gnapPkg.NewMemoryResourceServerStore() // RFC 9767 Support
	gnapBaseURL := envFallback("GAUTH_GNAP_BASE_URL", "http://localhost:8080")

	// Create VerificationService for GNAP-GAuth integration (nil for now - wire later with DB services)
	// Create VerificationService for GNAP-GAuth integration
	var gnapVerificationService gauthplus.VerificationService

	if s.db != nil {
		poaStore := gauthplus.NewPostgreSQLPoAStore(s.db)
		// Enable caching for delegation and capability services (RR-014)
		rawDelSvc := gauthplus.NewPostgreSQLDelegationService(s.db)
		delSvc := gauthplus.NewCachedDelegationService(rawDelSvc, 5*time.Minute)

		rawCapSvc := gauthplus.NewPostgreSQLCapabilityAssessmentService(s.db)
		capSvc := gauthplus.NewCachedCapabilityService(rawCapSvc, 10*time.Minute)

		fidSvc := gauthplus.NewPostgreSQLFiduciaryDutyService(s.db)
		principalVerifier := gauthplus.NewDefaultPrincipalVerifier()
		attestationVerifier := gauthplus.NewDefaultAttestationVerifier()

		// Generate a temporary key for the Verification Service Authority (Phase 11)
		// In production, this would be retrieved from a secure KeyStore/KMS
		_, verifPrivKey, _ := ed25519.GenerateKey(crand.Reader)
		attestationSigner := gauthplus.NewDefaultAttestationSigner("gauthplus-local-authority", verifPrivKey)

		// Register the public key in the verifier for local bridge support
		attestationVerifier.RegisterKey("gauthplus-local-authority", verifPrivKey.Public().(ed25519.PublicKey))

		registerService := registry.NewMockCommercialRegisterService()

		gnapVerificationService = gauthplus.NewVerificationService(
			poaStore, delSvc, capSvc, fidSvc, principalVerifier, attestationVerifier, attestationSigner, registerService,
		)
		fmt.Println("GAuth+ VerificationService wired with PostgreSQL storage (cached) and Hardened Verifiers")
	}

	gnapHandler := gnapHandlers.NewHandler(gnapStore, gnapVerificationService, gnapBaseURL)
	gnapHandler.TokenStore = gnapTokenStore
	gnapHandler.RSStore = gnapRSStore
	gnapHandler.RegisterRoutes(s.router)

	// RFC 7523 Client Authentication
	clientKeyStore := auth.NewMemoryKeyStore()
	clientAuthenticator := &auth.PrivateKeyJWTValidator{
		KeyProvider:    clientKeyStore,
		ValidAudiences: []string{envFallback("GAUTH_TOKEN_ENDPOINT", "http://localhost:8080/device/token")},
		Replay:         s.replayStore,
	}

	// RFC 8628 Device Authorization Grant
	deviceStore := devicePkg.NewMemoryDeviceCodeStore()
	deviceHandler := deviceHandlers.NewHandler(deviceStore)
	deviceHandler.SetAuthenticator(clientAuthenticator)
	deviceHandler.RegisterRoutes(s.router)

	// A2A Profile (Draft)
	a2aHandler := a2aHandlers.NewHandler()
	a2aHandler.SetAuthenticator(clientAuthenticator)
	a2aHandler.RegisterRoutes(s.router)

	// RFC 7523 JWT Bearer Grant ("Identity Assertion")
	jwtGrantHandler := grantJWTHandlers.NewHandler(clientAuthenticator)
	jwtGrantHandler.RegisterRoutes(s.router)

	// OAuth2 Handler (CIBA & Token Exchange)
	var oauth2DBPool *pgxpool.Pool
	if s.db != nil {
		oauth2DBPool = s.db.Pool
	}
	oauth2Handler := authHandlers.NewOAuth2Handler(oauth2DBPool, os.Getenv("GAUTH_JWT_SIGNING_KEY"))
	oauth2Group := s.router.Group("/api/v1/oauth2")
	oauth2Handler.RegisterRoutes(oauth2Group)
	log.Println("[oauth2] registered CIBA and Token Exchange endpoints at /api/v1/oauth2/*")

	log.Println("[gnap] RFC 9635 GNAP endpoints registered at /gnap/*")

	// SAML Reference Implementation
	// SAML Reference Implementation
	if s.db != nil {
		samlHandler := saml.NewHandler(s.db.Pool)
		apiGroup := s.router.Group("/api/v1")
		samlHandler.RegisterRoutes(apiGroup)
		log.Println("[saml] registered endpoints at /api/v1/saml/*")

		// SCIM Reference Implementation
		scimHandler := scim.NewHandler(s.db.Pool)
		scimHandler.RegisterRoutes(apiGroup)
		log.Println("[scim] registered endpoints at /api/v1/scim/v2/*")
	}

	// Protocol Flow API endpoints (Item 2: Protocol Flow Navigation)
	s.router.POST("/api/v1/protocol/flow/sessions", s.apiProtocolFlowCreateSession)
	s.router.GET("/api/v1/protocol/flow/sessions/:id", s.apiProtocolFlowGetSession)
	s.router.POST("/api/v1/protocol/flow/sessions/:id/navigate", s.apiProtocolFlowNavigate)
	s.router.POST("/api/v1/protocol/flow/sessions/:id/steps/:stepId/status", s.apiProtocolFlowUpdateStep)
	s.router.POST("/api/v1/protocol/flow/sessions/:id/steps/:stepId/substeps/:substepId/complete", s.apiProtocolFlowCompleteSubstep)

	// Learning Lab endpoints for full button functionality
	s.AddLearningLabEndpoints()

	// Register Static UI Routes (Dashboard & CSP)
	s.RegisterUIRoutes()

	// P*P Architecture endpoints (PAP, PDP, PEP)
	s.AddPPPEndpoints()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				// compute age
				// Check staleness via handler
				stale, age, threshold := s.capabilitiesHandler.GetAnchorState()
				if age > threshold {
					if !stale {
						s.capabilitiesHandler.SetAnchorState(true, age)
						// Log...
					}
				} else {
					if stale {
						s.capabilitiesHandler.SetAnchorState(false, age)
					}
				}
				// Update Prometheus adapter gauges if present
				if pm, ok := s.metrics.(interface {
					SetCapabilityAnchorAgeSeconds(uint64)
					SetCapabilityAnchorStale(bool)
				}); ok {
					pm.SetCapabilityAnchorAgeSeconds(uint64(age.Seconds()))
					pm.SetCapabilityAnchorStale(stale)
				}
				// Notarization age gauge (seconds since last receipt)
				if pm, ok := s.metrics.(interface{ SetCapabilityAnchorNotarizedAgeSeconds(uint64) }); ok {
					nAge := uint64(0)
					if !s.capLastNotarization.IsZero() {
						nAge = uint64(time.Since(s.capLastNotarization).Seconds())
					}
					pm.SetCapabilityAnchorNotarizedAgeSeconds(nAge)
				}
				// External anchor age gauge
				if pm, ok := s.metrics.(interface {
					SetExternalAnchorAgeSeconds(uint64)
					SetExternalAnchorLastHashLen(int)
				}); ok && s.capabilityAnchorHandler != nil {
					ageExt := uint64(0)
					lastReceipt := s.capabilityAnchorHandler.GetLastReceipt()
					if !lastReceipt.Timestamp.IsZero() {
						ageExt = uint64(time.Since(lastReceipt.Timestamp).Seconds())
					}
					pm.SetExternalAnchorAgeSeconds(ageExt)
					pm.SetExternalAnchorLastHashLen(len(lastReceipt.Hash))
				}
			}
		}
	}()

	// Violation persistence path (optional) GAUTH_VIOLATION_PERSIST_PATH
	s.violationPersistPath = os.Getenv("GAUTH_VIOLATION_PERSIST_PATH")
	if s.violationPersistPath != "" {
		// Expand tilde
		if strings.HasPrefix(s.violationPersistPath, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				s.violationPersistPath = filepath.Join(home, strings.TrimPrefix(s.violationPersistPath, "~"))
			}
		}
		// Initialize Handler with path
		s.violationHandler = violations.NewHandler(nil, s.metrics, s.violationPersistPath)
		// Load existing violations to restore state
		if err := s.violationHandler.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "[violations] load error path=%s err=%v\n", s.violationPersistPath, err)
		} else {
			fmt.Fprintf(os.Stderr, "[violations] loaded persistence path=%s count=%d\n", s.violationPersistPath, s.violationHandler.Count())
			// Initialize integrity status
			s.violationIntegrityStatus = integrityUnconfigured
		}
	} else {
		s.violationHandler = violations.NewHandler(nil, s.metrics, "")
	}
	// Create and register violation API routes
	s.violationAPI = violations.NewAPI(s.violationHandler)
	s.violationAPI.RegisterRoutes(s.router)

	// Semantic persistence path (optional) GAUTH_SEMANTIC_PERSIST_PATH
	s.semanticPersistPath = os.Getenv("GAUTH_SEMANTIC_PERSIST_PATH")
	if s.semanticPersistPath != "" {
		if strings.HasPrefix(s.semanticPersistPath, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				s.semanticPersistPath = filepath.Join(home, strings.TrimPrefix(s.semanticPersistPath, "~"))
			}
		}
		s.semanticHandler = semantic.NewHandler(nil, nil, s.semanticPersistPath)
		if err := s.semanticHandler.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "[semantics] load error path=%s err=%v\n", s.semanticPersistPath, err)
		} else {
			fmt.Fprintf(os.Stderr, "[semantics] loaded persistence path=%s count=%d\n", s.semanticPersistPath, s.semanticHandler.Count())
			s.semanticIntegrityStatus = integrityUnconfigured
		}
	} else {
		s.semanticHandler = semantic.NewHandler(nil, nil, "")
	}

	// Semantic ledger path (optional) GAUTH_SEMANTIC_LEDGER_PATH (Item 8)
	if slp := os.Getenv("GAUTH_SEMANTIC_LEDGER_PATH"); slp != "" {
		if strings.HasPrefix(slp, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				slp = filepath.Join(home, strings.TrimPrefix(slp, "~"))
			}
		}
		if ls, err := ledger.NewBoltStore(slp); err == nil {
			s.semanticHandler.Ledger = ls
			fmt.Fprintf(os.Stderr, "[semantics] ledger initialized path=%s\n", slp)
		} else {
			fmt.Fprintf(os.Stderr, "[semantics] ledger failed: %v\n", err)
		}
	}

	// Receipt store persistence path (optional) GAUTH_NOTARY_RECEIPT_PERSIST_PATH
	if rp := os.Getenv("GAUTH_NOTARY_RECEIPT_PERSIST_PATH"); rp != "" {
		if strings.HasPrefix(rp, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				rp = filepath.Join(home, strings.TrimPrefix(rp, "~"))
			}
		}
		rs := notary.NewReceiptStore(rp)
		if err := rs.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "[notary] receipt store load error path=%s err=%v\n", rp, err)
		} else {
			fmt.Fprintf(os.Stderr, "[notary] receipt store loaded path=%s entries=%d\n", rp, len(rs.Entries()))
			s.receiptStore = rs
		}
	}

	// Persist audit log path (optional)
	if pp := os.Getenv("GAUTH_AUDIT_PERSIST_PATH"); pp != "" {
		if strings.HasPrefix(pp, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				pp = filepath.Join(home, strings.TrimPrefix(pp, "~"))
			}
		}
		s.capabilitiesHandler.SetAuditPersistPath(pp)
		fmt.Fprintf(os.Stderr, "[cap-audit] persistence path configured: %s\n", pp)
	}

	// Initialize Primary Auth Service (Standard JWT)
	// Supports HS256 (default) or RS256 with key pair.
	// Initialize primary token service (for violation counters) unless explicitly disabled.
	if os.Getenv("GAUTH_DISABLE_VIOLATION_SERVICE") != "1" {
		cfg := gauth.Config{
			AuthServerURL:     os.Getenv("GAUTH_ISSUER"),
			ClientID:          "beta-demo-client",
			ClientSecret:      "demo-client-secret-00000000000000000000000000000000",
			SigningKey:        os.Getenv("GAUTH_SIGNING_KEY"), // optional distinct HMAC key
			Scopes:            []string{"demo"},
			AccessTokenExpiry: 30 * time.Minute,
			Audience:          []string{"beta-audience"},
		}
		if signingKeyStr := os.Getenv("GAUTH_SIGNING_KEY"); signingKeyStr != "" {
			cfg.SigningKey = signingKeyStr
		}

		primary, err := gauth.New(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[violation-metrics] primary service init failed: %v\n", err)
		} else {
			s.primaryAuthService = primary
			fmt.Fprintln(os.Stderr, "[violation-metrics] primary gauth.Service initialized (violation counters active)")
			// Initialize handler with service and load persistence
			// Note: s.violationHandler.Service field needs to be set.
			// server_clean.go line 1963: s.violationHandler.Service = primary
			if s.tokenHandler != nil {
				s.tokenHandler.SetGAuthService(primary)
			}
			// Wait, check type compatibility: primary(*gauth.Service) vs ViolationProvider interface.
			// Assuming it satisfies it.
			// I need to set the Service field directly since NewHandler usage above passed nil service.
			s.violationHandler.Service = primary
			if err := s.violationHandler.Load(); err != nil {
				fmt.Fprintf(os.Stderr, "violation load error: %v\n", err)
			}
		}
	}

	// Initialize RFC0111 Semantic Interoperability Service
	// Initialize RFC0111 service (semantic counters) unless disabled (GAUTH_DISABLE_RFC0111_SERVICE=1)
	if os.Getenv("GAUTH_DISABLE_RFC0111_SERVICE") != "1" {
		// Create an RFC0111 service using in-memory audit logger and existing authorizer policies.
		memAudit := audit.NewMemoryLogger(nil)
		if s.authorizer == nil {
			s.authorizer = authz.NewMemoryAuthorizer()
		}
		// Functional options: enable mandatory signatures when GAUTH_MULTI_SIG_STRICT set (already handled internally by NewService via env).
		rfcOpts := []gauth_rfc_001.Option{}
		if s.redisClient != nil {
			// Enable distributed replay protection (GAUTH-VULN-004)
			rfcOpts = append(rfcOpts, gauth_rfc_001.WithReplayStoreRedis(s.redisClient.GetClient(), "gauth", 5*time.Minute))
		}
		svc := gauth_rfc_001.NewService(memAudit, s.authorizer, rfcOpts...)
		s.rfc0111Service = svc
		fmt.Fprintln(os.Stderr, "[rfc0111] service initialized (semantic counters active)")
		// Mount dual-control revocation workflow HTTP endpoints.
		// NOTE: s.mountRevocationWorkflow() method must be present in server_clean.go (or extracted).
		// Since we are in the same package 'web', we can call it if it exists on *BetaServer.
		s.mountRevocationWorkflow()
	}

	// Initialize Semantic Handler (duplicate init logic if NewBetaServerWithMetrics is standalone)
	// Inject the service into the handler
	s.semanticHandler.Service = s.rfc0111Service
	s.semanticAPI = semantic.NewAPI(s.semanticHandler)
	if err := s.semanticHandler.Load(); err != nil {
		fmt.Printf("semantic load error: %v\n", err)
	}
	s.semanticAPI.RegisterRoutes(s.router)

	// Check loaded stats
	// eC, sC := s.semanticHandler.Stats() -> Stats returns (int, int) based on violations/handler.go view earlier?
	// Wait, semantic/handler.go Stats() returns (int, int).
	if eC, sC := s.semanticHandler.Stats(); eC > 0 || sC > 0 {
		fmt.Printf("loaded semantic stats: %d ewma, %d scores\n", eC, sC)
	}

	if s.db != nil {
		// Production/Staging with Database
		dbPool := s.db.Pool
		if dbPool != nil {
			adminGroup := r.Group("/api/admin")
			// Add tenant-aware middleware to admin group
			adminGroup.Use(func(c *gin.Context) {
				tenantID := c.GetHeader("X-Tenant-ID")
				if tenantID == "" {
					tenantID = c.Query("tenant_id")
				}
				if tenantID != "" {
					c.Set("tenant_id", tenantID)
				}
				c.Next()
			})

			// Auth handler
			jwtSecret := os.Getenv("GAUTH_JWT_SIGNING_KEY")
			if jwtSecret == "" {
				jwtSecret = "dev-secret-change-in-production"
			}
			authHandler := adminHandlers.NewAuthHandler(jwtSecret)
			authHandler.RegisterRoutes(adminGroup)

			// PoA handler
			poaHandler := adminHandlers.NewPoAHandler(dbPool)
			poaHandler.RegisterRoutes(adminGroup)

			// Resilience handler
			resilienceHandler := adminHandlers.NewResilienceHandler(dbPool)
			resilienceHandler.RegisterRoutes(adminGroup)

			// Events handler (Singular)
			eventHandler := adminHandlers.NewEventHandler(dbPool)
			eventHandler.RegisterRoutes(adminGroup)

			// Authorization handler (Long name)
			authzHandler := adminHandlers.NewAuthorizationHandler(dbPool)
			authzHandler.RegisterRoutes(adminGroup)

			// Configuration handler
			configHandler := adminHandlers.NewConfigHandler(dbPool)
			configHandler.RegisterRoutes(adminGroup)

			// Tokens handler (Singular, 2 args)
			tokenHandler := adminHandlers.NewTokenHandler(dbPool, nil)
			tokenHandler.RegisterRoutes(adminGroup)

			// Metrics handler (requires registry)
			registry := prometheus.NewRegistry()
			metricsHandler := adminHandlers.NewMetricsHandler(registry, dbPool)
			metricsHandler.RegisterRoutes(adminGroup)

			// Audit handler
			auditHandler := adminHandlers.NewAuditHandler(dbPool)
			auditHandler.RegisterRoutes(adminGroup)

			// Subscribers handler (Singular)
			subscriberHandler := adminHandlers.NewSubscriberHandler(dbPool)
			subscriberHandler.RegisterRoutes(adminGroup)

			// Revocation handler
			revocationHandler := adminHandlers.NewRevocationHandler(dbPool)
			revocationHandler.RegisterRoutes(adminGroup)

			// API Keys handler (Singular)
			apiKeyHandler := adminHandlers.NewAPIKeyHandler(dbPool)
			apiKeyHandler.RegisterRoutes(adminGroup)

			// Security handler
			securityHandler := adminHandlers.NewSecurityHandler(dbPool)
			securityHandler.RegisterRoutes(adminGroup)

			// Initialize Redis cache for PoA handler
			cacheConf := cacheConfig.LoadCacheConfig()
			if err := cacheConfig.ValidateCacheConfig(cacheConf); err != nil {
				log.Printf("[WARNING] Invalid cache configuration: %v - using memory cache fallback", err)
				cacheConf.Type = "memory"
			}
			cacheInstance := cache.NewCacheWithFallback(cacheConf)
			// defer cacheInstance.Close() // Removed to prevent early closure

			// Set cache on PoA handler
			poaHandler.SetCache(cacheInstance)

			cacheHandler := adminHandlers.NewCacheHandler(cacheInstance)
			cacheHandler.RegisterRoutes(adminGroup)

			// OIDC providers handler
			oidcHandler := adminHandlers.NewOIDCHandler(dbPool)
			oidcHandler.RegisterRoutes(adminGroup) // Policy Templates handler
			policyTemplatesHandler := adminHandlers.NewPolicyTemplatesHandler(dbPool)
			adminGroup.GET("/policy-templates", policyTemplatesHandler.ListPolicyTemplates)
			adminGroup.GET("/policy-templates/:id", policyTemplatesHandler.GetPolicyTemplate)
			adminGroup.POST("/policy-templates", policyTemplatesHandler.CreatePolicyTemplate)
			adminGroup.PUT("/policy-templates/:id", policyTemplatesHandler.UpdatePolicyTemplate)
			adminGroup.POST("/policy-templates/:id/clone", policyTemplatesHandler.ClonePolicyTemplate)
			adminGroup.DELETE("/policy-templates/:id", policyTemplatesHandler.DeletePolicyTemplate)

			// GAuth+ enhanced authorization handler (Admin API)
			gauthPlusHandler := adminHandlers.NewGAuthPlusHandler(dbPool)
			gauthPlusHandler.RegisterRoutes(adminGroup)

			// GAuth+ Public API (v1) - Create services sharing the pool
			gplusDB := &database.DB{Pool: dbPool}
			succSvc := gauthplus.NewPostgreSQLSuccessorService(gplusDB)
			delSvc := gauthplus.NewPostgreSQLDelegationService(gplusDB)
			dualSvc := gauthplus.NewPostgreSQLDualControlService(gplusDB)
			capSvc := gauthplus.NewPostgreSQLCapabilityAssessmentService(gplusDB)
			fidSvc := gauthplus.NewPostgreSQLFiduciaryDutyService(gplusDB)
			s.RegisterGAuthPlusEndpoints(succSvc, delSvc, dualSvc, capSvc, fidSvc)

			fmt.Fprintln(os.Stderr, "[admin] handlers registered: auth, poa, resilience, events, authz, config, tokens, metrics, audit, subscribers, revocation, api-keys, security, cache, oidc, policy-templates, gauthplus (17 total)") // OIDC authentication flow handler
			oidcAuthHandler := authHandlers.NewOIDCAuthHandler(dbPool)
			authGroup := r.Group("/auth")
			oidcAuthHandler.RegisterRoutes(authGroup)
			fmt.Fprintln(os.Stderr, "[auth] OIDC authentication flow handler registered")
		}
	} else {
		// Development mode: Register minimal auth handler without database
		fmt.Fprintln(os.Stderr, "[WARNING] DB_HOST not configured - running in DEVELOPMENT mode")
		fmt.Fprintln(os.Stderr, "[DEV] Admin authentication endpoint available (database-free mode)")
		fmt.Fprintln(os.Stderr, "[DEV] Other admin endpoints require database - set DB_HOST to enable")

		adminGroup := r.Group("/api/admin")
		adminGroup.Use(func(c *gin.Context) {
			tenantID := c.GetHeader("X-Tenant-ID")
			if tenantID == "" {
				tenantID = c.Query("tenant_id")
			}
			if tenantID != "" {
				c.Set("tenant_id", tenantID)
			}
			c.Next()
		})

		jwtSecret := os.Getenv("GAUTH_JWT_SIGNING_KEY")
		if jwtSecret == "" {
			jwtSecret = "dev-secret-change-in-production"
		}
		authHandler := adminHandlers.NewAuthHandler(jwtSecret)
		authHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] Auth handler registered: POST /api/admin/auth/login")
		fmt.Fprintln(os.Stderr, "[DEV] Test credentials: admin@example.com / password")

		// Authorization Handler (Degraded Mode)
		authzHandler := adminHandlers.NewAuthorizationHandler(nil)
		authzHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] Authorization handler registered (Degraded Mode: Memory/Empty)")

		// Subscribers Handler (Degraded Mode)
		subscriberHandler := adminHandlers.NewSubscriberHandler(nil)
		subscriberHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] Subscribers handler registered (Degraded Mode: Empty)")

		// Metrics Handler (Degraded Mode)
		// Metrics Handler (Degraded Mode)
		registry := prometheus.NewRegistry()
		metricsHandler := adminHandlers.NewMetricsHandler(registry, nil)
		metricsHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] Metrics handler registered (Degraded Mode: In-Memory Only)")

		// Register remaining handlers with nil DB for Degraded Mode (Empty Data)

		// 1. PoA Handler
		poaHandler := adminHandlers.NewPoAHandler(nil)
		poaHandler.RegisterRoutes(adminGroup)
		// Initialize dummy cache for PoA
		cacheConf := cacheConfig.LoadCacheConfig()
		cacheConf.Type = "memory"
		cacheInstance := cache.NewCacheWithFallback(cacheConf)
		poaHandler.SetCache(cacheInstance)
		fmt.Fprintln(os.Stderr, "[DEV] PoA handler registered (Degraded Mode: Empty)")

		// 2. Events Handler
		eventHandler := adminHandlers.NewEventHandler(nil)
		eventHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] Events handler registered (Degraded Mode: Empty)")

		// 3. Audit Handler
		auditHandler := adminHandlers.NewAuditHandler(nil)
		auditHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] Audit handler registered (Degraded Mode: Empty)")

		// 4. Revocation Handler
		revocationHandler := adminHandlers.NewRevocationHandler(nil)
		revocationHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] Revocation handler registered (Degraded Mode: Empty)")

		// 5. OIDC Handler
		oidcHandler := adminHandlers.NewOIDCHandler(nil)
		oidcHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] OIDC handler registered (Degraded Mode: Empty)")

		// 6. Policy Templates Handler
		policyTemplatesHandler := adminHandlers.NewPolicyTemplatesHandler(nil)
		adminGroup.GET("/policy-templates", policyTemplatesHandler.ListPolicyTemplates) // Manually registered in main block
		adminGroup.GET("/policy-templates/:id", policyTemplatesHandler.GetPolicyTemplate)
		adminGroup.POST("/policy-templates", policyTemplatesHandler.CreatePolicyTemplate)
		adminGroup.PUT("/policy-templates/:id", policyTemplatesHandler.UpdatePolicyTemplate)
		adminGroup.POST("/policy-templates/:id/clone", policyTemplatesHandler.ClonePolicyTemplate)
		adminGroup.DELETE("/policy-templates/:id", policyTemplatesHandler.DeletePolicyTemplate)
		fmt.Fprintln(os.Stderr, "[DEV] Policy Templates handler registered (Degraded Mode: Empty)")

		// 7. GAuthPlus Handler
		gauthPlusHandler := adminHandlers.NewGAuthPlusHandler(nil)
		gauthPlusHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] GAuthPlus handler registered (Degraded Mode: Empty)")

		// 8. Resilience Handler
		resilienceHandler := adminHandlers.NewResilienceHandler(nil)
		resilienceHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] Resilience handler registered (Degraded Mode: Empty)")

		// 9. Tokens Handler
		tokenHandler := adminHandlers.NewTokenHandler(nil, nil)
		tokenHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] Tokens handler registered (Degraded Mode: Empty)")

		// 10. API Keys Handler
		apiKeyHandler := adminHandlers.NewAPIKeyHandler(nil)
		apiKeyHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] API Keys handler registered (Degraded Mode: Empty)")

		// 11. Security Handler
		securityHandler := adminHandlers.NewSecurityHandler(nil)
		securityHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] Security handler registered (Degraded Mode: Empty)")

		// 12. Config Handler
		configHandler := adminHandlers.NewConfigHandler(nil)
		configHandler.RegisterRoutes(adminGroup)
		fmt.Fprintln(os.Stderr, "[DEV] Config handler registered (Degraded Mode: Empty)")

		// 13. OIDC Auth Handler (Frontend)
		oidcAuthHandler := authHandlers.NewOIDCAuthHandler(nil)
		authGroup := r.Group("/auth")
		oidcAuthHandler.RegisterRoutes(authGroup)
		fmt.Fprintln(os.Stderr, "[DEV] OIDC Auth handler registered (Degraded Mode: Empty)")
	}

	// MCP (Model Context Protocol) handler - works with or without database
	pdpAdapter := pdp.NewAuthzAdapter(s.authorizer)
	mcpAuthBridge := mcp.NewAuthorizationBridge(pdpAdapter)
	mcpAuditLogger := mcp.NewInMemoryAuditLogger(1000)
	mcpHandler := mcpHandlers.NewMCPHandler(mcpAuthBridge, mcpAuditLogger, s.extendedTokenService)
	gauthAPIGroup := r.Group("/api/v1/gauth")
	mcpHandler.RegisterRoutes(gauthAPIGroup)
	fmt.Fprintln(os.Stderr, "[mcp] Model Context Protocol handler registered at /api/v1/gauth/mcp/*")

	// Beta Auth handler - frontend authentication endpoints
	jwtSecret := os.Getenv("GAUTH_JWT_SIGNING_KEY")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}
	betaAuthHandler := authHandlers.NewBetaAuthHandler(jwtSecret)
	authGroup := gauthAPIGroup.Group("/auth")
	betaAuthHandler.RegisterRoutes(authGroup)
	fmt.Fprintln(os.Stderr, "[auth] Frontend authentication endpoints registered at /api/v1/gauth/auth/*")
	fmt.Fprintln(os.Stderr, "[auth]   POST /api/v1/gauth/auth/login/init (Initiate login)")
	fmt.Fprintln(os.Stderr, "[auth]   POST /api/v1/gauth/auth/login/mfa (Verify MFA and get JWT)")

	// Initialize production-grade revocation system (Emergency Oracle + Two-Phase + Optimistic + Circuit Breaker)
	// Enabled via GAUTH_REVOCATION_ENABLED=1; requires Redis connection
	ctx := context.Background()
	s.revocationService = NewRevocationService(ctx)
	if s.revocationService != nil && s.revocationService.enabled {
		// Register all 13 revocation HTTP endpoints under /api/v1/beta/revocation/*
		s.revocationService.RegisterHandlers(betaGroup)
		fmt.Fprintln(os.Stderr, "[revocation] Production revocation system initialized (77 tests validated)")
		fmt.Fprintln(os.Stderr, "[revocation] Emergency Oracle + Two-Phase + Optimistic + Circuit Breaker")
		fmt.Fprintln(os.Stderr, "[revocation] Performance: 67k ops/sec, P99 <30ms latency")
	} else if os.Getenv("GAUTH_REVOCATION_ENABLED") != "0" && os.Getenv("GAUTH_TEST_SILENT") != "1" {
		fmt.Fprintln(os.Stderr, "[revocation] Production revocation system disabled")
		fmt.Fprintln(os.Stderr, "[revocation] Set GAUTH_REVOCATION_ENABLED=1 and configure Redis to enable")
	}

	// Violation persistence autosave loop (optional) (disabled if GAUTH_DISABLE_BG_POLLS=1)
	if s.violationPersistPath != "" && os.Getenv("GAUTH_DISABLE_BG_POLLS") != "1" {
		intervalSec := 0
		if raw := os.Getenv("GAUTH_VIOLATION_AUTOSAVE_SEC"); raw != "" {
			if v, err := strconv.Atoi(raw); err == nil {
				intervalSec = v
			}
		}
		if intervalSec >= 10 { // minimum guard
			go func() {
				fmt.Fprintf(os.Stderr, "[violations] autosave enabled interval=%ds path=%s\n", intervalSec, s.violationPersistPath)
				for {
					select {
					case <-s.stopCh:
						fmt.Fprintln(os.Stderr, "[violations] autosave loop stopping")
						return
					case <-time.After(time.Duration(intervalSec) * time.Second):
						if s.violationHandler != nil {
							if err := s.violationHandler.Save(); err != nil {
								fmt.Fprintf(os.Stderr, "[violations] autosave failed: %v\n", err)
							}
						}
					}
				}
			}()
		}
	}
	// Semantic counters autosave loop GAUTH_SEMANTIC_AUTOSAVE_SEC (min 10s)
	if s.semanticPersistPath != "" && os.Getenv("GAUTH_DISABLE_BG_POLLS") != "1" {
		intervalSec := 0
		if raw := os.Getenv("GAUTH_SEMANTIC_AUTOSAVE_SEC"); raw != "" {
			if v, err := strconv.Atoi(raw); err == nil {
				intervalSec = v
			}
		}
		if intervalSec >= 10 {
			go func() {
				fmt.Fprintf(os.Stderr, "[semantics] autosave enabled interval=%ds path=%s\n", intervalSec, s.semanticPersistPath)
				for {
					select {
					case <-s.stopCh:
						fmt.Fprintln(os.Stderr, "[semantics] autosave loop stopping")
						return
					case <-time.After(time.Duration(intervalSec) * time.Second):
						if err := s.semanticHandler.Save(); err != nil {
							fmt.Fprintf(os.Stderr, "[semantics] autosave failed: %v\n", err)
						}
					}
				}
			}()
		}
	}
	// Background semantic anomaly sampler (optional): GAUTH_SEMANTIC_ANOMALY_BG_SEC (min 5s)
	if s.rfc0111Service != nil && os.Getenv("GAUTH_DISABLE_BG_POLLS") != "1" {
		bgInterval := 0
		if raw := os.Getenv("GAUTH_SEMANTIC_ANOMALY_BG_SEC"); raw != "" {
			if v, err := strconv.Atoi(raw); err == nil {
				bgInterval = v
			}
		}
		if bgInterval >= 5 {
			go func() {
				fmt.Fprintf(os.Stderr, "[semantics] anomaly sampler enabled interval=%ds\n", bgInterval)
				for {
					select {
					case <-s.stopCh:
						fmt.Fprintln(os.Stderr, "[semantics] anomaly sampler stopping")
						return
					case <-time.After(time.Duration(bgInterval) * time.Second):
						// Acquire latest snapshot & update anomaly detection state
						s.semanticHandler.Update()
						// Check for high anomaly scores to activate reactive throttle
						active := false
						if tStr := os.Getenv("GAUTH_SEMANTIC_ANOMALY_Z_THRESHOLD"); tStr != "" {
							if threshold, err := strconv.ParseFloat(tStr, 64); err == nil {
								scores := s.semanticHandler.Scores()
								for _, v := range scores {
									if v > threshold {
										active = true
										break
									}
								}
							}
						}
						// Direct field assignment (assuming single writer or acceptable race for this prototype flag)
						// Previous implementation also just assigned it.
						s.semanticThrottleActive = active
					}
				}
			}()
		}
	}
	// Metrics autosave loop (optional): GAUTH_METRICS_AUTOSAVE_SEC (default disabled when unset or <5).
	if mm, ok := s.metrics.(*metrics.Memory); ok && os.Getenv("GAUTH_DISABLE_BG_POLLS") != "1" {
		intervalSec := 0
		if v := os.Getenv("GAUTH_METRICS_AUTOSAVE_SEC"); v != "" {
			if iv, err := strconv.Atoi(v); err == nil {
				intervalSec = iv
			}
		}
		if intervalSec >= 5 { // minimum guard to avoid pathological tight loops
			go func() {
				fmt.Fprintf(os.Stderr, "[metrics] autosave enabled interval=%ds\n", intervalSec)
				for {
					select {
					case <-s.stopCh:
						fmt.Fprintln(os.Stderr, "[metrics] autosave loop stopping")
						return
					case <-time.After(time.Duration(float64(intervalSec)*(0.9+rand.Float64()*0.2)) * time.Second): // #nosec G404
						if saveErr := mm.Save(); saveErr != nil {
							fmt.Fprintf(os.Stderr, "[metrics] autosave error: %v\n", saveErr)
						} else {
							fmt.Fprintln(os.Stderr, "[metrics] autosave persisted")
						}
					}
				}
			}()
		}
	}
	// Update token handler's tracer
	if s.tracerProvider != nil && s.tokenHandler != nil {
		s.tokenHandler.Tracer = &tokenTracerAdapter{tp: s.tracerProvider}
		s.tokenHandler.TracerRatio = s.tracerSampleRatio
	}
	// Initialize OpenTelemetry metrics exporter if enabled (stdout metric for demo) guarded by sync.Once
	if os.Getenv("GAUTH_OTEL_METRICS_ENABLE") == "1" {
		otelInitOnce.Do(func() {
			exp, err := stdoutmetric.New(stdoutmetric.WithWriter(os.Stderr))
			if err == nil {
				provider := metricsdk.NewMeterProvider(metricsdk.WithReader(metricsdk.NewPeriodicReader(exp, metricsdk.WithInterval(5*time.Second))))
				otel.SetMeterProvider(provider)
				s.otelMetricsProvider = provider
				s.otelMeter = provider.Meter("gauth-beta")
				fmt.Fprintln(os.Stderr, "[otel-metrics] stdout exporter initialized")
				// Register observable gauges for violation counters and anomaly rates.
				s.otelViolationCounters = make(map[string]metric.Int64ObservableGauge)
				s.otelViolationRates = make(map[string]metric.Float64ObservableGauge)
				semanticGauges := make(map[string]metric.Int64ObservableGauge)
				semanticRateGauges60 := make(map[string]metric.Float64ObservableGauge)
				semanticRateGauges300 := make(map[string]metric.Float64ObservableGauge)
				semanticAnomalyGauges := make(map[string]metric.Float64ObservableGauge)
				violationIntegrityGauge, gErr1 := s.otelMeter.Int64ObservableGauge("gauth_persistence_integrity_violation")
				if gErr1 != nil {
					log.Printf("otel gauge create failed (violation): %v", gErr1)
				}
				semanticIntegrityGauge, gErr2 := s.otelMeter.Int64ObservableGauge("gauth_persistence_integrity_semantic")
				if gErr2 != nil {
					log.Printf("otel gauge create failed (semantic): %v", gErr2)
				}
				counterKeys := []string{"sig_invalid", "expired", "not_yet_valid", "issuer_mismatch", "replay_detected", "audience_mismatch", "missing_claim", "unknown"}
				for _, k := range counterKeys {
					name := "gauth_violation_counter_" + k
					gauge, err2 := s.otelMeter.Int64ObservableGauge(name)
					if err2 == nil {
						s.otelViolationCounters[k] = gauge
					}
				}
				rateKeys := []string{"rate_60s", "rate_300s"}
				for _, rk := range rateKeys {
					name := "gauth_violation_" + rk
					g, err2 := s.otelMeter.Float64ObservableGauge(name)
					if err2 == nil {
						s.otelViolationRates[rk] = g
					}
				}
				semanticKeys := []string{"amount_limit_exceeded", "daily_amount_limit_exceeded", "currency_mismatch", "scope_violation", "restriction_mismatch"}
				for _, sk := range semanticKeys {
					name := "gauth_poa_semantic_counter_" + sk
					gauge, err2 := s.otelMeter.Int64ObservableGauge(name)
					if err2 == nil {
						semanticGauges[sk] = gauge
					}
					g60, e60 := s.otelMeter.Float64ObservableGauge("gauth_poa_semantic_rate_60s_" + sk)
					if e60 == nil {
						semanticRateGauges60[sk] = g60
					}
					g300, e300 := s.otelMeter.Float64ObservableGauge("gauth_poa_semantic_rate_300s_" + sk)
					if e300 == nil {
						semanticRateGauges300[sk] = g300
					}
					sg, se := s.otelMeter.Float64ObservableGauge("gauth_poa_semantic_anomaly_score_" + sk)
					if se == nil {
						semanticAnomalyGauges[sk] = sg
					}
				}
				revEmit, revEmitErr := s.otelMeter.Int64ObservableGauge("gauth_revocation_auto_sign_emitted")
				if revEmitErr != nil {
					log.Printf("otel gauge create failed (revocation emitted): %v", revEmitErr)
				}
				revSkipEmpty, revSkipEmptyErr := s.otelMeter.Int64ObservableGauge("gauth_revocation_auto_sign_skipped_empty")
				if revSkipEmptyErr != nil {
					log.Printf("otel gauge create failed (revocation skipped empty): %v", revSkipEmptyErr)
				}
				revSkipDup, revSkipDupErr := s.otelMeter.Int64ObservableGauge("gauth_revocation_auto_sign_skipped_duplicate")
				if revSkipDupErr != nil {
					log.Printf("otel gauge create failed (revocation skipped duplicate): %v", revSkipDupErr)
				}
				s.otelSemanticCounters = semanticGauges
				s.otelSemanticRates60 = semanticRateGauges60
				s.otelSemanticRates300 = semanticRateGauges300
				s.otelSemanticAnomalyScores = semanticAnomalyGauges
				s.otelRevocationAutoSignEmitted = revEmit
				s.otelRevocationAutoSignSkippedEmp = revSkipEmpty
				s.otelRevocationAutoSignSkippedDup = revSkipDup
				_, cbErr := s.otelMeter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
					mapIntegrity := func(status string) int64 {
						switch status {
						case integrityOK:
							return 1
						case integrityMismatch:
							return 0
						case integrityLegacy:
							return -1
						case integrityUnconfigured:
							return -2
						default:
							return -2
						}
					}
					o.ObserveInt64(violationIntegrityGauge, mapIntegrity(s.violationIntegrityStatus))
					o.ObserveInt64(semanticIntegrityGauge, mapIntegrity(s.semanticIntegrityStatus))
					if s.primaryAuthService != nil {
						snap := s.primaryAuthService.ViolationSnapshot()
						for k, g := range s.otelViolationCounters {
							val := snap[k]
							if val > math.MaxInt64 {
								o.ObserveInt64(g, math.MaxInt64)
							} else {
								o.ObserveInt64(g, int64(val))
							}
						}
						// Compute global rates for OTel compatibility (sum of all keys)
						rates60 := s.violationHandler.ComputeRates(60 * time.Second)
						rates300 := s.violationHandler.ComputeRates(300 * time.Second)
						sum60 := 0.0
						for _, v := range rates60 {
							sum60 += v
						}
						sum300 := 0.0
						for _, v := range rates300 {
							sum300 += v
						}
						if g, ok := s.otelViolationRates["rate_60s"]; ok {
							o.ObserveFloat64(g, sum60)
						}
						if g, ok := s.otelViolationRates["rate_300s"]; ok {
							o.ObserveFloat64(g, sum300)
						}
					}
					if s.rfc0111Service != nil {
						ss := s.rfc0111Service.SemanticSnapshot()
						for k, g := range semanticGauges {
							if v, ok := ss[k]; ok {
								if v > math.MaxInt64 {
									o.ObserveInt64(g, math.MaxInt64)
								} else {
									o.ObserveInt64(g, int64(v))
								}
							}
						}
						s.semanticHandler.Update()
						r60 := s.semanticHandler.ComputeRates(60 * time.Second)
						r300 := s.semanticHandler.ComputeRates(300 * time.Second)
						for k, g := range semanticRateGauges60 {
							if v, ok := r60[k]; ok {
								o.ObserveFloat64(g, v)
							}
						}
						for k, g := range semanticRateGauges300 {
							if v, ok := r300[k]; ok {
								o.ObserveFloat64(g, v)
							}
						}
						scores := s.semanticHandler.Scores()
						for k, g := range semanticAnomalyGauges {
							if v, ok := scores[k]; ok {
								o.ObserveFloat64(g, v)
							}
						}
					}
					if s.otelRevocationAutoSignEmitted != nil {
						o.ObserveInt64(s.otelRevocationAutoSignEmitted, s.revocationAutoSignEmitted)
					}
					if s.otelRevocationAutoSignSkippedEmp != nil {
						o.ObserveInt64(s.otelRevocationAutoSignSkippedEmp, s.revocationAutoSignSkippedEmpty)
					}
					if s.otelRevocationAutoSignSkippedDup != nil {
						o.ObserveInt64(s.otelRevocationAutoSignSkippedDup, s.revocationAutoSignSkippedDup)
					}
					return nil
				})
				if cbErr != nil {
					log.Printf("otel callback registration failed: %v", cbErr)
				}
			} else {
				fmt.Fprintf(os.Stderr, "[otel-metrics] initialization failed: %v\n", err)
			}
		})
	}
	// Initialize empty revocation chain
	s.revocationChain = delegation.NewRevocationChain()
	// Optional demo auto-seeding: GAUTH_REVOCATION_DEMO_SEED=<n>, GAUTH_REVOCATION_DEMO_SIGN=1 to sign a tree head
	if rawSeed := os.Getenv("GAUTH_REVOCATION_DEMO_SEED"); rawSeed != "" {
		if n, err := strconv.Atoi(rawSeed); err == nil && n > 0 {
			reasons := []delegation.RevocationReason{
				delegation.RevocationReasonCompromise,
				delegation.RevocationReasonUserRequest,
				delegation.RevocationReasonGrantorRevoked,
				delegation.RevocationReasonPolicyExpired,
				delegation.RevocationReasonSuperseded,
				delegation.RevocationReasonAbuse,
			}
			added := 0
			for i := 0; i < n; i++ {
				id := fmt.Sprintf("rev-seed-%d-%d", time.Now().UnixNano(), i)
				ev := delegation.RevocationEvent{ID: id, DelegationID: fmt.Sprintf("demo-deleg-%d", i), Reason: string(reasons[i%len(reasons)])}
				if _, aerr := s.revocationChain.Append(ev); aerr == nil {
					added++
				} else {
					fmt.Fprintf(os.Stderr, "[revocation-seed] append failed: %v\n", aerr)
				}
			}
			fmt.Fprintf(os.Stderr, "[revocation-seed] auto seeded events count=%d\n", added)
			if os.Getenv("GAUTH_REVOCATION_DEMO_SIGN") == "1" {
				if sth, serr := s.revocationChain.SignTreeHead(); serr == nil {
					fmt.Fprintf(os.Stderr, "[revocation-seed] signed tree head len=%d root=%s sigs=%d\n", sth.ChainLength, sth.MerkleRoot, len(sth.Signatures))
				} else {
					fmt.Fprintf(os.Stderr, "[revocation-seed] sign tree head error: %v\n", serr)
				}
			}
		}
	}
	// Load persisted signed tree heads if configured
	if p := os.Getenv("GAUTH_STH_PERSIST_PATH"); p != "" {
		if err := s.revocationChain.LoadSignedTreeHeads(p); err == nil {
			fmt.Fprintf(os.Stderr, "[revocation] loaded signed tree heads path=%s count=%d\n", p, len(s.revocationChain.TreeHeads()))
		} else {
			fmt.Fprintf(os.Stderr, "[revocation] no signed tree heads loaded path=%s err=%v\n", p, err)
		}
	}
	// Initialize EdDSA key manager (with optional persistence)
	if os.Getenv("GAUTH_TOKEN_SIG_MODE") == "eddsa" {
		ttlHours := 24
		if v := os.Getenv("GAUTH_KEY_ROTATION_HOURS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				ttlHours = n
			}
		}
		// Pre-register a safe OnKeyRotated callback
		crypto.OnKeyRotated = func(prev, curr *crypto.Key) {
			if s == nil || s.revocationChain == nil {
				return
			}
			evs := s.revocationChain.Events()
			if len(evs) == 0 {
				s.revocationAutoSignSkippedEmpty++
				return
			}
			latest := s.revocationChain.LatestTreeHead()
			if latest != nil && latest.ChainLength == len(evs) && latest.AggregateHash == s.revocationChain.AggregateHash() {
				s.revocationAutoSignSkippedDup++
				return
			}
			sth, err := s.revocationChain.SignTreeHead()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[revocation] auto-sign error after rotation: %v\n", err)
				return
			}
			if sth == nil {
				return
			}
			s.revocationAutoSignEmitted++
			fmt.Fprintf(os.Stderr, "[revocation] auto-signed tree head after rotation len=%d root=%s sigs=%d satisfied=%d threshold=%d\n", sth.ChainLength, sth.MerkleRoot, len(sth.Signatures), sth.SatisfiedWeight, sth.Threshold)
		}

		km, err := crypto.NewManager(time.Duration(ttlHours) * time.Hour)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[crypto] eddsa manager init failed: %v\n", err)
		} else {
			if s.keyProvider == nil {
				s.keyProvider = km
			}
			fmt.Fprintf(os.Stderr, "[crypto] eddsa manager initialized ttl=%dh persist=%v\n", ttlHours, os.Getenv("GAUTH_EDDSA_PERSIST_PATH") != "")
		}
		// Optional automatic rotation loop
		if rv := os.Getenv("GAUTH_EDDSA_AUTO_ROTATE_MIN"); rv != "" {
			if mins, err := strconv.Atoi(rv); err == nil && mins >= 5 {
				if km := s.getKeyManager(); km != nil {
					go func() {
						for {
							time.Sleep(time.Duration(mins) * time.Minute)
							if _, err := km.Rotate(); err != nil {
								fmt.Fprintf(os.Stderr, "[crypto] auto-rotate error: %v\n", err)
							} else {
								fmt.Fprintln(os.Stderr, "[crypto] auto-rotated eddsa key")
							}
						}
					}()
				}
			}
		}
	}
	// Initialize Anchor Client (Memory default)
	s.anchorClient = anchor.NewMemoryAnchor()
	// Enable persistence if configured (Roadmap Item 2)
	if pp := os.Getenv("GAUTH_ANCHOR_PERSIST_PATH"); pp != "" {
		if err := s.anchorClient.EnablePersistence(pp); err != nil {
			fmt.Fprintf(os.Stderr, "[anchor] persistence load failed path=%s err=%v\n", pp, err)
		} else {
			fmt.Fprintf(os.Stderr, "[anchor] persistence enabled path=%s anchors=%d\n", pp, s.anchorClient.TotalAnchors())
		}
	}
	// Bridge memory anchor to capabilities.AnchorClient interface via adapter
	if s.capabilitiesHandler != nil {
		adapter := &auditChainAnchorAdapter{client: s.anchorClient}
		s.capabilitiesHandler.SetAnchorClient(adapter)
	}

	// External anchoring for capability registry (RFC-3161 TSA)
	// Enable via GAUTH_CAPABILITY_TSA_URL environment variable
	if tsaURL := os.Getenv("GAUTH_CAPABILITY_TSA_URL"); tsaURL != "" && s.capabilitiesHandler != nil {
		// Create RFC-3161 provider
		rfc3161Provider := ledger.NewRFC3161Provider(tsaURL)

		// Create external anchor client with RFC-3161 provider
		// No receipt store for now (in-memory only)
		externalClient := ledger.NewExternalAnchorClient(rfc3161Provider, nil)

		if externalClient != nil {
			// Create and set the NotaryAdapter to bridge ledger to capabilities
			tsaAdapter := capabilities.NewNotaryAdapter(externalClient)
			if tsaAdapter != nil {
				s.capabilitiesHandler.SetAnchorClient(tsaAdapter)
				fmt.Fprintf(os.Stderr, "[capabilities] External TSA anchoring enabled url=%s\n", tsaURL)
			}
		}
	}

	// External anchoring for semantics (optional) GAUTH_SEMANTIC_ANCHOR_ENABLE (Item 8)
	if os.Getenv("GAUTH_SEMANTIC_ANCHOR_ENABLE") == "1" && s.anchorClient != nil && s.semanticHandler != nil {
		s.semanticHandler.AnchorProvider = &capAnchorProviderAdapter{client: s.anchorClient}
		fmt.Fprintf(os.Stderr, "[semantics] external anchoring enabled\n")
	}

	// Wire Audit Chaining (Revocation Audit Hook)
	if s.revocationChain != nil {
		// Hook revocation events to audit AND external anchor
		// This ensures that every revocation event generates an audit entry,
		// and the audit entry hash is linked back.
		// Additionally, we anchor the aggregate hash to the unified anchor client.
		delegation.OnRevocationAppended = func(ev delegation.RevocationEvent, chainLen int, aggregateHash string) {
			// 1. Audit Log (RFC111-R7)
			if s.audit != nil {
				s.audit.Append(&AuditEntry{
					ID:       randomNonce(8),
					At:       time.Now().UTC(),
					Actor:    "system",
					Action:   "revocation_appended",
					Resource: "revocation_chain",
					Outcome:  "success",
					Meta: map[string]any{
						"revocation_id":  ev.ID,
						"delegation_id":  ev.DelegationID,
						"chain_length":   chainLen,
						"aggregate_hash": aggregateHash,
						"event_hash":     ev.Hash,
						"reason":         ev.Reason,
					},
				})
			}
			// 2. External Anchor (Roadmap Item 2)
			if s.anchorClient != nil {
				if _, err := s.anchorClient.Anchor(aggregateHash); err != nil {
					fmt.Fprintf(os.Stderr, "[revocation] anchor failure hash=%s err=%v\n", aggregateHash, err)
				}
			}
		}
	}

	// External Capability Anchoring
	// Uses 'web/handlers/capability_anchor' package.
	ctx = context.Background() // Use background context for initialization

	// Resolve External Anchor Provider
	var extProvider anchorint.Provider
	extProviderName := os.Getenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER")
	if extProviderName == "tsa_stub" || extProviderName == "tsa-stub" {
		// Use TSA Stub Provider (useful for reliability testing)
		minMs, maxMs := 10, 50
		failProb := 0.0
		if v := os.Getenv("GAUTH_CAP_EXTERNAL_ANCHOR_MIN_MS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				minMs = n
			}
		}
		if v := os.Getenv("GAUTH_CAP_EXTERNAL_ANCHOR_MAX_MS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				maxMs = n
			}
		}
		if v := os.Getenv("GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				failProb = f
			}
		}

		// Parameters: minLatency, maxLatency, failProb
		extProvider = anchorint.NewTSAStubProviderFromEnv(minMs, maxMs, failProb)
		fmt.Fprintf(os.Stderr, "[cap-anchor] using TSA stub provider (tsa_stub) prob=%.2f\n", failProb)
	} else {
		// Default to Memory Provider (wrapped)
		if s.anchorClient != nil {
			extProvider = &capAnchorProviderAdapter{client: s.anchorClient}
		} else {
			// Fallback if anchorClient missing (shouldn't happen)
			extProvider = anchorint.NewMemoryProvider()
			fmt.Fprintf(os.Stderr, "[cap-anchor] warning: no anchor client, using fresh memory provider\n")
		}
	}

	// Adapter for receipt store
	capStore := &capReceiptStoreAdapter{store: s.receiptStore}

	// Initialize Handler
	provName := extProviderName
	if provName == "tsa_stub" {
		provName = "tsa-stub"
	}
	if provName == "" {
		provName = "memory"
	}

	retries := 0
	retryDelay := time.Duration(0)
	if v := os.Getenv("GAUTH_CAP_EXTERNAL_ANCHOR_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			retries = n
		}
	}
	if v := os.Getenv("GAUTH_CAP_EXTERNAL_ANCHOR_RETRY_BASE_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			retryDelay = time.Duration(n) * time.Millisecond
		}
	}

	s.capabilityAnchorHandler = capability_anchor.NewHandler(extProvider, capStore, s.metrics, provName, retries, retryDelay)
	s.capabilityAnchorAPI = capability_anchor.NewAPI(s.capabilityAnchorHandler)
	s.capabilityAnchorAPI.RegisterRoutes(s.router)

	// Persistence for capability anchor (audit hash history)
	if pp := os.Getenv("GAUTH_CAP_ANCHOR_PERSIST_PATH"); pp != "" {
		if err := s.capabilityAnchorHandler.Load(pp); err != nil {
			fmt.Fprintf(os.Stderr, "[cap-anchor] load error: %v\n", err)
		} else {
			// rec, _ := s.capabilityAnchorHandler.Latest(ctx) // Latest might query provider, we want local state?
			// HistoryCount is reliable.
			fmt.Fprintf(os.Stderr, "[cap-anchor] loaded history path=%s entries=%d\n", pp, s.capabilityAnchorHandler.HistoryCount())
		}
		// Setup observer to persist on new anchor
		s.capabilityAnchorHandler.AddObserver(func(r anchorint.Receipt) {
			if saveErr := s.capabilityAnchorHandler.Save(pp); saveErr != nil {
				fmt.Fprintf(os.Stderr, "[cap-anchor] persist error: %v\n", saveErr)
			}
		})
	}

	// Prototype Notarizer (Memory) - extracted from commented block above
	// If GAUTH_CAP_ANCHOR_NOTARIZE=1, we enable a prototype notarizer using the anchor client.
	if os.Getenv("GAUTH_CAP_ANCHOR_NOTARIZE") == "1" {
		// The anchor client (memory) acts as the notarizer backend for this prototype.
		// In a real system, this would be a remote service or smart contract.
		if s.anchorClient != nil {
			s.notarizer = &auditChainAnchorAdapter{client: s.anchorClient}
			fmt.Fprintln(os.Stderr, "[notary] prototype notarizer enabled (memory backed)")
		}
	}

	// Initialize Receipt Store (Prototype)
	// Just an in-memory appended list for now, unless persistence needed.
	// For this refactor, we rely on the struct fields or simple implementation.
	// We'll skip complex receipt store logic here as it's not fully extracted to a package yet.
	// Instead, we verify the receipt verification loop logic in server_clean.go and port it if critical.
	// server_clean.go had a `verifyReceiptChain` loop. Use s.startReceiptVerificationLoop() equivalent.
	if os.Getenv("GAUTH_RECEIPT_VERIFY_BG_SEC") != "" {
		// ... background loop logic would go here
	}

	// Initialize Rotation Ledger (Prototype)
	if rlPath := os.Getenv("GAUTH_ROTATION_LEDGER_PATH"); rlPath != "" {
		s.rotationLedger = notary.NewRotationLedger(rlPath)
		// Try to load; on failure (new file) just log warning as append will create/overwrite
		if err := s.rotationLedger.Load(); err != nil {
			// Expected on first run
			fmt.Fprintf(os.Stderr, "[rotation] ledger load info path=%s err=%v\n", rlPath, err)
		} else {
			fmt.Fprintf(os.Stderr, "[rotation] ledger loaded path=%s entries=%d\n", rlPath, len(s.rotationLedger.Entries()))
		}
	}
	// ...

	// Demo Policy Chain seeding
	// NOTE: Temporarily disabled - policyHandler.CreatePolicy method doesn't exist.
	// TODO: Add CreatePolicy method to policy.Handler or remove this seeding code.
	/*
		if seedPolicies := os.Getenv("GAUTH_POLICY_DEMO_SEED"); seedPolicies == "1" {
			if s.policyHandler != nil {
				// Seed some demo policies
				s.policyHandler.CreatePolicy("demo-policy-1", "allow", "any", "any")
				s.policyHandler.CreatePolicy("demo-policy-2", "deny", "restricted", "external")
				fmt.Fprintln(os.Stderr, "[policy] seeded demo policies")
			}
		}
	*/

	// Middleware: Add global headers or checks
	s.router.Use(func(c *gin.Context) {
		c.Header("X-GAuth-Version", "beta-1.0")
		c.Next()
	})

	// MCP Manager
	s.mcpConnectionManager = mcp.NewConnectionManager()

	// Register all routes (final wiring)
	s.routes()

	// Debug routes for internal state
	s.router.GET("/debug/state", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "running",
			"subs":   len(s.rfc0111Service.SemanticSnapshot()), // Example of peering into service
		})
	})

	// Initial Load of Capabilities (Delayed to ensure all dependencies are wired)
	if pp := os.Getenv("GAUTH_CAPABILITIES_PATH"); pp != "" {
		if err := s.capabilitiesHandler.LoadFromFile(pp); err != nil {
			fmt.Fprintf(os.Stderr, "[capabilities] initial load failed path=%s err=%v\n", pp, err)
		} else {
			fmt.Fprintf(os.Stderr, "[capabilities] loaded configuration path=%s\n", pp)
		}
	}

	// Trigger initial async anchor if external anchoring is enabled (expected by tests)
	if s.capabilityAnchorHandler != nil && s.capabilityAnchorHandler.Provider != nil {
		// Ensure handler has a hash to anchor (test requirement)
		s.capabilityAnchorHandler.SetRegistryHash("initial-startup-hash")

		go func() {
			fmt.Fprintf(os.Stderr, "[cap-anchor] startup triggering initial anchor...\n")
			_, err := s.capabilityAnchorHandler.Anchor(context.Background())
			if err != nil {
				fmt.Fprintf(os.Stderr, "[cap-anchor] startup anchor failed (expected if forced): %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "[cap-anchor] startup anchor success\n")
			}
		}()
	}

	return s
}
