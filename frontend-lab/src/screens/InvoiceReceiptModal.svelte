<script lang="ts">
  /* Settlement receipt-capture modal — an L4 ejection (ActionSpec.modal) off the
   * Invoices ledger (owner ruling G1.2). Records a REAL customer receipt against
   * the invoice (amount/date/method/reference), NEVER a status flip: the bridge
   * calls CreateCustomerReceipt with the invoice bound, which funds a Payment row
   * and advances invoice payment state atomically. Field rendering is delegated
   * to the kernel FormModal via a row-scoped FormSpec, so this screen-level file
   * writes no layout CSS (L1); the receipt-specific shape (method vocabulary,
   * bank-transfer reference guard mirroring the server) lives in the spec. */
  import FormModal from '$kernel/archetypes/FormModal.svelte'
  import type { FormSpec } from '$kernel/form'
  import { recordCustomerReceipt, type InvoiceRow } from '../bridge'

  // ActionSpec.modal props contract (kernel/descriptor.ts) — row is `any` by
  // design (ActionHost erases the row generic when it forwards this component).
  let { row, reload, close }: { row: any; reload: () => Promise<void>; close: () => void } = $props()

  // Receipt method vocabulary — mirrors receipt_service.go normalizeCustomerReceiptMethod.
  const RECEIPT_METHODS = ['Bank Transfer', 'Cash', 'Cheque', 'Credit Card', 'LC', 'PDC', 'Other']

  interface ReceiptDraft {
    amount: number | null
    date: string
    method: string
    reference: string
    notes: string
  }

  // Derived so the seam's per-click `row` is read reactively (matches the PO
  // receive modal pattern) rather than captured once — silences the Svelte
  // state_referenced_locally warning and rebuilds the FormViewModel per row.
  const spec: FormSpec<ReceiptDraft> = $derived({
    title: `Record Receipt — ${(row as InvoiceRow | null)?.number ?? 'Invoice'}`,
    submitLabel: 'Record Receipt',
    initial: (r) => ({
      // Seed the full invoice amount as the default receipt (a common full
      // settlement); the user can reduce it for a partial receipt.
      amount: (r as InvoiceRow | null)?.amount ?? null,
      date: new Date().toISOString().slice(0, 10),
      method: 'Bank Transfer',
      reference: '',
      notes: '',
    }),
    fields: [
      {
        key: 'amount',
        label: 'Amount (BHD)',
        kind: 'number',
        required: true,
        step: '0.001',
        validate: (v) => (typeof v === 'number' && v > 0 ? null : 'Amount must be greater than zero'),
      },
      { key: 'date', label: 'Receipt date', kind: 'date', required: true },
      {
        key: 'method',
        label: 'Method',
        kind: 'select',
        required: true,
        options: RECEIPT_METHODS.map((m) => ({ value: m, label: m })),
      },
      {
        key: 'reference',
        label: 'Reference',
        kind: 'text',
        placeholder: 'Cheque no. / transfer ref',
        // Server requires a reference for bank/wire transfers — guard at the form
        // too so the user fixes it before the round trip, not after the error.
        validate: (v, d) =>
          d.method === 'Bank Transfer' && !String(v ?? '').trim() ? 'Reference is required for bank transfers' : null,
      },
      { key: 'notes', label: 'Notes', kind: 'textarea', placeholder: 'Optional' },
    ],
    submit: async (draft, r) => {
      const inv = r as InvoiceRow
      await recordCustomerReceipt({
        invoiceId: inv.id,
        amount: draft.amount ?? 0,
        date: draft.date,
        method: draft.method,
        reference: draft.reference.trim(),
        notes: draft.notes.trim(),
      })
    },
  })
</script>

<FormModal
  spec={spec}
  row={row}
  onDone={async () => {
    await reload()
    close()
  }}
  onCancel={close}
/>
