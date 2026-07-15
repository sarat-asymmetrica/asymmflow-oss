/* Bank Reconciliation bridge module — self-contained: types + mock + real +
 * switch. Transaction-level statement reconciliation: import a bank
 * statement (two-phase preview → confirm/discard, nothing persists until
 * Confirm), then match each statement LINE to an invoice/payment/expense/
 * payroll payout — one line ↔ one-or-many candidate documents via
 * AllocationMatchPanel. NOT BookBankReconciliation (bridge/book-bank-recon.ts),
 * which compares two month-end running totals; this bridge works the line
 * level (recon: old BankReconciliationScreen.svelte, 2140 lines).
 *
 * All FETCH bindings are real (single or parallel-merged calls — same shape
 * cheque-register.ts's realFetchAll uses for GetActiveBankAccounts →
 * GetOutstandingCheques). Every MUTATION is posting-adjacent and INTEG-gapped
 * on the real side, naming the exact binding; the MOCK side actually performs
 * the mutation against the cached dataset so the screen is interactive to
 * demo (book-bank-recon.ts / cheque-register.ts pattern). Adversarial by
 * doctrine: monsters woven into the dataset (see bridge/mock.ts). Deterministic
 * (seeded LCG). Synthetic-only data (SYNTHETIC_IDENTITY.md) — Gulf-style
 * placeholder names, never real companies/people/banks. */

import { pick } from './runtime'
import { goDate, num, str } from './map'
import type { MatchCandidate } from '$kernel/allocation'
// Only FETCH bindings are imported — every MUTATION below is an honest
// INTEG-gap throw naming the binding as a string (bank-accounts.ts /
// audit-trail.ts pattern: the real adapter never calls a binding it can't
// safely pass through).
import {
  GetActiveBankAccounts,
  GetBankStatements,
  GetBankStatementLines,
  GetCashPosition,
  ListCustomerInvoices,
  GetSupplierInvoices,
  GetAllSupplierPayments,
  ListExpenseEntries,
  ListUnreconciledPayrollPayouts,
  GetAuditTrail,
} from '$wails/go/main/FinanceService'

/* ---------------------------------------------------------------------- */
/* Types                                                                   */
/* ---------------------------------------------------------------------- */

export interface BankAccountOption {
  id: string
  bankName: string
  accountName: string
  accountNumber: string
  currency: string
  isActive: boolean
  division: string
}

export interface BankStatementRow {
  id: string
  bankAccountId: string
  statementNumber: string
  periodStart: string
  periodEnd: string
  openingBalance: number
  closingBalance: number
  currency: string
  status: string
  discrepancyAmount: number
  notes: string
  division: string
}

export interface BankStatementLineRow {
  id: string
  bankStatementId: string
  lineNumber: number
  transactionDate: string
  description: string
  reference: string
  debit: number
  credit: number
  balance: number
  transactionType: string
  isMatched: boolean
  matchConfidence: number
  matchedInvoiceIds: string
  extractedCustomer: string
  /** Adversarial marker (not a real Go field): forces the match candidate
   * pool to render empty for this line regardless of type, exercising
   * AllocationMatchPanel's own empty-state (recon adversarial spec). */
  zeroCandidatePool?: boolean
}

export interface CashAccountBalance {
  accountId: string
  accountName: string
  balanceBhd: number
  notice?: string
}

export interface CashPositionSummary {
  totalBhd: number
  byAccount: CashAccountBalance[]
  notices: string[]
}

export interface StatementImportPreview {
  id: string
  bankAccountId: string
  statementNumber: string
  periodStart: string
  periodEnd: string
  openingBalance: number
  closingBalance: number
  lines: BankStatementLineRow[]
}

export interface AuditTrailEntry {
  id: string
  timestamp: string
  action: string
  actor: string
  detail: string
  isReversed: boolean
}

export type CandidateType = 'CUSTOMER_INVOICE' | 'SUPPLIER_INVOICE' | 'SUPPLIER_PAYMENT' | 'EXPENSE' | 'PAYROLL_PAYOUT'

export interface BankMatchCandidatePool {
  customerInvoices: MatchCandidate[]
  supplierInvoices: MatchCandidate[]
  supplierPayments: MatchCandidate[]
  expenses: MatchCandidate[]
  payrollPayouts: MatchCandidate[]
}

export interface SplitAllocationInput {
  allocationType: string
  entityId: string
  allocatedAmount: number
}

export interface AutoMatchResult {
  matchedCount: number
  unmatchedCount: number
  totalLines: number
  matchedPercent: number
}

export interface BankStatementLineDraft {
  transactionDate: string
  description: string
  reference: string
  debit: number
  credit: number
}

export interface BankStatementDraft {
  openingBalance: number
  closingBalance: number
  periodStart: string
  periodEnd: string
  status: string
  notes: string
}

/* ---------------------------------------------------------------------- */
/* mock: adversarial + deterministic (see bridge/mock.ts)                  */
/* ---------------------------------------------------------------------- */

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
function hashSeed(text: string): number {
  let h = 0
  for (let i = 0; i < text.length; i++) h = (h * 31 + text.charCodeAt(i)) >>> 0
  return h
}
const pad = (n: number, w: number): string => String(n).padStart(w, '0')

const DIVISIONS = ['Acme Instrumentation', 'Beacon Controls']

const BANK_ACCOUNTS_SEED: { bankName: string; accountName: string }[] = [
  { bankName: 'Ahli United Bank', accountName: 'Main Operating Account' },
  { bankName: 'National Bank of Bahrain', accountName: 'Trade Settlement Account' },
  { bankName: 'BBK', accountName: 'Payroll Account' },
  { bankName: 'HSBC Bahrain', accountName: 'USD Trade Account' },
  { bankName: 'Standard Chartered', accountName: 'EUR Reserve Account' },
  { bankName: '', accountName: 'Petty Cash Reserve' },
]

const DESCRIPTIONS = [
  'Customer payment received — wire transfer',
  'EFT — supplier settlement',
  'Bank service charge',
  'Standing order — office rent',
  'Payroll transfer batch',
  'Interest credited',
  'POS settlement batch',
  'Cheque deposit — cleared',
  'Inter-account transfer',
  'Loan repayment installment',
]

const EXTRACTED_CUSTOMERS = [
  'Al Dana Trading Co.',
  'Sitra Contracting W.L.L.',
  'Manama Precision Instruments',
  'شركة الخليج للتجارة والمقاولات ذ.م.م',
  'مؤسسة الدرة للتوريدات الصناعية',
  '',
]

const TX_TYPES = ['CUSTOMER_PAYMENT', 'SUPPLIER_PAYMENT', 'BANK_FEE', 'INTEREST', 'TRANSFER', 'UNKNOWN']

const CUSTOMER_NAMES = [
  'Al Dana Trading Co.',
  'Sitra Contracting W.L.L.',
  'Manama Precision Instruments',
  'Zallaq Industrial Supplies',
  'Riffa Engineering Services',
  'شركة الخليج للتجارة والمقاولات ذ.م.م',
  '',
]
const SUPPLIER_NAMES = [
  'Bahrain Precision Instruments W.L.L.',
  'Gulf Valve & Actuator Trading Co.',
  'Al Manar Industrial Supplies',
  'International Establishment for Process Control Equipment, Calibration Services, Spare Parts Distribution and General Engineering Trading (formerly Gulf Technical Instrumentation Company) W.L.L.',
  'مؤسسة الخليج للمعدات الصناعية ذ.م.م',
  '',
]
const EMPLOYEE_NAMES = ['Ahmed Al-Khalifa', 'Fatima Hassan', 'Mohammed Bucheeri', 'Aisha Al-Rumaihi', '']
const EXPENSE_VENDORS = ['Bahrain Electricity & Water Authority', 'Gulf Office Supplies', 'Municipal Council — License Fees', '']

/* ---- Bank accounts + statements + lines ---- */

interface MockDataset {
  accounts: BankAccountOption[]
  statements: BankStatementRow[]
  lines: BankStatementLineRow[]
}

let cache: MockDataset | null = null
let previewCounter = 0
const pendingPreviews = new Map<string, StatementImportPreview>()

function makeMonsterLine(
  id: string,
  statementId: string,
  lineNumber: number,
  overrides: Partial<BankStatementLineRow>,
): BankStatementLineRow {
  return {
    id,
    bankStatementId: statementId,
    lineNumber,
    transactionDate: '2026-06-01',
    description: 'Transaction',
    reference: `REF-${id}`,
    debit: 0,
    credit: 0,
    balance: 0,
    transactionType: 'CUSTOMER_PAYMENT',
    isMatched: false,
    matchConfidence: 0,
    matchedInvoiceIds: '',
    extractedCustomer: '',
    ...overrides,
  }
}

function generate(): MockDataset {
  const rand = lcg(20260715)
  const accounts: BankAccountOption[] = BANK_ACCOUNTS_SEED.map((seed, i) => ({
    id: `bankacct-${i + 1}`,
    bankName: seed.bankName,
    accountName: seed.accountName,
    accountNumber: pad(Math.floor(rand() * 1e10), 12),
    currency: i === 3 ? 'USD' : i === 4 ? 'EUR' : 'BHD',
    isActive: i !== 5,
    division: DIVISIONS[i % DIVISIONS.length]!,
  }))

  const statements: BankStatementRow[] = []
  const lines: BankStatementLineRow[] = []
  let statementCounter = 0
  let lineCounter = 0

  for (const account of accounts) {
    const statementsForAccount = 3 + Math.floor(rand() * 2) // 3-4
    for (let s = 0; s < statementsForAccount; s++) {
      statementCounter++
      const statementId = `stmt-${statementCounter}`
      const monthIdx = statementCounter % 12
      const year = 2026 + Math.floor((monthIdx + s) / 12)
      const month = (monthIdx % 12) + 1
      const periodStart = `${year}-${pad(month, 2)}-01`
      const periodEnd = `${year}-${pad(month, 2)}-${pad(27 + (statementCounter % 3), 2)}`
      const opening = Math.round(rand() * 400_000 * 100) / 100
      const statuses = ['Imported', 'In Progress', 'Reconciled', 'Verified']
      const status = statementCounter % 47 === 0 ? 'UNKNOWN_STATE' : statuses[Math.floor(rand() * statuses.length)]!
      // Monster: one statement anywhere carries a non-zero discrepancy —
      // the cash-position/statement-check notice path.
      const discrepancyAmount = statementCounter === 5 ? 128.45 : 0

      const isFirstEver = statementCounter === 1

      const statementLines: BankStatementLineRow[] = []
      if (isFirstEver) {
        // Statement #1: hand-placed monster set — reachable on first paint
        // (this screen auto-selects the first account's first statement).
        statementLines.push(
          makeMonsterLine(`line-${++lineCounter}`, statementId, 1, {
            transactionDate: periodStart,
            description: 'Customer payment received — wire transfer',
            reference: 'REF-0001',
            credit: 15000.5,
            isMatched: true,
            matchConfidence: 0.94,
            matchedInvoiceIds: 'cust-inv-1',
            extractedCustomer: CUSTOMER_NAMES[0]!,
          }),
          makeMonsterLine(`line-${++lineCounter}`, statementId, 2, {
            transactionDate: periodStart,
            description: '', // empty description
            reference: '',
            debit: 42.75,
          }),
          makeMonsterLine(`line-${++lineCounter}`, statementId, 3, {
            transactionDate: periodStart,
            // 200-char unbroken description
            description: 'X'.repeat(200),
            reference: 'REF-0003',
            credit: 980.0,
            extractedCustomer: CUSTOMER_NAMES[5]!, // RTL
          }),
          makeMonsterLine(`line-${++lineCounter}`, statementId, 4, {
            transactionDate: periodStart,
            description: 'Very large wire transfer — annual contract settlement',
            reference: 'REF-0004',
            credit: 999999.999, // huge
          }),
          makeMonsterLine(`line-${++lineCounter}`, statementId, 5, {
            transactionDate: periodStart,
            description: 'Rounding adjustment',
            reference: 'REF-0005',
            debit: 0.001, // tiny
          }),
          makeMonsterLine(`line-${++lineCounter}`, statementId, 6, {
            transactionDate: periodStart,
            description: 'OCR parse — polarity uncertain',
            reference: 'REF-0006',
            debit: 500.0,
            credit: 500.0, // OCR flip: both nonzero
            transactionType: 'UNKNOWN',
          }),
          makeMonsterLine(`line-${++lineCounter}`, statementId, 7, {
            transactionDate: periodStart,
            description: 'Unclassified transaction',
            reference: 'REF-0007',
            debit: 310.25,
            transactionType: 'UNKNOWN',
          }),
          makeMonsterLine(`line-${++lineCounter}`, statementId, 8, {
            transactionDate: periodStart,
            description: 'Supplier settlement — batch 1',
            reference: 'REF-DUPLICATE-01',
            debit: 2200.0,
          }),
          makeMonsterLine(`line-${++lineCounter}`, statementId, 9, {
            transactionDate: periodStart,
            description: 'Supplier settlement — batch 2 (duplicate reference)',
            reference: 'REF-DUPLICATE-01',
            debit: 1150.0,
          }),
          makeMonsterLine(`line-${++lineCounter}`, statementId, 10, {
            transactionDate: periodStart,
            description: 'Matched to a since-deleted invoice',
            reference: 'REF-0010',
            credit: 3400.0,
            isMatched: true,
            matchConfidence: 0.61,
            matchedInvoiceIds: 'cust-inv-does-not-exist', // orphan match
            extractedCustomer: CUSTOMER_NAMES[1]!,
          }),
          makeMonsterLine(`line-${++lineCounter}`, statementId, 11, {
            transactionDate: periodStart,
            description: 'Unidentified inbound credit',
            reference: 'REF-0011',
            credit: 76.0,
            zeroCandidatePool: true, // AllocationMatchPanel empty-state
          }),
        )
      }

      const fillCount = isFirstEver ? 8 : 6 + Math.floor(rand() * 8)
      for (let l = 0; l < fillCount; l++) {
        lineCounter++
        const isCredit = rand() > 0.55
        const amount = Math.round((10 + rand() * 8000) * 1000) / 1000
        const matched = rand() > 0.4
        statementLines.push({
          id: `line-${lineCounter}`,
          bankStatementId: statementId,
          lineNumber: statementLines.length + 1,
          transactionDate: periodStart,
          description: DESCRIPTIONS[Math.floor(rand() * DESCRIPTIONS.length)]!,
          reference: `REF-${pad(lineCounter, 5)}`,
          debit: isCredit ? 0 : amount,
          credit: isCredit ? amount : 0,
          balance: 0,
          transactionType: TX_TYPES[Math.floor(rand() * TX_TYPES.length)]!,
          isMatched: matched,
          matchConfidence: matched ? Math.round((0.55 + rand() * 0.45) * 100) / 100 : 0,
          matchedInvoiceIds: matched ? `cust-inv-${1 + Math.floor(rand() * 200)}` : '',
          extractedCustomer: isCredit ? EXTRACTED_CUSTOMERS[Math.floor(rand() * EXTRACTED_CUSTOMERS.length)]! : '',
        })
      }

      // Running balance for display purposes only.
      let running = opening
      for (const l of statementLines) {
        running = running + l.credit - l.debit
        l.balance = Math.round(running * 1000) / 1000
      }

      const totalDebits = statementLines.reduce((s, l) => s + l.debit, 0)
      const totalCredits = statementLines.reduce((s, l) => s + l.credit, 0)

      statements.push({
        id: statementId,
        bankAccountId: account.id,
        statementNumber: `STMT-${year}-${pad(month, 2)}-${pad(statementCounter, 3)}`,
        periodStart,
        periodEnd,
        openingBalance: opening,
        closingBalance: Math.round((opening + totalCredits - totalDebits) * 1000) / 1000,
        currency: account.currency,
        status,
        discrepancyAmount,
        notes: statementCounter % 9 === 0 ? 'OCR summary: parsed from scanned PDF, confidence 82%.' : '',
        division: account.division,
      })
      lines.push(...statementLines)
    }
  }

  return { accounts, statements, lines }
}

async function mockFetchAccounts(): Promise<BankAccountOption[]> {
  cache ??= generate()
  await sleep(200)
  return [...cache.accounts]
}

async function mockFetchStatements(accountId: string): Promise<BankStatementRow[]> {
  cache ??= generate()
  await sleep(200)
  return cache.statements.filter((s) => s.bankAccountId === accountId)
}

async function mockFetchLines(statementId: string): Promise<BankStatementLineRow[]> {
  cache ??= generate()
  await sleep(200)
  return cache.lines.filter((l) => l.bankStatementId === statementId).map((l) => ({ ...l }))
}

async function mockFetchCashPosition(): Promise<CashPositionSummary> {
  cache ??= generate()
  await sleep(180)
  const byAccount: CashAccountBalance[] = cache.accounts
    .filter((a) => a.isActive)
    .map((a) => {
      const accountStatements = cache!.statements.filter((s) => s.bankAccountId === a.id)
      const latest = accountStatements[accountStatements.length - 1]
      const notice = latest && latest.discrepancyAmount !== 0
        ? `${a.bankName || a.accountName}: statement ${latest.statementNumber} shows a discrepancy of ${latest.discrepancyAmount.toFixed(3)} ${a.currency}.`
        : undefined
      return {
        accountId: a.id,
        accountName: a.bankName ? `${a.bankName} ${a.accountName}`.trim() : a.accountName,
        balanceBhd: latest ? latest.closingBalance : 0,
        ...(notice ? { notice } : {}),
      }
    })
  const totalBhd = byAccount.reduce((s, a) => s + a.balanceBhd, 0)
  const notices = byAccount.map((a) => a.notice).filter((n): n is string => !!n)
  return { totalBhd, byAccount, notices }
}

function candidateLabel(prefix: string, name: string, amount: number, currency = 'BHD'): string {
  return `${prefix} • ${name || 'Unknown'} • ${currency} ${amount.toFixed(3)}`
}

let candidatePoolCache: BankMatchCandidatePool | null = null

function generateCandidatePool(): BankMatchCandidatePool {
  const rand = lcg(20260715 ^ 0xba2c)
  // >200-row pool on customer invoices — proves AllocationMatchPanel's
  // amount-proximity sort + maxResults cap actually cuts the list down.
  const customerInvoices: MatchCandidate[] = []
  for (let i = 1; i <= 230; i++) {
    const name = CUSTOMER_NAMES[i % CUSTOMER_NAMES.length]!
    const amount = Math.round((5 + rand() * 20000) * 1000) / 1000
    customerInvoices.push({
      id: `cust-inv-${i}`,
      type: 'CUSTOMER_INVOICE',
      label: candidateLabel(`INV-${pad(i, 5)}`, name, amount),
      amount,
    })
  }

  const supplierInvoices: MatchCandidate[] = []
  for (let i = 1; i <= 60; i++) {
    const name = SUPPLIER_NAMES[i % SUPPLIER_NAMES.length]!
    const amount = Math.round((10 + rand() * 15000) * 1000) / 1000
    supplierInvoices.push({
      id: `sup-inv-${i}`,
      type: 'SUPPLIER_INVOICE',
      label: candidateLabel(`SINV-${pad(i, 5)}`, name, amount),
      amount,
    })
  }

  const supplierPayments: MatchCandidate[] = []
  for (let i = 1; i <= 40; i++) {
    const name = SUPPLIER_NAMES[i % SUPPLIER_NAMES.length]!
    const amount = Math.round((10 + rand() * 15000) * 1000) / 1000
    supplierPayments.push({
      id: `sup-pay-${i}`,
      type: 'SUPPLIER_PAYMENT',
      label: candidateLabel(`PAY-${pad(i, 5)}`, name, amount),
      amount,
    })
  }

  const expenses: MatchCandidate[] = []
  for (let i = 1; i <= 45; i++) {
    const name = EXPENSE_VENDORS[i % EXPENSE_VENDORS.length]!
    const amount = Math.round((5 + rand() * 3000) * 1000) / 1000
    expenses.push({
      id: `exp-${i}`,
      type: 'EXPENSE',
      label: candidateLabel(`EXP-${pad(i, 5)}`, name, amount),
      amount,
    })
  }

  const payrollPayouts: MatchCandidate[] = []
  for (let i = 1; i <= 20; i++) {
    const name = EMPLOYEE_NAMES[i % EMPLOYEE_NAMES.length]!
    const amount = Math.round((300 + rand() * 2200) * 1000) / 1000
    payrollPayouts.push({
      id: `payroll-${i}`,
      type: 'PAYROLL_PAYOUT',
      label: candidateLabel(`RUN-${pad(1 + Math.floor(i / 4), 4)}`, name, amount),
      amount,
    })
  }

  return { customerInvoices, supplierInvoices, supplierPayments, expenses, payrollPayouts }
}

async function mockFetchMatchCandidates(): Promise<BankMatchCandidatePool> {
  candidatePoolCache ??= generateCandidatePool()
  await sleep(220)
  return candidatePoolCache
}

const AUDIT_ACTIONS = ['IMPORT', 'MATCH', 'UNMATCH', 'SPLIT', 'AUTO_MATCH', 'FINALIZE']
const AUDIT_ACTORS = ['Aisha Al-Rumaihi', 'Mohammed Bucheeri', 'System (auto-match)', '']

async function mockFetchAuditTrail(statementId: string): Promise<AuditTrailEntry[]> {
  const rand = lcg(hashSeed(statementId))
  await sleep(150)
  const n = 3 + Math.floor(rand() * 6)
  const rows: AuditTrailEntry[] = []
  for (let i = 1; i <= n; i++) {
    rows.push({
      id: `audit-${statementId}-${i}`,
      timestamp: `2026-0${1 + (i % 7)}-${pad(1 + (i % 27), 2)}`,
      action: AUDIT_ACTIONS[Math.floor(rand() * AUDIT_ACTIONS.length)]!,
      actor: AUDIT_ACTORS[i % AUDIT_ACTORS.length]!,
      detail: i % 5 === 0 ? '' : `Applied to line REF-${pad(i, 5)}`,
      isReversed: i % 6 === 0,
    })
  }
  return rows
}

/* ---- mutations: mock actually performs them against the cache ---- */

async function mockPreviewImport(accountId: string): Promise<StatementImportPreview> {
  cache ??= generate()
  await sleep(300)
  previewCounter++
  const previewId = `preview-${previewCounter}`
  const rand = lcg(20260715 ^ previewCounter)
  const lineCount = 4 + Math.floor(rand() * 5)
  const previewLines: BankStatementLineRow[] = []
  for (let i = 1; i <= lineCount; i++) {
    const isCredit = rand() > 0.5
    const amount = Math.round((20 + rand() * 5000) * 100) / 100
    previewLines.push({
      id: `${previewId}-line-${i}`,
      bankStatementId: previewId,
      lineNumber: i,
      transactionDate: '2026-07-01',
      description: DESCRIPTIONS[Math.floor(rand() * DESCRIPTIONS.length)]!,
      reference: `REF-NEW-${pad(i, 3)}`,
      debit: isCredit ? 0 : amount,
      credit: isCredit ? amount : 0,
      balance: 0,
      transactionType: TX_TYPES[Math.floor(rand() * TX_TYPES.length)]!,
      isMatched: false,
      matchConfidence: 0,
      matchedInvoiceIds: '',
      extractedCustomer: '',
    })
  }
  const preview: StatementImportPreview = {
    id: previewId,
    bankAccountId: accountId,
    statementNumber: `STMT-PREVIEW-${pad(previewCounter, 4)}`,
    periodStart: '2026-07-01',
    periodEnd: '2026-07-31',
    openingBalance: 10000,
    closingBalance: 10000 + previewLines.reduce((s, l) => s + l.credit - l.debit, 0),
    lines: previewLines,
  }
  pendingPreviews.set(previewId, preview)
  return preview
}

async function mockConfirmImport(previewId: string): Promise<BankStatementRow> {
  cache ??= generate()
  const preview = pendingPreviews.get(previewId)
  if (!preview) throw new Error(`Import preview ${previewId} not found — it may already have been confirmed or discarded.`)
  await sleep(250)
  const account = cache.accounts.find((a) => a.id === preview.bankAccountId)
  const statement: BankStatementRow = {
    id: `stmt-${preview.id}`,
    bankAccountId: preview.bankAccountId,
    statementNumber: preview.statementNumber,
    periodStart: preview.periodStart,
    periodEnd: preview.periodEnd,
    openingBalance: preview.openingBalance,
    closingBalance: preview.closingBalance,
    currency: account?.currency ?? 'BHD',
    status: 'Imported',
    discrepancyAmount: 0,
    notes: '',
    division: account?.division ?? DIVISIONS[0]!,
  }
  cache.statements.push(statement)
  const newLines = preview.lines.map((l) => ({ ...l, bankStatementId: statement.id }))
  cache.lines.push(...newLines)
  pendingPreviews.delete(previewId)
  return statement
}

async function mockDiscardImportPreview(previewId: string): Promise<void> {
  await sleep(100)
  pendingPreviews.delete(previewId)
}

async function mockAutoMatch(statementId: string): Promise<AutoMatchResult> {
  cache ??= generate()
  await sleep(300)
  const statementLines = cache.lines.filter((l) => l.bankStatementId === statementId)
  let matchedNow = 0
  for (const line of statementLines) {
    if (!line.isMatched && !line.zeroCandidatePool && line.debit !== line.credit) {
      line.isMatched = true
      line.matchConfidence = 0.72
      line.matchedInvoiceIds = line.credit > 0 ? `cust-inv-${1 + Math.floor(Math.random() * 200)}` : `sup-inv-${1 + Math.floor(Math.random() * 60)}`
      matchedNow++
    }
  }
  const matchedCount = statementLines.filter((l) => l.isMatched).length
  const totalLines = statementLines.length
  return {
    matchedCount: matchedNow,
    unmatchedCount: totalLines - matchedCount,
    totalLines,
    matchedPercent: totalLines > 0 ? Math.round((matchedCount / totalLines) * 100) : 0,
  }
}

async function mockManualMatch(lineId: string, type: string, candidateId: string, _user: string): Promise<void> {
  cache ??= generate()
  const line = cache.lines.find((l) => l.id === lineId)
  await sleep(180)
  if (!line) throw new Error(`Statement line ${lineId} not found`)
  line.isMatched = true
  line.matchConfidence = 1
  line.matchedInvoiceIds = `${type}:${candidateId}`
}

async function mockCreateSplitAllocation(lineId: string, allocations: SplitAllocationInput[], _user: string): Promise<void> {
  cache ??= generate()
  const line = cache.lines.find((l) => l.id === lineId)
  await sleep(220)
  if (!line) throw new Error(`Statement line ${lineId} not found`)
  line.isMatched = true
  line.matchConfidence = 1
  line.matchedInvoiceIds = allocations.map((a) => `${a.allocationType}:${a.entityId}`).join(',')
}

async function mockUnmatchLine(lineId: string, _user: string, _reason: string): Promise<void> {
  cache ??= generate()
  const line = cache.lines.find((l) => l.id === lineId)
  await sleep(150)
  if (!line) throw new Error(`Statement line ${lineId} not found`)
  line.isMatched = false
  line.matchConfidence = 0
  line.matchedInvoiceIds = ''
}

async function mockFinalizeReconciliation(statementId: string, _user: string): Promise<void> {
  cache ??= generate()
  const statement = cache.statements.find((s) => s.id === statementId)
  await sleep(220)
  if (!statement) throw new Error(`Statement ${statementId} not found`)
  statement.status = 'Reconciled'
}

async function mockDeleteBankStatement(statementId: string): Promise<void> {
  cache ??= generate()
  await sleep(180)
  cache.statements = cache.statements.filter((s) => s.id !== statementId)
  cache.lines = cache.lines.filter((l) => l.bankStatementId !== statementId)
}

async function mockUpdateBankStatement(statementId: string, draft: BankStatementDraft): Promise<void> {
  cache ??= generate()
  const statement = cache.statements.find((s) => s.id === statementId)
  await sleep(180)
  if (!statement) throw new Error(`Statement ${statementId} not found`)
  statement.openingBalance = draft.openingBalance
  statement.closingBalance = draft.closingBalance
  statement.periodStart = draft.periodStart
  statement.periodEnd = draft.periodEnd
  statement.status = draft.status
  statement.notes = draft.notes
}

async function mockCreateBankStatementLine(statementId: string, draft: BankStatementLineDraft): Promise<void> {
  cache ??= generate()
  await sleep(150)
  const statementLines = cache.lines.filter((l) => l.bankStatementId === statementId)
  cache.lines.push({
    id: `line-new-${Date.now()}-${statementLines.length + 1}`,
    bankStatementId: statementId,
    lineNumber: statementLines.length + 1,
    transactionDate: draft.transactionDate,
    description: draft.description,
    reference: draft.reference,
    debit: draft.debit,
    credit: draft.credit,
    balance: 0,
    transactionType: 'UNKNOWN',
    isMatched: false,
    matchConfidence: 0,
    matchedInvoiceIds: '',
    extractedCustomer: '',
  })
}

async function mockUpdateBankStatementLine(lineId: string, draft: BankStatementLineDraft): Promise<void> {
  cache ??= generate()
  const line = cache.lines.find((l) => l.id === lineId)
  await sleep(150)
  if (!line) throw new Error(`Statement line ${lineId} not found`)
  line.transactionDate = draft.transactionDate
  line.description = draft.description
  line.reference = draft.reference
  line.debit = draft.debit
  line.credit = draft.credit
  // Preserved surprise: editing a matched line clears its match so it
  // re-enters the review queue (old screen's OCR-fix flow).
  line.isMatched = false
  line.matchConfidence = 0
  line.matchedInvoiceIds = ''
}

async function mockDeleteBankStatementLine(lineId: string): Promise<void> {
  cache ??= generate()
  await sleep(120)
  cache.lines = cache.lines.filter((l) => l.id !== lineId)
}

/* ---------------------------------------------------------------------- */
/* real: FETCH bindings wired; every MUTATION is an honest INTEG-gap throw */
/* ---------------------------------------------------------------------- */

function mapBankAccount(r: Record<string, unknown>): BankAccountOption {
  return {
    id: str(r.id),
    bankName: str(r.bank_name),
    accountName: str(r.account_name),
    accountNumber: str(r.account_number),
    currency: str(r.currency) || 'BHD',
    isActive: Boolean(r.is_active),
    division: str(r.division),
  }
}

function mapBankStatement(r: Record<string, unknown>): BankStatementRow {
  return {
    id: str(r.id),
    bankAccountId: str(r.bank_account_id),
    statementNumber: str(r.statement_number),
    periodStart: goDate(r.period_start),
    periodEnd: goDate(r.period_end),
    openingBalance: num(r.opening_balance),
    closingBalance: num(r.closing_balance),
    currency: str(r.currency) || 'BHD',
    status: str(r.status) || 'Imported',
    discrepancyAmount: num(r.discrepancy_amount),
    notes: str(r.notes),
    division: str(r.division),
  }
}

function mapBankStatementLine(r: Record<string, unknown>): BankStatementLineRow {
  return {
    id: str(r.id),
    bankStatementId: str(r.bank_statement_id),
    lineNumber: num(r.line_number),
    transactionDate: goDate(r.transaction_date),
    description: str(r.description),
    reference: str(r.reference),
    debit: num(r.debit),
    credit: num(r.credit),
    balance: num(r.balance),
    transactionType: str(r.transaction_type) || 'UNKNOWN',
    isMatched: Boolean(r.is_matched),
    matchConfidence: num(r.match_confidence),
    matchedInvoiceIds: str(r.matched_invoice_ids),
    extractedCustomer: str(r.extracted_customer),
  }
}

async function realFetchAccounts(): Promise<BankAccountOption[]> {
  const rows = await GetActiveBankAccounts()
  return (rows ?? []).map((r) => mapBankAccount(r as unknown as Record<string, unknown>))
}

async function realFetchStatements(accountId: string): Promise<BankStatementRow[]> {
  const rows = await GetBankStatements(accountId)
  return (rows ?? []).map((r) => mapBankStatement(r as unknown as Record<string, unknown>))
}

async function realFetchLines(statementId: string): Promise<BankStatementLineRow[]> {
  const rows = await GetBankStatementLines(statementId)
  return (rows ?? []).map((r) => mapBankStatementLine(r as unknown as Record<string, unknown>))
}

async function realFetchCashPosition(): Promise<CashPositionSummary> {
  const raw = (await GetCashPosition()) as unknown as Record<string, unknown>
  const rawAccounts = Array.isArray(raw?.accounts) ? (raw.accounts as Record<string, unknown>[]) : []
  const byAccount: CashAccountBalance[] = rawAccounts.map((a) => {
    const notice = a.notice ? str(a.notice) : undefined
    return {
      accountId: str(a.id ?? a.account_id),
      accountName: str(a.account_name ?? a.bank_name ?? a.name),
      // Defensive probe — GetCashPosition returns an untyped map; field
      // casing/shape isn't guaranteed (build brief).
      balanceBhd: num(a.current_balance_bhd ?? a.CurrentBalanceBHD ?? a.current_balance ?? a.CurrentBalance),
      ...(notice ? { notice } : {}),
    }
  })
  const fallbackTotal = num(raw?.total_bhd ?? raw?.TotalBHD ?? raw?.total_balance_bhd ?? raw?.current_balance_bhd)
  const totalBhd = byAccount.length > 0 ? byAccount.reduce((s, a) => s + a.balanceBhd, 0) : fallbackTotal
  const notices = byAccount.map((a) => a.notice).filter((n): n is string => !!n)
  return { totalBhd, byAccount, notices }
}

function invoiceCandidate(r: Record<string, unknown>): MatchCandidate {
  const amount = num(r.outstanding_bhd) || num(r.grand_total_bhd)
  return {
    id: str(r.id),
    type: 'CUSTOMER_INVOICE',
    label: candidateLabel(str(r.invoice_number) || 'Invoice', str(r.customer_name), amount),
    amount,
  }
}
function supplierInvoiceCandidate(r: Record<string, unknown>): MatchCandidate {
  return {
    id: str(r.id),
    type: 'SUPPLIER_INVOICE',
    label: candidateLabel(str(r.invoice_number) || 'Invoice', str(r.supplier_name), num(r.total_bhd)),
    amount: num(r.total_bhd),
  }
}
function supplierPaymentCandidate(r: Record<string, unknown>): MatchCandidate {
  const label = str(r.reference) || str(r.invoice_number) || 'Payment'
  return {
    id: str(r.id),
    type: 'SUPPLIER_PAYMENT',
    label: candidateLabel(label, str(r.supplier_name), num(r.amount_bhd)),
    amount: num(r.amount_bhd),
  }
}
function expenseCandidate(r: Record<string, unknown>): MatchCandidate {
  const amount = num(r.total_amount) || num(r.amount) + num(r.vat_amount)
  const name = str(r.vendor_name) || str(r.category_name) || str(r.cost_center)
  return {
    id: str(r.id),
    type: 'EXPENSE',
    label: candidateLabel(str(r.entry_number) || 'Expense', name, amount),
    amount,
  }
}
function payrollCandidate(r: Record<string, unknown>): MatchCandidate {
  const ref = str(r.payment_reference)
  const base = candidateLabel(str(r.run_number) || 'Payroll', str(r.employee_name), num(r.amount))
  return {
    id: str(r.id),
    type: 'PAYROLL_PAYOUT',
    label: ref ? `${base} • ${ref}` : base,
    amount: num(r.amount),
  }
}

async function realFetchMatchCandidates(): Promise<BankMatchCandidatePool> {
  const [customerInvoicesRaw, supplierInvoicesRaw, supplierPaymentsRaw, expensesRaw, payrollPayoutsRaw] = await Promise.all([
    ListCustomerInvoices(1000, 0),
    GetSupplierInvoices(),
    GetAllSupplierPayments(),
    ListExpenseEntries('', false),
    ListUnreconciledPayrollPayouts(),
  ])

  const customerInvoices = (customerInvoicesRaw ?? [])
    .map((r) => r as unknown as Record<string, unknown>)
    .filter((r) => num(r.outstanding_bhd) > 0 && !['cancelled', 'void', 'draft'].includes(str(r.status).toLowerCase()))
    .map(invoiceCandidate)

  const supplierInvoices = (supplierInvoicesRaw ?? [])
    .map((r) => r as unknown as Record<string, unknown>)
    .filter(
      (r) =>
        str(r.payment_status).toLowerCase() !== 'paid' &&
        !['cancelled', 'void', 'rejected'].includes(str(r.status).toLowerCase()),
    )
    .map(supplierInvoiceCandidate)

  const supplierPayments = (supplierPaymentsRaw ?? [])
    .map((r) => r as unknown as Record<string, unknown>)
    .map(supplierPaymentCandidate)

  const expenses = (expensesRaw ?? [])
    .map((r) => r as unknown as Record<string, unknown>)
    .filter((r) => {
      const status = str(r.status).toLowerCase()
      const paymentStatus = str(r.payment_status).toLowerCase()
      return (
        paymentStatus !== 'paid' &&
        !['cancelled', 'canceled', 'void', 'rejected'].includes(status) &&
        (num(r.total_amount) || num(r.amount) + num(r.vat_amount)) > 0
      )
    })
    .map(expenseCandidate)

  const payrollPayouts = (payrollPayoutsRaw ?? []).map((r) => payrollCandidate(r as unknown as Record<string, unknown>))

  return { customerInvoices, supplierInvoices, supplierPayments, expenses, payrollPayouts }
}

function mapAuditLog(r: Record<string, unknown>): AuditTrailEntry {
  return {
    id: str(r.id),
    timestamp: goDate(r.performed_at) || goDate(r.created_at),
    action: str(r.action) || 'UNKNOWN_ACTION',
    actor: str(r.performed_by),
    detail: str(r.action_detail) || str(r.reason),
    isReversed: Boolean(r.is_reversed),
  }
}

async function realFetchAuditTrail(statementId: string): Promise<AuditTrailEntry[]> {
  const rows = await GetAuditTrail(statementId)
  return (rows ?? []).map((r) => mapAuditLog(r as unknown as Record<string, unknown>))
}

async function realPreviewImport(_accountId: string): Promise<StatementImportPreview> {
  throw new Error('INTEG gap: PreviewBankStatementImportWithDialog — wires at K5')
}
async function realConfirmImport(_previewId: string): Promise<BankStatementRow> {
  throw new Error('INTEG gap: ConfirmBankStatementImport — wires at K5')
}
async function realDiscardImportPreview(_previewId: string): Promise<void> {
  throw new Error('INTEG gap: DiscardBankStatementImportPreview — wires at K5')
}
async function realAutoMatch(_statementId: string): Promise<AutoMatchResult> {
  throw new Error('INTEG gap: AutoMatchBankLines — wires at K5')
}
async function realManualMatch(_lineId: string, _type: string, _candidateId: string, _user: string): Promise<void> {
  throw new Error('INTEG gap: ManualMatchLine — wires at K5')
}
async function realCreateSplitAllocation(_lineId: string, _allocations: SplitAllocationInput[], _user: string): Promise<void> {
  throw new Error('INTEG gap: CreateSplitAllocation — wires at K5')
}
async function realUnmatchLine(_lineId: string, _user: string, _reason: string): Promise<void> {
  throw new Error('INTEG gap: UnmatchLine — wires at K5')
}
async function realFinalizeReconciliation(_statementId: string, _user: string): Promise<void> {
  throw new Error('INTEG gap: FinalizeReconciliation — wires at K5 (HOT-ZONE: posting-adjacent)')
}
async function realDeleteBankStatement(_statementId: string): Promise<void> {
  throw new Error('INTEG gap: DeleteBankStatement — wires at K5 (HOT-ZONE: posting-adjacent)')
}
async function realUpdateBankStatement(_statementId: string, _draft: BankStatementDraft): Promise<void> {
  throw new Error('INTEG gap: UpdateBankStatement — wires at K5')
}
async function realCreateBankStatementLine(_statementId: string, _draft: BankStatementLineDraft): Promise<void> {
  throw new Error('INTEG gap: CreateBankStatementLine — wires at K5')
}
async function realUpdateBankStatementLine(_lineId: string, _draft: BankStatementLineDraft): Promise<void> {
  throw new Error('INTEG gap: UpdateBankStatementLine — wires at K5')
}
async function realDeleteBankStatementLine(_lineId: string): Promise<void> {
  throw new Error('INTEG gap: DeleteBankStatementLine — wires at K5')
}

/* ---------------------------------------------------------------------- */
/* public switched API (viewmodel imports THESE)                           */
/* ---------------------------------------------------------------------- */

export const fetchBankAccounts = (): Promise<BankAccountOption[]> => pick(realFetchAccounts, mockFetchAccounts)()
export const fetchBankStatements = (accountId: string): Promise<BankStatementRow[]> =>
  pick(realFetchStatements, mockFetchStatements)(accountId)
export const fetchBankStatementLines = (statementId: string): Promise<BankStatementLineRow[]> =>
  pick(realFetchLines, mockFetchLines)(statementId)
export const fetchCashPosition = (): Promise<CashPositionSummary> => pick(realFetchCashPosition, mockFetchCashPosition)()
export const fetchMatchCandidates = (): Promise<BankMatchCandidatePool> =>
  pick(realFetchMatchCandidates, mockFetchMatchCandidates)()
export const fetchAuditTrail = (statementId: string): Promise<AuditTrailEntry[]> =>
  pick(realFetchAuditTrail, mockFetchAuditTrail)(statementId)

export const previewBankStatementImport = (accountId: string): Promise<StatementImportPreview> =>
  pick(realPreviewImport, mockPreviewImport)(accountId)
export const confirmBankStatementImport = (previewId: string): Promise<BankStatementRow> =>
  pick(realConfirmImport, mockConfirmImport)(previewId)
export const discardBankStatementImportPreview = (previewId: string): Promise<void> =>
  pick(realDiscardImportPreview, mockDiscardImportPreview)(previewId)
export const autoMatchBankLines = (statementId: string): Promise<AutoMatchResult> =>
  pick(realAutoMatch, mockAutoMatch)(statementId)
export const manualMatchLine = (lineId: string, type: string, candidateId: string, user: string): Promise<void> =>
  pick(realManualMatch, mockManualMatch)(lineId, type, candidateId, user)
export const createSplitAllocation = (lineId: string, allocations: SplitAllocationInput[], user: string): Promise<void> =>
  pick(realCreateSplitAllocation, mockCreateSplitAllocation)(lineId, allocations, user)
export const unmatchLine = (lineId: string, user: string, reason: string): Promise<void> =>
  pick(realUnmatchLine, mockUnmatchLine)(lineId, user, reason)
export const finalizeReconciliation = (statementId: string, user: string): Promise<void> =>
  pick(realFinalizeReconciliation, mockFinalizeReconciliation)(statementId, user)
export const deleteBankStatement = (statementId: string): Promise<void> =>
  pick(realDeleteBankStatement, mockDeleteBankStatement)(statementId)
export const updateBankStatement = (statementId: string, draft: BankStatementDraft): Promise<void> =>
  pick(realUpdateBankStatement, mockUpdateBankStatement)(statementId, draft)
export const createBankStatementLine = (statementId: string, draft: BankStatementLineDraft): Promise<void> =>
  pick(realCreateBankStatementLine, mockCreateBankStatementLine)(statementId, draft)
export const updateBankStatementLine = (lineId: string, draft: BankStatementLineDraft): Promise<void> =>
  pick(realUpdateBankStatementLine, mockUpdateBankStatementLine)(lineId, draft)
export const deleteBankStatementLine = (lineId: string): Promise<void> =>
  pick(realDeleteBankStatementLine, mockDeleteBankStatementLine)(lineId)

export const CANDIDATE_TYPE_OPTIONS: { value: CandidateType; label: string }[] = [
  { value: 'CUSTOMER_INVOICE', label: 'Customer Invoice' },
  { value: 'SUPPLIER_INVOICE', label: 'Supplier Invoice' },
  { value: 'SUPPLIER_PAYMENT', label: 'Supplier Payment' },
  { value: 'EXPENSE', label: 'Expense' },
  { value: 'PAYROLL_PAYOUT', label: 'Payroll Payout' },
]

export const SINGLE_SELECT_CANDIDATE_TYPES: string[] = ['SUPPLIER_PAYMENT', 'PAYROLL_PAYOUT']
