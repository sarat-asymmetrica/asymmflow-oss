<script lang="ts">
  /* The real app shell (K5). Replaces the K0–K4 dev harness: a license gate
   * (the ONE live auth path — owner ruling: license-only), session + divisions
   * store init at boot, a permission-filtered sidebar built from the screen
   * registry, and navigation-store routing. Under mock (the lab) the license
   * validates to a synthetic admin so the app boots straight in; under the real
   * Wails runtime it validates the device license or shows the activation
   * screen. Screen rendering dispatches by archetype, exactly as before. */
  import { usingWails } from './bridge'
  import { validateLicense, type AuthResult } from './bridge/auth'
  import { initDivisions } from './stores/divisions.svelte'
  import { setSession, hasPermission, getCurrentUser } from './stores/session.svelte'
  import { currentRoute, navigate, setInitialRoute } from './stores/navigation.svelte'
  import LicenseActivation from './app/LicenseActivation.svelte'
  import DocumentLedger from '$kernel/archetypes/DocumentLedger.svelte'
  import EntityMaster from '$kernel/archetypes/EntityMaster.svelte'
  import Hub from '$kernel/archetypes/Hub.svelte'
  import type { NavIntent } from '$kernel/hub'
  import { screens, screensByGroup, type ScreenEntry } from './screens/registry'

  type AuthState = 'checking' | 'license_needed' | 'approved'
  let authState = $state<AuthState>('checking')
  let navOpen = $state(false)

  const groups = $derived(
    screensByGroup()
      .map((g) => ({ group: g.group, items: g.items.filter((s) => hasPermission(s.permission ?? '')) }))
      .filter((g) => g.items.length > 0),
  )
  const visibleKeys = $derived(new Set(groups.flatMap((g) => g.items.map((s) => s.key))))
  const active = $derived(
    screens.find((s) => s.key === currentRoute().key) ?? groups[0]?.items[0] ?? screens[0],
  )
  const route = $derived(currentRoute())

  function applyAuth(result: AuthResult) {
    setSession(
      { id: result.deviceHash || 'user', fullName: result.displayName || 'User', roleName: result.role },
      result.permissions,
    )
    authState = 'approved'
    // Land on the first permitted screen.
    const first = screensByGroup()
      .flatMap((g) => g.items)
      .find((s) => hasPermission(s.permission ?? ''))
    if (first) setInitialRoute(first.key)
  }

  async function boot() {
    await initDivisions()
    try {
      const result = await validateLicense()
      if (result.ok) applyAuth(result)
      else authState = 'license_needed'
    } catch {
      authState = 'license_needed'
    }
  }
  void boot()

  function pick(key: string) {
    navigate(key)
    navOpen = false
  }
  function onNavIntent(intent: NavIntent) {
    if (!visibleKeys.has(intent.key)) return
    navigate(intent.key, intent.query ? { query: intent.query } : undefined)
  }

  const isLedger = (s: ScreenEntry) => s.archetype === 'ledger'
  const isEntity = (s: ScreenEntry) => s.archetype === 'entity'
  const isHub = (s: ScreenEntry) => s.archetype === 'hub'
  const user = $derived(getCurrentUser())
</script>

{#if authState === 'checking'}
  <div class="k-app-splash">
    <span class="k-app-splash-brand">AsymmFlow</span>
    <span class="k-app-splash-note">Starting…</span>
  </div>
{:else if authState === 'license_needed'}
  <LicenseActivation onActivated={applyAuth} />
{:else}
  <div class="k-app" class:nav-open={navOpen}>
    <button class="k-app-navtoggle" aria-label="Toggle navigation" onclick={() => (navOpen = !navOpen)}>☰</button>
    <aside class="k-app-side" class:open={navOpen}>
      <div class="k-app-brand">AsymmFlow</div>
      <nav class="k-app-nav">
        {#each groups as g (g.group)}
          <div class="k-app-group">
            <span class="k-app-group-label">{g.group}</span>
            {#each g.items as s (s.key)}
              <button class="lab-tab" class:active={active?.key === s.key} onclick={() => pick(s.key)}>
                {s.label}
              </button>
            {/each}
          </div>
        {/each}
      </nav>
      {#if user}
        <span class="k-app-user">{user.fullName} · {user.roleName || 'User'}</span>
      {/if}
      <span class="k-app-bridge" class:real={usingWails()}>
        {usingWails() ? 'REAL (Wails)' : 'mock'}
      </span>
    </aside>

    <main class="lab-main k-app-main">
      {#if active}
        {#if isLedger(active)}
          {#key active.key}
            <DocumentLedger descriptor={active.descriptor} initialQuery={route.query} />
          {/key}
        {:else if isEntity(active)}
          {#key active.key}
            <EntityMaster descriptor={active.descriptor} initialQuery={route.query} />
          {/key}
        {:else if isHub(active)}
          {#key active.key}
            <Hub descriptor={active.descriptor} navigate={onNavIntent} />
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
{/if}

<style>
  .k-app-splash {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--k-space-xs);
    height: 100%;
  }
  .k-app-splash-brand {
    font-family: var(--font-display);
    font-weight: 700;
    font-size: calc(18px * var(--ui-font-scale));
  }
  .k-app-splash-note {
    font-size: var(--meta-size);
    color: var(--text-secondary);
  }
  .k-app {
    display: flex;
    height: 100%;
    min-height: 0;
    min-width: 0;
    position: relative;
  }
  .k-app-navtoggle {
    display: none;
    position: fixed;
    top: 8px;
    left: 8px;
    z-index: 30;
    width: 34px;
    height: 34px;
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    background: var(--surface);
    color: var(--text-primary);
    font-size: 16px;
    cursor: pointer;
  }
  .k-app-side {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-md);
    width: 210px;
    flex-shrink: 0;
    padding: var(--k-space-md);
    border-right: var(--border-width) solid var(--border);
    background: var(--surface);
    overflow-y: auto;
  }
  .k-app-brand {
    font-family: var(--font-display);
    font-weight: 700;
    font-size: calc(15px * var(--ui-font-scale));
    flex-shrink: 0;
  }
  .k-app-nav {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-md);
    flex: 1;
    min-height: 0;
  }
  .k-app-group {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .k-app-group-label {
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
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .lab-tab.active {
    background: var(--onyx-tint);
    color: var(--text-primary);
    font-weight: 600;
  }
  .k-app-user {
    flex-shrink: 0;
    font-size: calc(11px * var(--ui-font-scale));
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .k-app-main {
    flex: 1;
    min-height: 0;
    min-width: 0;
  }
  .k-app-bridge {
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
  .k-app-bridge.real {
    background: rgba(30, 130, 76, 0.12);
    color: #1e824c;
  }

  @media (max-width: 720px) {
    .k-app-navtoggle {
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .k-app-side {
      position: fixed;
      top: 0;
      left: 0;
      bottom: 0;
      z-index: 20;
      transform: translateX(-100%);
      transition: transform var(--motion-medium, 200ms) var(--ease-standard, ease);
    }
    .k-app-side.open {
      transform: none;
      box-shadow: 0 0 24px rgba(0, 0, 0, 0.18);
    }
    .k-app-main {
      width: 100%;
    }
    .k-app-main :global(.k-page-header) {
      padding-left: 40px;
    }
  }
</style>
