/* Book vs Bank Reconciliation viewmodel — L5's reactive half: list/select/
 * finalize state plus the pure balance arithmetic, no layout. Same split as
 * serial-trace.svelte.ts. BookBankRecon.svelte binds an instance of this and
 * renders on primitives only (L1). The adjusted-balance/variance math and the
 * BalanceComparisonPanel column shaping live here — ONE definition (L2) — so
 * the list's variance-toned column and the detail panel's variance banner
 * can never disagree about what "reconciled" means. */

import type { Tone } from '$kernel/tones'
import {
  fetchBookBankReconciliations,
  finalizeBookBankReconciliation,
  type BookBankReconciliationRow,
  type ReconciliationLine,
} from '../bridge/book-bank-recon'

const RECONCILED_TOLERANCE = 0.001

function sum(lines: ReconciliationLine[]): number {
  return lines.reduce((s, l) => s + l.value, 0)
}

export function adjustedBankBalance(row: BookBankReconciliationRow): number {
  return row.bankBalance + sum(row.depositsInTransit) - sum(row.outstandingCheques)
}

export function adjustedBookBalance(row: BookBankReconciliationRow): number {
  return row.bookBalance + sum(row.bookAdjustments)
}

export function variance(row: BookBankReconciliationRow): number {
  return adjustedBankBalance(row) - adjustedBookBalance(row)
}

export function isReconciled(row: BookBankReconciliationRow): boolean {
  return Math.abs(variance(row)) < RECONCILED_TOLERANCE
}

/** Variance-toned column for the list DataTable, and the same fact renders
 * as the detail panel's banner tone — one predicate, two surfaces. */
export function varianceTone(row: BookBankReconciliationRow): Tone {
  return isReconciled(row) ? 'success' : 'danger'
}

export interface ComparisonColumn {
  title: string
  lines: { label: string; value: number; note?: string }[]
  total: { label: string; value: number }
}

/** Bank-vs-book column pair for BalanceComparisonPanel. Outstanding cheques
 * reduce the bank side, so they're negated into plain addend lines — keeps
 * the panel's own math generic (it just sums nothing; the total is declared,
 * not derived from the lines client-side, in case a caller wants lines that
 * don't foot exactly — e.g. this screen's own empty-lines monster row). */
export function comparisonColumns(row: BookBankReconciliationRow): ComparisonColumn[] {
  return [
    {
      title: 'Bank statement',
      lines: [
        { label: 'Statement balance', value: row.bankBalance },
        ...row.depositsInTransit.map((l) => ({ label: l.label, value: l.value, ...(l.note ? { note: l.note } : {}) })),
        ...row.outstandingCheques.map((l) => ({ label: l.label, value: -l.value, ...(l.note ? { note: l.note } : {}) })),
      ],
      total: { label: 'Adjusted bank balance', value: adjustedBankBalance(row) },
    },
    {
      title: 'Book (GL)',
      lines: [
        { label: 'Book balance', value: row.bookBalance },
        ...row.bookAdjustments.map((l) => ({ label: l.label, value: l.value, ...(l.note ? { note: l.note } : {}) })),
      ],
      total: { label: 'Adjusted book balance', value: adjustedBookBalance(row) },
    },
  ]
}

export function comparisonVariance(row: BookBankReconciliationRow): { label: string; value: number } {
  return { label: 'Variance (bank − book)', value: variance(row) }
}

export class BookBankReconViewModel {
  rows = $state<BookBankReconciliationRow[]>([])
  loading = $state(true)
  error = $state<string | null>(null)
  selectedId = $state<string | null>(null)
  finalizing = $state(false)
  finalizeError = $state<string | null>(null)
  confirmOpen = $state(false)

  selected = $derived(this.rows.find((r) => r.id === this.selectedId) ?? null)

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      this.rows = await fetchBookBankReconciliations()
      if (!this.selectedId && this.rows.length) this.selectedId = this.rows[0]!.id
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
      this.rows = []
    } finally {
      this.loading = false
    }
  }

  select(row: BookBankReconciliationRow): void {
    this.selectedId = row.id
    this.finalizeError = null
  }

  /** Financial hot-zone: finalize is a consequential action, gated behind an
   * explicit ConfirmDialog rather than a bare button click. */
  requestFinalize(): void {
    if (!this.selected || this.selected.status === 'Finalized') return
    this.confirmOpen = true
  }

  cancelFinalize(): void {
    this.confirmOpen = false
  }

  async confirmFinalize(): Promise<void> {
    if (!this.selected) return
    this.confirmOpen = false
    this.finalizing = true
    this.finalizeError = null
    try {
      await finalizeBookBankReconciliation(this.selected.id)
      await this.load()
    } catch (e) {
      this.finalizeError = e instanceof Error ? e.message : String(e)
    } finally {
      this.finalizing = false
    }
  }
}
