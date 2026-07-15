/* Accounting bridge module — self-contained: types + mock + real + switch.
 * GL / Chart-of-Accounts / Journal Entries / Reports console (K5 build).
 *
 * DROPPED on purpose: the old screen's VAT summary computed "output VAT" /
 * "input VAT" by string-matching 'sales'/'purchase' inside journal
 * descriptions and multiplying by a hardcoded 10% — a known-fragile
 * heuristic (recon-flagged). It is NOT ported here. A VAT position, if shown
 * at all, must be sourced from real VAT accounts on the Chart of Accounts or
 * the VAT Return export — never re-derived client-side from free text. See
 * screens/parity/Accounting.parity.md. */
import { pick } from './runtime'
import { goDate, goTime, num, str } from './map'
import type { finance } from '$wails/go/models'
import {
  GetChartOfAccounts,
  GetJournalEntries,
  GetPostingCoverageReport,
  GetTrialBalanceGate,
  GetCashflowEvidenceCommandCenter,
  ListCashflowEvidenceProposalReviews,
  GenerateProfitAndLoss,
  GenerateBalanceSheet,
  CreateAccount,
  CreateJournalEntry,
  ReviewCashflowEvidenceProposal,
} from '$wails/go/main/FinanceService'

/* ---- types (camelCase mirrors of the Go models named in BUILD_CONTEXT) ---- */

export interface ChartOfAccountRow {
  id: string
  accountCode: string
  accountName: string
  accountType: string
  balance: number
  isActive: boolean
  isVatAccount: boolean
  vatDirection: string
  accountGroup: string
}

export interface JournalLineRow {
  id: string
  accountId: string
  accountName: string
  debit: number
  credit: number
  description: string
}

export interface JournalEntryRow {
  id: string
  entryNumber: string
  entryDate: string
  description: string
  debitTotal: number
  creditTotal: number
  isPosted: boolean
  fiscalYear: number
  fiscalPeriod: number
  sourceType: string
  lines: JournalLineRow[]
}

export interface CoverageRowItem {
  sourceType: string
  label: string
  total: number
  linked: number
  missing: number
  draftEntries: number
  isComplete: boolean
}

export interface PostingCoverageReport {
  rows: CoverageRowItem[]
  total: number
  linked: number
  missing: number
  draftEntries: number
  isComplete: boolean
}

export interface TrialBalanceGateRow {
  fiscalYear: number
  fiscalPeriod: number
  entryCount: number
  debitTotal: number
  creditTotal: number
  difference: number
  isBalanced: boolean
}

export interface EvidenceSourceRow {
  sourceType: string
  label: string
  required: number
  present: number
  missing: number
  confidence: number
  status: string
}

export interface ActionProposalRow {
  action: string
  label: string
  reason: string
  priority: string
  sourceType: string
  mutatesState: boolean
  requiredDeterministicService: string
}

export interface CashflowCommandCenterRow {
  overallStatus: string
  nextAction: string
  cashTotalAttention: number
  cashOverdueAR: number
  postingMissingJournals: number
  postingDraftEntries: number
  postingStatus: string
  unmatchedBankLines: number
  unmatchedBankAmount: number
  openFollowUpTasks: number
  exportableAuditItems: number
  evidenceSources: EvidenceSourceRow[]
  actionProposals: ActionProposalRow[]
}

export interface CashflowProposalReviewRow {
  id: string
  proposalKey: string
  action: string
  label: string
  status: string
}

export interface PLReportRow {
  year: number
  salesRevenue: number
  otherIncome: number
  totalRevenue: number
  costOfGoodsSold: number
  grossProfit: number
  grossProfitMargin: number
  operatingExpenses: number
  netProfit: number
  netProfitMargin: number
  currency: string
}

export interface BalanceSheetRow {
  asOfDate: string
  year: number
  cash: number
  accountsReceivable: number
  inventory: number
  totalCurrentAssets: number
  totalAssets: number
  accountsPayable: number
  totalCurrentLiabilities: number
  totalLiabilities: number
  retainedEarnings: number
  totalEquity: number
  currency: string
}

export interface NewAccountDraft {
  accountCode: string
  accountName: string
  accountType: string
  isVatAccount: boolean
  vatDirection: string
}

export interface NewJournalLineDraft {
  accountId: string
  accountName: string
  debit: number
  credit: number
  description: string
}

export interface NewJournalEntryDraft {
  entryDate: string
  description: string
  fiscalYear: number
  fiscalPeriod: number
  lines: NewJournalLineDraft[]
}

/** proposal_key composition — mirrors the old screen's proposalReviewKey so
 * a synced review row can be matched back to the action_proposal it was
 * queued from (action + source + required service + label). */
export function proposalKeyOf(p: ActionProposalRow): string {
  return [p.action, p.sourceType, p.requiredDeterministicService, p.label].join('|')
}

/* ---- mock: adversarial + deterministic (see bridge/mock.ts doctrine) ---- */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
const pad = (n: number, w: number): string => String(n).padStart(w, '0')
const THIS_YEAR = new Date().getFullYear()

const ACCOUNT_TYPES = ['Asset', 'Liability', 'Equity', 'Revenue', 'Expense'] as const
const ACCOUNT_NAMES: Record<(typeof ACCOUNT_TYPES)[number], string[]> = {
  Asset: [
    'Cash at Bank — BBK Operating',
    'Cash at Bank — NBB Current',
    'Accounts Receivable',
    'Inventory — Instrumentation',
    'Prepaid Expenses',
    'Fixed Assets — Equipment',
    'VAT Receivable (Input)',
  ],
  Liability: ['Accounts Payable', 'Accrued Expenses', 'VAT Payable (Output)', 'Short-Term Loan', 'Unearned Revenue'],
  Equity: ["Owner's Capital", 'Retained Earnings', 'Current Year Earnings'],
  Revenue: ['Sales Revenue — Instrumentation', 'Sales Revenue — Services', 'Other Income', 'Interest Income'],
  Expense: ['Cost of Goods Sold', 'Salaries & Wages', 'Rent Expense', 'Utilities Expense', 'Bank Charges'],
}
const ACCOUNT_GROUP: Record<string, string> = { Asset: 'BS', Liability: 'BS', Equity: 'BS', Revenue: 'PL', Expense: 'PL' }

let accountsCache: ChartOfAccountRow[] | null = null

function generateAccounts(): ChartOfAccountRow[] {
  const rand = lcg(20260715)
  const rows: ChartOfAccountRow[] = []
  let code = 1000
  let n = 0
  for (const type of ACCOUNT_TYPES) {
    for (const name of ACCOUNT_NAMES[type]) {
      n++
      code += 10
      const isVat = name.includes('VAT')
      const vatDirection = name.includes('Output') ? 'output' : name.includes('Input') ? 'input' : ''
      let balance = Math.round((rand() * 500_000 - 50_000) * 1000) / 1000
      let accountName = name
      // Monster injections — deterministic by running index, not random luck.
      if (n === 3) {
        accountName =
          'International Establishment for Facilities Management, Cleaning Contracts and General Maintenance Services (formerly Gulf Property Care Company) Reconciliation Suspense Clearing Control Account'.padEnd(
            200,
            ' —',
          )
      }
      if (n === 6) accountName = '' // empty name adversary
      if (n === 9) balance = -874_213.5 // overdrawn control account
      if (n === 12) balance = 999_999_999.999 // monster balance
      rows.push({
        id: `coa-${n}`,
        accountCode: `${code}`,
        accountName,
        accountType: type,
        balance,
        isActive: n % 17 !== 0,
        isVatAccount: isVat,
        vatDirection,
        accountGroup: ACCOUNT_GROUP[type]!,
      })
    }
  }
  // Adversarial UNKNOWN_TYPE row — proves the badge/group fallback path.
  n++
  rows.push({
    id: `coa-${n}`,
    accountCode: `9${n}99`,
    accountName: 'Suspense — Unclassified Legacy Import',
    accountType: 'UNKNOWN_TYPE',
    balance: 4_512.75,
    isActive: true,
    isVatAccount: false,
    vatDirection: '',
    accountGroup: 'PL',
  })
  return rows
}

const JOURNAL_SOURCE_TYPES = ['manual', 'sales', 'purchase', 'payment', 'grn', 'bank']

let journalCache: JournalEntryRow[] | null = null
let journalLineSeq = 0

function makeLine(rand: () => number, accounts: ChartOfAccountRow[], debit: number, credit: number, desc: string): JournalLineRow {
  const acct = accounts[Math.floor(rand() * accounts.length)]!
  journalLineSeq++
  return { id: `jl-${journalLineSeq}`, accountId: acct.id, accountName: acct.accountName || 'Unnamed account', debit, credit, description: desc }
}

function generateJournalEntries(): JournalEntryRow[] {
  const accounts = accountsCache ?? (accountsCache = generateAccounts())
  const rand = lcg(20260716)
  const rows: JournalEntryRow[] = []
  const n = 64
  for (let i = 1; i <= n; i++) {
    // Monster indices (1-6) always land in the default fetch year so they're
    // visible on first load; the rest spread across this year + last year.
    const fiscalYear = i <= 6 ? THIS_YEAR : THIS_YEAR - (i % 3 === 0 ? 1 : 0)
    const fiscalPeriod = 1 + Math.floor(rand() * 12)
    const entryDate = `${fiscalYear}-${pad(fiscalPeriod, 2)}-${pad(1 + Math.floor(rand() * 27), 2)}`
    const sourceType = JOURNAL_SOURCE_TYPES[i % JOURNAL_SOURCE_TYPES.length]!
    const isPosted = i % 3 !== 0

    let lines: JournalLineRow[] = []
    let description = `${sourceType} entry — ${entryDate}`
    let debitTotal = 0
    let creditTotal = 0

    if (i === 1) {
      // Monster: intentionally unbalanced — proves the trial-balance gate flags it.
      lines = [makeLine(rand, accounts, 5200, 0, 'Unbalanced debit leg'), makeLine(rand, accounts, 0, 4700, 'Unbalanced credit leg')]
      debitTotal = 5200
      creditTotal = 4700
      description = 'Manual correction — out of balance (unresolved)'
    } else if (i === 2) {
      lines = [makeLine(rand, accounts, 1250, 0, 'Reversal debit'), makeLine(rand, accounts, 0, 1250, 'Reversal credit')]
      debitTotal = 1250
      creditTotal = 1250
      description = `Reversal of JE-${fiscalYear}-0005`
    } else if (i === 3) {
      lines = [] // monster: zero-line entry
      description = 'Draft entry — lines not yet captured'
    } else if (i === 4) {
      // Monster: 40 lines.
      for (let j = 0; j < 20; j++) {
        const amt = Math.round(rand() * 900 * 100) / 100
        lines.push(makeLine(rand, accounts, amt, 0, `Split line ${j + 1}`))
        lines.push(makeLine(rand, accounts, 0, amt, `Split line ${j + 1}`))
        debitTotal += amt
        creditTotal += amt
      }
      description = 'Payroll run — 40-line split across cost centers'
    } else if (i === 5) {
      lines = [makeLine(rand, accounts, 800, 0, 'قيد تسوية'), makeLine(rand, accounts, 0, 800, 'قيد تسوية')]
      debitTotal = 800
      creditTotal = 800
      description = 'قيد يومية لتسوية حساب العميل بعد المطابقة الشهرية'
    } else if (i === 6) {
      lines = [makeLine(rand, accounts, 300, 0, 'Long narrative'), makeLine(rand, accounts, 0, 300, 'Long narrative')]
      debitTotal = 300
      creditTotal = 300
      description =
        'Consolidated adjustment covering three concurrent corrections across the receivables, payables and VAT control accounts following the Q3 reconciliation review meeting with the external auditors and the finance controller'.padEnd(
          200,
          ' —',
        )
    } else {
      const amt = Math.round(rand() * 12_000 * 100) / 100
      lines = [makeLine(rand, accounts, amt, 0, 'Primary leg'), makeLine(rand, accounts, 0, amt, 'Offset leg')]
      debitTotal = amt
      creditTotal = amt
    }

    rows.push({
      id: `je-${i}`,
      entryNumber: `JE-${fiscalYear}-${pad(i, 4)}`,
      entryDate,
      description,
      debitTotal: Math.round(debitTotal * 1000) / 1000,
      creditTotal: Math.round(creditTotal * 1000) / 1000,
      isPosted,
      fiscalYear,
      fiscalPeriod,
      sourceType,
      lines,
    })
  }
  return rows
}

function generateCoverage(): PostingCoverageReport {
  const rows: CoverageRowItem[] = [
    { sourceType: 'sales', label: 'Sales Invoices', total: 142, linked: 142, missing: 0, draftEntries: 0, isComplete: true },
    { sourceType: 'purchases', label: 'Purchase Invoices', total: 88, linked: 85, missing: 3, draftEntries: 1, isComplete: false },
    { sourceType: 'payments', label: 'Customer Payments', total: 96, linked: 96, missing: 0, draftEntries: 0, isComplete: true },
    { sourceType: 'expenses', label: 'Expense Entries', total: 64, linked: 58, missing: 6, draftEntries: 2, isComplete: false },
    { sourceType: 'bank', label: 'Bank Adjustments', total: 21, linked: 21, missing: 0, draftEntries: 0, isComplete: true },
  ]
  return {
    rows,
    total: rows.reduce((s, r) => s + r.total, 0),
    linked: rows.reduce((s, r) => s + r.linked, 0),
    missing: rows.reduce((s, r) => s + r.missing, 0),
    draftEntries: rows.reduce((s, r) => s + r.draftEntries, 0),
    isComplete: rows.every((r) => r.isComplete),
  }
}

/** Monster: a tiny 0.001 fils boundary difference — proves the trial-balance
 * gate's danger tone fires even on a near-zero discrepancy, not just a huge
 * one. Deterministic; not a function of (year, period) beyond echoing them. */
function generateTrialBalance(year: number, period: number): TrialBalanceGateRow {
  return {
    fiscalYear: year,
    fiscalPeriod: period,
    entryCount: 58,
    debitTotal: 412_305.774,
    creditTotal: 412_305.773,
    difference: 0.001,
    isBalanced: false,
  }
}

const STATUS_BAND = ['ready', 'review', 'blocked', 'critical'] as const
function bandFor(n: number): string {
  return STATUS_BAND[((n % STATUS_BAND.length) + STATUS_BAND.length) % STATUS_BAND.length]!
}

function generateCommandCenter(days: number): CashflowCommandCenterRow {
  const overallStatus = bandFor(Math.floor(days / 30))
  const evidenceSources: EvidenceSourceRow[] = [
    { sourceType: 'invoices', label: 'Sales Invoices', required: 42, present: 42, missing: 0, confidence: 0.98, status: 'ready' },
    { sourceType: 'receipts', label: 'Customer Receipts', required: 30, present: 24, missing: 6, confidence: 0.71, status: 'review' },
    { sourceType: 'bank-statements', label: 'Bank Statements', required: 12, present: 9, missing: 3, confidence: 0.55, status: 'blocked' },
    { sourceType: 'expense-receipts', label: 'Expense Receipts', required: 20, present: 6, missing: 14, confidence: 0.22, status: 'critical' },
    { sourceType: 'grn', label: 'Goods Received Notes', required: 15, present: 15, missing: 0, confidence: 0.9, status: 'ready' },
  ]
  const actionProposals: ActionProposalRow[] = [
    {
      action: 'post.sales.reconcile',
      label: 'Post reconciled sales entries',
      reason: '42 invoices matched with zero variance',
      priority: 'low',
      sourceType: 'invoices',
      mutatesState: true,
      requiredDeterministicService: 'FinanceService.CreateJournalEntry',
    },
    {
      action: 'draft.bank.match',
      label: 'Draft bank-match journal',
      reason: '3 statements unmatched over BHD 9,400',
      priority: 'high',
      sourceType: 'bank-statements',
      mutatesState: true,
      requiredDeterministicService: 'FinanceService.CreateJournalEntry',
    },
    {
      action: 'inspect.expense.gap',
      label: 'Inspect missing expense receipts',
      reason: '14 receipts missing supporting evidence',
      priority: 'medium',
      sourceType: 'expense-receipts',
      mutatesState: false,
      requiredDeterministicService: '',
    },
    {
      action: 'export.evidence.pack',
      label: 'Export evidence pack',
      reason: 'Audit window closing in 5 days',
      priority: 'medium',
      sourceType: 'bank-statements',
      mutatesState: false,
      requiredDeterministicService: 'FinanceService.ExportCashflowEvidencePack',
    },
    {
      action: 'review.ar.aging',
      label: 'Review AR aging over 90 days',
      reason: 'BHD 18,200 overdue beyond 90 days',
      priority: 'high',
      sourceType: 'receipts',
      mutatesState: false,
      requiredDeterministicService: '',
    },
    {
      action: 'sync.grn.review',
      label: 'Confirm GRN postings',
      reason: 'All 15 GRNs already linked — informational only',
      priority: 'low',
      sourceType: 'grn',
      mutatesState: false,
      requiredDeterministicService: '',
    },
  ]
  return {
    overallStatus,
    nextAction: overallStatus === 'ready' ? 'No action required this window.' : 'Review the highest-priority proposal below.',
    cashTotalAttention: 27_650.5,
    cashOverdueAR: 18_200,
    postingMissingJournals: 9,
    postingDraftEntries: 3,
    postingStatus: bandFor(Math.floor(days / 30) + 1),
    unmatchedBankLines: 3,
    unmatchedBankAmount: 9_400.25,
    openFollowUpTasks: 5,
    exportableAuditItems: 61,
    evidenceSources,
    actionProposals,
  }
}

let proposalReviewsCache: CashflowProposalReviewRow[] | null = null
let reviewSeq = 0

function seedProposalReviews(): CashflowProposalReviewRow[] {
  const proposals = generateCommandCenter(30).actionProposals
  const first = proposals[0]!
  const third = proposals[2]!
  reviewSeq += 2
  return [
    { id: `cfr-${reviewSeq - 1}`, proposalKey: proposalKeyOf(first), action: first.action, label: first.label, status: 'approved' },
    { id: `cfr-${reviewSeq}`, proposalKey: proposalKeyOf(third), action: third.action, label: third.label, status: 'rejected' },
  ]
}

async function mockFetchAccounts(): Promise<ChartOfAccountRow[]> {
  accountsCache ??= generateAccounts()
  await sleep(220)
  return accountsCache.map((r) => ({ ...r }))
}

async function mockFetchJournalEntries(year: number, period: number, isPosted: boolean | null, limit: number): Promise<JournalEntryRow[]> {
  journalCache ??= generateJournalEntries()
  await sleep(220)
  let rows = journalCache.filter((r) => r.fiscalYear === year)
  if (period > 0) rows = rows.filter((r) => r.fiscalPeriod === period)
  if (isPosted != null) rows = rows.filter((r) => r.isPosted === isPosted)
  return rows.slice(0, limit).map((r) => ({ ...r, lines: r.lines.map((l) => ({ ...l })) }))
}

async function mockFetchCoverage(): Promise<PostingCoverageReport> {
  await sleep(150)
  return generateCoverage()
}

async function mockFetchTrialBalance(year: number, period: number): Promise<TrialBalanceGateRow> {
  await sleep(150)
  return generateTrialBalance(year, period)
}

async function mockFetchCommandCenter(days: number): Promise<CashflowCommandCenterRow> {
  await sleep(200)
  return generateCommandCenter(days)
}

async function mockFetchProposalReviews(days: number, pendingOnly: boolean): Promise<CashflowProposalReviewRow[]> {
  void days
  proposalReviewsCache ??= seedProposalReviews()
  await sleep(150)
  const rows = pendingOnly ? proposalReviewsCache.filter((r) => r.status === 'pending') : proposalReviewsCache
  return rows.map((r) => ({ ...r }))
}

async function mockSyncProposalReviews(days: number): Promise<CashflowProposalReviewRow[]> {
  proposalReviewsCache ??= seedProposalReviews()
  const proposals = generateCommandCenter(days).actionProposals
  for (const p of proposals) {
    const key = proposalKeyOf(p)
    if (!proposalReviewsCache.some((r) => r.proposalKey === key)) {
      reviewSeq++
      proposalReviewsCache.push({ id: `cfr-${reviewSeq}`, proposalKey: key, action: p.action, label: p.label, status: 'pending' })
    }
  }
  await sleep(200)
  return proposalReviewsCache.map((r) => ({ ...r }))
}

async function mockReviewProposal(id: string, status: string, _note: string): Promise<CashflowProposalReviewRow> {
  proposalReviewsCache ??= seedProposalReviews()
  const row = proposalReviewsCache.find((r) => r.id === id)
  if (!row) throw new Error(`Proposal review ${id} not found`)
  await sleep(150)
  row.status = status
  return { ...row }
}

/** Deterministic per-year adversary: year % 7 === 0 forces a zero-revenue
 * year (guards margin division-by-zero); year % 3 === 0 forces a
 * loss-making year (negative net_profit). */
function generatePL(year: number): PLReportRow {
  const rand = lcg(300_000 + year)
  const zeroRevenue = year % 7 === 0
  const salesRevenue = zeroRevenue ? 0 : Math.round(rand() * 900_000 * 100) / 100
  const otherIncome = zeroRevenue ? 0 : Math.round(rand() * 20_000 * 100) / 100
  const totalRevenue = Math.round((salesRevenue + otherIncome) * 100) / 100
  const costOfGoodsSold = zeroRevenue
    ? Math.round(rand() * 40_000 * 100) / 100
    : Math.round(salesRevenue * (0.55 + rand() * 0.15) * 100) / 100
  const grossProfit = Math.round((totalRevenue - costOfGoodsSold) * 100) / 100
  const grossProfitMargin = totalRevenue > 0 ? grossProfit / totalRevenue : 0
  const lossMaking = year % 3 === 0
  const operatingExpenses = lossMaking
    ? Math.round((grossProfit + 10_000 + rand() * 50_000) * 100) / 100
    : Math.round(grossProfit * (0.4 + rand() * 0.3) * 100) / 100
  const netProfit = Math.round((grossProfit - operatingExpenses) * 100) / 100
  const netProfitMargin = totalRevenue > 0 ? netProfit / totalRevenue : 0
  return {
    year,
    salesRevenue,
    otherIncome,
    totalRevenue,
    costOfGoodsSold,
    grossProfit,
    grossProfitMargin,
    operatingExpenses,
    netProfit,
    netProfitMargin,
    currency: 'BHD',
  }
}

function generateBS(year: number): BalanceSheetRow {
  const rand = lcg(500_000 + year)
  const cash = Math.round(rand() * 200_000 * 100) / 100
  const accountsReceivable = Math.round(rand() * 150_000 * 100) / 100
  const inventory = Math.round(rand() * 100_000 * 100) / 100
  const totalCurrentAssets = Math.round((cash + accountsReceivable + inventory) * 100) / 100
  const fixedAssets = Math.round(rand() * 300_000 * 100) / 100
  const totalAssets = Math.round((totalCurrentAssets + fixedAssets) * 100) / 100
  const accountsPayable = Math.round(rand() * 90_000 * 100) / 100
  const totalCurrentLiabilities = accountsPayable
  const longTermLoan = Math.round(rand() * 120_000 * 100) / 100
  const totalLiabilities = Math.round((totalCurrentLiabilities + longTermLoan) * 100) / 100
  const retainedEarnings = Math.round(rand() * 250_000 * 100) / 100
  const totalEquity = Math.round((totalAssets - totalLiabilities) * 100) / 100
  return {
    asOfDate: `${year}-12-31`,
    year,
    cash,
    accountsReceivable,
    inventory,
    totalCurrentAssets,
    totalAssets,
    accountsPayable,
    totalCurrentLiabilities,
    totalLiabilities,
    retainedEarnings,
    totalEquity,
    currency: 'BHD',
  }
}

async function mockGeneratePL(year: number): Promise<PLReportRow> {
  await sleep(300)
  return generatePL(year)
}

async function mockGenerateBS(year: number): Promise<BalanceSheetRow> {
  await sleep(300)
  return generateBS(year)
}

async function mockCreateAccount(draft: NewAccountDraft): Promise<ChartOfAccountRow> {
  accountsCache ??= generateAccounts()
  await sleep(200)
  const row: ChartOfAccountRow = {
    id: `coa-new-${accountsCache.length + 1}`,
    accountCode: draft.accountCode,
    accountName: draft.accountName,
    accountType: draft.accountType,
    balance: 0,
    isActive: true,
    isVatAccount: draft.isVatAccount,
    vatDirection: draft.vatDirection,
    accountGroup: ACCOUNT_GROUP[draft.accountType] ?? 'PL',
  }
  accountsCache.push(row)
  return row
}

async function mockUpdateAccount(id: string, patch: Partial<ChartOfAccountRow>): Promise<void> {
  accountsCache ??= generateAccounts()
  const row = accountsCache.find((a) => a.id === id)
  if (!row) throw new Error(`Account ${id} not found`)
  await sleep(150)
  Object.assign(row, patch)
}

async function mockCreateJournalEntry(draft: NewJournalEntryDraft): Promise<JournalEntryRow> {
  journalCache ??= generateJournalEntries()
  await sleep(250)
  const debitTotal = Math.round(draft.lines.reduce((s, l) => s + l.debit, 0) * 1000) / 1000
  const creditTotal = Math.round(draft.lines.reduce((s, l) => s + l.credit, 0) * 1000) / 1000
  const seq = journalCache.length + 1
  const row: JournalEntryRow = {
    id: `je-new-${seq}`,
    entryNumber: `JE-${draft.fiscalYear}-${pad(seq, 4)}`,
    entryDate: draft.entryDate,
    description: draft.description,
    debitTotal,
    creditTotal,
    isPosted: false,
    fiscalYear: draft.fiscalYear,
    fiscalPeriod: draft.fiscalPeriod,
    sourceType: 'manual',
    lines: draft.lines.map((l, i) => ({ id: `jl-new-${seq}-${i}`, ...l })),
  }
  journalCache.unshift(row)
  return row
}

async function mockExportCashflowEvidencePack(_days: number): Promise<string> {
  await sleep(200)
  return 'exports/cashflow-evidence-pack.zip'
}
async function mockExportBalanceSheetCSV(_year: number): Promise<string> {
  await sleep(200)
  return 'exports/balance-sheet.csv'
}
async function mockExportGeneralLedgerCSV(_year: number): Promise<string> {
  await sleep(200)
  return 'exports/general-ledger.csv'
}
async function mockExportJournalCSV(_year: number): Promise<string> {
  await sleep(200)
  return 'exports/journal.csv'
}
async function mockExportVATReturnData(_year: number, _quarter: number): Promise<string> {
  await sleep(200)
  return 'exports/vat-return.csv'
}

/* ---- real: FETCH bindings wired; every mutation + this multi-shape fetch
 * cluster is INTEG-gapped (honest throw naming the exact binding) ---- */

function mapAccount(r: Record<string, unknown>): ChartOfAccountRow {
  return {
    id: str(r.id),
    accountCode: str(r.account_code),
    accountName: str(r.account_name),
    accountType: str(r.account_type) || 'Asset',
    balance: num(r.balance),
    isActive: Boolean(r.is_active),
    isVatAccount: Boolean(r.is_vat_account),
    vatDirection: str(r.vat_direction),
    accountGroup: str(r.account_group),
  }
}

function mapJournalLine(r: Record<string, unknown>): JournalLineRow {
  return {
    id: str(r.id),
    accountId: str(r.account_id),
    accountName: str(r.account_name),
    debit: num(r.debit),
    credit: num(r.credit),
    description: str(r.description),
  }
}

function mapJournalEntry(r: Record<string, unknown>): JournalEntryRow {
  return {
    id: str(r.id),
    entryNumber: str(r.entry_number),
    entryDate: goDate(r.entry_date),
    description: str(r.description),
    debitTotal: num(r.debit_total),
    creditTotal: num(r.credit_total),
    isPosted: Boolean(r.is_posted),
    fiscalYear: num(r.fiscal_year),
    fiscalPeriod: num(r.fiscal_period),
    sourceType: str(r.source_type),
    lines: Array.isArray(r.lines) ? (r.lines as Record<string, unknown>[]).map(mapJournalLine) : [],
  }
}

function mapCoverageRow(r: Record<string, unknown>): CoverageRowItem {
  return {
    sourceType: str(r.source_type),
    label: str(r.label),
    total: num(r.total),
    linked: num(r.linked),
    missing: num(r.missing),
    draftEntries: num(r.draft_entries),
    isComplete: Boolean(r.is_complete),
  }
}

function mapEvidenceSource(r: Record<string, unknown>): EvidenceSourceRow {
  return {
    sourceType: str(r.source_type),
    label: str(r.label),
    required: num(r.required),
    present: num(r.present),
    missing: num(r.missing),
    confidence: num(r.confidence),
    status: str(r.status),
  }
}

function mapActionProposal(r: Record<string, unknown>): ActionProposalRow {
  return {
    action: str(r.action),
    label: str(r.label),
    reason: str(r.reason),
    priority: str(r.priority),
    sourceType: str(r.source_type),
    mutatesState: Boolean(r.mutates_state),
    requiredDeterministicService: str(r.required_deterministic_service),
  }
}

function mapProposalReview(r: Record<string, unknown>): CashflowProposalReviewRow {
  return {
    id: str(r.id),
    proposalKey: str(r.proposal_key),
    action: str(r.action),
    label: str(r.label),
    status: str(r.status),
  }
}

async function realFetchAccounts(): Promise<ChartOfAccountRow[]> {
  const rows = await GetChartOfAccounts('All')
  return (rows ?? []).map((x) => mapAccount(x as unknown as Record<string, unknown>))
}

async function realFetchJournalEntries(year: number, period: number, isPosted: boolean | null, limit: number): Promise<JournalEntryRow[]> {
  const rows = await GetJournalEntries(year, period, isPosted, limit)
  return (rows ?? []).map((x) => mapJournalEntry(x as unknown as Record<string, unknown>))
}

async function realFetchCoverage(): Promise<PostingCoverageReport> {
  const r = (await GetPostingCoverageReport()) as unknown as Record<string, unknown>
  return {
    rows: Array.isArray(r.rows) ? (r.rows as Record<string, unknown>[]).map(mapCoverageRow) : [],
    total: num(r.total),
    linked: num(r.linked),
    missing: num(r.missing),
    draftEntries: num(r.draft_entries),
    isComplete: Boolean(r.is_complete),
  }
}

async function realFetchTrialBalance(year: number, period: number): Promise<TrialBalanceGateRow> {
  const r = (await GetTrialBalanceGate(year, period)) as unknown as Record<string, unknown>
  return {
    fiscalYear: num(r.fiscal_year),
    fiscalPeriod: num(r.fiscal_period),
    entryCount: num(r.entry_count),
    debitTotal: num(r.debit_total),
    creditTotal: num(r.credit_total),
    difference: num(r.difference),
    isBalanced: Boolean(r.is_balanced),
  }
}

async function realFetchCommandCenter(days: number): Promise<CashflowCommandCenterRow> {
  const r = (await GetCashflowEvidenceCommandCenter(days)) as unknown as Record<string, unknown>
  const cash = (r.cash ?? {}) as Record<string, unknown>
  const posting = (r.posting ?? {}) as Record<string, unknown>
  return {
    overallStatus: str(r.overall_status),
    nextAction: str(r.next_action),
    cashTotalAttention: num(cash.total_attention),
    cashOverdueAR: num(cash.overdue_ar),
    postingMissingJournals: num(posting.missing_journals),
    postingDraftEntries: num(posting.draft_entries),
    postingStatus: str(posting.status),
    unmatchedBankLines: num(r.unmatched_bank_lines),
    unmatchedBankAmount: num(r.unmatched_bank_amount),
    openFollowUpTasks: num(r.open_follow_up_tasks),
    exportableAuditItems: num(r.exportable_audit_items),
    evidenceSources: Array.isArray(r.evidence_sources) ? (r.evidence_sources as Record<string, unknown>[]).map(mapEvidenceSource) : [],
    actionProposals: Array.isArray(r.action_proposals) ? (r.action_proposals as Record<string, unknown>[]).map(mapActionProposal) : [],
  }
}

async function realFetchProposalReviews(days: number, pendingOnly: boolean): Promise<CashflowProposalReviewRow[]> {
  const rows = await ListCashflowEvidenceProposalReviews(days, pendingOnly)
  return (rows ?? []).map((x) => mapProposalReview(x as unknown as Record<string, unknown>))
}

async function realGeneratePL(year: number): Promise<PLReportRow> {
  const r = (await GenerateProfitAndLoss(year)) as unknown as Record<string, unknown>
  return {
    year: num(r.year),
    salesRevenue: num(r.sales_revenue),
    otherIncome: num(r.other_income),
    totalRevenue: num(r.total_revenue),
    costOfGoodsSold: num(r.cost_of_goods_sold),
    grossProfit: num(r.gross_profit),
    grossProfitMargin: num(r.gross_profit_margin),
    operatingExpenses: num(r.operating_expenses),
    netProfit: num(r.net_profit),
    netProfitMargin: num(r.net_profit_margin),
    currency: str(r.currency) || 'BHD',
  }
}

async function realGenerateBS(year: number): Promise<BalanceSheetRow> {
  const r = (await GenerateBalanceSheet(year)) as unknown as Record<string, unknown>
  return {
    asOfDate: goDate(r.as_of_date),
    year: num(r.year),
    cash: num(r.cash),
    accountsReceivable: num(r.accounts_receivable),
    inventory: num(r.inventory),
    totalCurrentAssets: num(r.total_current_assets),
    totalAssets: num(r.total_assets),
    accountsPayable: num(r.accounts_payable),
    totalCurrentLiabilities: num(r.total_current_liabilities),
    totalLiabilities: num(r.total_liabilities),
    retainedEarnings: num(r.retained_earnings),
    totalEquity: num(r.total_equity),
    currency: str(r.currency) || 'BHD',
  }
}

/* ---- real mutations ---- */
async function realCreateAccount(draft: NewAccountDraft): Promise<ChartOfAccountRow> {
  // FinanceService.CreateAccount(finance.ChartOfAccount) → finance.ChartOfAccount.
  // Only the user-authored fields are sent; the backend assigns id/balance/etc.
  const arg = {
    account_code: draft.accountCode,
    account_name: draft.accountName,
    account_type: draft.accountType,
    is_vat_account: draft.isVatAccount,
    vat_direction: draft.vatDirection,
  } as unknown as finance.ChartOfAccount
  const created = await CreateAccount(arg)
  return mapAccount(created as unknown as Record<string, unknown>)
}
async function realUpdateAccount(_id: string, _patch: Partial<ChartOfAccountRow>): Promise<void> {
  // GAP: UpdateAccount(id, Record<string, any>) — the patch arg is an untyped
  // map; neither the accepted key set nor the camelCase→snake_case contract is
  // verifiable from the binding signature. Left gapped rather than guess keys.
  throw new Error('INTEG gap: UpdateAccount — arg2 is an untyped Record<string, any> patch; accepted keys unverifiable from the binding')
}
async function realCreateJournalEntry(draft: NewJournalEntryDraft): Promise<JournalEntryRow> {
  // FinanceService.CreateJournalEntry(finance.JournalEntry) → finance.JournalEntry
  // (double-entry posting). The draft is a 1:1 shape of JournalEntry: each field
  // maps to a named struct field, each line to a named JournalLine field. Totals
  // are the sum of the line legs (the backend enforces debit==credit and rejects
  // an unbalanced entry, which surfaces here as a thrown error).
  const debitTotal = Math.round(draft.lines.reduce((s, l) => s + l.debit, 0) * 1000) / 1000
  const creditTotal = Math.round(draft.lines.reduce((s, l) => s + l.credit, 0) * 1000) / 1000
  const arg = {
    entry_date: goTime(draft.entryDate),
    description: draft.description,
    fiscal_year: draft.fiscalYear,
    fiscal_period: draft.fiscalPeriod,
    debit_total: debitTotal,
    credit_total: creditTotal,
    source_type: 'manual',
    lines: draft.lines.map((l) => ({
      account_id: l.accountId,
      account_name: l.accountName,
      debit: l.debit,
      credit: l.credit,
      description: l.description,
    })),
  } as unknown as finance.JournalEntry
  const created = await CreateJournalEntry(arg)
  return mapJournalEntry(created as unknown as Record<string, unknown>)
}
async function realSyncProposalReviews(_days: number): Promise<CashflowProposalReviewRow[]> {
  throw new Error('INTEG gap: SyncCashflowEvidenceProposalReviews — wires at K5')
}
async function realReviewProposal(id: string, status: string, note: string): Promise<CashflowProposalReviewRow> {
  // FinanceService.ReviewCashflowEvidenceProposal(id, status, note) →
  // main.CashflowEvidenceProposalReview. Not a GL posting; updates a review row.
  const reviewed = await ReviewCashflowEvidenceProposal(id, status, note)
  return mapProposalReview(reviewed as unknown as Record<string, unknown>)
}
async function realExportCashflowEvidencePack(_days: number): Promise<string> {
  throw new Error('INTEG gap: ExportCashflowEvidencePack — wires at K5')
}
async function realExportBalanceSheetCSV(_year: number): Promise<string> {
  throw new Error('INTEG gap: ExportBalanceSheetCSV — wires at K5')
}
async function realExportGeneralLedgerCSV(_year: number): Promise<string> {
  throw new Error('INTEG gap: ExportGeneralLedgerCSV — wires at K5')
}
async function realExportJournalCSV(_year: number): Promise<string> {
  throw new Error('INTEG gap: ExportJournalCSV — wires at K5')
}
async function realExportVATReturnData(_year: number, _quarter: number): Promise<string> {
  throw new Error('INTEG gap: ExportVATReturnData — wires at K5')
}

/* ---- public switched API (viewmodel imports THESE) ---- */
export const fetchChartOfAccounts = (): Promise<ChartOfAccountRow[]> => pick(realFetchAccounts, mockFetchAccounts)()
export const fetchJournalEntries = (year: number, period: number, isPosted: boolean | null, limit: number): Promise<JournalEntryRow[]> =>
  pick(realFetchJournalEntries, mockFetchJournalEntries)(year, period, isPosted, limit)
export const fetchPostingCoverage = (): Promise<PostingCoverageReport> => pick(realFetchCoverage, mockFetchCoverage)()
export const fetchTrialBalanceGate = (year: number, period: number): Promise<TrialBalanceGateRow> =>
  pick(realFetchTrialBalance, mockFetchTrialBalance)(year, period)
export const fetchCashflowCommandCenter = (days: number): Promise<CashflowCommandCenterRow> =>
  pick(realFetchCommandCenter, mockFetchCommandCenter)(days)
export const fetchCashflowProposalReviews = (days: number, pendingOnly: boolean): Promise<CashflowProposalReviewRow[]> =>
  pick(realFetchProposalReviews, mockFetchProposalReviews)(days, pendingOnly)
export const generateProfitAndLoss = (year: number): Promise<PLReportRow> => pick(realGeneratePL, mockGeneratePL)(year)
export const generateBalanceSheet = (year: number): Promise<BalanceSheetRow> => pick(realGenerateBS, mockGenerateBS)(year)

export const createAccount = (draft: NewAccountDraft): Promise<ChartOfAccountRow> => pick(realCreateAccount, mockCreateAccount)(draft)
export const updateAccount = (id: string, patch: Partial<ChartOfAccountRow>): Promise<void> =>
  pick(realUpdateAccount, mockUpdateAccount)(id, patch)
export const createJournalEntry = (draft: NewJournalEntryDraft): Promise<JournalEntryRow> =>
  pick(realCreateJournalEntry, mockCreateJournalEntry)(draft)
export const syncCashflowProposalReviews = (days: number): Promise<CashflowProposalReviewRow[]> =>
  pick(realSyncProposalReviews, mockSyncProposalReviews)(days)
export const reviewCashflowProposal = (id: string, status: string, note: string): Promise<CashflowProposalReviewRow> =>
  pick(realReviewProposal, mockReviewProposal)(id, status, note)
export const exportCashflowEvidencePack = (days: number): Promise<string> =>
  pick(realExportCashflowEvidencePack, mockExportCashflowEvidencePack)(days)
export const exportBalanceSheetCSV = (year: number): Promise<string> =>
  pick(realExportBalanceSheetCSV, mockExportBalanceSheetCSV)(year)
export const exportGeneralLedgerCSV = (year: number): Promise<string> =>
  pick(realExportGeneralLedgerCSV, mockExportGeneralLedgerCSV)(year)
export const exportJournalCSV = (year: number): Promise<string> => pick(realExportJournalCSV, mockExportJournalCSV)(year)
export const exportVATReturnData = (year: number, quarter: number): Promise<string> =>
  pick(realExportVATReturnData, mockExportVATReturnData)(year, quarter)
