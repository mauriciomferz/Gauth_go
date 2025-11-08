package notary

import "crypto/ed25519"

// Rotation verification reason constants (subset not already defined in rotation.go):
const (
	reasonContinuityFailure = "continuity_failure"
	reasonKidNotFoundOld    = "kid_not_found_old"
	reasonKidNotFoundNew    = "kid_not_found_new"
)

// RotationVerificationResult represents verification outcome for a single rotation descriptor.
// It focuses on continuity (PrevRotationHash linkage) and dual signature validity.
// Fields:
//
//	Index           - ordinal position in sequence (0-based as encountered).
//	OldKeyID/NewKeyID - key identifiers from descriptor.
//	ContinuityOK    - whether PrevRotationHash matches hash of previous rotation receipt (if provided).
//	SignaturesOK    - both dual signatures verified.
//	Reason          - first failure reason (continuity_failure, kid_mismatch_old, kid_mismatch_new,
//	                  missing_old_signature, missing_new_signature, old_sig_invalid, new_sig_invalid,
//	                  descriptor_nil, serialization_error)
//	PrevHashExpected - expected previous rotation hash (empty if none or continuity cannot be evaluated).
//	PrevHashActual   - actual PrevRotationHash field value.
type RotationVerificationResult struct {
	Index            int    `json:"index"`
	OldKeyID         string `json:"old_key_id"`
	NewKeyID         string `json:"new_key_id"`
	ContinuityOK     bool   `json:"continuity_ok"`
	SignaturesOK     bool   `json:"signatures_ok"`
	Reason           string `json:"reason,omitempty"`
	PrevHashExpected string `json:"prev_hash_expected,omitempty"`
	PrevHashActual   string `json:"prev_hash_actual,omitempty"`
}

// RotationVerificationSummary aggregates results across all rotation descriptors.
// Fields:
//
//	Total          - number of rotation descriptors assessed.
//	AllContinuityOK - true if every descriptor with a prev hash field linked correctly.
//	AllSignaturesOK - true if all signature checks succeeded.
//	Failures       - count of descriptors with any failure.
//	Results        - slice of per-descriptor results (length == Total).
type RotationVerificationSummary struct {
	Total           int                          `json:"total"`
	AllContinuityOK bool                         `json:"all_continuity_ok"`
	AllSignaturesOK bool                         `json:"all_signatures_ok"`
	Failures        int                          `json:"failures"`
	Results         []RotationVerificationResult `json:"results"`
}

// VerifyAllRotations performs continuity + signature verification over provided rotation descriptors.
// prevHashes should be a slice of receipt hashes in chronological order corresponding 1:1 with descriptors.
// For continuity, we check that descriptor[i].PrevRotationHash == prevHashes[i-1] (if i>0 and previous hash non-empty).
// Signatures use provided old/new public keys per descriptor; caller supplies them via parallel slices oldPubs/newPubs.
// Length mismatch returns empty summary.
func VerifyAllRotations(descriptors []*KeyRotationDescriptor, receiptHashes []string, oldPubs []ed25519.PublicKey, newPubs []ed25519.PublicKey) RotationVerificationSummary {
	summary := RotationVerificationSummary{}
	if len(descriptors) == 0 || len(descriptors) != len(receiptHashes) || len(descriptors) != len(oldPubs) || len(descriptors) != len(newPubs) {
		return summary
	}
	summary.Total = len(descriptors)
	summary.AllContinuityOK = true
	summary.AllSignaturesOK = true
	for i, rd := range descriptors {
		res := RotationVerificationResult{Index: i}
		if rd == nil {
			res.Reason = "descriptor_nil"
			summary.Failures++
			summary.AllContinuityOK = false
			summary.AllSignaturesOK = false
			summary.Results = append(summary.Results, res)
			continue
		}
		res.OldKeyID = rd.OldKeyID
		res.NewKeyID = rd.NewKeyID
		// Continuity
		if i == 0 {
			// First rotation has no previous; treat continuity as true if PrevRotationHash empty.
			if rd.PrevRotationHash == "" {
				res.ContinuityOK = true
			} else {
				res.ContinuityOK = false
				res.Reason = reasonContinuityFailure
				res.PrevHashActual = rd.PrevRotationHash
			}
		} else {
			expected := receiptHashes[i-1]
			res.PrevHashExpected = expected
			res.PrevHashActual = rd.PrevRotationHash
			if expected != "" && rd.PrevRotationHash == expected {
				res.ContinuityOK = true
			} else {
				res.ContinuityOK = false
				if res.Reason == "" {
					res.Reason = reasonContinuityFailure
				}
			}
		}
		if !res.ContinuityOK {
			summary.AllContinuityOK = false
		}
		// Signatures
		switch {
		case len(oldPubs[i]) == 0:
			res.SignaturesOK = false
			if res.Reason == "" {
				res.Reason = reasonKidNotFoundOld
			}
			summary.AllSignaturesOK = false
		case len(newPubs[i]) == 0:
			res.SignaturesOK = false
			if res.Reason == "" {
				res.Reason = reasonKidNotFoundNew
			}
			summary.AllSignaturesOK = false
		default:
			validSigs, sigReason := VerifyRotationDescriptor(rd, oldPubs[i], newPubs[i])
			if validSigs {
				res.SignaturesOK = true
			} else {
				res.SignaturesOK = false
				if res.Reason == "" {
					res.Reason = sigReason
				}
				summary.AllSignaturesOK = false
			}
		}
		if !res.ContinuityOK || !res.SignaturesOK {
			summary.Failures++
		}
		summary.Results = append(summary.Results, res)
	}
	return summary
}
