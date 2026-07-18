# Parity Ledger — NotificationsScreen (old) vs Notifications (bespoke K4)

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Cross-device feed merging task-assignment items + delete/employee-archive review cards | **DONE (fetch INTEG-gapped)** | `bridge/notifications.ts` merges the same two review-request kinds Approvals Queue lists, plus plain task items. Old bindings route through a hand-written `$lib/api/collaboration` wrapper whose transport wasn't confirmed as direct Wails IPC during recon; combined with admin-privileged review actions and employee-PII, the whole real side is INTEG-gapped, same posture as `bridge/approvals.ts`. |
| 2 | Grouped by day | **DONE** | `NotificationsViewModel.grouped` buckets by `date` (YYYY-MM-DD) and labels Today/Yesterday/formatted date, newest day first. |
| 3 | Filter unread-only | **DONE** | `FilterChips` single "Unread (N)" chip drives `vm.unreadOnly`; the count itself (`vm.unreadCount`) is a live derivation over all rows, not just the visible slice, so it stays accurate while the filter is on. |
| 4 | Mark-read on click | **DONE (mock mutation, real INTEG-gapped)** | Plain unread task rows expose a nav-intent through `ActivityFeed`'s existing click slot (repurposed as "mark read" rather than navigation, since this screen has nowhere else to send that click); review cards don't mark-read via click — their status changes via Approve/Reject instead, matching the old screen's own behavior where a review card's "read" state was implied by the decision, not a separate click. |
| 5 | Delete-approval review card: approve/reject inline | **DONE** | Approve → `ConfirmDialog` (plain confirm, ActionHost's non-form escalation path). Reject → `FormModal` with a required reason textarea (ROW-AWARE FORMS pattern, identical shape to Approvals Queue's `rejectForm`). Both call through `bridge/notifications.ts`, INTEG-gapped for the same reasons as #1. |
| 6 | Employee-archive review card: approve/reject inline | **DONE** | Same action pair as #5; the row's `kind` (`'archive-approval'`) only changes the badge tone/label and the confirm-dialog copy, not the mechanism — matching Approvals Queue's own single merged action pair across delete/archive kinds. |
| 7 | Live `EventsOn("notifications:new"/"updated")` push updates | **DEFER** | Not modeled — this K4 build is poll-on-mount + reload-after-mutation (`vm.load()`), same posture as every other K4 screen. Live event wiring is an INTEG-era concern (needs the real Wails event bus), tracked here rather than faked with a `setInterval`. |
| 8 | Deep-linked from ApprovalsQueueScreen (arrive pre-scrolled to a specific notification) | **DEFER** | No `initialQuery`-style seeding on this viewmodel yet — out of scope for a first bespoke pass; would need a `NavIntent` query param (e.g. `{ highlightId }`) threaded through if picked up later. |
| 9 | Two-stage employee-archive approval (`RequiredApprovals`, first/second approver) | **DEFER** | Same call as Approvals Queue's own #6 — this screen treats archive requests as single-decision (pending→approved/rejected) per click, matching what the old screen's click-to-review flow actually does. |

## Reading

This screen deliberately reuses the Approvals Queue merge logic and INTEG
posture rather than re-deriving it — both request kinds, the review-decision
vocabulary (`pending`/`approved`/`rejected`, `approve`/`reject`), and the
reason-capture pattern are the same real backend surface viewed two ways
(a durable queue vs. a day-grouped feed). Nothing here claims a real fetch
or a real decision path; every mutation is mock-backed and every real
binding throws an honest INTEG-gap error naming the exact call it stands in
for.
