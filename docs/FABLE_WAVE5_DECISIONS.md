# Fable Wave 5 — Decision Log ("Peel by the Map")

Wave 5 executes the W4-D9 fan-out map: the cheap-seams queue, then the
payroll golden-test-first peel, session inactivity through AuthManager
(Mission B), and hospitality bill split (Mission C.1). Decisions are
numbered W5-D1…; **[Mirror]** paragraphs record what generalizes.

## W5-D1. Serials peel: when the models already moved, the peel needs no ports

`pkg/crm/fulfillment.Serials` is the first Wave 5 peel. It came out
simpler than the reference shape (W4-D1) predicted: the aggregate
(`crm.SerialNumber`) and its neighbors (`ProductMaster`, `DeliveryNote`)
had already moved to `pkg/crm` in an earlier wave — root only held type
aliases. So the whole lifecycle (register, GRN assignment, atomic DN
allocation, shipped/delivered transitions, invoice linking, warranty and
calibration) moved inward with only a `*gorm.DB` dependency and ZERO host
ports. The RBAC guards (`requirePermission("grn:*")`) stay in the root's
thin delegates — auth is a hub (W4-D9), and a permission string check is
host policy, not serial-lifecycle logic.

Two cleanups rode along deliberately:

- The generic fulfillment `Handlers` (the trampoline idiom) carried two
  serial closures next to its delivery-note ones. They are deleted and the
  generic narrows to `Handlers[DeliveryNoteModel]` — the seam no longer
  mixes an honest peel with decorative indirection in one package.
- `escapeLikeWildcards` existed twice (root `security_helpers.go`,
  private copy in `pkg/finance/banking`). The canonical home is now
  `pkg/kernel/text.EscapeLike` (pure string transformation — kernel-pure);
  root delegates. The banking private copy is left untouched this wave
  (it is package-internal and correct; converging it is churn, not risk
  retirement).

**[Mirror]** Before designing ports for a peel, check whether earlier
waves already relocated the data model — a peel whose aggregate has moved
often needs no ports at all, and inventing an IdentityPort for a bare
permission-string check would manufacture abstraction where a thin
delegate is the honest shape. Port count is a cost, not a badge.

## W5-D2. Golden-first held for FX: exact-binary fixtures beat tolerances

The FX peel (A.1c) followed the A.2 payroll rule even though the map
filed it under "cheap seams": revaluation multiplies balances by rates —
that IS financial arithmetic. The goldens were committed green against
the untouched root code in their own commit (4d0b1f9), then the peel
commit ran them unchanged. Fixture design choice: balance 1024.0 and
rates 0.375 / 0.4375 / 0.5 are all exact in binary floating point, so
every assertion is exact float64 equality — no tolerance argument to
review, no epsilon that could silently mask a real drift.

**[Mirror]** When pinning float arithmetic, choose fixtures whose values
(and products) are exactly representable in binary. A tolerance-based
golden proves "close"; an exact-binary golden proves "identical", and its
diff on failure is meaningful to the reviewer.

## W5-D3. Device "CRUD" re-measured: half of it is the auth hub

The map's cheap-seam line "assets/device (small CRUD)" held for assets
and for HALF of device. Ground truth: SetupAdminAccount, ApproveDevice,
and LoginDevice mint users with bcrypt, mutate the session
(currentUser/currentUserID), and call GlobalValidator / GlobalAuditLogger
/ GlobalRateLimiter / hydrateUserRole — that is the auth/RBAC hub wearing
a device costume. Per the W4-D9 standing rule those flows STAY in root;
what moved is the genuinely device-owned logic: fingerprinting,
first-setup/pending registration, lifecycle queries, block/unblock. The
audit calls around Block stay in the root delegate (audit is host policy
about the action, not device logic).

**[Mirror]** A file's NAME is not its coupling profile. Before executing
a mapped extraction, grep the file for session mutation and hub helpers —
the honest peel boundary often cuts through the middle of a file, and
"we moved the whole file" is the wrong success metric.

## W5-D4. Contract body-move: the free audit found seeds that never worked

The CRM contract "cleanest existing peel" turned out to be an EMPTY stub —
`pkg/crm/contract` held a 12-line Service with zero methods while the real
845-line ContractService sat in root, and `a.services.contract` was
constructed but never called. The body-move (types + all methods into the
package; root keeps aliases: ContractService, Contract, ContractTemplate,
ContractClause, requests) is now the real exemplar. `wrapText` graduated
to `pkg/kernel/text.Wrap` (many costing-export callers keep the root
name as a delegate).

Found while writing the pkg tests (the W3-D9 "free audit" effect):
Contract, ContractTemplate, and ContractClause never embedded
`shareddomain.Base`, so nothing minted their IDs — `SeedContractClauses`
inserted the first clause with ID "" and every subsequent insert failed
on a duplicate empty-string primary key. The seeds could NEVER have
completed on a fresh database. Fixed with BeforeCreate hooks matching
Base.BeforeCreate exactly (an intent-restoring bug fix, not a semantic
change — recorded per W4-D2: the straggler is where the live bug still
is).

**[Mirror]** "Constructed but never called" is the service-level version
of the write-only SessionManager (W4-D3): a seam that LOOKS colonized in
the wiring diagram but moved nothing. Audit extraction progress by where
method bodies live, not by which packages appear in initServices.

## W5-D5. Payroll peel: pkg/finance/payroll, posting inward, four ports out

The payroll domain moved to **pkg/finance/payroll** (not pkg/hr — the
accrual/payout journals ARE finance posting, and the finance models it
writes already live in pkg/finance). Boundary decisions:

- **Posting moved INWARD**, not behind a port: JournalEntry/JournalLine/
  ChartOfAccount are pkg/finance models, so the accrual journal (6000/
  6050 debits against 2210/2211/2212), the payout journal (2210 against
  1000 cash), balance maintenance, and the private ensureAccount all
  travel with the domain. The handoff's "posting stays behind ports"
  assumed the models were host-side; they weren't. The golden tests are
  the arbiter: they pass unchanged.
- **Four ports out**: IdentityPort (user/actor/display — the employee-
  context fallback chain stays host-side), DirectoryPort (Employee lives
  with the collaboration hub; payroll consumes a 4-field EmployeeRef),
  EventsPort (Wails runtime emission), and ExpenseBridgePort — the
  payroll→expense-ledger mirror STAYS in root because it drives the
  expense service's numbering, approval recording, and events (another
  root cluster; port, don't drag).
- The kernel approval gate moved in as payroll.GateRunApproval; root
  keeps a one-line gatePayrollRunApproval shim so approval_routing_test
  is untouched (behavior-identity proof stays valid).
- Model names shortened inside the package (Run, RunItem, Payout…); root
  aliases keep every historical name, the table shapes, and the registry.

Proof: the golden tests committed one commit earlier (e289a68) against
the untouched code pass unmodified against the peel — run generation
totals, the five accrual lines balanced at 2056, account balances, and
the expense mirror, all exact float64 equality.

**[Mirror]** "X stays behind a port" instructions written before a peel
should be re-derived from where the DATA lives at execution time. A port
for logic whose models are already inside the boundary is pure ceremony;
the honest rule is: port the neighbors you'd otherwise drag, inline the
ones that already moved.

## W5-D6. Mission B: inactivity enforcement at the chokepoint, session rows for audit

Policy locked with the Commander at kickoff: **30 minutes idle; any bound
call counts as activity; expired calls refused with a "session expired"
error; the frontend returns to the login screen.**

Ground truth first: AuthManager's UserSession table served ONLY the OAuth
(Microsoft) token flow — the interactive login (LoginDevice /
SetupAdminAccount) never created a session row, so there was nothing to
expire. Design decisions:

- **Enforcement point = requirePermission.** Every bound endpoint already
  passes through it, so it is simultaneously the refusal point and the
  activity signal. This satisfies the W4-D3 mirror rule by construction:
  the component's READ side is the RBAC chokepoint itself, and the tests
  drive bound-call behavior (expiry blocks, activity extends, logout
  invalidates), not just row states.
- **In-memory clock, DB audit.** The desktop app's interactive session
  dies with the process, so the authoritative inactivity clock is the App
  field; the UserSession row (created at login through the same table
  AuthManager owns, invalidated with reasons: inactivity_timeout /
  user_logout / superseded_by_new_login) is the audit trail.
  last_activity_at writes are throttled to one per minute so dashboard
  polling doesn't become a write amplifier — the row lags ≤60s, recorded
  here as a known property.
- **Token columns are unique-indexed**, so interactive sessions mint
  opaque per-session token hashes; two logins with empty-string tokens
  would otherwise collide — the second row would silently fail.
- **Scope**: enforcement covers the paths that mint a.currentUser at
  runtime (LoginDevice, SetupAdminAccount). Tests and license-based flows
  that never begin an interactive session are untouched — deliberately,
  so the change is small and composable per the CIA-audit plan.
- Frontend: one Wails event (auth:session-expired) returns the app to the
  login screen with a toast; a new LogoutInteractiveSession binding gives
  the UI a proper logout (none existed for interactive logins).

**[Mirror]** When adding a session timeout to an app with an RBAC
middleware, the middleware IS the natural enforcement point — a separate
"session validator" that endpoints must remember to call recreates the
write-only-component failure mode. Enforce where the calls already flow.

## W5-D7. Bill split: whole lines only, and the line→invoice stamp it forced

`SplitSession` closes one open session by issuing one ZATCA invoice per
assignment group, whole-line assignment only. Design decisions:

- **Quantity splitting refused, not implemented.** Whole-line assignment
  makes the sum invariant hold by CONSTRUCTION: the ZATCA arithmetic
  rounds per line, so a line carries identical rounded amounts whichever
  document it lands on, and the split totals sum exactly to the
  single-invoice reference. A defense-in-depth guard still computes the
  reference and refuses the whole transaction on drift (W4-D6: refuse,
  never adjust) — it should never fire; it pins the invariant. The tests
  compare against an identical session closed as ONE invoice.
- **The split forced a schema truth into the open**: refunds resolved an
  invoice's lines BY SESSION (billableLines), which is only correct while
  a session has one invoice. OrderLine gains a nullable InvoiceID stamped
  at issuance (CloseSession too); refund lookups prefer the stamp and
  fall back to session-scoped-with-NULL-stamp for legacy invoices — exact
  for them, and the NULL filter means a later split on the same database
  can never leak lines across the boundary. Without this, a refund
  against split invoice A could credit B's lines and the cumulative cap
  would compare against the wrong total.
- Every split invoice is its own document on the shared ICV/PIH chain,
  issued inside ONE transaction (chainHead reads its own transaction's
  uncommitted predecessors, so the chain can't fork); one InvoiceCreated
  event per document after commit; the closed session points at its
  first split invoice (the full set hangs off session_id).
- Kernel CanApprove gates the split exactly like CloseSession; agents
  never issue. Every refusal path leaves the session open and unbilled.

**[Mirror]** A "split one document into N" feature is really a foreign
key you didn't know was missing: find every query that resolves children
through the PARENT of the document (session, order, batch) — each is a
latent cross-document leak the moment two documents share that parent.
Stamp the children at issuance and give legacy rows an exact fallback.
