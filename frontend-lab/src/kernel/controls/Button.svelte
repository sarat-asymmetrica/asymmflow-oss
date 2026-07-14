<script lang="ts">
  import type { Snippet } from 'svelte'

  type Variant = 'primary' | 'ghost' | 'danger'

  let {
    variant = 'ghost',
    disabled = false,
    onclick,
    children,
  }: {
    variant?: Variant
    disabled?: boolean
    onclick?: (e: MouseEvent) => void
    children: Snippet
  } = $props()
</script>

<button class="k-btn k-btn-{variant}" {disabled} {onclick}>
  {@render children()}
</button>

<style>
  .k-btn {
    font-family: var(--font-ui);
    font-size: calc(13px * var(--ui-font-scale));
    font-weight: 500;
    padding: 7px 14px;
    border-radius: var(--border-radius-sm);
    border: var(--border-width) solid transparent;
    cursor: pointer;
    transition:
      background var(--motion-fast) var(--ease-standard),
      transform var(--motion-fast) var(--ease-standard);
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .k-btn:active:not(:disabled) {
    transform: scale(0.97);
  }
  .k-btn:disabled {
    opacity: 0.45;
    cursor: default;
  }
  .k-btn-primary {
    background: var(--brand-indigo);
    color: var(--canvas);
  }
  .k-btn-primary:hover:not(:disabled) {
    background: var(--brand-indigo-hover);
  }
  .k-btn-ghost {
    background: var(--surface);
    border-color: var(--border);
    color: var(--text-primary);
  }
  .k-btn-ghost:hover:not(:disabled) {
    background: var(--onyx-tint);
  }
  .k-btn-danger {
    background: var(--surface);
    border-color: var(--border);
    color: #b3261e;
  }
  .k-btn-danger:hover:not(:disabled) {
    background: rgba(179, 38, 30, 0.06);
  }
</style>
