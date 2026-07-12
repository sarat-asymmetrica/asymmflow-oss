<script lang="ts">
  import { createBubbler, stopPropagation } from 'svelte/legacy';

  const bubble = createBubbler();
  import { onMount, createEventDispatcher } from 'svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import { toast } from '$lib/stores/toasts';
  import { confirm } from '$lib/stores/confirm';
  import { formatBHDValue } from '$lib/utils/formatters';
  import { GetSupplierFullProfile } from '../../../wailsjs/go/main/App';
import { AddSupplierNote, AddSupplierIssue, ResolveSupplierIssue, UpdateSupplier, AddSupplierContact, ListSupplierContacts } from '../../../wailsjs/go/main/CRMService';
  import type { main } from '../../../wailsjs/go/models';

  interface Props {
    supplierId: string;
  }

  let { supplierId }: Props = $props();

  const dispatch = createEventDispatcher();

  let loading = $state(true);
  let error: string | null = $state(null);
  let profile: main.SupplierFullProfile | null = $state(null);
  let activeTab = $state('overview');
  let showNoteModal = $state(false);
  let showIssueModal = $state(false);
  let newNoteType = $state('general');
  let newNoteContent = $state('');
  let newIssueOrderRef = $state('');
  let newIssueDescription = $state('');
  let newIssueCost = $state(0);
  let editMode = $state(false);
  let editForm: any = $state({});
  let saveLoading = $state(false);
  let supplierContacts: any[] = $state([]);
  let showContactModal = $state(false);
  let contactForm = $state({ contact_name: '', job_title: '', email: '', phone: '', address: '', is_primary_contact: false });

  // Wave 9.6 Sh1: SupplierMaster.BrandsHandled/ProductTypes are JSON-encoded
  // STRING columns, but SupplierFullProfile carries them as string[] (decoded
  // for display). If we pass the array straight through, Wails serializes it
  // as a JSON array and Go's json.Unmarshal into the string field errors,
  // rejecting the whole update. Re-encode arrays back to their JSON-string
  // form so the round-trip stays type-correct.
  function encodeStringArrayField(value: any): string {
    if (Array.isArray(value)) return JSON.stringify(value);
    return value || '';
  }

  function buildSupplierUpdatePayload(source: any): any {
    return {
      id: source?.id || '',
      supplier_code: source?.supplier_code || '',
      supplier_name: source?.supplier_name || '',
      country: source?.country || '',
      lead_time_days: Number(source?.lead_time_days || 0) || 0,
      tax_id: source?.tax_id || '',
      supplier_type: source?.supplier_type || '',
      brands_handled: encodeStringArrayField(source?.brands_handled),
      product_types: encodeStringArrayField(source?.product_types),
      primary_contact: source?.primary_contact || '',
      email: source?.email || '',
      phone: source?.phone || '',
      address: source?.address || '',
      bank_name: source?.bank_name || '',
      account_number: source?.account_number || '',
      iban: source?.iban || '',
      swift_code: source?.swift_code || '',
      payment_terms: source?.payment_terms || 'Net 30',
      rating: Number(source?.rating || 0) || 0,
      notes: typeof source?.notes === 'string' ? source.notes : ''
    };
  }

  function startEdit() {
    if (profile) {
      editForm = buildSupplierUpdatePayload(profile);
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
      await UpdateSupplier(buildSupplierUpdatePayload(editForm));
      toast.success('Supplier updated successfully');
      editMode = false;
      await loadProfile();
    } catch (err) {
      const errorMsg = err?.message || String(err);
      toast.danger('Failed to update supplier: ' + errorMsg);
    } finally {
      saveLoading = false;
    }
  }

  const tabs = [
    { id: 'overview', label: 'Overview' },
    { id: 'pos', label: 'Purchase Orders' },
    { id: 'invoices', label: 'Invoices' },
    { id: 'issues', label: 'Issues' },
    { id: 'notes', label: 'Notes' },
  ];

  async function loadProfile() {
    loading = true;
    error = null;
    try {
      profile = await GetSupplierFullProfile(supplierId);
      console.log('Supplier Profile loaded:', profile);
      await loadContacts();
    } catch (err) {
      console.error('Failed to load supplier profile:', err);
      error = err instanceof Error ? err.message : 'Failed to load supplier profile';
      toast.danger(error);
    } finally {
      loading = false;
    }
  }

  async function loadContacts() {
    try {
      supplierContacts = await ListSupplierContacts(supplierId) || [];
    } catch { supplierContacts = []; }
  }

  async function saveContact() {
    if (!contactForm.contact_name.trim()) { toast.warning('Name is required'); return; }
    try {
      await AddSupplierContact({ ...contactForm, supplier_id: supplierId } as any);
      toast.success('Contact added');
      showContactModal = false;
      contactForm = { contact_name: '', job_title: '', email: '', phone: '', address: '', is_primary_contact: false };
      await loadContacts();
    } catch (err) {
      toast.danger('Failed to add contact: ' + err);
    }
  }

  async function saveNote() {
    if (!newNoteContent.trim()) {
      toast.warning('Please enter note content');
      return;
    }
    try {
      await AddSupplierNote(supplierId, newNoteType, newNoteContent);
      toast.success('Note added');
      showNoteModal = false;
      newNoteContent = '';
      await loadProfile();
    } catch (err) {
      console.error('Failed to add note:', err);
      toast.danger('Failed to add note');
    }
  }

  async function saveIssue() {
    if (!newIssueOrderRef.trim() || !newIssueDescription.trim()) {
      toast.warning('Please enter order reference and description');
      return;
    }
    try {
      await AddSupplierIssue(supplierId, newIssueOrderRef, newIssueDescription, newIssueCost);
      toast.success('Issue reported');
      showIssueModal = false;
      newIssueOrderRef = '';
      newIssueDescription = '';
      newIssueCost = 0;
      await loadProfile();
    } catch (err) {
      console.error('Failed to report issue:', err);
      toast.danger('Failed to report issue');
    }
  }

  async function resolveIssue(issueId: string) {
    const r = await confirm.askForReason({
      title: 'Resolve Issue',
      message: 'Provide resolution notes for this issue.',
      reasonLabel: 'Resolution notes',
      reasonRequired: true
    });
    if (!r.confirmed) return;
    const resolution = r.reason;

    try {
      await ResolveSupplierIssue(issueId, resolution);
      toast.success('Issue resolved');
      await loadProfile();
    } catch (err) {
      console.error('Failed to resolve issue:', err);
      toast.danger('Failed to resolve issue');
    }
  }

  function goBack() {
    dispatch('back');
  }

  function navigateTo(target: Record<string, string>) {
    window.dispatchEvent(new CustomEvent('navigateToScreen', { detail: target }));
  }

  // B3 360-continuity: PO/invoice rows drill into Operations at the right tab.
  // PurchaseOrdersScreen / SupplierInvoicesScreen have no pending-store reader
  // yet, so this lands on the correct surface without a doc preselect (residue,
  // matched to CustomerDetailView's orders-row drill which has the same gap).
  function openPO() {
    navigateTo({ screen: 'operations', tab: 'pos' });
  }

  function openSupplierInvoice() {
    // Wave 9.2 B2/B7: supplier invoices now live in the Finance hub's AP cluster.
    navigateTo({ screen: 'finance', tab: 'supplier_invoices' });
  }

  function startNewPO() {
    if (!profile) return;
    sessionStorage.setItem(
      'asymmflow.pendingPOSupplier',
      JSON.stringify({ id: supplierId, name: profile.supplier_name })
    );
    navigateTo({ screen: 'operations', tab: 'pos' });
  }

  function formatDate(date: any): string {
    if (!date) return 'N/A';
    try {
      // Handle time.Time objects from Go backend
      const d = typeof date === 'string' ? new Date(date) : new Date(date);
      if (isNaN(d.getTime())) return 'N/A';
      return d.toLocaleDateString('en-US', {
        month: 'short', day: 'numeric', year: 'numeric'
      });
    } catch {
      return 'N/A';
    }
  }

  function formatCurrency(value: number): string {
    return formatBHDValue(value || 0);
  }

  onMount(loadProfile);
</script>

<div class="detail-view">
  <header class="header">
    <button class="back-btn" onclick={goBack}>Back to Suppliers</button>
    {#if profile}
      <div class="header-info">
        <h1>{profile.supplier_name}</h1>
        <div class="header-badges">
          {#if profile.rating > 0}
          <span class="grade-badge grade-{profile.rating >= 4 ? 'a' : profile.rating >= 3 ? 'b' : profile.rating >= 2 ? 'c' : 'd'}">
            {profile.rating >= 4 ? 'A' : profile.rating >= 3 ? 'B' : profile.rating >= 2 ? 'C' : 'D'}
          </span>
          {/if}
          <span class="status-badge">{profile.supplier_type || 'Supplier'}</span>
          {#if !editMode}
            <Button variant="secondary" size="sm" on:click={startNewPO}>New PO</Button>
            <Button variant="secondary" size="sm" on:click={startEdit}>Edit</Button>
          {/if}
        </div>
      </div>
    {/if}
  </header>

  {#if loading}
    <div class="loading-state"><WabiSpinner size="lg" /></div>
  {:else if error}
    <div class="error-state">
      <p class="error-message">{error}</p>
      <Button variant="primary" on:click={loadProfile}>Retry</Button>
    </div>
  {:else if profile}
    <!-- Contacts Strip - Full Width -->
    <div class="contacts-strip">
      <div class="contacts-strip-header">
        <h3 class="strip-title">CONTACTS</h3>
        <button class="add-contact-btn-inline" onclick={() => showContactModal = true}>+ Add</button>
      </div>
      <div class="contacts-scroll-container">
        {#if supplierContacts.length > 0}
          {#each supplierContacts as contact}
            <div class="contact-card">
              <div class="contact-card-avatar">
                {contact.contact_name?.charAt(0)?.toUpperCase() || '?'}
              </div>
              <div class="contact-card-info">
                <span class="contact-card-name">
                  {contact.contact_name}
                  {#if contact.is_primary_contact}<span class="primary-tag">Primary</span>{/if}
                </span>
                {#if contact.job_title}<span class="contact-card-role">{contact.job_title}</span>{/if}
                {#if contact.email}<span class="contact-card-detail">{contact.email}</span>{/if}
                {#if contact.phone}<span class="contact-card-detail">{contact.phone}</span>{/if}
              </div>
            </div>
          {/each}
        {:else if profile.primary_contact || profile.email || profile.phone}
          <div class="contact-card">
            <div class="contact-card-avatar">
              {profile.primary_contact?.charAt(0)?.toUpperCase() || '?'}
            </div>
            <div class="contact-card-info">
              {#if profile.primary_contact}<span class="contact-card-name">{profile.primary_contact}</span>{/if}
              {#if profile.email}<span class="contact-card-detail">{profile.email}</span>{/if}
              {#if profile.phone}<span class="contact-card-detail">{profile.phone}</span>{/if}
            </div>
          </div>
        {:else}
          <div class="contact-card empty-contact">
            <span class="empty-contact-text">No contacts on file</span>
          </div>
        {/if}
      </div>
    </div>

    <div class="content-layout">
      <!-- Left Sidebar -->
      <aside class="sidebar">
        <Card padding="md">
          <h3 class="sidebar-title">OVERVIEW</h3>
          <div class="info-grid">
            <div class="info-item">
              <span class="label">Supplier Code</span>
              <span class="value code">{profile.supplier_code}</span>
            </div>
            <div class="info-item">
              <span class="label">Location</span>
              <span class="value">{profile.country}</span>
            </div>
            {#if profile.lead_time_days > 0}
            <div class="info-item">
              <span class="label">Lead Time</span>
              <span class="value">{profile.lead_time_days} days</span>
            </div>
            {/if}
            {#if profile.rating > 0}
            <div class="info-item">
              <span class="label">Rating</span>
              <span class="value">{profile.rating}/5</span>
            </div>
            {/if}
            {#if profile.on_time_delivery_pct > 0}
            <div class="info-item">
              <span class="label">On-Time Delivery</span>
              <span class="value">{profile.on_time_delivery_pct.toFixed(0)}%</span>
            </div>
            {/if}
          </div>
        </Card>

        <Card padding="md">
          <h3 class="sidebar-title">BANK DETAILS</h3>
          {#if profile.bank_name || profile.account_number || profile.iban || profile.swift_code}
            <div class="info-grid">
              {#if profile.bank_name}
              <div class="info-item">
                <span class="label">Bank</span>
                <span class="value">{profile.bank_name}</span>
              </div>
              {/if}
              {#if profile.account_number}
              <div class="info-item">
                <span class="label">Account</span>
                <span class="value code">{profile.account_number}</span>
              </div>
              {/if}
              {#if profile.iban}
              <div class="info-item">
                <span class="label">IBAN</span>
                <span class="value code">{profile.iban}</span>
              </div>
              {/if}
              {#if profile.swift_code}
              <div class="info-item">
                <span class="label">SWIFT</span>
                <span class="value code">{profile.swift_code}</span>
              </div>
              {/if}
            </div>
          {:else}
            <p class="empty-text">No bank details on file</p>
          {/if}
        </Card>

        {#if profile.brands_handled?.length > 0}
          <Card padding="md">
            <h3 class="sidebar-title">BRANDS</h3>
            <div class="brands-list">
              {#each profile.brands_handled as brand}
                <span class="brand-tag">{brand}</span>
              {/each}
            </div>
          </Card>
        {/if}
      </aside>

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
            {#if editMode}
              <Card padding="md">
                <h3 class="section-title">EDIT SUPPLIER</h3>
                <div class="edit-form">
                  <div class="form-row">
                    <div class="form-group"><label for="supplier-detail-name">Supplier Name</label><input id="supplier-detail-name" type="text" bind:value={editForm.supplier_name} /></div>
                    <div class="form-group"><label for="supplier-detail-code">Supplier Code</label><input id="supplier-detail-code" type="text" bind:value={editForm.supplier_code} /></div>
                  </div>
                  <div class="form-row">
                    <div class="form-group"><label for="supplier-detail-primary-contact">Primary Contact</label><input id="supplier-detail-primary-contact" type="text" bind:value={editForm.primary_contact} /></div>
                    <div class="form-group"><label for="supplier-detail-email">Email</label><input id="supplier-detail-email" type="email" bind:value={editForm.email} /></div>
                  </div>
                  <div class="form-row">
                    <div class="form-group"><label for="supplier-detail-phone">Phone</label><input id="supplier-detail-phone" type="text" bind:value={editForm.phone} /></div>
                    <div class="form-group"><label for="supplier-detail-country">Country</label><input id="supplier-detail-country" type="text" bind:value={editForm.country} /></div>
                  </div>
                  <div class="form-row">
                    <div class="form-group"><label for="supplier-detail-payment-terms">Payment Terms</label>
                      <select id="supplier-detail-payment-terms" bind:value={editForm.payment_terms}>
                        <option value="Net 30">Net 30</option><option value="Net 60">Net 60</option>
                        <option value="Net 90">Net 90</option><option value="CIA">CIA</option>
                        <option value="COD">COD</option><option value="LC">LC</option>
                      </select>
                    </div>
                    <div class="form-group"><label for="supplier-detail-lead-time-days">Lead Time (Days)</label><input id="supplier-detail-lead-time-days" type="number" bind:value={editForm.lead_time_days} /></div>
                  </div>
                  <div class="form-row">
                    <div class="form-group"><label for="supplier-detail-address">Address</label><input id="supplier-detail-address" type="text" bind:value={editForm.address} /></div>
                    <div class="form-group"><label for="supplier-detail-tax-id">Tax ID</label><input id="supplier-detail-tax-id" type="text" bind:value={editForm.tax_id} /></div>
                  </div>
                  <div class="form-actions">
                    <Button variant="secondary" on:click={cancelEdit} disabled={saveLoading}>Cancel</Button>
                    <Button variant="primary" on:click={saveEdit} disabled={saveLoading}>
                      {saveLoading ? 'Saving...' : 'Save Changes'}
                    </Button>
                  </div>
                </div>
              </Card>
            {/if}
            <div class="metrics-grid">
              <Card padding="md">
                <div class="metric">
                  <span class="metric-label">Total Purchases</span>
                  <span class="metric-value">{formatCurrency(profile.total_purchases)} BHD</span>
                </div>
              </Card>
              <Card padding="md">
                <div class="metric">
                  <span class="metric-label">Total POs</span>
                  <span class="metric-value">{profile.total_pos}</span>
                </div>
              </Card>
              <Card padding="md">
                <div class="metric">
                  <span class="metric-label">Avg PO Value</span>
                  <span class="metric-value">{formatCurrency(profile.avg_po_value)} BHD</span>
                </div>
              </Card>
              <Card padding="md">
                <div class="metric">
                  <span class="metric-label">Open Issues</span>
                  <span class="metric-value">{profile.open_issues}</span>
                </div>
              </Card>
            </div>

            {#if profile.outstanding_bhd > 0 || profile.overdue_bhd > 0}
            <Card padding="md">
              <h3 class="section-title">PAYABLES</h3>
              <div class="aging-grid">
                <div class="aging-item">
                  <span class="aging-label">Outstanding</span>
                  <span class="aging-value">{formatCurrency(profile.outstanding_bhd)} BHD</span>
                </div>
                <div class="aging-item">
                  <span class="aging-label">Overdue</span>
                  <span class="aging-value" class:danger={profile.overdue_bhd > 0}>{formatCurrency(profile.overdue_bhd)} BHD</span>
                </div>
              </div>
            </Card>
            {/if}

          {:else if activeTab === 'pos'}
            <Card padding="md">
              <h3 class="section-title">PURCHASE ORDERS</h3>
              {#if profile.recent_pos?.length > 0}
                <div class="list">
                  {#each profile.recent_pos as po}
                    <div
                      class="list-item"
                      role="button"
                      tabindex="0"
                      onclick={() => openPO()}
                      onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), openPO())}
                    >
                      <div class="item-main">
                        <span class="item-code">{po.po_number}</span>
                        <span class="item-date">{formatDate(po.po_date)}</span>
                      </div>
                      <div class="item-meta">
                        <span class="item-status">{po.status}</span>
                        <span class="item-amount">{formatCurrency(po.total_bhd)} BHD</span>
                      </div>
                    </div>
                  {/each}
                </div>
              {:else}
                <p class="empty-text">No purchase orders found</p>
              {/if}
            </Card>

          {:else if activeTab === 'invoices'}
            <Card padding="md">
              <h3 class="section-title">SUPPLIER INVOICES</h3>
              {#if profile.recent_invoices?.length > 0}
                <div class="list">
                  {#each profile.recent_invoices as invoice}
                    <div
                      class="list-item"
                      role="button"
                      tabindex="0"
                      onclick={() => openSupplierInvoice()}
                      onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), openSupplierInvoice())}
                    >
                      <div class="item-main">
                        <span class="item-code">{invoice.invoice_number}</span>
                        <span class="item-date">{formatDate(invoice.invoice_date)}</span>
                      </div>
                      <div class="item-meta">
                        <span class="item-status status-{invoice.status?.toLowerCase()}">{invoice.status}</span>
                        <span class="item-amount">{formatCurrency(invoice.total_bhd)} BHD</span>
                      </div>
                    </div>
                  {/each}
                </div>
              {:else}
                <p class="empty-text">No invoices found</p>
              {/if}
            </Card>

          {:else if activeTab === 'issues'}
            <Card padding="md">
              <div class="notes-header">
                <h3 class="section-title">ISSUES</h3>
                <Button variant="primary" on:click={() => showIssueModal = true}>+ Report Issue</Button>
              </div>
              {#if profile.issues?.length > 0}
                <div class="issues-list">
                  {#each profile.issues as issue}
                    <div class="issue-item">
                      <div class="issue-header">
                        <div class="issue-title-row">
                          <span class="issue-code">{issue.order_ref || 'N/A'}</span>
                          <span class="status-badge status-{issue.status}">{issue.status}</span>
                        </div>
                        <span class="issue-date">{formatDate(issue.created_at)}</span>
                      </div>
                      <p class="issue-description">{issue.description}</p>
                      {#if issue.cost_bhd > 0}
                        <div class="issue-cost">Cost: {formatCurrency(issue.cost_bhd)} BHD</div>
                      {/if}
                      {#if issue.status === 'resolved' && issue.resolved_at}
                        <div class="resolved-info">
                          Resolved: {formatDate(issue.resolved_at)}
                          {#if issue.resolution}
                            <br><span class="resolution-text">{issue.resolution}</span>
                          {/if}
                        </div>
                      {:else}
                        <Button variant="ghost" size="sm" on:click={() => resolveIssue(issue.id)}>
                          Mark Resolved
                        </Button>
                      {/if}
                    </div>
                  {/each}
                </div>
              {:else}
                <p class="empty-text">No issues reported</p>
              {/if}
            </Card>

          {:else if activeTab === 'notes'}
            <Card padding="md">
              <div class="notes-header">
                <h3 class="section-title">NOTES</h3>
                <Button variant="primary" on:click={() => showNoteModal = true}>+ Add Note</Button>
              </div>
              {#if profile.notes?.length > 0}
                <div class="notes-list">
                  {#each profile.notes as note}
                    <div class="note-item">
                      <div class="note-header">
                        <span class="note-type">{note.note_type}</span>
                        <span class="note-date">{formatDate(note.created_at)}</span>
                      </div>
                      <p class="note-content">{note.content}</p>
                    </div>
                  {/each}
                </div>
              {:else}
                <p class="empty-text">No notes yet</p>
              {/if}
            </Card>
          {/if}
        </div>
      </main>
    </div>
  {/if}
</div>

{#if showNoteModal}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="modal-backdrop" onclick={() => showNoteModal = false}>
    <div class="modal" onclick={stopPropagation(bubble('click'))}>
      <h3>Add Note</h3>
      <div class="form-group">
        <label for="supplier-note-type">Type</label>
        <select id="supplier-note-type" bind:value={newNoteType}>
          <option value="general">General</option>
          <option value="delivery">Delivery</option>
          <option value="quality">Quality</option>
          <option value="pricing">Pricing</option>
        </select>
      </div>
      <div class="form-group">
        <label for="supplier-note-content">Content</label>
        <textarea id="supplier-note-content" bind:value={newNoteContent} rows="4" placeholder="Enter note..."></textarea>
      </div>
      <div class="modal-actions">
        <Button variant="ghost" on:click={() => showNoteModal = false}>Cancel</Button>
        <Button variant="primary" on:click={saveNote}>Save Note</Button>
      </div>
    </div>
  </div>
{/if}

{#if showIssueModal}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="modal-backdrop" onclick={() => showIssueModal = false}>
    <div class="modal" onclick={stopPropagation(bubble('click'))}>
      <h3>Report Issue</h3>
      <div class="form-group">
        <label for="supplier-issue-order-reference">Order Reference</label>
        <input id="supplier-issue-order-reference" type="text" bind:value={newIssueOrderRef} placeholder="PO or order number" />
      </div>
      <div class="form-group">
        <label for="supplier-issue-description">Description</label>
        <textarea id="supplier-issue-description" bind:value={newIssueDescription} rows="4" placeholder="Describe the issue..."></textarea>
      </div>
      <div class="form-group">
        <label for="supplier-issue-cost-impact">Cost Impact (BHD)</label>
        <input id="supplier-issue-cost-impact" type="number" step="0.001" bind:value={newIssueCost} placeholder="0.000" />
      </div>
      <div class="modal-actions">
        <Button variant="ghost" on:click={() => showIssueModal = false}>Cancel</Button>
        <Button variant="primary" on:click={saveIssue}>Report Issue</Button>
      </div>
    </div>
  </div>
{/if}

{#if showContactModal}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="modal-backdrop" onclick={() => showContactModal = false}>
    <div class="modal" onclick={stopPropagation(bubble('click'))}>
      <h3>Add Contact</h3>
      <div class="form-group">
        <label for="supplier-contact-name">Name *</label>
        <input id="supplier-contact-name" type="text" bind:value={contactForm.contact_name} placeholder="Contact person name" />
      </div>
      <div class="form-group">
        <label for="supplier-contact-job-title">Job Title</label>
        <input id="supplier-contact-job-title" type="text" bind:value={contactForm.job_title} placeholder="e.g. Sales Manager" />
      </div>
      <div class="form-group">
        <label for="supplier-contact-email">Email</label>
        <input id="supplier-contact-email" type="email" bind:value={contactForm.email} placeholder="email@company.com" />
      </div>
      <div class="form-group">
        <label for="supplier-contact-phone">Phone</label>
        <input id="supplier-contact-phone" type="text" bind:value={contactForm.phone} placeholder="+49 123 456 7890" />
      </div>
      <div class="form-group">
        <label for="supplier-contact-address">Address</label>
        <textarea id="supplier-contact-address" bind:value={contactForm.address} rows="2" placeholder="Address"></textarea>
      </div>
      <div class="form-group">
        <label for="supplier-contact-primary"><input id="supplier-contact-primary" type="checkbox" bind:checked={contactForm.is_primary_contact} /> Primary Contact</label>
      </div>
      <div class="modal-actions">
        <Button variant="ghost" on:click={() => showContactModal = false}>Cancel</Button>
        <Button variant="primary" on:click={saveContact}>Save Contact</Button>
      </div>
    </div>
  </div>
{/if}

<style>
  .detail-view { padding: 16px; }

  .header { margin-bottom: 24px; }
  .back-btn { background: none; border: none; color: var(--brand-indigo); cursor: pointer; font-size: 14px; padding: 0; margin-bottom: 12px; transition: all var(--transition-fast); }
  .back-btn:hover { text-decoration: underline; }

  .header-info { display: flex; justify-content: space-between; align-items: center; }
  .header-info h1 { margin: 0; font-size: 24px; font-weight: 500; }
  .header-badges { display: flex; gap: 8px; }

  .grade-badge { padding: 4px 12px; border-radius: 4px; font-size: 12px; font-weight: 600; }
  .grade-badge.grade-a { background: #DCFCE7; color: #166534; }
  .grade-badge.grade-b { background: #DBEAFE; color: #1E40AF; }
  .grade-badge.grade-c { background: #FEF9C3; color: #854D0E; }
  .grade-badge.grade-d { background: #FEE2E2; color: #991B1B; }

  .status-badge { padding: 4px 12px; background: var(--surface-elevated); border: 1px solid var(--border); border-radius: 4px; font-size: 12px; }
  .status-badge.active { background: #DCFCE7; color: #166534; border-color: #166534; }
  .status-badge.status-resolved { background: #DCFCE7; color: #166534; border: none; }
  .status-badge.status-open { background: #FEE2E2; color: #991B1B; border: none; }
  .status-badge.status-investigating { background: #FEF9C3; color: #854D0E; border: none; }

  .loading-state { display: flex; justify-content: center; padding: 100px; }

  .error-state { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 100px; gap: 16px; }
  .error-message { color: var(--text-danger); font-size: 14px; margin: 0; }

  .content-layout { display: grid; grid-template-columns: 280px 1fr; gap: 24px; }

  .sidebar { display: flex; flex-direction: column; gap: 16px; }
  .sidebar-title { font-size: 11px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-secondary); margin: 0 0 12px 0; }

  .info-grid { display: flex; flex-direction: column; gap: 12px; }
  .info-item { display: flex; flex-direction: column; gap: 2px; }
  .label { font-size: 11px; color: var(--text-muted); text-transform: uppercase; }
  .value { font-size: 14px; font-weight: 500; }
  .value.code { font-family: 'JetBrains Mono', monospace; color: var(--brand-indigo); }

  .contacts-strip {
    margin-bottom: 24px;
    background: var(--surface, #fff);
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 12px;
    padding: 16px;
  }

  .contacts-strip-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }

  .strip-title {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
    margin: 0;
  }

  .add-contact-btn-inline {
    background: none;
    border: 1px solid var(--border);
    padding: 4px 12px;
    font-size: 12px;
    color: var(--text-secondary);
    cursor: pointer;
    border-radius: 6px;
    transition: all 0.2s;
  }

  .add-contact-btn-inline:hover {
    border-color: var(--brand-indigo, #1D1D1F);
    color: var(--text-primary);
  }

  .contacts-scroll-container {
    display: flex;
    gap: 12px;
    overflow-x: auto;
    padding-bottom: 8px;
    scroll-snap-type: x mandatory;
  }

  /* Scrollbar styling */
  .contacts-scroll-container::-webkit-scrollbar {
    height: 4px;
  }
  .contacts-scroll-container::-webkit-scrollbar-track {
    background: transparent;
  }
  .contacts-scroll-container::-webkit-scrollbar-thumb {
    background: var(--border);
    border-radius: 2px;
  }

  .contact-card {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    padding: 12px 16px;
    background: var(--surface-elevated, #F8F8F8);
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 8px;
    min-width: 240px;
    max-width: 280px;
    flex-shrink: 0;
    scroll-snap-align: start;
    transition: border-color 0.2s;
  }

  .contact-card:hover {
    border-color: var(--brand-indigo, #6366F1);
  }

  .contact-card-avatar {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    background: var(--brand-indigo, #1D1D1F);
    color: white;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    font-size: 14px;
    flex-shrink: 0;
  }

  .contact-card-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .contact-card-name {
    font-weight: 600;
    font-size: 13px;
    color: var(--text-primary);
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .primary-tag {
    font-size: 9px;
    background: var(--brand-indigo, #1D1D1F);
    color: white;
    padding: 1px 6px;
    border-radius: 3px;
    font-weight: 600;
    text-transform: uppercase;
  }

  .contact-card-role {
    font-size: 12px;
    color: var(--text-secondary);
    font-weight: 500;
  }

  .contact-card-detail {
    font-size: 11px;
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .empty-contact {
    border-style: dashed;
    justify-content: center;
    min-width: 200px;
  }

  .empty-contact-text {
    font-size: 13px;
    color: var(--text-muted);
    font-style: italic;
  }

  .brands-list { display: flex; flex-wrap: wrap; gap: 8px; }
  .brand-tag { padding: 4px 8px; background: var(--surface-elevated); border: 1px solid var(--border); border-radius: 4px; font-size: 11px; }

  .empty-text { font-size: 13px; color: var(--text-muted); font-style: italic; }

  .tabs { display: flex; gap: 0; border-bottom: 1px solid var(--border); margin-bottom: 16px; }
  .tab { padding: 12px 20px; background: none; border: none; border-bottom: 2px solid transparent; font-size: 14px; font-weight: 500; color: var(--text-secondary); cursor: pointer; transition: all var(--transition-fast); }
  .tab:hover { color: var(--text-primary); }
  .tab.active { color: var(--text-primary); border-bottom-color: var(--brand-indigo); }

  .metrics-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 16px; }
  .metric { text-align: center; }
  .metric-label { display: block; font-size: 11px; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 4px; }
  .metric-value { font-size: 24px; font-weight: 600; }

  .section-title { font-size: 12px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-secondary); margin: 0 0 16px 0; }

  .list { display: flex; flex-direction: column; gap: 8px; }
  .list-item { display: flex; justify-content: space-between; align-items: center; padding: 12px; background: var(--surface-elevated); border-radius: 6px; transition: all var(--transition-fast); cursor: pointer; }
  .list-item:hover { background: var(--surface-hover); }
  .list-item:focus-visible { outline: 2px solid var(--brand-indigo, #6366F1); outline-offset: -2px; }
  .item-main { display: flex; flex-direction: column; gap: 2px; }
  .item-code { font-weight: 600; font-family: 'JetBrains Mono', monospace; color: var(--brand-indigo); }
  .item-date { font-size: 12px; color: var(--text-muted); }
  .item-due { font-size: 11px; color: var(--text-secondary); }
  .item-meta { display: flex; align-items: center; gap: 16px; }
  .item-status { font-size: 12px; padding: 2px 8px; background: var(--surface); border-radius: 4px; }
  .item-status.status-paid { color: #10B981; }
  .item-status.status-pending { color: #F59E0B; }
  .item-status.status-overdue { color: #EF4444; }
  .item-amount { font-weight: 600; font-family: 'JetBrains Mono', monospace; }

  .issues-list { display: flex; flex-direction: column; gap: 12px; }
  .issue-item { padding: 16px; background: var(--surface-elevated); border-radius: 6px; border-left: 4px solid var(--border); }

  .issue-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8px; }
  .issue-title-row { display: flex; gap: 8px; align-items: center; }
  .issue-code { font-family: 'JetBrains Mono', monospace; font-size: 11px; color: var(--text-muted); }

  .issue-date { font-size: 12px; color: var(--text-muted); }
  .issue-description { margin: 0 0 12px 0; font-size: 14px; line-height: 1.5; color: var(--text-secondary); }
  .issue-cost { font-size: 12px; font-weight: 600; color: #EF4444; margin-bottom: 8px; font-family: 'JetBrains Mono', monospace; }
  .resolved-info { font-size: 12px; color: #10B981; padding: 8px; background: var(--surface); border-radius: 4px; }
  .resolution-text { font-size: 11px; color: var(--text-secondary); font-style: italic; }

  .aging-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px; }
  .aging-item { text-align: center; padding: 12px; background: var(--surface-elevated); border-radius: 8px; }
  .aging-label { display: block; font-size: 11px; color: var(--text-muted); margin-bottom: 4px; }
  .aging-value { font-size: 16px; font-weight: 600; font-family: 'JetBrains Mono', monospace; }
  .aging-value.danger { color: #EF4444; }

  .notes-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
  .notes-list { display: flex; flex-direction: column; gap: 12px; }
  .note-item { padding: 12px; background: var(--surface-elevated); border-radius: 6px; }
  .note-header { display: flex; justify-content: space-between; margin-bottom: 8px; }
  .note-type { font-size: 11px; text-transform: uppercase; padding: 2px 8px; background: var(--brand-indigo-tint); color: var(--brand-indigo); border-radius: 4px; }
  .note-date { font-size: 12px; color: var(--text-muted); }
  .note-content { margin: 0; font-size: 14px; line-height: 1.5; }

  .modal-backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
  .modal { background: var(--surface); padding: 24px; border-radius: 12px; width: 500px; max-height: 80vh; overflow-y: auto; }
  .modal h3 { margin: 0 0 16px 0; }
  .form-group { margin-bottom: 16px; }
  .form-group label { display: block; font-size: 12px; text-transform: uppercase; color: var(--text-muted); margin-bottom: 4px; }
  .form-group input, .form-group select, .form-group textarea { width: 100%; padding: 8px; border: 1px solid var(--border); border-radius: 6px; font-size: 14px; background: var(--surface); color: var(--text-primary); font-family: var(--font-family); }
  .form-group textarea { resize: vertical; }
  .modal-actions { display: flex; justify-content: flex-end; gap: 12px; margin-top: 20px; }
  .edit-form { display: flex; flex-direction: column; gap: 12px; }
  .edit-form .form-row { display: flex; gap: 16px; }
  .edit-form .form-row .form-group { flex: 1; }
  .edit-form .form-group label { display: block; font-size: 11px; font-weight: 500; color: var(--steel, #86868B); margin-bottom: 4px; text-transform: uppercase; letter-spacing: 0.03em; }
  .edit-form .form-group input, .edit-form .form-group select { width: 100%; padding: 8px 10px; border: 1px solid var(--border, #E5E5E5); border-radius: 6px; font-size: 14px; }
  .edit-form .form-group input:focus, .edit-form .form-group select:focus { outline: none; border-color: var(--onyx, #1D1D1F); }
  .form-actions { display: flex; justify-content: flex-end; gap: 12px; margin-top: 8px; }
</style>
