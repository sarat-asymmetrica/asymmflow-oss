<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    header?: Snippet;
    children?: Snippet;
  }

  let { header, children }: Props = $props();
  // Simple flexible table wrapper
</script>

<div class="table-container">
  <table class="data-table">
    {@render header?.()}
    {@render children?.()}
  </table>
</div>
