import { expect, test } from '@playwright/test';
import { installMockWailsBridge } from './helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

test.describe('Dashboard OCR drag and drop', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
  });

  test('native Wails file drop opens the global OCR modal from the dashboard', async ({ page }) => {
    await page.goto(BASE_URL);

    await page.waitForFunction(() => (window as any).__wailsFileDropCallbacks?.length > 0);

    const registration = await page.evaluate(() => {
      const entry = (window as any).__wailsFileDropCallbacks.at(-1);
      return { useDropTarget: entry.useDropTarget };
    });
    expect(registration.useDropTarget).toBe(false);

    await page.evaluate(() => {
      const entry = (window as any).__wailsFileDropCallbacks.at(-1);
      entry.callback(80, 120, ['/tmp/ph-holdings-smoke/bank-statement.pdf']);
    });

    await expect(page.getByText('Document Type')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Save as Bank Statement' })).toBeVisible();

    await expect.poll(async () => page.evaluate(() => (window as any).__wailsOCRCalls.length)).toBe(1);
    const call = await page.evaluate(() => (window as any).__wailsOCRCalls[0]);
    expect(call).toEqual({
      filePath: '/tmp/ph-holdings-smoke/bank-statement.pdf',
      docType: 'auto',
    });
  });
});
