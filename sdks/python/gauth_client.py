"""
AgentAuth Python SDK

Official Python client library for the AgentAuth OAuth 2.0 Authorization Server.
Supports RFC-0111 subscription flow, Power of Attorney, and more.

Version: 1.0.0-beta
Author: AgentAuth Team
License: MIT
"""

from typing import Dict, List, Optional, Any, Literal
from dataclasses import dataclass
from datetime import datetime
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry


# ============================================================================
# Data Classes
# ============================================================================

@dataclass
class Subscription:
    """RFC-0111 Subscription"""
    id: str
    client_id: str
    scope: str
    status: Literal['initiated', 'in_progress', 'completed', 'failed']
    current_step: Literal['I', 'II', 'III', 'IV', 'V', 'VI', 'VII', 'VIII']
    created_at: str
    updated_at: str
    completed_at: Optional[str] = None


@dataclass
class PoA:
    """Power of Attorney"""
    id: str
    grantor: str
    grantee: str
    scope: List[str]
    valid_from: str
    valid_until: str
    status: Literal['active', 'expired', 'revoked']
    created_at: str
    updated_at: str
    use_count: int
    resource_pattern: Optional[str] = None
    max_uses: Optional[int] = None
    revoked_at: Optional[str] = None


@dataclass
class Token:
    """Access Token"""
    access_token: str
    token_type: str
    expires_in: int
    scope: str
    issued_at: str


# ============================================================================
# Exceptions
# ============================================================================

class AgentAuthError(Exception):
    """Base exception for AgentAuth SDK"""
    
    def __init__(self, error: str, error_description: str, 
                 error_uri: Optional[str] = None, timestamp: Optional[str] = None):
        self.error = error
        self.error_description = error_description
        self.error_uri = error_uri
        self.timestamp = timestamp or datetime.utcnow().isoformat()
        super().__init__(f"{error}: {error_description}")


class AgentAuthHTTPError(AgentAuthError):
    """HTTP error from AgentAuth API"""
    
    def __init__(self, response: requests.Response):
        try:
            data = response.json()
            error = data.get('error', 'unknown_error')
            error_description = data.get('error_description', f'HTTP {response.status_code}')
            error_uri = data.get('error_uri')
            timestamp = data.get('timestamp')
        except Exception:
            error = 'unknown_error'
            error_description = f'HTTP {response.status_code}: {response.reason}'
            error_uri = None
            timestamp = None
        
        super().__init__(error, error_description, error_uri, timestamp)
        self.status_code = response.status_code
        self.response = response


# ============================================================================
# Main Client Class
# ============================================================================

class AgentAuthClient:
    """
    AgentAuth API Client
    
    Example usage:
        >>> client = AgentAuthClient(base_url='http://localhost:8080')
        >>> # Complete RFC-0111 subscription flow
        >>> result = client.complete_subscription_flow(
        ...     client_id='my-app',
        ...     scope='read write'
        ... )
        >>> client.access_token = result['token']
        >>> # Create Power of Attorney
        >>> poa = client.create_poa(
        ...     grantor='alice@example.com',
        ...     grantee='bob@example.com',
        ...     scope=['read:documents'],
        ...     valid_from='2025-01-01T00:00:00Z',
        ...     valid_until='2025-12-31T23:59:59Z'
        ... )
    """
    
    def __init__(self, base_url: str, api_key: Optional[str] = None, 
                 access_token: Optional[str] = None, timeout: int = 30):
        """
        Initialize AgentAuth client
        
        Args:
            base_url: Base URL of the AgentAuth API (e.g., 'http://localhost:8080')
            api_key: API key for authentication (optional)
            access_token: OAuth 2.0 access token (optional)
            timeout: Request timeout in seconds (default: 30)
        """
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.access_token = access_token
        self.timeout = timeout
        
        # Configure session with retries
        self.session = requests.Session()
        retry_strategy = Retry(
            total=3,
            backoff_factor=1,
            status_forcelist=[429, 500, 502, 503, 504],
            allowed_methods=["HEAD", "GET", "OPTIONS", "POST", "PUT", "DELETE"]
        )
        adapter = HTTPAdapter(max_retries=retry_strategy)
        self.session.mount("http://", adapter)
        self.session.mount("https://", adapter)
    
    def _get_headers(self) -> Dict[str, str]:
        """Get request headers with authentication"""
        headers = {'Content-Type': 'application/json'}
        
        if self.access_token:
            headers['Authorization'] = f'Bearer {self.access_token}'
        
        if self.api_key:
            headers['X-API-Key'] = self.api_key
        
        return headers
    
    def _request(self, method: str, path: str, **kwargs) -> Any:
        """Make HTTP request to AgentAuth API"""
        url = f"{self.base_url}{path}"
        headers = self._get_headers()
        headers.update(kwargs.pop('headers', {}))
        
        try:
            response = self.session.request(
                method=method,
                url=url,
                headers=headers,
                timeout=self.timeout,
                **kwargs
            )
            response.raise_for_status()
            return response.json()
        except requests.HTTPError:
            raise AgentAuthHTTPError(response)
        except requests.RequestException as e:
            raise AgentAuthError('request_failed', str(e))
    
    # ========================================================================
    # RFC-0111 Subscription Flow Methods
    # ========================================================================
    
    def create_subscription(self, client_id: str, scope: str, 
                          redirect_uri: Optional[str] = None,
                          state: Optional[str] = None) -> Subscription:
        """
        Create a new subscription (Step I)
        
        Args:
            client_id: Client identifier
            scope: Requested scope
            redirect_uri: Redirect URI (optional)
            state: State parameter (optional)
        
        Returns:
            Subscription object
        """
        data = {'client_id': client_id, 'scope': scope}
        if redirect_uri:
            data['redirect_uri'] = redirect_uri
        if state:
            data['state'] = state
        
        result = self._request('POST', '/api/v1/rfc0111/subscriptions', json=data)
        return Subscription(**result)
    
    def get_subscription(self, subscription_id: str) -> Subscription:
        """Get subscription details"""
        result = self._request('GET', f'/api/v1/rfc0111/subscriptions/{subscription_id}')
        return Subscription(**result)
    
    def list_subscriptions(self, client_id: Optional[str] = None,
                          status: Optional[str] = None,
                          limit: int = 50, offset: int = 0) -> Dict[str, Any]:
        """
        List subscriptions with optional filters
        
        Returns:
            Dictionary with 'subscriptions' list and 'total' count
        """
        params = {'limit': limit, 'offset': offset}
        if client_id:
            params['client_id'] = client_id
        if status:
            params['status'] = status
        
        return self._request('GET', '/api/v1/rfc0111/subscriptions', params=params)
    
    def execute_step_ii(self, subscription_id: str, proof_type: str, 
                       proof_data: str) -> Dict[str, Any]:
        """Execute Step II - Authorizer Authentication (PVP)"""
        data = {'proof_type': proof_type, 'proof_data': proof_data}
        return self._request('POST', f'/api/v1/rfc0111/subscriptions/{subscription_id}/step-ii', json=data)
    
    def execute_step_iii(self, subscription_id: str) -> Dict[str, Any]:
        """Execute Step III - Client Owner Identification"""
        return self._request('POST', f'/api/v1/rfc0111/subscriptions/{subscription_id}/step-iii')
    
    def execute_step_iv(self, subscription_id: str) -> Dict[str, Any]:
        """Execute Step IV - Client Owner Authorization"""
        return self._request('POST', f'/api/v1/rfc0111/subscriptions/{subscription_id}/step-iv')
    
    def execute_step_v(self, subscription_id: str) -> Dict[str, Any]:
        """Execute Step V - Client Authorization"""
        return self._request('POST', f'/api/v1/rfc0111/subscriptions/{subscription_id}/step-v')
    
    def execute_step_vi(self, subscription_id: str) -> Dict[str, Any]:
        """Execute Step VI - Resource Owner Identification"""
        return self._request('POST', f'/api/v1/rfc0111/subscriptions/{subscription_id}/step-vi')
    
    def execute_step_vii(self, subscription_id: str) -> Dict[str, Any]:
        """Execute Step VII - Resource Owner Authorization"""
        return self._request('POST', f'/api/v1/rfc0111/subscriptions/{subscription_id}/step-vii')
    
    def execute_step_viii(self, subscription_id: str) -> Dict[str, Any]:
        """
        Execute Step VIII - Resource Server Verification (Final Step)
        
        Returns:
            Dictionary with 'status', 'token', 'token_type', and 'expires_in'
        """
        return self._request('POST', f'/api/v1/rfc0111/subscriptions/{subscription_id}/step-viii')
    
    def complete_subscription_flow(self, client_id: str, scope: str,
                                   redirect_uri: Optional[str] = None,
                                   state: Optional[str] = None) -> Dict[str, Any]:
        """
        Complete entire RFC-0111 subscription flow automatically
        
        Executes all 8 steps in sequence and returns the access token.
        Automatically sets the access token on the client instance.
        
        Returns:
            Dictionary with 'subscription' and 'token'
        """
        # Step I: Create subscription
        subscription = self.create_subscription(client_id, scope, redirect_uri, state)
        
        # Steps II-VIII: Execute automatically
        self.execute_step_ii(subscription.id, 'document', 'auto_verified')
        self.execute_step_iii(subscription.id)
        self.execute_step_iv(subscription.id)
        self.execute_step_v(subscription.id)
        self.execute_step_vi(subscription.id)
        self.execute_step_vii(subscription.id)
        result = self.execute_step_viii(subscription.id)
        
        # Set access token
        self.access_token = result['token']
        
        return {
            'subscription': self.get_subscription(subscription.id),
            'token': result['token']
        }
    
    # ========================================================================
    # Power of Attorney (PoA) Methods
    # ========================================================================
    
    def create_poa(self, grantor: str, grantee: str, scope: List[str],
                   valid_from: str, valid_until: str,
                   resource_pattern: Optional[str] = None,
                   max_uses: Optional[int] = None,
                   constraints: Optional[Dict[str, Any]] = None) -> PoA:
        """
        Create a new Power of Attorney
        
        Args:
            grantor: Entity granting the authority
            grantee: Entity receiving the authority
            scope: List of granted permissions
            valid_from: Start of validity period (ISO 8601)
            valid_until: End of validity period (ISO 8601)
            resource_pattern: Resource path pattern (optional)
            max_uses: Maximum number of uses (optional)
            constraints: Additional constraints (optional)
        
        Returns:
            PoA object
        """
        data = {
            'grantor': grantor,
            'grantee': grantee,
            'scope': scope,
            'valid_from': valid_from,
            'valid_until': valid_until
        }
        if resource_pattern:
            data['resource_pattern'] = resource_pattern
        if max_uses is not None:
            data['max_uses'] = max_uses
        if constraints:
            data['constraints'] = constraints
        
        result = self._request('POST', '/api/v1/beta/poa', json=data)
        return PoA(**result)
    
    def get_poa(self, poa_id: str) -> PoA:
        """Get PoA details"""
        result = self._request('GET', f'/api/v1/beta/poa/{poa_id}')
        return PoA(**result)
    
    def list_poas(self, grantor: Optional[str] = None,
                  grantee: Optional[str] = None,
                  status: Optional[str] = None,
                  limit: int = 50, offset: int = 0) -> Dict[str, Any]:
        """
        List Powers of Attorney with optional filters
        
        Returns:
            Dictionary with 'poas' list and 'total' count
        """
        params = {'limit': limit, 'offset': offset}
        if grantor:
            params['grantor'] = grantor
        if grantee:
            params['grantee'] = grantee
        if status:
            params['status'] = status
        
        return self._request('GET', '/api/v1/beta/poa', params=params)
    
    def update_poa(self, poa_id: str, scope: Optional[List[str]] = None,
                   valid_until: Optional[str] = None) -> PoA:
        """Update a Power of Attorney"""
        data = {}
        if scope:
            data['scope'] = scope
        if valid_until:
            data['valid_until'] = valid_until
        
        result = self._request('PUT', f'/api/v1/beta/poa/{poa_id}', json=data)
        return PoA(**result)
    
    def revoke_poa(self, poa_id: str) -> Dict[str, Any]:
        """Revoke a Power of Attorney"""
        return self._request('DELETE', f'/api/v1/beta/poa/{poa_id}')
    
    def validate_poa(self, poa_id: str, action: str, 
                    resource: Optional[str] = None,
                    context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        Validate a Power of Attorney for a specific action
        
        Returns:
            Dictionary with 'valid', 'poa_id', 'action', 'reason' (if invalid), 'validated_at'
        """
        data = {'action': action}
        if resource:
            data['resource'] = resource
        if context:
            data['context'] = context
        
        return self._request('POST', f'/api/v1/beta/poa/{poa_id}/validate', json=data)
    
    # ========================================================================
    # PVP (Person Verification Protocol) Methods
    # ========================================================================
    
    def verify_pvp(self, subject: str, proof_type: str, proof_data: str) -> Dict[str, Any]:
        """
        Verify identity via PVP
        
        Args:
            subject: Subject identifier
            proof_type: Type of proof ('document', 'biometric', or 'challenge')
            proof_data: Base64-encoded proof data
        
        Returns:
            Dictionary with verification result
        """
        data = {
            'subject': subject,
            'proof_type': proof_type,
            'proof_data': proof_data
        }
        return self._request('POST', '/api/v1/beta/pvp/verify', json=data)
    
    # ========================================================================
    # Commercial Registry Methods
    # ========================================================================
    
    def verify_entity(self, entity_id: str, registry: str) -> Dict[str, Any]:
        """Verify a commercial entity"""
        data = {'entity_id': entity_id, 'registry': registry}
        return self._request('POST', '/api/v1/beta/registry/verify-entity', json=data)
    
    def verify_signatory(self, signatory_id: str, entity_id: str, 
                        role: str) -> Dict[str, Any]:
        """Verify signatory authority"""
        data = {
            'signatory_id': signatory_id,
            'entity_id': entity_id,
            'role': role
        }
        return self._request('POST', '/api/v1/beta/registry/verify-signatory', json=data)
    
    # ========================================================================
    # Authorization Methods
    # ========================================================================
    
    def evaluate_authorization(self, subject: str, action: str, 
                             resource: str, context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        Evaluate an authorization request
        
        Returns:
            Dictionary with 'decision' ('permit' or 'deny'), 'policy_id', 'reason', etc.
        """
        data = {
            'subject': subject,
            'action': action,
            'resource': resource
        }
        if context:
            data['context'] = context
        
        return self._request('POST', '/api/v1/beta/authz/evaluate', json=data)
    
    def get_authz_metrics(self) -> Dict[str, Any]:
        """Get authorization metrics"""
        return self._request('GET', '/api/v1/beta/authz/metrics')
    
    # ========================================================================
    # Token Methods
    # ========================================================================
    
    def create_token(self, client_id: str, scope: str, 
                    expires_in: Optional[int] = None) -> Token:
        """Create a new token"""
        data = {'client_id': client_id, 'scope': scope}
        if expires_in:
            data['expires_in'] = expires_in
        
        result = self._request('POST', '/api/v1/token/create', json=data)
        return Token(**result)
    
    def validate_token(self, token: str) -> Dict[str, Any]:
        """Validate a token"""
        return self._request('POST', '/api/v1/token/validate', json={'token': token})
    
    def revoke_token(self, token: str) -> Dict[str, Any]:
        """Revoke a token"""
        return self._request('POST', '/api/v1/token/revoke', json={'token': token})
    
    # ========================================================================
    # System Methods
    # ========================================================================
    
    def health(self) -> Dict[str, Any]:
        """Health check"""
        return self._request('GET', '/api/v1/beta/health')
    
    def info(self) -> Dict[str, Any]:
        """Get server information"""
        return self._request('GET', '/api/v1/beta/info')
    
    def ping(self) -> Dict[str, Any]:
        """Ping endpoint"""
        return self._request('GET', '/api/v1/beta/ping')


# ============================================================================
# Convenience Functions
# ============================================================================

def create_client(base_url: str = 'http://localhost:8080', **kwargs) -> AgentAuthClient:
    """
    Create a AgentAuth client with default settings
    
    Example:
        >>> client = create_client()
        >>> client.health()
    """
    return AgentAuthClient(base_url=base_url, **kwargs)
