---
title: Directory Reorganization
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Directory Reorganization Guide

This document describes the directory structure reorganization implemented to improve code organization and scalability.

## Changes Made

### 1. Frontend Code Reorganization

**Previous Structure:**
```
web/
  ├── ui-react/          # React application
  ├── static/            # Static assets (CSS, JS)
  ├── templates/         # HTML templates
  └── swagger_ui/        # Swagger UI files
```

**New Structure:**
```
frontend/
  ├── ui-react/          # React application
  ├── static/            # Static assets (CSS, JS)
  ├── templates/         # HTML templates
  └── swagger_ui/        # Swagger UI files

web/
  ├── handlers/          # HTTP handlers (unchanged)
  ├── middleware/        # Middleware (unchanged)
  ├── server.go          # Main server (updated paths)
  └── *_test.go          # Web integration tests (unchanged)
```

**Rationale:** Separates frontend assets from backend code, making it clearer that `web/` contains backend HTTP handling logic while `frontend/` contains UI assets.

### 2. Integration Tests Reorganization

**Previous Structure:**
```
pkg/
  └── [package]/
      └── integration_test.go
web/
  └── *_integration_test.go
```

**New Structure:**
```
test/
  └── integration/
      └── pkg/
          └── [package]/
              └── integration_test.go
```

**Rationale:** Centralizes integration tests in a dedicated location, making it easier to run all integration tests together and distinguishing them from unit tests.

### 3. Policy Store Implementation (New)

**Added Files:**
- `pkg/gauth/policy_store.go` - PolicyStore interface and in-memory implementation
- `pkg/gauth/policy_store_db.go` - PostgreSQL-backed implementation

**Rationale:** Addresses the in-memory scaling limitation by introducing a pluggable storage interface that supports both in-memory (for development/testing) and persistent storage (for production).

## Updated Configuration Files

### Makefile
Updated JavaScript build targets to reference `frontend/` instead of `web/`:
- `JS_FILES` path: `web/static/js` → `frontend/static/js`
- `js-build` target: Updated bundle path references

### Go Embed Directives (web/server_clean.go)
Updated embed directives to reference the new frontend location:
```go
//go:embed ../frontend/templates/index.html
//go:embed ../frontend/static/css/style.css
//go:embed ../frontend/static/js/app.js
// ... etc
```

### Task Definitions (.vscode/tasks.json - if present)
Update any task definitions that reference `web/ui-react` to `frontend/ui-react`.

## Migration for Developers

### If You Have Local Changes

1. **Frontend Changes:**
   ```bash
   # Your changes in web/ui-react, web/static, etc. need to move
   cd /path/to/Gauth_go
   
   # Frontend assets are now in frontend/
   # Example: Edit frontend/ui-react/src/... instead of web/ui-react/src/...
   ```

2. **Integration Tests:**
   ```bash
   # Integration tests are now copied to test/integration/
   # Original files in pkg/ remain for backward compatibility
   # New integration tests should go in test/integration/
   ```

3. **Policy Storage:**
   ```go
   // Old way (direct in-memory storage)
   pap := gauth.NewPowerAdministrationPoint("id", "name", "desc")
   
   // New way (default in-memory)
   pap := gauth.NewPowerAdministrationPoint("id", "name", "desc")
   
   // New way (with custom store, e.g., PostgreSQL)
   dbStore, err := gauth.NewDatabasePolicyStore(db)
   if err != nil {
       log.Fatal(err)
   }
   pap := gauth.NewPowerAdministrationPointWithStore("id", "name", "desc", dbStore)
   ```

### Running Tests

```bash
# Unit tests (unchanged)
make test

# Integration tests (new location)
go test ./test/integration/...

# All tests
make test-all
```

### Building

```bash
# Build with updated frontend paths
make build

# Build JavaScript assets
make js-build
```

## Benefits

1. **Clearer Separation:** Backend (web/) vs Frontend (frontend/) vs Tests (test/)
2. **Better Scalability:** PolicyStore interface enables production-grade persistence
3. **Improved Organization:** Integration tests centralized for easier management
4. **Maintainability:** Easier to understand project structure for new contributors

## Backward Compatibility

- Original integration test files in `pkg/` directories are preserved
- Copies exist in `test/integration/` for centralized test execution
- All existing APIs remain unchanged
- PolicyStore uses in-memory storage by default (same behavior as before)

## Future Considerations

1. **Complete Migration:** Eventually remove duplicate integration tests from `pkg/` directories
2. **Additional Storage Backends:** Redis, S3, or other backends for PolicyStore
3. **Frontend Build Pipeline:** Consider moving to a separate repository if frontend grows significantly
4. **Test Organization:** Further categorize tests (e2e, smoke, performance) under `test/`

## Questions or Issues?

If you encounter any issues with the reorganized structure:
1. Check that your Go workspace is clean (`go mod tidy`)
2. Rebuild assets (`make js-build`)
3. Verify paths in any custom scripts or configurations
4. Consult this guide for the new structure

## Related Documentation

- [API Reference](API_REFERENCE.md)
- [OAuth 2.0 Migration Feasibility Study](docs/OAUTH_2_MIGRATION_FEASIBILITY_STUDY.md)
- [P*P User Guide](docs/P_STAR_P_USER_GUIDE.md)
