package crypto

import "testing"

func TestBLSSimpleAggregator(t *testing.T) {
    msg := []byte("aggregate-msg")
    // Generate 3 keys & signatures
    var pubKeys [][]byte
    var sigs [][]byte
    for i := 0; i < 3; i++ {
        k, err := GenerateBLSKey()
        if err != nil { t.Fatalf("gen bls: %v", err) }
        sig, err := BLSSign(k, msg)
        if err != nil { t.Fatalf("sign: %v", err) }
        pubKeys = append(pubKeys, k.Public.Serialize())
        sigs = append(sigs, sig)
    }
    agg := NewBLSSimpleAggregator(msg)
    for i := range pubKeys {
        if err := agg.Add(pubKeys[i], sigs[i]); err != nil { t.Fatalf("add failed: %v", err) }
    }
    aggSig, err := agg.Aggregate()
    if err != nil { t.Fatalf("aggregate: %v", err) }
    if !agg.Verify(msg, aggSig, pubKeys) { t.Fatalf("aggregate verify failed") }
    // Negative: verify with missing pub key
    if agg.Verify(msg, aggSig, pubKeys[:2]) { t.Fatalf("verify should fail with subset pubkeys") }
}
