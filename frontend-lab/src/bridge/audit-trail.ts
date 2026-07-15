/* Audit Trail bridge module — self-contained: types + mock + real + switch.
 * Old screen: AuditTrailViewer.svelte — bank-reconciliation audit trail,
 * browsed per bank account → per statement → `GetAuditTrail(statementId)`
 * (real signature confirmed, `finance.BankReconciliationAuditLog`). This
 * bridge flattens that three-level real fetch (accounts → statements →
 * per-statement audit log) into one feed, the same shape cheque-register's
 * `realFetchAll` merges across bank accounts.
 *
 * Per the K4 orchestrator brief, BOTH fetch and mutations are INTEG-gapped —
 * this is a financial/ledger-integrity control (Article V.4/B3(c): row-click
 * is read-only, Reverse is a separate explicit action, never click-to-reverse)
 * and the real fetch chain is multi-level, so mock stands in entirely until K5. */

import { pick } from './runtime'

export interface AuditTrailRow {
  id: string
  timestamp: string
  /** IMPORT | MATCH | UNMATCH | SPLIT | CATEGORIZE | RECONCILE | VERIFY. */
  action: string
  statementRef: string
  actor: string
  amount: number
  reversed: boolean
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

const ACTIONS = ['IMPORT', 'MATCH', 'UNMATCH', 'SPLIT', 'CATEGORIZE', 'RECONCILE', 'VERIFY']
const ACTORS = ['Aisha Al-Rumaihi', 'Mohammed Bucheeri', 'Fatima Al-Zayani', 'System (auto-match)', '']

let cache: AuditTrailRow[] | null = null

function generate(): AuditTrailRow[] {
  const rand = lcg(20260714 + 5)
  const rows: AuditTrailRow[] = []
  const n = 200
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 12)
    const day = 1 + Math.floor(rand() * 27)
    const timestamp = `2026-${pad(1 + (monthIdx % 12), 2)}-${pad(day, 2)}`
    const action = i % 97 === 0 ? 'UNKNOWN_ACTION' : ACTIONS[Math.floor(r * ACTIONS.length)]!
    const amount = i % 89 === 0 ? 76543210987.654 : i % 53 === 0 ? 0.001 : Math.round(rand() * 500_000) / 100
    // MATCH/RECONCILE actions are the ones that get reversed in practice —
    // IMPORT/VERIFY never do. ~12% of eligible actions seasoned as reversed.
    const eligible = action === 'MATCH' || action === 'RECONCILE' || action === 'CATEGORIZE'
    const reversed = eligible && i % 8 === 0

    rows.push({
      id: `atl-${i}`,
      timestamp,
      action,
      statementRef: `STMT-2026-${pad(1 + (i % 24), 4)}`,
      actor: ACTORS[i % ACTORS.length]!,
      amount,
      reversed,
    })
  }
  return rows
}

async function mockFetch(): Promise<AuditTrailRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

async function mockReverse(row: AuditTrailRow, reason: string): Promise<void> {
  void reason // reversal_reason — recorded server-side, not surfaced as a column here
  cache ??= generate()
  const found = cache.find((x) => x.id === row.id)
  if (found) found.reversed = true
  await sleep(150)
}

/* ---- real: INTEG-gapped entirely (multi-level fetch chain; reversal is a
 * ledger-integrity hot zone) — naming the exact bindings for K5 ---- */

async function realFetch(): Promise<AuditTrailRow[]> {
  throw new Error(
    'INTEG gap: GetActiveBankAccounts → GetBankStatements(accountId) → GetAuditTrail(statementId), merged — wires at K5',
  )
}

async function realReverse(_row: AuditTrailRow, _reason: string): Promise<void> {
  void _row
  void _reason
  throw new Error('INTEG gap: ReverseAction(logId, user, reason) — wires at K5')
}

/* ---- public switched API (descriptor imports THESE) ---- */

export const fetchAuditTrail = (): Promise<AuditTrailRow[]> => pick(realFetch, mockFetch)()
export const reverseAuditAction = (row: AuditTrailRow, reason: string): Promise<void> =>
  pick(realReverse, mockReverse)(row, reason)
