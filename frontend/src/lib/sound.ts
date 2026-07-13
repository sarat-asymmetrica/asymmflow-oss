/**
 * Wave 10 / B4 — the ENTIRE audio budget of the application.
 *
 * Article IV.3 (DESIGN_CONSTITUTION.md): "Sound is saffron. The application
 * budget is one sound, reserved for the operator's true win moment — a deal
 * closing as paid... No arrival sounds, no error sounds, no routine-save
 * sounds."
 *
 * This module contains the ONLY `new Audio(...)` construction in the whole
 * codebase. Do not add another one — if a future flow wants a sound, it
 * spends THIS budget or it doesn't happen.
 *
 * The asset is bundled by Vite (see the import below) and served locally by
 * Wails' embedded asset server — no network fetch at runtime.
 */
import paidSettleUrl from '../assets/sounds/paid-settle.wav';
import { soundOnPaidEnabled } from './stores/soundSettings';
import { get } from 'svelte/store';

/**
 * Plays the one application sound — the "paid settle" — for the acting
 * user's own posting click that fully applies a customer invoice to PAID.
 *
 * MUST be called synchronously as the first statement in the click-handler
 * path that follows a successful PAID transition (no `await` before this
 * call), or WebView2/Chromium's autoplay-with-sound gesture attribution is
 * lost and playback silently fails.
 *
 * Never throws — swallows playback errors so a missing/blocked sound can
 * never interrupt the real posting flow.
 */
export function playPaidSettle(): void {
  if (!get(soundOnPaidEnabled)) return;
  try {
    new Audio(paidSettleUrl).play().catch(() => {});
  } catch {
    // Construction itself should never throw, but never let a sound
    // failure surface to the user.
  }
}
