# Architecture & Flow Diagram Placeholders

These Mermaid placeholders outline key flows for the upcoming demo. Replace or refine with actual identifiers once final.

## 1. Rotation Summary Multi-Signature Flow
```mermaid
flowchart LR
    A[Start Rotation Event] --> B[Update Ledger Entry]
    B --> C[Compute Canonical Payload]
    C --> D[Collect Active Ed25519 Keys]
    D --> E[Sign Payload (each key)]
    E --> F[Aggregate Signatures + SatisfiedWeight]
    F --> G{Threshold Met?}
    G -- Yes --> H[Publish /api/v1/beta/rotations/summary]
    G -- No --> I[Error rotation_threshold_unsatisfied]
```

## 2. Model Limits Attestation & Verification
```mermaid
sequenceDiagram
    participant Client
    participant Server
    Client->>Server: GET /model/limits/attestation
    Server->>Server: Build unsigned attestation
    Server->>Server: Prefix + Sign (GAUTH_MODEL_LIMIT_ATTEST:)
    Server-->>Client: JSON attestation + signature
    Client->>Server: POST /model/limits/attestation/verify (attestation)
    Server->>Server: Rebuild unsigned + prefix
    Server->>Server: Verify Ed25519 signature
    Server-->>Client: {valid:true, combined_hash}
```

## 3. Revocation Merkle Inclusion Proof (Future Endpoint)
```mermaid
flowchart LR
    A[Revocation Event] --> B[Add Leaf Hash]
    B --> C[Recompute Merkle Root]
    C --> D[Store Root / Anchor Option]
    Client --> E[GET /revocation/proof/:hash]
    E --> F[Return siblings + root]
    Client --> G[Locally recompute root]
    G --> H{Root Matches?}
    H -- Yes --> I[Proof Valid]
    H -- No --> J[Integrity Failure]
```

## 4. Semantic Anomaly Throttle Activation
```mermaid
flowchart LR
    A[Incoming PoA Validation Request] --> B[Semantic Counters Update]
    B --> C[Compute 60s Rates]
    C --> D[Calculate EWMA + Z-Score]
    D --> E{Z > Threshold?}
    E -- Yes --> F[Set throttle_active]
    F --> G[Future Requests => 429 semantic_throttle_active]
    E -- No --> H[Continue Normal Flow]
```

## 5. PoA Lifecycle & Delegation
```mermaid
flowchart LR
    A[Issue PoA Definition] --> B[Assign Parties & Requirements]
    B --> C[Sign PoA (future multi-weight)]
    C --> D[Use PoA in Transaction]
    D --> E[Validate Scope & Requirements]
    E --> F{Valid?}
    F -- Yes --> G[Authorize Action]
    F -- No --> H[Reject + semantic counter]
    G --> I[Potential Revocation]
    I --> J[Update Revocation Chain]
```

---
Refinement Checklist:
- Add concrete hash examples once stable
- Show threshold math after weight remediation
- Insert replay nonce field in attestation diagram on implementation
- Add partial revocation / suspension states to PoA lifecycle when extended
