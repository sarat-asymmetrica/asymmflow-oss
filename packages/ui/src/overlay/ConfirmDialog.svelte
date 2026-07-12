<script lang="ts">
  /**
   * ConfirmDialog — composition over Modal.
   *
   * Wraps Modal with a focused, single-purpose layout:
   *   message + (optional) description + cancel + confirm.
   *
   * Autofocus lands on the SAFE action (cancel) per the constitution's
   * §2.6 accessibility floor — never autofocus the destructive action.
   *
   * Destructive mode renders the confirm button in --af-danger; primary
   * mode uses the inverse-surface pattern (same as Button variant=primary).
   *
   * onConfirm / onCancel are synchronous callbacks. The parent controls
   * the open binding.
   */
  import Modal from './Modal.svelte';

  export interface ConfirmDialogProps {
    /** Bindable open state. */
    open?: boolean;
    /** Short summary shown as the dialog title. */
    message: string;
    /** Optional longer explanation shown in the body. */
    description?: string;
    /** Label for the confirm action. Default "Confirm". */
    confirmLabel?: string;
    /** Label for the cancel action. Default "Cancel". */
    cancelLabel?: string;
    /**
     * When true, the confirm button uses --af-danger styling.
     * Use for deletes, irreversible writes, destructive operations.
     */
    destructive?: boolean;
    /** Called when user confirms. Parent should set open = false. */
    onConfirm?: () => void;
    /** Called when user cancels (or closes via scrim / Escape). */
    onCancel?: () => void;
  }

  let {
    open = $bindable(false),
    message,
    description = '',
    confirmLabel = 'Confirm',
    cancelLabel = 'Cancel',
    destructive = false,
    onConfirm,
    onCancel,
  }: ConfirmDialogProps = $props();

  function handleConfirm() {
    onConfirm?.();
  }

  function handleCancel() {
    open = false;
    onCancel?.();
  }
</script>

<!--
  size="sm" — confirm dialogs are intentionally compact.
  closeOnScrim triggers handleCancel so the callback fires on all paths.
  focusTrap initial focus is NOT set here — focusTrap defaults to first focusable.
  The SAFE action (cancel) is rendered first in DOM order and receives initial focus.
-->
<Modal bind:open size="sm" title={message} closeOnScrim={true}>
  {#snippet children()}
    {#if description}
      <p class="af-confirm__desc">{description}</p>
    {/if}
  {/snippet}

  {#snippet footer()}
    <!--
      Cancel first in DOM = first focusable = autofocus on the SAFE action.
      Constitution §2.6: never autofocus the destructive control.
    -->
    <button
      class="af-confirm__btn af-confirm__btn--cancel"
      onclick={handleCancel}
      type="button"
    >
      {cancelLabel}
    </button>
    <button
      class="af-confirm__btn"
      class:af-confirm__btn--destructive={destructive}
      class:af-confirm__btn--primary={!destructive}
      onclick={handleConfirm}
      type="button"
    >
      {confirmLabel}
    </button>
  {/snippet}
</Modal>

<style>
  .af-confirm__desc {
    font-size: var(--af-text-md);
    line-height: var(--af-leading-relaxed);
    color: var(--af-text-secondary);
  }

  /* ── Shared button base ───────────────────────────────────────────────── */
  .af-confirm__btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: var(--af-control-height);
    padding: 0 var(--af-space-4);
    border: 1px solid transparent;
    border-radius: var(--af-radius-sm);
    font-family: var(--af-font-body);
    font-weight: var(--af-weight-semibold);
    font-size: var(--af-text-sm);
    cursor: pointer;
    white-space: nowrap;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      opacity var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-confirm__btn:active {
    transform: scale(0.985);
  }

  /* ── Cancel — secondary ghost style ──────────────────────────────────── */
  .af-confirm__btn--cancel {
    background: var(--af-surface);
    color: var(--af-text);
    border-color: var(--af-border-strong);
  }

  .af-confirm__btn--cancel:hover {
    background: var(--af-surface-raised);
    border-color: var(--af-text-muted);
    box-shadow: var(--af-shadow-sm);
  }

  /* ── Primary confirm — inverse surface (executive monochrome) ─────────── */
  .af-confirm__btn--primary {
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    border-color: var(--af-inverse-surface);
  }

  .af-confirm__btn--primary:hover {
    background: color-mix(in srgb, var(--af-inverse-surface) 88%, transparent);
    box-shadow: var(--af-shadow-lift);
  }

  /* ── Destructive confirm ──────────────────────────────────────────────── */
  .af-confirm__btn--destructive {
    background: var(--af-danger);
    color: var(--af-accent-contrast);
    border-color: var(--af-danger);
  }

  .af-confirm__btn--destructive:hover {
    background: color-mix(in srgb, var(--af-danger) 88%, transparent);
    box-shadow: var(--af-shadow-lift);
  }
</style>
