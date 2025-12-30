# API Keys Acquisition Guide

## Task 5: Obtain Sandbox API Keys for Identity Verification

This guide provides step-by-step instructions for obtaining sandbox/test API keys from Persona and Trulioo.

---

## 🔐 Persona API Keys

### Overview
Persona provides identity verification services with support for 200+ countries. Their sandbox environment allows testing without real data.

### Steps to Obtain API Keys

#### 1. Create Persona Account
1. Visit [https://withpersona.com/](https://withpersona.com/)
2. Click "Get Started" or "Sign Up"
3. Fill in:
   - Email address
   - Company name
   - Password
4. Verify your email address

#### 2. Access Dashboard
1. Log in to [https://dashboard.withpersona.com/](https://dashboard.withpersona.com/)
2. Complete the onboarding questionnaire:
   - Business type
   - Use case (select "Testing/Development")
   - Expected volume

#### 3. Get API Keys
1. Navigate to **Settings** → **API Keys**
2. You'll see two environments:
   - **Sandbox**: For testing (uses test data)
   - **Production**: For live verification (requires approval)

3. Copy the **Sandbox API Key**:
   ```
   Format: sk_sand_xxxxxxxxxxxxxxxxxxxxxxxxxx
   ```

#### 4. Test the API
```bash
# Test Persona API
curl https://withpersona.com/api/v1/verifications \
  -H "Authorization: Bearer YOUR_SANDBOX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "type": "verification/government-id",
      "attributes": {
        "inquiry-template-id": "itmpl_xxxxx"
      }
    }
  }'
```

#### 5. Documentation
- API Docs: [https://docs.withpersona.com/](https://docs.withpersona.com/)
- Sandbox Guide: [https://docs.withpersona.com/docs/sandbox-data](https://docs.withpersona.com/docs/sandbox-data)

### Sandbox Test Data
Persona provides test identities for various countries:

**United States:**
- Name: Alex Smith
- DOB: 1980-01-01
- SSN: 123-45-6789

**Canada:**
- Name: John Doe
- DOB: 1990-01-15
- SIN: 046-454-286

**Brazil:**
- Name: João Silva
- DOB: 1985-03-20
- CPF: 123.456.789-09

**Full list:** [https://docs.withpersona.com/docs/sandbox-data](https://docs.withpersona.com/docs/sandbox-data)

### Integration Example
```go
// pkg/agentauth/external/persona_connector.go
package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type PersonaConnectorConfig struct {
	APIKey         string
	BaseURL        string // Default: https://withpersona.com/api/v1
	RequestTimeout time.Duration
}

type PersonaConnector struct {
	config *PersonaConnectorConfig
	client *http.Client
}

func NewPersonaConnector(config *PersonaConnectorConfig) *PersonaConnector {
	if config.BaseURL == "" {
		config.BaseURL = "https://withpersona.com/api/v1"
	}
	return &PersonaConnector{
		config: config,
		client: &http.Client{Timeout: config.RequestTimeout},
	}
}

func (c *PersonaConnector) VerifyIdentity(ctx context.Context, req *VerificationRequest) (*VerificationResponse, error) {
	// Implementation here
}
```

---

## 🌍 Trulioo API Keys

### Overview
Trulioo GlobalGateway provides identity verification in 195+ countries with support for 5 billion+ individuals.

### Steps to Obtain API Keys

#### 1. Create Trulioo Account
1. Visit [https://www.trulioo.com/](https://www.trulioo.com/)
2. Click "Get Started" or "Request Demo"
3. Fill out the contact form:
   - First name, Last name
   - Email
   - Company name
   - Country
   - Phone number

#### 2. Trial Account Setup
1. Trulioo team will contact you (usually within 1-2 business days)
2. Request a **Trial Account** specifically for testing
3. You'll receive:
   - Portal access credentials
   - API credentials
   - Trial credits (usually $100-$500 worth)

#### 3. Access Developer Portal
1. Log in to [https://gateway-admin.trulioo.com/](https://gateway-admin.trulioo.com/)
2. Navigate to **Settings** → **API Keys**
3. Generate new API keys:
   - **Username**: Your API username
   - **Password**: Your API password

#### 4. Get API Credentials
The credentials will be in this format:
```
Username: trial_username_xxxxx
Password: xxxxxxxxxxxxxxxxxxxxxxx
```

#### 5. Test the API
```bash
# Test Trulioo API - Get Country Codes
curl https://api.globaldatacompany.com/configuration/v1/countrycodes/Identity%20Verification \
  -u "trial_username:trial_password" \
  -H "Content-Type: application/json"

# Verify Identity (Brazil example)
curl https://api.globaldatacompany.com/verifications/v1/verify \
  -u "trial_username:trial_password" \
  -H "Content-Type: application/json" \
  -d '{
    "AcceptTruliooTermsAndConditions": true,
    "CountryCode": "BR",
    "CustomerReferenceID": "test-001",
    "DataFields": {
      "PersonInfo": {
        "FirstGivenName": "João",
        "FirstSurName": "Silva",
        "DayOfBirth": 20,
        "MonthOfBirth": 3,
        "YearOfBirth": 1985
      },
      "NationalIds": [
        {
          "Type": "CPF",
          "Number": "12345678909"
        }
      ]
    }
  }'
```

#### 6. Documentation
- API Docs: [https://developer.trulioo.com/](https://developer.trulioo.com/)
- SDK & Libraries: [https://developer.trulioo.com/docs/sdks](https://developer.trulioo.com/docs/sdks)
- Country Coverage: [https://developer.trulioo.com/docs/supported-countries](https://developer.trulioo.com/docs/supported-countries)

### Test Mode
Trulioo provides test mode data:

**Test Record (Always Returns Match):**
```json
{
  "FirstGivenName": "Test",
  "FirstSurName": "Match",
  "DayOfBirth": 1,
  "MonthOfBirth": 1,
  "YearOfBirth": 1980
}
```

**Test Record (Always Returns No Match):**
```json
{
  "FirstGivenName": "Test",
  "FirstSurName": "NoMatch",
  "DayOfBirth": 1,
  "MonthOfBirth": 1,
  "YearOfBirth": 1980
}
```

### Integration Example
```go
// pkg/agentauth/external/trulioo_connector.go
package external

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TruliooConnectorConfig struct {
	Username       string
	Password       string
	BaseURL        string // Default: https://api.globaldatacompany.com
	RequestTimeout time.Duration
}

type TruliooConnector struct {
	config     *TruliooConnectorConfig
	client     *http.Client
	authHeader string
}

func NewTruliooConnector(config *TruliooConnectorConfig) *TruliooConnector {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.globaldatacompany.com"
	}
	
	// Create Basic Auth header
	auth := base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%s:%s", config.Username, config.Password)),
	)
	
	return &TruliooConnector{
		config:     config,
		client:     &http.Client{Timeout: config.RequestTimeout},
		authHeader: fmt.Sprintf("Basic %s", auth),
	}
}

func (c *TruliooConnector) VerifyIdentity(ctx context.Context, country string, req *TruliooVerificationRequest) (*TruliooVerificationResponse, error) {
	// Implementation here
}
```

---

## 📋 Configuration

### Environment Variables
Add these to your `.env` file:

```bash
# Persona
PERSONA_API_KEY=sk_sand_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
PERSONA_BASE_URL=https://withpersona.com/api/v1

# Trulioo
TRULIOO_USERNAME=trial_username_xxxxx
TRULIOO_PASSWORD=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
TRULIOO_BASE_URL=https://api.globaldatacompany.com
```

### Application Configuration
```yaml
# config/external_connectors.yaml
external_connectors:
  persona:
    enabled: true
    api_key: ${PERSONA_API_KEY}
    base_url: ${PERSONA_BASE_URL}
    timeout: 30s
    
  trulioo:
    enabled: true
    username: ${TRULIOO_USERNAME}
    password: ${TRULIOO_PASSWORD}
    base_url: ${TRULIOO_BASE_URL}
    timeout: 30s
```

---

## 🧪 Testing

### Run Integration Tests
```bash
# Test Persona connector
go test -v ./pkg/agentauth/external -run TestPersonaConnector

# Test Trulioo connector
go test -v ./pkg/agentauth/external -run TestTruliooConnector

# Run all external connector tests
go test -v ./pkg/agentauth/external/...
```

### Manual Testing
```bash
# Test Persona verification
curl -X POST http://localhost:8080/api/v1/external/persona/verify \
  -H "Content-Type: application/json" \
  -d '{
    "inquiry_template_id": "itmpl_xxxxx",
    "reference_id": "test-001",
    "country": "US"
  }'

# Test Trulioo verification
curl -X POST http://localhost:8080/api/v1/external/trulioo/verify \
  -H "Content-Type: application/json" \
  -d '{
    "country_code": "BR",
    "customer_reference_id": "test-002",
    "first_given_name": "João",
    "first_sur_name": "Silva",
    "cpf": "12345678909"
  }'
```

---

## 📊 Next Steps

1. **✅ Obtain API Keys**: Follow the steps above
2. **✅ Update Configuration**: Add keys to `.env`
3. **✅ Implement Connectors**: Create connector files in `pkg/agentauth/external/`
4. **✅ Add Tests**: Create integration tests
5. **✅ Run Load Tests**: Use k6 script to test performance
6. **✅ Monitor Metrics**: Check Prometheus for API call metrics
7. **✅ Review Audit Logs**: Verify all verification attempts are logged

---

## 🔗 Additional Resources

- **Persona Documentation**: https://docs.withpersona.com/
- **Trulioo Documentation**: https://developer.trulioo.com/
- **Identity Verification Best Practices**: https://www.trulioo.com/resources/
- **Regulatory Compliance**: GDPR, CCPA, KYC/AML guidelines

---

## 📞 Support

If you encounter issues:

1. **Persona Support**: support@withpersona.com
2. **Trulioo Support**: support@trulioo.com or use the chat in the portal
3. **Internal Team**: Slack channel #identity-verification
