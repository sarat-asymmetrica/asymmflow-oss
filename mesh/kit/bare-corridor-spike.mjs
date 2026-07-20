// bare-corridor-spike.mjs — SC-3a (Sealed Corridor campaign) gate: the
// two-sided corridor ceremony (bare-guide.mjs's menu [2] founder/joiner
// fork) through the REAL launcher, two real sealed-kit processes, a real
// bidirectional relay between them.
//
// RULE 3 (mesh/docs/bare-campaign/CAMPAIGN_REPORT.md §4, restated for this
// wave in FABLE_CAMPAIGN_SEALED_CORRIDOR.md's own SC-3 gate text): drives
// `run_bare_mesh.cmd`, never `bare.exe app.bundle` directly. The `cmd.exe /c
// <abs path>` + `shell:false` + `ASYMMFLOW_KIT_NONINTERACTIVE=1` technique
// below is copied from sealed-corridor-gate.mjs's own `runLauncher` —
// EXACT SAME shape, not reinvented — because that file is the orchestrator's
// own independent proof that Git Bash mangles `cmd //c` (three false
// failures on this campaign line already; do not run this file from a POSIX
// shell).
//
// WHY THIS FILE DOES NOT REUSE spawn-pipe-harness.mjs's runSpawnPipe() (OR
// sealed-corridor-gate.mjs's own fixed-stdin runLauncher) FOR THE POSITIVE
// CEREMONY — the same reason bare-net-spike.mjs's own header gives for its
// choice: the ceremony needs a REAL bidirectional relay between two LIVE
// processes (kit B's pairing code, read off B's live stdout, typed into A's
// live stdin, mid-run) — there is no way to precompute that stdin script.
// `spawnLauncherLive`/`makeLineReader`/`expectLine`/`send` below are this
// file's own version of exactly what bare-net-spike.mjs already proved for
// the lower-level `bare-corridor-entry.mjs` protocol; this file drives the
// CLIENT-FACING guide instead, through the REAL launcher instead of
// `bare.exe app.bundle` directly (that is the whole point of this gate —
// SC-2's own spike explicitly does NOT prove the client's onboarding path,
// see that file's own header).
//
// The SINGLE-kit negative controls (malformed code / well-formed-but-wrong
// code) need no relay at all — those use a fixed-stdin variant of the same
// launcher technique, `runLauncherFixed`, copied in spirit from
// sealed-corridor-gate.mjs's own `runLauncher`.
//
// NEGATIVE CONTROLS RUN FIRST, ALWAYS (Rule 1: a probe that cannot report
// the opposite result proves nothing). See `provingGroundsRed()`.
//
// Run: npm run sc3aspike   (equivalently: node kit/bare-corridor-spike.mjs)

import { spawn, execFileSync } from 'node:child_process'
import { mkdtempSync, rmSync, existsSync, cpSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { randomBytes } from 'node:crypto'
import { encodeInviteCode } from '../host/invite-code.mjs'

const __dirname = dirname(fileURLToPath(import.meta.url))
const meshRoot = join(__dirname, '..')

// LEAK FOUND LIVE AT THIS GATE'S OWN FIRST FULL RUN (2026-07-20), flagged by
// the orchestrator after reclaiming 2.17 GB of abandoned kit directories:
// `child.kill('SIGKILL')` on the spawned `cmd.exe` process does NOT kill its
// `bare.exe` GRANDCHILD on Windows — cmd.exe's own child tree is not
// cascaded by a plain kill. Every round this gate deliberately terminates
// EARLY (the FAST negative-control leg, N=16, kills before the guide's own
// 90s wait completes BY DESIGN — see controlGhostFast) therefore orphaned a
// live bare.exe still holding file handles open on that round's kit
// directory, which made releaseKit's `rmSync` fail SILENTLY (its own
// best-effort catch), leaking ~62-137 MB per orphan. The disk-full errors
// this then produced downstream ("EINPROGRESS", actually errno 112 =
// ERROR_DISK_FULL — see assertStagingRoom below) were a SYMPTOM; this is the
// disease. Fixed by killing the whole process TREE via `taskkill /T /F`,
// never a bare `child.kill()`, anywhere this file terminates a launcher
// before it exits on its own.
function killTree(pid) {
  if (!pid) return
  try { execFileSync('taskkill', ['/T', '/F', '/PID', String(pid)], { stdio: 'ignore' }) } catch { /* already gone, or never had children */ }
}

// Own, PRIVATE output dir — never shared with any other mission's gate
// (build-bare-kit.mjs unconditionally wipes its target; a shared dir means
// one gate's rmSync can delete another gate's kit mid-copy — the exact
// self-inflicted failure bare-net-spike.mjs's own header already names).
const OUT_DIR = 'kit/.sc3a-dist'
const BUILT = join(meshRoot, OUT_DIR)
// Measured directly against THIS entry (kit/bare-guide-entry.mjs), same
// four network-carrying addons SC0_PORT_MAP.md §2/§3.2 named for the
// corridor: udx-native + sodium-native (hyperswarm/hyperdht's DHT+crypto),
// bare-tcp (the LAN fallback), bare-dns (hyperdht's own bootstrap resolve).
// This is `--require-addons`, the SAME opt-in hard gate build-bare-kit.mjs's
// own §2c documents: if bare-pack ever stops resolving menu [1]'s or menu
// [2]'s dynamic `import('./bare-net.mjs')`/`import('./bare-connection-
// check.mjs')`, THIS BUILD fails loudly here rather than the field failing
// quietly with a kit that renders its whole ceremony and can never reach
// the network.
const REQUIRED_ADDONS = 'udx-native,sodium-native,bare-tcp,bare-dns'

let checks = 0
let failures = 0
function check(name, cond, detail = '') {
  checks++
  if (cond) console.log(`  OK   ${name}`)
  else { failures++; console.log(`  FAIL ${name}${detail ? ' -- ' + detail : ''}`) }
  return cond
}

// ── hostile-directory staging — same discipline as sealed-corridor-
// gate.mjs's own freshKit/releaseKit/cleanup (its own header explains the
// EINPROGRESS lesson this copies verbatim): assert no '#' in the path
// (Bare addon resolution breaks otherwise — merge-gate finding 2026-07-20),
// retry setup (never the ceremony itself) against transient Windows FS
// errors, and RELEASE each staged kit the moment its own run is done rather
// than accumulating ~65 MB copies across a long run. ─────────────────────
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

const tmpDirs = []
function hostileDir(tag) {
  const d = mkdtempSync(join(tmpdir(), `asymm-sc3a-${tag}-`))
  if (d.includes('#')) throw new Error(`hostile dir contains '#', which breaks Bare addon resolution: ${d}`)
  tmpDirs.push(d)
  return d
}
// THE MISLABELLED ERROR THAT COST THIS CAMPAIGN AN HOUR, restated here
// rather than only cited (sealed-corridor-gate.mjs's own header has the full
// account): on Windows, a disk-full condition surfaces from `cpSync` as
// `Error: EINPROGRESS, unknown error '\\?\C:\...'  errno: 112`. `EINPROGRESS`
// READS like a transient race worth retrying. IT IS NOT -- errno 112 is
// `ERROR_DISK_FULL`, and Node has no mapping for it on Windows, so it falls
// back to a name that actively misleads. Retrying a full disk cannot
// succeed; retrying it is exactly what buries the real cause (a genuine
// transient error does not reproduce identically three times running, so a
// same-error retry loop is itself a signal, not a fix). Check FIRST and say
// the true thing, copied in spirit from sealed-corridor-gate.mjs's own
// `assertStagingRoom`/`freeBytesOnTemp` (independently re-derived here, not
// imported, per single-writer file ownership -- that file is not touched by
// this mission).
function freeBytesOnTemp() {
  try {
    const out = execFileSync('cmd.exe', ['/c', 'dir', '/-c', tmpdir()], { encoding: 'utf8' })
    const m = out.match(/(\d+)\s+bytes free/i)
    return m ? Number(m[1]) : null
  } catch { return null }
}

// The corridor kit is ~73 MB per the orchestrator's own measurement
// (2026-07-20, grown from ~62 MB since the network addons landed); rounded
// up with headroom. A ceremony round stages TWO kits (A and B) at once, so
// the room check below requires margin for the current round's PAIR plus
// slack for the next round to start before this one's release lands.
const KIT_STAGE_BYTES = 80 * 1024 * 1024

function assertStagingRoom(tag) {
  const free = freeBytesOnTemp()
  if (free === null) return // could not measure -- do not invent a verdict
  if (free < KIT_STAGE_BYTES * 4) {
    throw new Error(
      `not enough free disk to stage a kit for ${tag}: ${(free / 1e6).toFixed(0)} MB free, `
      + `each staged kit is ~${(KIT_STAGE_BYTES / 1e6).toFixed(0)} MB and a ceremony round stages two. `
      + 'NOTE: Windows reports this through Node as "EINPROGRESS, unknown error" (errno 112 = '
      + 'ERROR_DISK_FULL), which reads like a transient race and is not one. '
      + 'Check %TEMP% for leaked kit copies from earlier gate runs.')
  }
}

// HARNESS ROBUSTNESS, narrowed: retries are for genuinely transient Windows
// FS contention (AV/OS holding a handle briefly) ONLY. A disk-full error is
// classified and thrown IMMEDIATELY, never retried -- see the block comment
// above for why retrying it cannot succeed and actively hides the cause.
async function freshKit(tag) {
  assertStagingRoom(tag)
  let lastErr = null
  for (let attempt = 1; attempt <= 3; attempt++) {
    if (attempt > 1) await sleep(750 * attempt)
    const d = hostileDir(tag)
    try {
      cpSync(BUILT, d, { recursive: true })
      return d
    } catch (err) {
      lastErr = err
      if (err?.errno === 112 || err?.code === 'EINPROGRESS') {
        throw new Error(
          `staging failed for ${tag}: the disk is FULL (errno 112 = ERROR_DISK_FULL, which Node `
          + `surfaces as the misleading "EINPROGRESS"). Not retried -- retrying a full disk cannot `
          + `succeed. Free space in ${tmpdir()} and re-run. Original: ${err?.message ?? err}`)
      }
      console.log(`  (setup retry ${attempt}/3 for ${tag}: ${err?.code ?? ''} ${err?.message ?? err})`)
    }
  }
  throw new Error(`could not stage a fresh kit for ${tag} after 3 attempts: ${lastErr?.message ?? lastErr}`)
}
function releaseKit(d) {
  try { rmSync(d, { recursive: true, force: true }) } catch { /* best-effort */ }
  const i = tmpDirs.indexOf(d)
  if (i !== -1) tmpDirs.splice(i, 1)
}
function cleanup() {
  for (const d of [...tmpDirs]) releaseKit(d)
}

// ── the real launcher, fixed-stdin variant — for the single-kit negative
// controls, which need no live relay. Copied in spirit from sealed-
// corridor-gate.mjs's own `runLauncher` (same cmd.exe /c technique, same
// ASYMMFLOW_KIT_NONINTERACTIVE opt-in, same explicit-timeout-means-HANG
// semantics). ────────────────────────────────────────────────────────────
function runLauncherFixed({ kitDir, stdin, timeoutMs = 45000 }) {
  return new Promise((resolve) => {
    const comspec = process.env.ComSpec || 'C:\\Windows\\System32\\cmd.exe'
    const child = spawn(comspec, ['/c', join(kitDir, 'run_bare_mesh.cmd')], {
      cwd: kitDir, stdio: ['pipe', 'pipe', 'pipe'], shell: false,
      env: { ...process.env, ASYMMFLOW_KIT_NONINTERACTIVE: '1' },
    })
    let stdout = ''
    let stderr = ''
    let settled = false
    const t0 = Date.now()
    const timer = setTimeout(() => {
      if (settled) return
      settled = true
      // TREE kill, not a bare child.kill() -- see killTree's own comment
      // above (the leak this gate found in itself). This leg (controlGhostFast)
      // deliberately fires this path on EVERY round, so getting it wrong
      // here is what actually leaked, not an edge case.
      killTree(child.pid)
      resolve({ stdout, stderr, code: null, hang: true, ms: Date.now() - t0 })
    }, timeoutMs)
    child.stdout.on('data', (d) => { stdout += d.toString('utf8') })
    child.stderr.on('data', (d) => { stderr += d.toString('utf8') })
    child.on('close', (code) => {
      if (settled) return
      settled = true
      clearTimeout(timer)
      resolve({ stdout, stderr, code, hang: false, ms: Date.now() - t0 })
    })
    child.on('error', (err) => {
      if (settled) return
      settled = true
      clearTimeout(timer)
      resolve({ stdout, stderr: stderr + String(err), code: -1, hang: false, ms: Date.now() - t0 })
    })
    child.stdin.write(stdin)
    child.stdin.end()
  })
}
const stdinScript = (lines) => lines.join('\r\n') + '\r\n'

// ── the real launcher, LIVE variant — for the two-sided ceremony. Same
// spawn technique as runLauncherFixed (same cmd.exe /c, same
// ASYMMFLOW_KIT_NONINTERACTIVE), but stdin/stdout stay open for interactive
// send()/expectLine() rather than being fed one fixed blob up front. ─────
function spawnLauncherLive(kitDir, label) {
  const comspec = process.env.ComSpec || 'C:\\Windows\\System32\\cmd.exe'
  const child = spawn(comspec, ['/c', join(kitDir, 'run_bare_mesh.cmd')], {
    cwd: kitDir, stdio: ['pipe', 'pipe', 'pipe'], shell: false,
    env: { ...process.env, ASYMMFLOW_KIT_NONINTERACTIVE: '1' },
  })
  return { child, reader: makeLineReader(child), label }
}

// Same queue-of-lines reader shape as bare-net-spike.mjs's own
// makeLineReader (design law: distinguish a real line from a timeout from a
// closed process — never collapse the three).
function makeLineReader(child) {
  let buf = ''
  const queue = []
  const waiters = []
  let ended = false
  let stderrBuf = ''
  child.stdout.on('data', (d) => {
    buf += d.toString('utf8')
    let idx
    while ((idx = buf.indexOf('\n')) !== -1) {
      const line = buf.slice(0, idx).replace(/\r$/, '')
      buf = buf.slice(idx + 1)
      if (waiters.length) waiters.shift()(line)
      else queue.push(line)
    }
  })
  child.stderr.on('data', (d) => { stderrBuf += d.toString('utf8') })
  child.on('close', () => { ended = true; while (waiters.length) waiters.shift()(null) })
  return {
    // Returns a line, null (process ended, no more lines ever), or
    // undefined (timed out — three distinct outcomes, never collapsed).
    nextLine(timeoutMs) {
      if (queue.length) return Promise.resolve(queue.shift())
      if (ended) return Promise.resolve(null)
      return new Promise((resolve) => {
        const timer = setTimeout(() => {
          const i = waiters.indexOf(onLine)
          if (i !== -1) waiters.splice(i, 1)
          resolve(undefined)
        }, timeoutMs)
        function onLine(line) { clearTimeout(timer); resolve(line) }
        waiters.push(onLine)
      })
    },
    get stderr() { return stderrBuf },
  }
}

/** expectLine(reader, predicate, timeoutMs, label) -> matching line, or
 * throws with a plain reason (never a bare undefined/null). */
async function expectLine(reader, predicate, timeoutMs, label) {
  const deadline = Date.now() + timeoutMs
  for (;;) {
    const remaining = deadline - Date.now()
    if (remaining <= 0) throw new Error(`timeout waiting for ${label}`)
    const line = await reader.nextLine(remaining)
    if (line === null) throw new Error(`process closed while waiting for ${label} (stderr: ${reader.stderr.slice(0, 300)})`)
    if (line === undefined) throw new Error(`timeout waiting for ${label}`)
    if (predicate(line)) return line
  }
}

function send(child, line) { child.stdin.write(line + '\n') }

async function closeLauncher(proc, { graceful = true } = {}) {
  if (graceful) {
    try { send(proc.child, '/exit'); send(proc.child, '5') } catch { /* best-effort */ }
  }
  await Promise.race([
    new Promise((r) => proc.child.once('close', r)),
    sleep(15000),
  ])
  // TREE kill -- see killTree's own comment. A no-op (already-exited PID) if
  // the graceful close above already succeeded; the leak-relevant case is
  // when it did NOT, and the bare.exe grandchild would otherwise survive
  // this call and keep the round's kit directory un-releasable.
  killTree(proc.child.pid)
}

// ── the invite code line, as printed: `groupInFours` inserts a single
// space every 4 characters (kit/bare-guide.mjs's own `groupInFours`,
// byte-for-byte the same regex as guide.mjs's — see that file for why this
// is safe to strip: `normalizeCode` on the receiving end does the exact
// inverse). Stripping ALL whitespace reconstructs the original string
// exactly, because grouping only ever INSERTS separators, never alters
// characters. ───────────────────────────────────────────────────────────
const degroup = (line) => line.replace(/\s+/g, '')

// ═══════════════════════════════════════════════════════════════════════
// PROVING GROUNDS — negative controls, run FIRST, gate everything else.
// Per Rule 1: if any control here fails to go red, no positive result
// below is admissible.
// ═══════════════════════════════════════════════════════════════════════

// Negative control (i): a MALFORMED invite code. decodeInviteCode's own
// error is plain language; the guide must refuse with it (never a stack
// trace ON THE CLIENT'S OWN LINE — reportError()'s fold convention puts the
// stack BELOW a fold, which this assertion respects: it checks for the
// plain sentence, not the absence of a stack trace anywhere in the output)
// and the guide must SURVIVE (reach Goodbye after returning to its menu).
// Not hang-plausible (a synchronous decode failure, no network, no wait) —
// N=5 is a correctness/shape proof, not a rate measurement (Rule 5 exempts
// non-race-prone checks; same reasoning as sealed-corridor-gate.mjs's own
// registry-robustness N=3 choice).
async function controlMalformedCode(runs) {
  console.log(`\n== control (i): a malformed invite code, N=${runs} ==`)
  let ok = 0
  let firstBad = null
  for (let i = 1; i <= runs; i++) {
    const kit = await freshKit('ctl-malformed')
    const r = await runLauncherFixed({
      kitDir: kit,
      // 'connect' answers the unified "already have a conversation?" / "start
      // one" question (openMessenger's own D3 fix, 2026-07-20) -- Enter alone
      // would skip past the code question entirely and just open/create the
      // local room. See that function's own comment for why the question is
      // now the SAME regardless of hasRoom.
      stdin: stdinScript(['2', 'skip', 'connect', `not-a-real-invite-code-${i}`, '5']),
      timeoutMs: 30000,
    })
    const plainRefusal = r.stdout.includes('not an AsymmFlow room invite code')
      && r.stdout.includes('read this line to the person on the phone')
    const survived = r.stdout.includes('Goodbye -- this window is safe to close.') && !r.hang
    if (plainRefusal && survived) ok++
    else if (!firstBad) firstBad = { i, hang: r.hang, plainRefusal, survived, out: r.stdout.slice(-500) }
    releaseKit(kit)
  }
  console.log(`  control (i): OK=${ok}/${runs}`)
  return check(`control (i): a malformed invite code refuses in plain language and the guide survives, ${ok}/${runs}`,
    ok === runs, firstBad ? JSON.stringify(firstBad) : '')
}

/** A syntactically-valid room2 invite code for a room NOBODY founded —
 * decodeInviteCode accepts it (correct shape), but no founder is listening
 * on its derived topic and no writer grant will ever arrive. Exercises
 * "well-formed but wrong" (ii) and doubles as "unreachable network" (iii):
 * this specific room's network path genuinely cannot ever complete, which
 * is the honest scope note for why this gate does not separately sever the
 * real internet/DHT — see this file's own report for that framing. */
function ghostInviteCode() {
  const rb = (n) => randomBytes(n)
  return encodeInviteCode({
    baseKey: rb(32), authorityPub: rb(32), inviteSeed: rb(32),
    inviteId: 'ghost:1', encryptionKey: rb(32),
  })
}

// Negative control (ii)/(iii), FAST leg: hang-plausible by definition (a
// real bounded network wait), so Rule 5 demands N>=16. This leg does NOT
// wait for the guide's own JOIN_WRITABLE_TIMEOUT_MS (90s) to fire — it
// kills the process at a much shorter window and asserts the ONE thing a
// fast sweep can honestly prove across N=16: the join success marker NEVER
// appears. Being killed mid-wait is the EXPECTED, chosen shape for this
// leg (not a hang finding) — see controlGhostSlow below for the leg that
// proves the internal timeout itself is real and prints honest copy.
async function controlGhostFast(runs) {
  console.log(`\n== control (ii)/(iii) FAST: well-formed-but-wrong code, correctness sweep, N=${runs} ==`)
  let ok = 0
  let firstBad = null
  for (let i = 1; i <= runs; i++) {
    const kit = await freshKit('ctl-ghost-fast')
    const code = ghostInviteCode()
    // '' answers the LAN-address question -- genuinely no path exists, per
    // this control's own framing.
    const r = await runLauncherFixed({
      kitDir: kit,
      stdin: stdinScript(['2', 'skip', 'connect', code, '']),
      timeoutMs: 15000, // deliberately shorter than the guide's own 90s wait
    })
    const falseJoin = r.stdout.includes('(joined -- you can post now)')
    if (!falseJoin) ok++
    else if (!firstBad) firstBad = { i, out: r.stdout.slice(-500) }
    releaseKit(kit)
  }
  console.log(`  control (ii)/(iii) FAST: OK=${ok}/${runs} (never falsely joined)`)
  return check(`control (ii)/(iii) FAST: a well-formed-but-wrong invite code never joins, ${runs}/${runs}, N>=16 per hang-plausible law`,
    ok === runs, firstBad ? JSON.stringify(firstBad) : '')
}

// Negative control (ii)/(iii), HONEST-COPY leg: lets the guide's own
// JOIN_WRITABLE_TIMEOUT_MS actually fire and proves the SHAPE — the exact
// honest-failure copy appears, the guide returns to its menu, and it closes
// cleanly (never a literal hang). Correctness/shape proof, not a rate
// (same N=3 reasoning as sealed-corridor-gate.mjs's own registry-robustness
// leg) — this is expensive per-run (bounded by the guide's own 90s wait
// plus process overhead) so a small N is a declared choice, not a shortfall.
async function controlGhostSlow(runs) {
  console.log(`\n== control (ii)/(iii) SLOW: well-formed-but-wrong code, honest-failure-copy shape, N=${runs} ==`)
  let ok = 0
  let firstBad = null
  for (let i = 1; i <= runs; i++) {
    const kit = await freshKit('ctl-ghost-slow')
    const code = ghostInviteCode()
    const r = await runLauncherFixed({
      kitDir: kit,
      stdin: stdinScript(['2', 'skip', 'connect', code, '', '5']),
      timeoutMs: 150000, // 90s internal wait + generous margin for process overhead
    })
    const honestCopy = r.stdout.includes('timed out waiting for the other computer letting you in')
      && r.stdout.includes('read this line to the person on the phone')
    const neverJoined = !r.stdout.includes('(joined -- you can post now)')
    const survived = r.stdout.includes('Goodbye -- this window is safe to close.') && !r.hang
    if (honestCopy && neverJoined && survived) ok++
    else if (!firstBad) firstBad = { i, hang: r.hang, ms: r.ms, honestCopy, neverJoined, survived, out: r.stdout.slice(-600) }
    releaseKit(kit)
  }
  console.log(`  control (ii)/(iii) SLOW: OK=${ok}/${runs}`)
  return check(`control (ii)/(iii) SLOW: the honest failure copy appears and the guide closes cleanly, never a hang, ${ok}/${runs}`,
    ok === runs, firstBad ? JSON.stringify(firstBad) : '')
}

async function provingGroundsRed() {
  console.log('\n== proving grounds: can this driver report a FAILURE? (Rule 1) ==')
  const a = await controlMalformedCode(5)
  const b = await controlGhostFast(16)
  const c = await controlGhostSlow(3)
  return a && b && c
}

// ═══════════════════════════════════════════════════════════════════════
// POSITIVE: the real two-sided ceremony through the REAL launcher.
// A mints, B redeems (via a LIVE relay, scripted by THIS process reading
// A's/B's real stdout and typing into the other's real stdin), A grants,
// B posts, A reads B's message back — content-asserted BOTH directions
// (the campaign spec asks for one direction; both is strictly stronger and
// cheap, matching SC-2's own both-directions assertion).
// ═══════════════════════════════════════════════════════════════════════

/** One full round. `lan`: true = B is given A's 127.0.0.1:<port> LAN hint
 * (deterministic, fast — the GATED leg); false = B relies on hyperswarm
 * alone (the MEASURED, not gated, leg — matching SC-2's own TCP-vs-swarm
 * split and its stated reason: live DHT is legitimately allowed to be
 * flaky, SC0_PORT_MAP.md §1). Returns { ok, reason }. */
async function runCeremonyRound(tag, lan) {
  const dirA = await freshKit(`${tag}-a`)
  const dirB = await freshKit(`${tag}-b`)
  const A = spawnLauncherLive(dirA, 'A')
  const B = spawnLauncherLive(dirB, 'B')
  const markA = `sc3a-${tag}-from-A-${Date.now()}`
  const markB = `sc3a-${tag}-from-B-${Date.now()}`
  try {
    // A: choose messenger, skip the one-time firewall notice, type "connect"
    // at the unified "already have a conversation?"/"start one" question
    // (openMessenger's own D3 fix, 2026-07-20 -- Enter alone would just
    // open/create the local room and never reach the code question), press
    // Enter at "did someone send you a code?" -- founder path.
    send(A.child, '2')
    send(A.child, 'skip')
    send(A.child, 'connect')
    send(A.child, '')

    // 90s, not 20s like every later wait — this FIRST wait is the only one
    // that absorbs the whole cold boot (bare.exe start + Defender scanning a
    // freshly copied unsigned 45 MB exe + bundle load + reducer wasm + room
    // founding); every wait after it runs in a warm process. Measured at the
    // SC-5 merge gate (2026-07-20): with 20s this bound produced exactly one
    // false red in 32 otherwise-green rounds (run 1: 15/16, the miss HERE and
    // only here; run 2: 16/16) — the same cold-start class sealed-corridor-
    // gate.mjs's own leg C already widened its timeout for, same reasoning:
    // measure the kit, not the machine's load. A genuine wedge still fails.
    await expectLine(A.reader, (l) => l.includes('Here is the code for the OTHER computer'), 90000, 'A invite intro')
    const codeLine = await expectLine(A.reader, (l) => l.trim().length > 100, 20000, 'A invite code line')
    const inviteCode = degroup(codeLine)

    let lanHint = ''
    if (lan) {
      const portLine = await expectLine(A.reader, (l) => /port (\d+)/i.test(l), 20000, 'A tcp port')
      const port = portLine.match(/port (\d+)/i)[1]
      lanHint = `127.0.0.1:${port}`
    }

    // B: choose messenger, skip firewall, "connect", paste A's invite code,
    // answer the LAN-address question (hint or blank, per `lan`).
    send(B.child, '2')
    send(B.child, 'skip')
    send(B.child, 'connect')
    send(B.child, inviteCode)
    send(B.child, lanHint)

    await expectLine(B.reader, (l) => l.includes('Your code for the OTHER computer'), 20000, 'B pairing intro')
    const pairingLine = await expectLine(B.reader, (l) => degroup(l).length === 64 && /^[0-9a-f]+$/i.test(degroup(l)), 20000, 'B pairing code line')
    const pairingCode = degroup(pairingLine)

    // A is still parked at its own "paste the pairing code" prompt --
    // send it now.
    send(A.child, pairingCode)
    await expectLine(A.reader, (l) => l.includes('the other computer can finish joining now'), 20000, 'A addwriter ack')

    await expectLine(B.reader, (l) => l.includes('(joined -- you can post now)'), lan ? 30000 : 60000, 'B joined')

    send(A.child, markA)
    await expectLine(A.reader, (l) => l.startsWith('  (posted, seq'), 20000, 'A posted')
    send(B.child, markB)
    await expectLine(B.reader, (l) => l.startsWith('  (posted, seq'), 20000, 'B posted')

    // A reads B's message back -- poll `/read`, one full exchange per
    // attempt (never pipeline unacked commands -- bare-net-spike.mjs's own
    // collectUntil discipline). This is the spec's own literal requirement
    // ("A reads B's message back").
    //
    // `/read` (bare-guide.mjs's own `showConversation`, added mid-mission at
    // THIS gate's own finding -- see that function's header comment) prints
    // the last 10 messages, one per line, in canonical order -- NOT `/rooms`'
    // single "last message" summary line. An earlier draft of this function
    // used `/rooms` and posted a SECOND marker from A to work around
    // `/rooms` structurally being unable to show A's message once B's own
    // later post became the canonical "last" one -- `/read` needs no such
    // workaround, since it is a real multi-message history, not a
    // single-line summary. Simpler and closer to how a real ceremony is
    // actually verified (§4c of the runbook: "use /read, not /rooms").
    let sawB = false
    for (let i = 0; i < 20 && !sawB; i++) {
      send(A.child, '/read')
      try {
        const l = await expectLine(A.reader, (x) => x.includes(markB), 5000, 'A /read response')
        if (l.includes(markB)) sawB = true
      } catch { /* this round timed out -- try again */ }
      if (!sawB) await sleep(400)
    }
    if (!sawB) return { ok: false, reason: 'A never saw B\'s exact message text' }

    // Both directions, content-asserted (stronger than the spec's own
    // literal ask of one direction) -- B's own `/read` for A's message,
    // no second post needed.
    let sawA = false
    for (let i = 0; i < 20 && !sawA; i++) {
      send(B.child, '/read')
      try {
        const l = await expectLine(B.reader, (x) => x.includes(markA), 5000, 'B /read response')
        if (l.includes(markA)) sawA = true
      } catch { /* this round timed out -- try again */ }
      if (!sawA) await sleep(400)
    }
    if (!sawA) return { ok: false, reason: 'B never saw A\'s exact message text' }

    return { ok: true, reason: '' }
  } catch (err) {
    return { ok: false, reason: err?.message ?? String(err) }
  } finally {
    await closeLauncher(A)
    await closeLauncher(B)
    releaseKit(dirA)
    releaseKit(dirB)
  }
}

async function ceremonyLeg(label, lan, runs) {
  console.log(`\n== positive: two-sided ceremony (${label}), N=${runs} ==`)
  let ok = 0
  const bad = []
  for (let i = 1; i <= runs; i++) {
    const r = await runCeremonyRound(`${label.replace(/[^a-z0-9]/gi, '')}${i}`, lan)
    console.log(`  round ${i}/${runs}: ${r.ok ? 'OK' : 'FAIL -- ' + r.reason}`)
    if (r.ok) ok++
    else bad.push({ i, reason: r.reason })
  }
  console.log(`  ${label}: OK=${ok}/${runs}`)
  return { ok, runs, bad }
}

// ── main ──────────────────────────────────────────────────────────────────
console.log('bare-corridor-spike -- SC-3a: the two-sided corridor ceremony, real launcher, live relay\n')

try {
  console.log(`building the sealed kit into ${OUT_DIR} (entry: kit/bare-guide-entry.mjs)...`)
  execFileSync(process.execPath, [
    join(meshRoot, 'kit', 'build-bare-kit.mjs'),
    '--entry=kit/bare-guide-entry.mjs',
    `--out=${OUT_DIR}`,
    `--require-addons=${REQUIRED_ADDONS}`,
  ], { cwd: meshRoot, stdio: 'pipe' })

  check('build: app.bundle produced', existsSync(join(BUILT, 'app.bundle')))
  check('build: bare.exe copied into the kit', existsSync(join(BUILT, 'bare.exe')))
  check('build: run_bare_mesh.cmd produced (the layer this gate exists to drive)', existsSync(join(BUILT, 'run_bare_mesh.cmd')))
  check('build: dist/reducer.wasm offloaded', existsSync(join(BUILT, 'dist', 'reducer.wasm')))
  check(`build: required network addons present (${REQUIRED_ADDONS}) -- build-bare-kit.mjs's own hard gate would have thrown otherwise`,
    existsSync(join(BUILT, 'node_modules', 'bare-tcp')))

  const red = await provingGroundsRed()
  if (!red) {
    console.log('\nREFUSING TO REPORT POSITIVE RESULTS: at least one negative control did not go red.')
    console.log('A driver that cannot report failure proves nothing by succeeding (Rule 1).')
    console.log(`\n${checks} check(s), ${failures} failure(s).`)
    console.log('\nSC3A CORRIDOR SPIKE RED (controls)')
    cleanup()
    process.exit(1)
  }
  console.log('  (all controls went red as required -- positive results below are admissible)')

  const lanLeg = await ceremonyLeg('LAN-assisted', true, 16)
  check(`positive: two-sided ceremony, LAN-assisted (deterministic), content-asserted BOTH directions, ${lanLeg.runs}/${lanLeg.runs}`,
    lanLeg.ok === lanLeg.runs, JSON.stringify(lanLeg.bad).slice(0, 500))

  const swarmLeg = await ceremonyLeg('hyperswarm-only', false, 5)
  console.log(`  hyperswarm-only: MEASURED OK=${swarmLeg.ok}/${swarmLeg.runs} (live DHT -- environment-dependent, reported honestly, not gated pass/fail, same convention as bare-net-spike.mjs's own swarm leg)`)
  check(`positive: hyperswarm-only measured fraction recorded (OK=${swarmLeg.ok}/${swarmLeg.runs}, not gated pass/fail)`, true)

  console.log(`\n${checks} check(s), ${failures} failure(s).`)
  console.log(`LAN-assisted: OK=${lanLeg.ok}/${lanLeg.runs}   hyperswarm-only: OK=${swarmLeg.ok}/${swarmLeg.runs}`)
} catch (err) {
  check('gate did not throw', false, err?.message ?? String(err))
} finally {
  cleanup()
}

console.log(`\n${checks} check(s), ${failures} failure(s).`)
console.log(failures === 0 ? '\nSC3A CORRIDOR SPIKE GREEN' : `\nSC3A CORRIDOR SPIKE RED (${failures} failure(s))`)
process.exit(failures === 0 ? 0 : 1)
