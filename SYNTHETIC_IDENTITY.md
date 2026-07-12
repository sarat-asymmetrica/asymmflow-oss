# Synthetic Demo Data Canon — AsymmFlow

All sample, seed, test, and demo data in this repository describes a **fictional**
instrumentation-trading business. This file is the single source of truth for that
fictional world so the data stays internally consistent across Go source, tests,
frontend, and docs. **Never introduce real company names, people, tax IDs, bank
details, or financial figures.**

> Bahrain / BHD / VAT 10% are kept as realistic **domain context** (not sensitive).
> The `INV-YYYY-NNNN` document-number format and the `PH-` license-key prefix are
> demo conventions.

## Hard rules

1. **Value-only edits.** Change string-literal *contents*, comments, and data values
   only. Do **not** rename Go symbols, functions, types, struct fields, or change
   signatures — that preserves the call graph and keeps the build green.
2. **Keep it compiling & green.** Same types, same field counts. If you change one
   side of a test assertion, change the expected side to match.
3. **Use the names below.** For any entity not listed, invent a clearly-fictional
   name and reuse it consistently.

## The operating company (two divisions)

| Division | Legal name | VAT TRN |
|---|---|---|
| Acme Instrumentation | ACME INSTRUMENTATION W.L.L. | `990000000000000` |
| Beacon Controls | BEACON CONTROLS W.L.L. | `990000000000001` |

The `division` column stores `"Acme Instrumentation"` or `"Beacon Controls"`.

## Banking (demo)

| Holder | Bank | Account | IBAN (`BH\d{2}[A-Z]{4}\d{14}`) | SWIFT |
|---|---|---|---|---|
| Acme | Demo Bank A | `10000000001` | `BH29DMOA10000000000001` | `DMOABHBM` |
| Acme | Demo Bank B | `10000000002` | `BH29DMOB10000000000002` | `DMOBBHBM` |
| Acme | Demo Bank C | `10000000003` | `BH29DMOC10000000000003` | `DMOCBHBM` |
| Acme | Demo Bank D | `10000000004` | `BH29DMOD10000000000004` | `DMODBHBM` |
| Beacon | Demo Bank C | `20000000001` | `BH29DMOC20000000000001` | `DMOCBHBM` |

## Demo customers

National Petroleum Co. (`NPC`) · Gulf Smelting Co. (`GSC`) · North Grid Authority
(`NGA`) · Delta Petrochemicals (`DPC`) · Vertex Energy (`VTX`) · Coastal Gas Co.
(`CGC`) · Horizon Petroleum (`HZP`) · National Oil Authority (`NOA`) · Summit Light
Metals (`SLM`) · AquaPure Technologies · BlueWave Marine · Meadow Dairy · Riverside
Power O&M · Eastside Wastewater · Crescent Trading · Pinnacle O&M (`PNM`) · Intercon
Group · Metalworks Services (`MWS`) · Coastal JV W.L.L. (`CJV`) · Northstar Trading ·
Cascade Water (`CSW`) · Polaris Cooling (`PLC`) · Logica Systems (`LGS`) · Aeromech
Services (`AMS`) · Sandhill Industrial (`SHI`) · Xenon Tech (`XNT`) · Alder Works
(`ALW`) · Zenith Trading (`ZNT`) · Summit Systems (`SMT`).

## Demo suppliers / manufacturers

Rhine Instruments · Oxan Analytics · Helvetia Metering · Helix Automation · Northwind
Controls · Apex Process · Meridian Systems · Volta Electric · Lumera Metering ·
Stonewell Systems.

## Demo people

Use first-name forms in fixtures: Jordan · Alex · Sam · Casey · Taylor · Jamie ·
Riley · Devin · Quinn. Emails → `first@<synthetic-domain>.example`. Phones →
obviously-fake (`+973-1700-0000`). Dev home paths → `/Users/developer/...` or
`t.TempDir()`. Addresses → `PO Box 0000, Manama, Bahrain`.

## Demo financials

Fictional P&L (clearly demo, never real audited figures): Revenue 2.40M BHD (2024) /
3.10M BHD (2023); Net profit 180K / 250K; gross margin 22%; current ratio 2.10x.
Never name a real auditor.

## Domain context that is intentionally kept (not sensitive)

- **Public bank names in statement parsers/recognition** (e.g. `NBB`, `BBK`,
  `HSBC`, `Al Salam`, format detectors like `parseNBBFormat`). These recognize
  public institutions and document formats — like knowing what a PDF header looks
  like. Only the operating company's *own* account numbers, IBANs, and SWIFT codes
  are synthesized (see the Banking table). Parser test fixtures use fake accounts.
- `BHD`, VAT 10%, Bahrain — realistic domain context.
- The `PH-` license/document-number prefix and the `ph_holdings` internal module/db
  identifier are kept conventions (a later rebrand wave, not client data).

## Never commit

- API keys, passwords, tokens, or usable default "master" keys.
- Real client data of any kind, or pointers that resolve to it.
