<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  interface Props {
    collapsed?: boolean;
    currentScreen?: string;
  }

  let { collapsed = $bindable(false), currentScreen = 'dashboard' }: Props = $props();

  const dispatch = createEventDispatcher();

  interface NavItem {
    id: string;
    label: string;
    icon: string;
    children?: string[];
  }

  const navItems: NavItem[] = [
    { id: 'dashboard', label: 'Dashboard', icon: 'home' },
    {
      id: 'sales',
      label: 'Sales Hub',
      icon: 'briefcase',
      children: ['rfqs', 'offers', 'orders']
    },
    {
      id: 'operations',
      label: 'Operations',
      icon: 'truck',
      children: ['purchase-orders', 'goods-receipt', 'supplier-invoices', 'delivery-notes']
    },
    {
      id: 'finance',
      label: 'Finance',
      icon: 'dollar',
      children: ['invoices', 'payments', 'accounting']
    },
    { id: 'customers', label: 'Customers', icon: 'users' },
    { id: 'products', label: 'Products', icon: 'box' },
    { id: 'settings', label: 'Settings', icon: 'settings' },
  ];

  let expandedItems: Record<string, boolean> = $state({});

  function navigate(itemId: string) {
    dispatch('navigate', { screen: itemId });
  }

  function toggleExpanded(itemId: string) {
    expandedItems[itemId] = !expandedItems[itemId];
  }

  // Icon SVG paths (simple, recognizable shapes)
  const icons: Record<string, string> = {
    home: 'M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z',
    briefcase: 'M20 7h-4V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2H4a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2z',
    truck: 'M1 3h15v13H1V3zm17 6h4l2 3v4h-6V9z',
    dollar: 'M12 1v2m0 18v2m-7-11h14m-7-6C9.2 6 7 8.2 7 11s2.2 5 5 5 5-2.2 5-5-2.2-5-5-5z',
    users: 'M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2m8-10a4 4 0 1 0 0-8 4 4 0 0 0 0 8z',
    box: 'M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z',
    settings: 'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zm9.4-4.7l-1.9-3.3a1 1 0 0 0-1.4-.4l-1.6.9c-.5-.4-1-.7-1.6-.9V4.5a1 1 0 0 0-1-1h-3.8a1 1 0 0 0-1 1v2.1c-.6.2-1.1.5-1.6.9l-1.6-.9a1 1 0 0 0-1.4.4L2.6 10.3a1 1 0 0 0 .4 1.4l1.6.9c0 .6 0 1.2 0 1.8l-1.6.9a1 1 0 0 0-.4 1.4l1.9 3.3a1 1 0 0 0 1.4.4l1.6-.9c.5.4 1 .7 1.6.9v2.1a1 1 0 0 0 1 1h3.8a1 1 0 0 0 1-1v-2.1c.6-.2 1.1-.5 1.6-.9l1.6.9a1 1 0 0 0 1.4-.4l1.9-3.3a1 1 0 0 0-.4-1.4l-1.6-.9c0-.6 0-1.2 0-1.8l1.6-.9a1 1 0 0 0 .4-1.4z',
  };
</script>

<aside class="sidebar" class:collapsed>
  <nav class="nav">
    {#each navItems as item}
      <div class="nav-item-wrapper">
        <button
          class="nav-item"
          class:active={currentScreen === item.id}
          onclick={() => {
            if (item.children) {
              toggleExpanded(item.id);
            } else {
              navigate(item.id);
            }
          }}
          aria-current={currentScreen === item.id ? 'page' : undefined}
          title={collapsed ? item.label : undefined}
        >
          <svg
            class="nav-icon"
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d={icons[item.icon]} stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          {#if !collapsed}
            <span class="nav-label">{item.label}</span>
            {#if item.children}
              <svg
                class="nav-chevron"
                class:rotated={expandedItems[item.id]}
                width="16"
                height="16"
                viewBox="0 0 16 16"
                fill="none"
              >
                <path
                  d="M4 6L8 10L12 6"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                />
              </svg>
            {/if}
          {/if}
        </button>

        {#if item.children && expandedItems[item.id] && !collapsed}
          <div class="nav-children">
            {#each item.children as child}
              <button
                class="nav-child"
                class:active={currentScreen === child}
                onclick={() => navigate(child)}
              >
                <span class="nav-child-label">{child.replace(/-/g, ' ')}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {/each}
  </nav>

  <!-- Collapse Toggle -->
  <button
    class="collapse-toggle"
    onclick={() => collapsed = !collapsed}
    aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
    title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
  >
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
      {#if collapsed}
        <path d="M6 4L10 8L6 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
      {:else}
        <path d="M10 4L6 8L10 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
      {/if}
    </svg>
  </button>
</aside>

<style>
  .sidebar {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    width: 240px;
    background: var(--surface);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    transition: width var(--transition-base);
    z-index: var(--z-sticky);
  }

  .sidebar.collapsed {
    width: 60px;
  }

  .nav {
    flex: 1;
    padding: 16px 8px;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .nav-item-wrapper {
    margin-bottom: 4px;
  }

  .nav-item {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    border: none;
    background: none;
    color: var(--text-secondary);
    cursor: pointer;
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
    font-family: var(--font-family);
    font-size: 14px;
    font-weight: 500;
    text-align: left;
  }

  .sidebar.collapsed .nav-item {
    justify-content: center;
    padding: 10px;
  }

  .nav-item:hover {
    background: var(--brand-indigo-tint);
    color: var(--text-primary);
  }

  .nav-item.active {
    background: var(--brand-indigo);
    color: white;
  }

  .nav-item:focus-visible {
    outline: 2px solid var(--brand-indigo);
    outline-offset: 2px;
  }

  .nav-icon {
    flex-shrink: 0;
  }

  .nav-label {
    flex: 1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .nav-chevron {
    flex-shrink: 0;
    transition: transform var(--transition-fast);
  }

  .nav-chevron.rotated {
    transform: rotate(180deg);
  }

  .nav-children {
    margin-top: 4px;
    padding-left: 32px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .nav-child {
    width: 100%;
    display: flex;
    align-items: center;
    padding: 8px 12px;
    border: none;
    background: none;
    color: var(--text-muted);
    cursor: pointer;
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
    font-family: var(--font-family);
    font-size: 13px;
    text-align: left;
    text-transform: capitalize;
  }

  .nav-child:hover {
    background: var(--brand-indigo-tint);
    color: var(--text-primary);
  }

  .nav-child.active {
    background: var(--indigo-contrast-surface);
    color: var(--brand-indigo);
  }

  .nav-child-label {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .collapse-toggle {
    margin: 16px 8px;
    padding: 8px;
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text-secondary);
    cursor: pointer;
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .collapse-toggle:hover {
    background: var(--surface-elevated);
    border-color: var(--text-muted);
    color: var(--text-primary);
  }

  .collapse-toggle:focus-visible {
    outline: 2px solid var(--brand-indigo);
    outline-offset: 2px;
  }

  /* Scrollbar styling */
  .nav::-webkit-scrollbar {
    width: 4px;
  }

  .nav::-webkit-scrollbar-track {
    background: transparent;
  }

  .nav::-webkit-scrollbar-thumb {
    background: var(--border);
    border-radius: 4px;
  }

  .nav::-webkit-scrollbar-thumb:hover {
    background: var(--text-muted);
  }
</style>
