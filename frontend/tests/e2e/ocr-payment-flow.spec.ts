import { expect, test } from '@playwright/test';
import { openGlobalOcrFromDashboard } from './helpers/globalOcr';
import { installMockWailsBridge } from './helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

test.describe('OCR workflow smoke', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
  });

  test('opens OCR capture from the dashboard and shows Butler analysis for a bank statement', async ({ page }) => {
    await page.goto(BASE_URL);

    await openGlobalOcrFromDashboard(page);
    await expect(page.getByText('AI Summary:')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Mock Butler analysis: one credit transaction and a clean balance trail.')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Save as Bank Statement' })).toBeVisible();
  });

  test('lets the operator inspect extracted transactions and switch the target save type', async ({ page }) => {
    await page.goto(BASE_URL);

    await openGlobalOcrFromDashboard(page);
    await page.getByRole('button', { name: /Transactions \(1\)/ }).click();

    await expect(page.locator('.items-table input[type="text"]').first()).toHaveValue('Deposit');

    await page.getByRole('button', { name: 'Report / Summary' }).click();
    await expect(page.getByRole('button', { name: 'Save as Report / Summary' })).toBeVisible();
  });

  test('saves a classified bank statement through the backend routing bridge', async ({ page }) => {
    await page.goto(BASE_URL);

    await openGlobalOcrFromDashboard(page);
    await page.locator('#bank_account_id').selectOption('bank-1');
    await page.getByRole('button', { name: 'Save as Bank Statement' }).click();

    await expect.poll(async () => page.evaluate(() => (window as any).__wailsDocumentSaveCalls.length)).toBe(1);

    const saveCall = await page.evaluate(() => {
      const call = (window as any).__wailsDocumentSaveCalls[0];
      return {
        ...call,
        extractedData: JSON.parse(call.extractedDataJSON || '{}'),
      };
    });

    expect(saveCall.documentType).toBe('bank_statement');
    expect(saveCall.fileName).toBe('bank-statement.pdf');
    expect(saveCall.confidence).toBe(0.96);
    expect(saveCall.engine).toBe('mock-ocr');
    expect(saveCall.extractedData.bank_account_id).toBe('bank-1');
    expect(saveCall.extractedData.bank_name).toBe('National Bank of Bahrain');
    expect(saveCall.extractedData.line_items).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          description: 'Deposit',
          credit: 500,
          balance: 1500,
        }),
      ]),
    );
  });
});
