/* The form schema — forms as declared data, same philosophy as ledgers.
 * A FormSpec describes fields, validation, and submission; the FormModal
 * archetype renders it. Line-item repeaters (proforma/credit-note items)
 * are a planned field kind — until then those flows eject to slots (L4). */

export interface FormFieldOption {
  value: string
  label: string
}

export interface FormFieldSpec<Draft> {
  key: keyof Draft & string
  label: string
  kind: 'text' | 'textarea' | 'number' | 'date' | 'select'
  required?: boolean
  placeholder?: string
  /** Input step for number fields — '0.001' for BHD money. */
  step?: string
  /** Static options, or async (e.g. open orders fetched when the form opens). */
  options?: FormFieldOption[] | (() => Promise<FormFieldOption[]>)
  /** Conditional fields — evaluated against the live draft. */
  visible?: (draft: Draft) => boolean
  /** Return an error message, or null when valid. Runs after `required`. */
  validate?: (value: unknown, draft: Draft) => string | null
}

export interface FormSpec<Draft> {
  title: string
  submitLabel?: string
  initial: () => Draft
  fields: FormFieldSpec<Draft>[]
  submit: (draft: Draft) => Promise<void>
}
