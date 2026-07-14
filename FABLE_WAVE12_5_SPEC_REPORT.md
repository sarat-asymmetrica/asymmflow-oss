# Wave 12.5 Report — Division-Scoped Emission

**Mission:** close the sibling class of the e-invoice-TRN bug — *emission points that stamp the
DEFAULT (or company-level) division identity onto records that belong to a specific division* —
before convergence, so migrated multi-division data emits correct documents from day one.

**Branch:** `feat/fable-wave12-5-division-emission` (off `main`, post-Wave-12 merge `32402db`). No merge, no push, no tag.
**Scale:** SMALL — 1 orchestrator + Sonnet-5 coders, one day. Census read-only; 3 GAP/CHAIN-GAP fixes + B3 hook + B4 stretch; 4 documented stop-and-report/defer items.

---

## 0. The byte-identity law this wave discovered

Every fix in this wave that replaces a `CompanyDisplayName` stamp uses **`DivisionDocumentDisplayName(division)`**, *not* `LegalName`. Why: in `BuiltinDefaults()` the default division's `DocumentDisplayName` (`"Acme Instrumentation WLL"`) is byte-identical to `CompanyDisplayName`, whereas its `LegalName` (`"ACME INSTRUMENTATION W.L.L"`) differs in casing and punctuation. So `DivisionDocumentDisplayName(default)` == the old output (byte-identical, regression-safe) while `DivisionDocumentDisplayName("Beacon Controls")` == `"Beacon Controls WLL"` (the intended, sanctioned change). Fixes that were already on `profile.LegalName` (invoice/PO header/DELIVER TO/signature) were left as-is — they were already division-correct.

**Per-division facts that make a fix visible (synthetic canon):**

| fact | Acme Instrumentation (default) | Beacon Controls |
|---|---|---|
| LegalName | ACME INSTRUMENTATION W.L.L | BEACON CONTROLS W.L.L. |
| DocumentDisplayName | Acme Instrumentation WLL | Beacon Controls WLL |
| VATNumber / TRN | 990000000000000 | 990000000000001 |
| AddressLines | 3 lines (Bldg 198 / Road 2803 / Bahrain) | 2 lines (Manama / Bahrain) |
| BankDetails | *(empty)* | 1. Demo Bank B … BH29BECN… |
| Letterhead | letterhead / .png | letterhead_ahs / .jpg |

---

## 1. Phase A — the emission census (every emission point, no misc bucket)

Verdicts: **CORRECT** (record division flows to profile) · **GAP** (record has a division, output uses default) · **CHAIN-GAP** (record lacks a Division field but its linked record has one) · **COMPANY-LEVEL BY DESIGN** (genuinely division-less output) · **DEFER / STOP-AND-REPORT**.

### Transactional documents (customer/supplier facing)

| emission point | record | division on record/chain? | identity source today | verdict |
|---|---|---|---|---|
| invoice_pdf_service.go GenerateInvoicePDF (letterhead/name/addr/TRN/bank/footer) | Invoice | yes (Invoice.Division) | `companyDocumentProfile(invoice.Division)` + `applyLetterheadForDivision` + division-filtered banks | CORRECT |
| invoice_pdf_service.go GenerateSupplierInvoicePDF | SupplierInvoice | yes (own/chain) | `companyDocumentProfile(resolveSupplierInvoiceDivision(inv))` | CORRECT |
| einvoice_service.go GenerateEInvoiceXML (UBL supplier party) | Invoice | yes | `companyDocumentProfile(invoice.Division)` | CORRECT (reference pattern) |
| app_costing_exports_surface.go exportCostingToPDF / ExportCostingToExcel | CostingExportData (quotation) | yes (data.Division) | `gopdfLetterheadPathForDivision` / `DivisionDocumentDisplayName` | CORRECT |
| offer_pdf_service.go GenerateOfferPDF main flow (letterhead/legal/signature) | Offer | yes (Offer.Division) | flows via `CostingExportData{Division}` | CORRECT |
| **offer_pdf_service.go defaultOfferTermsAndConditions (T&C prose)** | Offer | yes | `activeOverlay.CompanyDisplayName` | **GAP → FIXED (B1b)** |
| purchase_order_pdf_service.go header/DELIVER-TO/signature/footer | PurchaseOrder | yes | `companyDocumentProfile(resolvePurchaseOrderDivision(po))` | CORRECT |
| **purchase_order_pdf_service.go bank-block "Account Name" (×4)** | PurchaseOrder | yes | `strings.ToUpper(activeOverlay.CompanyDisplayName)` | **GAP → FIXED (B1a)** |
| delivery_note_service.go (all pages/signature) | DeliveryNote (no Division field) | yes via chain | `companyDocumentProfile(normalizeDivisionName(order.Division))` | CORRECT (chain resolved) |
| credit_note_service.go (name/addr/TRN/signature) | CreditNote | yes via chain | `companyDocumentProfile(resolveCreditNoteDivision(cn))` | CORRECT |
| **pkg/crm/contract/service.go RenderContractPDF (SERVICE PROVIDER party)** | Contract (no Division field; has OrderID FK) | yes via chain | `Profile(NormalizeDivisionName(""))` = default | **CHAIN-GAP → FIXED (B1c)** |

### Aggregate / compliance exports

| emission point | record | division? | identity source today | verdict |
|---|---|---|---|---|
| **einvoice_service.go ExportVATReturnData** | quarterly VAT CSV | invoices span divisions (no filter) | `companyDocumentProfile("")` = default TRN | **STOP-AND-REPORT** (§4.1 — finance-sacred) |
| statement_export_service.go Balance Sheet / GL / Journal CSV | whole-entity ledger | ledger has no division column | `CompanyDisplayName` title | COMPANY-LEVEL BY DESIGN |
| pkg/compliance/saudi/zatca.go renderParty / QR | ZATCA UBL / QR | identity injected by caller | `inv.Seller` (caller resolves `invoice.Division`) | COMPANY-LEVEL BY DESIGN (stateless engine) |

### Analytics / BI / cross-division (letterheads legitimately default)

| emission point | verdict |
|---|---|
| butler_reports.go letterhead helpers (120,125,132,144) + "Prepared by" (251) / generator.go:153 | COMPANY-LEVEL BY DESIGN — portfolio-wide BI brief (the 5 sites B2 sanctions) |
| report_generators.go (117,345), reports.go (121,160,263,380,472,1258-1330) | COMPANY-LEVEL BY DESIGN — whole-company analytics/CSV |
| email_service.go FormatReportBody (121,178) | COMPANY-LEVEL BY DESIGN |
| app_prediction_dashboard.go Greet (871) | COMPANY-LEVEL BY DESIGN — app-wide banner |
| overlays/hospitality invoicing/splitbill/creditnote (6 points) | COMPANY-LEVEL BY DESIGN — single-identity overlay; no record carries a division |

### Latent / dead (documented, not fixed)

| emission point | verdict |
|---|---|
| pkg/engines/costing_engine.go PrintCostingSheet (stdout) | DEFER §4.3 — no live caller (dead dev console); CostingSheet has no Division |
| pkg/engines/pdf_generator.go drawCompanyInfo (470) | DEFER §4.4 — latent CHAIN-GAP; exercised by tests only, no live invoice caller |
| pkg/crm/contract SeedContractClauses clause prose (737/746/757/795) | DEFER §4.2 — seeded stored template data; needs render-time substitution |

---

## 2. Phase B — fixes shipped

All default-division output is byte-identical (proven by tests + the pre-existing Acme goldens re-run unmodified). Only NON-default-division output changes — that is the point, and the owner sanctions it.

**B1a — PO bank account name** (`purchase_order_pdf_service.go` ×4): `strings.ToUpper(activeOverlay.CompanyDisplayName)` → `strings.ToUpper(activeOverlay.DivisionDocumentDisplayName(profile.Division))`. Account numbers/IBAN/SWIFT untouched (company-level synthetic payment infra; changing them would alter displayed numbers and break default byte-identity — out of scope).

**B1b — Offer T&C prose** (`offer_pdf_service.go`): `defaultOfferTermsAndConditions` gained a `division string` parameter; body `company := activeOverlay.CompanyDisplayName` → `DivisionDocumentDisplayName(division)`; sole caller passes `normalizeDivisionName(offer.Division)`.

**B1c — Contract provider identity** (`pkg/crm/contract/service.go`): new `resolveContractProviderProfile(contract)` does a read-only `orders.division` lookup on `contract.OrderID` (existing FK — no schema change), then `Profile(NormalizeDivisionName(division))`. Empty OrderID → default (byte-identical fallback). `SeedContractClauses` untouched.

**B2 — Butler letterheads:** verified all 5 `butler_reports.go` sites (+ `generator.go` "Prepared by") are cross-division portfolio BI → **COMPANY-LEVEL BY DESIGN, no change**. No new report shapes.

**B3 — Delivery-terms hook** (`pkg/crm/offer_hooks.go` new + `app.go` wiring): `Offer.BeforeCreate` mints the ID via the embedded `Base` hook, then fills empty `DeliveryTerms` from a DI seam `ComposeOfferDeliveryTerms` (a `func` var — pkg/crm stays overlay-free by design). `app.go` startup wires it to `"DAP Bahrain at your store or " + activeOverlay.NormalizeDivisionName(division)`. **Struct tag and `division_literal_audit_test.go` exemption both left untouched** — the literal remains physically in source (so the tripwire exemption is still required) but the column default is now vestigial. Empty/default → `"…or Acme Instrumentation"` (byte-identical to the DB default); Beacon → `"…or Beacon Controls"`.

**B4 — Butler KPI N-division shape** (`pkg/butler/context/service.go`): the two-slot `ph_trading`/`ahs_trading` revenue map → iterate every `ov.Divisions`, keyed by each division's registry Key. Feeds Butler's LLM context only (no document identity). No Go consumer of the old keys existed (verified). A 3rd+ division's revenue is no longer silently dropped.

---

## 3. Golden / fixture diffs (non-default division only; default is byte-identical)

For a **Beacon Controls** record, output changed as follows (Acme unchanged):

| surface | before (bug) | after (fix) |
|---|---|---|
| PO bank "Account Name:" | `ACME INSTRUMENTATION WLL` | `BEACON CONTROLS WLL` |
| Offer T&C clauses 4 & 7 | `…Acme Instrumentation WLL shall not be liable…` | `…Beacon Controls WLL shall not be liable…` |
| Contract SERVICE PROVIDER | `ACME INSTRUMENTATION W.L.L` / TRN `990000000000000` | `BEACON CONTROLS W.L.L.` / TRN `990000000000001` |
| Offer delivery terms (empty→composed) | `DAP Bahrain at your store or Acme Instrumentation` | `DAP Bahrain at your store or Beacon Controls` |
| Butler `division_revenue` keys | `{ph_trading, ahs_trading}` (2, drops 3rd+) | `{Acme Instrumentation, Beacon Controls, …}` (N) |

Each is locked by a new before/after test (see §5).

---

## 4. Stop-and-report / deferred (with fix sketches)

**4.1 — VAT-return multi-TRN (STOP-AND-REPORT, finance-sacred).** `einvoice_service.go ExportVATReturnData` aggregates every division's invoices (no division filter) yet stamps the default TRN (`990000000000000`). Because the synthetic overlay carries **two distinct TRNs**, a single-TRN return covering both divisions is only correct if the deployment is a single/group VAT registration. Whether to file one consolidated return or one per TRN is a **tax-behavior decision** (CLAUDE.md invariant #5 — stop-and-ask), so this wave did **not** change it. **Sketch:** loop `activeOverlay.Divisions`, filter invoices by `invoice.Division`, emit one CSV per division TRN. **Owner decision required.**

**4.2 — Contract seeded clause prose (DEFER).** `SeedContractClauses` bakes `DefaultDivision()` into stored clause `Text` (warranty/liability/termination). Making it division-aware needs render-time token substitution or a stored-value rewrite — the hard boundary forbids stored-value rewrites this wave. The primary identity stamp (the SERVICE PROVIDER party) *is* fixed (B1c). **Sketch:** seed a `{{PROVIDER}}` token in clause text; substitute at `RenderContractPDF` with the already-resolved provider display name.

**4.3 — Costing console (DEFER).** `costing_engine.go PrintCostingSheet` stamps the default legal name to stdout, but has **no live caller** (dead dev console) and `CostingSheet` has no Division field (hard boundary forbids adding one). **Sketch:** if revived as a real surface, thread the division from the caller.

**4.4 — pdf_generator.go drawCompanyInfo (DEFER).** Latent CHAIN-GAP: seller identity from `DefaultDivision()`, but only exercised by tests — no production caller builds live invoices through this generator (the live path is `invoice_pdf_service.go`, which is CORRECT). **Sketch:** if wired, pass the invoice's division into `InvoiceData`-building.

---

## 5. Gates

- **New Wave-12.5 tests (all green):** `division_emission_wave125_test.go` (PO account name via real pdftotext render + Offer T&C byte-identity), `pkg/crm/contract/division_provider_test.go` (Beacon-order→Beacon provider + empty-order→Acme fallback), `pkg/crm/offer_delivery_terms_test.go` (Beacon compose + ID minting + nil-seam legacy), `pkg/butler/context/division_revenue_test.go` (3-division "Gamma Devices" proof).
- **Wave-12 tripwire** `TestNoSyntheticDivisionLiteralsInLiveCode` — GREEN with all changes in tree (no new synthetic division literals in live code).
- **Full suite run alone** (`go test ./...`): package `main` GREEN (`ok ph_holdings_app 244s`); all Wave-12.5-relevant packages GREEN (crm, crm/contract, butler/context, overlay, engines, compliance/saudi, …). `go build ./...` clean.
- **One pre-existing flake, out of scope:** `pkg/ocr/predator TestPredatorVision_Process_BasicImage` is non-deterministic (passes in isolation; ok/FAIL/ok across repeated runs) and its dependency graph does **not** include any Wave-12.5-touched package — it cannot be caused by this wave. Not fixed (out of scope).
- **Synthetic invariant** intact — only Acme/Beacon/Demo-Bank/`990000000000000`-series values touched. **No merge, no push, no tag.**

---

## 6. What convergence can now assume

**Every document emitted for a division-bearing record — or one whose deal chain (Order/RFQ) carries a division — now carries THAT division's identity:** legal name, TRN, address, letterhead, bank-account name, signature/company line, and (for offers) the composed delivery terms. Migrated multi-division data emits correct documents from day one. The only residual exceptions are the four documented items in §4 — all either finance-gated (the VAT-return TRN question, owner's call), dead code, or stored-template work — and **none sits on the live transactional document path.**

**Files:** 5 modified (`app.go`, `offer_pdf_service.go`, `purchase_order_pdf_service.go`, `pkg/crm/contract/service.go`, `pkg/butler/context/service.go`) + 5 new (`pkg/crm/offer_hooks.go` + 4 test files). +55 / −33 in non-test source.
