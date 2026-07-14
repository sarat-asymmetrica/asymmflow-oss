<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { getDefaultDivisionKey } from '$lib/divisions.svelte';

  interface Props {
    activeRoute?: string;
  }

  let { activeRoute = 'dashboard' }: Props = $props();
  let collapsed = $state(false);

  const dispatch = createEventDispatcher();

  const routes = [
    { id: 'dashboard', label: 'Dashboard', icon: 'dashboard' },
    { id: 'opportunities', label: 'Opportunities', icon: 'lightbulb' },
    { id: 'orders', label: 'Offers & Orders', icon: 'shopping_cart' },
    { id: 'customers', label: 'Customers', icon: 'group' },
    { id: 'suppliers', label: 'Suppliers', icon: 'inventory' },
    { separator: true },
    { id: 'reports', label: 'Reports', icon: 'bar_chart' },
    { id: 'butler', label: 'Butler', icon: 'folder_open' },
    { separator: true },
    { id: 'settings', label: 'Settings', icon: 'settings' }
  ];

  function navigate(id) {
    dispatch('navigate', id);
  }

  function toggleCollapse() {
    collapsed = !collapsed;
    dispatch('collapse', collapsed);
  }

  // Inline SVGs for icons (keeping it sovereign)
  const icons = {
    dashboard: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"></rect><rect x="14" y="3" width="7" height="7"></rect><rect x="14" y="14" width="7" height="7"></rect><rect x="3" y="14" width="7" height="7"></rect></svg>`,
    lightbulb: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="17" x2="12" y2="17"></line><path d="M15 14c.2-1 .7-1.7 1.5-2.5 1-1 1.5-2.2 1.5-3.5A6 6 0 0 0 6 8c0 1 .2 2.2 1.5 3.5.7.7 1.3 1.5 1.5 2.5"></path><path d="M9 18h6"></path><path d="M10 22h4"></path></svg>`,
    shopping_cart: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="21" r="1"></circle><circle cx="20" cy="21" r="1"></circle><path d="M1 1h4l2.68 13.39a2 2 0 0 0 2 1.61h9.72a2 2 0 0 0 2-1.61L23 6H6"></path></svg>`,
    group: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg>`,
    inventory: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line></svg>`,
    bar_chart: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="20" x2="12" y2="10"></line><line x1="18" y1="20" x2="18" y2="4"></line><line x1="6" y1="20" x2="6" y2="16"></line></svg>`,
    folder_open: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg>`,
    settings: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>`,
    chevron_left: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"></polyline></svg>`,
    chevron_right: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>`
  };
</script>

<nav class="zen-nav h-full flex flex-col border-r border-gray-200 bg-[#fdfbf7] transition-all duration-300 ease-wabi-sabi {collapsed ? 'w-20' : 'w-64'}" style="padding-top: var(--space-md); padding-bottom: var(--space-md)">

  <!-- Brand Header -->
  <div class="flex items-center relative h-10" style="padding-left: var(--space-md); padding-right: var(--space-md); margin-bottom: var(--space-lg); gap: var(--space-sm)">
    <div class="w-8 h-8 bg-black rounded-full flex-shrink-0 flex items-center justify-center text-white font-bold font-serif shadow-lg">P</div>
    <h1 class="font-serif text-lg tracking-widest whitespace-nowrap transition-opacity duration-200 {collapsed ? 'opacity-0 w-0' : 'opacity-100'}">{getDefaultDivisionKey()}</h1>
    
    <!-- Collapse Toggle -->
    <button class="absolute -right-3 top-1 w-6 h-6 bg-white border border-gray-200 rounded-full flex items-center justify-center text-gray-400 hover:text-black shadow-sm z-10" onclick={toggleCollapse}>
      {@html collapsed ? icons.chevron_right : icons.chevron_left}
    </button>
  </div>

  <div class="flex-grow overflow-y-auto scrollbar-hide" style="padding-left: var(--space-sm); padding-right: var(--space-sm); display: flex; flex-direction: column; gap: 1px">
    {#if !collapsed}
      <div class="text-[10px] font-mono text-gray-400 uppercase tracking-widest opacity-100 transition-opacity duration-300" style="margin-bottom: var(--space-xs); margin-top: var(--space-xs); padding-left: var(--space-sm)">Operations</div>
    {/if}
    {#each routes.filter(r => ['dashboard', 'opportunities', 'orders'].includes(r.id)) as route}
        <button
          class="w-full flex items-center rounded-lg font-serif text-sm transition-all duration-200 group relative
                 {activeRoute === route.id ? 'bg-black text-white shadow-md' : 'text-gray-500 hover:bg-black/5 hover:text-gray-900'}"
          style="gap: var(--space-sm); padding-left: var(--space-sm); padding-right: var(--space-sm); padding-top: var(--space-sm); padding-bottom: var(--space-sm)"
          onclick={() => navigate(route.id)}
          title={collapsed ? route.label : ''}
        >
          <span class="flex-shrink-0 transition-transform duration-200 {activeRoute === route.id ? 'scale-110' : 'group-hover:scale-110'}">{@html icons[route.icon] || ''}</span>
          <span class="whitespace-nowrap overflow-hidden transition-all duration-200 {collapsed ? 'w-0 opacity-0' : 'w-auto opacity-100'}">{route.label}</span>
        </button>
    {/each}

    {#if !collapsed}
      <div class="text-[10px] font-mono text-gray-400 uppercase tracking-widest opacity-100 transition-opacity duration-300" style="margin-bottom: var(--space-xs); margin-top: var(--space-md); padding-left: var(--space-sm)">Relationships</div>
    {:else}
      <div style="height: var(--space-sm)"></div> <!-- Spacer when collapsed -->
    {/if}
    {#each routes.filter(r => ['customers', 'suppliers'].includes(r.id)) as route}
        <button
          class="w-full flex items-center rounded-lg font-serif text-sm transition-all duration-200 group relative
                 {activeRoute === route.id ? 'bg-black text-white shadow-md' : 'text-gray-500 hover:bg-black/5 hover:text-gray-900'}"
          style="gap: var(--space-sm); padding-left: var(--space-sm); padding-right: var(--space-sm); padding-top: var(--space-sm); padding-bottom: var(--space-sm)"
          onclick={() => navigate(route.id)}
          title={collapsed ? route.label : ''}
        >
          <span class="flex-shrink-0 transition-transform duration-200 {activeRoute === route.id ? 'scale-110' : 'group-hover:scale-110'}">{@html icons[route.icon] || ''}</span>
          <span class="whitespace-nowrap overflow-hidden transition-all duration-200 {collapsed ? 'w-0 opacity-0' : 'w-auto opacity-100'}">{route.label}</span>
        </button>
    {/each}

    {#if !collapsed}
      <div class="text-[10px] font-mono text-gray-400 uppercase tracking-widest opacity-100 transition-opacity duration-300" style="margin-bottom: var(--space-xs); margin-top: var(--space-md); padding-left: var(--space-sm)">Intelligence</div>
    {:else}
      <div style="height: var(--space-sm)"></div>
    {/if}
    {#each routes.filter(r => ['reports', 'butler'].includes(r.id)) as route}
        <button
          class="w-full flex items-center rounded-lg font-serif text-sm transition-all duration-200 group relative
                 {activeRoute === route.id ? 'bg-black text-white shadow-md' : 'text-gray-500 hover:bg-black/5 hover:text-gray-900'}"
          style="gap: var(--space-sm); padding-left: var(--space-sm); padding-right: var(--space-sm); padding-top: var(--space-sm); padding-bottom: var(--space-sm)"
          onclick={() => navigate(route.id)}
          title={collapsed ? route.label : ''}
        >
          <span class="flex-shrink-0 transition-transform duration-200 {activeRoute === route.id ? 'scale-110' : 'group-hover:scale-110'}">{@html icons[route.icon] || ''}</span>
          <span class="whitespace-nowrap overflow-hidden transition-all duration-200 {collapsed ? 'w-0 opacity-0' : 'w-auto opacity-100'}">{route.label}</span>
        </button>
    {/each}

    <div style="margin-top: auto; padding-top: var(--space-sm)">
        {#each routes.filter(r => ['settings'].includes(r.id)) as route}
            <button
            class="w-full flex items-center rounded-lg font-serif text-sm transition-all duration-200 group relative
                    {activeRoute === route.id ? 'bg-black text-white shadow-md' : 'text-gray-500 hover:bg-black/5 hover:text-gray-900'}"
            style="gap: var(--space-sm); padding-left: var(--space-sm); padding-right: var(--space-sm); padding-top: var(--space-sm); padding-bottom: var(--space-sm)"
            onclick={() => navigate(route.id)}
            title={collapsed ? route.label : ''}
            >
            <span class="flex-shrink-0 transition-transform duration-200 {activeRoute === route.id ? 'scale-110' : 'group-hover:scale-110'}">{@html icons[route.icon] || ''}</span>
            <span class="whitespace-nowrap overflow-hidden transition-all duration-200 {collapsed ? 'w-0 opacity-0' : 'w-auto opacity-100'}">{route.label}</span>
            </button>
        {/each}
    </div>
  </div>


  <div class="border-t border-gray-200 overflow-hidden" style="padding-left: var(--space-md); padding-right: var(--space-md); padding-top: var(--space-sm)">
    <div class="text-[10px] font-mono text-gray-400 uppercase tracking-widest whitespace-nowrap {collapsed ? 'opacity-0' : 'opacity-100'} transition-opacity" style="margin-bottom: 1px">System</div>
    <div class="flex justify-between items-center text-xs text-gray-500 whitespace-nowrap">
      <span class="{collapsed ? 'opacity-0' : 'opacity-100'} transition-opacity">v2.0 Moonshot</span>
      <span class="w-2 h-2 rounded-full bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.6)] animate-pulse"></span>
    </div>
  </div>
</nav>
