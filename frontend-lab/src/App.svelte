<script lang="ts">
  import { usingWails } from './bridge'
  import DocumentLedger from '$kernel/archetypes/DocumentLedger.svelte'
  import EntityMaster from '$kernel/archetypes/EntityMaster.svelte'
  import Hub from '$kernel/archetypes/Hub.svelte'
  import type { NavIntent } from '$kernel/hub'
  import type { LedgerQuery } from '$kernel/ledger-core'
  import { screens, screensByGroup, type ScreenEntry } from './screens/registry'

  let activeKey = $state(screens[0]?.key ?? '')
  // Drill-down seed for the screen we navigated TO (parity #4). Cleared on a
  // manual tab click so a hand-picked screen never inherits a stale filter.
  let pendingQuery = $state<Partial<LedgerQuery> | undefined>(undefined)
  const active = $derived(screens.find((s) => s.key === activeKey) ?? screens[0])
  const groups = screensByGroup()

  function pick(key: string) {
    pendingQuery = undefined
    activeKey = key
  }
  function navigate(intent: NavIntent) {
    if (!screens.some((s) => s.key === intent.key)) return
    pendingQuery = intent.query
    activeKey = intent.key
  }

  function isLedger(s: ScreenEntry) {
    return s.archetype === 'ledger'
  }
  function isEntity(s: ScreenEntry) {
    return s.archetype === 'entity'
  }
  function isHub(s: ScreenEntry) {
    return s.archetype === 'hub'
  }
</script>

<div class="lab-shell">
  <aside class="lab-side">
    <div class="lab-brand">Kernel Lab</div>
    <nav class="lab-nav">
      {#each groups as g (g.group)}
        <div class="lab-group">
          <span class="lab-group-label">{g.group}</span>
          {#each g.items as s (s.key)}
            <button class="lab-tab" class:active={activeKey === s.key} onclick={() => pick(s.key)}>
              {s.label}
            </button>
          {/each}
        </div>
      {/each}
    </nav>
    <span class="lab-bridge" class:real={usingWails()}>
      bridge: {usingWails() ? 'REAL (Wails)' : 'mock'}
    </span>
  </aside>

  <main class="lab-main">
    {#if active}
      {#if isLedger(active)}
        {#key active.key}
          <DocumentLedger descriptor={active.descriptor} initialQuery={pendingQuery} />
        {/key}
      {:else if isEntity(active)}
        {#key active.key}
          <EntityMaster descriptor={active.descriptor} initialQuery={pendingQuery} />
        {/key}
      {:else if isHub(active)}
        {#key active.key}
          <Hub descriptor={active.descriptor} {navigate} />
        {/key}
      {:else if active.component}
        {@const Bespoke = active.component}
        {#key active.key}
          <Bespoke />
        {/key}
      {/if}
    {/if}
  </main>
</div>

<style>
  /* Lab shell chrome — dev harness territory, not a product screen.
   * The real app shell (sidebar nav, routing, auth) is built at K5. */
  .lab-shell {
    display: flex;
    height: 100%;
    min-height: 0;
    min-width: 0;
  }
  .lab-side {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-md);
    width: 200px;
    flex-shrink: 0;
    padding: var(--k-space-md);
    border-right: var(--border-width) solid var(--border);
    background: var(--surface);
    overflow-y: auto;
  }
  .lab-brand {
    font-family: var(--font-display);
    font-weight: 700;
    font-size: calc(14px * var(--ui-font-scale));
    flex-shrink: 0;
  }
  .lab-nav {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-md);
    flex: 1;
    min-height: 0;
  }
  .lab-group {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .lab-group-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-muted);
    padding: 0 8px;
    margin-bottom: 2px;
  }
  .lab-tab {
    font: inherit;
    font-size: calc(13px * var(--ui-font-scale));
    text-align: left;
    padding: 5px 8px;
    border: none;
    border-radius: var(--border-radius-sm);
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
  }
  .lab-tab.active {
    background: var(--onyx-tint);
    color: var(--text-primary);
    font-weight: 600;
  }
  .lab-main {
    flex: 1;
    min-height: 0;
    min-width: 0;
  }
  .lab-bridge {
    flex-shrink: 0;
    font-size: calc(11px * var(--ui-font-scale));
    font-weight: 600;
    color: var(--text-muted);
    padding: 2px 10px;
    border-radius: var(--border-radius-pill);
    background: var(--onyx-tint);
    white-space: nowrap;
    text-align: center;
  }
  .lab-bridge.real {
    background: rgba(30, 130, 76, 0.12);
    color: #1e824c;
  }
</style>
