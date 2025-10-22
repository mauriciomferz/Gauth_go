package testutil

import "testing"

// Benchmarks focus on hashing & canonicalization overhead. They are micro-level and
// meant to catch regressions if future normalization logic grows more complex.

func BenchmarkSHA256HexRawTransfer(b *testing.B) {
    raw := CapTransferIssueDelegationCreateV1
    for i := 0; i < b.N; i++ {
        _ = SHA256Hex(raw)
    }
}

func BenchmarkCanonicalizeRegistry(b *testing.B) {
    raw := CapTransferIssueDelegationCreateV1
    for i := 0; i < b.N; i++ {
        _ = CanonicalizeRegistry(raw)
    }
}

func BenchmarkCanonicalRegistryHash(b *testing.B) {
    raw := CapTransferIssueDelegationCreateV1
    for i := 0; i < b.N; i++ {
        _ = CanonicalRegistryHash(raw)
    }
}

func BenchmarkCanonicalizePolicyBundle(b *testing.B) {
    raw := PolicyBundleB2V1
    for i := 0; i < b.N; i++ {
        _ = CanonicalizePolicyBundle(raw)
    }
}

func BenchmarkCanonicalPolicyBundleHash(b *testing.B) {
    raw := PolicyBundleB2V1
    for i := 0; i < b.N; i++ {
        _ = CanonicalPolicyBundleHash(raw)
    }
}

// Additional policy bundle benchmarks covering permutation and semantic change scenarios.

func BenchmarkCanonicalizePolicyBundleMultiPermutation(b *testing.B) {
    raw := PolicyBundleMultiPerm1V1
    for i := 0; i < b.N; i++ {
        _ = CanonicalizePolicyBundle(raw)
    }
}

func BenchmarkCanonicalPolicyBundleHashMultiPermutation(b *testing.B) {
    raw := PolicyBundleMultiPerm1V1
    for i := 0; i < b.N; i++ {
        _ = CanonicalPolicyBundleHash(raw)
    }
}

func BenchmarkCanonicalizePolicyBundleMultiSemanticAdd(b *testing.B) {
    raw := PolicyBundleMultiPlusP3V1
    for i := 0; i < b.N; i++ {
        _ = CanonicalizePolicyBundle(raw)
    }
}

func BenchmarkCanonicalPolicyBundleHashMultiSemanticAdd(b *testing.B) {
    raw := PolicyBundleMultiPlusP3V1
    for i := 0; i < b.N; i++ {
        _ = CanonicalPolicyBundleHash(raw)
    }
}
