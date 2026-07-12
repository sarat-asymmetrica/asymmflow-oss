# Fable Wave 3 — Progress Audit

Honest self-audit of the Wave 3 handoff (`FABLE_WAVE3_HANDOFF.md` — "Make the
Proof the Law"), written by the model that did the work. Timestamps are
measured from the git log of `feat/fable-wave3-composition-law` (all
2026-07-03, local; the sprint began ~16:15 with branch-base clarification and
required reading).

## Measured timeline

| Time | Commit | What |
| --- | --- | --- |
| 16:41 | 0163954 | **A.1** composition seam (`pkg/runtime/composition`); found + fixed the dead-DSN-pragmas bug (pilot ran DELETE journal, not WAL) |
| 16:53 | 82db577 | **A.2** registered model-set + sqlite_master schema golden (found GORM's nondeterministic CHECK-constraint order) |
| 17:03 | b9af0cd, 4aeac29 | **A.3** seeds behind the overlay (gate-in-place, nil = run-everything) · **A.4** one engine list, one registration path |
| 17:18 | 2ff35a3 | **B.1** `pkg/approvals` on kernel approval+actor; agents refused in both flows, three depths of tests |
| 17:27 | f64f99f | **B.2** audit convergence; restored the silently-dropped resourceID/description; dead AuditEvent deleted loudly |
| 17:49 | fb4dda8 | **C.2** ZATCA-profile CSR (hand-assembled PKCS#10, secp256k1), verified with OpenSSL as a foreign parser |
| 18:01 | 19d7e25 | **C.1** both ❓ flags closed against the SDK's own materials (Commander downloaded the zip); two parser bugs fixed |
| 18:13 | bfe9d2b | **C.3** hospitality credit notes: one ICV/PIH chain across document types, negative tenders, net day close; InstructionNote placement bug fixed |
| 18:26 | c5bca38 | **B.3** Excel header engine (two lookup semantics kept apart) · **B.4** PDF seller identity onto the overlay |

Sequencing followed §7's risk-retirement rule: A.1 first (if the seam can't
serve both verticals, know in hour one — it could), then strictly cheapest
falsification.

## Mission status

### Mission A — decompose the trading composition root: A.1–A.4 DONE

The trading app and the hospitality app boot through the same composition
seam, and the `startup()` diff is deletions and delegations only (A.2 alone:
−148/+11 in app.go). Concretely:

- `pkg/runtime/composition`: overlay load (standard search cascade), SQLite
  DSN construction, per-model migration, bus + compliance registry + hook.
  Both `app.go startup()` and `cmd/hospitality` delegate to it.
- The ~90-model AutoMigrate list is a registered model-set
  (`tradingModels()`), schema pinned by a deterministic `sqlite_master`
  golden (constraints sorted — GORM emits them in map order).
- Seeds are overlay-selected (`seed_sets`), gate-in-place so boot ordering is
  untouched; absent config = today's behavior exactly (pinned nil-vs-empty).
- Compliance engines are declared once (`tradingTaxEngines`) and registered
  through the seam only.
- **Found by measuring, fixed with authorization:** the trading DSN's
  mattn-style params were silently ignored by ncruces — the pilot had been
  running `journal_mode=DELETE` with a 60s default busy timeout. Now WAL +
  NORMAL + 5s, pinned by a live-pragma test.

**A.5 (stretch) deliberately folded into B.1** — the delete-approval
decision half now lives behind `pkg/approvals`; peeling GRN numbering just
to tick a box would be motion, not extraction (W3-D11).

### Mission B — A.1 engine residue: B.1–B.2 DONE, B.3/B.4 stretch DONE (minimum honest)

- **B.1** `pkg/approvals`: Assessment (routing) + Transition (deciding) on
  kernel vocabulary. Costing risk and delete approval rewired
  behavior-identically (warning ORDER pinned; the historically-dead
  rule-specific recommendations recorded, not carried). Agents provably
  refused at kernel construction, engine transition, and both call sites.
- **B.2** one audit path (`pkg/infra/audit`): logAudit's dropped
  resourceID/description restored end-to-end; the discarded AuditEvent
  vocabulary deleted loudly; AuditLogger now persists (masked) through the
  same recorder.
- **B.3** three header-matching implementations → `pkg/documents/excel`,
  preserving BOTH tie-break semantics (column- vs variant-priority) with a
  test that documents the disagreement.
- **B.4** minimum honest version delivered: PDF seller identity from
  `overlay.Active()` (the hardcoded lines were drifted copies); the
  three-generator-path unification remains residue.
- Offer NN-YY numbering: untouched, per standing stop-and-ask.

### Mission C — ZATCA hardening + hospitality graduation: DONE

- **C.1** Both Wave 2 ❓ flags CONFIRMED against primary source — the
  Commander downloaded the SDK zip linked on zatca.gov.sa; its jar's own
  signing template matches our SignedInfo exactly (pinned by a test that
  extracts the strings from the vendored template). Tag 9 = certificate
  signature bytes (current standard + two current implementations; moot for
  standard invoices here — we never mint their QRs). **Version caveat
  recorded:** the publicly-linked SDK zip is v2.03 (2022) and its QR
  validator still implements the DRAFT R/S tag semantics. Free audit: the
  SDK's headerless-base64 key/cert exposed that our PEM parsers couldn't
  read the gateway's actual binarySecurityToken form — fixed, with the
  ZATCA-test-CA-issued cert as the regression fixture (which also pins the
  SAN-surname quirk and the genesis PIH).
- **C.2** `GenerateCSR`: ZATCA-profile PKCS#10, hand-assembled ASN.1,
  environment-selected template names, OpenSSL-verified (foreign parser).
  The EGS serial travels as surname (2.5.4.4) — OpenSSL's "SN" alias,
  confirmed in ZATCA's own issued cert.
- **C.3** Hospitality credit notes: manager-PIN + kernel-authority gated,
  one ICV/PIH chain across invoices AND credit notes (chain continuity
  pinned), refunds as negative tender rows netting into the settlement day
  close (same-day sale+refund closes at zero), full-refund-once semantics.
  Found + fixed in the ZATCA module: the credit-note reason belongs in
  `cac:PaymentMeans/cbc:InstructionNote` (BR-KSA-17), not `cbc:Note`. The
  demo binary now runs money both ways and exits 0.

### Mission D — the mirror: DONE

- `docs/FABLE_WAVE3_DECISIONS.md` — W3-D1…D11, [Mirror] discipline.
- This progress audit.
- `KERNEL_V2_DRAFT.md`: **not written, deliberately.** Wave 3 is the second
  data point and it CONFIRMS the kernel: `pkg/kernel/approval` +
  `pkg/kernel/actor` composed into `pkg/approvals` and gated two live flows
  with zero kernel changes; money/actor/settlement absorbed the credit-note
  flow unchanged. A draft that says "no changes needed" is a decision-log
  line, not a document.

## Thesis proven: ~92%

Wave 2 said ~85%, with the biggest gap being that the ORIGINAL vertical
ignored the substrate. That gap is now closed at the composition level: one
seam boots both verticals; models, seeds and compliance are configuration;
the kernel's authority boundary gates every approval transition in the
trading app, not just the proof vertical. The proof vertical also graduated
past happy-path-only (refunds, chain integrity, net settlement).

What honestly keeps it from higher:

1. **The App god-object still exists.** The composition ROOT is decomposed;
   the ~1229-method App and its ~140 root files are not. That is Wave 4+
   material by design, but "the vertical is configuration plus a thin domain
   package" is only fully true of hospitality; trading is configuration plus
   a very thick one.
2. **ZATCA remains gateway-unexercised.** Both ❓ flags are closed and the
   CSR half of onboarding exists, but no invoice of ours has round-tripped
   the sandbox compliance check (needs portal OTPs, human-in-the-loop), and
   the current-generation SDK validator (R3.4.x, JDK 11–14) has not been run
   against our XML.
3. Seed selection is config-driven but the seed CONTENTS are still trading
   code; a third vertical would want seed bundles as data.

## Residue for Wave 4

- The App fan-out: peel domain services off the 1229-method App behind
  `pkg/` seams (delete-approval's storage/notification half is a mapped
  starter; GRN numbering next).
- PDF generation: unify the three generator paths (B.4 did identity only).
- Offer NN-YY numbering (standing stop-and-ask: OneDrive folder coupling).
- ZATCA: sandbox compliance-check round-trip with portal OTPs; run the
  R3.4.x SDK validator (needs JDK 11–14); wire CSR generation into an
  onboarding flow (api.go client + GenerateCSR are both ready).
- Hospitality: partial refunds / line-level credit notes (current flow is
  full-refund-once); a `CreditNoteIssued` domain event (deliberately not
  smuggled under `InvoiceCreated`); bill split; print queue.
- Seed bundles as data for a third vertical.
- `security_enhancements.go` carries more residue (SessionManager,
  RateLimiter) whose usage should get the same B.2 convergence treatment.
