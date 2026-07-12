import { formatBHDValue } from '$lib/utils/formatters';

export function formatDate(date: any): string {
  if (!date) return 'N/A';
  try {
    // Handle time.Time objects from Go backend
    const d = typeof date === 'string' ? new Date(date) : new Date(date);
    if (isNaN(d.getTime())) return 'N/A';
    return d.toLocaleDateString('en-US', {
      month: 'short', day: 'numeric', year: 'numeric'
    });
  } catch {
    return 'N/A';
  }
}

export function formatCurrency(value: number): string {
  return formatBHDValue(value || 0);
}
