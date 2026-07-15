/* Bank Reconciliation viewmodel — L5's reactive half. Owns: account/statement/
 * line selection + loading, the two-phase import flow (preview → confirm/
 * discard, nothing persists until Confirm), the AllocationMatchPanel draft
 * array + its confirm resolution (CreateSplitAllocation for >1 allocation,
 * ManualMatchLine for exactly 1 — both INTEG-gapped), unmatch/auto-match/
 * finalize/delete (all posting-adjacent hot zones, every mutation surfaces
 * its thrown error inline rather than swallowing it — BookBankRecon
 * finalizeError pattern), statement/line CRUD, and an optional per-statement
 * audit-trail drawer (GetAuditTrail is a real binding the old screen never
 * called). No layout — the screen renders this state on primitives only. */

import type { Tone } from '$kernel/tones'
import { allocationKey, type AllocationDraft, type MatchCandidate } from '$kernel/allocation'
import {
  fetchBankAccounts,
  fetchBankStatements,
  fetchBankStatementLines,
  fetchCashPosition,
  fetchMatchCandidates,
  fetchAuditTrail,
  previewBankStatementImport,
  confirmBankStatementImport,
  discardBankStatementImportPreview,
  autoMatchBankLines,
  manualMatchLine,
  createSplitAllocation,
  unmatchLine as bridgeUnmatchLine,
  finalizeReconciliation,
  deleteBankStatement,
  updateBankStatement,
  createBankStatementLine,
  updateBankStatementLine,
  deleteBankStatementLine,
  CANDIDATE_TYPE_OPTIONS,
  SINGLE_SELECT_CANDIDATE_TYPES,
  type BankAccountOption,
  type BankStatementRow,
  type BankStatementLineRow,
  type CashPositionSummary,
  type BankMatchCandidatePool,
  type AuditTrailEntry,
  type StatementImportPreview,
  type BankStatementDraft,
  type BankStatementLineDraft,
} from '../bridge/bank-reconciliation'

const TOLERANCE = 0.001

/** The amount a statement line is matched against — credit-dominant when an
 * OCR-flip line carries both (mirrors the old screen's `credit > 0` branch). */
export function lineAmount(line: BankStatementLineRow): number {
  return line.credit > 0 ? line.credit : line.debit
}

/** Credit lines only ever match customer invoices; debit lines match
 * everything else. Declared once (L2) — the match modal's type filter and
 * the candidate-pool selector agree by construction. */
export function allowedCandidateTypes(line: BankStatementLineRow): string[] {
  return line.credit > 0
    ? ['CUSTOMER_INVOICE']
    : ['SUPPLIER_INVOICE', 'EXPENSE', 'SUPPLIER_PAYMENT', 'PAYROLL_PAYOUT']
}

export const STATEMENT_STATUS_TONES: Record<string, Tone> = {
  Imported: 'info',
  'In Progress': 'warning',
  Reconciled: 'success',
  Verified: 'success',
}

export function matchTone(line: BankStatementLineRow): Tone {
  return line.isMatched ? 'success' : 'danger'
}

const TX_TYPE_TONES: Record<string, Tone> = {
  CUSTOMER_PAYMENT: 'success',
  SUPPLIER_PAYMENT: 'info',
  BANK_FEE: 'warning',
  INTEREST: 'info',
  TRANSFER: 'neutral',
}
export function transactionTypeTone(line: BankStatementLineRow): Tone {
  return TX_TYPE_TONES[line.transactionType] ?? 'neutral'
}

function poolForTypes(pool: BankMatchCandidatePool, types: string[]): MatchCandidate[] {
  const out: MatchCandidate[] = []
  if (types.includes('CUSTOMER_INVOICE')) out.push(...pool.customerInvoices)
  if (types.includes('SUPPLIER_INVOICE')) out.push(...pool.supplierInvoices)
  if (types.includes('SUPPLIER_PAYMENT')) out.push(...pool.supplierPayments)
  if (types.includes('EXPENSE')) out.push(...pool.expenses)
  if (types.includes('PAYROLL_PAYOUT')) out.push(...pool.payrollPayouts)
  return out
}

export class BankReconciliationViewModel {
  // No session/auth primitive exists in the kernel lab yet (K5 gap) — this
  // placeholder stands in for the authenticated actor the real bindings
  // require. Real mutations are INTEG-gapped regardless, so this value never
  // reaches a live backend; it only lets the mock mutations attribute an
  // action the way the old screen's $currentUser.id did.
  readonly actor = 'lab-user'

  bankAccounts = $state<BankAccountOption[]>([])
  loading = $state(true)
  error = $state<string | null>(null)

  cashPosition = $state<CashPositionSummary | null>(null)

  selectedAccountId = $state<string | null>(null)
  statements = $state<BankStatementRow[]>([])
  statementsLoading = $state(false)

  selectedStatementId = $state<string | null>(null)
  lines = $state<BankStatementLineRow[]>([])
  linesLoading = $state(false)

  candidatePool = $state<BankMatchCandidatePool | null>(null)
  candidatesLoading = $state(false)

  selectedAccount = $derived(this.bankAccounts.find((a) => a.id === this.selectedAccountId) ?? null)
  selectedStatement = $derived(this.statements.find((s) => s.id === this.selectedStatementId) ?? null)

  totalMatched = $derived(this.lines.filter((l) => l.isMatched).length)
  totalUnmatched = $derived(this.lines.length - this.totalMatched)
  unmatchedCredit = $derived(this.lines.filter((l) => !l.isMatched && l.credit > 0).reduce((s, l) => s + l.credit, 0))
  matchedPercent = $derived(this.lines.length > 0 ? Math.round((this.totalMatched / this.lines.length) * 100) : 0)

  /* ---- import (two-phase: preview -> confirm/discard) ---- */
  importOpen = $state(false)
  importAccountId = $state<string>('')
  importLoading = $state(false)
  importError = $state<string | null>(null)
  importPreview = $state<StatementImportPreview | null>(null)
  confirmingImport = $state(false)

  /* ---- match modal ---- */
  matchOpen = $state(false)
  matchingLine = $state<BankStatementLineRow | null>(null)
  allocations = $state<AllocationDraft[]>([])
  matchBalanced = $state(false)
  matchSaving = $state(false)
  matchError = $state<string | null>(null)

  matchCandidates = $derived.by((): MatchCandidate[] => {
    const line = this.matchingLine
    if (!line || !this.candidatePool) return []
    if (line.zeroCandidatePool) return []
    return poolForTypes(this.candidatePool, allowedCandidateTypes(line))
  })

  matchCandidateTypeOptions = $derived.by(() => {
    const line = this.matchingLine
    if (!line) return CANDIDATE_TYPE_OPTIONS
    const types = new Set(allowedCandidateTypes(line))
    return CANDIDATE_TYPE_OPTIONS.filter((o) => types.has(o.value))
  })

  singleSelectTypes = SINGLE_SELECT_CANDIDATE_TYPES

  /* ---- unmatch / auto-match / finalize / delete ---- */
  actionError = $state<string | null>(null)
  autoMatching = $state(false)
  autoMatchResult = $state<{ matchedCount: number; unmatchedCount: number } | null>(null)
  unmatching = $state<string | null>(null)

  finalizeConfirmOpen = $state(false)
  finalizing = $state(false)
  finalizeError = $state<string | null>(null)

  deleteConfirmOpen = $state(false)
  deleting = $state(false)
  deleteError = $state<string | null>(null)

  /* ---- statement edit ---- */
  editStatementOpen = $state(false)
  statementDraft = $state<BankStatementDraft>({ openingBalance: 0, closingBalance: 0, periodStart: '', periodEnd: '', status: 'Imported', notes: '' })
  savingStatement = $state(false)
  statementError = $state<string | null>(null)

  /* ---- line add/edit ---- */
  lineModalOpen = $state(false)
  editingLineId = $state<string | null>(null)
  editingLineWasMatched = $state(false)
  lineDraft = $state<BankStatementLineDraft>({ transactionDate: '', description: '', reference: '', debit: 0, credit: 0 })
  savingLine = $state(false)
  lineError = $state<string | null>(null)

  /* ---- audit trail drawer (optional, real GetAuditTrail binding) ---- */
  auditOpen = $state(false)
  auditEntries = $state<AuditTrailEntry[]>([])
  auditLoading = $state(false)
  auditError = $state<string | null>(null)

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      const [accounts, cash] = await Promise.all([fetchBankAccounts(), fetchCashPosition()])
      this.bankAccounts = accounts
      this.cashPosition = cash
      const firstActive = accounts.find((a) => a.isActive) ?? accounts[0] ?? null
      if (firstActive) {
        this.selectedAccountId = firstActive.id
        await this.loadStatements()
      }
      void this.loadCandidates()
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  async loadCandidates(): Promise<void> {
    this.candidatesLoading = true
    try {
      this.candidatePool = await fetchMatchCandidates()
    } finally {
      this.candidatesLoading = false
    }
  }

  async selectAccount(accountId: string): Promise<void> {
    if (accountId === this.selectedAccountId) return
    this.selectedAccountId = accountId
    this.selectedStatementId = null
    this.lines = []
    await this.loadStatements()
  }

  async loadStatements(): Promise<void> {
    if (!this.selectedAccountId) return
    this.statementsLoading = true
    try {
      this.statements = await fetchBankStatements(this.selectedAccountId)
      if (this.statements.length > 0) {
        this.selectedStatementId = this.statements[0]!.id
        await this.loadLines()
      } else {
        this.selectedStatementId = null
        this.lines = []
      }
    } finally {
      this.statementsLoading = false
    }
  }

  async selectStatement(row: BankStatementRow): Promise<void> {
    this.selectedStatementId = row.id
    this.actionError = null
    this.autoMatchResult = null
    await this.loadLines()
  }

  async loadLines(): Promise<void> {
    if (!this.selectedStatementId) return
    this.linesLoading = true
    try {
      this.lines = await fetchBankStatementLines(this.selectedStatementId)
    } finally {
      this.linesLoading = false
    }
  }

  /* ---- import ---- */

  openImport(): void {
    this.importAccountId = this.selectedAccountId ?? this.bankAccounts[0]?.id ?? ''
    this.importError = null
    this.importOpen = true
  }

  closeImport(): void {
    this.importOpen = false
  }

  async runPreview(): Promise<void> {
    if (!this.importAccountId) return
    this.importLoading = true
    this.importError = null
    try {
      this.importPreview = await previewBankStatementImport(this.importAccountId)
      this.importOpen = false
    } catch (e) {
      this.importError = e instanceof Error ? e.message : String(e)
    } finally {
      this.importLoading = false
    }
  }

  async confirmImport(): Promise<void> {
    const preview = this.importPreview
    if (!preview) return
    this.confirmingImport = true
    try {
      const importedAccountId = preview.bankAccountId
      await confirmBankStatementImport(preview.id)
      this.importPreview = null
      this.selectedAccountId = importedAccountId
      await this.loadStatements()
      this.cashPosition = await fetchCashPosition()
    } catch (e) {
      this.importError = e instanceof Error ? e.message : String(e)
    } finally {
      this.confirmingImport = false
    }
  }

  async cancelImportPreview(): Promise<void> {
    const preview = this.importPreview
    if (preview) {
      try {
        await discardBankStatementImportPreview(preview.id)
      } catch {
        // best-effort discard — nothing was persisted either way
      }
    }
    this.importPreview = null
  }

  /* ---- match modal ---- */

  openMatch(line: BankStatementLineRow): void {
    this.matchingLine = line
    this.allocations = []
    this.matchBalanced = false
    this.matchError = null
    this.matchOpen = true
  }

  closeMatch(): void {
    this.matchOpen = false
    this.matchingLine = null
    this.allocations = []
  }

  addAllocation(candidate: MatchCandidate, amount: number): void {
    const key = allocationKey(candidate.type, candidate.id)
    if (this.allocations.some((a) => a.key === key)) return
    const single = this.singleSelectTypes.includes(candidate.type)
    const draft: AllocationDraft = {
      key,
      candidateId: candidate.id,
      candidateType: candidate.type,
      label: candidate.label,
      amount: single ? candidate.amount : amount,
      maxAmount: candidate.amount,
    }
    this.allocations = single ? [draft] : [...this.allocations, draft]
  }

  changeAllocationAmount(key: string, amount: number): void {
    this.allocations = this.allocations.map((a) => (a.key === key ? { ...a, amount } : a))
  }

  removeAllocation(key: string): void {
    this.allocations = this.allocations.filter((a) => a.key !== key)
  }

  async confirmMatch(): Promise<void> {
    const line = this.matchingLine
    if (!line || this.allocations.length === 0) return
    this.matchSaving = true
    this.matchError = null
    try {
      if (this.allocations.length === 1) {
        const a = this.allocations[0]!
        await manualMatchLine(line.id, a.candidateType, a.candidateId, this.actor)
      } else {
        await createSplitAllocation(
          line.id,
          this.allocations.map((a) => ({ allocationType: a.candidateType, entityId: a.candidateId, allocatedAmount: a.amount })),
          this.actor,
        )
      }
      this.closeMatch()
      await this.loadLines()
    } catch (e) {
      this.matchError = e instanceof Error ? e.message : String(e)
    } finally {
      this.matchSaving = false
    }
  }

  async unmatch(line: BankStatementLineRow): Promise<void> {
    this.unmatching = line.id
    this.actionError = null
    try {
      await bridgeUnmatchLine(line.id, this.actor, 'Manual unmatch')
      await this.loadLines()
    } catch (e) {
      this.actionError = e instanceof Error ? e.message : String(e)
    } finally {
      this.unmatching = null
    }
  }

  async autoMatch(): Promise<void> {
    const statementId = this.selectedStatementId
    if (!statementId) return
    this.autoMatching = true
    this.actionError = null
    this.autoMatchResult = null
    try {
      const result = await autoMatchBankLines(statementId)
      this.autoMatchResult = { matchedCount: result.matchedCount, unmatchedCount: result.unmatchedCount }
      await this.loadLines()
    } catch (e) {
      this.actionError = e instanceof Error ? e.message : String(e)
    } finally {
      this.autoMatching = false
    }
  }

  /* ---- finalize (HOT-ZONE: gated on zero unmatched lines) ---- */

  requestFinalize(): void {
    if (!this.selectedStatement || this.totalUnmatched > 0) return
    this.finalizeError = null
    this.finalizeConfirmOpen = true
  }

  cancelFinalize(): void {
    this.finalizeConfirmOpen = false
  }

  async confirmFinalize(): Promise<void> {
    const statement = this.selectedStatement
    if (!statement) return
    this.finalizeConfirmOpen = false
    this.finalizing = true
    this.finalizeError = null
    try {
      await finalizeReconciliation(statement.id, this.actor)
      await this.loadStatements()
    } catch (e) {
      this.finalizeError = e instanceof Error ? e.message : String(e)
    } finally {
      this.finalizing = false
    }
  }

  /* ---- delete statement (HOT-ZONE) ---- */

  requestDelete(): void {
    if (!this.selectedStatement) return
    this.deleteError = null
    this.deleteConfirmOpen = true
  }

  cancelDelete(): void {
    this.deleteConfirmOpen = false
  }

  async confirmDelete(): Promise<void> {
    const statement = this.selectedStatement
    if (!statement) return
    this.deleteConfirmOpen = false
    this.deleting = true
    this.deleteError = null
    try {
      await deleteBankStatement(statement.id)
      this.selectedStatementId = null
      this.lines = []
      await this.loadStatements()
    } catch (e) {
      this.deleteError = e instanceof Error ? e.message : String(e)
    } finally {
      this.deleting = false
    }
  }

  /* ---- statement edit ---- */

  openEditStatement(): void {
    const s = this.selectedStatement
    if (!s) return
    this.statementDraft = {
      openingBalance: s.openingBalance,
      closingBalance: s.closingBalance,
      periodStart: s.periodStart,
      periodEnd: s.periodEnd,
      status: s.status,
      notes: s.notes,
    }
    this.statementError = null
    this.editStatementOpen = true
  }

  closeEditStatement(): void {
    this.editStatementOpen = false
  }

  async saveStatement(): Promise<void> {
    const s = this.selectedStatement
    if (!s) return
    this.savingStatement = true
    this.statementError = null
    try {
      await updateBankStatement(s.id, this.statementDraft)
      this.editStatementOpen = false
      await this.loadStatements()
    } catch (e) {
      this.statementError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingStatement = false
    }
  }

  /* ---- line add/edit ---- */

  openAddLine(): void {
    if (!this.selectedStatementId) return
    this.editingLineId = null
    this.editingLineWasMatched = false
    this.lineDraft = { transactionDate: '', description: '', reference: '', debit: 0, credit: 0 }
    this.lineError = null
    this.lineModalOpen = true
  }

  openEditLine(line: BankStatementLineRow): void {
    this.editingLineId = line.id
    this.editingLineWasMatched = line.isMatched
    this.lineDraft = {
      transactionDate: line.transactionDate,
      description: line.description,
      reference: line.reference,
      debit: line.debit,
      credit: line.credit,
    }
    this.lineError = null
    this.lineModalOpen = true
  }

  closeLineModal(): void {
    this.lineModalOpen = false
  }

  async saveLine(): Promise<void> {
    if (!this.lineDraft.transactionDate || !this.lineDraft.description) {
      this.lineError = 'Transaction date and description are required.'
      return
    }
    if (this.lineDraft.debit === 0 && this.lineDraft.credit === 0) {
      this.lineError = 'Either debit or credit must be non-zero.'
      return
    }
    this.savingLine = true
    this.lineError = null
    try {
      if (this.editingLineId) {
        await updateBankStatementLine(this.editingLineId, this.lineDraft)
      } else {
        if (!this.selectedStatementId) return
        await createBankStatementLine(this.selectedStatementId, this.lineDraft)
      }
      this.lineModalOpen = false
      await this.loadLines()
    } catch (e) {
      this.lineError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingLine = false
    }
  }

  async deleteLine(): Promise<void> {
    if (!this.editingLineId) return
    this.savingLine = true
    this.lineError = null
    try {
      await deleteBankStatementLine(this.editingLineId)
      this.lineModalOpen = false
      await this.loadLines()
    } catch (e) {
      this.lineError = e instanceof Error ? e.message : String(e)
    } finally {
      this.savingLine = false
    }
  }

  /* ---- audit trail drawer ---- */

  async openAuditTrail(): Promise<void> {
    if (!this.selectedStatementId) return
    this.auditOpen = true
    this.auditLoading = true
    this.auditError = null
    try {
      this.auditEntries = await fetchAuditTrail(this.selectedStatementId)
    } catch (e) {
      this.auditError = e instanceof Error ? e.message : String(e)
      this.auditEntries = []
    } finally {
      this.auditLoading = false
    }
  }

  closeAuditTrail(): void {
    this.auditOpen = false
  }
}
