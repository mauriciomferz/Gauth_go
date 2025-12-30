---
title: "Multi-Signature PoA Examples"
category: example
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# Multi-Signature Power of Attorney (PoA) - Beta Implementation

**GAP_MATRIX Reference:** `sec3.item3` - Joint/collective signature enforcement  
**AAP-002 Section:** Section B (Authorization Type)  
**Status:** ✅ **IMPLEMENTED** (Beta)

## Overview

This **beta implementation** provides M-of-N threshold signature collection and verification for Power of Attorney (PoA) delegations, enabling multi-party authorization workflows with cryptographic proof of collective approval.

### Key Features

- **M-of-N Threshold Verification**: Require M valid signatures from N authorized signers
- **Weighted Signatures**: Optional weighted voting (e.g., CEO=3 votes, CFO=2 votes)
- **Canonical Digest**: Tamper-proof signature verification using AAP-001 canonical digest
- **Concurrent Submission**: Thread-safe parallel signature collection
- **Lifecycle Management**: `pending` → `completed` → `active` state transitions
- **Expiration Control**: Configurable signature collection windows
- **Comprehensive Metrics**: 8 granular failure categorization counters
- **REST API (Beta)**: Endpoints for signature orchestration

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Multi-Signature System                    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────┐      ┌─────────────────────────┐      │
│  │  REST API Layer  │─────▶│  Signature Manager      │      │
│  │  (api.go)        │      │  (manager.go)           │      │
│  │                  │      │                         │      │
│  │ POST /sign       │      │ • InitiateCollection()  │      │
│  │ GET  /status     │      │ • SubmitSignature()     │      │
│  │ POST /activate   │      │ • GetStatus()           │      │
│  │ GET  /pending    │      │ • ActivatePoA()         │      │
│  └──────────────────┘      │ • GetSignatures()       │      │
│                            │ • RejectCollection()    │      │
│                            └────────────┬────────────┘      │
│                                         │                    │
│                            ┌────────────▼────────────┐      │
│                            │ Verification Provider   │      │
│                            │                         │      │
│                            │ • PublicKey()           │      │
│                            │ • VerifySignature()     │      │
│                            └─────────────────────────┘      │
│                                                               │
├─────────────────────────────────────────────────────────────┤
│                  Core Verification Engine                    │
│              (pkg/rfc0111/rfc0111.go)                        │
│                                                               │
│  verifyMultiSignatures() - 180+ lines                        │
│  • Threshold M-of-N enforcement                              │
│  • Weighted signature support (AGENTAUTH_MULTI_SIG_WEIGHTS)      │
│  • Canonical digest computation & verification               │
│  • Comprehensive error handling & metrics                    │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

```
1. PoA Creation
   ├─ Define threshold (e.g., 3-of-5)
   ├─ Specify required signers
   └─ Compute canonical digest
         │
         ▼
2. Signature Collection (parallel/async)
   ├─ Signer 1 signs digest → POST /api/v1/beta/poa/sign
   ├─ Signer 2 signs digest → POST /api/v1/beta/poa/sign
   ├─ Signer 3 signs digest → POST /api/v1/beta/poa/sign
   └─ ... until threshold met
         │
         ▼
3. Threshold Completion
   ├─ Status: pending → completed
   ├─ CompletedAt timestamp recorded
   └─ Ready for activation
         │
         ▼
4. PoA Activation
   ├─ POST /api/v1/beta/poa/:id/activate
   ├─ Status: completed → active
   └─ PoA now authorized for use
```

## API Reference

### POST /api/v1/beta/poa/sign

Submit a signature for multi-signature PoA.

**Request:**
```json
{
  "poa_id": "poa-board-approval-2025-001",
  "signer_id": "Alice Chen (CEO)",
  "key_id": "key-64d94d94b1ae9c3a",
  "signature": "SVP5LMnUT0kr82pTXtbIPQjR7Myg487i...",
  "metadata": {
    "ip_address": "192.168.1.1",
    "user_agent": "BoardMemberApp/1.0"
  }
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Signature accepted",
  "status": "completed",
  "threshold_met": true,
  "collected_count": 3,
  "required_count": 3
}
```

**Error Responses:**
- `400 Bad Request`: Invalid signature, unauthorized signer
- `401 Unauthorized`: Signature verification failed
- `404 Not Found`: PoA not found
- `409 Conflict`: Duplicate signature, already completed
- `410 Gone`: Collection window expired

### GET /api/v1/beta/poa/:id/multisig/status

Retrieve multi-signature collection status.

**Response (200 OK):**
```json
{
  "poa_id": "poa-board-approval-2025-001",
  "status": "completed",
  "threshold": 3,
  "required_signers": [
    "Alice Chen (CEO)",
    "Bob Smith (CFO)",
    "Carol Johnson (CTO)",
    "David Lee (COO)",
    "Eve Wilson (General Counsel)"
  ],
  "collected": [
    "Alice Chen (CEO)",
    "Bob Smith (CFO)",
    "Carol Johnson (CTO)"
  ],
  "remaining": [
    "David Lee (COO)",
    "Eve Wilson (General Counsel)"
  ],
  "signatures": {
    "Alice Chen (CEO)": {
      "signer_id": "Alice Chen (CEO)",
      "key_id": "key-64d94d94b1ae9c3a",
      "signature": "SVP5LMnUT0kr82pTXtbIPQjR7Myg487i...",
      "signed_at": "2025-10-23T21:36:23Z",
      "weight": 1,
      "ip_address": "192.168.1.1",
      "user_agent": "BoardMemberApp/1.0"
    }
  },
  "created_at": "2025-10-23T21:35:00Z",
  "completed_at": "2025-10-23T21:36:24Z",
  "expires_at": "2025-10-24T21:35:00Z",
  "use_weighted": false
}
```

### POST /api/v1/beta/poa/:id/activate

Activate a PoA after threshold completion.

**Response (200 OK):**
```json
{
  "success": true,
  "message": "PoA activated",
  "poa_id": "poa-board-approval-2025-001"
}
```

### GET /api/v1/beta/poa/multisig/pending

List all pending multi-signature collections.

**Response (200 OK):**
```json
{
  "pending": [
    {
      "poa_id": "poa-pending-1",
      "threshold": 2,
      "required_signers": ["alice", "bob"],
      "status": "pending",
      "created_at": "2025-10-23T20:00:00Z"
    }
  ],
  "count": 1
}
```

## Usage Examples

### Basic 3-of-5 Threshold

```go
package main

import (
    "context"
    "github.com/AgentAuth-Foundation/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/internal/multisig"
    "github.com/AgentAuth-Foundation/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/rfc0111"
)

func main() {
    // Create signature manager
    manager := multisig.NewSignatureManager(keyProvider)
    
    // Define PoA with 3-of-5 threshold
    poa := &rfc0111.PowerOfAttorney{
        ID:        "poa-001",
        Threshold: 3,
        Signers:   []string{"alice", "bob", "carol", "dave", "eve"},
    }
    
    // Initiate signature collection (24 hour expiration)
    ctx := context.Background()
    err := manager.InitiateCollection(ctx, poa, 24*time.Hour)
    
    // Submit signatures (async)
    manager.SubmitSignature(ctx, "poa-001", "alice", keyID1, sig1, metadata)
    manager.SubmitSignature(ctx, "poa-001", "bob", keyID2, sig2, metadata)
    manager.SubmitSignature(ctx, "poa-001", "carol", keyID3, sig3, metadata)
    
    // Check status
    status, _ := manager.GetStatus(ctx, "poa-001")
    // status.Status == multisig.StatusCompleted
    
    // Activate PoA
    manager.ActivatePoA(ctx, "poa-001")
}
```

### Weighted Signatures

Configure weighted voting via environment variable:

```bash
export AGENTAUTH_MULTI_SIG_WEIGHTS="CEO=5,CFO=3,CTO=2,COO=1"
```

```go
poa := &rfc0111.PowerOfAttorney{
    ID:        "poa-weighted",
    Threshold: 7,  // Require cumulative weight >= 7
    Signers:   []string{"CEO", "CFO", "CTO", "COO"},
}

// CEO (weight=5) + CFO (weight=3) = 8 >= 7 ✓
// CEO (weight=5) + CTO (weight=2) = 7 >= 7 ✓
// CFO (weight=3) + CTO (weight=2) = 5 < 7 ✗
```

### Complete Demo

See `examples/multi_signature_poa/main.go` for a full working example:

```bash
go run examples/multi_signature_poa/main.go
```

Output demonstrates:
- 5 board member keypair generation
- 3-of-5 threshold PoA creation
- Parallel signature collection
- Threshold completion detection
- PoA activation lifecycle

## Testing

### Run Test Suite

```bash
# All multi-signature tests
go test ./internal/multisig -v

# Specific test
go test ./internal/multisig -v -run=TestSignatureManager_SubmitSignature

# With coverage
go test ./internal/multisig -cover
```

### Test Coverage

The test suite (`internal/multisig/manager_test.go`) includes 10 comprehensive tests:

1. **TestSignatureManager_InitiateCollection**: Collection initialization, expiration setup
2. **TestSignatureManager_SubmitSignature**: Threshold completion (1/2 → 2/2)
3. **TestSignatureManager_DuplicateSignature**: Duplicate submission rejection
4. **TestSignatureManager_InvalidSignature**: Cryptographic verification
5. **TestSignatureManager_UnauthorizedSigner**: Authorization enforcement
6. **TestSignatureManager_ActivatePoA**: Activation workflow
7. **TestSignatureManager_Expiration**: Time-based expiration
8. **TestSignatureManager_GetSignatures**: AAP-001 format export
9. **TestSignatureManager_ListPending**: Pending collections query
10. **TestSignatureManager_RejectCollection**: Manual rejection

**Status:** ✅ All tests passing (100%)

### Core Verification Tests

The underlying verification engine has extensive test coverage:

```bash
# Multi-signature threshold tests
go test ./pkg/rfc0111 -v -run=MultiSignature

# Weighted signature tests
go test ./pkg/rfc0111 -v -run=Weighted

# Granular metrics tests
go test ./pkg/rfc0111 -v -run=Granular
```

## Metrics & Observability

### Prometheus Metrics

```prometheus
# Success counter
agentauth_rfc0111_multi_signature_verifications_total

# Failure counters (granular categorization)
agentauth_rfc0111_multi_signature_verification_failures_total
agentauth_rfc0111_multi_signature_structural_failures_total
agentauth_rfc0111_multi_signature_digest_failures_total
agentauth_rfc0111_multi_signature_public_key_missing_failures_total
agentauth_rfc0111_multi_signature_invalid_signature_failures_total
agentauth_rfc0111_multi_signature_threshold_failures_total
agentauth_rfc0111_multi_signature_weight_failures_total

# Latency histogram
agentauth_rfc0111_multi_signature_verification_latency_seconds
```

### Failure Taxonomy

- **Structural**: Malformed signer list, invalid weight map
- **Digest**: Canonical digest computation failure
- **Public Key Missing**: Key material not found
- **Invalid Signature**: Cryptographic verification failed
- **Threshold**: Valid signatures < required threshold
- **Weight**: Cumulative weight < required threshold

## Security Considerations

### Signature Replay Prevention

- **Canonical Digest Binding**: Digest includes `threshold` and `signers` to prevent downgrade attacks
- **Unique Signer Enforcement**: Each signer can submit exactly one signature
- **Nonce/JTI Integration**: PoA ID acts as unique identifier

### Expiration & Time Windows

```go
// Short collection window (high-security scenario)
manager.InitiateCollection(ctx, poa, 1*time.Hour)

// Standard window
manager.InitiateCollection(ctx, poa, 24*time.Hour)

// Extended window (regulatory approvals)
manager.InitiateCollection(ctx, poa, 7*24*time.Hour)
```

### Rate Limiting

**Note:** This is a beta implementation. Rate limiting should be implemented via external middleware:

```go
// Example middleware (not included in beta)
rateLimiter := NewRateLimiter(100, time.Minute) // 100 req/min
http.Handle("/api/v1/beta/poa/sign", rateLimiter.Wrap(api.HandleSign))
```

### Audit Trail

All signature submissions include optional metadata:

```go
metadata := map[string]string{
    "ip_address": clientIP,
    "user_agent": userAgent,
    "session_id": sessionID,
    "mfa_verified": "true",
}
manager.SubmitSignature(ctx, poaID, signerID, keyID, signature, metadata)
```

## Deployment Considerations (Beta)

### Environment Configuration

```bash
# Weighted voting (optional)
export AGENTAUTH_MULTI_SIG_WEIGHTS="signer1=weight1,signer2=weight2"

# Domain separation V2 (recommended)
export AGENTAUTH_MULTI_SIG_DOMAIN_V2=true
```

### Integration Example

```go
// Initialize with key provider (beta - basic example)
keyProvider := &HSMKeyProvider{
    endpoint: os.Getenv("HSM_ENDPOINT"),
    authToken: os.Getenv("HSM_AUTH_TOKEN"),
}

manager := multisig.NewSignatureManager(keyProvider)

// Create API with metrics
metricsProvider := metrics.NewPrometheusMetrics(opts)
api := multisig.NewAPI(manager, metricsProvider)

// Register handlers
http.HandleFunc("/api/v1/beta/poa/sign", api.HandleSign)
http.HandleFunc("/api/v1/beta/poa/multisig/status", api.HandleStatus)
http.HandleFunc("/api/v1/beta/poa/activate", api.HandleActivate)
http.HandleFunc("/api/v1/beta/poa/multisig/pending", api.HandleListPending)
```

### Known Limitations (Beta)

- **State Persistence**: Current implementation uses in-memory state; persistent storage needed for durability
- **Distributed Lock**: Single-instance design; multi-instance deployments require distributed coordination
- **Event Sourcing**: Consider event-driven architecture for enhanced auditability
- **Rate Limiting**: External middleware recommended for API protection
- **HSM Integration**: Key provider interface supports HSM but not fully integrated

## References

- **ADR**: `docs/ADR-multi-signature-threshold-enforcement.md`
- **Core Implementation**: `pkg/rfc0111/rfc0111.go` (lines 177-370)
- **Manager**: `internal/multisig/manager.go`
- **REST API**: `internal/multisig/api.go`
- **Tests**: `internal/multisig/manager_test.go`, `pkg/rfc0111/multi_signature_*.go`
- **GAP_MATRIX**: `artifacts/gap_matrix.csv` (sec3.item3: Implemented)

## AAP-002 Compliance

This **beta implementation** satisfies **AAP-002 Section B (Authorization Type)** requirements for joint/collective signature enforcement:

✅ M-of-N threshold policies  
✅ Canonical digest stability  
✅ Signature collection workflow  
✅ Verification with cryptographic proof  
✅ Status tracking and lifecycle management  
✅ Comprehensive audit trail  
✅ REST API (Beta)

**GAP_MATRIX Status:** sec3.item3 = **Implemented** (Beta) ✓
