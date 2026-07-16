/* Notifications bridge — self-contained: types + mock + real + switch.
 * Merges two real streams the old NotificationsScreen.svelte reads: task-
 * assignment notices and delete/employee-archive approval review cards
 * (same two request kinds Approvals Queue lists — see bridge/approvals.ts).
 * Real bindings route through `$lib/api/collaboration`, a hand-written
 * wrapper whose transport wasn't confirmed as direct Wails IPC during recon
 * (see Notifications.parity.md); combined with the admin-privileged review
 * actions and employee-PII surface, the whole real side is INTEG-gapped for
 * K4, same posture as approvals.ts. */

import { pick } from './runtime'
import { goDate, str } from './map'
import { ListNotificationFeed, MarkNotificationAsRead, ReviewDeleteApprovalRequest, ReviewEmployeeArchiveRequest } from '$wails/go/main/App'
import type { Tone } from '../kernel/tones'

export interface NotificationRow {
  id: string
  kind: 'task' | 'delete-approval' | 'archive-approval'
  title: string
  subtitle: string
  /** YYYY-MM-DD — the day-grouping key. */
  date: string
  time: string
  read: boolean
  tone: Tone
  /** '' for plain task items; pending | approved | rejected for review cards. */
  reviewStatus: string
  requestedBy: string
  reason: string
  /** The UNDERLYING delete/archive request id (notification.source_id) — the id
   * the Review* bindings act on. Empty for plain task items. */
  sourceId: string
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

// Synthetic operator/employee names (SYNTHETIC_IDENTITY.md canon) — mock/test
// data, exempt territory.
const ASSIGNERS = ['Aisha Al-Rumaihi', 'Mohammed Bucheeri', 'Fatima Al-Zayani', 'Yusuf Kanoo', '']
const TASK_TITLES = [
  'New task assigned: Prepare Q3 costing sheet',
  'Task assigned: Review supplier invoice SI-2026-0093',
  'Task assigned: Follow up on RFQ-2026-0417',
  'Task reassigned to you',
  'Task due tomorrow: Finalize offer OFR-2026-0201',
]
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

let cache: NotificationRow[] | null = null

/** Days-ago -> YYYY-MM-DD against a fixed "today" so the mock stays
 * deterministic across runs (no wall-clock drift in the day grouping). */
const TODAY = new Date('2026-07-14T00:00:00Z')
function dayOffset(daysAgo: number): string {
  const d = new Date(TODAY)
  d.setUTCDate(d.getUTCDate() - daysAgo)
  return d.toISOString().slice(0, 10)
}

function generate(): NotificationRow[] {
  const rand = lcg(20260714 + 11)
  const rows: NotificationRow[] = []
  const n = 54
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const daysAgo = Math.floor(rand() * 9) // spread across the last 9 days
    const date = dayOffset(daysAgo)
    const time = `${pad(8 + Math.floor(rand() * 10), 2)}:${pad(Math.floor(rand() * 60), 2)}`
    const isReview = i % 3 === 0
    const read = !isReview && r > 0.45

    if (!isReview) {
      rows.push({
        id: `notif-${i}`,
        kind: 'task',
        title: TASK_TITLES[i % TASK_TITLES.length]!,
        subtitle: `Assigned by ${ASSIGNERS[i % ASSIGNERS.length]! || '(unknown)'}`,
        date,
        time,
        read,
        tone: read ? 'neutral' : 'info',
        reviewStatus: '',
        requestedBy: ASSIGNERS[i % ASSIGNERS.length]!,
        reason: '',
        sourceId: '',
      })
      continue
    }

    const archive = i % 9 === 0
    const decided = i % 97 === 0 ? 'UNKNOWN_STATUS' : r < 0.65 ? 'pending' : r < 0.85 ? 'approved' : 'rejected'
    const target = archive ? EMPLOYEE_TARGETS[i % EMPLOYEE_TARGETS.length]! : DELETE_TARGETS[i % DELETE_TARGETS.length]!
    rows.push({
      id: `notif-${i}`,
      kind: archive ? 'archive-approval' : 'delete-approval',
      title: archive ? `Employee archive request: ${target || '(unnamed)'}` : `Delete approval requested: ${target}`,
      subtitle: `Requested by ${ASSIGNERS[i % ASSIGNERS.length]! || '(unknown)'}`,
      date,
      time,
      read: decided !== 'pending',
      tone: archive ? 'warning' : 'info',
      reviewStatus: decided,
      requestedBy: ASSIGNERS[i % ASSIGNERS.length]!,
      reason: REASONS[i % REASONS.length]!,
      // Mock request id (source_id) the review card would act on.
      sourceId: `req-${i}`,
    })
  }
  return rows
}

async function mockFetch(): Promise<NotificationRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

async function mockMarkRead(id: string): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (row) row.read = true
  await sleep(100)
}

async function mockApprove(row: NotificationRow): Promise<void> {
  cache ??= generate()
  const found = cache.find((r) => r.id === row.id)
  if (found) {
    found.reviewStatus = 'approved'
    found.read = true
  }
  await sleep(120)
}

async function mockReject(row: NotificationRow, reason: string): Promise<void> {
  void reason // reviewer note recorded server-side, not surfaced as a column here
  cache ??= generate()
  const found = cache.find((r) => r.id === row.id)
  if (found) {
    found.reviewStatus = 'rejected'
    found.read = true
  }
  await sleep(120)
}

/* ---- real: INTEG-gapped entirely (unconfirmed transport, admin-privileged,
 * employee-PII surface — see file header) ---- */

/* notification_type vocabulary (collaboration_service.go / delete_approval /
 * employee_archive): task | project | delete_approval |
 * employee_archive_approval | document_expiry. Only the two approval kinds get
 * a review card; everything else renders as a plain feed item. */
const KIND_MAP: Record<string, NotificationRow['kind']> = {
  delete_approval: 'delete-approval',
  employee_archive_approval: 'archive-approval',
}

function mapNotification(raw: unknown): NotificationRow {
  const r = raw as Record<string, unknown>
  const kind = KIND_MAP[str(r.notification_type)] ?? 'task'
  const iso = str(r.created_at)
  const read = !!r.read_at || str(r.status).toLowerCase() === 'read'
  return {
    id: str(r.id),
    kind,
    title: str(r.title),
    subtitle: str(r.message),
    date: goDate(r.created_at),
    time: iso.length >= 16 ? iso.slice(11, 16) : '',
    read,
    tone: read ? 'neutral' : kind === 'archive-approval' ? 'warning' : 'info',
    // The notification record does not carry the approval decision, requester, or
    // reason — those live on the underlying delete/archive request. Honest blanks
    // here (still enriched from the request when that read lands). Live-push
    // (EventsOn) stays DEFER. source_id IS the request id the Review* bindings act on.
    reviewStatus: '',
    requestedBy: '',
    reason: '',
    sourceId: str(r.source_id),
  }
}

async function realFetch(): Promise<NotificationRow[]> {
  // ListNotificationFeed(limit, unreadOnly) — full feed (unreadOnly=false).
  const rows = await ListNotificationFeed(100, false)
  return (rows ?? []).map(mapNotification)
}

async function realMarkRead(id: string): Promise<void> {
  await MarkNotificationAsRead(id)
}

// WIRED (G3): the Review* bindings act on the underlying *request* id, which is
// the notification's source_id (delete_approval notifications set SourceID =
// request.ID, delete_approval_service.go; employee_archive likewise). The mapper
// now carries `sourceId`, so the review card acts on the correct record. The
// bindings take (requestID, decision, notes) — NO actor arg; the reviewer is
// derived server-side from the session (same shape as the Approvals Queue path,
// bridge/approvals.ts, and covered by the R2 persistence tests). A review card
// only ever renders for the two approval kinds, so `sourceId` is always present.
async function realApprove(row: NotificationRow): Promise<void> {
  if (row.kind === 'archive-approval') {
    await ReviewEmployeeArchiveRequest(row.sourceId, 'approve', '')
  } else {
    await ReviewDeleteApprovalRequest(row.sourceId, 'approve', '')
  }
}

async function realReject(row: NotificationRow, reason: string): Promise<void> {
  if (row.kind === 'archive-approval') {
    await ReviewEmployeeArchiveRequest(row.sourceId, 'reject', reason)
  } else {
    await ReviewDeleteApprovalRequest(row.sourceId, 'reject', reason)
  }
}

/* ---- public switched API (screen/viewmodel imports THESE) ---- */

export const fetchNotifications = (): Promise<NotificationRow[]> => pick(realFetch, mockFetch)()
export const markNotificationRead = (id: string): Promise<void> => pick(realMarkRead, mockMarkRead)(id)
export const approveNotification = (row: NotificationRow): Promise<void> => pick(realApprove, mockApprove)(row)
export const rejectNotification = (row: NotificationRow, reason: string): Promise<void> =>
  pick(realReject, mockReject)(row, reason)
