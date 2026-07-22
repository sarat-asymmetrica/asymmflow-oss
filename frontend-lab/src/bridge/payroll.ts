/* Payroll bridge module — self-contained: types + mock + real + switch.
 * PII hot-zone (K5 tranche): salary/allowance/deduction AMOUNTS and employee
 * NAMES are the sensitive surface here (there is no employee IBAN in this
 * domain — the bank picker on Mark Paid is the COMPANY's disbursing account,
 * not an employee bank detail). Wails IPC only — real adapters bind
 * `$wails/go/main/FinanceService` (+ `GetActiveBankAccounts` on `App`),
 * confirmed against `frontend/wailsjs/go/main/FinanceService.d.ts` and the
 * `payroll` namespace in `frontend/wailsjs/go/models.ts`. FETCH bindings are
 * real (list/get are single-call, non-aggregating); every mutation is a
 * financial+PII hot-zone and throws an honest INTEG-gap naming the exact
 * real binding — see Payroll.parity.md for the full ledger. Synthetic-only
 * mock data (SYNTHETIC_IDENTITY.md) — invented Gulf names, no real people.
 */
import { pick } from './runtime'
import { goDate, goTime, num, str } from './map'
import type { payroll } from '$wails/go/models'
/* Only the FETCH bindings are actually invoked below — mutations are
 * INTEG-gap throws that NAME the real binding without importing it (same
 * convention as bridge/expenses.ts: importing a binding this file never
 * calls would just be an unused-import lint trap). */
import {
  ApprovePayrollRun,
  CreatePayrollPeriod,
  GeneratePayrollRun,
  GeneratePayslipPDF,
  GetActiveBankAccounts,
  GetPayrollRun,
  ListEmployeeCompensationProfiles,
  ListPayrollPayouts,
  ListPayrollPeriods,
  ListPayrollRuns,
  MarkPayrollRunPaid,
  PostPayrollRun,
  UpsertEmployeeCompensationProfile,
} from '$wails/go/main/FinanceService'
// Employee master list for the Compensation form's picker — a DIFFERENT
// (collaboration) domain than payroll. Read-only; verified service seam.
import { ListEmployeeProfiles } from '$wails/go/main/App'

/* ---- types (camelCase; mirrors payroll.* / finance.CompanyBankAccount) ---- */

export interface PayrollComponent {
  id: string
  payrollRunItemId: string
  componentType: string // 'earning' | 'deduction' | 'employer_cost'
  code: string
  name: string
  amount: number
}

export interface PayrollRunItem {
  id: string
  payrollRunId: string
  employeeId: string
  /** Live employee name; empty when the employee record has no current name
   * on file — the row falls back to `employeeNameSnapshot` (the name at the
   * time the run was generated), never a raw blank. */
  employeeName: string
  employeeNameSnapshot: string
  jobTitleSnapshot: string
  baseSalary: number
  allowancesTotal: number
  deductionsTotal: number
  employerCostTotal: number
  grossPay: number
  netPay: number
  status: string
  payoutId: string
  payoutStatus: string
  payoutPaidAt: string
  components: PayrollComponent[]
}

export interface PayrollRun {
  id: string
  runNumber: string
  payrollPeriodId: string
  division: string
  periodName: string
  status: string // draft | approved | posted | paid (+ adversarial unknown_status)
  generatedAt: string
  approvedAt: string
  postedAt: string
  paidAt: string
  paymentReference: string
  bankAccountId: string
  totalEmployees: number
  grossTotal: number
  deductionsTotal: number
  netTotal: number
  employerCostTotal: number
  currency: string
  notes: string
  items: PayrollRunItem[]
}

/** List-shape run row (no `items` — `ListPayrollRuns` returns summaries;
 * only `GetPayrollRun` hydrates the full item list). */
export type PayrollRunSummary = Omit<PayrollRun, 'items'>

export interface PayrollPeriod {
  id: string
  name: string
  division: string
  periodStart: string
  periodEnd: string
  paymentDate: string
  status: string // open | closed
  notes: string
}

export interface PayrollPeriodDraft {
  name: string
  division: string
  periodStart: string
  periodEnd: string
  paymentDate: string
  notes: string
}

export interface PayrollPayout {
  id: string
  payrollRunId: string
  employeeId: string
  division: string
  employeeName: string
  runNumber: string
  scheduledAt: string
  paidAt: string
  amount: number
  currency: string
  status: string // scheduled | paid
  paymentReference: string
}

export interface CompensationProfile {
  id: string
  employeeId: string
  employeeName: string
  jobTitle: string
  division: string
  payFrequency: string
  currency: string
  baseSalary: number
  housingAllowance: number
  transportAllowance: number
  otherAllowance: number
  standardDeduction: number
  taxDeduction: number
  employerCost: number
  effectiveFrom: string
  effectiveTo: string
  isActive: boolean
  /** Derived, not a backend field (matches bank-accounts.ts's `status`
   * convention) — feeds DataTable's StatusSpec badge column. */
  status: 'Active' | 'Inactive'
  notes: string
}

export interface CompensationProfileDraft {
  id?: string
  employeeId: string
  division: string
  payFrequency: string
  baseSalary: number
  housingAllowance: number
  transportAllowance: number
  otherAllowance: number
  standardDeduction: number
  taxDeduction: number
  employerCost: number
  effectiveFrom: string
  effectiveTo: string
  isActive: boolean
  notes: string
}

export interface PayrollBankAccount {
  id: string
  division: string
  bankName: string
  accountName: string
  accountNumber: string
  currency: string
}

/** Lab-only picker source for the Compensation form's employee field — see
 * `fetchPayrollEmployeeOptions` below for why this isn't a straight `pick()`
 * of a real binding. */
export interface PayrollEmployeeOption {
  id: string
  name: string
  jobTitle: string
  division: string
}

/* ---- mock: adversarial + deterministic (see bridge/mock.ts pattern) ---- */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
const pad = (n: number, w: number): string => String(n).padStart(w, '0')
const todayIso = (): string => new Date().toISOString().slice(0, 10)

// Third entry is a deliberate legacy-cased mismatch (recon convention: a
// pre-normalization division string) — the client-side filter below does
// plain equality, so this row never matches either canonical chip. See
// Payroll.parity.md "Division scoping is client-side" note.
const DIVISIONS = ['Acme Instrumentation', 'Beacon Controls', 'beacon controls (legacy)']

const MONSTER_EMPLOYEE_NAME =
  'MOHAMMEDABDULRAHMANALSHAMSISENIORFIELDINSTRUMENTATIONANDCALIBRATIONTECHNICIANFORMERLYGULFTECHNICALSERVICESDEPARTMENTOFPROCESSCONTROLENGINEERING'.padEnd(
    200,
    'X',
  )

const EMPLOYEE_NAMES = [
  'Ahmed Al-Khalifa',
  'Fatima Mohammed Al-Sayed',
  'Yusuf Kanoo',
  'Al', // 2-char
  '', // empty — exercises the run item's employeeNameSnapshot fallback
  'محمد عبدالله الأنصاري الكوري', // RTL
  MONSTER_EMPLOYEE_NAME, // unbroken 200-char token
  'Layla Haidar',
  'Karim Nasser',
  'Bashir Al-Ansari',
  'X',
  'Noor Al-Zayani',
  'Hessa Al-Dosari',
  'Omar Fakhro',
]

const JOB_TITLES = ['Field Engineer', 'Calibration Technician', 'Finance Officer', 'HR Coordinator', 'Warehouse Supervisor', 'Sales Executive', '']
const PAY_FREQUENCIES = ['monthly', 'biweekly', 'weekly']
const MONTH_NAMES = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
const BANKS = ['Ahli United Bank', 'National Bank of Bahrain', 'BBK']

interface Dataset {
  employees: PayrollEmployeeOption[]
  profiles: CompensationProfile[]
  periods: PayrollPeriod[]
  runs: PayrollRun[]
  payouts: PayrollPayout[]
  bankAccounts: PayrollBankAccount[]
}

let cache: Dataset | null = null

function generate(): Dataset {
  const rand = lcg(20260714 ^ 0x9a7 /* payroll seed tweak, matches sibling bridges' xor convention */)

  // ---- employees: 14 total, only 12 get a compensation profile — the last
  // 2 are unprofiled "new hires" so the Compensation form's employee picker
  // has real choices beyond employees who already have a profile on file.
  const EMP_COUNT = 14
  const employees: PayrollEmployeeOption[] = []
  for (let i = 1; i <= EMP_COUNT; i++) {
    employees.push({
      id: `emp-${i}`,
      name: EMPLOYEE_NAMES[(i - 1) % EMPLOYEE_NAMES.length]!,
      jobTitle: JOB_TITLES[(i - 1) % JOB_TITLES.length]!,
      division: DIVISIONS[(i - 1) % 2]!, // never seeds an employee into the legacy division
    })
  }

  // ---- compensation profiles: one per employee for emp-1..emp-12.
  const PROFILE_COUNT = 12
  const profiles: CompensationProfile[] = []
  for (let i = 1; i <= PROFILE_COUNT; i++) {
    const emp = employees[i - 1]!
    let baseSalary = Math.round((300 + rand() * 2700) * 1000) / 1000
    let housingAllowance = Math.round(rand() * 500 * 1000) / 1000
    let transportAllowance = Math.round(rand() * 150 * 1000) / 1000
    const otherAllowance = Math.round(rand() * 100 * 1000) / 1000
    const standardDeduction = Math.round(rand() * 80 * 1000) / 1000
    const taxDeduction = Math.round(rand() * 40 * 1000) / 1000
    let employerCost = Math.round(rand() * 300 * 1000) / 1000

    // Monster: base salary 0 but nonzero allowances (all-allowance comp package).
    if (i === 3) {
      baseSalary = 0
      housingAllowance = 450
      transportAllowance = 120
    }
    // Monster: negative base salary (data-entry error / clawback row).
    if (i === 5) baseSalary = -750.5
    // Monster: huge base salary.
    if (i === 7) baseSalary = 999999999.999
    // Monster: tiny base salary.
    if (i === 9) baseSalary = 0.001

    let effectiveFrom = `2025-0${1 + (i % 9)}-01`
    let effectiveTo = ''
    // Monster: effective_to before effective_from (contradiction).
    if (i === 11) {
      effectiveFrom = '2026-06-01'
      effectiveTo = '2026-01-01'
    }

    const isActive = i % 6 !== 0
    profiles.push({
      id: `profile-${i}`,
      employeeId: emp.id,
      employeeName: emp.name,
      jobTitle: emp.jobTitle,
      division: emp.division,
      payFrequency: PAY_FREQUENCIES[i % PAY_FREQUENCIES.length]!,
      currency: i % 11 === 0 ? 'USD' : 'BHD',
      baseSalary,
      housingAllowance,
      transportAllowance,
      otherAllowance,
      standardDeduction,
      taxDeduction,
      employerCost,
      effectiveFrom,
      effectiveTo,
      isActive,
      status: isActive ? 'Active' : 'Inactive',
      notes: i % 4 === 0 ? '' : 'Synthetic compensation profile.',
    })
  }

  // ---- periods: 9, spread across divisions + open/closed.
  const PERIOD_COUNT = 9
  const periods: PayrollPeriod[] = []
  for (let i = 1; i <= PERIOD_COUNT; i++) {
    const monthIdx = (i + 3) % 12
    const year = 2025 + Math.floor((i + 3) / 12)
    const division = DIVISIONS[i % 2]!
    periods.push({
      id: `per-${i}`,
      name: `${MONTH_NAMES[monthIdx]} ${year} Payroll`,
      division,
      periodStart: `${year}-${pad(monthIdx + 1, 2)}-01`,
      periodEnd: `${year}-${pad(monthIdx + 1, 2)}-28`,
      paymentDate: `${year}-${pad(monthIdx + 1, 2)}-27`,
      status: i % 4 === 0 ? 'closed' : 'open',
      notes: i % 5 === 0 ? 'Includes mid-cycle joiner proration.' : '',
    })
  }

  // ---- runs (+ items), 12 total, referencing periods 1:1 for the first 9,
  // extra 3 stacked onto period 1 (multiple runs per period is legal — a
  // regenerate-after-correction scenario).
  const RUN_COUNT = 12
  const runs: PayrollRun[] = []
  const STATUS_CYCLE = ['draft', 'approved', 'posted', 'paid']
  for (let r = 1; r <= RUN_COUNT; r++) {
    const period = periods[(r - 1) % periods.length]!
    let status = STATUS_CYCLE[(r - 1) % STATUS_CYCLE.length]!
    // Monster: exactly one run in an unrecognized status — Stepper's
    // currentIndex is -1, every step renders pending, doesn't crash.
    if (r === 12) status = 'unknown_status'

    let approvedAt = ''
    let postedAt = ''
    let paidAt = ''
    if (['approved', 'posted', 'paid'].includes(status)) approvedAt = period.periodEnd
    if (['posted', 'paid'].includes(status)) postedAt = period.periodEnd
    if (status === 'paid') paidAt = period.paymentDate
    // Monster: posted run that ALSO has paid_at set — status/timestamp
    // contradiction (post-before-pay isn't strictly enforced server-side;
    // see Payroll.parity.md "preserve + flag").
    if (r === 4) {
      status = 'posted'
      postedAt = period.periodEnd
      paidAt = period.paymentDate
    }

    // Item generation: normal runs get 3-9 items; run 6 has ZERO items
    // (generated-but-empty edge case); run 9 has 240 items (bulk-division
    // monster).
    const items: PayrollRunItem[] = []
    let itemCount = 3 + (r % 7)
    if (r === 6) itemCount = 0
    if (r === 9) itemCount = 240

    let grossTotal = 0
    let deductionsTotal = 0
    let netTotal = 0
    let employerCostTotal = 0

    for (let j = 1; j <= itemCount; j++) {
      const emp = employees[(r * 7 + j) % employees.length]!
      const base = Math.round((250 + rand() * 2200) * 1000) / 1000
      const allowances = Math.round(rand() * 400 * 1000) / 1000
      const deductions = Math.round(rand() * 150 * 1000) / 1000
      const employerCostItem = Math.round(rand() * 200 * 1000) / 1000
      const gross = Math.round((base + allowances) * 1000) / 1000
      const net = Math.round((gross - deductions) * 1000) / 1000

      // Monster (run 1, item 1 only): live employee name blank — the row
      // must fall back to its snapshot name, never render a raw blank.
      const blankLiveName = r === 1 && j === 1
      const itemStatus = status === 'paid' ? 'paid' : status === 'posted' ? 'processing' : 'pending'

      items.push({
        id: `item-${r}-${j}`,
        payrollRunId: `run-${r}`,
        employeeId: emp.id,
        employeeName: blankLiveName ? '' : emp.name,
        employeeNameSnapshot: blankLiveName ? 'Ahmed Al-Khalifa (snapshot at run time)' : emp.name,
        jobTitleSnapshot: emp.jobTitle,
        baseSalary: base,
        allowancesTotal: allowances,
        deductionsTotal: deductions,
        employerCostTotal: employerCostItem,
        grossPay: gross,
        netPay: net,
        status: itemStatus,
        payoutId: status === 'paid' ? `payout-${r}-${j}` : '',
        payoutStatus: status === 'paid' ? 'paid' : 'scheduled',
        payoutPaidAt: status === 'paid' ? paidAt : '',
        components: [
          { id: `cmp-${r}-${j}-1`, payrollRunItemId: `item-${r}-${j}`, componentType: 'earning', code: 'BASE', name: 'Base Salary', amount: base },
          { id: `cmp-${r}-${j}-2`, payrollRunItemId: `item-${r}-${j}`, componentType: 'deduction', code: 'STD', name: 'Standard Deduction', amount: deductions },
          { id: `cmp-${r}-${j}-3`, payrollRunItemId: `item-${r}-${j}`, componentType: 'employer_cost', code: 'EMP', name: 'Employer Cost', amount: employerCostItem },
        ],
      })
      grossTotal += gross
      deductionsTotal += deductions
      netTotal += net
      employerCostTotal += employerCostItem
    }

    runs.push({
      id: `run-${r}`,
      runNumber: `PR-${period.periodStart.slice(0, 4)}-${pad(r, 3)}`,
      payrollPeriodId: period.id,
      division: period.division,
      periodName: period.name,
      status,
      generatedAt: period.periodStart,
      approvedAt,
      postedAt,
      paidAt,
      paymentReference: status === 'paid' ? `PMT-PAY-${pad(r, 4)}` : '',
      bankAccountId: status === 'paid' ? 'pbank-1' : '',
      totalEmployees: itemCount,
      grossTotal: Math.round(grossTotal * 1000) / 1000,
      deductionsTotal: Math.round(deductionsTotal * 1000) / 1000,
      netTotal: Math.round(netTotal * 1000) / 1000,
      employerCostTotal: Math.round(employerCostTotal * 1000) / 1000,
      currency: period.division === 'Beacon Controls' ? 'USD' : 'BHD',
      notes: r === 2 ? 'Manual correction applied for a mid-cycle joiner.' : '',
      items,
    })
  }

  // ---- payouts: one per paid/processing item across all runs, plus two
  // deliberate monsters appended at the end.
  const payouts: PayrollPayout[] = []
  for (const run of runs) {
    for (const item of run.items) {
      if (item.status !== 'paid') continue
      payouts.push({
        id: `payout-${run.id}-${item.id}`,
        payrollRunId: run.id,
        employeeId: item.employeeId,
        division: run.division,
        employeeName: item.employeeName || item.employeeNameSnapshot,
        runNumber: run.runNumber,
        scheduledAt: run.postedAt || run.generatedAt,
        paidAt: run.paidAt,
        amount: item.netPay,
        currency: run.currency,
        status: 'paid',
        paymentReference: run.paymentReference,
      })
    }
  }
  // Monster: scheduled payout that ALSO has paid_at set (contradiction).
  payouts.push({
    id: 'payout-monster-scheduled-paid',
    payrollRunId: 'run-1',
    employeeId: 'emp-2',
    division: DIVISIONS[0]!,
    employeeName: 'Fatima Mohammed Al-Sayed',
    runNumber: runs[0]!.runNumber,
    scheduledAt: '2026-01-27',
    paidAt: '2026-01-30', // set despite status still reading 'scheduled'
    amount: 812.5,
    currency: 'BHD',
    status: 'scheduled',
    paymentReference: '',
  })
  // Monster: dangling FK — payrollRunId not present in `runs` at all. Any
  // row-click handler MUST guard against this (selectRun on a missing id).
  payouts.push({
    id: 'payout-monster-dangling-fk',
    payrollRunId: 'run-999-does-not-exist',
    employeeId: 'emp-6',
    division: DIVISIONS[1]!,
    employeeName: 'محمد عبدالله الأنصاري الكوري',
    runNumber: 'PR-ORPHANED',
    scheduledAt: '2026-02-27',
    paidAt: '',
    amount: 640,
    currency: 'BHD',
    status: 'scheduled',
    paymentReference: '',
  })

  const bankAccounts: PayrollBankAccount[] = [
    { id: 'pbank-1', division: DIVISIONS[0]!, bankName: BANKS[0]!, accountName: 'Payroll Disbursement — Acme', accountNumber: '010-778899-001', currency: 'BHD' },
    { id: 'pbank-2', division: DIVISIONS[1]!, bankName: BANKS[1]!, accountName: 'Payroll Disbursement — Beacon', accountNumber: '021-334455-002', currency: 'USD' },
    { id: 'pbank-3', division: DIVISIONS[0]!, bankName: BANKS[2]!, accountName: 'Payroll Reserve', accountNumber: '', currency: 'BHD' },
  ]

  return { employees, profiles, periods, runs, payouts, bankAccounts }
}

async function mockFetchProfiles(activeOnly: boolean): Promise<CompensationProfile[]> {
  cache ??= generate()
  await sleep(220)
  return cache.profiles.filter((p) => !activeOnly || p.isActive).map((p) => ({ ...p }))
}

async function mockFetchPeriods(openOnly: boolean): Promise<PayrollPeriod[]> {
  cache ??= generate()
  await sleep(180)
  return cache.periods.filter((p) => !openOnly || p.status === 'open').map((p) => ({ ...p }))
}

async function mockFetchRuns(periodId: string): Promise<PayrollRunSummary[]> {
  cache ??= generate()
  await sleep(200)
  return cache.runs
    .filter((r) => !periodId || r.payrollPeriodId === periodId)
    .map(({ items: _items, ...summary }) => ({ ...summary }))
}

async function mockFetchRun(runId: string): Promise<PayrollRun> {
  cache ??= generate()
  await sleep(160)
  const run = cache.runs.find((r) => r.id === runId)
  if (!run) throw new Error(`Payroll run ${runId} not found`)
  return { ...run, items: run.items.map((i) => ({ ...i, components: [...i.components] })) }
}

async function mockFetchPayouts(runId: string): Promise<PayrollPayout[]> {
  cache ??= generate()
  await sleep(180)
  return cache.payouts.filter((p) => !runId || p.payrollRunId === runId).map((p) => ({ ...p }))
}

async function mockFetchBankAccounts(): Promise<PayrollBankAccount[]> {
  cache ??= generate()
  await sleep(120)
  return cache.bankAccounts.map((b) => ({ ...b }))
}

async function mockFetchEmployeeOptions(): Promise<PayrollEmployeeOption[]> {
  cache ??= generate()
  await sleep(120)
  return cache.employees.map((e) => ({ ...e }))
}

function profileFromDraft(draft: CompensationProfileDraft, employeeName: string, jobTitle: string): CompensationProfile {
  return {
    id: draft.id || `profile-new-${Math.floor(Math.random() * 100000)}`,
    employeeId: draft.employeeId,
    employeeName,
    jobTitle,
    division: draft.division,
    payFrequency: draft.payFrequency,
    currency: 'BHD',
    baseSalary: draft.baseSalary,
    housingAllowance: draft.housingAllowance,
    transportAllowance: draft.transportAllowance,
    otherAllowance: draft.otherAllowance,
    standardDeduction: draft.standardDeduction,
    taxDeduction: draft.taxDeduction,
    employerCost: draft.employerCost,
    effectiveFrom: draft.effectiveFrom,
    effectiveTo: draft.effectiveTo,
    isActive: draft.isActive,
    status: draft.isActive ? 'Active' : 'Inactive',
    notes: draft.notes,
  }
}

async function mockUpsertProfile(draft: CompensationProfileDraft): Promise<CompensationProfile> {
  cache ??= generate()
  await sleep(180)
  const emp = cache.employees.find((e) => e.id === draft.employeeId)
  const employeeName = emp?.name ?? ''
  const jobTitle = emp?.jobTitle ?? ''
  if (draft.id) {
    const idx = cache.profiles.findIndex((p) => p.id === draft.id)
    if (idx === -1) throw new Error(`Compensation profile ${draft.id} not found`)
    const updated = profileFromDraft(draft, employeeName, jobTitle)
    cache.profiles[idx] = updated
    return { ...updated }
  }
  const created = profileFromDraft(draft, employeeName, jobTitle)
  cache.profiles.unshift(created)
  return { ...created }
}

async function mockCreatePeriod(draft: PayrollPeriodDraft): Promise<PayrollPeriod> {
  cache ??= generate()
  await sleep(160)
  const created: PayrollPeriod = {
    id: `per-new-${cache.periods.length + 1}`,
    name: draft.name || 'Untitled Payroll Period',
    division: draft.division,
    periodStart: draft.periodStart,
    periodEnd: draft.periodEnd,
    paymentDate: draft.paymentDate,
    status: 'open',
    notes: draft.notes,
  }
  cache.periods.unshift(created)
  return { ...created }
}

async function mockGenerateRun(periodId: string): Promise<PayrollRun> {
  cache ??= generate()
  await sleep(220)
  const period = cache.periods.find((p) => p.id === periodId)
  if (!period) throw new Error(`Payroll period ${periodId} not found`)
  const eligible = cache.profiles.filter((p) => p.isActive && p.division === period.division)
  const items: PayrollRunItem[] = eligible.map((p, idx) => {
    const gross = p.baseSalary + p.housingAllowance + p.transportAllowance + p.otherAllowance
    const deductions = p.standardDeduction + p.taxDeduction
    return {
      id: `item-new-${idx}`,
      payrollRunId: '',
      employeeId: p.employeeId,
      employeeName: p.employeeName,
      employeeNameSnapshot: p.employeeName,
      jobTitleSnapshot: p.jobTitle,
      baseSalary: p.baseSalary,
      allowancesTotal: p.housingAllowance + p.transportAllowance + p.otherAllowance,
      deductionsTotal: deductions,
      employerCostTotal: p.employerCost,
      grossPay: gross,
      netPay: gross - deductions,
      status: 'pending',
      payoutId: '',
      payoutStatus: 'scheduled',
      payoutPaidAt: '',
      components: [],
    }
  })
  const run: PayrollRun = {
    id: `run-new-${cache.runs.length + 1}`,
    runNumber: `PR-${period.periodStart.slice(0, 4)}-NEW${pad(cache.runs.length + 1, 3)}`,
    payrollPeriodId: period.id,
    division: period.division,
    periodName: period.name,
    status: 'draft',
    generatedAt: todayIso(),
    approvedAt: '',
    postedAt: '',
    paidAt: '',
    paymentReference: '',
    bankAccountId: '',
    totalEmployees: items.length,
    grossTotal: items.reduce((s, i) => s + i.grossPay, 0),
    deductionsTotal: items.reduce((s, i) => s + i.deductionsTotal, 0),
    netTotal: items.reduce((s, i) => s + i.netPay, 0),
    employerCostTotal: items.reduce((s, i) => s + i.employerCostTotal, 0),
    currency: 'BHD',
    notes: '',
    items: items.map((i) => ({ ...i, payrollRunId: `run-new-${cache!.runs.length + 1}` })),
  }
  cache.runs.unshift(run)
  return { ...run }
}

function findRunOrThrow(runId: string): PayrollRun {
  cache ??= generate()
  const run = cache.runs.find((r) => r.id === runId)
  if (!run) throw new Error(`Payroll run ${runId} not found`)
  return run
}

async function mockApproveRun(runId: string, _notes: string): Promise<PayrollRun> {
  await sleep(160)
  const run = findRunOrThrow(runId)
  run.status = 'approved'
  run.approvedAt = todayIso()
  return { ...run }
}

async function mockPostRun(runId: string): Promise<PayrollRun> {
  await sleep(160)
  const run = findRunOrThrow(runId)
  run.status = 'posted'
  run.postedAt = todayIso()
  return { ...run }
}

async function mockGeneratePayslipPDF(employeeId: string, _payrollPeriodId: string): Promise<string> {
  await sleep(300)
  return `C:\\Users\\demo\\Documents\\AsymmFlow Exports\\Reports\\Payslip_${employeeId}_Jun_2026_Payroll.pdf`
}

async function mockMarkPaid(runId: string, paidAtIso: string, paymentReference: string, bankAccountId: string): Promise<PayrollRun> {
  await sleep(180)
  const run = findRunOrThrow(runId)
  run.status = 'paid'
  run.paidAt = paidAtIso || todayIso()
  run.paymentReference = paymentReference
  run.bankAccountId = bankAccountId
  return { ...run }
}

/* ---- real: FETCH is wired (single-call, non-aggregating); every mutation
 * is an honest INTEG-gap throw naming the exact real binding — a financial +
 * PII hot-zone doesn't get a naive pass-through. ---- */

function mapComponent(c: Record<string, unknown>): PayrollComponent {
  return {
    id: str(c.id),
    payrollRunItemId: str(c.payroll_run_item_id),
    componentType: str(c.component_type),
    code: str(c.code),
    name: str(c.name),
    amount: num(c.amount),
  }
}

function mapRunItem(i: Record<string, unknown>): PayrollRunItem {
  return {
    id: str(i.id),
    payrollRunId: str(i.payroll_run_id),
    employeeId: str(i.employee_id),
    employeeName: str(i.employee_name),
    employeeNameSnapshot: str(i.employee_name_snapshot),
    jobTitleSnapshot: str(i.job_title_snapshot),
    baseSalary: num(i.base_salary),
    allowancesTotal: num(i.allowances_total),
    deductionsTotal: num(i.deductions_total),
    employerCostTotal: num(i.employer_cost_total),
    grossPay: num(i.gross_pay),
    netPay: num(i.net_pay),
    status: str(i.status),
    payoutId: str(i.payout_id),
    payoutStatus: str(i.payout_status),
    payoutPaidAt: goDate(i.payout_paid_at),
    components: ((i.components as unknown as Record<string, unknown>[] | undefined) ?? []).map(mapComponent),
  }
}

function mapRun(r: Record<string, unknown>): PayrollRun {
  return {
    id: str(r.id),
    runNumber: str(r.run_number),
    payrollPeriodId: str(r.payroll_period_id),
    division: str(r.division),
    periodName: str(r.period_name),
    status: str(r.status) || 'draft',
    generatedAt: goDate(r.generated_at),
    approvedAt: goDate(r.approved_at),
    postedAt: goDate(r.posted_at),
    paidAt: goDate(r.paid_at),
    paymentReference: str(r.payment_reference),
    bankAccountId: str(r.bank_account_id),
    totalEmployees: num(r.total_employees),
    grossTotal: num(r.gross_total),
    deductionsTotal: num(r.deductions_total),
    netTotal: num(r.net_total),
    employerCostTotal: num(r.employer_cost_total),
    currency: str(r.currency) || 'BHD',
    notes: str(r.notes),
    items: ((r.items as unknown as Record<string, unknown>[] | undefined) ?? []).map(mapRunItem),
  }
}

function mapPeriod(p: Record<string, unknown>): PayrollPeriod {
  return {
    id: str(p.id),
    name: str(p.name),
    division: str(p.division),
    periodStart: goDate(p.period_start),
    periodEnd: goDate(p.period_end),
    paymentDate: goDate(p.payment_date),
    status: str(p.status) || 'open',
    notes: str(p.notes),
  }
}

function mapPayout(p: Record<string, unknown>): PayrollPayout {
  return {
    id: str(p.id),
    payrollRunId: str(p.payroll_run_id),
    employeeId: str(p.employee_id),
    division: str(p.division),
    employeeName: str(p.employee_name),
    runNumber: str(p.run_number),
    scheduledAt: goDate(p.scheduled_at),
    paidAt: goDate(p.paid_at),
    amount: num(p.amount),
    currency: str(p.currency) || 'BHD',
    status: str(p.status) || 'scheduled',
    paymentReference: str(p.payment_reference),
  }
}

function mapProfile(p: Record<string, unknown>): CompensationProfile {
  const isActive = Boolean(p.is_active)
  return {
    id: str(p.id),
    employeeId: str(p.employee_id),
    employeeName: str(p.employee_name),
    jobTitle: str(p.job_title),
    division: str(p.division),
    payFrequency: str(p.pay_frequency) || 'monthly',
    currency: str(p.currency) || 'BHD',
    baseSalary: num(p.base_salary),
    housingAllowance: num(p.housing_allowance),
    transportAllowance: num(p.transport_allowance),
    otherAllowance: num(p.other_allowance),
    standardDeduction: num(p.standard_deduction),
    taxDeduction: num(p.tax_deduction),
    employerCost: num(p.employer_cost),
    effectiveFrom: goDate(p.effective_from),
    effectiveTo: goDate(p.effective_to),
    isActive,
    status: isActive ? 'Active' : 'Inactive',
    notes: str(p.notes),
  }
}

function mapBankAccount(b: Record<string, unknown>): PayrollBankAccount {
  return {
    id: str(b.id),
    division: str(b.division),
    bankName: str(b.bank_name),
    accountName: str(b.account_name),
    accountNumber: str(b.account_number),
    currency: str(b.currency) || 'BHD',
  }
}

async function realFetchProfiles(activeOnly: boolean): Promise<CompensationProfile[]> {
  const rows = await ListEmployeeCompensationProfiles(activeOnly)
  return (rows ?? []).map((r) => mapProfile(r as unknown as Record<string, unknown>))
}

async function realFetchPeriods(openOnly: boolean): Promise<PayrollPeriod[]> {
  const rows = await ListPayrollPeriods(openOnly)
  return (rows ?? []).map((r) => mapPeriod(r as unknown as Record<string, unknown>))
}

async function realFetchRuns(periodId: string): Promise<PayrollRunSummary[]> {
  const rows = await ListPayrollRuns(periodId)
  return (rows ?? []).map((r) => {
    const { items: _items, ...summary } = mapRun(r as unknown as Record<string, unknown>)
    return summary
  })
}

async function realFetchRun(runId: string): Promise<PayrollRun> {
  const r = await GetPayrollRun(runId)
  return mapRun(r as unknown as Record<string, unknown>)
}

async function realFetchPayouts(runId: string): Promise<PayrollPayout[]> {
  const rows = await ListPayrollPayouts(runId)
  return (rows ?? []).map((r) => mapPayout(r as unknown as Record<string, unknown>))
}

async function realFetchBankAccounts(): Promise<PayrollBankAccount[]> {
  const rows = await GetActiveBankAccounts()
  return (rows ?? []).map((r) => mapBankAccount(r as unknown as Record<string, unknown>))
}

/** Employee master list for the Compensation form's picker. Cross-domain by
 * design — the old screen used collaboration.listEmployeeProfiles; the bound
 * `App.ListEmployeeProfiles(activeOnly)` is the same read (returns []Employee).
 * Read-only. Division is left blank when the backend doesn't carry one on the
 * Employee (the form falls back to the first division chip). */
async function realFetchEmployeeOptions(): Promise<PayrollEmployeeOption[]> {
  const rows = await ListEmployeeProfiles(true)
  return (rows ?? []).map((r) => {
    const e = r as unknown as Record<string, unknown>
    return {
      id: str(e.id),
      name: str(e.full_name) || str(e.preferred_name),
      jobTitle: str(e.job_title),
      division: str(e.division),
    }
  })
}

/** UpsertEmployeeCompensationProfile — financial + PII hot-zone. Assembles the
 * full payroll.CompensationProfile struct from the draft (R1 technique). The
 * server owns the actor (CreatedBy = getCurrentUserID), currency default, and
 * the cross-division clobber guard — we send exactly the draft's fields. Nullable
 * effective dates cross as `null` when blank (a *time.Time nil), never as Go
 * zero-time. currency is fixed BHD (the draft carries no currency field). */
async function realUpsertProfile(draft: CompensationProfileDraft): Promise<CompensationProfile> {
  const payload = {
    id: draft.id ?? '',
    employee_id: draft.employeeId,
    division: draft.division,
    pay_frequency: draft.payFrequency,
    currency: 'BHD',
    base_salary: draft.baseSalary,
    housing_allowance: draft.housingAllowance,
    transport_allowance: draft.transportAllowance,
    other_allowance: draft.otherAllowance,
    standard_deduction: draft.standardDeduction,
    tax_deduction: draft.taxDeduction,
    employer_cost: draft.employerCost,
    effective_from: draft.effectiveFrom ? goTime(draft.effectiveFrom) : null,
    effective_to: draft.effectiveTo ? goTime(draft.effectiveTo) : null,
    is_active: draft.isActive,
    notes: draft.notes,
  }
  const result = await UpsertEmployeeCompensationProfile(payload as unknown as Parameters<typeof UpsertEmployeeCompensationProfile>[0])
  return mapProfile(result as unknown as Record<string, unknown>)
}

async function realCreatePeriod(draft: PayrollPeriodDraft): Promise<PayrollPeriod> {
  // Guard the two required dates at the seam: the backend rejects a zero-time
  // period (start/end required, end >= start) — a blank date must never cross
  // the wire as Go zero-time. payment_date is OMITTED when blank so the
  // backend's nil-default (payment_date := period_end) applies, rather than a
  // stored 0001 date. name/status left for the backend to default when empty.
  if (!draft.periodStart || !draft.periodEnd) {
    throw new Error('Period start and end dates are required.')
  }
  const period = {
    name: draft.name,
    division: draft.division,
    period_start: goTime(draft.periodStart),
    period_end: goTime(draft.periodEnd),
    ...(draft.paymentDate ? { payment_date: goTime(draft.paymentDate) } : {}),
    notes: draft.notes,
  } as unknown as payroll.Period
  const created = await CreatePayrollPeriod(period)
  return mapPeriod(created as unknown as Record<string, unknown>)
}

async function realGenerateRun(periodId: string): Promise<PayrollRun> {
  const run = await GeneratePayrollRun(periodId)
  return mapRun(run as unknown as Record<string, unknown>)
}

async function realApproveRun(runId: string, notes: string): Promise<PayrollRun> {
  const run = await ApprovePayrollRun(runId, notes)
  return mapRun(run as unknown as Record<string, unknown>)
}

async function realPostRun(runId: string): Promise<PayrollRun> {
  // Posts the salary GL — irreversible. The backend enforces the state machine
  // (only an approved run posts); the seam just passes the id through.
  const run = await PostPayrollRun(runId)
  return mapRun(run as unknown as Record<string, unknown>)
}

async function realMarkPaid(runId: string, paidAtIso: string, paymentReference: string, bankAccountId: string): Promise<PayrollRun> {
  // MarkPayrollRunPaid(runID, paidAtISO, paymentReference, bankAccountID) — the
  // date arg is a plain ISO string on the Go side (parsed server-side), NOT a
  // time.Time binding param, so it passes through without goTime.
  const run = await MarkPayrollRunPaid(runId, paidAtIso, paymentReference, bankAccountId)
  return mapRun(run as unknown as Record<string, unknown>)
}

/** GeneratePayslipPDF — export/read action, NOT a financial mutation: it
 * renders a PDF from data already committed by the run/approve/post/pay
 * steps above and returns the saved file path. Wired for real like the
 * FETCH bindings (no INTEG-gap throw). */
async function realGeneratePayslipPDF(employeeId: string, payrollPeriodId: string): Promise<string> {
  return GeneratePayslipPDF(employeeId, payrollPeriodId)
}

/* ---- public switched API (viewmodel imports THESE) ---- */
export const fetchCompensationProfiles = (activeOnly = false): Promise<CompensationProfile[]> =>
  pick(realFetchProfiles, mockFetchProfiles)(activeOnly)
export const fetchPayrollPeriods = (openOnly = false): Promise<PayrollPeriod[]> =>
  pick(realFetchPeriods, mockFetchPeriods)(openOnly)
export const fetchPayrollRuns = (periodId = ''): Promise<PayrollRunSummary[]> =>
  pick(realFetchRuns, mockFetchRuns)(periodId)
export const fetchPayrollRun = (runId: string): Promise<PayrollRun> => pick(realFetchRun, mockFetchRun)(runId)
export const fetchPayrollPayouts = (runId = ''): Promise<PayrollPayout[]> =>
  pick(realFetchPayouts, mockFetchPayouts)(runId)
export const fetchActiveBankAccounts = (): Promise<PayrollBankAccount[]> =>
  pick(realFetchBankAccounts, mockFetchBankAccounts)()
export const fetchPayrollEmployeeOptions = (): Promise<PayrollEmployeeOption[]> =>
  pick(realFetchEmployeeOptions, mockFetchEmployeeOptions)()

export const upsertCompensationProfile = (draft: CompensationProfileDraft): Promise<CompensationProfile> =>
  pick(realUpsertProfile, mockUpsertProfile)(draft)
export const createPayrollPeriod = (draft: PayrollPeriodDraft): Promise<PayrollPeriod> =>
  pick(realCreatePeriod, mockCreatePeriod)(draft)
export const generatePayrollRun = (periodId: string): Promise<PayrollRun> =>
  pick(realGenerateRun, mockGenerateRun)(periodId)
export const approvePayrollRun = (runId: string, notes: string): Promise<PayrollRun> =>
  pick(realApproveRun, mockApproveRun)(runId, notes)
export const postPayrollRun = (runId: string): Promise<PayrollRun> => pick(realPostRun, mockPostRun)(runId)
export const markPayrollRunPaid = (
  runId: string,
  paidAtIso: string,
  paymentReference: string,
  bankAccountId: string,
): Promise<PayrollRun> => pick(realMarkPaid, mockMarkPaid)(runId, paidAtIso, paymentReference, bankAccountId)
export const generatePayslipPdf = (employeeId: string, payrollPeriodId: string): Promise<string> =>
  pick(realGeneratePayslipPDF, mockGeneratePayslipPDF)(employeeId, payrollPeriodId)

/** Client-side division vocabulary — the divisions store lands at K5; until
 * then every payroll list is scoped in the viewmodel via plain equality
 * against this fixed set (mirrors the old screen's `matchesCompany`). */
export const payrollDivisionOptions = (): { value: string; label: string }[] =>
  DIVISIONS.slice(0, 2).map((d) => ({ value: d, label: d }))
