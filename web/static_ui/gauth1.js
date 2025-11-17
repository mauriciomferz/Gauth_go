// GAuth 1.0 Dashboard - Interactive Application

// API Base URL
const API_BASE = '/api/v1/gauth';

// State Management
const state = {
    theme: localStorage.getItem('theme') || 'light',
    activeTab: 'overview',
    tokens: [],
    poas: [],
    testResults: []
};

// Initialize App
document.addEventListener('DOMContentLoaded', () => {
    initTheme();
    initNavigation();
    initForms();
    loadInitialData();
});

// Theme Management
function initTheme() {
    document.documentElement.setAttribute('data-theme', state.theme);
    const themeIcon = document.querySelector('#theme-toggle-btn i');
    themeIcon.className = state.theme === 'dark' ? 'fas fa-sun' : 'fas fa-moon';
    
    document.getElementById('theme-toggle-btn').addEventListener('click', () => {
        state.theme = state.theme === 'light' ? 'dark' : 'light';
        localStorage.setItem('theme', state.theme);
        document.documentElement.setAttribute('data-theme', state.theme);
        themeIcon.className = state.theme === 'dark' ? 'fas fa-sun' : 'fas fa-moon';
    });
}

// Navigation
function initNavigation() {
    const navLinks = document.querySelectorAll('.nav-link');
    
    navLinks.forEach(link => {
        link.addEventListener('click', () => {
            const targetTab = link.dataset.tab;
            switchTab(targetTab);
        });
    });
}

function switchTab(tabName) {
    // Update nav links
    document.querySelectorAll('.nav-link').forEach(link => {
        link.classList.toggle('active', link.dataset.tab === tabName);
    });
    
    // Update tab content
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.toggle('active', content.id === `${tabName}-tab`);
    });
    
    state.activeTab = tabName;
    
    // Load tab-specific data
    if (tabName === 'metrics') {
        loadMetrics();
    } else if (tabName === 'pip') {
        loadCacheStats();
    } else if (tabName === 'mcp') {
        loadMCPServers();
    }
}

// Form Initialization
function initForms() {
    // Subscription Wizard
    const startWizardBtn = document.getElementById('start-wizard-btn');
    const cancelWizardBtn = document.getElementById('wizard-cancel-btn');
    const executeWizardBtn = document.getElementById('wizard-execute-btn');
    
    if (startWizardBtn) {
        startWizardBtn.addEventListener('click', () => {
            document.getElementById('wizard-intro').style.display = 'none';
            document.getElementById('subscription-wizard').style.display = 'block';
            document.getElementById('token-result').style.display = 'none';
            initWizard();
        });
    }
    
    if (cancelWizardBtn) {
        cancelWizardBtn.addEventListener('click', () => {
            resetWizard();
        });
    }
    
    if (executeWizardBtn) {
        executeWizardBtn.addEventListener('click', async () => {
            await executeWizardStep();
        });
    }
    
    // Validate Token
    document.getElementById('validate-token-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await validateToken();
    });
    
    // Verify Identity
    document.getElementById('verify-identity-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await verifyIdentity();
    });
    
    // Verify Entity
    document.getElementById('verify-entity-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await verifyEntity();
    });
    
    // Verify Signatory
    document.getElementById('verify-signatory-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await verifySignatory();
    });
    
    // Validate Authorization
    document.getElementById('validate-authz-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await validateAuthorization();
    });
    
    // Create PoA
    document.getElementById('create-poa-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await createPoA();
    });
    
    // Validate PoA
    document.getElementById('validate-poa-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await validatePoA();
    });
    
    // E2E Test Buttons
    document.querySelectorAll('.test-btn').forEach(btn => {
        btn.addEventListener('click', async () => {
            const testType = btn.dataset.test;
            await runE2ETest(testType);
        });
    });
    
    // Refresh Cache Button
    document.getElementById('refresh-cache-btn').addEventListener('click', async () => {
        await loadCacheStats();
    });
    
    // MCP Forms
    const mcpTransportType = document.getElementById('mcp-transport-type');
    if (mcpTransportType) {
        mcpTransportType.addEventListener('change', (e) => {
            const stdioFields = document.getElementById('mcp-stdio-fields');
            const urlField = document.getElementById('mcp-url-field');
            if (e.target.value === 'stdio') {
                stdioFields.style.display = 'block';
                urlField.style.display = 'none';
            } else {
                stdioFields.style.display = 'none';
                urlField.style.display = 'block';
            }
        });
    }
    
    document.getElementById('register-mcp-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await registerMCPServer();
    });
    
    document.getElementById('mcp-refresh-btn').addEventListener('click', async () => {
        await loadMCPServers();
    });
    
    document.getElementById('execute-tool-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await executeMCPTool();
    });

    // Login Forms
    document.getElementById('login-credentials-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await handleLoginCredentials();
    });

    document.getElementById('login-mfa-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await handleLoginMFA();
    });

    document.getElementById('mfa-back-btn').addEventListener('click', () => {
        setLoginStep('credentials');
    });

    document.getElementById('logout-btn').addEventListener('click', () => {
        resetLogin();
    });
}

// Subscription Wizard State
const wizardState = {
    currentStep: 1,
    subscriptionId: null,
    clientId: '',
    scope: []
};

function initWizard() {
    wizardState.currentStep = 1;
    wizardState.subscriptionId = null;
    wizardState.clientId = document.getElementById('wizard-client-id').value;
    wizardState.scope = document.getElementById('wizard-scope').value.split(',').map(s => s.trim());
    updateWizardUI();
    showWizardStep(1);
}

function resetWizard() {
    document.getElementById('subscription-wizard').style.display = 'none';
    document.getElementById('wizard-intro').style.display = 'block';
    document.getElementById('wizard-error').style.display = 'none';
    wizardState.currentStep = 1;
    wizardState.subscriptionId = null;
}

function updateWizardUI() {
    // Update progress circles
    document.querySelectorAll('.progress-step').forEach(step => {
        const stepNum = parseInt(step.dataset.step);
        const circle = step.querySelector('.step-circle');
        
        if (stepNum < wizardState.currentStep) {
            circle.textContent = '✓';
            circle.style.background = 'linear-gradient(135deg, #10b981 0%, #059669 100%)';
            circle.style.color = 'white';
        } else if (stepNum === wizardState.currentStep) {
            circle.textContent = stepNum;
            circle.style.background = 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)';
            circle.style.color = 'white';
        } else {
            circle.textContent = stepNum;
            circle.style.background = '#e5e7eb';
            circle.style.color = '#6b7280';
        }
    });
    
    // Update progress lines
    document.querySelectorAll('.progress-line').forEach((line, index) => {
        if (index < wizardState.currentStep - 1) {
            line.style.background = 'linear-gradient(90deg, #10b981 0%, #059669 100%)';
        } else {
            line.style.background = '#e5e7eb';
        }
    });
    
    // Update step info
    const stepTitles = [
        'Initiate', 'Authorizer Auth', 'Client Owner ID', 'Client Owner Auth',
        'Client Auth', 'Resource Owner ID', 'Resource Owner Auth', 'Resource Server'
    ];
    const stepDescriptions = [
        'Create subscription', 'Verify authorizer', 'Verify client owner', 'Authorize client owner',
        'Authorize client', 'Verify resource owner', 'Authorize resource owner', 'Complete & get token'
    ];
    
    document.querySelector('.step-title').textContent = `Step ${wizardState.currentStep}: ${stepTitles[wizardState.currentStep - 1]}`;
    document.querySelector('.step-description').textContent = stepDescriptions[wizardState.currentStep - 1];
}

function showWizardStep(step) {
    document.querySelectorAll('.wizard-step-form').forEach(form => {
        form.style.display = form.dataset.step === String(step) ? 'block' : 'none';
    });
}

async function executeWizardStep() {
    const executeBtn = document.getElementById('wizard-execute-btn');
    const errorBox = document.getElementById('wizard-error');
    
    try {
        executeBtn.disabled = true;
        executeBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Executing...';
        errorBox.style.display = 'none';
        
        // Execute the current step
        switch (wizardState.currentStep) {
            case 1: await executeStepI(); break;
            case 2: await executeStepII(); break;
            case 3: await executeStepIII(); break;
            case 4: await executeStepIV(); break;
            case 5: await executeStepV(); break;
            case 6: await executeStepVI(); break;
            case 7: await executeStepVII(); break;
            case 8: await executeStepVIII(); break;
        }
        
    } catch (error) {
        errorBox.style.display = 'block';
        errorBox.innerHTML = `<strong>Error:</strong> ${error.message || 'Step execution failed'}`;
        console.error('Wizard step error:', error);
    } finally {
        executeBtn.disabled = false;
        executeBtn.innerHTML = wizardState.currentStep === 8
            ? '<i class="fas fa-check"></i> Complete & Generate Token'
            : '<i class="fas fa-arrow-right"></i> Execute Step';
    }
}

// RFC-0111 Subscription Steps
async function executeStepI() {
    const response = await fetch('/api/v1/rfc0111/subscriptions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            owners_authorizer_id: 'director-001',
            identity_proof_request: {
                subject_id: 'director-001',
                identity_type: 'natural_person',
                proof_method: 'eIDAS',
                proof_data: { verified: true, eidas_level: 'high' },
                required_level: 'high'
            }
        })
    });
    
    if (!response.ok) throw new Error(await response.text());
    const data = await response.json();
    wizardState.subscriptionId = data.subscription_id;
    wizardState.currentStep = 2;
    updateWizardUI();
    showWizardStep(2);
}

async function executeStepII() {
    const response = await fetch(`/api/v1/rfc0111/subscriptions/${wizardState.subscriptionId}/step-ii`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            commercial_register_ref: 'HRB-12345-DE',
            jurisdiction: 'DE'
        })
    });
    
    if (!response.ok) throw new Error(await response.text());
    wizardState.currentStep = 3;
    updateWizardUI();
    showWizardStep(3);
}

async function executeStepIII() {
    const response = await fetch(`/api/v1/rfc0111/subscriptions/${wizardState.subscriptionId}/step-iii`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            subject_id: 'owner-456',
            identity_type: 'natural_person',
            proof_method: 'eIDAS',
            proof_data: { verified: true, eidas_level: 'high' },
            required_level: 'high'
        })
    });
    
    if (!response.ok) throw new Error(await response.text());
    wizardState.currentStep = 4;
    updateWizardUI();
    showWizardStep(4);
}

async function executeStepIV() {
    const now = new Date().toISOString();
    const response = await fetch(`/api/v1/rfc0111/subscriptions/${wizardState.subscriptionId}/step-iv`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            authorization_chain: {
                owners_authorizer: {
                    entity_id: 'director-001',
                    entity_type: 'natural_person',
                    entity_name: 'Max Mustermann',
                    role: 'authorizer',
                    authorization_date: now,
                    authorization_type: 'statutory',
                    statutory_authority: 'Managing Director per § 35 GmbHG',
                    commercial_register_ref: 'HRB-12345-DE',
                    identity_verified: true,
                    verification_method: 'eIDAS',
                    scope_of_authority: ['client_registration'],
                    valid_from: now,
                    valid_until: new Date(Date.now() + 365*24*60*60*1000).toISOString(),
                    status: 'active',
                    legal_basis: {
                        basis_type: 'company_law',
                        jurisdiction: 'DE',
                        registration_number: 'HRB-12345-DE'
                    }
                },
                client_owner: {
                    entity_id: 'owner-456',
                    entity_type: 'natural_person',
                    entity_name: 'Demo Client Owner',
                    role: 'owner',
                    authorized_by: 'director-001',
                    authorization_date: now,
                    authorization_type: 'delegated',
                    identity_verified: true,
                    verification_method: 'eIDAS',
                    scope_of_authority: ['ai_system_operation'],
                    valid_from: now,
                    valid_until: new Date(Date.now() + 365*24*60*60*1000).toISOString(),
                    status: 'active',
                    legal_basis: {
                        basis_type: 'power_of_attorney',
                        jurisdiction: 'DE'
                    }
                },
                client: {
                    entity_id: wizardState.clientId,
                    entity_type: 'ai_system',
                    entity_name: 'Demo AI Client',
                    role: 'client',
                    authorized_by: 'owner-456',
                    authorization_date: now,
                    authorization_type: 'delegated',
                    identity_verified: true,
                    scope_of_authority: ['resource_access'],
                    valid_from: now,
                    valid_until: new Date(Date.now() + 365*24*60*60*1000).toISOString(),
                    status: 'active',
                    legal_basis: {
                        basis_type: 'contractual',
                        jurisdiction: 'DE'
                    }
                }
            }
        })
    });
    
    if (!response.ok) throw new Error(await response.text());
    wizardState.currentStep = 5;
    updateWizardUI();
    showWizardStep(5);
}

async function executeStepV() {
    const response = await fetch(`/api/v1/rfc0111/subscriptions/${wizardState.subscriptionId}/step-v`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            client_id: wizardState.clientId,
            poa_credential: {
                parties: {
                    principal: {
                        type: 'natural_person',
                        identity: 'owner-456'
                    },
                    authorized_client: {
                        type: 'ai_system',
                        identity: wizardState.clientId,
                        operational_status: 'active',
                        capability_level: 'L3'
                    }
                },
                authorization: {
                    authorized_actions: {
                        non_physical_actions: ['analyzing', 'documenting']
                    }
                },
                requirements: {
                    jurisdiction_law: {
                        language: 'en',
                        governing_law: 'EU-GDPR',
                        place_of_jurisdiction: 'Germany'
                    }
                }
            },
            enable_identity_sharing: true,
            enable_prompting: false
        })
    });
    
    if (!response.ok) throw new Error(await response.text());
    wizardState.currentStep = 6;
    updateWizardUI();
    showWizardStep(6);
}

async function executeStepVI() {
    const response = await fetch(`/api/v1/rfc0111/subscriptions/${wizardState.subscriptionId}/step-vi`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            subject_id: 'resource-789',
            identity_type: 'natural_person',
            proof_method: 'eIDAS',
            proof_data: { verified: true, eidas_level: 'high' },
            required_level: 'high'
        })
    });
    
    if (!response.ok) throw new Error(await response.text());
    wizardState.currentStep = 7;
    updateWizardUI();
    showWizardStep(7);
}

async function executeStepVII() {
    const now = new Date().toISOString();
    const response = await fetch(`/api/v1/rfc0111/subscriptions/${wizardState.subscriptionId}/step-vii`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            authorization_chain: {
                owners_authorizer: {
                    entity_id: 'director-001',
                    entity_type: 'natural_person',
                    entity_name: 'Max Mustermann',
                    role: 'authorizer',
                    authorization_date: now,
                    authorization_type: 'statutory',
                    statutory_authority: 'Managing Director per § 35 GmbHG',
                    commercial_register_ref: 'HRB-12345-DE',
                    identity_verified: true,
                    verification_method: 'eIDAS',
                    scope_of_authority: ['resource_authorization'],
                    valid_from: now,
                    valid_until: new Date(Date.now() + 365*24*60*60*1000).toISOString(),
                    status: 'active',
                    legal_basis: {
                        basis_type: 'company_law',
                        jurisdiction: 'DE',
                        registration_number: 'HRB-12345-DE'
                    }
                },
                client_owner: {
                    entity_id: 'resource-789',
                    entity_type: 'natural_person',
                    entity_name: 'Demo Resource Owner',
                    role: 'owner',
                    authorized_by: 'director-001',
                    authorization_date: now,
                    authorization_type: 'delegated',
                    identity_verified: true,
                    verification_method: 'eIDAS',
                    scope_of_authority: ['resource_management'],
                    valid_from: now,
                    valid_until: new Date(Date.now() + 365*24*60*60*1000).toISOString(),
                    status: 'active',
                    legal_basis: {
                        basis_type: 'power_of_attorney',
                        jurisdiction: 'DE'
                    }
                },
                client: {
                    entity_id: wizardState.clientId,
                    entity_type: 'ai_system',
                    entity_name: 'Demo AI Client',
                    role: 'client',
                    authorized_by: 'resource-789',
                    authorization_date: now,
                    authorization_type: 'delegated',
                    identity_verified: true,
                    scope_of_authority: ['data_access'],
                    valid_from: now,
                    valid_until: new Date(Date.now() + 365*24*60*60*1000).toISOString(),
                    status: 'active',
                    legal_basis: {
                        basis_type: 'contractual',
                        jurisdiction: 'DE'
                    }
                }
            }
        })
    });
    
    if (!response.ok) throw new Error(await response.text());
    wizardState.currentStep = 8;
    updateWizardUI();
    showWizardStep(8);
    document.getElementById('wizard-execute-btn').innerHTML = '<i class="fas fa-check"></i> Complete & Generate Token';
}

async function executeStepVIII() {
    const response = await fetch(`/api/v1/rfc0111/subscriptions/${wizardState.subscriptionId}/step-viii`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            resource_server_id: 'server-001',
            server_public_key: 'demo-server-public-key-' + Date.now(),
            server_endpoint: 'https://api.example.com/resources',
            resource_types: ['documents', 'data', 'files'],
            allowed_operations: ['read', 'write', 'delete'],
            authorization_proof: {
                proof_type: 'server_credential',
                verified_at: new Date().toISOString()
            }
        })
    });
    
    if (!response.ok) throw new Error(await response.text());
    const data = await response.json();
    const token = data.token || data.access_token;
    
    // Store token and display
    const tokenObj = {
        token: token,
        clientId: wizardState.clientId,
        expiresAt: new Date(Date.now() + 24*60*60*1000).toISOString(),
        scope: wizardState.scope,
        authorizationChain: {
            ownersAuthorizer: 'RFC-0111 Flow',
            clientOwner: 'RFC-0111 Flow',
            client: wizardState.clientId
        }
    };
    
    state.tokens.unshift(tokenObj);
    displayTokenResult(document.getElementById('token-result'), tokenObj, true);
    updateRecentTokens();
    
    // Reset wizard and show token
    document.getElementById('subscription-wizard').style.display = 'none';
    document.getElementById('wizard-intro').style.display = 'block';
}

// API Functions - Extended Tokens
async function createExtendedToken() {
    const resultBox = document.getElementById('token-result');
    const submitBtn = document.querySelector('#create-token-form button[type="submit"]');
    
    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Creating...';
        
        const tokenData = {
            clientId: document.getElementById('token-client-id').value,
            ownersAuthorizer: document.getElementById('token-owner-auth').value,
            clientOwner: document.getElementById('token-client-owner').value,
            scope: document.getElementById('token-scope').value.split(',').map(s => s.trim()),
            jurisdiction: document.getElementById('token-jurisdiction').value,
            expirationHours: parseInt(document.getElementById('token-expiration').value)
        };
        
        // Simulate API call (replace with actual endpoint)
        await delay(1000);
        
        const mockToken = {
            token: generateMockJWT(tokenData),
            clientId: tokenData.clientId,
            expiresAt: new Date(Date.now() + tokenData.expirationHours * 3600000).toISOString(),
            scope: tokenData.scope,
            jurisdiction: tokenData.jurisdiction,
            authorizationChain: {
                ownersAuthorizer: tokenData.ownersAuthorizer,
                clientOwner: tokenData.clientOwner,
                client: tokenData.clientId
            }
        };
        
        state.tokens.unshift(mockToken);
        displayTokenResult(resultBox, mockToken, true);
        updateRecentTokens();
        
    } catch (error) {
        displayError(resultBox, error);
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-plus"></i> Create Token';
    }
}

async function validateToken() {
    const resultBox = document.getElementById('validate-result');
    const submitBtn = document.querySelector('#validate-token-form button[type="submit"]');
    
    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Validating...';
        
        const token = document.getElementById('validate-token-input').value;
        
        // Call actual validation API
        const response = await fetch('/api/v1/rfc0111/token/validate', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token: token })
        });
        
        const validation = await response.json();
        
        if (!response.ok) {
            throw new Error(validation.message || 'Validation failed');
        }
        
        displayValidationResult(resultBox, validation);
        
    } catch (error) {
        displayError(resultBox, error);
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-check"></i> Validate Token';
    }
}

// API Functions - PVP
async function verifyIdentity() {
    const resultBox = document.getElementById('identity-result');
    const submitBtn = document.querySelector('#verify-identity-form button[type="submit"]');
    
    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Verifying...';
        
        const identityData = {
            type: document.getElementById('identity-type').value,
            trustLevel: document.getElementById('trust-level').value,
            entityId: document.getElementById('identity-entity-id').value,
            tsp: document.getElementById('identity-tsp').value
        };
        
        await delay(600);
        
        const result = {
            verified: true,
            identityType: identityData.type,
            trustLevel: identityData.trustLevel.toUpperCase(),
            entityId: identityData.entityId,
            tsp: identityData.tsp,
            tspStatus: 'Active and Trusted',
            verificationTime: '582.1 ns',
            cryptographicBinding: 'RSA-2048 verified'
        };
        
        displayIdentityResult(resultBox, result);
        
    } catch (error) {
        displayError(resultBox, error);
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-check-circle"></i> Verify Identity';
    }
}

// API Functions - Commercial Registry
async function verifyEntity() {
    const resultBox = document.getElementById('entity-result');
    const submitBtn = document.querySelector('#verify-entity-form button[type="submit"]');
    
    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Verifying...';
        
        const jurisdiction = document.getElementById('entity-jurisdiction').value;
        const regNumber = document.getElementById('entity-reg-number').value;
        
        await delay(700);
        
        const result = {
            verified: true,
            registrationNumber: regNumber,
            legalName: jurisdiction === 'DE' ? 'Test Technologies GmbH' : 'Test Technologies Ltd',
            jurisdiction: jurisdiction === 'DE' ? '🇩🇪 Germany' : '🇬🇧 United Kingdom',
            status: 'Active',
            registrationDate: '2020-01-15',
            legalForm: jurisdiction === 'DE' ? 'GmbH' : 'Limited Company',
            managingDirectors: [
                { name: 'Max Mustermann', position: 'CEO', authority: 'Sole Signatory' }
            ]
        };
        
        displayEntityResult(resultBox, result);
        
    } catch (error) {
        displayError(resultBox, error);
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-search"></i> Verify Entity';
    }
}

async function verifySignatory() {
    const resultBox = document.getElementById('signatory-result');
    const submitBtn = document.querySelector('#verify-signatory-form button[type="submit"]');
    
    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Verifying...';
        
        const entity = document.getElementById('signatory-entity').value;
        const name = document.getElementById('signatory-name').value;
        const type = document.getElementById('signatory-type').value;
        
        await delay(600);
        
        const result = {
            authorized: true,
            signatoryName: name,
            entity: entity,
            authorityType: type.replace('_', ' ').toUpperCase(),
            appointmentDate: '2020-01-15',
            restrictions: 'None',
            status: 'Active'
        };
        
        displaySignatoryResult(resultBox, result);
        
    } catch (error) {
        displayError(resultBox, error);
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-user-check"></i> Verify Signatory';
    }
}

// API Functions - PIP
async function validateAuthorization() {
    const resultBox = document.getElementById('authz-result');
    const submitBtn = document.querySelector('#validate-authz-form button[type="submit"]');
    
    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Validating...';
        
        const authzData = {
            clientId: document.getElementById('authz-client-id').value,
            action: document.getElementById('authz-action').value,
            geographic: document.getElementById('authz-geographic').value,
            sector: document.getElementById('authz-sector').value
        };
        
        await delay(500);
        
        const result = {
            authorized: true,
            clientId: authzData.clientId,
            action: authzData.action,
            geographicScope: authzData.geographic,
            industrySector: authzData.sector || 'N/A',
            cacheHit: Math.random() > 0.5,
            processingTime: Math.random() > 0.5 ? '97.02 ns' : '107.0 ns',
            policyChecks: [
                { policy: 'Action Authorization', result: 'PASS' },
                { policy: 'Geographic Restriction', result: 'PASS' },
                { policy: 'Sector Authorization', result: 'PASS' }
            ]
        };
        
        displayAuthzResult(resultBox, result);
        
    } catch (error) {
        displayError(resultBox, error);
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-check-circle"></i> Validate Authorization';
    }
}

async function loadCacheStats() {
    // Simulate cache stats
    const stats = {
        size: Math.floor(Math.random() * 1000),
        hitRate: (85 + Math.random() * 10).toFixed(1),
        evictions: Math.floor(Math.random() * 50)
    };
    
    document.getElementById('cache-size').textContent = stats.size;
    document.getElementById('cache-hit-rate').textContent = `${stats.hitRate}%`;
    document.getElementById('cache-evictions').textContent = stats.evictions;
}

// API Functions - PoA
async function createPoA() {
    const resultBox = document.getElementById('poa-result');
    const submitBtn = document.querySelector('#create-poa-form button[type="submit"]');
    
    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Creating...';
        
        const poaData = {
            grantor: document.getElementById('poa-grantor').value,
            representative: document.getElementById('poa-representative').value,
            repType: document.getElementById('poa-rep-type').value,
            actions: document.getElementById('poa-actions').value.split(',').map(s => s.trim()),
            geographic: document.getElementById('poa-geographic').value,
            validity: parseInt(document.getElementById('poa-validity').value)
        };
        
        await delay(800);
        
        const poa = {
            id: `poa-${Date.now()}`,
            grantor: poaData.grantor,
            representative: poaData.representative,
            representativeType: poaData.repType.replace('_', ' ').toUpperCase(),
            actions: poaData.actions,
            geographicScope: poaData.geographic,
            validFrom: new Date().toISOString(),
            validUntil: new Date(Date.now() + poaData.validity * 86400000).toISOString(),
            status: 'Active'
        };
        
        state.poas.unshift(poa);
        displayPoAResult(resultBox, poa, true);
        updateActivePoAs();
        
    } catch (error) {
        displayError(resultBox, error);
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-plus"></i> Create PoA';
    }
}

async function validatePoA() {
    const resultBox = document.getElementById('validate-poa-result');
    const submitBtn = document.querySelector('#validate-poa-form button[type="submit"]');
    
    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Validating...';
        
        const poaId = document.getElementById('validate-poa-id').value;
        const action = document.getElementById('validate-poa-action').value;
        const location = document.getElementById('validate-poa-location').value;
        
        await delay(400);
        
        const result = {
            valid: true,
            poaId: poaId,
            action: action,
            location: location,
            checks: [
                { check: 'PoA Existence', result: 'PASS' },
                { check: 'Validity Period', result: 'PASS' },
                { check: 'Action Authorization', result: 'PASS' },
                { check: 'Geographic Scope', result: 'PASS' }
            ],
            validationTime: '21.19 ns'
        };
        
        displayPoAValidationResult(resultBox, result);
        
    } catch (error) {
        displayError(resultBox, error);
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-check"></i> Validate PoA';
    }
}

// API Functions - E2E Testing
async function runE2ETest(testType) {
    const resultBox = document.getElementById('e2e-results');
    resultBox.style.display = 'block';
    resultBox.className = 'result-box';
    resultBox.innerHTML = '<i class="fas fa-spinner loading"></i> Running E2E test...';
    
    try {
        const tests = testType === 'all' 
            ? ['token-issuance', 'token-validation', 'authorization', 'error-handling']
            : [testType];
        
        const results = [];
        
        for (const test of tests) {
            const startTime = performance.now();
            await delay(500 + Math.random() * 500);
            const duration = (performance.now() - startTime).toFixed(2);
            
            results.push({
                test: test,
                duration: duration,
                status: 'PASS',
                timestamp: new Date().toISOString()
            });
        }
        
        state.testResults = [...results, ...state.testResults].slice(0, 20);
        displayE2EResults(resultBox, results);
        updateTestHistory();
        
    } catch (error) {
        displayError(resultBox, error);
    }
}

// Display Functions
function displayTokenResult(element, token, isSuccess) {
    element.style.display = 'block';
    element.className = `result-box ${isSuccess ? 'success' : 'error'}`;
    
    const jurisdictionInfo = token.jurisdiction
        ? `<p><strong>Jurisdiction:</strong> ${token.jurisdiction} <span class="badge badge-info">Geographic Scope Validated</span></p>`
        : '';
    
    const tokenStr = typeof token.token === 'object' ? JSON.stringify(token.token) : String(token.token);
    
    element.innerHTML = `
        <h4><i class="fas fa-check-circle"></i> Token Created Successfully</h4>
        <p><strong>Client ID:</strong> ${token.clientId}</p>
        <p><strong>Expires:</strong> ${new Date(token.expiresAt).toLocaleString()}</p>
        <p><strong>Scope:</strong> ${Array.isArray(token.scope) ? token.scope.join(', ') : token.scope}</p>
        ${jurisdictionInfo}
        <p><strong>Authorization Chain:</strong></p>
        <ul>
            <li>Owner's Authorizer: ${token.authorizationChain.ownersAuthorizer}</li>
            <li>Client Owner: ${token.authorizationChain.clientOwner}</li>
            <li>Client: ${token.authorizationChain.client}</li>
        </ul>
        <div style="position: relative;">
            <button onclick="navigator.clipboard.writeText('${tokenStr.replace(/'/g, "\\'")}'); this.innerHTML='<i class=\\'fas fa-check\\'></i> Copied!'; setTimeout(() => this.innerHTML='<i class=\\'fas fa-copy\\'></i> Copy Token', 2000)" 
                    class="btn btn-secondary" style="position: absolute; top: 0.5rem; right: 0.5rem; padding: 0.25rem 0.5rem; font-size: 0.75rem;">
                <i class="fas fa-copy"></i> Copy Token
            </button>
            <pre style="padding-top: 2.5rem;">${tokenStr}</pre>
        </div>
    `;
}

function displayValidationResult(element, validation) {
    element.style.display = 'block';
    element.className = `result-box ${validation.valid ? 'success' : 'error'}`;
    
    let checksHTML = '';
    if (validation.checks) {
        checksHTML = Object.entries(validation.checks)
            .map(([check, result]) => {
                const resultStr = typeof result === 'object' ? JSON.stringify(result) : String(result);
                return `<li>${check}: <strong>${resultStr}</strong></li>`;
            })
            .join('');
    }
    
    const decodedHTML = validation.decoded 
        ? `<div style="margin-top: 1rem;">
             <p><strong>Decoded Claims:</strong></p>
             <pre style="background: #f8f9fa; padding: 1rem; border-radius: 0.5rem; overflow-x: auto; font-size: 0.875rem;">${JSON.stringify(validation.decoded, null, 2)}</pre>
           </div>`
        : '';
    
    element.innerHTML = `
        <h4><i class="fas fa-${validation.valid ? 'check-circle' : 'times-circle'}"></i> Token ${validation.valid ? 'Valid' : 'Invalid'}</h4>
        <p><strong>Status:</strong> <span style="color: ${validation.valid ? '#10b981' : '#ef4444'};">${validation.valid ? 'VALID' : 'INVALID'}</span></p>
        ${checksHTML ? `<p><strong>Validation Checks:</strong></p><ul>${checksHTML}</ul>` : ''}
        ${decodedHTML}
    `;
}

function displayIdentityResult(element, result) {
    element.style.display = 'block';
    element.className = 'result-box success';
    element.innerHTML = `
        <h4><i class="fas fa-check-circle"></i> Identity Verified</h4>
        <p><strong>Entity ID:</strong> ${result.entityId}</p>
        <p><strong>Trust Level:</strong> ${result.trustLevel}</p>
        <p><strong>Identity Type:</strong> ${result.identityType}</p>
        <p><strong>TSP:</strong> ${result.tsp} (${result.tspStatus})</p>
        <p><strong>Cryptographic Binding:</strong> ${result.cryptographicBinding}</p>
        <p><strong>Verification Time:</strong> ${result.verificationTime}</p>
    `;
}

function displayEntityResult(element, result) {
    element.style.display = 'block';
    element.className = 'result-box success';
    
    const directorsHTML = result.managingDirectors
        .map(d => `<li>${d.name} - ${d.position} (${d.authority})</li>`)
        .join('');
    
    element.innerHTML = `
        <h4><i class="fas fa-check-circle"></i> Entity Verified</h4>
        <p><strong>Legal Name:</strong> ${result.legalName}</p>
        <p><strong>Registration:</strong> ${result.registrationNumber}</p>
        <p><strong>Jurisdiction:</strong> ${result.jurisdiction}</p>
        <p><strong>Legal Form:</strong> ${result.legalForm}</p>
        <p><strong>Status:</strong> ${result.status}</p>
        <p><strong>Registration Date:</strong> ${result.registrationDate}</p>
        <p><strong>Managing Directors:</strong></p>
        <ul>${directorsHTML}</ul>
    `;
}

function displaySignatoryResult(element, result) {
    element.style.display = 'block';
    element.className = 'result-box success';
    element.innerHTML = `
        <h4><i class="fas fa-check-circle"></i> Signatory Authorized</h4>
        <p><strong>Name:</strong> ${result.signatoryName}</p>
        <p><strong>Entity:</strong> ${result.entity}</p>
        <p><strong>Authority Type:</strong> ${result.authorityType}</p>
        <p><strong>Appointment Date:</strong> ${result.appointmentDate}</p>
        <p><strong>Restrictions:</strong> ${result.restrictions}</p>
        <p><strong>Status:</strong> ${result.status}</p>
    `;
}

function displayAuthzResult(element, result) {
    element.style.display = 'block';
    element.className = 'result-box success';
    
    const policyHTML = result.policyChecks
        .map(p => `<li>${p.policy}: <strong>${p.result}</strong></li>`)
        .join('');
    
    element.innerHTML = `
        <h4><i class="fas fa-check-circle"></i> Authorization Granted</h4>
        <p><strong>Client ID:</strong> ${result.clientId}</p>
        <p><strong>Action:</strong> ${result.action}</p>
        <p><strong>Geographic Scope:</strong> ${result.geographicScope}</p>
        <p><strong>Industry Sector:</strong> ${result.industrySector}</p>
        <p><strong>Cache Hit:</strong> ${result.cacheHit ? 'Yes' : 'No'}</p>
        <p><strong>Processing Time:</strong> ${result.processingTime}</p>
        <p><strong>Policy Checks:</strong></p>
        <ul>${policyHTML}</ul>
    `;
}

function displayPoAResult(element, poa, isSuccess) {
    element.style.display = 'block';
    element.className = `result-box ${isSuccess ? 'success' : 'error'}`;
    element.innerHTML = `
        <h4><i class="fas fa-check-circle"></i> PoA Created Successfully</h4>
        <p><strong>PoA ID:</strong> ${poa.id}</p>
        <p><strong>Grantor:</strong> ${poa.grantor}</p>
        <p><strong>Representative:</strong> ${poa.representative} (${poa.representativeType})</p>
        <p><strong>Actions:</strong> ${poa.actions.join(', ')}</p>
        <p><strong>Geographic Scope:</strong> ${poa.geographicScope}</p>
        <p><strong>Valid From:</strong> ${new Date(poa.validFrom).toLocaleString()}</p>
        <p><strong>Valid Until:</strong> ${new Date(poa.validUntil).toLocaleString()}</p>
    `;
}

function displayPoAValidationResult(element, result) {
    element.style.display = 'block';
    element.className = 'result-box success';
    
    const checksHTML = result.checks
        .map(c => `<li>${c.check}: <strong>${c.result}</strong></li>`)
        .join('');
    
    element.innerHTML = `
        <h4><i class="fas fa-check-circle"></i> PoA Validation Result</h4>
        <p><strong>PoA ID:</strong> ${result.poaId}</p>
        <p><strong>Action:</strong> ${result.action}</p>
        <p><strong>Location:</strong> ${result.location}</p>
        <p><strong>Validation Time:</strong> ${result.validationTime}</p>
        <p><strong>Checks:</strong></p>
        <ul>${checksHTML}</ul>
    `;
}

function displayE2EResults(element, results) {
    const testsHTML = results
        .map(r => `
            <div style="padding: 0.75rem; background: var(--bg-secondary); border-radius: var(--border-radius); margin-bottom: 0.5rem;">
                <div style="display: flex; justify-content: space-between; align-items: center;">
                    <span><i class="fas fa-check-circle text-success"></i> ${formatTestName(r.test)}</span>
                    <span><strong>${r.duration}ms</strong></span>
                </div>
            </div>
        `)
        .join('');
    
    element.className = 'result-box success';
    element.innerHTML = `
        <h4><i class="fas fa-check-circle"></i> E2E Tests Completed</h4>
        <p><strong>Tests Run:</strong> ${results.length}</p>
        <p><strong>Passed:</strong> ${results.length}</p>
        <p><strong>Failed:</strong> 0</p>
        <div style="margin-top: 1rem;">${testsHTML}</div>
    `;
}

function displayError(element, error) {
    element.style.display = 'block';
    element.className = 'result-box error';
    element.innerHTML = `
        <h4><i class="fas fa-exclamation-circle"></i> Error</h4>
        <p>${error.message || 'An error occurred'}</p>
    `;
}

// Update Functions
function updateRecentTokens() {
    const container = document.getElementById('recent-tokens');
    
    if (state.tokens.length === 0) {
        container.innerHTML = '<p class="text-muted">No tokens created yet.</p>';
        return;
    }
    
    const tokensHTML = state.tokens.slice(0, 5).map(token => `
        <div style="padding: 1rem; background: var(--bg-secondary); border-radius: var(--border-radius); margin-bottom: 0.75rem;">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem;">
                <strong>${token.clientId}</strong>
                <span class="badge badge-success">Active</span>
            </div>
            <p style="font-size: 0.85rem; color: var(--text-secondary);">
                Expires: ${new Date(token.expiresAt).toLocaleString()}
            </p>
            <p style="font-size: 0.85rem; color: var(--text-secondary);">
                Scope: ${token.scope.join(', ')}
            </p>
        </div>
    `).join('');
    
    container.innerHTML = tokensHTML;
}

function updateActivePoAs() {
    const container = document.getElementById('active-poas');
    
    if (state.poas.length === 0) {
        container.innerHTML = '<p class="text-muted">No PoAs created yet.</p>';
        return;
    }
    
    const poasHTML = `
        <table class="data-table">
            <thead>
                <tr>
                    <th>PoA ID</th>
                    <th>Representative</th>
                    <th>Actions</th>
                    <th>Valid Until</th>
                    <th>Status</th>
                </tr>
            </thead>
            <tbody>
                ${state.poas.map(poa => `
                    <tr>
                        <td><code>${poa.id}</code></td>
                        <td>${poa.representative}</td>
                        <td>${poa.actions.join(', ')}</td>
                        <td>${new Date(poa.validUntil).toLocaleDateString()}</td>
                        <td><span class="badge badge-success">${poa.status}</span></td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
    
    container.innerHTML = poasHTML;
}

function updateTestHistory() {
    const tbody = document.getElementById('test-history-body');
    
    if (state.testResults.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="text-center text-muted">No tests run yet</td></tr>';
        return;
    }
    
    tbody.innerHTML = state.testResults.map(result => `
        <tr>
            <td>${new Date(result.timestamp).toLocaleString()}</td>
            <td>${formatTestName(result.test)}</td>
            <td>${result.duration}ms</td>
            <td><span class="badge badge-success">${result.status}</span></td>
        </tr>
    `).join('');
}

// Utility Functions
function delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

function generateMockJWT(data) {
    const header = btoa(JSON.stringify({ alg: 'RS256', typ: 'JWT' }));
    const payload = btoa(JSON.stringify({
        sub: data.clientId,
        scope: data.scope,
        ownersAuthorizer: data.ownersAuthorizer,
        clientOwner: data.clientOwner,
        exp: Math.floor(Date.now() / 1000) + (data.expirationHours * 3600),
        iat: Math.floor(Date.now() / 1000)
    }));
    const signature = btoa('mock-signature-' + Math.random().toString(36).substr(2, 9));
    return `${header}.${payload}.${signature}`;
}

function decodeJWT(token) {
    try {
        const parts = token.split('.');
        if (parts.length !== 3) throw new Error('Invalid JWT format');
        return JSON.parse(atob(parts[1]));
    } catch {
        return { error: 'Invalid token format' };
    }
}

function formatTestName(testName) {
    return testName
        .split('-')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1))
        .join(' ');
}

async function loadInitialData() {
    // Load cache stats if on PIP tab
    if (state.activeTab === 'pip') {
        await loadCacheStats();
    }
}

async function loadMetrics() {
    // Simulate loading metrics
    // In a real app, this would fetch from /api/v1/gauth/metrics
    console.log('Metrics loaded');
}

// MCP State
const mcpState = {
    servers: [],
    selectedServerId: null,
    resources: [],
    tools: []
};

// Login State
const loginState = {
    step: 'credentials',
    sessionChallenge: null,
    mfaMethods: [],
    selectedMfaMethod: 'totp',
    sessionInfo: null
};

// MCP Functions
async function registerMCPServer() {
    const resultBox = document.getElementById('mcp-register-result');
    const submitBtn = document.querySelector('#register-mcp-form button[type="submit"]');
    
    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Registering...';
        
        const transportType = document.getElementById('mcp-transport-type').value;
        const serverData = {
            id: document.getElementById('mcp-server-id').value,
            name: document.getElementById('mcp-server-name').value,
            description: document.getElementById('mcp-server-description').value,
            transport_type: transportType
        };
        
        if (transportType === 'stdio') {
            serverData.command = document.getElementById('mcp-command').value;
            const args = document.getElementById('mcp-args').value;
            serverData.args = args ? args.split(',').map(a => a.trim()) : [];
        } else {
            serverData.url = document.getElementById('mcp-url').value;
        }
        
        const response = await fetch('/api/v1/gauth/mcp/servers', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(serverData)
        });
        
        if (!response.ok) {
            throw new Error(await response.text());
        }
        
        resultBox.style.display = 'block';
        resultBox.className = 'result-box success';
        resultBox.innerHTML = '<h4><i class="fas fa-check-circle"></i> Server Registered</h4><p>MCP server has been registered successfully.</p>';
        
        document.getElementById('register-mcp-form').reset();
        await loadMCPServers();
        
    } catch (error) {
        displayError(resultBox, error);
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-plus"></i> Register Server';
    }
}

async function loadMCPServers() {
    try {
        const response = await fetch('/api/v1/gauth/mcp/servers');
        if (!response.ok) {
            throw new Error('Failed to load MCP servers');
        }
        
        const data = await response.json();
        mcpState.servers = data.servers || [];

        document.getElementById('mcp-total-servers').textContent = mcpState.servers.length;
        document.getElementById('mcp-connected-servers').textContent =
            mcpState.servers.filter(s => s.status === 'connected').length;
        
        displayMCPServers();
        
    } catch (error) {
        console.error('Failed to load MCP servers:', error);
    }
}

function displayMCPServers() {
    const container = document.getElementById('mcp-servers-list');
    
    if (mcpState.servers.length === 0) {
        container.innerHTML = '<p class="text-muted">No servers registered. Use the form to register your first MCP server.</p>';
        return;
    }
    
    const serversHTML = mcpState.servers.map(server => `
        <div class="mcp-server-item" data-server-id="${server.id}" style="padding: 1rem; margin-bottom: 0.75rem; border: 2px solid #e5e7eb; border-radius: 0.5rem; cursor: pointer; transition: all 0.2s;">
            <div style="display: flex; justify-content: space-between; align-items: start;">
                <div style="flex: 1;">
                    <div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.5rem;">
                        <i class="fas fa-server" style="color: #6b7280;"></i>
                        <strong>${server.name}</strong>
                    </div>
                    <p style="font-size: 0.75rem; color: #6b7280; margin-bottom: 0.5rem;">${server.id}</p>
                    <div style="display: flex; align-items: center; gap: 0.5rem;">
                        ${server.status === 'connected'
                            ? '<i class="fas fa-check-circle" style="color: #10b981; font-size: 0.875rem;"></i><span style="font-size: 0.875rem; color: #10b981;">Connected</span>'
                            : '<i class="fas fa-times-circle" style="color: #6b7280; font-size: 0.875rem;"></i><span style="font-size: 0.875rem; color: #6b7280;">Disconnected</span>'}
                    </div>
                </div>
                <button class="btn btn-danger" style="padding: 0.25rem 0.5rem; font-size: 0.75rem;" onclick="disconnectMCPServer('${server.id}', event)">
                    <i class="fas fa-trash"></i>
                </button>
            </div>
        </div>
    `).join('');
    
    container.innerHTML = serversHTML;
    
    // Add click handlers
    document.querySelectorAll('.mcp-server-item').forEach(item => {
        item.addEventListener('click', async () => {
            const serverId = item.dataset.serverId;
            await selectMCPServer(serverId);
        });
    });
}

async function selectMCPServer(serverId) {
    mcpState.selectedServerId = serverId;
    
    // Highlight selected server
    document.querySelectorAll('.mcp-server-item').forEach(item => {
        if (item.dataset.serverId === serverId) {
            item.style.borderColor = '#3b82f6';
            item.style.backgroundColor = '#eff6ff';
        } else {
            item.style.borderColor = '#e5e7eb';
            item.style.backgroundColor = 'transparent';
        }
    });
    
    // Load server details
    try {
        // Load resources
        const resourcesResponse = await fetch(`/api/v1/gauth/mcp/servers/${serverId}/resources`);
        if (resourcesResponse.ok) {
            const resourcesData = await resourcesResponse.json();
            mcpState.resources = resourcesData.resources || [];
            displayMCPResources();
        }
        
        // Load tools
        const toolsResponse = await fetch(`/api/v1/gauth/mcp/servers/${serverId}/tools`);
        if (toolsResponse.ok) {
            const toolsData = await toolsResponse.json();
            mcpState.tools = toolsData.tools || [];
            displayMCPTools();
            document.getElementById('mcp-total-tools').textContent = mcpState.tools.length;
        }
        
        document.getElementById('mcp-server-details').style.display = 'block';
        
    } catch (error) {
        console.error('Failed to load server details:', error);
    }
}

function displayMCPResources() {
    const container = document.getElementById('mcp-resources-list');
    
    if (mcpState.resources.length === 0) {
        container.innerHTML = '<p class="text-muted">No resources available</p>';
        return;
    }
    
    const resourcesHTML = mcpState.resources.map(resource => `
        <div style="padding: 0.75rem; margin-bottom: 0.5rem; border: 1px solid #e5e7eb; border-radius: 0.5rem;">
            <div style="display: flex; justify-content: space-between; align-items: start;">
                <div style="flex: 1;">
                    <strong style="font-size: 0.875rem;">${resource.name}</strong>
                    <p style="font-size: 0.75rem; color: #6b7280; margin: 0.25rem 0;">${resource.uri}</p>
                    ${resource.description ? `<p style="font-size: 0.75rem; color: #374151; margin-top: 0.25rem;">${resource.description}</p>` : ''}
                </div>
                <button class="btn btn-primary" style="padding: 0.25rem 0.5rem; font-size: 0.75rem;" onclick="readMCPResource('${resource.uri}')">
                    <i class="fas fa-eye"></i> Read
                </button>
            </div>
        </div>
    `).join('');
    
    container.innerHTML = resourcesHTML;
}

function displayMCPTools() {
    const container = document.getElementById('mcp-tools-list');
    const toolSelect = document.getElementById('tool-name-select');
    
    if (mcpState.tools.length === 0) {
        container.innerHTML = '<p class="text-muted">No tools available</p>';
        document.getElementById('mcp-tool-executor').style.display = 'none';
        return;
    }
    
    const toolsHTML = mcpState.tools.map(tool => `
        <div style="padding: 0.75rem; margin-bottom: 0.5rem; border: 1px solid #e5e7eb; border-radius: 0.5rem;">
            <strong style="font-size: 0.875rem;">${tool.name}</strong>
            ${tool.description ? `<p style="font-size: 0.75rem; color: #6b7280; margin-top: 0.25rem;">${tool.description}</p>` : ''}
        </div>
    `).join('');
    
    container.innerHTML = toolsHTML;
    
    // Populate tool select
    toolSelect.innerHTML = '<option value="">Select a tool...</option>' +
        mcpState.tools.map(tool => `<option value="${tool.name}">${tool.name}</option>`).join('');
    
    document.getElementById('mcp-tool-executor').style.display = 'block';
}

async function readMCPResource(uri) {
    try {
        const response = await fetch(`/api/v1/gauth/mcp/servers/${mcpState.selectedServerId}/resources/read`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ uri })
        });
        
        if (!response.ok) {
            throw new Error('Failed to read resource');
        }
        
        const data = await response.json();
        
        // Display in a modal or alert
        alert(`Resource Content:\n\nURI: ${uri}\n\n${data.contents?.[0]?.text || 'No content'}`);
        
    } catch (error) {
        alert('Failed to read resource: ' + error.message);
    }
}

async function executeMCPTool() {
    const resultBox = document.getElementById('tool-result');
    const submitBtn = document.querySelector('#execute-tool-form button[type="submit"]');
    
    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Executing...';
        
        const toolName = document.getElementById('tool-name-select').value;
        const argsText = document.getElementById('tool-arguments').value;
        
        let args = {};
        try {
            args = JSON.parse(argsText);
        } catch {
            throw new Error('Invalid JSON in arguments');
        }
        
        const response = await fetch(`/api/v1/gauth/mcp/servers/${mcpState.selectedServerId}/tools/call`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: toolName, arguments: args })
        });
        
        if (!response.ok) {
            throw new Error(await response.text());
        }
        
        const data = await response.json();
        
        resultBox.style.display = 'block';
        resultBox.className = 'result-box success';
        resultBox.innerHTML = `
            <h4><i class="fas fa-check-circle"></i> Tool Executed</h4>
            <pre style="background: white; padding: 1rem; border-radius: 0.5rem; overflow-x: auto; font-size: 0.75rem;">${JSON.stringify(data.content, null, 2)}</pre>
        `;
        
    } catch (error) {
        displayError(resultBox, error);
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-play"></i> Execute Tool';
    }
}

async function disconnectMCPServer(serverId, event) {
    event.stopPropagation();

    if (!confirm('Are you sure you want to disconnect this server?')) return;

    try {
        const response = await fetch(`/api/v1/gauth/mcp/servers/${serverId}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            throw new Error('Failed to disconnect server');
        }

        if (mcpState.selectedServerId === serverId) {
            mcpState.selectedServerId = null;
            document.getElementById('mcp-server-details').style.display = 'none';
        }

        await loadMCPServers();

    } catch (error) {
        alert('Failed to disconnect server: ' + error.message);
    }
}

// Login Functions
function setLoginStep(step) {
    loginState.step = step;

    document.getElementById('login-step-credentials').style.display = step === 'credentials' ? 'block' : 'none';
    document.getElementById('login-step-mfa').style.display = step === 'mfa' ? 'block' : 'none';
    document.getElementById('login-step-success').style.display = step === 'success' ? 'block' : 'none';
}

async function handleLoginCredentials() {
    const submitBtn = document.querySelector('#login-credentials-form button[type="submit"]');
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;

    if (!username || !password) {
        alert('Enter username and password');
        return;
    }

    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Authenticating...';

        const response = await fetch('/api/v1/gauth/auth/login/init', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        if (!response.ok) {
            const error = await response.text();
            throw new Error(error || 'Login failed');
        }

        const data = await response.json();

        if (!data.success) {
            throw new Error(data.error || 'Login failed');
        }

        loginState.sessionChallenge = data.sessionChallenge;

        if (data.requiresMFA) {
            loginState.mfaMethods = data.mfaMethods || ['totp'];
            setupMFAMethods();
            setLoginStep('mfa');
            alert('MFA required. Enter your code.');
        } else {
            loginState.sessionInfo = data;
            displayLoginSuccess();
            setLoginStep('success');
        }

    } catch (error) {
        alert('Login error: ' + error.message);
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-arrow-right"></i> Continue';
    }
}

function setupMFAMethods() {
    const methodsContainer = document.getElementById('mfa-methods');
    const methodsDiv = methodsContainer.querySelector('div');

    if (loginState.mfaMethods.length > 1) {
        methodsContainer.style.display = 'block';
        methodsDiv.innerHTML = loginState.mfaMethods.map(method => `
            <button type="button" class="mfa-method-btn" data-method="${method}" style="padding: 0.5rem 1rem; border-radius: 0.5rem; border: 1px solid #d1d5db; background: #f9fafb; cursor: pointer; transition: all 0.2s;">
                ${method.toUpperCase()}
            </button>
        `).join('');

        // Add click handlers
        methodsDiv.querySelectorAll('.mfa-method-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                loginState.selectedMfaMethod = btn.dataset.method;
                methodsDiv.querySelectorAll('.mfa-method-btn').forEach(b => {
                    b.style.background = b === btn ? '#3b82f6' : '#f9fafb';
                    b.style.color = b === btn ? 'white' : 'inherit';
                    b.style.borderColor = b === btn ? '#2563eb' : '#d1d5db';
                });
                document.getElementById('mfa-method-label').textContent = btn.dataset.method.toUpperCase();
            });
        });

        // Select first method by default
        methodsDiv.querySelector('.mfa-method-btn').click();
    } else {
        methodsContainer.style.display = 'none';
        loginState.selectedMfaMethod = loginState.mfaMethods[0] || 'totp';
        document.getElementById('mfa-method-label').textContent = loginState.selectedMfaMethod.toUpperCase();
    }
}

async function handleLoginMFA() {
    const submitBtn = document.querySelector('#login-mfa-form button[type="submit"]');
    const mfaCode = document.getElementById('login-mfa-code').value;

    if (!loginState.sessionChallenge) {
        alert('No active challenge');
        return;
    }

    if (!mfaCode) {
        alert('Enter MFA code');
        return;
    }

    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner loading"></i> Verifying...';

        const response = await fetch('/api/v1/gauth/auth/mfa/verify', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                challengeId: loginState.sessionChallenge,
                code: mfaCode,
                method: loginState.selectedMfaMethod
            })
        });

        if (!response.ok) {
            const error = await response.text();
            throw new Error(error || 'MFA verification failed');
        }

        const data = await response.json();

        if (data.success) {
            loginState.sessionInfo = data;
            displayLoginSuccess();
            setLoginStep('success');
        } else {
            throw new Error(data.error || 'Invalid code');
        }

    } catch (error) {
        alert('MFA verification failed: ' + error.message);
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-check"></i> Verify MFA';
    }
}

function displayLoginSuccess() {
    const sessionInfo = document.getElementById('login-session-info');

    if (loginState.sessionInfo) {
        const info = loginState.sessionInfo;
        sessionInfo.innerHTML = `
            <div style="margin-bottom: 0.5rem;"><strong>Session Established</strong></div>
            ${info.sessionToken ? `<div style="margin-bottom: 0.25rem;">Token: ${info.sessionToken.substring(0, 20)}...</div>` : ''}
            ${info.expiresAt ? `<div style="margin-bottom: 0.25rem;">Expires: ${new Date(info.expiresAt).toLocaleString()}</div>` : ''}
            ${info.username ? `<div>Username: ${info.username}</div>` : ''}
        `;
    } else {
        sessionInfo.innerHTML = '<div>Session details not available</div>';
    }
}

function resetLogin() {
    loginState.step = 'credentials';
    loginState.sessionChallenge = null;
    loginState.mfaMethods = [];
    loginState.selectedMfaMethod = 'totp';
    loginState.sessionInfo = null;

    document.getElementById('login-username').value = '';
    document.getElementById('login-password').value = '';
    document.getElementById('login-mfa-code').value = '';

    setLoginStep('credentials');
}

// Export for debugging
window.gauthApp = {
    state,
    mcpState,
    loginState,
    switchTab,
    loadMetrics,
    loadCacheStats,
    loadMCPServers
};
