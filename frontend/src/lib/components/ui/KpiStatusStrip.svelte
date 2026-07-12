<script lang="ts">
  export type KpiStatusItem = {
    label: string;
    value: string;
    meta?: string;
    status?: string;
    priority?: string;
  };

  interface Props {
    items: KpiStatusItem[];
  }

  let { items }: Props = $props();
</script>

<div class="kpi-status-strip">
  {#each items as item}
    <div class="kpi-status-item" data-status={item.status || ""} data-priority={item.priority || ""}>
      <span>{item.label}</span>
      <strong>{item.value}</strong>
      {#if item.meta}
        <small>{item.meta}</small>
      {/if}
    </div>
  {/each}
</div>

<style>
  .kpi-status-strip {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 1px;
    background: var(--border-subtle, #e5e1d8);
  }

  .kpi-status-item {
    min-width: 0;
    display: grid;
    gap: 4px;
    padding: 12px;
    background: var(--paper, #fff);
    box-shadow: inset 3px 0 0 transparent;
  }

  .kpi-status-item[data-status="ready"],
  .kpi-status-item[data-priority="low"] {
    box-shadow: inset 3px 0 0 #166534;
  }

  .kpi-status-item[data-status="review"],
  .kpi-status-item[data-priority="medium"],
  .kpi-status-item[data-priority="high"] {
    box-shadow: inset 3px 0 0 #b45309;
  }

  .kpi-status-item[data-status="blocked"],
  .kpi-status-item[data-status="critical"],
  .kpi-status-item[data-priority="critical"],
  .kpi-status-item[data-priority="urgent"] {
    box-shadow: inset 3px 0 0 #991b1b;
  }

  span {
    color: var(--ink-light, #666);
    font-size: 11px;
    text-transform: uppercase;
  }

  strong {
    min-width: 0;
    color: var(--ink, #1c1c1c);
    font-family: var(--font-mono, ui-monospace, SFMono-Regular, Consolas, monospace);
    font-size: 16px;
    overflow-wrap: anywhere;
  }

  small {
    min-width: 0;
    color: var(--ink-light, #666);
    font-size: 11px;
    overflow-wrap: anywhere;
  }

  @media (max-width: 860px) {
    .kpi-status-strip {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (max-width: 560px) {
    .kpi-status-strip {
      grid-template-columns: 1fr;
    }
  }
</style>
