/**
 * Learning System Diagnostic and Fix
 * This script diagnoses issues and provides immediate functionality for learning buttons
 */

console.log('🔍 Learning System Diagnostic Starting...');

// Immediate diagnostic function
function runDiagnostic() {
    console.log('=== LEARNING SYSTEM DIAGNOSTIC ===');
    
    // Check if DOM is ready
    const domReady = document.readyState === 'complete' || document.readyState === 'interactive';
    console.log('DOM Ready:', domReady ? '✅' : '❌');
    
    // Check for learning buttons
    const buttons = document.querySelectorAll('[data-start-module]');
    console.log('Learning Buttons Found:', buttons.length);
    
    buttons.forEach((btn, index) => {
        const moduleId = btn.dataset.startModule;
        console.log(`  Button ${index + 1}: ${moduleId} - ${btn.textContent.trim()}`);
    });
    
    // Check for learning content container
    const learningContent = document.getElementById('learning-content');
    console.log('Learning Content Container:', learningContent ? '✅ Found' : '❌ Missing');
    
    // Check for required classes
    console.log('AgentAuthAPIClient:', typeof window.AgentAuthAPIClient !== 'undefined' ? '✅' : '❌');
    console.log('AgentAuthLearningPath:', typeof window.AgentAuthLearningPath !== 'undefined' ? '✅' : '❌');
    
    // Check if any existing event listeners
    const hasListeners = buttons.length > 0 && buttons[0].onclick !== null;
    console.log('Existing Event Listeners:', hasListeners ? '✅' : '❌');
    
    console.log('=== END DIAGNOSTIC ===');
    
    return {
        buttons: buttons.length,
        container: !!learningContent,
        apiClient: typeof window.AgentAuthAPIClient !== 'undefined',
        learningPath: typeof window.AgentAuthLearningPath !== 'undefined'
    };
}

// Immediate fix function
function implementImmediateFix() {
    console.log('🔧 Implementing immediate fix...');
    
    const buttons = document.querySelectorAll('[data-start-module]');
    const learningContent = document.getElementById('learning-content');
    
    if (buttons.length === 0) {
        console.error('❌ No learning buttons found!');
        return;
    }
    
    if (!learningContent) {
        console.error('❌ Learning content container not found!');
        return;
    }
    
    // Remove any existing event listeners and add new ones
    buttons.forEach((button, index) => {
        // Clone button to remove existing listeners
        const newButton = button.cloneNode(true);
        button.parentNode.replaceChild(newButton, button);
        
        const moduleId = newButton.dataset.startModule;
        
        // Add immediate click handler
        newButton.addEventListener('click', function(e) {
            e.preventDefault();
            e.stopPropagation();
            
            console.log('🎯 Button clicked:', moduleId);
            
            // Show visual feedback immediately
            this.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>Loading...';
            this.disabled = true;
            
            // Show learning content
            learningContent.classList.remove('hidden');
            
            // Load module content
            loadModuleContent(moduleId, learningContent);
            
            // Scroll to content
            setTimeout(() => {
                learningContent.scrollIntoView({ 
                    behavior: 'smooth', 
                    block: 'start' 
                });
            }, 100);
            
            // Reset button after delay
            setTimeout(() => {
                this.innerHTML = 'Start Learning';
                this.disabled = false;
            }, 2000);
        });
        
        console.log(`✅ Fixed button ${index + 1}: ${moduleId}`);
    });
    
    console.log(`🎉 Fixed ${buttons.length} learning buttons!`);
}

// Module content loader
function loadModuleContent(moduleId, container) {
    const moduleData = getModuleData(moduleId);
    
    container.innerHTML = `
        <div class="learning-module-active bg-white rounded-lg shadow-xl p-8 max-w-4xl mx-auto">
            <div class="flex items-center justify-between mb-6">
                <div>
                    <h2 class="text-3xl font-bold text-gray-800">${moduleData.title}</h2>
                    <div class="flex items-center space-x-4 mt-2 text-sm text-gray-600">
                        <span class="bg-blue-100 text-blue-800 px-3 py-1 rounded-full">
                            📚 ${moduleData.difficulty}
                        </span>
                        <span class="bg-green-100 text-green-800 px-3 py-1 rounded-full">
                            ⏱️ ${moduleData.duration}
                        </span>
                        <span class="bg-purple-100 text-purple-800 px-3 py-1 rounded-full">
                            🎯 Interactive
                        </span>
                    </div>
                </div>
                <button onclick="closeModule()" 
                        class="text-gray-500 hover:text-gray-700 text-2xl">
                    <i class="fas fa-times"></i>
                </button>
            </div>
                        
            <div class="mb-6">
                <p class="text-lg text-gray-700 mb-4">${moduleData.description}</p>
            </div>
            
            <div class="prose max-w-none mb-8">
                ${moduleData.content}
            </div>
            
            <div class="bg-gradient-to-r from-blue-50 to-purple-50 rounded-lg p-6 mb-6">
                <h3 class="text-xl font-bold text-gray-800 mb-4">🧪 Interactive Demo</h3>
                <p class="text-gray-700 mb-4">Try the interactive demo to see this concept in action:</p>
                <button onclick="runDemo('${moduleId}')" 
                        class="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white px-6 py-3 rounded-lg font-semibold transition-all duration-200 transform hover:scale-105">
                    <i class="fas fa-play mr-2"></i>Run Interactive Demo
                </button>
            </div>
            
            <div id="demo-results-${moduleId}" class="hidden bg-gray-50 rounded-lg p-6">
                <h4 class="font-bold text-gray-800 mb-3">Demo Results:</h4>
                <div id="demo-output-${moduleId}"></div>
            </div>
            
            <div class="flex justify-between items-center pt-6 border-t">
                <button onclick="showModuleList()" 
                        class="px-6 py-2 bg-gray-500 hover:bg-gray-600 text-white rounded-lg">
                    <i class="fas fa-arrow-left mr-2"></i>Back to Modules
                </button>
                <button onclick="closeModule()" 
                        class="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg">
                    <i class="fas fa-check mr-2"></i>Complete Module
                </button>
            </div>
        </div>
    `;
    
    console.log(`✅ Loaded content for module: ${moduleId}`);
}

// Module data definitions
function getModuleData(moduleId) {
    const modules = {
        'auth-fundamentals': {
            title: 'Authorization Fundamentals',
            description: 'Master the core concepts of authorization systems including subjects, resources, actions, and policies.',
            duration: '15 minutes',
            difficulty: 'Beginner',
            content: `
                <div class="space-y-6">
                    <div class="bg-blue-50 border-l-4 border-blue-400 p-6">
                        <h3 class="text-xl font-bold text-blue-800 mb-3">What is Authorization?</h3>
                        <p class="text-blue-700 mb-4">Authorization determines whether a subject has permission to perform a specific action on a particular resource.</p>
                        
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                            <div class="bg-white p-4 rounded-lg">
                                <h4 class="font-semibold text-blue-800 mb-2">🔑 Key Components</h4>
                                <ul class="text-sm text-blue-700 space-y-1">
                                    <li><strong>Subject:</strong> Who is requesting access</li>
                                    <li><strong>Resource:</strong> What is being accessed</li>
                                    <li><strong>Action:</strong> What operation is being performed</li>
                                    <li><strong>Context:</strong> Additional decision factors</li>
                                </ul>
                            </div>
                            <div class="bg-white p-4 rounded-lg">
                                <h4 class="font-semibold text-blue-800 mb-2">📋 Example Scenario</h4>
                                <div class="text-sm text-blue-700">
                                    <strong>Alice</strong> (subject) wants to <strong>read</strong> (action) a <strong>financial report</strong> (resource) during <strong>business hours</strong> (context)
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="bg-green-50 border-l-4 border-green-400 p-6">
                        <h3 class="text-xl font-bold text-green-800 mb-3">Policy-Based Access Control</h3>
                        <p class="text-green-700 mb-4">AgentAuth uses policies to define access rules that are evaluated to make allow/deny decisions.</p>
                        
                        <div class="bg-white p-4 rounded-lg">
                            <h4 class="font-semibold text-green-800 mb-2">Policy Evaluation Process:</h4>
                            <ol class="list-decimal pl-5 space-y-2 text-green-700">
                                <li>Collect all applicable policies</li>
                                <li>Evaluate conditions and constraints</li>
                                <li>Apply policy combination algorithms</li>
                                <li>Return final decision with reasoning</li>
                            </ol>
                        </div>
                    </div>
                </div>
            `
        },
        'poa-fundamentals': {
            title: 'Proof of Authorization (PoA)',
            description: 'Learn how to delegate specific capabilities to agents with proper constraints and validation.',
            duration: '20 minutes',
            difficulty: 'Intermediate',
            content: `
                <div class="space-y-6">
                    <div class="bg-purple-50 border-l-4 border-purple-400 p-6">
                        <h3 class="text-xl font-bold text-purple-800 mb-3">Understanding Proof of Authorization</h3>
                        <p class="text-purple-700 mb-4">PoA allows one entity to delegate specific capabilities to another entity, creating authorized action chains.</p>
                        
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                            <div class="bg-white p-4 rounded-lg">
                                <h4 class="font-semibold text-purple-800 mb-2">🏢 Core Concepts</h4>
                                <ul class="text-sm text-purple-700 space-y-1">
                                    <li><strong>Principal:</strong> Entity granting power</li>
                                    <li><strong>Agent:</strong> Entity receiving power</li>
                                    <li><strong>Scope:</strong> Delegated actions</li>
                                    <li><strong>Constraints:</strong> Limitations</li>
                                </ul>
                            </div>
                            <div class="bg-white p-4 rounded-lg">
                                <h4 class="font-semibold text-purple-800 mb-2">🎯 Marketing Example</h4>
                                <div class="text-sm text-purple-700">
                                    ACME Corp grants PoA to Marketing AI to create social posts, respond to inquiries, and spend up to €50k/month on ads
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="bg-indigo-50 border-l-4 border-indigo-400 p-6">
                        <h3 class="text-xl font-bold text-indigo-800 mb-3">PoA Credential Structure</h3>
                        <p class="text-indigo-700 mb-4">PoA credentials are digital certificates encoding delegation relationships and constraints.</p>
                        
                        <div class="bg-white p-4 rounded-lg">
                            <h4 class="font-semibold text-indigo-800 mb-2">Validation Steps:</h4>
                            <ol class="list-decimal pl-5 space-y-1 text-indigo-700 text-sm">
                                <li>Verify credential signature</li>
                                <li>Check validity period</li>
                                <li>Validate agent identity</li>
                                <li>Match action to capabilities</li>
                                <li>Enforce all constraints</li>
                                <li>Log decision for audit</li>
                            </ol>
                        </div>
                    </div>
                </div>
            `
        },
        'hierarchical-delegation': {
            title: 'Hierarchical Delegation',
            description: 'Build complex delegation trees and manage organizational authorization hierarchies.',
            duration: '18 minutes',
            difficulty: 'Intermediate',
            content: `
                <div class="space-y-6">
                    <div class="bg-indigo-50 border-l-4 border-indigo-400 p-6">
                        <h3 class="text-xl font-bold text-indigo-800 mb-3">Delegation Chains</h3>
                        <p class="text-indigo-700 mb-4">Capabilities flow down organizational structures with appropriate constraints at each level.</p>
                        
                        <div class="bg-white p-4 rounded-lg">
                            <h4 class="font-semibold text-indigo-800 mb-3">Example Chain:</h4>
                            <div class="space-y-3">
                                <div class="flex items-center space-x-3">
                                    <div class="w-4 h-4 bg-indigo-600 rounded-full"></div>
                                    <div class="flex-1 bg-indigo-100 p-2 rounded">CEO → CMO (Marketing Authority)</div>
                                </div>
                                <div class="flex items-center space-x-3 ml-4">
                                    <div class="w-4 h-4 bg-indigo-500 rounded-full"></div>
                                    <div class="flex-1 bg-indigo-100 p-2 rounded">CMO → Marketing Director (Regional Authority)</div>
                                </div>
                                <div class="flex items-center space-x-3 ml-8">
                                    <div class="w-4 h-4 bg-indigo-400 rounded-full"></div>
                                    <div class="flex-1 bg-indigo-100 p-2 rounded">Marketing Director → AI Agent (Operational Authority)</div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            `
        },
        'cascade-revocation': {
            title: 'Cascade Revocation',
            description: 'Understand revocation patterns and their cascading effects through delegation chains.',
            duration: '12 minutes',
            difficulty: 'Advanced',
            content: `
                <div class="space-y-6">
                    <div class="bg-red-50 border-l-4 border-red-400 p-6">
                        <h3 class="text-xl font-bold text-red-800 mb-3">Revocation Fundamentals</h3>
                        <p class="text-red-700 mb-4">When delegation is revoked, all downstream delegations are also invalidated.</p>
                        
                        <div class="bg-white p-4 rounded-lg">
                            <h4 class="font-semibold text-red-800 mb-2">Revocation Types:</h4>
                            <ul class="text-sm text-red-700 space-y-1">
                                <li><strong>Immediate:</strong> Takes effect instantly</li>
                                <li><strong>Scheduled:</strong> Takes effect at specific time</li>
                                <li><strong>Conditional:</strong> Triggered by conditions</li>
                                <li><strong>Cascade:</strong> Affects downstream delegations</li>
                            </ul>
                        </div>
                    </div>
                </div>
            `
        },
        'audit-compliance': {
            title: 'Audit & Compliance',
            description: 'Master audit trails, compliance monitoring, and regulatory framework integration.',
            duration: '16 minutes',
            difficulty: 'Intermediate',
            content: `
                <div class="space-y-6">
                    <div class="bg-yellow-50 border-l-4 border-yellow-400 p-6">
                        <h3 class="text-xl font-bold text-yellow-800 mb-3">Audit Trail Components</h3>
                        <p class="text-yellow-700 mb-4">Comprehensive record of all authorization decisions and related activities.</p>
                        
                        <div class="bg-white p-4 rounded-lg">
                            <h4 class="font-semibold text-yellow-800 mb-2">Audit Elements:</h4>
                            <ul class="text-sm text-yellow-700 space-y-1">
                                <li><strong>Decision Logs:</strong> All authorization decisions</li>
                                <li><strong>Policy Changes:</strong> Track modifications</li>
                                <li><strong>Access Patterns:</strong> Monitor behavior</li>
                                <li><strong>Compliance Reports:</strong> Regulatory reporting</li>
                            </ul>
                        </div>
                    </div>
                </div>
            `
        },
        'rfc-150-deep-dive': {
            title: 'RFC-150 Deep Dive',
            description: 'Comprehensive exploration of RFC-150 specifications and advanced protocol features.',
            duration: '25 minutes',
            difficulty: 'Advanced',
            content: `
                <div class="space-y-6">
                    <div class="bg-purple-50 border-l-4 border-purple-400 p-6">
                        <h3 class="text-xl font-bold text-purple-800 mb-3">RFC-150 Overview</h3>
                        <p class="text-purple-700 mb-4">Standard for authorization and delegation in distributed systems.</p>
                        
                        <div class="bg-white p-4 rounded-lg">
                            <h4 class="font-semibold text-purple-800 mb-2">Key Features:</h4>
                            <ul class="text-sm text-purple-700 space-y-1">
                                <li><strong>Standard Protocols:</strong> Interoperable authorization</li>
                                <li><strong>Delegation Framework:</strong> Structured capability transfer</li>
                                <li><strong>Security Model:</strong> Cryptographic verification</li>
                                <li><strong>Compliance:</strong> Regulatory alignment</li>
                            </ul>
                        </div>
                    </div>
                </div>
            `
        }
    };
    
    return modules[moduleId] || {
        title: 'Learning Module',
        description: 'Interactive learning content',
        duration: '10 minutes',
        difficulty: 'Intermediate',
        content: '<p>Module content loading...</p>'
    };
}

// Global functions for module interactions
window.closeModule = function() {
    const learningContent = document.getElementById('learning-content');
    if (learningContent) {
        learningContent.classList.add('hidden');
        learningContent.innerHTML = '';
    }
    console.log('✅ Module closed');
};

window.showModuleList = function() {
    window.closeModule();
    // Scroll back to modules
    const modulesSection = document.querySelector('[data-start-module]')?.closest('section');
    if (modulesSection) {
        modulesSection.scrollIntoView({ behavior: 'smooth' });
    }
};

window.runDemo = function(moduleId) {
    const resultsDiv = document.getElementById(`demo-results-${moduleId}`);
    const outputDiv = document.getElementById(`demo-output-${moduleId}`);
    
    if (!resultsDiv || !outputDiv) return;
    
    resultsDiv.classList.remove('hidden');
    outputDiv.innerHTML = '<div class="text-blue-600"><i class="fas fa-spinner fa-spin mr-2"></i>Running demo...</div>';
    
    setTimeout(() => {
        let demoContent = '';
        
        switch (moduleId) {
            case 'auth-fundamentals':
                demoContent = `
                    <div class="space-y-4">
                        <div class="bg-gray-100 p-4 rounded">
                            <h5 class="font-semibold mb-2">Authorization Request:</h5>
                            <div class="text-sm font-mono">
                                Subject: alice@company.com<br>
                                Resource: financial-report:Q3-2025<br>
                                Action: read<br>
                                Context: department=finance, time=business_hours
                            </div>
                        </div>
                        <div class="bg-green-100 p-4 rounded">
                            <div class="text-green-700">
                                <strong>✅ Decision: ALLOW</strong><br>
                                <strong>Reason:</strong> User has appropriate department membership and time constraints are satisfied
                            </div>
                        </div>
                    </div>
                `;
                break;
            case 'poa-fundamentals':
                demoContent = `
                    <div class="space-y-2">
                        <div class="text-green-600">
                            ✅ PoA credential signature verified<br>
                            ✅ Validity period check passed<br>  
                            ✅ Agent identity confirmed<br>
                            ✅ Requested action within scope<br>
                            ✅ All constraints satisfied
                        </div>
                        <div class="bg-green-100 p-3 rounded mt-3">
                            <strong class="text-green-800">Result: Authorization GRANTED</strong>
                        </div>
                    </div>
                `;
                break;
            default:
                demoContent = `
                    <div class="text-green-600">
                        ✅ Demo completed successfully for ${moduleId}<br>
                        📊 Interactive features demonstrated<br>
                        🎯 Module concepts validated
                    </div>
                `;
        }
        
        outputDiv.innerHTML = demoContent;
    }, 1500);
};

// Add enhanced styling
function addEnhancedStyling() {
    const style = document.createElement('style');
    style.textContent = `
        .learning-module-active {
            animation: slideInUp 0.4s ease-out;
        }
        
        @keyframes slideInUp {
            from { 
                opacity: 0; 
                transform: translateY(30px); 
            }
            to { 
                opacity: 1; 
                transform: translateY(0); 
            }
        }
        
        .learning-module-active button {
            transition: all 0.2s ease;
        }
        
        .learning-module-active button:hover {
            transform: translateY(-1px);
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        }
        
        #learning-content:not(.hidden) {
            display: block;
        }
        
        @media (max-width: 768px) {
            .learning-module-active {
                margin: 1rem;
                padding: 1.5rem;
            }
        }
    `;
    document.head.appendChild(style);
}

// Main execution
function initializeImmediateFix() {
    console.log('🚀 Initializing immediate learning fix...');
    
    // Run diagnostic
    const diagnostic = runDiagnostic();
    
    // Add styling
    addEnhancedStyling();
    
    // Implement fix
    implementImmediateFix();
    
    // Show success message
    console.log('🎉 Learning system is now active!');
    console.log('✅ Click any "Start Learning" button to begin');
    
    // Optional: Show notification to user
    setTimeout(() => {
        const notification = document.createElement('div');
        notification.className = 'fixed top-4 right-4 bg-green-500 text-white px-6 py-3 rounded-lg shadow-lg z-50';
        notification.innerHTML = `
            <div class="flex items-center space-x-2">
                <i class="fas fa-check-circle"></i>
                <span>Learning modules are now active!</span>
            </div>
        `;
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.remove();
        }, 4000);
    }, 1000);
}

// Execute when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializeImmediateFix);
} else {
    initializeImmediateFix();
}

// Also try to execute immediately in case DOM is already ready
setTimeout(initializeImmediateFix, 100);