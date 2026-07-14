/* Shared Go-model → kernel-row mapping helpers. ONE place (L2) for the
 * coercions every per-entity real adapter needs: Go zero-time handling,
 * null-safe strings, NaN-safe numbers. Per KERNEL/real.ts: Go-date parsing
 * lives in the bridge layer, never in screens. */

/** Go time.Time → 'YYYY-MM-DD' ('' for Go zero time / garbage / pre-1971). */
export function goDate(value: unknown): string {
  if (!value) return ''
  const s = String(value)
  if (s.startsWith('0001-01-01')) return '' // Go zero time
  const d = new Date(s)
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 1971) return ''
  return s.slice(0, 10)
}

/** Null-safe string coercion. */
export function str(v: unknown): string {
  return v == null ? '' : String(v)
}

/** NaN-safe number coercion (defaults to 0). */
export function num(v: unknown): number {
  const n = typeof v === 'number' ? v : Number(v)
  return Number.isNaN(n) ? 0 : n
}
