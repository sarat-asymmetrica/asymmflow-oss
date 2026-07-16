<script lang="ts">
  /* Receive Items against PO — an L4 ejection (ActionSpec.modal) off the
   * Purchase Orders ledger: a bespoke line-item capture flow a flat FormSpec
   * can't express (per-line receiving/rejected quantities → GRNItem[]). Pure
   * bespoke-on-primitives (K5), same shape as Accounting.svelte's voucher
   * modal — Modal + LineItemsEditor, no local math (purchase-order-receive-vm
   * owns validate/build; this file only binds + renders, L5). */
  import { onMount } from 'svelte'
  import Modal from '$kernel/primitives/Modal.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import CalloutWidget from '$kernel/widgets/CalloutWidget.svelte'
  import LineItemsEditor from '$kernel/widgets/LineItemsEditor.svelte'
  import { sumField } from '$kernel/line-items'
  import type { LineColumn, LineFooterCell } from '$kernel/line-items'
  import {
    PurchaseOrderReceiveViewModel,
    blankReceiveRow,
    validateRow,
    type ReceiveRow,
  } from './purchase-order-receive-vm.svelte'

  // ActionSpec.modal's props contract (kernel/descriptor.ts) — row is typed
  // `any` deliberately (the seam is any-by-design; ActionHost erases the row
  // generic when it forwards this component).
  let { row, reload, close }: { row: any; reload: () => Promise<void>; close: () => void } = $props()

  const vm = $derived(new PurchaseOrderReceiveViewModel(row?.id ?? ''))
  onMount(() => void vm.load())

  const columns: LineColumn<ReceiveRow>[] = [
    { key: 'productCode', label: 'Product', kind: 'readonly', content: 'code', minWidth: 130, value: (r) => r.productCode },
    { key: 'description', label: 'Description', kind: 'readonly', content: 'text', grow: true, minWidth: 220, value: (r) => r.description },
    { key: 'quantityOrdered', label: 'Ordered', kind: 'readonly', content: 'quantity', minWidth: 90, value: (r) => r.quantityOrdered },
    {
      key: 'quantityAlreadyReceived',
      label: 'Already Rcvd',
      kind: 'readonly',
      content: 'quantity',
      minWidth: 100,
      value: (r) => r.quantityAlreadyReceived,
    },
    {
      key: 'quantityReceiving',
      label: 'Receiving',
      kind: 'number',
      minWidth: 100,
      value: (r) => r.quantityReceiving,
      set: (r, v) => {
        r.quantityReceiving = Number(v) || 0
      },
      tone: (r) => (validateRow(r) ? 'danger' : 'neutral'),
    },
    {
      key: 'quantityRejected',
      label: 'Rejected',
      kind: 'number',
      minWidth: 90,
      value: (r) => r.quantityRejected,
      set: (r, v) => {
        r.quantityRejected = Number(v) || 0
      },
      tone: (r) => (validateRow(r) ? 'danger' : 'neutral'),
    },
    {
      key: 'rejectionReason',
      label: 'Rejection Reason',
      kind: 'text',
      wide: true,
      value: (r) => r.rejectionReason,
      set: (r, v) => {
        r.rejectionReason = String(v)
      },
    },
  ]

  const lineFooter: LineFooterCell<ReceiveRow>[] = [
    { label: 'Total Receiving', content: 'quantity', value: (rows) => sumField(rows, (r) => r.quantityReceiving) },
  ]

  // Fixed line list — the PO's own items, not an arbitrary repeater — so
  // add/remove are pinned off (minRows === maxRows === current length).
  const rowErrors = $derived(
    vm.rows
      .map((r, i) => ({ line: i + 1, msg: validateRow(r) }))
      .filter((e): e is { line: number; msg: string } => e.msg != null),
  )

  async function onSubmit() {
    const ok = await vm.submit()
    if (ok) {
      await reload()
      close()
    }
  }
</script>

<Modal title={`Receive Items — ${row?.poNumber ?? 'Purchase Order'}`} onClose={close}>
  {#if vm.loading}
    <EmptyState message="Loading PO line items…" />
  {:else if vm.error}
    <CalloutWidget items={[{ label: 'Could not load PO items', text: vm.error, tone: 'danger' }]} />
  {:else}
    <Stack gap="md">
      <LineItemsEditor
        columns={columns}
        rows={vm.rows}
        createRow={() => blankReceiveRow()}
        onAdd={() => {}}
        onRemove={() => {}}
        minRows={vm.rows.length}
        maxRows={vm.rows.length}
        footer={lineFooter}
        disabled={vm.submitting}
        emptyMessage="No open line items on this PO."
      />
      {#if rowErrors.length}
        <CalloutWidget
          items={rowErrors.map((e) => ({ label: `Line ${e.line}`, text: e.msg, tone: 'danger' }))}
        />
      {/if}
      {#if vm.submitError}
        <CalloutWidget items={[{ label: 'Could not receive items', text: vm.submitError, tone: 'danger' }]} />
      {/if}
    </Stack>
  {/if}

  {#snippet footer()}
    <Button onclick={close} disabled={vm.submitting}>Cancel</Button>
    <Button variant="primary" onclick={() => void onSubmit()} disabled={!vm.canSubmit}>
      {vm.submitting ? 'Receiving…' : 'Receive Items'}
    </Button>
  {/snippet}
</Modal>
