/* ContentClass → presentation rules. ONE mapping (L2): every cell in every
 * ledger formats, aligns, and fonts itself through this file. */

import type { ContentClass } from './descriptor'
import { formatDate, formatMoney, formatNumber } from './format'

export function renderCell(content: ContentClass, value: unknown, currency?: string): string {
  switch (content) {
    case 'money':
      return formatMoney(typeof value === 'number' ? value : Number(value), currency ?? 'BHD')
    case 'quantity':
      return formatNumber(typeof value === 'number' ? value : Number(value))
    case 'date':
      return formatDate(value as string | Date | null | undefined)
    default: {
      const s = value == null ? '' : String(value)
      return s === '' ? '—' : s
    }
  }
}

/** Numbers align right; everything else reads left. */
export const cellAlign: Record<ContentClass, 'start' | 'end'> = {
  code: 'start',
  name: 'start',
  money: 'end',
  quantity: 'end',
  date: 'start',
  status: 'start',
  text: 'start',
}

/** Numeric/code cells use the tabular-numeral font stack. */
export function cellFontClass(content: ContentClass): string {
  return content === 'money' || content === 'quantity' || content === 'code' ? 'k-cell-numeric' : ''
}
