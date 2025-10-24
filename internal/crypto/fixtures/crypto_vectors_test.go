package fixtures

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	bls "github.com/herumi/bls-eth-go-binary/bls"
)

type vector struct {
    Alg         string `json:"alg"`
    MessageHex  string `json:"message_hex"`
    PublicHex   string `json:"public_hex"`
    SignatureHex string `json:"signature_hex"`
    Valid       bool   `json:"valid"`
    Note        string `json:"note,omitempty"`
}

// decodeDERSignature parses minimal DER (SEQUENCE len, INTEGER r, INTEGER s).
func decodeDERSignature(der []byte) (*big.Int, *big.Int, bool) {
    if len(der) < 8 || der[0] != 0x30 { return nil, nil, false }
    seqLen := int(der[1])
    if seqLen+2 != len(der) { return nil, nil, false }
    idx := 2
    if der[idx] != 0x02 { return nil, nil, false }
    rl := int(der[idx+1]); idx += 2
    if idx+rl > len(der) { return nil, nil, false }
    rBytes := der[idx:idx+rl]; idx += rl
    if idx >= len(der) || der[idx] != 0x02 { return nil, nil, false }
    sl := int(der[idx+1]); idx += 2
    if idx+sl != len(der) { return nil, nil, false }
    sBytes := der[idx:idx+sl]
    r := new(big.Int).SetBytes(rBytes)
    s := new(big.Int).SetBytes(sBytes)
    return r, s, true
}

func isHighS(s, n *big.Int) bool { half := new(big.Int).Rsh(n,1); return s.Cmp(half) == 1 }

func TestCryptoVectorsFixture(t *testing.T) {
    // Fixture resides in same directory as this test.
    path := "crypto_vectors.json"
    data, err := os.ReadFile(path)
    if err != nil { t.Fatalf("read fixture: %v", err) }
    var vecs []vector
    if err := json.Unmarshal(data, &vecs); err != nil { t.Fatalf("unmarshal: %v", err) }

    // Init BLS once.
    _ = bls.Init(bls.BLS12_381)

    for i, v := range vecs {
        msg, _ := hex.DecodeString(v.MessageHex)
        pubBytes, _ := hex.DecodeString(v.PublicHex)
        sigBytes, _ := hex.DecodeString(v.SignatureHex)
        switch v.Alg {
        case "Ed25519":
            if len(pubBytes) != ed25519.PublicKeySize { t.Fatalf("[%d] bad ed25519 pub size", i) }
            pk := ed25519.PublicKey(pubBytes)
            ok := ed25519.Verify(pk, msg, sigBytes)
            if ok != v.Valid {
                t.Errorf("[%d] ed25519 validity mismatch (expected %v note=%s)", i, v.Valid, v.Note)
            }
        case "ECDSA-P256":
            if len(pubBytes) == 0 || pubBytes[0] != 0x04 { t.Fatalf("[%d] ecdsa pub must be uncompressed SEC1", i) }
            curve := elliptic.P256()
            x := new(big.Int).SetBytes(pubBytes[1:33])
            y := new(big.Int).SetBytes(pubBytes[33:])
            pub := ecdsa.PublicKey{Curve: curve, X: x, Y: y}
            digest := sha256.Sum256(msg)
            r, s, okDer := decodeDERSignature(sigBytes)
            if !okDer {
                if v.Valid { t.Errorf("[%d] expected valid DER but parse failed", i) }
                continue
            }
            // Enforce low-S canonical rule: reject high-S.
            if isHighS(s, curve.Params().N) {
                if v.Valid {
                    t.Errorf("[%d] marked valid but high-S", i)
                }
                // Skip ecdsa.Verify which might accept high-S depending on lib; our policy rejects.
                continue
            }
            ok := ecdsa.Verify(&pub, digest[:], r, s)
            if ok != v.Valid {
                t.Errorf("[%d] ecdsa validity mismatch expected %v note=%s", i, v.Valid, v.Note)
            }
        case "BLS12-381":
            var pk bls.PublicKey
            if err := pk.Deserialize(pubBytes); err != nil {
                if v.Valid { t.Errorf("[%d] bls pub deserialize: %v", i, err) }
                continue
            }
            var sig bls.Sign
            if err := sig.Deserialize(sigBytes); err != nil {
                if v.Valid { t.Errorf("[%d] bls sig deserialize: %v", i, err) }
                continue
            }
            ok := sig.VerifyByte(&pk, msg)
            if ok != v.Valid {
                t.Errorf("[%d] bls validity mismatch expected %v note=%s", i, v.Valid, v.Note)
            }
        default:
            t.Errorf("[%d] unknown alg %s", i, v.Alg)
        }
    }
}
