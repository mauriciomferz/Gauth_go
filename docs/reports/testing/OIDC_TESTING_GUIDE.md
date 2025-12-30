---
title: Oidc Testing Guide
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# OIDC Testing Guide

## Quick Start Testing

### Prerequisites
- ✅ Backend running on port 8080
- ✅ Frontend built successfully
- ✅ Database migrations applied
- ✅ At least one OIDC provider configured

## Testing Steps

### 1. Test Provider Configuration (Admin)

Navigate to OIDC Providers management:
```
http://localhost:8080/admin/oidc-providers
```

**Actions**:
- View existing providers
- Create a new provider (Azure AD, Google, Okta, etc.)
- Configure client ID, client secret, and endpoints
- Enable auto-provisioning
- Set attribute mappings
- Define default role
- Save provider

### 2. Test Provider Selection Page

Navigate to OIDC Login:
```
http://localhost:8080/oidc-login
```

**Expected**:
- See list of active OIDC providers
- Provider cards with icons and colors
- Click provider redirects to OAuth authorize URL

**Verify URL Parameters**:
```
https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize
  ?client_id={client_id}
  &response_type=code
  &redirect_uri={callback_url}
  &scope=openid+profile+email
  &state={random_state}
  &code_challenge={pkce_challenge}
  &code_challenge_method=S256
```

### 3. Test OAuth Flow

**Steps**:
1. Click a provider card (e.g., Azure AD)
2. Redirected to identity provider login
3. Enter credentials at provider
4. Grant consent (if requested)
5. Redirected back to `/auth/callback?code=...&state=...`

**Expected Callback Behavior**:
- Loading spinner appears
- Backend processes code exchange
- Session created
- User provisioned (if new)
- Success message with user info
- Auto-redirect to dashboard after 2 seconds

### 4. Verify Session Storage

Open browser DevTools → Application → Local Storage:

**Check for**:
```javascript
// auth_session
{
  "sessionId": "sess_abc123...",
  "userId": "user_xyz789...",
  "user": {
    "email": "user@example.com",
    "name": "John Doe",
    "sub": "provider-specific-id"
  },
  "timestamp": "2024-11-12T22:30:00.000Z"
}

// user_id
"user_xyz789..."

// user_email
"user@example.com"
```

### 5. Verify Database Records

Check database after successful login:

```sql
-- Check user created
SELECT * FROM users WHERE email = 'user@example.com';

-- Check OIDC mapping
SELECT * FROM oidc_user_mappings 
WHERE user_id = (SELECT id FROM users WHERE email = 'user@example.com');

-- Check role assigned
SELECT * FROM user_roles 
WHERE user_id = (SELECT id FROM users WHERE email = 'user@example.com');

-- Check groups synced (if provider sends groups)
SELECT * FROM oidc_user_groups 
WHERE user_id = (SELECT id FROM users WHERE email = 'user@example.com');

-- Check last login updated
SELECT email, last_login_at FROM users WHERE email = 'user@example.com';
```

### 6. Test Error Scenarios

#### Invalid Provider ID
```
http://localhost:8080/auth/authorize?provider_id=invalid
```
**Expected**: 404 error page or redirect to login with error

#### User Denies Consent
At identity provider, click "Cancel" or "Deny"

**Expected**: 
- Redirect to `/auth/callback?error=access_denied&error_description=...`
- Error page displayed with message
- "Try Again" button returns to `/oidc-login`

#### Network Error
Disconnect network during callback processing

**Expected**:
- Error message displayed
- Retry option available

#### Auto-Provisioning Disabled
Set `auto_provision_users=false` for provider

**Expected**:
- New users rejected with error
- Existing users can still log in

### 7. Test Navigation Flow

**From Traditional Login**:
1. Navigate to `/login`
2. Click "Sign in with SSO Provider"
3. Redirected to `/oidc-login`
4. Select provider
5. Complete authentication
6. End up at dashboard

**Direct Access**:
1. Navigate directly to `/oidc-login`
2. Select provider
3. Complete flow

### 8. Test Multiple Providers

If you have multiple providers configured:

**Test**:
- Azure AD login
- Google login
- Okta login
- Verify each creates correct mapping
- Check provider_id is correct in database

### 9. Test Return URL

**Scenario**: User tries to access protected page, redirected to login

**Steps**:
1. Navigate to `/admin/dashboard` (while logged out)
2. Redirected to login
3. Click SSO button → stores return URL
4. Complete OIDC login
5. Should return to `/admin/dashboard` (not generic landing page)

### 10. Test Session Persistence

**Steps**:
1. Log in via OIDC
2. Close browser tab
3. Open new tab
4. Navigate to app
5. Should still be logged in (session in localStorage)

**Test Logout** (when implemented):
1. Click logout
2. localStorage cleared
3. Session invalidated on backend
4. Redirected to login

## Testing with Real Providers

### Azure AD Setup

1. **Register Application** in Azure Portal:
   - App registrations → New registration
   - Name: "AgentAuth Development"
   - Redirect URI: `http://localhost:8080/auth/callback`
   - Platform: Web

2. **Configure**:
   - Copy Application (client) ID
   - Create client secret (Certificates & secrets)
   - Copy secret value
   - Configure scopes: `openid`, `profile`, `email`

3. **In AgentAuth Admin UI**:
   - Create new OIDC provider
   - Type: Azure AD
   - Client ID: {from Azure}
   - Client Secret: {from Azure}
   - Tenant ID: {your tenant}
   - Discovery URL: `https://login.microsoftonline.com/{tenant}/v2.0/.well-known/openid-configuration`
   - Enable auto-provisioning
   - Set default role: "user"
   - Save

4. **Test**:
   - Go to `/oidc-login`
   - Click Azure AD
   - Sign in with Microsoft account
   - Verify callback and provisioning

### Google Workspace Setup

1. **Create OAuth Client** in Google Cloud Console:
   - APIs & Services → Credentials
   - Create OAuth client ID
   - Application type: Web application
   - Authorized redirect URIs: `http://localhost:8080/auth/callback`

2. **Configure**:
   - Copy client ID
   - Copy client secret
   - Enable Google+ API (for user info)

3. **In AgentAuth Admin UI**:
   - Create new OIDC provider
   - Type: Google
   - Client ID: {from Google}
   - Client Secret: {from Google}
   - Discovery URL: `https://accounts.google.com/.well-known/openid-configuration`
   - Scopes: `openid profile email`
   - Enable auto-provisioning
   - Save

4. **Test**: Similar to Azure AD

### Okta Setup

1. **Create OIDC Application** in Okta Admin:
   - Applications → Create App Integration
   - Sign-in method: OIDC
   - Application type: Web Application
   - Grant types: Authorization Code
   - Redirect URI: `http://localhost:8080/auth/callback`

2. **Configure**:
   - Copy client ID
   - Copy client secret
   - Note your Okta domain

3. **In AgentAuth Admin UI**:
   - Create new OIDC provider
   - Type: Okta
   - Client ID: {from Okta}
   - Client Secret: {from Okta}
   - Discovery URL: `https://{your-domain}.okta.com/.well-known/openid-configuration`
   - Save

4. **Test**: Similar to Azure AD

## Troubleshooting

### Issue: Redirect URI Mismatch
**Error**: "redirect_uri_mismatch"

**Solution**:
- Verify redirect URI in provider config matches exactly
- Check for trailing slashes
- Ensure protocol (http/https) matches

### Issue: Invalid State Parameter
**Error**: "Invalid state parameter"

**Solution**:
- State is stored in backend session
- Check session storage is working
- Verify state isn't expired (default 10 minutes)

### Issue: Token Validation Failed
**Error**: "Failed to verify ID token"

**Solution**:
- Check provider's JWKs endpoint is accessible
- Verify issuer claim matches expected value
- Ensure audience claim includes your client_id

### Issue: User Not Provisioned
**Error**: "User not found and auto-provisioning is disabled"

**Solution**:
- Enable auto-provisioning in provider config
- Or manually create user in database first
- Check required claims are present (email, sub)

### Issue: Groups Not Syncing
**Problem**: Groups in OIDC response not appearing in database

**Solution**:
- Verify provider is configured to send groups claim
- Check group claim path in provider config
- Ensure groups claim contains array of strings
- Test provider connection in admin UI

## Performance Testing

### Load Test Authentication Flow

Use tool like Apache Bench or k6:

```bash
# Example with k6
k6 run --vus 10 --duration 30s oidc-load-test.js
```

**Metrics to Monitor**:
- Response time for /auth/authorize
- Response time for /auth/callback
- Database query performance
- Token validation latency
- Cache hit rate for discovery documents

### Frontend Performance

**Check**:
- Page load time for /oidc-login
- Provider list fetch time
- Redirect performance
- Callback processing time

Use Chrome DevTools → Performance tab

## Security Testing

### Test PKCE
Verify code_challenge and code_verifier are used:
- Check authorize URL includes `code_challenge` and `code_challenge_method`
- Verify backend validates code_verifier on callback

### Test State Parameter
- Attempt callback with invalid state → Should fail
- Reuse state parameter → Should fail
- Omit state parameter → Should fail

### Test Token Validation
- Tamper with ID token → Should fail
- Use expired token → Should fail
- Use token from different issuer → Should fail

## Automated Testing (Future)

Consider adding:

```typescript
// Cypress E2E test example
describe('OIDC Login Flow', () => {
  it('should complete Azure AD login', () => {
    cy.visit('/oidc-login')
    cy.contains('Azure AD').click()
    // ... mock OAuth flow
    cy.url().should('include', '/admin/dashboard')
    cy.window().its('localStorage.auth_session').should('exist')
  })
})
```

## Success Criteria

✅ All providers load on `/oidc-login`  
✅ OAuth redirect includes correct parameters  
✅ Callback processes successfully  
✅ Session created and stored  
✅ User provisioned with correct attributes  
✅ Roles assigned from config  
✅ Groups synced from claims  
✅ Dashboard loads with user context  
✅ No console errors  
✅ No backend errors in logs  

## Reporting Issues

When reporting issues, include:
1. Provider type (Azure AD, Google, etc.)
2. Browser and version
3. Console errors (F12 → Console)
4. Network tab screenshot (F12 → Network)
5. Backend logs
6. Database state (relevant tables)
7. Steps to reproduce

---

**Happy Testing! 🚀**
