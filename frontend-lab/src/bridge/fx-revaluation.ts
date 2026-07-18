/* FX Revaluation bridge module — self-contained: types + mock + real +
 * switch. K4 scope: the PRIMARY ledger = Revaluations (per-account FX
 * revaluation runs, unrealized gain/loss, post/reverse). The old screen's
 * Exposure and Rates tabs are separate fetches against different shapes
 * (GetFXExposureReport; GetLatestFXRate/CreateFXRate) — ledgered as an
 * ENGINE multi-panel gap, see screens/parity/FxRevaluation.parity.md.
 *
 * Status is intentionally two-state (Draft/Posted), not three. Reading
 * pkg/finance/fx/fx.go's Reverse() directly: reversing a POSTED row does
 * NOT flip that row to a "Reversed" state — it inserts a brand-new
 * reversing row with rates/gain-loss negated (the original stays Posted,
 * unchanged); reversing a DRAFT row just deletes it. The schema has no
 * "Reversed" column, so this bridge doesn't invent one — see the parity
 * doc for the full reasoning. */
import { pick } from './runtime'
import { goDate, num, str } from './map'
import { GetActiveBankAccounts, GetFXRevaluations } from '$wails/go/main/FinanceService'
import { PostFXRevaluation, ReverseRevaluation } from '$wails/go/main/App'
import { actingUserId } from '../stores/session.svelte'

export interface FxRevaluationRow {
  id: string
  bankAccountId: string
  accountLabel: string
  currency: string
  revaluationDate: string
  foreignBalance: number
  previousRate: number
  previousBhd: number
  currentRate: number
  currentBhd: number
  gainLossBhd: number
  status: string
  postedBy: string
  postedAt: string
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
const round3 = (n: number): number => Math.round(n * 1000) / 1000

// Five synthetic foreign-currency accounts — adversarial breadth on the
// account label (monster legal name, RTL, single-char) rather than on
// currency codes, which stay a finite ISO vocabulary.
const FOREIGN_ACCOUNTS: { id: string; label: string; currency: string }[] = [
  { id: 'bank-usd-1', label: 'Ahli United Bank — USD Settlement', currency: 'USD' },
  { id: 'bank-eur-1', label: 'بنك البحرين الوطني — حساب اليورو التشغيلي', currency: 'EUR' },
  { id: 'bank-gbp-1', label: 'HSBC Bahrain — GBP Corporate', currency: 'GBP' },
  {
    id: 'bank-sar-1',
    label:
      'International Establishment for Foreign Currency Treasury Operations, Multi-Bank Settlement Coordination and General Financial Administration (formerly Gulf Treasury Services) W.L.L. — SAR Trade Account',
    currency: 'SAR',
  },
  { id: 'bank-kwd-1', label: 'X', currency: 'KWD' },
]

let cache: FxRevaluationRow[] | null = null
let reversalCount = 0

function generate(): FxRevaluationRow[] {
  const rand = lcg(20260714)
  const rows: FxRevaluationRow[] = []
  const n = 150
  for (let i = 1; i <= n; i++) {
    const acct = FOREIGN_ACCOUNTS[i % FOREIGN_ACCOUNTS.length]!
    const monthIdx = Math.floor(rand() * 18)
    const year = 2025 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    const revaluationDate = `${year}-${pad(month, 2)}-${pad(day, 2)}`

    const baseRate = acct.currency === 'KWD' ? 1.22 : acct.currency === 'GBP' ? 0.47 : acct.currency === 'EUR' ? 0.41 : 0.376
    const previousRate = round3(baseRate * (0.97 + rand() * 0.06))
    const currentRate = round3(baseRate * (0.97 + rand() * 0.06))
    const foreignBalance =
      i % 89 === 0 ? 987654321.123 : i % 53 === 0 ? 0.001 : Math.round(rand() * 800_000 * 100) / 100
    const previousBhd = round3(foreignBalance * previousRate)
    const currentBhd = round3(foreignBalance * currentRate)
    const gainLossBhd = round3(currentBhd - previousBhd)

    const status = i % 97 === 0 ? 'UNKNOWN_STATE' : i % 3 === 0 ? 'Posted' : 'Draft'
    const posted = status === 'Posted'
    const postedBy = posted ? (i % 41 === 0 ? '' : 'Finance Operator') : ''
    const postedAt = posted ? revaluationDate : ''

    rows.push({
      id: `fxr-${i}`,
      bankAccountId: acct.id,
      accountLabel: acct.label,
      currency: acct.currency,
      revaluationDate,
      foreignBalance,
      previousRate,
      previousBhd,
      currentRate,
      currentBhd,
      gainLossBhd,
      status,
      postedBy,
      postedAt,
    })
  }
  return rows
}

async function mockFetchAll(): Promise<FxRevaluationRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

async function mockPost(id: string): Promise<void> {
  cache ??= generate()
  const r = cache.find((x) => x.id === id)
  if (r && r.status === 'Draft') {
    r.status = 'Posted'
    r.postedBy = 'Mock Operator'
    r.postedAt = new Date().toISOString().slice(0, 10)
  }
  await sleep(150)
}

async function mockReverse(id: string, _reason: string): Promise<void> {
  cache ??= generate()
  const idx = cache.findIndex((x) => x.id === id)
  if (idx === -1) return
  const r = cache[idx]!
  if (r.status === 'Posted') {
    // Mirrors fx.go's Reverse(): a reversing row is INSERTED, negated
    // rates/gain-loss, still Posted — the original row is untouched.
    reversalCount++
    const today = new Date().toISOString().slice(0, 10)
    cache.unshift({
      ...r,
      id: `${r.id}-rev${reversalCount}`,
      revaluationDate: today,
      previousRate: r.currentRate,
      previousBhd: r.currentBhd,
      currentRate: r.previousRate,
      currentBhd: r.previousBhd,
      gainLossBhd: round3(-r.gainLossBhd),
      postedBy: 'Mock Operator',
      postedAt: today,
    })
  } else {
    cache.splice(idx, 1) // Reversing a Draft = delete, mirrors real semantics.
  }
  await sleep(150)
}

/* ---- real: fetch WIRED (merged across active foreign-currency accounts),
 * mutations are INTEG-gapped (honest throw) — same trade-off as
 * cheque-register.ts's realMarkStale/realCancel. ---- */
function mapRevaluation(
  bankAccountId: string,
  accountLabel: string,
  currency: string,
): (r: Record<string, unknown>) => FxRevaluationRow {
  return (r) => ({
    id: str(r.id),
    bankAccountId,
    accountLabel,
    currency,
    revaluationDate: goDate(r.revaluation_date),
    foreignBalance: num(r.foreign_balance),
    previousRate: num(r.previous_rate),
    previousBhd: num(r.previous_bhd),
    currentRate: num(r.current_rate),
    currentBhd: num(r.current_bhd),
    gainLossBhd: num(r.gain_loss_bhd),
    status: r.is_posted ? 'Posted' : 'Draft',
    postedBy: str(r.posted_by),
    postedAt: goDate(r.posted_at),
  })
}

async function realFetchAll(): Promise<FxRevaluationRow[]> {
  // No single "all revaluations" binding exists — GetFXRevaluations() is
  // scoped per bank account (fx_revaluation_service.go:102), same shape as
  // cheque.go's Outstanding(). The real bridge fetches every active
  // non-BHD account and merges their revaluation history into one feed.
  const accounts = await GetActiveBankAccounts()
  const foreign = (accounts ?? []).filter(
    (a) => str((a as unknown as Record<string, unknown>).currency) !== 'BHD',
  )
  const perAccount = await Promise.all(
    foreign.map(async (a) => {
      const rec = a as unknown as Record<string, unknown>
      const id = str(rec.id)
      const label = `${str(rec.bank_name)} (${str(rec.currency)})`
      const revals = await GetFXRevaluations(id)
      return (revals ?? []).map((r) =>
        mapRevaluation(id, label, str(rec.currency))(r as unknown as Record<string, unknown>),
      )
    }),
  )
  return perAccount.flat()
}

async function realPost(id: string): Promise<void> {
  // PostFXRevaluation(revaluationID, user) — posts the unrealized-gain/loss GL
  // entries; the acting user is the audit actor. fx_revaluation_golden_test.go
  // covers the posting math + reversal.
  await PostFXRevaluation(id, actingUserId())
}

async function realReverse(id: string, reason: string): Promise<void> {
  // ReverseRevaluation(revaluationID, user, reason) — the un-post path.
  await ReverseRevaluation(id, actingUserId(), reason)
}

/* ---- public switched API (descriptor imports THESE) ---- */
export const fetchFxRevaluations = (): Promise<FxRevaluationRow[]> => pick(realFetchAll, mockFetchAll)()
export const postFxRevaluation = (id: string): Promise<void> => pick(realPost, mockPost)(id)
export const reverseFxRevaluation = (id: string, reason: string): Promise<void> =>
  pick(realReverse, mockReverse)(id, reason)
