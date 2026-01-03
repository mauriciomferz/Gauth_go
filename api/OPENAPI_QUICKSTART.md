---
title: Openapi Quickstart
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AgentAuth+ OpenAPI Documentation - Quick Reference

## 📚 Files in this Directory

- **openapi.yaml** - Complete OpenAPI 3.0 specification (37+ endpoints)
- **swagger-ui.html** - Interactive API documentation viewer
- **README.md** - Original API directory documentation

## 🚀 View Interactive Docs

```bash
# Serve locally
python3 -m http.server 8000
open http://localhost:8000/swagger-ui.html
```

## 📋 Endpoint Summary

**37+ Total Endpoints** = 27 Authenticated + 10 Public

### Categories
1. **Proof of Authorization** (8) - Core CRUD operations
2. **Advanced Features** (11) - Successor, Delegation, Dual Control, Capability, Fiduciary
3. **Public Blockchain** (10) - No auth verification endpoints 🔓
4. **Admin & Monitoring** (8) - System management

## 🔑 Quick Start

### Authenticated Request
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/poa/{id}
```

### Public Request (No Auth)
```bash
curl http://localhost:8080/api/v1/public/blockchain/verify/poa-123
```

## 📦 Generate Client SDKs

```bash
npm install @openapitools/openapi-generator-cli -g

# TypeScript
openapi-generator-cli generate -i openapi.yaml -g typescript-axios -o clients/ts

# Python
openapi-generator-cli generate -i openapi.yaml -g python -o clients/python

# Go
openapi-generator-cli generate -i openapi.yaml -g go -o clients/go
```

---

**See `openapi.yaml` for full specification**
