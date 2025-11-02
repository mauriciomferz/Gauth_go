package replay

// ExternalReplayBackend defines the minimal contract for an external durable
// attestation nonce replay backend. Implementations MUST be concurrency-safe.
// Size may be approximate and can return 0 on error.
type ExternalReplayBackend interface {
    Seen(key string) (bool, error)   // returns true if key already recorded (replay)
    Record(key string) error         // records key with TTL semantics (implementation-defined)
    Size() (int, error)              // approximate number of stored keys
    Close() error                    // release resources (optional no-op)
}
