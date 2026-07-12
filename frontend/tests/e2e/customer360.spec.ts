import { expect, test } from '@playwright/test';
import { installMockWailsBridge } from './helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

test.describe('Relationships customer detail smoke', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
  });

  test('opens a customer profile from Relationships and shows overview data', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Relationships' }).click();
    await expect(page.getByRole('heading', { name: 'Relationships' })).toBeVisible();
    await expect(page.getByText('ALL CUSTOMERS')).toBeVisible();

    await page.locator('.customer-card').filter({ hasText: 'NPC' }).first().click();

    await expect(page.getByRole('button', { name: 'Back to Customers' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'NPC' })).toBeVisible();
    await expect(page.getByText('Government')).toBeVisible();
    await expect(page.getByText('Energy')).toBeVisible();
    await expect(page.getByText('Total Orders')).toBeVisible();
  });

  test('adds a management note and a contact on the customer detail view', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Relationships' }).click();
    await expect(page.getByText('ALL CUSTOMERS')).toBeVisible();
    await page.locator('.customer-card').filter({ hasText: 'NPC' }).first().click();

    await page.getByRole('button', { name: '+ Add' }).click();
    await page.getByPlaceholder('Contact person name').fill('Lina Thomas');
    await page.getByPlaceholder('email@company.com').fill('lina@natpetro.example');
    await page.getByRole('button', { name: 'Save Contact' }).click();

    await expect(page.getByText('Contact added')).toBeVisible();

    await page.getByRole('button', { name: 'Notes' }).click();
    await page.getByRole('button', { name: '+ Add Note' }).click();
    await page.getByPlaceholder('Enter note...').fill('Customer detail smoke note');
    await page.getByRole('button', { name: 'Save Note' }).click();

    await expect(page.getByText('Note added')).toBeVisible();
  });
});
