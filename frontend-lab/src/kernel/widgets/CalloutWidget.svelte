<script lang="ts">
  /* Callout widget — a stack of toned advisory rows (overdue warnings,
   * reconciliation notes, "3 POs pending approval"). Static, no drill-down —
   * CalloutItem carries no nav (hub.ts); the tone wash IS the affordance. */
  import type { CalloutItem } from '../hub'

  let { items }: { items: CalloutItem[] } = $props()
</script>

<div class="k-callouts">
  {#each items as item, i (item.label + i)}
    <div
      class="k-callout"
      style:background="var(--k-tone-{item.tone}-bg)"
      style:border-inline-start-color="var(--k-tone-{item.tone}-fg)">
      <span class="k-callout-label" style:color="var(--k-tone-{item.tone}-fg)">{item.label}</span>
      <span class="k-callout-text">{item.text}</span>
    </div>
  {/each}
</div>

<style>
  .k-callouts {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-callout {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: var(--k-space-sm) var(--k-space-md);
    border-radius: var(--border-radius-sm);
    border-inline-start: 3px solid;
    min-width: 0;
  }
  .k-callout-label {
    font-size: calc(12px * var(--ui-font-scale));
    font-weight: 700;
  }
  .k-callout-text {
    font-size: calc(12px * var(--ui-font-scale));
    color: var(--text-secondary);
    overflow-wrap: anywhere;
  }
</style>
