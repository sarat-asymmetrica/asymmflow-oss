<script lang="ts">
  import type { Snippet } from 'svelte';

  export interface FormGroupProps {
    /** Visible label text — rendered as <label> with correct for/id wiring. */
    label?: string;
    /**
     * The id of the control inside this group.
     * Drives the label's `for` attribute and the hint/error's `aria-describedby`.
     * If not provided the label is rendered as a plain <span> (still visually correct).
     */
    controlId?: string;
    /** Marks the field as required — adds an accessible asterisk to the label. */
    required?: boolean;
    /** Secondary hint rendered below the control when there is no error. */
    hint?: string;
    /**
     * Error message. When non-empty:
     *  - renders the danger-coloured error string
     *  - hides the hint
     *  - the control's aria-describedby should point to the generated error id
     */
    error?: string;
    /** The form control (Input, Select, CurrencyInput, etc.). */
    children?: Snippet;
  }

  let {
    label,
    controlId,
    required = false,
    hint,
    error,
    children,
  }: FormGroupProps = $props();

  const descId = $derived(controlId ? `${controlId}-desc` : undefined);
</script>

<div class="af-form-group" class:af-form-group--error={!!error}>
  {#if label}
    {#if controlId}
      <label class="af-label af-form-group__label" for={controlId}>
        {label}
        {#if required}
          <span class="af-form-group__req" aria-label="required">*</span>
        {/if}
      </label>
    {:else}
      <span class="af-label af-form-group__label">
        {label}
        {#if required}
          <span class="af-form-group__req" aria-label="required">*</span>
        {/if}
      </span>
    {/if}
  {/if}

  <div class="af-form-group__field">
    {@render children?.()}
  </div>

  {#if error}
    <span
      id={descId}
      class="af-form-group__msg af-form-group__msg--error"
      role="alert"
    >{error}</span>
  {:else if hint}
    <span id={descId} class="af-form-group__msg af-form-group__msg--hint">{hint}</span>
  {/if}
</div>

<style>
  /* ── Group container ───────────────────────────────────────────────── */
  .af-form-group {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
    width: 100%;
  }

  /* ── Label (.af-label from base.css: 11px, 600, uppercase, 0.08em) ── */
  .af-form-group__label {
    /* Inherits .af-label typography; just add a touch of bottom space */
    margin-block-end: var(--af-space-1);
  }

  /* Required asterisk */
  .af-form-group__req {
    color: var(--af-danger);
    margin-inline-start: var(--af-space-1);
    font-weight: var(--af-weight-bold);
  }

  /* ── Field slot ────────────────────────────────────────────────────── */
  .af-form-group__field {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
  }

  /* ── Hint / error messages ─────────────────────────────────────────── */
  .af-form-group__msg {
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    line-height: var(--af-leading-base);
  }

  .af-form-group__msg--hint {
    color: var(--af-text-muted);
  }

  .af-form-group__msg--error {
    color: var(--af-danger);
    font-weight: var(--af-weight-medium);
  }
</style>
