# India Spec-01 — The Indian Invoice — WAVE REPORT

**Branch:** `feat/fable-india-w1-indian-invoice` · **Spec:** `FABLE_INDIA_SPEC_01_THE_INDIAN_INVOICE.md` (7269c94, on main)
**Operating model:** Opus 4.8 orchestrator/gate + Sonnet 5 coders (b3/b4/b5/b6) + research/recon agents (a3/a4). Per-mission commits, each gated by the orchestrator with own-eyes diff review and own-run test verification.
**Context event:** the original orchestrator terminal died mid-wave (Bun segfault, 2026-07-22 ~13:40 IST). A fresh orchestrator audited the tree: ZERO work lost — B1–B2 (+ partial B3 relocation) survived on disk, were gated as-recovered, and checkpointed (`b9d0620`) before the wave resumed. One mojibake corruption found in the recovered diff and reverted (`5bb1f34`).

---

## 1. Phase A verdicts

**A1/A2 (plane/seam map, identity fields)** — resolved pre-crash, verified at gate: India mounts as `overlay.IndiaCompanyConfig` (company/PAN level) + `overlay.IndiaDivisionProfile` (division/GSTIN level), nil-inert for GCC; `IndiaConfig()` is the single 0⇒statutory-default resolution point; validation lives on the compliance side (`pkg/compliance/india/plane.go` imports overlay, never the reverse — SupplierAliasConfig precedent). PAN-level AATO: seam only this wave (`AATOSource` interface + `ResolveAATO` override; no invoice-history summer — see §5 residue).

**A3 (document pipeline)** — full mount-point map delivered (recon agent, archived). Headlines that shaped the build: **no jurisdiction branch existed in the PDF layer at all** (built new, additively); `Invoice.Status` carries a DB CHECK constraint → Bill of Supply became a `DocKind` discriminator column, never a Status value; numbering `Sequence` keys on (Prefix, Year) only → per-GSTIN series encode the GSTIN in `Spec.Prefix` while the rendered number stays short; the live renderer is gofpdf (`invoice_pdf_service.go`) — the unwired gopdf engine was not touched.

**A4 (GSTR-1 schema verification — the "verify the probe" centerpiece)** — two-layer verification:
- Research agent swept live sources; GSTN's own PDFs/SPA resisted fetching, so field-level shapes rest on a GSP conformance mirror + heavy corroboration — every fact carries a confidence flag (record archived in full).
- The orchestrator then **downloaded the official Returns Offline Tool ZIP from tutorial.gst.gov.in (32,001,280 bytes)** and read its release notes + section CSVs first-hand. CONFIRMED FIRST-PARTY: tool Release **3.2.4** (matches our shipped `GSTR1SchemaVersion` default exactly); **B2CL threshold ₹1,00,000** post-Aug-2024 and *period-aware* (₹2.5L for earlier periods) per Releases 3.2.1/3.2.4; **HSN Table-12 split into B2B/B2C** (Release 3.2.2 + separate sample CSVs); a **40% rate slab exists** (Release 3.2.3) — rate whitelisting anywhere would now be a bug; upload caps **5MB / 19,000 items** (Readme).
- Honest residue: the literal JSON payload key names were NOT read from a first-party artifact (the tool's installer is not archive-extractable). They remain SECONDARY-high-confidence via the GSP mirror. **Before any real portal upload: install the offline tool, open its sample JSON, diff against our goldens.**

**A5 (numbering/FY)** — survived pre-crash: `FiscalYearFor`, `{fy}` template token, `FYStartMonth` reset cadence, `ValidateGSTSeriesNumber` (Rule-46 16-char + charset). Calendar-year GCC specs regression-tested unchanged.

**A6 (demo overlay)** — synthetic canon extended in `SYNTHETIC_IDENTITY.md`: Meridian Instruments & Controls (PAN AABCM0472E; Mumbai 27 + Bengaluru 29 = intra AND inter-state demos on one PAN) and Kaveri Trade Links (composition). All GSTINs are checksum-valid constructions; the check-digit algorithm was **externally anchored** — the orchestrator hand-computed GSTN's published documentation sample (27AAPFU0939F1Z → V) through our implementation's algorithm.

## 2. Phase B — what shipped (per-mission commits)

| Mission | Commit | Substance |
|---|---|---|
| B1+B2 (recovered) | b9d0620 + 5bb1f34 | India plane (GSTIN/PAN/states/AATO/plane-validation, 01–38+97 state registry as embedded data, code 25 documented-absent), overlay fields nil-inert, FY-scoped numbering + Rule-46 validator, demo overlays |
| B3 | 7120485 | `ComputeInvoiceGST`: config-only rates (`RateForHSN` longest-prefix, legacy hsnRates never consulted), intra exact-half split (heads sum exactly on odd paise), inter IGST, SEZ always inter+zero-rated (distinct from configured-0% nil-rating), composition = no tax lines, cess per line, RCM notation-only, **whole-invoice refuse-to-generate** on HSN digit-tier violation (N/N 78/2020-CT via PAN-level AATO) or unconfigured rate. §170 nearest-rupee helper exists, NOT default (R-A4-2) |
| B4 | 6d2a78c | `invoice_pdf_india.go`: Rule-46 tax invoice (all 16 fields), **Bill of Supply enforced for composition divisions regardless of caller intent**, India CN with original-invoice reference, GSTIN/PAN/state seller identity, buyer/ship-to GSTIN split, lakh/crore grouping + `amountInWordsIndian`, per-GSTIN per-FY numbering series with Rule-46 validation. Additive schema: `Invoice{BuyerGSTIN, ShipToGSTIN, PlaceOfSupplyStateCode, DocKind, ReverseCharge}`, items `{HSNCode, UQC}`; `trading_schema.golden` regenerated deliberately (R-A3-7) |
| B5 | 79072c7 | `ExportGSTR1JSON` + `ValidateGSTR1Period` (dry run): per-GSTIN JSON on the `ExportVATReturnData` read-only pattern; b2b/b2cl/b2cs/cdnr/cdnur/hsn_b2b+hsn_b2c/doc_issue/nil; taxes recomputed via B3 engine; pre-upload validation (GSTIN/HSN/POS defects, size caps, GSTN Jul-2025 B2C-only dummy-row advisory — surfaced, never auto-injected); composition divisions statutorily skipped (they file CMP-08/GSTR-4) |
| B6 | b3d655b | `GetEInvoiceApplicability`: display-only G8 indicator, configured threshold (proven non-hardcoded in tests), honest "turnover not computed in-app" state; IN-W3 mount point; frontend card deferred (`BusinessSettings.svelte` named) |

## 3. Gate findings (what the orchestrator caught in coder output)

1. **B4 filename collision (real bug):** `filepath.Base` ran before `/`-substitution — `INV/25-26/001` and `INV/26-27/001` (two FYs legally sharing a calendar year) both exported as `Invoice_001.pdf`, silently overwriting. Fixed (India-only file, zero GCC surface).
2. **B4 hybrid-document hardening:** tax-summary block and line table decided "is this a BoS?" from different sources; a mis-stamped row could print a BoS table with tax-invoice totals. One explicit `isBoS` now flows to both.
3. **B5 `val` vs items disagreement:** invoice `val` came from stored `GrandTotalBHD` — which is still GCC Bahrain-10% creation math for India divisions (see §5.1) — while `itm_det` came from the GST engine. `val` now derives from the same engine computation.
4. **B5 ctin canonicalization** (Spec-07 law): grouping keys now upper/trimmed.
5. **B5 b2cl shape ruling:** restructured flat list → pos-grouped (mirrors the b2b/cdnr grouping family and the live GSTN schema; SECONDARY, re-verify per A4 residue).
6. **B6 canon violation:** test fixture GSTIN was checksum-invalid (orchestrator hand-verified the check digit); replaced with canon Meridian identity.
7. **Recovered-diff mojibake:** two `''`→`”` comment corruptions in `overlay.go` reverted.

Coder judgment calls reviewed and approved (all were honestly flagged, none hidden): composition zeroes RatePct (BoS has no rate column; GSTN's own SEZ-WOP samples carry Rate 0) · HSN digit check is a floor, not exact (voluntary extra digits pass) · `Invoice.ReverseCharge` added beyond R-A3-4's field list (G5 needs it; inert default) · composition demo overlay gained tax_categories (engine's validation pass legitimately needs rate resolution; config-only change) · B5 HSN warnings elevated to errors via `errors.As` (a warning tier the engine makes unreachable would be dishonest) · HSN summary aggregates invoices only, CNs not netted (sign conventions ambiguous — flagged, not guessed).

## 4. Acceptance criteria

1. **Rule-46 invoice (intra + inter) + Bill of Supply from the demo overlay** — proven by `india_documents_test.go` (12 tests): real PDFs generated through the full pipeline and content-verified via pdftotext — CGST/SGST split for Mumbai→Maharashtra, IGST for Mumbai→Telangana, BoS enforced for Kaveri with the mandatory legend. *On-screen screenshots deferred to the owner smoke test (no frontend changed this wave; the artifacts are the PDFs themselves).*
2. **GSTR-1 JSON golden** — pinned golden in `gstr1_export_service_test.go` with deterministic marshal-twice byte-equality; schema version **GST3.2.4** cited in-payload (first-party confirmed current).
3. **GCC byte-identity** — additive-only schema (columns + one deliberate schema-golden regen); overlay.json is read-only at runtime (no re-serialization surface); the A3 guardrail battery (invoice-PDF parity, VAT-return division partitioning, payroll/FX goldens, costing-attachment, signature-blocks, AHS branding) passed on the orchestrator's own run; **full suite run ALONE at wave close** (result appended below). Frontend untouched → QA sweep trivially no-diff.
4. **HSN refuse-to-generate** — engine tests (3-digit B2B under-5cr refuses; above-5cr needs 6 on B2C; export needs 8; unknown HSN refuses) + PDF-layer surfacing test (error returned, no file written).
5. **FY-boundary numbering** — `TestFiscalYearNumberingAprilBoundary`: INV/25-26/002 on Mar 31 → INV/26-27/001 on Apr 1, separate Sequence rows, GCC calendar rollover regression-pinned.

## 5. What IN-W2 / IN-W3 can now assume (the contract)

- Every India-plane document carries line-level HSN + UQC and header-level place-of-supply state code, buyer/ship-to GSTIN, reverse-charge flag. GSTIN identity lives on the division (`overlay.DivisionProfile.India`); PAN + thresholds + rate schedule live on `overlay.IndiaCompanyConfig` (all config-not-constant).
- `india.ComputeInvoiceGST` is the ONE tax computation seam — PDF and GSTR-1 both already consume it; anything new must too.
- GSTR-1 derives read-only from documents; numbering is per-GSTIN per-FY via `Spec.Prefix` keys (`ININV:`/`INBOS:`/`INCN:` + GSTIN).
- `GetEInvoiceApplicability` is the IN-W3 e-invoicing mount; `BusinessSettings.svelte` is where its card belongs.

### 5.1 Known gaps / residue (severity-honest)

- **[HIGH, IN-W2 headline] Invoice-creation math is still GCC-hardcoded:** `createInvoiceWithOptionsEx` computes `VATBHD`/`GrandTotalBHD` at Bahrain 10% for every division. India documents render and export CORRECTLY (both derive from the GST engine), but stored invoice totals for India divisions are wrong until creation is wired through `ComputeInvoiceGST`. Untouched this wave under the zero-posting-changes law.
- **[MED] GSTIN capture:** `CustomerMaster` has no GSTIN column; `BuyerGSTIN`/`ShipToGSTIN`/`PlaceOfSupplyStateCode` are not auto-populated at creation (no frontend surface yet). Fields exist, render, export.
- **[MED] GSTR-1 JSON keys are SECONDARY-confidence** (see A4). Diff goldens against the installed offline tool's sample JSON before first real upload. `nil` (Table 8) field names explicitly UNVERIFIED.
- **[LOW] AATO roll-up:** override-only this wave; the PAN-level invoice-history summer awaits a document-store mission.
- **[LOW] doc_issue `cancel` is structurally 0** (cancelled invoices filtered upstream, mirroring the VAT export); a real cancel count needs an unfiltered numbering query.
- **[LOW] No debit-note document type** (CDNR handles C/D; we emit C only). SEZ/export flags exist in the engine but have no Invoice-level capture yet.
- **[LOW] `numbering.Sequence.Prefix` gorm size:10 tag** is cosmetically wrong for 21-char India keys (SQLite ignores it; schema-hygiene pass someday).
- **[LOW] India renderer ignores GCC field-visibility toggles** (not a Rule-46 concern; noted).

## 6. Owner-reserved questions (unresolved by design)

Per spec §4: keyboard-first voucher-entry UX + Day Book home (Design Constitution matter, own wave) · pricing/partner economics · India-plane naming in user-facing copy. Nothing in this wave blocked on them.

## 7. Gate baseline (measured)

- `go build ./...` clean · `go vet ./...` clean (orchestrator runs).
- Package batteries: `pkg/compliance/india` (engine 15 + gstin/states/aato/plane), `pkg/overlay`, `pkg/documents/numbering` — all green, orchestrator-verified runs.
- Guardrail battery (A3 §7 list) green on orchestrator's own 66.9s run.
- New-surface batteries: 12 (B4 documents) + 10 (B5 GSTR-1) + 4 (B6 indicator) green.
- **Full suite `go test -count=1 -timeout 1800s ./...` run ALONE at wave close (measured):** **86 packages ok, 0 FAIL** (main package 285.3s; `pkg/compliance/india` 1.1s; `pkg/overlay` 1.9s; `pkg/documents/numbering` 0.5s; `overlays/hospitality` 5.7s — full log archived in the session scratchpad).

No merge, no push, no tag. Awaiting owner final gate.
