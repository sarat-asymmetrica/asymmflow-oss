# Design Under Fire

**A doctrine for building business software that survives reality.**
AsymmFlow ecosystem · first ratified 2026-07-19 · a standing document — the roadmap answers to it.

---

## Why this document exists

In 2026, a war in the Gulf closed airspace over four countries, shut the Strait of
Hormuz, struck a satellite communications facility, hit a water desalination plant
twice, and disrupted cloud datacenters that hosted, among other things, banking
infrastructure. Businesses across the region discovered — in the span of weeks — that
"resilient" systems were often one datacenter with extra steps, and that the core
assumption of modern SaaS ("the network is up, mostly; the cloud is elsewhere and
elsewhere is safe") can simply stop being true.

But here is the thing the war only made *legible*, not new: **most of the world
already lives under some version of these constraints, permanently.** A trading
company in a conflict zone, a clinic supplier in rural Ethiopia, a distributor in
Ghana on metered mobile data, a manufacturer in rurban India with a four-hour daily
power window — they are all underserved by the same assumption stack. Traditional
SaaS is designed for places where electricity, bandwidth, and political stability
are ambient. That is a minority of the earth.

So this is not a document about war. It is a document about **deep care for
reality** — using the hardest observed conditions as the design floor, so that the
software serves everyone above that floor for free. A system that works under fire
works during a monsoon outage, a fiber cut, a currency crisis, a cyclone, or a
plain old expired credit card on a cloud account.

We did not arrive at these principles in the abstract. We watched what broke.

## The constraint set (the reality requirements document)

Derived from observed failures — wartime and peacetime alike:

- **C1. Any central facility can be lost** — to a strike, a flood, a sanction, a
  bankruptcy, a misconfiguration. Datacenters, DNS providers, cloud accounts,
  head offices: all of them.
- **C2. Connectivity is intermittent by default.** Partitions last minutes, weeks,
  or indefinitely. Bandwidth may be metered, slow, or politically filtered.
- **C3. The layers below the stack are in scope.** Power, water, cooling, fuel.
  Software that requires a datacenter's utilities inherits that datacenter's wars.
- **C4. Failures do not respect their intended blast radius.** Design for the
  *unintended* loss, not just the targeted one. You cannot model precision —
  yours or anyone else's — so minimize what any single loss costs.
- **C5. People move, suddenly.** Evacuation, migration, seasonal work. A person's
  ability to operate must not be bound to a place, a specific device's survival,
  or reachability of a home server.
- **C6. Devices are lost, seized, and looted.** What a machine holds, someone
  else may one day hold.
- **C7. Institutional dependencies are fragile too.** The email provider for the
  password reset, the SMS gateway for the 2FA code, the payment processor for the
  subscription — every third party is a failure mode with a logo.

## The principles (what the constraints force)

### P1. The unit of survival is the machine, not the datacenter
Every device holds a full, usable, *sovereign replica* of everything its owner
needs — not a cache with delusions. If the region goes dark for a month, one
laptop IS the business, whole and operational. Offline is not a degraded mode;
offline is Tuesday. *(Answers C1, C2.)*

### P2. Convergence over availability
Do not chase "always connected to the quorum" — assume long partitions and design
for **eventual truthful convergence**. This makes merge logic the crown jewel:
deterministic, auditable, total. When two halves of a company operate separately
for three weeks, "last write wins" is data loss with good manners; a deterministic
fold over a causally-ordered log is a *reunion*. *(Answers C2, C5.)*

### P3. No single point of anything — including trust
Not one server, one cable, one DNS name, one cloud account, one jurisdiction, one
person who knows the passwords. Always-on peers may *accelerate* convergence but
must never be *required* for it: the mesh has to work peer-to-peer with nothing
but two machines and an invite code read over a phone. Every "central" component
is permanently demoted to "convenient." *(Answers C1, C4, C7.)*

### P4. Transport promiscuity
The data layer must not care how bytes travel: DHT hole-punching today, a direct
TCP link tomorrow, a tunnel the day after — and, as the honest endpoint of the
principle, **sneakernet**: a USB stick carried across a border is a first-class
replication transport. Append-only signed logs make this free; verified data does
not care whether it arrived by fiber or by pocket. When the airspace closes, the
pocket still travels. *(Answers C2, C4.)*

### P5. Energy-proportional computing
Run on what a UPS, a car inverter, or a solar panel can feed: a compiled binary,
an embedded database, a small sidecar — on one laptop. Resilience correlates with
smallness. Every dependency you do not have is a dependency that cannot be
bombed, sanctioned, rate-limited, or price-hiked. *(Answers C3.)*

### P6. Encryption at rest is a humanitarian feature
A machine abandoned in an evacuation must hold nothing readable. Key material
belongs to the person and their recovery ceremony, not to the device. This is not
compliance; it is protecting people whose office may become a checkpoint.
*(Answers C6.)*

### P7. Identity travels with the human
Cryptographic identity (keys held by the person), recoverable through a **social
ceremony** — a quorum of people who know you vouching you back in — never through
"we emailed you a reset link" to a mail server in the burning region. A person
who walks out with nothing must be re-enterable into the system by their
community. Community is the root of trust. *(Answers C5, C6, C7.)*

### P8. Paper is the final replica
The vital records — who owes what, who owns what, who is owed — must export to a
form that works with zero electricity. Print-and-verify beats cloud-and-pray.
People have carried ledgers through worse than anything in this document.
*(Answers C1, C2, C3, all at once, at the cost of latency measured in humans.)*

## The fire test (how the roadmap answers to this document)

Every proposed feature, dependency, or architectural change faces five questions:

1. **Partition:** does it still work with zero connectivity for a month?
2. **Loss:** if the machine (or datacenter, or provider) hosting it vanishes
   tonight, what is permanently lost? (Acceptable answer: convenience. Nothing else.)
3. **Person:** can a user who lost every device and document be restored by the
   people who know them?
4. **Power:** does it run on one laptop on a UPS?
5. **Pocket:** can its data travel by USB stick and arrive verifiable?

A feature may fail a question *knowingly* — with the failure written down and
ruled on — but never silently. Convenience is allowed to depend on infrastructure;
**truth is not.**

## What this already looks like in practice

This doctrine was not written first and implemented later; it was extracted from
a working system and now governs its future. In the AsymmFlow ecosystem today:
sovereign local replicas with background convergence; a deterministic Go reducer
(compiled to WASM) as the single fold over a causally-ordered multi-writer log;
always-on peers explicitly demoted to "anchors" (accelerators, never
requirements); ciphertext-only mirrors for untrusted infrastructure; multiple
interchangeable transports including direct TCP for when the DHT is unreachable;
encrypted-at-rest stores with OS-keystore-held keys; invite and pairing
ceremonies designed to be performed over a phone call, in plain words, in groups
of four characters.

And the near roadmap this document already implies: a sneakernet replication
ceremony (P4), social-recovery key ceremonies (P7), and a paper-export pass for
vital records (P8). Small missions. Load-bearing ones.

## The closing argument

Designing for the world's hardest conditions is not designing for the margins —
it is designing for the *majority*, and for everyone else's worst week. The same
architecture that holds a family business together under air-raid sirens holds it
together through a monsoon, a grid collapse, a fiber cut, or a hyperscaler's
regional outage. Dignity and resilience turn out to have the same architecture:
your business should not need anyone's permission — or anyone's uptime — to know
its own truth.

Build for fire. Everyone above the fire line is served for free.
