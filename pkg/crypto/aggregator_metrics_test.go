package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
)

func TestBLSSimpleAggregatorMetrics(t *testing.T) {
	m := imetrics.NewMemory()
	msg := []byte("agg-metrics")
	agg := NewBLSSimpleAggregatorWithMetrics(msg, m)
	// create 4 signatures
	var pubKeys [][]byte
	for i := 0; i < 4; i++ {
		k, err := GenerateBLSKey()
		if err != nil {
			t.Fatalf("gen bls: %v", err)
		}
		sig, err := BLSSign(k, msg)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		if err := agg.Add(k.Public.Serialize(), sig); err != nil {
			t.Fatalf("add: %v", err)
		}
		pubKeys = append(pubKeys, k.Public.Serialize())
	}
	aSig, err := agg.Aggregate()
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	if !agg.Verify(msg, aSig, pubKeys) {
		t.Fatalf("verify failed")
	}
	// Basic assertions
	snap := m.SnapshotEx()
	if snap.MultiSignatureAggregateLatencyCount == 0 {
		t.Fatalf("expected aggregate latency count >0")
	}
	if snap.MultiSignatureBatchSizeMax < 4 {
		t.Fatalf("expected batch size max >=4 got %d", snap.MultiSignatureBatchSizeMax)
	}
	if snap.MultiSignatureVerifications == 0 {
		t.Fatalf("expected multi-signature verifications >0")
	}
	if snap.MultiSignatureVerificationFailures != 0 {
		t.Fatalf("unexpected failures")
	}
}

func TestBatchVerifyInstrumentedEd25519(t *testing.T) {
	m := imetrics.NewMemory()
	// build 3 ed25519 keys
	var pubs []interface{}
	var msgs [][]byte
	var sigs [][]byte
	for i := 0; i < 3; i++ {
		pub, priv, err := ed25519Generate()
		if err != nil {
			t.Fatalf("gen: %v", err)
		}
		msg := []byte("msg-" + time.Now().String())
		sig := ed25519Sign(priv, msg)
		pubs = append(pubs, pub)
		msgs = append(msgs, msg)
		sigs = append(sigs, sig)
	}
	if !BatchVerifyEd25519Instrumented(m, pubs, msgs, sigs) {
		t.Fatalf("batch verify ed25519 failed")
	}
	snap := m.SnapshotEx()
	if snap.MultiSignatureVerifications == 0 {
		t.Fatalf("expected verifications >0")
	}
	if snap.MultiSignatureBatchSizeMax < 3 {
		t.Fatalf("expected batch size max >=3")
	}
}

// local helpers using existing ed25519 functions without importing extra packages repeatedly
func ed25519Generate() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	return pub, priv, err
}
func ed25519Sign(priv ed25519.PrivateKey, msg []byte) []byte { return ed25519.Sign(priv, msg) }
