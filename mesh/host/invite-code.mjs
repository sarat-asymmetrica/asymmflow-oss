// invite-code.mjs — Mission M2: the shareable room-invite code.
//
// "Click a code, you're in the PO room." One pasteable string carries the four
// facts a joiner needs: WHERE (the room's Autobase key), WHOSE LAW (the room
// authority's public key), WHICH OFFER (the inviteId), and THE SECRET (the
// invite keypair's seed — possession of which the redemption op proves, bound
// to the joining device, per mesh/reducer/capability.go verifyInviteProof).
//
// Key material rides hypercore-id-encoding (z-base-32, the ecosystem's
// standard for pasteable keys, with checksums); the envelope (prefix + dot
// separators + trailing inviteId) is ours. The TRANSPORT rendezvous (hs://
// url) is deliberately NOT in the code: transport-auth ≠ capability-auth —
// a code mints capability, the tunnel is arranged like any other peer.
//
// SECURITY NOTE: the code contains the invite SECRET. Sharing the code IS
// sharing the invite — treat it like the WhatsApp group link it replaces,
// except this one expires, is one-time by default, and is revocable (owner
// invite law, 2026-07-18).

import HypercoreID from 'hypercore-id-encoding'

const PREFIX = 'asymm-room1'

/** encodeInviteCode({baseKey, authorityPub, inviteSeed, inviteId}) -> string */
export function encodeInviteCode({ baseKey, authorityPub, inviteSeed, inviteId }) {
  const part = (buf) => HypercoreID.encode(Buffer.isBuffer(buf) ? buf : Buffer.from(buf, 'hex'))
  if (!/^[^.]+:\d+$/.test(inviteId)) {
    throw new Error(`inviteId must be {actor}:{seq}, got ${JSON.stringify(inviteId)}`)
  }
  return [PREFIX, part(baseKey), part(authorityPub), part(inviteSeed), inviteId].join('.')
}

/** decodeInviteCode(code) -> {baseKey, authorityPub, inviteSeed, inviteId} (hex/Buffer) */
export function decodeInviteCode(code) {
  const trimmed = String(code).trim()
  const segs = trimmed.split('.')
  if (segs.length !== 5 || segs[0] !== PREFIX) {
    throw new Error('not an AsymmFlow room invite code (expected 5 dot-separated parts, prefix asymm-room1)')
  }
  const [, baseKey, authorityPub, inviteSeed, inviteId] = segs
  if (!/^[^.]+:\d+$/.test(inviteId)) {
    throw new Error(`invite code carries a malformed inviteId ${JSON.stringify(inviteId)}`)
  }
  return {
    baseKey: HypercoreID.decode(baseKey).toString('hex'),
    authorityPub: HypercoreID.decode(authorityPub).toString('hex'),
    inviteSeed: HypercoreID.decode(inviteSeed), // Buffer — feeds inviteKeys()
    inviteId,
  }
}
