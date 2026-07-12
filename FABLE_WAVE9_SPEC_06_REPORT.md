# Wave 9 Spec-06 Status Report — Tight Ship (Residue + Hardening Sweep)

**Branch:** `feat/fable-wave9-6-tight-ship` off `main` — **NOT merged/pushed/tagged**, left for owner review.
**Operating model:** Opus 4.8 orchestrator as senior designer/tech-lead — Phase-A recon work orders (6 parallel read-only agents) → constitutional review of **every** diff before commit → central gating that never trusts a subagent's build claim → personal ownership of the shared-file seams (bindings regen, App.svelte, the financial-integrity guard calibrations, the B7 token-convergence call). Sonnet 5 `general-purpose` subagents wrote all code in file-disjoint batches; the monster file (OrdersScreen) got one owner. The C1 bug hunt ran 6 domain finders; C2 ran 7 flow tracers; C3 ran one error sweep; a completeness-critic closed the loop — and earned its keep, catching a signature-drift regression my own Sa2 change had introduced into the Butler action executor (below).

## Commit map (branch `feat/fable-wave9-6-tight-ship`, off `main`, not merged)
- `4ae663a` — feat: B1–B7 residue + C1/C2 hardening fixes (Go product + frontend)
- `a923d7c` — test: B5 de-flake + B6 cache cleanup + C4 coverage
- `3f97f24` — chore: regenerate wailsjs bindings
- `fb560a2` — fix: schema golden for GRN `completed_at` + Butler binding-arity regression
- _(this commit)_ — docs: this status report

## Gates (on the final code commit `fb560a2`; this report is a docs-only commit on top)
- `npx svelte-check` ✅ **0 errors / 14 warnings** (= pre-wave baseline; net **0 new errors, 0 new warnings**)
- `npx vite build` ✅ · `go build ./...` ✅ · `go vet ./...` ✅
- `go test` — **GREEN, established in pieces (honest environment caveat).** This session's box could not sustain a single `go test ./...` to completion: the main package's runtime (~400s idle in Run A) balloons past the harness's ~10-min wall limit under load, and long background runs were reaped mid-flight. Green was therefore established as: **(1)** Run A — a full `./...` background run — passed **all 83 packages except** main's single `TestTradingModels_SchemaGolden`; **(2)** that golden was the *only* failure, caused by B3's new `goods_received_notes.completed_at` column — regenerated via the test's own `-update-schema-golden` flag (verified: **exactly one line changed, only that column**) and confirmed passing (1.2s); **(3)** a targeted run over the **entire Wave-9.6 Go change surface + neighbors + the golden** — GRN/PO-transition/DN+serials/invoice-downgrade/hardware-ID/cheque/DIT/FX/bank-recon/customer+supplier-merge/RFQ/offer, plus all 6 new C4 test files — passed (83.5s, exit 0); **(4)** the de-flaked `TestFileWatcher_HandlerError` passed **20/20 under concurrent CPU load** (B5). No Go code changed after Run A except the golden (testdata) and the Butler frontend fix (not exercised by `go test`), so Run A's other-package passes still hold. `ENCRYPTION_MASTER_KEY` no longer required (B4 landed).
- **B4 no-hang PROVEN:** the crypto/hardware tests ran **without** `ENCRYPTION_MASTER_KEY` and completed in ~14s (the old `wmic` path hung for minutes). The env-var caveat from Specs 01–05 is now **gone**.

---

## Phase A — ground-truth verdicts (every anchor re-verified)

| # | Question | Verdict |
|---|---|---|
| **A1** | Opportunity stage vocabularies | **CONFIRMED — bigger than the spec assumed, and it hits the financial tripwire.** FOUR overlapping vocabularies across three entities (`RFQData.Stage` legacy-9-stage, `Opportunity.Stage` canonical-7, `Offer.Stage` DB-CHECK-5) + a dormant Cap'n Proto enum. Validation = 5 mutually-inconsistent allowlists + 2 write paths with none (OneDrive import, SSOT importer). **Stage strings FEED FINANCIAL COMPUTATION** — `app_prediction_dashboard.go:395-551` branches on literal stage values for WinRate and PipelineValueBHD. Per B1's own guard clause, the historical-data migration is therefore **STOP-AND-REPORT**. Full mapping table below. |
| **A2** | LineItemsEditor order mode vs OrdersScreen | **CONFIRMED.** Editor block `OrdersScreen.svelte:1422-1511`; the shared component's `mode="order"` branch was **unconsumed** (zero blast radius to edit). Fields: product_code, description, quantity, unit_price_bhd, live total. All math in `sanitizeFormItems`/`recalculateFormItems`/`handleSubmit` (must stay). Live per-keystroke total via `oninput`. Component confirmed math-free. |
| **A3** | GRN completion signal | **CONFIRMED.** `GoodsReceivedNote` `pkg/crm/domain.go:823` has no completion column; `is_completed` derived from a StockMovement's existence (`grnHasPostedMovement` `grn_service.go:37`). All-rejected GRN posts no movement → permanent blind spot + latent re-complete double-count. Guard in `CompleteGRN` `grn_service.go:467-575` (row-lock :506). Migration seam = `ensureCrossModuleSchemaExtensions` (mature-DB path). **NB: GRNScreen is owner-deprecated/unreachable** ("per user request"). |
| **A4** | getHardwareID / wmic | **CONFIRMED + empirical finding.** `settings_service.go:199`; unbounded `wmic baseboard get serialnumber`, no memo. 3 consumers incl. `field_crypto.go:63` (PBKDF2 password — byte-identity CRITICAL). go-ole present but `exec.CommandContext` chosen (COM can't be context-cancelled). **Empirically BOTH `wmic` AND `Get-CimInstance` time out on this box (winmgmt wedged)** → the timeout, not the method choice, is the load-bearing fix; byte-identity holds on machines where the query succeeds; the guard test skips here. |
| **A5** | Cache/flake/tokens | **CONFIRMED.** 27 `NewCache()` sites in `cache_test.go` + ~11 ad-hoc test apps leak; `TestFileWatcher_HandlerError` used a fixed 200ms sleep; 6 undefined tokens (`--text-danger` #b42318, `--text-danger-strong` #912018, `--text-danger-deep` #7f1d1d, `--text-success` #027a48, `--text-neutral` #344054, `--surface-warning` #F59E0B). `--text-danger` is used with 4 different fallback hexes across sites (convergence flagged). |
| **A6** | C2 live-feasibility | **STATIC TRACING (spec's fallback).** No practical real-backend live-walk: Wails uses a native webview not headlessly drivable; the app's own boot calls `getHardwareID`→wmic (hangs pre-B4); and the repo's own frontend integration tests are Playwright-against-mocks (bridge stubbed), which don't exercise the real Go→persistence chain. C2 ran as static chain-of-custody tracing — the same method the five prior waves used. |

### B1 stage mapping table (for the owner — the migration is STOP-AND-REPORT)
| Legacy / ad-hoc string | → canonical | display bucket | confidence |
|---|---|---|---|
| `RFQ Received`, `Costing`, `Tender` | New / Proposal | Pipeline | high |
| `Offer Sent` | Quoted | Quoted | high |
| `Follow-up/Eval` | Quoted | Quoted | needs ratification |
| `Closed (Payment)` | Won | Won | high |
| `PO/LOI Received`, `Order Placed`, `In Process`, `Delivered` | Won | Won | **needs owner ratification** (collapses 4 post-sale stages; fulfillment already lives on `Order`) |
| `Closed (Lost)` | Lost | Lost | high |

---

## Phase B — per-item status

| Item | Status | Notes |
|---|---|---|
| **B1** stage consolidation | **SPLIT — migration STOP-AND-REPORT; safe half shipped** | A1 proved stage strings feed WinRate/PipelineValueBHD, so per B1's own guard the historical-data migration + backend enum enforcement are escalated to the owner (they'd silently move numbers stakeholders may have seen). Shipped the **presentation-only** half: `displayStage()` now maps every legacy/ad-hoc string to its canonical display bucket, so **no reachable card renders a raw legacy string** and legacy-staged cards are filterable under the right tab. Zero backend/data/financial change. Mapping table above. |
| **B2** Orders → LineItemsEditor | **shipped** | OrdersScreen consumes `mode="order"`; product_code/description/qty/unit_price + live per-keystroke total + validation + the `sanitizeFormItems` reconciliation heuristic all preserved; all math stays in OrdersScreen; costing mode byte-for-byte unchanged; component still contains **zero** calculation logic. Minor intentional convergence (the shared bottom Add button + item counter replace the old top Add button). |
| **B3** GRN completion flag | **shipped (backend)** | `CompletedAt *time.Time` on `GoodsReceivedNote`; set inside the existing locked transaction; guard now `alreadyApplied || CompletedAt != nil` (closes the all-rejected double-count); `is_completed` now derives from the flag; idempotent backfill from StockMovement (mature-DB path). The row-lock idempotency guard from 9.5 **stays**. NB: GRNScreen is owner-deprecated, so the flag serves backend correctness + the new C4 tests; its UI value awaits GRN's fate (owner Q). |
| **B4** retire wmic | **shipped — no-hang PROVEN** | Memoized resolution: CIM primary (6s ctx) → wmic byte-identical fallback (3s ctx) → `os.Hostname()` **unchanged**. Diagnostics wmic sites (memory, CPU, RAM) timeout-bounded. Guard test asserts identity when wmic answers (skips on this wedged box) + a no-hang test. `exec.CommandContext` (not COM) so a hung subprocess is killable. |
| **B5** de-flake filewatcher | **shipped** | 200ms fixed sleep → poll-until-`WatchStatusFailed` (3s deadline, ~20ms cadence); every assertion unchanged. |
| **B6** cache cleanup | **shipped** | `t.Cleanup(cache.Stop)` at 27 `cache_test.go` sites + `setupFullTestApp` + the mid-test cache replacement + `deployment_audit_test.go` + 10 ad-hoc `manual_*`/service test apps. No test-created cache goroutine outlives its test. |
| **B7** define tokens | **shipped** | The 6 tokens defined in `theme.css` at their fallback values. SettingsScreen hex retired **pixel-identical**. `--text-danger` **converges** 4 divergent danger-reds (#dc2626/#EF4444/#991b1b) onto the canonical #b42318 at FinancialDashboard/ApprovalsQueue/CustomerDetailView/SupplierDetailView — a sub-perceptible shift, a deliberate Article VI "one engine" convergence, flagged for sensory-wave ratification. `--danger`/`--success`/`--warning` untouched. |

---

## Phase C

### C1 — Adversarial bug hunt (triage ledger)
6 domain finders + the C2 tracers + a completeness critic. **Every find, fixed or not, is listed.** The fix-in-wave set converged; the residual backlog is financial-semantics / schema / owner-policy / the GRN deprecation — handed to a recommended successor wave.

**FIXED IN-WAVE (hardening):**
| Find | Domain | Fix |
|---|---|---|
| Wails v2 drops every 3-return bound method → 4 finance screens silently broken (Cheque Outstanding tab always empty, BookBank DIT/cheque pre-fill always zero, FX Exposure always empty, FX Revalue-All toast always "0 accounts") | recon | **Rec1** — 7 methods + 1 bonus repackaged as `(*Result, error)` structs; all Go/frontend callers updated; bindings regen. Non-financial (the sacred FX math is byte-for-byte untouched — only its return packaging changed). |
| Customer-invoice Edit could revert a posted (Sent) invoice to Draft + rewrite its line items (customer-side analog of the mandatory Wave 9.2 supplier fix; the intended guard `isCustomerInvoicePostedStatus` was dead code) | AR/AP | **AR1** — backend guard blocks posted→Draft/Proforma (orchestrator-narrowed so legitimate Sent→Cancelled/Void still works) + item-guard reads original status + a Go test; Edit dropdown restricted to legal transitions. |
| Supplier edit-save HARD-FAILS for any supplier with brands/product-types (JS array sent to a string column → `json.Unmarshal` error → every save rejected); create path stored CSV verbatim so brands never displayed | shell/CRM | **Sh1** — payload JSON-encodes the arrays; create path CSV→JSON. |
| Customer edit-save silently ZEROED the credit limit on every save (unconditional merge of a field the form can't populate) | shell/CRM | **Sh2** — `CreditLimitBHD` zero-guard (mirrors the supplier Rating precedent). |
| Customer `mobile_number` clobbered + supplier `Notes` wiped on every edit-save (same class as Sh2, surfaced by C2) | shell/CRM | **guards** in `MergeCustomerUpdate`/`MergeSupplierUpdate`. |
| AccountingScreen "Generate P&L / Balance Sheet" discarded the computed report (toast only) | recon | **Rec2** — captures + renders the statement (mirrors SettingsScreen). |
| ReportsScreen date-range picker decorative (always "month"); export offered PDF/Excel while backend implements CSV only → every export failed | recon | **Rec3** removed the decorative pickers; export now offers **CSV** (PDF/Excel disabled "coming soon"). |
| Customer phone/email/vat dead inputs (never seeded, never saved) | shell/CRM | **Sh3** — removed. |
| Dead PO "GRN" button dispatched an event with no listener; DN post-confirm order-progression failures swallowed; dead DN "Draft" tab | inventory | **Inv1/Inv4/Inv7** — button removed; post-confirm failures now surface a warning toast (DN stays confirmed); dead tab removed. |
| Reactivating an archived employee silently didn't restore revoked access/memberships | people | **PP2** — persistent warning toast naming what needs manual re-grant. |
| Manual RFQ create discarded the user's product lines (costing then seeded blank) | sales | **Sa2** — products serialized into `ProductDetails`, threaded through `CreateRFQ`. |
| `MatchedInvoiceIDs` written in two formats; dead Delete-account button; supplier-invoice dead imports; accounting/reports missing from persistentScreenIDs | recon/AR/shell | **Rec4/Rec5/AR3/Sh4** — consistency + dead-code + state-persistence. |
| **Butler "create opportunity" called `CreateRFQ` with 4 positional args after Sa2 made it 5** (Butler uses `invokeAppBridge` directly, bypassing the typed wrapper — Wails validates arity, so it would 500). A self-inflicted regression the completeness critic caught. | butler | Pass the 5th `productDetails` arg (`""`). |
| **Butler "mark offer won" passed the literal `"Butler"` as `MarkOfferWon`'s 2nd param — which is `customerPO`** (required, persisted onto the Order, part of its idempotency key), poisoning the data and colliding repeat wins. | butler | Refuse with "I need the customer PO number…" unless the action payload carries a real PO (matches the file's existing missing-field pattern). |

**REPORT-ONLY (financial-semantics / schema / policy / owner-decision — NOT changed this wave):**
- **Stock adjustments double-applied** — `CreateStockAdjustment` AND `ApproveStockAdjustment` both post a StockMovement (reachable via Butler). Doubles every approved adjustment. *Financial — stop-and-report; owner must decide which call owns the movement.*
- **GRN receiving deprecated/unreachable** ("per user request") — and it **severs the only stock-write path** (`reconcileInventoryReceipt`) **and the only serial-mint path** (`assignSerialsToGRN`). Consequence: PO "Received" is a cosmetic status flip, there is no reachable on-hand register, and the serial-trace well runs dry going forward. *Product decision — see owner questions.*
- **`MarkOfferWon`/`MarkOfferLost` update `RFQData.Status` but not `.Stage`** — the stale Stage overrides the correctly-updated Opportunity in the merged pipeline view, deflating win-rate. *Bundles with the B1 stage-vocab escalation (win-rate is financial).*
- **Expense self-approval** — no segregation-of-duties check (the supplier-invoice flow has `CreatedBy != approver`; expense doesn't). *Financial control policy — owner decision.*
- **Payroll comp-profile is per-employee-unique but the UI models it per-company** — switching companies + save can clobber the other division's profile. *Schema/financial.*
- **`Number(vat_percent) || 10`** treats a genuine 0% (zero-rated/export) invoice as 10%. *Tax math — stop-and-report.*
- **Order manual-create = 2 non-atomic calls** (ghost itemless-order risk); **DeliveryTracking screen orphaned** (swapped `CreateShipment` args + nil-deref); **GRN discrepancy CRUD** unimplemented placeholder; **employee-archive "pending approval" review** is dead code (self-approves); **WorkHub project-admin buttons** render for all roles (backend safely rejects, but the UI should hide them). *All reported for a successor wave.*

### C2 — E2E flow verification (the headline artifact): **26 🟢 · 11 🟡 · 2 🔴** (audit was 9🟢 / 16🟡 / 14🔴)
| Domain | Flow | 2026-07-10 | TODAY | Break point (if not 🟢) |
|---|---|---|---|---|
| Sales | RFQ/opportunity intake | 🟡 | 🟡 | stage-vocab escalated (display fixed) |
| Sales | Costing create & revise | 🟢 | 🟢 | — |
| Sales | Offer create/revise/send/won/lost | 🟢 | 🟢 | — |
| Sales | Offer→Order conversion | 🟢 | 🟢 | — |
| Sales | Order lifecycle & handoffs | 🟢 | 🟢 | — |
| Inventory | PO creation & lifecycle | 🟡 | 🟡 | "Received" now cosmetic (GRN gone) |
| Inventory | Goods receipt (PO→GRN→stock) | 🟢 | 🔴 | **owner-deprecated / unreachable** |
| Inventory | Stock visibility | 🔴 | 🔴 | InventoryScreen removed; no on-hand register |
| Inventory | Delivery note from order | 🟡 | 🟢 | ⬆ POD + warning toast + dead tab removed |
| Inventory | Serial trace | 🟢 | 🟢 | (well runs dry as GRN stays gone) |
| People | Employee onboarding | 🔴 | 🟢 | ⬆⬆ |
| People | Profile & HR admin | 🔴 | 🟢 | ⬆⬆ |
| People | Archive / deactivation | 🟡 | 🟡 | reactivate = manual re-grant (PP2 warns) |
| People | Payroll run | 🔴 | 🟡 | ⬆ (per-company profile risk) |
| People | App users/roles vs employees | 🔴 | 🟢 | ⬆⬆ |
| Work | Project creation & setup | 🔴 | 🟢 | ⬆⬆ |
| Work | Allocating people/work | 🔴 | 🟢 | ⬆⬆ |
| Work | Day-to-day tracking | 🔴 | 🟢 | ⬆⬆ |
| Work | Project administration | 🟡 | 🟡 | admin buttons not role-hidden |
| Work | Work/approvals routing | 🔴 | 🟢 | ⬆⬆ |
| Fin AR/AP | Customer invoice lifecycle | 🟡 | 🟡 | AR1 fixed; ‖10 VAT residual |
| Fin AR/AP | Customer receipt & allocation | 🟡 | 🟢 | ⬆ |
| Fin AR/AP | Supplier invoice→match→approve→pay | 🟡 | 🟢 | ⬆ |
| Fin AR/AP | Supplier payment recording | 🔴 | 🟢 | ⬆⬆ |
| Fin AR/AP | Expense entry | 🔴 | 🟡 | ⬆ (self-approval SoD gap) |
| Fin AR/AP | FinanceHub tab IA | 🟡 | 🟢 | ⬆ |
| Fin recon | Bank reconciliation (match) | 🟡 | 🟡 | (unchanged; solid) |
| Fin recon | Book-vs-bank (prove) | 🔴 | 🟢 | ⬆⬆ Rec1 un-broke DIT/cheque pre-fill |
| Fin recon | Cheque register lifecycle | 🔴 | 🟢 | ⬆⬆ Rec1 un-broke Outstanding tab |
| Fin recon | FX revaluation run | 🟡 | 🟢 | ⬆ Rec1 un-broke exposure + toast |
| Fin recon | Accounting / statements | 🟡 | 🟢 | ⬆ Rec2 displays P&L/BS |
| Fin recon | FinancialDashboard | 🟢 | 🟢 | — |
| Fin recon | Reports + audit trail | 🔴 | 🟡 | ⬆ export now works (CSV) |
| CRM/shell | Global navigation IA | 🟡 | 🟢 | ⬆ |
| CRM/shell | Dashboard morning-start | 🟡 | 🟢 | ⬆ |
| CRM/shell | Customer 360 → act | 🟢 | 🟢 | — |
| CRM/shell | Supplier 360 | 🟡 | 🟡 | notes-clobber fixed; still no drill |
| CRM/shell | Data-quality review | 🟢 | 🟢 | — |
| CRM/shell | Notifications / Inbox | 🟡 | 🟡 | (Inbox retired; solid) |

The 2 remaining 🔴 are both the GRN deprecation's shadow (unreachable goods-receipt + no stock register). Neither is a hardening fix — re-enabling goods receipt is a product decision (a new feature domain, out of this wave's bounds). Reported honestly: an accurate red beats a false green.

### C3 — Swallowed-error sweep (frontend): **the main app is clean.**
337 catch blocks inventoried. Virtually every user-initiated save/create/update/delete/post already routes failure through `toast.danger`. The only must-surface instance (`FocusCard.svelte` optimistic add/toggle/delete with console-only catch) is in **orphaned/dead code** (zero importers) — no live fix needed. Two empty catches on optional loads (`QuotationScreen:99`, `UserManagementScreen:107`) are low-priority hygiene. **No must-surface fix was required in the live app** — a clean bill.

### C4 — Targeted test hardening (Go): 4 new files, 11 tests, all green.
- `grn_completion_test.go` — all-rejected GRN completes once + sets `CompletedAt` + no double-count; accepted-qty GRN posts exactly one movement + idempotent; `is_completed` derives from the flag.
- `po_transition_test.go` — 14-case transition table (legal accepted incl. canonical spaced strings, illegal + unspaced rejected) + approval-threshold gate.
- `delivery_note_wave9_test.go` — `CreateDNWithSerials` happy path + failure-cleanup (orphaned DN removed, serials released); full-delivery progresses the order (accounts for the new `(string, error)` signature).
- `order_delivery_status_batch_test.go` — shape/empty/missing-order for `GetOrderDeliveryStatusBatch`.
Plus the AR1 downgrade-guard test. (One observed non-blocking behavior logged for visibility: `CreateDNWithSerials` cleanup soft-deletes the DN but leaves `DeliveryNoteItem` rows orphaned — noted, out of C4's test-only scope.)

---

## Decisions taken (orchestrator)
1. **B1 migration is stop-and-report** per the spec's own financial tripwire (A1 §6 proved stage feeds WinRate/PipelineValueBHD). Shipped only the safe, presentation-only display-mapping half; the backend enum + historical migration await owner sign-off with the mapping table above.
2. **B4 uses `exec.CommandContext`, not go-ole COM.** COM calls into a wedged `winmgmt` can't be context-cancelled and would reintroduce the hang; a killable subprocess with a bounded timeout is the real fix. Empirically both wmic and CIM hang on this box, so the timeout — not the method — is load-bearing.
3. **AR1 guard narrowed to Draft/Proforma.** The coder's first cut used the full closed-workflow set (which includes Cancelled/Void); I narrowed it so only the invoice-re-opening downgrades are blocked and legitimate Sent→Cancelled/Void still works.
4. **The Rec1 Wails-binding fix is a wiring fix, not a financial change** — the FX revaluation math is byte-for-byte identical; only the 3-value return was repackaged into a struct (the 2-value return the Wails bridge marshals correctly).
5. **Bindings regenerated centrally** (`wails generate module`); the dist placeholder restored after every build.

## Constitution deviations
- **Article VI (B7 token convergence, flagged).** Defining the single `--text-danger` collapses 4 pre-existing divergent danger-reds onto #b42318 — a sub-perceptible shift at 3-4 non-Settings sites. This is the correct "one engine" outcome but it changes rendered pixels slightly; **flagged for sensory-wave ratification** rather than improvised silently.
- **B1 migration deferred as stop-and-report** (financial semantics) — an Article VII deviation from the literal "ship the migration" AC, taken because the spec's own guard mandates it. Owner sign-off needed.
- **No rounding/posting/tax/VAT/payment math changed anywhere.** Every financial-adjacent find was routed to report-only.

## Keep-list attestation (audit §4, all domains)
- **Sales:** pending-store handoffs, status-driven CTAs, the costing calculator + suggested-vs-override pricing (verified intact through the B2 extraction — the component is math-free), revision model, MarkOfferWon PO capture, won/lost edit-locking, order traceability. ✅
- **Inventory:** Order→DN store handoff + no-items guard, GRN remaining-qty defaulting (backend intact), serial-trace read-only search + warranty coloring. ✅ (Goods-receipt UI is owner-deprecated, not regressed by this wave.)
- **People/Work:** payroll state machine untouched; archive safety contract; race guards; ContextTaskModal prefill; ApprovalsQueue. ✅
- **Finance AR/AP:** gated Match→Approve→Settle chain (strengthened — AR1 closes the customer-side analog of the supplier bypass); apply-receipt ergonomics; confirm-twice + posted/paid locks; `matchesCompany` scoping. ✅
- **Recon/acct:** bank-recon Finalize/Reopen gating; server-resolved identity; FinancialDashboard drill-throughs; cheque next-number preview. ✅ (Rec1 restored the data these screens display; no gating changed.)
- **CRM/shell:** the one-source navItems (verified clean); customer-360 drills; guarded soft-delete block reasons; role-adaptive dashboard. ✅

## Known residue / follow-ups (recommended Spec-07)
- The full **report-only backlog** above (stock-adjustment double-post, expense SoD, payroll per-company profile, ‖10 VAT, order-create atomicity, DeliveryTracking orphan, GRN discrepancy stub, employee-archive dead approval, WorkHub role-gate, MarkOfferWon RFQ.Stage).
- **The B1 backend stage-vocabulary consolidation + historical migration** (financial — owner ratification of the mapping first).
- **The GRN deprecation's systemic consequence** (no stock-write, no serial-mint, no on-hand register) — a product decision.
- `CreateDNWithSerials` cleanup leaves orphaned `DeliveryNoteItem` rows (C4 observation).
- The main test package exceeds Go's default 600s per-package timeout under load (pre-existing; use `-timeout 1800s`).

## Open questions for the owner
1. **B1 stage migration** — ratify the mapping table (especially the 4 post-sale stages → Won) so the backend enum + idempotent historical migration can ship next wave? It will move WinRate/PipelineValueBHD for mis-tagged historical rows (that's the whole point) — confirm you accept the number movement.
2. **GRN / goods receipt** — it's deprecated "per user request," which currently severs the only stock-write and serial-mint paths and leaves PO "Received" cosmetic. Intended? Or should a lean receiving path be restored (product decision)?
3. **B7 token convergence** — accept `--text-danger` = #b42318 across all sites (sub-perceptible shift at a few), or reserve the final red for the Sensory & Brand wave?
4. **Stock-adjustment double-post** and **expense self-approval** — both are live financial-control issues needing your authorization to fix (which write owns the movement; whether expenses need SoD).
5. **Fast-track vs Spec-07** — the report-only backlog is sizeable but well-characterized; schedule it as the successor "tight ship 2" wave?

---

*Definition of done: Phase A recorded ✅ · B1–B7 shipped or explicitly split-and-escalated with reason ✅ · C1 hunted (6 finders + 7 tracers + completeness critic) with the full triage ledger ✅ · C2's 39-flow table complete, every 🔴 fixed or reasoned ✅ · C3 classified (main app clean) ✅ · C4 tests in ✅ · gates green on the final commit ✅. Branch left local for owner review — no merge, no push, no tag.*
