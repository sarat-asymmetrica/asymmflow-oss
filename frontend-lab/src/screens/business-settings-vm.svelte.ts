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
  fetchBusinessSettings,
  updateBusinessSettings,
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

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      this.draft = await fetchBusinessSettings()
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
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
