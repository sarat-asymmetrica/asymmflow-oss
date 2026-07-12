<script lang="ts">
  import { createBubbler, stopPropagation } from 'svelte/legacy';

  const bubble = createBubbler();
  import Button from '$lib/components/ui/Button.svelte';
  import { toast } from '$lib/stores/toasts';
  import { AddCustomerContact } from '../../../../wailsjs/go/main/CRMService';
  import type { main } from '../../../../wailsjs/go/models';

  interface Props {
    customerId: string;
    profile: main.CustomerFullProfile;
    showContactModal: boolean;
    onSaved: () => void | Promise<void>;
  }

  let { customerId, profile, showContactModal = $bindable(), onSaved }: Props = $props();

  let contactForm = $state({ contact_name: '', job_title: '', email: '', phone: '', address: '', is_primary_contact: false });

  async function saveContact() {
    if (!contactForm.contact_name.trim()) { toast.warning('Name is required'); return; }
    try {
      await AddCustomerContact({ ...contactForm, customer_id: customerId } as any);
      toast.success('Contact added');
      showContactModal = false;
      contactForm = { contact_name: '', job_title: '', email: '', phone: '', address: '', is_primary_contact: false };
      await onSaved();
    } catch (err) {
      toast.danger('Failed to add contact: ' + err);
    }
  }
</script>

<!-- Contacts Strip - Full Width -->
<div class="contacts-strip">
  <div class="contacts-strip-header">
    <h3 class="strip-title">CONTACTS</h3>
    <button class="add-contact-btn-inline" onclick={() => showContactModal = true}>+ Add</button>
  </div>
  <div class="contacts-scroll-container">
    {#if profile.contacts?.length > 0}
      {#each profile.contacts as contact}
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
    {:else}
      <div class="contact-card empty-contact">
        <span class="empty-contact-text">No contacts on file</span>
      </div>
    {/if}
  </div>
</div>

{#if showContactModal}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="modal-backdrop" onclick={() => showContactModal = false}>
    <div class="modal" onclick={stopPropagation(bubble('click'))}>
      <h3>Add Contact</h3>
      <div class="form-group">
        <label for="customer-contact-name">Name *</label>
        <input id="customer-contact-name" type="text" bind:value={contactForm.contact_name} placeholder="Contact person name" />
      </div>
      <div class="form-group">
        <label for="customer-contact-job-title">Job Title</label>
        <input id="customer-contact-job-title" type="text" bind:value={contactForm.job_title} placeholder="e.g. Procurement Manager" />
      </div>
      <div class="form-group">
        <label for="customer-contact-email">Email</label>
        <input id="customer-contact-email" type="email" bind:value={contactForm.email} placeholder="email@company.com" />
      </div>
      <div class="form-group">
        <label for="customer-contact-phone">Phone</label>
        <input id="customer-contact-phone" type="text" bind:value={contactForm.phone} placeholder="+973 1234 5678" />
      </div>
      <div class="form-group">
        <label for="customer-contact-address">Address</label>
        <textarea id="customer-contact-address" bind:value={contactForm.address} rows="2" placeholder="Address"></textarea>
      </div>
      <div class="form-group">
        <label for="customer-contact-primary"><input id="customer-contact-primary" type="checkbox" bind:checked={contactForm.is_primary_contact} /> Primary Contact</label>
      </div>
      <div class="modal-actions">
        <Button variant="ghost" on:click={() => showContactModal = false}>Cancel</Button>
        <Button variant="primary" on:click={saveContact}>Save Contact</Button>
      </div>
    </div>
  </div>
{/if}

<style>
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

  .modal-backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
  .modal { background: var(--surface); padding: 24px; border-radius: 12px; width: 400px; }
  .modal h3 { margin: 0 0 16px 0; }
  .form-group { margin-bottom: 16px; }
  .form-group label { display: block; font-size: 12px; text-transform: uppercase; color: var(--text-muted); margin-bottom: 4px; }
  .form-group textarea { width: 100%; padding: 8px; border: 1px solid var(--border); border-radius: 6px; font-size: 14px; background: var(--surface); color: var(--text-primary); font-family: var(--font-family); }
  .form-group textarea { resize: vertical; }
  .modal-actions { display: flex; justify-content: flex-end; gap: 12px; }
</style>
