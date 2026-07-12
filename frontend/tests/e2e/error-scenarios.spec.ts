import { expect, test } from '@playwright/test';
import { installMockWailsBridge } from './helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

test.describe('Workflow guardrails', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
  });

  test('clearing all Butler chats removes persisted threads', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Intelligence' }).click();
    const chatInput = page.getByPlaceholder('Ask about financials, customers, suppliers...');

    await chatInput.fill('First workflow guardrail prompt');
    await chatInput.press('Enter');
    await expect(page.locator('#chat-feed').getByText('Mock Butler response persisted.')).toBeVisible();

    const clearAllButton = page.getByRole('button', { name: /Clear All Chats|Click again to confirm/ });
    await clearAllButton.click();
    await expect(clearAllButton).toHaveText(/Click again to confirm/);
    await clearAllButton.click();

    await expect(page.getByText('All conversations have been cleared.')).toBeVisible();
    await expect(page.locator('.conv-item')).toHaveCount(0);
    await expect(page.getByText('No conversations yet')).toBeVisible();
    await expect(chatInput).toBeVisible();
  });

  test('opportunity detail modal closes cleanly with Escape and keeps the list usable', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Opportunities' }).click();
    await page.getByRole('button', { name: /ACTIUM PG/i }).first().click();

    await expect(page.locator('#opportunity-modal-title')).toHaveText(/ACTIUM PG \/ 2026-311/i);
    await page.keyboard.press('Escape');

    await expect(page.getByRole('dialog')).toHaveCount(0);
    await expect(page.getByRole('button', { name: /ACTIUM PG/i }).first()).toBeVisible();
  });
});
