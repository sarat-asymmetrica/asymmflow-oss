# The Sealed Corridor — Field Runbook

**Status:** COMPLETE — the ceremony script (§4) is written from a ceremony that
was actually driven end to end, not from the design. Two sealed kits formed one
room through the real launchers and messages crossed both ways.
**Campaign:** The Sealed Corridor · **Deliverable:** SC-4
**Audience:** the two humans who actually run the ceremony, and the person on
the phone guiding them.

> **If you are the receptionist or the field contact:** you only need §3, §4
> and §7. Everything else is for the person supporting you.

---

## 1. What this is, in one paragraph

Two computers — one in India, one in Bahrain — each get the same folder. Each
person double-clicks one file in it. One person starts a conversation and reads
a code to the other; the other pastes that code in and reads a shorter code
back. After that, messages typed on either computer appear on the other. There
is no account, no server, no installer, and nothing is installed on either
machine.

## 2. Before ceremony day — the go / no-go decision

**This decision is made BEFORE anyone travels or schedules a call, not during
the ceremony.**

| condition | action |
|---|---|
| The sealed corridor kit is gate-green (SC-4 + SC-5 both passed) | run the ceremony on the sealed kit |
| The sealed corridor kit is NOT gate-green by ceremony day | **run the ceremony on the Node A2.1 corridor kit instead**, which stays packaged and ready. The sealed campaign continues afterwards with no deadline pressure. |

**This rollback is binding, and it is not a failure.** The campaign charter
says it plainly: *a slipped gate is a report, not a failure; a fudged gate is a
failure.* Nobody is authorised to relax a gate in order to keep a ceremony
date. If you are reading this on ceremony morning and the gate is not green,
the A2.1 zip is the answer and that is a completely acceptable outcome.

The A2.1 kit is the proven rollback path: it is the Node line, it is closed to
further investment, but it is warm and its spikes are green.

## 3. Preparing the two folders

*(SC-3 fills the exact per-step copy here once the ceremony flow lands. What is
already fixed is below.)*

### 3a. Getting the folder onto each machine — read this before you copy

**Use a USB stick if you possibly can.** Windows attaches a "mark of the web"
(MOTW) to files that arrive over the internet, and SmartScreen then challenges
anything it does not recognise. In practice:

- **USB / local network copy:** usually silent. No prompt.
- **Downloaded from a browser, or extracted from a downloaded .zip:** Windows
  is likely to show *"Windows protected your PC"* the first time the launcher
  runs. This is expected for a small self-contained tool from an unknown
  publisher. The path through it is **More info → Run anyway**.
- **Emailed .zip:** same as downloaded, and some mail clients strip or quarantine
  executables entirely. Avoid.

> **Honesty note:** this kit is **not** code-signed. Authenticode signing was
> deliberately deferred. A SmartScreen prompt is therefore the *expected*
> behaviour on a downloaded copy, not a sign that anything is wrong. Do not
> tell a client "you will not see any warnings" — tell them which warning they
> may see and exactly which button to press.

If the person on the other end is not comfortable clicking through a security
prompt, get the folder to them on physical media instead. That is a support
decision, not a technical one.

### 3b. Where you put the folder matters — the `#` rule

**The folder path must not contain a `#` character anywhere.**

This is a real defect in the Bare runtime's addon resolution (a `#` is parsed
as a URL fragment and truncates the path), found at the Sealed Ship's merge
gate on 2026-07-20. A kit at `C:\Users\a#b\corridor\` will fail in a confusing
way that looks like a broken download.

Practical guidance: put the folder somewhere short and boring —
`C:\corridor\` or the Desktop. Avoid paths with `#`, and be aware that a
Windows username containing `#` will poison a Desktop path.

The kit's own verifier (`verify_clean_machine.cmd`) checks this and warns
loudly before it wastes anyone's time. Run it first (§7).

### 3c. Synthetic data only

Every name and message used in the ceremony must be **made up**. No real
customer, no real document, no real business information, ever — including in
the "test message" both sides send. This is a standing campaign invariant, not
ceremony-day caution.

## 4. The ceremony

**Two people, one phone call.** Decide first who is **STARTING** and who is
**JOINING**. It does not matter which machine is which — but it must be
decided before you begin, not negotiated halfway through.

Both people: double-click **`run_bare_mesh.cmd`**, then type `2` and press
Enter for *Open the messenger*. You will each see a one-time Windows-permission
notice — press Enter, or type `skip` and press Enter. Either is fine.

You will then both see the same question. **This is where the two paths split.**

### 4a. The person STARTING

1. Type `connect` and press Enter.
2. At *"Did someone send you a code?"* — **just press Enter** (you are the one
   starting).
3. The screen prints a **long code**. **Send it** — WhatsApp, email, whatever
   is easiest. Reading it aloud is the fallback, not the plan; if you must, it
   is printed in groups of four.
4. The screen then waits and says it is listening. **Leave this window open.**
5. The other person will read you back a **short code**. **Type or paste it in
   and press Enter.** The screen says *"the other computer can finish joining
   now"*.
6. You are now in the messenger. Type a message and press Enter. Type `/read`
   to see the conversation.

### 4b. The person JOINING

1. Type `connect` and press Enter.
2. **Paste the long code** they sent you and press Enter. (Extra spaces or
   line breaks from a chat app are fine — they are stripped automatically.)
3. Your screen prints a **short code**. **Read it back to them** — this one is
   short enough to say out loud, in groups of four.
4. You will be asked for an address. **If you are in different offices or
   different countries, just press Enter.** Only type something here if you are
   both on the same office network *and* someone has told you an address like
   `192.168.1.5:4300`.
5. The screen says it is waiting for the other computer to let you in. This can
   take a little while. When it succeeds it says *"joined — you can post now"*.
6. Type a message and press Enter. Type `/read` to see the conversation.

### 4c. Proving it actually worked

Do **both** of these, in this order. One direction is not a corridor.

1. The JOINER types a message. The STARTER types `/read` and reads it aloud.
2. The STARTER types a message. The JOINER types `/read` and reads it aloud.

> **Use `/read`, not `/rooms`.** `/rooms` shows only a one-line summary of the
> *last* message in the room's canonical order, which is **not** necessarily
> the newest one you just received — if you both typed at about the same time,
> a perfectly delivered message can be missing from that line. `/read` shows
> the actual conversation. This is not a nicety; a gate run misdiagnosed a
> working corridor as broken for exactly this reason.

When you are done, type `/exit` then `5` to close.

## 5. What to photograph

Enough that a failure can be diagnosed later without booking a second call:

- **Both screens** at the moment of the verdict — the joiner's *"joined — you
  can post now"* and the starter's *"the other computer can finish joining
  now"*.
- **Both `/read` outputs** from §4c, showing each other's messages. This is the
  actual proof; everything else is supporting evidence.
- **Any red or unexpected line, in full**, including the detail below the fold
  line. The guide deliberately prints one plain sentence and then the raw
  detail underneath — photograph both halves, not just the sentence.
- If anything failed: the whole **`VERIFY_EVIDENCE.txt` + `verify-logs\`**
  folder from §7.

**Do not photograph the long invite code.** It is a bearer secret — anyone who
has it can read the room. Send it once, to the person who needs it, and do not
put it in a group chat or a support ticket.

## 6. What "it worked" actually means

Do not accept any of these as proof:

- **A window that closed without an error.** The runtime exits 0 on real
  failure modes, including total loss of its own output. Exit codes lie at
  three independent layers in this stack.
- **"It says Goodbye."** A kit that renders its entire ceremony and silently
  cannot post also says Goodbye.

The **only** acceptable proof is content, both directions:

1. A message typed on machine A is **visible on machine B**, and
2. a message typed on machine B is **visible on machine A**.

Both, in the same session, with the actual text matching. Anything less is not
a corridor.

## 7. If something goes wrong

Run `verify_clean_machine.cmd` in the kit folder. It checks the machine, warns
about the `#` path hazard, and runs the ceremony 16 times, writing everything
to `VERIFY_EVIDENCE.txt` and `verify-logs\`. **Send that whole folder back** —
it is the artifact that makes remote diagnosis possible.

For the connection specifically, menu **[1] Check the connection** reports
whether this computer can reach the internet meeting point.

> **Read this before you act on a red result.** A single connection check can
> come back red while the corridor is in fact perfectly usable. Measured on the
> development machine on 2026-07-20: **1 of 7** two-process attempts failed
> after the connection had already been established, with a verified negative
> control proving the measurement was real. **One failed check is not proof
> that anything is broken.** Try it again before escalating. The guide's own
> copy says so, deliberately.

The real end-to-end test is always the messenger itself (§6), not the
diagnostic.

## 8. What this runbook does NOT cover

Stated plainly rather than left to be discovered on ceremony day:

- **Anything that changes this computer.** The always-on anchor and the
  automatic firewall rule are honest stubs in this kit — they say so on
  screen. If Windows asks for permission during connection, click Yes; nothing
  else needs doing by hand.
- **Two machines behind the same restrictive corporate firewall with no
  internet path.** The kit carries a direct local-network fallback for that
  case, but arranging it needs someone who can read an IP address off the other
  machine.
- **Recovery of a conversation from a deleted folder.** The folder *is* the
  data. Deleting it deletes the conversation.

---

*Port the proven, seal the port, prove the seal.* 🐻
