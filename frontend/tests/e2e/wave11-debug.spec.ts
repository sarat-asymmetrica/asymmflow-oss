import { test, expect } from '@playwright/test';
import { installMockWailsBridge } from './helpers/mockWailsBridge';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const __dirname = path.dirname(fileURLToPath(import.meta.url));
const EN = JSON.parse(fs.readFileSync(path.resolve(__dirname, '../../../pkg/i18n/messages/en.json'), 'utf8'));
const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';
const OUT = path.resolve(__dirname, '../../../docs/wave11-qa/debug');

test('probe people detail tabs', async ({ page }) => {
  await installMockWailsBridge(page, { translations: EN });
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto(BASE_URL);
  await page.evaluate(() => { window.location.hash = '#people'; });
  await page.waitForTimeout(2500);
  // Employee auto-selects (load() picks employees[0]); detail pane + sub-tabs render.
  fs.mkdirSync(OUT, { recursive: true });
  await page.screenshot({ path: path.join(OUT, 'people-detail.png'), fullPage: true });

  const info = await page.evaluate(() => {
    const tabs = document.querySelector('.detail-subtabs') as HTMLElement | null;
    const btn = document.querySelector('.detail-subtabs button') as HTMLElement | null;
    const cs = (el: HTMLElement | null) => el ? {
      rect: el.getBoundingClientRect(),
      display: getComputedStyle(el).display,
      flexDirection: getComputedStyle(el).flexDirection,
      alignItems: getComputedStyle(el).alignItems,
      height: getComputedStyle(el).height,
      bg: getComputedStyle(el).backgroundColor,
      borderRadius: getComputedStyle(el).borderRadius,
    } : null;
    const root = getComputedStyle(document.documentElement);
    return {
      tabsExists: !!tabs,
      tabs: cs(tabs),
      firstBtn: cs(btn),
      tokenAccentPrimary: root.getPropertyValue('--accent-primary'),
      tokenAccent: root.getPropertyValue('--accent'),
    };
  });
  console.log('PEOPLE_PROBE=' + JSON.stringify(info, null, 2));
  expect(true).toBe(true);
});
