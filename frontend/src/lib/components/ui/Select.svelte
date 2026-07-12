<script module lang="ts">
  export type SelectOption = { value: string; label: string; disabled?: boolean };
</script>

<script lang="ts">
  import { createBubbler, stopPropagation } from 'svelte/legacy';

  const bubble = createBubbler();
  /**
   * Enterprise Select Component
   * Philosophy: Clean dropdown with optional search for long lists
   * Follows: Apple-level polish × Bloomberg-level data density
   */
  import { createEventDispatcher, onMount } from 'svelte';

  
  interface Props {
    // SelectOption type is exported from the module context above
    options?: SelectOption[];
    value?: string;
    placeholder?: string;
    label?: string;
    error?: string;
    disabled?: boolean;
    required?: boolean;
    searchable?: boolean;
    id?: string;
    name?: string;
  }

  let {
    options = [],
    value = $bindable(''),
    placeholder = 'Select...',
    label = '',
    error = '',
    disabled = false,
    required = false,
    searchable = false,
    id = `select-${Math.random().toString(36).substr(2, 9)}`,
    name = ''
  }: Props = $props();

  const dispatch = createEventDispatcher();

  let isOpen = $state(false);
  let searchQuery = $state('');
  let highlightedIndex = $state(-1);
  let selectRef: HTMLDivElement = $state();

  let filteredOptions = $derived(searchable && searchQuery
    ? options.filter(opt =>
        opt.label.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : options);

  let selectedOption = $derived(options.find(opt => opt.value === value));

  function toggleDropdown() {
    if (disabled) return;
    isOpen = !isOpen;
    if (isOpen) {
      highlightedIndex = filteredOptions.findIndex(opt => opt.value === value);
    }
  }

  function selectOption(option: SelectOption) {
    if (option.disabled) return;
    value = option.value;
    isOpen = false;
    searchQuery = '';
    dispatch('change', { value, option });
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (!isOpen) {
      if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown') {
        e.preventDefault();
        isOpen = true;
      }
      return;
    }

    switch (e.key) {
      case 'Escape':
        e.preventDefault();
        isOpen = false;
        break;
      case 'ArrowDown':
        e.preventDefault();
        highlightedIndex = Math.min(highlightedIndex + 1, filteredOptions.length - 1);
        break;
      case 'ArrowUp':
        e.preventDefault();
        highlightedIndex = Math.max(highlightedIndex - 1, 0);
        break;
      case 'Enter':
        e.preventDefault();
        if (highlightedIndex >= 0 && filteredOptions[highlightedIndex]) {
          selectOption(filteredOptions[highlightedIndex]);
        }
        break;
    }
  }

  function handleClickOutside(e: MouseEvent) {
    if (selectRef && !selectRef.contains(e.target as Node)) {
      isOpen = false;
      searchQuery = '';
    }
  }

  onMount(() => {
    document.addEventListener('click', handleClickOutside);
    return () => document.removeEventListener('click', handleClickOutside);
  });
</script>

<div class="select-wrapper" class:has-error={!!error} class:disabled bind:this={selectRef}>
  {#if label}
    <label for={id} class="label">
      {label}
      {#if required}<span class="required" aria-label="required">*</span>{/if}
    </label>
  {/if}

  {#if !searchable}
    <!-- Simple native select for non-searchable -->
    <select
      {id}
      {name}
      {disabled}
      {required}
      bind:value
      class="select"
      onchange={(e) => {
        const option = options.find(opt => opt.value === value);
        dispatch('change', { value, option });
      }}
      aria-invalid={!!error}
      aria-describedby={error ? `${id}-error` : undefined}
    >
      {#if placeholder}
        <option value="" disabled selected={!value}>{placeholder}</option>
      {/if}
      {#each options as option}
        <option value={option.value} disabled={option.disabled}>
          {option.label}
        </option>
      {/each}
    </select>
  {:else}
    <!-- Custom searchable select -->
    <div
      class="select-custom"
      role="combobox"
      aria-expanded={isOpen}
      aria-haspopup="listbox"
      aria-controls="{id}-listbox"
      aria-invalid={!!error}
      tabindex={disabled ? -1 : 0}
      onclick={toggleDropdown}
      onkeydown={handleKeyDown}
    >
      <div class="select-value">
        {selectedOption?.label || placeholder}
      </div>
      <svg class="select-icon" class:open={isOpen} width="16" height="16" viewBox="0 0 16 16" fill="none">
        <path d="M4 6L8 10L12 6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    </div>

    {#if isOpen}
      <div class="dropdown" id="{id}-listbox" role="listbox">
        {#if searchable}
          <div class="search-wrapper">
            <input
              type="text"
              class="search-input"
              placeholder="Search..."
              bind:value={searchQuery}
              onclick={stopPropagation(bubble('click'))}
            />
          </div>
        {/if}
        <div class="options-list">
          {#each filteredOptions as option, idx (option.value)}
            <div
              class="option"
              class:selected={option.value === value}
              class:highlighted={idx === highlightedIndex}
              class:disabled={option.disabled}
              role="option"
              tabindex="-1"
              aria-selected={option.value === value}
              onclick={stopPropagation(() => selectOption(option))}
              onkeydown={stopPropagation((e: KeyboardEvent) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  selectOption(option);
                }
              })}
              onmouseenter={() => highlightedIndex = idx}
            >
              {option.label}
            </div>
          {:else}
            <div class="option disabled">No results found</div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}

  {#if error}
    <span id="{id}-error" class="error-message" role="alert">
      {error}
    </span>
  {/if}
</div>

<style>
  .select-wrapper {
    display: flex;
    flex-direction: column;
    gap: 6px;
    width: 100%;
    position: relative;
  }

  .label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    line-height: var(--line-height-tight);
  }

  .required {
    color: #DC2626;
    margin-left: 2px;
  }

  /* Native select */
  .select {
    width: 100%;
    padding: 8px 12px;
    font-size: 14px;
    font-family: var(--font-family);
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
    cursor: pointer;
  }

  .select:hover:not(:disabled) {
    border-color: var(--text-muted);
  }

  .select:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 3px var(--brand-indigo-tint);
  }

  .select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background: var(--surface-elevated);
  }

  /* Custom searchable select */
  .select-custom {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 8px 12px;
    font-size: 14px;
    font-family: var(--font-family);
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    transition: all var(--transition-fast);
    cursor: pointer;
  }

  .select-custom:hover {
    border-color: var(--text-muted);
  }

  .select-custom:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 3px var(--brand-indigo-tint);
  }

  .select-value {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .select-icon {
    flex-shrink: 0;
    color: var(--text-muted);
    transition: transform var(--transition-fast);
  }

  .select-icon.open {
    transform: rotate(180deg);
  }

  /* Dropdown */
  .dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    margin-top: 4px;
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    box-shadow: var(--shadow-md);
    z-index: var(--z-dropdown);
    max-height: 300px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .search-wrapper {
    padding: 8px;
    border-bottom: var(--border-width) solid var(--border);
  }

  .search-input {
    width: 100%;
    padding: 6px 10px;
    font-size: 13px;
    font-family: var(--font-family);
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: 6px;
  }

  .search-input:focus {
    outline: none;
    border-color: var(--brand-indigo);
  }

  .options-list {
    overflow-y: auto;
    max-height: 260px;
  }

  .option {
    padding: 8px 12px;
    font-size: 14px;
    color: var(--text-primary);
    cursor: pointer;
    transition: background var(--transition-fast);
  }

  .option:hover:not(.disabled) {
    background: var(--brand-indigo-tint);
  }

  .option.highlighted {
    background: var(--brand-indigo-tint);
  }

  .option.selected {
    background: var(--brand-indigo-tint-medium);
    color: var(--brand-indigo);
    font-weight: 500;
  }

  .option.disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .has-error .select,
  .has-error .select-custom {
    border-color: #DC2626;
  }

  .has-error .select:focus,
  .has-error .select-custom:focus {
    border-color: #DC2626;
    box-shadow: 0 0 0 3px rgba(220, 38, 38, 0.1);
  }

  .error-message {
    font-size: 12px;
    color: #DC2626;
    line-height: var(--line-height-tight);
  }

  .disabled {
    opacity: 0.6;
  }
</style>
