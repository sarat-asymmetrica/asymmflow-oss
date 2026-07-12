<script lang="ts">
  import { Dropdown, Tooltip } from '@asymmflow/ui';
  import type { DropdownItem } from '@asymmflow/ui';

  // ── Dropdown demos ───────────────────────────────────────────────────────
  let lastSelected = $state<string | null>(null);

  const actionItems: DropdownItem[] = [
    { id: 'edit', label: 'Edit record' },
    { id: 'duplicate', label: 'Duplicate' },
    { id: 'export', label: 'Export as CSV' },
    { id: 'archive', label: 'Archive', disabled: true },
    { id: 'delete', label: 'Delete', danger: true },
  ];

  const contextItems: DropdownItem[] = [
    { id: 'view', label: 'View details' },
    { id: 'approve', label: 'Approve' },
    { id: 'reject', label: 'Reject' },
    { id: 'reassign', label: 'Reassign' },
  ];

  function handleSelect(item: DropdownItem) {
    lastSelected = item.label;
  }
</script>

<div class="sections">
  <!-- ── Dropdown intro ─────────────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">Dropdown</h2>
    <p class="intro">
      Anchored menu. Trigger Snippet + items array or free-form children.
      clickOutside + Escape dismiss. Arrow keys rove; Home/End jump; Enter selects.
      Viewport-aware flip: if there's insufficient space below, the menu renders above.
      R2 Optimize 140ms — a micro-interaction, not an entrance.
    </p>
  </section>

  {#if lastSelected}
    <div class="result-notice">
      Last selection: <strong>{lastSelected}</strong>
    </div>
  {/if}

  <!-- ── Basic actions ──────────────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">Actions menu</h2>
    <p class="intro">
      Standard record-action pattern. The danger item renders in --af-danger;
      the disabled item is unreachable by pointer and keyboard.
    </p>
    <div class="demo-stage card">
      <Dropdown items={actionItems} onSelect={handleSelect}>
        {#snippet trigger()}
          <button class="menu-btn" type="button">
            Actions
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
              <path d="M2.5 4.5L6 8L9.5 4.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
        {/snippet}
      </Dropdown>
    </div>
  </section>

  <!-- ── End-aligned ───────────────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">End-aligned (right in LTR)</h2>
    <p class="intro">
      Pass <code>align="end"</code> when the trigger is at the end of a row —
      the menu aligns its trailing edge to the trigger's trailing edge.
    </p>
    <div class="demo-stage demo-stage--end card">
      <Dropdown items={contextItems} align="end" onSelect={handleSelect}>
        {#snippet trigger()}
          <button class="menu-btn menu-btn--ghost" type="button" aria-label="Row actions">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <circle cx="8" cy="3" r="1.25" fill="currentColor"/>
              <circle cx="8" cy="8" r="1.25" fill="currentColor"/>
              <circle cx="8" cy="13" r="1.25" fill="currentColor"/>
            </svg>
          </button>
        {/snippet}
      </Dropdown>
    </div>
  </section>

  <!-- ── Custom children ───────────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">Custom children Snippet</h2>
    <p class="intro">
      When items arrays aren't expressive enough, pass a children Snippet.
      The close callback is provided via the render argument.
    </p>
    <div class="demo-stage card">
      <Dropdown>
        {#snippet trigger()}
          <button class="menu-btn" type="button">
            Filter by status
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
              <path d="M2.5 4.5L6 8L9.5 4.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
        {/snippet}
        {#snippet children({ close })}
          <div class="custom-menu">
            <div class="af-label custom-menu__group">Status</div>
            {#each ['All', 'Draft', 'Pending', 'Approved', 'Paid'] as status}
              <button
                class="custom-menu__item"
                type="button"
                onclick={() => { lastSelected = status; close(); }}
              >
                {status}
              </button>
            {/each}
          </div>
        {/snippet}
      </Dropdown>
    </div>
  </section>

  <!-- ── Tooltip intro ─────────────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">Tooltip</h2>
    <p class="intro">
      Hover or focus a trigger to reveal a tooltip after ~500ms. Instant out.
      Inverse surface, text-inverse, text-xs, max 36ch. Never traps pointer.
      role=tooltip + aria-describedby wired by the component.
    </p>
  </section>

  <!-- ── Position variants ─────────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">Positions</h2>
    <div class="demo-stage demo-stage--tall card">
      <div class="tooltip-grid">

        <Tooltip text="Above the trigger — default position" position="top">
          <button class="tip-target" type="button">top (default)</button>
        </Tooltip>

        <Tooltip text="Below the trigger" position="bottom">
          <button class="tip-target" type="button">bottom</button>
        </Tooltip>

        <Tooltip text="To the right of the trigger" position="right">
          <button class="tip-target" type="button">right</button>
        </Tooltip>

        <Tooltip text="To the left of the trigger" position="left">
          <button class="tip-target" type="button">left</button>
        </Tooltip>

      </div>
    </div>
  </section>

  <!-- ── Content length ────────────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">Long content (max 36ch)</h2>
    <p class="intro">
      The 36ch cap keeps tooltips legible. Longer strings wrap; they never
      overflow the viewport in normal usage.
    </p>
    <div class="demo-stage card">
      <Tooltip
        text="This field is required for compliance with the Bahrain VAT Return filing. Leave blank only if the vendor is exempt."
        position="top"
      >
        <button class="tip-target" type="button">Hover for compliance note</button>
      </Tooltip>
    </div>
  </section>

  <!-- ── On an icon button ─────────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">On an icon-only button</h2>
    <p class="intro">
      The most important Tooltip use case: annotating icon-only controls that
      have no visible label. Hover the button below.
    </p>
    <div class="demo-stage card">
      <Tooltip text="Download as CSV" position="top">
        <button class="icon-btn" type="button" aria-label="Download as CSV">
          <svg width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden="true">
            <path d="M9 2v10M5 8l4 4 4-4M3 14h12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </button>
      </Tooltip>

      <Tooltip text="Print preview" position="top">
        <button class="icon-btn" type="button" aria-label="Print preview">
          <svg width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden="true">
            <rect x="4" y="7" width="10" height="7" rx="1" stroke="currentColor" stroke-width="1.5"/>
            <path d="M6 7V4h6v3M6 11h6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        </button>
      </Tooltip>

      <Tooltip text="Share link (copied on click)" position="top">
        <button class="icon-btn" type="button" aria-label="Share">
          <svg width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden="true">
            <circle cx="14" cy="4" r="2" stroke="currentColor" stroke-width="1.5"/>
            <circle cx="4" cy="9" r="2" stroke="currentColor" stroke-width="1.5"/>
            <circle cx="14" cy="14" r="2" stroke="currentColor" stroke-width="1.5"/>
            <path d="M6 8l6-3M6 10l6 3" stroke="currentColor" stroke-width="1.5"/>
          </svg>
        </button>
      </Tooltip>
    </div>
  </section>
</div>

<style>
  .sections {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-6);
  }

  .intro {
    color: var(--af-text-secondary);
    font-size: var(--af-text-md);
    max-width: 64ch;
    margin-top: var(--af-space-2);
    margin-bottom: var(--af-space-4);
  }

  code {
    font-family: 'Courier New', monospace;
    font-size: 0.9em;
    background: var(--af-surface-sunken);
    padding: 1px var(--af-space-1);
    border-radius: 4px;
  }

  /* ── Demo stages ─────────────────────────────────────────────────────── */
  .card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-card-padding);
  }

  .demo-stage {
    display: flex;
    align-items: center;
    gap: var(--af-space-4);
    background: var(--af-surface-raised);
    min-height: 88px;
  }

  .demo-stage--end {
    justify-content: flex-end;
  }

  .demo-stage--tall {
    min-height: 140px;
    justify-content: center;
  }

  /* ── Menu trigger buttons ────────────────────────────────────────────── */
  .menu-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--af-space-2);
    min-height: var(--af-control-height);
    padding: 0 var(--af-space-3);
    background: var(--af-surface);
    color: var(--af-text);
    border: 1px solid var(--af-border-strong);
    border-radius: var(--af-radius-sm);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .menu-btn:hover {
    background: var(--af-surface-raised);
    box-shadow: var(--af-shadow-sm);
  }

  .menu-btn:active { transform: scale(0.985); }

  .menu-btn--ghost {
    background: transparent;
    border-color: transparent;
    color: var(--af-text-secondary);
  }

  .menu-btn--ghost:hover {
    background: var(--af-tint);
    color: var(--af-text);
    box-shadow: none;
  }

  /* ── Custom menu content ─────────────────────────────────────────────── */
  .custom-menu {
    padding: var(--af-space-1);
  }

  .custom-menu__group {
    padding: var(--af-space-2) var(--af-space-3) var(--af-space-1);
  }

  .custom-menu__item {
    display: flex;
    width: 100%;
    padding: var(--af-space-2) var(--af-space-3);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    color: var(--af-text);
    background: transparent;
    border: none;
    border-radius: var(--af-radius-sm);
    cursor: pointer;
    text-align: start;
    transition: background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .custom-menu__item:hover {
    background: var(--af-tint);
  }

  /* ── Tooltip demo targets ────────────────────────────────────────────── */
  .tooltip-grid {
    display: flex;
    flex-wrap: wrap;
    gap: var(--af-space-5);
    justify-content: center;
    align-items: center;
    padding: var(--af-space-5) 0;
  }

  .tip-target {
    min-height: var(--af-control-height);
    padding: 0 var(--af-space-4);
    background: var(--af-surface);
    color: var(--af-text);
    border: 1px solid var(--af-border-strong);
    border-radius: var(--af-radius-sm);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    cursor: default;
    transition: border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .tip-target:hover {
    border-color: var(--af-accent);
  }

  /* ── Icon buttons ────────────────────────────────────────────────────── */
  .icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    background: var(--af-surface);
    color: var(--af-text-secondary);
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .icon-btn:hover {
    background: var(--af-tint);
    color: var(--af-text);
    border-color: var(--af-border-strong);
  }

  .icon-btn:active { transform: scale(0.94); }

  /* ── Result notice ───────────────────────────────────────────────────── */
  .result-notice {
    display: inline-flex;
    padding: var(--af-space-2) var(--af-space-3);
    background: var(--af-surface-raised);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    font-size: var(--af-text-sm);
    color: var(--af-text-secondary);
  }
</style>
