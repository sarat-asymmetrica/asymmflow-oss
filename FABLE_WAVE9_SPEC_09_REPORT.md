# Wave 9 Spec-09 — Ecosystem Hardening — STATUS REPORT

**Branch:** `feat/fable-wave9-9-ecosystem-hardening` (off `main`) — **not merged, not pushed** (owner review pending).
**Operating model:** Opus 4.8 orchestrator + Sonnet-5 coders in file-disjoint batches; Phase-A recon first; constitutional review + independent verification of every diff before commit; central bindings regen.
**Outcome:** All Phase-A verdicts delivered; **C1–C8 shipped**; C2 landed as the final code commit; publishability CRITICAL fixed at HEAD; gates green. One publishability item (history purge) is correctly deferred to the owner at push time (never rewritten unilaterally).

---

## 1. Commits (oldest → newest)

| Commit | Task | Summary |
|---|---|---|
| `6fdecfc` | C8(a) | GRN-discrepancy surface made honest — SupplierIssue is THE record; dead stubs removed |
| `b54a769` | C1 + C8(c) | Hardware-ID sidecar wrapped in DPAPI keystore + migration; wmic test honest-skip |
| `e7de0ab` | C3 | WorkHub.svelte 2791→1443 + 7 extracted components |
| `9c4f1f3` | C4 | CustomerDetailView.svelte 919→291 + 9 extracted components |
| `3c4189c` | C8(b) | DocNumberEncrypted sync-exclusion ratified into constitution |
| `b9eb3c8` | C6 | Mechanical staticcheck/modernize sweep (interface{}→any + 20 S-fixes) |
| `1acefee` | C5 | Activity-monitoring relocated Sales → Deployment/ops |
| `5c9840a` | C7 | Publishability fixes (synthetic-data + hygiene) |
| `429b882` | — | Central Wails bindings regen after C8(a) |
| `7e0e30b` | C2 | LF line-ending policy pinned via `.gitattributes` (**LAST code commit**) |

---

## 2. Phase A — recon verdicts

- **A1 (key material):** 3 consumers of `getHardwareID()` (`settings_service.go:45`, `field_crypto.go:63`, `auth_handler.go:854`), each deriving its own key from the same resolved value; live path is PBKDF2-HMAC-SHA256(600k)→HKDF→AES-256-GCM. `golang.org/x/sys/windows` already vendored — `CryptProtectData`/`CryptUnprotectData` available, **no new dependency**. Test seam = `hardwareIDSidecarPathOverride`. Non-Windows honestly keeps the plaintext sidecar (no CGO keychain exists to fake).
- **A2 (line endings):** The git **index is already 100% LF** (1496/1547 text files LF, rest binary/empty; zero CRLF/mixed in-index). On-disk CRLF is purely a `core.autocrlf=true` checkout artifact with nothing pinning it. C2 = land `.gitattributes`. Two anomalies (`manual_onedrive_seed_export_test.go` bare-CR; `frontend/dist/index.html` binary) both resolved — the former's BOM/CR was normalized by C6's gofmt pass, the latter pinned `binary`.
- **A3 (monster files):** WorkHub.svelte 2791 lines → 7 seams (task board, my-work, team-board, task-modal, member-mgmt, projects, helpers); CustomerDetailView.svelte 919 lines → sidebar/tabs/strips + root-owned Escape chain. Extraction order that keeps every step green documented and followed.
- **A4 (activity monitor):** `DeploymentHub.svelte` already exists as the ops surface. The Efficiency panel (`GetWeeklyUserActivityReport`, gated `CanViewUserActivityMonitoring`) moves there; the Conflicts tools stay in SalesHub. `activityMonitor.ts` (write-side collector) must not move. Flagged: DeploymentHub's route adds an outer `settings:update` gate → additively stricter (intended).
- **A5 (staticcheck):** `interface{}`→`any` is a `gofmt -r`/modernize rewrite (NOT a staticcheck check): 1936 occurrences / 205 hand-written files. QF1012 = **0**. 20 additional content-neutral S-simplifications. Out of scope (content/logic changes): U1000 dead code (125), ST1005 message literals (44), SA4006/4009 (17), QF1003 (19). Generated `schemas/go/**/*.capnp.go` excluded.
- **A6 (publishability):** History is already squashed to one never-pushed initial commit (`8f66639`); full-history secret scan **clean** (no keys/tokens/credentialed DSNs/`.db` ever). CRITICAL found: real client identities inside the binary import-template xlsx (evaded text scans). HIGH/MEDIUM: FABLE_WAVE1 narrative + collaborator paths + a real hostname in a test. See §4.

---

## 3. Phase B — the hardening ledger

**C1 — Hardware-ID → OS keystore (DPAPI) + migration. ✅ SHIPPED.**
Windows stores the sidecar as a DPAPI machine-scoped blob (`.hardware_id.dpapi`); the resolved hardware-ID VALUE and the field-crypto derivation are **unchanged** (proven by the FieldCrypto suite passing post-change). Resolution order: keystore → plaintext (migrate-on-read) → live re-resolve. Migration protects the plaintext value, writes the blob, **reads it back and CryptUnprotectData-verifies the round-trip decrypts to the same value**, and only then renames the plaintext to `.hardware_id.migrated` (rollback-safe). Any protect/write/verify failure leaves the plaintext fully intact and logs — key material is never stranded. Fresh Windows install = DPAPI-only, no plaintext ever written. Non-Windows = honest plaintext passthrough (`keystoreAvailable()=false`), no fake keystore, no CGO. Tests cover all three AC scenarios via the override seam.
**Review note (crypto — owner may weigh in):** the coder chose `CRYPTPROTECT_LOCAL_MACHINE` (machine scope), not per-user. Rationale accepted: this ERP can run under a service account or a different Windows user; per-user DPAPI would fail to decrypt across a user switch → live re-resolution → a *different* hardware ID → **existing field-crypto data would become undecryptable**. Machine scope avoids that data-loss footgun while still meeting the stated security property (a stolen DB directory no longer carries the key-derivation input). `.hardware_id.migrated` intentionally retains the old plaintext for rollback (spec-authorized "rename over delete").

**C2 — Repo-wide CRLF normalization. ✅ SHIPPED (last code commit).**
Index was already 100% LF, so `git add --renormalize .` staged **zero** content changes beyond `.gitattributes` (verified: `git diff --cached --ignore-cr-at-eol` excluding `.gitattributes` is empty). Policy: `* text=auto eol=lf` + explicit source-type pins + binary declarations + `frontend/dist/** binary`. This removes the root cause of the recurring no-op `wails generate` CRLF diffs. Endings-only, gates green after.

**C3 — WorkHub decomposition. ✅ SHIPPED.**
2791 → 1443 lines; 7 components under `components/workhub/` (all ≤572 lines). Keep-list behaviors verified intact: two-press task delete (root-owned toggle), Team Board drag-and-drop (state fully local to TeamBoardPanel, zero root residue), project archive/shelve/delete required-reason flow. `ContextTaskModal` untouched. svelte-check 0/14.
**AMENDMENT (flagged):** the composition root remains **1443 lines (> ~800 target)** because all CRUD/business-logic handlers stay in it; a handler→actions-`.ts` split is a safe follow-up, deliberately not rushed on a behavior-critical screen pre-publish. Every *extracted* component is well under 800.

**C4 — CustomerDetailView decomposition. ✅ SHIPPED.**
919 → 291 lines; 9 components under `components/customer/` (all <260 lines) — cleanly under target. The single `handleKeydown` Escape-priority chain (`showDeleteConfirm > showTaskModal > showNoteModal > showContactModal`) and all four modal booleans stay root-owned; children receive them as `$bindable()` props. Edit-mode button-hiding + click-outside-to-close preserved. svelte-check 0/14.

**C5 — Activity-monitoring relocation. ✅ SHIPPED.**
Efficiency panel extracted to `UserActivityMonitorPanel.svelte`, mounted under a new gated "Activity" tab in DeploymentHub; same `GetWeeklyUserActivityReport` backend, same `CanViewUserActivityMonitoring` allowlist gate on both tab and panel. SalesHub Admin keeps the opportunity edit-conflict tools; SalesAdminTools is now Conflicts-only. `activityMonitor.ts` untouched. **Additive stricter gate noted:** the panel now sits behind `settings:update` (DeploymentHub route) AND the allowlist — a superset restriction, not a widening.

**C6 — staticcheck/modernize sweep. ✅ SHIPPED.**
`interface{}`→`any` across 205 files (independently verified: removed `interface{}` == added `any` == **1930**, exact token balance) + 20 content-neutral S-simplifications (S1039/S1008/S1016/S1024/S1031/S1011/S1005). 62 additional files carry gofmt whitespace/import-sort/BOM normalization only. QF1012 = 0. **Honest residual (out of scope):** staticcheck is NOT driven to literal zero — U1000 (125), ST1005 (44), SA4006/4009 (17), QF1003 (19) all require content/logic changes beyond a mechanical sweep. Note for future sweeps: `pkg/data/phreconcile/phreconcile.go` and `pkg/overlay/overlay.go` were left not-fully-gofmt-canonical on purpose — a bare `gofmt -w` there corrupts an escaped-SQL `''` doc-comment literal into a curly quote.

**C7 — Publishability milestone. ✅ FIXES SHIPPED; history purge = OWNER action (see §4).**

**C8 — Wave 9.8 residue intake. ✅ SHIPPED.**
(a) `RaiseGRNDiscrepancy`'s dead `GRNDiscrepancy` struct + "Placeholder" lie removed; the never-persisting `GetGRNDiscrepancies`/`ResolveGRNDiscrepancy` stubs (and their CRMService wrappers) deleted; SupplierIssue documented as THE discrepancy record. Go model + migration kept (no schema movement); bindings regenerated centrally. (b) `DocNumberEncrypted` `json:"-"` sync-exclusion recorded in the constitution ratifications log. (c) `TestGetHardwareID_ByteIdenticalToWmic` now skips honestly on known BIOS/SMBIOS placeholder serials (this was the sole pre-existing baseline red — now green/skip).

---

## 4. Publishability verdict (its own section)

**Bottom line: HEAD is clean and a fresh clone builds with the documented steps. One owner-executed step remains before a public push — a history re-squash — because the pre-fix real-data blob still lives in the (never-pushed) initial commit.**

**Secrets / credentials across full tracked history:** CLEAN. No API keys, tokens, private keys, or credentialed connection strings at any commit; no `.env` with real values ever committed; no `.db` ever tracked. `wails build`'s `go:embed` only ever ships a placeholder `dist/index.html`.

**CRITICAL — real client identities in the binary import template (FIXED at HEAD).** `AsymmFlow_Data_Import_Template_2026_02_23.xlsx` carried real counterparty identities (a real customer, supplier, and bank — including a real TRN, IBAN, SWIFT/BIC, contact emails, and supplier product/order codes; literal values deliberately not reproduced here) in its example rows — inside `xl/sharedStrings.xml`, which is why every plain-text `git grep` missed it. Replaced all 25 real identifiers with synthetic canon (National Petroleum Co./NPC, Rhine Instruments, Demo Bank A canon banking, canon people/phones/addresses); 190 shared-string substitutions; workbook re-validated (12 sheets load); **zero residual real tokens** in the archive.
> **⚠ OWNER ACTION — history re-squash (your ratified choice).** Fixing HEAD does not remove the original real-data blob from commit `8f66639`; a plain `git push` would still publish it. Since history is already a single never-pushed commit, the clean path is to rebuild it as one fresh commit before pushing:
> ```bash
> # after this branch is reviewed & merged to main, from a clean main:
> git checkout --orphan clean-main
> git add -A
> git commit -m "AsymmFlow — public release (synthetic reference data)"
> git branch -D main && git branch -m main
> git push -u origin main            # first push of the purged history
> ```
> (Alternative if you want to preserve wave-by-wave history: `git filter-repo --path AsymmFlow_Data_Import_Template_2026_02_23.xlsx --invert-paths` then re-add the fixed file. Either way, this is your call — no history was rewritten in this wave.)

**MEDIUM — real colleague hostname (FIXED).** `device_service_test.go` hardcoded a collaborator's real machine hostname (×4, literal value not reproduced here) → synthetic `demo-workstation.local`; hash-stability test semantics unchanged.

**Doc scrub (owner-approved, DONE).** Stripped collaborator Rahul's absolute private-repo paths (`C:\Projects\rahul\CS-Invoice`, `PP_Killer`) from 5 wave docs — 11 occurrences; project names kept as credited reference work. Softened the FABLE_WAVE1 "Mission 0" narrative: marked the completed hygiene gate as historical and neutralized the "live business data / real books" framing (the repo ships synthetic-only). Rahul's authorship attribution (LICENSE-ROADMAP copyright) kept — legitimate co-author.

**LICENSE — CONFIRMED (owner).** AGPL-3.0 + LICENSE-ROADMAP (2-year rolling MIT) + CLA.md — present, consistent, intended.

**README — refreshed.** Added an Architecture / overlay-model section (kernel → engine → overlay); `.env.example` made honest (cloud sync is OPTIONAL; offline-first needs none); `.gitignore` now covers `.hardware_id*` key-material sidecars.

**NOTED — owner-optional (left as-is per your Q2 scope):** the maintainer's own absolute paths to the private sibling repo (`C:\Projects\asymmflow\ph_holdings`) appear in **8 tracked docs / 16 lines** as intentional methodology-exemplar references (per A6). These reveal machine layout + the private repo name but no client data. Your doc-scrub decision covered Rahul's paths + the WAVE1 narrative; these your-own-path references were left untouched — flag if you'd like them swept in a follow-up. Similarly `test_data/eh_sample_basket.xml` retains real Endress+Hauser **product order-codes** in an E+H shop-basket **format** (customer is already synthetic "Acme … Test Customer"); this matches the repo's own kept-public-formats policy (like the kept bank-statement parsers) and is unreferenced by any Go test — classified acceptable, noted for your awareness.

---

## 5. Gate results (final commit = C2 `7e0e30b`)

- `go build ./...` — **clean** (exit 0)
- `go vet ./...` — **clean** (exit 0)
- `svelte-check` — **0 errors / 14 warnings** (baseline-identical; all 14 in unrelated pre-existing files)
- `vite build` — **clean** (built in ~18s; the >500 kB chunk-size note is pre-existing informational, not an error); `frontend/dist` build artifacts reverted per standing practice
- `go test -count=1 -timeout 1800s ./...` — **green** (exit 0; **84 packages ok, 0 FAIL**; the wmic byte-identity test skips honestly per C8c)

Baseline before the wave had exactly one red — `TestGetHardwareID_ByteIdenticalToWmic` (wmic returns the BIOS placeholder `"Default string"` on this box) — now an honest environmental skip (C8c).

---

## 6. Hard-boundary compliance

- **Zero domain-behavior change:** no flow, permission, financial, or schema movement. C1 changes key *wrapping* only (derived key proven unchanged). C8(a) removed dead code + never-persisting stubs (no live flow altered). C5 is pure relocation (gate additively stricter, never widened).
- **C2 last:** no branch off main during it; nothing in flight; it staged endings-only.
- **No history rewriting / push / tag / merge:** the one history item is reported with ratified options for the owner, not executed.
- **Keep-lists + Wave 9.x behavior binding:** preserved and verified per screen. No sensory/brand work.
- **Decompositions extract, not redesign:** byte-faithful; the sole "while I'm here" line (WorkHub root size) is reported as an amendment, not silently changed.

---

## 7. Recommendation

Merge after review. Then, before the first `git push`, execute the history re-squash in §4 (your ratified choice) so the purged history is what goes public. After that: **ship tight → Wave 10 (Sensory & Brand).**
