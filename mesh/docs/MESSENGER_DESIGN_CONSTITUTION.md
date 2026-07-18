# MESSENGER DESIGN CONSTITUTION

**Status:** RATIFIED by owner 2026-07-18 (design session, four-agent research pass)
**Scope:** governs all messenger waves from M5 onward, including the kernel-screen UI wave.
**Sits on:** `FABLE_CAMPAIGN_MESSENGER.md` (the build ladder) and `MESSENGER_DECISIONS.md`
(MSG-D1..D15, the built physics). Where this document and a future convenience conflict,
this document wins until the owner amends it.

Every article below is either (a) evidence-backed by the 2026-07-18 research pass
(four independent research reports: SMB behavior, feature usage, chat-as-ops,
async & safety — summarized in the appendix), or (b) an explicit owner values ruling.
Articles marked ⚖️ are values rulings; articles marked 📊 carry research citations.

---

## Article I — What this is (and is not)

**This is not "Slack inside AsymmFlow." Every business object can hold a conversation,
and every conversation can graduate into business fact.**

- 📊 The graveyard is unambiguous: bolt-on social layers next to the tools of work die
  ("non-contextual social software" — Chatter/Jam: 57% of companies saw only 10-20%
  activation). Chat embedded *inside* the transaction gets used (DingTalk/WeCom OA,
  WhatsApp B2B commerce).
- The room list is not a sidebar of channels. It is the RFQ pipeline, the open POs,
  the shipments in transit — each carrying its own thread. General chat is the
  *degenerate case* (a room anchored to nothing), not the design center.
- 📊 The per-deal/per-shipment group chat is already the native organizational unit of
  trading SMBs (emergent, unstructured, on WhatsApp). We formalize an existing habit;
  we do not ask for a behavior change.
- **We do not compete with WhatsApp as the external send/receive layer.** WhatsApp is
  the atmosphere (97% of surveyed India MSMEs; 90%+ GCC penetration). We win on what
  it structurally cannot do: org-owned history, anchored context, capability law,
  graduation into an audited system of record.

## Article II — The room taxonomy

Two constitutionally distinct room classes. The distinction is *topological*, not a
settings flag.

**Anchored rooms** (work): authority = the org (`AuthorityPub` = org authority key).
Born with `anchorType`/`anchorId` in the signed manifest (PO, RFQ, shipment, offer,
customer…). Mirrored for delivery. Append-only history is owed to the organization —
it survives staff turnover. Read cursors exist as *operational fact* (did the warehouse
see the delivery note). Claim/assign ops exist (Article VI).

**Social rooms & DMs** (human): authority = the participants themselves. The org
authority key is *not in the room* — there is no admin who can be granted in later,
because grants come from the room's own authority plane. Privacy is topology, not
policy. Never emitted: read cursors, presence, typing state. Disposable: every
participant discarding the base and forgetting the key is true deletion of the *room*
(each participant's own copy remains their own until they choose — see Article V).

- 📊 The owner's #1 pain in the wild — "the employee leaves and the chat history walks
  out the door on a personal phone" — is solved by the anchored class. The DingTalk
  resentment — being watched in the human layer — is solved by the social class.
- ⚖️ Owner ruling: work-related record is owed to the organization; the human layer
  belongs to its owners. Both, structurally, at once.
- ⚖️ **Amendment (owner-approved 2026-07-18):** a room's *identity* may span a
  SEQUENCE of Autobases. Because a content encryption key cannot rotate mid-base
  (MSG-D18, source-verified), a revocation-driven re-key mints a successor Autobase
  (new bootstrap key + new encryption key) whose `room.manifest` carries a
  `predecessorRoomKey` pointer — the room is a chain of crypto-epoch containers,
  its history discoverable across the chain, its live container always the newest.
  "One room, one Autobase" holds *within* an epoch, not across a re-key.

## Article III — The Correspondence Model (async law)

**Every signal in social space is volunteered, never harvested.** Messages are letters,
not pages. The silence is not information.

1. **No typing indicators, no online dots, no delivery ticks, anywhere.** 📊 Read-receipt
   / presence signals are a documented anxiety source (~31% report texting as daily
   stress; the common coping behavior is turning the signals off). We don't ship the
   thing people turn off.
2. **The voluntary wave.** The *recipient* may send a lightweight ack ("read, thinking,
   will reply properly") — built on `msg.react`. The signal is given, never taken.
3. **Sender-side expectation tags:** `whenever 🍃 / today 🌤 / urgent 🔥` on `msg.post`,
   default `whenever`. 📊 This is the single most evidence-backed feature in the entire
   design: Cornell/LBS (peer-reviewed, N=4,004) — receivers over-estimate required
   response speed by ~36%, the gap (not real urgency) predicts stress, and *senders
   stating explicit expectations* is the tested fix. No shipped product does this
   systematically. White space, claimed.
4. **The kettle's on 🫖:** presence exists only as an opt-in broadcast ("at my desk,
   interruptible") — a gift of availability. Absence of the kettle means nothing.
5. **Schedule-based quiet hours, timezone-aware, on by default.** 📊 Schedule-based DND
   beats remember-to-toggle; right-to-disconnect laws (France, Australia 2024-25) are
   making after-hours-message-as-obligation legally contested. We're ahead of it by
   default, not by configuration.
6. **Read cursors:** anchored rooms only, as operational fact. In social rooms the op
   is *never appended* — unread tracking is local state. Nobody can build a whip from
   bytes that were never written.
7. ⚖️ Informality and non-urgency are different axes. We keep SMB warmth (voice notes,
   reactions, the vent) and remove obligation (no harvested signals, calm defaults).
   We do not import Basecamp-style formality to get calm. (Research flag: "SMBs want
   async discipline" is an untested hypothesis — we validate it live, on our own pilot.)

## Article IV — The Prohibition List (banned by name) ⚖️📊

The DingTalk backlash is mechanism-specific, and Lark won users by positioning as the
anti-DingTalk — the humane variant *sells*. The following are banned by name, forever,
in both room classes:

1. **Escalating acknowledgment chases** (DingTalk's "DING": push → SMS → phone call
   until acknowledged). No message may escalate transport to force a human response.
2. **Location/attendance harvesting** (GPS/WiFi clock-in, geofenced presence).
3. **Forced read-receipts** and any boss-visible "has read your report" signal.
4. **Silent unsend** ("delete for everyone" without a tombstone) — see Article V.
5. **Admin export or covert membership in social rooms/DMs** — impossible by topology
   (Article II), prohibited as a product feature besides.
6. **AI summaries presented as fact, and AI-authored messages with ambiguous
   authorship.** 📊 Users punish authoritative-but-wrong hard (Apple Intelligence
   fabricated-headline failure). AI output in the messenger is always labeled, always
   draft, never auto-sent — matching the kernel's AI-authority boundary: agents draft,
   humans sign.
7. **Notification-heavy defaults.** 📊 Notification fatigue is complaint #1 across every
   workplace tool studied (~78% overwhelmed). Conservative defaults + granular per-room
   muting ship in v1, not as an apology later.

## Article V — Safety law (the humane floor) ⚖️📊

Under-reporting is the dominant harassment failure mode — the design optimizes for
*silent self-protection* first, reporting second.

1. **Consent-first contact:** DMs open by invitation only (M2 law). Declining is silent.
2. **Silent blocking:** block = discard your grant to their room + auto-refuse their
   invites. The blocked party experiences only silence — no signal to rage at.
   (Instagram's message-requests precedent: consent gating without rejection
   notification is a shipped, mainstream pattern.)
3. **No unsend, by physics:** each participant holds their own full copy of the log;
   a sender can never reach into another device and erase their words. Deletion is a
   tombstone (MSG-D5) — 📊 users trust tombstoned deletion *more* than traceless unsend.
4. **Non-repudiation, stated honestly:** every message is Ed25519-signed by the sending
   device. A harassment transcript is cryptographic evidence, not a doctorable
   screenshot. This is the opposite of Signal's deniability — a deliberate workplace
   values choice (deniability mostly protects the person who said the awful thing),
   and the UI says it plainly: *"messages here are signed — your words are provably
   yours."* The seatbelt sentence.
5. **Self-serve evidence export:** one gesture exports a signed transcript from the
   target's own copy — no IT/admin mediation that could tip off an abuser with account
   access. Hers alone.
6. **Room disposability never destroys another's evidence:** "delete the room" is each
   owner discarding their *own* copy. Mutual forgetting is possible; forced forgetting
   is not.

## Article VI — Ownership of the thread (anchored rooms) 📊

Shared-inbox literature is unanimous: multi-party threads decay without an explicit
"who owns this right now." Anchored rooms get a first-class **claim/assign** op
(`room.claim {assignee}` — authority- or self-assigned, reassignable, visible in the
manifest projection). Social rooms never have it — ownership is a work concept.

## Article VII — Threading law 📊

Slack-style retrofit branch-threading is the most-disliked model in the research and is
**never built**. Our shape is already the liked pattern:

- **Topic-first at the room level** (the anchor IS the topic — Zulip's insight without
  Zulip's friction).
- **Inside a room: flat timeline + `replyTo` quoting** (WhatsApp-style, already built).
- If a tangent deserves its own life, it gets its own room (possibly anchored to the
  same object) — never an in-room fork.

## Article VIII — The graduation seam is the trust pattern 📊

What works at scale (WhatsApp B2B in Brazil/India) is *chat as negotiation +
documentation surface, regulated system as settlement rail* — quote and invoice in the
thread, an explicit visible step-out for the money-moving act. That is exactly our
seam: `msg.draft-op` carries the business op as inert cargo; graduation is a separate
human-signed op on the business base, rendered in UI as a deliberate ceremony (the
"step out"), never a swipe. Trust in-chat action is highest when chat is the front end
to an audited system of record — which is literally our architecture.

## Article IX — External parties: the compliance doorway ⚖️

**Owner ruling 2026-07-18: Option B.** External counterparties (forwarders, customs
agents, principals, customers) join via invite codes that open a **lightweight web
client** — no install, no seat fee, capability-scoped (observer or writer per the
invite's grant). The full room stays sovereign; the web client is a window, not a copy.

Positioning (honesty-forward, per the Asymmetrica sovereignty brief): WhatsApp-for-
business became the norm through convenience, but the regulatory direction is against
it — multi-billion-dollar off-channel-comms fines in banking, EU data-protection
authorities moving officials off WhatsApp, EU sovereignty momentum (France's Microsoft
exodus, €180M sovereign cloud awards), DPDP/GDPR. The pitch is not "stop using
WhatsApp"; it is *"business-related communication happens in a compliant, org-owned,
auditable room — and it's one click for your counterparty, easier than a WhatsApp
group."* The convenience argument and the compliance argument point at the same door.

Build notes (for the wave that implements this):
- The invite code (M2, `asymm-room1.…`) is already capability-complete; the web client
  is a rendering + transport shim, not new law.
- Observer role (read-only in full, M2/MSG-D12) is the default external grant;
  writer grants are deliberate.
- The web client never holds the business base — anchored-room view + post only.
  Graduation remains inside the sovereign app, behind org identity.

## Article X — Feature canon 📊

**Build (evidence-backed):** voice notes (~7B/day on WhatsApp, over-indexed in our
geographies; already an `audio/webm` attachment — M3), reactions (high-frequency,
low-effort ack), edit with visible "edited" marker (table stakes; 15-min-window
precedent), tombstoned delete, attachments with end-to-end sha256 (M3), per-deal
rooms (M1), invites (M2), offline delivery via mirror (M4), expectation tags,
voluntary wave, kettle's-on, claim/assign, quiet hours, self-serve evidence export.

**Skip (evidence-backed):** stories/status (content-broadcast feature, not a messaging
primitive — thinly watched even where heavily built), polls (no engagement evidence at
small-team scale), disappearing messages (niche; and in tension with Article V evidence
preservation — TechSafety.org documents auto-delete destroying survivors' records).

**Avoid (see Article IV):** ambient AI summaries, AI auto-replies, aggressive
notification defaults, any presence/read harvesting.

## Article XI — DMs and the mirror (delegated ruling)

Owner delegated this call at spec-laydown (2026-07-18): **social rooms and DMs may
transit the org mirror ONLY encrypted** (Autobase encryptionKey — the M4 stage-2
doctrine). Until the encryption doctrine lands, **DMs are peer-to-peer only** — worse
delivery, zero exposure. A plaintext DM on org hardware is DingTalk with extra steps;
we never ship that intermediate state. Anchored rooms continue to mirror per M4
(plaintext stage-1 acceptable there: the org owns that record anyway; encryption still
lands for them at stage 2 as defense-in-depth). Metadata honesty: even encrypted,
the mirror sees who-talks-to-whom-when for rooms it carries; the owner may veto
DM-mirroring entirely at the stage-2 gate.

**RULED 2026-07-18 (post stage-2, owner):** DM-mirroring is ON — social rooms and
DMs transit the org mirror as ciphertext ("it feels more natural"). The encryption
prerequisite landed same day (Autobase encryptionKey end-to-end, keyless-probe
proven, midnight-vent spike). The metadata boundary was presented plainly and
accepted: the mirror learns which room keys sync when, never a byte of content.
The P2P-only interim clause above is hereby retired; it remains in the text as
the record of why the intermediate state was never shipped.

## Article XII — Amendment

Only the owner amends this constitution. Engineering may propose; articles change by
explicit ruling, recorded here with date.

**Amendment log:**
- 2026-07-18 — Art. II: room identity = a sequence of crypto-epoch Autobase
  containers linked by `predecessorRoomKey` (rotate-on-revoke = room re-issue;
  proposed from MSG-D18's rotation findings, owner-approved same day).
- 2026-07-18 — Art. XI: DM-mirroring ruled ON (owner, post stage-2) — DMs/social
  rooms ride the org mirror encrypted; metadata boundary accepted; P2P-only
  interim clause retired.

Stop-and-asks currently open at the owner's
desk: mailbox-vs-pure-mirror, ops packaging for the
office-machine mirror, kernel-screen UI wave (joint design IN PROGRESS
2026-07-18), M5+ ladder (mobile/push/calls).

---

## Appendix — Evidence base (2026-07-18 research pass)

Four independent research reports (Sonnet agents, web research, confidence-graded):

1. **SMB communication behavior** — WhatsApp dominance (97% India MSME survey,
   N=10,000+; 90%+ GCC penetration; 50M+ business accounts), why Slack/Teams lose SMBs
   (cost, onboarding, external-party gravity, the guest-seat pricing cliff), per-deal
   group chats as the native unit, the employee-exit history-loss pain. Caveat: neutral
   non-vendor data is thin; percentages beyond the India survey are directional.
2. **Feature usage reality** — voice notes ~7B/day; reactions high-frequency;
   stories/status thinly watched; read-receipt anxiety (~31%/~35%, coping = turn off);
   edit/tombstone-delete stuck, AI-summaries-as-fact failed (Apple Intelligence);
   Slack-threads most disliked, topic-first preferred; notification fatigue = complaint
   #1 (~78%); WhatsApp forward-cap study (arXiv 1909.08740) the one peer-reviewed gem.
3. **Chat-as-ops lessons** — DingTalk/WeCom free-OA bundling as adoption driver; the
   surveillance backlash is mechanism-specific (DING, GPS, forced reads — independent
   journalism, high confidence); Lark = the anti-DingTalk positioning; Chatter/Jam
   died of non-contextuality (57% cos at 10-20% activation); WhatsApp-as-ERP =
   negotiation surface + settlement rail; shared-inbox ownership discipline is a hard
   prerequisite. Gaps flagged: WhatsApp-as-ERP failure modes, Lark SMB traction,
   primary in-chat-payment trust research.
4. **Async & safety** — Cornell/LBS urgency-bias study (the Article III §3 spine);
   right-to-disconnect laws; pure-async-as-discipline (even Doist walked back full
   async); harassment via workplace messaging (12% report; under-reporting dominant);
   silent blocking, consent-first contact, evidence-vs-disappearing tension
   (TechSafety.org), deniability-vs-non-repudiation as a values fork.

Plus: Asymmetrica sovereignty evidence brief (`asymmetrica-capabilities.html`,
2026-06-03, 50 citations) — the cloud-exit/regulatory-momentum case that Article IX's
positioning stands on.

Design-session rulings incorporated: room taxonomy + DM topology (2026-07-18),
Correspondence Model (2026-07-18), safety stance incl. non-repudiation (2026-07-18),
Option B external doorway (2026-07-18), M2+M3+M4 scope & invite law & reactor gating
(2026-07-18, pre-build).
