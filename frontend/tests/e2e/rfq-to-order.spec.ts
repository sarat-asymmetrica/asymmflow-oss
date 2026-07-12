import { expect, test } from '@playwright/test';
import { installMockWailsBridge } from './helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

test.describe('Commercial pipeline transitions', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
    await page.goto(BASE_URL);
    await page.getByRole('link', { name: 'Opportunities' }).click();
  });

  test('RFQ list shows seeded opportunities for commercial follow-up', async ({ page }) => {
    await page.getByRole('tab', { name: /^RFQs$/ }).click();

    await expect(page.getByRole('button', { name: /RFQ-2026-001 2026 RFQ NPC/ })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'NPC / RFQ-2026-001' })).toBeVisible();
  });

  test('new opportunity appears in the modal workflow and can be handed into costing', async ({ page }) => {
    const opportunityTitle = 'NPC / RFQ-2026-002';

    await page.getByRole('button', { name: /\+ New Opportunity/ }).click();

    await page.locator('input[list="cust-list"]').fill('NPC');
    await page.getByPlaceholder('Project name').fill('Pipeline handoff smoke');
    await page.getByPlaceholder('0.00').fill('18500');
    await page.getByPlaceholder('Add context, scope, customer request details, exclusions, or next steps...').fill('Carry this into costing for the sales team.');
    await page.getByRole('button', { name: 'Create' }).click();

    await expect(page.getByRole('heading', { name: opportunityTitle })).toBeVisible();
    await page.getByRole('button', { name: new RegExp(opportunityTitle.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')) }).first().click();

    await expect(page.locator('#opportunity-modal-title')).toHaveText(opportunityTitle);
    await expect(page.getByText('Carry this into costing for the sales team.')).toBeVisible();
    await expect(page.getByText('REFERENCE').first()).toBeVisible();
  });
});
