# FABLE Wave 2 — Decisions Log (Composition Proof + GCC Deepening)

**Scope:** Wave 2 per `FABLE_WAVE2_HANDOFF.md` — Mission A (engine promotion +
second-vertical composition proof), Mission B (Saudi ZATCA compliance), Mission C
(agentic profile extraction). Same format as `FABLE_WAVE1_DECISIONS.md`: every
consequential decision recorded — what was decided, what was rejected, why.

Mission C annotations appear as **[Mirror]** notes: what a less capable model
would need to be told at this decision point.

---

## W2-D1. Banked the uncommitted 2026-06-15 session on the Wave 2 branch (not main)

**Found:** `main` was dirty — 18 modified + 6 untracked files, all from the
2026-06-15 "Phase 4 + loose ends" session (see
`docs/status/SESSION_2026-06-15_PHASE4_AND_LOOSE_ENDS.md`). That note says the
work is complete and green but "the push/commit is the Commander's to do next
session" — and 2.5 weeks later it was still sitting untracked on a dirty tree.

**Options:**
(a) leave the tree dirty and layer Wave 2 on top — rejected: three weeks of
uncommitted D5/D6/D7/D8 residue work would be indistinguishable from Wave 2's own
diff, and a single bad `git checkout --` away from destruction;
(b) commit directly on `main` — rejected: the session note explicitly reserved
that call for the Commander;
(c) **chosen:** create `feat/fable-wave2-composition-proof` and commit the work
there as the branch's first commits, grouped as the session note suggested.
`main` stays byte-identical at `70adcb6`; the Commander reviews everything at
merge time anyway.

**Deviation from the note's suggested 3-way grouping:** `app.go` contains BOTH
the division-SQL rewiring AND the FX-constant removal, so groups (b) and (c)
cannot be split without dissecting hunks of an already-verified diff. Committed
as two commits instead: (a) ADR-001 docs; (b) the config-extraction loose ends
(D5 division SQL + D7 contract terms + D8 branding + D6 EUR FX config-drive)
together — they were built and verified as one session, and splitting verified
work along artificial lines adds risk for zero information.

**[Mirror]** A less capable model would either commit straight to the dirty main
(violating the explicit reservation) or stash/ignore the changes (risking silent
loss of 3 weeks of residue work). The rule: *uncommitted work you did not author
gets identified, verified green, and preserved in git before your own work
starts — on a branch if authorship of main is reserved.*

## W2-D2. Fixed two date time-bombs in the Butler harness (test-only)

**Found:** `TestButlerBusinessHarness_GroundedQuestionBank/customer_quarter_invoices`
failed deterministically. The seed invoice was pinned to 2026-04-12 (Q2) while the
prompt asks "this quarter" — the suite was green on June 15 and broke on July 1
when the quarter rolled. Same latent class: the seeded offer (2026-04-08,
`Year: 2026`) feeds an "offers this year" assertion that would detonate 2027-01-01.

**Fix:** seed those two records at `time.Now().UTC()` (not `now - N days`, which
just moves the bomb to the first N days of each quarter). All other fixed dates
in the harness are left untouched — no time-scoped assertion reads them.

**[Mirror]** The failure log said "found no recorded invoices in Q3 2026" — the
signal is IN the failure text (Q3 = the run date's quarter, seed = Q2). A less
capable model pattern-matches "test broke → my changes broke it" and starts
reverting; the discipline is: *reproduce in isolation, read what the assertion
actually says, and check what changed in the WORLD (the date) before what changed
in the code.* Also: when fixing a date bomb, grep the file for the whole class
(`time.Date(`) — bombs come in batches.

## W2-D3. Second vertical = HOSPITALITY (`overlays/hospitality/`)

**Options:** hospitality (PP_Killer reference) vs professional services (CS-Invoice
reference). **Chosen: hospitality.** Why:

1. **Strongest composition proof.** The professional-services workflow
   (client → ledger → invoice → payment) is near-isomorphic to trading
   (customer → order → invoice → payment) — proving it composes would prove
   little. The hospitality shape (table session → order lines with modifiers →
   KOT kitchen states → settle → day-close) is genuinely alien to trading; if the
   kernel's Workflow/Policy/Actor primitives can express a KOT lifecycle and a
   PIN-gated void, the substrate thesis actually gets stress-tested.
2. **Mission B synergy.** A Saudi restaurant issues SIMPLIFIED invoices — exactly
   ZATCA's B2C path (self-generated QR with cryptographic stamp, reporting ≤24h).
   The composition proof can therefore exercise the Saudi compliance module
   end-to-end, making the two missions load-bearing for each other.
3. **Commercial relevance.** GCC restaurants were named by the Commander as a
   promising market; professional services would additionally point at Indian GST,
   which the substrate already has (`pkg/compliance/india`).

CS-Invoice is NOT wasted: its Excel-import shape, encrypted-backup format,
voucher-as-projection accounting, and PIN-lock pattern inform engine designs.

**[Mirror]** The trap for a less capable model is picking the vertical that
*looks easiest* (professional services, because it resembles the existing code).
The proof-value of a composition test is proportional to how DIFFERENT the second
domain is. Pick the vertical that maximizes information, not similarity.

## W2-D4. Sequencing: B-validation first, then A.1 engines, then B-deep, then A.2

The substrate map shows the compliance seam (`TaxEngine` interface + event bus +
per-division TRN dispatch) already exists and is proven by TWO implementations
(Bahrain, India). So the handoff's "design the shared Jurisdiction interface" is
already largely done — Mission B splits into a cheap part and an expensive part:

1. **B1 (first, ~additive):** `pkg/compliance/saudi/vat.go` implementing the
   existing `TaxEngine` + registration + SAR jurisdiction inference + tests.
   Early win; validates the interface against a 3rd jurisdiction immediately.
2. **A.1 (the enabler):** engine promotions in ROI order from the substrate map —
   document numbering, settlement/day-close (greenfield, PP_Killer-informed;
   the second vertical needs it), backup+restore, approval routing retrofitted
   onto the kernel, audit-log convergence. **Deferred:** Excel unification and
   PDF de-contamination (not on the composition proof's critical path; the two
   PDF libraries + 3 render paths need a deliberate consolidation wave).
3. **B2 (the real Saudi build):** ZATCA UBL 2.1 XML + crypto (SHA-256/c14n11,
   ECDSA secp256k1) + QR TLV + API client with sandbox/mock boundary, in
   `pkg/compliance/saudi/`.
4. **A.2:** `overlays/hospitality/` + composition root + synthetic seed + one
   end-to-end workflow (open session → lines → KOT → settle → simplified invoice
   → Saudi compliance validation → day-close).
5. **C:** synthesis (profiles, kernel v2 draft, progress + honest thesis %).

## W2-D5. Reference-codebase study notes stay OUT of the repo

The CS-Invoice / PP_Killer study reports contain detailed internals of Rahul's
private codebases. `asymmflow-oss` is headed for public GitHub. Study notes live
in the session scratchpad only; what enters the repo is the *re-implemented
pattern* (per invariant 8) and decision-log rationale at pattern level. The ZATCA
research IS committed (`docs/research/ZATCA_PHASE2_RESEARCH.md`) — it is public
regulatory information.

**[Mirror]** "Document everything" has a scope boundary: everything about *this*
codebase. A less capable model told to keep a decision journal will happily paste
a third party's proprietary architecture into a public repo. State the
information-flow rule explicitly: reference material flows in as patterns, never
as text.

## W2-D6. Composition proof boots via a separate `cmd/` composition root

**Options:** (a) graft the hospitality vertical into the existing Wails `App`
(feature-flagged); (b) a second Wails app; (c) **chosen:** a plain-Go composition
root (`cmd/hospitality/`) that boots `overlays/hospitality/` against
kernel+engines+overlay+SQLite, seeds synthetic data, and executes the full domain
workflow, verified by an end-to-end test and runnable as a CLI demo.

Why: (a) bloats the god-App the proof is supposed to escape and couples the proof
to trading's ~90-model migration set; (b) violates the spirit of invariant 4
(no Wails churn) and buys UI, not proof. The handoff is explicit that backend
composition is the proof and UI is skeletal-at-most. A `cmd/` root also *forces*
the engines to be genuinely importable outside `package main` — the compile
failure IS the extraction test.

## W2-D7. Saudi VAT engine implements the EXISTING TaxEngine interface; no new "Jurisdiction interface"

The handoff asked for "shared interfaces in pkg/compliance/compliance.go that
both Bahrain and Saudi implement". Investigation showed that interface already
exists — `compliance.TaxEngine` — proven by two implementations (Bahrain, India)
plus a registry, event hook, and jurisdiction inference. **Decision: implement
`pkg/compliance/saudi.SaudiVAT` against the existing interface unchanged**, and
resist redesigning a working seam. A third implementation is a better validation
of the interface than a rewrite of it. Additions kept additive: the
`JurisdictionSaudi` constant, a `SAR` currency-inference case, and registration
at both binding sites.

Design details worth recording:
- **Reverse charge** (imported services): `TotalAmount == BaseAmount` while
  `TaxAmount` carries the 15% self-accounted VAT with an explicit
  "VAT (reverse charge)" breakdown component — the supplier is not paid the VAT,
  but the buyer's obligation is visible to accounting.
- **`CategoryCode()`** maps categories to ZATCA's UBL codes (S/Z/E/O) now, so the
  Phase-2 XML generator (B2) consumes the same classification the calculator uses
  — one source of truth for category semantics.
- **Arithmetic mismatch is a WARNING, not an error** — a mixed-category invoice
  legitimately has tax ≠ 15% of base; only structural defects (bad VAT number
  format, negative amounts, bad rates) hard-fail.

## W2-D8. Overlay-driven jurisdiction routing via `JurisdictionCode()`

The invoice event already had an explicit-jurisdiction field, but the publisher
left it blank, so routing degraded to currency inference (SAR→SA works, but a
Saudi deployment invoicing in USD would silently route to Bahrain). Added an
optional `jurisdiction` overlay field with country-name fallback
(`overlay.JurisdictionCode()`), published on every `InvoiceCreated`. Precedence:
explicit config → country name → blank (= legacy currency inference).
Reference build still routes BH — pinned by test.

**[Mirror]** Two generalizable rules here. (1) *Before building the interface an
instruction asks for, check whether it exists* — the handoff was written from a
strategy session, not from the code; specs drift from repos. (2) When a routing
value can come from three sources, define precedence explicitly and pin the
default deployment's behavior with a test, or config additions silently change
production routing.

## W2-D9. Document-numbering engine: `pkg/documents/numbering`, write-first transaction

Promoted the 4 near-identical generators (INV/CN/PO/DN) to a generic engine:
`Spec{Prefix, Template, Pad, Seed}` + `Next()/NextInTx()/Render()`. Placement:
`pkg/documents/numbering` (documents is the doc-capabilities home and already
depends on GORM; `pkg/engines` stays computation-only). The `Sequence` model
binds to the existing `invoice_sequences` table so trading data carries over
untouched.

**Deliberate improvement over byte-identical:** the old pattern was
SELECT-FOR-UPDATE-then-save. On SQLite the locking clause is a no-op, so
concurrent allocations do a read→write lock upgrade — my concurrency test
(20 goroutines) failed with "database is locked" *before any rewiring*, i.e.
the old code's discipline was Postgres-correct but SQLite-fragile. The engine
issues the increment UPDATE as the transaction's FIRST statement (writer from
the start), which eliminates the upgrade-deadlock class; formats and
first-of-year seed queries are preserved byte-identically at the call sites.
The Offer NN-YY legacy scheme (`app_sales_pipeline.go`) is deliberately NOT
migrated — Wave 1 flagged it as coupled to OneDrive folder matching
(stop-and-ask); left as residue.

**[Mirror]** Writing the engine's concurrency test BEFORE rewiring call sites
surfaced that the pattern being extracted was itself broken under concurrency.
Extraction is the moment you get a free correctness audit of the original —
test the extracted engine harder than the original was tested, and when the
original fails your test, fix it in the engine and record it (never silently).

---

## W2-D10 — Mission B2: ZATCA Phase 2 module (UBL 2.1 + crypto + QR + API)

**Decision.** Built `pkg/compliance/saudi/` out to full ZATCA Phase 2 scope in
four files: `zatca.go` (invoice model, 7-digit subtype codes, totals, UBL 2.1
XML, XAdES-BES signing), `crypto.go` (secp256k1 ECDSA, invoice hash, manual
X.509 parsing), `qr.go` (TLV tags 1–9), `api.go` (Fatoora gateway client:
CSID onboarding, reporting, clearance). All tax math, XML generation and
crypto are production-grade and test-verified end-to-end (sign → verify
signature over SignedInfo → decode QR → all 9 tags round-trip).

**Key sub-decisions and why:**

1. **Canonical-emission hashing boundary.** ZATCA specifies C14N 1.1 of the
   invoice (with UBLExtensions/QR-ref/cac:Signature removed) as the hash
   input. Instead of implementing general C14N 1.1, the XML builder emits
   *already-canonical* XML deterministically and we hash our own
   pre-signature bytes — canonicalization is the identity transform on our
   own output. One builder emits both the pre-signature form and the final
   signed form (artifacts injected only when present), so the hashed bytes
   are the exact subset of the final document by construction. Full C14N of
   arbitrary third-party XML is explicitly out of scope (documented in
   `crypto.go`).
2. **secp256k1, not P-256.** ZATCA's security standard mandates secp256k1 —
   research flagged wrong-curve as the #1 signature-rejection cause. Go's
   stdlib rejects the curve, so: decred `dcrec/secp256k1/v4` (pure Go, no
   CGO — invariant preserved), DER-encoded signatures (matches
   OpenSSL/BouncyCastle reference implementations), manual ASN.1 SPKI
   assembly for QR tag 8, and a hand-rolled TBS-certificate element walk in
   `ParseCertificateDER` because `x509.ParseCertificate` refuses secp256k1
   SPKIs outright. (encoding/asn1 struct tags don't compose with RawValue
   for the context-specific `[0] version` element — hence the walk.)
3. **QR only on simplified invoices.** Standard (B2B) invoices get their
   stamp from ZATCA at clearance; we only mint the TLV QR for simplified
   subtypes, and `SubmissionResult.ClearedInvoiceXML` is documented as the
   thing to persist/share for cleared invoices (never our locally-built XML).
4. **API client boundary.** The HTTP client is a faithful implementation of
   the gateway contract (paths per environment, CSID basic-auth, OTP header,
   Accept-Version V2, Clearance-Status, 200/202/400/401/413 semantics
   including "202 = accepted WITH warnings, never resubmit") but has NOT
   been exercised against the live gateway — documented at the top of
   `api.go`. This is the sanctioned stub/sandbox boundary from the handoff:
   tax math + XML + crypto real; live onboarding needs Fatoora portal OTPs.

**Open items (flagged ❓ in `docs/research/ZATCA_PHASE2_RESEARCH.md`),
to confirm against ZATCA's official sample set before production onboarding:**
the exact XPath transform strings inside ds:Reference, and the precise
semantics of QR tag 9 (certificate signature bytes) for standard invoices.

**[Mirror]** A lesser model would reach for `crypto/x509` + `crypto/ecdsa`,
hit "unsupported elliptic curve", and either give up or silently switch to
P-256 (producing invoices ZATCA rejects). The load-bearing knowledge is:
(a) the curve is non-negotiable and stdlib-hostile, (b) hashing rules are
about *byte-exact canonical form*, so owning the emitter sidesteps an entire
C14N implementation, (c) gateway status codes carry business semantics
(202 ≠ retry). Encode all three in comments at the exact point of use, not
just in docs.

---

## W2-D11 — Mission A.2: the hospitality composition proof BOOTS

**Decision.** Built the second vertical as `overlays/hospitality/` (domain
package + `overlay.json`) with composition root `cmd/hospitality/`. It is a
Saudi café POS: order sessions → kitchen tickets (KOT state machine) →
manager-PIN void → signed ZATCA simplified invoice → payment → compliance
validation via the event bus → tender-reconciled day close. `go run
./cmd/hospitality` executes the whole day against synthetic seed data and
exits 0; `overlays/hospitality/e2e_test.go` pins it in CI.

**What the proof actually demonstrates (the thesis, itemized):**

- **Zero engine code duplicated.** The vertical imports numbering,
  settlement, pinlock, events, compliance, overlay, kernel money/actor —
  and writes only domain vocabulary (~5 files). The trading app is not
  imported anywhere.
- **The overlay IS the deployment.** Same `overlay.json` schema as trading;
  `jurisdiction: "SA"` + `currency: SAR/2dp` + division VAT number is all it
  takes to retarget compliance, money scale and seller identity.
- **Cross-engine composition is load-bearing, not decorative:** KOT and
  invoice numbers come from the SAME `invoice_sequences` table the trading
  documents use (keyed by prefix — shared by design); `CloseDay` is a thin
  wrapper over `settlement.Compute`+`Close`; PIN lockout state persists via
  the vertical's own settings table exactly as the engine's contract asks.
- **The AI-authority boundary holds at every authority-bearing call site**
  (void, invoice issue, day close) via kernel `actor.CanApprove` — the demo
  binary itself proves the agent is refused, then the manager succeeds.
- **Compliance is event-driven:** the vertical never imports a validator; it
  publishes `InvoiceCreated` with `Jurisdiction: ov.JurisdictionCode()` and
  the hook routes to the Saudi engine.

**Sub-decisions:**

1. **`saudi.NewSelfSignedCertificate` promoted from test helper** — offline
   demos/dev need a signing cert before ZATCA onboarding exists; documented
   as NOT gateway-acceptable. Its name encoding had to change from
   PrintableString to UTF8String: the very first boot failed on
   "Wasela Café" (é), a real-name bug the synthetic canon caught for free.
2. **`InvoiceCreated.Amount` = NET base, not total.** The first boot ran
   clean but the compliance hook warned "tax ≠ 15% of base" — I had
   published the VAT-inclusive total. Every registered engine treats
   `Amount` as the taxable base; fixed the publisher, warning gone. The
   compliance hook caught a real integration bug during the demo build:
   the seam works.
3. **ICV/PIH chain head is read inside the writer transaction** so two
   concurrent session closes cannot fork the invoice hash chain.
4. **Menu prices are NET halalas** (int64 minor units); floats appear only
   at the ZATCA XML boundary, which owns the 2dp rounding rules.
5. **hosp_ table prefix** everywhere except the deliberately shared
   `invoice_sequences`.

**[Mirror]** The two integration bugs (é in PrintableString, total-vs-base
Amount) were both invisible statically and both surfaced within seconds of
BOOTING the composition. This is why the handoff demands the proof boot
rather than merely compile: composition bugs live in the seams, and seams
only exercise at runtime. A lesser model should be instructed: after wiring
a second consumer of any engine, RUN an end-to-end scenario before writing
more code — the first run is the cheapest audit you will ever get.
