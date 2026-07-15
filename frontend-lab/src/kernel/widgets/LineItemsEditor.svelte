<script lang="ts" generics="Row">
  /* LineItemsEditor — the kernel's repeating editable-row widget. Renders a
   * declared column config as an aligned grid of inputs + a live footer, and
   * plumbs every edit back to the caller via each column's set(). It computes
   * NOTHING itself (L5): line math, totals, and balance live in the consuming
   * viewmodel — this widget only reads already-computed values and forwards
   * edits. Serves the accounting journal voucher (account/debit/credit +
   * balanced badge) and the CostingSheet pricing waterfall (wide product
   * fields + readonly computed cells) from one component.
   *
   * Layout doctrine: the row grid can exceed the container on wide configs, so
   * the scroll region carries the detector-whitelisted `k-table-wrap` class —
   * horizontal scroll is declared, never an accidental page blow-out (L1). */
  import type { LineColumn, LineFooterCell, LineBalanceCheck } from '../line-items'
  import { renderCell, cellAlign } from '../content'
  import type { ContentClass } from '../descriptor'
  import Button from '../controls/Button.svelte'

  let {
    columns,
    rows,
    createRow,
    onAdd,
    onRemove,
    minRows = 0,
    maxRows,
    footer,
    balance,
    disabled = false,
    addLabel = '+ Add line',
    emptyMessage = 'No lines yet.',
  }: {
    columns: LineColumn<Row>[]
    /** The SAME reactive array the caller's VM owns — cells write through set(). */
    rows: Row[]
    createRow: () => Row
    /** Called after a new row should be appended (caller mutates `rows`). */
    onAdd: () => void
    /** Called to remove the row at `index` (caller mutates `rows`). */
    onRemove: (index: number) => void
    minRows?: number
    maxRows?: number
    footer?: LineFooterCell<Row>[]
    balance?: LineBalanceCheck<Row>
    disabled?: boolean
    addLabel?: string
    emptyMessage?: string
  } = $props()

  const mainCols = $derived(columns.filter((c) => !c.wide))
  const wideCols = $derived(columns.filter((c) => c.wide))

  // Grid template: fixed px per column, one optional grower, + a remove column.
  const template = $derived(
    mainCols.map((c) => (c.grow ? 'minmax(140px, 1fr)' : `${c.minWidth ?? 120}px`)).join(' ') + ' 40px',
  )
  const totalMin = $derived(mainCols.reduce((s, c) => s + (c.minWidth ?? 120), 0) + 40)

  const canAdd = $derived(!disabled && (maxRows == null || rows.length < maxRows))
  const balanced = $derived(balance ? balance.isBalanced(rows) : true)

  // Default readonly/footer formatting per kind when `content` is unset.
  function contentFor<R>(col: LineColumn<R>): ContentClass {
    if (col.content) return col.content
    switch (col.kind) {
      case 'money':
        return 'money'
      case 'number':
      case 'percent':
        return 'quantity'
      default:
        return 'text'
    }
  }

  function alignFor<R>(col: LineColumn<R>): 'start' | 'end' {
    if (col.align) return col.align
    if (col.kind === 'number' || col.kind === 'money' || col.kind === 'percent') return 'end'
    return cellAlign[contentFor(col)]
  }

  function granularity<R>(col: LineColumn<R>): 'input' | 'change' {
    return col.eventGranularity ?? (col.kind === 'text' || col.kind === 'textarea' ? 'input' : 'change')
  }

  function isNumeric<R>(col: LineColumn<R>): boolean {
    return col.kind === 'number' || col.kind === 'money' || col.kind === 'percent'
  }

  function write(col: LineColumn<Row>, row: Row, raw: string) {
    if (!col.set) return
    col.set(row, isNumeric(col) ? Number(raw) || 0 : raw)
  }

  function inputStr(col: LineColumn<Row>, row: Row): string {
    const v = col.value(row)
    return v == null ? '' : String(v)
  }
</script>

<div class="k-lie">
  <div class="k-table-wrap">
    <div class="k-lie-grid" style:min-width="{totalMin}px">
      <!-- header -->
      <div class="k-lie-head" style:grid-template-columns={template}>
        {#each mainCols as col (col.key)}
          <span class="k-lie-th" style:text-align={alignFor(col)} title={col.label}>{col.label}</span>
        {/each}
        <span class="k-lie-th"></span>
      </div>

      {#if rows.length === 0}
        <div class="k-lie-empty">{emptyMessage}</div>
      {/if}

      {#each rows as row, i (i)}
        <div class="k-lie-rowwrap">
          <div class="k-lie-row" style:grid-template-columns={template}>
            {#each mainCols as col (col.key)}
              {@const tone = col.tone?.(row)}
              <div class="k-lie-cell" style:text-align={alignFor(col)}>
                {#if col.cell}
                  {@const Cell = col.cell}
                  <Cell {row} onInput={(v) => col.set?.(row, v)} />
                {:else if col.kind === 'readonly'}
                  <span
                    class="k-lie-ro"
                    class:k-lie-numeric={isNumeric(col) || contentFor(col) === 'money' || contentFor(col) === 'quantity'}
                    style:color={tone ? `var(--k-tone-${tone}-fg)` : undefined}
                    style:font-weight={tone ? 600 : undefined}
                    title={renderCell(contentFor(col), col.value(row), col.currency?.(row))}
                  >
                    {renderCell(contentFor(col), col.value(row), col.currency?.(row))}
                  </span>
                {:else if col.kind === 'select'}
                  <select
                    class="k-lie-input"
                    {disabled}
                    value={inputStr(col, row)}
                    onchange={(e) => write(col, row, e.currentTarget.value)}
                  >
                    <option value="">{col.placeholder?.(row) ?? 'Select…'}</option>
                    {#each col.options?.(row) ?? [] as opt (opt.value)}
                      <option value={opt.value}>{opt.label}</option>
                    {/each}
                  </select>
                {:else if col.kind === 'textarea'}
                  <textarea
                    class="k-lie-input k-lie-area"
                    {disabled}
                    placeholder={col.placeholder?.(row) ?? ''}
                    value={inputStr(col, row)}
                    oninput={granularity(col) === 'input'
                      ? (e) => write(col, row, e.currentTarget.value)
                      : undefined}
                    onchange={granularity(col) === 'change'
                      ? (e) => write(col, row, e.currentTarget.value)
                      : undefined}
                  ></textarea>
                {:else}
                  <input
                    class="k-lie-input"
                    class:k-lie-numeric={isNumeric(col)}
                    type={isNumeric(col) ? 'number' : 'text'}
                    step={col.step}
                    {disabled}
                    style:color={tone ? `var(--k-tone-${tone}-fg)` : undefined}
                    placeholder={col.placeholder?.(row) ?? ''}
                    value={inputStr(col, row)}
                    oninput={granularity(col) === 'input'
                      ? (e) => write(col, row, e.currentTarget.value)
                      : undefined}
                    onchange={granularity(col) === 'change'
                      ? (e) => write(col, row, e.currentTarget.value)
                      : undefined}
                  />
                {/if}
              </div>
            {/each}
            <div class="k-lie-cell k-lie-remove-cell">
              <button
                class="k-lie-remove"
                aria-label="Remove line"
                disabled={disabled || rows.length <= minRows}
                onclick={() => onRemove(i)}
              >×</button>
            </div>
          </div>

          {#if wideCols.length}
            <div class="k-lie-wide">
              {#each wideCols as col (col.key)}
                <label class="k-lie-wide-field">
                  <span class="k-lie-wide-label">{col.label}</span>
                  {#if col.kind === 'textarea'}
                    <textarea
                      class="k-lie-input k-lie-area"
                      {disabled}
                      placeholder={col.placeholder?.(row) ?? ''}
                      value={inputStr(col, row)}
                      oninput={(e) => write(col, row, e.currentTarget.value)}
                    ></textarea>
                  {:else}
                    <input
                      class="k-lie-input"
                      type="text"
                      {disabled}
                      placeholder={col.placeholder?.(row) ?? ''}
                      value={inputStr(col, row)}
                      oninput={(e) => write(col, row, e.currentTarget.value)}
                    />
                  {/if}
                </label>
              {/each}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  </div>

  <div class="k-lie-actions">
    <Button onclick={onAdd} disabled={!canAdd}>{addLabel}</Button>
    {#if maxRows != null}
      <span class="k-lie-count">{rows.length}{maxRows ? ` / ${maxRows}` : ''}</span>
    {/if}
  </div>

  {#if footer?.length || balance}
    <div class="k-lie-footer">
      {#if footer?.length}
        {#each footer as fc (fc.label)}
          {@const tone = fc.tone?.(rows)}
          <div class="k-lie-foot-cell">
            <span class="k-lie-foot-label">{fc.label}</span>
            <span
              class="k-lie-foot-value k-lie-numeric"
              style:color={tone ? `var(--k-tone-${tone}-fg)` : undefined}
            >{renderCell(fc.content ?? 'money', fc.value(rows), fc.currency)}</span>
          </div>
        {/each}
      {/if}
      {#if balance}
        <span
          class="k-lie-balance"
          style:background={`var(--k-tone-${balanced ? 'success' : 'danger'}-bg)`}
          style:color={`var(--k-tone-${balanced ? 'success' : 'danger'}-fg)`}
        >{balanced ? (balance.balancedLabel ?? 'Balanced') : (balance.unbalancedLabel ?? 'Unbalanced')}</span>
      {/if}
    </div>
  {/if}
</div>

<style>
  .k-lie {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-table-wrap {
    overflow-x: auto;
    min-width: 0;
  }
  .k-lie-grid {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-xs);
  }
  .k-lie-head,
  .k-lie-row {
    display: grid;
    gap: var(--k-space-sm);
    align-items: center;
    min-width: 0;
  }
  .k-lie-th {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.02em;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .k-lie-cell {
    min-width: 0;
  }
  .k-lie-input {
    width: 100%;
    min-width: 0;
    font-family: var(--font-ui);
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    padding: 6px 8px;
    outline: none;
    transition: border-color var(--motion-fast) var(--ease-standard);
  }
  .k-lie-input:focus {
    border-color: var(--onyx);
  }
  .k-lie-area {
    min-height: 44px;
    resize: vertical;
  }
  .k-lie-numeric {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    text-align: right;
  }
  .k-lie-ro {
    display: block;
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .k-lie-remove-cell {
    display: flex;
    justify-content: center;
  }
  .k-lie-remove {
    width: 26px;
    height: 26px;
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    background: var(--surface);
    color: var(--text-secondary);
    font-size: 16px;
    line-height: 1;
    cursor: pointer;
  }
  .k-lie-remove:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  .k-lie-wide {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-xs);
    padding: 0 var(--k-space-sm) var(--k-space-sm);
  }
  .k-lie-wide-field {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .k-lie-wide-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.02em;
    color: var(--text-muted);
  }
  .k-lie-empty {
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-secondary);
    padding: var(--k-space-md) 0;
  }
  .k-lie-actions {
    display: flex;
    align-items: center;
    gap: var(--k-space-sm);
  }
  .k-lie-count {
    font-size: var(--meta-size);
    color: var(--text-muted);
  }
  .k-lie-footer {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: flex-end;
    gap: var(--k-space-md);
    padding-top: var(--k-space-sm);
    border-top: var(--border-width) solid var(--border);
    min-width: 0;
  }
  .k-lie-foot-cell {
    display: flex;
    align-items: baseline;
    gap: var(--k-space-sm);
  }
  .k-lie-foot-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.02em;
    color: var(--text-secondary);
  }
  .k-lie-foot-value {
    font-size: calc(14px * var(--ui-font-scale));
    font-weight: 700;
    color: var(--text-primary);
  }
  .k-lie-balance {
    font-size: calc(13px * var(--ui-font-scale));
    font-weight: 700;
    padding: 4px 12px;
    border-radius: var(--border-radius-pill);
    white-space: nowrap;
  }
</style>
