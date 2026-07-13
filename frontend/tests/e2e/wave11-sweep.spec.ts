import { test, expect } from '@playwright/test';
import { installMockWailsBridge } from './helpers/mockWailsBridge';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

/**
 * Wave 11 — Polish & True Mirror: the standing QA sweep.
 *
 * Drives every primary NAV_ITEMS screen (plus a few deep-link surfaces) against
 * the vite dev server with the synthetic mock Wails bridge installed, and takes
 * a full-page screenshot at two widths. Screens that render blank without data
 * are themselves findings — the sweep never fails on a missing data call; it
 * captures whatever the app renders so a human can review the LAYOUT truthfully.
 *
 * Output: docs/wave11-qa/<width>/<screen>.png  (repo-relative, synthetic only).
 * See docs/wave11-qa/QA_SWEEP.md for the full recipe.
 */

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

// Repo-root-relative output dir (spec runs from frontend/, docs/ is one level up).
const OUT_ROOT = path.resolve(__dirname, '../../../docs/wave11-qa');

// Real English messages so the shell renders proper labels (not raw i18n keys)
// and the startup initI18n() call resolves. Sourced from the Go embed.
const EN_TRANSLATIONS: Record<string, string> = JSON.parse(
  fs.readFileSync(path.resolve(__dirname, '../../../pkg/i18n/messages/en.json'), 'utf8'),
);

// Every primary nav screen (NAV_ITEMS order) + key deep-link surfaces.
// `tab` optional: a query the app understands for a sub-view, applied after the
// screen mounts by clicking a tab control if present.
const SCREENS: Array<{ id: string; label: string }> = [
  { id: 'dashboard', label: 'Dashboard' },
  { id: 'opportunities', label: 'Opportunities' },
  { id: 'operations', label: 'Operations' },
  { id: 'finance', label: 'Finance' },
  { id: 'accounting', label: 'Accounting' },
  { id: 'reports', label: 'Reports' },
  { id: 'work', label: 'Work' },
  { id: 'people', label: 'People' },
  { id: 'notifications', label: 'Notifications' },
  { id: 'relationships', label: 'Relationships' },
  { id: 'intelligence', label: 'Intelligence' },
  { id: 'settings', label: 'Settings' },
  { id: 'usermanagement', label: 'UserManagement' },
  { id: 'deployment', label: 'Deployment' },
];

const WIDTHS = [
  { name: '1440', width: 1440, height: 900 },
  { name: '1100', width: 1100, height: 900 },
];

for (const vp of WIDTHS) {
  test.describe(`wave11 sweep @ ${vp.name}px`, () => {
    for (const screen of SCREENS) {
      test(`${screen.label}`, async ({ page }) => {
        await installMockWailsBridge(page, { translations: EN_TRANSLATIONS });
        await page.setViewportSize({ width: vp.width, height: vp.height });
        await page.goto(BASE_URL);
        // Land on the shell (dashboard heading) before deep-navigating.
        await page.waitForLoadState('domcontentloaded');
        await page.evaluate((id) => { window.location.hash = '#' + id; }, screen.id);
        // Give lazy-loaded chunk + data calls time to settle; tolerate failures.
        await page.waitForTimeout(1500);
        try { await page.waitForLoadState('networkidle', { timeout: 4000 }); } catch { /* tolerate */ }

        const dir = path.join(OUT_ROOT, vp.name);
        fs.mkdirSync(dir, { recursive: true });
        await page.screenshot({
          path: path.join(dir, `${screen.id}.png`),
          fullPage: true,
        });
        // The sweep asserts nothing about content — capture is the deliverable.
        expect(true).toBe(true);
      });
    }
  });
}
