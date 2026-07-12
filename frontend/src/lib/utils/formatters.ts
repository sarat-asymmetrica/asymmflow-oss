/**
 * Formatters - Consistent number/currency/date formatting utilities
 * Used across all screens for uniform display
 */

/**
 * Format days value to 1 decimal place
 * @param value - number or string
 * @returns formatted string like "131.0"
 */
export function formatDays(value: number | string | null | undefined): string {
  if (value === null || value === undefined) return '—';
  const num = toNumber(value);
  if (isNaN(num)) return '—';
  return num.toFixed(1);
}

function toNumber(value: number | string | null | undefined): number {
  if (value === null || value === undefined) return NaN;
  if (typeof value === 'number') return Number.isFinite(value) ? value : NaN;
  return parseFloat(String(value).replace(/,/g, ''));
}

/**
 * Format a number with thousands separators.
 * @param value - number or string
 * @param decimals - fixed decimal places
 * @returns formatted string like "1,234.567"
 */
export function formatNumber(value: number | string | null | undefined, decimals = 0): string {
  const num = toNumber(value);
  if (isNaN(num)) {
    return Number(0).toLocaleString('en-US', {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals
    });
  }
  return num.toLocaleString('en-US', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals
  });
}

/**
 * Format BHD currency with 3 decimal places
 * @param value - number or string
 * @returns formatted string like "1,234.567 BHD"
 */
export function formatBHD(value: number | string | null | undefined): string {
  return formatBHDValue(value) + ' BHD';
}

/**
 * Format BHD without the "BHD" suffix
 * @param value - number or string
 * @returns formatted string like "1,234.567"
 */
export function formatBHDValue(value: number | string | null | undefined): string {
  return formatNumber(value, 3);
}

/**
 * Format percentage to 1 decimal place
 * @param value - number or string
 * @returns formatted string like "12.5%"
 */
export function formatPercent(value: number | string | null | undefined): string {
  if (value === null || value === undefined) return '0.0%';
  const num = toNumber(value);
  if (isNaN(num)) return '0.0%';
  return formatNumber(num, 1) + '%';
}

/**
 * Format ratio to 2 decimal places with 'x' suffix
 * @param value - number or string
 * @returns formatted string like "4.70x"
 */
export function formatRatio(value: number | string | null | undefined): string {
  if (value === null || value === undefined) return '0.00x';
  const num = toNumber(value);
  if (isNaN(num)) return '0.00x';
  return formatNumber(num, 2) + 'x';
}

/**
 * Format currency with K/M abbreviations for large numbers
 * @param value - number
 * @returns formatted string like "1.5M" or "250K"
 */
export function formatCurrencyCompact(value: number | null | undefined): string {
  if (value === null || value === undefined) return '0';
  if (value >= 1000000) return `${(value / 1000000).toFixed(1)}M`;
  if (value >= 1000) return `${(value / 1000).toFixed(0)}K`;
  return value.toFixed(0);
}

/**
 * Format number with 1 decimal place
 * @param value - number or string
 * @returns formatted string like "131.0"
 */
export function formatNumber1(value: number | string | null | undefined): string {
  if (value === null || value === undefined) return '—';
  const num = toNumber(value);
  if (isNaN(num)) return '—';
  return formatNumber(num, 1);
}

/**
 * Format date - handles both string and Go Time object
 * @param dateStr - date string or Go Time object
 * @returns formatted string like "05 Feb 2026"
 */
export function formatDate(dateStr: any): string {
  if (!dateStr) return '—';
  // Convert to string if it's an object (Go Time type)
  const dateVal = typeof dateStr === 'string' ? dateStr : String(dateStr);
  const date = new Date(dateVal);
  if (isNaN(date.getTime())) return '—';
  return date.toLocaleDateString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric'
  });
}

/**
 * Safe string display - returns value or em-dash for empty/null
 * @param value - any value
 * @returns string representation or "—"
 */
export function safeString(value: any): string {
  if (value === null || value === undefined || value === '') return '—';
  return String(value);
}
