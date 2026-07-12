import { expect, test } from '@playwright/test';
import { installMockWailsBridge } from './helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

test.describe('Pipeline and commercial smoke', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
  });

  test('creates an opportunity and converts a blank costing into an offer', async ({ page }) => {
    const opportunityTitle = 'NPC / RFQ-2026-002';

    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Opportunities' }).click();
    await expect(page.getByRole('button', { name: /\+ New Opportunity/ })).toBeVisible();

    await page.getByRole('button', { name: /\+ New Opportunity/ }).click();
    await page.locator('input[list="cust-list"]').fill('NPC');
    await page.getByPlaceholder('Project name').fill('Launch readiness smoke');
    await page.getByPlaceholder('0.00').fill('12500');
    await page.getByRole('button', { name: 'Create' }).click();

    await expect(page.getByRole('heading', { name: opportunityTitle })).toBeVisible();

    await page.getByRole('tab', { name: /Costing/i }).click();
    await page.getByRole('button', { name: 'Start Blank Costing' }).click();

    await page.locator('select.input').first().selectOption({ label: 'NPC' });
    await page.locator('select.input-sm').nth(1).selectOption({ index: 1 });
    await page.getByPlaceholder('Product name').fill('Pressure transmitter');
    await page.locator('input.money.sell-price').first().fill('1450');

    await page.getByRole('button', { name: 'Save as Offer' }).click();

    await expect(page.getByText(/Offer PHO-2026-002 created!/)).toBeVisible();
    await expect(page.getByRole('button', { name: /\+ New Offer \(via Costing\)/ })).toBeVisible();
    await expect(page.getByRole('cell', { name: 'PHO-2026-002' })).toBeVisible();
  });

  test('creates a supplier purchase order from operations hub', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Operations' }).click();
    await expect(page.getByRole('button', { name: /\+ New (Supplier|Purchase) Order/ })).toBeVisible();

    await page.getByRole('button', { name: /\+ New (Supplier|Purchase) Order/ }).click();
    await expect(page.getByText('Create Purchase Order')).toBeVisible();

    await page.locator('.po-form select.select-input').first().selectOption({ label: 'Rhine Instruments' });
    await page.getByRole('button', { name: '+ Add Item' }).click();
    await page.getByPlaceholder('Product or service description').fill('Temperature transmitter');
    await page.locator('.items-list input[type="number"]').first().fill('3');
    await page.locator('.items-list input[type="number"]').nth(1).fill('850');

    await expect(page.getByText('2,550.000 BHD').first()).toBeVisible();
    await expect(page.getByText('2,805.000 BHD').first()).toBeVisible();
    await expect(page.getByRole('button', { name: 'Create PO' })).toBeVisible();
  });
});
