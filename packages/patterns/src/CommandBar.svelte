<script lang="ts">
  /**
   * CommandBar — the Ctrl+K command palette.
   *
   * Constitution §2.6: focus containment via focusTrap action.
   * Constitution §4d: frosted glass surface (--af-glass-bg + blur + --af-shadow-overlay).
   * Constitution §4e: R2 optimize open motion — must feel instant (140ms).
   * Constitution §3: no emoji-as-icons in product UI.
   *
   * Composites:
   *   - focusTrap from @asymmflow/ui (focus containment)
   *   - portal from @asymmflow/ui (escape stacking contexts)
   *
   * ARIA: combobox/listbox semantics (role="combobox" + role="listbox").
   */

  import type { Snippet } from 'svelte';
  import { focusTrap, portal } from '@asymmflow/ui';

  // ─── Public interface ────────────────────────────────────────────────────────

  export interface Command {
    id: string;
    label: string;
    group?: string;
    hint?: string;
    /** Optional icon Snippet rendered before the label. */
    icon?: Snippet;
    action: () => void;
  }

  export interface CommandBarProps {
    /** Bindable open state. */
    open?: boolean;
    /** All available commands. */
    commands?: Command[];
    /**
     * When true, registers a global keydown listener for Ctrl+K / Cmd+K.
     * Default: true.
     */
    registerGlobalShortcut?: boolean;
  }

  // ─── Props ───────────────────────────────────────────────────────────────────

  let {
    open = $bindable(false),
    commands = [],
    registerGlobalShortcut = true,
  }: CommandBarProps = $props();

  // ─── Recent commands (in-module memory, max 5) ───────────────────────────────

  let recentIds = $state<string[]>([]);

  function recordRecent(id: string) {
    recentIds = [id, ...recentIds.filter((r) => r !== id)].slice(0, 5);
  }

  // ─── Search ──────────────────────────────────────────────────────────────────

  let query = $state('');

  const baseId = $props.id();
  const inputId = `${baseId}-input`;
  const listId  = `${baseId}-list`;

  /** Flatten commands, applying search filter. */
  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (!q) return commands;
    return commands.filter(
      (c) =>
        c.label.toLowerCase().includes(q) ||
        c.group?.toLowerCase().includes(q) ||
        c.hint?.toLowerCase().includes(q),
    );
  });

  /**
   * Grouped results for rendering.
   * When there is a search query: flat list with implicit group.
   * When empty query: "Recent" first (if any), then groups.
   */
  const grouped = $derived.by((): { group: string; items: Command[] }[] => {
    if (query.trim()) {
      if (!filtered.length) return [];
      return [{ group: 'Results', items: filtered }];
    }

    const recents = recentIds
      .map((id) => commands.find((c) => c.id === id))
      .filter((c): c is Command => !!c);

    const groups = new Map<string, Command[]>();

    for (const cmd of commands) {
      const g = cmd.group ?? 'Commands';
      if (!groups.has(g)) groups.set(g, []);
      groups.get(g)!.push(cmd);
    }

    const result: { group: string; items: Command[] }[] = [];
    if (recents.length) result.push({ group: 'Recent', items: recents });
    for (const [g, items] of groups) result.push({ group: g, items });

    return result;
  });

  /** Flat ordered list for keyboard navigation. */
  const flatList = $derived(grouped.flatMap((g) => g.items));

  // ─── Keyboard navigation ─────────────────────────────────────────────────────

  let activeIndex = $state(-1);

  $effect(() => {
    // Reset active index whenever filtered results change.
    activeIndex = flatList.length > 0 ? 0 : -1;
  });

  const activeItemId = $derived(
    activeIndex >= 0 && activeIndex < flatList.length
      ? `af-cb-item-${flatList[activeIndex].id}`
      : undefined,
  );

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      close();
      return;
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      activeIndex = Math.min(activeIndex + 1, flatList.length - 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      activeIndex = Math.max(activeIndex - 1, 0);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (activeIndex >= 0 && flatList[activeIndex]) {
        execute(flatList[activeIndex]);
      }
    }
  }

  // ─── Open / close ─────────────────────────────────────────────────────────────

  function open_() {
    open = true;
    query = '';
  }

  function close() {
    open = false;
  }

  function handleScrimPointer(e: PointerEvent) {
    if (e.target === e.currentTarget) close();
  }

  function execute(cmd: Command) {
    recordRecent(cmd.id);
    close();
    cmd.action();
  }

  // ─── Global Ctrl+K shortcut ───────────────────────────────────────────────────

  $effect(() => {
    if (!registerGlobalShortcut) return;

    function onGlobalKey(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        if (open) {
          close();
        } else {
          open_();
        }
      }
    }

    window.addEventListener('keydown', onGlobalKey);
    return () => window.removeEventListener('keydown', onGlobalKey);
  });

  // Body scroll lock
  $effect(() => {
    if (open) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  });
</script>

{#if open}
  <div use:portal>
    <!-- Scrim: click-outside closes -->
    <div
      class="af-cb-scrim"
      role="presentation"
      onpointerdown={handleScrimPointer}
    >
      <!--
        Dialog with combobox/listbox semantics.
        focusTrap keeps Tab inside; onEscape closes.
      -->
      <div
        class="af-cb"
        role="dialog"
        aria-modal="true"
        aria-label="Command bar"
        use:focusTrap={{ active: open, onEscape: close }}
      >
        <!-- Search input (combobox) -->
        <div class="af-cb__search-row" role="search">
          <!-- Magnifier icon — stroked SVG, no emoji -->
          <svg
            class="af-cb__search-icon"
            width="16"
            height="16"
            viewBox="0 0 16 16"
            fill="none"
            aria-hidden="true"
          >
            <circle cx="7" cy="7" r="4.5" stroke="currentColor" stroke-width="1.4" />
            <path d="M10.5 10.5L13.5 13.5" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
          </svg>

          <input
            id={inputId}
            type="text"
            class="af-cb__input"
            placeholder="Search commands…"
            autocomplete="off"
            autocorrect="off"
            spellcheck={false}
            role="combobox"
            aria-expanded={flatList.length > 0}
            aria-controls={listId}
            aria-activedescendant={activeItemId}
            aria-autocomplete="list"
            bind:value={query}
            onkeydown={handleKeydown}
          />

          <!-- Escape hint badge -->
          <kbd class="af-cb__esc-hint" aria-hidden="true">Esc</kbd>
        </div>

        <!-- Results listbox -->
        <div
          id={listId}
          role="listbox"
          aria-label="Commands"
          class="af-cb__list"
        >
          {#if grouped.length === 0}
            <div class="af-cb__empty">
              <svg width="28" height="28" viewBox="0 0 28 28" fill="none" aria-hidden="true">
                <circle cx="14" cy="14" r="10" stroke="currentColor" stroke-width="1.2" opacity="0.3" />
                <path d="M10 14h5M12 11l3 3-3 3" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" opacity="0.5" />
              </svg>
              <span class="af-cb__empty-text">No commands match</span>
            </div>
          {:else}
            {#each grouped as grp, gi}
              {@const groupOffset = grouped.slice(0, gi).reduce((n, g) => n + g.items.length, 0)}
              <div class="af-cb__group" role="presentation">
                <div class="af-cb__group-label af-label" role="presentation">{grp.group}</div>
                {#each grp.items as cmd, ci}
                  {@const flatIdx = groupOffset + ci}
                  {@const isActive = flatIdx === activeIndex}
                  <div
                    id="af-cb-item-{cmd.id}"
                    class="af-cb__item"
                    class:af-cb__item--active={isActive}
                    role="option"
                    tabindex={-1}
                    aria-selected={isActive}
                    onpointerenter={() => { activeIndex = flatIdx; }}
                    onpointerdown={(e) => { e.preventDefault(); execute(cmd); }}
                  >
                    {#if cmd.icon}
                      <span class="af-cb__item-icon" aria-hidden="true">
                        {@render cmd.icon()}
                      </span>
                    {/if}
                    <span class="af-cb__item-label">{cmd.label}</span>
                    {#if cmd.hint}
                      <span class="af-cb__item-hint af-meta">{cmd.hint}</span>
                    {/if}
                    {#if recentIds.includes(cmd.id) && !query.trim()}
                      <span class="af-cb__item-recent-dot" aria-hidden="true" title="Recently used"></span>
                    {/if}
                  </div>
                {/each}
              </div>
            {/each}
          {/if}
        </div>

        <!-- Footer: keyboard hint row -->
        <div class="af-cb__footer" aria-hidden="true">
          <span class="af-cb__hint-group">
            <kbd>↑</kbd><kbd>↓</kbd>
            <span class="af-cb__hint-text">Navigate</span>
          </span>
          <span class="af-cb__hint-group">
            <kbd>Enter</kbd>
            <span class="af-cb__hint-text">Run</span>
          </span>
          <span class="af-cb__hint-group">
            <kbd>Esc</kbd>
            <span class="af-cb__hint-text">Close</span>
          </span>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  /* ── Scrim ─────────────────────────────────────────────────────────────── */
  .af-cb-scrim {
    position: fixed;
    inset: 0;
    background: var(--af-scrim);
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: clamp(var(--af-space-6), 12vh, 160px);
    z-index: var(--af-z-modal);
    /* R2 optimize — palette must feel instant */
    animation: af-cb-scrim-in var(--af-motion-optimize-duration) var(--af-motion-optimize-ease) both;
  }

  @keyframes af-cb-scrim-in {
    from { opacity: 0; }
    to   { opacity: 1; }
  }

  /* ── Panel — frosted glass surface ─────────────────────────────────────── */
  .af-cb {
    width: min(640px, calc(100vw - var(--af-space-5)));
    background: var(--af-glass-bg);
    backdrop-filter: var(--af-glass-blur);
    -webkit-backdrop-filter: var(--af-glass-blur);
    border: 1px solid var(--af-glass-border);
    border-radius: var(--af-radius-lg);
    box-shadow: var(--af-shadow-overlay);
    overflow: hidden;
    /* R2 optimize entrance — instant but not jarring */
    animation: af-cb-panel-in var(--af-motion-optimize-duration) var(--af-motion-optimize-ease) both;
  }

  @keyframes af-cb-panel-in {
    from {
      opacity: 0;
      transform: translateY(-8px) scale(0.98);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  /* ── Search row ─────────────────────────────────────────────────────────── */
  .af-cb__search-row {
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
    padding: var(--af-space-3) var(--af-space-4);
    border-bottom: 1px solid var(--af-border);
  }

  .af-cb__search-icon {
    color: var(--af-text-muted);
    flex-shrink: 0;
  }

  .af-cb__input {
    flex: 1;
    border: none;
    background: transparent;
    font-family: var(--af-font-body);
    font-size: var(--af-text-md);
    color: var(--af-text);
    outline: none;
    min-width: 0;
  }

  .af-cb__input::placeholder {
    color: var(--af-text-muted);
  }

  .af-cb__esc-hint {
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    color: var(--af-text-muted);
    background: var(--af-surface-sunken);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    padding: 2px var(--af-space-2);
    flex-shrink: 0;
  }

  /* ── List ───────────────────────────────────────────────────────────────── */
  .af-cb__list {
    max-height: 360px;
    overflow-y: auto;
    padding: var(--af-space-2) 0;
    scrollbar-width: thin;
    scrollbar-color: var(--af-border-strong) transparent;
  }

  .af-cb__list::-webkit-scrollbar { width: 4px; }
  .af-cb__list::-webkit-scrollbar-track { background: transparent; }
  .af-cb__list::-webkit-scrollbar-thumb {
    background: var(--af-border-strong);
    border-radius: var(--af-radius-pill);
  }

  /* ── Group ──────────────────────────────────────────────────────────────── */
  .af-cb__group {
    padding: var(--af-space-1) 0;
  }

  .af-cb__group-label {
    padding: var(--af-space-1) var(--af-space-4);
    /* .af-label: 11px, 600, uppercase, 0.08em — from base.css */
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
    color: var(--af-text-muted);
    user-select: none;
  }

  /* ── Item ───────────────────────────────────────────────────────────────── */
  .af-cb__item {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
    padding: var(--af-space-2) var(--af-space-4);
    margin: 0 var(--af-space-2);
    border-radius: var(--af-radius-sm);
    cursor: pointer;
    user-select: none;
    min-height: 40px;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
    position: relative;
  }

  .af-cb__item--active {
    background: var(--af-tint-medium);
  }

  .af-cb__item-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    color: var(--af-text-secondary);
    flex-shrink: 0;
  }

  .af-cb__item-label {
    flex: 1;
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    color: var(--af-text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .af-cb__item-hint {
    font-size: var(--af-text-xs);
    color: var(--af-text-muted);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .af-cb__item-recent-dot {
    width: 5px;
    height: 5px;
    border-radius: var(--af-radius-pill);
    background: var(--af-accent);
    flex-shrink: 0;
    opacity: 0.6;
  }

  /* ── Empty ──────────────────────────────────────────────────────────────── */
  .af-cb__empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--af-space-3);
    padding: var(--af-space-6) var(--af-space-4);
    color: var(--af-text-muted);
  }

  .af-cb__empty-text {
    font-size: var(--af-text-sm);
    color: var(--af-text-muted);
  }

  /* ── Footer hint bar ────────────────────────────────────────────────────── */
  .af-cb__footer {
    display: flex;
    align-items: center;
    gap: var(--af-space-4);
    padding: var(--af-space-2) var(--af-space-4);
    border-top: 1px solid var(--af-border);
    background: var(--af-surface-raised);
  }

  .af-cb__hint-group {
    display: inline-flex;
    align-items: center;
    gap: var(--af-space-1);
  }

  .af-cb__hint-text {
    font-size: var(--af-text-xs);
    color: var(--af-text-muted);
  }

  kbd {
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    color: var(--af-text-secondary);
    background: var(--af-surface);
    border: 1px solid var(--af-border-strong);
    border-radius: 4px;
    padding: 1px var(--af-space-1);
    line-height: 1.4;
  }

  /* ── Reduced motion ─────────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .af-cb-scrim,
    .af-cb {
      animation: none;
    }
  }
</style>
