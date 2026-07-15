<script lang="ts">
  /* CRMHub — K5 tab-navigator over the CRM dashboards + data quality. The old
   * hub's drill-in (customer/supplier dashboard → a detail view) is preserved by
   * the dashboards' own KPI/widget `navigate` intents, which route (via the
   * shell navigation store) to the customer-360 / suppliers detail screens
   * rather than nesting — cleaner than the old replace-the-tab-bar UX. Pure
   * composition over built screens (embedded). */
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import TabShell from '$kernel/primitives/TabShell.svelte'
  import DocumentLedger from '$kernel/archetypes/DocumentLedger.svelte'
  import Hub from '$kernel/archetypes/Hub.svelte'
  import { crmCustomerHubDescriptor } from './dashboards/crm-customer.hub'
  import { crmSupplierHubDescriptor } from './dashboards/crm-supplier.hub'
  import { dataQualityDescriptor } from './data-quality.descriptor'
  import { navigate } from '../stores/navigation.svelte'
  import type { NavIntent } from '$kernel/hub'

  let active = $state('customers')
  const nav = (intent: NavIntent) => navigate(intent.key, intent.query ? { query: intent.query } : undefined)
</script>

{#snippet t_customers()}<Hub descriptor={crmCustomerHubDescriptor} navigate={nav} embedded />{/snippet}
{#snippet t_suppliers()}<Hub descriptor={crmSupplierHubDescriptor} navigate={nav} embedded />{/snippet}
{#snippet t_data_quality()}<DocumentLedger descriptor={dataQualityDescriptor} embedded />{/snippet}

<PageShell title="Relationships" subtitle="Customer and supplier overviews, plus data quality.">
  <TabShell
    activeKey={active}
    onSelect={(k) => (active = k)}
    tabs={[
      { key: 'customers', label: 'Customers', content: t_customers },
      { key: 'suppliers', label: 'Suppliers', content: t_suppliers },
      { key: 'data-quality', label: 'Data Quality', content: t_data_quality },
    ]}
  />
</PageShell>
