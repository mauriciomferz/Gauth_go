package attest

// TrustAnchor represents an issuer's public key binding for attestation proofs.
// In future this can be extended with validity windows, certificate chains, or
// external transparency log references. For the current implementation we keep
// minimal fields necessary for strict issuer→(algo,keyID) verification.
type TrustAnchor struct {
    Issuer   string `json:"issuer"`
    Algorithm string `json:"alg"`
    KeyID    string `json:"kid"`
}

// TrustAnchorRegistry provides lookup for issuer bindings. It is a simple
// in-memory map; loading from file/env is handled externally (service setup).
type TrustAnchorRegistry struct {
    anchors map[string]TrustAnchor
}

// NewTrustAnchorRegistry constructs an empty registry.
func NewTrustAnchorRegistry() *TrustAnchorRegistry {
    return &TrustAnchorRegistry{anchors: make(map[string]TrustAnchor)}
}

// Add inserts or replaces a trust anchor for an issuer.
func (r *TrustAnchorRegistry) Add(a TrustAnchor) {
    if r == nil || a.Issuer == "" { return }
    if r.anchors == nil { r.anchors = make(map[string]TrustAnchor) }
    r.anchors[a.Issuer] = a
}

// Get returns the anchor for an issuer or false if not present.
func (r *TrustAnchorRegistry) Get(issuer string) (TrustAnchor, bool) {
    if r == nil || issuer == "" { return TrustAnchor{}, false }
    a, ok := r.anchors[issuer]
    return a, ok
}

// Size returns number of registered anchors.
func (r *TrustAnchorRegistry) Size() int {
    if r == nil { return 0 }
    return len(r.anchors)
}
