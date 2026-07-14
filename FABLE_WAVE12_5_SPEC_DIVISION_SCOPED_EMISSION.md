# Wave 12.5 Spec — Division-Scoped Emission (small residue wave)

**Mission:** Wave 12 made division *vocabulary* single-sourced. This small wave closes the sibling class of bug the Wave-12 report surfaced: **emission points that stamp the DEFAULT division's (or the company-level) identity onto records that belong to a specific division.** A Beacon-division purchase order printing "ACME INSTRUMENTATION WLL" in its header is the e-invoice-TRN bug wearing a different hat — same class: *record has a division; output ignores it.* Close the class before convergence, so migrated multi-division data emits correct documents from day one.

**Scale: SMALL.** One orchestrator + 2–3 coders, one day. Sequencing: after Wave 12 (division registry). Before convergence data missions.
**Repo:** `asymmflow-oss` (PUBLIC). **Branch:** `feat/fable-wave12-5-division-emission` off `main` (post-Wave-12 merge). No merge, no push, no tag.
**Authority docs:** `CLAUDE.md` → Wave 12's report §5 (the inherited contract) → keep-lists → this spec.

**Fact-check inherited from the owner's review (trust these, don't re-derive):**
- The Wave-12 report's residual #2 (e-invoice TRN, `einvoice_service.go:49`) was a **stale flag** — the per-division fix already landed in Mission D (`ce2ff1a`): `companyDocumentProfile(invoice.Division)` normalizes and resolves per-division TRN/legal-name/address. It is the PATTERN to replicate, not a gap to fix.
- Known candidate gaps found in pre-flight: PO PDF header emits `ToUpper(CompanyDisplayName)` (company-level) — Wave 12 kept this byte-identical deliberately; `butler_reports.go` letterheads always use `DefaultDivision()` (5 sites); costing console + quotation template emit default-division identity.
- Model reality is MIXED: `Invoice`, `Order`, `CustomerReceipt`, `Expense`, `Payroll`… carry `Division`; **`Offer` and `DeliveryNote` do NOT** — their division context lives on the linked Order/RFQ chain.
- Wave-12 residuals #1 and #3 fold into this wave (see B3/B4).

## 1. Phase A — the emission census (read-only; the report's centerpiece)

For EVERY document/export/compliance emission point (PDF services, XML/CSV exports, print surfaces, letterhead application, signature blocks, VAT returns), one census row:

| emission point | record type | does the record (or its deal chain) carry a division? | identity source used today | verdict |

Verdicts (exactly one each):
- **CORRECT** — record division flows to the profile (the `companyDocumentProfile(invoice.Division)` pattern).
- **GAP** — record/chain has a division; output uses default/company identity. → B1.
- **CHAIN-GAP** — the record itself lacks `Division` (Offer, DeliveryNote) but its linked Order/RFQ has one; output uses default. → B1 via chain lookup (READ the linked record — do NOT add columns).
- **COMPANY-LEVEL BY DESIGN** — genuinely division-less output (whole-company statements, butler cross-division analyses). Recorded with one sentence of justification; default/company identity is correct here.

Also in Phase A: which letterhead/bank-details/signature facts differ per division in `BuiltinDefaults()` (that's what makes a GAP *visible* in synthetic output — e.g. Beacon has BankDetails, Acme doesn't).

## 2. Phase B — the fixes

**B1 — Close every GAP and CHAIN-GAP.** The one pattern: resolve the profile from the record's (or chain's) division — `companyDocumentProfile(NormalizeDivisionName(...))` — for legal name, TRN, address, letterhead, bank details, and signature-company lines. Chain lookups are read-only joins on existing FKs.
**⚠️ This is a DELIBERATE behavior change** (unlike Wave 12): documents for NON-default-division records will change identity — that is the point, and the owner sanctions it here. For default-division records output stays byte-identical (regression-checked). Every changed output gets a before/after fixture test for a Beacon-division record (offer→PO→DN→invoice chain where fixtures exist); goldens updated ONLY for non-default-division fixtures, with the diff quoted in the report.

**B2 — Butler letterheads.** The 5 `butler_reports.go` default pulls: division-scoped reports (if any exist per A's census) resolve from their subject division; genuinely cross-division reports stay default and get the COMPANY-LEVEL verdict. No new report shapes.

**B3 — The GORM struct-tag residual** (`pkg/crm/domain.go:232`, delivery-terms column default): add a `BeforeCreate` hook that fills empty delivery terms from the registry-composed value. **Do NOT touch the struct tag** — the schema golden must stay byte-identical; the column default becomes vestigial. Drop the audit exemption if the literal becomes unreachable prose-only.

**B4 — Butler KPI N-division shape (OPTIONAL — attempt only if A+B1 land early).** The two-slot primary/secondary revenue breakdown → iterate registry divisions. Display-shape change only; if it grows beyond a trivial diff, DEFER with a fix sketch — this wave's must-haves are B1–B3.

## 3. Hard boundaries

- **Zero schema changes, zero migrations, zero stored-value rewrites.** CHAIN-GAPs are resolved by reading linked records, never by adding Division columns (that's a convergence-era decision).
- **Behavior change is scoped to non-default-division OUTPUT identity only.** Amounts, totals, VAT math, numbering, routing, permissions: untouched — any fix that would alter a number is stop-and-report.
- Default-division output byte-identical (goldens prove it).
- Wave-12 tripwire + Wave-10 audits + gate baseline (suite run ALONE) green at final commit. Synthetic invariant. No merge/push/tag.

## 4. Definition of done + report

Done = the full census table (every emission point, no "misc" bucket); all GAP/CHAIN-GAP rows fixed with per-division fixture proof; B3 hooked; B4 shipped or sketched; gates green. Write `FABLE_WAVE12_5_SPEC_REPORT.md`, commit, paste verbatim — census table + the quoted golden diffs + an updated "what convergence can now assume" §: *every document emitted for a division-bearing record carries that division's identity.*
