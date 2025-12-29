/**
 * Setup tasks that run before each test file
 * Use this for authentication, data seeding, etc.
 */

import { test as setup } from '@playwright/test'

const _authFile = 'playwright/.auth/user.json'

setup('prepare test environment', async () => {
  console.log('📝 Running test setup...')

  // Example: Pre-authenticate user if needed
  // await page.goto('/login')
  // await page.fill('input[name="username"]', 'test-user')
  // await page.fill('input[name="password"]', 'test-password')
  // await page.click('button[type="submit"]')
  // await page.waitForURL('/dashboard')
  // await page.context().storageState({ path: authFile })

  console.log('✅ Test setup complete')
})
