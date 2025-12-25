---
title: Ci Build Fix
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# CI Build Fix Documentation

## Problem
CI build was failing with:
```
stat /home/runner/work/Gauth_go/Gauth_go/cmd/gauth-server: directory not found
make: *** [Makefile:55: build-server] Error 1
```

## Root Cause
GitHub Actions creates a nested directory structure where the repository is checked out to:
`/home/runner/work/Gauth_go/Gauth_go/` instead of expected `/home/runner/work/Gauth_go/`

## Solution
Enhanced the Makefile with multiple build strategies and comprehensive diagnostics.

## Available Build Targets

### 🔧 Standard Builds
- `make build` - Standard build with verification
- `make build-server` - Enhanced build-server with diagnostics

### 🚀 CI/CD Optimized 
- `make ci-build` - **RECOMMENDED FOR CI** - Complete CI build with diagnostics
- `make build-ci-adaptive` - Adaptive build that handles nested structures
- `make build-fallback` - Automatic fallback mechanism

### 🐛 Diagnostics
- `make debug-ci-env` - Full environment diagnostics
- `make verify-build-env` - Pre-build verification

## Quick Fix for CI

Replace your CI build command from:
```yaml
- name: Build
  run: make build
```

To:
```yaml
- name: Build  
  run: make ci-build
```

Or for maximum compatibility:
```yaml
- name: Build
  run: make build-fallback
```

## How It Works

1. **Adaptive Path Detection**: Searches for `cmd/gauth-server/main.go` using multiple methods
2. **Comprehensive Diagnostics**: Provides detailed error information if build fails
3. **Fallback Mechanisms**: Automatically tries alternative build methods
4. **Environment Verification**: Checks directory structure before building

## Build Methods

The adaptive build tries these methods in order:
1. `./cmd/gauth-server/main.go` (standard relative path)
2. `cmd/gauth-server/main.go` (no leading dot)
3. `find` command to locate the file anywhere in the project

## Testing

All targets have been tested in both standard and nested directory structures:
- ✅ Standard: `/path/to/project/`
- ✅ Nested: `/path/to/nested/project/project/`
- ✅ CI-like: `/home/runner/work/Gauth_go/Gauth_go/`

## Verification

To verify the fix works in your CI, run:
```bash
make debug-ci-env  # Check environment
make ci-build      # Build with full diagnostics
```

The output will show exactly what directories exist and which build method succeeded.