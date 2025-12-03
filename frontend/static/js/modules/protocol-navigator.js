// protocol-navigator.js - GAuth Protocol Flow Navigator
// Peer-level menu system showing current position in GAuth workflow
// Inspired by Entra Agent ID style navigation
//
// Updated: November 16, 2025 - Phase 2A Integration & Token Security
// Phase 2A Backend Integration:
//   - PVP Identity Verification (/api/v1/beta/pvp/verify)
//   - Registry Entity/Signatory Verification (/api/v1/beta/registry/*)
//   - Power of Attorney (PoA) Management (/api/v1/beta/poa/*)
//   - RFC-0111 Token Authorization (/api/v1/rfc0111/authorize)
//   - Token Security Validation (/api/v1/rfc0111/token/validate)
//
// Token Security Model:
//   - Full RFC-0111 token creation requires 8-step subscription flow
//   - Mock tokens are correctly rejected with 401 (security working)
//   - Valid tokens require completed subscription + PoA credential
//
// Environment: Requires GAUTH_RFC0111_ENABLED=1

/**
 * GAuth Protocol Flow Steps
 * Based on RFC-0111 and RFC-0115 authorization flow
 */
const PROTOCOL_STEPS = {
    SUBSCRIPTION: {
        id: 'subscription',
        name: 'Subscription',
        description: 'Authorization server registration and client setup',
        icon: '📝',
        substeps: [
            { id: 'register', name: 'Register Client', status: 'pending' },
            { id: 'configure', name: 'Configure Scopes', status: 'pending' },
            { id: 'credentials', name: 'Obtain Credentials', status: 'pending' }
        ],
        apis: ['/api/v1/subscribe', '/api/v1/client/register']
    },
    MATCHING: {
        id: 'matching',
        name: 'Matching',
        description: 'PoA Definition validation and capability matching',
        icon: '🔍',
        substeps: [
            { id: 'validate_poa', name: 'Validate PoA Definition', status: 'pending' },
            { id: 'authorization_chain', name: 'Authorization Chain Validation', status: 'pending' },
            { id: 'commercial_register', name: 'Commercial Register Check', status: 'pending' },
            { id: 'check_capabilities', name: 'Check AI Capabilities', status: 'pending' },
            { id: 'verify_jurisdiction', name: 'Verify Jurisdiction', status: 'pending' },
            { id: 'formal_requirements', name: 'Formal Requirements Check', status: 'pending' },
            { id: 'match_policies', name: 'Match Policies', status: 'pending' }
        ],
        apis: [
            '/api/v1/beta/poa',
            '/api/v1/beta/poa/:id/validate',
            '/api/v1/beta/registry/verify-entity',
            '/api/v1/beta/registry/verify-signatory',
            '/api/v1/ai/capabilities',
            '/api/v1/authchain/validate'
        ]
    },
    SUBSET_REQUEST: {
        id: 'subset_request',
        name: 'Subset/Request',
        description: 'Authorization request with scope subset selection',
        icon: '🎯',
        substeps: [
            { id: 'subscription_flow', name: 'RFC-0111 Subscription (Steps I-VIII)', status: 'pending' },
            { id: 'create_request', name: 'Create Auth Request', status: 'pending' },
            { id: 'request_compliance', name: 'Request Compliance Validation', status: 'pending' },
            { id: 'select_scope', name: 'Select Scope Subset', status: 'pending' },
            { id: 'pip_query', name: 'PIP Policy Info Query', status: 'pending' },
            { id: 'pdp_decision', name: 'PDP Decision', status: 'pending' },
            { id: 'generate_extended_token', name: 'Generate Extended Token', status: 'pending' },
            { id: 'grant_compliance', name: 'Grant Compliance Validation', status: 'pending' }
        ],
        apis: [
            '/api/v1/rfc0111/subscriptions',
            '/api/v1/rfc0111/authorize',
            '/api/v1/authorize',
            '/api/v1/token',
            '/api/v1/compliance/validate',
            '/api/v1/beta/poa'
        ],
        notes: [
            'RFC-0111 requires 8-step subscription before token generation',
            'Steps I-VIII: Identity proofs, authentication, authorization checks',
            'Token generation uses completed subscription + PoA credential'
        ]
    },
    ENFORCEMENT: {
        id: 'enforcement',
        name: 'Enforcement',
        description: 'PEP enforcement (supply-side and demand-side)',
        icon: '🛡️',
        substeps: [
            { id: 'supply_pep', name: 'Supply-Side PEP', status: 'pending' },
            { id: 'demand_pep', name: 'Demand-Side PEP', status: 'pending' },
            { id: 'disclosure', name: 'Disclosure Requirements', status: 'pending' },
            { id: 'audit', name: 'Audit Logging', status: 'pending' }
        ],
        apis: ['/api/v1/enforce/supply', '/api/v1/enforce/demand']
    },
    VERIFICATION: {
        id: 'verification',
        name: 'Verification',
        description: 'Token security validation and PVP identity verification',
        icon: '✓',
        substeps: [
            { id: 'token_security', name: 'Token Security Validation', status: 'pending' },
            { id: 'validate_extended_token', name: 'Validate Extended Token', status: 'pending' },
            { id: 'verify_signature', name: 'Verify JWE Signature', status: 'pending' },
            { id: 'check_revocation', name: 'Check Revocation Status', status: 'pending' },
            { id: 'pvp_identity', name: 'PVP Identity Verification', status: 'pending' },
            { id: 'authorization_chain_verify', name: 'Authorization Chain Verify', status: 'pending' },
            { id: 'formal_requirements_verify', name: 'Formal Requirements Verify', status: 'pending' }
        ],
        apis: [
            '/api/v1/rfc0111/token/validate',
            '/api/v1/validate',
            '/api/v1/verify',
            '/api/v1/beta/pvp/verify'
        ],
        notes: [
            'Token validation properly rejects invalid/mock tokens with 401',
            'Valid tokens require completed RFC-0111 subscription flow (8 steps)',
            'Security validation confirms proper authentication enforcement'
        ]
    },
    AUDIT: {
        id: 'audit',
        name: 'Audit & Compliance',
        description: 'Audit trail and compliance reporting',
        icon: '📊',
        substeps: [
            { id: 'log_event', name: 'Log Audit Event', status: 'pending' },
            { id: 'compliance_check', name: 'Compliance Check', status: 'pending' },
            { id: 'report_generation', name: 'Generate Reports', status: 'pending' }
        ],
        apis: ['/api/v1/audit', '/api/v1/compliance/report']
    }
};

/**
 * ProtocolNavigator - Main navigation controller
 */
class ProtocolNavigator {
    constructor() {
        this.currentStep = null;
        this.currentSubstep = null;
        this.flowHistory = [];
        this.stepStates = this.initializeStepStates();
        this.observers = [];
    }

    /**
     * Initialize all steps with pending status
     */
    initializeStepStates() {
        const states = {};
        Object.values(PROTOCOL_STEPS).forEach(step => {
            states[step.id] = {
                status: 'pending', // pending, in-progress, completed, error
                completedSubsteps: [],
                startTime: null,
                endTime: null,
                error: null
            };
        });
        return states;
    }

    /**
     * Navigate to a specific step
     */
    navigateToStep(stepId, substepId = null) {
        console.log('[ProtocolNavigator] navigateToStep:', stepId, substepId);
        const step = Object.values(PROTOCOL_STEPS).find(s => s.id === stepId);
        if (!step) {
            console.error(`Invalid step: ${stepId}`);
            return false;
        }

        // Record history
        if (this.currentStep) {
            this.flowHistory.push({
                step: this.currentStep,
                substep: this.currentSubstep,
                timestamp: new Date()
            });
        }

        this.currentStep = stepId;
        this.currentSubstep = substepId;

        // Update step state
        if (this.stepStates[stepId].status === 'pending') {
            this.stepStates[stepId].status = 'in-progress';
            this.stepStates[stepId].startTime = new Date();
        }

        this.notifyObservers('navigate', { step: stepId, substep: substepId });
        return true;
    }

    /**
     * Mark current substep as completed
     */
    completeSubstep(stepId, substepId) {
        if (!this.stepStates[stepId]) return;

        if (!this.stepStates[stepId].completedSubsteps.includes(substepId)) {
            this.stepStates[stepId].completedSubsteps.push(substepId);
        }

        // Check if all substeps completed
        const step = Object.values(PROTOCOL_STEPS).find(s => s.id === stepId);
        if (step && this.stepStates[stepId].completedSubsteps.length === step.substeps.length) {
            this.completeStep(stepId);
        }

        this.notifyObservers('substep_complete', { step: stepId, substep: substepId });
    }

    /**
     * Mark entire step as completed
     */
    completeStep(stepId) {
        if (!this.stepStates[stepId]) return;

        this.stepStates[stepId].status = 'completed';
        this.stepStates[stepId].endTime = new Date();
        this.notifyObservers('step_complete', { step: stepId });
    }

    /**
     * Mark step as error
     */
    errorStep(stepId, error) {
        if (!this.stepStates[stepId]) return;

        this.stepStates[stepId].status = 'error';
        this.stepStates[stepId].error = error;
        this.notifyObservers('step_error', { step: stepId, error });
    }

    /**
     * Get current progress percentage
     */
    getProgress() {
        const totalSteps = Object.keys(PROTOCOL_STEPS).length;
        const completedSteps = Object.values(this.stepStates)
            .filter(s => s.status === 'completed').length;
        return Math.round((completedSteps / totalSteps) * 100);
    }

    /**
     * Get step state
     */
    getStepState(stepId) {
        return this.stepStates[stepId];
    }

    /**
     * Reset flow to beginning
     */
    reset() {
        this.currentStep = null;
        this.currentSubstep = null;
        this.flowHistory = [];
        this.stepStates = this.initializeStepStates();
        this.notifyObservers('reset', {});
    }

    /**
     * Observer pattern for UI updates
     */
    subscribe(callback) {
        this.observers.push(callback);
    }

    notifyObservers(event, data) {
        console.log('[ProtocolNavigator] notifyObservers:', event, 'to', this.observers.length, 'observers');
        this.observers.forEach(callback => callback(event, data));
    }

    /**
     * Get navigation breadcrumb
     */
    getBreadcrumb() {
        const breadcrumb = [];
        
        if (this.currentStep) {
            const step = Object.values(PROTOCOL_STEPS).find(s => s.id === this.currentStep);
            breadcrumb.push({
                type: 'step',
                id: this.currentStep,
                name: step.name,
                icon: step.icon
            });

            if (this.currentSubstep) {
                const substep = step.substeps.find(s => s.id === this.currentSubstep);
                if (substep) {
                    breadcrumb.push({
                        type: 'substep',
                        id: this.currentSubstep,
                        name: substep.name
                    });
                }
            }
        }

        return breadcrumb;
    }
}

/**
 * ProtocolNavigatorUI - Renders navigation interface
 */
class ProtocolNavigatorUI {
    constructor(navigator, containerId = 'protocol-navigator') {
        this.navigator = navigator;
        this.container = document.getElementById(containerId);
        
        console.log('[ProtocolNavigatorUI] Constructor - container ID:', containerId, 'found:', !!this.container);
        
        if (!this.container) {
            console.error(`❌ Container #${containerId} not found in DOM!`);
            console.log('Available elements:', Array.from(document.querySelectorAll('[id]')).map(el => el.id));
            return;
        }

        console.log('[ProtocolNavigatorUI] Subscribing to navigator events and rendering...');
        this.navigator.subscribe((event, data) => this.handleNavigatorEvent(event, data));
        this.render();
        console.log('[ProtocolNavigatorUI] Render complete, container innerHTML length:', this.container.innerHTML.length);
    }

    /**
     * Handle navigator events
     */
    handleNavigatorEvent(event, data) {
        console.log('[ProtocolNavigatorUI] handleNavigatorEvent:', event, data);
        switch (event) {
            case 'navigate':
                this.highlightCurrentStep();
                this.updateBreadcrumb();
                break;
            case 'step_complete':
                this.markStepComplete(data.step);
                break;
            case 'substep_complete':
                this.markSubstepComplete(data.step, data.substep);
                break;
            case 'step_error':
                this.markStepError(data.step, data.error);
                break;
            case 'reset':
                this.render();
                break;
        }
        this.updateProgressBar();
    }

    /**
     * Render complete navigation UI
     */
    render() {
        const html = `
            <div class="protocol-navigator-wrapper">
                <div class="navigator-header">
                    <h2>🚀 GAuth Protocol Flow</h2>
                    <div class="navigator-progress">
                        <div class="progress-bar">
                            <div class="progress-fill" style="width: 0%"></div>
                        </div>
                        <span class="progress-text">0% Complete</span>
                    </div>
                </div>

                <div class="navigator-breadcrumb" id="nav-breadcrumb">
                    <span class="breadcrumb-item">Start</span>
                </div>

                <div class="navigator-steps" id="nav-steps">
                    ${this.renderSteps()}
                </div>

                <div class="navigator-actions">
                    <button class="btn-reset" data-action="reset">
                        🔄 Reset Flow
                    </button>
                    <button class="btn-export" data-action="export">
                        📥 Export History
                    </button>
                </div>
            </div>
        `;

        this.container.innerHTML = html;
        this.attachEventListeners();
    }

    /**
     * Render all protocol steps
     */
    renderSteps() {
        return Object.values(PROTOCOL_STEPS).map(step => {
            const state = this.navigator.getStepState(step.id);
            const statusClass = `step-${state.status}`;
            
            return `
                <div class="protocol-step ${statusClass}" 
                     data-step="${step.id}">
                    <div class="step-header">
                        <span class="step-icon">${step.icon}</span>
                        <div class="step-info">
                            <h3 class="step-name">${step.name}</h3>
                            <p class="step-description">${step.description}</p>
                        </div>
                        <span class="step-status ${state.status}">
                            ${this.getStatusIcon(state.status)}
                        </span>
                    </div>
                    <div class="step-substeps" id="substeps-${step.id}" style="display: none;">
                        ${this.renderSubsteps(step, state)}
                    </div>
                    <div class="step-apis">
                        <strong>APIs:</strong> ${step.apis.join(', ')}
                    </div>
                    ${step.notes ? `
                    <div class="step-notes">
                        <strong>Notes:</strong>
                        <ul>
                            ${step.notes.map(note => `<li>${note}</li>`).join('')}
                        </ul>
                    </div>
                    ` : ''}
                </div>
            `;
        }).join('');
    }

    /**
     * Render substeps for a step
     */
    renderSubsteps(step, state) {
        return step.substeps.map(substep => {
            const isCompleted = state.completedSubsteps.includes(substep.id);
            const isCurrent = this.navigator.currentSubstep === substep.id;
            const statusClass = isCompleted ? 'completed' : (isCurrent ? 'current' : 'pending');

            return `
                <div class="substep ${statusClass}" 
                     data-substep="${substep.id}"
                     data-step-id="${step.id}">
                    <span class="substep-marker">${isCompleted ? '✓' : '○'}</span>
                    <span class="substep-name">${substep.name}</span>
                    ${isCurrent ? '<span class="substep-current">← Current</span>' : ''}
                </div>
            `;
        }).join('');
    }

    /**
     * Get status icon
     */
    getStatusIcon(status) {
        const icons = {
            'pending': '○',
            'in-progress': '⟳',
            'completed': '✓',
            'error': '✗'
        };
        return icons[status] || '○';
    }

    /**
     * Toggle step expansion
     */
    toggleStep(stepId) {
        const substepsEl = document.getElementById(`substeps-${stepId}`);
        if (substepsEl) {
            const isHidden = substepsEl.style.display === 'none';
            substepsEl.style.display = isHidden ? 'block' : 'none';
        }
    }

    /**
     * Select and navigate to substep
     */
    selectSubstep(stepId, substepId) {
        this.navigator.navigateToStep(stepId, substepId);
        this.render(); // Update UI after navigation
    }

    /**
     * Highlight current step
     */
    highlightCurrentStep() {
        // Remove previous highlights
        document.querySelectorAll('.protocol-step').forEach(el => {
            el.classList.remove('step-current');
        });

        // Add current highlight
        if (this.navigator.currentStep) {
            const stepEl = document.querySelector(`[data-step="${this.navigator.currentStep}"]`);
            if (stepEl) {
                stepEl.classList.add('step-current');
                stepEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
            }
        }
    }

    /**
     * Update breadcrumb navigation
     */
    updateBreadcrumb() {
        const breadcrumbEl = document.getElementById('nav-breadcrumb');
        if (!breadcrumbEl) return;

        const breadcrumb = this.navigator.getBreadcrumb();
        
        if (breadcrumb.length === 0) {
            breadcrumbEl.innerHTML = '<span class="breadcrumb-item">Start</span>';
            return;
        }

        const html = breadcrumb.map((item, index) => {
            const separator = index > 0 ? '<span class="breadcrumb-sep">›</span>' : '';
            const icon = item.icon || '';
            return `${separator}<span class="breadcrumb-item ${item.type}">${icon} ${item.name}</span>`;
        }).join('');

        breadcrumbEl.innerHTML = html;
    }

    /**
     * Update progress bar
     */
    updateProgressBar() {
        const progress = this.navigator.getProgress();
        const progressFill = document.querySelector('.progress-fill');
        const progressText = document.querySelector('.progress-text');

        if (progressFill) {
            progressFill.style.width = `${progress}%`;
        }
        if (progressText) {
            progressText.textContent = `${progress}% Complete`;
        }
    }

    /**
     * Mark step as complete
     */
    markStepComplete(stepId) {
        const stepEl = document.querySelector(`[data-step="${stepId}"]`);
        if (stepEl) {
            stepEl.classList.remove('step-in-progress');
            stepEl.classList.add('step-completed');
        }
    }

    /**
     * Mark substep as complete
     */
    markSubstepComplete(stepId, substepId) {
        const substepsContainer = document.getElementById(`substeps-${stepId}`);
        if (substepsContainer) {
            const substepEl = substepsContainer.querySelector(`[data-substep="${substepId}"]`);
            if (substepEl) {
                substepEl.classList.remove('pending', 'current');
                substepEl.classList.add('completed');
                const marker = substepEl.querySelector('.substep-marker');
                if (marker) marker.textContent = '✓';
            }
        }
    }

    /**
     * Mark step as error
     */
    markStepError(stepId, error) {
        const stepEl = document.querySelector(`[data-step="${stepId}"]`);
        if (stepEl) {
            stepEl.classList.add('step-error');
            const errorMsg = document.createElement('div');
            errorMsg.className = 'step-error-message';
            errorMsg.textContent = `Error: ${error}`;
            stepEl.appendChild(errorMsg);
        }
    }

    /**
     * Attach event listeners
     */
    attachEventListeners() {
        // Reset and Export buttons
        const resetBtn = this.container.querySelector('.btn-reset');
        const exportBtn = this.container.querySelector('.btn-export');
        
        if (resetBtn) {
            resetBtn.addEventListener('click', () => {
                this.navigator.reset();
            });
        }
        
        if (exportBtn) {
            exportBtn.addEventListener('click', () => {
                this.exportHistory();
            });
        }

        // Protocol step clicks (toggle)
        this.container.querySelectorAll('.protocol-step').forEach(stepEl => {
            stepEl.addEventListener('click', (e) => {
                // Don't toggle if clicking on a substep
                if (e.target.closest('.substep')) return;
                
                const stepId = stepEl.getAttribute('data-step');
                this.toggleStep(stepId);
            });
        });

        // Substep clicks
        this.container.querySelectorAll('.substep').forEach(substepEl => {
            substepEl.addEventListener('click', (e) => {
                e.stopPropagation(); // Prevent triggering step toggle
                const stepId = substepEl.getAttribute('data-step-id');
                const substepId = substepEl.getAttribute('data-substep');
                this.selectSubstep(stepId, substepId);
            });
        });
    }

    /**
     * Export navigation history
     */
    exportHistory() {
        const history = {
            flowHistory: this.navigator.flowHistory,
            stepStates: this.navigator.stepStates,
            currentStep: this.navigator.currentStep,
            progress: this.navigator.getProgress(),
            timestamp: new Date().toISOString()
        };

        const blob = new Blob([JSON.stringify(history, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `gauth-flow-${Date.now()}.json`;
        a.click();
        URL.revokeObjectURL(url);
    }
}

/**
 * Initialize navigator globally
 */
function initProtocolNavigator(containerId = 'protocol-navigator') {
    // Prevent double initialization - return existing instance if already initialized
    if (window.protocolNav && window.protocolNav.navigator) {
        console.log('[protocol-navigator] Already initialized, returning existing instance');
        return {
            navigator: window.protocolNav.navigator,
            ui: window.protocolNav.ui
        };
    }
    
    const navigator = new ProtocolNavigator();
    const ui = new ProtocolNavigatorUI(navigator, containerId);
    
    // Expose globally for easy access
    window.protocolNav = {
        navigator,
        ui,
        toggleStep: (stepId) => ui.toggleStep(stepId),
        selectSubstep: (stepId, substepId) => ui.selectSubstep(stepId, substepId),
        exportHistory: () => ui.exportHistory()
    };

    console.log('[protocol-navigator] Initialized new instance');
    return { navigator, ui };
}

// Auto-initialize if container exists
document.addEventListener('DOMContentLoaded', () => {
    if (document.getElementById('protocol-navigator')) {
        initProtocolNavigator();
    }
});

// Export for module usage
export { ProtocolNavigator, ProtocolNavigatorUI, initProtocolNavigator, PROTOCOL_STEPS };
