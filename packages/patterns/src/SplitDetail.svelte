<script lang="ts" generics="T">
  /**
   * SplitDetail — master-detail composition.
   *
   * List pane (caller-provided via children Snippet) + Drawer-based detail panel
   * on row select. Selection state is URL-free ($bindable selectedId).
   *
   * Constitution §2.6: focus containment lives inside Drawer (already has it).
   * Constitution §2.7: interruptible motion from Drawer's slide-in/out.
   * Constitution §4d: elevation on the Drawer panel (--af-shadow-overlay).
   */

  import type { Snippet } from 'svelte';
  import { Drawer } from '@asymmflow/ui';

  // ─── Props ───────────────────────────────────────────────────────────────────

  export interface SplitDetailProps<T> {
    /** Bindable selected item identifier. null = no selection / drawer closed. */
    selectedId?: string | number | null;
    /** Drawer title. If not provided falls back to "Detail". */
    drawerTitle?: string;
    /** Drawer size tier. Default md (480px). */
    drawerSize?: 'sm' | 'md' | 'lg';
    /** The master list content (DataShell or any list UI). */
    children?: Snippet;
    /**
     * Snippet for the detail panel body.
     * Receives selectedId so the caller can look up their data.
     */
    detail?: Snippet<[string | number]>;
    /**
     * Snippet for the detail panel footer (action buttons).
     * Receives selectedId.
     */
    detailFooter?: Snippet<[string | number]>;
  }

  let {
    selectedId = $bindable(null),
    drawerTitle = 'Detail',
    drawerSize = 'md',
    children,
    detail,
    detailFooter,
  }: SplitDetailProps<T> = $props();

  // Bindable local: the Drawer's open state.
  // Derived from selectedId; Drawer's internal close (Esc/scrim) is caught
  // via a one-shot $effect that only fires AFTER the initial mount.
  let drawerOpen = $state(selectedId != null);

  // Keep drawerOpen in sync whenever selectedId changes from the outside.
  $effect(() => {
    drawerOpen = selectedId != null;
  });

  // Detect when Drawer closes itself (Esc / scrim-click) and clear selection.
  // We guard with a mounted flag so the initial render doesn't clear a non-null selectedId.
  let mounted = false;
  $effect(() => {
    // Subscribe to drawerOpen changes after mount.
    void drawerOpen;
    if (!mounted) { mounted = true; return; }
    if (!drawerOpen) {
      selectedId = null;
    }
  });
</script>

<div class="af-split">
  <!-- Master list pane -->
  <div class="af-split__list" aria-label="List">
    {@render children?.()}
  </div>

  <!-- Detail drawer: portals to body, R1 slide-in, R3 slide-out -->
  <Drawer
    bind:open={drawerOpen}
    title={drawerTitle}
    size={drawerSize}
    side="right"
  >
    {#snippet children()}
      {#if selectedId != null && detail}
        {@render detail(selectedId)}
      {/if}
    {/snippet}

    {#snippet footer()}
      {#if selectedId != null && detailFooter}
        {@render detailFooter(selectedId)}
      {/if}
    {/snippet}
  </Drawer>
</div>

<style>
  .af-split {
    display: contents;
  }

  .af-split__list {
    /* The list takes up whatever space its parent gives it.
       display: contents passes layout through; this wrapper just provides
       an accessible aria landmark hook. */
    width: 100%;
  }
</style>
