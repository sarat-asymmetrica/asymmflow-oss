<script lang="ts" generics="Row">
  import type { ColumnSpec, StatusSpec } from '../descriptor'
  import { renderCell, cellAlign, cellFontClass } from '../content'
  import Badge from '../controls/Badge.svelte'
  import type { Tone } from '../controls/Badge.svelte'

  let {
    columns,
    rows,
    id,
    status,
    selectedId = null,
    onSelect,
  }: {
    columns: ColumnSpec<Row>[]
    rows: Row[]
    id: (row: Row) => string
    status?: StatusSpec<Row> | undefined
    selectedId?: string | null
    onSelect?: (row: Row) => void
  } = $props()

  function toneFor(row: Row): Tone {
    if (!status) return 'neutral'
    return status.tones[status.value(row)] ?? 'neutral'
  }

  // The table is never narrower than the sum of declared column minimums —
  // below that it scrolls inside k-table-wrap instead of crushing a column
  // to zero (fixed layout would otherwise collapse the grow column first).
  const totalMinWidth = $derived(columns.reduce((sum, c) => sum + (c.minWidth ?? 140), 0))
</script>

<!-- Layout doctrine: the table owns overflow-x internally; it can never
     widen the page. table-layout:fixed makes truncation predictable. -->
<div class="k-table-wrap">
  <table class="k-table" style:min-width="{totalMinWidth}px">
    <colgroup>
      {#each columns as col (col.key)}
        <col style:width={col.grow ? 'auto' : `${col.minWidth ?? 140}px`} />
      {/each}
    </colgroup>
    <thead>
      <tr>
        {#each columns as col (col.key)}
          <th class="k-th" style:text-align={cellAlign[col.content]}>{col.label}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each rows as row (id(row))}
        <tr
          class="k-tr"
          class:selected={selectedId === id(row)}
          class:clickable={!!onSelect}
          onclick={() => onSelect?.(row)}
        >
          {#each columns as col (col.key)}
            {#if col.cell}
              {@const Cell = col.cell}
              <td class="k-td"><Cell {row} /></td>
            {:else if col.content === 'status' && status}
              <td class="k-td">
                <Badge tone={toneFor(row)} label={String(col.value(row) ?? '—')} />
              </td>
            {:else}
              {@const text = renderCell(col.content, col.value(row), col.currency?.(row))}
              <td
                class="k-td {cellFontClass(col.content)}"
                style:text-align={cellAlign[col.content]}
                title={text}
              >
                {text}
              </td>
            {/if}
          {/each}
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<style>
  .k-table-wrap {
    overflow-x: auto;
    min-width: 0;
  }
  .k-table {
    width: 100%;
    table-layout: fixed;
    border-collapse: collapse;
    font-size: var(--table-text-size);
  }
  .k-th {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    letter-spacing: 0.02em;
    text-transform: uppercase;
    color: var(--text-secondary);
    padding: 10px 12px;
    border-bottom: var(--border-width) solid var(--border);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .k-td {
    height: var(--table-row-height);
    padding: 0 12px;
    border-bottom: var(--border-width) solid var(--border);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .k-cell-numeric {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
  }
  .k-tr.clickable {
    cursor: pointer;
  }
  .k-tr.clickable:hover {
    background: var(--onyx-tint);
  }
  .k-tr.selected {
    background: var(--onyx-tint-medium);
  }
</style>
