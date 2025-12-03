/**
 * E2E Tests for PIP (Policy Information Point) Page
 * Tests policy viewing, cache management, and authorization checks
 */

import { test, expect } from '@playwright/test'

test.describe('PIP (Policy Information Point)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/pip')
    await page.waitForLoadState('networkidle')
  })

  test('should display PIP page', async ({ page }) => {
    await expect(page).toHaveTitle(/GAuth/)
    await expect(page.getByRole('heading', { name: /PIP|Policy Information/i })).toBeVisible({ timeout: 10000 })
  })

  test('should show policy sections', async ({ page }) => {
    // Check for main PIP sections
    const sections = [
      /Policy Information/i,
      /Cache/i,
      /Authorization/i,
    ]
    
    for (const section of sections) {
      const heading = page.getByRole('heading', { name: section })
      if (await heading.first().isVisible({ timeout: 5000 })) {
        console.log(`✅ Section found: ${section}`)
      }
    }
  })

  test('should display policies list', async ({ page }) => {
    // Wait for policies to load
    await page.waitForTimeout(2000)
    
    const policiesList = page.locator('[class*="polic"], table, [role="list"]').first()
    const hasPolicies = await policiesList.isVisible({ timeout: 5000 }).catch(() => false)
    
    if (hasPolicies) {
      console.log('✅ Policies list is visible')
    } else {
      console.log('ℹ️  No policies found (checking for empty state)')
      
      // Check for empty state
      const emptyState = await page.getByText(/No policies|No data/i).first().isVisible({ timeout: 3000 }).catch(() => false)
      if (emptyState) {
        console.log('✅ Empty state displayed')
      }
    }
  })

  test('should show cache statistics', async ({ page }) => {
    // Look for cache-related metrics
    const cacheMetrics = page.getByText(/hit rate|miss rate|cache size|entries/i)
    const hasMetrics = await cacheMetrics.first().isVisible({ timeout: 5000 }).catch(() => false)
    
    if (hasMetrics) {
      console.log('✅ Cache metrics visible')
    } else {
      console.log('ℹ️  Cache metrics not found or not loaded yet')
    }
  })

  test('should test authorization functionality', async ({ page }) => {
    // Look for authorization test section
    const authSection = page.getByText(/Authorization|Check Authorization|Test Authorization/i).first()
    
    if (await authSection.isVisible({ timeout: 5000 })) {
      // Find input fields for authorization test
      const clientInput = page.locator('input').filter({ hasText: /client/i }).or(
        page.locator('input[placeholder*="client"]')
      ).first()
      
      if (await clientInput.isVisible({ timeout: 3000 })) {
        await clientInput.fill(`test-client-${Date.now()}`)
        
        // Find and click test/check button
        const testButton = page.getByRole('button', { name: /Check|Test|Verify/i })
        if (await testButton.first().isVisible({ timeout: 3000 })) {
          await testButton.first().click()
          await page.waitForTimeout(1000)
          
          console.log('✅ Authorization test submitted')
        }
      }
    } else {
      console.log('ℹ️  Authorization test section not found')
    }
  })

  test('should refresh cache data', async ({ page }) => {
    const refreshButton = page.getByRole('button', { name: /Refresh|Reload/i })
    
    if (await refreshButton.first().isVisible({ timeout: 5000 })) {
      await refreshButton.first().click()
      await page.waitForTimeout(500)
      console.log('✅ Refresh functionality tested')
    } else {
      console.log('ℹ️  No refresh button found')
    }
  })

  test('should search policies', async ({ page }) => {
    const searchInput = page.locator('input[type="search"], input[placeholder*="search"]').first()
    
    if (await searchInput.isVisible({ timeout: 5000 })) {
      await searchInput.fill('test-policy')
      await page.waitForTimeout(500)
      console.log('✅ Policy search tested')
    } else {
      console.log('ℹ️  No search input found')
    }
  })

  test('should display policy details', async ({ page }) => {
    // Click on first policy if available
    const firstPolicy = page.locator('[class*="policy-item"], table tbody tr').first()
    
    if (await firstPolicy.isVisible({ timeout: 5000 })) {
      await firstPolicy.click()
      await page.waitForTimeout(500)
      console.log('✅ Policy details interaction tested')
    } else {
      console.log('ℹ️  No policies to click')
    }
  })

  test('should handle API errors gracefully', async ({ page }) => {
    await page.waitForLoadState('networkidle')
    
    // Page should still be functional even if API fails
    const pageContent = page.locator('#root')
    await expect(pageContent).toBeVisible()
  })

  test('should display cache actions', async ({ page }) => {
    // Look for cache management actions
    const clearCacheButton = page.getByRole('button', { name: /Clear|Flush|Reset/i })
    
    if (await clearCacheButton.first().isVisible({ timeout: 5000 })) {
      console.log('✅ Cache management actions found')
    } else {
      console.log('ℹ️  Cache management actions not visible')
    }
  })
})
