<script lang="ts">
  import type { ActionSpec } from '../descriptor'
  import FormModal from './FormModal.svelte'
  import ConfirmDialog from '../controls/ConfirmDialog.svelte'

  /* ONE action-running path for every archetype (L2). Declarative actions
   * escalate by shape: form → FormModal, confirm → ConfirmDialog, else run.
   *
   * Deliberately non-generic: `bind:this` erases component generics
   * (svelte2tsx instantiates them as unknown), so this seam types rows as
   * `any`. Callers stay fully typed — their descriptor's ActionSpec<Row>
   * is what flows in. */
  type Row = any

  let { reload }: { reload: () => Promise<void> } = $props()

  let formAction = $state<{ action: ActionSpec<Row>; row: Row | null } | null>(null)
  let confirmAction = $state<{ action: ActionSpec<Row>; row: Row | null; message: string } | null>(
    null,
  )

  export function run(action: ActionSpec<Row>, row: Row | null): void {
    if (action.form) {
      formAction = { action, row }
    } else if (action.confirm) {
      confirmAction = { action, row, message: action.confirm(row) }
    } else {
      void action.run({ row, reload })
    }
  }
</script>

{#if formAction}
  <FormModal
    spec={formAction.action.form!}
    row={formAction.row}
    onDone={() => {
      formAction = null
      void reload()
    }}
    onCancel={() => (formAction = null)}
  />
{/if}

{#if confirmAction}
  <ConfirmDialog
    title={confirmAction.action.label}
    message={confirmAction.message}
    confirmLabel={confirmAction.action.label}
    onConfirm={() => {
      const { action, row } = confirmAction!
      confirmAction = null
      void action.run({ row, reload })
    }}
    onCancel={() => (confirmAction = null)}
  />
{/if}
