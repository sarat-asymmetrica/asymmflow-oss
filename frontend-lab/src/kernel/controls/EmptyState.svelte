<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    message,
    actions,
  }: {
    message: string
    actions?: Snippet
  } = $props()
</script>

<div class="k-empty">
  <p class="k-empty-msg">{message}</p>
  {#if actions}
    <div class="k-empty-actions">{@render actions()}</div>
  {/if}
</div>

<style>
  .k-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-md);
    padding: var(--k-space-xl);
    text-align: center;
    min-width: 0;
  }
  .k-empty-msg {
    color: var(--text-secondary);
    font-size: calc(13px * var(--ui-font-scale));
    max-width: 48ch;
    overflow-wrap: break-word;
  }
</style>
