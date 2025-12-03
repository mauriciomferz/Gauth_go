// Test script to verify learning functionality
document.addEventListener('DOMContentLoaded', function() {
    console.log('🧪 Learning system test loaded');
    
    // Test 1: Check if required elements exist
    const learningContent = document.getElementById('learning-content');
    const startButtons = document.querySelectorAll('[data-start-module]');
    
    console.log('Learning content element:', learningContent ? '✅ Found' : '❌ Missing');
    console.log('Start module buttons:', startButtons.length, 'found');
    
    // Test 2: Check if classes are available
    console.log('GAuthAPIClient available:', typeof window.GAuthAPIClient !== 'undefined' ? '✅ Yes' : '❌ No');
    console.log('GAuthLearningPath available:', typeof window.GAuthLearningPath !== 'undefined' ? '✅ Yes' : '❌ No');
    
    // Test 3: Check if learning path is initialized
    console.log('window.learningPath initialized:', typeof window.learningPath !== 'undefined' ? '✅ Yes' : '❌ No');
    
    // Test 4: Add click listener to test functionality
    startButtons.forEach((button, index) => {
        const moduleId = button.dataset.startModule;
        console.log(`Button ${index + 1}: ${moduleId}`);
        
        button.addEventListener('click', function(e) {
            console.log('🎯 Button clicked:', moduleId);
            
            // Show learning content
            if (learningContent) {
                learningContent.classList.remove('hidden');
                learningContent.innerHTML = `
                    <div class="bg-blue-50 p-6 rounded-lg">
                        <h3 class="text-xl font-bold text-blue-800 mb-4">✅ Learning Module Test</h3>
                        <p class="text-blue-700 mb-4">Module "${moduleId}" button was clicked successfully!</p>
                        <p class="text-blue-600 text-sm">If you see this message, the basic event handling is working.</p>
                        <button onclick="this.parentElement.parentElement.classList.add('hidden')" 
                                class="mt-4 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
                            Close Test
                        </button>
                    </div>
                `;
                
                // Scroll to learning content
                learningContent.scrollIntoView({ behavior: 'smooth' });
            }
        });
    });
    
    console.log('🎯 Learning system test completed');
});