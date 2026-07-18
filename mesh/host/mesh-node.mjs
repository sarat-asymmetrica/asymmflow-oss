// mesh-node.mjs — a Sovereign Mesh peer: Corestore + Autobase whose view is the
// LINEARIZED OP LOG, with state materialized by the Go/WASM kernel reducer.
//
// Design (Wave 1):
//   - apply() is deliberately minimal: it only handles writer grants and appends
//     op values to the view core. It never touches external state (Autobase may
//     undo/reapply the view during reordering — see autobase README "IMPORTANT").
//   - State is materialized OUTSIDE apply by folding the whole linearized view
//     through the wasip1 reducer (mesh/dist/reducer.wasm). The reducer re-sorts
//     ops into the canonical (Seq, Actor, SKU, TS) total order internally, so
//     the materialized state is insensitive to interim linearization churn —
//     while the CONVERGED view itself must still be byte-identical across peers
//     (that is the stronger property wave1 gates on).
//   - Convergence checks: viewDigest() hashes the raw linearized view entries;
//     state().digest is the reducer's canonical state digest (matches Wave 0's
//     golden for the same op set, by construction).

import Corestore from 'corestore'
import Autobase from 'autobase'
import { createHash } from 'node:crypto'
import { applyViaWasm } from './apply.mjs'

/**
 * An op value appended by writers. Common envelope: { seq, actor, ts, kind? }.
 * kind "" / "inventory.move" additionally needs { sku, delta } (Wave 0/1 shape);
 * other kinds (ar.*, approval.decide, policy.*) carry their own fields — the
 * REDUCER is the validator of record for those (kernel law), the host only
 * screens the envelope.
 */
function isOp(v) {
  if (!v || typeof v !== 'object' ||
      !Number.isInteger(v.seq) || typeof v.actor !== 'string' || !Number.isInteger(v.ts)) {
    return false
  }
  if (v.kind === undefined || v.kind === '' || v.kind === 'inventory.move') {
    return typeof v.sku === 'string' && Number.isInteger(v.delta)
  }
  return typeof v.kind === 'string'
}

/**
 * createMeshNode({ storage, bootstrap, primaryKey, authorityPub }) -> MeshNode
 *   storage      — corestore storage dir (or RAM factory)
 *   bootstrap    — the autobase key (hex string or Buffer) to join; null to found a new base
 *   primaryKey   — optional 32-byte Buffer for deterministic core keys (goldenable runs)
 *   authorityPub — optional hex Ed25519 mesh-authority key: turns ON Mission D
 *                  capability enforcement in the reducer (signed ops + grants).
 *                  Mesh-genesis data, distributed like the bootstrap key.
 *   mode         — optional reducer fold: '' (default) = business; 'room' =
 *                  Messenger room fold (Wave 1). A room is its OWN Autobase.
 *   wakeup       — optional shared protomux-wakeup instance (Mission M4):
 *                  pass the SAME instance to BlindPeering so mirror sockets
 *                  carry the autobase's wakeup/announce protocol.
 */
export async function createMeshNode({ storage, bootstrap = null, primaryKey, authorityPub, mode = '', wakeup } = {}) {
  // unsafe:true only acknowledges the PINNED primaryKey (test determinism);
  // production nodes omit primaryKey and get random identities.
  const store = new Corestore(storage, primaryKey ? { primaryKey, unsafe: true } : {})
  const boot = typeof bootstrap === 'string' ? Buffer.from(bootstrap, 'hex') : bootstrap

  const base = new Autobase(store, boot, {
    valueEncoding: 'json',
    ...(wakeup ? { wakeup } : {}),
    ackInterval: 100, // eager acks help the linearizer merge causal forks
    open(viewStore) {
      return viewStore.get('inventory-ops', { valueEncoding: 'json' })
    },
    async apply(nodes, view, host) {
      for (const node of nodes) {
        const value = node.value
        if (value && typeof value.addWriter === 'string') {
          // Guard the decode: a malformed key in the log must be IGNORED, not
          // thrown — an unguarded throw here is a poison-pill op that crashes
          // every peer on every refold, forever (found live: a writer key
          // pasted with '<>' wrapping crashed the host until its store was
          // deleted). apply must never crash on hostile/typo'd values.
          const key = Buffer.from(value.addWriter, 'hex')
          if (key.length === 32) await host.addWriter(key, { indexer: true })
          continue
        }
        if (!isOp(value)) continue // unknown/malformed values are ignored, never crash apply
        await view.append(value)
      }
    },
  })

  await base.ready()

  const node = {
    store,
    base,
    get key() { return base.key.toString('hex') },
    get writerKey() { return base.local.key.toString('hex') },
    get writable() { return base.writable },

    /** Grant another peer write access (their writerKey, hex). Validates
     * BEFORE appending — a malformed key must never enter the shared log. */
    async addWriter(writerKeyHex) {
      if (!/^[0-9a-fA-F]{64}$/.test(writerKeyHex)) {
        throw new Error(`writer key must be 64 hex chars (got ${JSON.stringify(writerKeyHex)}) — paste it bare, no <> or quotes`)
      }
      await base.append({ addWriter: writerKeyHex.toLowerCase() })
    },

    /** Append one inventory op. */
    async append(op) {
      if (!isOp(op)) throw new Error(`not a valid op: ${JSON.stringify(op)}`)
      await base.append(op)
    },

    /** The linearized op log, as plain values, in view order. */
    async ops() {
      await base.update()
      const view = base.view
      const out = []
      for (let i = 0; i < view.length; i++) out.push(await view.get(i))
      return out
    },

    /** sha256 over the raw linearized view — peers agree iff these match. */
    async viewDigest() {
      const entries = await node.ops()
      return createHash('sha256').update(JSON.stringify(entries)).digest('hex')
    },

    /** Materialized state via the Go/WASM kernel reducer (capability
     * enforcement on when the node was created with authorityPub; room
     * nodes fold through the Messenger room law instead). */
    async state() {
      return applyViaWasm(await node.ops(), authorityPub ? { authorityPub } : undefined, mode)
    },

    /** One replication wire to another in-process node. Returns an unreplicate(). */
    connect(other) {
      const s1 = store.replicate(true)
      const s2 = other.store.replicate(false)
      s1.pipe(s2).pipe(s1)
      return () => { s1.destroy(); s2.destroy() }
    },

    async close() {
      await base.close()
      await store.close()
    },
  }

  return node
}

/** Poll until fn() is truthy or timeout. Returns the last fn() result. */
export async function waitFor(fn, { timeout = 15000, interval = 100, label = 'condition' } = {}) {
  const deadline = Date.now() + timeout
  let last
  while (Date.now() < deadline) {
    last = await fn()
    if (last) return last
    await new Promise((r) => setTimeout(r, interval))
  }
  throw new Error(`timed out waiting for ${label}`)
}
