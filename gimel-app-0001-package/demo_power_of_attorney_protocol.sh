#!/bin/bash

# GAuth Power-of-Attorney Protocol (P*P) Demonstration
# Showcasing POWER DELEGATION vs POLICY ENFORCEMENT

echo "========================================================"
echo "GAuth Power-of-Attorney Protocol (P*P) Demonstration"
echo "========================================================"
echo "🚀 PARADIGM SHIFT: Power Delegation vs Policy Enforcement"
echo "📋 The 'P' in P*P = POWER-OF-ATTORNEY (not policies!)"
echo ""

echo "🏢 TRADITIONAL IT MODEL (What GAuth REPLACES):"
echo "   - IT creates policies"
echo "   - Technical rules govern access"
echo "   - IT is RESPONSIBLE for decisions"
echo ""

echo "⚡ GAUTH P*P MODEL (Revolutionary Approach):"
echo "   - Business owners DELEGATE powers"
echo "   - Legal frameworks govern authorization"
echo "   - Business owners are ACCOUNTABLE for decisions"
echo ""

# Check server status
echo "1. Power Authorization Server Status:"
echo "-------------------------------------"
curl -s http://localhost:8080/health | jq .
echo ""

# Demonstrate Business Owner Power Delegation (RFC111)
echo "2. Business Owner Power Delegation (RFC111):"
echo "---------------------------------------------"
echo "🎯 BUSINESS CONTEXT: CFO delegates financial authority to AI assistant"
echo "💼 ACCOUNTABILITY: CFO remains accountable (not IT department)"
echo "⚖️ LEGAL BASIS: Corporate power-of-attorney legislation"
echo ""

curl -X POST http://localhost:8080/api/v1/rfc111/authorize \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "cfo_ai_assistant",
    "response_type": "code", 
    "scope": ["financial_power_of_attorney", "corporate_transactions", "regulatory_compliance"],
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
    },
    "legal_framework": {
      "jurisdiction": "US",
      "entity_type": "corporation", 
      "capacity_verification": true,
      "business_accountability_rules": ["executive_oversight", "board_reporting", "regulatory_compliance"]
    },
    "delegated_powers": ["authorize_payments", "sign_contracts", "manage_investments", "regulatory_filings"],
    "business_restrictions": {
      "amount_limit": 500000,
      "geo_restrictions": ["US", "EU"],
      "business_hours_only": true,
      "approval_threshold": 100000,
      "executive_oversight_required": true
    },
    "accountability_context": {
      "business_justification": "AI delegation for operational efficiency while maintaining CFO accountability",
      "legal_responsibility": "CFO remains legally responsible for all delegated actions",
      "compliance_framework": ["SOX", "GAAP", "SEC_regulations"]
    }
  }' | jq .
echo ""

# Demonstrate Advanced Business Power Delegation (RFC115)
echo "3. Advanced Business Power Delegation (RFC115):"
echo "------------------------------------------------"
echo "🎯 BUSINESS CONTEXT: Board Chair delegates governance powers with enhanced attestation"
echo "💼 ACCOUNTABILITY: Board Chair with multi-signature attestation requirements"
echo "⚖️ LEGAL BASIS: Corporate governance and fiduciary responsibility laws"
echo ""

curl -X POST http://localhost:8080/api/v1/rfc115/delegation \
  -H "Content-Type: application/json" \
  -d '{
    "principal_id": "board_chair_robert_wilson",
    "delegate_id": "governance_ai_system", 
    "power_type": "corporate_governance_delegation",
    "scope": ["board_resolutions", "shareholder_communications", "regulatory_filings", "compliance_monitoring"],
    "business_owner": {
      "owner_id": "board_chair_robert_wilson",
      "role": "Chairman of the Board",
      "department": "Corporate Governance",
      "delegation_authority": "fiduciary_powers",
      "accountability_level": "board_fiduciary"
    },
    "power_restrictions": {
      "amount_limit": 1000000,
      "geo_restrictions": ["US", "EU", "CA"],
      "business_hours_only": false,
      "board_oversight_required": true,
      "fiduciary_compliance": true
    },
    "attestation_requirement": {
      "type": "board_resolution_signature",
      "level": "fiduciary_enhanced", 
      "multi_signature": true,
      "attesters": ["board_secretary", "legal_counsel", "compliance_officer"],
      "business_witnesses": ["independent_director", "audit_committee_chair"]
    },
    "validity_period": {
      "start_time": "2025-09-23T15:00:00Z",
      "end_time": "2026-09-23T15:00:00Z",
      "business_review_periods": [
        {
          "quarterly_review": true,
          "board_approval_renewal": true,
          "fiduciary_assessment": true
        }
      ],
      "governance_constraints": ["board_oversight", "shareholder_transparency"]
    },
    "jurisdiction": "US",
    "legal_basis": "corporate_fiduciary_responsibility_act_2024",
    "business_accountability": {
      "fiduciary_responsibility": "Board Chair maintains fiduciary duty for all delegated governance actions",
      "shareholder_accountability": "Actions must align with shareholder interests and corporate governance standards",
      "regulatory_compliance": ["SEC", "NYSE", "corporate_governance_standards"]
    }
  }' | jq .
echo ""

echo "========================================================"
echo "🎯 POWER-OF-ATTORNEY PROTOCOL (P*P) IMPLEMENTATION:"
echo "========================================================"
echo "✅ Business Owner Authority: Functional owners delegate powers"
echo "✅ Legal Framework Compliance: Power-of-attorney legal basis"
echo "✅ Business Accountability: Owners accountable for delegations"  
echo "✅ Contextual Authorization: Business relationships drive access"
echo "✅ Regulatory Alignment: Compliance with PoA legislation"
echo "✅ Enterprise Scalability: Distributed business delegation"
echo ""
echo "🚫 NOT IMPLEMENTED (Traditional IT Model):"
echo "🚫 Policy-based permission systems"
echo "🚫 IT-administered access controls"
echo "🚫 Technical rule-based authorization"
echo "🚫 Administrative responsibility by IT teams"
echo ""
echo "🎯 THE REVOLUTION: Power Delegation vs Policy Administration"
echo "💼 Business owners CONTROL delegations (not IT bottlenecks)"
echo "⚖️ Legal frameworks GOVERN authorization (not technical policies)"
echo "🎯 Functional accountability DRIVES decisions (not IT responsibility)"
echo ""
echo "🌟 GAUTH P*P: Where the first 'P' truly means POWER-OF-ATTORNEY!"
echo ""