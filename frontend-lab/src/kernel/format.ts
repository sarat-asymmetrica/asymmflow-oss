/* L2 — one definition per utility. The old frontend defines formatDate
 * 20 separate times across 17 screens; this file is where that stops. */

/** dd MMM yyyy — the display format used across AsymmFlow documents. */
export function formatDate(value: string | Date | null | undefined): string {
  if (!value) return '—'
  const d = typeof value === 'string' ? new Date(value) : value
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
}

/** BHD uses 3 decimal places (fils); other currencies default to 2. */
export function formatMoney(amount: number | null | undefined, currency = 'BHD'): string {
  if (amount == null || Number.isNaN(amount)) return '—'
  const decimals = currency === 'BHD' ? 3 : 2
  return `${currency} ${amount.toLocaleString('en-US', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  })}`
}

export function formatNumber(value: number | null | undefined): string {
  if (value == null || Number.isNaN(value)) return '—'
  return value.toLocaleString('en-US')
}
