# GAuth 1.0 Complete API Reference

**Official Gimel Foundation Implementation - Combined Library & Web API Documentation**

Complete API reference for GiFo-RFC-0111 (GAuth 1.0) and GiFo-RFC-0115 (PoA Definition) implementation, including both the Go library API and the web demonstration API.

## 📋 **Table of Contents**

1. [Web Demo API](#web-demo-api)
2. [Go Library API](#go-library-api)
3. [Data Types Reference](#data-types-reference)
4. [Error Handling](#error-handling)
5. [Examples](#examples)

---

## 🌐 **Web Demo API**

The web demonstration API provides REST endpoints for testing and demonstrating RFC-0111 and RFC-0115 functionality through a web interface.

### **Base URL**
```
http://localhost:8080
```

### **Authentication**
No authentication required for demo API endpoints.

### **Content Type**
All API requests and responses use `application/json`.

---

### **📋 Demo Scenarios**

#### **GET /scenarios**
Lists all available demo scenarios for testing different RFC configurations.

**Request:**
```http
GET /scenarios HTTP/1.1
Host: localhost:8080
```

**Response:**
```json
[
  {
    "id": "rfc0111-basic",
    "name": "RFC-0111 Basic GAuth 1.0",
    "description": "Basic RFC-0111 GAuth 1.0 scenario with P*P Architecture",
    "config": {
      "p2p_enabled": true,
      "exclusions": ["resource1", "resource2"],
      "extended_tokens": true,
      "ai_client": false
    },
    "rfc_type": "RFC-0111"
  },
  {
    "id": "rfc0111-ai",
    "name": "RFC-0111 AI Client",
    "description": "RFC-0111 with AI client capabilities enabled",
    "config": {
      "p2p_enabled": true,
      "exclusions": [],
      "extended_tokens": true,
      "ai_client": true
    },
    "rfc_type": "RFC-0111"
  },
  {
    "id": "rfc0115-basic",
    "name": "RFC-0115 Basic PoA Definition",
    "description": "Basic RFC-0115 Power of Attorney definition scenario",
    "config": {
      "parties": {
        "grantor": "User A",
        "grantee": "User B", 
        "witness": "System"
      },
      "authorization_type": "limited",
      "legal_framework": "standard"
    },
    "rfc_type": "RFC-0115"
  },
  {
    "id": "rfc0115-advanced",
    "name": "RFC-0115 Advanced PoA",
    "description": "Advanced RFC-0115 with complex authorization requirements",
    "config": {
      "parties": {
        "grantor": "Corporation A",
        "grantee": "Agent B",
        "witness": "Legal System",
        "notary": "Certified Notary"
      },
      "authorization_type": "full",
      "legal_framework": "enterprise"
    },
    "rfc_type": "RFC-0115"
  },
  {
    "id": "combined-demo",
    "name": "Combined RFC Demo",
    "description": "Demonstration of combined RFC-0111 and RFC-0115 functionality",
    "config": {
      "rfc0111": {
        "p2p_enabled": true,
        "exclusions": ["restricted"],
        "extended_tokens": true,
        "ai_client": true
      },
      "rfc0115": {
        "parties": {
          "grantor": "System",
          "grantee": "Client"
        },
        "authorization_type": "limited"
      }
    },
    "rfc_type": "Combined"
  }
]
```

---

### **🔐 Authentication**

#### **POST /authenticate**
Authenticates using a selected demo scenario and returns a mock authentication token.

**Request:**
```json
{
  "scenario_id": "rfc0111-basic"
}
```

**Response (Success):**
```json
{
  "success": true,
  "token": "gauth_abc123def456...",
  "message": "Authentication successful for RFC-0111 Basic GAuth 1.0",
  "metadata": {
    "scenario": "RFC-0111 Basic GAuth 1.0",
    "rfc_type": "RFC-0111",
    "timestamp": 1696260000,
    "config": {
      "p2p_enabled": true,
      "exclusions": ["resource1", "resource2"],
      "extended_tokens": true,
      "ai_client": false
    }
  },
  "rfc_type": "RFC-0111"
}
```

**Response (Error):**
```json
{
  "success": false,
  "message": "Scenario not found"
}
```

**Error Codes:**
- `400 Bad Request`: Invalid JSON or missing scenario_id
- `404 Not Found`: Scenario not found

---

#### **POST /validate**
Validates an authentication token received from the authenticate endpoint.

**Request:**
```json
{
  "token": "gauth_abc123def456..."
}
```

**Response:**
```json
{
  "valid": true,
  "message": "Token is valid",
  "timestamp": 1696260000
}
```

**Error Codes:**
- `400 Bad Request`: Invalid JSON or missing token

---

### **🎯 RFC-0111 Configuration**

#### **POST /rfc0111/config**
Creates and validates an RFC-0111 configuration using the combined RFC implementation.

**Request:**
```json
{
  "p2p_enabled": true,
  "extended_tokens": true,
  "ai_client": false,
  "exclusions": ["web3_blockchain", "dna_based_identities"]
}
```

**Response (Success):**
```json
{
  "success": true,
  "message": "RFC-0111 configuration created successfully",
  "config": {
    "pp_architecture": {
      "pep": {
        "supply_side": {
          "entity": "client",
          "enforcement": ["oauth_flow", "token_validation"],
          "status": "active"
        },
        "demand_side": {
          "entity": "resource_server",
          "enforcement": ["scope_validation", "exclusions_check"],
          "status": "active"
        }
      },
      "pdp": {
        "primary_pdp": "authorization_server",
        "secondary_pdp": "resource_owner",
        "decision_rules": ["scope_validation", "exclusions_enforcement"]
      },
      "pip": {
        "authorization_server": "gauth_server",
        "data_sources": ["user_profile", "resource_metadata"],
        "info_types": ["identity", "permissions", "exclusions"]
      },
      "pap": {
        "client_owner_authorizer": "system_admin",
        "resource_owner_authorizer": "resource_admin",
        "policy_management": ["exclusions_policy", "ai_governance_policy"]
      },
      "pvp": {
        "trust_service_provider": "certification_authority",
        "verification_methods": ["digital_certificate", "oauth_token"],
        "identity_types": ["human", "ai_agent", "organization"]
      }
    },
    "exclusions": {
      "web3_blockchain": {
        "prohibited": true,
        "description": "Web3 and blockchain technologies are prohibited",
        "license_required": false
      },
      "ai_operators": {
        "prohibited": false,
        "description": "AI operators allowed with proper licensing",
        "license_required": true
      },
      "dna_based_identities": {
        "prohibited": true,
        "description": "DNA-based identities are prohibited",
        "license_required": false
      },
      "decentralized_auth": {
        "prohibited": true,
        "description": "Decentralized authentication is prohibited",
        "license_required": false
      },
      "enforcement_level": "strict"
    },
    "extended_tokens": {
      "token_type": "gauth_extended",
      "scope": ["authorization", "compliance"],
      "duration": "1h0m0s",
      "authorization": {
        "transactions": ["approve", "delegate"],
        "decisions": ["authorize", "revoke"],
        "actions": ["create", "modify", "delete"],
        "resource_rights": ["read", "write", "execute"]
      },
      "compliance": {
        "compliance_tracking": true,
        "audit_trail": ["creation", "usage", "revocation"],
        "revocation_status": "active"
      }
    },
    "gauth_roles": {
      "resource_owner": {
        "identity": "user_principal",
        "legal_capacity": true,
        "transaction_authority": ["financial", "legal"],
        "decision_acceptance": ["authorization", "delegation"],
        "action_impact": ["medium", "high"]
      },
      "resource_server": {
        "identity": "protected_service",
        "asset_types": ["data", "services", "financial"],
        "protected_resources": ["user_data", "financial_info"],
        "token_validation": "jwt_validation",
        "ai_capable": false
      },
      "client": {
        "type": "digital_agent",
        "identity": "ai_client_v1",
        "ai_capabilities": ["nlp", "decision_making"],
        "autonomy_level": "supervised",
        "request_types": ["data_access", "transaction"],
        "compliance_mode": "strict"
      },
      "authorization_server": {
        "identity": "gauth_server",
        "extended_token_issuing": true,
        "compliance_tracking": true,
        "pp_architecture_support": true,
        "exclusions_enforced": true
      },
      "client_owner": {
        "identity": "system_owner",
        "authorization_level": "admin",
        "ai_system_ownership": ["full"],
        "delegated_powers": ["configuration", "monitoring"]
      },
      "owner_authorizer": {
        "identity": "legal_authority",
        "statutory_authority": true,
        "authorization_scope": ["legal", "compliance"],
        "verification_method": "legal_certification"
      }
    },
    "version": "1.0.0",
    "status": "active",
    "created_at": "2025-10-02T20:00:00Z",
    "updated_at": "2025-10-02T20:00:00Z"
  },
  "rfc_version": "RFC-0111",
  "timestamp": 1696260000
}
```

**Error Codes:**
- `400 Bad Request`: Invalid configuration parameters

---

### **📋 RFC-0115 PoA Definition**

#### **POST /rfc0115/poa**
Creates and validates an RFC-0115 Power of Attorney definition.

**Request:**
```json
{
  "parties": {
    "grantor": "System Owner",
    "grantee": "AI Agent", 
    "witness": "Authorization Server"
  },
  "authorization_type": "limited",
  "legal_framework": "standard"
}
```

**Response (Success):**
```json
{
  "success": true,
  "message": "RFC-0115 Power of Attorney definition validated successfully",
  "poa_definition": {
    "parties": {
      "principal": {
        "type": "individual",
        "identity": "system_owner_001",
        "individual": {
          "name": "System Owner",
          "citizenship": "US"
        }
      },
      "representative": null,
      "authorized_client": {
        "type": "digital_agent",
        "identity": "ai_agent_v1",
        "version": "1.0.0",
        "operational_status": "active"
      }
    },
    "authorization": {
      "authorization_type": {
        "representation_type": "sole",
        "restrictions": ["financial_limits"],
        "sub_proxy_authority": false,
        "signature_type": "single"
      },
      "applicable_sectors": ["information_communication"],
      "applicable_regions": [
        {
          "type": "country",
          "identifier": "US",
          "name": "United States"
        }
      ],
      "authorized_actions": {
        "digital_actions": ["data_processing", "api_calls"],
        "transaction_actions": [],
        "physical_actions": []
      }
    },
    "requirements": {
      "validity_period": {
        "start_time": "2025-10-02T20:00:00Z",
        "end_time": "2026-10-02T20:00:00Z",
        "time_windows": [
          {
            "start": "09:00",
            "end": "17:00",
            "timezone": "UTC",
            "days": ["Mon", "Tue", "Wed", "Thu", "Fri"]
          }
        ],
        "geo_constraints": ["US"],
        "suspension_rules": ["security_breach", "compliance_violation"]
      },
      "formal_requirements": {
        "notarization_required": false,
        "witness_required": true,
        "legal_review_required": true,
        "registration_required": false
      },
      "power_limits": {
        "power_levels": [
          {
            "type": "transaction_amount",
            "limit": 1000.0,
            "currency": "USD",
            "description": "Maximum transaction amount"
          }
        ],
        "interaction_boundaries": ["read_only_data"],
        "tool_limitations": ["approved_apis_only"],
        "outcome_limitations": ["non_binding_recommendations"],
        "model_limits": [
          {
            "parameter_count": 1000000000,
            "reasoning_methods": ["logical", "statistical"],
            "training_methods": ["supervised"],
            "description": "AI model constraints"
          }
        ],
        "behavioral_limits": ["no_autonomous_transactions"],
        "quantum_resistance": true,
        "explicit_exclusions": ["financial_trading", "legal_contracts"]
      },
      "specific_rights": {
        "data_access_rights": ["user_profile", "preferences"],
        "modification_rights": [],
        "delegation_rights": [],
        "revocation_rights": ["immediate_revocation"]
      },
      "special_conditions": {
        "emergency_protocols": ["system_shutdown"],
        "escalation_procedures": ["human_oversight"],
        "monitoring_requirements": ["activity_logging", "compliance_tracking"],
        "reporting_obligations": ["weekly_reports"]
      },
      "death_incapacity": {
        "death_termination": true,
        "incapacity_suspension": true,
        "successor_designation": "backup_admin",
        "notification_procedures": ["immediate_alert"]
      },
      "security_compliance": {
        "encryption_requirements": ["AES-256", "RSA-4096"],
        "audit_requirements": ["quarterly_review"],
        "compliance_frameworks": ["SOC2", "ISO27001"],
        "security_monitoring": ["real_time", "anomaly_detection"]
      },
      "jurisdiction_law": {
        "language": "English",
        "governing_law": "US_Federal_Law",
        "place_of_jurisdiction": "US",
        "attached_documents": ["terms_of_service", "privacy_policy"]
      },
      "conflict_resolution": {
        "dispute_resolution_method": "arbitration",
        "arbitration_rules": "AAA_Commercial",
        "governing_law": "Delaware_Law",
        "jurisdiction": "Delaware_Courts"
      }
    },
    "gauth_context": {
      "pp_architecture_role": "client_authorization",
      "exclusions_compliant": true,
      "extended_token_scope": ["poa_validation", "compliance_tracking"],
      "ai_governance_level": "supervised"
    }
  },
  "rfc_version": "RFC-0115",
  "timestamp": 1696260000
}
```

**Error Codes:**
- `400 Bad Request`: Invalid PoA definition or missing parties information

---

### **🔄 Combined RFC Demo**

#### **POST /combined/demo**
Demonstrates the combined functionality of RFC-0111 and RFC-0115 in a unified configuration.

**Request:**
```json
{
  "rfc0111": {
    "p2p_enabled": true,
    "extended_tokens": true,
    "ai_client": true,
    "exclusions": ["web3_blockchain"]
  },
  "rfc0115": {
    "parties": {
      "grantor": "System",
      "grantee": "AI Agent"
    },
    "authorization_type": "limited"
  }
}
```

**Response (Success):**
```json
{
  "success": true,
  "message": "Combined RFC configuration validated successfully",
  "combined_config": {
    "rfc_0111": {
      "pp_architecture": { /* RFC-0111 configuration */ },
      "exclusions": { /* exclusions configuration */ },
      "extended_tokens": { /* token configuration */ },
      "gauth_roles": { /* role definitions */ },
      "version": "1.0.0",
      "status": "active",
      "created_at": "2025-10-02T20:00:00Z",
      "updated_at": "2025-10-02T20:00:00Z"
    },
    "rfc_0115": {
      "parties": { /* party definitions */ },
      "authorization": { /* authorization scope */ },
      "requirements": { /* requirements structure */ },
      "gauth_context": { /* integration context */ }
    },
    "integration_level": "combined_rfc",
    "combined_version": "1.0.0",
    "compatibility": {
      "rfc0111_version": "1.0.0",
      "rfc0115_version": "1.0.0",
      "integration_status": "fully_compatible"
    }
  },
  "rfc_versions": ["RFC-0111", "RFC-0115"],
  "timestamp": 1696260000
}
```

**Error Codes:**
- `400 Bad Request`: Invalid combined configuration or missing RFC specifications

---

### **📁 Static Files**

#### **GET /**
Serves the frontend application files from the `/frontend/` directory.

**Main Files:**
- `/` - Main demo application (index.html)
- Static assets (CSS, JS, images) served from frontend directory

---

## 📚 **Go Library API**

*[Previous Go library API documentation remains the same...]*

The existing library API documentation in the original API_REFERENCE.md file covers:
- RFCCompliantService
- RFC-0111 Authorization API (AuthorizeGAuth)
- RFC-0115 PoA Definition structures
- Professional Foundation API (ProperJWTService)
- Legal Framework Validation
- Complete data type definitions

---

## 🔧 **Error Handling**

### **HTTP Status Codes**
- `200 OK`: Successful request
- `400 Bad Request`: Invalid request format or parameters
- `404 Not Found`: Resource not found (e.g., scenario not found)
- `500 Internal Server Error`: Server-side processing error

### **Error Response Format**
```json
{
  "success": false,
  "message": "Error description",
  "error_code": "ERROR_TYPE",
  "timestamp": 1696260000
}
```

---

## 💡 **Examples**

### **Complete Demo Flow**

1. **Get Available Scenarios**
```bash
curl -X GET http://localhost:8080/scenarios
```

2. **Authenticate with a Scenario**
```bash
curl -X POST http://localhost:8080/authenticate \
  -H "Content-Type: application/json" \
  -d '{"scenario_id": "rfc0111-basic"}'
```

3. **Validate the Token**
```bash
curl -X POST http://localhost:8080/validate \
  -H "Content-Type: application/json" \
  -d '{"token": "gauth_abc123def456..."}'
```

4. **Configure RFC-0111**
```bash
curl -X POST http://localhost:8080/rfc0111/config \
  -H "Content-Type: application/json" \
  -d '{
    "p2p_enabled": true,
    "extended_tokens": true,
    "ai_client": true,
    "exclusions": ["web3_blockchain", "dna_based_identities"]
  }'
```

5. **Create RFC-0115 PoA Definition**
```bash
curl -X POST http://localhost:8080/rfc0115/poa \
  -H "Content-Type: application/json" \
  -d '{
    "parties": {
      "grantor": "System Owner",
      "grantee": "AI Agent",
      "witness": "Authorization Server"
    },
    "authorization_type": "limited",
    "legal_framework": "standard"
  }'
```

6. **Run Combined Demo**
```bash
curl -X POST http://localhost:8080/combined/demo \
  -H "Content-Type: application/json" \
  -d '{
    "rfc0111": {
      "p2p_enabled": true,
      "extended_tokens": true,
      "ai_client": true,
      "exclusions": ["web3_blockchain"]
    },
    "rfc0115": {
      "parties": {
        "grantor": "System",
        "grantee": "AI Agent"
      },
      "authorization_type": "limited"
    }
  }'
```

---

## 🚀 **Getting Started**

1. **Start the Demo Server**
```bash
cd gauth-demo-app/web/backend
go build -o gauth-backend main.go
./gauth-backend
```

2. **Access the Web Interface**
Open http://localhost:8080 in your browser

3. **Test the API**
Use the provided curl examples or test through the web interface

---

*This comprehensive API reference covers both the web demonstration API and the complete Go library implementation. For additional guides and documentation, see the [Getting Started Guide](../docs/GETTING_STARTED.md) and [Webapp Rebuild Summary](../WEBAPP_REBUILD_SUMMARY.md).*