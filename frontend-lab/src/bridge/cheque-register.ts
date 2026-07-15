/* Cheque Register bridge module — self-contained: types + mock + real +
 * switch. K1 scope: the PRIMARY ledger only (Outstanding cheques). The old
 * screen's other two sub-views (cheque-book Registers, Stale-only tab) are
 * separate bank-account-scoped fetches — ledgered, see
 * screens/parity/ChequeRegister.parity.md. Issue Cheque and the bank-line
 * picker on Mark Cleared are financial-hot-zone SLOTs, also ledgered. */
import { pick } from './runtime'
import { goDate, num, str } from './map'
import { GetActiveBankAccounts, GetOutstandingCheques } from '$wails/go/main/FinanceService'

export interface OutstandingChequeRow {
  id: string
  bankAccountId: string
  chequeNumber: string
  amount: number
  currency: string
  issuedDate: string
  payeeName: string
  payeeType: string
  purpose: string
  status: string
  isStale: boolean
}

/* ---- mock: adversarial + deterministic (see bridge/mock.ts) ---- */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
const pad = (n: number, w: number): string => String(n).padStart(w, '0')

// Two synthetic bank accounts — the real account-picker (an ENGINE-scoped
// selector that reloads data) is out of K1 scope, so mock shows cheques
// across both accounts at once rather than gating on a selection.
const BANK_ACCOUNTS = ['bank-1', 'bank-2']

const SUPPLIER_PAYEES = [
  'Bahrain Precision Instruments W.L.L.',
  'Gulf Valve & Actuator Trading Co.',
  'Al Manar Industrial Supplies',
  'International Establishment for Process Control Equipment, Calibration Services, Spare Parts Distribution and General Engineering Trading (formerly Gulf Technical Instrumentation Company) W.L.L.',
  'شركة الخليج للتوريدات الصناعية والمعايرة ذ.م.م',
  'Sitra Metal Works',
  'X',
]
const EMPLOYEE_PAYEES = ['Ahmed Al-Khalifa', 'Fatima Hassan', 'Store Petty Cash Custodian', '']
const OTHER_PAYEES = [
  'Bahrain Electricity & Water Authority',
  'Ministry of Industry & Commerce — Licensing Fees',
  'Municipal Council — Trade License Renewal',
]

const PURPOSES = [
  'Supplier settlement',
  'Payroll advance',
  'Utility payment',
  'Petty cash top-up',
  'Government fee',
  'Rent',
]

// Mirrors pkg/finance/cheque/cheque.go's status-changing methods exactly:
// MarkCleared/MarkStale/MarkBounced gate on status IN (ISSUED, PRESENTED);
// Cancel gates on ISSUED only. Real "Outstanding" fetch (cheque.go:274) only
// ever returns ISSUED/PRESENTED rows — CLEARED/CANCELLED/BOUNCED/STALE are
// seasoned into the mock as adversarial breadth for badge/tone rendering,
// not because the real endpoint would surface them here.
const STATUSES = ['ISSUED', 'PRESENTED', 'CLEARED', 'STALE', 'CANCELLED', 'BOUNCED']

let cache: OutstandingChequeRow[] | null = null

function generate(): OutstandingChequeRow[] {
  const rand = lcg(20260714)
  const rows: OutstandingChequeRow[] = []
  const n = 260
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 20)
    const year = 2024 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    const issuedDate = `${year}-${pad(month, 2)}-${pad(day, 2)}`

    const status =
      i % 97 === 0
        ? 'UNKNOWN_STATE'
        : i % 11 === 0
          ? 'CLEARED'
          : i % 13 === 0
            ? 'CANCELLED'
            : i % 17 === 0
              ? 'BOUNCED'
              : i % 7 === 0
                ? 'STALE'
                : i % 3 === 0
                  ? 'PRESENTED'
                  : 'ISSUED'

    const payeeType = i % 23 === 0 ? 'OTHER' : i % 5 === 0 ? 'EMPLOYEE' : 'SUPPLIER'
    const payeePool = payeeType === 'OTHER' ? OTHER_PAYEES : payeeType === 'EMPLOYEE' ? EMPLOYEE_PAYEES : SUPPLIER_PAYEES
    const payeeName =
      i % 67 === 0
        ? ''
        : i % 83 === 0
          ? 'M'.repeat(200) // unbroken 200-char monster
          : payeePool[i % payeePool.length]!

    const amount = i % 89 === 0 ? 234567890.123 : i % 53 === 0 ? 0.001 : Math.round(rand() * 50_000) / 100
    const currency = i % 41 === 0 ? 'USD' : 'BHD'
    const isStale = status === 'STALE' || (status !== 'CANCELLED' && status !== 'CLEARED' && status !== 'BOUNCED' && r < 0.06)

    rows.push({
      id: `chq-${i}`,
      bankAccountId: BANK_ACCOUNTS[i % BANK_ACCOUNTS.length]!,
      chequeNumber: i % 71 === 0 ? `${pad(i, 40)}` : `${pad(100000 + i, 6)}`,
      amount,
      currency,
      issuedDate,
      payeeName,
      payeeType,
      purpose: PURPOSES[i % PURPOSES.length]!,
      status,
      isStale,
    })
  }
  return rows
}

async function mockFetchAll(): Promise<OutstandingChequeRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

async function mockMarkStale(chequeNumber: string): Promise<void> {
  cache ??= generate()
  const c = cache.find((x) => x.chequeNumber === chequeNumber)
  if (c) {
    c.status = 'STALE'
    c.isStale = true
  }
  await sleep(150)
}

async function mockCancel(chequeNumber: string, _reason: string): Promise<void> {
  cache ??= generate()
  const c = cache.find((x) => x.chequeNumber === chequeNumber)
  if (c) c.status = 'CANCELLED'
  await sleep(150)
}

/* ---- real: fetch WIRED (merged across active bank accounts), mutations
 * are INTEG-gapped (honest throw) ---- */
function mapOutstandingCheque(r: Record<string, unknown>): OutstandingChequeRow {
  return {
    id: str(r.id),
    bankAccountId: str(r.bank_account_id),
    chequeNumber: str(r.cheque_number),
    amount: num(r.amount),
    currency: str(r.currency) || 'BHD',
    issuedDate: goDate(r.issued_date),
    payeeName: str(r.payee_name),
    payeeType: str(r.payee_type),
    purpose: str(r.purpose),
    status: str(r.status) || 'ISSUED',
    isStale: !!r.is_stale,
  }
}

async function realFetchAll(): Promise<OutstandingChequeRow[]> {
  // No single unpaged "all cheques" binding exists — Outstanding() is scoped
  // per bank account (cheque.go:272). The account-picker itself is out of
  // K1 scope (ledgered), so the real bridge merges every active account's
  // outstanding cheques into one feed, same shape the mock presents.
  const accounts = await GetActiveBankAccounts()
  const perAccount = await Promise.all(
    (accounts ?? []).map((a) => GetOutstandingCheques(str((a as unknown as Record<string, unknown>).id))),
  )
  return perAccount.flatMap((result) => {
    const cheques = (result as unknown as { cheques?: unknown[] })?.cheques ?? []
    return cheques.map((c) => mapOutstandingCheque(c as unknown as Record<string, unknown>))
  })
}

async function realMarkStale(_chequeNumber: string): Promise<void> {
  throw new Error('INTEG gap: MarkChequeStale — wires at K5')
}

async function realCancel(_chequeNumber: string, _reason: string): Promise<void> {
  throw new Error('INTEG gap: CancelCheque — wires at K5')
}

/* ---- public switched API (descriptor imports THESE) ---- */
export const fetchChequeRegister = (): Promise<OutstandingChequeRow[]> => pick(realFetchAll, mockFetchAll)()
export const markChequeStale = (chequeNumber: string): Promise<void> =>
  pick(realMarkStale, mockMarkStale)(chequeNumber)
export const cancelCheque = (chequeNumber: string, reason: string): Promise<void> =>
  pick(realCancel, mockCancel)(chequeNumber, reason)
