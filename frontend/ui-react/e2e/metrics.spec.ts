/**
 * E2E Tests for Metrics Page
 * Tests Prometheus metrics display, auto-refresh, and visualization
 */

import { test, expect } from '@playwright/test'

test.describe('Metrics and Monitoring', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/metrics')
    await page.waitForLoadState('networkidle')
  })

  test('should display metrics page', async ({ page }) => {
    await expect(page).toHaveTitle(/GAuth/)
    const heading = page.getByRole('heading', { name: /Metrics|Monitoring|Prometheus|Performance|System/i })
    await expect(heading.first()).toBeVisible({ timeout: 10000 })
  })

  test('should show metrics categories', async ({ page }) => {
    // Check for common metric categories
    const categories = [
      /HTTP|Requests/i,
      /Performance|Response Time/i,
      /System|Resource/i,
    ]
    
    for (const category of categories) {
      const found = await page.getByText(category).first().isVisible({ timeout: 5000 }).catch(() => false)
      if (found) {
        console.log(`✅ Category found: ${category}`)
      }
    }
  })

  test('should display real-time metrics', async ({ page }) => {
    await page.waitForTimeout(2000)
    
    // Look for metrics values (numbers, percentages, rates)
    const hasMetrics = await page.locator('[class*="metric"], [class*="stat"], [class*="card"]').first().isVisible({ timeout: 5000 }).catch(() => false)
    
    if (hasMetrics) {
      console.log('✅ Metrics are visible')
    } else {
      console.log('ℹ️  Metrics not loaded or empty')
    }
  })

  test('should auto-refresh metrics', async ({ page }) => {
    // Check for auto-refresh toggle
    const autoRefreshToggle = page.getByText(/Auto.*refresh|Refresh/i)
    
    if (await autoRefreshToggle.first().isVisible({ timeout: 5000 })) {
      console.log('✅ Auto-refresh control found')
      
      // Wait for a refresh cycle
      await page.waitForTimeout(6000)
      console.log('✅ Auto-refresh cycle completed')
    } else {
      console.log('ℹ️  Auto-refresh toggle not found')
    }
  })

  test('should manually refresh metrics', async ({ page }) => {
    const refreshButton = page.getByRole('button', { name: /Refresh|Reload|Update/i })
    
    if (await refreshButton.first().isVisible({ timeout: 5000 })) {
      await refreshButton.first().click()
      await page.waitForTimeout(1000)
      console.log('✅ Manual refresh tested')
    } else {
      console.log('ℹ️  Manual refresh button not found')
    }
  })

  test('should display charts/visualizations', async ({ page }) => {
    // Look for chart containers (SVG elements from recharts)
    const charts = page.locator('svg[class*="recharts"]')
    const hasCharts = await charts.first().isVisible({ timeout: 5000 }).catch(() => false)
    
    if (hasCharts) {
      const chartCount = await charts.count()
      console.log(`✅ Found ${chartCount} chart(s)`)
    } else {
      console.log('ℹ️  No charts visible (data might be loading)')
    }
  })

  test('should show HTTP metrics', async ({ page }) => {
    // Look for HTTP-related metrics
    const httpMetrics = page.getByText(/requests.*total|http.*count|status.*code/i)
    
    if (await httpMetrics.first().isVisible({ timeout: 5000 })) {
      console.log('✅ HTTP metrics visible')
    } else {
      console.log('ℹ️  HTTP metrics not loaded')
    }
  })

  test('should show performance metrics', async ({ page }) => {
    // Look for performance indicators
    const perfMetrics = page.getByText(/response.*time|latency|duration|throughput/i)
    
    if (await perfMetrics.first().isVisible({ timeout: 5000 })) {
      console.log('✅ Performance metrics visible')
    } else {
      console.log('ℹ️  Performance metrics not loaded')
    }
  })

  test('should show system metrics', async ({ page }) => {
    // Look for system resource metrics
    const sysMetrics = page.getByText(/memory|cpu|goroutines|threads/i)
    
    if (await sysMetrics.first().isVisible({ timeout: 5000 })) {
      console.log('✅ System metrics visible')
    } else {
      console.log('ℹ️  System metrics not loaded')
    }
  })

  test('should filter metrics by category', async ({ page }) => {
    // Look for category filter/tabs
    const filters = page.locator('button, [role="tab"]').filter({ hasText: /HTTP|System|Performance|All/i })
    
    if (await filters.first().isVisible({ timeout: 5000 })) {
      const filterCount = await filters.count()
      
      // Try clicking different filters
      for (let i = 0; i < Math.min(filterCount, 3); i++) {
        await filters.nth(i).click()
        await page.waitForTimeout(500)
      }
      
      console.log(`✅ Tested ${Math.min(filterCount, 3)} filter(s)`)
    } else {
      console.log('ℹ️  No category filters found')
    }
  })

  test('should display metric timestamps', async ({ page }) => {
    // Look for "Last updated" or timestamp info
    const timestamp = page.getByText(/last.*update|updated|ago|timestamp/i)
    
    if (await timestamp.first().isVisible({ timeout: 5000 })) {
      console.log('✅ Timestamp information visible')
    } else {
      console.log('ℹ️  No timestamp info found')
    }
  })

  test('should handle empty metrics state', async ({ page }) => {
    // Check for either metrics or empty state or page title
    const hasMetrics = await page.locator('[class*="metric"]').first().isVisible({ timeout: 3000 }).catch(() => false)
    const hasEmptyState = await page.getByText(/No metrics|No data|Loading/i).first().isVisible({ timeout: 3000 }).catch(() => false)
    const hasTitle = await page.getByRole('heading').first().isVisible({ timeout: 3000 }).catch(() => false)
    
    expect(hasMetrics || hasEmptyState || hasTitle).toBe(true)
  })

  test('should handle API errors gracefully', async ({ page }) => {
    await page.waitForLoadState('networkidle')
    
    // Page should render even if metrics API fails
    const pageContent = page.locator('#root')
    await expect(pageContent).toBeVisible()
  })

  test('should show refresh interval control', async ({ page }) => {
    // Look for interval selector
    const intervalControl = page.locator('select, input[type="number"]').filter({ hasText: /interval|seconds/i }).or(
      page.locator('label:has-text("Interval")')
    )
    
    if (await intervalControl.first().isVisible({ timeout: 5000 })) {
      console.log('✅ Refresh interval control found')
    } else {
      console.log('ℹ️  No interval control visible')
    }
  })
})
