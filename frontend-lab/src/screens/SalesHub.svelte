<script lang="ts">
  /* SalesHub — K5 tab-navigator over the built sales screens. The old
   * "opportunities" tab (labeled "RFQs") rendered OpportunitiesScreen → maps to
   * the kernel `opportunities` ledger (the standalone `rfqs` K1 ledger stays its
   * own nav entry). The conditional SalesAdminTools tab (CanResolveOpportunity-
   * Conflicts gate) is DEFERRED. Pure composition over built screens (embedded). */
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import TabShell from '$kernel/primitives/TabShell.svelte'
  import DocumentLedger from '$kernel/archetypes/DocumentLedger.svelte'
  import CostingSheet from './CostingSheet.svelte'
  import { opportunitiesDescriptor } from './opportunities.descriptor'
  import { offersDescriptor } from './offers.descriptor'
  import { ordersDescriptor } from './orders.descriptor'
  import { currentRoute, routeTabOr } from '../stores/navigation.svelte'

  const TAB_KEYS = ['opportunities', 'costing', 'offers', 'orders'] as const
  let active = $state(routeTabOr(TAB_KEYS, 'opportunities'))
  $effect(() => {
    const t = currentRoute().tab
    if (t && TAB_KEYS.includes(t as (typeof TAB_KEYS)[number])) active = t
  })
</script>

{#snippet t_opportunities()}<DocumentLedger descriptor={opportunitiesDescriptor} embedded />{/snippet}
{#snippet t_costing()}<CostingSheet embedded />{/snippet}
{#snippet t_offers()}<DocumentLedger descriptor={offersDescriptor} embedded />{/snippet}
{#snippet t_orders()}<DocumentLedger descriptor={ordersDescriptor} embedded />{/snippet}

<PageShell title="Sales" subtitle="Opportunities, costing sheets, offers, and orders.">
  <TabShell
    activeKey={active}
    onSelect={(k) => (active = k)}
    tabs={[
      { key: 'opportunities', label: 'Opportunities', content: t_opportunities },
      { key: 'costing', label: 'Costing', content: t_costing },
      { key: 'offers', label: 'Offers', content: t_offers },
      { key: 'orders', label: 'Orders', content: t_orders },
    ]}
  />
</PageShell>
