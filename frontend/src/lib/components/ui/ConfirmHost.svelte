<script lang="ts">
  /**
   * ConfirmHost — the single rendered surface for the canonical confirm primitive.
   * Mounted once in App.svelte (next to ToastContainer). Screens never render this;
   * they call `confirm.ask(...)` / `confirm.askForReason(...)` and await the result.
   *
   * Design Constitution: Article III.6 (native confirm/prompt banned),
   * Article VI.2 (one canonical component per primitive).
   */
  import Modal from '$lib/components/layout/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import { confirm, type ConfirmVariant } from '$lib/stores/confirm';

  let reason = $state('');

  // Track the last-seen open state so we can clear the reason field on each fresh open.
  let wasOpen = $state(false);
  $effect(() => {
    if ($confirm.open && !wasOpen) {
      reason = '';
    }
    wasOpen = $confirm.open;
  });

  const buttonVariant: Record<ConfirmVariant, 'primary' | 'danger' | 'warning' | 'success'> = {
    primary: 'primary',
    danger: 'danger',
    warning: 'warning',
    success: 'success',
  };

  let confirmDisabled = $derived(
    $confirm.withReason && $confirm.reasonRequired && reason.trim().length === 0,
  );

  function onConfirm() {
    if (confirmDisabled) return;
    confirm._resolve(true, reason.trim());
  }

  function onCancel() {
    confirm._resolve(false, '');
  }
</script>

<Modal
  open={$confirm.open}
  title={$confirm.title}
  size="sm"
  on:close={onCancel}
>
  <div class="confirm-body">
    <p class="confirm-message">{$confirm.message}</p>

    {#if $confirm.withReason}
      <label class="reason-field">
        <span class="reason-label">
          {$confirm.reasonLabel}{#if $confirm.reasonRequired}<span class="req" aria-hidden="true"> *</span>{/if}
        </span>
        <textarea
          class="reason-input"
          rows="3"
          bind:value={reason}
          placeholder={$confirm.reasonPlaceholder}
          aria-required={$confirm.reasonRequired}
        ></textarea>
      </label>
    {/if}
  </div>

  {#snippet footer()}
    <Button variant="secondary" on:click={onCancel}>{$confirm.cancelLabel}</Button>
    <Button
      variant={buttonVariant[$confirm.variant]}
      on:click={onConfirm}
      disabled={confirmDisabled}
    >
      {$confirm.confirmLabel}
    </Button>
  {/snippet}
</Modal>

<style>
  .confirm-body {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-md, 16px);
  }

  .confirm-message {
    margin: 0;
    color: var(--text-primary, #1a1a1a);
    font-size: 14px;
    line-height: 1.5;
    white-space: pre-line;
  }

  .reason-field {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-xs, 6px);
  }

  .reason-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-secondary, #555);
  }

  .req {
    color: var(--color-danger, #ef4444);
  }

  .reason-input {
    width: 100%;
    box-sizing: border-box;
    padding: 8px 10px;
    border: 1px solid var(--border-color, #d0d0d0);
    border-radius: var(--border-radius-sm, 6px);
    font-family: var(--font-family, inherit);
    font-size: 14px;
    color: var(--text-primary, #1a1a1a);
    background: var(--surface, #fff);
    resize: vertical;
  }

  .reason-input:focus {
    outline: none;
    border-color: var(--color-primary, #2563eb);
    box-shadow: 0 0 0 3px var(--color-primary-alpha, rgba(37, 99, 235, 0.15));
  }
</style>
