// export-transcript.mjs — Constitution Article V §5: self-serve evidence export.
//
// One gesture, from the target's OWN copy, no admin mediation: a plain
// JSON-able bundle carrying the room's linearized signed ops VERBATIM. The
// envelope (devicePub + sig, exactly as it came off node.ops()) IS the
// evidence — this module never re-signs, filters, or normalizes a single op.
// Tombstoned messages export as the tombstone (MSG-D5): the log stays honest,
// deleted content stays blanked, exactly as every other peer already sees it.
//
// See verify-transcript.mjs for the offline, trust-nothing counterpart, and
// MSG-D19 (mesh/docs/MESSENGER_DECISIONS.md) for the transcript law.

/**
 * exportTranscript(node, { exportedBy }) -> plain JSON-able bundle.
 * node: a room-mode MeshNode (mesh-node.mjs, mode: 'room').
 * exportedBy: an optional human-facing label for who ran the export (a CLAIM,
 *   not a proof — see the honest-limitation note in MSG-D19: this field is not
 *   signed by anyone, so it documents intent, it does not attest identity).
 */
export async function exportTranscript(node, { exportedBy } = {}) {
  const ops = await node.ops()
  const viewDigest = await node.viewDigest()
  const state = await node.state()

  // Key order pinned per the standing determinism standard (never a key-array
  // replacer) — a re-export of the same room state produces byte-identical JSON.
  return {
    format: 'asymm-transcript.v1',
    roomKey: node.key,
    authorityPub: node.authorityPub ?? null,
    exportedBy: exportedBy ?? null,
    ops,
    stateDigest: state.digest,
    viewDigest,
  }
}
