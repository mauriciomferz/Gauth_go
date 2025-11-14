---
title: JWKS Integrity & Key Deprecation Guide
category: security-jwks
status: implemented
lastUpdated: 2025-11-12
owners: security-team
source: internal
refreshCadence: quarterly
---
# JWKS Integrity & Key Deprecation

**Status**: Implemented (P2.4)  
**GAP Requirement**: sec10.item2 - Well-known discovery endpoints with JWKS integrity signature & structured deprecation metadata  
**Version**: 1.0  
**Last Updated**: 2025-11-05

## Overview

GAuth implements RFC0115-compliant JWKS integrity mechanisms to protect key distribution and provide structured key lifecycle signals to clients. This includes:

1. **Integrity Signatures**: Optional HMAC-SHA256 signatures on JWKS responses (X-JWKS-Signature header)
2. **Deprecation Metadata**: Structured key lifecycle timestamps (`deprecated_after`, `sunset_after`)
3. **Warning Headers**: HTTP Warning headers when deprecated keys are active
4. **ETag Support**: Conditional requests with If-None-Match for bandwidth optimization

## Architecture

### Key Lifecycle Phases

```
Key Creation
    ↓
Active Phase (0%-80% of TTL)
    ↓
Deprecation Warning (80%-100% of TTL)  ← deprecated_after timestamp
    ↓
Sunset / Expiration (100% of TTL)      ← sunset_after timestamp
```

### Deprecation Timestamps

**DeprecatedAfter** (RFC0115 §6.2.1 "Key Deprecation Warning"):
- Signals clients to begin rotating to newer keys
- Computed as `CreatedAt + (TTL * 0.8)` (80% of key lifetime)
- Keys remain valid for verification but should not be used for new signatures
- Included in JWKS response as `deprecated_after` (RFC3339 format)

**SunsetAfter** (RFC0115 §6.2.2 "Key Sunset"):
- Hard cutoff timestamp when key expires
- Equals `ExpiresAt` (100% of key lifetime)
- Included in JWKS response as `sunset_after` (RFC3339 format)
- After sunset, key is removed from JWKS and signature verification fails

**Example Timeline** (24-hour TTL):
```
00:00 - Key created (CreatedAt)
19:12 - Deprecation warning begins (DeprecatedAfter = 80% of 24h)
        → Warning: 299 - "Keys deprecated: <kid>" header appears
        → Clients should rotate to newer keys
24:00 - Key sunset (SunsetAfter = ExpiresAt)
        → Key removed from JWKS
        → Signature verification fails
```

## JWKS Response Schema

### EdDSA (OKP) Key Entry

```json
{
  "keys": [
    {
      "kty": "OKP",
      "crv": "Ed25519",
      "alg": "EdDSA",
      "kid": "jzABkhZzkp0",
      "use": "sig",
      "x": "jzABkhZzkp3k4Liump5HGOL_jOcz9vtkSUB00r2VPPA",
      "expires_at": "2025-11-06T17:36:59Z",
      "deprecated_after": "2025-11-06T13:50:23Z",
      "sunset_after": "2025-11-06T17:36:59Z"
    }
  ]
}
```

### Field Descriptions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `kty` | string | Yes | Key type (`OKP` for EdDSA) |
| `crv` | string | Yes | Curve name (`Ed25519`) |
| `alg` | string | Yes | Algorithm (`EdDSA`) |
| `kid` | string | Yes | Key ID (base64url-encoded public key prefix) |
| `use` | string | Yes | Public key use (`sig` for signature) |
| `x` | string | Yes | Base64url-encoded public key |
| `expires_at` | string | Yes | RFC3339 timestamp when key expires |
| `deprecated_after` | string | Optional | RFC3339 timestamp when deprecation warning begins |
| `sunset_after` | string | Optional | RFC3339 timestamp for hard cutoff (same as `expires_at`) |

## HTTP Headers

### ETag (Cache Validation)

```http
GET /.well-known/jwks.json HTTP/1.1

HTTP/1.1 200 OK
ETag: W/"a3f2c8e..."
Cache-Control: public, max-age=60
```

**Conditional Request**:
```http
GET /.well-known/jwks.json HTTP/1.1
If-None-Match: W/"a3f2c8e..."

HTTP/1.1 304 Not Modified
```

### Warning Header (Deprecation Signal)

When any key is past `deprecated_after` but before `expires_at`:

```http
HTTP/1.1 200 OK
Warning: 299 - "Keys deprecated: jzABkhZzkp0, aB3CdEfGhI8"
```

**Format**: RFC 7234 Warning header (code 299 = miscellaneous persistent warning)

### JWKS Signature (Optional)

Enable with:
```bash
export GAUTH_JWKS_SIGNING_KEY="your-secret-key"
export GAUTH_JWKS_SIGNING_KEY_ENABLED="1"
```

Response headers:
```http
X-JWKS-Signature: <base64url-encoded-HMAC-SHA256>
X-JWKS-Signature-Alg: HMAC-SHA256
```

**Verification** (client-side):
```python
import hmac
import hashlib
import base64

def verify_jwks(jwks_json, signature_b64, secret):
    mac = hmac.new(secret.encode(), jwks_json.encode(), hashlib.sha256)
    expected = base64.urlsafe_b64encode(mac.digest()).rstrip(b'=')
    return expected.decode() == signature_b64
```

## Client Migration Guide

### Phase 1: Deprecation Awareness (Immediate)

**Objective**: Start monitoring deprecation signals without behavioral changes.

```python
import requests
import json
from datetime import datetime

def fetch_jwks_with_deprecation_check():
    resp = requests.get("https://auth.example.com/.well-known/jwks.json")
    
    # Check for deprecation warning
    if 'Warning' in resp.headers:
        print(f"ALERT: {resp.headers['Warning']}")
        # Log to monitoring system
        log_deprecation_warning(resp.headers['Warning'])
    
    jwks = resp.json()
    for key in jwks['keys']:
        if 'deprecated_after' in key:
            deprecated_at = datetime.fromisoformat(key['deprecated_after'].replace('Z', '+00:00'))
            sunset_at = datetime.fromisoformat(key['sunset_after'].replace('Z', '+00:00'))
            now = datetime.now(deprecated_at.tzinfo)
            
            if now > deprecated_at:
                days_until_sunset = (sunset_at - now).total_seconds() / 86400
                print(f"Key {key['kid']} deprecated! Sunset in {days_until_sunset:.1f} days")
    
    return jwks
```

### Phase 2: Proactive Rotation (Week 2)

**Objective**: Automatically rotate to non-deprecated keys for new signatures.

```python
def select_signing_key(jwks):
    """Select best key for signing (prefer non-deprecated)."""
    now = datetime.now(datetime.timezone.utc)
    
    # Filter to EdDSA keys
    eddsa_keys = [k for k in jwks['keys'] if k.get('alg') == 'EdDSA']
    
    # Prioritize non-deprecated keys
    active_keys = []
    deprecated_keys = []
    
    for key in eddsa_keys:
        if 'deprecated_after' in key:
            deprecated_at = datetime.fromisoformat(key['deprecated_after'].replace('Z', '+00:00'))
            if now < deprecated_at:
                active_keys.append(key)
            else:
                deprecated_keys.append(key)
        else:
            active_keys.append(key)  # No deprecation metadata = active
    
    # Prefer active keys, fall back to deprecated if necessary
    if active_keys:
        return active_keys[0]  # Use newest active key
    elif deprecated_keys:
        print("WARNING: Only deprecated keys available!")
        return deprecated_keys[0]
    else:
        raise Exception("No signing keys available")
```

### Phase 3: Strict Enforcement (Week 4)

**Objective**: Reject signatures from keys past `deprecated_after`.

```python
def verify_signature_strict(kid, signature, payload, jwks):
    """Verify signature with strict deprecation policy."""
    now = datetime.now(datetime.timezone.utc)
    
    key = next((k for k in jwks['keys'] if k['kid'] == kid), None)
    if not key:
        raise Exception(f"Unknown kid: {kid}")
    
    # Check deprecation status
    if 'deprecated_after' in key:
        deprecated_at = datetime.fromisoformat(key['deprecated_after'].replace('Z', '+00:00'))
        if now > deprecated_at:
            raise Exception(f"Key {kid} is deprecated (since {deprecated_at})")
    
    # Check sunset
    if 'sunset_after' in key:
        sunset_at = datetime.fromisoformat(key['sunset_after'].replace('Z', '+00:00'))
        if now > sunset_at:
            raise Exception(f"Key {kid} expired (sunset: {sunset_at})")
    
    # Perform cryptographic verification
    return verify_ed25519_signature(key['x'], signature, payload)
```

## Operational Guide

### Recommended TTL & Deprecation Settings

| Environment | Key TTL | Deprecation Warning | Sunset Buffer |
|-------------|---------|---------------------|---------------|
| **Development** | 1 hour | 48 minutes (80%) | 12 minutes (20%) |
| **Staging** | 24 hours | 19.2 hours (80%) | 4.8 hours (20%) |
| **Production** | 7 days | 5.6 days (80%) | 1.4 days (20%) |

**Rationale**:
- **80% deprecation threshold**: Provides sufficient warning period (20% of TTL) for clients to rotate
- **Production 7-day TTL**: Balances security (frequent rotation) with operational stability (clients have >1 day to adopt new keys)

### Configuration

```bash
# Key rotation schedule (automatic background rotation)
export GAUTH_EDDSA_ROTATION_INTERVAL_MINS=10080  # 7 days

# Key TTL (lifetime per key)
export GAUTH_EDDSA_TTL_HOURS=168  # 7 days

# Optional JWKS signing (integrity protection)
export GAUTH_JWKS_SIGNING_KEY="production-jwks-secret-2025"
export GAUTH_JWKS_SIGNING_KEY_ENABLED="1"
```

### Monitoring Metrics

**Prometheus Queries**:

```promql
# Keys in deprecation warning period
count(time() - crypto_key_created_timestamp_seconds > crypto_key_ttl_seconds * 0.8 
      and time() < crypto_key_expired_timestamp_seconds)

# Keys past sunset (stale keys)
count(time() > crypto_key_expired_timestamp_seconds)

# Clients fetching JWKS with deprecated keys
rate(http_requests_total{path="/.well-known/jwks.json",status="200",warning_header="true"}[5m])
```

**Alert Rules**:

```yaml
groups:
  - name: jwks_deprecation
    rules:
      - alert: JWKSOnlyDeprecatedKeys
        expr: |
          count(time() < crypto_key_deprecated_timestamp_seconds) == 0
          and count(crypto_key_deprecated_timestamp_seconds) > 0
        for: 15m
        annotations:
          summary: "All JWKS keys are deprecated (no active keys)"
          description: "Clients may start experiencing signature failures within {{ $labels.sunset_hours }}h"
      
      - alert: JWKSHighDeprecationRate
        expr: rate(http_response_warning_total{endpoint="jwks"}[10m]) > 0.5
        annotations:
          summary: "High rate of JWKS responses with deprecation warnings"
          description: ">50% of JWKS fetches return deprecated keys ({{ $value }}/s)"
```

### Client Best Practices

1. **Cache JWKS with ETag**: Reduce bandwidth and server load
   ```python
   etag_cache = {}
   
   def fetch_jwks_cached(url):
       etag = etag_cache.get(url)
       headers = {'If-None-Match': etag} if etag else {}
       resp = requests.get(url, headers=headers)
       
       if resp.status_code == 304:
           return etag_cache[f"{url}_body"]
       
       etag_cache[url] = resp.headers.get('ETag')
       etag_cache[f"{url}_body"] = resp.json()
       return resp.json()
   ```

2. **Respect Cache-Control**: Default `max-age=60` (1 minute)
3. **Monitor Warning Headers**: Log and alert on deprecation warnings
4. **Pre-fetch on Deprecation**: When `deprecated_after` approaches, proactively fetch updated JWKS
5. **Graceful Degradation**: Accept deprecated keys for verification (backward compatibility) but reject for new signatures

## Examples

### Example 1: Basic JWKS Fetch

```bash
curl https://localhost:8080/.well-known/jwks.json

{
  "keys": [
    {
      "kty": "OKP",
      "crv": "Ed25519",
      "alg": "EdDSA",
      "kid": "jzABkhZzkp0",
      "use": "sig",
      "x": "jzABkhZzkp3k4Liump5HGOL_jOcz9vtkSUB00r2VPPA",
      "expires_at": "2025-11-12T17:36:59Z",
      "deprecated_after": "2025-11-11T13:50:23Z",
      "sunset_after": "2025-11-12T17:36:59Z"
    }
  ]
}
```

### Example 2: JWKS with Deprecation Warning

```bash
curl -i https://localhost:8080/.well-known/jwks.json

HTTP/1.1 200 OK
Content-Type: application/json
ETag: W/"a3f2c8e9..."
Cache-Control: public, max-age=60
Warning: 299 - "Keys deprecated: jzABkhZzkp0"

{
  "keys": [...]
}
```

### Example 3: Conditional Request (304 Not Modified)

```bash
curl -i -H 'If-None-Match: W/"a3f2c8e9..."' https://localhost:8080/.well-known/jwks.json

HTTP/1.1 304 Not Modified
ETag: W/"a3f2c8e9..."
```

### Example 4: JWKS with Integrity Signature

```bash
curl -i https://localhost:8080/.well-known/jwks.json

HTTP/1.1 200 OK
X-JWKS-Signature: jzABkhZzkp3k4Liump5HGOL_jOcz9vtkSUB00r2VPPA
X-JWKS-Signature-Alg: HMAC-SHA256

{
  "keys": [...]
}
```

**Client Verification**:
```python
import hmac, hashlib, base64, requests

resp = requests.get("https://localhost:8080/.well-known/jwks.json")
secret = "your-shared-secret"

mac = hmac.new(secret.encode(), resp.content, hashlib.sha256)
expected_sig = base64.urlsafe_b64encode(mac.digest()).rstrip(b'=').decode()
actual_sig = resp.headers['X-JWKS-Signature']

if expected_sig == actual_sig:
    print("JWKS signature valid!")
else:
    print("WARNING: JWKS signature mismatch!")
```

## Security Considerations

1. **Deprecation ≠ Revocation**: Deprecated keys remain valid for verification (backward compatibility). Use revocation for compromised keys.

2. **Warning Period Sizing**: 20% TTL (default 80% deprecation threshold) balances:
   - Too short: Clients may miss rotation window → signature failures
   - Too long: Extended exposure if key compromised during deprecation

3. **JWKS Signature Secrets**: 
   - Rotate `GAUTH_JWKS_SIGNING_KEY` independently from signing keys
   - Use strong secrets (≥256 bits entropy)
   - Store securely (environment variables, not hardcoded)

4. **Sunset Enforcement**: Clients MUST reject signatures after `sunset_after` (hard security boundary)

5. **Clock Skew**: Allow ±5 minutes tolerance for deprecation checks (NTP drift)

## Migration from Non-Deprecation JWKS

### Step 1: Server-Side Deployment (Week 1)

Deploy GAuth with deprecation metadata (backward compatible):

```bash
# Existing JWKS consumers see new fields (ignored if not supported)
curl /.well-known/jwks.json
# Response includes deprecated_after & sunset_after (optional fields)
```

### Step 2: Client Monitoring (Week 2-3)

Update clients to log deprecation warnings:

```python
# Phase 1: Monitoring only (no behavioral changes)
jwks = fetch_jwks_with_deprecation_check()  # Logs warnings
```

### Step 3: Client Rotation Logic (Week 4-5)

Implement proactive rotation on deprecation signal:

```python
# Phase 2: Rotate to non-deprecated keys for new signatures
key = select_signing_key(jwks)  # Prefers active keys
```

### Step 4: Strict Enforcement (Week 6+)

Reject new signatures from deprecated keys:

```python
# Phase 3: Fail if signing with deprecated key
if now > key['deprecated_after']:
    raise Exception("Cannot sign with deprecated key")
```

## Testing

### Unit Tests

```bash
# Run JWKS integrity tests
go test -v -run "TestJWKS" ./web

# Expected output:
# PASS: TestJWKSDiscoveryMetadata
# PASS: TestJWKSConditionalETag
# PASS: TestJWKSOptionalSignature
# PASS: TestJWKSDeprecationMetadata
# PASS: TestJWKSDeprecationWarningHeader
# PASS: TestJWKSNoWarningWhenNoDeprecation
```

### Integration Test (Deprecation Lifecycle)

```bash
# Start server with 10-second TTL (rapid deprecation)
export GAUTH_TOKEN_SIG_MODE=eddsa
export GAUTH_EDDSA_TTL_HOURS=0.00277  # 10 seconds
./bin/web-server

# T+0s: Fetch JWKS (fresh key, no warning)
curl -i localhost:8080/.well-known/jwks.json
# Expected: No Warning header, deprecated_after in future

# T+9s: Fetch JWKS (deprecated key, warning present)
sleep 9
curl -i localhost:8080/.well-known/jwks.json
# Expected: Warning: 299 - "Keys deprecated: <kid>"

# T+11s: Fetch JWKS (key expired, removed from JWKS)
sleep 2
curl localhost:8080/.well-known/jwks.json
# Expected: No keys in response (or new key if rotation occurred)
```

## Future Enhancements

1. **Multi-Tenant Deprecation Policies**: Per-tenant TTL/deprecation thresholds
2. **Gradual Rollout**: Deprecate subsets of keys (A/B testing)
3. **External Notarization**: Anchor JWKS snapshots to transparency logs (RFC 6962)
4. **Deprecation Reasons**: Extend metadata with deprecation_reason field (e.g., "key-rotation", "algorithm-upgrade")
5. **Automated Client Alerts**: Push notifications when deprecation begins (WebSub)

## References

- RFC 7517: JSON Web Key (JWK)
- RFC 7234: HTTP Caching (Warning header)
- RFC 0115 §6.2: Key Lifecycle Management
- NIST SP 800-57: Cryptographic Key Management Guidance

---

**Implemented**: P2.4 (2025-11-05)  
**Authors**: GAuth Core Team  
**Reviewers**: Security Team, Platform Engineering
