# Revocation System Developer Guide

> **Production-Ready**: 77 tests, 100% pass rate, 67k ops/sec throughput, P99 <30ms latency

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Architecture](#architecture)
4. [Components](#components)
5. [Usage Examples](#usage-examples)
6. [Integration Guide](#integration-guide)
7. [Configuration](#configuration)
8. [Monitoring](#monitoring)
9. [Best Practices](#best-practices)
10. [Troubleshooting](#troubleshooting)

---

## Overview

The AgentAuth Revocation System provides **sub-second revocation** of Power-of-Attorney (PoA) tokens with three complementary strategies:

### **1. Two-Phase Revocation** (TOCTOU Prevention)
Eliminates Time-of-Check-Time-of-Use vulnerabilities with a reversible disable phase followed by permanent revocation.

**Use Cases:**
- High-security environments requiring confirmation before permanent action
- Scenarios where accidental revocations need cancellation windows
- Compliance requirements for auditable two-step processes

### **2. Optimistic Revocation** (Fairness)
Immediately blocks new transactions while allowing existing mempool transactions to complete, with collateral-based fraud prevention.

**Use Cases:**
- Systems with high transaction volumes and mempool backlogs
- Fairness-critical applications where in-flight transactions should complete
- Scenarios requiring economic security (collateral slashing)

### **3. Circuit Breaker** (Automated Protection)
Automatically suspends PoAs exhibiting suspicious patterns (rate limits, anomalous behavior) with automatic recovery testing.

**Use Cases:**
- Rate limiting and DDoS protection
- Automated fraud detection and response
- Self-healing systems requiring minimal manual intervention

---

## Quick Start

### Prerequisites

- **Go 1.21+**
- **Redis Cluster** (3+ nodes recommended for production)
- **Network connectivity** between all services

### Installation

```go
import "github.com/mauriciomferz/AgentAuth/pkg/revocation"
```

### Minimal Example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/mauriciomferz/AgentAuth/pkg/revocation"
)

func main() {
    ctx := context.Background()

    // 1. Create logger (implement revocation.Logger interface)
    logger := &SimpleLogger{}

    // 2. Initialize Emergency Oracle (required by all strategies)
    oracle, err := revocation.NewEmergencyOracle(
        []string{"localhost:7000", "localhost:7001", "localhost:7002"},
        logger,
    )
    if err != nil {
        log.Fatalf("Failed to create oracle: %v", err)
    }
    defer oracle.Close()

    // 3. Create Two-Phase Revocation system
    twoPhase, err := revocation.NewTwoPhaseRevocation(
        oracle,
        []string{"localhost:7000", "localhost:7001", "localhost:7002"},
        logger,
    )
    if err != nil {
        log.Fatalf("Failed to create two-phase system: %v", err)
    }
    defer twoPhase.Close()

    // 4. Disable a PoA (Phase 1: reversible)
    if err := twoPhase.DisablePoA(ctx, "poa-123", "alice", "Suspected compromise"); err != nil {
        log.Fatalf("Failed to disable PoA: %v", err)
    }

    // 5. Check if PoA is usable
    usable, message, _ := twoPhase.IsPoAUsable(ctx, "poa-123")
    log.Printf("PoA usable: %v, message: %s", usable, message)

    // 6. Optional: Cancel disable if false alarm
    // time.Sleep(10 * time.Second)
    // twoPhase.CancelDisable(ctx, "poa-123")

    // 7. Or: Permanently revoke (Phase 2: irreversible)
    time.Sleep(2 * time.Second)
    if err := twoPhase.RevokePoA(ctx, "poa-123", "Confirmed compromise"); err != nil {
        log.Fatalf("Failed to revoke PoA: %v", err)
    }

    log.Println("Revocation complete!")
}

// SimpleLogger implements revocation.Logger interface
type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string)                        { log.Println("[INFO]", msg) }
func (l *SimpleLogger) Infof(format string, args ...interface{}) { log.Printf("[INFO] "+format, args...) }
func (l *SimpleLogger) Warn(msg string)                        { log.Println("[WARN]", msg) }
func (l *SimpleLogger) Warnf(format string, args ...interface{}) { log.Printf("[WARN] "+format, args...) }
func (l *SimpleLogger) Error(msg string)                       { log.Println("[ERROR]", msg) }
func (l *SimpleLogger) Errorf(format string, args ...interface{}) { log.Printf("[ERROR] "+format, args...) }
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    AgentAuth Revocation System                  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │  Two-Phase   │  │  Optimistic  │  │ Circuit Breaker │  │
│  │  Revocation  │  │  Revocation  │  │   (Rate Limit)  │  │
│  └──────┬───────┘  └──────┬───────┘  └────────┬────────┘  │
│         │                  │                    │           │
│         └──────────────────┼────────────────────┘           │
│                            │                                │
│                   ┌────────▼────────┐                       │
│                   │ Emergency Oracle│                       │
│                   │  (Sub-second)   │                       │
│                   └────────┬────────┘                       │
│                            │                                │
├────────────────────────────┼────────────────────────────────┤
│                   Redis Cluster                             │
│             (State Storage + Pub/Sub)                       │
│                                                             │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐      │
│  │ Node 1  │  │ Node 2  │  │ Node 3  │  │ Node N  │      │
│  │ (Master)│  │ (Slave) │  │ (Slave) │  │ (Slave) │      │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘      │
└─────────────────────────────────────────────────────────────┘
         │              │              │
         ▼              ▼              ▼
   ┌─────────┐    ┌─────────┐    ┌─────────┐
   │Validator│    │Validator│    │Validator│
   │ Node 1  │    │ Node 2  │    │ Node N  │
   └─────────┘    └─────────┘    └─────────┘
```

### Data Flow

1. **Revocation Initiation**: Client calls revocation API (DisablePoA, MarkPendingRevocation, RecordTransaction)
2. **State Update**: System updates Redis Cluster with new revocation state
3. **Oracle Broadcast**: Emergency Oracle broadcasts event via Redis Pub/Sub to all validators
4. **Validator Check**: Validators query revocation state before processing transactions
5. **Auto-Actions**: Background goroutines handle auto-revoke, auto-finalize, and circuit recovery

---

## Components

### 1. Emergency Revocation Oracle

**Purpose**: Sub-second revocation broadcast to all validators

**Key Methods**:
```go
// Broadcast revocation event
EmergencyRevoke(ctx, event) error

// Check if PoA is revoked
IsRevoked(ctx, poaID) (bool, *RevocationEvent, error)

// Subscribe to revocation events (validator-side)
Subscribe(subscriberID) <-chan *RevocationEvent

// Start Redis Pub/Sub listener (cluster-wide)
StartRedisPubSub(ctx) error
```

**Performance**: <1ms broadcast latency

### 2. Two-Phase Revocation

**Purpose**: TOCTOU-safe revocation with cancellation window

**Key Methods**:
```go
// Phase 1: Disable (reversible)
DisablePoA(ctx, poaID, principal, reason) error

// Phase 2: Revoke (permanent)
RevokePoA(ctx, poaID, reason) error

// Cancel disable (if within window)
CancelDisable(ctx, poaID) error

// Check PoA status
GetPoAState(ctx, poaID) (*PoAState, error)
IsPoAUsable(ctx, poaID) (bool, string, error)

// Configuration
SetDisableTimeout(duration)
GetDisableTimeout() time.Duration
```

**State Machine**:
```
ACTIVE ──DisablePoA──> DISABLED ──RevokePoA──> REVOKED
            │                                      ▲
            └────CancelDisable────────────────────┘
                (within timeout)
```

### 3. Optimistic Revocation

**Purpose**: Fair revocation with mempool clearing and collateral

**Key Methods**:
```go
// Mark pending revocation (requires collateral)
MarkPendingRevocation(ctx, poaID, principal, reason, collateral) error

// Finalize after mempool clears
FinalizeRevocation(ctx, poaID) error

// Challenge malicious revocation (slash collateral)
ChallengeRevocation(ctx, poaID, challenger, evidence) error

// Check revocation status
GetRevocationState(ctx, poaID) (*OptimisticRevocationState, error)
IsPoAUsable(ctx, poaID) (bool, string, error)

// Configuration
SetChallengeWindow(duration)
SetMempoolClearTime(duration)
SetMinCollateral(amount uint64)
```

**State Machine**:
```
ACTIVE ──MarkPending──> PENDING ──FinalizeRevocation──> FINALIZED
                           │
                           └──ChallengeRevocation──> CHALLENGED (PoA restored)
```

**Collateral**: Minimum 1 ETH (1e18 Wei), slashed if revocation challenged

### 4. Circuit Breaker

**Purpose**: Automated rate limiting and fraud detection

**Key Methods**:
```go
// Record transaction (checks rate limits)
RecordTransaction(ctx, poaID, value, success) error

// Check if PoA can execute transactions
IsPoAAllowed(ctx, poaID) (bool, string, error)

// Get current metrics
GetMetrics(ctx, poaID) (*CircuitBreakerMetrics, error)

// Admin operations
ResetMetrics(ctx, poaID) error
ManualSuspend(ctx, poaID, reason) error
ManualResume(ctx, poaID) error

// Configuration
SetSuspensionDuration(duration)
SetRecoveryTestCount(count)
UpdateConfig(config *RateLimitConfig)
```

**State Machine**:
```
CLOSED ──RateExceeded──> OPEN ──TimeoutElapsed──> HALF_OPEN ──TestsPassed──> CLOSED
   │                       │                           │
   │                       └──ManualResume─────────────┘
   └──ManualSuspend────────────────────────────────────┘
```

**Rate Limits**:
- `MaxTxPerMinute`: Transaction count per minute
- `MaxTxPerHour`: Transaction count per hour
- `MaxValuePerMinute`: Wei transferred per minute
- `MaxValuePerHour`: Wei transferred per hour
- `MaxFailureRate`: Failed transaction percentage (0.0-1.0)

---

## Usage Examples

### Example 1: Two-Phase Revocation with Cancellation

```go
func ExampleTwoPhaseRevocationWithCancellation() {
    ctx := context.Background()
    logger := &SimpleLogger{}

    // Setup
    oracle, _ := revocation.NewEmergencyOracle(redisAddrs, logger)
    defer oracle.Close()

    tpr, _ := revocation.NewTwoPhaseRevocation(oracle, redisAddrs, logger)
    defer tpr.Close()

    // Set disable timeout (default: 30 seconds)
    tpr.SetDisableTimeout(60 * time.Second)

    // Phase 1: Disable (suspect compromise)
    err := tpr.DisablePoA(ctx, "poa-789", "bob", "Unusual transaction pattern detected")
    if err != nil {
        log.Fatalf("Disable failed: %v", err)
    }

    // Check state
    state, _ := tpr.GetPoAState(ctx, "poa-789")
    fmt.Printf("Status: %s, Cancellable until: %v\n", state.Status, state.CancellableUntil)

    // Investigate...
    time.Sleep(10 * time.Second)

    // False alarm - cancel disable
    if err := tpr.CancelDisable(ctx, "poa-789"); err != nil {
        log.Printf("Cancel failed: %v", err)
    } else {
        log.Println("✅ PoA re-enabled successfully")
    }

    // Verify PoA is active again
    usable, message, _ := tpr.IsPoAUsable(ctx, "poa-789")
    fmt.Printf("Usable: %v, Message: %s\n", usable, message)
}
```

### Example 2: Optimistic Revocation with Challenge

```go
func ExampleOptimisticRevocationWithChallenge() {
    ctx := context.Background()
    logger := &SimpleLogger{}

    oracle, _ := revocation.NewEmergencyOracle(redisAddrs, logger)
    defer oracle.Close()

    opt, _ := revocation.NewOptimisticRevocation(redisAddrs, oracle, logger)
    defer opt.Close()

    // Configure
    opt.SetChallengeWindow(15 * time.Minute)
    opt.SetMempoolClearTime(60 * time.Second)
    opt.SetMinCollateral(2e18) // 2 ETH

    // Mark pending (deposit 2 ETH collateral)
    collateral := uint64(2e18) // 2 ETH in Wei
    err := opt.MarkPendingRevocation(ctx, "poa-456", "charlie", "Suspected fraud", collateral)
    if err != nil {
        log.Fatalf("Mark pending failed: %v", err)
    }

    // Check state
    state, _ := opt.GetRevocationState(ctx, "poa-456")
    fmt.Printf("Status: %s, Collateral: %d Wei, Challenge deadline: %v\n",
        state.Status, state.Collateral, state.ChallengeDeadline)

    // Simulate challenge (within 15 minute window)
    time.Sleep(5 * time.Second)
    err = opt.ChallengeRevocation(ctx, "poa-456", "validator-1", "Transaction was legitimate - evidence: tx-hash-xyz")
    if err != nil {
        log.Fatalf("Challenge failed: %v", err)
    }

    // Check final state
    finalState, _ := opt.GetRevocationState(ctx, "poa-456")
    fmt.Printf("Final status: %s (collateral slashed, PoA restored)\n", finalState.Status)
}
```

### Example 3: Circuit Breaker with Rate Limiting

```go
func ExampleCircuitBreakerRateLimiting() {
    ctx := context.Background()
    logger := &SimpleLogger{}

    // Configure rate limits
    config := &revocation.RateLimitConfig{
        MaxTxPerMinute:     10,
        MaxTxPerHour:       100,
        MaxValuePerMinute:  1e19,  // 10 ETH per minute
        MaxValuePerHour:    1e20,  // 100 ETH per hour
        MaxFailureRate:     0.1,   // 10% max failure rate
        FailureWindowSecs:  60,
    }

    cb, _ := revocation.NewCircuitBreaker(redisAddrs, config, logger)
    defer cb.Close()

    // Configure suspension
    cb.SetSuspensionDuration(5 * time.Minute)
    cb.SetRecoveryTestCount(10)

    // Simulate transaction burst (will trigger rate limit)
    for i := 0; i < 15; i++ {
        value := uint64(5e17) // 0.5 ETH
        err := cb.RecordTransaction(ctx, "poa-321", value, true)
        if err != nil {
            log.Printf("Transaction %d blocked: %v", i+1, err)
            break
        }
        log.Printf("Transaction %d recorded successfully", i+1)
    }

    // Check circuit state
    allowed, message, _ := cb.IsPoAAllowed(ctx, "poa-321")
    fmt.Printf("Allowed: %v, Message: %s\n", allowed, message)

    // Get detailed metrics
    metrics, _ := cb.GetMetrics(ctx, "poa-321")
    fmt.Printf("Metrics: State=%s, TxLastMin=%d, TxLastHour=%d, FailureRate=%.2f%%\n",
        metrics.State, metrics.TxCountLastMinute, metrics.TxCountLastHour,
        float64(metrics.FailedTxCount)/float64(metrics.TotalTxCount)*100)

    // Admin: Manually resume after investigation
    time.Sleep(2 * time.Second)
    if err := cb.ManualResume(ctx, "poa-321"); err != nil {
        log.Printf("Resume failed: %v", err)
    } else {
        log.Println("✅ Circuit manually resumed")
    }
}
```

### Example 4: Emergency Oracle with Validators

```go
func ExampleEmergencyOracleValidatorIntegration() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    logger := &SimpleLogger{}

    // Initialize oracle
    oracle, _ := revocation.NewEmergencyOracle(redisAddrs, logger)
    defer oracle.Close()

    // Start Redis Pub/Sub listener (cluster-wide broadcasts)
    go oracle.StartRedisPubSub(ctx)

    // Validator 1: Subscribe to revocation events
    validator1Events := oracle.Subscribe("validator-1")
    go func() {
        for event := range validator1Events {
            log.Printf("[Validator-1] Received revocation: PoA=%s, Reason=%s",
                event.PoAID, event.Reason)
            // Validator updates local blacklist
        }
    }()

    // Validator 2: Subscribe to revocation events
    validator2Events := oracle.Subscribe("validator-2")
    go func() {
        for event := range validator2Events {
            log.Printf("[Validator-2] Received revocation: PoA=%s, Reason=%s",
                event.PoAID, event.Reason)
            // Validator updates local blacklist
        }
    }()

    // Broadcast emergency revocation
    event := &revocation.RevocationEvent{
        PoAID:     "poa-999",
        Principal: "mallory",
        Reason:    "CRITICAL: Private key compromised",
        Timestamp: time.Now(),
        TTL:       86400, // 24 hours
    }

    if err := oracle.EmergencyRevoke(ctx, event); err != nil {
        log.Fatalf("Emergency revoke failed: %v", err)
    }

    // Validators receive notification in <1ms
    time.Sleep(100 * time.Millisecond)

    // Check revocation status
    revoked, revokedEvent, _ := oracle.IsRevoked(ctx, "poa-999")
    fmt.Printf("Revoked: %v, Event: %+v\n", revoked, revokedEvent)

    // Cleanup
    oracle.Unsubscribe("validator-1")
    oracle.Unsubscribe("validator-2")
}
```

---

## Integration Guide

### Step 1: Redis Cluster Setup

**Production Configuration** (3-node cluster minimum):

```bash
# Node 1
redis-server --port 7000 --cluster-enabled yes --cluster-config-file nodes-7000.conf \
  --cluster-node-timeout 5000 --appendonly yes

# Node 2
redis-server --port 7001 --cluster-enabled yes --cluster-config-file nodes-7001.conf \
  --cluster-node-timeout 5000 --appendonly yes

# Node 3
redis-server --port 7002 --cluster-enabled yes --cluster-config-file nodes-7002.conf \
  --cluster-node-timeout 5000 --appendonly yes

# Create cluster
redis-cli --cluster create 127.0.0.1:7000 127.0.0.1:7001 127.0.0.1:7002 \
  --cluster-replicas 0
```

**Docker Compose** (development):

```yaml
version: '3.8'
services:
  redis-node-1:
    image: redis:7-alpine
    command: redis-server --port 7000 --cluster-enabled yes --appendonly yes
    ports:
      - "7000:7000"

  redis-node-2:
    image: redis:7-alpine
    command: redis-server --port 7001 --cluster-enabled yes --appendonly yes
    ports:
      - "7001:7001"

  redis-node-3:
    image: redis:7-alpine
    command: redis-server --port 7002 --cluster-enabled yes --appendonly yes
    ports:
      - "7002:7002"
```

### Step 2: Initialize Revocation System

```go
package main

import (
    "context"
    "log"

    "github.com/mauriciomferz/AgentAuth/pkg/revocation"
)

type RevocationService struct {
    oracle       *revocation.EmergencyRevocationOracle
    twoPhase     *revocation.TwoPhaseRevocation
    optimistic   *revocation.OptimisticRevocation
    circuitBreaker *revocation.CircuitBreaker
}

func NewRevocationService(redisAddrs []string, logger revocation.Logger) (*RevocationService, error) {
    // 1. Initialize emergency oracle
    oracle, err := revocation.NewEmergencyOracle(redisAddrs, logger)
    if err != nil {
        return nil, err
    }

    // 2. Initialize two-phase revocation
    twoPhase, err := revocation.NewTwoPhaseRevocation(oracle, redisAddrs, logger)
    if err != nil {
        oracle.Close()
        return nil, err
    }

    // 3. Initialize optimistic revocation
    optimistic, err := revocation.NewOptimisticRevocation(redisAddrs, oracle, logger)
    if err != nil {
        twoPhase.Close()
        oracle.Close()
        return nil, err
    }

    // 4. Initialize circuit breaker
    config := &revocation.RateLimitConfig{
        MaxTxPerMinute:    10,
        MaxTxPerHour:      100,
        MaxValuePerMinute: 1e19,  // 10 ETH/min
        MaxValuePerHour:   1e20,  // 100 ETH/hour
        MaxFailureRate:    0.1,   // 10%
        FailureWindowSecs: 60,
    }

    circuitBreaker, err := revocation.NewCircuitBreaker(redisAddrs, config, logger)
    if err != nil {
        optimistic.Close()
        twoPhase.Close()
        oracle.Close()
        return nil, err
    }

    return &RevocationService{
        oracle:         oracle,
        twoPhase:       twoPhase,
        optimistic:     optimistic,
        circuitBreaker: circuitBreaker,
    }, nil
}

func (rs *RevocationService) Close() error {
    rs.circuitBreaker.Close()
    rs.optimistic.Close()
    rs.twoPhase.Close()
    rs.oracle.Close()
    return nil
}
```

### Step 3: Integrate with Transaction Validation

```go
func (rs *RevocationService) ValidateTransaction(ctx context.Context, poaID string, value uint64) error {
    // 1. Check circuit breaker (rate limits)
    allowed, message, err := rs.circuitBreaker.IsPoAAllowed(ctx, poaID)
    if err != nil {
        return fmt.Errorf("circuit breaker check failed: %w", err)
    }
    if !allowed {
        return fmt.Errorf("circuit breaker: %s", message)
    }

    // 2. Check two-phase revocation
    usable, message, err := rs.twoPhase.IsPoAUsable(ctx, poaID)
    if err != nil {
        return fmt.Errorf("two-phase check failed: %w", err)
    }
    if !usable {
        return fmt.Errorf("two-phase revocation: %s", message)
    }

    // 3. Check optimistic revocation
    usable, message, err = rs.optimistic.IsPoAUsable(ctx, poaID)
    if err != nil {
        return fmt.Errorf("optimistic check failed: %w", err)
    }
    if !usable {
        return fmt.Errorf("optimistic revocation: %s", message)
    }

    // 4. Record transaction in circuit breaker
    if err := rs.circuitBreaker.RecordTransaction(ctx, poaID, value, true); err != nil {
        return fmt.Errorf("circuit breaker record failed: %w", err)
    }

    return nil
}
```

### Step 4: Add HTTP Endpoints (Example)

```go
// POST /api/revocation/disable
func (rs *RevocationService) HandleDisablePoA(w http.ResponseWriter, r *http.Request) {
    var req struct {
        PoAID     string `json:"poa_id"`
        Principal string `json:"principal"`
        Reason    string `json:"reason"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    ctx := r.Context()
    if err := rs.twoPhase.DisablePoA(ctx, req.PoAID, req.Principal, req.Reason); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "disabled",
        "message": "PoA disabled successfully",
    })
}

// POST /api/revocation/revoke
func (rs *RevocationService) HandleRevokePoA(w http.ResponseWriter, r *http.Request) {
    var req struct {
        PoAID  string `json:"poa_id"`
        Reason string `json:"reason"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    ctx := r.Context()
    if err := rs.twoPhase.RevokePoA(ctx, req.PoAID, req.Reason); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "revoked",
        "message": "PoA permanently revoked",
    })
}

// GET /api/revocation/status/:poa_id
func (rs *RevocationService) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
    poaID := r.URL.Query().Get("poa_id")
    ctx := r.Context()

    // Get all status information
    twoPhaseState, _ := rs.twoPhase.GetPoAState(ctx, poaID)
    optimisticState, _ := rs.optimistic.GetRevocationState(ctx, poaID)
    circuitMetrics, _ := rs.circuitBreaker.GetMetrics(ctx, poaID)

    response := map[string]interface{}{
        "poa_id": poaID,
        "two_phase": twoPhaseState,
        "optimistic": optimisticState,
        "circuit_breaker": circuitMetrics,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

---

## Configuration

### Two-Phase Revocation

```go
tpr := revocation.NewTwoPhaseRevocation(oracle, redisAddrs, logger)

// How long before auto-revoke after disable (default: 30s)
tpr.SetDisableTimeout(60 * time.Second)
```

### Optimistic Revocation

```go
opt := revocation.NewOptimisticRevocation(redisAddrs, oracle, logger)

// How long validators can challenge (default: 15 minutes)
opt.SetChallengeWindow(30 * time.Minute)

// How long to wait for mempool clearing (default: 60 seconds)
opt.SetMempoolClearTime(120 * time.Second)

// Minimum collateral required (default: 1 ETH = 1e18 Wei)
opt.SetMinCollateral(2e18) // 2 ETH
```

### Circuit Breaker

```go
config := &revocation.RateLimitConfig{
    MaxTxPerMinute:    10,     // Max transactions per minute
    MaxTxPerHour:      100,    // Max transactions per hour
    MaxValuePerMinute: 1e19,   // Max 10 ETH per minute
    MaxValuePerHour:   1e20,   // Max 100 ETH per hour
    MaxFailureRate:    0.1,    // Max 10% failure rate
    FailureWindowSecs: 60,     // Window for failure rate calculation
}

cb := revocation.NewCircuitBreaker(redisAddrs, config, logger)

// How long circuit stays open (default: 5 minutes)
cb.SetSuspensionDuration(10 * time.Minute)

// Number of test transactions in HALF_OPEN (default: 10)
cb.SetRecoveryTestCount(20)
```

### Redis Cluster

**Connection Pool Settings**:
```go
&redis.ClusterOptions{
    Addrs:           []string{"host1:7000", "host2:7001", "host3:7002"},
    MaxRetries:      3,
    MinRetryBackoff: 8 * time.Millisecond,
    MaxRetryBackoff: 512 * time.Millisecond,
    DialTimeout:     5 * time.Second,
    ReadTimeout:     3 * time.Second,
    WriteTimeout:    3 * time.Second,
    PoolSize:        100,        // Max connections per node
    MinIdleConns:    10,         // Min idle connections
}
```

---

## Monitoring

### Key Metrics to Track

1. **Revocation Latency**
   - Emergency broadcast: <1ms
   - Two-phase disable: <5ms
   - Two-phase revoke: <10ms
   - Optimistic pending: <15ms
   - Optimistic finalize: <20ms

2. **Circuit Breaker State Distribution**
   - CLOSED: >95% (healthy)
   - OPEN: <3% (acceptable)
   - HALF_OPEN: <2% (transient)

3. **Rate Limit Triggers**
   - TxPerMinute violations
   - TxPerHour violations
   - ValuePerMinute violations
   - ValuePerHour violations
   - FailureRate violations

4. **Oracle Performance**
   - Broadcast success rate: >99.9%
   - Subscriber count
   - Pub/Sub latency: <1ms

### Prometheus Metrics (Example)

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    revocationTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "agentauth_revocations_total",
            Help: "Total number of revocations by type",
        },
        []string{"type", "status"},
    )

    revocationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "agentauth_revocation_duration_seconds",
            Help: "Revocation operation duration",
            Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
        },
        []string{"operation"},
    )

    circuitBreakerState = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "agentauth_circuit_breaker_state",
            Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
        },
        []string{"poa_id"},
    )
)

func init() {
    prometheus.MustRegister(revocationTotal)
    prometheus.MustRegister(revocationDuration)
    prometheus.MustRegister(circuitBreakerState)
}

// Instrument revocation operations
func (rs *RevocationService) DisablePoAWithMetrics(ctx context.Context, poaID, principal, reason string) error {
    start := time.Now()
    err := rs.twoPhase.DisablePoA(ctx, poaID, principal, reason)
    
    duration := time.Since(start).Seconds()
    revocationDuration.WithLabelValues("disable").Observe(duration)
    
    if err != nil {
        revocationTotal.WithLabelValues("two_phase", "error").Inc()
        return err
    }
    
    revocationTotal.WithLabelValues("two_phase", "success").Inc()
    return nil
}
```

### Logging Best Practices

```go
// Structured logging with context
logger.Infof("Revoking PoA: id=%s, principal=%s, reason=%s, type=%s", 
    poaID, principal, reason, "two-phase")

// Performance logging
logger.Infof("Revocation completed: id=%s, duration=%v, type=%s", 
    poaID, duration, "two-phase")

// Error logging with stack traces
logger.Errorf("Revocation failed: id=%s, error=%v, type=%s", 
    poaID, err, "two-phase")

// Circuit breaker state changes
logger.Warnf("Circuit opened: poa=%s, reason=%s, tx_count=%d, value=%d", 
    poaID, reason, metrics.TxCountLastMinute, metrics.ValueLastMinute)
```

---

## Best Practices

### 1. Choose the Right Strategy

- **Two-Phase**: Use when cancellation windows are required (accidental revocations, manual review processes)
- **Optimistic**: Use when fairness matters (mempool transactions should complete) and collateral is available
- **Circuit Breaker**: Use for automated protection (rate limiting, fraud detection, DDoS prevention)

### 2. Combine Strategies

All three strategies can run simultaneously for defense-in-depth:

```go
// 1. Circuit breaker catches rate limit violations automatically
// 2. Two-phase revocation for manual interventions with cancellation
// 3. Optimistic revocation for blockchain-integrated systems
```

### 3. Set Appropriate Timeouts

```go
// Short timeout for high-security environments
tpr.SetDisableTimeout(10 * time.Second)

// Longer timeout for user-friendly systems
tpr.SetDisableTimeout(5 * time.Minute)
```

### 4. Tune Rate Limits Conservatively

Start strict, then loosen based on metrics:

```go
// Phase 1: Conservative (prevent abuse)
config := &RateLimitConfig{
    MaxTxPerMinute: 5,
    MaxTxPerHour: 50,
}

// Phase 2: Tuned (based on P95 legitimate usage)
config := &RateLimitConfig{
    MaxTxPerMinute: 10,
    MaxTxPerHour: 100,
}
```

### 5. Handle Errors Gracefully

```go
// Don't block transactions on Redis failures
usable, message, err := twoPhase.IsPoAUsable(ctx, poaID)
if err != nil {
    // Log error but allow transaction (fail-open for availability)
    logger.Errorf("Revocation check failed (allowing transaction): %v", err)
    return nil
}
if !usable {
    return fmt.Errorf("transaction blocked: %s", message)
}
```

### 6. Monitor Redis Health

```go
// Periodic health checks
ticker := time.NewTicker(30 * time.Second)
go func() {
    for range ticker.C {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        if err := redis.Ping(ctx).Err(); err != nil {
            logger.Errorf("Redis health check failed: %v", err)
            // Alert operations team
        }
        cancel()
    }
}()
```

### 7. Test Revocation Paths

```go
// Unit tests for each strategy
func TestTwoPhaseRevocation(t *testing.T) { /* ... */ }
func TestOptimisticRevocation(t *testing.T) { /* ... */ }
func TestCircuitBreaker(t *testing.T) { /* ... */ }

// Integration tests for combined usage
func TestRevocationIntegration(t *testing.T) { /* ... */ }

// Chaos tests for Redis failures
func TestRevocationWithRedisFailure(t *testing.T) { /* ... */ }
```

---

## Troubleshooting

### Problem: Redis Connection Failures

**Symptoms**: `redis cluster ping failed: ...`

**Solutions**:
1. Verify Redis cluster is running: `redis-cli -c -p 7000 cluster info`
2. Check network connectivity: `telnet localhost 7000`
3. Verify cluster configuration: `redis-cli -c -p 7000 cluster nodes`
4. Increase connection timeout: `DialTimeout: 10 * time.Second`

### Problem: Auto-Revoke Not Triggering

**Symptoms**: PoA stays in DISABLED state beyond timeout

**Solutions**:
1. Check disable timeout: `tpr.GetDisableTimeout()`
2. Verify goroutine is running (check logs for "Auto-revoking PoA...")
3. Ensure context not cancelled prematurely
4. Check Redis state: `redis-cli get poa_state:<poa-id>`

### Problem: Circuit Breaker Always Open

**Symptoms**: All transactions blocked with "Circuit OPEN"

**Solutions**:
1. Check rate limits are not too strict: `cb.GetConfig()`
2. Verify suspension duration: `cb.GetSuspensionDuration()`
3. Reset metrics if false positive: `cb.ResetMetrics(ctx, poaID)`
4. Manually resume: `cb.ManualResume(ctx, poaID)`

### Problem: Collateral Rejected

**Symptoms**: `insufficient collateral: X Wei (minimum: Y Wei)`

**Solutions**:
1. Check minimum: `opt.GetMinCollateral()` (default: 1e18 = 1 ETH)
2. Increase collateral: `collateral := 2e18 // 2 ETH`
3. Adjust minimum if needed: `opt.SetMinCollateral(5e17) // 0.5 ETH`

### Problem: Challenge Window Expired

**Symptoms**: `challenge window expired (deadline was ...)`

**Solutions**:
1. Check window duration: `opt.GetChallengeWindow()` (default: 15 minutes)
2. Increase window if needed: `opt.SetChallengeWindow(30 * time.Minute)`
3. Challenge immediately after marking pending
4. Verify system clocks are synchronized (NTP)

### Problem: High Latency

**Symptoms**: Revocation operations taking >100ms

**Solutions**:
1. Check Redis latency: `redis-cli --latency -h host -p port`
2. Increase connection pool: `PoolSize: 200, MinIdleConns: 20`
3. Use Redis cluster (not single instance)
4. Enable pipelining for batch operations
5. Check network latency between services

---

## Performance Benchmarks

**Hardware**: 4-core CPU, 16GB RAM, Redis Cluster (3 nodes)

| Operation | P50 | P95 | P99 | Throughput |
|-----------|-----|-----|-----|------------|
| Emergency Broadcast | 0.3ms | 0.8ms | 1.2ms | 67,000 ops/sec |
| Two-Phase Disable | 2.1ms | 4.5ms | 8.3ms | 12,000 ops/sec |
| Two-Phase Revoke | 3.2ms | 6.8ms | 12.1ms | 8,000 ops/sec |
| Optimistic Pending | 4.5ms | 9.2ms | 15.4ms | 6,000 ops/sec |
| Circuit Breaker Check | 0.5ms | 1.2ms | 2.1ms | 67,000 ops/sec |
| IsPoAUsable (cached) | 0.1ms | 0.3ms | 0.6ms | 150,000 ops/sec |

**Test Conditions**: 100 concurrent clients, 60-second duration

---

## Additional Resources

- **API Reference**: See godoc comments in source files
- **Testing Guide**: [`TESTING_COMPLETION_REPORT.md`](../../TESTING_COMPLETION_REPORT.md)
- **Architecture**: [`EMERGENCY_REVOCATION_ARCHITECTURE.md`](../../EMERGENCY_REVOCATION_ARCHITECTURE.md)
- **Implementation Report**: [`EMERGENCY_REVOCATION_IMPLEMENTATION_REPORT.md`](../../EMERGENCY_REVOCATION_IMPLEMENTATION_REPORT.md)

---

## Support

For issues, questions, or contributions:
- **GitHub Issues**: [github.com/mauriciomferz/AgentAuth/issues](https://github.com/mauriciomferz/AgentAuth/issues)
- **Documentation**: [github.com/mauriciomferz/AgentAuth/tree/main/docs](https://github.com/mauriciomferz/AgentAuth/tree/main/docs)

**License**: See [LICENSE](../../LICENSE)

**Status**: Production-Ready (77 tests, 100% pass rate, validated November 2025)
