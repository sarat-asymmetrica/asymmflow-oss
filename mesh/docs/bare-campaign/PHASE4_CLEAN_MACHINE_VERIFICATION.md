# Phase 4 — Clean-Machine Verification (CAMPAIGN_REPORT §8.3)

**Date:** 2026-07-20 · **Owner-executed** (per Phase 4 reservation) · **Verifier:** the
kit's own `verify_clean_machine.cmd` (shipped in-kit since `695243c`)

## Round 1 — Windows Sandbox: PASS ✅

Run by the owner on Windows Sandbox (build 10.0.26100, `WDAGUtilityAccount`), kit copied
to the sandbox desktop from a read-only mapped folder, verifier double-clicked.

```
A: ok, not resolvable: node / npm / npx / bare
A: MACHINE IS CLEAN for the Node-free claim.
A: informational: vcruntime140.dll in System32 = False
C: probe control correctly NOT ok (verdict=CONTENT_FAIL)
D: TALLY  OK=16/16  HANG=0/16  CONTENT_FAIL=0/16
=== VERDICT: PASS -- CLEAN machine, 16/16 content-verified ceremonies. ===
```

**Findings:**

1. **The Node-free claim held on a machine where `node`/`npm`/`npx`/`bare` were
   provably unresolvable.** 16/16 content-verified ceremonies, zero hangs, and the
   probe control went red first — the pass is from an instrument that can report
   failure.
2. **`vcruntime140.dll` was absent from System32 and the kit ran anyway.** Strong
   evidence `bare.exe` does not depend on a separately-installed VC++ runtime — the
   single most common "works only on dev machines" Windows failure. Caveat kept
   honest: Sandbox shares host OS binaries copy-on-write, so a WinSxS-resolved copy
   cannot be fully excluded from inside the sandbox; Round 2 on real non-dev hardware
   settles it.
3. Timing: ~1 ceremony/second, no first-run Defender stall observed in the sandbox.

**Scope caveat (why Round 1 alone does not close the gate):** Windows Sandbox is
ephemeral and definitely Node-free, but it is a copy-on-write view of the host OS, not
a from-scratch install.

## Round 2 — REDEFINED: real field hardware (owner ruling, 2026-07-20)

The planned Hyper-V fresh-install VM was dropped (host disk constraints on the dev
N100 machine; the install starved system storage and was killed). Owner ruling:
**Round 2 runs directly on the receptionist's machine** — a real business PC that has
never had Node — using the identical in-kit verifier. This is *more* representative
than a VM: it is the actual class of hardware the corridor ceremony targets, and the
verifier records its own cleanliness evidence either way (`VERIFY_EVIDENCE.txt` +
`verify-logs\` come back as the artifact).

Field notes for Round 2:
- If the zip arrives via browser download, mark-of-the-web will likely trigger
  SmartScreen on first launch ("More info → Run anyway") — record what it does; this
  feeds the deferred Authenticode decision (DP2 §9). A USB copy typically carries no
  MOTW and skips it.
- Expect a slower first run if Defender scans the 45 MB unsigned exe; note it for the
  runbook, it is not a failure.
- If the window flashes and dies: run `bare.exe -e "console.log(1)"` from a terminal
  to surface the real error; correlate with the vcruntime line in the evidence.

## Status

- §8.3 honesty gap: **half-closed** (clean-by-evidence pass on ephemeral Windows;
  real-hardware pass pending at the receptionist machine).
- The verifier itself is field-proven: correctly reported the dev machine NOT CLEAN,
  the sandbox CLEAN, and its control run red in both environments.
