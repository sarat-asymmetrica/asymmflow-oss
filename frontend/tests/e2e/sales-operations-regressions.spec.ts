import { expect, test } from '@playwright/test';
import { installMockWailsBridge } from './helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

test.describe('Sales and operations regression smoke', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
  });

  test('sales hub tabs remain clickable across RFQs, offers, and customer orders', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Opportunities' }).click();
    await expect(page.getByRole('button', { name: /\+ New Opportunity/ })).toBeVisible();

    await page.getByRole('tab', { name: /^Offers$/ }).click();
    await expect(page.getByRole('button', { name: /\+ New Offer/i })).toBeVisible();

    await page.getByRole('tab', { name: /^Customer Orders$/ }).click();
    await expect(page.getByText('Customer Orders')).toBeVisible();

    await page.getByRole('tab', { name: /^RFQs$/ }).click();
    await expect(page.getByRole('button', { name: /\+ New Opportunity/ })).toBeVisible();
  });

  test('purchase order details modal shows supplier, line items, and foreign totals', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Operations' }).click();
    await page.getByLabel('View PO details').first().click();

    const modal = page.getByLabel('Purchase Order Details');

    await expect(modal.getByText('Purchase Order Details')).toBeVisible();
    await expect(modal.getByText('Rhine Instruments')).toBeVisible();
    await expect(modal.getByRole('heading', { name: 'Items' })).toBeVisible();
    await expect(modal.getByText('Temperature transmitter')).toBeVisible();
    await expect(modal.getByText('Signal isolator')).toBeVisible();
    await expect(modal.getByText('6,820.000 EUR')).toBeVisible();
    await expect(modal.getByText('2,796.200 BHD')).toBeVisible();
  });
});
