# Fable Wave 6 — Decision Log ("Empty the God's Pockets")

Wave 6 works the two work-lists the Wave 5 map handed over (butler read
paths, the deletion Executor dispatch), lands the Commander-decided
payroll refusal, and wires the logout UI. Decisions are numbered W6-D1…;
**[Mirror]** paragraphs record what generalizes.

## W6-D1. Payroll negative net: refuse the whole run, name the employees

The Wave 5 residue pinned (but did not change) this behavior: when an
employee's deductions exceed gross, the item's net clamps to 0 while the
accrual journal still debits FULL gross — so debits ≠ credits and the
journal does not balance. The Commander decided the fix at handoff:
**refuse to generate**. Implementation decisions:

- The check lives in `payroll.GenerateRun`'s item loop, and the refusal
  is WHOLE-RUN: every offending employee is collected first, then one
  error names each employee with the exact deduction and gross amounts
  ("correct the compensation profile(s) and regenerate"). No partial
  generation, no silent skipping — the same philosophy as the W4-D6
  rounding guard: refuse loudly at the moment of issuance, never adjust.
- The refusal happens BEFORE the transaction opens, so a refused
  generation persists nothing (both the pkg test and the root golden
  assert zero run rows).
- The Wave 5 golden `TestPayrollRunGeneration_NegativeNetClampsToZero`
  pinned the OLD clamp behavior. It is now
  `TestPayrollRunGeneration_NegativeNetRefused`, pinning the refusal —
  a deliberate, Commander-authorized golden change, stated in the commit
  message because a silently edited golden is worse than none. The new
  golden seeds a healthy employee alongside the offender to prove the
  refusal is whole-run.
- The balanced-journal goldens (`TestPayrollPostingJournal_GoldenNumbers`,
  `TestPayrollRunGeneration_GoldenTotals`) pass byte-identical — they
  never exercised the imbalance.
- Capping and receivable-booking were considered by the Commander and
  NOT chosen; scope ends exactly at the refusal.

**[Mirror-D1]** When a clamp exists only to keep an aggregate presentable
(net ≥ 0) while a downstream ledger consumes the UNclamped inputs (full
gross on the debit side), the clamp is not a safety feature — it is the
mechanism that unbalances the books. The honest fixes are: propagate the
truth (book a receivable), or refuse the input. Silently clamping one
side of a double-entry is never one of them.

## W6-D2. Butler context peel: pkg/butler/context, RBAC computed at the host, six-hole HostPort

`butler_ai_context.go` (5,070 LOC, 57 App methods) was the W4-D9 map's
"cheapest untouched seam" and the ground agreed: measured coupling was
233x `a.db`, exactly TWO `requirePermission` calls (both computing the
same `hasFinanceAccess` at the two entry points), one call into finance
reporting, and reads of five root-owned models. The context builders,
entity resolvers, period/year summaries, and the finance redaction rules
moved to **pkg/butler/context** (~4,540 LOC of logic). Boundary calls:

- **RBAC stays at the host chokepoint**: `BuildIntentContext` /
  `BuildFullContext` take `hasFinanceAccess bool`; the root delegates
  compute it via requirePermission. The redaction LOGIC (what gets
  removed without finance access) moved in — it is context policy, not
  auth policy — and a pkg test pins that a false flag redacts revenue.
- **HostPort (six methods)** carries what would otherwise drag hubs in:
  work/task context, employee resolution + context, and quick captures
  (collaboration-hub models: Employee, TaskItem, TaskComment,
  Notification, QuickCapture), the cashflow projection (finance
  reporting surface), and `OpenDedupedOpportunities` — the open-pipeline
  dedup rides on the OneDrive folder-meta normalization helpers
  (`normalizeOpportunityForList` & co.), which are sales-pipeline
  territory this wave (A.3 says ports, not relocation), so the host
  returns the deduped set and the forecast math moved in.
- Models were already pkg-owned (crm/finance/infra aliases in root) —
  the pkg file re-aliases them locally to keep the moved bodies
  byte-close to the originals (the W5-D1 lesson: when models already
  moved, the peel needs no model ports).
- The pure query parsers and small helpers (`ParseYearWindowFromQuery`,
  `ParseQuarterWindowFromQuery`, `FirstNonEmpty`, `Round3`, …) are
  exported from the package; root keeps same-named unexported wrappers
  so the ~15 other root call sites are untouched. NOTE: this package's
  `FirstNonEmpty` returns the UNTRIMMED value — kernel
  `text.FirstNonEmpty` trims — so the two were deliberately NOT merged.
- Proof: full root suite (incl. butler intent/usability/prompt-harness
  and workflow regression tests) passes untouched; new pkg tests cover
  BusinessSummary, customer resolution (exact/fuzzy/ambiguous/empty),
  HostPort wiring + redaction, and the pure parsers.

**[Mirror-D2]** When an entry point computes a permission ONCE and
threads it through fifty builders as a bool, the honest peel signature
is that bool as a parameter — moving the permission CHECK inward would
couple the package to the auth hub for one line, and re-checking per
builder would multiply an RBAC query by fifty. Redaction-on-false is
domain policy and moves with the domain; deciding the flag is host
policy and stays at the chokepoint.

## W6-D3. Grounded fastpath: builders move, orchestrators stay

`butler_grounded_fastpath.go` (1,682 LOC) splits along a line the file
itself already drew: the `try*` methods are ORCHESTRATORS (intent
gating, hint inference, and — in the task-creation path — a hub write
via `CreateCollaborativeTask`), while the `build*Response` cluster is
pure read (only `a.db` on customers/suppliers/invoices/offers/notes).
The 13 pure builders (~720 LOC) moved to
`pkg/butler/context/grounded_responses.go` as Service methods; the nine
`try*` orchestrators stay root and call them through
`a.butlerContextService()` — no delegate wrappers, because every call
site was inside the file itself. Boundary calls:

- `buildEmployeeTaskOverviewResponse` sits IN THE MIDDLE of the moved
  block by line number but reads the collaboration hub's `TaskItem`
  model — it stays root. W5-D3 again: the boundary cuts mid-file, and
  here mid-cluster.
- `tryGroundedOfferDraftFastPath` also sits inside the moved line range
  but is an orchestrator (calls the fastpath service for a DRAFT — the
  AI-authority boundary's allowed verb) — stays root.
- An earlier partial migration (`pkg/butler/fastpath`, map-based
  `DatabasePort`) already fronts these paths: root `try*` methods call
  pkg fastpath FIRST and fall back to the root logic. Moving the
  builders into THAT package would have forced a typed-GORM → map-port
  rewrite, violating behavior-identity. They went to
  `pkg/butler/context` instead, which already holds the typed-GORM read
  surface and the customer/supplier resolvers the builders call.
- The four small helpers split by usership: `appendIfPresent`,
  `uniqueStrings`, `joinOrNone` had no remaining root users and moved;
  `minInt` is still used by the staying task-overview builder, so root
  keeps it and the package carries its own copy. A shared helper is not
  worth a cross-layer import.

**[Mirror-D3]** When a partial migration already exists (a pkg twin with
a different port style), the peel must choose between the twin's style
and the original's. Rewriting moved logic to fit the twin's ports is a
rewrite wearing a refactor's clothes — put the logic where its current
shape already fits, even if that means the domain now spans two packages
with different port disciplines. Unifying the twins is its own future
mission, honestly scoped.

## W6-D4. Delete extraction shape — and the supplier-payment delete was never able to run

Mission A.2's first pair (customer payment, supplier payment) set the
extraction shape for the rest of the dispatch:

- **The moved logic is a plain package function**
  (`payment.DeletePayment(db, id, audit)`), not a method on the generic
  handler-shell Service — the shells forward through `Handlers` closures
  back to root, and delete extraction REPLACES that loop: the two
  handler fields and their forwarding methods are deleted, not rewired.
- **Root keeps guard + RBAC + audit sink**: `guardDeleteOrRequest` and
  `requirePermission` stay in the App method; the audit hook rides in as
  a nil-able closure called BEFORE the destructive write, preserving the
  audit-before-delete ordering (the closure carries `getCurrentUserID`
  and `GlobalAuditLogger`, which are host identity/hub concerns).
- **`pkg/kernel/apperr`** (new, kernel-pure) reproduces the host's
  `[CODE] Message: details` error string format so moved logic returns
  byte-identical errors to the UI. Root's `AppError` is never
  type-asserted anywhere — the string IS the contract.

**The whistleblow:** the historical `deleteSupplierPayment` rolled the
paid amount back into `supplier_invoices.amount_paid_bhd` — a column
that NO model or migration in the repo defines (repo-wide grep: the
delete was its only mention). On any schema migrated from these models
the UPDATE fails, the transaction rolls back, and supplier payments can
NEVER be deleted (pinned empirically by a pkg test before fixing). The
Commander authorized the in-wave fix: re-derive `payment_status`
(Paid/Partial/Unpaid) from the payments that remain — the exact
derivation `UpdateSupplierPayment` already uses. One deliberate
behavior extension: an orphaned payment (invoice row already gone)
stays deletable, because the old rollback UPDATE was a silent no-op in
that case.

**[Mirror-D4]** A delete path that "rolls back" into a ledger column is
only as real as the column. When a write path derives state one way
(sum-then-classify) and its inverse "un-derives" another way
(decrement-a-column), one of them is fiction — and nothing notices until
someone actually runs the inverse. Peeling forces the read; that is half
the value of the peel.

## W6-D5. The A.2 ledger: 14 moved, 2 already there, 7 honestly stay

The deletion Executor's dispatch, entity by entity:

**Moved to pkg (14):** Payment + SupplierPayment
(`pkg/finance/payment`), CustomerInvoice + SupplierInvoice
(`pkg/finance/invoice`), PurchaseOrder + GRN (`pkg/crm/procurement`),
DeliveryNote (`pkg/crm/fulfillment`), ExpenseCategory + ExpenseVendor +
ExpenseEntry (`pkg/finance/expense`), Customer + Supplier +
CustomerContact + SupplierContact (`pkg/crm/customer`). Every move
keeps guard + RBAC at the root wrapper and carries a pkg golden.

**Already in pkg (2):** BankStatement and BankStatementLine — the Wave
5 banking peel took their delete logic wholesale (with guard/RBAC
ports); the survey confirmed there is nothing left at root but the
one-line bindings.

(A.3's measurement later moved one more: OfferNote — see W6-D7 — making
the final tally 15 moved, 2 already there, 6 stay.)

**Stay at root, deliberately (7 at A.2 close, 6 after W6-D7):**
- **Order** — its cascade is ONE transaction spanning crm (order, POs,
  delivery notes, items) and finance (invoices, invoice items, plus a
  payments-exist precondition). Splitting it across the two domain
  packages would either break the transaction or force a crm↔finance
  import that no other file needs. The seam there is a saga, not a
  package boundary.
- **RFQ, RFQWithCascade, CostingSheet** — `RFQData` and
  `CostingSheetData` are still ROOT-owned models; the sales pipeline's
  model migration hasn't happened (A.3 is ports-only prep this wave,
  by design). Moving the deletes would mean moving the models first —
  a different mission. (OfferNote was initially grouped here but rides
  on pkg-owned `crm.OfferNote` — it moved under W6-D7.)
- **QuickCapture, CollaborativeTask** — collaboration-hub records
  (W4-D9: hubs get ports, never relocation).

**[Mirror-D5]** A dispatch table is not a work-list of equals. Sorting
its cases by *what the logic touches* — single-domain, cross-domain
transaction, hub — before moving anything turns "extract the dispatch"
into three honest verdicts instead of one dishonest relocation. The
partial list, finished, beats all of it shuffled.

## W6-D6. Configurable inactivity timeout: clamp, don't refuse; apply live

Mission C.2 makes the Wave 5 idle window configurable
(`security.session_timeout_minutes` in settings; General section of the
Settings screen). Decisions:

- **Clamped to 5–480 minutes, never refused.** settings.json is
  hand-editable; a typo ("3000") must degrade to the nearest sane bound,
  not silently disable the timeout or lock the user out of fixing it.
  Below 5 minutes the app is unusable; above 8 hours the timeout no
  longer protects an unattended terminal within a working day.
- **Resolved at login, applied live on save.** `beginInteractiveSession`
  reads the setting once (not per bound call — `touchInteractiveSession`
  runs on EVERY bound call and must not do file IO); `UpdateSettings`
  applies a changed value to the RUNNING session immediately, so
  shortening the window doesn't require a re-login to take effect.
- **Zero/absent leaves the default** (30 min, the Wave 5
  Commander-decided policy) — the constant is now the fallback, not the
  law.

**[Mirror-D6]** When a security policy becomes configurable, the
validation posture flips: a service refuses bad input loudly (W6-D1),
but a *policy knob* clamps it — because the failure mode of refusing a
mis-typed timeout is "no timeout at all," which is strictly worse than
either bound.

## W6-D7. Sales-pipeline measurement: one leaf, no new ports

Mission A.3 asked for port stabilization, not relocation. The
measurement of `app_sales_pipeline.go` (4,213 LOC): 149× `a.db`, 43×
`requirePermission`, plus offer numbering (`ensureOfferNumberAvailable`,
`generateOfferNumber` — the standing stop-and-ask, untouched), RFQ/
costing lifecycle methods on ROOT-owned models (`RFQData`,
`CostingSheetData`), and opportunity-stage sync. Verdicts:

- **One self-contained leaf found and peeled**: `DeleteOfferNote` —
  db-only coupling on the pkg-owned `crm.OfferNote`. Moved to
  `pkg/crm/pipeline` (its first real code), with a pkg golden. This
  also closes one more Executor dispatch case.
- **No new ports minted.** The A.2 extractions stabilized the shape the
  pipeline peel will reuse (plain pkg functions taking `*gorm.DB`, root
  wrappers holding guard + RBAC + host events, `apperr` for coded
  errors); the existing deletion Identity/Notifier and payroll port
  vocabulary covers the rest. Inventing pipeline-specific interfaces
  before the models migrate would be speculative — port count is a cost
  (W5-D5).
- The real precondition for the pipeline peel is the MODEL migration
  (`RFQData`, `CostingSheetData` out of root), which belongs to a wave
  that can afford the schema-compatibility audit.

**[Mirror-D7]** "Prepare the ports" sometimes resolves to "the pattern
is the port." When five sibling extractions have already fixed the
calling convention, the preparation a future peel needs is that
convention held steady — not a new interface file that guesses at it.
