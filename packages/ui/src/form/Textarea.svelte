<script lang="ts">
  import type { HTMLTextareaAttributes } from 'svelte/elements';

  export interface TextareaProps extends Omit<HTMLTextareaAttributes, 'value'> {
    /** Bound text content. */
    value?: string;
    /** Number of visible text rows (ignored when autoResize is true). */
    rows?: number;
    /** When true, expands vertically to fit content; disables manual resize. */
    autoResize?: boolean;
    /** When true renders the danger border and sets aria-invalid. */
    invalid?: boolean;
  }

  let {
    value = $bindable(''),
    rows = 3,
    autoResize = false,
    disabled = false,
    readonly = false,
    invalid = false,
    id,
    class: extraClass,
    ...restProps
  }: TextareaProps = $props();

  const generatedId = $props.id();
  const uid = $derived(id ?? generatedId);

  let el: HTMLTextAreaElement | undefined = $state();

  function resize() {
    if (!autoResize || !el) return;
    el.style.height = 'auto';
    el.style.height = `${el.scrollHeight}px`;
  }

  $effect(() => {
    // Run resize whenever value changes (covers external $bindable updates too)
    if (autoResize && el) {
      // Microtask so the DOM has updated
      Promise.resolve().then(resize);
    }
  });
</script>

<textarea
  bind:this={el}
  id={uid}
  bind:value
  {rows}
  {disabled}
  {readonly}
  aria-invalid={invalid || undefined}
  class="af-textarea {extraClass ?? ''}"
  class:af-textarea--invalid={invalid}
  class:af-textarea--disabled={disabled}
  class:af-textarea--readonly={readonly}
  class:af-textarea--autoresize={autoResize}
  oninput={resize}
  {...restProps}
></textarea>

<style>
  .af-textarea {
    display: block;
    width: 100%;
    min-height: calc(var(--af-control-height) * 2);
    padding: var(--af-space-2) var(--af-space-3);
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    color: var(--af-text);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    line-height: var(--af-leading-base);
    resize: vertical;
    outline: none;
    transition:
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-textarea::placeholder {
    color: var(--af-text-muted);
  }

  .af-textarea:hover:not(:disabled):not(:read-only) {
    border-color: var(--af-border-strong);
  }

  .af-textarea:focus-visible {
    border-color: var(--af-accent);
    box-shadow: 0 0 0 3px var(--af-accent-tint);
    /* outline suppressed — wrapper draws the ring via box-shadow */
    outline: none;
  }

  .af-textarea--invalid {
    border-color: var(--af-danger);
  }

  .af-textarea--invalid:focus-visible {
    border-color: var(--af-danger);
    box-shadow: 0 0 0 3px var(--af-danger-tint);
  }

  .af-textarea--disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .af-textarea--readonly {
    background: var(--af-surface-raised);
    resize: none;
  }

  .af-textarea--autoresize {
    resize: none;
    overflow: hidden;
  }
</style>
