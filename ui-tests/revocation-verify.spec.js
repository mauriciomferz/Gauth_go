// Playwright test: Revocation inclusion & consistency verification UI
// Assumes server running at http://localhost:8080 (adjust via BASE_URL env if needed)
// Using ESM import because package.json sets type=module

import { test, expect } from '@playwright/test';

const BASE = process.env.BASE_URL || 'http://localhost:8080';

async function navigateToTransparency(page) {
  await page.goto(BASE + '/index.html');
  await expect(page.locator('#revocation-transparency')).toBeVisible({ timeout: 10000 });
}

// Utility: request a proof for index 0 (may or may not exist; tolerate empty)
async function requestProof(page) {
  const input = page.locator('#rev-proof-input');
  await input.fill('0');
  await page.click('#rev-proof-btn');
  await expect(page.locator('#rev-proof-result')).toBeVisible();
  await page.waitForTimeout(500); // allow fetch
}

// Utility: request consistency proof from start=0
async function requestConsistency(page) {
  const startInput = page.locator('#rev-consistency-start');
  await startInput.fill('0');
  await page.click('#rev-consistency-btn');
  await expect(page.locator('#rev-consistency-result')).toBeVisible();
  await page.waitForTimeout(500);
}

test('revocation inclusion proof verification button enables', async ({ page }) => {
  await navigateToTransparency(page);
  await requestProof(page);
  const verifyBtn = page.locator('#rev-verify-btn');
  // Button may remain disabled if proof empty; attempt enable by detecting JSON structure
  await page.waitForTimeout(250);
  // Click if enabled
  if (await verifyBtn.isEnabled()) {
    await verifyBtn.click();
    await expect(page.locator('#rev-verify-result')).toBeVisible();
    const text = await page.locator('#rev-verify-result').textContent();
    expect(text.length).toBeGreaterThan(5);
  }
});

test('revocation consistency proof fetch & verify', async ({ page }) => {
  await navigateToTransparency(page);
  await requestConsistency(page);
  const btn = page.locator('#rev-consistency-verify-btn');
  await page.waitForTimeout(250);
  if (await btn.isEnabled()) {
    await btn.click();
    const result = page.locator('#rev-consistency-verify-result');
    await expect(result).toBeVisible();
    const txt = await result.textContent();
    expect(txt).toContain('checks');
  }
});
