/* OneDrive Import viewmodel — L5's reactive half: all state + bridge calls
 * for the 3-step Wizard (configure paths -> review scanned deals -> run
 * import). OneDriveImport.svelte binds an instance of this and renders on
 * primitives only (L1); it computes nothing itself. Same split as
 * serial-trace.svelte.ts. */

import {
  detectOneDrivePath,
  importOneDriveDeals,
  scanOneDrivePaths,
  validateOneDrivePath,
  type OneDriveImportResult,
  type ReviewDeal,
} from '../bridge/onedrive-import'

export interface PathEntry {
  value: string
  validating: boolean
  valid: boolean | undefined
  estimatedDeals: number | undefined
  error: string | undefined
}

export const WIZARD_STEPS: { key: string; label: string; description?: string }[] = [
  { key: 'configure', label: 'Configure Paths', description: 'Where your deal folders live' },
  { key: 'review', label: 'Review Deals', description: 'Confirm customers to import' },
  { key: 'import', label: 'Import', description: 'Run the import' },
]

function emptyPathEntry(): PathEntry {
  return { value: '', validating: false, valid: undefined, estimatedDeals: undefined, error: undefined }
}

export class OneDriveImportViewModel {
  currentIndex = $state(0)
  paths = $state<PathEntry[]>([emptyPathEntry()])
  detecting = $state(false)

  deals = $state<ReviewDeal[]>([])
  scanErrors = $state<string[]>([])
  scanning = $state(false)
  hasScanned = $state(false)

  results = $state<OneDriveImportResult[]>([])
  importing = $state(false)
  hasImported = $state(false)

  error = $state<string | null>(null)

  /** Wizard-level busy: gates Back/Next together, one flag per KERNEL Wizard's
   * `busy` prop contract. */
  busy = $derived(this.detecting || this.scanning || this.importing)

  validPaths = $derived(
    this.paths.filter((p) => p.valid === true).map((p) => p.value.trim()),
  )

  /** Included iff the checkbox is on AND a customer is confirmed (non-empty). */
  includedDeals = $derived(this.deals.filter((d) => d.selected && d.confirmedCustomerId))

  canAdvance = $derived.by(() => {
    if (this.currentIndex === 0) return this.validPaths.length > 0
    if (this.currentIndex === 1) return this.includedDeals.length > 0
    return false // step 2 (results) is terminal — Next stays disabled
  })

  nextLabel = $derived.by(() => {
    if (this.currentIndex === 0) return 'Scan Paths'
    if (this.currentIndex === 1) return `Import ${this.includedDeals.length} Deal${this.includedDeals.length === 1 ? '' : 's'}`
    return 'Import Complete'
  })

  importedCount = $derived(this.results.filter((r) => r.success).length)

  /** Step-2 results joined back to their deal for display (folder name isn't
   * carried on OneDriveImportResult). */
  resultRows = $derived.by(() =>
    this.results.map((r) => ({
      ...r,
      folderName: this.deals.find((d) => d.localId === r.dealLocalId)?.folderName || r.dealLocalId,
    })),
  )

  /** Prefill path 0 from DetectOneDrivePath — called once from the view's
   * mount effect (same pattern as SerialTrace's vm.loadRecent()). */
  async detectInitialPath(): Promise<void> {
    if (this.paths[0]!.value.trim()) return // user already typed something
    this.detecting = true
    try {
      const detected = await detectOneDrivePath()
      if (detected && !this.paths[0]!.value.trim()) {
        this.paths[0]!.value = detected
        await this.validatePath(0)
      }
    } catch {
      // Detection is advisory only — the user can still type a path by hand.
    } finally {
      this.detecting = false
    }
  }

  addPath(): void {
    this.paths.push(emptyPathEntry())
  }

  removePath(index: number): void {
    if (this.paths.length <= 1) return
    this.paths.splice(index, 1)
  }

  updatePathValue(index: number, value: string): void {
    const entry = this.paths[index]
    if (!entry) return
    entry.value = value
    entry.valid = undefined
    entry.error = undefined
    entry.estimatedDeals = undefined
  }

  async validatePath(index: number): Promise<void> {
    const entry = this.paths[index]
    if (!entry || !entry.value.trim()) return
    entry.validating = true
    try {
      const result = await validateOneDrivePath(entry.value.trim())
      entry.valid = result.valid
      entry.estimatedDeals = result.estimatedDeals
      entry.error = result.error
    } catch (e) {
      entry.valid = false
      entry.error = e instanceof Error ? e.message : String(e)
    } finally {
      entry.validating = false
    }
  }

  async scan(): Promise<void> {
    this.scanning = true
    this.hasScanned = true
    this.error = null
    try {
      const result = await scanOneDrivePaths(this.validPaths)
      this.deals = result.deals
      this.scanErrors = result.errors
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
      this.deals = []
      this.scanErrors = []
    } finally {
      this.scanning = false
    }
  }

  async runImport(): Promise<void> {
    this.importing = true
    this.hasImported = true
    this.error = null
    try {
      this.results = await importOneDriveDeals(this.includedDeals)
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
      this.results = []
    } finally {
      this.importing = false
    }
  }

  /** Wizard's Next: advances the step pointer AND triggers that step's entry
   * action (scan on 0->1, import on 1->2) — mirrors the brief's "on entering,
   * call ScanOneDrivePaths / ImportOneDriveDeals" spec. */
  async goNext(): Promise<void> {
    if (this.busy || !this.canAdvance) return
    if (this.currentIndex === 0) {
      this.currentIndex = 1
      await this.scan()
    } else if (this.currentIndex === 1) {
      this.currentIndex = 2
      await this.runImport()
    }
  }

  goBack(): void {
    if (this.busy || this.currentIndex === 0) return
    this.currentIndex -= 1
    // A stale scan/import error must never bleed backward and be misread as
    // a problem with the step the user just landed on.
    this.error = null
  }

  /** "Start Over" — back to step 0. Keeps the validated paths (no need to
   * retype), but drops scan/import state so a fresh Next re-scans. */
  reset(): void {
    this.currentIndex = 0
    this.deals = []
    this.scanErrors = []
    this.hasScanned = false
    this.results = []
    this.hasImported = false
    this.error = null
  }
}
