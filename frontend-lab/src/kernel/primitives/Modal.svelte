<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    title,
    onClose,
    children,
    footer,
  }: {
    title: string
    onClose: () => void
    children: Snippet
    footer?: Snippet
  } = $props()

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onClose()
  }
</script>

<svelte:window onkeydown={onKeydown} />

<!-- The kernel's ONE overlay primitive. Backdrop click + Esc close;
     the panel is a declared scroll region for tall content. -->
<div
  class="k-modal-backdrop"
  onclick={(e) => {
    if (e.target === e.currentTarget) onClose()
  }}
  role="presentation"
>
  <div class="k-modal" role="dialog" aria-modal="true" aria-label={title}>
    <header class="k-modal-head">
      <h2 class="k-modal-title">{title}</h2>
      <button class="k-modal-close" onclick={onClose} aria-label="Close">×</button>
    </header>
    <div class="k-modal-body k-page-content">
      {@render children()}
    </div>
    {#if footer}
      <footer class="k-modal-foot">{@render footer()}</footer>
    {/if}
  </div>
</div>

<style>
  .k-modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.32);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--k-space-lg);
    z-index: 100;
    animation: k-fade var(--motion-base) var(--ease-decelerate);
  }
  .k-modal {
    background: var(--surface);
    border-radius: var(--border-radius-lg);
    box-shadow: var(--shadow-lg);
    width: min(560px, 100%);
    max-height: 100%;
    display: flex;
    flex-direction: column;
    min-width: 0;
    animation: k-rise var(--motion-base) var(--ease-decelerate);
  }
  .k-modal-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--k-space-md);
    padding: var(--card-padding-lg) var(--card-padding-lg) var(--k-space-sm);
    flex-shrink: 0;
  }
  .k-modal-title {
    font-family: var(--font-display);
    font-size: var(--modal-title-size);
    font-weight: var(--modal-title-weight);
    min-width: 0;
    overflow-wrap: break-word;
  }
  .k-modal-close {
    font-size: 20px;
    line-height: 1;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 2px 8px;
    border-radius: var(--border-radius-sm);
    flex-shrink: 0;
  }
  .k-modal-close:hover {
    background: var(--onyx-tint);
  }
  .k-modal-body {
    padding: var(--k-space-sm) var(--card-padding-lg);
    overflow-y: auto;
    min-height: 0;
  }
  .k-modal-foot {
    display: flex;
    justify-content: flex-end;
    gap: var(--k-space-sm);
    padding: var(--k-space-md) var(--card-padding-lg) var(--card-padding-lg);
    flex-shrink: 0;
    flex-wrap: wrap;
  }
  @keyframes k-fade {
    from {
      opacity: 0;
    }
  }
  @keyframes k-rise {
    from {
      opacity: 0;
      transform: translateY(8px);
    }
  }
</style>
