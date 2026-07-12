<script lang="ts">
  import { run } from 'svelte/legacy';

  import { onMount, onDestroy, tick } from 'svelte';
  import ModuleLayout from '$lib/components/layout/ModuleLayout.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import ErrorBoundary from '$lib/components/ErrorBoundary.svelte';
  import PurchaseOrdersScreen from './PurchaseOrdersScreen.svelte';
  // GRNScreen deprecated per user request - not needed
  // Wave 9.2 B2/B7: Supplier Invoices moved to the Finance hub (AP loop lives in
  // one home beside Supplier Payments) — removed from Operations to avoid a
  // duplicate path (Design Constitution III.1).
  import DeliveryNotesScreen from './DeliveryNotesScreen.svelte';
  import SerialTraceScreen from './SerialTraceScreen.svelte';
  import InventoryFulfillmentScreen from './InventoryFulfillmentScreen.svelte';
  import type { Tab } from '$lib/types/components';
  import { GetPurchaseOrders, GetInventoryPendingFulfillmentReport } from '../../../wailsjs/go/main/App';
import { GetDeliveryNotes } from '../../../wailsjs/go/main/CRMService';
  import { toast } from '$lib/stores/toasts';
  import { pendingDNCreate, pendingOrderView } from '$lib/stores/navigation';

  
  interface Props {
    // Props
    embedded?: boolean;
  }

  let { embedded = false }: Props = $props();
  run(() => {
    embedded;
  });

  // Tab state - GRNs removed per user request (deprecated)
  // Operations = supplier-side online orders, supplier invoices, delivery notes to customers
  const tabs: Tab[] = $state([
    { id: 'pos', label: 'Supplier Orders', count: 0 },           // Supplier-side online orders
    { id: 'delivery-notes', label: 'Delivery Notes', count: 0 },        // DNs TO customers (links to customer invoices)
    { id: 'fulfillment', label: 'Fulfillment', count: 0 },              // Sold but not yet delivered/invoiced
    { id: 'serials', label: 'Serials', count: 0 },                      // Serial lifecycle traceability (read-only search)
  ]);

  let activeTab = $state('pos');
  let loading = true;

  // Load counts for each tab (GRNs removed)
  async function loadCounts() {
    loading = true;
    try {
      const [pos, deliveryNotes, pendingFulfillment] = await Promise.all([
        GetPurchaseOrders().catch(() => []),
        GetDeliveryNotes().catch(() => []),
        GetInventoryPendingFulfillmentReport(500).catch(() => []),
      ]);

      // Update tab counts
      tabs[0].count = pos?.length || 0;
      tabs[1].count = deliveryNotes?.length || 0;
      tabs[2].count = pendingFulfillment?.length || 0;
    } catch (err) {
      console.error('Failed to load counts:', err);
      toast.warning('Failed to load tab counts');
    } finally {
      loading = false;
    }
  }

  function handleTabChange(e: CustomEvent<string>) {
    activeTab = e.detail;
  }

  // Header actions based on active tab - switch tabs and dispatch modal events
  async function handleNewPO() {
    activeTab = 'pos';
    await tick();
    // Delay ensures child component has mounted and registered listeners
    setTimeout(() => {
      window.dispatchEvent(new CustomEvent('openCreatePO'));
    }, 200);
  }

  // GRN deprecated - function removed

  async function handleNewDN() {
    activeTab = 'delivery-notes';
    await tick();
    // Delay ensures child component has mounted and registered listeners
    setTimeout(() => {
      window.dispatchEvent(new CustomEvent('openCreateDN'));
    }, 200);
  }

  // Cross-screen navigation handlers (GRN removed)
  function handleNavigateToScreen(e) {
    const { tab } = e.detail;
    if (tab) activeTab = tab;
  }

  onMount(() => {
    window.addEventListener('navigateToScreen', handleNavigateToScreen);
    loadCounts();

    // If there's a pending DN creation, switch to delivery-notes tab
    if ($pendingDNCreate) {
      activeTab = 'delivery-notes';
    }

    // B1b: OperationsHub has no Orders tab — OrdersScreen actually lives under
    // the Sales hub (App.svelte screen id "opportunities" loads SalesHub.svelte,
    // whose "orders" tab hosts OrdersScreen). If a pendingOrderView handoff (set
    // by OffersScreen after MarkOfferWon) lands us on Operations anyway, forward
    // the user to where the order detail can actually open instead of leaving
    // the store set with nowhere to consume it.
    if ($pendingOrderView) {
      window.dispatchEvent(new CustomEvent('navigateToScreen', {
        detail: { screen: 'opportunities', tab: 'orders' }
      }));
    }
  });

  onDestroy(() => {
    window.removeEventListener('navigateToScreen', handleNavigateToScreen);
  });
</script>

<ErrorBoundary name="Operations Hub">
  <ModuleLayout title="Operations Hub" {tabs} {activeTab} on:tabChange={handleTabChange}>
    <!-- @migration-task: migrate this slot by hand, `header-actions` is an invalid identifier -->
  <svelte:fragment slot="header-actions">
      {#if activeTab === 'pos'}
        <Button variant="primary" on:click={handleNewPO}>
          + New Supplier Order
        </Button>
      {:else if activeTab === 'delivery-notes'}
        <Button variant="primary" on:click={handleNewDN}>
          + New Delivery Note
        </Button>
      {/if}
    </svelte:fragment>

    {#if activeTab === 'pos'}
      <PurchaseOrdersScreen embedded={true} />
    {:else if activeTab === 'delivery-notes'}
      <DeliveryNotesScreen embedded={true} />
    {:else if activeTab === 'fulfillment'}
      <InventoryFulfillmentScreen embedded={true} />
    {:else if activeTab === 'serials'}
      <SerialTraceScreen embedded={true} />
    {/if}
  </ModuleLayout>
</ErrorBoundary>

<style>
  /* All styling now handled by Button component and ModuleLayout */
</style>
