// attachments.mjs — Mission M3: content-addressed attachments over Hyperblobs.
//
// Campaign invariant 4: "a message references a blob by key+hash; the fold
// never inlines file bytes into the room log." The reducer already carries
// `attachment` as INERT opaque cargo (reserved at Wave 1, MSG-D8) — so the
// entire attachment pipeline lives HERE, in the host:
//
//   sender:   bytes → its own blob core (one per writer, in its corestore)
//             → an attachment REF (JSON string, sorted keys) into msg.post
//   receiver: ref → open the sender's blob core by key → stream the bytes
//             P2P (Hypercore Merkle proofs guard transport integrity)
//             → verify the ref's sha256 END-TO-END before handing bytes over
//
// The sha256 in the ref is the CONTENT address: transport integrity comes from
// Hypercore's Merkle tree, but the ref pins what the SENDER promised — a
// tampered or wrong blob fails verification no matter how it arrived.
//
// A voice note is just an attachment with an audio/* contentType (recon: no
// live-media stack needed — capture is UI-side MediaRecorder → webm bytes).

import Hyperblobs from 'hyperblobs'
import { createHash } from '#crypto'

const BLOB_CORE_NAME = 'room-blobs'

/** The writer's own blob store (lazy). One blob core per corestore identity. */
export async function openBlobStore(store) {
  const core = store.get({ name: BLOB_CORE_NAME })
  await core.ready()
  return { core, blobs: new Hyperblobs(core) }
}

/**
 * putAttachment(blobStore, {name, contentType, bytes}) -> ref STRING for
 * msg.post's `attachment` field. Deterministic: sorted keys, no timestamps —
 * the same bytes from the same core yield the same ref (goldenable).
 */
export async function putAttachment({ core, blobs }, { name, contentType, bytes }) {
  const id = await blobs.put(bytes)
  // Key order is fixed by construction (insertion order, already sorted);
  // NEVER pass a key array as the stringify replacer — it filters RECURSIVELY
  // and would strip the nested hyperblobs locator's fields (found the fun way).
  const ref = {
    blobKey: core.key.toString('hex'),
    byteLength: bytes.length,
    contentType,
    id: { // hyperblobs locator, fields pinned in stable order
      byteOffset: id.byteOffset,
      blockOffset: id.blockOffset,
      blockLength: id.blockLength,
      byteLength: id.byteLength,
    },
    name,
    sha256: createHash('sha256').update(bytes).digest('hex'),
  }
  return JSON.stringify(ref)
}

/**
 * getAttachment(store, refString) -> { bytes, ref } — streams the blob from
 * whichever peer replicates it, then verifies the ref's sha256 END-TO-END.
 * Throws on any mismatch: a wrong blob never reaches the caller.
 */
export async function getAttachment(store, refString, { timeout = 15000 } = {}) {
  const ref = JSON.parse(refString)
  const core = store.get(Buffer.from(ref.blobKey, 'hex'))
  await core.ready()
  const blobs = new Hyperblobs(core)
  const bytes = await blobs.get(ref.id, { timeout })
  if (!bytes || bytes.length !== ref.byteLength) {
    throw new Error(`attachment ${ref.name}: expected ${ref.byteLength} bytes, got ${bytes?.length ?? 0}`)
  }
  const sha = createHash('sha256').update(bytes).digest('hex')
  if (sha !== ref.sha256) {
    throw new Error(`attachment ${ref.name}: sha256 mismatch — content is NOT what the sender promised`)
  }
  return { bytes, ref }
}

/** verifyAttachmentBytes(refString, bytes) -> '' | reason. The pure check,
 * host-side twin of the fold's honesty: typed, never silent. */
export function verifyAttachmentBytes(refString, bytes) {
  const ref = JSON.parse(refString)
  if (bytes.length !== ref.byteLength) return `byteLength mismatch (${bytes.length} != ${ref.byteLength})`
  const sha = createHash('sha256').update(bytes).digest('hex')
  if (sha !== ref.sha256) return 'sha256 mismatch'
  return ''
}
