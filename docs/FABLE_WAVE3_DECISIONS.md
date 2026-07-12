# FABLE Wave 3 — Decisions Log (Make the Proof the Law)

**Scope:** Wave 3 per `FABLE_WAVE3_HANDOFF.md` — Mission A (decompose the
trading composition root), Mission B (A.1 engine residue: approval routing,
audit convergence), Mission C (ZATCA hardening + hospitality graduation),
Mission D (this mirror). Same format as Waves 1–2: every consequential
decision recorded — what was decided, what was rejected, why. **[Mirror]**
annotations state what a less capable model would need to be told.

---

## W3-D1. Branch base: Wave 2 merged to main first (Commander's call)

Wave 2's branch was NOT merged when Wave 3 started — main sat at the
merge-base. Per the handoff's "do NOT assume" instruction, asked the
Commander; answer: fast-forward `main` to `feat/fable-wave2-composition-proof`
(clean descendant, no merge commit needed), then branch
`feat/fable-wave3-composition-law` off main.

**[Mirror]** When a spec says "ask which base if unclear", the cheap move is
a one-question prompt BEFORE the first commit, not a guess that turns into a
rebase later. Also: check `git merge-base` yourself first so the question
carries the answer options.

## W3-D2. The trading DSN's pragmas were dead — measured, then fixed to WAL

**Found (measured, not assumed):** the trading app's DSN used mattn-style
params — `?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&…` —
which the ncruces driver **silently ignores**. A two-test probe against both
DSN styles showed the pilot had actually been running `journal_mode=DELETE`,
`synchronous=FULL`, with the driver-default 60s busy timeout. The comment
"WAL mode supports concurrent readers" above `SetMaxOpenConns(4)` was a
false premise the whole time; the driver's generous default busy timeout is
what kept lock errors rare. (This also sharpens Wave 2's numbering-deadlock
finding: DELETE-mode SQLite serializes readers and writers harder than WAL.)

**Options:** (a) preserve delete-mode byte-identically (behavior-identical
purism); (b) fix to the code's stated intent (WAL + NORMAL + 5s timeout);
(c) WAL but keep synchronous=FULL. Commander chose **(b) fix to intent** —
the mis-state was a bug, not a behavior anyone chose. Side effects accepted:
`-wal`/`-shm` files appear next to the pilot DB; first open converts the
journal transparently.

**Where the fix lives:** `composition.DefaultPragmas` emits the ncruces
`?_pragma=name(value)` form, and `TestSQLiteDSN_PragmasAreHonored` PINS that
the pragmas actually apply (journal_mode, busy_timeout, synchronous,
foreign_keys, cache_size all asserted against a live connection). If the
driver's DSN contract ever changes, that test fails loudly.

**[Mirror]** Two rules. (1) *"Behavior-identical" requires knowing what the
behavior actually IS* — a config string is a claim, not a fact; probe the
runtime state (`PRAGMA journal_mode`) before promising to preserve it.
(2) When a fix changes production runtime characteristics (file layout,
locking), it is the deployment owner's call, not the refactorer's — ask,
with the measurement in hand, and record the authorization.

## W3-D3. A.2 model registry: verbatim extraction + schema golden with sorted constraints

The ~90-model AutoMigrate list moved verbatim from `startup()` into
`tradingModels()` (`trading_models.go`); the migrate-each-model-individually
loop moved into `composition.MigrateModels` with a report callback so the
trading app keeps its diagnostic logging. Order preserved; a sorted-diff of
the deleted lines against the new file confirmed the move byte-identical.

**Schema pin:** `TestTradingModels_SchemaGolden` migrates the registered set
into a fresh DB and compares a `sqlite_master` dump against
`testdata/trading_schema.golden` (regenerate deliberately with
`-update-schema-golden`). First run exposed that GORM emits CHECK
constraints in **map order** — the same model set produces differently
ordered `CONSTRAINT` clauses run to run. The dump therefore sorts the
trailing CONSTRAINT clauses per CREATE TABLE (column order untouched) before
comparing; three consecutive runs pinned stable.

**[Mirror]** A golden test that fails nondeterministically teaches the team
to ignore it. When pinning generated DDL, know which parts the generator
orders deterministically (columns: struct order) and which it doesn't
(constraints: map iteration) — normalize exactly the nondeterministic part,
nothing more, or the golden stops guarding column order too.

## W3-D4. A.3 seeds: gate in place, don't relocate

The seven trading seed/asset bundles (license-keys, employee-keys,
default-assets, rbac-roles, demo-products, demo-customers, demo-bank) run at
FOUR different points in startup() — license keys right after migration,
RBAC after DBManager, demo rows after the job queue. **Rejected:** moving
them into one composition-seam "seed stage" — that reorders side-effectful
boot steps whose dependencies (RBAC bypass windows, DBManager, settings
service) are implicit, for zero configuration gain. **Chosen:** each call
site stays exactly where it is and gains an `activeOverlay.SeedEnabled(name)`
gate; the overlay grows an optional `seed_sets` array. Semantics pinned by
test: absent field → every bundle runs (existing deployments byte-identical);
`[]` → no optional seeding; a list → exactly those bundles ("all" sentinel).
The nil-vs-empty distinction is load-bearing and JSON-decode-tested.

**[Mirror]** "Config-drive the seeds" has two readings: relocate them or
gate them. Relocation looks cleaner but silently reorders a boot sequence
nobody fully specified. When behavior-identical is the contract, prefer the
transformation whose identity is provable from the diff alone: a gate that
defaults to true changes nothing; a move changes ordering you can't see.

## W3-D5. A.4: one engine list, one registration path — plus a fallback that shares it

`initComplianceEventBus` already delegated to the seam after A.1; the
remaining second registration site was `GetComplianceDashboard` building a
private registry inline. It now reads the process registry from the
composition root; the only remaining inline construction is its
pre-wiring fallback (the dashboard binding can be called before
initComplianceEventBus runs late in startup), and that fallback consumes the
same `tradingTaxEngines()` list. Result: the engine SET is declared once;
registration flows through `composition.WireCompliance` in both verticals.

**[Mirror]** "Exactly one place" is about the *source of truth*, not the
call count. A fallback path is fine as long as it consumes the same
declaration; what must die is the second hand-maintained list that can
drift (the dashboard would happily have kept validating with two engines
after a third was added — that class of bug).

## W3-D6. B.1 approval routing: rules stay in the vertical, the FOLD is the engine

`pkg/approvals` expresses both existing approval flows on kernel vocabulary
with two small shapes: `Assessment` (routing — domain rules emit Findings,
the engine folds them into DecisionApproved/DecisionPending) and
`Transition` (deciding — kernel `ValidTransition` legality + actor
`CanApprove` authority on every pending→approved/rejected move).
**Rejected:** a rule DSL that would move the costing thresholds into the
engine — the thresholds are trading policy (overlay BusinessRules), and a
predicate DSL abstracting six if-statements is ceremony, not substrate.

Two behavior notes found during extraction, both preserved deliberately:
(1) the costing flow's rule-specific recommendations ("Require full payment
upfront or decline", "Review pricing urgently") were ALWAYS overwritten by
the final blanket "Requires manager approval before proceeding" — dead
assignments; the engine keeps the observable outcome and the pinning test
locks warning ORDER as well as text. (2) the delete flow's "already
%status%" guard is subsumed by the kernel transition table but kept — its
error message is user-facing contract.

The AI-authority boundary is now enforced at three depths, each tested:
kernel construction (an agent actor cannot be MINTED with approve
authority), `approvals.Transition` (an agent — or an observer — is refused
approve AND reject), and both call-site flows (delete review builds the
actor from the authenticated session via `currentApprovalActor`; costing's
pending decisions refuse agent transition in `TestCostingApproval_AgentRefused`).

**[Mirror]** When told to "promote routing onto the kernel", the failure
mode is promoting the RULES (domain policy that belongs to the vertical)
instead of the DECISION ALGEBRA (what the kernel actually owns: state
legality + actor authority). Ask of each moved line: would a second vertical
with different thresholds reuse this? The fold, yes; the 20% number, never.
Also: when extraction reveals dead writes (the overwritten recommendations),
preserve the observable behavior and RECORD the deadness — do not "fix" it
mid-refactor, and do not silently carry the dead code forward either.

## W3-D7. B.2 audit convergence: one Entry shape, one table; the dead vocabulary deleted

**Found:** two audit systems. The live one (`App.logAudit` → `infra.AuditLog`)
accepted `resourceID` and `description` and silently dropped both behind a
"Field removed in simplified schema" comment — the linter even flags the
unused parameters. The second (`security_enhancements.go AuditLogger`)
constructed rich `AuditEvent` values and immediately DISCARDED them
(`_ = AuditEvent{…}`); its four Log* methods reached the structured security
log only. Nothing security-critical was ever persisted to `audit_logs`
beyond (userID, action, resource).

**Decision:** one engine-backed path — `pkg/infra/audit.Recorder` with a
single `Entry{UserID, Action, Resource, ResourceID, Description}` shape.
`infra.AuditLog` gains the two dropped columns (additive migration — no
existing rows rewritten; schema golden regenerated deliberately, +2 columns
+2 indexes). `logAudit` delegates (async, as before) and now persists what it
is handed. `AuditLogger` keeps its structured security logging AND persists
through the recorder once startup wires it (`SetRecorder` after DB open —
before that it is log-only, the historical behavior). The `AuditEvent`
struct is **deleted loudly**: a NOTE at its former site names this decision,
and this entry is the record. Financial amounts are masked
(`MaskPaymentAmount`) before entering the description — audit rows must not
leak what the security log masks.

**[Mirror]** Two audit systems is not redundancy, it is a gap: each caller
picks one and half the trail is missing. When converging, (1) the persisted
schema wins over the richer in-memory vocabulary — extend the table, delete
the struct; (2) grep for what the dead system's callers PASS (masked
amounts) and preserve those guarantees in the converged path; (3) delete
with a signpost comment at the site, because the next reader will remember
the old struct and go looking for it.

## W3-D8. C.2 CSR generation: match OpenSSL's bytes, including its "SN" quirk

`saudi.GenerateCSR` hand-assembles the ZATCA-profile PKCS#10 (stdlib refuses
secp256k1, as with certificates). Profile cross-checked against two
independent SDK-aligned implementations (Saleh7/php-zatca-xml
CertificateBuilder, wes4m/zatca-xml-js csr_template); both agree: subject
CN/OU/O/C=SA; extensionRequest carrying exactly (a) certificateTemplateName
(MS OID 1.3.6.1.4.1.311.20.2) as UTF8String — ZATCA-/PREZATCA-/TSTZATCA-
Code-Signing by environment — and (b) subjectAltName = directoryName with
SN, UID, title, registeredAddress, businessCategory; **no keyUsage** (the
research notes suggested one; both references omit it — references win).

Load-bearing quirk: every reference generates through OpenSSL config, where
the dirName key `SN` is OpenSSL's short name for **surname (2.5.4.4)**, not
serialNumber (2.5.4.5). ZATCA therefore RECEIVES the EGS serial as a surname
attribute, and we encode 2.5.4.4 deliberately rather than "correcting" it.
Verified end-to-end with OpenSSL itself: `openssl req -noout -text -verify`
reports self-signature OK, secp256k1, and the exact expected DirName —
independent-tool verification, not just our own parser agreeing with our own
emitter. Input validation pins ZATCA's VAT-number shape (15 digits, starts
and ends with 3) and the 4-digit binary TSCZ flags.

**[Mirror]** When a spec is de-facto defined by "what OpenSSL emits from
this config", the config's KEY NAMES are part of the wire format — resolve
each through the tool's own alias table (SN→surname) instead of assuming the
semantically-obvious OID. And always verify hand-assembled DER with a
foreign parser; a round-trip through your own code proves consistency, not
correctness.

## W3-D9. C.1 verdict: both ❓ flags CONFIRMED against the SDK itself; the SDK zip is stale

The Commander downloaded the SDK zip linked from zatca.gov.sa (the
classifier rightly refused to let the agent pull external code itself).
Findings, full detail in `docs/research/ZATCA_PHASE2_RESEARCH.md`:

1. **Transforms confirmed by primary source.** The SDK jar embeds its own
   signing template (`xml/ubl.xml`); our SignedInfo matches it exactly. The
   pinning test (`TestSignedInfo_MatchesSDKTemplate`) EXTRACTS the
   algorithms and XPath strings from the vendored template and asserts our
   emission contains them in order — the template is the single source, not
   a copied string that could drift with it.
2. **Tag 9 = certificate signature bytes** (current standard + two current
   implementations; our code already matched). Standard invoices never mint
   a local QR here, so standard-tag-9 semantics stay moot by design.
3. **The publicly-linked SDK zip is v2.03 (May 2022)** — its QR validator
   still implements the DRAFT tag semantics (8/9 = signature R/S). Recorded
   loudly so nobody "validates" our QRs against it and panics.
4. **Free audit strikes again:** the SDK's key/cert ship as bare base64
   without PEM armor — the same form as the live gateway's
   binarySecurityToken — and our parsers required PEM headers while their
   doc-comments claimed otherwise. Fixed; the SDK's ZATCA-test-CA-issued
   certificate and matching key are now the regression fixtures, which also
   pins the SAN-surname quirk (the CA itself encodes 2.5.4.4) and the
   genesis PIH.

**[Mirror]** Reference materials from the vendor are worth more as TEST
FIXTURES than as reading: parse them with your real code paths and every
mismatch is a production bug found offline. But CHECK THE VERSION of what a
vendor portal serves — official ≠ current; a stale SDK validator would have
"refuted" our correct QR implementation.

## W3-D10. C.3 credit notes: one chain, negative tender rows, and a misplaced reason fixed

The hospitality refund flow (`RefundInvoice`) issues a full-refund ZATCA
credit note (TypeCode 381, BillingReference to the original, mandatory
reason), gated by kernel authority AND manager PIN, refusing agents at both
depths. Design points:

1. **One ICV/PIH chain per EGS unit, across document types.** ZATCA chains
   every e-document the unit issues, so the chain head is now read across
   `hosp_invoices` AND `hosp_credit_notes` inside the writer transaction
   (`chainHead`); `CloseSession` was rewired onto the same helper. The e2e
   test pins invoice → credit note → next invoice continuity.
2. **Refunds are NEGATIVE tender rows** (`Payment.CreditNoteID` set,
   negative amount, today's business date), so `ExpectedTenders` — and
   therefore the settlement-engine day close — reconciles the NET drawer
   movement. A same-day sale+refund closes at zero, pinned by test.
   Rejected: a separate refunds table the day close would have to join —
   the drawer doesn't distinguish, so the books shouldn't need two queries.
3. **Only PAID invoices refund, once** (status gate + UNIQUE
   original_invoice_id); the original flips to `refunded`.
4. **Found and fixed in the ZATCA module:** the credit-note reason rendered
   as free-text `cbc:Note`, but BR-KSA-17 validates it at
   `cac:PaymentMeans/cbc:InstructionNote` (KSA-10). Moved (PaymentMeansCode
   10 per the reference samples). Bytes changed for a shape NO deployment
   has ever issued (the vertical didn't expose credit notes until this
   commit), so the §8 already-issued-shapes stop-and-ask doesn't trigger.
5. No compliance event is published for credit notes — the events
   vocabulary only has `InvoiceCreated`, and publishing a credit note under
   that name would poison future consumers (revenue aggregation). A
   `CreditNoteIssued` event is Wave 4 residue, recorded, not smuggled.

**[Mirror]** "The module supports credit notes" is a claim about the
LIBRARY; wiring a vertical through it is what tests the claim — and it
promptly found the InstructionNote placement bug. When a document-type flag
exists but no caller has ever set it, treat the first real caller as the
feature's actual acceptance test.

## W3-D11. Stretch B.3/B.4: two lookup semantics kept apart; PDF seller identity onto the overlay

**B.3 (Excel header matching).** The three root implementations collapse
into `pkg/documents/excel/header.go` — but as TWO functions, deliberately:
`FindInHeader` (column-priority scan, import_2026_data's semantics) and
`HeaderIndex.Find` (variant-priority lookup, tally/etl's semantics). They
tie-break differently when a sheet carries more than one candidate column;
`TestColumnVsVariantPriority` exists specifically to stop a future
"simplification" from silently changing which column wins. Duplicate
headers keep the LAST occurrence (map-overwrite), matching the four
hand-rolled colMap loops the engine replaced. tally_importer's four loops,
etl_service's two inline builds + getCell closure, and findColumnIndex all
now delegate.

**B.4 (PDF de-contamination, minimum honest version).** The engine's
hardcoded seller block ("ACME INSTRUMENTATION W.L.L" + three address lines)
now renders from `overlay.Active()`'s default division profile — the same
pattern costing_engine already uses. Note: the hardcoded lines were DRIFTED
copies of the overlay profile's values (e.g. "Flat/Shop No.91" vs the
profile's "PO Box 0000"), so default-deployment PDFs now print the
overlay's (synthetic) values — rendering alignment with the SSOT, not a
financial change. The header adapts to 0–N address lines (pinned by test).
The three-generator-path unification remains Wave 4 residue.

**A.5 (domain-service peel) — deliberately NOT done as a separate step.**
B.1 already peeled the delete-approval flow's DECISION half behind
`pkg/approvals`; the remaining storage/notification half is glue with no
second consumer. Peeling GRN numbering just to tick the box would be
motion, not extraction. Recorded as residue instead.

**[Mirror]** De-duplication has a hidden failure mode: two functions that
LOOK like duplicates but tie-break differently are not duplicates, they are
two contracts. Before merging near-identical helpers, construct the input
where they disagree; if you can, they both survive — with a test that
documents the disagreement.
