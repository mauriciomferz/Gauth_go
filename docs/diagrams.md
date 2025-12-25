## Protocol & Diagnostics Diagrams

### Revocation Inclusion Flow
```mermaid
sequenceDiagram
  participant Client
  participant Server
  participant RevocationChain as Revocation Chain
  participant Notary as Anchor/Signer
  Client->>Server: POST /delegation/revoke (delegation_id)
  Server->>RevocationChain: Append RevocationEvent
  RevocationChain-->>Server: Updated chain (events, merkle_root)
  Server->>RevocationChain: SignTreeHead()
  RevocationChain-->>Server: SignedTreeHead (root, signatures)
  Server-->>Client: 200 {revocation_event, signed_tree_head}
  Client->>Server: GET /token/revocation/proof?id=rev-42
  Server->>RevocationChain: Build Merkle Proof
  RevocationChain-->>Server: {leaf_hash, siblings[], index}
  Server-->>Client: 200 {proof, signed_tree_head}
  Client->>Client: Verify inclusion (recompute root + signature)
```

### Capability & Rotation Anchoring
```mermaid
flowchart LR
  subgraph Capability Registry
    A[Capabilities JSON] --> H[Hash SHA256]
  end
  subgraph Rotation Ledger
    R1[Rotation Descriptor 1] --> R2[Rotation Descriptor 2] --> Rn[Rotation Descriptor n]
    Rn --> RH[Head Hash]
  end
  H --> C[Combine cap_hash:rot_head]
  RH --> C
  C --> D[Digest SHA256]
  D --> E[External Anchor Emission]
  E --> F[(Receipt Store)]
```

### Semantic Diagnostics Feedback Loop
```mermaid
stateDiagram-v2
  [*] --> Snapshot: Collect Counters
  Snapshot --> HistoryUpdate: Append History (<=1/sec throttle)
  HistoryUpdate --> RateCalc: Compute per-minute window rates
  RateCalc --> AnomalyScores: Update EWMA mean/variance & z-scores
  AnomalyScores --> ThrottleCheck: Compare scores vs threshold
  ThrottleCheck --> Snapshot: Loop
  ThrottleCheck --> ThrottledAction: If threshold exceeded
  ThrottledAction --> Snapshot
```

### Attestation Verification Path
```mermaid
sequenceDiagram
  participant Client
  participant Server
  participant KeyRegistry
  Client->>Server: POST /model/limits/attestation/verify {payload+signature}
  Server->>KeyRegistry: Lookup kid
  KeyRegistry-->>Server: {public_key}
  Server->>Server: Canonicalize unsigned payload
  Server->>Server: Ed25519 Verify(signature, canonical_bytes)
  alt Success
    Server-->>Client: 200 {success:true, valid:true, kid}
  else Failure
    Server-->>Client: 401 {success:false, code: attestation_signature_invalid, rfc_ref: rfc111:attestation_integrity}
  end
```

### Rotation Summary Invariants
```mermaid
sequenceDiagram
  participant Client
  participant Server
  participant Ledger
  Client->>Server: GET /beta/rotations/summary
  Server->>Ledger: Load Entries
  Server->>Server: Continuity Scan (PrevHash == prior Hash)
  Server->>Server: Build Aggregate Hash
  alt Signing Enabled
    Server->>KeyRegistry: Active Ed25519 Key
    KeyRegistry-->>Server: {priv, kid}
    Server->>Server: Sign canonical summary
    Server->>Server: Verify signature (sanity)
  end
  alt Continuity Gap
    Server-->>Client: 400 rotation_continuity_gap (rfc111:rotations)
  else Signature Invalid
    Server-->>Client: 400 rotation_summary_signature_invalid (rfc111:rotations)
  else Success
    Server-->>Client: 200 {summary, anchored:false}
  end
```

### Legend
- rfc111:* indicates governance & integrity related clauses.
- rfc115:* indicates semantic diagnostics & reactive controls.
- EdDSA signatures domain separated (GAUTH_ROTATION_SUMMARY:, GAUTH_ROTATION_DESCRIPTOR:).

### Future Extensions
- Inclusion proof: add optional batch verification endpoint.
- Multi-signature rotation summaries (threshold >1).
- External transparency log publishing for revocation & rotation roots.
# GAuth Mermaid Diagrams

## Token Issuance Flow
```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant Auth as Auth Service
    participant Ledger as RevocationLedger
    participant Signer as EdDSA Signer
    Client->>Auth: POST /api/v1/token/create {subject, ttl}
    Auth->>Auth: Validate payload (subject, ttl bounds, clock skew)
    Auth->>Ledger: Record issuance event (token_id, subject)
    Auth->>Signer: Sign JWT (header+claims)
    Signer-->>Auth: Signature (EdDSA)
    Auth-->>Client: 200 {token, token_id, issued_at, expires_at}
    Note over Auth,Ledger: Nonce/JTI stored for replay protection
```

## Multi-Sig Aggregated BLS Signature (Issue Mode)
```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant BLS as BLS Engine
    participant Keys as Key Holders
    Client->>BLS: POST /api/v1/crypto/bls/aggregate {mode=issue, message, participants}
    loop Generate Keys
        BLS->>Keys: Derive per-participant keypair
        Keys-->>BLS: Public & Private key
    end
    BLS->>BLS: Hash message & compute individual signatures
    BLS->>BLS: Aggregate signatures into single signature
    BLS-->>Client: 200 {aggregated_signature_b64, public_keys_b64[], key_ids[]}
    Note over BLS: Aggregation validated before response
```

## BLS Proof-of-Possession Challenge & Verify
```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant BLS as BLS Engine
    Client->>BLS: POST /api/v1/crypto/bls/aggregate {mode=issue, require_pop=true}
    BLS->>BLS: Generate challenge per public key (random scalar)
    BLS-->>Client: 200 {challenges_b64[], public_keys_b64[]}
    Client->>Client: Collect signatures from holders over challenges
    Client->>BLS: POST /api/v1/crypto/bls/pop/verify {pairs[]}
    BLS->>BLS: Verify each PoP signature
    BLS-->>Client: 200 {valid, failures, failure_indices[]}
    Note over BLS: Fails fast on malformed input
```

## Revocation Anchoring (Merkle Root Emit)
```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant RevChain as RevocationChain
    participant Anchor as External AnchorClient
    Client->>RevChain: (implicit) Previously issued/revoked events update Merkle root
    Client->>Anchor: POST /api/v1/anchor/revocation/emit
    Anchor->>RevChain: Read current MerkleRoot()
    alt Chain Empty
        Anchor-->>Client: 404 revocation_chain_empty
    else Valid Root
        Anchor->>Anchor: Idempotent anchor(root)
        Anchor-->>Client: 200 {hash, merkle_root, chain_length}
    end
    Note over Anchor,RevChain: Receipt stored; total anchors incremented
```

## Semantic Diagnostics Snapshot
```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant Server
    Client->>Server: GET /api/v1/diagnostics/semantic
    Server->>Server: Collect current counters (optional wiring)
    Server->>Server: Append history entry (throttled)
    Server->>Server: Compute 60s & 300s per-minute rates
    Server->>Server: Update EWMA & anomaly scores
    Server-->>Client: 200 {counters, history[], anomaly{rates,scores}, integrity_status}
    Note over Server: History capped at semanticHistoryCap
```

## Delegation (POA) Authorization Overview
```mermaid
flowchart TD
    A[Incoming POA Request] --> B[Validate JSON & required fields]
    B --> C{Scope Narrowing?}
    C -->|Widening| E[Reject: delegation_scope_widening]
    C -->|Valid/Narrow| D[Verify Chain Signatures]
    D --> F{Revoked?}
    F -->|Yes| G[Reject: delegation_revoked]
    F -->|No| H[Check Capability Lifecycle]
    H --> I{Deprecated/Sunset?}
    I -->|Violation| J[Emit lifecycle metrics]
    I -->|OK| K[Authorize & emit metrics]
    J --> K
    K --> L[Respond 200 Authorization Decision]
```

---
Generated diagrams reflect current endpoint & error taxonomy conventions (codes like revocation_chain_empty, delegation_scope_widening). Update as flows evolve.

## GNAP Resource Server Connection (RFC 9767)
```mermaid
sequenceDiagram
    autonumber
    participant RS as Resource Server
    participant AS as GAuth AS
    participant Client
    
    Note over RS, AS: Dynamic Registration
    RS->>AS: POST /gnap/rs/register {name, resource_uris, key}
    AS-->>RS: 201 Created {instance_id}
    
    Note over RS, Client: Token Usage
    Client->>RS: Request Resource (Authorization: GNAP <token>)
    
    Note over RS, AS: Introspection
    RS->>AS: POST /gnap/rs/introspect {token}
    AS->>AS: Validate Token & PoA Links
    AS-->>RS: 200 OK {active: true, access: [...], poa: {...}}
    RS-->>Client: Resource Data
```

## OAuth 2.0 CIBA Flow
```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant AS as GAuth Authentication Endpoint
    participant User as User (Authentication Device)
    
    Client->>AS: POST /bc-authorize {login_hint, binding_message}
    AS-->>Client: 200 OK {auth_req_id, interval}
    
    par Backchannel Authentication
        AS->>User: Push Notification / Authentication Request
        User-->>AS: Approve/Deny
    and Client Polling
        loop Until Completed
            Client->>AS: POST /token {grant_type=ciba, auth_req_id}
            AS-->>Client: 400 {error: authorization_pending}
        end
    end
    
    Client->>AS: POST /token {grant_type=ciba, auth_req_id}
    AS-->>Client: 200 OK {access_token, id_token}
```

## SAML 2.0 & SCIM 2.0 Architecture
```mermaid
flowchart TD
    subgraph Federated Identity
        IDP[External IdP]
    end
    
    subgraph GAuth System
        SP[SAML SP Handler]
        SCIM[SCIM Handler]
        UserStore[(User Store)]
    end
    
    subgraph Admin Client
        AdminUI[AuthAI Admin Portal]
    end

    IDP -->|SAML Response| SP
    SP -->|Create/Update Session| UserStore
    
    AdminUI -->|Manage Clients| SCIM
    SCIM -->|Provision Users| UserStore
```
