# A6 — Toast Census (feeds B6)

## Toast API
`frontend/src/lib/stores/toasts.ts` — Svelte writable store. API: `toast.success/warning/danger/info(msg, duration?)` (alias `toasts.*`), `.dismiss(id)`, `.clear()`. Default 4000ms; `duration:0` = persists until dismissed. Rendered via `ToastContainer.svelte` → `WabiSabiToast.svelte`. No `.error()` — `danger` IS the error variant (persists/until-dismiss pattern is via duration:0 where used).

## Totals
826 `toast.*` sites / 63 files. danger 336, warning 232, success 231, info 27. 7 are dev-only in `ToastTestButton.svelte`. Effective production ≈ 819.

## Classification
- **CONFIRM (keep): ~814** — danger/warning are near-universally validation errors / "Failed to X" (direct response to user's click); success = "X created/saved/updated".
- **ANNOUNCE (Article IV.4 violations): 4**
  1. `App.svelte:91` — `toast.info(\`${label}: ${event.FileName}\`)` — file-watcher/background import event, no user click.
  2. `App.svelte:627` — `toast.warning("Session expired — please sign in again.")` — background auth expiry.
  3. `lib/components/OpportunityDetail.svelte:342` — `toast.warning("This opportunity changed on another device…")` — background conflict sync.
  4. `lib/screens/WorkHub.svelte:972` — `toast.warning("Showing cached task details while sync catches up.")` — background sync state.
- **DUPLICATE (toast + inline for same event): 1**
  - `lib/screens/BankReconciliationScreen.svelte:412-413` — `toast.success('Statement reconciled successfully')` then `showHandoffBanner=true` renders persistent inline `.handoff-banner` (line 1060-1063). Drop the toast, keep the inline banner (it has the CTA).

Borderline (NOT flagged, note only): `CostingSheetScreen.svelte:1232` info("Costing draft saved on this device…") pairs with a persistent "Saved Draft" recovery panel on next visit — not simultaneous, left as CONFIRM.

## B6 fix list
1. `App.svelte:91` — background import announce → remove (or route to notifications/digest if one exists).
2. `App.svelte:627` — session-expired: NOT noise — it's important, but it's an announce. Route to the lock/login redirect surface it already triggers; keep the user informed there, drop the toast ONLY if the redirect already conveys it. Coder decides + reports; do not silently strand info.
3. `OpportunityDetail.svelte:342` — cross-device conflict → route to the approvals/notification surface (Article V), not a toast; if no surface, keep + flag.
4. `WorkHub.svelte:972` — cached-while-sync → remove (pure background state).
5. `BankReconciliationScreen.svelte:412` — delete the toast, keep inline banner.
6. Delete `frontend/src/lib/components/ToastTestButton.svelte` (7 dev-only sites) — verify it is not imported by any shipping screen first.

## AC
Every remaining toast = direct echo of a user action; zero announce-class. Success toasts get B2 motion; error/danger toasts keep persist-until-dismiss where they already use duration:0.
