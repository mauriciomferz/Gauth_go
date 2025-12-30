---
title: Saml Scim Implementation Plan
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# SAML & SCIM Reference Implementation Plan

## Goal
Extend the AgentAuth platform to support **SAML 2.0** (as a Service Provider) and **SCIM 2.0** (for user provisioning), enabling enterprise integrations.

## User Review Required
> [!IMPORTANT]
> This is a **Reference Implementation**.
> - **SAML**: Will verify signatures and assertions but may skip advanced features like encryption or complex attribute mapping initially.
> - **SCIM**: Will support core `/Users` endpoints (Create, Read, Update, Delete) compliant with RFC 7643/7644, but simplified.

## Proposed Changes

### Database Schema
#### [NEW] `schema/migrations/012_create_saml_tables.sql`
- `saml_providers`: Stores Identity Provider (IdP) configurations.
    - `entity_id` (Issuer)
    - `sso_url` (Single Sign-On Endpoint)
    - `certificate` (X.509 Certificate for verification)
    - `attribute_mapping` (JSON)

#### [NEW] `schema/migrations/013_create_scim_tables.sql`
- `scim_clients`: (Optional) if strictly needed, otherwise we can leverage existing API keys with a new `scim_access` scope.
- *Note*: SCIM will primarily interact with existing `users` and `groups` tables.

### Backend (`pkg/`)
#### [NEW] `pkg/saml`
- `repository.go`: CRUD for `saml_providers`.
- `service.go`: Logic to generate `AuthnRequest` and validate `SAMLResponse`.
- `handlers.go`:
    - `POST /api/saml/acs` (Assertion Consumer Service)
    - `GET /api/saml/metadata` (Service Provider Metadata)

#### [NEW] `pkg/scim`
- `models.go`: Add `SCIMClient` struct matching `scim_clients` table.
- `repository.go`: Add CRUD methods for `scim_clients`.
- `service.go`: Add wrappers for Client management.
- `handler.go`:
    - `GET /api/scim/v2/Users` (Existing)
    - `...` (Existing User endpoints)
    - `GET /api/v1/admin/scim/clients` (List Clients)
    - `POST /api/v1/admin/scim/clients` (Create Client - Generate Token)
    - `DELETE /api/v1/admin/scim/clients/:id` (Revoke Client)

### Admin API (`web/handlers/admin`)
#### [MODIFY] `web/router.go`
- Register new routes.

#### [NEW] `web/handlers/admin/saml_handler.go`
- CRUD endpoints for configuring SAML Providers via Admin UI.

### Frontend (`ui-react`)
#### [MODIFY] `src/pages/admin/`
- **SAML Providers**: New page to list/add/edit SAML IdP configurations.
- **SCIM Settings**: Page to view SCIM Base URL and generate Tokens.

## Verification Plan

### Automated Tests
- Unit tests for SAML Request generation and Response validation (using mock data).
- Curl tests for SCIM endpoints (`POST /api/scim/v2/Users`).

### Manual Verification
- **SAML**: Mock an IdP login flow (or use a tool like SAMLTest.id if accessible, otherwise mock the POST).
- **SCIM Users**: Use `curl` to create a user and verify it appears in the database.
- **SCIM Clients**: Use Browser Subagent to go to `/admin/scim-settings`, create a new client, and verify it appears in the list.
