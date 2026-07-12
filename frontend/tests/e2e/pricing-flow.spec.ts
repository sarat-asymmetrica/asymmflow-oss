import { expect, test } from '@playwright/test';
import { installMockWailsBridge } from './helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

test.describe('Commercial pricing and quotation flow', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
    await page.goto(BASE_URL);
    await page.getByRole('link', { name: 'Opportunities' }).click();
  });

  test('blank costing can be priced and saved as a quoted offer', async ({ page }) => {
    await page.getByRole('tab', { name: /Costing/i }).click();
    await page.getByRole('button', { name: 'Start Blank Costing' }).click();

    await page.locator('select.input').first().selectOption('NPC');
    await page.locator('select.input-sm').first().selectOption({ index: 1 });
    await page.getByPlaceholder('Product name').fill('Marine engine lubricant');
    await page.locator('input.money.sell-price').first().fill('1450');

    await page.getByRole('button', { name: 'Save as Offer' }).click();

    await expect(page.getByText(/Offer PHO-2026-002 created!/)).toBeVisible();
    await expect(page.getByRole('cell', { name: 'PHO-2026-002' })).toBeVisible();
    await expect(page.getByRole('cell', { name: '1,595.000 BHD' })).toBeVisible();
  });

  test('costing numeric fields tolerate temporarily blank freight and hidden charges', async ({ page }) => {
    const pageErrors: string[] = [];
    page.on('pageerror', error => pageErrors.push(error.message));

    await page.getByRole('tab', { name: /Costing/i }).click();
    await page.getByRole('button', { name: 'Start Blank Costing' }).click();

    await page.getByPlaceholder('Product name').fill('Flowmeter panel');
    await page.locator('.line-row-pricing input.money').first().fill('5000');
    await page.locator('.line-row-pricing input.money').nth(1).fill('');

    await expect(page.getByText('Freight %: 0%').first()).toBeVisible();
    await page.locator('.summary-row').filter({ hasText: 'Hidden Charges' }).locator('input').fill('125');
    await expect(page.getByText('Internal Hidden Charges')).toBeVisible();
    await expect(page.getByText('125.000 BHD')).toBeVisible();
    expect(pageErrors).toEqual([]);
  });

  test('unsaved costing draft survives app reload with user-entered identifiers', async ({ page }) => {
    await page.getByRole('tab', { name: /Costing/i }).click();
    await page.getByRole('button', { name: 'Start Blank Costing' }).click();

    await page.locator('input[placeholder="Enter folder no."]').fill('MANUAL-42-26');
    await page.locator('input[placeholder="Enter costing ID"]').fill('Manual Costing 42-26');
    await page.getByPlaceholder('Product name').fill('Draft restore flowmeter');

    page.once('dialog', async dialog => {
      await dialog.accept();
    });
    await page.reload();
    await page.getByRole('link', { name: 'Opportunities' }).click();
    await page.getByRole('tab', { name: /Costing/i }).click();

    await expect(page.getByText('Restored an unsaved costing draft from this device.')).toBeVisible();
    await expect(page.locator('input[placeholder="Enter folder no."]')).toHaveValue('MANUAL-42-26');
    await expect(page.locator('input[placeholder="Enter costing ID"]')).toHaveValue('Manual Costing 42-26');
    await expect(page.getByPlaceholder('Product name')).toHaveValue('Draft restore flowmeter');
  });

  test('back to opportunities warns before leaving an unsaved costing sheet', async ({ page }) => {
    await page.getByRole('tab', { name: /Costing/i }).click();
    await page.getByRole('button', { name: 'Start Blank Costing' }).click();

    await page.getByPlaceholder('Product name').fill('Warning gate flowmeter');
    await page.getByRole('button', { name: 'Back to Opportunities' }).click();

    await expect(page.getByRole('dialog', { name: 'Leave this costing sheet?' })).toBeVisible();
    await page.getByRole('button', { name: 'Stay on Sheet' }).click();
    await expect(page.getByPlaceholder('Product name')).toHaveValue('Warning gate flowmeter');

    await page.getByRole('button', { name: 'Back to Opportunities' }).click();
    await page.getByRole('button', { name: 'Keep Draft & Go Back' }).click();

    await expect(page.getByText('Select an opportunity to create a costing sheet.')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Resume Draft' })).toBeVisible();
    await expect(page.getByPlaceholder('Product name')).toHaveCount(0);
  });

  test('quoted offers expose the current commercial actions', async ({ page }) => {
    await page.getByRole('tab', { name: /Offers/i }).click();

    await expect(page.getByRole('cell', { name: 'PHO-2026-001' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Edit offer' }).first()).toBeVisible();
    await expect(page.getByRole('button', { name: 'Download PDF' }).first()).toBeVisible();
    await expect(page.getByRole('button', { name: 'Mark as won' }).first()).toBeVisible();
    await expect(page.getByRole('button', { name: 'Schedule follow-up' })).toHaveCount(0);
  });

  test('editing an offer opens it directly in the costing sheet', async ({ page }) => {
    await page.getByRole('tab', { name: /Offers/i }).click();
    await page.getByRole('button', { name: 'Edit offer' }).first().click();

    await expect(page.getByRole('tab', { name: /^Costing$/i })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('input[placeholder="Enter costing ID"]')).toHaveValue('PHO-2026-001');
    await expect(page.getByPlaceholder('Product name')).toHaveValue('Pressure transmitter');
  });

  test('operations purchase order draft keeps pricing totals visible before submit', async ({ page }) => {
    await page.getByRole('link', { name: 'Operations' }).click();
    await page.getByRole('button', { name: /\+ New (Supplier|Purchase) Order/ }).click();

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
