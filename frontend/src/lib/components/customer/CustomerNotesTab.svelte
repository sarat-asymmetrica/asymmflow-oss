<script lang="ts">
  import { createBubbler, stopPropagation } from 'svelte/legacy';

  const bubble = createBubbler();
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import { toast } from '$lib/stores/toasts';
  import { formatDate } from './customerFormatters';
  import { AddCustomerNote } from '../../../../wailsjs/go/main/CRMService';
  import type { main } from '../../../../wailsjs/go/models';

  interface Props {
    customerId: string;
    profile: main.CustomerFullProfile;
    showNoteModal: boolean;
    onSaved: () => void | Promise<void>;
  }

  let { customerId, profile, showNoteModal = $bindable(), onSaved }: Props = $props();

  let newNoteType = $state('general');
  let newNoteContent = $state('');

  async function saveNote() {
    if (!newNoteContent.trim()) {
      toast.warning('Please enter note content');
      return;
    }
    try {
      await AddCustomerNote(customerId, newNoteType, newNoteContent);
      toast.success('Note added');
      showNoteModal = false;
      newNoteContent = '';
      await onSaved();
    } catch (err) {
      console.error('Failed to add note:', err);
      toast.danger('Failed to add note');
    }
  }
</script>

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

{#if showNoteModal}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="modal-backdrop" onclick={() => showNoteModal = false}>
    <div class="modal" onclick={stopPropagation(bubble('click'))}>
      <h3>Add Note</h3>
      <div class="form-group">
        <label for="customer-note-type">Type</label>
        <select id="customer-note-type" bind:value={newNoteType}>
          <option value="general">General</option>
          <option value="delivery">Delivery</option>
          <option value="payment">Payment</option>
          <option value="issue">Issue</option>
        </select>
      </div>
      <div class="form-group">
        <label for="customer-note-content">Content</label>
        <textarea id="customer-note-content" bind:value={newNoteContent} rows="4" placeholder="Enter note..."></textarea>
      </div>
      <div class="modal-actions">
        <Button variant="ghost" on:click={() => showNoteModal = false}>Cancel</Button>
        <Button variant="primary" on:click={saveNote}>Save Note</Button>
      </div>
    </div>
  </div>
{/if}

<style>
  .section-title { font-size: 12px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-secondary); margin: 0 0 16px 0; }
  .empty-text { font-size: 13px; color: var(--text-muted); font-style: italic; }

  .notes-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
  .notes-list { display: flex; flex-direction: column; gap: 12px; }
  .note-item { padding: 12px; background: var(--surface-elevated); border-radius: 6px; }
  .note-header { display: flex; justify-content: space-between; margin-bottom: 8px; }
  .note-type { font-size: 11px; text-transform: uppercase; padding: 2px 8px; background: var(--brand-indigo-tint); color: var(--brand-indigo); border-radius: 4px; }
  .note-date { font-size: 12px; color: var(--text-muted); }
  .note-content { margin: 0; font-size: 14px; line-height: 1.5; }

  .modal-backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
  .modal { background: var(--surface); padding: 24px; border-radius: 12px; width: 400px; }
  .modal h3 { margin: 0 0 16px 0; }
  .form-group { margin-bottom: 16px; }
  .form-group label { display: block; font-size: 12px; text-transform: uppercase; color: var(--text-muted); margin-bottom: 4px; }
  .form-group select, .form-group textarea { width: 100%; padding: 8px; border: 1px solid var(--border); border-radius: 6px; font-size: 14px; background: var(--surface); color: var(--text-primary); font-family: var(--font-family); }
  .form-group textarea { resize: vertical; }
  .modal-actions { display: flex; justify-content: flex-end; gap: 12px; }
</style>
