// drill_mesh_restore.mjs — Custodian Wave 1, Mission CW1-B, mesh leg.
//
// "The folder IS the data" (CW10_INVENTORY.md Surface 7): mesh has no backup
// engine of its own — resurrection is a folder copy of data/keys/ +
// data/corestore/. This drill proves that claim end-to-end on scratch
// directories only, imitating the structure kit-spike.mjs already proved out
// (createKitHost / createCommandLayer / loadRoomRegistry, single-process,
// TCP-fallback-only, no hyperswarm/DHT — see kit-spike.mjs's own "HONEST
// SCOPE NOTE" for why that path is the deterministic one to drill hermetically).
//
// This is a NEW spike-style script; it does not modify any mesh runtime file.
//
// Prerequisite: `npm run build` in mesh/ (produces dist/reducer.wasm) — the
// same prerequisite kitspike has. Run from anywhere; all imports below are
// resolved relative to THIS FILE's own location, not the working directory.
//
// Run (from repo root, after `npm --prefix mesh run build`):
//   node scripts/custodian/drill_mesh_restore.mjs

import { mkdtempSync, rmSync, cpSync, existsSync, readdirSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { createKitHost } from '../../mesh/kit/kit-host.mjs'
import { createCommandLayer } from '../../mesh/kit/kit-repl.mjs'
// Imported by relative path (not the bare specifier) because this file lives
// outside mesh/, and Node's package resolution for a bare "corestore" only
// searches node_modules trees at or above the IMPORTING file — mesh/'s own
// node_modules is not on that path from scripts/custodian/.
import Corestore from '../../mesh/node_modules/corestore/index.js'

let failures = 0
function check(name, cond, detail = '') {
  if (cond) console.log(`  [PASS] ${name}`)
  else { failures++; console.log(`  [FAIL] ${name}${detail ? ' — ' + detail : ''}`) }
}
function checkExpectFailure(name, cond, detail = '') {
  // cond === true means "the failure we wanted actually happened" (RED as required).
  if (cond) console.log(`  [PASS-RED] ${name}${detail ? ' — ' + detail : ''}`)
  else { failures++; console.log(`  [FAIL] ${name} — expected this to fail, but it did not`) }
}
function section(title) { console.log(`\n== ${title} ==`) }

console.log('Custodian Wave 1 — CW1-B mesh-folder restore drill (folder IS the data)\n')

const runTS = new Date().toISOString().replace(/[:.]/g, '-')
const scratch = mkdtempSync(join(tmpdir(), `custodian-drill-mesh-${runTS}-`))
console.log(`Scratch root: ${scratch}`)

const originalDataDir = join(scratch, 'device-original')
const backupDataDir = join(scratch, 'backup-copy') // the "backup" — a folder copy, nothing more
const restoredDataDir = join(scratch, 'restored') // green-path restore target
const restoredNoKeysDataDir = join(scratch, 'restored-no-keys') // negative-control target

const sentinelBody = `CUSTODIAN-DRILL-MESH-SENTINEL-${runTS}`
const roomTitle = 'Custodian Drill Room'

let timers = {}

// ── 1. Scenario build: boot a device, create a room, post a sentinel ──
section('Scenario build (real kit host + real room)')
let host = await createKitHost({
  dataDir: originalDataDir, actor: 'custodian-drill', tcpPort: 0, useHyperswarm: false, log: () => {},
})
let cmds = createCommandLayer(host)

check('boot: device has a real identity', /^[0-9a-f]{64}$/.test(host.keys.pubHex))

await cmds.create(roomTitle)
const roomKey = host.currentRoomKey
check('create: room open on the drill device', typeof roomKey === 'string' && roomKey.length > 0)

await cmds.post(sentinelBody, '')
const preBackupState = await host.client.request('roomState', { roomKey })
check('scenario: sentinel message present before backup',
  preBackupState.messages.some((m) => m.body === sentinelBody))

await host.close()
check('scenario: device closed cleanly before folder copy (no live writer during backup)', true)

// ── 2. Backup == folder snapshot (the whole doctrine in one line) ──
section('Backup (folder copy — this IS the mesh backup)')
const backupStart = Date.now()
cpSync(originalDataDir, backupDataDir, { recursive: true })
const backupDuration = Date.now() - backupStart
check('backup: keys/ subtree copied', existsSync(join(backupDataDir, 'keys', 'device-seed.hex')))
check('backup: keys/rooms.json copied', existsSync(join(backupDataDir, 'keys', 'rooms.json')))
check('backup: corestore/ subtree copied', existsSync(join(backupDataDir, 'corestore')))
console.log(`  STOPWATCH backup (folder copy) leg: ${backupDuration}ms`)

// ── 3. Destroy the original (proves restore isn't reading the live dir) ──
section('Destroy original')
rmSync(originalDataDir, { recursive: true, force: true })
check('original data dir removed', !existsSync(originalDataDir))

// ── 4. Restore: copy the backup to a NEW location and reopen ──
section('Restore (green path) — reopen via the kit registry, content-assert the sentinel')
const restoreStart = Date.now()
cpSync(backupDataDir, restoredDataDir, { recursive: true })
const restoreDuration = Date.now() - restoreStart
console.log(`  STOPWATCH restore (folder copy) leg: ${restoreDuration}ms`)

let restoredHost = await createKitHost({
  dataDir: restoredDataDir, actor: 'custodian-drill', tcpPort: 0, useHyperswarm: false,
  log: (m) => console.log(`  [host log] ${m}`),
})
const roomCameBack = restoredHost.server.rooms.has(roomKey)
check('restore: the room comes back automatically via kit-registry.mjs', roomCameBack)
if (roomCameBack) {
  const restoredState = await restoredHost.client.request('roomState', { roomKey })
  check('restore: manifest title survives', restoredState.manifest?.title === roomTitle)
  check('restore: sentinel message reads back CONTENT-identical after restore',
    restoredState.messages.some((m) => m.body === sentinelBody))
} else {
  check('restore: manifest title survives', false, 'skipped — room did not come back, see [host log] above')
  check('restore: sentinel message reads back CONTENT-identical after restore', false, 'skipped — room did not come back')
}
await restoredHost.close()

// ── 4b. Root-cause diagnostic (read-only probe, no runtime code touched) ──
// If the reopen above failed with a device-file / "was modified" error, this
// probe confirms WHY: hypercore-storage (node_modules/hypercore-storage,
// via node_modules/device-file) writes a sentinel "CORESTORE" file recording
// the storage directory's inode at creation time, and refuses to reopen if
// the inode differs — a move/copy-safety guard against silently forking a
// replicated multiwriter store. A plain recursive folder copy ALWAYS
// produces a new inode for every file, so this guard trips on every
// restore-via-folder-copy, not just a corrupted one. hypercore-storage
// exposes `allowBackup: true` (threaded from `new Corestore(dir, opts)`)
// specifically to skip this check when the caller KNOWS the directory is a
// deliberate restore, not tampering — but mesh-node.mjs (the only place
// AsymmFlow constructs a Corestore for a real room) never passes it. Fixing
// that is a runtime-code change and is OUT OF SCOPE for this drill
// (stop-and-report, not silently patched) — this probe exists only to prove
// the diagnosis, using the dependency directly, touching zero mesh/ files.
section('Root-cause diagnostic — device-file inode guard (read-only probe)')
{
  const probeDir = join(restoredDataDir, 'corestore', 'room-does-not-exist-guard-probe')
  // Re-derive the actual copied room's corestore path from the registry entry
  // written during scenario build, rather than hardcoding the storage name.
  const roomsJsonPath = join(restoredDataDir, 'keys', 'rooms.json')
  let realStorageDir = probeDir
  try {
    const { readFileSync } = await import('node:fs')
    const entries = JSON.parse(readFileSync(roomsJsonPath, 'utf8'))
    const entry = entries.find((e) => e.roomKey === roomKey)
    if (entry) realStorageDir = join(restoredDataDir, 'corestore', entry.storage)
  } catch { /* best-effort — falls back to the (nonexistent) probeDir, probe just no-ops */ }

  try {
    const probeStore = new Corestore(realStorageDir, { allowBackup: true })
    await probeStore.ready()
    check('diagnostic: allowBackup:true opens the SAME copied directory that just failed',
      true, 'root cause confirmed = device-file inode guard, not data corruption')
    await probeStore.close()
  } catch (err) {
    check('diagnostic: allowBackup:true opens the SAME copied directory that just failed',
      false, `probe also failed (${err.message}) — root cause is NOT solely the inode guard, needs further investigation`)
  }
}

// ── 5. NEGATIVE CONTROL: restore with data/keys/ withheld ──
section('Negative control — restore WITHOUT data/keys/ (mandatory RED)')
cpSync(join(backupDataDir, 'corestore'), join(restoredNoKeysDataDir, 'corestore'), { recursive: true })
check('negative-control fixture: corestore/ present, keys/ deliberately withheld',
  existsSync(join(restoredNoKeysDataDir, 'corestore')) && !existsSync(join(restoredNoKeysDataDir, 'keys')))

let noKeysHost
let noKeysThrew = false
try {
  // No --actor on file and stdin is non-interactive in this drill process,
  // so a fresh keys/ dir would normally prompt; pass actor explicitly so the
  // negative control fails for the REASON under test (missing room registry
  // / new device identity), not an unrelated stdin prompt.
  noKeysHost = await createKitHost({
    dataDir: restoredNoKeysDataDir, actor: 'custodian-drill-reopened-without-keys', tcpPort: 0, useHyperswarm: false, log: () => {},
  })
} catch (err) {
  noKeysThrew = true
  console.log(`  (createKitHost threw: ${err.message})`)
}

if (!noKeysThrew) {
  const gotNewIdentity = noKeysHost.keys.pubHex !== host.keys.pubHex
  checkExpectFailure('negative control: a NEW device identity was minted (old identity is gone with keys/)',
    gotNewIdentity, `new pubHex=${noKeysHost.keys.pubHex.slice(0, 16)}…`)
  checkExpectFailure('negative control: the room is NOT found (no rooms.json survived)',
    !noKeysHost.server.rooms.has(roomKey), 'server.rooms does not contain the original roomKey')
  await noKeysHost.close()
} else {
  checkExpectFailure('negative control: reopen without keys/ failed outright', true, 'createKitHost threw as expected')
}

// ── cleanup ──
try { rmSync(scratch, { recursive: true, force: true }) } catch {}

section('Summary')
console.log(`Backup (folder copy) leg:  ${backupDuration}ms`)
console.log(`Restore (folder copy) leg: ${restoreDuration}ms`)
console.log(failures === 0
  ? '\nMESH RESTORE DRILL GREEN — folder backup/restore proven, negative control fired red as required.'
  : `\nMESH RESTORE DRILL RED — ${failures} assertion(s) failed. See [FAIL] lines above.`)
process.exit(failures === 0 ? 0 : 1)
