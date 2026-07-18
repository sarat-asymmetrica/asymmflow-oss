// reissue-room.mjs — the room re-issue ceremony (Constitution Art. II
// amendment 2026-07-18, MSG-D18 GATE RULING, MSG-D20): rotation-on-revoke is
// NOT a rotate-in-place API (none exists in the installed autobase — MSG-D18
// verified that against the source before this was ever proposed). Rotation
// IS minting a SUCCESSOR Autobase: a fresh bootstrap key, a fresh content
// `encryptionKey`, and a `room.manifest` that carries the predecessor's OWN
// base key as `predecessorRoomKey` — the fold RECORDS that pointer, it does
// not follow it (mesh/reducer/room_domain.go's applyManifest just copies the
// signed field; navigating the chain is entirely a host concern, done here
// and in reissue-spike.mjs step 5).
//
// The predecessor is NEVER touched by this ceremony: no tombstone op, no
// close, no write of any kind. It remains a fully readable historical
// container for whoever already holds its key — append-only history is owed
// to the room's own past members (Constitution Art. V's honesty class), and
// the successor's manifest pointer is the only linkage the design needs; a
// second "this room is superseded" op on the OLD base would just be another
// mutation of history nobody asked for.
//
// What this ceremony does NOT give you: forward secrecy against a device
// that already held the predecessor's encryptionKey. A revoked device that
// kept K_old can still decrypt every message the predecessor ever carried —
// that was true before re-issue and stays true after (MSG-D18's stated
// boundary, option (3), still the honest interim truth). What re-issue DOES
// buy: the revoked device holds no grant in the NEW room and — because the
// new room is encrypted under a KEY IT NEVER HAD — cannot even make sense of
// new ciphertext crossing the mirror or a peer, let alone forge a state
// change that survives a refold. reissue-spike.mjs step 4 is the crypto
// proof of exactly that boundary, both directions.

import { createMeshNode } from './mesh-node.mjs'
import { signOp, grantOp } from './capability.mjs'
import { encodeInviteCode } from './invite-code.mjs'
import { createHash } from 'node:crypto'

/**
 * reissueRoom({ predecessorNode, storage, primaryKey, authorityKeys,
 *   newEncryptionKey, surviving, ts, actor }) -> { node, inviteCodesByDevice }
 *
 * predecessorNode  — the OLD room's MeshNode (mode:'room'). Read ONLY: its
 *                     current state supplies the title/anchorType/anchorId
 *                     to copy forward, and its own base key (`.key`) becomes
 *                     the successor's `predecessorRoomKey`. Never appended
 *                     to, never closed by this function.
 * storage/primaryKey — the SUCCESSOR's Corestore storage (+ optional pinned
 *                     primaryKey for goldenable runs) — a brand-new Autobase
 *                     is always founded here (`bootstrap` is never accepted;
 *                     re-issue always MINTS, it never joins an existing base).
 * authorityKeys    — the room authority's device keypair (capability.mjs
 *                     `deviceKeys` shape). Signs the successor's manifest
 *                     and every re-grant — same authority identity as the
 *                     predecessor's; re-issue changes the CONTAINER, not who
 *                     holds the room's authority.
 * newEncryptionKey — 32-byte Buffer: the successor Autobase's own
 *                     `encryptionKey` (mesh-node.mjs). MUST differ from the
 *                     predecessor's key for the ceremony to mean anything —
 *                     this function does not enforce that (it is not the
 *                     fold's job to police key hygiene), the caller must.
 * surviving        — array of device public-key HEX STRINGS to re-grant at
 *                     epoch 0 of the new room. Anyone not listed here simply
 *                     never appears in the successor's grant table — there
 *                     is no revocation op to write, absence IS the
 *                     revocation (MSG-D20).
 * ts               — base event-data timestamp for the ceremony's own ops
 *                     (never a live clock, same discipline as every spike
 *                     script's hardcoded ts values). Each op gets its own
 *                     small offset off this base so the ceremony's internal
 *                     ordering stays legible in a log dump. Default 0.
 * actor            — the authority's actor string on the successor room
 *                     (default 'hub', the campaign's canon vocabulary).
 *
 * Returns the successor `node` plus one FRESH `asymm-room2.` invite code per
 * surviving device, keyed by that device's pubHex. HONEST FRAMING (the
 * report's load-bearing sentence): the capability grant is ALREADY DIRECT —
 * this function appends a real `cap.grant` op for every survivor before it
 * ever builds a code. The invite plane's `inviteSeed`/`inviteId` fields
 * riding inside each `asymm-room2.` code are therefore NOT a redeemable
 * offer (there is no matching `invite.offer` anywhere in this room); nobody
 * calls `invite.redeem` against them. They exist purely because
 * `encodeInviteCode` co-locates a shareable capability envelope with a
 * content key, and reusing that ONE proven format is simpler and safer than
 * inventing a second bearer-secret string shape. The code's job here is
 * narrower than in the M2 invite flow: it is the KEY-TRANSPORT envelope —
 * "here is the new room and the new content key" — for a device whose
 * capability already lives in the op log. (Same honest-envelope framing
 * MSG-D18's mirror-spike.mjs used for its own synthetic INVITE_SEED.)
 */
export async function reissueRoom({
  predecessorNode, storage, primaryKey, authorityKeys, newEncryptionKey,
  surviving, ts = 0, actor = 'hub',
}) {
  const predecessorState = await predecessorNode.state()
  const manifest = predecessorState.manifest ?? {}

  const node = await createMeshNode({
    storage, primaryKey,
    authorityPub: authorityKeys.pubHex, mode: 'room',
    encryptionKey: newEncryptionKey,
  })

  let seq = 1
  const manifestOp = signOp({
    seq: seq++, actor, ts,
    kind: 'room.manifest',
    title: manifest.title ?? '',
    anchorType: manifest.anchorType ?? '',
    anchorId: manifest.anchorId ?? '',
    observersAllowed: manifest.observersAllowed ?? false,
    predecessorRoomKey: predecessorNode.key,
  }, authorityKeys)
  await node.append(manifestOp)

  const inviteCodesByDevice = {}
  for (const devicePubHex of surviving) {
    const grant = grantOp({ seq: seq++, actor, ts: ts + 1, device: devicePubHex, epoch: 0 }, authorityKeys)
    await node.append(grant)

    // Deterministic per-device envelope seed — NOT key-generation randomness
    // (determinism landmine #2): derived from the device's own public key so
    // a re-run of the ceremony over the same inputs is goldenable. It has no
    // cryptographic role beyond filling the invite-code envelope's seed slot
    // (see the honest-framing note above — the real capability is the
    // cap.grant already appended).
    const seed = createHash('sha256').update(`reissue-seed:${devicePubHex}`).digest()
    inviteCodesByDevice[devicePubHex] = encodeInviteCode({
      baseKey: node.key,
      authorityPub: authorityKeys.pubHex,
      inviteSeed: seed,
      inviteId: `${actor}:${grant.seq}`,
      encryptionKey: newEncryptionKey,
    })
  }

  return { node, inviteCodesByDevice }
}
