# FP-3 — Field Staging & the Handover Note · Report

**Date:** 2026-07-23 · **Orchestrator:** Opus 4.8 · **Campaign:** Field Packet
(`f62ef37`)

## 1. The staging folder

`C:\Projects\asymmflow\FIELD_KIT_STAGING\` (outside the repo, nothing tracked):

| file | bytes | sha256 (first 18) |
|---|---|---|
| `AsymmFlow-SealedKit-Field-20260723.zip` | 24,179,118 | `130BF121870BF47CA9` (byte-identical to the FP-1 gated zip — verified by hash, not assumed) |
| `FIELD_PACKET.pdf` | 445,104 | `A8278FD8887B4BB1CA` (the FP-2 final cut) |
| `README_OWNER.txt` | 1,901 | owner checklist, one screen, plain ASCII |

`README_OWNER.txt` is the owner's six checkbox moves: USB → receptionist
machine → extract (Desktop or `C:\corridor\`, never a `#` path) → verifier
FIRST → evidence + SmartScreen observation back → only then the ceremony call,
packet in both humans' hands, A2.1 on standby, results into MSG-D29. A person
who has read nothing else can follow it; every deeper question is answered by
the packet it points to.

## 2. The results ledger — MSG-D29 DRAFT

`mesh/docs/MESSENGER_DECISIONS.md` gains **MSG-D29 (DRAFT)** — "Field results:
receptionist Round-2 + first India↔Bahrain sealed corridor". Next number read
from the file (last entry MSG-D28; no D29 existed), not assumed. Slots:

- Round-2: verdict line, tally, phase-A cleanliness lines (the machine that
  CAN prove the Node-free claim), the `vcruntime140.dll` True/False (settles
  PHASE4 Round 1's Sandbox copy-on-write caveat), Defender first-run delay,
  **SmartScreen/MOTW behaviour — explicitly marked as the input that feeds
  the deferred DP2 §9 Authenticode decision**, evidence archive location.
- Ceremony: roles (no real names — the file is public), invite channel
  (never the code), the both-ways proof (the only acceptable "it worked"),
  LAN-prompt path, red lines in full, photographs index, rollback record.
- Ratification line — the entry is NOT a decision until the field fills the
  slots and the owner signs.

## 3. Gate FP-3

- Staging folder complete and self-explaining ✅ (hash-verified against the
  gated artifacts; not copies-of-copies).
- Ledger draft committed ✅ (this commit).
- No tracked-path pollution ✅ — `git status` shows only the ledger edit and
  mission docs; zip/PDF/screenshot artifacts all live outside the repo;
  `dist-bare/` confirmed gitignored at FP-1.
- The A2.1 rollback zip (`AsymmFlow-Corridor-MachineB.zip`, 103.4 MB,
  2026-07-19) confirmed present at `C:\Projects\asymmflow\`, untouched ✅.

## 4. Not verified, stated plainly

- Nothing in the staging folder has traveled a USB stick or a browser
  download yet — MOTW/SmartScreen behaviour of THIS zip on real field
  hardware is exactly the observation MSG-D29 collects.
- The README's six steps have not been walked by anyone but their author;
  the owner is the first real reader.

**Gate FP-3: PASS.**

*Pack the proven, prove the pack, hand it over.* 🐻📦
