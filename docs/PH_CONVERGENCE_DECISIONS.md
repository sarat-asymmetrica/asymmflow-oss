# PH Convergence — Decisions

Campaign decision log (PC-D#), written when decided, per Mission F of
`FABLE_CAMPAIGN_PH_CONVERGENCE.md`. `[Mirror]` paragraphs record what
generalizes beyond this campaign.

## PC-D1 — Party deletes compose the child-count guard WITH the approval workflow (Commander, 2026-07-05)

OSS's `guardDeleteOrRequest` answers only *who may delete*; PH's
child-count guard answers *whether this party is safe to delete at
all*. They are orthogonal, so we take both: the child-count check lives
inside `pkg/crm/customer/delete.go` (the pkg owns existence
verification, so it protects every caller including
`performApprovedDelete`), returning `CUSTOMER_HAS_LINKED_RECORDS` /
`SUPPLIER_HAS_LINKED_RECORDS` with exact counts; the approval workflow
stays layered above in the app wrapper.

[Mirror] Authorization guards and integrity guards look alike (both say
"no") but answer different questions; composing them beats choosing.
The integrity half belongs in the engine, the authorization half at the
host — an approved request must still fail if the operation is unsafe.

## PC-D2 — Invoice PDF: all four measured divergences port (Commander, 2026-07-05)

Division-bank filter, buyer-address Attention* fallback, 40mm top
margin, reference-number truncation. Gated as customer-facing financial
appearance; the Commander chose full parity — required anyway for
Mission E's byte-identical reconciliation. The signature block is
already substantively equivalent and stays as-is.

[Mirror] When the campaign's terminal test is byte-identity, "cosmetic"
divergences are not cosmetic; they are reconciliation failures paid
later. Port appearance with the same discipline as arithmetic.

## PC-D3 — Postgres sync hardenings port; Turso/CDC decision deferred to cutover (Commander, 2026-07-05)

PH runs Postgres sync today, so its path stays robust through
convergence: varchar→TEXT widening, NULL-PK UUID backfill before sync,
`SKIP_REMOTE_MIGRATION`. Choosing CDC later wastes nothing; leaving the
live path fragile through the convergence window risks the actual
deployment.

[Mirror] Harden the path you are standing on, even if you plan to leave
it. Migration-era robustness work on the legacy path is not waste; it
is the insurance that lets the migration be unhurried.

## PC-D4 — Bank reconciliation adopts PH's refuse-until-reopened (Commander, 2026-07-05)

An audited final state (Reconciled/Verified) must not silently
un-finalize. OSS's banking engine auto-reverted finalized statements to
InProgress on line change; it will instead refuse edit/delete/match/
line-change until the statement is explicitly reopened, with the status
normalizer and editable-status whitelist from PH's policy seam. This
ratifies the fork in PH's favor and replaces the substrate's
auto-revert.

[Mirror] Ergonomic auto-transitions out of terminal states are
integrity leaks wearing convenience's clothes. Terminal means terminal;
reopening is a deliberate, auditable act (same posture as W6-D1's
refuse-don't-clamp for services).

## PC-D5 — Band-2 policy seam ports as pkg engines, per the ledger decomposition (ratified by Wave 1 sign-off, 2026-07-06)

The April write-policy seam is not one unit; the verified ledger split
it four ways and each fate held: bank-recon lifecycle was decided at
PC-D4; seed catalogues are overlay config (Mission D); identity-write
core lifted into `pkg/crm/customer` (write.go) with the three gaps
closed in passing; product-supplier linkage and valuation/weighted-
average costing ported as new engines (`pkg/crm/supplierlink`,
`pkg/inventory`). The Commander reviewed Wave 1's residue plan naming
exactly this scope and ordered Wave 2 on it — recorded here as the
ratification of the stop-and-ask registry's "Band-2 write-policy-seam
decision".

Sub-decisions taken inside that frame, each stated in its commit:
- **Alias vocabulary is config, not code.** The supplierlink engine
  takes an injectable `AliasConfig`; PH's real principal alias tables
  never enter this repo. Default carries only the synthetic SVX→SRVX
  already in the seed canon.
- **Seed products stay unlinked on resolution failure** (PH skips the
  product). OSS supplier seeding is user-invoked, so skipping would
  silently empty the catalog; an empty link is honest and readers
  resolve lazily through the same engine.
- **Supplier Rating 0 falls back on update** (the policy behavior)
  rather than PH HEAD's unconditional copy — a zero from a partial
  payload must not wipe a real rating. Same G1 non-destructive posture
  as the customer metric mask.

[Mirror] A "policy seam" accumulated in a god object is usually several
seams. Measure it behavior-by-behavior before porting; the fates
diverge (engine / config / already-subsumed / genuine fork), and
porting it as one unit would have forced the wrong answer on three of
the four.

## PC-D6 — Importer carries everything it can name, and shouts about the rest (2026-07-06)

Mission C's importer copies with intersected explicit column lists in
PH's dependency order, blanks machine-bound HMAC hashes for
destination-salt recompute, and refuses to carry machine-bound
ciphertext. Nothing is silently dropped: peeled tables are skipped WITH
row counts and reasons, `customer_receipts` is marked PENDING DECISION
(transform-into-payments vs drop — Commander call before any Mission E
rehearsal), and source tables outside the copy set are reported
UNMAPPED. A foreign-key check gates the commit; violations roll the
whole import back.

[Mirror] A migration report that only lists what moved is half a lie.
The load-bearing half is what did NOT move and why — silent truncation
reads as "covered everything" precisely when it wasn't.

## PC-D7 — customer_receipts transform into payments (Commander decision, 2026-07-06)

The open question from PC-D6 is decided: **transform**. Measurement
first: PH's `receipt_service` creates a real `payments` row for every
receipt allocation at apply time, and `payments` is already in the
importer's copy set — so the APPLIED portion of every receipt arrives
with the straight copy, and transforming it again would double-count
cash. What has no payments row is the UNAPPLIED on-account remainder.

The transform therefore carries exactly that: each live receipt
(`deleted_at IS NULL`, status ≠ Reversed) with an unapplied remainder
> 0.0005 BHD becomes one invoice-less payment row — id and idempotency
key deterministic on the receipt id, method normalized into the
payments check-constraint vocabulary (port of PH's
`normalizeCustomerReceiptMethod`), amount copied exactly from
`unapplied_amount_bhd` with no arithmetic, and the receipt number +
customer id packed into `reference` (OSS payments has no customer
column) so on-account money stays traceable to its customer. The
report accounts for every receipt: transformed / fully-applied
(already represented) / reversed-or-voided.
`customer_receipt_allocations` are skipped as "represented" — their
money IS the copied payments rows.

[Mirror] "Transform X into Y" is underdetermined until you measure
what already flows into Y through another pipe. Here the naive
transform (every receipt → a payment) would have double-counted every
applied bahraini fils; the correct transform carries only the residue
the existing copy cannot see.

## PC-D8 — Mission D is a closure sprint, not a build (2026-07-06)

Measurement (full-repo survey) showed the overlay seam ALREADY existed
and was live: divisions/branding, VAT default, FX rates, business-rule
numbers, product markups, jurisdiction routing, and seed-set gating all
read `overlay.json` through `pkg/overlay`. Mission D therefore closed
the residual company-facts-in-code gaps instead of building a mechanism:

- **Supplier alias vocabulary** → `overlay.supplier_aliases`
  (`SupplierAliasVocabulary()`), wired into `supplierLinkAliases()`.
  Nil = built-in default (the one synthetic SVX→SRVX alias); explicit
  empty = cleared. Real principal catalogues are sovereign-overlay facts.
- **License key prefix** → `overlay.license_key_prefix`
  (`LicenseKeyPrefixOrDefault()`, blank → "PH"). Generation, format
  validation (length derived from prefix), and the reseed dev-key filter
  all read it; the built-in default keeps every existing 13-char
  activation valid byte-identically.
- **Banking division normaliser consolidated** into
  `overlay.NormalizeDivisionName`. This deleted the last real-company
  residue in engine code (legacy "a h s trading w.l.l" spellings) —
  historic division spellings are DATA, declared under
  `divisions[].aliases` in the sovereign overlay. Deviation noted: rows
  carrying those spellings normalise to the default division unless the
  deployment declares the alias.
- **VAT default chain unified**: the settings layer's three literal 10s
  now fall back to `overlay.DefaultVATRate` (user setting → overlay →
  built-in 10), and the VAT-return CSV label renders the overlay rate
  (byte-identical "(10%)" for the default — the label is appearance of a
  regulatory export, so this was done only because the default bytes are
  unchanged; a different-rate deployment's return stops lying).
- **Letterhead asset key** comes from the overlay division profile, not
  an `if division == "Beacon Controls"` comparison.

Deliberately NOT moved (flag register in docs/PH_SOVEREIGN_FORK.md):
DeliveryTerms GORM default (schema DDL + offer-text path, needs its own
golden-first pass), demo bank/product/supplier seed rows (fixtures gated
by seed_sets — the config seam is the gate, not the rows), legacy
importer alias maps (superseded by the phimport flow). Jurisdiction VAT
constants stay in pkg/compliance: the overlay picks the ENGINE, the
engine owns its statutory rate.

[Mirror] "Extract X into config" starts with measuring how much of X is
already config — porting a plan instead of the residual gap would have
rebuilt a live mechanism beside itself. And the boundary that survives:
config owns company facts and picks engines; engines own statutory
facts. A Saudi VAT rate is not a company fact, so it stays code.

## PC-D9 — Mission E rehearsal: the fresh-provision path is part of the product (2026-07-07)

The first end-to-end rehearsal (OSS binary → fresh DB → phimport from a
PH snapshot copy → reconcile) found and fixed three classes of ground
truth, all invisible until someone actually walked the documented path:

- **Fresh provisioning was incomplete.** `chart_of_accounts` and
  `account_mappings` were AutoMigrated only in tests; the expense
  foundation *ensures* accounts in them, so a fresh database hard-exited
  at startup (refuse-to-run working as designed). Mature databases never
  noticed because the tables always pre-existed. Both are now in
  `criticalDeploymentModels()`. Banking-suite tables (`bank_statements`
  et al.) are still not provisioned on fresh files — carried as an open
  finding, since PH's live rows there are few (2 statements / 12 lines).
- **Seeded-baseline collisions.** A freshly provisioned destination
  already contains the app's own skeleton (ensured accounts, seeded
  categories/bank fixtures/assets/roles) whose unique keys collide with
  the imported company's rows — and imported children reference the
  SOURCE row ids. `replaceTables` in phimport now replaces the
  provisioned baseline wholesale for exactly those tables, counted in
  the report, sound only under the tool's fresh-destination contract.
- **Legacy vocabulary in old rows.** PH data carries PO status
  `'Completed'`, which PH itself normalises to `'Closed'` on read but
  the OSS CHECK constraint rejects on write. `columnTransforms` in
  phimport ports PH's own read-side normalisation (never an invented
  mapping); unknown spellings pass through so the CHECK stays the last
  line of defence.

Rehearsal result: 25/25 reconciliation checks match to the fils
(invoices 480 / VAT / outstanding / per-status / per-year, payments,
supplier invoices+payments, items, orders, offers, GRNs), referential
integrity of replaced tables intact, app boots on the imported file
with zero errors and recomputes all 480 invoice hashes under the new
install's salt. Cutover itself remains Commander-gated.

[Mirror] A migration tool's contract with its destination ("freshly
provisioned") is what makes destructive-looking operations (replace the
baseline) safe — state the contract, then lean on it. And: reconcile
with queries that distinguish "both sides equal" from "both sides threw
the same error", or a missing column reads as a match.

---

# Wave 4 (Mission G — "Prove the Parity, Certify the Cutover"), 2026-07-07

Decisions taken while producing `docs/PH_PARITY_MAP.md` and closing its gaps.
Four were put to the Commander (§6 stop-and-ask); the rest this document or the
ground decided. See `docs/PH_PARITY_CERTIFICATE.md`.

## PC-D10 — Customer receipts: ratify payments-only (Commander, Q3)
PH has a runtime Customer-Receipt entity (receipt → many-invoice allocation +
receipt numbering). OSS folded receipts into `payments` at import time (PC-D7)
and neither repo ships a receipt UI. **Decision: ratify payments-only** as the
model; no runtime receipt entity ported. Recorded as DIVERGENT-INTENTIONAL.

[Mirror] When two systems model the same event at different grains, converge on
the one the UI actually needs — a backend entity with no screen and no caller is
scope, not parity.

## PC-D11 — Supplier AP cash-control gate: tighten to the OSS lifecycle (Commander, Q1)
OSS let a supplier invoice be PAID while `Pending` (never matched/approved) and
APPROVED unless flagged `Discrepancy`; PH requires a clean 3-way match.
**Decision: tighten** — pay requires `Approved` or `Verified`; approve requires
`MatchStatus == Matched`. While pinning this, the probe surfaced a **latent
fresh-DB bug**: the `supplier_invoices` status/match_status CHECK constraints
excluded `Verified`/`Review Required`/`Disputed` — the very values the code
writes — so a clean match could not persist on a from-zero DB. Widened both
CHECKs to the code's own vocabulary (kept `Dispute` for imported-data compat).

[Mirror] A relaxed control and a too-strict schema constraint are the same class
of bug seen from opposite ends: the code's written vocabulary and the schema's
allowed vocabulary must be the same set, or one of them is lying. Test the round
trip (write the value the state machine produces, read it back) — a status the
code emits but the table rejects is invisible until a fresh provision.

## PC-D12 — Fresh-provision the finance-reconciliation suite, create-if-missing (ground)
15 bank-recon/FX/VAT-return models were compiled and wired into live services but
in no boot migration set — a fresh OSS DB never created them (the single largest
provisioning gap). Registered them in `criticalDeploymentModels()`. First pass
added them unconditionally; that would have tripped a FOREIGN KEY check on the
*live* deployment DB at boot (re-migrating a table that already holds rows).
**Decision: create-if-missing** (`shouldSkipCriticalAutoMigrate` skips when the
table exists) — provisioning parity on fresh DBs, no rebuild on mature ones.

[Mirror] "Provision every table" and "never disturb existing data" are both true
at once only if provisioning is create-if-missing. An unconditional AutoMigrate
in an always-run startup path is a loaded gun pointed at every mature deployment;
the runtime deployment-DB test (not the fresh-DB test) is what caught it — keep a
test that runs against a copy of real production state.

## PC-D13 — Customer-facing PDFs: watermark yes, signatures keep-OSS-and-register (Commander, Q2)
Every counterparty PDF diverges in its signature block (`offer_signature_blocks.go`
never ported); PH's PO PDF also watermarks unapproved POs. **Decision:** port ONLY
the PO DRAFT/PENDING "NOT VALID FOR SUPPLIER ISSUE" watermark (an integrity control,
not signer identity); leave signature blocks as OSS's generic form and register the
appearance divergence. Porting PH's named prepared-by blocks would risk
reintroducing real PH people into source unless overlay-driven (invariant #2).

[Mirror] Separate the integrity content of a document (a "not valid" stamp) from
its identity/appearance (whose name signs it). The first is a control you can port
freely; the second carries real-person data and belongs in overlay config, never
hardcoded.

## PC-D14 — Party-delete: the pre-ratified divergence is stale; it is PRESENT (ground)
Handoff §3.2 pre-ratified party-delete orphaning as a KEEP-OSS DIVERGENT-INTENTIONAL,
on the premise "OSS orphans a party's transactional children." **That premise is
stale** — Wave 1 (PC-D1) already moved the identical `*_HAS_LINKED_RECORDS`
child-count guard into `pkg/crm/customer/delete.go`. OSS blocks the orphaning delete
exactly as PH does. **Classified PRESENT (parity), not divergent.**

[Mirror] A handoff's "already decided, don't re-ask" list is only as fresh as the
snapshot it was written against. Re-measure a pre-decided call before acting on it
when prior waves may have moved the ground — the ground wins, the mirror records why.

## PC-D15 — OneDrive discovery-walk cluster: defer to Mission H (Commander, Q4)
~18 discovery/disambiguation functions (batch containers, nested/direct-file deals,
section aliasing, offer/folder-# collision disambiguation) are ABSENT; core import
works. They matter only for a live OneDrive re-scan, which belongs to the data wave.
**Decision: defer to Mission H**; close the small D1/D2 digit-guard residue now.

[Mirror] Draw the parity/data boundary by *when the code runs*: logic exercised only
during a data re-scan is data-wave work, not parity work, even when it lives in the
engine.

## PC-D16 — extracted_documents: SKIP with reason (Mission H, measured)
The 359-row table is flat OCR-scan metadata from the historical OneDrive extraction
sweep. Measured: no blobs, no FK references, and **nothing in PH reads it** — its only
code reference is a sync-coverage *exclusion* list. The rows survive in the archived
cutover snapshot, and the at-cutover OneDrive re-scan (PC-D15) supersedes them.
**Decision: skip-with-reason**, alongside the same-session rulings for `*_backup`
(point-in-time copies), `intelligence_*` (recomputable enrichment scaffolding),
`sqlite_stat1` (ANALYZE stats), and `collaborative_pending_operations` (transient
op-queue). Every skip carries its honest row count.

[Mirror] "Does PH's own app read it?" is the faithful-carry test. A column or table
the source system itself can no longer see is not data the migration owes the
destination — but it must be counted, reported, and preserved in the archive.

## PC-D17 — Column drift: credit-note VAT rename + the drop ledger (Mission H)
A full drift scan of every carried table (source columns missing in the destination)
found ONE real money drift: PH persists `CreditNote.VATBHD` as `vat_bhd`, OSS as
`vatbhd` — the intersect copy would have silently dropped credit-note VAT (0 rows
today; any future credit note would have lost it). **Decision: explicit
`columnRenames` pair in phimport**, applied only where PH's own model reads the
source spelling. Every other drifted-and-populated column (legacy customers
contact fields superseded by the modern columns, `order_items.brand/token`,
extraction-era `costing_sheet_data` aggregates, `invoices.notes` breadcrumbs,
`supplier_payments.payment_number`, `customer_contacts.salutation`) is dead in PH's
own code — dropped faithfully but now REPORTED per-column with non-empty counts in
the import report (`column_drops`), forming Mission I's work-list.

[Mirror] Intersect-by-name copies lie twice: they drop renamed columns silently and
they drop dead-but-populated columns silently. Diff the schemas, measure the data,
rename what the source still reads, and report the rest.

## PC-D18 — Demo bank fixtures seeded on every boot: gate enforced at the seam
The Mission H dress rehearsal's POST-BOOT reconcile (a check Mission E never ran)
caught `company_bank_accounts` growing 8 → 13 after first startup on the imported
file: `seedCompanyBankAccountsInternal` ran unconditionally from four call paths,
so a sovereign deployment's first boot injected five synthetic-IBAN demo accounts
next to the company's real ones — demo bank details one dropdown away from a real
invoice. The flag register had always *claimed* the `demo-bank` seed-set gate;
it was never enforced. **Decision: gate inside the internal seed function** (all
callers covered; default overlays with nil seed_sets keep seeding, demo installs
unaffected), pinned by `TestDemoBankSeedRespectsSeedSets`.

[Mirror] Reconcile AFTER the first boot, not only after the import — provisioning
and startup "ensure" paths are writers too. A dress rehearsal that stops at the
import gate certifies half the pipeline.

## PC-D19 — Mission I Wave 6: latent-PH bugs fixed forward in OSS (2026-07-09)
Mission I's audits (I-11 RBAC sweep, I-12 destructive-save) found defects
that exist byte-identically in DEPLOYED PH: (1) UpdateOrder's item
replacement wipes OrderItem.QuantityShipped/QuantityInvoiced on every order
edit — fulfillment tracking silently lost; (2) SeedLicenseKeys /
SeedEmployeeKeys are frontend-bound with no RBAC gate — any bound caller can
mint authentication credentials; (3) MarkCustomerInvoicePaid zeroes the
balance with no Payment record. **Decision (spec-authorized, surfaced for
Commander review): fix forward in OSS, do NOT patch frozen PH** — the Freeze
Law stands, PH is superseded at cutover, and the cutover runbook re-keys
credentials anyway. The OSS fixes carry regression tests; PH's copies of
these bugs are listed here so the parallel-run watch knows order edits and
mark-paid on the PH side are lossy during the overlap window.

[Mirror] An audit that only checks "does OSS match PH" certifies PH's bugs
into the substrate. Parity is the floor, not the ceiling — measure both
trees, fix forward, record where the deployed twin stays broken.

## PC-D20 — D-I-4: absent-model dispositions (Commander, 2026-07-09)
Of the 4 PH models ABSENT in OSS, **`costing_sheet_attachments` ports NOW**
(Mission I Wave 7, unblocks I-25 PDF datasheet bundling — customer-facing
offer completeness). **`extracted_documents`, `data_quality_reviews`,
`employee_archive_requests` are DEFERRED, not closed**: full port specs are
recorded in `docs/MISSION_I_DEFERRED_MODEL_SPECS.md` and they will be
tackled across subsequent waves — the handover target is a complete system,
so the queue drains until done. This sprint's completion line is I-25.

[Mirror] Decision-gated backlog items get one of three states — port now,
defer-with-spec, or formally-closed — never a silent drop. A deferred item
without a measured spec is a future re-measurement tax.

## PC-D21 — D-I-6 + D-I-7: Band-3 data dispositions (Commander, 2026-07-09)
Ground measurement overturned the spec's premise: the "12 wrong-date
invoices" are NOT year typos — all 12 are live-FY2026 Drafts from genuine
Won offers sharing one placeholder date (2026-05-20); the spec's -1yr remedy
would have misdated them into the closed FY2025 book. **D-I-6 decision:
anchor to source dates** — 3 from the source Quote Date, 7 from the linked
offer's real quotation date, 2 unanchored keep the placeholder (flagged for
SPOC); due dates recomputed +30d; FY2026 preserved. **D-I-7 decision: apply
the measured dispositions on the 35 no-item offer shells** — DELETE 20
duplicate/junk stubs (line items live in their populated twins), ARCHIVE 11
real empty 2026 quotes as pipeline history, REPAIR 4 from source costings
(one more than the spec's 3 — SA-05-SULB was priced in source but imported
as zero). All changes applied to the out-of-repo shadow DB with a pre-fix
backup and a per-row before/after audit trail
(`mission_i_data/band3_fix_report.json`); repairs must reconcile to the fils
or stay un-repaired.

[Mirror] A data-quality remedy computed from the flag ("date > FY") rather
than the record's lineage (what the row links to) can be worse than the
defect. Anchor fixes to provenance, not to the audit rule that fired.

**Outcome (applied 2026-07-09, shadow DB, backup + per-row audit trail):**
10 invoices anchored / 2 kept-and-flagged for SPOC (their offer date IS the
placeholder); 20 shells soft-deleted / 12 archived (stage=Expired — the
app's real lapsed-quote mechanism) / 3 repaired to the fils (delta 0.0000)
/ 1 (SA-05-SULB) NOT repaired — its items are absent from the migration
snapshot and its value exists only in an external EUR sheet with no
reconcilable BHD total, so it was archived + flagged rather than fabricated.
Post-fix reconcile: 60/62 with exactly the 2 expected deltas (repaired
offer_items +36; live offer count −20, value identical), 0 unexpected.
Follow-through: the startup audit's hardcoded `>= 2026` rule was replaced
with future-dated-vs-today (`countFutureDatedInvoices`, golden-tested) so
the fixed dataset doesn't re-flag forever.

## PC-D22 — D-I-5: enrich back ALL 19 dropped import columns (Commander, 2026-07-09)
The I-32 ledger captured 19 source columns the importer drops, all carrying
real data (referenced by shape only; exact non-empty counts live in the
out-of-repo `mission_i_data/import_report.json`). **Decision: enrich all
four groups back** — customer/contact reach-data, order-item Brand×Token
product identity + unit price, document references (supplier payment
numbers, invoice notes), AND the costing OCR-extraction metadata. Nothing
is closed-as-lost. Implementation: phimport gains the 19 mappings (+ dest
schema where missing), shadow DB re-enriched, Band-3 fixes preserved,
reconcile re-run. Rationale: the handover target is a complete system —
the importer is what runs at cutover, so it must carry everything.

[Mirror] A "dropped columns" ledger is only half a decision artifact — the
counts of non-empty rows are what turn "schema mismatch" into "219 customer
emails lost." Capture the density, not just the names.
