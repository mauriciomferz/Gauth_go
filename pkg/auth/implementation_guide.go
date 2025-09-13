package auth
/*
 * Copyright (c) 2025 Gimel Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * This implementation builds on:
 * - OAuth 2.0 (RFC 6749) under Apache 2.0 license
 * - OpenID Connect under Apache 2.0 license
 * - Model Context Protocol (MCP) under MIT license
 */

package auth

// GAuth Implementation Guide
/*
This implementation provides a framework for AI governance and authorization following
the GAuth standard (RFC111). Key aspects of the implementation:

1. Core Components:
   - PowerOfAttorney: Comprehensive authorization model
   - LegalFramework: Jurisdiction and legal compliance
   - AITeamControls: Centralized AI team authorization
   - ComplianceTracker: Authorization monitoring

2. Key Interfaces:
   - PowerEnforcementPoint: Authorization enforcement
   - VerificationSystem: Identity and power verification
   - OpenIDIntegration: ACR level mapping
   - CommercialRegister: AI registration system

3. Implementation Requirements:
   a. Must maintain human accountability
   b. Must use centralized authorization
   c. Must not use excluded technologies:
      - No Web3/blockchain
      - No AI for authorization control
      - No DNA-based identity systems

4. Protocol Flow:
   a. Subscription Steps (I-VIII)
   b. Request Steps (a-i)
   See abstract flow in RFC111 Section 6

5. Usage Guidelines:
   a. Always verify human in authorization chain
   b. Maintain clear audit trails
   c. Enforce jurisdiction-specific rules
   d. Implement dual control principle
   e. Verify legal capacity of all parties

6. Extension Points:
   a. Custom jurisdiction rules
   b. Additional verification methods
   c. Enhanced compliance tracking
   d. Custom approval rules

7. Security Considerations:
   a. Protect against unauthorized delegation
   b. Maintain authorization centralization
   c. Implement proper revocation handling
   d. Ensure proper identity verification

8. Integration Notes:
   a. OAuth/OpenID Connect integration
   b. MCP compatibility
   c. Commercial register integration
   d. Identity provider integration

For detailed API documentation, see pkg/auth/doc.go
For implementation examples, see examples/
For security guidelines, see docs/SECURITY.md
*/