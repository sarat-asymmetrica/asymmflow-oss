# Fable Wave 2 — Progress Audit

Honest self-audit of the Wave 2 handoff (`FABLE_WAVE2_HANDOFF.md`), written by
the model that did the work. Timestamps are measured from the git log of
`feat/fable-wave2-composition-proof` (all 2026-07-03, local).

## Measured timeline

| Time | Commit | What |
| --- | --- | --- |
| 12:00 | f269a52, 23e5cd6, 30a61cc | Banked Wave 1 residue (uncommitted D5–D8 work + date-bomb fixes) as the branch base; main untouched |
| 12:05 | dcd1dd8 | Phase 1 study complete: decisions W2-D3..D6 + ZATCA research notes committed |
| 12:11 | ce2a263 | **B1**: Saudi VAT engine + overlay-driven jurisdiction routing (3rd TaxEngine impl — seam validated, not redesigned) |
| 12:25–12:34 | 1705b8e, 9ec2745, bf63f50, e71e0db | **A.1**: numbering, settlement, backup/restore, PIN-lock engines (4 commits, each green) |
| 15:38 | b8e4cc8 | **B2**: full ZATCA Phase 2 (UBL 2.1, secp256k1/XAdES signing, QR TLV, Fatoora client) — the deep-crypto block, incl. a session pause |
| (this commit) | — | **A.2**: hospitality vertical boots end-to-end + Mission C docs |

Sequencing followed the logged plan (W2-D4): B1 → A.1 → B2 → A.2 → C —
cheapest-falsification first.

## Mission status

### Mission A.1 — engine promotion: CORE DONE, residue logged

Promoted/built, each with tests exceeding the original's coverage:

- `pkg/documents/numbering` — replaced 4 near-identical root generators;
  call sites (INV/CN/PO/DN) rewired byte-identically. **Found and fixed a
  real concurrency bug in the pattern being extracted** (SQLite read→write
  lock-upgrade deadlock; engine is now UPDATE-first).
- `pkg/finance/settlement` — pure day-close engine on kernel money+actor.
- `pkg/infra/db` backup — promoted VACUUM-INTO backup + added verified
  restore (integrity-check gate + pre-restore snapshot).
- `pkg/infra/auth` PIN lock — greenfield PBKDF2 + lockout engine.

**Deferred (residue, deliberate):** approval-routing retrofit onto
`pkg/kernel/approval`; audit-log convergence (two parallel systems); Excel
findColumn de-triplication; PDF engine de-contamination (3 generator paths);
Offer NN-YY numbering (coupled to OneDrive folder matching — stop-and-ask).

### Mission B — Saudi ZATCA: DONE to the sanctioned boundary

- `pkg/compliance/saudi`: VAT engine (15%/zero/exempt/out-of-scope/reverse
  charge, halala rounding, VAT-number validation) registered as the third
  TaxEngine; overlay `jurisdiction` field routes invoices (explicit →
  country → currency inference), default pinned by test.
- ZATCA Phase 2: UBL 2.1 XML (7-digit subtypes, dual TaxTotal, per-category
  buckets), ICV/PIH hash chain, ECDSA secp256k1 XAdES-BES signing (pure Go,
  manual ASN.1 where stdlib refuses the curve), QR TLV tags 1–9, Fatoora
  API client with correct 200/202/400/401/413 semantics.
- **Boundary (documented in code):** tax math, XML and crypto are
  production-grade and test-verified; the HTTP client is faithful to the
  gateway contract but has NOT been exercised against ZATCA's live gateway.
  Two ❓ items to confirm against ZATCA's official samples before production
  onboarding: exact ds:Reference XPath transform strings; QR tag-9 semantics
  (see `docs/research/ZATCA_PHASE2_RESEARCH.md`).

### Mission A.2 — composition proof: DONE, AND IT BOOTS

`overlays/hospitality` (Saudi café POS domain package + overlay.json) +
`cmd/hospitality` composition root. `go run ./cmd/hospitality` executes a
full business day against synthetic seed data and exits 0; the e2e test
suite pins it. Zero engine code duplicated; zero trading imports; the AI
agent actor is refused at every authority-bearing call site. Two seam bugs
found within a minute of first boot (certificate PrintableString vs UTF-8;
event Amount base-vs-total) — both fixed, both logged (W2-D11).

### Mission C — the mirror: DONE

- `docs/FABLE_WAVE2_DECISIONS.md` — W2-D1..D11, each with a `[Mirror]`
  annotation (what a lesser model would need).
- `docs/agentic_profiles/` — README + Opus 4.8 (deep worker), Sonnet 5
  (scoped worker), orchestrator (wave coordinator).
- This progress audit.
- KERNEL_V2_DRAFT.md: **not written** — deliberate. One composition proof is
  a single data point; kernel changes deserve a second vertical's evidence
  (residue for Wave 3).

## Thesis proven: ~85%

The Wave 1 audit said ~60%: engines existed but nothing proved a second
vertical could compose them. Wave 2 closed the biggest gap — the proof
boots, end to end, with real cross-engine composition (shared numbering
table, settlement-gated day close, event-routed compliance, kernel authority
at every seam).

What keeps it from >90%, honestly:

1. **The trading vertical itself still doesn't use the substrate the way
   hospitality does.** Its composition root (`app.go` startup, ~90 hardcoded
   models, unconditional seeds) predates the overlay architecture; the proof
   shows a NEW vertical composes cleanly, not that the ORIGINAL one has been
   decomposed. That is the core Wave 3 job.
2. Approval routing and audit remain unpromoted (A.1 residue) — two of the
   seven capabilities named in the handoff.
3. ZATCA is unexercised against the live gateway (sanctioned boundary, but
   still an unproven seam with two ❓ flags).

## Residue for Wave 3

- Decompose the trading app's composition root onto the overlay/substrate
  pattern `cmd/hospitality` demonstrates (biggest thesis gap).
- Promote approval routing onto `pkg/kernel/approval`; converge the two
  audit-log systems.
- Excel findColumn de-triplication; PDF generator de-contamination.
- Offer NN-YY numbering (stop-and-ask: OneDrive folder coupling).
- ZATCA: validate hash/signature against ZATCA's official sample set;
  resolve ❓ XPath-transform and ❓ tag-9; exercise sandbox onboarding with
  real portal OTPs (human-in-the-loop).
- Hospitality vertical follow-ups if it graduates from proof to product:
  bill split/refund, credit notes (module supports them; vertical doesn't
  expose them), print queue, empty-session administrative close.
- KERNEL_V2_DRAFT.md once a third vertical's evidence exists.
- Sprint-4/PH operational items tracked outside this repo remain with the
  Commander.
