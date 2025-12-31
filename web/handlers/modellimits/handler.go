package modellimits

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
	ratelimit "github.com/mauriciomferz/AgentAuth/pkg/limits"
)

// Notarizer interface handles external notarization
type Notarizer interface {
	Notarize(digest string) (Receipt, error)
}

// Receipt represents a notarization receipt
type Receipt struct {
	Provider       string  `json:"provider"`
	Timestamp      string  `json:"timestamp"`
	LatencySeconds float64 `json:"latency_seconds"`
	Success        bool    `json:"success"`
}

// Metrics interface for observability
type Metrics interface {
	IncModelLimitSurge()
	IncModelUnknown()
	IncModelLimitExceeded()
	IncModelOutputLimitExceeded()
	IncModelRateLimitExceeded()
	IncModelUserInputLimitExceeded()
	IncModelUserOutputLimitExceeded()
	IncModelUserRateLimitExceeded()
	RecordDecision(kind, id, result string, d time.Duration)
}

// KeyManager abstracts key operations needed by handler
type KeyManager interface {
	Active() *crypto.Key
	FindByID(id string) *crypto.Key
}

// Handler manages model limits governance and attestation
type Handler struct {
	// Configuration
	LimitsPath     string
	ReloadInterval time.Duration
	AuditPath      string
	AnchorPath     string
	AnchorInterval int
	StrictUnknown  bool
	SurgeFactor    float64
	SurgeMinEvents int

	// Dependencies
	KeyManager KeyManager // Optional for signing
	Notarizer  Notarizer  // Optional for notarization
	Metrics    Metrics    // Optional for metrics

	// Internal State
	mu                 sync.RWMutex
	limits             map[string]int
	outputLimits       map[string]int
	rateLimits         map[string]int
	rateState          map[string]*rateState
	rateLimitsExtended map[string][]struct {
		Limit  int
		Period time.Duration
	}
	rateStateExtended map[string]map[time.Duration]*rateStateExtended
	userLimits        map[string]map[string]struct{ InputLimit, OutputLimit, RateLimit int }
	userRateState     map[string]map[string]*rateState

	// Audit State
	auditMu         sync.Mutex
	auditPrevHash   string
	auditEntryCount int

	// Dynamic Rolad State
	lastMtime    time.Time
	snapshotHash string
	snapshotAt   time.Time

	// Surge State
	surgeMu          sync.Mutex
	surgeState       map[string][]int
	surgeLast        map[string]time.Time
	surgeLastTrigger time.Time

	// Anchor State
	anchorMu       sync.Mutex
	anchorPrevHash string

	// Verification State
	knownModels map[string]bool

	// Streaming
	streamMu       sync.Mutex
	streamSubs     map[chan ModelLimitsAttestation]struct{}
	streamCounts   map[string]uint64
	streamCountsMu sync.Mutex
}

// NewHandler creates a new model limits handler
func NewHandler(limitsPath, auditPath, anchorPath string) *Handler {
	h := &Handler{
		LimitsPath:     limitsPath,
		AuditPath:      auditPath,
		AnchorPath:     anchorPath,
		ReloadInterval: 5 * time.Minute, // Default
		SurgeFactor:    3.0,
		SurgeMinEvents: 10,
		limits:         make(map[string]int),
		outputLimits:   make(map[string]int),
		rateLimits:     make(map[string]int),
		rateState:      make(map[string]*rateState),
		rateLimitsExtended: make(map[string][]struct {
			Limit  int
			Period time.Duration
		}),
		rateStateExtended: make(map[string]map[time.Duration]*rateStateExtended),
		userLimits:        make(map[string]map[string]struct{ InputLimit, OutputLimit, RateLimit int }),
		userRateState:     make(map[string]map[string]*rateState),
		surgeState:        make(map[string][]int),
		surgeLast:         make(map[string]time.Time),
		streamSubs:        make(map[chan ModelLimitsAttestation]struct{}),
		streamCounts:      make(map[string]uint64),
		knownModels:       make(map[string]bool),
	}

	if os.Getenv("AGENTAUTH_MODEL_LIMITS_STRICT_UNKNOWN") == "1" {
		h.StrictUnknown = true
	}
	if s := os.Getenv("AGENTAUTH_MODEL_LIMIT_SURGE_FACTOR"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			h.SurgeFactor = v
		}
	}
	if s := os.Getenv("AGENTAUTH_MODEL_LIMIT_SURGE_MIN_EVENTS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			h.SurgeMinEvents = v
		}
	}
	return h
}

// Init loads initial limits and starts reload loop if interval > 0
func (h *Handler) Init(ctx context.Context) error {
	h.LoadFromDisk()
	if h.ReloadInterval > 0 && os.Getenv("AGENTAUTH_DISABLE_BG_POLLS") != "1" {
		go h.reloader(ctx)
	}
	return nil
}

func (h *Handler) reloader(ctx context.Context) {
	ticker := time.NewTicker(h.ReloadInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.LoadFromDisk()
		}
	}
}

// LoadFromDisk reads the limits JSON file and updates in-memory state
func (h *Handler) LoadFromDisk() bool {
	if h.LimitsPath == "" {
		return false
	}
	b, err := os.ReadFile(h.LimitsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[model-limits] read failed path=%s err=%v\n", h.LimitsPath, err)
		return false
	}

	info, err := os.Stat(h.LimitsPath)
	if err == nil && !info.ModTime().After(h.lastMtime) {
		return false // No change
	}

	raw, err := ParseModelLimitsJSON(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[model-limits] invalid JSON path=%s\n", h.LimitsPath)
		return false
	}

	// Calculate snapshot hash
	sh := sha256.Sum256(b)
	snapHash := fmt.Sprintf("sha256:%x", sh[:])

	h.mu.Lock()
	defer h.mu.Unlock()

	// Update state
	h.limits = make(map[string]int)
	h.outputLimits = make(map[string]int)
	h.rateLimits = make(map[string]int)
	h.rateLimitsExtended = make(map[string][]struct {
		Limit  int
		Period time.Duration
	})
	h.userLimits = make(map[string]map[string]struct{ InputLimit, OutputLimit, RateLimit int })
	h.knownModels = make(map[string]bool)

	for id, lim := range raw.ModelLimits {
		h.knownModels[id] = true
		if lim.MaxInputTokens > 0 {
			h.limits[id] = lim.MaxInputTokens
		}
		if lim.MaxOutputTokens > 0 {
			h.outputLimits[id] = lim.MaxOutputTokens
		}
		if lim.MaxRequestsPerMinute > 0 {
			h.rateLimits[id] = lim.MaxRequestsPerMinute
		}

		if len(lim.RateLimitsExtended) > 0 {
			for _, rateStr := range lim.RateLimitsExtended {
				rl, parseErr := ratelimit.ParseRateLimit(rateStr)
				if parseErr != nil {
					fmt.Fprintf(os.Stderr, "[model-limits] invalid rate limit %q for model %s: %v\n", rateStr, id, parseErr)
					continue
				}
				h.rateLimitsExtended[id] = append(h.rateLimitsExtended[id], struct {
					Limit  int
					Period time.Duration
				}{Limit: rl.Limit, Period: rl.Period})
			}
		}
	}

	for mid, users := range raw.UserLimits {
		if h.userLimits[mid] == nil {
			h.userLimits[mid] = make(map[string]struct{ InputLimit, OutputLimit, RateLimit int })
		}
		for uid, ulim := range users {
			h.userLimits[mid][uid] = struct{ InputLimit, OutputLimit, RateLimit int }{
				InputLimit:  ulim.MaxInputTokens,
				OutputLimit: ulim.MaxOutputTokens,
				RateLimit:   ulim.MaxRequestsPerMinute,
			}
		}
	}

	h.lastMtime = info.ModTime()
	h.snapshotHash = snapHash
	h.snapshotAt = time.Now()

	// Notify stream subscribers of config change
	go h.EmitAttestation("config_reload")

	return true
}

// ComputeSnapshot returns current config hash and generation time
func (h *Handler) ComputeSnapshot() (hash string, at time.Time, models map[string]int, users map[string]map[string]struct{ InputLimit, OutputLimit, RateLimit int }) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	// Return copies if needed, but for snapshot endpoint just returning maps is okay if caller serializes immediately
	// For thread safety during serialization we should probably return copies or hold lock.
	// The original implementation returned struct with copies?
	// s.computeModelLimitsSnapshot returned `snap struct { Models ..., Users ... }` and `hash`.
	// Let's simpler return hash and timestamp for now, or full data if needed by API.
	// API `apiModelLimitsSnapshot` needs models and users maps.
	// We'll return copies to be safe.
	// Simplification: just return hash and at. API will access state directly or we add `GetSnapshotData`.
	// Let's implement full return as the original did implicitly.

	// Deep copy maps
	ms := make(map[string]int, len(h.limits))
	for k, v := range h.limits {
		ms[k] = v
	}

	us := make(map[string]map[string]struct{ InputLimit, OutputLimit, RateLimit int }, len(h.userLimits))
	for k, v := range h.userLimits {
		sub := make(map[string]struct{ InputLimit, OutputLimit, RateLimit int }, len(v))
		for uk, uv := range v {
			sub[uk] = uv
		}
		us[k] = sub
	}

	return h.snapshotHash, h.snapshotAt, ms, us
}

// CheckLimit enforces limits for model request
func (h *Handler) CheckLimit(modelID, userID string, inputTokens, outputTokens int) LimitCheckResult {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check knowledge
	if !h.knownModels[modelID] {
		if h.LimitsPath != "" && h.StrictUnknown {
			if h.Metrics != nil {
				h.Metrics.RecordDecision("model_validate", modelID, "deny", time.Duration(0))
				h.Metrics.IncModelUnknown()
			}
			return LimitCheckResult{Allowed: false, Error: "model_unknown"}
		}
		// If unknown and not strict, allow (but skip logic)
		if h.Metrics != nil {
			h.Metrics.RecordDecision("model_validate", modelID, "allow", time.Duration(0))
		}
		return LimitCheckResult{Allowed: true, LimitEnforced: false}
	}

	// User overrides
	if userID != "" {
		var uLim struct{ InputLimit, OutputLimit, RateLimit int }
		if ml, ok := h.userLimits[modelID]; ok {
			uLim = ml[userID]
		}

		if uLim.InputLimit > 0 && inputTokens > uLim.InputLimit {
			h.writeAudit(modelID, "user_input", inputTokens, uLim.InputLimit, 0, 0, userID)
			if h.Metrics != nil {
				h.Metrics.RecordDecision("model_validate", modelID+":"+userID, "deny", time.Duration(0))
				h.Metrics.IncModelUserInputLimitExceeded()
			}
			return LimitCheckResult{Allowed: false, Error: "model_user_input_limit_exceeded", Limit: uLim.InputLimit}
		}

		if uLim.OutputLimit > 0 && outputTokens > uLim.OutputLimit {
			h.writeAudit(modelID, "user_output", outputTokens, uLim.OutputLimit, 0, 0, userID)
			if h.Metrics != nil {
				h.Metrics.RecordDecision("model_validate", modelID+":"+userID, "deny", time.Duration(0))
				h.Metrics.IncModelUserOutputLimitExceeded()
			}
			return LimitCheckResult{Allowed: false, Error: "model_user_output_limit_exceeded", Limit: uLim.OutputLimit}
		}

		if uLim.RateLimit > 0 {
			now := time.Now()
			if h.userRateState[modelID] == nil {
				h.userRateState[modelID] = make(map[string]*rateState)
			}
			st := h.userRateState[modelID][userID]
			if st == nil {
				st = &rateState{}
				h.userRateState[modelID][userID] = st
			}
			stWin := time.Unix(st.WindowStart, 0)
			if st.WindowStart == 0 || now.Sub(stWin) >= time.Minute {
				st.WindowStart = now.Unix()
				st.Count = 0
			}
			st.Count++
			if st.Count > uLim.RateLimit {
				h.writeAudit(modelID, "user_rate", st.Count, uLim.RateLimit, st.WindowStart, 60, userID)
				if h.Metrics != nil {
					h.Metrics.RecordDecision("model_validate", modelID+":"+userID, "deny", time.Duration(0))
					h.Metrics.IncModelUserRateLimitExceeded()
				}
				return LimitCheckResult{Allowed: false, Error: "model_user_rate_limit_exceeded", Limit: uLim.RateLimit, WindowSeconds: 60}
			}
		}
	}

	// Global limits
	limit, hasLimit := h.limits[modelID]
	if hasLimit && limit > 0 && inputTokens > limit {
		h.writeAudit(modelID, "input", inputTokens, limit, 0, 0, "")
		if h.Metrics != nil {
			h.Metrics.RecordDecision("model_validate", modelID, "deny", time.Duration(0))
			h.Metrics.IncModelLimitExceeded()
		}
		return LimitCheckResult{Allowed: false, Error: "model_limit_exceeded", Limit: limit}
	}

	if outLimit, ok := h.outputLimits[modelID]; ok && outLimit > 0 && outputTokens > outLimit {
		h.writeAudit(modelID, "output", outputTokens, outLimit, 0, 0, "")
		if h.Metrics != nil {
			h.Metrics.RecordDecision("model_validate", modelID, "deny", time.Duration(0))
			h.Metrics.IncModelOutputLimitExceeded()
		}
		return LimitCheckResult{Allowed: false, Error: "model_output_limit_exceeded", Limit: outLimit}
	}

	// Rate limiting
	if rateLimit, ok := h.rateLimits[modelID]; ok && rateLimit > 0 {
		now := time.Now()
		st := h.rateState[modelID]
		if st == nil {
			st = &rateState{}
			h.rateState[modelID] = st
		}
		stWin := time.Unix(st.WindowStart, 0)
		if st.WindowStart == 0 || now.Sub(stWin) >= time.Minute {
			st.WindowStart = now.Unix()
			st.Count = 0
		}
		st.Count++
		if st.Count > rateLimit {
			h.writeAudit(modelID, "rate", st.Count, rateLimit, st.WindowStart, 60, "")
			if h.Metrics != nil {
				h.Metrics.RecordDecision("model_validate", modelID, "deny", time.Duration(0))
				h.Metrics.IncModelRateLimitExceeded()
			}
			return LimitCheckResult{Allowed: false, Error: "model_rate_limit_exceeded", Limit: rateLimit, WindowSeconds: 60}
		}
	}

	// Extended rate limiting
	if exts, ok := h.rateLimitsExtended[modelID]; ok && len(exts) > 0 {
		now := time.Now()
		if h.rateStateExtended[modelID] == nil {
			h.rateStateExtended[modelID] = make(map[time.Duration]*rateStateExtended)
		}
		for _, rl := range exts {
			st := h.rateStateExtended[modelID][rl.Period]
			if st == nil {
				st = &rateStateExtended{}
				h.rateStateExtended[modelID][rl.Period] = st
			}
			stWin := time.Unix(st.WindowStart, 0)
			if st.WindowStart == 0 || now.Sub(stWin) >= rl.Period {
				st.WindowStart = now.Unix()
				st.Count = 0
			}
			st.Count++
			if st.Count > rl.Limit {
				h.writeAudit(modelID, "rate_extended", st.Count, rl.Limit, time.Now().Unix(), int(rl.Period.Seconds()), "")
				if h.Metrics != nil {
					h.Metrics.RecordDecision("model_validate", modelID, "deny", time.Duration(0))
					h.Metrics.IncModelRateLimitExceeded()
				}
				return LimitCheckResult{Allowed: false, Error: "model_rate_limit_exceeded", Limit: rl.Limit, WindowSeconds: int(rl.Period.Seconds()), Period: ratelimit.FormatPeriod(rl.Period)}
			}
		}
	}

	if h.Metrics != nil {
		h.Metrics.RecordDecision("model_validate", modelID, "allow", time.Duration(0))
	}
	return LimitCheckResult{Allowed: true, LimitEnforced: true, Limit: limit, RateLimit: h.rateLimits[modelID]}
}

// writeAudit appends an audit entry
func (h *Handler) writeAudit(modelID, kind string, provided, limit int, windowStart int64, windowSeconds int, userID string) {
	if h.AuditPath == "" {
		return
	}
	h.auditMu.Lock()
	entry := map[string]any{
		"ts":        time.Now().Unix(),
		"model_id":  modelID,
		"kind":      kind,
		"provided":  provided,
		"limit":     limit,
		"prev_hash": h.auditPrevHash,
		"hash":      "", // Placeholder for ordering? No, encoding/json sorts keys.
	}
	if userID != "" {
		entry["user_id"] = userID
	}
	if windowStart > 0 {
		entry["window_start"] = windowStart
		entry["window_seconds"] = windowSeconds
	}

	raw, _ := json.Marshal(entry)
	hash := sha256.Sum256(append([]byte(h.auditPrevHash), raw...))
	entry["hash"] = fmt.Sprintf("sha256:%x", hash[:])

	final, _ := json.Marshal(entry)

	if f, err := os.OpenFile(h.AuditPath, os.O_APPEND|os.O_WRONLY, 0600); err == nil {
		if _, err := f.Write(append(final, byte('\n'))); err != nil {
			fmt.Fprintf(os.Stderr, "[model-limits] audit write failed: %v\n", err)
		}
		_ = f.Close()
		h.auditPrevHash = entry["hash"].(string)
		h.auditEntryCount++
		go h.EmitAttestation("audit_append")
	}

	h.auditMu.Unlock()

	// Trigger anchor check
	go h.AnchorAuditIfNeeded()

	// Record surge
	if kind == "input" || kind == "output" || kind == "rate" || kind == "rate_extended" {
		go h.RecordSurge(modelID)
	}
}

// RecordSurge updates surge state
func (h *Handler) RecordSurge(modelID string) {
	now := time.Now()
	sec := now.Unix()
	h.surgeMu.Lock()
	defer h.surgeMu.Unlock()

	state := h.surgeState[modelID]
	if len(state) == 0 {
		state = make([]int, 60)
	}

	lastT := h.surgeLast[modelID]
	if !lastT.IsZero() {
		elapsed := sec - lastT.Unix()
		if elapsed > 0 {
			if elapsed >= 60 {
				for i := range state {
					state[i] = 0
				}
			} else {
				for i := int64(1); i <= elapsed; i++ {
					idx := int((lastT.Unix() + i) % 60)
					state[idx] = 0
				}
			}
		}
	}

	idx := int(sec % 60)
	state[idx]++
	h.surgeState[modelID] = state
	h.surgeLast[modelID] = now

	// Check trigger
	var total, counted int
	for _, v := range state {
		if v > 0 {
			total += v
			counted++
		}
	}
	var last10 int
	for k := int64(0); k < 10; k++ {
		idxK := int((sec - k) % 60)
		if idxK < 0 {
			idxK += 60
		}
		last10 += state[idxK]
	}

	avg := 0.0
	if counted > 0 {
		avg = float64(total) / float64(counted)
	}

	if last10 >= h.SurgeMinEvents && avg > 0 && float64(last10) > avg*h.SurgeFactor {
		if time.Since(h.surgeLastTrigger) > 15*time.Second {
			h.surgeLastTrigger = now
			if h.Metrics != nil {
				h.Metrics.IncModelLimitSurge()
			}
			go h.EmitAttestation("surge_trigger")
		}
	}
}

// AnchorAuditIfNeeded writes anchor entry
func (h *Handler) AnchorAuditIfNeeded() {
	if h.AnchorPath == "" || h.AnchorInterval <= 0 {
		return
	}

	h.auditMu.Lock()
	entries := h.auditEntryCount
	lastHash := h.auditPrevHash
	h.auditMu.Unlock()

	if entries == 0 || entries%h.AnchorInterval != 0 || lastHash == "" {
		return
	}

	h.anchorMu.Lock()
	defer h.anchorMu.Unlock()

	anchor := map[string]any{
		"ts":              time.Now().Unix(),
		"audit_last_hash": lastHash,
		"audit_entries":   entries,
		"prev_hash":       h.anchorPrevHash,
		"hash":            "",
	}

	raw, _ := json.Marshal(anchor)
	hash := sha256.Sum256(append([]byte(h.anchorPrevHash), raw...))
	anchor["hash"] = fmt.Sprintf("sha256:%x", hash[:])

	final, _ := json.Marshal(anchor)

	if f, err := os.OpenFile(h.AnchorPath, os.O_APPEND|os.O_WRONLY, 0600); err == nil {
		if _, err := f.Write(append(final, byte('\n'))); err != nil {
			fmt.Fprintf(os.Stderr, "[model-limits] anchor write failed: %v\n", err)
		}
		_ = f.Close()
		h.anchorPrevHash = anchor["hash"].(string)
		go h.EmitAttestation("anchor_commit")
	}
}

// SubscribeAttestation registers a new channel for attestation events
func (h *Handler) SubscribeAttestation() chan ModelLimitsAttestation {
	ch := make(chan ModelLimitsAttestation, 8)
	h.streamMu.Lock()
	h.streamSubs[ch] = struct{}{}
	h.streamMu.Unlock()
	return ch
}

// UnsubscribeAttestation removes subscription
func (h *Handler) UnsubscribeAttestation(ch chan ModelLimitsAttestation) {
	h.streamMu.Lock()
	if _, ok := h.streamSubs[ch]; ok {
		delete(h.streamSubs, ch)
		close(ch)
	}
	h.streamMu.Unlock()
}

// EmitAttestation broadcasts a fresh attestation to subscribers
func (h *Handler) EmitAttestation(reason string) {
	if os.Getenv("AGENTAUTH_ATTEST_STREAM_ENABLE") != "1" {
		return
	}
	att, err := h.BuildUnsignedAttestation()
	if err != nil {
		return
	}

	if att.Reason == "" {
		att.Reason = reason
	}

	att = h.MaybeAugmentAndSign(att)

	h.streamCountsMu.Lock()
	h.streamCounts[reason]++
	h.streamCountsMu.Unlock()

	h.streamMu.Lock()
	for ch := range h.streamSubs {
		select {
		case ch <- att:
		default:
		}
	}
	h.streamMu.Unlock()
}

// BuildUnsignedAttestation constructs the core attestation structure
func (h *Handler) BuildUnsignedAttestation() (ModelLimitsAttestation, error) {
	snapHash, _, _, _ := h.ComputeSnapshot()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	att := ModelLimitsAttestation{}
	att.Success = true
	att.Snapshot.Hash = snapHash
	att.Snapshot.GeneratedAt = now
	att.StrictUnknown = h.StrictUnknown

	if h.AuditPath == "" {
		att.Configured = false
		att.Reason = "audit_disabled"
		return att, nil
	}

	h.auditMu.Lock()
	att.Audit = &struct {
		HeadHash string `json:"head_hash"`
		Entries  int    `json:"entries"`
	}{HeadHash: h.auditPrevHash, Entries: h.auditEntryCount}
	h.auditMu.Unlock()

	h.anchorMu.Lock()
	if h.AnchorPath != "" && h.anchorPrevHash != "" {
		att.Anchor = &struct {
			LatestHash string `json:"latest_hash"`
			Entries    int    `json:"entries"`
			Interval   int    `json:"interval"`
		}{LatestHash: h.anchorPrevHash, Entries: 0, Interval: h.AnchorInterval}
	}
	h.anchorMu.Unlock()

	att.Configured = true
	return att, nil
}

// MaybeAugmentAndSign attaches surge stats, notarization receipt, and signature
func (h *Handler) MaybeAugmentAndSign(att ModelLimitsAttestation) ModelLimitsAttestation {
	if att.Configured {
		h.surgeMu.Lock()
		if time.Since(h.surgeLastTrigger) < 5*time.Second {
			att.Surge = &struct {
				ModelID   string  `json:"model_id"`
				Last10Sec int     `json:"last_10s_exceed_events"`
				AvgActive float64 `json:"avg_active_seconds"`
				Factor    float64 `json:"factor"`
				MinEvents int     `json:"min_events"`
				Triggered bool    `json:"triggered"`
				At        string  `json:"triggered_at,omitempty"`
			}{ModelID: "surge-active", Triggered: true, At: time.Now().UTC().Format(time.RFC3339Nano)}
		}
		h.surgeMu.Unlock()

		if os.Getenv("AGENTAUTH_MODEL_LIMIT_ATTEST_NOTARIZE") == "1" && h.Notarizer != nil && att.Snapshot.Hash != "" {
			auditHead := ""
			if att.Audit != nil {
				auditHead = att.Audit.HeadHash
			}
			anchorHead := ""
			if att.Anchor != nil {
				anchorHead = att.Anchor.LatestHash
			}
			seed := fmt.Sprintf("attest|%s|%s|%s", att.Snapshot.Hash, auditHead, anchorHead)
			hash := sha256.Sum256([]byte(seed))
			digest := fmt.Sprintf("sha256:%x", hash[:])
			if receipt, err := h.Notarizer.Notarize(digest); err == nil {
				att.Notarization = &struct {
					Provider       string  `json:"provider"`
					Timestamp      string  `json:"timestamp"`
					LatencySeconds float64 `json:"latency_seconds"`
					Success        bool    `json:"success"`
				}{Provider: receipt.Provider, Timestamp: receipt.Timestamp, LatencySeconds: receipt.LatencySeconds, Success: receipt.Success}
			}
		}
	}

	if os.Getenv("AGENTAUTH_MODEL_LIMIT_ATTEST_SIGN") == "1" && h.KeyManager != nil {
		if active := h.KeyManager.Active(); active != nil && len(active.Private) == ed25519.PrivateKeySize {
			if att.Nonce == "" {
				var nb [16]byte
				_, _ = rand.Read(nb[:])
				att.Nonce = base64.RawStdEncoding.EncodeToString(nb[:])
			}
			unsigned := att
			unsigned.Signature = ""
			raw, _ := json.Marshal(unsigned)
			sig := ed25519.Sign(active.Private, append([]byte("AGENTAUTH_MODEL_LIMIT_ATTEST:"), raw...))
			att.Signature = base64.RawStdEncoding.EncodeToString(sig)
			att.SigKid = active.ID
			att.SigMode = "eddsa"

			if prefix := os.Getenv("AGENTAUTH_ATTEST_DOMAIN_PREFIX"); prefix != "" {
				dsig := ed25519.Sign(active.Private, append([]byte(prefix), raw...))
				att.DomainSignature = base64.RawStdEncoding.EncodeToString(dsig)
				att.DomainPrefix = prefix
			}
		}
	}
	return att
}

// VerificationResult contains details of attestation verification
type VerificationResult struct {
	Valid        bool   `json:"valid"`
	Error        string `json:"error,omitempty"`
	Kid          string `json:"kid,omitempty"`
	SigMode      string `json:"sig_mode,omitempty"`
	CombinedHash string `json:"combined_hash,omitempty"`
}

// VerifyAttestation validates an incoming attestation signature and structure
func (h *Handler) VerifyAttestation(att ModelLimitsAttestation) VerificationResult {
	if att.Signature == "" || att.SigKid == "" || att.SigMode != "eddsa" {
		return VerificationResult{Valid: false, Error: "missing_signature_fields"}
	}

	if h.KeyManager == nil {
		return VerificationResult{Valid: false, Error: "no_key_registry"}
	}

	key := h.KeyManager.FindByID(att.SigKid)
	if key == nil {
		return VerificationResult{Valid: false, Error: "unknown_kid"}
	}

	// Reconstruct unsigned object with identical field order
	// We use an anonymous struct matching the input exactly to control serialization order
	type unsignedStruct struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
		Reason     string `json:"reason,omitempty"`
		Nonce      string `json:"nonce,omitempty"`
		Snapshot   struct {
			Hash        string `json:"hash"`
			GeneratedAt string `json:"generated_at"`
		} `json:"snapshot"`
		Audit *struct {
			HeadHash string `json:"head_hash"`
			Entries  int    `json:"entries"`
		} `json:"audit,omitempty"`
		Anchor *struct {
			LatestHash string `json:"latest_hash"`
			Entries    int    `json:"entries"`
			Interval   int    `json:"interval"`
		} `json:"anchor,omitempty"`
		StrictUnknown bool `json:"strict_unknown"`
		Surge         *struct {
			ModelID   string  `json:"model_id"`
			Last10Sec int     `json:"last_10s_exceed_events"`
			AvgActive float64 `json:"avg_active_seconds"`
			Factor    float64 `json:"factor"`
			MinEvents int     `json:"min_events"`
			Triggered bool    `json:"triggered"`
			At        string  `json:"triggered_at,omitempty"`
		} `json:"surge,omitempty"`
		Notarization *struct {
			Provider       string  `json:"provider"`
			Timestamp      string  `json:"timestamp"`
			LatencySeconds float64 `json:"latency_seconds"`
			Success        bool    `json:"success"`
		} `json:"notarization,omitempty"`
	}

	u := unsignedStruct{
		Success: att.Success, Configured: att.Configured, Reason: att.Reason, Nonce: att.Nonce,
		Snapshot: att.Snapshot, Audit: att.Audit, Anchor: att.Anchor,
		StrictUnknown: att.StrictUnknown, Surge: att.Surge, Notarization: att.Notarization,
	}

	raw, _ := json.Marshal(u)
	sigBytes, err := base64.RawStdEncoding.DecodeString(att.Signature)
	if err != nil {
		return VerificationResult{Valid: false, Error: "bad_signature_base64"}
	}

	prefixed := append([]byte("AGENTAUTH_MODEL_LIMIT_ATTEST:"), raw...)
	valid := ed25519.Verify(key.Public, prefixed, sigBytes)
	if !valid {
		return VerificationResult{Valid: false, Error: "signature_invalid", Kid: att.SigKid, SigMode: att.SigMode}
	}

	if att.DomainSignature != "" {
		if att.DomainPrefix == "" {
			return VerificationResult{Valid: false, Error: "domain_signature_prefix_missing"}
		}
		dsigBytes, err := base64.RawStdEncoding.DecodeString(att.DomainSignature)
		if err != nil {
			return VerificationResult{Valid: false, Error: "domain_signature_base64_invalid"}
		}
		prefixedDomain := append([]byte(att.DomainPrefix), raw...)
		if !ed25519.Verify(key.Public, prefixedDomain, dsigBytes) {
			return VerificationResult{Valid: false, Error: "domain_signature_invalid"}
		}
	}

	auditHead := ""
	if att.Audit != nil {
		auditHead = att.Audit.HeadHash
	}
	anchorHead := ""
	if att.Anchor != nil {
		anchorHead = att.Anchor.LatestHash
	}

	seed := fmt.Sprintf("attest|%s|%s|%s", att.Snapshot.Hash, auditHead, anchorHead)
	ch := sha256.Sum256([]byte(seed))
	combined := fmt.Sprintf("sha256:%x", ch[:])

	return VerificationResult{Valid: true, Kid: att.SigKid, SigMode: att.SigMode, CombinedHash: combined}
}

// VerifyAudit checks the integrity of the audit log file
func (h *Handler) VerifyAudit() (entries int, lastHash string, valid bool, err error) {
	if h.AuditPath == "" {
		return 0, "", false, fmt.Errorf("audit_disabled")
	}
	b, err := os.ReadFile(h.AuditPath)
	if err != nil {
		return 0, "", false, err
	}
	// Simple string split (assuming small/medium files as per prototype)
	// For production this should be a streaming reader.
	// But I can implement simple line splitting or just use bytes.Split
	// Let's assume I can add "strings" or use bytes.
	// Using "bytes" is safer if "strings" not imported.
	// But `json.Unmarshal` takes []byte.

	// Wait, I need to check imports. `handler.go` has `encoding/json` etc.
	// I'll add "bytes" and "strings" to imports if I rewrite header, OR use what I have.
	// I can use `bytes.Split`.

	// Actually, I can use `json.Decoder` but the format is JSONL.
	// I will use `bytes.Split`.
	return h.verifyChain(b, "hash")
}

// VerifyAnchor checks the integrity of the anchor log file
func (h *Handler) VerifyAnchor() (entries int, lastHash string, valid bool, err error) {
	if h.AnchorPath == "" {
		return 0, "", false, fmt.Errorf("anchor_disabled")
	}
	b, err := os.ReadFile(h.AnchorPath)
	if err != nil {
		return 0, "", false, err
	}
	return h.verifyChain(b, "hash")
}

func (h *Handler) verifyChain(content []byte, hashField string) (entries int, lastHash string, valid bool, err error) {
	// naive split
	parts := [][]byte{}
	start := 0
	for i, b := range content {
		if b == '\n' {
			if i > start {
				parts = append(parts, content[start:i])
			}
			start = i + 1
		}
	}
	if start < len(content) {
		parts = append(parts, content[start:])
	}

	valid = true
	prev := ""
	count := 0

	for _, line := range parts {
		line = trimSpace(line)
		if len(line) == 0 {
			continue
		}
		count++

		// Unmarshal to get fields
		var e struct {
			PrevHash string `json:"prev_hash"`
			Hash     string `json:"hash"`
		}
		if json.Unmarshal(line, &e) != nil {
			valid = false
			break
		}

		// Recompute
		var full map[string]any
		if json.Unmarshal(line, &full) != nil {
			valid = false
			break
		}

		actualHash, ok := full[hashField].(string)
		if !ok {
			valid = false
			break
		}

		full[hashField] = ""
		tmp, _ := json.Marshal(full)
		hh := sha256.Sum256(append([]byte(e.PrevHash), tmp...))
		recomputed := fmt.Sprintf("sha256:%x", hh[:])

		if recomputed != actualHash || e.PrevHash != prev {
			valid = false
			break
		}
		prev = actualHash
		lastHash = actualHash
	}
	return count, lastHash, valid, nil
}

func trimSpace(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t' || b[0] == '\r') {
		b = b[1:]
	}
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}
