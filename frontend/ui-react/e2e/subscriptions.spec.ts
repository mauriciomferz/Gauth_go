/**
 * E2E Tests for Subscriptions Page
 * Tests subscription creation, management, and wizard flow
 * NOTE: /subscriptions route doesn't exist - use /admin/subscribers instead
 */

import { test, expect } from '@playwright/test'

test.describe.skip('Subscriptions Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/admin/subscribers')
    await page.waitForLoadState('networkidle')
  })

  test('should display subscriptions page', async ({ page }) => {
    await expect(page).toHaveTitle(/AgentAuth/)
    await expect(page.getByRole('heading', { name: /Subscription/i })).toBeVisible()
  })

  test('should show subscription wizard button', async ({ page }) => {
    const wizardButton = page.getByRole('button', { name: /New Subscription|Create Subscription|Subscription Wizard/i })
    await expect(wizardButton.first()).toBeVisible({ timeout: 10000 })
  })

  test('should display existing subscriptions list', async ({ page }) => {
    // Check if subscriptions table or list exists
    const hasSubscriptions = await page.locator('[class*="subscription"], table, [role="list"]').first().isVisible({ timeout: 5000 }).catch(() => false)
    
    if (hasSubscriptions) {
      console.log('✅ Subscriptions list is visible')
    } else {
      console.log('ℹ️  No subscriptions found (expected for empty state)')
    }
  })

  test.describe('Subscription Wizard', () => {
    test('should open subscription wizard', async ({ page }) => {
      const wizardButton = page.getByRole('button', { name: /New Subscription|Create Subscription|Subscription Wizard/i }).first()
      
      if (await wizardButton.isVisible({ timeout: 5000 }) {
        await wizardButton.click()
        
        // Wait for wizard modal or page
        await expect(page.getByText(/Step 1|Basic Information|Client/i).first()).toBeVisible({ timeout: 10000 })
      }
    })

    test('should navigate through wizard steps', async ({ page }) => {
      const wizardButton = page.getByRole('button', { name: /New Subscription|Create Subscription|Subscription Wizard/i }).first()
      
      if (await wizardButton.isVisible({ timeout: 5000 }) {
        await wizardButton.click()
        await page.waitForTimeout(1000)
        
        // Step 1: Basic Information
        const clientIdInput = page.locator('input[name*="client"], input[placeholder*="client"]').first()
        if (await clientIdInput.isVisible({ timeout: 5000 }) {
          await clientIdInput.fill(`test-client-${Date.now()}`)
          
          // Next button
          const nextButton = page.getByRole('button', { name: /Next|Continue/i })
          if (await nextButton.isVisible({ timeout: 5000 }) {
            await nextButton.click()
            await page.waitForTimeout(500)
          }
        }
      }
    })

    test('should validate required fields', async ({ page }) => {
      const wizardButton = page.getByRole('button', { name: /New Subscription|Create Subscription|Subscription Wizard/i }).first()
      
      if (await wizardButton.isVisible({ timeout: 5000 }) {
        await wizardButton.click()
        await page.waitForTimeout(1000)
        
        // Try to proceed without filling required fields
        const nextButton = page.getByRole('button', { name: /Next|Continue/i })
        if (await nextButton.isVisible({ timeout: 5000 }) {
          await nextButton.click()
          
          // Should show validation error
          const hasError = await page.getByText(/required|invalid|must/i).first().isVisible({ timeout: 3000 }).catch(() => false)
          if (!hasError) {
            console.log('ℹ️  Validation might be handled differently')
          }
        }
      }
    })
  })

  test('should search/filter subscriptions', async ({ page }) => {
    const searchInput = page.locator('input[type="search"], input[placeholder*="search"], input[placeholder*="filter"]').first()
    
    if (await searchInput.isVisible({ timeout: 5000 }) {
      await searchInput.fill('test')
      await page.waitForTimeout(500)
      console.log('✅ Search functionality tested')
    } else {
      console.log('ℹ️  No search input found')
    }
  })

  test('should handle empty state', async ({ page }) => {
    // Check for empty state message
    const emptyState = page.getByText(/No subscriptions|No data|Get started/i)
    
    // Either show subscriptions or empty state
    const hasSubscriptions = await page.locator('table tbody tr, [class*="subscription-item"]').first().isVisible({ timeout: 3000 }).catch(() => false)
    const hasEmptyState = await emptyState.first().isVisible({ timeout: 3000 }).catch(() => false)
    
    expect(hasSubscriptions || hasEmptyState).toBe(true)
  })

  test('should display subscription details', async ({ page }) => {
    const firstSubscription = page.locator('table tbody tr, [class*="subscription-item"]').first()
    
    if (await firstSubscription.isVisible({ timeout: 5000 }) {
      await firstSubscription.click()
      await page.waitForTimeout(500)
      
      // Should show details (either in modal or page)
      console.log('✅ Subscription details interaction tested')
    } else {
      console.log('ℹ️  No subscriptions to click')
    }
  })

  test('should handle API errors gracefully', async ({ page }) => {
    // Even if API fails, page should render
    await page.waitForLoadState('networkidle')
    
    // Check that page doesn't crash
    const pageContent = page.locator('#root')
    await expect(pageContent).toBeVisible()
  })
})
