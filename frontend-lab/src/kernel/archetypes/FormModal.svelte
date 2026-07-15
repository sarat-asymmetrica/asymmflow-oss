<script lang="ts" generics="Draft">
  import type { FormFieldOption, FormSpec } from '../form'
  import { FormViewModel } from '../form.svelte'
  import { visibleFields } from '../form-core'
  import Modal from '../primitives/Modal.svelte'
  import FormGrid from '../primitives/FormGrid.svelte'
  import Button from '../controls/Button.svelte'

  let {
    spec,
    row = null,
    onDone,
    onCancel,
  }: {
    spec: FormSpec<Draft>
    /** The clicked row for row-scoped form actions; null for screen creates. */
    row?: unknown
    /** Called after a successful submit (caller reloads + closes). */
    onDone: () => void
    onCancel: () => void
  } = $props()

  const vm = $derived(new FormViewModel(spec, row))

  // Resolve async select options once per spec instance.
  let optionsMap = $state<Record<string, FormFieldOption[]>>({})
  $effect(() => {
    const fields = spec.fields
    for (const f of fields) {
      if (!f.options) continue
      if (Array.isArray(f.options)) {
        optionsMap[f.key] = f.options
      } else {
        void f.options().then((opts) => {
          optionsMap[f.key] = opts
        })
      }
    }
  })

  async function submit() {
    if (await vm.submit()) onDone()
  }
</script>

<Modal title={spec.title} onClose={onCancel}>
  <FormGrid columns={2}>
    {#each visibleFields(spec, vm.draft) as field (field.key)}
      <label class="k-field" class:k-field-wide={field.kind === 'textarea'}>
        <span class="k-field-label">
          {field.label}{#if field.required}<span class="k-field-req" aria-hidden="true">*</span>{/if}
        </span>

        {#if field.kind === 'select'}
          <select class="k-input" bind:value={(vm.draft as Record<string, any>)[field.key]}>
            <option value="">{field.placeholder ?? 'Select…'}</option>
            {#each optionsMap[field.key] ?? [] as opt (opt.value)}
              <option value={opt.value}>{opt.label}</option>
            {/each}
          </select>
        {:else if field.kind === 'textarea'}
          <textarea
            class="k-input k-input-area"
            placeholder={field.placeholder}
            bind:value={(vm.draft as Record<string, any>)[field.key]}
          ></textarea>
        {:else if field.kind === 'number'}
          <input
            class="k-input"
            type="number"
            step={field.step ?? 'any'}
            placeholder={field.placeholder}
            bind:value={(vm.draft as Record<string, any>)[field.key]}
          />
        {:else if field.kind === 'date'}
          <input
            class="k-input"
            type="date"
            bind:value={(vm.draft as Record<string, any>)[field.key]}
          />
        {:else}
          <input
            class="k-input"
            type="text"
            placeholder={field.placeholder}
            bind:value={(vm.draft as Record<string, any>)[field.key]}
          />
        {/if}

        {#if vm.errors[field.key]}
          <span class="k-field-error">{vm.errors[field.key]}</span>
        {/if}
      </label>
    {/each}
  </FormGrid>

  {#if vm.submitError}
    <p class="k-form-error">Could not save: {vm.submitError}</p>
  {/if}

  {#snippet footer()}
    <Button onclick={onCancel} disabled={vm.submitting}>Cancel</Button>
    <Button variant="primary" onclick={submit} disabled={vm.submitting}>
      {vm.submitting ? 'Saving…' : (spec.submitLabel ?? 'Save')}
    </Button>
  {/snippet}
</Modal>

<style>
  .k-field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }
  .k-field-wide {
    grid-column: 1 / -1;
  }
  .k-field-label {
    font-size: var(--modal-label-size);
    font-weight: var(--modal-label-weight);
    color: var(--text-secondary);
  }
  .k-field-req {
    color: #b3261e;
    margin-left: 2px;
  }
  .k-input {
    font-family: var(--font-ui);
    font-size: var(--modal-body-size);
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    padding: 8px 10px;
    max-width: 100%;
    min-width: 0;
    outline: none;
    transition: border-color var(--motion-fast) var(--ease-standard);
  }
  .k-input:focus {
    border-color: var(--onyx);
  }
  .k-input-area {
    min-height: 72px;
    resize: vertical;
  }
  .k-field-error {
    font-size: var(--meta-size);
    color: #b3261e;
  }
  .k-form-error {
    margin-top: var(--k-space-sm);
    font-size: var(--modal-body-size);
    color: #b3261e;
    overflow-wrap: break-word;
  }
</style>
