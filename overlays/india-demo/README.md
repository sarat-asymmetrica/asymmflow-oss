# India demo overlays

Two fixture overlays for the India GST jurisdiction plane (India Spec-01,
`FABLE_INDIA_SPEC_01_THE_INDIAN_INVOICE.md`). Both describe **fictional**
companies — public-repo law, same as every other fixture in this repo (see
`SYNTHETIC_IDENTITY.md`).

- **`overlay.json`** — Meridian Instruments & Controls Pvt Ltd, a synthetic
  instrumentation trader with two divisions in different GST states: Meridian
  Mumbai (Maharashtra, state code 27) and Meridian Bengaluru (Karnataka, state
  code 29). Loading both divisions together demonstrates intra-state
  (CGST+SGST) and inter-state (IGST) tax splits from one company/PAN. A demo
  HSN/SAC rate schedule (`india.tax_categories`) seeds the rates used by the
  GST tax engine tests and the demo tax invoice.

- **`composition/overlay.json`** — Kaveri Trade Links, a synthetic composition
  taxable person (`india.composition: true` on its one division). This is the
  Bill-of-Supply demo company: composition dealers issue a Bill of Supply
  (no tax lines, no ITC, mandatory legend) instead of a GST tax invoice — a
  genuine second document type, not a zero-rated invoice (§0 G6).

## Synthetic identity

Company names, PANs, GSTINs, and addresses are invented and checksum-valid but
**not real registrations**. See `SYNTHETIC_IDENTITY.md` → "India demo canon"
for the full reserved list (companies, PANs, GSTINs, demo customer names).

## How tests load these fixtures

`pkg/compliance/india/plane_test.go` loads each directory with
`overlay.LoadOverlay([]string{dir})` and asserts `ValidateOverlayIndia` returns
no problems for the shipped files, plus named errors for deliberately
corrupted in-memory variants (bad check digit, mismatched state). `overlay.
LoadOverlay` requires at least one division to accept a file — both fixtures
qualify.
