# GAuth Protocol Basics – Minimal Examples

> Last Updated: 2025-10-17
> Status: Active


This directory contains clear, concise Go examples demonstrating the core GAuth protocol flows:

- **Power of Attorney (POA) Request/Response**
- **Advanced POA (RFC 111) Scenarios**
- **Delegation (RFC 115) Request/Response**
- **Token Creation and Validation**

Each example is self-contained and annotated for Beta demonstration clarity.

---


## 1. Power of Attorney (POA) Flow

- Shows how a client requests a POA and receives a response.

## 1a. Advanced POA (RFC 111) Scenarios

- Demonstrates negative and positive cases for RFC 111 POA validation:
	- Invalid jurisdiction (should fail)
	- Disallowed scope (should fail)
	- Missing required fields (should fail)
	- Valid advanced POA (should succeed)

## 2. Delegation Flow

- Demonstrates how a principal delegates authority to another entity.

## 3. Token Creation & Validation

- Illustrates how to create and validate a token using the GAuth service.

---


Run any example with:

```bash
cd examples/gauth_protocol_basics
# For Minimal POA
go run minimal_poa
# For Advanced POA (RFC 111)
go run advanced_poa
# For Delegation
go run delegation
# For Token
go run token
```

---

For more, see the main [README](../../README.md) and [docs/LOG_STREAMING.md](../../docs/LOG_STREAMING.md).

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
