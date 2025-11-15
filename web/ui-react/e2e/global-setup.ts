/**
 * Global Setup for Playwright Tests
 * Runs once before all tests
 */

async function globalSetup() {
  console.log('🚀 Starting global test setup...')
  
  // Check if backend is available
  const backendUrl = process.env.API_URL || 'http://localhost:8080'
  
  try {
    const response = await fetch(`${backendUrl}/health`, {
      method: 'GET',
      headers: { 'Accept': 'application/json' },
    })
    
    if (response.ok) {
      console.log('✅ Backend health check passed')
    } else {
      console.warn('⚠️  Backend returned non-200 status:', response.status)
    }
  } catch (error) {
    console.warn('⚠️  Backend health check failed:', error)
    console.warn('   Tests will continue but may fail if backend is required')
  }
  
  console.log('✅ Global setup complete')
}

export default globalSetup
