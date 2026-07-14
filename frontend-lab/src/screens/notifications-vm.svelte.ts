/* Notifications viewmodel — L5's reactive half. State + derivations only, no
 * rendering/layout (mirrors kernel/ledger.svelte.ts's shape for this
 * bespoke, non-ledger screen): load, day-group, unread filter, mark-read,
 * approve/reject. Mutations reload from the bridge afterward rather than
 * mutating rows in place, same posture as the ledger ActionHost's
 * run→reload cycle.
 *
 * Named `notifications-vm` (not `notifications.svelte.ts`) so its stem never
 * differs from `Notifications.svelte` by case only — that collides under
 * TypeScript's case-insensitive file resolution on Windows. */

import {
  approveNotification,
  fetchNotifications,
  markNotificationRead,
  rejectNotification,
  type NotificationRow,
} from '../bridge/notifications'
import { formatDate } from '$kernel/format'

export interface NotificationDayGroup {
  date: string
  label: string
  items: NotificationRow[]
}

/** Fixed "today" the mock generator anchors its day offsets against — keeps
 * Today/Yesterday labels stable across runs instead of drifting with the
 * wall clock. */
const MOCK_TODAY = '2026-07-14'

function dayLabel(date: string, today: string): string {
  if (date === today) return 'Today'
  const diffDays = Math.round((new Date(today).getTime() - new Date(date).getTime()) / 86_400_000)
  if (diffDays === 1) return 'Yesterday'
  return formatDate(date)
}

export class NotificationsViewModel {
  rows = $state<NotificationRow[]>([])
  loading = $state(true)
  error = $state<string | null>(null)
  unreadOnly = $state(false)

  visible = $derived.by(() => (this.unreadOnly ? this.rows.filter((r) => !r.read) : this.rows))

  grouped = $derived.by((): NotificationDayGroup[] => {
    const byDate = new Map<string, NotificationRow[]>()
    for (const row of this.visible) {
      if (!byDate.has(row.date)) byDate.set(row.date, [])
      byDate.get(row.date)!.push(row)
    }
    return [...byDate.entries()]
      .sort((a, b) => (a[0] < b[0] ? 1 : -1)) // newest day first
      .map(([date, items]) => ({ date, label: dayLabel(date, MOCK_TODAY), items }))
  })

  unreadCount = $derived.by(() => this.rows.filter((r) => !r.read).length)

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      this.rows = await fetchNotifications()
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  async markRead(row: NotificationRow): Promise<void> {
    if (row.read) return
    try {
      await markNotificationRead(row.id)
      await this.load()
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    }
  }

  async approve(row: NotificationRow): Promise<void> {
    try {
      await approveNotification(row)
      await this.load()
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    }
  }

  /** Thrown errors surface through FormViewModel's submitError (the caller
   * is FormModal's rejectForm.submit) — no local try/catch here. */
  async reject(row: NotificationRow, reason: string): Promise<void> {
    await rejectNotification(row, reason)
    await this.load()
  }
}
