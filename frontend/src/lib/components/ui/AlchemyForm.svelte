<script lang="ts">
    import { createBubbler, preventDefault } from 'svelte/legacy';

    const bubble = createBubbler();
    interface Props {
        entity_name?: string;
        fields?: any;
    }

    let { entity_name = "", fields = [] }: Props = $props();

    // Generate unique IDs for accessibility
    function getFieldId(field, index) {
        return `alchemy-field-${entity_name.toLowerCase().replace(/\s+/g, '-')}-${field.Label.toLowerCase().replace(/\s+/g, '-')}-${index}`;
    }
</script>

<div class="bg-[var(--bg-color)] border border-[var(--border-color)] p-6 rounded-lg shadow-sm h-full">
    <h3 class="text-xl font-serif mb-6 pb-2 border-b border-[var(--border-color)] opacity-80">{entity_name}</h3>

    <form onsubmit={preventDefault(bubble('submit'))} class="flex flex-wrap gap-4">
        {#each fields as field, index}
            {@const fieldId = getFieldId(field, index)}
            <div style="width: {field.Width === '100%' ? '100%' : `calc(${field.Width} - 16px)`}; min-width: 250px;" class="flex-grow">
                <label for={fieldId} class="block text-xs font-mono uppercase tracking-widest opacity-50 mb-1.5 ml-1">{field.Label}</label>

                {#if field.Type === 'text' || field.Type === 'string'}
                    <input
                        id={fieldId}
                        type="text"
                        placeholder={field.Placeholder}
                        aria-label={field.Label}
                        class="w-full bg-[var(--bg-color)] border border-[var(--border-color)] rounded p-2 text-sm focus:border-[var(--accent-color)] focus:outline-none transition-colors"
                    />
                {:else if field.Type === 'number'}
                     <input
                        id={fieldId}
                        type="number"
                        placeholder={field.Placeholder}
                        aria-label={field.Label}
                        class="w-full bg-[var(--bg-color)] border border-[var(--border-color)] rounded p-2 text-sm focus:border-[var(--accent-color)] focus:outline-none transition-colors"
                    />
                {:else if field.Type === 'enum'}
                     <select
                        id={fieldId}
                        aria-label={field.Label}
                        class="w-full bg-[var(--bg-color)] border border-[var(--border-color)] rounded p-2 text-sm focus:border-[var(--accent-color)] focus:outline-none transition-colors"
                     >
                        <option>Default</option>
                        <option>Option A</option>
                        <option>Option B</option>
                     </select>
                {/if}
            </div>
        {/each}

        <div class="w-full mt-4 flex justify-end">
            <button type="submit" class="bg-[var(--text-color)] text-[var(--bg-color)] px-6 py-2 rounded text-sm font-medium hover:opacity-90 transition-opacity">
                Save {entity_name}
            </button>
        </div>
    </form>
</div>
