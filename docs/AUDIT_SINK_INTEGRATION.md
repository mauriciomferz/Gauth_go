# Audit Sink Integration Guide

## Overview

The Audit Sink Integration (P1.4) enables RFC0111 token lifecycle events to be sent to external audit destinations such as SIEM systems, log aggregators, compliance archives, and message queues. This feature provides real-time audit trail streaming while maintaining backward compatibility with existing deployments.

**Key Features:**
- **Opt-in Design**: Audit sinks are disabled by default (backward compatible)
- **Async/Sync Modes**: Buffered async sinks prevent blocking token operations
- **Multiple Destinations**: Multiplex to SIEM, compliance DB, and message queues simultaneously
- **Event Filtering**: Send only specific event types, actions, or results to targeted sinks
- **Fail-Open**: Sink errors don't block token operations (configurable)
- **Comprehensive Events**: CreateDelegation, VerifyToken, RevokeToken, AttachEvidence

---

## Architecture

### Integration Points

```
┌─────────────────────────────────────────────────────────────┐
│                    RFC0111 Service                          │
│                                                             │
│  ┌──────────────┐       ┌──────────────┐                   │
│  │ CreateDelegation│───>│ AuditLogger  │                   │
│  └──────────────┘       │ (MemoryLogger│                   │
│          │              │  or FileLogger│                   │
│          │              └──────────────┘                   │
│          │                     │                            │
│          │                     │ Log()                      │
│          │                     ▼                            │
│          │              ┌──────────────┐                   │
│          └─────────────>│ sendToAuditSink()│              │
│                         └──────────────┘                   │
│                                │                            │
│                                │ Send()                     │
│                                ▼                            │
│                         ┌──────────────┐                   │
│                         │  AuditSink   │ (opt-in)          │
│                         │ (external)   │                   │
│                         └──────────────┘                   │
│                                │                            │
│          ┌─────────────────────┴───────────────────────┐  │
│          │                     │                       │   │
│          ▼                     ▼                       ▼   │
│   ┌───────────┐        ┌───────────┐          ┌───────────┐│
│   │   SIEM    │        │ Compliance│          │  Message  ││
│   │  System   │        │  Archive  │          │   Queue   ││
│   └───────────┘        └───────────┘          └───────────┘│
│                                                             │
└─────────────────────────────────────────────────────────────┘

Event Flow:
1. CreateDelegation/VerifyToken/RevokeToken operations execute
2. Audit event logged to AuditLogger (MemoryLogger/FileLogger)
3. Event sent to external AuditSink (if configured) via sendToAuditSink()
4. Sink distributes to external destinations (SIEM, compliance, queue)
5. Errors in sink don't fail operation (fail-open for resilience)
```

**Key Components:**
- **AuditLogger**: Internal audit chain (MemoryLogger with hash chaining)
- **AuditSink**: External destination interface (SIEM, compliance, queue)
- **sendToAuditSink()**: Integration helper with 5s timeout + fail-open
- **WithAuditSink()**: Service option for opt-in sink configuration

**Event Lifecycle:**
1. Token operation completes → audit event created
2. Event logged to internal AuditLogger (always, required)
3. Event sent to external AuditSink (optional, if configured)
4. Sink errors logged but don't block operation (fail-open)

---

## Usage Examples

### 1. Enable Basic Audit Sink

```go
import (
    "context"
    "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc0111"
    "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
)

// Create simple sink that sends events to SIEM
type SiemSink struct {
    endpoint string
}

func (s *SiemSink) Send(ctx context.Context, event *audit.Event) error {
    // Send event to SIEM endpoint
    return sendToSiem(s.endpoint, event)
}

func (s *SiemSink) Close() error {
    return nil
}

// Configure service with sink
siemSink := &SiemSink{endpoint: "https://siem.example.com/ingest"}
svc := rfc0111.NewService(logger, authorizer, rfc0111.WithAuditSink(siemSink))
```

### 2. Use Async Sink for High Throughput

For slow destinations (databases, remote APIs), use `AsyncAuditSink` to prevent blocking token operations:

```go
// Create slow destination (e.g., compliance database)
complianceDB := &ComplianceDatabaseSink{
    dbURL: "postgres://compliance.example.com/audit",
}

// Wrap with async buffer (1000 events)
asyncSink := rfc0111.NewAsyncAuditSink(complianceDB, 1000)
defer asyncSink.Close() // Flush on shutdown

svc := rfc0111.NewService(logger, authorizer, rfc0111.WithAuditSink(asyncSink))

// Token operations won't block waiting for DB writes
resp, err := svc.CreateDelegationCtx(ctx, req)
```

**Async Sink Benefits:**
- **Non-blocking**: Token operations complete immediately (queued)
- **Buffered**: Up to 1000 events in memory before flushing
- **Fail-open**: Buffer overflow drops oldest events (logged)
- **Graceful Shutdown**: `Close()` flushes remaining events

### 3. Multiplex to Multiple Destinations

Send events to SIEM + compliance DB + message queue simultaneously:

```go
siemSink := &SiemSink{endpoint: "https://siem.example.com"}
complianceSink := &ComplianceDatabaseSink{dbURL: "postgres://..."}
queueSink := &MessageQueueSink{topic: "audit-events"}

// Multiplex to all 3 destinations
multiplex := rfc0111.NewMultiplexAuditSink(siemSink, complianceSink, queueSink)

svc := rfc0111.NewService(logger, authorizer, rfc0111.WithAuditSink(multiplex))

// Events sent to all 3 destinations in parallel
resp, err := svc.CreateDelegationCtx(ctx, req)
```

**Multiplex Benefits:**
- **Parallel**: All sinks receive events concurrently
- **Partial Success**: Some sinks can fail without affecting others
- **Error Aggregation**: Returns combined error if multiple sinks fail

### 4. Filter Events by Action

Send only revocation events to compliance archive (high-severity filtering):

```go
complianceSink := &ComplianceDatabaseSink{dbURL: "postgres://..."}

// Filter: only "revoke_delegation" actions
filtered := rfc0111.NewFilteredAuditSink(
    complianceSink,
    rfc0111.FilterByAction("revoke_delegation"),
)

svc := rfc0111.NewService(logger, authorizer, rfc0111.WithAuditSink(filtered))

// CreateDelegation events ignored (filtered out)
resp, _ := svc.CreateDelegationCtx(ctx, req)

// RevokeDelegation events sent to compliance DB
_ = svc.RevokeDelegation(resp.POA.ID, "alice@example.com")
```

**Filter Predicates:**
- `FilterByAction(actions...)`: Allow specific actions (e.g., "revoke_delegation")
- `FilterByEventType(types...)`: Allow specific event types (e.g., `audit.TypeAuth`)
- `FilterByResult(results...)`: Allow specific results (e.g., `audit.ResultFailure`)
- **Custom**: `func(*audit.Event) bool` for advanced logic

### 5. Send Only Failures to Alerting System

Alert on authorization failures but don't spam with successes:

```go
alertSink := &AlertingSink{webhook: "https://alerts.example.com/webhook"}

// Filter: only failures
failuresOnly := rfc0111.NewFilteredAuditSink(
    alertSink,
    rfc0111.FilterByResult(audit.ResultFailure),
)

svc := rfc0111.NewService(logger, authorizer, rfc0111.WithAuditSink(failuresOnly))

// Failed delegations trigger alerts
_, err := svc.CreateDelegationCtx(ctx, unauthorizedReq) // Alert sent!
```

### 6. Complex Multi-Sink Setup

Real-world deployment with SIEM (all events), compliance DB (revocations only), and alerts (failures only):

```go
// Sink 1: SIEM (all events, async)
siemSink := rfc0111.NewAsyncAuditSink(
    &SiemSink{endpoint: "https://siem.example.com"},
    1000,
)

// Sink 2: Compliance DB (revocations only, async)
complianceSink := rfc0111.NewFilteredAuditSink(
    rfc0111.NewAsyncAuditSink(&ComplianceDatabaseSink{dbURL: "postgres://..."}, 500),
    rfc0111.FilterByAction("revoke_delegation"),
)

// Sink 3: Alerts (failures only, sync)
alertSink := rfc0111.NewFilteredAuditSink(
    &AlertingSink{webhook: "https://alerts.example.com"},
    rfc0111.FilterByResult(audit.ResultFailure),
)

// Multiplex all 3
multiplex := rfc0111.NewMultiplexAuditSink(siemSink, complianceSink, alertSink)

svc := rfc0111.NewService(logger, authorizer, rfc0111.WithAuditSink(multiplex))
```

---

## Sink Implementations

### SIEM Sink (Splunk/Elasticsearch)

```go
type SplunkSink struct {
    endpoint string
    token    string
    client   *http.Client
}

func (s *SplunkSink) Send(ctx context.Context, event *audit.Event) error {
    body, _ := json.Marshal(map[string]interface{}{
        "event":      event,
        "sourcetype": "gauth:audit",
        "index":      "security",
    })

    req, _ := http.NewRequestWithContext(ctx, "POST", s.endpoint, bytes.NewReader(body))
    req.Header.Set("Authorization", "Splunk "+s.token)

    resp, err := s.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("splunk: %s", resp.Status)
    }
    return nil
}

func (s *SplunkSink) Close() error {
    return nil
}
```

### Compliance Database Sink (PostgreSQL)

```go
type PostgresSink struct {
    db *sql.DB
}

func (s *PostgresSink) Send(ctx context.Context, event *audit.Event) error {
    query := `
        INSERT INTO audit_trail (event_id, timestamp, type, action, result, subject, object, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `
    metadata, _ := json.Marshal(event.Metadata)

    _, err := s.db.ExecContext(ctx, query,
        event.ID, event.Timestamp, event.Type, event.Action,
        event.Result, event.Subject, event.Object, metadata,
    )
    return err
}

func (s *PostgresSink) Close() error {
    return s.db.Close()
}
```

### Message Queue Sink (Kafka/RabbitMQ)

```go
type KafkaSink struct {
    producer sarama.SyncProducer
    topic    string
}

func (k *KafkaSink) Send(ctx context.Context, event *audit.Event) error {
    data, _ := json.Marshal(event)

    msg := &sarama.ProducerMessage{
        Topic: k.topic,
        Key:   sarama.StringEncoder(event.ID),
        Value: sarama.ByteEncoder(data),
    }

    _, _, err := k.producer.SendMessage(msg)
    return err
}

func (k *KafkaSink) Close() error {
    return k.producer.Close()
}
```

---

## Configuration

### Environment Variables

```bash
# Sink timeout (default: 5s)
GAUTH_AUDIT_SINK_TIMEOUT=10s

# Async buffer size (default: 1000)
GAUTH_AUDIT_SINK_BUFFER_SIZE=5000

# Fail-closed mode (sink errors fail operations)
GAUTH_AUDIT_SINK_FAIL_CLOSED=true
```

### Service Configuration

```go
svc := rfc0111.NewService(
    logger,
    authorizer,
    rfc0111.WithAuditSink(sink),           // External sink
    rfc0111.WithJurisdictionEnforcement(), // P1.3 jurisdiction
    rfc0111.WithKMS(kms),                  // Key management
)
```

---

## Testing

### Run Audit Sink Tests

```bash
# All audit sink integration tests
go test -v ./pkg/rfc0111 -run TestAuditSinkIntegration

# Specific test
go test -v ./pkg/rfc0111 -run TestAuditSinkIntegration_AsyncSink
```

### Test Results (All 11/11 Passing)

```
✅ TestAuditSinkIntegration_Disabled              # Backward compat (no sink)
✅ TestAuditSinkIntegration_CreateDelegation      # Basic sink integration
✅ TestAuditSinkIntegration_VerifyToken           # Token verification events
✅ TestAuditSinkIntegration_RevokeToken           # Token revocation events
✅ TestAuditSinkIntegration_AsyncSink             # Async buffering
✅ TestAuditSinkIntegration_ErrorHandling         # Fail-open behavior
✅ TestAuditSinkIntegration_MultiplexSink         # Multiple destinations
✅ TestAuditSinkIntegration_FilteredSink          # Action filtering
✅ TestAuditSinkIntegration_FilterByEventType     # EventType filtering
✅ TestAuditSinkIntegration_FilterByResult        # Result filtering
✅ TestAsyncAuditSink_BufferOverflow              # Buffer overflow handling
```

---

## Security Considerations

### 1. **Sensitive Data Leakage**

**Risk**: Audit events contain PII (grantor, grantee, scope, restrictions)

**Mitigation**:
- **Encrypt in Transit**: Use TLS for sink connections (HTTPS, TLS-secured Kafka)
- **Encrypt at Rest**: Sink implementations should encrypt sensitive metadata
- **Redaction**: Filter sensitive fields before sending to external sinks
```go
type RedactingSink struct {
    sink AuditSink
}

func (r *RedactingSink) Send(ctx context.Context, event *audit.Event) error {
    redacted := *event
    delete(redacted.Metadata, "ssn")      // Remove SSN
    redacted.Subject = hashPII(event.Subject) // Hash grantor
    return r.sink.Send(ctx, &redacted)
}
```

### 2. **Sink Authentication**

**Best Practices**:
- Use API tokens, OAuth, or mutual TLS for sink authentication
- Rotate credentials regularly (90-day rotation)
- Store credentials in secret manager (Vault, AWS Secrets Manager)

```go
siemSink := &SiemSink{
    endpoint: os.Getenv("SIEM_ENDPOINT"),
    token:    secretManager.Get("SIEM_API_TOKEN"), // From secret manager
}
```

### 3. **Sink Availability**

**Risk**: Sink downtime blocks operations (if fail-closed)

**Mitigation**:
- **Default Fail-Open**: Sink errors don't block token operations (logged only)
- **Async Buffering**: Use `AsyncAuditSink` to queue events during outages
- **Circuit Breaker**: Disable sink after N consecutive failures

### 4. **Event Integrity**

**Risk**: Events modified in transit or at sink

**Mitigation**:
- **Hash Chaining**: Internal AuditLogger maintains hash-chained events
- **Sink Verification**: Sinks should verify event hashes against chain
- **Digital Signatures**: Sign events before sending to sink

---

## Troubleshooting

### Issue 1: Events Not Appearing in Sink

**Symptoms**: Token operations succeed but sink has no events

**Possible Causes:**
1. **Sink Not Configured**: `WithAuditSink()` not called
2. **Filtering**: Events filtered out by predicate
3. **Sink Errors**: Sink.Send() returning errors (check logs)
4. **Async Buffer**: Events buffered but not flushed (call `Close()`)

**Diagnosis:**
```go
// Check if sink is configured
if service.auditSink == nil {
    log.Println("No audit sink configured")
}

// Check async sink metrics
if async, ok := sink.(*AsyncAuditSink); ok {
    log.Printf("Sent: %d, Dropped: %d, Errors: %d", async.Sent, async.Dropped, async.Errors)
}
```

### Issue 2: High Latency in Token Operations

**Symptoms**: Token operations slow (> 100ms) after enabling sink

**Possible Causes:**
1. **Sync Sink**: Blocking on slow destination (DB, API)
2. **Small Buffer**: Async buffer too small, causing backpressure

**Fix**: Use `AsyncAuditSink` with larger buffer
```go
// Before (slow, sync sink)
svc := rfc0111.NewService(logger, authorizer, rfc0111.WithAuditSink(slowSink))

// After (fast, async sink with 5000 buffer)
asyncSink := rfc0111.NewAsyncAuditSink(slowSink, 5000)
svc := rfc0111.NewService(logger, authorizer, rfc0111.WithAuditSink(asyncSink))
```

### Issue 3: Events Dropped (Buffer Overflow)

**Symptoms**: `AsyncAuditSink.Dropped` counter increasing

**Possible Causes:**
1. **High Event Rate**: More events than sink can process
2. **Slow Sink**: Sink.Send() taking too long
3. **Small Buffer**: Buffer size insufficient

**Fix**: Increase buffer size or optimize sink
```go
// Increase buffer from 1000 to 10000
asyncSink := rfc0111.NewAsyncAuditSink(slowSink, 10000)
```

---

## Migration Guide

### Phase 1: Assessment (Week 1)

1. **Inventory Audit Requirements**:
   - Identify compliance requirements (SOX, GDPR, HIPAA)
   - Determine required event retention (90 days, 7 years)
   - List audit destinations (SIEM, compliance DB, message queue)

2. **Evaluate Sink Implementations**:
   - Choose sink types (SIEM, database, queue)
   - Assess latency requirements (async vs sync)
   - Determine filtering needs (all events vs high-severity only)

### Phase 2: Pilot (Weeks 2-3)

1. **Implement Sink**:
```go
// Start with simple sync sink for testing
testSink := &SimpleSink{endpoint: "https://test-siem.example.com"}
svc := rfc0111.NewService(logger, authorizer, rfc0111.WithAuditSink(testSink))
```

2. **Test Event Flow**:
   - Create test delegations, verify events in sink
   - Revoke delegations, verify revocation events
   - Verify event filtering if configured

3. **Monitor Performance**:
   - Measure latency impact (should be < 5ms with async sink)
   - Check buffer overflow metrics (`AsyncAuditSink.Dropped`)
   - Monitor sink error rates

### Phase 3: Production Rollout (Weeks 4-6)

1. **Deploy Async Sink**:
```go
// Production: async multiplex to SIEM + compliance DB
siemSink := rfc0111.NewAsyncAuditSink(&SiemSink{...}, 5000)
complianceSink := rfc0111.NewAsyncAuditSink(&ComplianceSink{...}, 2000)
multiplex := rfc0111.NewMultiplexAuditSink(siemSink, complianceSink)

svc := rfc0111.NewService(logger, authorizer, rfc0111.WithAuditSink(multiplex))
```

2. **Enable Monitoring**:
   - Prometheus metrics for sink errors (`audit_sink_errors_total`)
   - Alerts for dropped events (`audit_sink_dropped_total > 100`)
   - Dashboards for event throughput

3. **Gradual Rollout**:
   - Week 4: 10% of traffic
   - Week 5: 50% of traffic
   - Week 6: 100% of traffic

---

## API Reference

### AuditSink Interface

```go
type AuditSink interface {
    Send(ctx context.Context, event *audit.Event) error
    Close() error
}
```

### WithAuditSink(sink AuditSink) Option

Configures external audit sink for token lifecycle events.

**Parameters**:
- `sink`: AuditSink implementation (required, must not be nil)

**Returns**: `Option` for NewService

**Example**:
```go
svc := rfc0111.NewService(logger, authorizer, rfc0111.WithAuditSink(mySink))
```

### NewAsyncAuditSink(sink AuditSink, bufferSize int) *AsyncAuditSink

Creates async wrapper with buffered background flushing.

**Parameters**:
- `sink`: Base sink to wrap
- `bufferSize`: Event buffer capacity (default 1000)

**Returns**: `*AsyncAuditSink`

**Example**:
```go
asyncSink := rfc0111.NewAsyncAuditSink(slowSink, 5000)
defer asyncSink.Close() // Flush on shutdown
```

### NewMultiplexAuditSink(sinks ...AuditSink) *MultiplexAuditSink

Sends events to multiple sinks in parallel.

**Parameters**:
- `sinks`: Variable number of AuditSink implementations

**Returns**: `*MultiplexAuditSink`

**Example**:
```go
multiplex := rfc0111.NewMultiplexAuditSink(siem, compliance, queue)
```

### NewFilteredAuditSink(sink AuditSink, predicate func(*audit.Event) bool) *FilteredAuditSink

Wraps sink with event filtering.

**Parameters**:
- `sink`: Base sink to wrap
- `predicate`: Filter function (return true to allow event)

**Returns**: `*FilteredAuditSink`

**Example**:
```go
filtered := rfc0111.NewFilteredAuditSink(
    sink,
    rfc0111.FilterByAction("revoke_delegation"),
)
```

---

## Roadmap

### P1.4.1: Enhanced Sink Features (Q2 2025)

- **Batching**: Send events in batches (reduce API calls)
- **Compression**: gzip compression for large payloads
- **Retry Logic**: Exponential backoff for transient failures
- **Dead Letter Queue**: Persist failed events for manual retry

### P1.4.2: Additional Sink Implementations (Q3 2025)

- **CloudWatch Logs**: AWS CloudWatch integration
- **Azure Monitor**: Azure Monitor Logs integration
- **Google Cloud Logging**: GCP Cloud Logging integration
- **Prometheus**: Metrics export for observability

### P1.4.3: Advanced Features (Q4 2025)

- **Event Sampling**: Send 10% of events for high-volume systems
- **Rate Limiting**: Limit events/sec to protect sink
- **Schema Validation**: Validate events against JSON schema before sending
- **Encryption**: End-to-end encryption for sensitive events
