/* Accounting viewmodel — L5's reactive half for the GL / Chart-of-Accounts /
 * Journal Entries / Reports console. ALL state, derivation, filtering,
 * selection and submit logic lives here; Accounting.svelte only composes
 * primitives and binds this instance (L1/L5 split, mirrors BookBankRecon +
 * Pricing). Named `accounting-vm` (not `accounting.svelte.ts`) so its stem
 * never differs from `Accounting.svelte` by case only — collision-free on
 * Windows' case-insensitive filesystem.
 *
 * The old screen's client-side VAT heuristic (string-matching journal
 * descriptions for 'sales'/'purchase' at a hardcoded 10%) is deliberately
 * NOT reproduced here — see bridge/accounting.ts and
 * screens/parity/Accounting.parity.md. */

import type { Tone } from '$kernel/tones'
import type { ContentClass } from '$kernel/descriptor'
import type { CalloutItem, ListRow, WidgetSegment } from '$kernel/hub'
import { formatMoney } from '$kernel/format'
import {
  createAccount,
  createJournalEntry,
  exportBalanceSheetCSV,
  exportCashflowEvidencePack,
  exportGeneralLedgerCSV,
  exportJournalCSV,
  exportVATReturnData,
  fetchCashflowCommandCenter,
  fetchCashflowProposalReviews,
  fetchChartOfAccounts,
  fetchJournalEntries,
  fetchPostingCoverage,
  fetchTrialBalanceGate,
  generateBalanceSheet,
  generateProfitAndLoss,
  proposalKeyOf,
  reviewCashflowProposal,
  syncCashflowProposalReviews,
  updateAccount,
  type ActionProposalRow,
  type BalanceSheetRow,
  type CashflowCommandCenterRow,
  type CashflowProposalReviewRow,
  type ChartOfAccountRow,
  type CoverageRowItem,
  type JournalEntryRow,
  type NewJournalEntryDraft,
  type PLReportRow,
  type PostingCoverageReport,
  type TrialBalanceGateRow,
} from '../bridge/accounting'

export type AccountingView = 'overview' | 'coa' | 'journal' | 'reports'

export interface VoucherLine {
  accountId: string
  description: string
  debit: number
  credit: number
}

export interface AccountDraft {
  id: string | null
  accountCode: string
  accountName: string
  accountType: string
  isVatAccount: boolean
  vatDirection: string
  isActive: boolean
}

export interface ProposalReviewRow {
  proposal: ActionProposalRow
  key: string
  reviewId: string | null
  reviewStatus: string | null
}

export const KNOWN_ACCOUNT_TYPES = ['Asset', 'Liability', 'Equity', 'Revenue', 'Expense']

function todayIso(): string {
  return new Date().toISOString().slice(0, 10)
}

/* ---- pure helpers (unit-testable, reused by the screen for badges) ---- */

export function accountTypeTone(type: string): Tone {
  switch (type) {
    case 'Asset':
      return 'info'
    case 'Liability':
      return 'warning'
    case 'Equity':
      return 'success'
    case 'Revenue':
      return 'success'
    case 'Expense':
      return 'danger'
    default:
      return 'neutral'
  }
}

/** ready/review/blocked/critical — the cashflow-evidence status vocabulary
 * (see bridge/accounting.ts). Unknown values render neutral, never crash. */
export function statusTone(status: string): Tone {
  switch (status) {
    case 'ready':
      return 'success'
    case 'review':
      return 'warning'
    case 'blocked':
    case 'critical':
      return 'danger'
    default:
      return 'neutral'
  }
}

export function balanceTone(value: number): Tone {
  return value < 0 ? 'danger' : 'neutral'
}

export function coverageRowTone(row: CoverageRowItem): Tone {
  return row.isComplete ? 'success' : 'warning'
}

export class AccountingViewModel {
  loading = $state(true)
  error = $state<string | null>(null)

  activeView = $state<AccountingView>('overview')

  accounts = $state<ChartOfAccountRow[]>([])
  journalEntries = $state<JournalEntryRow[]>([])
  coverage = $state<PostingCoverageReport | null>(null)
  trialBalance = $state<TrialBalanceGateRow | null>(null)
  cashflow = $state<CashflowCommandCenterRow | null>(null)
  proposalReviews = $state<CashflowProposalReviewRow[]>([])
  reviewingKey = $state<string | null>(null)
  reviewError = $state<string | null>(null)

  coaTypeFilter = $state('')
  accountFormOpen = $state(false)
  accountFormMode = $state<'create' | 'edit'>('create')
  accountDraft = $state<AccountDraft | null>(null)
  accountSaving = $state(false)
  accountError = $state<string | null>(null)

  voucherOpen = $state(false)
  voucherDate = $state(todayIso())
  voucherDescription = $state('')
  voucherLines = $state<VoucherLine[]>([])
  voucherSaving = $state(false)
  voucherError = $state<string | null>(null)

  reportYear = $state(new Date().getFullYear())
  plReport = $state<PLReportRow | null>(null)
  bsReport = $state<BalanceSheetRow | null>(null)
  reportLoading = $state('')
  reportError = $state<string | null>(null)
  exportLoading = $state('')
  exportError = $state<string | null>(null)
  exportMessage = $state<string | null>(null)

  /* ---- derived: overview ---- */

  totals = $derived.by(() => {
    const sumType = (t: string) => this.accounts.filter((a) => a.accountType === t).reduce((s, a) => s + a.balance, 0)
    const assets = sumType('Asset')
    const liabilities = sumType('Liability')
    const equity = sumType('Equity')
    const cash = this.accounts
      .filter((a) => a.accountType === 'Asset' && a.accountName.toLowerCase().includes('cash'))
      .reduce((s, a) => s + a.balance, 0)
    return { assets, liabilities, equity, cash }
  })

  overviewStatSections = $derived.by(() => [
    {
      title: 'Balance sheet position',
      items: [
        { label: 'Assets', value: this.totals.assets, content: 'money' as ContentClass },
        { label: 'Liabilities', value: this.totals.liabilities, content: 'money' as ContentClass },
        {
          label: 'Equity',
          value: this.totals.equity,
          content: 'money' as ContentClass,
          tone: balanceTone(this.totals.equity),
        },
        { label: 'Cash', value: this.totals.cash, content: 'money' as ContentClass },
      ],
    },
  ])

  compositionSegments = $derived.by((): WidgetSegment[] => [
    { key: 'assets', label: 'Assets', value: Math.max(0, this.totals.assets), tone: 'info' },
    { key: 'liabilities', label: 'Liabilities', value: Math.max(0, this.totals.liabilities), tone: 'warning' },
    { key: 'equity', label: 'Equity', value: Math.max(0, this.totals.equity), tone: 'success' },
  ])

  coverageSegments = $derived.by(
    (): WidgetSegment[] =>
      this.coverage?.rows.map((r) => ({ key: r.sourceType, label: r.label, value: r.total, tone: coverageRowTone(r) })) ?? [],
  )

  trialBalanceCallout = $derived.by((): CalloutItem[] => {
    const tb = this.trialBalance
    if (!tb) return []
    const tone: Tone = tb.isBalanced ? 'success' : 'danger'
    const label = tb.isBalanced ? 'Trial balance is balanced' : 'Trial balance is out of balance'
    const text = `FY${tb.fiscalYear} — ${tb.entryCount} entries · Debit ${formatMoney(tb.debitTotal)} vs Credit ${formatMoney(tb.creditTotal)} · Difference ${formatMoney(tb.difference)}`
    return [{ label, text, tone }]
  })

  cashflowKpiSections = $derived.by(() => {
    const c = this.cashflow
    if (!c) return []
    return [
      {
        title: `Cashflow evidence — ${c.overallStatus}`,
        items: [
          { label: 'Attention', value: formatMoney(c.cashTotalAttention), tone: statusTone(c.overallStatus) },
          { label: 'Overdue AR', value: formatMoney(c.cashOverdueAR), tone: c.cashOverdueAR > 0 ? ('warning' as Tone) : ('neutral' as Tone) },
          { label: 'Missing journals', value: c.postingMissingJournals, tone: statusTone(c.postingStatus) },
          {
            label: 'Unmatched bank lines',
            value: c.unmatchedBankLines,
            tone: c.unmatchedBankLines > 0 ? ('warning' as Tone) : ('success' as Tone),
          },
          { label: 'Evidence items', value: c.exportableAuditItems },
          {
            label: 'Follow-ups open',
            value: c.openFollowUpTasks,
            tone: c.openFollowUpTasks > 0 ? ('warning' as Tone) : ('neutral' as Tone),
          },
        ],
      },
    ]
  })

  evidenceSourceRows = $derived.by(
    (): ListRow[] =>
      this.cashflow?.evidenceSources.map((s) => ({
        label: s.label,
        detail: `${s.present}/${s.required} present · ${Math.round(s.confidence * 100)}% confidence`,
        value: s.status,
        tone: statusTone(s.status),
      })) ?? [],
  )

  actionProposalRows = $derived.by((): ProposalReviewRow[] =>
    (this.cashflow?.actionProposals ?? []).map((p) => {
      const key = proposalKeyOf(p)
      const review = this.proposalReviews.find((r) => r.proposalKey === key) ?? null
      return { proposal: p, key, reviewId: review?.id ?? null, reviewStatus: review?.status ?? null }
    }),
  )

  /* ---- derived: chart of accounts ---- */

  accountTypeOptions = $derived.by(() => {
    const counts = new Map<string, number>()
    for (const a of this.accounts) counts.set(a.accountType, (counts.get(a.accountType) ?? 0) + 1)
    const known = KNOWN_ACCOUNT_TYPES.filter((t) => counts.has(t)).map((t) => ({ value: t, label: t, count: counts.get(t) ?? 0 }))
    const extra = [...counts.keys()]
      .filter((t) => !KNOWN_ACCOUNT_TYPES.includes(t))
      .map((t) => ({ value: t, label: t, count: counts.get(t) ?? 0 }))
    return [...known, ...extra]
  })

  filteredAccounts = $derived.by(() =>
    this.coaTypeFilter ? this.accounts.filter((a) => a.accountType === this.coaTypeFilter) : this.accounts,
  )

  activeAccounts = $derived.by(() => this.accounts.filter((a) => a.isActive))

  /* ---- load ---- */

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      const [accounts, journalEntries, coverage, trialBalance, cashflow, proposalReviews] = await Promise.all([
        fetchChartOfAccounts(),
        fetchJournalEntries(this.reportYear, 0, null, 100),
        fetchPostingCoverage(),
        fetchTrialBalanceGate(this.reportYear, 0),
        fetchCashflowCommandCenter(30),
        fetchCashflowProposalReviews(30, false),
      ])
      this.accounts = accounts
      this.journalEntries = journalEntries
      this.coverage = coverage
      this.trialBalance = trialBalance
      this.cashflow = cashflow
      this.proposalReviews = proposalReviews
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  selectView(key: string): void {
    this.activeView = key as AccountingView
  }

  setCoaFilter(type: string): void {
    this.coaTypeFilter = type
  }

  /* ---- Chart of Accounts: create/edit (INTEG-gapped mutation) ---- */

  openCreateAccount(): void {
    this.accountDraft = { id: null, accountCode: '', accountName: '', accountType: 'Asset', isVatAccount: false, vatDirection: '', isActive: true }
    this.accountFormMode = 'create'
    this.accountError = null
    this.accountFormOpen = true
  }

  openEditAccount(row: ChartOfAccountRow): void {
    this.accountDraft = {
      id: row.id,
      accountCode: row.accountCode,
      accountName: row.accountName,
      accountType: row.accountType,
      isVatAccount: row.isVatAccount,
      vatDirection: row.vatDirection,
      isActive: row.isActive,
    }
    this.accountFormMode = 'edit'
    this.accountError = null
    this.accountFormOpen = true
  }

  closeAccountForm(): void {
    this.accountFormOpen = false
    this.accountDraft = null
  }

  toggleAccountDraftVat(checked: boolean): void {
    if (!this.accountDraft) return
    this.accountDraft.isVatAccount = checked
    if (!checked) this.accountDraft.vatDirection = ''
  }

  async saveAccount(): Promise<void> {
    const draft = this.accountDraft
    if (!draft) return
    this.accountSaving = true
    this.accountError = null
    try {
      if (this.accountFormMode === 'create') {
        const row = await createAccount({
          accountCode: draft.accountCode,
          accountName: draft.accountName,
          accountType: draft.accountType,
          isVatAccount: draft.isVatAccount,
          vatDirection: draft.vatDirection,
        })
        this.accounts = [...this.accounts, row]
      } else if (draft.id) {
        const id = draft.id
        await updateAccount(id, {
          accountCode: draft.accountCode,
          accountName: draft.accountName,
          accountType: draft.accountType,
          isVatAccount: draft.isVatAccount,
          vatDirection: draft.vatDirection,
          isActive: draft.isActive,
        })
        this.accounts = this.accounts.map((a) =>
          a.id === id
            ? {
                ...a,
                accountCode: draft.accountCode,
                accountName: draft.accountName,
                accountType: draft.accountType,
                isVatAccount: draft.isVatAccount,
                vatDirection: draft.vatDirection,
                isActive: draft.isActive,
              }
            : a,
        )
      }
      this.accountFormOpen = false
      this.accountDraft = null
    } catch (e) {
      this.accountError = e instanceof Error ? e.message : String(e)
    } finally {
      this.accountSaving = false
    }
  }

  /* ---- Journal Entries: voucher creation (INTEG-gapped mutation) ---- */

  blankVoucherLine(): VoucherLine {
    return { accountId: '', description: '', debit: 0, credit: 0 }
  }

  openVoucher(): void {
    this.voucherDate = todayIso()
    this.voucherDescription = ''
    this.voucherLines = [this.blankVoucherLine(), this.blankVoucherLine()]
    this.voucherError = null
    this.voucherOpen = true
  }

  closeVoucher(): void {
    this.voucherOpen = false
  }

  async submitVoucher(): Promise<void> {
    this.voucherSaving = true
    this.voucherError = null
    try {
      const accountsById = new Map(this.accounts.map((a) => [a.id, a]))
      const lines = this.voucherLines.filter((l) => l.accountId && (l.debit > 0 || l.credit > 0))
      const entryDate = this.voucherDate || todayIso()
      const parsedDate = new Date(entryDate)
      const draft: NewJournalEntryDraft = {
        entryDate,
        description: this.voucherDescription,
        fiscalYear: Number.isNaN(parsedDate.getTime()) ? this.reportYear : parsedDate.getFullYear(),
        fiscalPeriod: Number.isNaN(parsedDate.getTime()) ? 1 : parsedDate.getMonth() + 1,
        lines: lines.map((l) => ({
          accountId: l.accountId,
          accountName: accountsById.get(l.accountId)?.accountName ?? '',
          debit: l.debit,
          credit: l.credit,
          description: l.description,
        })),
      }
      const created = await createJournalEntry(draft)
      this.journalEntries = [created, ...this.journalEntries]
      this.voucherOpen = false
    } catch (e) {
      this.voucherError = e instanceof Error ? e.message : String(e)
    } finally {
      this.voucherSaving = false
    }
  }

  /* ---- Cashflow evidence: human review log (INTEG-gapped mutation) ---- */

  async reviewProposal(proposal: ActionProposalRow, status: 'approved' | 'needs_input' | 'rejected'): Promise<void> {
    const key = proposalKeyOf(proposal)
    this.reviewError = null
    let review = this.proposalReviews.find((r) => r.proposalKey === key)
    if (!review) {
      try {
        this.proposalReviews = await syncCashflowProposalReviews(30)
        review = this.proposalReviews.find((r) => r.proposalKey === key)
      } catch (e) {
        this.reviewError = e instanceof Error ? e.message : String(e)
        return
      }
    }
    if (!review) {
      this.reviewError = 'Queue this proposal before reviewing it.'
      return
    }
    this.reviewingKey = key
    try {
      const updated = await reviewCashflowProposal(review.id, status, '')
      this.proposalReviews = [updated, ...this.proposalReviews.filter((r) => r.id !== updated.id)]
    } catch (e) {
      this.reviewError = e instanceof Error ? e.message : String(e)
    } finally {
      this.reviewingKey = null
    }
  }

  /* ---- Reports ---- */

  setReportYear(year: number): void {
    this.reportYear = year
  }

  async generatePLReport(): Promise<void> {
    if (this.reportYear < 2000 || this.reportYear > 2100) {
      this.reportError = 'Report year must be between 2000 and 2100'
      return
    }
    this.reportLoading = 'pl'
    this.reportError = null
    try {
      this.plReport = await generateProfitAndLoss(this.reportYear)
    } catch (e) {
      this.reportError = e instanceof Error ? e.message : String(e)
    } finally {
      this.reportLoading = ''
    }
  }

  async generateBSReport(): Promise<void> {
    if (this.reportYear < 2000 || this.reportYear > 2100) {
      this.reportError = 'Report year must be between 2000 and 2100'
      return
    }
    this.reportLoading = 'balance'
    this.reportError = null
    try {
      this.bsReport = await generateBalanceSheet(this.reportYear)
    } catch (e) {
      this.reportError = e instanceof Error ? e.message : String(e)
    } finally {
      this.reportLoading = ''
    }
  }

  async runExport(kind: 'balance-csv' | 'gl-csv' | 'journal-csv' | 'vat' | 'evidence-pack'): Promise<void> {
    this.exportLoading = kind
    this.exportError = null
    this.exportMessage = null
    try {
      let path = ''
      if (kind === 'balance-csv') path = await exportBalanceSheetCSV(this.reportYear)
      else if (kind === 'gl-csv') path = await exportGeneralLedgerCSV(this.reportYear)
      else if (kind === 'journal-csv') path = await exportJournalCSV(this.reportYear)
      else if (kind === 'vat') path = await exportVATReturnData(this.reportYear, Math.floor(new Date().getMonth() / 3) + 1)
      else path = await exportCashflowEvidencePack(30)
      this.exportMessage = `Exported: ${path}`
    } catch (e) {
      this.exportError = e instanceof Error ? e.message : String(e)
    } finally {
      this.exportLoading = ''
    }
  }
}
