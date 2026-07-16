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

/** Fixed field order — MIRROR of mesh/reducer/capability.go signable(). */
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

/** The canonical signed payload: "meshop.v1" + netstring per field. */
export function signable(op) {
  const parts = [Buffer.from('meshop.v1')]
  for (const get of FIELDS) {
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
