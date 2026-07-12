<script lang="ts">
  /**
   * Enterprise Toggle Component
   * Philosophy: Switch-style toggle with indigo active state
   * Follows: Apple-level polish with accessibility
   */
  import { createEventDispatcher } from 'svelte';

  interface Props {
    checked?: boolean;
    label?: string;
    description?: string;
    disabled?: boolean;
    id?: string;
    name?: string;
  }

  let {
    checked = $bindable(false),
    label = '',
    description = '',
    disabled = false,
    id = `toggle-${Math.random().toString(36).substr(2, 9)}`,
    name = ''
  }: Props = $props();

  const dispatch = createEventDispatcher();

  function handleChange(e: Event) {
    const target = e.target as HTMLInputElement;
    checked = target.checked;
    dispatch('change', { checked, event: e });
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (disabled) return;
    if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault();
      checked = !checked;
      dispatch('change', { checked, event: e });
    }
  }
</script>

<div class="toggle-wrapper" class:disabled>
  <label class="toggle-label" for={id}>
    <div class="toggle-control">
      <input
        {id}
        {name}
        type="checkbox"
        bind:checked
        {disabled}
        class="toggle-input"
        onchange={handleChange}
        role="switch"
        aria-checked={checked}
        aria-label={label || undefined}
      />
      <div class="toggle-track" class:checked>
        <div class="toggle-thumb" class:checked></div>
      </div>
    </div>
    {#if label || description}
      <div class="toggle-content">
        {#if label}
          <span class="toggle-text">{label}</span>
        {/if}
        {#if description}
          <span class="toggle-description">{description}</span>
        {/if}
      </div>
    {/if}
  </label>
</div>

<style>
  .toggle-wrapper {
    display: inline-flex;
    width: 100%;
  }

  .toggle-label {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    cursor: pointer;
    user-select: none;
  }

  .disabled .toggle-label {
    cursor: not-allowed;
    opacity: 0.5;
  }

  .toggle-control {
    position: relative;
    flex-shrink: 0;
  }

  .toggle-input {
    position: absolute;
    opacity: 0;
    width: 0;
    height: 0;
  }

  .toggle-track {
    position: relative;
    width: 44px;
    height: 24px;
    background: var(--border);
    border-radius: 12px;
    transition: background var(--transition-fast);
  }

  .toggle-track.checked {
    background: var(--brand-indigo);
  }

  .toggle-thumb {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 20px;
    height: 20px;
    background: white;
    border-radius: 10px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
    transition: transform var(--transition-fast);
  }

  .toggle-thumb.checked {
    transform: translateX(20px);
  }

  .toggle-input:focus-visible + .toggle-track {
    outline: 2px solid var(--brand-indigo);
    outline-offset: 2px;
  }

  .toggle-label:hover:not(.disabled) .toggle-track:not(.checked) {
    background: var(--text-muted);
  }

  .toggle-content {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding-top: 2px;
  }

  .toggle-text {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-primary);
    line-height: var(--line-height-tight);
  }

  .toggle-description {
    font-size: 12px;
    color: var(--text-secondary);
    line-height: var(--line-height-base);
  }

  .disabled {
    opacity: 0.5;
  }
</style>
