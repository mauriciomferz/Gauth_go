package modellimits

// ModelLimitsAttestation models the attestation response for model limits governance.
// Structured as a deterministic JSON serialization target to enable stable Ed25519 signing.
// Optional signature fields (signature, sig_kid, sig_mode) are added only when
// GAUTH_MODEL_LIMIT_ATTEST_SIGN=1 and a GlobalEdDSARegistry active key exists.
// Exported for handler usage.
type ModelLimitsAttestation struct {
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
	Signature       string `json:"signature,omitempty"`
	SigKid          string `json:"sig_kid,omitempty"`
	SigMode         string `json:"sig_mode,omitempty"`
	DomainSignature string `json:"domain_signature,omitempty"`
	DomainPrefix    string `json:"domain_prefix,omitempty"`
}

// Internal structures for state tracking

// rateState tracks simple rate limit window
type rateState struct {
	WindowStart int64 // timestamp seconds
	Count       int
}

// rateStateExtended tracks extended window
type rateStateExtended struct {
	WindowStart int64
	Count       int
}

// LimitCheckResult encapsulates the decision from CheckLimit
type LimitCheckResult struct {
	Allowed       bool
	Error         string
	LimitEnforced bool
	Limit         int
	RateLimit     int
	WindowSeconds int
	Period        string
}
