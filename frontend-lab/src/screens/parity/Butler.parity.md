# Butler — parity notes

**Entity:** `butler` · **Group:** System · **Archetype:** bespoke (K4 L-monster)

Old: `ButlerScreen.svelte` (2960 lines) — AI-assistant chat (Mistral-backed, optional).
New: `bridge/butler.ts` + `screens/butler-actions.ts` (domain-logic co-module, ~955 lines) +
`butler-vm.svelte.ts` + `Butler.svelte`. Two-pane console (conversation sidebar + chat panel)
using the new `ChatTranscript` primitive. **RETIRES IntelligenceHub** (nav `intelligence` →
`butler` at K5 — orchestrator-owned).

## Capability census

| # | Old behavior | Verdict | Notes |
|---|---|---|---|
| 1 | Conversation sidebar (New/Clear-All arm/list/delete-arm/quick-actions) | **DONE** | Stack/Row/Button/Badge composition; `isActive` surfaced as an Active/Idle Badge (new — exercises the stale-is_active mock case). |
| 2 | Message feed + hand-rolled markdown (headings/bullets/numbered/tables via `{@html}`) | **DONE (replaced)** | `ChatTranscript` primitive + kernel **escape-first** `renderMarkdown` — same visual vocabulary, security-ordered (escape-first, not escape-some-paths), dependency-free. Verified live against a huge multi-block response and a ragged table. |
| 3 | Action type/target alias normalization | **DONE (1:1)** | `butler-actions.ts`; dropped 3 duplicate/unreachable `resolveActionTarget` switch cases (dead code). |
| 4 | `validateActionPayload` (~180 lines) | **DONE (1:1)** | Same per-target rules + STATUS_CONSTRAINTS; drives chip status. |
| 5 | `getActionRuntimeState` (ready/needs_input/needs_approval/invalid_payload) | **DONE** | needs_approval/invalid_payload require the backend to stamp the action JSON (never client-computed) — mock embeds those keys on 2 seeded actions. |
| 6 | **Arm/confirm/6s-timeout hot-zone** (single global arm slot, key=JSON of type+target+label+data) | **DONE — preserved exactly, verified live** | Lives in the VM (`armedKey`/`arm`/`clearArm`); `ChatTranscript` renders `armedChipId` + forwards `onChipClick`. Live-confirmed: click-1 arms + posts preview, click-2 confirms + dispatches, 6s timeout auto-disarms. Content-addressed key = old `pendingActionKey` semantics. |
| 7 | navigate/analyze/fetch/clarify single-click; create/update/approve/reject two-click armed | **DONE** | `isWriteAction()` gates the arm requirement. |
| 8 | 23 write-action bindings | **INTEG (ONE seam)** | All routing/guards in `resolveCreate/Update/ApprovalAction`; the bridge's `executeButlerActionBinding(name)` is the single always-throwing seam (`INTEG gap: <name> — wires at K5`), called by the VM's one `executeButlerAction` dispatcher. Verified end-to-end for CreateOfferDraftFromButler. |
| 9 | `ChatWithButlerPersistent` 3-tier fallback | **COLLAPSED to ONE INTEG throw** | Per spec; the old `!window.go` canned-reply mock branch preserved as `mockSendMessage` so chat stays interactive in the lab. |
| 10 | DeleteConversation / PurgeAllConversations | **INTEG (mock functional)** | Standard convention: mock simulates so the delete-arm/clear-all-arm hot-zones are testable; real throws naming the binding. Verified live (delete removes exactly one). |
| 11 | ListConversations, GetConversationMessages, ListCustomers/GetCustomer, ListSuppliers/GetSupplier (5m TTL lookup cache) | **FETCH (real, wired)** | Real adapters call the Wails bindings; mock supplies the adversarial dataset. |
| 12 | **PRESERVED**: MarkOfferWon refuses if customer_po missing (never substitutes a literal) | **DONE** | `resolveApprovalAction` returns a refusal before reaching the bridge. |
| 13 | **PRESERVED**: stock-adjustment "update" supports ONLY approval; other statuses refused | **DONE** | Identical refusal message. |
| 14 | AI-authority boundary (Butler emits ButlerAction only; deterministic backend executes) | **DONE (structurally enforced)** | No code path where an armed+confirmed action silently succeeds client-side — every write funnels through the one throwing seam. |
| 15 | `insights`/`butler:event` feed | **DROP (dead code)** | Populated but never rendered in the old screen; not ported. |
| 16 | IntelligenceHub screen | **RETIRE** | Nav repoint is orchestrator-owned at K5. |
| 17 | Cross-screen `navigate`-type action | **DEFER** | No `navigate` callback exists for bespoke `ScreenEntry.component` (unlike Hub) — same gap class as SerialTrace. Surfaces as an informational chat message; wires at K5 app shell. |
| 18 | Adversarial mock | **DONE** | 18 conversations: 200-char/empty→Untitled/RTL titles, stale is_active, 42-msg scroll convo, empty convo, huge multi-block markdown, ragged table, 4-chip wrap, bare-string data chip, legacy-fields message, explicit needs_approval + invalid_payload chips, huge/negative/zero amounts. Seeded LCG, synthetic Gulf identity only. |

## Orchestrator kernel fixes (from Butler's flagged gaps)
- **`.k-grow`** utility added to kernel.css — a bare input beside a fixed button (the chat bar) now
  absorbs the row via `class="k-input k-grow"` (no screen layout CSS).
- **`Button` `min-width:0`** — buttons can now shrink+ellipsis in a tight flex Row (long conversation-title button).
- **`Row` `shrink={false}` prop** — a fixed trailing control cluster (badge + delete button) keeps its content
  width and refuses to shrink, so the flexible label sibling absorbs the squeeze instead of the cluster overflowing.
  These three propagate to every screen (full 41/41 gate re-run clean).

## Deferred to K5 (app shell)
- **No "fill remaining page height" chain** — `ChatTranscript`'s internal auto-scroll can't engage without a
  bounded parent height threaded from the viewport; the page scrolls instead. This is the viewport-height chain
  the real app shell owns. Chat is fully functional meanwhile. (Documented in Butler.svelte's header.)
- Cross-screen navigate hook for bespoke components (item 17).
- Minor: `ChatTranscript` type exports live in the instance `<script>`, not a `<script module>`; the VM uses
  structurally-identical local types. Candidate for a `kernel/chat.ts` extraction later (like line-items.ts/allocation.ts).
