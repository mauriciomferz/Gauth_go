/**
 * AgentAuth Learning Path - Interactive Educational System
 * Provides comprehensive learning modules for Authorization, PoA, Delegation, etc.
 */
class AgentAuthLearningPath {
    constructor(apiClient) {
        this.api = apiClient;
        this.currentModule = null;
        this.currentStep = 0;
        this.completedModules = new Set();
        this.userProgress = {};
        this.init();
    }

    init() {
        this.bindEvents();
        this.loadProgress();
    }

    bindEvents() {
        // Module navigation
        document.addEventListener('click', (e) => {
            if (e.target.matches('[data-start-module]') {
                const moduleId = e.target.dataset.startModule;
                this.startModule(moduleId);
            }
            if (e.target.matches('[data-next-step]') {
                this.nextStep();
            }
            if (e.target.matches('[data-prev-step]') {
                this.previousStep();
            }
            if (e.target.matches('[data-run-exercise]') {
                const exerciseId = e.target.dataset.runExercise;
                this.runExercise(exerciseId);
            }
        });
    }

    // Learning Modules Definition
    getModules() {
        return {
            'auth-fundamentals': {
                title: 'Authorization Fundamentals',
                description: 'Learn the core concepts of authorization in AgentAuth',
                duration: '15 minutes',
                difficulty: 'Beginner',
                steps: [
                    {
                        title: 'What is Authorization?',
                        content: this.getAuthFundamentalsStep1(),
                        interactive: true,
                        exercise: 'basic-authz-check'
                    },
                    {
                        title: 'Subject, Resource, Action Model',
                        content: this.getAuthFundamentalsStep2(),
                        interactive: true,
                        exercise: 'authz-components'
                    },
                    {
                        title: 'Policy-Based Access Control',
                        content: this.getAuthFundamentalsStep3(),
                        interactive: true,
                        exercise: 'policy-evaluation'
                    },
                    {
                        title: 'Context and Attributes',
                        content: this.getAuthFundamentalsStep4(),
                        interactive: true,
                        exercise: 'context-demo'
                    }
                ]
            },
            'poa-fundamentals': {
                title: 'Power of Attorney (PoA)',
                description: 'Master delegation and power of attorney concepts',
                duration: '20 minutes',
                difficulty: 'Intermediate',
                steps: [
                    {
                        title: 'Understanding PoA',
                        content: this.getPoAStep1(),
                        interactive: true,
                        exercise: 'poa-basics'
                    },
                    {
                        title: 'Creating PoA Credentials',
                        content: this.getPoAStep2(),
                        interactive: true,
                        exercise: 'poa-creation'
                    },
                    {
                        title: 'PoA Validation & Enforcement',
                        content: this.getPoAStep3(),
                        interactive: true,
                        exercise: 'poa-validation'
                    },
                    {
                        title: 'Marketing PoA Use Case',
                        content: this.getPoAMarketingCase(),
                        interactive: true,
                        exercise: 'marketing-poa'
                    }
                ]
            },
            'hierarchical-delegation': {
                title: 'Hierarchical Delegation',
                description: 'Learn about delegation chains and hierarchies',
                duration: '18 minutes',
                difficulty: 'Intermediate',
                steps: [
                    {
                        title: 'Delegation Concepts',
                        content: this.getDelegationStep1(),
                        interactive: true,
                        exercise: 'delegation-basics'
                    },
                    {
                        title: 'Building Delegation Chains',
                        content: this.getDelegationStep2(),
                        interactive: true,
                        exercise: 'delegation-chain'
                    },
                    {
                        title: 'Constraint Propagation',
                        content: this.getDelegationStep3(),
                        interactive: true,
                        exercise: 'constraint-demo'
                    }
                ]
            },
            'cascade-revocation': {
                title: 'Cascade Revocation',
                description: 'Understand revocation and its cascading effects',
                duration: '12 minutes',
                difficulty: 'Advanced',
                steps: [
                    {
                        title: 'Revocation Fundamentals',
                        content: this.getRevocationStep1(),
                        interactive: true,
                        exercise: 'revocation-basics'
                    },
                    {
                        title: 'Cascade Effects',
                        content: this.getRevocationStep2(),
                        interactive: true,
                        exercise: 'cascade-demo'
                    },
                    {
                        title: 'Revocation Strategies',
                        content: this.getRevocationStep3(),
                        interactive: true,
                        exercise: 'revocation-strategies'
                    }
                ]
            },
            'audit-compliance': {
                title: 'Audit & Compliance',
                description: 'Learn about audit trails and compliance monitoring',
                duration: '16 minutes',
                difficulty: 'Intermediate',
                steps: [
                    {
                        title: 'Audit Trail Basics',
                        content: this.getAuditStep1(),
                        interactive: true,
                        exercise: 'audit-basics'
                    },
                    {
                        title: 'Compliance Monitoring',
                        content: this.getAuditStep2(),
                        interactive: true,
                        exercise: 'compliance-check'
                    },
                    {
                        title: 'Regulatory Frameworks',
                        content: this.getAuditStep3(),
                        interactive: true,
                        exercise: 'regulatory-demo'
                    }
                ]
            },
            'rfc-150-deep-dive': {
                title: 'RFC-150 Deep Dive',
                description: 'Comprehensive exploration of RFC-150 specifications',
                duration: '25 minutes',
                difficulty: 'Advanced',
                steps: [
                    {
                        title: 'RFC-150 Overview',
                        content: this.getRFC150Step1(),
                        interactive: true,
                        exercise: 'rfc-overview'
                    },
                    {
                        title: 'Protocol Implementation',
                        content: this.getRFC150Step2(),
                        interactive: true,
                        exercise: 'protocol-demo'
                    },
                    {
                        title: 'Advanced Features',
                        content: this.getRFC150Step3(),
                        interactive: true,
                        exercise: 'advanced-features'
                    }
                ]
            }
        };
    }

    // Module Content Methods
    getAuthFundamentalsStep1() {
        return `
            <div class="learning-content">
                <h3 class="text-xl font-bold mb-4">What is Authorization?</h3>
                <div class="space-y-4">
                    <p class="text-gray-700">Authorization is the process of determining whether a subject (user, service, or entity) has permission to perform a specific action on a particular resource.</p>
                    
                    <div class="bg-blue-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-blue-800 mb-2">Key Components:</h4>
                        <ul class="list-disc pl-5 space-y-1 text-blue-700">
                            <li><strong>Subject:</strong> Who is requesting access</li>
                            <li><strong>Resource:</strong> What is being accessed</li>
                            <li><strong>Action:</strong> What operation is being performed</li>
                            <li><strong>Context:</strong> Additional information that influences the decision</li>
                        </ul>
                    </div>

                    <div class="bg-yellow-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-yellow-800 mb-2">Example Scenario:</h4>
                        <p class="text-yellow-700">Alice (subject) wants to read (action) a financial report (resource) during business hours (context).</p>
                    </div>

                    <div class="bg-green-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-green-800 mb-2">Try It Yourself:</h4>
                        <p class="text-green-700">Click the exercise button below to test a real authorization check!</p>
                    </div>
                </div>
            </div>
        `;
    }

    getAuthFundamentalsStep2() {
        return `
            <div class="learning-content">
                <h3 class="text-xl font-bold mb-4">Subject, Resource, Action Model</h3>
                <div class="space-y-4">
                    <p class="text-gray-700">The SRA (Subject-Resource-Action) model is the foundation of authorization decisions in AgentAuth.</p>
                    
                    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                        <div class="bg-blue-50 p-4 rounded-lg">
                            <h4 class="font-semibold text-blue-800 mb-2">Subject</h4>
                            <ul class="text-sm text-blue-700 space-y-1">
                                <li>• User accounts</li>
                                <li>• Service principals</li>
                                <li>• API clients</li>
                                <li>• Roles & groups</li>
                            </ul>
                        </div>
                        <div class="bg-green-50 p-4 rounded-lg">
                            <h4 class="font-semibold text-green-800 mb-2">Resource</h4>
                            <ul class="text-sm text-green-700 space-y-1">
                                <li>• Documents</li>
                                <li>• APIs</li>
                                <li>• Databases</li>
                                <li>• Services</li>
                            </ul>
                        </div>
                        <div class="bg-purple-50 p-4 rounded-lg">
                            <h4 class="font-semibold text-purple-800 mb-2">Action</h4>
                            <ul class="text-sm text-purple-700 space-y-1">
                                <li>• Read</li>
                                <li>• Write</li>
                                <li>• Delete</li>
                                <li>• Execute</li>
                            </ul>
                        </div>
                    </div>

                    <div class="bg-gray-100 p-4 rounded-lg">
                        <h4 class="font-semibold mb-2">Interactive Example:</h4>
                        <div class="grid grid-cols-3 gap-2 text-sm">
                            <div class="text-center">
                                <div class="bg-blue-200 p-2 rounded">alice@company.com</div>
                                <div class="text-xs mt-1">Subject</div>
                            </div>
                            <div class="text-center">
                                <div class="bg-green-200 p-2 rounded">financial-report</div>
                                <div class="text-xs mt-1">Resource</div>
                            </div>
                            <div class="text-center">
                                <div class="bg-purple-200 p-2 rounded">read</div>
                                <div class="text-xs mt-1">Action</div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    getAuthFundamentalsStep3() {
        return `
            <div class="learning-content">
                <h3 class="text-xl font-bold mb-4">Policy-Based Access Control</h3>
                <div class="space-y-4">
                    <p class="text-gray-700">AgentAuth uses policies to define access rules. Policies are evaluated to make allow/deny decisions.</p>
                    
                    <div class="bg-indigo-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-indigo-800 mb-2">Policy Structure:</h4>
                        <pre class="text-sm text-indigo-700 bg-indigo-100 p-3 rounded overflow-x-auto">
{
  "policy_id": "finance-read-policy",
  "rules": [
    {
      "effect": "allow",
      "subject": {"department": "finance"},
      "resource": {"type": "financial-report"},
      "action": "read",
      "conditions": {
        "time": "business_hours",
        "classification": ["public", "internal"]
      }
    }
  ]
}</pre>
                    </div>

                    <div class="bg-amber-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-amber-800 mb-2">Policy Evaluation Process:</h4>
                        <ol class="list-decimal pl-5 space-y-2 text-amber-700">
                            <li>Collect all applicable policies</li>
                            <li>Evaluate conditions and constraints</li>
                            <li>Apply policy combination algorithms</li>
                            <li>Return final decision with reasoning</li>
                        </ol>
                    </div>
                </div>
            </div>
        `;
    }

    getAuthFundamentalsStep4() {
        return `
            <div class="learning-content">
                <h3 class="text-xl font-bold mb-4">Context and Attributes</h3>
                <div class="space-y-4">
                    <p class="text-gray-700">Context provides additional information that influences authorization decisions beyond the basic SRA model.</p>
                    
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div class="bg-teal-50 p-4 rounded-lg">
                            <h4 class="font-semibold text-teal-800 mb-2">Environmental Context:</h4>
                            <ul class="text-sm text-teal-700 space-y-1">
                                <li>• Time of day</li>
                                <li>• Location/IP address</li>
                                <li>• Device type</li>
                                <li>• Network security level</li>
                            </ul>
                        </div>
                        <div class="bg-rose-50 p-4 rounded-lg">
                            <h4 class="font-semibold text-rose-800 mb-2">Subject Attributes:</h4>
                            <ul class="text-sm text-rose-700 space-y-1">
                                <li>• Department</li>
                                <li>• Clearance level</li>
                                <li>• Role/title</li>
                                <li>• Employment status</li>
                            </ul>
                        </div>
                    </div>

                    <div class="bg-gray-50 p-4 rounded-lg">
                        <h4 class="font-semibold mb-2">Context Example:</h4>
                        <div class="text-sm bg-white p-3 rounded border">
                            <strong>Request:</strong> Marketing agent wants to post on social media<br>
                            <strong>Context:</strong>
                            <ul class="mt-2 ml-4 space-y-1">
                                <li>• Time: Business hours (9-17 CET)</li>
                                <li>• Budget remaining: €15,000</li>
                                <li>• Target audience: Engineering professionals</li>
                                <li>• Content type: Product announcement</li>
                                <li>• Compliance: GDPR-compliant</li>
                            </ul>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    getPoAStep1() {
        return `
            <div class="learning-content">
                <h3 class="text-xl font-bold mb-4">Understanding Power of Attorney (PoA)</h3>
                <div class="space-y-4">
                    <p class="text-gray-700">Power of Attorney in AgentAuth allows one entity to delegate specific capabilities to another entity, creating a chain of authorized actions.</p>
                    
                    <div class="bg-blue-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-blue-800 mb-2">Core Concepts:</h4>
                        <ul class="list-disc pl-5 space-y-1 text-blue-700">
                            <li><strong>Principal:</strong> The entity granting the power</li>
                            <li><strong>Agent:</strong> The entity receiving the power</li>
                            <li><strong>Scope:</strong> What actions are delegated</li>
                            <li><strong>Constraints:</strong> Limitations on the delegation</li>
                        </ul>
                    </div>

                    <div class="bg-green-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-green-800 mb-2">Enterprise Marketing Example:</h4>
                        <p class="text-green-700">ACME Corp (Principal) grants PoA to Marketing AI Agent (Agent) to:</p>
                        <ul class="list-disc pl-5 mt-2 text-green-600">
                            <li>Create social media posts</li>
                            <li>Respond to customer inquiries</li>
                            <li>Spend up to €50,000/month on advertising</li>
                            <li>Target specific demographics</li>
                        </ul>
                    </div>

                    <div class="bg-yellow-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-yellow-800 mb-2">PoA Benefits:</h4>
                        <ul class="list-disc pl-5 text-yellow-700">
                            <li>Automated decision-making within bounds</li>
                            <li>Reduced manual approval overhead</li>
                            <li>Consistent policy enforcement</li>
                            <li>Audit trail of delegated actions</li>
                        </ul>
                    </div>
                </div>
            </div>
        `;
    }

    getPoAStep2() {
        return `
            <div class="learning-content">
                <h3 class="text-xl font-bold mb-4">Creating PoA Credentials</h3>
                <div class="space-y-4">
                    <p class="text-gray-700">PoA credentials are digital certificates that encode delegation relationships and constraints.</p>
                    
                    <div class="bg-indigo-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-indigo-800 mb-2">PoA Credential Structure:</h4>
                        <pre class="text-sm text-indigo-700 bg-indigo-100 p-3 rounded overflow-x-auto">
{
  "poa_id": "acme-marketing-poa-2025",
  "principal": "acme.com",
  "agent": "marketing-ai-agent",
  "capabilities": [
    {
      "action": "social_media_post",
      "resources": ["twitter", "linkedin"],
      "constraints": {
        "budget_limit": 50000,
        "currency": "EUR",
        "time_window": "business_hours",
        "content_approval": "auto"
      }
    },
    {
      "action": "customer_response",
      "resources": ["support_tickets"],
      "constraints": {
        "response_time": "24h",
        "escalation_threshold": "complex_issues"
      }
    }
  ],
  "valid_from": "2025-01-01T00:00:00Z",
  "valid_until": "2025-12-31T23:59:59Z",
  "issuer_signature": "..."
}</pre>
                    </div>

                    <div class="bg-green-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-green-800 mb-2">Creation Process:</h4>
                        <ol class="list-decimal pl-5 space-y-2 text-green-700">
                            <li>Define delegation scope and constraints</li>
                            <li>Create cryptographic credential</li>
                            <li>Sign with principal's private key</li>
                            <li>Register in capability registry</li>
                            <li>Distribute to agent</li>
                        </ol>
                    </div>
                </div>
            </div>
        `;
    }

    getPoAStep3() {
        return `
            <div class="learning-content">
                <h3 class="text-xl font-bold mb-4">PoA Validation & Enforcement</h3>
                <div class="space-y-4">
                    <p class="text-gray-700">When an agent makes a request, the system validates the PoA credential and enforces constraints.</p>
                    
                    <div class="bg-red-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-red-800 mb-2">Validation Steps:</h4>
                        <ol class="list-decimal pl-5 space-y-2 text-red-700">
                            <li>Verify credential signature</li>
                            <li>Check validity period</li>
                            <li>Validate agent identity</li>
                            <li>Match requested action to capabilities</li>
                            <li>Enforce all constraints</li>
                            <li>Log decision for audit</li>
                        </ol>
                    </div>

                    <div class="bg-purple-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-purple-800 mb-2">Enforcement Example:</h4>
                        <div class="text-sm bg-purple-100 p-3 rounded">
                            <strong>Request:</strong> Marketing agent wants to spend €45,000 on LinkedIn ads<br>
                            <strong>Validation:</strong>
                            <ul class="mt-2 ml-4 space-y-1">
                                <li>✅ PoA signature valid</li>
                                <li>✅ Within time window</li>
                                <li>✅ LinkedIn in allowed resources</li>
                                <li>✅ €45,000 < €50,000 budget limit</li>
                                <li>✅ Business hours constraint met</li>
                            </ul>
                            <strong class="text-green-600">Result: ALLOW</strong>
                        </div>
                    </div>

                    <div class="bg-amber-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-amber-800 mb-2">Constraint Types:</h4>
                        <ul class="list-disc pl-5 text-amber-700">
                            <li><strong>Temporal:</strong> Time windows, expiration dates</li>
                            <li><strong>Financial:</strong> Budget limits, spending rates</li>
                            <li><strong>Operational:</strong> Resource types, action scopes</li>
                            <li><strong>Compliance:</strong> Regulatory requirements</li>
                        </ul>
                    </div>
                </div>
            </div>
        `;
    }

    getPoAMarketingCase() {
        return `
            <div class="learning-content">
                <h3 class="text-xl font-bold mb-4">Marketing PoA Use Case: RAG System</h3>
                <div class="space-y-4">
                    <p class="text-gray-700">ACME Corp deploys a Retrieval-Augmented Generation (RAG) system with PoA credentials for autonomous market research and digital marketing.</p>
                    
                    <div class="bg-blue-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-blue-800 mb-2">RAG System Capabilities:</h4>
                        <ul class="list-disc pl-5 space-y-1 text-blue-700">
                            <li>Analyze market trends in smart manufacturing</li>
                            <li>Monitor regulatory changes in industrial AI</li>
                            <li>Synthesize insights from multiple sources</li>
                            <li>Generate evidence-based recommendations</li>
                        </ul>
                    </div>

                    <div class="bg-green-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-green-800 mb-2">Digital Marketing Agent Powers:</h4>
                        <ul class="list-disc pl-5 space-y-1 text-green-700">
                            <li>Craft content based on trending topics</li>
                            <li>Respond to client queries in real-time</li>
                            <li>Curate user-generated content</li>
                            <li>Manage advertising budgets autonomously</li>
                        </ul>
                    </div>

                    <div class="bg-purple-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-purple-800 mb-2">PoA Enforcement Example:</h4>
                        <div class="text-sm bg-purple-100 p-3 rounded">
                            <strong>Scenario:</strong> Marketing agent wants to target engineers in Southeast Asia with smart manufacturing content<br>
                            <strong>PoA Check:</strong>
                            <ul class="mt-2 ml-4 space-y-1">
                                <li>✅ Target audience: Engineering professionals (allowed)</li>
                                <li>✅ Geographic region: Southeast Asia (in scope)</li>
                                <li>✅ Content topic: Smart manufacturing (approved)</li>
                                <li>✅ Budget impact: €5,000 (within limits)</li>
                                <li>✅ Messaging: Adheres to corporate guidelines</li>
                            </ul>
                            <strong class="text-green-600">Decision: APPROVED - Campaign Launched</strong>
                        </div>
                    </div>

                    <div class="bg-yellow-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-yellow-800 mb-2">Business Value:</h4>
                        <ul class="list-disc pl-5 text-yellow-700">
                            <li>Swift reaction to market demands</li>
                            <li>Automated, evidence-based decisions</li>
                            <li>Dynamic brand presence maintenance</li>
                            <li>Proactive budget management</li>
                        </ul>
                    </div>
                </div>
            </div>
        `;
    }

    // Additional module content methods would continue here...
    getDelegationStep1() {
        return `
            <div class="learning-content">
                <h3 class="text-xl font-bold mb-4">Delegation Concepts</h3>
                <div class="space-y-4">
                    <p class="text-gray-700">Hierarchical delegation allows capabilities to be passed down through organizational structures with appropriate constraints.</p>
                    
                    <div class="bg-indigo-50 p-4 rounded-lg">
                        <h4 class="font-semibold text-indigo-800 mb-2">Delegation Chain Example:</h4>
                        <div class="space-y-2 text-indigo-700">
                            <div class="flex items-center space-x-2">
                                <div class="w-4 h-4 bg-indigo-600 rounded"></div>
                                <span>CEO → CMO (Marketing Authority)</span>
                            </div>
                            <div class="flex items-center space-x-2 ml-4">
                                <div class="w-4 h-4 bg-indigo-500 rounded"></div>
                                <span>CMO → Marketing Director (Regional Authority)</span>
                            </div>
                            <div class="flex items-center space-x-2 ml-8">
                                <div class="w-4 h-4 bg-indigo-400 rounded"></div>
                                <span>Marketing Director → AI Agent (Operational Authority)</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    getDelegationStep2() {
        return `
            <div class="learning-content">
                <h3 class="text-xl font-bold mb-4">Building Delegation Chains</h3>
                <p class="text-gray-700">Learn how to construct and validate delegation chains with proper constraint inheritance.</p>
            </div>
        `;
    }

    getDelegationStep3() {
        return `
            <div class="learning-content">
                <h3 class="text-xl font-bold mb-4">Constraint Propagation</h3>
                <p class="text-gray-700">Understand how constraints flow down delegation chains and are enforced at each level.</p>
            </div>
        `;
    }

    // Revocation, Audit, and RFC-150 step methods would continue...
    getRevocationStep1() {
        return `<div class="learning-content"><h3 class="text-xl font-bold mb-4">Revocation Fundamentals</h3><p>Understanding when and how to revoke delegated capabilities.</p></div>`;
    }

    getRevocationStep2() {
        return `<div class="learning-content"><h3 class="text-xl font-bold mb-4">Cascade Effects</h3><p>How revocation affects downstream delegations.</p></div>`;
    }

    getRevocationStep3() {
        return `<div class="learning-content"><h3 class="text-xl font-bold mb-4">Revocation Strategies</h3><p>Best practices for managing revocation in complex systems.</p></div>`;
    }

    getAuditStep1() {
        return `<div class="learning-content"><h3 class="text-xl font-bold mb-4">Audit Trail Basics</h3><p>Understanding audit requirements and trail generation.</p></div>`;
    }

    getAuditStep2() {
        return `<div class="learning-content"><h3 class="text-xl font-bold mb-4">Compliance Monitoring</h3><p>Automated compliance checking and reporting.</p></div>`;
    }

    getAuditStep3() {
        return `<div class="learning-content"><h3 class="text-xl font-bold mb-4">Regulatory Frameworks</h3><p>Working with GDPR, SOX, and other regulatory requirements.</p></div>`;
    }

    getRFC150Step1() {
        return `<div class="learning-content"><h3 class="text-xl font-bold mb-4">RFC-150 Overview</h3><p>Comprehensive overview of the RFC-150 specification.</p></div>`;
    }

    getRFC150Step2() {
        return `<div class="learning-content"><h3 class="text-xl font-bold mb-4">Protocol Implementation</h3><p>How AgentAuth implements the RFC-150 protocol.</p></div>`;
    }

    getRFC150Step3() {
        return `<div class="learning-content"><h3 class="text-xl font-bold mb-4">Advanced Features</h3><p>Advanced RFC-150 features and capabilities.</p></div>`;
    }

    // Module Management Methods
    async startModule(moduleId) {
        const modules = this.getModules();
        const module = modules[moduleId];
        
        if (!module) {
            console.error('Module not found:', moduleId);
            return;
        }

        this.currentModule = moduleId;
        this.currentStep = 0;
        
        this.renderModule(module);
        this.updateProgress();
    }

    renderModule(module) {
        const container = document.getElementById('learning-content');
        if (!container) return;

        const step = module.steps[this.currentStep];
        
        container.innerHTML = `
            <div class="learning-module">
                <div class="module-header mb-6">
                    <h2 class="text-2xl font-bold text-gray-800">${module.title}</h2>
                    <div class="flex items-center space-x-4 mt-2 text-sm text-gray-600">
                        <span>📚 ${module.difficulty}</span>
                        <span>⏱️ ${module.duration}</span>
                        <span>📍 Step ${this.currentStep + 1} of ${module.steps.length}</span>
                    </div>
                    <div class="progress-bar mt-3">
                        <div class="bg-gray-200 rounded-full h-2">
                            <div class="bg-blue-600 h-2 rounded-full transition-all duration-300" 
                                 style="width: ${((this.currentStep + 1) / module.steps.length) * 100}%"></div>
                        </div>
                    </div>
                </div>

                <div class="step-content mb-6">
                    ${step.content}
                </div>

                <div class="step-controls flex justify-between items-center">
                    <button ${this.currentStep === 0 ? 'disabled' : ''} 
                            data-prev-step 
                            class="px-4 py-2 bg-gray-500 text-white rounded disabled:opacity-50">
                        ← Previous
                    </button>
                    
                    <div class="space-x-2">
                        ${step.exercise ? `
                            <button data-run-exercise="${step.exercise}" 
                                    class="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700">
                                🧪 Try Exercise
                            </button>
                        ` : ''}
                        
                        <button data-next-step 
                                class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
                            ${this.currentStep === module.steps.length - 1 ? 'Complete Module' : 'Next →'}
                        </button>
                    </div>
                </div>

                <div id="exercise-output" class="mt-6 hidden">
                    <div class="bg-gray-50 rounded-lg p-4">
                        <h4 class="font-semibold mb-2">Exercise Output:</h4>
                        <div id="exercise-results" class="text-sm"></div>
                    </div>
                </div>
            </div>
        `;
    }

    nextStep() {
        const modules = this.getModules();
        const module = modules[this.currentModule];
        
        if (this.currentStep < module.steps.length - 1) {
            this.currentStep++;
            this.renderModule(module);
        } else {
            // Module completed
            this.completeModule();
        }
        
        this.updateProgress();
    }

    previousStep() {
        if (this.currentStep > 0) {
            this.currentStep--;
            const modules = this.getModules();
            const module = modules[this.currentModule];
            this.renderModule(module);
            this.updateProgress();
        }
    }

    async runExercise(exerciseId) {
        const outputDiv = document.getElementById('exercise-output');
        const resultsDiv = document.getElementById('exercise-results');
        
        if (!outputDiv || !resultsDiv) return;
        
        outputDiv.classList.remove('hidden');
        resultsDiv.innerHTML = '<div class="text-blue-600">Running exercise...</div>';

        try {
            const result = await this.executeExercise(exerciseId);
            resultsDiv.innerHTML = result;
        } catch (error) {
            resultsDiv.innerHTML = `<div class="text-red-600">Exercise failed: ${error.message}</div>`;
        }
    }

    async executeExercise(exerciseId) {
        switch (exerciseId) {
            case 'basic-authz-check':
                return await this.runBasicAuthzCheck();
            case 'marketing-poa':
                return await this.runMarketingPoADemo();
            case 'poa-validation':
                return await this.runPoAValidation();
            default:
                return `<div class="text-gray-600">Exercise "${exerciseId}" demonstration would run here.</div>`;
        }
    }

    async runBasicAuthzCheck() {
        try {
            const authzData = {
                subject: 'alice@company.com',
                resource: 'financial-report:Q3-2025',
                action: 'read',
                context: {
                    department: 'finance',
                    time: 'business_hours'
                }
            };

            const result = await this.api.evaluateAuthorization(authzData);
            
            return `
                <div class="space-y-2">
                    <div><strong>Request:</strong> ${JSON.stringify(authzData, null, 2)}</div>
                    <div><strong>Decision:</strong> <span class="${result.decision?.allow ? 'text-green-600' : 'text-red-600'}">${result.decision?.allow ? 'ALLOW' : 'DENY'}</span></div>
                    <div><strong>Reason:</strong> ${result.decision?.reason || 'No reason provided'}</div>
                </div>
            `;
        } catch (error) {
            return `<div class="text-red-600">Authorization check failed: ${error.message}</div>`;
        }
    }

    async runMarketingPoADemo() {
        // Simulate marketing PoA validation
        const marketingRequest = {
            agent: 'acme-marketing-ai',
            action: 'social_media_campaign',
            resource: 'linkedin_ads',
            parameters: {
                budget: 45000,
                currency: 'EUR',
                target_audience: 'engineers_southeast_asia',
                content_type: 'smart_manufacturing_trends'
            }
        };

        // Simulate PoA validation logic
        const poaValidation = {
            credential_valid: true,
            budget_check: marketingRequest.parameters.budget <= 50000,
            audience_approved: true,
            content_compliant: true,
            time_window_valid: true
        };

        const decision = Object.values(poaValidation).every(check => check);

        return `
            <div class="space-y-3">
                <div><strong>Marketing Request:</strong></div>
                <pre class="text-xs bg-gray-100 p-2 rounded">${JSON.stringify(marketingRequest, null, 2)}</pre>
                
                <div><strong>PoA Validation:</strong></div>
                <ul class="text-sm space-y-1">
                    ${Object.entries(poaValidation).map(([check, result]) => `
                        <li class="${result ? 'text-green-600' : 'text-red-600'}">
                            ${result ? '✅' : '❌'} ${check.replace(/_/g, ' ')}: ${result ? 'PASS' : 'FAIL'}
                        </li>
                    `).join('')}
                </ul>
                
                <div class="p-3 rounded ${decision ? 'bg-green-50 text-green-800' : 'bg-red-50 text-red-800'}">
                    <strong>Final Decision: ${decision ? 'APPROVED' : 'DENIED'}</strong>
                    <br>Campaign ${decision ? 'launched successfully!' : 'blocked due to policy violations.'}
                </div>
            </div>
        `;
    }

    async runPoAValidation() {
        // Similar to marketing demo but more generic
        return `
            <div class="text-green-600">
                ✅ PoA credential signature verified<br>
                ✅ Validity period check passed<br>  
                ✅ Agent identity confirmed<br>
                ✅ Requested action within scope<br>
                ✅ All constraints satisfied<br>
                <br>
                <strong>Result: Authorization GRANTED</strong>
            </div>
        `;
    }

    completeModule() {
        this.completedModules.add(this.currentModule);
        this.saveProgress();
        
        // Show completion message
        const container = document.getElementById('learning-content');
        if (container) {
            container.innerHTML = `
                <div class="text-center py-12">
                    <div class="text-6xl mb-4">🎉</div>
                    <h2 class="text-2xl font-bold text-green-600 mb-2">Module Completed!</h2>
                    <p class="text-gray-600 mb-6">You've successfully completed this learning module.</p>
                    <button onclick="window.location.reload()" class="px-6 py-3 bg-blue-600 text-white rounded-lg">
                        Return to Learning Path
                    </button>
                </div>
            `;
        }
    }

    updateProgress() {
        // Update progress indicators
        const modules = this.getModules();
        Object.keys(modules).forEach(moduleId => {
            const el = document.querySelector(`[data-module="${moduleId}"] .progress-indicator`);
            if (el) {
                if (this.completedModules.has(moduleId) {
                    el.className = 'progress-indicator completed';
                    el.textContent = '✓';
                } else if (moduleId === this.currentModule) {
                    el.className = 'progress-indicator current';
                    el.textContent = `${this.currentStep + 1}/${modules[moduleId].steps.length}`;
                } else {
                    el.className = 'progress-indicator';
                    el.textContent = '○';
                }
            }
        });
    }

    loadProgress() {
        try {
            const saved = localStorage.getItem('agentauth-learning-progress');
            if (saved) {
                const progress = JSON.parse(saved);
                this.completedModules = new Set(progress.completed || []);
                this.userProgress = progress.user || {};
            }
        } catch (error) {
            console.warn('Failed to load learning progress:', error);
        }
    }

    saveProgress() {
        try {
            const progress = {
                completed: Array.from(this.completedModules),
                user: this.userProgress,
                lastUpdated: new Date().toISOString()
            };
            localStorage.setItem('agentauth-learning-progress', JSON.stringify(progress));
        } catch (error) {
            console.warn('Failed to save learning progress:', error);
        }
    }
}

// Export for use in other modules
window.AgentAuthLearningPath = AgentAuthLearningPath;