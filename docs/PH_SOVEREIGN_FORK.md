# The Sovereign Fork — deploying AsymmFlow as YOUR company

Mission D of the PH convergence campaign, made literal: **the company is a
JSON file, not a program.** This repository ships with a fully synthetic
demo company ("Acme Instrumentation"). A real deployment does not fork the
code — it authors one `overlay.json` and (optionally) imports its data.
No company-specific fact may live in Go source; if you find one, that is a
bug (see the flag register at the bottom).

**The law this document exists to enforce:** real company facts — legal
names, TRN/VAT registration numbers, addresses, bank accounts and IBANs,
principal/brand alias catalogues, historic division spellings — live ONLY
in the sovereign deployment's `overlay.json` and database. They never enter
this public repository, not in code, not in docs, not in test fixtures.

## 1. Where overlay.json goes

The app searches, in order: next to the executable, `./data/`, the working
directory, then the platform app-data directory (`%APPDATA%\AsymmFlow` on
Windows, `~/.local/share/AsymmFlow` elsewhere). First valid file wins; no
file means the built-in synthetic defaults.

The file must be **complete** — it is parsed directly, not merged over the
defaults. Start from the annotated example at `data/overlay.json` and edit
every section. Two fields are exceptions with safe fallbacks: a missing
`supplier_aliases` keeps the built-in vocabulary, and a blank
`license_key_prefix` keeps `"PH"` (so existing license activations never
break on a partial file).

## 2. What each section carries for a real deployment

| Section | Real-world content (sovereign side only) |
|---|---|
| `company_display_name`, `industry`, `country`, `jurisdiction` | Trading name; `jurisdiction` routes invoices to the matching `pkg/compliance` engine (BH/SA/IN) |
| `currency`, `currency_decimals`, `default_vat_rate` | e.g. BHD/3/10.0 — the VAT default seeds the settings layer; a user setting overrides it at runtime |
| `exchange_rates_to_base` | The single FX source of truth for import-time AND live conversion |
| `divisions[]` | One entry per trading division: registered legal name, VAT/TRN number, address lines, bank-detail strings printed on documents, letterhead asset key + artwork filename |
| `divisions[].aliases` | **Every historic spelling of the division found in your data** (lowercase). The Go normaliser and the SQL backfills both read these; a spelling you omit falls back to the default division |
| `supplier_aliases` | Principal/brand vocabulary for supplier resolution: `canonical_codes` (variant code → canonical supplier code) and `brand_aliases` (brand token → supplier search terms) |
| `business_rules` | Margin floors, competitor-specific minimums, grade payment terms — your commercial policy |
| `product_markup_rules` | Standard margins by product type |
| `license_key_prefix` | Keep `"PH"` if keys were already issued; changing it invalidates the format check for existing keys |
| `seed_sets` | For a real deployment with imported data: list only what you want (e.g. `["rbac-roles"]`). The demo bundles (`demo-products`, `demo-customers`, `demo-bank`, example license keys) exist for evaluation builds — your real rows arrive via import, not seed |

## 3. Bringing your data (the PH path)

Order matters:

1. Run the OSS binary once against a fresh database file **with your
   overlay.json in place and demo seed sets disabled** — this provisions
   the schema without demo rows. Rehearsal-proven setting:
   `"seed_sets": ["default-assets"]` — do NOT enable seed bundles whose
   rows the import will carry (`rbac-roles`, `license-keys`,
   `demo-*`); your company's roles, users, and reference data arrive
   with the data. (The importer replaces the app's own foundation-seeded
   skeleton — accounts, categories, bank fixtures, assets — with your
   rows and reports each replacement, but seeds it cannot reconcile are
   your responsibility to leave off.)
2. Close the app and run `cmd/phimport` from a machine-local copy of the
   source database into a copy of the fresh one. Read its JSON report end
   to end — skipped and UNMAPPED tables are the load-bearing half.
3. Re-enter machine-bound secrets (encrypted settings are never carried)
   and let the first startup recompute document hashes under the new
   install's salt.
4. Reconcile before cutover with `cmd/phreconcile` (Mission H): every
   carry count and every money sum — invoices, payments, credit notes,
   supplier ledger, POs, the banking/FX/VAT suite — must match to the
   fils, both after the import AND after the first boot. The full
   procedure is `docs/PH_CUTOVER_RUNBOOK.md`.

(The Mission E "banking suite not provisioned" gap was closed by Mission G
— the suite now provisions create-if-missing on fresh files — and Mission H
carries + reconciles it; see `PH_CONVERGENCE_PROGRESS.md` Wave 5.)

## 4. Flag register — company facts still in code (deliberate deferrals)

These are known, synthetic-valued, and queued rather than hidden:

- `pkg/crm/domain.go` — the DeliveryTerms GORM column default embeds the
  demo division name in DDL. Moving a schema-level default into config
  touches offer-document text paths; deferred to its own golden-first pass.
- `bank_accounts_service.go` — the demo bank-account seed rows (synthetic
  IBANs) are code. They are gated by the `demo-bank` seed set, which a
  real deployment disables; the rows themselves stay demo fixtures.
- `import_2026_data.go` / `onedrive_import_service.go` — legacy importer
  alias maps (customer-code and folder-name vocabularies). One-time
  import paths; a sovereign fork replaces its import vocabulary with the
  overlay-era `phimport` flow instead.
- Seed catalogues (`seedDefaultSuppliers`, `seedProductDatabaseInternal`)
  are demo fixtures by design: the config seam is the `seed_sets` gate,
  not the rows (PC-D8).
