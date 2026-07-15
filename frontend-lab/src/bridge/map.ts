/* Shared Go-model → kernel-row mapping helpers. ONE place (L2) for the
 * coercions every per-entity real adapter needs: Go zero-time handling,
 * null-safe strings, NaN-safe numbers, and the forward date→time.Time bridge.
 * Per KERNEL/real.ts: Go-date parsing lives in the bridge layer, never in
 * screens. */

import type { time } from '$wails/go/models'

/** Go time.Time → 'YYYY-MM-DD' ('' for Go zero time / garbage / pre-1971). */
export function goDate(value: unknown): string {
  if (!value) return ''
  const s = String(value)
  if (s.startsWith('0001-01-01')) return '' // Go zero time
  const d = new Date(s)
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 1971) return ''
  return s.slice(0, 10)
}

/**
 * The FORWARD date bridge: a form date string → a Go `time.Time` binding arg.
 * The ONE kernel-level conversion (I1.3) — a screen/descriptor must never build
 * a time.Time itself (L2). Every write binding whose Go signature takes a
 * `time.Time` (SetExchangeRate, CalculateFXRevaluation, CreateBookBankRecon, …)
 * routes its date through here.
 *
 * How it works: Wails JSON-serializes binding arguments over the IPC boundary,
 * and Go's `time.Time.UnmarshalJSON` expects a quoted RFC3339 string. The
 * generated `time.Time` TS class is an empty codegen stub (its constructor
 * stores nothing), so constructing it is useless — the value that actually
 * crosses the wire is whatever we pass, JSON-encoded. We therefore emit the
 * RFC3339 string itself (UTC midnight of the calendar date, explicit `Z` to
 * avoid the local-timezone ambiguity of `new Date('YYYY-MM-DD')`) and satisfy
 * the binding's `time.Time` parameter type with a structural cast.
 *
 * A `<input type=date>` yields 'YYYY-MM-DD'; a datetime value is passed through.
 * Empty/blank input maps to Go's ZERO time ('0001-01-01T00:00:00Z') — callers
 * that require a real date MUST guard before calling (an empty date is a
 * validation error at the form, not a silent 0001 write).
 */
export function goTime(dateStr: string): time.Time {
  const s = (dateStr ?? '').trim()
  if (!s) return '0001-01-01T00:00:00Z' as unknown as time.Time
  // Already carries a time component (has 'T') → pass through; else UTC midnight.
  const rfc = s.includes('T') ? s : `${s}T00:00:00Z`
  return rfc as unknown as time.Time
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
