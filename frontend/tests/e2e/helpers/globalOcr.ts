import { expect, type Page } from '@playwright/test';

export async function openGlobalOcrFromDashboard(page: Page) {
  await page.waitForFunction(() => (window as any).__wailsFileDropCallbacks?.length > 0);

  await page.evaluate(() => {
    const entry = (window as any).__wailsFileDropCallbacks.at(-1);
    entry.callback(80, 120, ['/tmp/ph-holdings-smoke/bank-statement.pdf']);
  });

  await expect(page.getByText('Document Type')).toBeVisible();
}
