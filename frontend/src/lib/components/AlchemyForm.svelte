<script lang="ts">
  
import { devLog } from "$lib/utils/devLog";
import { onMount } from "svelte";
  // GetFormLayout not yet implemented in backend
const GetFormLayout = async (_entity: string) => null;

  interface Props {
    entity?: string;
  }

  let { entity = "customer" }: Props = $props();

  let layout = $state(null);
  let loading = $state(true);

  onMount(async () => {
    try {
      layout = await GetFormLayout(entity);
    } catch (e) {
      devLog.error("Failed to germinate UI:", e);
    } finally {
      loading = false;
    }
  });
</script>

{#if loading}
  <div class="p-8 text-center text-stone-500 animate-pulse font-mono text-sm">
    Initializing {entity} protocol...
  </div>
{:else if layout}
  <div class="alchemy-form p-8 bg-white rounded-xl shadow-lg max-w-5xl mx-auto border border-stone-200">
    
    <header class="mb-8 border-b border-stone-100 pb-4">
      <h2 class="text-3xl font-bold text-primary mb-2 font-display">
        {layout.entity_name}
      </h2>
      <div class="text-xs font-mono text-stone-400 uppercase tracking-widest">
        ID: {layout.seed} • DR: {layout.digital_root}
      </div>
    </header>

    <div class="form-grid">
      {#each layout.fields as field, index}
        {@const fieldId = `${entity}-field-${index}-${field.label.toLowerCase().replace(/\s+/g, '-')}`}
        <div class="field-wrapper" style="width: {field.width || '100%'}">
          <label for={fieldId} class="block text-xs font-bold uppercase tracking-wider mb-2 text-stone-500 break-words">
            {field.label}
          </label>

          {#if field.type === 'text'}
            <input
              id={fieldId}
              type="text"
              placeholder={field.placeholder}
              aria-label={field.label}
              class="form-input w-full"
            />
          {:else if field.type === 'number'}
            <input
              id={fieldId}
              type="number"
              placeholder={field.placeholder}
              aria-label={field.label}
              class="form-input w-full font-mono"
            />
          {:else if field.type === 'boolean'}
            <div class="flex items-center h-12 p-3 bg-stone-50 rounded-lg border border-stone-200">
              <input
                id={fieldId}
                type="checkbox"
                aria-label={field.label}
                class="form-checkbox"
              />
              <span class="text-sm font-medium text-stone-700 ml-3">Enable</span>
            </div>
          {/if}
        </div>
      {/each}
    </div>
    
    <div class="mt-8 flex justify-end gap-4 pt-6 border-t border-stone-100">
        <button class="px-6 py-3 text-sm font-bold uppercase tracking-wider text-stone-500 hover:text-black transition-colors">
            Cancel
        </button>
        <button class="px-8 py-3 bg-black text-white rounded-lg font-bold shadow-lg shadow-gray-200 hover:bg-gray-900 hover:shadow-xl hover:-translate-y-0.5 transition-all active:scale-95">
            Save {layout.entity_name}
        </button>
    </div>
  </div>
{/if}

<style>
  .form-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 1.5rem;
    align-items: flex-start;
  }
  
  .field-wrapper {
    flex-grow: 1;
    min-width: 240px;
  }

  /* CUSTOM INPUT STYLES */
  .form-input {
    padding: 12px 16px;
    border: 1px solid var(--border-strong, #cbd5e1);
    border-radius: 8px;
    font-size: 15px;
    color: var(--text-primary);
    background-color: white;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .form-input:hover {
    border-color: #666;
  }

  .form-input:focus {
    outline: none;
    border-color: black;
    box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.1);
  }

  .form-checkbox {
    width: 1.25rem;
    height: 1.25rem;
    border-radius: 4px;
    border: 2px solid var(--stone);
    color: black;
    cursor: pointer;
  }
  
  .form-checkbox:checked {
    background-color: black;
    border-color: black;
  }
</style>
