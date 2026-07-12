<script lang="ts">
  /**
   * OneDriveImportScreen — 3-Step Offer Folder Import Wizard
   *
   * Step 1: Configure & validate OneDrive folder paths
   * Step 2: Review matched deals, assign customers, select for import
   * Step 3: Execute import with live per-deal progress
   *
   * Design System: Onyx & Ether (light — carbon/onyx/ether palette)
   */

  import { fade, fly } from 'svelte/transition';
  import { toast } from '$lib/stores/toasts';
  import {
    ValidateOneDrivePath } from '../../../wailsjs/go/main/App';
import { ScanOneDrivePaths, ConfirmOneDriveDeal, ImportOneDriveDeals } from '../../../wailsjs/go/main/InfraService';
  import { main } from '../../../wailsjs/go/models';

  // ─── Wizard state ───────────────────────────────────────────────────────────
  let currentStep = $state(1);

  // Step 1 state
  interface PathEntry {
    id: number;
    value: string;
    validating: boolean;
    validation: { valid: boolean; estimated_deals: number; error?: string } | null;
  }
  let pathIdCounter = 1;
  let paths: PathEntry[] = $state([{ id: pathIdCounter++, value: '', validating: false, validation: null }]);
  let scanning = $state(false);

  // Step 2 state
  interface CustomerMatch {
    customer_id: string;
    business_name: string;
    score: number;
  }
  interface Deal {
    local_id: string;
    id: string;
    folder_path: string;
    folder_name: string;
    final_path: string;
    root_path: string;
    instrument_type: string;
    year: number | null;
    file_count: number;
    excel_count: number;
    pdf_count: number;
    has_costing_sheet: boolean;
    files: main.DiscoveredFile[];
    customer_matches: CustomerMatch[];
    confirmed_customer_id: string;
  }
  let scanResult: { total_deals: number; total_files: number; deals: Deal[] } | null = $state(null);
  let selectedDealIDs: Set<string> = $state(new Set());
  let reviewFilter: 'all' | 'high' | 'needs_review' | 'costing' = $state('all');

  // Step 3 state
  let importing = $state(false);
  let importLog: Array<{ deal_id: string; folder_name: string; success: boolean; error: string | null; done: boolean }> = $state([]);
  let importDone = $state(false);

  // ─── Computed ────────────────────────────────────────────────────────────────
  let validPaths = $derived(paths.filter(p => p.validation?.valid));
  let canScan = $derived(validPaths.length > 0 && !scanning);

  let filteredDeals = $derived((scanResult?.deals ?? []).filter(deal => {
    const topScore = deal.customer_matches[0]?.score ?? 0;
    if (reviewFilter === 'high') return topScore >= 0.9;
    if (reviewFilter === 'needs_review') return topScore < 0.7;
    if (reviewFilter === 'costing') return deal.has_costing_sheet;
    return true;
  }));

  let selectedCount = $derived(selectedDealIDs.size);
  let importSuccessCount = $derived(importLog.filter(r => r.success).length);
  let importErrorCount = $derived(importLog.filter(r => !r.success && r.done).length);

  function parseDealYear(yearHint?: string): number | null {
    const match = (yearHint || '').match(/\b(20\d{2})\b/);
    return match ? Number(match[1]) : null;
  }

  function toDealView(raw: main.DiscoveredDeal): Deal {
    const files = raw.files || [];
    const excelCount = files.filter(f => ['.xlsx', '.xls'].includes((f.extension || '').toLowerCase())).length;
    const pdfCount = files.filter(f => (f.extension || '').toLowerCase() === '.pdf').length;

    return {
      ...raw,
      id: raw.local_id,
      local_id: raw.local_id,
      folder_path: raw.folder_path,
      folder_name: raw.folder_name,
      final_path: raw.final_path,
      root_path: raw.root_path,
      instrument_type: raw.instrument_type,
      year: parseDealYear(raw.year_hint),
      file_count: files.length,
      excel_count: excelCount,
      pdf_count: pdfCount,
      has_costing_sheet: files.some(f => f.file_type === 'costing_sheet'),
      files,
      customer_matches: (raw.customer_matches || []).map(match => ({
        customer_id: match.customer_id,
        business_name: match.business_name,
        score: match.score,
      })),
      confirmed_customer_id: raw.confirmed_customer_id || '',
    };
  }

  // ─── Step 1: Path management ─────────────────────────────────────────────────
  function addPath() {
    paths = [...paths, { id: pathIdCounter++, value: '', validating: false, validation: null }];
  }

  function removePath(id: number) {
    if (paths.length === 1) return;
    paths = paths.filter(p => p.id !== id);
  }

  async function validatePath(entry: PathEntry) {
    if (!entry.value.trim()) {
      toast.warning('Enter a folder path first');
      return;
    }
    entry.validating = true;
    entry.validation = null;
    paths = paths; // trigger reactivity
    try {
      const result = await ValidateOneDrivePath(entry.value.trim());
      entry.validation = {
        valid: Boolean(result?.valid),
        estimated_deals: Number(result?.estimated_deals ?? 0),
        error: result?.error ? String(result.error) : undefined,
      };
    } catch (e) {
      entry.validation = { valid: false, estimated_deals: 0, error: String(e) };
    } finally {
      entry.validating = false;
      paths = paths;
    }
  }

  async function startScan() {
    if (!canScan) return;
    scanning = true;
    try {
      const pathValues = validPaths.map(p => p.value.trim());
      const result = await ScanOneDrivePaths(pathValues);
      scanResult = {
        total_deals: result?.total_folders ?? result?.deals?.length ?? 0,
        total_files: result?.total_files ?? 0,
        deals: (result?.deals ?? []).map(toDealView),
      };

      // Pre-select confirmed matches and initialize confirmed customer
      if (scanResult?.deals) {
        for (const deal of scanResult.deals) {
          const top = deal.customer_matches?.[0];
          if (top && top.score >= 0.85) {
            deal.confirmed_customer_id = top.customer_id;
            selectedDealIDs.add(deal.id);
          }
        }
        selectedDealIDs = new Set(selectedDealIDs); // trigger reactivity
      }

      currentStep = 2;
    } catch (e) {
      toast.danger('Scan failed: ' + e);
    } finally {
      scanning = false;
    }
  }

  // ─── Step 2: Review & selection ──────────────────────────────────────────────
  function selectAllHighConfidence() {
    for (const deal of scanResult?.deals ?? []) {
      if ((deal.customer_matches[0]?.score ?? 0) >= 0.9) {
        selectedDealIDs.add(deal.id);
      }
    }
    selectedDealIDs = new Set(selectedDealIDs);
  }

  function toggleDeal(id: string) {
    if (selectedDealIDs.has(id)) {
      selectedDealIDs.delete(id);
    } else {
      selectedDealIDs.add(id);
    }
    selectedDealIDs = new Set(selectedDealIDs);
  }

  type ReviewFilter = 'all' | 'high' | 'needs_review' | 'costing';
  function setFilter(id: string) {
    reviewFilter = id as ReviewFilter;
  }

  function setCustomer(deal: Deal, customerId: string) {
    deal.confirmed_customer_id = customerId;
    scanResult = scanResult; // trigger reactivity
  }

  function proceedToImport() {
    if (selectedCount === 0) {
      toast.warning('Select at least one deal to import');
      return;
    }
    importLog = [];
    importDone = false;
    currentStep = 3;
  }

  // ─── Step 3: Import ───────────────────────────────────────────────────────────
  async function runImport() {
    if (importing) return;
    importing = true;
    importDone = false;

    const deals = (scanResult?.deals ?? []).filter(d => selectedDealIDs.has(d.id));
    importLog = deals.map(d => ({ deal_id: d.id, folder_name: d.folder_name, success: false, error: null, done: false }));

    try {
      const result = await ImportOneDriveDeals(
        deals.map(d => main.DiscoveredDeal.createFrom({
          local_id: d.local_id,
          folder_path: d.folder_path,
          folder_name: d.folder_name,
          final_path: d.final_path,
          root_path: d.root_path,
          customer_matches: d.customer_matches,
          files: d.files,
          instrument_type: d.instrument_type,
          year_hint: d.year ? String(d.year) : '',
          status: 'confirmed',
          confirmed_customer_id: d.confirmed_customer_id,
        }))
      );

      // Update log entries from result
      for (const r of result || []) {
        const entry = importLog.find(l => l.deal_id === r.deal_local_id);
          if (entry) {
            entry.success = r.success;
            entry.error = r.success ? null : (r.message || 'Import failed');
            entry.done = true;
          }
      }
    } catch (e) {
      // Mark all undone as failed
      for (const entry of importLog) {
        if (!entry.done) {
          entry.success = false;
          entry.error = String(e);
          entry.done = true;
        }
      }
      toast.danger('Import failed: ' + e);
    } finally {
      importing = false;
      importDone = true;
      importLog = importLog;
    }
  }

  function resetWizard() {
    currentStep = 1;
    paths = [{ id: pathIdCounter++, value: '', validating: false, validation: null }];
    scanResult = null;
    selectedDealIDs = new Set();
    importLog = [];
    importDone = false;
  }

  function navigateToOffers() {
    window.dispatchEvent(new CustomEvent('navigate', { detail: { screen: 'offers' } }));
  }

  // ─── Helpers ─────────────────────────────────────────────────────────────────
  function scoreColor(score: number): string {
    if (score >= 0.9) return 'badge-score-high';
    if (score >= 0.7) return 'badge-score-mid';
    return 'badge-score-low';
  }

  function formatScore(score: number): string {
    return Math.round(score * 100) + '%';
  }

  function truncate(str: string, max: number): string {
    return str.length > max ? str.slice(0, max) + '…' : str;
  }

  function fileBreakdown(deal: Deal): string {
    const parts = [];
    if (deal.excel_count > 0) parts.push(`${deal.excel_count} Excel`);
    if (deal.pdf_count > 0) parts.push(`${deal.pdf_count} PDF`);
    if (parts.length === 0) return `${deal.file_count} file${deal.file_count !== 1 ? 's' : ''}`;
    return `${deal.file_count} file${deal.file_count !== 1 ? 's' : ''} (${parts.join(', ')})`;
  }
</script>

<!-- ═══════════════════════════════════════════════════════════════════════════
     TEMPLATE
     ═══════════════════════════════════════════════════════════════════════════ -->

<div class="page">

  <!-- Page header -->
  <div class="page-header">
    <div>
      <h1>OneDrive Offer Import</h1>
      <p class="page-subtitle">Scan locally-synced OneDrive folders and import offer deals with automatic customer matching</p>
    </div>
    <!-- Step breadcrumb -->
    <nav class="breadcrumb" aria-label="Wizard progress">
      {#each [{ n: 1, label: 'Configure Paths' }, { n: 2, label: 'Review Deals' }, { n: 3, label: 'Import' }] as step}
        <div class="bc-step" class:bc-active={currentStep === step.n} class:bc-done={currentStep > step.n}>
          <span class="bc-num">{currentStep > step.n ? '✓' : step.n}</span>
          <span class="bc-label">{step.label}</span>
        </div>
        {#if step.n < 3}
          <div class="bc-line" class:bc-line-done={currentStep > step.n}></div>
        {/if}
      {/each}
    </nav>
  </div>

  <!-- ── STEP 1: Configure Paths ─────────────────────────────────────────────── -->
  {#if currentStep === 1}
    <div class="step-body" in:fade={{ duration: 200 }}>
      <div class="card step-card">
        <div class="card-header">
          <h2>Folder Paths</h2>
          <p class="card-desc">
            Add the full paths to your locally-synced OneDrive offer folders. For 2025 data you may need multiple folders.
            Each path must be accessible on this machine.
          </p>
        </div>

        <div class="paths-list">
          {#each paths as entry (entry.id)}
            <div class="path-row" in:fly={{ y: 8, duration: 180 }}>
              <div class="path-input-wrap">
                <input
                  type="text"
                  class="input path-input"
                  class:input-error={entry.validation && !entry.validation.valid}
                  class:input-success={entry.validation?.valid}
                  bind:value={entry.value}
                  placeholder="e.g. /Users/yourname/OneDrive - Acme Instrumentation/Offers/2025"
                  onkeydown={e => e.key === 'Enter' && validatePath(entry)}
                />

                <div class="path-actions">
                  <button
                    class="btn btn-secondary"
                    onclick={() => validatePath(entry)}
                    disabled={entry.validating || !entry.value.trim()}
                  >
                    {#if entry.validating}
                      <span class="spinner-tiny"></span> Validating…
                    {:else}
                      Validate
                    {/if}
                  </button>

                  {#if paths.length > 1}
                    <button class="btn btn-ghost btn-icon-only" title="Remove path" onclick={() => removePath(entry.id)}>
                      <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5">
                        <line x1="2" y1="2" x2="12" y2="12"/><line x1="12" y1="2" x2="2" y2="12"/>
                      </svg>
                    </button>
                  {/if}
                </div>
              </div>

              {#if entry.validation}
                <div class="validation-result" class:valid={entry.validation.valid} class:invalid={!entry.validation.valid}>
                  {#if entry.validation.valid}
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="7" cy="7" r="6"/><polyline points="4.5,7 6.5,9 9.5,5"/>
                    </svg>
                    ~{entry.validation.estimated_deals} deals found
                  {:else}
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="7" cy="7" r="6"/><line x1="5" y1="5" x2="9" y2="9"/><line x1="9" y1="5" x2="5" y2="9"/>
                    </svg>
                    {entry.validation.error ?? 'Path not found or not accessible'}
                  {/if}
                </div>
              {/if}
            </div>
          {/each}
        </div>

        <div class="path-footer">
          <button class="btn btn-ghost" onclick={addPath}>
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="7" y1="2" x2="7" y2="12"/><line x1="2" y1="7" x2="12" y2="7"/>
            </svg>
            Add Another Path
          </button>

          <div class="path-footer-right">
            {#if validPaths.length > 0}
              <span class="path-summary">
                {validPaths.length} valid path{validPaths.length !== 1 ? 's' : ''} —
                ~{validPaths.reduce((sum, p) => sum + (p.validation?.estimated_deals ?? 0), 0)} estimated deals
              </span>
            {/if}

            <button
              class="btn btn-primary"
              onclick={startScan}
              disabled={!canScan}
            >
              {#if scanning}
                <span class="spinner-tiny spinner-white"></span> Scanning…
              {:else}
                Start Scan
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3,7 7,3 11,7"/><line x1="7" y1="3" x2="7" y2="11"/>
                </svg>
              {/if}
            </button>
          </div>
        </div>

        {#if scanning}
          <div class="scan-progress" in:fade>
            <div class="scan-bar">
              <div class="scan-bar-fill"></div>
            </div>
            <p class="scan-status">Scanning folders for offer documents…</p>
          </div>
        {/if}
      </div>

      <!-- Tips card -->
      <div class="card tips-card">
        <h3>Tips</h3>
        <ul class="tips-list">
          <li>On Mac, OneDrive typically syncs to <code>~/Library/CloudStorage/OneDrive-…</code> or <code>~/OneDrive</code></li>
          <li>On Windows, usually <code>C:\Users\You\OneDrive - Acme Instrumentation WLL\Offers\2025</code></li>
          <li>Validate each path before scanning — validation checks folder accessibility and counts deal subfolders</li>
          <li>Add multiple paths if 2025 offers are split across separate base folders</li>
          <li>The scan reads folder names and file metadata only — no file content is uploaded</li>
        </ul>
      </div>
    </div>
  {/if}

  <!-- ── STEP 2: Review Deals ──────────────────────────────────────────────────── -->
  {#if currentStep === 2 && scanResult}
    <div class="step-body" in:fade={{ duration: 200 }}>

      <!-- Summary bar -->
      <div class="summary-bar">
        <div class="summary-stats">
          <div class="stat-pill">
            <span class="stat-value">{scanResult.total_deals}</span>
            <span class="stat-label">Deals Found</span>
          </div>
          <div class="stat-pill">
            <span class="stat-value">{scanResult.total_files}</span>
            <span class="stat-label">Files Indexed</span>
          </div>
          <div class="stat-pill">
            <span class="stat-value">{(scanResult.deals ?? []).filter(d => (d.customer_matches[0]?.score ?? 0) >= 0.9).length}</span>
            <span class="stat-label">High Confidence</span>
          </div>
          <div class="stat-pill">
            <span class="stat-value">{(scanResult.deals ?? []).filter(d => (d.customer_matches[0]?.score ?? 0) < 0.7).length}</span>
            <span class="stat-label">Need Review</span>
          </div>
          <div class="stat-pill">
            <span class="stat-value">{(scanResult.deals ?? []).filter(d => d.has_costing_sheet).length}</span>
            <span class="stat-label">With Costing Sheet</span>
          </div>
        </div>

        <div class="summary-actions">
          <button class="btn btn-secondary" onclick={() => { currentStep = 1; }}>
            ← Back to Paths
          </button>
        </div>
      </div>

      <!-- Filter + Select All row -->
      <div class="table-toolbar">
        <div class="filter-tabs">
          {#each [
            { id: 'all', label: 'All', count: scanResult.deals?.length ?? 0 },
            { id: 'high', label: 'High Confidence', count: (scanResult.deals ?? []).filter(d => (d.customer_matches[0]?.score ?? 0) >= 0.9).length },
            { id: 'needs_review', label: 'Needs Review', count: (scanResult.deals ?? []).filter(d => (d.customer_matches[0]?.score ?? 0) < 0.7).length },
            { id: 'costing', label: 'Costing Sheet', count: (scanResult.deals ?? []).filter(d => d.has_costing_sheet).length },
          ] as f}
            <button
              class="filter-tab"
              class:active={reviewFilter === f.id}
              onclick={() => setFilter(f.id)}
            >
              {f.label}
              <span class="filter-count">{f.count}</span>
            </button>
          {/each}
        </div>

        <div class="toolbar-right">
          <button class="btn btn-ghost" onclick={selectAllHighConfidence}>
            Select All High Confidence
          </button>
          <span class="selection-count">{selectedCount} selected</span>
        </div>
      </div>

      <!-- Deals table -->
      <div class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th class="col-check"></th>
              <th class="col-folder">Folder Name</th>
              <th class="col-instrument">Instrument</th>
              <th class="col-year">Year</th>
              <th class="col-files">Files</th>
              <th class="col-match">Top Match</th>
              <th class="col-score">Score</th>
              <th class="col-customer">Customer</th>
              <th class="col-cs">Costing</th>
            </tr>
          </thead>
          <tbody>
            {#each filteredDeals as deal (deal.id)}
              {@const topMatch = deal.customer_matches[0]}
              {@const topScore = topMatch?.score ?? 0}
              <tr class:row-selected={selectedDealIDs.has(deal.id)} class:row-unmatched={topScore < 0.7}>
                <!-- Include checkbox -->
                <td class="col-check">
                  <input
                    type="checkbox"
                    class="deal-checkbox"
                    checked={selectedDealIDs.has(deal.id)}
                    onchange={() => toggleDeal(deal.id)}
                  />
                </td>

                <!-- Folder name with tooltip -->
                <td class="col-folder">
                  <span class="folder-name" title={deal.folder_name}>
                    {truncate(deal.folder_name, 42)}
                  </span>
                </td>

                <!-- Instrument -->
                <td class="col-instrument">
                  {#if deal.instrument_type}
                    <span class="badge badge-neutral">{deal.instrument_type}</span>
                  {:else}
                    <span class="text-muted">—</span>
                  {/if}
                </td>

                <!-- Year -->
                <td class="col-year">
                  {#if deal.year}
                    <span class="year-chip">{deal.year}</span>
                  {:else}
                    <span class="text-muted">—</span>
                  {/if}
                </td>

                <!-- Files -->
                <td class="col-files">
                  <span class="file-count">{fileBreakdown(deal)}</span>
                </td>

                <!-- Top match name -->
                <td class="col-match">
                  {#if topMatch}
                    <span class="match-name">{topMatch.business_name}</span>
                  {:else}
                    <span class="text-muted no-match">No match</span>
                  {/if}
                </td>

                <!-- Score badge -->
                <td class="col-score">
                  {#if topMatch}
                    <span class="score-badge {scoreColor(topScore)}">
                      {formatScore(topScore)}
                    </span>
                  {:else}
                    <span class="text-muted">—</span>
                  {/if}
                </td>

                <!-- Customer selector -->
                <td class="col-customer">
                  <select
                    class="customer-select"
                    value={deal.confirmed_customer_id}
                    onchange={e => setCustomer(deal, e.currentTarget.value)}
                  >
                    <option value="">— Skip (don't import) —</option>
                    {#each deal.customer_matches as match}
                      <option value={match.customer_id}>
                        {match.business_name} ({formatScore(match.score)})
                      </option>
                    {/each}
                  </select>
                </td>

                <!-- Costing sheet indicator -->
                <td class="col-cs align-center">
                  {#if deal.has_costing_sheet}
                    <span class="cs-dot" title="Costing sheet found"></span>
                  {:else}
                    <span class="text-muted">—</span>
                  {/if}
                </td>
              </tr>
            {:else}
              <tr>
                <td colspan="9" class="empty-row">No deals match this filter</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <!-- Proceed bar -->
      <div class="proceed-bar">
        <p class="proceed-hint">
          {#if selectedCount === 0}
            Select deals above to proceed. Use the customer dropdown to assign each deal before importing.
          {:else}
            {selectedCount} deal{selectedCount !== 1 ? 's' : ''} selected for import.
            {(scanResult?.deals ?? []).filter(d => selectedDealIDs.has(d.id) && !d.confirmed_customer_id).length > 0
              ? `⚠ ${(scanResult?.deals ?? []).filter(d => selectedDealIDs.has(d.id) && !d.confirmed_customer_id).length} have no customer assigned — they will be imported without a customer link.`
              : 'All selected deals have customer assignments.'}
          {/if}
        </p>
        <button
          class="btn btn-primary"
          onclick={proceedToImport}
          disabled={selectedCount === 0}
        >
          Proceed to Import ({selectedCount})
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="3,7 7,11 11,7"/><line x1="7" y1="3" x2="7" y2="11"/>
          </svg>
        </button>
      </div>
    </div>
  {/if}

  <!-- ── STEP 3: Import Progress ────────────────────────────────────────────────── -->
  {#if currentStep === 3}
    <div class="step-body" in:fade={{ duration: 200 }}>
      <div class="card step-card">
        <div class="card-header">
          <h2>Import {selectedCount} Deal{selectedCount !== 1 ? 's' : ''}</h2>
          <p class="card-desc">
            Each deal folder will be read, its documents parsed, and an offer record created in AsymmFlow.
            Costing sheets will be linked automatically.
          </p>
        </div>

        {#if !importing && !importDone}
          <!-- Pre-import confirmation -->
          <div class="import-confirm">
            <div class="import-confirm-stats">
              <div class="ic-row">
                <span class="ic-label">Deals to import</span>
                <span class="ic-value">{selectedCount}</span>
              </div>
              <div class="ic-row">
                <span class="ic-label">With customer assignment</span>
                <span class="ic-value">
                  {(scanResult?.deals ?? []).filter(d => selectedDealIDs.has(d.id) && d.confirmed_customer_id).length}
                </span>
              </div>
              <div class="ic-row">
                <span class="ic-label">With costing sheet</span>
                <span class="ic-value">
                  {(scanResult?.deals ?? []).filter(d => selectedDealIDs.has(d.id) && d.has_costing_sheet).length}
                </span>
              </div>
              <div class="ic-row">
                <span class="ic-label">Without customer (will still import)</span>
                <span class="ic-value ic-warn">
                  {(scanResult?.deals ?? []).filter(d => selectedDealIDs.has(d.id) && !d.confirmed_customer_id).length}
                </span>
              </div>
            </div>

            <div class="import-actions-row">
              <button class="btn btn-secondary" onclick={() => { currentStep = 2; }}>
                ← Back to Review
              </button>
              <button class="btn btn-primary" onclick={runImport}>
                Import {selectedCount} Deal{selectedCount !== 1 ? 's' : ''}
              </button>
            </div>
          </div>
        {/if}

        {#if importing || importDone}
          <!-- Progress list -->
          <div class="import-progress-header">
            {#if importing}
              <span class="spinner-tiny"></span>
              <span>Importing… {importLog.filter(r => r.done).length} / {importLog.length}</span>
            {:else if importDone}
              <span>
                Import complete — {importSuccessCount} succeeded,
                {#if importErrorCount > 0}<span class="err-text">{importErrorCount} failed</span>{:else}0 failed{/if}
              </span>
            {/if}

            {#if importDone}
              <!-- Overall progress bar -->
              <div class="progress-bar-wrap">
                <div class="progress-bar">
                  <div
                    class="progress-fill"
                    class:progress-fill-error={importErrorCount > 0}
                    style="width: {importLog.length > 0 ? Math.round((importSuccessCount / importLog.length) * 100) : 0}%"
                  ></div>
                </div>
                <span class="progress-pct">
                  {importLog.length > 0 ? Math.round((importSuccessCount / importLog.length) * 100) : 0}%
                </span>
              </div>
            {/if}
          </div>

          <div class="import-log">
            {#each importLog as entry (entry.deal_id)}
              <div
                class="log-row"
                class:log-success={entry.done && entry.success}
                class:log-error={entry.done && !entry.success}
                class:log-pending={!entry.done}
                in:fly={{ y: 4, duration: 150 }}
              >
                <span class="log-icon">
                  {#if !entry.done}
                    <span class="spinner-tiny"></span>
                  {:else if entry.success}
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="7" cy="7" r="6"/><polyline points="4.5,7 6.5,9 9.5,5"/>
                    </svg>
                  {:else}
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="7" cy="7" r="6"/><line x1="5" y1="5" x2="9" y2="9"/><line x1="9" y1="5" x2="5" y2="9"/>
                    </svg>
                  {/if}
                </span>

                <span class="log-name" title={entry.folder_name}>{truncate(entry.folder_name, 60)}</span>

                {#if entry.done && !entry.success && entry.error}
                  <span class="log-error-msg">{entry.error}</span>
                {/if}
              </div>
            {/each}
          </div>

          {#if importDone}
            <div class="import-done-actions">
              <button class="btn btn-ghost" onclick={resetWizard}>
                Back to Start
              </button>
              <button class="btn btn-secondary" onclick={() => { currentStep = 2; }}>
                ← Back to Review
              </button>
              <button class="btn btn-primary" onclick={navigateToOffers}>
                View Imported Offers
              </button>
            </div>
          {/if}
        {/if}
      </div>
    </div>
  {/if}
</div>

<!-- ═══════════════════════════════════════════════════════════════════════════
     STYLES
     ═══════════════════════════════════════════════════════════════════════════ -->
<style>
  /* ── Layout ──────────────────────────────────────────────────────────────── */
  .page {
    padding: var(--page-padding, 24px);
    height: 100%;
    background: var(--bg-base, #F5F5F7);
    color: var(--text-primary, #1D1D1F);
    display: flex;
    flex-direction: column;
    gap: 20px;
    box-sizing: border-box;
    font-family: var(--font-family, 'Inter', system-ui, sans-serif);
    overflow-y: auto;
  }

  /* ── Page header ─────────────────────────────────────────────────────────── */
  .page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 24px;
    flex-wrap: wrap;
  }

  h1 {
    font-size: 24px;
    font-weight: 700;
    letter-spacing: -0.02em;
    color: var(--onyx, #1D1D1F);
    margin: 0 0 4px 0;
  }

  .page-subtitle {
    font-size: 13px;
    color: var(--steel, #86868B);
    margin: 0;
    max-width: 560px;
  }

  /* ── Breadcrumb wizard progress ──────────────────────────────────────────── */
  .breadcrumb {
    display: flex;
    align-items: center;
    gap: 0;
    flex-shrink: 0;
  }

  .bc-step {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 0;
  }

  .bc-num {
    width: 26px;
    height: 26px;
    border-radius: 50%;
    background: var(--border, #E5E5E5);
    color: var(--steel, #86868B);
    font-size: 11px;
    font-weight: 600;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: all 0.2s;
  }

  .bc-step.bc-active .bc-num {
    background: var(--carbon, #000);
    color: #fff;
  }

  .bc-step.bc-done .bc-num {
    background: var(--onyx, #1D1D1F);
    color: #fff;
  }

  .bc-label {
    font-size: 12px;
    font-weight: 500;
    color: var(--steel, #86868B);
    white-space: nowrap;
    transition: color 0.2s;
  }

  .bc-step.bc-active .bc-label {
    color: var(--onyx, #1D1D1F);
    font-weight: 600;
  }

  .bc-step.bc-done .bc-label {
    color: var(--text-muted, #AEAEB2);
  }

  .bc-line {
    width: 32px;
    height: 1px;
    background: var(--border, #E5E5E5);
    margin: 0 8px;
    flex-shrink: 0;
    transition: background 0.2s;
  }

  .bc-line.bc-line-done {
    background: var(--onyx, #1D1D1F);
  }

  /* ── Step body ───────────────────────────────────────────────────────────── */
  .step-body {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  /* ── Cards ───────────────────────────────────────────────────────────────── */
  .card {
    background: var(--surface, #FFFFFF);
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 12px;
    overflow: hidden;
  }

  .step-card {
    flex: 1;
  }

  .card-header {
    padding: 20px 24px 0;
    border-bottom: 1px solid var(--border, #E5E5E5);
    padding-bottom: 16px;
    margin-bottom: 0;
  }

  .card-header h2 {
    font-size: 16px;
    font-weight: 600;
    color: var(--onyx, #1D1D1F);
    margin: 0 0 4px 0;
    letter-spacing: -0.01em;
  }

  .card-desc {
    font-size: 13px;
    color: var(--steel, #86868B);
    margin: 0;
    line-height: 1.5;
  }

  /* ── Path inputs (Step 1) ────────────────────────────────────────────────── */
  .paths-list {
    padding: 20px 24px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .path-row {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .path-input-wrap {
    display: flex;
    gap: 10px;
    align-items: center;
  }

  .path-input {
    flex: 1;
    font-size: 13px;
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: var(--text-primary, #1D1D1F);
  }

  .path-actions {
    display: flex;
    gap: 6px;
    align-items: center;
    flex-shrink: 0;
  }

  .validation-result {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 500;
    padding: 0 4px;
  }

  .validation-result.valid {
    color: #16a34a;
  }

  .validation-result.invalid {
    color: #dc2626;
  }

  .path-footer {
    padding: 16px 24px 20px;
    border-top: 1px solid var(--border, #E5E5E5);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .path-footer-right {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .path-summary {
    font-size: 13px;
    color: var(--steel, #86868B);
  }

  /* ── Scan progress ───────────────────────────────────────────────────────── */
  .scan-progress {
    padding: 16px 24px;
    border-top: 1px solid var(--border, #E5E5E5);
  }

  .scan-bar {
    height: 3px;
    background: var(--border, #E5E5E5);
    border-radius: 2px;
    overflow: hidden;
    margin-bottom: 8px;
  }

  .scan-bar-fill {
    height: 100%;
    background: var(--carbon, #000);
    border-radius: 2px;
    animation: scan-pulse 1.4s ease-in-out infinite;
    width: 40%;
  }

  @keyframes scan-pulse {
    0% { transform: translateX(-100%); width: 40%; }
    50% { width: 60%; }
    100% { transform: translateX(250%); width: 40%; }
  }

  .scan-status {
    font-size: 12px;
    color: var(--steel, #86868B);
    margin: 0;
  }

  /* ── Tips card ───────────────────────────────────────────────────────────── */
  .tips-card {
    padding: 16px 20px;
  }

  .tips-card h3 {
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--steel, #86868B);
    margin: 0 0 10px 0;
  }

  .tips-list {
    margin: 0;
    padding-left: 16px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .tips-list li {
    font-size: 12px;
    color: var(--text-muted, #AEAEB2);
    line-height: 1.5;
  }

  .tips-list code {
    font-family: var(--font-mono, monospace);
    font-size: 11px;
    background: var(--surface-elevated, #FAFAFA);
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 4px;
    padding: 1px 4px;
    color: var(--onyx, #1D1D1F);
  }

  /* ── Summary bar (Step 2) ────────────────────────────────────────────────── */
  .summary-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
  }

  .summary-stats {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .stat-pill {
    background: var(--surface, #fff);
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 8px;
    padding: 8px 14px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    min-width: 80px;
  }

  .stat-value {
    font-size: 20px;
    font-weight: 700;
    color: var(--onyx, #1D1D1F);
    font-variant-numeric: tabular-nums;
    line-height: 1;
  }

  .stat-label {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--steel, #86868B);
    text-align: center;
  }

  .summary-actions {
    display: flex;
    gap: 8px;
  }

  /* ── Table toolbar ───────────────────────────────────────────────────────── */
  .table-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }

  .filter-tabs {
    display: flex;
    gap: 2px;
    background: var(--surface, #fff);
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 8px;
    padding: 3px;
  }

  .filter-tab {
    background: transparent;
    border: none;
    border-radius: 6px;
    padding: 6px 14px;
    font-size: 12px;
    font-weight: 500;
    color: var(--steel, #86868B);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
    font-family: var(--font-family, 'Inter', system-ui, sans-serif);
    transition: all 0.15s;
    white-space: nowrap;
  }

  .filter-tab:hover {
    background: var(--surface-elevated, #FAFAFA);
    color: var(--onyx, #1D1D1F);
  }

  .filter-tab.active {
    background: var(--carbon, #000);
    color: #fff;
  }

  .filter-count {
    font-size: 11px;
    font-weight: 600;
    opacity: 0.7;
    font-variant-numeric: tabular-nums;
  }

  .filter-tab.active .filter-count {
    opacity: 1;
  }

  .toolbar-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .selection-count {
    font-size: 13px;
    font-weight: 600;
    color: var(--onyx, #1D1D1F);
    font-variant-numeric: tabular-nums;
  }

  /* ── Deals table (Step 2) ────────────────────────────────────────────────── */
  .table-container {
    background: var(--surface, #fff);
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 12px;
    overflow: hidden;
    overflow-x: auto;
  }

  .data-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }

  .data-table thead {
    position: sticky;
    top: 0;
    background: rgba(255, 255, 255, 0.92);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    z-index: 10;
    border-bottom: 1px solid var(--border, #E5E5E5);
  }

  .data-table th {
    padding: 10px 12px;
    text-align: left;
    font-size: 11px;
    font-weight: 600;
    color: var(--steel, #86868B);
    text-transform: uppercase;
    letter-spacing: 0.06em;
    white-space: nowrap;
  }

  .data-table td {
    padding: 10px 12px;
    border-bottom: 1px solid var(--border, #E5E5E5);
    color: var(--text-primary, #1D1D1F);
    vertical-align: middle;
  }

  .data-table tbody tr {
    transition: background 0.12s;
  }

  .data-table tbody tr:last-child td {
    border-bottom: none;
  }

  .data-table tbody tr:hover {
    background: var(--surface-elevated, #FAFAFA);
  }

  .data-table tbody tr.row-selected {
    background: rgba(0, 0, 0, 0.025);
  }

  .data-table tbody tr.row-unmatched td:first-child {
    border-left: 2px solid #f59e0b;
  }

  /* Column sizing */
  .col-check { width: 36px; }
  .col-folder { min-width: 220px; max-width: 320px; }
  .col-instrument { width: 110px; }
  .col-year { width: 70px; }
  .col-files { width: 160px; }
  .col-match { width: 160px; }
  .col-score { width: 72px; }
  .col-customer { width: 220px; }
  .col-cs { width: 60px; text-align: center; }

  .align-center { text-align: center; }

  .folder-name {
    font-size: 12.5px;
    font-family: var(--font-mono, monospace);
    color: var(--onyx, #1D1D1F);
    cursor: default;
  }

  .year-chip {
    font-size: 12px;
    font-weight: 600;
    color: var(--steel, #86868B);
    font-variant-numeric: tabular-nums;
  }

  .file-count {
    font-size: 12px;
    color: var(--steel, #86868B);
  }

  .match-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--onyx, #1D1D1F);
  }

  .text-muted {
    color: var(--text-muted, #AEAEB2);
  }

  .no-match {
    font-style: italic;
    font-size: 12px;
  }

  /* Score badges */
  .score-badge {
    display: inline-flex;
    align-items: center;
    padding: 2px 8px;
    border-radius: 100px;
    font-size: 11px;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.01em;
  }

  .badge-score-high {
    background: #dcfce7;
    color: #15803d;
  }

  .badge-score-mid {
    background: #fef3c7;
    color: #b45309;
  }

  .badge-score-low {
    background: #fee2e2;
    color: #b91c1c;
  }

  /* Neutral badge (instrument type) */
  .badge-neutral {
    display: inline-flex;
    align-items: center;
    padding: 2px 8px;
    border-radius: 100px;
    font-size: 11px;
    font-weight: 500;
    background: var(--surface-elevated, #FAFAFA);
    color: var(--steel, #86868B);
    border: 1px solid var(--border, #E5E5E5);
    white-space: nowrap;
  }

  /* Customer dropdown */
  .customer-select {
    width: 100%;
    padding: 5px 8px;
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 6px;
    font-size: 12px;
    font-family: var(--font-family, 'Inter', system-ui, sans-serif);
    color: var(--text-primary, #1D1D1F);
    background: var(--surface, #fff);
    cursor: pointer;
    transition: border-color 0.15s;
  }

  .customer-select:focus {
    outline: none;
    border-color: var(--onyx, #1D1D1F);
    box-shadow: 0 0 0 3px rgba(29, 29, 31, 0.06);
  }

  /* Costing sheet dot */
  .cs-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--carbon, #000);
  }

  .deal-checkbox {
    width: 16px;
    height: 16px;
    cursor: pointer;
    accent-color: var(--carbon, #000);
  }

  .empty-row {
    text-align: center;
    padding: 32px 16px;
    color: var(--steel, #86868B);
    font-size: 13px;
  }

  /* ── Proceed bar ─────────────────────────────────────────────────────────── */
  .proceed-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
  }

  .proceed-hint {
    font-size: 13px;
    color: var(--steel, #86868B);
    margin: 0;
    flex: 1;
    line-height: 1.4;
  }

  /* ── Step 3: Import ──────────────────────────────────────────────────────── */
  .import-confirm {
    padding: 20px 24px;
  }

  .import-confirm-stats {
    display: flex;
    flex-direction: column;
    gap: 0;
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 10px;
    overflow: hidden;
    margin-bottom: 20px;
    max-width: 480px;
  }

  .ic-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    border-bottom: 1px solid var(--border, #E5E5E5);
  }

  .ic-row:last-child {
    border-bottom: none;
  }

  .ic-label {
    font-size: 13px;
    color: var(--steel, #86868B);
  }

  .ic-value {
    font-size: 14px;
    font-weight: 700;
    color: var(--onyx, #1D1D1F);
    font-variant-numeric: tabular-nums;
  }

  .ic-warn {
    color: #b45309;
  }

  .import-actions-row {
    display: flex;
    gap: 10px;
    align-items: center;
  }

  /* Import log */
  .import-progress-header {
    padding: 14px 24px;
    border-top: 1px solid var(--border, #E5E5E5);
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 13px;
    color: var(--steel, #86868B);
    flex-wrap: wrap;
  }

  .err-text {
    color: #dc2626;
    font-weight: 600;
  }

  .progress-bar-wrap {
    display: flex;
    align-items: center;
    gap: 10px;
    flex: 1;
    min-width: 160px;
  }

  .progress-bar {
    flex: 1;
    height: 4px;
    background: var(--border, #E5E5E5);
    border-radius: 2px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: var(--carbon, #000);
    border-radius: 2px;
    transition: width 0.3s ease;
  }

  .progress-fill.progress-fill-error {
    background: linear-gradient(90deg, var(--carbon, #000) 0%, #dc2626 100%);
  }

  .progress-pct {
    font-size: 12px;
    font-weight: 600;
    color: var(--onyx, #1D1D1F);
    font-variant-numeric: tabular-nums;
    width: 36px;
    text-align: right;
  }

  .import-log {
    max-height: 420px;
    overflow-y: auto;
    border-top: 1px solid var(--border, #E5E5E5);
  }

  .log-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 9px 24px;
    border-bottom: 1px solid var(--border, #E5E5E5);
    font-size: 12.5px;
    transition: background 0.1s;
  }

  .log-row:last-child {
    border-bottom: none;
  }

  .log-row.log-success {
    background: #f0fdf4;
  }

  .log-row.log-error {
    background: #fef2f2;
  }

  .log-row.log-pending {
    background: transparent;
    opacity: 0.6;
  }

  .log-icon {
    flex-shrink: 0;
    width: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .log-row.log-success .log-icon {
    color: #16a34a;
  }

  .log-row.log-error .log-icon {
    color: #dc2626;
  }

  .log-name {
    flex: 1;
    font-family: var(--font-mono, monospace);
    font-size: 12px;
    color: var(--onyx, #1D1D1F);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .log-error-msg {
    font-size: 11px;
    color: #dc2626;
    flex-shrink: 0;
    max-width: 200px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .import-done-actions {
    padding: 16px 24px;
    border-top: 1px solid var(--border, #E5E5E5);
    display: flex;
    gap: 10px;
    align-items: center;
    justify-content: flex-end;
  }

  /* ── Shared button styles ─────────────────────────────────────────────────── */
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    padding: 9px 18px;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 500;
    font-family: var(--font-family, 'Inter', system-ui, sans-serif);
    cursor: pointer;
    border: none;
    transition: all 0.15s;
    white-space: nowrap;
    text-decoration: none;
  }

  .btn:disabled {
    opacity: 0.38;
    cursor: not-allowed;
  }

  .btn-primary {
    background: var(--carbon, #000);
    color: #fff;
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--onyx, #1D1D1F);
  }

  .btn-secondary {
    background: transparent;
    border: 1px solid var(--border, #E5E5E5);
    color: var(--text-primary, #1D1D1F);
  }

  .btn-secondary:hover:not(:disabled) {
    border-color: var(--onyx, #1D1D1F);
    background: var(--surface-elevated, #FAFAFA);
  }

  .btn-ghost {
    background: transparent;
    color: var(--steel, #86868B);
  }

  .btn-ghost:hover:not(:disabled) {
    background: var(--onyx-tint, rgba(29,29,31,0.04));
    color: var(--onyx, #1D1D1F);
  }

  .btn-icon-only {
    padding: 8px;
    width: 34px;
    height: 34px;
  }

  /* ── Input ───────────────────────────────────────────────────────────────── */
  .input {
    width: 100%;
    padding: 9px 12px;
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 8px;
    font-size: 13px;
    font-family: var(--font-family, 'Inter', system-ui, sans-serif);
    color: var(--text-primary, #1D1D1F);
    background: var(--surface, #fff);
    box-sizing: border-box;
    transition: border-color 0.15s, box-shadow 0.15s;
  }

  .input:focus {
    outline: none;
    border-color: var(--onyx, #1D1D1F);
    box-shadow: 0 0 0 3px rgba(29, 29, 31, 0.06);
  }

  .input::placeholder {
    color: var(--text-muted, #AEAEB2);
  }

  .input-error {
    border-color: #dc2626;
  }

  .input-error:focus {
    border-color: #dc2626;
    box-shadow: 0 0 0 3px rgba(220, 38, 38, 0.1);
  }

  .input-success {
    border-color: #16a34a;
  }

  /* ── Spinner ─────────────────────────────────────────────────────────────── */
  .spinner-tiny {
    display: inline-block;
    width: 12px;
    height: 12px;
    border: 1.5px solid var(--border, #E5E5E5);
    border-top-color: var(--carbon, #000);
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
    flex-shrink: 0;
  }

  .spinner-white {
    border-color: rgba(255, 255, 255, 0.3);
    border-top-color: #fff;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
