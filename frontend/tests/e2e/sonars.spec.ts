import { expect, test } from '@playwright/test';
import { openGlobalOcrFromDashboard } from './helpers/globalOcr';
import { installMockWailsBridge } from './helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

async function collectPageErrors(page: any) {
  const errors: string[] = [];
  page.on('pageerror', (error: Error) => errors.push(error.message));
  page.on('console', (msg: any) => {
    if (msg.type() === 'error') {
      errors.push(msg.text());
    }
  });
  return errors;
}

test.describe('UI health sonars', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
  });

  test('shell sonar: dashboard renders primary navigation quickly', async ({ page }) => {
    const errors = await collectPageErrors(page);
    const start = Date.now();

    await page.goto(BASE_URL);
    await expect(page.getByRole('heading', { name: /Good .*Developer/i })).toBeVisible();

    const duration = Date.now() - start;

    await expect(page.getByRole('link', { name: 'Dashboard' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Opportunities' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Operations' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Intelligence' })).toBeVisible();
    expect(duration).toBeLessThan(3000);
    expect(errors).toEqual([]);
  });

  test('customer sonar: relationships detail flow stays interactive without page errors', async ({ page }) => {
    const errors = await collectPageErrors(page);

    await page.goto(BASE_URL);
    await page.getByRole('link', { name: 'Relationships' }).click();
    await expect(page.getByRole('heading', { name: 'Relationships' })).toBeVisible();
    await page.locator('.customer-card').filter({ hasText: 'NPC' }).first().click();

    await expect(page.getByText('Amina Yusuf')).toBeVisible();
    await expect(page.getByText('Procurement Manager')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'NPC' })).toBeVisible();
    expect(errors).toEqual([]);
  });

  test('opportunity sonar: modal opens and closes without breaking the list view', async ({ page }) => {
    const errors = await collectPageErrors(page);

    await page.goto(BASE_URL);
    await page.getByRole('link', { name: 'Opportunities' }).click();
    await page.getByRole('button', { name: /ACTIUM PG/i }).first().click();

    await expect(page.locator('#opportunity-modal-title')).toBeVisible();
    await page.getByRole('button', { name: /Close/ }).click();
    await expect(page.getByRole('button', { name: /ACTIUM PG/i }).first()).toBeVisible();
    expect(errors).toEqual([]);
  });

  test('intelligence sonar: Butler thread survives a page reload', async ({ page }) => {
    const errors = await collectPageErrors(page);

    await page.goto(BASE_URL);
    await page.getByRole('link', { name: 'Intelligence' }).click();
    await page.getByPlaceholder(/Ask about financials, customers, suppliers/i).fill('Draft a short pitch for NPC.');
    await page.keyboard.press('Enter');

    await expect(page.locator('#chat-feed').getByText('Mock Butler response persisted.')).toBeVisible();
    await page.reload();
    await page.getByRole('link', { name: 'Intelligence' }).click();
    await expect(page.getByText('Draft a short pitch for NPC.').first()).toBeVisible();
    await expect(page.locator('#chat-feed').getByText('Mock Butler response persisted.')).toBeVisible();
    expect(errors).toEqual([]);
  });

  test('document sonar: OCR modal returns structured analysis for bank statements', async ({ page }) => {
    const errors = await collectPageErrors(page);

    await page.goto(BASE_URL);
    await openGlobalOcrFromDashboard(page);

    await expect(page.getByText('Document Capture')).toBeVisible();
    await expect(page.getByText('Document Type')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Bank Statement', exact: true })).toBeVisible();
    await expect(page.getByText(/AI Summary:/)).toBeVisible();
    await expect(page.getByRole('button', { name: 'Save as Bank Statement' })).toBeVisible();
    expect(errors).toEqual([]);
  });
});
