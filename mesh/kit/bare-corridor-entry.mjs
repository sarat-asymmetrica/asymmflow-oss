// bare-corridor-entry.mjs — SC-2 gate harness: a packable, headless, Bare-
// sealed entry that lets bare-net-spike.mjs drive TWO real sealed-kit
// processes through a scripted network-replication ceremony, over stdin,
// with machine-greppable stdout lines. This gates the NETWORK MODULE
// (bare-net.mjs) at the sealed-artifact layer — it is NOT the client-facing
// guide/ceremony (that is SC-3's job, wiring bare-guide.mjs's menu). Do not
// read a green run of this file as proof of the client's onboarding path.
//
// SCOPE, declared: this harness deliberately bypasses the invite/capability
// ceremony (social-room.mjs, capability.mjs, invite-code.mjs) — that
// machinery is real, already Bare-proven (bare-bridge.mjs), and is SC-3's
// concern, not this mission's. What SC-2 owns is strictly lower-level: can
// two Autobase nodes, on two sealed Bare processes, replicate a room's op
// log over (a) the TCP fallback and (b) hyperswarm? Proving that needs only
// mesh-node.mjs's own primitives — `createMeshNode`, `node.addWriter()`
// (Autobase's own writer-admission primitive, unconditionally required for
// a second writer's ops to be admitted at all, independent of the
// capability plane), and `node.append()`/`node.state()`. Rooms created here
// carry NO authorityPub, so the reducer folds them UNENFORCED (capability
// checks skipped) — verified against reducer/room_domain.go: `applyPost`
// requires neither a signature nor a manifest to accept a `msg.post` op.
// `mesh/reducer/**` is untouched; this is a scope choice in the HARNESS,
// not a change to the reducer's law.
//
// WASM ASSET INJECTION — copied verbatim from kit/bare-guide-entry.mjs (read
// that file's own header before touching this block): apply-bare.mjs's
// default self-locating read resolves to a virtual, unreadable path once
// packed. `import.meta.asset()` is the only form bare-pack's static asset
// detector recognises; omitting this makes a kit that boots, prints
// everything, and silently cannot post.
import * as fs from 'bare-fs'
import { setWasmSource } from '../host/apply-bare.mjs'
import { createMeshNode, waitFor } from '../host/mesh-node.mjs'
import { createNetwork } from './bare-net.mjs'
import { getRealStdio } from '../host/bare-bridge.mjs'
import { createGuideIO } from './bare-guide.mjs'

const wasmAssetPath = import.meta.asset('../dist/reducer.wasm')
setWasmSource(fs.readFileSync(new URL(wasmAssetPath)))

const DATA_DIR = './data'

function seqOfMsgId(msgId) {
  const i = typeof msgId === 'string' ? msgId.lastIndexOf(':') : -1
  return i === -1 ? 0 : Number(msgId.slice(i + 1))
}

async function nextSeqFor(node) {
  const ops = await node.ops()
  let max = 0
  for (const op of ops) if (Number.isInteger(op.seq) && op.seq > max) max = op.seq
  return max + 1
}

/** runCorridor({ io }) — io defaults to the real stdio (getRealStdio()),
 * matching bare-guide.mjs's own runGuide() shape so a caller (a future
 * direct-import test) can inject a fake io without this function owning —
 * or killing — the real process. */
export async function runCorridor({ io } = {}) {
  const realIo = io ?? await getRealStdio()
  // NOT `realIo.write(s + '\n')` — `write(str)` (getRealStdio's real
  // implementation) is `console.log(str)`, which already appends its own
  // trailing newline (RULE 2, bare-bridge.mjs's own header); adding a
  // second one here would double every line. Same convention bare-guide.mjs
  // uses throughout.
  const write = (s) => realIo.write(s)
  const cmdIo = createGuideIO(realIo)

  let net = null
  let node = null
  let actor = 'corridor'

  write('CORRIDOR READY')

  for (;;) {
    const raw = await cmdIo.ask('')
    if (raw === null) break
    const line = raw.trim()
    if (line === '') continue
    const sp = line.indexOf(' ')
    const cmd = (sp === -1 ? line : line.slice(0, sp)).toUpperCase()
    const rest = sp === -1 ? '' : line.slice(sp + 1)

    try {
      if (cmd === 'ACTOR') {
        actor = rest.trim() || 'corridor'
        write(`ACTOR ${actor}`)
      } else if (cmd === 'NET') {
        // NET <useHyperswarm 0|1>
        const useHyperswarm = rest.trim() !== '0'
        net = createNetwork({ useHyperswarm })
        write(`NET ${useHyperswarm ? 'hyperswarm+tcp' : 'tcp-only'}`)
      } else if (cmd === 'CREATE') {
        node = await createMeshNode({ storage: `${DATA_DIR}/corestore/founder`, mode: 'room' })
        write(`ROOMKEY ${node.key}`)
      } else if (cmd === 'JOIN') {
        const roomKey = rest.trim()
        node = await createMeshNode({
          storage: `${DATA_DIR}/corestore/joined-${roomKey.slice(0, 16)}`,
          bootstrap: roomKey, mode: 'room',
        })
        write(`WRITERKEY ${node.writerKey}`)
      } else if (cmd === 'LISTEN') {
        const port = Number(rest.trim()) || 0
        const bound = await net.listenTcp(port, node)
        write(`LISTENING ${bound}`)
      } else if (cmd === 'CONNECT') {
        const [host, portStr] = rest.trim().split(' ')
        await net.connectTcp(host, Number(portStr), node)
        write('CONNECTED')
      } else if (cmd === 'JOINSWARM') {
        const ok = net.joinHyperswarm(node.key, node)
        write(ok ? 'JOINEDSWARM' : 'JOINSWARMFAILED')
      } else if (cmd === 'ADDWRITER') {
        await node.addWriter(rest.trim())
        write('ADDEDWRITER')
      } else if (cmd === 'WAITWRITABLE') {
        const timeout = Number(rest.trim()) || 15000
        try {
          await waitFor(async () => { await node.base.update(); return node.writable }, { timeout, label: 'corridor writable' })
          write('WRITABLE')
        } catch (err) {
          write(`NOTWRITABLE ${err.message}`)
        }
      } else if (cmd === 'POST') {
        const seq = await nextSeqFor(node)
        await node.append({ seq, actor, ts: Date.now(), kind: 'msg.post', body: rest, expectation: '' })
        write(`POSTED ${seq}`)
      } else if (cmd === 'LIST') {
        const state = await node.state()
        write('MSGSTART')
        for (const m of state.messages ?? []) {
          write(`MSG|${seqOfMsgId(m.msgId)}|${m.actor}|${m.body ?? ''}`)
        }
        write('MSGEND')
      } else if (cmd === 'PEERCOUNT') {
        write(`PEERCOUNT ${net.peerCount(node.key)}`)
      } else if (cmd === 'QUIT') {
        break
      } else {
        write(`UNKNOWNCOMMAND ${cmd}`)
      }
    } catch (err) {
      write(`ERROR ${err?.message ?? String(err)}`)
    }
  }

  cmdIo.close()
  if (node) { try { await node.close() } catch { /* best-effort */ } }
  if (net) { try { await net.close() } catch { /* best-effort */ } }
  write('BYE')

  // RULE 3 (mesh/docs/bare-campaign — binding across this whole codebase):
  // an explicit exit call is load-bearing, never relied-upon-natural-drain.
  // Same guard as bare-guide.mjs's own runGuide(): only fires when this
  // function owns the real stdio it was given.
  if (!io) {
    if (typeof Bare !== 'undefined') Bare.exit(0)
    else process.exit(0)
  }
}

await runCorridor()
