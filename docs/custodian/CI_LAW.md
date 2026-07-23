# CI Law — what `.github/workflows/gate.yml` enforces, and what it doesn't

**Wave:** Custodian 1 "The Existential Floor" (`FABLE_CUSTODIAN_SPEC_01_THE_EXISTENTIAL_FLOOR.md`),
mission CW1-C (item C7 on `CUSTODIANS_LEDGER.md`).
**Date:** 2026-07-23 · **Branch:** `feat/fable-custodian-w1`.

## 0. The floor, not the gate

CI is a **floor**: the minimum bar every push/PR to `main` clears automatically,
secretlessly, before a human looks at it. It is not a replacement for the full
local wave-gate discipline this project already runs (spikes, N≥16 acceptance
harnesses, mesh network legs between real peers, drive-through ceremonies,
independent re-verification by the orchestrator). Those stay manual/local — CI
proves the parts that are deterministic, secretless, and fast enough to run on
every push.

## 1. Secretless doctrine

Per spec §0 stop-and-report: **the gate floor must run secretless.** `gate.yml`
contains zero references to `secrets.*` (grepped and reverified at CW1-G).
Nothing in the workflow requires `ENCRYPTION_MASTER_KEY`, Azure/Supabase
credentials, a Mistral key, or any other repo secret. If a future test demands
one to pass, that is a **finding to report**, not a secret to upload to GitHub.
No such test exists today — see §3.

## 2. What runs, on what, and why

| Job | Runner | Commands | Why this runner |
|---|---|---|---|
| `go` | `windows-latest` | `go build ./...`, `go vet ./...`, `go test ./...` | AsymmFlow ships as a Windows Wails desktop app; several packages are OS-coupled by build tag (`hardware_id_keystore_windows.go` / `_other.go`, `command_window_windows.go`). Running the full suite on the product's actual OS is the honest floor, and the measured wall-clock (below) is small enough that a Windows/Linux split is not needed. |
| `frontend` | `ubuntu-latest` | `npm ci` + `npm run build` (vite) in `frontend-lab/` | Pure JS/TS build, not OS-coupled; wails.json's `frontend:dir` build step. Cheaper on Linux, no reason to burn Windows minutes on it. |
| `mesh` | `ubuntu-latest` | `npm ci` + `npm run smoke` in `mesh/` | The deterministic Go→WASM→JS reducer boundary check `mesh/README.md` itself documents as "Run it." Pure-JS/WASM, not OS-coupled. |

### Runner-split decision (measured, not assumed)

Local wall-clock on the dev machine (`asymmflow-custodian` worktree, warm-ish
module cache, 2026-07-23):

| Command | Wall-clock |
|---|---|
| `go build ./...` | 2m42.4s |
| `go vet ./...` | 0m57.0s |
| `go test ./...` | 5m55.8s (81 packages, 0 failures) |
| `frontend-lab`: `npm ci` | 0m25.3s |
| `frontend-lab`: `npm run build` | 0m28.8s (vite build alone: 16.85s) |
| `mesh`: `npm ci` | 0m39.6s |
| `mesh`: `npm run smoke` | 0m5.6s |

Combined Go floor (build + vet + test, run sequentially as three separate
steps) measures **~9m35s total** (2m42s + 0m57s + 5m56s) on the dev machine.
That is well under a single job's sensible CI budget (GitHub Actions default
job timeout is 6 hours; a floor job in single-digit-to-low-double-digit
minutes keeps the PR loop tight). At these numbers a Windows/Linux package
split buys nothing — it would add a second checkout + setup-go cold-start for
no wall-clock win, since the whole suite already compiles and runs in well
under 10 minutes on Windows alone, and splitting would still need the
OS-coupled packages (hardware ID, command-window) on Windows regardless.
**Decision: one `windows-latest` job for the whole Go floor.** Revisit if the
measured total materially grows (rule of thumb: re-measure and reconsider the
split above ~15-20 minutes, or if a cloud run's timing diverges significantly
from this local measurement).

## 3. Known env-coupled / hardware-coupled tests — how CI handles them

The suite already contains two families of tests that behave differently with
and without environment/hardware, and **both self-skip using guards that exist
in the test files today** — CI adds no new skip logic, per spec (`"CI must
not invent new skips"`):

1. **Manual/opt-in tests** (`b2_stock_adjustment_*_test.go`,
   `manual_onedrive_*_test.go`, `manual_master_data_cleanup_test.go`,
   `manual_supabase_schema_test.go`, `manual_won_offer_repair_test.go`,
   `manual_payroll_expense_backfill_test.go`, `manual_opportunity_repair_test.go`,
   `manual_deployment_package_test.go`, `manual_butler_acceptance_test.go`,
   `manual_customer_reference_seed_test.go`, `manual_invoice_item_backfill_test.go`,
   `provision_seed_seedgen_test.go`, `deployment_audit_test.go`, and others) gate
   their destructive/slow/external-resource branches behind an explicit
   `if os.Getenv("SOME_COMMIT_OR_PATH_VAR") != "1" { t.Skip(...) }`-style guard.
   CI sets none of these vars, so these tests run their default (read-only /
   no-op) path exactly as they do on a clean local checkout with no `.env`.
2. **Hardware-ID / `wmic` tests** (`hardware_id_test.go`,
   `hardware_id_keystore_test.go`, `hardware_id_keystore_windows.go`-adjacent
   tests): use pre-existing honest-skip guards — `t.Skip("no OS keystore on
   this platform...")`, `t.Skip("wmic unavailable within timeout...")`,
   `t.Skipf("wmic returned a known BIOS/SMBIOS placeholder serial...")`. On
   `windows-latest` these run for real (GitHub's Windows runners expose a real
   `wmic`/`Get-CimInstance` baseboard serial); if a given runner image lacks
   one, the pre-existing skip fires — not a CI-invented skip.

No test in the suite was found that hard-fails (rather than skips) when a
secret/env var is absent. If CW1-G's independent re-run finds one, it is a
stop-and-report finding, not something patched into the workflow.

## 4. What is deliberately excluded from CI, and why

Every exclusion below is a public, named decision — not a silently dropped
check.

- **`mesh/host/*holesail*.mjs`, `mesh/host/peer.mjs` (`:host`/`:join` scripts),
  `mesh/host/mirror-spike.mjs`, `mesh/host/reactor-parity-spike.mjs`,
  `mesh/kit/kit-net.mjs`, `mesh/kit/bare-net*.mjs`, `mesh/kit/bare-corridor*.mjs`,
  `mesh/kit/anchor*.mjs`, and the rest of the `*spike*` family under
  `mesh/host`/`mesh/kit`** — these open a Hyperswarm/DHT/Holesail tunnel,
  spawn a second peer process, or otherwise depend on live P2P networking
  between ≥2 machines. They are **AMBER by nature** (spec §CW1-C): correct by
  design, but not deterministic in a shared CI runner (NAT/firewall variance,
  DHT bootstrap flakiness, no second peer to talk to). They stay part of the
  full local wave-gate, run by a human/orchestrator on real hardware, never in
  `gate.yml`.
- **`go test ./mesh/...` as a standalone mesh job** — not excluded, just not
  duplicated: it already runs inside the `go` job's `go test ./...`.
- **`frontend-lab` test suite (`vitest run`) and `svelte-check`** — NOT wired
  into CI this wave. `npm run build` (the thing wails.json actually needs to
  produce a shippable frontend) is proven; `vitest`/`svelte-check` were not
  exercised as part of this floor and are left as residue rather than wired in
  without having measured them for flake risk. See `CW1C_REPORT.md` residue.
- **Wails desktop build (`wails build`)** — NOT in CI. It requires the full
  Wails/CGO-adjacent Windows toolchain and produces a signed installer
  artifact; that belongs to the release ceremony (ledger item C8), not the
  push/PR floor. Out of scope for this wave.
- **DPAPI/keystore round-trip on a genuinely different machine, foreign
  hardware restore** — cannot be proven in CI at all (single ephemeral runner
  per job); this is the same honesty boundary CW1-A/CW1-B state for the
  recovery rehearsal and restore drill.

## 5. What a contributor sees on a PR

Three required status checks: `go (windows-latest)`, `frontend-lab build`,
`mesh smoke`. All three must go green before merge is sensible (branch
protection enabling "required" status checks is an owner action, not enabled
by this wave — recorded as residue). A red `go` job means build, vet, or a
real (non-skipped) test failure — the contributor re-runs the same three
commands locally (`go build ./...`, `go vet ./...`, `go test ./...`) to
reproduce byte-for-byte, since CI runs nothing the contributor cannot run
themselves.

## 6. Tagging / release convention (see also `CHANGELOG.md`)

Semantic versioning (`vMAJOR.MINOR.PATCH`), matching `wails.json`'s
`info.productVersion` (currently `2.3.0`). A tag is cut at a merge commit on
`main`, never on a feature branch. **No tags were created this wave** — C7
seeds the convention and the CHANGELOG; the first tagged release is a future,
owner-gated action.

## 7. Residue (see `CW1C_REPORT.md` for the full list)

- First cloud-green run of `gate.yml` is **pending push** — this wave proves
  every command verbatim locally; it does not and cannot claim the GitHub
  Actions cloud run works until the branch is actually pushed and the
  Actions tab shows it.
- Branch protection / required-status-check enforcement on `main` is an owner
  GitHub-settings action, not part of this wave's file changes.
- `frontend-lab` test/typecheck jobs, Wails desktop build, and mesh's wider
  network-dependent floor are named exclusions above, not silently dropped.
