package auditor

import (
	poa "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
)

// VerifyPOA performs digest and signature threshold verification returning a summary map.
func VerifyPOA(doc *poa.ProofOfAuthorization) map[string]any {
	canon := poa.CanonicalDigest(doc)
	digestValid := (doc != nil && doc.Digest != "" && doc.Digest == canon)
	validSigs, thresholdMet, required := poa.VerifyMultiSig(doc)
	return map[string]any{
		"digest_valid":       digestValid,
		"provided_digest":    doc.Digest,
		"recomputed_digest":  canon,
		"signature_count":    len(doc.Signatures),
		"valid_signatures":   validSigs,
		"threshold_met":      thresholdMet,
		"threshold_required": required,
	}
}
