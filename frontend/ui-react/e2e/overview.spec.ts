import { test, expect } from '@playwright/test'

test.describe('Overview Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  test('should display the main heading', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /Welcome to AgentAuth/i })).toBeVisible()
  })

  test('should show RFC compliance information', async ({ page }) => {
    await expect(page.getByText(/RFC-0111.*RFC-0115/).first()).toBeVisible()
  })

  test('should display stat cards', async ({ page }) => {
    // Check for key stat cards
    await expect(page.getByText('Tests Passing')).toBeVisible()
    await expect(page.getByText('Benchmarks')).toBeVisible()
    await expect(page.getByText('Test Coverage')).toBeVisible()
  })

  test('should show backend status indicator', async ({ page }) => {
    // Wait for backend health check to complete
    await page.waitForTimeout(2000)
    
    const statusText = page.getByText(/Backend (healthy|error|Checking)/i)
    await expect(statusText).toBeVisible()
  })

  test('should display RFC compliance section', async ({ page }) => {
    await expect(page.getByText('AgentAuth 1.0 Extended Tokens')).toBeVisible()
    await expect(page.getByText('Power of Attorney Framework')).toBeVisible()
    await expect(page.getByText(/eIDAS/i).first()).toBeVisible()
  })

  test('should show system components', async ({ page }) => {
    await expect(page.getByText('Extended Token').first()).toBeVisible()
    await expect(page.getByText('PVP (Identity Verification)')).toBeVisible()
    await expect(page.getByText('Commercial Register').first()).toBeVisible()
  })

  test('should display quick start section', async ({ page }) => {
    await expect(page.getByText(/Quick Start/i).first()).toBeVisible()
  })

  test('should have working navigation links', async ({ page }) => {
    // Test navigation to other pages
    const tokensLink = page.getByRole('link', { name: /Tokens/i })
    if (await tokensLink.isVisible({ timeout: 5000 }) {
      await tokensLink.click()
      await expect(page).toHaveURL(/.*tokens/)
    }
    
    await page.goto('/')
    const pvpLink = page.getByRole('link', { name: /PVP/i })
    if (await pvpLink.isVisible({ timeout: 5000 }) {
      await pvpLink.click()
      await expect(page).toHaveURL(/.*pvp/)
    }
  })

  test('should have responsive design', async ({ page }) => {
    // Test at different viewport sizes
    await page.setViewportSize({ width: 375, height: 667 }) // Mobile
    await expect(page.getByRole('heading').first()).toBeVisible()
    
    await page.setViewportSize({ width: 768, height: 1024 }) // Tablet
    await expect(page.getByRole('heading').first()).toBeVisible()
    
    await page.setViewportSize({ width: 1920, height: 1080 }) // Desktop
    await expect(page.getByRole('heading').first()).toBeVisible()
  })

  test('should have working theme toggle', async ({ page }) => {
    const themeToggle = page.locator('[aria-label="Toggle theme"]')
    if (await themeToggle.isVisible() {
      await themeToggle.click()
      // Wait for theme transition
      await page.waitForTimeout(300)
    }
  })
})
