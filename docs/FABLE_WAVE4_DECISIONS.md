# Fable Wave 4 — Decision Log

Wave 4 ("Shrink the God") ran without a written handoff: the spec was the
residue list in `docs/FABLE_WAVE3_PROGRESS.md`, scoped with the Commander at
kickoff (ZATCA sandbox deferred — it needs portal OTPs; App fan-out
confirmed as the headline). Decisions are numbered W4-D1…; **[Mirror]**
paragraphs record what generalizes beyond this codebase.

## W4-D1. The reference peel shape: aggregate + logic inward, host behind ports

`pkg/infra/deletion` is the first REAL peel off the 1229-method App. The
existing "extractions" in app_services.go are mostly handler trampolines —
generic `Service[T…]` structs whose Handlers are closures calling straight
back into root functions — a seam, but the logic never moved. The banking
`Ports` pattern was the one honest inward idiom, so the peel extends it:

- The AGGREGATE moves: `deletion.Request` owns the GORM model; root keeps
  `type DeleteApprovalRequest = deletion.Request`, so the table shape, JSON
  contract, model registry and Wails-visible behavior don't change.
- The WORKFLOW moves: request dedup, admin fan-out content, the kernel
  approval gate, execute-before-persist review, requester notification.
- The HOST stays behind three narrow ports: identity (session + license
  machinery), notification delivery (rows, receipts, sync queue, UI
  events), and delete execution (the 24-way entity dispatch, which fans
  into every domain and belongs where the domains are).

Proof of behavior-identity: the pre-existing app-level end-to-end tests
pass untouched.

**[Mirror]** When a codebase already contains a failed extraction idiom
(the trampoline), don't add a parallel new one — extend whichever existing
idiom actually moved logic, and SAY which one that is. The next reader must
be able to tell the honest seams from the decorative ones.

## W4-D2. GRN numbering: finish the S4 migration, seed from MAX not COUNT

GRN was the last document type still allocating numbers via raw
`BEGIN EXCLUSIVE` + max-scan — the exact pattern the S4 fixes replaced for
INV/CN/PO/DN (the lock committed BEFORE the number was used, and any read
error silently restarted the year's sequence at 0001: duplicate risk).
`GenerateGRNNumber` now delegates to `pkg/documents/numbering` like every
other document. Not treated as stop-and-ask because the format is unchanged
and the migration completes an already-authorized pattern; the one
deliberate divergence: the first-of-year seed parses the MAX existing legacy
number instead of the COUNT the DN/PO seeds use, so deleted GRNs can never
cause a number to be reissued. (The count-based seeds in DN/PO share that
theoretical reuse hazard on their migration year only — recorded here, not
silently "fixed", since their migration years are behind us.)

**[Mirror]** When several call sites migrated to an engine and one
straggler remains, the straggler is where the ORIGINAL bug still lives.
Migrating it is not refactoring backlog — it's an open defect with a known
fix.

## W4-D3. SessionManager: write-only security theater, deleted loudly

The in-memory SessionManager in security_enhancements.go advertised
8h-inactivity/24h-max-age session security. Measured reality: LoginDevice
stored a session into a `sync.Map` (discarding the return value and
ignoring the deviceID argument), and NOTHING ever called IsSessionValid /
UpdateActivity / EndSession. The policy was never enforced anywhere; the
map was never read; an immortal goroutine cleaned up entries with no
observable effect. Meanwhile the REAL session system — the DB-backed
AuthManager in auth_session.go (hashed tokens, validate/invalidate/cleanup)
— ran beside it. Same disease as Wave 3's discarded AuditEvent, same
treatment: deleted with a signpost NOTE. If interactive-session inactivity
timeout is wanted, it gets wired through AuthManager as a deliberate
security change, not by resurrecting the map. The live RateLimiter was the
keeper: promoted verbatim to `pkg/infra/ratelimit` with a type alias.

**[Mirror]** Audit a security component by tracing its READ side. A
component that is only ever written to enforces nothing — its existence is
worse than absence, because it reads as coverage in every security review
that doesn't trace data flow.

## W4-D4. PDF "three generator paths": closed by measurement, no code change

The Wave 3 residue said "unify the three generator paths." Ground truth,
measured: seller identity is ALREADY unified on the overlay everywhere
(W3 B.4 fixed the engine; company_branding.go was already clean). The
engine's invoice renderer (`engines.PDFGenerator.Generate`) has ZERO live
product callers — nothing in root constructs `engines.InvoiceData`;
contracts borrow the engine's canvas via RenderContractPDF, nothing more.
The LIVE business documents are a consistent jung-kurt/gofpdf fleet
(invoice, supplier invoice, PO, DN, CN, butler/ops reports) sharing the
overlay letterhead helpers; costing exports and analytics reports draw with
signintech/gopdf directly. The two invoice builders share ~30 lines of page
scaffold, not layout.

So the actual unification is: re-render live pilot documents through one
canvas. That changes the appearance of customer-facing financial documents
— a visual-regression project needing per-document sign-off, not a
refactor commit. Deliberately NOT attempted; reframed in the Wave 5 residue
with this map attached. (Also noted: jung-kurt/gofpdf is archived upstream,
which is the eventual forcing function.)

**[Mirror]** Residue lines written from altitude ("unify the three X")
deserve re-measurement before execution. Sometimes the honest deliverable
is the corrected map and a NO — recorded loudly enough that the next wave
doesn't re-derive it.

## W4-D5. Partial refunds: quantity is truth, amounts are per-document

`RefundInvoiceLines` credits named quantities of named lines. Design calls:

- The refund LEDGER (`hosp_credit_note_lines`) is quantity-truth: each
  line caps at its billed quantity, cumulative across credit notes. Amounts
  are derived per document by the ZATCA arithmetic.
- NO invented "partially_refunded" invoice status. The invoice stays `paid`
  until the last billed quantity is credited, then flips to `refunded`.
  The negative tender rows carry the financial truth in between; a new
  status would have rippled through day close and every status switch for
  zero information gain.
- Full refund (`RefundInvoice`) is behavior-identical on a clean invoice
  (existing tests untouched) and now explicitly refuses an invoice carrying
  partials — finishing line-by-line is the coherent path.
- The legacy UNIQUE(original_invoice_id) index ("one full refund per
  invoice") is retired EXPLICITLY in migrate — AutoMigrate never drops
  indexes, and the replacement index gets a new name so pre-partial
  databases converge on the same shape as fresh ones.

## W4-D6. The rounding guard: refuse the split that over-refunds

Per-line VAT rounding means splitting one line's QUANTITY across credit
notes can round each partial UP: Karak Chai at 8.48 net → full-line VAT
1.27, but two half-quantity documents carry 0.64 each — 976 halalas
credited against a 975-halala invoice. The cumulative guard refuses any
credit note that would push total credited past what the guest paid, with
an explicit "rounding drift" error, rather than silently handing back an
extra halala (or silently shaving one off the last document, which would
break the signed document's own arithmetic). Whole-line splits are
drift-free by construction (the document arithmetic rounds per line). A
crafted half-quantity test pins the guard; the refused document leaves no
rows behind.

**[Mirror]** When money is split across independently-rounded documents,
the invariant to enforce is on the SUM, and the enforcement point is the
moment of issuance — not a reconciliation report later. Refuse loudly;
never adjust a signed document's numbers to make a ledger fit.

## W4-D7. CreditNoteIssued: its own event, positive magnitudes

Wave 3 deliberately refused to smuggle credit notes under `InvoiceCreated`;
Wave 4 gives them their own event. `events.CreditNoteIssued` carries the
same tax-relevant fields (NET base, tax, currency, jurisdiction) plus the
referenced original invoice number. Amounts are positive magnitudes — the
event TYPE conveys direction, matching the stored document's convention.
The compliance hook subscribes to it and validates under the credit-note
event name; its record-keeping had `EventInvoiceCreated` hardcoded into
every entry, which the generalization fixed (`handleEvent` now records the
actual event's name — previously a PaymentRecorded validation would have
been logged as an invoice validation).

**[Mirror]** An event bus where every subscriber handler is named after one
event ("OnInvoiceCreated") will misattribute every OTHER event routed
through it. Name handlers after what they DO (validate), parameterize what
they receive.

## W4-D8. Payroll approval onto the kernel; RBAC stays the authority source

`ApprovePayrollRun` approved money with only an RBAC string check — the gap
Wave 3 B.1 closed for costing and delete approval, still open here. It now
routes through `pkg/approvals`. One deliberate asymmetry vs delete-approval:
the operator actor is minted with approve authority BECAUSE the session
already passed `requirePayrollApprove` — RBAC remains the human-authority
source (a non-admin HR manager with the permission must keep working), and
the kernel gate's contribution is the transition table plus the type-based
agent boundary. "draft" joined the approvals engine's status synonyms for
pending — it is generic approval vocabulary, not payroll's.

**[Mirror]** When retrofitting a kernel authority gate onto an RBAC-guarded
flow, decide explicitly which system is the SOURCE of human authority and
mint the actor from it — deriving authority from a different predicate
(e.g. "is admin") silently revokes permissions RBAC deliberately granted.

## W4-D9. The fan-out map (measured, for the next peels)

The root package: 218 files, ~128k LOC, 1230 `func (a *App)` methods,
7 Wails-bound objects (`app` + 6 façade services that are pure re-binding
shims — FinanceService 253 methods, CRMService 228, InfraService 185,
DocumentsService 103, SyncServiceBinding 54, ButlerService 36). Top clusters
by method count: app_setup_documents_surface (73 methods / 4531 LOC),
butler_ai_context (57 / 5070), app_sales_pipeline (57 / 4213, contains the
stop-and-ask Offer NN-YY site), app_order_customer_surface (50 / 2535),
collaboration_service (49 / 2547), app_accounting_inventory (34),
payroll_service (33), app_auth_rbac (27), customer_invoice_service (26).

Idioms found: the honest inward seams are banking `Ports` and now
`pkg/infra/deletion`; the generic `Service[T…]`+Handlers services are
trampolines (logic still in root as free functions the closures call back
into). Coupling reality: `a.db` appears in 84 files and is CHEAP to carry;
the EXPENSIVE coupling is to the two HUB clusters — auth/RBAC
(`currentUser`, `requirePermission`, `hydrateUserRole`) and
collaboration/notifications (`emitCollaborationEvent`,
`enqueueCollaborativeOperation`, `recordNotificationReceipt`,
`GetCurrentEmployeeContext`), which everything leans on.
`guardDeleteOrRequest` alone has 13 call sites across the domains, and the
deletion Executor port's ~26-way dispatch is the natural work-list for
entity-by-entity extraction.

**The standing rule for the hubs: PORTS, not relocations.** Auth/RBAC and
collaboration/notifications get their interfaces stabilized behind ports
(the deletion Notifier port is instance #1); relocating them would drag the
whole root along.

Cheap next seams, in recommended order: serial numbers →
pkg/crm/fulfillment (pairs with GRN completion); cheque register and FX
revaluation → pkg/finance/* (a.db-only leaves, target dirs exist);
assets/device (small CRUD, momentum); the CRM contract body-move (finish
the cleanest existing peel as the exemplar). Butler read paths are also
cheap — invariant #4 means they can't persist, so no RBAC/notification
entanglement. Expensive, do later or only behind ports: the sales-pipeline
surfaces (dense RFQ→offer→costing→order cross-calls, every delete gated),
OneDrive/ETL import (fileWatcher/graphClient/dbManager state), and the two
hubs. Payroll's coupling profile (53× a.db, identity helpers, posting
journal, event emitter) makes it the best next FULL-domain peel — after
its posting semantics get golden coverage.

## W4-D10. Deferred with reasons

- **ZATCA sandbox round-trip** — needs Fatoora portal OTPs and JDK 11–14;
  human-in-the-loop, deferred at kickoff by the Commander.
- **Offer NN-YY numbering** — standing stop-and-ask (OneDrive coupling),
  untouched again.
- **Full PDF canvas unification** — see W4-D4: a visual sign-off project,
  not a refactor.
