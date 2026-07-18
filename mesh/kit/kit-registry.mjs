// kit-registry.mjs — Mission U1.5: persists which rooms THIS device knows
// about across restarts (GL-5 reopen discipline: state must survive a real
// process restart, not just an in-memory reconnect). Deliberately tiny —
// one JSON array in `data/keys/rooms.json` (SIBLING of `data/corestore/`,
// same doctrine as the device identity itself): `{roomKey, storage
// (relative to data/corestore/), authorityPub, encryptionKey (hex|null),
// bootstrap (hex|null), title}` per room this device created or joined.
// kit-host.mjs reads it at boot and reopens every room automatically
// (createMeshNode against the SAME storage dir + same authorityPub/
// encryptionKey/bootstrap it was opened with originally — mesh-node.mjs's
// own reopen contract, proven by social-spike.mjs's "next morning" step:
// same storage dir + same keys IS the same device waking up, not a new
// identity). No corestore internals are read to recover a path (Corestore
// doesn't expose one reliably) — the kit chooses and remembers its own
// stable directory names instead.

// U1.6 addition: each entry may also carry `lastPeer` (`"ip:port"` string,
// absent until a network path has actually succeeded once) - the F2
// "auto-reconnect" fix. Written after any successful outbound `/connect`
// (the exact dialed address - always correct) and, best-effort, after an
// INBOUND connection is accepted while listening (the remote IP is real,
// but its LISTENING port is not observable from an accepted socket - the
// kit assumes its own default port, the same convention the README's
// ceremony instructs both machines to use; see kit-repl.mjs's
// ensureTcpListen). A stale or wrong lastPeer is harmless: kit-host.mjs's
// boot-time auto-reconnect attempt just fails quietly and prints the
// outcome, exactly like any other /connect that can't reach its target.

import { readFileSync, writeFileSync, existsSync } from 'node:fs'
import { join } from 'node:path'

function registryPath(keysDir) { return join(keysDir, 'rooms.json') }

export function loadRoomRegistry(keysDir) {
  const p = registryPath(keysDir)
  if (!existsSync(p)) return []
  try {
    const parsed = JSON.parse(readFileSync(p, 'utf8'))
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return [] // a corrupt registry must never crash boot — worst case, rooms don't auto-reopen
  }
}

/** Idempotent by roomKey — a room already on file is never duplicated or overwritten. */
export function saveRoomRegistryEntry(keysDir, entry) {
  const p = registryPath(keysDir)
  const list = loadRoomRegistry(keysDir)
  if (list.some((r) => r.roomKey === entry.roomKey)) return
  list.push(entry)
  writeFileSync(p, JSON.stringify(list, null, 2))
}

/** Record the last address a room's replication actually succeeded over.
 * No-op (not an error) if the room isn't registered yet. */
export function updateRoomRegistryPeer(keysDir, roomKey, peerAddr) {
  const p = registryPath(keysDir)
  const list = loadRoomRegistry(keysDir)
  const entry = list.find((r) => r.roomKey === roomKey)
  if (!entry) return
  entry.lastPeer = peerAddr
  writeFileSync(p, JSON.stringify(list, null, 2))
}
