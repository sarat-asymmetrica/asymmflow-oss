<script lang="ts">
  import Button from '$lib/components/ui/Button.svelte';
  import type { main } from '../../../../wailsjs/go/models';

  interface Props {
    profile: main.CustomerFullProfile | null;
    editMode: boolean;
    customerDeleted: boolean;
    showTaskModal: boolean;
    showDeleteConfirm: boolean;
    goBack: () => void;
    startNewRFQ: () => void;
    startEdit: () => void;
  }

  let {
    profile,
    editMode,
    customerDeleted,
    showTaskModal = $bindable(),
    showDeleteConfirm = $bindable(),
    goBack,
    startNewRFQ,
    startEdit,
  }: Props = $props();
</script>

<header class="header">
  <Button variant="ghost" size="sm" on:click={goBack}>Back to Customers</Button>
  {#if profile}
    <div class="header-info">
      <h1>{profile.business_name}</h1>
      <div class="header-badges">
        <span class="grade-badge grade-{profile.payment_grade?.toLowerCase()}">{profile.payment_grade}</span>
        <span class="type-badge">{profile.customer_type}</span>
        {#if !editMode}
          <Button variant="secondary" size="sm" on:click={startNewRFQ}>New RFQ</Button>
          <Button variant="secondary" size="sm" on:click={() => showTaskModal = true}>Create Task</Button>
          <Button variant="secondary" size="sm" on:click={startEdit}>Edit</Button>
          {#if !customerDeleted}
            <Button variant="danger" size="sm" on:click={() => showDeleteConfirm = true}>Delete</Button>
          {/if}
        {/if}
      </div>
    </div>
  {/if}
</header>

<style>
  .header { margin-bottom: 24px; }

  .header-info { display: flex; justify-content: space-between; align-items: center; }
  .header-info h1 { margin: 0; font-size: 24px; font-weight: 500; }
  .header-badges { display: flex; gap: 8px; }

  .grade-badge { padding: 4px 12px; border-radius: 4px; font-size: 12px; font-weight: 600; }
  .grade-badge.grade-a { background: #DCFCE7; color: #166534; }
  .grade-badge.grade-b { background: #DBEAFE; color: #1E40AF; }
  .grade-badge.grade-c { background: #FEF9C3; color: #854D0E; }
  .grade-badge.grade-d { background: #FEE2E2; color: #991B1B; }

  .type-badge { padding: 4px 12px; background: var(--surface-elevated); border: 1px solid var(--border); border-radius: 4px; font-size: 12px; }
</style>
