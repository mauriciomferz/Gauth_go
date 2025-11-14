# JWE Monitoring and Metrics

## Overview

This document describes monitoring metrics for JWE (JSON Web Encryption) operations in GAuth.

## Performance Metrics

### Encryption Operations

```
jwe_encryption_duration_microseconds
  - Description: Time taken to encrypt JWT to JWE
  - Type: Histogram
  - Labels: key_id, algorithm
  - Target: < 200 μs (P99)
  - Current: ~126 μs (average)
```

```
jwe_encryption_total
  - Description: Total number of encryption operations
  - Type: Counter
  - Labels: key_id, algorithm, status (success/failure)
```

```
jwe_encryption_failures_total
  - Description: Number of failed encryption operations
  - Type: Counter
  - Labels: key_id, error_type
```

### Decryption Operations

```
jwe_decryption_duration_microseconds
  - Description: Time taken to decrypt JWE to JWT
  - Type: Histogram
  - Labels: key_id, algorithm
  - Target: < 1000 μs (P99)
  - Current: ~833 μs (average)
```

```
jwe_decryption_total
  - Description: Total number of decryption operations
  - Type: Counter
  - Labels: key_id, algorithm, status (success/failure)
```

```
jwe_decryption_failures_total
  - Description: Number of failed decryption operations
  - Type: Counter
  - Labels: key_id, error_type
```

### Token Size Metrics

```
jwe_token_size_bytes
  - Description: Size of JWE tokens in bytes
  - Type: Histogram
  - Labels: key_id
  - Typical: 800-1200 bytes (depending on Extended Token content)
```

```
jwe_compression_ratio
  - Description: Compression ratio achieved by DEFLATE
  - Type: Histogram
  - Target: 0.6-0.7 (30-40% size reduction)
```

### Key Rotation Metrics

```
jwe_active_keys
  - Description: Number of active encryption keys
  - Type: Gauge
  - Labels: key_id
```

```
jwe_key_rotation_total
  - Description: Number of key rotation events
  - Type: Counter
  - Labels: old_key_id, new_key_id
```

```
jwe_key_age_days
  - Description: Age of current encryption key in days
  - Type: Gauge
  - Labels: key_id
  - Alert: > 365 days (annual rotation recommended)
```

## Error Metrics

### Encryption Errors

```
jwe_encryption_error_invalid_input
  - Description: Invalid JWT input (malformed, missing claims)
  - Type: Counter
```

```
jwe_encryption_error_key_not_found
  - Description: Encryption key not found
  - Type: Counter
  - Alert: Critical (service degraded)
```

```
jwe_encryption_error_algorithm_failure
  - Description: Cryptographic algorithm failure
  - Type: Counter
  - Alert: Critical (potential security issue)
```

### Decryption Errors

```
jwe_decryption_error_malformed
  - Description: Malformed JWE token (not 5 parts)
  - Type: Counter
```

```
jwe_decryption_error_tampered
  - Description: Token failed integrity check (tampered)
  - Type: Counter
  - Alert: Warning (potential attack)
```

```
jwe_decryption_error_key_mismatch
  - Description: Key ID not found or key mismatch
  - Type: Counter
```

```
jwe_decryption_error_expired_key
  - Description: Decryption attempted with expired key
  - Type: Counter
```

## Alerting Rules

### Critical Alerts

1. **High Encryption Failure Rate**
   ```
   rate(jwe_encryption_failures_total[5m]) > 0.05
   ```
   Action: Check key availability, inspect logs for error patterns

2. **High Decryption Failure Rate**
   ```
   rate(jwe_decryption_failures_total[5m]) > 0.10
   ```
   Action: Check for token tampering, verify key rotation status

3. **Encryption Performance Degradation**
   ```
   histogram_quantile(0.99, jwe_encryption_duration_microseconds) > 500
   ```
   Action: Check CPU utilization, inspect key size (should be 2048-bit RSA)

### Warning Alerts

1. **Key Rotation Overdue**
   ```
   jwe_key_age_days > 365
   ```
   Action: Schedule key rotation

2. **Increasing Decryption Errors**
   ```
   rate(jwe_decryption_error_tampered[1h]) > 10
   ```
   Action: Investigate potential attack, review access logs

## Prometheus Configuration Example

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'gauth-jwe'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

## Grafana Dashboard

### Key Panels

1. **JWE Operations Rate**
   - Metric: `rate(jwe_encryption_total[5m])` + `rate(jwe_decryption_total[5m])`
   - Visualization: Time series

2. **Operation Latency (P50, P95, P99)**
   - Metric: `histogram_quantile(0.50, jwe_encryption_duration_microseconds)`
   - Visualization: Time series with multiple percentiles

3. **Error Rate**
   - Metric: `rate(jwe_encryption_failures_total[5m]) + rate(jwe_decryption_failures_total[5m])`
   - Visualization: Time series with threshold

4. **Token Size Distribution**
   - Metric: `jwe_token_size_bytes`
   - Visualization: Histogram

5. **Key Rotation Status**
   - Metric: `jwe_active_keys`, `jwe_key_age_days`
   - Visualization: Gauge

## Implementation Notes

### Metric Collection Points

1. **Encryption Path** (`EncryptToken` method):
   ```go
   start := time.Now()
   defer func() {
       duration := time.Since(start).Microseconds()
       metrics.RecordJWEEncryption(keyID, algorithm, duration, err == nil)
   }()
   ```

2. **Decryption Path** (`DecryptToken` method):
   ```go
   start := time.Now()
   defer func() {
       duration := time.Since(start).Microseconds()
       metrics.RecordJWEDecryption(keyID, algorithm, duration, err == nil)
   }()
   ```

3. **Key Rotation** (`RotateKeys` method):
   ```go
   metrics.RecordKeyRotation(oldKeyID, newKeyID)
   metrics.SetActiveKeys(len(keys))
   ```

### Integration with Existing Metrics

GAuth already has a metrics infrastructure (`pkg/metrics/`). JWE metrics should be added to the existing Prometheus exporter:

- File: `pkg/metrics/prometheus.go`
- Namespace: `gauth`
- Subsystem: `jwe`

## Future Enhancements

1. **Detailed Error Classification**: Break down error types into subcategories
2. **Geographic Distribution**: Track JWE operations by region/data center
3. **Client Metrics**: Track per-client encryption/decryption rates
4. **Key Usage Heatmap**: Visualize which keys are used most frequently
5. **Compression Efficiency**: Track compression ratios over time

## References

- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [JWE RFC 7516](https://tools.ietf.org/html/rfc7516)
- [GAuth Metrics Package](../../pkg/metrics/)
