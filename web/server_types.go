package web

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	anchorint "github.com/mauriciomferz/AgentAuth/internal/anchor"
	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/internal/notary"
	"github.com/mauriciomferz/AgentAuth/internal/tracing"
	anchor "github.com/mauriciomferz/AgentAuth/pkg/anchor"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/blockchain"
	"github.com/mauriciomferz/AgentAuth/pkg/common/clock"
	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
	"github.com/mauriciomferz/AgentAuth/pkg/database"
	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
	"github.com/mauriciomferz/AgentAuth/pkg/gauth"
	"github.com/mauriciomferz/AgentAuth/pkg/gauth_aap_001"
	"github.com/mauriciomferz/AgentAuth/pkg/mcp"
	"github.com/mauriciomferz/AgentAuth/pkg/redis"
	"github.com/mauriciomferz/AgentAuth/web/handlers/admin"
	authzHandlers "github.com/mauriciomferz/AgentAuth/web/handlers/authz"
	"github.com/mauriciomferz/AgentAuth/web/handlers/capabilities"
	"github.com/mauriciomferz/AgentAuth/web/handlers/capability_anchor"
	delegationHandlers "github.com/mauriciomferz/AgentAuth/web/handlers/delegation"
	eventsHandlers "github.com/mauriciomferz/AgentAuth/web/handlers/events"
	"github.com/mauriciomferz/AgentAuth/web/handlers/examples"
	modellimits "github.com/mauriciomferz/AgentAuth/web/handlers/modellimits"
	poaHandlers "github.com/mauriciomferz/AgentAuth/web/handlers/poa"
	policyHandler "github.com/mauriciomferz/AgentAuth/web/handlers/policy"
	"github.com/mauriciomferz/AgentAuth/web/handlers/semantic"
	"github.com/mauriciomferz/AgentAuth/web/handlers/token"
	"github.com/mauriciomferz/AgentAuth/web/handlers/violations"
	"go.opentelemetry.io/otel/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
)

type BetaServer struct {
	router               *gin.Engine
	mu                   sync.RWMutex
	gauthPlusInitialized bool
	start                time.Time
	keyProvider          crypto.KeyProvider
	// legacyAliasHits counts invocations of deprecated /api/governance/lifecycle_timeline for deprecation timing.

	legacyAliasHits  uint64
	examplesAPI      *examples.API
	audit            *AuditLog
	events           *EventHub
	eventsHubAdapter *eventsHandlers.Hub // New events hub for extracted handlers
	// Primary token service (optional). When set, exposes violation counters.
	primaryAuthService interface{ ViolationSnapshot() map[string]uint64 }

	// Handlers
	poaHandler        *poaHandlers.Handler
	delegationHandler *delegationHandlers.Handler
	tokenHandler      *token.Handler

	tokens *token.Store
	// stopCh closes to signal background goroutines to exit gracefully.
	stopCh  chan struct{}
	stopped atomic.Bool // indicates Shutdown() invoked
	// AAP001 delegation service (prototype) to surface semantic counters and dual-control revocation workflow methods.
	// These are no-ops when GAUTH_DISABLE_AAP001_SERVICE=1 (service nil) and handlers will fail closed.
	aap001Service interface {
		SemanticSnapshot() map[string]uint64
		InitiateRevocation(ctx context.Context, req gauth_aap_001.RevocationRequest) error
		ApproveRevocation(ctx context.Context, poaID, approver string) error
		CancelRevocation(ctx context.Context, poaID, actor string) error
	}
	// RFC service reference for hierarchical digest features and delegation graph building
	rfcService interface {
		BuildDelegationGraph(ctx context.Context) ([]gauth_aap_001.DelegationGraphNode, error)
		AttachEvidenceHashes(ctx context.Context, poaID string, hashes []string) (*gauth_aap_001.PowerOfAttorney, error)
		ListDelegations(userID string) ([]*gauth_aap_001.PowerOfAttorney, error)
	}
	violationHandler *violations.Handler
	violationAPI     *violations.API

	violationPersistPath string // Keep for path passing if needed or just use handler
	// Semantic counters persistence path & last persist timestamp (separate file for clarity)
	semanticPersistPath string
	semanticLastPersist time.Time
	// Semantic Anomaly Detection
	semanticHandler *semantic.Handler
	semanticAPI     *semantic.API

	// Capability Anchor Handler
	capabilityAnchorHandler *capability_anchor.Handler
	capabilityAnchorAPI     *capability_anchor.API

	// Capabilities Handler (New)
	capabilitiesHandler *capabilities.Handler
	capabilitiesAPI     *capabilities.API

	authzAPI *authzHandlers.API

	// Admin Handlers
	adminTokenHandler *admin.TokenHandler
	apiKeyHandler     *admin.APIKeyHandler
	resilienceHandler *admin.ResilienceHandler

	// OTel Metrics (prototype)
	// Persistence integrity tracking (hash chain)
	violationPrevHash string
	semanticPrevHash  string
	// Latest persistence integrity verification status codes (string form). Possible values:
	// "ok" | "mismatch" | "legacy" | "unconfigured". Mapped to numeric gauges for metrics:
	// ok=1 mismatch=0 legacy=-1 unconfigured=-2
	violationIntegrityStatus string
	semanticIntegrityStatus  string
	// Delegation lifecycle prototype store (id -> status). Real implementation would live in AAP001 service repo.
	port string // bound address (":8080" style)
	// Policy Handler (Refactored)
	policyHandler *policyHandler.Handler
	policyAPI     *policyHandler.API

	// Embedded MemoryAuthorizer for advanced metrics exposure & demo evaluation
	authorizer *authz.MemoryAuthorizer
	// Replay protection for issuance nonces/JTIs (demo only)
	replayStore *token.ReplayNonceStore
	// Optional anchoring client (memory prototype); future: pluggable external providers
	anchorClient *anchor.MemoryAnchor
	// Delegation revocation chain prototype (global for demo token space)
	revocationChain *delegation.RevocationChain
	// Revocation anchor idempotency tracking (hash of sha256(merkle_root) and last merkle root)
	revocationLastAnchorHash string
	revocationLastAnchorRoot string
	// Production revocation service (Emergency Oracle + Two-Phase + Optimistic + Circuit Breaker)
	revocationService *RevocationService
	// Metrics collector (in-memory) for lifecycle & multi-signature instrumentation
	metrics        metrics.Metrics
	tracerProvider *tracing.TracerProvider
	// OTEL metrics objects (initialized if GAUTH_OTEL_METRICS_ENABLE=1)
	otelMeter             metric.Meter
	otelViolationCounters map[string]metric.Int64ObservableGauge   // gauge snapshots for counters
	otelViolationRates    map[string]metric.Float64ObservableGauge // per-minute anomaly rates
	// OTEL semantic counters and per-minute rates (added for per-category semantic anomaly detection)
	otelSemanticCounters map[string]metric.Int64ObservableGauge
	otelSemanticRates60  map[string]metric.Float64ObservableGauge // per-category 60s window rates
	otelSemanticRates300 map[string]metric.Float64ObservableGauge // per-category 300s window rates
	// OTEL semantic anomaly score gauges (per-category EWMA z-score)
	otelSemanticAnomalyScores map[string]metric.Float64ObservableGauge
	otelMetricsProvider       *metricsdk.MeterProvider
	// Revocation auto-sign metrics (OTEL counters when enabled)
	otelRevocationAutoSignEmitted    metric.Int64ObservableGauge
	otelRevocationAutoSignSkippedEmp metric.Int64ObservableGauge
	otelRevocationAutoSignSkippedDup metric.Int64ObservableGauge
	revocationAutoSignEmitted        int64
	revocationAutoSignSkippedEmpty   int64
	revocationAutoSignSkippedDup     int64
	// Lifecycle timeline (in-memory ring buffers per entity id)
	lifecycleMu     sync.RWMutex
	lifecycleEvents map[string][]*LifecycleEvent // key: entityType+":"+id -> slice (append-only capped)
	lifecycleCap    int
	// JWKS metadata tracking for discovery integrity
	jwksSignature   string
	jwksETag        string
	jwksLastRotated time.Time
	// Deprecation metadata (Discovery Hardening RFC compliance)
	deprecationSchedule map[string]time.Time
	// Capability governance fields
	// SLA freshness tracking for capability anchor (stale detection)
	// Capability audit hash chain persistence (tracking capability-related audit entries: create, revoke, enforce)
	// Moved to capabilities.Handler
	// Protocol flow manager for interactive AgentAuth flow guidance
	protocolFlowManager *ProtocolFlowManager
	// Capability registry external anchor artifact (periodic file emission)
	capAnchorFilePath      string
	capAnchorLastWrite     time.Time
	capAnchorWriteInterval time.Duration
	capAnchorEmitted       bool   // Whether artifact was emitted after last reload
	capAnchorArtifact      []byte // Latest emitted artifact bytes
	// Capability anchor observers (external timestamping / publication hooks)
	capAnchorObservers []CapabilityAnchorObserver
	// Emission interval jitter tracking (rolling window)
	capIntervalMu    sync.Mutex
	capIntervals     []time.Duration // recent successful emission intervals (len<=20)
	capIntervalM2    float64         // Welford variance accumulator
	capIntervalMean  float64         // Welford mean
	capIntervalCount int             // samples ingested
	// External notarization (prototype) fields
	notarizer interface {
		Notarize(string) (notary.Receipt, error)
	} // pluggable; memory when GAUTH_CAP_ANCHOR_NOTARIZE=1
	capLastNotarization        time.Time      // timestamp of last successful notarization
	capLastNotarizationReceipt notary.Receipt // last receipt (prototype)
	// Notarization receipt persistence (hash-chain)
	receiptStore interface {
		Append(notary.Receipt) (notary.StoredReceipt, error)
		Latest() notary.StoredReceipt
		Entries() []notary.StoredReceipt
		Load() error
	} // extended for verification
	// Receipt chain integrity status (ok|mismatch|unconfigured|empty)
	receiptIntegrityStatus  string
	receiptIntegrityVersion int
	receiptLastVerify       time.Time // timestamp of last integrity verification (metrics custom endpoint freshness)
	// Adaptive verification state
	adaptiveLastAdjust    time.Time
	adaptiveIntervalSec   int // current dynamic interval for auto verification
	adaptiveMismatchCount int // recent mismatch detections (reset after successful verify)
	adaptiveAppendCount   int // receipts appended since last adjust
	// Rotation ledger (optional) for dedicated chain separate from receipts when GAUTH_ROTATION_LEDGER_PATH set
	rotationLedger interface {
		Load() error
		AppendDescriptor(*notary.KeyRotationDescriptor) (notary.RotationLedgerRecord, error)
		Entries() []notary.RotationLedgerRecord
		HeadHash() string
	}
	rotationLastAnchoredHash string // last anchored rotation head (avoid duplicate anchors)
	// rotationLastV2Hash stores the previous artifact hash (canonical digest) for rotation V2 chaining.
	rotationLastV2Hash string
	// Tracing sampling ratio for capability diff endpoint (0 or negative => always sample)
	tracerSampleRatio float64
	// Capability diff snapshots ring buffer (optional)
	capDiffSnapshots *capSnapshots
	// Reactive semantic throttle activation flag (test instrumentation)
	semanticThrottleActive bool
	// External capability registry anchoring provider (distinct from pkg/anchor hash history)
	// Defined in handlers

	externalReceiptLastVerify time.Time // last integrity verification timestamp
	// Combined anchor chain (in-memory append-only for capability+rotation digest)
	combinedAnchorMu    sync.Mutex
	combinedAnchorChain []combinedAnchorEntry
	// Rotation V2 continuity tracking (test support). Stores last artifact canonical digest.
	rotationV2LastHash string

	// MCP connection manager for Phase 2B MCP Integration
	mcpConnectionManager *mcp.ConnectionManager
	// Database connection pool (optional, for persistent audit logging and deep health checks)
	db *database.DB

	// Redis client for token blacklist, rate limiting, etc.
	redisClient *redis.Client

	// Blockchain components (Active if GAUTH_ETH_RPC_URL is set)
	blockchainRegistry blockchain.BlockchainRegistry
	syncService        blockchain.SyncService

	// Model Limits Handler (refactored)
	modelLimitsHandler *modellimits.Handler
	modelLimitsAPI     *modellimits.API

	// System Clock Monitor (RR-015)
	systemClockMonitor *clock.SystemClockMonitor

	// Extended Token Service for RFC-0111 validation
	extendedTokenService *gauth.ExtendedTokenService
}

// auditChainAnchorAdapter adapts anchor.Provider, anchor.AnchorClient and capabilities.AnchorClient interfaces.
// It bridges the gap between different anchor definitions in the codebase by wrapping the memory anchor client.
// It implements TotalAnchors() which is required by capabilities.AnchorClient but missing in anchor.Provider.
type auditChainAnchorAdapter struct {
	client interface {
		Anchor(string) (anchor.AnchorRecord, error)
		LatestAnchor() (anchor.AnchorRecord, error)
		TotalAnchors() int
	}
}

func (a *auditChainAnchorAdapter) Anchor(hash string) (*notary.Receipt, error) {
	rec, err := a.client.Anchor(hash)
	if err != nil {
		return nil, err
	}
	// Convert anchor.AnchorRecord to notary.Receipt
	return &notary.Receipt{
		Hash:      rec.Hash,
		Timestamp: rec.AnchoredAt.Format(time.RFC3339),
		Provider:  rec.Provider,
		Version:   1, // Assuming version 1 for memory anchor records
	}, nil
}

func (a *auditChainAnchorAdapter) TotalAnchors() int64 {
	return int64(a.client.TotalAnchors())
}

// Notarize implements the notarizer interface by wrapping the Anchor method.
// This allows the adapter to be used as both an AnchorClient and a Notarizer.
func (a *auditChainAnchorAdapter) Notarize(hash string) (notary.Receipt, error) {
	start := time.Now()
	rec, err := a.client.Anchor(hash)
	elapsed := time.Since(start).Seconds()

	if err != nil {
		return notary.Receipt{}, err
	}
	prov := rec.Provider
	if prov == "" {
		prov = "memory"
	}
	return notary.Receipt{
		Hash:           rec.Hash,
		Timestamp:      rec.AnchoredAt.Format(time.RFC3339),
		Provider:       prov,
		Version:        1,
		Success:        true,
		LatencySeconds: elapsed,
	}, nil
}

// capAnchorProviderAdapter adapts pkg/anchor.AnchorClient to internal/anchor.Provider
type capAnchorProviderAdapter struct {
	client anchor.AnchorClient
}

func (a *capAnchorProviderAdapter) Anchor(hash string) (anchorint.Receipt, error) {
	r, err := a.client.Anchor(hash)
	if err != nil {
		return anchorint.Receipt{}, err
	}
	return anchorint.Receipt{Hash: r.Hash, Timestamp: r.AnchoredAt, Provider: r.Provider, Version: 1}, nil
}
func (a *capAnchorProviderAdapter) Latest() anchorint.Receipt {
	r, _ := a.client.LatestAnchor()
	return anchorint.Receipt{Hash: r.Hash, Timestamp: r.AnchoredAt, Provider: r.Provider, Version: 1}
}
func (a *capAnchorProviderAdapter) Verify(r anchorint.Receipt) error { return nil }

// capReceiptStoreAdapter adapts internal/notary.ReceiptStore to capability_anchor.ReceiptStore
type capReceiptStoreAdapter struct {
	store interface {
		Append(notary.Receipt) (notary.StoredReceipt, error)
		Latest() notary.StoredReceipt
		Entries() []notary.StoredReceipt
		Load() error
	}
}

func (a *capReceiptStoreAdapter) Append(r anchorint.ExternalAnchorReceipt) (anchorint.StoredExternalAnchorReceipt, error) {
	if a.store == nil {
		return anchorint.StoredExternalAnchorReceipt{}, nil
	}
	nr := notary.Receipt{Hash: r.Hash, Timestamp: r.Timestamp, Provider: r.Provider, Version: r.Version, Success: true}
	res, err := a.store.Append(nr)
	if err != nil {
		return anchorint.StoredExternalAnchorReceipt{}, err
	}
	return anchorint.StoredExternalAnchorReceipt{
		ExternalAnchorReceipt: r,
		PrevHash:              res.PrevHash,
		ChainHash:             res.ChainHash,
	}, nil
}

func (a *capReceiptStoreAdapter) Latest() anchorint.StoredExternalAnchorReceipt {
	if a.store == nil {
		return anchorint.StoredExternalAnchorReceipt{}
	}
	res := a.store.Latest()
	return anchorint.StoredExternalAnchorReceipt{
		ExternalAnchorReceipt: anchorint.ExternalAnchorReceipt{
			Hash:      res.Hash,
			Timestamp: res.Timestamp,
			Provider:  res.Provider,
			Version:   res.Version,
		},
		PrevHash:  res.PrevHash,
		ChainHash: res.ChainHash,
	}
}

func (a *capReceiptStoreAdapter) Entries() []anchorint.StoredExternalAnchorReceipt {
	if a.store == nil {
		return nil
	}
	entries := a.store.Entries()
	out := make([]anchorint.StoredExternalAnchorReceipt, len(entries))
	for i, e := range entries {
		out[i] = anchorint.StoredExternalAnchorReceipt{
			ExternalAnchorReceipt: anchorint.ExternalAnchorReceipt{
				Hash:      e.Hash,
				Timestamp: e.Timestamp,
				Provider:  e.Provider,
				Version:   e.Version,
			},
			PrevHash:  e.PrevHash,
			ChainHash: e.ChainHash,
		}
	}
	return out
}

func (a *capReceiptStoreAdapter) Load() error {
	if a.store == nil {
		return nil
	}
	return a.store.Load()
}

func (a *capReceiptStoreAdapter) VerifyIncremental() (string, int, string) {
	if a.store == nil {
		return "empty", -1, ""
	}
	if s, ok := a.store.(interface {
		VerifyIncremental() (string, int, string)
	}); ok {
		return s.VerifyIncremental()
	}
	return "unsupported", -1, ""
}
