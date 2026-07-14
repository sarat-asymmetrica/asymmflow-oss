/* Form viewmodel — thin rune shell over form-core (L5). */

import type { FormSpec } from './form'
import { validateForm } from './form-core'

export class FormViewModel<Draft> {
  draft = $state() as Draft
  errors = $state<Record<string, string>>({})
  submitting = $state(false)
  submitError = $state<string | null>(null)

  constructor(readonly spec: FormSpec<Draft>) {
    this.draft = spec.initial()
  }

  validate(): boolean {
    this.errors = validateForm(this.spec, this.draft)
    return Object.keys(this.errors).length === 0
  }

  /** Returns true on success (caller closes + reloads). */
  async submit(): Promise<boolean> {
    if (!this.validate()) return false
    this.submitting = true
    this.submitError = null
    try {
      await this.spec.submit(this.draft)
      return true
    } catch (e) {
      this.submitError = e instanceof Error ? e.message : String(e)
      return false
    } finally {
      this.submitting = false
    }
  }
}
