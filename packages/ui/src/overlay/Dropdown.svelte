<script lang="ts">
  /**
   * Dropdown — anchored menu with keyboard roving, viewport-aware flip.
   *
   * This is a micro-interaction (R2 Optimize, 140ms), NOT an entrance.
   * Constitution §4e: opacity + transform only.
   * Constitution §2.6: no focus trap (menus don't trap — they use roving tabindex).
   * Constitution §3: no colored hover states; tint wash only.
   */
  import type { Snippet } from 'svelte';
  import { clickOutside } from '../actions/clickOutside.js';

  export interface DropdownItem {
    id: string;
    label: string;
    /** Renders in danger color. */
    danger?: boolean;
    disabled?: boolean;
    /** Optional leading icon Snippet. */
    icon?: Snippet;
  }

  export interface DropdownProps {
    /** Trigger slot — the element that opens the menu. */
    trigger: Snippet;
    /** Static item list. Provide items OR children Snippet, not both. */
    items?: DropdownItem[];
    /** Free-form content instead of items array. */
    children?: Snippet<[{ close: () => void }]>;
    /** Called when an item is selected (items array mode only). */
    onSelect?: (item: DropdownItem) => void;
    /** Horizontal anchor. Default start (left in LTR). */
    align?: 'start' | 'end';
    /** Disabled — suppresses toggle. */
    disabled?: boolean;
  }

  let {
    trigger,
    items,
    children,
    onSelect,
    align = 'start',
    disabled = false,
  }: DropdownProps = $props();

  let open = $state(false);
  let menuEl = $state<HTMLElement | null>(null);
  let triggerEl = $state<HTMLElement | null>(null);
  let activeIndex = $state(-1);
  // true = menu renders below trigger; false = above (flip)
  let positionBelow = $state(true);

  const menuId = $props.id();

  // Effective (non-disabled) items for roving nav.
  const navigable = $derived(
    (items ?? []).map((item, i) => ({ item, i })).filter(({ item }) => !item.disabled),
  );

  function close() {
    open = false;
    activeIndex = -1;
    // Return focus to trigger.
    triggerEl?.focus();
  }

  function toggle() {
    if (disabled) return;
    if (!open) {
      // Measure flip before opening.
      if (triggerEl) {
        const rect = triggerEl.getBoundingClientRect();
        const spaceBelow = window.innerHeight - rect.bottom;
        // 240px is a reasonable menu max-height threshold.
        positionBelow = spaceBelow >= 240 || spaceBelow > rect.top;
      }
      open = true;
      activeIndex = -1;
    } else {
      close();
    }
  }

  function selectItem(item: DropdownItem) {
    if (item.disabled) return;
    onSelect?.(item);
    close();
  }

  function handleTriggerKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      toggle();
      if (!open) {
        // After opening, set active to first navigable item.
        activeIndex = navigable.length ? navigable[0].i : -1;
      }
    }
    if (e.key === 'ArrowDown' && !open) {
      e.preventDefault();
      toggle();
      activeIndex = navigable.length ? navigable[0].i : -1;
    }
  }

  function handleMenuKeydown(e: KeyboardEvent) {
    if (!open) return;
    const nav = navigable;
    const currentPos = nav.findIndex(({ i }) => i === activeIndex);

    switch (e.key) {
      case 'Escape':
        e.stopPropagation();
        close();
        break;
      case 'ArrowDown':
        e.preventDefault();
        activeIndex = currentPos < nav.length - 1
          ? nav[currentPos + 1].i
          : nav[0].i; // wrap
        break;
      case 'ArrowUp':
        e.preventDefault();
        activeIndex = currentPos > 0
          ? nav[currentPos - 1].i
          : nav[nav.length - 1].i; // wrap
        break;
      case 'Home':
        e.preventDefault();
        if (nav.length) activeIndex = nav[0].i;
        break;
      case 'End':
        e.preventDefault();
        if (nav.length) activeIndex = nav[nav.length - 1].i;
        break;
      case 'Enter':
        e.preventDefault();
        if (activeIndex >= 0 && items?.[activeIndex]) {
          selectItem(items[activeIndex]);
        }
        break;
      case 'Tab':
        // Tab leaves the menu — close it.
        close();
        break;
    }
  }

  // Keep DOM focus in sync with activeIndex when navigating via keyboard.
  $effect(() => {
    if (open && activeIndex >= 0 && menuEl) {
      const btn = menuEl.querySelector<HTMLElement>(
        `[data-dd-index="${activeIndex}"]`,
      );
      btn?.focus({ preventScroll: true });
    }
  });
</script>

<!--
  Wrapper is position:relative so the menu can anchor to it.
  clickOutside listens for pointer-down outside the wrapper.
-->
<!-- a11y: layout container only; keydown is bubbled menu navigation. The interactive children carry roles (trigger role="button", menu items their own). -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="af-dropdown"
  class:af-dropdown--disabled={disabled}
  use:clickOutside={close}
  onkeydown={handleMenuKeydown}
>
  <!-- Trigger wrapper — a11y: button role on the trigger itself (provided by consumer). -->
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    bind:this={triggerEl}
    class="af-dropdown__trigger"
    aria-haspopup="menu"
    aria-expanded={open}
    aria-controls={open ? menuId : undefined}
    onkeydown={handleTriggerKeydown}
    onclick={toggle}
    role="button"
    tabindex={disabled ? -1 : 0}
  >
    {@render trigger()}
  </div>

  {#if open}
    <div
      id={menuId}
      bind:this={menuEl}
      class="af-dropdown__menu"
      class:af-dropdown__menu--above={!positionBelow}
      class:af-dropdown__menu--end={align === 'end'}
      role="menu"
      aria-orientation="vertical"
    >
      {#if items}
        {#each items as item, i}
          <button
            class="af-dropdown__item"
            class:af-dropdown__item--danger={item.danger}
            class:af-dropdown__item--active={i === activeIndex}
            class:af-dropdown__item--disabled={item.disabled}
            role="menuitem"
            tabindex={item.disabled ? -1 : 0}
            data-dd-index={i}
            disabled={item.disabled}
            onclick={() => selectItem(item)}
            type="button"
          >
            {#if item.icon}
              <span class="af-dropdown__item-icon" aria-hidden="true">
                {@render item.icon()}
              </span>
            {/if}
            <span class="af-dropdown__item-label">{item.label}</span>
          </button>
        {/each}
      {:else if children}
        {@render children({ close })}
      {/if}
    </div>
  {/if}
</div>

<style>
  /* ── Wrapper ──────────────────────────────────────────────────────────── */
  .af-dropdown {
    position: relative;
    display: inline-block;
  }

  .af-dropdown--disabled {
    opacity: 0.46;
    pointer-events: none;
  }

  /* ── Trigger ──────────────────────────────────────────────────────────── */
  .af-dropdown__trigger {
    cursor: pointer;
    display: inline-flex;
  }

  /* Touch devices: ensure the trigger meets the WCAG 2.5.5 tap floor.
     Desktop (pointer: fine) sizing is unchanged. */
  @media (pointer: coarse) {
    .af-dropdown__trigger {
      min-height: var(--af-tap-min);
    }
  }

  /* ── Menu panel ───────────────────────────────────────────────────────── */
  .af-dropdown__menu {
    position: absolute;
    /* Default: below-start */
    top: calc(100% + var(--af-space-1));
    inset-inline-start: 0;
    min-width: 180px;
    max-width: 320px;
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    /* §4d: true elevation only on layered surfaces */
    box-shadow: var(--af-shadow-overlay);
    padding: var(--af-space-1);
    z-index: var(--af-z-dropdown);

    /* R2 optimize — this is a micro-interaction, NOT an entrance */
    animation: af-dd-open var(--af-motion-optimize-duration) var(--af-motion-optimize-ease) both;
  }

  .af-dropdown__menu--end {
    inset-inline-start: auto;
    inset-inline-end: 0;
  }

  /* Flip: render above trigger when not enough space below */
  .af-dropdown__menu--above {
    top: auto;
    bottom: calc(100% + var(--af-space-1));
    animation-name: af-dd-open-above;
  }

  @keyframes af-dd-open {
    from {
      opacity: 0;
      transform: translateY(-4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes af-dd-open-above {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  /* ── Items ────────────────────────────────────────────────────────────── */
  .af-dropdown__item {
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
    width: 100%;
    padding: var(--af-space-2) var(--af-space-3);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-regular);
    color: var(--af-text);
    background: transparent;
    border: none;
    border-radius: var(--af-radius-sm);
    cursor: pointer;
    text-align: start;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-dropdown__item:hover:not(.af-dropdown__item--disabled) {
    background: var(--af-tint);
  }

  .af-dropdown__item--active:not(.af-dropdown__item--disabled) {
    background: var(--af-tint-medium);
  }

  .af-dropdown__item--danger {
    color: var(--af-danger);
  }

  .af-dropdown__item--danger:hover:not(.af-dropdown__item--disabled) {
    background: var(--af-danger-tint);
    color: var(--af-danger);
  }

  .af-dropdown__item--disabled {
    opacity: 0.38;
    cursor: not-allowed;
    pointer-events: none;
  }

  /* Touch devices: each menu item meets the WCAG 2.5.5 tap floor.
     Items are full-width rows, so only the vertical axis needs the floor.
     Desktop (pointer: fine) sizing is unchanged. */
  @media (pointer: coarse) {
    .af-dropdown__item {
      min-height: var(--af-tap-min);
    }
  }

  .af-dropdown__item-icon {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    color: var(--af-text-secondary);
  }

  .af-dropdown__item-label {
    flex: 1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* ── Reduced motion ───────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .af-dropdown__menu,
    .af-dropdown__menu--above {
      animation: af-dd-open-reduced var(--af-motion-optimize-duration) var(--af-motion-optimize-ease) both;
    }

    @keyframes af-dd-open-reduced {
      from { opacity: 0; }
      to   { opacity: 1; }
    }
  }
</style>
