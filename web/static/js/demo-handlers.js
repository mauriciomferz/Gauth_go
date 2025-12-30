// AgentAuth Demo Handlers - Connect webapp UI to real backend functionality
class AgentAuthDemoHandlers {
    constructor(apiClient) {
        this.api = apiClient;
        this.activeStreams = new Map();
        this.runningJobs = new Map();
        this.eventLog = [];
        this.auditLog = [];
        this.metricsUpdateInterval = null;
        this.setupEventListeners();
        this.initializeMetricsUpdates();
    }

    setupEventListeners() {
        document.addEventListener('click', (e) => {
            this.handleButtonClicks(e);
        });

        document.addEventListener('submit', (e) => {
            this.handleFormSubmits(e);
        });

        // Tab handling
        document.addEventListener('click', (e) => {
            if (e.target.closest('.tab-button')) {
                this.handleTabSwitch(e.target.closest('.tab-button'));
            }
        });
    }

    handleTabSwitch(tabButton) {
        // Remove active from all tabs
        document.querySelectorAll('.tab-button').forEach(btn => {
            btn.classList.remove('active');
            btn.setAttribute('aria-selected', 'false');
            btn.setAttribute('tabindex', '-1');
        });

        // Hide all tab content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.add('hidden');
        });

        // Activate clicked tab
        tabButton.classList.add('active');
        tabButton.setAttribute('aria-selected', 'true');
        tabButton.removeAttribute('tabindex');

        // Show corresponding content
        const tabId = tabButton.getAttribute('data-tab');
        const content = document.getElementById(tabId);
        if (content) {
            content.classList.remove('hidden');
            
            // Load tab-specific data
            this.loadTabData(tabId);
        }
    }

    async loadTabData(tabId) {
        switch (tabId) {
            case 'samples-demo':
                await this.loadSamples();
                await this.loadJobPanel();
                break;
            case 'token-demo':
                await this.loadTokenMetrics();
                break;
            case 'authz-demo':
                await this.loadAuthzMetrics();
                break;
            case 'policy-demo':
                await this.loadPolicyGovernance();
                break;
            case 'audit-demo':
                await this.loadAuditPreview();
                break;
            case 'event-demo':
                // Events are loaded on-demand
                break;
        }
    }

    async handleButtonClicks(e) {
        const action = e.target.closest('[data-action]')?.getAttribute('data-action');
        const exampleId = e.target.closest('[data-example-id]')?.getAttribute('data-example-id');

        if (!action && !exampleId) return;

        try {
            if (exampleId) {
                await this.runExample(exampleId);
                return;
            }

            switch (action) {
                // Token Management
                case 'create-token':
                    await this.createToken();
                    break;
                case 'validate-token':
                    await this.validateToken();
                    break;
                case 'revoke-token':
                    await this.revokeToken();
                    break;

                // Authorization
                case 'check-authorization':
                    await this.checkAuthorization();
                    break;

                // Events
                case 'publish-event':
                    await this.publishEvent();
                    break;
                case 'subscribe-events':
                    await this.subscribeEvents();
                    break;

                // Audit
                case 'view-audit-log':
                    await this.viewAuditLog();
                    break;
                case 'generate-report':
                    await this.generateReport();
                    break;

                // Policy
                case 'policy-evaluate':
                    await this.evaluatePolicy();
                    break;
                case 'policy-submit-bundle':
                    await this.submitPolicyBundle();
                    break;
                case 'policy-provenance':
                    await this.getPolicyProvenance();
                    break;
                case 'policy-consistency':
                    await this.getPolicyConsistency();
                    break;
                case 'policy-chain-page':
                    await this.getPolicyChain();
                    break;
                case 'policy-rollback':
                    await this.rollbackPolicy();
                    break;

                // Samples
                case 'run-all-samples':
                    await this.runAllSamples();
                    break;
                case 'run-all-basics':
                    await this.runAllBasics();
                    break;
                case 'run-advanced-suite':
                    await this.runAdvancedSuite();
                    break;

                // General
                case 'start-learning-path':
                    this.startLearningPath();
                    break;
                case 'quick-compliance-check':
                    await this.quickComplianceCheck();
                    break;

                default:
                    console.log('Unhandled action:', action);
            }
        } catch (error) {
            this.showError(`Action failed: ${error.message}`);
        }
    }

    async handleFormSubmits(e) {
        if (e.target.id === 'authz-eval-form') {
            e.preventDefault();
            await this.evaluateAuthzForm(e.target);
        }
    }

    // Token Management Implementation
    async createToken() {
        const outputDiv = document.getElementById('token-output');
        this.appendOutput(outputDiv, 'Creating demo token...', 'info');

        try {
            const tokenData = {
                subject: 'demo-user@example.com',
                scopes: ['read', 'write'],
                expires_in: 3600
            };

            const result = await this.api.createToken(tokenData);
            
            if (result.success && result.token) {
                this.appendOutput(outputDiv, `✓ Token created successfully`, 'success');
                this.appendOutput(outputDiv, `Token: ${result.token.substring(0, 50)}...`, 'info');
                
                // Store token for validation
                this.lastToken = result.token;
                
                // Auto-validate after creation
                setTimeout(() => this.validateToken(), 1000);
            } else {
                throw new Error(result.message || 'Token creation failed');
            }
        } catch (error) {
            this.appendOutput(outputDiv, `✗ Token creation failed: ${error.message}`, 'error');
        }
    }

    async validateToken() {
        const outputDiv = document.getElementById('token-output');
        
        if (!this.lastToken) {
            this.appendOutput(outputDiv, '⚠ No token to validate. Create a token first.', 'warning');
            return;
        }

        this.appendOutput(outputDiv, 'Validating token...', 'info');

        try {
            const result = await this.api.validateToken(this.lastToken);
            
            if (result.valid) {
                this.appendOutput(outputDiv, `✓ Token is VALID`, 'success');
                if (result.claims) {
                    this.appendOutput(outputDiv, `Subject: ${result.claims.sub || 'unknown'}`, 'info');
                    this.appendOutput(outputDiv, `Expires: ${new Date(result.claims.exp * 1000).toLocaleString()}`, 'info');
                }
            } else {
                this.appendOutput(outputDiv, `✗ Token is INVALID: ${result.reason || 'Unknown reason'}`, 'error');
            }
        } catch (error) {
            this.appendOutput(outputDiv, `✗ Token validation failed: ${error.message}`, 'error');
        }
    }

    async revokeToken() {
        const outputDiv = document.getElementById('token-output');
        
        if (!this.lastToken) {
            this.appendOutput(outputDiv, '⚠ No token to revoke. Create a token first.', 'warning');
            return;
        }

        this.appendOutput(outputDiv, 'Revoking token...', 'info');

        try {
            const result = await this.api.revokeToken(this.lastToken);
            
            if (result.success) {
                this.appendOutput(outputDiv, `✓ Token revoked successfully`, 'success');
                this.lastToken = null;
            } else {
                throw new Error(result.message || 'Token revocation failed');
            }
        } catch (error) {
            this.appendOutput(outputDiv, `✗ Token revocation failed: ${error.message}`, 'error');
        }
    }

    // Authorization Implementation
    async checkAuthorization() {
        const outputDiv = document.getElementById('authz-output');
        const actionSelect = document.getElementById('resource-action');
        const action = actionSelect ? actionSelect.value : 'read';

        this.appendOutput(outputDiv, `Checking authorization for action: ${action}`, 'info');

        try {
            const authzData = {
                subject: 'alice@example.com',
                action: action,
                resource: 'document:financial-report',
                context: {
                    department: 'finance',
                    classification: 'internal'
                }
            };

            const result = await this.api.evaluateAuthorization(authzData);
            
            if (result.decision) {
                const decision = result.decision.allow ? 'ALLOW' : 'DENY';
                const className = result.decision.allow ? 'success' : 'error';
                
                this.appendOutput(outputDiv, `✓ Authorization Decision: ${decision}`, className);
                if (result.decision.reason) {
                    this.appendOutput(outputDiv, `Reason: ${result.decision.reason}`, 'info');
                }
                if (result.decision.policies_evaluated) {
                    this.appendOutput(outputDiv, `Policies evaluated: ${result.decision.policies_evaluated}`, 'info');
                }
            } else {
                throw new Error('No decision in response');
            }
        } catch (error) {
            this.appendOutput(outputDiv, `✗ Authorization check failed: ${error.message}`, 'error');
        }
    }

    async evaluateAuthzForm(form) {
        const formData = new FormData(form);
        const data = Object.fromEntries(formData.entries());
        
        const payload = {
            subject: data.subject,
            resource: data.resource,
            action: data.action,
            context: {
                department: data.department,
                classification: data.classification,
                roles: data.roles ? data.roles.split(',').map(r => r.trim()) : []
            }
        };

        try {
            const result = await this.api.evaluateAuthorization(payload);
            const resultDiv = document.getElementById('authz-eval-result');
            
            if (result.decision) {
                const decision = result.decision.allow ? 'ALLOW' : 'DENY';
                const className = result.decision.allow ? 'text-green-600' : 'text-red-600';
                
                resultDiv.innerHTML = `
                    <div class="p-2 border rounded bg-gray-50">
                        <div class="font-semibold ${className}">Decision: ${decision}</div>
                        <div class="text-gray-600 text-xs mt-1">Reason: ${result.decision.reason || 'No reason provided'}</div>
                    </div>
                `;
            }
        } catch (error) {
            const resultDiv = document.getElementById('authz-eval-result');
            resultDiv.innerHTML = `<div class="text-red-600 text-xs">Error: ${error.message}</div>`;
        }
    }

    // Samples Implementation
    async loadSamples() {
        try {
            const result = await this.api.getExamplesCatalog();
            const samplesList = document.getElementById('samples-list');
            
            if (!samplesList) return;

            if (result.success && result.examples) {
                this.renderSamplesList(result.examples, samplesList);
            } else {
                samplesList.innerHTML = '<div class="text-center text-gray-400 py-8">Failed to load examples</div>';
            }
        } catch (error) {
            console.error('Failed to load samples:', error);
            const samplesList = document.getElementById('samples-list');
            if (samplesList) {
                samplesList.innerHTML = '<div class="text-center text-red-400 py-8">Error loading examples</div>';
            }
        }
    }

    renderSamplesList(examples, container) {
        const html = examples.map(example => `
            <div class="flex items-center justify-between p-3 border-b border-gray-200 hover:bg-gray-50 cursor-pointer" 
                 data-example-id="${example.id}">
                <div class="flex-1">
                    <div class="font-semibold text-sm">${example.title}</div>
                    <div class="text-xs text-gray-600">${example.description}</div>
                    <div class="flex items-center gap-2 mt-1">
                        <span class="px-2 py-1 rounded text-xs font-medium bg-${this.getGroupColor(example.group)}-100 text-${this.getGroupColor(example.group)}-800">
                            ${example.group}
                        </span>
                        <span class="text-xs text-gray-500">${example.estimated_seconds}s</span>
                    </div>
                </div>
                <button class="bg-blue-600 hover:bg-blue-700 text-white text-xs font-semibold px-3 py-1 rounded"
                        data-example-id="${example.id}">
                    Run
                </button>
            </div>
        `).join('');
        
        container.innerHTML = html;
    }

    getGroupColor(group) {
        const colors = {
            'basics': 'green',
            'advanced': 'purple',
            'negative': 'red'
        };
        return colors[group] || 'gray';
    }

    async runExample(exampleId) {
        const outputDiv = document.getElementById('samples-output');
        this.appendOutput(outputDiv, `Starting example: ${exampleId}`, 'info');

        try {
            const result = await this.api.runExample(exampleId);
            
            if (result.job_id && (result.success || result.state === 'running')) {
                this.appendOutput(outputDiv, `✓ Example queued with job ID: ${result.job_id}`, 'success');
                this.appendOutput(outputDiv, `Status: ${result.state || 'running'}`, 'info');
                
                // Track the job
                this.runningJobs.set(result.job_id, {
                    exampleId,
                    startTime: Date.now()
                });

                // Start polling job status
                this.pollJobStatus(result.job_id);
            } else {
                throw new Error(result.message || 'Failed to start example');
            }
        } catch (error) {
            this.appendOutput(outputDiv, `✗ Failed to run example: ${error.message}`, 'error');
        }
    }

    async pollJobStatus(jobId) {
        const maxAttempts = 30;
        let attempts = 0;

        const poll = async () => {
            try {
                const status = await this.api.getJobStatus(jobId);
                attempts++;

                const outputDiv = document.getElementById('samples-output');
                const jobInfo = this.runningJobs.get(jobId);

                if (status.state === 'completed') {
                    this.runningJobs.delete(jobId);
                    this.appendOutput(outputDiv, `✓ Example completed: ${status.message || 'Success'}`, 'success');
                    
                    // Try to get logs
                    try {
                        const logs = await this.api.getJobLogs(jobId);
                        if (logs && logs.output) {
                            this.appendOutput(outputDiv, '--- Job Output ---', 'info');
                            this.appendOutput(outputDiv, logs.output, 'output');
                        }
                    } catch (logError) {
                        console.warn('Could not fetch job logs:', logError);
                    }

                    this.updateJobPanel();
                    return;
                } else if (status.state === 'failed') {
                    this.runningJobs.delete(jobId);
                    this.appendOutput(outputDiv, `✗ Example failed: ${status.message || 'Unknown error'}`, 'error');
                    this.updateJobPanel();
                    return;
                } else if (status.state === 'running' || status.state === 'queued') {
                    if (attempts < maxAttempts) {
                        setTimeout(poll, 1000);
                    } else {
                        this.appendOutput(outputDiv, '⚠ Example taking longer than expected...', 'warning');
                    }
                } else {
                    this.appendOutput(outputDiv, `Job status: ${status.state}`, 'info');
                    if (attempts < maxAttempts) {
                        setTimeout(poll, 1000);
                    }
                }
            } catch (error) {
                console.error('Failed to poll job status:', error);
                this.runningJobs.delete(jobId);
                this.updateJobPanel();
            }
        };

        poll();
    }

    async loadJobPanel() {
        try {
            const jobs = await this.api.getActiveJobs();
            this.updateJobPanelWithData(jobs);
        } catch (error) {
            console.error('Failed to load job panel:', error);
        }
    }

    async updateJobPanel() {
        await this.loadJobPanel();
    }

    updateJobPanelWithData(jobsData) {
        const tbody = document.getElementById('job-table-body');
        if (!tbody) return;

        if (!jobsData || !jobsData.jobs || jobsData.jobs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="text-center text-gray-400 py-4">No active jobs</td></tr>';
            return;
        }

        const html = jobsData.jobs.map(job => {
            const startTime = new Date(job.started_at);
            const duration = job.completed_at ? 
                Math.round((new Date(job.completed_at) - startTime) / 1000) :
                Math.round((Date.now() - startTime) / 1000);

            const stateClass = {
                'running': 'text-yellow-600',
                'completed': 'text-green-600',
                'failed': 'text-red-600',
                'queued': 'text-blue-600'
            }[job.state] || 'text-gray-600';

            return `
                <tr>
                    <td class="px-3 py-2 font-mono text-xs">${job.id}</td>
                    <td class="px-3 py-2 text-xs">${job.example_id || 'Unknown'}</td>
                    <td class="px-3 py-2">
                        <span class="px-2 py-1 rounded text-xs font-semibold ${stateClass}">
                            ${job.state}
                        </span>
                    </td>
                    <td class="px-3 py-2 text-xs">${startTime.toLocaleTimeString()}</td>
                    <td class="px-3 py-2 text-xs">${duration}s</td>
                    <td class="px-3 py-2">
                        ${job.state === 'running' || job.state === 'queued' ? 
                            `<button class="bg-red-600 hover:bg-red-700 text-white text-xs px-2 py-1 rounded" 
                                     onclick="gAuthDemo.cancelJob('${job.id}')">Cancel</button>` :
                            '-'
                        }
                    </td>
                </tr>
            `;
        }).join('');

        tbody.innerHTML = html;
    }

    async cancelJob(jobId) {
        try {
            await this.api.cancelJob(jobId);
            this.updateJobPanel();
            
            const outputDiv = document.getElementById('samples-output');
            this.appendOutput(outputDiv, `✓ Job ${jobId} cancelled`, 'info');
        } catch (error) {
            console.error('Failed to cancel job:', error);
        }
    }

    // Events Implementation
    async publishEvent() {
        const outputDiv = document.getElementById('event-output');
        this.appendOutput(outputDiv, 'Publishing demo event...', 'info');

        // Since there's no direct event publish endpoint in the API spec,
        // we'll simulate by triggering an action that generates events
        try {
            await this.api.createToken({
                subject: 'event-demo-user',
                scopes: ['read'],
                expires_in: 600
            });
            
            this.appendOutput(outputDiv, '✓ Event published (via token creation)', 'success');
        } catch (error) {
            this.appendOutput(outputDiv, `✗ Failed to publish event: ${error.message}`, 'error');
        }
    }

    async subscribeEvents() {
        const outputDiv = document.getElementById('event-output');
        this.appendOutput(outputDiv, 'Subscribing to event stream...', 'info');

        try {
            // Try to create an event stream if the endpoint exists
            const eventStream = this.api.createEventStream(
                '/api/v1/beta/events/stream',
                (data) => {
                    this.eventLog.push(data);
                    this.appendOutput(outputDiv, `📡 Event: ${JSON.stringify(data)}`, 'event');
                },
                (error) => {
                    this.appendOutput(outputDiv, `⚠ Event stream error: ${error.message}`, 'warning');
                },
                () => {
                    this.appendOutput(outputDiv, '✓ Connected to event stream', 'success');
                }
            );

            this.activeStreams.set('events', eventStream);
        } catch (error) {
            this.appendOutput(outputDiv, `✗ Failed to subscribe to events: ${error.message}`, 'error');
        }
    }

    // Utility Methods
    appendOutput(outputDiv, message, type = 'info') {
        if (!outputDiv) return;

        const timestamp = new Date().toLocaleTimeString();
        const colors = {
            info: 'text-blue-400',
            success: 'text-green-400',
            error: 'text-red-400',
            warning: 'text-yellow-400',
            event: 'text-purple-400',
            output: 'text-gray-300'
        };

        const color = colors[type] || colors.info;
        const line = document.createElement('div');
        line.className = color;
        line.innerHTML = `[${timestamp}] ${message}`;
        
        outputDiv.appendChild(line);
        outputDiv.scrollTop = outputDiv.scrollHeight;
    }

    showError(message) {
        console.error(message);
        
        // Create toast notification
        const toast = document.createElement('div');
        toast.className = 'fixed top-4 right-4 bg-red-600 text-white px-6 py-3 rounded-lg shadow-lg z-50 transition-all duration-300';
        toast.innerHTML = `
            <div class="flex items-center space-x-2">
                <i class="fas fa-exclamation-triangle"></i>
                <span>${message}</span>
            </div>
        `;
        
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.style.opacity = '0';
            toast.style.transform = 'translateX(100%)';
            setTimeout(() => toast.remove(), 300);
        }, 5000);
    }

    showSuccess(message) {
        const toast = document.createElement('div');
        toast.className = 'fixed top-4 right-4 bg-green-600 text-white px-6 py-3 rounded-lg shadow-lg z-50 transition-all duration-300';
        toast.innerHTML = `
            <div class="flex items-center space-x-2">
                <i class="fas fa-check-circle"></i>
                <span>${message}</span>
            </div>
        `;
        
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.style.opacity = '0';
            toast.style.transform = 'translateX(100%)';
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }

    // Initialize metrics updates
    initializeMetricsUpdates() {
        this.metricsUpdateInterval = setInterval(() => {
            this.updateMetrics();
        }, 5000); // Update every 5 seconds
    }

    async updateMetrics() {
        // Update various metrics panels if visible
        try {
            await this.updatePOAMetrics();
            await this.updateViolationMetrics();
            await this.updateSemanticMetrics();
        } catch (error) {
            console.warn('Metrics update failed:', error);
        }
    }

    async updatePOAMetrics() {
        try {
            const metrics = await this.api.getPOAMetrics();
            const poaPanel = document.getElementById('poa-metrics');
            
            if (poaPanel && metrics.success) {
                poaPanel.classList.remove('hidden');
                // Update POA metrics display
                const reqEl = document.getElementById('m-poa-req');
                if (reqEl) reqEl.textContent = metrics.metrics.total_requests || 0;
            }
        } catch (error) {
            // POA metrics not available
        }
    }

    async updateViolationMetrics() {
        try {
            const metrics = await this.api.getViolationMetrics();
            // Update violation metrics if panel exists
        } catch (error) {
            // Violation metrics not available
        }
    }

    async updateSemanticMetrics() {
        try {
            const metrics = await this.api.getSemanticMetrics();
            // Update semantic metrics if panel exists
        } catch (error) {
            // Semantic metrics not available
        }
    }

    // Cleanup
    destroy() {
        if (this.metricsUpdateInterval) {
            clearInterval(this.metricsUpdateInterval);
        }

        this.activeStreams.forEach(stream => {
            stream.close();
        });
        this.activeStreams.clear();
    }

    // Placeholder methods for features not yet implemented
    async loadTokenMetrics() {
        console.log('Loading token metrics...');
    }

    async loadAuthzMetrics() {
        console.log('Loading authz metrics...');
    }

    async loadPolicyGovernance() {
        console.log('Loading policy governance...');
    }

    async loadAuditPreview() {
        console.log('Loading audit preview...');
    }

    async viewAuditLog() {
        const outputDiv = document.getElementById('audit-output');
        this.appendOutput(outputDiv, 'Loading audit log...', 'info');
        // Implementation would go here
    }

    async generateReport() {
        const outputDiv = document.getElementById('audit-output');
        this.appendOutput(outputDiv, 'Generating compliance report...', 'info');
        // Implementation would go here
    }

    async evaluatePolicy() {
        const outputDiv = document.getElementById('policy-output');
        this.appendOutput(outputDiv, 'Evaluating policy...', 'info');
        // Implementation would go here
    }

    async submitPolicyBundle() {
        const outputDiv = document.getElementById('policy-output');
        this.appendOutput(outputDiv, 'Submitting policy bundle...', 'info');
        // Implementation would go here
    }

    async getPolicyProvenance() {
        const outputDiv = document.getElementById('policy-output');
        this.appendOutput(outputDiv, 'Getting policy provenance...', 'info');
        // Implementation would go here
    }

    async getPolicyConsistency() {
        const outputDiv = document.getElementById('policy-output');
        this.appendOutput(outputDiv, 'Checking policy consistency...', 'info');
        // Implementation would go here
    }

    async getPolicyChain() {
        const outputDiv = document.getElementById('policy-output');
        this.appendOutput(outputDiv, 'Loading policy chain...', 'info');
        // Implementation would go here
    }

    async rollbackPolicy() {
        const outputDiv = document.getElementById('policy-output');
        this.appendOutput(outputDiv, 'Rolling back policy...', 'info');
        // Implementation would go here
    }

    async runAllSamples() {
        const outputDiv = document.getElementById('samples-output');
        this.appendOutput(outputDiv, 'Running all samples...', 'info');
        // Implementation would iterate through all samples
    }

    async runAllBasics() {
        const outputDiv = document.getElementById('samples-output');
        this.appendOutput(outputDiv, 'Running basic samples...', 'info');
        // Implementation would run basic samples
    }

    async runAdvancedSuite() {
        const outputDiv = document.getElementById('samples-output');
        this.appendOutput(outputDiv, 'Running advanced samples...', 'info');
        // Implementation would run advanced samples
    }

    startLearningPath() {
        const target = document.getElementById('learning-path');
        if (target) {
            target.scrollIntoView({ behavior: 'smooth', block: 'start' });
            this.showSuccess('Starting your AgentAuth learning journey!');
        }
    }

    async quickComplianceCheck() {
        const target = document.getElementById('compliance');
        if (target) {
            target.scrollIntoView({ behavior: 'smooth', block: 'start' });
            this.showSuccess('Running RFC-0150 compliance check...');
            // Implementation would check compliance
        }
    }
}

// Initialize demo handlers when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    // Wait for API client to be available
    if (window.AgentAuthAPI) {
        window.gAuthDemo = new AgentAuthDemoHandlers(window.AgentAuthAPI);
        console.log('AgentAuth Demo Handlers initialized');
    } else {
        console.error('AgentAuth API Client not available');
    }
});

console.log('AgentAuth Demo Handlers loaded');