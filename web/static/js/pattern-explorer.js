/**
 * Interactive Pattern Explorer - Marketing Use Cases with PoA
 * Focuses on RAG systems and AI-powered digital marketing agents
 */
class InteractivePatternExplorer {
    constructor(apiClient) {
        this.api = apiClient;
        this.currentPattern = null;
        this.simulationState = {};
        this.init();
    }

    init() {
        this.bindEvents();
        this.loadPatterns();
    }

    bindEvents() {
        document.addEventListener('click', (e) => {
            if (e.target.matches('[data-explore-pattern]') {
                const patternId = e.target.dataset.explorePattern;
                this.explorePattern(patternId);
            }
            if (e.target.matches('[data-run-simulation]') {
                const simulationId = e.target.dataset.runSimulation;
                this.runSimulation(simulationId);
            }
            if (e.target.matches('[data-modify-parameters]') {
                this.showParameterEditor();
            }
        });
    }

    getPatterns() {
        return {
            'enterprise-rag-market-research': {
                title: 'Enterprise RAG Market Research System',
                category: 'Market Intelligence',
                complexity: 'Advanced',
                description: 'Retrieval-Augmented Generation system for supercharging market research and competitive intelligence',
                useCase: {
                    problem: 'Manual market research is time-consuming and may miss emerging trends',
                    solution: 'AI-powered RAG system analyzes multiple data sources for instant insights',
                    value: 'Swift reaction to market demands, automated evidence-based recommendations'
                },
                architecture: {
                    components: [
                        {
                            name: 'RAG Engine',
                            description: 'Core retrieval and generation system',
                            capabilities: ['Document analysis', 'Trend identification', 'Insight synthesis']
                        },
                        {
                            name: 'Data Sources',
                            description: 'Multiple information feeds',
                            capabilities: ['Internal reports', 'Public publications', 'Patents', 'News feeds', 'Social media']
                        },
                        {
                            name: 'PoA Controller',
                            description: 'Authorization and compliance layer',
                            capabilities: ['Access control', 'Budget limits', 'Compliance checking']
                        }
                    ]
                },
                workflows: [
                    {
                        name: 'Market Trend Analysis',
                        steps: [
                            'Query: "Latest trends in smart manufacturing in Southeast Asia"',
                            'RAG scans internal company reports and patents',
                            'Analyzes public publications and news feeds',
                            'Reviews social media discussions among engineers',
                            'Synthesizes insights into actionable recommendations',
                            'Validates recommendations against PoA constraints'
                        ]
                    },
                    {
                        name: 'Regulatory Intelligence',
                        steps: [
                            'Query: "Regulatory changes for industrial AI in European Union"',
                            'Scans regulatory databases and government publications',
                            'Analyzes industry compliance reports',
                            'Identifies upcoming policy changes',
                            'Generates compliance strategy recommendations'
                        ]
                    }
                ],
                poaRequirements: {
                    principal: 'ACME Corp',
                    agent: 'RAG Market Research System',
                    capabilities: [
                        'Access internal research databases',
                        'Query external market intelligence APIs',
                        'Generate market analysis reports',
                        'Recommend strategic actions'
                    ],
                    constraints: [
                        'Query budget: €5,000/month',
                        'Data retention: 90 days max',
                        'Geographic scope: Global',
                        'Confidentiality: Internal use only'
                    ]
                }
            },
            'enterprise-digital-marketing-agent': {
                title: 'AI-Powered Digital Marketing Agent',
                category: 'Marketing Automation',
                complexity: 'Advanced',
                description: 'Autonomous digital marketing agent for amplifying brand presence and customer engagement',
                useCase: {
                    problem: 'Manual social media management cannot keep pace with real-time market dynamics',
                    solution: 'AI agent autonomously manages multi-channel marketing while adhering to brand guidelines',
                    value: 'Dynamic brand presence, increased engagement, proactive message adaptation'
                },
                architecture: {
                    components: [
                        {
                            name: 'Content Intelligence Engine',
                            description: 'AI-powered content analysis and generation',
                            capabilities: ['Trend analysis', 'Sentiment monitoring', 'Content creation', 'Audience targeting']
                        },
                        {
                            name: 'Multi-Channel Manager',
                            description: 'Cross-platform marketing execution',
                            capabilities: ['LinkedIn automation', 'Twitter engagement', 'Blog publishing', 'Email campaigns']
                        },
                        {
                            name: 'PoA Enforcement Layer',
                            description: 'Real-time authorization and compliance',
                            capabilities: ['Budget control', 'Content approval', 'Brand guideline compliance', 'Legal review']
                        }
                    ]
                },
                workflows: [
                    {
                        name: 'Autonomous Content Creation',
                        steps: [
                            'Monitor trending topics in engineering communities',
                            'Analyze audience sentiment and engagement patterns',
                            'Generate content aligned with corporate messaging',
                            'Validate content against brand guidelines (PoA check)',
                            'Schedule and publish across approved channels',
                            'Monitor performance and adjust strategy'
                        ]
                    },
                    {
                        name: 'Real-Time Customer Engagement',
                        steps: [
                            'Monitor social media mentions and queries',
                            'Analyze customer intent and sentiment',
                            'Generate appropriate responses within PoA scope',
                            'Escalate complex issues to human agents',
                            'Log interactions for compliance audit'
                        ]
                    }
                ],
                poaRequirements: {
                    principal: 'ACME Marketing Department',
                    agent: 'Digital Marketing AI Agent',
                    capabilities: [
                        'Create and publish social media content',
                        'Respond to customer inquiries',
                        'Manage advertising budgets',
                        'Target specific demographics',
                        'Curate user-generated content'
                    ],
                    constraints: [
                        'Monthly budget: €50,000',
                        'Content approval: Auto for standard, manual for sensitive',
                        'Response time: <2 hours for customer inquiries',
                        'Geographic targeting: Global (exclude restricted countries)',
                        'Brand compliance: All content must pass guideline check'
                    ]
                }
            },
            'marketing-poa-enforcement': {
                title: 'Marketing PoA Rule-Based Enforcement',
                category: 'Compliance & Governance',
                complexity: 'Intermediate',
                description: 'Real-time PoA validation for marketing agent requests with rule-based enforcement',
                useCase: {
                    problem: 'Marketing agents need autonomy while ensuring compliance with corporate policies',  
                    solution: 'Automated PoA validation system with granular rule enforcement',
                    value: 'Reduced manual oversight, consistent policy application, audit trail'
                },
                architecture: {
                    components: [
                        {
                            name: 'PoA Validator',
                            description: 'Core authorization validation engine',
                            capabilities: ['Credential verification', 'Constraint checking', 'Decision logging']
                        },
                        {
                            name: 'Rule Engine',
                            description: 'Policy enforcement and business rules',
                            capabilities: ['Budget limits', 'Content guidelines', 'Temporal constraints', 'Geographic restrictions']
                        },
                        {
                            name: 'Audit Logger',
                            description: 'Comprehensive decision tracking',
                            capabilities: ['Decision logging', 'Compliance reporting', 'Performance metrics']
                        }
                    ]
                },
                workflows: [
                    {
                        name: 'Social Media Targeting Request',
                        steps: [
                            'Marketing agent requests social media campaign',
                            'System extracts request attributes (budget, audience, content)',
                            'PoA credential validation performed',
                            'Rule-based constraint checking applied',
                            'Decision rendered with detailed reasoning',
                            'Action logged for audit and compliance'
                        ]
                    }
                ],
                poaRequirements: {
                    principal: 'ACME Corporate Marketing',
                    agent: 'Marketing Campaign Manager',
                    capabilities: [
                        'Social media campaign creation',
                        'Audience targeting and segmentation',
                        'Budget allocation and spending'
                    ],
                    constraints: [
                        'Budget per campaign: €10,000 max',
                        'Target audience: B2B engineering professionals',
                        'Content type: Product announcements, thought leadership',
                        'Geographic scope: EMEA region',
                        'Approval required for: Regulatory content, pricing information'
                    ]
                }
            }
        };
    }

    loadPatterns() {
        const container = document.getElementById('pattern-explorer');
        if (!container) return;

        const patterns = this.getPatterns();
        
        container.innerHTML = `
            <div class="pattern-explorer">
                <div class="explorer-header mb-6">
                    <h2 class="text-2xl font-bold text-gray-800 mb-2">Interactive Pattern Explorer</h2>
                    <p class="text-gray-600">Explore real-world marketing use cases with PoA-enabled AI agents</p>
                </div>

                <div class="patterns-grid grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
                    ${Object.entries(patterns).map(([patternId, pattern]) => `
                        <div class="pattern-card bg-white rounded-lg shadow-lg overflow-hidden hover:shadow-xl transition-shadow">
                            <div class="pattern-header bg-gradient-to-r from-blue-500 to-purple-600 text-white p-4">
                                <h3 class="font-bold text-lg">${pattern.title}</h3>
                                <div class="flex items-center justify-between mt-2 text-sm">
                                    <span class="bg-white bg-opacity-20 px-2 py-1 rounded">${pattern.category}</span>
                                    <span class="bg-white bg-opacity-20 px-2 py-1 rounded">${pattern.complexity}</span>
                                </div>
                            </div>
                            
                            <div class="pattern-content p-4">
                                <p class="text-gray-600 text-sm mb-4">${pattern.description}</p>
                                
                                <div class="use-case mb-4">
                                    <h4 class="font-semibold text-sm text-gray-800 mb-2">Value Proposition:</h4>
                                    <p class="text-xs text-gray-600">${pattern.useCase.value}</p>
                                </div>

                                <div class="architecture-preview mb-4">
                                    <h4 class="font-semibold text-sm text-gray-800 mb-2">Components:</h4>
                                    <div class="space-y-1">
                                        ${pattern.architecture.components.slice(0, 2).map(comp => `
                                            <div class="text-xs bg-gray-50 p-2 rounded">
                                                <span class="font-medium">${comp.name}:</span> ${comp.description}
                                            </div>
                                        `).join('')}
                                        ${pattern.architecture.components.length > 2 ? `
                                            <div class="text-xs text-gray-500">+${pattern.architecture.components.length - 2} more components</div>
                                        ` : ''}
                                    </div>
                                </div>

                                <button data-explore-pattern="${patternId}" 
                                        class="w-full bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 transition-colors">
                                    Explore Pattern
                                </button>
                            </div>
                        </div>
                    `).join('')}
                </div>

                <div id="pattern-detail" class="mt-8 hidden">
                    <!-- Pattern detail view will be inserted here -->
                </div>
            </div>
        `;
    }

    explorePattern(patternId) {
        const patterns = this.getPatterns();
        const pattern = patterns[patternId];
        
        if (!pattern) {
            console.error('Pattern not found:', patternId);
            return;
        }

        this.currentPattern = patternId;
        this.renderPatternDetail(pattern);
    }

    renderPatternDetail(pattern) {
        const detailContainer = document.getElementById('pattern-detail');
        if (!detailContainer) return;

        detailContainer.classList.remove('hidden');
        detailContainer.innerHTML = `
            <div class="pattern-detail bg-white rounded-lg shadow-lg">
                <div class="detail-header bg-gradient-to-r from-indigo-500 to-purple-600 text-white p-6">
                    <h2 class="text-2xl font-bold mb-2">${pattern.title}</h2>
                    <p class="text-indigo-100">${pattern.description}</p>
                    <div class="flex space-x-4 mt-4 text-sm">
                        <span class="bg-white bg-opacity-20 px-3 py-1 rounded-full">${pattern.category}</span>
                        <span class="bg-white bg-opacity-20 px-3 py-1 rounded-full">${pattern.complexity}</span>
                    </div>
                </div>

                <div class="detail-content p-6">
                    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
                        <!-- Use Case Section -->
                        <div class="use-case-section">
                            <h3 class="text-xl font-bold mb-4 text-gray-800">Use Case Overview</h3>
                            <div class="space-y-4">
                                <div class="bg-red-50 p-4 rounded-lg">
                                    <h4 class="font-semibold text-red-800 mb-2">Problem</h4>
                                    <p class="text-red-700 text-sm">${pattern.useCase.problem}</p>
                                </div>
                                <div class="bg-blue-50 p-4 rounded-lg">
                                    <h4 class="font-semibold text-blue-800 mb-2">Solution</h4>
                                    <p class="text-blue-700 text-sm">${pattern.useCase.solution}</p>
                                </div>
                                <div class="bg-green-50 p-4 rounded-lg">
                                    <h4 class="font-semibold text-green-800 mb-2">Business Value</h4>
                                    <p class="text-green-700 text-sm">${pattern.useCase.value}</p>
                                </div>
                            </div>
                        </div>

                        <!-- Architecture Section -->
                        <div class="architecture-section">
                            <h3 class="text-xl font-bold mb-4 text-gray-800">System Architecture</h3>
                            <div class="space-y-3">
                                ${pattern.architecture.components.map(comp => `
                                    <div class="component bg-gray-50 p-4 rounded-lg">
                                        <h4 class="font-semibold text-gray-800 mb-2">${comp.name}</h4>
                                        <p class="text-gray-600 text-sm mb-2">${comp.description}</p>
                                        <div class="capabilities">
                                            <div class="flex flex-wrap gap-1">
                                                ${comp.capabilities.map(cap => `
                                                    <span class="bg-blue-100 text-blue-800 text-xs px-2 py-1 rounded">${cap}</span>
                                                `).join('')}
                                            </div>
                                        </div>
                                    </div>
                                `).join('')}
                            </div>
                        </div>
                    </div>

                    <!-- PoA Requirements Section -->
                    <div class="poa-section mt-8">
                        <h3 class="text-xl font-bold mb-4 text-gray-800">Power of Attorney Requirements</h3>
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                            <div class="poa-capabilities bg-green-50 p-4 rounded-lg">
                                <h4 class="font-semibold text-green-800 mb-3">Delegated Capabilities</h4>
                                <ul class="space-y-2 text-sm">
                                    ${pattern.poaRequirements.capabilities.map(cap => `
                                        <li class="flex items-start">
                                            <span class="text-green-600 mr-2">✓</span>
                                            <span class="text-green-700">${cap}</span>
                                        </li>
                                    `).join('')}
                                </ul>
                            </div>
                            
                            <div class="poa-constraints bg-amber-50 p-4 rounded-lg">
                                <h4 class="font-semibold text-amber-800 mb-3">Operational Constraints</h4>
                                <ul class="space-y-2 text-sm">
                                    ${pattern.poaRequirements.constraints.map(constraint => `
                                        <li class="flex items-start">
                                            <span class="text-amber-600 mr-2">⚠</span>
                                            <span class="text-amber-700">${constraint}</span>
                                        </li>
                                    `).join('')}
                                </ul>
                            </div>
                        </div>
                    </div>

                    <!-- Workflows Section -->
                    <div class="workflows-section mt-8">
                        <h3 class="text-xl font-bold mb-4 text-gray-800">Key Workflows</h3>
                        <div class="space-y-6">
                            ${pattern.workflows.map((workflow, index) => `
                                <div class="workflow bg-indigo-50 p-4 rounded-lg">
                                    <h4 class="font-semibold text-indigo-800 mb-3">${workflow.name}</h4>
                                    <div class="steps space-y-2">
                                        ${workflow.steps.map((step, stepIndex) => `
                                            <div class="step flex items-start">
                                                <div class="step-number bg-indigo-600 text-white rounded-full w-6 h-6 flex items-center justify-center text-xs font-bold mr-3 mt-0.5">
                                                    ${stepIndex + 1}
                                                </div>
                                                <div class="step-content text-sm text-indigo-700">${step}</div>
                                            </div>
                                        `).join('')}
                                    </div>
                                </div>
                            `).join('')}
                        </div>
                    </div>

                    <!-- Interactive Simulation Section -->
                    <div class="simulation-section mt-8">
                        <h3 class="text-xl font-bold mb-4 text-gray-800">Interactive Simulation</h3>
                        <div class="bg-purple-50 p-6 rounded-lg">
                            <p class="text-purple-700 mb-4">Experience this pattern in action with a live simulation</p>
                            <div class="space-x-3">
                                <button data-run-simulation="${this.currentPattern}" 
                                        class="bg-purple-600 text-white px-6 py-2 rounded hover:bg-purple-700">
                                    🚀 Run Simulation
                                </button>
                                <button data-modify-parameters 
                                        class="bg-gray-600 text-white px-6 py-2 rounded hover:bg-gray-700">
                                    ⚙️ Modify Parameters
                                </button>
                            </div>
                        </div>
                        
                        <div id="simulation-output" class="mt-6 hidden">
                            <div class="bg-gray-900 text-green-400 p-4 rounded-lg font-mono text-sm">
                                <div id="simulation-log"></div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;

        // Scroll to detail view
        detailContainer.scrollIntoView({ behavior: 'smooth' });
    }

    async runSimulation(patternId) {
        const outputDiv = document.getElementById('simulation-output');
        const logDiv = document.getElementById('simulation-log');
        
        if (!outputDiv || !logDiv) return;
        
        outputDiv.classList.remove('hidden');
        logDiv.innerHTML = '<div class="text-blue-400">Initializing simulation...</div>';

        try {
            await this.executePatternSimulation(patternId);
        } catch (error) {
            logDiv.innerHTML += `<div class="text-red-400">Simulation error: ${error.message}</div>`;
        }
    }

    async executePatternSimulation(patternId) {
        const logDiv = document.getElementById('simulation-log');
        const patterns = this.getPatterns();
        const pattern = patterns[patternId];

        switch (patternId) {
            case 'enterprise-rag-market-research':
                await this.simulateRAGSystem(logDiv, pattern);
                break;
            case 'enterprise-digital-marketing-agent':
                await this.simulateMarketingAgent(logDiv, pattern);
                break;
            case 'marketing-poa-enforcement':
                await this.simulatePoAEnforcement(logDiv, pattern);
                break;
            default:
                logDiv.innerHTML = '<div class="text-yellow-400">Generic simulation not implemented for this pattern</div>';
        }
    }

    async simulateRAGSystem(logDiv, pattern) {
        const steps = [
            'Starting RAG Market Research System...',
            'Loading internal company research database...',
            'Connecting to external market intelligence APIs...',
            'Query received: "Latest trends in smart manufacturing in Southeast Asia"',
            'Scanning internal reports (2,847 documents)...',
            'Analyzing patents database (1,234 relevant patents found)...',
            'Processing news feeds from last 30 days...',
            'Extracting insights from social media discussions...',
            'Synthesizing findings with AI engine...',
            '📊 Key trends identified:',
            '  • Edge computing integration +45% mentions',
            '  • Sustainability focus +62% industry discussion',
            '  • AI-driven predictive maintenance +38% patent filings',
            'Validating recommendations against PoA constraints...',
            '✅ Budget check: €2,400 < €5,000 monthly limit',
            '✅ Geographic scope: Southeast Asia ✓',
            '✅ Confidentiality: Internal use only ✓',
            '📋 Generated market analysis report (Report-ID: MAR-2025-1030)',
            '🎯 Strategic recommendations:',
            '  1. Accelerate edge computing R&D investment',
            '  2. Strengthen sustainability messaging in SEA markets',
            '  3. Expand predictive maintenance solutions portfolio',
            '✅ Simulation completed successfully!'
        ];

        for (let i = 0; i < steps.length; i++) {
            await this.delay(800);
            logDiv.innerHTML += `<div>${steps[i]}</div>`;
            logDiv.scrollTop = logDiv.scrollHeight;
        }
    }

    async simulateMarketingAgent(logDiv, pattern) {
        const steps = [
            'Initializing Digital Marketing AI Agent...',
            'Loading brand guidelines and corporate messaging framework...',
            'Connecting to social media monitoring APIs...',
            '🔍 Analyzing trending topics in engineering communities:',
            '  • "Industrial IoT security" - 15,000 mentions (+25%)',
            '  • "Digital twin technology" - 8,500 mentions (+18%)',
            '  • "Sustainable manufacturing" - 12,200 mentions (+32%)',
            '🎯 Identifying target audience segments:',
            '  • Manufacturing engineers (EMEA): 45,000 active users',
            '  • Industrial IoT specialists: 23,000 active users',
            '  • Sustainability officers: 18,000 active users',
            '✍️  Generating content for LinkedIn campaign...',
            'Content created: "ACME Digital Twin Solutions for Sustainable Manufacturing"',
            '🔒 Validating content against PoA constraints:',
            '  ✅ Budget allocation: €8,500 < €50,000 monthly limit',
            '  ✅ Brand guideline compliance: PASSED',
            '  ✅ Target audience: B2B engineering professionals ✓',
            '  ✅ Content type: Product announcement ✓',
            '  ✅ Geographic scope: EMEA region ✓',
            '📱 Publishing content across channels:',
            '  • LinkedIn: Post scheduled for 2:00 PM CET',
            '  • Twitter: Thread created (3 tweets)',
            '  • Blog: Article draft generated',
            '📊 Real-time engagement monitoring activated',
            '💬 Auto-response system enabled for customer inquiries',
            '📈 Performance tracking initialized',
            '✅ Campaign launched successfully!',
            '📋 Campaign metrics will be available in 1 hour'
        ];

        for (let i = 0; i < steps.length; i++) {
            await this.delay(600);
            logDiv.innerHTML += `<div>${steps[i]}</div>`;
            logDiv.scrollTop = logDiv.scrollHeight;
        }
    }

    async simulatePoAEnforcement(logDiv, pattern) {
        const steps = [
            'Starting PoA Rule-Based Enforcement Simulation...',
            '📥 Marketing agent request received:',
            '   Agent: acme-marketing-campaign-manager',
            '   Action: create_social_media_campaign',
            '   Target: LinkedIn ads',
            '   Budget: €8,500',
            '   Audience: Engineers in EMEA region',
            '   Content: Smart manufacturing trends',
            '',
            '🔐 Validating PoA credential:',
            '  ✅ Credential signature: VALID',
            '  ✅ Credential expiry: Valid until 2025-12-31',
            '  ✅ Agent identity: CONFIRMED',
            '',
            '⚖️  Applying rule-based enforcement:',
            '  🔍 Budget constraint check:',
            '    • Requested: €8,500',
            '    • PoA limit: €10,000 per campaign',
            '    • Result: ✅ WITHIN LIMITS',
            '',
            '  🔍 Audience targeting check:',
            '    • Requested: Engineers in EMEA',
            '    • PoA scope: B2B engineering professionals',
            '    • Geographic: EMEA region allowed',
            '    • Result: ✅ COMPLIANT',
            '',
            '  🔍 Content type validation:',
            '    • Content: Smart manufacturing trends',
            '    • PoA allows: Product announcements, thought leadership',
            '    • Classification: Thought leadership',
            '    • Result: ✅ APPROVED',
            '',
            '  🔍 Approval requirements:',
            '    • Content sensitivity: Standard',
            '    • Manual approval required: NO',
            '    • Auto-approval eligible: YES',
            '    • Result: ✅ AUTO-APPROVED',
            '',
            '📊 Final authorization decision:',
            '  🟢 DECISION: APPROVED',
            '  📝 Reason: All PoA constraints satisfied',
            '  ⏱️  Processing time: 145ms',
            '  📋 Audit log: Entry created (ID: AUD-20251030-1547)',
            '',
            '🚀 Campaign execution authorized!',
            '✅ PoA enforcement simulation completed'
        ];

        for (let i = 0; i < steps.length; i++) {
            await this.delay(400);
            logDiv.innerHTML += `<div>${steps[i]}</div>`;
            logDiv.scrollTop = logDiv.scrollHeight;
        }
    }

    showParameterEditor() {
        const modal = document.createElement('div');
        modal.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50';
        modal.innerHTML = `
            <div class="bg-white rounded-lg p-6 max-w-2xl w-full max-h-96 overflow-y-auto">
                <h3 class="text-xl font-bold mb-4">Simulation Parameters</h3>
                <div class="space-y-4">
                    <div>
                        <label class="block text-sm font-medium mb-1">Marketing Budget (EUR)</label>
                        <input type="number" value="8500" class="w-full border rounded px-3 py-2">
                    </div>
                    <div>
                        <label class="block text-sm font-medium mb-1">Target Audience</label>
                        <select class="w-full border rounded px-3 py-2">
                            <option>Engineers (EMEA)</option>
                            <option>Manufacturing professionals</option>
                            <option>IoT specialists</option>
                        </select>
                    </div>
                    <div>
                        <label class="block text-sm font-medium mb-1">Campaign Duration (days)</label>
                        <input type="number" value="30" class="w-full border rounded px-3 py-2">
                    </div>
                    <div>
                        <label class="block text-sm font-medium mb-1">Content Type</label>
                        <select class="w-full border rounded px-3 py-2">
                            <option>Product announcement</option>
                            <option>Thought leadership</option>
                            <option>Case study</option>
                        </select>
                    </div>
                </div>
                <div class="flex justify-end space-x-3 mt-6">
                    <button onclick="this.closest('.fixed').remove()" class="px-4 py-2 bg-gray-500 text-white rounded">
                        Cancel
                    </button>
                    <button onclick="this.closest('.fixed').remove()" class="px-4 py-2 bg-blue-600 text-white rounded">
                        Apply Parameters
                    </button>
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

// Export for use in other modules
window.InteractivePatternExplorer = InteractivePatternExplorer;