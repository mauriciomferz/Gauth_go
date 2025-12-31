/**
 * Learning System Enhancement - Ensures full functionality for all learning modules
 * This script provides fallback functionality and debugging for the learning system
 */

document.addEventListener('DOMContentLoaded', function() {
    console.log('🚀 Enhanced Learning System Initializing...');
    
    // Initialize API client if not already done
    if (!window.apiClient) {
        window.apiClient = new AgentAuthAPIClient();
        console.log('✅ API Client initialized');
    }
    
    // Initialize learning path if not already done
    if (!window.learningPath) {
        window.learningPath = new AgentAuthLearningPath(window.apiClient);
        console.log('✅ Learning Path initialized');
    }
    
    // Enhanced event handling for learning modules
    function enhanceLearningButtons() {
        const buttons = document.querySelectorAll('[data-start-module]');
        
        buttons.forEach(button => {
            // Remove any existing listeners to avoid duplicates
            const newButton = button.cloneNode(true);
            button.parentNode.replaceChild(newButton, button);
            
            // Add enhanced click handler
            newButton.addEventListener('click', function(e) {
                e.preventDefault();
                e.stopPropagation();
                
                const moduleId = this.dataset.startModule;
                console.log('🎯 Starting module:', moduleId);
                
                // Show learning content area
                const learningContent = document.getElementById('learning-content');
                if (learningContent) {
                    learningContent.classList.remove('hidden');
                    
                    // Scroll to learning content
                    setTimeout(() => {
                        learningContent.scrollIntoView({ 
                            behavior: 'smooth', 
                            block: 'start' 
                        });
                    }, 100);
                }
                
                // Start the module
                if (window.learningPath && typeof window.learningPath.startModule === 'function') {
                    window.learningPath.startModule(moduleId);
                } else {
                    // Fallback implementation
                    startModuleFallback(moduleId);
                }
                
                // Update button state
                this.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>Loading...';
                this.disabled = true;
                
                setTimeout(() => {
                    this.innerHTML = 'Start Learning';
                    this.disabled = false;
                }, 2000);
            });
        });
        
        console.log(`✅ Enhanced ${buttons.length} learning module buttons`);
    }
    
    // Fallback module implementation for basic functionality
    function startModuleFallback(moduleId) {
        console.log('🔄 Using fallback implementation for:', moduleId);
        
        const moduleContent = getModuleContent(moduleId);
        const learningContent = document.getElementById('learning-content');
        
        if (learningContent && moduleContent) {
            learningContent.innerHTML = `
                <div class="learning-module-fallback">
                    <div class="bg-white rounded-lg shadow-lg p-6">
                        <div class="flex items-center justify-between mb-6">
                            <h2 class="text-2xl font-bold text-gray-800">${moduleContent.title}</h2>
                            <button onclick="this.closest('#learning-content').classList.add('hidden')" 
                                    class="text-gray-500 hover:text-gray-700">
                                <i class="fas fa-times text-xl"></i>
                            </button>
                        </div>
                        
                        <div class="mb-6">
                            <div class="flex items-center space-x-4 text-sm text-gray-600 mb-4">
                                <span class="bg-blue-100 text-blue-800 px-2 py-1 rounded-full">
                                    📚 ${moduleContent.difficulty}
                                </span>
                                <span class="bg-green-100 text-green-800 px-2 py-1 rounded-full">
                                    ⏱️ ${moduleContent.duration}
                                </span>
                            </div>
                            <p class="text-gray-700">${moduleContent.description}</p>
                        </div>
                        
                        <div class="prose max-w-none">
                            ${moduleContent.content}
                        </div>
                        
                        <div class="mt-8 flex justify-between items-center">
                            <button onclick="runModuleDemo('${moduleId}')" 
                                    class="bg-green-600 hover:bg-green-700 text-white px-6 py-2 rounded-lg">
                                <i class="fas fa-play mr-2"></i>Try Interactive Demo
                            </button>
                            <button onclick="this.closest('#learning-content').classList.add('hidden')" 
                                    class="bg-gray-500 hover:bg-gray-600 text-white px-6 py-2 rounded-lg">
                                Close Module
                            </button>
                        </div>
                        
                        <div id="demo-output" class="mt-6 hidden">
                            <div class="bg-gray-50 rounded-lg p-4">
                                <h4 class="font-semibold mb-2">Demo Results:</h4>
                                <div id="demo-results"></div>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }
    }
    
    // Module content definitions
    function getModuleContent(moduleId) {
        const modules = {
            'auth-fundamentals': {
                title: 'Authorization Fundamentals',
                description: 'Learn the core concepts of authorization in AgentAuth',
                duration: '15 minutes',
                difficulty: 'Beginner',
                content: `
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
                    </div>
                `
            },
            'poa-fundamentals': {
                title: 'Proof of Authorization (PoA)',
                description: 'Master delegation and power of attorney concepts',
                duration: '20 minutes',
                difficulty: 'Intermediate',
                content: `
                    <h3 class="text-xl font-bold mb-4">Understanding Proof of Authorization (PoA)</h3>
                    <div class="space-y-4">
                        <p class="text-gray-700">Proof of Authorization in AgentAuth allows one entity to delegate specific capabilities to another entity, creating a chain of authorized actions.</p>
                        
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
                    </div>
                `
            },
            'hierarchical-delegation': {
                title: 'Hierarchical Delegation',
                description: 'Learn about delegation chains and hierarchies',
                duration: '18 minutes',
                difficulty: 'Intermediate',
                content: `
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
                `
            },
            'cascade-revocation': {
                title: 'Cascade Revocation',
                description: 'Understand revocation and its cascading effects',
                duration: '12 minutes',
                difficulty: 'Advanced',
                content: `
                    <h3 class="text-xl font-bold mb-4">Revocation Fundamentals</h3>
                    <div class="space-y-4">
                        <p class="text-gray-700">Cascade revocation ensures that when a delegation is revoked, all downstream delegations are also invalidated.</p>
                        
                        <div class="bg-red-50 p-4 rounded-lg">
                            <h4 class="font-semibold text-red-800 mb-2">Revocation Types:</h4>
                            <ul class="list-disc pl-5 space-y-1 text-red-700">
                                <li><strong>Immediate:</strong> Takes effect instantly</li>
                                <li><strong>Scheduled:</strong> Takes effect at a specific time</li>
                                <li><strong>Conditional:</strong> Takes effect under certain conditions</li>
                                <li><strong>Cascade:</strong> Affects all downstream delegations</li>
                            </ul>
                        </div>
                    </div>
                `
            },
            'audit-compliance': {
                title: 'Audit & Compliance',
                description: 'Learn about audit trails and compliance monitoring',
                duration: '16 minutes',
                difficulty: 'Intermediate',
                content: `
                    <h3 class="text-xl font-bold mb-4">Audit Trail Basics</h3>
                    <div class="space-y-4">
                        <p class="text-gray-700">Audit trails provide a comprehensive record of all authorization decisions and related activities.</p>
                        
                        <div class="bg-yellow-50 p-4 rounded-lg">
                            <h4 class="font-semibold text-yellow-800 mb-2">Audit Components:</h4>
                            <ul class="list-disc pl-5 space-y-1 text-yellow-700">
                                <li><strong>Decision Logs:</strong> Record of all authorization decisions</li>
                                <li><strong>Policy Changes:</strong> Track policy modifications</li>
                                <li><strong>Access Patterns:</strong> Monitor access behavior</li>
                                <li><strong>Compliance Reports:</strong> Regulatory reporting</li>
                            </ul>
                        </div>
                    </div>
                `
            },
            'rfc-150-deep-dive': {
                title: 'RFC-150 Deep Dive',
                description: 'Comprehensive exploration of RFC-150 specifications',
                duration: '25 minutes',
                difficulty: 'Advanced',
                content: `
                    <h3 class="text-xl font-bold mb-4">RFC-150 Overview</h3>
                    <div class="space-y-4">
                        <p class="text-gray-700">RFC-150 defines the standard for authorization and delegation in distributed systems.</p>
                        
                        <div class="bg-purple-50 p-4 rounded-lg">
                            <h4 class="font-semibold text-purple-800 mb-2">RFC-150 Key Features:</h4>
                            <ul class="list-disc pl-5 space-y-1 text-purple-700">
                                <li><strong>Standard Protocols:</strong> Interoperable authorization</li>
                                <li><strong>Delegation Framework:</strong> Structured capability transfer</li>
                                <li><strong>Security Model:</strong> Cryptographic verification</li>
                                <li><strong>Compliance:</strong> Regulatory framework alignment</li>
                            </ul>
                        </div>
                    </div>
                `
            }
        };
        
        return modules[moduleId];
    }
    
    // Demo runner function
    window.runModuleDemo = function(moduleId) {
        const outputDiv = document.getElementById('demo-output');
        const resultsDiv = document.getElementById('demo-results');
        
        if (!outputDiv || !resultsDiv) return;
        
        outputDiv.classList.remove('hidden');
        resultsDiv.innerHTML = '<div class="text-blue-600"><i class="fas fa-spinner fa-spin mr-2"></i>Running demo...</div>';
        
        setTimeout(() => {
            let demoResult = '';
            
            switch (moduleId) {
                case 'auth-fundamentals':
                    demoResult = `
                        <div class="space-y-2">
                            <div><strong>Authorization Check:</strong></div>
                            <div class="bg-gray-100 p-2 rounded text-sm">
                                Subject: alice@company.com<br>
                                Resource: financial-report:Q3-2025<br>
                                Action: read<br>
                                Context: department=finance, time=business_hours
                            </div>
                            <div><strong>Decision:</strong> <span class="text-green-600">ALLOW</span></div>
                            <div><strong>Reason:</strong> User has appropriate department membership and time constraints met</div>
                        </div>
                    `;
                    break;
                case 'poa-fundamentals':
                    demoResult = `
                        <div class="space-y-2">
                            <div><strong>PoA Validation Demo:</strong></div>
                            <div class="text-green-600">
                                ✅ PoA credential signature verified<br>
                                ✅ Validity period check passed<br>  
                                ✅ Agent identity confirmed<br>
                                ✅ Requested action within scope<br>
                                ✅ All constraints satisfied<br>
                            </div>
                            <div><strong>Result:</strong> <span class="text-green-600">Authorization GRANTED</span></div>
                        </div>
                    `;
                    break;
                default:
                    demoResult = `
                        <div class="text-green-600">
                            ✅ Demo completed successfully for ${moduleId}<br>
                            📊 Interactive features would be available in the full implementation<br>
                            🎯 Module content delivered and validated
                        </div>
                    `;
            }
            
            resultsDiv.innerHTML = demoResult;
        }, 1500);
    };
    
    // Initialize enhanced functionality
    enhanceLearningButtons();
    
    // Add CSS for better styling
    const style = document.createElement('style');
    style.textContent = `
        .learning-module-fallback {
            animation: fadeIn 0.3s ease-in;
        }
        
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }
        
        .progress-indicator.completed {
            background-color: #10b981;
            color: white;
        }
        
        .progress-indicator.current {
            background-color: #3b82f6;
            color: white;
        }
        
        #learning-content:not(.hidden) {
            display: block;
            animation: slideIn 0.4s ease-out;
        }
        
        @keyframes slideIn {
            from { opacity: 0; transform: translateY(-10px); }
            to { opacity: 1; transform: translateY(0); }
        }
    `;
    document.head.appendChild(style);
    
    console.log('🎉 Enhanced Learning System Ready!');
});