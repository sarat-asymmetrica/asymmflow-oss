<script>
  import { onMount } from "svelte";
  import { GetOCRStats } from "../../../wailsjs/go/main/App";

  // State management
  let loading = $state(true);
  let error = $state(null);
  let stats = null;

  // Formatted values
  let totalDocuments = $state(0);
  let avgConfidence = $state(0);
  let cacheHitRate = $state(0);
  let gpuUtilization = $state(0);
  let costSavings = $state(0);
  let engineDistribution = $state([]);

  // Fetch stats from backend
  async function fetchStats() {
    loading = true;
    error = null;

    try {
      const data = await GetOCRStats();
      stats = data;

      // Calculate metrics
      totalDocuments = stats.total_documents || 0;
      avgConfidence = (stats.avg_confidence * 100) || 0;

      // Cache hit rate
      const cacheHits = stats.dna_cache_hits || 0;
      cacheHitRate = totalDocuments > 0 ? (cacheHits / totalDocuments) * 100 : 0;

      // GPU utilization
      const gpuProcessed = stats.gpu_processed || 0;
      gpuUtilization = totalDocuments > 0 ? (gpuProcessed / totalDocuments) * 100 : 0;

      // Cost savings estimate (DNA cache = 10× speed, GPU = 5× speed)
      const cacheSpeedup = cacheHits * 10;
      const gpuSpeedup = gpuProcessed * 5;
      costSavings = ((cacheSpeedup + gpuSpeedup) / Math.max(totalDocuments, 1)) * 100;

      // Engine distribution
      if (stats.engine_distribution && Array.isArray(stats.engine_distribution)) {
        engineDistribution = stats.engine_distribution
          .map(e => ({
            name: e.Engine || e.engine || "Unknown",
            count: e.Count || e.count || 0
          }))
          .sort((a, b) => b.count - a.count);
      }

      loading = false;
    } catch (err) {
      error = err.message || "Failed to fetch OCR stats";
      loading = false;
      console.error("OCR Stats Error:", err);
    }
  }

  // Calculate bar width percentage for engine distribution
  function getBarWidth(count) {
    if (!engineDistribution.length) return 0;
    const maxCount = Math.max(...engineDistribution.map(e => e.count));
    return maxCount > 0 ? (count / maxCount) * 100 : 0;
  }

  // Format number with commas
  function formatNumber(num) {
    return Math.round(num).toLocaleString();
  }

  // Format percentage
  function formatPercent(num) {
    return num.toFixed(1);
  }

  // Auto-refresh every 30 seconds
  onMount(() => {
    fetchStats();
    const interval = setInterval(fetchStats, 30000);
    return () => clearInterval(interval);
  });
</script>

<div class="ocr-stats-widget" role="region" aria-label="OCR Statistics Dashboard">
  {#if loading}
    <div class="loading-state" role="status" aria-live="polite">
      <div class="loading-spinner"></div>
      <p class="loading-text">Loading OCR statistics...</p>
    </div>
  {:else if error}
    <div class="error-state" role="alert">
      <div class="error-icon">!</div>
      <p class="error-text">{error}</p>
      <button class="retry-btn" onclick={fetchStats}>Retry</button>
    </div>
  {:else}
    <!-- Header -->
    <div class="widget-header">
      <h3 class="widget-title">OCR Intelligence</h3>
      <button
        class="refresh-btn"
        onclick={fetchStats}
        aria-label="Refresh OCR statistics"
      >
        Refresh
      </button>
    </div>

    <!-- Main Stats Grid -->
    <div class="stats-grid">
      <!-- Total Documents -->
      <div class="stat-card">
        <div class="stat-label">Documents</div>
        <div class="stat-value">{formatNumber(totalDocuments)}</div>
      </div>

      <!-- Average Confidence -->
      <div class="stat-card">
        <div class="stat-label">Confidence</div>
        <div class="stat-value stat-green">{formatPercent(avgConfidence)}%</div>
      </div>

      <!-- Cache Hit Rate -->
      <div class="stat-card">
        <div class="stat-label">DNA Cache</div>
        <div class="stat-value stat-green">{formatPercent(cacheHitRate)}%</div>
        <div class="stat-subtitle">10× speedup</div>
      </div>

      <!-- GPU Utilization -->
      <div class="stat-card">
        <div class="stat-label">GPU Usage</div>
        <div class="stat-value stat-green">{formatPercent(gpuUtilization)}%</div>
        <div class="stat-subtitle">5× speedup</div>
      </div>

      <!-- Cost Savings -->
      <div class="stat-card full-width">
        <div class="stat-label">Efficiency Gain</div>
        <div class="stat-value stat-green">{formatPercent(costSavings)}%</div>
        <div class="stat-subtitle">Combined optimization impact</div>
      </div>
    </div>

    <!-- Engine Distribution -->
    {#if engineDistribution.length > 0}
      <div class="engine-section">
        <h4 class="section-title">Engine Distribution</h4>
        <div class="engine-list">
          {#each engineDistribution as engine}
            <div class="engine-item">
              <div class="engine-header">
                <span class="engine-name">{engine.name}</span>
                <span class="engine-count">{formatNumber(engine.count)}</span>
              </div>
              <div class="engine-bar-container">
                <div
                  class="engine-bar"
                  style="width: {getBarWidth(engine.count)}%"
                  role="progressbar"
                  aria-valuenow={engine.count}
                  aria-valuemin="0"
                  aria-valuemax={Math.max(...engineDistribution.map(e => e.count))}
                  aria-label="{engine.name}: {engine.count} documents"
                ></div>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  /* ================================================================
     OCR STATS WIDGET - WABI-SABI DESIGN
     Rice paper aesthetic with φ-based spacing
     ================================================================ */

  .ocr-stats-widget {
    background: rgba(255, 255, 255, 0.3);
    backdrop-filter: blur(8px);
    border-radius: var(--radius-xl, 32px);
    padding: var(--space-6, 20px);
    font-family: var(--font-sans, 'DM Sans', sans-serif);
    min-height: 320px;
    display: flex;
    flex-direction: column;
    gap: var(--space-5, 16px);
  }

  /* Loading State */
  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4, 12px);
    min-height: 280px;
  }

  .loading-spinner {
    width: 34px;
    height: 34px;
    border: 3px solid rgba(0, 0, 0, 0.1);
    border-top-color: var(--color-safe, #15803d);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .loading-text {
    font-size: var(--text-sm, 0.75rem);
    color: var(--ink-light, #666);
    margin: 0;
  }

  /* Error State */
  .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-3, 8px);
    min-height: 280px;
    padding: var(--space-5, 16px);
  }

  .error-icon {
    font-size: 34px;
  }

  .error-text {
    font-size: var(--text-sm, 0.75rem);
    color: var(--color-danger, #dc2626);
    text-align: center;
    margin: 0;
  }

  .retry-btn {
    margin-top: var(--space-3, 8px);
    padding: var(--space-3, 8px) var(--space-5, 16px);
    background: var(--ink, #1c1c1c);
    color: var(--paper, white);
    border: none;
    border-radius: var(--radius-pill, 100px);
    font-size: var(--text-sm, 0.75rem);
    font-family: var(--font-sans, 'DM Sans', sans-serif);
    cursor: pointer;
    transition: transform 0.2s ease, box-shadow 0.2s ease;
  }

  .retry-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  }

  /* Header */
  .widget-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-bottom: var(--space-3, 8px);
    border-bottom: 1px solid rgba(0, 0, 0, 0.08);
  }

  .widget-title {
    font-family: var(--font-heading, Georgia, serif);
    font-size: var(--text-xl, 1.25rem);
    font-weight: var(--font-weight-regular, 400);
    color: var(--ink, #1c1c1c);
    margin: 0;
  }

  .refresh-btn {
    width: 34px;
    height: 34px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.04);
    border: none;
    border-radius: 50%;
    font-size: var(--text-xl, 1.25rem);
    cursor: pointer;
    transition: all 0.2s ease;
    color: var(--ink, #1c1c1c);
  }

  .refresh-btn:hover {
    background: rgba(0, 0, 0, 0.08);
    transform: rotate(90deg);
  }

  /* Stats Grid */
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--space-4, 12px);
  }

  .stat-card {
    background: rgba(255, 255, 255, 0.5);
    border-radius: var(--radius-lg, 20px);
    padding: var(--space-4, 12px);
    display: flex;
    flex-direction: column;
    gap: var(--space-2, 4px);
  }

  .stat-card.full-width {
    grid-column: 1 / -1;
  }

  .stat-label {
    font-family: 'Courier Prime', monospace;
    font-size: var(--text-xs, 0.6875rem);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--ink-light, #666);
  }

  .stat-value {
    font-family: var(--font-heading, Georgia, serif);
    font-size: var(--text-2xl, 1.75rem);
    font-weight: var(--font-weight-bold, 700);
    color: var(--ink, #1c1c1c);
    line-height: 1;
  }

  .stat-value.stat-green {
    color: var(--color-safe, #15803d);
  }

  .stat-subtitle {
    font-size: var(--text-2xs, 0.625rem);
    color: var(--ink-faint, #999);
    font-family: 'Courier Prime', monospace;
  }

  /* Engine Distribution */
  .engine-section {
    margin-top: var(--space-4, 12px);
  }

  .section-title {
    font-family: 'Courier Prime', monospace;
    font-size: var(--text-xs, 0.6875rem);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--ink-light, #666);
    margin: 0 0 var(--space-3, 8px) 0;
  }

  .engine-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3, 8px);
  }

  .engine-item {
    display: flex;
    flex-direction: column;
    gap: var(--space-2, 4px);
  }

  .engine-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .engine-name {
    font-family: 'Courier Prime', monospace;
    font-size: var(--text-xs, 0.6875rem);
    color: var(--ink, #1c1c1c);
    font-weight: var(--font-weight-medium, 500);
  }

  .engine-count {
    font-family: 'Courier Prime', monospace;
    font-size: var(--text-xs, 0.6875rem);
    color: var(--ink-light, #666);
  }

  .engine-bar-container {
    width: 100%;
    height: 8px;
    background: rgba(0, 0, 0, 0.06);
    border-radius: var(--radius-sm, 6px);
    overflow: hidden;
  }

  .engine-bar {
    height: 100%;
    background: linear-gradient(90deg, var(--color-safe, #15803d), var(--color-info, #2563eb));
    border-radius: var(--radius-sm, 6px);
    transition: width 0.3s cubic-bezier(0.16, 1, 0.3, 1);
  }

  /* Responsive Design */
  @media (max-width: 640px) {
    .stats-grid {
      grid-template-columns: 1fr;
    }

    .stat-card.full-width {
      grid-column: 1;
    }
  }
</style>
