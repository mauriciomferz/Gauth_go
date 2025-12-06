# Web Server Refactoring Roadmap

> **Status**: Documented | **Priority**: Medium | **Est. Effort**: 8-12 hours

## Overview

`web/server_clean.go` contains **12,298 lines** with **152 methods**. While functional, this monolithic file should be split for better maintainability.

## Current Handler Structure

Existing extractions in `web/handlers/`:
- `admin/` (17 files)
- `anchor/` (3 files)
- `audit/` (2 files)
- `auth/` (2 files)
- `beta/` (4 files)
- `capabilities/` (2 files)
- `disclosure/` (1 file)
- `docs/` (3 files)
- `externalreceipts/` (2 files)
- `gauthplus/` (5 files)
- `mcp/` (1 file)
- `rfc0111/` (2 files)

## Remaining Handler Groups in server_clean.go

### 1. Model Limits (~1,500 lines)
**Target**: `web/handlers/model_limits/`

- `loadModelLimitsFromDisk`, `modelLimitsReloader`
- `computeModelLimitsSnapshot`, `apiModelLimitsSnapshot`
- `apiModelValidate`
- `writeModelLimitAudit`, `recordModelLimitExceed`
- `apiModelLimitAudit*`, `apiModelLimitsAttestation*`

### 2. Semantic Anomaly Detection (~800 lines)
**Target**: `web/handlers/semantic/`

- `initSemanticAnomaly`, `SemanticAnomalyStats`
- `apiSemanticCounters*`
- `updateSemanticAnomalies`, `currentSemanticScores`
- `semanticRatesForWindows`
- Persistence: `loadSemanticPersistence`, `saveSemanticPersistence`

### 3. Violation Tracking (~500 lines)
**Target**: `web/handlers/violations/`

- `violationRatesForWindows`
- `apiViolationMetrics*`, `apiViolationPersistenceVerify`
- Persistence: `loadViolationPersistence`, `saveViolationPersistence`

### 4. Capability Anchoring (~400 lines)
**Target**: `web/handlers/capability_anchor/`

- `apiCapabilityAnchor*`
- `apiExternalAnchorReceipt*`
- `apiNotarizationReceipt*`
- `apiCapabilityAnchorPrometheus`

## Migration Steps (Future)

1. **Phase A**: Create handler files with struct holding `*BetaServer`
2. **Phase B**: Move methods one group at a time, update route registration
3. **Phase C**: Remove old methods from `server_clean.go`
4. **Phase D**: Rename `server_clean.go` → `server.go`

## Why Deferred

- High risk of regressions without comprehensive e2e tests
- Route registration tightly coupled to BetaServer
- Backend/frontend currently working correctly
- Better handled as dedicated sprint/epic
