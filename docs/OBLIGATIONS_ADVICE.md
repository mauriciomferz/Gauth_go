---
title: Obligations Advice
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Obligations & Advice Processing (P2.1)

**Status:** ✅ Implemented (P2 Priority)  
**Version:** 1.0  
**Last Updated:** 2025-11-05

## Overview

The GAuth PDP implements XACML-style **obligations** and **advice** processing for post-decision actions:

- **Obligations**: Mandatory actions that **MUST** succeed (failure can flip `allow→deny` with `denyOnMandatoryFailure=true`)
- **Advice**: Non-mandatory recommendations emitted asynchronously to clients (failures logged but don't affect decisions)

### Key Features

✅ **Extensible Handler Registry**: Register custom obligation handlers (log, notify, rate_limit built-in)  
✅ **Advice Channel**: Async buffered channel for client notifications (non-blocking)  
✅ **Persistent Audit Sink**: Durable storage for obligation execution results  
✅ **Mandatory Obligation Semantics**: Failures can reverse authorization decisions  
✅ **Metrics Integration**: Latency, success/failure counters, mandatory failure tracking  
✅ **Built-in Handlers**: Log, notify, rate_limit with extensible plugin architecture

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                       PDP Engine Evaluation                         │
│  1. Policy Matching                                                 │
│  2. Combining Strategy                                              │
│  3. Decision (allow/deny)                                           │
└────────────┬────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────────┐
│          Obligation/Advice Processing (Post-Decision)               │
│  ┌─────────────────────────────────────────────────────────┐        │
│  │  ExtendedObligationExecutor                             │        │
│  │  ┌────────────────┐  ┌─────────────────┐  ┌──────────┐ │        │
│  │  │ LogHandler     │  │ NotifyHandler   │  │ Custom   │ │        │
│  │  │ (built-in)     │  │ (built-in)      │  │ Handler  │ │        │
│  │  └────────────────┘  └─────────────────┘  └──────────┘ │        │
│  │                                                         │        │
│  │  Execution Results:                                     │        │
│  │  • Success → Increment metrics                          │        │
│  │  • Failure (mandatory) → Flip decision                  │        │
│  │  • Failure (advice) → Log only                          │        │
│  └─────────────────────────────────────────────────────────┘        │
└────┬──────────────────────────────────┬──────────────────────────────┘
     │                                  │
     ▼                                  ▼
┌──────────────────────┐   ┌─────────────────────────────────────┐
│ ObligationAuditSink  │   │    AdviceChannel (Buffered)         │
│ (Persistent Storage) │   │  ┌───────────────────────────────┐  │
│ • JSONL File         │   │  │ Client 1: Advice Consumer     │  │
│ • Database           │   │  ├───────────────────────────────┤  │
│ • External SIEM      │   │  │ Client 2: Monitoring System   │  │
│                      │   │  ├───────────────────────────────┤  │
│ Record:              │   │  │ Client N: ...                 │  │
│ - Timestamp          │   │  └───────────────────────────────┘  │
│ - Obligation ID      │   │                                     │
│ - Success/Error      │   │ Events: AdviceEvent{               │
│ - Duration           │   │   AdviceID, Type, Message,         │
│ - Metadata           │   │   Subject, Action, Resource        │
│                      │   │ }                                   │
└──────────────────────┘   └─────────────────────────────────────┘
```

## Core Components

### 1. AdviceChannel (Non-Mandatory Recommendations)

**Interface:**
```go
type AdviceChannel interface {
    Emit(ctx context.Context, event AdviceEvent) error
    AdviceEvents() <-chan AdviceEvent
    Close() error
}
```

**AdviceEvent Structure:**
```go
type AdviceEvent struct {
    Timestamp  time.Time
    Subject    string  // Authorization request subject
    Action     string  // Authorization request action
    Resource   string  // Authorization request resource
    AdviceID   string  // Unique advice identifier
    AdviceType string  // e.g., "performance", "security", "compliance"
    Message    string  // Human-readable recommendation
    Metadata   map[string]string
}
```

**Buffered Implementation:**
- **Buffer Size**: Configurable (default: 100 events)
- **Overflow Behavior**: Drop oldest events (advice is non-mandatory)
- **Concurrency**: Thread-safe emission and consumption

### 2. ObligationAuditSink (Persistent Storage)

**Interface:**
```go
type ObligationAuditSink interface {
    RecordObligationExecution(ctx context.Context, record ObligationAuditRecord) error
    Close() error
}
```

**ObligationAuditRecord Structure:**
```go
type ObligationAuditRecord struct {
    Timestamp     time.Time
    Subject       string
    Action        string
    Resource      string
    Decision      string  // "allow" or "deny"
    ObligationID  string
    ObligationType string
    Mandatory     bool
    Success       bool
    DurationMS    float64
    Error         string  // Populated on failure
    Metadata      map[string]string
}
```

**Implementations:**
- **JSONFileObligationAuditSink**: JSONL append-only file (built-in)
- **Custom**: Database, SIEM, message queue (implement interface)

### 3. ExtendedObligationExecutor (Execution Engine)

**Features:**
- Handler registry (thread-safe)
- Async audit sink integration (fire-and-forget)
- Advice channel emission
- Built-in handlers (log, notify, rate_limit)

**Creation:**
```go
exec := NewExtendedObligationExecutor(
    WithAdviceChannel(adviceChannel),
    WithObligationAuditSink(auditSink),
)
```

## Usage Examples

### Example 1: Basic Obligation Execution

```go
package main

import (
    "context"
    "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/pdp"
)

func main() {
    // Create PDP engine
    engine := pdp.NewInMemoryEngine(pdp.DenyOverridesStrategy{})

    // Create obligation executor
    exec := pdp.NewExtendedObligationExecutor()

    // Configure engine with obligations
    engine.WithObligations(exec, "")  // Empty audit path (no file logging)
    engine.WithObligationFailureDenies(true)  // Mandatory failures flip decision

    // Add policy with obligations
    policy := pdp.Policy{
        ID:       "policy1",
        Subjects: []string{"alice"},
        Rules: []pdp.Rule{
            {
                ID:     "rule1",
                Action: "read",
                Resource: "doc:*",
                Effect:   "allow",
            },
        },
        Obligations: []pdp.Obligation{
            {ID: "log:user_access", Mandatory: false},
            {ID: "notify:security_team", Mandatory: true},
        },
    }
    engine.AddPolicy(policy)

    // Evaluate request
    req := pdp.Request{
        Subject:  "alice",
        Action:   "read",
        Resource: "doc:123",
    }
    dec, _ := engine.Evaluate(context.Background(), req)
    
    // If "notify:security_team" fails and mandatory=true, dec.Allow will be flipped to false
}
```

### Example 2: Advice Channel with Client Consumption

```go
package main

import (
    "context"
    "fmt"
    "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/pdp"
)

func main() {
    // Create advice channel
    adviceChannel := pdp.NewBufferedAdviceChannel(100)
    defer adviceChannel.Close()

    // Create executor with advice
    exec := pdp.NewExtendedObligationExecutor(
        pdp.WithAdviceChannel(adviceChannel),
    )

    // Start advice consumer goroutine
    go func() {
        for advice := range adviceChannel.AdviceEvents() {
            fmt.Printf("[Advice] %s: %s (Subject: %s, Action: %s, Resource: %s)\n",
                advice.AdviceID,
                advice.Message,
                advice.Subject,
                advice.Action,
                advice.Resource,
            )
        }
    }()

    // Emit advice
    advice := pdp.AdviceEvent{
        AdviceID:   "rate_limit_warning",
        AdviceType: "performance",
        Subject:    "alice",
        Action:     "read",
        Resource:   "doc:123",
        Message:    "User approaching rate limit (80% capacity)",
        Metadata:   map[string]string{"current_rate": "800/1000"},
    }
    exec.EmitAdvice(context.Background(), advice)
}
```

### Example 3: Persistent Audit Sink

```go
package main

import (
    "context"
    "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/pdp"
)

func main() {
    // Create audit sink
    auditSink := pdp.NewJSONFileObligationAuditSink("/var/log/gauth/obligations.jsonl")
    defer auditSink.Close()

    // Create executor with audit
    exec := pdp.NewExtendedObligationExecutor(
        pdp.WithObligationAuditSink(auditSink),
    )

    // Configure PDP engine
    engine := pdp.NewInMemoryEngine(pdp.DenyOverridesStrategy{})
    engine.WithObligations(exec, "")

    // Obligations will be audited to /var/log/gauth/obligations.jsonl
    // Each execution produces a JSON record:
    // {
    //   "timestamp": "2025-11-05T17:00:00Z",
    //   "obligation_id": "log:user_action",
    //   "success": true,
    //   "duration_ms": 0.123,
    //   ...
    // }
}
```

### Example 4: Custom Obligation Handler

```go
package main

import (
    "context"
    "fmt"
    "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/pdp"
)

// Custom handler: Send Slack notification
type SlackNotificationHandler struct {
    WebhookURL string
}

func (h *SlackNotificationHandler) Type() string {
    return "slack"
}

func (h *SlackNotificationHandler) Execute(ctx context.Context, obligationID string, params map[string]string) error {
    // TODO: Send POST request to Slack webhook
    fmt.Printf("Slack notification: %s to %s\n", obligationID, h.WebhookURL)
    return nil
}

func main() {
    // Create executor
    exec := pdp.NewExtendedObligationExecutor()

    // Register custom handler
    exec.RegisterHandler(&SlackNotificationHandler{
        WebhookURL: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    })

    // Now "slack:security_alert" obligations will use this handler
    results := exec.Execute(context.Background(), []string{"slack:security_alert"})
    fmt.Printf("Obligation executed: success=%v\n", results[0].Success)
}
```

### Example 5: Full Integration (Engine + Advice + Audit)

```go
package main

import (
    "context"
    "fmt"
    "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/pdp"
)

func main() {
    // 1. Create advice channel
    adviceChannel := pdp.NewBufferedAdviceChannel(100)
    defer adviceChannel.Close()

    // 2. Create audit sink
    auditSink := pdp.NewJSONFileObligationAuditSink("/var/log/gauth/obligations.jsonl")
    defer auditSink.Close()

    // 3. Create extended executor
    exec := pdp.NewExtendedObligationExecutor(
        pdp.WithAdviceChannel(adviceChannel),
        pdp.WithObligationAuditSink(auditSink),
    )

    // 4. Create PDP engine
    engine := pdp.NewInMemoryEngine(pdp.DenyOverridesStrategy{})
    engine.WithObligations(exec, "")  // Use audit sink instead of file path
    engine.WithObligationFailureDenies(true)  // Mandatory failures flip decision
    engine.WithAdviceChannel(adviceChannel)  // Enable advice emission from engine

    // 5. Add policy with obligations
    policy := pdp.Policy{
        ID:       "policy1",
        Subjects: []string{"alice"},
        Rules: []pdp.Rule{{ID: "rule1", Action: "read", Resource: "doc:*", Effect: "allow"}},
        Obligations: []pdp.Obligation{
            {ID: "log:user_access", Mandatory: false},
            {ID: "notify:admin", Mandatory: true},
        },
    }
    engine.AddPolicy(policy)

    // 6. Start advice consumer
    go func() {
        for advice := range adviceChannel.AdviceEvents() {
            fmt.Printf("[Advice] %s: %s\n", advice.AdviceID, advice.Message)
        }
    }()

    // 7. Evaluate request
    req := pdp.Request{Subject: "alice", Action: "read", Resource: "doc:123"}
    dec, _ := engine.Evaluate(context.Background(), req)

    fmt.Printf("Decision: allow=%v, reason=%s\n", dec.Allow, dec.Reason)
    // Obligations executed, audited to file, advice emitted to channel
}
```

## Built-in Obligation Handlers

### 1. LogObligationHandler

**Type:** `log`  
**Purpose:** Logs obligation execution (stdout)

**Example:**
```go
// Obligation ID: "log:user_access"
// Output: [Obligation:Log] Executed obligation: user_access, params: map[]
```

### 2. NotifyObligationHandler

**Type:** `notify`  
**Purpose:** Notification stub (integrate with email, Slack, webhook)

**Example:**
```go
// Obligation ID: "notify:admin_alert"
// Output: [Obligation:Notify] Notification triggered: admin_alert, params: map[]
```

**Integration Points:**
- Email (SMTP)
- Slack (webhook)
- PagerDuty (API)
- Custom webhook

### 3. RateLimitObligationHandler

**Type:** `rate_limit`  
**Purpose:** Rate limit enforcement stub (integrate with Redis, in-memory store)

**Example:**
```go
// Obligation ID: "rate_limit:api_calls"
// Output: [Obligation:RateLimit] Rate limit check: api_calls, params: map[]
```

**Integration Points:**
- Redis (distributed rate limiting)
- In-memory cache (single-node)
- External rate limiter service

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GAUTH_ADVICE_BUFFER_SIZE` | Advice channel buffer size | 100 |
| `GAUTH_OBLIGATION_AUDIT_PATH` | JSONL audit file path | (none) |
| `GAUTH_DENY_ON_MANDATORY_FAILURE` | Flip decision on mandatory obligation failure | false |

### PDP Engine Options

```go
engine := pdp.NewInMemoryEngine(pdp.DenyOverridesStrategy{})
    .WithObligations(executor, auditPath)  // Executor + optional JSONL file
    .WithObligationFailureDenies(true)     // Mandatory semantics
    .WithAdviceChannel(adviceChannel)      // Advice emission
```

## Testing

### Run Tests

```bash
# All obligation/advice tests
go test -v ./pkg/pdp -run "TestBufferedAdviceChannel|TestExtendedObligationExecutor|TestObligationNameParsing"

# Specific test
go test -v ./pkg/pdp -run TestExtendedObligationExecutor_WithAuditSink
```

### Test Results (P2.1)

✅ **12/12 tests passing (100%)**

| Test | Status | Description |
|------|--------|-------------|
| `TestBufferedAdviceChannel_EmitAndReceive` | ✅ PASS | Advice emission and reception |
| `TestBufferedAdviceChannel_BufferFull` | ✅ PASS | Drop behavior when buffer full |
| `TestBufferedAdviceChannel_ClosedChannel` | ✅ PASS | Emission fails after close |
| `TestExtendedObligationExecutor_LogHandler` | ✅ PASS | Log obligation execution |
| `TestExtendedObligationExecutor_NotifyHandler` | ✅ PASS | Notify obligation execution |
| `TestExtendedObligationExecutor_RateLimitHandler` | ✅ PASS | Rate limit obligation execution |
| `TestExtendedObligationExecutor_UnknownHandler` | ✅ PASS | Error on unknown obligation type |
| `TestExtendedObligationExecutor_MultipleObligations` | ✅ PASS | Batch execution (3 obligations) |
| `TestExtendedObligationExecutor_WithAuditSink` | ✅ PASS | Audit sink integration |
| `TestExtendedObligationExecutor_WithAdviceChannel` | ✅ PASS | Advice emission integration |
| `TestExtendedObligationExecutor_CustomHandler` | ✅ PASS | Custom handler registration |
| `TestObligationNameParsing` | ✅ PASS | Name parsing (3 subtests) |

## Security Considerations

### 1. Mandatory Obligation Failures

**Risk:** Flipping `allow→deny` on mandatory obligation failure can be abused (DoS).

**Mitigation:**
- Set `denyOnMandatoryFailure=false` in non-critical environments
- Monitor `mandatory_obligation_failures_total` metric for anomalies
- Implement circuit breakers for external obligation dependencies

### 2. Advice Channel Overflow

**Risk:** High advice emission rate can cause buffer overflow (dropped events).

**Mitigation:**
- Increase buffer size (`GAUTH_ADVICE_BUFFER_SIZE=1000`)
- Implement backpressure (block emission on critical advice)
- Monitor advice drop rate

### 3. Audit Sink Availability

**Risk:** Audit sink failures can block obligation execution if not fire-and-forget.

**Mitigation:**
- Use async audit (fire-and-forget pattern in `ExtendedObligationExecutor`)
- Implement audit sink retries with exponential backoff
- Monitor audit sink error rate

### 4. Custom Handler Security

**Risk:** Malicious custom handlers can leak sensitive data or cause DoS.

**Mitigation:**
- Validate custom handler implementations
- Implement handler timeout (5s default)
- Sandbox handler execution (future: WebAssembly isolation)

## Troubleshooting

### Issue 1: Obligations Not Executing

**Symptoms:**
- No obligation logs
- Metrics show 0 executions

**Diagnosis:**
```bash
# Check if executor configured
grep "WithObligations" your_code.go

# Verify policy has obligations
curl http://localhost:8080/policies | jq '.[] | select(.obligations | length > 0)'
```

**Solution:**
- Ensure `engine.WithObligations(exec, "")` called
- Verify policy has `Obligations: []pdp.Obligation{...}` defined

### Issue 2: Advice Not Received

**Symptoms:**
- Advice channel has no events
- Consumer goroutine blocks indefinitely

**Diagnosis:**
```go
// Check if advice channel configured
if engine.adviceChannel == nil {
    fmt.Println("Advice channel not configured")
}

// Verify buffer size
fmt.Printf("Buffer size: %d\n", cap(adviceChannel.AdviceEvents()))
```

**Solution:**
- Call `engine.WithAdviceChannel(ch)` before evaluation
- Ensure consumer goroutine started before emission

### Issue 3: Audit Records Missing

**Symptoms:**
- JSONL file empty
- No audit sink writes

**Diagnosis:**
```bash
# Check file permissions
ls -la /var/log/gauth/obligations.jsonl

# Verify sink configured
grep "WithObligationAuditSink" your_code.go
```

**Solution:**
- Ensure audit sink passed to executor: `WithObligationAuditSink(sink)`
- Check file permissions (writable)
- Verify audit sink `RecordObligationExecution()` not returning error

## Migration Guide

### Phase 1: Assessment (Week 1)

1. **Inventory existing policies**: Identify policies needing obligations
2. **Define obligation types**: Map business requirements (log, notify, rate_limit)
3. **Plan advice scenarios**: Determine when to emit recommendations (e.g., approaching rate limit)

### Phase 2: Pilot (Weeks 2-3)

1. **Create obligation executor**:
   ```go
   exec := pdp.NewExtendedObligationExecutor()
   ```

2. **Configure advice channel** (optional):
   ```go
   adviceChannel := pdp.NewBufferedAdviceChannel(100)
   exec = pdp.NewExtendedObligationExecutor(pdp.WithAdviceChannel(adviceChannel))
   ```

3. **Add obligations to 1-2 policies**:
   ```go
   policy.Obligations = []pdp.Obligation{
       {ID: "log:pilot_access", Mandatory: false},
   }
   ```

4. **Monitor metrics**:
   - `gauth_rfc0111_obligations_executed_total`
   - `gauth_rfc0111_obligations_failed_total`

### Phase 3: Production Rollout (Weeks 4-6)

1. **Enable mandatory semantics** (if needed):
   ```go
   engine.WithObligationFailureDenies(true)
   ```

2. **Add persistent audit sink**:
   ```go
   auditSink := pdp.NewJSONFileObligationAuditSink("/var/log/gauth/obligations.jsonl")
   exec = pdp.NewExtendedObligationExecutor(pdp.WithObligationAuditSink(auditSink))
   ```

3. **Implement custom handlers** (e.g., Slack, PagerDuty):
   ```go
   exec.RegisterHandler(&SlackNotificationHandler{...})
   ```

4. **Set alerts**:
   - High obligation latency (p95 > 25ms)
   - Mandatory failure surge
   - Advice buffer overflow

## API Reference

### Core Functions

#### NewBufferedAdviceChannel

```go
func NewBufferedAdviceChannel(bufferSize int) *BufferedAdviceChannel
```

Creates advice channel with specified buffer size (default: 100).

#### NewExtendedObligationExecutor

```go
func NewExtendedObligationExecutor(opts ...ExtendedExecutorOption) *ExtendedObligationExecutor
```

Creates obligation executor with optional advice channel and audit sink.

**Options:**
- `WithAdviceChannel(ch AdviceChannel)`
- `WithObligationAuditSink(sink ObligationAuditSink)`

#### RegisterHandler

```go
func (e *ExtendedObligationExecutor) RegisterHandler(handler ObligationHandler)
```

Registers custom obligation handler (thread-safe).

#### WithAdviceChannel (Engine)

```go
func (e *InMemoryEngine) WithAdviceChannel(ch AdviceChannel) *InMemoryEngine
```

Configures advice emission channel for PDP engine.

## Roadmap

### Phase 1: Enhancements (Q1 2026)

- [ ] **Handler Timeout**: 5s default timeout for obligation execution
- [ ] **Retry Logic**: Exponential backoff for transient failures
- [ ] **Batching**: Batch audit sink writes for performance

### Phase 2: External Integrations (Q2 2026)

- [ ] **Email Handler**: SMTP integration for notifications
- [ ] **Slack Handler**: Webhook integration
- [ ] **PagerDuty Handler**: API integration for incident creation
- [ ] **Redis Rate Limiter**: Distributed rate limiting

### Phase 3: Advanced Features (Q3 2026)

- [ ] **WebAssembly Sandboxing**: Isolated handler execution
- [ ] **Conditional Obligations**: Execute only if condition met
- [ ] **Obligation Chaining**: Sequential obligation dependencies

## FAQ

### Q1: What's the difference between obligations and advice?

**Obligations:** Mandatory actions. If `denyOnMandatoryFailure=true`, failures can flip `allow→deny`.  
**Advice:** Non-mandatory recommendations emitted asynchronously. Failures logged but don't affect decisions.

### Q2: Can I use multiple advice channels?

No. The PDP engine supports one `AdviceChannel`. However, you can create a fan-out channel that broadcasts to multiple consumers.

### Q3: How do I integrate with external notification services?

Implement `ObligationHandler` interface:
```go
type SlackHandler struct { WebhookURL string }
func (h *SlackHandler) Type() string { return "slack" }
func (h *SlackHandler) Execute(ctx context.Context, id string, params map[string]string) error {
    // Send POST to Slack webhook
}
```

Then register: `exec.RegisterHandler(&SlackHandler{...})`

### Q4: What happens if audit sink fails?

Audit failures don't block obligation execution (fire-and-forget pattern). Errors logged but execution continues.

### Q5: Can obligations modify the authorization context?

No. Obligations execute **after** the decision is finalized. They cannot modify request attributes or decision outcome (except mandatory obligation failures with `denyOnMandatoryFailure=true`).

## References

- **P1.4 Audit Sink Integration**: [docs/AUDIT_SINK_INTEGRATION.md](AUDIT_SINK_INTEGRATION.md)
- **XACML Obligations**: [OASIS XACML 3.0 Specification §7.18](https://docs.oasis-open.org/xacml/3.0/xacml-3.0-core-spec-os-en.html)
- **Metrics Instrumentation**: [docs/metrics-obligations.md](metrics-obligations.md)
- **GAP Matrix**: [docs/GAP_MATRIX.auto.md](GAP_MATRIX.auto.md) (sec2.item3)

---

**Implementation Status:** ✅ Implemented (12/12 tests passing)  
**Last Updated:** 2025-11-05T18:00:00Z  
**Contributors:** GAuth Core Team
