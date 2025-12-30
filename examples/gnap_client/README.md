---
title: Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GNAP Client Example

This example demonstrates how to use the Grant Negotiation and Authorization Protocol (RFC 9635) with AgentAuth.

## Prerequisites

Start the AgentAuth server:

```bash
GAUTH_JWT_SIGNING_KEY=your-secret-key go run ./cmd/web-server
```

## Run the Example

```bash
go run ./examples/gnap_client/main.go
```

## Expected Output

```
=== Step 1: Discovery ===
Grant endpoint: http://localhost:8080/gnap/tx

=== Step 2: Grant Request ===
Access Token: gauth_gnap_xxxxx
Expires In: 3600 seconds

=== Step 3: Use Token ===
Authorization: GNAP gauth_gnap_xxxxx

=== Step 4: Continuation Available ===
Continue URI: http://localhost:8080/gnap/continue/gnt_xxxxx
Continue Token: cnt_xxxxx

✓ GNAP flow complete!
```

## GNAP Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/.well-known/gnap-as-rs` | GET | Discovery metadata |
| `/gnap/tx` | POST | Request a grant |
| `/gnap/continue/:id` | POST | Continue a pending grant |
| `/gnap/continue/:id` | PATCH | Modify a grant request |
| `/gnap/continue/:id` | DELETE | Cancel a grant |
| `/gnap/token/:id` | POST | Rotate a token |
| `/gnap/token/:id` | DELETE | Revoke a token |

## Token Usage

Use the access token in API requests:

```
Authorization: GNAP gauth_gnap_xxxxx
```

## References

- [RFC 9635 - GNAP](https://datatracker.ietf.org/doc/html/rfc9635)
- [RFC 9421 - HTTP Message Signatures](https://datatracker.ietf.org/doc/html/rfc9421)
