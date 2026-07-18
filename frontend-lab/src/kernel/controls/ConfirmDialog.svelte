<script lang="ts">
  import Modal from '../primitives/Modal.svelte'
  import Button from './Button.svelte'

  let {
    title = 'Are you sure?',
    message,
    confirmLabel = 'Confirm',
    danger = true,
    reasonLabel,
    reasonPlaceholder = '',
    requireReason = false,
    onConfirm,
    onCancel,
  }: {
    title?: string
    message: string
    confirmLabel?: string
    danger?: boolean
    /** When set, the dialog shows a reason textarea and passes its value to
     * onConfirm — for HOT-ZONE mutations that carry a mandatory audit reason
     * (project delete/shelve, employee archive). */
    reasonLabel?: string
    reasonPlaceholder?: string
    /** Disable Confirm until a non-empty reason is entered. */
    requireReason?: boolean
    onConfirm: (reason?: string) => void
    onCancel: () => void
  } = $props()

  let reason = $state('')
  const blocked = $derived(requireReason && reason.trim() === '')
</script>

<Modal {title} onClose={onCancel}>
  <p class="k-confirm-msg">{message}</p>
  {#if reasonLabel}
    <label class="k-field k-confirm-reason">
      <span class="k-field-label">{reasonLabel}{#if requireReason}<span aria-hidden="true"> *</span>{/if}</span>
      <textarea class="k-input k-input-area" bind:value={reason} placeholder={reasonPlaceholder}></textarea>
    </label>
  {/if}
  {#snippet footer()}
    <Button onclick={onCancel}>Cancel</Button>
    <Button
      variant={danger ? 'danger' : 'primary'}
      disabled={blocked}
      onclick={() => onConfirm(reasonLabel ? reason.trim() : undefined)}
    >{confirmLabel}</Button>
  {/snippet}
</Modal>

<style>
  .k-confirm-msg {
    color: var(--text-primary);
    font-size: var(--modal-body-size);
    line-height: var(--modal-line-height);
    overflow-wrap: break-word;
  }
  .k-confirm-reason {
    margin-top: var(--k-space-md);
  }
</style>
