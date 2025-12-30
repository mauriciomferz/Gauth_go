/**
 * AgentAuth Lab Pattern Explorer - Bridge for HTML onclick handlers
 * This creates the window.AgentAuthLab.PatternExplorer namespace expected by the HTML buttons
 */

// Create AgentAuthLab namespace if it doesn't exist
if (typeof window.AgentAuthLab === 'undefined') {
    window.AgentAuthLab = {};
}

// Pattern Explorer for the HTML onclick handlers
window.AgentAuthLab.PatternExplorer = {
    patterns: {
        'delegation-chain': {
            name: 'Hierarchical Delegation Chain',
            description: 'CEO → CFO → Finance AI delegation pattern',
            visualization: `
                <div class="delegation-flow">
                    <div class="flow-step">
                        <div class="entity">
                            <div class="avatar bg-blue-600">CEO</div>
                            <div class="label">Chief Executive Officer</div>
                            <div class="permissions">Full Authority</div>
                        </div>
                        <div class="arrow">⬇️</div>
                    </div>
                    <div class="flow-step">
                        <div class="entity">
                            <div class="avatar bg-green-600">CFO</div>
                            <div class="label">Chief Financial Officer</div>
                            <div class="permissions">Financial Operations</div>
                        </div>
                        <div class="arrow">⬇️</div>
                    </div>
                    <div class="flow-step">
                        <div class="entity">
                            <div class="avatar bg-purple-600">AI</div>
                            <div class="label">Finance AI Agent</div>
                            <div class="permissions">Report Generation, Analysis</div>
                        </div>
                    </div>
                </div>
            `,
            parameters: {
                ceo: 'john.doe@acme.com',
                cfo: 'jane.smith@acme.com',
                aiAgent: 'finance-ai-001',
                scope: 'financial-reports',
                duration: '90 days',
                budget: '€50,000'
            }
        },
        'hierarchical': {
            name: 'Hierarchical Authorization Pattern',
            description: 'Multi-level approval workflow simulation',
            visualization: `
                <div class="hierarchy-structure">
                    <div class="level level-1">
                        <div class="role-card manager">
                            <i class="fas fa-user-tie"></i>
                            <span>Manager</span>
                            <div class="permissions">Approve < €10K</div>
                        </div>
                    </div>
                    <div class="level level-2">
                        <div class="role-card director">
                            <i class="fas fa-user-crown"></i>
                            <span>Director</span>
                            <div class="permissions">Approve < €50K</div>
                        </div>
                    </div>
                    <div class="level level-3">
                        <div class="role-card exec">
                            <i class="fas fa-user-shield"></i>
                            <span>Executive</span>
                            <div class="permissions">Unlimited Authority</div>
                        </div>
                    </div>
                </div>
            `,
            parameters: {
                requestAmount: '€25,000',
                requestType: 'Capital Expenditure',
                requester: 'alice@acme.com',
                approvalLevel: 'Director',
                priority: 'High'
            }
        }
    },

    currentPattern: null,
    simulationActive: false,

    showPattern: function(patternId) {
        console.log('🎯 Load/Simulate Pattern clicked:', patternId);
        
        this.currentPattern = patternId;
        const pattern = this.patterns[patternId];
        
        if (!pattern) {
            this.showError(`Pattern "${patternId}" not found`);
            return;
        }

        this.displayPatternModal(pattern, patternId);
    },

    displayPatternModal: function(pattern, patternId) {
        // Remove existing modal if present
        const existingModal = document.getElementById('pattern-explorer-modal');
        if (existingModal) {
            existingModal.remove();
        }

        // Create modal
        const modal = document.createElement('div');
        modal.id = 'pattern-explorer-modal';
        modal.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4';
        
        modal.innerHTML = `
            <div class="bg-white rounded-2xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-hidden">
                <!-- Modal Header -->
                <div class="bg-gradient-to-r from-purple-600 to-blue-600 text-white p-6">
                    <div class="flex justify-between items-center">
                        <div>
                            <h2 class="text-2xl font-bold">${pattern.name}</h2>
                            <p class="text-purple-100 mt-1">${pattern.description}</p>
                        </div>
                        <button onclick="this.closest('#pattern-explorer-modal').remove()" 
                                class="text-white hover:text-gray-200 text-2xl font-bold leading-none">
                            ×
                        </button>
                    </div>
                </div>

                <!-- Modal Content -->
                <div class="p-6 overflow-y-auto max-h-[70vh]">
                    <div class="grid md:grid-cols-2 gap-8">
                        <!-- Visualization Section -->
                        <div>
                            <h3 class="text-xl font-bold text-gray-800 mb-4 flex items-center">
                                <i class="fas fa-project-diagram text-purple-600 mr-2"></i>
                                Pattern Visualization
                            </h3>
                            <div class="bg-gray-50 rounded-lg p-6 border">
                                ${pattern.visualization}
                            </div>
                        </div>

                        <!-- Parameters Section -->
                        <div>
                            <h3 class="text-xl font-bold text-gray-800 mb-4 flex items-center">
                                <i class="fas fa-cogs text-blue-600 mr-2"></i>
                                Pattern Parameters
                            </h3>
                            <div class="space-y-3">
                                ${Object.entries(pattern.parameters).map(([key, value]) => `
                                    <div class="bg-blue-50 p-3 rounded-lg border border-blue-200">
                                        <div class="flex justify-between items-center">
                                            <span class="font-medium text-blue-800 capitalize">${key.replace(/([A-Z])/g, ' $1')}</span>
                                            <span class="text-blue-600 font-mono text-sm">${value}</span>
                                        </div>
                                    </div>
                                `).join('')}
                            </div>
                        </div>
                    </div>

                    <!-- Simulation Section -->
                    <div class="mt-8">
                        <h3 class="text-xl font-bold text-gray-800 mb-4 flex items-center">
                            <i class="fas fa-play-circle text-green-600 mr-2"></i>
                            Interactive Simulation
                        </h3>
                        
                        <div class="bg-gradient-to-r from-green-50 to-blue-50 p-6 rounded-lg border">
                            <p class="text-gray-700 mb-4">
                                Experience this authorization pattern in real-time with our interactive simulator.
                            </p>
                            
                            <div class="grid md:grid-cols-3 gap-4 mb-6">
                                <button onclick="window.AgentAuthLab.PatternExplorer.startSimulation('${patternId}')"
                                        class="bg-green-600 hover:bg-green-700 text-white font-semibold py-3 px-4 rounded-lg transition-colors">
                                    <i class="fas fa-rocket mr-2"></i>
                                    Start Simulation
                                </button>
                                <button onclick="window.AgentAuthLab.PatternExplorer.modifyParameters('${patternId}')"
                                        class="bg-blue-600 hover:bg-blue-700 text-white font-semibold py-3 px-4 rounded-lg transition-colors">
                                    <i class="fas fa-edit mr-2"></i>
                                    Modify Parameters
                                </button>
                                <button onclick="window.AgentAuthLab.PatternExplorer.exportPattern('${patternId}')"
                                        class="bg-purple-600 hover:bg-purple-700 text-white font-semibold py-3 px-4 rounded-lg transition-colors">
                                    <i class="fas fa-download mr-2"></i>
                                    Export Config
                                </button>
                            </div>

                            <!-- Simulation Output -->
                            <div id="simulation-output-${patternId}" class="hidden">
                                <div class="bg-gray-900 text-green-400 p-4 rounded-lg font-mono text-sm">
                                    <div class="text-blue-400 mb-2">🔧 AgentAuth Pattern Simulator v2.0</div>
                                    <div id="simulation-log-${patternId}"></div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Modal Footer -->
                <div class="bg-gray-50 px-6 py-4 flex justify-between items-center">
                    <div class="text-sm text-gray-600">
                        <i class="fas fa-info-circle mr-1"></i>
                        Pattern simulation runs in safe sandbox environment
                    </div>
                    <div class="space-x-3">
                        <button onclick="this.closest('#pattern-explorer-modal').remove()" 
                                class="px-4 py-2 bg-gray-500 hover:bg-gray-600 text-white rounded transition-colors">
                            Close
                        </button>
                        <button onclick="window.AgentAuthLab.PatternExplorer.savePattern('${patternId}')"
                                class="px-4 py-2 bg-purple-600 hover:bg-purple-700 text-white rounded transition-colors">
                            Save to Workspace
                        </button>
                    </div>
                </div>
            </div>
        `;

        document.body.appendChild(modal);
        
        // Add styles for pattern visualization
        this.addPatternStyles();
        
        // Show success notification
        this.showNotification(`✅ Pattern "${pattern.name}" loaded successfully!`, 'success');
    },

    startSimulation: function(patternId) {
        if (this.simulationActive) {
            this.showNotification('⚠️ Simulation already running', 'warning');
            return;
        }

        const pattern = this.patterns[patternId];
        const outputDiv = document.getElementById(`simulation-output-${patternId}`);
        const logDiv = document.getElementById(`simulation-log-${patternId}`);
        
        if (!outputDiv || !logDiv) return;
        
        outputDiv.classList.remove('hidden');
        logDiv.innerHTML = '<div class="text-blue-400">🚀 Initializing pattern simulation...</div>';
        
        this.simulationActive = true;
        this.runPatternSimulation(patternId, logDiv);
    },

    async runPatternSimulation(patternId, logDiv) {
        const pattern = this.patterns[patternId];
        
        let steps = [];
        
        if (patternId === 'delegation-chain') {
            steps = [
                '📋 Loading delegation chain configuration...',
                '🔐 Validating CEO credentials (john.doe@acme.com)...',
                '✅ CEO authentication successful',
                '🔄 Processing delegation: CEO → CFO',
                '👤 Delegating financial operations authority to CFO...',
                '✅ CFO (jane.smith@acme.com) delegation confirmed',
                '🔄 Processing sub-delegation: CFO → Finance AI',
                '🤖 Initializing Finance AI Agent (finance-ai-001)...',
                '📊 Granting report generation permissions...',
                '📈 Granting financial analysis capabilities...',
                '🔒 Setting budget constraint: €50,000',
                '⏰ Setting duration: 90 days',
                '🔍 Validating delegation chain integrity...',
                '✅ All delegation links verified',
                '📝 Recording delegation in audit log...',
                '🎯 Simulating AI report generation request...',
                '🔐 Checking AI agent permissions...',
                '✅ Permission granted: Generate financial report',
                '📊 Report generated: Q4_Financial_Analysis.pdf',
                '📈 Report includes: Revenue trends, cost analysis, forecasts',
                '✅ Delegation chain simulation completed successfully!'
            ];
        } else if (patternId === 'hierarchical') {
            steps = [
                '📋 Initializing hierarchical authorization simulation...',
                '💰 Processing request: €25,000 Capital Expenditure',
                '👤 Request submitted by: alice@acme.com',
                '🔍 Determining required approval level...',
                '📊 Amount: €25,000 > Manager limit (€10,000)',
                '⬆️  Escalating to Director level...',
                '👔 Director approval required for €25,000 request',
                '🔐 Validating Director credentials...',
                '✅ Director authentication successful',
                '📋 Review criteria:',
                '   • Budget impact: Medium',
                '   • Strategic alignment: High',
                '   • Risk assessment: Low',
                '   • ROI projection: 24 months',
                '⏱️  Request under review...',
                '💡 Director decision: APPROVED',
                '📝 Approval reason: "Strategic IT infrastructure upgrade"',
                '✅ Capital expenditure authorized',
                '📧 Notification sent to requester',
                '📊 Budget allocation updated',
                '🔒 Transaction logged in financial system',
                '✅ Hierarchical approval simulation completed!'
            ];
        }

        for (let i = 0; i < steps.length; i++) {
            await this.delay(600);
            logDiv.innerHTML += `<div class="mb-1">${steps[i]}</div>`;
            logDiv.scrollTop = logDiv.scrollHeight;
        }

        this.simulationActive = false;
        
        // Show final success message
        logDiv.innerHTML += `
            <div class="mt-4 p-3 bg-green-800 text-green-200 rounded border border-green-600">
                🎉 Simulation completed successfully!<br>
                📊 Pattern behavior validated<br>
                ⏱️  Total execution time: ${(steps.length * 0.6).toFixed(1)}s
            </div>
        `;
        
        this.showNotification('🎉 Pattern simulation completed successfully!', 'success');
    },

    modifyParameters: function(patternId) {
        const pattern = this.patterns[patternId];
        
        // Create parameter editor modal
        const paramModal = document.createElement('div');
        paramModal.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-60 p-4';
        
        paramModal.innerHTML = `
            <div class="bg-white rounded-lg shadow-xl max-w-2xl w-full max-h-[80vh] overflow-hidden">
                <div class="bg-blue-600 text-white p-4">
                    <h3 class="text-xl font-bold">Modify Pattern Parameters</h3>
                    <p class="text-blue-100 mt-1">${pattern.name}</p>
                </div>
                
                <div class="p-6 overflow-y-auto max-h-96">
                    <div class="space-y-4">
                        ${Object.entries(pattern.parameters).map(([key, value]) => `
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1 capitalize">
                                    ${key.replace(/([A-Z])/g, ' $1')}
                                </label>
                                <input type="text" 
                                       value="${value}" 
                                       data-param-key="${key}"
                                       class="w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500">
                            </div>
                        `).join('')}
                    </div>
                </div>
                
                <div class="bg-gray-50 px-6 py-4 flex justify-end space-x-3">
                    <button onclick="this.closest('.fixed').remove()" 
                            class="px-4 py-2 bg-gray-500 hover:bg-gray-600 text-white rounded transition-colors">
                        Cancel
                    </button>
                    <button onclick="window.AgentAuthLab.PatternExplorer.applyParameters('${patternId}', this)" 
                            class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors">
                        Apply Changes
                    </button>
                </div>
            </div>
        `;
        
        document.body.appendChild(paramModal);
    },

    applyParameters: function(patternId, button) {
        const modal = button.closest('.fixed');
        const inputs = modal.querySelectorAll('input[data-param-key]');
        
        const newParams = {};
        inputs.forEach(input => {
            newParams[input.dataset.paramKey] = input.value;
        });
        
        // Update pattern parameters
        this.patterns[patternId].parameters = { ...this.patterns[patternId].parameters, ...newParams };
        
        modal.remove();
        this.showNotification('✅ Pattern parameters updated successfully!', 'success');
        
        // Refresh the main pattern modal with new parameters
        this.showPattern(patternId);
    },

    exportPattern: function(patternId) {
        const pattern = this.patterns[patternId];
        const exportData = {
            patternId: patternId,
            name: pattern.name,
            description: pattern.description,
            parameters: pattern.parameters,
            exportedAt: new Date().toISOString(),
            version: '1.0'
        };
        
        const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        
        const a = document.createElement('a');
        a.href = url;
        a.download = `gauth-pattern-${patternId}-${Date.now()}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        
        this.showNotification(`📥 Pattern "${pattern.name}" exported successfully!`, 'success');
    },

    savePattern: function(patternId) {
        const pattern = this.patterns[patternId];
        
        // Save to localStorage for persistence
        const savedPatterns = JSON.parse(localStorage.getItem('gauth-saved-patterns') || '{}');
        savedPatterns[patternId] = {
            ...pattern,
            savedAt: new Date().toISOString()
        };
        localStorage.setItem('gauth-saved-patterns', JSON.stringify(savedPatterns));
        
        this.showNotification(`💾 Pattern "${pattern.name}" saved to workspace!`, 'success');
    },

    showError: function(message) {
        this.showNotification(`❌ Error: ${message}`, 'error');
    },

    showNotification: function(message, type = 'info') {
        // Remove existing notification
        const existing = document.getElementById('pattern-notification');
        if (existing) existing.remove();
        
        const notification = document.createElement('div');
        notification.id = 'pattern-notification';
        notification.className = `fixed top-4 right-4 px-6 py-3 rounded-lg shadow-lg text-white z-50 transition-all duration-300`;
        
        switch (type) {
            case 'success':
                notification.className += ' bg-green-600';
                break;
            case 'error':
                notification.className += ' bg-red-600';
                break;
            case 'warning':
                notification.className += ' bg-yellow-600';
                break;
            default:
                notification.className += ' bg-blue-600';
        }
        
        notification.innerHTML = `
            <div class="flex items-center">
                <span>${message}</span>
                <button onclick="this.parentElement.parentElement.remove()" class="ml-4 text-white hover:text-gray-200">
                    ×
                </button>
            </div>
        `;
        
        document.body.appendChild(notification);
        
        // Auto-remove after 5 seconds
        setTimeout(() => {
            if (notification.parentElement) {
                notification.remove();
            }
        }, 5000);
    },

    addPatternStyles: function() {
        if (document.getElementById('pattern-explorer-styles')) return;
        
        const styles = document.createElement('style');
        styles.id = 'pattern-explorer-styles';
        styles.textContent = `
            .delegation-flow {
                display: flex;
                flex-direction: column;
                align-items: center;
                gap: 1rem;
            }
            
            .flow-step {
                display: flex;
                flex-direction: column;
                align-items: center;
                gap: 0.5rem;
            }
            
            .entity {
                text-align: center;
                padding: 1rem;
                border: 2px solid #e5e7eb;
                border-radius: 0.75rem;
                background: white;
                min-width: 200px;
            }
            
            .avatar {
                width: 60px;
                height: 60px;
                border-radius: 50%;
                display: flex;
                align-items: center;
                justify-content: center;
                color: white;
                font-weight: bold;
                font-size: 1.1rem;
                margin: 0 auto 0.5rem;
            }
            
            .label {
                font-weight: bold;
                color: #374151;
                margin-bottom: 0.25rem;
            }
            
            .permissions {
                font-size: 0.875rem;
                color: #6b7280;
            }
            
            .arrow {
                font-size: 1.5rem;
                color: #9ca3af;
            }
            
            .hierarchy-structure {
                display: flex;
                flex-direction: column;
                gap: 1rem;
            }
            
            .level {
                display: flex;
                justify-content: center;
            }
            
            .role-card {
                display: flex;
                flex-direction: column;
                align-items: center;
                padding: 1rem;
                border-radius: 0.75rem;
                color: white;
                text-align: center;
                min-width: 180px;
            }
            
            .role-card.manager { background: linear-gradient(135deg, #3b82f6, #1d4ed8); }
            .role-card.director { background: linear-gradient(135deg, #10b981, #047857); }
            .role-card.exec { background: linear-gradient(135deg, #8b5cf6, #6d28d9); }
            
            .role-card i {
                font-size: 2rem;
                margin-bottom: 0.5rem;
            }
            
            .role-card span {
                font-weight: bold;
                font-size: 1.1rem;
            }
            
            .role-card .permissions {
                font-size: 0.8rem;
                opacity: 0.9;
                margin-top: 0.25rem;
            }
        `;
        
        document.head.appendChild(styles);
    },

    delay: function(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
};

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    console.log('🔧 AgentAuth Lab Pattern Explorer initialized');
});

console.log('✅ AgentAuth Lab Pattern Explorer bridge loaded');