<script lang="ts">
  import { fade, slide } from 'svelte/transition';
  import { motionMs } from '../motion';
  import WabiCard from './ui/WabiCard.svelte';
  import type { ArchaeologyReportData } from '../types/archaeology';

  
  interface Props {
    // Props
    report?: ArchaeologyReportData | null;
  }

  let { report = null }: Props = $props();

  // Helpers
  function formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1073741824) return `${(bytes / 1048576).toFixed(1)} MB`;
    return `${(bytes / 1073741824).toFixed(2)} GB`;
  }

  function formatDate(dateStr: string): string {
    if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return 'N/A';
    return new Date(dateStr).toLocaleDateString();
  }

  // Quality level color
  function qualityColor(level: string): string {
    switch(level) {
      case 'high': return 'rgba(21, 128, 61, 0.15)'; // green
      case 'medium': return 'rgba(197, 160, 89, 0.15)'; // gold
      case 'low': return 'rgba(217, 119, 6, 0.15)'; // amber
      case 'unreadable': return 'rgba(220, 38, 38, 0.15)'; // red
      default: return 'rgba(87, 83, 78, 0.1)';
    }
  }

  function qualityTextColor(level: string): string {
    switch(level) {
      case 'high': return '#15803d';
      case 'medium': return '#c5a059';
      case 'low': return '#d97706';
      case 'unreadable': return '#dc2626';
      default: return '#57534e';
    }
  }

  // Uncertainty severity icon
  function severityIcon(type: string): string {
    switch(type) {
      case 'duplicates': return 'Dup';
      case 'clarification_needed': return '!';
      case 'conflicting_totals': return 'Conflict';
      default: return '*';
    }
  }

  // Section collapse state
  let sectionsOpen = $state({
    overview: true,
    clusters: true,
    quality: true,
    languages: true,
    uncertainties: true
  });

  function toggleSection(section: string) {
    sectionsOpen = {
      ...sectionsOpen,
      [section]: !sectionsOpen[section]
    };
  }
</script>

<!-- Report Layout -->
{#if report}
<div class="archaeology-report" in:fade={{ duration: motionMs(500) }}>
  <!-- Header -->
  <header class="report-header">
    <h1>Archaeology Report</h1>
    <p class="generated-at">Generated {formatDate(report.generated_at)}</p>
  </header>

  <!-- Section 1: Workspace Overview -->
  <WabiCard variant="elevated" padding="none">
    <button class="section-header" onclick={() => toggleSection('overview')}>
      <span class="section-icon">Stats</span>
      <h2>Workspace Overview</h2>
      <span class="toggle">{sectionsOpen.overview ? '−' : '+'}</span>
    </button>
    {#if sectionsOpen.overview}
      <div class="section-content" transition:slide={{ duration: motionMs(200) }}>
        <div class="stats-grid">
          <div class="stat">
            <span class="stat-value">{report.workspace_overview.total_files}</span>
            <span class="stat-label">Files</span>
          </div>
          <div class="stat">
            <span class="stat-value">{report.workspace_overview.total_folders}</span>
            <span class="stat-label">Folders</span>
          </div>
          <div class="stat">
            <span class="stat-value">{formatBytes(report.workspace_overview.total_size_bytes)}</span>
            <span class="stat-label">Total Size</span>
          </div>
        </div>

        <!-- Date Range -->
        <div class="info-row">
          <span class="info-label">Date Range:</span>
          <span class="info-value">
            {formatDate(report.workspace_overview.oldest_file)} → {formatDate(report.workspace_overview.newest_file)}
          </span>
        </div>

        <!-- Formats detected -->
        <div class="formats-section">
          <h4>Formats Detected</h4>
          <div class="format-chips">
            {#each report.workspace_overview.formats_detected as ext}
              <span class="format-chip">{ext}</span>
            {/each}
          </div>
        </div>
      </div>
    {/if}
  </WabiCard>

  <!-- Section 2: Detected Clusters -->
  <WabiCard variant="elevated" padding="none">
    <button class="section-header" onclick={() => toggleSection('clusters')}>
      <span class="section-icon">Clusters</span>
      <h2>Detected Clusters</h2>
      <span class="toggle">{sectionsOpen.clusters ? '−' : '+'}</span>
    </button>
    {#if sectionsOpen.clusters}
      <div class="section-content" transition:slide={{ duration: motionMs(200) }}>
        {#if report.detected_clusters.length === 0}
          <p class="empty-state">No clusters detected. Files appear independent.</p>
        {:else}
          {#each report.detected_clusters as cluster, i}
            <div class="cluster-card">
              <div class="cluster-header">
                <span class="cluster-number">Cluster {i + 1}</span>
                <span class="cluster-confidence">
                  {(cluster.confidence * 100).toFixed(0)}% confidence
                </span>
              </div>
              <p class="cluster-description">{cluster.description}</p>
              <details class="cluster-files">
                <summary>{cluster.file_paths.length} files</summary>
                <ul>
                  {#each cluster.file_paths.slice(0, 10) as path}
                    <li>{path}</li>
                  {/each}
                  {#if cluster.file_paths.length > 10}
                    <li class="more-files">...and {cluster.file_paths.length - 10} more</li>
                  {/if}
                </ul>
              </details>
            </div>
          {/each}
        {/if}
      </div>
    {/if}
  </WabiCard>

  <!-- Section 3: Document Quality Summary -->
  <WabiCard variant="elevated" padding="none">
    <button class="section-header" onclick={() => toggleSection('quality')}>
      <span class="section-icon">Quality</span>
      <h2>Document Quality Summary</h2>
      <span class="toggle">{sectionsOpen.quality ? '−' : '+'}</span>
    </button>
    {#if sectionsOpen.quality}
      <div class="section-content" transition:slide={{ duration: motionMs(200) }}>
        <div class="quality-grid">
          <div class="quality-card" style="background: {qualityColor('high')}">
            <div class="quality-value" style="color: {qualityTextColor('high')}">
              {report.quality_summary.high_confidence}
            </div>
            <div class="quality-label">High Confidence</div>
            <div class="quality-desc">Clarity ≥ 80%</div>
          </div>

          <div class="quality-card" style="background: {qualityColor('medium')}">
            <div class="quality-value" style="color: {qualityTextColor('medium')}">
              {report.quality_summary.medium_confidence}
            </div>
            <div class="quality-label">Medium Confidence</div>
            <div class="quality-desc">Clarity 50–80%</div>
          </div>

          <div class="quality-card" style="background: {qualityColor('low')}">
            <div class="quality-value" style="color: {qualityTextColor('low')}">
              {report.quality_summary.low_confidence}
            </div>
            <div class="quality-label">Low Confidence</div>
            <div class="quality-desc">Clarity 20–50%</div>
          </div>

          <div class="quality-card" style="background: {qualityColor('unreadable')}">
            <div class="quality-value" style="color: {qualityTextColor('unreadable')}">
              {report.quality_summary.unreadable}
            </div>
            <div class="quality-label">Unreadable</div>
            <div class="quality-desc">Clarity &lt; 20%</div>
          </div>
        </div>
      </div>
    {/if}
  </WabiCard>

  <!-- Section 4: Languages & Formats -->
  <WabiCard variant="elevated" padding="none">
    <button class="section-header" onclick={() => toggleSection('languages')}>
      <span class="section-icon">Languages</span>
      <h2>Languages & Formats</h2>
      <span class="toggle">{sectionsOpen.languages ? '−' : '+'}</span>
    </button>
    {#if sectionsOpen.languages}
      <div class="section-content" transition:slide={{ duration: motionMs(200) }}>
        <!-- Languages -->
        <div class="subsection">
          <h4>Languages</h4>
          <div class="lang-stats">
            <div class="lang-row">
              <span class="lang-label">English</span>
              <span class="lang-value">{report.languages_formats.english}</span>
            </div>
            <div class="lang-row">
              <span class="lang-label">Arabic</span>
              <span class="lang-value">{report.languages_formats.arabic}</span>
            </div>
            <div class="lang-row">
              <span class="lang-label">Mixed</span>
              <span class="lang-value">{report.languages_formats.mixed}</span>
            </div>
          </div>
        </div>

        <!-- Formats -->
        <div class="subsection">
          <h4>Formats</h4>
          <div class="format-stats">
            <div class="format-row">
              <span class="format-label">Scanned PDFs</span>
              <span class="format-value">{report.languages_formats.scanned_pdfs}</span>
            </div>
            <div class="format-row">
              <span class="format-label">Native PDFs</span>
              <span class="format-value">{report.languages_formats.native_pdfs}</span>
            </div>
            <div class="format-row">
              <span class="format-label">Excel Files</span>
              <span class="format-value">{report.languages_formats.excel_files}</span>
            </div>
            <div class="format-row">
              <span class="format-label">Word Documents</span>
              <span class="format-value">{report.languages_formats.word_files}</span>
            </div>
            <div class="format-row">
              <span class="format-label">Other Formats</span>
              <span class="format-value">{report.languages_formats.other_formats}</span>
            </div>
          </div>
        </div>
      </div>
    {/if}
  </WabiCard>

  <!-- Section 5: Explicit Uncertainty -->
  <WabiCard variant="elevated" padding="none">
    <button class="section-header" onclick={() => toggleSection('uncertainties')}>
      <span class="section-icon">!</span>
      <h2>Explicit Uncertainties</h2>
      <span class="toggle">{sectionsOpen.uncertainties ? '−' : '+'}</span>
    </button>
    {#if sectionsOpen.uncertainties}
      <div class="section-content" transition:slide={{ duration: motionMs(200) }}>
        {#if report.uncertainties.length === 0}
          <p class="empty-state">No significant uncertainties detected. All documents appear consistent.</p>
        {:else}
          {#each report.uncertainties as uncertainty, i}
            <div class="uncertainty-card">
              <div class="uncertainty-header">
                <span class="uncertainty-icon">{severityIcon(uncertainty.type)}</span>
                <span class="uncertainty-type">{uncertainty.type.replace(/_/g, ' ')}</span>
              </div>
              <p class="uncertainty-description">{uncertainty.description}</p>
              {#if uncertainty.affected_docs.length > 0}
                <details class="uncertainty-docs">
                  <summary>{uncertainty.affected_docs.length} affected documents</summary>
                  <ul>
                    {#each uncertainty.affected_docs.slice(0, 5) as doc}
                      <li>{doc}</li>
                    {/each}
                    {#if uncertainty.affected_docs.length > 5}
                      <li class="more-files">...and {uncertainty.affected_docs.length - 5} more</li>
                    {/if}
                  </ul>
                </details>
              {/if}
            </div>
          {/each}
        {/if}
      </div>
    {/if}
  </WabiCard>

  <!-- Footer Note -->
  <div class="report-footer">
    <p>This report describes the workspace as observed. No judgment. No fixes applied.</p>
  </div>
</div>
{:else}
<div class="no-report">
  <p>No report available yet. Start a scan to generate insights.</p>
</div>
{/if}

<style>
  .archaeology-report {
    font-family: 'Georgia', serif;
    color: var(--color-ink, #1c1c1c);
    max-width: 900px;
    margin: 0 auto;
    padding: 2rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .report-header {
    text-align: center;
    margin-bottom: 1rem;
  }

  .report-header h1 {
    font-size: 2rem;
    font-weight: normal;
    margin: 0;
    letter-spacing: -0.5px;
    color: var(--color-ink, #1c1c1c);
  }

  .generated-at {
    font-size: 0.85rem;
    color: rgba(28,28,28,0.5);
    margin-top: 0.5rem;
    font-family: 'Courier New', monospace;
  }

  /* Section Header */
  .section-header {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1.25rem 1.5rem;
    background: none;
    border: none;
    cursor: pointer;
    text-align: left;
    border-bottom: 1px solid rgba(0,0,0,0.08);
    transition: background 0.2s ease;
  }

  .section-header:hover {
    background: rgba(0,0,0,0.02);
  }

  .section-header h2 {
    flex: 1;
    font-size: 1.2rem;
    font-weight: 500;
    margin: 0;
    color: var(--color-ink, #1c1c1c);
  }

  .section-icon {
    font-size: 1.4rem;
  }

  .toggle {
    font-size: 1.5rem;
    color: rgba(28,28,28,0.4);
    font-weight: 300;
  }

  .section-content {
    padding: 1.5rem;
  }

  /* Stats Grid (Overview) */
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 1rem;
    margin-bottom: 1.5rem;
  }

  .stat {
    text-align: center;
    padding: 1.5rem 1rem;
    background: rgba(0,0,0,0.03);
    border-radius: 8px;
    border: 1px solid rgba(0,0,0,0.05);
  }

  .stat-value {
    display: block;
    font-size: 2rem;
    font-weight: 600;
    color: var(--color-ink, #1c1c1c);
    margin-bottom: 0.25rem;
  }

  .stat-label {
    font-size: 0.75rem;
    color: rgba(28,28,28,0.6);
    text-transform: uppercase;
    letter-spacing: 0.8px;
    font-family: 'Courier New', monospace;
  }

  .info-row {
    display: flex;
    justify-content: space-between;
    padding: 0.75rem 0;
    border-bottom: 1px solid rgba(0,0,0,0.05);
    font-size: 0.95rem;
  }

  .info-label {
    font-weight: 500;
    color: rgba(28,28,28,0.7);
  }

  .info-value {
    font-family: 'Courier New', monospace;
    color: var(--color-ink, #1c1c1c);
  }

  /* Formats */
  .formats-section {
    margin-top: 1.5rem;
  }

  .formats-section h4 {
    font-size: 0.9rem;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: rgba(28,28,28,0.6);
    margin-bottom: 0.75rem;
    font-weight: 500;
  }

  .format-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .format-chip {
    background: rgba(0,0,0,0.06);
    color: var(--color-ink, #1c1c1c);
    padding: 0.4rem 0.75rem;
    border-radius: 6px;
    font-size: 0.85rem;
    font-family: 'Courier New', monospace;
    border: 1px solid rgba(0,0,0,0.08);
  }

  /* Clusters */
  .cluster-card {
    background: rgba(0,0,0,0.02);
    border: 1px solid rgba(0,0,0,0.06);
    border-radius: 8px;
    padding: 1rem;
    margin-bottom: 1rem;
  }

  .cluster-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .cluster-number {
    font-weight: 600;
    font-size: 0.9rem;
    color: var(--color-ink, #1c1c1c);
  }

  .cluster-confidence {
    font-size: 0.8rem;
    color: rgba(28,28,28,0.5);
    font-family: 'Courier New', monospace;
  }

  .cluster-description {
    font-size: 0.95rem;
    color: rgba(28,28,28,0.8);
    margin-bottom: 0.75rem;
  }

  .cluster-files summary {
    cursor: pointer;
    font-size: 0.85rem;
    color: rgba(28,28,28,0.6);
    text-decoration: underline;
    list-style: none;
  }

  .cluster-files ul {
    margin-top: 0.5rem;
    padding-left: 1.5rem;
    font-size: 0.8rem;
    font-family: 'Courier New', monospace;
    color: rgba(28,28,28,0.7);
  }

  .cluster-files li {
    margin-bottom: 0.25rem;
  }

  /* Quality Grid */
  .quality-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
    gap: 1rem;
  }

  .quality-card {
    padding: 1.5rem 1rem;
    border-radius: 8px;
    text-align: center;
    border: 1px solid rgba(0,0,0,0.08);
  }

  .quality-value {
    font-size: 2.5rem;
    font-weight: 700;
    margin-bottom: 0.5rem;
  }

  .quality-label {
    font-size: 0.85rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: rgba(28,28,28,0.8);
    margin-bottom: 0.25rem;
  }

  .quality-desc {
    font-size: 0.75rem;
    font-family: 'Courier New', monospace;
    color: rgba(28,28,28,0.5);
  }

  /* Subsections */
  .subsection {
    margin-bottom: 1.5rem;
  }

  .subsection h4 {
    font-size: 0.9rem;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: rgba(28,28,28,0.6);
    margin-bottom: 0.75rem;
    font-weight: 500;
  }

  .lang-stats, .format-stats {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .lang-row, .format-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 1rem;
    background: rgba(0,0,0,0.02);
    border-radius: 6px;
    border: 1px solid rgba(0,0,0,0.05);
  }

  .lang-label, .format-label {
    font-size: 0.9rem;
    color: rgba(28,28,28,0.7);
  }

  .lang-value, .format-value {
    font-weight: 600;
    font-family: 'Courier New', monospace;
    font-size: 1rem;
    color: var(--color-ink, #1c1c1c);
  }

  /* Uncertainties */
  .uncertainty-card {
    background: rgba(255, 193, 7, 0.08);
    border: 1px solid rgba(255, 193, 7, 0.3);
    border-radius: 8px;
    padding: 1rem;
    margin-bottom: 1rem;
  }

  .uncertainty-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }

  .uncertainty-icon {
    font-size: 1.2rem;
  }

  .uncertainty-type {
    font-weight: 600;
    text-transform: capitalize;
    font-size: 0.9rem;
    color: var(--color-ink, #1c1c1c);
  }

  .uncertainty-description {
    font-size: 0.95rem;
    color: rgba(28,28,28,0.8);
    margin-bottom: 0.75rem;
  }

  .uncertainty-docs summary {
    cursor: pointer;
    font-size: 0.85rem;
    color: rgba(28,28,28,0.6);
    text-decoration: underline;
    list-style: none;
  }

  .uncertainty-docs ul {
    margin-top: 0.5rem;
    padding-left: 1.5rem;
    font-size: 0.8rem;
    font-family: 'Courier New', monospace;
    color: rgba(28,28,28,0.7);
  }

  .uncertainty-docs li {
    margin-bottom: 0.25rem;
  }

  /* Empty State */
  .empty-state {
    text-align: center;
    padding: 2rem 1rem;
    color: rgba(28,28,28,0.4);
    font-style: italic;
  }

  .more-files {
    color: rgba(28,28,28,0.5);
    font-style: italic;
  }

  /* Footer */
  .report-footer {
    margin-top: 2rem;
    padding-top: 1rem;
    border-top: 1px solid rgba(0,0,0,0.1);
    text-align: center;
  }

  .report-footer p {
    font-size: 0.85rem;
    color: rgba(28,28,28,0.5);
    font-style: italic;
  }

  /* No Report State */
  .no-report {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 300px;
    padding: 2rem;
  }

  .no-report p {
    font-size: 1.1rem;
    color: rgba(28,28,28,0.4);
    font-style: italic;
    text-align: center;
  }
</style>
