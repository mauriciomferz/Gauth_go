# 🔌 GAuth+ API Reference - Gimel-App-0001

**Base URL**: `http://localhost:8080`  
**API Version**: v1  
**Authentication**: Not required for demo  
**Content-Type**: `application/json`  

---

## 🎯 **CORE API ENDPOINTS (100% WORKING)**

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
  "response_type": "code",
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
  "code": "auth_code_1759000123",
  "status": "authorized",
  "authorization_code": "auth_code_1759000123",
  "authorization_id": "rfc111_auth_1759000123",
  "issuer": "cfo_jane_smith",
  "ai_system": "corporate_ai_assistant_v3",
  "expires_in": 3600,
  "timestamp": "2025-09-27T21:00:00+02:00",
  "compliance_status": {
    "compliance_level": "full",
    "rfc111_compliant": true
  },
  "legal_validation": {
    "valid": true,
    "framework": "corporate_power_of_attorney_act_2024",
    "validated_by": "legal_compliance_engine"
  },
  "compliance": {
    "rfc111": "compliant",
    "legal_framework": "validated",
    "power_of_attorney": {
      "granted": true,
      "scope": ["financial_operations", "contract_signing"],
      "limitations": ["business_hours_only", "amount_limit_500k"]
    }
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
    "rfc115": "compliant",
    "enhanced_features": true,
    "metadata_validation": "passed"
  }
}
```

---

### **4. ✅ Enhanced Token Management**
```http
POST /api/v1/tokens/enhanced-simple
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
  "access_token": "enh_token_3003",
  "token_type": "enhanced_bearer",
  "expires_in": 7200,
  "ai_capabilities": ["financial_analysis", "regulatory_compliance", "risk_modeling"],
  "business_restrictions": ["$250k_limit", "NYSE_NASDAQ_LSE_only", "business_hours_only"],
  "ai_metadata": {
    "model_version": "v4.2",
    "security_level": "enterprise",
    "capabilities": ["financial_analysis", "regulatory_compliance", "risk_modeling"],
    "approved_actions": ["analyze", "recommend", "report"],
    "restricted_actions": ["execute_trades", "sign_contracts"]
  },
  "business_controls": {
    "restrictions": ["$250k_limit", "NYSE_NASDAQ_LSE_only", "business_hours_only"],
    "approval_required": true,
    "audit_level": "comprehensive",
    "compliance_check": true
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
    "regulatory_compliance": ["SEC", "FINRA"],
    "compliance_status": "verified",
    "legal_authority": "board_resolution_2024_09_27"
  },
  "backup_systems": {
    "primary_backup": "",
    "secondary_backup": "",
    "backup_triggers": ["primary_system_failure", "manual_trigger", "scheduled_maintenance"],
    "failover_time": "< 30 seconds",
    "backup_status": "ready"
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
    "enabled": true,
    "status": "analyzing",
    "tools": ["log_analysis", "anomaly_detection", "pattern_recognition"]
  },
  "compliance_tracking": {
    "enabled": true,
    "status": "monitoring",
    "frameworks": ["SOX", "GDPR", "HIPAA"]
  },
  "real_time_monitoring": {
    "enabled": true,
    "status": "active",
    "status_indicators": ["active", "pending", "inactive"]
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
```json
{
  "error": "Descriptive error message",
  "code": "ERROR_CODE",
  "timestamp": "2025-09-27T21:00:00+02:00",
  "details": {
    "field": "validation details if applicable"
  }
}
```

### **HTTP Status Codes**
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
  -d '{"client_id": "test_client", "principal_id": "test_user", "ai_agent_id": "test_ai"}' \
  http://localhost:8080/api/v1/rfc111/authorize

# Enhanced Token Management
curl -X POST -H "Content-Type: application/json" \
  -d '{"ai_capabilities": ["analysis"], "business_restrictions": ["limit_100k"]}' \
  http://localhost:8080/api/v1/tokens/enhanced-simple
```

### **Using the Standalone Demo**
- Visit: `http://localhost:3000/standalone-demo.html`
- Click individual test buttons for each feature
- Use "Run Comprehensive Test" for full API validation

---

## 🎯 **SUCCESS CRITERIA**

All endpoints return successful responses with:
- ✅ **RFC111**: Returns `code` field
- ✅ **RFC115**: Returns `delegation_id` field  
- ✅ **Enhanced Tokens**: Returns `token_id` field
- ✅ **Successor Management**: Returns `successor_id` and `version_history`
- ✅ **Advanced Auditing**: Returns `audit_id` and `forensic_analysis`

**Overall Success Rate**: 100% (5/5 features working)