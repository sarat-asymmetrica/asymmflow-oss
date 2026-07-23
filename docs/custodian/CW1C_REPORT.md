# CW1-C Mission Report — Public CI (C7)

**Wave:** Custodian 1 "The Existential Floor"
(`FABLE_CUSTODIAN_SPEC_01_THE_EXISTENTIAL_FLOOR.md`) · **Mission:** CW1-C
**Branch:** `feat/fable-custodian-w1` · **Date:** 2026-07-23

## What was built

1. **`.github/workflows/gate.yml`** — three jobs on `push`/`pull_request` to
   `main`: `go` (`windows-latest`: `go build ./...`, `go vet ./...`,
   `go test ./...`), `frontend` (`ubuntu-latest`: `npm ci` + `npm run build`
   in `frontend-lab/`), `mesh` (`ubuntu-latest`: `npm ci` + `npm run smoke` in
   `mesh/`). Secretless — zero `secrets.*` references anywhere in the file.
2. **`docs/custodian/CI_LAW.md`** — the CI-floor-vs-local-gate boundary, the
   runner-OS decision with measured timings, the secretless doctrine, the
   contributor-facing status-check list, the exclusion list with reasons, the
   tagging convention.
3. **`CHANGELOG.md`** — `[Unreleased]` section + reconstructed headline
   entries for the major merged waves (India W1 back through Wave 10), plus
   the tagging convention (semver, tag-at-merge-on-main, no tags this wave).

## Workflow design decisions

- **Go job runner: `windows-latest`, single job, no split.** Measured
  locally: `go build ./...` 2m42.4s, `go vet ./...` 0m57.0s,
  `go test ./...` 5m55.8s (81 packages, 0 failures) — combined ~9m35s. Well
  under a sensible CI budget; a Windows/Linux split would add a second
  checkout/setup-go cold start for no measured benefit, and the OS-coupled
  packages (`hardware_id_keystore_windows.go`, `command_window_windows.go`)
  need Windows regardless. Full reasoning + numbers in `CI_LAW.md` §2.
- **Go version pin:** `go.mod` declares `go 1.25.0` with no separate
  `toolchain` directive, so `actions/setup-go@v5` pins `go-version: "1.25.0"`
  exactly, with `cache: true` for module caching.
- **Frontend job:** `frontend-lab/` is the real Wails frontend
  (`wails.json` → `frontend:dir`). `npm ci` (from the committed
  `frontend-lab/package-lock.json`) + `npm run build` (vite) is deterministic
  and headless. Measured: `npm ci` 25.3s, `npm run build` 28.8s (vite build
  alone 16.85s). Put on `ubuntu-latest` since it is not OS-coupled — cheaper
  than burning Windows minutes on a pure JS/TS build.
  `vitest run`/`svelte-check` were deliberately NOT wired in this wave — see
  Findings/Residue.
- **Mesh job:** scoped to `npm run smoke` only (builds `dist/reducer.wasm`
  then runs `host/smoke.mjs`), the exact command `mesh/README.md`'s own "Run
  it" section documents as the floor check. Measured: `npm ci` 39.6s,
  `npm run smoke` 5.6s. Everything else under `mesh/host`/`mesh/kit` that
  opens a swarm/DHT/Holesail tunnel or spawns a second peer process
  (`*holesail*`, `peer.mjs :host`/`:join`, `mirror-spike.mjs`,
  `reactor-parity-spike.mjs`, `kit-net.mjs`, `bare-net*`, `bare-corridor*`,
  `anchor*`, the rest of the `*spike*` family) is explicitly excluded — AMBER
  by nature per spec §CW1-C, not deterministic on a shared CI runner.
  `go test ./mesh/...` is not duplicated as a separate job since it already
  runs inside the `go` job's `go test ./...`.
- **No env-coupled tests required new CI-added skips.** Grepped every
  `os.Getenv(` call inside `*_test.go` files: every manual/opt-in test
  (`B2_STOCK_*`, `ONEDRIVE_IMPORT_*`, the `*_COMMIT` family,
  `APPLY_SUPABASE_SCHEMA`, etc.) already self-skips its destructive/external
  branch when the var is unset — CI sets none of them. The hardware-ID/`wmic`
  tests use pre-existing honest-skip guards
  (`hardware_id_test.go:28,37,49,52`, `hardware_id_keystore_test.go:46,103,216`)
  and run for real on `windows-latest` since it has a genuine baseboard
  serial. No test was found that hard-fails secretless — nothing to
  stop-and-report on this front.

## Local verbatim proof — green

All three commands run from a clean shell in the worktree, exactly as the
workflow runs them, sequentially, on `feat/fable-custodian-w1`:

```
$ cd C:/Projects/asymmflow/asymmflow-custodian
$ date && time (go build ./...)
Thu Jul 23 13:23:47 IST 2026
real    2m42.364s
user    0m0.107s
sys     0m0.171s
$ echo EXIT:$?
EXIT:0
```

```
$ date && time (go vet ./...)
Thu Jul 23 13:26:39 IST 2026
real    0m56.952s
user    0m0.000s
sys     0m0.123s
$ echo EXIT:$?
EXIT:0
```

```
$ date && time (go test ./...)
Thu Jul 23 13:27:43 IST 2026
...
ok      ph_holdings_app                                340.543s
ok      ph_holdings_app/integration                    19.162s
ok      ph_holdings_app/internal/viewmodel/cashflow    2.382s
[... 81 packages total, all "ok", 0 FAIL, 0 panic — full transcript in
     go test log captured during this session ...]
real    5m55.844s
user    0m0.015s
sys     0m0.000s
DONE_EXIT:0
```

Frontend build:

```
$ cd frontend-lab && date && time (npm ci)
Thu Jul 23 13:28:33 IST 2026
added 77 packages, and audited 78 packages in 23s
real    0m25.325s
$ date && time (npm run build)
Thu Jul 23 13:29:38 IST 2026
✓ 359 modules transformed.
✓ built in 16.85s
real    0m28.768s
EXIT:0
```
(`npm audit` reported 7 pre-existing vulnerabilities in transitive deps —
not a build failure, not addressed by this wave; noted as a finding below.)

Mesh smoke:

```
$ cd mesh && date && time (npm ci)
Thu Jul 23 13:27:49 IST 2026
added 188 packages, and audited 189 packages in 38s
real    0m39.631s
$ date && time (npm run smoke)
Thu Jul 23 13:29:08 IST 2026
built .../mesh/dist/reducer.wasm
  ✓ boundary: reducer.wasm ran and returned JSON
  ✓ state / invariant / convergence / golden — 11/11 checks
SMOKE GREEN ✅
real    0m5.563s
EXIT:0
```

(`npm ci` in `mesh/` emitted `npm warn tar TAR_ENTRY_ERROR ENOENT` lines for
`bare-crypto` — cosmetic npm/tar warnings during extraction of an optional
native-fallback package, not failures; `npm ci` still exited 0 and
`npm run smoke` passed clean. Noted as a finding below in case it becomes a
real problem on a fresh GitHub-hosted runner.)

Both `npm ci` runs (mesh + frontend-lab) left `package-lock.json` byte-
unchanged except one transient dependency-graph line in `mesh/package-lock.json`
(a `bare-tcp` entry appearing under one package's `devDependencies` list) and
`frontend-lab/dist/index.html`'s asset hash (rebuilt by `npm run build`) —
both reverted with `git checkout --` before finishing this mission, verified
byte-identical to the pre-run tree by `git status`.

## Local verbatim proof — red, then reverted

Deliberately broke `TestBHDConstructorPrecision` in
`pkg/kernel/money/money_test.go` (want-value changed from the correct
`125556` to a wrong `999999`, comment/message left untouched):

```diff
-       if a.Minor() != 125556 {
+       if a.Minor() != 999999 {
```

Package-scoped run (fast confirmation):

```
$ go test ./pkg/kernel/money/...
--- FAIL: TestBHDConstructorPrecision (0.00s)
    money_test.go:16: minor units: got 125556, want 125556
FAIL
FAIL    ph_holdings_app/pkg/kernel/money       0.553s
EXIT_CODE:1
```

Then the exact workflow command (`go test ./...`) run against the same
broken tree, verbatim:

```
$ date && time (go test ./...)
Thu Jul 23 13:36:23 IST 2026
...
--- FAIL: TestBHDConstructorPrecision (0.00s)
    money_test.go:16: minor units: got 125556, want 125556
FAIL
FAIL    ph_holdings_app/pkg/kernel/money       1.110s
FAIL
...
real    4m23.664s
user    0m0.030s
sys     0m0.031s
EXIT_CODE:1
```
Confirms: the exact command the `go` CI job runs goes red (non-zero exit)
on the deliberately broken test, isolated to the one package touched — every
other package still reported `ok`.

Revert:

```
$ git diff --stat pkg/kernel/money/money_test.go
[no output — reverted to original]
$ git status --porcelain pkg/kernel/money/money_test.go
[no output — file untouched]
$ go test ./pkg/kernel/money/...
ok      ph_holdings_app/pkg/kernel/money       (cached)
EXIT:0
```
File confirmed byte-identical to its pre-experiment state; test confirmed
green again.

## YAML validation

```
$ python3 -c "
import yaml, json
with open('.github/workflows/gate.yml') as f:
    doc = yaml.safe_load(f)
print('YAML PARSE OK')
print(json.dumps(list(doc.keys())))
print('jobs:', list(doc['jobs'].keys()))
"
YAML PARSE OK
["name", true, "concurrency", "jobs"]
jobs: ['go', 'frontend', 'mesh']
```

Note: PyYAML (YAML 1.1) parses the bare `on:` key as the boolean `True` —
this is the well-known YAML 1.1 `on/off/yes/no` sigil quirk, not a workflow
error. GitHub Actions' own workflow parser treats a top-level `on` key
specially regardless of this ambiguity (every GitHub Actions workflow in
existence uses unquoted `on:`), so this is purely a PyYAML-vs-GHA-parser
difference, not a defect — recorded here for full honesty rather than
silently omitted.

Grep for secret-shaped references:

```
$ grep -ni "secrets\.\|SECRET\|TOKEN\|API_KEY\|PASSWORD" .github/workflows/gate.yml
4:# Secretless by doctrine: no `secrets.*` reference anywhere in this file...
63:  # not a secret). Measured locally: npm ci ~25s, vite build ~17s.
```
Only doctrine comments match — zero actual `secrets.*`/credential references.

## Findings

1. **`npm audit` reports 7 vulnerabilities (5 moderate, 1 high, 1 critical)**
   in `frontend-lab`'s transitive dependency tree. Pre-existing (not
   introduced by this wave), not gated by the new workflow (a build-only
   job, not `npm audit`). Flagged for the owner/a future wave — CI does not
   currently fail on this.
2. **`bare-crypto` tar-extraction warnings during `mesh/ npm ci`**
   (`npm warn tar TAR_ENTRY_ERROR ENOENT ... bare-crypto/...`) — cosmetic on
   this machine (exit 0, smoke still green), but unverified on a fresh
   GitHub-hosted `ubuntu-latest` runner with no local npm cache. If the mesh
   job goes red on first cloud run with a `bare-crypto`-shaped error, this is
   the suspect.
3. **Nothing hard-fails secretless in the Go suite** — the manual/opt-in and
   hardware-ID test families all self-skip via pre-existing guards; no
   stop-and-report needed on the CI-secretless front.

## Residue

- **First cloud-green run of `gate.yml` is PENDING PUSH.** Every command
  above was proven locally, verbatim, on the dev machine. This report does
  NOT claim the GitHub Actions cloud run has been observed — that only
  happens once `feat/fable-custodian-w1` (or a subsequent branch carrying
  this file) is actually pushed and the Actions tab shows a run. Post-push
  checklist item, owner-cadence.
- **Branch protection / required-status-checks on `main`** is a GitHub repo
  settings action, not a file change — not enabled by this wave.
- **`frontend-lab` test suite (`vitest run`) and `svelte-check`** not wired
  into CI — left as residue rather than added without measuring flake risk.
- **`npm audit` findings** (finding 1 above) not remediated this wave —
  scope was CI wiring, not dependency hygiene.
- **Wails desktop build / installer** not in CI — belongs to the release
  ceremony (ledger item C8), out of scope here.
