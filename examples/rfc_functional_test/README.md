---
title: Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# RFC Functional Test Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates functional validation of RFC-0111 and RFC-0115 compliance in the GAuth framework. It runs a series of automated tests to verify:

- Power-of-Attorney request handling
- Jurisdiction and AI capability validation
- Advanced delegation and attestation
- Legal compliance and audit record creation
- Error handling for invalid requests

## How to Run

```sh
go run main.go
```

## What to Expect
- The output will show a sequence of tests, indicating which RFC requirements are validated and which error cases are correctly handled.
- Successful tests print authorization codes, legal compliance status, and audit record IDs.
- Failed tests indicate missing or incorrect validation logic.

## Beta Notes
- This example is intended for developers and auditors to verify RFC compliance in real scenarios.
- All validation logic is implemented (no stubs).
- The code is documented for clarity and maintainability.

## Last Updated
October 6, 2025

---
For more details, see the main project README and docs/COMPLETE_API_REFERENCE.md.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
