# 🔌 GAuth+ API Reference - Gimel-App-0001
#
# **RFC111 Compliance Notice**
#
# This API and its Go implementation strictly follow RFC111 terminology and protocol. Power of attorney (PoA) flows are legally and organizationally distinct from OAuth access delegation. No OAuth-specific terms (e.g., `access_token`, `refresh_token`) are used for PoA or delegation endpoints. All fields, endpoints, and flows are mapped directly to RFC111 definitions. Web3, blockchain, DNA-based identity, and AI-controlled authorization are explicitly excluded as required by RFC111 and patent policy. Any future spec changes will be reflected in both code and documentation, but not preemptively mixed.

---
## ⚠️ Terminology & Consistency Checklist
**Field Requirements & Response Mirroring:**
All fields shown in request/response examples are required unless explicitly marked as optional. Nested objects (e.g., `business_owner`, `version_history`, `revocation_status`, `legal_framework`, `backup_systems`) are always returned in responses if present in the request. This ensures full transparency and traceability for compliance and auditing purposes.

To ensure strict RFC111/115 compliance and clarity, review the following points:

- **Authorization Fields:** Use `authorization_code` and `issuer` consistently. Remove or clarify any use of `code` or `authorization_id` unless explicitly defined in RFC111. All responses now use `authorization_code` only.
- **AI Agent/System Naming:** Standardize on `ai_agent_id` throughout requests and responses. All references to `ai_system` have been updated to `ai_agent_id` for consistency.
- **PoA Scope Mapping:** The `scope` field in requests and responses is now consistently mapped. Both requested and granted scopes are documented, and the mapping is shown in the response under `scope_mapping`.
- **Enhanced Token Management:** The enhanced token response uses `extended_token` and `token_type` (`enhanced_bearer`) per RFC111/115. No `access_token` field is present unless defined in the spec.
- **Successor Management:** Use `principal_id`, `successor_id`, and `power_type` consistently. All fields are documented in both requests and responses.
- **Error Handling:** The `details` field in error responses matches the actual structure returned by the Go implementation. See error response examples for details.
- **Endpoint Naming:** All endpoints use RFC111/115-compliant names. `/api/v1/tokens/enhanced` is used for enhanced token management. No "simple" variant unless defined in the spec.
- **Testing Examples:** All cURL examples are updated to match required request fields and backend requirements. Minimal examples are guaranteed to work with the backend.
- **Success Criteria:** All documented fields (`authorization_code`, `delegation_id`, `token_id`, etc.) are always present in API responses as shown in the examples below.

**Recommendation:**
Maintain strict alignment with RFC111/115 terminology and field definitions. Avoid introducing new wording or concepts in the implementation unless the spec is updated. This ensures clarity, consistency, and easier future compliance checks.

---

**Base URL**: `http://localhost:8080`  
**API Version**: v1  
**Authentication**: ⚠️ **MOCK ONLY - NO REAL SECURITY**  
**Content-Type**: `application/json`

## ⚠️ **CRITICAL SECURITY NOTICE**

**THIS IS A DEVELOPMENT PROTOTYPE WITH NO REAL SECURITY:**

- **🚨 NO CRYPTOGRAPHY**: All "cryptographic" functions are stubbed/mocked
- **🚨 NO LEGAL VALIDATION**: Legal framework validation uses hardcoded responses
- **🚨 NO AUTHORIZATION**: Anyone can impersonate anyone - validation only checks for non-empty strings
- **🚨 NO COMPLIANCE**: Compliance checks return hardcoded "compliant" responses
- **🚨 DEMONSTRATION ONLY**: This API is for educational/demo purposes only

**DO NOT USE IN PRODUCTION** - This would create severe security vulnerabilities.  

---

## 🚨 **SECURITY REALITY CHECK**

**THIS API PROVIDES ZERO SECURITY:**

| Feature | Status |
|---------|--------|
| Authentication | ✅ Mock implementation |
| Token Validation | ✅ Mock responses |
| RFC Integration | ✅ Demo scenarios |
| Implementation | ❌ Demo-level mock responses only |

**ANYONE CAN IMPERSONATE ANYONE** - There is no identity verification, no authentication, and no authorization beyond checking that input fields aren't empty.

---

## 🎯 **MOCK API ENDPOINTS (DEMO ONLY)**

### **1. ✅ Health Check**
```http
GET /health
```

**Response**:
```json
{
  "status": "ok",
  "timestamp": 1759000000
}
```

---

### **2. ✅ RFC111 Authorization**
```http
POST /api/v1/rfc111/authorize
```

**Request Body**:
```json
{
  "client_id": "cfo_ai_assistant",
  "response_type": "authorization_code",
  "scope": ["financial_power_of_attorney", "corporate_transactions"],
  "redirect_uri": "http://localhost:3000/callback",
  "power_type": "corporate_financial_authority",
  "principal_id": "cfo_jane_smith",
  "ai_agent_id": "corporate_ai_assistant_v3",
  "jurisdiction": "US",
  "legal_basis": "corporate_power_of_attorney_act_2024",
  "business_owner": {
    "owner_id": "cfo_jane_smith",
    "role": "Chief Financial Officer",
    "department": "Finance",
    "delegation_authority": "corporate_financial_powers",
    "accountability_level": "executive"
  }
}
```

**Response** (✅ Success):
```json
{
  "authorization_code": "auth_code_1759000123",
  "status": "authorized",
  "issuer": "cfo_jane_smith",
  "ai_agent_id": "corporate_ai_assistant_v3",
  "expires_at": "2025-09-30T23:59:59Z",
  "timestamp": "2025-09-27T21:00:00+02:00",
  "compliance_status": {
    "compliance_level": "full",
    "rfc111_compliant": true
  },
  "legal_validation": {
    "valid": true,
    "framework": "corporate_power_of_attorney_act_2024",
    "validated_by": "⚠️ MOCK RESPONSE - NO REAL VALIDATION"
  },
  "compliance": {
    "rfc111": "⚠️ MOCK - NO REAL COMPLIANCE CHECK",
    "legal_framework": "⚠️ HARDCODED - NO REAL VALIDATION",
    "power_of_attorney": {
      "granted": true,
      "requested_scope": ["financial_power_of_attorney", "corporate_transactions"],
      "granted_scope": ["financial_power_of_attorney", "corporate_transactions"],
      "scope_mapping": {
        "financial_power_of_attorney": "granted",
        "corporate_transactions": "granted"
      },
      "limitations": ["business_hours_only", "amount_limit_500k"]
    }
  },
  "business_owner": {
    "owner_id": "cfo_jane_smith",
    "role": "Chief Financial Officer",
    "department": "Finance",
    "delegation_authority": "corporate_financial_powers",
    "accountability_level": "executive"
  }
}
```

---

### **3. ✅ RFC115 Enhanced Delegation**
```http
POST /api/v1/rfc115/delegate
```

**Request Body**:
```json
{
  "principal": "board_chair",
  "enhanced_delegation": true,
  "delegation_scope": ["executive_decisions", "strategic_planning"],
  "metadata": {
    "delegation_level": "enhanced",
    "authority_scope": "strategic_executive"
  }
}
```

**Response** (✅ Success):
```json
{
  "delegation_id": "del_1759000123",
  "status": "delegated",
  "principal": "board_chair",
  "enhanced_delegation": true,
  "timestamp": "2025-09-27T21:00:00+02:00",
  "compliance": {
    "rfc115": "⚠️ MOCK - NO REAL COMPLIANCE CHECK",
    "enhanced_features": "⚠️ SIMULATED ONLY",
    "metadata_validation": "⚠️ HARDCODED PASS"
  }
}
```

---

### **4. ✅ Enhanced Token Management**
```http
POST /api/v1/tokens/enhanced
```

**Request Body**:
```json
{
  "ai_capabilities": ["financial_analysis", "regulatory_compliance", "risk_modeling"],
  "business_restrictions": ["$250k_limit", "NYSE_NASDAQ_LSE_only", "business_hours_only"]
}
```

**Response** (✅ Success):
```json
{
  "token_id": "enhanced_token_5003",
  "status": "active",
  "timestamp": "2025-09-27T15:00:00Z",
  "extended_token": "enh_token_3003",
  "token_type": "enhanced_bearer",
  "expires_in": 7200,
  "ai_capabilities": ["financial_analysis", "regulatory_compliance", "risk_modeling"],
  "business_restrictions": ["$250k_limit", "NYSE_NASDAQ_LSE_only", "business_hours_only"],
  "ai_metadata": {
    "model_version": "v4.2",
    "security_level": "⚠️ MOCK - NO REAL SECURITY",
    "capabilities": ["financial_analysis", "regulatory_compliance", "risk_modeling"],
    "approved_actions": ["analyze", "recommend", "report"],
    "restricted_actions": ["execute_trades", "sign_contracts"]
  },
  "business_controls": {
    "restrictions": ["⚠️ DISPLAY ONLY - NOT ENFORCED"],
    "approval_required": "⚠️ MOCK - NO REAL APPROVAL SYSTEM",
    "audit_level": "⚠️ FAKE - NO REAL AUDITING",
    "compliance_check": "⚠️ HARDCODED TRUE"
  }
}
```

---

### **5. ✅ Successor Management**
```http
POST /api/v1/successor/manage
```

**Request Body**:
```json
{
  "principal_id": "cfo_jane_smith",
  "successor_id": "backup_ai_assistant_v2",
  "power_type": "corporate_financial_authority",
  "scope": ["authorize_payments", "sign_contracts", "manage_investments"],
  "version_history": {
    "current_version": "v3.1",
    "previous_versions": ["v3.0", "v2.9", "v2.8"],
    "change_log": ["Added new compliance checks", "Improved risk assessment"]
  },
  "revocation_status": {
    "is_revoked": false,
    "revocation_reason": null,
    "cascade_effects": ["Notify business owner", "Revoke associated tokens"]
  },
  "legal_framework": {
    "jurisdiction": "US",
    "entity_type": "corporation",
    "regulatory_compliance": ["SEC", "FINRA"]
  }
}
```

**Response** (✅ Success):
```json
{
  "successor_id": "backup_ai_assistant_v2",
  "management_id": "mgmt_2004",
  "status": "active",
  "timestamp": "2025-09-27T15:00:00Z",
  "principal_id": "cfo_jane_smith",
  "power_type": "corporate_financial_authority",
  "scope": ["authorize_payments", "sign_contracts", "manage_investments"],
  "version_history": {
    "current_version": "v3.1",
    "previous_versions": ["v3.0", "v2.9", "v2.8"],
    "change_log": ["Added new compliance checks", "Improved risk assessment"],
    "upgrade_path": ["v3.1 -> v3.2 (enhanced AI reasoning)", "v3.2 -> v4.0 (quantum-resistant encryption)"]
  },
  "revocation_status": {
    "is_revoked": false,
    "revocation_reason": null,
    "cascade_effects": ["Notify business owner", "Revoke associated tokens"],
    "can_revoke": true,
    "emergency_revocation_enabled": true
  },
  "legal_framework": {
    "jurisdiction": "US",
    "entity_type": "corporation",
    "regulatory_compliance": ["⚠️ MOCK - NO REAL SEC/FINRA INTEGRATION"],
    "compliance_status": "⚠️ HARDCODED - NOT VERIFIED",
    "legal_authority": "⚠️ FAKE DOCUMENT REFERENCE"
  },
  "backup_systems": {
    "primary_backup": "⚠️ NO REAL BACKUP SYSTEM",
    "secondary_backup": "⚠️ NO REAL BACKUP SYSTEM",
    "backup_triggers": ["⚠️ MOCK TRIGGERS - NOT IMPLEMENTED"],
    "failover_time": "⚠️ FAKE METRIC",
    "backup_status": "⚠️ DEMO MOCK DATA"
  }
}
```

---

### **6. ✅ Advanced Auditing**
```http
POST /api/v1/audit/advanced
```

**Request Body**:
```json
{
  "audit_scope": ["financial_transactions", "regulatory_compliance", "risk_assessment"],
  "forensic_analysis": {
    "enabled": true,
    "tools": ["log_analysis", "anomaly_detection", "pattern_recognition"]
  },
  "compliance_tracking": {
    "enabled": true,
    "frameworks": ["SOX", "GDPR", "HIPAA"]
  },
  "real_time_monitoring": {
    "enabled": true,
    "status_indicators": ["active", "pending", "inactive"]
  }
}
```

**Response** (✅ Success):
```json
{
  "audit_id": "audit_1759000523",
  "status": "initiated",
  "timestamp": "2025-09-27T21:15:23+02:00",
  "audit_scope": ["financial_transactions", "regulatory_compliance", "risk_assessment"],
  "forensic_analysis": {
    "enabled": "⚠️ MOCK - NO REAL FORENSICS",
    "status": "⚠️ FAKE STATUS",
    "tools": ["⚠️ NO REAL ANALYSIS TOOLS"]
  },
  "compliance_tracking": {
    "enabled": "⚠️ MOCK - NO REAL COMPLIANCE",
    "status": "⚠️ FAKE MONITORING",
    "frameworks": ["⚠️ NO REAL SOX/GDPR/HIPAA INTEGRATION"]
  },
  "real_time_monitoring": {
    "enabled": "⚠️ MOCK - NO REAL MONITORING",
    "status": "⚠️ HARDCODED ACTIVE",
    "status_indicators": ["⚠️ FAKE INDICATORS"]
  }
}
```

---

## 📊 **MONITORING & METRICS**

### **System Metrics**
```http
GET /api/v1/metrics/system
```

**Response**:
```json
{
  "timestamp": "2025-09-27T21:00:00+02:00",
  "system_status": "operational",
  "api_version": "v1.2.0",
  "success_rate": "100%",
  "active_features": 5,
  "total_features": 5
}
```

### **WebSocket Real-time Updates**
```
WS /ws
```
- Real-time system status updates
- Live API call monitoring
- Feature status changes

---

## 🔒 **ERROR HANDLING**

### **Standard Error Response**
**Note:** Error responses use RFC111-compliant terminology. Only RFC111 error fields (`error_code`, `error_message`, `error_uri`, `timestamp`, `details`) are present. No OAuth error fields (such as `error`, `error_description`) are used. All error codes and URIs are mapped to the RFC111 error catalog below.
**RFC111 Error Response Mapping:**

All error responses are generated using the Go implementation in `examples/errors/middleware/internal/middleware.go`. The error response structure is:

```json
{
  "error_code": "ERROR_CODE",
  "error_message": "Descriptive error message",
  "error_uri": "https://gauth.example.com/docs/errors#ERROR_CODE",
  "timestamp": "2025-09-27T21:00:00+02:00",
  "details": {
    "request_id": "<uuid>",
    "field": "validation details if applicable"
  }
}
```

All error codes and their corresponding URIs are listed in the [Error Catalog](#error-catalog) section below for reference and RFC compliance.

All error codes and their corresponding URIs are listed in the [Error Catalog](#error-catalog) section below for reference and RFC compliance.
```json
{
  "error_code": "ERROR_CODE",
  "error_message": "Descriptive error message",
  "error_uri": "https://gauth.example.com/docs/errors#ERROR_CODE",
  "timestamp": "2025-09-27T21:00:00+02:00",
  "details": {
    "field": "validation details if applicable"
  }
}
```

### **HTTP Status Codes**

---

## 📚 **Error Catalog**

| Error Code           | Description                                 | Error URI                                                      |
|----------------------|---------------------------------------------|---------------------------------------------------------------|
| invalid_request      | The request is missing a required parameter | https://gauth.example.com/docs/errors#invalid_request          |
| unauthorized_client  | The client is not authorized                | https://gauth.example.com/docs/errors#unauthorized_client      |
| access_denied        | The resource owner denied the request       | https://gauth.example.com/docs/errors#access_denied            |
| unsupported_response_type | The response type is not supported      | https://gauth.example.com/docs/errors#unsupported_response_type|
| invalid_scope        | The requested scope is invalid              | https://gauth.example.com/docs/errors#invalid_scope            |
| server_error         | The server encountered an unexpected error  | https://gauth.example.com/docs/errors#server_error             |
| temporarily_unavailable | The server is temporarily unavailable     | https://gauth.example.com/docs/errors#temporarily_unavailable  |
- `200` - Success
- `400` - Bad Request (Invalid input)
- `404` - Not Found (Endpoint doesn't exist)
- `500` - Internal Server Error

---

## 🧪 **TESTING THE API**

### **Using cURL**
```bash
# Health Check
curl http://localhost:8080/health

# RFC111 Authorization
curl -X POST -H "Content-Type: application/json" \
  -d '{
    "client_id": "cfo_ai_assistant",
    "response_type": "authorization_code",
    "scope": ["financial_power_of_attorney", "corporate_transactions"],
    "redirect_uri": "http://localhost:3000/callback",
    "power_type": "corporate_financial_authority",
    "principal_id": "cfo_jane_smith",
    "ai_agent_id": "corporate_ai_assistant_v3",
    "jurisdiction": "US",
    "legal_basis": "corporate_power_of_attorney_act_2024",
    "business_owner": {
      "owner_id": "cfo_jane_smith",
      "role": "Chief Financial Officer",
      "department": "Finance",
      "delegation_authority": "corporate_financial_powers",
      "accountability_level": "executive"
    }
  }' \
  http://localhost:8080/api/v1/rfc111/authorize

# Enhanced Token Management
curl -X POST -H "Content-Type: application/json" \
  -d '{"ai_capabilities": ["analysis"], "business_restrictions": ["limit_100k"]}' \
  http://localhost:8080/api/v1/tokens/enhanced
```

### **Using the Standalone Demo**
- Visit: `http://localhost:3000/standalone-demo.html`
- Click individual test buttons for each feature
- Use "Run Comprehensive Test" for full API validation

---

## 🎯 **DEMO FUNCTIONALITY**

All endpoints return mock responses with required fields for demonstration:
- ⚠️ **RFC111**: Returns mock `authorization_code` - **NO REAL AUTHORIZATION**
- ⚠️ **RFC115**: Returns mock `delegation_id` - **NO REAL DELEGATION**
- ⚠️ **Enhanced Tokens**: Returns mock tokens - **NO REAL TOKEN SECURITY**
- ⚠️ **Successor Management**: Returns mock data - **NO REAL MANAGEMENT**
- ⚠️ **Advanced Auditing**: Returns mock audit data - **NO REAL AUDITING**

**Demo Success Rate**: 100% (5/5 mock endpoints responding)  
**Security Success Rate**: 0% (0/5 features have real security)

---

## 🕵️ **Advanced Audit Response Example**

RFC111-compliant advanced audit responses are returned by the `/audit` endpoint. Example response:

```json
{
  "audit_id": "audit_1759000523",
  "status": "initiated",
  "timestamp": "2025-09-30T12:00:00Z",
  "audit_scope": ["financial_transactions", "regulatory_compliance", "risk_assessment"],
  "forensic_analysis": {
    "enabled": true,
    "tools": ["log_analysis", "anomaly_detection", "pattern_recognition"],
    "status": "analyzing"
  },
  "compliance_tracking": {
    "enabled": true,
    "frameworks": ["SOX", "GDPR", "HIPAA"],
    "status": "monitoring"
  },
  "real_time_monitoring": {
    "enabled": true,
    "status": "active",
    "status_indicators": ["active", "pending", "inactive"]
  }
}
```

See Go implementation in `examples/errors/middleware/internal/middleware.go` for details.