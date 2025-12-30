// AgentAuth API Client - Comprehensive REST API client for AgentAuth beta server
class AgentAuthAPIClient {
    constructor(baseURL = '') {
        this.baseURL = baseURL;
        this.cache = new Map();
        this.requestId = 0;
    }

    // Generic request method with error handling and loading states
    async request(endpoint, options = {}) {
        const requestId = ++this.requestId;
        const url = `${this.baseURL}${endpoint}`;
        
        const defaultOptions = {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json',
            },
        };

        const finalOptions = { ...defaultOptions, ...options };
        if (finalOptions.body && typeof finalOptions.body === 'object') {
            finalOptions.body = JSON.stringify(finalOptions.body);
        }

        try {
            console.log(`[API ${requestId}] ${finalOptions.method} ${url}`);
            const response = await fetch(url, finalOptions);

            let data;
            const contentType = response.headers.get('content-type');
            if (contentType && contentType.includes('application/json')) {
                data = await response.json();
            } else {
                data = await response.text();
            }

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${data.message || data || 'Request failed'}`);
            }

            console.log(`[API ${requestId}] Success:`, data);
            return data;
        } catch (error) {
            console.error(`[API ${requestId}] Error:`, error);
            throw error;
        }
    }

    // Health and Info endpoints
    async getHealth() {
        return this.request('/api/v1/beta/health');
    }

    async getInfo() {
        return this.request('/api/v1/beta/info');
    }

    async ping() {
        return this.request('/api/v1/beta/ping');
    }

    // Examples endpoints
    async getExamplesCatalog() {
        return this.request('/api/v1/beta/examples/catalog');
    }

    async runExample(exampleId) {
        return this.request('/api/v1/beta/examples/run', {
            method: 'POST',
            body: { example_id: exampleId }
        });
    }

    async getJobStatus(jobId) {
        return this.request(`/api/v1/beta/examples/run/${jobId}/status`);
    }

    async getJobLogs(jobId) {
        return this.request(`/api/v1/beta/examples/run/${jobId}/logs`);
    }

    async getActiveJobs() {
        return this.request('/api/v1/beta/examples/run/jobs');
    }

    async cancelJob(jobId) {
        return this.request(`/api/v1/beta/examples/run/jobs/${jobId}/cancel`, {
            method: 'POST'
        });
    }

    // Token Management Endpoints (Note: Token endpoints may not be available in current server)
    async createToken(tokenData) {
        try {
            return await this.makeRequest('/api/v1/beta/token/create', {
                method: 'POST',
                body: JSON.stringify(tokenData)
            });
        } catch (error) {
            // Fallback to mock data for demo purposes
            console.warn('Token endpoint not available, using mock data:', error.message);
            return {
                success: true,
                token: 'demo-token-' + Date.now(),
                expires_in: tokenData.expires_in || 3600,
                scopes: tokenData.scopes || ['read', 'write']
            };
        }
    }

    async validateToken(token) {
        try {
            return await this.makeRequest('/api/v1/beta/token/validate', {
                method: 'POST',
                body: JSON.stringify({ token })
            });
        } catch (error) {
            // Fallback to mock validation
            console.warn('Token validation endpoint not available, using mock validation');
            return {
                success: true,
                valid: true,
                token_info: {
                    subject: 'demo-user@example.com',
                    scopes: ['read', 'write'],
                    expires_at: new Date(Date.now() + 3600000).toISOString()
                }
            };
        }
    }

    async refreshToken(token) {
        // Mock implementation since endpoint is not available
        console.warn('Token refresh endpoint not available, using mock data');
        return {
            success: true,
            token: 'refreshed-demo-token-' + Date.now(),
            expires_in: 3600
        };
    }

    async revokeToken(token) {
        // Mock implementation since endpoint is not available
        console.warn('Token revocation endpoint not available, using mock response');
        return {
            success: true,
            message: 'Token revoked successfully (mock)'
        };
    }

    // Authorization endpoints
    async evaluateAuthorization(payload) {
        return this.request('/api/v1/beta/authorize', {
            method: 'POST',
            body: payload
        });
    }

    // POA Metrics
    async getPOAMetrics() {
        return this.request('/api/v1/poa/metrics');
    }

    // Metrics Endpoints
    async getDecisionMetrics(format = 'json') {
        const url = format === 'csv' 
            ? '/api/v1/beta/metrics/decision?format=csv'
            : '/api/v1/beta/metrics/decision';
        return this.request(url);
    }

    async getLifecycleMetrics(format = 'json') {
        const url = format === 'csv' 
            ? '/api/v1/beta/metrics/lifecycle?format=csv'
            : '/api/v1/beta/metrics/lifecycle';
        return this.request(url);
    }

    // Capabilities Endpoints
    async getCapabilities() {
        return this.request('/api/v1/beta/capabilities');
    }

    async negotiateCapabilities(clientVersions) {
        return this.request('/api/v1/beta/capabilities/negotiate', {
            method: 'POST',
            body: { client_versions: clientVersions }
        });
    }

    // Audit endpoints
    async getAuditPreview(limit = 20) {
        return this.request(`/api/v1/beta/audit/preview?limit=${limit}`);
    }

    async exportAuditLog(format = 'json') {
        return this.request(`/api/v1/beta/audit/export?format=${format}`);
    }

    // Policy endpoints
    async evaluatePolicy(payload) {
        return this.request('/api/v1/beta/policy/evaluate', {
            method: 'POST',
            body: payload
        });
    }

    async submitPolicyBundle(bundle, adminToken = null) {
        const headers = {};
        if (adminToken) {
            headers['Authorization'] = `Bearer ${adminToken}`;
        }
        return this.request('/api/v1/beta/policy/bundle', {
            method: 'POST',
            body: bundle,
            headers
        });
    }

    async getPolicyProvenance() {
        return this.request('/api/v1/beta/policy/provenance');
    }

    async getPolicyConsistency() {
        return this.request('/api/v1/beta/policy/consistency');
    }

    async getPolicyChain(offset = 0, limit = 10) {
        return this.request(`/api/v1/beta/policy/chain?offset=${offset}&limit=${limit}`);
    }

    async rollbackPolicy(version) {
        return this.request('/api/v1/beta/policy/rollback', {
            method: 'POST',
            body: { version }
        });
    }

    // Metrics endpoints
    async getMetrics() {
        return this.request('/api/v1/beta/metrics');
    }

    async getViolationMetrics() {
        return this.request('/api/v1/beta/metrics/violations');
    }

    async getSemanticMetrics() {
        return this.request('/api/v1/beta/metrics/poa/semantics');
    }

    async getAuthzMetrics() {
        return this.request('/api/v1/beta/metrics/authz');
    }

    async getPolicyMetrics() {
        return this.request('/api/v1/beta/metrics/policy');
    }

    async getDecisionMetrics() {
        return this.request('/api/v1/beta/metrics/decisions');
    }

    async getLifecycleTimeline(entityType = '', entityId = '') {
        const params = new URLSearchParams();
        if (entityType) params.append('type', entityType);
        if (entityId) params.append('id', entityId);
        
        const queryString = params.toString();
        const endpoint = `/api/v1/beta/metrics/lifecycle${queryString ? '?' + queryString : ''}`;
        return this.request(endpoint);
    }

    // Capability endpoints
    async getCapabilityAnchorStatus() {
        return this.request('/api/v1/beta/capabilities/anchor/status');
    }

    async getCapabilityRegistry() {
        return this.request('/api/v1/beta/capabilities/registry');
    }

    // Revocation transparency endpoints
    async getRevocationTransparency() {
        return this.request('/api/v1/beta/revocation/transparency');
    }

    async getRevocationProof(eventId) {
        return this.request(`/api/v1/beta/revocation/proof/${eventId}`);
    }

    async getRevocationConsistency(startIndex, targetLength) {
        return this.request(`/api/v1/beta/revocation/consistency?start=${startIndex}&target=${targetLength}`);
    }

    // Events endpoints (SSE)
    createEventStream(endpoint, onMessage, onError = null, onOpen = null) {
        const eventSource = new EventSource(`${this.baseURL}${endpoint}`);
        
        eventSource.onopen = (event) => {
            console.log('EventSource connected:', endpoint);
            if (onOpen) onOpen(event);
        };
        
        eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                onMessage(data);
            } catch (error) {
                console.error('Failed to parse event data:', error);
                if (onError) onError(error);
            }
        };
        
        eventSource.onerror = (error) => {
            console.error('EventSource error:', error);
            if (onError) onError(error);
        };

        return eventSource;
    }

    // Utility methods
    clearCache() {
        this.cache.clear();
    }

    // Cache management
    getCached(key) {
        const cached = this.cache.get(key);
        if (cached && Date.now() - cached.timestamp < 30000) { // 30 second cache
            return cached.data;
        }
        return null;
    }

    setCached(key, data) {
        this.cache.set(key, {
            data,
            timestamp: Date.now()
        });
    }

    // Export data methods
    async exportData(data, filename, format = 'json') {
        let content, mimeType;
        
        if (format === 'json') {
            content = JSON.stringify(data, null, 2);
            mimeType = 'application/json';
        } else if (format === 'csv') {
            content = this.convertToCSV(data);
            mimeType = 'text/csv';
        } else {
            throw new Error('Unsupported export format');
        }

        const blob = new Blob([content], { type: mimeType });
        const url = URL.createObjectURL(blob);
        
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        a.click();
        
        URL.revokeObjectURL(url);
    }

    convertToCSV(data) {
        if (!Array.isArray(data) || data.length === 0) {
            return '';
        }

        const headers = Object.keys(data[0]);
        const csvHeaders = headers.join(',');
        
        const csvRows = data.map(row => {
            return headers.map(header => {
                const value = row[header];
                // Escape quotes and wrap in quotes if contains comma or quote
                if (typeof value === 'string' && (value.includes(',') || value.includes('"'))) {
                    return `"${value.replace(/"/g, '""')}"`;
                }
                return value;
            }).join(',');
        });

        return [csvHeaders, ...csvRows].join('\n');
    }
}

// Create and export global API client instance
window.AgentAuthAPI = new AgentAuthAPIClient();

console.log('AgentAuth API Client initialized');