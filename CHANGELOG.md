# Changelog

All notable changes to AsymmFlow are documented here. Format loosely follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); this project has not
yet cut a tagged release, so every entry below is reconstructed from the wave
history on `main` (`FABLE_*_REPORT.md` docs and merge commits) rather than
from release notes written at the time. See §"Tagging convention" below for
what happens going forward.

## [Unreleased]

### Added
- `.github/workflows/gate.yml` — public CI floor: Go build/vet/test on
  `windows-latest`, frontend-lab build, mesh smoke gate. Secretless by
  doctrine. (Custodian Wave 1, CW1-C.)
- `docs/custodian/CI_LAW.md` — what the CI floor enforces vs. the full local
  wave-gate; runner-OS decision with measured timings; exclusion list.
- This `CHANGELOG.md`.

---

## Wave history (reconstructed headlines)

Entries below summarize the major merged waves on `main`, newest first. Each
maps to a `FABLE_*_REPORT.md` or `FABLE_*_HANDOFF.md` in the repo root for
full detail; commit hashes are the merge commits from `git log --merges`.

### India W1 — The Indian Invoice (`3c41f17`)
Full India/GST compliance plane: PAN/GSTIN identity + state registry, GST
computation engine (intra CGST/SGST split, inter IGST, SEZ, composition,
RCM, refuse-to-generate on HSN/rate violations), Rule-46 tax invoice + Bill
of Supply PDFs, India credit notes, per-GSTIN FY-scoped numbering, GSTR-1
JSON export with pre-upload validation, e-invoice applicability indicator.
Additive-only schema; GCC byte-identity preserved. See
`FABLE_INDIA_SPEC_01_REPORT.md`.

### Wave 13 — Perception & Print (`fb27926`)
Butler AI and OCR consolidated onto Mistral as the sole provider (AIMLAPI/
Fly.io retired, keys revoked); embedded fonts for PDF rendering; payslip PDF
generation. See `FABLE_WAVE13_REPORT.md`.

### The Sealed Corridor (`0fd87b0`)
Corridor field-kit campaign: sealed kits reaching an encrypted room both
ways, two-instrument-defect fix at gate, field ceremony proof.

### The Sealed Ship — Bare runtime, Phases 0-3 (`d31271d`)
Bare-runtime campaign for the mesh field kit: clean-machine rounds, in-kit
self-certifying verifier, sealed corridor groundwork.

### Mission A2.1 — Reception Grade (`0e46377`)
Field-failure fix (lazy Holesail init + geography-hermetic gates) plus the
Guided Path (`START_HERE.cmd`) — receptionists never touch a command line.

### Mission A2 — The Corridor (`08fde66`)
India↔Bahrain WAN field kit: probe tooling, bundled-runtime kit 2.0,
always-on anchor peer, Bare-runtime findings.

### Sovereign Mesh + Messenger track (`4588e9d`)
Go→WASM deterministic room fold (Missions M1-M4 + human layer), messenger
design constitution + decisions, sidecar protocol v0 bridge, Correspondence
kernel screen, kitchen-table field kit — field-confirmed end-to-end over
real hardware (bidirectional messaging + SHA-256-verified file transfer).

### DP2 — The Installer (`f39d352`)
Per-user NSIS installer, three-plane deployment layout, seal icon, identity-
plane overlay-loading fix, BOM-tolerant overlay parse, uninstall self-delete
+ resurrection proven.

### DP1 — The Planes and the Contract (`f2f4bd9`)
Three-plane deployment layout (`pkg/infra/deploy`), unified DB path
resolver, seed/migrate/stamp/downgrade update contract; legacy AppData
layout structurally retired (~860 LOC removed).

### Gap-Close G1-G5 (`f8e580a`)
INTEG gap count 23 → 0. Butler draft-binding split, settlement receipt-
capture path, standalone invoice-create retired, real win-rate aggregation,
payroll compensation hot-zone, artifact-proven exports.

### Frontend-kernel campaign K1-K6 (`b723847`)
49 screens rebuilt on the descriptor kernel in `frontend-lab/`; parity
sign-off in `FABLE_WAVE_K6_PARITY.md`.

### Wave 12 / 12.5 — Division Registry & Emission (`32402db`, `565ddeb`)
One division vocabulary end-to-end (frontend store + backend); every
division-bearing record (PO, offer, contract, VAT return) emits its own
identity; per-TRN VAT returns.

### Wave 10-11 — Sensory & Brand, Polish & True Mirror (`891a6ad`, `e4af964`)
Motion vocabulary, deal-spine timeline, brand slot + rituals, phi-token
systemic fix, standing Playwright QA sweep.

---

## Tagging convention

- **Scheme:** semantic versioning, `vMAJOR.MINOR.PATCH` (matches
  `wails.json`'s `info.productVersion`, currently `2.3.0`).
- **When:** a tag is cut at a merge commit on `main` only — never on a
  feature/wave branch.
- **Who:** the owner (release ceremony, ledger item C8 — not yet built).
- **This wave:** no tags were created or pushed. Custodian Wave 1 seeds the
  convention and this changelog; the first tagged release is a future,
  explicitly owner-gated action.
