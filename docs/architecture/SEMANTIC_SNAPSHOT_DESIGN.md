# Semantic Snapshot Archival and Rotation Design

**Version**: 1.0  
**Date**: 2025-12-27  
**Status**: Design Approved  
**Implementation**: Phase 17 (P2)

---

## Overview

The semantic analysis snapshot system currently maintains in-memory EWMA (Exponentially Weighted Moving Average) state for request rate analysis. This design extends the snapshot capability to support:
1. **Multi-file rotation** for long-term historical analysis
2. **External anchoring** via RFC-3161 TSA for integrity verification
3. **Surge indicator alerts** with configurable thresholds
4. **Archive management** with retention policies

---

## Architecture

### Current State
- **In-Memory Snapshots**: EWMA rate state maintained in memory
- **Single Snapshot**: One snapshot per service provider
- **No Persistence**: State lost on restart
- **No History**: No long-term trend analysis

### Proposed State
- **Persistent Snapshots**: Daily/weekly rotation to disk
- **Multi-File Archives**: Historical snapshots retained per policy
- **External Anchoring**: Snapshots anchored to TSA for tamper-proofing
- **Alert Hooks**: Webhook/SMTP notifications on surge detection

---

## Design

### 1. Snapshot Rotation Policy

```go
type SnapshotRotationPolicy struct {
    Interval     time.Duration // Rotation interval (e.g., 24h for daily)
    Retention    time.Duration // How long to keep archives (e.g., 90 days)
    MaxSnapshots int           // Max snapshots before deletion (FIFO)
    Anchoring    bool          // Enable external TSA anchoring
}
```

**Rotation Triggers**:
- **Time-based**: Every `Interval` (e.g., daily at midnight UTC)
- **Size-based**: When snapshot exceeds `MaxSize` bytes
- **On-demand**: Via admin API for ad-hoc snapshots

### 2. Snapshot File Format

```
snapshots/
├── semantic_snapshot_2025-12-27_00-00-00.json
├── semantic_snapshot_2025-12-27_00-00-00.json.anchor  (TSA receipt)
├── semantic_snapshot_2025-12-26_00-00-00.json
└── semantic_snapshot_2025-12-26_00-00-00.json.anchor
```

**Snapshot Schema**:
```json
{
  "timestamp": "2025-12-27T00:00:00Z",
  "service_providers": {
    "sp_123": {
      "request_rate_ewma": 1234.56,
      "surge_detected": false,
      "surge_threshold": 5000.0,
      "sample_count": 10000,
      "last_updated": "2025-12-27T00:00:00Z"
    }
  },
  "integrity": {
    "hash_algorithm": "SHA-256",
    "snapshot_hash": "abc123...",
    "anchored": true,
    "anchor_receipt_path": "./semantic_snapshot_2025-12-27_00-00-00.json.anchor"
  }
}
```

### 3. External Anchoring Integration

**Process**:
1. Snapshot created and written to disk
2. Compute SHA-256 hash of snapshot file
3. Submit hash to RFC-3161 TSA via `ExternalAnchorClient`
4. Store TSA receipt in `.anchor` file
5. Optionally verify receipt on read

**Benefits**:
- **Tamper-proof**: Cannot modify snapshot without detection
- **Compliance**: Regulatory requirements for audit trails
- **Forensics**: Cryptographic proof of snapshot state at time T

### 4. Surge Indicator Alerts

**Alert Configuration**:
```go
type AlertConfig struct {
    Enabled       bool
    ThresholdType string  // "absolute" or "relative" (e.g., 200% of baseline)
    Threshold     float64
    Recipients    []string // Email addresses or webhook URLs
    Cooldown      time.Duration // Min time between alerts
}
```

**Alert Triggers**:
- **Absolute Threshold**: `request_rate_ewma > threshold`
- **Relative Threshold**: `request_rate_ewma > baseline * multiplier`
- **Sustained Surge**: Rate exceeds threshold for > N snapshots

**Alert Channels**:
- **Webhook**: `POST {url}/alerts/surge` with JSON payload
- **SMTP**: Email to configured recipients
- **Metrics**: `surge_alerts_total` counter incremented

---

## Implementation Plan

### Phase 1: Snapshot Rotation (Week 3, Day 1-2)
- [ ] Implement `SnapshotRotator` service
- [ ] Add rotation policy configuration
- [ ] File-based snapshot persistence
- [ ] FIFO retention enforcement

### Phase 2: External Anchoring (Week 3, Day 3-4)
- [ ] Integrate with `ExternalAnchorClient`
- [ ] TSA receipt storage and verification
- [ ] Optional anchor verification on read

### Phase 3: Alert Hooks (Week 3, Day 5)
- [ ] Surge detection logic
- [ ] Webhook/SMTP alert delivery
- [ ] Alert cooldown and rate limiting

### Phase 4: Archive Management API (Optional)
- [ ] `GET /api/v1/admin/snapshots` - List snapshots
- [ ] `GET /api/v1/admin/snapshots/{id}` - Retrieve snapshot
- [ ] `POST /api/v1/admin/snapshots` - Create ad-hoc snapshot
- [ ] `DELETE /api/v1/admin/snapshots/{id}` - Purge snapshot

---

## Configuration

### Environment Variables
```bash
# Rotation
AGENTAUTH_SNAPSHOT_ROTATION_INTERVAL=24h
AGENTAUTH_SNAPSHOT_RETENTION=90d
AGENTAUTH_SNAPSHOT_MAX_COUNT=90

# Anchoring
AGENTAUTH_SNAPSHOT_ANCHORING_ENABLED=true
AGENTAUTH_SNAPSHOT_TSA_URL=https://freetsa.org/tsr

# Alerts
AGENTAUTH_SNAPSHOT_ALERT_ENABLED=true
AGENTAUTH_SNAPSHOT_ALERT_THRESHOLD=5000.0
AGENTAUTH_SNAPSHOT_ALERT_WEBHOOK=https://alerts.example.com/webhook
```

---

## Operational Considerations

### Storage Requirements
- **Snapshot Size**: ~50KB per snapshot (100 service providers)
- **Daily Rotation**: ~18MB/year
- **90-Day Retention**: ~4.5MB storage

### Performance Impact
- **Rotation Overhead**: Negligible (background job, off-peak)
- **Anchoring Latency**: +500ms per rotation (TSA roundtrip)
- **Read Impact**: None (snapshots are historical, read-only)

### Disaster Recovery
- **Backup**: Snapshots backed up with main database
- **Restore**: Snapshots can be replayed for rate state reconstruction
- **Anchor Verification**: Verify all anchor receipts post-restore

---

## Future Enhancements

1. **Compression**: gzip snapshots to reduce storage by 70%
2. **S3/GCS Upload**: Cloud storage for long-term retention
3. **Snapshot Diff**: Compute delta between snapshots for anomaly detection
4. **Machine Learning**: Anomaly detection using historical snapshot trends

---

## References
- [Semantic Analysis Implementation](file:///Users/mauricio.fernandez_fernandezsiemens.co/AgentAuth/web/handlers/semantics/)
- [External Anchoring](file:///Users/mauricio.fernandez_fernandezsiemens.co/AgentAuth/pkg/ledger/external_anchor.go)
- [RFC 3161 Client](file:///Users/mauricio.fernandez_fernandezsiemens.co/AgentAuth/pkg/ledger/rfc3161/client.go)

---

**Status**: Design complete. Implementation scheduled for Phase 17 expansion (P2 priority).
