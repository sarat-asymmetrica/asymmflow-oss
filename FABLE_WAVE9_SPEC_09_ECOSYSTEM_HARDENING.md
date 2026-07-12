# Wave 9 Spec 09 — Ecosystem Hardening (The Substrate, Made Publishable)

**Mission:** Harden everything about the substrate that is NOT PH-domain work: key material, file hygiene, monster files, tool-noise, and the repo-hygiene/cloud-push milestone that Wave 10 explicitly gates on. After this wave the repo is fit to leave this machine.
**Sequencing:** runs AFTER Spec-08 (Residue Zero) is merged. C2 (CRLF normalization) is deliberately the LAST code commit of the wave — a repo-wide line-ending rewrite is merge poison for any in-flight branch, so it lands when nothing is in flight, immediately before the push milestone.
**Repo:** `asymmflow-oss`. **Branch:** `feat/fable-wave9-9-ecosystem-hardening` off `main`. Do not merge or push; leave for owner review. The cloud push itself (C7) is OWNER-EXECUTED — this wave only makes it safe.
**Authority documents, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` → `FABLE_WAVE9_UIUX_AUDIT.md` (keep-lists) → this spec.
**Prior art:** all prior reports; Spec-07's B7(g) security note and Spec-05's C1/C4 findings are direct inputs.

## 0. Read before anything

1. `CLAUDE.md` — synthetic-data invariant. C7's whole job is proving it held: no client identity, credentials, or figures anywhere in tracked history.
2. Spec-07 report §4 security note — the plaintext hardware-ID sidecar is an ACCEPTED interim; C1 is its ratified successor (Spec-07 §7.2, owner-approved 2026-07-12).
3. Decomposition rule (C3/C4): behavior-preserving means BYTE-preserving where possible — extract, don't rewrite. The audit keep-lists apply to every extracted component.

## 1. Operating model

Identical to Specs 02–08: Opus 4.8 orchestrator; Sonnet 5 coders in file-disjoint batches; Phase-A recon first; constitutional review of every diff; central gating + bindings regen.

**Lessons inherited (do not relearn):** anchors drift — verify before coding · `git checkout -- frontend/dist/index.html` after builds · gate baseline: vite clean, svelte-check 0 errors/14 warnings, `go build`/`go vet` clean, full `go test -count=1 -timeout 1800s ./...` green · monster files get one coder · bindings regen central · CRLF-only no-op binding diffs get reverted (Waves 9.4/9.5 precedent) — until C2 lands, then the problem is gone at the root.

## 2. Phase A — recon (read-only, do first; verdicts in the report)

| # | Question | Feeds |
|---|---|---|
| A1 | Key-material surface: every consumer of `getHardwareID()`; the exact DPAPI call surface available from Go on Windows (golang.org/x/sys/windows CryptProtectData, or the smallest audited dependency); what macOS/Linux fallbacks the codebase can honestly support; the migration path for installs that already have a plaintext `.hardware_id` sidecar (Wave 9.7). | C1 |
| A2 | Line-ending census: how many tracked files are CRLF vs LF by type; what `.gitattributes` policy fits (Go LF, Svelte/TS LF, `.bat`/`.ps1` CRLF?); whether `gofmt`/`wails generate` will fight the chosen policy; the exact renormalization sequence that produces ONE clean mechanical commit. | C2 |
| A3 | WorkHub.svelte map: current line count, its natural seams (task board, project panel, approvals queue, member management), which seams share state, and the extraction order that keeps every intermediate commit green. Same for CustomerDetailView. | C3, C4 |
| A4 | Activity-monitoring placement: its current home (SalesHub Admin tab, developer-allowlist gated — Spec-05 C1), what a Deployment/ops home looks like, and whether the move is pure relocation (it must be). | C5 |
| A5 | staticcheck full run: the real count of `interface{}`→`any` + QF1012 + anything else informational; which are mechanical vs which touch generated/binding files that must be excluded. | C6 |
| A6 | Publishability audit: scan full tracked HISTORY (not just HEAD) for secrets, tokens, client names/figures, real PII, absolute local paths; the seed/dev DB story (is any `.db` tracked? should it be?); `.gitignore` completeness; LICENSE/README state; anything `wails build` embeds that shouldn't ship. | C7 |

## 3. Phase B — the hardening ledger

**C1 — Hardware-ID → OS keystore (Spec-07 §7.2, RATIFIED).**
Replace the plaintext sidecar as the PRIMARY store: Windows = DPAPI (machine+user scope — a stolen DB directory no longer carries the key-derivation input). Per A1, non-Windows platforms may honestly keep the file sidecar (documented as such) — do not fake a keystore.
**Migration (the hard part, get it right):** existing installs have a plaintext sidecar whose value ALREADY derives their field-crypto key. On first boot with C1: read the sidecar → re-protect via DPAPI → verify round-trip decrypt → only then delete the plaintext file. If DPAPI fails, KEEP the sidecar and log — never strand key material between stores. The derivation formula and the resolved VALUE never change; only the wrapping does.
**AC:** fresh install → DPAPI-only; upgraded install → migrated with round-trip verification, plaintext gone; DPAPI-failure path leaves the old sidecar intact; tests cover all three via the existing path-override seam.

**C2 — Repo-wide CRLF normalization (deferred since Spec-03; LAST commit of the wave).**
Per A2's policy: commit `.gitattributes`, renormalize (`git add --renormalize .`), one mechanical commit touching line endings ONLY — zero content changes, verified by `git diff --ignore-cr-at-eol` being empty. Full gate immediately after (gofmt/vite/svelte-check must not fight the policy).
**AC:** one commit, endings-only; all gates green after; `wails generate module` no longer produces CRLF-noise no-op diffs.

**C3 — WorkHub decomposition (deferred since the audit).**
Extract along A3's seams into components with the existing naming/token idioms; every intermediate commit compiles and passes svelte-check. Target: no file over ~800 lines, zero behavior change (keep-lists binding — task-delete two-press, ContextTaskModal, Team Board all byte-identical in behavior).
**AC:** WorkHub.svelte is a thin composition root; every extracted component has one owner-seam; svelte-check 0 errors; no visual or behavioral diff.

**C4 — CustomerDetailView decomposition.** Same rules as C3, same AC, its own coder (monster files get one coder each).

**C5 — Activity-monitoring relocation (Spec-05 §Q3).**
Move it from the SalesHub Admin tab to the Deployment/ops surface A4 identifies. Pure relocation: same component, same developer-allowlist gate, same backend. SalesHub Admin keeps the opportunity edit-conflict tools (those ARE sales-admin).
**AC:** cross-department surveillance no longer lives under Sales; gate unchanged; conflict tools stay put.

**C6 — staticcheck sweep.**
Mechanical: `interface{}`→`any`, QF1012, and whatever else A5 confirms is content-neutral. Exclude generated/binding files. One commit, no logic changes.
**AC:** staticcheck informational count ~0 on hand-written code; `go test` green; diff is grep-verifiably mechanical.

**C7 — Publishability milestone (REPORT + FIXES; the push itself is the owner's).**
Close every A6 finding: untrack what shouldn't be tracked, complete `.gitignore`, write/refresh README (what the substrate is, how to build, the overlay/synthetic-data model), confirm LICENSE intent with the owner (report question if unset). If HISTORY contains a real secret/PII (not just HEAD), report it with options (BFG/filter-repo vs rotate-and-accept) — history rewriting is OWNER-DECIDED, never unilateral.
**AC:** a fresh clone builds with documented steps; the tracked tree + history scan comes back clean or every exception is reported with a recommendation; the owner can run `git push` with confidence the same day.

**C8 — Wave 9.8 residue intake (small, honest closures).**
(a) `ResolveGRNDiscrepancy` is a stub that persists nothing (the Spec-08 A1 finding; discrepancies actually live as SupplierIssues, resolved via `ResolveSupplierIssue`). Make the surface honest: either delete the stub pair's dead halves and document SupplierIssue as THE discrepancy record, or persist for real — whichever the recon shows is truthful. No new screens.
(b) B4's encrypted document number is excluded from collaborative sync (`json:"-"` on `DocNumberEncrypted`) — a deliberate PII-minimization. Ratify and RECORD it (constitution ratifications log, next to the Offer.Stage entry) so a future wave doesn't "fix" it into a leak; if the owner instead wants sync, that's a separate authorization, not this wave.
(c) `TestGetHardwareID_ByteIdenticalToWmic` fails environmentally on dev machines where wmic returns the BIOS placeholder ("Default string"). C1's keystore work touches exactly this surface — while there, make the test skip honestly when wmic yields a known placeholder (an environmental skip with reason beats a standing red that trains people to ignore the suite).
**AC:** no stub that silently drops data; the sync exclusion is recorded law; the suite runs fully green on this dev machine.

**Boundary note — cloud-sync transport:** the current DSN-based remote-PostgreSQL sync layer (Supabase/duckdns route) is slated for replacement by holesail.io P2P tunnels under `FABLE_CAMPAIGN_SOVEREIGN_MESH.md`. That is campaign scope, NOT this wave: do not invest hardening effort in the old transport beyond what C7's publishability scan requires (no credentials in tracked files). The B5 toolkit's `postgres://` support (main `e011260`) already covers the operational need against today's sync layer.

## 4. Hard boundaries

- **Zero domain-behavior change.** No flow, permission, financial, or schema movement anywhere in this wave (C1 changes key WRAPPING only — the derived key must decrypt existing data, proven by round-trip tests).
- C2 is last; C3/C4 never run concurrently with C2; nothing branches off main mid-C2.
- No history rewriting, no push, no tag, no merge — C7 findings that need them are reported, not executed.
- Keep-lists + all Wave 9.x shipped behavior binding. No sensory/brand work (Wave 10 next).
- Decompositions extract; they do not redesign. Any "while I'm here" improvement is a report line, not a diff.

## 5. Definition of done + status report

Done = Phase A verdicts; C1–C7 shipped or explicitly skipped with reason; the publishability scan verdict printed; gates green on the final commit (which is C2's normalization, followed only by the report commit).

Write `FABLE_WAVE9_SPEC_09_REPORT.md`, commit it, and paste it verbatim as your final message (established template + the publishability verdict as its own section). Severity honesty is law: an accurate red beats a false green.

**After this wave:** owner reviews, merges, pushes — and then, ship tight, Wave 10 (Sensory & Brand) begins.
