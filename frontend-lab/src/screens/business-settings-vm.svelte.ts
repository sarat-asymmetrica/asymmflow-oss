/* Business Settings viewmodel — L5's reactive half. Load once, edit a flat
 * draft in place, Save. No DocumentLedger/FormModal machinery here: this is
 * the one genuinely bespoke form in the K4 SettingsScreen split (see
 * screens/parity/Settings.parity.md) — a single settings record, not a
 * list, so the ledger/form archetypes don't fit.
 *
 * Named `business-settings-vm` (not `.svelte.ts` on the component stem) so
 * it never collides case-insensitively with `BusinessSettings.svelte` on
 * Windows (same fix as pricing-vm.svelte.ts). */

import {
  fetchAIProviderKey,
  fetchBusinessSettings,
  saveAIProviderKey,
  updateBusinessSettings,
  type AIProviderKeyState,
  type BusinessSettingsData,
} from '../bridge/business-settings'

const BLANK: BusinessSettingsData = {
  companyName: '',
  baseCurrency: 'BHD',
  defaultMarginPercent: 0,
  vatRatePercent: 0,
  fiscalYearStartMonth: 1,
}

export class BusinessSettingsViewModel {
  draft = $state<BusinessSettingsData>({ ...BLANK })
  loading = $state(true)
  error = $state<string | null>(null)

  saving = $state(false)
  saveError = $state<string | null>(null)
  /** True briefly after a successful save — the screen shows a "Saved" note. */
  saved = $state(false)

  // ---- R4: AI provider (Butler/Mistral) key. The masked state is all we ever
  // hold; `aiKeyInput` is the (write-only) plaintext the operator types, cleared
  // the moment it's submitted so the secret never lingers in VM state.
  aiKey = $state<AIProviderKeyState>({ maskedKey: '(not set)', isSet: false })
  aiKeyInput = $state('')
  aiKeySaving = $state(false)
  aiKeyError = $state<string | null>(null)
  aiKeySaved = $state(false)

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      this.draft = await fetchBusinessSettings()
      // Best-effort: a missing AI key must not block the settings form.
      this.aiKey = await fetchAIProviderKey().catch(() => ({ maskedKey: '(not set)', isSet: false }))
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  async saveAIKey(): Promise<void> {
    const key = this.aiKeyInput.trim()
    if (!key) {
      this.aiKeyError = 'Enter the API key before saving.'
      return
    }
    this.aiKeySaving = true
    this.aiKeyError = null
    this.aiKeySaved = false
    try {
      await saveAIProviderKey(key)
      // Never keep the plaintext around; re-read the server-masked value.
      this.aiKeyInput = ''
      this.aiKey = await fetchAIProviderKey().catch(() => this.aiKey)
      this.aiKeySaved = true
    } catch (e) {
      this.aiKeyError = e instanceof Error ? e.message : String(e)
    } finally {
      this.aiKeySaving = false
    }
  }

  async save(): Promise<void> {
    this.saving = true
    this.saveError = null
    this.saved = false
    try {
      await updateBusinessSettings(this.draft)
      this.saved = true
    } catch (e) {
      this.saveError = e instanceof Error ? e.message : String(e)
    } finally {
      this.saving = false
    }
  }
}
