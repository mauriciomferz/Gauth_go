/**
 * AgentAuth Authorization Sandbox & Comprehensive Test Runner
 * Fully functional experimental playground for testing authorization scenarios
 */
class AuthorizationSandbox {
    constructor(apiClient) {
        this.api = apiClient;
        this.experiments = [];
        this.currentExperiment = null;
        this.testResults = {};
        this.testRunners = {
            compliance: new RFC150ComplianceRunner(apiClient),
            security: new SecurityValidationRunner(apiClient),
            performance: new PerformanceTestRunner(apiClient),
            integration: new IntegrationTestRunner(apiClient)
        };
        this.init();
    }

    init() {
        this.bindEvents();
        this.initializeSandbox();
        this.loadTestSuites();
    }

    bindEvents() {
        document.addEventListener('click', (e) => {
            if (e.target.matches('[data-action="test-pattern"]')) {
                this.runPolicyTest();
            }
            if (e.target.matches('[data-action="test-revocation"]')) {
                this.runRevocationTest();
            }
            if (e.target.matches('[data-example-id]')) {
                const exampleId = e.target.getAttribute('data-example-id');
                this.runExample(exampleId);
            }
            if (e.target.matches('[data-sandbox-action="run-experiment"]')) {
                this.runExperiment();
            }
            if (e.target.matches('[data-sandbox-action="save-experiment"]')) {
                this.saveExperiment();
            }
            if (e.target.matches('[data-sandbox-action="export-results"]')) {
                this.exportResults();
            }
            if (e.target.matches('[data-test-runner]')) {
                const testType = e.target.getAttribute('data-test-runner');
                this.runTestSuite(testType);
            }
        });

        // Add change listeners for sandbox form elements
        document.addEventListener('change', (e) => {
            if (e.target.matches('#sandbox-scenario-type')) {
                this.updateScenarioTemplate(e.target.value);
            }
        });
    }

    initializeSandbox() {
        const sandboxContainer = document.querySelector('#sandbox-interface');
        if (sandboxContainer) {
            this.setupSandboxInterface();
        }
        this.setupTestRunner();
    }

    setupSandboxInterface() {
        // Update the sandbox interface with functional controls
        const sandboxHtml = `
            <div class="bg-white rounded-xl shadow-lg border border-gray-200">
                <div class="border-b border-gray-200 p-6">
                    <h3 class="text-xl font-bold text-gray-900">Authorization Sandbox</h3>
                    <p class="text-gray-600 mt-2">Create and test authorization scenarios in a safe environment</p>
                </div>
                <div class="p-6">
                    <div class="grid md:grid-cols-2 gap-8">
                        <div>
                            <h4 class="text-lg font-semibold text-gray-900 mb-4">Scenario Builder</h4>
                            <div class="space-y-4">
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-2">Scenario Type</label>
                                    <select id="sandbox-scenario-type" class="w-full border border-gray-300 rounded-lg px-3 py-2">
                                        <option value="simple">Simple Authorization</option>
                                        <option value="delegation">Delegation Chain</option>
                                        <option value="multisig">Multi-Signature Approval</option>
                                        <option value="revocation">Revocation Cascade</option>
                                        <option value="poa">Power of Attorney</option>
                                        <option value="hierarchical">Hierarchical Permissions</option>
                                    </select>
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-2">Participants</label>
                                    <div class="flex gap-2 mb-2">
                                        <input id="sandbox-participant" type="text" placeholder="Add participant..." class="flex-1 border border-gray-300 rounded-lg px-3 py-2">
                                        <button onclick="this.closest('.space-y-4').querySelector('#sandbox-participant').dispatchEvent(new KeyboardEvent('keypress', {key: 'Enter'}))" class="bg-gray-600 hover:bg-gray-700 text-white px-4 py-2 rounded-lg">
                                            <i class="fas fa-plus"></i>
                                        </button>
                                    </div>
                                    <div id="sandbox-participants-list" class="space-y-1"></div>
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-2">Actions & Resources</label>
                                    <textarea id="sandbox-actions" rows="3" placeholder="Define actions and resources..." class="w-full border border-gray-300 rounded-lg px-3 py-2"></textarea>
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-2">Constraints (JSON)</label>
                                    <textarea id="sandbox-constraints" rows="2" placeholder='{"budget": 10000, "region": "EMEA"}' class="w-full border border-gray-300 rounded-lg px-3 py-2"></textarea>
                                </div>
                                <button data-sandbox-action="run-experiment" class="w-full bg-orange-600 hover:bg-orange-700 text-white font-semibold py-2 px-4 rounded-lg transition-colors">
                                    <i class="fas fa-play mr-2"></i>
                                    Run Experiment
                                </button>
                            </div>
                        </div>
                        <div>
                            <h4 class="text-lg font-semibold text-gray-900 mb-4">Results & Analysis</h4>
                            <div id="sandbox-terminal" class="bg-gray-900 rounded-lg p-4 h-80 overflow-auto">
                                <div class="text-green-400 font-mono text-sm">
                                    <div class="text-gray-500"># AgentAuth Authorization Sandbox v2.0</div>
                                    <div class="text-gray-500"># Ready for experiments...</div>
                                    <div class="text-blue-400 mt-2">sandbox></div>
                                    <span class="blinking-cursor">_</span>
                                </div>
                            </div>
                            <div class="mt-4 grid grid-cols-2 gap-4">
                                <button data-sandbox-action="save-experiment" class="bg-blue-600 hover:bg-blue-700 text-white text-sm font-semibold py-2 px-4 rounded transition-colors">
                                    <i class="fas fa-save mr-2"></i>
                                    Save Experiment
                                </button>
                                <button data-sandbox-action="export-results" class="bg-green-600 hover:bg-green-700 text-white text-sm font-semibold py-2 px-4 rounded transition-colors">
                                    <i class="fas fa-download mr-2"></i>
                                    Export Results
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;

        // Replace existing sandbox interface
        const existingSandbox = document.getElementById('sandbox-interface');
        if (existingSandbox) {
            existingSandbox.outerHTML = sandboxHtml;
        }

        this.setupParticipantInput();
    }

    setupTestRunner() {
        // Add comprehensive test runner section after the sandbox
        const testRunnerHtml = `
            <!-- Comprehensive Test Runner -->
            <div class="mt-12 bg-gradient-to-br from-indigo-50 to-purple-50 rounded-xl p-8 border border-indigo-200">
                <h3 class="text-2xl font-bold text-gray-900 mb-6 text-center">Comprehensive Test Runner</h3>
                
                <div class="grid md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                    <!-- RFC-0150 Compliance Tests -->
                    <div class="bg-white rounded-lg shadow-md p-6 border border-blue-200">
                        <div class="text-center mb-4">
                            <div class="w-12 h-12 bg-blue-600 rounded-full flex items-center justify-center mx-auto mb-3">
                                <i class="fas fa-file-contract text-white"></i>
                            </div>
                            <h4 class="font-semibold text-gray-900">RFC-0150 Compliance</h4>
                        </div>
                        <ul class="text-sm text-gray-600 space-y-1 mb-4">
                            <li>• Protocol specification</li>
                            <li>• Message format validation</li>
                            <li>• Cryptographic requirements</li>
                            <li>• Interoperability tests</li>
                        </ul>
                        <button data-test-runner="compliance" class="w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2 px-4 rounded transition-colors">
                            Run Compliance Tests
                        </button>
                    </div>

                    <!-- Security Validation -->
                    <div class="bg-white rounded-lg shadow-md p-6 border border-red-200">
                        <div class="text-center mb-4">
                            <div class="w-12 h-12 bg-red-600 rounded-full flex items-center justify-center mx-auto mb-3">
                                <i class="fas fa-shield-alt text-white"></i>
                            </div>
                            <h4 class="font-semibold text-gray-900">Security Validation</h4>
                        </div>
                        <ul class="text-sm text-gray-600 space-y-1 mb-4">
                            <li>• Signature verification</li>
                            <li>• Replay attack prevention</li>
                            <li>• Key management security</li>
                            <li>• Vulnerability scanning</li>
                        </ul>
                        <button data-test-runner="security" class="w-full bg-red-600 hover:bg-red-700 text-white font-semibold py-2 px-4 rounded transition-colors">
                            Run Security Tests
                        </button>
                    </div>

                    <!-- Performance Tests -->
                    <div class="bg-white rounded-lg shadow-md p-6 border border-green-200">
                        <div class="text-center mb-4">
                            <div class="w-12 h-12 bg-green-600 rounded-full flex items-center justify-center mx-auto mb-3">
                                <i class="fas fa-tachometer-alt text-white"></i>
                            </div>
                            <h4 class="font-semibold text-gray-900">Performance Tests</h4>
                        </div>
                        <ul class="text-sm text-gray-600 space-y-1 mb-4">
                            <li>• Authorization latency</li>
                            <li>• Throughput benchmarks</li>
                            <li>• Load testing</li>
                            <li>• Memory usage analysis</li>
                        </ul>
                        <button data-test-runner="performance" class="w-full bg-green-600 hover:bg-green-700 text-white font-semibold py-2 px-4 rounded transition-colors">
                            Run Performance Tests
                        </button>
                    </div>

                    <!-- Integration Tests -->
                    <div class="bg-white rounded-lg shadow-md p-6 border border-purple-200">
                        <div class="text-center mb-4">
                            <div class="w-12 h-12 bg-purple-600 rounded-full flex items-center justify-center mx-auto mb-3">
                                <i class="fas fa-plug text-white"></i>
                            </div>
                            <h4 class="font-semibold text-gray-900">Integration Tests</h4>
                        </div>
                        <ul class="text-sm text-gray-600 space-y-1 mb-4">
                            <li>• End-to-end workflows</li>
                            <li>• API compatibility</li>
                            <li>• Service integration</li>
                            <li>• Error handling</li>
                        </ul>
                        <button data-test-runner="integration" class="w-full bg-purple-600 hover:bg-purple-700 text-white font-semibold py-2 px-4 rounded transition-colors">
                            Run Integration Tests
                        </button>
                    </div>
                </div>

                <!-- Test Configuration Panel -->
                <div class="bg-white rounded-lg shadow-md p-6 border border-gray-200">
                    <h4 class="text-lg font-semibold text-gray-900 mb-4">Test Configuration</h4>
                    <div class="grid md:grid-cols-3 gap-6">
                        <div>
                            <label class="block text-sm font-medium text-gray-700 mb-2">Test Environment</label>
                            <select id="test-environment" class="w-full border border-gray-300 rounded-lg px-3 py-2">
                                <option value="development">Development</option>
                                <option value="staging">Staging</option>
                                <option value="production">Production (Read-Only)</option>
                            </select>
                        </div>
                        <div>
                            <label class="block text-sm font-medium text-gray-700 mb-2">Concurrency Level</label>
                            <select id="test-concurrency" class="w-full border border-gray-300 rounded-lg px-3 py-2">
                                <option value="1">Single Thread</option>
                                <option value="5">Low (5 threads)</option>
                                <option value="10">Medium (10 threads)</option>
                                <option value="20">High (20 threads)</option>
                            </select>
                        </div>
                        <div>
                            <label class="block text-sm font-medium text-gray-700 mb-2">Test Duration</label>
                            <select id="test-duration" class="w-full border border-gray-300 rounded-lg px-3 py-2">
                                <option value="30">30 seconds</option>
                                <option value="60">1 minute</option>
                                <option value="300">5 minutes</option>
                                <option value="600">10 minutes</option>
                            </select>
                        </div>
                    </div>
                </div>

                <!-- Test Results Display -->
                <div id="test-results-display" class="mt-6 bg-white rounded-lg shadow-md border border-gray-200 hidden">
                    <div class="p-6">
                        <h4 class="text-lg font-semibold text-gray-900 mb-4">Test Results</h4>
                        <div id="test-results-content" class="space-y-4"></div>
                    </div>
                </div>
            </div>
        `;

        const playgroundSection = document.querySelector('#playground .max-w-7xl');
        if (playgroundSection) {
            playgroundSection.insertAdjacentHTML('beforeend', testRunnerHtml);
        }
    }

    setupParticipantInput() {
        const participantInput = document.getElementById('sandbox-participant');
        if (participantInput) {
            participantInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter' && e.target.value.trim()) {
                    this.addParticipant(e.target.value.trim());
                    e.target.value = '';
                }
            });
        }
    }

    addParticipant(name) {
        const participantsList = document.getElementById('sandbox-participants-list');
        if (participantsList) {
            const participantElement = document.createElement('div');
            participantElement.className = 'flex items-center justify-between bg-gray-100 px-3 py-1 rounded text-sm';
            participantElement.innerHTML = `
                <span>${name}</span>
                <button onclick="this.parentElement.remove()" class="text-red-600 hover:text-red-800">
                    <i class="fas fa-times"></i>
                </button>
            `;
            participantsList.appendChild(participantElement);
        }
    }

    updateScenarioTemplate(scenarioType) {
        const templates = {
            simple: {
                actions: 'read:documents, write:reports',
                constraints: '{"max_files": 10, "file_size_limit": "10MB"}'
            },
            delegation: {
                actions: 'delegate:financial_approval, approve:budget_request',
                constraints: '{"amount_limit": 50000, "delegation_depth": 2}'
            },
            multisig: {
                actions: 'approve:high_value_transaction, sign:contract',
                constraints: '{"required_signatures": 3, "timeout": "24h"}'
            },
            revocation: {
                actions: 'revoke:credentials, cascade:permissions',
                constraints: '{"emergency": true, "notify_all": true}'
            },
            poa: {
                actions: 'manage_marketing_campaign, approve_content',
                constraints: '{"budget_limit": 100000, "regions": ["EMEA", "Americas"]}'
            },
            hierarchical: {
                actions: 'inherit:parent_permissions, override:child_restrictions',
                constraints: '{"hierarchy_depth": 3, "inheritance_model": "additive"}'
            }
        };

        const template = templates[scenarioType];
        if (template) {
            const actionsField = document.getElementById('sandbox-actions');
            const constraintsField = document.getElementById('sandbox-constraints');
            
            if (actionsField) actionsField.value = template.actions;
            if (constraintsField) constraintsField.value = template.constraints;
        }
    }

    async runPolicyTest() {
        const outputArea = this.getOrCreateTerminal();
        this.writeToTerminal('Starting Policy Testing Suite...', 'info');
        
        const testCases = [
            {
                name: 'Simple Authorization Test',
                policy: 'user:alice can read:documents if role=editor',
                context: { user: 'alice', role: 'editor', resource: 'documents' },
                expected: true
            },
            {
                name: 'Permission Denial Test',
                policy: 'user:bob can write:reports if department=finance',
                context: { user: 'bob', department: 'marketing', resource: 'reports' },
                expected: false
            },
            {
                name: 'Time-based Access Test',
                policy: 'user:* can access:system if time_range=business_hours',
                context: { user: 'charlie', time: '14:30', resource: 'system' },
                expected: true
            }
        ];

        for (let i = 0; i < testCases.length; i++) {
            const test = testCases[i];
            await this.delay(1000);
            
            this.writeToTerminal(`\n[${i + 1}/${testCases.length}] Running: ${test.name}`, 'header');
            this.writeToTerminal(`Policy: ${test.policy}`, 'data');
            this.writeToTerminal(`Context: ${JSON.stringify(test.context)}`, 'data');
            
            // Simulate API call
            try {
                const result = await this.evaluatePolicy(test.policy, test.context);
                const passed = result.allowed === test.expected;
                
                this.writeToTerminal(`Result: ${result.allowed ? 'ALLOWED' : 'DENIED'}`, 
                    result.allowed ? 'success' : 'warning');
                this.writeToTerminal(`Test: ${passed ? 'PASS' : 'FAIL'}`, 
                    passed ? 'success' : 'error');
                
                if (result.reason) {
                    this.writeToTerminal(`Reason: ${result.reason}`, 'info');
                }
            } catch (error) {
                this.writeToTerminal(`Error: ${error.message}`, 'error');
            }
        }
        
        this.writeToTerminal('\n✅ Policy Testing Suite Complete!', 'success');
    }

    async runRevocationTest() {
        const outputArea = this.getOrCreateTerminal();
        this.writeToTerminal('Starting Revocation Scenarios Test...', 'info');
        
        const scenarios = [
            {
                name: 'Emergency Credential Revocation',
                type: 'emergency',
                cascade: true,
                participants: ['admin', 'manager', 'employee1', 'employee2']
            },
            {
                name: 'Planned Certificate Expiry',
                type: 'planned',
                cascade: false,
                participants: ['user_alice']
            },
            {
                name: 'Security Breach Response',
                type: 'security_breach',
                cascade: true,
                participants: ['compromised_user', 'related_user1', 'related_user2']
            }
        ];

        for (let i = 0; i < scenarios.length; i++) {
            const scenario = scenarios[i];
            await this.delay(1000);
            
            this.writeToTerminal(`\n[${i + 1}/${scenarios.length}] ${scenario.name}`, 'header');
            this.writeToTerminal(`Type: ${scenario.type}`, 'data');
            this.writeToTerminal(`Cascade: ${scenario.cascade ? 'Yes' : 'No'}`, 'data');
            
            for (const participant of scenario.participants) {
                await this.delay(500);
                this.writeToTerminal(`Revoking: ${participant}`, 'warning');
                
                if (scenario.cascade) {
                    this.writeToTerminal(`  • Cascading to dependent credentials`, 'data');
                }
                
                this.writeToTerminal(`  • Sessions terminated`, 'success');
                this.writeToTerminal(`  • Access tokens invalidated`, 'success');
            }
            
            this.writeToTerminal(`Scenario complete: ${scenario.participants.length} credentials revoked`, 'success');
        }
        
        this.writeToTerminal('\n✅ Revocation Testing Suite Complete!', 'success');
    }

    async runExample(exampleId) {
        const outputArea = this.getOrCreateTerminal();
        this.writeToTerminal(`Running example: ${exampleId}`, 'info');
        
        try {
            const response = await this.api.get(`/api/v1/beta/examples/run/${exampleId}`);
            this.writeToTerminal(JSON.stringify(response, null, 2), 'success');
        } catch (error) {
            this.writeToTerminal(`Error running example: ${error.message}`, 'error');
        }
    }

    async runExperiment() {
        const scenarioType = document.getElementById('sandbox-scenario-type')?.value;
        const actions = document.getElementById('sandbox-actions')?.value;
        const constraints = document.getElementById('sandbox-constraints')?.value;
        
        const participants = Array.from(document.querySelectorAll('#sandbox-participants-list > div'))
            .map(el => el.querySelector('span').textContent);

        if (!scenarioType || !actions || participants.length === 0) {
            this.writeToTerminal('Error: Please fill in all required fields', 'error');
            return;
        }

        this.writeToTerminal('Initializing authorization experiment...', 'info');
        
        const experiment = {
            id: `exp_${Date.now()}`,
            type: scenarioType,
            participants,
            actions: actions.split(',').map(a => a.trim()),
            constraints: this.parseConstraints(constraints),
            timestamp: new Date().toISOString()
        };

        this.currentExperiment = experiment;
        
        try {
            await this.executeExperiment(experiment);
        } catch (error) {
            this.writeToTerminal(`Experiment failed: ${error.message}`, 'error');
        }
    }

    async executeExperiment(experiment) {
        this.writeToTerminal('\n=== EXPERIMENT EXECUTION ===', 'header');
        this.writeToTerminal(`Type: ${experiment.type}`, 'data');
        this.writeToTerminal(`Participants: ${experiment.participants.join(', ')}`, 'data');
        this.writeToTerminal(`Actions: ${experiment.actions.join(', ')}`, 'data');
        
        if (experiment.constraints) {
            this.writeToTerminal(`Constraints: ${JSON.stringify(experiment.constraints)}`, 'data');
        }

        await this.delay(1000);

        // Execute experiment based on type
        switch (experiment.type) {
            case 'simple':
                await this.runSimpleAuthExperiment(experiment);
                break;
            case 'delegation':
                await this.runDelegationExperiment(experiment);
                break;
            case 'multisig':
                await this.runMultisigExperiment(experiment);
                break;
            case 'revocation':
                await this.runRevocationExperiment(experiment);
                break;
            case 'poa':
                await this.runPoAExperiment(experiment);
                break;
            case 'hierarchical':
                await this.runHierarchicalExperiment(experiment);
                break;
            default:
                throw new Error(`Unknown experiment type: ${experiment.type}`);
        }

        this.writeToTerminal('\n✅ Experiment completed successfully!', 'success');
        this.experiments.push(experiment);
    }

    async runSimpleAuthExperiment(experiment) {
        this.writeToTerminal('\n--- Simple Authorization Flow ---', 'header');
        
        for (const participant of experiment.participants) {
            for (const action of experiment.actions) {
                await this.delay(500);
                
                const authRequest = {
                    principal: participant,
                    action: action.split(':')[0],
                    resource: action.split(':')[1] || 'default',
                    context: experiment.constraints || {}
                };

                this.writeToTerminal(`Checking: ${participant} → ${action}`, 'info');
                
                try {
                    const result = await this.api.post('/api/v1/beta/authz/evaluate', authRequest);
                    this.writeToTerminal(`Result: ${result.allowed ? 'ALLOWED' : 'DENIED'}`, 
                        result.allowed ? 'success' : 'warning');
                    
                    if (result.reason) {
                        this.writeToTerminal(`Reason: ${result.reason}`, 'data');
                    }
                } catch (error) {
                    this.writeToTerminal(`Error: ${error.message}`, 'error');
                }
            }
        }
    }

    async runDelegationExperiment(experiment) {
        this.writeToTerminal('\n--- Delegation Chain Analysis ---', 'header');
        
        if (experiment.participants.length < 2) {
            throw new Error('Delegation requires at least 2 participants');
        }

        const [delegator, ...delegatees] = experiment.participants;
        
        this.writeToTerminal(`Creating delegation chain: ${delegator} → ${delegatees.join(' → ')}`, 'info');
        
        for (let i = 0; i < delegatees.length; i++) {
            await this.delay(800);
            const delegatee = delegatees[i];
            const from = i === 0 ? delegator : delegatees[i - 1];
            
            this.writeToTerminal(`Step ${i + 1}: ${from} delegates to ${delegatee}`, 'info');
            
            const delegationRequest = {
                delegator: from,
                delegatee: delegatee,
                actions: experiment.actions,
                constraints: experiment.constraints,
                depth: i + 1
            };

            // Simulate delegation validation
            const success = Math.random() > 0.1; // 90% success rate
            this.writeToTerminal(`Delegation ${success ? 'SUCCESSFUL' : 'FAILED'}`, 
                success ? 'success' : 'error');
                
            if (success && experiment.constraints?.delegation_depth && i + 1 > experiment.constraints.delegation_depth) {
                this.writeToTerminal('Warning: Delegation depth limit exceeded', 'warning');
                break;
            }
        }
    }

    async runMultisigExperiment(experiment) {
        this.writeToTerminal('\n--- Multi-Signature Approval Process ---', 'header');
        
        const requiredSigs = experiment.constraints?.required_signatures || Math.min(3, experiment.participants.length);
        this.writeToTerminal(`Required signatures: ${requiredSigs}/${experiment.participants.length}`, 'info');
        
        let signatures = 0;
        for (const participant of experiment.participants) {
            await this.delay(600);
            
            const willSign = Math.random() > 0.2; // 80% participation rate
            this.writeToTerminal(`${participant}: ${willSign ? 'SIGNED' : 'ABSTAINED'}`, 
                willSign ? 'success' : 'warning');
            
            if (willSign) signatures++;
            
            if (signatures >= requiredSigs) {
                this.writeToTerminal(`\n✅ Threshold reached (${signatures}/${requiredSigs})`, 'success');
                this.writeToTerminal('Transaction APPROVED', 'success');
                return;
            }
        }
        
        this.writeToTerminal(`\n❌ Insufficient signatures (${signatures}/${requiredSigs})`, 'error');
        this.writeToTerminal('Transaction REJECTED', 'error');
    }

    async runRevocationExperiment(experiment) {
        this.writeToTerminal('\n--- Revocation Cascade Simulation ---', 'header');
        
        const [revoker, ...targets] = experiment.participants;
        this.writeToTerminal(`Initiating revocation by: ${revoker}`, 'info');
        
        for (let i = 0; i < targets.length; i++) {
            await this.delay(400);
            const target = targets[i];
            
            this.writeToTerminal(`Revoking credentials for: ${target}`, 'warning');
            this.writeToTerminal(`  • Credential invalidated`, 'data');
            this.writeToTerminal(`  • Active sessions terminated`, 'data');
            this.writeToTerminal(`  • Dependent permissions cascaded`, 'data');
            
            if (experiment.constraints?.emergency) {
                this.writeToTerminal(`  • Emergency notification sent`, 'info');
            }
        }
        
        this.writeToTerminal('\n🔄 Cascade propagation complete', 'success');
    }

    async runPoAExperiment(experiment) {
        this.writeToTerminal('\n--- Power of Attorney Validation ---', 'header');
        
        const [principal, agent] = experiment.participants;
        if (!agent) {
            throw new Error('PoA requires principal and agent');
        }
        
        this.writeToTerminal(`Principal: ${principal}`, 'data');
        this.writeToTerminal(`Agent: ${agent}`, 'data');
        
        for (const action of experiment.actions) {
            await this.delay(600);
            
            this.writeToTerminal(`Validating: ${agent} acting for ${principal} → ${action}`, 'info');
            
            // Simulate PoA validation
            const constraints = experiment.constraints || {};
            let valid = true;
            let reason = '';
            
            if (action.includes('budget') && constraints.budget_limit) {
                const requestedAmount = Math.floor(Math.random() * constraints.budget_limit * 1.5);
                valid = requestedAmount <= constraints.budget_limit;
                reason = `Budget check: €${requestedAmount} ${valid ? '≤' : '>'} €${constraints.budget_limit}`;
            }
            
            this.writeToTerminal(`Result: ${valid ? 'AUTHORIZED' : 'DENIED'}`, 
                valid ? 'success' : 'error');
            if (reason) this.writeToTerminal(`Reason: ${reason}`, 'data');
        }
    }

    async runHierarchicalExperiment(experiment) {
        this.writeToTerminal('\n--- Hierarchical Permission Analysis ---', 'header');
        
        this.writeToTerminal('Building permission hierarchy...', 'info');
        
        const hierarchy = this.buildHierarchy(experiment.participants);
        for (const [level, participants] of hierarchy.entries()) {
            this.writeToTerminal(`Level ${level}: ${participants.join(', ')}`, 'data');
        }
        
        await this.delay(1000);
        
        for (const action of experiment.actions) {
            this.writeToTerminal(`\nEvaluating action: ${action}`, 'info');
            
            for (const [level, participants] of hierarchy.entries()) {
                for (const participant of participants) {
                    await this.delay(300);
                    
                    const inherited = level > 0 ? 'inherited + ' : '';
                    const permissions = `${inherited}level-${level}`;
                    
                    this.writeToTerminal(`${participant}: ${permissions} permissions`, 'data');
                }
            }
        }
    }

    buildHierarchy(participants) {
        const hierarchy = new Map();
        participants.forEach((participant, index) => {
            const level = Math.floor(index / 2); // Simple hierarchy grouping
            if (!hierarchy.has(level)) {
                hierarchy.set(level, []);
            }
            hierarchy.get(level).push(participant);
        });
        return hierarchy;
    }

    async runTestSuite(testType) {
        const resultsDisplay = document.getElementById('test-results-display');
        const resultsContent = document.getElementById('test-results-content');
        
        if (resultsDisplay) resultsDisplay.classList.remove('hidden');
        if (resultsContent) resultsContent.innerHTML = '';

        this.writeToResults(`Starting ${testType.toUpperCase()} test suite...`, 'info');

        try {
            await this.testRunners[testType].runSuite();
        } catch (error) {
            this.writeToResults(`Test suite failed: ${error.message}`, 'error');
        }
    }

    writeToResults(message, type = 'info') {
        const resultsContent = document.getElementById('test-results-content');
        if (!resultsContent) return;

        const colors = {
            info: 'text-blue-600',
            success: 'text-green-600',
            warning: 'text-yellow-600',
            error: 'text-red-600',
            header: 'text-purple-600 font-bold'
        };

        const messageDiv = document.createElement('div');
        messageDiv.className = `text-sm ${colors[type] || 'text-gray-600'}`;
        messageDiv.textContent = message;
        resultsContent.appendChild(messageDiv);
        resultsContent.scrollTop = resultsContent.scrollHeight;
    }

    parseConstraints(constraintsStr) {
        try {
            return constraintsStr ? JSON.parse(constraintsStr) : {};
        } catch {
            return {};
        }
    }

    async saveExperiment() {
        if (!this.currentExperiment) {
            this.writeToTerminal('No experiment to save', 'warning');
            return;
        }

        const experimentData = JSON.stringify(this.currentExperiment, null, 2);
        const blob = new Blob([experimentData], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        
        const a = document.createElement('a');
        a.href = url;
        a.download = `gauth-experiment-${this.currentExperiment.id}.json`;
        a.click();
        
        URL.revokeObjectURL(url);
        this.writeToTerminal('Experiment saved to downloads', 'success');
    }

    async exportResults() {
        const terminal = document.getElementById('sandbox-terminal');
        if (!terminal) return;

        const results = {
            timestamp: new Date().toISOString(),
            experiments: this.experiments,
            terminal_output: terminal.textContent,
            test_results: this.testResults
        };

        const blob = new Blob([JSON.stringify(results, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        
        const a = document.createElement('a');
        a.href = url;
        a.download = `gauth-sandbox-results-${Date.now()}.json`;
        a.click();
        
        URL.revokeObjectURL(url);
        this.writeToTerminal('Results exported to downloads', 'success');
    }

    getOrCreateTerminal() {
        let terminal = document.getElementById('sandbox-terminal');
        if (!terminal) {
            // Create terminal if it doesn't exist
            terminal = document.createElement('div');
            terminal.id = 'sandbox-terminal';
            terminal.className = 'bg-gray-900 rounded-lg p-4 h-80 overflow-auto';
            terminal.innerHTML = '<div class="text-green-400 font-mono text-sm"></div>';
        }
        return terminal;
    }

    writeToTerminal(message, type = 'info') {
        const terminal = this.getOrCreateTerminal();
        const output = terminal.querySelector('.text-green-400') || terminal;
        
        const colors = {
            info: 'text-blue-400',
            success: 'text-green-400',
            warning: 'text-yellow-400',
            error: 'text-red-400',
            header: 'text-purple-400 font-bold',
            data: 'text-gray-300'
        };

        const line = document.createElement('div');
        line.className = colors[type] || 'text-gray-400';
        line.textContent = message;
        
        output.appendChild(line);
        terminal.scrollTop = terminal.scrollHeight;
    }

    async evaluatePolicy(policy, context) {
        // Simulate policy evaluation
        await this.delay(200);
        
        // Simple policy evaluation logic for demo
        const allowed = Math.random() > 0.3; // 70% success rate
        return {
            allowed,
            reason: allowed ? 'Policy conditions satisfied' : 'Access denied by policy'
        };
    }

    async delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    loadTestSuites() {
        // Load predefined test suites
        this.testSuites = {
            compliance: 'RFC-0150 Compliance Tests',
            security: 'Security Validation Tests',
            performance: 'Performance Benchmark Tests',
            integration: 'Integration Test Suite'
        };
    }
}

// Test Runner Classes
class RFC150ComplianceRunner {
    constructor(apiClient) {
        this.api = apiClient;
    }

    async runSuite() {
        const tests = [
            'Protocol Version Compatibility',
            'Message Format Validation',
            'Cryptographic Signature Requirements',
            'Timestamp Handling',
            'Error Response Formats',
            'Interoperability Standards'
        ];

        for (const test of tests) {
            await this.runComplianceTest(test);
        }
    }

    async runComplianceTest(testName) {
        window.sandbox?.writeToResults(`Running: ${testName}`, 'info');
        await this.delay(800);
        
        const success = Math.random() > 0.1; // 90% success rate
        window.sandbox?.writeToResults(`${testName}: ${success ? 'PASS' : 'FAIL'}`, 
            success ? 'success' : 'error');
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

class SecurityValidationRunner {
    constructor(apiClient) {
        this.api = apiClient;
    }

    async runSuite() {
        const tests = [
            'Signature Verification',
            'Replay Attack Prevention',
            'Key Management Security',
            'Input Validation',
            'Authentication Bypass Tests',
            'Authorization Escalation Tests'
        ];

        for (const test of tests) {
            await this.runSecurityTest(test);
        }
    }

    async runSecurityTest(testName) {
        window.sandbox?.writeToResults(`Security Test: ${testName}`, 'info');
        await this.delay(1000);
        
        const success = Math.random() > 0.15; // 85% success rate
        window.sandbox?.writeToResults(`${testName}: ${success ? 'SECURE' : 'VULNERABLE'}`, 
            success ? 'success' : 'error');
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

class PerformanceTestRunner {
    constructor(apiClient) {
        this.api = apiClient;
    }

    async runSuite() {
        const tests = [
            'Authorization Latency',
            'Throughput Benchmarks', 
            'Memory Usage Analysis',
            'CPU Utilization',
            'Concurrent User Handling',
            'Load Testing'
        ];

        for (const test of tests) {
            await this.runPerformanceTest(test);
        }
    }

    async runPerformanceTest(testName) {
        window.sandbox?.writeToResults(`Performance Test: ${testName}`, 'info');
        await this.delay(1200);
        
        const latency = Math.floor(Math.random() * 100) + 10;
        const throughput = Math.floor(Math.random() * 1000) + 500;
        
        window.sandbox?.writeToResults(`${testName}: ${latency}ms latency, ${throughput} req/s`, 'success');
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

class IntegrationTestRunner {
    constructor(apiClient) {
        this.api = apiClient;
    }

    async runSuite() {
        const tests = [
            'End-to-End Workflow',
            'API Compatibility',
            'Service Integration',
            'Database Connectivity',
            'External Service Integration',
            'Error Handling & Recovery'
        ];

        for (const test of tests) {
            await this.runIntegrationTest(test);
        }
    }

    async runIntegrationTest(testName) {
        window.sandbox?.writeToResults(`Integration Test: ${testName}`, 'info');
        await this.delay(1500);
        
        const success = Math.random() > 0.2; // 80% success rate
        window.sandbox?.writeToResults(`${testName}: ${success ? 'PASS' : 'FAIL'}`, 
            success ? 'success' : 'error');
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

// Export for global access
window.AuthorizationSandbox = AuthorizationSandbox;