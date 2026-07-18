// verify-transcript.mjs — the offline, trust-nothing counterpart to
// export-transcript.mjs. No Autobase, no network, no storage: just the
// bundle + crypto + the wasm fold. Nothing the bundle CLAIMS is taken on
// faith — every verdict below is recomputed from the ops it carries.
//
// Two independent checks, both required for VERIFIED:
//   1. Per-op signature: recompute the versioned signable (v1/v2/v3 BY KIND,
//      via capability.mjs's own signable() — the same function the reducer's
//      Go mirror agrees with) and verify the Ed25519 signature against the
//      op's OWN devicePub. A forged op can self-sign consistently (attacker
//      controls both payload and key) — that alone can't be caught here.
//   2. Whole-transcript refold: re-fold the verbatim ops through the REAL
//      wasm room reducer (mode 'room', same authorityPub the room used) and
//      compare the recomputed state digest against the bundle's claimed
//      digest. This is what catches a forged-key op: a device with no grant
//      in the room's capability table is rejected by the kernel, so the
//      refolded digest diverges from what the bundle claims — even though
//      the forged op's own signature checked out cleanly.
//
// See MSG-D19 (mesh/docs/MESSENGER_DECISIONS.md) for the transcript law and
// the honest limitations of what a VERIFIED verdict does and does not prove.

import hcrypto from 'hypercore-crypto'
import { createHash } from 'node:crypto'
import { signable } from './capability.mjs'
import { applyViaWasm } from './apply.mjs'

function verifyOpSignature(op) {
  if (!op || typeof op !== 'object') return { sigValid: false, reason: 'op is not an object' }
  if (typeof op.devicePub !== 'string' || !op.devicePub) return { sigValid: false, reason: 'missing devicePub' }
  if (typeof op.sig !== 'string' || !op.sig) return { sigValid: false, reason: 'missing sig' }
  try {
    const digest = createHash('sha256').update(signable(op)).digest()
    const sig = Buffer.from(op.sig, 'hex')
    const pub = Buffer.from(op.devicePub, 'hex')
    if (sig.length !== 64 || pub.length !== 32) return { sigValid: false, reason: 'malformed sig/devicePub length' }
    const ok = hcrypto.verify(digest, sig, pub)
    return ok ? { sigValid: true, reason: null } : { sigValid: false, reason: 'signature does not match devicePub' }
  } catch (err) {
    return { sigValid: false, reason: `verify threw: ${err.message}` }
  }
}

/**
 * verifyTranscript(bundle) -> verdict object. Pure function: never mutates
 * `bundle` or anything reachable from it (tamper/drop scenarios in the spike
 * operate on deep copies precisely so this holds).
 */
export function verifyTranscript(bundle) {
  const formatOk = bundle && bundle.format === 'asymm-transcript.v1'
  const ops = Array.isArray(bundle?.ops) ? bundle.ops : []

  const opVerdicts = ops.map((op) => {
    const { sigValid, reason } = verifyOpSignature(op)
    return { seq: op?.seq, actor: op?.actor, kind: op?.kind, sigValid, reason }
  })
  const allSigsValid = opVerdicts.length > 0 && opVerdicts.every((v) => v.sigValid)

  // The real kernel fold, not a re-derivation of it: this is a room transcript
  // by construction (export-transcript.mjs only ever exports rooms), so the
  // reducer mode is always 'room'; authorityPub travels IN the bundle because
  // an offline verifier has no other way to know it.
  const config = bundle?.authorityPub ? { authorityPub: bundle.authorityPub } : undefined
  let recomputedDigest = null
  let refoldError = null
  try {
    const state = applyViaWasm(ops, config, 'room')
    recomputedDigest = state?.digest ?? null
  } catch (err) {
    refoldError = err?.message ?? String(err)
  }

  const expectedDigest = bundle?.stateDigest ?? null
  const digestMatches = recomputedDigest !== null && recomputedDigest === expectedDigest

  const verified = formatOk && allSigsValid && digestMatches && !refoldError

  return {
    verified,
    formatOk,
    allSigsValid,
    digestMatches,
    expectedDigest,
    recomputedDigest,
    refoldError,
    ops: opVerdicts,
  }
}
