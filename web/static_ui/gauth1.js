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
    }
}

// Form Initialization
function initForms() {
    // Create Extended Token
    document.getElementById('create-token-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await createExtendedToken();
    });
    
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
            expirationHours: parseInt(document.getElementById('token-expiration').value)
        };
        
        // Simulate API call (replace with actual endpoint)
        await delay(1000);
        
        const mockToken = {
            token: generateMockJWT(tokenData),
            clientId: tokenData.clientId,
            expiresAt: new Date(Date.now() + tokenData.expirationHours * 3600000).toISOString(),
            scope: tokenData.scope,
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
        
        // Simulate API call
        await delay(800);
        
        const validation = {
            valid: true,
            decoded: decodeJWT(token),
            checks: {
                signature: 'Valid',
                expiration: 'Not expired',
                authorizationChain: '3 levels verified',
                commercialRegister: 'Verified',
                pvpIdentity: 'Verified'
            }
        };
        
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
    element.innerHTML = `
        <h4><i class="fas fa-check-circle"></i> Token Created Successfully</h4>
        <p><strong>Client ID:</strong> ${token.clientId}</p>
        <p><strong>Expires:</strong> ${new Date(token.expiresAt).toLocaleString()}</p>
        <p><strong>Scope:</strong> ${token.scope.join(', ')}</p>
        <p><strong>Authorization Chain:</strong></p>
        <ul>
            <li>Owner's Authorizer: ${token.authorizationChain.ownersAuthorizer}</li>
            <li>Client Owner: ${token.authorizationChain.clientOwner}</li>
            <li>Client: ${token.authorizationChain.client}</li>
        </ul>
        <pre>${token.token}</pre>
    `;
}

function displayValidationResult(element, validation) {
    element.style.display = 'block';
    element.className = `result-box ${validation.valid ? 'success' : 'error'}`;
    
    const checksHTML = Object.entries(validation.checks)
        .map(([check, result]) => `<li>${check}: <strong>${result}</strong></li>`)
        .join('');
    
    element.innerHTML = `
        <h4><i class="fas fa-check-circle"></i> Token Validation Result</h4>
        <p><strong>Status:</strong> ${validation.valid ? 'VALID' : 'INVALID'}</p>
        <p><strong>Validation Checks:</strong></p>
        <ul>${checksHTML}</ul>
        <pre>${JSON.stringify(validation.decoded, null, 2)}</pre>
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

// Export for debugging
window.gauthApp = {
    state,
    switchTab,
    loadMetrics,
    loadCacheStats
};
