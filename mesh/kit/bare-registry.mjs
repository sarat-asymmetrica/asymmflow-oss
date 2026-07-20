// bare-registry.mjs — SC-1 (Sealed Corridor campaign): a Bare-native port of
// mesh/kit/kit-registry.mjs. PRECEDENT: kit-registry.mjs — same JSON shape
// (one array in `<keysDir>/rooms.json`), same idempotent-by-roomKey
// semantics, same `updateRoomRegistryPeer` no-op-if-unregistered contract,
// same "a corrupt registry must never crash boot" law. This file changes
// exactly two things from that precedent, both forced by the Bare import
// discipline (bare-bridge.mjs's own header, binding here too):
//   1. `fs` comes from `#fs` (mesh/package.json's condition map: `bare-fs`
//      under Bare, `fs` under Node) instead of a bare `node:fs` import.
//   2. Path joining is hand-rolled (`joinPath` below, copied verbatim in
//      shape from bare-bridge.mjs's own `joinPath` — no `node:path`,
//      no `#path` alias for a single trivial join).
// Everything else — field names, control flow, comments explaining WHY —
// is the same design, re-typed for the runtime that needs it.

import fs from '#fs'

// Same hand-rolled join as bare-bridge.mjs's `joinPath` (that file's own
// header: "the only three operations this file needs... not a general-
// purpose `path` replacement"). Only the one operation this file needs.
function joinPath(...parts) { return parts.filter((p) => p !== '' && p != null).join('/').replace(/\/{2,}/g, '/') }

function registryPath(keysDir) { return joinPath(keysDir, 'rooms.json') }

/** Load-or-empty. A corrupt/missing registry must NEVER crash boot — worst
 * case, rooms don't auto-reopen (kit-registry.mjs's own standard, ported
 * verbatim). */
export function loadRoomRegistry(keysDir) {
  const p = registryPath(keysDir)
  if (!fs.existsSync(p)) return []
  try {
    const parsed = JSON.parse(fs.readFileSync(p, 'utf8'))
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

/** Idempotent by roomKey — a room already on file is never duplicated or
 * overwritten (kit-registry.mjs's own contract, unchanged). */
export function saveRoomRegistryEntry(keysDir, entry) {
  const p = registryPath(keysDir)
  const list = loadRoomRegistry(keysDir)
  if (list.some((r) => r.roomKey === entry.roomKey)) return
  list.push(entry)
  fs.writeFileSync(p, JSON.stringify(list, null, 2))
}

/** Record the last address a room's replication actually succeeded over.
 * No-op (not an error) if the room isn't registered yet — same contract as
 * kit-registry.mjs's own. Not wired into bare-guide.mjs this mission (SC-2's
 * network leg is a separate mission); exported for that later caller and to
 * keep this port's surface a faithful match of its precedent. */
export function updateRoomRegistryPeer(keysDir, roomKey, peerAddr) {
  const p = registryPath(keysDir)
  const list = loadRoomRegistry(keysDir)
  const entry = list.find((r) => r.roomKey === roomKey)
  if (!entry) return
  entry.lastPeer = peerAddr
  fs.writeFileSync(p, JSON.stringify(list, null, 2))
}
