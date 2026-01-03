# AgentAuth+ Verification Service Architecture

The `agentauthplus` package provides a high-level verification service that aggregates multiple sources of truth to determine the validity of a Proof of Authorization (PoA) and the entities involved.

## Overview

The `VerificationService` is the primary entry point for verifying authorizations. It coordinates:
- **PoA Validity**: Checks signatures, expiration, and revocation status.
- **Scope Narrowing**: Ensures requested actions fall within the PoA's granted scope.
- **Principal Verification**: Distinguishes between human and AI entities using the `PrincipalVerifier`.
- **Representative Verification**: Validates corporate positions and signing authority via the `CommercialRegisterService`.
- **Compliance Checks**: Validates fiduciary duties and AI capability assessments.

## Attestation Proof Generation (Phase 11)

One of the key features of AgentAuth+ is the ability to issue **Authoritative Attestations**. When the service verifies a condition (e.g., that a representative is indeed authorized by their organization), it can generate a cryptographically signed proof.

### Components
- **`AttestationSigner`**: Interface for creating Ed25519 signatures over attestation data.
- **`AttestationVerifier`**: Interface for validating those signatures, allowing AgentAuth+ to verify its own proofs locally.
- **`VerificationReport`**: Aggregates all individual attestations into a single, verifiable package for relying parties.

### Workflow
1. A verification request is received (e.g., via GNAP).
2. The `VerificationService` performs internal checks (PoA store, commercial register, etc.).
3. If a check succeeds, the `AttestationSigner` creates a proof.
4. The proof is attached to the result (`PrincipalStatusResult`, `PositionVerificationResult`).
5. The `VerificationReport` collects all proofs and presents a summary of "Overall Validity".

## Storage & Persistence
The service uses a PostgreSQL backend (via `pgx`) for:
- `PoAStore`: Durable storage of Proof of Authorization records.
- `FiduciaryDutyService`: Tracking violations and compliance.
- `CapabilityAssessmentService`: Storing AI agent evaluations.

## Security Considerations
- **Signature Algorithms**: Currently supports Ed25519 for attestation.
- **Key Management**: Signer keys should be managed via a secure KMS (integrated in `server_factory.go`).
- **Fail-Closed Design**: Verification fails if critical services (like the PoA store) are unavailable.
