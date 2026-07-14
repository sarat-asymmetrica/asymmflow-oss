/* Pure form logic — the vitest-testable half (L5), mirror of ledger-core. */

import type { FormFieldSpec, FormSpec } from './form'

export function isBlank(value: unknown): boolean {
  return (
    value == null ||
    (typeof value === 'string' && value.trim() === '') ||
    (typeof value === 'number' && Number.isNaN(value))
  )
}

/** Fields currently visible for a draft (hidden fields never validate). */
export function visibleFields<Draft>(spec: FormSpec<Draft>, draft: Draft): FormFieldSpec<Draft>[] {
  return spec.fields.filter((f) => !f.visible || f.visible(draft))
}

/** key → error message; empty object = valid. */
export function validateForm<Draft>(spec: FormSpec<Draft>, draft: Draft): Record<string, string> {
  const errors: Record<string, string> = {}
  for (const field of visibleFields(spec, draft)) {
    const value = (draft as Record<string, unknown>)[field.key]
    if (field.required && isBlank(value)) {
      errors[field.key] = `${field.label} is required`
      continue
    }
    if (field.validate && !isBlank(value)) {
      const msg = field.validate(value, draft)
      if (msg) errors[field.key] = msg
    }
  }
  return errors
}
