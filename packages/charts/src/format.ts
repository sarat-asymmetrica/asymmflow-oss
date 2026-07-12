/**
 * format.ts — numeric display formatting for chart labels and tooltips.
 *
 * Every financial figure renders in tabular numerals via the .af-numeric
 * class (base.css §4a) — these helpers only produce the STRING.
 */

/** Compact notation for axis ticks: 1200 → '1.2K', 3_400_000 → '3.4M'. */
export function formatCompact(n: number, locale = 'en'): string {
  return new Intl.NumberFormat(locale, {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(n);
}

/**
 * Currency formatting. Fraction digits follow the currency's convention
 * (BHD → 3 decimals, USD/EUR → 2) unless overridden.
 */
export function formatCurrency(
  n: number,
  currency = 'BHD',
  opts: { locale?: string; compact?: boolean } = {},
): string {
  return new Intl.NumberFormat(opts.locale ?? 'en', {
    style: 'currency',
    currency,
    ...(opts.compact ? { notation: 'compact' as const, maximumFractionDigits: 1 } : {}),
  }).format(n);
}

/** Percentage with fixed digits: formatPercent(0.875) → '88%'. Input is a ratio. */
export function formatPercent(ratio: number, digits = 0): string {
  return new Intl.NumberFormat('en', {
    style: 'percent',
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(ratio);
}

/** Plain grouped number: 1234567 → '1,234,567'. */
export function formatNumber(n: number, locale = 'en'): string {
  return new Intl.NumberFormat(locale).format(n);
}
