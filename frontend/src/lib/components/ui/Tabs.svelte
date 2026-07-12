<script lang="ts">
  /**
   * Enterprise Tab Navigation Component
   * Follows design system: Apple-level polish × Bloomberg-level data density
   *
   * Features:
   * - Underline and pill variants
   * - Optional count badges
   * - Disabled state
   * - Full keyboard navigation (Arrow keys, Home, End)
   * - ARIA accessibility
   */

  import { createEventDispatcher, onMount } from 'svelte';
  import type { Tab } from '$lib/types/components';

  interface Props {
    tabs?: Tab[];
    activeTab?: string;
    variant?: 'underline' | 'pill';
  }

  let { tabs = [], activeTab = $bindable(''), variant = 'underline' }: Props = $props();

  const dispatch = createEventDispatcher<{ change: string }>();

  let tabButtons: HTMLButtonElement[] = $state([]);

  function handleTabClick(tabId: string, disabled: boolean) {
    if (disabled) return;
    activeTab = tabId;
    dispatch('change', tabId);
  }

  function handleKeyDown(event: KeyboardEvent, index: number) {
    let newIndex = index;

    switch (event.key) {
      case 'ArrowLeft':
        event.preventDefault();
        newIndex = index > 0 ? index - 1 : tabs.length - 1;
        break;
      case 'ArrowRight':
        event.preventDefault();
        newIndex = index < tabs.length - 1 ? index + 1 : 0;
        break;
      case 'Home':
        event.preventDefault();
        newIndex = 0;
        break;
      case 'End':
        event.preventDefault();
        newIndex = tabs.length - 1;
        break;
      default:
        return;
    }

    // Skip disabled tabs
    while (tabs[newIndex]?.disabled && newIndex !== index) {
      if (event.key === 'ArrowLeft' || event.key === 'Home') {
        newIndex = newIndex > 0 ? newIndex - 1 : tabs.length - 1;
      } else {
        newIndex = newIndex < tabs.length - 1 ? newIndex + 1 : 0;
      }
    }

    if (tabButtons[newIndex] && !tabs[newIndex].disabled) {
      tabButtons[newIndex].focus();
      handleTabClick(tabs[newIndex].id, false);
    }
  }

  onMount(() => {
    // Set initial active tab if not provided
    if (!activeTab && tabs.length > 0) {
      const firstEnabledTab = tabs.find(t => !t.disabled);
      if (firstEnabledTab) {
        activeTab = firstEnabledTab.id;
      }
    }
  });
</script>

<div class="tabs" role="tablist" aria-label="Content sections">
  {#each tabs as tab, index (tab.id)}
    <button
      bind:this={tabButtons[index]}
      class="tab"
      class:pill={variant === 'pill'}
      class:active={activeTab === tab.id}
      role="tab"
      aria-selected={activeTab === tab.id}
      aria-controls="{tab.id}-panel"
      aria-disabled={tab.disabled}
      tabindex={activeTab === tab.id ? 0 : -1}
      disabled={tab.disabled}
      onclick={() => handleTabClick(tab.id, !!tab.disabled)}
      onkeydown={(e) => handleKeyDown(e, index)}
    >
      {tab.label}
      {#if tab.count !== undefined}
        <span class="tab-count" aria-label="{tab.count} items">
          {tab.count}
        </span>
      {/if}
    </button>
  {/each}
</div>

<style>
  /* Tabs are styled in design-tokens.css, but add component-specific styles here */

  .tab-count {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 20px;
    height: 20px;
    padding: 0 6px;
    margin-left: 6px;
    font-size: 11px;
    font-weight: 600;
    background: var(--surface-elevated);
    color: var(--text-secondary);
    border-radius: 10px;
    transition: all var(--transition-fast);
  }

  .tab.active .tab-count {
    background: var(--indigo-contrast-surface);
    color: var(--brand-indigo);
  }

  .tab.pill.active .tab-count {
    background: rgba(255, 255, 255, 0.15);
    color: var(--brand-indigo);
  }

  .tab:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .tab:disabled:hover {
    color: var(--text-secondary);
  }

  /* Keyboard focus indicator */
  .tab:focus-visible {
    outline: 2px solid var(--brand-indigo);
    outline-offset: -2px;
  }
</style>
