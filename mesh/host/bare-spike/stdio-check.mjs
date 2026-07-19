// stdio-check.mjs — Band 4 step 1b: does Bare expose newline-delimited
// stdio the way the protocol-v0 framing needs (bridge-server.mjs's ndjson
// convention, adapted from TCP sockets to process stdin/stdout)?
import process from 'bare-process'

process.stdout.write(JSON.stringify({ event: 'ready' }) + '\n')

let buf = ''
process.stdin.on('data', (chunk) => {
  buf += chunk.toString('utf8')
  let idx
  while ((idx = buf.indexOf('\n')) !== -1) {
    const line = buf.slice(0, idx)
    buf = buf.slice(idx + 1)
    if (!line.trim()) continue
    process.stdout.write(JSON.stringify({ echoed: line }) + '\n')
  }
})
process.stdin.on('end', () => process.exit(0))
