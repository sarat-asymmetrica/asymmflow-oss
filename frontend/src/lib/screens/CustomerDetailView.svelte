<script lang="ts">
  import { createBubbler, stopPropagation } from 'svelte/legacy';

  const bubble = createBubbler();
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import { brand } from '$lib/brand';
  import TableSkeleton from '$lib/components/ui/TableSkeleton.svelte';
  import CardGridSkeleton from '$lib/components/ui/CardGridSkeleton.svelte';
  import ContextTaskModal from '$lib/components/ContextTaskModal.svelte';
  import CustomerSidebar from '$lib/components/customer/CustomerSidebar.svelte';
  import CustomerOrdersTab from '$lib/components/customer/CustomerOrdersTab.svelte';
  import CustomerInvoicesTab from '$lib/components/customer/CustomerInvoicesTab.svelte';
  import CustomerRFQsTab from '$lib/components/customer/CustomerRFQsTab.svelte';
  import CustomerNotesTab from '$lib/components/customer/CustomerNotesTab.svelte';
  import CustomerContactsStrip from '$lib/components/customer/CustomerContactsStrip.svelte';
  import CustomerOverviewTab from '$lib/components/customer/CustomerOverviewTab.svelte';
  import CustomerDetailHeader from '$lib/components/customer/CustomerDetailHeader.svelte';
  import { toast } from '$lib/stores/toasts';
  import { GetCustomerFullProfile } from '../../../wailsjs/go/main/App';
import { UpdateCustomer, DeleteCustomer } from '../../../wailsjs/go/main/CRMService';
  import type { main } from '../../../wailsjs/go/models';

  interface Props {
    customerId: string;
  }

  let { customerId }: Props = $props();

  const dispatch = createEventDispatcher();

  let loading = $state(true);
  let error: string | null = $state(null);
  let profile: main.CustomerFullProfile | null = $state(null);
  let activeTab = $state('overview');
  let showNoteModal = $state(false);
  let editMode = $state(false);
  let editForm: any = $state({});
  let saveLoading = $state(false);
  let showContactModal = $state(false);
  let showDeleteConfirm = $state(false);
  let deleteLoading = $state(false);
  let customerDeleted = $state(false);
  let showTaskModal = $state(false);

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (showDeleteConfirm) showDeleteConfirm = false;
      else if (showTaskModal) showTaskModal = false;
      else if (showNoteModal) showNoteModal = false;
      else if (showContactModal) showContactModal = false;
    }
  }

  async function handleDeleteCustomer() {
    deleteLoading = true;
    try {
      await DeleteCustomer(customerId);
      customerDeleted = true;
      toast.success('Customer deleted successfully');
      dispatch('back', { refresh: true });
    } catch (err: any) {
      toast.danger('Failed to delete customer: ' + (err?.message || err));
    } finally {
      deleteLoading = false;
      showDeleteConfirm = false;
    }
  }

  function startEdit() {
    if (profile) {
      editForm = { ...profile };
      editMode = true;
    }
  }

  function cancelEdit() {
    editMode = false;
    editForm = {};
  }

  async function saveEdit() {
    saveLoading = true;
    try {
      await UpdateCustomer(editForm);
      toast.success('Customer updated successfully');
      editMode = false;
      await loadProfile();
    } catch (err) {
      toast.danger('Failed to update customer: ' + err);
    } finally {
      saveLoading = false;
    }
  }

  const tabs = [
    { id: 'overview', label: 'Overview' },
    { id: 'orders', label: 'Orders' },
    { id: 'invoices', label: 'Invoices' },
    { id: 'rfqs', label: 'RFQs' },
    { id: 'notes', label: 'Notes' },
  ];

  async function loadProfile() {
    loading = true;
    error = null;
    try {
      profile = await GetCustomerFullProfile(customerId);
    } catch (err) {
      console.error('Failed to load customer profile:', err);
      error = err instanceof Error ? err.message : 'Failed to load customer profile';
      toast.danger(error);
    } finally {
      loading = false;
    }
  }

  function goBack() {
    dispatch('back');
  }

  function navigateTo(target: Record<string, string>) {
    window.dispatchEvent(new CustomEvent('navigateToScreen', { detail: target }));
  }

  // B3 360-continuity: drill from a customer's order/invoice/RFQ row to the
  // matching document, mirroring CostingSheetScreen's pending-store handoff.
  function openOrder() {
    // OrdersScreen (SalesHub "orders" tab) has no pending-store reader yet, so
    // this lands on the correct surface without a doc preselect (residue).
    navigateTo({ screen: 'opportunities', tab: 'orders' });
  }

  function openInvoice(invoice: any) {
    sessionStorage.setItem(
      'asymmflow.pendingInvoiceFocus',
      JSON.stringify({ id: invoice.id, invoice_number: invoice.invoice_number })
    );
    navigateTo({ screen: 'finance', tab: 'invoices', company: invoice.division || brand.defaultDivision });
  }

  function openRFQ(rfq: any) {
    if (!rfq?.id) return;
    sessionStorage.setItem('asymmflow.pendingOpportunityId', String(rfq.id));
    navigateTo({ screen: 'opportunities' });
  }

  function startNewRFQ() {
    if (!profile) return;
    sessionStorage.setItem(
      'asymmflow.pendingRFQCustomer',
      JSON.stringify({ id: customerId, name: profile.business_name })
    );
    navigateTo({ screen: 'opportunities' });
  }

  onMount(loadProfile);
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="detail-view">
  <CustomerDetailHeader
    {profile}
    {editMode}
    {customerDeleted}
    bind:showTaskModal
    bind:showDeleteConfirm
    {goBack}
    {startNewRFQ}
    {startEdit}
  />

  {#if loading}
    <div class="skeleton-strip">
      {#each Array(5) as _, i (i)}
        <div class="skeleton-tab-pill"></div>
      {/each}
    </div>
    <div class="content-layout">
      <aside class="sidebar-skeleton">
        <CardGridSkeleton statCards={0} panels={2} panelCols={1} panelRows={5} />
      </aside>
      <main class="main">
        <TableSkeleton rows={6} cols={4} />
      </main>
    </div>
  {:else if error}
    <div class="error-state">
      <p class="error-message">{error}</p>
      <Button variant="primary" on:click={loadProfile}>Retry</Button>
    </div>
  {:else if profile}
    <CustomerContactsStrip {customerId} {profile} bind:showContactModal onSaved={loadProfile} />

    <div class="content-layout">
      <!-- Left Sidebar -->
      <CustomerSidebar {profile} />

      <!-- Main Content -->
      <main class="main">
        <nav class="tabs">
          {#each tabs as tab}
            <button
              class="tab"
              class:active={activeTab === tab.id}
              onclick={() => activeTab = tab.id}
            >
              {tab.label}
            </button>
          {/each}
        </nav>

        <div class="tab-content">
          {#if activeTab === 'overview'}
            <CustomerOverviewTab {profile} {editMode} bind:editForm {saveLoading} {cancelEdit} {saveEdit} />

          {:else if activeTab === 'orders'}
            <CustomerOrdersTab {profile} {openOrder} />

          {:else if activeTab === 'invoices'}
            <CustomerInvoicesTab {profile} {openInvoice} />

          {:else if activeTab === 'rfqs'}
            <CustomerRFQsTab {profile} {openRFQ} />

          {:else if activeTab === 'notes'}
            <CustomerNotesTab {customerId} {profile} bind:showNoteModal onSaved={loadProfile} />
          {/if}
        </div>
      </main>
    </div>
  {/if}

  {#if showDeleteConfirm}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="delete-overlay" onclick={() => showDeleteConfirm = false}>
      <div class="delete-modal" onclick={stopPropagation(bubble('click'))}>
        <h3 style="margin: 0 0 12px 0; color: #e74c3c;">Delete Customer</h3>
        <p style="margin: 0 0 20px 0; color: var(--text-secondary);">
          Are you sure you want to delete <strong>{profile.business_name}</strong>? This action will soft-delete the customer record.
        </p>
        <div style="display: flex; gap: 8px; justify-content: flex-end;">
          <Button variant="secondary" size="sm" on:click={() => showDeleteConfirm = false} disabled={deleteLoading}>Cancel</Button>
          <Button variant="primary" size="sm" on:click={handleDeleteCustomer} disabled={deleteLoading} style="background: #e74c3c; border-color: #e74c3c;">
            {deleteLoading ? 'Deleting...' : 'Delete'}
          </Button>
        </div>
      </div>
    </div>
  {/if}
</div>

{#if profile}
  <ContextTaskModal
    open={showTaskModal}
    title="Create Customer Task"
    subtitle={`Link work directly to ${profile.business_name}`}
    defaults={{
      customer_id: customerId,
      seed_title: `Customer follow-up: ${profile.business_name}`,
    }}
    on:close={() => showTaskModal = false}
    on:created={() => showTaskModal = false}
  />
{/if}

<style>
  .detail-view { padding: 16px; }

  .skeleton-strip { display: flex; gap: 12px; border-bottom: 1px solid var(--border); margin-bottom: 16px; padding-bottom: 12px; }
  .skeleton-tab-pill {
    width: 84px;
    height: 14px;
    border-radius: var(--border-radius-sm);
    background: var(--surface-elevated);
  }
  .sidebar-skeleton { display: flex; flex-direction: column; }

  .error-state { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 100px; gap: 16px; }
  .error-message { color: var(--text-danger); font-size: 14px; margin: 0; }

  .content-layout { display: grid; grid-template-columns: 280px 1fr; gap: 24px; }

  .tabs { display: flex; gap: 0; border-bottom: 1px solid var(--border); margin-bottom: 16px; }
  .tab { padding: 12px 20px; background: none; border: none; border-bottom: 2px solid transparent; font-size: 14px; font-weight: 500; color: var(--text-secondary); cursor: pointer; transition: all var(--transition-fast); }
  .tab:hover { color: var(--text-primary); }
  .tab.active { color: var(--text-primary); border-bottom-color: var(--brand-indigo); }

  .delete-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .delete-modal {
    background: var(--surface, #fff);
    border-radius: 12px;
    padding: 24px;
    max-width: 420px;
    width: 90%;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  }
</style>
