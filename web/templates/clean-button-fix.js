// CLEAN EMERGENCY BUTTON FIX - Direct event listeners as backup  
function emergencyButtonFix() {
    console.log('EMERGENCY BUTTON FIX - Setting up direct event listeners...');
    
    // Get all main buttons
    const buttons = {
        startLearning: document.querySelector('[data-action="start-learning-path"]'),
        quickCompliance: document.querySelector('[data-action="quick-compliance-check"]'), 
        createToken: document.querySelector('[data-action="create-token"]')
    };
    
    // Add direct click handlers as backup
    if (buttons.startLearning) {
        console.log('Adding direct handler to Start Learning button');
        buttons.startLearning.addEventListener('click', function(e) {
            e.preventDefault();
            e.stopPropagation();
            console.log('DIRECT: Start Learning Journey button clicked!');
            if (typeof handleStartLearningPath === 'function') {
                handleStartLearningPath(this, this.innerHTML, this.className);
            } else {
                console.log('Starting learning path...');
                alert('Starting Learning Journey!');
            }
        });
    }
    
    if (buttons.quickCompliance) {
        console.log('Adding direct handler to Quick Compliance button');
        buttons.quickCompliance.addEventListener('click', function(e) {
            e.preventDefault();
            e.stopPropagation();
            console.log('DIRECT: Quick Compliance Check button clicked!');
            if (typeof handleQuickComplianceCheck === 'function') {
                handleQuickComplianceCheck(this, this.innerHTML, this.className);
            } else {
                console.log('Starting compliance check...');
                alert('Starting Quick Compliance Check!');
            }
        });
    }
    
    if (buttons.createToken) {
        console.log('Adding direct handler to Create Token button');
        buttons.createToken.addEventListener('click', function(e) {
            e.preventDefault();
            e.stopPropagation();
            console.log('DIRECT: Create Token button clicked!');
            if (typeof handleCreateToken === 'function') {
                handleCreateToken(this, this.innerHTML, this.className);
            } else {
                console.log('Creating demo token...');
                alert('Creating Demo Token!');
            }
        });
    }
    
    // Add handlers for all other data-action buttons
    const allActionButtons = document.querySelectorAll('[data-action]');
    console.log('Found ' + allActionButtons.length + ' buttons with data-action attributes');
    
    allActionButtons.forEach(function(button, index) {
        const action = button.getAttribute('data-action');
        console.log('Setting up direct handler for button ' + (index + 1) + ': ' + action);
        
        // Skip if already handled above
        if (['start-learning-path', 'quick-compliance-check', 'create-token'].includes(action)) {
            return;
        }
        
        button.addEventListener('click', function(e) {
            e.preventDefault();
            e.stopPropagation();
            console.log('DIRECT: Action button clicked: ' + action);
            if (typeof handleGenericAction === 'function') {
                handleGenericAction(this, action, this.innerHTML, this.className);
            } else {
                console.log('Generic action triggered:', action);
                alert('Button clicked: ' + action);
            }
        });
    });
    
    console.log('EMERGENCY BUTTON FIX COMPLETE - All buttons should now work!');
}

// Test button detection and apply emergency fix
setTimeout(function() {
    console.log('BUTTON TEST - Looking for main buttons...');
    const testButtons = [
        document.querySelector('[data-action="start-learning-path"]'),
        document.querySelector('[data-action="quick-compliance-check"]'), 
        document.querySelector('[data-action="create-token"]')
    ];
    
    testButtons.forEach(function(btn, i) {
        console.log('Button ' + (i+1) + ':', btn ? 'FOUND' : 'NOT FOUND', btn ? btn.getAttribute('data-action') : '');
    });
    
    // Apply emergency fix
    emergencyButtonFix();
}, 1000);