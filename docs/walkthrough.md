#### 11. Cleanup Legacy RFC References
- **Objective**: Standardise terminology by replacing legacy "RFC 111" and "RFC 115" references with "AAP-001" and "AAP-002" respectively.
- **Actions**:
    - Renamed all \`rfc0111_*\` and \`rfc0115_*\` files in \`pkg/agentauth_aap_001\`, \`pkg/poa\`, and \`scripts\`.
    - Renamed \`docs/rfc_endpoint_mapping.md\` to \`docs/aap_endpoint_mapping.md\`.
    - Performed bulk text replacement of "RFC 111" -> "AAP-001", "RFC 115" -> "AAP-002", etc.
    - Updated references in documentation (`final_compliance_quality_report.md`, etc.) to point to new AAP filenames.
    - **Round 2**: Removed lingering "AAP-RFC-0111/0115" references from `frontend/templates/demo.html` and `sdks/typescript/agentauth-client.ts`.
- **Verification**: Ran \`go test ./pkg/...\` and \`go build ./...\` which passed. Checked for "0111"/"0115" strings.
