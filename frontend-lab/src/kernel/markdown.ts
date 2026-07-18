/* Minimal, dependency-free markdown → safe HTML for the chat transcript.
 * Ported in spirit from the old ButlerScreen's hand-rolled formatter, but with
 * the security order made explicit and load-bearing: EVERY input character is
 * HTML-escaped FIRST, THEN a small, closed set of block/inline structures is
 * re-introduced from the escaped text. Because escaping runs before any tag is
 * emitted, no substring of the model's output can ever become live markup —
 * the renderer can only ever emit the fixed tag vocabulary below. This is the
 * kernel's stance: chat responses are untrusted (LLM/Mistral) content rendered
 * via {@html}, so the escape-first invariant is a security boundary, not a
 * style choice. No marked/remark/commonmark dependency (kernel no-dep posture).
 *
 * Supported: #..###### headings, - / * / • bullets, 1. / 1) ordered lists,
 * **bold**, `code`, GFM pipe tables, blank-line paragraph breaks. Everything
 * else renders as escaped plain text. */

const H_TAG: Record<number, string> = { 1: 'h3', 2: 'h4', 3: 'h5', 4: 'h6', 5: 'h6', 6: 'h6' }

export function escapeHtml(input: string): string {
  return String(input ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

/** Inline formatting applied to ALREADY-ESCAPED text. The `*` and `` ` ``
 * characters survive escaping, so these regexes are safe: their capture groups
 * are escaped content, and only fixed <strong>/<code> tags are added. */
function inline(escaped: string): string {
  return escaped
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
}

function isTableSeparator(line: string): boolean {
  // | --- | :--: | style row (dashes/colons/pipes/spaces only, ≥1 dash).
  const t = line.trim()
  return /\|/.test(t) && /-/.test(t) && /^\|?[\s:|-]+\|?$/.test(t)
}

function splitRow(line: string): string[] {
  let t = line.trim()
  if (t.startsWith('|')) t = t.slice(1)
  if (t.endsWith('|')) t = t.slice(0, -1)
  return t.split('|').map((c) => c.trim())
}

/** Render untrusted markdown text to a safe HTML string. */
export function renderMarkdown(text: string): string {
  const rawLines = String(text ?? '').split(/\r?\n/)
  const out: string[] = []
  let listType: 'ul' | 'ol' | null = null

  const closeList = () => {
    if (listType) {
      out.push(`</${listType}>`)
      listType = null
    }
  }

  for (let i = 0; i < rawLines.length; i++) {
    const line = rawLines[i]!
    const trimmed = line.trim()

    // GFM table: a pipe header line immediately followed by a separator line.
    if (/\|/.test(trimmed) && i + 1 < rawLines.length && isTableSeparator(rawLines[i + 1]!)) {
      closeList()
      const headers = splitRow(line)
      i += 2 // skip header + separator
      const bodyRows: string[][] = []
      while (i < rawLines.length && /\|/.test(rawLines[i]!.trim()) && rawLines[i]!.trim() !== '') {
        bodyRows.push(splitRow(rawLines[i]!))
        i++
      }
      i-- // for-loop will ++
      const thead = headers.map((h) => `<th>${inline(escapeHtml(h))}</th>`).join('')
      const tbody = bodyRows
        .map((r) => `<tr>${r.map((c) => `<td>${inline(escapeHtml(c))}</td>`).join('')}</tr>`)
        .join('')
      out.push(`<table class="k-md-table"><thead><tr>${thead}</tr></thead><tbody>${tbody}</tbody></table>`)
      continue
    }

    if (trimmed === '') {
      closeList()
      continue
    }

    const heading = /^(#{1,6})\s+(.*)$/.exec(trimmed)
    if (heading) {
      closeList()
      const tag = H_TAG[heading[1]!.length]!
      out.push(`<${tag} class="k-md-h">${inline(escapeHtml(heading[2]!))}</${tag}>`)
      continue
    }

    const bullet = /^[-*•]\s+(.*)$/.exec(trimmed)
    if (bullet) {
      if (listType !== 'ul') {
        closeList()
        out.push('<ul class="k-md-ul">')
        listType = 'ul'
      }
      out.push(`<li>${inline(escapeHtml(bullet[1]!))}</li>`)
      continue
    }

    const numbered = /^\d+[.)]\s+(.*)$/.exec(trimmed)
    if (numbered) {
      if (listType !== 'ol') {
        closeList()
        out.push('<ol class="k-md-ol">')
        listType = 'ol'
      }
      out.push(`<li>${inline(escapeHtml(numbered[1]!))}</li>`)
      continue
    }

    closeList()
    out.push(`<p class="k-md-p">${inline(escapeHtml(trimmed))}</p>`)
  }

  closeList()
  return out.join('')
}
