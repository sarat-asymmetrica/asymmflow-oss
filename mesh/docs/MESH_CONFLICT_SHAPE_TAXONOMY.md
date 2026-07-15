# MESH — Conflict-Shape Taxonomy (Mission B)

> **The load-bearing distinction of the whole campaign.** CRDTs and Autobase are
> different tools; the mesh needs BOTH, sorted by *conflict-shape*. Mis-sorting an
> entity into the wrong bucket is the signature failure mode (FABLE_CAMPAIGN_SOVEREIGN_MESH.md §2).

- **CRDT — the commutative ~80%.** Always-accept, always-converges. There is *no
  invariant to break*, so a merge can never be "wrong", only late. Use for
  append-only and last-writer-wins-per-field data. Cost: cannot preserve a floor/ceiling.
- **Autobase-apply — the invariant-bound ~20%.** Deterministic linearized order +
  the pure Go kernel reducer (compiled to wasip1 — proven in the spike, see
  `MESH_PROGRESS.md`). CAN enforce an invariant, at the cost that a conflicting
  offline write may be **deterministically rejected** on merge. For money/stock,
  "reject the oversell" is the *correct* behaviour, surfaced as a typed
  `Unconfirmed`/`Rejected` state — never a silent bad number.

**Rule:** never CRDT the invariant-bound set; never pay Autobase's rejection cost
on the commutative set.

Status legend: ✅ classified & argued · 🔬 needs a determinism-audit pass in Mission C ·
❓ open question for the Commander.

---

## The table (draft — refine during Mission B with the domain owners)

| Entity | Bucket | CRDT type / invariant | Rationale | Status |
|---|---|---|---|---|
| **Inventory / stock level** | Autobase | invariant: `qty ≥ 0` (per SKU/location floor) | The spike domain. Two offline sales of the last unit must not converge to −1; the later-in-canonical-order write is rejected. | ✅ (spike proven) |
| **Credit limit / customer balance** | Autobase | invariant: `exposure ≤ limit` | A ceiling; concurrent orders that jointly breach the limit must have the breaching one rejected, identically on every peer. | ✅ |
| **Invoice number (per-division sequence)** | Autobase | invariant: uniqueness + gap-free per issuer | A shared sequence needs coordination. **But see the ZATCA note below** — per-device chains dissolve most of this. | 🔬 |
| **Payment application (receipt → invoices)** | Autobase | invariant: `Σapplied ≤ receipt AND ≤ invoice due` | Over-application is a money error; must reject the breaching apply deterministically. | ✅ |
| **GRN / goods receipt against PO** | Autobase | invariant: `Σreceived ≤ ordered` (+ over-receipt policy) | Receiving more than ordered is policy-gated, not free-merge. | 🔬 |
| **Approval state transition** | Autobase | invariant: only legal transitions (typed `Decision` machine) + `CanApprove()` | The kernel approval machine IS an apply reducer already. An AI actor's approve is rejected in the replication layer itself (`CanApprove()==false`). | ✅ |
| **Orders (header, append)** | CRDT | G-Set (append-only) | New orders never conflict; they accumulate. Line-level edits are LWW-Map fields. | ✅ |
| **Audit log / event trail** | CRDT | G-Set (append-only, immutable) | Pure accretion; ordering is by (writer, seq); never mutated or deleted. | ✅ |
| **Customer / supplier profile fields** | CRDT | LWW-Map (per field) | Name, phone, address, terms: last-writer-wins per field converges fine; no cross-field invariant. | ✅ |
| **Notes / comments / free text** | CRDT | G-Set or RGA (if collaborative text) | Additive; RGA only if two people edit the *same* note body concurrently. | ✅ |
| **Product / instrument catalog fields** | CRDT | LWW-Map (per field) | Descriptive metadata; no floor/ceiling. Price is LWW unless a pricing invariant is declared. | 🔬 |
| **Customer visits / spend counters (hospitality)** | CRDT | PN-Counter / G-Counter | Commutative accumulation; exact total not invariant-bound. | ✅ |
| **Opportunity / RFQ pipeline stage** | CRDT | LWW-Map (stage field) + G-Set (history) | Stage is advisory LWW; the history of stage changes is append-only. | ✅ |
| **Costing sheet (draft)** | CRDT | LWW-Map (per field) until "Save as Offer" | A draft has no invariant; the *conversion to an Offer* (a financial doc) crosses into Autobase. | 🔬 |
| **Delivery note / dispatch** | Autobase | invariant: `Σdispatched ≤ order qty` | Dispatching more than ordered is a fulfilment error. | 🔬 |
| **Serial / warranty record** | CRDT | G-Set (append) + LWW-Map (status field) | Serials accrete; their status (delivered/returned) is LWW or a small state machine. | 🔬 |

---

## Delightful alignment to exploit (Mission E)

**ZATCA does NOT require a global invoice sequence** — the ICV counter is *per
device / EGS unit*. A per-device append-only counter is exactly a **Hypercore**
(one keypair, one log, single-writer, `ICV = core.length`). So per-device invoice
chains need **no cross-node coordination for numbering** — the mandate never asked
for it. This moves a big chunk of the "invoice number" row out of the coordinated
Autobase bucket and into a trivially-correct per-device Hypercore. Two
independently-designed systems share one shape; use it.

## Open questions for the Commander (❓)

- **Per-division invoice numbering** beyond ZATCA's per-device ICV: does PH need a
  gap-free *per-division* human-facing sequence that survives multi-branch, or is
  per-device + division prefix acceptable? (Determines whether the "invoice number"
  row stays coordinated-Autobase or dissolves like ZATCA's.)
- **Over-receipt / over-dispatch policy**: are these hard rejections or
  policy-approved exceptions? (Determines Autobase-reject vs Autobase-with-approval.)
- **Collaborative text** (notes edited concurrently): is RGA worth the complexity,
  or is per-note LWW acceptable for the pilot?
