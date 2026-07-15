<script lang="ts">
  /* OperationsHub — K5 tab-navigator over the built operations screens. The old
   * hub's per-tab badge COUNTS (open PO / DN / pending-fulfillment) are DEFERRED:
   * TabShell's `badge` prop supports them, but the counts need a light fetch the
   * hub doesn't own yet (K5 polish). Pure composition over built screens (embedded). */
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import TabShell from '$kernel/primitives/TabShell.svelte'
  import DocumentLedger from '$kernel/archetypes/DocumentLedger.svelte'
  import SerialTrace from './SerialTrace.svelte'
  import { purchaseOrdersDescriptor } from './purchase-orders.descriptor'
  import { deliveryNotesDescriptor } from './delivery-notes.descriptor'
  import { inventoryFulfillmentDescriptor } from './inventory-fulfillment.descriptor'
  import { currentRoute, routeTabOr } from '../stores/navigation.svelte'

  const TAB_KEYS = ['purchase-orders', 'delivery-notes', 'fulfillment', 'serials'] as const
  let active = $state(routeTabOr(TAB_KEYS, 'purchase-orders'))
  $effect(() => {
    const t = currentRoute().tab
    if (t && TAB_KEYS.includes(t as (typeof TAB_KEYS)[number])) active = t
  })
</script>

{#snippet t_pos()}<DocumentLedger descriptor={purchaseOrdersDescriptor} embedded />{/snippet}
{#snippet t_dns()}<DocumentLedger descriptor={deliveryNotesDescriptor} embedded />{/snippet}
{#snippet t_fulfillment()}<DocumentLedger descriptor={inventoryFulfillmentDescriptor} embedded />{/snippet}
{#snippet t_serials()}<SerialTrace embedded />{/snippet}

<PageShell title="Operations" subtitle="Purchase orders, delivery notes, fulfillment, and serial trace.">
  <TabShell
    activeKey={active}
    onSelect={(k) => (active = k)}
    tabs={[
      { key: 'purchase-orders', label: 'Purchase Orders', content: t_pos },
      { key: 'delivery-notes', label: 'Delivery Notes', content: t_dns },
      { key: 'fulfillment', label: 'Fulfillment', content: t_fulfillment },
      { key: 'serials', label: 'Serial Trace', content: t_serials },
    ]}
  />
</PageShell>
