<script lang="ts">
  /* ViewSwitcher — a segmented view/mode control. The kernel's answer to the
   * hand-rolled left-nav / tab-bar / mode-switcher that recurs across console
   * screens (Accounting's GL/CoA/Journal/Reports, Payroll's compensation/runs/
   * payouts, the operational hubs). Screens declare views + an active key and
   * render the selected section themselves in a Stack — this owns ONLY the nav
   * chrome + active state (L1), never the content layout, so it composes into
   * any screen without dictating structure.
   *
   * Horizontal (default) = a segmented tab bar above the content; vertical =
   * a left rail. Horizontal is the kernel-preferred form (dodges fixed-sidebar
   * width math and keeps content full-bleed); vertical is available for genuine
   * long nav lists. */
  import type { Tone } from '../tones'

  let {
    views,
    activeKey,
    onSelect,
    orientation = 'horizontal',
    ariaLabel = 'Views',
  }: {
    views: { key: string; label: string; badge?: string | number; badgeTone?: Tone }[]
    activeKey: string
    onSelect: (key: string) => void
    orientation?: 'horizontal' | 'vertical'
    ariaLabel?: string
  } = $props()
</script>

<nav class="k-vs k-vs-{orientation}" aria-label={ariaLabel}>
  {#each views as v (v.key)}
    <button
      class="k-vs-item"
      class:active={v.key === activeKey}
      aria-current={v.key === activeKey ? 'page' : undefined}
      onclick={() => onSelect(v.key)}
    >
      <span class="k-vs-label">{v.label}</span>
      {#if v.badge != null && v.badge !== ''}
        <span
          class="k-vs-badge"
          style:background={v.badgeTone ? `var(--k-tone-${v.badgeTone}-bg)` : undefined}
          style:color={v.badgeTone ? `var(--k-tone-${v.badgeTone}-fg)` : undefined}
        >{v.badge}</span>
      {/if}
    </button>
  {/each}
</nav>

<style>
  .k-vs {
    display: flex;
    min-width: 0;
    gap: var(--k-space-xs);
  }
  .k-vs-horizontal {
    flex-direction: row;
    flex-wrap: wrap;
    border-bottom: var(--border-width) solid var(--border);
  }
  .k-vs-vertical {
    flex-direction: column;
    flex: 0 0 auto;
  }
  .k-vs-item {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-xs);
    font-family: var(--font-ui);
    font-size: calc(13px * var(--ui-font-scale));
    font-weight: 500;
    color: var(--text-secondary);
    background: transparent;
    border: none;
    padding: 8px 12px;
    cursor: pointer;
    max-width: 100%;
    white-space: nowrap;
    transition: color var(--motion-fast) var(--ease-standard);
  }
  .k-vs-horizontal .k-vs-item {
    border-bottom: 2px solid transparent;
    margin-bottom: calc(-1 * var(--border-width));
  }
  .k-vs-vertical .k-vs-item {
    justify-content: space-between;
    text-align: left;
    border-radius: var(--border-radius-sm);
    border-left: 2px solid transparent;
  }
  .k-vs-item:hover {
    color: var(--text-primary);
  }
  .k-vs-horizontal .k-vs-item.active {
    color: var(--text-primary);
    font-weight: 600;
    border-bottom-color: var(--onyx);
  }
  .k-vs-vertical .k-vs-item.active {
    color: var(--text-primary);
    font-weight: 600;
    background: var(--onyx-tint);
    border-left-color: var(--onyx);
  }
  .k-vs-label {
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .k-vs-badge {
    font-size: calc(11px * var(--ui-font-scale));
    font-weight: 600;
    line-height: 1;
    padding: 2px 7px;
    border-radius: var(--border-radius-pill);
    background: var(--onyx-tint);
    color: var(--text-secondary);
    flex-shrink: 0;
  }
</style>
