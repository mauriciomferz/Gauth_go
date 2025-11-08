package rfc0111

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/observability"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/attest"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
	cr "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/crypto"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/crypto/keyring"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/delegation"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/ledger"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/pdp"
	poaPkg "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/token"
	"github.com/google/uuid"
	"github.com/o1egl/paseto"
)

// Typed context keys (phase 1 hardening). We retain legacy string lookup for backward compatibility.
type ctxKey int

const (
	ctxKeyRequestedAmount ctxKey = iota
	ctxKeySubject
)

// String forms retained for transitional compatibility (tests or external callers still using raw strings).
const (
	LegacyCtxRequestedAmount = "requested_amount"
	LegacyCtxSubject         = "subject"
)

// WithRequestedAmount attaches a requested amount string to context using typed key (and legacy for transitional consumers).
func WithRequestedAmount(ctx context.Context, amt string) context.Context {
	// Store under typed key only; legacy path read fallback retained until migration complete.
	return context.WithValue(ctx, ctxKeyRequestedAmount, amt)
}

// WithSubject attaches a subject identity to context using typed key.
func WithSubject(ctx context.Context, subject string) context.Context {
	return context.WithValue(ctx, ctxKeySubject, subject)
}

// Precompiled UUID v4 regex: 8-4-4-4-12 hex, version 4, variant 8-9-a-b.
var uuidV4Rx = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

func isUUIDv4(s string) bool { return uuidV4Rx.MatchString(s) }

// POAStatus represents lifecycle states for a PowerOfAttorney.
type POAStatus string

const (
	POAStatusActive  POAStatus = "active"
	POAStatusRevoked POAStatus = "revoked"
	POAStatusExpired POAStatus = "expired"
	// POAStatusDraft represents a PoA that is prepared but not yet finalized (e.g., awaiting multi-signature quorum)
	POAStatusDraft POAStatus = "draft"
	// POAStatusSuspended is a temporary hold (can transition back to active or to terminated)
	POAStatusSuspended POAStatus = "suspended"
	// POAStatusTerminated is a permanent closure distinct from revocation (e.g., natural end-of-contract);
	// terminated PoAs cannot return to active.
	POAStatusTerminated POAStatus = "terminated"
)

// PowerOfAttorney represents a delegation grant between a grantor and grantee.
type PowerOfAttorney struct {
	ID           string            `json:"id"`
	Version      int               `json:"version"` // structural version (included in canonical digest)
	Grantor      string            `json:"grantor"`
	Grantee      string            `json:"grantee"`
	Scope        []string          `json:"scope"`
	Restrictions map[string]string `json:"restrictions,omitempty"`
	// Taxonomy expansion (RB2): classify agents and actions for governance analytics & scoped delegation.
	// Included in canonical digest starting with Version >=3. Empty values treated as absent (legacy compatibility).
	AgentType   string    `json:"agent_type,omitempty"`
	Sector      string    `json:"sector,omitempty"`
	ActionClass string    `json:"action_class,omitempty"`
	ValidFrom   time.Time `json:"valid_from"`
	ValidUntil  time.Time `json:"valid_until"`
	Status      POAStatus `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Legal & evidentiary extension fields (Beta MVP – excluded from canonical digest):
	Jurisdiction     string        `json:"jurisdiction,omitempty"`      // Primary governing jurisdiction
	Witnesses        []string      `json:"witnesses,omitempty"`         // Optional witness identities
	Attestations     []string      `json:"attestations,omitempty"`      // External attestations / certifications IDs
	RevokedAt        *time.Time    `json:"revoked_at,omitempty"`        // Timestamp of revocation (if revoked)
	RevocationReason string        `json:"revocation_reason,omitempty"` // Human-readable structured reason code
	Signature        *POASignature `json:"signature,omitempty"`
	// Multi-signature prototype fields (RFC115-C8): if Signers provided and Threshold>1 we require aggregated validation.
	Signers   []string       `json:"signers,omitempty"`
	Threshold int            `json:"threshold,omitempty"`
	Weights   map[string]int `json:"weights,omitempty"` // signer -> weight (positive int); included in canonical digest when present
	// MultiSignatures holds individual signatures (each over canonical POA digest) keyed by signer identity.
	// For threshold verification we require at least Threshold valid entries.
	MultiSignatures map[string]*POASignature `json:"multi_signatures,omitempty"`
	// SatisfiedWeight records cumulative weight verified (only set for weighted multi-sig mode on success).
	SatisfiedWeight int `json:"satisfied_weight,omitempty"`
	// SatisfiedSignatures records count of valid signatures contributing to threshold (set on success).
	SatisfiedSignatures int `json:"satisfied_signatures,omitempty"`
	// Hierarchical delegation (sub-delegation) fields (excluded from canonical digest for v1-v3; future version may incorporate):
	ParentPOAID string `json:"parent_poa_id,omitempty"`
	// ParentDigest binds this delegation to its parent's canonical digest (hierarchical integrity). Included in canonical digest for Version>=4.
	ParentDigest string `json:"parent_digest,omitempty"`
	Depth        int    `json:"depth,omitempty"` // 0=root; derived when ParentPOAID set
	// Dual-control revocation governance fields (beta design placeholders)
	Controllers       []string                `json:"controllers,omitempty"`        // Optional explicit controller identities authorized for quorum revocation
	PendingRevocation *PendingRevocationState `json:"pending_revocation,omitempty"` // Non-nil when a revocation workflow is in progress
	// Evidence hash attachments for forensic strengthening (excluded from canonical digest)
	EvidenceHashes []string `json:"evidence_hashes,omitempty"`
}

// POASignature provides authenticity metadata; signature covers canonical digest.
type POASignature struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	DigestHex string `json:"dig"`
	SigBase64 string `json:"sig"`
	Canonical []byte `json:"-"` // cached canonical form (not serialized)
}

// PendingRevocationState captures an in-progress quorum / dual-control revocation workflow.
// Quorum rules (design): either RequiredCount (distinct approvers) OR RequiredWeight (sum of signer/controller weights) must be met.
// Only controllers (when set), Grantor, or declared Signers may approve. Approvals map records timestamp of each distinct approver.
// Finalization transitions POA.Status to revoked and sets RevokedAt/RevocationReason. Cancellation returns to prior status.
// All fields are persisted inside the PoA record for durability.
type PendingRevocationState struct {
	InitiatedAt     time.Time            `json:"initiated_at"`
	Initiator       string               `json:"initiator"`
	Reason          string               `json:"reason,omitempty"`
	EvidenceHashes  []string             `json:"evidence_hashes,omitempty"`  // optional supporting evidence content-addressed hashes
	Approvals       map[string]time.Time `json:"approvals,omitempty"`        // approver -> timestamp
	RequiredCount   int                  `json:"required_count,omitempty"`   // quorum size by distinct approvers (if >0)
	RequiredWeight  int                  `json:"required_weight,omitempty"`  // cumulative weight target (alternative mode)
	SatisfiedWeight int                  `json:"satisfied_weight,omitempty"` // running total in weight mode
	Finalized       bool                 `json:"finalized"`
	Canceled        bool                 `json:"canceled"`
}

// RevocationRequest represents an initiation request for dual-control revocation.
type RevocationRequest struct {
	POAID          string   `json:"poa_id"`
	Initiator      string   `json:"initiator"`
	Reason         string   `json:"reason,omitempty"`
	EvidenceHashes []string `json:"evidence_hashes,omitempty"`
}

// ========================================
// RFC 0111 & 0115 Conformance Types
// ========================================

// TokenResult represents the result of token verification or issuance operations (RFC 0111:1).
type TokenResult struct {
	Token     string                   `json:"token"`
	ExpiresAt time.Time                `json:"expires_at"`
	Status    string                   `json:"status"`
	Result    *TokenVerificationResult `json:"verification_result,omitempty"`
	Error     string                   `json:"error,omitempty"`
}

// ScopeItem represents a single scope entry with semantic validation (RFC 0115:2).
type ScopeItem struct {
	Action      string            `json:"action"`
	Resource    string            `json:"resource,omitempty"`
	Constraints map[string]string `json:"constraints,omitempty"`
}

// ScopeValidator provides scope semantic validation (RFC 0115:2).
type ScopeValidator struct {
	AllowedActions   []string          `json:"allowed_actions,omitempty"`
	RequiredFields   []string          `json:"required_fields,omitempty"`
	ConstraintRules  map[string]string `json:"constraint_rules,omitempty"`
	StrictValidation bool              `json:"strict_validation"`
}

// ValidateScope performs semantic validation of scope items (RFC 0115:2).
func ValidateScope(items []ScopeItem, validator *ScopeValidator) error {
	if validator == nil || !validator.StrictValidation {
		return nil // permissive mode
	}
	for _, item := range items {
		if item.Action == "" {
			return fmt.Errorf("scope item missing action")
		}
		// Check against allowed actions if configured
		if len(validator.AllowedActions) > 0 {
			found := false
			for _, allowed := range validator.AllowedActions {
				if item.Action == allowed {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("action %s not in allowed list", item.Action)
			}
		}
	}
	return nil
}

// FormalValidation represents formal requirement validation state (RFC 0115:4).
type FormalValidation struct {
	RequirementsMet bool      `json:"requirements_met"`
	ChecksPassed    []string  `json:"checks_passed,omitempty"`
	ChecksFailed    []string  `json:"checks_failed,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

// RequirementCheck represents a single formal requirement check (RFC 0115:4).
type RequirementCheck struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Passed      bool      `json:"passed"`
	Message     string    `json:"message,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
}

// DailyLimit represents daily transaction limits (RFC 0115:5).
type DailyLimit struct {
	MaxAmount       float64   `json:"max_amount"`
	Currency        string    `json:"currency,omitempty"`
	TransactionCap  int       `json:"transaction_cap,omitempty"`
	CurrentAmount   float64   `json:"current_amount"`
	CurrentCount    int       `json:"current_count"`
	ResetAt         time.Time `json:"reset_at"`
	LastTransaction time.Time `json:"last_transaction,omitempty"`
}

// PowerLimit represents general power-of-attorney limits (RFC 0115:5).
type PowerLimit struct {
	Type         string            `json:"type"` // e.g., "daily", "transaction", "cumulative"
	MaxValue     float64           `json:"max_value"`
	CurrentValue float64           `json:"current_value"`
	Unit         string            `json:"unit,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// TransactionLimit represents per-transaction limits (RFC 0115:5).
type TransactionLimit struct {
	MaxAmount       float64 `json:"max_amount"`
	MinAmount       float64 `json:"min_amount,omitempty"`
	Currency        string  `json:"currency,omitempty"`
	RequireApproval bool    `json:"require_approval,omitempty"`
}

// Rights represents rights granted under power of attorney (RFC 0115:6).
type Rights struct {
	Actions               []string          `json:"actions"`
	Resources             []string          `json:"resources,omitempty"`
	Constraints           map[string]string `json:"constraints,omitempty"`
	ValidFrom             time.Time         `json:"valid_from,omitempty"`
	ValidUntil            time.Time         `json:"valid_until,omitempty"`
	TransferableSubRights bool              `json:"transferable_sub_rights,omitempty"`
}

// Obligations represents obligations under power of attorney (RFC 0115:6).
type Obligations struct {
	DutiesOfCare      []string          `json:"duties_of_care,omitempty"`
	ReportingRequired bool              `json:"reporting_required,omitempty"`
	NotificationRules map[string]string `json:"notification_rules,omitempty"`
	ComplianceChecks  []string          `json:"compliance_checks,omitempty"`
}

// DutyOfCare represents a specific duty of care obligation (RFC 0115:6).
type DutyOfCare struct {
	Description string            `json:"description"`
	Standard    string            `json:"standard,omitempty"` // e.g., "reasonable_person", "professional"
	Scope       []string          `json:"scope,omitempty"`
	Evidence    map[string]string `json:"evidence,omitempty"`
}

// ConditionalExpression represents a conditional special condition (RFC 0115:7).
type ConditionalExpression struct {
	Condition  string            `json:"condition"`
	Expression string            `json:"expression"`
	Variables  map[string]string `json:"variables,omitempty"`
	Evaluated  bool              `json:"evaluated,omitempty"`
	Result     interface{}       `json:"result,omitempty"`
}

// RuntimeEvaluation represents runtime evaluation of special conditions (RFC 0115:7).
type RuntimeEvaluation struct {
	ExpressionID string                  `json:"expression_id"`
	Context      map[string]interface{}  `json:"context,omitempty"`
	Result       bool                    `json:"result"`
	Error        string                  `json:"error,omitempty"`
	EvaluatedAt  time.Time               `json:"evaluated_at"`
	Conditions   []ConditionalExpression `json:"conditions,omitempty"`
}

// ThresholdValidation represents joint signature threshold validation (RFC 0115:8).
type ThresholdValidation struct {
	RequiredSignatures int             `json:"required_signatures"`
	ProvidedSignatures int             `json:"provided_signatures"`
	ValidSignatures    int             `json:"valid_signatures"`
	Threshold          int             `json:"threshold"`
	ThresholdMet       bool            `json:"threshold_met"`
	SignerIdentities   []string        `json:"signer_identities,omitempty"`
	ValidationResults  map[string]bool `json:"validation_results,omitempty"`
	ValidatedAt        time.Time       `json:"validated_at"`
}

// Reusable constants (reduce duplication and goconst warnings)
const (
	poaVersionV1 = "poa/v1"
	algEd25519   = "ed25519"
	// Digest mismatch classification reasons (used in envelope & signature verification metrics)
	reasonDomainConflict  = "domain_conflict"
	reasonTamperSuspected = "tamper_suspected"
	reasonOther           = "other"
	// Canonicalization digest mismatch sub-reason (standardized literal)
	reasonCanonicalizationError = "canonicalization_error"
	// Generic semantic mismatch keys (deduplicated for metrics snapshot maps)
	counterCurrencyMismatch    = "currency_mismatch"
	counterRestrictionMismatch = "restriction_mismatch"
	// Error message fragments (avoid repeating string literals for goconst hygiene)
	errDigestMismatch         = "digest mismatch"
	errPOADigestMismatch      = "poa digest mismatch"
	errCurrencyMismatchFmt    = "currency mismatch expected %s got %s"
	errRestrictionMismatchFmt = "restriction %s mismatch expected %s got %s"
)

// ValidateMultiSignature performs minimal structural checks for multi-signature PoA prototype.
// Rules:
// - If Threshold == 0 => treat as single signature mode (no error)
// - If Threshold > 0 and len(Signers) < Threshold => error
// - Signer identities must be non-empty unique strings
func ValidateMultiSignature(p *PowerOfAttorney) error {
	if p == nil {
		return fmt.Errorf("nil poa")
	}
	if p.Threshold <= 1 {
		return nil
	}
	if len(p.Signers) < p.Threshold {
		return fmt.Errorf("insufficient signers: have %d need %d", len(p.Signers), p.Threshold)
	}
	seen := make(map[string]struct{}, len(p.Signers))
	for _, s := range p.Signers {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("empty signer identity")
		}
		if _, ok := seen[s]; ok {
			return fmt.Errorf("duplicate signer: %s", s)
		}
		seen[s] = struct{}{}
	}
	// If multi-signature map present ensure there are no unknown signers recorded (defensive)
	if p.MultiSignatures != nil {
		for signer := range p.MultiSignatures {
			if _, ok := seen[signer]; !ok {
				return fmt.Errorf("multi_signatures contains unknown signer: %s", signer)
			}
		}
	}
	// Weights validation (optional weighted mode)
	if len(p.Weights) > 0 {
		var total int
		for k, v := range p.Weights {
			if _, ok := seen[k]; !ok {
				return fmt.Errorf("weight key not a declared signer: %s", k)
			}
			if v <= 0 || v > 1_000_000 {
				return fmt.Errorf("invalid weight for signer %s: %d", k, v)
			}
			total += v
		}
		if total < p.Threshold { // interpret threshold as required cumulative weight when weights provided
			return fmt.Errorf("cumulative weights %d below threshold %d", total, p.Threshold)
		}
	}
	return nil
}

// verifyMultiSignatures enforces threshold Ed25519 verification semantics for a POA.
// Requirements:
// - If Threshold <= 1 => skip (single signature path handled elsewhere)
// - Structural validation must already have passed.
// - Canonical digest computed once; each signature must match digest and verify.
// - At least Threshold signatures must be present AND valid; invalid signatures are counted separately.
// Failure modes (returned as RFC integrity_failure errors):
//   - insufficient_threshold (not enough valid signatures)
//   - canonical_digest_failed
//   - invalid_signature_<signer>
//   - missing_signature_<signer> (optional; only enforced if mandatorySignatures enabled in service)
//
//nolint:gocyclo // Multi-signature verification with threshold logic
func (s *Service) verifyMultiSignatures(p *PowerOfAttorney) error {
	if p == nil {
		return nil
	}
	if p.Threshold <= 1 {
		return nil
	}
	start := time.Now()
	if err := ValidateMultiSignature(p); err != nil {
		return rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("multi-signature structural invalid: %v", err))
	}
	// Weighted threshold mode: prefer embedded POA weights; fallback to env (GAUTH_MULTI_SIG_WEIGHTS) for transitional compatibility.
	weights := p.Weights
	useWeights := false
	if len(weights) == 0 {
		if raw := os.Getenv("GAUTH_MULTI_SIG_WEIGHTS"); raw != "" {
			parts := strings.Split(raw, ",")
			parsed := map[string]int{}
			valid := true
			var total int
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				kv := strings.SplitN(part, "=", 2)
				if len(kv) != 2 {
					valid = false
					break
				}
				w64, err := strconv.ParseInt(strings.TrimSpace(kv[1]), 10, 32)
				if err != nil {
					valid = false
					break
				}
				w := int(w64)
				if w <= 0 || w > 1_000_000 {
					valid = false
					break
				}
				parsed[strings.TrimSpace(kv[0])] = w
				total += w
			}
			if valid && total >= p.Threshold {
				weights = parsed
			}
		}
	}
	if len(weights) > 0 {
		useWeights = true
	}
	// Count-based structural guard only applies when weights not enabled. In weighted mode we allow fewer signatures
	// as long as cumulative weight meets threshold later.
	if !useWeights && len(p.MultiSignatures) < p.Threshold {
		if s.metrics != nil {
			s.metrics.IncMultiSignatureStructuralFailures()
			s.metrics.IncMultiSignatureVerificationFailures()
		}
		return rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("insufficient_threshold: have %d signatures need %d", len(p.MultiSignatures), p.Threshold))
	}
	digestHex, canon, derr := CanonicalPOADigest(p)
	if derr != nil {
		if s.metrics != nil {
			s.metrics.IncMultiSignatureDigestFailures()
			s.metrics.IncMultiSignatureVerificationFailures()
		}
		return rfc.New(rfc.ErrIntegrityFailure, "canonical_digest_failed")
	}
	validCount := 0       // simple count (fallback)
	validWeightTotal := 0 // accumulated weights when useWeights
	var firstErr error
	for _, signer := range p.Signers {
		sig := p.MultiSignatures[signer]
		if sig == nil {
			// enforce presence only if service mandates signatures (opt-in strictness)
			if s != nil && s.mandatorySignatures {
				return rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("missing_signature_%s", signer))
			}
			continue
		}
		if sig.DigestHex != digestHex {
			if firstErr == nil {
				firstErr = rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("invalid_digest_%s", signer))
			}
			if s.metrics != nil {
				s.metrics.IncMultiSignatureInvalidSignatureFailures()
				s.metrics.IncMultiSignatureVerificationFailures()
			}
			continue
		}
		if sig.Algorithm != algEd25519 {
			if firstErr == nil {
				firstErr = rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("invalid_algorithm_%s", signer))
			}
			if s.metrics != nil {
				s.metrics.IncMultiSignatureInvalidSignatureFailures()
				s.metrics.IncMultiSignatureVerificationFailures()
			}
			continue
		}
		// locate public key via keyProvider or keyRing using sig.KeyID
		var pub []byte
		if s != nil && s.keyProvider != nil && sig.KeyID != "" {
			if pkBytes, algo, err := s.keyProvider.PublicKey(sig.KeyID); err == nil && algo == cr.AlgoEd25519 && len(pkBytes) == ed25519.PublicKeySize {
				pub = pkBytes
			}
		}
		if len(pub) == 0 && s != nil && s.keyRing != nil && sig.KeyID != "" {
			if ak := s.keyRing.Active(); ak != nil && ak.ID == sig.KeyID {
				pub = ak.Material
			}
			if len(pub) == 0 {
				for _, pk := range s.keyRing.Previous() {
					if pk != nil && pk.ID == sig.KeyID {
						pub = pk.Material
						break
					}
				}
			}
		}
		if len(pub) != ed25519.PublicKeySize {
			if s.metrics != nil {
				s.metrics.IncMultiSignaturePublicKeyMissing()
				s.metrics.IncMultiSignatureVerificationFailures()
			}
			if s != nil && s.strictAuthenticity {
				return rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("public_key_missing_%s", signer))
			}
			if firstErr == nil {
				firstErr = rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("public_key_missing_%s", signer))
			}
			continue
		}
		sigBytes, decErr := base64.StdEncoding.DecodeString(sig.SigBase64)
		if decErr != nil || len(sigBytes) != ed25519.SignatureSize || !ed25519.Verify(ed25519.PublicKey(pub), canon, sigBytes) {
			if s.metrics != nil {
				s.metrics.IncMultiSignatureInvalidSignatureFailures()
				s.metrics.IncMultiSignatureVerificationFailures()
			}
			if firstErr == nil {
				firstErr = rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("invalid_signature_%s", signer))
			}
			continue
		}
		validCount++
		if useWeights {
			if w, ok := weights[signer]; ok {
				validWeightTotal += w
			}
		}
	}
	if useWeights {
		if validWeightTotal < p.Threshold { // interpret Threshold as required cumulative weight
			if s.metrics != nil {
				s.metrics.IncMultiSignatureWeightFailures()
				s.metrics.IncMultiSignatureVerificationFailures()
			}
			// surface primary signature error if any else weight failure
			if firstErr != nil {
				return firstErr
			}
			return rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("insufficient_weight_valid: have %d need %d", validWeightTotal, p.Threshold))
		}
	} else {
		if validCount < p.Threshold {
			if s.metrics != nil {
				s.metrics.IncMultiSignatureThresholdFailures()
				s.metrics.IncMultiSignatureVerificationFailures()
			}
			if firstErr != nil {
				return firstErr
			}
			return rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("insufficient_threshold_valid: have %d need %d", validCount, p.Threshold))
		}
	}
	// Success: record satisfied metrics
	p.SatisfiedSignatures = validCount
	if useWeights {
		p.SatisfiedWeight = validWeightTotal
	}
	if s.metrics != nil {
		s.metrics.IncMultiSignatureVerifications()
		s.metrics.ObserveMultiSignatureVerificationLatency(time.Since(start))
	}
	return nil
}

// DelegationRequest captures inputs to create a delegation.
type DelegationRequest struct {
	Grantor      string            `json:"grantor"`
	Grantee      string            `json:"grantee"`
	Scope        []string          `json:"scope"`
	Restrictions map[string]string `json:"restrictions,omitempty"`
	Duration     time.Duration     `json:"duration"`
	Signers      []string          `json:"signers,omitempty"`
	Threshold    int               `json:"threshold,omitempty"`
	Weights      map[string]int    `json:"weights,omitempty"`
	ParentPOAID  string            `json:"parent_poa_id,omitempty"`
	// Optional taxonomy fields (RB2). When any are provided the resulting POA Version is auto-bumped to >=3
	// and canonical digest includes taxonomy object (unless multi-sig domain path).
	AgentType   string `json:"agent_type,omitempty"`
	Sector      string `json:"sector,omitempty"`
	ActionClass string `json:"action_class,omitempty"`
	// Optional claims for jurisdiction enforcement context (P1.3). Used to provide additional
	// enforcement context beyond structured fields (e.g., gdpr_consent, ccpa_opt_out, data_type).
	Claims map[string]interface{} `json:"claims,omitempty"`
}

// generateLocalKey returns a 32-byte random key for PASETO local tokens.
// Order of precedence:
// 1. GAUTH_TOKEN_SYM_KEY env var (base64, >=32 bytes)
// 2. crypto/rand 32 bytes
// 3. uuid fallback (NOT cryptographically strong; diagnostics only)
func generateLocalKey() []byte {
	if env := os.Getenv("GAUTH_TOKEN_SYM_KEY"); env != "" {
		decoded, err := base64.StdEncoding.DecodeString(env)
		if err == nil && len(decoded) >= 32 {
			return decoded[:32]
		}
	}
	out := make([]byte, 32)
	if _, err := rand.Read(out); err != nil {
		uid := uuid.New()
		copy(out, uid[:])
	}
	return out
}

// Option configures a Service during construction.
type Option func(*Service)

// WithMetrics injects a metrics implementation (defaults to metrics.Noop if not supplied).
func WithMetrics(m metrics.Metrics) Option {
	return func(s *Service) {
		if m != nil {
			s.metrics = m
		}
	}
}

// WithCollectorRegistry injects a CollectorRegistry as the metrics implementation.
// This allows using multiple simultaneous metrics collectors (Prometheus + StatsD + JSON, etc.).
// The registry dispatches metric events to all registered collectors in parallel.
//
// Example:
//
//	registry := metrics.NewCollectorRegistry(true) // concurrent dispatch
//	registry.Register(collectors.NewPrometheusCollector("prom", promMetrics, "Prometheus exporter"))
//	registry.Register(collectors.NewJSONCollector("json", "/tmp/metrics.json", false))
//	svc := rfc0111.New(store, rfc0111.WithCollectorRegistry(registry))
func WithCollectorRegistry(registry *metrics.CollectorRegistry) Option {
	return func(s *Service) {
		if registry != nil {
			s.metrics = registry
		}
	}
}

// WithSignerProvider injects a signer provider function returning an active crypto.Signer.
func WithSignerProvider(fn func() (cr.Signer, error)) Option {
	return func(s *Service) {
		if fn != nil {
			s.signerProvider = fn
		}
	}
}

// WithInMemoryAlgorithm installs an in-memory key provider + signer for a specific algorithm.
// Supported values: ed25519, ecdsa-p256. It sets both signerProvider and keyProvider so issuance
// and verification succeed without additional configuration. Unknown algorithms are ignored.
func WithInMemoryAlgorithm(algo string) Option {
	return func(s *Service) {
		switch algo {
		case cr.AlgoEd25519:
			if kp, err := cr.NewInMemoryEd25519Provider(); err == nil {
				s.signerProvider = kp.ActiveSigner
				s.keyProvider = kp
			}
		case cr.AlgoECDSAP256:
			if kp, err := cr.NewInMemoryECDSAProvider(); err == nil {
				s.signerProvider = kp.ActiveSigner
				s.keyProvider = kp
			}
		}
	}
}

// WithKMS installs a KMS abstraction; if provided it supersedes signerProvider in issuance paths.
func WithKMS(kms cr.KMS) Option {
	return func(s *Service) {
		if kms != nil {
			s.signerProvider = kms.ActiveSigner
			// Provide a thin adapter for KeyProvider if kms lacks VerifyWith.
			if _, ok := interface{}(kms).(cr.KeyProvider); ok {
				s.keyProvider = kms.(cr.KeyProvider)
			} else {
				// Wrap using anonymous struct implementing VerifyWith.
				s.keyProvider = &kmsKeyProviderAdapter{kms: kms}
			}
		}
	}
}

// WithAttestationTrustAnchors installs a trust anchor registry used for optional
// strict attestation issuer enforcement (GAUTH_ATTEST_REQUIRE_TRUST_ANCHOR=1).
// If nil is passed it is ignored. The registry can be mutated after service
// construction to add anchors dynamically in tests or initialization code.
func WithAttestationTrustAnchors(reg *attest.TrustAnchorRegistry) Option {
	return func(s *Service) {
		if reg != nil {
			s.attestAnchors = reg
		}
	}
}

// kmsKeyProviderAdapter adapts a KMS to KeyProvider for verification.
type kmsKeyProviderAdapter struct{ kms cr.KMS }

func (a *kmsKeyProviderAdapter) ActiveSigner() (cr.Signer, error) { return a.kms.ActiveSigner() }
func (a *kmsKeyProviderAdapter) PublicKey(id string) ([]byte, string, error) {
	return a.kms.PublicKey(id)
}
func (a *kmsKeyProviderAdapter) VerifyWith(msg, sig []byte, keyID string) error {
	pk, algo, err := a.kms.PublicKey(keyID)
	if err != nil {
		return err
	}
	if algo != cr.AlgoEd25519 {
		return fmt.Errorf("unsupported algo %s", algo)
	}
	if len(sig) != ed25519.SignatureSize {
		return errors.New("invalid signature length")
	}
	if !ed25519.Verify(ed25519.PublicKey(pk), msg, sig) {
		return errors.New("invalid signature")
	}
	return nil
}

// WithKeyProvider supplies an asymmetric key provider for signature verification.
func WithKeyProvider(kp cr.KeyProvider) Option {
	return func(s *Service) {
		if kp != nil {
			s.keyProvider = kp
		}
	}
}

// WithStrictAuthenticity enables strict authenticity mode (missing signature public key causes integrity failure).
func WithStrictAuthenticity() Option { return func(s *Service) { s.strictAuthenticity = true } }

// WithSemanticValidator installs a semantic PoA validator.
func WithSemanticValidator(v PoAValidator) Option {
	return func(s *Service) {
		if v != nil {
			s.poaValidator = v
		}
	}
}

// WithEnhancedValidator installs the enhanced PoA validator with warning collection and daily limits.
func WithEnhancedValidator(v *EnhancedPoAValidator) Option {
	return func(s *Service) {
		if v != nil {
			s.enhancedValidator = v
		}
	}
}

// WithMandatorySignatures enforces that every issuance must be successfully signed; failures abort CreateDelegation.
func WithMandatorySignatures() Option { return func(s *Service) { s.mandatorySignatures = true } }

// WithReplayFailClosed enforces fail-closed semantics on distributed replay protection.
// When enabled, any error from the ReplayStore (Seen or Record) aborts verification instead
// of being treated as a cache miss. This increases security (no silent replay bypass on store
// outage) but can reduce availability if the store is unstable.
func WithReplayFailClosed() Option { return func(s *Service) { s.failClosedReplay = true } }

// WithPDP injects a PDP engine; if present it overrides legacy Authorizer for delegation authorization decisions.
func WithPDP(engine pdp.Engine) Option {
	return func(s *Service) {
		if engine != nil {
			s.pdp = engine
		}
	}
}

// WithLedger installs an immutable audit ledger backend (append-only, hash-chained entries).
// Issuance and revocation events will be appended best-effort; failures are non-fatal.
func WithLedger(l ledger.Store) Option {
	return func(s *Service) {
		if l != nil {
			s.ledger = l
		}
	}
}

// NewService creates a new RFC 0111 service (in-memory prototype) with optional functional options.
func NewService(auditLogger *audit.MemoryLogger, authorizer authz.Authorizer, opts ...Option) *Service {
	s := &Service{
		repo:         newMemoryRepository(),
		audit:        auditLogger,
		authz:        authorizer,
		nowFn:        time.Now,
		revChain:     delegation.NewRevocationChain(),
		issChain:     NewDelegationChain(),
		tokenKey:     generateLocalKey(),
		keyRing:      keyring.NewKeyRing(),
		metrics:      metrics.Noop,
		limits:       ValidationLimits{},
		poaValidator: selectPoAValidator(),
		dailyAmounts: make(map[string]float64),
	}
	// Optional persistent repository activation via env path
	if path := os.Getenv("GAUTH_PERSIST_PATH"); path != "" {
		if br, err := NewBoltRepository(path); err == nil {
			s.repo = br
		} else {
			fmt.Fprintf(os.Stderr, "warn: bolt repo init failed (%v) falling back to memory\n", err)
		}
	}
	// semanticCounters: zero-values (prototype) will accumulate semantic rejection reasons in future validation path.
	s.limits.applyDefaults()
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	if os.Getenv("GAUTH_MULTI_SIG_STRICT") == "1" {
		s.mandatorySignatures = true
		s.strictAuthenticity = true
	}
	// Default strict authenticity unless explicitly disabled (GAUTH_STRICT_AUTHENTICITY=0)
	if os.Getenv("GAUTH_STRICT_AUTHENTICITY") != "0" && !s.strictAuthenticity {
		s.strictAuthenticity = true
	}
	return s
}

// NewServicePersistent creates a service using a persistent file-backed audit logger.
// path example: data/audit/poa.log
func NewServicePersistent(auditLogPath string, authorizer authz.Authorizer, opts ...Option) (*Service, error) {
	fl, err := audit.OpenFileLogger(auditLogPath)
	if err != nil {
		return nil, err
	}
	s := &Service{
		repo:         newMemoryRepository(), // default; can be overridden via WithPOARepository option (e.g. Bolt)
		audit:        fl,
		authz:        authorizer,
		nowFn:        time.Now,
		revChain:     delegation.NewRevocationChain(),
		issChain:     NewDelegationChain(),
		tokenKey:     generateLocalKey(),
		keyRing:      keyring.NewKeyRing(),
		metrics:      metrics.Noop,
		limits:       ValidationLimits{},
		poaValidator: selectPoAValidator(),
		dailyAmounts: make(map[string]float64),
	}
	if path := os.Getenv("GAUTH_PERSIST_PATH"); path != "" {
		if br, err := NewBoltRepository(path); err == nil {
			s.repo = br
		} else {
			fmt.Fprintf(os.Stderr, "warn: bolt repo init failed (%v) falling back to memory\n", err)
		}
	}
	// semanticCounters: zero-values (prototype) will accumulate semantic rejection reasons in future validation path.
	s.limits.applyDefaults()
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	if os.Getenv("GAUTH_STRICT_AUTHENTICITY") != "0" && !s.strictAuthenticity {
		s.strictAuthenticity = true
	}
	return s, nil
}

type GAuth10Framework struct {
	AuthServer string
	Clients    []string
}

// ToJSON serializes the framework to JSON
func (f *GAuth10Framework) ToJSON() ([]byte, error) {
	return json.MarshalIndent(f, "", "  ")
}

// GetStatus returns the current status of the framework
func (f *GAuth10Framework) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"auth_server":    f.AuthServer,
		"clients_count":  len(f.Clients),
		"version":        "1.0.0",
		"rfc":            "GiFo-RFC-0111",
		"implementation": "Go Reference Implementation",
	}
}

// DelegationResponse represents the response to a delegation request
type DelegationResponse struct {
	POA       PowerOfAttorney     `json:"power_of_attorney"`
	AuthToken string              `json:"auth_token"`
	ExpiresAt time.Time           `json:"expires_at"`
	Warnings  []ValidationWarning `json:"warnings,omitempty"` // validation warnings from enhanced validator
}

// ValidationContext carries structured inputs for delegation validation beyond simple action string.
// Future expansion can include currency, location, device attributes, risk score, etc.
type ValidationContext struct {
	Action          string            // action being performed (required)
	RequestedAmount *float64          // optional numeric amount for restriction checks
	Metadata        map[string]string // auxiliary attributes (for future policy engines / audit)
}

// TokenVerificationResult carries structured validation outcome for an auth token.
type TokenVerificationResult struct {
	DelegationID    string            `json:"delegation_id"`
	Grantor         string            `json:"grantor"`
	Grantee         string            `json:"grantee"`
	Scope           []string          `json:"scope"`
	Restrictions    map[string]string `json:"restrictions,omitempty"`
	ExpiresAt       time.Time         `json:"expires_at"`
	Status          string            `json:"status"`
	IssuanceChain   string            `json:"issuance_chain_tip,omitempty"`
	RevocationChain string            `json:"revocation_chain_tip,omitempty"`
	Expired         bool              `json:"expired"`
	Revoked         bool              `json:"revoked"`
	Suspended       bool              `json:"suspended"`
	SignatureValid  bool              `json:"signature_valid"`
	PublicKeyFound  bool              `json:"public_key_found"`
	// RawPOA exposes the embedded canonical PoA JSON when present (EnvelopeV2 with GAUTH_EMBED_FULL_POA enabled and within size cap).
	RawPOA     string `json:"raw_poa,omitempty"`
	PoAVersion string `json:"poa_version,omitempty"`
	// DetachedSignatureValid indicates the optional detached signature (when present) verified successfully.
	DetachedSignatureValid bool                `json:"detached_signature_valid,omitempty"`
	Warnings               []ValidationWarning `json:"warnings,omitempty"` // validation warnings from enhanced validator
}

// VerifyToken parses & validates a PASETO delegation token. Steps:
// 1. Decrypt using active, previous, then legacy key.
// 2. Sanity check envelope fields.
// 3. Lookup current POA and derive revoked / expired status.
// 4. If POA has a signature: verify digest, locate public key, verify signature.
// Returns a rich result struct and possible RFC error (expired, revoked, integrity_failure, unauthorized, etc.).
//
//nolint:gocyclo // RFC-0111 token verification with delegation chain validation
//nolint:gocyclo // RFC-0111 token verification with delegation chain validation
func (s *Service) VerifyToken(ctx context.Context, tokenString string) (*TokenVerificationResult, error) {
	if tokenString == "" {
		return nil, rfc.New(rfc.ErrInvalidRequest, "empty token")
	}
	v2 := paseto.NewV2()
	// Decrypt raw payload bytes first to inspect version tag.
	var raw map[string]interface{}
	// Accumulate candidate keys
	keys := make([][]byte, 0)
	if s.keyRing != nil {
		if ak := s.keyRing.Active(); ak != nil {
			keys = append(keys, ak.Material)
		}
		for _, pk := range s.keyRing.Previous() {
			if pk != nil {
				keys = append(keys, pk.Material)
			}
		}
	}
	if s.tokenKey != nil {
		keys = append(keys, s.tokenKey)
	}
	decrypted := false
	var plain []byte
	for _, k := range keys {
		var holder json.RawMessage
		if err := v2.Decrypt(tokenString, k, &holder, nil); err == nil {
			decrypted = true
			plain = holder
			break
		}
	}
	if !decrypted {
		return nil, rfc.New(rfc.ErrUnauthorized, "unable to decrypt token")
	}
	if err := json.Unmarshal(plain, &raw); err != nil {
		return nil, rfc.New(rfc.ErrInvalidRequest, "malformed envelope json")
	}
	ver, _ := raw["ver"].(string)
	// Parse version-specific envelope
	var env token.Envelope
	var env2 token.EnvelopeV2
	useV2 := strings.HasSuffix(ver, "env2")
	if useV2 {
		if err := json.Unmarshal(plain, &env2); err != nil {
			return nil, rfc.New(rfc.ErrInvalidRequest, "malformed envelope v2")
		}
		if env2.DelegationID == "" || env2.Grantor == "" || env2.Grantee == "" {
			return nil, rfc.New(rfc.ErrInvalidRequest, "missing envelope fields")
		}
	} else {
		if err := json.Unmarshal(plain, &env); err != nil {
			return nil, rfc.New(rfc.ErrInvalidRequest, "malformed envelope v1")
		}
		if env.DelegationID == "" || env.Grantor == "" || env.Grantee == "" {
			return nil, rfc.New(rfc.ErrInvalidRequest, "missing envelope fields")
		}
	}
	// Normalize common fields
	delegationID := env.DelegationID
	if useV2 {
		delegationID = env2.DelegationID
	}
	// Replay protection & mandatory JTI: require JTI unless explicitly allowed via GAUTH_ALLOW_MISSING_JTI.
	jti := env.JTI
	if useV2 {
		jti = env2.JTI
	}
	if jti == "" {
		if os.Getenv("GAUTH_ALLOW_MISSING_JTI") != "1" {
			return nil, rfc.New(rfc.ErrInvalidRequest, "missing jti")
		}
	} else {
		if !isUUIDv4(jti) {
			return nil, rfc.New(rfc.ErrInvalidRequest, "malformed jti (must be uuid v4)")
		}
		if s.replayStore != nil {
			startRS := s.nowFn()
			seen, err := s.replayStore.Seen(jti)
			if s.metrics != nil {
				s.metrics.ObserveReplayStoreLatency(s.nowFn().Sub(startRS))
			}
			if err != nil {
				if s.metrics != nil {
					s.metrics.IncReplayStoreErrors()
				}
				if s.failClosedReplay {
					return nil, rfc.New(rfc.ErrInvalidRequest, "replay store error")
				}
			} else if seen {
				if s.metrics != nil {
					s.metrics.IncReplayHits()
				}
				return nil, rfc.New(rfc.ErrReplay, "token replay detected")
			}
			recStart := s.nowFn()
			if recErr := s.replayStore.Record(jti, s.nowFn()); recErr != nil {
				if s.metrics != nil {
					s.metrics.IncReplayStoreErrors()
				}
				if s.failClosedReplay {
					return nil, rfc.New(rfc.ErrInvalidRequest, "replay store record error")
				}
			} else if s.metrics != nil {
				s.metrics.ObserveReplayStoreLatency(s.nowFn().Sub(recStart))
			}
			if s.metrics != nil {
				s.metrics.IncReplayMisses()
			}
		} else if s.replay != nil {
			if s.replay.Seen(jti) {
				if s.metrics != nil {
					s.metrics.IncReplayHits()
				}
				return nil, rfc.New(rfc.ErrReplay, "token replay detected")
			}
			s.replay.Record(jti, s.nowFn())
			if s.metrics != nil {
				s.metrics.IncReplayMisses()
			}
		}
	}
	poa, ok := s.repo.Get(delegationID)
	if !ok || poa == nil {
		return nil, rfc.New(rfc.ErrNotFound, delegationID)
	}

	// Jurisdiction enforcement (P1.3): Validate jurisdiction-specific rules DURING token verification.
	// This enforces runtime compliance with GDPR consent, CCPA opt-out, cross-border rules, data residency, and blocked actions.
	// Extract claims from the envelope for enforcement context.
	enforcementClaims := make(map[string]interface{})
	if useV2 {
		// EnvelopeV2 may carry additional fields for jurisdiction context
		if env2.DelegationID != "" {
			enforcementClaims["delegation_id"] = env2.DelegationID
		}
	}
	// Add action claim if we can infer it from envelope scope
	if useV2 && len(env2.Scope) > 0 {
		enforcementClaims["action"] = env2.Scope[0] // Use first scope item as action
	} else if len(env.Scope) > 0 {
		enforcementClaims["action"] = env.Scope[0]
	}

	// Enforce jurisdiction rules (when enabled, this is a no-op otherwise)
	if err := s.enforceJurisdictionOnVerification(ctx, poa, enforcementClaims); err != nil {
		return nil, rfc.New(rfc.ErrUnauthorized, fmt.Sprintf("jurisdiction enforcement denied: %v", err))
	}

	// Offline verification mode (GAUTH_OFFLINE_VERIFICATION=1): prefer embedded PoA over repository lookup
	// when RawPOA is present in EnvelopeV2. This enables token verification without external dependencies.
	// Feature-gated because it changes verification semantics (embedded PoA may differ from repository state).
	offlineMode := os.Getenv("GAUTH_OFFLINE_VERIFICATION") == "1"
	if offlineMode && useV2 && env2.RawPOA != "" {
		// Attempt to extract embedded PoA
		var embeddedPoA PowerOfAttorney
		if err := json.Unmarshal([]byte(env2.RawPOA), &embeddedPoA); err == nil {
			// Validate embedded PoA ID matches envelope
			if embeddedPoA.ID == delegationID {
				// Use embedded PoA instead of repository lookup
				poa = &embeddedPoA
				ok = true
				if s.metrics != nil {
					// Record offline verification usage (reuse existing metric)
					s.metrics.IncEnvelopeRawPOAEmbedded()
				}
			} else if s.metrics != nil {
				// Embedded PoA ID mismatch - fall back to repository PoA but record anomaly
				s.metrics.IncEnvelopeDigestMismatch()
			}
		} else if s.metrics != nil {
			// Embedded PoA parse failure - fall back to repository PoA but record anomaly
			s.metrics.IncEnvelopeDigestMismatch()
		}
	}

	res := &TokenVerificationResult{
		DelegationID: delegationID,
		Grantor: func() string {
			if useV2 {
				return env2.Grantor
			}
			return env.Grantor
		}(),
		Grantee: func() string {
			if useV2 {
				return env2.Grantee
			}
			return env.Grantee
		}(),
		Scope: func() []string {
			if useV2 {
				return env2.Scope
			}
			return env.Scope
		}(),
		Restrictions: func() map[string]string {
			if useV2 {
				return env2.Restrictions
			}
			return env.Restrictions
		}(),
		ExpiresAt: func() time.Time {
			if useV2 {
				return env2.ExpiresAt
			}
			return env.ExpiresAt
		}(),
		Status: func() string {
			if useV2 {
				return env2.Status
			}
			return env.Status
		}(),
		IssuanceChain: func() string {
			if useV2 {
				return env2.IssuanceChain
			}
			return env.IssuanceChain
		}(),
		RevocationChain: func() string {
			if useV2 {
				return env2.RevocationChain
			}
			return env.RevocationChain
		}(),
		RawPOA: func() string {
			if useV2 {
				return env2.RawPOA
			}
			return ""
		}(),
		PoAVersion: func() string {
			if useV2 {
				return env2.PoAVersion
			}
			return ""
		}(),
	}
	now := s.nowFn().UTC()
	exp := res.ExpiresAt
	if now.After(exp) {
		res.Expired = true
	}
	if poa.Status == POAStatusRevoked {
		res.Revoked = true
	}
	if poa.Status == POAStatusSuspended {
		res.Suspended = true
		if s.metrics != nil {
			s.metrics.IncDelegationStatusTransitionFailures() // Reuse existing metric for suspended rejections
		}
		return nil, rfc.New(rfc.ErrUnauthorized, "delegation is suspended")
	}
	// Advanced claims validation (P2.10 sec1.item2): Enforce typ semantic rules when AdvancedClaims present.
	// Feature-gated by GAUTH_ADVANCED_CLAIMS=1 for backward compatibility with tokens issued before P2.10.
	if useV2 && env2.AdvancedClaims != nil && os.Getenv("GAUTH_ADVANCED_CLAIMS") == "1" {
		// Validate AdvancedClaims semantic rules (expiration, typ, metadata)
		if err := env2.AdvancedClaims.ValidateSemantics(); err != nil {
			if s.metrics != nil {
				s.metrics.IncSignatureVerificationFailures() // Reuse existing metric for claims validation failures
			}
			return nil, rfc.New(rfc.ErrUnauthorized, fmt.Sprintf("advanced claims validation failed: %v", err))
		}
		// Enforce typ-specific validation rules
		switch env2.AdvancedClaims.TokenType {
		case "gauth.delegation":
			// Delegation tokens must have non-empty delegation ID and scope
			if env2.DelegationID == "" {
				return nil, rfc.New(rfc.ErrUnauthorized, "typ=gauth.delegation requires valid delegation_id")
			}
			if len(env2.AdvancedClaims.Scope) == 0 {
				return nil, rfc.New(rfc.ErrUnauthorized, "typ=gauth.delegation requires non-empty scope")
			}
		case "gauth.capability":
			// Capability tokens must have scope prefixed with "cap:"
			hasCapScope := false
			for _, scope := range env2.AdvancedClaims.Scope {
				if len(scope) > 4 && scope[:4] == "cap:" {
					hasCapScope = true
					break
				}
			}
			if !hasCapScope {
				return nil, rfc.New(rfc.ErrUnauthorized, "typ=gauth.capability requires at least one 'cap:' prefixed scope")
			}
		case "gauth.token":
			// Generic tokens have no special requirements (minimal validation)
		default:
			// Unknown typ values rejected for security (fail-closed)
			return nil, rfc.New(rfc.ErrUnauthorized, fmt.Sprintf("unknown token type: %s", env2.AdvancedClaims.TokenType))
		}
	}
	// Authenticity verification (shared helper)
	if poa.Signature != nil {
		verr := s.verifyPOASignature(poa)
		if verr != nil {
			// Distinguish missing key vs failure: helper returns wrapped rfc error already
			return nil, verr
		} else {
			res.SignatureValid = true
		}
	}
	// Detached signature verification (envelope-level) – only for V2 envelopes carrying detached fields.
	requireDetachedSig := os.Getenv("GAUTH_REQUIRE_DETACHED_SIGNATURE") == "1"
	hasDetachedSig := useV2 && env2.DetachedSignature != "" && env2.DetachedSignatureKid != "" && env2.CanonicalDigest != ""

	// Fail-closed mode: if detached signatures are mandatory, reject tokens without them
	if requireDetachedSig && !hasDetachedSig {
		if s.metrics != nil {
			s.metrics.IncSignatureVerificationFailures()
		}
		incDetachedVerify("missing_required_signature")
		return nil, rfc.New(rfc.ErrUnauthorized, "detached signature required but missing")
	}

	if hasDetachedSig {
		// Recompute canonical digest + bytes from stored POA (repository copy) for authenticity binding.
		if dig, canon, derr := CanonicalPOADigest(poa); derr == nil {
			if dig == env2.CanonicalDigest { // only attempt if digest aligns
				// locate public key (keyProvider preferred then keyRing) matching detached kid
				var pub []byte
				if s.keyProvider != nil {
					if pkBytes, algo, err := s.keyProvider.PublicKey(env2.DetachedSignatureKid); err == nil && algo == cr.AlgoEd25519 && len(pkBytes) == ed25519.PublicKeySize {
						pub = pkBytes
					}
				}
				if len(pub) == 0 && s.keyRing != nil {
					if ak := s.keyRing.Active(); ak != nil && ak.ID == env2.DetachedSignatureKid {
						pub = ak.Material
					}
					if len(pub) == 0 {
						for _, pk := range s.keyRing.Previous() {
							if pk != nil && pk.ID == env2.DetachedSignatureKid {
								pub = pk.Material
								break
							}
						}
					}
				}
				if len(pub) == ed25519.PublicKeySize {
					sigBytes, decErr := base64.StdEncoding.DecodeString(env2.DetachedSignature)
					if decErr == nil && len(sigBytes) == ed25519.SignatureSize && ed25519.Verify(ed25519.PublicKey(pub), canon, sigBytes) {
						res.DetachedSignatureValid = true
						if s.metrics != nil {
							s.metrics.IncSignatureVerifications()
						}
						incDetachedVerify("success")
					} else if s.metrics != nil { // verification failure
						s.metrics.IncSignatureVerificationFailures()
						if decErr != nil {
							incDetachedVerify("invalid_signature")
						} else {
							incDetachedVerify("invalid_signature")
						}
					}
				} else if s.metrics != nil { // missing public key
					s.metrics.IncSignaturePublicKeyMissing()
					incDetachedVerify("pubkey_missing")
				}
			} else { // digest mismatch
				if s.metrics != nil {
					s.metrics.IncSignatureVerificationFailures()
					s.metrics.IncEnvelopeDigestMismatch()
				}
				incDetachedVerify("digest_mismatch")
			}
		} // end canonical digest recompute block
	}
	if poa.Threshold > 1 {
		if err := s.verifyMultiSignatures(poa); err != nil {
			return nil, err
		}
	}
	// Subject enforcement: prefer typed key then legacy fallback for transitional callers.
	if sub := ctx.Value(ctxKeySubject); sub != nil {
		if sStr, ok2 := sub.(string); ok2 && sStr != res.Grantee {
			return nil, rfc.New(rfc.ErrUnauthorized, "subject mismatch")
		}
	} else if legacy := ctx.Value(LegacyCtxSubject); legacy != nil { // legacy string key path
		if sStr, ok2 := legacy.(string); ok2 && sStr != res.Grantee {
			return nil, rfc.New(rfc.ErrUnauthorized, "subject mismatch")
		}
	}
	if res.Expired {
		return res, rfc.New(rfc.ErrExpired, "token expired")
	}
	if res.Revoked {
		return res, rfc.New(rfc.ErrRevoked, "delegation revoked")
	}
	// Conditional restriction enforcement (advanced validator semantics). We evaluate current time-based gating.
	restrict := res.Restrictions
	if restrict != nil {
		// valid_hours HH-HH
		if vh, ok := restrict["valid_hours"]; ok {
			parts := strings.Split(vh, "-")
			if len(parts) == 2 {
				sh, eh := parts[0], parts[1]
				if len(sh) == 2 && len(eh) == 2 {
					if sHi, err1 := strconv.Atoi(sh); err1 == nil {
						if eHi, err2 := strconv.Atoi(eh); err2 == nil {
							nowHour := now.Hour()
							allowed := false
							if sHi <= eHi { // normal interval
								allowed = nowHour >= sHi && nowHour < eHi
							} else { // wrap-around (e.g., 22-06)
								allowed = nowHour >= sHi || nowHour < eHi
							}
							if !allowed {
								return nil, rfc.New(rfc.ErrUnauthorized, "outside valid_hours window")
							}
						}
					}
				}
			}
		}
		if vwd, ok := restrict["valid_weekdays"]; ok {
			items := strings.Split(vwd, ",")
			if len(items) > 0 {
				weekday := int(now.Weekday()) // Sunday=0
				okDay := false
				for _, it := range items {
					it = strings.TrimSpace(it)
					if it == "" {
						continue
					}
					if v, err := strconv.Atoi(it); err == nil && v == weekday {
						okDay = true
						break
					}
				}
				if !okDay {
					return nil, rfc.New(rfc.ErrUnauthorized, "weekday not permitted")
				}
			}
		}
	}
	return res, nil
}

// VerifyDelegationToken is a convenience exported wrapper around Service.VerifyToken.
// Provides a stable external API surface if later we abstract service internals.
func VerifyDelegationToken(ctx context.Context, svc *Service, tok string) (*TokenVerificationResult, error) {
	if svc == nil {
		return nil, rfc.New(rfc.ErrInvalidRequest, "nil service")
	}
	return svc.VerifyToken(ctx, tok)
}

// ExtractEmbeddedPoA extracts and validates the embedded PoA definition from a TokenVerificationResult.
// This enables offline token verification without requiring access to the PoA repository.
//
// Requirements (RFC0115 sec3.item2):
//   - RawPOA field must be non-empty (GAUTH_EMBED_FULL_POA=1 must have been enabled during token issuance)
//   - RawPOA must contain valid canonical JSON representing a PowerOfAttorney
//   - Extracted PoA.ID must match the DelegationID from the token envelope
//   - Extracted PoA must pass basic structural validation (non-empty fields, valid timestamps)
//
// Returns:
//   - *PowerOfAttorney: the extracted and validated PoA definition
//   - error: validation failure (ErrInvalidRequest if RawPOA missing/invalid, ErrIntegrityFailure if ID mismatch)
//
// Usage:
//
//	result, _ := svc.VerifyToken(ctx, tokenString)
//	poa, err := ExtractEmbeddedPoA(result)
//	if err != nil {
//	    // Fall back to repository lookup or reject token
//	}
//	// Use poa for authorization without repository access
func ExtractEmbeddedPoA(result *TokenVerificationResult) (*PowerOfAttorney, error) {
	if result == nil {
		return nil, rfc.New(rfc.ErrInvalidRequest, "nil verification result")
	}

	// Check if RawPOA is present (requires GAUTH_EMBED_FULL_POA=1 during issuance)
	if result.RawPOA == "" {
		return nil, rfc.New(rfc.ErrInvalidRequest, "no embedded poa definition (GAUTH_EMBED_FULL_POA=1 not enabled)")
	}

	// Parse canonical JSON - the canonical format encodes version as a string (for digest stability),
	// but PowerOfAttorney expects an int. We need a custom unmarshal step.
	// Create an intermediate struct that accepts version as either string or int.
	type canonicalPoA struct {
		PowerOfAttorney
		VersionRaw interface{} `json:"version"` // Accept string or int
	}

	var intermediate canonicalPoA
	if err := json.Unmarshal([]byte(result.RawPOA), &intermediate); err != nil {
		return nil, rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("invalid embedded poa json: %v", err))
	}

	// Convert version from string to int if needed
	switch v := intermediate.VersionRaw.(type) {
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			intermediate.PowerOfAttorney.Version = parsed
		} else {
			return nil, rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("invalid version format: %s", v))
		}
	case float64: // JSON numbers decode as float64
		intermediate.PowerOfAttorney.Version = int(v)
	case int:
		intermediate.PowerOfAttorney.Version = v
	default:
		return nil, rfc.New(rfc.ErrInvalidRequest, "version field must be string or number")
	}

	poa := &intermediate.PowerOfAttorney

	// Validate extracted PoA matches token envelope DelegationID
	if poa.ID == "" {
		return nil, rfc.New(rfc.ErrIntegrityFailure, "embedded poa missing id field")
	}
	if result.DelegationID != "" && poa.ID != result.DelegationID {
		return nil, rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("embedded poa id mismatch: envelope=%s embedded=%s", result.DelegationID, poa.ID))
	}

	// Basic structural validation (ensures PoA is not malformed)
	if poa.Grantor == "" {
		return nil, rfc.New(rfc.ErrInvalidRequest, "embedded poa missing grantor")
	}
	if poa.Grantee == "" {
		return nil, rfc.New(rfc.ErrInvalidRequest, "embedded poa missing grantee")
	}
	if len(poa.Scope) == 0 {
		return nil, rfc.New(rfc.ErrInvalidRequest, "embedded poa missing scope")
	}

	// Temporal validation (ensure timestamps are reasonable)
	if poa.ValidUntil.IsZero() {
		return nil, rfc.New(rfc.ErrInvalidRequest, "embedded poa missing valid_until")
	}
	if !poa.ValidFrom.IsZero() && poa.ValidUntil.Before(poa.ValidFrom) {
		return nil, rfc.New(rfc.ErrInvalidRequest, "embedded poa valid_until before valid_from")
	}

	return poa, nil
}

// ExtractEmbeddedPoAWithAudit is an enhanced version of ExtractEmbeddedPoA that logs extraction
// events to the audit trail and tracks metrics. This completes sec3.item2 P1 requirements for
// audit persistence and monitoring of embedded PoA usage.
//
// This method should be used when Service context is available. For offline extraction without
// Service context, use ExtractEmbeddedPoA directly.
//
// Parameters:
//   - ctx: context for audit logging
//   - result: token verification result containing embedded PoA
//
// Returns:
//   - *PowerOfAttorney: extracted and validated PoA
//   - error: validation/extraction failure
//
// Audit events logged:
//   - embedded_poa_extraction_success (includes PoA ID, grantor, grantee, extraction timestamp)
//   - embedded_poa_extraction_failure (includes error reason, token JTI if available)
//
// Metrics tracked:
//   - Extraction attempts (success/failure counters)
//   - Extraction latency (histogram)
//   - Embedded PoA size distribution (histogram)
func (s *Service) ExtractEmbeddedPoAWithAudit(ctx context.Context, result *TokenVerificationResult) (*PowerOfAttorney, error) {
	start := time.Now()

	// Attempt extraction using base function
	poa, err := ExtractEmbeddedPoA(result)

	// Track metrics
	extractionLatency := time.Since(start)
	if s.metrics != nil {
		if err == nil {
			s.metrics.IncEnvelopeRawPOAEmbedded() // Reuse existing metric for extraction success
			// Track size if RawPOA present
			if result.RawPOA != "" {
				// Size distribution tracking could be added here
				_ = len(result.RawPOA)
			}
		} else {
			s.metrics.IncEnvelopeRawPOATooLarge() // Reuse as generic extraction failure counter
		}
	}

	// Log audit event
	if s.audit != nil {
		auditEvent := map[string]interface{}{
			"event_type":            "embedded_poa_extraction",
			"timestamp":             s.nowFn().UTC().Format(time.RFC3339Nano),
			"extraction_latency_ms": extractionLatency.Milliseconds(),
		}

		if err == nil && poa != nil {
			auditEvent["status"] = "success"
			auditEvent["poa_id"] = poa.ID
			auditEvent["grantor"] = poa.Grantor
			auditEvent["grantee"] = poa.Grantee
			auditEvent["poa_version"] = poa.Version
			auditEvent["scope"] = poa.Scope
			if result.RawPOA != "" {
				auditEvent["raw_poa_size_bytes"] = len(result.RawPOA)
			}
		} else {
			auditEvent["status"] = "failure"
			auditEvent["error"] = err.Error()
			if result.DelegationID != "" {
				auditEvent["delegation_id"] = result.DelegationID
			}
		}

		// Log asynchronously (non-blocking)
		_ = s.audit.Log(ctx, auditEvent)
	}

	return poa, err
}

// Service provides RFC 0111 power-of-attorney services
// AuditLogger is the interface subset we rely on (MemoryLogger & FileLogger both satisfy).
type AuditLogger interface {
	Log(context.Context, interface{}) error
	Query(context.Context, *audit.Filter) ([]*audit.Event, error)
}

type Service struct {
	repo                    POARepository // persistence abstraction (Milestone 2A)
	audit                   AuditLogger
	authz                   authz.Authorizer
	pdp                     pdp.Engine   // optional modern PDP engine
	ledger                  ledger.Store // optional immutable audit ledger backend
	nowFn                   func() time.Time
	clockSkew               time.Duration // tolerated clock skew for ValidFrom/ValidUntil windows
	revChain                *delegation.RevocationChain
	issChain                *DelegationChain
	tokenKey                []byte                      // legacy symmetric key (to be deprecated after envelope migration)
	keyRing                 *keyring.KeyRing            // Milestone 2A: key management (active + previous)
	metrics                 metrics.Metrics             // instrumentation (noop by default)
	signerProvider          func() (cr.Signer, error)   // optional signing capability
	strictAuthenticity      bool                        // if true, missing public key becomes integrity failure instead of soft skip
	anchorClient            AnchorClient                // optional external anchoring client
	keyProvider             cr.KeyProvider              // asymmetric key provider for signature verification
	replay                  *replayCache                // optional in-memory replay protection
	replayStore             ReplayStore                 // optional external distributed replay store (takes precedence if non-nil)
	sigReplayStore          SignatureReplayStore        // optional signature replay protection store (issuance path)
	failClosedReplay        bool                        // if true, replay store errors become invalid_request instead of fail-open
	limits                  ValidationLimits            // configurable validation limits
	poaValidator            PoAValidator                // semantic validator applied post basic validation
	enhancedValidator       *EnhancedPoAValidator       // enhanced validator with warning collection and daily limits (optional)
	mandatorySignatures     bool                        // if true, issuance aborts when signature cannot be produced
	attestAnchors           *attest.TrustAnchorRegistry // optional trust anchor registry for attestation proofs
	jurisdictionEnforcement *JurisdictionEnforcement    // optional jurisdiction-specific enforcement (P1.3)
	auditSink               AuditSink                   // optional external audit sink for token lifecycle events (P1.4)
	// semanticCounters prototype: fine-grained semantic rejection reasons (will be surfaced via endpoints later)
	semanticCounters struct {
		AmountLimitExceeded      uint64
		DailyAmountLimitExceeded uint64
		CurrencyMismatch         uint64
		ScopeViolation           uint64
		RestrictionMismatch      uint64
	}
	dailyAmounts   map[string]float64 // key: delegationID|YYYY-MM-DD cumulative requested amount
	dailyAmountsMu sync.Mutex
}

// AttachEvidenceHashes appends new evidence hash(es) to a POA ensuring uniqueness and basic validation.
// Rules:
// - POA must exist and not be revoked/terminated (evidence on historical records disallowed for now)
// - Each hash must be lowercase hex (0-9a-f) length 64 (sha256) or 128 (sha512) – future extension may allow prefix algo:
// - Duplicates (already present) are ignored but if submission contains only duplicates we treat as failure for observability
// - On success: UpdatedAt updated, repository persisted, audit event emitted, metrics incremented, per-POA gauge updated
func (s *Service) AttachEvidenceHashes(ctx context.Context, poaID string, hashes []string) (*PowerOfAttorney, error) {
	if poaID == "" {
		return nil, rfc.New(rfc.ErrInvalidRequest, "missing poa id")
	}
	if len(hashes) == 0 {
		if s.metrics != nil {
			s.metrics.IncEvidenceAttachmentFailures()
		}
		return nil, rfc.New(rfc.ErrInvalidRequest, "no hashes provided")
	}
	p, ok := s.repo.Get(poaID)
	if !ok {
		if s.metrics != nil {
			s.metrics.IncEvidenceAttachmentFailures()
		}
		return nil, rfc.New(rfc.ErrNotFound, "poa not found")
	}
	if p.Status == POAStatusRevoked || p.Status == POAStatusTerminated {
		if s.metrics != nil {
			s.metrics.IncEvidenceAttachmentFailures()
		}
		return nil, rfc.New(rfc.ErrInvalidRequest, "cannot attach evidence to finalized poa")
	}
	// build set of existing
	existing := make(map[string]struct{}, len(p.EvidenceHashes))
	for _, eh := range p.EvidenceHashes {
		existing[eh] = struct{}{}
	}
	added := 0
	hexRx := regexp.MustCompile(`^[0-9a-f]{64}([0-9a-f]{64})?$`)
	for _, h := range hashes {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			continue
		}
		if !hexRx.MatchString(h) {
			if s.metrics != nil {
				s.metrics.IncEvidenceAttachmentFailures()
			}
			return nil, rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("invalid hash format: %s", h))
		}
		if _, dup := existing[h]; dup {
			continue
		}
		p.EvidenceHashes = append(p.EvidenceHashes, h)
		existing[h] = struct{}{}
		added++
	}
	if added == 0 {
		if s.metrics != nil {
			s.metrics.IncEvidenceAttachmentFailures()
		}
		return nil, rfc.New(rfc.ErrInvalidRequest, "no new hashes (all duplicates)")
	}
	p.UpdatedAt = s.nowFn()
	if err := s.repo.Update(p); err != nil {
		if s.metrics != nil {
			s.metrics.IncEvidenceAttachmentFailures()
		}
		return nil, rfc.New(rfc.ErrInternal, fmt.Sprintf("update failed: %v", err))
	}
	// audit event
	if s.audit != nil {
		ev := audit.NewEvent(audit.TypeAuth, "evidence_attach", audit.ResultSuccess)
		ev.Subject = p.Grantor
		ev.Object = poaID
		ev.Metadata = map[string]interface{}{"added": added, "total": len(p.EvidenceHashes)}
		_ = s.audit.Log(ctx, ev)
		// Send to external audit sink (P1.4)
		s.sendToAuditSink(ctx, ev)
	}
	if s.metrics != nil {
		for i := 0; i < added; i++ {
			s.metrics.IncEvidenceAttachment()
		}
		s.metrics.SetEvidenceHashesPerPOA(poaID, len(p.EvidenceHashes))
	}
	return p, nil
}

// ValidationLimits defines configurable bounds for delegation request validation.
// Zero/negative values are replaced by safe defaults in applyDefaults.
type ValidationLimits struct {
	MaxScopeItems        int
	MaxScopeLen          int
	MaxRestrictions      int
	MaxRestrictionKeyLen int
	MaxRestrictionValLen int
	MaxDuration          time.Duration // absolute cap on delegation duration
}

func (vl *ValidationLimits) applyDefaults() {
	if vl.MaxScopeItems <= 0 {
		vl.MaxScopeItems = 32
	}
	if vl.MaxScopeLen <= 0 {
		vl.MaxScopeLen = 128
	}
	if vl.MaxRestrictions <= 0 {
		vl.MaxRestrictions = 32
	}
	if vl.MaxRestrictionKeyLen <= 0 {
		vl.MaxRestrictionKeyLen = 64
	}
	if vl.MaxRestrictionValLen <= 0 {
		vl.MaxRestrictionValLen = 256
	}
	if vl.MaxDuration <= 0 {
		vl.MaxDuration = 24 * time.Hour * 365
	}
}

// WithValidationLimits installs custom validation limits (defaults filled for unset fields).
func WithValidationLimits(l ValidationLimits) Option {
	return func(s *Service) { l.applyDefaults(); s.limits = l }
}

// replayCache provides naive in-memory replay tracking.
type replayCache struct {
	mu      sync.Mutex
	entries map[string]time.Time
	order   []string
	max     int
	ttl     time.Duration
}

// ReplayStore defines a distributed JTI tracking interface.
// Implementations MUST be safe for concurrent use. Seen returns true if the JTI is already recorded
// (indicating replay) and false otherwise. Record stores the JTI; implementations SHOULD set a TTL
// matching retention policies (e.g. token lifetime). Errors are surfaced so callers can instrument
// store health; verification logic will fail open (treat as miss) but still increment error metrics.
type ReplayStore interface {
	Seen(jti string) (bool, error)
	Record(jti string, at time.Time) error
}

// SignatureReplayStore tracks previously used signature digests (digest+keyid compound) to
// prevent replay of an identical signature over a mutated POA payload. While canonical digest
// domain separation mitigates cross‑context confusion, an attacker who obtains a valid signature
// over a canonical form could attempt to reuse it for issuance if dynamic fields were excluded.
// Tracking first‑use prevents silent acceptance of duplicate signatures (forensic strengthening).
// Implementations MUST be concurrency‑safe. Seen returns true if compound key already present.
type SignatureReplayStore interface {
	SeenSignature(sigKey string) (bool, error)
	RecordSignature(sigKey string, at time.Time) error
}

func newReplayCache(max int, ttl time.Duration) *replayCache {
	return &replayCache{entries: make(map[string]time.Time), order: make([]string, 0, max), max: max, ttl: ttl}
}

func (rc *replayCache) cleanup(now time.Time) {
	if len(rc.order) == 0 {
		return
	}
	fresh := rc.order[:0]
	for _, j := range rc.order {
		if t := rc.entries[j]; now.Sub(t) > rc.ttl {
			delete(rc.entries, j)
			continue
		}
		fresh = append(fresh, j)
	}
	rc.order = fresh
}

func (rc *replayCache) Seen(jti string) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cleanup(time.Now())
	_, ok := rc.entries[jti]
	return ok
}

func (rc *replayCache) Record(jti string, now time.Time) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if _, exists := rc.entries[jti]; exists {
		return
	}
	if len(rc.order) >= rc.max {
		old := rc.order[0]
		rc.order = rc.order[1:]
		delete(rc.entries, old)
	}
	rc.entries[jti] = now
	rc.order = append(rc.order, jti)
}

// mockReplayStore is a simple in-memory implementation of ReplayStore for tests.
type mockReplayStore struct {
	mu   sync.Mutex
	seen map[string]time.Time
	ttl  time.Duration
}

func newMockReplayStore(ttl time.Duration) *mockReplayStore {
	return &mockReplayStore{seen: make(map[string]time.Time), ttl: ttl}
}
func (m *mockReplayStore) Seen(jti string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.seen[jti]; ok {
		if m.ttl > 0 && time.Since(t) > m.ttl {
			delete(m.seen, jti)
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

func (m *mockReplayStore) Record(jti string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.seen[jti]; !exists {
		m.seen[jti] = at
	}
	return nil
}

// verifyPOASignature performs canonical digest computation and Ed25519 verification.
// Returns nil on success or an RFC-wrapped error on failure. Side-effects metrics counters.
func (s *Service) verifyPOASignature(poa *PowerOfAttorney) error {
	if poa == nil || poa.Signature == nil {
		return nil
	}
	dig, canon, derr := CanonicalPOADigest(poa)
	if derr != nil {
		if s.metrics != nil {
			// Treat canonicalization failure as a digest mismatch classification for observability purposes.
			s.metrics.IncSignatureVerificationFailures()
			s.metrics.IncEnvelopeDigestMismatch()
			s.metrics.IncEnvelopeDigestMismatchReason("canonicalization_error")
		}
		return rfc.New(rfc.ErrIntegrityFailure, "canonical digest failed")
	}
	if dig != poa.Signature.DigestHex {
		if s.metrics != nil {
			// Reason classification: equal length => domain conflict, else tamper suspected.
			class := reasonTamperSuspected
			if len(poa.Signature.DigestHex) == len(dig) {
				class = reasonDomainConflict
			}
			s.metrics.IncSignatureVerificationFailures()
			s.metrics.IncEnvelopeDigestMismatch()
			s.metrics.IncEnvelopeDigestMismatchReason(class)
		}
		return rfc.New(rfc.ErrIntegrityFailure, errDigestMismatch)
	}
	// Dispatch verification through registry; algorithm must be registered.
	if cr.GetAlgorithm(poa.Signature.Algorithm) == nil {
		if s.metrics != nil {
			s.metrics.IncSignatureVerificationFailures()
		}
		return rfc.New(rfc.ErrIntegrityFailure, "unsupported algorithm")
	}
	if err := cr.VerifyAlgorithm(poa.Signature.Algorithm, canon, poa.Signature.SigBase64, poa.Signature.KeyID, s.keyProvider); err != nil {
		if s.metrics != nil {
			s.metrics.IncSignatureVerificationFailures()
		}
		if errors.Is(err, cr.ErrUnknownKey) { // treat unknown key as soft skip unless strictAuthenticity
			if s.metrics != nil {
				s.metrics.IncSignaturePublicKeyMissing()
			}
			if s.strictAuthenticity {
				return rfc.New(rfc.ErrIntegrityFailure, "signature public key missing (strict mode)")
			}
			return nil
		}
		return rfc.New(rfc.ErrIntegrityFailure, "signature verification failed")
	}
	if s.metrics != nil {
		s.metrics.IncSignatureVerifications()
	}
	return nil
}

// WithAnchorClient injects an AnchorClient implementation.
func WithAnchorClient(ac AnchorClient) Option { return func(s *Service) { s.anchorClient = ac } }

// WithReplayProtection enables simple in-memory replay protection.
// maxEntries caps tracked JTIs; ttl bounds lifetime of each entry.
func WithReplayProtection(maxEntries int, ttl time.Duration) Option {
	return func(s *Service) {
		if maxEntries <= 0 {
			maxEntries = 10000
		}
		if ttl <= 0 {
			ttl = 15 * time.Minute
		}
		s.replay = newReplayCache(maxEntries, ttl)
	}
}

// WithReplayStore installs a distributed replay store implementation. If provided, VerifyToken will
// consult the store first (atomic first-seen semantics) and fall back to in-memory cache only if the
// store is nil. Store errors are treated as miss (fail-open) but should be instrumented by metrics.
//
// For production deployments requiring replay protection across process restarts, use DurableReplayStore
// with automatic snapshot scheduling and crash recovery:
//
//	config := replay.DurableReplayStoreConfig{
//	    WALPath:          "/var/lib/gauth/replay.wal",
//	    TTL:              24 * time.Hour,
//	    SnapshotInterval: 5 * time.Minute,
//	}
//	durableStore, _ := replay.NewDurableReplayStore(config)
//	adapter := replay.NewDurableReplayStoreAdapter(durableStore)
//	svc, _ := NewService(WithReplayStore(adapter), ...)
//
// See docs/REPLAY_PERSISTENCE.md for architecture, recovery procedures, and operational guide.
func WithReplayStore(rs ReplayStore) Option {
	return func(s *Service) {
		if rs != nil {
			s.replayStore = rs
		}
	}
}

// WithSignatureReplayStore installs a signature replay store used during delegation issuance.
// If installed, CreateDelegation will reject issuance when an identical digest+keyid has been
// previously observed (ErrReplay). Store errors are treated as miss unless failClosedReplay enabled.
func WithSignatureReplayStore(ss SignatureReplayStore) Option {
	return func(s *Service) {
		if ss != nil {
			s.sigReplayStore = ss
		}
	}
}

// DelegationGraphNode represents a hierarchical delegation node for export.
// Scope is copied (non-mutated) to avoid exposing internal slice references.
type DelegationGraphNode struct {
	ID     string    `json:"id"`
	Parent string    `json:"parent,omitempty"`
	Depth  int       `json:"depth"`
	Scope  []string  `json:"scope,omitempty"`
	Status POAStatus `json:"status"`
}

// BuildDelegationGraph returns all current delegations as a flat slice of nodes including
// parent linkage and depth. Repository must support List(); if not available an error is returned.
func (s *Service) BuildDelegationGraph(ctx context.Context) ([]DelegationGraphNode, error) {
	type lister interface{ List() []*PowerOfAttorney }
	l, ok := s.repo.(lister)
	if !ok {
		return nil, fmt.Errorf("repository does not support List for graph export")
	}
	poas := l.List()
	nodes := make([]DelegationGraphNode, 0, len(poas))
	for _, p := range poas {
		if p == nil {
			continue
		}
		// defensive copy of scope slice
		scopeCopy := append([]string{}, p.Scope...)
		nodes = append(nodes, DelegationGraphNode{ID: p.ID, Parent: p.ParentPOAID, Depth: p.Depth, Scope: scopeCopy, Status: p.Status})
	}
	if s.metrics != nil {
		s.metrics.IncDelegationGraphExports()
		s.metrics.SetDelegationGraphNodeCount(len(nodes))
	}
	return nodes, nil
}

// AnchorClient defines minimal interface for external anchoring of chain tips.
type AnchorClient interface{ Anchor(hash string) error }

// NoopAnchorClient implements AnchorClient and always succeeds.
type NoopAnchorClient struct{}

func (n NoopAnchorClient) Anchor(hash string) error { return nil }

// (Constructors with options defined earlier in file)

// WithClock allows tests to override the time source (experimental)
func (s *Service) WithClock(f func() time.Time) *Service { s.nowFn = f; return s }

// CreateDelegation creates a new power-of-attorney delegation
// CreateDelegation is a backward-compatible wrapper that uses context.Background().
func (s *Service) CreateDelegation(req DelegationRequest) (*DelegationResponse, error) {
	return s.CreateDelegationCtx(context.Background(), req)
	//nolint:gocyclo // Delegation creation with constraint validation
}

// CreateDelegationCtx creates a new power-of-attorney with a caller-supplied context (supports cancellation / deadlines).
//
//nolint:gocyclo // Delegation creation with constraint validation
func (s *Service) CreateDelegationCtx(ctx context.Context, req DelegationRequest) (*DelegationResponse, error) {
	// Validate request
	if err := s.validateDelegationRequest(req); err != nil {
		return nil, rfc.New(rfc.ErrInvalidRequest, err.Error())
	}
	// Sub-delegation depth derivation (before authorization side-effects)
	depth := 0
	if req.ParentPOAID != "" {
		parent, ok := s.repo.Get(req.ParentPOAID)
		if !ok || parent == nil {
			return nil, rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("parent_poa_id not found: %s", req.ParentPOAID))
		}
		depth = parent.Depth + 1
		maxDepth := 5
		if v := os.Getenv("GAUTH_MAX_DELEGATION_DEPTH"); v != "" {
			if iv, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && iv > 0 {
				maxDepth = iv
			}
		}
		if depth > maxDepth {
			return nil, rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("delegation depth %d exceeds max %d", depth, maxDepth))
		}
		// Scope inheritance enforcement: child scope must be subset of parent scope semantics.
		if err := validateInheritedScope(parent.Scope, req.Scope); err != nil {
			return nil, rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("scope inheritance invalid: %v", err))
		}
	}

	// Check authorization via PDP if present; otherwise legacy authorizer.
	if s.pdp != nil {
		pdpDec, err := s.pdp.Evaluate(ctx, pdp.Request{Subject: req.Grantor, Action: "create_delegation", Resource: "poa", Attributes: map[string]string{"grantee": req.Grantee}, Time: s.nowFn()})
		if err != nil {
			return nil, rfc.New(rfc.ErrInternal, fmt.Sprintf("pdp evaluation failed: %v", err))
		}
		if !pdpDec.Allow {
			return nil, rfc.New(rfc.ErrUnauthorized, pdpDec.Reason)
		}
	} else {
		authReq := authz.Request{Subject: req.Grantor, Action: "create_delegation", Resource: "poa", Context: map[string]string{"grantee": req.Grantee}}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		decision, err := s.authz.Authorize(ctx, authReq)
		if err != nil {
			return nil, rfc.New(rfc.ErrInternal, fmt.Sprintf("authorization failed: %v", err))
		}
		if !decision.Allow {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			event := audit.NewEvent(audit.TypeAuth, "create_delegation", audit.ResultFailure)
			event.Subject = req.Grantor
			event.Object = "poa"
			if event.Metadata == nil {
				event.Metadata = map[string]interface{}{}
			}
			event.Metadata["reason"] = decision.Reason
			if err := s.audit.Log(ctx, event); err != nil {
				return nil, fmt.Errorf("audit log failed: %w", err)
			}
			// Send to external audit sink (P1.4)
			s.sendToAuditSink(ctx, event)
			return nil, rfc.New(rfc.ErrUnauthorized, decision.Reason)
		}
	}

	// Create power-of-attorney
	now := s.nowFn()
	poa := &PowerOfAttorney{
		ID:           generatePOAID(),
		Grantor:      req.Grantor,
		Grantee:      req.Grantee,
		Scope:        req.Scope,
		Restrictions: req.Restrictions,
		ValidFrom:    now,
		ValidUntil:   now.Add(req.Duration),
		Status:       POAStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
		Version:      1,
		Signers:      req.Signers,
		Threshold:    req.Threshold,
		Weights:      req.Weights,
		AgentType:    req.AgentType,
		Sector:       req.Sector,
		ActionClass:  req.ActionClass,
		ParentPOAID:  req.ParentPOAID,
		Depth:        depth,
	}

	// Jurisdiction enforcement (P1.3): Validate jurisdiction-specific rules BEFORE creating delegation.
	// This gates creation based on GDPR consent, CCPA opt-out, cross-border rules, data residency, and blocked actions.
	// When enforcement is disabled (nil), this is a no-op allowing all operations (backward compatible).
	if err := s.enforceJurisdictionOnIssuance(ctx, req, poa); err != nil {
		return nil, rfc.New(rfc.ErrUnauthorized, fmt.Sprintf("jurisdiction enforcement denied: %v", err))
	}

	// Hierarchical digest activation gating (Version 4). Controlled by env flags:
	// GAUTH_ENABLE_HIER_DIGEST=1 enables automatic Version bump to 4 for new issuances.
	// GAUTH_FORCE_HIER_DIGEST=1 enforces Version=4 even if enabling logic conditions fail (defensive activation).
	enableHier := os.Getenv("GAUTH_ENABLE_HIER_DIGEST") == "1"
	forceHier := os.Getenv("GAUTH_FORCE_HIER_DIGEST") == "1"
	if enableHier || forceHier {
		poa.Version = 4
		// When sub-delegation capture parent's canonical digest for binding (ParentDigest). Missing parent digest increments metric & may error if forced.
		if poa.ParentPOAID != "" {
			parent, ok := s.repo.Get(poa.ParentPOAID)
			if ok && parent != nil {
				if dig, _, derr := CanonicalPOADigest(parent); derr == nil {
					poa.ParentDigest = dig
				} else {
					if s.metrics != nil {
						s.metrics.IncHierDigestParentDigestMissing()
					}
					if forceHier {
						return nil, rfc.New(rfc.ErrIntegrityFailure, "parent canonical digest failed")
					}
				}
			} else {
				if s.metrics != nil {
					s.metrics.IncHierDigestParentDigestMissing()
				}
				if forceHier {
					return nil, rfc.New(rfc.ErrIntegrityFailure, "parent not found")
				}
			}
		} else {
			// Root issuance still increments issued metric when hierarchical digest enabled.
			if s.metrics != nil {
				s.metrics.IncHierDigestIssued()
			}
		}
	}
	// Auto-bump version when taxonomy present (non-empty fields) to engage V3 canonical domain separation logic for single-sig.
	if poa.AgentType != "" || poa.Sector != "" || poa.ActionClass != "" {
		// Preserve hierarchical V4 if already bumped; else set to 3
		if poa.Version < 4 {
			poa.Version = 3
		}
	}

	// Multi-signature enforcement (prototype RFC115-C8). If Threshold>1 require structural validity.
	if err := ValidateMultiSignature(poa); err != nil {
		return nil, rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("multi-signature invalid: %v", err))
	}

	// Taxonomy validation (Version>=3 only; legacy versions ignore taxonomy fields even if set)
	if poa.Version >= 3 {
		if err := ValidateTaxonomy(poa); err != nil {
			return nil, rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("taxonomy invalid: %v", err))
		}
	}

	// Semantic validation (cross-field invariants)
	if s.poaValidator != nil {
		if verr := s.poaValidator.Validate(poa); verr != nil {
			return nil, verr
		}
	}

	// Enhanced validation with warning collection
	var warnings []ValidationWarning
	if s.enhancedValidator != nil {
		result := s.enhancedValidator.ValidateWithResult(ctx, poa)
		if !result.Valid {
			return nil, result.Error
		}
		warnings = result.Warnings
	}

	// Attempt canonical digest + signature
	if s.signerProvider != nil {
		if signer, err := s.signerProvider(); err == nil && signer != nil {
			if dig, canon, derr := CanonicalPOADigest(poa); derr == nil {
				if sig, serr := signer.Sign(canon); serr == nil {
					poa.Signature = &POASignature{Algorithm: signer.Algorithm(), KeyID: signer.KeyID(), DigestHex: dig, SigBase64: base64.StdEncoding.EncodeToString(sig), Canonical: canon}
					if s.metrics != nil {
						s.metrics.IncSignaturesIssued()
					}
					// Signature replay protection (digest+keyid compound). Only apply to successful signature issuance.
					if s.sigReplayStore != nil && poa.Signature != nil {
						compound := poa.Signature.DigestHex + "|" + poa.Signature.KeyID
						seen, rErr := s.sigReplayStore.SeenSignature(compound)
						if rErr != nil {
							if s.metrics != nil {
								s.metrics.IncReplayStoreErrors()
							}
							if s.failClosedReplay {
								return nil, rfc.New(rfc.ErrInvalidRequest, "signature replay store error")
							}
						} else if seen {
							if s.metrics != nil {
								s.metrics.IncReplayHits()
							}
							return nil, rfc.New(rfc.ErrReplay, "signature replay detected")
						}
						if recErr := s.sigReplayStore.RecordSignature(compound, s.nowFn()); recErr != nil {
							if s.metrics != nil {
								s.metrics.IncReplayStoreErrors()
							}
							if s.failClosedReplay {
								return nil, rfc.New(rfc.ErrInvalidRequest, "signature replay record error")
							}
						} else if s.metrics != nil {
							s.metrics.IncReplayMisses()
						}
					}
				} else {
					if s.metrics != nil {
						s.metrics.IncSignatureIssueFailures()
					}
					if s.mandatorySignatures {
						return nil, rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("signature issuance failed: %v", serr))
					}
				}
			} else {
				if s.metrics != nil {
					s.metrics.IncSignatureIssueFailures()
				}
				if s.mandatorySignatures {
					return nil, rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("canonical digest failed: %v", derr))
				}
			}
		} else {
			if s.metrics != nil {
				s.metrics.IncSignatureIssueFailures()
			}
			if s.mandatorySignatures {
				return nil, rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("signer unavailable: %v", err))
			}
		}
	}
	if s.mandatorySignatures && poa.Signature == nil {
		// Covers case of absent signerProvider or unsuccessful attempt
		return nil, rfc.New(rfc.ErrIntegrityFailure, "signature required but not produced")
	}

	// Store POA via repository abstraction
	_ = s.repo.Create(poa)

	// Append issuance event to chain (tamper-evident provenance)
	if s.issChain != nil {
		_ = s.issChain.Append(IssuanceEvent{ID: fmt.Sprintf("iss_%s_%d", poa.ID, now.UnixNano()), DelegationID: poa.ID, Grantor: poa.Grantor, Grantee: poa.Grantee, Scope: poa.Scope, Restrictions: poa.Restrictions, IssuedAt: now})
		if s.anchorClient != nil { // external anchoring attempt
			if s.metrics != nil {
				s.metrics.IncAnchorAttempts()
			}
			if err := s.anchorClient.Anchor(chainTip(s.issChain)); err != nil {
				if s.metrics != nil {
					s.metrics.IncAnchorFailures()
				}
			}
		}
	}

	// Generate signed auth token (prototype PASETO)
	authToken := generateAuthToken(s, poa)

	// Audit log
	event := audit.NewEvent(audit.TypeAuth, "create_delegation", audit.ResultSuccess)
	event.Subject = req.Grantor
	event.Object = poa.ID
	event.Metadata = map[string]interface{}{
		"grantee":    req.Grantee,
		"scope":      req.Scope,
		"expires_at": poa.ValidUntil,
	}
	if err := s.audit.Log(ctx, event); err != nil {
		return nil, rfc.New(rfc.ErrInternal, fmt.Sprintf("audit log failed: %v", err))
	}
	// Send to external audit sink (P1.4)
	s.sendToAuditSink(ctx, event)
	// Ledger append (best-effort)
	if s.ledger != nil {
		_ = s.ledger.Append(ctx, &ledger.Entry{ID: fmt.Sprintf("led_iss_%s", poa.ID), TS: now, Type: "delegation_issuance", Subject: req.Grantor, Object: poa.ID, Metadata: map[string]interface{}{"grantee": req.Grantee, "scope": req.Scope, "valid_until": poa.ValidUntil}})
	}

	resp := &DelegationResponse{
		POA:       *poa,
		AuthToken: authToken,
		ExpiresAt: poa.ValidUntil,
		Warnings:  warnings,
	}
	// Metrics
	if s.metrics != nil {
		s.metrics.IncDelegationsCreated()
	}
	return resp, nil
}

// ValidateDelegation validates a power-of-attorney for a specific action
// ValidateDelegation is a backward-compatible wrapper using context.Background().
//
//nolint:gocyclo // Delegation validation with chain verification
func (s *Service) ValidateDelegation(poaID, grantee, action string) error {
	return s.ValidateDelegationCtx(context.Background(), poaID, grantee, action)
}

//nolint:gocyclo // Delegation validation with chain verification

// ValidateDelegationCtx validates a power-of-attorney for a specific action with context support.
func (s *Service) ValidateDelegationCtx(ctx context.Context, poaID, grantee, action string) error {
	start := s.nowFn()
	defer func() {
		if s.metrics != nil {
			s.metrics.ObserveValidationLatency(s.nowFn().Sub(start))
		}
	}()
	poa, exists := s.repo.Get(poaID)
	if !exists || poa == nil {
		return rfc.New(rfc.ErrNotFound, poaID)
	}
	// Hierarchical parent digest verification (Version>=4). Ensures recorded ParentDigest matches current parent canonical digest.
	if poa.Version >= 4 && poa.ParentPOAID != "" {
		parent, ok := s.repo.Get(poa.ParentPOAID)
		if !ok || parent == nil {
			if s.metrics != nil {
				s.metrics.IncHierDigestParentDigestMissing()
			}
			return rfc.New(rfc.ErrIntegrityFailure, "parent delegation missing for hierarchical digest")
		}
		if dig, _, derr := CanonicalPOADigest(parent); derr == nil {
			if poa.ParentDigest == "" || poa.ParentDigest != dig {
				if s.metrics != nil {
					s.metrics.IncHierDigestVersionMismatch()
				}
				return rfc.New(rfc.ErrIntegrityFailure, "parent digest mismatch")
			}
		} else {
			if s.metrics != nil {
				s.metrics.IncHierDigestParentDigestMissing()
			}
			return rfc.New(rfc.ErrIntegrityFailure, "parent canonicalization failed")
		}
	}

	// Signature verification (if signature present and provider available). Performed early
	// before revocation / status checks so integrity/authenticity failures are surfaced distinctly.
	if poa.Signature != nil {
		if dig, canon, derr := CanonicalPOADigest(poa); derr == nil {
			if dig != poa.Signature.DigestHex {
				if s.metrics != nil {
					var class string
					// Heuristic classification: same length implies domain conflict, otherwise potential tamper.
					if len(poa.Signature.DigestHex) == len(dig) {
						class = reasonDomainConflict
					} else {
						class = reasonTamperSuspected
					}
					s.metrics.IncEnvelopeDigestMismatch()
					s.metrics.IncEnvelopeDigestMismatchReason(class)
				}
				return rfc.New(rfc.ErrIntegrityFailure, errPOADigestMismatch)
			}
			// Attempt to locate public key by KeyID in keyRing.
			if s.keyRing != nil && poa.Signature.KeyID != "" {
				var pub []byte
				if ak := s.keyRing.Active(); ak != nil && ak.ID == poa.Signature.KeyID {
					pub = ak.Material
				}
				if len(pub) == 0 {
					for _, pk := range s.keyRing.Previous() {
						if pk != nil && pk.ID == poa.Signature.KeyID {
							pub = pk.Material
							break
						}
					}
				}
				if len(pub) == ed25519.PublicKeySize {
					sigBytes, decErr := base64.StdEncoding.DecodeString(poa.Signature.SigBase64)
					if decErr != nil {
						return rfc.New(rfc.ErrIntegrityFailure, "invalid signature encoding")
					}
					if len(sigBytes) != ed25519.SignatureSize {
						return rfc.New(rfc.ErrIntegrityFailure, "invalid signature size")
					}
					if !ed25519.Verify(ed25519.PublicKey(pub), canon, sigBytes) {
						if s.metrics != nil {
							s.metrics.IncSignatureVerificationFailures()
						}
						return rfc.New(rfc.ErrIntegrityFailure, "signature verification failed")
					} else {
						if s.metrics != nil {
							s.metrics.IncSignatureVerifications()
						}
					}
				} else { // key not found
					if s.metrics != nil {
						s.metrics.IncSignaturePublicKeyMissing()
					}
					if s.strictAuthenticity {
						return rfc.New(rfc.ErrIntegrityFailure, "signature public key missing (strict mode)")
					}
				}
			}
		} else {
			if s.metrics != nil {
				s.metrics.IncSignatureVerificationFailures()
				s.metrics.IncEnvelopeDigestMismatch()
				s.metrics.IncEnvelopeDigestMismatchReason(reasonCanonicalizationError)
			}
			return rfc.New(rfc.ErrIntegrityFailure, "canonicalization failed for signature verification")
		}
	}

	// Multi-signature threshold verification (performed after single signature, before revocation/status).
	if poa.Threshold > 1 {
		if err := s.verifyMultiSignatures(poa); err != nil {
			return err
		}
	}
	// Integrity & revocation chain check (tamper-evident log of revocations)
	if s.revChain != nil {
		if err := s.revChain.Verify(); err != nil {
			if s.metrics != nil {
				s.metrics.IncRevocationIntegrityFailures()
			}
			return rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("revocation chain integrity failure: %v", err))
		}
		if s.revChain.IsDelegationRevoked(poa.ID, "") { // we use ID reference for POA revocation events
			if poa.Status != POAStatusRevoked { // synchronize status if chain indicates revocation
				poa.Status = POAStatusRevoked
				poa.UpdatedAt = s.nowFn()
				_ = s.repo.Update(poa)
			}
			if s.metrics != nil {
				s.metrics.IncRevoked()
			}
			return rfc.New(rfc.ErrRevoked, "delegation revoked")
		}
	}

	// Check status
	if poa.Status != POAStatusActive {
		if poa.Status == POAStatusRevoked && s.metrics != nil {
			s.metrics.IncRevoked()
		}
		if poa.Status == POAStatusExpired && s.metrics != nil {
			s.metrics.IncExpired()
		}
		return rfc.New(rfc.ErrRevoked, fmt.Sprintf("delegation not active: %s", poa.Status))
	}

	// Clock skew tolerance (applies to ValidFrom future and ValidUntil past within skew window)
	now := s.nowFn()
	skew := s.clockSkew
	if skew == 0 {
		if raw := os.Getenv("GAUTH_CLOCK_SKEW_SECONDS"); raw != "" {
			if v, err := strconv.Atoi(raw); err == nil && v >= 0 && v <= 3600 {
				skew = time.Duration(v) * time.Second
				s.clockSkew = skew
			}
		}
	}
	// Not-before: reject only if outside tolerance window
	if now.Add(skew).Before(poa.ValidFrom) {
		return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("delegation not yet valid until %s", poa.ValidFrom.UTC().Format(time.RFC3339Nano)))
	}
	// Expiry: treat within skew window as still valid (soft grace); mark expired only beyond skew
	if now.After(poa.ValidUntil.Add(skew)) {
		poa.Status = POAStatusExpired
		poa.UpdatedAt = now
		_ = s.repo.Update(poa)
		if s.metrics != nil {
			s.metrics.IncExpired()
		}
		return rfc.New(rfc.ErrExpired, "delegation expired")
	}

	// Check grantee
	if poa.Grantee != grantee {
		if s.metrics != nil {
			s.metrics.IncUnauthorized()
		}
		return rfc.New(rfc.ErrUnauthorized, fmt.Sprintf("grantee mismatch expected %s got %s", poa.Grantee, grantee))
	}

	// Check scope
	if !containsScope(poa.Scope, action) {
		if s.metrics != nil {
			s.metrics.IncScopeViolations()
		}
		return rfc.New(rfc.ErrScopeViolation, action)
	}

	// Enforce simple numeric restriction: max_amount if action appears monetary (prototype)
	if limitStr, ok := poa.Restrictions["max_amount"]; ok {
		// For this prototype we only enforce on actions that start with "transaction:".
		if len(action) >= 12 && action[:12] == "transaction:" {
			// Parse decimal (no big.Float needed for prototype); ignore errors gracefully.
			var limit float64
			_, err := fmt.Sscan(limitStr, &limit)
			if err == nil && limit >= 0 {
				// In a real system, requested amount would come from context; simulate via metadata field.
				if requestedStr, ok2 := auditActionAmount(ctx); ok2 {
					var requested float64
					_, rErr := fmt.Sscan(requestedStr, &requested)
					if rErr == nil && requested > limit {
						if s.metrics != nil {
							s.metrics.IncRestrictionViolations()
						}
						return rfc.New(rfc.ErrRestrictionExceeded, fmt.Sprintf("max_amount %.2f exceeded by %.2f", limit, requested))
					}
				}
			}
		}
	}

	// Audit successful validation
	if err := ctx.Err(); err != nil { // Cancellation before logging
		return err
	}
	event := audit.NewEvent(audit.TypeAuth, "validate_delegation", audit.ResultSuccess)
	event.Subject = grantee
	event.Object = poaID
	if event.Metadata == nil {
		event.Metadata = map[string]interface{}{}
	}
	event.Metadata["action"] = action
	event.Metadata["grantor"] = poa.Grantor
	if err := s.audit.Log(ctx, event); err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("audit log failed: %v", err))
	}
	// Send to external audit sink (P1.4)
	s.sendToAuditSink(ctx, event)

	return nil
}

// InitiateRevocation starts a dual-control revocation workflow if enabled.
// It sets PendingRevocation with quorum parameters derived from env or defaults.
// Env:
//
//	GAUTH_REVOCATION_REQUIRED_COUNT (int) overrides RequiredCount
//	GAUTH_REVOCATION_REQUIRED_WEIGHT (int) enables weight mode when >0
//
// Authorization: only Grantor or Controllers may initiate.
func (s *Service) InitiateRevocation(ctx context.Context, req RevocationRequest) error {
	if req.POAID == "" {
		return rfc.New(rfc.ErrInvalidRequest, "poa_id required")
	}
	poa, ok := s.repo.Get(req.POAID)
	if !ok || poa == nil {
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowInitiationFailures()
		}
		return rfc.New(rfc.ErrNotFound, req.POAID)
	}
	if poa.Status != POAStatusActive && poa.Status != POAStatusSuspended {
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowInitiationFailures()
		}
		return rfc.New(rfc.ErrInvalidRequest, "revocation can only initiate from active or suspended")
	}
	// Auth check
	if req.Initiator != poa.Grantor && !identityIn(req.Initiator, poa.Controllers) {
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowUnauthorized()
		}
		return rfc.New(rfc.ErrUnauthorized, "initiator not authorized for dual-control revocation")
	}
	if poa.PendingRevocation != nil {
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowInitiationFailures()
		}
		return rfc.New(rfc.ErrInvalidRequest, "revocation already pending")
	}
	pr := &PendingRevocationState{InitiatedAt: s.nowFn(), Initiator: req.Initiator, Reason: req.Reason, EvidenceHashes: req.EvidenceHashes, Approvals: map[string]time.Time{}}
	if v := os.Getenv("GAUTH_REVOCATION_REQUIRED_COUNT"); v != "" {
		if iv, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && iv > 0 {
			pr.RequiredCount = iv
		}
	}
	if v := os.Getenv("GAUTH_REVOCATION_REQUIRED_WEIGHT"); v != "" {
		if iv, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && iv > 0 {
			pr.RequiredWeight = iv
		}
	}
	if pr.RequiredCount == 0 && pr.RequiredWeight == 0 {
		pr.RequiredCount = 2
	} // default dual-control (2 approvals)
	poa.PendingRevocation = pr
	poa.Status = POAStatusSuspended // place into suspended during pending revocation (prevents usage)
	poa.UpdatedAt = s.nowFn()
	_ = s.repo.Update(poa)
	//nolint:gocyclo // Revocation approval with state transitions
	if s.metrics != nil {
		s.metrics.IncRevocationWorkflowInitiated()
	}
	return nil
	//nolint:gocyclo // Revocation approval with state transitions
}

// ApproveRevocation records an approval and evaluates quorum satisfaction.
// Approvers: Grantor, Controllers, Signers (if multi-sig) – duplicates ignored.
func (s *Service) ApproveRevocation(ctx context.Context, poaID, approver string) error {
	if poaID == "" {
		return rfc.New(rfc.ErrInvalidRequest, "poa_id required")
	}
	poa, ok := s.repo.Get(poaID)
	if !ok || poa == nil {
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowApprovalFailures()
		}
		return rfc.New(rfc.ErrNotFound, poaID)
	}
	if poa.PendingRevocation == nil {
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowApprovalFailures()
		}
		return rfc.New(rfc.ErrInvalidRequest, "no pending revocation")
	}
	if poa.PendingRevocation.Finalized {
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowApprovalFailures()
		}
		return rfc.New(rfc.ErrInvalidRequest, "revocation already finalized")
	}
	if poa.PendingRevocation.Canceled {
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowApprovalFailures()
		}
		return rfc.New(rfc.ErrInvalidRequest, "revocation canceled")
	}
	if approver != poa.Grantor && !identityIn(approver, poa.Controllers) && !identityIn(approver, poa.Signers) {
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowUnauthorized()
		}
		return rfc.New(rfc.ErrUnauthorized, "approver not authorized")
	}
	if poa.PendingRevocation.Approvals == nil {
		poa.PendingRevocation.Approvals = map[string]time.Time{}
	}
	if _, exists := poa.PendingRevocation.Approvals[approver]; !exists {
		poa.PendingRevocation.Approvals[approver] = s.nowFn()
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowApprovals()
		}
		// Weight accumulation if in weight mode
		if poa.PendingRevocation.RequiredWeight > 0 && len(poa.Weights) > 0 {
			if w, okW := poa.Weights[approver]; okW {
				poa.PendingRevocation.SatisfiedWeight += w
			}
		}
	}
	// Quorum evaluation
	if poa.PendingRevocation.RequiredWeight > 0 {
		if poa.PendingRevocation.SatisfiedWeight >= poa.PendingRevocation.RequiredWeight {
			if s.metrics != nil {
				s.metrics.IncRevocationWorkflowQuorumSatisfied()
			}
			return s.finalizeRevocation(poa)
		}
	} else if poa.PendingRevocation.RequiredCount > 0 {
		if len(poa.PendingRevocation.Approvals) >= poa.PendingRevocation.RequiredCount {
			if s.metrics != nil {
				s.metrics.IncRevocationWorkflowQuorumSatisfied()
			}
			return s.finalizeRevocation(poa)
		}
	}
	_ = s.repo.Update(poa)
	return nil
}

// CancelRevocation aborts a pending revocation returning POA to its prior status (active) if not finalized.
// Only Grantor or Controllers may cancel.
func (s *Service) CancelRevocation(ctx context.Context, poaID, actor string) error {
	poa, ok := s.repo.Get(poaID)
	if !ok || poa == nil {
		return rfc.New(rfc.ErrNotFound, poaID)
	}
	if poa.PendingRevocation == nil {
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowCancellationFailures()
		}
		return rfc.New(rfc.ErrInvalidRequest, "no pending revocation")
	}
	if poa.PendingRevocation.Finalized {
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowCancellationFailures()
		}
		return rfc.New(rfc.ErrInvalidRequest, "revocation finalized")
	}
	if actor != poa.Grantor && !identityIn(actor, poa.Controllers) {
		if s.metrics != nil {
			s.metrics.IncRevocationWorkflowUnauthorized()
		}
		return rfc.New(rfc.ErrUnauthorized, "cancel not authorized")
	}
	poa.PendingRevocation.Canceled = true
	poa.Status = POAStatusActive
	poa.UpdatedAt = s.nowFn()
	_ = s.repo.Update(poa)
	if s.metrics != nil {
		s.metrics.IncRevocationWorkflowCanceled()
	}
	return nil
}

// finalizeRevocation (internal) applies the revocation and records metadata.
func (s *Service) finalizeRevocation(poa *PowerOfAttorney) error {
	now := s.nowFn()
	poa.PendingRevocation.Finalized = true
	poa.Status = POAStatusRevoked
	poa.RevokedAt = &now
	if poa.PendingRevocation != nil && poa.PendingRevocation.Reason != "" {
		poa.RevocationReason = poa.PendingRevocation.Reason
	}
	poa.UpdatedAt = now
	_ = s.repo.Update(poa)
	return nil
}

// identityIn helper checks presence of id in slice (case-sensitive exact match).
func identityIn(id string, list []string) bool {
	for _, v := range list {
		//nolint:gocyclo // Rich delegation validation with context checks
		if v == id {
			return true
		}
	}
	return false
	//nolint:gocyclo // Rich delegation validation with context checks
}

// ValidateDelegationRich validates a POA using a structured ValidationContext.
// Falls back to context-based requested_amount extraction if RequestedAmount is nil.
func (s *Service) ValidateDelegationRich(ctx context.Context, poaID, grantee string, vctx ValidationContext) error {
	if vctx.Action == "" {
		return rfc.New(rfc.ErrInvalidRequest, "action required in validation context")
	}
	poa, exists := s.repo.Get(poaID)
	if !exists || poa == nil {
		return rfc.New(rfc.ErrNotFound, poaID)
	}
	// Revocation chain integrity
	if s.revChain != nil {
		if err := s.revChain.Verify(); err != nil {
			if s.metrics != nil {
				s.metrics.IncRevocationIntegrityFailures()
			}
			return rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("revocation chain integrity failure: %v", err))
		}
		if s.revChain.IsDelegationRevoked(poa.ID, "") {
			if poa.Status != POAStatusRevoked {
				poa.Status = POAStatusRevoked
				poa.UpdatedAt = s.nowFn()
				_ = s.repo.Update(poa)
			}
			if s.metrics != nil {
				s.metrics.IncRevoked()
			}
			return rfc.New(rfc.ErrRevoked, "delegation revoked")
		}
	}
	if poa.Status != POAStatusActive {
		if poa.Status == POAStatusRevoked && s.metrics != nil {
			s.metrics.IncRevoked()
		}
		if poa.Status == POAStatusExpired && s.metrics != nil {
			s.metrics.IncExpired()
		}
		return rfc.New(rfc.ErrRevoked, fmt.Sprintf("delegation not active: %s", poa.Status))
	}
	now2 := s.nowFn()
	skew2 := s.clockSkew
	if skew2 == 0 {
		if raw := os.Getenv("GAUTH_CLOCK_SKEW_SECONDS"); raw != "" {
			if v, err := strconv.Atoi(raw); err == nil && v >= 0 && v <= 3600 {
				skew2 = time.Duration(v) * time.Second
				s.clockSkew = skew2
			}
		}
	}
	if now2.Add(skew2).Before(poa.ValidFrom) {
		return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("delegation not yet valid until %s", poa.ValidFrom.UTC().Format(time.RFC3339Nano)))
	}
	if now2.After(poa.ValidUntil.Add(skew2)) {
		poa.Status = POAStatusExpired
		poa.UpdatedAt = now2
		_ = s.repo.Update(poa)
		if s.metrics != nil {
			s.metrics.IncExpired()
		}
		return rfc.New(rfc.ErrExpired, "delegation expired")
	}
	if poa.Grantee != grantee {
		if s.metrics != nil {
			s.metrics.IncUnauthorized()
		}
		return rfc.New(rfc.ErrUnauthorized, fmt.Sprintf("grantee mismatch expected %s got %s", poa.Grantee, grantee))
	}
	if !containsScope(poa.Scope, vctx.Action) {
		if s.metrics != nil {
			s.metrics.IncScopeViolations()
		}
		// semantic counter prototype
		s.semanticCounters.ScopeViolation++
		return rfc.New(rfc.ErrScopeViolation, vctx.Action)
	}
	// Restriction enforcement (structured amount first, then context fallback)
	if limitStr, ok := poa.Restrictions["max_amount"]; ok {
		if strings.HasPrefix(vctx.Action, "transaction:") {
			var limit float64
			if _, err := fmt.Sscan(limitStr, &limit); err == nil && limit >= 0 {
				var requested float64
				have := false
				if vctx.RequestedAmount != nil {
					requested = *vctx.RequestedAmount
					have = true
				} else if rs, ok2 := auditActionAmount(ctx); ok2 {
					if _, rErr := fmt.Sscan(rs, &requested); rErr == nil {
						have = true
					}
				}
				if have && requested > limit {
					if s.metrics != nil {
						s.metrics.IncRestrictionViolations()
					}
					// semantic counter prototype
					s.semanticCounters.AmountLimitExceeded++
					return rfc.New(rfc.ErrRestrictionExceeded, fmt.Sprintf("max_amount %.2f exceeded by %.2f", limit, requested))
				}
				// Daily cumulative limit enforcement (max_daily_amount)
				if dlStr, okDL := poa.Restrictions["max_daily_amount"]; okDL {
					var dailyLimit float64
					if _, dlErr := fmt.Sscan(dlStr, &dailyLimit); dlErr == nil && dailyLimit >= 0 && have {
						// Key: delegationID|YYYY-MM-DD
						dayKey := fmt.Sprintf("%s|%s", poa.ID, s.nowFn().UTC().Format("2006-01-02"))
						s.dailyAmountsMu.Lock()
						current := s.dailyAmounts[dayKey]
						newTotal := current + requested
						if newTotal > dailyLimit {
							s.dailyAmountsMu.Unlock()
							if s.metrics != nil {
								s.metrics.IncRestrictionViolations()
							}
							s.semanticCounters.DailyAmountLimitExceeded++
							return rfc.New(rfc.ErrRestrictionExceeded, fmt.Sprintf("max_daily_amount %.2f exceeded by cumulative %.2f", dailyLimit, newTotal))
						}
						s.dailyAmounts[dayKey] = newTotal
						s.dailyAmountsMu.Unlock()
					}
				}
			}
		}
	}
	// Currency mismatch enforcement for transaction actions
	if strings.HasPrefix(vctx.Action, "transaction:") {
		if expectedCur, ok := poa.Restrictions["currency"]; ok {
			if vctx.Metadata != nil {
				if providedCur, ok2 := vctx.Metadata["currency"]; ok2 && providedCur != expectedCur {
					// semantic counter increment
					s.semanticCounters.CurrencyMismatch++
					return rfc.New(rfc.ErrRestrictionExceeded, fmt.Sprintf(errCurrencyMismatchFmt, expectedCur, providedCur))
				}
			}
		}
	}
	// Generic restriction mismatches: if metadata provides a key present in restrictions and value differs
	if vctx.Metadata != nil {
		for rk, rv := range poa.Restrictions {
			// Skip special keys already handled
			if rk == "max_amount" || rk == "currency" {
				continue
			}
			if provided, ok := vctx.Metadata[rk]; ok && provided != rv {
				// semantic counter increment
				s.semanticCounters.RestrictionMismatch++
				return rfc.New(rfc.ErrRestrictionExceeded, fmt.Sprintf(errRestrictionMismatchFmt, rk, rv, provided))
			}
		}
	}
	// Audit success
	if err := ctx.Err(); err != nil {
		return err
	}
	event := audit.NewEvent(audit.TypeAuth, "validate_delegation", audit.ResultSuccess)
	event.Subject = grantee
	event.Object = poaID
	if event.Metadata == nil {
		event.Metadata = map[string]interface{}{}
	}
	event.Metadata["action"] = vctx.Action
	event.Metadata["grantor"] = poa.Grantor
	if vctx.RequestedAmount != nil {
		event.Metadata["requested_amount"] = *vctx.RequestedAmount
	}
	if err := s.audit.Log(ctx, event); err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("audit log failed: %v", err))
	}
	// Send to external audit sink (P1.4)
	s.sendToAuditSink(ctx, event)
	return nil
}

// validateInheritedScope enforces conservative subset semantics for hierarchical delegation.
// Rules:
//  1. Each child scope entry must be covered by at least one parent entry.
//  2. Coverage definitions:
//     a. Exact match (parent == child)
//     b. Parent wildcard suffix using '*' matches prefix (e.g. parent 'audit.*' covers 'audit.log.write')
//     c. Global wildcard '*' (if present in parent list) covers all child scopes
//  3. No broadening: child entry not covered => error.
//  4. Duplicate child entries rejected for determinism.
//  5. Empty child scope not allowed.
//
// Advanced patterns (regex/range) will be layered later; current function is intentionally strict.
func validateInheritedScope(parentScopes, childScopes []string) error {
	if len(childScopes) == 0 {
		return fmt.Errorf("child scope empty")
	}
	parentExact := make(map[string]struct{}, len(parentScopes))
	parentWild := make([]string, 0)
	parentRegex := make([]*regexp.Regexp, 0)
	hasGlobal := false
	enableAdvanced := os.Getenv("GAUTH_ENABLE_ADVANCED_SCOPE") == "1"
	for _, p := range parentScopes {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "*" {
			hasGlobal = true
			continue
		}
		if strings.HasSuffix(p, "*") {
			pref := strings.TrimSuffix(p, "*")
			parentWild = append(parentWild, pref)
			continue
		}
		// Advanced regex pattern: parent entries beginning with 're:' treat remainder as a Go regex.
		if enableAdvanced && strings.HasPrefix(p, "re:") {
			pattern := strings.TrimPrefix(p, "re:")
			if pattern == "" {
				return fmt.Errorf("empty regex pattern in parent scope")
			}
			rx, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid regex pattern '%s': %v", pattern, err)
			}
			parentRegex = append(parentRegex, rx)
			continue
		}
		parentExact[p] = struct{}{}
	}
	sort.Slice(parentWild, func(i, j int) bool { return len(parentWild[i]) > len(parentWild[j]) })
	seenChild := make(map[string]struct{}, len(childScopes))
	for _, c := range childScopes {
		c = strings.TrimSpace(c)
		if c == "" {
			return fmt.Errorf("child scope contains empty entry")
		}
		if _, dup := seenChild[c]; dup {
			return fmt.Errorf("duplicate child scope entry: %s", c)
		}
		seenChild[c] = struct{}{}
		if hasGlobal {
			continue
		}
		if _, ok := parentExact[c]; ok {
			continue
		}
		covered := false
		for _, pref := range parentWild {
			if strings.HasPrefix(c, pref) {
				covered = true
				break
			}
		}
		if !covered && enableAdvanced {
			for _, rx := range parentRegex {
				if rx.MatchString(c) {
					covered = true
					break
				}
			}
		}
		if !covered {
			return fmt.Errorf("child scope '%s' not covered by parent", c)
		}
	}
	return nil
}

// RevokeDelegation revokes a power-of-attorney delegation
// RevokeDelegation is a backward-compatible wrapper using context.Background().
func (s *Service) RevokeDelegation(poaID, revoker string) error {
	return s.RevokeDelegationCtx(context.Background(), poaID, revoker)
}

// RevokeDelegationCtx revokes a power-of-attorney with context support.
func (s *Service) RevokeDelegationCtx(ctx context.Context, poaID, revoker string) error {
	poa, exists := s.repo.Get(poaID)
	if !exists || poa == nil {
		return rfc.New(rfc.ErrNotFound, poaID)
	}

	// Only grantor can revoke
	if poa.Grantor != revoker {
		return rfc.New(rfc.ErrUnauthorized, fmt.Sprintf("only grantor can revoke: grantor=%s revoker=%s", poa.Grantor, revoker))
	}

	// Check authorization
	authReq := authz.Request{
		Subject:  revoker,
		Action:   "revoke_delegation",
		Resource: poaID,
		Context:  map[string]string{"grantee": poa.Grantee},
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	decision, err := s.authz.Authorize(ctx, authReq)
	if err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("authorization failed: %v", err))
	}

	if !decision.Allow {
		return rfc.New(rfc.ErrUnauthorized, decision.Reason)
	}

	// Revoke POA
	poa.Status = POAStatusRevoked
	poa.UpdatedAt = s.nowFn()
	_ = s.repo.Update(poa)
	// Append revocation event to chain (tamper-evident)
	if s.revChain != nil {
		if _, err := s.revChain.Append(delegation.RevocationEvent{ID: fmt.Sprintf("rev_%s_%d", poa.ID, s.nowFn().UnixNano()), DelegationID: poa.ID, Reason: string(delegation.RevocationReasonGrantorRevoked)}); err != nil {
			return rfc.New(rfc.ErrInternal, fmt.Sprintf("append revocation event: %v", err))
		}
		// External anchoring attempt of revocation chain tip (best-effort)
		if s.anchorClient != nil {
			if s.metrics != nil {
				s.metrics.IncAnchorAttempts()
			}
			if err := s.anchorClient.Anchor(chainTipRev(s.revChain)); err != nil {
				if s.metrics != nil {
					s.metrics.IncAnchorFailures()
				}
			}
		}
	}

	// Audit log
	if err := ctx.Err(); err != nil {
		return err
	}
	event := audit.NewEvent(audit.TypeAuth, "revoke_delegation", audit.ResultSuccess)
	event.Subject = revoker
	event.Object = poaID
	if event.Metadata == nil {
		event.Metadata = map[string]interface{}{}
	}
	event.Metadata["grantee"] = poa.Grantee
	event.Metadata["reason"] = string(delegation.RevocationReasonGrantorRevoked)
	if err := s.audit.Log(ctx, event); err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("audit log failed: %v", err))
	}
	// Send to external audit sink (P1.4)
	s.sendToAuditSink(ctx, event)

	// Ledger append (best-effort) for revocation
	if s.ledger != nil {
		_ = s.ledger.Append(ctx, &ledger.Entry{ID: fmt.Sprintf("led_rev_%s", poa.ID), TS: s.nowFn(), Type: "delegation_revocation", Subject: revoker, Object: poa.ID, Metadata: map[string]interface{}{"grantee": poa.Grantee, "reason": string(delegation.RevocationReasonGrantorRevoked)}})
	}

	return nil
}

// SuspendDelegation temporarily suspends an active delegation (can be resumed later).
// Only transitions active -> suspended are allowed. Returns error for invalid transitions.
func (s *Service) SuspendDelegation(ctx context.Context, poaID, actor, reason string) error {
	poa, exists := s.repo.Get(poaID)
	if !exists || poa == nil {
		return rfc.New(rfc.ErrNotFound, poaID)
	}

	// Only grantor can suspend
	if poa.Grantor != actor {
		return rfc.New(rfc.ErrUnauthorized, fmt.Sprintf("only grantor can suspend: grantor=%s actor=%s", poa.Grantor, actor))
	}

	// Check current status - only active can be suspended
	if poa.Status != POAStatusActive {
		return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("cannot suspend delegation in status %s (must be active)", poa.Status))
	}

	// Check authorization
	authReq := authz.Request{
		Subject:  actor,
		Action:   "suspend_delegation",
		Resource: poaID,
		Context:  map[string]string{"grantee": poa.Grantee},
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	decision, err := s.authz.Authorize(ctx, authReq)
	if err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("authorization failed: %v", err))
	}
	if !decision.Allow {
		return rfc.New(rfc.ErrUnauthorized, decision.Reason)
	}

	// Suspend POA
	poa.Status = POAStatusSuspended
	poa.UpdatedAt = s.nowFn()
	if err := s.repo.Update(poa); err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("update failed: %v", err))
	}

	// Audit log
	if err := ctx.Err(); err != nil {
		return err
	}
	event := audit.NewEvent(audit.TypeAuth, "suspend_delegation", audit.ResultSuccess)
	event.Subject = actor
	event.Object = poaID
	if event.Metadata == nil {
		event.Metadata = map[string]interface{}{}
	}
	event.Metadata["grantee"] = poa.Grantee
	event.Metadata["reason"] = reason
	event.Metadata["prev_status"] = string(POAStatusActive)
	if err := s.audit.Log(ctx, event); err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("audit log failed: %v", err))
	}
	s.sendToAuditSink(ctx, event)

	// Ledger append
	if s.ledger != nil {
		_ = s.ledger.Append(ctx, &ledger.Entry{
			ID:       fmt.Sprintf("led_suspend_%s", poa.ID),
			TS:       s.nowFn(),
			Type:     "delegation_suspension",
			Subject:  actor,
			Object:   poa.ID,
			Metadata: map[string]interface{}{"grantee": poa.Grantee, "reason": reason},
		})
	}

	return nil
}

// ResumeDelegation reactivates a suspended delegation.
// Only transitions suspended -> active are allowed. Returns error for invalid transitions.
func (s *Service) ResumeDelegation(ctx context.Context, poaID, actor string) error {
	poa, exists := s.repo.Get(poaID)
	if !exists || poa == nil {
		return rfc.New(rfc.ErrNotFound, poaID)
	}

	// Only grantor can resume
	if poa.Grantor != actor {
		return rfc.New(rfc.ErrUnauthorized, fmt.Sprintf("only grantor can resume: grantor=%s actor=%s", poa.Grantor, actor))
	}

	// Check current status - only suspended can be resumed
	if poa.Status != POAStatusSuspended {
		return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("cannot resume delegation in status %s (must be suspended)", poa.Status))
	}

	// Check authorization
	authReq := authz.Request{
		Subject:  actor,
		Action:   "resume_delegation",
		Resource: poaID,
		Context:  map[string]string{"grantee": poa.Grantee},
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	decision, err := s.authz.Authorize(ctx, authReq)
	if err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("authorization failed: %v", err))
	}
	if !decision.Allow {
		return rfc.New(rfc.ErrUnauthorized, decision.Reason)
	}

	// Resume POA
	poa.Status = POAStatusActive
	poa.UpdatedAt = s.nowFn()
	if err := s.repo.Update(poa); err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("update failed: %v", err))
	}

	// Audit log
	if err := ctx.Err(); err != nil {
		return err
	}
	event := audit.NewEvent(audit.TypeAuth, "resume_delegation", audit.ResultSuccess)
	event.Subject = actor
	event.Object = poaID
	if event.Metadata == nil {
		event.Metadata = map[string]interface{}{}
	}
	event.Metadata["grantee"] = poa.Grantee
	event.Metadata["prev_status"] = string(POAStatusSuspended)
	if err := s.audit.Log(ctx, event); err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("audit log failed: %v", err))
	}
	s.sendToAuditSink(ctx, event)

	// Ledger append
	if s.ledger != nil {
		_ = s.ledger.Append(ctx, &ledger.Entry{
			ID:       fmt.Sprintf("led_resume_%s", poa.ID),
			TS:       s.nowFn(),
			Type:     "delegation_resumption",
			Subject:  actor,
			Object:   poa.ID,
			Metadata: map[string]interface{}{"grantee": poa.Grantee},
		})
	}

	return nil
}

// ScopeUpdate records a scope reduction event for audit trail.
type ScopeUpdate struct {
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	PrevScope []string  `json:"prev_scope"`
	NewScope  []string  `json:"new_scope"`
	Reason    string    `json:"reason,omitempty"`
}

// UpdateDelegationScope performs partial revocation by reducing the scope of an active/suspended delegation.
// New scope must be a non-empty subset of current scope. Tracks history in ScopeHistory field.
func (s *Service) UpdateDelegationScope(ctx context.Context, poaID, actor string, newScope []string, reason string) error {
	poa, exists := s.repo.Get(poaID)
	if !exists || poa == nil {
		return rfc.New(rfc.ErrNotFound, poaID)
	}

	// Only grantor can update scope
	if poa.Grantor != actor {
		return rfc.New(rfc.ErrUnauthorized, fmt.Sprintf("only grantor can update scope: grantor=%s actor=%s", poa.Grantor, actor))
	}

	// Can only update scope for active or suspended delegations
	if poa.Status != POAStatusActive && poa.Status != POAStatusSuspended {
		return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("cannot update scope in status %s (must be active or suspended)", poa.Status))
	}

	// Validate new scope is non-empty
	if len(newScope) == 0 {
		return rfc.New(rfc.ErrInvalidRequest, "new scope cannot be empty (use revocation to remove all permissions)")
	}

	// Validate new scope is subset of current scope
	currentScopeSet := make(map[string]bool)
	for _, s := range poa.Scope {
		currentScopeSet[s] = true
	}
	for _, s := range newScope {
		if !currentScopeSet[s] {
			return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("new scope contains permission not in current scope: %s", s))
		}
	}

	// Check if scope actually changed
	if scopesEqual(poa.Scope, newScope) {
		return rfc.New(rfc.ErrInvalidRequest, "new scope is identical to current scope")
	}

	// Check authorization
	authReq := authz.Request{
		Subject:  actor,
		Action:   "update_delegation_scope",
		Resource: poaID,
		Context:  map[string]string{"grantee": poa.Grantee},
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	decision, err := s.authz.Authorize(ctx, authReq)
	if err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("authorization failed: %v", err))
	}
	if !decision.Allow {
		return rfc.New(rfc.ErrUnauthorized, decision.Reason)
	}

	// Record scope history (store in Restrictions map for backward compatibility)
	update := ScopeUpdate{
		Timestamp: s.nowFn(),
		Actor:     actor,
		PrevScope: poa.Scope,
		NewScope:  newScope,
		Reason:    reason,
	}
	if poa.Restrictions == nil {
		poa.Restrictions = make(map[string]string)
	}
	historyJSON, _ := json.Marshal([]ScopeUpdate{update})
	if existing, ok := poa.Restrictions["__scope_history"]; ok {
		var history []ScopeUpdate
		_ = json.Unmarshal([]byte(existing), &history)
		history = append(history, update)
		historyJSON, _ = json.Marshal(history)
	}
	poa.Restrictions["__scope_history"] = string(historyJSON)

	// Update scope
	poa.Scope = newScope
	poa.UpdatedAt = s.nowFn()
	if err := s.repo.Update(poa); err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("update failed: %v", err))
	}

	// Audit log
	if err := ctx.Err(); err != nil {
		return err
	}
	event := audit.NewEvent(audit.TypeAuth, "update_delegation_scope", audit.ResultSuccess)
	event.Subject = actor
	event.Object = poaID
	if event.Metadata == nil {
		event.Metadata = map[string]interface{}{}
	}
	event.Metadata["grantee"] = poa.Grantee
	event.Metadata["prev_scope"] = update.PrevScope
	event.Metadata["new_scope"] = newScope
	event.Metadata["reason"] = reason
	if err := s.audit.Log(ctx, event); err != nil {
		return rfc.New(rfc.ErrInternal, fmt.Sprintf("audit log failed: %v", err))
	}
	s.sendToAuditSink(ctx, event)

	// Ledger append
	if s.ledger != nil {
		_ = s.ledger.Append(ctx, &ledger.Entry{
			ID:      fmt.Sprintf("led_scope_%s", poa.ID),
			TS:      s.nowFn(),
			Type:    "delegation_scope_reduction",
			Subject: actor,
			Object:  poa.ID,
			Metadata: map[string]interface{}{
				"grantee":    poa.Grantee,
				"prev_scope": update.PrevScope,
				"new_scope":  newScope,
				"reason":     reason,
			},
		})
	}

	return nil
}

// scopesEqual checks if two scope slices contain the same elements (order-independent).
func scopesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	setA := make(map[string]bool)
	for _, s := range a {
		setA[s] = true
	}
	for _, s := range b {
		if !setA[s] {
			return false
		}
	}
	return true
}

// ErrCanceled is exposed for callers to compare error values (mirrors context.Canceled without dependency leakage in tests if wrapped).
var ErrCanceled = context.Canceled

// AuditEvents returns a snapshot of audit events recorded by the service (testing / diagnostics)
func (s *Service) AuditEvents() []audit.Event {
	evPtrs, _ := s.audit.Query(context.Background(), nil)
	out := make([]audit.Event, len(evPtrs))
	for i, ev := range evPtrs {
		out[i] = *ev
	}
	return out
}

// LedgerEntries returns ledger entries filtered by subject or object (testing/diagnostics helper).
// If both subject and object are empty or ledger not configured returns nil.
func (s *Service) LedgerEntries(subject, object string) ([]*ledger.Entry, error) {
	if s.ledger == nil {
		return nil, nil
	}
	ctx := context.Background()
	if subject != "" {
		return s.ledger.QueryBySubject(ctx, subject)
	}
	if object != "" {
		return s.ledger.QueryByObject(ctx, object)
	}
	return nil, nil
}

// SemanticSnapshot returns a copy of semantic rejection counters.
// These prototype counters will later be exported via HTTP/Prometheus/OTEL.
func (s *Service) SemanticSnapshot() map[string]uint64 {
	if s == nil {
		return map[string]uint64{}
	}
	return map[string]uint64{
		"amount_limit_exceeded":       s.semanticCounters.AmountLimitExceeded,
		"daily_amount_limit_exceeded": s.semanticCounters.DailyAmountLimitExceeded,
		"currency_mismatch":           s.semanticCounters.CurrencyMismatch,
		"scope_violation":             s.semanticCounters.ScopeViolation,
		"restriction_mismatch":        s.semanticCounters.RestrictionMismatch,
	}
}

// SetSemanticSnapshot restores semantic counters from a snapshot map. Missing keys are treated as zero.
// It overwrites current values (does not accumulate) and is intended for startup restore after loading persistence.
// Future enhancement: add validation (e.g., monotonicity guarantees or maximum allowed delta).
func (s *Service) SetSemanticSnapshot(snapshot map[string]uint64) {
	if s == nil || snapshot == nil {
		return
	}
	// Overwrite counters; absence implies zero.
	s.semanticCounters.AmountLimitExceeded = snapshot["amount_limit_exceeded"]
	s.semanticCounters.DailyAmountLimitExceeded = snapshot["daily_amount_limit_exceeded"]
	s.semanticCounters.CurrencyMismatch = snapshot[counterCurrencyMismatch]
	s.semanticCounters.ScopeViolation = snapshot["scope_violation"]
	s.semanticCounters.RestrictionMismatch = snapshot[counterRestrictionMismatch]
}

// GetDelegation retrieves a power-of-attorney by ID
func (s *Service) GetDelegation(poaID string) (*PowerOfAttorney, error) {
	poa, ok := s.repo.Get(poaID)
	if !ok || poa == nil {
		return nil, rfc.New(rfc.ErrNotFound, poaID)
	}
	copy := *poa
	return &copy, nil
}

// ListDelegations lists all delegations for a user (as grantor or grantee)
func (s *Service) ListDelegations(userID string) ([]*PowerOfAttorney, error) {
	list := s.repo.ListByPrincipal(userID)
	out := make([]*PowerOfAttorney, 0, len(list))
	for _, poa := range list {
		if poa == nil {
			continue
		}
		cp := *poa
		out = append(out, &cp)
	}
	return out, nil
}

// VerifyIntegrity performs combined integrity verification for the revocation chain.
// (Future: extend to delegation chains once POA issuance migrates to hash chain.)
func (s *Service) VerifyIntegrity() error {
	if s.revChain == nil {
		// No chain present; nothing to verify
		return nil
	}
	if err := s.revChain.Verify(); err != nil {
		if s.metrics != nil {
			s.metrics.IncRevocationIntegrityFailures()
		}
		return err
	}
	if s.issChain != nil {
		if err := s.issChain.Verify(); err != nil {
			return err
		}
	}
	return nil
}

type IssuanceEvent struct {
	ID           string            `json:"id"`
	DelegationID string            `json:"delegation_id"`
	Grantor      string            `json:"grantor"`
	Grantee      string            `json:"grantee"`
	Scope        []string          `json:"scope"`
	Restrictions map[string]string `json:"restrictions,omitempty"`
	IssuedAt     time.Time         `json:"issued_at"`
	PrevHash     string            `json:"prev_hash"`
	Hash         string            `json:"hash"`
	Index        int               `json:"index"`
}

// DelegationChain maintains a hash chain of issuance events.
type DelegationChain struct {
	events []IssuanceEvent
}

// NewDelegationChain constructs an empty DelegationChain.
func NewDelegationChain() *DelegationChain { return &DelegationChain{events: make([]IssuanceEvent, 0)} }

// Append adds an issuance event computing its hash.
func (dc *DelegationChain) Append(ev IssuanceEvent) error {
	idx := len(dc.events)
	var prev string
	if idx > 0 {
		prev = dc.events[idx-1].Hash
	}
	ev.Index = idx
	ev.PrevHash = prev
	hInput, _ := json.Marshal(struct {
		ID, DelegationID, Grantor, Grantee string
		Scope                              []string
		Restrictions                       map[string]string
		IssuedAt                           time.Time
		PrevHash                           string
		Index                              int
	}{ev.ID, ev.DelegationID, ev.Grantor, ev.Grantee, ev.Scope, ev.Restrictions, ev.IssuedAt.UTC(), prev, idx})
	ev.Hash = computeHash(hInput)
	dc.events = append(dc.events, ev)
	return nil
}

// Verify checks the hash chain integrity.
func (dc *DelegationChain) Verify() error {
	for i, ev := range dc.events {
		var prev string
		if i > 0 {
			prev = dc.events[i-1].Hash
		}
		hInput, _ := json.Marshal(struct {
			ID, DelegationID, Grantor, Grantee string
			Scope                              []string
			Restrictions                       map[string]string
			IssuedAt                           time.Time
			PrevHash                           string
			Index                              int
		}{ev.ID, ev.DelegationID, ev.Grantor, ev.Grantee, ev.Scope, ev.Restrictions, ev.IssuedAt.UTC(), prev, i})
		expect := computeHash(hInput)
		if ev.Hash != expect || ev.PrevHash != prev || ev.Index != i {
			return fmt.Errorf("delegation chain integrity failure at index %d", i)
		}
		//nolint:gocyclo // Request validation with business rules
	}
	return nil
}

// computeHash is a small helper (reuse revocation chain style) – SHA256 hex.
func computeHash(data []byte) string {
	//nolint:gocyclo // Request validation with business rules
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:])
}

// validateDelegationRequest validates a delegation request
func (s *Service) validateDelegationRequest(req DelegationRequest) error {
	if req.Grantor == "" {
		return fmt.Errorf("grantor is required")
	}
	if req.Grantee == "" {
		return fmt.Errorf("grantee is required")
	}
	if len(req.Scope) == 0 {
		return fmt.Errorf("scope is required")
	}
	// Basic string hygiene (UTF-8, printable, non-empty items)
	for i, sc := range req.Scope {
		if sc == "" {
			return fmt.Errorf("scope[%d] must be non-empty", i)
		}
		if !utf8.ValidString(sc) {
			if s.metrics != nil {
				s.metrics.IncViolation(observability.ScopeUTF8Invalid)
			}
			return fmt.Errorf("scope[%d] invalid utf8", i)
		}
		for _, r := range sc {
			if (r >= 0 && r < 0x20) || r == 0x7f {
				if s.metrics != nil {
					s.metrics.IncViolation(observability.ScopeControlChar)
				}
				return fmt.Errorf("scope[%d] contains control character", i)
			}
		}
	}
	for k, v := range req.Restrictions {
		if k == "" {
			return fmt.Errorf("restriction key must be non-empty")
		}
		if v == "" {
			return fmt.Errorf("restriction value for key '%s' must be non-empty", k)
		}
		if !utf8.ValidString(k) {
			if s.metrics != nil {
				s.metrics.IncViolation(observability.RestrictionUTF8Invalid)
			}
			return fmt.Errorf("restriction key '%s' invalid utf8", k)
		}
		if !utf8.ValidString(v) {
			if s.metrics != nil {
				s.metrics.IncViolation(observability.RestrictionUTF8Invalid)
			}
			return fmt.Errorf("restriction value for key '%s' invalid utf8", k)
		}
		for _, r := range k {
			if (r >= 0 && r < 0x20) || r == 0x7f {
				if s.metrics != nil {
					s.metrics.IncViolation(observability.RestrictionControlChar)
				}
				return fmt.Errorf("restriction key '%s' contains control character", k)
			}
		}
		for _, r := range v {
			if (r >= 0 && r < 0x20) || r == 0x7f {
				if s.metrics != nil {
					s.metrics.IncViolation(observability.RestrictionControlChar)
				}
				return fmt.Errorf("restriction value for key '%s' contains control character", k)
			}
		}
	}
	// Size limits (configurable)
	lim := s.limits
	if len(req.Scope) > lim.MaxScopeItems {
		return fmt.Errorf("scope item count exceeds %d", lim.MaxScopeItems)
	}
	for _, sc := range req.Scope {
		if len(sc) > lim.MaxScopeLen {
			return fmt.Errorf("scope item length exceeds %d", lim.MaxScopeLen)
		}
	}
	if len(req.Restrictions) > lim.MaxRestrictions {
		return fmt.Errorf("restriction count exceeds %d", lim.MaxRestrictions)
	}
	for k, v := range req.Restrictions {
		if len(k) > lim.MaxRestrictionKeyLen {
			return fmt.Errorf("restriction key length exceeds %d", lim.MaxRestrictionKeyLen)
		}
		if len(v) > lim.MaxRestrictionValLen {
			return fmt.Errorf("restriction value length exceeds %d", lim.MaxRestrictionValLen)
		}
	}
	if req.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if req.Duration > lim.MaxDuration {
		return fmt.Errorf("duration cannot exceed %s", lim.MaxDuration)
	}

	return nil
}

// validateInheritedScopeV2 ensures childScope does not broaden parentScope. Rules:
// Parent patterns supported (same semantics as containsScope):
//   - Exact: resource.action
//   - Global wildcard: * (allows any child scope)
//   - Prefix wildcard: prefix.* (allows any action that starts with prefix.)
//
// Child entries must each be authorized by at least one parent entry; rejection occurs on first broaden attempt.
// Regex and numeric range patterns are conservatively unsupported for inheritance (must match exactly or via wildcard);
// if present in parent they are treated as exact strings (no expansion) until advanced inheritance implemented.
func validateInheritedScopeV2(parentScope, childScope []string) error {
	if len(childScope) == 0 {
		return fmt.Errorf("child scope must be non-empty")
	}
	if len(parentScope) == 0 {
		return fmt.Errorf("parent scope empty")
	}
	// Fast path: global wildcard present in parent => allow all
	for _, p := range parentScope {
		if p == "*" {
			return nil
		}
	}
	// Normalize parent into classification buckets
	exact := map[string]struct{}{}
	prefixes := make([]string, 0)
	for _, p := range parentScope {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "*" {
			return nil
		}
		if strings.HasSuffix(p, ".*") {
			prefixes = append(prefixes, strings.TrimSuffix(p, ".*"))
			continue
		}
		// treat regex:/ and range patterns conservatively as exact strings (no expansion)
		exact[p] = struct{}{}
	}
	// For each child entry ensure coverage by exact match or prefix wildcard.
	for _, c := range childScope {
		c = strings.TrimSpace(c)
		if c == "" {
			return fmt.Errorf("child scope contains empty entry")
		}
		if _, ok := exact[c]; ok {
			continue
		}
		covered := false
		for _, pre := range prefixes {
			if strings.HasPrefix(c, pre+".") {
				covered = true
				break
			}
		}
		if !covered {
			return fmt.Errorf("child scope entry '%s' not covered by parent scope", c)
		}
	}
	return nil
}

// containsScope checks if an action is within the permitted scope using enhanced pattern semantics.
// Supported patterns:
//  1. Exact: "resource.action" must equal the action string
//  2. Global wildcard: "*" matches any action
//  3. Prefix wildcard: "prefix.*" matches any action with leading segment(s) "prefix." (e.g. "audit.*" matches "audit.read.log")
//  4. Regex: "regex:/<expr>/" (only active when GAUTH_SCOPE_ALLOW_REGEX=1). On regex compile error pattern is skipped.
//  5. Numeric range: "base[min-max]" where action is "base<integer>" and integer lies inclusively within [min,max].
//     Example: pattern "rate[5-10]" matches action "rate7".
//
// Notes:
//   - Evaluation short-circuits on first match.
//   - Invalid patterns are ignored (never match) to avoid unintended broad authorization.
//   - Regex gating prevents accidental performance impact or security risk unless explicitly enabled.
func containsScope(scope []string, action string) bool {
	allowRegex := os.Getenv("GAUTH_SCOPE_ALLOW_REGEX") == "1"
	for _, raw := range scope {
		if raw == "" { // skip empty entries
			continue
		}
		// 1. Global wildcard
		if raw == "*" {
			return true
		}
		// 2. Regex pattern: regex:/.../
		if strings.HasPrefix(raw, "regex:/") && strings.HasSuffix(raw, "/") {
			if !allowRegex {
				// Regex patterns ignored when not enabled
				continue
			}
			pattern := raw[len("regex:/") : len(raw)-1]
			re, err := regexp.Compile(pattern)
			if err != nil {
				continue // invalid regex silently ignored
			}
			if re.MatchString(action) {
				return true
			}
			continue
		}
		// 3. Prefix wildcard suffix "*" (but not solitary "*")
		if strings.HasSuffix(raw, "*") && raw != "*" {
			prefix := strings.TrimSuffix(raw, "*")
			if prefix == "" { // Edge case: "*" already handled; empty prefix after trim means malformed pattern
				continue
			}
			if strings.HasPrefix(action, prefix) {
				return true
			}
			// continue; maybe exact match below
		}
		// 4. Numeric range: base[min-max]
		if lb := strings.Index(raw, "["); lb != -1 && strings.HasSuffix(raw, "]") {
			base := raw[:lb]
			rng := raw[lb+1 : len(raw)-1]
			parts := strings.Split(rng, "-")
			if len(parts) == 2 {
				start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err1 == nil && err2 == nil && start <= end && strings.HasPrefix(action, base) {
					valStr := action[len(base):]
					if num, err3 := strconv.Atoi(valStr); err3 == nil {
						if num >= start && num <= end {
							return true
						}
					}
				}
			}
			// continue after range attempt
		}
		// 5. Exact match
		if raw == action {
			return true
		}
	}
	return false
}

// ScopeContains is an exported wrapper around containsScope for external validation & testing.
// It preserves internal pattern semantics while offering a stable public API.
func ScopeContains(scope []string, action string) bool { return containsScope(scope, action) }

// auditActionAmount extracts a simulated requested transaction amount from context (prototype).
// Future enhancement: pass a structured request carrying amount; for now always returns none.
func auditActionAmount(ctx context.Context) (string, bool) {
	// Typed key preferred; legacy string fallback retained for transitional callers.
	v := ctx.Value(ctxKeyRequestedAmount)
	if v == nil {
		v = ctx.Value(LegacyCtxRequestedAmount)
	}
	if v == nil {
		return "", false
	}
	if s, ok := v.(string); ok {
		//nolint:gocyclo // Auth token generation with capability encoding
		return s, true
	}
	return fmt.Sprintf("%v", v), true
}

// generatePOAID generates a unique ID for a power-of-attorney
// poaIDGenerator is a variable to allow deterministic override in tests (signature replay scenarios).
//
//nolint:gocyclo // Auth token generation with capability encoding
var poaIDGenerator = func() string { return fmt.Sprintf("poa_%d", time.Now().UnixNano()) }

func generatePOAID() string { return poaIDGenerator() }

// generateAuthToken generates an authorization token for a POA
func generateAuthToken(s *Service, poa *PowerOfAttorney) string {
	// Build structured envelope (versioned). Default V1; optional V2 when env flag set.
	now := s.nowFn().UTC()
	// Issuance cadence: compute seconds since last issuance if metrics memory tracks timestamp.
	// Removed obsolete atomic cadence placeholder block
	activeKey := s.tokenKey
	kid := "legacy"
	if s.keyRing != nil && s.keyRing.Active() != nil {
		activeKey = s.keyRing.Active().Material
		kid = s.keyRing.Active().ID
	}
	useV2 := os.Getenv("GAUTH_POA_ENVELOPE_V2") == "1"
	var plain []byte
	var err error
	if useV2 {
		// Attempt canonical digest (best-effort; fallback to empty digest on error without aborting issuance)
		var digest string
		var canonicalJSON []byte
		if dig, canon, derr := CanonicalPOADigest(poa); derr == nil {
			digest = dig
			canonicalJSON = canon
		}
		// Embedding controls: GAUTH_EMBED_FULL_POA=1 enables RawPOA population when canonical serialization available.
		// GAUTH_MAX_RAW_POA_BYTES bounds size (defaults to 8192). If exceeded we omit RawPOA and record a metric.
		embedEnabled := os.Getenv("GAUTH_EMBED_FULL_POA") == "1"
		maxRaw := 8192
		if limStr := os.Getenv("GAUTH_MAX_RAW_POA_BYTES"); limStr != "" {
			if v, err2 := strconv.Atoi(limStr); err2 == nil && v > 0 && v < 10_000_000 {
				maxRaw = v
			}
		}
		var rawPOA string
		var poaVersion string
		var rawPOAChain string
		if embedEnabled && len(canonicalJSON) > 0 {
			if len(canonicalJSON) <= maxRaw {
				// Canonical JSON is already deterministic minimal encoding; reuse directly.
				rawPOA = string(canonicalJSON)
				poaVersion = poaVersionV1
				if s.metrics != nil {
					s.metrics.IncEnvelopeRawPOAEmbedded()
				}
			} else {
				// Size exceeded; omit RawPOA.
				if s.metrics != nil {
					s.metrics.IncEnvelopeRawPOATooLarge()
				}
			}
		}
		// RawPOAChain embedding (prototype). Feature-gated independently (GAUTH_EMBED_RAW_POA_CHAIN=1).
		embedChain := os.Getenv("GAUTH_EMBED_RAW_POA_CHAIN") == "1"
		var rawPOAChainAlgo string
		if embedChain {
			// Hash algorithm negotiation (default sha256). Supported: sha256, blake2b256, sha3_256.
			algoName := strings.ToLower(os.Getenv("GAUTH_RAW_POA_CHAIN_HASH_ALGO"))
			var algo poaPkg.RawPOAHashAlg = poaPkg.RawPOAHashSHA256
			switch algoName {
			case "blake2b256":
				algo = poaPkg.RawPOAHashBLAKE2b256
			case "sha3_256":
				algo = poaPkg.RawPOAHashSHA3_256
			}
			// Build minimal chain snapshot (single item) representing this issuance.
			// Clamp timestamp to <=255 to satisfy minimal CBOR encoder integer encoding (supports <256 path).
			item := poaPkg.RawPOAItem{ID: poa.ID, Issuer: poa.Grantor, Subject: poa.Grantee, Timestamp: now.Unix() % 256, Algo: algEd25519}
			if poa.Signature != nil && poa.Signature.SigBase64 != "" {
				if sigBytes, decErr := base64.StdEncoding.DecodeString(poa.Signature.SigBase64); decErr == nil {
					item.Signature = sigBytes
				}
			}
			chainBytes, cErr := poaPkg.EncodeRawPOAChain([]poaPkg.RawPOAItem{item})
			if cErr == nil {
				if len(chainBytes) <= maxRaw {
					// Compute chain hash + algo via streaming decode (single item continuity trivial).
					if chainDec, decErr := poaPkg.DecodeRawPOAStreamWith(bytes.NewReader(chainBytes), poaPkg.DefaultStreamLimits, algo, false); decErr == nil {
						rawPOAChainAlgo = chainDec.HashAlgo.String()
					}
					rawPOAChain = base64.StdEncoding.EncodeToString(chainBytes)
					if s.metrics != nil {
						s.metrics.IncEnvelopeRawPOAEmbedded()
					}
				} else if s.metrics != nil {
					s.metrics.IncEnvelopeRawPOATooLarge()
				}
			}
		}
		env2 := token.EnvelopeV2{
			Version:             "gauth-rfc0111-env2",
			KeyID:               kid,
			DelegationID:        poa.ID,
			Grantor:             poa.Grantor,
			Grantee:             poa.Grantee,
			Scope:               poa.Scope,
			Restrictions:        poa.Restrictions,
			Status:              string(poa.Status),
			IssuedAt:            now,
			ExpiresAt:           poa.ValidUntil.UTC(),
			IssuanceChain:       chainTip(s.issChain),
			RevocationChain:     chainTipRev(s.revChain),
			JTI:                 uuid.NewString(),
			CanonicalDigest:     digest,
			SatisfiedWeight:     poa.SatisfiedWeight,
			SatisfiedSignatures: poa.SatisfiedSignatures,
			PoAVersion:          poaVersion,
			RawPOA:              rawPOA,
			RawPOAChain:         rawPOAChain,
			RawPOAChainAlgo:     rawPOAChainAlgo,
		}
		// Detached signature issuance (feature-gated). We sign the canonical JSON bytes (not the digest hex) so that
		// external verifiers can recompute the canonical representation and verify directly without relying on hash preimage.
		if os.Getenv("GAUTH_DETACHED_SIGNATURE") == "1" && len(canonicalJSON) > 0 && s.signerProvider != nil {
			if signer, sErr := s.signerProvider(); sErr == nil && signer != nil {
				if sig, sigErr := signer.Sign(canonicalJSON); sigErr == nil {
					env2.DetachedSignature = base64.StdEncoding.EncodeToString(sig)
					env2.DetachedSignatureAlg = signer.Algorithm()
					env2.DetachedSignatureKid = signer.KeyID()
					if s.metrics != nil {
						s.metrics.IncSignaturesIssued() // reuse existing counter
					}
				} else if s.metrics != nil {
					s.metrics.IncSignatureIssueFailures()
				}
			} else if sErr := os.Getenv("GAUTH_DETACHED_SIGNATURE"); sErr == "1" && s.metrics != nil { // signer unavailable
				s.metrics.IncSignatureIssueFailures()
			}
		}
		// Advanced claims integration (P2.10 sec1.item2): Populate AdvancedClaims with typ semantic enforcement
		// and claims set metadata. Feature-gated by GAUTH_ADVANCED_CLAIMS=1 for backward compatibility.
		if os.Getenv("GAUTH_ADVANCED_CLAIMS") == "1" {
			// Compute delegation chain length (best-effort; fallback to 1 if parent chain unavailable)
			chainLength := 1
			if poa.ParentPOAID != "" {
				// Count chain depth by traversing parent references
				depth := 1
				parentID := poa.ParentPOAID
				for depth < 100 { // Safety limit to prevent infinite loops
					if parentPOA, found := s.repo.Get(parentID); found && parentPOA != nil {
						depth++
						if parentPOA.ParentPOAID == "" {
							break
						}
						parentID = parentPOA.ParentPOAID
					} else {
						break
					}
				}
				chainLength = depth
			}
			// Determine token type based on delegation properties
			tokenType := "gauth.delegation" // Default typ for delegation tokens
			if len(poa.Scope) == 0 {
				tokenType = "gauth.token" // Generic token without specific delegated scopes
			}
			// Check if token represents capability-based access (heuristic: "cap:" scope prefix)
			for _, scope := range poa.Scope {
				if len(scope) > 4 && scope[:4] == "cap:" {
					tokenType = "gauth.capability"
					break
				}
			}
			// Build claims metadata with issuer trust level, delegation chain length, and policy version
			claimsMeta := &gauth.ClaimsMetadata{
				Version:      "1.0",             // Claims schema version
				Capabilities: poa.Scope,         // Supported capabilities = delegated scopes
				Source:       "rfc0111-service", // Claims source identifier
				Confidence:   1.0,               // Full confidence for directly issued tokens
			}
			// Populate AdvancedClaims with standard JWT claims + GAuth-specific metadata
			env2.AdvancedClaims = &gauth.AdvancedClaims{
				Subject:        poa.Grantee,
				Issuer:         poa.Grantor,
				Audience:       []string{poa.Grantee}, // Audience = grantee (token intended for grantee's use)
				ExpiresAt:      poa.ValidUntil.Unix(),
				IssuedAt:       now.Unix(),
				NotBefore:      now.Unix(),
				JWTID:          env2.JTI,
				Scope:          poa.Scope,
				TokenType:      tokenType, // typ semantic enforcement (gauth.delegation, gauth.token, gauth.capability)
				ClientID:       poa.ID,    // ClientID = delegation ID for traceability
				ClaimsMetadata: claimsMeta,
				Custom: map[string]interface{}{
					"delegation_chain_length": chainLength,
					"poa_version":             poaVersion,
					"canonical_digest":        digest,
				},
			}
		}
		plain, err = json.Marshal(env2)
		if s.metrics != nil {
			s.metrics.IncEnvelopeV2Issued()
		}
		// Best-effort adoption ratio update (works for both memory & prometheus implementations; memory uses stored counters).
		// Update adoption ratio gauge if memory metrics installed.
		if mem, ok := s.metrics.(*metrics.Memory); ok {
			v1 := mem.EnvelopeV1IssuedCount()
			v2 := mem.EnvelopeV2IssuedCount()
			if total := v1 + v2; total > 0 {
				s.metrics.SetEnvelopeV2AdoptionRatio(float64(v2) / float64(total))
			}
			// Cadence observation
			prev := mem.LastEnvelopeIssuanceUnix()
		//nolint:gosec // G115: Unix timestamp always positive, safe conversion
			cur := uint64(now.Unix())
			if prev != 0 && cur > prev {
				s.metrics.ObserveEnvelopeIssuanceCadence(float64(cur - prev))
			}
			mem.SetLastEnvelopeIssuanceUnix(cur)
		}
	} else {
		env := token.Envelope{
			Version:         "gauth-rfc0111-env1",
			KeyID:           kid,
			DelegationID:    poa.ID,
			Grantor:         poa.Grantor,
			Grantee:         poa.Grantee,
			Scope:           poa.Scope,
			Restrictions:    poa.Restrictions,
			Status:          string(poa.Status),
			IssuedAt:        now,
			ExpiresAt:       poa.ValidUntil.UTC(),
			IssuanceChain:   chainTip(s.issChain),
			RevocationChain: chainTipRev(s.revChain),
			JTI:             uuid.NewString(),
		}
		plain, err = json.Marshal(env)
		if s.metrics != nil {
			s.metrics.IncEnvelopeV1Issued()
		}
		if mem, ok := s.metrics.(*metrics.Memory); ok {
			v1 := mem.EnvelopeV1IssuedCount()
			v2 := mem.EnvelopeV2IssuedCount()
			if total := v1 + v2; total > 0 {
				s.metrics.SetEnvelopeV2AdoptionRatio(float64(v2) / float64(total))
			}
			prev := mem.LastEnvelopeIssuanceUnix()
		//nolint:gosec // G115: Unix timestamp always positive, safe conversion
			cur := uint64(now.Unix())
			if prev != 0 && cur > prev {
				s.metrics.ObserveEnvelopeIssuanceCadence(float64(cur - prev))
			}
			mem.SetLastEnvelopeIssuanceUnix(cur)
		}
	}
	// Cadence already recorded inside issuance branch above.
	if err != nil {
		return fmt.Sprintf("rfc0111_token_fallback_%s_%d", poa.ID, time.Now().UnixNano())
	}
	tok, err := paseto.NewV2().Encrypt(activeKey, plain, nil)
	if err != nil {
		return fmt.Sprintf("rfc0111_token_fallback_%s_%d", poa.ID, time.Now().UnixNano())
	}
	return tok
}

func chainTip(dc *DelegationChain) string {
	if dc == nil || len(dc.events) == 0 {
		return ""
	}
	return dc.events[len(dc.events)-1].Hash
}

func chainTipRev(rc *delegation.RevocationChain) string {
	if rc == nil {
		return ""
	}
	evs := rc.Events()
	if len(evs) == 0 {
		return ""
	}
	return evs[len(evs)-1].Hash
}

// decryptWithAnyKey attempts decryption using active then previous keys; falls back to legacy tokenKey.
// decryptWithAnyKey is retained for future token backward-compat validation paths.
// nolint:unused // referenced in forthcoming validation enhancements
func (s *Service) decryptWithAnyKey(token string, dest interface{}) error {
	v2 := paseto.NewV2()
	// Try active key
	if s.keyRing != nil && s.keyRing.Active() != nil {
		if err := v2.Decrypt(token, s.keyRing.Active().Material, dest, nil); err == nil {
			return nil
		}
		// Try previous keys
		for _, pk := range s.keyRing.Previous() {
			if pk == nil {
				continue
			}
			if err := v2.Decrypt(token, pk.Material, dest, nil); err == nil {
				return nil
			}
		}
	}
	// Legacy fallback
	if s.tokenKey != nil {
		if err := v2.Decrypt(token, s.tokenKey, dest, nil); err == nil {
			return nil
		}
	}
	return fmt.Errorf("unable to decrypt token with any known key")
}

// Demo demonstrates RFC 0111 power-of-attorney functionality
func Demo() error {
	fmt.Println("=== RFC 0111 Power-of-Attorney Demo ===")

	// Create service dependencies
	auditLogger := audit.NewMemoryLogger(nil) // Use nil for demo logger
	authorizer := authz.NewMemoryAuthorizer()

	// Create RFC 0111 service
	service := NewService(auditLogger, authorizer)

	// Demo delegation creation
	fmt.Println("\n📋 Step 1: Create Power-of-Attorney Delegation")
	delegationReq := DelegationRequest{
		Grantor: "alice@example.com",
		Grantee: "bob@example.com",
		Scope:   []string{"transaction:execute", "account:read"},
		Restrictions: map[string]string{
			"max_amount": "1000.00",
			"currency":   "USD",
		},
		Duration: 24 * time.Hour, // 1 day
	}

	delegation, err := service.CreateDelegation(delegationReq)
	if err != nil {
		return fmt.Errorf("failed to create delegation: %w", err)
	}

	fmt.Printf("   ✅ Delegation created: %s\n", delegation.POA.ID)
	fmt.Printf("   �� Grantor: %s\n", delegation.POA.Grantor)
	fmt.Printf("   👤 Grantee: %s\n", delegation.POA.Grantee)
	fmt.Printf("   📜 Scope: %v\n", delegation.POA.Scope)
	fmt.Printf("   ⏰ Expires: %s\n", delegation.ExpiresAt.Format(time.RFC3339))

	// Demo delegation validation
	fmt.Println("\n🔍 Step 2: Validate Delegation for Action")
	err = service.ValidateDelegation(delegation.POA.ID, "bob@example.com", "transaction:execute")
	if err != nil {
		return fmt.Errorf("delegation validation failed: %w", err)
	}
	fmt.Println("   ✅ Delegation validated successfully")

	// Demo invalid action
	fmt.Println("\n❌ Step 3: Test Invalid Action")
	err = service.ValidateDelegation(delegation.POA.ID, "bob@example.com", "admin:delete")
	if err != nil {
		fmt.Printf("   ✅ Correctly rejected invalid action: %s\n", err.Error())
	} else {
		return fmt.Errorf("should have rejected invalid action")
	}

	// Demo delegation listing
	fmt.Println("\n📋 Step 4: List Delegations")
	delegations, err := service.ListDelegations("alice@example.com")
	if err != nil {
		return fmt.Errorf("failed to list delegations: %w", err)
	}
	fmt.Printf("   📊 Found %d delegations for alice@example.com\n", len(delegations))

	// Demo delegation revocation
	fmt.Println("\n🚫 Step 5: Revoke Delegation")
	err = service.RevokeDelegation(delegation.POA.ID, "alice@example.com")
	if err != nil {
		return fmt.Errorf("failed to revoke delegation: %w", err)
	}
	fmt.Println("   ✅ Delegation revoked successfully")

	// Verify revocation
	err = service.ValidateDelegation(delegation.POA.ID, "bob@example.com", "transaction:execute")
	if err != nil {
		fmt.Printf("   ✅ Correctly rejected revoked delegation: %s\n", err.Error())
	} else {
		return fmt.Errorf("should have rejected revoked delegation")
	}

	fmt.Println("\n🎉 RFC 0111 Power-of-Attorney demo completed successfully!")
	return nil
}
