import { expect, test } from '@playwright/test';
import { installMockWailsBridge } from './helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

test.describe('Application shell setup', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
  });

  test('loads the dashboard shell and primary navigation', async ({ page }) => {
    await page.goto(BASE_URL);

    await expect(page.getByRole('link', { name: 'Dashboard' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Opportunities' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Operations' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Intelligence' })).toBeVisible();

    await expect(page.getByRole('heading', { name: /Good .*Developer/i })).toBeVisible();
    await expect(page.getByLabel('Primary dashboard metrics').getByText('Active RFQs')).toBeVisible();
    await expect(page.getByRole('button', { name: /Open Opportunity pipeline/i })).toBeVisible();
  });

  test('text size controls persist a readable system scale', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.getByRole('button', { name: 'Large text' }).click();
    await expect.poll(() => page.evaluate(() => document.documentElement.dataset.textScale)).toBe('large');
    await expect.poll(() => page.evaluate(() => window.localStorage.getItem('asymmflow.textScale'))).toBe('1.25');

    await page.reload();
    await expect(page.getByRole('link', { name: 'Dashboard' })).toBeVisible();
    await expect.poll(() => page.evaluate(() => document.documentElement.dataset.textScale)).toBe('large');

    await page.getByRole('link', { name: 'Settings' }).click();
    await expect(page.locator('.text-preset-buttons').getByRole('button', { name: /Large/ })).toHaveAttribute('aria-pressed', 'true');
    await expect(page.locator('#settings-text-scale')).toHaveValue('125');
  });
});
