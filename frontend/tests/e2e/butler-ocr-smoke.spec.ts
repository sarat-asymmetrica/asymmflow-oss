import { expect, test } from '@playwright/test';
import { openGlobalOcrFromDashboard } from './helpers/globalOcr';
import { installMockWailsBridge } from './helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

test.describe('Butler and OCR smoke', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
  });

  test('Butler chat persists across reloads', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Intelligence' }).click();
    await expect(page.getByRole('button', { name: /\+ New Chat/ })).toBeVisible();

    await page.getByRole('button', { name: /\+ New Chat/ }).click();

    const chatInput = page.getByPlaceholder('Ask about financials, customers, suppliers...');
    const prompt = 'Butler smoke test: keep this conversation after reload.';
    await chatInput.fill(prompt);
    await chatInput.press('Enter');

    await expect(page.locator('#chat-feed').getByText(prompt)).toBeVisible();
    await expect(page.locator('#chat-feed').getByText('Mock Butler response persisted.')).toBeVisible();

    await page.goto(BASE_URL);
    await page.getByRole('link', { name: 'Intelligence' }).click();

    await expect(page.locator('#chat-feed').getByText(prompt)).toBeVisible();
    await expect(page.locator('#chat-feed').getByText('Mock Butler response persisted.')).toBeVisible();
  });

  test('OCR modal opens and Butler analysis populates a bank statement', async ({ page }) => {
    await page.goto(BASE_URL);

    await openGlobalOcrFromDashboard(page);
    await expect(page.getByRole('button', { name: 'Save as Bank Statement' })).toBeVisible();

    await expect(page.getByText('AI Summary:')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Mock Butler analysis: one credit transaction and a clean balance trail.')).toBeVisible();

    await page.getByRole('button', { name: /Transactions \(1\)/ }).click();
    await expect(page.locator('.items-table input[type="text"]').first()).toHaveValue('Deposit');

    await page.getByRole('button', { name: 'Report / Summary' }).click();
    await expect(page.getByRole('button', { name: 'Save as Report / Summary' })).toBeVisible();
  });
});
