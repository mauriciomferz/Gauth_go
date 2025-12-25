---
title: Rfc9767 Gnap Rs Connections
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# RFC 9767: GNAP Resource Server Connections

**Status**: Implemented (Experimental/Mocked)
**Extends**: RFC 9635 (GNAP)

## Overview
RFC 9767 defines standard methods for Resource Servers (RS) to connect with Authorization Servers (AS) within the GNAP ecosystem.

## Features Implemented
*   [x] **Dynamic RS Registration**: `POST /gnap/rs/register` allows RSs to register and receive an instance ID.
*   [x] **RS-Specific Introspection**: `POST /gnap/rs/introspect` allows RSs to validate tokens and receive extended metadata (including PoA references).

## Endpoints

### 1. RS Registration (`POST /gnap/rs/register`)
Accepts `ResourceServerRequest` with client key material. Returns `instance_id`.

### 2. RS Introspection (`POST /gnap/rs/introspect`)
Accepts a token and validates it. Returns `IntrospectionResponse` with `active` status and `poa_id` if applicable.
