/* Approvals Queue bridge module — self-contained: types + mock + real + switch.
 * Durable admin queue merging two real request kinds: delete-approval requests
 * (`pkg/infra/deletion`, list via `ListDeleteApprovalRequests`) and employee-
 * archive requests (`ListEmployeeArchiveRequests`) — same merge the old
 * ApprovalsQueueScreen.svelte does. Both request kinds share one status
 * vocabulary end to end: "pending" | "approved" | "rejected", decisions are
 * "approve" | "reject" (confirmed verbatim in `pkg/infra/deletion/deletion.go`
 * and `employee_archive_service_test.go`).
 *
 * Per the K4 orchestrator brief, BOTH fetch and mutations are INTEG-gapped —
 * this queue is admin-privileged (`ListDeleteApprovalRequests` returns an
 * empty list for non-admin sessions server-side) and carries employee PII, so
 * the lab stays on synthetic data end to end until K5. */

import { pick } from './runtime'

export interface ApprovalRow {
  id: string
  kind: 'delete' | 'archive'
  target: string
  requestedBy: string
  requestedAt: string
  reason: string
  /** Mirrors the real status vocabulary verbatim: pending | approved | rejected. */
  status: string
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

// Synthetic operator names (SYNTHETIC_IDENTITY.md canon) — mock/test data,
// exempt territory.
const REQUESTERS = ['Aisha Al-Rumaihi', 'Mohammed Bucheeri', 'Fatima Al-Zayani', 'Yusuf Kanoo', '', 'X']
const DELETE_TARGETS = [
  'Invoice INV-2026-0417',
  'Purchase Order PO-2025-1188',
  'Cheque CHQ-000482',
  'Supplier Invoice SI-2026-0093',
  'Credit Note CN-2025-0061',
]
const EMPLOYEE_TARGETS = ['Khalid Al-Mannai', 'Noora Al-Sulaiti', 'Ibrahim Alawi', 'Reem Al-Khalifa', '']
const REASONS = [
  'Duplicate entry — created twice by mistake',
  'Customer requested cancellation before dispatch',
  'Employee has resigned, final settlement processed',
  '',
  'Restructuring — role eliminated',
]

let cache: ApprovalRow[] | null = null

function generate(): ApprovalRow[] {
  const rand = lcg(20260714 + 4)
  const rows: ApprovalRow[] = []
  const n = 60
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const monthIdx = Math.floor(rand() * 6) // recent — this is a live queue, not a year-deep archive
    const day = 1 + Math.floor(rand() * 27)
    const requestedAt = `2026-${pad(1 + (monthIdx % 6), 2)}-${pad(day, 2)}`
    const kind: ApprovalRow['kind'] = i % 3 === 0 ? 'archive' : 'delete'
    // Mostly pending (it's a queue), with a decided tail so the distribution
    // bar and filters have something to show beyond one bucket.
    const status = i % 97 === 0 ? 'UNKNOWN_STATUS' : r < 0.6 ? 'pending' : r < 0.8 ? 'approved' : 'rejected'

    rows.push({
      id: `apr-${i}`,
      kind,
      target: kind === 'delete' ? DELETE_TARGETS[i % DELETE_TARGETS.length]! : EMPLOYEE_TARGETS[i % EMPLOYEE_TARGETS.length]!,
      requestedBy: REQUESTERS[i % REQUESTERS.length]!,
      requestedAt,
      reason: REASONS[i % REASONS.length]!,
      status,
    })
  }
  return rows
}

async function mockFetch(): Promise<ApprovalRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

async function mockApprove(row: ApprovalRow): Promise<void> {
  cache ??= generate()
  const found = cache.find((x) => x.id === row.id)
  if (found) found.status = 'approved'
  await sleep(120)
}

async function mockReject(row: ApprovalRow, notes: string): Promise<void> {
  void notes // reviewer note — recorded server-side (review_notes), not surfaced as a column here
  cache ??= generate()
  const found = cache.find((x) => x.id === row.id)
  if (found) found.status = 'rejected'
  await sleep(120)
}

/* ---- real: INTEG-gapped entirely (admin-privileged, PII-bearing) — naming
 * the exact bindings for K5 ---- */

async function realFetch(): Promise<ApprovalRow[]> {
  throw new Error('INTEG gap: ListDeleteApprovalRequests + ListEmployeeArchiveRequests (merged) — wires at K5')
}

async function realApprove(_row: ApprovalRow): Promise<void> {
  void _row
  throw new Error('INTEG gap: ReviewDeleteApprovalRequest / ReviewEmployeeArchiveRequest (decision="approve") — wires at K5')
}

async function realReject(_row: ApprovalRow, _notes: string): Promise<void> {
  void _row
  void _notes
  throw new Error('INTEG gap: ReviewDeleteApprovalRequest / ReviewEmployeeArchiveRequest (decision="reject") — wires at K5')
}

/* ---- public switched API (descriptor imports THESE) ---- */

export const fetchApprovals = (): Promise<ApprovalRow[]> => pick(realFetch, mockFetch)()
export const approveApproval = (row: ApprovalRow): Promise<void> => pick(realApprove, mockApprove)(row)
export const rejectApproval = (row: ApprovalRow, notes: string): Promise<void> =>
  pick(realReject, mockReject)(row, notes)
