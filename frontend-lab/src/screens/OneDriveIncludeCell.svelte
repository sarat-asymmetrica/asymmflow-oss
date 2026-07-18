<script lang="ts">
  /* L4 cell ejection (descriptor.ts's ColumnSpec.cell): the "include this
   * deal in the import" checkbox. Mutates the row directly — vm.deals is
   * $state-backed (onedrive-import-vm.svelte.ts), so this reactively updates
   * the wizard's includedDeals/canAdvance/nextLabel derivations. */
  import type { ReviewDeal } from '../bridge/onedrive-import'

  let { row }: { row: ReviewDeal } = $props()
</script>

<div class="od-include-wrap">
  <input
    type="checkbox"
    checked={row.selected}
    onchange={() => (row.selected = !row.selected)}
    aria-label="Include {row.folderName.trim() || row.localId} in the import"
  />
</div>

<style>
  /* The td DataTable wraps this cell in has no declared text-align for
   * col.cell columns (only the plain-text branch gets one) — center the
   * checkbox within its own wrapper instead of fighting the table. */
  .od-include-wrap {
    text-align: center;
  }
</style>
