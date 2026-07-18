import { describe, it, expect } from 'vitest'
import { escapeHtml, renderMarkdown } from '../src/kernel/markdown'

describe('escapeHtml', () => {
  it('neutralizes all HTML metacharacters', () => {
    expect(escapeHtml(`<script>"&'`)).toBe('&lt;script&gt;&quot;&amp;&#39;')
  })
})

describe('renderMarkdown — security (escape-first)', () => {
  it('never emits a live script tag from model output', () => {
    const html = renderMarkdown('<script>alert(1)</script>')
    expect(html).not.toContain('<script>')
    expect(html).toContain('&lt;script&gt;')
  })
  it('escapes html inside a bold span (capture group is pre-escaped)', () => {
    const html = renderMarkdown('**<img src=x onerror=alert(1)>**')
    expect(html).toContain('<strong>')
    expect(html).not.toContain('<img')
    expect(html).toContain('&lt;img')
  })
  it('escapes html inside table cells', () => {
    const html = renderMarkdown('| a | b |\n| --- | --- |\n| <b>x</b> | y |')
    expect(html).toContain('<table')
    expect(html).not.toContain('<b>x</b>')
    expect(html).toContain('&lt;b&gt;x&lt;/b&gt;')
  })
})

describe('renderMarkdown — structure', () => {
  it('renders headings', () => {
    expect(renderMarkdown('## Title')).toContain('<h4 class="k-md-h">Title</h4>')
  })
  it('renders bullet lists', () => {
    const html = renderMarkdown('- one\n- two')
    expect(html).toContain('<ul class="k-md-ul">')
    expect(html).toContain('<li>one</li>')
    expect(html).toContain('<li>two</li>')
    expect(html).toContain('</ul>')
  })
  it('renders ordered lists', () => {
    const html = renderMarkdown('1. first\n2) second')
    expect(html).toContain('<ol class="k-md-ol">')
    expect(html).toContain('<li>first</li>')
    expect(html).toContain('<li>second</li>')
  })
  it('renders a GFM table with header and body', () => {
    const html = renderMarkdown('| Name | Qty |\n|------|-----|\n| Widget | 3 |')
    expect(html).toContain('<th>Name</th>')
    expect(html).toContain('<th>Qty</th>')
    expect(html).toContain('<td>Widget</td>')
    expect(html).toContain('<td>3</td>')
  })
  it('renders inline bold and code', () => {
    const html = renderMarkdown('a **bold** and `code` here')
    expect(html).toContain('<strong>bold</strong>')
    expect(html).toContain('<code>code</code>')
  })
  it('renders paragraphs and tolerates blank lines', () => {
    const html = renderMarkdown('para one\n\npara two')
    expect(html).toContain('<p class="k-md-p">para one</p>')
    expect(html).toContain('<p class="k-md-p">para two</p>')
  })
  it('handles a ragged table row without crashing', () => {
    const html = renderMarkdown('| a | b | c |\n|---|---|---|\n| only-one |')
    expect(html).toContain('<table')
    expect(html).toContain('<td>only-one</td>')
  })
})
