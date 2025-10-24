package crypto

// Instrumented batch verification wrappers adding multi-signature metrics.
// Existing naive batch functions remain unchanged for backwards compatibility.

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"math/big"
	"time"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"github.com/herumi/bls-eth-go-binary/bls"
)

// BatchVerifyEd25519Instrumented mirrors BatchVerifyEd25519 but records metrics.
func BatchVerifyEd25519Instrumented(m imetrics.Metrics, publicKeys []interface{}, messages [][]byte, signatures [][]byte) bool {
    start := time.Now()
    n := len(publicKeys)
    if n == 0 || len(messages) != n || len(signatures) != n {
        if m != nil { m.IncMultiSignatureVerificationFailures() }
        return false
    }
    for i := 0; i < n; i++ {
        pk, ok := publicKeys[i].(ed25519.PublicKey)
        if !ok || len(pk) != ed25519.PublicKeySize { if m != nil { m.IncMultiSignaturePublicKeyMissing() }; return false }
        if !ed25519.Verify(pk, messages[i], signatures[i]) { if m != nil { m.IncMultiSignatureInvalidSignatureFailures(); m.IncMultiSignatureVerificationFailures() }; return false }
    }
    latency := time.Since(start)
    if m != nil { m.ObserveMultiSignatureVerificationLatency(latency); m.ObserveMultiSignatureBatchSize(n); m.IncMultiSignatureVerifications() }
    return true
}

// BatchVerifyECDSAInstrumented mirrors BatchVerifyECDSA but records metrics.
func BatchVerifyECDSAInstrumented(m imetrics.Metrics, publicKeys []interface{}, messages [][]byte, signatures [][]byte) bool {
    start := time.Now()
    n := len(publicKeys)
    if n == 0 || len(messages) != n || len(signatures) != n { if m != nil { m.IncMultiSignatureVerificationFailures() }; return false }
    for i := 0; i < n; i++ {
        pub, ok := publicKeys[i].(*ecdsa.PublicKey)
        if !ok { if m != nil { m.IncMultiSignaturePublicKeyMissing() }; return false }
        sig := signatures[i]
        if len(sig) < 2 || len(sig)%2 != 0 { if m != nil { m.IncMultiSignatureInvalidSignatureFailures(); m.IncMultiSignatureVerificationFailures() }; return false }
        mid := len(sig)/2
        r := new(big.Int).SetBytes(sig[:mid])
        s := new(big.Int).SetBytes(sig[mid:])
        h := sha256.Sum256(messages[i])
        if !ecdsa.Verify(pub, h[:], r, s) { if m != nil { m.IncMultiSignatureInvalidSignatureFailures(); m.IncMultiSignatureVerificationFailures() }; return false }
    }
    latency := time.Since(start)
    if m != nil { m.ObserveMultiSignatureVerificationLatency(latency); m.ObserveMultiSignatureBatchSize(n); m.IncMultiSignatureVerifications() }
    return true
}

// BatchVerifyBLSInstrumented mirrors BatchVerifyBLS but records metrics.
func BatchVerifyBLSInstrumented(m imetrics.Metrics, publicKeys []interface{}, messages [][]byte, signatures [][]byte) bool {
    start := time.Now()
    n := len(publicKeys)
    if n == 0 || len(messages) != n || len(signatures) != n { if m != nil { m.IncMultiSignatureVerificationFailures() }; return false }
    for i := 0; i < n; i++ {
        pk, ok := publicKeys[i].(bls.PublicKey)
        if !ok { if m != nil { m.IncMultiSignaturePublicKeyMissing() }; return false }
        if !BLSVerify(&BLSKey{Public: pk}, messages[i], signatures[i]) { if m != nil { m.IncMultiSignatureInvalidSignatureFailures(); m.IncMultiSignatureVerificationFailures() }; return false }
    }
    latency := time.Since(start)
    if m != nil { m.ObserveMultiSignatureVerificationLatency(latency); m.ObserveMultiSignatureBatchSize(n); m.IncMultiSignatureVerifications() }
    return true
}
