<script lang="ts">
  /* L4 cell ejection (descriptor.ts's ColumnSpec.cell): the customer-select
   * dropdown for one scanned deal, sourced from that deal's own
   * customer_matches (never a shared/global customer list — each deal's
   * candidates differ). Mutates the row directly, same reactive contract as
   * OneDriveIncludeCell.svelte. */
  import type { ReviewDeal } from '../bridge/onedrive-import'

  let { row }: { row: ReviewDeal } = $props()
</script>

<select
  class="k-input"
  value={row.confirmedCustomerId}
  onchange={(e) => (row.confirmedCustomerId = e.currentTarget.value)}
>
  <option value="">— skip / unmatched —</option>
  {#each row.customerMatches as m (m.customerId)}
    <option value={m.customerId}>{m.businessName} ({m.score.toFixed(2)})</option>
  {/each}
</select>
