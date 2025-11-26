package attest

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/crypto"
)

// Domain separation prefix identical to signing path for model limits attestation.
const AttestationDomainPrefix = "GAUTH_MODEL_LIMIT_ATTEST:"

// Attestation mirrors the JSON structure necessary for verification (subset of full endpoint).
type Attestation struct {
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
	DomainSignature string `json:"domain_signature,omitempty"`
	DomainPrefix    string `json:"domain_prefix,omitempty"`
	Signature       string `json:"signature"`
	SigKid          string `json:"sig_kid"`
	SigMode         string `json:"sig_mode"`
}

// ReplayStrategy abstracts nonce replay protection.
// Seen should return true if nonce was already observed (prior to recording this invocation).
// Record persists the nonce for future replay detection.
type ReplayStrategy interface {
	Seen(nonce string) bool
	Record(nonce string)
}

// VerificationResult summarizes outcome of verification.
type VerificationResult struct {
	Valid            bool
	Kid              string
	SigMode          string
	CombinedHash     string
	FailureCode      string // e.g. unknown_kid|signature_invalid|nonce_replay|signature_fields_missing
	ErrorCode        string // structured API code (attestation_unknown_kid ...)
	HTTPStatus       int    // recommended HTTP status (soft invalid signature returns 200)
	SoftInvalid      bool   // true when response should be 200 with valid=false (signature mismatch)
	LatencySeconds   float64
	ReplayDetected   bool
	NotarizationWarn bool // inconsistent notarization
}

// VerifyModelLimitsAttestation performs signature + nonce replay + basic notarization consistency verification.
// It mirrors legacy behavior while centralizing logic under pkg/attest for RB7.
// KeyFinder abstracts key lookup (allows tests to provide stub without full Manager).
type KeyFinder interface{ FindByID(id string) *crypto.Key }

func VerifyModelLimitsAttestation(att *Attestation, keyRegistry KeyFinder, replay ReplayStrategy, now time.Time) (VerificationResult, error) {
	start := time.Now()
	res := VerificationResult{HTTPStatus: 200}
	if att == nil {
		return VerificationResult{Valid: false, FailureCode: "invalid_json", ErrorCode: "attestation_invalid_json", HTTPStatus: 400}, errors.New("nil_attestation")
	}
	// Signature field presence & mode
	if att.Signature == "" || att.SigKid == "" || att.SigMode != sigModeEdDSA {
		return VerificationResult{Valid: false, FailureCode: "signature_fields_missing", ErrorCode: "attestation_signature_fields_missing", HTTPStatus: 400}, nil
	}
	if keyRegistry == nil {
		return VerificationResult{Valid: false, FailureCode: "key_registry_unavailable", ErrorCode: "attestation_key_registry_unavailable", HTTPStatus: 500}, nil
	}
	key := keyRegistry.FindByID(att.SigKid)
	if key == nil {
		return VerificationResult{Valid: false, FailureCode: "unknown_kid", ErrorCode: "attestation_unknown_kid", HTTPStatus: 404}, nil
	}
	// Reconstruct unsigned attestation (exclude signature fields)
	unsigned := struct {
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
	}{
		Success: att.Success, Configured: att.Configured, Reason: att.Reason, Nonce: att.Nonce, Snapshot: att.Snapshot, Audit: att.Audit, Anchor: att.Anchor, StrictUnknown: att.StrictUnknown, Surge: att.Surge, Notarization: att.Notarization,
	}
	raw, _ := json.Marshal(unsigned)
	msg := append([]byte(AttestationDomainPrefix), raw...)
	sigBytes, err := base64.RawStdEncoding.DecodeString(att.Signature)
	if err != nil {
		return VerificationResult{Valid: false, FailureCode: "signature_base64_invalid", ErrorCode: "attestation_signature_base64_invalid", HTTPStatus: 400}, nil
	}
	if len(key.Public) != ed25519.PublicKeySize || !ed25519.Verify(key.Public, msg, sigBytes) {
		// Soft invalid maintains HTTP 200 with valid=false
		return VerificationResult{Valid: false, Kid: att.SigKid, SigMode: att.SigMode, FailureCode: "signature_invalid", SoftInvalid: true}, nil
	}
	// Optional domain signature verification (if present). Failure is soft invalid with distinct failure code.
	if att.DomainSignature != "" {
		if att.DomainPrefix == "" { // prefix required when domain signature present
			return VerificationResult{Valid: false, Kid: att.SigKid, SigMode: att.SigMode, FailureCode: "domain_signature_prefix_missing", SoftInvalid: true}, nil
		}
		dsigBytes, derr := base64.RawStdEncoding.DecodeString(att.DomainSignature)
		if derr != nil {
			return VerificationResult{Valid: false, Kid: att.SigKid, SigMode: att.SigMode, FailureCode: "domain_signature_base64_invalid", SoftInvalid: true}, nil
		}
		dmsg := append([]byte(att.DomainPrefix), raw...)
		if len(key.Public) != ed25519.PublicKeySize || !ed25519.Verify(key.Public, dmsg, dsigBytes) {
			return VerificationResult{Valid: false, Kid: att.SigKid, SigMode: att.SigMode, FailureCode: "domain_signature_invalid", SoftInvalid: true}, nil
		}
	}
	// Nonce replay detection
	if replay != nil {
		if att.Nonce == "" {
			return VerificationResult{Valid: false, FailureCode: "nonce_missing", ErrorCode: "attestation_nonce_missing", HTTPStatus: 400}, nil
		}
		if replay.Seen(att.Nonce) {
			return VerificationResult{Valid: false, Kid: att.SigKid, SigMode: att.SigMode, FailureCode: "nonce_replay", ErrorCode: "attestation_nonce_replay", HTTPStatus: 409, ReplayDetected: true}, nil
		}
		// Record after check
		replay.Record(att.Nonce)
	}
	// Notarization consistency warning (treat unsuccessful receipt as error)
	if att.Notarization != nil && !att.Notarization.Success {
		return VerificationResult{Valid: false, FailureCode: "notarization_inconsistent", ErrorCode: "attestation_notarization_inconsistent", HTTPStatus: 422}, nil
	}
	// Combined hash triple
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
	res.Valid = true
	res.Kid = att.SigKid
	res.SigMode = att.SigMode
	res.CombinedHash = fmt.Sprintf("sha256:%x", ch[:])
	res.LatencySeconds = time.Since(start).Seconds()
	return res, nil
}
