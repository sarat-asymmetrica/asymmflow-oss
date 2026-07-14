<script lang="ts">
  import Modal from '../primitives/Modal.svelte'
  import Button from './Button.svelte'

  let {
    title = 'Are you sure?',
    message,
    confirmLabel = 'Confirm',
    danger = true,
    onConfirm,
    onCancel,
  }: {
    title?: string
    message: string
    confirmLabel?: string
    danger?: boolean
    onConfirm: () => void
    onCancel: () => void
  } = $props()
</script>

<Modal {title} onClose={onCancel}>
  <p class="k-confirm-msg">{message}</p>
  {#snippet footer()}
    <Button onclick={onCancel}>Cancel</Button>
    <Button variant={danger ? 'danger' : 'primary'} onclick={onConfirm}>{confirmLabel}</Button>
  {/snippet}
</Modal>

<style>
  .k-confirm-msg {
    color: var(--text-primary);
    font-size: var(--modal-body-size);
    line-height: var(--modal-line-height);
    overflow-wrap: break-word;
  }
</style>
