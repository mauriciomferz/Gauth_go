// Playwright test: Revocation Transparency panel proof generation
// Minimal smoke test to ensure the panel renders and proof request handles empty input gracefully.
// Extend later to create a delegation & revoke to produce real chain entries.

import { test, expect } from '@playwright/test';

// Assumption: server running locally on GAUTH_WEB_PORT (default 8080 or overridden to 9090). Use 9090 if set.
const BASE = process.env.GAUTH_WEB_PORT ? `http://localhost:${process.env.GAUTH_WEB_PORT}` : 'http://localhost:9090';

test.describe('Revocation Transparency Panel', () => {
  test('renders panel and handles empty proof input', async ({ page }) => {
    await page.goto(BASE);
    // Wait for panel heading
    const panel = page.locator('#revocation-transparency');
    await expect(panel).toBeVisible();

    // Ensure core fields appear (may still be loading but element should exist)
    await expect(page.locator('#rev-chain-head')).toBeVisible();
    await expect(page.locator('#rev-chain-length')).toBeVisible();

    // Click proof with empty input
    await page.click('#rev-proof-btn');
    const result = page.locator('#rev-proof-result');
    await expect(result).toContainText('(enter id | index | hash)');
  });
});
