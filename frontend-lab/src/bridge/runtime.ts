/* Bridge runtime — the ONE place that decides real-Wails vs mock, shared by
 * every per-entity bridge module. Extracted from index.ts so the pilots and
 * all new entity modules pick their implementation identically (L2). */

export function usingWails(): boolean {
  return typeof window !== 'undefined' && !!(window as { go?: unknown }).go
}

/** Choose the real binding when the Wails runtime is present, else the mock. */
export const pick = <T>(real: T, mock: T): T => (usingWails() ? real : mock)
