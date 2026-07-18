// social-room.mjs — Constitution Art. II (social rooms & DMs), Art. XI, MSG-D21.
//
// "Privacy is topology, not policy." A social room is an ENFORCED room whose
// AuthorityPub is a PARTICIPANT'S OWN device key — the creator's — and whose
// signed manifest carries NO anchor (AnchorType ""). There is no separate
// admin identity: the org authority key simply never appears anywhere in a
// social room's op log, ever. This module reuses every mechanism Mission D
// (capability) and M2 (invites) already built — nothing new in the reducer,
// nothing new in the capability plane — the only thing that changes is WHOSE
// key sits in the AuthorityPub slot and WHAT the manifest points at (nothing).
//
// A DM is the degenerate case of a social room: two participants, opened via
// a REAL one-time `invite.offer` + `asymm-room2.` code + `invite.redeem` —
// the exact M2 ceremony, now carrying the room's content `encryptionKey`
// (MSG-D18) so the whole thing transits the org mirror ONLY as ciphertext
// (Art. XI). This is deliberately the SAME machinery reissue-room.mjs and
// invite-spike.mjs already prove; a DM room founder is just a participant
// instead of an org, and the manifest is unanchored instead of anchored.
//
// Read cursors are banned in social rooms TWICE, defense-in-depth (Art. III
// §6 / MSG-D21): the FOLD refuses to fold one (room_domain.go's applyRead,
// checked before this module ever runs), and this module's own convention is
// to never build a msg.read op for a social room in the first place. Both
// halves matter — a future UI bug that tries to emit one still can't make it
// into the log.
//
// Ownership/claim: room.claim is likewise refused by the fold in an
// unanchored room ("claims are a work concept", MSG-D17) — this module never
// builds one either. Nothing here re-implements that check; it is inherited
// for free from the same fold every anchored room uses.

import { createMeshNode } from './mesh-node.mjs'
import { signOp, grantOp, epochOp, inviteKeys, inviteOfferOp } from './capability.mjs'
import { encodeInviteCode } from './invite-code.mjs'

/**
 * createSocialRoom({ creatorKeys, storage, primaryKey, title, encryptionKey,
 *   ts, actor }) -> MeshNode
 *
 * creatorKeys    — the founding participant's device keypair (capability.mjs
 *                   deviceKeys shape). Becomes the room's AuthorityPub — the
 *                   room's law plane is THIS PERSON, not an org. There is no
 *                   code path anywhere that lets an org key be granted into
 *                   a social room later: AuthorityPub is fixed at
 *                   createMeshNode time and the manifest never carries one.
 * storage/primaryKey — the new Autobase's Corestore storage (+ optional
 *                   pinned primaryKey for goldenable runs), same shape as
 *                   every other createMeshNode caller in this codebase.
 * title          — human label for the room (required by applyManifest —
 *                   the fold skips a titleless manifest, same law as an
 *                   anchored room).
 * encryptionKey  — 32-byte Buffer (Mission M4 stage 2, MSG-D18): threaded
 *                   straight into createMeshNode. A social room without an
 *                   encryptionKey is legal (unenforced-crypto, still
 *                   enforced-capability) but Art. XI's mirror-transit law
 *                   only clears an ENCRYPTED social room — callers that
 *                   intend to mirror a DM must pass one.
 * ts/actor       — event-data timestamp (never a live clock) and the actor
 *                   string the manifest op signs as (e.g. 'ana'). Defaults
 *                   0 / 'hub' match reissue-room.mjs's own defaults, but a
 *                   social room's creator is a person, not the org hub — real
 *                   callers should pass their own actor name.
 *
 * Returns the ready node with ONE op already appended: the authority-signed,
 * UNANCHORED manifest (AnchorType/AnchorID simply never set — "" is the
 * fold's own social-room marker, Art. II/MSG-D17's existing vocabulary,
 * nothing new).
 */
export async function createSocialRoom({
  creatorKeys, storage, primaryKey, title, encryptionKey, ts = 0, actor = 'hub',
}) {
  const node = await createMeshNode({
    storage, primaryKey,
    authorityPub: creatorKeys.pubHex, mode: 'room',
    encryptionKey,
  })
  const manifestOp = signOp({
    seq: 1, actor, ts, kind: 'room.manifest', title,
    // anchorType/anchorId deliberately absent: "" is the fold's own
    // unanchored-room marker (applyClaim/applyRead both key off it).
  }, creatorKeys)
  await node.append(manifestOp)
  return node
}

/**
 * openDmInvite(node, { creatorKeys, inviteSeed, ts, seq, actor }) -> string
 *
 * Appends a REAL one-time `invite.offer` (writer role, one use, 72h TTL —
 * inviteOfferOp's own creation defaults, MSG-D11, untouched here) and
 * returns the full pasteable `asymm-room2.` code — the room's base key,
 * authority (the creator's device pub), the invite seed/id, AND the room's
 * own `encryptionKey` (read off the node via mesh-node.mjs's read-only
 * passthrough), all in one bearer string. Redemption is the real M2
 * ceremony (inviteRedeemOp + the fold's applyInvite) — this function only
 * builds the OFFER side; the joining device signs its own redeem op.
 *
 * creatorKeys — MUST be the room's own authority (the offer op's signature
 *   is checked against AuthorityPub by the fold — an offer from anyone else
 *   is simply rejected, same law as every other invite.offer).
 * inviteSeed  — 32-byte Buffer feeding inviteKeys() (capability.mjs); the
 *   SECRET half of the offer. Deterministic seeds make a scenario goldenable
 *   (canon rule); production callers should use real randomness.
 * seq         — this op's seq on the room's own writer-seq counter (the
 *   caller tracks the room's op count, same discipline every spike already
 *   follows — no hidden seq state lives in this module).
 * actor       — the offer-signing actor string; defaults 'hub' to match
 *   createSocialRoom's own default, but should match whatever actor the
 *   room's creator used there.
 */
export async function openDmInvite(node, { creatorKeys, inviteSeed, ts = 0, seq, actor = 'hub' }) {
  const invite = inviteKeys(inviteSeed)
  const offer = inviteOfferOp({ seq, actor, ts, invitePub: invite.pubHex }, creatorKeys)
  await node.append(offer)
  return encodeInviteCode({
    baseKey: node.key,
    authorityPub: creatorKeys.pubHex,
    inviteSeed,
    inviteId: `${actor}:${seq}`,
    encryptionKey: node.encryptionKey,
  })
}

/**
 * blockDevice(node, { authorityKeys, devicePub, ts, seq, actor }) -> void
 *
 * THE PARTICIPANT-AUTHORITY BLOCKING PRIMITIVE (Constitution Art. V §2:
 * "silent blocking... the blocked party experiences only silence"). Reuses
 * the exact capability-plane mechanism Mission D / reissue-room.mjs already
 * trust: an epoch bump followed by re-grants for every OTHER currently
 * current-epoch device, read directly off the room's own live state — the
 * caller never has to hand-maintain a survivor list. The room's authority
 * itself needs no re-grant (capability.go's capabilityGate: "the authority
 * is implicitly granted" — that line is what makes silent blocking cheap:
 * one epoch bump plus zero-or-more re-grants, never a self-grant).
 *
 * HONEST BOUNDARY, stated in code because it is easy to misread this
 * function as more powerful than it is:
 *
 *   1. This function is a bouncer for the room's OWN AUTHORITY ONLY. It
 *      works because `authorityKeys` signs the epoch bump and the fold
 *      requires `isAuthority` for `cap.epoch`/`cap.grant` (capability.go).
 *      Call it with any other device's keys and every op it builds is
 *      simply REJECTED by the fold on every peer — it does nothing.
 *
 *   2. A NON-authority participant (an ordinary DM member who isn't the
 *      room's founder) has NO fold-level block available — there is no op
 *      kind that lets a member revoke another member's grant, because
 *      membership law in a social room is exactly the same capability plane
 *      Mission D built for anchored rooms, and only the authority holds
 *      that pen. A non-authority participant's "block" is CLIENT-SIDE ONLY:
 *      discard your own local copy of the room's base/key (you stop
 *      replicating and stop reading), and refuse to redeem any future
 *      invite from that room. The fold cannot help a participant block
 *      someone in a room they do not govern — that is not a bug to route
 *      around here, it is the honest shape of "participant authority" (Art.
 *      II): SOMEONE has to hold the pen, and in a 2-person DM the founder is
 *      that someone. A future multi-party social room with rotating/shared
 *      authority is a different design, not built by this function.
 *
 *   3. NO op kind anywhere in this vocabulary signals "you were blocked" —
 *      not a rejection reason a blocked device could read (rejections are
 *      per-peer local computation, never broadcast), not a tombstone, not a
 *      new message kind. The blocked device's own subsequent ops simply
 *      start rejecting on every peer's refold, forever, with the ordinary
 *      "no grant for device" / "stale" capability wording — indistinguishable
 *      from "the room went quiet." Silence is structural (Art. V §2), not a
 *      missing feature.
 */
export async function blockDevice(node, { authorityKeys, devicePub, ts = 0, seq, actor = 'hub' }) {
  const state = await node.state()
  const currentEpoch = state.capEpoch ?? 0
  const survivors = Object.entries(state.grants ?? {})
    .filter(([pub, grant]) => grant.epoch === currentEpoch && pub !== devicePub)
    .map(([pub]) => pub)
    .sort() // deterministic re-grant order — no map-iteration-order dependence in the ceremony itself

  const newEpoch = currentEpoch + 1
  let s = seq
  await node.append(epochOp({ seq: s++, actor, ts, epoch: newEpoch }, authorityKeys))
  for (const survivor of survivors) {
    await node.append(grantOp({ seq: s++, actor, ts: ts + 1, device: survivor, epoch: newEpoch }, authorityKeys))
  }
}

// ── True deletion is physics, not an API (Art. V §6) ────────────────────────
//
// Deliberately NOT a function in this module. "Delete the room" is each
// participant independently running `rmSync(storage, { recursive: true,
// force: true })` on their OWN Corestore directory and forgetting the room's
// key (base key + encryptionKey) — nothing more, nothing less. A
// `deleteRoom()` wrapper here would misrepresent what actually happens: its
// name would suggest an operation that reaches the ROOM (deletes it for
// everyone, or for the other party), when what genuinely occurs is each
// owner discarding THEIR COPY. Forced forgetting must not exist (Art. V §6)
// — wrapping `rmSync` in a room-shaped name is exactly the kind of API that
// would eventually get called with that wrong expectation. social-spike.mjs
// demonstrates the real physics directly: both sides call `rmSync` on their
// own storage, and the mirror's copy (if the room was ever mirrored) outlives
// them as unreadable ciphertext once every key-holder has forgotten the key
// — nobody can act on the mirror's copy to make it un-outlive them, because
// the mirror never held a grant or a key of its own (MSG-D15).
