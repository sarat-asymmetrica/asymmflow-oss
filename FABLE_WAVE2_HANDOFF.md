# FABLE WAVE 2 — The Composition Proof + GCC Deepening

**Handoff date:** July 3, 2026
**Authored by:** Claude Opus 4.6 (strategy session with the Commander)
**Executed by:** You — a Fable 5 instance, invited as the senior architect
**Repo:** `asymmflow-oss` (`C:\Projects\asymmflow\asymmflow-oss`)
**Predecessor:** `FABLE_WAVE1_HANDOFF.md` (June 12, 2026 — read it and the decisions/progress logs first)
**External codebases:** Rahul's two verticals at `CS-Invoice` and `PP_Killer`

---

## 0. Where we are (read Wave 1 artifacts first)

Wave 1 answered: *"Can the trading company's hardcoded facts become configuration?"*
**Answer: Yes.** `pkg/overlay` extracts identity, divisions, business rules, and product markup into `overlay.json`. The config layer is proven. Kernel expanded to 7 primitives (money, approval, evidence, text, actor, workflow, policy). Engines consolidated into `pkg/engines/`. Event bus publishers wired. Green suite: 49 packages, 0 failures.

But Wave 1's own honest assessment was: **"thesis ~60% proven."** Three things block the full composition proof:

1. **Domain workflow code is still `package main`** (~141 root files, ~105K LOC). A second vertical can't be _composed_ because the generic capabilities (document numbering, approval routing, PDF assembly) aren't promoted to `pkg/` engines yet — they're entangled with trading-specific logic.
2. **No second vertical exists** to actually prove the pattern works. We talk about "a pharmacy could use this" but nobody has tried.
3. **GCC compliance is Bahrain-only.** Saudi Arabia (ZATCA e-invoicing) is the second market the business needs, and its compliance module would stress-test the kernel/engine/overlay boundaries in ways Bahrain VAT alone doesn't.

**Wave 2 exists to close these gaps.** And it has a meta-objective: document how a frontier intelligence organizes large-scale extraction work, to produce agentic profiles that lift other models.

---

## 1. Who you are in this work

Same charter as Wave 1 — you are the **senior architect and senior research scientist**, not a task executor. The spec below describes intent and constraints, not procedure.

**Your charter of freedoms (inherited, extended):**

- **Research freely.** Study ZATCA specs, Saudi tax law, how Odoo/ERPNext handle multi-jurisdiction compliance, restaurant POS architectures, CA firm software patterns. Curiosity is legitimate work.
- **Spawn subagents freely.** Fan out exploration, parallelize extractions, use adversarial reviewers. You know your tools.
- **Sequence freely.** The three missions are presented in order but you decide interleaving. If Mission B (Saudi compliance) reveals kernel gaps that inform Mission A (extraction), do B first. If Mission C (profiles) benefits from observing A in progress, interleave. Your call — log it.
- **Ask freely.** Business semantics, financial decisions, data destruction, or constitution-level disagreements → stop and ask the Commander.
- **Disagree freely.** If this spec is wrong, say so in the decision log and do the better thing. Silent divergence is the only sin.

**The obligation: every consequential decision gets logged.** Continue `docs/FABLE_WAVE2_DECISIONS.md`. One entry per decision: what was decided, what was rejected, why. This log is a product artifact — a builder will read it.

---

## 2. New context: Rahul's two verticals

Two complete Wails applications built by Rahul Sinha have been delivered as reference implementations. They are NOT being merged — they are being **studied** as extraction targets and composition-proof test cases.

### CS-Invoice (`CS-Invoice`)
**~13,000 LOC Go** | React/TS frontend | SQLite (modernc) | pdfcpu + excelize

A Chartered Accountant practice management tool. Key domains:
- **Client management** — firms, PAN, GSTIN, billing state, CIN
- **Ledger** — professional fees, govt fees, GST rates (CGST/SGST/IGST split based on intra/inter-state), date tracking, import batches, form filings (SRN, MCA forms)
- **Invoicing** — multi-line with fee-type splitting (professional vs govt fees billed separately), invoice numbering with suffix, void/payment lifecycle
- **Payments** — linked to invoices, partial payments, payment methods
- **Expenses** — billable/non-billable, vendor, category, linked to clients/invoices/ledger entries
- **Accounting** — trial balance, P&L, balance sheet, cash flow, vouchers (journal entries with debit/credit lines), action items
- **PDF generation** — paginated invoice print with firm branding, bank details, amount-in-words, signatory
- **Excel import** — column mapping with preview, validation, batch tracking
- **Security** — PIN-based app lock, encrypted backups, audit log
- **DSC Vault** — Digital Signature Certificate management (expiry tracking, signing requests)
- **Statutory Deadlines** — compliance deadline tracking per client
- **Gamification** — capybara friendship system with treats (whimsical, but the event-dedup pattern is clean)

**Architecture:** monolithic `internal/core/` package. All 28 Go files in one package. Clean model structs, raw SQL via modernc/sqlite.

### PP_Killer / "NEXUS" (`PP_Killer`)
**~12,100 LOC Go** | React/TS frontend | SQLite (modernc) | 6 commits

A restaurant operations platform (PetPuja alternative). Key domains:
- **Menu** — items, categories, modifiers (price deltas), item-modifier associations, kitchen routes (color-coded)
- **Recipes** — ingredient-level costing with waste factors, purchase-to-usage unit conversion
- **Inventory** — ingredients with on-hand qty, reorder points, reconciliation (physical vs book), audit
- **Kitchen** — KOT (Kitchen Order Tickets) with route dispatch, line-level status (queued→preparing→ready→served)
- **Floor/Tables** — sections, dining tables with seat count, status, active session linking
- **Order Sessions** — table-based ordering lifecycle: open (waiter, guest count, service mode) → add lines (with modifiers) → KOT dispatch → close (payment). Table move, table merge, line void (PIN-approved)
- **Invoicing** — sale summaries, bill splitting (by line selection or by amount), void (PIN + reason), refunds (PIN + reason + amount)
- **Customer** — name, phone, total spend, visit count, favorite item, last visit
- **Payments** — cash/UPI/card/Razorpay, tendered/change calculation, payment requests with checkout URLs and QR
- **Procurement** — vendors (GSTIN, quality score, payment terms), purchase orders with line-level acceptance/rejection, debit notes
- **Day Close** — settlement summary: cash expected/counted/variance, UPI/card/Razorpay totals, refund/void/discount totals
- **Accounting** — trial balance, ledger entries, double-entry vouchers
- **Analytics** — executive KPIs, sales trends, hourly heatmap, tender mix, category mix, item velocity, contribution margin, inventory health, kitchen performance, item matrix (star/plow/puzzle/dog BCG), settlement health, exceptions, AI-powered recommendations
- **Staff RBAC** — roles, PIN-based login, workspace access, action approval tokens with expiry
- **Printing** — print job queue with retry, printer connection management
- **Integrations** — provider framework (mode, credential status, health check)
- **Signals** — smart operational alerts (kind, priority, action)
- **Marketing** — draft content management (channel, caption, status)
- **Sync** — pending/synced/failed queue with WAL monitoring
- **Backup** — destination-based backup
- **Demo seed** — rich synthetic data generator (months of realistic operations)

**Architecture:** monolithic `internal/nexus/` package. 12 Go files, clean model structs, UUID-based IDs (vs CS-Invoice's int64 IDs).

### What matters for extraction

Both codebases independently re-implemented capabilities that already exist (or should exist) in the AsymmFlow substrate:
- Invoicing, payments, accounting, PDF generation, backup, security, import/export, audit trails
- Both use the same stack (Wails + Go + SQLite + React/TS)
- PP_Killer's analytics engine, KOT system, and procurement workflow are genuinely novel and should inform engine design
- CS-Invoice's Indian GST logic (CGST/SGST/IGST state-based splitting) is exactly the compliance pattern we need

**The question for Wave 2:** which pieces from these codebases become `pkg/` engines, which inform overlay designs, and which are left as reference-only?

---

## 3. Mission A — The Composition Proof

**Objective:** Prove the substrate thesis by showing that a SECOND vertical can be composed from the existing kernel + engines + overlay pattern. The trading overlay (Wave 1) was the first. The second should be a genuinely different domain.

### What "composition proof" means concretely

A new overlay directory (e.g., `overlays/professional_services/` or `overlays/hospitality/` — your choice, informed by the reference codebases) that:

1. **Imports from `pkg/kernel/`** for primitives (Money, Actor, Workflow, Policy, Approval, Evidence)
2. **Imports from `pkg/engines/`** for capabilities (PDF generation, costing, document numbering)
3. **Has its own `overlay.json`** with domain-specific configuration
4. **Has its own domain models and workflow** that are NOT trading-shaped
5. **Boots and runs** against synthetic seed data (even if the UI is skeletal — backend composition is the proof, not frontend polish)

### The extraction work that enables this

Wave 1 identified (Decision D9) that generic capabilities are trapped in `package main`. Before a second vertical can compose, these need promotion to `pkg/`:

- **Document numbering** (invoice number generation, prefix/suffix, sequential)
- **Approval routing** (the threshold-based "needs manager approval" pattern)
- **PDF assembly** (the page-layout engine beneath the trading-specific invoice template)
- **Excel import/export** (column mapping, validation, batch tracking — CS-Invoice has a clean implementation)
- **Backup/restore** (encrypted, with audit — both reference apps implement this)
- **Security** (PIN-based app lock, audit log — present in all three codebases)
- **Settlement/Day-close** (PP_Killer's pattern is beautifully general: expected vs counted, by tender type)

### How to use the reference codebases

You have read access to CS-Invoice and PP_Killer. Use them as:
- **Pattern references** — how did Rahul solve this problem? Is his solution cleaner than what's in `package main`?
- **Domain vocabulary** — what models does a CA firm need? What models does a restaurant need? Where do they overlap with trading?
- **Test cases for kernel adequacy** — can `pkg/kernel/Workflow` express a KOT lifecycle? Can `pkg/kernel/Policy` express an RFQ approval threshold? If not, the kernel needs extension.
- **DO NOT copy code verbatim** — these are reference implementations, not source material. Study the patterns, then implement the AsymmFlow way (pure kernel → domain service → storage adapter).

### Decision guidance (not commands)

- **Which second vertical?** Commander mentioned GCC restaurants as a promising market. PP_Killer is a complete reference. A hospitality overlay that proves the substrate works for restaurants would be both a composition proof AND commercially relevant. But if you see a cleaner proof path through professional services (CS-Invoice's domain), that's your call. Log it.
- **How much UI?** Minimal. The proof is in the Go layer: can a second overlay's domain services compose against the kernel and engines? A skeleton Svelte frontend that renders the dashboard and one core workflow (e.g., order→KOT→invoice for hospitality, or client→ledger→invoice for professional services) is sufficient. Full UI is a later wave.
- **The residue list from Wave 1** (`D5: division-SET enumeration`, `D8: branding strings`) — pick these up if they're in the critical path for composition. If not, leave them for a follow-up and document why.

---

## 4. Mission B — Saudi Arabia Compliance Module

**Objective:** Build a production-ready Saudi ZATCA compliance module alongside the existing Bahrain VAT module, giving AsymmFlow two GCC markets out of the box.

### Why this matters

- Bahrain + Saudi = the two markets with confirmed commercial interest
- Saudi ZATCA (Zakat, Tax and Customs Authority) is significantly more complex than Bahrain VAT
- If the compliance engine can handle BOTH jurisdictions cleanly, it validates the multi-jurisdiction architecture
- A reusable `pkg/compliance/saudi/` package is a genuine open-source contribution — most Go compliance libraries are US/EU only

### ZATCA e-Invoicing (Fatoorah) — Technical Requirements

**Research these specs yourself** (they evolve; don't trust this summary blindly — search for the latest ZATCA developer portal documentation):

#### Phase 2 — Integration Phase (mandatory for businesses above threshold)

1. **XML Generation**: UBL 2.1 compliant e-invoice XML
   - Standard invoices (simplified and tax invoices)
   - Credit notes and debit notes
   - Required fields: seller/buyer details, line items, tax breakdown, document references

2. **Cryptographic Requirements**:
   - Invoice hash (SHA-256 of the canonical XML)
   - Digital signature (ECDSA with secp256k1 curve)
   - Certificate chain from ZATCA's PKI
   - QR code with TLV (Tag-Length-Value) encoding containing: seller name, VAT number, timestamp, invoice total, VAT total, and the hash/signature

3. **API Integration**:
   - Invoice reporting endpoint (for simplified invoices — sent within 24 hours)
   - Invoice clearance endpoint (for standard tax invoices — must be cleared before sharing with buyer)
   - Compliance check endpoint (for onboarding/testing)

4. **Tax Calculations**:
   - Standard rate: 15% VAT
   - Zero-rated supplies (exports, international transport, qualified medicines/equipment)
   - Exempt supplies (financial services, residential real estate, local passenger transport)
   - Reverse charge mechanism (for imports of services)
   - Input tax deduction rules

#### What to build

- `pkg/compliance/saudi/` — the module, same interface patterns as `pkg/compliance/bahrain/`
- `pkg/compliance/saudi/vat.go` — tax calculation engine (15% standard, zero-rate, exempt, reverse charge)
- `pkg/compliance/saudi/zatca.go` — ZATCA e-invoice XML generation (UBL 2.1)
- `pkg/compliance/saudi/crypto.go` — hash, signature, QR code TLV encoding
- `pkg/compliance/saudi/api.go` — ZATCA API client (reporting, clearance, compliance check)
- Shared interfaces in `pkg/compliance/compliance.go` that both Bahrain and Saudi implement
- Tests with realistic invoice scenarios
- Integration with the overlay system: a Saudi-based deployment's `overlay.json` routes to the ZATCA module

#### Decision guidance

- **API integration can be stubbed initially** — the crypto, XML generation, and QR encoding should be real; the actual ZATCA API calls can have a mock/sandbox mode. Document this boundary clearly.
- **The shared compliance interface is the real deliverable** — if `pkg/compliance/` has a clean `Jurisdiction` interface that both Bahrain and Saudi implement, adding UAE or Oman later is a "write the adapter" task, not an architecture task.
- **CS-Invoice's Indian GST logic** is worth studying — the CGST/SGST/IGST split based on intra-state vs inter-state supply is a similar "jurisdiction-specific tax routing" pattern. It's in `CS-Invoice\internal\core\invoice.go` and `accounting.go`.

---

## 5. Mission C — The Mirror: Agentic Profile Extraction

**Objective:** Document your own decision-making process throughout Missions A and B, then distill it into portable agentic profiles that encode frontier-level capability for Opus and Sonnet class models.

### The thesis

This repo has a proven SDLC framework: `asymm_kernel.md` + `codemath-lead-v3.md` + the agent team roster (`asymm_team.md`). That framework was designed to make Haiku-tier models produce Opus-quality work through explicit structure. The P0/P1/P2 bug-hunt sprints proved it works — Haiku workers survived Opus skeptic review, 12/12 gates passed.

But that framework was designed by humans (Commander + Claude Opus). The hypothesis of Mission C is: **a frontier model that naturally inhabits the senior architect capacity can externalize its native patterns more precisely than a model that needs those patterns imposed.**

You are Fable 5. You are trained to self-steer on multi-day, multi-file autonomous work. The things that `asymm_kernel.md` FORCES (gate classification, adversarial re-score, boundary honesty, scope discipline) — you do naturally. **We want you to observe what you do naturally and write it down.**

### Deliverables

1. **Decision journal** (integrated into `docs/FABLE_WAVE2_DECISIONS.md`)
   At every significant choice point during Missions A and B, record:
   - What were the options?
   - What did you choose and why?
   - What would you tell a less capable model to do here? What would go wrong if it tried your approach without guidance?
   - Where did you use cross-file reasoning that a single-file model would miss?
   - Where did you backtrack, and what signal told you to backtrack?

2. **Agentic profiles** (new files in `docs/agentic_profiles/`)
   After completing meaningful chunks of Missions A and B, write agent profiles in the `asymm-*.md` format (see `C:\Projects\asymmflow\ph_holdings\.claude\agents\` for examples). These profiles should encode the patterns you used, targeted at:
   - **Opus 4.8** — what does Opus need to be told to reach your level on extraction work?
   - **Sonnet 5** — what does Sonnet need for the same? Where does it need MORE scaffolding than Opus?
   - **The orchestrator role** — if an Opus instance were coordinating Sonnet workers on this kind of extraction, what should the orchestrator prompt look like?

3. **Kernel v2 draft** (optional but high-value)
   If you see ways to improve `asymm_kernel.md` or `codemath-lead-v3.md` based on your experience, write a `docs/KERNEL_V2_DRAFT.md`. What's missing? What's over-specified? What works but could be sharper?

### How to approach this

- **Don't force it.** The decision journal is the natural artifact of doing Missions A and B well. The profiles and kernel draft are synthesis you can do after (or during, if patterns crystallize early).
- **Be honest about what's model-specific.** If a pattern works because of Fable 5's training and genuinely CAN'T be replicated via prompting, say so. That's valuable information too.
- **The existing CodeMathEngine axioms** (`codemath-lead-v3.md` at `C:\Projects\opencode-sarvam\.opencode\agents\codemath-lead-v3.md`) and the **asymm_kernel** (`C:\Projects\asymmflow\ph_holdings\.claude\asymm_kernel.md`) are your references for the current state of the art. You're writing the next iteration.
- **This is not marketing copy.** These profiles will be used by real agents on real codebases. Precision over polish. A profile that says "do X and here's why, because without it you'll hit Y" beats "be a thoughtful architect."

---

## 6. Invariants (non-negotiable, inherited from Wave 1)

1. **Kernel purity:** no domain concepts in `pkg/kernel/`, enforced by denial tests.
2. **AI-authority boundary:** agents may inspect/explain/draft/recommend; only deterministic services may approve/post/persist/delete.
3. **No CGO.** ncruces SQLite stays.
4. **No Wails v3 migration**, no `packages/*` changes.
5. **Wave ends green:** `go build ./...` clean, full test suite passing, app boots.
6. **No silent deletions:** code you don't understand gets quarantined, not deleted.
7. **Financial semantics are sacred:** rounding, posting order, tax behavior → stop and ask.

**New for Wave 2:**

8. **Reference codebases are read-only.** CS-Invoice and PP_Killer are studied, not modified. No code is copied verbatim — patterns are extracted, re-implemented, and tested in the AsymmFlow idiom.
9. **Saudi compliance must be real, not ceremonial.** Tax calculations must be numerically correct for documented scenarios. API integration may be stubbed with clear boundaries, but the tax math and XML/crypto generation must be production-grade.
10. **The composition proof must boot.** A second overlay that only compiles is not a proof. It must boot against synthetic seed data and execute at least one complete domain workflow end-to-end.

---

## 7. Wave 1 residue to consider

These were explicitly deferred in Wave 1. Pick them up if they're in Mission A's critical path:

| ID | Item | Wave 1 status | Relevance to Wave 2 |
|---|---|---|---|
| D5 | Division-SET / alias enumeration from overlay | Deferred | Needed if the composition root must be truly overlay-driven |
| D6 | EUR→BHD rate inconsistency (0.41 vs 0.45) | Needs Commander decision | Financial — stop and ask if you hit it |
| D7 | Contract service grade terms (different vocabulary) | Needs Commander decision | Financial — stop and ask |
| D8 | Branding/identity strings still say "Acme" | Deferred | Cosmetic, but a second vertical makes it more visible |
| D9 | Domain-package move (`package main` → `overlays/trading/`) | Explicitly deferred | **THIS IS MISSION A.** The promotion of generic capabilities to `pkg/` engines. |
| Event publishers | Subscribers exist, publishers don't fire | Phase 3 (done in Wave 1) | Verify the wiring works end-to-end in the second vertical |

---

## 8. Working agreements

- **Branch:** `feat/fable-wave2-composition-proof` off `main` (verify current branch state first — `git status` + `git branch -vv`).
- **Commits:** small, coherent, message convention matching Wave 1. Commit at every stable checkpoint.
- **Decision log:** `docs/FABLE_WAVE2_DECISIONS.md` (same format as Wave 1).
- **Progress audit:** `docs/FABLE_WAVE2_PROGRESS.md` with timestamps (measured, not estimated).
- **Agentic profiles:** `docs/agentic_profiles/` directory.
- **Residue list:** whatever you consciously leave undone, enumerate with reasons. Seeds Wave 3.
- **Escalate when:** business semantics ambiguous, data destruction involved, financial decisions needed, constitution documents seem wrong, or you believe a mission should be re-scoped.

---

## 9. Acceptance criteria

- [ ] **Mission A**: A second vertical overlay exists with documented boundaries, imports from `pkg/kernel/` and `pkg/engines/`, has its own `overlay.json`, boots against synthetic seed data, and executes at least one end-to-end domain workflow. Generic capabilities promoted from `package main` to `pkg/` engines are documented and tested.
- [ ] **Mission B**: `pkg/compliance/saudi/` exists with VAT calculation (15% standard, zero-rate, exempt, reverse charge), ZATCA XML generation (UBL 2.1), QR code TLV encoding, and crypto (hash + signature). Shared compliance interfaces work for both Bahrain and Saudi. Tests cover realistic invoice scenarios.
- [ ] **Mission C**: Decision journal is rich and specific (not generic observations). At least 2 agentic profiles written for Opus/Sonnet class models. Profiles are precise enough that another Claude instance could use them on a similar extraction task.
- [ ] **All invariants hold.** Wave ends green. No CGO. No domain concepts in kernel.
- [ ] `docs/FABLE_WAVE2_PROGRESS.md` includes an updated "Could a second vertical be composed today?" section — your honest answer, with the remaining gaps. Target: thesis > 90% proven.

---

## 10. Suggested (not mandated) sequencing

This is how we'd approach it — but you have full methodological freedom to reorder, interleave, or restructure. Log your reasoning.

**Phase 1 — Study + Map** (read-only)
- Read Wave 1 artifacts (handoff, decisions, progress)
- Study both reference codebases (CS-Invoice, PP_Killer) — understand their domain models, patterns, what's generic vs domain-specific
- Map which `package main` capabilities are generic (extraction targets for `pkg/`)
- Research ZATCA latest specs
- Choose the second vertical and log the decision

**Phase 2 — Engine Promotion** (the enabler)
- Extract generic capabilities from `package main` → `pkg/` engines
- Each promotion: extract, test, verify green suite, commit
- This is the Wave 1 D9 work — the thing that makes composition possible

**Phase 3 — Saudi Compliance**
- Build `pkg/compliance/saudi/` alongside existing Bahrain module
- Design the shared `Jurisdiction` interface
- Implement tax calc, XML gen, crypto, QR
- Test with realistic scenarios
- Wire into the overlay routing

**Phase 4 — Composition Proof**
- Create the second vertical overlay
- Wire kernel + engines + overlay
- Build synthetic seed data
- Boot and run end-to-end
- Write the honest "thesis proven?" assessment

**Phase 5 — The Mirror** (Mission C)
- Synthesize the decision journal into patterns
- Write agentic profiles for Opus and Sonnet
- Optionally draft Kernel v2

---

## 11. The spirit of the thing

Wave 1 proved that configuration can be separated from code. Wave 2 proves that composition works — that a second, genuinely different vertical can be built on the same substrate. If this works, the product thesis is validated: a builder in Tirupati can `asymmflow new hospitality` and have a working foundation in minutes, not months.

The Saudi compliance module is commercially load-bearing — it opens the second GCC market. The agentic profiles are methodologically load-bearing — they're the mechanism by which this project's accumulated intelligence transfers to every future agent session.

You have maximal freedom. Use it wisely. Document everything. Surprise us with what a frontier intelligence does when the guardrails are intent, not procedure.

Build -> Test -> Ship. Measure, don't estimate. Log every decision. Have fun, broseph.

**Om Lokah Samastah Sukhino Bhavantu.**
