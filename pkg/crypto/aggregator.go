package crypto

// Aggregator provides multi-signature aggregation semantics. Phase 1 implements
// BLS aggregation over identical messages. Threshold / heterogeneous aggregation
// deferred to later phases.

import (
	"errors"
	"time"

	bls "github.com/herumi/bls-eth-go-binary/bls"
	imetrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
)

type Aggregator interface {
	Add(pubKey, sig []byte) error
	Aggregate() ([]byte, error)
	Verify(msg []byte, aggSig []byte, pubKeys [][]byte) bool
}

// BLSSimpleAggregator accumulates BLS signatures that individually verify over
// the same message. It does not prevent duplicate public keys (caller responsibility).
type BLSSimpleAggregator struct {
	message    []byte
	signatures [][]byte
	pubKeys    [][]byte
	m          imetrics.Metrics // optional
}

func NewBLSSimpleAggregator(message []byte) *BLSSimpleAggregator {
	return &BLSSimpleAggregator{message: append([]byte(nil), message...)}
}

func NewBLSSimpleAggregatorWithMetrics(message []byte, m imetrics.Metrics) *BLSSimpleAggregator {
	return &BLSSimpleAggregator{message: append([]byte(nil), message...), m: m}
}

func (a *BLSSimpleAggregator) Add(pubKey, sig []byte) error {
	if len(pubKey) == 0 || len(sig) == 0 {
		return errors.New("empty_inputs")
	}
	// Deserialize public key
	var pk bls.PublicKey
	if err := pk.Deserialize(pubKey); err != nil {
		return errors.New("invalid_pubkey")
	}
	// Verify individual signature before inclusion
	var s bls.Sign
	if err := s.Deserialize(sig); err != nil {
		return errors.New("invalid_signature")
	}
	if !s.VerifyByte(&pk, a.message) {
		return errors.New("sig_verify_fail")
	}
	a.pubKeys = append(a.pubKeys, append([]byte(nil), pubKey...))
	a.signatures = append(a.signatures, append([]byte(nil), sig...))
	return nil
}

func (a *BLSSimpleAggregator) Aggregate() ([]byte, error) {
	if len(a.signatures) == 0 {
		return nil, errors.New("no_signatures")
	}
	start := time.Now()
	agg, err := BLSAggregate(a.signatures)
	latency := time.Since(start)
	if a.m != nil {
		a.m.ObserveMultiSignatureAggregateLatency(latency)
		a.m.ObserveMultiSignatureBatchSize(len(a.signatures))
	}
	if err != nil {
		if a.m != nil {
			a.m.IncMultiSignatureVerificationFailures()
		}
		return nil, err
	}
	return agg, nil
}

func (a *BLSSimpleAggregator) Verify(msg []byte, aggSig []byte, pubKeys [][]byte) bool {
	if len(pubKeys) == 0 || len(aggSig) == 0 {
		return false
	}
	if len(msg) == 0 {
		return false
	}
	start := time.Now()
	// Build public key slice
	pks := make([]bls.PublicKey, 0, len(pubKeys))
	for _, raw := range pubKeys {
		var pk bls.PublicKey
		if err := pk.Deserialize(raw); err != nil {
			if a.m != nil {
				a.m.IncMultiSignaturePublicKeyMissing()
			}
			return false
		}
		pks = append(pks, pk)
	}
	ok := BLSAggregateVerify(pks, msg, aggSig)
	latency := time.Since(start)
	if a.m != nil {
		a.m.ObserveMultiSignatureVerificationLatency(latency)
		a.m.ObserveMultiSignatureBatchSize(len(pubKeys))
		if ok {
			a.m.IncMultiSignatureVerifications()
		} else {
			a.m.IncMultiSignatureVerificationFailures()
		}
	}
	return ok
}
