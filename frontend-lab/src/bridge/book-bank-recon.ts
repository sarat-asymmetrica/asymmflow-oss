/* Book vs Bank Reconciliation bridge module — self-contained: types + mock +
 * real + switch. Classic month-end statement reconciliation (bank balance +
 * deposits in transit - outstanding cheques = adjusted bank; book balance +
 * errors/NSF/interest = adjusted book; the two must agree). NOT the same
 * screen as transaction-level BankReconciliationScreen (see recon-K4.md
 * synthesis note) — this one compares two running totals, once per period
 * per account, then finalizes. Adversarial by doctrine: monsters are woven
 * INTO the dataset (see bridge/mock.ts). Deterministic (seeded LCG) so
 * Playwright baselines are stable run-to-run. */
import { pick } from './runtime'
import { FinalizeBookBankReconciliation } from '$wails/go/main/InfraService'
import { actingUserId } from '../stores/session.svelte'

export interface ReconciliationLine {
  label: string
  value: number
  note?: string
}

export interface BookBankReconciliationRow {
  id: string
  period: string // 'YYYY-MM'
  bankAccountName: string
  bankAccountNumber: string
  currency: string
  status: string
  bankBalance: number
  depositsInTransit: ReconciliationLine[]
  outstandingCheques: ReconciliationLine[]
  bookBalance: number
  bookAdjustments: ReconciliationLine[]
  finalizedAt: string | null
  finalizedBy: string
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

const BANK_ACCOUNTS: { name: string; number: string }[] = [
  { name: 'BBK Operating', number: '010-233456-001' },
  { name: 'NBB Current', number: '021-887744-002' },
  { name: 'HSBC USD Current', number: '445-102938-010' },
  {
    // Unbroken monster token — a real bank export line pasted whole.
    name: 'INTERNTIONALESTABLISHMENTOPERATINGACCOUNTFORMERLYGULFTECHNICALHOLDCO'.padEnd(70, 'X'),
    number: '',
  },
  { name: '', number: '099-000000-000' },
]

const DEPOSIT_LABELS = [
  'Deposit in transit — month end',
  'EFT received, not yet cleared',
  'Cheque lodged, awaiting clearance',
  'Card settlement pending',
]

const CHEQUE_LABELS = [
  'Cheque #4821 — Al Dana Engineering',
  'Cheque #4855 — Sitra Contracting',
  'Cheque #4901 — payroll run',
  'Cheque #4912 — supplier payment',
  'Cheque #4930 — utility payment',
]

const ADJUSTMENT_POOL: { label: string; sign: 1 | -1; note?: string }[] = [
  { label: 'Bank service charge', sign: -1 },
  { label: 'Interest earned', sign: 1 },
  { label: 'NSF cheque returned', sign: -1, note: 'Customer cheque bounced' },
  { label: 'Recording error correction', sign: 1, note: 'Book entry corrected to match statement' },
]

const STATUSES = ['Draft', 'Reconciled', 'Finalized']

function makeLines(
  pool: string[],
  rand: () => number,
  count: number,
  magnitude: number,
): ReconciliationLine[] {
  const lines: ReconciliationLine[] = []
  for (let j = 0; j < count; j++) {
    lines.push({
      label: pool[Math.floor(rand() * pool.length)]!,
      value: Math.round(rand() * magnitude * 100) / 100,
    })
  }
  return lines
}

let cache: BookBankReconciliationRow[] | null = null

function generate(): BookBankReconciliationRow[] {
  const rand = lcg(20260714)
  const rows: BookBankReconciliationRow[] = []
  const n = 26
  for (let i = 1; i <= n; i++) {
    const monthIdx = i % 13 // spread across ~13 months
    const year = 2025 + Math.floor((6 + monthIdx) / 12)
    const month = ((6 + monthIdx) % 12) + 1
    const period = `${year}-${pad(month, 2)}`
    const acct = BANK_ACCOUNTS[i % BANK_ACCOUNTS.length]!
    const currency = i % 41 === 0 ? 'USD' : acct.name.startsWith('HSBC') ? 'USD' : 'BHD'

    // Monster row: huge statement balance.
    const bankBalance = i % 23 === 0 ? 123456789012.345 : Math.round(rand() * 4_000_000) / 100
    // Monster row: negative book balance (overdrawn control account).
    const bookBalanceRaw = Math.round(rand() * 4_000_000) / 100
    const bookBalance = i % 17 === 0 ? -Math.abs(bookBalanceRaw) : bookBalanceRaw

    // Monster row: zero supporting lines at all — variance (if any) has no
    // detail to explain it, exercising the panel's empty-lines rendering.
    const zeroLines = i % 13 === 0
    const depositsInTransit = zeroLines ? [] : makeLines(DEPOSIT_LABELS, rand, 1 + Math.floor(rand() * 3), 40_000)
    const outstandingCheques = zeroLines ? [] : makeLines(CHEQUE_LABELS, rand, 1 + Math.floor(rand() * 3), 15_000)

    let bookAdjustments: ReconciliationLine[] = []
    if (!zeroLines) {
      const adjCount = 1 + Math.floor(rand() * 3)
      for (let j = 0; j < adjCount; j++) {
        const a = ADJUSTMENT_POOL[Math.floor(rand() * ADJUSTMENT_POOL.length)]!
        bookAdjustments.push({
          label: a.label,
          value: a.sign * Math.round(rand() * 2_000 * 100) / 100,
          ...(a.note ? { note: a.note } : {}),
        })
      }
    }

    // Monster row: forced-reconciled — adjustments engineered so the two
    // sides land exactly on tolerance (proves the "Reconciled" success path
    // renders, not just the danger path every other row exercises).
    if (i % 7 === 0 && !zeroLines) {
      const adjustedBank =
        bankBalance + depositsInTransit.reduce((s, l) => s + l.value, 0) - outstandingCheques.reduce((s, l) => s + l.value, 0)
      bookAdjustments = [{ label: 'Balancing entry', value: adjustedBank - bookBalance }]
    }

    const status = i % 89 === 0 ? 'UNKNOWN_STATE' : i % 5 === 0 ? 'Finalized' : STATUSES[Math.floor(rand() * STATUSES.length)]!
    const finalized = status === 'Finalized'

    rows.push({
      id: `bbr-${i}`,
      period,
      bankAccountName: acct.name,
      bankAccountNumber: acct.number,
      currency,
      status,
      bankBalance,
      depositsInTransit,
      outstandingCheques,
      bookBalance,
      bookAdjustments,
      finalizedAt: finalized ? `${period}-28` : null,
      finalizedBy: finalized ? 'F. Al Khalifa' : '',
    })
  }
  return rows
}

async function mockFetch(): Promise<BookBankReconciliationRow[]> {
  cache ??= generate()
  await sleep(250)
  return cache.map((r) => ({ ...r, depositsInTransit: [...r.depositsInTransit], outstandingCheques: [...r.outstandingCheques], bookAdjustments: [...r.bookAdjustments] }))
}

async function mockFinalize(id: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (!row) throw new Error(`Reconciliation ${id} not found`)
  await sleep(150)
  row.status = 'Finalized'
  row.finalizedAt = new Date().toISOString().slice(0, 10)
  row.finalizedBy = 'You (mock)'
}

/* ---- real: GetBookBankReconciliations returns the header rows; the detail
 * lines come from two further calls per record (GetDepositsInTransit,
 * GetOutstandingCheques — FinanceService), and book-side adjustments from
 * UpdateBookBankReconciliationAdjustments's read path. All INTEG-gapped —
 * wiring is a 3-call aggregation per record (same shape as cheque-register.ts's
 * realFetchAll), not a straight 1:1 swap. ---- */
async function realFetch(): Promise<BookBankReconciliationRow[]> {
  throw new Error(
    'INTEG gap: GetBookBankReconciliations + GetDepositsInTransit + GetOutstandingCheques — wires at K5',
  )
}

async function realFinalize(id: string): Promise<void> {
  // InfraService.FinalizeBookBankReconciliation(id, user) → void. Finalizing
  // user comes from the session (posting-adjacent: locks the period recon).
  await FinalizeBookBankReconciliation(id, actingUserId())
}

/* ---- public switched API (viewmodel imports THESE) ---- */
export const fetchBookBankReconciliations = (): Promise<BookBankReconciliationRow[]> =>
  pick(realFetch, mockFetch)()
export const finalizeBookBankReconciliation = (id: string): Promise<void> =>
  pick(realFinalize, mockFinalize)(id)
