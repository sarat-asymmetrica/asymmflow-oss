# India Spec 01 — The Indian Invoice (a third jurisdiction on the one seam)

**Mission:** make AsymmFlow emit a legally complete Indian GST tax invoice and a portal-uploadable GSTR-1 JSON — with **zero external APIs, zero recurring cost** — by mounting India as a new jurisdiction plane on the existing compliance/division seam. This is the proof wave for the India track: after it, the sentence "two jurisdictions (GCC + India), three verticals, one kernel" is a demonstrable artifact, not a pitch line. The strategy doctrine is **minimum-lovable-compliance** (see `../INDIA_COMPLIANCE_DOSSIER_2026-07-22.md` for the full research substrate): the smallest slice an Indian SMB's accountant would call *correct*, not the largest slice we can imagine.

**Sequencing:** parallel India track, first wave. Builds on main (post-DP1 planes contract, post-Wave-12.5 division emission). Does NOT wait for, or interfere with, the GCC track (Wave 10 Sensory & Brand, PH work).
**Repo:** `asymmflow-oss` (PUBLIC — synthetic names ONLY, no real Indian businesses, no real GSTINs/PANs; construct test GSTINs from synthetic PANs). **Branch:** `feat/fable-india-w1-indian-invoice` off `main`. No merge, no push, no tag.
**Operating model:** Opus 4.8 orchestrator + Sonnet 5 coders; orchestrator gates; owner final-gates the pasted report.
**Authority docs:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` → shipped Wave 9–12.5 behavior + DP1 planes contract → this spec. Where this spec conflicts with observed plane/registry architecture, the architecture's *pattern* wins and the deviation goes in the report.

**Lessons inherited (do not relearn):** config-not-constant for anything a government can renotify (thresholds moved 5x since 2020) · canonicalize BOTH sides of every comparison (Spec-07 law) · byte-identical non-target behavior is the refactor acceptance test (Wave 12 law) · gate baseline: vite clean, svelte-check 0 errors, go build/vet clean, `go test -count=1 -timeout 1800s ./...` green, suite run ALONE · severity honesty in the report is law · stop-and-report on anything touching posting logic — **zero posting-change authorizations this wave**.

---

## 0. Compliance ground truth (embedded; the build must match THIS, not training-data memory)

Facts researched 2026-07-22 from live sources; confidence flags preserved. The dossier at `../INDIA_COMPLIANCE_DOSSIER_2026-07-22.md` holds sources and the fuller ledger.

**G1 — GSTIN & registration.** GSTIN = 15 chars: 2-digit state code + 10-char PAN + entity code + "Z" + check digit. One GSTIN per state under one PAN (s.25(2) CGST Act); each GSTIN is legally a **distinct person** — inter-GSTIN transfers are taxable supplies needing real invoices (out of scope this wave; the *model* must not preclude it). **AATO (aggregate annual turnover) is PAN-level across all sibling GSTINs**, not per-GSTIN.

**G2 — Tax split.** Intra-state supply (supplier state == place of supply) → **CGST + SGST, each exactly half** the applicable rate. Inter-state → **IGST at full rate**. Place of supply for goods = generally the delivery destination state. SEZ supplies are ALWAYS inter-state/zero-rated even within the same state (IGST Act s.16) — this wave: model the flag + zero-rating; LUT workflow itself is out of scope.

**G3 — Rates.** GST rate schedule (0%, 5%, 12%, 18%, 28% + cess) is **data, not code** — rates attach to HSN/SAC via configurable tax-category records. Do not hardcode a rate table into logic; seed a synthetic-demo rate set.

**G4 — HSN digit mandate** (N/N 78/2020-CT, PIB-confirmed current): AATO ≤ ₹5cr → 4-digit HSN mandatory on B2B invoices (optional B2C); AATO > ₹5cr → 6-digit on ALL invoices; 8-digit on exports always. Tier derives from **PAN-level** AATO. Validate at document-creation time; violation = refuse-to-generate with a clear message (house doctrine).

**G5 — Rule 46 invoice fields (all 16):** supplier name/address/GSTIN · sequential invoice number, **max 16 chars**, unique per FY · date · buyer name/address/GSTIN (B2B) · place of supply (state name + code) · HSN/SAC per line · description · quantity + unit (UQC) · taxable value · rate per line · tax amount split CGST/SGST/IGST/cess · total value · **reverse-charge statement** ("tax payable on reverse charge basis" — a flag rendered as text, not a doc type) · supplier signature slot. Also carry **bill-to vs ship-to GSTIN as separate fields** (mandatory in e-invoice/EWB APIs from 1 Aug 2026 — future-proofing now costs nothing).

**G6 — Composition dealers** (≤₹1.5cr goods / ₹50L services) issue a **Bill of Supply**: no tax lines, no ITC, mandatory legend "composition taxable person, not eligible to collect tax on supplies." This is a genuine second document type, not a zero-rate invoice.

**G7 — GSTR-1** = the outward-supplies return. This wave ships it as **portal-uploadable JSON** (the free offline-utility path — zero API cost). Sections in scope: B2B, B2CS (consolidated small B2C), B2CL (large inter-state B2C), CDNR/CDNUR (credit/debit notes), HSN summary, document-series summary, NIL/exempt. **Phase A verifies the current JSON schema against the official GST-portal offline tool** — the schema is versioned by GSTN and blog posts about it go stale (this includes verifying the current B2CL value threshold; sources conflict between ₹1L and ₹2.5L).

**G8 — E-invoicing (IRN/QR) is OUT of this wave.** Threshold is AATO > ₹5cr and the free portal path covers below-threshold businesses entirely. BUT: store the config parameter (`einvoiceThresholdAATO`, default ₹5cr) and surface an **applicability indicator** (display-only) so Wave IN-W3 mounts cleanly. The threshold has moved five times — config, never constant.

**G9 — Fiscal year is April–March** for everything (invoice-number series reset, GSTR periods, AATO computation). FY start month must be a per-book/per-overlay setting, defaulting to April for the India plane and leaving existing GCC books untouched.

**G10 — Currency/format.** INR, ₹, and **Indian digit grouping (lakh/crore: 12,34,567.89)** on India-plane documents. Amount-in-words in Indian convention ("Twelve Lakh Thirty-Four Thousand…") — invoice-total-in-words is customary on Indian invoices.

---

## 1. Phase A — recon (read-only; census in the report)

| # | Question | Feeds |
|---|---|---|
| A1 | **The plane/seam map.** How does the DP1 planes contract + compliance seam actually mount a jurisdiction today (ZATCA/Bahrain)? Where do: tax computation, document field schemas, PDF emission, VAT-return generation, and division-registry identity fields plug in? Deliver the "mount points" list this wave will use — the India plane must ride these seams, not fork beside them. | B1–B5 |
| A2 | **Division/identity fields.** What does a `DivisionProfile` / overlay identity carry now (TRN, letterhead, bank facts, Wave-12.5 emission fields)? Design the India additions: GSTIN, PAN, state code, composition flag, FY start — as plane-scoped extensions that are **absent/inert for GCC overlays**. Where does PAN-level AATO aggregation live (a computed roll-up over sibling divisions sharing a PAN)? | B1 |
| A3 | **Document pipeline reality.** How are invoice/credit-note documents modeled and rendered (fields, numbering series, PDF layer)? What's the smallest correct way to add: per-line HSN + UQC, place-of-supply, CGST/SGST/IGST tax lines, reverse-charge text, bill-to/ship-to split, a second doc type (Bill of Supply), and 16-char FY-scoped numbering — without disturbing GCC document output byte-for-byte? | B3, B4 |
| A4 | **GSTR-1 schema verification.** Obtain the current official GSTR-1 offline-utility JSON schema (GST portal). Record version, section list, field names, the current B2CL threshold, and any surprises vs §0 G7. This is the "verify the probe" step — the export in B5 is built against THIS artifact, not against blog posts. | B5 |
| A5 | **Numbering & FY plumbing.** How do numbering series work today (GRN fix history, per-division series)? What breaks if a series resets on April 1 instead of January 1? Where is "fiscal year" currently assumed = calendar year? | B2 |
| A6 | **India demo overlay.** Design the synthetic India overlay: one synthetic Indian company, 2 divisions in different states (inter+intra-state demos), synthetic GSTINs built from a synthetic PAN, a small HSN'd product set, one composition-scheme customer scenario. Throwaway synthetic names; public-repo law. | AC |

## 2. Phase B — the build

**B1 — The India jurisdiction plane.** Mount India on the seam per A1/A2: plane-scoped identity fields (GSTIN, PAN, state code, composition flag), GST state-code registry (01–38, names+codes, data file), PAN-level AATO roll-up (computed, feeds G4 tier + G8 indicator), FY-start-month setting (G9). **GCC overlays must not observe any change.**

**B2 — Numbering + FY.** FY-scoped sequential invoice numbering, ≤16 chars, per division, resetting at FY start (April for India plane). Calendar-year books (GCC) untouched. AC: series demo across a synthetic April 1 boundary.

**B3 — The GST tax engine.** Given supplier division (state), place of supply, line items (HSN, qty, UQC, taxable value, tax category): compute per-line CGST/SGST or IGST (G2), honoring SEZ zero-rating flag and reverse-charge flag (computation unchanged; liability notation only). Rates from config data (G3). HSN digit validation by PAN-level tier (G4) — refuse-to-generate on violation with a message naming the rule. Pure functions, table-driven tests: intra, inter, SEZ-same-state, composition, cess, rounding (statutory rounding per invoice for tax totals — verify the exact rounding rule in A4/A3 recon; flag in report).

**B4 — Document emission.** (a) India tax invoice: all 16 Rule-46 fields (G5), bill-to/ship-to split, ₹ + lakh/crore formatting + amount-in-words (G10), rendered through the existing document/PDF pipeline with division-emitted identity (Wave-12.5 pattern). (b) **Bill of Supply** as a true second document type (G6) with its mandatory legend. (c) Credit/debit notes referencing the original invoice number/date. **Byte-identity law: all existing GCC document goldens unchanged.**

**B5 — GSTR-1 portal JSON export.** Generate the period's GSTR-1 JSON per the A4-verified schema from emitted documents: B2B, B2CS, B2CL, CDNR/CDNUR, HSN summary, doc-series, NIL. Deterministic output (golden-testable). Ship a validation pass that reports what the portal would reject (missing GSTINs, HSN gaps) BEFORE export — the accountant sees problems in-app, not as portal errors. This export is derived read-only reporting over documents — it posts nothing, mutates nothing.

**B6 — The e-invoicing applicability indicator.** Config `einvoiceThresholdAATO` (default ₹5,00,00,000) + display-only indicator ("e-invoicing applicable from AATO > ₹5cr — this deployment: not applicable / applicable, integration pending"). Honest, cheap, and the IN-W3 mount point.

## 3. Hard boundaries

- **Zero posting-logic changes.** The GST engine computes and renders; it does not introduce new GL posting behavior. If correct Indian invoicing appears to REQUIRE a posting change, stop-and-report with evidence — owner-gated, zero pre-authorizations.
- **Byte-identical GCC behavior.** Existing overlays, goldens, PDFs, tests: unchanged. The India plane is additive and inert unless mounted.
- **Public-repo synthetic law.** No real Indian company names, GSTINs, PANs, addresses. Synthetic GSTINs must still pass format validation (construct them properly from synthetic PANs).
- **Config-not-constant** for every threshold/rate in §0 (HSN tiers, e-invoice threshold, composition ceilings, rates). A government notification must never require a code change.
- **No external network calls.** The portal-JSON path is offline by design. No GSP, no IRP, no GSTN API this wave.
- Keep-lists binding; Wave-10 audits + QA sweep green at final commit. No merge, no push, no tag.

## 4. Definition of done + report

Done = A1–A6 verdicts (A1 mount-point map + A4 schema-verification record are the report's centerpieces) · B1–B6 shipped with table-driven tests green · ACs proven: (1) India demo overlay produces a Rule-46-complete tax invoice (intra AND inter-state) + a Bill of Supply, screenshots in report; (2) GSTR-1 JSON golden for the demo period, with the A4 schema version cited; (3) GCC byte-identity: goldens + suite green, QA sweep no-diff on non-India screens; (4) HSN refuse-to-generate demo; (5) FY-boundary numbering demo · gates green (suite ALONE).

Write `FABLE_INDIA_SPEC_01_REPORT.md`, commit, paste verbatim — established template + an explicit **"what IN-W2/IN-W3 can now assume"** section (the contract the Tally-importer and connected-flow waves inherit: e.g. "every India-plane document carries line-level HSN + place of supply; GSTR-1 derives from documents; GSTIN identity lives on the division"). Severity honesty is law.

**Known owner-reserved questions (do NOT resolve in-wave; flag if they block):** keyboard-first voucher-entry UX + Day Book home (Design Constitution matter, own wave) · pricing/partner economics · India-plane naming in user-facing copy.
