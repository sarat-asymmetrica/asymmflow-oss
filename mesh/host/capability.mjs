// capability.mjs — Mission D host-side: Ed25519 device keys + op signing.
//
// The REDUCER is the enforcer (mesh/reducer/capability.go); this module only
// produces what the reducer verifies: device keypairs (libsodium Ed25519 via
// hypercore-crypto — RFC 8032, interoperable with Go crypto/ed25519) and op
// signatures over sha256(signable(op)).
//
// signable() MUST stay byte-identical to the Go mirror. Netstrings, not JSON:
// JS and Go disagree on JSON key order and escaping; length-prefixed fields in
// fixed order have exactly one encoding. The missiond gate is the standing
// cross-runtime proof (JS signs → Go/WASM verifies).

import hcrypto from 'hypercore-crypto'
import { createHash } from 'node:crypto'

/** Fixed field order — MIRROR of mesh/reducer/capability.go signableV1(). */
const FIELDS = [
  (op) => String(op.seq ?? 0),
  (op) => op.actor ?? '',
  (op) => String(op.ts ?? 0),
  (op) => op.kind ?? '',
  (op) => op.sku ?? '',
  (op) => String(op.delta ?? 0),
  (op) => op.customer ?? '',
  (op) => String(op.amountMinor ?? 0),
  (op) => String(op.limitMinor ?? 0),
  (op) => op.currency ?? '',
  (op) => op.subject ?? '',
  (op) => op.subjectType ?? '',
  (op) => op.decision ?? '',
  (op) => op.reason ?? '',
  (op) => op.correlationId ?? '',
  (op) => op.actorType ?? '',
  (op) => String(op.authority ?? 0),
  (op) => op.policyId ?? '',
  (op) => op.device ?? '',
  (op) => op.role ?? '',
  (op) => String(op.epoch ?? 0),
  (op) => op.devicePub ?? '',
]

/**
 * Room fields appended for the "meshop.v2" payload — MIRROR of signableV2().
 * Selected by kind (room.manifest / msg.*), so every legacy kind keeps its
 * exact v1 bytes and Mission A-D signatures/goldens stay untouched (MSG-D2).
 */
const FIELDS_V2 = [
  ...FIELDS,
  (op) => op.msgId ?? '',
  (op) => op.body ?? '',
  (op) => op.replyTo ?? '',
  (op) => op.emoji ?? '',
  (op) => (op.on ? 'true' : 'false'),
  (op) => op.upToActor ?? '',
  (op) => String(op.upToSeq ?? 0),
  (op) => op.title ?? '',
  (op) => op.anchorType ?? '',
  (op) => op.anchorId ?? '',
  (op) => (op.observersAllowed ? 'true' : 'false'),
  (op) => op.draft ?? '',
  (op) => op.attachment ?? '',
  // expectation tags + claim/assign (MSG-D16, appended at the end)
  (op) => op.expectation ?? '',
  (op) => op.assignee ?? '',
  // predecessor room pointer (MSG-D20, appended at the end — third growth;
  // room.manifest ONLY, but the signable includes it unconditionally like
  // every other room field)
  (op) => op.predecessorRoomKey ?? '',
]

/**
 * Invite fields appended for the "meshop.v3" payload — MIRROR of signableV3().
 * Selected ONLY by invite.* kinds (MSG-D11): v1 AND v2 bytes stay untouched.
 * predecessorRoomKey rides along inside FIELDS_V2's spread, in the same
 * relative slot as v2 — right before the invite fields.
 */
const FIELDS_V3 = [
  ...FIELDS_V2,
  (op) => op.inviteId ?? '',
  (op) => op.invitePub ?? '',
  (op) => op.inviteProof ?? '',
  (op) => String(op.expiresAt ?? 0),
  (op) => String(op.maxUses ?? 0),
]

export function isRoomKind(kind) {
  // Exact mirror of Go: kind == "room.manifest" || kind == "room.claim" ||
  // (len > 4 && prefix "msg.").
  return kind === 'room.manifest' || kind === 'room.claim' ||
    (typeof kind === 'string' && kind.length > 4 && kind.startsWith('msg.'))
}

export function isInviteKind(kind) {
  // Exact mirror of Go: len > 7 && prefix "invite.".
  return typeof kind === 'string' && kind.length > 7 && kind.startsWith('invite.')
}

/** The canonical signed payload: version prefix + netstring per field. */
export function signable(op) {
  const [prefix, fields] = isInviteKind(op.kind)
    ? ['meshop.v3', FIELDS_V3]
    : isRoomKind(op.kind) ? ['meshop.v2', FIELDS_V2] : ['meshop.v1', FIELDS]
  const parts = [Buffer.from(prefix)]
  for (const get of fields) {
    const f = Buffer.from(get(op), 'utf8')
    parts.push(Buffer.from(`${f.length}:`), f, Buffer.from(','))
  }
  return Buffer.concat(parts)
}

/**
 * deviceKeys(seed?) -> { publicKey, secretKey, pubHex }
 * seed: optional 32-byte Buffer for deterministic test identities;
 * production omits it and gets a random keypair.
 */
export function deviceKeys(seed) {
  const kp = seed ? hcrypto.keyPair(seed) : hcrypto.keyPair()
  return { ...kp, pubHex: kp.publicKey.toString('hex') }
}

/**
 * signOp(op, keys) -> a NEW op object carrying devicePub + sig.
 * The signature covers sha256(signable(op-with-devicePub)) — devicePub is
 * inside the signed payload, so a signature can't be replayed under another key.
 */
export function signOp(op, keys) {
  const withPub = { ...op, devicePub: keys.pubHex }
  const digest = createHash('sha256').update(signable(withPub)).digest()
  const sig = hcrypto.sign(digest, keys.secretKey)
  return { ...withPub, sig: sig.toString('hex') }
}

/** A capability grant, authored + signed by the mesh AUTHORITY. */
export function grantOp({ seq, actor, ts, device, role = 'writer', epoch = 0 }, authorityKeys) {
  return signOp({ seq, actor, ts, kind: 'cap.grant', device, role, epoch }, authorityKeys)
}

/** An epoch bump (revocation wave): grants not re-issued at `epoch` go stale. */
export function epochOp({ seq, actor, ts, epoch }, authorityKeys) {
  return signOp({ seq, actor, ts, kind: 'cap.epoch', epoch }, authorityKeys)
}

/** A targeted single-device revocation. */
export function revokeOp({ seq, actor, ts, device }, authorityKeys) {
  return signOp({ seq, actor, ts, kind: 'cap.revoke', device }, authorityKeys)
}

// ── Mission M2: invites ──────────────────────────────────────────────────────

/** An invite keypair. The SEED travels inside the shareable code; the pub is
 * what the offer op pins. Same derivation as device keys (RFC 8032). */
export function inviteKeys(seed) {
  return deviceKeys(seed)
}

/** The proof payload the INVITE key signs at redemption — binds possession of
 * the invite secret to the JOINING device. MIRROR of Go inviteProofPayload. */
export function inviteProofPayload(devicePubHex) {
  return Buffer.concat([Buffer.from('meshinvite.v1:'), Buffer.from(devicePubHex, 'utf8')])
}

export function inviteProof(inviteSecretKeys, devicePubHex) {
  const digest = createHash('sha256').update(inviteProofPayload(devicePubHex)).digest()
  return hcrypto.sign(digest, inviteSecretKeys.secretKey).toString('hex')
}

/** invite.offer — authority-signed grant offer. Owner defaults (2026-07-18):
 * ONE-TIME (maxUses 1) and 72h expiry, both set HERE at creation; the fold
 * enforces whatever the offer says (0 expiresAt = never, explicit opt-in). */
export const INVITE_DEFAULT_TTL_MS = 72 * 60 * 60 * 1000

export function inviteOfferOp({ seq, actor, ts, invitePub, role = 'writer', expiresAt, maxUses = 1 }, authorityKeys) {
  return signOp({
    seq, actor, ts, kind: 'invite.offer', invitePub, role,
    expiresAt: expiresAt ?? ts + INVITE_DEFAULT_TTL_MS, maxUses,
  }, authorityKeys)
}

/** invite.redeem — signed by the JOINING device, carrying the invite proof. */
export function inviteRedeemOp({ seq, actor, ts, inviteId }, inviteSecretKeys, deviceKeysOfJoiner) {
  return signOp({
    seq, actor, ts, kind: 'invite.redeem', inviteId,
    inviteProof: inviteProof(inviteSecretKeys, deviceKeysOfJoiner.pubHex),
  }, deviceKeysOfJoiner)
}

/** invite.revoke — authority tombstones the offer. */
export function inviteRevokeOp({ seq, actor, ts, inviteId }, authorityKeys) {
  return signOp({ seq, actor, ts, kind: 'invite.revoke', inviteId }, authorityKeys)
}
