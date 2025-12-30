import { test, expect, Page } from '@playwright/test';

// Capture browser console output for debugging when assertions fail
test.beforeEach(async ({ page }) => {
  page.on('console', msg => {
    // echo through test runner logs
    console.log(`[browser:${msg.type()}]`, msg.text());
  });
});

// Basic smoke test for Policy (Experimental) panel
// Assumes server already running at baseURL.

async function openPolicyTab(page: Page) {
  // Resilient open: poll up to N attempts (handles race with server embedding readiness)
  const maxAttempts = 8;
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    const cacheBust = Date.now();
    await page.goto(`/index.html?cb=${cacheBust}`, { waitUntil: 'domcontentloaded' });
    let tab = page.locator('[data-testid="policy-tab"]');
    if (!(await tab.count()) {
      // Fallback: locate button by visible text
      tab = page.locator('button:has-text("Policy (Experimental)")');
    }
    if (await tab.count() {
      try {
        await tab.first().waitFor({ state: 'visible', timeout: 2000 });
        await tab.first().click();
        return;
      } catch (e) {
        // fallthrough to retry
      }
    }
    // Short delay before retry
    await page.waitForTimeout(500);
  }
  const content = await page.content();
  throw new Error('Failed to open policy tab after retries; page snippet: ' + content.slice(0, 500));
}

test.describe('Policy Engine UI', () => {
  test('provenance -> submit bundle -> evaluate', async ({ page }) => {
    await openPolicyTab(page);

  // Provenance (empty chain) should succeed
  await page.getByRole('button', { name: 'Provenance' }).click();
  // Console should contain success soon (simple heuristic)
	await expect(page.locator('#policy-output')).toContainText(/provenance/i, { timeout: 15000 });

    // Submit bundle (uses prefilled textarea)
  await page.getByRole('button', { name: 'Submit Bundle' }).click();
	await expect(page.locator('#policy-output')).toContainText(/bundle appended head=/i, { timeout: 15000 });

    // Evaluate (prefilled inputs)
  await page.getByRole('button', { name: 'Evaluate' }).click();
  await expect(page.locator('#policy-output')).toContainText(/allow=true/i, { timeout: 15000 });
  });
});
