<script lang="ts">
  /* TabShell — a hub console that hosts several independent child surfaces as
   * tabs. Distinct from ViewSwitcher: ViewSwitcher toggles views of ONE
   * dataset the host renders itself; TabShell's tabs are separate, lazily
   * loaded surfaces — often a whole embedded screen (PeopleHub → Payroll,
   * WorkHub → Approvals) with its own fetch lifecycle. Tabs are
   * permission-gateable (a hidden tab renders nothing, never a disabled stub),
   * and each tab's content mounts only on first selection and then STAYS
   * mounted (display-toggled) so switching back doesn't refetch or lose state.
   * Owns only the tab chrome + a shared header slot (L1). Reuses ViewSwitcher
   * for the tab bar. Serves the K4 operational hubs AND the K5 tab-navigators
   * (Finance/Sales/CRM/Operations) — one primitive. */
  import type { Snippet } from 'svelte'
  import type { Tone } from '../tones'
  import ViewSwitcher from './ViewSwitcher.svelte'

  let {
    tabs,
    activeKey,
    onSelect,
    header,
    lazy = true,
    ariaLabel = 'Sections',
  }: {
    tabs: {
      key: string
      label: string
      badge?: string | number
      badgeTone?: Tone
      /** Permission gate — a tab with visible:false is omitted entirely. */
      visible?: boolean
      content: Snippet
    }[]
    activeKey: string
    onSelect: (key: string) => void
    /** Shared strip above the tab bar (composer, summary cards). */
    header?: Snippet
    /** Mount each tab's content only once first selected (default). */
    lazy?: boolean
    ariaLabel?: string
  } = $props()

  const visibleTabs = $derived(tabs.filter((t) => t.visible !== false))
  const switcherViews = $derived(
    visibleTabs.map((t) => ({
      key: t.key,
      label: t.label,
      ...(t.badge !== undefined ? { badge: t.badge } : {}),
      ...(t.badgeTone !== undefined ? { badgeTone: t.badgeTone } : {}),
    })),
  )

  // Keep-mounted set: any tab that has ever been active stays rendered (hidden)
  // so its child screen's state + fetched data survive tab switches.
  let seen = $state<string[]>([])
  $effect(() => {
    if (activeKey && !seen.includes(activeKey)) seen = [...seen, activeKey]
  })
</script>

<div class="k-tabshell">
  {#if header}
    <div class="k-tabshell-header">{@render header()}</div>
  {/if}

  <ViewSwitcher views={switcherViews} {activeKey} {onSelect} {ariaLabel} />

  <div class="k-tabshell-body">
    {#each visibleTabs as tab (tab.key)}
      {#if !lazy || seen.includes(tab.key)}
        <div class="k-tabshell-panel" style:display={activeKey === tab.key ? 'block' : 'none'}>
          {@render tab.content()}
        </div>
      {/if}
    {/each}
  </div>
</div>

<style>
  .k-tabshell {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-md);
    min-width: 0;
  }
  .k-tabshell-header,
  .k-tabshell-body,
  .k-tabshell-panel {
    min-width: 0;
  }
</style>
