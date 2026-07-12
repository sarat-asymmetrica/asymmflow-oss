<!-- @migration-task Error while migrating Svelte code: This migration would change the name of a slot (trigger to trigger_1) making the component unusable -->
<script lang="ts">
  /**
   * Enterprise Dropdown Component
   * Click or hover-triggered dropdown with proper layering and accessibility
   *
   * Features:
   * - Click or hover trigger modes
   * - Left/right alignment
   * - Proper z-index management
   * - Full keyboard navigation (Escape, Arrow keys)
   * - Click-outside to close
   * - ARIA accessibility
   */

  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import { clickOutside } from '$lib/actions/clickOutside';
  import type { DropdownOption } from '$lib/types/components';

  export let options: DropdownOption[] = [];
  export let trigger: 'click' | 'hover' = 'click';
  export let align: 'left' | 'right' = 'left';
  export let disabled: boolean = false;

  const dispatch = createEventDispatcher<{ select: string }>();

  let isOpen = false;
  let dropdownElement: HTMLDivElement;
  let selectedIndex = -1;

  function toggle() {
    if (disabled) return;
    isOpen = !isOpen;
    if (isOpen) {
      selectedIndex = -1;
    }
  }

  function open() {
    if (disabled || trigger !== 'hover') return;
    isOpen = true;
  }

  function close() {
    if (trigger !== 'hover') return;
    isOpen = false;
  }

  function handleClose() {
    isOpen = false;
    selectedIndex = -1;
  }

  function selectOption(option: DropdownOption) {
    if (option.disabled) return;
    dispatch('select', option.value);
    handleClose();
  }

  function handleKeyDown(event: KeyboardEvent) {
    if (!isOpen) return;

    switch (event.key) {
      case 'Escape':
        event.preventDefault();
        handleClose();
        break;
      case 'ArrowDown':
        event.preventDefault();
        selectedIndex = Math.min(selectedIndex + 1, options.length - 1);
        // Skip disabled options
        while (options[selectedIndex]?.disabled && selectedIndex < options.length - 1) {
          selectedIndex++;
        }
        break;
      case 'ArrowUp':
        event.preventDefault();
        selectedIndex = Math.max(selectedIndex - 1, 0);
        // Skip disabled options
        while (options[selectedIndex]?.disabled && selectedIndex > 0) {
          selectedIndex--;
        }
        break;
      case 'Enter':
        event.preventDefault();
        if (selectedIndex >= 0 && selectedIndex < options.length) {
          selectOption(options[selectedIndex]);
        }
        break;
    }
  }

  onMount(() => {
    if (trigger === 'click') {
      document.addEventListener('keydown', handleKeyDown);
    }
  });

  onDestroy(() => {
    if (trigger === 'click') {
      document.removeEventListener('keydown', handleKeyDown);
    }
  });
</script>

<div
  class="dropdown"
  class:disabled
  role="presentation"
  on:mouseenter={open}
  on:mouseleave={close}
  use:clickOutside={handleClose}
>
  <div
    class="dropdown-trigger"
    role="button"
    tabindex={disabled ? -1 : 0}
    aria-haspopup="true"
    aria-expanded={isOpen}
    on:click={toggle}
    on:keydown={(e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        toggle();
      }
    }}
  >
    <slot name="trigger">
      <button class="btn btn-secondary" {disabled}>
        Menu
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
          <path
            d="M3 4.5L6 7.5L9 4.5"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
    </slot>
  </div>

  {#if isOpen}
    <div
      bind:this={dropdownElement}
      class="dropdown-menu"
      class:align-left={align === 'left'}
      class:align-right={align === 'right'}
      role="menu"
      aria-orientation="vertical"
    >
      {#each options as option, index (option.value)}
        <button
          class="dropdown-item"
          class:selected={index === selectedIndex}
          class:disabled={option.disabled}
          role="menuitem"
          tabindex={option.disabled ? -1 : 0}
          on:click={() => selectOption(option)}
        >
          {#if option.icon}
            <span class="dropdown-icon">{option.icon}</span>
          {/if}
          <span class="dropdown-label">{option.label}</span>
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .dropdown {
    position: relative;
    display: inline-block;
  }

  .dropdown.disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .dropdown-trigger {
    cursor: pointer;
  }

  .dropdown-trigger:focus-visible {
    outline: 2px solid var(--brand-indigo);
    outline-offset: 2px;
    border-radius: var(--border-radius-sm);
  }

  .dropdown-menu {
    position: absolute;
    top: calc(100% + 4px);
    min-width: 180px;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--border-radius);
    box-shadow: var(--shadow-md);
    padding: 4px;
    z-index: var(--z-dropdown);
    animation: slideDown var(--transition-fast) var(--easing-smooth);
  }

  .dropdown-menu.align-left {
    left: 0;
  }

  .dropdown-menu.align-right {
    right: 0;
  }

  .dropdown-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 8px 12px;
    font-size: 14px;
    color: var(--text-primary);
    background: transparent;
    border: none;
    border-radius: var(--border-radius-sm);
    cursor: pointer;
    transition: background var(--transition-fast);
    text-align: left;
    font-family: var(--font-family);
  }

  .dropdown-item:hover:not(.disabled) {
    background: var(--brand-indigo-tint);
  }

  .dropdown-item.selected:not(.disabled) {
    background: var(--brand-indigo-tint-medium);
  }

  .dropdown-item.disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .dropdown-item:focus-visible {
    outline: 2px solid var(--brand-indigo);
    outline-offset: -2px;
  }

  .dropdown-icon {
    font-size: 16px;
    line-height: 1;
  }

  .dropdown-label {
    flex: 1;
  }

  @keyframes slideDown {
    from {
      opacity: 0;
      transform: translateY(-8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
</style>
