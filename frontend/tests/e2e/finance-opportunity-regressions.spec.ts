import { expect, test } from '@playwright/test';
import { installMockWailsBridge } from './helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

test.describe('Opportunity and finance regression coverage', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
  });

  test('opportunity modal shows extracted items and persists edited notes', async ({ page }) => {
    test.slow();
    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Opportunities' }).click();
    await page.getByRole('button', { name: /ACTIUM PG/i }).first().click();

    await expect(page.locator('#opportunity-modal-title')).toHaveText(/ACTIUM PG \/ 2026-311/i);
    await expect(page.getByText('Line Items (2)')).toBeVisible();
    await expect(page.getByText('Gas analyzer transmitter')).toBeVisible();
    await expect(page.getByText('Sample conditioning panel')).toBeVisible();

    const notesBox = page.getByPlaceholder('Add a comment...');
    await expect(notesBox).toBeEditable({ timeout: 10000 });
    await notesBox.fill('Updated regression note for deployment readiness.');
    await page.getByRole('button', { name: 'Post' }).click();
    await expect(page.getByText('Comment added')).toBeVisible();

    await page.getByRole('button', { name: /Close/ }).click();
    await page.getByRole('button', { name: /ACTIUM PG/i }).first().click();
    await expect(page.getByText('Initial technical scope captured from OCR.')).toBeVisible();
  });

  test('supplier invoice edit flow can mark an invoice paid and request payment entry creation', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Operations' }).click();
    await page.getByRole('button', { name: /Supplier Invoices/i }).click();
    await page.getByRole('button', { name: 'Edit invoice' }).first().click();

    await expect(page.getByText('Edit Supplier Invoice')).toBeVisible();
    const modal = page.locator('.modal-overlay').filter({ hasText: 'Edit Supplier Invoice' }).last();
    const paymentStatusGroup = modal.locator('.form-group').filter({ has: page.locator('label', { hasText: 'Payment Status' }) });
    const paymentMethodGroup = modal.locator('.form-group').filter({ has: page.locator('label', { hasText: 'Payment Method' }) });
    await paymentStatusGroup.locator('select').selectOption('Paid');
    await paymentMethodGroup.locator('select').selectOption('Bank Transfer');
    await modal.getByPlaceholder('Transfer / cheque reference').fill('TXN-1001');
    await modal.getByLabel(/Create supplier payment ledger entry/i).check();
    await modal.getByRole('button', { name: 'Save Changes' }).click();

    await expect(page.getByText('Supplier invoice updated and payment entry created')).toBeVisible();
    await expect(page.getByText('Paid').first()).toBeVisible();
  });

  test('opportunity detail can jump into costing while folder and subject stay user-entered', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Opportunities' }).click();
    await page.getByRole('button', { name: /ACTIUM PG/i }).first().click();
    await page.getByRole('button', { name: 'Create Costing Sheet' }).click();

    await expect(page.getByText('Linked Opportunity:')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('2026-311')).toBeVisible();
    await page.getByRole('button', { name: /Compliance, Certificates & VAT/i }).click();
    await expect(page.locator('input[placeholder="Enter folder no."]')).toHaveValue('');
    await expect(page.locator('input[placeholder="Enter costing ID"]')).toHaveValue('');
    await expect(page.locator('input[placeholder="Subject line for customer PDF"]')).toHaveValue('');
    await expect(page.getByText('Manual Unit Price').first()).toBeVisible();
    await expect(page.getByText('Exch. Rate').first()).toBeVisible();
    await expect(page.getByText('Freight %').first()).toBeVisible();
  });

  test('offer header edits persist after saving in the view modal', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.getByRole('link', { name: 'Opportunities' }).click();
    await page.getByRole('tab', { name: /Offers/i }).click();
    await page.getByRole('button', { name: 'View' }).first().click();

    await expect(page.getByText('Offer Details')).toBeVisible();
    await page.getByPlaceholder('Folder #').fill('64-26');
    await page.getByPlaceholder('RFQ / enquiry ref').fill('EH-64-26-R0');
    await page.getByPlaceholder('e.g. Net 30').fill('30 days from Date of Delivery');
    await page.getByPlaceholder('e.g. DAP Bahrain').fill('Delivery Duty Paid');
    await page.getByRole('button', { name: 'Save Changes' }).click();

    await expect(page.getByText('Offer details saved')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Offer Details' })).not.toBeVisible();
    await page.getByLabel('View full details').first().click();

    await expect(page.getByPlaceholder('Folder #')).toHaveValue('64-26');
    await expect(page.getByPlaceholder('RFQ / enquiry ref')).toHaveValue('EH-64-26-R0');
    await expect(page.getByPlaceholder('e.g. Net 30')).toHaveValue('30 days from Date of Delivery');
    await expect(page.getByPlaceholder('e.g. DAP Bahrain')).toHaveValue('Delivery Duty Paid');
  });
});
