// invite-code.mjs — Mission M2: the shareable room-invite code.
//
// "Click a code, you're in the PO room." One pasteable string carries the four
// facts a joiner needs: WHERE (the room's Autobase key), WHOSE LAW (the room
// authority's public key), WHICH OFFER (the inviteId), and THE SECRET (the
// invite keypair's seed — possession of which the redemption op proves, bound
// to the joining device, per mesh/reducer/capability.go verifyInviteProof).
// (M4 stage 2, room2): a FIFTH fact for encrypted rooms — THE CONTENT KEY
// (the room's Autobase `encryptionKey`, mesh-node.mjs) — rides the SAME code.
//
// Key material rides hypercore-id-encoding (z-base-32, the ecosystem's
// standard for pasteable keys, with checksums); the envelope (prefix + dot
// separators + trailing inviteId) is ours. The TRANSPORT rendezvous (hs://
// url) is deliberately NOT in the code: transport-auth ≠ capability-auth —
// a code mints capability, the tunnel is arranged like any other peer.
//
// SECURITY NOTE: the code contains the invite SECRET and, for room2, the
// room's ENCRYPTION KEY — everything needed to read the room lives in this
// one string. Sharing the code IS sharing the room. It is a bearer secret;
// this module deliberately has NO opinion on how it travels (email, chat,
// QR, hand-typed) — that transport choice is the human's, per MSG-D11's
// transport-≠-capability stance, unchanged by encryption riding along.
// Treat it like the WhatsApp group link it replaces, except this one
// expires, is one-time by default, and is revocable (owner invite law,
// 2026-07-18).

import HypercoreID from 'hypercore-id-encoding'

const PREFIX_V1 = 'asymm-room1'
const PREFIX_V2 = 'asymm-room2'

/**
 * encodeInviteCode({baseKey, authorityPub, inviteSeed, inviteId, encryptionKey?}) -> string
 * Emits room2 (6 dot-separated parts, encryptionKey as the 6th) when an
 * encryptionKey is supplied; room1 (5 parts, unchanged bytes) otherwise —
 * the existing invite-spike scenario (unencrypted rooms) stays green
 * unmodified, byte-for-byte, on this path.
 */
export function encodeInviteCode({ baseKey, authorityPub, inviteSeed, inviteId, encryptionKey }) {
  const part = (buf) => HypercoreID.encode(Buffer.isBuffer(buf) ? buf : Buffer.from(buf, 'hex'))
  if (!/^[^.]+:\d+$/.test(inviteId)) {
    throw new Error(`inviteId must be {actor}:{seq}, got ${JSON.stringify(inviteId)}`)
  }
  const parts = [PREFIX_V1, part(baseKey), part(authorityPub), part(inviteSeed), inviteId]
  if (encryptionKey) {
    parts[0] = PREFIX_V2
    parts.push(part(encryptionKey))
  }
  return parts.join('.')
}

/**
 * decodeInviteCode(code) -> {baseKey, authorityPub, inviteSeed, inviteId, encryptionKey}
 * (hex/Buffer; encryptionKey is a Buffer for room2, null for room1.)
 * Accepts both room1 (5 parts) and room2 (6 parts) codes.
 */
export function decodeInviteCode(code) {
  const trimmed = String(code).trim()
  const segs = trimmed.split('.')
  const isV2 = segs[0] === PREFIX_V2
  const isV1 = segs[0] === PREFIX_V1
  const wantLen = isV2 ? 6 : 5
  if (!(isV1 || isV2) || segs.length !== wantLen) {
    throw new Error(
      'not an AsymmFlow room invite code (expected 5 dot-separated parts prefixed asymm-room1, ' +
      'or 6 parts prefixed asymm-room2)'
    )
  }
  const [, baseKey, authorityPub, inviteSeed, inviteId, encryptionKeyPart] = segs
  if (!/^[^.]+:\d+$/.test(inviteId)) {
    throw new Error(`invite code carries a malformed inviteId ${JSON.stringify(inviteId)}`)
  }
  return {
    baseKey: HypercoreID.decode(baseKey).toString('hex'),
    authorityPub: HypercoreID.decode(authorityPub).toString('hex'),
    inviteSeed: HypercoreID.decode(inviteSeed), // Buffer — feeds inviteKeys()
    inviteId,
    encryptionKey: isV2 ? HypercoreID.decode(encryptionKeyPart) : null, // Buffer — feeds createMeshNode()
  }
}
